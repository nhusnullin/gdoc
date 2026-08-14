"""Resolving which page each heading landed on, from rendered page text."""

from io import BytesIO

import pytest

from gdoc.render import pagination


def minimal_pdf(pages):
    """Build a small PDF with one text line per entry, per page.

    Hand-built so the suite needs no binary fixture and no PDF writer. Standard
    Helvetica, uncompressed streams, which is all pypdf needs to extract text.
    """
    objects = [None, None, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"]
    font_id, page_ids = 3, []

    for lines in pages:
        stream_lines = ["BT", "/F1 12 Tf", "1 0 0 1 50 750 Tm", "14 TL"]
        for line in lines:
            escaped = line.replace("\\", r"\\").replace("(", r"\(").replace(")", r"\)")
            stream_lines += [f"({escaped}) Tj", "T*"]
        stream_lines.append("ET")
        stream = "\n".join(stream_lines)
        content_id = len(objects) + 1
        objects.append(f"<< /Length {len(stream)} >>\nstream\n{stream}\nendstream")
        page_ids.append(len(objects) + 1)
        objects.append(
            f"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] "
            f"/Resources << /Font << /F1 {font_id} 0 R >> >> "
            f"/Contents {content_id} 0 R >>"
        )

    kids = " ".join(f"{pid} 0 R" for pid in page_ids)
    objects[0] = "<< /Type /Catalog /Pages 2 0 R >>"
    objects[1] = f"<< /Type /Pages /Count {len(page_ids)} /Kids [{kids}] >>"

    out = BytesIO()
    out.write(b"%PDF-1.4\n")
    offsets = []
    for number, body in enumerate(objects, start=1):
        offsets.append(out.tell())
        out.write(f"{number} 0 obj\n{body}\nendobj\n".encode("latin-1"))
    xref_at = out.tell()
    out.write(f"xref\n0 {len(objects) + 1}\n".encode())
    out.write(b"0000000000 65535 f \n")
    for offset in offsets:
        out.write(f"{offset:010d} 00000 n \n".encode())
    out.write(f"trailer\n<< /Size {len(objects) + 1} /Root 1 0 R >>\n"
              f"startxref\n{xref_at}\n%%EOF\n".encode())
    return out.getvalue()


@pytest.fixture
def sample_pdf(tmp_path):
    """Three pages: a contents list, then a body page, then an appendix page.

    Page 2 mentions "Appendices" in prose and page 3 carries it as a heading. That
    is the case that made a substring match put two entries on the wrong page.
    """
    path = tmp_path / "sample.pdf"
    path.write_bytes(minimal_pdf([
        ["Contents", "1-Purpose 4", "Appendices 5"],
        ["1-Purpose", "Body text mentioning Appendices in prose"],
        ["Appendices", "Appendix 1 - Associated Documents"],
    ]))
    return path


def test_page_count_reads_the_pdf(sample_pdf):
    assert pagination.page_count(sample_pdf) == 3


def test_page_lines_returns_the_text_of_each_page(sample_pdf):
    lines = pagination.page_lines(sample_pdf)
    assert len(lines) == 3
    assert "Contents" in lines[0]
    assert "1-Purpose" in lines[1]


def test_from_pdf_resolves_headings_against_a_real_pdf(sample_pdf):
    found = pagination.from_pdf(sample_pdf, ((1, "1-Purpose"), (1, "Appendices")))
    assert found == {"1-Purpose": 2, "Appendices": 3}


def test_no_external_program_is_used(sample_pdf, monkeypatch):
    """Reading a PDF must not shell out. That is the whole point of the change."""
    import subprocess

    def refuse(*args, **kwargs):
        raise AssertionError("pagination shelled out to an external program")

    monkeypatch.setattr(subprocess, "run", refuse)
    monkeypatch.setattr(subprocess, "Popen", refuse)
    assert pagination.page_count(sample_pdf) == 3
    assert len(pagination.page_lines(sample_pdf)) == 3


def test_an_unreadable_file_is_reported_clearly(tmp_path):
    broken = tmp_path / "broken.pdf"
    broken.write_bytes(b"this is not a pdf")
    with pytest.raises(pagination.PaginationError):
        pagination.page_count(broken)


HEADINGS = (
    (1, "1-Purpose"),
    (2, "1.1-Policy Owner"),
    (1, "Appendices"),
    (2, "Appendix 1 - Associated Documents"),
)

# Page 1 cover, page 2 control tables, page 3 the contents list, then the body.
PAGES = [
    ["Third Party and Outsourcing Policy"],
    ["Version Control", "Document Owner", "Head of Procurement"],
    ["Contents", "1-Purpose 4", "1.1-Policy Owner 4", "Appendices 5",
     "Appendix 1 - Associated Documents 5"],
    ["1-Purpose", "The Altery Group refers to all the legal entities",
     "1.1-Policy Owner", "The owner is accountable.",
     "The document is divided into Appendices and Addenda"],
    ["Appendices", "Appendix 1 - Associated Documents", "Group Corporate Governance Policy"],
]


def test_each_heading_resolves_to_the_page_holding_its_body_heading():
    found = pagination.resolve(PAGES, HEADINGS)
    assert found == {
        "1-Purpose": 4,
        "1.1-Policy Owner": 4,
        "Appendices": 5,
        "Appendix 1 - Associated Documents": 5,
    }


def test_the_contents_page_itself_is_never_the_answer():
    """Every heading appears on the contents page too. It must be skipped."""
    found = pagination.resolve(PAGES, HEADINGS)
    assert 3 not in found.values()


def test_a_heading_mentioned_in_prose_does_not_win():
    """"Appendices" appears inside a sentence on page 4 and as a heading on page 5.

    A substring match lands it on page 4, which is wrong. This is not theoretical:
    it happened, and it put two entries on the wrong page.
    """
    found = pagination.resolve(PAGES, HEADINGS)
    assert found["Appendices"] == 5


def test_a_heading_that_never_appears_is_left_out_rather_than_guessed():
    found = pagination.resolve(PAGES, HEADINGS + ((1, "9-Nowhere"),))
    assert "9-Nowhere" not in found


def test_a_wrapped_heading_still_resolves():
    """pdftotext can break a long heading across lines, so allow a prefix match."""
    long = "Addendum 1 - Variation for Altery Connect El Salvador and its branches"
    pages = PAGES + [[long[:45], long[45:]]]
    found = pagination.resolve(pages, ((2, long),))
    assert found[long] == 6


def test_whitespace_differences_do_not_matter():
    pages = [["Contents"], ["1-Purpose"], ["  1.1-Policy   Owner  "]]
    found = pagination.resolve(pages, ((2, "1.1-Policy Owner"),))
    assert found["1.1-Policy Owner"] == 3


def test_drift_reports_what_moved():
    written = {"1-Purpose": 4, "Appendices": 5}
    published = {"1-Purpose": 4, "Appendices": 6}
    assert pagination.drift(written, published) == {"Appendices": (5, 6)}


def test_no_drift_is_an_empty_report():
    same = {"1-Purpose": 4}
    assert pagination.drift(same, dict(same)) == {}


def test_pages_without_a_contents_page_start_from_the_beginning():
    pages = [["1-Purpose"], ["1.1-Policy Owner"]]
    found = pagination.resolve(pages, ((1, "1-Purpose"),))
    assert found["1-Purpose"] == 1
