# gdoc v2 Milestone 32h: the mark, sharp and a press. Plan

Tasks: 6
Review: ui

2026-10-30. v2.29.0 put the firm's mark beside DOCUMENTS, and Nail looked
at it on his own screen within the hour: "logo is good in general but im
doubt about its resolution. also, when i click i expect default browser
with picture from gdrive to be opened". The lead measured the first half
(the view hands a 509 by 185 px bitmap to the compositor and lets it scale
by six and a half, which leaves the letters ragged beside anti-aliased
words) and wrote the second half up as a press. The decision is
`docs/v2/DECISIONS.md`, 2026-10-30, "The mark, sharp and a press", at line
1815, with its register row at line 120, and the spec paragraph "The firm's
mark beside DOCUMENTS" (`docs/v2/SPEC.md:218` to 231) already says both.
Both are on `main` before this run; the guardian's sentence on the entry
replaces "pending the plan" before the launch. A Go change, a Swift change
in three targets, two documents and the renders. One commit per task.
Where the code, the decision and this plan disagree, the task stops and
says so.

## Principles

Serves 3 of `PRINCIPLES.md` (not knowing never resolves toward the
destructive answer: the app opens one shape of address it was handed and
no other, and a press is drawn only where there is something to open) and
4 (the mark is drawn once at its own pixel size, so a reader sees the
wordmark and not the compositor's work on it). Serves principle 4 of
`docs/design/2026-10-17-presence-design-principles.md`, dead controls are
not shown as live: bytes with no address draw the mark and no press, and
no file draws nothing.

Strains one sentence of the 2026-10-29 entry, "It is not a press and has
no tooltip", which the 2026-10-30 entry supersedes in that one clause. It
widens two closed lists in the app by one each, and both widenings are the
decision's: the host `drive.google.com` joins the two hosts the source rule
allows (`SourceRulesTests.swift:88`), and the Opener gains a fifth verb
beside its four (`Opener.swift:81` to 125). The guardian signs that. The
app still holds no Drive id: the address is built by the binary from the id
the house file already carries (`house.yaml:1163`), handed over on the one
form of the help that carries `logo_path`, and checked by `LinkLine` before
anything opens it. No argv list, no signal, no grant and no presence field
changes, and the app reads no new byte.

## Context (from discovery)

Checked against `main` at 57db8f6. The M32i plan
(`docs/plans/2026-10-30-gdoc-v2-m32i-the-reopen-count.md`) touches
`Facts.swift`, `PanelModel.swift`, `PanelWords.swift`, `Scenes.swift`,
`RenderTests.swift`, `StateTests.swift`, `SessionTests.swift`,
`HealthPoller.swift`, `UpdateAction.swift`, `Shots.swift`, the guide and
the principles file, which this plan touches too. M32h runs first, and
M32i is cut from the `main` it merged to.

- **The help today.** `helpReport.LogoPath` (`go/cmd/gdoc/help.go:39` to
  48) is set at lines 170 to 173, when `json && len(words) == 0`, from
  `logoCached()` (lines 189 to 201: `house.Load()`, `cfg.LogoDrive()`,
  `config.AssetPath`). The pipe goldens under `go/cmd/gdoc/testdata/pipe/`
  run without `--json`, so a field on the full `--json` form leaves them
  byte for byte, and `--brief` is a second form that does not carry it.
  `TestHelpNamesWhereTheLogoIsCached` (`helpjson_test.go:224` to 275) is
  the model for a field on that one form: it holds the path as a literal
  through `houseLogoDrive` (line 228) and proves the three other forms carry
  nothing. `presenceflag_test.go:24` to 53 pins the whole help object by
  length and sha256, with `logo_path` masked to a frozen literal at line 141
  because it is a path on this machine; an address built from the house id
  is the same string on every machine, so it needs no mask and only a
  re-taken pin, under the reason the comment at lines 29 to 36 already
  lists. `publish.DocumentURL` (`go/internal/publish/publish.go:145` to 148)
  is the one URL builder shape in the tree. The help's rules are at
  `go/cmd/gdoc/doc.go:309` to 315.
- **The facts' route, as M32f built it.** `HelpFacts.logoPath`
  (`HealthFacts.swift:42` to 44) is decoded strictly at line 72; a number
  there fails the whole decode (`HealthFactsTests.swift:74` to 100).
  `HealthReading.logo` (`HealthPoller.swift:79` to 109) is filled once per
  help answer in `askedHelp` (lines 219 to 227), carried by both
  constructions in `askedAuth` (lines 235 to 253), and written into the
  facts by `applied(to:)` (lines 178 to 209) from the reading and never
  from the facts. `UpdateAction.applied(to:)` (`UpdateAction.swift:162`,
  `logo:` at 172) rebuilds the facts field by field and would drop a field
  it does not name. `Facts.logo` (`Facts.swift:235` to 238, the init at
  296, `logo:` at 302) and `PanelModel.logo` (`PanelModel.swift:343`,
  filled at 450). The three tests are
  `HealthPollerTests.testTheLogoIsReadOncePerHelpAnswerFromThePathItNames`
  (line 180; its `FakeGdoc` writes `logo_path` into the help answer at
  lines 270 and 288), `UpdateActionTests.testTheLogoSurvivesTheUpdateActionsRebuild`
  (line 222) and `StateTests.testTheModelCarriesTheLogoAndOnlyFromTheFacts`
  (line 1015).
- **The URLs the app may build.** `LinkLine` (`presence/Sources/PresenceCore/LinkLine.swift`):
  `documentPrefix` and `documentSuffix` (lines 29 to 31), `isDocumentID`
  (lines 50 to 57: at least `idFloor` 20 characters, each ASCII letter,
  digit, `_` or `-`), `documentURL(for:)` (lines 60 to 63) and
  `signInURL(_:)` (lines 89 to 97), which compares scheme, host and path
  whole and refuses a credential. The file opens nothing. Its tests are
  `PresenceCoreTests/LinkLineTests.swift` (`testTheDocsURLIsBuiltFromAValidIdOnly`
  at 17, `testOnlyGooglesSignInURLIsAccepted` at 72).
- **The one room that opens.** `Opener` (`Opener.swift:81` to 125) has
  four verbs: `openDocument` (94), `copyLink` (103), `openNote` (112) and
  `continueWithClaude` (122), over a `Desk` (lines 42 to 49) that
  `SystemDesk` (52 to 78) answers with `NSWorkspace`. The header says
  "Four verbs" at line 3 and "Nothing here decides a URL" at line 14; the
  `Desk` doc still says "three verbs" at line 37, which is stale and is
  corrected in passing. `OpenerTests.swift` holds the literals at lines 16
  to 18, `testNoTitleEverReachesAURLOrThePasteboard` at 41 and the recording
  `NoteDesk` at 208 to 240. `SourceRulesTests.testTheSystemIsAskedOnlyThroughTheOpenerAndTheRelauncher`
  (lines 245 to 270) holds the pair, and
  `testTheOnlySchemesAndHostsAreGooglesAndTheVaults` (lines 367 to 385) is
  exact equality over `Sources/`: every host under `google.com` a source
  file names (the scan at lines 530 to 543 finds any spelling, a comment
  included) must be in `hosts` (line 88), and every listed host must be
  named somewhere, so the list and `LinkLine` move together.
- **What a press is.** `ActionKind` (`PanelModel.swift:29` to 61) is the
  closed list of twenty kinds; `Session.perform(_:_:)`
  (`SessionActions.swift:27` to 78) is the one exhaustive switch over it,
  and its header at line 3 says "twenty-one", which is stale by one and is
  corrected in passing. `PanelHeader.tooltipAlign(of:)`
  (`PanelHeader.swift:143`) compares one case and does not switch.
  `PanelActions` (`PanelModel.swift:114` to 120) is one closure, and a test
  records through it (`CardTests.swift:419`). `DocumentPress`
  (`DocumentPress.swift:25` to 32) is the shape of a press that belongs to
  no icon: a static function that hands back a closure or nil, read in a
  test by calling it. `SessionTests.testARowPressOpensTheDocumentAndLeavesTheStrip`
  (line 469) is the shape of a session test over `perform` with a
  `FakeDesk`.
- **How a control looks.** `ActionButton` (`ActionButton.swift:127` to
  159): a plain `Button`, `palette.hover.color` filled behind it on
  `Grid.controlRadius` while `hovering && pressable` (lines 138 to 145),
  `.onHover` at 150, `.tooltip(_:align:forced:)` at 154 and
  `.accessibilityLabel` at 158. The tooltip modifier is
  `TooltipLayer.swift:208` to 212; the plate it draws is probed as
  `"tooltip"` at line 202 and appears only under a `.tooltipLayer()`, which
  the controls sheet adds per piece (`ControlTests.swift:425` to 445): each
  piece sits in its own frame of `Self.tooltipBand` height (line 461),
  aligned `.topTrailing`, because a layer with no room below flips the
  bubble over the thing it explains. The delay is `Timing.tooltipDelay`,
  0.5 (`Timing.swift:12`). Principle 8 of
  the principles file (lines 38 to 42): hover goes darker, never lighter.
  Principle 3: tooltips after half a second. No press in the panel changes
  the pointer today, and the decision keeps the arrow on the mark too: no
  `NSCursor` anywhere in this milestone.
- **Where the mark is drawn.** `SectionLabel` (`SectionView.swift:27` to
  104): `picture` (lines 46 to 52) builds `NSImage(data:)` on every body
  evaluation and sets `isTemplate`; the body (lines 87 to 97) draws it
  `.renderingMode(.template).resizable().aspectRatio(contentMode: .fit)
  .frame(height: Grid.markHeight)` in the label's ink, `.accessibilityHidden(true)`,
  probed as `"section-label.mark"`, no gesture, no tooltip. `SectionView`
  (107 to 144) hands `mark:` through; `PanelView.section(_:)` and
  `mark(of:)` (`PanelView.swift:105` to 117) give it to DOCUMENTS alone.
  `Grid.markHeight` is 14 (`Grid.swift:49`, pinned at `ThemeTests.swift:187`);
  `Grid.unit` 8 (line 19), `Grid.controlRadius` 6 (54), `Grid.rowInset` 12
  (63), `Grid.sectionLabelHeight` 28 (27). `Palette.hover` is
  `Palette.swift:87`. The cached file is 509 by 185 px; at 14 pt on a 2x
  screen that is 77 by 28 px, and the lead's count on Nail's crop is 93
  half-grey edge pixels against about 300 for an area-averaged downsample
  of the same file to the same size. The plan reviewer measured the route
  on this 2x Mac (its probes are under the session's scratchpad, `scale/`):
  `NSImage(cgImage:size:)` makes a snapshot rep whose pixel count follows
  the main screen's scale and not the `CGImage`, so a 116 by 42 bitmap
  wrapped at 38.67 by 14 pt reports 77 by 28, and `Image(nsImage:)` under
  a renderer at another scale resamples again. `Image(decorative:scale:)`
  over the `CGImage` itself draws 1:1 at 1x, 2x and 3x. A 509 by 185
  picture of one-pixel stripes shrunk to 77 by 28 through a `CGContext`
  at `.high` is 2,156 of 2,156 pixels a flat mid grey; at `.none` and
  `.low` it aliases. That stripe picture is the decision's own test of
  the averaging.
- **The source rule on reading.** `SourceRulesTests.testAByteIsReadOnlyInTheFiveReaders`
  (lines 457 to 495) refuses thirteen spellings outside five files, and the
  image loaders among them are `NSImage(contentsOf`, `NSImage(contentsOfFile`,
  `NSImage(byReferencing`, `CGImageSourceCreateWithURL` and `AsyncImage`.
  Making a bitmap out of bytes already in hand is not a read: a `CGContext`
  over the `CGImage` that `NSImage(data:)` already holds names none of the
  thirteen, and Task 4 uses nothing else. `CGImageSourceCreateWithData`
  would also pass the scan, and is not needed.
- **The tests that measure.** `Shots.swift`: `png` (line 17) renders
  through `ImageRenderer` at `scale = 2`, as do `probeList` (131),
  `pixel` (164) and `inkEnds` (187). The renderer already averages when it
  draws the old resizable path, so no render test goes red on the
  raggedness Nail saw: the sharpness on a screen is his hand check alone,
  and the tests hold the bitmap's pixel size, its averaging and that the
  label draws that bitmap and nothing else. The mark's tests are
  `RowTests.swift:495` to 610 (the content edge is `contentEdge` at 487,
  348 pt) and they must keep passing as they are. The row sheet
  (`RowTests.testTheRowSheetRenders`, lines 770 to 838) draws DOCUMENTS
  twice at lines 826 and 828, words alone and with `SceneMark.standIn`.
  `RenderTests.testEverySheetWordIsAWordAPanelDraws` (line 688) holds every
  string a sheet draws to `panelVocabulary()` (`Shots.swift:487` to 521),
  which is built from the scenes, so a tooltip shown on a sheet must be a
  word a scene's model carries. `SheetPieces.words()` (`Shots.swift:441`
  to 478) is the sheet's side of that pair.
- **The scenes.** `Scenes.machine(...)` (`Scenes.swift:259` to 284) takes
  `logo: Data? = SceneMark.standIn`; e5 passes `logo: nil` (line 483) and
  its caption says why (lines 469 to 481); e2's caption ends "and nothing
  presses it" (line 448), which stops being true. `SceneMark` (lines 91 to
  98) holds the stand-in and `notAPicture`. The index page's paragraph on
  the stand-in is `RenderTests.swift:770` to 775. The binary names
  `logo_url` whenever the house id exists, file or not, so a first run has
  the address and no bytes: e5 keeps the address and is the true "address,
  no bytes, no press" scene. The repository carries no logo and no Drive id
  of the firm's, by the decision of 2026-10-16, and M32f's Task 6 grep
  holds the app to that; the renders and the fact tests need a page
  address, so they use a stand-in id of the right shape, never the real
  one or a fragment of it.
