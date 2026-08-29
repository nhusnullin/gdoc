"""Build an Altery house .docx from house.yaml plus a markdown body.

Nothing here reads the master .docx. The only inputs are the config file and
the body content.
"""
import argparse, base64, re, zipfile
from pathlib import Path
from xml.sax.saxutils import escape, quoteattr
import yaml

TW = lambda pt: str(int(round(float(pt) * 20)))       # points -> twips
EMU = lambda pt: str(int(round(float(pt) * 12700)))   # points -> EMU
HALF = lambda pt: str(int(round(float(pt) * 2)))      # points -> half-points
def hexc(c):
    return (c or "#000000").lstrip("#").lower()

NS = (
 'xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" '
 'xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" '
 'xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" '
 'xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" '
 'xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture" '
 'xmlns:v="urn:schemas-microsoft-com:vml" '
 'xmlns:o="urn:schemas-microsoft-com:office:office" '
 'xmlns:w10="urn:schemas-microsoft-com:office:word" '
 'xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006" '
 'xmlns:w14="http://schemas.microsoft.com/office/word/2010/wordml"'
)
XMLDECL = '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
ALIGN = {"left": "left", "right": "right", "center": "center",
         "justify": "both", "start": "left", "end": "right"}


# ------------------------------------------------------------------ runs ----
def rpr(size_pt=None, bold=False, italic=False, color=None, font=None,
        highlight=None, underline=False):
    p = []
    if font:
        p.append(f'<w:rFonts w:ascii={quoteattr(font)} w:hAnsi={quoteattr(font)} '
                 f'w:cs={quoteattr(font)} w:eastAsia={quoteattr(font)}/>')
    if bold:
        p.append('<w:b w:val="1"/><w:bCs w:val="1"/>')
    if italic:
        p.append('<w:i w:val="1"/><w:iCs w:val="1"/>')
    if underline:
        p.append('<w:u w:val="single"/>')
    if color:
        p.append(f'<w:color w:val="{hexc(color)}"/>')
    if size_pt:
        p.append(f'<w:sz w:val="{HALF(size_pt)}"/><w:szCs w:val="{HALF(size_pt)}"/>')
    if highlight:
        p.append(f'<w:highlight w:val="{highlight}"/>')
    return "<w:rPr>" + "".join(p) + "</w:rPr>" if p else ""


def run(text="", **kw):
    return f'<w:r>{rpr(**kw)}<w:t xml:space="preserve">{escape(text)}</w:t></w:r>'


def tabs_run(n, **kw):
    return f'<w:r>{rpr(**kw)}' + ("<w:tab/>" * n) + "</w:r>"


def page_field_run(**kw):
    r = rpr(**kw)
    return (f'<w:r>{r}<w:fldChar w:fldCharType="begin"/>'
            f'<w:instrText xml:space="preserve">PAGE</w:instrText>'
            f'<w:fldChar w:fldCharType="separate"/><w:t>1</w:t>'
            f'<w:fldChar w:fldCharType="end"/></w:r>')


def ppr(style=None, align=None, before=None, after=None, line=None,
        indent_start=None, hanging=None, first_line=None,
        keep_next=False, keep_lines=False,
        page_break=False, numid=None, ilvl=0, rpr_xml="", borders=None):
    p = []
    if style:
        p.append(f'<w:pStyle w:val="{style}"/>')
    if keep_next:
        p.append('<w:keepNext w:val="1"/>')
    if keep_lines:
        p.append('<w:keepLines w:val="1"/>')
    if page_break:
        p.append('<w:pageBreakBefore w:val="1"/>')
    if numid is not None:
        p.append(f'<w:numPr><w:ilvl w:val="{ilvl}"/><w:numId w:val="{numid}"/></w:numPr>')
    if borders:
        p.append(borders)
    sp = []
    if before is not None:
        sp.append(f'w:before="{TW(before)}"')
    if after is not None:
        sp.append(f'w:after="{TW(after)}"')
    if line is not None:
        sp.append(f'w:line="{int(round(line*240))}" w:lineRule="auto"')
    if sp:
        p.append("<w:spacing " + " ".join(sp) + "/>")
    if indent_start is not None or hanging is not None or first_line is not None:
        bits = []
        if indent_start is not None:
            bits.append(f'w:left="{TW(indent_start)}"')
        if hanging:
            bits.append(f'w:hanging="{TW(hanging)}"')
        else:
            bits.append(f'w:firstLine="{TW(first_line or 0)}"')
        p.append("<w:ind " + " ".join(bits) + "/>")
    if align:
        p.append(f'<w:jc w:val="{ALIGN[align]}"/>')
    if rpr_xml:
        p.append(rpr_xml)
    return "<w:pPr>" + "".join(p) + "</w:pPr>" if p else ""


