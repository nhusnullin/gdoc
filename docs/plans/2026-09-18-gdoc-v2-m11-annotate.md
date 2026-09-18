# gdoc v2 Milestone 11: annotate

2026-09-18. A colleague quotes words from a document and gdoc leaves a comment
on exactly those words, anchored, opening with 🤖, and changing nothing. A
fifth writer beside probe, reply, propose and withdraw.

## Principles

Serves: 3, uncertainty never resolves toward the destructive answer. A comment
is the least a writer can do inside somebody's document: no character moves,
and the batch that carries it cannot move one whatever Google does with the
write mode. The quote is matched exactly once or refused, the body is checked
before anything leaves the machine, and the anchor is witnessed in the export
rather than trusted from the listing. Serves 2, the hub is where the user
stands: a colleague's own review skill reads the document, decides in the hub,
and hands gdoc the words and the reason.

Strains: 3, the probe clause. PRINCIPLES.md says the capability probe runs
every time, and this writer does not run it. The probe answers one question:
does SUGGEST make a real suggestion today, or a silent direct edit. A batch
holding only an `insertComment` makes no suggestion and can make no edit, so
the question has no bearing on what this write can do wrong. The bend is one
writer, and the DECISIONS.md entry in Task 6 records it. Nothing else in the
guard moves: its write levels, its comment routes and its suggestion rules are
untouched, and two new tests pin that the shape this command sends already
passes and its unsuggested twin is already refused.

## Overview

| Piece | Today | After |
|---|---|---|
| a comment on quoted text | only beside a replacement, through `propose` | `gdoc annotate <url>`, one comment or a file of many, no suggestion |
| the 🤖 prefix | reply: the caller writes it; propose: the binary adds it | annotate: the binary adds it, a body already carrying it is refused |
| the probe | `propose` runs it every time | not run: a batch holding only `insertComment` cannot change a character |
| the read-back | propose: three routes | annotate: two routes, Drive's listing and the docx export, `verified` is both |
| the note | `propose --md` records suggestion ids for `withdraw` | nothing recorded: there is no suggestion and nothing to take back |
| undo | `withdraw` for a suggestion, none for a comment | still none: the guard carries `POST` on comments and no `DELETE` |
| the guard | admits a SUGGEST batch on a handed-in id at `LevelSuggest` | the same, with two pins naming the comment-only shape |
| the docs | SPEC.md describes `reply`, `propose` and withdrawing | and `annotate`, with the decision entry written the same day |

## Decisions Nail took, 2026-09-18

Taken in the brainstorm that produced this plan and written into DECISIONS.md
in Task 6. Each is a decision and not a refactor. A task that finds one wrong
stops and says so.

1. **A new command and a new package, not a fourth shape of `propose`.** A
   proposal with an empty replacement is refused today on purpose, `propose`
   needs the probe and the note, and its comment id is what lets `withdraw`
   find its suggestion. A comment with no suggestion belongs beside those, not
   inside them.
2. **The name is `annotate`.** `comment` sits one letter from `comments`, the
   read command, and a typo in either direction would be silent.
3. **Both input forms.** `--quote` with `--body-file` for one comment by hand,
   `--from` for a file of many. Exclusive, and a missing pair is refused by
   name before any request.
4. **The binary adds the prefix.** The file carries the reason and gdoc writes
   `🤖 ` in front of it, like `propose`. A reason that already opens with the
   robot is refused so a comment can never carry two. The 🤖 stays the only
   record of authorship there is, because Google records every comment under
   the operator's own account.
5. **No probe, no folder, no note.** The probe answers whether SUGGEST makes a
   real suggestion, and this batch makes none: a request that inserts a
   comment cannot edit a character even when the mode is ignored. Nothing is
   written into a note because nothing can be withdrawn.
6. **Two read-backs, and nothing raises after the write.** Drive's listing must
   carry the id with the sent body, and the export must enclose the quoted
   words in a comment range, because the listing reports a destroyed anchor as
   healthy. `verified: false` is reported, never raised: the comment exists.
7. **The limits are stated, not lifted.** A document with more than one tab is
   refused, as `propose` refuses it. Body text only: headers, footers and
   footnotes cannot be quoted. Tables are walked and the live test says
   whether they land. A wrong comment is removed by a person in the document.
   Lifting the tab rule is its own small change and not part of this one.
