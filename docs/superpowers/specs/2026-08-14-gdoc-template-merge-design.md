# gdoc template merge and version bookkeeping, design

Date: 2026-08-14
Status: Task 1 built and merged. Tasks 2, 3 and 6 need the amendment below.
Supersedes: [2026-08-14-gdoc-apply-first-run-design.md](2026-08-14-gdoc-apply-first-run-design.md)
Extends: [2026-08-14-gdoc-storage-and-iteration-design.md](2026-08-14-gdoc-storage-and-iteration-design.md)
Amended by: this document's own "Amendment, 2026-08-15" section

## Amendment, 2026-08-15: LibreOffice is gone

**Read this before the rest of the document.** Everything below still describes a
LibreOffice pass. That pass no longer exists, so several sections are now wrong.

This document's own "Deferred, deliberately" section predicted the change:

> Writing those styles ourselves, the way `shell.py` already writes heading
> styles, would leave LibreOffice needed only for `--pdf`.

That is what happened, and then `--pdf` went too. PR #11 merged
`gdoc/render/contents.py`, which writes the contents list directly, and
`gdoc/render/pagination.py`, which gets page numbers from Google. The evidence is
in [2026-08-14-gdoc-toc-libreoffice-findings.md](2026-08-14-gdoc-toc-libreoffice-findings.md)
and the reasoning in
[2026-08-14-gdoc-pure-python-publish-design.md](2026-08-14-gdoc-pure-python-publish-design.md).

The one measurement that drove it: **Google Docs never refreshes an imported
contents field.** Two documents were left untouched and polled for over thirteen
minutes; both kept the master's placeholder contents list. So computing that
cached result was the only thing LibreOffice was ever needed for.

### What this changes, section by section

| Section below | Now |
|---|---|
| "a LibreOffice contents-list refresh" (line 44) | gone, `contents.write` does it |
| `toc.py`, `refresh_toc`, `normalise_toc_tabs` (line 84) | deleted |
| `gdoc build ... [--pdf] [--skip-toc]` (line 123) | both flags gone, see below |
| "those styles only exist after the LibreOffice pass" (line 139) | `contents.py` ships them |
| `tests/test_render_fidelity.py` needing LibreOffice (lines 257-259) | see Task 2 below |
| "the LibreOffice requirement" in docs (line 295) | already written, PR #11 |
| "Dropping the LibreOffice dependency" as deferred (line 324) | **done** |

### The build interface, corrected

```
gdoc.render.build(md_path, out_path=None, *, template=..., title=None, pages=None)
  -> BuildResult(docx_path, title, template, blocks, entries)
```

`want_pdf` and `skip_toc` are gone. `pages` is new: a mapping of heading text to
page number, which the caller supplies. `BuildResult` lost `pdf_path` and gained
`entries`.

**Page numbers are now the caller's problem, and that is deliberate.** They do not
exist until something lays the document out. `generate` gets them from Google by
uploading once and reading the export. A caller with none, such as an offline
build, gets a contents list with correct entries, correct hierarchy, working links
and **blank** page numbers. Any desktop refresh fills them in. A blank is honest;
a wrong number is not.

### No local PDF

`--pdf` needed LibreOffice to render. Export the published document from Drive
instead, which is more faithful anyway, because it is what the reader sees.

### Task 2 needs redesigning, not editing

Task 2's pixel comparisons are built on `build(want_pdf=True)`, then
`soffice --headless --convert-to pdf`, then `pdftoppm`. All three are gone, and its
`_tools_present("soffice", "pandoc", "pdftoppm", "pdftotext", "pdfinfo")` gate would
skip the whole thing today.

The findings document argues the metric was wrong regardless. Measured on the same
document: **22.7% of pixels differing for harmless vertical drift, 3.1% for a
contents page describing a different document.** It ranked the defects backwards, so
a threshold on it would have failed the healthy page and passed the broken one. The
harness also cannot compare a local render against a Google export at all, because
one rasterises to 1241x1754 and the other to 1242x1755, and it refuses on size
mismatch. It could never see the thing it existed to protect.

What replaced it is already merged: assert the contents entries match the document's
real headings, with page numbers inside the page count. No rasterising, no external
program, and it catches the whole class of defect. See
`tests/test_contents_integration.py`.

Task 2 should therefore port the 25 checks that are still meaningful and drop the
two pixel comparisons, or rebuild them against Google's export with a
drift-tolerant comparison. That is a decision to make when Task 2 starts, not now.

### Task 6 grew

It was "`generate` renders through the template". It is now that **plus** the
two-pass that gets page numbers from Google:

1. `build` the document with no pages, so the contents list has blank numbers
2. upload it, export the PDF, read which page each heading landed on
3. `build` again with those pages
4. upload the version that gets published, and check `pagination.drift` is empty

