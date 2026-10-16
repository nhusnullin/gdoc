# M21. A contents entry opens its heading

2026-10-16. Found by Nail in "Embedded retail card concept paper v0.9": after
a refresh of the contents list in Google Docs, every entry's hover card says
"Tab 1" and a click lands on the title page, not on the heading it names.

## Principles

Serves: 3, because the publish read-back gains a check that catches this class
of silent failure, and 4, because a reader who clicks a contents entry should
land on the section and not on the cover.

Strains: none. The 2026-09-27 decision "a heading carries a bookmark only when
a link names it" is superseded in part, with the measurement that retires it.

## The bug, measured

Every document gdoc has published has it. It is not new with M19.

| Document | Headings | With a heading id | Contents entries | Entry link shape |
|---|---|---|---|---|
| v0.9, `1AbUOmAgEJdyCZdHFKdSlX8FrbCfQlEpTWSRAloaIHV0` | 28 | 0 | 28 | `heading: {tabId: t.0}`, no `id` |
| v0.5, 13 October | 19 | 0 | 19 | the same |
| v0.4, 13 October | 16 | 0 | 16 | the same |
| `toc-font-experiment-copy`, 19 September | 12 | 0 | 12 | the same |
| Nail's "Group Application Development Policy V4.0", written in Docs | 85 | 85 | 20 | `heading: {id: h.…, tabId: t.0}` |

A contents entry is a link to a heading by its id. gdoc's headings come back
from Drive's import with no `headingId` at all, so the refresh writes a link
with the tab and no id, and Docs resolves that to the start of the tab.

## The cause

Drive's docx import mints a heading id for a heading paragraph only when the
paragraph carries a bookmark whose name opens with `_`. That is the convention
Google's own docx export uses: every heading gets
`<w:bookmarkStart w:name="_2csi3wz2phe8"/><w:bookmarkEnd/>` before its first
run, and a link to the heading is `<w:hyperlink w:anchor="_2csi3wz2phe8">`.
Word treats a bookmark opening with `_` as hidden, so it shows in no bookmark
list there either.

gdoc writes no such bookmark. Since 2026-09-27 a heading carries a bookmark
only when a `#` link in the note names it, named `h_<slug>`, wrapping the
heading's runs. That one is kept as a bookmark on import, shows as a flag, and
mints no id.

Measured on 2026-10-16 with one probe note (three headings, one `#` link, one
page break), uploaded with conversion into the test folder. All six are
`toc-headingid-probe-*` there.

| Variant | What each heading carried | Heading ids | The `#` link came back as |
|---|---|---|---|
| a, as gdoc builds it today (`1ODMFgBjoDJktG7DtbCUroUhtEPLcju-EEB5UvMoH9FE`) | nothing, except `h_who_decides` wrapping the linked heading | none | `link.bookmark` |
| b | an empty `_probeNNN` pair before the runs, on the headings with no bookmark | the two with the pair; the linked heading still none | `link.bookmark` |
| c | an empty `_probeNNN` pair on every heading, beside the `h_` one | all three | `link.bookmark` |
| d | an empty `probeNNN` pair, no underscore | none | `link.bookmark` |
| e | `_probeNNN` wrapping the runs, like the `h_` one | the two that carried it | `link.bookmark` |
| f, the shape this plan builds (`118is97pMYpL14kgtolRM3JuBlFup5B8rTZD5Pr8Ah5s`) | one empty `_h_<slug>` pair before the runs, on every heading, and the `#` link anchored at it | all three | `link.heading` with that heading's id |

So the leading underscore is the trigger, the range does not matter, the name
is not kept (Google mints its own `h.` id), and a jump anchored at the hidden
bookmark becomes a heading link, which `internal/docs` already decodes and
`internal/view` already prints as `#` and the heading's words.

Two more facts the plan rests on:

- Re-applying `namedStyleType: HEADING_1` through `updateParagraphStyle` on
  variant a mints no id. The API cannot repair a document already published.
  The route for an existing document is a fresh publish from its note.
- The variant f document is still unrefreshed. The one step this session could
  not take is the click in the Docs UI, because the browser pane here is not
  signed in. That click is Nail's gate below.

