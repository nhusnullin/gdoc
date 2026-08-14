# Pure-Python Publish Path Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove every external program from the publish path, so `gdoc build`
produces a complete document with nothing but Python installed.

**Architecture:** `pypdf` replaces poppler as the page-text reader, keeping
`pagination.resolve` untouched because it holds two hard-won matching rules.
LibreOffice leaves the codebase entirely: `build` now writes the contents list
itself through `gdoc.render.contents`, which already produces a correct, refreshable,
clickable contents list.

**Tech Stack:** Python 3.11+, python-docx, lxml, pypdf, pytest.

**Spec:** `docs/superpowers/specs/2026-08-14-gdoc-pure-python-publish-design.md`

**Background evidence:** `docs/superpowers/specs/2026-08-14-gdoc-toc-libreoffice-findings.md`

## Global Constraints

- `requires-python = ">=3.11"`. Do not lower it.
- **No module under `gdoc/render/` may call an external program.** After this plan,
  `grep -rn "subprocess" gdoc/render/` must return nothing. `gdoc/export.py` keeps
  its pandoc fallback: that is out of scope, stated in the spec, and the guard test
  is scoped to match. Do not widen the guard to `gdoc/` in this plan; it would fail.
- `pypdf` must stay a plain runtime dependency. It is pure Python with no compiled
  extension, which is why it can be installed from a wheelhouse in an isolated
  environment.
- Run the suite with `~/.config/gdoc-agent/venv/bin/pytest`.
- Writing style for docstrings and commits: plain short English, no em dashes.
- `gdoc.render.pagination.resolve` and `.drift` must not change behaviour. Only the
  text source changes.

---

### Task 1: Read PDFs in Python instead of shelling out to poppler

**Files:**
- Modify: `pyproject.toml:10-16` (add the dependency), `pyproject.toml:32-36` (marker text)
- Modify: `gdoc/render/pagination.py:1-30` (docstring, imports, constants), `:77-100` (`page_count`, `page_lines`)
- Test: `tests/test_render_pagination.py` (add a helper and four checks)

**Interfaces:**
- Consumes: `pagination.resolve(page_lines, headings)` and `pagination.drift`, both unchanged.
- Produces: `pagination.page_count(pdf) -> int`, `pagination.page_lines(pdf) -> list[list[str]]`,
  `pagination.from_pdf(pdf, headings) -> dict[str, int]`, and `pagination.PaginationError`.
  Same names and signatures as today, so `from_pdf` callers need no change.

- [ ] **Step 1: Add the dependency**

In `pyproject.toml`, the `dependencies` list becomes:

```toml
dependencies = [
    "google-auth",
    "google-api-python-client",
    "PyYAML",
    "python-docx>=1.1",
    "lxml",
    "pypdf>=5",
]
```

Then install it:

```bash
~/.config/gdoc-agent/venv/bin/pip install -q -e ".[dev]"
~/.config/gdoc-agent/venv/bin/python -c "import pypdf; print(pypdf.__version__)"
```

Expected: a version number, 5 or higher.

- [ ] **Step 2: Write the failing test**

Add to the top of `tests/test_render_pagination.py`, after the existing imports:

```python
from io import BytesIO


def minimal_pdf(pages):
    """Build a small PDF with one text line per entry, per page.

    Hand-built so the suite needs no binary fixture and no PDF writer. Standard
    Helvetica, uncompressed streams, which is all pypdf needs to extract text.
    """
    objects = [None, None, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"]
    font_id, page_ids = 3, []

    for lines in pages:
        stream_lines = ["BT", "/F1 12 Tf", "1 0 0 1 50 750 Tm", "14 TL"]
        for line in lines:
            escaped = line.replace("\\", r"\\").replace("(", r"\(").replace(")", r"\)")
            stream_lines += [f"({escaped}) Tj", "T*"]
        stream_lines.append("ET")
        stream = "\n".join(stream_lines)
        content_id = len(objects) + 1
        objects.append(f"<< /Length {len(stream)} >>\nstream\n{stream}\nendstream")
        page_ids.append(len(objects) + 1)
        objects.append(
            f"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] "
            f"/Resources << /Font << /F1 {font_id} 0 R >> >> "
            f"/Contents {content_id} 0 R >>"
        )

    kids = " ".join(f"{pid} 0 R" for pid in page_ids)
    objects[0] = "<< /Type /Catalog /Pages 2 0 R >>"
    objects[1] = f"<< /Type /Pages /Count {len(page_ids)} /Kids [{kids}] >>"

    out = BytesIO()
    out.write(b"%PDF-1.4\n")
    offsets = []
    for number, body in enumerate(objects, start=1):
        offsets.append(out.tell())
        out.write(f"{number} 0 obj\n{body}\nendobj\n".encode("latin-1"))
    xref_at = out.tell()
    out.write(f"xref\n0 {len(objects) + 1}\n".encode())
    out.write(b"0000000000 65535 f \n")
    for offset in offsets:
        out.write(f"{offset:010d} 00000 n \n".encode())
    out.write(f"trailer\n<< /Size {len(objects) + 1} /Root 1 0 R >>\n"
              f"startxref\n{xref_at}\n%%EOF\n".encode())
    return out.getvalue()


@pytest.fixture
def sample_pdf(tmp_path):
    """Three pages: a contents list, then a body page, then an appendix page.

    Page 2 mentions "Appendices" in prose and page 3 carries it as a heading. That
    is the case that made a substring match put two entries on the wrong page.
    """
    path = tmp_path / "sample.pdf"
    path.write_bytes(minimal_pdf([
        ["Contents", "1-Purpose 4", "Appendices 5"],
        ["1-Purpose", "Body text mentioning Appendices in prose"],
        ["Appendices", "Appendix 1 - Associated Documents"],
    ]))
    return path


def test_page_count_reads_the_pdf(sample_pdf):
    assert pagination.page_count(sample_pdf) == 3


def test_page_lines_returns_the_text_of_each_page(sample_pdf):
    lines = pagination.page_lines(sample_pdf)
    assert len(lines) == 3
    assert "Contents" in lines[0]
    assert "1-Purpose" in lines[1]


def test_from_pdf_resolves_headings_against_a_real_pdf(sample_pdf):
    found = pagination.from_pdf(sample_pdf, ((1, "1-Purpose"), (1, "Appendices")))
    assert found == {"1-Purpose": 2, "Appendices": 3}


def test_no_external_program_is_used(sample_pdf, monkeypatch):
    """Reading a PDF must not shell out. That is the whole point of the change."""
    import subprocess

    def refuse(*args, **kwargs):
        raise AssertionError("pagination shelled out to an external program")

    monkeypatch.setattr(subprocess, "run", refuse)
    monkeypatch.setattr(subprocess, "Popen", refuse)
    assert pagination.page_count(sample_pdf) == 3
    assert len(pagination.page_lines(sample_pdf)) == 3


def test_an_unreadable_file_is_reported_clearly(tmp_path):
    broken = tmp_path / "broken.pdf"
    broken.write_bytes(b"this is not a pdf")
    with pytest.raises(pagination.PaginationError):
        pagination.page_count(broken)
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_render_pagination.py -v`

Expected: the five new checks fail. `test_no_external_program_is_used` fails with
the `AssertionError` from `refuse`, because `page_count` still calls `pdfinfo`.

- [ ] **Step 4: Replace the poppler calls**

In `gdoc/render/pagination.py`, replace the module docstring's last paragraph and
the two functions. Delete `import subprocess` and the `PDFTOTEXT` / `PDFINFO`
constants, and add `from pypdf import PdfReader`.

The docstring paragraph that currently reads:

```
`resolve` is kept free of subprocesses so it can be tested against page text
directly. `from_pdf` is the thin shell around poppler.
```

becomes:

```
`resolve` takes page text directly, so it can be tested without a PDF at all.
`from_pdf` is the thin wrapper that reads one. Reading is done in Python rather
than by shelling out, because the tool has to run where no external program can
be installed.
```

The two functions become:

