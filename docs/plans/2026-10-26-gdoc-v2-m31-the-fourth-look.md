# gdoc v2 Milestone 31: the fourth look. Plan

Tasks: 10
Review: ui

2026-10-26. Nail looked at the panel of v2.20.0 and sent the sentences the
nine backlog items below hold. He decided seven things the same day, and
they are the whole scope: every document on the panel opens on a click; the
mark before a document means one thing, failures only; names survive; the
update buttons say what happened; menus keep the panel open; a large hub
folder cannot slow the Mac; the flaky panel test is fixed. The decision is
`docs/v2/DECISIONS.md`, 2026-10-26, "The fourth look", and the spec
paragraph is changed with it, both on `main` before this run. One commit per
task. Where the code, the spec and this plan disagree, the task stops and
says so.

Every task that lands a backlog item closes that item's file with `git rm`
in the same commit.

## Principles

Serves 4 (every word costs a reader's attention: a mark with one meaning, a
name that can be read, a button that says why it did nothing) and 3 (not
knowing never resolves toward the destructive answer: a click on a row never
stops a review). Strains one thing, named in Task 2: the record's `title` on
the end of a publish may now be the note's cover title where the run stopped
before the read-back. No new field, no new document word.

## Context (from discovery)

Each backlog item's `where` was checked against `main` at 8e25db5.

- **The live row** (`docs/backlog/a-live-row-has-no-link-to-its-document.md`).
  NOW is drawn by `NowList` in `presence/Sources/PresenceUI/PanelView.swift`
  over `NowItem` (`PanelModel.swift`): `.hold`, `.stuck`, `.job`, `.live`.
  The Drive id each one can name is already in the fold:
  `PresenceCore.HoldCard.doc`, `StuckWrite.doc`, `NowRun.doc` and
  `LiveSession.doc` (`presence/Sources/PresenceCore/Now.swift`,
  `LiveSession.swift`). `LiveRow` carries one control, Stop; the stuck card
  already carries the Google Docs icon where it has an id.
- **How a document opens today.** A DOCUMENTS row opens its document from its
  first icon, not from a click on the row: `ActionModel(.openDoc, icon: .gdocs)`
  in `PanelModel.openDoc`, pressed through `PanelActions.perform`, which
  `Session.perform` (`presence/Sources/PresenceApp/SessionActions.swift`)
  turns into `Opener.openDocument(id)` (`Opener.swift`), which builds the URL
  in `PresenceCore/LinkLine.documentURL` from an id held to the Drive shape
  and hands it to `Desk.open`, that is `NSWorkspace.shared.open`. That is the
  one path this plan reuses for every new click. It runs no program and reads
  nothing, so the app's invariant in `CLAUDE.md` holds exactly: six argv lists
  from `Runner.swift`, one SIGINT from `Stopper.swift`, three things read.
  `SourceRulesTests.testTheSystemIsAskedOnlyThroughTheOpenerAndTheRelauncher`
  already holds that no other file asks the system.
