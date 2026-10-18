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

- [ ] the header carries the account's state: the status glyph (check when
      the token works, exclamation when it does not, a ring while nobody has
      asked), the address, and the one action, Sign out, as the icon the
      HEALTH row carried; with no token the header shows Sign in where the
      icon was; a test states the three pieces and the height, which stays
      36 pt
- [ ] "Sign out?" asks in place of the header line: the question, its
      consequence line under it, Sign out and Keep where the actions were;
      the question waits for its answer as PR 91 made it
- [ ] the signed-in banner, the sign-in countdown and the Not-you line move
      with the account: they are drawn under the header, not under a HEALTH
      row
- [ ] HEALTH loses the Account row; the Account changed quiet line goes with
      it under the header
- [ ] the header carries the gdoc version after the name, in the caption
      grey ("gdoc" then "v2.16.0"), and the update action where a newer
      release is listed: the same icon and the same ring-while-it-runs the
      gdoc row had, now beside the version; with nothing newer, no icon;
      HEALTH loses the gdoc row too; a test states the header's pieces with
      and without an update
- [ ] `make app-lint`, `make app-test`

### Task 3: the group is SETTINGS

**Files:**
- Modify: `presence/Sources/PresenceUI/Words.swift` or `PanelWords.swift`,
  `PanelHealth.swift` and its callers, `docs/guide/presence.md`, the scene
  captions

- [ ] the bottom group's label is SETTINGS: Hub (Task 7), Launch at login,
      Quit, and the Network and Claude Desktop rows when a fact carries
      them; the Thread mark row goes, and the mark stays where it was, in the
      header's own mark; the type and file names follow the word where a
      name said health, and a test states the label and the absent row
- [ ] `PanelModel`'s "pinned first" rule for the three screens stays for the
      two that are still about the group (no token, a sign-in in flight) and
      drops the question, which now lives in the header
- [ ] `docs/guide/presence.md` names the group and what moved up
- [ ] `make app-lint`, `make app-test`

### Task 4: the middle scrolls, the edges stay

**Files:**
- Modify: `presence/Sources/PresenceUI/PanelView.swift`,
  `Components/ScrollBox.swift`, `presence/Tests/PresenceUITests/ScrollTests.swift`,
  `RenderTests.swift`

- [ ] the panel is header, NOW, the middle, SETTINGS, in that order; the
      header, NOW and SETTINGS never scroll; the middle (DOCUMENTS, TODAY, Yesterday)
      is the one `ScrollBox`, and its height is what is left of 600 pt after
      the three fixed parts; a test measures the four heights on scene e3
- [ ] NOW, when it alone is taller than the panel allows, is capped at half
      the body and scrolls inside itself, so SETTINGS is never pushed off; a test
      on a fixture with four hold cards
- [ ] the scroll thumb is an overlay 3 pt inside the box's right edge, drawn
      while the content moves and for one second after, and never otherwise;
      the fade stays; a test states the inset, the width and the second
- [ ] `make app-lint`, `make app-test`

### Task 5: DOCUMENTS by what was just made, then by use

**Files:**
- Modify: `presence/Sources/PresenceCore/Documents.swift`,
  `presence/Tests/PresenceCoreTests/DocumentsTests.swift`, the scene
  fixtures and `RenderTests.swift`

- [ ] the order: first every document with an action today, newest first;
      then the rest of the fourteen days by the count of finished actions in
      those days, most first, ties newest first; a table test over a
      fixture with three documents touched today and four older ones with
      different counts
- [ ] the row's sub-line keeps saying the last action and its time; a
      document listed by use shows nothing new, because a count is a
      judgement the panel does not print
- [ ] "Show N more" and the horizon are unchanged
- [ ] `make app-lint`, `make app-test`

### Task 6: pinned and hidden documents

**Files:**
- Modify: `presence/Sources/PresenceApp/Preferences.swift`,
  `presence/Sources/PresenceCore/Documents.swift`,
  `presence/Sources/PresenceUI/PanelModel.swift`, `IconStrip.swift`,
  `IconShapes.swift`, `SessionActions.swift`, tests beside each

- [ ] two more preferences keys, `pinned` and `hidden`: lists of document
      ids, each with the moment and the last title; never pruned by age; the
      source scan's count becomes eight and its sentence says why
- [ ] the four icons stay four. A DOCUMENTS row has a menu, on a secondary
      click and on a press held half a second: "Pin" or "Unpin", and "Hide
      from the list". A pinned row shows a small pin glyph in its status
      column before the state glyph; a test over the menu's words in both
      states and the glyph
- [ ] a hidden document leaves DOCUMENTS and its actions stop counting for
      the order; TODAY and Yesterday still show its runs, because the log is
      the log; the next finished action gdoc does on that document brings it
      back into the list, which is the one way back, and the guide says so;
      a test over hide, the count, and the return
