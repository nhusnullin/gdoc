"""One CLI for every Google side effect.

Everything prints a single JSON object so the skill can read the result without
parsing prose. Reply bodies come from a file rather than an argument, because
they are multi-line plain text and shell quoting would mangle them.
"""

import argparse
import json
import sys
from dataclasses import asdict, replace
from datetime import date
from pathlib import Path

from googleapiclient.errors import HttpError

from gdoc import oauth, render
from gdoc.auth import DEFAULT_KEY_PATH, SCOPES, drive_service
from gdoc.config import Config, load_config
from gdoc.docid import extract_doc_id, extract_folder_id
from gdoc.export import export_markdown
from gdoc.fetch import fetch_threads
from gdoc.filters import forced_kind, is_addressed, partition
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
from gdoc.render import frontmatter, profiles
from gdoc.reply import post_reply


# Enough of an existing reply to recognise it, not enough to bloat the payload.
_REPLY_PREVIEW = 400


def _reply_json(reply) -> dict:
    return {
        "id": reply.id,
        "author": reply.author_name,
        "content": reply.content[:_REPLY_PREVIEW],
        "by_gdoc": reply.by_agent or reply.by_marker,
    }


def _thread_json(thread: Thread) -> dict:
    return {
        "id": thread.id,
        "content": thread.content,
        "author": thread.author_name,
        "quoted": thread.quoted,
        "anchored": thread.is_anchored,
        "forced_kind": forced_kind(thread.content),
        "marked": is_addressed(thread.content),
        "answered": thread.has_agent_reply,
        "replies": [_reply_json(reply) for reply in thread.replies],
    }


def _emit(payload: dict) -> int:
    print(json.dumps(payload, indent=2))
    return 0


def _fail(message: str) -> int:
    return _fail_with({"error": message})


def _fail_with(payload: dict) -> int:
    """Fail with more than a message, for a refusal a caller can act on."""
    print(json.dumps(payload, indent=2))
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


def _configured_mode() -> str:
    try:
        return load_config().auth_mode
    except (FileNotFoundError, ValueError):
        return "oauth"


def _me_is_agent() -> bool:
    """Whether Drive's `me` flag identifies gdoc rather than the person.

    True under the service account, false under oauth, where the credential is
    Nail and his own replies would otherwise read as gdoc's.
    """
    return _configured_mode() == "service_account"


def cmd_read(args) -> int:
    doc_id = extract_doc_id(args.url)
    drive = drive_service(doc_ids=doc_id)
    threads = fetch_threads(drive, doc_id, me_is_agent=_me_is_agent())
    addressed, skipped = partition(threads, include_unmarked=args.all)
    meta = _file_meta(drive, doc_id)
    return _emit(
        {
            "doc_id": doc_id,
            "name": meta.get("name"),
            "slug": slugify(meta.get("name", "")),
            "mode": "all" if args.all else "marked",
            "addressed": [_thread_json(t) for t in addressed],
            "skipped": [_thread_json(t) for t in skipped],
        }
    )


def cmd_reply(args) -> int:
    doc_id = extract_doc_id(args.doc)
    body = Path(args.body_file).read_text()
    reply_id = post_reply(drive_service(doc_ids=doc_id), doc_id, args.comment_id, body)
    return _emit({"reply_id": reply_id, "comment_id": args.comment_id})


def cmd_export(args) -> int:
    """Fetch the document as markdown. Write nothing unless asked.

    stdout is raw markdown, not the JSON envelope the other commands print,
    because the point is to pipe it into diff.
    """
    doc_id = extract_doc_id(args.url)
    markdown = export_markdown(drive_service(doc_ids=doc_id), doc_id)
    if args.out:
        out = Path(args.out)
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(markdown)
        return 0
    sys.stdout.write(markdown)
    return 0


