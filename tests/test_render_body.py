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


def test_a_hand_numbered_heading_does_not_advance_the_counter():
    """A skipped heading must not consume a number, or the next one jumps."""
    n = numberer()
    n.prefix(1, "Introduction")          # 1-
    n.prefix(1, "2. Hand numbered")      # suppressed
    assert n.prefix(1, "Next") == "2-"


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
