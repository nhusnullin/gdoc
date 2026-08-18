from unittest.mock import MagicMock, patch

import pytest

from gdoc.auth import (
    SCOPES,
    drive_service,
    load_credentials,
    load_service_account_credentials,
    resolve_auth_mode,
)
from gdoc.config import Config
from gdoc.guard import GuardedHttp


def test_scope_is_drive_only():
    """There is no narrower scope that reads comments and writes replies."""
    assert SCOPES == ["https://www.googleapis.com/auth/drive"]


def test_missing_key_names_the_path_and_the_spec(tmp_path):
    missing = tmp_path / "sa-key.json"
    with pytest.raises(FileNotFoundError) as excinfo:
        load_service_account_credentials(missing)
    message = str(excinfo.value)
    assert str(missing) in message
    assert "2026-08-13-gdoc-ai-agent-design" in message


def test_service_account_mode_reads_the_key(tmp_path):
    missing = tmp_path / "sa-key.json"
    with pytest.raises(FileNotFoundError) as excinfo:
        load_credentials(key_path=missing, mode="service_account")
    assert str(missing) in str(excinfo.value)


def test_oauth_mode_reads_the_token():
    with patch("gdoc.auth.oauth.load", return_value="oauth-credentials") as load:
        assert load_credentials(mode="oauth") == "oauth-credentials"
    load.assert_called_once_with(SCOPES)


def test_the_mode_comes_from_config_when_not_given():
    with patch("gdoc.auth.load_config", return_value=Config(auth_mode="oauth")), patch(
        "gdoc.auth.oauth.load", return_value="oauth-credentials"
    ):
        assert load_credentials() == "oauth-credentials"


def test_a_missing_config_on_a_bare_machine_resolves_to_oauth(tmp_path):
    """gdoc read works with no config.json today, and must keep working.

    The credential paths are pointed at an empty directory on purpose. Without
    that, this asserts something about the developer's own machine.
    """
    with patch("gdoc.auth.load_config", side_effect=FileNotFoundError("no config")), patch(
        "gdoc.auth.DEFAULT_KEY_PATH", tmp_path / "sa-key.json"
    ), patch("gdoc.auth.oauth.DEFAULT_TOKEN_PATH", tmp_path / "oauth-token.json"), patch(
        "gdoc.auth.oauth.load", return_value="oauth-credentials"
    ):
        assert load_credentials() == "oauth-credentials"


def test_a_missing_config_beside_a_key_resolves_to_the_service_account(tmp_path):
    """The upgrade case again, through the door every command actually uses."""
    key = tmp_path / "sa-key.json"
    key.write_text("{}")
    with patch("gdoc.auth.load_config", side_effect=FileNotFoundError("no config")), patch(
        "gdoc.auth.DEFAULT_KEY_PATH", key
    ), patch("gdoc.auth.oauth.DEFAULT_TOKEN_PATH", tmp_path / "oauth-token.json"), patch(
        "gdoc.auth.load_service_account_credentials", return_value="sa-credentials"
    ):
        assert load_credentials() == "sa-credentials"


# ---------------------------------------------------------------------------
# resolve_auth_mode: which credential an unstated config means
# ---------------------------------------------------------------------------


def _files(tmp_path, key=False, token=False):
    """The two credential files, present or not. Returns their paths."""
    key_path = tmp_path / "sa-key.json"
    token_path = tmp_path / "oauth-token.json"
    if key:
        key_path.write_text("{}")
    if token:
        token_path.write_text("{}")
    return {"key_path": key_path, "token_path": token_path}


def test_a_stated_mode_wins_over_every_file(tmp_path):
    """The config is the author's word. Nothing on disk may overrule it."""
    paths = _files(tmp_path, key=True, token=True)
    assert resolve_auth_mode("service_account", **paths) == "service_account"


def test_a_stated_oauth_wins_even_with_only_a_key_present(tmp_path):
    paths = _files(tmp_path, key=True)
    assert resolve_auth_mode("oauth", **paths) == "oauth"


def test_an_unstated_mode_with_a_key_and_no_token_is_the_service_account(tmp_path):
    """The upgrade case. A working service_account install must not flip."""
    paths = _files(tmp_path, key=True)
    assert resolve_auth_mode(None, **paths) == "service_account"


