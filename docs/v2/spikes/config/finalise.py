"""Merge the structural comparison, the rendered measurements and the config
sizes into one evidence file."""
import base64, json
from pathlib import Path
import yaml

HERE = Path(__file__).resolve().parent
ev = json.load(open(HERE / "tmplconfig-evidence.json"))
cfg_text = (HERE / "house.yaml").read_text()
cfg = yaml.safe_load(cfg_text)
b64 = len(cfg["logo"]["base64"])

# Declared-style items: the config states the value a reader actually sees,
# where the template declares one thing and overrides it on every paragraph.
DECLARED_ONLY = {
    "HEADING_1 colour", "HEADING_2 colour",
    "HEADING_1 indentStart", "HEADING_2 indentStart", "HEADING_3 indentStart",
    "HEADING_4 indentStart", "HEADING_5 indentStart", "HEADING_6 indentStart",
}
INHERITED = {
    "HEADING_3 spaceBelow", "HEADING_4 fontSize", "HEADING_4 colour",
    "HEADING_4 spaceBelow", "HEADING_5 fontSize", "HEADING_5 colour",
    "HEADING_5 spaceBelow", "HEADING_6 fontSize", "HEADING_6 colour",
    "HEADING_6 spaceBelow", "TITLE colour", "SUBTITLE lineSpacing",
    "body H3 run colour",
}
for r in ev["differences"]:
    if r["item"] in DECLARED_ONLY:
        r["classification"] = "declared-only"
        r["reader_sees"] = ("identical: every heading in the template's body "
                            "overrides the declared value, and the config "
                            "states the overridden value instead")
    elif r["item"] in INHERITED:
        r["classification"] = "inherited-vs-stated"
        r["reader_sees"] = ("identical: the template leaves the field unset "
                            "and inherits exactly the value the config states")
    elif r["item"] == "rendered PDF page count":
        r["classification"] = "stale table of contents"
        r["reader_sees"] = ("the template copy carries the master's stale "
                            "contents list, 14 entries over a page. The "
                            "config build carries a one-line placeholder. "
                            "Both are stale until refreshed; neither is "
                            "house furniture")
    elif r["verdict"] != "IDENTICAL":
        r["classification"] = "measured"

summary = {}
for r in ev["differences"]:
    summary[r["verdict"]] = summary.get(r["verdict"], 0) + 1
visible = [r for r in ev["differences"]
           if r["verdict"] != "IDENTICAL"
           and r.get("classification") not in ("declared-only",
                                               "inherited-vs-stated")]

ev["summary"] = summary
ev["config"] = {
    "file": "house.yaml",
    "bytes": len(cfg_text.encode()),
    "logo_base64_bytes": b64,
    "logo_share_percent": round(100.0 * b64 / len(cfg_text.encode()), 1),
    "bytes_without_logo": len(cfg_text.encode()) - b64,
    "logo_png_bytes": len(base64.b64decode(cfg["logo"]["base64"])),
    "top_level_keys": list(cfg.keys()),
}
ev["rendered"] = {
    "word_position_offsets": json.load(open(HERE / "word-offsets.json")),
    "pixel_diff": json.load(open(HERE / "pixel-diff.json")),
    "note": ("positions measured from the two PDF exports with pdftotext "
             "-bbox, in points. Pages 1 and 2 carry all the house furniture; "
             "page 3 differs only in stale contents-list text"),
}
ev["visible_differences"] = [
    {"item": r["item"], "config": r["config"], "template_copy": r["template_copy"],
     "verdict": r["verdict"]} for r in visible]
ev["method"] = {
    "arm_A": ("house.yaml plus body.md -> gen.py emits a .docx from scratch "
              "-> uploaded to Drive with conversion. No master .docx is read."),
    "arm_B": ("files.copy of the template Google Doc -> Docs API fills the "
              "title, date and running head, deletes the sample body and "
              "writes the same body content."),
    "body": "the same body.md in both arms",
}
json.dump(ev, open(HERE / "tmplconfig-evidence.json", "w"), indent=1)
print(json.dumps(summary))
print("visible differences:", len(visible))
for r in visible:
    print("  ", r["verdict"], r["item"], "|", r["config"], "vs", r["template_copy"])
print("config:", json.dumps(ev["config"], indent=1))
