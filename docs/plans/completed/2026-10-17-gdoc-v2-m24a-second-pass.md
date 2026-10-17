# gdoc v2 Milestone 24a, the second reviewer's pass. Plan

2026-10-17, evening. A short run on the branch `gdoc-v2-m24a-the-panel` after
its twenty-six tasks, before the pull request. A reviewer who drew nothing ran
the checklist of `docs/design/2026-10-17-presence-design-principles.md`
("Review before showing Nail") over the renders of commit 47034ad and wrote
thirty-three findings. Three block showing the panel, fifteen should be fixed,
fifteen are notes. This plan fixes the three and the fifteen, carries the
notes that are Nail's call into the captions, and runs the pass again. One
commit per task. The spec `docs/plans/2026-10-17-gdoc-v2-m24-the-panel-spec.md`
holds the rules and this plan repeats none; where a fix below and the spec
disagree, the task stops and says so.

The visual source of truth is still
`docs/design/2026-10-17-presence-screens-v3.html`. Its titles and addresses
are invented stand-ins and so are the fixtures'; nothing in this plan changes
that.

## Context (from discovery)

- The scenes are `presence/Tests/PresenceUITests/Scenes.swift` (one `Scene`
  per screen, with its `caption`, the index page's text under each render)
  and the fixtures under `presence/Tests/PresenceUITests/Scenes/*.jsonl`,
  one per scene, each its own day. `Shots.swift` writes the PNGs and
  `index.html` under `PRESENCE_SHOTS`; `RenderTests.swift` holds the
  assertions over the rendered scenes.
- `r1.jsonl` carries a `step` line with `"title":"Chapter 4"` for run
  `3000000000000001`; `r2.jsonl` and every later fixture keep only that run's
  `start` and `end`, which carry no title, so the stopped publish is nameless
  from the second screen on. The fold takes a run's title from its step lines
  only.
- Run `3000000000000003`, the 11:38 annotate, is `via: code` in `r3.jsonl`
  and `e6.jsonl` and `via: terminal` in r4 to r9, e3, e9 and e10.
- `r7.jsonl` writes the approved hold as a `landed` event from the chat pid
  and a second reply run `via: terminal` from the same pid 7300, so TODAY
  shows two rows for one write.
- `e9.jsonl` ends the watch with `stop: auth`, so the process is gone; the
  words on the card and the row say "paused" and the Stop icon is dimmed.
- The DOCUMENTS header has three forms on screen: "N today" (r3 to r8, and
  e2 where it says "1 today" above four rows from three days), "14 days · N"
  (r9, e1) and "14 days · 3 today" (the rows sheet).
- The principles file lists "Release? Sign out?" as the inline confirms in
  principle 3; the cards sheet draws "Dismiss this hold?".

## Development Approach

TDD, one commit per task, the test first and failing, under
`cd presence && swift test`; `make app-lint` and `make app` green at every
commit; `make test` green with the live variables unset. Every colour, size,
weight and count a test states is a literal. Renders go under
`PRESENCE_SHOTS`, outside the repository. A task that finds the screens file,
the spec and the code disagree stops and says so.

## Validation Commands

```bash
make app-lint
make app-test
make app
env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test
PRESENCE_SHOTS=/tmp/presence-shots make app-test
```

## Progress Tracking

Six tasks. A task is done when every box is ticked and its commit is on the
branch.

## Tasks

### Task 1: the fixtures tell one story

**Files:**
- Modify: `presence/Tests/PresenceUITests/Scenes/*.jsonl`,
  `presence/Sources/PresenceCore/Fold.swift` (or where the run's title is
  folded), `presence/Tests/PresenceCoreTests/*Tests.swift`,
  `presence/Tests/PresenceUITests/RenderTests.swift`

- [x] a test that a run whose `start` carries a title, or whose any event
      carries one, is named in the stuck card and in its TODAY row; the fold
      keeps the title from any event of the run, the latest one winning
- [x] `r2.jsonl` to `r6.jsonl` carry the stopped publish's title the same
      way `r1.jsonl` does, so the card reads "Publish stopped · Chapter 4"
      and the TODAY row "Publish · Chapter 4"; `e3.jsonl`'s running publish
      carries its title too (its step line has none)
- [x] run `3000000000000003` is `via: code` in every fixture of that day
      (r3 to r9, e3, e6, e9, e10), one run, one source
- [x] `r7.jsonl`, `r8.jsonl`, `r9.jsonl`: the approved hold is one write.
      The reply that landed is the chat's own run (`via: chat`, the chat pid)
      and there is no second terminal run; TODAY shows "Reply · approved by
      you in chat · verified" as one row, as the mockup draws it
- [x] `e1.jsonl`: six titles that survive the middle ellipsis at the row's
      width (they differ at the head or the tail, not only in a number in
      the middle); a render test asserts the six rows render as six
      different strings
- [x] the 13:10 reply in r4 to r9 moves to a minute of its own, so the TODAY
      pair no longer reads as a duplicate
- [x] `make app-lint`, `make app-test`

### Task 2: the words say what the record says

**Files:**
- Modify: `presence/Sources/PresenceUI/Words.swift`, the stuck card, the
  hold row's sub-line, the DOCUMENTS header, `presence/Tests/PresenceUITests/WordsTests.swift`,
  `RenderTests.swift`, `Scenes.swift` (e9 caption)

- [x] e9: a watch that ended with `stop: auth` is stopped, not paused. The
      row says "stopped 15:29 · sign-in expired", the card says "1 live
      watch stopped. Sign in and start it again.", the Stop icon is not
      drawn (nothing to stop), and the caption says that whether gdoc resumes
      from the same cursor after a sign-in is unmeasured; a test pins the
      three strings and the absent icon
- [x] the hold's TODAY sub-line uses the heading form of the command
      ("Proposal held by rule Unread target", "Reply held by rule
      Dictated"), never the raw command name; a test over every command the
      vocabulary names
- [x] the stuck card's reason for the `other` code is only where the reason
      is ("The terminal that ran it has the reason."), with no second
      "stopped" and no time, because the title, the glyph and the TODAY row
      already carry both; a test pins the string
- [x] one DOCUMENTS header rule: "N today" when every row listed is from
      today, otherwise "14 days · N", where N counts every document in the
      list including the ones behind "Show N more"; e2 reads "14 days · 4";
      the rows sheet uses the same words; a test over the three cases
- [x] `make app-lint`, `make app-test`

### Task 3: the rows and the cards use their width

**Files:**
- Modify: the TODAY row, the stuck card, the scroll box, the tooltip, the
  hold confirm card, under `presence/Sources/PresenceUI/`; `RowTests.swift`,
  `CardTests.swift`, `ScrollTests.swift`, `ControlTests.swift`

- [x] TODAY sub-lines wrap to a second line before they truncate, the same
      rule the DOCUMENTS rows already follow; "reply held by rule Dictated ·
      waiting in Desktop chat" is whole on two lines; a test measures the
      row's height at that string
- [x] the stuck card's title: when the head cannot keep eight characters
      before the ellipsis, the tail is dropped and the title goes in whole on
      two lines, and the second line uses the full width under the icons;
      "Export stopped · Proposal to the Falcon partners, draft v1" renders
      with the word "Proposal" whole; a test pins the two lines
- [x] one scroll thumb per edge: the inner DOCUMENTS box's thumb sits 3 pt
      inside the box, and the panel's own indicator is hidden while an inner
      box is scrollable, or the e1 caption says the panel scrolls as well;
      a test counts the thumbs in the e1 render
- [x] the action tooltip is anchored under its icon and ends at the panel's
      right inset, never covering the row below; a test places the tooltip
      for the third icon and asserts its frame against the icon's
- [x] the hold confirm's consequence line sits inside the card's padding,
      aligned with the title; a test measures its x
- [x] `make app-lint`, `make app-test`

### Task 4: the captions name what is still Nail's call

**Files:**
- Modify: `presence/Tests/PresenceUITests/Scenes.swift`, `Shots.swift`
  (the sheets), `docs/design/2026-10-17-presence-design-principles.md`

- [x] the index page opens with a list titled "Still your call", one line
      each, naming where the render may go: the idle bar's dim mark against
      no mark (principle 1, mockup r9); Launch at login on by default,
      against product decision 32; a publish with no document yet kept out
      of DOCUMENTS until it has an id (against decision 10 and the mockup's
      dimmed row); a trashed document's row unstruck (e6); the e8 countdown
      (decision 39 against principle 5); and the DOCUMENTS header rule of
      Task 2
- [x] the e8 caption says the render stacks two moments (waiting for Google
      and just signed in) on one panel, as the mockup did
- [x] the HEALTH row rule is said once in the index: Account first, Quit
      last, and Skills, Claude Desktop and Network drawn only when the record
      or the check carries the fact
- [x] the sheets use the scenes' words and nothing else: no "Watching", no
      bare "Mark", no "In the trash" with a struck title, no "Proposed 2", no
      "Nothing since", no "14 days · 3 today"; a test asserts every string on
      a sheet is in the vocabulary list
- [x] principle 3 names the one inline confirm that exists, "Dismiss this
      hold?" with Dismiss and Keep, and the amendments section gets a dated
      entry for this pass, the findings listed by their letters
- [x] `make app-lint`, `make app-test`

### Task 5: the reviewer's pass, second round

**Files:**
- Modify: whatever the pass finds, under `presence/` and `docs/design/`

- [x] render every scene (`PRESENCE_SHOTS=... make app-test`), then run the
      checklist of the principles file over `index.html` and the PNGs as a
      reviewer who drew nothing (a subagent that reads images), against the
      mockup scene by scene; every finding it rates as blocking or should-fix
      is fixed in this task, and the findings and fixes are listed in the
      commit message; a finding that is a product choice goes to the "Still
      your call" list instead
- [x] `make app-lint`, `make app-test`, `make app`, and
      `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`

### Task 6: the close

**Files:**
- Modify: `docs/plans/completed/2026-10-17-gdoc-v2-m24a-the-panel.md`
  (one line under its acceptance rows naming this pass), this plan

- [x] the acceptance rows of the M24a plan gain one row: the second
      reviewer's pass ran on the renders of the final branch, and its
      report's three blocking findings are closed by Tasks 1, 2 and 4
- [x] move this plan to `docs/plans/completed/`
- [x] `make app-lint`, `make app-test`
