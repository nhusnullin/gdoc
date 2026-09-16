# gdoc v2 Milestone 9: the release, the nightly, and the updater

2026-09-16. Revised the same evening: updates are on demand only, and the
skills travel as a Claude Code plugin from a marketplace in this repository.

## Principles

Serves: 1, it runs on someone else's machine. The release is one zip a
colleague unpacks or one line they paste, one binary, and the skills through
the channel Claude Code already has for them. Nothing else lands on the
machine. Serves 4, every word costs attention: a colleague learns gdoc from a
hundred-line README and from `gdoc help`, and an update is one command they
chose to run.

Strains: none. The updater runs only when a person types `gdoc update`, so
nothing changes under anyone. The first draft of this plan had a minor
version applying itself before a session; Nail withdrew that the same
evening, and with it the one strain on principle 3 this plan carried.

## Overview

gdoc goes to the team. Colleagues are on macOS and Windows, all with Claude
Code, and they will try the whole lifecycle: review, publish, restyle.
Feedback comes as a GitHub Issue or a message to Nail. macOS ships first;
Windows follows under its own tag when a colleague has run its checklist.

| Piece | Today | After |
|---|---|---|
| version | none; no tags | `x.y.z` baked in from the tag, in `help`, `auth status` and every envelope |
| release | `make dist` by hand | a tag builds it in CI, zips it, and publishes it as a GitHub Release of this repository, which is public |
| nightly | none | main moved since the last tag: CI bumps the plugin version, tags `x.y.(z+1)` at 02:00 UTC and releases it |
| binary install | `install.sh` in a checkout | one line, `curl -fsSL .../release/install.sh \| bash`, which fetches the latest release zip, verifies it and copies the binary; the same script runs from an unpacked zip |
| skills install | a symlink into a checkout | three routes by the machine's policy: `/plugin marketplace add nhusnullin/gdoc` then `/plugin install gdoc@gdoc` where plugins are open; `install.sh --skills global\|local` copying from the zip where marketplaces are restricted but local skills load; and the administrator enabling this plugin in managed settings where nothing else loads |
| update | by hand | `gdoc update`, when a person runs it and never otherwise: minor by default, `--major` for a major, `--nightly` for a nightly, `--check` to look, `--rollback` to go back. Skills update through Claude Code's own plugin toggle |
| skills | say "Nail", point at Nail's checkout | say "you", carry no path, and name the binary version they need |
| page breaks | none in the docx route | before "Version Control" and before "Contents", in both routes, from one block in the house style |
| feedback | none | an issue template asking for version, command, object and expectation |

## Decisions Nail took, 2026-09-16

Taken in the brainstorm that produced this plan and in its revision the same
evening, and written into DECISIONS.md. Each is a decision and not a
refactor. A task that finds one wrong stops and says so.

1. **Releases are GitHub Releases of this repository, which is public.**
   Nail made `nhusnullin/gdoc` public on 2026-09-16 after the assessment in
   DECISIONS.md: no secret beyond the Internal OAuth client, no document,
   and the client secret left the source the same day, injected at build
   time from `GDOC_OAUTH_CLIENT_SECRET`, already stored as a repository
   secret. Colleagues download from this repository's releases page and the
   updater fetches from it with no credential.
2. **Versions are `x.y.z`.** Nail tags `x.y.0` by hand. The nightly tags
   `x.y.(z+1)`. The number is the channel: `z == 0` is stable, `z > 0` is
   nightly. The first tag is `v2.0.0`. The plugin's version in
   `.claude-plugin/plugin.json` is the same number, and a release whose tag
   and plugin version disagree is refused by the workflow.
3. **Updates are on demand.** `gdoc update` runs when a person types it and
   never otherwise: no check at session start, no scheduler, no stamp. What
   it applies when run: same `x`, higher `y`, by default; a higher `x` only
   with `--major`; a higher `z` only with `--nightly`; never down. It says
   what it did in its object.
4. **The skills travel as a Claude Code plugin, with two fallbacks by
   policy.** This repository carries `.claude-plugin/plugin.json` and
   `.claude-plugin/marketplace.json`, so it is both the plugin and the
   marketplace, and `skills/` stays exactly where it is. Where plugins are
   open, a colleague installs with two `/plugin` commands, chooses global or
   per-project in Claude Code's own terms, and updates through Claude Code's
   per-marketplace toggle. Where an administrator has restricted
   marketplaces but personal and project skills still load, the zip carries
   `skills/` and `install.sh --skills global|local` copies them, marked so a
   re-run replaces only what it wrote; re-running the installer is the
   update. Where `strictPluginOnlyCustomization` blocks everything but
   plugins and managed settings, the administrator enables this plugin in
   `managed-settings.json` with `extraKnownMarketplaces` and
   `enabledPlugins`, which is a two-line change because the plugin format is
   what managed settings speak. The README says which route applies and
   how to tell. The 2026-08-14 decision, skills are linked and never copied,
   holds for a checkout; a release copy exists only on the fallback route,
   and nothing in gdoc's binary copies a skill folder.
