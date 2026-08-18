"""What was edited in the document after it was generated.

The baseline records the document at the one moment it and the source markdown
provably matched. Diffing a fresh export against it shows what Nail changed in
the browser. Both sides come out of the same export path, so the distortion that
path introduces is on both sides and cancels.

The result is deliberately coarse. A hunk names the heading it sits under and
carries the text before and after, because the source markdown is hand-written
and not export-shaped: it is located by what it says, never by line number.
Deciding where a hunk belongs in the source is judgment, and judgment belongs to
the agent reading this, not to a difflib call.
"""

import difflib
import re
from dataclasses import dataclass

HEADING_RE = re.compile(r"^\s{0,3}#{1,6}\s")

# A blank line separates blocks. Two blank lines and one are the same separator:
# an export moves them around and that is not an edit.
_BLOCK_SPLIT = re.compile(r"\n\s*\n")


@dataclass(frozen=True)
class Hunk:
    kind: str          # change, insert or delete
    heading: str       # the nearest heading above it, or "" before the first
    before: str        # the baseline text, empty for an insert
    after: str         # the exported text, empty for a delete


def blocks(markdown: str) -> list[str]:
    """Split into paragraphs, with the whitespace noise taken out."""
    found = []
    for block in _BLOCK_SPLIT.split(markdown or ""):
        stripped = "\n".join(line.rstrip() for line in block.strip().splitlines())
        if stripped:
            found.append(stripped)
    return found


def _heading_before(blocks_list: list[str], index: int) -> str:
    """The nearest heading at or above this block."""
    for candidate in reversed(blocks_list[: index + 1]):
        first_line = candidate.splitlines()[0]
        if HEADING_RE.match(first_line):
            return first_line.strip()
    return ""


def diff_markdown(baseline: str, current: str) -> tuple[Hunk, ...]:
    """The edits that turn the baseline into the current export."""
    old, new = blocks(baseline), blocks(current)
    hunks = []
    matcher = difflib.SequenceMatcher(a=old, b=new, autojunk=False)
    for tag, i1, i2, j1, j2 in matcher.get_opcodes():
        if tag == "equal":
            continue
        kind = {"replace": "change", "delete": "delete", "insert": "insert"}[tag]
        # An insert has no old block to sit under, so its heading comes from
        # where it landed, which is the document Nail was looking at.
        heading = (
            _heading_before(new, j1) if kind == "insert" else _heading_before(old, i1)
        )
        hunks.append(
            Hunk(
                kind=kind,
                heading=heading,
                before="\n\n".join(old[i1:i2]),
                after="\n\n".join(new[j1:j2]),
            )
        )
    return tuple(hunks)
