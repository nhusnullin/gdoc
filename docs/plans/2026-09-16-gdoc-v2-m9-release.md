# gdoc v2 Milestone 9: the release, the nightly, and the updater

2026-09-16.

## Principles

Serves: 1, it runs on someone else's machine. The release is one zip a
colleague unpacks, one installer they run, and one binary that keeps itself
and its skills current from then on, with nothing else on the machine. Serves
4, every word costs attention: a colleague learns gdoc from a hundred-line
README and from `gdoc help`, and an update is one line in a session, never a
question.

Strains: 3, uncertainty never resolves toward the destructive answer, in one
place. A minor version updates the binary with no question asked. That is
Nail's decision, and the plan bounds it: every download is verified against a
checksum before a byte is replaced, the previous binary is kept and one
command swaps it back, the object and the session both say it happened, and a
major version never moves without a person running the command.

## Overview

gdoc goes to the team. Colleagues are on macOS and Windows, all with Claude
Code, some with access to this repository and some without, and they will try
the whole lifecycle: review, publish, restyle. Feedback comes as a GitHub
Issue or a message to Nail. macOS ships first; Windows follows under its own
tag when a colleague has run its checklist.

| Piece | Today | After |
|---|---|---|
| version | none; no tags | `x.y.z` baked in from the tag, in `help`, `auth status` and every envelope |
| release | `make dist` by hand | a tag builds it in CI, zips it, and publishes it as a GitHub Release of this repository, which is public |
| nightly | none | main moved since the last tag: CI tags `x.y.(z+1)` at 02:00 UTC and releases it |
| install | `install.sh` in a checkout | one line, `curl -fsSL .../release/install.sh \| bash`, which fetches the latest release zip and installs it; the same script runs from an unpacked zip. Asks global or local, strips quarantine |
| update | by hand | `gdoc update`, run by every skill before its first call, once a day: minor updates apply, major ones are reported, nightly ones need the channel |
| skills | say "Nail", point at Nail's checkout | say "you", carry no path, and travel in the zip with a version marker |
| page breaks | none in the docx route | before "Version Control" and before "Contents", in both routes, from one block in the house style |
| feedback | none | an issue template asking for version, command, object and expectation |

## Decisions Nail took, 2026-09-16

Taken in the brainstorm that produced this plan, and written into
DECISIONS.md the same day. Each is a decision and not a refactor. A task that
finds one wrong stops and says so.

1. **Releases are GitHub Releases of this repository, which is public.**
   Nail made `nhusnullin/gdoc` public on 2026-09-16 after the assessment in
   DECISIONS.md: no secret beyond the Internal OAuth client, no document,
   and the client secret left the source the same day, injected at build
   time from `GDOC_OAUTH_CLIENT_SECRET`. So there is no second repository
   and no second token: colleagues download from this repository's releases
   page, the updater fetches from it with no credential, and the release
   workflow builds with the secret from this repository's secrets.
2. **Versions are `x.y.z`.** Nail tags `x.y.0` by hand. The nightly tags
   `x.y.(z+1)`. The number is the channel: `z == 0` is stable, `z > 0` is
   nightly. The first tag is `v2.0.0`.
3. **The update policy.** Same `x`, higher `y`: update, no question, and say
   so. Higher `x`: report only, with the command to run. Higher `z`: only on
   `channel: nightly`, off by default. The session prints one line when the
   binary changed under it. That line is the one place this plan does not do
   "silent": a tool changing with no trace is what principle 3 exists to
   prevent, and one line costs nothing.
4. **Skills are linked in a checkout and copied from a release.** The
   2026-08-14 decision gains that clause. A release copy carries a `.release`
   marker naming its version, so the skills and the binary on a machine always
   came from one zip.
5. **The installer asks global or local**, and nothing else. Global is
   `~/.claude/skills`; local is `.claude/skills` in the folder it is run from,
   one hub. `--skills global|local` answers without the prompt. The installer
   is a shell script a person runs, so it may ask; the binary still never
   does.
