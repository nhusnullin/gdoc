"""The Commenter guarantee, asserted against the live API.

These assert on what Google permits, not on what the agent chooses to do.
A test that only proved the agent did not try to edit would prove nothing:
the guarantee is that it cannot.
"""

import pytest
from googleapiclient.errors import HttpError

pytestmark = pytest.mark.integration


def test_can_still_read_the_file(drive, test_doc_id):
    meta = drive.files().get(fileId=test_doc_id, fields="id,name").execute()
    assert meta["id"] == test_doc_id


def test_can_still_read_comments(drive, test_doc_id):
    res = drive.comments().list(fileId=test_doc_id, fields="comments(id)").execute()
    assert len(res.get("comments", [])) >= 3


def test_google_says_the_agent_cannot_edit(drive, test_doc_id):
    caps = drive.files().get(fileId=test_doc_id, fields="capabilities").execute()["capabilities"]
    assert caps["canEdit"] is False
    assert caps["canComment"] is True


def test_renaming_the_file_is_refused(drive, test_doc_id):
    """The rename must be a real change.

    An earlier probe patched the name to its existing value and got 200 back.
    That proved nothing: Drive had nothing to change. The new name has to
    differ, and the test reverts it if the call unexpectedly succeeds.
    """
    original = drive.files().get(fileId=test_doc_id, fields="name").execute()["name"]
    try:
        drive.files().update(fileId=test_doc_id, body={"name": original + " PROBE"}).execute()
    except HttpError as error:
        assert error.resp.status in (403, 404)
        return
    drive.files().update(fileId=test_doc_id, body={"name": original}).execute()
    pytest.fail("rename succeeded under Commenter, so the guarantee does not hold")


def test_editing_the_text_is_refused(drive, test_doc_id):
    """The write that actually matters.

    A rename is metadata. This inserts a character into the body, which is the
    thing the design promises can never happen. The Docs API is reached with
    the same drive scope.
    """
    from googleapiclient.discovery import build

    try:
        credentials = drive._http.credentials
    except AttributeError:
        from tools.gdoc.auth import load_credentials

        credentials = load_credentials()

    docs = build("docs", "v1", credentials=credentials, cache_discovery=False)
    try:
        docs.documents().batchUpdate(
            documentId=test_doc_id,
            body={"requests": [{"insertText": {"location": {"index": 1}, "text": "PROBE"}}]},
        ).execute()
    except HttpError as error:
        assert error.resp.status in (403, 404), f"unexpected status: {error}"
        return
    pytest.fail("text insert succeeded under Commenter, so the guarantee does not hold")


def test_deleting_a_comment_is_refused(drive, test_doc_id):
    res = drive.comments().list(fileId=test_doc_id, fields="comments(id)").execute()
    comment_id = res["comments"][0]["id"]
    with pytest.raises(HttpError) as excinfo:
        drive.comments().delete(fileId=test_doc_id, commentId=comment_id).execute()
    assert excinfo.value.resp.status in (403, 404)


def test_permissions_list_is_refused(drive, test_doc_id):
    """Already observed on 2026-08-13: 403.

    This is asserted, not printed, because a change here would change the
    design. Section 10's external-audience check cannot be done by the agent,
    so it belongs to Nail. If this ever starts passing, revisit that decision.
    """
    with pytest.raises(HttpError) as excinfo:
        drive.permissions().list(
            fileId=test_doc_id, fields="permissions(role,type,emailAddress)"
        ).execute()
    assert excinfo.value.resp.status in (403, 404)


def test_the_output_folder_accepts_new_files(drive):
    """The one place the agent is allowed to write."""
    from tools.gdoc.config import load_config

    folder_id = load_config().output_folder_id
    folder = drive.files().get(
        fileId=folder_id, supportsAllDrives=True, fields="name,driveId,capabilities(canAddChildren)"
    ).execute()
    assert folder["capabilities"]["canAddChildren"] is True
    assert folder.get("driveId"), "the output folder should be on a shared drive"
