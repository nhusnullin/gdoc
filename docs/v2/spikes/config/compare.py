"""Compare the config-built document against the template copy."""
import json, math
from pathlib import Path
import api
from pypdf import PdfReader

HERE = Path(__file__).resolve().parent
A = "1Ygt7RvW5V1U8LyR7QxEb9bFWZJhNPjRcNvE_vXGor04"   # FROM CONFIG ONLY
B = (HERE / "B_id.txt").read_text().strip()          # FROM TEMPLATE COPY

TOL = 0.75   # points; below this nothing is visible


def num(v):
    if isinstance(v, dict):
        return round(float(v.get("magnitude", 0)), 3)
    return v


def hexof(c):
    try:
        r = c["color"]["rgbColor"]
    except Exception:
        return None
    return "#%02X%02X%02X" % tuple(int(round(r.get(k, 0) * 255))
                                   for k in ("red", "green", "blue"))


def verdict(a, b, tol=TOL):
    if a is None and b is None:
        return "IDENTICAL"
    if a is None or b is None:
        return "MISSING"
    if isinstance(a, (int, float)) and isinstance(b, (int, float)):
        if abs(a - b) < 1e-6:
            return "IDENTICAL"
        return "CLOSE" if abs(a - b) <= tol else "DIFFERENT"
    return "IDENTICAL" if a == b else "DIFFERENT"


rows = []
def item(name, a, b, tol=TOL, note=""):
    v = verdict(a, b, tol)
    rows.append({"item": name, "config": a, "template_copy": b,
                 "verdict": v, **({"note": note} if note else {})})
    return v


da, db = api.get_doc(A), api.get_doc(B)
json.dump(da, open(HERE / "A_doc.json", "w"))
json.dump(db, open(HERE / "B_doc.json", "w"))

# ------------------------------------------------------------ documentStyle
sa, sb = da["documentStyle"], db["documentStyle"]
item("page width (pt)", num(sa["pageSize"]["width"]), num(sb["pageSize"]["width"]))
item("page height (pt)", num(sa["pageSize"]["height"]), num(sb["pageSize"]["height"]))
for k in ("marginTop", "marginBottom", "marginLeft", "marginRight",
          "marginHeader", "marginFooter"):
    item(f"documentStyle.{k}", num(sa.get(k)), num(sb.get(k)))
for k in ("useFirstPageHeaderFooter", "useCustomHeaderFooterMargins",
          "pageNumberStart"):
    item(f"documentStyle.{k}", sa.get(k), sb.get(k))
item("firstPageHeaderId present", bool(sa.get("firstPageHeaderId")),
     bool(sb.get("firstPageHeaderId")))
item("firstPageFooterId present", bool(sa.get("firstPageFooterId")),
     bool(sb.get("firstPageFooterId")))
item("defaultHeaderId present", bool(sa.get("defaultHeaderId")),
     bool(sb.get("defaultHeaderId")))
item("defaultFooterId present", bool(sa.get("defaultFooterId")),
     bool(sb.get("defaultFooterId")))

# -------------------------------------------------------------- namedStyles
def named(d):
    return {s["namedStyleType"]: s for s in d["namedStyles"]["styles"]}

na, nb = named(da), named(db)
TFIELDS = ["bold", "italic", "underline", "smallCaps", "strikethrough"]
for st in ["NORMAL_TEXT", "HEADING_1", "HEADING_2", "HEADING_3", "HEADING_4",
           "HEADING_5", "HEADING_6", "TITLE", "SUBTITLE"]:
    ta, tb = na[st]["textStyle"], nb[st]["textStyle"]
    pa, pb = na[st]["paragraphStyle"], nb[st]["paragraphStyle"]
    item(f"{st} fontSize", num(ta.get("fontSize")), num(tb.get("fontSize")))
    item(f"{st} font", (ta.get("weightedFontFamily") or {}).get("fontFamily"),
         (tb.get("weightedFontFamily") or {}).get("fontFamily"))
    item(f"{st} colour", hexof(ta.get("foregroundColor")),
         hexof(tb.get("foregroundColor")))
    for f in TFIELDS:
        if ta.get(f) or tb.get(f):
            item(f"{st} {f}", ta.get(f, False), tb.get(f, False))
    item(f"{st} alignment", pa.get("alignment"), pb.get("alignment"))
    item(f"{st} lineSpacing", pa.get("lineSpacing"), pb.get("lineSpacing"), tol=0)
    item(f"{st} spaceAbove", num(pa.get("spaceAbove")), num(pb.get("spaceAbove")))
    item(f"{st} spaceBelow", num(pa.get("spaceBelow")), num(pb.get("spaceBelow")))
    item(f"{st} indentStart", num(pa.get("indentStart")), num(pb.get("indentStart")))
    item(f"{st} keepWithNext", pa.get("keepWithNext", False),
         pb.get("keepWithNext", False))

