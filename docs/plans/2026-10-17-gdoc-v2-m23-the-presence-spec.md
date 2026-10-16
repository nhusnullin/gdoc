# gdoc v2 Milestone 23: the presence. The channel, the flag, the delivery, the first glance. Specification

2026-10-17. Nail's brief, given the same night and summarised in
`docs/design/2026-10-17-presence-mechanism.md`: gdoc itself must not change
in kind; it may report what it is doing in real time, and senior engineers
decide how. The presence is a separate application whose changes never touch
gdoc. `gdoc update` updates it too. It runs only on Nail's machine for now,
behind a settings flag an argument can also set. Main gdoc stays backward
compatible. Four seats designed the mechanism in parallel and the project
manager resolved their disagreements; Nail has not reviewed the result, and
the seven decisions he is most likely to reverse are listed in that file. The
decision entry is in `docs/v2/DECISIONS.md` under 2026-10-17, and the task
list is `docs/plans/2026-10-17-gdoc-v2-m23-the-presence.md`. It starts from
`main` after M22 and M21 merged.

## Principles

Serves: 4, every word costs a reader's attention. Today a live review is
hours of silence in a chat window, and alive and dead look the same. One
glance at the menu bar answers "is it alive" and "is it waiting on me" without
a word in the chat: a hold waiting, a run in flight, a watch, or a run that
stopped with its outcome unknown; the failed read-back joins the glyph with
the green check in the next milestone. Serves 3 as well: the record
the binary writes is facts only, `verified: false` stays a fact, and nothing
in the design lets the app approve a held write.

Strains: 2, nothing lives hidden beside the hub. The binary writes a history
of its own runs into the config directory, one line per command start and
end. It is in the config directory and not the hub, like `update-check.json`
and `login-pending.json`, gdoc never reads it, and it carries no document word
but a title. Nail accepted that the config directory is the place when he
chose a menu bar over a thread. And 1, if it is not inside the binary it does
not travel: the release zip carries a second program. gdoc does not depend on
it, never starts it, and is whole without it, as it is whole without the
skills and the extension template the zip already carries.

## What was found before deciding

- Every command, terminal or chat, passes `dispatch` in `go/cmd/gdoc/main.go`,
  and every envelope passes `printResult`. A chat tool call reaches `dispatch`
  through `safeDispatch`. A held chat write never reaches it; holds live in
  `mcpHolds` and die with the process after `mcpHoldLife`, 30 minutes.
- A live watch is a series of `comments --wait` processes, so a start and an
  end per call shows "watching, last seen" with no event per poll and no
  change to `internal/comments`.
- `mcplog.go` already writes one fact line per tool call with no document word
  in it, pinned by three tests. It is the precedent for lifting named keys out
  of an envelope rather than writing the envelope.
- Settings are decoded strictly and a refusal stops every command. A binary
  older than a key refuses a file that carries it.
- `~/.local/bin/gdoc` on Nail's machine is a release binary with
  `gdoc.previous` beside it, so `gdoc update` is the real route there, and
  the CLAUDE.md sentence about a link to `bin/gdoc` is stale.
- Measured: a universal Swift bundle builds with `swift build` in 13 seconds
  with no Xcode project; a bundle a Go program unzips carries no quarantine
  attribute and keeps a valid ad-hoc signature; a `DispatchSource` tail
  follows appends, partial lines and a monthly rotation; the panel renders to
  PNG from `swift test` with no Screen Recording permission. The machine holds
  no code-signing identity; Developer ID stays with M17.
- `go/boundary/workflows_test.go` fails the suite unless `release.yml` has
  exactly one job, and its `Needs` field is a string.

## Decisions taken

Nail, 2026-10-17, in chat:

| Rule | Where |
|---|---|
| gdoc does not change in kind. It may report what it is doing in real time; the engineers choose the channel | cmd, presence |
| The presence is a separate application, so a change to it never touches gdoc | presence/, release |
| `gdoc update` updates the application too | update, cmd |
| A flag in settings, also settable by an argument, makes it available only to whoever sets it | settings, cmd |
| Every existing function of gdoc keeps working exactly as today | every test that exists |
| Plan, TDD, review, run board, release, as every milestone | docs/plans |

