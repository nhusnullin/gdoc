# gdoc v2 Milestone 32a: four small fixes. Plan

Tasks: 8
Review: ui

2026-10-28. Four fixes Nail chose in one word, "all 4": the panel drops the
published version number, a cancelled update leaves the terminal wrapping, a
backslash escape in a note publishes as the character, and Tab completion
follows the installed gdoc. Run by a ralphex worker on Claude, one task, one
commit, on the local branch `gdoc-v2-m32a-four-small-fixes`. `make release`
runs without `GO`, so the push and the PR wait for Nail's word.

## Principles

Serves: 1, because each fix removes a small thing that slows Nail or a
colleague down: a line he said he does not need, a terminal left broken, a
backslash in a published document, a Tab that does not know a command. 4,
because the panel still tells the truth: the version is on the glyph's
tooltip, and a press that installed nothing still says why under the header.

Strains: one written-down thing. A note that held a backslash on purpose
before ASCII punctuation now needs two. That is accepted in the decision
entry, because CommonMark and pandoc already read it this way and gdoc's own
guides already tell people to write `\{+`.

## Overview

The decision entry of 2026-10-28 in `docs/v2/DECISIONS.md` ("Four small
fixes: ...") is the scope. The lead wrote it and the matching
`docs/v2/SPEC.md` lines on main before this run. This plan touches neither
file.

What lands, in the order the tasks take it:

1. **The panel drops the published number.** No "vX is published" line under
   the header. The tooltip keeps the number. A press that installed nothing
   still writes its line.
2. **Wrap comes back on at the end of every draw** of `gdoc update`'s live
   list, so Ctrl-C between two draws leaves the terminal wrapping.
3. **A backslash escape publishes as the character.** One pass in
   `internal/body` over a text run.
4. **`gdoc completion zsh --loader --out <path>`** writes a loader of a few
   lines that asks the gdoc on PATH for its completion at each new shell.
5. **`gdoc update` replaces gdoc's own completion file beside the token** with
   the loader, and leaves any other file there alone.
6. **The colleague's installer writes the loader.**
7. Acceptance.
8. Documents.

## Context (from discovery)

- The published line is built in
  `presence/Sources/PresenceUI/PanelHeaderModel.swift`, in the `published`
  value: `facts.updateRun.map { Sentences.updateMiss(...) } ?? latest.map {
  "\($0) is published" }`. Its tests are in
  `presence/Tests/PresenceUITests/StateTests.swift` (three asserts that expect
  `["v2.15.0 is published"]`), and `presence/Tests/PresenceUITests/RowTests.swift`
  carries a comment about it. The tooltip is `Tips.updateTo(latest)`.
  `docs/guide/presence.md` describes the line near "the quiet line under the
  header" and in the paragraph "The published version is a line under the
  header".
- The wrap codes are written in `draw` in `go/cmd/gdoc/progress.go`: `WrapOff`
  on the first draw, `WrapOn` only in the last. The test is
  `TestALiveListTurnsWrapOffAndBackOn` in `go/cmd/gdoc/progress_test.go`, and
  the rule is stated in the comment in `go/cmd/gdoc/update.go` that names that
  test. `TestOnlyTheWaitTrapsTheSignal` in `go/cmd/gdoc/main_test.go` must keep
  passing untouched.
- The text of a run is `decodeEntities` in `go/internal/body/inline.go`, which
  calls goldmark's `util.ResolveEntityNames(util.ResolveNumericReferences(...))`
  and nothing for backslashes. The backlog item
  `docs/backlog/build-never-unescapes-a-backslash.md` holds the measured
  table. Code spans and fenced code are other nodes and must keep their
  backslashes.
- `gdoc completion` is `go/cmd/gdoc/completion.go`. It writes through
  `atomicfile.Replace` after `freeToWrite(out, force)`. The colleague's
  installer, `release/install.sh`, writes `$CONFIG_DIR/completion.zsh` with
  `completion zsh --out "$COMPLETION" --force` and prints the `source` line.
  The checkout's `install.sh` writes `bin/gdoc.zsh` and is out of scope.
- `gdoc update` is `go/cmd/gdoc/update.go`; the config dir comes from
  `internal/config`.
- Backlog files this run removes with `git rm` in the task that lands each
  fix: `docs/backlog/ctrl-c-during-update-leaves-auto-wrap-off.md`,
  `docs/backlog/build-never-unescapes-a-backslash.md`,
  `docs/backlog/update-leaves-the-completion-stale.md`.

## Development Approach

- TDD. Each task writes its failing test first and runs it red.
- One task, one commit, `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test` green before the commit, with
  `GDOC_LIVE_TEST` and `GDOC_LIVE_WRITE` unset. A Swift task also runs
  `make app-test` and `make app-lint`.
- A test that states a visible value states it as a literal.
- Every `doc.go` rule this run changes names its test.
- Plain, short English in comments and documents. No em dashes.

## Testing Strategy

Unit tests in the package each fix lives in. No live test: none of the four
reaches Google. The loader is tested by reading the bytes it writes, never by
running it: no file under `go/` starts zsh or any program, by `os/exec`,
`os.StartProcess` or `syscall`, which keeps true the `completion.go` comment
"No test sources the script in a real shell". Sourcing the loader is checked
by a shell test outside `go/`, `release/test-completion.sh`, written the way
`release/test-desktop.sh` is: a fake `gdoc`, its own HOME, and a skip when zsh
is missing. CI runs it in one step beside the desktop extension's.

## Solution Overview

The panel keeps the line under the header for the answer to a press, which is
the case the person is owed words for, and gives the number to the tooltip.

Wrap goes back on at the end of every draw. Each draw turns it off before it
moves up and writes, and on after its last line. Between draws, which is where
a Ctrl-C lands, the terminal wraps.

The backslash is handled where entities already are, in one pass, so the two
never run in an order that changes `\&amp;`.

Completion: the gdoc doing an update is the old binary and cannot know the
new commands, so the refresh comes from the binary on PATH at the first moment
it can, a new shell. The loader is version-free text owned by the binary, so
the installer and `update` write the same bytes.

## Technical Details

- **The loader.** About five lines of zsh. It finds its own directory
  (`${${(%):-%x}:A:h}`), runs `gdoc completion zsh --out
  "<dir>/completion-script.zsh" --force >/dev/null 2>&1`, and sources that file
  when it is readable. If `gdoc` is not on PATH it writes nothing and prints
  nothing, but still sources an existing `completion-script.zsh`, so a
  `.zshrc` that adds `~/.local/bin` to PATH after the `source` line keeps
  the completion it had. Its first line is a fixed comment, `# gdoc completion loader`, which
  is one of the two ways `update` knows the file is gdoc's own. The other is
  the static script's first three lines, byte for byte: `#compdef gdoc`, `#`,
  `# zsh completion for gdoc, written by gdoc itself:` (see
  `go/cmd/gdoc/completion_zsh.tmpl`). The code reads those three lines off
  `zshTemplate` rather than typing them again; the tests state them as
  literals. `#compdef gdoc` alone is zsh's standard
  first line and does not count.
- **`completion zsh --loader`.** Refused for bash by name, refused without
  `--out`, and under `freeToWrite` like every write. The object it prints names
  the path and the same `add_to_zshrc` line as today.
- **`update`'s step.** After a run that installed a binary, and only then:
  the `installDecided` path, action `updated`. Neither `--rollback` nor
  `refreshHere` writes it. The path comes from a new
  `config.CompletionPath()` beside `LastCheckPath()`.
  `os.Lstat` the path `<config dir>/completion.zsh` first. A symlink, a FIFO, a
  directory or anything that is not a regular file is left alone with a
  warning naming it. A regular file is read for its header bytes only, never
  whole. If it opens with the static header or the loader's line, it is
  replaced with the loader through `atomicfile`; if it is already the loader
  byte for byte, nothing is written. Anything else is left as it is with a
  warning. A missing file is nothing to do. The result's data gains
  `completion`, the path written, only when this run changed the file, so the
  field means one thing: this run changed what new terminals do. If the
  existing result line under the box can say in one clause that the change
  reaches terminals opened from now on, it says so; if it cannot, nothing is
  added. No new step line on the
  terminal box unless the existing recorded screens can carry it; if they
  cannot, the worker regenerates `TestTheUpdateScreensAreTheirRecordedBytes`'s
  records and the commit says the diff is that one field or line.
- **The backslash pass.** In `decodeEntities`: walk the bytes once. A
  backslash followed by an ASCII punctuation byte writes that byte and skips
  both. An `&` hands the rest to goldmark's entity resolution for that
  reference only. Anything else is copied. `\&amp;` gives `&amp;`.

## What Goes Where

| Fix | Code | Tests | Words |
|---|---|---|---|
| Panel line | `PanelHeaderModel.swift` | `StateTests.swift`, `RowTests.swift` comment | `docs/guide/presence.md` |
| Wrap | `progress.go` | `progress_test.go` | the `update.go` comment |
| Backslash | `internal/body/inline.go` | `inline_test.go` | `internal/body/doc.go`, `docs/guide/exporting.md` if it hedges |
| Loader | `completion.go`, `commands.go` | `completion_test.go` | `cmd/gdoc/doc.go` |
| Update step | `update.go` | `update_test.go` | `cmd/gdoc/doc.go` |
| Loader check | `release/test-completion.sh` | itself, and a CI step in `.github/workflows/go.yml` | |
| Installer | `release/install.sh` | `release/test-desktop.sh` | `README.md` if it names the file |

## Implementation Steps

### Task 1: the panel drops the published number

**Files:** `presence/Sources/PresenceUI/PanelHeaderModel.swift`,
`presence/Tests/PresenceUITests/StateTests.swift`,
`presence/Tests/PresenceUITests/RowTests.swift`, `docs/guide/presence.md`

- [x] test first: `testAPublishedReleaseWritesNoLineUnderTheHeader`, a newer
      stable release and no press gives `header.quiet == []`
- [x] test: `testTheTooltipStillNamesThePublishedVersion`, the update action's
      tip is the literal `Tips.updateTo` text for that version (no separate
      test: the literal `"Update to v2.15.0"` is asserted inside
      `testAPublishedReleaseWritesNoLineUnderTheHeader` and at StateTests
      :141 and :175, so a third copy would say the same thing again)
- [x] test: a press that installed nothing still writes its fixed sentence
      under the header (the existing
      `testEachRunThatInstalledNothingSaysWhyInAFixedSentence` keeps passing)
- [x] run them red
- [x] drop `?? latest.map { "\($0) is published" }`; rewrite the comment above
      `published` to say why the number lives on the tooltip, naming the new
      tests
- [x] the three `StateTests` asserts expect `[]`; rename
      `testARunThatDidSomethingLeavesThePublishedLineWhereItWas`
      (StateTests.swift:265) to say what it now checks
      (`testARunThatDidSomethingLeavesNoLineUnderTheHeader`)
- [x] `RowTests.swift`: the fixture near :485 drops `quiet: ["v2.17.0 is
      published"]`, the message near :520 drops "why the number is under the
      line", and the comments near :616 and :690 stop describing the line
- [x] `PanelHeaderModel.swift`: rewrite the long comment above `look` as well
      as the one above `published`, and rename `published`, which no longer
      holds a published version (it is `answer` now)
- [x] `testTheTooltipStillNamesThePublishedVersion` states the literal
      `"Update to v2.15.0"`; dropped, because the asserts at StateTests :141
      and :175 already pin exactly that
- [x] `docs/guide/presence.md`: the available glyph has its dot and the number
      in its tooltip; the paragraph about the line under the header now says
      it is for the answer to a press
- [x] `make app-test`, `make app-lint` and `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test` green

### Task 2: wrap comes back on at the end of every draw

**Files:** `go/cmd/gdoc/progress.go`, `go/cmd/gdoc/progress_test.go`,
`go/cmd/gdoc/update.go`

- [x] test first: `TestWrapIsOnBetweenEveryTwoDraws`, a live list with three
      draws: after every write, the last wrap code in the output so far is
      `WrapOn`
- [x] `TestALiveListTurnsWrapOffAndBackOn` becomes
      `TestEachDrawTurnsWrapOffBeforeItMovesAndOnAfterItsLines`: every
      `WrapOff` comes before that draw's `Up` and `ClearBelow`, every `WrapOn`
      after that draw's last line; a plain list writes neither
- [x] in `TestWrapIsOnBetweenEveryTwoDraws`, stop the spinner the way
      `TestTheStepMarksAreTheirRolesOnATerminal` does, so the draws are
      exactly the ones the test makes
- [x] `lastScreen()` in `go/cmd/gdoc/updatescreen_test.go` drops one trailing
      `tty.WrapOn`, with a comment saying why, so the six records of
      `TestTheUpdateScreensAreTheirRecordedBytes` stay byte-identical and
      `TestALongReasonWrapsInsideTheBoxAndIsNeverShortened` keeps its eighty
      columns. Both are named in the commit
- [x] run them red
- [x] `draw` writes `WrapOff` at its start and `WrapOn` at its end, every time;
      drop the `unwrapped` field if nothing else reads it
- [x] `go/internal/tty/style.go`'s comment on `WrapOff`/`WrapOn` ("for the
      life of a moving list") says off while a draw writes, on between draws
- [x] the `update.go` comment says wrap is off only while a draw writes, so a
      Ctrl-C between two draws leaves it on, and names both tests
- [x] `TestOnlyTheWaitTrapsTheSignal` passes untouched
- [x] `git rm docs/backlog/ctrl-c-during-update-leaves-auto-wrap-off.md`
- [x] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test` green

### Task 3: a backslash escape publishes as the character

**Files:** `go/internal/body/inline.go`, `go/internal/body/inline_test.go`,
`go/internal/body/doc.go`, `go/internal/view/text.go`, `go/internal/view/doc.go`

- [x] test first, `TestABackslashEscapeIsTheCharacter`, table of literals:
      `A \[bracket\] here.` gives `A [bracket] here.`; `A \*star\* here.`
      gives `A *star* here.`; `A \{+ marker here.` gives `A {+ marker here.`;
      `\\` gives `\`; `\&amp;` gives `&amp;`; `&amp;` gives `&`; `&#8212;`
      gives `"\u2014"`; `\a` (not punctuation) gives `\a`; `a\\` at a line
      end gives `a\`; a table cell `\|` gives `|`
- [x] test: `TestCodeKeepsItsBackslashes`, a code span `` `\[x\]` `` keeps both
      backslashes; a fenced block reaches no run at all, by Nail's decision
      that code stays out of the house style, so
      `TestACodeBlockWarnsAndIsLeftOut` is that half and the test names it
- [x] test: `TestAnExportedLinkTextRoundTrips`, link text `a\]b` from
      `internal/view`'s escaping builds as `a]b`, and
      `TestAnEscapedPipeInACellIsTheCharacter` for the escaped pipe in a cell
- [x] run them red
- [x] the one-pass writer in `decodeEntities`, as in Technical Details; its
      comment says why one pass and not two
- [x] `internal/body/doc.go` states the rule and names the tests
- [x] the comments at `go/internal/view/text.go` near :1366 and
      `go/internal/view/doc.go` near :244 stop saying body never unescapes a
      punctuation mark and stop naming the backlog file; the footnote half
      stays, measured: body's line scan still refuses an escaped `\[^1]`. The
      same false claim in `cannotCarry`, `text.go` near :1777 and `doc.go`
      near :556, is corrected too: measured, an escaped `\==` now publishes as
      `==`, so writing the syntax escaped is a decision for Nail rather than
      something the build refuses
- [x] `git rm docs/backlog/build-never-unescapes-a-backslash.md`
- [x] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test` green

### Task 4: `gdoc completion zsh --loader`

**Files:** `go/cmd/gdoc/completion.go`, `go/cmd/gdoc/commands.go`,
`go/cmd/gdoc/completion_test.go`, `go/cmd/gdoc/doc.go`, help goldens

- [x] test first: `TestTheLoaderOpensWithItsOwnFirstLine` (the literal
      `# gdoc completion loader`), `TestTheLoaderNamesNoVersionAndNoOtherCommand`
      (the bytes hold no version, and no command name from the table except
      the one call `gdoc completion zsh`, and no flag but `--out` and
      `--force`, so they never go stale),
      `TestTheLoaderIsRefusedForBash`, `TestTheLoaderNeedsOut`,
      `TestTheLoaderRefusesAnExistingOutUnlessForced`
- [x] test: `TestTheLoaderIsQuietWithoutGdoc`, by reading the bytes: the call
      is guarded by a check that `gdoc` is on PATH and sends both streams to
      `/dev/null`, and `source` is guarded by a readable-file check that does
      not depend on that guard. No test runs the loader: no file under `go/` starts zsh or
      any program, by `os/exec`, `os.StartProcess` or `syscall`
- [x] test: `TestCompletionNeverReachesTheCheck`, copied from
      `TestReadNeverReachesTheCheck`: a `completion zsh --loader` run with a
      month-old stamp and a reach that fails the test makes no request and
      leaves the stamp unchanged. `completion` runs at every new shell now, so
      this is what keeps it from being a third command that asks unasked
- [x] tighten `TestNothingRunsAnExternalProgram` in
      `go/boundary/boundary_test.go` to also refuse any call to
      `os.StartProcess`, `syscall.Exec`, `syscall.ForkExec` and
      `syscall.StartProcess`. Nothing in the tree calls them today
- [x] run them red
- [x] `--loader` in the `completion` table entry; the loader text as one
      constant in `completion.go`
- [x] regenerate `go/cmd/gdoc/testdata/pipe/help.stdout.golden`: `frozen()`
      never replaces a file, so delete it, run the test once (it writes and
      fails), read it, run again. The commit says the diff is the `--loader`
      line
- [x] `release/test-completion.sh`, outside `go/`: a fake `gdoc` that writes a
      script defining `_gdoc`, its own HOME, `autoload -U compinit && compinit
      -u`, source the loader, `whence -w _gdoc`; then with no `gdoc` on PATH,
      sourcing prints nothing and `_gdoc` is still defined. Skips when zsh is
      missing. One step in `.github/workflows/go.yml` beside the desktop
      extension's runs it
- [x] the rules live in `completion.go`'s file comment: say what the loader is
      for; the paragraph "No test sources the script in a real shell" now says
      no Go test does, and names `release/test-completion.sh`. Name the tests, `TestCompletionNeverReachesTheCheck` among them, and
      `TestOnlyTheReadBackWritersCarryVerified`, whose reports table keeps
      completion out of the presence record. `cmd/gdoc/doc.go`'s completion
      bullet becomes `completion <shell> --out [--force] [--loader]`
- [x] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test` green

### Task 5: `gdoc update` puts the loader in place of gdoc's own file

**Files:** `go/cmd/gdoc/update.go`, `go/cmd/gdoc/update_test.go`,
`go/cmd/gdoc/doc.go`

- [x] test first: `TestAnUpdateReplacesGdocsStaticCompletionWithTheLoader`:
      a config dir whose `completion.zsh` opens with the static script's first
      line; after an update that installed, it holds the loader bytes and the
      result's data names the path
- [x] test: `TestAnUpdateLeavesAFileItDidNotWriteAlone`: a `completion.zsh`
      that opens with any other line is byte-identical after the run and a
      warning names it; so is one that opens `#compdef gdoc` followed by any
      line but the static header's second and third
- [x] test: `TestAnUpdateLeavesASymlinkedCompletionAlone`: a symlink at that
      path, to a gdoc-headed file, is left as it is, target unchanged, with a
      warning
- [x] test: `TestAnUpdateWritesNothingOverTheLoaderItself`: a file that is
      already the loader byte for byte is not written and the result has no
      `completion` field
- [x] test: `TestAnUpdateWithNoCompletionFileWritesNone`
- [x] test: `TestAnUpdateThatInstalledNothingTouchesNoCompletion`
- [x] run them red
- [x] every new test uses `installedAt`, which sets `GDOC_CONFIG_DIR`, so no
      test can touch the real `~/.config/gdoc-agent/completion.zsh`
- [x] `config.CompletionPath()` beside `LastCheckPath()`, with its test
- [x] the step, `os.Lstat` first, header bytes only, through `atomicfile`,
      only after a run that installed
- [x] if the recorded update screens change, regenerate them; the commit says
      the diff (they do not change: the clause under the box is said only on a
      run whose object carries `completion`, and `installedData()` in
      `updatescreen_test.go` carries none, so the six records stay
      byte-identical)
- [x] the rule and its tests go in `update.go`'s file comment
- [x] the line in `docs/backlog/release-pipeline-needs-a-rethink.md` near :67
      that names the removed file says the completion now follows the
      installed gdoc
- [x] `git rm docs/backlog/update-leaves-the-completion-stale.md`
- [x] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test` green

### Task 6: the colleague's installer writes the loader

**Files:** `release/install.sh`, and `README.md` only if it describes the
completion file

- [x] the completion step runs `"$installed" completion zsh --loader --out
      "$COMPLETION" --force`, and when that fails, today's call without
      `--loader`: the one-liner fetches this script from main but installs the
      newest stable binary, which may not know `--loader` yet. Its comment
      says both
- [x] `release/test-desktop.sh` still passes, run as `sh
      release/test-desktop.sh`; add an assertion that the completion call
      carries `--loader`
- [x] the `source` line it prints is unchanged
- [x] `env -u GDOC_LIVE_TEST -u GDOC_LIVE_WRITE make test` green

### Task 7: Verify acceptance criteria

- [x] `make test`, `make vet`, `make app-test`, `make app-lint` green, live
      variables unset (363 Swift tests, 0 failures; gofmt and vet clean)
- [x] `sh release/test-completion.sh` and `sh release/test-desktop.sh` pass
      (seven checks and sixteen, all ok)
- [x] a built binary: `completion zsh --loader --out <tmp>/c.zsh` writes the
      loader; under `zsh -f`, after `autoload -U compinit && compinit -u`,
      sourcing it with that binary first on PATH makes `whence -w _gdoc` say
      function (it says `_gdoc: function`)
- [x] `release/install.sh` names `--loader` in its completion call (line 423,
      with the fallback without it on 425)
- [x] the three backlog files are gone, and the only other backlog file
      touched is the one line in `release-pipeline-needs-a-rethink.md` (item 8,
      which now says the completion follows the installed gdoc)
- [x] `TestOnlyTheWaitTrapsTheSignal` passes untouched, and the boundary suite
      passes, tightened only by the ban on `os.StartProcess` and syscall's
      exec calls (`main_test.go` has no diff against main)

### Task 8: [Final] Update documentation

- [x] `docs/guide/from-a-checkout.md` is that page: its completion section
      names `--loader`, says it is zsh only, and says why the refresh comes
      from the installed binary
- [x] `docs/guide/exporting.md` and `skills/gdoc-export/markers.md` lose any
      hedge that the backslash reaches the document (markers.md said `publish`
      would put the backslashes into a document; both pages now say an escape
      publishes as the character, and exporting.md names the code span as the
      other side)
- [x] do not edit `CLAUDE.md`, `docs/v2/SPEC.md` or `docs/v2/DECISIONS.md`
- [x] move this plan to `docs/plans/completed/`

## Post-Completion

- The lead renders the panel once and the design reviewer reads the renders
  before Nail sees the panel.
- The lead runs the suites by hand, then `make release
  BRANCH=gdoc-v2-m32a-four-small-fixes` without `GO`.
- Nail's gdoc came from `update`, and the gdoc that runs an update is the old
  one, so his first update to this release leaves his static completion file
  alone. After the release the lead runs, on Nail's machine, `gdoc completion
  zsh --loader --out ~/.config/gdoc-agent/completion.zsh --force` with the new
  binary, once, and says so. From then on every update reaches Tab.
- Nail's check after the release, in what he sees: the panel with an update
  waiting shows the blue glyph and no line under the header; Ctrl-C during
  `gdoc update` leaves long lines wrapping; a note with `\[` publishes `[`; a
  new terminal tab after `gdoc update` offers a new command on Tab.
- Release as a minor, on Nail's word.