8. **A run stops at the first entry that cannot be sent, as `propose` does.**
   The envelope is then `ok: false`, and the report still carries one entry
   per annotation in the file, each answering `sent` for itself, so a stop in
   the middle names what landed and what never left. `ok: true` over a run
   where an entry failed would give the envelope's `ok` a second meaning, and
   the exit code follows `ok`.

## Context (from discovery)

- **`propose.FindSpan(d, quoted)`** in `go/internal/propose/span.go:48` turns
  quoted words into a `docs.Range` in UTF-16 units, reading a document that
  just came back. It refuses an empty quote, a document with more than one
  tab, no match, more than one match, a match crossing a run the walk does not
  index, and a match inside somebody's pending suggestion. `paragraphs` walks
  table cells. It is exported and annotate imports it; the span walk keeps one
  owner. `propose` imports `docs`, `comments`, `docx`, `frontmatter` and
  `plaintext`, none of which will import annotate, so there is no cycle.
  `propose.BatchURL` at `propose.go:336` is exported too and is the one
  spelling of the batchUpdate URL.
- **`propose.Proposal.Check`** refuses an empty quote, an empty replacement, a
  line break on either side, and asks `plaintext.Markdown` of the reason
  alone. `reply.Check` requires the prefix instead of adding it. The two
  conventions are documented in each package's comment.
- **The batch `propose` sends** is three requests under
  `writeControl.writeMode: SUGGEST`, the third being
  `{"insertComment": {"range": {"startIndex": N, "endIndex": M}, "content":
  "..."}}`, measured 2026-08-29 and 2026-09-07 and recorded in DECISIONS.md.
  `assigneeEmailAddress` is the documented optional field. The answer carries
  `replies[i].insertComment.commentThread.commentId`, with the flat
  `insertComment.commentId` as the fallback, and `commentUpdateState`.
  `go/internal/propose/testdata/batch-saved-measured.json` is that answer with
  the ids replaced.
- **`propose.Verify`** in `verify.go:31` reads three routes and never fails:
  the document with suggestions inline, the preview view, and the docx export.
  It calls no Drive listing. Its docx route is `docxHolds` at `verify.go:150`:
  export through `docx.Export`, parse with `docx.Parse`, collect the comments
  in `f.Comments` whose `Text` matches the sent body, give no answer when two
  hits disagree about being attached, and answer `Anchored` of the one. A
  `docx.Comment` carries `Text`, `Anchored` and `Span`, the enclosed words.
  Annotate copies that shape for its docx route and adds the Drive listing as
  its other route. `docx.Match` is not used: it joins threads by words and
  author, needs a third read to build the threads, and answers a question
  `docx.Comment` answers directly.
- **The Drive listing** is `comments.ListURL` and `comments.Fetch` in
  `go/internal/comments/comments.go:130` and `:163`, returning `RawComment`
  with `ID`, `Content` and `QuotedFileContent`.
- **`propose.Session`** is the three-method interface that keeps `net/http`
  out of the package: `GetJSON`, `GetBytes`, `PostJSON`. Annotate declares the
  same shape.
- **The guard**: `judgeDocs` at `policy.go:577` admits `POST {id}:batchUpdate`
  on a handed-in id when `isSuggestMode(body)`, and the refusal at
  `policy.go:584` names SUGGEST. `judgeRequests` at `policy.go:1092` carries an
  unknown request kind at every level but `LevelInPlace`, and `insertComment`
  carries no `suggestion` in its name, so it is carried. `commentWrites`
  carries `POST` alone on the comment surface. The house pattern for pinning
  this is beside the writer: `TestTheGuardCarriesEveryOneOfProposesRequests`
  and `TestTheGuardRefusesTheSameBatchWithoutSuggestMode` in
  `propose_test.go:511` and `:533` send the batch `propose` really builds.
