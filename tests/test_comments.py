"""Copy the open comment threads of one document onto another.

The anchors cannot come across, and neither can the authors: Drive writes every
comment as the authenticated user. So the thread's own text has to carry both,
and these tests say what that text looks like.
"""

from unittest.mock import MagicMock

from gdoc.comments import copy_comments, thread_body
from gdoc.model import Reply, Thread


def _thread(
    id="c1",
    content="this number is wrong",
    author="William Mejia",
    quoted=None,
    resolved=False,
    replies=(),
):
    return Thread(
        id=id,
        content=content,
        author_name=author,
        author_email=None,
        by_agent=False,
        quoted=quoted,
        resolved=resolved,
        replies=tuple(replies),
    )


def _reply(content, author):
    return Reply(id="r", content=content, by_agent=False, author_name=author)


def _drive(new_ids=("n1", "n2", "n3")):
    drive = MagicMock()
    drive.comments().create.return_value.execute.side_effect = [
        {"id": each} for each in new_ids
    ]
    return drive


# ----------------------------------------------------------------- the body --


def test_a_lone_comment_carries_its_author_name():
    body = thread_body(_thread())
    assert body == "William Mejia: this number is wrong"


def test_every_reply_keeps_its_own_author_in_order():
    thread = _thread(
        replies=[_reply("fixed in the table", "Anna"), _reply("thanks", "Nail")]
    )
    assert thread_body(thread) == (
        "William Mejia: this number is wrong\n"
        "Anna: fixed in the table\n"
        "Nail: thanks"
    )


def test_a_quoted_thread_says_what_it_pointed_at():
    """The anchor cannot survive the new template, so the quote is the pointer."""
    body = thread_body(_thread(quoted="the fee is 2%"))
    assert body.startswith('On "the fee is 2%":\n\n')
    assert body.endswith("William Mejia: this number is wrong")


def test_an_unquoted_thread_gets_no_quote_line():
    assert "On " not in thread_body(_thread(quoted=None))


def test_markdown_in_somebody_elses_comment_is_carried_unchanged():
    """reply.assert_plain_text guards what gdoc writes, not what it carries."""
    body = thread_body(_thread(content="rename it to **client_money**"))
    assert "**client_money**" in body


def test_no_gdoc_marker_because_these_are_not_gdocs_words():
    assert "[gdoc]" not in thread_body(_thread(replies=[_reply("ok", "Anna")]))


# ---------------------------------------------------------------- the copy --


def test_open_threads_are_created_on_the_target_document():
    drive = _drive()
    result = copy_comments(drive, [_thread(id="a"), _thread(id="b")], "1NewDoc")
    assert result.copied == 2
    assert drive.comments().create.call_count == 2
    assert drive.comments().create.call_args.kwargs["fileId"] == "1NewDoc"


def test_a_created_comment_names_no_anchor():
    """There is nothing to anchor to: the new document has different structure."""
    drive = _drive()
    copy_comments(drive, [_thread()], "1NewDoc")
    body = drive.comments().create.call_args.kwargs["body"]
    assert "anchor" not in body
    assert body["content"] == "William Mejia: this number is wrong"


def test_resolved_threads_are_dropped_and_counted():
    drive = _drive()
    result = copy_comments(
        drive, [_thread(id="a"), _thread(id="b", resolved=True)], "1NewDoc"
    )
    assert result.copied == 1
    assert result.skipped_resolved == 1
    assert drive.comments().create.call_count == 1


def test_one_failing_thread_does_not_stop_the_next():
    """The document already exists. Dying here would hide it."""
    drive = MagicMock()
    drive.comments().create.return_value.execute.side_effect = [
        RuntimeError("boom"),
        {"id": "n2"},
    ]
    result = copy_comments(drive, [_thread(id="a"), _thread(id="b")], "1NewDoc")
    assert result.copied == 1
    assert len(result.errors) == 1
    assert "boom" in result.errors[0]
    assert "a" in result.errors[0]


def test_nothing_to_copy_is_not_an_error():
    drive = _drive()
    result = copy_comments(drive, [], "1NewDoc")
    assert result.copied == 0
    assert result.errors == ()
    drive.comments().create.assert_not_called()