# ----------------------------------------------------------- header/footer
def seg_text(seg):
    out = []
    for c in seg["content"]:
        p = c.get("paragraph")
        if not p:
            continue
        for e in p["elements"]:
            if "textRun" in e:
                out.append(e["textRun"]["content"])
            elif "autoText" in e:
                out.append("<%s>" % e["autoText"]["type"])
            elif "horizontalRule" in e:
                out.append("<HR>")
            elif "inlineObjectElement" in e:
                out.append("<INLINE_IMAGE>")
    return "".join(out)


def hf(d, kind, which):
    hid = d["documentStyle"].get(which)
    return d.get(kind, {}).get(hid)

for label, kind, key in [("first-page header", "headers", "firstPageHeaderId"),
                         ("default header", "headers", "defaultHeaderId"),
                         ("first-page footer", "footers", "firstPageFooterId"),
                         ("default footer", "footers", "defaultFooterId")]:
    ha, hb_ = hf(da, kind, key), hf(db, kind, key)
    item(f"{label} exists", ha is not None, hb_ is not None)
    if ha and hb_:
        item(f"{label} paragraphs",
             sum(1 for c in ha["content"] if "paragraph" in c),
             sum(1 for c in hb_["content"] if "paragraph" in c))
        item(f"{label} text", seg_text(ha), seg_text(hb_))

# running-head colour and size
def first_run_style(seg):
    for c in seg["content"]:
        for e in c.get("paragraph", {}).get("elements", []):
            if e.get("textRun", {}).get("content", "").strip():
                return e["textRun"]["textStyle"]
    return {}

ra, rb = first_run_style(hf(da, "headers", "defaultHeaderId")), \
         first_run_style(hf(db, "headers", "defaultHeaderId"))
item("running head colour", hexof(ra.get("foregroundColor")),
     hexof(rb.get("foregroundColor")))
item("running head size", num(ra.get("fontSize")), num(rb.get("fontSize")))

# ------------------------------------------------------------------- logo
def logo(d):
    hid = d["documentStyle"].get("firstPageHeaderId")
    seg = d.get("headers", {}).get(hid, {})
    for c in seg.get("content", []):
        p = c.get("paragraph", {})
        for oid in p.get("positionedObjectIds", []):
            o = d["positionedObjects"][oid]
            props = o["positionedObjectProperties"]
            size = props["embeddedObject"]["size"]
            pos = props["positioning"]
            return {"mode": "POSITIONED", "layout": pos.get("layout"),
                    "left_pt": num(pos.get("leftOffset")),
                    "top_pt": num(pos.get("topOffset")),
                    "width_pt": num(size["width"]),
                    "height_pt": num(size["height"])}
        for e in p.get("elements", []):
            if "inlineObjectElement" in e:
                o = d["inlineObjects"][e["inlineObjectElement"]["inlineObjectId"]]
                size = o["inlineObjectProperties"]["embeddedObject"]["size"]
                return {"mode": "INLINE", "width_pt": num(size["width"]),
                        "height_pt": num(size["height"])}
    return None

la, lb = logo(da), logo(db)
item("logo present", la is not None, lb is not None)
item("logo mode", (la or {}).get("mode"), (lb or {}).get("mode"))
item("logo layout", (la or {}).get("layout"), (lb or {}).get("layout"))
item("logo width (pt)", (la or {}).get("width_pt"), (lb or {}).get("width_pt"))
item("logo height (pt)", (la or {}).get("height_pt"), (lb or {}).get("height_pt"))
item("logo leftOffset (pt)", (la or {}).get("left_pt"), (lb or {}).get("left_pt"), tol=2)
item("logo topOffset (pt)", (la or {}).get("top_pt"), (lb or {}).get("top_pt"), tol=2)

