"""Read the three front-matter tables out of the template document, once.

Read only. The output is a config fragment; after this the template document
is not needed again.
"""
import json
import api

TABLES = ["version_control", "revision_history", "document_classification"]


def num(v):
    return round(float(v.get("magnitude", 0)), 3) if isinstance(v, dict) else None


def hexof(c):
    try:
        r = c["color"]["rgbColor"]
    except Exception:
        return None
    return "#%02X%02X%02X" % tuple(int(round(r.get(k, 0) * 255))
                                   for k in ("red", "green", "blue"))


def cell_paras(cell):
    out = []
    for c in cell["content"]:
        p = c.get("paragraph")
        if not p:
            continue
        ps = p.get("paragraphStyle", {})
        runs = []
        for e in p["elements"]:
            tr = e.get("textRun")
            if not tr:
                continue
            txt = tr["content"].replace("\n", "")
            ts = tr.get("textStyle", {})
            r = {"text": txt}
            if ts.get("bold"):
                r["bold"] = True
            if ts.get("italic"):
                r["italic"] = True
            fg = hexof(ts.get("foregroundColor"))
            if fg and fg != "#000000":
                r["color"] = fg
            bg = hexof(ts.get("backgroundColor"))
            if bg:
                r["highlight"] = bg
            if ts.get("fontSize"):
                r["size_pt"] = num(ts["fontSize"])
            if txt:
                runs.append(r)
        # Only what the template states. An absent field is inherited, and
        # writing a value for it is how a copy stops looking like the master.
        d = {"runs": runs}
        if ps.get("alignment"):
            d["align"] = (ps["alignment"].lower().replace("start", "left")
                          .replace("end", "right").replace("justified", "justify"))
        if "lineSpacing" in ps:
            d["line_spacing"] = ps["lineSpacing"] / 100.0
        if "spaceAbove" in ps:
            d["space_before_pt"] = num(ps["spaceAbove"])
        if "spaceBelow" in ps:
            d["space_after_pt"] = num(ps["spaceBelow"])
        if num(ps.get("indentStart")):
            d["indent_start_pt"] = num(ps["indentStart"])
        if num(ps.get("indentFirstLine")):
            d["indent_first_line_pt"] = num(ps["indentFirstLine"])
        if ps.get("keepLinesTogether"):
            d["keep_lines_together"] = True
        if ps.get("keepWithNext"):
            d["keep_with_next"] = True
        out.append(d)
    return out


def main():
    d = api.get_doc(api.TEMPLATE)                 # READ ONLY
    tables = [c["table"] for c in d["body"]["content"] if "table" in c][:3]
    out = {}
    for name, t in zip(TABLES, tables):
        cols = [num(x.get("width")) for x in
                t["tableStyle"]["tableColumnProperties"]]
        b = t["tableRows"][0]["tableCells"][0]["tableCellStyle"]["borderTop"]
        rows = []
        for r in t["tableRows"]:
            cells = []
            for cell in r["tableCells"]:
                s = cell["tableCellStyle"]
                pads = [num(s.get(k)) for k in
                        ("paddingTop", "paddingLeft", "paddingBottom",
                         "paddingRight")]
                # an absent padding field means zero, and zero is not "unset":
                # two of the three tables rely on it
                # An absent padding field is Google's own default, which
                # renders as no vertical padding and 5pt horizontal. A docx
                # has to say that explicitly.
                default = [0, 5, 0, 5]
                pads = [d if p is None else p for p, d in zip(pads, default)]
                cells.append({
                    "fill": hexof(s.get("backgroundColor")),
                    "pad_pt": pads[1],
                    "pad_tlbr_pt": pads,
                    "valign": (s.get("contentAlignment") or "TOP").lower(),
                    "paragraphs": cell_paras(cell),
                })
            rows.append({"min_height_pt": num(r["tableRowStyle"]
                                              .get("minRowHeight")) or 0,
                         "cells": cells})
        out[name] = {"columns_pt": cols,
                     "border": {"width_pt": num(b.get("width")),
                                "color": hexof(b.get("color"))},
                     "rows": rows}
    json.dump(out, open("tables.json", "w"), indent=1)
    print("wrote tables.json", len(json.dumps(out)), "chars")
    for k, v in out.items():
        print(" ", k, len(v["rows"]), "rows",
              "border", v["border"]["width_pt"], v["border"]["color"])


if __name__ == "__main__":
    main()