- [ ] pinned documents are listed first in DOCUMENTS, in the order they were
      pinned, newest pin first, and never fall out of the list when their
      fourteen days pass; a pinned document that the record has no line for
      any more still draws its last known title and its open action from the
      pin's own stored title and id, which the pin keeps for that reason; a
      test over a fixture whose pinned document is sixty days old
- [ ] the header count counts them: "14 days · N" counts pinned rows too and
      hidden rows not, and the index caption says so
- [ ] `make app-lint`, `make app-test`

### Task 7: the hub is chosen in the panel

**Files:**
- Modify: `presence/Sources/PresenceUI/PanelHealth.swift` (the SETTINGS
  rows), `presence/Sources/PresenceApp/SessionActions.swift`,
  `Preferences.swift`, `docs/guide/presence.md`, tests beside each

- [ ] a Hub row in SETTINGS: the chosen folder as `~/…`, or "Choose…" when
      none; pressing it opens the system folder panel (`NSOpenPanel`, folders
      only, the app's own window and no program), and the choice goes into
      the `hubPath` key the guide already names; a test over the row's two
      states and the stored path through an injected chooser
- [ ] the md icon's tooltip with no hub names the row ("Choose the hub in
      SETTINGS"), so a grey icon says why
- [ ] the guide's `defaults write` paragraph becomes the row, and keeps the
      command as the way from a terminal
- [ ] `make app-lint`, `make app-test`

### Task 8: the session a run came from

**Files:**
- Modify: `go/internal/presence/event.go`, `go/cmd/gdoc/main.go`,
  `go/internal/presence/doc.go`, `go/internal/presence/presence_test.go`,
  `go/cmd/gdoc/*_test.go`, `presence/Sources/PresenceCore/Event.swift`,
  `Documents.swift`, `presence/Sources/PresenceApp/Opener.swift`,
  `presence/Tests/...`

- [ ] the record gains `session`: the value of `CLAUDE_CODE_SESSION_ID`
      when `via` is `code`, read in `viaOfEnv` beside `CLAUDECODE`, written
      on the start line and nowhere else; absent under terminal and chat; the
      allowlist test names it; a test shows the field under Claude Code and
      its absence elsewhere, and `TestTheFlagChangesNoByteOfTheObject` still
      holds
- [ ] the Swift `Event` decodes `session`, and a document remembers the
      session of its latest `via: code` run
- [ ] Open chat on a document whose latest run came from Claude Code with a
      session opens `claude://code/continue?session=<id>&source=gdoc`; a
      document with a chat run keeps the activation it has; a document with
      neither keeps the activation; the URL is built by one function with a
      test over the three cases, and the opener's allowlist of schemes gains
      this one path and no other
- [ ] the measurement: Nail opens one such link while watching the window,
      and the result goes to `docs/v2/MEASURED.md` with the date; if the Code
      tab does not take a session id, the link is not shipped and this task's
      commit says so, leaving the activation as it is and the field in the
      record for the day it does
- [ ] `make vet`, `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`,
      `make app-lint`, `make app-test`

### Task 9: the words

**Files:**
- Modify: `docs/guide/presence.md`, `CLAUDE.md` (the presence invariant's
  field list, if it names fields), `go/internal/presence/doc.go`,
  `docs/design/2026-10-17-presence-screens-v3.html` (scene captions only)

- [ ] the guide walks the panel in its new order: header with the account
      with its version and update, and Sign out, NOW, DOCUMENTS with pins,
      hides and the order rule, TODAY, Yesterday, SETTINGS with the Hub row
      and no Thread mark; and names the session link and when it is used
- [ ] the scene captions in the screens file that say HEALTH or place the
      account there say the new place
- [ ] CLAUDE.md stays under 300 lines
- [ ] `make test`, `make app-lint`

### Task 10: the reviewer's pass

**Files:**
- Modify: whatever the pass finds under `presence/`

- [ ] render every scene (`PRESENCE_SHOTS=... make app-test`) and run the
      checklist of `docs/design/2026-10-17-presence-design-principles.md`
      over every scene as a reviewer who drew nothing (a subagent that reads
      images), against the mockup where it still applies and against this
      plan's sentences where it does not; every blocking or should-fix
      finding is fixed here and listed in the commit message
- [ ] `make app-lint`, `make app-test`, `make app`

### Task 11: the acceptance rows and the close

**Files:**
- Modify: this plan, `docs/v2/DECISIONS.md` only if a task found the
  decision wrong

- [ ] an acceptance table at the foot of this plan: each of Nail's ten
      sentences with the test that holds its answer
- [ ] the render index's "Still your call" list drops what this plan
      decided and keeps the rest
- [ ] move this plan to `docs/plans/completed/`
- [ ] `make vet`, `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`,
      `make app-lint`, `make app-test`, `make app`