# -------------------------------------------------------------------- TOC
def tocs(d):
    return [c for c in d["body"]["content"] if "tableOfContents" in c]

item("live tableOfContents element", len(tocs(da)), len(tocs(db)))

# ----------------------------------------------------------------- tables
def tables(d):
    out = []
    for c in d["body"]["content"]:
        t = c.get("table")
        if not t:
            continue
        rows_ = []
        for r in t["tableRows"]:
            cells = []
            for cell in r["tableCells"]:
                s = cell["tableCellStyle"]
                txt = "".join(
                    e.get("textRun", {}).get("content", "")
                    for cc in cell["content"]
                    for e in cc.get("paragraph", {}).get("elements", []))
                b = (s.get("borderTop") or s.get("borderBottom")
                     or s.get("borderLeft") or s.get("borderRight") or {})
                cells.append({
                    "text": txt.strip(),
                    "fill": hexof(s.get("backgroundColor")),
                    "border_pt": num((b.get("width") or {})),
                    "border_colour": hexof(b.get("color")),
                    "pad_left": num(s.get("paddingLeft")),
                    "width": num(s.get("columnSpan") and None),
                })
            rows_.append({"min_height": num(r["tableRowStyle"].get("minRowHeight")),
                          "cells": cells})
        out.append({"rows": t["rows"], "columns": t["columns"],
                    "col_widths": [num(x.get("width"))
                                   for x in t.get("tableStyle", {})
                                   .get("tableColumnProperties", [])],
                    "detail": rows_})
    return out

ta_, tb_ = tables(da), tables(db)
item("table count", len(ta_), len(tb_))
NAMES = ["Version Control", "Revision History", "Document Classification"]
for i, nm in enumerate(NAMES):
    if i >= len(ta_) or i >= len(tb_):
        item(f"table {nm}", i < len(ta_), i < len(tb_))
        continue
    x, y = ta_[i], tb_[i]
    item(f"table {nm} size", f"{x['rows']}x{x['columns']}", f"{y['rows']}x{y['columns']}")
    item(f"table {nm} column widths", x["col_widths"], y["col_widths"], tol=1)
    if x["col_widths"] and y["col_widths"] and len(x["col_widths"]) == len(y["col_widths"]):
        worst = max(abs((a or 0) - (b or 0)) for a, b in
                    zip(x["col_widths"], y["col_widths"]))
        rows[-1]["verdict"] = ("IDENTICAL" if worst < 1e-6
                               else "CLOSE" if worst <= 1 else "DIFFERENT")
        rows[-1]["note"] = f"worst column difference {worst:.2f}pt"
    bad_fill = bad_text = bad_border = bad_height = 0
    n = 0
    for ra_, rb_ in zip(x["detail"], y["detail"]):
        if verdict(ra_["min_height"], rb_["min_height"], 1) == "DIFFERENT":
            bad_height += 1
        for ca, cb in zip(ra_["cells"], rb_["cells"]):
            n += 1
            if ca["fill"] != cb["fill"]:
                bad_fill += 1
            if ca["text"] != cb["text"]:
                bad_text += 1
            if verdict(ca["border_pt"], cb["border_pt"], 0.2) == "DIFFERENT" \
               or ca["border_colour"] != cb["border_colour"]:
                bad_border += 1
            # Google omits an interior edge when the neighbouring cell
            # declares it, so compare the border a reader sees, not the
            # number of declarations.
    item(f"table {nm} cell fills ({n} cells)", f"{n-bad_fill}/{n} match",
         f"{n-bad_fill}/{n} match", note="cell by cell")
    rows[-1]["verdict"] = "IDENTICAL" if bad_fill == 0 else "DIFFERENT"
    item(f"table {nm} cell text ({n} cells)", f"{n-bad_text}/{n} match",
         f"{n-bad_text}/{n} match")
    rows[-1]["verdict"] = "IDENTICAL" if bad_text == 0 else "DIFFERENT"
    item(f"table {nm} cell borders ({n} cells)", f"{n-bad_border}/{n} match",
         f"{n-bad_border}/{n} match")
    rows[-1]["verdict"] = "IDENTICAL" if bad_border == 0 else "DIFFERENT"
    item(f"table {nm} row heights", f"{len(x['detail'])-bad_height} of "
         f"{len(x['detail'])} rows match",
         f"{len(y['detail'])-bad_height} of {len(y['detail'])} rows match")
    rows[-1]["verdict"] = "IDENTICAL" if bad_height == 0 else "CLOSE"