`tests/test_contents_integration.py` is the working reference for all four steps.
Task 6 also still carries a `skipif soffice` gate that must go.

**Task 6 is the task that pays.** Until it lands, `gdoc generate` still runs
`pandoc md -o docx`, so the skills publish plain documents with no house style, and
none of the merged renderer work reaches a real document.

### Unaffected

Tasks 4, 5, 7, 8 and 9 do not touch any of this. Task 10 is partly done: PR #11
added the "No external programs" section to `CLAUDE.md` and corrected `README.md`.

Two changes that turned out to be one. The document renderer moves into this
repo, and `generate` starts recording the version it just created.

## Why they are one change

The storage spec describes step 2 of the loop as "Generate a document from it in
Altery style, using a separate skill". That separate skill is
`altery-doc-template`, which lives in the Altery hub and knows nothing about
pairings. The first live run of `/gdoc-apply` found the consequence: v1 gets
created somewhere that does not record it, and every later step assumes the
record exists.

Fixing the bookkeeping alone would leave that second door open, because the
styling step would still be outside the tool. Moving the renderer in closes it:
after this change the only way to publish a document is `gdoc generate`, which
is already the only place that knows the upload succeeded.

They also collide in the code. Both rewrite `cmd_generate` and both rewrite
`gdoc-apply` Steps 4 and 5. Done in sequence, the second rewrites what the first
just wrote.

Combining them also removes a problem neither could solve alone. The first-run
findings ask where `n` comes from in `--name "<doc name> v<n>"`. Reporting the
version in the JSON does not answer it, because `--name` is an input and the
agent still has to do the arithmetic first. After the merge the tool knows the
cover title from front matter and the pairing knows the version count, so
`--name` becomes optional and the arithmetic disappears.

## What today looks like

`gdoc generate` is `pandoc md -o docx`, then an upload. No cover, no house
style, no contents list. So `/gdoc-apply` publishes a plain Word file and calls
it the new version.

`altery-doc-template` is the renderer that is missing: template surgery, a
Markdown to template-idiomatic OOXML pass, constants measured off the master,
front matter parsing, a LibreOffice contents-list refresh, and 25 checks
including two pixel comparisons against the master.

The overlap between them is only at the ends. Both parse front matter and both
upload. Everything in between is complementary.

## Decisions

| Question | Decision |
|---|---|
| Where the master template lives | In this repo, as a bundled profile, and it is the default |
| How other templates are chosen | `--template NAME\|PATH\|none`, default from config |
| Missing `title` | The CLI refuses and reports a candidate. The skill proposes it, and on approval writes it into the note |
| Which flag gates the pairing write | `--baseline-root`, no new flag |
| Should `pair add-version` move `doc_id` | Yes |
| `gdoc-apply` Step 1 when unpaired | Treat it as the first run and proceed |
| A standalone build skill | Not in this iteration |
| The hub skill | Untouched. Nail retires it when he is ready |

### On shipping an Altery template in a generic tool

CLAUDE.md says no path in this repo may name a specific repository, and a
bundled Altery master is a deliberate exception to the spirit of that rule. It
is taken knowingly, on two conditions. The template is a *profile*, one of a
set, resolved by name, so nothing in the code branches on "Altery". And which
profile is the default is a config value, not a constant.

The rule it must not break stands unchanged: no path names a repository, and
nothing is written into this repo at runtime.

## Layout

```
gdoc/
  render/
    __init__.py       build() is the only entry point callers use
    frontmatter.py    ported, plus the title candidate
    body.py           ported, markdown to template-idiomatic OOXML
    ooxml.py          ported, element helpers and measured constants
    shell.py          ported, template surgery
    toc.py            refresh_toc and normalise_toc_tabs
    profiles.py       resolves a --template value to a profile
    scripts/          update-toc.sh, UpdateToc.bas
  templates/
    altery-group-policy-v1.0/
      template.docx   the master
      example.md      the worked example
      README.md       the front-matter reference
  generate.py         calls render.build() instead of pandoc
```

`scripts/` sits under `render/`, not under a profile, because driving
LibreOffice is template-agnostic.

`pyproject.toml` gains `python-docx` and `lxml`, a `[tool.setuptools]`
package-data entry so the `.docx`, the `.sh` and the `.bas` are installed, and a
`slow` marker alongside the existing `integration` one. The install is editable,
so the files resolve from the working tree either way.

### Profiles

A profile is a directory holding `template.docx`. `--template` accepts a bare
name resolved under `gdoc/templates/`, a path to a profile directory, a path to
a `.docx` used as an ad-hoc master, or `none` for the plain pandoc path.

`~/.config/gdoc-agent/config.json` gains one optional key, `template`,
defaulting to `altery-group-policy-v1.0`.

