# gdoc v2 Milestone 32f: the firm's mark beside DOCUMENTS. Plan

Tasks: 7
Review: ui

2026-10-29. Nail moved the firm's mark from the foot of the panel to the
right end of the DOCUMENTS label, and after the visual designer gave the
position and the lead drew it on the v2.25.0 render he answered: "Да,
логотип стоит правильно. Запиши это как решение вместо «внизу панели» и
готовь веху." The decision is `docs/v2/DECISIONS.md`, 2026-10-29, "The
firm's mark beside DOCUMENTS", with the trust guardian's sentence on it, and
the spec paragraph of the same name, both on `main` before this run. The
app reads one more file for it, so this is the milestone the fifth look's
entry promised: a Go change, a Swift change, a line in `CLAUDE.md` and the
decision. One commit per task. Where the code, the spec and this plan
disagree, the task stops and says so.

## Principles

Serves 3 of `PRINCIPLES.md` (the firm's documents under the firm's mark, in
the house look a reader trusts) and 4 (nothing is built for a machine that
has no picture: no placeholder, no fetch, no watcher). Serves principle 5 of
`docs/design/2026-10-17-presence-design-principles.md`, facts and not
promises: the mark is drawn from a file that exists and from nothing else.

Strains one invariant in `CLAUDE.md`, "The menu bar app runs six argv
lists, signals one wait and reads three things", which becomes four things
by this decision: the fourth is the one path `help --json` names. The
guardian signed that route on 2026-10-28 and signs this entry before the
run; the two routes it vetoed, the app reading the cache by its own copy of
the Drive id and the binary handing bytes over, are not in this plan. No
argv list, signal, URL, grant or presence field changes.

## Context (from discovery)

Checked against `main` at 03843ca. M32d's branch
(`gdoc-v2-m32d-the-stuck-cards-open-icon`, six commits) touches
`PanelView.swift`, `PanelModel.swift`, `RowTests.swift`, `Scenes.swift`,
`StateTests.swift`, the guide and the principles file, the same files this
plan touches, and appends its amendment where Task 7 appends. So this run
starts only after M32d has merged to `main`, and is cut from that `main`;
the line numbers below were read before the merge and may move by a few
lines. Task 7's amendment then goes after M32d's section.

- **The help today.** `helpReport` (`go/cmd/gdoc/help.go`, about lines 22
  to 38) carries `commands`, `settings` and `update`. `cmdHelp` (about line
  122) builds it twice: the `--brief` form at about line 134 and the full
  form at about line 154. The app asks for the full form,
  `["help", "--json"]` (`presence/Sources/PresenceApp/Runner.swift:38`);
  skills open on `--brief`, whose size is pinned under `briefCeiling`, 600
  bytes (`helpjson_test.go`, about line 158), and every per-command help
  (`help publish --json`) is what a skill reads. The pipe goldens under
  `go/cmd/gdoc/testdata/pipe/` are read only by the rule at the top of
  `pipe_test.go`; they run `help`, `help publish` and `help comments`
  without `--json` (`pipe_test.go:114` to 132), so a field that only the
  `--json` form carries leaves them byte for byte. The house file the binary
  carries names the logo's Drive file: `house.Load()` and
  `(*Config).LogoDrive()` (`go/internal/house/house.go`, about lines 573
  and 611), the id at `house.yaml:1160`. The cache lives at
  `config.AssetPath(id)` (`go/internal/config/config.go:162`),
  `assets/<id>` under the config dir, written by `internal/asset` after a
  build or publish; `help` never touches it (`TestHelpNeverReadsTheToken`
  in `helpscreen_test.go:229` is the shape of a help test that proves a
  non-read). `go/cmd/gdoc/doc.go` names the test for every rule it states.
