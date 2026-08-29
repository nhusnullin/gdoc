import json, urllib.request, urllib.error
from gauth import creds
from google.auth.transport.requests import Request as GReq
from googleapiclient.discovery import build

c = creds()
if not c.valid: c.refresh(GReq())
TOK = c.token
drive = build("drive","v3",credentials=c)

def http(method, url, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method, headers={
        "Authorization": f"Bearer {TOK}", "Content-Type":"application/json"})
    try:
        with urllib.request.urlopen(req, timeout=60) as r: return r.status, r.read().decode()
    except urllib.error.HTTPError as e: return e.code, e.read().decode()

doc = drive.files().create(body={"name":"gdoc PREVIEW GATE proof",
    "mimeType":"application/vnd.google-apps.document",
    "parents":["1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"]},
    fields="id", supportsAllDrives=True).execute()
d = doc["id"]; U=f"https://docs.googleapis.com/v1/documents/{d}"
print("doc:", d)

http("POST", U+":batchUpdate", {"requests":[{"insertText":{"location":{"index":1},
     "text":"The supplier register is reviewed annually by the operations team.\n"}}]})

s,b = http("POST", U+":batchUpdate", {
  "requests":[{"insertText":{"location":{"index":30},"text":"critical "}}],
  "writeControl":{"writeMode":"SUGGEST"}})
print("SUGGEST insert        ->", s)

s,b = http("POST", U+":batchUpdate", {
  "requests":[{"deleteContentRange":{"range":{"startIndex":1,"endIndex":4}}}],
  "writeControl":{"writeMode":"SUGGEST"}})
print("SUGGEST delete        ->", s)

s,b = http("POST", U+":batchUpdate", {"requests":[{"insertComment":{
    "location":{"index":10},
    "comment":{"content":"gdoc: proposing 'critical' here, per the 2026-08 policy."}}}]})
print("insertComment         ->", s, b.replace("\n"," ")[:150])

s,b = http("GET", U+"?suggestionsViewMode=SUGGESTIONS_INLINE")
j = json.loads(b) if s==200 else {}
ins = json.dumps(j).count("suggestedInsertionIds")
dele = json.dumps(j).count("suggestedDeletionIds")
sids = set()
def walk(o):
    if isinstance(o,dict):
        for k,v in o.items():
            if k in ("suggestedInsertionIds","suggestedDeletionIds") and v: sids.update(v)
            walk(v)
    elif isinstance(o,list):
        for x in o: walk(x)
walk(j)
print(f"SUGGESTIONS_INLINE    -> insertionIds fields {ins}, deletionIds {dele}, ids {sorted(sids)[:4]}")

s,b = http("GET", U+"?commentsViewMode=COMMENTS_VIEW_MODE_INCLUDED&includeTabsContent=true")
print("commentsViewMode      ->", s, ("comments key present" if s==200 and '"comments"' in b else b.replace("\n"," ")[:120]))

if sids:
    one = sorted(sids)[0]
    s,b = http("POST", U+":batchUpdate", {"requests":[{"acceptSuggestion":{"suggestionId":one}}]})
    print("acceptSuggestion      ->", s, b.replace("\n"," ")[:120])
print("\nhttps://docs.google.com/document/d/%s/edit" % d)
