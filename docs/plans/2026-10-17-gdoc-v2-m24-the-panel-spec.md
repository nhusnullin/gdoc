# gdoc v2 Milestone 24: the panel. The presence as the screens draw it. Specification

2026-10-17, morning. Nail opened the M23 presence after the night's release
and said it looks nothing like the design discussed the evening before. He is
right: M23 shipped the mechanism and a plain panel, and deferred the drawn
panel to this milestone, a scoping choice the project manager made and Nail
did not ask for. Nail's instruction this morning: run the next milestone now,
and make the app start by itself. This specification covers the whole drawn
panel in two halves that run in parallel: M24a, the Swift app against the
record as M23 writes it, and M24b, the six things the binary adds so the
panel can say what the screens say. The visual source of truth is
`docs/design/2026-10-17-presence-screens-v3.html`, the third version of the
screens Nail reviewed on 2026-10-17 (its mock account addresses are example
domains; the firm's domain stays out of the tree). The product decisions and
design principles of 2026-10-17 bind where the screens are silent. Two seats
prepared this, a macOS engineer with the visual designer's brief and the
trust guardian with the Go binary's owner; their reports are summarised in
`docs/design/2026-10-17-presence-mechanism.md` under M24. The decision entry
is in `docs/v2/DECISIONS.md` under 2026-10-17, and the task lists are
`docs/plans/2026-10-17-gdoc-v2-m24a-the-panel.md` and
`docs/plans/completed/2026-10-17-gdoc-v2-m24b-what-the-panel-needs.md`.

## Principles

Serves: 4, every word costs a reader's attention. The panel is the screens
Nail chose: one icon by priority, colour only in the status glyph, actions as
icons and words only for questions, facts and never predictions. Serves 3:
nothing in the panel approves a held write, a hold stops being sendable the
moment the sign-in it was held under changes, and every outbound act of the
app is a fixed argv at an absolute path or a URL built from a checked id.

Strains: 1 and 2 as M23 stated them, no further. One M23 sentence is
reversed: the app now launches processes, four fixed argv lists and nothing
else, and signals one kind of process. The record's allowlist gains fields
but no sentence of document text, no path and no account.

## What was found before deciding

- `publish` names no `<url>`, so its record carries no document id: a
  published document cannot be opened or linked from the panel today.
- A hold already survives a change of account: the release checks only that
  a token exists, so a `gdoc auth login` as another account, or a sign-out
  followed by a sign-in, would post a held write under the next account.
  `gapi.Session` already treats a changed refresh token as a newer sign-in,
  which is the rule a hold can borrow.
- A live watch is signalable only while a `comments --wait` is in flight:
  that process traps SIGINT and still writes its end; between waits there is
  no pid. Chat never waits, so every watch is `via: terminal`.
- The screens draw six things the record forbids or lacks: a Drive folder
  name, an error sentence with a line number, a judgement on thread state,
  an account change read from the log, the mark, and per-step counts.
- Measured 2026-10-17 at 09:25: a bundle swapped by rename under a running
  instance leaves the process alive and reading its own path's plist as the
  new version. Measured at 09:17: the bundle `gdoc update` installed launched
  with no dialog. Both in `docs/v2/MEASURED.md`.
- CLAUDE.md stands at 297 of 300 lines.

## Decisions taken

Nail, 2026-10-17 morning, in chat:

| Rule | Where |
|---|---|
| The next milestone starts now, and it is the panel as the screens draw it | this file |
| The app starts by itself: launch at login on by default, and after `gdoc update` replaces the bundle the app comes back on its own. The first open after the first install is the only manual step, ever | presence/ |
| He quit the M23 app; the released 2.14.0 bundle is what `gdoc update` left in `~/Applications` | nothing |

Taken by the project manager with the two seats, each a sentence Nail can strike:

| Rule | Where |
|---|---|
| Two halves run in parallel from `main` in two worktrees. M24a touches `presence/` and `docs/`; M24b touches `go/`, the shared fixture and the Swift decoder for the new fields. M24b merges first, then M24a resolves the decoder and the fixture | plans |
| The token sheet is the CSS of the screens file as Swift literals: surfaces, three text levels, hairlines, three status colours, the hold and stuck tints, the light palette as its own set, the two shadow stacks. No semantic NSColor stands in for a panel colour, because the native look is the complaint. A house-style test states each value as a literal | PresenceUI/Theme |
| The panel is 320 px on the 8 pt grid, capped at 600 px with the expanded section scrolling inside, header 36, section label 28, rows 32, 44 and 28, action column 108, radius 12, two type weights plus the light label's third. Hairlines are one device pixel | PresenceUI |
| One SwiftUI view per component as the blueprint lists them: header, section, row in five layouts, middle-ellipsis title, status glyph in seven kinds and six tones, ring in two sizes, hold card, stuck card with its dismiss, live row with Stop, job card with step rows, quiet line, chevron row, Yesterday section, scroll box, action icons as Path shapes ported from the SVG (not SF Symbols, so the four row icons match), tooltip layer drawn by the app after half a second, switch, inline confirm for three seconds, button styles, empty state, icon strip, URL line, signed-in banner, health rows | PresenceUI/Components |
| NOW orders holds, then stuck, then running jobs, then live watches, oldest first within a class. Stuck is an `end` with `ok: false` of reply, propose, withdraw, annotate, publish, restyle or export, gone when its key clears or when dismissed: for a command that names a document, the same command starting again on that document; for a failed publish, which names none, any later publish starting. A failed read is not stuck. `verified: false` is never stuck: a hollow check, as principle 5 says, and the 2026-10-16 vote's "a failed read-back is red" yields to the principle. A start with no end from a dead pid is stuck when it is a write started within 14 days, and otherwise a TODAY row "outcome unknown" | PresenceCore/Fold |
| A live session is the `comments --wait` runs on one document with gaps under three minutes; it shows since, replied and proposed counts derived from that document's ok ends, the time of the last write, and whether the last poll failed. Never "waiting for the author" | PresenceCore/Fold |
| DOCUMENTS groups every event by document inside 14 days of its last action, newest first, no pinning; a read alone makes no row; the default is 5 with Show more; the label counts the horizon and today. TODAY is the local day newest first, default 8, with synthetic rows for a hold raised, a hold closed, live started and live stopped; reads are left out. Yesterday is its own section, one chevron row with the counts, opening inline. Nothing older appears | PresenceCore/Fold |
| The menu bar item is one template image per state: the mark alone for live, the mark with a 10 pt badge for stuck, hold, job and check. Priority: stuck, hold, job, live, idle. Not signed in, sign-in expired and gdoc not found count as stuck; network trouble does not change the icon. Idle is the dim mark at 0.45, so the panel stays reachable; the vote's "nothing" waits for Nail. The green check lasts five seconds after a `verified: true` end that arrived live, never for a replayed line | Presence/BarImage |
| The app launches exactly six argv lists and nothing else: `help --json`, `auth status`, `auth login`, `auth logout`, `update` and `update --desktop` under one runner file, at the resolved absolute path `~/.local/bin/gdoc` first and beside the bundle second, never from PATH, with stdin the null device and an environment of HOME, PATH, LANG and NO_COLOR. No setting flag and no flag only the app uses. One child per kind at a time. A source scan enumerates the set | Presence/Runner |
| `auth status` is asked on panel open, at most once a minute, and every ten minutes while a live session exists. The three words are OK, Needs sign-in, Missing scopes. An absent account with a token present reads "signed in, account not read". The expected domain is an app preference, empty by default, a hint row that blocks nothing; "Account changed" is the app's own memory of the emails it saw, never the log | Presence |
| Sign in runs `auth login` as the app's own child, takes the one URL from that child's stderr, accepts only `https://accounts.google.com` with the sign-in path, opens it with NSWorkspace and shows it as a copyable line with a 180 s countdown and Cancel, which terminates the child. Sign out confirms inline for three seconds naming the terminal watches it will stop, stops them, runs `auth logout`, and refreshes `auth status` | Presence |
| Open in Google Docs opens `https://docs.google.com/document/d/<id>/edit` for an id matching the Drive shape; Copy link copies that plain string and shows the green check for one second; both are greyed with a tooltip when there is no id | Presence/Opener |
| Open md needs no field in the record. The app reads the hub the person chose in HEALTH (a folder, kept in preferences), builds an index from the `gdoc:` block of every note's front matter, document id to note path, rescans on file events, and opens `obsidian://` with the vault and the relative path after proving the file is a regular `.md` under the hub. Unpaired: greyed, "No note is paired with this document". Moved: "Note moved or deleted". No line parameter is promised. This widens M23's "opens only the event files and its plist" by one read of the hub's front matter | Presence/HubIndex |
| Continue with Claude activates Claude Desktop and nothing more until a prefill scheme is measured; the right-click alternative waits. The hold card's "Open chat" is the same activation | Presence/Opener |
| Stop live sends one SIGINT, never repeated and never SIGKILL, only to a pid whose `comments --wait` start has no end, is alive, runs the app's gdoc binary (or its `.previous`) by `proc_pidpath`, belongs to the user, and started within ten seconds before the event's time. Anything else greys the icon with "This watch is not one gdoc presence can stop". No confirm | Presence/Stopper |
| Update runs `update` with no flag and shows the ring on the gdoc row until the end; the button says "Update", because the app never asks GitHub itself. Install extension is `update --desktop` | Presence |
| The hold card carries "Open chat" and "Dismiss". Dismiss is the app's own: the card and the pause glyph go, the TODAY row stays, the hold itself lives in the chat until its `until`, and the sub-line says so. The panel never says "Release". Product decision 31 waits for Nail | PresenceUI/HoldCard |
| Dismissals, the hub folder, the strip seen, the launch-at-login choice, the relaunch guard and the last account are the app's state, in UserDefaults, pruned past 62 days. Nothing else persists | Presence |
| Launch at login: on the first run ever, with no choice recorded, the app registers itself with `SMAppService.mainApp` and records "on"; the HEALTH switch records the person's choice; an "off" is never overridden by a relaunch or an update; `requiresApproval` shows a row that opens the Login Items settings and is not retried. This reverses product decision 32 | Presence/LoginItem |
| Self-relaunch: when the bundle on disk differs in version from the running app, no `update` run is live and the app's own update child has exited, the app waits two seconds, checks the on-disk identifier and the static signature, opens its own bundle path as a new instance with `activates = false`, and quits only after the open succeeded. Guards: at most once per on-disk version in ten minutes, a new instance waits up to five seconds for the pid that launched it, a second instance of the same identifier quits. `.gone` never relaunches. A failed open leaves the reopen row | Presence/Relauncher |
| `gdoc auth logout` removes the token file at `config.TokenPath()` and nothing else, reads its absence back, refuses by name a symlink or a non-regular file, a file that does not parse as the token, and a waiting sign-in lock, answers `ok: true, removed: false` with no token, makes no request, is no chat tool, reports a start and an end with no document, and is a visible command with its help and completion lines | cmd, auth |
| A hold remembers the sign-in it was held under: `mcpHolds` keeps a hash of the refresh token beside the hold, in memory and never in `internal/chat`; a release or a sweep that finds another token or none drops the hold, removes its confirm tool, writes `dropped`, and answers that the sign-in changed and nothing was sent; a token that cannot be read keeps the hold and sends nothing. No email is compared. This settles product decision 48 with no channel from the app | cmd/gdoc/mcphold |
| The record gains, all at schema `v: 1` because the M23 app ignores unknown fields and skips unknown kinds: `made` on a publish end (the created document's id, absent after a rollback); a `step` event for publish with a fixed name from build, upload, read_back, link, its index and count, the first step's title once the render is done; `stop` on a failed end as a fixed code from svg_picture, multi_tab, not_found, auth, network, refused, other, with `line` where the cause has one, carried from the command to the record on `emit.Result` in a field the object never prints, as `Screen` is, and set in `cmd/gdoc` from typed failures the packages raise, so `internal/presence` imports nothing new; `via: code` when the binary runs under Claude Code, with `chat` winning over it inside `gdoc mcp`; `interrupted` on a comments end; and the `dropped` hold ending. Never: a path, a folder name, a sentence, a count from the document, the mark, the account | presence, cmd |
| Every scene of the screens file is a render fixture: the panel and the bar in both appearances at 2x, with an index page for the reviewer, written outside the repository. A reviewer who drew nothing runs the design principles' checklist over the index before any screen reaches Nail, and its findings become fix tasks first | Tests/PresenceUITests |
| CLAUDE.md's presence invariant gains the app's six argv lists and the hold's sign-in rule in place of words cut elsewhere; the file stays at or under 300 | docs |

## What is not built

"Release without writing" from the panel, and any channel from the app into
`gdoc mcp`. A note path in the record. Resuming a Claude Code session, which
needs a session id gdoc does not know. The Drive folder name, page and
suggestion counts, "waiting for the author", and the mark in the record. A
Developer ID. Windows. A second instance for a second user. M17's Keychain,
which logout learns when it lands.

## Measurement

Into `docs/v2/MEASURED.md`, each with its method, Nail confirming:

1. Self-relaunch, twenty runs (Nail, from a terminal, after the merge): a bundle with another cdhash swapped under a
   running instance, the app relaunches itself, exactly one instance remains
   each time, at the new version, executing out of the new bundle.
2. The login item under an ad-hoc update (Nail, it needs a log out): registered once, the cdhash
   changes with a swap, log out and log in, the item still starts the app;
   the count of "login item added" notices macOS shows.
3. `proc_pidpath` of a running wait after `gdoc update` renamed its binary (M24a Task 18, headless, over a renamed `sleep`).
4. A SIGINT from the app to a wait under Claude Code's Bash tool: the end
   event carries `interrupted: true` and the skill prints its totals (Nail,
   after both halves merged).
5. Whether `ImageRenderer` draws a SwiftUI `ScrollView`; if not, the render
   mode clips a plain stack.
6. Whether Claude Desktop registers a URL scheme that prefills a chat (M24a
   Task 17 reads LaunchServices, no launch); until measured, Continue with
   Claude only activates the app.

## Acceptance

Filled in when both runs finish, one row per rule above, pass or the test
that says otherwise.

### M24b, the Go half

Done 2026-10-17 in eight commits, from `gdoc auth logout` to the words. One
row per clause of the three rules this half owns, the logout command, the
hold's sign-in, and the record's new fields, each naming the test that holds
it. Every row passes. `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test`,
`make vet` and `make app-test` are green at every commit, offline, with the
live variables unset.

| Clause of the rule | Test |
|---|---|
| `logout` removes the token file at `config.TokenPath()` and nothing else | `TestLogoutTouchesNothingElseInTheConfigDir`, and `TestOnlyLogoutRemovesTheToken` in `go/boundary` over the tree |
| and reads its absence back | `TestLogoutRemovesTheTokenAndReadsItsAbsenceBack` |
| refuses by name a symlink or a non-regular file | `TestLogoutRefusesASymlinkOrANonRegularFileAndRemovesNothing` |
| refuses a file that does not parse as the token | `TestLogoutRefusesATokenItCannotParseAndLeavesIt`, with `TestLogoutRemovesATokenARefreshCouldNotUse` for the token that parses and Google has stopped taking |
| refuses a waiting sign-in lock | `TestLogoutRefusesWhileASignInIsWaiting` |
| answers `ok: true, removed: false` with no token | `TestLogoutWithNoTokenIsOkAndRemovesNothing`, `TestTheLogoutCommandAnswersWhatItRemoved` |
| makes no request | `TestLogoutMakesNoRequest` |
| is no chat tool | `TestLogoutIsNoChatTool`, and `TestNothingInTheChatSignsAnybodyOut` in `go/boundary` |
| reports a start and an end with no document | `TestLogoutWritesAStartAndAnEndWithNoDoc` |
| is a visible command with its help and completion lines | the re-recorded screens, pipe and completion goldens, `TestTheLogoutSummarySaysTheGrantStaysAtGoogle` for the sentence about the grant Google keeps, `everyCommandName` in `help_test.go` and `TestTheUsageLineNamesEveryCommand` |
| `mcpHolds` keeps a hash of the refresh token beside the hold | `TestAHoldRemembersTheSignInItWasHeldUnder`, over the mark itself `TestTheSignInMarkIsTheDomainPrefixedHashOfTheRefreshToken`, `TestTheSameRefreshTokenMarksTheSameAndAnotherOneDoesNot`, `TestATokenWithNoRefreshTokenHasNoMark` |
| in memory and never in `internal/chat` | `TestTheSignInMarkReachesNoAnswerNoToolAndNoLine`, and `TestTheSignInMarkCarriesNoWordOfTheToken` for the hash |
| a release that finds another token or none drops the hold, removes its confirm tool, writes `dropped`, and answers that the sign-in changed and nothing was sent | `TestAReleaseAfterTheSignInChangedSendsNothingAndDropsTheHold`, `TestAReleaseAfterASignOutSendsNothingAndDropsTheHold`, and `TestAReleaseUnderTheSameSignInStillSends` the other way |
| a sweep that finds the same does the same | `TestTheSweepDropsAHoldWhoseSignInChanged` |
| and every way out is still exactly one line | `TestEveryWayAHoldLeavesIsExactlyOneLine`, which gained `dropped` |
| a token that cannot be read keeps the hold and sends nothing | `TestAnUnreadableTokenAtReleaseKeepsTheHoldAndSendsNothing` |
| no email is compared | the mark is the refresh token's hash alone, and nothing else reaches it: `TestTheSignInMarkCarriesNoWordOfTheToken` |
| the strain the rule accepts: this process's own rotation reads as another sign-in and drops the hold too | `TestARotatedRefreshTokenRefusesTheRelease`, written down in `go/cmd/gdoc/doc.go` and in `docs/guide/presence.md` |
| the record's new fields are all at schema `v: 1` | `TestTheSwiftFixtureIsTheGoFixture` over the one fixture, and `testAVersionOneLineDecodesAndAnUnknownFieldIsIgnored` on the Swift side |
| `made` on a publish end, absent after a rollback | `TestAPublishEndNamesTheDocumentItCreated`, `TestARolledBackPublishNamesNoDocument`, `TestOnlyPublishCarriesMade`, Swift `testMadeDecodesAndDefaultsToNil` |
| a `step` event for publish with a fixed name from build, upload, read_back, link, its index and count | `TestAPublishStepsAreTheFixedNamesInOrder`, `TestAStepIsItsNameAndItsPlaceInTheFixedList`, `TestAStepNameThatIsNotOneOfTheFourWritesNothing`, `TestAStepOutsideARunWritesNothing`, `TestStepHasOnlyPublishAsCaller` in `go/boundary`, `TestAPanicAfterAStepStillWritesTheEnd`, Swift `testAStepLineDecodesWithItsNameIndexAndCount` |
| the first step's title once the render is done | `TestTheFirstStepsEndCarriesTheTitle` |
| `stop` on a failed end as a fixed code from the seven, with `line` where the cause has one | `TestAStopIsOneOfTheSevenCodesAndNeverASentence` in `cmd/gdoc` over the reader and in `internal/presence` over the record, `TestTheSevenCodesAreLiterals`, `TestTheSVGRefusalCarriesItsLine`, `TestANotFoundIsNotFound`, `TestAnAuthRefusalIsAuth`, `TestAHandshakeTimeoutIsNetwork`, `TestAGuardRefusalIsRefused`, `TestAnUnknownFailureIsOther`, `TestAStopReachesNoSuccessfulEnd`, Swift `testStopAndLineDecodeAndDefaultToNil` |
| carried on `emit.Result` in a field the object never prints, as `Screen` is | `TestStopNeverReachesTheObject`, with `TestTheFlagChangesNoByteOfTheObject` still holding the envelope |
| set in `cmd/gdoc` from typed failures the packages raise | `TestAPictureThatCannotBeEmbeddedCarriesItsLine`, `TestAStatusErrorExposesItsStatus`, `TestATokenRefusedTwiceIsAStatusError`, `TestMoreThanOneTabIsATypedRefusal` in `internal/propose` and again in `internal/annotate`, `TestTheGuardsRefusalIsTyped`; every word a person reads stayed the word it was, which `Unwrap` and the unchanged sentence tests hold |
| so `internal/presence` imports nothing new | `TestPresenceOpensNoWireAndRunsNothing` and `presenceClosedTo` unchanged |
| `via: code` when the binary runs under Claude Code, with `chat` winning inside `gdoc mcp` | `TestViaIsCodeUnderClaudeCode`, `TestViaChatWinsOverCodeInTheMcpSession`, `TestChatWinsOverCode` over the context itself, Swift `testViaCodeAndInterruptedDecode`; the variable gdoc reads is measured in `docs/v2/MEASURED.md` under "What Claude Code puts in a child's environment" |
| `interrupted` on a comments end | `TestAnInterruptedWaitSaysSo`, and `TestAStoppedWaitsEndSaysSo` over a real `comments --wait` somebody stopped |
| the `dropped` hold ending | the hold rows above, and Swift `testADroppedLineDecodesAsAHoldEnding` |
| never a path, a folder name, a sentence, a count from the document, the mark, the account | `TestTheEventCarriesOnlyItsListedFields`, `TestAnEventCarriesNoWordOfTheEnvelopeButTheTitle` extended with an error sentence that must not reach the line, `TestNoPercentageOrEstimateIsEverWritten` |
| two halves run in parallel, M24b merges first | the branch is a worktree off `main` after M23, touching `go/`, the two fixtures and the Swift decoder only; the merge itself is Nail's |
| CLAUDE.md's presence invariant | this half changed one word, "sixteen" to "seventeen" on line 24 for the new command; M24a owns the invariant's words and the 300-line ceiling after the merge |

The one measurement this half could take alone is in `docs/v2/MEASURED.md`,
"What Claude Code puts in a child's environment", with its method. The six
numbered measurements above are M24a's or Nail's; measurement 4, a SIGINT from
the app to a wait, needs both halves merged and is Nail's.
