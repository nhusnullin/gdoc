# gdoc v2 Milestone 16: settings, the token in the Keychain, and answers an AI can act on. Specification, draft

2026-10-04. DRAFT. Every decision below is Nail's and settled; one measurement
is still owed, the two-build test with gdoc's own Developer ID certificate, and
Part C waits on it. The task list is written next. It grew out of the 2026-10-03 brainstorm on how a
colleague's own skill uses gdoc (George's `docs-gdoc-suggestion-edits`). It
was checked against main after M14 and M15 merged (`ed9d269`).

Three parts. Part A is small fixes that change no contract. Part B is the
settings, with the thread mark as the first one. Part C moves the token out of
the config folder and into the macOS Keychain.

## Principles

Serves: 4, every word costs a reader's attention. A colleague picks a mark that
is quieter than 🤖, and an AI that reads gdoc's refusals is told what to do
next. Serves: the token stops being a plain file anybody's backup or sync tool
can copy.

Strains, each needing a DECISIONS.md entry before code:

- "Everything gdoc writes into a thread opens with the robot prefix". The
  prefix becomes the mark in effect, and the mark may be empty.
- "The marker is the only record of authorship there is". With an empty mark
  there is none.
- "Secrets live in `~/.config/gdoc-agent/`" in CLAUDE.md. On macOS the token
  lives in the Keychain.
- Principle 1, a static binary with three modules and no external program, by
  whichever Keychain route C2 picks.

## Decisions taken

Nail, 2026-10-04:

| Rule | Part |
|---|---|
| The thread mark is a setting. 🤖, ⁘, ∴, an empty mark and other values are allowed | B |
| An empty mark is allowed: gdoc then writes the words with nothing in front | B |
| Settings live in a file, and the path to that file can be given as an argument when gdoc starts, so tests run in parallel with their own settings | B |
| Every setting can also be given as a direct argument, which overrides the file | B |
| The token is not a setting. It never lives in the settings file | C |
| On macOS the token moves into the Keychain and leaves the config folder | C |
| Signing in again after an update is acceptable | C |
| Any symbol may be set as the mark. gdoc suggests a default | B |
| The default mark, for a colleague who sets nothing, is ⁘ | B |
| `reply --body "<text>"` joins this milestone, and chat's `body` maps to it | B |
| Windows is out of scope: its token stays where it is | C |

From the 2026-10-03 brainstorm, taken then:

| Rule | Part |
|---|---|
| George keeps his own skill. gdoc's own answers have to be enough for it | A |
| Removing `propose --folder` stays in its backlog item, as M14 decided | none |

## Part A: answers an AI can act on

No contract changes. Each is one commit, test first.

1. Three refusals stop naming a person: `propose.go:178`, `annotate.go:67`,
   `cmd/gdoc/annotate.go:93` say "the reader has to guess at".
2. `propose/doc.go:306` says "both block routes", and a new test pins that a
   words proposal lands inside a table cell. No offline test pins it today;
   the only evidence is the 2026-09-18 live run in MEASURED.md.
3. An unknown field in a `--from` file is refused with the fields that entry
   takes, decoded entry by entry, still strictly. Today: `json: unknown field
   "reason"`.
4. A deletion is refused with the way out: quote a neighbour word and keep it
   in the replacement.
5. A refusal during the write names the proposal by number and quote.
6. The `--from` summary in help names `why`, the one required field it leaves
   out.
7. `propose` reads the document once before its first write, not twice.

## Part B: settings

### The file and the arguments

- The settings file is `settings.yaml` in the config folder, decoded strictly:
  an unknown key is refused by name. No file means every default.
- `--config <file>` is accepted by every command and by `gdoc mcp`. It names the
  settings file to use instead of the default one. A named file that does not
  exist is refused, never treated as empty.
- Each setting is also an argument of its own. The first is `--mark <value>`.
  An argument overrides the file. `--mark ""` is the empty mark.
- The order: argument, then the file in use, then the built-in default.
- `help --json` carries the settings in effect and where each one came from:
  `settings: {mark: "⁘", mark_from: "file"}`. A fact, not advice.
- The token, its path and the update stamp are not settings.

### The mark

- The mark in effect opens everything gdoc writes into a thread: a reply, a
  proposal's comment, an annotation. Then one space, then the words. An empty
  mark writes the words alone.
- `reply` adds the mark itself, as `propose` and `annotate` already do. Today
  `reply` refuses a body without 🤖, and M14's chat text tells the model to
  type it. Both change. For one release a body that already opens with a known
  mark is accepted with a warning, and the mark is not doubled.
- A value is refused when a thread would show it wrong: a line break,
  markdown, one of gdoc's document markers, or more than eight characters.
