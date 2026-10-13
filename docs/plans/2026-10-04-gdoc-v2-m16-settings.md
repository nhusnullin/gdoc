# gdoc v2 Milestone 16: settings, the mark, and answers an AI can act on, plan

2026-10-04. The task list for the specification beside it,
`docs/plans/2026-10-04-gdoc-v2-m16-settings-spec.md`, whose decisions are
Nail's and settled. One commit per task. Released as v2.11.0, carrying Parts A
and B, after Nail's acceptance. Part C, the token in the Keychain and signed
releases, moved to M17 on 2026-10-13 (Nail):
`docs/plans/2026-10-13-gdoc-v2-m17-keychain.md`.

The spec holds the rules, the decisions and the measurements. This plan repeats
none of them. Where the code, the spec or this plan disagree, the task stops and
says so rather than choosing.

Task 1 and Part A (Tasks 2 to 8) are run by hand in the session that wrote
this plan. Part A changes no contract and needs no decision. Part B follows
from Task 9.

## Principles

Serves: 4, every word costs a reader's attention. A refusal says what to do
next, and a colleague picks a quieter mark.

Strains, each written into DECISIONS.md by Task 1 before the code: the robot
prefix invariant, and "the marker is the only record of authorship".

## Context (from discovery)

- `go/internal/propose/propose.go:178` and `go/internal/annotate/annotate.go:67`
  refuse a missing reason with "one Nail has to guess at";
  `go/cmd/gdoc/annotate.go:93` has the same words. No test quotes them.
- `go/internal/propose/doc.go:306` says "A quote inside a table cell is refused
  on both routes". The paragraph is about the two block routes. `paragraphs()`
  (`span.go:183`) walks into tables, and no offline test pins a words proposal
  in a cell; the evidence is MEASURED.md "A proposal into a table cell lands",
  2026-09-18.
- `readProposals` (`go/cmd/gdoc/write.go:557`) decodes the whole list with
  `DisallowUnknownFields`, so an unknown field reads `json: unknown field
  "reason"`. `readAnnotations` (`go/cmd/gdoc/annotate.go:143`) does the same.
  `TestProposeRefusesAnUnknownFieldInABlockEntry` asserts only that the field
  is named.
- `Proposal.checkWords` (`propose.go:118`) refuses a deletion with "a plain
  deletion is not a shape this write has", naming no way forward.
- `runPropose` (`write.go:267`) returns `err.Error()` from `propose.Apply` raw
  when a proposal fails before its write, so the refusal does not say which
  proposal it was.
- The `--from` summary (`commands.go:278`) names `quoted, replacement` and the
  block fields but not `why`, which `checkWhy` requires.
- `runPropose` reads the document (`docs.Fetch`, for the tab count), then
  `applyWords` and `ApplyBlock` read it again through `readOneTab`, which is the
  same `docs.Fetch` plus the same tab refusal.

## Context (from discovery), Part B

- The mark today is two constants in `go/internal/plaintext/plaintext.go`:
  `Robot = "🤖"` (line 104) and `Prefix = Robot + " "` (line 99), with
  `OpensWithRobot` (line 118) the one reader. Three writers re-export them as
  package constants: `reply.Prefix` (`reply.go:95`), `propose.Prefix` and
  `propose.Robot` (`propose.go:29,33`), `annotate.Prefix` and `annotate.Robot`
  (`annotate.go:25,29`). Nothing passes a mark around as a value yet.
- Where the mark is written: `propose.Batch` (`propose.go:487`, `Prefix +
  p.Why`), `propose.BlockBatch` (`blockbatch.go:217`), the two read-backs that
  compare against the sent string (`propose.go:344`, `blockapply.go:118`), and
  `annotate.Body` (`annotate.go:104`). propose and annotate refuse a reason
  that opens with `Robot` (`propose.go:194`, `annotate.go:80`). `reply.Check`
  (`reply.go:112`) requires the body to open with exactly `Prefix`, and
  `reply.Post` sends the body verbatim.
- Where the mark is read: `comments.byGdoc` (`comments.go:309`), called from
  `replies()` (`comments.go:270`); `comments.Threads(raw, d)` has four callers:
  `cmd/gdoc/read.go:558`, `cmd/gdoc/mcptools.go:701`,
  `internal/comments/wait.go:177` and `internal/restyle/survey.go:135`. The
  survey only counts threads and reads no `ByGdoc`. `mcpRemarks`
  (`mcptools.go:520`) asks `OpensWithRobot` of a thread's opening comment.
  `chat.robotNotOurs` (`chat/facts.go:143`) is the third reader.
- `TestTheRobotMarkIsReadInOnePlace` (`go/boundary/boundary_test.go:366`)
  finds every constant whose value is the literal `robotMark = "🤖"`
  (line ~425) and allows prefix checks only in `markWriters` (line ~334:
  `propose.go`, `annotate.go`, `reply.go`). It also fails if a listed reader
  (`internal/comments`, `internal/chat`, `cmd/gdoc`) stops calling
  `plaintext.OpensWithRobot`. It has to move with the API in Task 15.
- Arguments: `route()` (`cmd/gdoc/main.go:97`) sends `mcp` to `serveMCP` before
  `run()`. `serveMCP` (`mcp.go:42`) calls `checkMCPArgs` (`mcp.go:28`), which
  refuses anything after the word: `TestMcpTakesNoWordsAndNoFlags`.
  `dispatch` (`main.go:201`) answers `--help`/`-h` through `helpAsked`
  (`help.go:249`) and `helpWords` (`help.go:272`), then walks the table and
  calls `parseArgsN` (`read.go:217`). `safeDispatch` has two callers:
  `run()` (`main.go:146`) and the MCP tool call (`mcptools.go:460`), which
  builds an argv from the tool's arguments.
- `parseArgsN` refuses an empty value in both spellings (`read.go:242,253`),
  so `--mark ""` cannot go through it. The two setting flags need their own
  reader, the way `--help` has one.
