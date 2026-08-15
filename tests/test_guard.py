"""The reachable set, asserted against the guard.

Under a service account the credential reached only the documents shared with
it. OAuth has no such limit: the token is Nail, the scope is full Drive, and
every file Nail owns is reachable. This file is what narrows it back down.

The rule is about files, never about methods. On a file the command was given,
every method is carried, an edit included. On any other file, nothing is,
a read included. Never widen the set to make a test pass.
"""

import json

import pytest

from gdoc.guard import CARRY, LEARN, REFUSE, GuardedHttp, file_id, verdict

MINE = "1TheDocumentIWasGiven"
YOURS = "1SomeOtherFileEntirely"
ALLOWED = frozenset({MINE})

DRIVE = "https://www.googleapis.com/drive/v3"
UPLOAD = "https://www.googleapis.com/upload/drive/v3"
DOCS = "https://docs.googleapis.com/v1"


class FakeHttp:
    """Records what it was asked to carry, and answers with a canned body."""

    def __init__(self, body=b"{}", status="200"):
        self.calls = []
        self.body = body
        self.status = status
        self.credentials = "the credentials"

    def request(self, uri, method="GET", body=None, headers=None, **kwargs):
        self.calls.append((method, uri))
        return ({"status": self.status}, self.body)


def guarded(allowed=ALLOWED, **kwargs):
    inner = FakeHttp(**kwargs)
    return GuardedHttp(inner, allowed), inner


# --- reading the file id out of a url ------------------------------------


def test_the_id_is_read_from_a_drive_path():
    assert file_id(f"{DRIVE}/files/{MINE}") == MINE


def test_the_id_is_read_from_a_drive_subpath():
    assert file_id(f"{DRIVE}/files/{MINE}/comments/c1/replies") == MINE


def test_the_id_is_read_from_an_upload_path():
    assert file_id(f"{UPLOAD}/files/{MINE}?uploadType=media") == MINE


def test_the_id_is_read_from_a_docs_path():
    assert file_id(f"{DOCS}/documents/{MINE}") == MINE


def test_the_id_stops_at_the_colon_in_a_docs_method():
    """documents/{id}:batchUpdate is the write that matters most."""
    assert file_id(f"{DOCS}/documents/{MINE}:batchUpdate") == MINE


def test_a_collection_path_carries_no_id():
    assert file_id(f"{DRIVE}/files") is None


def test_discovery_carries_no_id():
    assert file_id("https://www.googleapis.com/discovery/v1/apis/drive/v3/rest") is None


# --- every method is carried on the file we were given -------------------


@pytest.mark.parametrize(
    "method,uri",
    [
        ("GET", f"{DRIVE}/files/{MINE}"),
        ("GET", f"{DRIVE}/files/{MINE}/export?mimeType=application%2Fpdf"),
        ("GET", f"{DRIVE}/files/{MINE}/comments?fields=comments(id)"),
        ("POST", f"{DRIVE}/files/{MINE}/comments"),
        ("POST", f"{DRIVE}/files/{MINE}/comments/c1/replies?fields=id"),
        ("PATCH", f"{DRIVE}/files/{MINE}"),
        ("PUT", f"{UPLOAD}/files/{MINE}?uploadType=media"),
        ("DELETE", f"{DRIVE}/files/{MINE}/comments/c1"),
        ("POST", f"{DOCS}/documents/{MINE}:batchUpdate"),
    ],
)
def test_any_method_is_carried_on_the_named_file(method, uri):
    assert verdict(method, uri, ALLOWED) == CARRY


def test_trashing_the_measuring_copy_is_carried():
    """generate creates a copy, measures it, then bins it. This is the bin."""
    assert verdict("PATCH", f"{DRIVE}/files/{MINE}", ALLOWED) == CARRY


# --- nothing at all on any other file ------------------------------------


