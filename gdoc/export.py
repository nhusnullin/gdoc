"""Get a Google Doc as markdown.

Drive can export a Doc straight to markdown. That path is preferred because it
avoids a temp file and a subprocess. It is new enough that it may not be
available for every document, so a docx export through pandoc is kept as a
fallback: pandoc is already required for generating new versions, so the
fallback adds a code path but no new dependency.
"""

import io
import re
import shutil
import subprocess
import tempfile
import zipfile
from dataclasses import dataclass
from pathlib import Path

from googleapiclient.errors import HttpError

MARKDOWN_MIME = "text/markdown"
DOCX_MIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

# Where Homebrew puts it on an Apple silicon Mac. A fallback, not the answer: this
# path was hardcoded once and meant the tool only ran on one machine.
HOMEBREW_PANDOC = "/opt/homebrew/bin/pandoc"


class PandocNotFound(RuntimeError):
    """pandoc is needed here and could not be found."""


def find_pandoc() -> str:
    """Locate pandoc, at call time rather than at import time.

    Resolved on every call on purpose. Resolving once at import would freeze the
    answer for the life of the process, so installing pandoc while the tool is
    running, or shipping a wheelhouse that adds it later, would not take effect.
    """
    found = shutil.which("pandoc")
    if found:
        return found
    if Path(HOMEBREW_PANDOC).is_file():
        return HOMEBREW_PANDOC
    raise PandocNotFound(
        "pandoc is not installed, or not on PATH. It is needed to read markdown. "
        "Install it and make sure `pandoc` runs from your shell, or put it at "
        f"{HOMEBREW_PANDOC}."
    )

# Statuses that mean "this conversion is not offered", as opposed to a real fault.
_UNSUPPORTED = (400, 415)


def _pandoc_docx_to_markdown(docx_bytes: bytes) -> str:
    with tempfile.TemporaryDirectory() as tmp:
        docx_path = Path(tmp) / "doc.docx"
        docx_path.write_bytes(docx_bytes)
        result = subprocess.run(
            [find_pandoc(), str(docx_path), "-f", "docx", "-t", "markdown", "--wrap=none"],
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout


def export_markdown(drive, doc_id: str) -> str:
    try:
        data = drive.files().export(fileId=doc_id, mimeType=MARKDOWN_MIME).execute()
        return data.decode("utf-8") if isinstance(data, bytes) else data
    except HttpError as error:
        if error.resp.status not in _UNSUPPORTED:
            raise
    docx = drive.files().export(fileId=doc_id, mimeType=DOCX_MIME).execute()
    return _pandoc_docx_to_markdown(docx)


# ---------------------------------------------------------------- pictures --
# Drive's markdown export drops embedded pictures and Google Drawings entirely.
# Verified against a real document on 2026-08-18: four drawings, and not one
# image reference of any kind in the markdown it returned. The docx export
# carries all four as PNG bytes, so the docx route is the only one that can
# bring a picture across, and asking for markdown first would only waste a call.
#
# A picture that does not arrive is the failure this exists to remove, so
# anything that could not be carried is reported rather than dropped.

_IMAGE_LINK = re.compile(r"!\[([^\]]*)\]\(([^)\s]+)\)")


@dataclass(frozen=True)
class Export:
    markdown: str
    images: tuple[Path, ...]
    warnings: tuple[str, ...]


def _pandoc_docx_with_media(docx_path: str, extract_to: str) -> str:
    result = subprocess.run(
        [
            find_pandoc(),
            str(docx_path),
            "-f",
            "docx",
            "-t",
            "markdown",
            "--wrap=none",
            f"--extract-media={extract_to}",
        ],
        capture_output=True,
        text=True,
        check=True,
    )
    return result.stdout


def relocate_media(markdown: str, extracted_root: Path, media_dir: Path, out_path: Path) -> Export:
    """Move the extracted pictures beside the markdown, and relink them.

    pandoc writes absolute paths into a directory of its own choosing, nested
    one level deeper than the one it was given. Neither is any use in a vault
    that gets moved or synced, so the files land in one flat directory and the
    links become relative to the markdown that points at them.
    """
    extracted_root = Path(extracted_root)
    media_dir = Path(media_dir)
    written: list[Path] = []
    used: set[Path] = set()

    def move(match):
        alt, target = match.group(1), match.group(2)
        source = Path(target)
        if not source.is_absolute() or not source.is_file():
            return match.group(0)
        media_dir.mkdir(parents=True, exist_ok=True)
        destination = media_dir / source.name
        shutil.copyfile(source, destination)
        used.add(source.resolve())
        written.append(destination)
        relative = _relative(destination, out_path)
        return f"![{alt}]({relative})"

    rewritten = _IMAGE_LINK.sub(move, markdown)

    warnings = tuple(
        f"{found.name} was in the document but nothing in the text points at it, "
        "so it was not carried across"
        for found in sorted(extracted_root.rglob("*"))
        if found.is_file() and found.resolve() not in used
    )
    return Export(markdown=rewritten, images=tuple(written), warnings=warnings)


def _relative(target: Path, out_path: Path) -> str:
    """The link to write, relative to the markdown file that carries it."""
    try:
        return str(target.resolve().relative_to(Path(out_path).resolve().parent))
    except ValueError:
        # Outside the markdown's own tree. An absolute path is honest, and the
        # publish will still find it.
        return str(target.resolve())


def export_with_media(drive, doc_id: str, media_dir: Path, out_path: Path) -> Export:
    """Export the document, pictures included.

    Always the docx route: the markdown export has no pictures in it.
    """
    docx = drive.files().export(fileId=doc_id, mimeType=DOCX_MIME).execute()
    with tempfile.TemporaryDirectory() as tmp:
        docx_path = Path(tmp) / "doc.docx"
        docx_path.write_bytes(docx)
        extract_to = Path(tmp) / "extracted"
        markdown = _pandoc_docx_with_media(str(docx_path), str(extract_to))
        export = relocate_media(markdown, extract_to, Path(media_dir), Path(out_path))
        return Export(
            markdown=export.markdown,
            images=export.images,
            warnings=export.warnings + _missing_from_the_docx(docx, export.images),
        )


def _missing_from_the_docx(docx: bytes, carried) -> tuple[str, ...]:
    """Pictures Google put in the docx that never reached the markdown.

    pandoc extracts only what the body references, so a header logo or a drawing
    it could not place is dropped without a word. Verified against a real
    document on 2026-08-18: four images in the docx, three in the markdown.
    """
    carried_names = {path.name for path in carried}
    try:
        with zipfile.ZipFile(io.BytesIO(docx)) as archive:
            inside = [
                Path(name).name
                for name in archive.namelist()
                if name.startswith("word/media/")
            ]
    except (zipfile.BadZipFile, OSError):
        return ()
    return tuple(
        f"{name} is a picture in the document that could not be carried across. "
        "It is usually a header image, or a drawing the converter could not "
        "place. Add it by hand if it matters."
        for name in sorted(set(inside) - carried_names)
    )