- **The command table** in `go/cmd/gdoc/commands.go` is one row per command
  with `words`, `flags` of `{name, kind, need, help}`, `summary`, `example`
  and `run`. Kinds today: `kindNone`, `kindFile`, `kindFolderID`,
  `kindCursor`, `kindDuration`. A `--quote` flag carrying free text needs a
  new kind that reads a string as it is. A kind touches `placeholder()` at
  `commands.go:35`, `TestEveryFlagIsReadTheWayItsKindSays`, `zshFlagSpec` at
  `completion.go:243` and `bashFlagArms` at `completion.go:366`, where the
  text flag lands in the opaque arm, and `help.go:101` and `:224`.
  `needEither` marks flags of which one is required, and the usage joiner at
  `help.go:200` joins `needEither` flags with ` | ` only when they are
  adjacent in the row. `TestTheUsageLineMarksWhatIsOptionalAndWhatIsAnAlternative`
  at `help_test.go:274` spells every usage line out as a literal.
  `restyle.go:69` is the pattern for refusing two exclusive flags by name.
- **`runReply` and `runPropose`** in `go/cmd/gdoc/write.go`, with `replyData`
  and `proposeData` as what each prints. `runPropose` at `write.go:270` stops
  at the first `Apply` error, returns `ok: false`, and still reports one entry
  per proposal with `sent` answered through `notSent`; warnings reach the
  envelope's one list through `about(quoted, warnings)`. `readBody` at
  `write.go:146` reads a body file and its error text names reply.
  `write_test.go` holds `TestReplyRefusesMarkdownBeforeAnyRequest`,
  `TestReplyRefusesABodyWithoutTheRobot`,
  `TestProposeRefusesAnUnknownKeyInTheProposalsFile`,
  `TestProposeRefusesAnEmptyProposalList` and
  `TestProposeReportsEveryProposalWhenOneOfThemCannotBeSent`, the shapes the
  annotate tests follow. `write.go` is 585 lines and `write_test.go` is 961,
  so annotate gets its own pair, as `restyle.go` has.
- **Boundary tests** that must stay green on the new files:
  `TestEveryPackageHasExactlyOnePackageComment`, `TestOnlyThePackageCommentReachesGoDoc`,
  `TestTheTaskMapNamesFilesThatExist`, `TestCLAUDEmdIsUnderTheCeiling` (256
  lines today, ceiling 300), `TestNetHTTPStaysInItsRooms`,
  `TestNothingRunsAnExternalProgram`. In `cmd/gdoc`:
  `TestEveryCommandInTheTableIsDispatchedAndNothingElseIs`,
  `TestEveryExampleIsACallTheTableAccepts`,
  `TestEverySkillNamesOnlyCommandsAndFlagsTheBinaryHas`.
- **The live tests** in `go/internal/live/` run under `GDOC_LIVE_TEST=1` and
  the writers under `GDOC_LIVE_WRITE=1` with `GDOC_LIVE_FOLDER_ID`, each in a
  document it creates in the test folder. `TestLiveProposeReplyWithdraw` in
  `live_test.go` is the pattern: create, write, read back both ways, trash.
  `live_test.go` is 722 lines and the house splits per feature
  (`publish_test.go`, `restyle_test.go`, `anchors_test.go`), so annotate gets
  `annotate_test.go`.
- **The documents**: SPEC.md has `### reply` at 275, `### propose` at 288 and
  `### Withdrawing a proposal` at 303, and the Never list; PLAN.md records
  each milestone under its own heading and moves it to the Done table when it
  runs; `docs/guide/writing.md` opens "Writing into a document: probe, reply,
  propose, withdraw" and holds the four call lines; README.md has the command
  table at line 138 and the guide index row naming the four writers at line
  189; `docs/backlog/propose-inside-tables.md` is the open question the live
  table case answers for the comment-only shape;
  `docs/backlog/a-withdrawn-proposals-comment-stays.md` records why there is
  no delete.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it
  to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own
  commit, and each ends with the test gate as its last checkbox.
- **CRITICAL: every task MUST include new or updated tests**, success and
  failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** `make test`
  runs `-race`; keep it there.
- **CRITICAL: the guard does not move.** No policy change, no new grant, no new
  route. Every existing guard test stays green without an assertion changed,
  and the two new pins live beside the writer.
- **CRITICAL: nothing raises after the write.** A test for each read-back
  failing shows a warning and `ok: true`.
- **CRITICAL: every refusal is by name and before the first request.** A test
  for each bad shape asserts the stub wire saw nothing.
