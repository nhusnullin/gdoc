# gdoc v2 Milestone 32d: the stuck card's open icon. Plan

Tasks: 5
Review: ui

2026-10-29. After v2.25.0 the design reviewer named one should-fix item in
the renders: the stuck card in NOW still carries an icon that opens the
document, while a click on the card has opened it since the fourth look.
Nail answered: "Yes, drop the open icon from the stuck-run card in the next
release; the row already opens the document." The decision is
`docs/v2/DECISIONS.md`, 2026-10-29, "The stuck card loses its open icon",
and the spec paragraph is changed with it, both on `main` before this run.
The reviewer's second should-fix item, a render in which the forced card's
row is not shaded, is in this plan too, because it is the lead's how and
costs one line. One commit per task. Where the code, the spec and this plan
disagree, the task stops and says so.

## Principles

Serves 4 of `PRINCIPLES.md` (one control per action: the card is the
press, and an icon that does what the card does is a second control) and
principle 5 of `docs/design/2026-10-17-presence-design-principles.md`,
facts and not promises (a still states what the app shows: a card is only
ever up on a hovered row, so a still of a forced card draws the row
hovered). Task 2 changes one boolean in production code that only a still
reaches: `UIState.swift`, about line 47, says nothing in the app ever sets
`forcedCard`. Strains nothing. No Go file, no argv list, no signal, no read
and no field changes.

## Context (from discovery)

Checked against `main` at 03c65d6.

- **Where the icon is built.** `PanelModel.stuckActions`
  (`presence/Sources/PresenceUI/PanelModel.swift`, about line 570) appends
  `openDoc(doc)` first when the write names a document, then `openNote`,
  `openChat`, `copyLink` and the dismiss. The private builder
  `openDoc(_:forced:)` (about line 724) is its only caller since the fifth
  look took the icon off the DOCUMENTS rows; it draws `Icon.gdocs` with
  `Tips.openDoc` ("Open in Google Docs", `PanelWords.swift:18`) and
  `Tips.noDoc` ("No Google Doc yet", line 19). `Tips.openDoc` stays: it is
  also `NowPress.words` (`PanelView.swift:149`), the tooltip a NOW row's and
  a stuck card's title shows, pinned by
  `StateTests.testEveryNowRowThatNamesADocumentOpensIt` (about line 909)
  and `CardTests.testTheRowOpensAndOnlyStopStops` (about line 393). Nothing
  reads `Tips.noDoc` once the builder is gone, and the `noLink` comment at
  `PanelWords.swift:31` to 33 refers to it. The
  machine's card (about line 560) builds Sign in and opens nothing; the
  write that created nothing builds the chat and the cross. None of those
  change.
- **The press stays.** `ActionKind.openDoc` (`PanelModel.swift:30`) is the
  kind a NOW row's and a stuck card's click leaves as, through
  `NowPress.open` in `PanelView.swift` and `Session.perform(.openDoc, _)`
  in `SessionActions.swift`, which opens and teaches nothing. The kind,
  the press and `SessionTests.testTheFirstRowPressPutsTheHintAway` stay as
  they are.
