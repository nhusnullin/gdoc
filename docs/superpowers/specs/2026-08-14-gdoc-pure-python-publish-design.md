# Taking LibreOffice and poppler out of the publish path

Date: 2026-08-14. Status: implemented.

## Correction, made during implementation

This document was first written and titled as "a publish path with no external
binaries". **That was wrong.** `gdoc/render/body.py:384` runs
`subprocess.run([PANDOC, "-f", PANDOC_FORMAT, "-t", "json"])`: pandoc is the
markdown parser that `build` depends on. The error came from a grep over
`gdoc/*.py`, which does not recurse into `gdoc/render/`. A Task 3 implementer found
it and stopped rather than land a guard test that could never pass.

What this plan actually delivers is narrower and still worth having: **LibreOffice
and poppler both leave**, which is 833MB of programs that cannot be copied into an
isolated environment. **pandoc stays**, so the publish path is not yet pip-only.

Removing pandoc means replacing it with a pure-Python markdown parser and rewriting
body.py's AST walker, which is most of its 579 lines, in the module where the
document body's pixel fidelity lives. That needs its own spec. It is not a step here.

## Why

`gdoc` runs where the source documents live, and that is an isolated environment
with no package manager to hand. Nail's constraint is plain: copying files in is
easy, asking an admin to install software is not. Today the publish path needs two
programs that cannot be copied in usefully:

| Program | Size | Used for |
|---|---|---|
| LibreOffice | 800MB | refreshing the contents field, and `build --pdf` |
| poppler | 33MB | reading page numbers back out of a rendered PDF |

The LibreOffice half is already solved on this branch. `gdoc/render/contents.py`
writes the contents list directly, and
[the findings](2026-08-14-gdoc-toc-libreoffice-findings.md) record the evidence:
Google Docs never refreshes an imported contents field, so computing that cached
result was the only thing LibreOffice was ever needed for, and its resave was also
stripping heading weight, breaking list numbering and mis-numbering the first entry.

What remains is poppler, and the code that still calls LibreOffice.

Measured on the bundled example: `pypdf` returns the same page for all twelve
headings as `pdftotext` does. So poppler is replaceable by a pure-Python wheel.

pandoc remains after that, as the correction above records, so the publish path does
not yet install with pip alone. It goes from three external programs to one, and
from 833MB of them to about 150MB.

A frozen single binary and a Go rewrite were both considered and declined. Freezing
works (a 14MB PyInstaller build produced output byte-identical across all 23 zip
parts) but needs a build machine per platform, which is the cost Nail does not want.

## What changes

### 1. Pagination reads PDFs in Python

`gdoc/render/pagination.py` keeps `resolve` and `drift` exactly as they are. Only
the text source changes: `page_lines` uses `pypdf` instead of shelling out to
`pdftotext`, and `page_count` becomes `len(reader.pages)` instead of parsing
`pdfinfo` output.

`resolve` is deliberately untouched because it holds the two lessons that cost real
debugging: match whole lines rather than substrings, or a heading like `Appendices`
lands on whichever page mentions it in prose; and skip the contents page, because
every heading appears there too.

### 2. LibreOffice leaves the codebase

Delete `gdoc/render/toc.py`, `gdoc/render/scripts/update-toc.sh` and
`gdoc/render/scripts/UpdateToc.bas`. Remove the `refresh_toc` and
`normalise_toc_tabs` calls from `build`, and `normalise_toc_tabs` itself, which
existed only to repair TOC styles LibreOffice invented.

### 3. `build` writes the contents list

`build` currently leaves the field alone and hands off to LibreOffice. It should
call `contents.write` itself, so a build is complete on its own, and take an
optional `pages` mapping to pass through.

### 4. `skip_toc` and `want_pdf` go

`skip_toc` has nothing left to skip. `want_pdf` needed LibreOffice to render, so
local PDF output goes with it. This is a real loss and Nail has accepted it: a PDF
is still available by exporting the published document from Drive, which is more
faithful anyway because it is what the reader actually sees.

## Behaviour after this

| Command | Needs | Produces |
|---|---|---|
| `gdoc build` | nothing external, no network | a complete `.docx`, contents entries and links correct, **page numbers blank** |
| `gdoc build` with `pages` | nothing external | a complete `.docx` including page numbers |
| `gdoc generate` | network | as today, plus pagination from Google (not wired here, see below) |

Blank page numbers offline are honest rather than broken. The entries, hierarchy and
links are all correct, and because the contents list is still a real field, opening
the file in Word or LibreOffice and refreshing fills the numbers in. A wrong number
would be worse than a blank one.

## Out of scope

- **Wiring `generate` to the two-pass.** PR #5's Task 6 owns `generate`. The pieces
  it needs already exist and `tests/test_contents_integration.py` shows the exact
  composition. Doing it here would collide.
- **pandoc, in three separate places.** This is the big one, and the correction at
  the top of this document explains why it is not here:
  - `gdoc/render/body.py:384`, as the **markdown parser** for `build`. Load-bearing.
    Removing it means a pure-Python parser plus a rewritten AST walker, in the module
    that decides how the body looks. Needs its own spec.
  - `gdoc/export.py`, as a markdown fallback when Drive's own `text/markdown` export
    is unavailable. Verified against the live account that Drive supports it, so this
    fallback looks removable on its own.
  - `gdoc/generate.py`, for the plain non-template path, which PR #5 Task 6 retires.
- **A frozen binary, and Go.** Both declined.
- **Heading levels past 3.** They collapse onto `TOC3` today. Unchanged here.
- **PR #5's Task 2 pixel comparisons.** They render locally through LibreOffice, so
  they will need rethinking. The findings document already argues that `AE%` is the
  wrong gate and that a contents-versus-headings assertion is worth more.

## Risks

**pypdf extracts text differently from poppler on some documents.** The bundled
example agrees exactly, but that is one document. Three things contain the risk, and
none of them is new code: `resolve` already falls back to a prefix match for
headings that wrap across lines; a heading it cannot find is left out rather than
guessed at; and `generate`'s `drift` check compares what was written against the
published document, so a disagreement is reported rather than shipped. A test
asserting every heading in the bundled example resolves makes the regression visible.

**pypdf is another dependency.** It is a pure-Python wheel with no compiled
extension, which is the point: it installs from a wheelhouse without a compiler or a
system package.

## Done when

Corrected during implementation. The first two criteria originally overclaimed.

- No module under `gdoc/render/` names `soffice`, `libreoffice`, `pdftotext`,
  `pdfinfo` or `pdftoppm` in code.
- The set of modules under `gdoc/render/` importing `subprocess` is exactly
  `{body.py}`, pinned by a test. An allowlist rather than a ban, because it fails
  both if `subprocess` spreads to another module and if body.py's pandoc dependency
  disappears without the test being updated.
- `gdoc build` works with `soffice` and `pdftotext` absent from `PATH`. It still
  needs pandoc, which is invoked by absolute path, so a stripped `PATH` proves
  nothing about it either way. Say so rather than implying otherwise.
- The suite passes, including a check that every heading in the bundled example
  resolves to a page through the pure-Python reader.
- `pyproject.toml` declares `pypdf`, no longer claims the `slow` marker needs
  LibreOffice, and no longer ships the LibreOffice scripts as package data.
