from gdoc.filters import forced_kind, needs_action, partition
from gdoc.model import Reply, Thread, parse_thread

ME = "Nail Khusnullin"


def thread(**kwargs):
    defaults = dict(
        id="t1",
        content="ai: rephrase",
        author_name=ME,
        author_email=None,
        by_agent=False,
        quoted="some text",
        resolved=False,
        replies=(),
    )
    return Thread(**{**defaults, **kwargs})


def test_marker_is_required():
    assert needs_action(thread(content="just a note to myself")) is False


def test_marker_is_case_insensitive():
    assert needs_action(thread(content="AI: rephrase")) is True


def test_marker_must_start_the_comment():
    assert needs_action(thread(content="I think ai: should handle this")) is False


def test_bare_ai_without_punctuation_is_not_a_request():
    """Otherwise every comment starting with a sentence about AI becomes a job."""
    assert needs_action(thread(content="AI is going to change how we write these")) is False


def test_all_three_signs_are_markers():
    assert needs_action(thread(content="ai: rephrase")) is True
    assert needs_action(thread(content="ai? is this right")) is True
    assert needs_action(thread(content="ai! renumber the sections")) is True


def test_the_old_at_form_still_works():
    """Nail used @ai first. Ignoring it silently would be the worst outcome."""
    assert needs_action(thread(content="@ai rephrase")) is True
    assert needs_action(thread(content="@ai: rephrase")) is True


def test_leading_whitespace_is_tolerated():
    assert needs_action(thread(content="  ai: rephrase")) is True


def test_resolved_threads_are_skipped():
    assert needs_action(thread(resolved=True)) is False


def test_a_marked_comment_the_credential_wrote_is_still_work():
    """The regression this whole change turns on.

    Under OAuth the credential is Nail, so by_agent is true for every comment he
    writes. If it still gated needs_action, gdoc read would return nothing.
    """
    assert needs_action(thread(by_agent=True)) is True


def test_a_marker_reply_counts_as_answered_even_without_me():
    answered = thread(
        replies=(Reply(id="r1", content="Done.\n\n[gdoc]", by_agent=False, by_marker=True),)
    )
    assert needs_action(answered) is False


def test_threads_the_agent_already_answered_are_skipped():
    answered = thread(replies=(Reply(id="r1", content="done", by_agent=True),))
    assert needs_action(answered) is False


def test_a_human_reply_does_not_count_as_answered():
    discussed = thread(replies=(Reply(id="r1", content="good point", by_agent=False),))
    assert needs_action(discussed) is True


def test_question_override():
    assert forced_kind("ai? is this right") == "question"


def test_instruction_override():
    assert forced_kind("ai! renumber everything") == "instruction"


def test_colon_leaves_the_decision_to_the_agent():
    assert forced_kind("ai: rephrase this") is None


def test_old_at_form_overrides_too():
    assert forced_kind("@ai? is this right") == "question"
    assert forced_kind("@ai rephrase this") is None


def test_a_marked_comment_from_anyone_is_actionable():
    """The marker is the instruction. Who typed it does not change the work."""
    threads = (thread(author_name="Nail Khusnullin"), thread(author_name="William Mejia"))
    addressed, skipped = partition(threads)
    assert len(addressed) == 2
    assert skipped == ()


def test_an_unmarked_comment_is_still_skipped():
    addressed, skipped = partition((thread(content="Looks fine to me"),))
    assert addressed == ()
    assert len(skipped) == 1


def test_an_already_answered_comment_is_still_skipped():
    """Dropping the author check must not weaken the idempotence guard."""
    answered = thread(replies=(Reply(id="r1", content="done", by_agent=True),))
    addressed, skipped = partition((answered,))
    assert addressed == ()


def test_partition_returns_tuples():
    addressed, skipped = partition((thread(), thread(resolved=True)))
    assert isinstance(addressed, tuple) and isinstance(skipped, tuple)


def test_all_mode_takes_unmarked_comments_too():
    threads = (thread(content="ai: rephrase"), thread(content="This reads oddly"))
    addressed, skipped = partition(threads, include_unmarked=True)
    assert len(addressed) == 2
    assert skipped == ()


def test_all_mode_still_leaves_resolved_threads_alone():
    addressed, skipped = partition((thread(resolved=True),), include_unmarked=True)
    assert addressed == ()
    assert len(skipped) == 1


def test_all_mode_shows_a_thread_gdoc_already_answered():
    """has_agent_reply counts a `me` reply, and under OAuth `me` is Nail.

    Hiding answered threads here would hide every thread he replied to by hand,
    which is the opposite of what all-comments mode is for.
    """
    answered = thread(
        replies=(Reply(id="r1", content="Done.\n\n[gdoc]", by_agent=False, by_marker=True),)
    )
    addressed, _ = partition((answered,), include_unmarked=True)
    assert len(addressed) == 1


def test_default_mode_is_unchanged_by_the_new_argument():
    threads = (thread(content="ai: rephrase"), thread(content="This reads oddly"))
    assert partition(threads) == partition(threads, include_unmarked=False)


def test_a_reply_nail_typed_himself_does_not_count_as_gdocs():
    """Under oauth, Drive marks Nail's own reply `me`.

    Reading that as gdoc would drop the thread into `skipped` on every run,
    forever. The legacy argument for keeping `me` does not rescue it either: a
    thread the service account answered has `me` false when Nail's token reads
    it, so under oauth `me` buys nothing and costs real threads.

    The fix lives in parse_thread, which is why this goes through it: by_agent
    now means "gdoc wrote it", not "the credential wrote it".
    """
    raw = {
        "id": "t1",
        "content": "ai? does this need a Consumer Duty ref",
        "author": {"displayName": "Nail Khusnullin", "me": True},
        "resolved": False,
        "replies": [
            {
                "id": "r1",
                "content": "and check the FCA handbook ref",
                "author": {"displayName": "Nail Khusnullin", "me": True},
            }
        ],
    }
    assert needs_action(parse_thread(raw, me_is_agent=False)) is True
    # Under the service account the same shape does mean gdoc answered.
    assert needs_action(parse_thread(raw, me_is_agent=True)) is False
