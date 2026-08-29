import io,json,re,urllib.request,urllib.error,zipfile
from gauth import creds
from google.auth.transport.requests import Request as GReq
from googleapiclient.discovery import build
from googleapiclient.http import MediaIoBaseDownload
c=creds()
if not c.valid: c.refresh(GReq())
TOK=c.token
drive=build("drive","v3",credentials=c)
D=drive.files().create(body={"name":"gdoc AUTHORSHIP probe",
 "mimeType":"application/vnd.google-apps.document",
 "parents":["1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"]},fields="id",supportsAllDrives=True).execute()["id"]
U=f"https://docs.googleapis.com/v1/documents/{D}"
def http(m,u,b=None):
    data=json.dumps(b).encode() if b is not None else None
    r=urllib.request.Request(u,data=data,method=m,headers={"Authorization":f"Bearer {TOK}","Content-Type":"application/json"})
    try:
        with urllib.request.urlopen(r,timeout=60) as x: return x.status,x.read().decode()
    except urllib.error.HTTPError as e: return e.code,e.read().decode()

http("POST",U+":batchUpdate",{"requests":[{"insertText":{"location":{"index":1},
  "text":"Alpha sentence one here. Beta sentence two here. Gamma sentence three here.\n"}}]})

# --- can an author be supplied anywhere? probe every plausible field ---
print("=== can authorship be set? ===")
for label,payload in [
  ("writeControl.author", {"requests":[{"insertText":{"location":{"index":1},"text":"x"}}],"writeControl":{"writeMode":"SUGGEST","author":"gdoc agent"}}),
  ("writeControl.suggestionAuthor",{"requests":[{"insertText":{"location":{"index":1},"text":"x"}}],"writeControl":{"writeMode":"SUGGEST","suggestionAuthor":"gdoc agent"}}),
  ("top-level author",{"requests":[{"insertText":{"location":{"index":1},"text":"x"}}],"writeControl":{"writeMode":"SUGGEST"},"author":"gdoc agent"}),
  ("insertComment.author",{"requests":[{"insertComment":{"range":{"startIndex":1,"endIndex":6},"content":"y","author":"gdoc agent"}}]}),
  ("insertComment.authorDisplayName",{"requests":[{"insertComment":{"range":{"startIndex":1,"endIndex":6},"content":"y","authorDisplayName":"gdoc"}}]}),
]:
    s,b=http("POST",U+":batchUpdate",payload)
    msg=json.loads(b).get("error",{}).get("message","OK") if s!=200 else "ACCEPTED"
    print(f"  {label:32s} -> {s} {msg[:95]}")

# --- who is actually credited? ---
http("POST",U+":batchUpdate",{"requests":[{"insertText":{"location":{"index":6},"text":"INSERTED "}}],
     "writeControl":{"writeMode":"SUGGEST"}})
http("POST",U+":batchUpdate",{"requests":[{"insertComment":{"range":{"startIndex":1,"endIndex":6},
     "content":"gdoc: proposing a change here."}}]})
s,b=http("GET",U+"?suggestionsViewMode=SUGGESTIONS_INLINE&commentsViewMode=COMMENTS_VIEW_MODE_INCLUDED&includeTabsContent=true")
print("\n=== who is credited? ===")
names=set(re.findall(r'"displayName"\s*:\s*"([^"]*)"',b))
print("  displayName values in the document:", sorted(names))
print("  'me': true present:", '"me": true' in b or '"me":true' in b)
req=drive.files().export_media(fileId=D,mimeType="application/vnd.openxmlformats-officedocument.wordprocessingml.document")
buf=io.BytesIO(); dl=MediaIoBaseDownload(buf,req); done=False
while not done: _,done=dl.next_chunk()
buf.seek(0)
with zipfile.ZipFile(buf) as z:
    dx=z.read("word/document.xml").decode()
    cx=z.read("word/comments.xml").decode() if "word/comments.xml" in z.namelist() else ""
print("  w:ins author in export     :", re.findall(r'<w:ins[^>]*w:author="([^"]*)"',dx))
print("  w:comment author in export :", re.findall(r'<w:comment[^>]*w:author="([^"]*)"',cx))
print("\nhttps://docs.google.com/document/d/%s/edit"%D)
