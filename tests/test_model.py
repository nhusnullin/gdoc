from gdoc.model import parse_thread

ANCHORED = {
    "id": "AAACFjp837M",
    "content": "ai: rephrase",
    "author": {"displayName": "Nail Khusnullin", "emailAddress": None, "me": False},
    "quotedFileContent": {"value": "asdasd"},
    "resolved": False,
    "replies": [],
}

UNANCHORED = {
    "id": "AAACFjp837Z",
    "content": "ai? does this read as a policy or a procedure",
    "author": {"displayName": "Nail Khusnullin", "me": False},
    "resolved": False,
    "replies": [],
}

ALREADY_ANSWERED = {
    "id": "AAACFjp837I",
    "content": "ai? who you are",
    "author": {"displayName": "Nail Khusnullin", "me": False},
    "quotedFileContent": {"value": "Asdasdasd"},
    "resolved": False,
    "replies": [
        {
            "id": "AAACFjd7zKM",
            "content": "I am Nail AI, a service account.",
            "author": {"displayName": "nail-ai@doc-agent-505414.iam.gserviceaccount.com", "me": True},
        }
    ],
}


def test_parses_an_anchored_comment():
    thread = parse_thread(ANCHORED)
    assert thread.id == "AAACFjp837M"
    assert thread.content == "ai: rephrase"
    assert thread.author_name == "Nail Khusnullin"
    assert thread.quoted == "asdasd"
    assert thread.is_anchored is True


def test_author_email_is_none_and_that_is_expected():
    assert parse_thread(ANCHORED).author_email is None


def test_unanchored_comment_has_no_quote():
    thread = parse_thread(UNANCHORED)
    assert thread.quoted is None
    assert thread.is_anchored is False


def test_detects_an_existing_agent_reply():
    thread = parse_thread(ALREADY_ANSWERED)
    assert thread.has_agent_reply is True
    assert thread.replies[0].by_agent is True


def test_thread_without_replies_has_no_agent_reply():
    assert parse_thread(ANCHORED).has_agent_reply is False


def test_missing_author_block_does_not_crash():
    thread = parse_thread({"id": "x", "content": "ai: hi"})
    assert thread.author_name == "unknown"
    assert thread.by_agent is False


def test_thread_is_immutable():
    thread = parse_thread(ANCHORED)
    try:
        thread.content = "changed"
    except Exception:
        return
    raise AssertionError("Thread should be frozen")


def test_thread_is_hashable_so_it_can_go_in_a_set():
    assert isinstance(hash(parse_thread(ANCHORED)), int)


MARKED_REPLY = {
    "id": "AAACFjp837Q",
    "content": "ai! renumber the annex",
    "author": {"displayName": "William Mejia", "me": False},
    "quotedFileContent": {"value": "Annex 2"},
    "resolved": False,
    "replies": [
        {
            "id": "AAACFjd7zKQ",
            "content": "Captured as item 2.\n\n[gdoc]",
            "author": {"displayName": "Nail Khusnullin", "me": True},
        }
    ],
}


def test_a_reply_ending_in_the_marker_is_gdocs_own():
    thread = parse_thread(MARKED_REPLY)
    assert thread.replies[0].by_marker is True


def test_the_marker_alone_makes_a_thread_answered():
    """Under OAuth `me` is Nail, so the marker has to carry this on its own."""
    raw = {
        **MARKED_REPLY,
        "replies": [
            {
                "id": "r1",
                "content": "Captured as item 2.\n\n[gdoc]",
                "author": {"displayName": "Nail Khusnullin", "me": False},
            }
        ],
    }
    thread = parse_thread(raw)
    assert thread.replies[0].by_agent is False
    assert thread.replies[0].by_marker is True
    assert thread.has_agent_reply is True


def test_me_alone_still_makes_a_thread_answered():
    """Threads the service account answered before the marker existed."""
    assert parse_thread(ALREADY_ANSWERED).has_agent_reply is True


def test_a_reply_mentioning_the_marker_mid_sentence_is_not_gdocs():
    raw = {
        **MARKED_REPLY,
        "replies": [
            {
                "id": "r1",
                "content": "I saw [gdoc] answer this elsewhere",
                "author": {"displayName": "William Mejia", "me": False},
            }
        ],
    }
    thread = parse_thread(raw)
    assert thread.replies[0].by_marker is False
    assert thread.has_agent_reply is False


def test_replies_carry_their_author_name():
    assert parse_thread(MARKED_REPLY).replies[0].author_name == "Nail Khusnullin"


def test_a_reply_with_no_author_block_is_unknown():
    raw = {**MARKED_REPLY, "replies": [{"id": "r1", "content": "hi"}]}
    assert parse_thread(raw).replies[0].author_name == "unknown"
