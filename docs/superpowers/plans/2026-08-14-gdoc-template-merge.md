# gdoc template merge and version bookkeeping, implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the Altery document renderer into this repo so `gdoc` produces house-style documents itself, and make `generate` record the version it just created.

**Architecture:** The renderer is ported, not rewritten, from the `altery-doc-template` skill into a new `gdoc/render/` subpackage, with the master template as a bundled profile under `gdoc/templates/`. A new `gdoc build` command renders locally with no Drive call; `gdoc generate` renders through the same code and then uploads. After a successful upload, and only when `--baseline-root` is passed, `generate` writes the pairing frontmatter that until now nothing wrote.

**Tech Stack:** Python 3.11, python-docx, lxml, PyYAML, pypdf, pandoc (parser only), pytest.

**Spec:** [docs/superpowers/specs/2026-08-14-gdoc-template-merge-design.md](../specs/2026-08-14-gdoc-template-merge-design.md)

---

## Amendment, 2026-08-15: read this before any task

**Task 1 is done and merged.** Tasks 2, 3 and 6 below are written against code that
no longer exists, and Task 10 is partly done. Tasks 4, 5, 7, 8 and 9 are unaffected.

PR #11 removed headless LibreOffice and poppler from the publish path.
`gdoc/render/contents.py` now writes the contents list itself and
`gdoc/render/pagination.py` gets page numbers from Google. The spec's
"Amendment, 2026-08-15" section explains why, and the evidence is in
[the findings](../specs/2026-08-14-gdoc-toc-libreoffice-findings.md).

### What no longer exists

`gdoc/render/toc.py`, `scripts/update-toc.sh`, `scripts/UpdateToc.bas`,
`shell.normalise_toc_tabs`, `refresh_toc`, `TocError`, `build(want_pdf=...)`,
`build(skip_toc=...)` and `BuildResult.pdf_path`. This plan still mentions them
**31 times**. Every one of those mentions is stale.

### The current interface

```python
gdoc.render.build(md_path, out_path=None, *, template=..., title=None, pages=None)
    -> BuildResult(docx_path, title, template, blocks, entries)

gdoc.render.contents.write(src, dst, pages=None) -> ContentsResult
gdoc.render.contents.rewrite_in_place(path, pages=None) -> ContentsResult
gdoc.render.pagination.from_pdf(pdf, headings) -> dict[str, int]
gdoc.render.pagination.drift(written, published) -> dict
```

### Task status

| Task | Do this |
|---|---|
| 1. Port the renderer | **Done and merged.** Skip it. |
| 2. Port the 25 checks | **Port what is meaningful, drop both pixel comparisons.** Decided, see below. |
| 3. `gdoc build` + `template` key | **Mostly dropped.** Keep the config key and fold it into Task 6. Do not build the command. |
| 4. The title contract | Unchanged. |
| 5. Drop the title heading | Unchanged. |
| 6. `generate` through the template | **Amend and expand.** See below. |
| 7. `generate` records the version | Unchanged. |
| 8. `pair add-version` | Unchanged. |
| 9. Rewrite the `gdoc-apply` skill | Unchanged. |
| 10. README and CLAUDE.md | **Partly done** by PR #11. |

### Task 2, redesign

Its two pixel comparisons run `build(want_pdf=True)`, then
`soffice --headless --convert-to pdf`, then `pdftoppm`. None of that exists, and its
`_tools_present(...)` gate would skip the whole file today.

The metric was also wrong. Measured on one document: **22.7% of pixels differing for
harmless drift, 3.1% for a contents page describing a different document.** It ranked
the defects backwards. And `tests/support/pdfdiff.py` cannot compare a local render
against a Google export at all, because they rasterise to 1241x1754 and 1242x1755 and
it refuses on a size mismatch, so it could never see the published look it existed to
protect.

**Decided by Nail, 2026-08-15: port the 25 checks that still mean something, and drop
both pixel comparisons.** Fidelity is covered by the assertion that is already merged
and green: the contents entries must match the document's real headings, with page
numbers inside the page count. It catches the defect that would actually reach a
committee, and it runs offline with no external program.

Rebuilding a visual check against Google's export is **issue #13**, so it is recorded
rather than lost. Do not build it as part of this task.

### Task 3, mostly dropped

This task bundles two unrelated things. One is needed, one is not.

**Keep `Config.template`.** Task 6 consumes it directly, at
`template = args.template or config.template`, so `generate` has no way to know which
template to render through without it. It is a dataclass field, a default of
`profiles.DEFAULT_TEMPLATE`, and a line in `load_config`. **Fold it into Task 6**,
where it is used, and keep the two config tests that come with it.

**Do not build the `gdoc build` command.** Three reasons:

1. Nail does not want local files. His words: "for me generate only google doc is
   enough, i do not need docx and pdf to be honest." A local `.docx` is exactly what
   this command produces.
2. It got worse since the plan was written. `build` used to run LibreOffice and
   produce a document with real page numbers. Offline it now writes **blank** ones,
   because nothing has laid the document out. So the command's output is a
   half-product: correct entries, correct links, empty page column.
3. Nothing needs it. The tests call `gdoc.render.build()` directly in Python. A CLI
   wrapper is a surface to document, version and maintain, with no caller.

If a preview is ever wanted, the honest shape is `--dry-run` on `generate`, not a
separate command, because `generate` is the thing that knows how to get page numbers.

No README change is needed. It already says `gdoc.render.build` "is a function, not a
command", and lists the subcommands that actually exist.

### Task 6, amend and expand

It also absorbs `Config.template` from Task 3, since it is the only consumer:

```python
@dataclass(frozen=True)
class Config:
    output_folder_id: str | None = None
    template: str = profiles.DEFAULT_TEMPLATE
```

read in `load_config`, and used as `template = args.template or config.template`.

The rest was "render through the template". It is now that plus the two-pass:

1. `build` with no pages, so the contents list carries blank page numbers
2. upload, export the PDF, `pagination.from_pdf` to learn the real pages
3. `build` again with those pages
4. upload the version that gets published, then assert `pagination.drift` is empty

`tests/test_contents_integration.py` is the working reference for all four steps.
Delete the `skipif soffice` gate that Task 6 currently carries.

**Do this task first.** Until it lands, `gdoc generate` still runs
`pandoc md -o docx`, so both skills publish plain documents with no house style, and
none of the merged renderer work reaches a real document.

## Global Constraints

- **Source of the port:** `/Users/nailkhusnullin/src/altery/Altery-Platform-Hub/.claude/skills/altery-doc-template/`. Referred to below as `$SRC`.
- **Never modify `$SRC`.** Not the `.claude/skills/` copy, not the identical `.agents/skills/` copy. Copy files out of it and leave it working exactly as it is. Retiring it is Nail's call, and not part of this plan.
- **Ported code is copied, not rewritten.** Change imports, the template path and the `sys.dont_write_bytecode` hack. Nothing else. The pixel fidelity lives in measured constants and in OOXML child ordering.
- **Do not split `body.py` (578 lines) or `shell.py` (625 lines).** They arrive as they are. The spec defers that deliberately.
- **Test command:** `~/.config/gdoc-agent/venv/bin/pytest`. There is no other environment. No `uv` anywhere in this repo.
- **TDD:** the failing test comes first, and you run it and see it fail before writing the implementation.
- **Commits:** `<type>: <description>`, types `feat, fix, refactor, docs, test, chore`. Commit at the end of every task. **Never push.**
- **Writing style for any prose you add**, in docs, skills, docstrings and error messages: plain short English, no em dash characters (`—`) anywhere.
- **The tool must work without git.** Nothing may refuse to run because git is missing.
- **Never write tool files into this repo at runtime.** `.gdoc/` lives beside the source markdown being reviewed.

---

## File Structure

**Created:**