def cmd_capture(args) -> int:
    # The id first: the client is built to reach this document and no other.
    doc_id = extract_doc_id(args.doc)
    drive = drive_service(doc_ids=doc_id)
    repo_root = Path(args.repo_root)
    threads = {
        t.id: t for t in fetch_threads(drive, doc_id, me_is_agent=_me_is_agent())
    }
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
    """Publish the markdown, then record what happened. The upload wins either way.

    Once result.doc_id is set, a document was really created, and everything
    after that point is recording facts about it: a baseline snapshot and a
    pairing version. Each of those two writes gets its own broad except, on
    purpose, and each is guarded on its own rather than the two sharing one
    try, so a failure in one does not stop the other from being recorded.

    _write_baseline_for makes a Drive network call (export_markdown) and a disk
    write (write_baseline); either can fail in ways that are not an OSError, such
    as HttpError on a dropped connection or an expired token, or PandocNotFound
    if the plain export falls back to pandoc and it is missing. _record_version
    calls read_pairing and write_pairing, which can fail below the OSError layer
    too: a UnicodeDecodeError reading a note saved with bad bytes, or an
    AttributeError or ValueError from front matter a hand edit or a bad merge
    left as a list instead of a mapping. A narrow catch on either would let its
    failure escape uncaught and crash before the payload below is ever printed,
    hiding a document that Drive already created. That is the one outcome this
    function must never produce, so both catches are as wide as the exceptions
    this code can throw, not as wide as the ones a narrower reading predicted.
    gdoc/pairing.py's find_by_doc_id and gdoc/generate.py's _check_drift catch
    this broadly for the same reason. Each exception is named verbatim in its
    own payload key (baseline_error, pairing_error), so a real programming error
    is still visible rather than silently reported as an ordinary failure, and
    the two stay distinguishable from each other.
    """
    folder_id = extract_folder_id(args.folder_id) if args.folder_id else None
    config = _config_for(folder_id)
    template = args.template or config.template
    master = None if template == profiles.NO_TEMPLATE else template
    md_path = Path(args.md)
    folder_id = folder_id or config.output_folder_id
    try:
        name = args.name or _version_name(md_path, master, args.title)
        # No input document. The output folder is the one file this command was
        # given; every other id it touches is one it created.
        drive = drive_service(doc_ids=[folder_id] if folder_id else ())
        result = generate(
            drive,
            md_path,
            name,
            Path(args.out),
            folder_id=folder_id,
            template=master,
            title=args.title,
        )
    except frontmatter.MissingTitle as error:
        return _fail_with(_missing_title(error, md_path))
    payload = {k: (str(v) if isinstance(v, Path) else v) for k, v in asdict(result).items()}
    if args.baseline_root and result.doc_id:
        payload["slug"] = slug_for_source(md_path)
        try:
            payload["baseline_path"] = str(_write_baseline_for(drive, args, result.doc_id))
        except Exception as error:  # noqa: BLE001 - see the docstring
            payload["baseline_error"] = (
                f"the document was published but its baseline was not recorded: "
                f"{type(error).__name__}: {error}"
            )
        try:
            payload["version"] = _record_version(md_path, result.doc_id)
        except Exception as error:  # noqa: BLE001 - see the docstring
            payload["pairing_error"] = (
                f"the document was published but its version was not recorded: "
                f"{type(error).__name__}: {error}"
            )
    return _emit(payload)


def _config_for(folder_id: str | None) -> Config:
    """The config file, or plain defaults when the caller already named the folder.

    The output folder is the one setting generate cannot work out for itself, so
    a run that was handed one has everything it needs and must not be refused for
    a file it no longer reads. Everything else in the config has a default: the
    template falls back to the bundled house style, so publishing without a config
    file still produces a house-styled document rather than a plain one.

    With no folder on the command line the file is the only source left, so its
    absence stays the error it always was, naming what to write and where.
    """
    if folder_id is None:
        return load_config()
    try:
        return load_config()
    except FileNotFoundError:
        return Config()


def _record_version(md_path: Path, doc_id: str) -> int:
    """Write the pairing for a document that was just created. Returns its number.

    This is a fact, not a policy: a document with this id was created from this
    markdown today. Which folder, which filename and which version label stay
    with the skill. The baseline write already establishes that recording facts
    at this moment is generate's job.

    Gated on --baseline-root, which already means "this is a tracked version of
    a tracked file", so a one-off document does not stamp frontmatter on a note.
    """
    pairing = read_pairing(md_path) or Pairing(doc_id=doc_id)
    created = date.today().isoformat()
    updated = add_version(replace(pairing, doc_id=doc_id), doc_id, created)
    write_pairing(md_path, updated)
    return len(updated.versions)


def _version_name(md_path: Path, template: str | None, title: str | None = None) -> str:
    """'<cover title> v<n>'. n is the recorded version count plus one.

    An unpaired note has no recorded versions, so it gets v1. With no template
    there may be no front matter to read, so the file stem stands in for the
    title.
    """
    pairing = read_pairing(md_path)
    number = len(pairing.versions) + 1 if pairing else 1
    if template is None:
        return f"{md_path.stem} v{number}"
    return f"{render.meta_for(md_path, title=title)['cover_title']} v{number}"


