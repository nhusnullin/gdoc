# Can a Go port match the Python renderer, pixel for pixel?

**Spike, 29 August 2026. Throwaway code, kept only as evidence.**

## Principles

Serves: principle 1. pandoc is the one external program left, and it survives
because removing it in Python means writing a Markdown parser. In Go it is an
import, and the build is one static binary with nothing to install on the machine
it runs on.

Strains: principle 1's own "a new third-party package needs a stated reason".
This adds five at once. Each is named with its reason and its licence under
Dependencies below, all are permissive and pure Go, and none of them is a program
a package manager has to fetch, which is the line principle 1 actually draws.

## The answer

Yes, exactly. Not "close enough": zero differing pixels.

Six documents were built by both renderers, published to Google Drive by both,
exported as PDF by Google, rasterised at 300 dpi and compared pixel by pixel with
a zero tolerance. **418,385,088 pixels compared. 0 differ.**

That is the whole finding. The rest of this note is how it was measured, what had
to be answered differently in Go, and the six real defects the port found in the
Python renderer along the way.

## What was measured

Ground truth is Google's own PDF export of the published document, because Google
is the layout engine this tool already treats as authoritative. LibreOffice was
used only as a fast inner loop during development and is not part of any result
below.

For each document: build with `gdoc.render.build`, build with `gdocgo build`,
publish both through their own two-pass pagination flow, export both PDFs from
Drive, rasterise both with `pdftoppm`, and diff.

| Document | What it exercises | Pages | Pixels at 300 dpi | Differing |
|---|---|---|---|---|
| 01-kitchen-sink | headings 1-6, nested lists, ordered lists restarting and starting at 7, three tables, code block, block quote, smart punctuation, self-numbered headings, appendix | 6 | 52,298,136 | 0 |
| 02-pictures | PNG figure, data-URI picture, picture inside a Strong-wrapped heading, two pictures after one paragraph, a picture wider than the column, links in prose and in table cells | 7 | 61,014,492 | 0 |
| 03-policy | a realistic policy: full front matter, revisions, confidential classification, alternate title, appendices | 6 | 52,298,136 | 0 |
| 04-minimal | three lines of front matter, everything else defaulted | 4 | 34,865,424 | 0 |
| 05-edge-cases | ampersands, accents, duplicate heading text, empty table cell, unbreakable token, task list, strikethrough, JPEG, a link inside a heading | 5 | 43,581,780 | 0 |
| 06-long | 40 sections, 80 contents entries, 20 pages | 20 | 174,327,120 | 0 |
| **Total** | | **48** | **418,385,088** | **0** |

The contents lists agree entry for entry as well, including all 80 entries and
their page numbers on the long document, with no blanks on either side.

The whole comparison was run twice, on separate publishes, with the same result.

## Why it can be exact

Because neither renderer builds a .docx. Both copy the master and operate on it.

Everything the eye recognises as the house document, the cover, the logo, the
running head, the footer, the coloured control tables, is carried across
untouched as bytes. Only the parts that must change are parsed and rewritten. A
.docx is a zip of XML, so Go's `archive/zip` plus an XML DOM does exactly what
python-docx plus lxml does, with the same reach and rather more control.

The generated body is the other half, and it is exact for a duller reason: the
Python renderer already writes its own OOXML by hand, using constants measured
out of the template. Those constants port as literals. Nothing about them is
Python.

## Where Go needed a different answer

Four places, and one of them is a genuine improvement rather than a workaround.

**Markdown parsing.** pandoc becomes goldmark, which is a library rather than a
program. This is the single biggest gain: it removes the last load-bearing
external program from the publish path, so the renderer becomes one static
binary. Matching pandoc's output took three adjustments: the typographer had to
be configured to emit real characters rather than HTML entities, HTML entity
references in the source had to be decoded (goldmark resolves those in its HTML
renderer, which this is not), and `==mark==` needed a small extension, written
here in about fifty lines rather than pulled in as a dependency.