- **CRITICAL: the span walk and the batch URL have one owner.** Annotate
  imports `FindSpan` and `BatchURL` from `propose`; nothing is copied.
- **CRITICAL: new code goes in new files.** `cmd/gdoc/annotate.go` and its
  test, `live/annotate_test.go`. The three shared files above the 800 ceiling
  or near it do not grow.
- **CRITICAL: facts only in Go.** `verified` and `checks` say what was read
  back. Nothing says a comment was needed, handled or useful.
- **CRITICAL: one object on stdout, always**, through `internal/emit`.
- **CRITICAL: no `os/exec` anywhere, tests included.**
- **CRITICAL: the tests state literals.** The prefix is `"🤖 "` in the test,
  never read from `plaintext`, and the usage line is spelled out.
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces. Plain English.

## Testing Strategy

- **Unit, `internal/annotate`**: `Check` refuses an empty quote, a quote with
  a line break, an empty why, a why opening with 🤖, and markdown in the why,
  each by name; `Body` is `"🤖 "` plus the why as a literal; `Apply` refuses a
  document with more than one tab before any write; `Apply` sends one batch
  holding one `insertComment` at the range `FindSpan` found, under SUGGEST,
  with `assigneeEmailAddress` present only when set; the guard carries that
  batch on a handed-in id and refuses it without SUGGEST; the comment id is
  read from `commentThread` first and the flat field second;
  `commentUpdateState` other than `ALL_SAVED` is a warning; a guard refusal
  before the write is an error and the wire saw no batch; each read-back that
  cannot be read or does not hold is a warning and never an error; both holding
  is `verified: true`.
- **Unit, `cmd/gdoc`**: `--from` beside `--quote` is refused; `--quote` without
  `--body-file` and the reverse are refused; no input at all is refused; an
  unknown key in the file is refused; an empty list is refused; a why with 🤖
  is refused before any request; markdown is refused before any request; a
  file of two lands two batches and prints two results; the by-hand form lands
  one; a second entry that cannot be sent stops the run with `ok: false` and
  both entries reported; the usage line is the literal in the help table; the
  envelope is one object and the exit code follows `ok`.
- **Boundary**: the package comment tests, the task map, the CLAUDE.md ceiling,
  the wire rooms, the external-program scan, all green on the new files.
- **Live, opt-in under `GDOC_LIVE_WRITE=1`**: one comment on a paragraph and
  one inside a two-cell table, each verified by both routes, on a throwaway
  document the test creates and trashes.
- Coverage standard: every exported function in the new package has a test;
  the package at or above 80%.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- `cd go && go vet ./...`
- `make build && bin/gdoc help annotate` prints
  `Usage: gdoc annotate <url> --quote <text> | --from <file> [--body-file <file>]`
- `bin/gdoc annotate` with no url fails with one object naming the url
- `cd go && GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1 GDOC_LIVE_FOLDER_ID=... go test -race -run TestLiveAnnotateParagraphAndTable ./internal/live/`
- `wc -l CLAUDE.md` under 300

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

**The package** is `go/internal/annotate`. `Annotation{Quoted, Why, Assignee}`
is one entry. `Check` is the shape rule, asked of every entry before the first
leaves the machine. `Body(why)` is the prefix plus the why. `Batch(r, body,
assignee)` is the one request body, so a test can send what the writer really
builds through the guard. `Apply(ctx, s, docID, a)` reads the document, finds
the span through `propose.FindSpan`, sends one batch and verifies. `Result`
carries `quoted`, `comment_id`, `comment_update_state`, `verified`, `checks`
and `warnings`. `verify.go` holds the two read-backs, each answering for
itself. `doc.go` names the test for every rule.

**The command** is one row in the table, and `go/cmd/gdoc/annotate.go` holds
`cmdAnnotate`, which turns the flags into a list of one or many, refuses the
exclusive pair, reads the file strictly and reads the body file with its own
error text, and `runAnnotate`, which checks every entry, places them in order
each with its own read, stops at the first that cannot be sent with
`ok: false`, and reports one entry per annotation with `sent` answered.

**The wire**: `POST` on `propose.BatchURL(id)` with

