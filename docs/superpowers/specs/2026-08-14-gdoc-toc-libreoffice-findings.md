# Publishing without LibreOffice: what it cost, and what it fixed

Date: 2026-08-14. Status: solution built and tested, not yet wired into `generate`.

Issue #8 asked how to distribute the tool. That became a question about the publish
path's heavy dependencies, and since pixel fidelity is the priority, it narrowed to
one thing: **what does headless LibreOffice actually buy, and can anything else buy
it?**

Answer: it buys the contents list, and nothing else. Everything else it does to the
document is damage. A replacement is now implemented in `gdoc/render/contents.py`
and `gdoc/render/pagination.py`, tested offline and against live Drive.

All measurements used the bundled example on `gdoc-template` (PR #5, `fd75158`),
built with `gdoc.render.build`, uploaded to Drive as a Google Doc, exported back to
PDF, and compared at 150 DPI.

## 1. Google Docs never refreshes an imported contents list

A `.docx` contents list is a Word field carrying a cached result. Google renders
that cached text and never recomputes it.

Two documents were uploaded and left untouched. Both were polled for over thirteen
minutes and never changed: each kept the master template's placeholder contents
list the whole time. Every refresh seen earlier in the investigation was a human
clicking "Update table of contents", and the timings lined up with those clicks.

This contradicts `gdoc/render/shell.py:493`, which says regenerating the field is
something "Google Docs does on import". **That comment is wrong and should be
fixed.**

The consequence is severe. The master's cached result names the template's own
placeholder sections (`4. Policy Compliance`, `7. Policy Statements 1 - Details`,
`Addendum 1 - Variation of procedure for legal entity X`) with page numbers running
to 8 in a 6-page document. Without the LibreOffice pass, a published document keeps
a contents page describing a different document, indefinitely. The body is correct.
Only the contents page lies, which makes it the worst kind of defect for a policy:
the document looks immaculate.

## 2. Everything else LibreOffice does is damage

Uploading the untouched master to Google gives a reference for the house style.
Measured against it, and against the current pipeline's output:

| Property | House style (master) | Current pipeline (LibreOffice) | New path |
|---|---|---|---|
| `HEADING_1` bold | **true** | `null`, weight lost | **true** |
| Numbered list | 1, 2, 3 | 1, 2, **2** | 1, 2, 3 |
| First contents entry | page 4 | page **3**, wrong | page 4 |
| Contents font | Calibri | Calibri | Calibri |

LibreOffice's resave strips the heading weight, breaks list numbering, and puts the
wrong page number on the first contents entry. A manual TOC update corrects none of
them. It also shifts vertical metrics throughout, which is what made the current
output differ from every other variant by about 7% of pixels.

The new path is pixel-identical to a plain `--skip-toc` build on 5 of 6 pages
(**zero** differing pixels on pages 1, 2, 4, 5 and 6), differing only on the
contents page, which is the page it deliberately rewrites.

One comparison that looked meaningful is not: comparing a build's pages 1 and 2
against the master's is invalid, because `build_shell` fills the real front matter
into the cover and control tables. Those pages differ by design.

## 3. The replacement

`gdoc/render/contents.py` replaces the field's **cached result** and keeps the
field.

An earlier attempt deleted the field outright. That was wrong, and Nail caught it:
without a field, Google imports plain text, so the published document has no
contents list at all, only text shaped like one. No update button, no clickable
entries, and a heading somebody adds later never appears. Keeping the field means
the document holds a real contents list a reader can refresh, while the cached
result we write is already correct so nobody needs to.

Entries are written the way a well-behaved application writes them:

- Styled `TOC1..TOC3`, declaring no font and no weight, so they inherit the
  document default, Calibri, through the `Index` parent. Left to its own defaults
  Google renders a contents page in Arial with bold level-one entries.
- Wrapped in `w:hyperlink` to a `w:bookmarkStart` on the heading, so entries are
  clickable. Measured on the published document: 12 of 12 entries linked.
- Carrying the `IndexLink` character style, which is empty on purpose. It overrides
  Word's `Hyperlink` style, which would otherwise render every entry blue and
  underlined.

The malformed begin run is carried across untouched rather than repaired. Google
accepts it and still builds a real contents list, so repairing it is a change with
no observed benefit and unmeasured risk.

### Matching what a refresh produces

Keeping the field means a reader can refresh it, and Google imposes its own layout
when they do. Only the font is inherited; spacing and indents are Google's. So the
entries are written with Google's own values, or the contents page visibly jumps the
first time anybody clicks update. Measured on published documents:

| Property | Template default | Google's refresh | What we write |
|---|---|---|---|
| Line pitch | ~21.6pt | 16.4pt | 16.4pt (`after=20`, `line=276`) |
| Space above the block | 0 | 3pt, first entry only | 3pt, first entry only |
| Level 2 indent | 283tw | 360tw (18pt) | 360tw |
| Level 3 indent | 567tw | 720tw (36pt) | 720tw |

Space before and space after are added, not collapsed, by both Word and Google. So
the 3pt above the block goes on the first entry as direct formatting; putting it on
the style would widen every gap instead of just the first.

Result: the contents page before a refresh differs from after one by **0.087% of
pixels**, down from 1.127% before this was tuned. The residue is 0.2pt of cumulative
rounding across twelve lines, which is sub-pixel per line. Every other page is
pixel-identical.

The `TOC1..TOC3` and `Index` styles are measured constants in that module, lifted
once off a LibreOffice-produced document. The right tab at 9864 twips is the text
edge, which puts the page number at the margin; levels indent 0/283/567 twips.

The master's field is malformed, exactly as `toc.py` says: the begin marker, the
instruction and the separator all sit in one run where the format wants one run
each. So the field is located by finding the paragraph carrying the instruction and
running forward to the paragraph carrying the end marker.

### Page numbers, without a local layout engine

`gdoc/render/pagination.py` makes **Google the layout engine**. `generate` uploads
once with blank page numbers, exports the PDF, reads which page each heading landed
on, writes those numbers in, and uploads the version it publishes.

Two details that cost real debugging:

- **Match whole lines, never substrings.** `Appendices` also occurs inside body
  prose, and a substring match put two entries on the wrong page.
- **Skip the contents page.** Every heading appears there too.

`drift()` compares what was written against the published export. Empty means the
contents list describes the document it sits in. In measured runs it was stable on
the first try, because a page number is one or two characters at the right margin.

A caller with no pagination to offer, such as an offline `gdoc build`, passes no
pages and gets blank page numbers. A blank is honest. A wrong number is not.

## 4. The pixel gate ranks defects backwards

PR #5 Task 2 plans pixel comparisons as the release gate. On this evidence the
metric is actively misleading:

| Page | AE% | What was actually wrong |
|---|---|---|
| 2 | 22.7% | Nothing. A few pixels of vertical drift. |
| 3 | 3.1% | The contents list described a different document. |

A threshold on AE% fails on the harmless page and passes the broken one. A font
swap on thin text over a mostly white page scores about 1.5%; two pixels of drift
scores 22%. Two things to fix in Task 2:

1. **`AE%` is unusable as a threshold.**
2. **`tests/support/pdfdiff.py` cannot compare against Google output at all.** A
   LibreOffice render rasterises to 1241x1754 at 150 DPI and a Google export to
   1242x1755, and the harness refuses on size mismatch. The gate cannot currently
   see the thing it exists to protect.

The assertion worth more than both pixel comparisons needs no rasterising, and now
exists as `tests/test_contents_integration.py`:

> after publishing, assert every contents entry resolves to the page it claims.

## 5. Tests

- `tests/test_render_contents.py`, 16 checks, fully offline. Covers the field being
  **kept** with all four markers intact, the field bracketing the entries, no
  template placeholder surviving inside it, every entry linking to a bookmark that
  exists, one bookmark per heading with a matching end, the styles being added,
  entries declaring no font or weight, page numbers written or left blank,
  level-to-style mapping, and a refusal when a document has no headings.
- `tests/test_render_pagination.py`, 9 checks, no subprocess. Covers whole-line
  matching, the contents page being skipped, prose mentions not winning, wrapped
  headings, whitespace, and drift reporting.
- `tests/test_contents_integration.py`, the live path. Builds, publishes twice,
  asserts no drift, no leaked placeholder, a real `tableOfContents` object in the
  published document, and every entry clickable. Trashes both throwaway documents
  in a `finally`. Opt-in behind `GDOC_LIVE_PUBLISH_TEST=1`, because unlike the
  read-only integration checks it creates documents, and a plain `pytest` must not
  write to somebody's Drive.

Full suite: **187 passed, 1 skipped**, up from PR #5's 162, nothing broken.

Verified on the published document: native contents object present, 12 of 12
entries clickable, font Calibri, bold false, page numbers stable with no drift.

## 6. What is left

- **Wire it into `generate`.** Deliberately not done here, because PR #5 Task 6 owns
  that code and has nine open tasks. The integration test shows the exact
  composition: `build(skip_toc=True)`, `contents.write`, upload, `pagination.from_pdf`,
  `contents.write` with pages, upload, `pagination.drift` to verify.
- **Fix the comment at `shell.py:493`.**
- **Decide the fate of `toc.py`, `scripts/update-toc.sh` and `UpdateToc.bas`.** They
  become dead once `generate` uses this path, along with the LibreOffice dependency
  and the `slow` marker's need for it. `pdftotext` from poppler becomes the only
  external binary, and only for `generate`.
- **`--skip-toc` needs renaming or retiring.** There is no TOC pass to skip.
- **Deeper heading levels.** Only `TOC1..TOC3` ship, matching the three levels the
  template uses. A document nesting deeper needs `TOC4` and beyond, and today level
  4+ collapses onto `TOC3`.

## 7. Distribution, the original question

Pixel fidelity needs `python-docx`, `lxml` and a bundled binary template, none of
which ports to a single static Go binary. So the split stands:

- **Review** (`read`, `reply`, `export`, `capture`, `pair`): HTTP and JSON only.
  Portable, could be one binary.
- **Publish** (`build`, `generate`): needs the template and python-docx. Now also
  needs poppler, but no longer LibreOffice.

Dropping LibreOffice removes roughly 700MB from the publish environment and, on the
evidence above, improves the output.

## Evidence

`~/Documents/gdoc-toc-evidence/` holds the built `.docx` files, the Drive PDF
exports at each stage, and the rendered contents pages. Kept outside the job
scratch directory, which does not survive.

Documents named `SKIPTOC-EXPERIMENT *` remain in the configured output folder. None
have been trashed. One of them carries a stray Cyrillic character typed during a
manual update, which is its own argument against manual steps.