| Path | Responsibility |
|---|---|
| `gdoc/render/__init__.py` | `build()`, the one entry point. Front matter, shell, body, contents list, in order |
| `gdoc/render/frontmatter.py` | Front matter parsing, validation, and the title candidate |
| `gdoc/render/body.py` | Markdown to template-idiomatic OOXML. Ported unchanged |
| `gdoc/render/ooxml.py` | Element helpers and the constants measured off the master. Ported unchanged |
| `gdoc/render/shell.py` | Template surgery: cover, tables, header, comment stripping. Ported unchanged |
| ~~`gdoc/render/toc.py`~~ | ~~`refresh_toc`, the LibreOffice driver~~ **Deleted by PR #11.** Replaced by `gdoc/render/contents.py`, which writes the contents list, and `gdoc/render/pagination.py`, which gets page numbers from Google |
| `gdoc/render/profiles.py` | Resolves a `--template` value to a master `.docx` |
| `gdoc/render/scripts/update-toc.sh`, `UpdateToc.bas` | The LibreOffice macro. Template-agnostic, so not per profile |
| `gdoc/templates/altery-group-policy-v1.0/template.docx` | The master |
| `gdoc/templates/altery-group-policy-v1.0/example.md` | The worked example, and the test fixture |
| `gdoc/templates/altery-group-policy-v1.0/README.md` | The front-matter reference |
| `tests/support/pdfdiff.py` | The pixel differ. A helper, not a test |
| `tests/support/checks.py` | The ported `check_*` functions and their `Result` collector |
| `tests/test_render_fidelity.py` | The two pixel comparisons against the master |
| `tests/test_render_structure.py` | Styles, numbering, contents list, headings, tables, cover |
| `tests/test_render_frontmatter.py` | Front matter validation and the title candidate |
| `tests/test_render_profiles.py` | Template resolution |
| `tests/test_build_cli.py` | The `gdoc build` command |

**Modified:**

| Path | Change |
|---|---|
| `gdoc/generate.py` | `generate()` renders through `render.build()` when a template is in play |
| `gdoc/cli.py` | New `build` subcommand; `generate` gains `--template`, `--name` becomes optional, and it writes the pairing |
| `gdoc/config.py` | New optional `template` key |
| `pyproject.toml` | `python-docx`, `lxml`, the `gdoc.render` package, package data, the `slow` marker |
| `tests/conftest.py` | The session-scoped built-document fixture |
| `tests/test_generate.py`, `tests/test_cli.py`, `tests/test_pairing.py` | New cases |
| `skills/gdoc-apply/SKILL.md` | First run, the title step, Step 4 simplified, Step 5 deleted |
| `README.md`, `CLAUDE.md` | The new command, the new dependency, the bundled profile |

---

## Task 1: Port the renderer into `gdoc/render/` [DONE, MERGED]

> **Do not run this task.** It shipped in commit `9de4523` and merged to `main` in
> PR #11. `gdoc/render/toc.py` was deleted afterwards; see the amendment at the top.

**Files:**
- Create: `gdoc/render/__init__.py`, `frontmatter.py`, `body.py`, `ooxml.py`, `shell.py`, `toc.py`, `profiles.py`, `scripts/update-toc.sh`, `scripts/UpdateToc.bas`
- Create: `gdoc/templates/altery-group-policy-v1.0/template.docx`, `example.md`, `README.md`
- Modify: `pyproject.toml`
- Test: `tests/test_render_profiles.py`, `tests/test_render_smoke.py`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `gdoc.render.build(md_path: Path, out_path: Path | None = None, *, template: str | None = None, title: str | None = None, want_pdf: bool = False, skip_toc: bool = False) -> BuildResult`
  - `BuildResult` is a frozen dataclass: `docx_path: Path`, `pdf_path: Path | None`, `title: str`, `template: str`, `blocks: int`
  - `gdoc.render.BuildError(RuntimeError)`
  - `gdoc.render.profiles.resolve(spec: str) -> Path | None` returning the master `.docx`, or `None` for `"none"`
  - `gdoc.render.profiles.ProfileError(RuntimeError)`
  - `gdoc.render.profiles.DEFAULT_TEMPLATE = "altery-group-policy-v1.0"`
  - `gdoc.render.frontmatter.parse(markdown_text, *, title_override=None, source_name=None) -> (meta, body_markdown)`

- [ ] **Step 1: Copy the files out of the skill, without touching it**

```bash
SRC="$HOME/src/altery/Altery-Platform-Hub/.claude/skills/altery-doc-template"
mkdir -p gdoc/render/scripts gdoc/templates/altery-group-policy-v1.0 tests/support

cp "$SRC/scripts/body.py"        gdoc/render/body.py
cp "$SRC/scripts/ooxml.py"       gdoc/render/ooxml.py
cp "$SRC/scripts/shell.py"       gdoc/render/shell.py
cp "$SRC/scripts/frontmatter.py" gdoc/render/frontmatter.py
cp "$SRC/scripts/update-toc.sh"  gdoc/render/scripts/update-toc.sh
cp "$SRC/scripts/UpdateToc.bas"  gdoc/render/scripts/UpdateToc.bas
cp "$SRC/scripts/pdfdiff.py"     tests/support/pdfdiff.py

cp "$SRC/assets/altery-group-policy-template-v1.0.docx" \
   gdoc/templates/altery-group-policy-v1.0/template.docx
cp "$SRC/assets/example-policy.md" gdoc/templates/altery-group-policy-v1.0/example.md
cp "$SRC/references/front-matter.md" gdoc/templates/altery-group-policy-v1.0/README.md

chmod +x gdoc/render/scripts/update-toc.sh
touch tests/support/__init__.py
```

Do not copy `build.py` or `selftest.py`. Their logic is rewritten in this task and Task 2.

- [ ] **Step 2: Fix the imports in the copied modules**

Three edits, and nothing else in these three files.

In `gdoc/render/body.py`, line 19:

```python
# was: from ooxml import (BODY_FONT, BODY_SZ, CELL_BORDER, CELL_MARGIN_H,
from gdoc.render.ooxml import (BODY_FONT, BODY_SZ, CELL_BORDER, CELL_MARGIN_H,
```

Keep the rest of that import's continuation lines exactly as they are, adjusting only their indentation to line up with the new opening bracket.

Also in `gdoc/render/body.py`, in `parse_markdown`, use the same pandoc the rest of the tool uses, so a machine without Homebrew behaves consistently:

```python
from gdoc.export import PANDOC   # add near the top, after the ooxml import
```
```python
        result = subprocess.run(
            [PANDOC, "-f", PANDOC_FORMAT, "-t", "json"],
```

In `gdoc/render/shell.py`, the `from ooxml import (...)` block becomes `from gdoc.render.ooxml import (...)`, again with only the indentation adjusted.

One more edit, in `gdoc/render/frontmatter.py`, because `build()` needs it in Task 3 and it is two lines. Change the signature:

```python
def parse(markdown_text, *, title_override=None):
```

and insert this immediately after `data, body = split(markdown_text)`:

```python
    if title_override:
        data = {**data, "title": title_override}
```

Nothing else in that file changes here. Task 4 rewrites the missing-title behaviour.

- [ ] **Step 3: Write the failing test for template resolution**

Create `tests/test_render_profiles.py`:

```python
import pytest

from gdoc.render import profiles


def test_a_bundled_name_resolves_to_its_master_docx():
    master = profiles.resolve("altery-group-policy-v1.0")
    assert master.name == "template.docx"
    assert master.is_file()


def test_the_default_is_the_altery_profile():
    assert profiles.resolve(profiles.DEFAULT_TEMPLATE).is_file()


def test_none_means_no_template():
    assert profiles.resolve("none") is None


def test_a_path_to_a_docx_is_used_as_the_master(tmp_path):
    master = tmp_path / "house.docx"
    master.write_bytes(b"PK\x03\x04")
    assert profiles.resolve(str(master)) == master


def test_a_path_to_a_profile_directory_finds_its_template(tmp_path):
    profile = tmp_path / "house"
    profile.mkdir()
    (profile / "template.docx").write_bytes(b"PK\x03\x04")
    assert profiles.resolve(str(profile)) == profile / "template.docx"


def test_an_unknown_name_lists_what_is_available():
    with pytest.raises(profiles.ProfileError) as exc:
        profiles.resolve("no-such-template")
    assert "altery-group-policy-v1.0" in str(exc.value)
```

