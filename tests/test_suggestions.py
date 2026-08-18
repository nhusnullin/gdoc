"""Reading pending suggestions out of the Docs API. Issue #29.

Drive's markdown export renders a suggested document as if no suggestion
existed, so a document reviewed entirely in suggesting mode diffs to nothing.
This is the only way to see that work.
"""

from gdoc.suggestions import parse_suggestions


def run(text, insertion=None, deletion=None):
    element = {"textRun": {"content": text}}
    if insertion:
        element["textRun"]["suggestedInsertionIds"] = [insertion]
    if deletion:
        element["textRun"]["suggestedDeletionIds"] = [deletion]
    return element


def paragraph(*elements, style="NORMAL_TEXT"):
    return {"paragraph": {"elements": list(elements), "paragraphStyle": {"namedStyleType": style}}}


def document(*content):
    return {"body": {"content": list(content)}}


def test_a_clean_document_has_no_suggestions():
    assert parse_suggestions(document(paragraph(run("Plain text.")))) == ()


def test_an_inserted_run_is_a_suggestion():
    doc = document(paragraph(run("Scheme means "), run("Visa or ", insertion="s1"), run("Mastercard.")))
    found = parse_suggestions(doc)
    assert [s.kind for s in found] == ["insertion"]
    assert found[0].text == "Visa or"


def test_a_deleted_run_is_a_suggestion():
    doc = document(paragraph(run("Scheme means "), run("only ", deletion="s2"), run("Mastercard.")))
    found = parse_suggestions(doc)
    assert [s.kind for s in found] == ["deletion"]
    assert found[0].text == "only"


def test_a_suggestion_carries_the_heading_above_it():
    doc = document(
        paragraph(run("2. Routes"), style="HEADING_2"),
        paragraph(run("Route one ", insertion="s1")),
    )
    assert parse_suggestions(doc)[0].heading == "2. Routes"


def test_a_suggestion_before_any_heading_has_none():
    assert parse_suggestions(document(paragraph(run("New opener.", insertion="s1"))))[0].heading == ""


def test_runs_of_one_suggestion_are_joined():
    """Docs splits a typed sentence across runs at every formatting boundary."""
    doc = document(paragraph(run("We ", insertion="s1"), run("decide in September.", insertion="s1")))
    found = parse_suggestions(doc)
    assert len(found) == 1
    assert found[0].text == "We decide in September."


def test_two_different_suggestions_stay_apart():
    doc = document(
        paragraph(run("One.", insertion="s1"), run(" kept "), run("Two.", insertion="s2")),
    )
    assert len(parse_suggestions(doc)) == 2


def test_an_insertion_and_a_deletion_side_by_side_are_two():
    doc = document(paragraph(run("new ", insertion="s1"), run("old ", deletion="s2")))
    assert [s.kind for s in parse_suggestions(doc)] == ["insertion", "deletion"]


def test_a_suggestion_inside_a_table_is_found():
    cell = {"content": [paragraph(run("Fee 1.5%", insertion="s1"))]}
    table = {"table": {"tableRows": [{"tableCells": [cell]}]}}
    assert len(parse_suggestions(document(table))) == 1


def test_a_heading_that_is_itself_suggested_still_names_itself():
    doc = document(
        paragraph(run("2. Routes to membership", insertion="s1"), style="HEADING_2"),
        paragraph(run("Body.")),
    )
    found = parse_suggestions(doc)
    assert found[0].heading == "2. Routes to membership"


def test_a_document_with_no_body_is_empty_rather_than_an_error():
    assert parse_suggestions({}) == ()
