import json, os
from google.oauth2.credentials import Credentials
from google.auth.transport.requests import Request
from googleapiclient.discovery import build

TOKEN = os.path.expanduser("~/.config/gdoc-agent/oauth-token.json")

DENY = {
 "1mi_p-JMsf-98MEVoa2LLRRjQuR9Ui8zW1Pq1Kp8Yguc",
 "1gVk9TA3FYJo_badoc2PEYoy5TtkgZ_sp0z3xEm4Yuy0",
 "1aN8WLaesO18m8KKXADlYOymMAv7GzijooctZOUWjRHA",
 "1faFtGazDmaemMNs-qE83ns4yz83Wy0Den36X9uMwhkw",
 "1BZZm6WRA8lqFxN_IeaZwLOy83h3t6nKMltsLGdKNsi8",
}
DENY_NAME_SUBSTR = ["revision-history test","in-place styling test","HOUSE TEMPLATE","PUBLISH-BY-COPY","footnote and anchor test"]

def assert_ok(drive, fid):
    assert fid not in DENY, f"DENYLIST id {fid}"
    meta = drive.files().get(fileId=fid, fields="id,name").execute()
    for s in DENY_NAME_SUBSTR:
        assert s.lower() not in meta["name"].lower(), f"DENYLIST name {meta['name']}"
    return meta

def creds(scopes=None):
    c = Credentials.from_authorized_user_file(TOKEN, scopes or ["https://www.googleapis.com/auth/drive"])
    if not c.valid and c.refresh_token:
        c.refresh(Request())
        open(TOKEN,"w").write(c.to_json())
    return c

def drive():
    return build("drive","v3",credentials=creds(),cache_discovery=False)

def docs():
    return build("docs","v1",credentials=creds(),cache_discovery=False)