**Picture embedding.** python-docx owns the media part, the relationship, the
content-type declaration and the DrawingML behind `run.add_picture`. In Go that
is written by hand: read the PNG `pHYs` chunk or the JPEG frame header for size
and density, fall back to 72 dpi exactly as python-docx does, then emit the same
`wp:inline`. This is the only place the Go port carries real weight the Python
one does not, and it is where a wrong answer would be most visible. The extents
match to the EMU.

**Page numbers.** This is the improvement. The Python renderer finds which page a
heading landed on by extracting text from Google's PDF export and matching it
line by line. Two Go PDF text extractors were tried and both were unusable:
"Contents" came back as `d4 gnOia1-ehne eC stntno`, and two headings merged into
one row. Worse than failing, the fuzzy fallback matched a heading against its own
contents entry and reported the contents page as the heading's page.

Google's PDF export carries a proper document outline, one entry per heading,
each pointing at the page object it sits on. Reading that is exact, needs no text
reassembly, and has no failure mode of this kind. The Go port uses it, matched
with a cursor that only moves forward so two sections called "Overview" resolve
to their own pages rather than both to the first. **The Python renderer should
adopt this too**; it is a smaller change there than it was here.

**Attribute order.** Go's regexp engine has no lookahead, so two patterns that
the Python code writes with lookaheads had to be rewritten to match the tag whole
and read its attributes out. Forced into that, the port found that one of those
patterns was already wrong on the Python side. See defect 4.

## What the port found in the Python renderer

Porting code is the cheapest code review there is, because every assumption has
to be restated. Six defects, all real, all now fixed in this worktree. The Python
suite still passes: 652 tests.

1. **`==highlight==` never highlights.** pandoc 3.x emits a `Span` carrying the
   class `mark`, not the `Highlighted` node the branch was written for, so that
   branch is dead code. The one mark that means "a human still has to fill this
   in" was silently rendering as ordinary text.

2. **Every heading numbered `0.1-` when a picture sits on its own line.**
   `shallowest_heading_level` skips a heading holding nothing but a picture, and
   the skip is defeated by exactly the `Strong` wrapper Drive puts around
   `# **![][image1]**` that the rest of the module was hardened against. The
   surviving empty `Strong` counts as a word, the heading counts as level 1, and
   every real heading beneath it is numbered one level too deep.

3. **`~~strikeout~~` renders as nothing at all.** The branch handling
   `Strikeout`, `SmallCaps`, `Superscript` and `Subscript` takes `content[-1]`,
   which is right for `Span` and `Cite`, whose content is `[attributes, inlines]`.
   For these four the content *is* the inline list, so `content[-1]` takes the
   last node rather than the list, and iterating a dict yields its keys. Text
   disappears from the document without a warning.

4. **Every published document carries two dangling `<Override>` entries.**
   `strip_comments` removes the comment parts and their relationships but its
   content-type pattern is anchored on `PartName` coming first. The bundled
   master is a Google Docs export and writes `ContentType` first, so the pattern
   matches nothing. The package then declares content types for two parts that no
   longer exist. Google tolerates this. It is still invalid OOXML, and it is the
   same attribute-order trap the bookmark code documents at length.

5. **python-docx refuses a JPEG with no JFIF or Exif segment.** Go's own
   `image/jpeg` encoder produces exactly such a file, and the publish dies with a
   bare `UnrecognizedImageError`. Such files are unusual but legal. The Go reader
   takes the size from the frame header and accepts them. Evidence kept at
   `go/testdata/docs/photo-nojfif.jpg`.

6. **Link destinations are dropped.** `inline_runs` recurses into a `Link` and
   keeps only its text, so a published document has no live links anywhere. This
   is longstanding rather than a regression, but it is the one item on this list
   Nail names as something he relies on.

## What Go adds

Real hyperlinks, behind `--links`. A `w:hyperlink` wrapping the run, the
destination held as an external package relationship, styled the way a reader
expects. Verified end to end: seven live external links survive the conversion
into Google Docs, including one carrying a query string with an ampersand, and
two sitting inside table cells. The current pipeline produces zero.

The default keeps Python's behaviour, so the parity measurement above is honest.

Also, and worth stating plainly:

