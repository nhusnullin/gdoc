"""Carry the comment threads of one document onto another.

Two things cannot come across, and the thread's own text has to make up for
both.

The anchor is the first. A comment is useful because it points at a span of
text, and the anchor is computed against the document's structure. A restyled
document is the house template, so its structure is different by construction
and there is no anchor to recompute. The quoted sentence is the only pointer
left, and a human can follow it.

The author is the second. Drive creates every comment as the authenticated
user, and offers no field for saying otherwise. Under oauth every copy arrives
authored by Nail. Prefixing each message with the name of whoever wrote it is
what stops the new document from claiming he said all of it.
"""

from dataclasses import dataclass, field
from typing import Iterable

from gdoc.model import Thread


@dataclass(frozen=True)
class CopyResult:
    copied: int = 0
    skipped_resolved: int = 0
    errors: tuple[str, ...] = field(default_factory=tuple)


def thread_body(thread: Thread) -> str:
    """One comment's worth of text: who said what, in the order they said it.

    No [gdoc] marker. The marker means gdoc wrote this, and these are other
    people's words; stamping them would claim authorship of text gdoc only
    carried.

    No markdown check either. reply.assert_plain_text guards what gdoc writes,
    because Docs renders a comment literally and gdoc can always rephrase. A
    copied comment is somebody else's text, verbatim, and refusing to carry it
    over an asterisk would lose the comment to protect its formatting.
    """
    lines = [f"{thread.author_name}: {thread.content}"]
    lines += [f"{reply.author_name}: {reply.content}" for reply in thread.replies]
    said = "\n".join(lines)
    if thread.quoted:
        return f'On "{thread.quoted}":\n\n{said}'
    return said


def copy_comments(drive, threads: Iterable[Thread], doc_id: str) -> CopyResult:
    """Create one unanchored comment per open thread.

    Resolved threads are dropped, and counted, because a run that carries four
    of seven threads should say where the other three went.

    Each create is guarded on its own. By the time this runs the document
    exists, so a thread that fails is recorded and the next one is still tried:
    raising here would hide a document Drive has already created, which is the
    one outcome that cannot be undone.
    """
    copied = 0
    skipped = 0
    errors: list[str] = []
    for thread in threads:
        if thread.resolved:
            skipped += 1
            continue
        try:
            drive.comments().create(
                fileId=doc_id,
                body={"content": thread_body(thread)},
                fields="id",
            ).execute()
        except Exception as error:  # noqa: BLE001 - see the docstring
            errors.append(
                f"thread {thread.id} was not copied: "
                f"{type(error).__name__}: {error}"
            )
            continue
        copied += 1
    return CopyResult(copied=copied, skipped_resolved=skipped, errors=tuple(errors))
