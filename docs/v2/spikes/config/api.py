"""Drive/Docs access for this experiment, with a hard denylist.

Nothing mutating runs against an id this process did not create.
"""
import json, os
from pathlib import Path
from google.oauth2.credentials import Credentials
from google.auth.transport.requests import Request, AuthorizedSession

TOKEN = os.path.expanduser("~/.config/gdoc-agent/oauth-token.json")
FOLDER = "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"
TEMPLATE = "1R4gmDnE9uZqltTHN7xWzGOe1klPB3Z2BVEVYfVRRSZw"   # READ ONLY

DENY = {
    "1W8GK52CWG8tb2B-0n_AnXd8YsvhayN_nGmLJF6Rv8lc",
    "1LXTWhtimCkqvacfMeGg8ScNWbX6Ce0Dyy4oVREdZ5Dc",
    "1lQuQybi0Jr6pr2gT9tWeYrbY7-q58Ry1_lstenhkITQ",
    "1-ZDmc6EGpaOcNElUNKqKlKg4F83CAAE4iULxsUwfkKw",
    "18Q_NarPRTqTqWEsvblEgw1Lyn_51_74RD-R5EehjeA0",
    "1-1MPzJfXFvypfd43OdUMN8tO36JsInlHoMGB0OiFQbs",
    TEMPLATE,
}
NAME_DENY = ("PROPOSAL DEMO", "commenter")

STATE = Path(__file__).resolve().parent / "created-ids.json"
CREATED = set(json.loads(STATE.read_text())) if STATE.exists() else set()


def _remember(doc_id):
    CREATED.add(doc_id)
    STATE.write_text(json.dumps(sorted(CREATED), indent=1))


def assert_writable(doc_id, name=""):
    assert doc_id not in DENY, f"DENYLIST HIT: {doc_id}"
    assert doc_id != FOLDER, "refusing to write to the folder itself"
    for bad in NAME_DENY:
        assert bad.lower() not in (name or "").lower(), f"NAME DENY: {name}"
    assert doc_id in CREATED, f"NOT CREATED BY THIS RUN: {doc_id}"


_S = None
def s():
    global _S
    if _S is None:
        c = Credentials.from_authorized_user_file(TOKEN)
        if not c.valid:
            c.refresh(Request())
        _S = AuthorizedSession(c)
    return _S


SAD = {"supportsAllDrives": "true"}


def upload_convert(path, name):
    meta = {"name": name, "parents": [FOLDER],
            "mimeType": "application/vnd.google-apps.document"}
    files = {
        "metadata": ("metadata", json.dumps(meta), "application/json; charset=UTF-8"),
        "file": (os.path.basename(path), open(path, "rb"),
                 "application/vnd.openxmlformats-officedocument.wordprocessingml.document"),
    }
    r = s().post("https://www.googleapis.com/upload/drive/v3/files",
                 params={"uploadType": "multipart", **SAD, "fields": "id,name"},
                 files=files)
    r.raise_for_status()
    j = r.json()
    _remember(j["id"])
    return j


def copy_template(name):
    r = s().post(f"https://www.googleapis.com/drive/v3/files/{TEMPLATE}/copy",
                 params={**SAD, "fields": "id,name"},
                 json={"name": name, "parents": [FOLDER]})
    r.raise_for_status()
    j = r.json()
    _remember(j["id"])
    return j


def get_doc(doc_id, **params):
    r = s().get(f"https://docs.googleapis.com/v1/documents/{doc_id}", params=params)
    r.raise_for_status()
    return r.json()


def batch(doc_id, requests, name=""):
    assert_writable(doc_id, name)
    r = s().post(f"https://docs.googleapis.com/v1/documents/{doc_id}:batchUpdate",
                 json={"requests": requests})
    if r.status_code >= 400:
        raise RuntimeError(f"{r.status_code} {r.text[:800]}")
    return r.json()


def export_pdf(doc_id, out):
    r = s().get(f"https://www.googleapis.com/drive/v3/files/{doc_id}/export",
                params={"mimeType": "application/pdf", **SAD})
    r.raise_for_status()
    Path(out).write_bytes(r.content)
    return out