@pytest.mark.parametrize(
    "method,uri",
    [
        ("GET", f"{DRIVE}/files/{YOURS}"),
        ("GET", f"{DRIVE}/files/{YOURS}/comments"),
        ("GET", f"{DRIVE}/files/{YOURS}/export?mimeType=application%2Fpdf"),
        ("POST", f"{DRIVE}/files/{YOURS}/comments"),
        ("PATCH", f"{DRIVE}/files/{YOURS}"),
        ("DELETE", f"{DRIVE}/files/{YOURS}"),
        ("POST", f"{DOCS}/documents/{YOURS}:batchUpdate"),
    ],
)
def test_nothing_is_carried_on_a_file_we_were_not_given(method, uri):
    assert verdict(method, uri, ALLOWED) == REFUSE


def test_reading_someone_elses_document_is_refused():
    """The failure the verb allowlist could not see. A GET is not harmless."""
    assert verdict("GET", f"{DRIVE}/files/{YOURS}", ALLOWED) == REFUSE


def test_an_empty_allowed_set_refuses_a_read():
    """A command that forgets to name its document reaches nothing."""
    assert verdict("GET", f"{DRIVE}/files/{MINE}", frozenset()) == REFUSE


# --- the collection paths ------------------------------------------------


def test_listing_drive_is_refused():
    """gdoc never searches Drive. This is most of 'it sees only what it is given'."""
    assert verdict("GET", f"{DRIVE}/files?q=name+contains+'policy'", ALLOWED) == REFUSE


def test_creating_a_file_is_carried_and_learned():
    assert verdict("POST", f"{DRIVE}/files", frozenset()) == LEARN


def test_creating_a_file_with_media_is_carried_and_learned():
    assert verdict("POST", f"{UPLOAD}/files?uploadType=multipart", frozenset()) == LEARN


# --- refused whatever the file ------------------------------------------


def test_sharing_the_named_file_is_refused():
    """Granting other people access is a different authority from editing."""
    assert verdict("POST", f"{DRIVE}/files/{MINE}/permissions", ALLOWED) == REFUSE


def test_revoking_access_to_the_named_file_is_refused():
    assert verdict("DELETE", f"{DRIVE}/files/{MINE}/permissions/p1", ALLOWED) == REFUSE


def test_changing_a_permission_on_the_named_file_is_refused():
    assert verdict("PATCH", f"{DRIVE}/files/{MINE}/permissions/p1", ALLOWED) == REFUSE


def test_reading_who_has_access_to_the_named_file_is_carried():
    """A read of the named file like any other.

    Under a service account Google answers this with a 403 regardless, which
    tests/test_access_integration.py records. Refusing it here would buy
    nothing and would hide that.
    """
    assert verdict("GET", f"{DRIVE}/files/{MINE}/permissions", ALLOWED) == CARRY


def test_reading_who_has_access_to_another_file_is_still_refused():
    assert verdict("GET", f"{DRIVE}/files/{YOURS}/permissions", ALLOWED) == REFUSE


def test_a_batch_request_is_refused():
    """The file ids live in the body, where the guard cannot see them."""
    assert verdict("POST", "https://www.googleapis.com/batch/drive/v3", ALLOWED) == REFUSE


# --- carried with no file involved ---------------------------------------


def test_discovery_is_carried():
    uri = "https://www.googleapis.com/discovery/v1/apis/drive/v3/rest"
    assert verdict("GET", uri, frozenset()) == CARRY


def test_about_get_is_carried():
    """auth status asks who is signed in, and that names no file."""
    assert verdict("GET", f"{DRIVE}/about?fields=user", frozenset()) == CARRY


# --- matching details ----------------------------------------------------


def test_the_query_string_changes_no_verdict():
    assert verdict("POST", f"{UPLOAD}/files?uploadType=resumable", frozenset()) == LEARN
    assert verdict("GET", f"{DRIVE}/files?corpora=allDrives", ALLOWED) == REFUSE


def test_the_host_is_matched_not_only_the_path():
    assert verdict("GET", f"https://example.com/drive/v3/files/{MINE}", ALLOWED) == REFUSE


