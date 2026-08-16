"""Put markdown into a house template and produce a .docx.

The pipeline: parse the front matter, copy the master and fill its front matter,
splice the rendered body in, then write the contents list.

The master is copied and edited, never rebuilt. That is what keeps the cover,
logo, running head, footer and coloured tables pixel-identical to the original.

Nothing here calls an external program. Page numbers are the caller's to supply,
because they do not exist until something lays the document out. gdoc generate
gets them from Google. A caller with none, such as an offline build, gets a
contents list with correct entries and blank page numbers, which is honest.
"""

import datetime
import os
import re
import unicodedata
from dataclasses import dataclass
from pathlib import Path
from typing import Mapping

from docx import Document

from gdoc.render import body, contents, frontmatter, profiles, shell
from gdoc.render.ooxml import Numbering


class BuildError(RuntimeError):
    """Anything that should stop the build with a message rather than a stack."""


@dataclass(frozen=True)
class BuildResult:
    docx_path: Path
    title: str
    template: str
    blocks: int
    entries: int


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


def drop_title_heading(body_markdown: str, title: str) -> str:
    """Remove a leading H1 that repeats the title.

    Without this the title prints twice, once on the cover and again above the
    first paragraph. Keyed on the text matching, so it also helps a hand-written
    front matter whose title repeats the note's own heading.
    """
    lines = body_markdown.splitlines()
    for index, line in enumerate(lines):
        if not line.strip():
            continue
        match = frontmatter.H1_RE.match(line)
        if match and match.group(1).strip() == title.strip():
            return "\n".join(lines[index + 1:]).lstrip("\n")
        return body_markdown
    return body_markdown


def meta_for(md_path: Path, *, title: str | None = None) -> dict:
    """The front matter of a file, without building anything."""
    text = Path(md_path).read_text(encoding="utf-8")
    meta, _ = frontmatter.parse(
        text, title_override=title, source_name=Path(md_path).name
    )
    return meta


def build(
    md_path: Path,
    out_path: Path | None = None,
    *,
    template: str = profiles.DEFAULT_TEMPLATE,
    title: str | None = None,
    pages: Mapping | None = None,
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
    meta, body_markdown = frontmatter.parse(
        markdown_text, title_override=title, source_name=source_path.name
    )
    body_markdown = drop_title_heading(body_markdown, meta["title"])
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

    written = contents.rewrite_in_place(output_path, pages=pages)

    return BuildResult(
        docx_path=output_path,
        title=meta["cover_title"],
        template=template,
        blocks=blocks,
        entries=len(written.entries),
    )
