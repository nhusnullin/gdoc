"""Resolving which page each heading landed on, from rendered page text."""

import pytest

from gdoc.render import pagination

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