def _missing_title(error: frontmatter.MissingTitle, md_path: Path) -> dict:
    """The refusal payload. It names the file, a candidate, and the ways out.

    The candidate is a suggestion for a human to approve, never a title the tool
    uses. Nothing here writes anything into the note.
    """
    return {
        "error": str(error),
        "missing": "title",
        "md": str(md_path),
        "suggested_title": error.candidate,
        "suggested_from": error.source,
        "hint": (
            "Ask Nail to approve a title before using this suggestion, then add "
            "'title: <the approved title>' to the front matter of the note or pass "
            "it with --title to publish once without editing the note. "
            "--template none publishes without the house style and needs no title"
        ),
    }


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
    # The pure add_version is left alone. Moving the pointer belongs here,
    # so no command can leave gdoc: naming an old version.
    new_pairing = add_version(replace(pairing, doc_id=args.version_id),
                               args.version_id, args.created)
    write_pairing(path, new_pairing)
    return _emit(
        {
            "md": str(path),
            "doc_id": new_pairing.doc_id,
            "versions_count": len(new_pairing.versions),
        }
    )


def cmd_pair_find(args) -> int:
    found = find_by_doc_id(Path(args.repo_root), args.doc_id)
    return _emit({"doc_id": args.doc_id, "path": str(found) if found else None})


# ---------------------------------------------------------------------------
# auth subcommand handlers
# ---------------------------------------------------------------------------


def cmd_auth(args) -> int:
    return args.auth_func(args)


def cmd_auth_login(args) -> int:
    credentials = oauth.login(
        SCOPES, client_path=args.client, token_path=args.token
    )
    user = oauth.account(drive_service(credentials=credentials))
    return _emit(
        {
            "logged_in": True,
            "account": user.get("emailAddress"),
            "name": user.get("displayName"),
            "token_path": str(args.token or oauth.DEFAULT_TOKEN_PATH),
        }
    )


def cmd_auth_status(args) -> int:
    """Say what is set up and what is broken. Never fail.

    The except is deliberately broad. This command exists to report a broken
    credential, so any exception is its output rather than its failure.
    """
    payload = {
        "auth_mode": _configured_mode(),
        "token_path": str(oauth.DEFAULT_TOKEN_PATH),
        "client_path": str(oauth.DEFAULT_CLIENT_PATH),
        "key_path": str(DEFAULT_KEY_PATH),
        "revoke_url": "https://myaccount.google.com/permissions",
    }
    try:
        user = oauth.account(drive_service())
        payload["account"] = user.get("emailAddress")
        payload["name"] = user.get("displayName")
        payload["ready"] = True
    except Exception as error:  # noqa: BLE001
        payload["ready"] = False
        payload["problem"] = str(error)
    return _emit(payload)


def cmd_auth_logout(args) -> int:
    removed = oauth.logout(token_path=args.token)
    return _emit(
        {
            "logged_out": removed,
            "token_path": str(args.token or oauth.DEFAULT_TOKEN_PATH),
        }
    )


# ---------------------------------------------------------------------------
# Argument parser
# ---------------------------------------------------------------------------


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="gdoc")
    sub = parser.add_subparsers(dest="command", required=True)

    read = sub.add_parser("read", help="list comment threads, partitioned")
    read.add_argument("url")
    read.add_argument(
        "--all",
        action="store_true",
        help="offer every unresolved comment, not only the marked ones",
    )
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
    gen.add_argument("--name", help="default: '<cover title> v<n>'")
    gen.add_argument("--out", required=True)
    gen.add_argument(
        "--folder-id",
        help="Drive folder URL or id to publish into. Given one, no config file is needed",
    )
    gen.add_argument("--template", help="profile name, path, or 'none' for plain pandoc")
    gen.add_argument("--title", help="cover title for this run, without editing the note")
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

    auth = sub.add_parser("auth", help="manage the Google credential")
    auth.set_defaults(func=cmd_auth)
    auth_sub = auth.add_subparsers(dest="auth_command", required=True)

    login_cmd = auth_sub.add_parser("login", help="authorise in a browser as yourself")
    login_cmd.add_argument("--client", help="path to the Desktop OAuth client JSON")
    login_cmd.add_argument("--token", help="where to write the token")
    login_cmd.set_defaults(auth_func=cmd_auth_login)

    status_cmd = auth_sub.add_parser("status", help="show the credential in use")
    status_cmd.set_defaults(auth_func=cmd_auth_status)

    logout_cmd = auth_sub.add_parser("logout", help="delete the local OAuth token")
    logout_cmd.add_argument("--token", help="path to the token to delete")
    logout_cmd.set_defaults(auth_func=cmd_auth_logout)

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
