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
| `go/cmd/gdoc/` | the entry point and the twelve commands. Arguments in, one JSON object out, exit |
| `go/internal/emit/` | the output envelope every command prints through |
| `go/internal/guard/` | the network policy, and the only place a client is built |
| `go/internal/auth/` | the token file, its refresh, and the login flow |
| `go/internal/auth/loopback/` | the one-shot localhost listener the browser redirect lands on |
| `go/internal/config/` | where the per-user files live, per platform |
| `go/internal/gapi/` | the authenticated session. The one room that builds a request |
| `go/internal/docs/` | the Docs read: tabs, the document tree, suggestion ids, comment ranges |
| `go/internal/view/` | the document as the text `read` prints, and as the tree `--structure` prints |
| `go/internal/comments/` | Drive's threads joined to the Docs ranges, the cursor, and the poll |
| `go/internal/suggestions/` | what is pending, and what stopped being pending since the snapshot |
| `go/internal/docx/` | the docx export reader, and the witness match against threads |
| `go/internal/probe/` | the throwaway document that asks whether SUGGEST is honoured today |
| `go/internal/reply/` | one robot reply into a thread, and the thread read back |
| `go/internal/propose/` | a change as a suggestion, its comment, and the three read-backs |
| `go/internal/withdraw/` | gdoc taking back one of its own pending proposals |
| `go/internal/publish/` | the upload with conversion, and the three read-backs on what came out |
| `go/internal/restyle/` | the survey, and the house look applied where the document stands |
| `go/internal/prelude/` | the house template as Docs requests, proposed rather than written, and the named range that marks it |
| `go/internal/docsreq/` | the shapes a Docs request is made of: a measurement, a colour, an alignment, a length, a style object with its mask |
| `go/internal/drive/` | the trash, its confirming read, and nothing else Drive does |
| `go/internal/plaintext/` | what gdoc may write into a comment thread: the robot prefix, and no markdown |
| `go/internal/frontmatter/` | the `gdoc:` block in a note's YAML front matter, and nothing else in the file |
| `go/internal/house/` | the Altery house style as a parsed file, embedded in the binary |
| `go/internal/render/` | the docx itself: the twelve parts and the logo, the cover, the tables, the header and footer, the contents field |
| `go/internal/body/` | the note's markdown walked with goldmark into the house style's own paragraphs |
| `go/internal/cover/` | the words that reach the cover and the running head |
| `go/internal/drift/` | the one list of measured values, read out of a docx and out of a Docs answer |
| `go/internal/atomicfile/` | the temp-file-and-rename write. The one room that replaces a file's contents |
| `go/internal/live/` | the opt-in end-to-end tests. Tests only, no production code |
| `go/boundary/` | the allowlist tests over the wire, the dependencies and these documents |
| `skills/` | `gdoc-review`. Symlinked into `~/.claude/skills/`, so edits are live |
| `docs/v2/` | what v2 is, what is next, why, and what Google does |
| `docs/backlog/` | deferred work, one file per item |
| `docs/plans/` | the milestone plans. `completed/` holds the ones that ran |
| `bin/` | what `make build` and `make dist` write. Not in git, so both targets create it |

Secrets live in `~/.config/gdoc-agent/`, never in this repo.

## The invariants

Each is one sentence and names the test that holds it, except the two the
binary cannot measure: identity, which is a rule about what no field means, and
the skill links, which live in `install.sh`. Breaking one is a decision somebody
writes into `docs/v2/DECISIONS.md`, not a refactor.

- **The guard owns the wire.** `guard.NewClient` is the only place an
  `*http.Client` is made, so the first request in the program's history has
  already been judged: `TestOnlyTheGuardBuildsTheWire` and
  `TestNetHTTPStaysInItsRooms`.
- **The reachable set has two doors**, the ids a command was handed and the id a
  create the guard itself carried came back with, and naming a folder to create
  in does not open a third: `TestAllowCreateInDoesNotAdmitTheFolder` and
  `TestCreateTeachesThePolicy`.
- **Every change inside a handed-in document is a suggestion**, refused in the
  process unless the body says `writeMode: SUGGEST`:
  `TestAHandedInDocumentIsNeverDirectlyEditedWithoutTheGrant`.
- **The one door in that wall is `Policy.GrantInPlace`**, which raises one id for
  one run to four styling request kinds, none of which can change a character:
  `TestNothingAtLevelInPlaceCanChangeACharacter`.
- **A grant names one object and dies with the process.** `AllowCreateIn`,
  `AllowReject`, `AllowCopy`, `AllowMarker` and `GrantInPlace` are per-run, and
  nothing writes one down:
  `TestAGrantedRejectSuggestionCarriesAndNothingElseInTheFamilyDoes` and
  `TestAGrantedMarkerCarriesAndNothingElseDoes`.
- **Nothing trusts a success.** Every write is read back through a route it did
  not go out on, and `verified: false` is reported rather than raised, because
  the write happened: `TestVerifyCatchesADirectEditWhoseReplacementCarriesTheQuote`.
- **The binary prints facts and the skills judge.** No field under
  `go/internal/` says whether something was handled, accepted or matters:
  `TestThreadsCarriesEveryFactAndJudgesNone` and
  `TestMatchGivesAnchoredDetachedAndUnmatched`.
