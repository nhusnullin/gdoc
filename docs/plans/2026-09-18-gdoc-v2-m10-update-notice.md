# gdoc v2 Milestone 10: the update notice

2026-09-18. The binary notices a newer release by itself and says so, once a
day, without slowing any run. The plugin follows the stable number, and the hub
hands colleagues the marketplace with auto-update on.

## Principles

Serves: 1, it runs on someone else's machine. A colleague who never reads the
releases page still hears about a release, from the tool itself, in the
session they already opened. Serves 4, every word costs attention: the notice
is one line, printed once a day, naming the version and the one command that
installs it, and it is absent when there is nothing to say.

Strains: 3, in one bounded place, and the bound is the whole of this plan's
design. The 2026-09-16 decision refused any check that a person had not typed,
and this milestone reverses one clause of it: `help` reaches GitHub on its own.
What keeps the strain small is that nothing moves. The check reads one listing,
writes one file of gdoc's own, replaces no binary and touches no document. It
runs in one command, at most once in 24 hours, under a two-second ceiling, on a
policy that admits no document id and carries no credential. A failed check is
written down too, so a network that refuses GitHub costs two seconds a day and
not two seconds a run. Every other command is exactly as fast and as offline
from GitHub as it was.

## Overview

| Piece | Today | After |
|---|---|---|
| the check | `gdoc update --check`, when a person types it | the same, and `help` runs one by itself when its stamp is missing or older than 24 hours |
| the stamp | none, by decision | `update-check.json` beside `oauth-token.json`, written through `internal/atomicfile`, holding when gdoc last asked and what it heard |
| the notice | none | facts under `update` in help's object; one line on stderr for a person, only when a newer stable of the same major exists |
| a checkout build | never reaches GitHub | still never reaches GitHub: no version, nothing to compare, no check |
| the skills | stop on a binary older than `needs` | the same, and one line when `latest_stable` is ahead of `installed`, then carry on |
| `needs` | any `x.y.z` | a stable `x.y.0`, pinned by a test |
| `plugin.json` | bumped by `make tag` and by the nightly | bumped by `make tag` only; the plugin carries a stable number and nothing else |
| the release check | every tag must match `plugin.json` | every `x.y.0` tag must match `plugin.json`; a nightly tag is not checked against it |
| skills install | `/plugin marketplace add` and `/plugin install` by hand | the hub's `.claude/settings.json` declares the marketplace with `autoUpdate: true` and turns the plugin on; the two `/plugin` lines stay as the route outside the hub |
| README | "Nothing updates on its own" | gdoc tells you about a release and installs it when you type `gdoc update` |

## Decisions Nail took, 2026-09-18

Taken in the brainstorm that produced this plan and written into DECISIONS.md.
Each is a decision and not a refactor. A task that finds one wrong stops and
says so.

1. **The binary checks by itself, in `help`, once a day.** This reverses the
   "no check, no stamp file" clause of the 2026-09-16 decision and nothing
   else in it. `help` is the one place, because it is the first call of every
   skill session, it already carries `version`, and no document is open in
   front of it. `read`, `publish` and every other command stay exactly as
   they are.
2. **The notice costs no run anything.** Printing it is one read of one small
   file. The fetch behind it runs only when that file is missing or older than
   24 hours, under a ceiling of two seconds instead of the listing's five, and
   a failed fetch is written to the file too, so a machine that cannot reach
   GitHub pays once a day.
3. **Facts only in the object, one line for a person.** The object under
   `update` carries `installed`, `latest_stable`, `latest_nightly`,
   `checked_at` and `error`, and judges nothing. The line on stderr, beside
   the help prose, names the newer stable and `gdoc update`, and `--major`
   when the difference is a major. Stdout stays one JSON object.
4. **A checkout build never checks.** No `version` means nothing to compare,
   so `make build` binaries and the test suite never reach GitHub from `help`.
5. **The skills say one line about the binary and carry on.** They read the
   facts `help` printed and compare three numbers, which they already do for
   `needs`. `needs` stays the only hard gate, and it names a stable `x.y.0`,
   so a skill never sends a colleague after a release `gdoc update` cannot
   install.
