import json
from unittest.mock import MagicMock, patch

import pytest
from googleapiclient.errors import HttpError

from gdoc.cli import main
from gdoc.config import Config
from gdoc.generate import Result as GenerateResult
from gdoc.pairing import Pairing, read_pairing, write_pairing


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
# generate subcommand, the template and the version name
# ---------------------------------------------------------------------------


_NOTE = "---\ntitle: Kickoff Notes\n---\n\n# Section one\n\nBody.\n"


def _run_generate_with(tmp_path, text, config, extra=()):
    """Run generate with both Drive and the generator stubbed. Returns the stub."""
    from gdoc.config import Config
    from gdoc.generate import Result

    md = tmp_path / "note.md"
    md.write_text(text)
    assert isinstance(config, Config)
    with patch("gdoc.cli.drive_service"), patch(
        "gdoc.cli.load_config", return_value=config
    ), patch("gdoc.cli.generate") as fake_generate:
        fake_generate.return_value = Result(docx_path=tmp_path / "v1.docx", doc_id="1New")
        exit_code = main(["generate", "--md", str(md), "--out", str(tmp_path / "v1.docx"), *extra])
    assert exit_code == 0
    return fake_generate


def test_generate_names_the_version_from_the_cover_title(capsys, tmp_path):
    """No --name: the cover title plus the next version number."""
    from gdoc.config import Config

    fake = _run_generate_with(tmp_path, _NOTE, Config(output_folder_id="0AF"))
    capsys.readouterr()
    assert fake.call_args.args[2] == "Kickoff Notes v1"


def test_generate_counts_the_recorded_versions(capsys, tmp_path):
    from gdoc.config import Config

    text = (
        "---\ntitle: Kickoff Notes\ngdoc: docABC\n"
        "gdoc_versions:\n  - id: docV1\n    created: '2026-08-01'\n"
        "  - id: docV2\n    created: '2026-08-10'\n---\n\n# Section one\n\nBody.\n"
    )
    fake = _run_generate_with(tmp_path, text, Config(output_folder_id="0AF"))
    capsys.readouterr()
    assert fake.call_args.args[2] == "Kickoff Notes v3"


def test_generate_names_the_version_from_the_file_stem_without_a_template(capsys, tmp_path):
    """With no template there may be no front matter, so the file stem stands in."""
    from gdoc.config import Config

    fake = _run_generate_with(
        tmp_path, "# Plain\n\nBody.\n", Config(output_folder_id="0AF", template="none")
    )
    capsys.readouterr()
    assert fake.call_args.args[2] == "note v1"
    assert fake.call_args.kwargs["template"] is None


def test_generate_passes_the_configured_template_through(capsys, tmp_path):
    from gdoc.config import Config

    config = Config(output_folder_id="0AF", template="altery-group-policy-v1.0")
    fake = _run_generate_with(tmp_path, _NOTE, config)
    capsys.readouterr()
    assert fake.call_args.kwargs["template"] == "altery-group-policy-v1.0"


def test_the_template_argument_beats_the_config(capsys, tmp_path):
    from gdoc.config import Config

    fake = _run_generate_with(
        tmp_path,
        _NOTE,
        Config(output_folder_id="0AF", template="none"),
        extra=("--template", "altery-group-policy-v1.0"),
    )
    capsys.readouterr()
    assert fake.call_args.kwargs["template"] == "altery-group-policy-v1.0"
    assert fake.call_args.args[2] == "Kickoff Notes v1"


def _run_generate_on(tmp_path, name, text, extra=()):
    """Run generate on a named file, with Drive and the generator stubbed."""
    from gdoc.config import Config
    from gdoc.generate import Result

    md = tmp_path / name
    md.write_text(text)
    with patch("gdoc.cli.drive_service"), patch(
        "gdoc.cli.load_config", return_value=Config(output_folder_id="0AF")
    ), patch("gdoc.cli.generate") as fake_generate:
        fake_generate.return_value = Result(docx_path=tmp_path / "v1.docx", doc_id="1New")
        exit_code = main(
            ["generate", "--md", str(md), "--out", str(tmp_path / "v1.docx"), *extra]
        )
    return exit_code, fake_generate, md


_UNTITLED = "---\ngdoc: 1abc\n---\n\n# Miguel kickoff call\n\nBody.\n"


def test_generate_refuses_a_note_with_no_title_and_suggests_one(capsys, tmp_path):
    """Half the live notes have gdoc: and no title:. The refusal has to be usable."""
    exit_code, fake_generate, md = _run_generate_on(
        tmp_path, "2026-08-12-miguel-kickoff-call.md", _UNTITLED
    )
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert payload["missing"] == "title"
    assert payload["suggested_title"] == "Miguel kickoff call"
    assert payload["suggested_from"] == "h1"
    assert payload["md"] == str(md)
    assert "--title" in payload["hint"]
    # The suggestion is Nail's to approve. Without this the hint reads as
    # permission to use the candidate, which is what the refusal exists to stop.
    assert "approve" in payload["hint"]
    assert "Nail" in payload["hint"]
    fake_generate.assert_not_called()