5. **The zip carries the binary, the README and the skills**, and the
   installer copies the binary, strips quarantine, and says the plugin
   commands. It copies the skills only when `--skills` says so, and asks
   nothing. The binary still never prompts.
6. **A skill names the binary version it needs.** Its Setup runs `gdoc help`
   first, reads the version from the object, and when it is older than the
   skill's minimum says so and names `gdoc update`. Skills and binary can
   drift by a minor version without harm, because a skill holds no flag list.
7. **The guard gains one read-only door for updates.** `AllowUpdateFrom`
   names one repository for one run and admits GET on `api.github.com`,
   `github.com` and the asset host for that repository's releases, with no
   Authorization header, because the only bearer gdoc holds is Google's.
   Opened by `gdoc update` alone.
8. **Page breaks are a block in the house style**, `- block: page_break`,
   before the `version_control` label and after the `document_classification`
   table, read by the docx renderer and by the prelude. One layout, two
   writers. The prelude's own cover page break becomes that block. Measured
   in Task 3: the drift gate reports no new row, because no item in
   `drift.Items` reads a page break, so `drift.Known` gains nothing.
9. **Builds are trimmed and stripped.** `-trimpath` and `-ldflags "-s -w"` in
   `build` and `dist`: Nail's home path is not shipped, and a tagged build is
   the same bytes on every machine.
10. **Windows ships when its checklist has run.** `release/platforms` lists
    `darwin-arm64` and `darwin-amd64` for `v2.0.0`. Adding the Windows line
    is its own commit after a colleague pastes the checklist back.

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
- **`internal/atomicfile.Replace`** is the one room that replaces a file's
  contents. The binary itself is replaced by rename, because a running
  executable cannot be written through, and the rename works on both
  platforms.
- **The house style's front-matter blocks** are `cover`, `label`, `table`,
  `blank`, `legend` and `toc`, decoded in `go/internal/house/house.go` and
  walked by `go/internal/render/front.go` and
  `go/internal/prelude/frontmatter.go`. The renderer already knows
  `PageBreak` on a paragraph; the prelude already has `pageBreak()`, called
  once from `cover.go:43`, pinned by `TestTheCoverEndsWithAPageBreak`.
- **The master template has one page break**, before its first heading, so
  the two new ones are two rows in `drift.Known`, `go/internal/drift/compare.go`.
- **`make dist`** builds three binaries with `CGO_ENABLED=0` and, since
  b8d903a, `-ldflags` carrying the client secret from the environment. The
  Go workflow runs gofmt, vet, the raced suite and `make dist` on every push.
- **A tag pushed with `GITHUB_TOKEN` triggers no other workflow.** The
  nightly cannot rely on the tag event; it calls the release workflow as a
  reusable one. `GITHUB_TOKEN` can create a release on this repository, so no
  second token is needed.
- **The client secret is a repository secret**, `GDOC_OAUTH_CLIENT_SECRET`,
  set on 2026-09-16, and the release workflow passes it to `make dist`. A
  build without it cannot sign anyone in, so a release built without the
  secret is a failed release, and the workflow refuses to publish one.
- **A managed Claude Code can lock this down.** `strictKnownMarketplaces`
  and `blockedMarketplaces` refuse a marketplace before any network call;
  `strictPluginOnlyCustomization` blocks skills from `~/.claude/skills` and
  `.claude/skills` entirely, leaving plugins and managed settings as the
  only sources. An organisation distributes to everyone through
  `managed-settings.json`, with `extraKnownMarketplaces` naming the
  marketplace and `enabledPlugins` turning the plugin on, or through the
  admin console. code.claude.com/docs/en/plugin-marketplaces,
  settings-reference, admin-setup.
- **A Claude Code plugin** is a folder with `.claude-plugin/plugin.json`,
  holding `name`, `description` and an optional `version`, and a `skills/`
  directory beside it. A marketplace is a repository with
  `.claude-plugin/marketplace.json` listing plugins by source. Both files
  can sit at this repository's root, with `./` as the plugin source, and
  `skills/` is already there. Claude Code re-reads a skill's `SKILL.md` live
  and updates marketplace plugins on its per-marketplace toggle. Plugins may
  carry a `bin/` on PATH, but three binaries per release in git is the blob
  problem M7d's review found, so the binary stays a release asset.
  code.claude.com/docs/en/plugins, discover-plugins, skills.
- **The zip on macOS is quarantined** when it arrives through a browser, and
  an unsigned binary then refuses to run. The installer strips the attribute.
  A binary the updater downloads itself is not quarantined.
- **`go/version` is not semver.** Three integers and a comparison is a page
  of code with no dependency.
- **`build` has no network at all**, and stays that way. Only `update`
  reaches GitHub, and only when typed.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it
  to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own
  commit, and each ends with the test gate as its last checkbox.
- **CRITICAL: every task MUST include new or updated tests**, success and
  failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** `make test`
  runs `-race`; keep it there.
- **CRITICAL: nothing checks for updates on its own.** No skill runs
  `gdoc update`, no command calls it, and there is no stamp file. A test in
  `skills_test.go` holds that no SKILL.md runs `update` without a person's
  word, and `update` itself has no code path that fires unasked.