```json
{"requests": [{"insertComment": {"range": {"startIndex": 41, "endIndex": 58},
                                  "content": "🤖 The 2026 register says quarterly.",
                                  "assigneeEmailAddress": "x@altery.com"}}],
 "writeControl": {"writeMode": "SUGGEST"}}
```

then `GET` Drive's comment listing and `GET` the docx export, the two routes
the batch did not go out on.

**The guard** is unchanged and pinned from the writer's side.

**The documents**: DECISIONS.md, SPEC.md, PLAN.md, CLAUDE.md,
`docs/guide/writing.md`, README.md and the tables backlog item, all in one
task, the same day as the code.

## Technical Details

**The file `--from` reads**, a strict-decoded JSON array:

```json
[
  {"quoted": "reviewed annually", "why": "The 2026 register says quarterly."},
  {"quoted": "the Cyprus entity", "why": "Named twice with two spellings.", "assignee": "x@altery.com"}
]
```

**The by-hand form**: `--quote "reviewed annually" --body-file why.txt`, where
the file holds the bare why with no prefix. Trailing newlines are trimmed as
`readBody` trims them for reply, with an error text that names annotate.

**The table row**, in this order so the usage joiner sees the two alternatives
side by side: `--quote` (either, the new text kind), `--from` (either, file),
`--body-file` (optional, file). Usage line:

```
Usage: gdoc annotate <url> --quote <text> | --from <file> [--body-file <file>]
```

**The result**, one per entry, inside the envelope's `data.annotations`:

```json
{"quoted": "reviewed annually", "sent": true, "comment_id": "AAAC...", "comment_update_state": "ALL_SAVED",
 "verified": true, "checks": {"drive_listing": true, "docx_anchored": true}}
```

An entry that could not be sent is `{"quoted": "...", "sent": false}`, the
envelope is `ok: false` with `error` naming why, and every later entry is
reported `sent: false` too.

**Refusals, each a sentence naming the entry**: quotes no text; the quote
carries a line break; gives no reason; the reason opens with the robot, which
this writer adds itself; the reason carries markdown Docs would render
literally, naming the mark; the document has more than one tab; `--from`
beside `--quote`; `--quote` without `--body-file`, and the reverse; no input;
an unknown key; an empty list.

**Constants**: the prefix from `plaintext`; the export byte limit from `docx`.

## Implementation Steps

### Task 1: the annotation and its shape rule

**Files:**
- Create: `go/internal/annotate/doc.go`, `annotate.go`, `annotate_test.go`

- [x] Test first, `TestCheckRefusesEachBadShapeByName`: table of an empty
      quote, a quote with `\n`, an empty why, a why opening with `🤖 `, a why
      holding `**bold**`; each refused with a sentence naming what is wrong.
- [x] Test, `TestCheckAcceptsAPlainAnnotation`: a quote and a why pass, with
      and without an assignee.
- [x] Test, `TestBodyIsTheRobotAndTheWhy`: `Body("The register.")` is the
      literal `"🤖 The register."`.
- [x] `Annotation`, `Check`, `Body` in `annotate.go`, the markdown rule asked
      through `plaintext.Markdown` of the why alone.
- [x] `doc.go` opens `Package annotate`, says why the prefix is added here and
      required in reply, and names the three tests.
- [x] `cd go && go test -race ./internal/annotate/` passes.
- [x] `git commit -m "feat(v2): the annotation and its shape rule"`

### Task 2: the batch, the guard, and the id it comes back with

**Files:**
- Create: `go/internal/annotate/testdata/before.json`, `two-tabs.json`,
  `batch-saved.json`, `batch-saved-flat.json`, `batch-failed.json`
- Modify: `go/internal/annotate/annotate.go`, `annotate_test.go`, `doc.go`

- [x] Test first, `TestBatchIsOneInsertCommentUnderSuggest`: `Batch` over a
      literal range and body is one request at the literal start and end,
      content the literal robot body, `writeMode` SUGGEST, and no
      `assigneeEmailAddress` key; with an assignee the key is present.
- [x] Test, `TestTheGuardCarriesTheAnnotateBatchOnAHandedInDocument`: a policy
      with one handed-in id and nothing granted carries `POST` on
      `propose.BatchURL(id)` with `Batch`'s real output.
