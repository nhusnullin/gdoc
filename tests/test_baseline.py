import os
import subprocess

import pytest

from gdoc.baseline import (
    BaselineConflict,
    baseline_path,
    has_git,
    is_dirty,
    slugify,
    write_baseline,
)


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


def test_files_go_in_a_dot_gdoc_directory(tmp_path):
    path = baseline_path(tmp_path, "my-doc")
    assert path == tmp_path / ".gdoc" / "my-doc" / "baseline.md"


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


def test_has_git_is_true_inside_a_repository(repo):
    assert has_git(repo) is True


def test_has_git_is_false_outside_a_repository(tmp_path):
    assert has_git(tmp_path) is False


def test_is_dirty_still_raises_when_git_cannot_answer(tmp_path):
    """git exists but the command fails. That is a real fault and must stay loud."""
    path = tmp_path / "policy.md"
    path.write_text("# Existing\n")
    with pytest.raises(BaselineConflict, match="git could not be consulted"):
        is_dirty(path)


def test_writes_a_new_file_outside_a_git_repo(tmp_path):
    path = write_baseline(tmp_path, "my-doc", "# Title\n")
    assert path.read_text() == "# Title\n"


def test_refuses_to_overwrite_outside_a_git_repo(tmp_path):
    write_baseline(tmp_path, "my-doc", "# First\n")
    with pytest.raises(BaselineConflict, match="force"):
        write_baseline(tmp_path, "my-doc", "# Second\n")


def test_the_refusal_says_git_is_unavailable(tmp_path):
    """Otherwise the message reads as a complaint about uncommitted work."""
    write_baseline(tmp_path, "my-doc", "# First\n")
    with pytest.raises(BaselineConflict, match="git"):
        write_baseline(tmp_path, "my-doc", "# Second\n")


def test_force_overwrites_outside_a_git_repo(tmp_path):
    write_baseline(tmp_path, "my-doc", "# First\n")
    path = write_baseline(tmp_path, "my-doc", "# Second\n", force=True)
    assert path.read_text() == "# Second\n"


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
