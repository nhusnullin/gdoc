import json

import pytest

from gdoc.config import Config, load_config
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