```python
def page_count(pdf: Path) -> int:
    return len(_reader(pdf).pages)


def page_lines(pdf: Path) -> list:
    """The text of each page, as a list of lines."""
    out = []
    for number, page in enumerate(_reader(pdf).pages, start=1):
        try:
            text = page.extract_text() or ""
        except Exception as error:  # noqa: BLE001 - one bad page must name itself
            raise PaginationError(f"could not read page {number} of {pdf}: {error}") from error
        out.append(text.splitlines())
    return out


def _reader(pdf: Path) -> "PdfReader":
    pdf = Path(pdf)
    if not pdf.is_file():
        raise PaginationError(f"no such file: {pdf}")
    try:
        return PdfReader(str(pdf))
    except Exception as error:  # noqa: BLE001 - pypdf raises several unrelated types
        raise PaginationError(f"could not read {pdf} as a PDF: {error}") from error
```

Note `_reader` is called twice when `from_pdf` runs, once by each function. That is
a cheap reparse of a file we just wrote, and it keeps both functions usable alone.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_render_pagination.py -v`

Expected: all checks pass, the nine existing ones included.

- [ ] **Step 6: Confirm the live path still agrees**

Run: `GDOC_LIVE_PUBLISH_TEST=1 ~/.config/gdoc-agent/venv/bin/pytest -m integration tests/test_contents_integration.py -v`

Expected: PASS. This publishes twice and asserts no drift, so it proves pypdf
resolves the real document the same way poppler did. Skip this step if there are no
credentials, and say so out loud rather than assuming it passed.

- [ ] **Step 7: Update the pytest marker**

In `pyproject.toml`, the `slow` marker no longer needs poppler:

```toml
markers = [
    "integration: calls the live Drive API, skipped without credentials",
    "slow: renders a real document",
]
```

- [ ] **Step 8: Commit**

```bash
git add pyproject.toml gdoc/render/pagination.py tests/test_render_pagination.py
git commit -m "feat: read PDFs with pypdf instead of poppler

Page numbers come from Google's own render, which means reading a PDF back.
Doing that with pdftotext put a 33MB external program on the publish path,
and the tool has to run where nothing can be installed.

resolve() is untouched. It holds two rules that cost real debugging: match
whole lines rather than substrings, or a heading like Appendices lands on
whichever page mentions it in prose, and skip the contents page, because
every heading appears there too.

The new tests build a 1.3KB PDF by hand, so the suite needs no binary
fixture, and one of them refuses subprocess entirely."
```

---

### Task 2: `build` writes its own contents list

**Files:**
- Modify: `gdoc/render/contents.py` (add `rewrite_in_place`, add `import os`)
- Modify: `gdoc/render/__init__.py:1-9` (docstring), `:22` (import), `:29-35` (`BuildResult`), `:62-121` (`build`)
- Test: `tests/test_render_contents.py` (two checks), `tests/test_render_smoke.py` (rewrite)
- Modify: `tests/test_contents_integration.py:64`

**Interfaces:**
- Consumes: `contents.write(src, dst, pages=None) -> ContentsResult` from the existing module.
- Produces:
  - `contents.rewrite_in_place(path, pages=None) -> ContentsResult`
  - `build(md_path, out_path=None, *, template=..., title=None, pages=None) -> BuildResult`
  - `BuildResult(docx_path: Path, title: str, template: str, blocks: int, entries: int)`.
    `pdf_path` and the `want_pdf` and `skip_toc` parameters are **gone**.

- [ ] **Step 1: Write the failing test for in-place rewriting**

Add to `tests/test_render_contents.py`:

```python
def test_rewriting_in_place_leaves_one_file(built, tmp_path):
    """A zip cannot be read and rewritten at the same path at once."""
    target = tmp_path / "inplace.docx"
    target.write_bytes(built.read_bytes())
    result = contents.rewrite_in_place(target, pages={"1-Purpose": 4})
    assert list(tmp_path.iterdir()) == [target], "a temporary file was left behind"
    assert len(result.entries) == 12
    assert "<w:t>4</w:t>" in document_xml(target)


def test_rewriting_in_place_cleans_up_after_a_failure(tmp_path):
    """The temporary file must not survive a failed rewrite."""
    junk = tmp_path / "junk.docx"
    junk.write_bytes(b"not a docx")
    with pytest.raises(zipfile.BadZipFile):
        contents.rewrite_in_place(junk)
    assert list(tmp_path.iterdir()) == [junk], "a temporary file was left behind"
```

- [ ] **Step 2: Run to verify it fails**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_render_contents.py -k in_place -v`