def test_the_suggestion_falls_back_to_the_file_name(capsys, tmp_path):
    exit_code, _fake, _md = _run_generate_on(
        tmp_path, "2026-08-12-miguel-kickoff-call.md", "---\ngdoc: 1abc\n---\n\nProse only.\n"
    )
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert payload["suggested_title"] == "Miguel kickoff call"
    assert payload["suggested_from"] == "filename"


def test_the_title_flag_publishes_without_touching_the_note(capsys, tmp_path):
    """The approved title can be used once before anyone edits the file."""
    exit_code, fake_generate, md = _run_generate_on(
        tmp_path,
        "2026-08-12-miguel-kickoff-call.md",
        _UNTITLED,
        extra=("--title", "Kickoff Notes"),
    )
    capsys.readouterr()
    assert exit_code == 0
    assert fake_generate.call_args.args[2] == "Kickoff Notes v1"
    assert fake_generate.call_args.kwargs["title"] == "Kickoff Notes"
    assert md.read_text() == _UNTITLED


def test_a_missing_title_found_during_the_build_is_reported_the_same_way(capsys, tmp_path):
    """With --name given, nothing parses the front matter until the build does."""
    from gdoc.config import Config
    from gdoc.render.frontmatter import MissingTitle

    md = tmp_path / "2026-08-12-miguel-kickoff-call.md"
    md.write_text(_UNTITLED)
    with patch("gdoc.cli.drive_service"), patch(
        "gdoc.cli.load_config", return_value=Config(output_folder_id="0AF")
    ), patch("gdoc.cli.generate", side_effect=MissingTitle("Miguel kickoff call", "h1")):
        exit_code = main(
            ["generate", "--md", str(md), "--name", "X v1", "--out", str(tmp_path / "v1.docx")]
        )
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert payload["missing"] == "title"
    assert payload["suggested_title"] == "Miguel kickoff call"


# ---------------------------------------------------------------------------
# generate: version bookkeeping
# ---------------------------------------------------------------------------


def _generate_ok(tmp_path, md, doc_id="1New", extra=()):
    """Run generate with a stubbed Drive and a stubbed export."""
    drive = MagicMock()
    with patch("gdoc.cli.drive_service", return_value=drive), \
         patch("gdoc.cli.load_config", return_value=Config(output_folder_id="0AF", template="none")), \
         patch("gdoc.cli.export_markdown", return_value="# exported\n"), \
         patch("gdoc.cli.generate") as fake_generate:
        fake_generate.return_value = GenerateResult(
            docx_path=tmp_path / "v.docx", doc_id=doc_id, link="https://x/edit"
        )
        argv = ["generate", "--md", str(md), "--out", str(tmp_path / "v.docx"),
                "--name", "X v1", "--baseline-root", str(tmp_path), *extra]
        return main(argv)


def test_generate_pairs_a_file_that_has_never_been_generated(capsys, tmp_path):
    md = tmp_path / "note.md"
    md.write_text("---\ntitle: Kickoff\n---\n\nBody.\n")
    assert _generate_ok(tmp_path, md) == 0
    payload = json.loads(capsys.readouterr().out)
    pairing = read_pairing(md)
    assert pairing.doc_id == "1New"
    assert [v["id"] for v in pairing.versions] == ["1New"]
    assert payload["version"] == 1
    assert payload["slug"] == "note"


def test_generate_appends_the_next_version_and_moves_the_pointer(tmp_path):
    md = tmp_path / "note.md"
    md.write_text("---\ntitle: Kickoff\n---\n\nBody.\n")
    write_pairing(md, Pairing(doc_id="1Old", versions=({"id": "1Old", "created": "2026-08-01"},)))
    assert _generate_ok(tmp_path, md, doc_id="1Next") == 0
    pairing = read_pairing(md)
    assert pairing.doc_id == "1Next"
    assert [v["id"] for v in pairing.versions] == ["1Old", "1Next"]


def test_generate_records_nothing_when_the_upload_failed(tmp_path):
    md = tmp_path / "note.md"
    original = "---\ntitle: Kickoff\n---\n\nBody.\n"
    md.write_text(original)
    drive = MagicMock()
    with patch("gdoc.cli.drive_service", return_value=drive), \
         patch("gdoc.cli.load_config", return_value=Config(output_folder_id="0AF", template="none")), \
         patch("gdoc.cli.generate") as fake_generate:
        fake_generate.return_value = GenerateResult(
            docx_path=tmp_path / "v.docx", doc_id=None, reason="upload refused with 403"
        )
        exit_code = main(["generate", "--md", str(md), "--out", str(tmp_path / "v.docx"),
                          "--name", "X v1", "--baseline-root", str(tmp_path)])
    assert exit_code == 0
    assert md.read_text() == original


