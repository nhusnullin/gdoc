"""Write the snapshot a later export is compared against.

The file records the document as it was generated, at the one moment the
document and the source markdown provably match. A later apply diffs it against
a fresh export to see what was edited directly in the document. Both sides carry
the same pandoc round-trip distortion, so it cancels.

It is not a mirror. Nothing reads it to learn the current state of the document.
"""

import os
import re
import subprocess
import tempfile
from pathlib import Path

# A dot directory because the root can be any directory: docs/ cannot be assumed
# free or appropriate, and a dot-namespace signals tool-owned. It also keeps
# these files out of Obsidian's search and graph.
GDOC_DIR = Path(".gdoc")

# mkstemp creates 0600.  Without an explicit chmod the rename would silently
# tighten permissions on a baseline that was readable before.
_DEFAULT_MODE = 0o644

_NON_WORD = re.compile(r"[^a-z0-9]+")


class BaselineConflict(RuntimeError):
    """The file on disk has uncommitted changes, so overwriting could lose work."""


def slugify(name: str) -> str:
    slug = _NON_WORD.sub("-", (name or "").lower()).strip("-")
    return slug or "untitled"


def baseline_path(repo_root: Path, slug: str) -> Path:
    return repo_root / GDOC_DIR / slug / "baseline.md"


def is_dirty(path: Path) -> bool:
    """True when git reports uncommitted changes for this path.

    An untracked file is not dirty in the sense that matters here: there is
    nothing committed to lose. Only tracked-and-modified counts.

    Raises BaselineConflict if git cannot be consulted (e.g. the path is outside
    a git repository). Not knowing whether edits exist must never resolve to
    "overwrite". Pass force=True to write_baseline to bypass this check.
    """
    result = subprocess.run(
        ["git", "status", "--porcelain", "--", str(path)],
        cwd=path.parent,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise BaselineConflict(
            f"git could not be consulted for {path}. "
            "Verify the path is inside a git repository, "
            "or pass force=True to write_baseline to overwrite without checking."
        )
    return any(line and not line.startswith("??") for line in result.stdout.splitlines())


def write_baseline(repo_root: Path, slug: str, markdown: str, force: bool = False) -> Path:
    """Write the baseline, refusing when that would discard uncommitted edits.

    git already knows whether the file on disk holds work that is not saved
    anywhere else, so no extra state is needed.
    """
    path = baseline_path(repo_root, slug)
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists() and not force and is_dirty(path):
        raise BaselineConflict(
            f"{path} has uncommitted changes. Commit or discard them, "
            "or pass force=True to overwrite."
        )
    mode = path.stat().st_mode & 0o777 if path.exists() else _DEFAULT_MODE
    fd, tmp_str = tempfile.mkstemp(dir=path.parent, suffix=".tmp")
    tmp = Path(tmp_str)
    try:
        os.write(fd, markdown.encode())
        os.close(fd)
        os.chmod(tmp, mode)
        os.replace(tmp, path)
    except Exception:
        try:
            os.close(fd)
        except OSError:
            pass
        tmp.unlink(missing_ok=True)
        raise
    return path
