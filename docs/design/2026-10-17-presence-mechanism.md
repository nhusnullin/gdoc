# The presence: the mechanism, decided by the team

This file is the record of a design night, kept as it was argued. The rules
that bind are in
`docs/plans/completed/2026-10-17-gdoc-v2-m23-the-presence-spec.md`; where this
file and the spec differ, the spec holds, and the section at the end says what
Codex and the plan review changed.

2026-10-17, night. Nail's brief, in meaning: gdoc itself must not change in
kind. It may report what it is doing in real time, "through a pipe or some
other way", and senior engineers decide how. The presence is a separate
application, so its changes never touch gdoc. `gdoc update` updates the
application too. For now it runs only on Nail's machine, behind a flag in
settings that an argument can also set. Main gdoc stays backward compatible.

Four seats read the code and the five design files, in parallel and without
seeing each other, and each wrote recommendations with reasons, costs and
guesses marked: the Go engineer who owns the binary, the macOS engineer, the
release engineer, and the trust guardian holding the skeptic's seat as well.
Their reports are summarised here; where they disagreed, the project manager
chose and says why. Nail has not reviewed this. Every choice below is his to
reverse in the morning, and the ones most likely to be reversed are listed
at the end.

## Facts the team found that change the design

1. There is no privacy toggle in the product decisions. Nail voted it down
   and voted yes on titles in the documents list. The brainstorm's rule 7,
   "no words from documents in the presence", still holds for every surface
   that is always visible: the menu bar glyph, its tooltip, any notification.
   The panel opens on a click and may show titles.
2. A hold already has a timer: 30 minutes in the process that made it
   (`mcpHoldLife`). The votes file said there is none. The panel needs a
   "ran out, nothing sent" state.
3. "Release" means send in the binary. Product decision 31's "Release
   without writing" uses the word for the opposite act. The panel never says
   Release. If a decline exists it is "Discard, nothing is sent", and in this
   design it does not exist (see decision 9).
4. Settings are decoded strictly and a settings error stops every command,
   `gdoc mcp` and `gdoc update` included. A binary older than the flag refuses
   a `settings.yaml` that carries it. So the flag ships one release before
   anyone sets it, and `--rollback` to an older release with the flag set
   breaks every command until the line is deleted by hand.
5. Every envelope passes one choke point, `printResult()` in `main.go`, and
   every command, terminal or chat, passes `dispatch`. A chat tool call reaches
   it through `safeDispatch`. That is where a start and an end are written.
6. A live watch is a series of processes: the skill calls
   `gdoc comments --since c --wait 9m` again and again. A start and an end
   per call shows "watching, last seen 14:31" with no event per poll and no
   change to `internal/comments`.
7. `mcplog.go` is the precedent: one line per tool call to stderr, never a
   document word, a hold's rule but not its text, pinned by three tests.
8. Product decision 7 says "append the same envelope". That would write
   document bodies, comment words and quoted errors to disk beside the token.
   All four seats refuse that wording.
9. `~/.local/bin/gdoc` on Nail's machine is a release binary (v2.12.1, with
   `gdoc.previous` beside it), not a checkout link. `gdoc update` is the real
   route there. The CLAUDE.md sentence saying it links to `bin/gdoc` is stale.
10. Measured tonight: a universal Swift bundle builds with `swift build` in
    13 seconds with no Xcode project; a bundle unzipped by a Go program carries
    no `com.apple.quarantine` and its ad-hoc signature stays valid; a tail on
    `DispatchSource` follows appends, partial lines and a monthly rotation;
    the panel renders to PNG headless from `swift test`, in light and dark,
    with SF Symbols, needing no Screen Recording permission.

## The channel: one append-only JSONL file, written by gdoc, read by the app

All four seats arrived at the same channel independently.

- `~/.config/gdoc-agent/presence/<yyyy>-<mm>.jsonl`, the month from the
  event's UTC time, so rotation is a file name and nothing is renamed.
  Directory 0700, file 0600, like the token. Each event is one open with
  `O_APPEND|O_CREATE`, one write of one line, one close.
- The binary never reads it. No code path opens it for reading, so nothing
  can arrive through it. A boundary test pins that the package exports no
  reader and that `O_APPEND` appears in no other room.