Expected: FAIL with `AttributeError: module 'gdoc.render.contents' has no attribute 'rewrite_in_place'`.

- [ ] **Step 3: Add `rewrite_in_place`**

At the top of `gdoc/render/contents.py`, add `import os` beside the existing
`import re`. At the end of the module, add:

```python
def rewrite_in_place(path: Path, pages: Mapping | None = None) -> ContentsResult:
    """Replace a document's contents list, leaving one file behind.

    Goes through a sibling temporary file, because a zip cannot be read and
    rewritten at the same path at once. The temporary file is removed whether or
    not the rewrite succeeds.
    """
    path = Path(path)
    temporary = path.with_name(path.name + ".contents-tmp")
    try:
        result = write(path, temporary, pages=pages)
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)
    return result
```

- [ ] **Step 4: Run to verify it passes**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_render_contents.py -v`

Expected: all pass.

- [ ] **Step 5: Rewrite the build smoke tests**

Replace the whole of `tests/test_render_smoke.py` with:

```python
import pytest

from gdoc.render import BuildError, build
from gdoc.render.profiles import TEMPLATES_DIR

pytestmark = pytest.mark.slow

EXAMPLE = TEMPLATES_DIR / "altery-group-policy-v1.0" / "example.md"


def test_the_bundled_example_builds_into_a_real_docx(tmp_path):
    """No skipif. A build needs no external program now, which is the point."""
    result = build(EXAMPLE, tmp_path / "out.docx")
    assert result.docx_path.read_bytes()[:2] == b"PK"
    assert result.title
    assert result.blocks > 0
    assert result.entries > 0


def test_the_build_writes_its_own_contents_list(tmp_path):
    result = build(EXAMPLE, tmp_path / "out.docx")
    assert result.entries == 12


def test_page_numbers_are_written_when_the_caller_knows_them(tmp_path):
    import zipfile

    out = tmp_path / "out.docx"
    build(EXAMPLE, out, pages={"1-Purpose": 4})
    xml = zipfile.ZipFile(out).read("word/document.xml").decode("utf8")
    assert "<w:t>4</w:t>" in xml


def test_a_missing_source_is_reported_before_anything_else(tmp_path):
    with pytest.raises(BuildError):
        build(tmp_path / "nope.md", tmp_path / "out.docx")
```

- [ ] **Step 6: Run to verify they fail**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_render_smoke.py -v`

Expected: FAIL. `build` still calls LibreOffice by default, and `BuildResult` has no
`entries` field, and `pages` is not a parameter.

- [ ] **Step 7: Change `build`**

In `gdoc/render/__init__.py`:

Replace the module docstring's first paragraph:

```python
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
```

Replace line 22:

```python
from gdoc.render import body, contents, frontmatter, profiles, shell
```

(and delete `from gdoc.render.toc import TocError, refresh_toc`)

Replace `BuildResult`:

```python
@dataclass(frozen=True)
class BuildResult:
    docx_path: Path
    title: str
    template: str
    blocks: int
    entries: int
```

Replace the signature and the tail of `build`. The signature becomes:

```python
def build(
    md_path: Path,
    out_path: Path | None = None,
    *,
    template: str = profiles.DEFAULT_TEMPLATE,
    title: str | None = None,
    pages: Mapping | None = None,
) -> BuildResult:
```

Add `from typing import Mapping` to the imports. Then replace everything from
`pdf_path = ...` (line 101) to the end of the function with:

```python
    written = contents.rewrite_in_place(output_path, pages=pages)

    return BuildResult(
        docx_path=output_path,
        title=meta["cover_title"],
        template=template,
        blocks=blocks,
        entries=len(written.entries),
    )
```

- [ ] **Step 8: Run to verify they pass**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_render_smoke.py -v`

Expected: all four pass.

- [ ] **Step 9: Drop `skip_toc` from the other callers**

`tests/test_render_contents.py:30`, inside the `built` fixture:

```python
    build(EXAMPLE, out)
```

`tests/test_contents_integration.py:64`:

```python
    build(EXAMPLE, source)
```

The integration test then calls `contents.write(source, first)` on a document whose
contents list `build` has already written, so `field()` must still find the field.
It does, because `write` keeps the field. No further change is needed there.

- [ ] **Step 10: Run the whole suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest -v`

