import json
from gauth import creds
from googleapiclient.discovery import build
from googleapiclient.errors import HttpError

FOLDER = "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"
c = creds()
drive = build("drive", "v3", credentials=c)
docs = build("docs", "v1", credentials=c)

doc = drive.files().create(
    body={"name": "gdoc probe3 (delete me)", "mimeType": "application/vnd.google-apps.document",
          "parents": [FOLDER]}, fields="id", supportsAllDrives=True).execute()
did = doc["id"]
print("doc:", did)

def attempt(label, wc, text):
    body = {"requests": [{"insertText": {"location": {"index": 1}, "text": text}}]}
    if wc is not None:
        body["writeControl"] = wc
    try:
        r = docs.documents().batchUpdate(documentId=did, body=body).execute()
        print(f"{label:34s} -> 200  {json.dumps(r)[:110]}")
        return True
    except HttpError as e:
        print(f"{label:34s} -> {e.resp.status}  {e.reason}")
        return False

try:
    attempt("control, no writeControl", None, "Base sentence.\n")
    attempt("writeMode SUGGEST", {"writeMode": "SUGGEST"}, "[[SUGGEST]]")
    attempt("writeMode SUGGEST (repeat)", {"writeMode": "SUGGEST"}, "[[SUGGEST2]]")
    attempt("writeMode EDIT", {"writeMode": "EDIT"}, "[[EDIT]]")
    attempt("writeMode WRITE_MODE_UNSPECIFIED", {"writeMode": "WRITE_MODE_UNSPECIFIED"}, "[[UNSPEC]]")
    attempt("writeControl empty", {}, "[[EMPTY]]")
    attempt("writeMode NOT_A_MODE", {"writeMode": "NOT_A_MODE"}, "[[BAD]]")

    d = docs.documents().get(documentId=did, suggestionsViewMode="PREVIEW_WITHOUT_SUGGESTIONS").execute()
    txt = "".join((pe.get("textRun") or {}).get("content","")
                  for el in d["body"]["content"]
                  for pe in (el.get("paragraph") or {}).get("elements", []))
    print("\nPREVIEW_WITHOUT_SUGGESTIONS:", repr(txt))
    print("suggestedInsertionIds present:", "suggestedInsertionIds" in json.dumps(d))
finally:
    drive.files().update(fileId=did, body={"trashed": True}, supportsAllDrives=True).execute()
    print("trashed:", did)
