import json, urllib.request, urllib.error
from gauth import creds
from google.auth.transport.requests import Request as GReq
from googleapiclient.discovery import build
c=creds()
if not c.valid: c.refresh(GReq())
TOK=c.token
drive=build("drive","v3",credentials=c)
d=drive.files().create(body={"name":"gdoc layout re-probe (delete me)",
  "mimeType":"application/vnd.google-apps.document",
  "parents":["1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"]},fields="id",supportsAllDrives=True).execute()["id"]
U=f"https://docs.googleapis.com/v1/documents/{d}"
def http(m,u,b=None):
    data=json.dumps(b).encode() if b is not None else None
    r=urllib.request.Request(u,data=data,method=m,headers={"Authorization":f"Bearer {TOK}","Content-Type":"application/json"})
    try:
        with urllib.request.urlopen(r,timeout=60) as x: return x.status,x.read().decode()
    except urllib.error.HTTPError as e: return e.code,e.read().decode()
http("POST",U+":batchUpdate",{"requests":[{"insertText":{"location":{"index":1},"text":"Heading\nBody text here.\n"}}]})
def probe(label,req):
    s,b=http("POST",U+":batchUpdate",{"requests":[req]})
    msg=json.loads(b).get("error",{}).get("message","OK") if s!=200 else "OK"
    print(f"{label:34s} -> {s}  {msg[:105]}")
probe("insertTableOfContents",{"insertTableOfContents":{"location":{"index":1}}})
probe("createTableOfContents",{"createTableOfContents":{"location":{"index":1}}})
probe("refreshTableOfContents",{"refreshTableOfContents":{}})
probe("createHeader FIRST_PAGE_HEADER",{"createHeader":{"type":"FIRST_PAGE_HEADER"}})
probe("updateDocumentStyle firstPageHeaderId",{"updateDocumentStyle":{"documentStyle":{"firstPageHeaderId":"kix.x"},"fields":"firstPageHeaderId"}})
probe("createPositionedObject",{"createPositionedObject":{}})
probe("insertPageNumber",{"insertPageNumber":{"location":{"index":1}}})
probe("insertAutoText",{"insertAutoText":{"location":{"index":1},"type":"PAGE_NUMBER"}})
probe("insertComment (control, works)",{"insertComment":{"range":{"startIndex":1,"endIndex":5},"content":"c"}})
drive.files().update(fileId=d,body={"trashed":True},supportsAllDrives=True).execute()
print("\ntrashed", d)
