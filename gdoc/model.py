"""Comment threads as immutable values.

replies is a tuple rather than a list so Thread stays hashable and can go in a
set, which the filters rely on.
"""

from dataclasses import dataclass

from gdoc.marker import has_marker


@dataclass(frozen=True)
class Reply:
    id: str
    content: str
    by_agent: bool
    author_name: str = "unknown"
    by_marker: bool = False


@dataclass(frozen=True)
class Thread:
    id: str
    content: str
    author_name: str
    author_email: str | None
    by_agent: bool
    quoted: str | None
    resolved: bool
    replies: tuple[Reply, ...]

    @property
    def is_anchored(self) -> bool:
        return self.quoted is not None

    @property
    def has_agent_reply(self) -> bool:
        """True when gdoc has already answered here.

        Two signals, because neither covers both credentials. The marker is the
        one that works under OAuth, where `me` is Nail. `me` is kept for threads
        the service account answered before the marker existed: dropping it
        would repost on every one of them.
        """
        return any(reply.by_agent or reply.by_marker for reply in self.replies)


def parse_thread(raw: dict) -> Thread:
    """Build a Thread from one Drive comments.list entry.

    Every field is defensive because the API omits rather than nulls: an
    unanchored comment has no quotedFileContent key at all, and author
    emailAddress is absent for every comment we have seen.
    """
    author = raw.get("author") or {}
    quoted_block = raw.get("quotedFileContent") or {}
    replies = tuple(
        Reply(
            id=item.get("id", ""),
            content=item.get("content") or "",
            by_agent=bool((item.get("author") or {}).get("me")),
            author_name=(item.get("author") or {}).get("displayName") or "unknown",
            by_marker=has_marker(item.get("content") or ""),
        )
        for item in raw.get("replies") or ()
    )
    return Thread(
        id=raw["id"],
        content=raw.get("content") or "",
        author_name=author.get("displayName") or "unknown",
        author_email=author.get("emailAddress"),
        by_agent=bool(author.get("me")),
        quoted=quoted_block.get("value"),
        resolved=bool(raw.get("resolved")),
        replies=replies,
    )
