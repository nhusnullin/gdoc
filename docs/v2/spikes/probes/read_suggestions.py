"""List every pending suggestion in a document, with the text it touches.

    read_suggestions.py <document-id-or-url>

Reads with suggestionsViewMode=SUGGESTIONS_INLINE, which is the only view that
carries suggestedInsertionIds and suggestedDeletionIds. Prints readable JSON:
one entry per text run that any suggestion touches.

Empty findings on a document you just wrote through the API is the proof that
writeMode=SUGGEST did nothing. To see the contrast, open the document, make one
edit by hand in Suggesting mode, and run this again: the same script will print
the run with its suggestedInsertionIds filled in.
"""

import json
import re
import sys

import google_auth_httplib2
import httplib2
from googleapiclient.discovery import build

sys.path.insert(0, "/Users/nailkhusnullin/src/personal/gdoc")
from gdoc import oauth  # noqa: E402

DRIVE_SCOPE = "https://www.googleapis.com/auth/drive"


def document_id(argument):
    match = re.search(r"/document/d/([A-Za-z0-9_-]+)", argument)
    return match.group(1) if match else argument


def docs_client():
    credentials = oauth.load([DRIVE_SCOPE])
    http = google_auth_httplib2.AuthorizedHttp(credentials, http=httplib2.Http())
    return build("docs", "v1", http=http, cache_discovery=False)


def findings(document):
    """Every text run carrying a suggestion, in document order."""
    found = []
    for block in document.get("body", {}).get("content", []):
        paragraph = block.get("paragraph")
        if not paragraph:
            continue
        for element in paragraph.get("elements", []):
            run = element.get("textRun")
            if not run:
                continue
            insertions = run.get("suggestedInsertionIds", [])
            deletions = run.get("suggestedDeletionIds", [])
            if not insertions and not deletions:
                continue
            found.append({
                "startIndex": element.get("startIndex"),
                "endIndex": element.get("endIndex"),
                "text": run.get("content", ""),
                "suggestedInsertionIds": insertions,
                "suggestedDeletionIds": deletions,
            })
    return found


def main(argument):
    doc_id = document_id(argument)
    document = docs_client().documents().get(
        documentId=doc_id, suggestionsViewMode="SUGGESTIONS_INLINE"
    ).execute()

    result = {
        "documentId": doc_id,
        "title": document.get("title"),
        "suggestionsViewMode": "SUGGESTIONS_INLINE",
        "suggestedChangesCount": len(document.get("suggestedDocumentStyleChanges", {}))
        + len(document.get("suggestedNamedStylesChanges", {})),
        "findings": findings(document),
    }
    result["findingsCount"] = len(result["findings"])
    print(json.dumps(result, indent=2))
    return result


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    main(sys.argv[1])