- **CRITICAL: the guard's Google rules do not move.** A policy without
  `AllowUpdateFrom` refuses every GitHub host exactly as today, and every
  existing guard test stays green without an assertion changed.
- **CRITICAL: no bearer to GitHub.** A test hands the recorded update request
  to the guard and asserts it carries no Authorization header.
- **CRITICAL: nothing is replaced before it is verified.** The checksum check
  comes before the rename, and a failed check leaves the old binary exactly
  as it was.
- **CRITICAL: no `os/exec` anywhere, tests included.** The updater cannot run
  the new binary to verify it; the next run's envelope is the proof, and
  `verified` says whether the file on disk hashes to what was promised.
- **CRITICAL: one object on stdout, always.** `update` reports through
  `internal/emit` like every command.
- **CRITICAL: facts only in Go.** `update` says what it found and what it did.
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces. Plain English.

## Testing Strategy

- **Unit, `cmd/gdoc`**: the version in `help`, `auth status` and the
  envelope; `update`'s arguments and its policy on every version pair;
  `--check`, `--major`, `--nightly`, `--rollback`; the unreachable answer.
- **Unit, `internal/guard`**: the update grant carries the three hosts with
  GET and nothing else; a policy without it refuses them; no Authorization
  header reaches GitHub; the Google rules unchanged.
- **Unit, `internal/update`**: version parsing and comparison against
  literals; release selection per channel and platform from a recorded API
  answer; checksum verification refusing a wrong file; the replace sequence
  on a temp dir; rollback.
- **Unit, `internal/house`, `render`, `prelude`**: the `page_break` block
  decoded, rendered as `w:pageBreakBefore`, proposed as `insertPageBreak`;
  literals throughout.
- **Unit, `internal/drift`**: the two new rows in `Known`, and the gate green
  against the master.
- **Boundary**: `allowedModules` unchanged; `TestNetHTTPStaysInItsRooms`
  green with the fetches in `gapi`; `TestNothingRunsAnExternalProgram`
  green; the plugin manifest parses and names the three skills that exist.
- **Skills**: `skills_test.go` gains three checks: no SKILL.md says "Nail",
  none names a path under a home directory, and none runs `update`.
- **Live, opt-in, Task 14**: an rc tag through the whole pipeline, the
  binary installed in a scratch home, the plugin installed from the branch,
  and an update from rc1 to rc2 on this machine.
- Coverage standard: every exported function under `go/internal/` has a test;
  the new package at or above 80%.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- `cd go && go vet ./...`
- `make build && bin/gdoc help 2>&1 >/dev/null | head -1` shows the version
- `make dist` and `strings bin/gdoc-darwin-arm64 | grep -c nailkhusnullin`
  prints 0
- `python3 -c 'import json; print(json.load(open(".claude-plugin/plugin.json"))["version"])'`
  prints the version the next tag will carry

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

**The version** is `var version = "dev"` in `go/cmd/gdoc/main.go`, set by
`-ldflags "-X main.version=$(git describe --tags --always)"` in both Make
targets. It goes into `emit.Result` as `version`, omitted when `dev`, and
into the help prose's first line and `auth status`'s data.

**The plugin and the marketplace** are two files at the root.
`.claude-plugin/plugin.json` names `gdoc`, describes it in one sentence, and
carries the version. `.claude-plugin/marketplace.json` names the marketplace
`gdoc` and lists one plugin with source `./`. `make tag VERSION=vX.Y.0`
writes the version into `plugin.json`, commits it, tags and pushes; the
nightly does the same for `x.y.(z+1)`.

**The release path** is one workflow, `release.yml`, callable two ways:
`push` of a tag `v*`, and `workflow_call` with a tag input. It checks that
`plugin.json` carries the tag's version, runs the same four checks the Go
workflow runs, then `make dist` with the secret, then for each line of
`release/platforms` packs `gdoc-<tag>-<platform>.zip` holding the binary,
`skills/`, `install.sh`, `README.md` and `example/`, writes
`SHA256SUMS-<tag>`, and
creates the GitHub Release. The nightly, `nightly.yml`, runs on a cron, reads
the last tag, and if main moved, bumps `plugin.json`, commits, tags
`x.y.(z+1)`, and calls `release.yml` with it.

**The updater** is `go/internal/update`, pure where it can be: parse and
compare versions, choose a release for a channel and a platform from the API
answer, verify a zip against its checksum line, plan the replacement. The
command in `cmd/gdoc/update.go` opens a policy with `AllowUpdateFrom`,
fetches through `gapi`, and carries the plan out: the zip to a temp dir
under the config dir, the binary to `<path>.new`, checksum, old to
`<path>.previous`, new to `<path>`. `--rollback` swaps `.previous` back.
`--check` reports without touching anything. It holds no state between runs.

