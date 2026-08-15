from unittest.mock import MagicMock

import pytest

from gdoc.reply import GLOBAL_REFUSAL, assert_plain_text, post_reply


def test_plain_prose_is_accepted():
    body = "Use safeguarded funds, not client money.\n\nReason: it matches the policy."
    assert assert_plain_text(body) == body


def test_bold_markers_are_rejected():
    with pytest.raises(ValueError, match="markdown"):
        assert_plain_text("Use **safeguarded funds** here.")


def test_backticks_are_rejected():
    with pytest.raises(ValueError, match="markdown"):
        assert_plain_text("Rename it to `client_money`.")


def test_heading_markers_are_rejected():
    with pytest.raises(ValueError, match="markdown"):
        assert_plain_text("## Proposed wording\n\nSomething.")


def test_hyphen_lists_are_allowed_because_they_read_fine_as_text():
    body = "Two options:\n- keep the clause\n- move it to the SOP"
    assert assert_plain_text(body) == body


def test_empty_body_is_rejected():
    with pytest.raises(ValueError, match="empty"):
        assert_plain_text("   ")


def test_post_reply_returns_the_new_reply_id():
    drive = MagicMock()
    drive.replies().create.return_value.execute.return_value = {"id": "AAACFjd7zKM"}
    assert post_reply(drive, "doc", "comment", "plain text") == "AAACFjd7zKM"


def test_post_reply_refuses_markdown_before_calling_the_api():
    drive = MagicMock()
    with pytest.raises(ValueError):
        post_reply(drive, "doc", "comment", "has **markdown**")
    drive.replies().create.assert_not_called()


def test_global_refusal_names_the_item_number():
    assert "item 2" in GLOBAL_REFUSAL.format(item=2)


def test_global_refusal_is_itself_plain_text():
    assert assert_plain_text(GLOBAL_REFUSAL.format(item=1))


def test_the_posted_body_carries_the_marker():
    drive = MagicMock()
    drive.replies().create.return_value.execute.return_value = {"id": "r1"}
    post_reply(drive, "doc", "comment", "Plain text answer.")
    posted = drive.replies().create.call_args.kwargs["body"]["content"]
    assert posted == "Plain text answer.\n\n[gdoc]"


def test_the_marker_itself_passes_the_markdown_check():
    """Otherwise the tool would refuse its own replies."""
    from gdoc.marker import with_marker

    assert assert_plain_text(with_marker("Plain text answer."))


def test_the_refusal_text_also_carries_the_marker_once_posted():
    drive = MagicMock()
    drive.replies().create.return_value.execute.return_value = {"id": "r1"}
    post_reply(drive, "doc", "comment", GLOBAL_REFUSAL.format(item=2))
    posted = drive.replies().create.call_args.kwargs["body"]["content"]
    assert posted.endswith("\n\n[gdoc]")
    assert posted.count("[gdoc]") == 1