- **What the app reads from the help.** `HelpFacts`
  (`presence/Sources/PresenceCore/HealthFacts.swift:33`) decodes `settings`
  and `update` at about lines 53 to 66; its tests are
  `PresenceCoreTests/HealthFactsTests.swift`. `HealthPoller.askedHelp`
  (`presence/Sources/PresenceApp/HealthPoller.swift`, about line 201) puts
  the decoded facts into `HealthReading` (about line 78) and
  `applied(to:)` (about line 170) writes the reading into `Facts`
  (`presence/Sources/PresenceUI/Facts.swift:220`, init at 292). Two more
  rooms rebuild `Facts` field by field and would drop a new field that
  defaults to nil: `UpdateAction.applied(to:)`
  (`presence/Sources/PresenceApp/UpdateAction.swift:162`), which
  `Session.recompose()` runs after the poller's (`Session.swift:445` to
  446), and `askedAuth` in the poller, which builds `HealthReading` twice.
  The poller's test is `PresenceAppTests/HealthPollerTests.swift`
  (`testThePolledFactsReachTheFacts`, about line 118; its `FakeGdoc` at
  about line 188 answers `help` with one fixed JSON string at about line
  218, and a poll with `.changed` always asks). `UpdateActionTests.swift`
  covers the other. The model is folded from the facts in `PanelModel`
  (`PanelModel.swift`, about line 392), and the fold always appends the
  DOCUMENTS section (about line 435), the "no documents" line included,
  under the centred empty state or not.
- **What the app reads from disk today, in three spellings.** The record
  (`LogTail.swift:155`, `Darwin.open(..., O_RDONLY | O_NOFOLLOW)` and
  `pread`), its plist (`BundleWatch.swift:54` and `Relauncher.swift:170`,
  `Data(contentsOf:)`) and the notes' front matter (`HubIndex.head(of:)`,
  `HubIndex.swift:265`, `FileHandle(forReadingFrom:)`). No test lists
  those files; the invariant in `CLAUDE.md` lines 182 to 192 does, and
  names `testAFileThatMerelyMentionsTheIdIsNotTheNote` for the hub half.
  The guardian's veto of 2026-10-29 was on a scan that named two of the
  three spellings and none of the image loaders; Task 2's scan names them
  all. The source-rule tests are
  `PresenceCoreTests/SourceRulesTests.swift`;
  `testProcessAppearsOnlyInTheRunner` (line 92) is the shape of "only this
  file does X".
- **Where the label is drawn.** `SectionLabel`
  (`presence/Sources/PresenceUI/Components/SectionView.swift:19` to 56): an
  `HStack` of the uppercased text and a `Spacer`, in `SectionLabel.ink`
  (tertiary on dark, secondary on light), 12 pt horizontal padding, 28 pt
  minimum height, probed as `"section-label"`. `SectionView` (line 58)
  draws it over the rows; `PanelView.section(_:)` (`PanelView.swift:98`)
  calls it with `placed.section.label`, and `PanelSection.label`
  (`PanelModel.swift`, about line 316) is "Now", "Documents" or
  "Settings". The content edge every icon ends on is 340 pt in the
  panel's own coordinates, after `PanelView`'s 8 pt padding; a label or a
  row drawn alone at `Grid.panelWidth` ends its ink at `Grid.panelWidth -
  Grid.rowInset`, 348, which is what
  `testAnActionStripMeetsTheColumnEveryValueEndsOn` asserts and what the
  label's own 12 pt padding gives the mark for free. The grid's numbers
  are pinned by `ThemeTests.testTheGridIsTheScreens` (about line 173) and
  the palette's literals by `testTheDarkPaletteIsTheScreensFileLiterals`
  and `testTheLightPaletteIsTheScreensFileLiterals`: no new colour.
- **The designer's numbers.** Height 14 pt (the wordmark's lowercase body
  is then the label's cap height, 8 pt); width by the file's aspect, 38.5
  pt at 2.75 to 1; right edge at 340 pt; centred on the 28 pt row, top at
  7 pt, which puts its baseline within a device pixel of the label's at 18
  pt; the label's own ink; drawn exactly where a DOCUMENTS label is
  drawn, the "no documents" line included, in both appearances, with no
  rule of its own: a machine that has the file has built a document, so a
  first run has nothing to draw; not a press, no tooltip.