**The installer** is one script with two entrances. Run from an unpacked zip
it installs the binary beside it. Run from `curl | bash` it asks the releases
API for the latest stable tag, downloads the zip for this machine's platform
and its checksum file, verifies, unpacks to a temp dir and installs from
there. Either way it copies the binary to `~/.local/bin/gdoc`, strips
quarantine, writes the completion into the config dir, prints the `source`
line when `.zshrc` lacks it, prints the two `/plugin` commands, and ends with
`gdoc auth status`.

## Technical Details

**The update object**:

```json
{"ok":true,"version":"v2.0.0","data":{"installed":"v2.0.0","latest_stable":"v2.1.0","latest_nightly":"v2.1.2","action":"updated","to":"v2.1.0","verified":true,"previous":"/Users/x/.local/bin/gdoc.previous"}}
```

`action` is one of `up_to_date`, `updated`, `major_available`,
`nightly_available`, `unreachable`, `rolled_back`, `checked`. A
`major_available` carries `run: "gdoc update --major"` beside it. An
`unreachable` is `ok: true` with a warning naming the cause. `verified` is
the checksum of the file at its final path.

**The policy, as a table the test walks:**

| installed | latest stable | latest nightly | flags | action |
|---|---|---|---|---|
| 2.0.0 | 2.0.0 | 2.0.3 | none | up_to_date, and `nightly_available` named |
| 2.0.0 | 2.1.0 | 2.1.2 | none | updated to 2.1.0 |
| 2.0.0 | 3.0.0 | | none | major_available |
| 2.0.0 | 3.0.0 | | `--major` | updated to 3.0.0 |
| 2.0.0 | 2.0.0 | 2.0.3 | `--nightly` | updated to 2.0.3 |
| 2.0.3 | 2.1.0 | 2.1.0 | `--nightly` | updated to 2.1.0 |
| 2.1.0 | 2.0.0 | | any | up_to_date, never down |
| any | any | any | `--check` | checked, nothing written |

**The guard grant.** `AllowUpdateFrom("nhusnullin/gdoc")` admits:
`GET api.github.com/repos/nhusnullin/gdoc/releases`, `GET
github.com/nhusnullin/gdoc/releases/download/<tag>/<asset>`, and
`GET objects.githubusercontent.com/...` reached by the redirect from the
second, for the run's length. Any other method, path or repository on those
hosts is refused by name. A request carrying Authorization to any of them is
refused before it leaves. The listing call has a five-second timeout and the
download a longer one.

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

**The skill's version line.** Each SKILL.md's Setup: run `$GDOC help`, read
`version` from the object, and if it is below the line `needs: v2.0.0` in
the skill's own front matter, say "gdoc is older than this skill needs; run
`gdoc update`" and stop. `skills_test.go` checks the line parses.

**Global constraints:**

- Exactly one JSON object on stdout, exit 0 if and only if `ok`. No prompting,
  no stdin. The installer asks nothing either.
- Strict argument parsing. `update` takes no words and the flags `--check`,
  `--major`, `--nightly`, `--rollback`; a combination that means two things
  is refused by name.
- Nothing under `go/` imports `os/exec`, tests included.
- `install.sh` never edits `.zshrc`. It prints the line.

## Implementation Steps

### Task 1: the version, trimmed and stripped

**Files:**
- Modify: `go/cmd/gdoc/main.go`, `help.go`, `main_test.go`, `help_test.go`
- Modify: `go/internal/emit/emit.go`, `emit_test.go`
- Modify: `Makefile`

- [x] Test first, `TestTheVersionReachesTheEnvelopeAndTheHelp`: with
      `version` set in the test, every object carries `version`, the help
      prose opens with it, and `auth status` data carries it; with `dev` the
      field is absent and no existing test sees a change.
- [x] `var version = "dev"` in `main.go`; `emit.Result` gains `Version string
      \`json:"version,omitempty"\``; `run` sets it.
- [x] `Makefile`: `VERSION := $(shell git describe --tags --always --dirty)`,
      and `LDFLAGS` gains `-s -w -X main.version=$(VERSION)` beside the
      client secret; both targets add `-trimpath`.
- [x] `make dist && strings bin/gdoc-darwin-arm64 | grep -c nailkhusnullin`
      prints 0, for all three binaries. Sizes: darwin-arm64 12,022,530 bytes,
      darwin-amd64 12,937,296 bytes, windows-amd64.exe 12,988,928 bytes.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): the version in every envelope, and builds trimmed and stripped"`

### Task 2: the skills say "you", and name the version they need

**Files:**
- Modify: `skills/gdoc-review/SKILL.md`, `skills/gdoc-publish/SKILL.md`,
  `skills/gdoc-restyle/SKILL.md`
- Modify: `go/cmd/gdoc/skills_test.go`

- [x] Test first, three checks added to the SKILL.md walk: no file contains
      the word `Nail`; none contains `/Users/` or `~/src/`; each front matter
      carries `needs: vX.Y.Z` that parses; and none runs `$GDOC update`.
      `TestNoSkillNamesAPersonOrAMachinesPath`,
      `TestEverySkillNamesTheGdocItNeeds`, `TestNoSkillRunsUpdateOnItsOwn`,
      with the `needs` reader pinned against literals by
      `TestANeedsLineIsReadAsThreeNumbers` and
      `TestANeedsLineThatIsNotAVersionIsCaught`.
