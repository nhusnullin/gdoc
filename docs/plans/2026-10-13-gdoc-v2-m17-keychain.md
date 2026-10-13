# gdoc v2 Milestone 17: the token in the macOS Keychain, and releases signed on Nail's Mac, plan

2026-10-13. Split from M16 by Nail on 2026-10-13, so v2.11.0 ships settings and
the mark without waiting for the certificate. The specification is
`docs/plans/2026-10-13-gdoc-v2-m17-keychain-spec.md`, and its decisions are
the ones taken for M16's Part C on 2026-10-04. One commit per task. Released as
v2.12.0, signed, after the Task 1 measurement passes and after Nail's
acceptance. It starts from `main` after M16 has merged.

## Context (from discovery)

- The token is read and written through four functions in `internal/auth`:
  `Load` (`auth.go:86`), `Save` (`auth.go:225`, through `atomicfile` at
  0o600), `Status` (`auth.go:275`, reports `token_path`) and `SignedInSince`
  (`pending.go:126`), which reads the token file's mtime as the saved instant.
  A Keychain store needs its own saved instant.
- Callers outside `auth`: `gapi.Open` (`gapi/session.go:188`), the refresh
  save (`session.go:566`) and `newerLogin` (`session.go:594`);
  `Pending.Wait` saves (`login.go:226`); `cmd/gdoc/mcplogin.go:141,182,197,217`
  and `mcptools.go:142` call `Load` and `SignedInSince`; `main.go:278,305` call
  `Status`; `statusscreen.go:61` prints `token_path` under the key `file`.
- `config.TokenPath()` (`config.go:66`) stays: the file store uses it, and on
  darwin the first run of v2.11.0 needs it to find the file it deletes.
  `login-pending.json` stays in the config folder: it carries no credential.
- `cmd/gdoc/doc.go:703`: "A test that wrote to the developer's own token file
  would be a test that signs them out." The same holds for the Keychain, and
  every test today isolates the token with `GDOC_CONFIG_DIR` alone.
- `go 1.27.0` in `go.mod`, so `testing.Testing()` is available.
- `internal/tty/terminal_darwin.go` already imports `unsafe` for an ioctl. No
  file in the tree uses `go:linkname` or `go:cgo_import_dynamic` today.
- The Makefile's `build` sets no `CGO_ENABLED`; `dist` sets `CGO_ENABLED=0`.
  The bridge uses no cgo, so both work.
- `release.yml` publishes with `gh release create "$TAG" ... dist/*.zip
  dist/SHA256SUMS-$TAG` for every tag, stable and nightly alike. A zip is flat:
  the binary as `gdoc`, `install.sh`, `README.md`, `example/`, `skills/`,
  `mcpb/manifest.json`. `release/platforms` lists `darwin-arm64` and
  `darwin-amd64` only, so every zip a release carries is a darwin zip.
- The updater skips drafts (`internal/update/choose.go:74`), so a stable
  release created as a draft is invisible to `gdoc update` until it is
  published. The checksum file is `SHA256SUMS-<tag>` (`choose.go:65`).
- gdoc's certificate and Altery's existing one are both "Developer ID
  Application: ALTERY LTD (8FGPRTH647)". `codesign -s "<that name>"` is
  ambiguous when both are in the login keychain, so signing names the
  certificate by its SHA-1 hash.
- Tests over the workflows and the Makefile live in
  `go/boundary/workflows_test.go` and `go/boundary/makefile_test.go`, and read
  the files as text.
- `go/internal/live/doc.go` names every live variable and every live test;
  `GDOC_LIVE_TEST`, `GDOC_LIVE_WRITE` and the rest each gate their own tests.

## Development Approach

The rules of M16's plan hold. M17 adds:

- **CRITICAL: no test touches the real Keychain.** Only the opt-in live test
  under `GDOC_LIVE_KEYCHAIN=1` does, with its own service name.
- **CRITICAL: no built binary runs on darwin without `GDOC_CONFIG_DIR` set to a
  temp directory.** From Task 3 on, a v2.12.0 binary deletes the
  `oauth-token.json` it finds, and Nail's real one is in the default folder.
- **CRITICAL: the guard does not move.** M17 adds no grant, host or request
  kind: `git diff main...HEAD -- go/internal/guard/policy.go` is empty.
