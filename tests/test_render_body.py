"""Heading numbering: the author's own numbers win over the computed ones.

Issue #27. A heading that numbers itself is the same problem as a heading that
names itself: "1. Key terms" must not publish as "1-1. Key terms".
"""

import pytest

from gdoc.render.body import HeadingNumberer


def numberer():
    return HeadingNumberer(enabled=True, top_level=1)


# ------------------------------------------------ headings that number themselves --
@pytest.mark.parametrize(
    "heading",
    [
        "1. Key terms",
        "2. Second",
        "1) Item",
        "3.1 Route 1: Altery's direct principal membership",
        "3.1. Route 1",
        "4.2.1 Something deeper",
        "  1. Leading space still counts",
    ],
)
def test_a_heading_that_numbers_itself_gets_no_computed_prefix(heading):
    assert numberer().prefix(1, heading) == ""


def test_a_bare_number_with_nothing_after_it_still_counts():
    assert numberer().prefix(1, "1.") == ""
    assert numberer().prefix(1, "3.1") == ""


# ------------------------------------------------------------- counter syncing --
def test_a_hand_numbered_heading_moves_the_counter_to_its_own_number():
    """Otherwise the author's 2. is followed by a computed 2-, two headings numbered 2."""
    n = numberer()
    assert n.prefix(1, "Introduction") == "1-"
    assert n.prefix(1, "4. Hand numbered") == ""
    assert n.prefix(1, "Next") == "5-"


def test_a_hand_numbered_parent_sets_the_level_its_children_count_from():
    """Without this "## 4. Controls" is followed by "1.1-Scope"."""
    n = numberer()
    assert n.prefix(1, "4. Controls") == ""
    assert n.prefix(2, "Scope") == "4.1-"


def test_a_hand_numbered_child_syncs_its_own_level():
    n = numberer()
    n.prefix(1, "Introduction")
    assert n.prefix(2, "3.1 Route one") == ""
    assert n.prefix(2, "Route two") == "3.2-"


def test_a_deeper_counter_is_cleared_by_a_hand_numbered_parent():
    n = numberer()
    n.prefix(1, "One")
    n.prefix(2, "One point one")
    assert n.prefix(1, "7. Seven") == ""
    assert n.prefix(2, "Child") == "7.1-"


# ------------------------------------------------------ numbers that are not sections --
def test_a_number_whose_depth_disagrees_is_suppressed_but_syncs_nothing():
    """"3.1 GHz band" as a top-level heading is a frequency, not a section number.

    It cannot be told apart from a section number, so it is left alone rather
    than doubled, exactly as "Appendix 2" is. What it must not do is move the
    counter, because 3.1 is not where this document is.
    """
    n = numberer()
    assert n.prefix(1, "3.1 GHz band") == ""
    assert n.prefix(1, "Next") == "1-"


# --------------------------------------------- headings that only start with digits --
@pytest.mark.parametrize(
    "heading",
    [
        "2026 plan",
        "1990s in review",
        "10 things we learned",
        "1.5x throughput",
    ],
)
def test_a_heading_that_merely_starts_with_a_digit_is_still_numbered(heading):
    assert numberer().prefix(1, heading) == "1-"


# ------------------------------------------------------- the existing rules hold --
def test_a_heading_that_names_itself_is_still_unnumbered():
    assert numberer().prefix(1, "Appendix 2") == ""


def test_an_ordinary_heading_is_still_numbered():
    assert numberer().prefix(1, "Purpose") == "1-"


def test_numbering_off_suppresses_everything():
    assert HeadingNumberer(enabled=False).prefix(1, "Purpose") == ""


# ------------------------------------------------------------------- pictures --
# Issue #28. A picture that lives at a URL cannot be embedded, and the message
# has to name the way to get one that can.

import pytest  # noqa: E402

from gdoc.render import build  # noqa: E402
from gdoc.render.body import BodyError  # noqa: E402


def _note(tmp_path, body):
    path = tmp_path / "note.md"
    path.write_text(
        "---\ntitle: Picture test\n---\n\n# Picture test\n\n## The picture\n\n" + body
    )
    return path


def test_a_remote_picture_names_the_command_that_makes_it_local(tmp_path):
    note = _note(tmp_path, "![](https://lh7-us.googleusercontent.com/abc)\n")
    with pytest.raises(BodyError) as caught:
        build(note, tmp_path / "out.docx")
    message = str(caught.value)
    assert "gdoc export" in message
    assert "--media-dir" in message


def test_a_local_picture_beside_the_note_is_embedded(tmp_path):
    import zipfile

    png = _one_pixel_png()
    (tmp_path / "note-media").mkdir()
    (tmp_path / "note-media" / "image1.png").write_bytes(png)
    note = _note(tmp_path, "![](note-media/image1.png)\n")
    out = tmp_path / "out.docx"
    build(note, out)
    assert any(n.startswith("word/media/") for n in zipfile.ZipFile(out).namelist())


def _one_pixel_png() -> bytes:
    """A real 1x1 PNG. python-docx reads the header, so a fake one will not do."""
    import struct
    import zlib

    def chunk(kind, payload):
        return (
            struct.pack(">I", len(payload))
            + kind
            + payload
            + struct.pack(">I", zlib.crc32(kind + payload) & 0xFFFFFFFF)
        )

    header = struct.pack(">IIBBBBB", 1, 1, 8, 2, 0, 0, 0)
    pixels = zlib.compress(b"\x00\xff\xff\xff")
    return b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", header) + chunk(b"IDAT", pixels) + chunk(b"IEND", b"")


def test_pictures_inside_a_sentence_are_still_embedded(tmp_path):
    """The real case from issue #28: a pulled document puts diagrams at the end
    of the paragraph that introduces them, so `lone_image` never saw them and
    two of three drawings were dropped without a word.
    """
    import zipfile

    media = tmp_path / "note-media"
    media.mkdir()
    for name in ("image1.png", "image2.png"):
        (media / name).write_bytes(_one_pixel_png())
    note = _note(
        tmp_path,
        "Non-cash incentives follow a request model."
        "![](note-media/image1.png)![](note-media/image2.png)\n",
    )
    out = tmp_path / "out.docx"
    build(note, out)
    document = zipfile.ZipFile(out).read("word/document.xml").decode("utf8")
    assert document.count("<w:drawing") == 2


def test_the_sentence_beside_a_picture_survives(tmp_path):
    import re
    import zipfile

    media = tmp_path / "note-media"
    media.mkdir()
    (media / "image1.png").write_bytes(_one_pixel_png())
    note = _note(tmp_path, "Route one is direct.![](note-media/image1.png)\n")
    out = tmp_path / "out.docx"
    build(note, out)
    document = zipfile.ZipFile(out).read("word/document.xml").decode("utf8")
    # Runs are split at every space, so read the text back out of them.
    words = re.sub(r"\s+", " ", "".join(re.findall(r"<w:t[^>]*>([^<]*)</w:t>", document)))
    assert "Route one is direct." in words
    assert document.count("<w:drawing") == 1
