import json

import pytest

from gdoc.config import Config, load_config


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