- **CRITICAL: no fourth module, and CGO stays off for `make dist`.**
- **CRITICAL: the signing identifier `com.altery.gdoc` is Nail's decision** and
  is not an example.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && test -z "$(gofmt -l .)" && go vet ./... && GOOS=darwin go vet ./...`
- `cd go && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o /dev/null ./cmd/gdoc`
  and the same for `amd64`.
- `grep -rln 'go:linkname\|go:cgo_import_dynamic' go --include='*.go'` names
  `go/internal/auth/keychain_darwin.go` only.
- `cd go && go test -race ./...` on Nail's Mac shows no dialog, and afterwards
  `security find-generic-password -s gdoc-live-test` finds nothing.
- `bash -n release/sign.sh`; `make -n sign-release TAG=v2.11.0` names the script.
- `git diff main...HEAD -- go/internal/guard/policy.go` is empty.
- `make dist` builds the binaries unsigned.

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Implementation Steps

⚠️ Open for Nail before Part C starts: nightlies stay unsigned (decided), and
the spec says the Keychain never ships unsigned (decided). Both cannot hold
once Part C merges, because the next nightly carries the Keychain unsigned.

### Task 1: the measurement, by hand, before any Part C code

⚠️ Nail's task. ralphex cannot sign with the Developer ID or open Claude
Desktop. The run stops here unless `docs/v2/MEASURED.md` already carries the
entry below, dated and confirmed by Nail.

**Files:**
- Modify: `docs/v2/MEASURED.md` (Nail writes it, or confirms what a session
  wrote)

- [ ] Nail measures, with two throwaway programs built from one source with
      one constant different, using service `gdoc-spike-*` and never `gdoc`,
      both signed `codesign --force --options runtime --identifier
      com.altery.gdoc --timestamp -s <SHA-1 of gdoc's Developer ID>`:
  1. Does B read A's item with interaction off and no dialog?
  2. Does a read B may not make (A unsigned, B signed, and the reverse) fail
     at once with no dialog, and with which code?
  3. Does `gdoc mcp` under Claude Desktop read the item the terminal wrote:
     the spike program run as the extension's command?
  4. Does a local build signed with the same certificate read the item a
     release-signed build wrote? (`make build` keeping Nail's sign-in.)
  5. The same two-build test on a Mac where the certificate was never
     installed or trusted, which is every colleague's.
  6. When B cannot read A's item, can B delete it and write its own, with no
     dialog? Every nightly and every unsigned build depends on this: signing
     in again writes over an item it could not read.
  7. Is hardened runtime on a Go binary any trouble for the reads above?
- [ ] Each result records how it was taken: the commands, macOS version, which
      keychain, and the exit or OSStatus code, in the shape of the 2026-10-04
      table in the spec.
- [ ] The gate: 1, 2, 3 and 6 must come out as the spec expects. If any does
      not, Part C stops here and the next step is Nail's (the spec's rule).
- [ ] The executor's half: read `MEASURED.md`. If the entry is missing, or any
      of 1, 2, 3 and 6 failed, stop and say so; write no code.
