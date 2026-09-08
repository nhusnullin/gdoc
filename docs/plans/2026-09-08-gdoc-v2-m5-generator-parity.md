# gdoc v2 Milestone 5: Generator parity

## Overview

The Go binary learns to turn a hub note into an Altery house-style `.docx`, and to prove the result matches what the Word master produced. `gdoc build --md note.md --out file.docx` writes the file and touches no network. Publishing it into Drive is M6.

The house style is `house.yaml`, not the master `.docx`: Nail's decision of 2026-08-29, recorded in DECISIONS.md. The file describes page geometry, nine named styles, the cover, the header and footer with the positioned logo, the three front-matter tables cell by cell, the legend, the live contents field, heading numbering, and the logo as base64. It is embedded in the binary, so installing gdoc is still one file, and `--house <path>` points at another copy for a change under review. Nothing reads the master at runtime. The master stays in the repo as provenance and as the drift test's fixture.

Two spikes exist and this milestone joins them. `docs/v2/spikes/config/gen.py` renders `house.yaml` plus markdown into a docx from scratch; its shell is complete and its markdown support is thin. `spike/render/internal/body` is the goldmark walker from the language-decision spike: nested lists, bold, italic, `==mark==`, links, images in headings, heading numbering, and the v1 table recipe. M5 ports gen.py's shell and the spike's body walker into `go/internal/`, adds pipe tables to the body, and ports `compare.py` twice: as an offline test on the docx XML that runs on every commit, and as an opt-in live test through the Docs API, which is the measurement that means something because Google's import is part of the result.

Decisions Nail took on 2026-09-08, before this plan: the drift gate runs both ways, offline in `make test` and live on request; the body must render markdown tables, and code blocks stay out; the cover fields keep v1's front-matter names so existing notes publish without edits; `house.yaml` is embedded in the binary with an override flag; a `build` command lands now so M5 is testable before M6; the contents list is a Word TOC field, which imports as a live list, so one upload and no measuring pass.

Spec: `docs/v2/SPEC.md` ("The binary", "The generator and `house.yaml`"). Master plan: `docs/v2/PLAN.md`, section M5. Decision record: `docs/v2/DECISIONS.md`, "2026-08-29. The house style is a config file. The .docx becomes provenance." Spike report: `docs/superpowers/specs/2026-08-29-go-render-spike.md`. Predecessor: `docs/plans/completed/2026-09-07-gdoc-v2-m4-live-session.md`.

## Context (from discovery)