def para(runs_xml="", **kw):
    return f"<w:p>{ppr(**kw)}{runs_xml}</w:p>"


def blank(spec):  # noqa: D401
    """An empty paragraph that still carries a run size, because an empty
    paragraph's height is what the cover's vertical rhythm is made of."""
    r = rpr(size_pt=spec.get("size_pt"))
    return para(f"<w:r>{r}</w:r>" if r else "<w:r/>",
                align=spec.get("align"),
                before=spec.get("space_before_pt"),
                after=spec.get("space_after_pt"),
                line=spec.get("line_spacing"),
                rpr_xml=r)


# --------------------------------------------------------------- styles ----
def styles_xml(cfg):
    d = cfg["defaults"]
    st = cfg["styles"]
    out = [XMLDECL, f"<w:styles {NS}>"]
    out.append(
        "<w:docDefaults><w:rPrDefault><w:rPr>"
        f'<w:rFonts w:ascii="{d["font"]}" w:hAnsi="{d["font"]}" '
        f'w:cs="{d["font"]}" w:eastAsia="{d["font"]}"/>'
        f'<w:sz w:val="{HALF(d["size_pt"])}"/><w:szCs w:val="{HALF(d["size_pt"])}"/>'
        '<w:lang w:val="en-GB"/></w:rPr></w:rPrDefault>'
        "<w:pPrDefault><w:pPr>"
        f'<w:spacing w:before="{TW(d["space_before_pt"])}" '
        f'w:after="{TW(d["space_after_pt"])}" '
        f'w:line="{int(round(d["line_spacing"]*240))}" w:lineRule="auto"/>'
        "</w:pPr></w:pPrDefault></w:docDefaults>")

    def one(sid, name, s, default=False, based=None):
        r = rpr(size_pt=s.get("size_pt"), bold=s.get("bold", False),
                italic=s.get("italic", False), color=s.get("color"),
                font=s.get("font"))
        p = ppr(align=s.get("align"), before=s.get("space_before_pt"),
                after=s.get("space_after_pt"), line=s.get("line_spacing"),
                indent_start=s.get("indent_start_pt"),
                keep_next=s.get("keep_with_next", False),
                keep_lines=s.get("keep_lines_together", False))
        p = p.replace("<w:pPr>", "<w:pPr>", 1) if p else "<w:pPr/>"
        basedon = f'<w:basedOn w:val="{based}"/>' if based else ""
        dflt = ' w:default="1"' if default else ""
        return (f'<w:style w:type="paragraph"{dflt} w:styleId="{sid}">'
                f'<w:name w:val={quoteattr(name)}/>{basedon}'
                f'<w:qFormat/>{p}{r}</w:style>')

    out.append(one("Normal", "Normal", st["normal"], default=True))
    for i in range(1, 7):
        out.append(one(f"Heading{i}", f"heading {i}", st[f"heading_{i}"],
                       based="Normal"))
    out.append(one("Title", "Title", st["title"], based="Normal"))
    out.append(one("Subtitle", "Subtitle", st["subtitle"], based="Normal"))
    out.append('<w:style w:type="table" w:default="1" w:styleId="TableNormal">'
               '<w:name w:val="Table Normal"/><w:tblPr/></w:style>')
    out.append("</w:styles>")
    return "".join(out)