6. **There is no nightly channel for skills.** Nail runs the symlinked
   checkout, which is ahead of nightly, and nobody else wants nightly skills.
   The nightly therefore stops bumping `plugin.json`; the plugin's version
   moves on `make tag` only, and the release check that the plugin carries the
   tag applies to `x.y.0` tags. This is Claude Code's own update mechanism,
   the version pin: a colleague receives the plugin when the string changes.
   No `stable` branch, because one pin serves one channel and one channel is
   all there is.
7. **The plugin nudge in the skill is not built.** It was for people with
   auto-update off, and the hub turns it on for everyone.
8. **The hub declares the marketplace.** Its committed `.claude/settings.json`
   carries `extraKnownMarketplaces.gdoc` with `autoUpdate: true` and
   `enabledPlugins["altery@gdoc"]`. Claude Code copies that flag into its own
   `known_marketplaces.json` on startup, so a colleague gets marketplace,
   plugin and auto-update after one trust prompt. On Nail's machine the hub's
   `settings.local.json` turns `altery@gdoc` off, so the plugin copy never
   loads beside the symlinks.
9. **An off switch for the check is deferred.** A stamped failure already
   bounds the cost. The backlog holds the item with its reason.

## Context (from discovery)

- **`cmdHelp(words, errOut)`** in `go/cmd/gdoc/help.go` reaches no network and
  takes no context. `dispatch` calls it for `help`, for `--help` anywhere on
  the line, and the command table's `help` row calls it through `run`. Bare
  `gdoc` prints the prose and fails without calling it.
- **`var version = "dev"`** in `main.go`, and `releaseVersion()` returns the
  empty string for it. The whole test suite runs as `dev` unless a test sets
  `version`, which `installedAt` in `update_test.go` does. Decision 4 is what
  keeps every existing `help` test off the network.
- **`openPlain`** is a variable in `update.go`, and `stubPlain` in
  `update_test.go` stands in for the wire, recording the context it was
  handed. `TestTheUpdateRunReachesTheReleasesAndNothingElse` captures the
  policy the run opened and judges URLs against it.
- **`gapi.Plain.GetJSON`** wraps the caller's context in `ListingTimeout`, five
  seconds. A parent context with a two-second deadline is honoured, because
  the earlier deadline wins, so the ceiling needs no change in `gapi`.
- **`runUpdate`** already does the whole read: policy, `AllowUpdateFrom`,
  `GetJSON` on `releasesURL`, `update.Choose` per channel. The check is that
  read without the decision, so it shares the constants and the reach.
- **`TestNothingChecksForUpdatesUnasked`** in `update_test.go` walks every
  non-test file in `cmd/gdoc` and allows `internal/update` in `update.go`
  alone, with `cmdUpdate` named once in `commands.go`.
- **`TestNoSkillRunsUpdateOnItsOwn`** reads the call lines of every SKILL.md
  and fails on one whose first word is `update`. It stays: saying is not
  running.
- **`config.Dir()`** is the config folder, `GDOC_CONFIG_DIR` overrides it, and
  every `cmd/gdoc` test sets that to a temp dir. **`atomicfile.Replace`** is
  the one room that replaces a file's contents.
- **Claude Code**, read from its docs and its own bundle on 2026-09-18: a
  plugin with `version` in `plugin.json` is pinned to that string and a user
  receives an update only when it changes; a git marketplace without a pin
  updates on the commit SHA; the update check runs after startup with a
  random delay of up to ten minutes and prompts `/reload-plugins`;
  `extraKnownMarketplaces[name].autoUpdate` in user or project settings is
  copied into `known_marketplaces.json` on startup, and managed settings or
  the `--settings` flag lock it; there is no CLI flag for it on
  `marketplace add`. The `{name}--v{version}` tags `claude plugin tag` cuts
  serve plugin-to-plugin dependency ranges and nothing a person installing
  `altery@gdoc` reads.
- **The hub** (`nhusnullin/intelligence-hub`) already commits
  `.claude/settings.json` with an `enabledPlugins` block and ignores
  `settings.local.json`.
- **`nightly.yml`** bumps `plugin.json` with `jq` because `release.yml`'s step
  "the plugin version is the tag" refuses a tag the plugin does not carry.
  `make tag` does the same with `sed` for `x.y.0`.