- Files and components involved: new packages `go/internal/house` (the config), `go/internal/render` (the docx writer: shell, cover, tables, header and footer, TOC, body assembly), `go/internal/body` (the goldmark walker), `go/internal/cover` (the note's author-side front matter), `go/internal/drift` (the offline comparison); modified `go/cmd/gdoc` (the `build` command), `go/boundary/boundary_test.go` (`allowedModules` gains two lines), `go/internal/live` (the live drift test), `go.mod`; docs.
- What exists to port: `docs/v2/spikes/config/gen.py` (597 lines: `styles_xml`, `numbering_xml`, `settings_xml`, `header_xml` with `logo_drawing`, `footer_xml`, `table_xml`, `toc_xml`, `cover_blocks`, `label_block`, `legend_block`, `document_xml` with the `sectPr`, the four `_rels` and `[Content_Types].xml`, `build`); `docs/v2/spikes/config/compare.py` (357 lines, 50 `item` calls over documentStyle, namedStyles, header and footer, the logo's positioned object, the TOC element, the three tables, the rendered body, and three PDF items); `spike/render/internal/body` (goldmark GFM plus typographer plus `Mark`, `makeTable` with v1's recipe, `numbering.go`, `image.go`, `inline.go`), `spike/render/internal/frontmatter` (title required; optional `alt_title`, `doc_type`, `version`, `date`, `owner`, `classification`, `heading_numbering`, `revisions`; `TitleCandidate` from the file name); `spike/render/testdata/docs` (six markdown documents and five images the spike was measured against).
- `house.yaml` shape (top level): `meta`, `page`, `palette`, `fonts`, `defaults`, `styles` (nine), `body` (`bullet`, `numbered` with glyphs and indents), `header` and `footer` (`first`, `default`, each a list of paragraphs of runs, the first header carrying `logo: true`), `cover` (`leading_blanks`, `lines`, `trailing_blanks`), `labels`, `legend`, `front_matter` (an ordered list of 13 blocks: cover, label, table, blank, legend, toc), `tables` (three, each `columns_pt`, `border`, `rows` of `cells` with fill, padding, valign and paragraphs of runs), `table_text`, `toc` (`live`, `instr`, `tab_stop_pt`, `size_pt`), `heading_numbering` (`mechanism: literal`, `level1_format: "{n}-{title}"`), `logo` (`mime`, `px`, `width_pt`, `height_pt`, `anchor`, `base64`).
- The master: `gdoc/templates/altery-group-policy-v1.0/template.docx`, 194 KB. etree round-trips it with one apostrophe of difference (spike `go/xmltest`).
- The 160-item measurement, 2026-08-29: 137 identical, 21 the same value stated two ways (the template declares a heading colour and overrides it per paragraph, the config states the effective one), two real: a 0.001pt rounding on the logo offset and a page count from a stale contents list. Worst position offset on the rendered PDFs 2.5pt.
- Dependencies identified: `github.com/beevik/etree` and `github.com/yuin/goldmark`, both agreed in SPEC.md with their reasons, both with no transitive modules. `goccy/go-yaml` already present parses `house.yaml` under the same strict rules as the front matter.
- v1's body table recipe (`gdoc/render/body.py`, `make_table`): fixed layout at the usable width, single 4-eighths borders, 3pt vertical cell margins, header row shaded and repeated, alternate row banding, `cantSplit`, column widths from content with floors scaled to the page. The spike ported it to Go already.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own commit.
- Make small, focused changes.
- **CRITICAL: every task MUST include new or updated tests** for the code it changes. Success and failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** `make test` runs `-race`; keep it green.
- **CRITICAL: update this plan file when scope changes during implementation.**
- **CRITICAL: a house-style test states the house value as a literal.** `if got != 595.28` and never `if got != house.Page.WidthPt`. A test that reads the constant it checks is a mirror. The v1 rule, CLAUDE.md "A house-style test must never read the constant it tests".
- **CRITICAL: no network in the render path.** `house`, `render`, `body`, `cover` and `drift` import neither `net/http` nor `internal/gapi`. `build` opens no policy and no session. The boundary test's allowlists do not change.
- **CRITICAL: facts only in Go.** The build reports what it wrote and what it could not render; whether the result is good is Nail's, in Word or in Drive.
- No em dashes in any text this plan produces, code comments, commit messages and the docs included.

## Testing Strategy

- **Unit tests**: `house` parses the embedded file and refuses an unknown key, a missing section and a wrong type by name. `render` is tested part by part on the XML it emits: the part exists, it parses, and the values are the house literals. `body` is tested on the six spike documents as golden files of the emitted `document.xml` body, plus focused tests per construct. `cover` is tested on the front matter shapes v1 accepts and refuses. `build` is tested over a temp directory: the file exists, unzips, every part is well-formed XML.
- **Offline drift test** (`go/internal/drift`, in `make test`): builds the spike's `03-policy.md` from the embedded config, opens the master `template.docx`, and compares every value compare.py checks that lives in the XML. Two known differences are named and allowed; anything else fails.
- **Live drift test** (`go/internal/live`, opt-in behind `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`): uploads the built docx and the master docx with conversion into the test folder, reads both through the Docs API, runs the ported 160-item comparison minus the three PDF items, prints the table, fails on any verdict that is not `IDENTICAL`, `CLOSE`, or one of the two known differences, and trashes both documents.
- **Boundary and modules**: `allowedModules` gains two entries with the SPEC reason in the map; `TestAllowedModulesAreReallyRequired` keeps them honest; `go mod graph` shows no module the allowlist does not name.
- Coverage standard: every exported function under `go/internal/` has a test; the new packages at or above 80%.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- `cd go && go vet ./...`
- `make build` and `make dist`, with the per-platform binary sizes recorded before and after in Task 8

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

`gdoc build` reads the note, splits its front matter into the author's keys (`cover` reads them) and the body markdown, parses the embedded `house.yaml` (`house`), walks the markdown with goldmark into body paragraphs (`body`), assembles the front-matter blocks in the order `house.yaml` lists them followed by the body (`render`), writes the eleven parts and the logo into a zip, and prints one object naming the file, its size, the title it used and every warning. No request leaves the machine.

Key design decisions and why:

- **From scratch, not surgery.** The docx is written from the config. Nothing copies the master and edits it. That is the 2026-08-29 decision: v1's surgery file became the real template and the master stopped describing the output. `render` is the one place that knows OOXML element names, and it reads every value from `house`.
- **etree for writing, too.** gen.py concatenates strings. The Go port builds elements with etree and serialises, because a string template escapes nothing and one `&` in a note's title breaks the part. The spike's `ooxml` helpers are the model.
- **The body walker is the spike's, with tables.** It already matches v1 on the six documents. Pipe tables get v1's `make_table` recipe, which the spike also carries. Code blocks are refused with a warning naming the line, never silently dropped: Nail decided they stay out, and a note that carries one should say so on the envelope.
- **Heading numbering as v1 does it.** `house.yaml` gives the level-1 format, `{n}-{title}`. Authored numbers already in a heading are respected, and `heading_numbering: false` in the front matter turns it off. The spike's `numbering.go` holds those rules.
- **The cover reads v1's keys.** `title` required, `alt_title`, `doc_type`, `version`, `date`, `owner`, `classification`, `heading_numbering`, `revisions` optional. A missing title is a refusal that names a candidate from the file name, the spike's `TitleCandidate`, and the skill proposes it; the binary never invents one. The `gdoc:` block stays `internal/frontmatter`'s and `cover` never reads it.
- **Two drift tests, one list of items.** The item list is written once, in `drift`, as names and extractors. The offline test feeds it values read from XML, the live test feeds it values read from the Docs API answer. Same names, so a difference in one is findable in the other.
- **The PDF items are dropped.** compare.py read a PDF export for three items (page count, page-1 size, page-1 image count). SPEC's Never list says gdoc never exports a PDF, so the live test does not either. The page count was one of the two known differences anyway. The plan says this rather than quietly reaching 157.

## Technical Details

**Principles.** Serves: 1 (one binary, two libraries with written reasons, the house style inside the file), 3 (the build reaches no network at all), 4 (the terminal carries the warnings; nothing is written into a document). The master `.docx` is provenance, not input.

**Tech stack.** Go 1.27, standard library plus `goccy/go-yaml`, `beevik/etree`, `yuin/goldmark`.

**Global constraints:**

- `gdoc build`: exactly one JSON object on stdout, exit 0 iff `ok`. It never prompts and never reads stdin.
- `--md` is required and must exist; `--out` is required, and an existing file there is refused unless `--force` says so (the atomicfile rule: not knowing must never resolve to overwrite). The write goes through `internal/atomicfile`.
- `--house <path>` replaces the embedded config for one run. The file is parsed under the same strict rules, and `data.house` names `"embedded"` or the path, so a document built from a draft style says so.
- Images: a relative path is resolved against the note's directory; a `data:` URI is decoded; an `http(s)` URL is refused naming the line, v1's rule, because a publish that reached the network would depend on a link that expires. PNG and JPEG only.
- Every `w:t` is escaped by etree; no value from the note or the config is ever concatenated into XML text.
- Commit messages: `feat(v2): ...`, `fix(v2): ...`, `test(v2): ...`, `docs: ...`. All commits from the repo root.

**`house`.** `//go:embed house.yaml` beside the package, copied from `docs/v2/spikes/config/house.yaml` in Task 1, and the spike copy is then a pointer to the real one (the spike README says so). Types mirror the YAML one to one, `yaml.Strict()`, and `house.Load()` returns the embedded config while `house.LoadFile(path)` returns a named one. Points stay `float64` in points; twips are computed at the writer (`twips(pt) = round(pt*20)`), never stored.

**`render`.** `render.Build(cfg *house.Config, c cover.Fields, body []*etree.Element, media []Media) (*Package, error)` returns the parts in order; `Package.WriteTo(w io.Writer)` zips them with `archive/zip`. Parts, as gen.py: `[Content_Types].xml`, `_rels/.rels`, `word/_rels/document.xml.rels`, `word/_rels/header2.xml.rels`, `word/document.xml`, `word/styles.xml`, `word/numbering.xml`, `word/settings.xml`, `word/header1.xml`, `word/header2.xml` (first page, carries the `wp:anchor` logo), `word/footer1.xml`, `word/footer2.xml`, `word/media/logo.png`, plus one `word/media/imageN.*` and one relationship per body image. The section properties carry the four header and footer references, page size and margins in twips, `pgNumType`, and `titlePg` when `different_first_page` is true. The running head in `header1.xml` is the cover's running head (alt title, else title).

**`body`.** `body.Render(cfg *house.Config, markdown []byte, base string, numbering bool) (Result, error)` with `Result{Blocks []*etree.Element; Media []Media; Warnings []string}`. Constructs: paragraphs, headings 1 to 6 with the house styles and literal numbering, bullet and numbered lists to three levels using `numbering.xml`'s two abstract lists, bold, italic, strikeout, `==mark==` as highlight, links as real hyperlinks (`w:hyperlink` with a relationship), inline images sized to the usable width at most, pipe tables with the v1 recipe, horizontal rules as a rule paragraph, block quotes as indented paragraphs. Refused with a warning naming the line: fenced and indented code blocks, HTML blocks, footnotes, task-list boxes (rendered as plain text). Smart punctuation as the spike sets it.

**`cover`.** `cover.Read(src []byte) (Fields, []byte, error)`: the fields, the body markdown after the front matter, and an error for a missing title carrying `Candidate`. Validation as the spike's: `classification` from the fixed set, `revisions` a list of `{version, date, author, approved_by, approval_date, section, change}`, the date rendered in UK long form on the cover. The `gdoc:` key is skipped, never validated here.

**`drift`.** `drift.Items` is the list: each item has a name, a tolerance, and two extractors, `FromDocx(*docx.Package) any` and `FromDoc(map[string]any) any`. `drift.Compare(a, b []Value) []Row` gives `IDENTICAL`, `CLOSE`, `DIFFERENT` or `MISSING` per row with compare.py's verdict rules (0.75pt default tolerance, 2pt on the logo offsets, 0 on line spacing and page count). `drift.Known` names the two accepted differences. The offline test runs `FromDocx` on both files; the live test runs `FromDoc` on both API answers.

**`gdoc build` output:**

```jsonc
{
  "out": "/abs/path/file.docx", "bytes": 48213,
  "title": "Supplier Register Policy", "running_head": "Altery - Supplier Register Policy",
  "house": "embedded",
  "body": { "paragraphs": 41, "headings": 9, "lists": 3, "tables": 2, "images": 1 },
  "warnings": ["line 88: a fenced code block is not rendered in the house style and was left out"]
}
```

## What Goes Where

- **Implementation Steps** (`[ ]` checkboxes): the five packages, the command, the two drift tests, the dependency allowlist, the documentation.
- **Post-Completion** (no checkboxes): the live drift run on Nail's machine, Nail opening a built docx in Word and in Drive, the binary-size delta recorded.

## Implementation Steps

---

### Task 1: `internal/house`: the house style as a parsed, embedded file

**Files:**
- Create: `go/internal/house/house.go`, `go/internal/house/house.yaml` (copied from `docs/v2/spikes/config/house.yaml`), `go/internal/house/house_test.go`
- Modify: `docs/v2/spikes/README.md` (the spike's `house.yaml` now says the live copy is `go/internal/house/house.yaml`)

**Interfaces:**
- Produces: `house.Config` and its nested types mirroring the YAML; `house.Load() (*Config, error)` for the embedded file; `house.LoadFile(path string) (*Config, error)`; `Config.Validate()` for what YAML cannot say (nine styles present, `front_matter` names only known blocks and existing table refs, the logo decodes as PNG of the stated pixel size).

- [x] write the failing tests: the embedded file loads; page width is `595.28` and height `841.89` as literals; `styles.heading_1` is 16pt, bold, `#22265F`, 12pt before, keep with next; `heading_numbering.level1_format` is `{n}-{title}`; `toc.instr` carries `Heading 1,1,Heading 2,2,Heading 3,3`; `front_matter` has 13 blocks in the order cover, label, table, blank, table, blank, legend, blank, label, table, blank, label, toc (check the actual order against the file first and state it as literals); an unknown key is refused naming it; a `front_matter` block naming a table that is not in `tables` is refused; the logo base64 decodes and is a PNG of the stated pixel size; `LoadFile` on a missing path is an error naming the path
- [x] run the tests and watch them fail
- [x] implement, with `//go:embed house.yaml` and `yaml.Strict()`
- [x] run the tests, gofmt, vet: green
- [x] commit: `feat(v2): house.yaml parsed and embedded, the house style as a file`

---

### Task 2: The two dependencies, allowlisted with their reasons

**Files:**
- Modify: `go/go.mod`, `go/go.sum`, `go/boundary/boundary_test.go`

- [ ] `go get github.com/beevik/etree` and `go get github.com/yuin/goldmark`, pinned
- [ ] add both paths to `allowedModules` with the SPEC.md reason beside each (etree: `encoding/xml` corrupts OOXML; goldmark: parses the hub markdown, images in headings included, mark in 55 lines)
- [ ] run the boundary test: `TestNoThirdPartyDependencies` and `TestAllowedModulesAreReallyRequired` both pass, and `go mod graph` names no module outside the four
- [ ] record `make dist` sizes for the three platforms before and after in this task's notes below
- [ ] commit: `chore(v2): allow etree and goldmark, the two modules SPEC.md agreed`

---

### Task 3: `internal/render`: the shell, from the config

**Files:**
- Create: `go/internal/render/render.go` (Package, Build, WriteTo), `go/internal/render/styles.go`, `go/internal/render/numbering.go`, `go/internal/render/headfoot.go` (header, footer, the positioned logo), `go/internal/render/front.go` (cover, labels, tables, legend, TOC, blanks), `go/internal/render/xml.go` (element helpers, twips, namespaces), `go/internal/render/render_test.go` and one test file per source file
- Create: `go/internal/render/testdata/` golden parts for a build with an empty body

**Interfaces:**
- Consumes: `house.Config`, `cover.Fields` (Task 5 defines it; this task uses a stub struct with title, alt title, running head, version, date, revisions, and a stand-in is replaced in Task 5), body elements.
- Produces: `render.Build`, `render.Package` with `Parts []Part{Name string; Data []byte}` in zip order, `Package.WriteTo(io.Writer) error`.

- [ ] write the failing tests: `Build` with an empty body produces exactly the eleven XML parts and the logo; every part parses with etree; `styles.xml` defines the nine styles with the house literals (Heading1 32 half-points, bold, `22265F`, 240 twips before); `numbering.xml` has two abstract lists with the three bullet glyphs `●`, `○`, `■` and the numbered formats at the house indents; `header2.xml` carries a `wp:anchor` with the logo's extents and offsets in EMU derived from the house points and `behindDoc` as the config says; `header1.xml` carries the running head text and the orange run; `document.xml` opens with the cover lines at 29pt bold centred, then the three tables with the row heights, cell fills (`#F5D1AE` on the first cell of version control), padding and border widths from the config, then the TOC field with the house `instr` and the `w:fldChar begin/separate/end` sequence, then the `sectPr` with A4 in twips (`11906` by `16838`), margins, `titlePg`, and the four references; a title containing `&` and `<` is escaped; the zip lists `[Content_Types].xml` first
- [ ] run the tests and watch them fail
- [ ] implement, porting gen.py function by function and reading every value from the config
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `feat(v2): render builds the house docx shell from house.yaml`

---

### Task 4: `internal/body`: the goldmark walker, with tables

**Files:**
- Create: `go/internal/body/body.go`, `go/internal/body/build.go`, `go/internal/body/inline.go`, `go/internal/body/numbering.go`, `go/internal/body/image.go`, `go/internal/body/table.go`, `go/internal/body/mark.go`, tests per file, `go/internal/body/testdata/` (the six spike documents and five images copied from `spike/render/testdata/docs`, plus golden `document.xml` bodies)

**Interfaces:**
- Consumes: `house.Config` for sizes, indents, colours and the level-1 format; the note's directory for image paths.
- Produces: `body.Render(cfg, markdown, base, numbering) (Result, error)` with `Result{Blocks, Media, Warnings, Counts}`.

- [ ] write the failing tests: each of the six documents renders to its golden body (generate the goldens from the spike's output where the spike agrees with v1, and read each once before committing it); a pipe table becomes a `w:tbl` with the v1 recipe as literals (fixed layout, borders `sz="4"`, header row shaded and `tblHeader`, alternate banding, `cantSplit`, column widths summing to the usable width); a heading `## 2. Scope` keeps its authored number and a heading without one gets `{n}-` at level 1; `heading_numbering=false` numbers nothing; nested bullets to three levels use `ilvl` 0, 1, 2; `==mark==` becomes a yellow highlight run; a link becomes `w:hyperlink` with a relationship id in `Media`; an image by relative path is read and sized to the usable width when wider; an `http` image is refused naming the line; a fenced code block yields a warning naming the line and no block; smart quotes are the characters, not entities
- [ ] run the tests and watch them fail
- [ ] implement, porting the spike's body package and adding `table.go`
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `feat(v2): body renders the hub markdown, tables included, through goldmark`

---

### Task 5: `internal/cover`: the note's own front matter

**Files:**
- Create: `go/internal/cover/cover.go`, `go/internal/cover/cover_test.go`
- Modify: `go/internal/render` (replace Task 3's stub with `cover.Fields`)

**Interfaces:**
- Produces: `cover.Fields{Title, AltTitle, DocType, Version, Date, Owner, Classification string; HeadingNumbering bool; Revisions []Revision}`, `cover.Read(src []byte) (Fields, body []byte, error)`, `cover.MissingTitle` error with `Candidate`, `cover.TitleCandidate(body, filename)`.

- [ ] write the failing tests: a note with only `title` reads with `HeadingNumbering` true and no revisions; every optional key reads; a `classification` outside the fixed set is refused naming the value; a revision missing `version` is refused naming the field; a note with no title is `MissingTitle` carrying the candidate from the first heading, else from the file name with the date prefix stripped; the `gdoc:` key is ignored whatever its shape (v1's string and v2's block both); a note with no front matter reads as no title and the whole file as body; the date renders as `8 September 2026`
- [ ] run the tests and watch them fail
- [ ] implement, porting the spike's frontmatter package under the new name
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `feat(v2): cover reads the note's title, version, date and revisions the way v1 did`

---

### Task 6: `gdoc build` on the envelope

**Files:**
- Modify: `go/cmd/gdoc/main.go` (dispatch and usage), create `go/cmd/gdoc/build.go`, `go/cmd/gdoc/build_test.go`, `go/cmd/gdoc/testdata/build-note.md`

**Interfaces:**
- Consumes: `cover.Read`, `house.Load`/`LoadFile`, `body.Render`, `render.Build`, `atomicfile`.
- Produces: `gdoc build --md <note> --out <file.docx> [--house <path>] [--force]` and the data shape in Technical Details.

- [ ] write the failing tests: a note builds to a file that unzips into the expected parts, and `data` reports `bytes`, `title`, `running_head`, `house: "embedded"` and the body counts; a missing `--md` or `--out` is refused naming the flag; an existing `--out` is refused without `--force` and replaced with it; `--house` pointing at a copy of the embedded file reports the path; a note with no title fails naming the candidate; the warnings from `body` reach `warnings`; no request is made (the session factory is never called); `--help` still lists the ten commands
- [ ] run the tests and watch them fail
- [ ] implement
- [ ] run the tests, gofmt, vet, `make build`: green; build the spike's `03-policy.md` by hand once and open the result in Word or upload it by hand to check it is a valid document
- [ ] commit: `feat(v2): gdoc build writes a house-style docx from a note, offline`

---

### Task 7: The drift tests, offline and live

**Files:**
- Create: `go/internal/drift/items.go`, `go/internal/drift/compare.go`, `go/internal/drift/docx.go` (reading values out of docx XML), `go/internal/drift/drift_test.go` (the offline gate), `go/internal/drift/testdata/master.docx` (a copy of `gdoc/templates/altery-group-policy-v1.0/template.docx`, with a note in the test saying which file it mirrors and that the file under `gdoc/templates/` stays v1's)
- Modify: `go/internal/live/live_test.go` (the live gate), `go/internal/docx` only if a reader helper is needed and belongs there

**Interfaces:**
- Produces: `drift.Items []Item`, `drift.Compare(a, b []Value) []Row`, `drift.Known` (the two accepted differences by item name), `drift.FromDocx(pkg) []Value`, `drift.FromDoc(answer) []Value`, `drift.Table(rows) string` for the printed report.

- [ ] write the failing tests: `Compare` gives `IDENTICAL` on equal, `CLOSE` within 0.75pt, `DIFFERENT` beyond, `MISSING` on one side nil, and honours a per-item tolerance; `FromDocx` on the master reads A4 in points, the margins, `titlePg`, the nine style sizes and colours, the logo extents and offsets, the TOC `instr`, the three tables' sizes, column widths, fills, texts, borders and row heights; **the offline gate**: build `03-policy.md` with the embedded config, run `FromDocx` on it and on the master, `Compare`, and fail on any row that is `DIFFERENT` or `MISSING` and not in `Known`, printing the table on failure
- [ ] run the tests and watch them fail
- [ ] implement `items.go` as one list with both extractors per item; port compare.py's 47 non-PDF items by name so a row here matches a row in the 2026-08-29 report
- [ ] write the live gate in `internal/live` behind `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`: upload the built docx and the master with conversion into the test folder (both learned by the guard from the creates), `documents.get` each with `includeTabsContent=true`, `FromDoc` on both, `Compare`, print the table, fail on any unknown `DIFFERENT` or `MISSING`, trash both in `t.Cleanup`
- [ ] run the offline gate: green, and paste its table into this plan under this task
- [ ] run the tests, gofmt, vet: green
- [ ] commit: `test(v2): the drift gate, offline on the XML and live through the Docs API`

---

### Task 8: Verify acceptance criteria

**Files:**
- Modify: whatever the checks below break.

- [ ] verify PLAN.md M5's gate: the 160 items with no new differences. Run the live gate once on this machine; if the token is not available to the run, say so here rather than skipping silently
- [ ] verify the module graph gained nothing transitive: `go mod graph` lists exactly `goccy/go-yaml`, `beevik/etree`, `yuin/goldmark` and the standard library
- [ ] record the cross-platform binary-size delta: `make dist` sizes for darwin/arm64, darwin/amd64 and windows/amd64, before Task 2 (from the M4 build) and after Task 7, in this plan
- [ ] verify no network in the render path: none of `house`, `render`, `body`, `cover`, `drift` imports `net/http` or `internal/gapi`, and the boundary test's allowlists are unchanged
- [ ] verify no judgement leaked into Go: grep the new packages for `handled`, `accepted`, `rejected`, `matters`, `drift` as a field name (the package name is fine), `should`, `decide`
- [ ] verify every house-style test states its value as a literal: grep the new tests for `cfg.` and `house.` on the right-hand side of a comparison, and rewrite any that read the constant
- [ ] run the full suite with `-race`, gofmt, vet, `make build`, `make dist`
- [ ] verify coverage: every exported function under `go/internal/` has a test, the five new packages at or above 80%
- [ ] commit any fixes this task made

---

### Task 9: [Final] Update documentation

**Files:**
- Modify: `README.md`, `CLAUDE.md`, `docs/v2/PLAN.md`, `docs/v2/SPEC.md`, `docs/v2/spikes/README.md`

- [ ] update `README.md`: `gdoc build`, the cover fields a note needs, what renders and what does not (code blocks), `--house`, and that publishing is M6
- [ ] update `CLAUDE.md` under "v2 lives at `go/`": the five new rooms in the table; a section "The generator reads `house.yaml` and nothing else" with the embed, the override, the escaping rule, the no-network rule, the literal-value test rule, the two drift gates and the two known differences, the PDF items dropped and why; `allowedModules` now names three
- [ ] update `docs/v2/SPEC.md` "The generator and `house.yaml`": the gate runs both ways, and the PDF items are out
- [ ] update `docs/v2/PLAN.md`: mark M5 done with the date, the binary-size delta, and what it leaves for M6 (the upload, the `gdoc:` block written by publish, the v1 front matter migration from `docs/backlog/v1-frontmatter-migration.md`)
- [ ] update `docs/v2/spikes/README.md`: `config/house.yaml` is superseded by `go/internal/house/house.yaml`, `gen.py` and `compare.py` are ported
- [ ] run the full test suite one more time
- [ ] commit: `docs: M5 lands, the generator written down`
- The harness moves this plan to `docs/plans/completed/` when the run finishes.

## Post-Completion

*Items needing a real account or a person. No checkboxes.*

**Manual verification:**

- Run the live drift gate once on Nail's machine: `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1 go test -run TestLiveDrift ./internal/live/`. It uploads two documents into the test folder and trashes both. Paste the table into the PR.
- Build one of Nail's real notes with `gdoc build` and open the docx in Word, then upload it by hand into the test folder and open it in Docs. Look at the cover, the logo, the three tables, the contents list after one refresh, a heading with a number, a bulleted list, a table in the body.
- Compare that document side by side with the same note published by v1.

**Known and out of scope for this plan:**

- Publishing into Drive, the `gdoc:` block written by the publish, and the v1 front matter migration are M6.
- Code blocks are not rendered, by decision. A note carrying one gets a warning naming the line.
- A second template. Everything here is measured against `altery-group-policy-v1.0`.
- Pictures inside a Google Doc on the way back out (`read`) are unchanged: `docs/backlog/read-pictures-and-drawings.md`.
