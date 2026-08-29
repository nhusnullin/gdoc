"""Write house.yaml from the already-extracted style profile plus the logo bytes.

Run once. The output is the deliverable; this script only exists so the config
is reproducible and so nobody has to hand-type a 7 KB base64 blob.
"""
import base64, json, sys
from pathlib import Path
import yaml

HERE = Path(__file__).resolve().parent
PROFILE = HERE.parent / "houselook" / "style-profile.json"
LOGO = HERE / "master" / "word" / "media" / "image1.png"

prof = json.loads(PROFILE.read_text())
pal = prof["palette"]
ds = prof["documentStyle"]

# Table detail comes from extract_tables.py, which read the template once.
TABLES = json.loads((HERE / "tables.json").read_text())


def tbl(name):
    return TABLES[name]

cfg = {
    "meta": {
        "name": "altery-group-policy",
        "version": "1.0",
        "note": "Everything gdoc needs to build the Altery house document. "
                "No .docx master, no template document in Drive.",
    },
    "page": {
        "size": "A4",
        "width_pt": round(ds["pageSizePT"]["width"], 2),
        "height_pt": round(ds["pageSizePT"]["height"], 2),
        "margin_top_pt": ds["marginTopPT"],
        "margin_bottom_pt": ds["marginBottomPT"],
        "margin_left_pt": ds["marginLeftPT"],
        "margin_right_pt": ds["marginRightPT"],
        "header_margin_pt": ds["marginHeaderPT"],
        "footer_margin_pt": ds["marginFooterPT"],
        "different_first_page": True,
        "page_number_start": ds["pageNumberStart"],
    },
    "palette": {
        "navy": pal["houseNavy"],
        "heading_blue_declared": pal["namedStyleHeadingBlue"],
        "teal": pal["teal"],
        "running_head_orange": pal["headerOrange"],
        "subtitle_grey": pal["subtitleGrey"],
        "legend_navy": "#0F1340",
        "rule_grey": "#A0A0A0",
        "highlight_yellow": pal["coverPlaceholderHighlight"],
        "placeholder_red": pal["coverPlaceholderRed"],
    },
    "fonts": {"body": "Calibri", "title": "Arial", "subtitle": "Georgia"},
    # docDefaults, folded into NORMAL_TEXT by Google's importer
    "defaults": {"font": "Calibri", "size_pt": 11,
                 "space_before_pt": 3, "space_after_pt": 6, "line_spacing": 1.15},
    "styles": {
        "normal":    {"size_pt": 11, "color": "#000000", "space_before_pt": 3,
                      "space_after_pt": 6, "line_spacing": 1.15},
        # EFFECTIVE values: what every heading in the body actually shows,
        # not what the master's named style declares (indent 36pt, #06436E).
        "heading_1": {"size_pt": 16, "bold": True, "color": "#22265F",
                      "space_before_pt": 12, "space_after_pt": 0,
                      "indent_start_pt": 0, "keep_with_next": True},
        "heading_2": {"size_pt": 14, "bold": False, "color": "#22265F",
                      "space_before_pt": 12, "space_after_pt": 0,
                      "indent_start_pt": 0, "keep_with_next": True},
        "heading_3": {"size_pt": 12, "bold": False, "color": "#549F99",
                      "space_before_pt": 12, "space_after_pt": 6,
                      "indent_start_pt": 0, "keep_with_next": True},
        "heading_4": {"size_pt": 11, "bold": True, "color": "#000000",
                      "space_before_pt": 12, "space_after_pt": 6,
                      "indent_start_pt": 0, "keep_with_next": True},
        "heading_5": {"size_pt": 11, "bold": True, "italic": True, "color": "#000000",
                      "space_before_pt": 12, "space_after_pt": 6, "indent_start_pt": 0},
        "heading_6": {"size_pt": 11, "bold": True, "color": "#000000",
                      "space_before_pt": 12, "space_after_pt": 6, "indent_start_pt": 0},
        "title":     {"size_pt": 16, "bold": True, "color": "#000000",
                      "font": "Arial", "align": "center",
                      "space_before_pt": 12, "space_after_pt": 3, "line_spacing": 1.0},
        "subtitle":  {"size_pt": 24, "italic": True, "color": "#666666",
                      "font": "Georgia", "space_before_pt": 18, "space_after_pt": 4,
                      "line_spacing": 1.15, "keep_with_next": True,
                      "keep_lines_together": True},
    },
    # How generated body content is written. The master's own body overrides
    # NORMAL_TEXT to 12pt with zero paragraph spacing, so a document that used
    # the declared 11pt would not look like the house style.
    "body": {
        "size_pt": 12, "align": "justify",
        "space_before_pt": 0, "space_after_pt": 0,
        "first_heading_page_break": True,
        "bullet": {"indent_start_pt": 36, "hanging_pt": 18,
                   "glyphs": ["●", "○", "■"]},
        "numbered": {"indent_start_pt": 36, "hanging_pt": 18, "format": "%1."},
    },
    "header": {
        "first": {
            # line spacing is per paragraph here: the middle one inherits
            "paragraphs": [
                {"align": "left", "line_spacing": 2.0,
                 "space_before_pt": 0, "space_after_pt": 0, "runs": [
                    {"text": "For internal use only ", "size_pt": 12, "color": "#22265F"},
                    {"tab": True}], "logo": True},
                # this one states nothing, and inherits, which is 9pt of height
                {"align": "center", "runs": []},
                {"align": "left", "line_spacing": 2.0,
                 "space_before_pt": 0, "space_after_pt": 0, "runs": []},
            ],
        },
        "default": {
            "align": "right",
            "paragraphs": [
                {"space_before_pt": 0, "space_after_pt": 0,
                 "runs": [{"text": "Altery - xxx Policy", "size_pt": 12,
                           "color": "#FF771C", "placeholder": "running_head"}]},
                {"space_before_pt": 0, "space_after_pt": 0, "size_pt": 9,
                 "rule": {"height_pt": 1.5, "color": "#A0A0A0"}},
                {"space_before_pt": 0, "space_after_pt": 0, "size_pt": 9,
                 "runs": []},
            ],
        },
    },
    "footer": {
        "first": {
            "line_spacing": 2.0, "indent_start_pt": 108,
            "paragraphs": [{"space_before_pt": 0, "space_after_pt": 0, "runs": [
                {"tabs": 6, "size_pt": 12, "color": "#22265F"},
                {"text": " " * 31, "size_pt": 12, "color": "#22265F"},
                {"tabs": 2, "size_pt": 12, "color": "#22265F"},
                {"page_number": True, "size_pt": 12, "color": "#22265F"}]}],
        },
        "default": {
            "line_spacing": 2.0,
            "paragraphs": [
                {"runs": [], "size_pt": 12, "color": "#22265F",
                 "space_before_pt": 0, "space_after_pt": 0},
                {"space_before_pt": 0, "space_after_pt": 0, "runs": [
                    {"text": "For internal use only ", "size_pt": 12, "color": "#22265F"},
                    {"tabs": 7, "size_pt": 12, "color": "#22265F"},
                    {"text": " " * 23, "size_pt": 12, "color": "#22265F"},
                    {"tabs": 3, "size_pt": 12, "color": "#22265F"},
                    {"page_number": True, "size_pt": 12, "color": "#22265F"}]},
            ],
        },
    },
    "cover": {
        # empty spacer paragraphs carry the cover's vertical rhythm, so their
        # run size and paragraph spacing are part of the house look
        "leading_blanks": [{}, {}, {}, {"size_pt": 18}, {"size_pt": 18}],
        "align": "center",
        "lines": [
            {"text": "Altery Group ", "size_pt": 29, "bold": True,
             "space_before_pt": 0, "space_after_pt": 0},
            {"text": "", "size_pt": 29, "bold": True,
             "space_before_pt": 0, "space_after_pt": 0},
            {"size_pt": 29, "bold": True, "placeholder": "title",
             "space_before_pt": 0, "space_after_pt": 0,
             "runs": [{"text": "(Name of)", "highlight": "yellow"},
                      {"text": " "},
                      {"text": "Framework/Policy", "color": "#FF0000"}]},
            {"text": "or", "size_pt": 29, "bold": True,
             "space_before_pt": 0, "space_after_pt": 0},
            # the alternative wording, which keeps its "fill me in" marks
            {"size_pt": 29, "bold": True,
             "space_before_pt": 0, "space_after_pt": 0,
             "runs": [{"text": "(Name of)", "highlight": "yellow"},
                      {"text": " "},
                      {"text": "Framework/Policy", "color": "#FF0000"}]},
            # the version and date lines inherit NORMAL_TEXT spacing
            {"size_pt": 20, "placeholder": "version",
             "runs": [{"text": "Version: "}, {"text": "1.0", "highlight": "yellow"}]},
            {"text": "May 2025", "size_pt": 20, "highlight": "yellow",
             "placeholder": "date"},
        ],
        "trailing_blanks": (
            [{"align": "justify", "space_before_pt": 18, "space_after_pt": 0,
              "size_pt": 10}] * 7
            + [{"align": "justify", "space_before_pt": 18, "space_after_pt": 0,
                "size_pt": 12}]),
    },
    "labels": {
        "version_control": {"text": "Version Control", "bold": True, "size_pt": 12,
                            "color": "#22265F", "align": "justify",
                            "space_before_pt": 18, "space_after_pt": 0},
        "document_classification": {
            "text": "Document Classification (Highlight as appropriate)",
            "bold": True, "size_pt": 12, "color": "#22265F",
            "line_spacing": 1.0, "space_before_pt": 0, "space_after_pt": 0},
        "contents": {"text": "Contents", "size_pt": 16, "color": "#22265F",
                     "space_before_pt": 24, "space_after_pt": 0,
                     "keep_lines_together": True},
    },
    "legend": {
        "size_pt": 12, "color": "#0F1340", "line_spacing": 1.0,
        "lines": [
            ["New         ", "  New information has been added"],
            ["Update     ", "  Existing information has been updated"],
            ["Amend     ", "  Existing information has been amended"],
            ["Remove    ", "  Existing information has been removed"],
        ],
    },
    # Order and spacing of the front matter. The blank paragraphs are not
    # padding: they carry the vertical rhythm, so each states its own size and
    # spacing.
    "front_matter": [
        {"block": "cover"},
        {"block": "label", "ref": "version_control"},
        {"block": "table", "ref": "version_control"},
        {"block": "blank", "align": "justify", "space_before_pt": 6,
         "space_after_pt": 0},
        {"block": "table", "ref": "revision_history"},
        {"block": "blank", "size_pt": 12, "line_spacing": 1.0,
         "space_after_pt": 0},
        {"block": "legend"},
        {"block": "blank", "size_pt": 12, "line_spacing": 1.0,
         "space_before_pt": 0, "space_after_pt": 0},
        {"block": "label", "ref": "document_classification"},
        {"block": "table", "ref": "document_classification"},
        {"block": "blank", "count": 3, "size_pt": 16, "space_before_pt": 24,
         "space_after_pt": 0},
        {"block": "label", "ref": "contents"},
        {"block": "toc"},
    ],
    "tables": {
        "version_control": tbl("version_control"),
        "revision_history": tbl("revision_history"),
        "document_classification": tbl("document_classification"),
    },
    "table_text": {"font": "Calibri", "default_size_pt": 12},
    "toc": {
        "live": True,
        "instr": ' TOC \\h \\u \\z \\t "Heading 1,1,Heading 2,2,Heading 3,3," ',
        "tab_stop_pt": 600,
        "size_pt": 11,
    },
    "heading_numbering": {
        "mechanism": "literal",
        "level1_format": "{n}-{title}",
        "note": "list numbering would renumber the ordinary paragraphs between headings",
    },
    "logo": {
        "mime": "image/png",
        "px": [195, 97],
        "width_pt": 146.25,
        "height_pt": 72.75,
        "anchor": {
            "in": "first_page_header",
            "relative_h": "column", "offset_x_pt": 404.25,
            "relative_v": "paragraph", "offset_y_pt": -5.15,
            "wrap": "square", "wrap_dist_pt": 9,
        },
        "base64": base64.b64encode(LOGO.read_bytes()).decode(),
    },
}

class Dumper(yaml.SafeDumper):
    pass

def str_presenter(dumper, data):
    if len(data) > 120:
        return dumper.represent_scalar("tag:yaml.org,2002:str", data, style="|")
    return dumper.represent_scalar("tag:yaml.org,2002:str", data)

Dumper.add_representer(str, str_presenter)

out = HERE / "house.yaml"
out.write_text(yaml.dump(cfg, Dumper=Dumper, sort_keys=False,
                         allow_unicode=True, width=100))
n = out.stat().st_size
b64 = len(cfg["logo"]["base64"])
print(f"house.yaml {n} bytes; base64 logo {b64} bytes ({100*b64/n:.1f}%); "
      f"without logo {n-b64} bytes")
