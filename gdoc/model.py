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
    created_time: str = ""
    modified_time: str = ""

    @property
    def is_agent(self) -> bool:
        """True when gdoc wrote this reply, under either credential."""
        return self.by_agent or self.by_marker


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
        return any(reply.is_agent for reply in self.replies)

    @property
    def new_replies(self) -> tuple[Reply, ...]:
        """The replies gdoc has not answered yet.

        Everything, when gdoc has never replied here. Otherwise what came after
        its last reply, which is the turn it has not seen. Issue #26: without
        this a whole thread went to `skipped` the moment it was answered once,
        so a new instruction typed inside it was invisible, and the run reported
        `addressed: []`, which reads as "nothing new".

        Position decides, because Drive returns replies in the order they were
        written and a time is not guaranteed to be there. A reply written before
        the answer but edited after it counts as new as well: that is Nail going
        back to his own reply to say what he meant.
        """
        last_agent = -1
        for index, reply in enumerate(self.replies):
            if reply.is_agent:
                last_agent = index
        if last_agent < 0:
            return self.replies
        cutoff = self.replies[last_agent].created_time
        return tuple(
            reply
            for index, reply in enumerate(self.replies)
            if not reply.is_agent
            and (
                index > last_agent
                or (cutoff and reply.modified_time > cutoff)
            )
        )

    @property
    def has_newer_replies(self) -> bool:
        """True when gdoc answered here and something has been said since."""
        return self.has_agent_reply and bool(self.new_replies)


def parse_thread(raw: dict, me_is_agent: bool = True) -> Thread:
    """Build a Thread from one Drive comments.list entry.

    Every field is defensive because the API omits rather than nulls: an
    unanchored comment has no quotedFileContent key at all, and author
    emailAddress is absent for every comment we have seen.

    me_is_agent says whether Drive's `me` flag identifies gdoc. Under
    auth_mode service_account it does. Under oauth it identifies Nail, so
    reading it as gdoc would mark every reply he types himself as already
    answered and skip that thread on every future run. It defaults to true
    because that is what `me` meant before OAuth existed.
    """
    author = raw.get("author") or {}
    quoted_block = raw.get("quotedFileContent") or {}
    replies = tuple(
        Reply(
            id=item.get("id", ""),
            content=item.get("content") or "",
            by_agent=me_is_agent and bool((item.get("author") or {}).get("me")),
            author_name=(item.get("author") or {}).get("displayName") or "unknown",
            by_marker=has_marker(item.get("content") or ""),
            created_time=item.get("createdTime") or "",
            modified_time=item.get("modifiedTime") or "",
        )
        for item in raw.get("replies") or ()
    )
    return Thread(
        id=raw["id"],
        content=raw.get("content") or "",
        author_name=author.get("displayName") or "unknown",
        author_email=author.get("emailAddress"),
        by_agent=me_is_agent and bool(author.get("me")),
        quoted=quoted_block.get("value"),
        resolved=bool(raw.get("resolved")),
        replies=replies,
    )
