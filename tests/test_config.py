import json
from unittest.mock import patch

import pytest

from gdoc.config import Config, load_config, write_auth_mode
from gdoc.render import profiles


def test_loads_the_output_folder(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    assert load_config(path) == Config(output_folder_id="0AFolderId")


def test_output_folder_may_be_absent(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({}))
    assert load_config(path).output_folder_id is None


def test_a_config_without_display_name_loads(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    assert load_config(path) == Config(output_folder_id="0AFolderId")


def test_an_old_config_with_display_name_still_loads(tmp_path):
    """Nail's live config has the key. Loading must not start failing on it."""
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"display_name": "Nail Khusnullin", "output_folder_id": None}))
    assert load_config(path).output_folder_id is None


def test_the_template_defaults_to_the_bundled_house_style(tmp_path):
    """A config that says nothing still gets the house template, not bare pandoc."""
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    assert load_config(path).template == "altery-group-policy-v1.0"
    assert load_config(path).template == profiles.DEFAULT_TEMPLATE


def test_a_config_can_choose_another_template(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"template": "/tmp/other-profile"}))
    assert load_config(path).template == "/tmp/other-profile"


def test_a_config_can_turn_the_template_off(tmp_path):
    """'none' is how a user keeps the plain pandoc path."""
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"template": "none"}))
    assert load_config(path).template == profiles.NO_TEMPLATE


def test_missing_file_names_the_path_and_an_example(tmp_path):
    path = tmp_path / "config.json"
    with pytest.raises(FileNotFoundError) as excinfo:
        load_config(path)
    assert str(path) in str(excinfo.value)
    assert "output_folder_id" in str(excinfo.value)


def test_config_is_immutable(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    config = load_config(path)
    with pytest.raises(Exception):
        config.output_folder_id = "0AOther"


def test_an_absent_auth_mode_is_unstated_rather_than_oauth(tmp_path):
    """A config that never mentioned the key must not pick the credential.

    Defaulting here would flip a working service_account install to oauth the
    moment it upgraded. Which credential an unstated config means depends on
    which credential files exist, and that is auth.resolve_auth_mode's job.
    """
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    assert load_config(path).auth_mode is None


def test_auth_mode_can_be_the_service_account(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"auth_mode": "service_account"}))
    assert load_config(path).auth_mode == "service_account"


def test_an_unknown_auth_mode_is_refused_and_names_both_options(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"auth_mode": "magic"}))
    with pytest.raises(ValueError) as excinfo:
        load_config(path)
    message = str(excinfo.value)
    assert "magic" in message
    assert "oauth" in message
    assert "service_account" in message


def test_nails_live_config_still_loads_without_the_new_key(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"display_name": "Nail Khusnullin", "output_folder_id": None}))
    config = load_config(path)
    assert config.auth_mode is None


# ---------------------------------------------------------------------------
# write_auth_mode
# ---------------------------------------------------------------------------


def test_write_auth_mode_keeps_every_other_key(tmp_path):
    """A credential switch must not cost the folder or the template."""
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"output_folder_id": "0AFolderId", "template": "none"}))
    write_auth_mode("oauth", path)
    data = json.loads(path.read_text())
    assert data == {
        "output_folder_id": "0AFolderId",
        "template": "none",
        "auth_mode": "oauth",
    }


def test_write_auth_mode_keeps_keys_the_tool_no_longer_reads(tmp_path):
    """display_name is gone from Config, and rewriting must not delete it."""
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"display_name": "Nail Khusnullin"}))
    write_auth_mode("oauth", path)
    assert json.loads(path.read_text())["display_name"] == "Nail Khusnullin"


def test_write_auth_mode_creates_the_file_when_there_is_none(tmp_path):
    path = tmp_path / "sub" / "config.json"
    write_auth_mode("oauth", path)
    assert json.loads(path.read_text()) == {"auth_mode": "oauth"}


def test_write_auth_mode_replaces_the_mode_already_there(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"auth_mode": "service_account"}))
    write_auth_mode("oauth", path)
    assert json.loads(path.read_text())["auth_mode"] == "oauth"


def test_write_auth_mode_refuses_a_mode_that_is_not_one_of_the_two(tmp_path):
    path = tmp_path / "config.json"
    with pytest.raises(ValueError) as excinfo:
        write_auth_mode("magic", path)
    assert "magic" in str(excinfo.value)
    assert not path.exists()


def test_write_auth_mode_refuses_a_file_holding_something_other_than_an_object(tmp_path):
    """A JSON list parses fine and then has no keys to carry over."""
    path = tmp_path / "config.json"
    path.write_text("[1, 2]")
    with pytest.raises(ValueError) as excinfo:
        write_auth_mode("oauth", path)
    assert str(path) in str(excinfo.value)
    assert path.read_text() == "[1, 2]"


def test_write_auth_mode_leaves_an_unreadable_file_alone(tmp_path):
    """Never overwrite settings that could not be read, or they are lost."""
    path = tmp_path / "config.json"
    path.write_text("{not json")
    with pytest.raises(ValueError):
        write_auth_mode("oauth", path)
    assert path.read_text() == "{not json"


def test_write_auth_mode_returns_the_mode_it_replaced(tmp_path):
    """The caller says out loud what it changed, so nothing is silent."""
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"auth_mode": "service_account"}))
    assert write_auth_mode("oauth", path) == "service_account"


def test_write_auth_mode_reports_no_previous_mode_when_unstated(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    assert write_auth_mode("oauth", path) is None


def test_a_write_that_fails_leaves_the_old_settings_in_place(tmp_path):
    """Truncating first would lose settings that were readable a moment ago.

    A full disk or a signal mid-write is the case. The recovery path was worse
    than the failure: a 0-byte file cannot be parsed, so the next write refuses.
    """
    path = tmp_path / "config.json"
    original = json.dumps({"display_name": "Nail", "output_folder_id": "0AFolderId"})
    path.write_text(original)
    with patch("gdoc.config.json.dump", side_effect=OSError("No space left on device")):
        with pytest.raises(OSError):
            write_auth_mode("oauth", path)
    assert json.loads(path.read_text())["output_folder_id"] == "0AFolderId"


def test_a_failed_write_leaves_nothing_behind(tmp_path):
    """The half-written file is the tool's, so it does not become the user's mess."""
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    with patch("gdoc.config.json.dump", side_effect=OSError("nope")):
        with pytest.raises(OSError):
            write_auth_mode("oauth", path)
    assert [p.name for p in tmp_path.iterdir()] == ["config.json"]


def test_write_auth_mode_names_the_file_it_could_not_parse(tmp_path):
    path = tmp_path / "config.json"
    path.write_text("{not json")
    with pytest.raises(ValueError) as excinfo:
        write_auth_mode("oauth", path)
    assert str(path) in str(excinfo.value)


def test_the_directory_it_creates_is_private(tmp_path):
    """It holds the token and the key, so it is 0700 like the tool's own."""
    path = tmp_path / "gdoc-agent" / "config.json"
    write_auth_mode("oauth", path)
    assert path.parent.stat().st_mode & 0o777 == 0o700


def test_a_config_that_is_not_an_object_is_refused_by_name(tmp_path):
    """json.loads accepts a list, and everything downstream expects .get."""
    path = tmp_path / "config.json"
    path.write_text("[1, 2]")
    with pytest.raises(ValueError) as excinfo:
        load_config(path)
    assert str(path) in str(excinfo.value)


def test_the_written_config_is_private(tmp_path):
    path = tmp_path / "config.json"
    write_auth_mode("oauth", path)
    assert path.stat().st_mode & 0o777 == 0o600
