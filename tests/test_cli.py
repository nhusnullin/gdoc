import json
from unittest.mock import MagicMock, patch

import pytest

from gdoc.cli import main


# ---------------------------------------------------------------------------
# read subcommand
# ---------------------------------------------------------------------------


def test_read_returns_one_addressed_list_with_authors(capsys):
    """The marker decides. The author name is a label, so it stays in the payload."""
    threads = [
        {
            "id": "t1",
            "content": "ai: rephrase",
            "author": {"displayName": "Nail Khusnullin", "me": False},
        },
        {
            "id": "t2",
            "content": "ai: check this",
            "author": {"displayName": "William Mejia", "me": False},
        },
    ]
    drive = MagicMock()
    drive.comments().list.return_value.execute.side_effect = [{"comments": threads}]
    drive.files().get.return_value.execute.return_value = {"name": "Test doc"}
    with patch("gdoc.cli.drive_service", return_value=drive):
        exit_code = main(["read", "https://docs.google.com/document/d/1AbC/edit"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert "mine" not in payload
    assert "others" not in payload
    assert [t["author"] for t in payload["addressed"]] == ["Nail Khusnullin", "William Mejia"]


def test_bad_url_exits_nonzero_with_json_error(capsys):
    exit_code = main(["read", "https://example.com/nope"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "error" in payload


# ---------------------------------------------------------------------------
# reply subcommand
# ---------------------------------------------------------------------------


_REAL_DOC_ID = "1tyVhOTw9-bJ99IJTBZoTfWAT6fm3rjGIzfpHQX91hkw"
_REAL_DOC_URL = f"https://docs.google.com/document/d/{_REAL_DOC_ID}/edit"


def test_reply_reads_the_body_from_a_file(capsys, tmp_path):
    body = tmp_path / "body.txt"
    body.write_text("Plain text answer.")
    drive = MagicMock()
    drive.replies().create.return_value.execute.return_value = {"id": "r1"}
    with patch("gdoc.cli.drive_service", return_value=drive):
        exit_code = main(["reply", _REAL_DOC_ID, "t1", "--body-file", str(body)])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["reply_id"] == "r1"


def test_reply_rejects_markdown_and_exits_nonzero(capsys, tmp_path):
    body = tmp_path / "body.txt"
    body.write_text("Has **markdown**.")
    drive = MagicMock()
    with patch("gdoc.cli.drive_service", return_value=drive):
        exit_code = main(["reply", _REAL_DOC_ID, "t1", "--body-file", str(body)])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "markdown" in payload["error"]
    drive.replies().create.assert_not_called()


def test_reply_accepts_a_full_url_and_extracts_the_id(capsys, tmp_path):
    """Passing a URL to reply must route to the same doc id as passing the bare id."""
    body = tmp_path / "body.txt"
    body.write_text("Plain text answer.")
    with patch("gdoc.cli.drive_service"), patch(
        "gdoc.cli.post_reply", return_value="r2"
    ) as mock_post:
        exit_code = main(["reply", _REAL_DOC_URL, "t1", "--body-file", str(body)])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    # The extracted id, not the full URL, must reach post_reply.
    _, positional, _ = mock_post.mock_calls[0]
    assert positional[1] == _REAL_DOC_ID


# ---------------------------------------------------------------------------
# unknown subcommand
# ---------------------------------------------------------------------------


def test_unknown_subcommand_exits_nonzero():
    with pytest.raises(SystemExit):
        main(["nonsense"])


# ---------------------------------------------------------------------------
# export subcommand
# ---------------------------------------------------------------------------


_EXPORT_URL = "https://docs.google.com/document/d/1AbC/edit"
_MARKDOWN = "# Title\n\nBody.\n"


def test_export_prints_markdown_to_stdout(capsys):
    """Raw markdown, not the JSON envelope, so the output can be piped into diff."""
    with patch("gdoc.cli.drive_service"), patch(
        "gdoc.cli.export_markdown", return_value=_MARKDOWN
    ):
        exit_code = main(["export", _EXPORT_URL])
    assert exit_code == 0
    assert capsys.readouterr().out == _MARKDOWN


def test_export_writes_to_the_named_file(capsys, tmp_path):
    out = tmp_path / "fetched.md"
    with patch("gdoc.cli.drive_service"), patch(
        "gdoc.cli.export_markdown", return_value=_MARKDOWN
    ):
        exit_code = main(["export", _EXPORT_URL, "--out", str(out)])
    capsys.readouterr()
    assert exit_code == 0
    assert out.read_text() == _MARKDOWN


def test_export_writes_nothing_when_out_is_absent(capsys, tmp_path):
    with patch("gdoc.cli.drive_service"), patch(
        "gdoc.cli.export_markdown", return_value=_MARKDOWN
    ):
        main(["export", _EXPORT_URL])
    capsys.readouterr()
    assert list(tmp_path.iterdir()) == []


def test_export_no_longer_takes_a_repo_root():
    """Filing is no longer export's job, so the flag must be gone, not ignored."""
    with patch("gdoc.cli.drive_service"), patch(
        "gdoc.cli.export_markdown", return_value=_MARKDOWN
    ), pytest.raises(SystemExit):
        main(["export", _EXPORT_URL, "--repo-root", "/tmp"])


# ---------------------------------------------------------------------------
# generate subcommand
# ---------------------------------------------------------------------------


def _generate_args(tmp_path):
    md = tmp_path / "2026-08-13-topic.md"
    md.write_text("# Source\n")
    out = tmp_path / "out" / "v2.docx"
    return md, out


def _run_generate(tmp_path, result, extra=()):
    """Run generate with the upload stubbed out. Returns the parsed JSON."""
    from gdoc.cli import main as cli_main

    md, out = _generate_args(tmp_path)
    with patch("gdoc.cli.drive_service"), patch("gdoc.cli.load_config") as config, patch(
        "gdoc.cli.generate", return_value=result
    ), patch("gdoc.cli.export_markdown", return_value="# Generated\n"):
        config.return_value.output_folder_id = "0AFolderId"
        exit_code = cli_main(
            [
                "generate",
                "--md",
                str(md),
                "--name",
                "My Doc",
                "--out",
                str(out),
                "--baseline-root",
                str(tmp_path),
                *extra,
            ]
        )
    return exit_code


def _uploaded(tmp_path):
    from gdoc.generate import Result

    return Result(
        docx_path=tmp_path / "out" / "v2.docx",
        doc_id="1NewDocId",
        link="https://docs.google.com/document/d/1NewDocId/edit",
    )


def _not_uploaded(tmp_path):
    from gdoc.generate import Result

    return Result(docx_path=tmp_path / "out" / "v2.docx", reason="no output_folder_id in config")


def test_generate_writes_the_baseline_after_a_successful_upload(capsys, tmp_path):
    exit_code = _run_generate(tmp_path, _uploaded(tmp_path))
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    # The source file's stem, not the document title: the title carries a version
    # number and would fork the directory on every iteration.
    baseline = tmp_path / ".gdoc" / "2026-08-13-topic" / "baseline.md"
    assert baseline.read_text() == "# Generated\n"
    assert payload["baseline_path"] == str(baseline)


def test_generate_writes_no_baseline_when_the_upload_failed(capsys, tmp_path):
    """A baseline for a document that was never created would be diffed against nothing."""
    exit_code = _run_generate(tmp_path, _not_uploaded(tmp_path))
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert not (tmp_path / ".gdoc").exists()
    assert "baseline_path" not in payload


def test_generate_overwrites_an_existing_baseline(capsys, tmp_path):
    """The second version must not be blocked by the first version's snapshot."""
    _run_generate(tmp_path, _uploaded(tmp_path))
    capsys.readouterr()
    exit_code = _run_generate(tmp_path, _uploaded(tmp_path))
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert "error" not in payload


def test_generate_writes_no_baseline_without_a_baseline_root(capsys, tmp_path):
    from gdoc.generate import Result

    md, out = _generate_args(tmp_path)
    result = Result(docx_path=out, doc_id="1NewDocId", link="link")
    with patch("gdoc.cli.drive_service"), patch("gdoc.cli.load_config") as config, patch(
        "gdoc.cli.generate", return_value=result
    ), patch("gdoc.cli.export_markdown") as export:
        config.return_value.output_folder_id = "0AFolderId"
        exit_code = main(["generate", "--md", str(md), "--name", "My Doc", "--out", str(out)])
    capsys.readouterr()
    assert exit_code == 0
    export.assert_not_called()


# ---------------------------------------------------------------------------
# pair subcommand, show mode
# ---------------------------------------------------------------------------


def test_pair_show_returns_pairing_when_paired(capsys, tmp_path):
    md = tmp_path / "policy.md"
    md.write_text("---\ngdoc: docABC\ngdoc_synced: '2026-08-01'\n---\n\nBody.\n")
    exit_code = main(["pair", "show", "--md", str(md)])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["paired"] is True
    assert payload["doc_id"] == "docABC"
    assert payload["synced"] == "2026-08-01"


def test_pair_show_returns_not_paired_for_plain_file(capsys, tmp_path):
    md = tmp_path / "plain.md"
    md.write_text("# No frontmatter\n\nBody.\n")
    exit_code = main(["pair", "show", "--md", str(md)])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["paired"] is False


# ---------------------------------------------------------------------------
# pair subcommand, set mode
# ---------------------------------------------------------------------------


def test_pair_set_creates_pairing_in_file(capsys, tmp_path):
    md = tmp_path / "doc.md"
    md.write_text("# My document\n\nSome content.\n")
    exit_code = main(["pair", "set", "--md", str(md), "--doc-id", "docXYZ"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["doc_id"] == "docXYZ"
    from gdoc.pairing import read_pairing

    written = read_pairing(md)
    assert written is not None
    assert written.doc_id == "docXYZ"
    assert written.synced is None


def test_pair_set_records_synced_date_when_provided(capsys, tmp_path):
    md = tmp_path / "doc.md"
    md.write_text("# Doc\n\nContent.\n")
    exit_code = main(
        ["pair", "set", "--md", str(md), "--doc-id", "docXYZ", "--synced", "2026-08-13"]
    )
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["synced"] == "2026-08-13"
    from gdoc.pairing import read_pairing

    written = read_pairing(md)
    assert written.synced == "2026-08-13"


def test_pair_set_preserves_versions_when_doc_id_matches(capsys, tmp_path):
    md = tmp_path / "doc.md"
    md.write_text(
        "---\ngdoc: docABC\ngdoc_versions:\n  - id: docV1\n    created: '2026-08-01'\n---\n\nBody.\n"
    )
    exit_code = main(
        ["pair", "set", "--md", str(md), "--doc-id", "docABC", "--synced", "2026-08-13"]
    )
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["versions_cleared"] == 0
    from gdoc.pairing import read_pairing

    written = read_pairing(md)
    assert len(written.versions) == 1
    assert written.versions[0]["id"] == "docV1"


def test_pair_set_clears_versions_when_doc_id_differs(capsys, tmp_path):
    md = tmp_path / "doc.md"
    md.write_text(
        "---\ngdoc: docABC\ngdoc_versions:\n  - id: docV1\n    created: '2026-08-01'\n  - id: docV2\n    created: '2026-08-10'\n---\n\nBody.\n"
    )
    exit_code = main(["pair", "set", "--md", str(md), "--doc-id", "docNEW"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["versions_cleared"] == 2
    from gdoc.pairing import read_pairing

    written = read_pairing(md)
    assert written.versions == ()


# ---------------------------------------------------------------------------
# pair subcommand, add-version mode
# ---------------------------------------------------------------------------


def test_pair_add_version_appends_a_version(capsys, tmp_path):
    md = tmp_path / "doc.md"
    md.write_text("---\ngdoc: docABC\n---\n\nBody.\n")
    exit_code = main(
        [
            "pair",
            "add-version",
            "--md",
            str(md),
            "--version-id",
            "docV1",
            "--created",
            "2026-08-01",
        ]
    )
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["versions_count"] == 1
    from gdoc.pairing import read_pairing

    written = read_pairing(md)
    assert len(written.versions) == 1
    assert written.versions[0]["id"] == "docV1"
    assert written.versions[0]["created"] == "2026-08-01"


def test_pair_add_version_fails_when_file_is_not_paired(capsys, tmp_path):
    md = tmp_path / "doc.md"
    md.write_text("# Not paired\n\nBody.\n")
    exit_code = main(
        [
            "pair",
            "add-version",
            "--md",
            str(md),
            "--version-id",
            "v1",
            "--created",
            "2026-08-01",
        ]
    )
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "error" in payload


# ---------------------------------------------------------------------------
# pair subcommand, find mode
# ---------------------------------------------------------------------------


def test_pair_find_defaults_to_the_working_directory():
    """The default is the whole point: no command may name a specific repository."""
    from gdoc.cli import build_parser

    args = build_parser().parse_args(["pair", "find", "--doc-id", "1AbC"])
    assert args.repo_root == "."


def test_pair_find_returns_path_when_matched(capsys, tmp_path):
    subdir = tmp_path / "subdir"
    subdir.mkdir()
    md = subdir / "doc.md"
    md.write_text("---\ngdoc: targetDocId\n---\n\nBody.\n")
    exit_code = main(["pair", "find", "--repo-root", str(tmp_path), "--doc-id", "targetDocId"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["path"] == str(md)
    assert payload["doc_id"] == "targetDocId"


def test_pair_find_returns_null_when_not_found(capsys, tmp_path):
    exit_code = main(["pair", "find", "--repo-root", str(tmp_path), "--doc-id", "missingId"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["path"] is None
    assert payload["doc_id"] == "missingId"


# ---------------------------------------------------------------------------
# HttpError handler
# ---------------------------------------------------------------------------


def test_main_handles_http_error_as_json_error(capsys):
    from googleapiclient.errors import HttpError

    fake_resp = MagicMock()
    fake_resp.status = 403
    error = HttpError(resp=fake_resp, content=b"Forbidden")

    drive = MagicMock()
    drive.files().get.return_value.execute.side_effect = error
    with patch("gdoc.cli.drive_service", return_value=drive), patch(
        "gdoc.cli.fetch_threads", return_value=()
    ):
        exit_code = main(["read", "https://docs.google.com/document/d/1AbC/edit"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "error" in payload
    assert "403" in payload["error"]


# ---------------------------------------------------------------------------
# capture subcommand
# ---------------------------------------------------------------------------


def test_capture_appends_a_global_item(capsys, tmp_path):
    from gdoc.model import Thread

    thread = Thread(
        id="c1",
        content="ai! add a section on refunds",
        author_name="Nail Khusnullin",
        author_email=None,
        by_agent=False,
        quoted="policy text",
        resolved=False,
        replies=(),
    )
    drive = MagicMock()
    with patch("gdoc.cli.drive_service", return_value=drive), patch(
        "gdoc.cli.fetch_threads", return_value=(thread,)
    ):
        exit_code = main(
            [
                "capture",
                "https://docs.google.com/document/d/1AbCdefghijklmnopqrstuvwx/edit",
                "c1",
                "--slug",
                "policy",
                "--repo-root",
                str(tmp_path),
            ]
        )
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["item"] == 1
    assert payload["comment_id"] == "c1"
    pending = tmp_path / ".gdoc" / "policy" / "pending.md"
    assert pending.exists()
    assert "c1" in pending.read_text()


_PAIRED_DOC_ID = "1AbCdefghijklmnopqrstuvwx"
_PAIRED_URL = f"https://docs.google.com/document/d/{_PAIRED_DOC_ID}/edit"


def _capture_thread():
    from gdoc.model import Thread

    return Thread(
        id="c1",
        content="ai! add a section on refunds",
        author_name="Nail Khusnullin",
        author_email=None,
        by_agent=False,
        quoted="policy text",
        resolved=False,
        replies=(),
    )


def _run_capture(tmp_path, extra=()):
    with patch("gdoc.cli.drive_service"), patch(
        "gdoc.cli.fetch_threads", return_value=(_capture_thread(),)
    ):
        return main(["capture", _PAIRED_URL, "c1", "--repo-root", str(tmp_path), *extra])


def test_capture_derives_the_slug_from_the_paired_source(capsys, tmp_path):
    """The document title changes every iteration. The source file does not."""
    source = tmp_path / "2026-08-13-topic.md"
    source.write_text(f"---\ngdoc: {_PAIRED_DOC_ID}\n---\n\nBody.\n")
    exit_code = _run_capture(tmp_path)
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["slug"] == "2026-08-13-topic"
    pending = tmp_path / ".gdoc" / "2026-08-13-topic" / "pending.md"
    assert "Source: 2026-08-13-topic.md" in pending.read_text()


def test_capture_finds_the_source_by_an_older_version_id(capsys, tmp_path):
    source = tmp_path / "2026-08-13-topic.md"
    source.write_text(
        "---\ngdoc: 1LaterVersion\n"
        f"gdoc_versions:\n  - id: {_PAIRED_DOC_ID}\n    created: 2026-08-13\n"
        "  - id: 1LaterVersion\n    created: 2026-08-14\n---\n\nBody.\n"
    )
    exit_code = _run_capture(tmp_path)
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["slug"] == "2026-08-13-topic"


def test_capture_fails_when_nothing_is_paired_and_no_slug_is_given(capsys, tmp_path):
    exit_code = _run_capture(tmp_path)
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "--slug" in payload["error"]


def test_capture_warns_when_the_queue_belongs_to_a_different_source(capsys, tmp_path):
    """Two source files in different folders can still share a stem."""
    other = tmp_path / "archive"
    other.mkdir()
    (other / "2026-08-13-topic.md").write_text("---\ngdoc: 1OtherDocument\n---\n\nBody.\n")
    source = tmp_path / "2026-08-13-topic.md"
    source.write_text(f"---\ngdoc: {_PAIRED_DOC_ID}\n---\n\nBody.\n")
    from dataclasses import replace

    from gdoc.pending import append_item

    append_item(
        tmp_path,
        "2026-08-13-topic",
        replace(_capture_thread(), id="c0"),
        doc_id="1OtherDocument",
        today="2026-08-13",
        source="archive/2026-08-13-topic.md",
    )
    exit_code = _run_capture(tmp_path, extra=("--slug", "2026-08-13-topic"))
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert "archive/2026-08-13-topic.md" in payload["source_collision_warning"]


# ---------------------------------------------------------------------------
# shared drive support
# ---------------------------------------------------------------------------


def _get_kwargs(drive):
    """The keyword arguments of the last files().get call."""
    return drive.files().get.call_args.kwargs


def test_read_asks_for_metadata_with_shared_drive_support(capsys):
    # Without supportsAllDrives, Drive answers 404 for a document on a Shared
    # Drive that the caller can genuinely read. The error would be a bare 404
    # rather than the clear message the design promises.
    drive = MagicMock()
    drive.comments().list.return_value.execute.side_effect = [{"comments": []}]
    drive.files().get.return_value.execute.return_value = {"name": "Test doc"}
    with patch("gdoc.cli.drive_service", return_value=drive):
        main(["read", "https://docs.google.com/document/d/1AbC/edit"])
    capsys.readouterr()
    assert _get_kwargs(drive)["supportsAllDrives"] is True