There is no plugin API for the surgery, and building one now would be
speculative. `shell.py` finds the cover by placeholder text and the tables by
their first-column labels, so a `.docx` that does not follow that contract fails
in `fill_cover`. What is honest today is one renderer, a documented placeholder
contract, and an error naming what it could not find. When a second real
template arrives, the seam is a per-profile surgery module and `profiles.py` is
where it hooks in.

## CLI surface

```bash
gdoc build --md X [--template NAME|PATH|none] [--title TEXT] [-o PATH] [--pdf] [--skip-toc]
gdoc generate --md X --out O [--name N] [--template ...] [--folder-id F] [--baseline-root R]
```

`build` touches Drive not at all: no credential, no network, works offline. Its
default output keeps the convention already in `default_output`,
`YYYY-MM-DD-<title-slug>.docx` beside the source.

`build` never writes front matter, because it never uploads and so has no
document to record.

`generate` keeps every flag it has and adds `--template`. Two rules are fixed
here rather than left to a caller:

- `generate` never skips the contents-list refresh. Google Docs regenerates the
  field on import, which is the case `normalise_toc_tabs` exists to survive, and
  those styles only exist after the LibreOffice pass.
- `generate` grows no `--pdf`. A PDF is `build`'s job.

`--name` becomes optional. Left out, the tool names the version
`<cover title> v<n>`, where `n` is the recorded version count plus one, so an
unpaired note gets v1.

`--title` is on `build` only. A tracked document's title belongs in its note,
which is what the title contract below arranges.

### `--terminal-only` gets a real answer

`gdoc-apply` currently tells Nail there is no local-only way to produce the
document, because `generate` always tries to upload. With `build` there is one,
and it produces a properly templated `.docx` rather than nothing.

## The title contract

The CLI never invents a cover title silently. When `title` is absent it refuses,
and the refusal carries a candidate:

```json
{
  "error": "no title in front matter, so the cover and running head would be blank",
  "missing": "title",
  "suggested_title": "Miguel kickoff call, 2026-08-12",
  "suggested_from": "h1"
}
```

The candidate is deterministic and testable: the first H1 if there is one,
otherwise the filename with a leading `YYYY-MM-DD-` stripped and hyphens turned
to spaces. `suggested_from` is `h1` or `filename`.

The skill turns that into a decision. It reads the note, proposes a title, the
candidate when it reads like a document title and a better one written from the
content when it does not, shows Nail the single line it wants to add, and on
approval writes `title:` into the front matter. The next run finds the field and
proposes nothing, so the title is decided once and then reused.

`write_pairing` already proves the front matter can be edited without disturbing
the rest of the file.

Two supporting rules:

- `--title TEXT` builds without touching the file, for a scratch document that
  should not be stamped.
- When the title equals the first H1, that H1 is dropped from the body, so the
  title does not print on the cover and again above the first paragraph. Keyed
  on equality, so it also helps hand-written front matter.

The build then refuses exactly one thing, down from two: an unrecognised
`classification`, because it shades a fixed row and an unknown value shades
nothing.

## Version bookkeeping

The findings this section answers are recorded in the superseded first-run
proposal. In short: nothing owned the first version, Step 5 failed on a first
run, `pair set` was the wrong workaround from v2 onward, `gdoc:` went stale from
v2 onward, and the queue slug was decided in two places.

After a successful upload, and only when `--baseline-root` is passed,
`cmd_generate` does three things in the same block that writes the baseline:

1. Sets `gdoc` to the new document id.
2. Appends `{id, created}` to `gdoc_versions`, preserving what is there.
3. Reports `slug` and `version` in the JSON alongside `baseline_path`.

`--baseline-root` is the gate because it already means "this is a tracked
version of a tracked file". The baseline and the pairing are the same fact
written in two places, so splitting them across two flags would invent a state
that has no use. A one-off `gdoc generate` without it stamps nothing.

`pair add-version` also moves `doc_id`, so no command can leave `gdoc:` naming
an old version. The pure `add_version` function is unchanged.

This does not breach "the CLI stays dumb". Which folder, which filename and
which version label are policy and stay in the skill. That a document with this
id was created from this markdown on this date is a fact, and the baseline write
already establishes that recording facts at this moment is `generate`'s job.

`pair set` is not changed. Its clearing behaviour is right for what it is,
repointing a file at an unrelated document. The fix is that the skill stops
routing the common case through it.

## Skills

`gdoc-review`: untouched.

`gdoc-apply`: four changes.

- **Step 1** gains the first run. Unpaired, plus a source markdown that exists,
  is v1, not a stranger's document. Refusal is kept for the real outsider case,
  a `pending.md` whose `Source:` names a file that is not there.
- **A new step before generating**, the title contract above.
- **Step 4** no longer computes `n` and no longer invents a slug. `--name` is
  omitted. `--template` is passed only to override the configured default.