- The event is a fixed, allowlisted struct built at `dispatch` and in
  `mcpHolds`, never the envelope. A canary test feeds an envelope full of
  marker strings through the projection and asserts none reaches the line.
- Rejected, with the reasons that decided it: a unix socket (two-way at the
  OS level, loses every event while the app is closed, a dial outside the
  guard), a named pipe (blocks or fails with no reader, no history, writes
  over 512 bytes interleave), stderr or OSC (lands in Claude Code transcripts,
  never reaches the app), distributed notifications (cgo, and the build is
  CGO off), a `gdoc status` command (turns the binary into a reader of its own
  record, and is a seventeenth command).
- Cost: a history on disk in the config directory. It strains principle 2
  mildly, with `update-check.json` and `login-pending.json` as precedent, and
  the strain is written into the spec's gate. The sentence "the wait writes
  no file of its own" in `comments/doc.go` stays literally true, because the
  hook is in `dispatch`, and gains a pointer.

### The event, schema version 1

```
{"v":1,"at":"…Z","gdoc":"v2.13.0","pid":4242,"run":"9f1c…","call":3,"event":"start","command":"comments","via":"terminal","doc":"1AbC…","wait":true}
{"v":1,"at":"…Z","gdoc":"v2.13.0","pid":4242,"run":"9f1c…","call":3,"event":"end","command":"comments","via":"terminal","doc":"1AbC…","title":"Supplier register","ok":true,"ms":540000}
{"v":1,"at":"…Z","gdoc":"v2.13.0","pid":4300,"run":"a77e…","event":"held","command":"reply","via":"chat","doc":"1AbC…","hold":"h_ab12","rule":"echo","until":"…Z"}
{"v":1,"at":"…Z","gdoc":"v2.13.0","pid":4300,"run":"a77e…","event":"released","hold":"h_ab12"}
{"v":1,"at":"…Z","gdoc":"v2.13.0","pid":4300,"run":"a77e…","event":"expired","hold":"h_ab12"}
```

- `pid` with `run` (eight random bytes per process) pairs events and tells
  one process from another; liveness is `kill(pid, 0)` in the app, and a pid
  reused within a day is an accepted, unmeasured risk. `call` pairs a start
  with its end inside one process. `gdoc` is the binary's version, absent for
  a checkout build. `via` is `terminal` or `chat`. `doc` is the document id
  the command was handed, at start, and at end `ok` is the envelope's `ok`
  while `title` and `verified` are lifted from the top level of the
  envelope's data only when the command prints them with those types.
  `title` is capped at 200 runes. `ms` is monotonic elapsed time. The end is
  deferred in `dispatch`, so a recovered panic still writes one.
- Never in an event: document text, comment or reply words, quotes, proposal
  text, the hold's value, reason or text, error or warning sentences, any
  path, the account email, anything about the token, argv, the cursor, the
  mark, and no judging word. "Stuck" is the app's reading of `verified:false`
  or of a start with no end from a dead pid.
- Commands that report: read, comments, suggestions, export, restyle, reply,
  propose, withdraw, annotate, publish, update, auth login. Silent: help,
  completion, auth status (the app polls it), build (no document), probe, and
  mcp's own session.
- One fixture file, `go/internal/presence/testdata/events-v1.jsonl`, is pinned
  byte for byte by a Go test and decoded by the Swift tests, so the schema
  cannot drift between two languages. The app is strict on `v` and ignores
  unknown fields.
- A disk that is full or a directory that cannot be written changes no byte of
  the envelope and no exit code. One line goes to stderr, once per process.

**Resolved disagreement, the title.** The trust seat would leave titles out
of the first milestone; the Go seat keeps `title` in the event because
without it the app would have to run `gdoc read` to fill a documents row,
which pulls a whole body into a second process. Decision: `title` is in the
schema from version 1, because Nail voted for titles in the panel and a
schema that gains a field later is two schemas. Retention is bounded instead
(decision 11 below).

## The flag

- Settings key `presence: true|false` in `settings.yaml`, and the argument
  `--presence true|false`, a third setting flag beside `--mark` and
  `--config`, with the same refusals: given twice, no value, a value that is
  not true or false. The argument overrides the file. Chat gets it only from
  the file, because the Desktop extension passes no flags.