## Nail's gate, passed

Nail opened variant f on 2026-10-16, refreshed the contents list and checked
it: an entry names its heading and opens it, no bookmark flag sits beside any
heading, and the `#` jump lands. The plan below rests on that.

## The fix

Every heading gdoc writes carries one empty bookmark before its first run,
named `_h_<slug>`, and a `#` link in the note anchors at that name. The `h_`
wrapping bookmark goes. Nothing else in the docx changes.

- `bookmarkName` keeps its rule (letters, digits, underscores, at most 40
  characters, a seven-digit hash past the ceiling) and opens with `_h_`.
  Word accepts a leading underscore because its own hidden bookmarks carry
  one.
- A figure-only heading and a heading inside a footnote definition still
  carry none, because they emit no heading paragraph.
- The pre-walk still collects every heading id, so a dead `#` still warns by
  line. The `linked` set stops deciding anything and goes.
- `publish` reads the document back already. It gains one fact,
  `headings_without_id`, and a warning when that count is not zero. That is
  the check that would have caught this a month ago.

Out of scope: refreshing the contents list from gdoc. The Docs API has no
request that builds or refreshes one, so the refresh stays a click in the UI,
as the guide already says. Also out of scope: repairing the documents already
published. The API cannot, and the note is the source for a fresh publish.

## Development approach

TDD, the failing test first in every task. `make test` keeps the race flag.
`GDOC_LIVE_TEST` and `GDOC_LIVE_WRITE` are set in the ralphex environment, so a
plain `make test` there writes to Drive; the live test in Task 5 creates only
in the test folder.

## Implementation steps

### Task 1: Write the measurement and the decision down first

**Files:**
- Modify: `docs/v2/MEASURED.md`
- Modify: `docs/v2/DECISIONS.md`

- [ ] add a section "Drive's import of a heading's bookmark" to
      `docs/v2/MEASURED.md` with the two tables above, the probe document ids,
      the `updateParagraphStyle` result and the date
- [ ] add a 2026-10-16 entry to `docs/v2/DECISIONS.md`: every heading carries
      one hidden bookmark, because Drive's import makes it the heading id, and
      a `#` link anchors at it. Name what it retires: the first half of
      2026-09-27 (a bookmark only on a linked heading, and the flag reasoning,
      which held only for a visible bookmark). The second half, a relative link
      is its words, holds
- [ ] update the register: the 2026-09-27 row becomes
      `superseded 2026-10-16 (the bookmark half only; a relative link is its words holds)`,
      and the new row is added
- [ ] run `make test`, because `go/boundary` reads these documents

### Task 2: Every heading carries one hidden bookmark

**Files:**
- Modify: `go/internal/body/body.go` (`bookmarkPrefix`, `headingBookmark`, the `linked` field and `linkedAnchors`)
- Modify: `go/internal/body/build.go` (`heading`: the empty pair before the runs)
- Modify: `go/internal/body/anchors_test.go`

- [ ] write `TestEveryHeadingCarriesAHiddenBookmark`: three headings, no `#`
      link, each heading paragraph holds one `w:bookmarkStart` named
      `_h_<slug>` and its `w:bookmarkEnd` directly after it, before the first
      `w:r`; it fails today
- [ ] rewrite `TestAnAnchorLinkIsAJumpAndNotARelationship` to expect
      `w:anchor="_h_scope"`; rewrite `TestBookmarkNameIsWordSafe` for the
      `_h_` opening and the 40-character ceiling with it
- [ ] delete `TestOnlyALinkedHeadingCarriesABookmark` and
      `TestANoteWithNoAnchorLinkCarriesNoBookmark`, whose claim the decision
      retired; keep `TestAFigureOnlyHeadingCarriesNoBookmark` and
      `TestAHeadingInsideAFootnoteIsNoAnchor` and make sure they still pass
- [ ] change `bookmarkPrefix` to `_h_`, make `headingBookmark` return the name
      for every heading goldmark gave an id, drop `linked` and `linkedAnchors`,
      and write the pair in `heading` as an empty pair before the runs
- [ ] run the body and render tests, then `make test`

### Task 3: The rules in the doc comments

