# gdoc v2 Milestone 15: Panels, what a person sees in a terminal, plan

2026-10-03. The task list for the specification at
`docs/design/decision-draft-help-on-a-terminal.md`, which Nail took in the TUI
brainstorm of 2026-10-03 and which was reread against M14 the same day. The
pictures are `docs/design/panels-round-two.html`: every screen this plan builds
is drawn there at its widths, dark and light. One commit per task, for ralphex.
It is released after v2.9.0, as v2.10.0, never before M14's gates.

The spec holds the rule, the decisions and what was rejected. This plan repeats
none of them. Where the code, the spec or this plan disagree, the task stops
and says so rather than choosing. The test names below are the ones Task 1
writes into DECISIONS.md; where the spec named a test differently, this plan's
name wins and Task 1 uses it.

## Principles

Serves: 4, every word costs a reader's attention. A person who types
`gdoc help` reads a screen built for them, not a 7.5 KB JSON line under it.
Also 1, it runs on someone else's machine: no new module, everything is the
standard library.

Strains: the invariant "One JSON object reaches stdout". A help screen on a
terminal without `--json` writes nothing there. Task 1 writes the narrowing
into DECISIONS.md and CLAUDE.md before any code. `--json` and the skills
passing it are what keep every program on the object.

## Decisions

The spec's decisions hold. In short, so a task can be checked against them:

| Rule | Where |
|---|---|
| On a terminal without `--json`, a help screen replaces the object. Help screens are `help` when it answers and bare `gdoc`. Every other result keeps its object, refusals and the panic envelope included | Task 8 |
| "Terminal" is the `isatty` question, asked with `TIOCGETA` on darwin. Every other platform answers "not a terminal", so it prints today's plain text and the object | Task 4 |
| `--json` on `help` prints the object wherever stdout goes and keeps stderr plain. It survives every spelling of help | Task 2 |
| The last line of a help screen that dropped its object is `Add --json to print the JSON object a skill reads.`, written by `run()`, never on bare `gdoc` | Task 8 |
| Bare `gdoc` opens with `gdoc needs a command.` | Task 9 |
| A pipe, a file, and every non-darwin run get today's text byte for byte, and no escape byte | Tasks 3, 6, 8, 11 |
| No status bar. `help` never reads the token file | Task 9 |
| Width: drawn at `min(width, 100)`. Two columns from 80, stacked 50 to 79, plain under 50 | Task 7 |
| One palette for dark and light: text in the terminal's own colour, accents only, reverse video on 16 colours | Task 5 |
| `auth status` names the account, read live through `accountOf` under a five-second ceiling, as the grant's second caller. `auth login`'s object does not | Task 12 |
| `auth login` waits on one `\r` spinner line and ends `✓ signed in` | Task 11 |
| `update` in the Panels style, with M14's extension text on its own line on a terminal. No progress bar | Task 10 |
| Windows is out of scope | Task 4 |

## Context (from discovery)

- `go/cmd/gdoc/main.go`: `route()` sends `mcp` away before `run()`. `run()`
  calls `safeDispatch`, sets `Version`, and prints through `emit.Print`. Bare
  `gdoc` is the `len(args) == 0` branch of `dispatch`, which prints
  `helpProse` to stderr and returns `gdoc needs a command. ` plus the usage
  line. `--help` and `-h` are found by `helpAsked` and handed to `cmdHelp`
  through `helpWords`, which keeps only command words, so a `--json` beside
  them is lost today. `gdoc help ...` itself is parsed through the table.
- `safeDispatch` is also called by `mcptools.go`. Its signature does not change.
- `authLogin` (main.go, around line 207) returns `authStatus()`, so anything
  added to `authStatus` reaches `auth login`'s object too.
- The CLI `login` variable (main.go, around line 50) calls `auth.Login`, which
  prints the link at `internal/auth/login.go:245`. `auth.StartLogin` and
  `Pending.Wait` are what `mcplogin.go` already calls. `StartLogin` refuses a
  build with no client secret, which every test build is.
