"""One CLI for every Google side effect.

Everything prints a single JSON object so the skill can read the result without
parsing prose. Reply bodies come from a file rather than an argument, because
they are multi-line plain text and shell quoting would mangle them.
"""

import argparse
import json
import sys
from dataclasses import asdict
from datetime import date
from pathlib import Path

from googleapiclient.errors import HttpError

from gdoc.auth import drive_service
from gdoc.config import load_config
from gdoc.docid import extract_doc_id
from gdoc.export import export_markdown
from gdoc.fetch import fetch_threads
from gdoc.filters import forced_kind, partition
from gdoc.generate import generate
from gdoc.baseline import slugify, write_baseline
from gdoc.model import Thread
from gdoc.pairing import (
    Pairing,
    add_version,
    find_by_doc_id,
    read_pairing,
    slug_for_source,
    write_pairing,
)
from gdoc.pending import append_item, pending_path, recorded_source
from gdoc.reply import post_reply


def _thread_json(thread: Thread) -> dict:
    return {
        "id": thread.id,
        "content": thread.content,
        "author": thread.author_name,
        "quoted": thread.quoted,
        "anchored": thread.is_anchored,
        "forced_kind": forced_kind(thread.content),
    }


def _emit(payload: dict) -> int:
    print(json.dumps(payload, indent=2))
    return 0


def _fail(message: str) -> int:
    print(json.dumps({"error": message}, indent=2))
    return 1


# ---------------------------------------------------------------------------
# Subcommand handlers
# ---------------------------------------------------------------------------


def _file_meta(drive, doc_id: str) -> dict:
    """Fetch file metadata.

    supportsAllDrives is required for documents that live on a Shared Drive.
    Without it Drive answers 404 for a file the caller can genuinely read.
    """
    return (
        drive.files()
        .get(fileId=doc_id, fields="name", supportsAllDrives=True)
        .execute()
    )


def cmd_read(args) -> int:
    doc_id = extract_doc_id(args.url)
    drive = drive_service()
    threads = fetch_threads(drive, doc_id)
    addressed, skipped = partition(threads)
    meta = _file_meta(drive, doc_id)
    return _emit(
        {
            "doc_id": doc_id,
            "name": meta.get("name"),
            "slug": slugify(meta.get("name", "")),
            "addressed": [_thread_json(t) for t in addressed],
            "skipped": [_thread_json(t) for t in skipped],
        }
    )


def cmd_reply(args) -> int:
    doc_id = extract_doc_id(args.doc)
    body = Path(args.body_file).read_text()
    reply_id = post_reply(drive_service(), doc_id, args.comment_id, body)
    return _emit({"reply_id": reply_id, "comment_id": args.comment_id})


def cmd_export(args) -> int:
    """Fetch the document as markdown. Write nothing unless asked.

    stdout is raw markdown, not the JSON envelope the other commands print,
    because the point is to pipe it into diff.
    """
    doc_id = extract_doc_id(args.url)
    markdown = export_markdown(drive_service(), doc_id)
    if args.out:
        out = Path(args.out)
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(markdown)
        return 0
    sys.stdout.write(markdown)
    return 0


def cmd_capture(args) -> int:
    drive = drive_service()
    doc_id = extract_doc_id(args.doc)
    repo_root = Path(args.repo_root)
    threads = {t.id: t for t in fetch_threads(drive, doc_id)}
    thread = threads.get(args.comment_id)
    if thread is None:
        return _fail(f"comment {args.comment_id} not found on {doc_id}")

    source = find_by_doc_id(repo_root, doc_id)
    if source is None and not args.slug:
        return _fail(
            f"no markdown under {repo_root} is paired to {doc_id}. "
            "Pair it with gdoc pair set, or pass --slug to choose a queue directory."
        )
    slug = args.slug or slug_for_source(source)
    label = _source_label(repo_root, source) if source else None
    warning = _check_source_collision(repo_root, slug, label) if label else None

    number = append_item(
        repo_root,
        slug,
        thread,
        doc_id=doc_id,
        today=date.today().isoformat(),
        source=label,
    )
    result = {"item": number, "slug": slug, "source": label, "comment_id": args.comment_id}
    if warning:
        result["source_collision_warning"] = warning
    return _emit(result)


def _source_label(repo_root: Path, source: Path) -> str:
    """The source path as recorded in pending.md, relative to the root where it can be."""
    try:
        return str(source.resolve().relative_to(repo_root.resolve()))
    except ValueError:
        return str(source.resolve())


def _check_source_collision(repo_root: Path, slug: str, source: str) -> str | None:
    """Warn when this queue was written for a different source file.

    One directory per source file, so two documents can no longer collide. Two
    source files in different folders sharing a stem still can, and that is now
    the only collision possible.
    """
    recorded = recorded_source(pending_path(repo_root, slug))
    if recorded and recorded != source:
        return (
            f"queue '{slug}' already holds items for {recorded}. "
            "Pass --slug to keep this source in its own directory."
        )
    return None


def cmd_generate(args) -> int:
    config = load_config()
    drive = drive_service()
    result = generate(
        drive,
        Path(args.md),
        args.name,
        Path(args.out),
        folder_id=args.folder_id or config.output_folder_id,
    )
    payload = {k: (str(v) if isinstance(v, Path) else v) for k, v in asdict(result).items()}
    if args.baseline_root and result.doc_id:
        payload["baseline_path"] = str(_write_baseline_for(drive, args, result.doc_id))
    return _emit(payload)