- **The documents.** The guide's "### Documents" (`docs/guide/presence.md:160`)
  says at lines 171 to 174 that the mark sits small and grey and the line
  is plain until the file exists; `TestThePresencePageWalksTheThirdLooksPanel`
  (`go/boundary/guide_test.go:201` to 242) holds that page to a list of
  strings. The principles file's last section is "## Amendments,
  2026-10-29, the firm's mark beside DOCUMENTS" at line 771, whose second
  bullet says "The app holds no Drive id and asks no host", which stays
  true. `CLAUDE.md` is at 300 lines (`TestCLAUDEmdIsUnderTheCeiling`), and
  its menu bar bullet (lines 182 to 192) names what the app runs, signals
  and reads, not what it opens, so it does not change.

### What the lead changes on `main` before the run

Done and committed before the launch: the decision entry, its register row,
the 2026-10-29 row's superseded clause, and the spec paragraph. The
guardian's sentence replaces "Trust guardian, 2026-10-30: pending the plan"
at `DECISIONS.md:1862` once it has read this plan. Nothing else.

### Out of scope

- A copy of the Drive id in the app, in any file under `presence/`.
- A tooltip, a line or a greyed shape where there is no mark.
- Any colour on the mark, pressed, hovered or not.
- The app reading Drive, or the help fetching, stat-ing or reading the
  logo: the help names an address, and opening it is the system's act.
