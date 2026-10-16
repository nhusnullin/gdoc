# gdoc v2 Milestone 22: the logo lives in Drive, not in the tree, plan

2026-10-17. The task list for the specification beside it,
`docs/plans/2026-10-17-gdoc-v2-m22-the-logo-lives-in-drive-spec.md`, whose
five decisions are Nail's and whose design rules are listed there for him to
strike. One commit per task. The spec holds the rules. This plan repeats none
of them. Where the code, the spec or this plan disagree, the task stops and
says so rather than choosing. It starts from `main` after M20 merged.

## Principles

Serves 3 and strains 1 in one place, as the spec says.

## Context (from discovery)

- `go/internal/house/house.yaml:1147`: the `logo:` block holds `mime`, `px`,
  `width_pt`, `height_pt`, `anchor` and `base64`. `house.go:516` is the
  `Logo` struct, `:572` `LogoBytes` decodes the base64 and nothing else,
  `:884` `validateLogo` requires `image/png`, two `px` values and a decodable
  PNG of the stated size, and `Validate` ends by calling it, so an absent logo
  is refused today. `TestTheLogoDecodesAsAPNGOfTheStatedSize` (`house_test.go:227`)
  and `TestALogoThatIsNotAPNGIsRefused` (`:496`) pin that.
  `TestNothingHereReachesTheNetwork` pins that this package reaches nothing.
  `LoadFile` (`house.go:541`) replaces the embedded file for one run, which is
  `--house` on `build` and `publish` (`cmd/gdoc/build.go:244` `readHouse`).
- `go/internal/render/render.go:122` calls `cfg.LogoBytes()` and `:141`
  always writes `word/media/logo.png`; `headfoot.go:123` `logoDrawing`
  anchors it and fails on a wrap other than `square`.
  `TestBuildWritesTheThirteenPartsAndTheLogo` (`render_test.go:73`),
  `TestTheLogoPartIsThePNGTheConfigCarries` (`:120`) and
  `TestTheFirstPageHeaderCarriesTheLogoWhereTheHouseAnchorsIt`
  (`headfoot_test.go:8`) read the embedded bytes.
- `go/internal/guard/policy.go:721`: `aboutPath`, `accountFields`,
  `AllowAccountRead`, `mayReadAccount`, `isAccountRead`, and `judgeDrive`
  asking it first. That is the shape the seventh grant copies.
  `params.go:72`, `:98`, `:142`, `:144` already admit `alt` and
  `supportsAllDrives` on `files.get`. `about_test.go` holds the three pins of
  the sixth grant. `go/boundary/account_test.go:34`
  `TestOnlyAccountOfCallsAllowAccountRead` and `:48`
  `TestAccountOfHasTwoCallers` hold its callers.
- `go/internal/config/config.go:66-115`: `TokenPath`, `LastCheckPath`,
  `LoginPendingPath`, `SettingsPath`, each one file in the config dir.
  `go/internal/lastcheck/doc.go`: the stamp, strict read, every refusal is
  stale rather than fatal, `Interval` is a day stated as a literal.
  `go/internal/atomicfile`: the one room that replaces a file's contents.
- `go/internal/update`: the one package today handed a client for a GET of
  bytes it then checks; its tests hand a client over a fake transport, and
  the asset room's tests do the same.
- `go/cmd/gdoc/build.go:49` and `publish.go:55` carry `House` in the
  envelope; `build.go:145` `readHouse`. `doc.go:72` says `build` is "no
  network at all". `export.go:121`, `helpmarks.go:68` and `restyle.go:321`
  call `house.Load()` and must not reach the asset room.
- `go/internal/drift/items.go:152-159`: seven logo items, presence, mode,
  layout, width, height, left and top offset, none a pixel. `drift_test.go:104`
  reads the master's logo geometry. `go/internal/drift/testdata/master.docx`
  carries the real PNG as `word/media/image1.png`, 5468 bytes, and two
  harmless diagrams. `go/internal/live/testdata/footnote-probe.docx` and
  `mark-probe.docx` carry it as `word/media/logo.png`. The PNG is 195 by 97
  pixels, RGBA. Its SHA-256 is `a0c4fc99472c7a8fb069d387b8755dd73f622c2ce261c35a34871e5f1af2b3ef`.
- `go/boundary/domain_test.go:19` `companyHostSHA256`, `:391`
  `TestNoCompanyDomainInTheTree`, and the scanner tests from `:274`, which
  read `.gitignore` themselves. The scan reads text; the logo scan has to
  open each docx as a zip and hash its members.
- `go/internal/restyle/readback.go:128`: `"the first-page header carrying the
  Altery logo"`. `skills/gdoc-publish/SKILL.md:316`,
  `skills/gdoc-restyle/SKILL.md:220`, `docs/guide/publishing.md:20`, `:164`,
  `docs/guide/restyle.md:147` name the logo. `go/internal/live/publish_test.go`
  asserts the logo around line 280.
