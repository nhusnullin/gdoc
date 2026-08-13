from unittest.mock import MagicMock

from tools.gdoc.fetch import fetch_threads


def _page(comments, next_token=None):
    page = {"comments": comments}
    if next_token:
        page["nextPageToken"] = next_token
    return page


def test_follows_pagination_until_exhausted():
    drive = MagicMock()
    drive.comments().list.return_value.execute.side_effect = [
        _page([{"id": "a", "content": "ai: one"}], next_token="tok"),
        _page([{"id": "b", "content": "ai: two"}]),
    ]
    threads = fetch_threads(drive, "docid")
    assert [t.id for t in threads] == ["a", "b"]


def test_returns_an_immutable_tuple():
    drive = MagicMock()
    drive.comments().list.return_value.execute.side_effect = [_page([{"id": "a", "content": "x"}])]
    assert isinstance(fetch_threads(drive, "docid"), tuple)


def test_empty_document_returns_empty_tuple():
    drive = MagicMock()
    drive.comments().list.return_value.execute.side_effect = [_page([])]
    assert fetch_threads(drive, "docid") == ()
