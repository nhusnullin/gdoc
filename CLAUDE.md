# Notes for AI assistants

Read [PRINCIPLES.md](PRINCIPLES.md) before proposing any design. It is four
constraints and the decisions that implement them, and it is short.

Then read the package comment of the package the task touches, in its `doc.go`
where there is one and in an ordinary source file otherwise. It holds that
package's rules, the reason for each, and the test that pins it. This file holds
only what is true across the whole tree: where things live, the invariants,
what to read for a task, how to build, and what never to do.

Two rules about the documents themselves:

- A change to `docs/v2/SPEC.md` is a `docs/v2/DECISIONS.md` entry first, written
  the same day, with its row in that file's register.
- A `doc.go` names the test for every rule it states, or carries
  `TODO(test)` where none exists yet.

## What lives where

| Path | Holds |
|---|---|
| `go/` | the binary. One Go module, three dependencies, `gdoc` on PATH |
| `go/cmd/gdoc/` | the entry point, the seventeen commands plus `help` and `completion`, and the one table that describes them. Arguments in, one JSON object out, exit. `mcp*.go` is the one command that is a session instead, and `desktop.go` the Claude Desktop extension |
| `go/internal/mcp/` | the Model Context Protocol over stdio, by hand: the framing, the four methods, the tools, the deadline. It knows no command |
| `go/internal/chat/` | what a chat adds to a command: the guide code, the labels and facts on a read, the ledger of this process, the hold rules, the card, the write memory |
| `go/internal/emit/` | the output envelope every command prints through |
| `go/internal/tty/`, `go/internal/panel/` | the terminal, its colours and width, and the boxes drawn on it |
| `go/internal/guard/` | the network policy, and the only place a client is built |
| `go/internal/asset/` | one file in Drive read as bytes, held to what the caller states, and cached beside the token. The house logo is its one caller |
| `go/internal/auth/`, `go/internal/auth/loopback/` | the token file, its refresh, the login flow, and the one-shot localhost listener the browser redirect lands on |
| `go/internal/config/` | where the per-user files live, per platform |
| `go/internal/settings/` | `settings.yaml`, the arguments over it, and the mark in effect |
| `go/internal/presence/` | the record of what gdoc is doing: the event, the month's file, the one append |
| `go/internal/gapi/` | the authenticated session. The one room that builds a request |
| `go/internal/docs/` | the Docs read: tabs, the document tree, suggestion ids, comment ranges |
| `go/internal/view/` | the document as the text `read` prints, and as the tree `--structure` prints |
| `go/internal/comments/` | Drive's threads joined to the Docs ranges, the cursor, and the poll |
| `go/internal/suggestions/` | what is pending, and what stopped being pending since the snapshot |
| `go/internal/docx/` | the docx export reader, and the witness match against threads |
| `go/internal/probe/` | the throwaway document that asks whether SUGGEST is honoured today |
| `go/internal/reply/` | one robot reply into a thread, and the thread read back |
| `go/internal/propose/`, `go/internal/withdraw/` | a change as a suggestion, its comment, and the three read-backs; and gdoc taking back one of its own pending proposals |
| `go/internal/annotate/` | a comment on the words a caller quotes, anchored, and nothing else |
| `go/internal/publish/` | the upload with conversion, and the three read-backs on what came out |
| `go/internal/export/`, `go/internal/markers/` | a Google Doc as Markdown in the hub: one file per tab, the house prelude taken out, the pictures beside it, and nothing replaced; and the eight openers gdoc writes into a file, so no route into a document carries one back |
| `go/internal/restyle/` | the survey, and the house look applied where the document stands |
| `go/internal/prelude/` | the house template as Docs requests, proposed rather than written, and the named range that marks it |
| `go/internal/docsreq/` | the shapes a Docs request is made of: a measurement, a colour, an alignment, a length, a style object with its mask |
| `go/internal/drive/` | the trash, its confirming read, and nothing else Drive does |
| `go/internal/plaintext/` | what gdoc may write into a comment thread: the mark, and no markdown |
| `go/internal/frontmatter/` | the `gdoc:` block in a note's YAML front matter, and nothing else in the file |
| `go/internal/house/` | the Altery house style as a parsed file, embedded in the binary |
| `go/internal/render/`, `go/internal/body/`, `go/internal/cover/` | the docx itself: the thirteen parts and the logo, the cover, the tables, the header and footer, the contents field; and the note's markdown walked with goldmark into the house style's own paragraphs, and the words that reach the cover and the running head |
| `go/internal/drift/` | the one list of measured values, read out of a docx and out of a Docs answer |
| `go/internal/atomicfile/` | the temp-file-and-rename write. The one room that replaces a file's contents |
| `go/internal/update/`, `go/internal/lastcheck/` | the releases GitHub lists, the version comparison, the channel policy, this binary replaced from a zip it checked, and the stamp beside the token of when gdoc last asked GitHub what is published |
| `go/internal/live/` | the opt-in end-to-end tests. Tests only, no production code |
| `go/boundary/` | the allowlist tests over the wire, the dependencies and these documents |
| `presence/` | the menu bar app, a Swift package of its own. The core that reads the record, the glyph, the panel, and the script that builds the bundle |
| `skills/` | `gdoc-review`, `gdoc-publish`, `gdoc-restyle`, `gdoc-export`, `gdoc-align`. Symlinked into `~/.claude/skills/`, so edits are live |
| `.claude-plugin/` | the plugin manifest and the marketplace entry, so this repository is how a colleague's Claude Code gets `skills/` |
| `release/` | what a colleague gets: the one-line installer, the example note, the Claude Desktop extension template under `mcpb/`, and the platform list a release walks. The README in the zip is this repository's `README.md` |
| `docs/guide/` | the colleague's pages the README links to: how gdoc stays inside the document, and each command walked through |
| `docs/v2/` | what v2 is, what is next, why, and what Google does |
| `docs/backlog/` | deferred work, one file per item |
| `docs/plans/` | the milestone plans. `completed/` holds the ones that ran |
| `bin/` | what `make build` and `make dist` write. Not in git, so both targets create it |

