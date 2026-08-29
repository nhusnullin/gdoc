import json, urllib.request, urllib.error
from gauth import creds
from google.auth.transport.requests import Request as GReq
from googleapiclient.discovery import build

c = creds()
if not c.valid:
    c.refresh(GReq())
TOK = c.token

def http(method, url, body=None, headers=None):
    data = json.dumps(body).encode() if body is not None else None
    h = {"Authorization": f"Bearer {TOK}", "Content-Type": "application/json"}
    h.update(headers or {})
    req = urllib.request.Request(url, data=data, headers=h, method=method)
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            return r.status, r.read().decode()
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()

# --- authenticated discovery, with the preview label ---
for label in ("", "&labels=DEVELOPER_PREVIEW"):
    s, b = http("GET", f"https://docs.googleapis.com/$discovery/rest?version=v1{label}")
    try:
        wc = json.loads(b)["schemas"]["WriteControl"]["properties"]
        print(f"authed discovery{label or ' (standard)':30s} -> {s} WriteControl props {sorted(wc)}")
    except Exception:
        print(f"authed discovery{label:30s} -> {s} {b[:160]}")

# --- make a doc and try the header variants on the real call ---
drive = build("drive", "v3", credentials=c)
doc = drive.files().create(
    body={"name": "gdoc probe5 (delete me)", "mimeType": "application/vnd.google-apps.document",
          "parents": ["1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"]},
    fields="id", supportsAllDrives=True).execute()
did = doc["id"]
print("\ndoc:", did)
URL = f"https://docs.googleapis.com/v1/documents/{did}:batchUpdate"
http("POST", URL, {"requests":[{"insertText":{"location":{"index":1},"text":"Base sentence.\n"}}]})

def attempt(label, headers=None, extra_qs="", body_extra=None):
    body = {"requests":[{"insertText":{"location":{"index":1},"text":f"[[{label}]]"}}],
            "writeControl":{"writeMode":"SUGGEST"}}
    if body_extra: body.update(body_extra)
    s, b = http("POST", URL + extra_qs, body, headers)
    msg = b.replace("\n"," ")[:150]
    print(f"{label:32s} -> {s}  {msg}")

attempt("PLAIN")
attempt("VIS_DEVPREVIEW", {"X-Goog-Visibilities": "DEVELOPER_PREVIEW"})
attempt("VIS_PREVIEW", {"X-Goog-Visibilities": "PREVIEW"})
attempt("VIS_TRUSTED", {"X-Goog-Visibilities": "TRUSTED_TESTER"})
attempt("QS_PREVIEWVERSION", extra_qs="?previewVersion=V1_20260707_PREVIEW")
attempt("BODY_PREVIEWVERSION", body_extra={"previewVersion": "V1_20260707_PREVIEW"})
attempt("APIVERSION_HEADER", {"X-Goog-Api-Version": "preview"})

drive.files().update(fileId=did, body={"trashed": True}, supportsAllDrives=True).execute()
print("trashed:", did)
