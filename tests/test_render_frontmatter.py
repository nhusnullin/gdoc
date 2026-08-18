"""The title contract: refuse a note with no title, and suggest one.

The suggestion is carried, never used. Inventing a cover title silently is the
one thing this contract exists to prevent: the skill proposes the candidate,
Nail approves it, and it is written into the note once.

Task 4 owns the rest of this contract: the filename rules, code fences, --title
on gdoc build, and letting a file with no front matter at all through.
"""

import pytest

from gdoc.render import frontmatter


def test_a_title_only_document_parses():
    meta, body = frontmatter.parse("---\ntitle: Something\n---\n\nBody.\n")
    assert meta["cover_title"] == "Something"
    assert body.strip() == "Body."


def test_a_missing_title_suggests_the_first_h1():
    text = "---\ngdoc: 1abc\n---\n\n# Miguel kickoff call, 2026-08-12\n\nBody.\n"
    with pytest.raises(frontmatter.MissingTitle) as excinfo:
        frontmatter.parse(text, source_name="2026-08-12-miguel-kickoff-call.md")
    assert excinfo.value.candidate == "Miguel kickoff call, 2026-08-12"
    assert excinfo.value.source == "h1"


def test_a_missing_title_falls_back_to_the_filename():
    text = "---\ngdoc: 1abc\n---\n\nJust prose, no heading.\n"
    with pytest.raises(frontmatter.MissingTitle) as excinfo:
        frontmatter.parse(text, source_name="2026-08-12-miguel-kickoff-call.md")
    assert excinfo.value.candidate == "Miguel kickoff call"
    assert excinfo.value.source == "filename"


def test_a_missing_title_is_still_a_front_matter_error():
    """Callers that only know the base class must keep catching it."""
    with pytest.raises(frontmatter.FrontMatterError):
        frontmatter.parse("---\ngdoc: 1abc\n---\n\nBody.\n", source_name="x.md")


def test_the_title_override_wins_and_nothing_is_raised():
    text = "---\ngdoc: 1abc\n---\n\n# Ignored\n\nBody.\n"
    meta, _ = frontmatter.parse(text, title_override="Explicit Title", source_name="x.md")
    assert meta["cover_title"] == "Explicit Title"


def test_a_file_with_no_front_matter_at_all_is_still_its_own_error():
    """Not a MissingTitle. Task 4 decides whether that changes."""
    with pytest.raises(frontmatter.FrontMatterError) as excinfo:
        frontmatter.parse("# Kickoff call\n\nBody.\n", source_name="kickoff.md")
    assert not isinstance(excinfo.value, frontmatter.MissingTitle)
    assert "front matter" in str(excinfo.value)


def test_the_no_front_matter_error_carries_the_fields_rather_than_a_file_to_go_and_find():
    """A reader must be able to fix the note from the message alone.

    The message used to end 'See references/front-matter.md for the full field
    list', and no such file has ever existed in this repo. An agent that took
    the sentence at its word searched the whole disk for two minutes and found
    nothing. So the message carries the answer instead of a pointer to it: the
    one required field, and the optional ones by name.
    """
    with pytest.raises(frontmatter.FrontMatterError) as excinfo:
        frontmatter.parse("# Kickoff call\n\nBody.\n", source_name="kickoff.md")
    message = str(excinfo.value)
    assert "references/front-matter.md" not in message
    assert "title" in message
    for optional in ("classification", "owner", "doc_type", "revisions"):
        assert optional in message, f"{optional} is not named in the message"


def test_an_unrecognised_classification_is_still_refused():
    text = "---\ntitle: X\nclassification: Secret\n---\n\nBody.\n"
    with pytest.raises(frontmatter.FrontMatterError) as excinfo:
        frontmatter.parse(text)
    assert "Secret" in str(excinfo.value)