- The firm's file: `logo.png` in the root of a Shared Drive, id
  `1J3uh6poVAneP_V_Pc5FuNI4-Gahlab0s`, shared to the domain as reader,
  verified byte-identical on 2026-10-16.
- `CLAUDE.md` names the six grants and the two doors in its invariants and
  `go/internal/` rooms in its table; `TestCLAUDEmdIsUnderTheCeiling` holds it
  under 300 lines.

## Development Approach

TDD, one commit per task, the test first and failing. Every id, every hash
and every size a test states is a literal. `make test` keeps the race flag
and stays offline: no test under `go/` reaches Drive except under
`GDOC_LIVE_TEST`. A task that finds the spec and the code disagree stops and
says so in the task log. The order keeps every commit green: the embedded
file keeps its `base64` until Task 6, when the drive id replaces it in the
same commit as the placeholder and the boundary test.

## Validation Commands

```bash
make test
make vet
cd go && go test ./internal/house ./internal/asset ./internal/guard -run 'Logo|Asset|Drive' -v
cd go && go test ./cmd/gdoc ./boundary -run 'Logo|Asset' -v
cd go && GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1 go test ./internal/live -run 'HouseLogo|Publish' -v
```

## Progress Tracking

- [x] Task 1: `logo:` names exactly one of `drive` and `base64`
- [x] Task 2: the seventh grant, `AllowAsset`
- [x] Task 3: the asset room, its cache and its stamp
- [x] Task 4: `build` and `publish` read the logo through the room, and say where it came from
- [ ] Task 5: a placeholder, and the tests that read the embedded bytes read it instead
- [ ] Task 6: the tree carries no logo
- [ ] Task 7: the words
- [ ] Task 8: the live measurement
- [ ] Task 9: the acceptance table

## Tasks

### Task 1: `logo:` names exactly one of `drive` and `base64`

`TestALogoNamesExactlyOneOfDriveAndBase64`: a file with both is refused
naming `logo`, and a file with neither is refused the same way.
`TestADriveIdThatIsNotOneIsRefusedByName`: `drive: ""`, `drive: "a/b"` and
`drive: "x"` are each refused naming `logo.drive`; an id of the shape Drive
mints, 25 or more of `[A-Za-z0-9_-]`, passes. `TestADriveLogoCarriesNoBytes`:
with `drive` set, `Validate` passes without decoding a PNG, and the sizes and
the anchor are still required. `Logo` gains `Drive string`, `validateLogo`
checks the id's shape when `drive` is set and the PNG when `base64` is,
`LogoBytes` returns the base64 bytes or an error naming that this house's
logo is in Drive. The embedded file is unchanged in this task, so every test
that reads it stays green. `house.go`'s comment says why the PNG check leaves
this package when the bytes do.

### Task 2: the seventh grant, `AllowAsset`

In `about_test.go`'s shape, `asset_test.go`:
`TestTheAssetGrantAdmitsOneMediaReadAndNothingElse`: after `AllowAsset(id)`,
`GET /drive/v3/files/{id}?alt=media&supportsAllDrives=true` passes, and the
same path without `alt=media`, with `fields`, with any other method, a
`/copy` under it, a Docs read of the id, and a second id are each refused.
`TestWithoutTheAssetGrantTheReadIsRefused`: a fresh policy refuses the read
by name. `TestTheAssetGrantAdmitsNoDocument`: a policy with only the grant
reaches no document and `documents.get` on the id is refused.
`TestTheAssetGrantDiesWithThePolicy`: a second policy knows nothing of it.
`Policy.AllowAsset(id string)`, `isAssetRead`, asked in `judgeDrive` beside
`isAccountRead`. `guard/doc.go` gains the seventh bullet, naming the third
door and why a PNG cannot grow the set, and its heading at line 42 and the
sentence at line 50 stop saying six.

### Task 3: the asset room, its cache and its stamp

`go/internal/asset`, with `doc.go`. `config.AssetPath(id)` and
`config.AssetStampPath(id)` under `assets/` in the config dir, the directory
made on first write. `Fetch(ctx, client, id, want) (Bytes, error)` where
`want` is the house's statement of mime and pixels and `Bytes` carries the
bytes, `Source` (`drive` or `cache`), `FetchedAt`, `Age` and `Reason`. Tests
over a fake transport: `TestAStaleStampFetchesOnce` (missing, unreadable and
25 hours old each make one request), `TestAFreshStampMakesNoRequest`,
`TestNoTokenAndACacheUsesTheCache`, `TestNoTokenAndNoCacheRefusesNamingTheLogin`,
`TestARefusedReadUsesTheCacheAndNamesTheReason` (403, 404, a dial error and a
timeout each use the cache with the status or the error as `Reason`),
`TestABadFileRefusesEvenWithAGoodCache` (not a PNG, wrong pixels, wrong mime
each refuse and leave the cache untouched),
`TestACacheWriteFailureStillBuilds` (an unwritable directory returns the
bytes with the failure named), `TestTheCacheIsWrittenAtomically` (through
`atomicfile`, 0o600). The stamp is `lastcheck`'s shape with its own keys,
`fetched_at` and `sha256`, read strictly, every refusal stale. The room
imports `net/http` for the client type only; `go/boundary`'s allowlists gain
it on the import side and not on the builder side.

