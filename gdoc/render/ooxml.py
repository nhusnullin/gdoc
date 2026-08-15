"""Low-level OOXML element helpers plus the formatting constants measured from
the Altery Group Framework/Policy template.

Every constant here was read out of the template's own `word/document.xml`,
not invented. That is the whole reason the generated body matches the master:
we reproduce the template's idioms rather than a tool's defaults.
"""

from lxml import etree

W = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
XML_SPACE = "{http://www.w3.org/XML/1998/namespace}space"

# --- measured from the template ---------------------------------------------
BODY_SZ = "24"          # 12pt half-points; Google Docs overrode Normal's 11pt
HEAD_COLOR = "222660"   # run-level navy the template puts on every heading
TBL_BORDER = "c9c9c9"   # table-level grid
HDR_FILL = "bdcdd2"     # header-row shading
ROW_FILL = "f3f8f9"     # body-row shading
BODY_FONT = "Calibri"

# The master draws a black border on every cell as well as the grey table grid,
# which turns a four-row table into a spreadsheet. Grey on every line lets the
# header band carry the structure instead of the gridlines.
CELL_BORDER = TBL_BORDER
NO_FILL = "ffffff"

# 3pt above and below the text in a cell. The master sets no cell margin at all,
# so descenders sit on the bottom border. Left and right stay at Word's own 108.
CELL_MARGIN_V = "60"
CELL_MARGIN_H = "108"

# 1.15 line spacing, in 240ths of a line (240 = single). House rule for body
# prose; the master uses it on its own body paragraphs. Table cells stay single.
LINE_SPACING = "276"

# Headings get the same 1.15 line so a heading that wraps to two lines is as
# readable as the prose beneath it, and so one rule covers the whole page.
HEADING_LINE_SPACING = LINE_SPACING

# 6pt beneath a heading. The master sets zero, which glues the heading to the
# first line of its section. Half the 12pt above it keeps the heading bound to
# the text it introduces rather than floating between two sections.
HEADING_SPACE_AFTER = "120"
HEADING_SPACE_BEFORE = "240"

# Weight for Heading3 to Heading6, which all share the 12pt body size because
# the template's size scale runs out at level 3.
#
# The master inverts the hierarchy here: Heading3 is regular while Heading4 and
# Heading6 are bold, so a child heading looks stronger than its parent. Emphasis
# is removed one step at a time instead, so weight falls as depth grows.
#
# Levels 5 and 6 deliberately look alike. Six heading levels in a Group document
# is a structure problem, not a formatting one, and inventing a sixth treatment
# would only hide it.
HEADING_EMPHASIS = {
    3: {"bold": True, "italic": False},
    4: {"bold": False, "italic": False},
    5: {"bold": False, "italic": True},
    6: {"bold": False, "italic": True},
}

PAGE_W = 11906
MARGIN_L = 1021
MARGIN_R = 1021
USABLE_TWIPS = PAGE_W - MARGIN_L - MARGIN_R

# abstractNum ids already defined in the template's numbering.xml
ABSTRACT_ORDERED = "3"  # decimal / lowerLetter / lowerRoman
ABSTRACT_BULLET = "2"   # filled disc / hollow circle / filled square

IND_LEFT = (720, 1440, 2160, 2880)
HANGING = 360

MAX_LIST_LEVEL = 8      # w:ilvl is 0..8


def qn(tag):
    """'w:val' -> Clark notation."""
    prefix, _, local = tag.partition(":")
    if prefix != "w":
        raise ValueError(f"only the w: namespace is handled here, got {tag!r}")
    return f"{{{W}}}{local}"


def el(tag, **attrs):
    """Create a free-standing w: element with w:-namespaced attributes."""
    e = etree.Element(f"{{{W}}}{tag}")
    for k, v in attrs.items():
        e.set(f"{{{W}}}{k}", str(v))
    return e


def sub(parent, tag, **attrs):
    """Append a w: child element with w:-namespaced attributes."""
    e = etree.SubElement(parent, f"{{{W}}}{tag}")
    for k, v in attrs.items():
        e.set(f"{{{W}}}{k}", str(v))
    return e