def test_generate_reports_a_pairing_failure_without_hiding_the_document(capsys, tmp_path):
    """The document was really uploaded. A failed pairing write must not hide that."""
    md = tmp_path / "note.md"
    md.write_text("---\ntitle: Kickoff\n---\n\nBody.\n")
    with patch("gdoc.cli.write_pairing", side_effect=OSError("disk full")):
        assert _generate_ok(tmp_path, md) == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["doc_id"] == "1New"
    assert payload["link"] == "https://x/edit"
    assert "pairing_error" in payload
    assert "version" not in payload


def test_generate_survives_a_non_ioerror_pairing_failure_too(capsys, tmp_path):
    """OSError is not the only way reading or writing the pairing can fail.

    A malformed front matter (bad merge, hand edit) makes read_pairing raise
    something that is not an OSError, such as AttributeError. The document was
    still published, so doc_id and link must still reach the payload.
    """
    md = tmp_path / "note.md"
    md.write_text("---\ntitle: Kickoff\n---\n\nBody.\n")
    with patch("gdoc.cli.read_pairing", side_effect=AttributeError("'list' object has no attribute 'get'")):
        assert _generate_ok(tmp_path, md) == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["doc_id"] == "1New"
    assert payload["link"] == "https://x/edit"
    assert "pairing_error" in payload
    assert "version" not in payload


def test_generate_survives_a_baseline_failure_too(capsys, tmp_path):
    """The baseline write can fail on its own, independent of the pairing write.

    export_markdown is a Drive network call. A dropped connection or a refused
    token surfaces as an HttpError, not an OSError. The document was still
    published, so doc_id and link must still reach the payload, and the pairing
    write is an independent fact that must still happen.
    """
    fake_resp = MagicMock()
    fake_resp.status = 500
    md = tmp_path / "note.md"
    md.write_text("---\ntitle: Kickoff\n---\n\nBody.\n")
    drive = MagicMock()
    with patch("gdoc.cli.drive_service", return_value=drive), \
         patch("gdoc.cli.load_config", return_value=Config(output_folder_id="0AF", template="none")), \
         patch("gdoc.cli.export_markdown", side_effect=HttpError(fake_resp, b"boom")), \
         patch("gdoc.cli.generate") as fake_generate:
        fake_generate.return_value = GenerateResult(
            docx_path=tmp_path / "v.docx", doc_id="1New", link="https://x/edit"
        )
        argv = ["generate", "--md", str(md), "--out", str(tmp_path / "v.docx"),
                "--name", "X v1", "--baseline-root", str(tmp_path)]
        exit_code = main(argv)
    assert exit_code == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["doc_id"] == "1New"
    assert payload["link"] == "https://x/edit"
    assert "baseline_error" in payload
    # independent of the baseline: the version is still recorded
    pairing = read_pairing(md)
    assert pairing.doc_id == "1New"
    assert payload["version"] == 1


def test_generate_records_nothing_without_baseline_root(tmp_path):
    md = tmp_path / "note.md"
    original = "---\ntitle: Kickoff\n---\n\nBody.\n"
    md.write_text(original)
    drive = MagicMock()
    with patch("gdoc.cli.drive_service", return_value=drive), \
         patch("gdoc.cli.load_config", return_value=Config(output_folder_id="0AF", template="none")), \
         patch("gdoc.cli.generate") as fake_generate:
        fake_generate.return_value = GenerateResult(
            docx_path=tmp_path / "v.docx", doc_id="1New", link="https://x/edit"
        )
        main(["generate", "--md", str(md), "--out", str(tmp_path / "v.docx"), "--name", "X v1"])
    assert md.read_text() == original


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


def test_pair_add_version_moves_the_current_pointer(tmp_path):
    """Nothing may leave gdoc: naming a version that is no longer current."""
    md = tmp_path / "note.md"
    md.write_text("---\ntitle: Kickoff\n---\n\nBody.\n")
    write_pairing(md, Pairing(doc_id="1Old", versions=({"id": "1Old", "created": "2026-08-01"},)))
    exit_code = main(["pair", "add-version", "--md", str(md),
                      "--version-id", "1Next", "--created", "2026-08-14"])
    pairing = read_pairing(md)
    assert exit_code == 0
    assert pairing.doc_id == "1Next"
    assert [v["id"] for v in pairing.versions] == ["1Old", "1Next"]


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


# ---------------------------------------------------------------------------
# auth subcommand
# ---------------------------------------------------------------------------