6. **The guard gains one read-only door for updates.** `AllowUpdateFrom`
   names one repository for one run and admits GET on
   `api.github.com`, `github.com` and the asset host for that repository's
   releases, with no Authorization header, because the only bearer gdoc holds
   is Google's. Opened by `gdoc update` alone.
7. **Page breaks are a block in the house style**, `- block: page_break`,
   before the `version_control` label and after the `document_classification`
   table, read by the docx renderer and by the prelude. One layout, two
   writers. The prelude's own cover page break becomes that block. The two
   rows the drift gate reports against the master join `drift.Known` with
   Nail's name and this date.
8. **Builds are trimmed and stripped.** `-trimpath` and `-ldflags "-s -w"` in
   `build` and `dist`: Nail's home path is not shipped, and a tagged build is
   the same bytes on every machine.
9. **Windows ships when its checklist has run.** `release/platforms` lists
   `darwin-arm64` and `darwin-amd64` for `v2.0.0`. Adding the Windows line is
   its own commit after a colleague pastes the checklist back.

## Context (from discovery)

- **The guard judges every host in one switch**, `Policy.Judge` in
  `go/internal/guard/policy.go`, and refuses any host but the three Google
  ones by name. The four grants beside the set, `AllowReject`, `AllowMarker`,
  `AllowCopy` and `AllowCreateIn`, are per-run and name one object each.
  `AllowUpdateFrom` is the fifth, in the same shape.
- **Only four rooms may import `net/http`**, held by
  `TestNetHTTPStaysInItsRooms` in `go/boundary`, and only `internal/gapi`
  builds a request. The updater's fetches are built there, on a client from
  `guard.NewClient` with no session, so no bearer is ever set. The allowlist
  does not widen.
- **`internal/config` knows two paths**, the dir and `oauth-token.json`.
  `UpdatePath` is the third, `update.json`, read strictly like the token file.
- **`internal/atomicfile.Replace`** is the one room that replaces a file's
  contents. It writes the config and the skill files. The binary itself is
  replaced by rename, because a running executable cannot be written through,
  and the rename works on both platforms.
- **The house style's front-matter blocks** are `cover`, `label`, `table`,
  `blank`, `legend` and `toc`, decoded in `go/internal/house/house.go` and
  walked by `go/internal/render/front.go` and
  `go/internal/prelude/frontmatter.go`. The renderer already knows
  `PageBreak` on a paragraph; the prelude already has `pageBreak()`, called
  once from `cover.go:43`, pinned by `TestTheCoverEndsWithAPageBreak`.
- **The master template has one page break**, before its first heading, so
  the two new ones are two rows in `drift.Known`, `go/internal/drift/compare.go`.
- **`make dist`** builds three binaries with `CGO_ENABLED=0` and nothing else.
  The Go workflow runs gofmt, vet, the raced suite and `make dist` on every
  push.
- **A tag pushed with `GITHUB_TOKEN` triggers no other workflow.** The
  nightly cannot rely on the tag event; it calls the release workflow as a
  reusable one. `GITHUB_TOKEN` can create a release on this repository, so no
  second token is needed.
- **The client secret is a repository secret**, `GDOC_OAUTH_CLIENT_SECRET`,
  and the release workflow passes it to `make dist`. A build without it
  cannot sign anyone in, so a release built without the secret is a failed
  release, and the workflow refuses to publish one.
- **The repository is public**, so `raw.githubusercontent.com` serves
  `release/install.sh` to anyone and the releases API answers without a
  token.
- **The zip on macOS is quarantined** when it arrives through a browser, and
  an unsigned binary then refuses to run. The installer strips the attribute.
  A binary the updater downloads itself is not quarantined.
- **`go/version` is not semver.** Three integers and a comparison is a page
  of code with no dependency.
- **The skills' Setup** runs `gdoc help <command>` before the first call. The
  update runs right before that.
- **`build` has no network at all**, and stays that way. Nothing in it checks.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it
  to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own
  commit, and each ends with the test gate as its last checkbox.