### Task 4: `build` and `publish` read the logo through the room, and say where it came from

`TestBuildReportsTheLogoSource` in `cmd/gdoc`: with a house whose logo is
`base64`, the envelope carries `logo.source: "file"` and no request is made;
with a `drive` logo and a fake transport, `source: "drive"`, `id` and
`fetched_at`; with the transport refusing and a cache present, `source:
"cache"`, `age` and `reason`. `TestBuildWithNoTokenAndNoCacheRefusesNamingTheLogin`.
`TestPublishReportsTheLogoSource` the same over `publish`'s envelope.
`render.Build` takes the logo bytes from the caller instead of calling
`cfg.LogoBytes()`, and `readHouse` grows into `readHouseAndLogo`, which
builds a policy with the one grant, a client through `guard.NewClient`, and
calls the room only when the house names a `drive` logo. In `go/boundary`,
`TestOnlyBuildAndPublishCallTheAssetRoom` and
`TestReadExportRestyleAndHelpNeverReachTheAssetRoom` read the syntax trees
as the account tests do. `TestReadNeverReachesTheCheck`'s family gains the
asset room. `doc.go`'s `build` line says "no network beyond the house logo,
once a day".

### Task 5: a placeholder, and the tests that read the embedded bytes read it instead

A neutral PNG of 195 by 97 pixels, RGBA, a flat light grey, written once by a
small Go program under `go/internal/house/testdata/` and committed as
`placeholder-logo.png` with its base64 in `testdata/house-base64.yaml`, a
copy of the embedded file whose `logo` carries that base64 in place of the
firm's. `TestThePlaceholderIsAPNGOfTheHouseSize` pins 195 by 97.
`TestTheLogoDecodesAsAPNGOfTheStatedSize` reads `house-base64.yaml`;
`TestTheEmbeddedLogoIsADriveIdAndCarriesNoBytes` is written now and marked
`(fails until Task 6)`. `render`'s three logo tests and `drift_test.go:104`
are handed the placeholder bytes. The embedded file is still unchanged.

### Task 6: the tree carries no logo

`TestTheLogoIsNotInTheTree` in `go/boundary`, beside the domain test: holds
the real PNG's SHA-256 as a literal, walks every file git does not ignore
with the domain scan's own walker, hashes each file, and for a `.docx` opens
it as a zip and hashes every member; any match fails naming the file and the
member. `TestTheLogoScanFindsAPlantedPNG` plants the bytes in a temp tree,
once bare and once inside a zip, before the scan is trusted to find nothing.
Then, in the same commit: `house.yaml`'s `logo` loses `base64` and gains
`drive: 1J3uh6poVAneP_V_Pc5FuNI4-Gahlab0s`; the three docx are re-zipped
with the placeholder in place of the real PNG, keeping every other member
byte for byte, by a one-off script run by hand and not committed; and
`TestTheEmbeddedLogoIsADriveIdAndCarriesNoBytes` goes green. `drift`'s
offline gate must pass unchanged, because its seven logo items read
geometry; if one reads a pixel, the task stops and says which.

### Task 7: the words

`restyle/readback.go:128` says "the first-page header carrying the house
logo", and `TestTheManualStepsNameTheMenuPathForEach` states the new literal.
`skills/gdoc-publish/SKILL.md`, `skills/gdoc-restyle/SKILL.md`,
`docs/guide/publishing.md` and `docs/guide/restyle.md` each say in one line
that the house logo is one file in the firm's Drive, fetched once a day and
cached beside the token, and that a run that cannot read it refuses. The
README gains one line: the firm's logo and marks are the firm's and are not
licensed with the code. `CLAUDE.md`: the reachable-set bullet names the third
door, the grants bullet names `AllowAsset`, the table gains
`go/internal/asset/`, and the task map gains a row for the logo; the file
stays under 300 lines. `go/boundary`'s guide gates learn the new lines if
they pin them.

### Task 8: the live measurement

`TestPublishCarriesTheHouseLogo` in `internal/live`, under `GDOC_LIVE_TEST=1
GDOC_LIVE_WRITE=1`: `build` with the embedded house and the real token, in a
temp config dir with no cache, makes one request to Drive, and the
`word/media/logo.png` in the docx hashes to the literal the boundary test
holds; a second `build` in the same config dir makes no request. The existing
live publish assertion on the logo reads the same hash. If the read needs a
parameter the guard does not admit, stop and write what Drive answered into
the task log for Nail.

### Task 9: the acceptance table

One row per rule in the spec, pass or the test that says otherwise, written
into the spec file. Both plan files move to `docs/plans/completed/`.