**Files:**
- Modify: `go/internal/body/doc.go`
- Modify: `go/internal/render/doc.go`

- [ ] rewrite "An anchor link is a jump to a bookmark on the heading it names"
      in `go/internal/body/doc.go`: every heading carries a hidden bookmark,
      why (Drive's import mints the heading id from it, Word hides it), what a
      `#` link anchors at, and the tests from Task 2 by name
- [ ] add one paragraph to "The contents list is a Word field" in
      `go/internal/render/doc.go`: an entry opens its heading only because of
      the hidden bookmark the body writes, naming the body test
- [ ] run `make test`, because `TestEveryPackageHasExactlyOnePackageComment`
      and the task map tests read these

### Task 4: The publish read-back counts headings without an id

**Files:**
- Modify: `go/internal/publish/publish.go`
- Modify: `go/internal/publish/doc.go`
- Create: `go/internal/publish/testdata/published-no-heading-ids.json`
- Modify: `go/internal/publish/publish_test.go`

- [ ] write `TestAPublishReportsHeadingsThatCameBackWithNoId`: a read-back
      where two of three headings carry no `headingId` reports
      `headings_without_id: 2` and one warning naming the count; a read-back
      where all carry one reports 0 and no warning
- [ ] add the count to the report from the inline read-back, and the warning
      when it is not zero; it is a fact and a warning, not a failure, because
      the document is there
- [ ] write the rule into `go/internal/publish/doc.go` with the test's name
- [ ] run the publish tests, then `make test`

### Task 5: The live test pins what Drive does

**Files:**
- Modify: `go/internal/live/publish_test.go` (or the live file that already publishes)
- Modify: `go/internal/live/doc.go`

- [ ] write `TestAPublishedHeadingCarriesAnIdAndAJumpIsAHeadingLink` under
      `GDOC_LIVE_WRITE`: publish a note with three headings and one `#` link
      into the test folder, read it back, every heading carries a `headingId`,
      the link came back as a heading link naming the right heading's id, and
      the report says `headings_without_id: 0`
- [ ] name the test and its variables in `go/internal/live/doc.go`
- [ ] run it once by hand with the variables set, and leave the document in
      the test folder

### Task 6: The guide, the skill and the help say what a reader gets

**Files:**
- Modify: `docs/guide/publishing.md`
- Modify: `skills/gdoc-publish/SKILL.md`
- Modify: `go/cmd/gdoc/doc.go` or the command table, if `publish` gains a note

- [ ] in `docs/guide/publishing.md`, the contents paragraph says that after the
      refresh each entry opens its heading, and names `headings_without_id` as
      the read-back fact that says the import kept the ids
- [ ] in `skills/gdoc-publish/SKILL.md`, the line about the contents list says
      the same, and the skill reads `headings_without_id` as a reason to stop
      and say so rather than to retry
- [ ] run `make test`, because the embedded skill core and the task map are
      pinned

### Task 7: Verify acceptance criteria

- [ ] `make vet` and `make test` pass
- [ ] `make build`, build the probe note from this plan, unzip the docx and
      confirm one `_h_` pair per heading and `w:anchor="_h_who_decides"`
- [ ] publish that note into the test folder, read it back through
      `gdoc read --structure`, and confirm the report says
      `headings_without_id: 0`
- [ ] Nail refreshes the contents list on that document and confirms the hover
      card names the heading, as on variant f

### Task 8: Update documentation and close

- [ ] `README.md` needs nothing unless it describes the contents list
- [ ] `CLAUDE.md`: no invariant changes; check the task map still names files
      that exist
- [ ] move this plan to `docs/plans/completed/`

## Post-completion

**Nail's documents.** Every document gdoc published so far has the bug, and the
API cannot repair one in place. Nail's decision of 2026-10-16: they stay as they
are. A document that needs working links gets a fresh publish from its note,
which creates a new document; nothing in this plan goes back for the old ones.

**Release.** After the merge into `main`, trigger the nightly
(`gh workflow run nightly.yml --ref main`) and report the tag, as after every
merge.

**Probe documents.** The six `toc-headingid-probe-*` documents stay in the test
folder beside the September `toc-font-experiment-*` ones, as the evidence the
MEASURED.md section points at.