- **CRITICAL: every task MUST include new or updated tests**, success and
  failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** `make test`
  runs `-race`; keep it there.
- **CRITICAL: the guard's Google rules do not move.** A policy without
  `AllowUpdateFrom` refuses every GitHub host exactly as today, and every
  existing guard test stays green without an assertion changed.
- **CRITICAL: no bearer to GitHub.** A test hands the recorded update request
  to the guard and asserts it carries no Authorization header.
- **CRITICAL: nothing is replaced before it is verified.** The checksum check
  comes before the rename, and a failed check leaves the old binary and the
  old skills exactly as they were.
- **CRITICAL: no `os/exec` anywhere, tests included.** The updater cannot run
  the new binary to verify it; the next run's envelope is the proof, and
  `verified` says whether the file on disk hashes to what was promised.
- **CRITICAL: one object on stdout, always.** `update` reports through
  `internal/emit` like every command.
- **CRITICAL: facts only in Go.** `update` says what it found and what it did.
  Whether to run `update --major` is the person's.
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces. Plain English.

## Testing Strategy

- **Unit, `cmd/gdoc`**: the version in `help`, `auth status` and the
  envelope; `update`'s arguments, its policy on every version pair, the
  throttle, `--now`, `--check`, `--major`, `--channel`, `--rollback`.
- **Unit, `internal/guard`**: the update grant carries the three hosts with
  GET and nothing else; a policy without it refuses them; no Authorization
  header reaches GitHub; the Google rules unchanged.
- **Unit, `internal/update`**: version parsing and comparison against
  literals; release selection per channel and platform from a recorded API
  answer; checksum verification refusing a wrong file; the replace sequence
  on a temp dir; the skill folder rules, marker present, marker absent,
  symlink.
- **Unit, `internal/house`, `render`, `prelude`**: the `page_break` block
  decoded, rendered as `w:pageBreakBefore`, proposed as `insertPageBreak`;
  literals throughout.
- **Unit, `internal/drift`**: the two new rows in `Known`, and the gate green
  against the master.
- **Boundary**: `allowedModules` unchanged; `TestNetHTTPStaysInItsRooms`
  green with the fetches in `gapi`; `TestNothingRunsAnExternalProgram` green.
- **Skills**: `skills_test.go` gains two checks: no SKILL.md says "Nail", and
  none names a path under a home directory.
- **Live, opt-in, Task 16**: an rc tag through the whole pipeline, an install
  in a scratch home, and an update from rc1 to rc2 on this machine.
- Coverage standard: every exported function under `go/internal/` has a test;
  the new package at or above 80%.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- `cd go && go vet ./...`
- `make build && bin/gdoc help 2>&1 >/dev/null | head -1` shows the version
- `make dist` and `strings bin/gdoc-darwin-arm64 | grep -c nailkhusnullin`
  prints 0

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

**The version** is `var version = "dev"` in `go/cmd/gdoc/main.go`, set by
`-ldflags "-X main.version=$(git describe --tags --always)"` in both Make
targets. It goes into `emit.Result` as `version`, omitted when `dev`, and
into the help prose's first line and `auth status`'s data.

**The release path** is one workflow, `release.yml`, callable two ways:
`push` of a tag `v*`, and `workflow_call` with a tag input. It runs the same
four checks the Go workflow runs, then `make dist`, then for each line of
`release/platforms` packs `gdoc-<tag>-<platform>.zip` holding the binary,
`skills/`, `install.sh`, `README.md` and `example/`, writes
`SHA256SUMS-<tag>`, and creates the GitHub Release here. The nightly, `nightly.yml`,
runs on a cron, reads the last tag, and if main moved, tags `x.y.(z+1)` and
calls `release.yml` with it.

**The updater** is `go/internal/update`, pure where it can be: parse and
compare versions, choose a release for a channel and a platform from the API
answer, verify a zip against its checksum line, plan the replacement. The
command in `cmd/gdoc/update.go` reads `update.json`, opens a policy with
`AllowUpdateFrom`, fetches through `gapi`, and carries the plan out: the zip
to a temp dir under the config dir, the binary to `<path>.new`, checksum, old
to `<path>.previous`, new to `<path>`, then every skill folder at a recorded
location that carries the `.release` marker. `--rollback` swaps `.previous`
back. `--check` reports without touching anything. Every run is throttled to
one check a day by `last_check` in `update.json` unless `--now`.

