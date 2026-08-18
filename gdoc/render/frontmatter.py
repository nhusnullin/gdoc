"""Parse and validate the YAML front matter of an Altery document's Markdown file.

Only `title` is required. Anything can go in the house template: a policy, a
brief, a report, a committee submission, a set of notes. `doc_type` is free text
and may be left out entirely.

Classification is the one value still validated, because it shades a fixed row
in the template, so an unknown value would silently shade nothing.
"""

import datetime
import re
from pathlib import Path

import yaml

CLASSIFICATIONS = {
    "confidential": "Confidential (C)",
    "restricted": "Restricted (R)",
    "internal": "Internal (I)",
    "public": "Public (P)",
}

REQUIRED = ("title",)

# Named in the error below so a reader can fix the note from the message alone.
# Every one of these is read in parse(); adding a field there means adding it
# here, and test_render_frontmatter.py holds the four that matter to the message.
OPTIONAL = (
    "alt_title", "doc_type", "version", "date", "owner", "last_approval",
    "review_frequency", "board_ratification", "distribution", "classification",
    "heading_numbering", "revisions",
)

NO_FRONT_MATTER = (
    "no YAML front matter found. The file must start with a '---' line, the "
    "metadata block, then a closing '---' line. "
    f"Required: {list(REQUIRED)}. "
    f"Optional: {list(OPTIONAL)}. "
    "The shortest note that publishes is three lines: '---', 'title: Some Title', '---'."
)

DEFAULT_VERSION = "1.0"

REVISION_FIELDS = ("version", "date", "author", "approved_by",
                   "approval_date", "section", "change")

FRONT_MATTER_RE = re.compile(r"\A---\s*\n(.*?)\n---\s*\n", re.DOTALL)

DATE_PREFIX_RE = re.compile(r"^\d{4}-\d{2}-\d{2}-")
H1_RE = re.compile(r"^#\s+(.+?)\s*#*\s*$")
FENCE_RE = re.compile(r"^\s*(```|~~~)")


class FrontMatterError(ValueError):
    """Raised when the front matter is missing, malformed or incomplete."""


class MissingTitle(FrontMatterError):
    """No title, so the cover and running head would be blank.

    Carries a candidate rather than using it. The tool does not invent a cover
    title silently; the skill proposes this one and writes it into the note once
    Nail agrees, so the decision is made once and then reused.
    """

    def __init__(self, candidate, source):
        self.candidate = candidate
        self.source = source
        super().__init__(
            "no title in front matter, so the cover and running head would be blank"
        )


def title_candidate(body_markdown, source_name):
    """Return (candidate, 'h1'|'filename'). Deterministic, so it can be tested."""
    in_fence = False
    for line in body_markdown.splitlines():
        if FENCE_RE.match(line):
            in_fence = not in_fence
            continue
        if in_fence:
            continue
        match = H1_RE.match(line)
        if match:
            return match.group(1).strip(), "h1"
    stem = Path(source_name).stem if source_name else ""
    words = DATE_PREFIX_RE.sub("", stem).replace("-", " ").replace("_", " ").strip()
    return (words[:1].upper() + words[1:]) if words else "Untitled", "filename"


def _as_text(value):
    """Render a YAML scalar as the string a reader expects.

    PyYAML turns an unquoted 2026-07-15 into a date object and 1.0 into a
    float, which would print as "1.0" today and "2026-07-15" in ISO form.
    Dates become UK long form, everything else is passed through as text.
    """
    if value is None:
        return ""
    if isinstance(value, datetime.date):
        return f"{value.day} {value.strftime('%B %Y')}"
    return str(value)


def split(markdown_text):
    """Return (front_matter_dict, body_markdown)."""
    match = FRONT_MATTER_RE.match(markdown_text)
    if not match:
        raise FrontMatterError(NO_FRONT_MATTER)
    try:
        data = yaml.safe_load(match.group(1)) or {}
    except yaml.YAMLError as exc:
        raise FrontMatterError(f"the front matter is not valid YAML: {exc}") from exc
    if not isinstance(data, dict):
        raise FrontMatterError("the front matter must be a block of key: value pairs")
    return data, markdown_text[match.end():]


