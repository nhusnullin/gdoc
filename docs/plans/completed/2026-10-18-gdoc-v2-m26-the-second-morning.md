# gdoc v2 Milestone 26: the second morning. Plan

2026-10-18. Nail used the panel for a morning and said six things, in this
order: the scrolling is wrong, the account and its sign-out belong at the
top, the group they leave is not HEALTH any more, Open chat opens Claude
and not the session the document came from, the menu bar mark is too dim,
and DOCUMENTS should put what gdoc just made first, then the most used, with
a way to pin a document so it never drops off. The sign-out that did nothing
was a seventh and shipped alone as v2.17.0. His answers to the questions the
list raised: the panel pins its top and bottom and scrolls its middle; the
session is worth measuring; the duplication between DOCUMENTS and TODAY
stays. Four more came while this plan was written: the gdoc version and
its update belong in the header too; a document must be removable from the
list and from the statistics; the md icon is grey on every row, because the
hub has no picker and his hub path was never set; and the Thread mark row
goes, "a pointless thing". The decision is `docs/v2/DECISIONS.md`, 2026-10-18, "The second
morning"; the spec paragraph is changed with it. One commit per task. Where
the code, the spec and this plan disagree, the task stops and says so.

The visual source of truth stays `docs/design/2026-10-17-presence-screens-v3.html`
where this plan does not change it; where it does, this plan's sentence wins
and the task updates the screens file's affected scene captions.

## Principles

Serves 4 (what the person reaches for is where the hand goes) and 3 (one
action vocabulary, one place per fact). Strains 2 by one more field in the
record, named in the decision.

## Context (from discovery)

- `presence/Sources/PresenceUI/PanelView.swift`: the header, then one
  `ScrollBox(height: bodyHeight)` holding every section; `Grid.panelMaxHeight
  = 600`. `ScrollBox` draws its thumb and fade when there is more under them.
  `PanelHeader.swift` is 36 pt: the mark, "gdoc", the address in the
  quietest grey, "and no action".
- `PanelHealth.swift` builds the HEALTH rows: Account (check, address,
  sign-out icon, or the inline "Sign out?" question), gdoc version, Thread
  mark, Launch at login (switch), Quit; Network and Claude Desktop when a
  fact carries them. `PanelModel.swift:352` pins HEALTH first on three
  screens. `SessionActions.swift` `askThenSignOut`, `signOut`.
- `presence/Sources/PresenceUI/BarImage.swift`: `idleOpacity = 0.45`,
  `BarTests` states it.
- `presence/Sources/PresenceCore/Documents.swift`: one row per document,
  newest first by its last action, the newest five until a person opens the
  rest, a 14-day horizon. Row actions: `openDoc`, `openNote`, `openChat`,
  `copyLink` (`IconStrip`, four icons: gdocs, md, branch, link).
- `presence/Sources/PresenceApp/Preferences.swift`: six keys in
  UserDefaults, a source scan holds the count; dismissals pruned at
  sixty-two days; the launch-at-login answer kept.
- `presence/Sources/PresenceApp/Opener.swift` `continueWithClaude()` is
  `desk.activate(Claude Desktop)` and no URL: MEASURED.md 2026-10-17 found
  the scheme claimed and no grammar documented. 2026-10-18: the strings of
  `/Applications/Claude.app/Contents/Resources/app.asar` carry
  `claude://code/continue?session=last&source=desktop_action`,
  `claude://code/new?source=desktop_action` and
  `claude://code/needs-input?source=desktop_action`, so the Code tab has a
  grammar. Whether `session=<id>` takes a session id and not only `last` is
  Task 6's measurement, done by a person watching the window.
