import pytest

from gdoc.render import profiles


def test_a_bundled_name_resolves_to_its_master_docx():
    master = profiles.resolve("altery-group-policy-v1.0")
    assert master.name == "template.docx"
    assert master.is_file()


def test_the_default_is_the_altery_profile():
    assert profiles.resolve(profiles.DEFAULT_TEMPLATE).is_file()


def test_none_means_no_template():
    assert profiles.resolve("none") is None


def test_a_path_to_a_docx_is_used_as_the_master(tmp_path):
    master = tmp_path / "house.docx"
    master.write_bytes(b"PK\x03\x04")
    assert profiles.resolve(str(master)) == master


def test_a_path_to_a_profile_directory_finds_its_template(tmp_path):
    profile = tmp_path / "house"
    profile.mkdir()
    (profile / "template.docx").write_bytes(b"PK\x03\x04")
    assert profiles.resolve(str(profile)) == profile / "template.docx"


def test_an_unknown_name_lists_what_is_available():
    with pytest.raises(profiles.ProfileError) as exc:
        profiles.resolve("no-such-template")
    assert "altery-group-policy-v1.0" in str(exc.value)
