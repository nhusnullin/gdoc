import io, json, re, urllib.request, urllib.error, zipfile
from gauth import creds
from google.auth.transport.requests import Request as GReq
from googleapiclient.discovery import build
from googleapiclient.http import MediaIoBaseDownload

c=creds()
if not c.valid: c.refresh(GReq())
TOK=c.token
drive=build("drive","v3",credentials=c)
doc=drive.files().create(body={"name":"gdoc IN-PLACE PROOF - comments and suggestions",
  "mimeType":"application/vnd.google-apps.document",
  "parents":["1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"]},fields="id",supportsAllDrives=True).execute()
D=doc["id"]; U=f"https://docs.googleapis.com/v1/documents/{D}"

def http(m,u,b=None):
    data=json.dumps(b).encode() if b is not None else None
    r=urllib.request.Request(u,data=data,method=m,headers={
      "Authorization":f"Bearer {TOK}","Content-Type":"application/json"})
    try:
        with urllib.request.urlopen(r,timeout=60) as x: return x.status,x.read().decode()
    except urllib.error.HTTPError as e: return e.code,e.read().decode()

BODY=("Supplier review\n"
      "Altery reviews each supplier annually and records the outcome in the register.\n"
      "The safeguarding account is reconciled on the last business day of each month.\n")
http("POST",U+":batchUpdate",{"requests":[{"insertText":{"location":{"index":1},"text":BODY}}]})

full=json.loads(http("GET",U)[1])
text="".join((pe.get("textRun") or {}).get("content","")
    for el in full["body"]["content"] for pe in (el.get("paragraph") or {}).get("elements",[]))
def span(needle):
    off=text.index(needle); return off+1, off+1+len(needle)

# 1. a comment anchored to an exact phrase
a,b_=span("annually")
s,_=http("POST",U+":batchUpdate",{"requests":[{"insertComment":{
    "range":{"startIndex":a,"endIndex":b_},
    "content":"gdoc: the 2026-08 policy moved critical suppliers to a six-month cycle. Proposing that change alongside this note."}}]})
print("insertComment on 'annually'   ->",s)

# 2. a suggestion in the same document
s,_=http("POST",U+":batchUpdate",{
  "requests":[{"insertText":{"location":{"index":span("supplier annually")[0]},"text":"critical "}}],
  "writeControl":{"writeMode":"SUGGEST"}})
print("SUGGEST insert 'critical '    ->",s)

# 3. a reply into the thread
inline=json.loads(http("GET",U+"?suggestionsViewMode=SUGGESTIONS_INLINE")[1])
s,bb=http("GET",U+"?suggestionsViewMode=SUGGESTIONS_INLINE&commentsViewMode=COMMENTS_VIEW_MODE_INCLUDED&includeTabsContent=true")
print("commentsViewMode read         ->",s, "comments present" if '"comments"' in bb else bb[:90])
cid=None
if s==200:
    m=re.search(r'"commentId"\s*:\s*"([^"]+)"', bb) or re.search(r'"suggestionId"\s*:\s*"([^"]+)"', bb)
    m2=re.findall(r'"id"\s*:\s*"([^"]{6,})"', bb)
    cid = m.group(1) if m else (m2[0] if m2 else None)
if cid:
    s,r=http("POST",U+":batchUpdate",{"requests":[{"addCommentReply":{
        "commentId":cid,"content":"gdoc: replying in the same thread."}}]})
    print("addCommentReply               ->",s, r.replace("\n"," ")[:90])

# 4. THE PROOF: export to docx and read the comment range
req=drive.files().export_media(fileId=D,
    mimeType="application/vnd.openxmlformats-officedocument.wordprocessingml.document")
buf=io.BytesIO(); dl=MediaIoBaseDownload(buf,req); done=False
while not done: _,done=dl.next_chunk()
buf.seek(0)
with zipfile.ZipFile(buf) as z:
    dx=z.read("word/document.xml").decode()
    cx=z.read("word/comments.xml").decode() if "word/comments.xml" in z.namelist() else ""
m=re.search(r'<w:commentRangeStart[^>]*/>(.*?)<w:commentRangeEnd', dx, re.S)
enclosed=re.sub(r'<[^>]+>','',m.group(1)) if m else None
print("\nEXPORT PROBE")
print("  comments.xml present :", bool(cx))
print("  enclosed span        :", repr(enclosed))
print("  w:ins in export      :", bool(re.search(r'<w:ins\b', dx)))
print("\nhttps://docs.google.com/document/d/%s/edit"%D)