- [x] Test, `TestTheGuardRefusesTheSameBatchWithoutSuggestMode`: the same body
      with no `writeControl` is refused and the refusal names SUGGEST. Both
      pass with no change under `guard/`. If one does not, stop: the design
      rests on this and the plan is wrong.
- [x] Test, `TestApplySendsTheBatchAtTheSpanItFound`: a stub session answering
      `before.json` to the document read; the POST body is `Batch` at the
      literal range the quote sits at.
- [x] Test, `TestApplyRefusesADocumentWithMoreThanOneTabBeforeAnyWrite`:
      `two-tabs.json`; an error naming the tab count, no POST.
- [x] Test, `TestApplyReadsTheCommentIdFromTheThreadFirst`: `batch-saved.json`
      yields the thread's id; `batch-saved-flat.json`, carrying only the flat
      field, yields that one.
- [x] Test, `TestAnUpdateStateOtherThanSavedIsAWarning`: `batch-failed.json`
      gives a result with a warning naming the state and no error.
- [x] Test, `TestApplyRefusesBeforeTheWriteAndSendsNothing`: a bad shape and
      a span not found each return an error and the stub saw no POST.
- [x] Test, `TestAGuardRefusalIsAnErrorAndTheWireSawNoBatch`: the session
      refuses the POST; the error carries the refusal.
- [x] `Session`, `Batch`, `Apply`, `Result`, the range from
      `propose.FindSpan`, the URL from `propose.BatchURL`, the id read as
      propose reads it. Verify stubbed to nothing until Task 3.
- [x] `doc.go` gains the sections on the batch and on the guard and names the
      tests.
- [x] `cd go && go test -race ./internal/annotate/` passes.
- [x] `git commit -m "feat(v2): annotate sends one comment and reads its id"`

### Task 3: the two read-backs

**Files:**
- Create: `go/internal/annotate/verify.go`, `verify_test.go`,
  `testdata/comments.json`, `testdata/document.xml`, `testdata/comments.xml`
- Modify: `go/internal/annotate/annotate.go`, `doc.go`

- [x] Test first, `TestVerifiedIsBothRoutesHolding`: the listing carries the
      id with the sent body and the quote as `quotedFileContent`, the export
      carries one comment reading the sent body, anchored, whose span holds
      the quote; `verified` true, both checks true, no warning.
- [x] Test, `TestAListingWithoutTheIdIsAWarningNotAnError`: `drive_listing`
      false, `verified` false, a warning naming the id, no error.
- [x] Test, `TestAnExportThatDoesNotAnchorTheCommentIsAWarning`: the comment
      is in the export and `Anchored` is false; `docx_anchored` false, the
      warning says it is attached to no text.
- [x] Test, `TestTwoExportedCommentsThatDisagreeGiveNoAnswer`: two comments
      reading the same body, one anchored and one not; `docx_anchored` false
      with the warning propose uses for the same case.
- [x] Test, `TestAReadBackThatFailedIsAWarningNamingTheRoute`: each GET
      failing gives a warning naming the route and leaves the other check to
      answer for itself.
- [x] `Verify(ctx, s, docID, commentID, quoted, body)` returning `Checks` and
      warnings. The Drive route through `comments.Fetch`, matching the id and
      comparing `Content` and `QuotedFileContent`. The docx route in the
      `docxHolds` shape: `docx.Export`, `docx.Parse`, the hits in
      `f.Comments` by `Text`, no answer when hits disagree, then `Anchored`
      and `Span` containing the quote.
- [x] `Apply` calls it after the write and never returns an error after the
      write.
- [x] `doc.go` gains the section on the read-backs and names the tests.
- [x] `cd go && go test -race ./internal/annotate/` passes.
- [x] `git commit -m "feat(v2): annotate reads its comment back two ways"`

### Task 4: the text flag kind

**Files:**
- Modify: `go/cmd/gdoc/commands.go`, `commands_test.go`, `completion.go`,
  `completion_test.go`, `help.go`

- [x] Test first, a case in `TestEveryFlagIsReadTheWayItsKindSays` for a kind
      that reads its value as given, spaces and quotes included, and refuses
      an empty value by name.
- [x] Test, a case in the completion tests: a text flag lands in the opaque
      arm of both scripts, with no file or folder completion offered.