- A pointer change on any press, and a sixth Opener verb.
- `--brief`, a command's help or the flagless help carrying the address.
- A third place that opens the picture.

## Development Approach

- TDD: the failing test first, in the file the Context names, then the
  smallest change that passes it.
- One commit per task, message in the house form, no attribution lines.
- Nothing is loosened: the help object's pin is re-taken under its own
  stated reason and not dropped; the schemes list stays at three; the
  readers list stays at five; the palette literals and the grid list stay;
  the mark's five existing tests pass unchanged.
- Go tests run by package, never `make test`: the session's environment
  carries the live variables, and `make test` would write to Drive.
- Plain English in every comment and caption, no em dashes.
- The source scans read comments and `Tests/` as well as code, so a
  comment in a new file never spells whole-word `Process` or
  `NSWorkspace`, any host under Google's domain, or a spelling from the
  readers' list; it says "the system", "the page's host" or "a read"
  instead.

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
PRESENCE_SHOTS="$HOME/gdoc-m32h-board/shots" make app-test
```

## Progress Tracking

Tick each box as it is done. A task ends with its commit.

## Tasks

### Task 1: the help names the picture's page

**Files**

- `go/cmd/gdoc/help.go`
- `go/cmd/gdoc/helpjson_test.go`
- `go/cmd/gdoc/presenceflag_test.go`
- `go/cmd/gdoc/doc.go`

**Steps**

- [x] In `helpjson_test.go`, a new `TestHelpNamesTheLogosPageInDrive`
      beside `TestHelpNamesWhereTheLogoIsCached`: `t.Setenv("GDOC_CONFIG_DIR",
      t.TempDir())`, run `help --json`, read `data.logo_url` and require
      it equal to the literal
      `"https://drive.google.com/file/d/1J3uh6poVAneP_V_Pc5FuNI4-Gahlab0s/view"`,
      written out in full and not built from `houseLogoDrive`, so a run
      that names another page fails here. Require that nothing exists under
      the config dir after the run, as the sibling test does. Then run
      `help --brief --json`, `help publish --json` and bare `help` on a pipe
      and require `logo_url` absent from each, with the same three
      subtests. Red.
- [x] In `help.go`, `helpReport` gains `LogoURL string \`json:"logo_url,omitempty"\``
      after `LogoPath`, with a comment in the file's voice: the picture's
      own page in Google Drive, named so the menu bar app can open it on a
      press without holding a Drive id; on the same one form as `logo_path`
      and on no other; built from the id and nothing fetched; name the
      test. A helper `logoPage()` beside `logoCached()` returns
      `"https://drive.google.com/file/d/" + id + "/view"` from
      `house.Load()` and `cfg.LogoDrive()`, and the empty string on any
      error or an empty id, for the reason `logoCached` states. Lines 171
      to 173 set `report.LogoURL = logoPage()` under the same `if` and
      nowhere else. Green.