- [ ] **Step 4: Run it and watch it fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_render_profiles.py -v`
Expected: FAIL, `ModuleNotFoundError: No module named 'gdoc.render'`.

- [ ] **Step 5: Write `gdoc/render/profiles.py`**

```python
"""Resolve a template choice to the master .docx it names.

A profile is a directory holding template.docx. Bundled profiles live in
gdoc/templates/, and the choice is a config value rather than a constant, so
nothing in the code branches on which house style is in use.

There is no plugin API for the surgery, deliberately. shell.py finds the cover
by placeholder text and the tables by their first-column labels, so a .docx that
does not follow that contract fails there, with a message naming what was not
found. When a second real template arrives, this is where a per-profile surgery
module hooks in.
"""

from pathlib import Path

TEMPLATES_DIR = Path(__file__).resolve().parent.parent / "templates"
DEFAULT_TEMPLATE = "altery-group-policy-v1.0"
MASTER_NAME = "template.docx"
NO_TEMPLATE = "none"


class ProfileError(RuntimeError):
    """The named template does not exist, or does not hold a master."""


def available() -> list[str]:
    if not TEMPLATES_DIR.is_dir():
        return []
    return sorted(p.name for p in TEMPLATES_DIR.iterdir() if (p / MASTER_NAME).is_file())


def resolve(spec: str) -> Path | None:
    """Return the master .docx for this spec, or None for the plain pandoc path.

    A spec that exists on disk is a path, either to a .docx or to a profile
    directory. Anything else is the name of a bundled profile.
    """
    if spec == NO_TEMPLATE:
        return None
    candidate = Path(spec).expanduser()
    if candidate.is_file():
        return candidate
    if candidate.is_dir():
        master = candidate / MASTER_NAME
        if not master.is_file():
            raise ProfileError(f"{candidate} holds no {MASTER_NAME}")
        return master
    master = TEMPLATES_DIR / spec / MASTER_NAME
    if not master.is_file():
        raise ProfileError(
            f"unknown template {spec!r}. Bundled templates: {available()}. "
            "A path to a .docx or to a profile directory also works, "
            f"and {NO_TEMPLATE!r} means no template at all."
        )
    return master
```

- [ ] **Step 6: Run the profile tests and watch them pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_render_profiles.py -v`
Expected: 6 passed.

- [ ] **Step 7: Write `gdoc/render/toc.py`**

```python
"""Give the contents list real page numbers.

A .docx stores its contents list as a field, and page numbers do not exist
until something lays the document out. Headless LibreOffice is that something.
The same pass repairs the malformed TOC and PAGE fields Google Docs left in the
master, where the field start, instruction and separator all sit in one run.
"""

import shutil
import subprocess
from pathlib import Path

UPDATE_TOC = Path(__file__).resolve().parent / "scripts" / "update-toc.sh"


class TocError(RuntimeError):
    """The contents-list refresh could not run, or failed."""


def refresh_toc(docx_path: Path, pdf_path: Path | None = None) -> None:
    if not shutil.which("soffice"):
        raise TocError(
            "LibreOffice (soffice) is not installed, so the contents list "
            "cannot be given page numbers. Install it with "
            "`brew install --cask libreoffice`, or pass --skip-toc."
        )
    # Two passes on purpose. update-toc.sh exports a PDF *instead of* re-saving
    # the .docx when given an output path, so asking only for the PDF would
    # leave the .docx carrying the master's stale page numbers.
    commands = [[str(UPDATE_TOC), str(docx_path)]]
    if pdf_path:
        commands.append([str(UPDATE_TOC), str(docx_path), str(pdf_path)])
    for command in commands:
        result = subprocess.run(command, capture_output=True, text=True)
        if result.returncode != 0:
            raise TocError(f"the contents-list refresh failed:\n{result.stderr}")
```

- [ ] **Step 8: Write `gdoc/render/__init__.py`**

This is `build.py` from the skill, minus its argparse, minus the rclone upload, and returning a result object instead of printing.

```python
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
```

`TocError` is imported so callers can catch it from `gdoc.render`. Leave the unused-import warning; it is re-exported on purpose.

- [ ] **Step 9: Update `pyproject.toml`**

```toml
dependencies = [
    "google-auth",
    "google-api-python-client",
    "PyYAML",
    "python-docx>=1.1",
    "lxml",
]
```

```toml
[tool.setuptools]
packages = ["gdoc", "gdoc.render"]

[tool.setuptools.package-data]
"gdoc" = ["templates/*/*.docx", "templates/*/*.md"]
"gdoc.render" = ["scripts/*.sh", "scripts/*.bas"]
```

```toml
markers = [
    "integration: calls the live Drive API, skipped without credentials",
    "slow: renders a real document, needs LibreOffice and poppler",
]
```

Then reinstall so the new dependencies land:

```bash
~/.config/gdoc-agent/venv/bin/pip install -q -e ".[dev]"
```

- [ ] **Step 10: Write the smoke test**

Create `tests/test_render_smoke.py`:

```python
import shutil

import pytest

from gdoc.render import build
from gdoc.render.profiles import TEMPLATES_DIR

pytestmark = pytest.mark.slow

EXAMPLE = TEMPLATES_DIR / "altery-group-policy-v1.0" / "example.md"


@pytest.mark.skipif(not shutil.which("soffice"), reason="LibreOffice not installed")
def test_the_bundled_example_builds_into_a_real_docx(tmp_path):
    result = build(EXAMPLE, tmp_path / "out.docx")
    assert result.docx_path.read_bytes()[:2] == b"PK"
    assert result.title
    assert result.blocks > 0


def test_skip_toc_builds_without_libreoffice(tmp_path):
    result = build(EXAMPLE, tmp_path / "out.docx", skip_toc=True)
    assert result.docx_path.exists()


def test_a_missing_source_is_reported_before_anything_else(tmp_path):
    from gdoc.render import BuildError

    with pytest.raises(BuildError):
        build(tmp_path / "nope.md", tmp_path / "out.docx", skip_toc=True)
```

- [ ] **Step 11: Run the whole suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest`
Expected: everything passes, including the three new smoke tests. If `test_the_bundled_example_builds_into_a_real_docx` fails, the port is wrong. Do not adjust the test to suit it. Compare the failing module against `$SRC` and fix the copy.

- [ ] **Step 12: Look at the result, which is this task's real exit condition**

```bash
~/.config/gdoc-agent/venv/bin/python -c "
from pathlib import Path
from gdoc.render import build
from gdoc.render.profiles import TEMPLATES_DIR
example = TEMPLATES_DIR / 'altery-group-policy-v1.0' / 'example.md'
print(build(example, Path('/tmp/port-check.docx'), want_pdf=True))
"
pdftoppm -png -r 80 -f 1 -l 3 /tmp/port-check.pdf /tmp/port-check && open /tmp/port-check-1.png
```

Page 1 is the cover with the logo and the title. Page 2 is Version Control, revision history and Document Classification. Page 3 is the contents list with page numbers that are not all the same. If any of that is wrong, the port is wrong.

- [ ] **Step 13: Commit**

```bash
git add gdoc/render gdoc/templates tests/support tests/test_render_profiles.py tests/test_render_smoke.py pyproject.toml
git commit -m "feat: port the document renderer into gdoc/render"
```

---

## Task 2: Port the 25 checks into pytest [AMEND, PIXEL COMPARISONS DROPPED]

> **Read the amendment at the top before starting.** The steps below call
> `build(want_pdf=True)`, `soffice --headless`, and `pdftoppm`. None of those exist.
>
> Nail decided on 2026-08-15: port the checks that still mean something, drop both
> pixel comparisons. A visual check against Google's export is issue #13.

**Files:**
- Create: `tests/support/checks.py`, `tests/test_render_fidelity.py`, `tests/test_render_structure.py`
- Modify: `tests/conftest.py`
- Test: the two new test modules are the test

**Interfaces:**
- Consumes: `gdoc.render.build`, `tests/support/pdfdiff.py`
- Produces: a session fixture `built` with attributes `docx`, `pdf`, `workdir`, and `template_pdf`

- [ ] **Step 1: Copy the check functions**

Copy `$SRC/scripts/selftest.py` to `tests/support/checks.py`, then edit only these parts:

1. Delete `main()`, the `if __name__` block and the `argparse` import.
2. Delete `build()` and `render_template_pdf()`. The fixture replaces them.
3. Change `import pdfdiff` to `from tests.support import pdfdiff`.
4. Change `from ooxml import (...)` to `from gdoc.render.ooxml import (...)`.
5. Replace the `SKILL_ROOT`, `SCRIPTS`, `TEMPLATE` and `EXAMPLE` constants with:

```python
from gdoc.render.profiles import TEMPLATES_DIR

