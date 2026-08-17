import pytest

from gdoc.docid import extract_doc_id, extract_folder_id

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


# ---------------------------------------------------------------------------
# folder ids
# ---------------------------------------------------------------------------

FOLDER_ID = "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"


def test_extracts_folder_id_from_a_shared_folder_url():
    url = f"https://drive.google.com/drive/folders/{FOLDER_ID}?usp=sharing"
    assert extract_folder_id(url) == FOLDER_ID


def test_extracts_folder_id_from_an_account_scoped_url():
    url = f"https://drive.google.com/drive/u/0/folders/{FOLDER_ID}"
    assert extract_folder_id(url) == FOLDER_ID


def test_accepts_a_bare_folder_id():
    assert extract_folder_id(FOLDER_ID) == FOLDER_ID


def test_a_document_url_is_not_a_folder():
    """A wrong paste that would otherwise be a 404 on upload."""
    with pytest.raises(ValueError, match="a document, not a folder"):
        extract_folder_id(f"https://docs.google.com/document/d/{DOC_ID}/edit")


def test_raises_on_a_url_that_names_no_folder():
    with pytest.raises(ValueError, match="not a Drive folder"):
        extract_folder_id("https://example.com/page")