- Claude Code hands its children `CLAUDE_CODE_SESSION_ID` beside
  `CLAUDECODE` (seen in this session's own environment). The record's
  `via: code` is read from `CLAUDECODE` in `main.go` `viaOfEnv`; the event
  struct is `go/internal/presence/event.go`, its fields allowlisted by
  `TestTheEventCarriesOnlyItsListedFields`.
- `docs/design/2026-10-17-presence-design-principles.md` holds the checklist
  a reviewer runs over renders before Nail sees them; Task 9 runs it.

## Development Approach

TDD, one commit per task, the failing test first; `make app-lint`,
`make app-test`, `make app` green at every commit; `make test` with the live
variables unset where Go changes. Every size, colour, count and word a test
states is a literal. Renders go under `PRESENCE_SHOTS`, outside the
repository. No task opens a real URL or starts the real app in a test.

## Validation Commands

```bash
make app-lint
make app-test
make app
make vet
env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test
PRESENCE_SHOTS=/tmp/presence-shots make app-test
```

## Progress Tracking

Eleven tasks. A task is done when every box is ticked and its commit is on the
branch.

## Tasks

### Task 1: the mark is bright

**Files:**
- Modify: `presence/Sources/PresenceUI/BarImage.swift`,
  `presence/Tests/PresenceUITests/BarTests.swift`

- [x] the idle mark is drawn at full opacity, the same ink as every other
      state's mark; `idleOpacity` goes, and the test that said 0.45 says the
      idle and the live mark are the same ink
- [x] the render index's "Still your call" line about the dim mark is
      replaced by the decision's sentence
- [x] `make app-lint`, `make app-test`

### Task 2: the account lives in the header

**Files:**
- Modify: `presence/Sources/PresenceUI/Components/PanelHeader.swift`,
  `PanelHealth.swift`, `PanelModel.swift`, `PanelView.swift`,
  `presence/Tests/PresenceUITests/RowTests.swift`, `StateTests.swift`,
  `RenderTests.swift`, the scene fixtures that touch the account

- [x] the header carries the account's state: the status glyph (check when
      the token works, exclamation when it does not, a ring while nobody has
      asked), the address, and the one action, Sign out, as the icon the
      HEALTH row carried; with no token the header shows Sign in where the
      icon was; a test states the three pieces and the height, which stays
      36 pt
- [x] "Sign out?" asks in place of the header line: the question, its
      consequence line under it, Sign out and Keep where the actions were;
      the question waits for its answer as PR 91 made it
- [x] the signed-in banner, the sign-in countdown and the Not-you line move
      with the account: they are drawn under the header, not under a HEALTH
      row
- [x] HEALTH loses the Account row; the Account changed quiet line goes with
      it under the header
- [x] the header carries the gdoc version after the name, in the caption
      grey ("gdoc" then "v2.16.0"), and the update action where a newer
      release is listed: the same icon and the same ring-while-it-runs the
      gdoc row had, now beside the version; with nothing newer, no icon;
      HEALTH loses the gdoc row too; a test states the header's pieces with
      and without an update
- [x] `make app-lint`, `make app-test`

### Task 3: the group is SETTINGS

**Files:**
- Modify: `presence/Sources/PresenceUI/Words.swift` or `PanelWords.swift`,
  `PanelHealth.swift` and its callers, `docs/guide/presence.md`, the scene
  captions

- [x] the bottom group's label is SETTINGS: Hub (Task 7), Launch at login,
      Quit, and the Network and Claude Desktop rows when a fact carries
      them; the Thread mark row goes, and the mark stays where it was, in the
      header's own mark; the type and file names follow the word where a
      name said health, and a test states the label and the absent row
- [x] `PanelModel`'s "pinned first" rule for the three screens stays for the
      two that are still about the group (no token, a sign-in in flight) and
      drops the question, which now lives in the header
- [x] `docs/guide/presence.md` names the group and what moved up
- [x] `make app-lint`, `make app-test`

### Task 4: the middle scrolls, the edges stay

**Files:**
- Modify: `presence/Sources/PresenceUI/PanelView.swift`,
  `Components/ScrollBox.swift`, `presence/Tests/PresenceUITests/ScrollTests.swift`,
  `RenderTests.swift`

- [x] the panel is header, NOW, the middle, SETTINGS, in that order; the
      header, NOW and SETTINGS never scroll; the middle (DOCUMENTS, TODAY, Yesterday)
      is the one `ScrollBox`, and its height is what is left of 600 pt after
      the three fixed parts; a test measures the four heights on scene e3