PROFILE = TEMPLATES_DIR / "altery-group-policy-v1.0"
TEMPLATE = PROFILE / "template.docx"
EXAMPLE = PROFILE / "example.md"
```

6. In `check_heading_level_shift`, `check_minimal_front_matter` and `check_rejects_bad_input`, which each build a document in a workdir by running `uv run build.py`, replace the subprocess call with a direct call:

```python
from gdoc.render import BuildError, build as render_build
from gdoc.render.frontmatter import FrontMatterError
```

A check that expects a successful build calls `render_build(source, output, skip_toc=True)`. A check that expects a refusal wraps it:

```python
    try:
        render_build(source, output, skip_toc=True)
        result.add(False, name, "the build accepted input it should have refused")
    except (BuildError, FrontMatterError):
        result.add(True, name)
```

Keep every other check body byte-identical. They are the reason this port is safe.

Leave the `Result` class where it is. The tests use it.

- [ ] **Step 2: Add the fixture to `tests/conftest.py`**

```python
import shutil
import subprocess
from dataclasses import dataclass
from pathlib import Path

import pytest


def _tools_present(*names):
    return all(shutil.which(name) for name in names)


@dataclass(frozen=True)
class Built:
    docx: Path
    pdf: Path
    template_pdf: Path
    workdir: Path


@pytest.fixture(scope="session")
def built(tmp_path_factory):
    """Build the bundled example once, and render both PDFs for comparison."""
    if not _tools_present("soffice", "pandoc", "pdftoppm", "pdftotext", "pdfinfo"):
        pytest.skip("needs LibreOffice, pandoc and poppler")
    from gdoc.render import build
    from tests.support.checks import EXAMPLE, TEMPLATE

    workdir = tmp_path_factory.mktemp("render")
    result = build(EXAMPLE, workdir / "generated.docx", want_pdf=True)
    subprocess.run(
        ["soffice", "--headless", "--convert-to", "pdf",
         "--outdir", str(workdir), str(TEMPLATE)],
        capture_output=True, check=True,
    )
    return Built(
        docx=result.docx_path,
        pdf=result.pdf_path,
        template_pdf=workdir / (TEMPLATE.stem + ".pdf"),
        workdir=workdir,
    )
```

- [ ] **Step 3: Write the fidelity tests**

Create `tests/test_render_fidelity.py`:

```python
"""The two pixel comparisons against the master.

The cover's header band must match at 0 pixels differing. The footer is allowed
100, which covers the page-number glyph shifting about one pixel when
LibreOffice re-renders the PAGE field. Do not raise a tolerance to make a
failure go away: that is the check that catches a real layout shift.
"""

import pytest

from tests.support import checks

pytestmark = pytest.mark.slow


def test_the_cover_furniture_matches_the_master(built):
    result = checks.Result()
    checks.check_pixels(result, built.template_pdf, built.pdf, built.workdir)
    assert not result.failed, result.failed
```

- [ ] **Step 4: Write the structure tests**

Create `tests/test_render_structure.py`. One test per check, so a failure names itself:

```python
import pytest

from tests.support import checks

pytestmark = pytest.mark.slow


def _run(check, *args):
    result = checks.Result()
    check(result, *args)
    assert not result.failed, result.failed


def test_no_review_comments_survive_into_the_output(built):
    _run(checks.check_no_comments, built.docx)


def test_every_style_referenced_resolves(built):
    _run(checks.check_styles_resolve, built.docx)


def test_list_numbering_is_defined(built):
    _run(checks.check_numbering, built.docx)


def test_the_contents_list_exists(built):
    _run(checks.check_toc, built.docx)


def test_contents_list_tabs_sit_on_the_text_edge(built):
    _run(checks.check_toc_tabs, built.docx)


def test_heading_indents(built):
    _run(checks.check_heading_indents, built.pdf)


def test_heading_spacing_is_house_style(built):
    _run(checks.check_heading_spacing, built.docx)


def test_heading_emphasis_steps_down_by_level(built):
    _run(checks.check_heading_emphasis, built.docx)


def test_tables_are_readable(built):
    _run(checks.check_table_readability, built.docx)


def test_the_front_matter_tables_are_filled(built):
    _run(checks.check_front_matter, built.docx, 2, "Internal (I)")


def test_no_placeholder_text_is_left_on_the_cover(built):
    _run(checks.check_cover_clean, built.docx)


def test_page_layout(built):
    _run(checks.check_page_layout, built.pdf)


def test_heading_levels_are_relative_to_the_shallowest(built, tmp_path):
    _run(checks.check_heading_level_shift, tmp_path)


def test_a_title_only_document_builds(built, tmp_path):
    _run(checks.check_minimal_front_matter, tmp_path)


def test_broken_input_is_refused(built, tmp_path):
    _run(checks.check_rejects_bad_input, tmp_path)
```

- [ ] **Step 5: Run them**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_render_fidelity.py tests/test_render_structure.py -v`
Expected: all pass, or all skip on a machine with no LibreOffice. A failure here means the port changed behaviour. Fix the port, not the check.

- [ ] **Step 6: Run the whole suite and commit**

```bash
~/.config/gdoc-agent/venv/bin/pytest
git add tests/
git commit -m "test: port the template selftest checks into pytest"
```

---

## Task 3: `gdoc build`, and the `template` config key [MOSTLY DROPPED]

> **Read the amendment at the top before starting.** Do not build the `gdoc build`
> command: Nail does not want local files, and offline it would write blank page
> numbers. Take only the `Config.template` half, which Task 6 needs, and implement it
> there. The steps below also still use `--pdf` and `--skip-toc`, which no longer
> exist.

**Files:**
- Modify: `gdoc/config.py`, `gdoc/cli.py`
- Test: `tests/test_config.py`, `tests/test_build_cli.py`

**Interfaces:**
- Consumes: `gdoc.render.build`, `gdoc.render.profiles.DEFAULT_TEMPLATE`
- Produces: `Config.template: str`; the `build` subcommand; `gdoc.cli._fail_with(payload: dict) -> int`

- [ ] **Step 1: Write the failing config test**

Add to `tests/test_config.py`:

```python
def test_template_defaults_to_the_bundled_altery_profile(tmp_path):
    path = tmp_path / "config.json"
    path.write_text('{"output_folder_id": "0AFolder"}')
    assert load_config(path).template == "altery-group-policy-v1.0"


def test_template_can_be_overridden_in_the_config(tmp_path):
    path = tmp_path / "config.json"
    path.write_text('{"template": "none"}')
    assert load_config(path).template == "none"
```

- [ ] **Step 2: Run it and watch it fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_config.py -v`
Expected: FAIL, `AttributeError: 'Config' object has no attribute 'template'`.

- [ ] **Step 3: Add the key**

In `gdoc/config.py`:

```python
from gdoc.render.profiles import DEFAULT_TEMPLATE


@dataclass(frozen=True)
class Config:
    output_folder_id: str | None = None
    template: str = DEFAULT_TEMPLATE
```

and in `load_config`:

```python
    return Config(
        output_folder_id=data.get("output_folder_id"),
        template=data.get("template") or DEFAULT_TEMPLATE,
    )
```

- [ ] **Step 4: Run it and watch it pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_config.py -v`
Expected: PASS.

- [ ] **Step 5: Write the failing CLI test**

Create `tests/test_build_cli.py`:

