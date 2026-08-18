"""What the credential is allowed to do, asserted against the live API.

These assert on what is permitted, not on what the agent chooses to do. A test
that only proved the agent did not try would prove nothing.

What is guaranteed depends on the credential, so the file splits by mode.

- service_account: Google refuses every write. Commenter access cannot edit,
  rename, or delete, and the credential reaches only documents shared with it.
- oauth: Google refuses nothing, because the token is Nail and the scope is full
  Drive. gdoc/guard.py provides a different guarantee: the client reaches only
  the file it was built for. So the OAuth cases assert the boundary, not the
  verb.

No test here attempts an edit on the test document. Under service_account Google
would refuse it; under oauth it would succeed, and succeeding means editing a
real document. Never add one.

Never loosen an assertion to suit the configured mode. Add the other mode's case.
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


def test_google_says_the_agent_cannot_edit(drive, test_doc_id, service_account_only):
    caps = drive.files().get(fileId=test_doc_id, fields="capabilities").execute()["capabilities"]
    assert caps["canEdit"] is False
    assert caps["canComment"] is True


def test_renaming_the_file_is_refused(drive, test_doc_id, service_account_only):
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


def test_editing_the_text_is_refused(drive, test_doc_id, service_account_only):
    """The write that actually matters.

    A rename is metadata. This inserts a character into the body, which is the
    thing the design promises can never happen. The Docs API is reached with
    the same drive scope.
    """
    from googleapiclient.discovery import build

    try:
        credentials = drive._http.credentials
    except AttributeError:
        from gdoc.auth import load_credentials

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


def test_deleting_a_comment_is_refused(drive, test_doc_id, service_account_only):
    res = drive.comments().list(fileId=test_doc_id, fields="comments(id)").execute()
    comment_id = res["comments"][0]["id"]
    with pytest.raises(HttpError) as excinfo:
        drive.comments().delete(fileId=test_doc_id, commentId=comment_id).execute()
    assert excinfo.value.resp.status in (403, 404)


def test_permissions_list_is_refused(drive, test_doc_id, service_account_only):
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
    """The one place the agent is allowed to write.

    Built with its own client, because the shared `drive` fixture is scoped to
    the test document and the guard refuses every other file. cmd_generate does
    the same thing: it passes the output folder id and nothing else.
    """
    from gdoc.auth import drive_service
    from gdoc.config import load_config

    folder_id = load_config().output_folder_id
    folder = drive_service(doc_ids=folder_id).files().get(
        fileId=folder_id, supportsAllDrives=True, fields="name,driveId,capabilities(canAddChildren)"
    ).execute()
    assert folder["capabilities"]["canAddChildren"] is True
    assert folder.get("driveId"), "the output folder should be on a shared drive"


def test_native_markdown_export_works_on_the_real_document(drive, test_doc_id):
    """Records whether the pandoc fallback is dead code or load-bearing."""
    from gdoc.export import export_markdown

    text = export_markdown(drive, test_doc_id)
    print(f"exported {len(text)} characters of markdown")
    assert text.strip()


# ---------------------------------------------------------------------------
# Under OAuth the guarantee is the boundary, not the verb
# ---------------------------------------------------------------------------


def test_reading_a_document_we_were_not_given_is_refused(
    drive, other_doc_id, oauth_only
):
    """The failure the first draft's verb allowlist could not see.

    OAuth reaches every file Nail owns, so a read is not harmless. This is the
    guarantee that replaces Commenter.
    """
    with pytest.raises(PermissionError) as excinfo:
        drive.files().get(fileId=other_doc_id, fields="name").execute()
    assert other_doc_id in str(excinfo.value)


def test_listing_drive_is_refused(drive, oauth_only):
    """gdoc never searches Drive, and under OAuth it must not be able to."""
    with pytest.raises(PermissionError):
        drive.files().list(pageSize=1, fields="files(id)").execute()


def test_reading_someone_elses_comments_is_refused(drive, other_doc_id, oauth_only):
    with pytest.raises(PermissionError):
        drive.comments().list(fileId=other_doc_id, fields="comments(id)").execute()


def test_sharing_the_named_document_is_refused(drive, test_doc_id, oauth_only):
    """Allowed file, refused anyway. Granting access is a different authority.

    Refused inside the process, so nothing is ever shared with anyone. If this
    test starts failing, the next run of it publishes a real document to the
    whole internet.
    """
    with pytest.raises(PermissionError):
        drive.permissions().create(
            fileId=test_doc_id, body={"role": "reader", "type": "anyone"}
        ).execute()


def test_the_docs_api_is_bounded_by_the_same_set(drive, other_doc_id, oauth_only):
    """The guard was never told the Docs API exists. It shares the transport."""
    from googleapiclient.discovery import build

    docs = build("docs", "v1", http=drive._http, cache_discovery=False)
    with pytest.raises(PermissionError):
        docs.documents().batchUpdate(
            documentId=other_doc_id,
            body={"requests": [{"insertText": {"location": {"index": 1}, "text": "PROBE"}}]},
        ).execute()


def test_reading_and_commenting_still_work_under_the_guard(
    drive, test_doc_id, oauth_only
):
    """The guard must not break what the tool is for."""
    meta = drive.files().get(fileId=test_doc_id, fields="id").execute()
    assert meta["id"] == test_doc_id
    res = drive.comments().list(fileId=test_doc_id, fields="comments(id)").execute()
    assert len(res.get("comments", [])) >= 3