**The installer** is one script with two entrances. Run from an unpacked zip
it installs what is beside it. Run from `curl | bash` it asks the releases
API for the latest stable tag, downloads the zip for this machine's platform
and its checksum file, verifies, unpacks to a temp dir and installs from
there. Either way it copies the binary, asks global or local, copies the skills with their
markers, strips quarantine, runs `gdoc update --set-skills <where>` and
`gdoc completion zsh --out` into the config dir, and ends with `gdoc auth
status`.

## Technical Details

**`update.json`**, read strictly, unknown keys refused:

```json
{
  "source": "nhusnullin/gdoc",
  "channel": "stable",
  "skills": ["global", "/Users/x/hub"],
  "last_check": "2026-09-16T09:00:00Z"
}
```

`source` is fixed by the installer and never changes on its own. `channel` is
`stable` or `nightly`. `skills` records where copies live, so an update
replaces exactly the folders the installer wrote. The binary writes the file
through `atomicfile`.

**The update object**:

```json
{"ok":true,"version":"v2.0.0","data":{"installed":"v2.0.0","channel":"stable","latest":"v2.1.0","action":"updated","verified":true,"skills_replaced":["global"],"previous":"/Users/x/.local/bin/gdoc.previous"}}
```

`action` is one of `up_to_date`, `updated`, `major_available`,
`nightly_available`, `checked_recently`, `rolled_back`. A `major_available`
carries `run: "gdoc update --major"` beside it. `verified` is the checksum
of the file at its final path.

**The policy, as a table the test walks:**

| installed | latest stable | latest nightly | channel | action |
|---|---|---|---|---|
| 2.0.0 | 2.0.0 | 2.0.3 | stable | up_to_date |
| 2.0.0 | 2.1.0 | 2.1.2 | stable | updated to 2.1.0 |
| 2.0.0 | 3.0.0 | | stable | major_available |
| 2.0.0 | 2.0.0 | 2.0.3 | nightly | updated to 2.0.3 |
| 2.0.3 | 2.1.0 | 2.1.0 | nightly | updated to 2.1.0 |
| 2.1.0 | 2.0.0 | | stable | up_to_date, never down |

**The guard grant.** `AllowUpdateFrom("nhusnullin/gdoc")` admits:
`GET api.github.com/repos/nhusnullin/gdoc/releases`, `GET
github.com/nhusnullin/gdoc/releases/download/<tag>/<asset>`, and
`GET objects.githubusercontent.com/...` reached by the redirect from the
second, for the run's length. Any other method, path or repository on those
hosts is refused by name. A request carrying Authorization to any of them is
refused before it leaves.

**The `page_break` block.** In `house.yaml`:

```yaml
front_matter:
- block: cover
- block: page_break
- block: label
  ref: version_control
...
- block: table
  ref: document_classification
- block: page_break
- block: label
  ref: contents
- block: toc
```

The renderer emits it as an empty paragraph with `w:pageBreakBefore`, which
is how the first heading breaks today. The prelude emits `insertPageBreak`.
The cover's `trailing_blanks` drop to what the layout needs without pushing.

**Global constraints:**

- Exactly one JSON object on stdout, exit 0 if and only if `ok`. No prompting,
  no stdin. The installer prompts; the binary never does.
- Strict argument parsing. `update` takes no words and the flags `--check`,
  `--now`, `--major`, `--channel <stable|nightly>`, `--set-skills <where>`,
  `--rollback`; a combination that means two things is refused by name.
- Nothing under `go/` imports `os/exec`, tests included.
- `install.sh` never edits `.zshrc`. It prints the line.

## Implementation Steps

### Task 1: the version, trimmed and stripped