- `go/cmd/gdoc/progress.go`: `isTerminal` reads the character-device bit
  (`TestOnlyACharDeviceIsATerminal`, and `cmd/gdoc/doc.go` around line 482
  states it). The escape codes are constants here, and tests pin their bytes.
  The pending mark is a space. A live line longer than its width is cut with
  `…`.
- `go/cmd/gdoc/update.go`: `resultLine` joins `actionLine` and
  `extensionLine` into one line.
- `accountOf` (mcplogin.go) is a variable; its comment, and the one on
  `gapi.Account` (`internal/gapi/account.go`, around line 30), say the MCP
  login tool is the only caller. No test pins the callers.
- `emit`'s package comment states the one-object contract with no exception.
- `CLAUDE.md` is 295 lines; the ceiling is 300.
- `cmd/gdoc` has no `TestMain`.
- CI runs on ubuntu. A darwin-only file is compiled by `make dist` and by
  `GOOS=darwin go vet`, not by the raced suite.
- Five skills call `gdoc help`; none calls bare `gdoc`. They change in the tag
  sitting, not in this run (Post-Completion).

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it
  to watch it fail, then implements until it passes.
- One commit per task. Before each commit: `cd go && go test -race ./...`,
  `test -z "$(gofmt -l .)"`, `go vet ./...` and `GOOS=darwin go vet ./...`.
- **CRITICAL: every task MUST include new or updated tests**, success and
  failure paths both.
- **CRITICAL: all tests must pass before the next task starts.**
- **CRITICAL: the pipe is today's text.** Task 3 freezes what a pipe gets,
  after `--json` exists and before any rendering code changes. No later task
  edits those goldens. A diff there is a bug in the task, not a new golden.
- **CRITICAL: the JSON objects do not change**, except help's new `--json`
  flag entry (Task 2, before the freeze) and `auth status` gaining `account`
  and `account_name` (Task 12).
- **CRITICAL: only `internal/tty` writes an escape byte.** Task 6 adds the
  boundary test; every task after it keeps it green.
- **CRITICAL: no MCP behaviour changes.** `internal/mcp` and `internal/chat`
  are untouched. In `cmd/gdoc/mcp*.go` the only edit allowed is the comment on
  `accountOf` (Task 12). The MCP tests pass untouched.
- **CRITICAL: `internal/auth` gains no terminal code.** The login screen is
  drawn in `cmd/gdoc` (Task 11).
- **CRITICAL: the guard moves by one caller, not one request.** Task 12 lets
  `auth status` reach `accountOf`. No new grant, host or request kind, and
  `policy.go` is untouched.
- **CRITICAL: no test reaches the network.** Task 12 adds a `TestMain` in
  `cmd/gdoc` that replaces `accountOf` before any test runs.
- **CRITICAL: no assertion is loosened.** Where an existing test compares a
  whole object that gains a field, the field is added to the expectation as a
  literal.
- **CRITICAL: no new module.** `TestNoThirdPartyDependencies` stays green.
- **CRITICAL: no real company domain anywhere.** Examples use `example.com`.
- **CRITICAL: a test states its value as a literal**, never reading the
  constant it checks.
- **CRITICAL: new tests go in new files** where the existing file is over 800
  lines.
- **CRITICAL: no doc.go rule without its test named**, and a doc.go or test
  name made false by a task is fixed in that task.
- **CRITICAL: no `make build` inside the run.** The Validation Commands build
  into a temp directory.
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces. Plain English.

## Testing Strategy

- **Terminal facts** (`internal/tty`): `isatty` true and false, `/dev/null`
  not a terminal, `NO_COLOR`, `TERM=dumb`, depth from `COLORTERM` and `TERM`,
  width from the ioctl, `COLUMNS`, then 80. The ioctls sit behind variables, so
  a test on ubuntu stands in for them.
- **Drawing** (`internal/panel`): golden screens with no colour at 100, 80, 60
  and 44 columns, and one golden per colour depth at 80. Visible width ignores
  escape bytes. A word longer than the line is cut.