def numbering_xml(cfg):
    b = cfg["body"]["bullet"]
    n = cfg["body"]["numbered"]
    lv = []
    for i, g in enumerate(b["glyphs"]):
        lv.append(
            f'<w:lvl w:ilvl="{i}"><w:start w:val="1"/><w:numFmt w:val="bullet"/>'
            f'<w:lvlText w:val={quoteattr(g)}/><w:lvlJc w:val="left"/><w:pPr>'
            f'<w:ind w:left="{TW(b["indent_start_pt"]*(i+1))}" '
            f'w:hanging="{TW(b["hanging_pt"])}"/></w:pPr>'
            '<w:rPr><w:rFonts w:ascii="Noto Sans Symbols" w:hAnsi="Noto Sans Symbols"/>'
            "</w:rPr></w:lvl>")
    for i in range(len(b["glyphs"]), 9):
        lv.append(f'<w:lvl w:ilvl="{i}"><w:numFmt w:val="bullet"/>'
                  f'<w:lvlText w:val="●"/><w:lvlJc w:val="left"/></w:lvl>')
    nlv = [f'<w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="decimal"/>'
           f'<w:lvlText w:val={quoteattr(n["format"])}/><w:lvlJc w:val="left"/>'
           f'<w:pPr><w:ind w:left="{TW(n["indent_start_pt"])}" '
           f'w:hanging="{TW(n["hanging_pt"])}"/></w:pPr></w:lvl>']
    for i in range(1, 9):
        nlv.append(f'<w:lvl w:ilvl="{i}"><w:numFmt w:val="decimal"/>'
                   f'<w:lvlText w:val="%{i+1}."/><w:lvlJc w:val="left"/></w:lvl>')
    return (XMLDECL + f"<w:numbering {NS}>"
            + '<w:abstractNum w:abstractNumId="1">' + "".join(lv) + "</w:abstractNum>"
            + '<w:abstractNum w:abstractNumId="2">' + "".join(nlv) + "</w:abstractNum>"
            + '<w:num w:numId="1"><w:abstractNumId w:val="1"/></w:num>'
            + '<w:num w:numId="2"><w:abstractNumId w:val="2"/></w:num>'
            + "</w:numbering>")


def settings_xml():
    return (XMLDECL + f"<w:settings {NS}><w:updateFields w:val=\"true\"/>"
            "<w:evenAndOddHeaders w:val=\"false\"/></w:settings>")


# ------------------------------------------------------- header / footer ----
def hf_runs(spec, defaults):
    out = []
    for r in spec.get("runs", []):
        kw = dict(size_pt=r.get("size_pt", defaults.get("size_pt")),
                  color=r.get("color", defaults.get("color")),
                  bold=r.get("bold", False))
        if r.get("page_number"):
            out.append(page_field_run(**kw))
        elif r.get("tabs"):
            out.append(tabs_run(r["tabs"], **kw))
        elif r.get("tab"):
            out.append("<w:r><w:tab/></w:r>")
        else:
            out.append(run(r.get("text", ""), **kw))
    return "".join(out)


def logo_drawing(cfg, rid):
    lg = cfg["logo"]
    a = lg["anchor"]
    d = EMU(a["wrap_dist_pt"])
    return (
        "<w:r><w:drawing>"
        f'<wp:anchor distT="{d}" distB="{d}" distL="{d}" distR="{d}" simplePos="0" '
        'relativeHeight="0" behindDoc="0" locked="0" layoutInCell="1" '
        'allowOverlap="1"><wp:simplePos x="0" y="0"/>'
        f'<wp:positionH relativeFrom="{a["relative_h"]}">'
        f'<wp:posOffset>{EMU(a["offset_x_pt"])}</wp:posOffset></wp:positionH>'
        f'<wp:positionV relativeFrom="{a["relative_v"]}">'
        f'<wp:posOffset>{EMU(a["offset_y_pt"])}</wp:posOffset></wp:positionV>'
        f'<wp:extent cx="{EMU(lg["width_pt"])}" cy="{EMU(lg["height_pt"])}"/>'
        '<wp:effectExtent l="0" t="0" r="0" b="0"/>'
        f'<wp:wrapSquare wrapText="bothSides" distT="{d}" distB="{d}" '
        f'distL="{d}" distR="{d}"/>'
        '<wp:docPr id="18" name="logo.png"/><a:graphic><a:graphicData '
        'uri="http://schemas.openxmlformats.org/drawingml/2006/picture">'
        '<pic:pic><pic:nvPicPr><pic:cNvPr id="0" name="logo.png"/>'
        '<pic:cNvPicPr preferRelativeResize="0"/></pic:nvPicPr>'
        f'<pic:blipFill><a:blip r:embed="{rid}"/>'
        '<a:srcRect b="0" l="0" r="0" t="0"/><a:stretch><a:fillRect/></a:stretch>'
        '</pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/>'
        f'<a:ext cx="{EMU(lg["width_pt"])}" cy="{EMU(lg["height_pt"])}"/></a:xfrm>'
        '<a:prstGeom prst="rect"/><a:ln/></pic:spPr></pic:pic></a:graphicData>'
        "</a:graphic></wp:anchor></w:drawing></w:r>")


