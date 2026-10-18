# gdoc v2 Milestone 27: the third look. Plan

2026-10-18, evening. Nail opened v2.18.0 and sent a screenshot with three
scroll thumbs on one panel: the middle box, DOCUMENTS inside it and TODAY
inside it. His words, in order: double scrolling is awful; show no more than
five documents and no activity log for today and yesterday at all; why show
the hub, it differs by document and the notes live in different folders; he
did not see how to pin a document; the use statistics are not needed; show
the last five plus the pinned ones, no duplicates, no scroll at all, let the
panel be as long as it comes out; check for a new version periodically, show
it beside the current one, and give the update glyph more than one state so
he can press it to check and install even before the timer has asked. The
decision is `docs/v2/DECISIONS.md`, 2026-10-18, "The third look"; the spec
paragraph is changed with it. One commit per task. Where the code, the spec
and this plan disagree, the task stops and says so.

## Principles

Serves 4 (the panel is what the person reaches for, nothing he has to
operate) and 1 (nothing drawn that the moment does not need). Strains
nothing new: the one new read is a Spotlight query the app makes, named in
the decision.

## Context (from discovery)

- `presence/Sources/PresenceUI/PanelView.swift`: header, NOW (capped at half
  the body, scrolling inside itself), the middle `ScrollBox` (DOCUMENTS,
  TODAY, Yesterday, each with its own inner scroll and Show more), SETTINGS.
  `Grid.panelMaxHeight = 600`; `Components/ScrollBox.swift` draws the thumb
  and fade. The screenshot shows three thumbs at once.
- `presence/Sources/PresenceCore/Documents.swift` (M26): pinned first, then
  today's newest first, then fourteen days by the count of finished actions;
  the newest five until a person opens the rest; `Days.swift` folds TODAY and
  Yesterday. `PanelModel.swift` draws the three middle sections.
- Pins and hides (M26 Task 6): `Preferences` keys `pinned` and `hidden`; a
  DOCUMENTS row's menu on secondary click and a held press offers Pin/Unpin
  and Hide; a pinned row shows a small pin glyph in its status column. Nail
  did not find the menu.
- `presence/Sources/PresenceCore/HubIndex.swift`: the map from document id
  to note path, built by reading the `gdoc:` front-matter block of every note
  under the one hub folder in the `hubPath` preference; `NoteFault.noHub`,
  `.notPaired`. M26 Task 7 added the Hub row in SETTINGS that chooses that
  folder.
- Measured 2026-10-18 20:5x on Nail's machine: `mdfind -onlyin ~ '<doc id>'`
  returns the note whose front matter names the id, and one other file that
  mentions the id in its body (a document map), in 0.25 s; the
  `kMDItemTextContent == "*<id>*"` form the same in 1.5 s; 727 Markdown files
  are indexed under the vault. So the index exists and the front-matter check
  that already exists tells the note from a file that merely mentions the id.
- The header (M26 Task 2): "gdoc" then the version in the caption grey, and
  the update action only when `help --json` listed a newer release; the
  `HealthPoller` asks `help --json` on open. `gdoc help` asks GitHub once in
  24 hours from the stamp; `gdoc update` asks GitHub every run and installs
  the newest stable, or says it is current. The app relaunches itself after
  the bundle swap (M24).
- `docs/design/2026-10-17-presence-design-principles.md` holds the checklist
  the reviewer's pass runs (Task 8).

## Development Approach

TDD, one commit per task, the failing test first; `make app-lint`,
`make app-test`, `make app` green at every commit. Every size, count, colour
and word a test states is a literal. Renders go under `PRESENCE_SHOTS`,
outside the repository. No test runs a real Spotlight query or a real child:
the query goes behind an injected finder, and the runner's recorder stands
in for gdoc.

## Validation Commands

```bash
make app-lint
make app-test
make app
env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test
PRESENCE_SHOTS=/tmp/presence-shots make app-test
```

## Progress Tracking

Nine tasks. A task is done when every box is ticked and its commit is on the
branch.

