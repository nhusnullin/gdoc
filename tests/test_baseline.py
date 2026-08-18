import os

import pytest

from gdoc.baseline import (
    BaselineConflict,
    baseline_path,
    slugify,
    write_baseline,
)


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


def test_write_baseline_creates_the_file_and_parents(tmp_path):
    path = write_baseline(tmp_path, "policy", "# Title\n")
    assert path.read_text() == "# Title\n"


# ------------------------------------------------------------ the one guard --
# Issue #24. The guard used to ask git whether the file held uncommitted work.
# It no longer asks anything: the root is usually not a repository, and where it
# is synced by Dropbox or Nextcloud git cannot see the rewrite anyway. So an
# existing baseline is never overwritten by accident, in any directory.


def test_an_existing_baseline_is_never_overwritten_without_force(tmp_path):
    write_baseline(tmp_path, "my-doc", "# First\n")
    with pytest.raises(BaselineConflict, match="force"):
        write_baseline(tmp_path, "my-doc", "# Second\n")


def test_the_refusal_names_the_file(tmp_path):
    write_baseline(tmp_path, "my-doc", "# First\n")
    with pytest.raises(BaselineConflict, match="baseline.md"):
        write_baseline(tmp_path, "my-doc", "# Second\n")


def test_the_refusal_does_not_talk_about_git(tmp_path):
    """It used to say "uncommitted changes" to someone with no repository."""
    write_baseline(tmp_path, "my-doc", "# First\n")
    with pytest.raises(BaselineConflict) as caught:
        write_baseline(tmp_path, "my-doc", "# Second\n")
    assert "git" not in str(caught.value).lower()
    assert "commit" not in str(caught.value).lower()


def test_force_overwrites(tmp_path):
    write_baseline(tmp_path, "my-doc", "# First\n")
    path = write_baseline(tmp_path, "my-doc", "# Second\n", force=True)
    assert path.read_text() == "# Second\n"


def test_nothing_here_runs_git(tmp_path, monkeypatch):
    """The guard is disk state now. A machine without git behaves identically."""
    import subprocess

    def refuse(*args, **kwargs):
        raise AssertionError("write_baseline ran a subprocess")

    monkeypatch.setattr(subprocess, "run", refuse)
    write_baseline(tmp_path, "my-doc", "# First\n")
    write_baseline(tmp_path, "my-doc", "# Second\n", force=True)


# ------------------------------------------------------------- writing safely --
def test_write_baseline_is_atomic(tmp_path, monkeypatch):
    write_baseline(tmp_path, "policy", "original\n")

    def boom(fd, data):
        raise OSError("simulated write failure")

    monkeypatch.setattr(os, "write", boom)

    with pytest.raises(OSError, match="simulated"):
        write_baseline(tmp_path, "policy", "new content\n", force=True)

    assert baseline_path(tmp_path, "policy").read_text() == "original\n"
    assert list(baseline_path(tmp_path, "policy").parent.glob("*.tmp")) == []


def test_write_baseline_preserves_the_existing_file_mode(tmp_path):
    path = write_baseline(tmp_path, "policy", "# One\n")
    path.chmod(0o600)
    write_baseline(tmp_path, "policy", "# Two\n", force=True)
    assert path.stat().st_mode & 0o777 == 0o600


def test_write_baseline_creates_a_readable_file(tmp_path):
    path = write_baseline(tmp_path, "policy", "# One\n")
    assert path.stat().st_mode & 0o777 == 0o644