def header_xml(cfg, which, rid=None):
    h = cfg["header"][which]
    ps = []
    for spec in h["paragraphs"]:
        if spec.get("rule"):
            rl = spec["rule"]
            r = ('<w:r><w:pict>'
                 f'<v:rect style="width:0.0pt;height:{rl["height_pt"]}pt" o:hr="t" '
                 f'o:hrstd="t" o:hralign="center" fillcolor="{rl["color"]}" '
                 'stroked="f"/></w:pict></w:r>')
        else:
            r = hf_runs(spec, {})
            if spec.get("logo"):
                r += logo_drawing(cfg, rid)
        ps.append(para(r, align=spec.get("align", h.get("align")),
                       before=spec.get("space_before_pt"),
                       after=spec.get("space_after_pt"),
                       line=spec.get("line_spacing", h.get("line_spacing"))))
    return XMLDECL + f"<w:hdr {NS}>" + "".join(ps) + "</w:hdr>"


def footer_xml(cfg, which):
    f = cfg["footer"][which]
    ps = []
    for spec in f["paragraphs"]:
        ps.append(para(hf_runs(spec, {}),
                       before=spec.get("space_before_pt"),
                       after=spec.get("space_after_pt"),
                       line=spec.get("line_spacing", f.get("line_spacing")),
                       indent_start=f.get("indent_start_pt")))
    return XMLDECL + f"<w:ftr {NS}>" + "".join(ps) + "</w:ftr>"