- **SPEC.md** states the on-demand rule in three places: the guard paragraph
  ("`gdoc update` is what opens it, and nothing else does"), "Install and
  update" ("No check when a session starts, no scheduler, no stamp file") and
  the Never list. CLAUDE.md's fifth-grant invariant says `gdoc update` is the
  only caller. `cmd/gdoc/doc.go` has a section titled "The update runs when
  it is typed, and at no other moment". The user README says "Nothing updates
  on its own" and "never updates itself"; the second stays true.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it
  to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own
  commit, and each ends with the test gate as its last checkbox.
- **CRITICAL: every task MUST include new or updated tests**, success and
  failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** `make test`
  runs `-race`; keep it there.
- **CRITICAL: only `help` and `update` reach `internal/update`.** The rewritten
  `TestNothingChecksForUpdatesUnasked` allows two files and no third.
- **CRITICAL: no other command gets slower or reaches GitHub.** A test runs
  `read` with a stale stamp and a reach that fails the test if called.
- **CRITICAL: a fresh stamp makes no request.** A test runs `help` with a
  stamp younger than 24 hours and a reach that fails the test if called.
- **CRITICAL: the ceiling is two seconds and the interval is 24 hours**, both
  literals in their tests, never read from the constants they check.
- **CRITICAL: a checkout build never checks.** A test runs `help` as `dev`
  with a stale stamp and a reach that fails the test if called.
- **CRITICAL: the guard's Google rules do not move**, and the policy `help`
  opens admits no document. Every existing guard test stays green without an
  assertion changed.
- **CRITICAL: no bearer to GitHub.** The check goes out on `gapi.Plain`, which
  holds none.
- **CRITICAL: one object on stdout, always.** The notice for a person goes to
  stderr, where the help prose already goes.
- **CRITICAL: facts only in Go.** The object says what is installed and what is
  published. Nothing in it says "available", "behind" or "should".
- **CRITICAL: no `os/exec` anywhere, tests included.**
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces. Plain English.

## Testing Strategy

- **Unit, `internal/lastcheck`**: a written stamp reads back byte for byte; a
  missing file, a corrupt file and a file older than the interval are each
  stale and each named; a file younger than the interval is fresh; the
  interval is `24h` as a literal; a write goes through `atomicfile` and
  leaves no temp file behind.
- **Unit, `cmd/gdoc`**: `help` with a stale stamp makes exactly one request,
  on a policy that admits the listing and refuses a document, under a context
  whose deadline is at most two seconds from now, and writes the stamp; `help`
  with a fresh stamp makes none; `help` as `dev` makes none; an unreachable
  GitHub is `ok: true`, writes the stamp with `error`, and carries a warning
  naming the cause; a stamp directory that cannot be written is `ok: true`
  with a warning and no second attempt in the same run; the stderr line
  appears for a newer stable of the same major, names `--major` for a higher
  major, and is absent when up to date; `read` never reaches the check.
- **Unit, `cmd/gdoc` skills walk**: every `needs` line is a stable `x.y.0`;
  `TestNoSkillRunsUpdateOnItsOwn` unchanged and green.
- **Boundary**: `nightly.yml` names no `plugin.json`; `release.yml`'s plugin
  check is gated on an `x.y.0` tag; `allowedModules` unchanged;
  `TestNetHTTPStaysInItsRooms` green; `TestNothingRunsAnExternalProgram`
  green; CLAUDE.md under its ceiling and its task map naming files that exist.
- **By hand, Task 6**: the hub's settings on this machine, a fresh clone in a
  scratch directory to see the trust prompt and the plugin arrive, and
  `bin/gdoc help` from a tagged build with the stamp removed, timed.
- Coverage standard: every exported function under `go/internal/` has a test;
  the new package at or above 80%.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- `cd go && go vet ./...`
- `make build && rm -f ~/.config/gdoc-agent/update-check.json && time bin/gdoc help >/dev/null`
  shows no stamp written and no delay, because a checkout build is `dev`
- `GDOC_CONFIG_DIR=$(mktemp -d) && time gdoc help >/dev/null && cat "$GDOC_CONFIG_DIR/update-check.json"`
  with a release binary on PATH shows one stamp and under three seconds