```python
import json
from unittest.mock import patch

import pytest

from gdoc.cli import main
from gdoc.render.profiles import TEMPLATES_DIR

EXAMPLE = TEMPLATES_DIR / "altery-group-policy-v1.0" / "example.md"


def test_build_renders_a_docx_and_reports_it(capsys, tmp_path):
    out = tmp_path / "out.docx"
    exit_code = main(["build", "--md", str(EXAMPLE), "-o", str(out), "--skip-toc"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 0
    assert payload["docx_path"] == str(out)
    assert payload["template"] == "altery-group-policy-v1.0"
    assert payload["pdf_path"] is None
    assert out.exists()


def test_build_never_calls_drive(tmp_path):
    with patch("gdoc.cli.drive_service") as drive:
        main(["build", "--md", str(EXAMPLE), "-o", str(tmp_path / "o.docx"), "--skip-toc"])
    drive.assert_not_called()


def test_build_reports_an_unknown_template_without_a_stack(capsys, tmp_path):
    exit_code = main([
        "build", "--md", str(EXAMPLE), "-o", str(tmp_path / "o.docx"),
        "--template", "no-such-template", "--skip-toc",
    ])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert "no-such-template" in payload["error"]
```

- [ ] **Step 6: Run it and watch it fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_build_cli.py -v`
Expected: FAIL, argparse exits because `build` is not a subcommand.

- [ ] **Step 7: Add the subcommand**

In `gdoc/cli.py`, add the imports:

```python
from gdoc import render
from gdoc.render import profiles
```

Add the error helper next to `_fail`:

```python
def _fail_with(payload: dict) -> int:
    """Fail with more than a message. The extra keys are for the skill to read."""
    print(json.dumps(payload, indent=2))
    return 1
```

Add the handler:

```python
def cmd_build(args) -> int:
    """Render markdown into a house template. No credential, no network."""
    template = args.template or load_config().template
    try:
        result = render.build(
            Path(args.md),
            Path(args.out) if args.out else None,
            template=template,
            title=args.title,
            want_pdf=args.pdf,
            skip_toc=args.skip_toc,
        )
    except (render.BuildError, profiles.ProfileError, render.TocError) as error:
        return _fail(str(error))
    return _emit({
        "docx_path": str(result.docx_path),
        "pdf_path": str(result.pdf_path) if result.pdf_path else None,
        "title": result.title,
        "template": result.template,
        "blocks": result.blocks,
    })
```

and the parser, next to the `generate` parser:

```python
    build_cmd = sub.add_parser("build", help="markdown to a .docx, no upload")
    build_cmd.add_argument("--md", required=True)
    build_cmd.add_argument("-o", "--out", help="default: YYYY-MM-DD-<title>.docx beside the source")
    build_cmd.add_argument("--template", help="profile name, path, or 'none'")
    build_cmd.add_argument("--title", help="cover title, for a file that should not be edited")
    build_cmd.add_argument("--pdf", action="store_true")
    build_cmd.add_argument("--skip-toc", action="store_true")
    build_cmd.set_defaults(func=cmd_build)
```

`cmd_build` calls `load_config()` only when `--template` is absent, so a machine with no config file can still build with an explicit template.

- [ ] **Step 8: Run the tests and watch them pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_build_cli.py tests/test_config.py -v`
Expected: PASS.

- [ ] **Step 9: Run the whole suite and commit**

```bash
~/.config/gdoc-agent/venv/bin/pytest
git add gdoc/config.py gdoc/cli.py tests/test_config.py tests/test_build_cli.py
git commit -m "feat: add gdoc build and a configurable default template"
```

---

## Task 4: The title contract

**Files:**
- Modify: `gdoc/render/frontmatter.py`, `gdoc/cli.py`
- Test: `tests/test_render_frontmatter.py`, `tests/test_build_cli.py`

**Interfaces:**
- Consumes: `gdoc.render.frontmatter.parse`
- Produces:
  - `frontmatter.MissingTitle(FrontMatterError)` with `.candidate: str` and `.source: str` (`"h1"` or `"filename"`)
  - `frontmatter.title_candidate(body_markdown: str, source_name: str) -> tuple[str, str]`
  - `parse(markdown_text, *, title_override=None, source_name=None)`

- [ ] **Step 1: Write the failing tests**

Create `tests/test_render_frontmatter.py`:

```python
import pytest

from gdoc.render import frontmatter


def test_a_title_only_document_parses():
    meta, body = frontmatter.parse("---\ntitle: Something\n---\n\nBody.\n")
    assert meta["cover_title"] == "Something"
    assert body.strip() == "Body."


def test_a_missing_title_suggests_the_first_h1():
    text = "---\ngdoc: 1abc\n---\n\n# Miguel kickoff call, 2026-08-12\n\nBody.\n"
    with pytest.raises(frontmatter.MissingTitle) as exc:
        frontmatter.parse(text, source_name="2026-08-12-miguel-kickoff-call.md")
    assert exc.value.candidate == "Miguel kickoff call, 2026-08-12"
    assert exc.value.source == "h1"


def test_a_missing_title_falls_back_to_the_filename():
    text = "---\ngdoc: 1abc\n---\n\nJust prose, no heading.\n"
    with pytest.raises(frontmatter.MissingTitle) as exc:
        frontmatter.parse(text, source_name="2026-08-12-miguel-kickoff-call.md")
    assert exc.value.candidate == "Miguel kickoff call"
    assert exc.value.source == "filename"


def test_a_hash_inside_a_code_fence_is_not_a_heading():
    text = "---\ngdoc: 1abc\n---\n\n```\n# not a heading\n```\n\nProse.\n"
    with pytest.raises(frontmatter.MissingTitle) as exc:
        frontmatter.parse(text, source_name="2026-01-01-notes.md")
    assert exc.value.source == "filename"


def test_a_document_with_no_front_matter_at_all_still_suggests_a_title():
    with pytest.raises(frontmatter.MissingTitle) as exc:
        frontmatter.parse("# Kickoff call\n\nBody.\n", source_name="kickoff.md")
    assert exc.value.candidate == "Kickoff call"


def test_the_title_override_wins_and_nothing_is_raised():
    text = "---\ngdoc: 1abc\n---\n\n# Ignored\n\nBody.\n"
    meta, _ = frontmatter.parse(text, title_override="Explicit Title", source_name="x.md")
    assert meta["cover_title"] == "Explicit Title"


def test_an_unrecognised_classification_is_still_refused():
    text = "---\ntitle: X\nclassification: Secret\n---\n\nBody.\n"
    with pytest.raises(frontmatter.FrontMatterError) as exc:
        frontmatter.parse(text)
    assert "Secret" in str(exc.value)
```

- [ ] **Step 2: Run them and watch them fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_render_frontmatter.py -v`
Expected: FAIL, `AttributeError: module 'gdoc.render.frontmatter' has no attribute 'MissingTitle'`.

- [ ] **Step 3: Implement the candidate and the refusal**

In `gdoc/render/frontmatter.py`, after the `FrontMatterError` class:

```python
DATE_PREFIX_RE = re.compile(r"^\d{4}-\d{2}-\d{2}-")
H1_RE = re.compile(r"^#\s+(.+?)\s*#*\s*$")
FENCE_RE = re.compile(r"^\s*(```|~~~)")


class MissingTitle(FrontMatterError):
    """No title, so the cover and running head would be blank.

    Carries a candidate rather than using it. The tool does not invent a cover
    title silently; the skill proposes this one and writes it into the note once
    Nail agrees, so the decision is made once and then reused.
    """

    def __init__(self, candidate, source):
        self.candidate = candidate
        self.source = source
        super().__init__(
            "no title in front matter, so the cover and running head would be blank"
        )


def title_candidate(body_markdown, source_name):
    """Return (candidate, 'h1'|'filename'). Deterministic, so it can be tested."""
    in_fence = False
    for line in body_markdown.splitlines():
        if FENCE_RE.match(line):
            in_fence = not in_fence
            continue
        if in_fence:
            continue
        match = H1_RE.match(line)
        if match:
            return match.group(1).strip(), "h1"
    stem = Path(source_name).stem if source_name else ""
    words = DATE_PREFIX_RE.sub("", stem).replace("-", " ").replace("_", " ").strip()
    return (words[:1].upper() + words[1:]) if words else "Untitled", "filename"
```

Add `from pathlib import Path` to the imports at the top of the module.

Change `split` so a file with no front matter is not an error any more, since the title candidate is more useful than a refusal:

```python
def split(markdown_text):
    """Return (front_matter_dict, body_markdown). No front matter gives {}."""
    match = FRONT_MATTER_RE.match(markdown_text)
    if not match:
        return {}, markdown_text
    ...
