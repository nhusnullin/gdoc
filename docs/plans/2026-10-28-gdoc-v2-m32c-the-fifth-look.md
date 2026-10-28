# gdoc v2 Milestone 32c: the fifth look. Plan

Tasks: 8
Review: ui

2026-10-28. Nail looked at the panel of v2.24.0 and sent four sentences:
details when the pointer is on a document, a click opens it, a wider
panel, the firm's mark. He answered the lead's five questions the same
evening: "ok, run with the recommendation: 1 yes, 2 a, 3 360, 4 yes, 5 b
after". The decision is `docs/v2/DECISIONS.md`, 2026-10-28, "The fifth
look", and the spec paragraph is changed with it, both on `main` before
this run. The mark is "after": decided, and not in this plan. One commit
per task. Where the code, the spec and this plan disagree, the task stops
and says so.

## Principles

Serves 4 (every word costs attention: the card is the row uncut and never
the row plus, and one control per action) and 1 (the app reads nothing
new: the card draws facts the row already draws, and the one thing written
down is the person's own first click). Strains nothing. No Go file, no
argv list, no signal, no read and no field changes.

## Context (from discovery)

Checked against `main` at f486c47, and the two width changes were run
once over a copy of `presence/` by the plan's reviewer, so every test
named below as breaking was seen to break.

- **How a DOCUMENTS row opens today.** From its first icon only:
  `PanelModel.documentsSection` (`presence/Sources/PresenceUI/PanelModel.swift`,
  about line 635) builds five `ActionModel`s per row, `openDoc` first, with
  `icon: .gdocs` and `tip: Tips.openDoc`. The press leaves as
  `PanelActions.perform(.openDoc, id)`, which `Session.perform`
  (`presence/Sources/PresenceApp/SessionActions.swift`) turns into
  `Opener.openDocument(id)`, and which also calls `sawTheStrip()`, so any
  `.openDoc` retires the icon labels. A NOW row already opens on a click of
  the row through the same kind: `NowPress.open` and the `OpensDocument`
  modifier in `PanelView.swift` (about lines 143 and 368), a `contentShape`
  and an `onTapGesture` over the row's box, with the icons on top taking
  their own clicks. The DOCUMENTS row carries no `.opensDocument` today, and
  `DocumentRowModel` has no `opens`. `LinkLine.isDocumentID` is the test of
  an id Drive could have written; a NOW row with any other id opens nothing.
- **The tap and the long press.** `RowView` puts `.heldPress(titleHeld)`
  on the title line (`Components/MenuAnchor.swift`, `onLongPressGesture` at
  `Timing.menuHold`, 0.5 s, popping the system `NSMenu`). That file's own
  comment, about line 122, records that a long press swallows the click
  under it. So a tap put on the whole row and a long press kept on the
  title line are two views, and a click on the name of an unpinned row
  may open nothing. The tap and the hold have to sit on one view, the
  title line, with the row's own press under it for the sub-line and the
  gaps. No test sends a real click; Nail's hand check holds the gesture.
- **The five icons.** `Grid.actionColumn` is 136, five 24 pt icons and
  four 4 pt gaps (`RowTests.testTheDocRowActionColumnIs136`,
  `ThemeTests.testTheGridIsTheScreens` line 182). `StripLabel` in
  `Components/IconStrip.swift` is five words, Doc first
  (`CardTests.testTheStripLabelsAreDocNoteChatLinkPin`). The stuck card in
  NOW also carries a `.gdocs` icon (`CardTests`, about line 348); it keeps
  it, so `Icon.gdocs` (`Components/IconShapes.swift`) and
  `ActionKind.openDoc` stay. Tests that read a document row's `openDoc`
  action and break without it: `StateTests.testADocumentRowsFifthIconIsThePinInItsTwoStates`
  (about line 537), `StateTests.testAnIdDriveCouldNotHaveWrittenGreysItsTwoIcons`
  (433, 443), `StateTests.testAPinnedDocumentWithNoLinesDrawsThePinsOwnRow`
  (612), and `SheetPieces.tips()` in `Shots.swift` (about line 424), which
  unwraps e6's first row's `.openDoc` and feeds
  `ControlTests.testTheControlSheetRenders`,
  `ControlTests.testTheSheetDrawsEachTooltipUnderItsOwnIcon` and
  `RenderTests.testEverySheetWordIsAWordAPanelDraws`. With 108, the head of
  the name in `RowTests.testAPrefixTakesItsRoomAndTheFloorCountsTheName`
  (line 142) is 14, not 10.
- **The width.** `Grid.panelWidth` is 320 (`Theme/Grid.swift`).
  `presence/Sources/Presence/Panel.swift` writes 320 twice as a literal, in
  the `KeyPanel`'s `contentRect` (line 45) and in `size()` (line 119); it
  already imports `PresenceUI`. `MiddleTitle.narrowestColumn` is derived
  from the grid and stated as the literal 112 in `RowTests` (line 34).
  At 360 these tests break, and some lose the premise they state:
  `ThemeTests` line 174 (320); `RowTests.testAPrefixTakesItsRoomAndTheFloorCountsTheName`
  (`nowColumn` 228 to 268, `JobCard.titleColumn` 240 to 280; at line 109
  the name now fits whole with and without the verb, and at line 116 the
  document column now has room for the verb); `RowTests.testNoHeadUnderEightCharactersAtPanelWidth`
  (lines 56 to 58: at 180 the floor no longer drops " announcements");
  `RowTests.testASubLineWrapsToASecondLineRatherThanBeingCut` (line 194,
  272 to 312); the header test at `RowTests` line 516 (a second version
  now leaves the address 90.5 pt, not under half); `CardTests` line 210
  (`reasonColumn` 256 to 296) and 248 (`titleColumn(actions: 3)` 172 to
  212); `ControlTests.testAConfirmRowWaitsForItsAnswer` line 270 (the
  sign-out sentence no longer wraps, 44 not 56);
  `ControlTests.testATooltipAppearsAfterHalfASecondAndFlipsInsideThePanel`
  line 74 (an anchor at x 280 no longer reaches the right margin);
  `RenderTests` line 73's message and `width="320"` at lines 714, 715, 728
  and 729 of the index page; `CardTests` line 482 frames a row at a
  literal 320. Comments under `presence/Sources` that say 320:
  `PanelView.swift:3`, `PanelWords.swift:298`, `PanelHeaderModel.swift:135`
  (whose "4 pt to spare" and "98 pt to 49" are false at 360),
  `Components/JobCard.swift:85`, `Components/StuckCard.swift:73`,
  `Components/MiddleTitle.swift:38,46`, `PresenceCore/Titles.swift:4`; and
  136 or five icons: `TooltipLayer.swift:30`, `ActionButton.swift:5`,
  `MenuAnchor.swift:18,88`, `PanelModel.swift:48`, `StateTests.swift:271`.
  `docs/guide/presence.md` says 320 at lines 47 and 88. `CLAUDE.md` says
  nothing about the width or the icons.
- **The tooltip layer.** `Components/TooltipLayer.swift`: one layer over
  the panel draws one `TooltipBubble`, a single line, from the last
  `TooltipRequest` or the forced one, after `Timing.tooltipDelay`, 0.5 s,
  measured as a rest and not an entry (`Tooltip.shows`). `Tooltip.place`
  already slides a bubble inside the panel's margin and flips it above the
  anchor when there is no room under. `TooltipPlacementLayout` sizes its
  subview with `sizeThatFits(.unspecified)`, so a view with no width of its
  own is drawn on one line. `RowView` puts `.tooltip(titleTip)` on the title
  line alone. In a row's tree the middle column comes before the trailing
  icons, so an icon's request is later than the title's and wins the layer.
  A render forces one tooltip through `UIState.forcedTooltip`.
- **The words.** `Phrases.documentSub` (`PanelWords.swift`, about line
  365) writes the second line: `state(row)`, the verb's past with
  `clockInSentence(row.at)`, or "outcome unknown · <clock>", then the note
  path through `Sentences.shortPath` or `Sentences.noNoteYet`. `Words.time`
  writes a clock today, Yesterday, a weekday inside the horizon, then
  "d MMM"; `Words.clock` is "HH:mm". No function writes a moment with its
  year. 28 Oct 2026 is a Wednesday. `facts.note(for:)` gives the note's
  path; `facts.home` the home. `PanelModel.swift` is 808 lines and
  `Session.swift` 911, both over the 800 ceiling: new code goes into new
  files or `PanelWords.swift`, never into those two beyond a line each.
- **The store.** `PresenceApp/Preferences.swift` keeps eight keys, stated
  in `PreferencesTests.testOnlyTheEightKeysPersist` and counted in the
  comment of `SourceRulesTests.testOnlyThePreferencesPersist`; a ninth is
  a decision, which the 2026-10-28 entry now is. `stripSeen` is the shape
  to copy: `UIState.stripSeen` (default true in `UIState`, false in the
  store), `UIState.afterAnIconClick()`, written by `Session.sawTheStrip`
  (`SessionActions.swift`, about line 77).
- **Scenes and renders.** 18 scenes in
  `presence/Tests/PresenceUITests/Scenes.swift`; e1 is the six-row list
  with one failed reply, e6 is the first hint with the strip and a forced
  tooltip. `RenderTests.testEveryScreenRendersInBothAppearances` writes
  the PNGs and `index.html` under `PRESENCE_SHOTS`.
- **The design.** `docs/design/2026-10-28-presence-fifth-look.md` is the
  seats' answer and the lead's reading; the screens are
  `2026-10-28-presence-fifth-look.html`, scene 2 the card, scene 3 the
  four labels, scene 1 the width. Where that note says the card is "panel
  width less the row inset", this plan reads it as the two edges it names:
  the title's left edge and the panel's one right content edge.
  `docs/design/2026-10-17-presence-design-principles.md` holds the
  checklist the design reviewer runs after this run.

### What the lead changes on `main` before the run

Not tasks of this plan:

- `docs/v2/DECISIONS.md`: the 2026-10-28 entry "The fifth look" with
  Nail's words verbatim and its register row.
- `docs/v2/SPEC.md`: the paragraph "**The fifth look.**" after "**The
  fourth look.**".

The guide, `docs/guide/presence.md`, is this plan's, in Task 8.

### Out of scope

The firm's mark on the panel, in any place: it is its own later milestone
with its own decision entry and the trust guardian's signature, and
`docs/backlog/the-panel-does-not-show-the-logo.md` stays. A card on a NOW
row. Anything read from Drive for the card: owner, editors, counts. Note
contents beyond the path. A button, a link or any control inside the
card. A placeholder, a gap or a "no logo yet" line at the foot. The
stuck card's own Google Docs icon in NOW, which stays. A right-click
taking the card away: the decision asks it of the held press alone, and
the system's context menu gives no moment to act on; Nail's hand check
notes what it sees. `CLAUDE.md` is not edited: the six argv lists, the one
signal and the three things read are unchanged, and the file names neither
the width nor the icons.

## Development Approach

TDD, one commit per task, the failing test first. `make app-lint` and
`make app-test` green at every commit. Every size, count, sentence and
number a test states is a literal, never the constant it checks. Where a
width change takes a test's premise away, the task says what the test
holds instead; a literal is never loosened to make a run green. No test
sends a real click or a real hover: the presses are read through a
recording `PanelActions` as `CardTests.testTheRowOpensAndOnlyStopStops`
does, and the card's showing is a pure function a test feeds events to.
Nail's hand check after the run holds the gestures themselves.

Renders are made once, in Task 6, into the one folder the lead hands the
design reviewer. No task opens a PNG. `make app-test` writes renders into
a temporary folder as a side effect, and no task reads them.

Area, for the worker's brief: presence for Tasks 1 to 7, docs for Task 8.
No file under `go/` changes.

## Validation Commands

```bash
make app-lint
make app-test
make app
```

Once, in Task 6 only:

```bash
rm -rf "$HOME/gdoc-m32c-board/shots"
PRESENCE_SHOTS="$HOME/gdoc-m32c-board/shots" make app-test
```

## Progress Tracking

Eight tasks. A task is done when every box is ticked and its commit is on
the branch.

## Tasks

### Task 1: a click on a document row opens it, and the Google Docs icon goes

**Files:**
- Create: `presence/Sources/PresenceUI/DocumentPress.swift`
- Modify: `presence/Sources/PresenceUI/PanelModel.swift`
  (`DocumentRowModel`, `documentsSection`, `ActionKind`), `PanelView.swift`
  (`DocumentsList`), `Components/RowView.swift`, `Components/IconStrip.swift`,
  `Components/MiddleTitle.swift`, `Components/TooltipLayer.swift`,
  `Components/ActionButton.swift`, `Components/MenuAnchor.swift`
  (comments), `Theme/Grid.swift` (`actionColumn`),
  `presence/Sources/PresenceApp/SessionActions.swift`,
  `presence/Tests/PresenceUITests/RowTests.swift`, `CardTests.swift`,
  `StateTests.swift`, `ThemeTests.swift`, `Shots.swift`,
  `presence/Tests/PresenceAppTests/SessionTests.swift`

- [x] `DocumentRowModel` carries `opens: String?`: the row's own id where
      `LinkLine.isDocumentID` accepts it, nothing otherwise, the rule a
      NOW row already follows. Its actions are four: the note, the chat,
      the link and the pin, in that order, and `openDoc` is no longer among
      them. `StateTests.testADocumentRowOpensItselfAndCarriesFourIcons`
      replaces `testADocumentRowsFifthIconIsThePinInItsTwoStates`: a table
      over a Drive-shaped id and one Drive could not have written, reading
      `opens` and the four kinds `[.openNote, .openChat, .copyLink, .pinDoc]`,
      `.unpinDoc` on a pinned row with its filled tack
- [x] `ActionKind.openRow`, the row's own press, which `Session.perform`
      routes to `Opener.openDocument` exactly as `.openDoc` and which does
      not call `sawTheStrip()`: a click on a row is not a click on an icon,
      so the four labels stay until an icon is clicked.
      `SessionTests.testARowPressOpensTheDocumentAndLeavesTheStrip`
- [x] `DocumentPress.open(_ row: DocumentRowModel, actions:) -> (() -> Void)?`
      in its own file, the shape of `NowPress.open`: a press sending one
      `.openRow` with the row's id, nil where `opens` is nil. `RowView`
      gains `titleTap: (() -> Void)?`, applied to the title line together
      with `.heldPress(titleHeld)` on the same view, so a release before
      the hold is the tap and the hold is the menu; `DocumentsList` hands
      the same press as `titleTap` and, through `.opensDocument`, to the
      row's box for the sub-line and the gaps. The icons keep their own
      press over it. `CardTests.testADocumentRowOpensAndItsIconsKeepTheirOwnPress`
      through a recording `PanelActions`: the press sends one `.openRow`
      with the id and nothing else; a row with no `opens` builds no press
- [x] the three tests that read the first icon are restated:
      `testAnIdDriveCouldNotHaveWrittenGreysItsTwoIcons` asserts `opens`
      is nil and Copy link is off; `testAPinnedDocumentWithNoLinesDrawsThePinsOwnRow`
      asserts `opens` where it read the `openDoc` action.
      `SheetPieces.tips()` in `Shots.swift` takes another live icon of e6's
      first row, `.copyLink`, and the two sheet bands in `ControlTests` draw
      that icon where they drew `.gdocs`
- [x] `Grid.actionColumn` is 108: four 24 pt icons and three 4 pt gaps.
      `RowTests.testTheDocRowActionColumnIs136` becomes
      `testTheDocRowActionColumnIs108`, with `4 * 24 + 3 * 4` as its
      arithmetic; `ThemeTests.testTheGridIsTheScreens` says 108.
      `MiddleTitle.narrowestColumn`'s literal in `RowTests` becomes 140 for
      this commit (the panel is still 320 here; Task 2 moves it again), its
      message says 108, and the head count at `RowTests` line 142 states
      the new literal the 140 pt column gives
- [x] `StripLabel` is four words, Note, Chat, Link, Pin.
      `CardTests.testTheStripLabelsAreDocNoteChatLinkPin` becomes
      `testTheStripLabelsAreNoteChatLinkPin` with the count 4, and the
      centring test over the labels still passes with four
- [x] every comment under `presence/Sources` that says five icons, a fifth
      icon or 136 for the column says four and 108: `IconStrip.swift`,
      `RowView.swift`, `TooltipLayer.swift:30`, `ActionButton.swift:5`,
      `MenuAnchor.swift:18,88`, `PanelModel.swift:48`
- [x] `ActionKind.openDoc`, `Icon.gdocs`, `Tips.openDoc` and the stuck
      card's icon in NOW are untouched; the test at `CardTests` line 348
      still reads `.gdocs` first on the stuck card
- [x] `make app-lint`, `make app-test`

### Task 2: the panel is 360 points wide

**Files:**
- Modify: `presence/Sources/PresenceUI/Theme/Grid.swift`,
  `presence/Sources/Presence/Panel.swift`, the comments named below,
  `presence/Tests/PresenceUITests/RowTests.swift`, `CardTests.swift`,
  `ControlTests.swift`, `ThemeTests.swift`, `RenderTests.swift`,
  `presence/Tests/PresenceCoreTests/SourceRulesTests.swift`

- [x] `Grid.panelWidth` is 360, and its comment names
      `docs/design/2026-10-28-presence-fifth-look.html` beside the v3
      screens file. `RowTests.testThePanelIs360PointsWide` states 360 as a
      literal and reads a `.doc` row's drawn width back through its probe;
      `ThemeTests` line 174 says 360
- [x] `Panel.swift` reads `Grid.panelWidth` in both places and holds no
      width literal of its own.
      `SourceRulesTests.testThePanelWindowTakesItsWidthFromTheGrid` reads
      `Sources/Presence/Panel.swift` as text: `Grid.panelWidth` is in it,
      and neither `320` nor `360` is
- [x] the literals that only move: `MiddleTitle.narrowestColumn` 180 in
      `RowTests`, with the arithmetic rewritten as 360 less 8 twice, 12
      twice, 16, 8 twice and 108; `nowColumn` 268 and `JobCard.titleColumn`
      280; the sub-line's 312 at `RowTests` line 194; `CardTests` 296 at
      line 210 and 212 at line 248; `CardTests` line 482 framed at
      `Grid.panelWidth`; `RenderTests` line 73's message and the index
      page's four `width="320"` written from `Grid.panelWidth`
- [x] the premises that die at 360, each replaced and none loosened:
      `testAPrefixTakesItsRoomAndTheFloorCountsTheName` takes a longer NOW
      name of the test's own, found by measurement, so the verb still takes
      room in 268 and still leaves the name under the floor in 180, both
      stated as literals; `testNoHeadUnderEightCharactersAtPanelWidth` takes
      a new wide-tail title whose tail the counted rule admits and the 180
      pt floor refuses, found by measurement and stated; the header test at
      line 516 asserts that a second version still cuts the address
      (crowded narrower than wanted) and its message no longer claims
      under half; `testAConfirmRowWaitsForItsAnswer` states 44 for the
      sign-out sentence, which fits one line at 360, and holds the wrap
      with a longer consequence of the test's own that measures 56;
      `testATooltipAppearsAfterHalfASecondAndFlipsInsideThePanel` anchors
      at x 320 so the bubble still reaches the right margin
- [x] every comment under `presence/Sources` that says 320 for the panel
      says 360: `PanelView.swift:3`, `PanelWords.swift:298`,
      `Components/JobCard.swift:85`, `Components/StuckCard.swift:73`,
      `Components/MiddleTitle.swift:38,46`, `PresenceCore/Titles.swift:4`;
      and `PanelHeaderModel.swift:135` is rewritten in words that are true
      at 360, measured, not a number swapped
- [x] the test-file comments and messages that say 320 for the panel say
      360: `StateTests.swift:271`, `CardTests.swift:23,212`,
      `RowTests.swift:35,87,122,191,443`, `ControlTests.swift:159`
- [x] `make app-lint`, `make app-test`, `make app`

### Task 3: the card's words

**Files:**
- Create: `presence/Sources/PresenceUI/DetailCardModel.swift`
- Modify: `presence/Sources/PresenceUI/PanelWords.swift` (`Phrases`,
  `Sentences`), `Words.swift`, `UIState.swift`, `PanelModel.swift`
  (`DocumentRowModel.card`, one line in `documentsSection`),
  `presence/Tests/PresenceUITests/WordsTests.swift`, `StateTests.swift`

- [x] `Words.fullMoment(_:calendar:)` writes "EEE d MMM yyyy, HH:mm":
      `WordsTests.testTheFullMomentCarriesTheDayDateAndClock` states
      "Wed 28 Oct 2026, 12:08" as a literal
- [x] `DetailCardModel` has exactly four fields: `title`, the row's title
      or its id; `when`, the second line's own state words with
      `fullMoment` where the second line has a clock ("published Wed 28 Oct
      2026, 12:08", "reply failed Wed 28 Oct 2026, 11:40", "live stopped
      Wed 28 Oct 2026, 12:26", "outcome unknown · Wed 28 Oct 2026, 12:08")
      and the state word alone where it has none ("live", "hold",
      "publishing"), nil for a pinned-only row; `note`, the whole path with
      the home as `~` through `Sentences.shortPath`, or
      `Sentences.noNoteYet`; `hint`, `Sentences.clickOpens` "Click opens in
      Google Docs" while `ui.cardHintSeen` is false and the row opens, nil
      otherwise. `Phrases.documentCard(_ row:, ui:)` builds it, and
      `DocumentRowModel` carries it as `card`. A table test over the eight
      `DocumentState`s: `StateTests.testTheCardIsTheRowUncutAndNeverTheRowPlus`,
      which also asserts the model has those four fields and no fifth
- [x] `UIState.cardHintSeen`, default true in `UIState` as `stripSeen` is,
      read here and written in Task 4
- [x] `make app-lint`, `make app-test`

### Task 4: the ninth key

**Files:**
- Modify: `presence/Sources/PresenceUI/UIState.swift`,
  `presence/Sources/PresenceApp/Preferences.swift`, `Session.swift`
  (the launch state, about line 247), `SessionActions.swift`,
  `presence/Tests/PresenceAppTests/PreferencesTests.swift`,
  `SessionTests.swift`, `presence/Tests/PresenceCoreTests/SourceRulesTests.swift`

- [ ] `Preferences.cardHintSeen`, false until written, the shape of
      `stripSeen`. `PreferencesTests.testOnlyTheEightKeysPersist` becomes
      `testOnlyTheNineKeysPersist` with `cardHintSeen` in its list;
      `SourceRulesTests`' comment counts nine and names the 2026-10-28
      entry as the decision for the ninth
- [ ] the first `.openRow` sets it: `UIState.afterARowOpened()`, called by
      `Session.perform` on `.openRow` and written to the store, the shape
      of `sawTheStrip`. A `.openDoc` from a NOW row or the stuck card does
      not set it, which is the decision's "first click on any document
      row". `SessionTests.testTheFirstRowPressPutsTheHintAway` presses a
      row once and reads the flag in the store and in the state, and
      presses a NOW row to read that nothing changed
- [ ] the launch state reads it: where `Session` builds its first
      `UIState` from the store (`stripSeen: preferences.stripSeen`, about
      line 247) it passes `cardHintSeen: preferences.cardHintSeen` too, or
      the state's default of true means the hint never shows in the app.
      `testTheFirstRowPressPutsTheHintAway` asserts the flag is false
      before the press
- [ ] `make app-lint`, `make app-test`

### Task 5: the card under a rested pointer

**Files:**
- Create: `presence/Sources/PresenceUI/Components/DetailCard.swift`,
  `presence/Sources/PresenceUI/Components/CardRest.swift`
- Modify: `presence/Sources/PresenceUI/Components/TooltipLayer.swift`,
  `Components/RowView.swift`, `PanelView.swift` (`DocumentsList`,
  `RowMenu.held`), `UIState.swift` (`forcedCard`),
  `presence/Tests/PresenceUITests/ControlTests.swift`, `Scenes.swift`

- [ ] the plate. `DetailCard` draws the model on the tooltip's surface,
      edge and shadow, at `Grid.controlRadius`, 8 pt padding, the title in
      `Typography.rowTitle` wrapped whole, the other lines in
      `Typography.caption` in the secondary ink, the hint in the tertiary
      ink. It frames its own width, `DetailCard.width`, the panel's width
      less `Grid.unit`, `Grid.rowInset` and `Grid.quietInset`: 296 on the
      360 panel, stated as a literal in
      `ControlTests.testTheCardSpansFromTheTitleToTheValueColumn` and read
      back through its probe, named "card". Its left edge is the title's,
      `Grid.quietInset` from the panel's edge; its right edge is the column
      every value ends on. It takes no clicks, which the layer already
      refuses for everything it draws
- [ ] the layer draws it. `TooltipRequest` carries a content, the words it
      carries today or a `DetailCardModel`, and `tooltipLayer()` draws
      `TooltipBubble` for the one and `DetailCard` for the other, placed by
      `Tooltip.place` with `.leading` and the same flip above the anchor
      when there is no room under. Which request the layer draws is a pure
      function, `Tooltip.chosen(_ requests:)`, the forced one else the
      last; `ControlTests.testAnIconsTooltipWinsOverTheCard` states it with
      a title's card request before an icon's words request
- [ ] when it shows is a pure function. `CardRest` takes events, enter,
      rested, leave, heldPress, click, and answers whether the card is
      drawn: shown only after rested; leave ends it; heldPress and click end
      it until the next enter. `ControlTests.testTheCardShowsAfterTheRestAndGoesOnLeaveHeldPressOrClick`
      walks the five
- [ ] where its state lives and who feeds it. `RowView` owns
      `@State private var rest: CardRest`, beside its `pointerInside`, and
      hands a `Binding` to a `.detailCard(_:rest:forced:)` carrier on its
      middle column, the title and the sub-line together, with the same
      half-second rest as `.tooltip`. `RowView`'s own wrappers around
      `titleTap` and `titleHeld` send the click and the heldPress event to
      `rest` first and then call the press they wrap. The held press's
      wrapper, in `RowView` and not in `RowMenu.held`, opens the menu on
      the next main-queue turn, after the card's removal has been drawn,
      so the menu and the card are never up together;
      `CardTests.testTheHeldPressOpensTheSystemMenuWithTheRowsOwnWords`
      calls `menu.held` synchronously and stays as it is. A DOCUMENTS row
      hands its card and no `titleTip`
- [ ] renders can show it. `UIState.forcedCard: String?`, a document id,
      the shape of `forcedTooltip`; the carrier treats it as forced. Scene
      e1 forces the card on its second row, the first of the five under
      the pin, with `cardHintSeen: false` so the fourth line is in the
      picture, and its caption says what the card is and what each of its
      four lines is. The eighteen scenes stay eighteen
- [ ] `make app-lint`, `make app-test`

### Task 6: the renders, once

**Files:**
- Modify: `presence/Tests/PresenceUITests/Scenes.swift` (captions only, if
  the check below finds one stale)

- [ ] read every caption in `Scenes.swift` as text against Tasks 1 to 5,
      and fix any caption that says five icons, a fifth icon, five labels,
      320, or that the first icon opens the document. The eighteen scenes
      stay eighteen
- [ ] `rm -rf "$HOME/gdoc-m32c-board/shots"`, then
      `PRESENCE_SHOTS="$HOME/gdoc-m32c-board/shots" make app-test`, once
- [ ] check with `ls` and nothing else: each of the 18 scene ids has its
      `<id>-dark.png`, and `index.html` is there. Open no PNG. The design
      reviewer reads them after the run
- [ ] the commit message names the folder as an absolute path, which the
      lead hands to `gdoc-design-reviewer`

### Task 7: Verify acceptance criteria

**Files:**
- Modify: this plan

- [ ] every row of the Acceptance table at the foot of this plan names a
      test that exists and passes; a test a task named differently is
      renamed here, not in the code
- [ ] `git diff --name-only main...HEAD`: nothing under `go/`, `CLAUDE.md`
      and `docs/v2/` not in the list, nothing under `docs/backlog/`
      deleted or added
- [ ] `grep -rn "320" presence/Sources` prints nothing;
      `grep -rn "NSWorkspace\|NSPasteboard" presence/Sources` names
      `Opener.swift` and `Relauncher.swift` only;
      `grep -rn "popover(" presence/Sources` prints nothing;
      `wc -l presence/Sources/PresenceUI/PanelModel.swift` is under 830
      and `wc -l presence/Sources/PresenceApp/Session.swift` under 920:
      both are over the 800 ceiling already, and this plan adds a field,
      a case and their comments, nothing more
- [ ] `make app-lint`, `make app-test`, `make app`

### Task 8: [Final] Update documentation

**Files:**
- Modify: `docs/guide/presence.md`,
  `docs/design/2026-10-17-presence-design-principles.md`,
  `docs/design/2026-10-28-presence-fifth-look.md`; move this plan

- [ ] the guide, each place by line on `main`: 47, 360 points wide; 88 to
      90, the header sentence rewritten in words true at 360, not a
      number swapped; "### Documents", a click on the row opens the
      document in your browser the way a Now row's does, and an id Drive
      could not have written opens nothing; the icon table is four rows,
      the Google Docs row gone, and the sentence over it says four; 196,
      "one of the four icons"; 215 to 219, the greyed-icon list without
      "No Google Doc yet" and without "the first two are one case"; 227,
      four labels; 470, four icon labels and nine things the app keeps; a
      new paragraph on the card: what rests it, its three lines, the
      fourth that goes after your first click on a document row and never
      comes back, and that it takes no click. Nothing about the mark
- [ ] the principles file gains "Amendments, 2026-10-28, the fifth look":
      the row is the press and one control per action; the card is the
      row uncut, never the row plus, a new fact on it is a decision; the
      panel is 360; the firm's mark is decided for the foot and not drawn
      until its own milestone
- [ ] `docs/design/2026-10-28-presence-fifth-look.md` opens with one line
      after its date: decided 2026-10-28 in Nail's words, and this plan's
      path
- [ ] `CLAUDE.md` untouched
- [ ] move this plan to `docs/plans/completed/`
- [ ] `make app-lint`, `make app-test`

## Acceptance

Nail's answers of 2026-10-28, with the test that holds each. A row's test
is the one that fails if its answer is taken away.

| Nail's answer | The answer | The test that holds it |
|---|---|---|
| 1 yes: the card | three lines the row already draws, cut nowhere, and the hint until the first row click | `StateTests.testTheCardIsTheRowUncutAndNeverTheRowPlus`, `WordsTests.testTheFullMomentCarriesTheDayDateAndClock` |
| the card is a rest, not an entry, and goes on leave, held press or click | a pure function of five events | `ControlTests.testTheCardShowsAfterTheRestAndGoesOnLeaveHeldPressOrClick` |
| the card is a plate under the row, inside the panel | from the title's edge to the value column, placed by the one layer, an icon's words winning over it | `ControlTests.testTheCardSpansFromTheTitleToTheValueColumn`, `ControlTests.testAnIconsTooltipWinsOverTheCard` |
| the hint is taught once | the ninth key, written on the first row press and by nothing else | `PreferencesTests.testOnlyTheNineKeysPersist`, `SessionTests.testTheFirstRowPressPutsTheHintAway` |
| 2 a: the row opens, the icon goes | `opens` on the row, four icons, the press sends `.openRow` down the opener's path and leaves the strip | `StateTests.testADocumentRowOpensItselfAndCarriesFourIcons`, `CardTests.testADocumentRowOpensAndItsIconsKeepTheirOwnPress`, `SessionTests.testARowPressOpensTheDocumentAndLeavesTheStrip`, `RowTests.testTheDocRowActionColumnIs108`, `CardTests.testTheStripLabelsAreNoteChatLinkPin`; the gesture itself is Nail's hand check |
| 3 360: the width | the grid says 360 and the window reads the grid | `RowTests.testThePanelIs360PointsWide`, `SourceRulesTests.testThePanelWindowTakesItsWidthFromTheGrid` |
| 4 yes: the empty column stays | the mark on `ok: false` and blank otherwise, unchanged | `StateTests.testADocumentRowIsMarkedOnlyWhenItsLastRunFailed`, unedited |
| 5 b after: the mark at the foot, later | nothing in this branch draws or reads a picture | held only by Task 7's `git diff --name-only`: no file under `go/`, no new read under `presence/Sources`; no source rule pins a file read today |

## Post-Completion

These are the lead's, after the run and before the PR.

**The design reviewer, once.** Call `gdoc-design-reviewer` with the renders
folder as an absolute path, `/Users/nailkhusnullin/gdoc-m32c-board/shots`,
and the two scenes to look at first: e1, the card over the list, and e6,
the four labels. Its blocking findings are fixed on the branch before the
PR; its notes go to `docs/backlog/`.

**The release.** `make release BRANCH=gdoc-v2-m32c-the-fifth-look` from the
checkout on `main`, and `GO=v2.25.0` only after Nail's word for that
version.

**Nail's hand check**, written for him in what he sees: the panel opens
wider; rest the mouse on a document's name and after half a second a card
shows its whole name, the date and time, and the note's path, and goes when
the mouse leaves; click the name of a document that is not pinned, and one
tab opens, not none and not two; click the grey second line, and it opens
too; hold the mouse down on the name and the menu opens with no card under
it; right-click and say whether the card stayed; the four icons do what
they did and their four labels stay until an icon is clicked; after the
first click on a row the card's last line is gone and stays gone after the
app is reopened.