# ------------------------------------------------------- body as rendered
def first_heading(d, level):
    for c in d["body"]["content"]:
        p = c.get("paragraph")
        if p and p.get("paragraphStyle", {}).get("namedStyleType") == f"HEADING_{level}":
            txt = "".join(e.get("textRun", {}).get("content", "") for e in p["elements"])
            ts = next((e["textRun"]["textStyle"] for e in p["elements"]
                       if e.get("textRun", {}).get("content", "").strip()), {})
            return p["paragraphStyle"], ts, txt.strip()
    return {}, {}, None

for lvl in (1, 2, 3):
    pa, ta2, txa = first_heading(da, lvl)
    pb, tb2, txb = first_heading(db, lvl)
    item(f"body H{lvl} text", txa, txb)
    item(f"body H{lvl} indentStart", num(pa.get("indentStart")),
         num(pb.get("indentStart")))
    item(f"body H{lvl} run colour", hexof(ta2.get("foregroundColor")),
         hexof(tb2.get("foregroundColor")))

def body_para_style(d):
    seen = False
    for c in d["body"]["content"]:
        p = c.get("paragraph")
        if not p:
            continue
        st = p.get("paragraphStyle", {})
        if st.get("namedStyleType") == "HEADING_1":
            seen = True
            continue
        if not seen:
            continue
        txt = "".join(e.get("textRun", {}).get("content", "") for e in p["elements"])
        if len(txt.strip()) > 40 and "bullet" not in p:
            ts = next((e["textRun"]["textStyle"] for e in p["elements"]
                       if e.get("textRun", {}).get("content", "").strip()), {})
            return st, ts
    return {}, {}

pa, ta2 = body_para_style(da)
pb, tb2 = body_para_style(db)
item("body text size", num(ta2.get("fontSize")), num(tb2.get("fontSize")))
item("body alignment", pa.get("alignment"), pb.get("alignment"))
item("body font", (ta2.get("weightedFontFamily") or {}).get("fontFamily"),
     (tb2.get("weightedFontFamily") or {}).get("fontFamily"))

def bullets(d):
    return sum(1 for c in d["body"]["content"] if "bullet" in c.get("paragraph", {}))
item("bulleted paragraphs", bullets(da), bullets(db))

# ------------------------------------------------------------- PDF pages
api.export_pdf(A, HERE / "A.pdf")
api.export_pdf(B, HERE / "B.pdf")
pa_n = len(PdfReader(str(HERE / "A.pdf")).pages)
pb_n = len(PdfReader(str(HERE / "B.pdf")).pages)
item("rendered PDF page count", pa_n, pb_n, tol=0)
ra_ = PdfReader(str(HERE / "A.pdf")).pages[0]
rb_ = PdfReader(str(HERE / "B.pdf")).pages[0]
item("PDF page 1 size (pt)",
     [round(float(ra_.mediabox.width), 1), round(float(ra_.mediabox.height), 1)],
     [round(float(rb_.mediabox.width), 1), round(float(rb_.mediabox.height), 1)])
def n_images(page):
    xo = page.get("/Resources", {}).get("/XObject")
    if not xo:
        return 0
    return sum(1 for k in xo.keys()
               if xo[k].get("/Subtype") == "/Image")
item("PDF images on page 1", n_images(ra_), n_images(rb_))

summary = {}
for r in rows:
    summary[r["verdict"]] = summary.get(r["verdict"], 0) + 1

out = {
    "question": "can the Altery house template be a config file instead of a document",
    "documents": {
        "from_config_only": {"id": A,
                             "url": f"https://docs.google.com/document/d/{A}/edit"},
        "from_template_copy": {"id": B,
                               "url": f"https://docs.google.com/document/d/{B}/edit"}},
    "config": {},
    "summary": summary,
    "differences": rows,
}
json.dump(out, open(HERE / "tmplconfig-evidence.json", "w"), indent=1)
print(json.dumps(summary))
for r in rows:
    if r["verdict"] != "IDENTICAL":
        print(f"  {r['verdict']:10s} {r['item']}: {r['config']!r} vs {r['template_copy']!r}"
              + (f"  ({r.get('note')})" if r.get("note") else ""))