- Every flag lives in the table in `commands.go`, with its summary. `jsonFlag`
  (`commands.go:100`) is the precedent for a flag read by two readers through
  one constant. `TestEveryFlagIsReadTheWayItsKindSays`
  (`commands_test.go:113`) loops the table's flags only, so setting flags kept
  in a second list need a test of their own.
- The help object is `helpReport` (`help.go:21`): `commands` and `update`. M15's
  pipe goldens (`cmd/gdoc/testdata/pipe/help*.golden`) and screen goldens
  (`testdata/screens/help*.golden`, `bare*.golden`) freeze it. M15 made them
  read only for M15's run; in this run a new field and a changed summary change
  them, and each task that does so says which golden it regenerates and why.
- Chat's reply tool maps `body` to `--body-file` through a temp file
  (`mcptools.go:283`, `mcpArg{file: "body.txt"}`; `mcpfiles.go:49`). A flag
  with `file: ""` is passed on the line as `flag=value`, but a value opening
  with `-` is refused first (`mcptools.go:834`, `dashRefusal`). A reply body
  opening with "- " is plain text today (`plaintext` allows hyphen bullets), so
  the move to `--body` must not inherit that refusal for `body`.
- The reply schema text names the prefix and tells the model to type it
  (`mcptools.go:288`). The table summaries of `reply` and `annotate` say "under
  the robot prefix" (`commands.go`), and `release/mcpb/manifest.json:32,36`
  repeats each tool description whole: `TestTheManifestListsTheToolsToolsListLists`
  compares them byte for byte.
- `go/cmd/gdoc/chatheader.md:44,93,96` names `robot_not_ours` and tells the
  model to draft a `🤖` reply. `go/cmd/gdoc/review.md` is a byte copy of
  `skills/gdoc-review/review.md` (`TestTheEmbeddedCoreIsTheSkillsCore`).
- `robot_not_ours`: field `chat.Facts.RobotNotOurs` (`chat/facts.go:30`), set
  at line 111, read by no hold rule since `7fea76a` (`chat/rules.go:320`).
  `Ledger.Wrote` stays: the Dictated rule asks it (`chat/rules.go:143`).
  Tests naming the fact: `chat/facts_test.go`, `chat/ledger_test.go`
  (`TestRobotNotOursUsesTheLedger`), `cmd/gdoc/mcpledger_test.go:172`,
  `mcplabel_test.go:278,300`, `mcpguide_test.go:304`, `mcplanded_test.go:20`.
- Skills that tell a session to type 🤖: `skills/gdoc-review/SKILL.md:181-316`,
  `review.md:15-166`, `live.md:74,81`, `propose.md:63-251`,
  `skills/gdoc-align/SKILL.md:272`. Guides: `docs/guide/writing.md:55,89,236`,
  `docs/guide/chat.md:75`, `README.md:98,146`.
- Strict YAML is `yaml.UnmarshalWithOptions(b, &v, yaml.Strict())`, as in
  `internal/frontmatter/frontmatter.go:108` and `internal/house/house.go:486`.
  goccy/go-yaml is already an allowed module.
- `internal/config` has no `doc.go`; its package comment is in `config.go`.
  `GDOC_CONFIG_DIR` is how every test isolates the config folder
  (`cmd/gdoc/doc.go:703`), so `settings.yaml` is absent in every test unless a
  test writes one.
- `CLAUDE.md` is 299 lines, ceiling 300. `README.md` is 200 lines, ceiling 200
  (`TestTheReleaseREADMEIsUnderTheCeiling`). Every line added to either is a
  line taken out.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it
  to watch it fail, then implements until it passes.
- One commit per task. Before each commit: `cd go && go test -race ./...`,
  `test -z "$(gofmt -l .)"`, `go vet ./...` and `GOOS=darwin go vet ./...`.
- **CRITICAL: every task MUST include new or updated tests.**
- **CRITICAL: all tests must pass before the next task starts.**
- **CRITICAL: no assertion is loosened.**
- **CRITICAL: a test states its value as a literal**, never reading the constant
  it checks.
- **CRITICAL: no doc.go rule without its test named**, and a doc.go or test name
  made false by a task is fixed in that task.
- **CRITICAL: no real company domain in examples.** `example.com` only.
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces. Plain English.

Part B adds:

- **CRITICAL: the mark is a value, never a constant a writer reads.** After
  Task 15 no package but `internal/plaintext` and `internal/settings` names a
  mark character outside tests.
- **CRITICAL: a mark test states the mark as a literal** (`"⁘"`, `"∴"`, `"🤖"`,
  `""`), never `plaintext.DefaultMark` read back.