Expected: everything passes, one skip for the opt-in live publish test.

- [ ] **Step 11: Commit**

```bash
git add gdoc/render/contents.py gdoc/render/__init__.py tests/test_render_contents.py \
        tests/test_render_smoke.py tests/test_contents_integration.py
git commit -m "feat: build writes its own contents list

build handed the contents list to LibreOffice. It now writes it directly, so a
build is complete on its own and needs no external program.

want_pdf and skip_toc are gone. A local PDF needed LibreOffice to render, and
skip_toc has nothing left to skip. A PDF is still available by exporting the
published document from Drive, which is more faithful anyway because it is what
the reader sees.

Page numbers are the caller's to supply. An offline build gets correct entries
and blank numbers, which any desktop refresh fills in. A wrong number would be
worse than a blank one."
```

---

### Task 3: Delete LibreOffice from the codebase

**Files:**
- Delete: `gdoc/render/toc.py`, `gdoc/render/scripts/update-toc.sh`, `gdoc/render/scripts/UpdateToc.bas`
- Modify: `gdoc/render/shell.py:481-...` (delete `normalise_toc_tabs`)
- Modify: `pyproject.toml:26-28` (package data no longer ships scripts)
- Test: `tests/test_no_external_programs.py` (create)

**Interfaces:**
- Consumes: nothing new.
- Produces: nothing new. This task only removes code, and adds one guard test.

- [ ] **Step 1: Write the guard test**

Create `tests/test_no_external_programs.py`:

```python
"""The publish path must not call out to anything.

gdoc runs where the source documents live, and that is an environment with no
package manager. A dependency that cannot be pip installed cannot be installed at
all, so this is a rule rather than a preference.

Scoped to gdoc/render/, the publish path. gdoc/export.py still shells out to pandoc
as a markdown fallback, and gdoc/generate.py has a plain pandoc path. Both are out
of scope here: Drive exports text/markdown natively, so the fallback is removable,
and PR #5's Task 6 retires the plain path. Widen this guard when they go.
"""

import ast
import re
from pathlib import Path

import pytest

RENDER = Path(__file__).resolve().parent.parent / "gdoc" / "render"
PROGRAMS = ("soffice", "libreoffice", "pdftotext", "pdfinfo", "pdftoppm", "pandoc")


def python_files():
    return sorted(RENDER.rglob("*.py"))


def code_only(text):
    """Source with docstrings and comments stripped.

    Naming a program while explaining why it is gone is fine, and several modules
    do exactly that. Only executable code counts.
    """
    without_docstrings = re.sub(r'("""|\'\'\')(?:.|\n)*?\1', '""', text)
    return "\n".join(
        line for line in without_docstrings.splitlines()
        if not line.strip().startswith("#")
    )


def test_there_are_python_files_to_check():
    """Guards the guard. An empty glob would make every check below vacuous."""
    assert len(python_files()) > 3


def test_the_stripper_keeps_code_and_drops_prose():
    """Guards the guard again. A stripper that ate everything would pass silently."""
    sample = '"""a docstring naming soffice"""\n# a comment naming pandoc\nx = "kept"\n'
    stripped = code_only(sample)
    assert "soffice" not in stripped
    assert "pandoc" not in stripped
    assert "kept" in stripped


def test_no_render_module_imports_subprocess():
    offenders = []
    for path in python_files():
        tree = ast.parse(path.read_text(encoding="utf-8"))
        for node in ast.walk(tree):
            if isinstance(node, ast.Import):
                if any(alias.name.split(".")[0] == "subprocess" for alias in node.names):
                    offenders.append(path.name)
            elif isinstance(node, ast.ImportFrom):
                if (node.module or "").split(".")[0] == "subprocess":
                    offenders.append(path.name)
    assert not offenders, f"subprocess is imported by {sorted(set(offenders))}"


@pytest.mark.parametrize("program", PROGRAMS)
def test_no_render_module_names_an_external_program_in_code(program):
    offenders = [
        path.name for path in python_files()
        if program in code_only(path.read_text(encoding="utf-8"))
    ]
    assert not offenders, f"{program} appears in the code of {offenders}"


def test_no_shell_scripts_ship_inside_the_package():
    package = RENDER.parent
    scripts = list(package.rglob("*.sh")) + list(package.rglob("*.bas"))
    assert not scripts, f"external scripts still shipped: {scripts}"
```

