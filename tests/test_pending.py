import pytest

from gdoc.model import Thread
from gdoc.pending import append_item, next_item_number, pending_path, recorded_source

TODAY = "2026-08-13"


def thread(id="t1", content="ai! renumber the sections", quoted="Section 4"):
    return Thread(
        id=id,
        content=content,
        author_name="Nail Khusnullin",
        author_email=None,
        by_agent=False,
        quoted=quoted,
        resolved=False,
        replies=(),
    )


def test_pending_path_sits_beside_the_baseline(tmp_path):
    assert pending_path(tmp_path, "policy") == tmp_path / ".gdoc" / "policy" / "pending.md"


def test_first_item_is_number_one(tmp_path):
    assert next_item_number(pending_path(tmp_path, "policy")) == 1


def test_append_creates_the_file_with_a_heading(tmp_path):
    number = append_item(tmp_path, "policy", thread(), doc_id="1AbC", today=TODAY)
    text = pending_path(tmp_path, "policy").read_text()
    assert number == 1
    assert text.startswith("# Pending global items")
    assert "1AbC" in text


def test_item_records_the_comment_id_text_quote_and_date(tmp_path):
    append_item(tmp_path, "policy", thread(), doc_id="1AbC", today=TODAY)
    text = pending_path(tmp_path, "policy").read_text()
    assert "## Item 1" in text
    assert "ai! renumber the sections" in text
    assert "Section 4" in text
    assert TODAY in text
    assert "t1" in text


def test_second_append_numbers_two_and_keeps_the_first(tmp_path):
    append_item(tmp_path, "policy", thread(id="t1"), doc_id="1AbC", today=TODAY)
    number = append_item(
        tmp_path, "policy", thread(id="t2", content="ai! split section 3"), doc_id="1AbC", today=TODAY
    )
    text = pending_path(tmp_path, "policy").read_text()
    assert number == 2
    assert "## Item 1" in text and "## Item 2" in text


def test_unanchored_thread_says_whole_document(tmp_path):
    append_item(tmp_path, "policy", thread(quoted=None), doc_id="1AbC", today=TODAY)
    assert "whole document" in pending_path(tmp_path, "policy").read_text()


def test_appending_the_same_comment_twice_is_refused(tmp_path):
    append_item(tmp_path, "policy", thread(id="t1"), doc_id="1AbC", today=TODAY)
    with pytest.raises(ValueError, match="already captured"):
        append_item(tmp_path, "policy", thread(id="t1"), doc_id="1AbC", today=TODAY)


def test_multiline_comment_every_line_gets_blockquote_prefix(tmp_path):
    multiline_thread = thread(content="line one\nline two")
    append_item(tmp_path, "policy", multiline_thread, doc_id="1AbC", today=TODAY)
    text = pending_path(tmp_path, "policy").read_text()
    assert "> line one" in text
    assert "> line two" in text


# ---------------------------------------------------------------------------
# the source file the queue belongs to
# ---------------------------------------------------------------------------


def test_the_header_records_the_source_file(tmp_path):
    append_item(
        tmp_path, "policy", thread(), doc_id="1AbC", today=TODAY, source="notes/policy.md"
    )
    assert "Source: notes/policy.md" in pending_path(tmp_path, "policy").read_text()


def test_recorded_source_reads_it_back(tmp_path):
    append_item(
        tmp_path, "policy", thread(), doc_id="1AbC", today=TODAY, source="notes/policy.md"
    )
    assert recorded_source(pending_path(tmp_path, "policy")) == "notes/policy.md"


def test_recorded_source_is_none_for_an_older_queue(tmp_path):
    """Queues written before the Source line must still be readable."""
    append_item(tmp_path, "policy", thread(), doc_id="1AbC", today=TODAY)
    path = pending_path(tmp_path, "policy")
    assert "Source:" not in path.read_text()
    assert recorded_source(path) is None


def test_recorded_source_is_none_when_there_is_no_queue(tmp_path):
    assert recorded_source(pending_path(tmp_path, "policy")) is None
