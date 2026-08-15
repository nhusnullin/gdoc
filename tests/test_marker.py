from gdoc.marker import MARKER, has_marker, with_marker


def test_the_marker_is_exactly_this():
    assert MARKER == "[gdoc]"


def test_marker_goes_on_its_own_last_line():
    assert with_marker("Use safeguarded funds.") == "Use safeguarded funds.\n\n[gdoc]"


def test_trailing_whitespace_does_not_produce_a_gap():
    assert with_marker("Use safeguarded funds.\n\n\n") == "Use safeguarded funds.\n\n[gdoc]"


def test_a_body_that_already_ends_with_the_marker_is_left_alone():
    once = with_marker("Answer.")
    assert with_marker(once) == once


def test_multiline_bodies_keep_their_shape():
    body = "Answer.\n\nSources: domains/regulatory/cbc-emi.md"
    assert with_marker(body) == body + "\n\n[gdoc]"


def test_has_marker_finds_it_as_the_last_line():
    assert has_marker("Answer.\n\n[gdoc]") is True


def test_has_marker_ignores_trailing_blank_lines():
    assert has_marker("Answer.\n\n[gdoc]\n\n") is True


def test_a_mid_sentence_mention_is_not_the_marker():
    assert has_marker("I ran [gdoc] on this and it worked") is False


def test_a_marker_that_is_not_last_does_not_count():
    assert has_marker("[gdoc]\n\nAnswer.") is False


def test_empty_text_has_no_marker():
    assert has_marker("") is False
    assert has_marker(None) is False