- [x] "Nail" becomes "you" for the person at the keyboard and "a colleague" or
      "the reviewer" where the text means somebody else in the document. The
      descriptions trigger on what the person says. Inside a skill "you" now
      means the person and nobody else: where the text meant the session it
      says so, or it says it as an imperative.
- [x] The review skill's `Spec:` line pointing into a checkout goes. Read the
      spec sections it names and confirm the skill already carries every
      sentence it needs; add the missing ones, not a path. Three sentences
      were missing and are now in the skills: every read takes in every tab,
      the docx export is the honest witness because Drive's own anchor
      survives a detachment, and `commentUpdateState: ALL_SAVED` is part of a
      reply's `verified`. The publish and restyle `Spec:` lines went the same
      way, because they named the same checkout.
- [x] Setup in each skill: `$GDOC help` first, the version read from the
      object, and the sentence naming `gdoc update` when it is below `needs`.
      A build with no `version` is a build from source and not an error.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(skill): the skills address whoever is at the keyboard, and name the gdoc they need"`

### Task 3: the page_break block in the docx route

**Files:**
- Modify: `go/internal/house/house.yaml`, `house.go`, `house_test.go`
- Modify: `go/internal/render/front.go`, `front_test.go`, `doc.go`
- Modify: `go/internal/prelude/frontmatter.go`, `cover_test.go`
- Modify: `docs/v2/DECISIONS.md`

- [x] Test first, literals: `house.yaml` decodes a `page_break` block at the
      two positions, and a `page_break` with any other key is refused by name.
- [x] Test, `TestTheFrontMatterBreaksBeforeVersionControlAndBeforeContents`:
      the rendered document has `w:pageBreakBefore` on exactly the two
      paragraphs, stated as literals, and the cover's trailing blanks are the
      new count.
- [x] `house.go` decodes the kind; `front.go` renders it as the empty paragraph
      the first heading already uses.
- [x] The offline drift gate is green. It reports no new difference and `Known`
      gains nothing: measured on 2026-09-16, 169 rows with 22 differences
      before the change and the same after, because no item in `drift.Items`
      reads a page break or counts a front-matter paragraph. The
      `docs/v2/DECISIONS.md` entry that expected two rows says what was
      measured instead.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): page breaks before Version Control and before Contents"`

Two things moved into this task, because every commit leaves the tree green
and `house.yaml` is read by both writers:

- The three blanks between the classification table and the contents label go
  with the eight under the date. Both were pushing a page with blank lines,
  which is what the two breaks replace.
- `internal/prelude`'s `block()` handles `page_break` and the call in its
  `"cover"` case goes, so the prelude sends one break where house.yaml states
  one. Task 4 keeps the rest: `coverBlock`'s own call, the read-back, `doc.go`
  and the two tests it names.

### Task 4: the page_break block in the prelude

**Files:**
- Modify: `go/internal/prelude/frontmatter.go`, `cover.go`, `cover_test.go`,
  `frontmatter_test.go`, `readback.go`, `readback_test.go`, `doc.go`

- [x] Test first, `TestTheFrontMatterEndsWithAPageBreak`: the requests carry
      an `insertPageBreak` after the classification table, and
      `TestTheCoverEndsWithAPageBreak` keeps passing with the break now coming
      from the block rather than from `cover.go:43`.
- [x] The hard-coded call in `coverBlock` goes, so one layout has two writers
      and no third. `block()` handling `page_break` landed in Task 3, which
      needed it to keep the suite green.
- [x] The read-back counts the second break, and `Verify` is unchanged in what
      it asks.
- [x] `doc.go` names the two tests.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): the prelude ends with a page break, from the same block"`

### Task 5: the plugin and the marketplace

**Files:**
- Create: `.claude-plugin/plugin.json`, `.claude-plugin/marketplace.json`
- Create: `go/boundary/plugin_test.go`
- Modify: `Makefile`, `install.sh`

- [x] Test first, `TestThePluginNamesTheSkillsThatExist`: `plugin.json`
      parses, its `name` is `gdoc`, its `version` is `vX.Y.Z` shaped, and
      `marketplace.json` lists exactly one plugin with source `./`. Every
      directory under `skills/` holds a `SKILL.md`, so the plugin ships
      nothing half made.
- [x] The two files, with the version `v2.0.0`.
      ➕ `TestTheInstallerLinksEverySkillThePluginShips` was added beside it:
      `install.sh`'s `SKILLS` array and the folders under `skills/` are two
      lists of one thing, so they fail in both directions like the wire and
      module lists.
      ➕ `marketplace.json` carries a `description`, which
      `claude plugin validate --strict` asks for.
- [x] `make tag VERSION=vX.Y.Z`: refuses a version that is not `x.y.0`,
      writes it into `plugin.json`, commits `chore: version vX.Y.Z`, tags,
      and pushes the commit and the tag. Nightly versions are CI's, Task 12.
- [x] `install.sh` in the checkout: unchanged for the binary and the symlinked
      skills, and its summary says the plugin is how a colleague gets them.