- **Who else names the glyph.** `Icon.gdocs` is drawn in
  `Components/IconShapes.swift` (about line 80) and named in
  `IconTests.testTheIconSheetRenders` (the closed list at line 32, "the
  thirteen the panel draws", and the ink-weight pair at lines 174 to 176),
  `RowTests` (two sample trailing buttons at about lines 297 and 319),
  and `CardTests` line 348, which reads e3's stuck card as
  `[.gdocs, .md, .branch, .link, .xmark]`. The ink-weight pair that dims
  `.gdocs` is in `IconTests.testAnOffIconIsDimmedWithItsOwnTooltip` (about
  lines 159 to 181), with the literals "Open in Google Docs" and "No Google
  Doc yet" at 159 to 167. `CardTests.testACardCutsATitleByTheRowsOwnRule`
  (about line 92) measures the stuck card's title column at `actions: 5`,
  which is e3's card today and is four after this; at four the column
  grows from 156 to 184 and "Export stopped" still splits. The only
  `switch` over `Icon` is `IconShapes.parts`; `Icon.allCases` is read only
  in `IconTests` (the closed list, two loops, the sheet), and the loops
  adapt by themselves. The one `|openDoc` is the builder's own line 728.
- **Comments that count or name the icon.** `IconShapes.swift:3` "Eleven of
  the thirteen" and line 23 "The thirteen icons the panel draws";
  `IconTests.swift` lines 3, 15 and 230 "thirteen" and line 5 "five of them
  sit side by side"; `Components/StuckCard.swift:100` "as little as 112 pt
  under five of them" and line 190 "With five live icons"; `PanelView.swift`
  139 to 141 "the kind a DOCUMENTS row's first icon sends" and 147 to 148
  "which are the Google Docs icon's"; `Components/RowView.swift:117` to 118
  "in the Google Docs icon's own words"; and the assertion messages at
  `StateTests.swift:910` and `CardTests.swift:393`.
- **The forced card's row.** `RowView.isFilled` is
  `hovering || pointerInside` (`Components/RowView.swift:308`), and the
  forced card comes in as `forcedCard: Bool` (line 137) from
  `PanelView.swift:250`, `model.forcedCard == row.doc`. Scene e1 forces the
  card on its second row and that row renders unshaded, so a reader cannot
  tell which row the card belongs to. The e1 caption (`Scenes.swift`, about
  line 366) says "The plate over the second row is the card"; the plate is
  under the second row and over the two below it. No test passes
  `hovering: true` today and `RowTests` measures no background, so the
  test is new in kind: the helper is `pixel(_:scheme:side:x:y:)` in
  `Tests/PresenceUITests/Shots.swift:164`, which renders a view in a square
  at 2x and reads one point. A `RowView` on its own draws no card, because
  the card goes up through the panel's `DetailCardCarrier`, so the sample
  point is never under a card. The hover colour is `palette.hover`,
  `0x2C2C2E` in dark and `0xECECEF` in light (`ThemeTests` lines 25 and
  91). `isFilled` is internal and reachable through `@testable`.
- **The renders.** `RenderTests` writes the PNGs under `PRESENCE_SHOTS`;
  the index page carries the captions. 18 dark scenes.
- **The documents.** `docs/guide/presence.md` lines 148 to 149 still say a
  NOW press "is the same press as a Documents row's first icon", split
  over the two lines, which the fifth look already made untrue; no bullet
  of the NOW section (lines 119 to 145) lists the stuck card's icons, so
  there is nothing there to change. `go/boundary/guide_test.go`,
  `TestThePresencePageWalksTheThirdLooksPanel`, requires strings from that
  page, none of them about an open icon, and one of them "Click opens in
  Google Docs" (line 224), which stays.
  `docs/design/2026-10-17-presence-design-principles.md`
  has an amendment "A card's title may take two lines" that counts "five
  live icons" on a stuck card; it stays as the record of its day and a
  2026-10-29 amendment says what changed.

### What the lead changes on `main` before the run

Done: the decision entry and its register row, and the spec paragraph
"The stuck card's open icon". Nothing else.

### Out of scope

- The firm's mark at the foot: its own milestone with the guardian's
  signature, after this one.
- Moving the card so it hides no rows: a real overlay hides rows too.
- The e6 Chat icon dimmed by a scene with no Claude Desktop, the e6 tooltip
  over row 3's last letter, the hint line's grey, the r3 caption: notes,
  not fixes, from the same review.
- CLAUDE.md, anything under `go/`.

## Development Approach

- TDD: the failing test first, in the file the Context names, then the
  smallest change that passes it.
- One commit per task, message in the house form, no attribution lines.
- Nothing is loosened: a test that read five icons now reads four by name,
  not by count.
- Plain English in every comment and caption, no em dashes.

## Validation Commands

```
make app-lint
make app-test
make app
```

Task 3 only:

```
PRESENCE_SHOTS="$HOME/gdoc-m32d-board/shots" make app-test
```

## Progress Tracking

Tick each box as it is done. A task ends with its commit.

## Tasks

### Task 1: the stuck card carries no open icon

**Files**

- `presence/Sources/PresenceUI/PanelModel.swift`
- `presence/Sources/PresenceUI/PanelWords.swift`
- `presence/Sources/PresenceUI/PanelView.swift`
- `presence/Sources/PresenceUI/Components/IconShapes.swift`
- `presence/Sources/PresenceUI/Components/StuckCard.swift`
- `presence/Sources/PresenceUI/Components/RowView.swift`
- `presence/Tests/PresenceUITests/CardTests.swift`
- `presence/Tests/PresenceUITests/IconTests.swift`
- `presence/Tests/PresenceUITests/RowTests.swift`
- `presence/Tests/PresenceUITests/StateTests.swift`

**Steps**

- [x] Write `CardTests.testTheStuckCardCarriesNoOpenIcon`: e3's stuck
      card's actions are `[.md, .branch, .link, .xmark]` by icon and
      `[.openNote, .openChat, .copyLink, .dismiss]` by kind; its `opens`
      still names the document; no NOW item of any scene carries an
      `ActionModel` of kind `.openDoc`; e9's machine card still carries
      `[.signin]`. It fails first on the first icon.
- [x] Change line 348's expectation in the test beside it to the four, and
      its comment from "the four of a row and the cross" to "the three the
      row shares and the cross". Change `testACardCutsATitleByTheRowsOwnRule`
      from `actions: 5` to `actions: 4` in its three places, with a comment
      that four is the card's count since 2026-10-29.
- [x] Drop `openDoc(doc)` from `stuckActions` and delete the private
      `openDoc(_:forced:)` builder. `Tips.openDoc` stays: it is the NOW
      title's words. Delete `Tips.noDoc` and reword the `noLink` comment
      above `Tips.noLink` so it no longer refers to it. The header comment
      of `stuckActions` names the new test and the decision date.
- [x] Delete `Icon.gdocs` and its parts in `IconShapes.swift`. In
      `IconTests.testTheIconSheetRenders` the list loses `.gdocs` and the
      words say "the twelve the panel draws". In
      `IconTests.testAnOffIconIsDimmedWithItsOwnTooltip` the pair becomes
      `ActionButton(.link, tip: Tips.copyLink)` and
      `ActionButton(.link, tip: Tips.copyLink, state: .off(Tips.noLink))`,
      and the two literals at lines 159 to 167 become `Tips.copyLink` and
      `Tips.noLink`. In `RowTests` the two sample trailing buttons become
      `ActionButton(.link, tip: Tips.copyLink)`.
- [x] Reword the comments the Context lists so none counts thirteen icons,
      five on a card, or names the Google Docs icon: `IconShapes.swift:3`
      "Ten of the twelve" and line 23 "The twelve icons the panel draws";
      `IconTests.swift` lines 3, 15 and 230 "twelve" and line 5 "four of
      them sit side by side"; `StuckCard.swift:100` and 190 say four icons
      and the width `titleColumn(actions: 4)` gives, which the task
      computes from the formula under it and states as the number;
      `PanelView.swift:139` to 141 and 147 to 148 and `RowView.swift:117`
      to 118 say the words are the NOW title's own, which a DOCUMENTS row's
      icon once shared; the assertion messages at `StateTests.swift:910`
      and `CardTests.swift:393` say the same.
- [x] `grep -rn "gdocs\|noDoc" presence/Sources presence/Tests` prints
      nothing. `grep -rn "|openDoc" presence/Tests` prints nothing.
      `grep -rn "thirteen\|five of them\|five live" presence/Sources
      presence/Tests` prints nothing.
- [x] `make app-lint`, `make app-test` green.
- [x] Commit: `feat: the stuck card carries no open icon`.

### Task 2: a still of a forced card draws its row hovered

**Files**

- `presence/Sources/PresenceUI/Components/RowView.swift`
- `presence/Tests/PresenceUITests/RowTests.swift`
- `presence/Tests/PresenceUITests/Scenes.swift`

**Steps**

- [x] Write `RowTests.testAForcedCardRowDrawsTheHoverFill`, with the
      `pixel(_:scheme:side:x:y:)` helper from `Shots.swift`: a document row
      from `SheetPieces.documentRows()[0]` built with `forcedCard: true`
      and `hovering: false`, rendered at `side: Grid.panelWidth` and read
      at `x: 6, y: side / 2` (inside the 12 pt row inset, past the 6 pt
      corner), is `palette.hover`, `0x2C2C2E` in dark and `0xECECEF` in
      light, the same point a row built with `hovering: true` reads; a
      row with neither reads alpha 0 there. Beside the pixels,
      `RowView(..., forcedCard: true).isFilled` is true through
      `@testable`. Both schemes. It fails first on the forced row.
- [x] `isFilled` becomes `hovering || pointerInside || forcedCard`, with
      the comment saying why: a card is only ever up on a hovered row, so a
      still that forces the card states the hover with it.
- [x] The e1 caption in `Scenes.swift`, about line 366: the sentence that
      opens "The plate over the second row is the card, which a pointer
      resting half a second on a document's name brings up and which goes
      when it leaves, is held down or is clicked." becomes "The plate under
      the second row, over the two below it, is the card, which a pointer
      resting half a second on a document's name brings up and which goes
      when it leaves, is held down or is clicked. The second row is shaded
      because the card is only ever up on a hovered row." The sentences
      after it stay.
- [x] `make app-lint`, `make app-test` green.
- [x] Commit: `fix: a still of a forced card draws its row hovered`.

### Task 3: the renders, once

**Files**

- none changed; the PNGs and the index page land under
  `~/gdoc-m32d-board/shots`

**Steps**

- [x] `PRESENCE_SHOTS="$HOME/gdoc-m32d-board/shots" make app-test` green.
- [x] `ls ~/gdoc-m32d-board/shots` lists `e1-dark.png`, `e3-dark.png` and
      `index.html`.
- [x] No commit: the renders are not in git.

### Task 4: Verify acceptance criteria

**Files**

- none

**Steps**

- [x] Every test named in the Acceptance table exists under
      `presence/Tests` with that name.
- [x] `git diff --name-only main...HEAD` lists only this plan,
      `presence/Sources` and `presence/Tests`. Nothing under `go/`,
      `CLAUDE.md` or `docs/v2/`.
- [x] `grep -rn "gdocs" presence/Sources` prints nothing.
- [x] `make app-lint`, `make app-test`, `make app` green.
- [x] No commit unless something had to change, and then it says what.

### Task 5: [Final] Update documentation

**Files**

- `docs/guide/presence.md`
- `docs/design/2026-10-17-presence-design-principles.md`
- `docs/plans/2026-10-29-gdoc-v2-m32d-the-stuck-cards-open-icon.md`

**Steps**

- [ ] In the guide, lines 148 to 149, the sentence "It is the same press
      as a Documents row's first icon" becomes "It is the same press as a
      click on a Documents row". No bullet of the NOW section lists the
      stuck card's icons, so nothing is added there. The line "Click opens
      in Google Docs" elsewhere on the page stays: a Go test requires it.
      `grep -c "same press as a click on a Documents row"
      docs/guide/presence.md` prints 1 and `grep -n "first icon"
      docs/guide/presence.md` prints nothing.
      `cd go && go test ./boundary/ -run TestThePresencePageWalksTheThirdLooksPanel`
      prints `ok`, not "no tests to run".
- [ ] In the principles file, a new heading "## Amendments, 2026-10-29,
      the stuck card's open icon" with two bullets: the stuck card's open
      icon went, one control per action, naming
      `CardTests.testTheStuckCardCarriesNoOpenIcon`; and a still of a
      forced card draws its row hovered, naming
      `RowTests.testAForcedCardRowDrawsTheHoverFill`. The "five live icons"
      line above stays as the record of its day.
- [ ] Move this plan to `docs/plans/completed/`.
- [ ] Commit: `docs: the stuck card's open icon`.

## Acceptance

| Claim | Test |
|---|---|
| e3's stuck card carries the note, the chat, the link and the cross, and no open icon | `CardTests.testTheStuckCardCarriesNoOpenIcon` |
| The stuck card's title column is measured at the card's real count | `CardTests.testACardCutsATitleByTheRowsOwnRule` |
| The dismiss stays the last icon | `CardTests.testTheStuckCardDismissIsTheLastIcon` |
| The icon set is the twelve the panel draws | `IconTests.testTheIconSheetRenders` |
| A NOW press still teaches nothing about the DOCUMENTS hint | `SessionTests.testTheFirstRowPressPutsTheHintAway` |
| A still of a forced card draws its row hovered | `RowTests.testAForcedCardRowDrawsTheHoverFill` |
| A card about a write that created nothing keeps the chat and the cross | `CardTests.testACardCarriesNoIconThatWouldDoNothing` |
| The guide says a NOW press is a click on a Documents row, and names no first icon | the two greps in Task 5, and `TestThePresencePageWalksTheThirdLooksPanel` green |

## Post-Completion

- The design reviewer reads `/Users/nailkhusnullin/gdoc-m32d-board/shots`:
  e3's card with four icons, e1's second row shaded under its card.
- `make release BRANCH=gdoc-v2-m32d-the-stuck-cards-open-icon` from the
  checkout on `main`; `GO=v2.26.0` only on Nail's word.
- Nail's hand check, in what he sees: a stopped run's card in NOW shows
  four small icons, the note, the chat, the link and the cross, and a click
  on the card's words opens the document in one tab.