- [x] Test, `placeholder()` prints `<text>` for the kind.
- [x] `kindText` in `commands.go`, its arm in `placeholder()`, its read in the
      parser, and its arms in `zshFlagSpec` and `bashFlagArms`. `help.go`
      wherever it switches on kind.
- [x] `cd go && go test -race ./cmd/gdoc/` passes.
- [x] `git commit -m "feat(v2): a flag kind that carries text as given"`

### Task 5: the command

**Files:**
- Create: `go/cmd/gdoc/annotate.go`, `annotate_test.go`,
  `testdata/annotate-before.json`, `annotate-batch.json`,
  `annotate-comments.json`, `annotate-document.xml`, `annotate-comments.xml`,
  `annotations.json`
- Modify: `go/cmd/gdoc/commands.go`, `help_test.go`, `doc.go`

- [x] Test first, `TestAnnotateRefusesFromBesideQuote`: both given, refused by
      name, no request.
- [x] Test, `TestAnnotateNeedsAQuoteWithItsBodyFile`: `--quote` alone,
      `--body-file` alone, and nothing at all, each refused by name.
- [x] Test, `TestAnnotateRefusesAnUnknownKeyInTheFile` and
      `TestAnnotateRefusesAnEmptyList`.
- [x] Test, `TestAnnotateRefusesARobotInTheWhyBeforeAnyRequest` and
      `TestAnnotateRefusesMarkdownBeforeAnyRequest`: the stub saw nothing.
- [x] Test, `TestAnnotatePlacesEachEntryAndVerifiesIt`: a file of two lands
      two batches in order, prints two results with `sent: true`, `ok: true`.
- [x] Test, `TestAnnotateByHandPlacesOne`: `--quote` and `--body-file` land one.
- [x] Test, `TestAnnotateStopsAtTheFirstEntryThatCannotBeSent`: the second
      quote is absent; the first lands with `sent: true`, the second is
      reported `sent: false`, the envelope is `ok: false` and `error` names
      the quote.
- [x] Test, the annotate row in
      `TestTheUsageLineMarksWhatIsOptionalAndWhatIsAnAlternative` with the
      literal `Usage: gdoc annotate <url> --quote <text> | --from <file> [--body-file <file>]`.
- [x] The table row: `annotate <url>`, flags in the order `--quote`
      (`kindText`, either), `--from` (`kindFile`, either), `--body-file`
      (`kindFile`, optional), summary "Leave a comment on the exact words you
      quote, under the robot prefix, changing nothing.", example
      `gdoc annotate https://docs.google.com/document/d/1AbC.../edit --from annotations.json`.
- [x] `cmdAnnotate`, `runAnnotate` and `annotateData` in `annotate.go`, with
      a body-file reader whose error text names annotate. Entries checked all
      before the first is sent; the loop stops at the first `Apply` error the
      way `runPropose` does, reporting every entry.
- [x] `cmd/gdoc/doc.go` gains the command's line in the list and a short
      section on why it takes no folder and no note.
- [x] `cd go && go test -race ./...` passes, including the table, example and
      skills walks.
- [x] `git commit -m "feat(v2): gdoc annotate"`

### Task 6: the documents, the same day

**Files:**
- Modify: `docs/v2/DECISIONS.md`, `docs/v2/SPEC.md`, `docs/v2/PLAN.md`,
  `CLAUDE.md`, `docs/guide/writing.md`, `README.md`,
  `docs/backlog/propose-inside-tables.md`

- [x] DECISIONS.md: an entry dated 2026-09-18, "annotate: a comment on quoted
      words, and nothing else", holding the eight decisions above with their
      reasons, the probe clause bent for this one writer and why, and its row
      in the register.
- [x] SPEC.md: `### annotate` after `### propose`, the two input forms, the
      prefix rule, no probe, the two read-backs, the stop at the first
      failure, the limits. The Never list gains nothing: the existing lines
      already cover a comment.
- [x] PLAN.md: `## M11. annotate` after M10, naming the decision date and this
      plan, and what lands. On completion the row moves to the Done table.
- [x] CLAUDE.md: the row `go/internal/annotate/` in "What lives where", the
      row "a comment gdoc leaves on quoted words" in "If you touch", the
      command count in the `go/cmd/gdoc/` row moved from thirteen to
      fourteen. Under 300 lines.