- [ ] **Step 2: Run to verify it fails**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_no_external_programs.py -v`

Expected: FAIL. `gdoc/render/toc.py` still imports `subprocess` and names `soffice`,
and the two script files still exist.

- [ ] **Step 3: Delete the LibreOffice code**

```bash
git rm gdoc/render/toc.py gdoc/render/scripts/update-toc.sh gdoc/render/scripts/UpdateToc.bas
```

If `gdoc/render/scripts/` is then empty, it goes too.

- [ ] **Step 4: Delete `normalise_toc_tabs`**

In `gdoc/render/shell.py`, delete the whole `normalise_toc_tabs` function starting at
line 481, including its docstring. It existed only to repair the `TOC1..TOC9` styles
LibreOffice invented during a refresh, and `gdoc/render/contents.py` now ships those
styles with correct tab stops. Nothing calls it: confirm with

```bash
grep -rn "normalise_toc_tabs" gdoc tests
```

Expected: no output.

- [ ] **Step 5: Stop shipping the scripts as package data**

In `pyproject.toml`, the package data section becomes:

```toml
[tool.setuptools.package-data]
"gdoc" = ["templates/*/*.docx", "templates/*/*.md"]
```

Delete the `"gdoc.render" = ["scripts/*.sh", "scripts/*.bas"]` line.

- [ ] **Step 6: Run to verify it passes**

Run: `~/.config/gdoc-agent/venv/bin/pytest tests/test_no_external_programs.py -v`

Expected: PASS.

- [ ] **Step 7: Run the whole suite**

Run: `~/.config/gdoc-agent/venv/bin/pytest -v`

Expected: everything passes, one skip. If an import of `gdoc.render.toc` remains
anywhere, it fails loudly here.

- [ ] **Step 8: Prove it by hand, with a stripped PATH**

```bash
cd /tmp && env PATH=/usr/bin:/bin ~/.config/gdoc-agent/venv/bin/python -c "
from gdoc.render import build
from gdoc.render.profiles import TEMPLATES_DIR
out = build(TEMPLATES_DIR / 'altery-group-policy-v1.0' / 'example.md', '/tmp/proof.docx')
print('built', out.docx_path, out.entries, 'entries')
"
```

Expected: `built /tmp/proof.docx 12 entries`. Confirm `soffice`, `pdftotext` and
`pandoc` are all absent from that `PATH` first with `command -v`.

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "chore: delete LibreOffice from the codebase

toc.py, update-toc.sh and UpdateToc.bas existed to refresh the contents field.
Nothing refreshes it now, because gdoc/render/contents.py writes it directly.
normalise_toc_tabs goes with them: it repaired TOC styles LibreOffice invented,
and contents.py ships those styles with correct tab stops instead.

A new guard test fails if any module under gdoc/ names an external program or
imports subprocess, because the tool has to run where nothing can be installed."
```

---

### Task 4: Say so in the documentation

**Files:**
- Modify: `README.md` (the requirements section, and the pandoc sentence at line 119)
- Modify: `CLAUDE.md` (a line under Testing)
- Modify: `docs/superpowers/specs/2026-08-14-gdoc-pure-python-publish-design.md` (status line)

**Interfaces:**
- Consumes: nothing. Documentation only.
- Produces: nothing.

- [ ] **Step 1: State the dependency rule in CLAUDE.md**

Add this section to `CLAUDE.md`, after "The tool must work without git":

```markdown
## No external programs

`gdoc` runs where the source documents live, and that environment has no package
manager. Anything that cannot be pip installed cannot be installed at all.

So no module under `gdoc/render/`, the publish path, may call an external program.
`tests/test_no_external_programs.py` fails if one does. LibreOffice and poppler were
removed for this reason.

`gdoc/export.py` still shells out to pandoc as a markdown fallback. Drive exports
`text/markdown` natively, verified against the live account, so that fallback is
removable and the guard should widen when it goes.

Page numbers are the one thing this costs. They do not exist until something lays
the document out, so `gdoc generate` gets them from Google: it uploads once, reads
which page each heading landed on out of the PDF export, writes them in, and
publishes. An offline `gdoc build` leaves them blank, which any desktop refresh
fills in. A wrong number would be worse than a blank one.
```