- **CRITICAL: no fourth module, and CGO stays off for `make dist`.**

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && test -z "$(gofmt -l .)" && go vet ./... && GOOS=darwin go vet ./...`
- `make dist` builds all three binaries.
- `wc -l CLAUDE.md` under 300.

Part B:

- `cd go && go build -o "$TMPDIR/gdoc-m16" ./cmd/gdoc`, then, each with
  `GDOC_CONFIG_DIR="$(mktemp -d)"`:
  - `"$TMPDIR/gdoc-m16" help --json 2>/dev/null | grep -o '"settings":{[^}]*}'`
    prints `"settings":{"mark":"⁘","mark_from":"default"}`.
  - The same with `--mark ∴` prints `"mark":"∴","mark_from":"argument"`.
  - The same with `--mark ""` prints `"mark":"","mark_from":"argument"`.
  - With `mark: "🤖"` in a temp file and `--config <file>`: `"mark":"🤖","mark_from":"file"`.
  - `--config /nonexistent/settings.yaml` exits 1 and the object says `ok: false`.
  - A settings file holding `colour: red` exits 1 and the error names `colour`.
  - `--mark '**'` exits 1 and the error names the markdown.
  - `printf '' | "$TMPDIR/gdoc-m16" mcp --mark ∴; echo $?` prints `0`;
    `"$TMPDIR/gdoc-m16" mcp --stdio; echo $?` prints `1`.
- `grep -rn 'robot_not_ours\|RobotNotOurs' go/` finds nothing.
- `grep -rn 'OpensWithRobot\|plaintext\.Prefix\|plaintext\.Robot' go/` finds nothing.

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Implementation Steps

### Task 1: the DECISIONS.md entries, SPEC.md and CLAUDE.md

Written before any settings or Keychain code. Part A needs no entry.

**Files:**
- Modify: `docs/v2/DECISIONS.md`, `docs/v2/SPEC.md`, `CLAUDE.md`

- [x] One entry, dated the day this task runs, "Settings, the mark in effect,
      and the token in the macOS Keychain", from the spec's decisions, in the
      file's own shape, with its register row `holds`. Its test list is this
      plan's test names.
- [x] The M14 entry's row names the one clause superseded: chat's
      `robot_not_ours` fact is removed. The 2026-08-29 "Everything gdoc writes
      opens with 🤖" row is superseded in the same words.
- [x] `SPEC.md`: the robot prefix becomes the mark in effect; the token on
      macOS lives in the Keychain; the settings file and its two arguments.
- [x] `CLAUDE.md`, under 300 lines: the robot prefix invariant names the mark
      in effect and that it may be empty; "Secrets live in" names the Keychain
      on macOS. The external-program invariant already names the `open`
      exception on main, so it is unchanged. CLAUDE.md is at 299 lines: a later
      task that adds a line shortens a row first.
- [x] `cd go && go test -race ./boundary/` passes.
- [x] `git commit -m "docs(decisions): settings, the mark in effect, and the token in the Keychain"`
- ➕ 2026-10-13, when Part C moved to M17: the entry is split in two, the
      settings and the mark, and the Keychain with signed releases, which
      says it is built in M17. The Keychain lines in `SPEC.md` and `CLAUDE.md`
      are taken back, because the token stays a file in v2.11.0. Committed as
      `docs(plans): Part C moves to M17`.

### Task 2: no refusal names a person

**Files:**
- Modify: `go/internal/propose/propose.go`, `go/internal/annotate/annotate.go`,
  `go/cmd/gdoc/annotate.go`
- Create: `go/boundary/refusal_test.go`

- [x] `TestNoRefusalNamesAPerson` reads every non-test `.go` file under `go/`
      and fails on a line that builds an error (`errors.New(`, `fmt.Errorf(`)
      or an `emit.Result` `Error:` and carries `Nail`.
- ➕ `TestTheRefusalScannerFindsAllThreeShapes`, so the scanner cannot pass a
      tree by finding nothing; it reads strings joined with `+`, which is how
      the `cmd/gdoc/annotate.go` sentence is built. `boundary/doc.go` names both.
- [x] The three sentences say "is one the reader has to guess at".
- [x] `git commit -m "fix: no refusal names a person"`

### Task 3: a words proposal lands inside a table cell, and the note says so

**Files:**
- Modify: `go/internal/propose/doc.go`
- Create: `go/internal/propose/span_table_test.go`

- [x] `TestFindSpanPlacesAQuoteInsideATableCell`: a document whose one table
      holds `Business` and `£20` in two cells; `FindSpan(d, "£20")` returns the
      cell's range, by literal indexes. It passed on first run: it pins
      behaviour that already worked.
- [x] `doc.go:306` says "on both block routes", and one sentence says a words
      proposal is placed in a cell, naming the test.
- [x] `git commit -m "docs(propose): a words proposal lands in a table cell, and a test pins it"`

### Task 4: an unknown field is refused with the fields the entry takes

**Files:**
- Modify: `go/cmd/gdoc/write.go`, `go/cmd/gdoc/annotate.go`,
  `go/cmd/gdoc/propose_block_test.go`
- Create: `go/cmd/gdoc/unknownfield_test.go`

- [x] The list is decoded into raw entries first, then each entry strictly, so
      a refusal names its entry: `proposals[0]`. One generic reader,
      `strictEntries` in `cmd/gdoc/entries.go`, serves both files.
- [x] `TestAnUnknownFieldIsRefusedWithTheFieldsTheEntryTakes`: a words entry
      carrying `reason` is refused with `proposals[0]`, `"reason"`, and the
      literal sentences of `propose.Fields`. Nothing reaches Google.
- ➕ `TestAnUnknownFieldNamesTheEntryItIsIn`: the stray field in the second
      entry is refused as `proposals[1]`.
- [x] `TestTheFieldListsNameEveryTag`: every JSON tag of `propose.Proposal`
      and of `annotate.Annotation` is a word of its `Fields` sentence.
- [x] The same for `annotate --from`:
      `TestAnUnknownAnnotationFieldIsRefusedWithTheFieldsItTakes`.
- [x] `TestProposeRefusesAnUnknownFieldInABlockEntry` and
      `TestAnnotateRefusesAnUnknownKeyInTheFile` still pass unchanged.
- [x] `git commit -m "fix(cmd): an unknown field is refused with the fields the entry takes"`

### Task 5: a deletion is refused with the way to propose it

**Files:**
- Modify: `go/internal/propose/propose.go`
- Create: `go/internal/propose/deletion_test.go`

- [x] `TestADeletionRefusalSaysToQuoteANeighbour`: the refusal for an empty
      replacement contains "quote a neighbouring word with it, and keep that
      word in the replacement".
- [x] `git commit -m "fix(propose): a deletion is refused with the way to propose it"`

### Task 6: a refusal during the run names its proposal

**Files:**
- Modify: `go/cmd/gdoc/write.go`
- Create: `go/cmd/gdoc/proposalnumber_test.go`

- [x] `TestARunTimeRefusalNamesTheProposalByNumber`: the second of two words
      proposals quotes text that is not in the document; the error opens with
      `proposal 2 (` and the quote, and the first entry reports `sent: true`.
- [x] `git commit -m "fix(cmd): a refusal during the run names its proposal"`

### Task 7: the `--from` summary names `why`

**Files:**
- Modify: `go/cmd/gdoc/commands.go`, `go/cmd/gdoc/testdata/pipe/help.stdout.golden`
- Create: `go/cmd/gdoc/fromsummary_test.go`

- [x] `TestTheFromSummaryNamesEveryRequiredField`: the summary of `propose
      --from` contains `why` and `quoted, replacement`.
- [x] The frozen pipe golden for `help` carries the new sentence. That golden
      is M15's freeze of what a pipe gets, changed on purpose here, in this
      one line.
- [x] `git commit -m "fix(help): the propose --from summary names why"`

### Task 8: propose reads the document once before its first write

**Files:**
- Modify: `go/internal/propose/propose.go`, `go/internal/propose/blockapply.go`,
  `go/cmd/gdoc/write.go`, `go/internal/propose/doc.go`
- Create: `go/cmd/gdoc/proposeread_test.go`

- [x] `propose.ApplyFrom(ctx, s, docID, p, d)` takes the read the caller
      already holds for the first proposal; `nil` reads, as `Apply` does now.
      Every later proposal still reads fresh.
- [x] `TestProposeReadsTheDocumentOnceBeforeTheFirstWrite`: a run of one
      words proposal makes exactly one Docs GET before its batchUpdate.
- [x] `TestEveryLaterProposalStillReadsFresh`: a run of two makes a GET
      between the first proposal's read-backs and the second's write.
- ➕ The fake wire's staged reads drop the second read before the first write:
      `proposeAnswers`, `proposeRun`, `runFixtures.answers`, `blockAnswers`,
      the replace fixture and `lostAnswerRun`. No assertion changed.
- [x] `git commit -m "refactor(propose): one read before the first write, not two"`

#### Part B: settings and the mark

### Task 9: plaintext owns the known marks and what a mark may be

Every other task asks this package, so it comes first. Additive: the old
constants stay until Task 15 moves the last caller.

**Files:**
- Modify: `go/internal/plaintext/plaintext.go`, `go/internal/plaintext/plaintext_test.go`

- [x] `DefaultMark = "⁘"`. `KnownMarks()` returns a new slice of `🤖`, `⁘`,
      `∴` on every call, so no caller can change the list.
- [x] `CheckMark(v string) error` refuses, naming what it found: a line break
      (`\r` or `\n`); markdown, asked of `v + " words"` so a `#` that would
      start a heading in front of the words is caught; one of gdoc's markers
      (`Marker`); more than eight characters, counted as runes. The empty
      string passes.
- [x] `Prefixed(mark, words string) string` is `mark + " " + words`, or
      `words` alone for the empty mark.
- [x] `OpensWithMark(text, inEffect string) bool` steps over leading
      whitespace and is true when the text opens with a known mark or with a
      non-empty mark in effect.
- [x] `CutMark(text, inEffect string) (string, bool)` takes a known mark or
      the mark in effect and one following space off the front, and says
      whether it did.
- [x] Tests: `TestTheKnownMarksAreTheRobotAndTheTwoDots`,
      `TestTheDefaultMarkIsTheFourDot` (both literals),
      `TestCheckMarkRefusesWhatAThreadWouldShowWrong` (one subtest per
      refusal: `"⁘\n"`, `"**"`, `` "`" ``, `"#"`, a marker, `"123456789"`),
      `TestCheckMarkAcceptsEachKnownMarkTheEmptyMarkAndEightCharacters`,
      `TestAnEmptyMarkPrefixesNothing`,
      `TestOpensWithMarkKnowsEveryKnownMarkAndTheMarkInEffect` (a person's
      own mark `"Bot:"` is false with another mark in effect and true with it),
      `TestCutMarkTakesOneMarkAndOneSpace`.
- [x] The package comment gains "# The mark is a setting, and three marks are
      always known", naming the tests above.
- [x] `git commit -m "feat(plaintext): the known marks, the default, and what a mark may be"`

### Task 10: internal/settings, the file and the order

A new package rather than `internal/config`: config answers where a file
lives and is imported by `auth` and `lastcheck`; settings decode YAML and ask
`plaintext` about the mark, and neither belongs under the token's feet.

**Files:**
- Create: `go/internal/settings/doc.go`, `go/internal/settings/settings.go`,
  `go/internal/settings/settings_test.go`
- Modify: `go/internal/config/config.go`, `go/internal/config/config_test.go`,
  `CLAUDE.md` (one "What lives where" row and one task-map row, if Task 1 did
  not add them)

- [x] `config.SettingsPath()` is `settings.yaml` in the config folder:
      `TestSettingsPathIsInTheConfigFolder`.
- [x] `settings.Given` is what the line said: `Config *string`, `Mark *string`.
      `settings.InEffect` is `Mark`, `MarkFrom` (`argument`, `file` or
      `default`) and `File` (the path read, or empty).
- [x] `Resolve(given Given, defaultPath string) (InEffect, error)`: the
      argument, then the file in use, then `plaintext.DefaultMark`. A named
      `--config` that does not exist is refused; the default file absent is
      every default.
- [x] The file is decoded with `yaml.Strict()`. An unknown key is refused by
      name. `mark: ""` is the empty mark. `mark:` with no value is refused,
      saying to write `mark: ""` for the empty mark. A file of zero bytes is
      every default.
- [x] The mark from either source goes through `plaintext.CheckMark`, and the
      refusal names the source: the file and its key, or `--mark`.
- [x] Tests: `TestNoFileIsEveryDefault`, `TestTheFileSetsTheMark`,
      `TestASettingArgumentOverridesTheFile`,
      `TestAConfigFileThatDoesNotExistIsRefused`,
      `TestAnUnknownKeyIsRefusedByName` (`colour: red` names `colour`),
      `TestAnEmptyMarkInTheFileIsTheEmptyMark`,
      `TestAMarkKeyWithNoValueIsRefused`,
      `TestABadMarkIsRefusedNamingWhereItCameFrom` (file and argument),
      `TestMarkFromSaysWhichSourceWon`, `TestTheTokenIsNotASetting`
      (`token: x` and `token_path: x` are unknown keys like any other).
- [x] ➕ `TestAFileOfNoBytesIsEveryDefault`: a zero-byte file is every default
      and is still the file in use, which `InEffect.File` names.
- [x] `doc.go` states the order, the strict decode and why the token is not a
      setting, naming each test.
- [x] `git commit -m "feat(settings): settings.yaml, strict, with the argument first"`

### Task 11: --config and --mark on every command and on gdoc mcp

**Files:**
- Modify: `go/cmd/gdoc/main.go`, `go/cmd/gdoc/commands.go`,
  `go/cmd/gdoc/read.go` (`args` gains the settings), `go/cmd/gdoc/mcp.go`,
  `go/cmd/gdoc/mcptools.go` (passes the session's settings to
  `safeDispatch`), `go/cmd/gdoc/help.go`, `go/cmd/gdoc/helpscreen.go`,
  `go/cmd/gdoc/completion_zsh.tmpl`, `go/cmd/gdoc/completion_bash.tmpl`,
  `go/cmd/gdoc/mcp_test.go`, `go/cmd/gdoc/doc.go`,
  `go/cmd/gdoc/testdata/pipe/help*.stderr.golden`,
  `go/cmd/gdoc/testdata/screens/help*.golden`, `bare*.golden`
- Create: `go/cmd/gdoc/settings_test.go`

- [x] `commands.go` gains `settingFlags`, the two flags with their summaries:
      `--config <file>` "the settings file to use instead of settings.yaml in
      the config folder", and `--mark <text>` "the mark gdoc writes in front
      of its words in a thread; an empty mark writes the words alone, and then
      nothing in the thread says which words are gdoc's". Constants
      `configFlag` and `markFlag` are their one spelling.
- [x] `takeSettings(args) (rest []string, given settings.Given, err error)`
      takes both flags off the line wherever they stand, in both spellings
      (`--mark x`, `--mark=x`), before `--help` is looked for and before
      `mcp` is routed. `--mark ""` and `--mark=` are the empty mark. A flag
      given twice, or with no value after it, is refused naming it.
- [x] `route()` calls it, then `settings.Resolve`. A refusal on a command is
      one envelope with exit 1; on `gdoc mcp` it is one line on stderr and exit
      1, as `checkMCPArgs` refuses today. `checkMCPArgs` itself is unchanged.
- [x] `run`, `safeDispatch` and `dispatch` take the `settings.InEffect`, and
      `dispatch` puts it on `args` for every command. The MCP session resolves
      once at start and hands the same value to every tool call.
- [x] Help prose and the help screen list the two flags once, under "Every
      command takes", from `settingFlags`. Both completion scripts offer them
      after any command, `--mark` as opaque text like `--quote`.
- [x] Tests: `TestEveryCommandTakesTheSettingFlags` (each table entry run with
      both flags reaches its own refusal or answer, never "is not a flag this
      command takes"), `TestTheSettingFlagsAreTakenAnywhereOnTheLine`,
      `TestAnEmptyMarkArgumentIsTheEmptyMark` (both spellings),
      `TestASettingFlagGivenTwiceIsRefused`,
      `TestABadSettingsFileIsOneEnvelope`, `TestAConfigFileThatDoesNotExistIsRefusedOnTheLine`,
      `TestMcpTakesTheSettingFlagsAndNothingElse` (replaces
      `TestMcpTakesNoWordsAndNoFlags`: `gdoc mcp --mark ∴` starts, `gdoc mcp
      --stdio` is still refused), `TestTheMcpSessionHandsItsSettingsToEveryCall`,
      `TestEverySettingFlagIsInHelpAndInBothScripts`,
      `TestTheMarkFlagSaysWhatAnEmptyMarkCosts`.
- [x] Goldens: the help stderr goldens and the help and bare screens gain the
      "Every command takes" lines. Regenerate those files only, and record in
      a ➕ note which files moved and that each diff is those lines alone.
- [x] `cmd/gdoc/doc.go`: a section "Two setting flags, taken off the line
      first" naming the tests; the mcp bullet (around line 110) and "The
      extension has no settings" (around line 626) say the line takes the two
      setting flags and the extension passes none.
- [x] ➕ The two flags are listed once per page rather than once per run: the
      grouped table carries the section once, and each command's own page
      carries it too, because a page is the whole of what a reader who typed
      `gdoc help publish` reads. The heading "Every command takes" is what
      keeps them from reading as that command's own.
- [x] ➕ `settings_test.go` adds `TestASettingFlagWithNoValueIsRefused` and
      `TestTheDefaultSettingsFileIsReadFromTheConfigFolder`. The first holds
      the two refusals the plan named together with the one it did not,
      `--mark --config f`, which would otherwise write "--config" in front of
      gdoc's words; the second holds that the default path is the config
      folder's `settings.yaml`.
- [x] ➕ `dispatching` is `dispatch` behind a variable in `main.go`, as
      `login`, `openSession` and `serveMCP` already are, so a test can read the
      line and the settings that reached it without a command running. Nothing
      in the binary assigns it.
- [x] ➕ Goldens regenerated: `testdata/pipe/help.stderr.golden`,
      `help-publish.stderr.golden`, `help-comments.stderr.golden`,
      `bare.stderr.golden`, and all fifteen of `testdata/screens/help-*.golden`
      and `bare-*.golden`. The stdout goldens did not move. The pipe diffs are
      the "Every command takes" lines alone. The screen diffs are those lines
      plus the name column, which widened by three cells in every box so that
      `--config <file>` and the rule holding it clear it: `settingColumn` is
      what both `commandColumn` and `flagColumn` now measure against, and
      without it the one rule would have lost the joint the rules above it draw.
- [x] ➕ `TestMcpTakesNoWordsAndNoFlags` is gone, replaced as the plan says;
      `doc.go` and `manifest_test.go` name the new test instead.
- [x] ➕ `TestATextFlagIsOpaqueToBothScripts`,
      `TestTheZshScriptNamesEveryCommandAndEveryFlag` and
      `TestTheBashScriptNamesEveryCommandAndEveryFlag` read the setting flags
      beside the table's own, through one new `flagKinds` helper. No assertion
      is loosened: the text and file arms are still read a flag at a time.
- [x] `git commit -m "feat(cmd): --config and --mark on every command and on mcp"`

### Task 12: help --json carries the settings in effect

**Files:**
- Modify: `go/cmd/gdoc/help.go`, `go/cmd/gdoc/help_test.go`,
  `go/cmd/gdoc/testdata/pipe/help*.stdout.golden`, `go/cmd/gdoc/doc.go`

- [x] `helpReport` gains `Settings helpSettings` with `json:"settings"`:
      `{"mark": "...", "mark_from": "argument" | "file" | "default"}`. Always
      present, the empty mark included: `"mark": ""` is a fact.
- [x] Tests: `TestHelpCarriesTheSettingsInEffectAndWhereEachCameFrom` (one
      subtest per source, values as literals),
      `TestHelpCarriesTheEmptyMarkAsAnEmptyString`,
      `TestHelpForOneCommandCarriesTheSettingsToo`.
- [x] The three help stdout goldens gain the field. Regenerate those three and
      nothing else; a ➕ note says so.
- [x] ➕ Regenerated, and only those three: `help.stdout.golden`,
      `help-publish.stdout.golden` and `help-comments.stdout.golden`, each
      gaining `"settings":{"mark":"⁘","mark_from":"default"}` and nothing else.
      The three stderr goldens did not move, because the two flags are in no
      entry of the object and the words a person reads already name them.
- [x] ➕ `cmdHelp` takes the `settings.InEffect`, so both of its call sites
      hand it over: `dispatch`'s `--help` reading in `main.go` and the table's
      own `help` entry in `commands.go`, which reads it off `a.set`.
- [x] ➕ The test helper reading the field is `settingsInObject`, not
      `helpSettings`, which is the name the new struct in `help.go` takes.
- [x] `git commit -m "feat(help): the settings in effect, and where each came from"`

### Task 13: propose and annotate write the mark in effect

**Files:**
- Modify: `go/internal/propose/propose.go`, `go/internal/propose/blockbatch.go`,
  `go/internal/propose/blockapply.go`, `go/internal/propose/doc.go`,
  `go/internal/annotate/annotate.go`, `go/internal/annotate/doc.go`,
  `go/cmd/gdoc/write.go`, `go/cmd/gdoc/annotate.go`, and the tests of both
  packages that state the sent comment
- Create: `go/internal/propose/mark_test.go`, `go/internal/annotate/mark_test.go`

- [x] `Batch`, `BlockBatch`, the apply functions and `annotate.Body` take the
      mark and write `plaintext.Prefixed(mark, why)`. The read-backs compare
      against that same string. `cmd/gdoc` passes `a.settings.Mark`.
- [x] The reason check refuses a reason opening with a known mark or with the
      mark in effect (`OpensWithMark`), naming the mark it found and saying
      gdoc adds it.
- [x] `propose.Prefix`, `propose.Robot`, `annotate.Prefix` and
      `annotate.Robot` are removed.
- [x] Tests: `TestProposeWritesTheMarkInEffect` and
      `TestAnnotateWritesTheMarkInEffect` (with `"∴"`),
      `TestAnEmptyMarkWritesTheWordsAlone` (one subtest per writer),
      `TestAReasonOpeningWithAnyKnownMarkIsRefused` (one subtest per known
      mark and the mark in effect, in both packages),
      `TestTheReadBackComparesTheMarkThatWasSent`.
- [x] Existing tests that state `"🤖 "` in a sent comment pass the mark
      explicitly as `"🤖"`; none is loosened.
- [x] ➕ `OpensWithMark` cannot name the mark it matched, and the refusal has to
      name the character the reason carries rather than the one this run writes.
      So `plaintext.MarkOpening(text, inEffect) (string, bool)` was added and
      `OpensWithMark` and `CutMark` now ask it, which keeps the reading side one
      loop instead of three: `TestMarkOpeningNamesTheMarkItFound`, with the
      package comment naming it beside the other two.
- [x] ➕ `Proposal.Check`, `Annotation.Check` and `checkWhy` take the mark, so
      `cmd/gdoc`'s `readProposals`, `readAnnotations`, `annotationsFor`,
      `runPropose` and `runAnnotate` take it too and read it off `a.set.Mark`.
      The whole-file check still happens before a session is opened.
- [x] ➕ The cmd tests over the recorded comment fixtures, which carry the
      robot, say so through `runJSONSet(t, robotInEffect(), ...)`: a new
      `robotInEffect` beside `defaultsInEffect`, and `runJSONSet` /
      `runJSONSetCtx` beside `runJSON`. Seven tests moved to it; nothing was
      loosened. The live tests name `liveMark = "⁘"`, the default a real write
      now carries.
- [x] ➕ `chatWith` in the tests was building a session with no settings, so a
      chat write sent the words with no mark at all while the CLI sent the
      default: `TestTheSameAnswerAsTheCLI` caught it. It now carries
      `defaultsInEffect()`, as `newMCPChat` does in production, through a new
      `chatWithSettings`.
- [x] `git commit -m "feat(propose,annotate): the mark in effect, or the words alone"`

### Task 14: reply adds the mark itself

**Files:**
- Modify: `go/internal/reply/reply.go`, `go/internal/reply/reply_test.go`,
  `go/cmd/gdoc/write.go`, `go/cmd/gdoc/write_test.go`,
  `go/cmd/gdoc/commands.go` (reply and annotate summaries),
  `go/cmd/gdoc/mcptools.go` (reply schema text), `release/mcpb/manifest.json`,
  `go/cmd/gdoc/chatheader.md`, `go/cmd/gdoc/doc.go`,
  `go/cmd/gdoc/testdata/pipe/help*.golden`, `testdata/screens/help*.golden`

- [x] `reply.Check(words, mark)` refuses an empty body, markdown and a marker,
      asked of the words alone. It no longer requires a prefix.
- [x] `reply.Post(ctx, s, docID, commentID, words, mark)` sends
      `plaintext.Prefixed(mark, words)` and reads that string back.
- [x] A body that opens with a known mark or the mark in effect has it taken
      off with `CutMark` and gets the mark in effect instead, once, with a
      warning: "the reply opened with 🤖, and gdoc adds the mark itself; it was
      written once, as ⁘". This holds for v2.11.0; the warning says so.
- [x] `reply.Prefix` is removed.
- [x] The reply and annotate summaries say "under your mark" instead of
      "under the robot prefix", and `manifest.json` follows them word for
      word. The reply schema's `body` says: "The words of the reply. gdoc puts
      its mark in front of them, so do not type one. Markdown is refused."
- [x] `chatheader.md` stops telling the model to draft a `🤖` reply.
- [x] Tests: `TestReplyAddsTheMarkInEffect`,
      `TestAnEmptyMarkWritesTheWordsAlone` (the reply subtest),
      `TestABodyOpeningWithAKnownMarkIsWrittenOnceWithAWarning` (one subtest
      per known mark, with leading space and newline),
      `TestCheckRefusesEachBadShapeByName` keeps its markdown, marker and empty
      cases. `TestReplyRefusesABodyWithoutTheRobot` and
      `TestCheckAcceptsAPlainRobotReply` are replaced by
      `TestReplyAcceptsABodyWithNoMark`; the `reply/doc` comment and
      `plaintext`'s "The two writers own the mark differently" section are
      rewritten to "all three writers add the mark".
- [x] Goldens: the help goldens move by the two summaries. A ➕ note names them.
- [x] ➕ The three mark tests went into a new `go/internal/reply/mark_test.go`,
      beside `propose/mark_test.go` and `annotate/mark_test.go`, with
      `TestAnEmptyMarkStillTakesATypedMarkOff` for the corner the two decisions
      make between them and `TestTheReadBackComparesTheMarkThatWasSent` for the
      read-back.
- [x] ➕ `signed(body, mark)` is the one room that puts the mark on, and the
      warning it returns names the mark the body carried rather than the one
      this run writes. `Post` reads it once, so the read-back compares the
      string that went out.
- [x] ➕ `TestPostSendsTheBodyVerbatimToTheRepliesURL` became
      `TestPostSendsTheWordsUnderTheMarkToTheRepliesURL`: nothing but the mark
      is added, and "verbatim" stopped being true.
- [x] ➕ `TestTheReplyCardAsksForTheMarkItself` became
      `TestTheReplyCardSaysGdocAddsTheMark`, which also fails if the card names
      any known mark: the mark is a value this run was given, and a card naming
      one is wrong for every colleague who set another.
- [x] ➕ The cmd-level `TestReplyRefusesABodyWithoutTheRobot` was replaced by
      `TestReplyRefusesAMarkerBeforeAnyRequest`, so the route still has a test
      that nothing reaches Drive for a body `Check` refused.
      `TestReplyPostsTheBodyAndReadsTheThreadBack` and
      `TestReplyNeverCutsItsReadBack` send the words and ask for
      `robotInEffect()`, which is the mark the recorded thread carries.
- [x] ➕ Chat's `heldBody` went: a held reply's body is `heldRun`, the words
      alone, and `heldSent` is what reaches the wire, `"⁘ " + heldRun`.
- [x] ➕ `internal/reply/reply.go` came off `markWriters` in
      `go/boundary/boundary_test.go`: it owns no prefix check now, it asks
      `plaintext.CutMark`.
- [x] ➕ Goldens regenerated: `testdata/screens/help-{44,60,80,100}.golden`,
      `help-80-truecolor.golden` and the five `bare-*` of each, by deleting
      them and running the test twice. `testdata/pipe/help.stderr.golden`,
      `help.stdout.golden` and `bare.stderr.golden` moved by the two summaries
      alone, so those three were edited in place.
- [x] `git commit -m "feat(reply): gdoc adds the mark, as propose and annotate do"`

### Task 15: by_gdoc is a known mark or the mark in effect

**Files:**
- Modify: `go/internal/comments/comments.go`, `go/internal/comments/wait.go`,
  `go/internal/comments/doc.go`, `go/internal/comments/comments_test.go`,
  `go/internal/restyle/survey.go`, `go/cmd/gdoc/read.go`,
  `go/cmd/gdoc/mcptools.go`, `go/internal/plaintext/plaintext.go`,
  `go/boundary/boundary_test.go`, `go/boundary/doc.go`

- [ ] `comments.Threads(raw, d, mark)` sets `Reply.ByGdoc` with
      `plaintext.OpensWithMark(content, mark)`. `WaitOptions` gains `Mark`.
      `read.go` and `mcptools.go` pass the mark in effect; `restyle.Survey`
      passes `""` with a comment saying the survey reads no `by_gdoc`.
- [ ] `mcpRemarks` asks `OpensWithMark` of a thread's opening comment with the
      session's mark.
- [ ] `plaintext.Robot`, `plaintext.Prefix` and `OpensWithRobot` are removed.
- [ ] `TestByGdocIsTrueOnlyForAReplyOpeningWithTheRobot` becomes
      `TestByGdocIsTrueForAKnownMarkOrTheMarkInEffect`: true for 🤖, ⁘ and ∴
      whatever the mark in effect, true for `"Bot:"` only when it is the mark
      in effect, false for every reply under the empty mark that opens with
      none of them.
- [ ] `TestTheRobotMarkIsReadInOnePlace` becomes `TestTheMarkIsReadInOnePlace`:
      its literal is the three known marks, the call it looks for is
      `plaintext.OpensWithMark`, `markWriters` keeps `propose.go`,
      `annotate.go` and `reply.go` (reply now cuts a known mark off a body),
      and it still fails in both directions. `boundary/doc.go` "The robot mark
      is read in one place" says the same with the new names.
- [ ] `git commit -m "feat(comments): by_gdoc reads the known marks and the mark in effect"`

### Task 16: reply --body, and chat writes no file for a reply

**Files:**
- Modify: `go/cmd/gdoc/commands.go`, `go/cmd/gdoc/write.go`,
  `go/cmd/gdoc/mcptools.go`, `go/cmd/gdoc/write_test.go`,
  `go/cmd/gdoc/mcpfiles_test.go`, `go/cmd/gdoc/doc.go`,
  `go/cmd/gdoc/testdata/pipe/help*.golden`, `testdata/screens/help*.golden`
- Create: `go/cmd/gdoc/replybody_test.go`

- [ ] `reply` takes `--body <text>` (`kindText`) and `--body-file <file>`,
      both `needEither`. Exactly one is given: both, or neither, is refused
      naming the two flags. The `--body` value is used as typed, with the same
      trailing-whitespace trim `readBody` applies to a file.
- [ ] Chat's `body` maps to `--body` with no file. The dash refusal is not
      asked of `body`: the value goes on the line joined as `--body=<value>`,
      which the parser cannot read as a flag, and a reply opening with "- " is
      plain text the binary accepts. Every other on-the-line argument keeps
      the dash refusal.
- [ ] Tests: `TestReplyTakesTheBodyOnTheLineOrInAFile`,
      `TestReplyRefusesBothBodyAndBodyFile`,
      `TestReplyRefusesNeitherBodyNorBodyFile` (replaces
      `TestReplyNeedsABodyFile`), `TestAChatReplyWritesNoFile` (no call
      directory is made), `TestAChatReplyOpeningWithAHyphenIsSent`,
      `TestEverySchemaPropertyMapsToAWordOrFlagAndBack` passes with `body`
      mapped to `--body`, `TestEveryRequiredFlagIsOneTheCommandRefusesToRunWithout`
      passes with the two marked `either`.
- [ ] Goldens: the help goldens move by the reply usage line. A ➕ note names them.
- [ ] `git commit -m "feat(reply): --body on the line, and chat sends it there"`

### Task 17: chat's robot_not_ours fact is removed

**Files:**
- Modify: `go/internal/chat/facts.go`, `go/internal/chat/doc.go`,
  `go/internal/chat/rules.go` (comment), `go/internal/chat/facts_test.go`,
  `go/internal/chat/ledger.go` (comment), `go/internal/chat/ledger_test.go`,
  `go/cmd/gdoc/mcpview.go`, `go/cmd/gdoc/chatheader.md`,
  `go/cmd/gdoc/mcpledger_test.go`, `go/cmd/gdoc/mcplabel_test.go`,
  `go/cmd/gdoc/mcpguide_test.go`, `go/cmd/gdoc/mcplanded_test.go`

- [ ] `Facts.RobotNotOurs` and `robotNotOurs` go. `FactsOf` no longer takes
      `OwnReplies` if nothing else in it asks; `OwnReplies` and
      `Ledger.Wrote` stay for the Dictated rule.
- [ ] `TestTheFactNamesAreTheLiterals` and the list in `mcplabel_test.go`
      name five facts. `TestRobotNotOursAsksTheRecordOfWhatThisProcessWrote`
      and `TestAReplyThisSessionWroteIsNotRobotNotOurs` are deleted;
      `TestRobotNotOursUsesTheLedger` becomes `TestWroteAnswersFromTheLedger`.
- [ ] `TestNoFactNamedRobotNotOursReachesTheModel`: a comments answer through
      `mcpReadAnswer` carries no `robot_not_ours` key anywhere.
- [ ] `chat/doc.go` "Six facts" becomes five, and the Flagged thread bullet
      loses its sentence about `robot_not_ours`. `chatheader.md` drops it from
      the list of facts.
- [ ] `git commit -m "refactor(chat): robot_not_ours goes, the ledger stays for Dictated"`

### Task 18: the documents of Part B

**Files:**
- Modify: `README.md`, `docs/guide/writing.md`, `docs/guide/chat.md`,
  `docs/guide/how-it-works.md`, `go/internal/plaintext/plaintext.go`
  (package comment), `go/internal/settings/doc.go`, `go/cmd/gdoc/doc.go`

- [ ] `README.md`, staying at 200 lines: line 98 says each reply opens with
      gdoc's mark, ⁘ unless you set another; one short paragraph says where
      `settings.yaml` lives, that `--mark` and `--config` override it, and
      what an empty mark costs: nothing in the thread says which words are
      gdoc's, and a later review reads gdoc's own replies as a person's. Take
      lines out elsewhere rather than past the ceiling.
- [ ] `docs/guide/writing.md`: `reply` no longer needs the mark typed; `--body`
      beside `--body-file`; propose and annotate refuse a reason opening with
      any known mark.
- [ ] `docs/guide/chat.md:75`: the mark in effect, not 🤖.
- [ ] `plaintext`'s package comment: "The mark is the only record of authorship
      there is" says the mark is a setting, the three known marks are always
      read, and with the empty mark there is no record at all. Each rule names
      its test.
- [ ] `cd go && go test -race ./boundary/` passes (task map, ceilings,
      package comments).
- [ ] `git commit -m "docs: the mark is a setting, and what the empty mark costs"`

### Task 19: verify acceptance, and M16 completed

**Files:**
- Modify: `docs/v2/PLAN.md`

- [ ] `make test`, `make vet`, `GOOS=darwin go vet ./...` and `make dist` pass.
- [ ] Every Validation Command for Part B gives the answer it states; record
      the counts here as ➕ notes.
- [ ] Every row of the spec's Decisions table for Parts A and B names a task
      above that built it.
- [ ] `git diff main...HEAD -- go/internal/guard/policy.go` is empty, and
      `go/internal/auth/` is untouched: the token stays a file in M16.
- [ ] `PLAN.md`: M16 under Done, with this plan's path, and M17 named as
      next. Move this plan and its spec to `docs/plans/completed/`, fixing
      every link to both.
- [ ] `git commit -m "docs(v2): M16, completed"`

## Post-Completion

- Nail's acceptance, then the tag sitting and `make tag VERSION=v2.11.0`, in one
  sitting, because a skill that stops typing 🤖 is refused by v2.10.0:
  `skills/gdoc-review/` (`SKILL.md`, `review.md`, `live.md`, `propose.md`) and
  `skills/gdoc-align/SKILL.md` stop telling a session to type 🤖 and say gdoc
  adds the mark; `go/cmd/gdoc/review.md` is copied from the skill's again
  (`TestTheEmbeddedCoreIsTheSkillsCore`); every skill's `needs` line moves to
  v2.11.0. v2.11.0 still accepts a body opening with a known mark, with a
  warning, so the order is safe either way for one release.
- v2.11.0 is unsigned and published by the workflow as today.