- [x] `docs/guide/writing.md`: the title and the opening say five commands,
      the call block gains the two annotate lines, a paragraph for `annotate`
      after `propose` saying what it sends, what it reads back, that the file
      carries no prefix, and the limits.
- [x] README.md: the command table row for `annotate <url>`, and the guide
      index row for Writing names it.
- [x] `docs/backlog/propose-inside-tables.md`: one line saying the comment-only
      shape is measured by `TestLiveAnnotateParagraphAndTable` in Task 7, with
      the answer filled in after that task runs.
- [x] `cd go && go test -race ./boundary/` passes: the task map, the ceiling.
- [x] `git commit -m "docs(v2): annotate, decided and described"`

### Task 7: the live test, opt-in

**Files:**
- Create: `go/internal/live/annotate_test.go`
- Modify: `go/internal/live/doc.go`

- [x] `TestLiveAnnotateParagraphAndTable`, under `GDOC_LIVE_WRITE=1`: create a
      document in the test folder holding one paragraph and a two-cell table,
      annotate one phrase in the paragraph and one inside a cell, assert both
      results are `verified: true` with both checks, then trash the document
      and confirm the trash.
      ➕ The paragraph case asserts both checks and `verified`. The table case
      logs them and asserts nothing, because it is the open question in
      `docs/backlog/propose-inside-tables.md` and the house rule for the live
      probes is that a measurement which fails the build has already decided
      the answer. The ⚠️ below is what that choice serves.
- [x] `live/doc.go` names the test and what it creates.
- [x] Run it once by hand against the test folder and record the table answer
      in `docs/backlog/propose-inside-tables.md`. ⚠️ If the table case does not
      verify, record which check failed and leave the backlog item open; the
      paragraph case is the acceptance bar for this milestone.
      ➕ Run 2026-09-18: both cases `drive_listing true, docx_anchored true,
      verified true`, the cell as the paragraph. Two of the three suspects in
      the backlog item are cleared and it stays open on the third,
      `deleteContentRange` inside a cell, which annotate never sends.
- [x] `cd go && go test -race ./...` still passes without the variables set.
- [x] `git commit -m "test(v2): annotate, live"`

### Task 8: Verify acceptance criteria

- [ ] `gdoc annotate <url> --from annotations.json` on a throwaway document
      leaves one anchored 🤖 comment per entry and the envelope says
      `verified: true` for each.
- [ ] `gdoc annotate <url> --quote "..." --body-file why.txt` leaves one.
- [ ] Every refusal in Technical Details is reproduced by hand once and names
      what is wrong.
- [ ] The guard tests are green with no assertion changed.
- [ ] `bin/gdoc help annotate` prints the usage line above, and
      `bin/gdoc completion zsh --out /dev/stdout` carries the command.
- [ ] `make test`, `make vet` and `make build` pass.
- [ ] Nothing to commit: this task changes no file. If it finds something,
      that is a ➕ task with its own commit.

### Task 9: Update documentation

- [ ] Re-read CLAUDE.md's invariants list: none needs a new line, since the
      prefix rule, the quote rule and the guard rule already cover annotate.
      Add one only if a test in this milestone pins something no line names.
- [ ] PLAN.md: the M11 heading becomes a row in the Done table.
- [ ] Move this plan to `docs/plans/completed/`.
- [ ] `cd go && go test -race ./boundary/` passes.
- [ ] `git commit -m "docs(v2): M11 annotate, completed"`

## Post-Completion

**By hand, in the hub**: nothing. No skill in this repository calls annotate
yet. A colleague's own review skill is what will, and the guide page is what
it reads.

**A question for Nail before Task 4 starts, not a blocker**: the by-hand
`--quote` form is the largest single source of complexity in the command, a
new flag kind, the exclusivity rule and three of the tests. It stands because
Nail chose both forms on 2026-09-18. If nobody will call it by hand, dropping
it removes Task 4 whole.

**A decision for later, not this milestone**: lifting the one-tab rule in
`FindSpan`, either by naming a tab in the entry or by searching every tab and
requiring the quote to be unique across them. Colleagues' documents with tabs
cannot be annotated until then, and the refusal says so.