# --------------------------------------------------------------- tables ----
def table_xml(cfg, spec):
    tt = cfg["table_text"]
    bw = int(round(spec["border"]["width_pt"] * 8))     # points -> eighths
    bc = hexc(spec["border"]["color"])
    def edge(t):
        return f'<w:{t} w:val="single" w:sz="{bw}" w:space="0" w:color="{bc}"/>'
    borders = "".join(edge(t) for t in
                      ("top", "left", "bottom", "right", "insideH", "insideV"))
    total = TW(sum(spec["columns_pt"]))
    grid = "".join(f'<w:gridCol w:w="{TW(w)}"/>' for w in spec["columns_pt"])
    rows = []
    for r in spec["rows"]:
        cells = []
        for ci, c in enumerate(r["cells"]):
            pt_, pl_, pb_, pr_ = c.get("pad_tlbr_pt") or ([c.get("pad_pt", 5)] * 4)
            shd = (f'<w:shd w:val="clear" w:color="auto" w:fill="{hexc(c["fill"])}"/>'
                   if c.get("fill") else "")
            tcb = "".join(edge(t) for t in ("top", "left", "bottom", "right"))
            body = "".join(
                para("".join(
                        run(x["text"], size_pt=x.get("size_pt", tt["default_size_pt"]),
                            font=tt["font"], bold=x.get("bold", False),
                            italic=x.get("italic", False), color=x.get("color"),
                            highlight=("yellow" if x.get("highlight") else None))
                        for x in p_["runs"])
                     or f'<w:r>{rpr(size_pt=tt["default_size_pt"], font=tt["font"])}</w:r>',
                     align=p_.get("align"), line=p_.get("line_spacing"),
                     before=p_.get("space_before_pt"),
                     after=p_.get("space_after_pt"),
                     indent_start=p_.get("indent_start_pt"),
                     # Docs measures a first-line indent from the margin;
                     # OOXML measures it from the paragraph indent
                     first_line=(p_["indent_first_line_pt"]
                                 - p_.get("indent_start_pt", 0)
                                 if "indent_first_line_pt" in p_ else None),
                     keep_lines=p_.get("keep_lines_together", False),
                     keep_next=p_.get("keep_with_next", False))
                for p_ in c["paragraphs"])
            cells.append(
                f'<w:tc><w:tcPr><w:tcW w:w="{TW(spec["columns_pt"][ci])}" '
                f'w:type="dxa"/><w:tcBorders>{tcb}</w:tcBorders>{shd}'
                f'<w:tcMar><w:top w:w="{TW(pt_)}" w:type="dxa"/>'
                f'<w:left w:w="{TW(pl_)}" w:type="dxa"/>'
                f'<w:bottom w:w="{TW(pb_)}" w:type="dxa"/>'
                f'<w:right w:w="{TW(pr_)}" w:type="dxa"/></w:tcMar>'
                f"</w:tcPr>{body}</w:tc>")
        rows.append(
            f'<w:tr><w:trPr><w:trHeight w:val="{TW(r["min_height_pt"])}" '
            f'w:hRule="atLeast"/></w:trPr>' + "".join(cells) + "</w:tr>")
    # Word insets a table by its first cell's left margin, so the text rather
    # than the cell edge lines up with the page margin. Google copies that on
    # import, and the house tables sit on the margin, so cancel it.
    first_pad = (spec["rows"][0]["cells"][0].get("pad_tlbr_pt")
                 or [0, spec["rows"][0]["cells"][0].get("pad_pt", 0)] * 2)[1]
    return (f'<w:tbl><w:tblPr><w:tblStyle w:val="TableNormal"/>'
            f'<w:tblW w:w="{total}" w:type="dxa"/>'
            f'<w:tblInd w:w="{TW(first_pad)}" w:type="dxa"/>'
            f'<w:tblLayout w:type="fixed"/>'
            f"<w:tblBorders>{borders}</w:tblBorders></w:tblPr>"
            f"<w:tblGrid>{grid}</w:tblGrid>" + "".join(rows) + "</w:tbl>")


# ------------------------------------------------------------------ TOC ----
def toc_xml(cfg):
    t = cfg["toc"]
    inner = para(
        f'<w:r><w:fldChar w:fldCharType="begin"/>'
        f'<w:instrText xml:space="preserve">{escape(t["instr"])}</w:instrText>'
        f'<w:fldChar w:fldCharType="separate"/></w:r>'
        f'{run("Refresh this table of contents in Google Docs.", size_pt=t["size_pt"])}'
        f'<w:r><w:fldChar w:fldCharType="end"/></w:r>',
        before=0, after=0, line=1.0)
    return ("<w:sdt><w:sdtPr><w:id w:val=\"707770780\"/><w:docPartObj>"
            '<w:docPartGallery w:val="Table of Contents"/><w:docPartUnique w:val="1"/>'
            f"</w:docPartObj></w:sdtPr><w:sdtContent>{inner}</w:sdtContent></w:sdt>")


# -------------------------------------------------------- markdown body ----
INLINE = re.compile(r"(\*\*[^*]+\*\*|\*[^*]+\*)")


def inline_runs(text, size_pt, color=None):
    out = []
    for part in INLINE.split(text):
        if not part:
            continue
        if part.startswith("**") and part.endswith("**"):
            out.append(run(part[2:-2], size_pt=size_pt, bold=True, color=color))
        elif part.startswith("*") and part.endswith("*"):
            out.append(run(part[1:-1], size_pt=size_pt, italic=True, color=color))
        else:
            out.append(run(part, size_pt=size_pt, color=color))
    return "".join(out)