def test_auth_login_reports_the_account(capsys, tmp_path):
    drive = MagicMock()
    with patch("gdoc.cli.oauth.login", return_value="creds") as login, patch(
        "gdoc.cli.drive_service", return_value=drive
    ), patch(
        "gdoc.cli.oauth.account",
        return_value={"displayName": "Nail Khusnullin", "emailAddress": "nail@altery.com"},
    ):
        exit_code = main(["auth", "login"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["logged_in"] is True
    assert payload["account"] == "nail@altery.com"
    assert login.call_args.args[0] == ["https://www.googleapis.com/auth/drive"]


def test_auth_login_passes_explicit_paths_through(capsys, tmp_path):
    client = tmp_path / "client.json"
    token = tmp_path / "token.json"
    with patch("gdoc.cli.oauth.login", return_value="creds") as login, patch(
        "gdoc.cli.drive_service"
    ), patch("gdoc.cli.oauth.account", return_value={}):
        main(["auth", "login", "--client", str(client), "--token", str(token)])
    assert login.call_args.kwargs["client_path"] == str(client)
    assert login.call_args.kwargs["token_path"] == str(token)


def test_auth_login_reports_a_missing_client_file_as_an_error(capsys, tmp_path):
    client = tmp_path / "client.json"
    with patch("gdoc.cli.oauth.login", side_effect=FileNotFoundError("no client")):
        exit_code = main(["auth", "login", "--client", str(client)])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "no client" in payload["error"]


def test_auth_status_says_it_is_ready(capsys):
    with patch("gdoc.cli.load_config", return_value=__import__("gdoc.config", fromlist=["Config"]).Config()), patch(
        "gdoc.cli.drive_service"
    ), patch("gdoc.cli.oauth.account", return_value={"emailAddress": "nail@altery.com"}):
        exit_code = main(["auth", "status"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["ready"] is True
    assert payload["auth_mode"] == "oauth"
    assert payload["account"] == "nail@altery.com"


def test_auth_status_never_fails_and_says_what_is_wrong(capsys):
    """Its whole job is to report a broken credential, so it must not raise."""
    with patch("gdoc.cli.load_config", side_effect=FileNotFoundError("no config")), patch(
        "gdoc.cli.drive_service", side_effect=FileNotFoundError("no OAuth token at /x")
    ):
        exit_code = main(["auth", "status"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["ready"] is False
    assert "no OAuth token" in payload["problem"]
    assert payload["auth_mode"] == "oauth"


def test_auth_status_prints_where_to_revoke(capsys):
    with patch("gdoc.cli.load_config", side_effect=FileNotFoundError), patch(
        "gdoc.cli.drive_service", side_effect=RuntimeError("nope")
    ):
        main(["auth", "status"])
    payload = json.loads(capsys.readouterr().out)
    assert "myaccount.google.com" in payload["revoke_url"]


# ---------------------------------------------------------------------------
# auth login switches the credential over, rather than asking for a hand edit
# ---------------------------------------------------------------------------


@pytest.fixture
def config_at(gdoc_agent_dir):
    """The config file inside this test's own gdoc-agent directory.

    The redirection itself is in conftest and applies to every test, because a
    login writes this file and no test may write the real one.
    """
    return gdoc_agent_dir / "config.json"


def _login(extra=()):
    """A successful browser flow. Returns nothing; the caller reads the config."""
    return patch("gdoc.cli.oauth.login", return_value="creds"), patch(
        "gdoc.cli.drive_service"
    ), patch("gdoc.cli.oauth.account", return_value={"emailAddress": "nail@altery.com"})


def test_auth_login_switches_a_service_account_config_over(capsys, config_at):
    """Signing in and then still using the old credential is the worst outcome.

    Every other command reads auth_mode, so a login that left it alone would
    report success and change nothing.
    """
    config_at.write_text(json.dumps({"auth_mode": "service_account"}))
    a, b, c = _login()
    with a, b, c:
        exit_code = main(["auth", "login"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert json.loads(config_at.read_text())["auth_mode"] == "oauth"
    assert payload["auth_mode"] == "oauth"
    assert payload["auth_mode_was"] == "service_account"


def test_auth_login_keeps_the_rest_of_the_config(capsys, config_at):
    config_at.write_text(
        json.dumps({"auth_mode": "service_account", "output_folder_id": "0AFolderId"})
    )
    a, b, c = _login()
    with a, b, c:
        main(["auth", "login"])
    capsys.readouterr()
    assert json.loads(config_at.read_text())["output_folder_id"] == "0AFolderId"


def test_auth_login_states_the_mode_a_config_left_unstated(capsys, config_at):
    """Writing it down is the point. An inferred mode changes as files appear."""
    config_at.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    a, b, c = _login()
    with a, b, c:
        main(["auth", "login"])
    payload = json.loads(capsys.readouterr().out)
    assert json.loads(config_at.read_text())["auth_mode"] == "oauth"
    assert payload["auth_mode"] == "oauth"
    assert payload["auth_mode_was"] is None


def test_auth_login_creates_a_config_when_there_is_none(capsys, config_at):
    a, b, c = _login()
    with a, b, c:
        main(["auth", "login"])
    capsys.readouterr()
    assert json.loads(config_at.read_text()) == {"auth_mode": "oauth"}


def test_auth_login_on_a_config_already_oauth_reports_no_change(capsys, config_at):
    config_at.write_text(json.dumps({"auth_mode": "oauth"}))
    a, b, c = _login()
    with a, b, c:
        main(["auth", "login"])
    payload = json.loads(capsys.readouterr().out)
    assert payload["auth_mode"] == "oauth"
    assert payload["auth_mode_was"] == "oauth"


def test_a_failed_login_leaves_the_config_alone(capsys, config_at):
    """No token was written, so switching the mode would break a working setup."""
    config_at.write_text(json.dumps({"auth_mode": "service_account"}))
    with patch("gdoc.cli.oauth.login", side_effect=FileNotFoundError("no client")):
        exit_code = main(["auth", "login"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "no client" in payload["error"]
    assert json.loads(config_at.read_text())["auth_mode"] == "service_account"


def test_a_config_that_cannot_be_written_does_not_lose_the_login(capsys, config_at):
    """The token is already on disk, so this is a warning, not a failure."""
    a, b, c = _login()
    with a, b, c, patch(
        "gdoc.cli.write_auth_mode", side_effect=OSError("read-only file system")
    ):
        exit_code = main(["auth", "login"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["logged_in"] is True
    assert "read-only file system" in payload["config_error"]


# ---------------------------------------------------------------------------
# auth use: the supported way to switch, in both directions
# ---------------------------------------------------------------------------


def test_auth_use_switches_to_the_service_account(capsys, config_at):
    """Hand editing JSON is not a setup step. This is the other direction."""
    config_at.write_text(json.dumps({"auth_mode": "oauth"}))
    exit_code = main(["auth", "use", "service_account"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert json.loads(config_at.read_text())["auth_mode"] == "service_account"
    assert payload["auth_mode"] == "service_account"
    assert payload["auth_mode_was"] == "oauth"


def test_auth_use_switches_to_oauth(capsys, config_at):
    config_at.write_text(json.dumps({"auth_mode": "service_account"}))
    exit_code = main(["auth", "use", "oauth"])
    capsys.readouterr()
    assert exit_code == 0
    assert json.loads(config_at.read_text())["auth_mode"] == "oauth"


def test_auth_use_keeps_the_rest_of_the_config(capsys, config_at):
    config_at.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    main(["auth", "use", "service_account"])
    capsys.readouterr()
    assert json.loads(config_at.read_text())["output_folder_id"] == "0AFolderId"


def test_auth_use_refuses_a_mode_that_does_not_exist(capsys, config_at):
    """argparse rejects it before any handler runs, so the config is untouched."""
    config_at.write_text(json.dumps({"auth_mode": "oauth"}))
    with pytest.raises(SystemExit) as excinfo:
        main(["auth", "use", "magic"])
    assert excinfo.value.code == 2
    assert json.loads(config_at.read_text())["auth_mode"] == "oauth"


def test_auth_use_warns_when_the_credential_it_switched_to_is_missing(capsys, config_at):
    """Switching to a credential that is not installed yet must not look fine."""
    exit_code = main(["auth", "use", "service_account"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert "sa-key.json" in payload["warning"]


def test_auth_use_says_nothing_extra_when_the_credential_is_there(
    capsys, config_at, gdoc_agent_dir
):
    (gdoc_agent_dir / "sa-key.json").write_text("{}")
    main(["auth", "use", "service_account"])
    payload = json.loads(capsys.readouterr().out)
    assert "warning" not in payload


def test_auth_logout_says_which_credential_runs_next(capsys, config_at, gdoc_agent_dir):
    """Deleting the token leaves every command broken, so say so and say the fix."""
    (gdoc_agent_dir / "oauth-token.json").write_text("{}")
    config_at.write_text(json.dumps({"auth_mode": "oauth"}))
    main(["auth", "logout"])
    payload = json.loads(capsys.readouterr().out)
    assert payload["auth_mode"] == "oauth"
    assert "gdoc auth login" in payload["next"]


def test_auth_logout_names_the_service_account_when_its_key_is_there(
    capsys, config_at, gdoc_agent_dir
):
    (gdoc_agent_dir / "oauth-token.json").write_text("{}")
    (gdoc_agent_dir / "sa-key.json").write_text("{}")
    config_at.write_text(json.dumps({"auth_mode": "oauth"}))
    main(["auth", "logout"])
    payload = json.loads(capsys.readouterr().out)
    assert "gdoc auth use service_account" in payload["next"]


def test_auth_logout_does_not_switch_the_mode_by_itself(capsys, config_at, gdoc_agent_dir):
    """Logging out to sign in as somebody else is the common case."""
    (gdoc_agent_dir / "oauth-token.json").write_text("{}")
    (gdoc_agent_dir / "sa-key.json").write_text("{}")
    config_at.write_text(json.dumps({"auth_mode": "oauth"}))
    main(["auth", "logout"])
    capsys.readouterr()
    assert json.loads(config_at.read_text())["auth_mode"] == "oauth"


def test_a_login_into_another_token_path_does_not_claim_the_mode(capsys, config_at, tmp_path):
    """Nothing else reads a custom token path, so oauth would find no token.

    Claiming the mode here left a working service_account install with a config
    saying oauth and no token where every command looks for one.
    """
    config_at.write_text(json.dumps({"auth_mode": "service_account"}))
    a, b, c = _login()
    with a, b, c:
        exit_code = main(["auth", "login", "--token", str(tmp_path / "elsewhere.json")])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert json.loads(config_at.read_text())["auth_mode"] == "service_account"
    assert "auth_mode" not in payload
    assert "elsewhere.json" in payload["config_note"]


def test_the_mode_is_written_before_the_account_is_looked_up(capsys, config_at):
    """The account lookup is a network call, and the token is already on disk.

    Writing the mode after it meant a lookup failure left the person signed in
    with the old credential still configured, and retrying changed nothing.
    """
    config_at.write_text(json.dumps({"auth_mode": "service_account"}))
    with patch("gdoc.cli.oauth.login", return_value="creds"), patch(
        "gdoc.cli.drive_service"
    ), patch("gdoc.cli.oauth.account", side_effect=OSError("network unreachable")):
        exit_code = main(["auth", "login"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "network unreachable" in payload["error"]
    assert json.loads(config_at.read_text())["auth_mode"] == "oauth"


def test_auth_use_oauth_warns_when_there_is_no_token_yet(capsys, config_at):
    exit_code = main(["auth", "use", "oauth"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert "gdoc auth login" in payload["warning"]


def test_logout_says_when_deleting_the_token_changes_the_credential(
    capsys, config_at, gdoc_agent_dir
):
    """An unstated config resolves by files, so deleting the token repoints it.

    Anyone who signed in before login wrote the config is in this state. The
    account that posts replies changes, so it cannot be left unsaid.
    """
    config_at.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    (gdoc_agent_dir / "oauth-token.json").write_text("{}")
    (gdoc_agent_dir / "sa-key.json").write_text("{}")
    main(["auth", "logout"])
    payload = json.loads(capsys.readouterr().out)
    assert payload["auth_mode"] == "service_account"
    assert payload["auth_mode_was"] == "oauth"
    assert "service_account" in payload["next"]


def test_auth_status_survives_a_config_that_is_not_a_json_object(capsys, config_at):
    """A list parses fine and then has no keys. It must not be a traceback."""
    config_at.write_text("[1, 2]")
    with patch("gdoc.cli.drive_service", side_effect=RuntimeError("nope")):
        exit_code = main(["auth", "status"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert str(config_at) in payload["config_error"]


def test_read_refuses_a_config_it_cannot_understand(capsys, config_at):
    """Not knowing which credential was asked for must not pick the wider one."""
    config_at.write_text(json.dumps({"auth_mode": "service-account"}))
    exit_code = main(["read", "https://docs.google.com/document/d/1AbC/edit"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "service-account" in payload["error"]


def test_auth_status_says_the_mode_came_from_the_config(capsys, config_at):
    config_at.write_text(json.dumps({"auth_mode": "service_account"}))
    with patch("gdoc.cli.drive_service", side_effect=RuntimeError("nope")):
        main(["auth", "status"])
    payload = json.loads(capsys.readouterr().out)
    assert payload["auth_mode"] == "service_account"
    assert payload["auth_mode_source"] == "config"


def test_auth_status_says_when_the_mode_was_only_inferred(capsys, config_at, gdoc_agent_dir):
    """An upgraded install has no auth_mode, so say why it is on this one."""
    config_at.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    (gdoc_agent_dir / "sa-key.json").write_text("{}")
    with patch("gdoc.cli.drive_service", side_effect=RuntimeError("nope")):
        main(["auth", "status"])
    payload = json.loads(capsys.readouterr().out)
    assert payload["auth_mode"] == "service_account"
    assert payload["auth_mode_source"] == "inferred"


def test_auth_status_does_not_guess_a_mode_for_a_broken_config(capsys, config_at):
    """Reporting a mode here would be a guess, and the wider one at that."""
    config_at.write_text(json.dumps({"auth_mode": "magic"}))
    with patch("gdoc.cli.drive_service", side_effect=RuntimeError("nope")):
        exit_code = main(["auth", "status"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["auth_mode"] is None
    assert payload["auth_mode_source"] == "unknown"
    assert "magic" in payload["config_error"]


def test_auth_status_names_a_config_it_could_not_read(capsys, config_at):
    """Falling back quietly would hide the typo that caused the fallback."""
    config_at.write_text(json.dumps({"auth_mode": "magic"}))
    with patch("gdoc.cli.drive_service", side_effect=RuntimeError("nope")):
        main(["auth", "status"])
    payload = json.loads(capsys.readouterr().out)
    assert "magic" in payload["config_error"]


def test_auth_status_says_nothing_about_a_config_that_is_merely_absent(capsys, config_at):
    """No config is the normal state of a new install, not a problem."""
    with patch("gdoc.cli.drive_service", side_effect=RuntimeError("nope")):
        main(["auth", "status"])
    payload = json.loads(capsys.readouterr().out)
    assert "config_error" not in payload


def test_auth_status_on_a_bare_machine_is_inferred_oauth(capsys, config_at):
    with patch("gdoc.cli.drive_service", side_effect=RuntimeError("nope")):
        main(["auth", "status"])
    payload = json.loads(capsys.readouterr().out)
    assert payload["auth_mode"] == "oauth"
    assert payload["auth_mode_source"] == "inferred"


def test_auth_logout_reports_that_it_removed_a_token(capsys, tmp_path):
    token = tmp_path / "token.json"
    token.write_text("{}")
    exit_code = main(["auth", "logout", "--token", str(token)])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["logged_out"] is True
    assert not token.exists()


def test_auth_logout_on_a_machine_with_no_token_says_so(capsys, tmp_path):
    exit_code = main(["auth", "logout", "--token", str(tmp_path / "token.json")])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["logged_out"] is False


def _read_payload(capsys, argv, threads):
    drive = MagicMock()
    drive.comments().list.return_value.execute.side_effect = [{"comments": threads}]
    drive.files().get.return_value.execute.return_value = {"name": "Test doc"}
    with patch("gdoc.cli.drive_service", return_value=drive):
        exit_code = main(argv)
    return exit_code, json.loads(capsys.readouterr().out)


_MARKED = {
    "id": "t1",
    "content": "ai: rephrase",
    "author": {"displayName": "Nail Khusnullin", "me": False},
}
_UNMARKED = {
    "id": "t2",
    "content": "This annex reads oddly",
    "author": {"displayName": "William Mejia", "me": False},
}
_ANSWERED = {
    "id": "t3",
    "content": "ai? who owns this",
    "author": {"displayName": "Nail Khusnullin", "me": False},
    "replies": [
        {
            "id": "r1",
            "content": "Compliance owns it.\n\n[gdoc]",
            "author": {"displayName": "Nail Khusnullin", "me": True},
        }
    ],
}


def test_read_reports_the_default_mode(capsys):
    _, payload = _read_payload(capsys, ["read", "https://docs.google.com/document/d/1AbC/edit"], [_MARKED])
    assert payload["mode"] == "marked"


def test_read_all_reports_all_mode(capsys):
    _, payload = _read_payload(
        capsys, ["read", "https://docs.google.com/document/d/1AbC/edit", "--all"], [_MARKED]
    )
    assert payload["mode"] == "all"


def test_read_skips_unmarked_comments_by_default(capsys):
    _, payload = _read_payload(
        capsys, ["read", "https://docs.google.com/document/d/1AbC/edit"], [_MARKED, _UNMARKED]
    )
    assert [t["id"] for t in payload["addressed"]] == ["t1"]
    assert [t["id"] for t in payload["skipped"]] == ["t2"]


def test_read_all_offers_unmarked_comments(capsys):
    _, payload = _read_payload(
        capsys,
        ["read", "https://docs.google.com/document/d/1AbC/edit", "--all"],
        [_MARKED, _UNMARKED],
    )
    assert [t["id"] for t in payload["addressed"]] == ["t1", "t2"]


def test_each_thread_says_whether_it_was_marked(capsys):
    _, payload = _read_payload(
        capsys,
        ["read", "https://docs.google.com/document/d/1AbC/edit", "--all"],
        [_MARKED, _UNMARKED],
    )
    marked = {t["id"]: t["marked"] for t in payload["addressed"]}
    assert marked == {"t1": True, "t2": False}


def test_each_thread_says_whether_it_was_answered(capsys):
    _, payload = _read_payload(
        capsys,
        ["read", "https://docs.google.com/document/d/1AbC/edit", "--all"],
        [_MARKED, _ANSWERED],
    )
    answered = {t["id"]: t["answered"] for t in payload["addressed"]}
    assert answered == {"t1": False, "t3": True}


def test_the_payload_carries_the_existing_replies(capsys):
    _, payload = _read_payload(
        capsys, ["read", "https://docs.google.com/document/d/1AbC/edit", "--all"], [_ANSWERED]
    )
    reply = payload["addressed"][0]["replies"][0]
    assert reply["author"] == "Nail Khusnullin"
    assert reply["by_gdoc"] is True
    assert "Compliance owns it." in reply["content"]


def test_a_thread_with_no_replies_carries_an_empty_list(capsys):
    _, payload = _read_payload(
        capsys, ["read", "https://docs.google.com/document/d/1AbC/edit"], [_MARKED]
    )
    assert payload["addressed"][0]["replies"] == []


# ---------------------------------------------------------------------------
# Every command tells the guard which file it may touch
#
# The guard is only as good as its seed. These assert the wire between the CLI
# and drive_service, which is the one place the id is actually chosen. Without
# them, dropping `doc_ids=` from a command is a silent, live-only regression.
# ---------------------------------------------------------------------------

DOC_URL = "https://docs.google.com/document/d/1AbCdEf/edit"


def test_read_builds_a_client_scoped_to_the_document():
    with patch("gdoc.cli.drive_service") as drive_service, patch(
        "gdoc.cli.fetch_threads", return_value=()
    ), patch("gdoc.cli._file_meta", return_value={"name": "doc"}):
        main(["read", DOC_URL])
    assert drive_service.call_args.kwargs["doc_ids"] == "1AbCdEf"


def test_reply_builds_a_client_scoped_to_the_document(tmp_path):
    body = tmp_path / "body.txt"
    body.write_text("Plain answer.")
    with patch("gdoc.cli.drive_service") as drive_service, patch(
        "gdoc.cli.post_reply", return_value="r1"
    ):
        main(["reply", DOC_URL, "c1", "--body-file", str(body)])
    assert drive_service.call_args.kwargs["doc_ids"] == "1AbCdEf"


def test_export_builds_a_client_scoped_to_the_document():
    with patch("gdoc.cli.drive_service") as drive_service, patch(
        "gdoc.cli.export_markdown", return_value="# doc"
    ):
        main(["export", DOC_URL])
    assert drive_service.call_args.kwargs["doc_ids"] == "1AbCdEf"


def test_capture_builds_a_client_scoped_to_the_document(tmp_path):
    """capture extracted the id after building the client once. It must not."""
    with patch("gdoc.cli.drive_service") as drive_service, patch(
        "gdoc.cli.fetch_threads", return_value=()
    ):
        main(["capture", DOC_URL, "c1", "--repo-root", str(tmp_path)])
    assert drive_service.call_args.kwargs["doc_ids"] == "1AbCdEf"


def test_generate_scopes_its_client_to_the_output_folder(tmp_path):
    """generate has no input document. The folder is the file it was given."""
    from gdoc.config import Config
    from gdoc.generate import Result

    md = tmp_path / "note.md"
    md.write_text("---\ntitle: A note\n---\n\nBody.\n")
    with patch("gdoc.cli.load_config", return_value=Config(output_folder_id="0AFolder")), patch(
        "gdoc.cli.drive_service"
    ) as drive_service, patch(
        "gdoc.cli.generate", return_value=Result(docx_path=tmp_path / "o.docx")
    ):
        main(["generate", "--md", str(md), "--out", str(tmp_path / "o.docx")])
    assert drive_service.call_args.kwargs["doc_ids"] == ["0AFolder"]


# ---------------------------------------------------------------------------
# generate: naming the folder on the command line
# ---------------------------------------------------------------------------

FOLDER_URL = "https://drive.google.com/drive/folders/0AFolderIdFromUrl?usp=sharing"


def _generate_with_no_config(tmp_path, extra):
    """Run generate with no config file at all. Returns the generate stub."""
    from gdoc.generate import Result

    md = tmp_path / "note.md"
    md.write_text(_NOTE)
    with patch("gdoc.cli.drive_service"), patch(
        "gdoc.cli.load_config",
        side_effect=FileNotFoundError("config not found at ~/.config/gdoc-agent/config.json"),
    ), patch("gdoc.cli.generate") as fake_generate:
        fake_generate.return_value = Result(docx_path=tmp_path / "v1.docx", doc_id="1New")
        exit_code = main(["generate", "--md", str(md), "--out", str(tmp_path / "v1.docx"), *extra])
    return exit_code, fake_generate


def test_generate_takes_the_folder_id_out_of_a_pasted_url(capsys, tmp_path):
    from gdoc.config import Config

    fake = _run_generate_with(tmp_path, _NOTE, Config(), extra=["--folder-id", FOLDER_URL])
    capsys.readouterr()
    assert fake.call_args.kwargs["folder_id"] == "0AFolderIdFromUrl"


def test_generate_needs_no_config_file_when_the_folder_is_given(capsys, tmp_path):
    """The folder is the one thing generate cannot work out for itself."""
    from gdoc.render import profiles

    exit_code, fake = _generate_with_no_config(tmp_path, ["--folder-id", FOLDER_URL])
    capsys.readouterr()
    assert exit_code == 0
    assert fake.call_args.kwargs["folder_id"] == "0AFolderIdFromUrl"
    # No config file still means the house style, not a plain document.
    assert fake.call_args.kwargs["template"] == profiles.DEFAULT_TEMPLATE


def test_generate_without_a_folder_still_reports_the_missing_config(capsys, tmp_path):
    exit_code, fake = _generate_with_no_config(tmp_path, [])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "config not found" in payload["error"]
    fake.assert_not_called()


def test_generate_refuses_a_document_url_as_the_folder(capsys, tmp_path):
    """Pasting the document instead of the folder fails here, not on upload."""
    exit_code, fake = _generate_with_no_config(
        tmp_path, ["--folder-id", "https://docs.google.com/document/d/1AbCdEfGhIjKlMnOpQrStUv/edit"]
    )
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "a document, not a folder" in payload["error"]
    fake.assert_not_called()