- [x] NOW, when it alone is taller than the panel allows, is capped at half
      the body and scrolls inside itself, so SETTINGS is never pushed off; a test
      on a fixture with four hold cards
- [x] the scroll thumb is an overlay 3 pt inside the box's right edge, drawn
      while the content moves and for one second after, and never otherwise;
      the fade stays; a test states the inset, the width and the second
- [x] `make app-lint`, `make app-test`

### Task 5: DOCUMENTS by what was just made, then by use

**Files:**
- Modify: `presence/Sources/PresenceCore/Documents.swift`,
  `presence/Tests/PresenceCoreTests/DocumentsTests.swift`, the scene
  fixtures and `RenderTests.swift`

- [x] the order: first every document with an action today, newest first;
      then the rest of the fourteen days by the count of finished actions in
      those days, most first, ties newest first; a table test over a
      fixture with three documents touched today and four older ones with
      different counts
- [x] the row's sub-line keeps saying the last action and its time; a
      document listed by use shows nothing new, because a count is a
      judgement the panel does not print
- [x] "Show N more" and the horizon are unchanged
- [x] `make app-lint`, `make app-test`

### Task 6: pinned and hidden documents

**Files:**
- Modify: `presence/Sources/PresenceApp/Preferences.swift`,
  `presence/Sources/PresenceCore/Documents.swift`,
  `presence/Sources/PresenceUI/PanelModel.swift`, `IconStrip.swift`,
  `IconShapes.swift`, `SessionActions.swift`, tests beside each

- [x] two more preferences keys, `pinned` and `hidden`: lists of document
      ids, each with the moment and the last title; never pruned by age; the
      source scan's count becomes eight and its sentence says why
- [x] the four icons stay four. A DOCUMENTS row has a menu, on a secondary
      click and on a press held half a second: "Pin" or "Unpin", and "Hide
      from the list". A pinned row shows a small pin glyph in its status
      column before the state glyph; a test over the menu's words in both
      states and the glyph
- [x] a hidden document leaves DOCUMENTS and its actions stop counting for
      the order; TODAY and Yesterday still show its runs, because the log is
      the log; the next finished action gdoc does on that document brings it
      back into the list, which is the one way back, and the guide says so;
      a test over hide, the count, and the return
- [x] pinned documents are listed first in DOCUMENTS, in the order they were
      pinned, newest pin first, and never fall out of the list when their
      fourteen days pass; a pinned document that the record has no line for
      any more still draws its last known title and its open action from the
      pin's own stored title and id, which the pin keeps for that reason; a
      test over a fixture whose pinned document is sixty days old
- [x] the header count counts them: "14 days · N" counts pinned rows too and
      hidden rows not, and the index caption says so
- [x] `make app-lint`, `make app-test`

### Task 7: the hub is chosen in the panel

**Files:**
- Modify: `presence/Sources/PresenceUI/PanelHealth.swift` (the SETTINGS
  rows), `presence/Sources/PresenceApp/SessionActions.swift`,
  `Preferences.swift`, `docs/guide/presence.md`, tests beside each