def body_blocks(cfg, md):
    b = cfg["body"]
    fmt = cfg["heading_numbering"]["level1_format"]
    out = []
    h1 = 0
    first = True
    for raw in md.split("\n"):
        line = raw.rstrip()
        if not line.strip():
            continue
        m = re.match(r"^(#{1,6})\s+(.*)$", line)
        if m:
            lvl, txt = len(m.group(1)), m.group(2)
            if lvl == 1:
                h1 += 1
                txt = fmt.format(n=h1, title=txt)
            st = cfg["styles"][f"heading_{lvl}"]
            out.append(para(run(txt, color=st["color"]),
                            style=f"Heading{lvl}", indent_start=0,
                            page_break=(first and b["first_heading_page_break"])))
            first = False
            continue
        m = re.match(r"^[-*]\s+(.*)$", line)
        if m:
            out.append(para(inline_runs(m.group(1), b["size_pt"]),
                            numid=1, ilvl=0, before=0, after=0,
                            align=b["align"],
                            indent_start=b["bullet"]["indent_start_pt"],
                            hanging=b["bullet"]["hanging_pt"]))
            continue
        m = re.match(r"^\d+\.\s+(.*)$", line)
        if m:
            out.append(para(inline_runs(m.group(1), b["size_pt"]),
                            numid=2, ilvl=0, before=0, after=0,
                            align=b["align"],
                            indent_start=b["numbered"]["indent_start_pt"],
                            hanging=b["numbered"]["hanging_pt"]))
            continue
        out.append(para(inline_runs(line, b["size_pt"]),
                        align=b["align"], before=b["space_before_pt"],
                        after=b["space_after_pt"]))
    return out


# ------------------------------------------------------------- document ----
def cover_blocks(cfg, fields):
    c = cfg["cover"]
    out = [blank({**b, "align": b.get("align", c["align"])})
           for b in c["leading_blanks"]]
    for line in c["lines"]:
        ph = line.get("placeholder")
        if ph and fields.get(ph) is not None:
            r = run(fields[ph], size_pt=line["size_pt"], bold=line.get("bold", False))
        elif line.get("runs"):
            r = "".join(run(x.get("text", ""), size_pt=line["size_pt"],
                            bold=line.get("bold", False), color=x.get("color"),
                            highlight=x.get("highlight"))
                        for x in line["runs"])
        elif not line.get("text"):
            out.append(blank({**line, "align": c["align"]}))
            continue
        else:
            r = run(line.get("text", ""), size_pt=line["size_pt"],
                    bold=line.get("bold", False), highlight=line.get("highlight"))
        out.append(para(r, align=c["align"],
                        before=line.get("space_before_pt"),
                        after=line.get("space_after_pt")))
    out += [blank(b) for b in c["trailing_blanks"]]
    return out


def label_block(cfg, key):
    L = cfg["labels"][key]
    return para(run(L["text"], size_pt=L["size_pt"], bold=L.get("bold", False),
                    color=L["color"]),
                align=L.get("align"), before=L.get("space_before_pt"),
                after=L.get("space_after_pt"), line=L.get("line_spacing"),
                keep_lines=L.get("keep_lines_together", False))


def legend_block(cfg):
    g = cfg["legend"]
    out = []
    for bold_part, rest in g["lines"]:
        out.append(para(
            run(bold_part, size_pt=g["size_pt"], bold=True, color=g["color"])
            + "<w:r><w:tab/></w:r>"
            + run(rest, size_pt=g["size_pt"], color=g["color"]),
            before=0, after=0, line=g["line_spacing"]))
    return out


def document_xml(cfg, md, fields):
    blocks = []
    for item in cfg["front_matter"]:
        kind = item["block"]
        if kind == "cover":
            blocks += cover_blocks(cfg, fields)
        elif kind == "label":
            blocks.append(label_block(cfg, item["ref"]))
        elif kind == "table":
            blocks.append(table_xml(cfg, cfg["tables"][item["ref"]]))
        elif kind == "blank":
            blocks += [blank(item)] * item.get("count", 1)
        elif kind == "legend":
            blocks += legend_block(cfg)
        elif kind == "toc":
            blocks.append(toc_xml(cfg))
        else:
            raise ValueError(f"unknown block {kind}")
    blocks += body_blocks(cfg, md)

    p = cfg["page"]
    sect = (
        "<w:sectPr>"
        '<w:headerReference r:id="rId4" w:type="default"/>'
        '<w:headerReference r:id="rId5" w:type="first"/>'
        '<w:footerReference r:id="rId6" w:type="default"/>'
        '<w:footerReference r:id="rId7" w:type="first"/>'
        f'<w:pgSz w:w="{TW(p["width_pt"])}" w:h="{TW(p["height_pt"])}" '
        'w:orient="portrait"/>'
        f'<w:pgMar w:top="{TW(p["margin_top_pt"])}" '
        f'w:bottom="{TW(p["margin_bottom_pt"])}" '
        f'w:left="{TW(p["margin_left_pt"])}" w:right="{TW(p["margin_right_pt"])}" '
        f'w:header="{TW(p["header_margin_pt"])}" '
        f'w:footer="{TW(p["footer_margin_pt"])}"/>'
        f'<w:pgNumType w:start="{p["page_number_start"]}"/>'
        + ('<w:titlePg w:val="1"/>' if p["different_first_page"] else "")
        + "</w:sectPr>")
    return (XMLDECL + f"<w:document {NS}><w:body>" + "".join(blocks)
            + sect + "</w:body></w:document>")


