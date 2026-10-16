# gdoc v2 Milestone 22: the logo lives in Drive, not in the tree. Specification

2026-10-17. Nail's decision, taken in a brainstorm on 2026-10-16 that put the
question to three readers: a Google Workspace API researcher, a UK brand
counsel, and Codex for a fresh read of the code against the principles. The
repository is public, and the firm's logo is a base64 PNG in `house.yaml`,
embedded in every release and written into every document gdoc builds,
including one built by a stranger who installs the tool. After this milestone
the tree carries no logo. `house.yaml` names the firm's logo by the id of one
file in the firm's Shared Drive, and only a run signed in as somebody that
file is shared with gets it. The decision entry is in `docs/v2/DECISIONS.md`
under 2026-10-17, and the task list is
`docs/plans/completed/2026-10-17-gdoc-v2-m22-the-logo-lives-in-drive.md`. It
starts from `main` after M20 merged.

## Principles

Serves: 3, uncertainty never resolves toward the destructive answer, because
the one door this opens admits one GET on one id and nothing else, a file
gdoc half understands still never reaches a document, and a run that cannot
read the logo refuses rather than shipping a document in the firm's name
without its mark. And the 2026-10-02 decision that the firm appears nowhere in
the tree: the logo joins the domain.

Strains: 1, if it is not inside the binary it does not travel. The logo no
longer travels with the binary. It travels with the firm's sign-in, is cached
beside the token, and a `build` on a machine that has never signed in and
holds no cache refuses. Nail accepted that on 2026-10-16: a colleague has a
token before the first build, because every skill asks for one, and a
stranger's build refusing is the point. The reachable set gains a third door
for one object that is not a document, which the decision entry names.

## What was found before deciding

- No Google API returns a Workspace organisation's logo. The Admin console
  logo is upload-only; Directory `customers.get` returns nine fields and none
  is a picture; Cloud Identity has no branding resource; the template gallery
  is not listable. The only two readable places are a file in Drive by id and
  an inline image inside a Doc, whose `contentUri` lives for thirty minutes on
  a host the guard refuses.
- The OAuth client is User type Internal, so only an account in the firm's
  Workspace can sign in. `publish` and `restyle` were already gated. What was
  exposed is `build`, which writes a branded docx with no sign-in, and the
  bytes themselves: `house.yaml`, three test docx and 28 release tags.
- The legal risk of the public PNG is low; the practical concern is a stranger
  producing a branded document. The repository has no licence file, so no code
  licence covers the logo by accident. A brand-exclusion line is cheap.
- The code assumes a logo in six places: `house.Validate`, the media part
  `render` always writes, `drift`'s seven logo items, `restyle`'s manual step
  naming "the Altery logo", the live publish test, and the render and house
  tests that pin it.

## Decisions taken

Nail, 2026-10-16, in chat:

| Rule | Where |
|---|---|
| `house.yaml` stays in the repository with the sizes, the styles, the palette and the cover. Only the logo moves to Drive | house |
| The logo is one PNG in the firm's Shared Drive, shared to the domain as reader. `house.yaml` names it as `logo.drive`, the file id, beside the sizes | house |
| The bytes are fetched at render, by `build` and `publish` only, through a cache beside the token with a stamp: one request a day, not one a run | asset, cmd |
| No cache and no access refuses the run. It never falls back to an unbranded document | asset, cmd |
| A file in Drive that is not a PNG of the stated size refuses the run even when a good cache exists, because the firm's file is wrong and the refusal is the only message that reaches whoever can fix it | asset |

Taken in the design pass by Claude, each one a sentence Nail can strike:

| Rule | Where |
|---|---|
| `logo:` carries exactly one of `drive` and `base64`. Both or neither is refused naming the key. `base64` stays for a test house and for a firm that prefers embedding | house |
| `house` still reaches nothing. It parses `logo.drive` and checks the id's shape; the PNG check moves to the room that has the bytes | house, asset |
| One new room, `internal/asset`: one Drive file by id, read as bytes, checked against what the house says of it, cached. Handed a client, never builds one, and knows no logo as such, so test data later is the same call with another id | asset |
| One new grant, `AllowAsset(id)`: per run, dies with the process, nowhere written down. Admits `GET /drive/v3/files/{id}` with `alt=media` and `supportsAllDrives=true`, both already in the guard's parameter list for `files.get`, and nothing else on that id: no metadata read, no Docs read, no patch, no copy. It is the seventh grant and the third door into the reachable set, and the set cannot grow through it because a PNG carries no id | guard |
| The cache is `assets/<id>` in the config dir, written through `internal/atomicfile`, with a stamp beside it in the shape `internal/lastcheck` writes. A stamp older than a day, missing or unreadable is stale and the run asks Drive once. A fresh stamp makes no request | asset, config |
| Online, the bytes are written to the cache when they differ. Offline, refused or timed out, the cache is used and the envelope says so with its age and the reason. No token and a cache is the same. No token and no cache refuses naming `gdoc login` | asset, cmd |
| A cache that cannot be written still builds the document from the fetched bytes, and the envelope says the cache was not saved | asset |
| `build` and `publish` print a `logo` block of facts: `source` is `drive`, `cache` or `file`, with `id` and `fetched_at` or `age` and `reason` as apply. In chat the person hears about the logo only when it came from the cache | cmd, chat |
| `export`, `restyle` and `help marks` read `house.yaml` for other reasons and never touch the asset room, so `read` still makes no request. `restyle`'s manual step says "the house logo" | cmd, restyle |
| The tree carries no logo: `house.yaml` loses `base64`, and the three test docx carry a neutral placeholder PNG of 195 by 97 pixels, so drift's seven logo items still measure. A boundary test holds the real PNG's SHA-256 the way the domain test holds the domain's, walks every tracked file and inside every docx, and fails on a match | house, drift, live, boundary |
| History and the 28 tags are not rewritten. Old binaries carry the old logo and that is written down as accepted | docs |
| The README gains one line: the firm's logo and marks are not licensed with the code. There is no licence file, and adding one is a separate decision | docs |
| The skills, the guide and the help say "the house logo", in one line each | skills, docs, cmd |