def _this_month():
    return datetime.date.today().strftime("%B %Y")


def _validate_revisions(raw):
    if raw is None:
        return []
    if not isinstance(raw, list):
        raise FrontMatterError("'revisions' must be a list of entries")
    revisions = []
    for index, entry in enumerate(raw, start=1):
        if not isinstance(entry, dict):
            raise FrontMatterError(f"revision {index} must be a block of key: value pairs")
        unknown = set(entry) - set(REVISION_FIELDS)
        if unknown:
            raise FrontMatterError(
                f"revision {index} has unknown field(s) {sorted(unknown)}. "
                f"Allowed: {list(REVISION_FIELDS)}"
            )
        revisions.append({field: _as_text(entry.get(field)) for field in REVISION_FIELDS})
    return revisions


def parse(markdown_text, *, title_override=None, source_name=None):
    """Validate the front matter and return (meta, body_markdown).

    `meta` is a plain dict of strings, ready for the template filler.

    source_name is the file name the text came from. It is only used to suggest
    a title when there is none, so a caller with nothing to offer can leave it
    out and still get the suggestion drawn from the first heading.
    """
    data, body = split(markdown_text)
    if title_override:
        data = {**data, "title": title_override}

    missing = [field for field in REQUIRED if not data.get(field)]
    if "title" in missing:
        raise MissingTitle(*title_candidate(body, source_name or ""))
    if missing:
        raise FrontMatterError(
            f"required front matter field(s) missing: {missing}. "
            f"Every document needs at least {list(REQUIRED)}."
        )

    # Free text, and optional. Whatever the author writes reaches the cover as
    # written, so "PRD" or "Board Submission" is not mangled into "Prd".
    doc_type = _as_text(data.get("doc_type")).strip()

    classification_key = _as_text(data.get("classification", "Internal")).strip().lower()
    classification_key = classification_key.split(" (")[0]
    if classification_key not in CLASSIFICATIONS:
        raise FrontMatterError(
            f"classification {data.get('classification')!r} is not recognised. "
            f"Use one of {sorted(CLASSIFICATIONS)}."
        )

    numbering = _as_text(data.get("heading_numbering", "auto")).strip().lower()
    if numbering not in ("auto", "none"):
        raise FrontMatterError("heading_numbering must be 'auto' or 'none'")

    title = _as_text(data["title"]).strip()
    meta = {
        "title": title,
        "alt_title": _as_text(data.get("alt_title")).strip(),
        "doc_type": doc_type,
        "version": _as_text(data.get("version")).strip() or DEFAULT_VERSION,
        "date": _as_text(data.get("date")).strip() or _this_month(),
        "owner": _as_text(data.get("owner")),
        "last_approval": _as_text(data.get("last_approval")),
        "review_frequency": _as_text(data.get("review_frequency", "Annually")),
        "board_ratification": _as_text(data.get("board_ratification")),
        "distribution": _as_text(data.get("distribution")),
        "classification": CLASSIFICATIONS[classification_key],
        "heading_numbering": numbering,
        "revisions": _validate_revisions(data.get("revisions")),
    }
    meta["cover_title"] = _cover_title(title, doc_type)
    meta["cover_alt_title"] = (
        _cover_title(meta["alt_title"], doc_type) if meta["alt_title"] else ""
    )
    meta["running_head"] = f"Altery - {meta['cover_title']}"
    return meta, body


def _cover_title(title, doc_type):
    """'Third Party Risk' + 'Policy' -> 'Third Party Risk Policy'.

    A title that already names its own type is left alone, so nobody ends up
    with a cover reading "Third Party Risk Policy Policy". With no doc_type the
    title stands on its own.
    """
    if not title:
        return ""
    if not doc_type or title.lower().endswith(doc_type.lower()):
        return title
    return f"{title} {doc_type}"
