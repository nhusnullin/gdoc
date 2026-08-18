"""Diffing the exported document against the baseline. Issue #29."""

from gdoc.edits import diff_markdown

BASELINE = """# Routing questions

## 1. Key terms

Scheme means Mastercard.

## 2. Routes

Route one is direct membership.

Route two is a sponsor.
"""


def kinds(hunks):
    return [h.kind for h in hunks]


def test_an_identical_export_has_no_hunks():
    assert diff_markdown(BASELINE, BASELINE) == ()


def test_whitespace_only_differences_are_not_edits():
    """Export round-trips move blank lines around. That is not Nail editing."""
    noisy = BASELINE.replace("\n\n", "\n\n\n").rstrip() + "\n   \n"
    assert diff_markdown(BASELINE, noisy) == ()


def test_a_changed_sentence_is_one_change_hunk():
    current = BASELINE.replace("Scheme means Mastercard.", "Scheme means Mastercard or Visa.")
    hunks = diff_markdown(BASELINE, current)
    assert kinds(hunks) == ["change"]
    assert hunks[0].before == "Scheme means Mastercard."
    assert hunks[0].after == "Scheme means Mastercard or Visa."


def test_a_hunk_carries_the_heading_it_sits_under():
    current = BASELINE.replace("Scheme means Mastercard.", "Scheme means Visa.")
    assert diff_markdown(BASELINE, current)[0].heading == "## 1. Key terms"


def test_a_deleted_paragraph_is_a_delete_hunk():
    current = BASELINE.replace("Route two is a sponsor.\n", "")
    hunks = diff_markdown(BASELINE, current)
    assert kinds(hunks) == ["delete"]
    assert "Route two" in hunks[0].before
    assert hunks[0].after == ""


def test_an_added_paragraph_is_an_insert_hunk():
    current = BASELINE + "\n## 3. Timing\n\nWe decide in September.\n"
    hunks = diff_markdown(BASELINE, current)
    assert kinds(hunks) == ["insert"]
    assert "September" in hunks[0].after
    assert hunks[0].before == ""


def test_a_renamed_heading_is_reported_as_a_change():
    current = BASELINE.replace("## 2. Routes", "## 2. Routes to membership")
    hunks = diff_markdown(BASELINE, current)
    assert kinds(hunks) == ["change"]
    assert hunks[0].after == "## 2. Routes to membership"


def test_an_insert_carries_the_heading_it_landed_under():
    current = BASELINE.replace(
        "Route one is direct membership.",
        "Route one is direct membership.\n\nIt needs capital.",
    )
    assert diff_markdown(BASELINE, current)[0].heading == "## 2. Routes"


def test_several_edits_come_back_separately():
    current = BASELINE.replace("Mastercard.", "Visa.").replace(
        "Route two is a sponsor.\n", ""
    )
    assert len(diff_markdown(BASELINE, current)) == 2


def test_an_empty_baseline_makes_the_whole_document_one_insert():
    hunks = diff_markdown("", BASELINE)
    assert kinds(hunks) == ["insert"]