**Files:**
- Modify: `go/cmd/gdoc/main.go`, `help.go`, `main_test.go`, `help_test.go`
- Modify: `go/internal/emit/emit.go`, `emit_test.go`
- Modify: `Makefile`

- [ ] Test first, `TestTheVersionReachesTheEnvelopeAndTheHelp`: with
      `version` set in the test, every object carries `version`, the help
      prose opens with it, and `auth status` data carries it; with `dev` the
      field is absent and no existing test sees a change.
- [ ] `var version = "dev"` in `main.go`; `emit.Result` gains `Version string
      \`json:"version,omitempty"\``; `run` sets it.
- [ ] `Makefile`: `VERSION := $(shell git describe --tags --always --dirty)`,
      and both `build` and `dist` pass `-trimpath -ldflags "-s -w -X
      main.version=$(VERSION)"`.
- [ ] `make dist && strings bin/gdoc-darwin-arm64 | grep -c nailkhusnullin`
      prints 0. Record the three binary sizes here.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): the version in every envelope, and builds trimmed and stripped"`

### Task 2: the skills say "you"

**Files:**
- Modify: `skills/gdoc-review/SKILL.md`, `skills/gdoc-publish/SKILL.md`,
  `skills/gdoc-restyle/SKILL.md`
- Modify: `go/cmd/gdoc/skills_test.go`

- [ ] Test first, two checks added to the SKILL.md walk: no file contains the
      word `Nail`, and none contains `/Users/` or `~/src/`.
- [ ] "Nail" becomes "you" for the person at the keyboard and "a colleague" or
      "the reviewer" where the text means somebody else in the document. The
      descriptions trigger on what the person says: "Use when you are given a
      Google Doc link and want the marked comments handled".
- [ ] The review skill's `Spec:` line pointing into a checkout goes. Read the
      spec sections it names and confirm the skill already carries every
      sentence it needs; add the missing ones, not a path.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(skill): the skills address whoever is at the keyboard"`

### Task 3: the page_break block in the docx route

**Files:**
- Modify: `go/internal/house/house.yaml`, `house.go`, `house_test.go`
- Modify: `go/internal/render/front.go`, `front_test.go`
- Modify: `go/internal/drift/compare.go`, `compare_test.go`

- [ ] Test first, literals: `house.yaml` decodes a `page_break` block at the
      two positions, and a `page_break` with any other key is refused by name.
- [ ] Test, `TestTheFrontMatterBreaksBeforeVersionControlAndBeforeContents`:
      the rendered document has `w:pageBreakBefore` on exactly the two
      paragraphs, stated as literals, and the cover's trailing blanks are the
      new count.
- [ ] `house.go` decodes the kind; `front.go` renders it as the empty paragraph
      the first heading already uses.
- [ ] The offline drift gate reports two new differences; they join `Known`
      with the reason "Nail, 2026-09-16: page breaks before Version Control
      and before Contents, which the master pushes with blank lines". The gate
      is green.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): page breaks before Version Control and before Contents"`

### Task 4: the page_break block in the prelude

**Files:**
- Modify: `go/internal/prelude/frontmatter.go`, `cover.go`, `cover_test.go`,
  `frontmatter_test.go`, `readback.go`, `readback_test.go`, `doc.go`

- [ ] Test first, `TestTheFrontMatterEndsWithAPageBreak`: the requests carry
      an `insertPageBreak` after the classification table, and
      `TestTheCoverEndsWithAPageBreak` keeps passing with the break now coming
      from the block rather than from `cover.go:43`.
- [ ] `block()` handles `page_break`; the hard-coded call in `coverBlock` goes,
      so one layout has two writers and no third.
- [ ] The read-back counts the second break, and `Verify` is unchanged in what
      it asks.
- [ ] `doc.go` names the two tests.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): the prelude ends with a page break, from the same block"`

### Task 5: update.json

**Files:**
- Create: `go/internal/update/config.go`, `config_test.go`, `doc.go`
- Modify: `go/internal/config/config.go`, `config_test.go`

