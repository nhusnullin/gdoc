"""The document to markdown pairing, stored in the markdown file's frontmatter.

Frontmatter rather than a central index because the skill is global and runs in
several repos. An index would have to live outside them all and would go stale
silently when a file moves.
"""

import datetime
from dataclasses import dataclass, replace
from pathlib import Path

import yaml

_FENCE = "---"


@dataclass(frozen=True)
class Pairing:
    doc_id: str
    synced: str | None = None
    versions: tuple[dict, ...] = ()


def _split(text: str) -> tuple[dict, str]:
    """Return (frontmatter dict, body). Missing frontmatter gives an empty dict."""
    if not text.startswith(_FENCE):
        return {}, text
    parts = text.split(_FENCE, 2)
    if len(parts) < 3:
        return {}, text
    return yaml.safe_load(parts[1]) or {}, parts[2].lstrip("\n")


def _str(value: object) -> str | None:
    """Convert YAML-parsed date objects to ISO strings; return strings unchanged."""
    if value is None:
        return None
    if isinstance(value, (datetime.date, datetime.datetime)):
        return value.isoformat()
    return str(value)


def read_pairing(md_path: Path) -> Pairing | None:
    front, _ = _split(md_path.read_text())
    doc_id = front.get("gdoc")
    if not doc_id:
        return None
    raw_versions = front.get("gdoc_versions") or ()
    versions = tuple(
        {k: _str(v) if isinstance(v, (datetime.date, datetime.datetime)) else v for k, v in ver.items()}
        for ver in raw_versions
    )
    return Pairing(doc_id=str(doc_id), synced=_str(front.get("gdoc_synced")), versions=versions)


def write_pairing(md_path: Path, pairing: Pairing) -> None:
    """Rewrite only the three gdoc keys, leaving every other key and the body alone."""
    front, body = _split(md_path.read_text())
    updated = dict(front)
    updated["gdoc"] = pairing.doc_id
    if pairing.synced:
        updated["gdoc_synced"] = pairing.synced
    if pairing.versions:
        updated["gdoc_versions"] = [dict(version) for version in pairing.versions]
    rendered = yaml.safe_dump(updated, sort_keys=False, allow_unicode=True).rstrip("\n")
    md_path.write_text(f"{_FENCE}\n{rendered}\n{_FENCE}\n\n{body}")


def add_version(pairing: Pairing, doc_id: str, created: str) -> Pairing:
    """Return a new Pairing with one more version. Never mutates the input."""
    return replace(pairing, versions=pairing.versions + ({"id": doc_id, "created": created},))


def find_by_doc_id(root: Path, doc_id: str) -> Path | None:
    """Search a tree for the markdown file already paired to this document."""
    for candidate in sorted(root.rglob("*.md")):
        try:
            pairing = read_pairing(candidate)
        except Exception:
            continue  # a malformed file must not stop the search
        if pairing and pairing.doc_id == doc_id:
            return candidate
    return None