- [x] `presenceflag_test.go`: the help object with the flag off is one
      field longer, so `helpObjectBytes` and `helpObjectSHA256` are
      re-taken from the object the test prints, and the comment's list at
      lines 29 to 36 gains "and re-taken in M32h for logo_url". No mask:
      the address is the same string on every machine, and the path mask
      at line 141 stays as it is. Every other byte stays pinned.
- [x] `TestHelpNamesWhereTheLogoIsCached`, `TestHelpBriefIsAFewHundredBytes`,
      `TestHelpNeverReadsTheToken` and `TestHelpOnAPipeIsTodaysTextByteForByte`
      green and untouched; no file under `testdata/pipe/` changes.
- [x] `go/cmd/gdoc/doc.go`, in the paragraph at lines 309 to 315: one
      sentence that the same form names `logo_url`, the picture's page in
      Drive built from the house id, so the app opens it without holding
      the id, naming `TestHelpNamesTheLogosPageInDrive`.
- [x] `cd go && go test -race ./cmd/gdoc/ ./boundary/` and `make vet`
      green. The boundary package is run because a new host literal under
      `go/` is the kind of thing it reads; if a test there names it, stop
      and say which.
- [x] Commit: `feat: the help names the logo's page in Drive`.

### Task 2: the address reaches the panel's facts

**Files**

- `presence/Sources/PresenceCore/HealthFacts.swift`
- `presence/Sources/PresenceCore/LinkLine.swift`
- `presence/Sources/PresenceApp/HealthPoller.swift`
- `presence/Sources/PresenceApp/UpdateAction.swift`
- `presence/Sources/PresenceUI/Facts.swift`
- `presence/Sources/PresenceUI/PanelModel.swift`
- `presence/Tests/PresenceCoreTests/HealthFactsTests.swift`
- `presence/Tests/PresenceCoreTests/LinkLineTests.swift`
- `presence/Tests/PresenceCoreTests/SourceRulesTests.swift`
- `presence/Tests/PresenceAppTests/HealthPollerTests.swift`
- `presence/Tests/PresenceAppTests/UpdateActionTests.swift`
- `presence/Tests/PresenceUITests/StateTests.swift`

**Steps**

The order is the one that compiles at every step: the core first, then
the facts' field with its nil default, then the three rooms that fill it.
The `UpdateAction` test is a real red only once `Facts` has the field.

- [x] In `LinkLineTests.swift`, `testTheMarksPageIsOneShapeAndNothingElse`:
      `LinkLine.markPageURL(_:)` over the literal
      `https://drive.google.com/file/d/<a 28-character stand-in id>/view`
      returns that string; nil for nil; nil for a Docs document URL; nil
      with a query, a fragment, a trailing slash or a credential before the
      host; nil for an uppercase host or an `http` scheme; nil for an id
      of 19 characters or one carrying a dot; and the string trimmed of
      surrounding whitespace comes back without it. Red.