- **The renders.** `PresenceUITests/Scenes.swift` builds every scene's
  `Facts` through one `Scenes.machine(...)` (about line 214, the `Facts(`
  at 231), and `Scenes.all` is a nonisolated `static let`, so a helper it
  calls cannot be `@MainActor`. `RenderTests.swift` writes the PNGs and
  `index.html` under `$PRESENCE_SHOTS` (`html(_:)` at about line 735, the
  lead paragraph at 764). `Shots.swift` holds the measuring helpers, all
  `@MainActor`: `inkEnds` (line 187), `inkBand` (216, which throws when it
  finds no ink), `pixel` (164, which frames the view in a `side` square, so
  a 28 pt label in a 360 square sits at y 166 to 194), `heaviestPixel`
  (54, one alpha for the whole view), `probes` (117, sizes only) and
  `probeList` (131, frames). `inkEnds` and `inkBand` count any pixel over
  0.05 alpha, so over a whole panel, whose plate is opaque, they say
  nothing; a probe does. The repository carries no logo by the decision
  of 2026-10-16, so a render draws a stand-in shape the test makes, never
  the firm's file, and the index says so.

### What the lead changes on `main` before the run

Done and committed before the launch: the decision entry with the
guardian's sentence, its register row, the fifth look's row marked
superseded in its one clause, and the spec paragraph "The firm's mark
beside DOCUMENTS". Nothing else. The launch itself waits for M32d's merge.

### Out of scope

- The app reading the cache by its own copy of the Drive id, or the binary
  handing bytes over: vetoed by the guardian on 2026-10-28.
- `--brief`, a command's help or the flagless help carrying the path: the
  skills do not draw, the brief's size is pinned and the pipe goldens are
  read only.
- A placeholder, a "no logo yet" line, a file watcher, a fetch of the logo
  by the help or by the app, a press or a tooltip on the mark.
- The firm's file in git, in a fixture or in a render.
- M32d's files, and anything the M32e branch touches (`house.yaml`'s
  anchor, `drift`).

## Development Approach

- TDD: the failing test first, in the file the Context names, then the
  smallest change that passes it.
- One commit per task, message in the house form, no attribution lines.
- Nothing is loosened: the brief's ceiling, the palette literals, the grid
  list and the source rules stay as they are, and the new rule gets its own
  test.
- Go tests run by package, never `make test`: the session's environment
  carries the live variables, and `make test` would write to Drive.
- Plain English in every comment and caption, no em dashes.

## Validation Commands

```
cd go && go test -race ./cmd/gdoc/ ./boundary/
make vet
make app-lint
make app-test
make app
```

Task 5 only:

```
PRESENCE_SHOTS="$HOME/gdoc-m32f-board/shots" make app-test
```

## Progress Tracking

Tick each box as it is done. A task ends with its commit.

## Tasks

### Task 1: the help names where the logo is cached

**Files**

- `go/cmd/gdoc/help.go`
- `go/cmd/gdoc/helpjson_test.go`
- `go/cmd/gdoc/doc.go`

**Steps**

- [ ] In `helpjson_test.go`, a new `TestHelpNamesWhereTheLogoIsCached`:
      `t.Setenv("GDOC_CONFIG_DIR", t.TempDir())`, run
      `help --json`, read `data.logo_path` and require it equal to
      `config.AssetPath("1J3uh6poVAneP_V_Pc5FuNI4-Gahlab0s")`, the house
      file's own id as a literal; require that no file exists at that path
      after the run and that nothing under `assets/` was created, so the
      help named the path and touched nothing. In the same test, run
      `help --brief --json`, `help publish --json` and bare `help` on a
      pipe, and require `logo_path` absent from each: the field belongs to
      the one argv the app runs, `["help", "--json"]`
      (`Runner.swift:38`), and to nothing a skill reads. Run it: red.