Secrets live in `~/.config/gdoc-agent/`, never in this repo.

## The invariants

Each is one sentence and names the test that holds it, except the two the
binary cannot measure: identity, a rule about what no field means, and the skill
links, which live in `install.sh`. Breaking one is a decision somebody writes
into `docs/v2/DECISIONS.md`, not a refactor.

- **The guard owns the wire.** `guard.NewClient` is the only place an
  `*http.Client` is made, so the first request in the program's history has
  already been judged: `TestOnlyTheGuardBuildsTheWire` and
  `TestNetHTTPStaysInItsRooms`.
- **The one host that is not Google's is the fifth grant.** `AllowUpdateFrom`
  names one GitHub repository for one run and admits GET on its releases
  listing, its download path and the two asset hosts that redirect lands on,
  with no credential on any of them, and its three callers are `gdoc update`
  and the daily check in `help` and in `gdoc mcp`, neither of which installs
  anything: `TestWithoutTheUpdateGrantGitHubIsRefused`,
  `TestTheUpdateGrantOpensNothingBesideThoseThreeReads` and
  `TestAnUpdateRequestCarriesNoBearer`.
- **Two commands ask unasked, `help` and `mcp`**, and no third: once in 24
  hours from one stamp they share, under a two-second ceiling, never from a
  checkout build, and once per process in `mcp`: `TestAStaleStampMakesHelpAskOnce`,
  `TestAFreshStampMakesNoRequest`, `TestTheCheckIsBoundedByTwoSeconds`,
  `TestACheckoutBuildNeverChecks` and `TestACheckoutBuildNeverAsksFromMcp`.
- **The reachable set has three doors**, the ids a command was handed, the id a
  create the guard itself carried came back with, and the one id `AllowAsset`
  admits one media read of; naming a folder to create in opens no fourth:
  `TestAllowCreateInDoesNotAdmitTheFolder`, `TestCreateTeachesThePolicy` and
  `TestTheAssetGrantAdmitsNoDocument`.
- **Every change inside a handed-in document is a suggestion**, refused in the
  process unless the body says `writeMode: SUGGEST`:
  `TestAHandedInDocumentIsNeverDirectlyEditedWithoutTheGrant`.
- **The one door in that wall is `Policy.GrantInPlace`**, which raises one id for
  one run to four styling request kinds, none of which can change a character.
  It never narrows: an id already at the full level, which only a create the
  guard itself carried gives, is left there and the call is noted:
  `TestNothingAtLevelInPlaceCanChangeACharacter` and
  `TestGrantInPlaceLeavesACreatedDocumentAtFull`.