- [x] In `SourceRulesTests.swift:88`, `hosts` gains `"drive.google.com"`
      with one line on why: the picture's page, opened on a press, the
      address built by the binary and checked here. The test's two
      messages at lines 383 and 384 say "three", and the stale words move
      with them: the file comment at lines 17 to 18 ("two schemes and two
      hosts"), the doc at line 87 ("both Google's") and the test's doc at
      line 363 ("Two schemes and two hosts") each say three hosts. Red,
      because no source names the host yet.
- [x] `LinkLine.swift`: `markPagePrefix = "https://drive.google.com/file/d/"`,
      `markPageSuffix = "/view"`, and `public static func markPageURL(_
      text: String?) -> String?`: trim, require the prefix and the suffix
      whole and case-sensitive, require what lies between to pass
      `isDocumentID`, and return `markPagePrefix + id + markPageSuffix`
      built again rather than the input, so nothing rides between the parts.
      The header's first line, "the one it may accept", becomes the two it
      may accept, and the paragraph under it gains one sentence: a third
      host, the Drive page of the firm's mark, from the binary's own
      `logo_url`. Green, both files.
- [x] In `HealthFactsTests.swift`, `testHelpNamesTheLogosPageAndItMayBeAbsent`,
      mirroring lines 74 to 100 with the stand-in id and never a fragment
      of the real one: `logo_url` decodes to `HelpFacts.logoURL`, an
      absent field is nil, and a numeric field fails the whole decode.
      Red. `HelpFacts` gains `public let logoURL: String?` after `logoPath`,
      read at line 72 the same way, with its one-line comment: the raw
      string as the help printed it; `LinkLine` judges it, not this file.
      Green.
- [x] `Facts` gains `public let logoURL: String?` after `logo` with the
      init's default nil and a comment: the page the mark opens, already
      held to one shape by the poller, nil where the help named none.
      Everything still compiles, because every caller takes the default.
- [x] In `HealthPollerTests.swift`, `FakeGdoc` gains `answersLogoPage(_:)`
      written into the help answer beside `logo_path` (lines 270 and 288),
      and a new `testTheLogosPageIsTakenOnlyInItsOneShape`: a help answer
      naming the stand-in page leaves `applied(to:).logoURL` equal to it; an
      answer naming a Docs URL there leaves it nil; an answer with no
      `logo_url` leaves it nil; an `auth status` after a help keeps it; a
      later help answer asked with `.changed` that names none clears it.
      Red.
- [x] `HealthReading` gains `public let logoURL: String?` after `logo`,
      default nil; `askedHelp` fills it with
      `help.logoURL.flatMap(LinkLine.markPageURL)`; both constructions in
      `askedAuth` carry `logoURL: reading.logoURL`; `applied(to:)` writes
      `logoURL: reading.logoURL`, the reading's own and never the facts'.
      Green.
- [x] In `UpdateActionTests.swift`, `testTheLogosPageSurvivesTheUpdateActionsRebuild`,
      mirroring line 222: facts with `logoURL` set come back with it. Red,
      because `UpdateAction.applied(to:)` rebuilds the facts without it;
      then `UpdateAction.swift:172` gains `logoURL: facts.logoURL`. Green.
- [x] In `StateTests.swift`, `testTheModelCarriesTheLogosPageAndOnlyFromTheFacts`,
      mirroring line 1015. Red, then `PanelModel` gains `public let
      logoURL: String?` filled from the facts at line 450. Green.
- [x] `make app-lint` and `make app-test` green.
- [x] Commit: `feat: the logo's page reaches the panel's facts`.

### Task 3: the Opener's fifth verb and the kind that reaches it

**Files**

- `presence/Sources/PresenceApp/Opener.swift`
- `presence/Sources/PresenceApp/SessionActions.swift`
- `presence/Sources/PresenceUI/PanelModel.swift`
- `presence/Tests/PresenceAppTests/OpenerTests.swift`
- `presence/Tests/PresenceAppTests/SessionTests.swift`

**Steps**

- [ ] In `OpenerTests.swift`, a literal `page` beside `id` and `link`
      (lines 16 to 18), built on a 28-character stand-in id, and a new
      `testOpenMarkOpensTheDrivePageAndNoOtherShape`: `opener.openMark(page)`
      is true and `desk.opened` holds exactly that URL; `openMark(nil)`,
      `openMark(Self.link)` and `openMark(page + "?usp=sharing")` are each
      false and open nothing more; nothing is copied or activated. Add
      `opener.openMark(page)` to the list in
      `testNoTitleEverReachesAURLOrThePasteboard` (lines 41 to 59). Red.
- [ ] `Opener.swift`: `@discardableResult public func openMark(_ page:
      String?) -> Bool`, after `continueWithClaude`: `LinkLine.markPageURL`
      on the string, `URL(string:)` on what comes back, `desk.open`, false
      otherwise. The header's "Four verbs" at line 3 becomes five with the
      mark's page named; line 10, "The folder panel was the fifth verb
      until the third look", says it was the fifth then and the mark's page
      is the fifth now; the `Desk` doc's "three verbs" at line 37 is
      corrected to the three things a desk does; the struct's doc at line
      80, "The four things a row's icons do", becomes the five things the
      panel opens or copies, one of them from no row. In
      `SourceRulesTests.swift:232`, "the four verbs" becomes five. Green.
- [ ] `PanelModel.swift`: `ActionKind` gains `case openMark` after
      `hideDoc`, with a doc comment: the press on the firm's mark, whose
      target is the page's address the facts carry and never an id.
      `SessionActions.perform` fails to compile until the next step, which
      is the exhaustive switch doing its job.
- [ ] In `SessionTests.swift`, `testTheMarkPressOpensItsPageAndMovesNothingElse`,
      in the shape of line 469: `session.perform(.openMark, page)` opens
      exactly that URL through the `FakeDesk`; `ui.stripSeen` and
      `ui.cardHintSeen` stay false and nothing is written to the store,
      because a click on the mark is a click on neither a row nor one of a
      row's icons. Red.
- [ ] `SessionActions.swift`: `case .openMark: opener.openMark(target)` and
      nothing else; the header's count at line 3, which says twenty-one
      over twenty cases today, becomes twenty-one for the twenty-one there
      now are, with "the press on the firm's mark" in its list. Green.
- [ ] `make app-lint` and `make app-test` green, the source rules included:
      `testTheSystemIsAskedOnlyThroughTheOpenerAndTheRelauncher` still names
      two files.
- [ ] Commit: `feat: the Opener's fifth verb opens the mark's page`.

### Task 4: the mark drawn sharp

**Files**

- `presence/Sources/PresenceUI/Components/MarkPicture.swift` (new)
- `presence/Sources/PresenceUI/Components/SectionView.swift`
- `presence/Tests/PresenceUITests/RowTests.swift`

**Steps**

What the picture is: a `CGImage`, never an `NSImage`. The reviewer
measured that `NSImage(cgImage:size:)` reports the main screen's pixel
count and not the bitmap's, and that `Image(nsImage:)` resamples again
under a renderer at another scale, so the one route that draws 1:1 at 1x,
2x and 3x is `Image(decorative:scale:)` over the bitmap itself. The tests
assert pixels on the bitmap, not points on a wrapper.

- [ ] In `RowTests.swift`, three tests over `MarkPicture`.
      `testTheMarkIsABitmapAtItsOwnPixelSize`:
      `MarkPicture.make(SceneMark.standIn, height: Grid.markHeight, scale:
      2)` is a `CGImage` 28 px high and 77 px wide (275 by 100 at 28 high
      is 77); at scale 3 it is 42 high and 116 wide, and the points it
      will draw at are the pixels over the scale, 14 high and 116 over 3
      wide, so the test asserts the height and the pixel width and never
      38.5; `SceneMark.notAPicture` gives nil; `SceneMark.standIn` at
      height 0 gives nil. `testTheMarkIsMadeOncePerBytesHeightAndScale`:
      two calls with the same bytes, height and scale return the identical
      object (`===`); a call at scale 3 returns another; a call at another
      height returns another. `testTheBitmapAveragesWhatItShrinks`, which
      is the decision's own test: a 509 by 185 picture of one-pixel black
      and white vertical stripes, drawn in the test as `SceneMark.standIn`
      is drawn, shrunk through `MarkPicture.make` to 28 high at scale 2 is
      77 by 28, and every one of its 2,156 pixels is a mid grey, each
      channel between 0.35 and 0.65; the reviewer measured 2,156 of 2,156
      with `.high` and aliasing with `.none` and `.low`, so a worker who
      reaches for a cheaper interpolation fails here. Red.
- [ ] `MarkPicture.swift`, new in `PresenceUI`: a file comment in the house
      voice (why the compositor's scaling left ragged edges, measured
      2026-10-30 in DECISIONS.md; what is made instead; why once; why a
      `CGImage` and not a wrapper, with the reviewer's measurement in one
      sentence), a `@MainActor public enum MarkPicture` with `public static
      func make(_ data: Data, height: CGFloat, scale: CGFloat) -> CGImage?`.
      It takes the `CGImage` out of `NSImage(data:)` with
      `cgImage(forProposedRect:context:hints:)`, refuses a zero or negative
      height, scale or source size, computes the pixel box as `height *
      scale` rounded, by the source's own aspect for the width, draws the
      source into a `CGContext` of exactly that box with
      `interpolationQuality = .high` and no other transform, and hands the
      context's image back. A memo keyed on the bytes, the height and the
      scale answers a repeat call with the same object, so a body
      evaluation costs a comparison and not a draw; the key carries the
      height because a call at another height after the label's would
      otherwise get the label's bitmap. No spelling from the readers' list
      (`SourceRulesTests.swift:459` to 474) appears in the file, and
      `testAByteIsReadOnlyInTheFiveReaders` still names five. Green.
- [ ] In `RowTests.swift`, `testTheLabelDrawsTheBitmapItWasMade`:
      `SectionLabel.picture(mark: SceneMark.standIn, scale: 2)` is `===`
      `MarkPicture.make(SceneMark.standIn, height: Grid.markHeight, scale:
      2)`, and nil for `SceneMark.notAPicture` and for nil bytes. This is
      the one red the wiring has: the renderer already averages the old
      resizable path, so no render test shows the raggedness Nail saw, and
      the sharpness on a screen is his hand check alone. Red.
- [ ] `SectionView.swift`: `SectionLabel` gains `public static func
      picture(mark: Data?, scale: CGFloat) -> CGImage?`, which is
      `MarkPicture.make` at `Grid.markHeight` or nil, reads
      `@Environment(\.displayScale)`, and its body draws
      `Image(decorative: picture, scale: displayScale)
      .renderingMode(.template)` with `.frame(height: Grid.markHeight)`,
      the ink, the opacity, the hidden accessibility and the probe as
      before, and no `.resizable()` and no `.aspectRatio`. The `picture`
      property at lines 46 to 52 and its `isTemplate` go; the comment at
      lines 40 to 45 says what is now true: the bitmap is made at the
      drawn size and the view scales nothing. Green.
- [ ] The five mark tests at `RowTests.swift:495` to 610 pass as they are:
      14 pt tall within 0.5, 38.5 wide, right edge at 348, top at 7, the
      label's ink in both appearances, no mark with no bytes, no mark for
      bytes that are no picture, one mark on e3. If the probe's frame
      moves because the renderer hands the view another `displayScale`
      than its own `scale`, stop and say so rather than fixing it with a
      constant.
- [ ] `make app-lint` and `make app-test` green.
- [ ] Commit: `feat: the mark is drawn from a bitmap at its own pixel size`.

### Task 5: the press, its words, the scenes and the renders

**Files**

- `presence/Sources/PresenceUI/PanelWords.swift`
- `presence/Sources/PresenceUI/MarkPress.swift` (new)
- `presence/Sources/PresenceUI/Components/SectionView.swift`
- `presence/Sources/PresenceUI/PanelView.swift`
- `presence/Tests/PresenceUITests/RowTests.swift`
- `presence/Tests/PresenceUITests/Scenes.swift`
- `presence/Tests/PresenceUITests/Shots.swift`
- `presence/Tests/PresenceUITests/RenderTests.swift`

**Steps**

- [ ] In `RowTests.swift`, three tests. `testTheMarkWithAPageIsAPressThatOpensIt`:
      through a recording `PanelActions` as `CardTests.swift:419` does,
      `MarkPress.open(model, actions:)` over a model whose `logo` and
      `logoURL` are both set returns a closure, and calling it records one
      press of kind `.openMark` with the page as its target.
      `testTheMarkWithoutAPageOrWithoutBytesIsNoPress`: nil with bytes and
      no address, nil with an address and no bytes, nil with neither.
      `testTheMarksWordsAreTheDrivePageAndOnlyWhereItPresses`: a
      `SectionLabel("Documents", mark: SceneMark.standIn, press: {},
      forcedTooltip: true)` under `.tooltipLayer()`, as
      `ControlTests.swift:436` to 445 draws a forced tooltip, reports a
      `"tooltip"` probe in `probeList`, and `Tips.openMark` is the literal
      "Open in Google Drive"; the same label with no press and
      `forcedTooltip: true` reports no `"tooltip"` probe; the
      `section-label.mark` frame is the same in both, 38.5 by 14 with its
      right edge at 348, so the press moved nothing. Red.
- [ ] `PanelWords.swift`: `Tips.openMark = "Open in Google Drive"` after
      `openDoc` (line 21), in the voice of the line above it: verb and
      object, sentence case, no full stop, with a one-line comment naming
      the decision.
- [ ] `MarkPress.swift`, new in `PresenceUI`, in the shape of
      `DocumentPress.swift`: `public enum MarkPress` with `public static
      func open(_ model: PanelModel, actions: PanelActions) -> (() ->
      Void)?`, which is nil unless the model carries both the bytes and the
      page, and otherwise `{ actions.perform(.openMark, page) }`. The file
      comment says why both are needed: no file, no mark and no press;
      bytes and no address, the mark and no press, which is principle 4.
- [ ] `SectionLabel` gains `public let press: (() -> Void)?` and `public
      let forcedTooltip: Bool`, both defaulted in the init, and `SectionView`
      hands them through. With a press the picture is the label of a plain
      `Button` that calls it; behind the picture a
      `RoundedRectangle(cornerRadius: Grid.controlRadius)` filled with
      `palette.hover.color` while hovering, reaching `Grid.unit / 2` beyond
      the picture on each side through `.padding` and the matching negative
      padding, so the layout frame and the probe do not move; the pointer
      stays the arrow, no `NSCursor` call;
      `.tooltip(Tips.openMark, align: .trailing, forced: forcedTooltip)`;
      `.accessibilityLabel(Tips.openMark)` in place of the hidden
      accessibility. Without a press the picture is drawn exactly as Task 4
      left it: no button, no plate, no hover, no tooltip, hidden from
      accessibility. The file's header gains one paragraph on the press
      and the decision. Green.
- [ ] `PanelView.swift`: `section(_:)` passes `press: markPress(of:)`,
      where `markPress(of:)` is `MarkPress.open(model, actions: actions)`
      for `.documents` and nil for the other two, beside `mark(of:)` at
      lines 115 to 118. `forcedTooltip` is not wired through the panel: no
      scene forces the mark's words, and the parameter is the sheet's
      alone.
- [ ] `Scenes.swift`: `SceneMark` gains `static let page`, the literal
      `https://drive.google.com/file/d/` plus a 28-character stand-in id of
      the Drive alphabet plus `/view`, with a comment that the real id never
      enters the repository; `Scenes.machine(...)` gains `logoURL: String? =
      SceneMark.page` passed into `Facts`; e5 keeps that default beside
      its `logo: nil` at line 483, because the binary names the address
      whenever the house id exists, file or not, which makes e5 the scene
      with an address, no bytes and no press. e2's caption at line 448 ends
      instead: "in the label's own grey, and a click on it opens the
      picture's own page in Google Drive." e5's caption already says there
      is no picture, and one clause is added: the address is there and the
      mark is not, so there is nothing to press. In `Shots.swift`,
      `panelVocabulary()` inserts `Tips.openMark` for any scene whose model
      carries both `logo` and `logoURL`, and `SheetPieces.words()` adds
      `Tips.openMark`, so `testEverySheetWordIsAWordAPanelDraws` holds the
      sheet to the scenes as before.
- [ ] `RowTests.testTheRowSheetRenders`: the DOCUMENTS label with the mark
      at line 828 gains `press: {}` and `forcedTooltip: true`, and sits in
      the frame pattern the controls sheet uses for a forced tooltip
      (`ControlTests.swift:425` to 445): its own `.frame(width:
      Grid.panelWidth, height: ControlTests.tooltipBand, alignment:
      .topTrailing)` under its own `.tooltipLayer()`, so the bubble has
      room below and does not flip over the plain DOCUMENTS label above
      it; the comment at lines 819 to 822 says so. If `tooltipBand` is
      private to `ControlTests`, move it to `Shots.swift` as a shared
      value rather than copying the number. `Scenes.sheets`' "rows" sentence
      (`Scenes.swift:215` to 223) ends with the mark's tooltip held open.
      In `RenderTests.swift`, the index paragraph at lines 770 to 775 gains
      one sentence: a click on the mark opens the picture's page in Drive,
      and the page address in the renders is a stand-in as the shape is.
- [ ] `PRESENCE_SHOTS="$HOME/gdoc-m32h-board/shots" make app-test` green.
      `ls ~/gdoc-m32h-board/shots` lists `e2-dark.png`, `e2-light.png`,
      `e5-dark.png`, `rows-dark.png`, `rows-light.png` and `index.html`;
      e2 draws the stand-in mark at the right end of DOCUMENTS as before,
      e5 draws none, and the rows sheet shows "Open in Google Drive" under
      the second DOCUMENTS label in both appearances. The renders are not
      in git.
- [ ] `make app-lint` and `make app-test` green.
- [ ] Commit: `feat: the mark is a press that opens its page in Drive`.

### Task 6: [Final] Verify acceptance and update documentation

**Files**

- `docs/design/2026-10-17-presence-design-principles.md`
- `docs/guide/presence.md`
- `go/boundary/guide_test.go`
- `docs/plans/2026-10-30-gdoc-v2-m32h-the-mark-sharp-and-a-press.md`

**Steps**

- [ ] Every test named in the Acceptance table exists under
      `presence/Tests` or `go/cmd/gdoc` with that name.
- [ ] `git diff --name-only main...HEAD` lists only this plan (at its
      path under `docs/plans/completed/` once the last step has moved it),
      `go/cmd/gdoc/help.go`, `go/cmd/gdoc/helpjson_test.go`,
      `go/cmd/gdoc/presenceflag_test.go`, `go/cmd/gdoc/doc.go`,
      `go/boundary/guide_test.go`, `docs/guide/presence.md`,
      `docs/design/2026-10-17-presence-design-principles.md`,
      `presence/Sources` and `presence/Tests`. Nothing under `go/internal/`
      and not `CLAUDE.md`.
- [ ] `grep -rn "1J3uh6poVAneP_V_Pc5FuNI4-Gahlab0s" presence/` prints
      nothing: the app holds no copy of the Drive id, in a source, a test
      or a scene.
- [ ] `grep -rln "logo_url" presence/Sources` prints `HealthFacts.swift`
      and no other file; `grep -rln "drive.google.com" presence/Sources`
      prints `LinkLine.swift` and no other file.
- [ ] In `guide_test.go`, the list at lines 202 to 238 gains
      `"Open in Google Drive"` with a comment line. Red. Then in the
      guide's "### Documents", the paragraph at lines 171 to 174 gains two
      sentences: a click on the mark opens the picture's own page in Google
      Drive in your browser, and rest the pointer on it for half a second
      and it says "Open in Google Drive". `cd go && go test ./boundary/
      -run TestThePresencePageWalksTheThirdLooksPanel` prints `ok`.
- [ ] In the principles file, after the 2026-10-29 section at line 771, a
      new "## Amendments, 2026-10-30, the mark sharp and a press" with two
      bullets. The first: the mark is drawn from a bitmap made once at its
      drawn pixel size, so its edges are anti-aliased like the words
      beside it, and the view scales nothing, naming
      `RowTests.testTheMarkIsABitmapAtItsOwnPixelSize` and
      `RowTests.testTheBitmapAveragesWhatItShrinks`, and saying in one
      clause that the sharpness on a screen is Nail's own check, because
      the renderer averages already. The second: the
      mark is a press that opens the picture's page in Drive, with the
      tooltip after half a second, the hover plate, and no
      press where there is no file or no address; the sentence "The app
      holds no Drive id and asks no host" in the section above stays true,
      because the page's address comes from the binary on the same help
      answer as the path and the app opens only that one shape; naming
      `RowTests.testTheMarkWithAPageIsAPressThatOpensIt`,
      `OpenerTests.testOpenMarkOpensTheDrivePageAndNoOtherShape` and
      `SourceRulesTests.testTheOnlySchemesAndHostsAreGooglesAndTheVaults`.
      The 2026-10-29 bullet's "not a press" clause is superseded in its
      place and stays as the record of its day.
- [ ] `CLAUDE.md` does not change: its menu bar bullet names what the app
      runs, signals and reads, and none of those moved. `cd go && go test
      ./boundary/ -run 'TestCLAUDEmd|TestTheTaskMap'` prints `ok`.
- [ ] `cd go && go test -race ./cmd/gdoc/ ./boundary/`, `make vet`,
      `make app-lint`, `make app-test`, `make app` green.
- [ ] Move this plan to `docs/plans/completed/`.
- [ ] Commit: `docs: the mark, sharp and a press`.

## Acceptance

| Claim | Test |
|---|---|
| Bare `help --json` names the picture's page from the house id, and the brief, a command's help and the flagless help do not | `TestHelpNamesTheLogosPageInDrive` |
| The path is still named and nothing is touched | `TestHelpNamesWhereTheLogoIsCached` |
| The brief stays under its ceiling, the help never reads the token, the pipe goldens do not change, the help object is pinned | `TestHelpBriefIsAFewHundredBytes`, `TestHelpNeverReadsTheToken`, `TestHelpOnAPipeIsTodaysTextByteForByte`, `TestAnExperimentalFlagIsInTheTableAndInNoScreen` (subtest "the help object with the flag off") |
| One shape of address, built again from its parts, and nothing else | `LinkLineTests.testTheMarksPageIsOneShapeAndNothingElse` |
| Three hosts in the sources, each named, and no fourth | `SourceRulesTests.testTheOnlySchemesAndHostsAreGooglesAndTheVaults` |
| The help fact decodes with and without the address | `HealthFactsTests.testHelpNamesTheLogosPageAndItMayBeAbsent` |
| The address is taken only in its one shape, once per help answer, survives an auth answer, and goes when the help names none | `HealthPollerTests.testTheLogosPageIsTakenOnlyInItsOneShape` |
| The address survives the update action's rebuild | `UpdateActionTests.testTheLogosPageSurvivesTheUpdateActionsRebuild` |
| The model carries the address only from the facts | `StateTests.testTheModelCarriesTheLogosPageAndOnlyFromTheFacts` |
| The Opener opens the page and no other shape | `OpenerTests.testOpenMarkOpensTheDrivePageAndNoOtherShape` |
| No title reaches a URL, the fifth verb included | `OpenerTests.testNoTitleEverReachesAURLOrThePasteboard` |
| The press opens the page and moves nothing else in the session | `SessionTests.testTheMarkPressOpensItsPageAndMovesNothingElse` |
| The system is asked through two files only | `SourceRulesTests.testTheSystemIsAskedOnlyThroughTheOpenerAndTheRelauncher` |
| The bitmap is the drawn pixel size, 77 by 28 at 2x and 116 by 42 at 3x, and nil for no picture | `RowTests.testTheMarkIsABitmapAtItsOwnPixelSize` |
| Made once per bytes, height and scale | `RowTests.testTheMarkIsMadeOncePerBytesHeightAndScale` |
| One-pixel stripes shrink to a flat mid grey, so the downsample averages | `RowTests.testTheBitmapAveragesWhatItShrinks` |
| The label draws that bitmap and no other picture | `RowTests.testTheLabelDrawsTheBitmapItWasMade` |
| 14 pt tall, right edge at 348, top at 7, the label's ink, nothing without bytes, one mark on e3 | `RowTests.testTheDocumentsLabelDrawsTheMarkFourteenPointsTallOnTheContentEdge`, `testTheMarkTakesTheLabelsOwnInk`, `testNoMarkDrawsNothingAndMovesNothing`, `testBytesThatAreNotAPictureDrawNothing`, `testOnlyTheDocumentsLabelCarriesTheMark` |
| A byte is read in five files and no sixth | `SourceRulesTests.testAByteIsReadOnlyInTheFiveReaders` |
| Bytes and an address are a press of kind `.openMark` with the address as its target | `RowTests.testTheMarkWithAPageIsAPressThatOpensIt` |
| No address, or no bytes, is no press | `RowTests.testTheMarkWithoutAPageOrWithoutBytesIsNoPress` |
| The words are "Open in Google Drive", shown only where it presses, and the press moves nothing | `RowTests.testTheMarksWordsAreTheDrivePageAndOnlyWhereItPresses` |
| Every word a sheet draws is a word a scene's panel draws | `RenderTests.testEverySheetWordIsAWordAPanelDraws` |
| The guide says what the mark opens | `TestThePresencePageWalksTheThirdLooksPanel` |
| CLAUDE.md stays at its ceiling | `TestCLAUDEmdIsUnderTheCeiling` |

## Post-Completion

- The design reviewer reads `/Users/nailkhusnullin/gdoc-m32h-board/shots`:
  e2 with the stand-in mark in both appearances, e5 without it, the rows
  sheet with "Open in Google Drive" held open under the second DOCUMENTS
  label, and the index's sentences that the shape and the address are
  stand-ins.
- `make release BRANCH=gdoc-v2-m32h-the-mark-sharp-and-a-press` from the
  checkout on `main`; `GO=` with the next version only on Nail's word.
- Nail's hand check, in what he sees: on his own machine the Documents
  line shows the Altery wordmark at the right end with edges as smooth as
  the letters beside it, in the dark and the light look; when the pointer
  rests on it a plate goes darker behind it; after half a second
  it says "Open in Google Drive"; a click opens the logo's page in Google
  Drive in his default browser. The sharpness and the real wordmark are
  his screen's to judge: the renders draw a stand-in.
