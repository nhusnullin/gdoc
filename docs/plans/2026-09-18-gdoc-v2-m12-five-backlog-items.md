# gdoc v2 Milestone 12: five backlog items

2026-09-18. Five items from `docs/backlog/` whose value was already agreed
(`worth: yes`), each with one design choice left, and Nail took each choice on
2026-09-18 in the session that wrote this plan. Nothing here opens a new door,
reaches a new host or adds a command. Two items narrow what the binary does
(the guard grant, the apply loop), one makes a test stricter (the drift pins),
and two fix the docx the generator writes (numbered lists, anchor links).

## Principles

Serves: 3, uncertainty never resolves toward the destructive answer. Task 2 is
the whole of that principle: a restyle whose batch answer names no revision
used to read the document for one and carry on, adopting whatever a colleague
typed in the gap; now it stops and says the document is half styled. Task 1
holds the guard's own rule that the levels are names and not a ladder, so a
grant can never take reach away in silence. Task 3 turns an allowlist of names
into an allowlist of measured values, so the gate that protects the house style
can no longer pass a row it stopped checking. Tasks 4 and 5 serve 1, the
document reads as the author wrote it: a second numbered list starts at 1, and
`[see below](#scope)` jumps.

Strains: nothing. Every task is offline. No live document is read or written by
any task, and no test needs a live variable.

## Overview

| Piece | Today | After |
|---|---|---|
| `Policy.GrantInPlace` on an id at `LevelFull` | writes `LevelInPlace` over it: the Drive `PATCH` goes and the styling allowlist starts to bind, silently | leaves the id at `LevelFull` and puts a warning on the envelope naming the id and the level |
| a restyle batch answer with no `requiredRevisionId`, mid-run | reads the document's current revision and sends the next batch against it | stops the run, `RevisionUnconfirmed` set, the half-styled sentence in the warnings, `ok: false` |
| `drift.Known` in the offline gate | a row named in it passes on any difference, in either direction | a row named in it must still read the pair of values that was measured, or the gate fails naming both |
| two numbered lists in a note | one `w:num`, so the second prints 4, 5, 6 | one `w:num` per numbered list, each starting at 1 |
| `[x](#scope)` in a note | an external relationship to `#scope`; Word opens nothing | a `w:hyperlink w:anchor` to a bookmark every heading with words now carries; a `#` link naming no heading is warned about and printed as plain text |

## Decisions Nail took, 2026-09-18

Each closes the open question its backlog file states. Task 6 writes them into
DECISIONS.md as one entry with one register row. A task that finds one wrong
stops and says so.

1. **The grant never narrows.** `GrantInPlace` on an id already at
   `LevelFull` leaves it there and records through `p.note` that the grant
   changed nothing, naming the id and the level. It is the friendlier of the
   two shapes the backlog offered, and it keeps the levels names rather than
   rungs: nothing is compared with `<` or `>`, the one case is matched with
   `==` on `LevelFull`. A second grant on an id already at `LevelInPlace`
   changes nothing and says nothing.
2. **The apply loop stops on a quiet answer.** An answer that names no
   revision, with a batch still to send, ends the run the way a refused batch
   does: the batches that landed stay, `leftBehind` says the document is half
   styled, and the error names the batch. The loop never reads the document
   between batches. `RevisionOf` stays exported for its one remaining caller,
   the prelude run in `cmd/gdoc/restyle.go`, whose own comment already says
   what that read costs at the phase boundary; that caller is not this
   milestone's.
