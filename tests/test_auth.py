import pytest

from tools.gdoc.auth import SCOPES, load_credentials


def test_scope_is_drive_only():
    assert SCOPES == ["https://www.googleapis.com/auth/drive"]


def test_missing_key_names_the_path_and_the_spec(tmp_path):
    missing = tmp_path / "sa-key.json"
    with pytest.raises(FileNotFoundError) as excinfo:
        load_credentials(missing)
    message = str(excinfo.value)
    assert str(missing) in message
    assert "2026-08-13-gdoc-ai-agent-design" in message
