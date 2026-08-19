"""Publish a house-styled copy of a document gdoc knows nothing about.

The document has no queue, no paired markdown and no baseline. Nothing here
creates any of those: the pulled markdown is an intermediate and it is deleted,
so a restyle leaves the root exactly as it found it. The original is read and
never written to.

One function rather than a sequence the skill composes, and the reason is the
guard. This reads the source document, creates a new one, and then writes
comments onto the new one. Inside one client the source and the folder are ids
the command was handed, and the new document's id is learned from the create
that made it, so gdoc/guard.py needs no third door. Split across three CLI runs
the last would have to be handed an id from outside.
"""

import shutil
import tempfile
from dataclasses import asdict, dataclass, field
from pathlib import Path

import yaml

from gdoc.comments import CopyResult
from gdoc.comments import copy_comments as copy_them
from gdoc.export import export_with_media
from gdoc.fetch import fetch_threads
from gdoc.generate import generate

def front_matter(title: str) -> str:
    """The shortest block that publishes, and only the title in it.

    The template's own defaults cover everything else, and an owner or a
    classification guessed from nothing would reach the cover of a document
    other people read.

    yaml.safe_dump rather than an f-string. The title here is a Drive document's
    name, written by somebody who has never seen a line of YAML, and a colon in
    it is the ordinary case: "Q3: Roadmap" hand-built is not a mapping, and the
    run dies parsing its own front matter before it publishes anything.
    """
    return "---\n" + yaml.safe_dump(
        {"title": title}, default_flow_style=False, allow_unicode=True
    ) + "---\n\n"

NO_FOLDER = (
    "no folder to publish into. Pass --folder-id with the Drive folder URL, or "
    "set output_folder_id in ~/.config/gdoc-agent/config.json."
)

NO_TITLE = (
    "the document has no name, so there is nothing to put on the cover. "
    "Pass --title with the title it should carry."
)


@dataclass(frozen=True)
class RestyleResult:
    source_doc_id: str
    source_name: str
    title: str
    folder_id: str
    doc_id: str | None = None
    link: str | None = None
    # None rather than 0 when nothing was copied: "you asked me not to" and
    # "there were none" are different answers.
    comments_copied: int | None = None
    comments_skipped_resolved: int | None = None
    comment_errors: tuple[str, ...] = field(default_factory=tuple)
    image_warnings: tuple[str, ...] = field(default_factory=tuple)
    drift: dict | None = None
    reason: str | None = None
    # The temp directory, kept and named only when something failed, because
    # then the pulled markdown is the only thing the run produced.
    workdir: str | None = None

    def as_json(self) -> dict:
        return asdict(self)


def source_name(drive, doc_id: str) -> str:
    """The document's own name. supportsAllDrives, or a Shared Drive answers 404."""
    meta = (
        drive.files()
        .get(fileId=doc_id, fields="name", supportsAllDrives=True)
        .execute()
    )
    return meta.get("name") or ""


def survey(drive, doc_id: str, *, me_is_agent: bool = True) -> dict:
    """What the run would do, without doing any of it.

    The confirmation the skill shows names the folder and how many threads come
    across, and both have to be on screen before a document exists.
    """
    threads = fetch_threads(drive, doc_id, me_is_agent=me_is_agent)
    return {
        "source_doc_id": doc_id,
        "source_name": source_name(drive, doc_id),
        "threads_open": sum(1 for t in threads if not t.resolved),
        "threads_resolved": sum(1 for t in threads if t.resolved),
    }


def restyle(
    drive,
    doc_id: str,
    *,
    folder_id: str | None,
    title: str | None = None,
    template: str | None = None,
    copy_comments: bool = True,
    me_is_agent: bool = True,
) -> RestyleResult:
    """Pull, publish, then carry the open comment threads across.

    The folder check is first, before the export, and the title check second, so
    a run that cannot publish does not spend a download and a pandoc conversion
    finding that out.

    Comments are copied only once generate reports a doc_id. Nothing else
    decides whether a document exists, here or in cli.cmd_generate.
    """
    if not folder_id:
        raise ValueError(NO_FOLDER)

    name = source_name(drive, doc_id)
    cover = title or name
    if not cover.strip():
        raise ValueError(NO_TITLE)
    threads = fetch_threads(drive, doc_id, me_is_agent=me_is_agent) if copy_comments else ()

    workdir = Path(tempfile.mkdtemp(prefix="gdoc-restyle-"))
    md_path = workdir / "source.md"
    try:
        export = export_with_media(drive, doc_id, workdir / "media", md_path)
        md_path.write_text(front_matter(cover) + export.markdown)
        result = generate(
            drive,
            md_path,
            cover,
            workdir / "restyled.docx",
            folder_id=folder_id,
            template=template,
            title=cover,
        )
    except Exception as error:
        # No document exists either way: generate reports a refused upload as a
        # reason rather than raising, and it bins its own measuring copy before
        # anything escapes. So the only question left is whether the pull is
        # worth keeping.
        #
        # It is, once it is on disk. A foreign document is exactly where the
        # template build chokes, and then the markdown is the only thing to look
        # at. The path goes into the message because nothing else will carry it:
        # this raises rather than returning a result, so there is no workdir key
        # for the skill to read. Before that point the directory is empty, and
        # leaving it would leak one per failure, unnamed and unreported.
        if md_path.exists():
            raise RuntimeError(
                f"{type(error).__name__}: {error}. The document was pulled but "
                f"not published. The markdown is in {workdir}."
            ) from error
        shutil.rmtree(workdir, ignore_errors=True)
        raise

    copied = None
    if result.doc_id and copy_comments:
        try:
            copied = copy_them(drive, threads, result.doc_id)
        except Exception as error:  # noqa: BLE001 - see below
            # The document exists by now, so nothing below may withhold its id.
            # copy_comments already guards each create on its own, so reaching
            # here means the loop itself failed, which is the kind of fault a
            # narrow catch would let escape and hide a real document behind.
            # cli.cmd_generate defends its baseline and pairing writes the same
            # way, for the same reason.
            copied = CopyResult(
                errors=(f"no comment was copied: {type(error).__name__}: {error}",)
            )

    if result.doc_id:
        shutil.rmtree(workdir, ignore_errors=True)

    return RestyleResult(
        source_doc_id=doc_id,
        source_name=name,
        title=cover,
        folder_id=folder_id,
        doc_id=result.doc_id,
        link=result.link,
        comments_copied=copied.copied if copied is not None else None,
        comments_skipped_resolved=(
            copied.skipped_resolved if copied is not None else None
        ),
        comment_errors=copied.errors if copied is not None else (),
        image_warnings=tuple(export.warnings),
        drift=result.drift,
        reason=result.reason,
        workdir=None if result.doc_id else str(workdir),
    )
