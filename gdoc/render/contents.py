"""Write the contents list ourselves, instead of leaving it to a field.

A .docx contents list is a Word field. It carries a cached result, the text some
application computed the last time it laid the document out. **Google Docs never
refreshes that cached result on import.** Measured: two documents left untouched
kept the master template's placeholder contents list for over thirteen minutes,
and only changed when a human clicked "Update table of contents".

The master's cached result lists the template's own placeholder sections, with
page numbers running past the end of the document. So a published document keeps
a contents page describing a different document, indefinitely. The body is
correct. Only the contents page lies, which makes it the worst kind of defect.

That cached result is the only thing headless LibreOffice was ever needed for. It
is not needed for the cover, the control tables, the colours or the layout, all of
which come from copying the master and are byte-identical without it.

So this module replaces the field's **cached result**, and keeps the field.

Keeping it matters. Delete the field and Google imports plain text: the published
document then has no table of contents at all, only text shaped like one. There is
no update button, and a heading somebody adds later never appears. Keeping the
field means the document still holds a real contents list that a reader can
refresh, while the cached result we write is already correct so nobody has to.

The entries are written the way a well-behaved application writes them:

- Styled `TOC1..TOC3`, declaring no font and no weight of their own, so they
  inherit the document default, Calibri. Left to its own defaults Google renders a
  contents page in Arial with bold level-one entries, neither of which is house
  style.
- Wrapped in a hyperlink to a bookmark on the heading, so entries are clickable.
- Carrying the `IndexLink` character style, which is deliberately empty: it
  overrides Word's `Hyperlink` style so entries do not come out blue and
  underlined.

Page numbers come from the caller. `gdoc generate` gets them by uploading once and
reading the pagination back out of Google's own PDF export, which makes Google the
layout engine. A caller with no pagination to offer, such as an offline
`gdoc build`, passes none and gets blank page numbers, because a blank is honest
and a wrong number is not.
"""

import re
import zipfile
from dataclasses import dataclass
from pathlib import Path
from typing import Mapping

MAX_LEVEL = 3
# Bookmark names must be unique and free of spaces. Prefixed so ours are
# recognisable and so a rewrite never collides with the template's own.
BOOKMARK_PREFIX = "_gdoc_toc"

# Explicit spacing on the entry styles, so a refresh does not visibly move them.
#
# Google imposes its own paragraph spacing when a reader refreshes the contents
# list, and only the font is inherited. Measured on a published document: entries
# sit at a 16 to 17pt pitch after a refresh. The template's own default,
# after=120 before=60 line=276, computes to about 21.6pt, so leaving the styles to
# inherit makes the contents list visibly tighten the first time anyone refreshes
# it. Declaring the tighter value up front means the page looks the same before and
# after, which is the point.
ENTRY_SPACING = '<w:spacing w:before="0" w:after="20" w:line="276" w:lineRule="auto"/>'
# Google leaves 3pt above the first entry only, not above every one. Word and
# Google both add space-before to space-after rather than collapsing them, so
# putting this on the style would widen every gap instead.
FIRST_ENTRY_SPACING = '<w:spacing w:before="60" w:after="20" w:line="276" w:lineRule="auto"/>'

# The right tab is the text edge, which is what puts the page number at the margin.
RIGHT_TAB = 9864
# One nesting step each, in twips. 360 and 720 are what Google's own refresh uses,
# measured off a published document: level two lands 18pt in, level three 36pt.
# The house template used 283 and 567, which is close but visibly different, and a
# refresh would move the entries. Matching Google keeps the page still.
LEVEL_INDENTS = (0, 360, 720)


def _level_style(level: int, indent: int) -> str:
    return (
        f'<w:style w:type="paragraph" w:styleId="TOC{level}">'
        f'<w:name w:val="toc {level}"/><w:basedOn w:val="Index"/><w:pPr><w:tabs>'
        '<w:tab w:val="clear" w:pos="720"/>'
        f'<w:tab w:val="right" w:pos="{RIGHT_TAB}" w:leader="dot"/></w:tabs>'
        f'{ENTRY_SPACING}<w:ind w:hanging="0" w:left="{indent}"/>'
        "</w:pPr><w:rPr/></w:style>"
    )

