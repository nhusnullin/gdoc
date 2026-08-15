"""Read every comment thread on a document."""

from gdoc.model import Thread, parse_thread

FIELDS = (
    "nextPageToken,"
    "comments(id,createdTime,modifiedTime,resolved,quotedFileContent(value),"
    "content,author(displayName,emailAddress,me),"
    "replies(id,content,author(displayName,emailAddress,me)))"
)


def fetch_threads(drive, doc_id: str, me_is_agent: bool = True) -> tuple[Thread, ...]:
    """Page through comments.list.

    fields is mandatory on this endpoint, and nextPageToken has to be inside it
    or pagination silently stops after the first page.

    me_is_agent is passed straight to parse_thread. See it there: under oauth
    Drive's `me` is Nail, not gdoc.
    """
    collected: list[Thread] = []
    page_token = None
    while True:
        response = (
            drive.comments()
            .list(
                fileId=doc_id,
                fields=FIELDS,
                pageSize=100,
                includeDeleted=False,
                pageToken=page_token,
            )
            .execute()
        )
        collected.extend(
            parse_thread(raw, me_is_agent) for raw in response.get("comments") or ()
        )
        page_token = response.get("nextPageToken")
        if not page_token:
            return tuple(collected)
