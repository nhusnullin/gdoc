# Can LibreOffice leave the publish path? Measured findings

Date: 2026-08-14. Status: paused mid-investigation, two data points outstanding.

Context: issue #8 asks how to distribute the tool. That turned into a question about
whether the publish path can drop its heavy dependencies. Pixel fidelity of the
generated Google Doc is the priority, so the question narrowed to one thing:
**what does headless LibreOffice actually buy us, and can anything else buy it?**

All measurements used the bundled example on branch `gdoc-template` (PR #5, commit
`fd75158`), built through `gdoc.render.build`, uploaded to Drive as a Google Doc,
exported back to PDF, and compared.

## The single most important finding

**Google Docs never regenerates an imported table of contents on its own.**

An earlier version of these notes claimed Google refreshed the field
asynchronously after about four minutes. That was wrong. The refreshes observed
were a human clicking "Update table of contents". Two documents left untouched
were polled for over thirteen minutes each and never changed: they held the
master template's placeholder contents list the whole time.

This contradicts the comment at `gdoc/render/shell.py:493`, which states that
regenerating the field is something "Google Docs does on import". It does not.
That comment should be corrected.

### Why it matters

A `.docx` table of contents is a Word field carrying a cached result. The master
template's cached result lists the template's own placeholder sections
(`4. Policy Compliance`, `7. Policy Statements 1 - Details`,
`Addendum 1 - Variation of procedure for legal entity X`) with page numbers
running to 8 in a document that is 6 pages long.

So without the LibreOffice pass, a published document keeps a contents page that
describes a different document, indefinitely, until a human clicks update. The
body is correct. Only the contents page lies. For a compliance policy this is the
worst kind of defect: the document looks immaculate.

**That is what LibreOffice actually buys.** Not styling, not layout. It computes
the cached field result so the published document is correct with no human action.

## What the pixel gate would have done

PR #5 Task 2 plans pixel comparisons as the release gate. On this evidence the
metric ranks defects backwards.

| Page | AE% | What was actually wrong |
|---|---|---|
| 2 | 22.7% | Nothing. A few pixels of vertical drift. |
| 3 | 3.1% | Contents list described a different document. |

A gate on AE% fails on the harmless page and passes the broken one. Two concrete
problems to fix in Task 2:

1. **`AE%` is unusable as a threshold.** A font swap on thin text over a mostly
   white page scores about 1.5%. Vertical drift of two pixels scores 22%.
2. **`tests/support/pdfdiff.py` cannot compare against Google output at all.** A
   LibreOffice render rasterises to 1241x1754 at 150 DPI and a Google export to
   1242x1755, and the harness refuses on size mismatch. The gate cannot currently
   see the Google Docs look it exists to protect.

The assertion worth more than both pixel comparisons needs no rasterising:

> after a build, extract the contents list entries and assert they equal the
> document's real headings, in order, with every page number inside the page count.

That catches this whole class of defect in any language.

## Styling: the contents page font and weight

With `--skip-toc` and no other change, the contents page renders in Arial with
bold level-one entries. That is Google's own default TOC formatting, applied
because the document declares no TOC styles.

It is not inherited from the headings. The master's `Heading1/2/3` are not bold
(they are 16/14/12pt in blue `0041d3`), and the heading paragraphs carry zero
direct bold runs. Google simply uses its own defaults.

The master declares **no TOC styles at all**. LibreOffice invents `TOC1`, `TOC2`,
`TOC3` as a side effect of refreshing the field. All three are `basedOn="Index"`
with no font of their own, so they inherit the document default, Calibri.

Measured state of every document built, after each had its TOC updated by hand:

| Build | docx TOC styles | TOC font | Bold | First entry says | Actual |
|---|---|---|---|---|---|
| A: full pipeline | `Index`,`TOC1-3` | Calibri | no | page 3 | page 4 (wrong) |
| B: `--skip-toc` | none | Arial | yes | page 4 | page 4 |
| C: B + `TOC1-3` | `TOC1-3` | Calibri | no | page 4 | page 4 |
| D: B + `TOC1-3` + `Index` | `Index`,`TOC1-3` | Calibri | no | page 4 | page 4 |

Two things follow. Adding `TOC1`, `TOC2`, `TOC3` to the master template fixes both
the font and the bold, and `Index` appears not to be required (C works without
it). And the full pipeline is the only variant that gets a page number wrong: it
says the first section is on page 3 when it is on page 4, and a manual update does
not correct it.

Everything else was identical across all builds. Cell shading fills match in value
and count (`f3f8f9`x24, `bdcdd2`x15, `f5d1ae`x10, `ffffff`x10, and four singles),
as do the cover, control tables, running head, footer and body typography.

## The Docs API can restyle a table of contents

`documents.batchUpdate` with `updateTextStyle` over the `tableOfContents`
element's own `startIndex..endIndex` range is **accepted**. Setting
`weightedFontFamily: Calibri` and `bold: false` cleared the explicit Arial and the
bold, leaving the runs inheriting the document default.

So the styling half can be fixed by the tool after upload, with no template change
and no LibreOffice. What the API cannot do is refresh the field: the Docs API has
no request for that, so the entries still need the human click.

## Two data points still outstanding

Both documents are uploaded and deliberately untouched. Each needs exactly one
click on "Update table of contents", then re-measuring.

- **E**, plain `--skip-toc` with the API restyle already applied, entries still
  stale: <https://docs.google.com/document/d/1GtfynPZyc6HJR3QfODTyuugGro3xEEut9kVHizx8OEU/edit>
  Question: does the API-applied Calibri survive the refresh?
- **F**, `TOC1-3` in the docx, never touched by anyone:
  <https://docs.google.com/document/d/14MsFCquyVSPHkOsZp0kVT6RjpDbdCbLjua3U2sj8DpE/edit>
  Question: do the template styles alone give Calibri, with no API call and no
  other variable?

Earlier documents A to D are confounded, because each was clicked by hand while
other variables were also changing. E and F exist to remove that.

## Where this leaves the distribution question

Pixel fidelity needs `python-docx`, `lxml` and a bundled binary template. None of
that ports to a single static Go binary, so the review side and the publish side
have to split:

- **Review** (`read`, `reply`, `export`, `capture`, `pair`): HTTP and JSON only.
  Portable, could be one binary.
- **Publish** (`build`, `generate`): needs the template and python-docx.

Whether *LibreOffice* stays is the open question, and it now reduces to one
decision: is a mandatory human click on "Update table of contents" acceptable for
every published version? If yes, LibreOffice can go, the template gains `TOC1-3`,
and the page numbers come out better than they do today. If no, LibreOffice stays,
because nothing else computes the cached field result.

If the click is accepted it must be enforced, not remembered. A published policy
with the template's contents page is worse than a late one. Note also that a manual
edit is itself a hazard: one of the test updates typed a stray Cyrillic character
into the published contents page.

## Evidence

Copied to `~/Documents/gdoc-toc-evidence/` because the job scratch directory does
not survive. Holds the built `.docx` files, the Drive PDF exports at each stage,
and the rendered contents pages.

Seven documents named `SKIPTOC-EXPERIMENT *` remain in the configured output
folder. None have been trashed.