- [x] On this machine: `/plugin marketplace add` from the local checkout path
      and `/plugin install gdoc@gdoc` in a scratch project, and the three
      skills show in `/plugin`. Paste the listing here.
      ➕ Run through the `claude plugin` CLI with `CLAUDE_CONFIG_DIR` pointed at
      a scratch config, so nothing landed in Nail's own. The listing:

      ```
      gdoc v2.0.0
        Description: Review, publish and restyle Google Docs in the Altery
        house style, through the gdoc binary.
        Source: gdoc@gdoc

      Component inventory
        Skills (3)  gdoc-publish, gdoc-restyle, gdoc-review
        Agents (0)
        Hooks (0)
        MCP servers (0)
        LSP servers (0)

      Projected token cost
        Always-on:   ~222 tok   added to every session
      ```

      `claude plugin validate .` passes on the marketplace manifest, `--strict`
      included. The plugin manifest passes with one warning, that CLAUDE.md at
      the plugin root is not loaded as plugin context. That is what we want:
      CLAUDE.md is for whoever works on this repository, and the plugin ships
      the skills.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(release): this repository is a Claude Code marketplace and a plugin"`

### Task 6: the guard's update grant

**Files:**
- Modify: `go/internal/guard/policy.go`, `doc.go`, `transport.go`
- Create: `go/internal/guard/update_test.go`

- [x] Test first, as an attack: a policy without the grant refuses
      `api.github.com` by name, as today. With `AllowUpdateFrom(repo)`: the
      releases listing carries, the download path for that repository
      carries, the asset host carries for GET; a POST, another repository,
      another path, and any Google request through the same policy are each
      refused by name.
- [x] Test, `TestAnUpdateRequestCarriesNoBearer`: a request to any of the
      three hosts with an Authorization header is refused before it leaves.
- [x] `AllowUpdateFrom` in the shape of `AllowCreateIn`: per-run, one
      repository, read-only. `Judge` gains the three hosts under it.
- [x] `doc.go`'s "Two doors into the set, and four grants beside it" becomes
      five, with the reason and the tests.
- [x] Every existing guard test passes without an assertion changed.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): one read-only guard door for updates"`

➕ The tests went into a new `update_test.go` rather than into
`policy_test.go`, which holds one subject as `copy_test.go` and
`marker_test.go` do. The no-bearer rule is in `transport.go`, because a header
is not something `Judge` is handed; it refuses on the host rather than on the
grant, so it holds for a run that was never granted an update. `probe/doc.go`
and `withdraw/doc.go` name the renamed guard section, so both were corrected
to "five grants beside it".

### Task 7: releases, versions and the policy

**Files:**
- Create: `go/internal/update/doc.go`, `version.go`, `version_test.go`,
  `choose.go`, `choose_test.go`, `policy.go`, `policy_test.go`,
  `testdata/releases.json`
- Create: `go/internal/gapi/plain.go`, `plain_test.go`
- Modify: `go/internal/gapi/session.go` (GitHub's own error message)

- [x] Test first, literals: `Parse("v2.1.3")` and every malformed form
      refused; `Compare` on the table in Technical Details; `Choose` from the
      recorded releases answer picks the highest `z == 0` for stable and the
      highest overall for nightly, for one platform, and returns nothing when
      the platform's asset is missing.
- [x] Test: the policy table, every row, as one table-driven test.
      `TestThePolicyTable` walks all eight rows plus the four `--check` cases,
      an rc below its release, and a channel nothing was found in.
- [x] `gapi.Get(url)` for a client with no session: no bearer, a five-second
      timeout on the listing, JSON or bytes back, and the guard judging it
      like every other request.
- [x] `doc.go` opens `Package update` and names the tests.
- [x] `cd go && go test -race ./...` passes. `internal/update` is at 97.3%,
      `internal/gapi` at 91.0%.
- [x] `git commit -m "feat(v2): choosing a release, and the update policy"`

➕ The policy went into `policy.go` beside `choose.go` rather than into the
command, because the table is arithmetic over versions and that is what makes
it a test over literals. `Decide` is handed a `State` and `Flags` and answers
a `Decision`; the command in Task 9 opens the policy, fetches, and carries it
out.

➕ A version carries an optional pre-release word: `v2.0.0-rc1` parses, and an
rc sorts below the release it names. Task 14 accepts this milestone by cutting
`v2.0.0-rc1`, installing it, and updating from it to `rc2`, which three plain
integers cannot express. Two pre-releases compare as text, so `rc10` sorts
below `rc2`; reading the digits out would be a semver implementation arriving
one function at a time.

➕ The zero `Version` is how a channel with no release in it is said, and
`v0.0.0` parses to that same value. gdoc's first release is `v2.0.0` and tags
only go up, so the one tag that could be confused with nothing found is a tag
that will never exist. `TestTheZeroVersionIsHowNothingFoundIsSaid` pins it.

➕ `Choose` skips a draft and a tag that does not parse, and refuses when the
latest release of the channel carries no zip for this platform or no checksum
file. It never falls back to the release before it: an update that installs
something other than the newest is worse than one that says what is missing.

➕ `nightly_available` is not an action. The row it was written for is
`up_to_date` with `run: "gdoc update --nightly"` beside it, and the object
already carries `latest_nightly`, so a second action for the same state would
be two ways to say one thing. `unreachable` and `rolled_back` stay in the
`Action` list for Task 9, which is where a wire and a file are.

➕ The unauthenticated reach is `gapi.Plain`, built by `OpenPlain`, rather
than a package-level `Get`: `GetJSON` is the listing and carries the
five-second timeout, `GetBytes` is the download and runs on the caller's own
deadline. `statusError` gained GitHub's top-level `message` field, so a rate
limit reaches the envelope as the sentence GitHub wrote.

### Task 8: download, verify, replace, roll back

**Files:**
- Create: `go/internal/update/apply.go`, `apply_test.go`

- [ ] Test first, on a temp dir: a zip and its checksum line verify; a wrong
      checksum refuses and leaves every file as it was; the replace sequence
      leaves `<path>` as the new binary and `<path>.previous` as the old; a
      second replace overwrites `.previous`; `Rollback` swaps back and refuses
      when there is no previous; a download cut short fails the checksum and
      replaces nothing.
- [ ] `Apply` and `Rollback`, `archive/zip`, `crypto/sha256`, `os.Rename`.
      Windows: the same rename sequence, and a note in `doc.go` that it is
      unmeasured until the Windows checklist.
- [ ] `verified` is the hash of the file at its final path.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): the update applied, verified, and reversible"`