def test_the_method_is_case_insensitive():
    assert verdict("patch", f"{DRIVE}/files/{MINE}", ALLOWED) == CARRY
    assert verdict("get", f"{DRIVE}/files/{YOURS}", ALLOWED) == REFUSE


def test_an_unknown_googleapis_path_is_refused():
    assert verdict("GET", "https://www.googleapis.com/gmail/v1/users/me/messages", ALLOWED) == REFUSE


# --- the transport -------------------------------------------------------


def test_a_carried_request_reaches_the_inner_transport():
    http, inner = guarded()
    http.request(f"{DRIVE}/files/{MINE}/comments", method="GET")
    assert inner.calls == [("GET", f"{DRIVE}/files/{MINE}/comments")]


def test_a_refused_request_never_reaches_the_inner_transport():
    http, inner = guarded()
    with pytest.raises(PermissionError):
        http.request(f"{DRIVE}/files/{YOURS}", method="GET")
    assert inner.calls == []


def test_the_refusal_names_the_method_the_path_and_the_file():
    http, _ = guarded()
    with pytest.raises(PermissionError) as excinfo:
        http.request(f"{DRIVE}/files/{YOURS}", method="PATCH")
    message = str(excinfo.value)
    assert "PATCH" in message
    assert f"/drive/v3/files/{YOURS}" in message
    assert YOURS in message
    assert "was given" in message


def test_the_default_method_is_get():
    http, inner = guarded()
    http.request(f"{DRIVE}/files/{MINE}")
    assert inner.calls == [("GET", f"{DRIVE}/files/{MINE}")]


def test_other_attributes_proxy_to_the_inner_transport():
    """googleapiclient reaches past request() during media upload."""
    http, _ = guarded()
    assert http.credentials == "the credentials"


# --- learning a created id -----------------------------------------------


def test_a_created_file_joins_the_allowed_set():
    body = json.dumps({"id": "1FreshlyCreated"}).encode()
    http, _ = guarded(allowed=frozenset(), body=body)
    http.request(f"{UPLOAD}/files?uploadType=multipart", method="POST")
    assert verdict("GET", f"{DRIVE}/files/1FreshlyCreated", http.allowed) == CARRY


def test_the_whole_generate_shape_works_from_an_empty_set():
    """create, export, create, trash. Three of the four address a new file."""
    body = json.dumps({"id": "1Measured"}).encode()
    http, inner = guarded(allowed=frozenset(), body=body)
    http.request(f"{UPLOAD}/files?uploadType=multipart", method="POST")
    http.request(f"{DRIVE}/files/1Measured/export?mimeType=application%2Fpdf")
    http.request(f"{DRIVE}/files/1Measured", method="PATCH")
    assert len(inner.calls) == 3


def test_a_create_that_answers_with_something_else_teaches_nothing():
    http, _ = guarded(allowed=frozenset(), body=b"<html>a proxy said no</html>")
    http.request(f"{DRIVE}/files", method="POST")
    assert http.allowed == frozenset()


def test_a_create_that_answers_without_an_id_teaches_nothing():
    http, _ = guarded(allowed=frozenset(), body=json.dumps({"kind": "drive#file"}).encode())
    http.request(f"{DRIVE}/files", method="POST")
    assert http.allowed == frozenset()


def test_a_failed_create_teaches_nothing():
    body = json.dumps({"id": "1NeverActuallyMade"}).encode()
    http, _ = guarded(allowed=frozenset(), body=body, status="403")
    http.request(f"{DRIVE}/files", method="POST")
    assert http.allowed == frozenset()


def test_learning_does_not_mutate_the_set_it_was_given():
    """The caller's frozenset must not change under it."""
    given = frozenset({MINE})
    body = json.dumps({"id": "1FreshlyCreated"}).encode()
    http, _ = guarded(allowed=given, body=body)
    http.request(f"{DRIVE}/files", method="POST")
    assert given == frozenset({MINE})
    assert http.allowed == frozenset({MINE, "1FreshlyCreated"})