- [ ] In `help.go`, `helpReport` gains `LogoPath string
      \`json:"logo_path,omitempty"\`` with a comment in the file's own
      voice: where `internal/asset` keeps the house logo, named so the menu
      bar app can draw it from that one file; carried by bare `help --json`
      alone, because that is the app's argv, and absent under `--brief`,
      under any command word and without the flag, so no skill's context
      and no golden carries a home path; never fetched, stat-ed or read by
      the help; name the test. A small helper beside `settingsAsked`
      returns the path from `house.Load()` and `cfg.LogoDrive()` through
      `config.AssetPath`, and the empty string on any error, because the
      help is an answer about the tool and a broken house file is not its
      question. The full form's return at about line 154 sets it when
      `json && len(words) == 0`, and nowhere else.
- [ ] `TestHelpBriefIsAFewHundredBytes`, `TestHelpNeverReadsTheToken` and
      `TestHelpOnAPipeIsTodaysTextByteForByte` green, untouched, and no
      file under `testdata/pipe/` changes: the goldens run without
      `--json`. The help screen prints nothing new.
- [ ] `go/cmd/gdoc/doc.go`: one rule in the help's section, that the full
      help names the logo's cache path and reads nothing for it, naming
      `TestHelpNamesWhereTheLogoIsCached`.
- [ ] `cd go && go test -race ./cmd/gdoc/ ./boundary/` and `make vet`
      green.
- [ ] Commit: `feat: the help names where the logo is cached`.

### Task 2: the one read, held to one file and one shape

**Files**

- `presence/Sources/PresenceCore/HealthFacts.swift`
- `presence/Sources/PresenceCore/MarkFile.swift` (new)
- `presence/Tests/PresenceCoreTests/HealthFactsTests.swift`
- `presence/Tests/PresenceCoreTests/MarkFileTests.swift` (new)
- `presence/Tests/PresenceCoreTests/SourceRulesTests.swift`

**Steps**

- [ ] In `HealthFactsTests.swift`, `testHelpNamesTheLogoPathAndItMayBeAbsent`:
      a `data` object with `logo_path` decodes to `HelpFacts.logoPath` equal
      to it; one without decodes with `logoPath` nil and nothing else
      changed; one whose `logo_path` is a number fails the whole decode,
      which is the file's strict rule and is stated in the test. Red.
- [ ] `HelpFacts` gains `public let logoPath: String?`, read the way
      `markFrom` is, with its one-line comment. Green.
- [ ] `MarkFileTests.swift`, new, over temp files: a 200-byte file that
      opens with the eight PNG signature bytes comes back as those bytes; a
      file opening with JPEG's bytes comes back nil; a PNG of exactly
      1,048,576 bytes comes back and one of 1,048,577 comes back nil; a
      directory, a path that does not exist and a symbolic link to a
      passing PNG each come back nil. Red.
- [ ] `MarkFile.swift`, new in `PresenceCore`: a package comment in the
      house voice (what it reads, the three checks, why a link is refused:
      the file named is the file read), `public enum MarkFile` with
      `public static let ceiling = 1_048_576`, the signature bytes as a
      literal, and `public static func read(at path: String) -> Data?`.
      It opens the path once with `Darwin.open(path, O_RDONLY | O_NOFOLLOW)`
      as `LogTail.swift:155` does, so a link is refused by the kernel and
      not by a stat that a swap could outrun; `fstat` on that descriptor
      must say a regular file of at most `ceiling` bytes; it then reads at
      most `ceiling + 1` bytes through the descriptor, closes it, and
      checks the signature. Any failure is nil, with no log line: a
      machine with no picture is not an error. Green.
