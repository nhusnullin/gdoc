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

So this module deletes the field and writes ordinary paragraphs in its place:

- Nothing is left to refresh, so nobody has to click anything.
- Google has no field to restyle, so it does not apply its own defaults. Left to
  those defaults it renders the contents page in Arial with bold level-one
  entries, neither of which is the house style.
- The entries declare no font and no weight of their own, so they inherit the
  document default, Calibri, through the TOC styles below.

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
    "TOC1": (
        '<w:style w:type="paragraph" w:styleId="TOC1"><w:name w:val="toc 1"/>'
        '<w:basedOn w:val="Index"/><w:pPr><w:tabs>'
        '<w:tab w:val="clear" w:pos="720"/>'
        '<w:tab w:val="right" w:pos="9864" w:leader="dot"/></w:tabs>'
        '<w:ind w:hanging="0" w:left="0"/></w:pPr><w:rPr/></w:style>'
    ),
    "TOC2": (
        '<w:style w:type="paragraph" w:styleId="TOC2"><w:name w:val="toc 2"/>'
        '<w:basedOn w:val="Index"/><w:pPr><w:tabs>'
        '<w:tab w:val="clear" w:pos="720"/>'
        '<w:tab w:val="right" w:pos="9864" w:leader="dot"/></w:tabs>'
        '<w:ind w:hanging="0" w:left="283"/></w:pPr><w:rPr/></w:style>'
    ),
    "TOC3": (
        '<w:style w:type="paragraph" w:styleId="TOC3"><w:name w:val="toc 3"/>'
        '<w:basedOn w:val="Index"/><w:pPr><w:tabs>'
        '<w:tab w:val="clear" w:pos="720"/>'
        '<w:tab w:val="right" w:pos="9864" w:leader="dot"/></w:tabs>'
        '<w:ind w:hanging="0" w:left="567"/></w:pPr><w:rPr/></w:style>'
    ),
}

_PARAGRAPH = re.compile(r"<w:p\b.*?</w:p>", re.S)
_TEXT = re.compile(r"<w:t[^>]*>([^<]*)</w:t>")
_HEADING = re.compile(r'w:pStyle w:val="Heading(\d)"')


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


def field_span(document_xml: str) -> tuple:
    """Where the contents field lives, as (start, end, paragraph_count).

    The master's field is malformed: the begin marker, the instruction and the
    separator all sit in a single run, where the format wants one run each. So the
    field cannot be found by matching runs. Find the paragraph carrying the
    instruction instead, then run forward to the paragraph carrying the end
    marker. Everything between the two is the cached result.
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
    return paragraphs[first].start(), paragraphs[last].end(), last - first + 1


def _entry(level: int, text: str, page) -> str:
    """One contents line: a styled paragraph, a tab, then the page number.

    No font and no weight are declared. That is deliberate: anything declared here
    overrides the template, and anything left undeclared inherits the document
    default. Google supplies Arial and bold only when nothing else does.
    """
    style = f"TOC{min(level, MAX_LEVEL)}"
    return (
        f'<w:p><w:pPr><w:pStyle w:val="{style}"/><w:tabs>'
        '<w:tab w:val="clear" w:pos="9864"/>'
        '<w:tab w:val="right" w:pos="9863" w:leader="dot"/>'
        "</w:tabs><w:rPr/></w:pPr>"
        f'<w:r><w:rPr/><w:t xml:space="preserve">{escape(text)}</w:t>'
        f"<w:tab/><w:t>{page}</w:t></w:r></w:p>"
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

    start, end, replaced = field_span(document)
    lookup = dict(pages or {})
    entries = "".join(_entry(level, text, lookup.get(text, "")) for level, text in found)
    document = document[:start] + entries + document[end:]
    styles = _with_styles(styles)

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