- [x] a Hub row in SETTINGS: the chosen folder as `~/…`, or "Choose…" when
      none; pressing it opens the system folder panel (`NSOpenPanel`, folders
      only, the app's own window and no program), and the choice goes into
      the `hubPath` key the guide already names; a test over the row's two
      states and the stored path through an injected chooser
- [x] the md icon's tooltip with no hub names the row ("Choose the hub in
      SETTINGS"), so a grey icon says why
- [x] the guide's `defaults write` paragraph becomes the row, and keeps the
      command as the way from a terminal
- [x] `make app-lint`, `make app-test`

### Task 8: the session a run came from

**Files:**
- Modify: `go/internal/presence/event.go`, `go/cmd/gdoc/main.go`,
  `go/internal/presence/doc.go`, `go/internal/presence/presence_test.go`,
  `go/cmd/gdoc/*_test.go`, `presence/Sources/PresenceCore/Event.swift`,
  `Documents.swift`, `presence/Sources/PresenceApp/Opener.swift`,
  `presence/Tests/...`

- [x] the record gains `session`: the value of `CLAUDE_CODE_SESSION_ID`
      when `via` is `code`, read in `viaOfEnv` beside `CLAUDECODE`, written
      on the start line and nowhere else; absent under terminal and chat; the
      allowlist test names it; a test shows the field under Claude Code and
      its absence elsewhere, and `TestTheFlagChangesNoByteOfTheObject` still
      holds
- [x] the Swift `Event` decodes `session`, and a document remembers the
      session of its latest `via: code` run
- [x] Open chat on a document whose latest run came from Claude Code with a
      session would open `claude://code/continue?session=<id>&source=gdoc`;
      the link is built and opened by nobody, which is the box below; a
      document with a chat run keeps the activation it has; a document with
      neither keeps the activation; the URL is built by one function with a
      test over the three cases, and the opener's allowlist of schemes gains
      this one path and no other
- [x] the measurement is done and is in `docs/v2/MEASURED.md` ("What
      `claude://code/continue` does from outside the app", 2026-10-18): none
      of the three forms switched the window, so the link is not shipped;
      the opener keeps the activation, the URL-building function and its
      test stay (the day a form switches, the opener has the id), and this
      task's commit says so
- [x] `make vet`, `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`,
      `make app-lint`, `make app-test`

### Task 9: the words

**Files:**
- Modify: `docs/guide/presence.md`, `CLAUDE.md` (the presence invariant's
  field list, if it names fields), `go/internal/presence/doc.go`,
  `docs/design/2026-10-17-presence-screens-v3.html` (scene captions only)

- [x] the guide walks the panel in its new order: header with the account
      with its version and update, and Sign out, NOW, DOCUMENTS with pins,
      hides and the order rule, TODAY, Yesterday, SETTINGS with the Hub row
      and no Thread mark; and names the session link and when it is used
      (`TestThePresencePageWalksTheSecondMorningsPanel` holds the phrases and
      the walk's order)
- [x] the scene captions in the screens file that say HEALTH or place the
      account there say the new place (r8, e1, e5, e8, e9, e10 and the foot;
      the drawn scenes stay the 17 October mockup, and each caption says what
      moved)
- [x] CLAUDE.md stays under 300 lines (300, at the ceiling; its presence
      invariant names no field, so the `session` field changed nothing there)
- [x] `make test`, `make app-lint`

### Task 10: the reviewer's pass

**Files:**
- Modify: whatever the pass finds under `presence/`

- [x] render every scene (`PRESENCE_SHOTS=... make app-test`) and run the
      checklist of `docs/design/2026-10-17-presence-design-principles.md`
      over every scene as a reviewer who drew nothing (a subagent that reads
      images), against the mockup where it still applies and against this
      plan's sentences where it does not; every blocking or should-fix
      finding is fixed here and listed in the commit message
- [x] `make app-lint`, `make app-test`, `make app`

### Task 11: the acceptance rows and the close

**Files:**
- Modify: this plan, `docs/v2/DECISIONS.md` only if a task found the
  decision wrong

- [x] an acceptance table at the foot of this plan: each of Nail's ten
      sentences with the test that holds its answer
- [x] the render index's "Still your call" list drops what this plan
      decided and keeps the rest (the dim mark goes, and is said once as
      `Scenes.markRule`, because the mockup still draws nothing at rest)
- [x] move this plan to `docs/plans/completed/`
- [x] `make vet`, `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`,
      `make app-lint`, `make app-test`, `make app`

## Acceptance

Nail's ten sentences, in the order of this plan's first paragraph, with the
answer that shipped and the test that holds it. A test named here is the one
that fails if the answer is taken back.

| Nail said | The answer | The test |
|---|---|---|
| the scrolling is wrong | header, NOW, SETTINGS never move; DOCUMENTS, TODAY and Yesterday are the one scroll box, and NOW stops at half the body | `testTheFourPartsAddUpToTheCapOnE3` (RenderTests), `testNowStopsAtHalfTheBodyAndSettingsKeepsItsPlace`, `testTheThumbSitsThreePointInAndLastsOneSecondAfterTheMove` (ScrollTests) |
| the account and its sign-out belong at the top | the header is 36 pt and carries the glyph, the address and Sign out, and asks "Sign out?" in its own line | `testTheHeaderIs36WithTheMarkTheNameTheVersionAndTheAccount`, `testTheQuestionAsksInPlaceOfTheHeaderLine` (RowTests), `testTheHeaderClaimsNothingAboutTheAccountBeforeTheFirstAnswer` (StateTests) |
| the group they leave is not HEALTH any more | the group is SETTINGS: Network, Claude Desktop, Hub, Launch at login, Quit | `testSettingsRowsKeepTheirOrderOnEveryScreen` (CardTests), `testSettingsKeepsItsOrderOnEveryScene` (RenderTests) |
| Open chat opens Claude and not the session the document came from | the record carries `session` on a Claude Code start, a row remembers it, and the link is built; none of the three `claude://code` forms switched the window, so the opener keeps the activation and the link is not shipped | `TestTheSessionOfACodeRunIsOnItsStart`, `TestTheSessionIsOnTheStartOfACodeRunAndNowhereElse` (Go), `testARowRemembersTheSessionOfItsLatestCodeRun` (FieldsTests), `testTheSessionLinkIsBuiltFromASessionIdOnly` (LinkLineTests); the measurement is `docs/v2/MEASURED.md`, 2026-10-18 |
| the menu bar mark is too dim | the idle mark is the live mark's own ink, and `idleOpacity` is gone | `testOneTemplateImagePerState` (BarTests) |
| DOCUMENTS should put what gdoc just made first, then the most used, with a way to pin | today's documents newest first, then the fortnight by the count of finished actions; pins come before all of them, newest pin first, and never fall out | `testTheOrderIsTodayNewestFirstThenTheFortnightByUse`, `testADocumentListedByUseShowsNothingNew`, `testAPinnedDocumentIsFirstAndNeverFallsOutOfTheList` (DocumentsTests), `testAPinnedDocumentWithNoLinesDrawsThePinsOwnRow` (StateTests), `testPinsComeBackNewestFirstAndUnpinLeavesNothing` (PreferencesTests) |
| the gdoc version and its update belong in the header too | the version follows the name in the caption grey, with the update icon beside it where a newer release is listed and no icon otherwise | `testTheHeaderIs36WithTheMarkTheNameTheVersionAndTheAccount` (RowTests) |
| a document must be removable from the list and from the statistics | Hide from the list, in the row's own menu: the row goes and its actions stop counting, TODAY keeps its runs, and the next finished action brings it back | `testAHiddenDocumentLeavesTheListUntilTheNextActionOnIt` (DocumentsTests), `testTheRowMenuSaysPinOrUnpinAndHide` (StateTests), `testAHideKeepsItsMomentAndIsNeverPruned` (PreferencesTests) |
| the md icon is grey on every row, because the hub has no picker | a Hub row in SETTINGS: the folder as `~/…` or "Choose…", the system folder panel behind it, and the grey icon's tooltip names the row | `testTheHubRowIsTheFolderOrTheWayToChooseOne` (StateTests), `testChoosingTheHubWritesItDownAndReadsTheNotesAtOnce` (SessionTests) |
| the Thread mark row goes, "a pointless thing" | there is no mark row: the mark is the header's own, drawn at the panel's top left | `testSettingsRowsKeepTheirOrderOnEveryScreen` (CardTests) |

Two more rows, for the two answers the questions above raised:

| The question | The answer | The test |
|---|---|---|
| does the count count a pin and a hide | "14 days · N" counts a pinned row however old it is and a hidden one not at all | `testAPinnedDocumentIsFirstAndNeverFallsOutOfTheList`, `testAHiddenDocumentLeavesTheListUntilTheNextActionOnIt` (DocumentsTests) |
| does the page say all of it | the guide walks the panel in its new order, names the session and the one way a hidden document comes back | `TestThePresencePageWalksTheSecondMorningsPanel` (`go/boundary`) |