- [ ] In `SourceRulesTests.swift`, `testAByteIsReadOnlyInTheFiveReaders`:
      walk every Swift file under `Sources` as `testProcessAppearsOnlyInTheRunner`
      does. Five files may read a byte, each named with one line on what
      it reads: `LogTail.swift` (the record), `HubIndex.swift` (a note's
      head), `BundleWatch.swift` and `Relauncher.swift` (the plist) and
      `MarkFile.swift` (the mark). Every read spelling is refused in any
      other file: `Data(contentsOf`, `NSData(contentsOf`,
      `FileHandle(forReadingFrom`, `FileHandle(forReadingAtPath`,
      `.contents(atPath`, `String(contentsOf`, `O_RDONLY`, `pread(` (not
      `read(`: `Runner.swift:530` reads a pipe), `NSImage(contentsOf`,
      `NSImage(contentsOfFile`,
      `NSImage(byReferencing`, `CGImageSourceCreateWithURL` and
      `AsyncImage`, spelled in pieces where the test file would otherwise
      name itself. The floor the other rules keep: each of the five files
      must contain at least one spelling, or the test guards nothing. A
      sixth file fails it. This is the test the guardian's signature
      names.
- [ ] `make app-lint` and `make app-test` green.
- [ ] Commit: `feat: the help's logo path and the one read of it`.

### Task 3: the bytes reach the panel's facts

**Files**

- `presence/Sources/PresenceApp/HealthPoller.swift`
- `presence/Sources/PresenceApp/UpdateAction.swift`
- `presence/Sources/PresenceUI/Facts.swift`
- `presence/Sources/PresenceUI/PanelModel.swift`
- `presence/Tests/PresenceAppTests/HealthPollerTests.swift`
- `presence/Tests/PresenceAppTests/UpdateActionTests.swift`
- `presence/Tests/PresenceUITests/StateTests.swift`

**Steps**

- [ ] In `HealthPollerTests.swift`, `FakeGdoc` gains a settable logo path
      written into its help answer, and a new
      `testTheLogoIsReadOncePerHelpAnswerFromThePathItNames`: a help answer
      whose `logo_path` names a temp PNG leaves `applied(to:).logo` equal
      to that file's bytes; an answer with no `logo_path` leaves it nil; an
      answer naming a path with no file leaves it nil; an `auth status`
      answer after a help answer keeps the logo; a later help answer, asked
      with `.changed`, that names no path clears it. Red.
- [ ] `HealthReading` gains `public let logo: Data?`; `askedHelp` fills it
      with `MarkFile.read(at:)` when the help names a path, once per help
      answer and never between them: the help is asked on the poll
      schedule, and that is how often the file is read. Both `HealthReading`
      constructions in `askedAuth` carry `logo: reading.logo`.
      `applied(to:)` writes `reading.logo` into the facts' `logo` (not
      `facts.logo`: a reading that names nothing draws nothing). Green.
- [ ] In `UpdateActionTests.swift`, `testTheLogoSurvivesTheUpdateActionsRebuild`:
      `UpdateAction.applied(to:)` over facts with `logo` set returns the
      same bytes. Red, then `UpdateAction.swift:162` passes
      `logo: facts.logo`. Green.
- [ ] `Facts` gains `public let logo: Data?` with the init's default nil
      and a comment: the one picture the panel draws, from the one file
      the help names, nil where there is none. Place it after `mark`.
- [ ] In `StateTests.swift`, `testTheModelCarriesTheLogoAndOnlyFromTheFacts`:
      a model folded from facts with `logo` set carries the same bytes;
      from facts without, nil. Red, then `PanelModel` gains
      `public let logo: Data?` filled from the facts in the fold. Green.
- [ ] `make app-lint` and `make app-test` green.
- [ ] Commit: `feat: the logo's bytes reach the panel's facts`.

### Task 4: the DOCUMENTS label draws the mark

**Files**