- **Step 5 is deleted.** `generate` records the version, so "never record a
  version for a document that failed to upload" becomes a tested code path
  instead of an instruction an agent can skip.

No new skill. Standalone "put this in the house template" is served by the hub
skill until Nail retires it, and a second skill with a near identical
description would only fight it for the trigger.

The hub `altery-doc-template` is not touched by this work, in either of its two
copies, `.claude/skills/` and `.agents/skills/`. It keeps its own `scripts/` and
`assets/` and keeps working exactly as today. Retiring it is Nail's call and a
separate task.

## Tests

The 25 checks in `selftest.py` are the reason the port is safe, so they arrive
with it, as pytest rather than a script:

- `tests/test_render_frontmatter.py`, `test_render_body.py`,
  `test_render_shell.py` take the structural, layout and input checks.
- `tests/test_render_fidelity.py` takes the two pixel comparisons against the
  master. It needs LibreOffice and `pdftoppm`, so it gets a `slow` marker and
  skips when `soffice` is absent, the same shape as the existing `integration`
  marker.
- `pdfdiff.py` becomes `tests/support/pdfdiff.py`. It is a helper, not a test.

New tests, written first:

| Test | Asserts |
|---|---|
| `generate` on an unpaired md | `gdoc` set, `gdoc_versions` has one entry |
| `generate` on a paired md with one version | two entries, the first unchanged, `gdoc` moved |
| `generate` when the upload fails | front matter untouched, no version recorded |
| `generate` without `--baseline-root` | front matter untouched |
| slug reported by `generate` | equals `slug_for_source(md)` |
| `--name` omitted | the document is named `<cover title> v<n>` |
| a note with an H1 and no title | candidate from the H1, `suggested_from: h1` |
| a note with neither | candidate from the filename, date prefix stripped |
| `--title` given | the file is not modified |
| `build` on a paired md | front matter untouched |

The failure-path test is the one that matters. That guarantee lives in a skill
instruction today, where nothing can test it.

## Migration order

Six phases, each green on the full suite before the next.

1. **Move the renderer, no behaviour change.** `gdoc/render/*`, `profiles.py`,
   `gdoc/templates/altery-group-policy-v1.0/`, the dependencies and package
   data, the ported tests, and `gdoc build`. `generate` still uses plain pandoc.
2. **The title contract.** The refusal with a candidate, `--title`, the H1
   de-duplication.
3. **`generate --template`.** The config default, the refresh always on,
   `--name` optional.
4. **Bookkeeping in `cmd_generate`.** The gated pairing write, `slug` and
   `version` in the JSON, `pair add-version` moving `doc_id`.
5. **The skill.** Rewrite `gdoc-apply`.
6. **Docs.** `README.md` and `CLAUDE.md`: the LibreOffice requirement, the
   bundled profile and why it is a knowing exception, and the new `build`
   command.

Phase 1 is the one that can go wrong quietly. Its exit condition is a real
document, built and looked at, not only a green suite.

## What does not move

- `push_to_google_doc` and `--gdoc REMOTE:FOLDER`. rclone `copyto` updates a Doc
  in place; this tool creates a new one per version and records the lineage.
  Two upload models, and only one belongs here.
- `references/google-doc-setup.md`, which is the rclone setup and dies with it.
- The Altery governance prose: the register types, the absence of a "Standard",
  the three approval chains, the hub placement rules. That is judgement about
  Altery documents, not tooling, and it stays in the hub.

## Known constraints, carried over

- The generated `.docx` is around 900KB against the master's 58KB, because
  LibreOffice rewrites the file during the refresh.
- The master's tracked review comments are stripped, so they do not reappear as
  live comments on an issued document.
- Column widths are estimated from content, so wide tables need an eye.
- Verified against LibreOffice. A document going to a committee should be opened
  once in Word first.

## Deferred, deliberately

**Dropping the LibreOffice dependency.** The pass exists because a contents list
is a field, and page numbers do not exist until something lays the document out.
A second reason is subtler: the master defines no TOC styles at all, and
LibreOffice inventing TOC1 to TOC9 during the pass is currently the only reason
those styles exist for `normalise_toc_tabs` to correct. Writing those styles
ourselves, the way `shell.py` already writes heading styles, would leave
LibreOffice needed only for `--pdf`. It is real work and a new fidelity risk, so
it does not ride along with the port.

**Splitting the two large modules.** `body.py` is 578 lines and `shell.py` 625,
both above the 200 to 400 the style rules call typical. They move as they are.
Splitting them in the same change as the port would make the pixel tests the
only thing standing between a refactor and a silent regression.

**A `gdoc slug` subcommand.** `generate` reports the slug, so nothing needs to
ask for it separately.