def set_fonts(rpr, font=BODY_FONT):
    f = sub(rpr, "rFonts")
    for attr in ("ascii", "hAnsi", "cs", "eastAsia"):
        f.set(qn(f"w:{attr}"), font)
    return f


def text_run(parent, text, *, bold=False, italic=False, mono=False,
             color=None, highlight=None, sz=BODY_SZ, font=None):
    """Append a fully-formed w:r carrying `text`.

    xml:space="preserve" is mandatory: leading and trailing spaces between
    runs are meaningful and Word silently eats them otherwise.
    """
    r = sub(parent, "r")
    rpr = sub(r, "rPr")
    if mono:
        set_fonts(rpr, "Consolas")
    elif font:
        set_fonts(rpr, font)
    if bold:
        sub(rpr, "b", val="1")
        sub(rpr, "bCs", val="1")
    if italic:
        sub(rpr, "i", val="1")
        sub(rpr, "iCs", val="1")
    if color:
        sub(rpr, "color", val=color)
    if highlight:
        sub(rpr, "highlight", val=highlight)
    if sz:
        sub(rpr, "sz", val=sz)
        sub(rpr, "szCs", val=sz)
    sub(rpr, "rtl", val="0")
    t = sub(r, "t")
    t.set(XML_SPACE, "preserve")
    t.text = text
    return r


# The order the schema requires inside w:pPr. Word rejects paragraph properties
# whose children are out of sequence, and it is easy to break by appending: a
# w:spacing added after a w:ind is already invalid. Only the tags this skill
# writes or is likely to write are listed; anything unlisted sorts to the end.
PPR_ORDER = (
    "w:pStyle", "w:keepNext", "w:keepLines", "w:pageBreakBefore",
    "w:widowControl", "w:numPr", "w:pBdr", "w:shd", "w:tabs",
    "w:suppressAutoHyphens", "w:bidi", "w:spacing", "w:ind",
    "w:contextualSpacing", "w:jc", "w:outlineLvl", "w:rPr", "w:sectPr",
)


def set_ppr_child(p_pr, tag, **attrs):
    """Replace `tag` inside `p_pr`, inserted where the schema wants it.

    Appending is what breaks: a w:spacing written after a w:ind is out of
    sequence and Word refuses the whole w:pPr. Existing copies of the tag are
    dropped first, so this is idempotent.
    """
    for existing in p_pr.findall(qn(tag)):
        p_pr.remove(existing)

    element = el(tag.split(":")[1], **attrs)
    rank = PPR_ORDER.index(tag) if tag in PPR_ORDER else len(PPR_ORDER)

    for child in p_pr:
        name = f"w:{etree.QName(child).localname}"
        child_rank = PPR_ORDER.index(name) if name in PPR_ORDER else len(PPR_ORDER)
        if child_rank > rank:
            child.addprevious(element)
            return element
    p_pr.append(element)
    return element


class Numbering:
    """Hands out a fresh w:numId per list instance.

    Two things make this non-obvious, both learned the hard way:

    1. Reuse the template's own abstractNum definitions, so bullet glyphs
       (filled disc / hollow circle / filled square) and the decimal /
       lower-letter / lower-roman sequence come from the master document
       rather than from us.
    2. A bare new w:num pointing at the same abstractNum does NOT restart
       numbering in LibreOffice: the second ordered list carries on 4., 5., 6.
       An explicit w:lvlOverride with w:startOverride on every one of the nine
       levels is what forces the restart.
    """

    def __init__(self, doc):
        self.part = doc.part.numbering_part.element
        used = [int(n.get(qn("w:numId"))) for n in self.part.findall(qn("w:num"))]
        self.next_id = max(used) + 1 if used else 1

    def new(self, ordered, start=1):
        num_id = self.next_id
        self.next_id += 1
        num = sub(self.part, "num", numId=str(num_id))
        sub(num, "abstractNumId",
            val=ABSTRACT_ORDERED if ordered else ABSTRACT_BULLET)
        for level in range(9):
            override = sub(num, "lvlOverride", ilvl=str(level))
            sub(override, "startOverride", val=str(start if level == 0 else 1))
        return num_id