## What is not built

A folder of assets found by listing: Drive has no file by path and `files.list`
stays refused, so every asset is its own id. A settings override for the id:
`--house <file>` with another `logo.drive` or a `base64` is the override. A
fetch from `login`, from the daily release check, from the installer, or from
a command of its own. A metadata read for the file's revision: the bytes are
five kilobytes and a day apart. A rewrite of history.

## Measurement

One, live: `files.get` with `alt=media` and `supportsAllDrives=true` on the
firm's file, signed with the token, returns the PNG whose SHA-256 is the one
the boundary test holds, under the `drive` scope the token already carries.
Measured by hand on 2026-10-16 through the Drive connector, byte for byte,
and `TestPublishCarriesTheHouseLogo` in `internal/live` repeats it under
`GDOC_LIVE_WRITE`. If the file read needs a parameter the guard does not
admit on `files.get`, the task stops and Nail decides.

## Acceptance

Filled in on 2026-10-17, when the run finished. One row per rule above. Every
row names the test that holds it, and a row whose rule is held by a document
rather than by a test says so. `make test` and `make vet` are clean, and the
live measurement ran against real Drive on 2026-10-17.

Nail's five:

| Rule | Verdict | Held by |
|---|---|---|
| `house.yaml` keeps the sizes, the styles, the palette and the cover. Only the logo moves | pass | `TestLoadReadsTheEmbeddedFile`, `TestPageIsA4InPoints`, `TestTheTenStylesAreThere`, `TestFrontMatterIsFourteenBlocksInOrder`, all against the shipped file, and `TestTheEmbeddedLogoIsADriveIdAndCarriesNoBytes` |
| The logo is one PNG in the firm's Shared Drive, named as `logo.drive` beside the sizes | pass | `TestTheEmbeddedLogoIsADriveIdAndCarriesNoBytes`, `TestADriveLogoCarriesNoBytes`, and `house.yaml`'s `logo.drive` |
| Fetched at render by `build` and `publish` only, through a cache with a stamp, one request a day | pass | `TestOnlyBuildAndPublishCallTheAssetRoom`, `TestReadExportRestyleAndHelpNeverReachTheAssetRoom`, `TestBuildWithAFreshCacheAsksDriveNothing`, `TestAFreshStampMakesNoRequest`, `TestAStaleStampFetchesOnce`, `TestTheIntervalIsADay` |
| No cache and no access refuses the run, and never falls back to an unbranded document | pass | `TestNoTokenAndNoCacheRefusesNamingTheLogin` in `internal/asset` and `TestBuildWithNoTokenAndNoCacheRefusesNamingTheLogin` in `cmd/gdoc`, which also holds that no file was written |
| A file in Drive that is not a PNG of the stated size refuses even with a good cache | pass | `TestABadFileRefusesEvenWithAGoodCache`, three cases, and `TestBuildRefusesAWrongFileInDriveWithAGoodCache` |

The design pass, each one Nail can still strike:

| Rule | Verdict | Held by |
|---|---|---|
| `logo:` carries exactly one of `drive` and `base64`, and both or neither is refused naming the key | pass | `TestALogoNamesExactlyOneOfDriveAndBase64`, `TestADriveIdThatIsNotOneIsRefusedByName` |
| `house` still reaches nothing. It checks the id's shape, and the PNG check moves to the room with the bytes | pass | `TestNothingHereReachesTheNetwork`, `TestADriveLogoCarriesNoBytes`, `TestALogoThatIsNotAPNGIsRefused` for the `base64` route, and `hold` in `internal/asset` for the Drive route |
| One room, `internal/asset`: one file by id, checked against what the house says, cached, handed a client and never building one, knowing no logo as such | pass, stronger than written | `TestOnlyTheGuardBuildsTheWire` and `TestNetHTTPStaysInItsRooms` unchanged: the room names no `net/http` type at all and takes a `Drive` interface, so neither allowlist grew. `Want` carries a mime and two pixel counts and the word logo appears nowhere in the room's API |
| One grant, `AllowAsset(id)`: one media read on one id, per run, nowhere written down, and the set cannot grow through it | pass | `TestTheAssetGrantAdmitsOneMediaReadAndNothingElse`, `TestWithoutTheAssetGrantTheReadIsRefused`, `TestTheAssetGrantAdmitsNoDocument`, `TestTheAssetGrantDiesWithThePolicy`, `TestAnEmptyAssetGrantOpensNothing`, and `TestTheLogoFetchOpensTheAssetGrantAndNothingElse` over the command |
| The cache is `assets/<id>` with a stamp beside it, written through `internal/atomicfile`. Missing, unreadable or older than a day is stale | pass | `TestAnAssetAndItsStampSitUnderAssets`, `TestAnAssetPathRefusesAnIdThatIsNotOne`, `TestTheCacheIsWrittenAtomically`, `TestTheStampIsOneLineAPersonCanRead`, `TestAStaleStampFetchesOnce` (eight cases), `TestAFreshStampMakesNoRequest` |
| Written to the cache when the bytes differ. Refused, offline or no token uses the cache with its age and reason. No token and no cache refuses naming the login | pass | `TestUnchangedBytesLeaveTheCacheFileAlone`, `TestARefusedReadUsesTheCacheAndNamesTheReason` (403, 404, a dial error, a timeout), `TestNoTokenAndACacheUsesTheCache`, `TestNoTokenAndNoCacheRefusesNamingTheLogin` |
| A cache that cannot be written still builds, and the envelope says it was not saved | pass | `TestACacheWriteFailureStillBuilds`, and `logo.cache_error` in the envelope |
| `build` and `publish` print a `logo` block of facts, and in chat the person hears about the logo only when it came from the cache | pass, and the chat half is narrower than this sentence | `TestBuildReportsTheLogoSourceFile`, `TestBuildReportsTheLogoSourceDrive`, `TestBuildReportsTheLogoSourceCache`, `TestPublishReportsTheLogoSource`, `TestPublishReportsALogoInTheFile`. See the two notes below |
| `export`, `restyle` and `help marks` never touch the asset room, so `read` still makes no request, and `restyle`'s manual step says "the house logo" | pass | `TestReadExportRestyleAndHelpNeverReachTheAssetRoom`, `TestReadNeverFetchesTheLogo`, `TestTheManualStepsNameTheMenuPathForEach` with the new literal |
| The tree carries no logo. The three docx carry a placeholder of 195 by 97 pixels, so drift's seven logo items still measure | pass | `TestTheLogoIsNotInTheTree` with the real PNG's SHA-256 as a literal, proved by `TestTheLogoScanFindsAPlantedPNG` first, bare and inside a zip. `TestThePlaceholderIsAPNGOfTheHouseSize`, and `TestFromDocxReadsTheMaster` and `TestTheOfflineGate` unchanged: every logo item is geometry and none is a pixel |
| History and the 28 tags are not rewritten, and that is written down as accepted | pass, a document and not a test | `docs/v2/DECISIONS.md`, 2026-10-17, which also marks the 2026-09-16 entry's logo assessment superseded |
| The README gains one line: the firm's logo and marks are not licensed with the code | pass | `TestTheREADMESaysTheLogoIsNotLicensedWithTheCode` |
| The skills, the guide and the help say "the house logo", one line each | pass | `TestTheWordsSayWhereTheLogoLives` over `skills/gdoc-publish/SKILL.md`, `skills/gdoc-restyle/SKILL.md`, `docs/guide/publishing.md` and `docs/guide/restyle.md`, and `TestTheManualStepsNameTheMenuPathForEach` for the help |

The measurement:

| Claim | Verdict | Held by |
|---|---|---|
| `files.get` with `alt=media` and `supportsAllDrives=true` on the firm's file returns the PNG whose SHA-256 the boundary test holds, under the scope the token already carries | pass, measured live on 2026-10-17 | `TestPublishCarriesTheHouseLogo` under `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1`: one request on the first build in a fresh config directory, none on the second, 5468 bytes, 195 by 97, the hash matched. No parameter outside the guard's list was needed, so nothing stopped for Nail |

Two notes, both narrowings the run chose and neither a rule it broke:

- The chat half of the envelope rule is vacuous as written and narrower in the
  code. Chat has six tools, `read`, `comments`, `suggestions`, `reply`,
  `annotate` and `propose`, and none of them renders a note, so no logo reaches
  chat at all. What a person reads is the envelope's `warnings`, and the run
  writes a line there in two cases rather than one: the cache used because
  Drive did not answer, and bytes fetched that could not be cached. A cache
  used because Drive was already asked today is the designed path and earns no
  line, although the envelope still says `source: "cache"`. The reason is in
  `cmd/gdoc/logo.go`'s comment on `logoFor`. Nail can widen it to every cache
  read in one line.
- `TestPublishCarriesTheHouseLogo` asks for both live variables although it
  creates nothing in Drive. It reads the firm's own file and writes a cache on
  this machine, which is why, and `internal/live/doc.go` says so. The rest of
  that package's rule is unchanged: a read needs only `GDOC_LIVE_TEST`.
