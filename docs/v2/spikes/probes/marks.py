import json
from gauth import creds
from googleapiclient.discovery import build

FOLDER = "1w0SresizE9Kr810VZRJwX4JtDBF4OqNr"
c = creds()
drive = build("drive", "v3", credentials=c)
docs  = build("docs", "v1", credentials=c)

doc = drive.files().create(
    body={"name": "gdoc PROPOSAL MARKING styles",
          "mimeType": "application/vnd.google-apps.document", "parents": [FOLDER]},
    fields="id", supportsAllDrives=True).execute()
did = doc["id"]

BODY = """Proposal marking styles

How gdoc shows you what it changed

Below, ordinary body text is untouched. Text gdoc proposes to ADD is green. Text gdoc proposes to REMOVE is red and struck through. Nothing is highlighted.

1 - An insertion on its own

Altery UK holds the card relationship and issues the e-money for the programme, and issuance is just in time rather than the named role only.

2 - A deletion on its own

The funding crypto assets are held in the custody of the Wallet Provider, not the customer, and this has not changed since the March position paper.

3 - Both in one sentence

ASV holds its own customer relationship with the same person for the crypto leg only, namely the standing mandate to pull from the smart contract and deliver fiat.

4 - Colour swatches, so you can pick the exact shade

Insertion green A. Insertion green B. Insertion green C.

Deletion red A. Deletion red B. Deletion red C.
"""

docs.documents().batchUpdate(documentId=did, body={
    "requests": [{"insertText": {"location": {"index": 1}, "text": BODY}}]}).execute()

def find(doc_json, needle):
    """(start, end) of the first occurrence of needle in the body text."""
    pos = 0
    for el in doc_json["body"]["content"]:
        for pe in (el.get("paragraph") or {}).get("elements", []):
            tr = pe.get("textRun")
            if not tr:
                continue
            txt = tr["content"]
            i = txt.find(needle)
            if i >= 0:
                return pe["startIndex"] + i, pe["startIndex"] + i + len(needle)
    raise KeyError(needle)

GREEN  = {"color": {"rgbColor": {"red": 0.18, "green": 0.60, "blue": 0.35}}}
GREEN_B= {"color": {"rgbColor": {"red": 0.30, "green": 0.72, "blue": 0.42}}}
GREEN_C= {"color": {"rgbColor": {"red": 0.42, "green": 0.80, "blue": 0.52}}}
RED    = {"color": {"rgbColor": {"red": 0.80, "green": 0.25, "blue": 0.22}}}
RED_B  = {"color": {"rgbColor": {"red": 0.88, "green": 0.42, "blue": 0.38}}}
RED_C  = {"color": {"rgbColor": {"red": 0.93, "green": 0.58, "blue": 0.55}}}

d = docs.documents().get(documentId=did).execute()

def style(needle, colour, strike=False):
    s, e = find(d, needle)
    ts = {"foregroundColor": colour}
    fields = "foregroundColor"
    if strike:
        ts["strikethrough"] = True
        fields += ",strikethrough"
    return {"updateTextStyle": {"range": {"startIndex": s, "endIndex": e},
                                "textStyle": ts, "fields": fields}}

reqs = [
    style("and issuance is just in time rather than the named role only", GREEN),
    style("of the Wallet Provider, not the customer,", RED, strike=True),
    style("for the crypto leg only", GREEN),
    style("and deliver fiat", RED, strike=True),
    style("Insertion green A.", GREEN),
    style("Insertion green B.", GREEN_B),
    style("Insertion green C.", GREEN_C),
    style("Deletion red A.", RED, strike=True),
    style("Deletion red B.", RED_B, strike=True),
    style("Deletion red C.", RED_C, strike=True),
]
docs.documents().batchUpdate(documentId=did, body={"requests": reqs}).execute()

# headings
d = docs.documents().get(documentId=did).execute()
h = []
for label, style_name in [("Proposal marking styles", "TITLE"),
                          ("How gdoc shows you what it changed", "SUBTITLE"),
                          ("1 - An insertion on its own", "HEADING_2"),
                          ("2 - A deletion on its own", "HEADING_2"),
                          ("3 - Both in one sentence", "HEADING_2"),
                          ("4 - Colour swatches, so you can pick the exact shade", "HEADING_2")]:
    s, e = find(d, label)
    h.append({"updateParagraphStyle": {"range": {"startIndex": s, "endIndex": e},
                                       "paragraphStyle": {"namedStyleType": style_name},
                                       "fields": "namedStyleType"}})
docs.documents().batchUpdate(documentId=did, body={"requests": h}).execute()

print("https://docs.google.com/document/d/%s/edit" % did)
