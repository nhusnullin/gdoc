"""Pending suggestions, read out of the Docs API.

Drive's markdown export renders a document as if its pending suggestions did not
exist. So a document Nail reviewed entirely in suggesting mode exports identical
to its baseline, and the diff sees nothing at all. That is the silent failure
this module exists to remove.

`documents.get` with suggestionsViewMode=SUGGESTIONS_INLINE marks each text run
with suggestedInsertionIds and suggestedDeletionIds. Reading them is enough:
gdoc never accepts a suggestion in the document, it reads it as an intention and
carries it into the markdown.

The API returns suggestion ids, not the person who made them. There is no author
here to report, and inventing one would be worse than saying nothing.
"""

from dataclasses import dataclass

_HEADING_STYLES = ("HEADING_", "TITLE", "SUBTITLE")


@dataclass(frozen=True)
class Suggestion:
    kind: str            # insertion or deletion
    text: str
    heading: str         # the nearest heading above it, or "" before the first
    suggestion_id: str


def _is_heading(paragraph: dict) -> bool:
    style = (paragraph.get("paragraphStyle") or {}).get("namedStyleType") or ""
    return style.startswith(_HEADING_STYLES)


def _paragraph_text(paragraph: dict) -> str:
    parts = [
        (element.get("textRun") or {}).get("content") or ""
        for element in paragraph.get("elements") or ()
    ]
    return "".join(parts).strip()


def _runs(paragraph: dict):
    """Each text run as (kind, suggestion_id, text), skipping the clean ones."""
    for element in paragraph.get("elements") or ():
        run = element.get("textRun")
        if not run:
            continue
        text = run.get("content") or ""
        for key, kind in (
            ("suggestedInsertionIds", "insertion"),
            ("suggestedDeletionIds", "deletion"),
        ):
            ids = run.get(key) or ()
            if ids:
                yield kind, ids[0], text


def _walk(content, state):
    """Collect suggestions in reading order, tracking the heading above them.

    Tables hold their own content, and a suggested fee in a table is exactly the
    kind of edit that must not be missed, so the walk goes into them.
    """
    for element in content or ():
        paragraph = element.get("paragraph")
        if paragraph:
            if _is_heading(paragraph):
                state["heading"] = _paragraph_text(paragraph)
            for kind, suggestion_id, text in _runs(paragraph):
                _add(state, kind, suggestion_id, text)
            continue
        table = element.get("table")
        if table:
            for row in table.get("tableRows") or ():
                for cell in row.get("tableCells") or ():
                    _walk(cell.get("content"), state)
            continue
        section = element.get("tableOfContents")
        if section:
            _walk(section.get("content"), state)


def _add(state, kind, suggestion_id, text):
    """Join runs that belong to one suggestion.

    Docs splits a typed sentence at every formatting boundary, so one edit
    arrives as several runs carrying the same id.
    """
    found = state["found"]
    if found and found[-1][0] == kind and found[-1][1] == suggestion_id:
        found[-1][3] += text
        return
    found.append([kind, suggestion_id, state["heading"], text])


def parse_suggestions(document: dict) -> tuple[Suggestion, ...]:
    """Every pending suggestion in the document, in reading order."""
    state = {"heading": "", "found": []}
    _walk((document.get("body") or {}).get("content"), state)
    return tuple(
        Suggestion(kind=kind, text=text.strip(), heading=heading, suggestion_id=suggestion_id)
        for kind, suggestion_id, heading, text in state["found"]
        if text.strip()
    )