- Recognising gdoc's own replies (`comments.Reply.ByGdoc`, chat's
  `robot_not_ours`) checks the list in decision B2, never the account.

### Open decisions for Part B

- B1, settled 2026-10-04 (Nail: the mark serves the review, so its judgement
  belongs to the skill). What a mark means is the skill's business: skipping a
  thread already answered, never taking gdoc's own words as an instruction, and
  naming who asked. The binary keeps two jobs: it writes the mark in effect, and
  it prints `by_gdoc` as a fact, "this text opens with a known mark or the mark
  in effect", because chat has no skill and every skill would otherwise carry
  its own copy of the list. With an empty mark `by_gdoc` is false for that
  person's replies, help and the README say what that costs, and the binary
  keeps no record of the replies it wrote.
- Chat's `robot_not_ours` fact is removed. Since `7fea76a` it flags nothing and
  is only shown. M14 recorded it, so its removal is a DECISIONS.md line.
Settled 2026-10-04, kept here for the reasoning:

- B2. Any value may be set. Each gdoc recognises the known list (🤖, ⁘, ∴)
  plus its own mark. A colleague's mark outside the list is read as a person's
  reply by everybody else's gdoc, which is the price of free choice.
- B3. The default is ⁘. Every colleague's replies change on their next update.
  🤖 stays recognised forever, because old replies keep it.
- B4. `reply --body "<text>"` joins, beside `--body-file`. Chat's `body` maps
  to it, so chat stops writing a temp file per reply.

## Part C: the token in the Keychain

### Behaviour

- On macOS the token is one Keychain item, a generic password with service
  `gdoc`. The config folder holds no token file.
- gdoc reads it without letting macOS show a dialog. An item gdoc may not read
  without one counts as no token, and the answer says to sign in, as today.
- An existing `oauth-token.json` is deleted on the first run of this release,
  not copied, and the answer says to sign in. A token that stayed in the file
  would be the thing this part exists to remove.
- `auth status` says where the token lives: `token_store: keychain`.
- The test suite never touches the real Keychain. The store is an interface,
  the suite uses a fake, and one opt-in live test, with its own variable, writes
  and reads a throwaway item.
- `--config` and parallel tests do not change where the token is. Every
  process of one user shares one sign-in.

### Measured first, by a spike, before any task

1. Does a release binary read an item a previous release wrote, without a
   dialog? An unsigned binary changes identity with every build, so the
   expected answer is no, and the next run signs in again.
2. Does the read fail cleanly, with no dialog, when it is not allowed?
3. Does `gdoc mcp`, launched by Claude Desktop, read the item the terminal
   wrote? The extension runs the same installed binary, so the expected answer
   is yes.
4. What `make build` does to a developer's sign-in each time it rebuilds.
5. Self-signing (Nail, 2026-10-04): two builds signed with the same
   self-made code-signing certificate and the same identifier. Does build B
   read build A's item with no dialog? Then the same on a Mac where the
   certificate was never installed or trusted, which is every colleague's.
   If yes, the identity is stable across updates for free, with no Apple
   account, and signing in again after an update goes away.

### Measured 2026-10-04 on Nail's Mac (macOS 26, login keychain)

Two throwaway Go programs, A and B, built from one source with one constant
different, so their bytes differ. Service `gdoc-spike-*`, never gdoc's item.
cgo was used for the spike only. Every item and file was deleted afterwards.