- **The rule** (`cmd/gdoc`): `run()` with the terminal check stubbed per
  writer. Help on a terminal writes nothing to stdout; `--json` writes the
  object; a pipe writes the object; bare `gdoc` writes nothing and exits 1; a
  refusal keeps its object; a panic keeps its envelope; the hint is the last
  line only when the object was dropped.
- **Pipes stay the same**: the goldens of Task 3.
- **Account**: `accountOf` replaced in tests by fakes that answer, fail and
  hang. No live test is added.
- **Boundary**: no escape literal outside `internal/tty`; no escape byte from
  any command on a buffer; one caller of `AllowAccountRead` and two of
  `accountOf`; no new module; the MCP tests untouched.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && test -z "$(gofmt -l .)" && go vet ./... && GOOS=darwin go vet ./...`
- `make dist` builds all three binaries.
- `cd go && go build -o "$TMPDIR/gdoc-m15" ./cmd/gdoc`, then:
  - `"$TMPDIR/gdoc-m15" help 2>/dev/null | head -c 1` prints `{`.
  - `"$TMPDIR/gdoc-m15" help --json 2>/dev/null | head -c 1` prints `{`.
  - `"$TMPDIR/gdoc-m15" help 2>&1 | grep -c $'\x1b'` prints `0`.
- `git diff main...HEAD --stat -- go/internal/guard/` names `doc.go` only.
- `git diff main...HEAD --stat -- go/internal/mcp go/internal/chat` is empty.
- `git diff main...HEAD -- go/cmd/gdoc/mcp*.go` touches comment lines only.
- `wc -l CLAUDE.md` under 300.

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

Three layers, each knowing less than the one above:

1. **`internal/tty`**: facts about one stream and the environment (terminal
   or not, colour depth, width, `NO_COLOR`), the palette, and every escape
   code. It knows no command and no layout. Tasks 4 to 6.
2. **`internal/panel`**: boxes with titles in the border, a two-column table
   with group rows, a stacked layout, wrapping, and the width rule. It takes
   strings and a `tty.Style` and returns lines. It knows no command. Task 7.
3. **`cmd/gdoc`**: which screen each command draws, the `group` field, the
   `--json` flag, and the rule in `run()`. Tasks 2, 3 and 8 to 12.

Then the documents, Task 13.

## Implementation Steps

### Task 1: the DECISIONS.md entry, SPEC.md and CLAUDE.md

Serves every decision. Written before the code it records.

**Files:**
- Modify: `docs/v2/DECISIONS.md`, `docs/v2/SPEC.md`, `CLAUDE.md`

- [ ] An entry dated the day this task runs, "On a terminal, a help screen is
      for the person: Panels on stderr and nothing on stdout, and `--json`
      always prints the object", from the spec's entry, in the file's own
      shape, with its register row `holds`. Its test list is this plan's
      test names.
- [ ] The 2026-10-03 "`gdoc mcp`" row: its status cell names the one clause
      superseded, "No CLI command gains it" (`AllowAccountRead`), in the
      part-supersede shape the register already uses.
- [ ] `SPEC.md`: the line "Every command writes exactly one JSON object to
      stdout and exits" gains the help-screen exception in the entry's words.
- [ ] `CLAUDE.md`, staying under 300 lines:
  - The stdout invariant is one sentence naming `mcp` and the help screen,
    replacing the two sentences it has now.
  - The grant line names `auth status` beside the MCP login as callers of
    `AllowAccountRead`, in the same line.
  - One row in "What lives where" for both new packages: `go/internal/tty/`,
    `go/internal/panel/`, "the terminal, its colours and width, and the boxes
    drawn on it".
  - If the file passes 299 lines, shorten the `go/internal/export/` row, which
    is the longest, rather than any invariant.
- [ ] `cd go && go test -race ./boundary/` passes.
- [ ] `git commit -m "docs(decisions): help on a terminal is for the person"`

### Task 2: `--json` on help, in every spelling

Before the freeze, so the frozen help object already carries the flag.

**Files:**
- Modify: `go/cmd/gdoc/commands.go`, `go/cmd/gdoc/help.go`,
  `go/cmd/gdoc/main.go`, `go/cmd/gdoc/help_test.go`, `go/cmd/gdoc/doc.go`
- Create: `go/cmd/gdoc/helpjson_test.go`

- [ ] `help` takes `--json`, optional, summary "print the JSON object even on
      a terminal, and keep stderr plain". The table lists it, so
      `gdoc help help` and completion name it.
- [ ] `helpWords` returns `(words []string, json bool)` and strips `--json`
      before `match`. `cmdHelp` takes `json bool` as a parameter, from the
      table's parsed flags on `gdoc help ...` and from `helpWords` on
      `--help` and `-h`.
- [ ] `TestHelpWithJSONPrintsTheObjectOnATerminal` is written in Task 8; here
      `TestEverySpellingOfHelpKeepsJSON` has one subtest per spelling:
      `gdoc help --json`, `gdoc help publish --json`,
      `gdoc publish --help --json`, `gdoc --help --json`, `gdoc -h --json`.
- [ ] `gdoc --json` alone stays an unknown command: `TestJSONAloneIsStillUnknown`.
- [ ] `TestHelpTakesWordsAndNoFlags` becomes
      `TestHelpTakesWordsAndOnlyTheJSONFlag`, still refusing any other flag;
      `cmd/gdoc/doc.go` (around line 232) names it.
- [ ] `git commit -m "feat(help): --json, in every spelling of help"`

### Task 3: freeze what a pipe gets today

Before any rendering code changes.

**Files:**
- Create: `go/cmd/gdoc/pipe_test.go`, `go/cmd/gdoc/testdata/pipe/*.golden`

- [ ] Goldens of stderr and stdout, through buffers (never a terminal), with
      the version and the notice fixed by the test: `gdoc help`,
      `gdoc help publish`, `gdoc help comments`, bare `gdoc`,
      `gdoc frobnicate`.
- [ ] Goldens of `update`'s plain lines: `newProgress` on a buffer, driven
      through each step state and a failure, plus `resultLine` for an install
      with and without the extension text.
- [ ] The login line is not frozen here: the real `StartLogin` refuses a test
      build. Task 11 pins it, test first, before it moves the print.
- [ ] `TestHelpOnAPipeIsTodaysTextByteForByte` and its siblings pass now. From
      this task on, these goldens are read only.
- [ ] `git commit -m "test(cmd): what a pipe gets today, frozen"`

### Task 4: internal/tty, is it a terminal

**Files:**
- Create: `go/internal/tty/doc.go`, `go/internal/tty/terminal.go`,
  `go/internal/tty/terminal_darwin.go`, `go/internal/tty/terminal_other.go`,
  `go/internal/tty/terminal_test.go`

- [ ] `terminal.go` holds `var isatty func(fd uintptr) bool` and
      `func IsTerminal(w io.Writer) bool`, which is true only for an
      `*os.File` for which `isatty(f.Fd())` is true. `IsTerminal` always
      calls the variable.
- [ ] `terminal_darwin.go` sets the variable's default to the `TIOCGETA`
      ioctl through `syscall.Syscall(syscall.SYS_IOCTL, ...)`.
      `terminal_other.go` (`//go:build !darwin`) sets it to a function that
      answers false. Neither returns from `IsTerminal` directly.
- [ ] `TestABufferIsNotATerminal`, `TestAPipeIsNotATerminal`,
      `TestDevNullIsNotATerminal` (opens `os.DevNull`, real ioctl on darwin,
      the stub elsewhere), `TestTheIoctlAnswerIsTheAnswer` (variable stubbed).
- [ ] `Colour(env func(string) string) Depth`: none when `NO_COLOR` is set
      and not empty, or `TERM=dumb`; truecolor for `COLORTERM` `truecolor` or
      `24bit`; 256 for a `TERM` ending `-256color`; else 16. A test per row.
- [ ] `Width(w io.Writer, env) int`: `TIOCGWINSZ` through a second variable
      on darwin, then a positive `COLUMNS`, then 80. A test per row.
- [ ] `doc.go` names every test above.
- [ ] `git commit -m "feat(tty): one answer to whether a stream is a terminal"`

### Task 5: internal/tty, the palette and the escape codes

**Files:**
- Create: `go/internal/tty/style.go`, `go/internal/tty/style_test.go`

- [ ] `Style` built from a `Depth`. Roles, not colours: `Title`, `Key`,
      `Border`, `Dim`, `OK`, `Fail`, `Warn`, `Chip`. Text has no role: it
      keeps the terminal's own colour. Hex, 256 index and 16-colour code per
      role are palette 4A in `panels-round-two.html`, stated as literals in
      the tests.
- [ ] The cursor and line codes live here too: up, clear below, clear to end
      of line, wrap off, wrap on.
- [ ] At depth none every role returns its text unchanged:
      `TestNoColourWritesNoEscapeByte`.
- [ ] At 16 colours a chip is reverse video: `TestAChipIsReverseVideoOnSixteen`.
- [ ] `VisibleWidth(s)` counts runes outside escape sequences, box-drawing and
      braille as one: `TestVisibleWidthSkipsEscapes`.
- [ ] `git commit -m "feat(tty): the one palette, by role and by depth"`

### Task 6: one definition of a terminal, and no escape byte outside tty

**Files:**
- Modify: `go/cmd/gdoc/progress.go`, `go/cmd/gdoc/progress_test.go`,
  `go/cmd/gdoc/doc.go`
- Create: `go/boundary/escape_test.go`

- [ ] `progress.go` uses `tty.IsTerminal`, `tty.Colour` and `tty.Style`, and
      its escape constants are gone, replaced by `tty`'s.
- [ ] `TestOnlyACharDeviceIsATerminal` becomes
      `TestOnlyATerminalDriverMakesATerminal`, stubbing `tty`'s variable; the
      paragraph in `cmd/gdoc/doc.go` (around line 482) says the same and names
      it.
- [ ] The progress tests that pinned the old colour bytes keep pinning them,
      as literals, now produced by `tty.Style` at depth 16.
- [ ] `TestNoEscapeLiteralOutsideTTY` parses every non-test Go file under
      `go/` with `go/ast` and fails on a string `BasicLit` holding `\x1b`,
      `\033` or `\u001b`, in any letter case, outside `internal/tty`. A
      comment that mentions one does not count.
- [ ] Task 3's goldens pass unchanged.
- [ ] `git commit -m "refactor(progress): the terminal check and the escapes live in tty"`

### Task 7: internal/panel, boxes and the width rule

**Files:**
- Create: `go/internal/panel/doc.go`, `go/internal/panel/panel.go`,
  `go/internal/panel/wrap.go`, `go/internal/panel/panel_test.go`,
  `go/internal/panel/testdata/*.golden`

- [ ] `Layout(width) Kind`: `TwoColumns` at 80 and over, drawn at
      `min(width, 100)`; `Stacked` from 50 to 79; `Plain` under 50. A test per
      band edge: 49, 50, 79, 80, 100, 160.
- [ ] `Box(title, right string, rows)`: a title in the top border, an
      optional right label, `├─ name ─┤` separators, a column joint `┬ ┼ ┴`
      for two columns.
- [ ] `Wrap(s, width)`: breaks at spaces; a word longer than the line starts a
      new line and is cut there, with no `…`: `TestAHashIsCutNotShortened`.
- [ ] Goldens with no colour at 100, 80, 60 and 44, and one per depth (16,
      256, truecolor) at 80. `TestNoLineIsWiderThanItsBox` walks every golden
      with `tty.VisibleWidth`.
- [ ] `git commit -m "feat(panel): titled boxes, two columns or stacked, by width"`

### Task 8: the rule in run()

**Files:**
- Modify: `go/internal/emit/emit.go`, `go/cmd/gdoc/main.go`,
  `go/cmd/gdoc/help.go`
- Create: `go/cmd/gdoc/screen_test.go`

- [ ] `emit.Result` gains `Screen bool` with the tag `json:"-"`. `emit`'s
      package comment names the exception. `TestScreenNeverReachesTheObject`
      in `emit`.
- [ ] `cmdHelp` sets `Screen` only on success and only when `json` is false.
      The bare branch of `dispatch` sets it. Nothing else does.
- [ ] `cmd/gdoc` holds one `var isTerminal = tty.IsTerminal`. A test stubs it
      per writer, so stdout and stderr answer apart.
- [ ] `run()`: when `r.Screen` and `isTerminal(out)`, it prints no object.
      Then, for `help` and not for bare `gdoc`, it writes the hint as the last
      line on `errOut`: `Add --json to print the JSON object a skill reads.`,
      dim when colour is on. The exit code stays `emit.ExitCode(r)`.
- [ ] Tests, stubbing `isTerminal`:
      `TestHelpOnATerminalWritesNothingToStdout` (exit 0),
      `TestBareGdocOnATerminalWritesNothingToStdoutAndExitsOne`,
      `TestHelpWithJSONPrintsTheObjectOnATerminal` (one subtest per spelling
      from Task 2), `TestARefusalKeepsItsObjectOnATerminal`
      (`gdoc help sing`), `TestAPanicKeepsItsEnvelopeOnATerminal`,
      `TestEveryOtherCommandKeepsItsObjectOnATerminal` (`auth status`),
      `TestTheHintIsTheLastLineOnlyWhenTheObjectWasDropped` (stdout a
      terminal: hint; stdout a pipe and stderr a terminal: no hint; bare:
      no hint).
- [ ] `TestNoEscapeByteReachesAPipe`: every command in the table run through
      `run()` with buffers, help and bare `gdoc` included, writes no `0x1b`
      on either stream.
- [ ] `TestHelpIsOneObjectAndTheProseIsOnStderr` still passes; its doc
      comment names the terminal exception.
- [ ] The MCP tests pass untouched: `go test ./cmd/gdoc -run 'Mcp|MCP'`.
- [ ] `git commit -m "feat(cmd): on a terminal a help screen replaces the object"`

### Task 9: help and bare gdoc as Panels

**Files:**
- Modify: `go/cmd/gdoc/commands.go`, `go/cmd/gdoc/help.go`,
  `go/cmd/gdoc/main.go`
- Create: `go/cmd/gdoc/helpscreen.go`, `go/cmd/gdoc/helpscreen_test.go`,
  `go/cmd/gdoc/testdata/screens/*.golden`

- [ ] The `command` struct gains `group`, one of `Read`, `Write into a doc`,
      `Make a doc`, `Account and tool`, set as the spec lists them, `mcp`
      under the last. The JSON object does not carry it:
      `TestEveryCommandHasAGroup`, `TestTheGroupIsNotInTheObject`.
- [ ] When `isTerminal(errOut)` and `json` is false, `cmdHelp` draws: the top
      box `gdoc <version>` with usage, and the release notice and its
      warnings when there are any; the grouped command table; and the
      `Run gdoc help <command> ...` line. `run()` adds the hint after it.
- [ ] `gdoc help <command>`: its box, usage, flags as two columns, and the
      example flush left under the box as one line.
- [ ] Bare `gdoc`: `gdoc needs a command.` first, then the help.
- [ ] Under `NO_COLOR` or `TERM=dumb` on a terminal, the same boxes with no
      escape byte.
- [ ] `TestHelpNeverReadsTheToken`: parses `help.go`, `helpscreen.go` and
      `notice.go` with `go/ast` and finds no reference to the `auth` package.
- [ ] Goldens with no colour at 100, 80, 60 and 44 for the three screens, and
      one at 80 in truecolor; they match `panels-round-two.html`. Task 3's
      pipe goldens still pass.
- [ ] `git commit -m "feat(help): Panels on a terminal, grouped by job"`

### Task 10: update as Panels

**Files:**
- Modify: `go/cmd/gdoc/progress.go`, `go/cmd/gdoc/update.go`
- Create: `go/cmd/gdoc/updatescreen_test.go`

- [ ] On a terminal, the live list is drawn inside a box titled `update`,
      with `vX › vY` on the right once the release is chosen. The pending mark
      becomes `○` on a terminal only. Spinner, `✓` and `✗` as today, in the
      palette's roles.
- [ ] A line longer than the box wraps through `panel.Wrap` into explicit
      lines, and the redraw counts the lines it wrote. The `…` cut is gone.
      A failure reason wraps the same way, never shortened.
- [ ] On a terminal, `actionLine` and `extensionLine` are two lines under the
      box. On a pipe `resultLine` stays one line, as Task 3 froze it.
- [ ] Goldens: running, installed, nothing newer, GitHub did not answer,
      failed at verify checksum, and installed with the extension line.
- [ ] Task 3's pipe goldens for `update` still pass.
- [ ] `git commit -m "feat(update): the step list inside a box"`

### Task 11: auth login waits on one line

**Files:**
- Modify: `go/cmd/gdoc/main.go`, `go/internal/auth/login.go`
- Create: `go/cmd/gdoc/loginscreen.go`, `go/cmd/gdoc/loginscreen_test.go`,
  a test in `go/internal/auth/` for `LinkLine`

- [ ] Test first: `auth.LinkLine(url string) string` returns exactly
      `"Open this link in your browser to sign in:\n" + url + "\n"`, pinned by
      that literal in `TestTheLinkLineIsTodays`. `auth.Login` prints through
      it, and its six tests pass unchanged.
- [ ] The CLI `login` variable calls `auth.StartLogin` and `Pending.Wait`
      itself, as `mcplogin.go` already does, so `cmd/gdoc` owns the stream.
      `auth.Login` stays for its own tests and callers.
- [ ] On a pipe, `cmd/gdoc` prints `auth.LinkLine(url)` and nothing else:
      `TestThePipeLoginLineIsTodays`.
- [ ] On a terminal: the link in a box, then one spinner line
      `waiting for the browser` redrawn with `\r` only, then `✓ signed in`
      followed by `tty`'s clear-to-end-of-line, so no spinner text is left.
      `TestTheLoginSpinnerMovesNoCursorButCarriageReturn` reads the bytes and
      finds no cursor-up and no wrap-off.
- [ ] A failed wait ends the spinner line with `✗` and the error, and the
      object says why, as today.
- [ ] The shared login lock of M14 is untouched: its tests pass unchanged.
- [ ] `git commit -m "feat(login): one waiting line, then signed in"`

### Task 12: auth status names the account

**Files:**
- Modify: `go/cmd/gdoc/main.go`, `go/cmd/gdoc/main_test.go` (only where a
  whole object is compared), `go/cmd/gdoc/doc.go`,
  `go/internal/guard/doc.go`, `go/internal/gapi/account.go` (comment only),
  `go/cmd/gdoc/mcplogin.go` (comment on `accountOf` only)
- Create: `go/cmd/gdoc/main_setup_test.go` (the `TestMain`),
  `go/cmd/gdoc/statusaccount_test.go`, `go/boundary/account_test.go`

- [ ] `TestMain` in `cmd/gdoc` replaces `accountOf` with a fake that answers
      `name@example.com`, `Example Person`, before any test runs, so no test
      reaches the network.
- [ ] `authStatus(ctx, withAccount bool)`. The `auth status` command passes
      true; `authLogin` passes false, so `auth login`'s object is unchanged.
- [ ] With `withAccount`, a token present and no missing scope, it calls
      `accountOf` under `accountCeiling`, a variable of five seconds pinned by
      a literal in `TestTheAccountCeilingIsFiveSeconds`. The answer goes into
      the data as `account` and `account_name`.
- [ ] `TestAuthStatusNamesTheAccountItReadsLive`.
- [ ] `TestNoTokenMakesNoAccountRequest` and
      `TestAMissingScopeMakesNoAccountRequest`.
- [ ] `TestAuthStatusOfflineStillAnswersWithoutTheAccount`: a fake that fails
      gives `ok: true`, no `account`, and one warning naming the reason.
- [ ] `TestAnAccountReadThatHangsStopsAtTheCeiling`: a fake that blocks on
      `ctx.Done()`, the ceiling set short by the test.
- [ ] `TestAuthLoginCarriesNoAccount`.
- [ ] On a terminal: a panel saying signed in, signed out or scopes missing,
      and the account, with the object below it as today. On a pipe, stderr
      stays empty: `TestAuthStatusOnAPipeWritesNoStderr`.
- [ ] `TestOnlyAccountOfCallsAllowAccountRead` and
      `TestAccountOfHasTwoCallers` read the syntax tree of every non-test
      file: one call of `AllowAccountRead`, inside `accountOf`; and two
      callers of `accountOf`, the MCP login tool in `mcplogin.go` and
      `authStatus` in `main.go`.
- [ ] The comments that say the MCP login is the only caller, on `accountOf`
      and on `gapi.Account`, name both. The guard's `doc.go` names both.
      `cmd/gdoc/doc.go`, "auth status is a report", gains the account and the
      ceiling, with the tests named.
- [ ] `git commit -m "feat(auth): status names the account it signs in as"`

### Task 13: the documents

**Files:**
- Modify: `go/cmd/gdoc/doc.go`, `docs/guide/` pages that show help or login
  output, `docs/v2/PLAN.md`, `README.md` if it shows help output

- [ ] `cmd/gdoc/doc.go`: the help-screen rule, `--json`, the hint and the bare
      `gdoc` screen, each with its test named.
- [ ] The guide pages show the new screens where they showed the old.
- [ ] `PLAN.md`: M15 under Done, with this plan's path.
- [ ] Move this plan to `docs/plans/completed/`, and the spec beside it as
      `2026-10-03-gdoc-v2-m15-panels-spec.md`; fix every link to both.
- [ ] `cd go && go test -race ./...` passes.
- [ ] `git commit -m "docs(v2): M15, completed"`

### Task 14: verify acceptance

- [ ] `make test`, `make vet`, `GOOS=darwin go vet ./...` and `make dist` pass.
- [ ] Every Validation Command gives the answer it states; record the counts
      here as ➕ notes.
- [ ] Every row of the Decisions table names a task above that built it.
- [ ] Nothing to commit unless a ➕ note was added.

## Post-Completion

**Before v2.10.0 is tagged, by Nail, in this order:**

1. **Look at it.** `make build`, then in Terminal and in iTerm2, dark and light
   profiles: `gdoc help`, `gdoc help publish`, `gdoc`, `gdoc auth status`,
   `gdoc auth login`, `gdoc update --check`, at a wide window, at 80 columns
   and at 60. Compare with `panels-round-two.html`.
2. **The AI still reads it.** In a Claude Code session, run each skill once.
   It must read help's object as before.
3. **The tag sitting.** In one sitting with `make tag VERSION=v2.10.0`: every
   skill passes `--json` on every `help` call and says that a call with no
   object is a failed call; their `needs` line moves to v2.10.0;
   `TestEverySkillPassesJSONToHelp` is added beside the SKILL.md call-line
   test. The skills move in the sitting, not in the run, because a skill
   passing `--json` to v2.9.0 is refused, the order the 2026-10-02 entry used
   for `--folder`.

**Then:** `gh workflow run nightly.yml --ref main`, and release notes saying
what a person sees now and that skills need v2.10.0.

**Not in this run, and named so nobody rediscovers it:** Windows; `auth login`
naming the account; a panel for an unknown command and whether it drops its
object; a download progress bar; a `GDOC_JSON` environment variable.