- It is a setting, never an identity check. "Only Nail" means whoever sets
  the flag. Nothing compares an email.
- It gates two things: the event write, and whether `gdoc update` touches the
  app. Off means: no `presence/` directory is created, not even empty; no
  envelope changes; `gdoc update` and `--rollback` are today's bytes; the app
  is never uninstalled, because not knowing never resolves to delete.
- The flag is in the command table with its sentence, marked experimental: no
  help screen, no completion script, and `help --json` shows it only when it
  is on. The 15 help-screen goldens, both completion scripts and the help
  object stay byte for byte for a colleague. The mark comes off in the
  decision that ships the panel to colleagues.
- No skill ever passes the flag; a boundary test scans `skills/` for it.
- Backward compatibility is proven by byte-compare tests over the stubbed
  session: read, comments --wait, reply and publish with the flag off and on
  print the same stdout, stderr and exit code, the only difference being the
  file. An unwritable log changes no byte of the object.

## Delivery

- The app travels inside both darwin zips at `presence/GdocPresence.app/`, as
  one universal binary, under the one `SHA256SUMS`. The guard, `Choose` and the
  asset names do not change. The app and the binary cannot come from different
  releases. Cost: every darwin download grows by under two megabytes whether
  the flag is on or not.
- Built by a second job in `release.yml` on a pinned macOS image: `swift
  test`, a universal release build, the bundle filled from `Info.plist` with
  `@VERSION@`, ad-hoc signed, verified (`codesign --verify --deep --strict`,
  `lipo -archs`, the plist version equals the tag), zipped with `ditto` and
  uploaded as an artifact. The ubuntu job gains `needs: presence`, downloads
  it, and copies it into the stage in a `darwin-*)` arm, so the platforms file
  stays the one list. A step before publish lists every zip and fails unless
  darwin zips carry the bundle and no other zip does.
- A broken Swift build fails the release rather than shipping without the
  app, because shipping without would split the version on Nail's machine.
  The same job runs on every push, so a break shows on its own commit.
- With the flag on, `gdoc update` installs the app at
  `~/Applications/GdocPresence.app`, keeping the previous bundle beside it as
  `GdocPresence.app.previous`. The zip is verified first; only entries under
  the bundle path are taken; absolute paths, `..`, symlinks, `__MACOSX`,
  `._*`, another bundle identifier or a plist version that is not the
  release's are refused by name; the bundle is staged beside the destination,
  read back, then swapped by rename after the binary's own three renames. Any
  app failure is a warning and never undoes the binary. Rollback swaps the
  binary as today, then the app if a previous one exists; twice is where it
  started. With nothing newer to install and the app missing or at another
  version, the release that is here installs it, which is also the first
  install.
- gdoc never launches the app. The first launch is the person, and the result
  says `open ~/Applications/GdocPresence.app`. After an update the app sees its
  on-disk bundle version differ from its own, opens its own path as a new
  instance and quits. `TestOnlyDesktopRunsAProgram` stays as it is, and a new
  test pins `withExtension` as the only caller of `runProgram`.
- Nightly: nothing new; `nightly.yml` calls `release.yml`. `release/install.sh`
  ignores `presence/` in this milestone, pinned by a test, because it reads no
  settings and a colleague must get exactly today. Windows and every
  non-darwin platform: no app, and with the flag on the updater writes one
  warning naming the platform.
- Gatekeeper: quarantine is set by the downloading program, and Go sets none,
  so a bundle `gdoc update` writes is not assessed on first launch and an
  ad-hoc signature suffices. Measured in part tonight (the binary installed
  by update carries only `com.apple.provenance`; a Go-unzipped bundle has no
  quarantine and a valid signature). Not yet measured: the launch of a
  CI-built bundle on macOS 26, a browser-downloaded copy being blocked as the
  negative control, and the swap-while-running relaunch. The plan carries
  these as measurements into `MEASURED.md`, confirmed by Nail. Developer ID
  and notarisation stay with M17 and are not needed while the app is Nail's.

