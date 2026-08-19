"""Publish a house-styled copy of a document gdoc knows nothing about.

Nothing here is paired, queued or recorded. The pulled markdown is an
intermediate, and the one thing these tests keep checking is that it leaves no
trace under the root once the document exists.
"""

from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from gdoc.export import Export
from gdoc.generate import Result
from gdoc.model import Thread
from gdoc.restyle import front_matter, restyle

SOURCE = "1SourceDoc"
NEW = "1NewDoc"
LINK = f"https://docs.google.com/document/d/{NEW}/edit"


def _thread(id="c1", resolved=False):
    return Thread(
        id=id,
        content="this number is wrong",
        author_name="William Mejia",
        author_email=None,
        by_agent=False,
        quoted=None,
        resolved=resolved,
        replies=(),
    )


def _drive(name="MC incentive routes"):
    drive = MagicMock()
    drive.files().get.return_value.execute.return_value = {"name": name}
    drive.comments().create.return_value.execute.return_value = {"id": "n1"}
    return drive


def _published(**kwargs):
    return Result(docx_path=Path("out.docx"), doc_id=NEW, link=LINK, **kwargs)


def _run(drive, threads=(), export=None, result=None, **kwargs):
    """restyle with its two network neighbours stubbed.

    export_with_media and generate are exercised by their own suites. What is
    under test here is the order the three are called in and what survives.
    """
    export = export or Export(markdown="# Body\n\nText.\n", images=(), warnings=())
    result = _published() if result is None else result
    with patch("gdoc.restyle.fetch_threads", return_value=tuple(threads)), patch(
        "gdoc.restyle.export_with_media", return_value=export
    ) as pull, patch("gdoc.restyle.generate", return_value=result) as publish:
        out = restyle(drive, SOURCE, folder_id="0AFolder", **kwargs)
    return out, pull, publish


# ---------------------------------------------------------------- the pull --


def test_the_document_name_becomes_the_title():
    result, _, publish = _run(_drive("MC incentive routes"))
    assert result.source_name == "MC incentive routes"
    assert result.title == "MC incentive routes"
    assert publish.call_args.kwargs["title"] == "MC incentive routes"


def test_an_explicit_title_wins_over_the_document_name():
    result, _, _ = _run(_drive("2026-08-19 draft v3 FINAL"), title="Incentive routes")
    assert result.title == "Incentive routes"


def test_front_matter_is_prepended_because_a_note_without_it_is_refused():
    """frontmatter.split refuses a file with no block, even given --title.

    Read at publish time. A successful run deletes the file, which is the point
    of test_the_working_directory_is_gone_when_the_run_succeeded below.
    """
    seen = []

    def capture(drive, md_path, *args, **kwargs):
        seen.append(Path(md_path).read_text())
        return _published()

    with patch("gdoc.restyle.fetch_threads", return_value=()), patch(
        "gdoc.restyle.export_with_media",
        return_value=Export(markdown="# Body\n\nText.\n", images=(), warnings=()),
    ), patch("gdoc.restyle.generate", side_effect=capture):
        restyle(_drive(), SOURCE, folder_id="0AFolder")

    assert seen[0].startswith("---\ntitle: MC incentive routes\n---\n")
    assert "# Body" in seen[0]


def test_nothing_but_the_title_is_invented():
    """A guessed owner on a cover page is worse than a blank one."""
    import yaml

    block = front_matter("Anything")
    assert yaml.safe_load(block.strip().strip("-")) == {"title": "Anything"}


@pytest.mark.parametrize(
    "name",
    [
        "Q3: Roadmap",
        "Fees, FX & margin",
        '"Final" draft',
        "#1 priority",
        "- leading dash",
        "yes",
        "2026-08-19",
    ],
)
def test_a_document_name_that_is_not_plain_yaml_still_publishes(name):
    """The point of restyle is documents whose owner never wrote any YAML.

    A colon in the name is the common one, and hand-built front matter breaks on
    it: 'title: Q3: Roadmap' is not a mapping and the run dies before it
    publishes.
    """
    from gdoc.render import frontmatter

    meta, _ = frontmatter.parse(front_matter(name) + "# Body\n")
    assert meta["title"] == name


def test_a_picture_that_could_not_be_carried_is_reported():
    export = Export(markdown="x", images=(), warnings=("logo.png could not be carried",))
    result, _, _ = _run(_drive(), export=export)
    assert result.image_warnings == ("logo.png could not be carried",)


# ------------------------------------------------------------- the publish --


def test_the_new_document_is_reported():
    result, _, _ = _run(_drive())
    assert result.doc_id == NEW
    assert result.link == LINK


def test_the_source_document_is_never_written_to():
    drive = _drive()
    _run(drive, threads=[_thread()])
    for call in drive.comments().create.call_args_list:
        assert call.kwargs["fileId"] != SOURCE
    drive.files().update.assert_not_called()