- [ ] Test first: the file read strictly, an unknown key refused by name, a
      channel other than the two refused, a missing file meaning "not
      installed from a release" rather than a failure, and a write through
      `atomicfile` byte-identical when nothing changed.
- [ ] `config.UpdatePath()` beside `TokenPath()`. `update.Config` with the four
      fields, `Load`, `Save`.
- [ ] `doc.go` opens `Package update` and names the tests.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): update.json, read strictly"`

### Task 6: the guard's update grant

**Files:**
- Modify: `go/internal/guard/policy.go`, `policy_test.go`, `doc.go`

- [ ] Test first, as an attack: a policy without the grant refuses
      `api.github.com` by name, as today. With `AllowUpdateFrom(repo)`: the
      releases listing carries, the download path for that repository
      carries, the asset host carries for GET; a POST, another repository,
      another path, and any Google request through the same policy are each
      refused by name.
- [ ] Test, `TestAnUpdateRequestCarriesNoBearer`: a request to any of the
      three hosts with an Authorization header is refused before it leaves.
- [ ] `AllowUpdateFrom` in the shape of `AllowCreateIn`: per-run, one
      repository, read-only. `Judge` gains the three hosts under it.
- [ ] `doc.go`'s "Two doors into the set, and four grants beside it" becomes
      five, with the reason and the tests.
- [ ] Every existing guard test passes without an assertion changed.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): one read-only guard door for updates"`

### Task 7: releases, versions and the policy

**Files:**
- Create: `go/internal/update/version.go`, `version_test.go`, `choose.go`,
  `choose_test.go`, `testdata/releases.json`
- Modify: `go/internal/gapi/` (one plain GET without bearer, on a guard
  client with no session), and its test

- [ ] Test first, literals: `Parse("v2.1.3")` and every malformed form
      refused; `Compare` on the table in Technical Details; `Choose` from the
      recorded releases answer picks the highest `z == 0` for stable and the
      highest overall for nightly, for one platform, and returns nothing when
      the platform's asset is missing.
- [ ] Test: the policy table, every row, as one table-driven test.
- [ ] `gapi.Get(url)` for a client with no session: no bearer, JSON or bytes
      back, and the guard judging it like every other request.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): choosing a release, and the update policy"`

### Task 8: download, verify, replace, roll back

**Files:**
- Create: `go/internal/update/apply.go`, `apply_test.go`, `skills.go`,
  `skills_test.go`

- [ ] Test first, on a temp dir: a zip and its checksum line verify; a wrong
      checksum refuses and leaves every file as it was; the replace sequence
      leaves `<path>` as the new binary and `<path>.previous` as the old; a
      second replace overwrites `.previous`; `Rollback` swaps back and refuses
      when there is no previous.
- [ ] Test, the skill folders: a folder with `.release` is replaced whole and
      the marker carries the new version; a folder without it is refused by
      name; a symlink is refused by name; a recorded location that no longer
      exists is a warning, not a failure.
- [ ] `Apply` and `Rollback`, `archive/zip`, `crypto/sha256`, `os.Rename`,
      `atomicfile` for the skill files. Windows: the same rename sequence, and
      a note in `doc.go` that it is unmeasured until the Windows checklist.
- [ ] `verified` is the hash of the file at its final path.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): the update applied, verified, and reversible"`

### Task 9: gdoc update, and the skills run it first

**Files:**
- Create: `go/cmd/gdoc/update.go`, `update_test.go`
- Modify: `go/cmd/gdoc/commands.go`, `doc.go`
- Modify: the three `skills/*/SKILL.md`

- [ ] Test first: strict arguments, every flag combination that means two
      things refused by name; `--check` touches nothing; the daily throttle
      answers `checked_recently` and `--now` overrides it; a machine with no
      `update.json` answers `not_installed_from_a_release` with `ok: true`;
      the envelope on each failure path; the object shape against literals.
- [ ] `--set-skills <global|path>` and `--channel` write the config and do no
      check. `--major` applies a major. `--rollback` calls `Rollback`.