- `presence/Sources/PresenceUI/Theme/Grid.swift`
- `presence/Sources/PresenceUI/Components/SectionView.swift`
- `presence/Sources/PresenceUI/PanelView.swift`
- `presence/Tests/PresenceUITests/ThemeTests.swift`
- `presence/Tests/PresenceUITests/RowTests.swift`
- `presence/Tests/PresenceUITests/Scenes.swift`
- `presence/Tests/PresenceUITests/Shots.swift`
- `presence/Tests/PresenceUITests/RenderTests.swift`

**Steps**

- [ ] A stand-in mark for the tests, in `Scenes.swift`: `enum SceneMark`
      with a nonisolated `static let standIn: Data`, PNG bytes of a filled
      rounded rectangle 275 by 100 px in black on transparent, drawn once
      with `NSBitmapImageRep`, so its aspect is the firm's file's 2.75 to
      1 and its ink fills its frame. Nonisolated because `Scenes.all` is,
      and the `@MainActor` helpers in `Shots.swift` cannot be called from
      it. Nothing from the firm, nothing from disk.
- [ ] In `ThemeTests.testTheGridIsTheScreens` (about line 173), the list
      gains `Grid.markHeight` at 14 as a literal, in whatever form that
      test holds the other numbers. If the test reads its numbers out of an
      html file rather than a list of literals, stop and say so. Red.
- [ ] `Grid.swift`: `public static let markHeight: CGFloat = 14` with the
      comment: the firm's mark in the DOCUMENTS label, 14 so the wordmark's
      lowercase body is the label's cap height, decided 2026-10-29,
      DECISIONS.md. Green.