def _write_baseline_for(drive, args, doc_id: str) -> Path:
    """Snapshot the document that was just created.

    This is the one moment the document and the markdown provably match, so it
    is the only honest moment to take the snapshot. force=True is correct here
    and only here: the write is meant to replace the previous version's
    baseline, and refusing would break the loop on the second version.
    """
    markdown = export_markdown(drive, doc_id)
    slug = slug_for_source(Path(args.md))
    return write_baseline(Path(args.baseline_root), slug, markdown, force=True)


# ---------------------------------------------------------------------------
# pair subcommand handlers
# ---------------------------------------------------------------------------


def cmd_pair(args) -> int:
    return args.pair_func(args)


def cmd_pair_show(args) -> int:
    path = Path(args.md)
    pairing = read_pairing(path)
    if pairing is None:
        return _emit({"md": str(path), "paired": False})
    return _emit(
        {
            "md": str(path),
            "paired": True,
            "doc_id": pairing.doc_id,
            "synced": pairing.synced,
            "versions": list(pairing.versions),
        }
    )


def cmd_pair_set(args) -> int:
    path = Path(args.md)
    existing = read_pairing(path)
    if existing is not None and existing.doc_id == args.doc_id:
        # Same document: version history is still valid, preserve it.
        versions = existing.versions
        versions_cleared = 0
    else:
        # Different document (or no prior pairing): old versions describe the
        # wrong document and must not be carried forward.
        versions = ()
        versions_cleared = len(existing.versions) if existing is not None else 0
    pairing = Pairing(doc_id=args.doc_id, synced=args.synced, versions=versions)
    write_pairing(path, pairing)
    return _emit(
        {
            "md": str(path),
            "doc_id": args.doc_id,
            "synced": args.synced,
            "versions_cleared": versions_cleared,
        }
    )


def cmd_pair_add_version(args) -> int:
    path = Path(args.md)
    pairing = read_pairing(path)
    if pairing is None:
        return _fail(f"{path} is not paired to a document")
    new_pairing = add_version(pairing, args.version_id, args.created)
    write_pairing(path, new_pairing)
    return _emit(
        {
            "md": str(path),
            "doc_id": pairing.doc_id,
            "versions_count": len(new_pairing.versions),
        }
    )


def cmd_pair_find(args) -> int:
    found = find_by_doc_id(Path(args.repo_root), args.doc_id)
    return _emit({"doc_id": args.doc_id, "path": str(found) if found else None})


# ---------------------------------------------------------------------------
# Argument parser
# ---------------------------------------------------------------------------


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="gdoc")
    sub = parser.add_subparsers(dest="command", required=True)

    read = sub.add_parser("read", help="list comment threads, partitioned")
    read.add_argument("url")
    read.set_defaults(func=cmd_read)

    reply = sub.add_parser("reply", help="post one plain-text reply")
    reply.add_argument("doc")
    reply.add_argument("comment_id")
    reply.add_argument("--body-file", required=True)
    reply.set_defaults(func=cmd_reply)

    export = sub.add_parser("export", help="fetch the document as markdown")
    export.add_argument("url")
    export.add_argument("--out", help="write to this file instead of stdout")
    export.set_defaults(func=cmd_export)

    capture = sub.add_parser("capture", help="append a global item to pending.md")
    capture.add_argument("doc")
    capture.add_argument("comment_id")
    capture.add_argument("--slug", help="queue directory name, default the paired source's stem")
    capture.add_argument("--repo-root", default=".")
    capture.set_defaults(func=cmd_capture)

    gen = sub.add_parser("generate", help="markdown to a new Google Doc")
    gen.add_argument("--md", required=True)
    gen.add_argument("--name", required=True)
    gen.add_argument("--out", required=True)
    gen.add_argument("--folder-id")
    gen.add_argument(
        "--baseline-root", help="write the new document's baseline under this directory"
    )
    gen.set_defaults(func=cmd_generate)

    pair = sub.add_parser("pair", help="manage document-to-markdown pairings")
    pair.set_defaults(func=cmd_pair)
    pair_sub = pair.add_subparsers(dest="pair_command", required=True)

    show = pair_sub.add_parser("show", help="show the pairing stored in a markdown file")
    show.add_argument("--md", required=True)
    show.set_defaults(pair_func=cmd_pair_show)

    set_cmd = pair_sub.add_parser("set", help="create or update a pairing")
    set_cmd.add_argument("--md", required=True)
    set_cmd.add_argument("--doc-id", required=True)
    set_cmd.add_argument("--synced")
    set_cmd.set_defaults(pair_func=cmd_pair_set)

    add_version_cmd = pair_sub.add_parser(
        "add-version", help="append a version to an existing pairing"
    )
    add_version_cmd.add_argument("--md", required=True)
    add_version_cmd.add_argument("--version-id", required=True)
    add_version_cmd.add_argument("--created", required=True)
    add_version_cmd.set_defaults(pair_func=cmd_pair_add_version)

    find_cmd = pair_sub.add_parser(
        "find", help="find the markdown file paired to a document id"
    )
    find_cmd.add_argument("--repo-root", default=".")
    find_cmd.add_argument("--doc-id", required=True)
    find_cmd.set_defaults(pair_func=cmd_pair_find)

    return parser


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------


def main(argv=None) -> int:
    args = build_parser().parse_args(argv if argv is not None else sys.argv[1:])
    try:
        return args.func(args)
    except (ValueError, OSError, RuntimeError) as error:
        return _fail(str(error))
    except HttpError as error:
        return _fail(f"Drive API {error.resp.status}: {error.reason}")


if __name__ == "__main__":
    sys.exit(main())