def test_no_folder_anywhere_is_refused_before_anything_is_pulled():
    drive = _drive()
    with patch("gdoc.restyle.export_with_media") as pull:
        with pytest.raises(ValueError) as error:
            restyle(drive, SOURCE, folder_id=None)
    assert "--folder-id" in str(error.value)
    assert "output_folder_id" in str(error.value)
    pull.assert_not_called()


# ------------------------------------------------------------ the comments --


def test_open_threads_are_copied_onto_the_new_document():
    drive = _drive()
    result, _, _ = _run(drive, threads=[_thread("a"), _thread("b", resolved=True)])
    assert result.comments_copied == 1
    assert result.comments_skipped_resolved == 1
    assert drive.comments().create.call_args.kwargs["fileId"] == NEW


def test_no_comments_reports_null_rather_than_zero():
    """'You asked me not to' and 'there were none' are different answers."""
    drive = _drive()
    result, _, _ = _run(drive, threads=[_thread()], copy_comments=False)
    assert result.comments_copied is None
    drive.comments().create.assert_not_called()


def test_nothing_is_copied_when_no_document_was_created():
    drive = _drive()
    failed = Result(docx_path=Path("out.docx"), doc_id=None, reason="403")
    result, _, _ = _run(drive, threads=[_thread()], result=failed)
    assert result.comments_copied is None
    drive.comments().create.assert_not_called()


# --------------------------------------------------------------- the mess --


def test_the_working_directory_is_gone_when_the_run_succeeded():
    result, _, publish = _run(_drive())
    assert result.workdir is None
    assert not publish.call_args.args[1].exists()


def test_a_failed_publish_keeps_the_pulled_markdown_and_says_where():
    """It is the only thing the run produced."""
    failed = Result(docx_path=Path("out.docx"), doc_id=None, reason="no folder")
    result, _, publish = _run(_drive(), result=failed)
    assert result.workdir is not None
    assert Path(result.workdir).is_dir()
    assert publish.call_args.args[1].exists()
    assert result.reason == "no folder"


def test_the_working_directory_is_not_under_the_root(tmp_path, monkeypatch):
    monkeypatch.chdir(tmp_path)
    _run(_drive())
    assert list(tmp_path.iterdir()) == []


def test_a_failed_pull_leaves_no_temp_directory_behind():
    """Nothing was created and nothing worth keeping was pulled, so clean up."""
    seen = []

    def explode(drive, doc_id, media_dir, out_path):
        seen.append(Path(out_path).parent)
        raise RuntimeError("pandoc is not installed")

    with patch("gdoc.restyle.fetch_threads", return_value=()), patch(
        "gdoc.restyle.export_with_media", side_effect=explode
    ):
        with pytest.raises(RuntimeError, match="pandoc"):
            restyle(_drive(), SOURCE, folder_id="0AFolder")

    assert not seen[0].exists()


def test_a_failed_render_keeps_the_pull_and_names_it_in_the_error():
    """A foreign document is exactly where the template build can choke, and
    then the pulled markdown is the only thing worth looking at."""
    seen = []

    def explode(drive, md_path, *args, **kwargs):
        seen.append(Path(md_path))
        raise RuntimeError("the template could not place a table")

    with patch("gdoc.restyle.fetch_threads", return_value=()), patch(
        "gdoc.restyle.export_with_media",
        return_value=Export(markdown="# Body\n", images=(), warnings=()),
    ), patch("gdoc.restyle.generate", side_effect=explode):
        with pytest.raises(RuntimeError) as error:
            restyle(_drive(), SOURCE, folder_id="0AFolder")

    assert "could not place a table" in str(error.value)
    assert str(seen[0].parent) in str(error.value)
    assert seen[0].exists()


def test_a_document_that_exists_is_never_hidden_by_a_failing_comment_copy():
    """cli.cmd_generate's rule, and it applies here for the same reason: once
    Drive has created the document, nothing below may withhold its id."""
    with patch("gdoc.restyle.fetch_threads", return_value=(_thread(),)), patch(
        "gdoc.restyle.export_with_media",
        return_value=Export(markdown="# Body\n", images=(), warnings=()),
    ), patch("gdoc.restyle.generate", return_value=_published()), patch(
        "gdoc.restyle.copy_them", side_effect=RuntimeError("boom")
    ):
        result = restyle(_drive(), SOURCE, folder_id="0AFolder")

    assert result.doc_id == NEW
    assert result.link == LINK
    assert result.comments_copied == 0
    assert "boom" in result.comment_errors[0]


def test_a_document_with_no_name_asks_for_a_title():
    """Otherwise it reaches frontmatter as a blank title, which reads as a bug."""
    with patch("gdoc.restyle.fetch_threads", return_value=()), patch(
        "gdoc.restyle.export_with_media"
    ) as pull:
        with pytest.raises(ValueError, match="--title"):
            restyle(_drive(name=""), SOURCE, folder_id="0AFolder")
    pull.assert_not_called()
