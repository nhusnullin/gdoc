import json, urllib.request, urllib.error
from gauth import creds
from google.auth.transport.requests import Request as GReq
from googleapiclient.discovery import build

c = creds()
if not c.valid: c.refresh(GReq())
TOK=c.token
drive = build("drive","v3",credentials=c)
doc = drive.files().create(body={"name":"gdoc INSERTCOMMENT probe",
  "mimeType":"application/vnd.google-apps.document",
  "parents":["1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"]}, fields="id",
  supportsAllDrives=True).execute()
D=doc["id"]; U=f"https://docs.googleapis.com/v1/documents/{D}"
print("doc:",D)

def http(m,u,b=None):
    data=json.dumps(b).encode() if b is not None else None
    r=urllib.request.Request(u,data=data,method=m,headers={
      "Authorization":f"Bearer {TOK}","Content-Type":"application/json"})
    try:
        with urllib.request.urlopen(r,timeout=60) as x: return x.status,x.read().decode()
    except urllib.error.HTTPError as e: return e.code,e.read().decode()

TEXT="Altery reviews each critical supplier every six months and records the outcome.\n"
http("POST",U+":batchUpdate",{"requests":[{"insertText":{"location":{"index":1},"text":TEXT}}]})
# "critical supplier" sits at 15..32
def try_shape(label, payload):
    s,b=http("POST",U+":batchUpdate",{"requests":[{"insertComment":payload}]})
    msg=json.loads(b).get("error",{}).get("message","") if s!=200 else "OK"
    print(f"{label:38s} -> {s}  {msg[:120]}")
    return s

try_shape("empty {}", {})
try_shape("quotedRange+comment", {"quotedRange":{"startIndex":15,"endIndex":32},"comment":{"content":"x"}})
try_shape("range+comment", {"range":{"startIndex":15,"endIndex":32},"comment":{"content":"x"}})
try_shape("anchor.range", {"anchor":{"range":{"startIndex":15,"endIndex":32}},"comment":{"content":"x"}})
try_shape("commentAnchor.range", {"commentAnchor":{"range":{"startIndex":15,"endIndex":32}},"comment":{"content":"x"}})
try_shape("anchor+content", {"anchor":{"range":{"startIndex":15,"endIndex":32}},"content":"x"})
try_shape("range+content", {"range":{"startIndex":15,"endIndex":32},"content":"x"})
try_shape("segmentRange", {"anchor":{"segmentId":"","startIndex":15,"endIndex":32},"content":"x"})
print("\nhttps://docs.google.com/document/d/%s/edit"%D)