Taken in the design pass by the team and the project manager, each one a
sentence Nail can strike:

| Rule | Where |
|---|---|
| The channel is one append-only JSONL file per month, `presence/<yyyy>-<mm>.jsonl` in the config directory, directory 0700 and file 0600. Each event is one open with `O_WRONLY\|O_APPEND\|O_CREATE\|O_NOFOLLOW`, one write of one line checked for a short write, one close. A symlink or a non-regular file at the path is refused by name and nothing is written. The month is the event's UTC month, so nothing is renamed. Local disk only: a config directory on a network or synced volume is unsupported, and the guide says so | presence, config |
| The binary never reads the file. The package exports no reader, `O_APPEND` appears in no other room, and no decision gdoc makes depends on a line in it: not a hold rule, not a receipt, not a cursor. The file is a best-effort record of observations, never authoritative state | presence, boundary |
| The event is a fixed allowlisted struct, schema `v: 1`, never the envelope. Fields: `v`, `at`, `gdoc` (the binary's version, absent for a checkout build), `pid`, `run` (a random per-process id), `call`, `event`, `command`, `via`, `doc`, `wait`, `title`, `ok`, `verified`, `ms`, `hold`, `rule`, `until`. `ok` is the envelope's `ok`. `title` is optional and lifted from the top level of `data` only when it is there as a string. `verified` is lifted only for the five commands that read a write back, reply, propose, withdraw, annotate and publish, marked in the table; `update` and `restyle` print a `verified: false` that means no write happened, and that must never become a failed read-back in a record meant to last. `ms` is monotonic elapsed time. `doc` is the id the existing id parser gave, never a raw argument, and absent for `update` and `auth login`. Never: document text, comment or reply words, quotes, proposal text, a hold's value, reason or text, error or warning sentences, any path, the account, anything about the token, argv, the cursor, the mark, or a judging word | presence |
| `title` is the one document word, capped at 200 runes, because without it the app would run `gdoc read` to fill a row. The cap bounds the line and removes no sensitivity; that is accepted with Nail's vote for titles | presence |
| Events are `start` and `end` for the commands that name a document plus `update` and `auth login`; for a chat hold, `held`, then exactly one of `released` (consumed by a send, which says nothing about whether the send succeeded), `expired`, or `landed` (removed because another send landed). `help`, `completion`, `auth status`, `build`, `probe` and the mcp session itself write none. What happens in chat before `safeDispatch` (the cache, an argument refusal, the pin, the judge) writes nothing: the presence is command execution and holds, and the guide says so | cmd |
| The end is deferred in `dispatch`, so a recovered panic still writes an `end` with `ok: false`. A start with no end whose pid is dead means the outcome is unknown, and that is the only thing the app may say about it | cmd, presence/ |
| A write that fails changes no byte of the envelope and no exit code. One line goes to stderr, once per process. The write is synchronous on local disk, and that latency is accepted | presence, cmd |
| The binary keeps the current and the previous month's file and removes older ones when it writes, under the flag: regular files of the exact `<yyyy>-<mm>.jsonl` name only. With the flag off nothing is removed, so old files then stay until a person deletes them. This reverses product decision 8 and is Nail's to reverse back | presence |
| One fixture, `go/internal/presence/testdata/events-v1.jsonl`, written by the Go writer and pinned byte for byte by a Go test, and a copy under `presence/Tests` held equal by a Go test, so the schema cannot drift between two languages | presence, presence/ |
| The flag is the setting `presence: true` and the argument `--presence true`, a third setting flag beside `--mark` and `--config`, with the same refusals. The argument overrides the file. Chat gets it from the file only. It is a setting and never an identity check | settings, cmd |
| The flag is in the command table with its sentence and marked experimental: no help screen line, no completion script line, and `help --json` carries it only when it is on. The 15 help goldens, both completion scripts and the help object stay byte for byte | cmd |
| Off means today's bytes: no `presence/` directory is created, no envelope changes, `gdoc update` and `--rollback` are unchanged, the app is never uninstalled. Proven by byte-compare tests over the stubbed session with the flag off and on | cmd |
| No skill passes the flag | boundary |
| The app travels inside both darwin zips at `presence/GdocPresence.app/`, one universal bundle, under the one `SHA256SUMS`. No other zip carries it. The bundle carries no symlink and no resource but its plist, so an unzip and a rezip keep its signature valid, which the release asserts after the repack | release |
| `release.yml` gains a second job on a pinned macOS image that tests, builds, stamps, ad-hoc signs, verifies and uploads the bundle; the ubuntu job needs it, copies it into the stage in a `darwin-*)` arm, and a step before publish lists every zip and fails unless darwin zips carry the bundle and no other does. `release.yml` runs on tags only, so `go.yml` gains the same Swift job for every push. A broken Swift build fails the release, and a failed release is re-run from the Actions page; the nightly is unchanged. The ubuntu job has no `codesign`, so the macOS job proves the zip round trip itself: it zips the bundle the way the pack step does, unzips it and verifies the signature | release, boundary |
| With the flag on, `gdoc update` installs the bundle at `~/Applications/GdocPresence.app`, keeping the previous one as `GdocPresence.app.previous`. The zip is verified first; only entries under the bundle path are taken; absolute paths, `..`, symlinks, `__MACOSX`, `._*`, a duplicate or case-colliding path, more than 2,000 entries or 64 MiB expanded, another bundle identifier and a plist version that is not the release's are refused by name; a destination that is a symlink, a file, or a bundle with another identifier is refused and nothing moves; the bundle is staged under a unique name beside the destination, read back and summed, and swapped by rename after the binary's three renames. Every app failure is a warning that names where the bundles are and never undoes the binary, so the installed app and binary can differ in version, and the app shows both. Rollback swaps the binary, then the app if a previous exists, each swap its own involution. A run with nothing newer installs the app only when the installed binary is the newest candidate and the app is missing or at another version, and never under `--check` | update, cmd |
| gdoc never launches the app. The first launch is the person, and the result says `open ~/Applications/GdocPresence.app`. After an update the app reads its on-disk `Info.plist` and shows one row, "Updated to X. Quit and reopen"; it does not relaunch itself in this milestone, because the swap while running is unmeasured. `withExtension` stays the only caller of `runProgram`, pinned | cmd, presence/, boundary |
| `release/install.sh` ignores `presence/` in this milestone, pinned by a test. Non-darwin platforms get no app, and with the flag on the updater writes one warning naming the platform | release, cmd |
| The app is native Swift: an AppKit `NSStatusItem` and a borderless key-capable `NSPanel` holding SwiftUI content, macOS 14 minimum, in its own Swift package at `presence/`, a sibling of `go/`. `PresenceCore` is Foundation only and holds the decoding, the fold, the tail, liveness and the bundle-version check; `PresenceUI` the views and the glyph; `Presence` the executable. XCTest over fixtures, render tests to PNG, no UI tests. The panel is plain native presentation in this milestone; the two palettes as drawn are M24 | presence/ |
| The app tails this and last month's file with `DispatchSource`, rescans at the month boundary on a timer, reopens when the file is replaced or removed, parses newline-terminated records only, keeps the trailing partial bytes in a bounded buffer, skips a record it cannot decode and keeps tailing, refuses an unknown `v` as "gdoc is newer than this app", ignores unknown fields, and treats every line as forgeable data: it never takes a URL, a path or a program from the log. Liveness is `kill(pid, 0)`; a reused pid within one day is an accepted, unmeasured risk, and `run` pairs events without the kernel | presence/ |
| The app launches no process in this milestone. It opens only the event files under `~/.config/gdoc-agent/presence/` and its own `Info.plist`. A source scan refuses `URLSession`, `import Network` and `Process` anywhere in the package. It never opens the token or `login-pending.json`. The account row, `auth status` and `auth login` from the panel are M24 | presence/ |
| The first glance, this milestone: the glyph is a ring while a start has no end and its pid is alive, the mark while a `comments --wait` runs, a pause glyph while a hold waits, and nothing when idle; the panel is NOW and TODAY from the fold, a row for a start with no end and a dead pid ("stopped, outcome unknown"), the app's version and the last event's `gdoc` version, the "Updated, quit and reopen" row when the plist differs, and Quit. The app's only state is its dismissals, in UserDefaults | presence/ |
| No channel into `gdoc mcp` from the app. The hold card says "Open chat" and nothing else; the panel never says "Release". Product decisions 31, 47 and 48 wait for Nail | presence/ |
| No macOS notifications. Rule 7 of the brainstorm holds for every always-visible surface: the glyph, its tooltip, the Dock. The panel opens on a click and may show titles; the signed-in email joins it with the account row in M24 | presence/ |
| CLAUDE.md gains the presence rows and the two invariants, stays under 300 lines, and its sentence about `~/.local/bin/gdoc` linking to `bin/gdoc` is corrected | docs |

## What is not built

The panel of the design files: DOCUMENTS with its 14-day horizon, Yesterday,
Show more, the action icons, Stop live, the stuck card's dismiss, the account
row, Sign in and Sign out, Launch at login, the self-relaunch after an
update, the green check, the empty and error states, the two palettes as
drawn. That is M24, planned after Nail has used M23 and the swap while
running is measured. `gdoc auth logout`. A discard of a hold from the panel.
A `gdoc status` command. A socket, a pipe, a notification. Developer ID and
notarisation. Windows. A change to `release/install.sh` or `install.sh`. A
rewrite of the help goldens. An amendment to PRINCIPLES.md: Codex read the
strain on principle 2 as needing a written exception rather than a strain,
because the file holds document ids and titles outside front matter; a
principle is argued first, so that sentence is Nail's to write or refuse, and
the gate above states the strain as it stands.

## Codex's read, 2026-10-17

Codex (gpt-6-astra, read-only) read the mechanism document against the tree
before this spec was finished. Accepted and written into the rules above:
`release.yml` runs on tags only, so the push job lives in `go.yml`; a nightly
tag whose release failed is never retried by the nightly, and the plan
review then found that the Actions page's re-run already covers it, so the
nightly stays as it is; the schema carried no gdoc version, and `ok` is on
the envelope rather than in `data`; `title` and `verified` exist on some
commands only, and the plan review narrowed `verified` to the five read-back
writers because `update` and `restyle` print a false that means no write; a recovered panic
would have left a start with no end in a live process, so the end is
deferred; `started` had no defined source, so a random `run` id pairs events
and the kernel is not asked; "the app and the binary cannot come from
different releases" contradicted "an app failure never undoes the binary",
so skew is accepted and shown; `O_WRONLY` was missing and symlinks at the
path were not refused; chat work before `safeDispatch` is not covered and is
now said so. Codex's cuts for one night, taken: no self-relaunch, no account
row and no `auth status` from the app, a plain native panel. Declined: cutting
TODAY and titles (the fold is fixture-tested and titles are Nail's vote), and
renaming "stuck", which is the product decisions' word and the app's reading
only.

## Measurement

Three, each written into `docs/v2/MEASURED.md` with how it was taken, and
Nail confirms the method before the app trusts the claim.

1. Two writers. Two built `gdoc` binaries append 10,000 lines each to one
   file in a shell loop; every line decodes strictly and all 20,000 are there.
   The `-race` test with sixteen goroutines on separate file descriptions is
   the offline half.
2. Gatekeeper, the Go route. A bundle built by the release job, unpacked by
   the updater's own code into a temp directory: `xattr -lr` shows no
   quarantine, `codesign --verify --deep --strict` says valid, `open` launches
   it with no dialog and `syspolicyd`'s log shows no block. The negative
   control: a copy with the quarantine attribute written by hand must be
   blocked. What can be measured before the first release is measured on a
   locally built bundle and said so.
3. The swap while running. A second build with another cdhash replaces the
   bundle by rename while the first runs; the running app keeps working and
   shows the "Updated, quit and reopen" row; after a quit and an `open`,
   exactly one instance runs at the new version. Whether a self-relaunch is
   safe is M24's question, asked only after this is recorded.

If measurement 2 fails on Nail's machine, the run stops and Nail decides
between the browser route with "Open Anyway" and waiting for M17's signing.

## Acceptance

Filled in when the run finishes, one row per rule above, pass or the test
that says otherwise.