def test_an_unstated_mode_with_nothing_installed_is_oauth(tmp_path):
    """The new install. No files, no sharing step, so oauth is the default."""
    paths = _files(tmp_path)
    assert resolve_auth_mode(None, **paths) == "oauth"


def test_a_token_beats_a_key_when_the_mode_is_unstated(tmp_path):
    """Signing in is the more deliberate act, so it decides."""
    paths = _files(tmp_path, key=True, token=True)
    assert resolve_auth_mode(None, **paths) == "oauth"


def test_an_unstated_mode_with_only_a_token_is_oauth(tmp_path):
    paths = _files(tmp_path, token=True)
    assert resolve_auth_mode(None, **paths) == "oauth"


def test_a_stated_mode_that_is_not_recognised_is_refused(tmp_path):
    """Falling through to inference would let files pick the wider credential.

    A hyphen typo in auth_mode resolved to oauth on a machine holding a token,
    which is full Drive scope, when the author had asked for the narrow one.
    """
    paths = _files(tmp_path, key=True, token=True)
    with pytest.raises(ValueError) as excinfo:
        resolve_auth_mode("service-account", **paths)
    message = str(excinfo.value)
    assert "service-account" in message
    assert "oauth" in message


def test_a_broken_config_reaches_the_caller_rather_than_being_inferred_past():
    """load_credentials must not resolve a config it could not understand."""
    with patch(
        "gdoc.auth.load_config", side_effect=ValueError("auth_mode in x is 'magic'")
    ), pytest.raises(ValueError):
        load_credentials()


def test_load_credentials_infers_the_service_account_from_the_key(tmp_path):
    """End to end: an upgraded config with no auth_mode reads the key."""
    key = tmp_path / "sa-key.json"
    key.write_text("{}")
    with patch("gdoc.auth.load_config", return_value=Config()), patch(
        "gdoc.auth.oauth.DEFAULT_TOKEN_PATH", tmp_path / "oauth-token.json"
    ), patch(
        "gdoc.auth.load_service_account_credentials", return_value="sa-credentials"
    ) as load_key:
        assert load_credentials(key_path=key) == "sa-credentials"
    load_key.assert_called_once_with(key)


def test_the_client_is_always_guarded():
    with patch("gdoc.auth.build") as build:
        drive_service(credentials=MagicMock())
    assert isinstance(build.call_args.kwargs["http"], GuardedHttp)


def test_the_guard_starts_with_the_ids_it_was_given():
    with patch("gdoc.auth.build") as build:
        drive_service(credentials=MagicMock(), doc_ids=["1AbC", "1DeF"])
    assert build.call_args.kwargs["http"].allowed == frozenset({"1AbC", "1DeF"})


def test_no_ids_means_an_empty_set():
    """generate names no input document. It creates, and learns from that."""
    with patch("gdoc.auth.build") as build:
        drive_service(credentials=MagicMock())
    assert build.call_args.kwargs["http"].allowed == frozenset()


def test_a_single_id_may_be_passed_as_a_string():
    """Every call site but one has exactly one document, so do not make it wrap."""
    with patch("gdoc.auth.build") as build:
        drive_service(credentials=MagicMock(), doc_ids="1AbC")
    assert build.call_args.kwargs["http"].allowed == frozenset({"1AbC"})


def test_the_guard_is_installed_without_a_config():
    """Not knowing must never resolve to the unguarded client."""
    with patch("gdoc.auth.load_config", side_effect=FileNotFoundError), patch(
        "gdoc.auth.build"
    ) as build:
        drive_service(credentials=MagicMock())
    assert isinstance(build.call_args.kwargs["http"], GuardedHttp)


def test_discovery_caching_stays_off():
    with patch("gdoc.auth.build") as build:
        drive_service(credentials=MagicMock())
    assert build.call_args.kwargs["cache_discovery"] is False


def test_credentials_are_not_passed_beside_http():
    """googleapiclient refuses both at once."""
    with patch("gdoc.auth.build") as build:
        drive_service(credentials=MagicMock())
    assert "credentials" not in build.call_args.kwargs