- [ ] **Step 2: Correct the pandoc sentence in README.md**

`README.md:119` currently says both sides of the baseline diff "carry the same
pandoc round-trip distortion, so it cancels". That is still true of the export side
today, so leave the sentence but add after it:

```markdown
Generation no longer goes through pandoc: `gdoc build` writes the .docx directly
from the house template. The export side still uses it as a fallback, which is
tracked separately.
```

- [ ] **Step 3: Mark the spec as done**

In the spec, change the status line to:

```markdown
Date: 2026-08-14. Status: implemented.
```

- [ ] **Step 4: Verify the claims in the docs are true**

```bash
grep -rn "subprocess" gdoc/render/ || echo "no subprocess in the publish path: correct"
grep -rn "subprocess" gdoc/ | grep -v render/ || true
~/.config/gdoc-agent/venv/bin/pytest -q
```

Expected: the first grep prints the "correct" line. The second **will** list
`gdoc/export.py`, which still uses pandoc and is out of scope. That is the honest
state, so the CLAUDE.md wording in step 1 says "no module under `gdoc/render/`", not
"no module under `gdoc/`". Fix the wording if it drifted.

- [ ] **Step 5: Commit**

```bash
git add README.md CLAUDE.md docs/superpowers/specs/2026-08-14-gdoc-pure-python-publish-design.md
git commit -m "docs: record the no-external-programs rule"
```

---

## Self-Review

**Spec coverage:**

| Spec section | Task |
|---|---|
| 1. Pagination reads PDFs in Python | Task 1 |
| 2. LibreOffice leaves the codebase | Task 3 |
| 3. `build` writes the contents list | Task 2 |
| 4. `skip_toc` and `want_pdf` go | Task 2 |
| Behaviour table | Task 2 steps 5 and 7, Task 3 step 8 |
| Risk: pypdf extracts differently | Task 1 steps 2 and 6 |
| Done when: no soffice/pdftotext/pdfinfo/pdftoppm | Task 3 step 1 |
| Done when: build works with nothing on PATH | Task 3 step 8 |
| Done when: pyproject declares pypdf, marker fixed | Task 1 steps 1 and 7 |

Out-of-scope items in the spec (wiring `generate`, pandoc, a frozen binary, Go,
`TOC4+`, Task 2 pixel comparisons) have no task, which is correct.

**Type consistency:** `BuildResult.entries` is an `int` in Task 2's dataclass and is
read as an `int` in the smoke tests. `contents.rewrite_in_place` returns
`ContentsResult`, whose `.entries` is the tuple of `(level, text)` pairs, so `build`
uses `len(written.entries)` to fill its own `int` field. Those two `entries` names
mean different things one call apart, which is a trap; the `len()` in Task 2 step 7
is deliberate and correct.

**Placeholders:** none. Every code step carries the code, and the hand-built PDF
helper in Task 1 was run before being written down: a 1,296 byte file from which
pypdf extracted all three pages correctly.

**Two bugs found and fixed during this review:**

1. The first draft of Task 3's guard test scanned all of `gdoc/` for banned words in
   raw source. It would have failed on `contents.py`, whose docstring explains why
   LibreOffice is gone, and on `export.py`, which still uses pandoc and is explicitly
   out of scope. The guard is now scoped to `gdoc/render/`, strips docstrings and
   comments before matching, and uses `ast` for the import check. Two extra checks
   guard the guard: one that the file glob is not empty, one that the stripper keeps
   code while dropping prose. A guard that silently passes is worse than none.
2. The Global Constraints and Task 4 verification both claimed `grep subprocess gdoc/`
   would come back empty. It will not, because of `export.py`. Both now say
   `gdoc/render/` and name the exception.

## One judgement call for the reviewer

Task 1 step 4 parses the PDF twice when `from_pdf` runs, once in `page_count` and
once in `page_lines`. Caching a reader would avoid it. It is not worth the state for
a file we wrote seconds earlier, and both functions stay usable alone. Reject this
if you disagree; it is a two-line change either way.