- **Stop has no confirm today.** `LiveRow.swift` says so in its header ("There
  is no stop all and no confirm"); the panel's two confirms are a hold's
  Dismiss and the header's Sign out. This plan keeps Stop exactly as it is,
  its own control with no confirm added, and the row click never reaches it.
- **The mark** (`docs/backlog/the-documents-mark-says-nothing-a-person-can-read.md`).
  `PanelModel.glyph(for:)` maps `DocumentState` to a glyph: hold pause, live
  dot, live stopped dim dot, done green check, failed red exclamation,
  running, unknown and pinned-only empty or blank. `Phrases.state` in
  `PanelWords.swift` writes the sub-line, and a failed run reads
  "stopped Tue" beside a stopped watch's "live stopped 12:26": the two look
  alike in both the mark column and the words. `ok: false` is
  `DocumentState.failed` (`PresenceCore/Documents.swift`, `every`).
- **The verb prefix** (`docs/backlog/a-verb-prefix-eats-the-document-name-in-a-row.md`).
  The `where`, `MiddleTitle.swift:70`, is `MiddleTitle.fitted`. TODAY, where
  the item was found, went at the third look, so the case now lives in NOW:
  `LiveRow`'s title "Live · <name>" through `RowView` and `MiddleTitle`, and
  `JobCard`'s title "Publishing · <name>", which is a plain `Text` cut at its
  tail. The stuck card breaks its title on the separator into two lines,
  each through `MiddleTitle`, so its name already keeps its room.
- **The nameless publish** (`docs/backlog/a-publish-that-stopped-in-the-render-has-no-name.md`).
  This one needs Go. `presence.Begin` writes no title, the `build` step
  carries none, and `Run.End` lifts a title from the answer's data only,
  which a publish that failed in `renderNote` (`go/cmd/gdoc/build.go:163`)
  never carries. `cover.Read` has already given the cover title at that
  point. No Swift change alone can name it. The smallest change is the
  item's first shape: a `presence.Named(ctx, title)` that keeps the note's
  cover title on the run, written on the end only where the answer carries
  no title of its own. One caller, `cmdPublish`, after the render. No new
  field, so `TestTheEventCarriesOnlyItsListedFields` is unchanged; the
  `title` field widens from "the title the read-back named" to "the name
  this run knows". It touches `go/internal/presence`, a guarded package,
  which is why the trust guardian signs the decision entry before the run.
  The job card's "Publishing" with no name during the `build` step stays:
  the run has written no line yet that could carry one.
- **The update buttons** (`docs/backlog/neither-update-button-says-anything-when-the-run-did-nothing.md`).
  Lines 73 and 133 have moved: `UpdateAction.installExtension` (about line
  96) keeps one Bool, and `UpdateAction.ran` (about line 160) keeps
  `.failed` and `.unchanged` with their words that nothing reads. Since the
  third look the update is the header's glyph, with a quiet line under the
  version (`PanelHeaderModel.swift`, `version.quiet`, "vX is published"); the
  extension is the Claude Desktop row in SETTINGS (`PanelSettings.swift`),
  whose `SettingsRowModel.quiet` exists. The action words are in
  `go/internal/update/policy.go`: `up_to_date`, `updated`,
  `major_available`, `unreachable`, and two the app never asks for,
  `checked` and `rolled_back`. `refreshHere` in `go/cmd/gdoc/update.go`
  (about line 705) answers the `--desktop` major case `ok: true` with a
  warning.
- **The held press** (`docs/backlog/the-held-press-may-never-reach-a-rows-menu.md`).
  The `where`, `PanelView.swift:355`, is now `RowMenu` at about line 222:
  `.contextMenu` for the secondary click, and `.onLongPressGesture` raising a
  SwiftUI `.popover`, which is a second window. `KeyPanel` closes on
  `windowDidResignKey` (`presence/Sources/Presence/Panel.swift:100`). The long
  press sits on the whole row, icons included. The hand check the item asks
  for was never done.
- **The folder picker** (`docs/backlog/the-hub-pickers-modal-closes-the-panel.md`).
  Its `where`, `Opener.swift:97`, is gone: the third look took out the Hub
  row and `SystemDesk.chooseFolder`, and
  `SourceRulesTests.testTheSystemIsAskedOnlyThroughTheOpenerAndTheRelauncher`
  refuses `NSOpenPanel` in every source file. Nothing is left to build; the
  item closes with a `git rm` in Task 5.
- **The hub walk** (`docs/backlog/a-chosen-hub-is-walked-with-no-bound.md`).
  `HubIndex.scan` (`PresenceCore/HubIndex.swift:79`) walks the whole tree;
  `Session.rescanTheHub` (`Session.swift:634`) starts a `Task.detached` per
  FSEvents burst and never cancels the last one. The folder can now only be
  set with `defaults write com.altery.gdoc.presence hubPath`, so `~` is one
  typo away. Measured 2026-10-26 on Nail's machine: his override, the
  Altery-Platform-Hub, is 1,376 entries outside dot folders, 742 `.md` files,
  7 folders deep; `find ~` with the same pruning did not finish in 120 s.
- **The flaky test** (`docs/backlog/the-override-before-spotlight-test-is-flaky.md`).
  It is an app race, not a test that waits wrong. `readTheHub` starts the
  override's scan on a detached task, and `lookUpTheNotes` (`Session.swift:653`)
  asks Spotlight for every drawn document `index.notes` does not pair yet.
  A draw that lands before the scan does asks Spotlight first, against the
  M27 rule that the override is asked first.
- **Scenes and renders.** 18 scenes in
  `presence/Tests/PresenceUITests/Scenes.swift`: r1 to r9 with r3l, e1, e2,
  e3, e5, e6, e8, e9, e10. `RenderTests.testEveryScreenRendersInBothAppearances`
  writes `<scene>-dark.png`, `<scene>-light.png`, the bar images and
  `index.html` under `PRESENCE_SHOTS`, or a temporary folder when it is not
  set. `Scenes.stillYourCall` ends with the nameless-publish line, which
  Task 2 retires.
- `docs/design/2026-10-17-presence-design-principles.md` holds the checklist
  the design reviewer runs after this run.

### What the lead changes on `main` before the run

Not tasks of this plan:

- `docs/v2/DECISIONS.md`: the 2026-10-26 entry "The fourth look" with
  Nail's seven answers verbatim and its register row, and the trust
  guardian's signature on it, because Task 2 touches `go/internal/presence`.
- `docs/v2/SPEC.md`, section "## Auth", the presence paragraphs: a new
  paragraph "**The fourth look.**" after "**The third look, the same
  evening.**", and one clause in "**The presence is a file gdoc appends to
  and never reads.**" saying the title on a publish's end is the note's cover
  title where the run stopped before its read-back.

The guide, `docs/guide/presence.md`, is this plan's: Task 10 changes "### Now",
"### Documents", "### Settings", "### Update, and Install extension", "## The
note beside the document" and "## The file, and what is never in it".

### Out of scope

An AI-run panel (`docs/backlog/a-presence-app-run-by-an-ai-skill.md`); the
logo in the panel; the skills version or the extension version on the panel;
the nightly zips carrying the app. Also left as they are: a DOCUMENTS row
keeps opening from its icon and gains no row click; the stuck card in NOW
keeps its title "Publish stopped"; `CLAUDE.md` is not edited.

## Development Approach

TDD, one commit per task, the failing test first. `make app-lint` and
`make app-test` green at every commit, and `make test` with the live
variables unset at every commit that touches `go/`. Every size, count,
sentence and number a test states is a literal. No test runs a real
Spotlight query, a real child or a real walk of the home folder: the finder,
the runner and the scan are injected.

Renders are made once, in Task 8, into the one folder the lead hands the
design reviewer. No task opens a PNG. `make app-test` writes renders into the
temporary folder as a side effect, and no task reads them.

Areas, for the worker's brief: presence for Tasks 1 and 3 to 8, go then
presence for Task 2, docs for Tasks 9 and 10.

## Validation Commands

```bash
make app-lint
make app-test
make app
env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test
make vet
```

Once, in Task 8 only:

```bash
rm -rf "$HOME/gdoc-m31-board/shots"
PRESENCE_SHOTS="$HOME/gdoc-m31-board/shots" make app-test
```

## Progress Tracking

Ten tasks. A task is done when every box is ticked and its commit is on the
branch.

## Tasks

### Task 1: the override is asked first, and its walk has a ceiling

**Files:**
- Modify: `presence/Sources/PresenceApp/Session.swift`,
  `presence/Sources/PresenceCore/HubIndex.swift`,
  `presence/Tests/PresenceAppTests/SessionTests.swift`,
  `presence/Tests/PresenceCoreTests/HubIndexTests.swift`
- Delete: `docs/backlog/the-override-before-spotlight-test-is-flaky.md`,
  `docs/backlog/a-chosen-hub-is-walked-with-no-bound.md`

- [ ] the scan is injected into `Session` (a closure from hub path to
      `HubIndex`, `HubIndex.scan` by default), so a test can hold it
- [ ] while the override's scan for the current hub has not landed,
      `lookUpTheNotes` asks Spotlight about nothing; the landing scan's
      recompose asks for what the override did not pair. Test
      `testSpotlightWaitsWhileTheOverrideIsBeingRead`: the scan held on a
      gate, the session draws, the recorded search is `[]`; the gate opens,
      the pairing is the override's note and the search is still `[]`.
      The gate covers only the first scan of the current hub: a watcher
      rescan, or a scan cancelled and replaced by a newer one, never holds
      Spotlight back again (`testOnlyTheFirstScanHoldsSpotlight`).
      `testTheOverrideIsAskedBeforeSpotlight` passes 50 runs in a row
      (`swift test --filter` in a loop, once, not kept as a test)
- [ ] `HubIndex.scan` visits at most 20,000 entries, dot folders not
      counted. Over it the walk stops and answers an index marked over the
      ceiling, with no notes; the session then sets the override aside for
      that hub, stops its watcher and lets the finder answer. The number is
      a named constant, about 14 times the 1,376 entries of Nail's hub.
      Tests over a fixture tree built with a small ceiling handed in:
      under it every note is found, over it nothing is and the index says
      so, and the session asks Spotlight and holds no watcher
- [ ] one walk at a time: `rescanTheHub` cancels the walk before it, and
      `scan` checks for cancellation between entries; a test holds the
      first scan, starts a second, and reads that the first stopped and its
      answer was never taken
- [ ] `git rm` both backlog files
- [ ] `make app-lint`, `make app-test`

### Task 2: a publish that stopped in the render keeps the note's name

**Files:**
- Modify: `go/internal/presence/event.go`, `go/internal/presence/doc.go`,
  `go/internal/presence/presence_test.go`,
  `go/cmd/gdoc/publish.go`, `go/cmd/gdoc/build.go` (`renderNote`, which
  `build` shares), `go/cmd/gdoc/presencestep_test.go`,
  `go/boundary/presence_test.go`
- Modify: `presence/Tests/PresenceUITests/Scenes/r2.jsonl` (the fixture),
  `presence/Tests/PresenceUITests/Scenes.swift` (r2's caption,
  `stillYourCall`), `presence/Tests/PresenceUITests/RenderTests.swift`
- Delete: `docs/backlog/a-publish-that-stopped-in-the-render-has-no-name.md`

- [ ] Go, test first: `presence.Named(ctx, title)` keeps a title on the run
      that is on the context, safe on a nil run and a context with no run,
      capped at the same 200 runes. `End` writes it only where `lift` found
      no title in the answer: `TestANamedTitleIsWrittenOnlyWhereTheAnswerCarriesNone`
      and `TestANamedTitleOutsideARunWritesNothing`. No new field:
      `TestTheEventCarriesOnlyItsListedFields` is unchanged and green
- [ ] `cmdPublish` calls it once, after the render has read the cover and
      before it returns, with the note's cover title. If `renderNote`
      returns no cover title on a failure past `cover.Read`, it sets the
      title as soon as `cover.Read` succeeds and nothing else in its
      answer moves. `TestAPublishThatStoppedInTheRenderNamesTheNote` over a
      note with an `svg` picture, the r2 case, reads `title` on the end
      line. `TestAPublishWhoseCoverCannotBeReadWritesNoTitle` holds the other
      side
- [ ] the one caller is held in `go/boundary/presence_test.go` as
      `TestNamedHasOnlyPublishAsCaller`, using `callsOf` the way
      `TestStepHasOnlyPublishAsCaller` does (the trust guardian's condition 1)
- [ ] `renderNote` sets `doc.Title` right after `cover.Read`, and the failure
      envelopes of `build` and `publish` stay byte for byte as today
      (`buildFailed`, `publish.go`). `TestAFailedBuildStillPrintsTheLogoBlock`
      stays green and unedited (the trust guardian's condition 2)
- [ ] `go/internal/presence/doc.go`'s paragraph on the one document word
      says the end's title is the answer's, or the name the run was given
      where the answer has none, and names the two tests
- [ ] Swift: r2's fixture puts the title on the end line, as gdoc now writes
      it, and the card reads "Publish stopped · Chapter 4"; the caption says
      so; the last line of `stillYourCall` goes.
      `testTheIndexOpensWithWhatIsStillYourCall` changes in two places: the
      count, 8 today, as a new literal, and the subject "nameless", which it
      no longer asserts
- [ ] `git rm` the backlog file
- [ ] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`, `make vet`,
      `make app-lint`, `make app-test`

### Task 3: the name keeps its room behind a verb

**Files:**
- Modify: `presence/Sources/PresenceUI/Components/MiddleTitle.swift`,
  `RowView.swift`, `LiveRow.swift`, `JobCard.swift`,
  `presence/Sources/PresenceUI/PanelWords.swift`, `PanelModel.swift`,
  `presence/Tests/PresenceUITests/RowTests.swift`, `CardTests.swift`
- Delete: `docs/backlog/a-verb-prefix-eats-the-document-name-in-a-row.md`

- [ ] the item's first shape: a title is handed over as two pieces, the
      prefix ("Live", "Live stopped", "Publishing") and the name. The prefix
      and its separator are drawn whole, and the eight-character floor and
      the middle cut are measured on the name alone, in the room the prefix
      leaves. A title with no prefix draws exactly as today
- [ ] `LiveRowModel` and `JobCardModel` carry the two pieces; the live row
      and the job card draw through `MiddleTitle`, so the job card's name
      is cut in the middle like every other name and not at its tail
- [ ] tests with literals: "Live stopped" in front of
      "Ch 5. Commercial Model v0.1" shows at least eight characters of the
      name's head and keeps " v0.1"; the job card with the same name does
      the same; a name with no prefix measures as it did
      (`testAPrefixTakesItsRoomAndTheFloorCountsTheName`)
- [ ] `git rm` the backlog file
- [ ] `make app-lint`, `make app-test`

### Task 4: the mark means one thing, a run that failed

**Files:**
- Modify: `presence/Sources/PresenceUI/PanelModel.swift` (`glyph(for:)`),
  `PanelWords.swift` (`Phrases.state`), `Words.swift`,
  `presence/Tests/PresenceUITests/StateTests.swift`, `FieldsTests.swift`,
  `Scenes/e1.jsonl` (the fixture), `Scenes.swift` (e1's caption, and any
  caption that names the check or the dot on a DOCUMENTS row)
- Delete: `docs/backlog/the-documents-mark-says-nothing-a-person-can-read.md`

- [ ] a DOCUMENTS row draws the red exclamation when its last run ended
      `ok: false`, and the blank glyph in every other state: done, live
      stopped, live, hold, running, unknown and pinned only. The column
      stays, so every title starts at one edge. A table test over the eight
      states (`testADocumentRowIsMarkedOnlyWhenItsLastRunFailed`)
- [ ] a failed run's sub-line names the command and says failed, "reply
      failed Tue", where it said "stopped Tue"; a stopped watch keeps "live
      stopped 12:26"; a test states both lines and that neither contains
      the other's word (`testAFailedRunDoesNotReadAsAStoppedOne`)
- [ ] `verified: false` stays a run that worked: no mark, and the word it
      has today
- [ ] e1's last five carry one failed reply, so a render shows the one mark
      the list can have; the caption says what the mark means
- [ ] `git rm` the backlog file
- [ ] `make app-lint`, `make app-test`

### Task 5: a press on a row opens the document

**Files:**
- Modify: `presence/Sources/PresenceUI/PanelModel.swift` (`NowItem`),
  `PanelView.swift` (`NowList`, `RowMenu`), `Components/LiveRow.swift`,
  `HoldCard.swift`, `StuckCard.swift`, `JobCard.swift`, `Words.swift`,
  `presence/Tests/PresenceUITests/CardTests.swift`, `StateTests.swift`,
  `presence/Tests/PresenceCoreTests/SourceRulesTests.swift`
- Delete: `docs/backlog/a-live-row-has-no-link-to-its-document.md`,
  `docs/backlog/the-hub-pickers-modal-closes-the-panel.md`

- [ ] every `NowItem` carries the document it opens: the hold's, the stuck
      write's, the job's and the watch's id, and nothing where the id is
      missing or not Drive-shaped (`LinkLine.isDocumentID`). A publish
      still running has none. A table test over the four shapes with and
      without an id (`testEveryNowRowThatNamesADocumentOpensIt`)
- [ ] the whole row or card is the click target, as the lead decided, and
      the press leaves as `PanelActions.perform(.openDoc, id)`, the kind a
      DOCUMENTS row's first icon sends, so `Session.perform` reaches
      `Opener.openDocument` and nothing new asks the system. The title area
      carries the Google Docs icon's tooltip. A hold card that is asking
      its question opens nothing
- [ ] the controls on a row keep their own press: Stop sends `.stopLive`
      alone and the row sends `.openDoc` alone. A test reads the two
      closures the view is built with through a recording `PanelActions`
      (`testTheRowOpensAndOnlyStopStops`). No offscreen click test: that
      test and Nail's hand check after the run hold the behaviour.
      Stop's state, ring and process check are unchanged and it gains no
      confirm
- [ ] `docs/backlog/the-hub-pickers-modal-closes-the-panel.md` is closed with
      no code: the picker went at the third look and
      `testTheSystemIsAskedOnlyThroughTheOpenerAndTheRelauncher` refuses
      `NSOpenPanel`. The commit message names that test
- [ ] `git rm` the two backlog files
- [ ] `make app-lint`, `make app-test`

### Task 6: the held press opens the system menu, and the panel stays

**Files:**
- Modify: `presence/Sources/PresenceUI/PanelView.swift` (`RowMenu`),
  a new `presence/Sources/PresenceUI/Components/MenuAnchor.swift`,
  `presence/Tests/PresenceUITests/CardTests.swift`,
  `presence/Tests/PresenceCoreTests/SourceRulesTests.swift`
- Delete: `docs/backlog/the-held-press-may-never-reach-a-rows-menu.md`

- [ ] the right-click stays SwiftUI's `.contextMenu`. Only the held press
      changes: it builds an `NSMenu` from the same `MenuItemModel`s the
      `.contextMenu` draws, and pops it up with `popUp(positioning:at:in:)`
      from a `MenuAnchor`, an `NSViewRepresentable` placed behind the row's
      title area. The anchor's `hitTest` returns nil, so it never catches a
      click meant for the row or for its icons
- [ ] no second window exists to take key. The gesture is on the title area and not
      over the five icons. A source rule: no `.popover(` under
      `Sources/PresenceUI`. A test through an injected menu presenter: the
      held press hands it the model's words and nothing else
- [ ] `git rm` the backlog file
- [ ] `make app-lint`, `make app-test`

### Task 7: the update buttons say what happened

**Files:**
- Modify: `presence/Sources/PresenceApp/UpdateAction.swift`,
  `Session.swift`, `SessionActions.swift`,
  `presence/Sources/PresenceUI/Facts.swift`, `PanelHeaderModel.swift`,
  `PanelSettings.swift`, `Words.swift`,
  `presence/Tests/PresenceAppTests/UpdateActionTests.swift`,
  `presence/Tests/PresenceUITests/StateTests.swift`,
  `Scenes/r8.jsonl` and `Scenes/e6.jsonl` (the fixtures), `Scenes.swift`
  (their captions)
- Delete: `docs/backlog/neither-update-button-says-anything-when-the-run-did-nothing.md`

- [ ] read `go/cmd/gdoc/update.go` and `go/internal/update/policy.go` first
      and confirm the `action` word each case prints, including the
      `--desktop` run whose newer release is a major one. The panel tells
      cases apart by `ok` and `action` alone and never by a warning or an
      error sentence. A case that cannot be told apart that way falls to
      the general sentence, and the commit says which
- [ ] `installExtension` hands back a standing like `update` does, and
      `Facts` carries the last run of each button with its moment
- [ ] the header's quiet line under the version says, for the update:
      `major_available`, "Only a major version is newer. gdoc update --major
      installs it."; `unreachable`, "GitHub did not answer at 14:02. Nothing
      changed."; `ok: false` or an answer the app cannot read, "The update
      stopped at 14:02. gdoc update in a terminal says why." `up_to_date`
      keeps the one-second check and adds no line; `updated` relaunches as
      today
- [ ] the Claude Desktop row's quiet line says, for the extension: the major
      case, "Update gdoc with --major first, then install the extension.";
      `ok: false`, "The install stopped at 14:02. gdoc update --desktop in a
      terminal says why."; any other run that handed nothing over, "Claude
      Desktop was not handed the extension at 14:02."
- [ ] every sentence is a fixed string with the clock in it, none carries
      the child's own words, and each fits two lines at 320 pt; a sentence
      stays until the next press of either button or the next launch of
      the app. A table test over every case and both rows
      (`testEachRunThatInstalledNothingSaysWhyInAFixedSentence`)
- [ ] r8 carries the unreachable update and e6 the extension's major case;
      their captions say so
- [ ] `git rm` the backlog file
- [ ] `make app-lint`, `make app-test`

### Task 8: the renders, once

**Files:**
- Modify: `presence/Tests/PresenceUITests/Scenes.swift` (captions only, if
  the check below finds one stale)

- [ ] read every caption in `Scenes.swift` as text against Tasks 2 to 7,
      and fix any caption that describes a drawing this plan changed. The
      eighteen scenes stay eighteen
- [ ] `rm -rf "$HOME/gdoc-m31-board/shots"`, then
      `PRESENCE_SHOTS="$HOME/gdoc-m31-board/shots" make app-test`, once
- [ ] check with `ls` and nothing else: each of the 18 scene ids has its
      `<id>-dark.png`, and `index.html` is there. The folder also holds the
      `-bar-` images and the sheet pages, which are not counted. Open no PNG. The design reviewer reads them after the run
- [ ] the commit message names the folder as an absolute path, which the
      lead hands to `gdoc-design-reviewer`

### Task 9: Verify acceptance criteria

**Files:**
- Modify: this plan

- [ ] every row of the Acceptance table at the foot of this plan names a
      test that exists and passes; a test a task named differently is
      renamed here, not in the code
- [ ] `git diff --name-only main...HEAD`: nothing under `go/` outside
      `go/internal/presence/`, `go/cmd/gdoc/` (publish, `build.go`'s
      `renderNote` and their tests) and `go/boundary/presence_test.go`,
      `CLAUDE.md` and `docs/v2/` not in the list, and the nine backlog files
      of this plan deleted
- [ ] `grep -rn "popover(" presence/Sources` prints nothing;
      `grep -rn "NSWorkspace\|NSPasteboard" presence/Sources` names
      `Opener.swift` and `Relauncher.swift` only
- [ ] `make vet`, `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`,
      `make app-lint`, `make app-test`, `make app`

### Task 10: [Final] Update documentation

**Files:**
- Modify: `docs/guide/presence.md`,
  `docs/design/2026-10-17-presence-design-principles.md`; move this plan

- [ ] the guide: "### Now", every row that names a document opens it on a
      click, and Stop is its own control; "### Documents", the mark is a
      failed last run and nothing else, the right-click and the held press
      open one menu; "### Settings" and "### Update, and Install extension",
      the sentences a run that installed nothing leaves; "## The note
      beside the document", the override's ceiling and what happens over it;
      "## The file, and what is never in it", the title on a publish's end
- [ ] the principles file gains "Amendments, 2026-10-26, the fourth look":
      the mark means a failed run; the eight-character floor counts the
      name and not the verb in front of it; the held press is the system
      menu and never a second window
- [ ] `CLAUDE.md` untouched
- [ ] move this plan to `docs/plans/completed/`
- [ ] `make app-lint`, `make app-test`

## Acceptance

Nail's decisions of 2026-10-26, with the test that holds each answer. A
row's test is the one that fails if its answer is taken away.

| Nail's decision | The answer | The test that holds it |
|---|---|---|
| every document on the panel opens on a click | each NOW row or card that names a Drive id carries it, and a press on the row sends `.openDoc` down the DOCUMENTS path | `StateTests.testEveryNowRowThatNamesADocumentOpensIt`; the gesture itself is held only by Nail's hand check, since no test sends a real click |
| a click on the row never stops a review | Stop sends `.stopLive` alone, the row `.openDoc` alone | `CardTests.testTheRowOpensAndOnlyStopStops` |
| the mark means one thing: failures only | the exclamation on `ok: false`, blank in every other state | `StateTests.testADocumentRowIsMarkedOnlyWhenItsLastRunFailed` |
| a failed run must not look like a stopped one | "reply failed Tue" against "live stopped 12:26" | `StateTests.testAFailedRunDoesNotReadAsAStoppedOne` |
| names survive a verb | the floor and the cut are measured on the name | `RowTests.testAPrefixTakesItsRoomAndTheFloorCountsTheName` |
| names survive a publish that stopped in the render | the end carries the note's cover title | `TestAPublishThatStoppedInTheRenderNamesTheNote`, `TestANamedTitleIsWrittenOnlyWhereTheAnswerCarriesNone`, `TestNamedHasOnlyPublishAsCaller` |
| the update buttons say what happened | one fixed sentence with its moment per case, on the row of the button | `StateTests.testEachRunThatInstalledNothingSaysWhyInAFixedSentence`, `UpdateActionTests` over the standings |
| menus keep the panel open | the held press is the system menu from the row's own view; no popover in the source | `SourceRulesTests` popover rule, the menu presenter test |
| the folder picker keeps the panel open | there is no picker since the third look | `SourceRulesTests.testTheSystemIsAskedOnlyThroughTheOpenerAndTheRelauncher` |
| a large hub folder cannot slow the Mac | 20,000 entries at most, one walk at a time, Spotlight over the ceiling | `HubIndexTests` ceiling test, `SessionTests` cancel and over-ceiling tests |
| the flaky panel test is fixed | Spotlight waits for the override's scan | `SessionTests.testSpotlightWaitsWhileTheOverrideIsBeingRead` |

## Post-Completion

These are the lead's, after the run and before the PR.

**The design reviewer, once.** Call `gdoc-design-reviewer` with the renders
folder as an absolute path, `/Users/nailkhusnullin/gdoc-m31-board/shots`:
the 18 dark whole-panel images, the captions as text from `index.html`, and
light only where dark has a finding. The lead does not open the images. The
findings go to the lead, who decides fix now or tomorrow; a fix now is a
small follow-up commit on the branch and no second review.

**The hand checks, Nail's.** On the built app: hold a press on a DOCUMENTS
row for half a second and say whether the menu opens and the panel stays;
click a live row's title and say whether the document opens and the watch
goes on; click Stop and say whether it stops and nothing opens.

**The release.** `make release BRANCH=gdoc-v2-m31-the-fourth-look` stops at
a green PR. The merge and the tag wait for Nail's word.