- **One JSON object reaches stdout**, always through `internal/emit`, and the
  exit code is 0 if and only if that object says `ok`:
  `TestOnlyJSONObjectRefusesAnythingAfterTheObject`,
  `TestLoginPrintsTheURLToStderrNotStdout` and `TestAPanicIsStillOneEnvelope`.
- **The binary never prompts and never reads stdin**, and it refuses what it did
  not understand rather than ignoring it: `TestTrailingArgumentsAreRefused` and
  `TestUnknownCommandFailsAndNamesItself`.
- **Nothing under `go/` runs an external program**, which is what lets the login
  print a URL instead of opening a browser: `TestNothingRunsAnExternalProgram`.
- **Nothing runs git.** No command and no skill runs it, to commit, to ask
  whether the tree is a repository, or to ask whether a file is dirty. Only
  `install.sh` reads git, and that is about this repository rather than about
  somebody's documents: `TestNothingRunsAnExternalProgram` holds the binary half.
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
- **Everything gdoc writes into a thread opens with the robot prefix and carries
  no markdown**, because a Docs thread renders markdown literally, and because
  the marker is the only record of authorship there is:
  `TestMarkdownNamesWhatDocsWouldRenderLiterally`.
- **Identity is never a gate.** Whose account wrote a comment decides nothing.
  The marker decides.
- **Skills are linked, not copied.** `~/.claude/skills/gdoc-review` is a symlink
  into `skills/`, so an edit is live the moment it is saved and before it is
  committed. `./install.sh` prints `+ uncommitted changes` on a dirty tree,
  refuses to replace a real directory whose contents differ, and is safe to
  re-run after any move.
- **A house-style test states its value as a literal**, never reading the
  constant it checks, because a test that reads the constant follows it wherever
  somebody moves it: `TestHeadingNumberingIsTheLiteralFormat`.
- **Each package carries exactly one comment opening `Package <name>`**, or
  `Command` for a main package, in a file `go doc` reads. A file may still open
  with a paragraph saying what that file is for, but a blank line separates it
  from the `package` clause, or `go doc` joins it into the package comment:
  `TestEveryPackageHasExactlyOnePackageComment` and
  `TestOnlyThePackageCommentReachesGoDoc`.
- **This file stays under 300 lines and its task map names files that exist**,
  because every session pays for it unasked: `TestCLAUDEmdIsUnderTheCeiling` and
  `TestTheTaskMapNamesFilesThatExist`.

## If you touch

| This | Read |
|---|---|
| the network, a URL, a request shape, a grant | `go/internal/guard/doc.go` |
| a command's arguments, its flags, its envelope | `go/cmd/gdoc/doc.go` |
| the token, the login, the scopes | `go/internal/auth/doc.go` |
| the Docs read: tabs, ranges, named ranges, elements | `go/internal/docs/doc.go` |
| what the text projection prints, and its escaping | `go/internal/view/doc.go` |
| threads, the cursor, the wait | `go/internal/comments/doc.go` |
| a suggestion gdoc writes, or takes back | `go/internal/propose/doc.go`, `go/internal/withdraw/doc.go` |
| whether SUGGEST is honoured today | `go/internal/probe/doc.go` |
| the survey, the apply loop, the in-place styling | `go/internal/restyle/doc.go` |
| the cover, the front-matter tables, the marker | `go/internal/prelude/doc.go` |
| the `gdoc:` block in a note | `go/internal/frontmatter/doc.go` |
| the docx the generator writes | `go/internal/render/doc.go`, `go/internal/body/doc.go` |
| a house style value, or the gate that measures it | `go/internal/house/house.yaml`, `go/internal/drift/doc.go` |
| putting a document into Drive | `go/internal/publish/doc.go` |
| an end-to-end test against real Drive | `go/internal/live/doc.go` |
| a guard over the wire, the modules or these documents | `go/boundary/doc.go` |
| what the review session does with what the binary prints | `skills/gdoc-review/SKILL.md` |
| what v2 is | `docs/v2/SPEC.md` |
| what is next | `docs/v2/PLAN.md` |
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

`~/.local/bin/gdoc` links to this repo's `bin/gdoc`, so `make build` refreshes
what a person typing `gdoc` gets, with no reinstall.

The live tests are opt-in and each needs its own variable, because a live read
is somebody's document and a live write is a document that did not exist a
second ago. `GDOC_LIVE_TEST=1` with `GDOC_LIVE_DOC_ID` reads and creates
nothing. Adding `GDOC_LIVE_WRITE=1` runs the writers, each in documents it makes
or copies itself. `go/internal/live/doc.go` names every variable and every test.
CI runs gofmt, vet, the raced suite and `make dist` on every push.

## Running a milestone

A milestone is a plan under `docs/plans/`, executed by ralphex and reviewed by
revmux. The plan and the `.ralphex/` configuration are committed on `main`
first, because ralphex branches from `main` and the reviewer would otherwise see
its own scripts as new code. Review is revmux only.

```bash
ralphex --tasks-only docs/plans/<plan>.md
ralphex --external-only
```

The first runs the tasks, one commit each. The second runs revmux until a clean
round or `review_patience` unchanged rounds. Both run from a normal terminal,
not from inside a Claude Code session. The plan moves to
`docs/plans/completed/` when the run finishes, by hand if the run was cut short.
Nail follows a run on the HTML board, not on status lines in chat. Read
`.ralphex/board/README.md` before starting one: it holds the launch line, the
refresh script and how to resume a run that died on a session limit.

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
