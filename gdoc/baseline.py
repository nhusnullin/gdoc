"""Write the snapshot a later export is compared against.

The file records the document as it was generated, at the one moment the
document and the source markdown provably match. A later apply diffs it against
a fresh export to see what was edited directly in the document. Both sides carry
the same pandoc round-trip distortion, so it cancels.

It is not a mirror. Nothing reads it to learn the current state of the document.
"""

import os
import re
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
    """Overwriting this baseline could lose something nothing else holds."""


def slugify(name: str) -> str:
    slug = _NON_WORD.sub("-", (name or "").lower()).strip("-")
    return slug or "untitled"


def baseline_path(repo_root: Path, slug: str) -> Path:
    return repo_root / GDOC_DIR / slug / "baseline.md"


def write_baseline(repo_root: Path, slug: str, markdown: str, force: bool = False) -> Path:
    """Write the baseline, refusing when that would discard work.

    An existing file is kept. There is no way to tell an edit from a stale copy,
    and not knowing must never resolve to "overwrite".

    This used to ask git whether the file held uncommitted work, and overwrite it
    when git said no. Two reasons that is gone. The root is usually not a
    repository, so the answer was normally "cannot tell" anyway. And where the
    root is a synced folder, Dropbox or Nextcloud rewriting the file underneath
    is exactly the case git cannot see, so the check read as safety while
    providing none.

    The one caller that means to overwrite says so: `generate` passes force at the
    moment the document and the markdown provably match, which is the only honest
    time to take a snapshot.
    """
    path = baseline_path(repo_root, slug)
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists() and not force:
        raise BaselineConflict(
            f"{path} already exists, and there is no way to tell an edit from a "
            "stale copy. Move the file, or pass force to overwrite it."
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