- **A grant names one object and dies with the process.** `AllowCreateIn`,
  `AllowReject`, `AllowCopy`, `AllowMarker`, `AllowUpdateFrom`, `AllowAsset`,
  `AllowAccountRead`, called by `gdoc mcp`'s `login` and `auth status`, and
  `GrantInPlace` are per-run, and nothing writes one down:
  `TestAGrantedRejectSuggestionCarriesAndNothingElseInTheFamilyDoes`,
  `TestTheAccountGrantDiesWithThePolicy` and
  `TestTheAssetGrantDiesWithThePolicy`.
- **Nothing trusts a success.** Every write is read back through a route it did
  not go out on, and `verified: false` is reported rather than raised, because
  the write happened: `TestVerifyCatchesADirectEditWhoseReplacementCarriesTheQuote`.
- **The binary prints facts and the skills judge**, with one written-down
  exception. No field under `go/internal/` says whether something was handled,
  accepted or matters: `TestThreadsCarriesEveryFactAndJudgesNone` and
  `TestMatchGivesAnchoredDetachedAndUnmatched`. The exception is
  `internal/chat`'s hold list, which judges that a chat write is risky, because
  a model asked whether a comment is steering it is the thing being steered.
  Nail accepted it in decision 16 of M14, and the list is fixed and in Go:
  `TestTheFirstRuleThatTripsIsTheOneNamed` and
  `TestAuthorDomainChangesNoOutcome`.
- **One JSON object reaches stdout**, always through `internal/emit`, except that
  `mcp` writes JSON-RPC lines, a help screen on a terminal without `--json` writes
  nothing, and `comments --follow` writes one object per line with its envelope
  last; the exit code is 0 if and only if that last object says `ok`, or 0 for
  `help` and 1 for bare `gdoc`: `TestOnlyJSONObjectRefusesAnythingAfterTheObject`,
  `TestAPanicIsStillOneEnvelope`, `TestStdoutCarriesOnlyJSONRPC`, `TestHelpOnATerminalWritesNothingToStdout` and `TestOnlyFollowWritesLines`.
- **The binary never prompts, and every command but `mcp` leaves stdin alone.**
  `mcp` reads its protocol there, and only `main` names the real stdin and
  stdout, so every other room is handed a reader and a writer. It still refuses
  what it did not understand: `TestOnlyMainNamesStdinAndStdout`,
  `TestTrailingArgumentsAreRefused` and `TestUnknownCommandFailsAndNamesItself`.
- **Nothing under `go/` runs an external program, except one `open`**, from
  `desktop.go` alone on macOS, which is why the login prints a URL instead of
  opening a browser. Two argument lists, the extension handed to Claude Desktop
  and `-g <bundle>` on the menu bar app, and two callers of the seam,
  `withExtension` and `openPresenceApp`, and no third:
  `TestNothingRunsAnExternalProgram`, `TestOnlyDesktopRunsAProgram` and
  `TestTheOneProgramOpensOnlyTheExtensionAndTheApp`.
- **Nothing runs git.** No command and no skill runs it, to commit or to ask
  whether a tree is a repository or a file dirty. Only `install.sh`, the
  Makefile's version stamp, `make tag` and the workflows read git, which is
  about this repository and not somebody's documents:
  `TestNothingRunsAnExternalProgram` holds the binary half.
- **Three modules, each named in `allowedModules` with its reason in the spec**,
  and both directions fail: `TestNoThirdPartyDependencies` and
  `TestAllowedModulesAreReallyRequired`.
- **A change names text, never a stored index**, and a quote that is not there
  exactly once is refused: `TestFindSpanRefusesTextThatOccursTwice` and
  `TestAQuoteCrossingAChipIsRefused`.
- **A file gdoc half understands never reaches a document.** Every input is
  decoded strictly and refused by name: `TestReadRefusesAndNamesWhatIsWrong`.
- **Not knowing never resolves to overwrite.** An output file needs `--force`,
  and a note's block is rewritten byte for byte through `internal/atomicfile`:
  `TestBuildRefusesAnExistingOutUnlessForced` and
  `TestWriteAnUnchangedBlockIsByteIdentical`.
- **Nothing is overwritten by `export`**, which has no `--force` to add: a taken
  path takes the next free number, and the one file it writes into is the note
  it stamps. `TestNothingCanReplaceAFile` reads the package's own source, and
  `TestCreateRefusesAPathThatExists` holds the link-not-rename underneath it.
- **Everything gdoc writes into a thread opens with the mark in effect and
  carries no markdown**, because a Docs thread renders markdown literally, and
  because the mark is the only record of authorship there is. The mark is a
  setting, ⁘ by default, and may be empty; 🤖, ⁘ and ∴ stay known marks
  (DECISIONS.md, 2026-10-04): `TestMarkdownNamesWhatDocsWouldRenderLiterally`.
