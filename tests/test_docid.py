import pytest

from tools.gdoc.docid import extract_doc_id

DOC_ID = "1tyVhOTw9-bJ99IJTBZoTfWAT6fm3rjGIzfpHQX91hkw"


def test_extracts_id_from_edit_url_with_tab_fragment():
    url = f"https://docs.google.com/document/d/{DOC_ID}/edit?tab=t.0"
    assert extract_doc_id(url) == DOC_ID


def test_extracts_id_from_url_with_heading_anchor():
    url = f"https://docs.google.com/document/d/{DOC_ID}/edit#heading=h.abc123"
    assert extract_doc_id(url) == DOC_ID


def test_extracts_id_from_bare_url_without_suffix():
    assert extract_doc_id(f"https://docs.google.com/document/d/{DOC_ID}") == DOC_ID


def test_accepts_a_bare_document_id():
    assert extract_doc_id(DOC_ID) == DOC_ID


def test_strips_surrounding_whitespace():
    assert extract_doc_id(f"  {DOC_ID}  ") == DOC_ID


def test_raises_on_a_non_docs_url():
    with pytest.raises(ValueError, match="not a Google Docs URL"):
        extract_doc_id("https://example.com/page")


def test_raises_on_empty_input():
    with pytest.raises(ValueError):
        extract_doc_id("")