```

Keep the rest of `split` as it is.

Change the head of `parse`:

```python
def parse(markdown_text, *, title_override=None, source_name=None):
    """Validate the front matter and return (meta, body_markdown)."""
    data, body = split(markdown_text)

    raw_title = title_override or data.get("title")
    if not raw_title:
        candidate, source = title_candidate(body, source_name or "")
        raise MissingTitle(candidate, source)
```

and further down, replace `title = _as_text(data["title"]).strip()` with:

```python
    title = _as_text(raw_title).strip()
```

Delete the old `REQUIRED` check block. `REQUIRED` itself can stay; `check_rejects_bad_input` may reference it.

Then pass the filename through, at both call sites in `gdoc/render/__init__.py`:

```python
    meta, _ = frontmatter.parse(text, title_override=title, source_name=Path(md_path).name)
```
```python
    meta, body_markdown = frontmatter.parse(
        markdown_text, title_override=title, source_name=source_path.name
    )
```

- [ ] **Step 4: Run them and watch them pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_render_frontmatter.py -v`
Expected: 7 passed.

- [ ] **Step 5: Write the failing CLI test for the refusal payload**

Add to `tests/test_build_cli.py`:

```python
def test_build_refuses_a_note_with_no_title_and_suggests_one(capsys, tmp_path):
    note = tmp_path / "2026-08-12-miguel-kickoff-call.md"
    note.write_text("---\ngdoc: 1abc\n---\n\n# Miguel kickoff call\n\nBody.\n")
    exit_code = main(["build", "--md", str(note), "-o", str(tmp_path / "o.docx"), "--skip-toc"])
    payload = json.loads(capsys.readouterr().out)
    assert exit_code == 1
    assert payload["missing"] == "title"
    assert payload["suggested_title"] == "Miguel kickoff call"
    assert payload["suggested_from"] == "h1"
    assert not (tmp_path / "o.docx").exists()


def test_the_title_flag_builds_without_touching_the_file(tmp_path):
    note = tmp_path / "2026-08-12-notes.md"
    original = "---\ngdoc: 1abc\n---\n\nBody text.\n"
    note.write_text(original)
    exit_code = main([
        "build", "--md", str(note), "-o", str(tmp_path / "o.docx"),
        "--title", "Kickoff Notes", "--skip-toc",
    ])
    assert exit_code == 0
    assert note.read_text() == original
```

- [ ] **Step 6: Run it and watch it fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_build_cli.py -v`
Expected: FAIL, the payload has only `error`.

- [ ] **Step 7: Handle `MissingTitle` in the CLI**

In `cmd_build`, catch it before the general case:

```python
    except frontmatter.MissingTitle as error:
        return _fail_with({
            "error": str(error),
            "missing": "title",
            "suggested_title": error.candidate,
            "suggested_from": error.source,
        })
```

Add `from gdoc.render import frontmatter` to the imports.

- [ ] **Step 8: Run the tests and watch them pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_build_cli.py -v`
Expected: PASS.

- [ ] **Step 9: Run the whole suite and commit**

```bash
~/.config/gdoc-agent/venv/bin/pytest
git add gdoc/render/frontmatter.py gdoc/cli.py tests/
git commit -m "feat: suggest a title instead of building a blank cover"
```

---

## Task 5: Drop the title heading from the body

**Files:**
- Modify: `gdoc/render/__init__.py`
- Test: `tests/test_render_smoke.py`

**Interfaces:**
- Consumes: `frontmatter.H1_RE`
- Produces: `gdoc.render.drop_title_heading(body_markdown: str, title: str) -> str`

- [ ] **Step 1: Write the failing test**

Add to `tests/test_render_smoke.py`:

```python
def test_a_heading_matching_the_title_is_dropped_from_the_body():
    from gdoc.render import drop_title_heading

    body = "# Kickoff Notes\n\nFirst paragraph.\n"
    assert drop_title_heading(body, "Kickoff Notes").strip() == "First paragraph."


def test_a_heading_that_is_not_the_title_stays():
    from gdoc.render import drop_title_heading

    body = "# Background\n\nFirst paragraph.\n"
    assert drop_title_heading(body, "Kickoff Notes") == body


def test_only_the_first_heading_is_considered():
    from gdoc.render import drop_title_heading

    body = "Intro line.\n\n# Kickoff Notes\n\nMore.\n"
    assert drop_title_heading(body, "Kickoff Notes") == body
```

- [ ] **Step 2: Run them and watch them fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_render_smoke.py -v`
Expected: FAIL, `ImportError: cannot import name 'drop_title_heading'`.

- [ ] **Step 3: Implement it**

In `gdoc/render/__init__.py`:

```python
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
```

and call it in `build`, right after the front matter is parsed:

```python
    body_markdown = drop_title_heading(body_markdown, meta["title"])
    if not body_markdown.strip():
        raise BuildError("the file has front matter but no body content")
```

Note it compares against `meta["title"]`, not `cover_title`, because `cover_title` may have the `doc_type` appended.

- [ ] **Step 4: Run them and watch them pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_render_smoke.py -v`
Expected: PASS.

- [ ] **Step 5: Run the whole suite and commit**

```bash
~/.config/gdoc-agent/venv/bin/pytest
git add gdoc/render/__init__.py tests/test_render_smoke.py
git commit -m "feat: do not print the title twice"
```

---

## Task 6: `generate` renders through the template [AMEND AND EXPAND, DO THIS FIRST]

> **Read the amendment at the top before starting.** This task now also owns the
> two-pass that gets page numbers from Google, and must lose its `skipif soffice`
> gate. `tests/test_contents_integration.py` is the working reference.
>
> Do this task before 2 and 3. Until it lands, `gdoc generate` still runs
> `pandoc md -o docx`, so the skills publish plain documents and none of the merged
> renderer work reaches a real document.

**Files:**
- Modify: `gdoc/generate.py`, `gdoc/cli.py`
- Test: `tests/test_generate.py`, `tests/test_cli.py`

**Interfaces:**
- Consumes: `gdoc.render.build`, `gdoc.render.meta_for`, `gdoc.pairing.read_pairing`
- Produces: `generate(drive, md_path, name, out_path, folder_id, *, template=None) -> Result`, where `template=None` keeps the plain pandoc path

- [ ] **Step 1: Write the failing tests**

Add to `tests/test_generate.py`:

```python
@pytest.mark.slow
@pytest.mark.skipif(not shutil.which("soffice"), reason="LibreOffice not installed")
def test_generate_renders_through_the_template_when_one_is_given(tmp_path):
    md = tmp_path / "in.md"
    md.write_text("---\ntitle: Kickoff Notes\n---\n\nBody.\n")
    drive = MagicMock()
    drive.files().create.return_value.execute.return_value = {"id": "1New", "webViewLink": "x"}
    result = generate(
        drive, md, "Kickoff Notes v1", tmp_path / "v1.docx",
        folder_id="0AFolder", template="altery-group-policy-v1.0",
    )
    assert result.doc_id == "1New"
    # The template's cover text proves the master was used, not bare pandoc.
    assert result.docx_path.stat().st_size > 40_000


def test_template_none_keeps_the_plain_pandoc_path(tmp_path):
    md = tmp_path / "in.md"
    md.write_text("# Title\n\nBody.\n")
    drive = MagicMock()
    drive.files().create.return_value.execute.return_value = {"id": "1New", "webViewLink": "x"}
    result = generate(
        drive, md, "X v1", tmp_path / "v1.docx", folder_id="0AFolder", template=None,
    )
    assert result.doc_id == "1New"
    assert result.docx_path.stat().st_size < 40_000
```

Add `import shutil` at the top of the file. The size assertion is crude on purpose. The master is 58KB before anything is added, and a bare pandoc `.docx` of two lines is a few KB, so the two cannot be confused.

- [ ] **Step 2: Run them and watch them fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_generate.py -v`
Expected: FAIL, `TypeError: generate() got an unexpected keyword argument 'template'`.

- [ ] **Step 3: Implement it**

In `gdoc/generate.py`:

```python
from gdoc import render