- **The presence file is the one file gdoc appends to, and gdoc never reads it.**
  `presence/<yyyy>-<mm>.jsonl` in the config directory is written only under the
  `presence` setting, carries the listed fields and no document word but the title,
  and nothing gdoc decides depends on a line in it. Beside those files the app
  holds `flock` on `presence/app.lock` for its life; gdoc holds no lock and reads
  no byte, asking once whether the app holds its own: `TestOnlyPresenceAppends`,
  `TestPresenceExportsNoReader`, `TestOffTouchesNothing`,
  `TestTheEventCarriesOnlyItsListedFields` and
  `TestTheOpenerAsksThreeThingsAndRunsOpenGOnce`.
- **The menu bar app runs six argv lists, signals one wait and reads three things.**
  `help --json`, `auth status`, `auth login`, `auth logout`, `update` and
  `update --desktop`, from `PresenceApp/Runner.swift` alone, at `~/.local/bin/gdoc`
  or beside the bundle and never from PATH, with four variables and no stdin; one
  SIGINT to a `comments --wait` or `--follow` the kernel agreed every claim about, from
  `Stopper.swift` alone; the record, its plist and the `gdoc:` front matter of
  the notes Spotlight names and of those under the `hubPath` override where one
  is set, and no other byte: `testTheArgvEnumIsClosedAtSix`,
  `testGdocIsResolvedAtLocalBinThenBesideTheBundleNeverFromPATH`,
  `testASignalIsSentOnlyByTheStopper`, `testAFileThatMerelyMentionsTheIdIsNotTheNote`
  and `testStopSendsOneSIGINTOnlyAfterTheProcessCheck`.
- **A hold dies with the sign-in it was raised under.** `mcpHolds` keeps a hash of
  the refresh token beside the hold and no email, and a release or a sweep that
  finds another token or none drops the hold and sends nothing, while a token nobody
  can read keeps it: `TestAReleaseAfterTheSignInChangedSendsNothingAndDropsTheHold`
  and `TestTheSweepDropsAHoldWhoseSignInChanged`.
- **Identity is never a gate.** Whose account wrote a comment decides nothing.
  The marker decides.
- **Skills are linked, not copied.** The five folders under `skills/` are
  symlinked into `~/.claude/skills/` by `./install.sh`, so an edit is live
  before it is committed; a colleague gets the same five through the plugin in
  `.claude-plugin/`, and `release/install.sh --skills` is the one route that
  copies one. The one committed copy is `skills/gdoc-review/review.md`, embedded
  for chat: `TestTheEmbeddedCoreIsTheSkillsCore`.
- **A house-style test states its value as a literal**, never reading the
  constant it checks, because a test that reads the constant follows it wherever
  somebody moves it: `TestHeadingNumberingIsTheLiteralFormat`.
- **Each package carries exactly one comment opening `Package <name>`**, or
  `Command` for a main package, in a file `go doc` reads. A file may open with a
  paragraph about itself, but a blank line separates it from the `package`
  clause, or `go doc` joins the two: `TestEveryPackageHasExactlyOnePackageComment`
  and `TestOnlyThePackageCommentReachesGoDoc`.
- **This file stays under 300 lines and its task map names files that exist**,
  because every session pays for it unasked: `TestCLAUDEmdIsUnderTheCeiling` and
  `TestTheTaskMapNamesFilesThatExist`.

## If you touch