- [ ] In `RowTests.swift`, four tests over `SectionLabel("Documents",
      mark:)` drawn alone at `Grid.panelWidth` in both schemes with the
      helpers in `Shots.swift`:
      `testTheDocumentsLabelDrawsTheMarkFourteenPointsTallOnTheContentEdge`
      (with the stand-in, the probe `section-label.mark` from `probeList`
      is 14 pt tall within 0.5, about 38.5 wide, its right edge at
      `Grid.panelWidth - Grid.rowInset`, 348, within 0.5, and its top at
      7 within 0.5; `inkEnds` over the whole label agrees on 348);
      `testTheMarkTakesTheLabelsOwnInk` (`pixel(_:scheme:side: 360, x:
      348 - 19.25, y: 180)`, the mark's centre in the square `pixel`
      frames, has the channels and alpha of `SectionLabel.ink(in:)` within
      0.02, in each scheme);
      `testNoMarkDrawsNothingAndMovesNothing` (without the mark,
      `probeList` has no `section-label.mark`, `inkEnds` is under 200, and
      the `section-label` probe's frame equals the one with the mark);
      `testBytesThatAreNotAPictureDrawNothing` (two hundred random bytes:
      no `section-label.mark` probe, `inkEnds` under 200). Red.
- [ ] `SectionLabel` gains `public let mark: Data?`, default nil in the
      init, and its body's `HStack` gains, after the `Spacer`, the mark
      when `NSImage(data:)` accepts the bytes: the image with
      `isTemplate = true`, as `Image(nsImage:)`, `.renderingMode(.template)`,
      `.resizable()`, `.aspectRatio(contentMode: .fit)`,
      `.frame(height: Grid.markHeight)`, under the same
      `.foregroundStyle` the label has, with `.metricProbe("section-label.mark")`
      so a test and a whole scene can find it. The 12 pt padding already
      lands its right edge on the content edge and the row centres it; add
      no offset. `.accessibilityHidden(true)`: it is not a control and
      says nothing a row does not. `NSImage(data:)` runs on each body
      evaluation; the bytes are a few kilobytes and that is accepted.
      Green.
- [ ] `PanelView.section(_:)` passes `model.logo` to `SectionView` for
      `.documents` and nil for the other two; `SectionView` takes `mark:`
      and hands it to `SectionLabel`. A fifth test in `RowTests`,
      `testOnlyTheDocumentsLabelCarriesTheMark`: `probeList` over the whole
      panel of e3, which draws NOW, DOCUMENTS and SETTINGS, holds exactly
      one `section-label.mark`, and its frame lies between the DOCUMENTS
      label's top and the first document row's top.
- [ ] `Scenes.swift`: `Scenes.machine(...)` gains `logo: Data? =
      SceneMark.standIn`, so every scene carries the stand-in, the "no
      documents" line's scenes included, because the mark goes wherever
      the label goes. One scene keeps `logo: nil` as the record of a
      machine with no picture: e9, and its caption says so in one
      sentence. In `RenderTests.swift`, the index page's lead paragraph
      (about line 764) gains one sentence: the mark in the DOCUMENTS label
      is a stand-in shape the test draws, not the firm's logo, which this
      repository does not carry. e2's caption gains one sentence naming
      the mark: 14 pt, on the right edge, the label's grey, not a press.
- [ ] `make app-lint` and `make app-test` green.
- [ ] Commit: `feat: the DOCUMENTS label draws the firm's mark`.

### Task 5: the renders, once

**Files**

- none changed; the PNGs and the index page land under
  `~/gdoc-m32f-board/shots`

**Steps**

- [ ] `PRESENCE_SHOTS="$HOME/gdoc-m32f-board/shots" make app-test` green.
- [ ] `ls ~/gdoc-m32f-board/shots` lists `e2-dark.png`, `e2-light.png`,
      `e9-dark.png` and `index.html`.
- [ ] No commit: the renders are not in git.

### Task 6: Verify acceptance criteria

**Files**

- none

**Steps**

- [ ] Every test named in the Acceptance table exists under
      `presence/Tests` or `go/cmd/gdoc` with that name.
- [ ] `git diff --name-only main...HEAD` lists only this plan,
      `go/cmd/gdoc/help.go`, `go/cmd/gdoc/helpjson_test.go`,
      `go/cmd/gdoc/doc.go`, `presence/Sources` and `presence/Tests`.
      Nothing under `go/internal/`, and not `CLAUDE.md` yet.
- [ ] `grep -rn "1J3uh6poVAneP_V_Pc5FuNI4-Gahlab0s" presence/` prints
      nothing: the app holds no copy of the Drive id.
- [ ] `grep -rln "logo_path" presence/Sources` prints `HealthFacts.swift`
      and no other file.
- [ ] `cd go && go test -race ./cmd/gdoc/ ./boundary/`, `make vet`,
      `make app-lint`, `make app-test`, `make app` green.
- [ ] No commit unless something had to change, and then it says what.

### Task 7: [Final] Update documentation

**Files**

- `CLAUDE.md`
- `docs/guide/presence.md`
- `docs/design/2026-10-17-presence-design-principles.md`
- `docs/backlog/the-panel-does-not-show-the-logo.md` (deleted)
- `docs/plans/2026-10-29-gdoc-v2-m32f-the-firms-mark-beside-documents.md`

**Steps**

- [ ] `CLAUDE.md` is at its 300-line ceiling (`TestCLAUDEmdIsUnderTheCeiling`
      fails at 301), so the invariant bullet at lines 182 to 192 is
      rewritten in the same number of lines or fewer: "reads three things"
      becomes "reads four things", the list gains "the one logo file bare
      `help --json` names, held to a PNG of at most a megabyte", and the test
      list gains `testAByteIsReadOnlyInTheFiveReaders`. Tighten the
      bullet's other words to pay for it. `cd go && go test ./boundary/
      -run 'TestCLAUDEmd|TestTheTaskMap'` prints `ok`.
- [ ] In the guide's "### Documents" section, one sentence after the
      section's first paragraph: the firm's mark sits small and grey at the
      right end of the Documents line once gdoc has built or published a
      document on this machine, and the line is plain until then. Every
      string `TestThePresencePageWalksTheThirdLooksPanel` requires stays;
      `cd go && go test ./boundary/ -run TestThePresencePageWalksTheThirdLooksPanel`
      prints `ok`.
- [ ] In the principles file, a new heading "## Amendments, 2026-10-29,
      the firm's mark beside DOCUMENTS" with two bullets: the mark's place
      and numbers, naming
      `RowTests.testTheDocumentsLabelDrawsTheMarkFourteenPointsTallOnTheContentEdge`
      and `testNoMarkDrawsNothingAndMovesNothing`, and that the 2026-10-28
      bullet "decided for the foot", two sections up past M32d's, is
      superseded in its place by Nail's word of 2026-10-29 and stays as
      the record of its day; and
      the one read, naming `MarkFileTests` and
      `SourceRulesTests.testAByteIsReadOnlyInTheFiveReaders`.
- [ ] `git rm docs/backlog/the-panel-does-not-show-the-logo.md`: the item
      is built.
- [ ] Move this plan to `docs/plans/completed/`.
- [ ] Commit: `docs: the firm's mark beside DOCUMENTS`.

## Acceptance

| Claim | Test |
|---|---|
| Bare `help --json` names the logo's cache path and touches nothing for it; the brief, a command's help and the flagless help do not carry it | `TestHelpNamesWhereTheLogoIsCached` |
| The brief help stays under its ceiling | `TestHelpBriefIsAFewHundredBytes` |
| The help never reads the token, and the pipe goldens do not change | `TestHelpNeverReadsTheToken`, `TestHelpOnAPipeIsTodaysTextByteForByte` |
| The help fact decodes with and without the path | `HealthFactsTests.testHelpNamesTheLogoPathAndItMayBeAbsent` |
| One file, one shape: a regular PNG of at most a megabyte, opened without following a link | `MarkFileTests` |
| A byte is read in five named files, in every spelling, and in no sixth | `SourceRulesTests.testAByteIsReadOnlyInTheFiveReaders` |
| The bytes come from the path the help names, once per help answer, survive an auth answer, and go when the help names none | `HealthPollerTests.testTheLogoIsReadOncePerHelpAnswerFromThePathItNames` |
| The logo survives the update action's rebuild of the facts | `UpdateActionTests.testTheLogoSurvivesTheUpdateActionsRebuild` |
| The model carries the logo only from the facts | `StateTests.testTheModelCarriesTheLogoAndOnlyFromTheFacts` |
| 14 pt tall, right edge on the content edge, top at 7 pt | `RowTests.testTheDocumentsLabelDrawsTheMarkFourteenPointsTallOnTheContentEdge` |
| The label's own ink in both appearances | `RowTests.testTheMarkTakesTheLabelsOwnInk` |
| No file: nothing drawn, nothing moves | `RowTests.testNoMarkDrawsNothingAndMovesNothing` |
| Bytes that are not a picture draw nothing | `RowTests.testBytesThatAreNotAPictureDrawNothing` |
| Only the DOCUMENTS label carries it | `RowTests.testOnlyTheDocumentsLabelCarriesTheMark` |
| The grid holds the height as a literal | `ThemeTests.testTheGridIsTheScreens` |
| CLAUDE.md says four things and stays at its ceiling | `TestCLAUDEmdIsUnderTheCeiling` |
| The guide walks the panel with the mark named | `TestThePresencePageWalksTheThirdLooksPanel` |

## Post-Completion

- The design reviewer reads `/Users/nailkhusnullin/gdoc-m32f-board/shots`:
  e2 with the stand-in mark at the right end of DOCUMENTS in both
  appearances, e9 without it, the index's sentence that the shape is a
  stand-in.
- `make release BRANCH=gdoc-v2-m32f-the-firms-mark-beside-documents` from
  the checkout on `main`, after M32d's release has merged; `GO=` with the
  next version only on Nail's word.
- Nail's hand check, in what he sees: on his own machine, which has built
  documents, the Documents line of the panel shows the Altery wordmark
  small and grey at its right end, in the dark and the light look, and a
  click on it does nothing. The firm's real wordmark, not the stand-in:
  the renders cannot show that, only his panel can.