- [ ] The three skills' Setup: `$GDOC update` right before the first `$GDOC
      help`, and the one line the skill prints when `action` is `updated`.
      `build` is never preceded by it.
- [ ] `doc.go`: a section on the update, why it is throttled, why it never
      runs before `build`, and the tests.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): gdoc update"`

### Task 10: the release installer

**Files:**
- Create: `release/install.sh`
- Create: `release/platforms`

- [ ] `release/platforms` holds `darwin-arm64` and `darwin-amd64`, one per
      line, with a comment saying Windows joins after its checklist.
- [ ] `install.sh` has two entrances. Beside a `gdoc` binary, it installs
      what is there. Otherwise it asks `api.github.com` for the latest
      release of `nhusnullin/gdoc` whose tag has `z == 0`, downloads
      `gdoc-<tag>-<platform>.zip` and `SHA256SUMS-<tag>` for `uname -m`,
      verifies with `shasum -a 256`, unpacks to a temp dir and continues from
      there. `--tag <tag>` picks a release by hand. It refuses to run from
      inside a checkout of this repository.
- [ ] It copies `gdoc` to `~/.local/bin/gdoc`, refusing a symlink there with a
      sentence naming the developer install; strips `com.apple.quarantine`
      when `xattr` is present; asks global or local unless `--skills` says;
      copies the three skills with `.release` markers under the three rules;
      runs `gdoc update --set-skills`, `gdoc completion zsh --out
      ~/.config/gdoc-agent/completion.zsh --force`, prints the `source` line
      when `.zshrc` lacks it in any spelling, and ends with `gdoc auth status`.
- [ ] Run it by hand against a scratch `HOME` on this machine, both
      placements and both entrances, and paste the summary into this task.
      The one-line form is
      `curl -fsSL https://raw.githubusercontent.com/nhusnullin/gdoc/main/release/install.sh | bash`.
- [ ] `git commit -m "feat(release): the installer in the zip"`

### Task 11: the user README, the example note, the issue template

**Files:**
- Create: `release/README.md`, `release/example/first-note.md`
- Create: `.github/ISSUE_TEMPLATE/report.md`

- [ ] `README.md` under a hundred lines: what gdoc is in three sentences;
      install; sign in with an `altery.com` account; three things to try;
      how updates work and how to turn nightly on; how to report; what gdoc
      never does. Plain English, no em dashes.
- [ ] `first-note.md`: a short note with a valid `gdoc:` front-matter block
      and a body that exercises a heading, a list and a table. `gdoc build`
      over it succeeds in a test.
- [ ] `report.md` asks for four things: version from `gdoc help`, the command
      as typed, the object it printed, what was expected.
- [ ] `git commit -m "docs(release): the user README, the example note, the issue template"`

### Task 12: the release workflow

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] Triggers: `push` of tags `v*`, and `workflow_call` with a `tag` input.
- [ ] Steps: checkout at the tag, setup-go from `go.mod`, gofmt, vet, the
      raced suite, `make dist`, pack one zip per line of `release/platforms`
      with the binary renamed to `gdoc`, `skills/`, `release/install.sh`,
      `release/README.md`, `release/example/`, write `SHA256SUMS-<tag>`,
      `gh release create <tag>` with the assets. `make dist` runs with
      `GDOC_OAUTH_CLIENT_SECRET` from the repository's secrets, and a step
      before packing refuses to continue when the secret is empty, because a
      release that cannot sign anyone in is not a release. Release notes are
      the commit subjects since the previous tag.
- [ ] `git commit -m "ci: the release workflow"`

### Task 13: the nightly

**Files:**
- Create: `.github/workflows/nightly.yml`

- [ ] Cron at 02:00 UTC and `workflow_dispatch`. Reads the last tag with
      `git describe --tags --abbrev=0`; exits quietly when main has not moved
      since it; otherwise tags `x.y.(z+1)`, pushes the tag with a token that
      may write contents, and calls `release.yml` with it.
- [ ] A dry-run input that computes and prints the next tag without creating
      it, for the acceptance task.
- [ ] `git commit -m "ci: the nightly tags and releases what moved"`