def generate(
    drive,
    md_path: Path,
    name: str,
    out_path: Path,
    folder_id: str | None,
    *,
    template: str | None = None,
) -> Result:
    """Convert, then try to upload. Never claim a document that was not created.

    The contents-list refresh is never skipped here. Google Docs regenerates the
    field on import, which is the case normalise_toc_tabs exists to survive, and
    those styles only exist after the LibreOffice pass.
    """
    if not md_path.exists():
        raise FileNotFoundError(f"markdown file not found: {md_path}")
    if template is None:
        docx_path = md_to_docx(md_path, out_path)
    else:
        out_path.parent.mkdir(parents=True, exist_ok=True)
        docx_path = render.build(md_path, out_path, template=template).docx_path
    ...
```

The rest of the function is unchanged.

- [ ] **Step 4: Run them and watch them pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_generate.py -v`
Expected: PASS, or the template case skips on a machine with no LibreOffice.

- [ ] **Step 5: Write the failing CLI test for the optional name**

Add to `tests/test_cli.py`:

```python
def test_generate_names_the_version_when_name_is_omitted(capsys, tmp_path):
    md = tmp_path / "note.md"
    md.write_text("---\ntitle: Kickoff Notes\n---\n\nBody.\n")
    drive = MagicMock()
    drive.files().create.return_value.execute.return_value = {"id": "1New", "webViewLink": "x"}
    with patch("gdoc.cli.drive_service", return_value=drive), \
         patch("gdoc.cli.load_config", return_value=Config(output_folder_id="0AF", template="none")), \
         patch("gdoc.cli.generate") as fake_generate:
        fake_generate.return_value = GenerateResult(docx_path=tmp_path / "v1.docx", doc_id="1New")
        main(["generate", "--md", str(md), "--out", str(tmp_path / "v1.docx")])
    assert fake_generate.call_args.args[2] == "Kickoff Notes v1"
```

with these imports added to the file:

```python
from gdoc.config import Config
from gdoc.generate import Result as GenerateResult
```

- [ ] **Step 6: Run it and watch it fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py -k names_the_version -v`
Expected: FAIL, argparse exits because `--name` is required.

- [ ] **Step 7: Make `--name` optional and pass the template**

In `gdoc/cli.py`, the parser:

```python
    gen.add_argument("--name", help="default: '<cover title> v<n>'")
    gen.add_argument("--template", help="profile name, path, or 'none'")
```

and the handler:

```python
def cmd_generate(args) -> int:
    config = load_config()
    template = args.template or config.template
    master = None if template == profiles.NO_TEMPLATE else template
    md_path = Path(args.md)
    try:
        name = args.name or _version_name(md_path, master)
    except frontmatter.MissingTitle as error:
        return _fail_with({
            "error": str(error),
            "missing": "title",
            "suggested_title": error.candidate,
            "suggested_from": error.source,
        })
    drive = drive_service()
    result = generate(
        drive, md_path, name, Path(args.out),
        folder_id=args.folder_id or config.output_folder_id,
        template=master,
    )
    ...


def _version_name(md_path: Path, template: str | None) -> str:
    """'<cover title> v<n>'. n is the recorded version count plus one.

    With no template there may be no front matter to read, so the file stem
    stands in for the title.
    """
    pairing = read_pairing(md_path)
    number = len(pairing.versions) + 1 if pairing else 1
    if template is None:
        return f"{md_path.stem} v{number}"
    return f"{render.meta_for(md_path)['cover_title']} v{number}"
```

- [ ] **Step 8: Run it and watch it pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py -v`
Expected: PASS.

- [ ] **Step 9: Run the whole suite and commit**

```bash
~/.config/gdoc-agent/venv/bin/pytest
git add gdoc/generate.py gdoc/cli.py tests/
git commit -m "feat: generate through the house template and name the version"
```

---

## Task 7: `generate` records the version

**Files:**
- Modify: `gdoc/cli.py`
- Test: `tests/test_cli.py`

**Interfaces:**
- Consumes: `gdoc.pairing.read_pairing`, `write_pairing`, `add_version`, `Pairing`, `slug_for_source`
- Produces: the `generate` JSON gains `slug` and `version`; the paired file gains `gdoc` and `gdoc_versions`

- [ ] **Step 1: Write the failing tests**

Add to `tests/test_cli.py`:

```python
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
```

with these imports added:

```python
from gdoc.pairing import Pairing, read_pairing, write_pairing
```

- [ ] **Step 2: Run them and watch them fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py -k "pairs_a_file or appends_the_next or records_nothing" -v`
Expected: FAIL. The first two fail because `read_pairing` returns `None`; the last two pass already and must keep passing.

- [ ] **Step 3: Implement the bookkeeping**

In `gdoc/cli.py`, replace the tail of `cmd_generate`:

```python
    payload = {k: (str(v) if isinstance(v, Path) else v) for k, v in asdict(result).items()}
    if args.baseline_root and result.doc_id:
        payload["baseline_path"] = str(_write_baseline_for(drive, args, result.doc_id))
        payload["slug"] = slug_for_source(md_path)
        payload["version"] = _record_version(md_path, result.doc_id)
    return _emit(payload)


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
    created = datetime.date.today().isoformat()
    updated = add_version(replace(pairing, doc_id=doc_id), doc_id, created)
    write_pairing(md_path, updated)
    return len(updated.versions)
```

Add the imports `import datetime`, `from dataclasses import asdict, replace`, and extend the `gdoc.pairing` import with `Pairing` and `write_pairing`.

The baseline is written before the pairing, so a baseline failure leaves nothing recorded rather than a pairing pointing at a snapshot that does not exist.

- [ ] **Step 4: Run them and watch them pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py -v`
Expected: PASS, all four.

- [ ] **Step 5: Run the whole suite and commit**

```bash
~/.config/gdoc-agent/venv/bin/pytest
git add gdoc/cli.py tests/test_cli.py
git commit -m "feat: record the version in generate, where the upload is known to have worked"
```

---

## Task 8: `pair add-version` moves the pointer

**Files:**
- Modify: `gdoc/cli.py`
- Test: `tests/test_cli.py`

**Interfaces:**
- Consumes: `gdoc.pairing.add_version`, `replace`
- Produces: no new names. `cmd_pair_add_version` now also sets `doc_id`

- [ ] **Step 1: Write the failing test**

Add to `tests/test_cli.py`:

```python
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
```

- [ ] **Step 2: Run it and watch it fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py -k add_version_moves -v`
Expected: FAIL, `assert '1Old' == '1Next'`.

- [ ] **Step 3: Implement it**

In `cmd_pair_add_version`, after reading the pairing and before writing:

```python
    # The pure add_version is left alone. Moving the pointer belongs here,
    # so no command can leave gdoc: naming an old version.
    updated = add_version(replace(pairing, doc_id=args.version_id),
                          args.version_id, args.created)
```

- [ ] **Step 4: Run it and watch it pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_cli.py tests/test_pairing.py -v`
Expected: PASS. If an existing test asserted that `add-version` leaves `doc_id` alone, it was asserting the bug. Update it and say so in the commit body.

- [ ] **Step 5: Run the whole suite and commit**

```bash
~/.config/gdoc-agent/venv/bin/pytest
git add gdoc/cli.py tests/test_cli.py
git commit -m "fix: keep gdoc: naming the current version"
```

---

## Task 9: Rewrite the `gdoc-apply` skill

**Files:**
- Modify: `skills/gdoc-apply/SKILL.md`
- Test: none. This is prose. Read it as if you had never seen the tool

**Interfaces:**
- Consumes: everything above.
- Produces: nothing code depends on.

Remember the skills are symlinked into `~/.claude/skills/`, so this edit is live the moment it is saved, before it is committed.

- [ ] **Step 1: Replace the `--terminal-only` section**

```markdown
## If Nail passes `--terminal-only`

Do Steps 1 to 4: work through the items, edit the markdown, show the diffs,
commit if there is a repository, and build the document locally:

```bash
$GDOC build --md <paired md file> --out "$ROOT/.gdoc/<slug>/out/v<n>.docx"
```

`build` never touches Drive. Then stop. Do not run Step 5. Tell Nail where the
`.docx` is, and that re-running without the flag will publish it.
```

- [ ] **Step 2: Replace the end of Step 1**

The current text says to stop when neither lookup answers. Replace that paragraph with:

```markdown
If neither answers, look at what you were given. A source markdown file that
exists, with no pairing, is the first run of the loop: it is v1, not a stranger's
document. Carry on to Step 2.

Stop only when there is nothing to work from: a `pending.md` whose `Source:`
names a file that is not there, and no source markdown given. Then the output is
a note for Nail, not a new document. Say so and stop.
```

- [ ] **Step 3: Add the title step, after Step 2 and before the save**

```markdown
## Step 3: Give the document a title

Every generated document needs a `title` in the paired markdown's frontmatter.
It fills the cover and the running head, and it is what the version is named
after.

If the file has one, use it and say nothing.

If it has none, `$GDOC build` and `$GDOC generate` both refuse, and the refusal
carries a suggestion:

```json
{
  "missing": "title",
  "suggested_title": "Miguel kickoff call",
  "suggested_from": "h1"
}
```

Propose a title to Nail. Use the suggestion when it reads like a document title,
and write a better one from the content when it does not. Show him the single
line you want to add:

```yaml
title: Miguel kickoff call
```

On his approval, add it to the frontmatter of the paired markdown. Do not add
anything else. The title is decided once and reused every run after this.
```

Renumber the steps that follow.

- [ ] **Step 4: Simplify Step 4, the generate step**

```markdown
## Step 5: Generate the new version

```bash
$GDOC generate --md <paired md file> \
  --out "$ROOT/.gdoc/<slug>/out/v<n>.docx" \
  --baseline-root "$ROOT"
```

`--name` is left out. The tool names the version `<cover title> v<n>` and counts
`n` itself, so nothing here has to do arithmetic.

The document is rendered into the house template configured in
`~/.config/gdoc-agent/config.json`. Pass `--template <name|path>` only when Nail
asks for a different one, or `--template none` for a plain document with no cover.

Add `--folder-id <id>` when Nail names a Drive folder. Without it the folder in
the config is used.

`--baseline-root` does two things, both of which matter for the next run. It
writes `.gdoc/<slug>/baseline.md`, the export of the document as published, and
it records the version in the paired markdown's frontmatter. The JSON reports
`baseline_path`, `slug` and `version`.

Use the `slug` from the JSON for the `.gdoc/<slug>/` path in later steps. Do not
invent one.

Two outcomes, both fine:

- `doc_id` and `link` present: the new Google Doc exists. Give Nail the link.
- `doc_id` null and a `reason`: the `.docx` is on disk at `docx_path`. Give him
  that path and the reason. No baseline is written and no version is recorded,
  because there is no document to record. Do not retry silently.
```

- [ ] **Step 5: Delete the old Step 5 and fix the Never list**

Delete the whole `## Step 5: Record the version` section, including its
`pair add-version` call and its commit. `generate` does this now.

In the `## Never` list, replace `Never write a gdoc_versions entry for a document that failed to upload` with:

```markdown
- Never edit the original Google Doc.
- Never edit `baseline.md`.
- Never report a commit that did not happen.
- Never run `$GDOC pair set` to fix an unpaired file. It clears `gdoc_versions`
  when the id differs, which silently drops the history. `generate` pairs the
  file itself.
```

- [ ] **Step 6: Read the whole file top to bottom**

Check the step numbers run 1, 2, 3, 4, 5, 6 with no gaps, that every `<slug>`
now comes from the JSON, and that no sentence says `--name` or `pair add-version`.

- [ ] **Step 7: Commit**

```bash
git add skills/gdoc-apply/SKILL.md
git commit -m "docs: first run, the title step, and generate owning the version record"
```

---

## Task 10: README and CLAUDE.md [PARTLY DONE]

> PR #11 already added the "No external programs" section to `CLAUDE.md` and
> corrected `README.md`. Do not write that LibreOffice is required. What is left is
> whatever Tasks 2 to 9 introduce.

**Files:**
- Modify: `README.md`, `CLAUDE.md`
- Test: none

- [ ] **Step 1: Update the README**

In `## Install`, after the pandoc mention, add the new requirement:

```markdown
Two external tools are needed: `pandoc`, used only as a markdown parser, and
LibreOffice, used only to give the contents list real page numbers.

```bash
brew install pandoc
brew install --cask libreoffice
```

`gdoc read`, `reply`, `capture` and `export` need neither.
```

In the direct CLI list, add `build` and drop `--name` from `generate`:

```bash
gdoc build --md <paired.md>                      # .docx in the house template, no upload
gdoc build --md <paired.md> --pdf
gdoc generate --md <paired.md> --out <out.docx> --baseline-root .
```

Add a section after `## Configure`:

```markdown
## Templates

Documents are rendered into a house template rather than left as plain pandoc
output. A template is a profile: a directory holding `template.docx`. The
bundled ones live in `gdoc/templates/`, and `--template` takes a bundled name, a
path to a profile directory, a path to a `.docx`, or `none` for no template at
all.

The default is a config value, not a constant:

```json
{"output_folder_id": "0AFolderId", "template": "altery-group-policy-v1.0"}
```

`gdoc build` renders locally and never touches Drive. `gdoc generate` renders
through the same code and then uploads.

Every document needs a `title` in its frontmatter, because it fills the cover and
the running head. Without one the build refuses and suggests a title, taken from
the first H1 or from the filename. The skill proposes it and writes it into the
note once you agree, so it is decided once.
```

- [ ] **Step 2: Update CLAUDE.md**

In the "What lives where" table, add two rows:

```markdown
| `gdoc/render/` | the document renderer, ported from the `altery-doc-template` skill. Do not rewrite it: the fidelity lives in measured constants and OOXML child ordering |
| `gdoc/templates/<profile>/` | bundled house templates. A profile is a directory holding `template.docx` |
```

Add a section after "One root, and it is never this repo":

```markdown
## The bundled template is a knowing exception

This repo ships an Altery master template and defaults to it. That sits against
the spirit of the rule above, and it was taken deliberately, on two conditions
that must hold:

- The template is a profile, one of a set, resolved by name. Nothing in the code
  branches on "Altery".
- Which profile is the default is a config value, not a constant.

The rule itself is unchanged: no path names a repository, and nothing is written
into this repo at runtime.

The renderer came from `altery-doc-template` in the Altery hub. That skill is not
this repo's to change or delete.
```

Add to "Never":

```markdown
- Never rewrite `gdoc/render/body.py`, `shell.py` or `ooxml.py` to make them
  tidier. The pixel tests are the only thing standing between a refactor and a
  silent regression, and they need LibreOffice to run at all.
```

In "Testing", add:

```markdown
The renderer tests are marked `slow` and need LibreOffice, pandoc and poppler.
They skip without them, so a green run on a bare machine proves less than it
looks. Run `pytest -m slow` on a full machine before saying the renderer works.
```

- [ ] **Step 3: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: the build command, the template profiles, and the LibreOffice requirement"
```

---

## Final verification

- [ ] **Full suite, including the slow tests**

```bash
~/.config/gdoc-agent/venv/bin/pytest -v
~/.config/gdoc-agent/venv/bin/pytest -m slow -v
```

- [ ] **A real document, end to end, looked at**

```bash
cd /tmp && mkdir -p gdoc-check && cd gdoc-check
cat > 2026-08-14-plan-check.md <<'MD'
---
title: Plan Check
doc_type: Report
classification: Internal
---

## Background

A paragraph to prove the body renders.

- one
- two

| Column | Column |
|---|---|
| a | b |
MD
~/.config/gdoc-agent/venv/bin/gdoc build --md 2026-08-14-plan-check.md --pdf
pdftoppm -png -r 80 -f 1 -l 3 *.pdf page && open page-1.png
```

Cover, then the control tables, then a contents list with real page numbers.

- [ ] **The first-run path, on a copy, never on the real hub note**

```bash
cd /tmp/gdoc-check
cp 2026-08-14-plan-check.md first-run.md
sed -i '' '/^title:/d' first-run.md
~/.config/gdoc-agent/venv/bin/gdoc build --md first-run.md --skip-toc
```

Expected: exit 1, and a payload carrying `suggested_title: "First run"` with
`suggested_from: "filename"`.

- [ ] **Install script still correct**

```bash
./install.sh
```

Expected: it reinstalls the package with the new dependencies and reports both
skills as linked. No new skill was added, so the `SKILLS` array is unchanged.