3. **The offline gate pins the measured pair, and the live gate is left
   alone.** `Known` stays a map of name to reason, shared by both gates. The
   offline gate gains a second map, `knownOffline`, in a test file, from name
   to the two values as `show` prints them, every one a literal. A `Known` row
   whose values move fails until somebody re-records the pair with its reason,
   which is the golden-file discipline the gate already has. The live gate is
   unchanged because no person has read its rows yet (PLAN.md, "Outstanding by
   hand"), so there is no measured pair to pin there.
4. **One `w:num` per numbered list, and the body says how many.** The body
   walker hands out a fresh list id to every ordered list that is not nested
   inside an ordered list, counts them on `Result`, and `render.Build` writes
   that many `w:num` entries, all pointing at the one numbered abstract list.
   A nested ordered list inside an ordered list names its parent's id, so
   level restarts work as they do today. The "carries on from the one above"
   warning goes, because it is no longer true. The "starts at N in the note
   and at 1 in the document" warning stays: honouring the author's start
   number is a `w:startOverride` per list and it is not in this milestone.
5. **Bookmarks on every heading, and `w:anchor` for a `#` link.** Every
   heading with words gets a `w:bookmarkStart`/`w:bookmarkEnd` pair around its
   runs, named from goldmark's auto heading id through one function that turns
   the id into a name Word accepts: `h_` and then the id with every character
   outside `[A-Za-z0-9_]` replaced by `_`, and when that name is longer than
   40 characters, Word's ceiling on a bookmark name, the first 32 characters
   of that name, then `_`, then the first 7 hex digits of the FNV-1a 32-bit
   hash of the goldmark id as it arrived. goldmark's generator only ever emits `[a-z0-9-]`, so the rule is
   deliberately wider than what it will meet; a heading written entirely in
   characters it drops gets the id `heading`, then `heading-1`, and its
   bookmark is `h_heading`. A link whose destination opens with
   `#` becomes `<w:hyperlink w:anchor="...">` with no relationship and no
   `Media` entry, through the same function. A `#` link naming no heading in
   the note is warned about naming the line, and its words are printed as
   plain text, which is how a code block is already refused. The walk collects
   the heading ids before it renders, so a link to a heading further down
   resolves, and it collects only the headings that will carry a bookmark: a
   figure-only heading emits no paragraph, so a link to it is a dead anchor
   and is warned about like any other.

## Context (from discovery)

- **The grant.** `Policy.GrantInPlace` at `go/internal/guard/policy.go:209`
  is six lines: lock, and if the id is known write `LevelInPlace`. `AllowFile`
  one function up refuses the analogous mistake through `p.note`, which takes
  the lock itself, so a note must be written before `p.mu.Lock()` and never
  under it. `p.level(id)` at `policy.go:406` reads a level under `RLock`.
  `judgeDrive` at `:763` carries `PATCH` on the Drive file at `LevelFull`
  alone; `judgeRequests` at `:1062` enforces `inPlaceKinds` at `LevelInPlace`
  alone (`:1128`). The tests live in `go/internal/guard/inplace_test.go`, with
  `granted(t)` at `:18` as the helper and `TestAllowFileCannotOpenTheInPlaceDoor`
  at `:284` as the shape a warning test takes. The doc paragraph is
  `go/internal/guard/doc.go:125`. Production callers: `cmd/gdoc/restyle.go:388`
  and `:498`, both on ids handed in at `LevelSuggest`, so no command reaches
  the new branch today; `internal/live/restyle_test.go:25` explains the
  two-policy workaround this item costs the live test, and that test is left
  as it is because its reason (exercising the allowlist) still holds.
- **The apply loop.** `apply` in `go/internal/restyle/apply.go:194`. The
  fallback is the `next == ""` branch at `:235`: the last batch takes
  `RevisionUnconfirmedWarning` and the flag and breaks; any other batch calls
  `RevisionOf` at `:246` and carries on. `out.leftBehind(total, false)` at
  `:324` is the half-styled sentence and `out.stopped` at `:262` is the shape
  of a report for a batch that failed; this case is neither stale nor lost, so
  it does not go through `stopped`. `RevisionOf` at `:472` carries a comment
  that already describes the gap and names the backlog file. The tests in
  `apply_test.go`: `TestTheLoopReadsForTheRevisionWhenAnAnswerCarriesNone` at
  `:359` and `TestAFailedRevisionReadStopsTheRun` at `:390` pin the old
  behaviour and go; `TestTheLastAnswerCarryingNoRevisionIsAWarningAndNotARead`
  at `:413` pins the last-batch case and stays. The `scripted` session at
  `:25` counts reads in `reads`. The package doc paragraph is
  `go/internal/restyle/doc.go:142`, "The loop reads between batches". The
  other caller of `RevisionOf` is `cmd/gdoc/restyle.go:581`. PRINCIPLES.md
  line 104 states the no-retry rule this extends; SPEC.md line 204 describes
  the styling run.
- **The drift gate.** `Known` at `go/internal/drift/compare.go:94`, 25
  entries; `Unexplained` at `:306`; `show` at `:326` prints a string with `%q`
  and anything else with `%v`. `TestTheOfflineGate` at `drift_test.go:185`,
  `TestEveryKnownDifferenceStillDiffers` at `:206`, and in `compare_test.go`
  `TestEveryKnownDifferenceCarriesItsReason` at `:115` and
  `TestEveryKnownDifferenceNamesARealItem` at `:125`. The live gate is
  `internal/live/publish_test.go:227`, on `drift.Unexplained` alone. The
  values the offline gate reads today, printed once with a throwaway test on
  2026-09-18, in `show` form (the task re-derives them the same way rather
  than copying from here):

  | Row | A (built) | B (master) |
  |---|---|---|
  | `HEADING_1 colour` | `"#22265F"` | `"#06436E"` |
  | `HEADING_2 colour` | `"#22265F"` | `"#06436E"` |
  | `HEADING_1 indentStart` | `0` | `36` |
  | `HEADING_2 indentStart` | `0` | `72` |
  | `HEADING_3 indentStart` | `0` | `108` |
  | `HEADING_4 indentStart` | `0` | `144` |
  | `HEADING_5 indentStart` | `0` | `180` |
  | `HEADING_6 indentStart` | `0` | `216` |
  | `default header text` | `"Altery - Supplier Management Policy"` | `"Altery - xxx Policy"` |
  | `table count` | `6` | `3` |
  | `table Revision History size` | `"2x7"` | `"3x7"` |
  | `table Revision History row heights` | `[32.4 28.35]` | `[32.4 28.35 28.35]` |
  | `body H1 text` | `"1-Third Party and Outsourcing Policy"` | `"1-Purpose"` |
  | `body H1 run colour` | `"#22265F"` | `"#222660"` |
  | `body H2 text` | `"1.1-Purpose and scope"` | `"Appendix 1 – Associated Documents"` |
  | `body H2 run colour` | `"#22265F"` | `"#222660"` |
  | `body H3 text` | `"1.1.1-What counts as outsourcing"` | `<none>` |
  | `body H3 indentStart` | `0` | `<none>` |
  | `body H3 run colour` | `"#549F99"` | `<none>` |
  | `bulleted paragraphs` | `17` | `27` |

  The five cell rows (`table Version Control cell text`, `table Revision
  History cell fills`, `table Revision History cell text`, `table Revision
  History cell borders`, `table Document Classification cell fills`) are long
  strings joined with `\x1f`; they are
  pinned the same way, as the literal `show` prints. `Bullets` in `docx.go:828`
  counts every paragraph carrying `w:numPr` whatever its `numId`, so Task 4
  does not move the `17`.
- **Numbered lists.** `render.numberingPart` at
  `go/internal/render/numbering.go:19` writes two abstract lists and two
  `w:num`, `BulletNumID = "1"` and `NumberNumID = "2"` at `:41`, exported for
  the body. `render.Build` at `render.go:106` takes `(cfg, fields, body,
  media)` and has three callers: `cmd/gdoc/build.go:155`,
  `internal/drift/drift_test.go:306` (the offline gate building the note) and
  `internal/live/publish_test.go:333`, each with `walked` in scope. In the body,
  `block` at `go/internal/body/body.go:441` picks the id per `ast.List` and
  calls `warnListNumbers` at `:427`, which counts `numberedLists` at `:80` and
  warns twice; `itemBlocks(item, level, numID string)` at `:368` carries the
  id down and `listCtx` at `:336` holds one field, `numID`; `listItem` at
  `build.go:327` writes `w:numPr`; `body.go:398` and `:631` compare an id with
  `render.NumberNumID` to know a list is numbered. `Result` at `body.go:51`
  and `Counts` at `:34`. Tests: `TestANumberedListNamesTheNumberedList` at
  `body_test.go:294`, `TestANumberedListWhoseNumbersAreNotTheAuthorsSaysSo` at
  `:1052`, and `TestNumberingHoldsTheBulletedAndTheNumberedList` at
  `render/numbering_test.go:21`. The doc section is `body/doc.go:159`. The
  guide paragraph is `docs/guide/publishing.md:75`. `body_test.go` is 1171
  lines, so new body tests go in a new file.
- **Anchor links.** `addRuns` at `go/internal/body/build.go:257` has six
  callers, `build.go:303`, `:323`, `:356`, `:372`, `:400` and `table.go:139`,
  none of which carries a line number; it wraps a run
  with a `Link` in `w:hyperlink r:id` through `r.linkID(url)` at
  `body.go:199`, which records a `Media` with `Target`; `render.documentRels`
  at `render.go:296` writes `TargetMode="External"` for every `Media` that
  `IsLink()`. `headingBlock` at `body.go:541` builds the runs, merges the
  number prefix in, and emits `r.heading(...)` at `:578`; the figure-only
  heading returns early at `:547`. The parser is built with
  `parser.WithAutoHeadingID()` at `body.go:167`, so every heading node carries
  its id as the `id` attribute (`heading.AttributeString("id")` gives it as
  `[]byte`), unique across the file. Tests:
  `TestALinkIsAHyperlinkWithARelationship` at `body_test.go:323` states the
  current shape and stays true for an `https` link;
  `TestAnEmailAutolinkCarriesTheMailtoScheme` at `inline_test.go:181` is the
  neighbouring case. The doc section is `body/doc.go:148`, whose last sentence
  names the backlog file. Word's own bookmark rule is letters, digits and
  underscores, opening with a letter, at most 40 characters; the docx export reader in
  `internal/docx` and the drift docx reader both read runs and ignore a
  bookmark element, and `TestTheOfflineGate` says so when Task 5 runs.
- **Boundary tests** that must stay green: `TestEveryPackageHasExactlyOnePackageComment`,
  `TestOnlyThePackageCommentReachesGoDoc`, `TestTheTaskMapNamesFilesThatExist`,
  `TestCLAUDEmdIsUnderTheCeiling`, `TestNothingRunsAnExternalProgram`, and in
  `guard/` every test with no assertion changed except the two this plan
  names.
- **The documents.** DECISIONS.md's register at line 15 and its last entry at
  line 2198, the annotate entry, which is the shape to copy; PRINCIPLES.md lines 102 to 107; SPEC.md line
  204; `docs/guide/publishing.md:75`; PLAN.md's Done table; the five backlog
  files, each removed in the commit that closes it, never in a cleanup commit.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it
  to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own
  commit, and each ends with the test gate as its last checkbox.
- **CRITICAL: every task MUST include new or updated tests**, success and
  failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** `make test`
  runs `-race`; keep it there.
- **CRITICAL: the guard moves in one function.** Task 1 changes
  `GrantInPlace` and nothing else under `guard/`. No new grant, no new route,
  no level compared with anything but `==`. Every existing guard test stays
  green without an assertion changed.
- **CRITICAL: the apply loop reads nothing between batches.** After Task 2,
  `RevisionOf` has one caller and it is in `cmd/gdoc`.
- **CRITICAL: a test states its value as a literal.** The drift pins are
  literals, never read from `house.yaml` or computed from the master; the
  bookmark name in a test is spelled out.
- **CRITICAL: the backlog file is `git rm`'d in the commit that lands its
  fix**, one file per task for Tasks 1 to 5.
- **CRITICAL: facts only in Go.** A warning says what happened, never whether
  it matters.
- **CRITICAL: new tests for the body go in new files.** `body_test.go` is
  over the ceiling already.
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces. Plain English.

## Testing Strategy

- **Unit, `guard`**: a `LevelFull` id survives the grant, its Drive `PATCH`
  is still carried, and one warning names the id and the level; a second
  grant on a `LevelInPlace` id changes nothing and warns nothing; every
  existing pin in `inplace_test.go` unchanged.
- **Unit, `restyle`**: a quiet answer with a batch still to send stops the
  run with no read, one POST, `Batches` 1, `RevisionID` the one that batch
  was sent against, `RevisionUnconfirmed` true, `Stale` and `MaybeApplied`
  false, the half-styled sentence in the warnings and an error naming the
  batch; the last-batch case unchanged.
- **Unit, `drift`**: every `Known` name has a pinned pair and every pinned
  pair names a `Known` entry; the offline gate fails on a `Known` row whose
  values moved, naming the row and both values; the gate is green on the
  tree as it stands.
- **Unit, `body` and `render`**: two top-level numbered lists name two ids;
  a nested numbered list names its parent's; a numbered list under a bullet
  opens its own; `Result.NumberedLists` counts them; the numbering part
  writes one `w:num` per list plus the bullet one, all on the numbered
  abstract list, and none when the body has no numbered list; the "carries
  on" warning is gone and the "starts at" warning stays. A heading carries a
  bookmark named from its id; a `#` link is a `w:anchor` with no relationship
  and no `Media`; a `#` link to a heading below it resolves; a `#` link to no
  heading warns naming the line and prints plain text; an `https` link is
  unchanged; a figure-only heading carries no bookmark; the bookmark name
  function is pinned on literals including a hyphen, a dot and a leading
  digit.
- **Boundary**: all green.
- **Offline drift gate**: green after Task 4 and after Task 5 with no pin
  changed, which is the proof neither moved a measured value.
- Coverage standard: every changed function has a test; no package below 80%.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && test -z "$(gofmt -l .)" && go vet ./...`
- `make build`, then build a scratch note (not committed) that copies
  `release/example/first-note.md`'s front matter and holds three headings,
  two top-level numbered lists and a `[see below](#the-third-heading)` link
  above the third heading:
  `bin/gdoc build /tmp/m12-note.md --out /tmp/m12.docx --force`. The parts are
  written on one line each, so count matches and not lines:
  `unzip -p /tmp/m12.docx word/numbering.xml | grep -o '<w:num ' | wc -l`
  prints 3, and `unzip -p /tmp/m12.docx word/document.xml | grep -o 'w:bookmarkStart' | wc -l`
  prints 3, and `grep -o 'w:anchor="[^"]*"'` on the same part prints one name
- `ls docs/backlog/ | wc -l` prints 18, five fewer than the 23 on main
- `wc -l CLAUDE.md` under 300

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

**Task 1** is one early return in `GrantInPlace`: read the level through
`p.level`, and when it is `LevelFull`, `p.note` and return before the lock.

**Task 2** deletes the read from the `next == ""` branch: the warning and the
flag go on for every quiet answer; the last batch breaks as today; any other
batch appends `leftBehind` and returns an error naming the batch.

**Task 3** adds `knownOffline` in a new `pins_test.go`, a map from name to a
`pair{A, B string}` of `show` literals, and makes `TestTheOfflineGate` check
every `Known` row against it, plus a two-direction test between the two maps.

**Task 4** turns `render.NumberNumID` into `render.NumberNumID(n int) string`
returning the id of the n-th numbered list (1-based, `"2"` for the first, so
today's documents are unchanged), gives `render.Build` a `numberedLists int`
argument, and has the body hand out ids from its `numberedLists` counter. The
arithmetic lives in `render` beside `numberingPart`, which writes the matching
`w:num` entries, so the two cannot drift; that is the reason the exported
constant becomes a function. The count goes on `body.Result`, not on
`Counts`: it is plumbing for `render.Build`, not a fact for a reader, and the
envelope `gdoc build` prints does not change.

**Task 5** adds `bookmarkName(id string) string` in the body, a pre-walk over
`ast.Heading` nodes collecting the ids of the headings that will carry a
bookmark, a bookmark pair around a heading's runs, and a `#` branch in
`addRuns`. The dead-anchor warning needs a line, and `addRuns` has six callers
with none in hand, so the renderer gets a `curLine int` field set once at the
top of `headingBlock`, `paragraphBlock` and the `*east.Table` case of `block`
(`r.line(typed)` of the table), and `addRuns` reads it; no signature changes.

## Technical Details

**The warning Task 1 writes**: `GrantInPlace on %q changed nothing: the id is
at the full level, which a create the guard carried gave it, and the in-place
level would narrow it`.

**The error Task 2 returns**: `batch %d of %d was accepted and its answer named
no revision id, so the run stopped rather than sending the next batch against
a revision read from the document, which could carry somebody else's edit`.

**The pin shape Task 3 writes**, in `pins_test.go`:

```go
type pair struct{ A, B string }

var knownOffline = map[string]pair{
	"HEADING_1 colour":      {`"#22265F"`, `"#06436E"`},
	"HEADING_1 indentStart": {`0`, `36`},
	// ... one line per Known entry
}
```

and the gate's new check, for each row whose name is in `Known`: `show(r.A)`
and `show(r.B)` equal the pinned pair, else
`t.Errorf("known difference %q now reads %s | %s, and the pinned pair is %s |
%s: re-record it in knownOffline with the reason, or fix what moved", ...)`.

**The numbering part Task 4 writes**, for a body with two numbered lists:

```xml
<w:num w:numId="1"><w:abstractNumId w:val="1"/></w:num>
<w:num w:numId="2"><w:abstractNumId w:val="2"/></w:num>
<w:num w:numId="3"><w:abstractNumId w:val="2"/></w:num>
```

**The bookmark Task 5 writes**, around a heading's runs, with `w:id` counting
from 0 across the document:

```xml
<w:p><w:pPr>...</w:pPr>
  <w:bookmarkStart w:id="0" w:name="h_purpose_and_scope"/>
  <w:r>...</w:r>
  <w:bookmarkEnd w:id="0"/>
</w:p>
```

and the link: `<w:hyperlink w:anchor="h_purpose_and_scope">` with the run
inside it coloured and underlined as any link is.

**The warning for a dead anchor**: `line %d: the link to #%s names no heading
in this note, so its words are printed as plain text`.

**The mid-run warning Task 2 keeps**: `RevisionUnconfirmedWarning` goes on the
report for the mid-run stop as it does for the last batch. Its text, "the last
batch was accepted and its answer named no revision id, so the revision
reported here is the one that batch was sent against", is true on both paths:
the last batch sent is the one whose answer went quiet, and `RevisionID` is
what it was sent against. It stays a constant, and the two `dropWarning` sites
in `cmd/gdoc/restyle.go` (`:487`, `:593`) are unaffected, because both sit on
paths that only run after a prelude phase that returned no error.

## Implementation Steps

### Task 1: the grant never narrows

**Files:**
- Modify: `go/internal/guard/policy.go`, `go/internal/guard/inplace_test.go`,
  `go/internal/guard/doc.go`
- Remove: `docs/backlog/grantinplace-silently-demotes-a-created-document.md`

- [ ] Test first, `TestGrantInPlaceLeavesACreatedDocumentAtFull`: `AllowFile("MADE",
      LevelFull)`, `GrantInPlace("MADE")`; the level is still `LevelFull`, a
      `PATCH` on `https://www.googleapis.com/drive/v3/files/MADE` with
      `{"trashed":true}` is carried, and `Warnings()` is one entry naming
      `MADE` and the word `full`. Watch it fail.
- [ ] Test, `TestASecondInPlaceGrantIsQuiet`: `granted(t)` then
      `GrantInPlace("DOC1")` again; level unchanged, no warning.
- [ ] `GrantInPlace`: read `p.level(id)`; when known and `LevelFull`, `p.note`
      the sentence in Technical Details and return; otherwise as today. The
      note is written outside the lock.
- [ ] The function's comment says why the grant never narrows and names both
      tests; `doc.go:125` gains one sentence naming the first.
- [ ] `cd go && go test -race ./internal/guard/` passes with no other
      assertion changed;
      `git diff main -- go/internal/guard/policy.go go/internal/guard/doc.go | grep '^-' | grep -v '^---' | grep -v '^-\s*//'`
      is empty: nothing left the guard but comment lines.
- [ ] `git rm docs/backlog/grantinplace-silently-demotes-a-created-document.md`
- [ ] `git commit -m "fix(guard): GrantInPlace leaves a created document at full"`

### Task 2: a quiet answer mid-run stops the loop

**Files:**
- Modify: `go/internal/restyle/apply.go`, `go/internal/restyle/apply_test.go`,
  `go/internal/restyle/doc.go`
- Remove: `docs/backlog/restyle-revision-fallback-breaks-the-chain.md`

- [ ] Test first, `TestAnAnswerCarryingNoRevisionMidRunStopsTheRun`: three
      one-request batches, answers `{"", "rev3", "rev4"}`, `readRevision`
      `"rev2"` available; the run returns an error containing `batch 1 of 3`
      and `revision`; `s.reads` is 0; one POST; `Batches` 1; `RevisionID`
      `"rev1"`; `RevisionUnconfirmed` true; `Stale` and `MaybeApplied` false;
      the joined warnings contain `half styled`. Watch it fail.
- [ ] Delete `TestTheLoopReadsForTheRevisionWhenAnAnswerCarriesNone` and
      `TestAFailedRevisionReadStopsTheRun`; keep
      `TestTheLastAnswerCarryingNoRevisionIsAWarningAndNotARead` unchanged.
      In `scripted`, drop `readRevision` and `readErr`, keep `reads`, and have
      `GetJSON` count the read and return an error saying the loop must not
      read; shrink the field comment to match.
- [ ] `apply`: in the `next == ""` branch, append `RevisionUnconfirmedWarning`
      and set the flag for every quiet answer; if it is the last batch, break;
      otherwise append `out.leftBehind(len(batches), false)` and return the
      error in Technical Details. The `RevisionOf` call leaves the loop.
- [ ] `RevisionOf`'s comment: it is no longer called by the loop, it names the
      new test, and it names its one caller in `cmd/gdoc`. `doc.go:142`: the
      paragraph "The loop reads between batches" becomes "The loop reads
      nothing between batches", says why, and names the two pins.
- [ ] `grep -rn 'RevisionOf(' go/` shows the definition and
      `cmd/gdoc/restyle.go` only.
- [ ] `cd go && go test -race ./internal/restyle/ ./cmd/...` passes.
- [ ] `git rm docs/backlog/restyle-revision-fallback-breaks-the-chain.md`
- [ ] `git commit -m "fix(restyle): a quiet answer mid-run stops the loop instead of reading the revision"`

### Task 3: the offline gate pins the measured pair

**Files:**
- Create: `go/internal/drift/pins_test.go`
- Modify: `go/internal/drift/drift_test.go`, `go/internal/drift/doc.go`,
  `go/internal/drift/compare.go` (the `Known` comment only)
- Remove: `docs/backlog/drift-known-pins-no-value.md`

- [ ] Print the current pair for every `Known` row once, with a throwaway
      test that logs `show(r.A)` and `show(r.B)` for the rows of
      `Compare(FromDocx(buildNote(t)), FromDocx(openMaster(t)))` whose name is
      in `Known`, and delete the throwaway before committing. The table in
      Context is what it printed on 2026-09-18; use what it prints now.
- [ ] Test first, `TestEveryKnownDifferenceHasItsPairPinned` in
      `pins_test.go`: every name in `Known` is a key of `knownOffline` and
      every key of `knownOffline` is a name in `Known`. With an empty
      `knownOffline` it fails 25 times. Watch it fail.
- [ ] Test, `TestAKnownRowWhoseValuesMovedFailsTheGate`: a row list built by
      hand with one `Known` name and values that differ from its pin goes
      through the same check the gate uses (extract it as
      `unpinned(rows) []string` in `pins_test.go`, returning one sentence per
      row) and comes back with one sentence naming the row, both values and
      the pin. A row matching its pin comes back with none. Watch it fail.
- [ ] `knownOffline`, 25 literal pairs, with a comment above it saying what a
      failure means and that re-recording is a decision written down with the
      reason, the same rule `Known` states.
- [ ] `TestTheOfflineGate` calls `unpinned(rows)` after `Unexplained` and
      fails on every sentence it returns.
- [ ] Set one pin wrong by hand, watch the gate fail naming the row, set it
      back. Change `heading_1.color` in `house.yaml` to `#FF00FF` by hand,
      watch the gate fail on `HEADING_1 colour`, set it back. Record both in
      this plan as ➕ notes.
- [ ] `compare.go`: the `Known` comment says the offline gate also pins the
      pair, in `pins_test.go`. `doc.go`'s "Known explains a difference" section
      says the same and names the two new tests.
- [ ] `cd go && go test -race ./internal/drift/` passes.
- [ ] `git rm docs/backlog/drift-known-pins-no-value.md`
- [ ] `git commit -m "test(drift): the offline gate pins the measured pair of every known difference"`

### Task 4: one numbered list definition per numbered list

**Files:**
- Modify: `go/internal/render/numbering.go`, `go/internal/render/render.go`,
  `go/internal/render/numbering_test.go`, `go/internal/body/body.go`,
  `go/internal/body/build.go`, `go/internal/body/doc.go`,
  `go/internal/body/body_test.go` (the two existing tests only),
  `go/cmd/gdoc/build.go`, `go/internal/drift/drift_test.go:306` and
  `go/internal/live/publish_test.go:333` (the `Build` call only, which is
  plumbing and not a pin change), `docs/guide/publishing.md`
- Create: `go/internal/body/lists_test.go`
- Remove: `docs/backlog/one-numbered-list-per-document.md`

- [ ] Test first, in `render/numbering_test.go`,
      `TestOneNumberedListDefinitionPerNumberedList`: `Build` with
      `numberedLists` 2 writes `w:num` ids `1`, `2`, `3`, the first on the
      bullet abstract list and the other two on the numbered one; with 0 it
      writes `1` alone; `NumberNumID(1)` is the literal `"2"` and
      `NumberNumID(3)` is `"4"`. Watch it fail to compile.
- [ ] Test, in `body/lists_test.go`, `TestASecondNumberedListStartsAgain`:
      `"1. a\n2. b\n\ntext\n\n1. c\n2. d\n"` names `w:numId` `2` on the first
      two items and `3` on the last two, `Result.NumberedLists` is 2, and no
      warning says "carries on".
- [ ] Test, `TestANestedNumberedListNamesItsParent`: a numbered list nested in
      a numbered item names the parent's id; `NumberedLists` is 1.
- [ ] Test, `TestANumberedListUnderABulletOpensItsOwn`: a numbered list nested
      in a bullet names `2`, and a second top-level numbered list after it
      names `3`; `NumberedLists` is 2.
- [ ] Test, `TestAListThatStartsElsewhereStillSaysSo`: `"5. a\n"` still
      warns "starts at 5 in the note and at 1 in the document".
- [ ] `render`: `NumberNumID(n int) string` returning `strconv.Itoa(n + 1)`;
      `numberingPart(numberedLists int)` writing the bullet `w:num` and one per
      list; `Build(cfg, f, body, media, numberedLists int)`; the doc comment on
      the constants says why the ids are computed.
- [ ] `body`: `Result.NumberedLists int`; `listCtx` gains `ordered bool` and
      `itemBlocks` takes `(item, level, list listCtx)` instead of the bare id;
      `block` opens a fresh id when the list is ordered and the enclosing
      `listCtx` is not ordered, by incrementing `numberedLists` and calling
      `render.NumberNumID`; a nested ordered list inside an ordered one reuses
      the enclosing id; the comparison at `body.go:398` reads the ctx's
      `ordered` and the one at `:631` reads `list.ordered`, and neither
      compares an id string any more; `warnListNumbers` keeps only the start
      warning and its comment shrinks to match.
- [ ] The three `Build` callers pass `walked.NumberedLists`:
      `cmd/gdoc/build.go:155`, `internal/drift/drift_test.go:306`,
      `internal/live/publish_test.go:333`.
- [ ] `TestANumberedListNamesTheNumberedList` and
      `TestANumberedListWhoseNumbersAreNotTheAuthorsSaysSo` updated to the new
      behaviour: the first still expects `2` for a single list; the second
      drops the "carries on" case and keeps the start and nested cases.
- [ ] `body/doc.go:159`: the section becomes "A numbered list starts at 1,
      and a list that opens elsewhere says so", names the four new tests and
      no longer names the backlog file. `docs/guide/publishing.md:75`: the
      paragraph says a second numbered list starts again at 1 and keeps the
      start-number sentence.
- [ ] `cd go && go test -race ./...` passes, and `TestTheOfflineGate` is green
      with no pin changed.
- [ ] `git rm docs/backlog/one-numbered-list-per-document.md`
- [ ] `git commit -m "fix(body): every numbered list gets its own definition and starts at 1"`

### Task 5: an internal anchor link jumps

**Files:**
- Modify: `go/internal/body/body.go`, `go/internal/body/build.go`,
  `go/internal/body/doc.go`
- Create: `go/internal/body/anchors_test.go`
- Remove: `docs/backlog/internal-anchor-links.md`

- [ ] Test first, `TestBookmarkNameIsWordSafe`: `bookmarkName("purpose-and-scope")`
      is the literal `"h_purpose_and_scope"`, `bookmarkName("1-1-purpose")` is
      `"h_1_1_purpose"`, `bookmarkName("a.b")` is `"h_a_b"` (an input goldmark
      never produces, pinned because the rule is wider than the generator),
      and `bookmarkName("what-counts-as-outsourcing-and-why-it-matters")` is
      40 characters long and opens with `h_what_counts_as_outsourcing_and_`
      (the first 32 of the underscored name, then `_`) followed by 7 hex
      digits. Watch it fail to compile. Once the function exists, print the
      value once with a throwaway test, delete the throwaway, and add the
      whole 40-character literal to the test as a second assertion, so the
      hash is pinned too.
- [ ] Test, `TestEveryHeadingCarriesABookmark`: `"# Purpose and scope\n\ntext\n\n## Scope\n"`
      serialises with `w:bookmarkStart w:id="0" w:name="h_purpose_and_scope"`
      and a matching `w:bookmarkEnd w:id="0"` inside the first heading's
      paragraph, and `w:id="1"` with `h_scope` in the second.
- [ ] Test, `TestAnAnchorLinkIsAJumpAndNotARelationship`:
      `"See [below](#scope).\n\n## Scope\n"` serialises with
      `<w:hyperlink w:anchor="h_scope">`, no `r:id` on it, `len(out.Media)`
      0, and the run inside coloured and underlined. The link sits above the
      heading, which is the forward case.
- [ ] Test, `TestAnAnchorToNoHeadingWarnsAndPrintsPlainText`:
      `"See [below](#nowhere).\n"` has no `w:hyperlink`, the words `below` are
      in an ordinary run, `Media` is empty, and one warning is
      `line 1: the link to #nowhere names no heading in this note, so its words are printed as plain text`.
- [ ] Test, `TestAFigureOnlyHeadingCarriesNoBookmark`: the note
      `"See [it](#altpicpng).\n\n## ![alt](pic.png)\n"` with `pic.png` a
      one-pixel PNG in the test's temp directory: the heading serialises with
      no `w:bookmarkStart` (goldmark names it `altpicpng`, built from the raw
      line), and the link is warned about as a dead anchor to `#altpicpng`
      and printed as plain text, because no bookmark was written for it.
- [ ] `TestALinkIsAHyperlinkWithARelationship` runs unchanged: an `https` link
      is as it was.
- [ ] `body`: `bookmarkName`, with `hash/fnv` for the long case; extract
      the figure-only test `headingBlock` already makes at `body.go:547` into
      `isFigureOnly(heading, source) bool` and use it in both places; a
      pre-walk in `Render` (before the block walk) over `ast.Heading` nodes
      that are not figure-only, reading `AttributeString("id")` into a set on
      the renderer; a `bookmarkID int` counter; a `curLine int` field on the
      renderer set at the top of `headingBlock`, `paragraphBlock` and the
      `*east.Table` case of `block` at `body.go:470`, where the node is in
      hand (`table.go` needs no edit); `heading` (in `build.go`) takes the name and
      writes the pair around the runs when the name is not empty; `addRuns`
      takes the `#` branch: strip `#`, look the id up in the set, and either
      `w:hyperlink w:anchor` or the plain run plus the warning at `curLine`.
      No caller of `addRuns` changes.
- [ ] `body/doc.go`: the autolink section's last sentence no longer names the
      backlog file, and a new section "An anchor link is a jump to a bookmark
      every heading with words carries" says why the name is rewritten, why a dead anchor
      is plain text rather than a broken jump, and names the five tests.
- [ ] `cd go && go test -race ./...` passes, and `TestTheOfflineGate` is green
      with no pin changed: a bookmark carries no text and no measured value.
- [ ] `git rm docs/backlog/internal-anchor-links.md`
- [ ] `git commit -m "fix(body): an internal anchor link jumps to a bookmark on the heading it names"`

### Task 6: the documents

**Files:**
- Modify: `docs/v2/DECISIONS.md`, `PRINCIPLES.md`, `docs/v2/SPEC.md`,
  `docs/v2/PLAN.md`, `CLAUDE.md` (only if a check below says so)

- [ ] DECISIONS.md: one entry, `## 2026-09-18. Five backlog items closed, and
      what each decided.`, in the shape of the annotate entry at line 2198, with the
      five decisions above as its paragraphs, each naming the test that pins
      it; and one register row, `holds`. The register row for 2026-09-09 M7b
      is unchanged: nothing in it is superseded, the grant is narrowed by
      nothing and widened by nothing.
- [ ] PRINCIPLES.md line 104: the no-rollback bullet gains one sentence, that
      a batch answer naming no revision mid-run stops the run for the same
      reason the refused batch is never retried.
- [ ] SPEC.md: read the styling paragraph at line 204 and the restyle section
      around it; if it says the loop reads between batches, or describes the
      grant on a created document, amend the sentence; if it says neither,
      change nothing and say so here as a ➕ note. Read the `publish` and
      `build` sections for a sentence about numbered lists or links; amend
      only what is now false.
- [ ] PLAN.md: this milestone is one row in the Done table, `M12`, listing
      the five items in one sentence, and the plan file's name.
- [ ] CLAUDE.md: re-read the invariants list against this milestone's tests.
      The grant invariant ("The one door in that wall is `Policy.GrantInPlace`")
      gains nothing unless a test here pins something no line names; expected
      answer is no change. `wc -l CLAUDE.md` under 300 either way.
- [ ] `cd go && go test -race ./boundary/` passes.
- [ ] `git commit -m "docs(v2): M12, five backlog items, decided and recorded"`

### Task 7: Verify acceptance criteria

- [ ] `make test`, `make vet` and `make build` pass.
- [ ] Every Validation Command above gives the answer it states. Record the
      three counts and the anchor name here as ➕ notes, and delete the scratch
      note and docx afterwards.
- [ ] `ls docs/backlog/` lists 18 files and none of the five.
- [ ] `git diff main...HEAD -- go/internal/guard/ | grep '^-' | grep -v '^---' | grep -v '^-\s*//'`
      is empty: nothing but comment lines left the guard.
- [ ] `grep -rn 'RevisionOf(' go/` shows the definition and one caller in
      `cmd/gdoc/restyle.go`.
- [ ] Nothing to commit: this task changes no file. If it finds something,
      that is a ➕ task with its own commit.

### Task 8: Update documentation

- [ ] Move this plan to `docs/plans/completed/`.
- [ ] `cd go && go test -race ./boundary/` passes.
- [ ] `git commit -m "docs(v2): M12 five backlog items, completed"`

## Post-Completion

**By hand, Nail, before the merge**: publish a note holding two numbered lists
and a `[see below](#heading)` link into the test folder with `gdoc publish`,
open it in Google Docs, and check the second list reads 1, 2 and the link
jumps. The Docs import of a `w:anchor` hyperlink and of a `w:bookmarkStart`
is unmeasured; if the jump does not survive the import, that is a MEASURED.md
row and a decision about whether the bookmark is still worth writing, not a
reason to revert the docx side, which Word reads.

**Not in this milestone, and named so nobody rediscovers it**: honouring an
author's start number (`5.`) is a `w:lvlOverride`/`w:startOverride` on the
list's own `w:num`, one element now that each list has one. The warning stays
until somebody decides it.

**The live test's two-policy workaround** in `internal/live/restyle_test.go`
stays after Task 1. Its stated reason is to exercise the in-place allowlist on
the copy, which the grant on a `LevelFull` id would still not do, so the
reason holds even though the demotion it also worked around is gone.
