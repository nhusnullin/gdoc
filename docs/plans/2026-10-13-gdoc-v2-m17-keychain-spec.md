# gdoc v2 Milestone 17: the token in the macOS Keychain, and releases signed on Nail's Mac. Specification

2026-10-13. Split from M16 by Nail, so v2.11.0 ships settings and the mark
without waiting for the certificate. Everything below was decided and measured
on 2026-10-04 as M16's Part C, and moves here unchanged except that it ships as
v2.12.0, its own release. The task list is
`docs/plans/2026-10-13-gdoc-v2-m17-keychain.md`. The settings file from M16 is
not where the token goes: the token is never a setting.

## Principles

Serves: the token stops being a plain file anybody's backup or sync tool can
copy. Serves 1: no fourth module, CGO stays off.

Strains, each needing a DECISIONS.md entry before code: "Secrets live in
`~/.config/gdoc-agent/`" in CLAUDE.md, because on macOS the token lives in the
Keychain.

## Decisions taken

Nail, 2026-10-04:

| Rule |
|---|
| The token is not a setting. It never lives in the settings file |
| On macOS the token moves into the Keychain and leaves the config folder |
| Signing in again after an update is acceptable |
| Windows is out of scope: its token stays where it is |

⚠️ Open for Nail before any code: nightlies stay unsigned (decided), and the
Keychain never ships unsigned (decided). Both cannot hold once this milestone
merges, because the next nightly carries the Keychain unsigned.

## The token in the Keychain

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
- The release was v2.11.0; since the split it is v2.12.0.

### Decisions, settled 2026-10-04

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
- C5. One release, v2.11.0, carrying Parts A, B and C, was the 2026-10-04
  decision. Superseded 2026-10-13: Part C is M17 and ships as v2.12.0. The
  Keychain never ships in an unsigned release.

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
