import os
import subprocess

import pytest

from tools.gdoc.mirror import MirrorConflict, mirror_path, slugify, write_mirror


def git(repo, *args):
    subprocess.run(["git", *args], cwd=repo, check=True, capture_output=True)


@pytest.fixture
def repo(tmp_path):
    git(tmp_path, "init")
    git(tmp_path, "config", "user.email", "test@example.com")
    git(tmp_path, "config", "user.name", "Test")
    (tmp_path / "seed.txt").write_text("seed")
    git(tmp_path, "add", ".")
    git(tmp_path, "commit", "-m", "seed")
    return tmp_path


def test_slugify_lowercases_and_hyphenates():
    assert slugify("Safeguarding Policy v3") == "safeguarding-policy-v3"


def test_slugify_drops_punctuation_and_collapses_gaps():
    assert slugify("  Q4 Budget (draft!) ") == "q4-budget-draft"


def test_slugify_falls_back_when_nothing_survives():
    assert slugify("!!!") == "untitled"


def test_mirror_path_is_under_docs_gdoc(repo):
    assert mirror_path(repo, "policy") == repo / "docs" / "gdoc" / "policy" / "mirror.md"


def test_write_mirror_creates_the_file_and_parents(repo):
    path = write_mirror(repo, "policy", "# Title\n")
    assert path.read_text() == "# Title\n"


def test_write_mirror_overwrites_a_committed_mirror(repo):
    write_mirror(repo, "policy", "# One\n")
    git(repo, "add", ".")
    git(repo, "commit", "-m", "mirror")
    write_mirror(repo, "policy", "# Two\n")
    assert mirror_path(repo, "policy").read_text() == "# Two\n"


def test_write_mirror_refuses_to_clobber_uncommitted_edits(repo):
    write_mirror(repo, "policy", "# One\n")
    git(repo, "add", ".")
    git(repo, "commit", "-m", "mirror")
    mirror_path(repo, "policy").write_text("# Edited by hand\n")
    with pytest.raises(MirrorConflict, match="uncommitted"):
        write_mirror(repo, "policy", "# From Drive\n")


def test_force_overrides_the_guard(repo):
    write_mirror(repo, "policy", "# One\n")
    git(repo, "add", ".")
    git(repo, "commit", "-m", "mirror")
    mirror_path(repo, "policy").write_text("# Edited by hand\n")
    write_mirror(repo, "policy", "# From Drive\n", force=True)
    assert mirror_path(repo, "policy").read_text() == "# From Drive\n"


def test_write_mirror_raises_when_git_cannot_be_consulted(tmp_path):
    # tmp_path is not a git repo, so git will fail with a non-zero exit code.
    # The guard only fires when the file already exists, so create it first.
    path = mirror_path(tmp_path, "policy")
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("# Existing\n")
    with pytest.raises(MirrorConflict, match="git could not be consulted"):
        write_mirror(tmp_path, "policy", "# From Drive\n")


def test_force_rescues_when_git_cannot_be_consulted(tmp_path):
    # tmp_path is outside any git repository, so is_dirty would raise.
    # force=True must short-circuit before is_dirty is called and still write.
    path = mirror_path(tmp_path, "policy")
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("# Existing\n")
    result = write_mirror(tmp_path, "policy", "# From Drive\n", force=True)
    assert result.read_text() == "# From Drive\n"


def test_write_mirror_is_atomic(repo, monkeypatch):
    # Create a committed mirror so is_dirty returns False on the next call.
    write_mirror(repo, "policy", "original\n")
    git(repo, "add", ".")
    git(repo, "commit", "-m", "mirror")

    def boom(fd, data):
        raise OSError("simulated write failure")

    monkeypatch.setattr(os, "write", boom)

    with pytest.raises(OSError, match="simulated"):
        write_mirror(repo, "policy", "new content\n")

    # Original content survives intact.
    assert mirror_path(repo, "policy").read_text() == "original\n"
    # No leftover temp file in the directory.
    assert list(mirror_path(repo, "policy").parent.glob("*.tmp")) == []
