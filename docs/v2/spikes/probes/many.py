import json,re,urllib.request,urllib.error
from gauth import creds
from google.auth.transport.requests import Request as GReq
from googleapiclient.discovery import build
c=creds()
if not c.valid: c.refresh(GReq())
TOK=c.token
drive=build("drive","v3",credentials=c)
D=drive.files().create(body={"name":"gdoc MANY SUGGESTIONS probe",
 "mimeType":"application/vnd.google-apps.document",
 "parents":["1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"]},fields="id",supportsAllDrives=True).execute()["id"]
U=f"https://docs.googleapis.com/v1/documents/{D}"
def http(m,u,b=None):
    data=json.dumps(b).encode() if b is not None else None
    r=urllib.request.Request(u,data=data,method=m,headers={"Authorization":f"Bearer {TOK}","Content-Type":"application/json"})
    try:
        with urllib.request.urlopen(r,timeout=90) as x: return x.status,x.read().decode()
    except urllib.error.HTTPError as e: return e.code,e.read().decode()

lines="".join(f"Paragraph {i} is a plain sentence of ordinary length.\n" for i in range(1,13))
http("POST",U+":batchUpdate",{"requests":[{"insertText":{"location":{"index":1},"text":lines}}]})

def body_text():
    j=json.loads(http("GET",U)[1])
    return "".join((pe.get("textRun") or {}).get("content","")
        for el in j["body"]["content"] for pe in (el.get("paragraph") or {}).get("elements",[]))

# 12 suggestions, one per paragraph, each inserted before the word "plain"
made=0
for i in range(1,13):
    t=body_text(); needle=f"Paragraph {i} is a "
    idx=t.index(needle)+len(needle)+1
    s,_=http("POST",U+":batchUpdate",
        {"requests":[{"insertText":{"location":{"index":idx},"text":f"S{i}-"}}],
         "writeControl":{"writeMode":"SUGGEST"}})
    made += (s==200)
print("suggestions created:", made)

def ids():
    j=json.loads(http("GET",U+"?suggestionsViewMode=SUGGESTIONS_INLINE")[1])
    out=[]
    def walk(o):
        if isinstance(o,dict):
            for k,v in o.items():
                if k=="suggestedInsertionIds" and v: out.extend(v)
                walk(v)
        elif isinstance(o,list):
            for x in o: walk(x)
    walk(j); return out

before=ids(); print("distinct ids:", len(set(before)), "of", len(before), "runs")
acc=[before[2], before[6]]; rej=[before[4]]
s1,_=http("POST",U+":batchUpdate",{"requests":[{"acceptSuggestion":{"suggestionId":acc[0]}}]})
s2,_=http("POST",U+":batchUpdate",{"requests":[{"rejectSuggestion":{"suggestionId":rej[0]}}]})
s3,_=http("POST",U+":batchUpdate",{"requests":[{"acceptSuggestion":{"suggestionId":acc[1]}}]})
print(f"accept #3 -> {s1}   reject #5 -> {s2}   accept #7 -> {s3}")

after=ids()
survived=[i for i in before if i in after]
print("ids stable for untouched suggestions:", len(set(survived)), "remain of", len(set(before))-3, "expected")
print("resolved ids gone:", all(x not in after for x in acc+rej))
t=body_text()
print("accepted marks present:", "S3-" in t, "S7-" in t)
print("rejected mark absent  :", "S5-" not in t)
print("untouched still pending:", sum(1 for i in range(1,13) if f'S{i}-' not in t and i not in (3,7,5)))
print("\nhttps://docs.google.com/document/d/%s/edit"%D)