### Task 14: the repository secret

Nail's action, or the session's with his word in the transcript.

- [ ] Store the rotated OAuth client secret as `GDOC_OAUTH_CLIENT_SECRET` in
      this repository's Actions secrets, with `gh secret set`.
- [ ] Record it as done here with the date. The value is written nowhere
      else.

### Task 15: the documents

**Files:**
- Modify: `docs/v2/SPEC.md`, `docs/v2/PLAN.md`, `CLAUDE.md`, `README.md`,
  `go/internal/guard/doc.go`, `go/cmd/gdoc/doc.go`

The DECISIONS.md entry is already written, dated 2026-09-16, with its
register row. This task makes the other documents agree with it.

- [ ] SPEC.md: an "Install and update" section in the present tense; the
      guard section names the fifth grant; the version rules; the `page_break`
      block under the generator.
- [ ] PLAN.md: the M9 section becomes the row in Task 17; M10 does not exist.
- [ ] CLAUDE.md: rows for `release/` and `go/internal/update/`; the invariant
      "Skills are linked, not copied" gains its release clause; "The guard
      owns the wire" names the fifth grant; "If you touch" gains the release
      row. Under 300 lines.
- [ ] README.md's install section says a checkout installs with `install.sh`
      and a colleague installs from a release zip.
- [ ] `cd go && go test -race ./...` passes, the docs tests included.
- [ ] `git commit -m "docs(v2): the release, the updater and the fifth grant in SPEC, PLAN and CLAUDE.md"`

### Task 16: acceptance, end to end on this machine

- [ ] Tag `v2.0.0-rc1` and push it. The release workflow runs green, and the
      releases page shows the release with two zips and the checksum file.
- [ ] In a scratch `HOME`: the one-line install with `--tag v2.0.0-rc1` and
      `--skills global`, then from the unpacked zip in a scratch hub with
      `--skills local`. `gdoc help` shows
      `v2.0.0-rc1`, `gdoc auth status` answers, the completion sources.
- [ ] Publish `example/first-note.md` from the scratch hub into Nail's test
      folder, and read the document back.
- [ ] Tag `v2.0.0-rc2`, push, wait for the release. In the scratch home, `gdoc
      update --now` on the stable channel does nothing, because both are
      pre-release; `gdoc update --now --channel nightly` is refused by name
      for a pre-release; then set `source` to the rc for the test and confirm
      the binary and both skill placements moved to rc2, `verified: true`,
      and `gdoc update --rollback` brings rc1 back.
- [ ] `nightly.yml` dry run prints the next tag it would cut.
- [ ] Delete both rc releases and tags.
- [ ] `make test`, `make vet`, `make dist` green; CI green on the branch.

### Task 17: close the milestone

- [ ] Move this plan to `docs/plans/completed/`.
- [ ] PLAN.md's done table gains the M9 row, and the open section goes.
- [ ] `git commit -m "docs(v2): close M9"`

## Post-Completion

**Nail's actions:**

- Merge the PR, then `git tag v2.0.0 && git push --tags`. The workflow builds
  and publishes; the zips appear on the releases page.
- Send colleagues the one-line install. Everyone can open Issues with the
  template now that the repository is public; a message to Nail still works.
- Windows: hand the Windows zip from a nightly build to one colleague with the
  checklist in `release/README.md`'s Windows section; when it comes back
  clean, add the line to `release/platforms` and tag the next `x.y.0`.

**Manual verification:**

- Open a fresh Claude Code session on a machine that installed from the zip,
  give it a document link, and watch `gdoc-review` run `gdoc update` and then
  `gdoc help` before its first call.
- After the first nightly, check that a stable-channel machine did not move
  and a nightly-channel one did.

## What this milestone leaves for later

- Signing and notarising the macOS binary, which needs an Apple developer
  account, and would let the installer stop stripping quarantine.
- PowerShell completion and the Windows installer script, with the Windows
  release.
- A `gdoc update` that also refreshes a skill copy the person edited: today
  an edited copy has no marker and is refused, by design.