CONTENT_TYPES = (XMLDECL +
 '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">'
 '<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>'
 '<Default Extension="xml" ContentType="application/xml"/>'
 '<Default Extension="png" ContentType="image/png"/>'
 '<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>'
 '<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>'
 '<Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/>'
 '<Override PartName="/word/settings.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.settings+xml"/>'
 '<Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>'
 '<Override PartName="/word/header2.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>'
 '<Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>'
 '<Override PartName="/word/footer2.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>'
 "</Types>")

ROOT_RELS = (XMLDECL +
 '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
 '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>'
 "</Relationships>")

R = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/"
DOC_RELS = (XMLDECL +
 '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
 f'<Relationship Id="rId1" Type="{R}styles" Target="styles.xml"/>'
 f'<Relationship Id="rId2" Type="{R}numbering" Target="numbering.xml"/>'
 f'<Relationship Id="rId3" Type="{R}settings" Target="settings.xml"/>'
 f'<Relationship Id="rId4" Type="{R}header" Target="header1.xml"/>'
 f'<Relationship Id="rId5" Type="{R}header" Target="header2.xml"/>'
 f'<Relationship Id="rId6" Type="{R}footer" Target="footer1.xml"/>'
 f'<Relationship Id="rId7" Type="{R}footer" Target="footer2.xml"/>'
 "</Relationships>")

H2_RELS = (XMLDECL +
 '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
 f'<Relationship Id="rId1" Type="{R}image" Target="media/logo.png"/>'
 "</Relationships>")


def build(config_path, body_path, out_path, fields):
    cfg = yaml.safe_load(Path(config_path).read_text())
    md = Path(body_path).read_text()
    parts = {
        "[Content_Types].xml": CONTENT_TYPES,
        "_rels/.rels": ROOT_RELS,
        "word/_rels/document.xml.rels": DOC_RELS,
        "word/_rels/header2.xml.rels": H2_RELS,
        "word/document.xml": document_xml(cfg, md, fields),
        "word/styles.xml": styles_xml(cfg),
        "word/numbering.xml": numbering_xml(cfg),
        "word/settings.xml": settings_xml(),
        "word/header1.xml": header_xml(cfg, "default"),
        "word/header2.xml": header_xml(cfg, "first", rid="rId1"),
        "word/footer1.xml": footer_xml(cfg, "default"),
        "word/footer2.xml": footer_xml(cfg, "first"),
    }
    if fields.get("running_head"):
        parts["word/header1.xml"] = parts["word/header1.xml"].replace(
            escape("Altery - xxx Policy"), escape(fields["running_head"]))
    with zipfile.ZipFile(out_path, "w", zipfile.ZIP_DEFLATED) as z:
        for name, xml in parts.items():
            z.writestr(name, xml)
        z.writestr("word/media/logo.png", base64.b64decode(cfg["logo"]["base64"]))
    return out_path


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--config", default="house.yaml")
    ap.add_argument("--body", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--title")
    ap.add_argument("--version")
    ap.add_argument("--date")
    ap.add_argument("--running-head")
    a = ap.parse_args()
    f = {"title": a.title, "version": a.version, "date": a.date,
         "running_head": a.running_head}
    build(a.config, a.body, a.out, {k: v for k, v in f.items() if v})
    print("wrote", a.out, Path(a.out).stat().st_size, "bytes")
