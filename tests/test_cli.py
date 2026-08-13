import json
from unittest.mock import MagicMock, patch

import pytest

from gdoc.cli import main


# ---------------------------------------------------------------------------
# read subcommand
# ---------------------------------------------------------------------------


def test_read_prints_partitioned_threads_as_json(capsys):
    threads = [
        {
            "id": "t1",
            "content": "ai: rephrase",
            "author": {"displayName": "Nail Khusnullin", "me": False},
            "quotedFileContent": {"value": "asdasd"},
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
    with patch("gdoc.cli.drive_service", return_value=drive), patch(
        "gdoc.cli.load_config"
    ) as config:
        config.return_value.display_name = "Nail Khusnullin"
        exit_code = main(["read", "https://docs.google.com/document/d/1AbC/edit"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert [t["id"] for t in payload["mine"]] == ["t1"]
    assert [t["id"] for t in payload["others"]] == ["t2"]
    assert payload["doc_id"] == "1AbC"
    assert payload["slug"] == "test-doc"


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


def test_export_writes_mirror_and_returns_metadata(capsys, tmp_path):
    drive = MagicMock()
    drive.files().get.return_value.execute.return_value = {"name": "My policy"}
    with patch("gdoc.cli.drive_service", return_value=drive), patch(
        "gdoc.cli.export_markdown", return_value="# Markdown content"
    ), patch("gdoc.cli.write_mirror", return_value=tmp_path / "mirror.md") as mock_write:
        exit_code = main(
            ["export", "https://docs.google.com/document/d/1AbC/edit", "--repo-root", str(tmp_path)]
        )
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["doc_id"] == "1AbC"
    assert payload["slug"] == "my-policy"
    mock_write.assert_called_once()


def test_export_reports_mirror_conflict_as_json_error(capsys, tmp_path):
    from gdoc.mirror import MirrorConflict

    drive = MagicMock()
    drive.files().get.return_value.execute.return_value = {"name": "My policy"}
    with patch("gdoc.cli.drive_service", return_value=drive), patch(
        "gdoc.cli.export_markdown", return_value="# Markdown"
    ), patch("gdoc.cli.write_mirror", side_effect=MirrorConflict("dirty file")):
        exit_code = main(
            ["export", "https://docs.google.com/document/d/1AbC/edit", "--repo-root", str(tmp_path)]
        )
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "error" in payload
    assert "dirty" in payload["error"]


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
# export, slug collision detection
# ---------------------------------------------------------------------------


def test_export_warns_when_slug_maps_to_a_different_document(capsys, tmp_path):
    # Set up an existing mirror file paired to a different doc
    from gdoc.mirror import MIRROR_DIR

    mirror_file = tmp_path / MIRROR_DIR / "my-policy" / "mirror.md"
    mirror_file.parent.mkdir(parents=True)
    mirror_file.write_text("---\ngdoc: otherDocId\n---\n\nExisting content.\n")

    drive = MagicMock()
    drive.files().get.return_value.execute.return_value = {"name": "My policy"}
    # write_mirror gets a real repo_root but we stub it so no file is overwritten
    with patch("gdoc.cli.drive_service", return_value=drive), patch(
        "gdoc.cli.export_markdown", return_value="# New content"
    ), patch("gdoc.cli.write_mirror", return_value=mirror_file):
        exit_code = main(
            [
                "export",
                "https://docs.google.com/document/d/newDocId12345678901234/edit",
                "--repo-root",
                str(tmp_path),
            ]
        )
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert "slug_collision_warning" in payload
    assert "otherDocId" in payload["slug_collision_warning"]


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
        "gdoc.cli.load_config"
    ) as config, patch("gdoc.cli.fetch_threads", return_value=()):
        config.return_value.display_name = "Nail Khusnullin"
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
    pending = tmp_path / "docs" / "gdoc" / "policy" / "pending.md"
    assert pending.exists()
    assert "c1" in pending.read_text()


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
    with patch("gdoc.cli.drive_service", return_value=drive), patch(
        "gdoc.cli.load_config"
    ) as config:
        config.return_value.display_name = "Nail Khusnullin"
        main(["read", "https://docs.google.com/document/d/1AbC/edit"])
    capsys.readouterr()
    assert _get_kwargs(drive)["supportsAllDrives"] is True


def test_export_asks_for_metadata_with_shared_drive_support(capsys, tmp_path):
    drive = MagicMock()
    drive.files().get.return_value.execute.return_value = {"name": "My policy"}
    with patch("gdoc.cli.drive_service", return_value=drive), patch(
        "gdoc.cli.export_markdown", return_value="# Markdown content"
    ), patch("gdoc.cli.write_mirror", return_value=tmp_path / "mirror.md"):
        main(
            ["export", "https://docs.google.com/document/d/1AbC/edit", "--repo-root", str(tmp_path)]
        )
    capsys.readouterr()
    assert _get_kwargs(drive)["supportsAllDrives"] is True