### Task 9: gdoc update

**Files:**
- Create: `go/cmd/gdoc/update.go`, `update_test.go`
- Modify: `go/cmd/gdoc/commands.go`, `doc.go`

- [ ] Test first: strict arguments, every flag combination that means two
      things refused by name; `--check` touches nothing; the object shape
      against literals; the envelope on each failure path.
- [ ] Test, `TestAnUnreachableGitHubIsAnAnswerAndNotAFailure`: a listing
      that times out, refuses the connection, answers 5xx or answers the rate
      limit each gives `ok: true`, `action: "unreachable"`, a warning naming
      the cause, and no file written.
- [ ] Test, `TestNothingChecksForUpdatesUnasked`: no other command's code
      path reaches `update`, held by reading the table: only the `update`
      entry names `cmdUpdate`.
- [ ] `--major`, `--nightly`, `--rollback` as the policy table says.
- [ ] `doc.go`: a section on the update, why it runs only when typed, why it
      never runs before `build`, and the tests.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "feat(v2): gdoc update, on demand"`

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
      when `xattr` is present; runs `gdoc completion zsh --out
      ~/.config/gdoc-agent/completion.zsh --force`; prints the `source` line
      when `.zshrc` lacks it in any spelling; prints the two `/plugin`
      commands; and ends with `gdoc auth status`. It asks nothing.
- [ ] `--skills global` or `--skills local` copies the three skill folders
      from the zip into `~/.claude/skills` or `./.claude/skills`, each with
      a `.gdoc-installed` file naming the version. A folder carrying that
      file is replaced; a folder without it is refused by name; a symlink is
      refused by name. Without the flag no skill folder is touched, and the
      summary says the plugin commands are the first route and `--skills`
      the second.
- [ ] Run it by hand against a scratch `HOME` on this machine, both
      entrances, and paste the summary into this task. The one-line form is
      `curl -fsSL https://raw.githubusercontent.com/nhusnullin/gdoc/main/release/install.sh | bash`.
- [ ] `git commit -m "feat(release): the installer in the zip, and the one line that fetches it"`

### Task 11: the user README, the example note, the issue template

**Files:**
- Create: `release/README.md`, `release/example/first-note.md`
- Create: `.github/ISSUE_TEMPLATE/report.md`

- [ ] `README.md` under a hundred lines: what gdoc is in three sentences;
      the one-line install; the skills by policy, in three short paragraphs:
      the two `/plugin` commands where plugins are open, `--skills` where a
      marketplace is refused, and the two managed-settings lines to hand to
      the administrator where nothing else loads, with the one command that
      tells which case a machine is in; where Claude Code keeps its plugin
      update toggle; sign in with an `altery.com` account, and that
      everyone signs in once more after 2026-09-16; three things to try;
      `gdoc update` and its flags, and that nothing updates on its own; how
      to report; what gdoc never does. Plain English, no em dashes.
- [ ] `first-note.md`: a short note with a valid `gdoc:` front-matter block
      and a body that exercises a heading, a list and a table. `gdoc build`
      over it succeeds in a test.
- [ ] `report.md` asks for four things: version from `gdoc help`, the command
      as typed, the object it printed, what was expected.
- [ ] `git commit -m "docs(release): the user README, the example note, the issue template"`

### Task 12: the release workflow and the nightly

**Files:**
- Create: `.github/workflows/release.yml`, `.github/workflows/nightly.yml`