# Measured off a LibreOffice-produced document, then frozen here. The right tab at
# 9864 twips is the text edge, which is what puts the page number at the margin.
# Each level indents by 283 twips, one for each nesting step.
TOC_STYLES = {
    "Index": (
        '<w:style w:type="paragraph" w:styleId="Index"><w:name w:val="Index"/>'
        '<w:basedOn w:val="Normal"/><w:qFormat/>'
        "<w:pPr><w:suppressLineNumbers/></w:pPr>"
        '<w:rPr><w:rFonts w:cs="Arial Unicode MS"/></w:rPr></w:style>'
    ),
    "TOC1": _level_style(1, LEVEL_INDENTS[0]),
    "TOC2": _level_style(2, LEVEL_INDENTS[1]),
    "TOC3": _level_style(3, LEVEL_INDENTS[2]),
    # Empty on purpose. It overrides Word's Hyperlink style, which would otherwise
    # render every entry blue and underlined.
    "IndexLink": (
        '<w:style w:type="character" w:styleId="IndexLink">'
        '<w:name w:val="Index Link"/><w:qFormat/><w:rPr/></w:style>'
    ),
}

_PARAGRAPH = re.compile(r"<w:p\b.*?</w:p>", re.S)
# Strict: <w:t> or <w:t attr=...>, never <w:instrText>, which a looser pattern
# matches by accident and which would drop a field instruction into the text.
_TEXT = re.compile(r"<w:t(?:\s[^>]*)?>([^<]*)</w:t>")
_HEADING = re.compile(r'w:pStyle w:val="Heading(\d)"')
_BOOKMARK_ID = re.compile(r'<w:bookmarkStart w:id="(\d+)"')


class ContentsError(RuntimeError):
    """The contents list could not be written, and guessing would be worse."""


@dataclass(frozen=True)
class ContentsResult:
    entries: tuple            # (level, text) in document order
    replaced_paragraphs: int  # how many field paragraphs were removed


def escape(text: str) -> str:
    """XML-escape a heading. Ampersands first, or the escapes get escaped."""
    return (
        text.replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace('"', "&quot;")
    )


def headings(document_xml: str) -> tuple:
    """Every heading in document order, as (level, text)."""
    found = []
    for paragraph in _PARAGRAPH.findall(document_xml):
        match = _HEADING.search(paragraph)
        if not match:
            continue
        text = "".join(_TEXT.findall(paragraph)).strip()
        if text:
            found.append((int(match.group(1)), text))
    return tuple(found)


@dataclass(frozen=True)
class Field:
    """Where the contents field is, and the two runs that must survive a rewrite."""

    start: int        # offset of the first paragraph the field spans
    end: int          # offset just past the last paragraph it spans
    begin_run: str    # the run carrying begin, the instruction and separate
    end_run: str      # the run carrying the end marker
    paragraphs: int


def _run_carrying(paragraph_xml: str, marker: str) -> str:
    for run in re.findall(r"<w:r\b.*?</w:r>", paragraph_xml, re.S):
        if marker in run:
            return run
    raise ContentsError(f"the contents field has no run carrying {marker}")


def field(document_xml: str) -> Field:
    """Locate the contents field.

    The master's field is malformed: the begin marker, the instruction and the
    separator all sit in a single run, where the format wants one run each. So the
    field cannot be found by matching runs. Find the paragraph carrying the
    instruction instead, then run forward to the paragraph carrying the end
    marker. Everything between the two is the cached result.

    Google accepts the malformed run and still builds a real contents list from it,
    so it is carried across untouched rather than repaired. Repairing it is a
    change with no observed benefit and an unmeasured risk.
    """
    paragraphs = list(_PARAGRAPH.finditer(document_xml))
    first = next((i for i, m in enumerate(paragraphs) if "<w:instrText" in m.group(0)), None)
    if first is None:
        raise ContentsError(
            "no contents field found in this document, so there is nothing to "
            "replace. Was the contents list already written?"
        )
    last = next(
        (i for i, m in enumerate(paragraphs)
         if i >= first and 'w:fldCharType="end"' in m.group(0)),
        None,
    )
    if last is None:
        raise ContentsError("the contents field has no end marker, refusing to guess where it stops")
    return Field(
        start=paragraphs[first].start(),
        end=paragraphs[last].end(),
        begin_run=_run_carrying(paragraphs[first].group(0), 'w:fldCharType="begin"'),
        end_run=_run_carrying(paragraphs[last].group(0), 'w:fldCharType="end"'),
        paragraphs=last - first + 1,
    )


def _bookmarked(paragraph_xml: str, name: str, bookmark_id: int) -> str:
    """Wrap a heading paragraph in a bookmark, so an entry can link to it."""
    start = f'<w:bookmarkStart w:id="{bookmark_id}" w:name="{name}"/>'
    end = f'<w:bookmarkEnd w:id="{bookmark_id}"/>'
    if "</w:pPr>" in paragraph_xml:
        head, _, tail = paragraph_xml.partition("</w:pPr>")
        body = f"{head}</w:pPr>{start}{tail}"
    else:
        after_open = paragraph_xml.index(">") + 1
        body = paragraph_xml[:after_open] + start + paragraph_xml[after_open:]
    return body[: body.rindex("</w:p>")] + end + "</w:p>"