## Tasks

### Task 1: nothing scrolls

**Files:**
- Modify: `presence/Sources/PresenceUI/PanelView.swift`, `Theme/Grid.swift`,
  `PanelModel.swift`; Delete: `Components/ScrollBox.swift` and its tests;
  Modify: `presence/Tests/PresenceUITests/RenderTests.swift`,
  `ScrollTests.swift` (removed or reduced to the one assertion that nothing
  scrolls), `Shots.swift`, `Scenes.swift`

- [ ] the panel is as tall as its content: no cap, no `ScrollBox`, no inner
      scroll in NOW or any list, no Show more; `Grid.panelMaxHeight` goes; a
      test renders scene e3 and asserts the panel's height equals the sum of
      its parts and that no view in the tree is a scroll view
- [ ] the render index draws one image per scene and appearance, since the
      capped pair no longer differs from the whole; the caption at the top
      of the index says so
- [ ] `docs/v2/MEASURED.md` gains the note that a panel taller than the
      screen is cut at the screen's edge by the window server, measured once
      on this machine with a fixture of twelve hold cards, with the height
      the cut happened at; the decision names this as the price Nail chose
- [ ] `make app-lint`, `make app-test`

### Task 2: DOCUMENTS is the pinned and the last five

**Files:**
- Modify: `presence/Sources/PresenceCore/Documents.swift`,
  `presence/Tests/PresenceCoreTests/DocumentsTests.swift`,
  `presence/Sources/PresenceUI/PanelModel.swift`, `Words.swift`; Delete:
  `presence/Sources/PresenceCore/Days.swift` and its tests, the TODAY and
  Yesterday views and words; Modify: the scene fixtures, `RenderTests.swift`

- [ ] DOCUMENTS lists the pinned documents first, newest pin first, then
      the last five documents by their latest action, newest first, skipping
      any already listed as pinned and any hidden; one row per document id;
      no horizon count and no "N today" in the header, the label alone; the
      by-use ordering of M26 is withdrawn; a table test over pinned, recent,
      duplicate and hidden
- [ ] TODAY and Yesterday are gone from the panel: the sections, their
      folds, their words and their tests; NOW stays as it is
- [ ] the scenes that showed TODAY and Yesterday are re-fixtured and their
      captions rewritten; e2 (midnight) becomes the scene of a pinned
      document surviving fourteen days
- [ ] `make app-lint`, `make app-test`

### Task 3: the pin is an icon

**Files:**
- Modify: `presence/Sources/PresenceUI/Components/IconStrip.swift`,
  `IconShapes.swift`, `PanelModel.swift`, `Words.swift`,
  `presence/Sources/PresenceApp/SessionActions.swift`, tests beside each

- [ ] a DOCUMENTS row's strip has five icons: the four it has and a pin at
      the end, outline when the document is not pinned and filled when it
      is; pressing it pins or unpins; the tooltip says "Pin" or "Unpin"; the
      small pin glyph in the status column goes, because the icon says it; a
      test over the five icons, their order and the two states
- [ ] the row's menu keeps Hide and loses Pin, which the icon now owns; a
      test over the menu's one word
- [ ] `make app-lint`, `make app-test`

### Task 4: the note is found, not configured

**Files:**
- Create: `presence/Sources/PresenceApp/NoteFinder.swift`,
  `presence/Tests/PresenceAppTests/NoteFinderTests.swift`
- Modify: `presence/Sources/PresenceCore/HubIndex.swift` (or its successor),
  `presence/Sources/PresenceApp/Opener.swift`, `SessionActions.swift`,
  `PanelHealth.swift` (the SETTINGS rows), `Preferences.swift`,
  `presence/Sources/PresenceUI/Words.swift`, tests beside each

