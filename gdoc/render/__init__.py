"""Put markdown into a house template and produce a .docx.

The pipeline: parse the front matter, copy the master and fill its front
matter, splice the rendered body in, then let headless LibreOffice refresh the
contents list so its page numbers are real.

The master is copied and edited, never rebuilt. That is what keeps the cover,
logo, running head, footer and coloured tables pixel-identical to the original.
"""

import datetime
import os
import re
import unicodedata
from dataclasses import dataclass
from pathlib import Path

from docx import Document

from gdoc.render import body, frontmatter, profiles, shell
from gdoc.render.ooxml import Numbering
from gdoc.render.toc import TocError, refresh_toc


class BuildError(RuntimeError):
    """Anything that should stop the build with a message rather than a stack."""


@dataclass(frozen=True)
class BuildResult:
    docx_path: Path
    pdf_path: Path | None
    title: str
    template: str
    blocks: int


def slugify(text: str) -> str:
    normalised = unicodedata.normalize("NFKD", text)
    ascii_only = normalised.encode("ascii", "ignore").decode("ascii")
    return re.sub(r"-+", "-", re.sub(r"[^a-z0-9]+", "-", ascii_only.lower())).strip("-")


def _today() -> str:
    return datetime.date.today().isoformat()


def default_output(meta: dict, source: Path) -> Path:
    """Name the output YYYY-MM-DD-<title-slug>.docx, beside the source."""
    date_prefix = os.environ.get("DOC_DATE") or _today()
    stem = slugify(meta["cover_title"]) or source.stem
    return source.resolve().parent / f"{date_prefix}-{stem}.docx"


def meta_for(md_path: Path, *, title: str | None = None) -> dict:
    """The front matter of a file, without building anything."""
    text = Path(md_path).read_text(encoding="utf-8")
    meta, _ = frontmatter.parse(text, title_override=title)
    return meta


def build(
    md_path: Path,
    out_path: Path | None = None,
    *,
    template: str = profiles.DEFAULT_TEMPLATE,
    title: str | None = None,
    want_pdf: bool = False,
    skip_toc: bool = False,
) -> BuildResult:
    source_path = Path(md_path).resolve()
    if not source_path.is_file():
        raise BuildError(f"input file not found: {source_path}")

    master = profiles.resolve(template)
    if master is None:
        raise BuildError(
            "build needs a template. Use gdoc generate for the plain pandoc path."
        )

    markdown_text = source_path.read_text(encoding="utf-8")
    meta, body_markdown = frontmatter.parse(markdown_text, title_override=title)
    if not body_markdown.strip():
        raise BuildError("the file has front matter but no body content")

    output_path = Path(out_path).resolve() if out_path else default_output(meta, source_path)
    output_path.parent.mkdir(parents=True, exist_ok=True)

    shell.build_shell(master, output_path, meta)

    doc = Document(output_path)
    blocks = body.render(
        doc,
        body_markdown,
        Numbering(doc),
        heading_numbering=meta["heading_numbering"],
        base_dir=source_path.parent,
    )
    doc.save(output_path)

    pdf_path = output_path.with_suffix(".pdf") if want_pdf else None
    if skip_toc:
        if want_pdf:
            raise BuildError(
                "a PDF needs the contents-list refresh, so --pdf cannot be "
                "combined with --skip-toc"
            )
    else:
        refresh_toc(output_path, pdf_path)
        # After the refresh, never before: LibreOffice creates the TOC styles
        # during that pass, and leaves each level's right tab pulled in by its
        # own indent.
        shell.normalise_toc_tabs(output_path)

    return BuildResult(
        docx_path=output_path,
        pdf_path=pdf_path,
        title=meta["cover_title"],
        template=template,
        blocks=blocks,
    )