- **One static binary.** `otool -L` shows only macOS system libraries. No
  pandoc, no Python, no venv. 3.6 MB for the renderer alone, 20 MB with the
  Google API client linked in. Nothing to install on any machine.
- **About five times faster.** 0.15s against 0.75s on the twenty-page document,
  and 0.10s against 0.51s per build on the six-page policy. Most of the Python
  time is interpreter start plus the pandoc subprocess.
- **Byte-for-byte deterministic**, as is the Python renderer.

## Dependencies, and their licences

All permissive, all pure Go, all vendorable. Nothing needs installing anywhere.

| Library | Purpose | Licence |
|---|---|---|
| `github.com/yuin/goldmark` | Markdown parser, replaces pandoc | MIT |
| `github.com/beevik/etree` | XML DOM for the surgery | BSD-2 |
| `github.com/pdfcpu/pdfcpu` | PDF outline, for page numbers | Apache-2.0 |
| `gopkg.in/yaml.v3` | Front matter | MIT and Apache-2.0 |
| `google.golang.org/api`, `golang.org/x/oauth2` | Drive | BSD-3 |

AGPL was permitted for this spike and turned out not to be needed. No docx
library was used at all: the raw zip and XML approach is what python-docx is
doing underneath, and doing it directly is both smaller and more controllable
when the whole point is to leave most of the file alone.

## What is not proven

- **One template.** Every measurement is against `altery-group-policy-v1.0`. The
  surgery finds the cover by placeholder text and the tables by their
  first-column labels, so a second template would exercise paths this spike did
  not.
- **One credential.** Tested under `oauth` only. `service_account` was not
  exercised.
- **Everything that is not rendering.** The review flow, the comment threads,
  `edits`, `suggestions`, `restyle`, the pairing and baseline bookkeeping, and
  the two skills are all untouched. The guard was ported and unit-tested because
  it is the constraint most likely to be broken by a rewrite, but that is the
  only non-render module here.
- **Windows and Linux.** Built and run on darwin/arm64 only.
- **Fonts.** Both sides go through Google, so font availability never differed.
  A renderer that had to lay text out itself would be a different question, and
  this design never does.

## Recommendation, and what was decided

The fidelity question is answered and it is not close. Pixel fidelity is not an
argument against a Go rewrite, and the render path, which everyone assumed was the
risk, is the part that ports cleanly.

**Decided on 29 August 2026: the next version of gdoc is written in Go.** Recorded
in PRINCIPLES.md under that date. The reason is principle 1 rather than anything
in the numbers above: a single binary is the end of the road this repository has
been walking since the bundled OAuth client and `install.sh`. The numbers above
only removed the objection.

The decision is taken with one thing unproven and stated: `gdoc export` uses
pandoc as a docx *reader*, goldmark does not replace it, and that reader has to be
written. Proving it is the first task of the port, not the last.

Two things are worth taking into the Python tool immediately, whatever the port's
pace:

1. **The six defects.** Four are fixed against the Python code in this worktree
   and the suite passes; the other two are the Go side's to carry. They matter
   today, on documents already being published.
2. **The outline-based pagination.** It is strictly better than matching
   extracted PDF text, and it is a small change in Python.

## Reproducing this

```bash
cd go
go test ./...                     # 106 tests
go build -o /tmp/gdocgo ./cmd/gdocgo
go build -o /tmp/pixdiff ./cmd/pixdiff

/tmp/gdocgo build  testdata/docs/01-kitchen-sink.md --template ../gdoc/templates/altery-group-policy-v1.0/template.docx --out /tmp/go.docx
/tmp/gdocgo generate testdata/docs/01-kitchen-sink.md --template ... --folder <id> --name "..." --out /tmp/go.docx
/tmp/pixdiff --tolerance 0 --diff-dir /tmp/diff /tmp/pages-py /tmp/pages-go
```

`cmd/pixdiff` writes a diff image per differing page, unchanged pixels faded and
differences in red. That is what turned a 75,000-pixel number into "the
typographer is emitting HTML entities, and everything below it re-wrapped".

Go test coverage on the render path: body 95.1%, ooxml 93.5%, frontmatter 91.3%,
contents 89.2%, docx 87.9%, shell 86.3%.