- [ ] `NoteFinder` asks Spotlight (`NSMetadataQuery`, scope the home folder,
      predicate: text content contains the document id, Markdown files) for
      candidate paths, then reads only the `gdoc:` front-matter block of each
      candidate, as `HubIndex` already does, and answers the one whose block
      names the id; two or more that do is a fault with its own sentence
      ("Two notes name this document"); none is `.notPaired`; the query runs
      off the main actor with a two-second ceiling and the answer is cached
      per document id until a Spotlight change notice names one of the
      candidates; a test through an injected finder over the four cases
- [ ] `hubPath` stays as an override: when set, the hub's index is asked
      first and Spotlight second; the Hub row leaves SETTINGS and the
      `noHub` fault goes, because there is no hub to choose; the guide says
      the override is a `defaults write` for a person whose notes are not
      indexed
- [ ] the md icon is grey only when no note names the document, and its
      tooltip says that; the Open md action opens the note the finder
      answered
- [ ] `docs/v2/MEASURED.md` records the query's timings on this machine
      with the method, and what Spotlight answers for a file that mentions
      the id in its body (the document map): a candidate the front-matter
      check rejects
- [ ] `make app-lint`, `make app-test`

### Task 5: the update glyph has three states and a clock

**Files:**
- Modify: `presence/Sources/PresenceUI/Components/PanelHeader.swift`,
  `IconShapes.swift`, `Theme/Palette.swift`, `Words.swift`,
  `presence/Sources/PresenceApp/HealthPoller.swift`, `UpdateAction.swift`,
  `Session.swift`, tests beside each

- [ ] the update glyph is always in the header after the version, in three
      states: quiet, the icon in the caption grey, when nothing newer is
      known, and pressing it runs `gdoc update`, which asks GitHub and
      installs if there is something newer or answers that this is current;
      available, the icon in the accent ink with a dot and the newer version
      written after the current one ("v2.18.0 → v2.19.0"), when `help
      --json` listed a newer stable, and pressing it runs the same update;
      running, the ring, while the child is up; a test over the three states
      and their words
- [ ] the app asks `help --json` on open as today and once every six hours
      while it runs, so a release cut during the day is noticed by evening;
      gdoc's own stamp keeps GitHub to once in 24 hours, and the test shows
      the timer asks the runner and nothing else
- [ ] after a successful update the app relaunches itself as M24 made it;
      after "already current" the glyph shows the check for one second and
      goes quiet
- [ ] `make app-lint`, `make app-test`

### Task 6: the words

**Files:**
- Modify: `docs/guide/presence.md`, `CLAUDE.md` (the app's read list in the
  presence invariant, if it names the hub), `docs/design/2026-10-17-presence-screens-v3.html`
  (scene captions only)

- [ ] the guide walks the panel as it now is: header with version and the
      update glyph's three states, NOW, DOCUMENTS as the pinned and the last
      five with the pin icon and Hide in the menu, SETTINGS with Launch at
      login and Quit; Open md through Spotlight and the `hubPath` override
- [ ] CLAUDE.md names the app's reads: the record, its plist, the front
      matter of the notes Spotlight names; under 300 lines
- [ ] `make test`, `make app-lint`

### Task 7: the acceptance rows

**Files:**
- Modify: this plan

- [ ] an acceptance table at the foot of this plan: each of Nail's
      sentences with the test that holds its answer
- [ ] `make app-lint`, `make app-test`

### Task 8: the reviewer's pass

**Files:**
- Modify: whatever the pass finds under `presence/`

- [ ] render every scene and run the checklist of the principles file over
      them as a reviewer who drew nothing (a subagent that reads images),
      against this plan's sentences; every blocking or should-fix finding is
      fixed here and listed in the commit message; the pass also states that
      no render shows a scroll thumb
- [ ] `make app-lint`, `make app-test`, `make app`

### Task 9: the close

**Files:**
- Modify: this plan, `docs/v2/DECISIONS.md` only if a task found the
  decision wrong

- [ ] the render index's "Still your call" list drops what this plan
      decided
- [ ] move this plan to `docs/plans/completed/`
- [ ] `make vet`, `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`,
      `make app-lint`, `make app-test`, `make app`