**Resolved disagreement, where the app lives.** The macOS seat wanted the
bundle beside the binary in `~/.local/bin` so the app finds gdoc without
configuration. The release seat wanted `~/Applications`: the per-user apps
folder, Spotlight-indexed, a real path for launch at login, and no executable
in a hidden bin folder. Decision: `~/Applications`, and the app looks for gdoc
at `~/.local/bin/gdoc` first and beside its own bundle second, showing the
path it tried in HEALTH.

## The app

- Native Swift, AppKit shell with SwiftUI content: an `NSStatusItem` and a
  borderless key-capable `NSPanel` holding an `NSHostingView`. Right-click
  opens the same panel (`sendAction(on: [.leftMouseUp, .rightMouseUp])`),
  which `MenuBarExtra` cannot do, and the design's white panel with a drawn
  edge and a three-layer shadow is custom drawing that `NSPopover` fights.
  Minimum macOS 14, for `@Observable`, `ImageRenderer` and `SMAppService`.
- Its own Swift package in a top-level `presence/`, a sibling of `go/`, so
  `go/boundary` never sees Swift: `PresenceCore` (event decoding, the fold
  into NOW, DOCUMENTS and TODAY, the tail, the relaunch decision, the child
  runner protocol; Foundation only), `PresenceUI` (views and the glyph),
  `Presence` (the executable), two XCTest targets with JSONL fixtures. Make
  targets `app`, `app-test`, `app-lint`. The bundle is `bin/GdocPresence.app`
  for a checkout build, which stamps `0.0.0`.
- The app tails the current and the previous month's file with
  `DispatchSource` on the file and the directory, decodes strictly, folds, and
  treats every line as forgeable data: it never takes a URL, a path or a
  program from the log, and builds a Docs URL itself from an id that matches
  the id pattern. Liveness is `kill(pid, 0)` plus the logged `started` against
  the process start time, so a reused pid does not look alive.
- The app's only outbound acts are fixed argv lists for the gdoc binary at an
  absolute path: `auth status` on open and every ten minutes while a watch
  runs, and `auth login` as a child whose first `https://` line on stderr is
  opened with `NSWorkspace`. No flag, environment variable or subcommand
  exists for the app alone, and gdoc never behaves differently because the
  app is the caller. An XCTest lists the complete argv set; a source scan
  fails on `URLSession`, `import Network` or a `Process` outside the one
  runner file. The app never opens the token or `login-pending.json`, and it
  never calls `gdoc help`, which would run the daily GitHub check unasked.
- The app's only state is its dismissals, in UserDefaults, keyed by run id.
  It writes nothing beside the token and never to the log.
- Screenshots for review and for Nail's report are offscreen renders of
  fixture events from `swift test`, in light and dark at 2x, needing no
  permission. A real-screen capture once per milestone is optional.

## What is not built, and what is refused

- No channel into `gdoc mcp` from the app. The panel's hold card carries
  "Open chat" and nothing else. Product decisions 31, 47 and 48 (discard a
  hold from the panel, sign out releases holds) would each need a route into
  the process that owns the confirm tools; that is a steering surface even if
  it only ever drops, so it is Nail's decision and not built. The trust seat
  refuses, in every form, an app that approves a held write.
- `gdoc auth logout` is not in this milestone. When it is built, a hold must
  not survive a token swap by another process; the trust seat suggests the
  hold record a hash of the refresh token and refuse release if it changed.
- gdoc never starts, signals, connects to or waits on the app. Every seat
  refused this independently. The one external program stays the one.
- No macOS notifications: Notification Center keeps titles and shows them on
  the lock screen.
- Retention is bounded by the binary: it keeps the current and the previous
  month's file and removes older ones when it writes, under the flag. This
  reverses product decision 8 ("the app never prunes"), which was a product
  owner's guess Nail had not reviewed; the trust seat's reason is that once
  the token moves to the Keychain, a forever history of titles would be the
  most sensitive readable thing in the config directory. Nail decides.

## Two milestones