- `grep -c plugin.json .github/workflows/nightly.yml` prints 0

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

**The stamp** is `go/internal/lastcheck`, a package over one file:
`update-check.json` in `config.Dir()`. It holds `checked_at` as RFC 3339 in
UTC, `latest_stable`, `latest_nightly` and `error`. `Read` returns the stamp
and whether it is stale, where stale is missing, unreadable, malformed or
older than `Interval`, 24 hours, against a clock the caller hands in. `Write`
goes through `atomicfile.Replace`. It holds no wire and decides nothing about
versions; `internal/update` stays the arithmetic and stays stateless.

**The check** is `go/cmd/gdoc/notice.go`. `cmdHelp` gains a context and calls
`notice(ctx, errOut)` once, before the prose. When `releaseVersion()` is empty
it returns nothing. Otherwise it reads the stamp; when stale it opens a policy
with `AllowUpdateFrom(updateRepo)`, a context with a two-second deadline, reads
`releasesURL` through `openPlain`, runs `update.Choose` for both channels, and
writes the stamp, with `error` set when the read or the choice failed. Then,
fresh or just refreshed, it returns the facts for the object and the warnings
for the envelope, and prints one line to stderr when `update.Decide` over the
stamp's versions with no flags says `updated` or `major_available`. A stamp
that cannot be written is one warning and no retry.

**The object** gains `update` under `data`, omitted for a checkout build.

**The skills** gain one paragraph under Setup, after the `needs` check: read
`update.latest_stable` when it is there, compare it with `version` as three
numbers, and when it is ahead say once that a newer gdoc is published and
`gdoc update` installs it, then carry on. Each `needs` becomes a stable
`x.y.0`; today all three say `v2.0.0`, which already is one.

**The workflows**: `nightly.yml` loses its `jq` step and commits nothing; it
tags `main` as it stands and calls `release.yml`. `release.yml`'s plugin check
runs when the tag matches `^v[0-9]+\.[0-9]+\.0$`. `make tag` is unchanged.

**The hub**: two keys in the committed settings file, by hand, in the other
repository.

## Technical Details

**The stamp file**:

```json
{"checked_at":"2026-09-18T07:12:03Z","latest_stable":"v2.3.0","latest_nightly":"v2.3.4"}
```

A failed check keeps the last versions it heard and adds the cause:

```json
{"checked_at":"2026-09-19T07:00:11Z","latest_stable":"v2.3.0","latest_nightly":"v2.3.4","error":"the request to https://api.github.com/repos/nhusnullin/gdoc/releases failed: context deadline exceeded"}
```

**The help object**, a release binary:

```json
{"ok":true,"version":"v2.2.0","data":{"commands":[...],"update":{"installed":"v2.2.0","latest_stable":"v2.3.0","latest_nightly":"v2.3.4","checked_at":"2026-09-18T07:12:03Z"}}}
```

**The stderr line**, printed before the prose and only when there is one:

```
gdoc v2.3.0 is published and this is v2.2.0. `gdoc update` installs it.
gdoc v3.0.0 is published and this is v2.2.0. It is a major release: `gdoc update --major` installs it.
```

**Constants**: `lastcheck.Interval = 24 * time.Hour`;
`checkTimeout = 2 * time.Second` in `notice.go`.

**The hub's settings**, added to the existing `.claude/settings.json`:

```json
"extraKnownMarketplaces": {
  "gdoc": {
    "source": { "source": "github", "repo": "nhusnullin/gdoc" },
    "autoUpdate": true
  }
},
"enabledPlugins": { "altery@gdoc": true }
```

## Implementation Steps

### Task 1: the stamp

**Files:**
- Create: `go/internal/lastcheck/doc.go`, `lastcheck.go`, `lastcheck_test.go`
- Modify: `go/internal/config/config.go`, `config_test.go` (the path)

- [x] Test first, `TestAWrittenStampReadsBackByteForByte`: write a stamp
      into a temp config dir, read the file, compare the bytes with a literal.
