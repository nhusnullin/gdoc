from tools.gdoc.model import parse_thread

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