def _with_bookmarks(document_xml: str, count: int) -> tuple:
    """Bookmark every heading. Returns the document and the anchor names."""
    existing = [int(i) for i in _BOOKMARK_ID.findall(document_xml)]
    next_id = max(existing) + 1 if existing else 1

    edits, anchors = [], []
    for match in _PARAGRAPH.finditer(document_xml):
        paragraph = match.group(0)
        if not _HEADING.search(paragraph):
            continue
        if not "".join(_TEXT.findall(paragraph)).strip():
            continue
        name = f"{BOOKMARK_PREFIX}{len(anchors) + 1}"
        anchors.append(name)
        edits.append((match.start(), match.end(),
                      _bookmarked(paragraph, name, next_id + len(anchors) - 1)))

    if len(anchors) != count:
        raise ContentsError(
            f"bookmarked {len(anchors)} headings but found {count}, refusing to link them up"
        )
    # Backwards, so each splice leaves earlier offsets untouched.
    for start, end, replacement in reversed(edits):
        document_xml = document_xml[:start] + replacement + document_xml[end:]
    return document_xml, anchors


def _entry(level: int, text: str, page, anchor: str, prefix: str = "", suffix: str = "",
           first: bool = False) -> str:
    """One contents line: a styled paragraph, a tab, then the page number.

    No font and no weight are declared. That is deliberate: anything declared here
    overrides the template, and anything left undeclared inherits the document
    default. Google supplies Arial and bold only when nothing else does.

    prefix and suffix carry the field's begin and end runs on the first and last
    entries, which is what keeps the contents list a field rather than plain text.
    """
    style = f"TOC{min(level, MAX_LEVEL)}"
    spacing = FIRST_ENTRY_SPACING if first else ""
    return (
        f'<w:p><w:pPr><w:pStyle w:val="{style}"/><w:tabs>'
        f'<w:tab w:val="clear" w:pos="{RIGHT_TAB}"/>'
        f'<w:tab w:val="right" w:pos="{RIGHT_TAB - 1}" w:leader="dot"/>'
        f"</w:tabs>{spacing}<w:rPr/></w:pPr>"
        f"{prefix}"
        f'<w:hyperlink w:anchor="{anchor}">'
        '<w:r><w:rPr><w:rStyle w:val="IndexLink"/><w:webHidden/></w:rPr>'
        f'<w:t xml:space="preserve">{escape(text)}</w:t>'
        f"<w:tab/><w:t>{page}</w:t></w:r></w:hyperlink>"
        f"{suffix}</w:p>"
    )


def _with_styles(styles_xml: str) -> str:
    for style_id, xml in TOC_STYLES.items():
        if f'w:styleId="{style_id}"' not in styles_xml:
            styles_xml = styles_xml.replace("</w:styles>", xml + "</w:styles>")
    return styles_xml


def write(src: Path, dst: Path, pages: Mapping | None = None) -> ContentsResult:
    """Replace the contents field with real entries.

    pages maps heading text to a page number. A heading missing from it gets a
    blank page number rather than a guess.
    """
    src, dst = Path(src), Path(dst)
    archive = zipfile.ZipFile(src)
    document = archive.read("word/document.xml").decode("utf8")
    styles = archive.read("word/styles.xml").decode("utf8")

    found = headings(document)
    if not found:
        raise ContentsError(
            "this document has no headings, so a contents list would be empty. "
            "Refusing to write one."
        )

    document, anchors = _with_bookmarks(document, len(found))
    located = field(document)
    lookup = dict(pages or {})
    last = len(found) - 1
    entries = "".join(
        _entry(
            level,
            text,
            lookup.get(text, ""),
            anchor,
            prefix=located.begin_run if index == 0 else "",
            suffix=located.end_run if index == last else "",
            first=index == 0,
        )
        for index, ((level, text), anchor) in enumerate(zip(found, anchors))
    )
    document = document[: located.start] + entries + document[located.end :]
    styles = _with_styles(styles)
    replaced = located.paragraphs

    dst.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(dst, "w", zipfile.ZIP_DEFLATED) as out:
        for item in archive.infolist():
            if item.filename == "word/document.xml":
                data = document.encode("utf8")
            elif item.filename == "word/styles.xml":
                data = styles.encode("utf8")
            else:
                data = archive.read(item.filename)
            out.writestr(item, data)

    return ContentsResult(entries=found, replaced_paragraphs=replaced)