- [ ] `git commit -m "docs(measured): the Keychain with gdoc's own Developer ID"` (only if the executor added nothing but a ➕ note, otherwise Nail's commit stands)

### Task 2: the token goes through a Store

No behaviour changes on any platform: the default store is still the file.

**Files:**
- Create: `go/internal/auth/store.go`, `go/internal/auth/filestore.go`,
  `go/internal/auth/store_test.go` (holds `memStore`)
- Modify: `go/internal/auth/auth.go`, `go/internal/auth/pending.go`,
  `go/internal/auth/doc.go`, `go/internal/auth/token_test.go`,
  `go/internal/auth/pending_test.go`

- [ ] `type Store interface { Load() (Token, time.Time, error); Save(Token)
      error; Delete() error; Where() string }`. The time is the instant the
      token was saved, which `SignedInSince` reads today off the file's mtime.
      `Where` is `file` or `keychain`.
- [ ] `fileStore` is today's code moved: `oauth-token.json`, google-auth
      shape, `atomicfile` at 0o600, the mtime as the saved instant. `Load`,
      `Save` and `SignedInSince` keep their signatures and ask the store in
      use, so `gapi`, `login.go` and `cmd/gdoc` do not change in this task.
- [ ] `StatusReport` gains `TokenStore string json:"token_store"`.
      `TokenPath` becomes `omitempty`: present for the file store, absent
      where there is no file.
- [ ] `memStore` in `store_test.go` runs the same contract tests as
      `fileStore`: `TestEveryStoreKeepsWhatWasSaved`,
      `TestEveryStoreSaysNoTokenWhenEmpty`,
      `TestSignedInSinceAsksTheStoresSavedInstant`,
      `TestDeleteLeavesNoToken`.
- [ ] Every existing auth test passes unchanged: the file store is today's
      file byte for byte (`TestTheFileStoreWritesTodaysFile`).
- [ ] `auth status` reports `token_store: file`:
      `TestAuthStatusSaysWhereTheTokenLives` (with the statusscreen row key
      `store`, value `file` and the path under it).
- [ ] `git commit -m "refactor(auth): the token goes through one Store"`

### Task 3: the Keychain store, with the bridge behind an interface

All of the store's logic, tested on ubuntu, with a fake bridge. The real one is
Task 4.

**Files:**
- Create: `go/internal/auth/keychain.go`, `go/internal/auth/keychain_test.go`
- Modify: `go/internal/auth/doc.go`

- [ ] `type itemBridge interface { find(service, account string) ([]byte,
      error); put(service, account string, data []byte) error;
      remove(service, account string) error }` with three sentinel errors:
      not found, not allowed (the item is there and this binary may not read
      it), and the rest carrying the OSStatus code.
- [ ] `keychainStore{service, bridge}`. Account `oauth-token` (the spec names
      the service only; this name is the plan's). The item holds
      `{"saved": <RFC 3339>, "token": <the google-auth shape>}`.
- [ ] Load: not found is `ErrNoToken`; not allowed is `ErrNoToken` with "the
      Keychain did not let this gdoc read the token, which happens after an
      update to a build signed another way. Run: gdoc auth login".
- [ ] Save: put; on not allowed, remove then put, as Task 1 measurement 6
      found possible. If Task 1 found otherwise, this bullet is what changes.
- [ ] The first run of v2.12.0: Load, before anything else, deletes an
      `oauth-token.json` at `config.TokenPath()`, never reading or copying it.
      With no item in the Keychain the answer is `ErrNoToken` saying the file
      was deleted and to sign in; with an item there, the run goes on and a
      warning says the file was deleted.
- [ ] Tests: `TestAnOldTokenFileIsDeletedAndSignInIsAsked`,
      `TestAnOldTokenFileIsNeverReadOrCopied` (a file holding a valid token
      and an empty Keychain: no token after),
      `TestAnItemCarriesTheSavedInstant`, `TestAReadTheKeychainRefusesIsNoToken`,
      `TestASaveOverAnItemItCannotReadReplacesIt`,
      `TestAnotherKeychainErrorIsNamedWithItsCode`,
      `TestTheKeychainStoreSaysKeychain`.
- [ ] `git commit -m "feat(auth): the Keychain store, its item and the old file"`

### Task 4: the darwin bridge, and the Keychain as the darwin default

**Files:**
- Create: `go/internal/auth/keychain_darwin.go`, `go/internal/auth/keychain_darwin.s`,
  `go/internal/auth/store_darwin.go`, `go/internal/auth/store_other.go`,
  `go/boundary/keychain_test.go`
- Modify: `go/internal/auth/doc.go`, `go/boundary/doc.go`, `go/cmd/gdoc/doc.go`

- [ ] `keychain_darwin.go`, about 200 lines, the one Go file that uses
      `go:linkname` and `go:cgo_import_dynamic`: Security.framework's
      `SecItemCopyMatching`, `SecItemAdd`, `SecItemUpdate`, `SecItemDelete`
      and `SecKeychainSetUserInteractionAllowed`, CoreFoundation's few
      constructors and `CFRelease`, reached through `syscall.syscall6` the way
      `crypto/x509/internal/macos` does. A generic password, service `gdoc`,
      in the file-based login keychain: no `kSecUseDataProtectionKeychain`.
- [ ] Interaction is switched off with
      `SecKeychainSetUserInteractionAllowed(false)` before the first call. The
      documented `kSecUseAuthenticationUI` flag is not used: the spec measured
      it hanging for 20 seconds. `-25300` is not found; `-25293` and `-25308`
      are not allowed.
- [ ] `keychain_darwin.s` holds only the trampolines, one `JMP` each, and
      builds for both darwin arches.
- [ ] `store_darwin.go`: the default store is the Keychain store with the
      real bridge, except in a test binary (`testing.Testing()`), where it is
      the file store under `GDOC_CONFIG_DIR` exactly as every test runs today.
      The bridge itself refuses in a test binary unless `GDOC_LIVE_KEYCHAIN=1`,
      and makes no call. `store_other.go` (`//go:build !darwin`): the file store.
- [ ] Tests: `TestTheSuiteNeverTouchesTheKeychain` (in a test binary the store
      in use says `file`; on darwin a bridge call without the live variable
      returns the refusal; the boundary half below),
      `TestOnlyTheKeychainBridgeUsesLinkname` (every `.go` file under `go/`
      parsed for `//go:linkname` and `//go:cgo_import_dynamic` comments: only
      `internal/auth/keychain_darwin.go`), `TestOnlyTheStoreChoiceAsksIfThisIsATest`
      (one `testing.Testing` call in non-test code, in `store_darwin.go`).
- [ ] Checks run by the task: `GOOS=darwin go vet ./...`,
      `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./cmd/gdoc` and the same
      for amd64, `go build ./...` on this machine. No built binary is run.
- [ ] `cmd/gdoc/doc.go` "GDOC_CONFIG_DIR moves the config dir" says the token
      itself is not in it on darwin, and the test binary keeps it there.
- [ ] `git commit -m "feat(auth): the Keychain bridge, interaction off, CGO still off"`

### Task 5: one opt-in live test of the real Keychain

**Files:**
- Create: `go/internal/live/keychain_test.go` (`//go:build darwin`)
- Modify: `go/internal/live/doc.go`, `go/internal/auth/store_darwin.go`
  (exports `KeychainStore(service string) Store` for this test)

- [ ] `GDOC_LIVE_KEYCHAIN=1` alone runs it. Service `gdoc-live-test`, never
      `gdoc`. It saves a throwaway token, reads it back with its saved instant,
      deletes it, reads again and gets no token, and deletes the item in a
      `t.Cleanup` whatever happened.
- [ ] `TestTheLiveKeychainKeepsAndDeletesAThrowawayItem`, and
      `TestTheLiveKeychainTestNeedsItsOwnVariable` (skips without it, under
      `GDOC_LIVE_TEST=1` too).
- [ ] `live/doc.go` names the variable and the test.
- [ ] `git commit -m "test(live): the Keychain, opt in, on its own service"`

### Task 6: a stable release is a draft until Nail signs it

**Files:**
- Modify: `.github/workflows/release.yml`, `go/boundary/workflows_test.go`,
  `go/boundary/doc.go`

- [ ] The publish step passes `--draft` when the tag ends in `.0`, and
      publishes a nightly as today. The notes step is unchanged.
- [ ] Tests: `TestAStableReleaseIsADraftAndANightlyIsNot` (reads the publish
      step), `TestTheReleaseNeverSigns` (no `codesign`, no certificate secret
      in either workflow: the key never goes to GitHub).
- [ ] `git commit -m "chore(release): a stable release waits as a draft for its signature"`

### Task 7: make sign-release, and a signed make build

The Makefile and `release/` are outside `go/`, so neither rule about external
programs or git applies; the script runs `gh`, `codesign`, `shasum`, `unzip`
and `zip`, and no git.

**Files:**
- Create: `release/sign.sh`
- Modify: `Makefile`, `go/boundary/makefile_test.go`, `go/boundary/release_test.go`,
  `go/boundary/doc.go`

- [ ] `make sign-release TAG=vX.Y.0` runs `release/sign.sh "$(TAG)"`. The
      script refuses, before it changes anything: a tag that is not `vX.Y.0`;
      a release that is not a draft; a machine that is not darwin;
      `GDOC_SIGN_IDENTITY` unset or not found by `security find-identity -v
      -p codesigning`. `GDOC_SIGN_IDENTITY` is the certificate's SHA-1 hash,
      because gdoc's certificate and Altery's carry the same name.
- [ ] It downloads the darwin zips and `SHA256SUMS-<tag>` with `gh release
      download`, checks them against the sums, then for each zip: unzip into
      a temp folder, `codesign --force --sign "$GDOC_SIGN_IDENTITY" --options
      runtime --identifier com.altery.gdoc --timestamp gdoc`, `codesign
      --verify --strict`, and `codesign -dv` must say `Identifier=com.altery.gdoc`
      and `TeamIdentifier=8FGPRTH647`. It zips the folder again flat, as the
      workflow does, and writes `SHA256SUMS-<tag>` again over every zip in the
      release in the workflow's format.
- [ ] It uploads with `gh release upload --clobber`, then `gh release edit
      <tag> --draft=false`, and prints the tag and both new sums.
- [ ] `make build` signs `bin/gdoc` the same way when `GDOC_SIGN_IDENTITY` is
      set and found, and prints one line either way: "signed as
      com.altery.gdoc" or "not signed: GDOC_SIGN_IDENTITY is not set, so each
      rebuild signs you in again". On a machine with no `codesign` it says it
      did not sign. `make dist` signs nothing.
- [ ] Tests (text of the files): `TestTheSigningScriptUsesTheIdentifierAndTheHardenedRuntime`,
      `TestTheSigningScriptRefusesANightly`,
      `TestTheSigningScriptChecksTheSumsBeforeItSigns`,
      `TestTheSigningScriptNamesTheCertificateByItsHash`,
      `TestABuildSaysWhetherItIsSigned`, `TestDistSignsNothing`.
- [ ] `bash -n release/sign.sh` passes; `make -n sign-release TAG=v2.12.0`
      prints the script call.
- [ ] `git commit -m "chore(build): make sign-release, and make build signs when it can"`

### Task 8: the documents of Part C

**Files:**
- Modify: `README.md`, `docs/guide/how-it-works.md`,
  `docs/guide/from-a-checkout.md`, `CLAUDE.md`, `go/internal/auth/doc.go`,
  `go/cmd/gdoc/doc.go`, `docs/v2/PLAN.md`

- [ ] `README.md`, at 200 lines: after updating to v2.12.0 on a Mac, sign in
      once more; the token is in the Keychain now; a nightly signs you in again
      after each update.
- [ ] `how-it-works.md`: where the token lives per platform, and that
      `auth status` says `token_store`. `from-a-checkout.md`:
      `GDOC_SIGN_IDENTITY` and what an unsigned `make build` costs.
- [ ] `CLAUDE.md`, under 300 lines, for each line M16's Task 1 did not
      already change: "Secrets live in" names the Keychain on macOS; the thread-mark
      invariant names the mark in effect; the `/usr/bin/open` exception in
      `cmd/gdoc/desktop.go` is named in "Nothing under `go/` runs an external
      program", which the spec found missing; the Keychain bridge is named as
      the one `go:linkname` file with `TestOnlyTheKeychainBridgeUsesLinkname`.
- [ ] `auth/doc.go`: the store, the item, the first-run deletion, the
      interaction switch and the test-binary rule, each with its test.
      "Save carries three fields" and "The token file is google-auth's shape"
      say they hold for the file store and for what the item holds.
- [ ] `PLAN.md`: M17 under Done, with this plan's path. Move this plan and its
      spec to `docs/plans/completed/`, fixing every link to both.
- [ ] `cd go && go test -race ./boundary/` passes.
- [ ] `git commit -m "docs(v2): M17, completed"`

### Task 9: verify acceptance

- [ ] `make test`, `make vet`, `GOOS=darwin go vet ./...` and `make dist` pass.
- [ ] Every Validation Command gives the answer it states; record the counts
      here as ➕ notes.
- [ ] Every decision in the spec names a task above that built it.
- [ ] `git diff main...HEAD -- go/internal/guard/policy.go` is empty.
- [ ] Nothing to commit unless a ➕ note was added.

## Post-Completion

**Before v2.12.0 is tagged, by Nail:**


1. **The live Keychain test:** `cd go && GDOC_LIVE_KEYCHAIN=1 go test
   ./internal/live -run Keychain -v`.
2. **His own build:** `GDOC_SIGN_IDENTITY=<hash> make build`, then `gdoc auth
   status` deletes his `oauth-token.json` and asks him to sign in; sign in;
   `token_store: keychain`; `make build` again and `gdoc auth status` still
   signed in. Then the same through Claude Desktop's `gdoc mcp`.
3. **The tag sitting**, in one sitting with `make tag VERSION=v2.12.0`, because
   a skill that stops typing 🤖 is refused by v2.10.0: `skills/gdoc-review/`
   (`SKILL.md`, `review.md`, `live.md`, `propose.md`) and
   `skills/gdoc-align/SKILL.md` stop telling a session to type 🤖 and say gdoc
   adds the mark; `go/cmd/gdoc/review.md` is copied from the skill's again
   (`TestTheEmbeddedCoreIsTheSkillsCore`); every skill's `needs` line moves to
   v2.12.0. v2.12.0 still accepts a body opening with a known mark, with a
   warning, so the order is safe either way for one release.
4. **Sign and publish:** the release workflow leaves v2.12.0 as a draft; Nail
   runs `make sign-release TAG=v2.12.0`.