| This | Read |
|---|---|
| the network, a URL, a request shape, a grant | `go/internal/guard/doc.go` |
| a command's arguments, its flags, its help, its envelope | `go/cmd/gdoc/doc.go` |
| the token, the login, the scopes | `go/internal/auth/doc.go` |
| the Docs read: tabs, ranges, named ranges, elements | `go/internal/docs/doc.go` |
| what the text projection prints, and its escaping | `go/internal/view/doc.go` |
| threads, the cursor, the wait | `go/internal/comments/doc.go` |
| a suggestion gdoc writes, or takes back | `go/internal/propose/doc.go`, `go/internal/withdraw/doc.go` |
| a comment gdoc leaves on quoted words | `go/internal/annotate/doc.go` |
| whether SUGGEST is honoured today | `go/internal/probe/doc.go` |
| the survey, the apply loop, the in-place styling | `go/internal/restyle/doc.go` |
| the cover, the front-matter tables, the marker | `go/internal/prelude/doc.go` |
| the `gdoc:` block in a note | `go/internal/frontmatter/doc.go` |
| a setting, the settings file, the mark in effect | `go/internal/settings/doc.go` |
| the docx the generator writes | `go/internal/render/doc.go`, `go/internal/body/doc.go` |
| a house style value, or the gate that measures it | `go/internal/house/house.yaml`, `go/internal/drift/doc.go` |
| the house logo, its Drive id, its cache or its stamp | `go/internal/asset/doc.go`, `go/cmd/gdoc/logo.go` |
| putting a document into Drive | `go/internal/publish/doc.go` |
| an end-to-end test against real Drive | `go/internal/live/doc.go` |
| a version, a tag, a release, the updater, its notice or the stamp | `go/internal/update/doc.go`, `go/internal/lastcheck/doc.go`, `.github/workflows/release.yml` |
| chat: the protocol, a tool, the code, a label, a fact, a hold, a card | `go/internal/mcp/doc.go`, `go/internal/chat/doc.go` |
| the Claude Desktop extension, either `--desktop` | `go/cmd/gdoc/desktop.go`, `release/mcpb/manifest.json` |
| a document written back into the hub | `go/internal/export/doc.go`, `go/internal/markers/markers.go` |
| the presence record: the event, the file, what is never in it | `go/internal/presence/doc.go` |
| the menu bar app: the glyph, the panel, the bundle | `presence/Sources/`, `presence/bundle.sh`, `docs/guide/presence.md` |
| what a colleague installs, and how | `README.md`, `release/install.sh`, `.claude-plugin/` |
| a guard over the wire, the modules or these documents | `go/boundary/doc.go` |
| what a skill tells a session to run, the review session included | `skills/`, `skills/gdoc-review/SKILL.md`, and `gdoc help <command>` |
| what v2 is, and what is next | `docs/v2/SPEC.md`, `docs/v2/PLAN.md` |
| why something is the way it is | `docs/v2/DECISIONS.md` |
| what Google does, or refuses to do | `docs/v2/MEASURED.md`, `docs/v2/BLOCKED-BY-API.md` |
| work that was deferred on purpose | `docs/backlog/` |
| running a milestone | `.ralphex/board/README.md` |

## Building and testing

TDD. Write the failing test first.

| Command | Does |
|---|---|
| `make test` | `cd go && go test -race ./...`. Keep the race flag |
| `make vet` | `go vet ./...` and the gofmt check |
| `make build` | `bin/gdoc`, for this machine |
| `make dist` | the three platform binaries, static, CGO off |
| `make app` | `bin/GdocPresence.app`, universal and signed ad-hoc. `make app-test` and `make app-lint` are the Swift halves of `make test` and `make vet` |
| `make tag` | `VERSION=vX.Y.0` into `plugin.json`, committed, tagged and pushed. CI does the rest |

`~/.local/bin/gdoc` is one of two things. From a checkout it is a link to this
repo's `bin/gdoc`, so `make build` refreshes what a person typing `gdoc` gets.
Where `gdoc update` has installed, it is that release's binary with
`gdoc.previous` beside it, and `make build` changes nothing a person types.
Nail's machine is the second one.

The live tests are opt-in and each needs its own variable, because a live write
is a document that did not exist a second ago. `GDOC_LIVE_TEST=1` with
`GDOC_LIVE_DOC_ID` reads and creates nothing; adding `GDOC_LIVE_WRITE=1` runs the
writers in documents it makes itself, and `go/internal/live/doc.go` names every
variable. CI runs gofmt, vet, the raced suite and `make dist` on every push.

## Running a milestone

A milestone is a plan under `docs/plans/`, run by ralphex and reviewed by
revmux, from a normal terminal and never from a Claude Code session. Read
`.ralphex/board/README.md` first: the launch lines, what `main` carries before a
run, the board, and how to resume a run that died.

## Never

- Never edit a reviewed Google Doc, except the one document a restyle was
  granted, for the one run it was granted in. Widening that door, by another
  request kind or by a second call site, is Nail's decision and not a refactor.
- Never make a test pass by loosening an assertion about what the credential can
  reach or what a guard refuses. A refused call usually means the command did
  not say which document it was for.
- Never commit anything from `~/.config/gdoc-agent/`.
- Never post markdown into a comment thread. The binary refuses it for a reason.

## Writing style

Plain, short English. No em dashes.
