import json

import pytest

from gdoc.config import Config, load_config


def test_loads_display_name_and_folder(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"display_name": "Test User", "output_folder_id": "0AFolderId"}))
    config = load_config(path)
    assert config == Config(display_name="Test User", output_folder_id="0AFolderId")


def test_output_folder_may_be_absent(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"display_name": "Test User"}))
    assert load_config(path).output_folder_id is None


def test_missing_file_names_the_path_and_the_required_key(tmp_path):
    path = tmp_path / "config.json"
    with pytest.raises(FileNotFoundError) as excinfo:
        load_config(path)
    assert str(path) in str(excinfo.value)
    assert "display_name" in str(excinfo.value)


def test_missing_display_name_is_rejected(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    with pytest.raises(ValueError, match="display_name"):
        load_config(path)


def test_config_is_immutable(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"display_name": "Test User"}))
    config = load_config(path)
    with pytest.raises(Exception):
        config.display_name = "Someone Else"