- [x] Test, `TestMissingCorruptAndOldAreEachStaleAndNamed`: no file, a file
      holding `{`, and a file whose `checked_at` is 25 hours before the clock
      handed in are each stale, and each reason names what was wrong. A file
      23 hours old is fresh.
- [x] Test, `TestTheIntervalIsADay`: the literal `24h`, never the constant.
- [x] Test, `TestAWriteLeavesNoTempFileBehind`: after `Write` the config dir
      holds one file.
- [x] `config.LastCheckPath()` beside `TokenPath()`, with its case in
      `TestEnvOverrideWins`.
- [x] `lastcheck.Read(path, now)` and `lastcheck.Write(path, stamp)`, the
      write through `atomicfile.Replace` at mode 0600 like the token. Strict
      decoding, refused by name, as every input is.
- [x] `doc.go` opens `Package lastcheck` and names the four tests.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): the stamp of the last release check"`

### Task 2: help checks once a day and prints the facts

**Files:**
- Create: `go/cmd/gdoc/notice.go`, `notice_test.go`
- Modify: `go/cmd/gdoc/help.go`, `help_test.go`, `main.go`, `commands.go`
- Modify: `go/cmd/gdoc/update_test.go` (`TestNothingChecksForUpdatesUnasked`)
- Modify: `go/cmd/gdoc/doc.go`

- [x] Test first, `TestAStaleStampMakesHelpAskOnce`: `version` set to
      `v2.2.0`, no stamp, `stubPlain` listing `v2.3.0` and `v2.3.4`; `help`
      makes one request, the stamp lands with both versions, the object's
      `update` carries the four facts, and stderr opens with the line naming
      `v2.3.0` and `gdoc update`.
- [x] Test, `TestAFreshStampMakesNoRequest`: a stamp 23 hours old and a reach
      whose `GetJSON` fails the test; the object still carries `update` from
      the stamp.
- [x] Test, `TestACheckoutBuildNeverChecks`: `version` left as `dev`, no stamp,
      the same failing reach; no request, no stamp written, no `update` key.
- [x] Test, `TestTheCheckIsBoundedByTwoSeconds`: the context the reach saw
      carries a deadline at most two seconds from the test's clock, literal.
- [x] Test, `TestAnUnreachableGitHubIsStampedAndHelpStillAnswers`: the four
      causes `TestAnUnreachableGitHubIsAnAnswerAndNotAFailure` lists; each is
      `ok: true`, writes the stamp with `error`, and carries a warning naming
      the cause. A second `help` in the same test makes no request.
- [x] Test, `TestAStampThatCannotBeWrittenIsOneWarning`: config dir made
      read-only; `ok: true`, one warning, one request and not two.
- [x] Test, `TestTheHelpCheckOpensThePolicyTheUpdateOpens`: the same judge as
      `TestTheUpdateRunReachesTheReleasesAndNothingElse`, against the policy
      `help` opened.
- [x] Test, `TestTheLineNamesMajorForAMajor`: latest stable `v3.0.0`, the
      line names `gdoc update --major`; latest stable equal to installed, no
      line at all.
- [x] Test, `TestReadNeverReachesTheCheck`: `read` with a stale stamp and the
      failing reach, against the existing fake session; no request.
- [x] `TestNothingChecksForUpdatesUnasked` rewritten: `update.go` and
      `notice.go` import `internal/update`, no third file does, and the table
      still names `cmdUpdate` once.
- [x] `notice.go` as the Solution Overview says. `cmdHelp` takes `ctx`; the
      three call sites hand it through.
- [x] `doc.go`: the section "The update runs when it is typed, and at no other
      moment" becomes "The update runs when it is typed; help asks once a
      day", naming the tests above and the two files that may reach the
      updater.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): help notices a newer release once a day"`

### Task 3: the skills say one line and carry on

**Files:**
- Modify: `skills/gdoc-review/SKILL.md`, `skills/gdoc-publish/SKILL.md`,
  `skills/gdoc-restyle/SKILL.md`
- Modify: `go/cmd/gdoc/skills_test.go`

- [ ] Test first, `TestEverySkillNeedsAStableRelease`: every `needs` line
      parses and its third number is 0; a fixture with `needs: v2.1.3` is
      caught and named.
