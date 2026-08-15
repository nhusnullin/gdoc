from unittest.mock import MagicMock, patch

import pytest

from gdoc.auth import (
    SCOPES,
    drive_service,
    load_credentials,
    load_service_account_credentials,
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


def test_a_missing_config_still_resolves_to_oauth():
    """gdoc read works with no config.json today, and must keep working."""
    with patch("gdoc.auth.load_config", side_effect=FileNotFoundError("no config")), patch(
        "gdoc.auth.oauth.load", return_value="oauth-credentials"
    ):
        assert load_credentials() == "oauth-credentials"


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
