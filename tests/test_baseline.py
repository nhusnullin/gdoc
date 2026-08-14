import os
import subprocess

import pytest

from gdoc.baseline import BaselineConflict, baseline_path, slugify, write_baseline


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


def test_baseline_path_is_under_docs_gdoc(repo):
    assert baseline_path(repo, "policy") == repo / "docs" / "gdoc" / "policy" / "baseline.md"


def test_the_written_file_is_named_baseline(tmp_path):
    path = write_baseline(tmp_path, "my-doc", "# Title\n")
    assert path.name == "baseline.md"


def test_write_baseline_creates_the_file_and_parents(repo):
    path = write_baseline(repo, "policy", "# Title\n")
    assert path.read_text() == "# Title\n"


def test_write_baseline_overwrites_a_committed_baseline(repo):
    write_baseline(repo, "policy", "# One\n")
    git(repo, "add", ".")
    git(repo, "commit", "-m", "baseline")
    write_baseline(repo, "policy", "# Two\n")
    assert baseline_path(repo, "policy").read_text() == "# Two\n"


def test_write_baseline_refuses_to_clobber_uncommitted_edits(repo):
    write_baseline(repo, "policy", "# One\n")
    git(repo, "add", ".")
    git(repo, "commit", "-m", "baseline")
    baseline_path(repo, "policy").write_text("# Edited by hand\n")
    with pytest.raises(BaselineConflict, match="uncommitted"):
        write_baseline(repo, "policy", "# From Drive\n")


def test_force_overrides_the_guard(repo):
    write_baseline(repo, "policy", "# One\n")
    git(repo, "add", ".")
    git(repo, "commit", "-m", "baseline")
    baseline_path(repo, "policy").write_text("# Edited by hand\n")
    write_baseline(repo, "policy", "# From Drive\n", force=True)
    assert baseline_path(repo, "policy").read_text() == "# From Drive\n"


def test_write_baseline_raises_when_git_cannot_be_consulted(tmp_path):
    # tmp_path is not a git repo, so git will fail with a non-zero exit code.
    # The guard only fires when the file already exists, so create it first.
    path = baseline_path(tmp_path, "policy")
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("# Existing\n")
    with pytest.raises(BaselineConflict, match="git could not be consulted"):
        write_baseline(tmp_path, "policy", "# From Drive\n")


def test_force_rescues_when_git_cannot_be_consulted(tmp_path):
    # tmp_path is outside any git repository, so is_dirty would raise.
    # force=True must short-circuit before is_dirty is called and still write.
    path = baseline_path(tmp_path, "policy")
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("# Existing\n")
    result = write_baseline(tmp_path, "policy", "# From Drive\n", force=True)
    assert result.read_text() == "# From Drive\n"


def test_write_baseline_is_atomic(repo, monkeypatch):
    # Create a committed baseline so is_dirty returns False on the next call.
    write_baseline(repo, "policy", "original\n")
    git(repo, "add", ".")
    git(repo, "commit", "-m", "baseline")

    def boom(fd, data):
        raise OSError("simulated write failure")

    monkeypatch.setattr(os, "write", boom)

    with pytest.raises(OSError, match="simulated"):
        write_baseline(repo, "policy", "new content\n")

    # Original content survives intact.
    assert baseline_path(repo, "policy").read_text() == "original\n"
    # No leftover temp file in the directory.
    assert list(baseline_path(repo, "policy").parent.glob("*.tmp")) == []


def test_write_baseline_preserves_the_existing_file_mode(repo):
    path = write_baseline(repo, "policy", "# One\n")
    path.chmod(0o644)
    git(repo, "add", ".")
    git(repo, "commit", "-m", "baseline")
    write_baseline(repo, "policy", "# Two\n")
    assert path.stat().st_mode & 0o777 == 0o644


def test_write_baseline_creates_a_readable_file(repo):
    path = write_baseline(repo, "policy", "# One\n")
    assert path.stat().st_mode & 0o777 == 0o644