| # | Signing | B reads A's item with | Result |
|---|---|---|---|
| 1 | none (Go's ad-hoc, `Identifier=a.out`) | `kSecUseAuthenticationUI = Fail` | blocked until the 20 s self-timeout: the documented flag does not stop the access dialog on this keychain |
| 2 | none | `SecKeychainSetUserInteractionAllowed(false)` | `-25293` (errSecAuthFailed) at once, no dialog |
| 3 | one self-made certificate, `--binary-identifier gdoc-spike`, signed by rcodesign 0.29.0, certificate then deleted from the Mac | the same switch | `-25293` at once. `codesign --verify` says B is valid and meets its designated requirement, `identifier "gdoc-spike" and certificate root = H"a711…"`, the same for A and B |

A always read its own item. What follows:

- gdoc must switch interaction off with the deprecated call. The flag Apple
  documents would hang `gdoc mcp` on a dialog nobody sees.
- Self-signing does not keep a sign-in across builds. The likely reason,
  unverified: since macOS 10.12 the keychain also checks a partition list, which
  for a binary with no Apple Team ID names its cdhash. A self-made certificate
  carries no Team ID (`TeamIdentifier=not set`). A Developer ID does, so it is
  expected to work, but nothing was measured with one.

Nail, 2026-10-04: gdoc is signed with its own Developer ID Application
certificate in Altery's team (ALTERY LTD, Team ID `8FGPRTH647`), separate from
the Altery certificate that already exists, so revoking gdoc's never touches
another Altery app. The two-build test is repeated with it before Part C is
planned.

Nail, 2026-10-04: the signing key never goes to GitHub. CI builds and drafts
the release; Nail signs the two macOS binaries on his Mac with one command,
which rebuilds the zips and `SHA256SUMS`, uploads them and publishes.
Nightlies stay unsigned, so a colleague on the nightly channel signs in again
after each nightly update.

Nail, 2026-10-04, on the rest of Part C:

- The old token file is deleted, not revoked at Google. No new grant in the
  guard. Copies in backups stay usable until Google expires the token or
  somebody revokes it by hand.
- Nail's own `make build` is signed with the same certificate, so his sign-in
  survives a rebuild.
- If the two-build test with the real certificate fails, Part C stops and the
  next step is discussed with Nail.
- Narrowing the token's Drive scope is not filed.
- The release is v2.11.0.

### Part C decisions, settled 2026-10-04

- C1. Releases are signed with gdoc's own Developer ID, on Nail's Mac, so the
  Keychain keeps trusting gdoc across updates. Hardened runtime on. No
  notarisation for now: gdoc installs through a script, not a browser.
- C2. gdoc reaches the Keychain through its own small bridge: about 200 lines of
  Go and one assembly file, no fourth module, CGO still off. Only that one file
  may use `go:linkname` and `cgo_import_dynamic`, held by a boundary test.
  Interaction is switched off with `SecKeychainSetUserInteractionAllowed(false)`,
  because the documented flag does not stop the dialog (measured above).
- C3. Windows is out of scope.
- C4. The signing identifier is `com.altery.gdoc`, chosen once: the Keychain
  trusts the team plus this name, so changing it later signs everybody out.
- C5. One release, v2.11.0, carries Parts A, B and C. It waits for signing to
  work. The Keychain never ships in an unsigned release.

C2 went to a senior engineer and a security engineer on 2026-10-04:

- Both reject `/usr/bin/security`: the item would then trust Apple's
  `security` tool, so any process could read the token with no dialog. Both
  reject an access list open to any program.
- The engineer built route b in a scratch folder with CGO off: it compiles for
  both darwin arches and passes vet. Go keeps `syscall.syscall6` reachable on
  purpose for x/sys, so no linker flag is needed. The binary already links
  Security.framework through `crypto/x509`. About 200 lines of Go and one
  60-line assembly file.
- The security engineer prefers reviewed code (purego or CGO) over a
  hand-written bridge, but assumed the bridge needs a blocked linkname. The
  engineer's build shows it does not.
- Both: the file-based login keychain, not the data protection keychain, which
  needs an entitlement an unsigned binary cannot carry. Unsigned, every update
  and every local `make build` is a stranger to the item, so the person signs in
  again.
- The security engineer: revoke the old token at Google rather than only
  deleting the file, because backups keep the file. That needs a new grant in
  the guard. Full `drive` scope is the bigger risk than where the token sits,
  and is a separate decision.
- Found in passing: `cmd/gdoc/desktop.go` runs `/usr/bin/open`, an exception to
  "nothing runs an external program" that DECISIONS.md records for M14 but the
  invariant line in CLAUDE.md does not name.

The gate before any task: a spike shows that a read gdoc may not make fails
with no dialog, including under Claude Desktop. If macOS shows a dialog anyway,
`gdoc mcp` could wait on a window nobody sees, and Part C is redesigned.

## Not in this milestone

Each stays a backlog item or an idea until Nail names it:

- A line break hides from `read` (`docs/backlog/a-line-break-hides-from-read-and-propose.md`).
- Help that carries an example input file, as a new field in the help object.
- Picking one cell when a value repeats (`near`).
- A proposal that answers the thread that asked (`reply_to`).
- Stable refusal codes and a feature list for skill authors.
- Removing `propose --folder`.

## Tests that will be named

Written into DECISIONS.md by the first task, once the decisions above are in:

- `TestNoRefusalNamesAPerson`
- `TestFindSpanPlacesAQuoteInsideATableCell`
- `TestAnUnknownFieldIsRefusedWithTheFieldsTheEntryTakes`
- `TestASettingArgumentOverridesTheFile`
- `TestAConfigFileThatDoesNotExistIsRefused`
- `TestReplyAddsTheMarkInEffect`
- `TestAnEmptyMarkWritesTheWordsAlone`
- `TestTheSuiteNeverTouchesTheKeychain`
- `TestAnOldTokenFileIsDeletedAndSignInIsAsked`