**M23, the presence: the channel, the flag, the delivery, the first glance.**
Everything above on the binary side, in full: the flag, the events including
holds, the fixture, the tests, the boundary tests, the delivery through
`gdoc update` with rollback, the release job. On the app side, the skeptic's
cut, narrowed once more by Codex: one `NSStatusItem` that shows a ring while
a start has no end and its pid is alive, the mark while a `comments --wait`
runs, a pause glyph while a hold waits, and nothing when idle; a plain panel
with NOW and TODAY built from the fixture-tested fold, the app's version, the
last event's gdoc version, a row saying "Updated, quit and reopen" when the
bundle on disk is newer than the running app, and Quit. It launches no
process at all, opens only the event files and its own plist, decodes
strictly. This is what Nail runs the morning after.

**M24, the panel.** DOCUMENTS with the 14-day horizon, Yesterday, Show more,
the two palettes as drawn, the action icons (Open in Docs first, then Copy
link, Open md, Continue the AI session), Stop live as a SIGINT to a pid the
app has checked is gdoc, the stuck card with its dismiss, the account row
from `auth status`, Sign in and Sign out, Launch at login, the self-relaunch
after an update once the swap while running is measured, the green check on
`verified: true`, every empty and error state the product decisions list,
and the review-before-show pass over rendered screens. M24 is planned after
M23 ships and Nail has used it.

## Codex's read, the same night

Codex (gpt-6-astra, read-only over the tree) read this document before the
specification was finished. Eight of its findings were real and are in the
specification now: `release.yml` runs on tags only, so the push job lives in
`go.yml`; a nightly tag whose release failed is never retried, so the nightly
learns to; the schema carried no gdoc version; `ok` is on the envelope, and
`title` and `verified` exist on some commands only; a recovered panic would
have left a start with no end in a live process; `started` had no defined
source; "the app and the binary cannot come from different releases"
contradicted "an app failure never undoes the binary", so skew is accepted
and shown; `O_WRONLY` was missing and a symlink at the path was not refused.
Three of its cuts were taken for M23: no self-relaunch, no account row and no
`auth status` from the app, a plain native panel. Two were declined: cutting
TODAY and titles, and renaming "stuck". One question it raised is Nail's: it
read the strain on principle 2 as needing a written exception, and a
principle is argued first.

The plan review that followed (the `plan-review` agent, against the tree)
added: the nightly retry is dropped, because the Actions page re-runs a
failed release already; `verified` is lifted only for the five read-back
writers; the repacked bundle is verified on the macOS job, because the ubuntu
job has no `codesign`; the app persists nothing in M23, since nothing can be
dismissed yet; the stage is a fixed `<dest>.new` removed first; the README
gains no line, since it is at its ceiling and the flag is experimental; and
the paragraphs above that still say `O_APPEND|O_CREATE`, "cannot come from
different releases", "the same job runs on every push", the self-relaunch,
"nightly: nothing new", the app looking for gdoc, `auth status` and `auth
login` from the app, and `started` against the process start are superseded
by the spec.

## Decisions Nail is most likely to reverse

1. `~/Applications/GdocPresence.app` rather than beside the binary.
2. Retention of two months, pruned by the binary, reversing product decision 8.
3. `title` in the event from version 1.
4. The experimental flag hidden from the help screen until colleagues get it.
5. A broken Swift build fails the whole release.
6. No discard of a hold from the panel, so decisions 31, 47 and 48 wait.
7. The name `GdocPresence`.

## M24, the morning after

Nail opened the M23 panel at 09:17 and said it looks nothing like the design.
Two seats prepared the full panel the same morning: the macOS engineer with
the visual designer's brief read the nineteen scenes of the third screens
version into a token sheet, a component list, the fold the panel needs, the
bar images, the outbound acts and a 24-task breakdown; the trust guardian
with the binary's owner found that publish writes no document id, that a
hold already survives a change of account, that a live watch is signalable
only while a wait is in flight, and that six things the screens draw are
forbidden in the record. The project manager's choices are in the M24
specification's decision table and the DECISIONS.md entry of the same date;
the ones most likely to be reversed: no note path in the record (the app
reads the hub's front matter instead), Dismiss instead of Release, the dim
mark when idle, launch at login on by default, and the stuck rule that keeps
`verified: false` a hollow check against the vote. The two seats' reports are
in the session's scratch folder, the two plans are M24a and M24b, and the
swap-under-a-running-instance measurement that self-relaunch rests on was
taken at 09:25.
