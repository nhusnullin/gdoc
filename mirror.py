"""Write the markdown mirror of a document, without losing local edits."""

import os
import re
import subprocess
import tempfile
from pathlib import Path

MIRROR_DIR = Path("docs") / "gdoc"

_NON_WORD = re.compile(r"[^a-z0-9]+")


class MirrorConflict(RuntimeError):
    """The mirror on disk has uncommitted changes, so overwriting could lose work."""


def slugify(name: str) -> str:
    slug = _NON_WORD.sub("-", (name or "").lower()).strip("-")
    return slug or "untitled"


def mirror_path(repo_root: Path, slug: str) -> Path:
    return repo_root / MIRROR_DIR / slug / "mirror.md"


def is_dirty(path: Path) -> bool:
    """True when git reports uncommitted changes for this path.

    An untracked file is not dirty in the sense that matters here: there is
    nothing committed to lose. Only tracked-and-modified counts.

    Raises MirrorConflict if git cannot be consulted (e.g. the path is outside
    a git repository). Not knowing whether edits exist must never resolve to
    "overwrite". Pass force=True to write_mirror to bypass this check.
    """
    result = subprocess.run(
        ["git", "status", "--porcelain", "--", str(path)],
        cwd=path.parent,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise MirrorConflict(
            f"git could not be consulted for {path}. "
            "Verify the path is inside a git repository, "
            "or pass force=True to write_mirror to overwrite without checking."
        )
    return any(line and not line.startswith("??") for line in result.stdout.splitlines())


def write_mirror(repo_root: Path, slug: str, markdown: str, force: bool = False) -> Path:
    """Write the mirror, refusing when that would discard uncommitted edits.

    The markdown is what Nail approves and edits, so a refresh from Drive must
    not overwrite work that is not yet in git. git already knows the answer, so
    no extra state is needed.
    """
    path = mirror_path(repo_root, slug)
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists() and not force and is_dirty(path):
        raise MirrorConflict(
            f"{path} has uncommitted changes. Commit or discard them, "
            "or pass force=True to overwrite."
        )
    fd, tmp_str = tempfile.mkstemp(dir=path.parent, suffix=".tmp")
    tmp = Path(tmp_str)
    try:
        os.write(fd, markdown.encode())
        os.close(fd)
        os.replace(tmp, path)
    except Exception:
        try:
            os.close(fd)
        except OSError:
            pass
        tmp.unlink(missing_ok=True)
        raise
    return path