- [ ] Test, `TestNoSkillRunsUpdateOnItsOwn` stays green after the wording
      lands, which proves the new paragraph names the command in prose and
      never on a call line.
- [ ] The paragraph under Setup in each skill, after the `needs` sentence:
      read `update.latest_stable` when the object carries it, compare with
      `version` as three numbers, and when it is ahead say once that a newer
      gdoc is published and that `gdoc update` installs it, then carry on. No
      `update` key is a checkout build, and the skill says nothing about it.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(skill): the skills mention a newer gdoc once and carry on"`

### Task 4: the plugin carries a stable number only

**Files:**
- Modify: `.github/workflows/nightly.yml`, `.github/workflows/release.yml`
- Modify: `go/boundary/workflows_test.go`, `doc.go`

- [ ] Test first, `TestOnlyMakeTagMovesThePluginVersion`: `nightly.yml`
      contains no `plugin.json` and no `git commit`; `release.yml`'s plugin
      check step carries an `if` on a tag matching `\.0$`.
- [ ] `nightly.yml`: the bump step becomes a tag step, tagging `main` as it
      stands and pushing the tag; the comment says why the plugin does not
      follow.
- [ ] `release.yml`: the check "the plugin version is the tag" runs for
      `x.y.0` tags only, with one sentence saying a nightly is a binary
      release and never a plugin release.
- [ ] `boundary/doc.go` names the test.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "ci: the nightly tags the binary and leaves the plugin version alone"`

### Task 5: the documents that stated the old rule

**Files:**
- Modify: `docs/v2/SPEC.md`, `docs/v2/DECISIONS.md`, `docs/v2/PLAN.md`
- Modify: `CLAUDE.md`
- Modify: `release/README.md`
- Modify: `go/internal/update/doc.go`
- Create: `docs/backlog/update-check-has-no-off-switch.md`

- [ ] Test first: `TestCLAUDEmdIsUnderTheCeiling` and
      `TestTheTaskMapNamesFilesThatExist` stay green, and the task map gains
      the row "a release notice, the stamp" naming `go/internal/lastcheck/doc.go`
      and `go/cmd/gdoc/doc.go`.
- [ ] SPEC.md: the guard paragraph says `gdoc update` and the daily check in
      `help` open the grant; "Install and update" replaces "No check when a
      session starts, no scheduler, no stamp file" with the stamp, the
      interval, the ceiling and the facts, and says the plugin carries a stable
      number; the Never list's "Never update unasked" becomes "Never install
      unasked", with the check named as the one thing that runs by itself.
- [ ] CLAUDE.md: the fifth-grant invariant names two callers; the skills
      invariant says the hub declares the marketplace.
- [ ] `release/README.md`: "The skills" opens with the hub route and one trust
      prompt, keeps the `/plugin` lines for a machine outside the hub, and
      keeps the two fallbacks; "Updates" says gdoc tells you once a day and
      installs when you type `gdoc update`; "What gdoc never does" keeps
      "never updates itself".
- [ ] `internal/update/doc.go`: one sentence that the stamp lives in
      `internal/lastcheck` and this package still holds no state.
- [ ] DECISIONS.md: the 2026-09-18 entry written with this plan gains what
      Task 2 measured, the time a cold `help` takes on this machine.
- [ ] PLAN.md: the standing fact about the plugin number.
- [ ] The backlog item: why there is no off switch, what it would look like,
      and what would make it worth adding.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "docs(v2): the release notice, and the plugin on the stable number"`

### Task 6: the hub, by hand

Not a ralphex task. Nail runs it in `nhusnullin/intelligence-hub` after the
milestone lands on `main` and a tagged release carries it.

- [ ] The two keys from Technical Details into the hub's `.claude/settings.json`,
      committed with `chore: the gdoc marketplace, auto-updated`.
- [ ] `"altery@gdoc": false` in the hub's `.claude/settings.local.json` on
      this machine.
- [ ] A fresh clone of the hub in a scratch directory, opened in Claude Code:
      the trust prompt appears once, `/plugin` lists `altery@gdoc` installed
      and the marketplace with auto-update on.
- [ ] `release/README.md` re-read against what the clone showed, and corrected
      in this repository if it differs.