- [ ] `release.yml` triggers: `push` of tags `v*`, and `workflow_call` with a
      `tag` input. Steps: checkout at the tag; refuse when `plugin.json`'s
      version is not the tag; setup-go from `go.mod`; gofmt, vet, the raced
      suite; `make dist` with `GDOC_OAUTH_CLIENT_SECRET` from secrets, and a
      refusal when it is empty; pack one zip per line of `release/platforms`
      with the binary renamed to `gdoc`, `skills/`, `release/install.sh`,
      `release/README.md`, `release/example/`; write `SHA256SUMS-<tag>`;
      `gh release create <tag>` with the assets and the commit subjects since
      the previous tag as notes.
- [ ] `nightly.yml`: cron at 02:00 UTC and `workflow_dispatch` with a dry-run
      input. Reads the last tag with `git describe --tags --abbrev=0`; exits
      quietly when main has not moved since it; otherwise writes
      `x.y.(z+1)` into `plugin.json`, commits `chore: nightly vX.Y.Z`, tags,
      pushes both with a token that may write contents, and calls
      `release.yml` with the tag. The dry run prints the tag it would cut.
- [ ] `git commit -m "ci: the release workflow, and the nightly that tags what moved"`

### Task 13: the documents

**Files:**
- Modify: `docs/v2/SPEC.md`, `docs/v2/PLAN.md`, `CLAUDE.md`, `README.md`,
  `go/internal/guard/doc.go`, `go/cmd/gdoc/doc.go`

The DECISIONS.md entries are already written, dated 2026-09-16. This task
makes the other documents agree with them.

- [ ] SPEC.md: an "Install and update" section in the present tense, the
      plugin and the marketplace, the guard section naming the fifth grant,
      the version rules, the `page_break` block under the generator.
- [ ] PLAN.md: the M9 section becomes the row in Task 15.
- [ ] CLAUDE.md: rows for `release/`, `.claude-plugin/` and
      `go/internal/update/`; "The guard owns the wire" names the fifth
      grant; "If you touch" gains the release row. Under 300 lines.
- [ ] README.md's install section says a checkout installs with `install.sh`
      and a colleague installs with the one line and the two `/plugin`
      commands.
- [ ] `cd go && go test -race ./...` passes, the docs tests included.
- [ ] `git commit -m "docs(v2): the release, the plugin, the updater and the fifth grant in SPEC, PLAN and CLAUDE.md"`

### Task 14: acceptance, end to end on this machine

- [ ] `make tag` is not used for an rc; by hand: write `v2.0.0-rc1` into
      `plugin.json`, commit, tag, push. The release workflow runs green and
      the releases page shows the release with two zips and the checksum
      file.
- [ ] In a scratch `HOME`: the one-line install with `--tag v2.0.0-rc1`.
      `gdoc help` shows `v2.0.0-rc1`, `gdoc auth status` answers, the
      completion sources.
- [ ] In a scratch project: `/plugin marketplace add` pointing at the branch
      and `/plugin install gdoc@gdoc`; `gdoc-publish` triggers on a sentence
      naming a note and a folder, runs `gdoc help` first, and reads the
      version. Then in a second scratch home, `install.sh --skills global`
      from the zip, and the same skill triggers from the copied folder.
- [ ] Publish `example/first-note.md` from the scratch project into Nail's
      test folder, and read the document back.
- [ ] Repeat the tag as `v2.0.0-rc2`. In the scratch home, `gdoc update
      --check` names rc2 as available; `gdoc update` moves the binary to
      rc2 with `verified: true`; `gdoc update --rollback` brings rc1 back.
- [ ] `nightly.yml` dry run prints the next tag it would cut.
- [ ] Delete both rc releases and tags, and reset `plugin.json` to `v2.0.0`.
- [ ] `make test`, `make vet`, `make dist` green; CI green on the branch.

### Task 15: close the milestone

- [ ] Move this plan to `docs/plans/completed/`.
- [ ] PLAN.md's done table gains the M9 row, and the open section goes.
- [ ] `git commit -m "docs(v2): close M9"`

## Post-Completion

**Nail's actions:**

- Merge the PR, then `make tag VERSION=v2.0.0`. The workflow builds and
  publishes; the zips appear on the releases page.
- Send colleagues the one line and the two `/plugin` commands. Everyone can
  open Issues with the template; a message to Nail still works.
- Windows: hand the Windows zip from a nightly build to one colleague with the
  checklist in `release/README.md`'s Windows section; when it comes back
  clean, add the line to `release/platforms` and tag the next `x.y.0`.

**Manual verification:**

- Open a fresh Claude Code session on a machine that installed from the zip
  and the marketplace, give it a document link, and watch `gdoc-review` run
  `gdoc help` first and read the version.
- After the first nightly, confirm no colleague's binary moved, and that
  `gdoc update --check` on one machine names the nightly as available.

## What this milestone leaves for later

- Signing the release: an ed25519 key pair with the public key inside the
  binary and the signing in CI, so `gdoc update` trusts the key and not the
  origin. Standard library, a page of code.
- Signing and notarising the macOS binary, which needs an Apple developer
  account, and would let the installer stop stripping quarantine.
- PowerShell completion and the Windows installer script, with the Windows
  release.
