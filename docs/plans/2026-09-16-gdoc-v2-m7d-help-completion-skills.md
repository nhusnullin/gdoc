# gdoc v2 Milestone 7d: help, completion, and the two skills

2026-09-16.

## Principles

Serves: 4, every word costs a reader's attention. A person at the terminal and
a session in Claude Code both learn the tool from the tool, in the words the
tool uses, and neither has to open Go source to find a flag. Serves 1: the help,
the completion and the skills add nothing that has to already be on the
machine. One text template in the standard library, no fourth module, no
external program.

Strains: none. One rule in `cmd/gdoc/doc.go` retires, "--help is a failure",
and that is a decision written in DECISIONS.md the same day, not a principle
bending.

## Overview

gdoc has three readers, and today all three learn it the hard way.

| Reader | Today | After |
|---|---|---|
| Nail at the terminal | `gdoc --help` is `ok: false`. No help, no completion | `gdoc help`, `gdoc help publish`, `--help` anywhere, and a zsh completion script the binary writes |
| a Claude Code session | one skill, `gdoc-review`. `build`, `publish` and `restyle` are learned from `doc.go` | `gdoc-publish` and `gdoc-restyle`, each running `gdoc help <command>` before its first call |
| a colleague's machine | the same binary, the same silence | the same binary, and the help travels inside it |

The design that makes this hold is **one command table**. Every command is
described once, in `go/cmd/gdoc/commands.go`: its name, the words it takes,
its flags, one sentence, one example, and the function that runs it. The
dispatcher walks the table. The usage line in every refusal is printed from
it. `help` prints it as JSON and as prose. `completion` prints it as a shell
script. A thirteenth command cannot exist without help and completion for it,
because the only way to be dispatched is to be in the table.

The skills hold no flag list. Each runs `gdoc help <command>` before its first
call in a session and reads the words and flags from the binary it is about to
run. That is the same rule as the symlink: the skill cannot drift from the
binary because it never held a copy.

## Decisions Nail took, 2026-09-16

Taken in the brainstorm that produced this plan. Each is a decision and not a
refactor. A task that finds one wrong stops and says so rather than adjusting
it.

1. **Help is an answer, and it exits 0.** `gdoc help` prints one JSON object
   on stdout and the human text on stderr, where the login URL already goes.
   It answers a question the way `auth status` with no token does. The output
   contract is unchanged: one object, prose on stderr, exit 0 if and only if
   the object says `ok`.
2. **Bare `gdoc` still fails.** It prints the human help to stderr, keeps
   today's `ok: false` object naming what is missing, and exits 1. Running the
   tool with no command did no work, and a caller that ran it by mistake must
   learn that from the exit code.
3. **`--help` and `-h` are aliases**, anywhere on the line. `gdoc read --help`
   and `gdoc --help read` both print the help for `read`. They are recognised
   before the strict parser runs, so the parser's refusals are untouched.
4. **The command table is the one description.** Nothing else lists a command
   or a flag: not the usage string, not a help text written by hand, not a
   skill. Everything that names one reads the table.
5. **Completion is a file the binary writes, never stdout.** `gdoc completion
   zsh --out <path>` writes the script and prints the JSON report of what it
   wrote. It refuses an existing path without `--force`, the rule `build
   --out` holds. A script on stdout would be the one command whose stdout is
   not an object, and the contract is worth more than one file that has to be
   rewritten after an upgrade. `install.sh` rewrites it on every run.
6. **zsh and bash now, PowerShell at M9** with the Windows smoke test.
7. **Two skills, not one.** `gdoc-publish` starts from a note in the hub.
   `gdoc-restyle` starts from a link to a document gdoc did not write. They
   trigger on different sentences and carry different judgement. `gdoc-apply`
   stays retired; the name is not reused.
8. **A skill never holds a flag list.** Before the first call of a command in
   a session it runs `gdoc help <command>`. A boundary test over every
   `SKILL.md` holds the other direction: every command and flag a skill names
   exists in the table.
9. **No fourth module.** Cobra would give the help and the completion for
   free, and it would replace a strict parser whose refusals are tested one by
   one. Twelve commands is a small table, and `text/template` is in the
   standard library.
10. **The M9 skills line moves here.** PLAN.md's M9 said the missing skills
    land at release unless Nail wants one sooner. He does.

## Context (from discovery)

- **The dispatcher is a `switch` in `go/cmd/gdoc/main.go`**, and the usage
  line is a `const` beside it. Each command builds its own `flagSet` inline
  when it parses, in `read.go`, `write.go`, `restyle.go`, `build.go` and
  `publish.go`. `parseArgsN` is strict in six ways and every one is pinned by
  a test. The table replaces the switch and the inline flag sets, and changes
  nothing about how the parser refuses.
- **Four tests pin the help as it is.** `TestTheUsageLineNamesEveryCommand`
  asserts `--help` fails and names every command. `TestEveryCommandDispatchReachesIsInTheUsageLine`
  reads the case labels out of `dispatch` with `go/ast` and asks the usage line
  for each. `TestUnknownCommandFailsAndNamesItself` and `TestNoArgumentsFails`
  stay as they are. The first two change with the design, and the AST one
  loses its target when the switch goes.
- **`cmd/gdoc/doc.go` states the rule this milestone retires**, in the
  paragraph "The binary never prompts, and --help is a failure". Its reason,
  that readable help would have to reach stdout beside the object or exit 0 on
  a run that did no work, is answered by stderr and by decision 1. The
  paragraph is rewritten, and the doc.go names the new tests, or carries
  `TODO(test)`, as CLAUDE.md requires.
- **`freeToWrite` in `build.go`** already refuses an existing `--out` without
  `--force` and refuses a directory whatever the flag says.
  `TestBuildRefusesAnExistingOutUnlessForced` is its pin. `completion --out`
  calls it rather than writing a second one.
- **Human words go to stderr**, and `TestLoginPrintsTheURLToStderrNotStdout`
  is the pattern: run with two buffers, assert the prose is in one and not the
  other. The help tests are written the same way.
- **`os/exec` is banned tree-wide, test files included**, by
  `TestNothingRunsAnExternalProgram` in `go/boundary`. So no test runs
  `zsh -n` over the generated script. The script is checked structurally, and
  sourcing it in a real shell is a Post-Completion check for a person.
- **`go/boundary` cannot import package `main`**, so the test over the skills
  lives in `go/cmd/gdoc/skills_test.go` beside the table it checks, reading
  `../../skills/*/SKILL.md` the way boundary reads `../../Makefile`.
- **`skills/gdoc-review/SKILL.md` calls the binary as `$GDOC <command>`**,
  with `GDOC=gdoc` set in its Setup. The new skills use the same spelling, and
  the test over the skills matches both `gdoc <command>` and `$GDOC <command>`.
- **`install.sh` links one skill**, named in a single `SKILL=` variable, and
  its "What is installed" summary prints that one. `bin/` is outside git and
  every target that writes there creates it first, held by
  `TestEveryTargetThatWritesIntoBinMakesIt`.
- **The reports the skills read are already shaped.** `publish` returns
  `url`, `verified`, `checks` with `read_back`, `one_tab` and `docx_export`,
  `files_changed`, `rolled_back` and `warnings`. `restyle --dry-run` returns
  `nothing_to_protect` among its counts. `restyle --from` returns `manual` as a
  list of steps with menu paths, `verified`, and `maybe_applied`. The skills
  read those field names and never a field that is not there.
- **`cover.Fields` names the thirteen cover values**, and `restyle --fields`
  reads that struct from JSON strictly. `internal/prelude/doc.go` says what a
  second run does and when a pending prelude is refused. Task 7 reads both
  rather than this plan guessing.
- **Nail's shell is zsh under oh-my-zsh**, and `.zshrc` already sources
  kubectl's completion with `source <(kubectl completion zsh)` after oh-my-zsh
  has run. So the generated script has to register itself with `compdef` the
  way kubectl's does, and needs no `fpath` arrangement. `bash-completion@2` is
  installed too.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it
  to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own
  commit, and each ends with the test gate as its last checkbox.
- **CRITICAL: every task MUST include new or updated tests**, success and
  failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** `make test`
  runs `-race`; keep it there.
- **CRITICAL: the parser's refusals do not change.** Task 1 moves the flag
  sets into the table and every existing `cmd/gdoc` test stays green without
  edits to its assertions. A refusal that changes wording is a bug in the
  move, not a chance to improve the wording.
- **CRITICAL: one object on stdout, always.** `help` and `completion` print
  through `internal/emit` like every other command. If a task finds itself
  wanting to print a script or a paragraph to stdout, that is decision 5 being
  undone, not a convenience.
- **CRITICAL: no `os/exec` anywhere, tests included.** The boundary test says
  so and this plan does not argue with it.
- **CRITICAL: facts only in Go.** The help says what a command takes. Whether
  to run it is the skill's judgement and Nail's.
- **CRITICAL: a skill never copies a flag list.** A task writing a SKILL.md
  that spells out `--md`, `--folder-id` and `--house` in a table has copied
  what decision 8 says it must ask the binary for.
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces: code comments, commit messages,
  the skills and the docs included. Plain English, short sentences.

## Testing Strategy

- **Unit, `cmd/gdoc`, the table**: every name the dispatcher answers is in the
  table and every table entry is dispatched, asked by calling both ways rather
  than by reading source. Every flag set a command parses with is the one its
  table entry describes.
- **Unit, `cmd/gdoc`, help**: the JSON shape against literals; prose on stderr
  and none on stdout; exit 0; `--help` and `-h` at either position; bare
  `gdoc` still exit 1 with the help on stderr; `help sing` fails naming it.
- **Unit, `cmd/gdoc`, completion**: the script names every command and every
  flag in the table; a file-valued flag completes files and an id-valued one
  does not; an existing path is refused without `--force` and left byte for
  byte; a directory is refused whatever the flag says; the report names the
  path and the line to add.
- **Unit, `cmd/gdoc`, the skills**: every `gdoc <command>` and `$GDOC
  <command>` in every `skills/*/SKILL.md` names a command in the table, and
  every `--flag` on the same line is one that command takes. Both directions
  fail: a skill naming a flag that is not there, and a test that finds no
  skill files at all.
- **Boundary**: `allowedModules` does not change. M7d adds no dependency.
  `TestNothingRunsAnExternalProgram` stays green over the new test files.
- **Docs**: `TestCLAUDEmdIsUnderTheCeiling`, `TestTheTaskMapNamesFilesThatExist`
  and `TestEveryPackageHasExactlyOnePackageComment` stay green. The new
  `commands.go` opens with a file paragraph separated from the `package`
  clause by a blank line, or `go doc` joins it into the package comment.
- Coverage standard: every exported function under `go/internal/` has a test;
  `cmd/gdoc` stays at or above where it is.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- `cd go && go vet ./...`
- `make build`, then `bin/gdoc help`, `bin/gdoc help publish`,
  `bin/gdoc restyle --help` and bare `bin/gdoc`, each read for its exit code
  and for what reached which stream
- `make dist`

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

**The table.** In `go/cmd/gdoc/commands.go`:

```go
type command struct {
    name    string   // "read", or "auth status": the words a caller types
    words   []string // what each positional is called: "<url>", "<comment id>"
    flags   []flag
    summary string   // one sentence, in the reader's words
    example string   // one full line a person can copy
    run     func(ctx context.Context, rest []string, errOut io.Writer) emit.Result
}

type flag struct {
    name    string // "--md"
    value   kind   // none, file, folderID, cursor, seconds, mask
    summary string
}
```

`kind` is one field with two readers. Help prints the placeholder for it,
`<file>`, `<folder id>`, `<cursor>`, `<seconds>`, `<field mask>`, or nothing.
Completion decides from it whether to offer file names. A flag's `flagSet`
entry, name against takes-a-value, is derived from `value != none`, so the
parser is fed exactly what the table says.

The dispatcher walks `commands` in order and matches the longest name whose
words are a prefix of the arguments, so `auth status` is found before `auth`
would be. The usage line is joined from the table's names. Both are pure
functions of the slice, and nothing else in the package lists a command.

**Help.** `help` and `completion` are entries in the same table, so they help
and complete themselves. `gdoc help` returns every command. `gdoc help <words>`
returns every command whose name starts with those words, so `help auth` gives
both auth commands and `help publish` gives one. No match is `ok: false`,
`unknown command`, with the usage line, the same refusal an unknown command
gets. The JSON and the prose are two renderings of the same entries, and the
prose goes to `errOut`.

**Completion.** A `text/template` per shell over the table. The zsh script
defines `_gdoc` with `_arguments` per command and ends with `compdef _gdoc
gdoc`, so `source <path>` works after oh-my-zsh has run compinit. The bash
script defines `_gdoc` over `COMP_WORDS` and ends with `complete -F _gdoc
gdoc`. `completion <shell> --out <path>` renders, calls `freeToWrite`, writes
through `internal/atomicfile`, and reports.

**The skills.** Each SKILL.md has the shape `gdoc-review` has: front matter
with a description that names the trigger sentence, Setup with `GDOC=gdoc` and
the one-object rule, the credential paragraph, the dry-run paragraph, then the
steps, then Never. The Setup adds one rule the review skill will inherit later:
before the first call of a command in a session, run `$GDOC help <command>`
and read its words and flags from the object.

## Technical Details

**The help object.** One shape for both forms, so a skill reads one thing:

```json
{
  "ok": true,
  "data": {
    "commands": [
      {
        "name": "publish",
        "words": [],
        "flags": [
          {"name": "--md", "value": "<file>", "summary": "the note to publish"},
          {"name": "--folder-id", "value": "<folder id>", "summary": "the Drive folder the document is created in"},
          {"name": "--house", "value": "<file>", "summary": "a house style file other than the embedded one"}
        ],
        "summary": "Build the note as a house-style document and upload it into one folder.",
        "example": "gdoc publish --md note.md --folder-id 1AbC..."
      }
    ]
  }
}
```

`value` is the placeholder as help prints it, or the empty string for a flag
that takes none. A sentence in `summary` says what the thing is, in the
reader's words, never which package handles it.

**The prose on stderr.** For `help`: the usage line, then one line per
command, name and summary, aligned. For `help <words>`: for each matched
command, the name with its words and flags on one line, the summary, each flag
on its own line with its placeholder and sentence, then `Example:` and the
example. Under twenty words a sentence, and no package names.

**`--help` anywhere.** Before the table is walked, the dispatcher looks for
`--help` or `-h` in the arguments. If found, the remaining arguments are the
command words and the result is `help` over them. This is what keeps
`gdoc restyle --from x --help` an answer rather than a refusal of `--help` as
an unknown flag, and it runs before `parseArgsN` so every parser refusal is
unchanged.

**The completion report.**

```json
{"ok":true,"data":{"shell":"zsh","wrote":"/Users/nail/.gdoc.zsh","add_to_zshrc":"source /Users/nail/.gdoc.zsh"}}
```

For bash the key is `add_to_bashrc`. The path is absolute, resolved the way
`build` resolves `--out`.

**What completes.** The first word: every table name's first word, with the
second word offered after `auth`. Then that command's flags, and a flag already
on the line is not offered again. A `file` flag offers file names. Every other
kind offers nothing, because nothing on the machine knows a folder id, a
cursor or a wait length. A document URL offers nothing either.

**Global constraints:**

- Exactly one JSON object on stdout, exit 0 if and only if `ok`. No prompting,
  no stdin.
- Strict argument parsing, as everywhere else. `help` takes words and no
  flags. `completion` takes one word, `zsh` or `bash`, and `--out` with
  `--force`; anything else is refused by name.
- Nothing under `go/` imports `os/exec`, tests included.
- `install.sh` never edits `.zshrc`. It prints the line to add.

## Implementation Steps

### Task 1: the command table, and the dispatcher over it

**Files:**
- Create: `go/cmd/gdoc/commands.go`
- Create: `go/cmd/gdoc/commands_test.go`
- Modify: `go/cmd/gdoc/main.go`, `read.go`, `write.go`, `restyle.go`, `build.go`, `publish.go`
- Modify: `go/cmd/gdoc/main_test.go`

- [x] Test first, `TestEveryCommandInTheTableIsDispatchedAndNothingElseIs`:
      for each table entry, call `dispatch` with its name and a deliberately
      bad flag, and assert the refusal is the command's own and not `unknown
      command`; then call with a word that is in no entry and assert `unknown
      command`. This replaces `TestEveryCommandDispatchReachesIsInTheUsageLine`,
      which read the switch that no longer exists.
- [x] Test, `TestEachCommandParsesWithTheFlagSetItsTableEntryDescribes`: for
      each entry, the `flagSet` derived from its flags is the one the command
      hands to `parseArgsN`. Written so a flag added to a command and not to
      its entry fails here.
- [x] `commands.go`: the `command` and `flag` types, the `kind` enum, the
      `commands` slice with all twelve entries, `flagSet()` on a command, and
      `usageLine()` joined from the names. Each summary in the reader's words,
      each example a line a person can copy.
- [x] `main.go`: `dispatch` walks the table by longest matching name. The
      `usage` const goes; the unknown-command and no-command refusals print
      `usageLine()`. The two-word `auth` rule stays true by construction and
      its test stays green.
- [x] Each `cmd*` function takes its `flagSet` from its table entry instead of
      an inline literal.
- [x] `TestTheUsageLineNamesEveryCommand` keeps its literal list of twelve and
      asserts against the unknown-command refusal rather than `--help`, since
      `--help` changes meaning in Task 2.
- [x] Every existing `cmd/gdoc` test passes without an assertion changed.
- ➕ Test, `TestEveryFlagIsReadTheWayItsKindSays`: a flag the table says
      carries a value is read for one, through `a.flags` or `required`, and a
      flag that carries none is read for its presence and nothing else. Added
      because the two tests above cannot catch a `kind` that lies: the parser
      is fed the table either way.
- ➕ Test, `TestNoCommandBuildsAFlagSetOfItsOwn`: no production file in the
      package builds a `flagSet` any more, which is the other direction of the
      flag-set test and what makes it more than a tautology.
- ⚠️ Scope, the `run` signature. The plan wrote
      `run(ctx, rest []string, errOut)` with each command parsing its own rest.
      It is `run(ctx, a *args, errOut)` instead: `dispatch` parses with the
      entry's flag set and `len(words)`, and hands the command its `*args`.
      One parse call rather than twelve, and "a command parses with the flag
      set its entry describes" is then true by construction rather than by
      convention. Every parser refusal is unchanged, because the parser and
      what it is fed are unchanged.
- ⚠️ Scope, the `kind` enum has no `mask`. The plan listed one, and no flag in
      the twelve is a field mask. `restyle --fields` names a file. Added when a
      flag needs it.
- ⚠️ Scope, `go/cmd/gdoc/doc.go` is touched in this task, not only in Task 9.
      It named `TestEveryCommandDispatchReachesIsInTheUsageLine`, which is
      gone, and said `gdoc auth login --token /path` is an unknown command.
      It is now refused naming the flag, which is the same rule one word later.
      The "--help is a failure" paragraph is untouched and stays for Task 2.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "refactor(v2): one command table, and the dispatcher over it"`

### Task 2: help

**Files:**
- Create: `go/cmd/gdoc/help.go`
- Create: `go/cmd/gdoc/help_test.go`
- Modify: `go/cmd/gdoc/commands.go`, `main.go`, `doc.go`

- [x] Test first, `TestHelpIsOneObjectAndTheProseIsOnStderr`: run `help` with
      two buffers; stdout decodes as one object with `ok: true` and
      `data.commands` naming all twelve plus `help` and `completion`; stderr
      holds the usage line and each name; exit 0.
- [x] Test, `TestHelpForOneCommandCarriesItsWordsFlagsAndExample`: `help
      publish` returns one entry whose flags, placeholders and example match
      literals written in the test, not read from the table.
- [x] Test, `TestHelpMatchesByPrefixAndRefusesWhatItDoesNotKnow`: `help auth`
      returns two entries; `help sing` is `ok: false`, exit 1, naming `sing`
      and the usage line.
- [x] Test, `TestDashDashHelpIsAnAliasAnywhereOnTheLine`: `--help`, `-h`,
      `read --help`, `--help read` and `restyle --from x --help` each answer
      the same as `help` or `help read` or `help restyle`.
- [x] Test, `TestBareGdocStillFailsAndPrintsTheHelpToStderr`: no arguments is
      `ok: false`, exit 1, the object unchanged from today, and the full help
      on stderr. `TestNoArgumentsFails` stays as it is.
- [x] Test, `TestHelpTakesWordsAndNoFlags`: `help --md x` is refused by name.
- [x] `help.go`: the `help` entry in the table, the prefix match, the JSON
      rendering and the prose rendering. The alias check in `dispatch` before
      the table walk.
- [x] `doc.go`: the paragraph "The binary never prompts, and --help is a
      failure" becomes "The binary never prompts, and help is an answer",
      stating the rule, the reason, and naming the tests above. The section
      "The twelve commands" says the table is the one description and names
      Task 1's tests.
- ⚠️ Scope, the table is a function and not a variable. `help` is an entry in
      the table and reads the table, so `var commands` refers to itself through
      a function, which Go refuses as an initialization cycle. `commands()`
      hands each caller its own slice. `commands_test.go` reads `commands()`
      instead of `commands`; no assertion in it changed.
- ⚠️ Scope, "the usage line" on stderr is `Usage: gdoc <command> [words]
      [flags]`, not the comma-joined line a refusal prints. Printing both would
      name every command twice on one screen, which is principle 4 read
      backwards. The refusal is unchanged and still carries `usageLine()`.
- ➕ `everyCommandName` in `help_test.go` is the literal list `help` must come
      back with, and Task 3 adds `completion` to it. The count is thirteen
      today and fourteen after Task 3, which is what Task 10 checks.
- ➕ `command.anyWords` and `command.wants()`: `help` takes however many words
      it is handed, because the words are another command's name and that is
      two, one, or none. `parseArgsN` already reads a negative want as "any",
      so no parser refusal changed.
- ➕ `unknownCommand` moved beside `usageLine` in `commands.go`, because both
      `dispatch` and `help` refuse a word the table does not carry, and they
      must refuse it with the same sentence.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): gdoc help, one object on stdout and the prose on stderr"`

### Task 3: completion for zsh

**Files:**
- Create: `go/cmd/gdoc/completion.go`
- Create: `go/cmd/gdoc/completion_zsh.tmpl` (embedded)
- Create: `go/cmd/gdoc/completion_test.go`
- Modify: `go/cmd/gdoc/commands.go`, `build.go` (only if `freeToWrite` needs
  to move to a shared file), `doc.go`

- [x] Add `completion` to `everyCommandName` in `help_test.go`, which is the
      literal list `gdoc help` must come back with.
- [x] Test first, `TestTheZshScriptNamesEveryCommandAndEveryFlag`: render
      the script and assert every table name and every flag appears, that a
      `file` flag is followed by `_files` and an id flag is not, that the
      script opens with `#compdef gdoc` and ends with `compdef _gdoc gdoc`.
- [x] Test, `TestCompletionWritesTheFileAndReportsTheLineToAdd`: `completion
      zsh --out <tmp>` writes the script, stdout is one object with `shell`,
      `wrote` as an absolute path and `add_to_zshrc`, exit 0.
- [x] Test, `TestCompletionRefusesAnExistingOutUnlessForced`, in the shape of
      `TestBuildRefusesAnExistingOutUnlessForced`: refused, file left byte for
      byte, then replaced with `--force`. A directory is refused with the flag.
- [x] Test, `TestCompletionArgumentsAreStrict`: no shell word, an unknown
      shell, a missing `--out`, and an extra word are each refused by name.
- [x] `completion.go`: the `completion` entry, the template data built from
      the table, `freeToWrite` reused, the write through `internal/atomicfile`.
- [x] `doc.go`: a section "Completion is a file, and the reason is the output
      contract", naming the tests.
- ⚠️ Scope, `completion` counts its own word. Its entry carries `anyWords`,
      as `help` does, and `oneShell` refuses a missing word, an unknown shell
      and an extra word by name. The parser's refusal for one missing word
      says "this command needs a document", which is the wrong sentence for a
      shell, and changing that sentence would change a refusal every other
      command shares.
- ⚠️ Scope, no `zsh -n` in a test, as the plan said. The script was checked by
      hand in this iteration instead: `zsh -n` is clean and a bare `zsh -f`
      sources it and registers `_gdoc`. Typing Tab at it is Task 10's.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): gdoc completion zsh, written to a file"`

### Task 4: completion for bash

**Files:**
- Create: `go/cmd/gdoc/completion_bash.tmpl` (embedded)
- Modify: `go/cmd/gdoc/completion.go`, `completion_test.go`

- [x] Test first, `TestTheBashScriptNamesEveryCommandAndEveryFlag`: the same
      assertions as zsh over the bash rendering, ending with `complete -F
      _gdoc gdoc`, with `compgen -f` after a `file` flag and nothing after an
      id flag.
- [x] The report key is `add_to_bashrc`, asserted, in
      `TestCompletionBashWritesTheFileAndNamesBashrc`, which also asserts the
      bash report never carries `add_to_zshrc`.
- ➕ `shells()` became a table of rows, each a shell with the key its line to
      add comes under and the function that renders it. The report is a map
      rather than a struct, because the third key names the shell's own file.
- ➕ `byFirstWord` was lifted out of `zshGroups`, because both renderings
      group the table the same way and only say it differently. `doc.go` says
      so in the completion section.
- ⚠️ Scope, no `bash -n` in a test, as with zsh. The script was checked by
      hand in this iteration: `bash -n` is clean, and sourcing it and driving
      `_gdoc` with `COMP_WORDS` set gives the commands at word one, `status
      login` under `auth`, the flags under `build`, `--house` for `--h`, and
      nothing after `--wait`.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "feat(v2): gdoc completion bash"`

### Task 5: install.sh writes the completion and links every skill

**Files:**
- Modify: `install.sh`

- [x] After the binary is built, `"$GO_BIN" completion zsh --out
      "$REPO/bin/gdoc.zsh" --force >/dev/null`, and the summary prints the
      `source` line when `.zshrc` does not already contain it. The script
      never edits `.zshrc`.
- [x] `SKILL=gdoc-review` becomes a list of three, and the link block runs
      once per skill with the same three branches: a link is repointed, a
      differing real directory is refused, an equal one is replaced.
- [x] The "What is installed" summary lists every skill linked.
- [x] The comment at the top of the file names the completion file and says it
      is rewritten on every run.
- [x] Run `./install.sh` on this machine and paste its output into this task
      as the check. `gdoc help` on PATH answers.

  ⚠️ The three-skill run is red until Task 7 lands, because `skills/gdoc-publish`
  and `skills/gdoc-restyle` are written in Tasks 6 and 7. Every source is
  checked before any link is made, so the run stops with nothing half done:

  ```
  install: missing /Users/nailkhusnullin/src/personal/gdoc/skills/gdoc-publish
  exit: 1
  ```

  The same script with `SKILLS=(gdoc-review)` runs green, which is the check of
  the completion, the loop and the summary:

  ```
  gdoc installed

    source   /Users/nailkhusnullin/src/personal/gdoc
    version  e70768c on gdoc-v2-m7d-help-completion-skills + uncommitted changes
    gdoc     /Users/nailkhusnullin/.local/bin/gdoc -> /Users/nailkhusnullin/src/personal/gdoc/bin/gdoc
    complete /Users/nailkhusnullin/src/personal/gdoc/bin/gdoc.zsh
             add this line to ~/.zshrc:  source /Users/nailkhusnullin/src/personal/gdoc/bin/gdoc.zsh

    skills (linked, so edits are live with no reinstall)
      gdoc-review  -> /Users/nailkhusnullin/src/personal/gdoc/skills/gdoc-review

    next     gdoc auth status, and gdoc auth login if it says signed out
  ```

  `gdoc help` on PATH answers with one object and exit 0. The full three-skill
  run is a checkbox in Task 10.
- [x] `git commit -m "chore: install.sh writes the completion and links three skills"`

### Task 6: the gdoc-publish skill

**Files:**
- Create: `skills/gdoc-publish/SKILL.md`

- [x] Front matter: `name: gdoc-publish`, and a description in one sentence
      that names the trigger: Nail names a note in the hub and a Drive folder
      and wants the note in Drive as a Google Doc in the house style.
- [x] Setup, in `gdoc-review`'s shape: `GDOC=gdoc`, `ROOT="$PWD"`, the
      one-object rule, and the new rule: before the first call of a command in
      this session, run `$GDOC help <command>` and read its words and flags
      from the object. The skill lists no flag itself.
- [x] The credential paragraph and the dry-run paragraph, as the review skill
      has them. A dry run here is `build` to a scratch path under the session's
      scratchpad, which touches no network, and the report read back.
- [x] Steps: read the note's front matter first, and if a `gdoc:` block already
      names a document, repeat the binary's refusal in Nail's words and never
      remove the block. Then `publish` with the folder Nail named. Read the
      object back and say: the URL, what `verified` and each check say, what
      `files_changed` lists, whether `rolled_back` is set, and every warning.
- [x] After a publish, say the three things only a person can check by opening
      the document: the logo in the first-page header, the contents list, the
      footer page numbers. Facts the binary printed, judged by Nail.
- [x] Never: never remove or edit a `gdoc:` block, never run git, never pass a
      folder the binary was not handed, never call `publish` twice on one note.
- [x] `git commit -m "feat(skill): gdoc-publish"`

### Task 7: the gdoc-restyle skill

**Files:**
- Create: `skills/gdoc-restyle/SKILL.md`

- [x] Read `go/internal/prelude/doc.go`, `go/internal/restyle/doc.go` and
      `cover.Fields` first. The cover values and the second-run rules in this
      skill come from those, not from this plan.
- [x] Front matter: `name: gdoc-restyle`, and a description naming the
      trigger: Nail gives a link to a Google Doc gdoc did not write and wants
      it in the house style where it stands.
- [x] Setup, credential and dry-run paragraphs as in Task 6, with the same
      help-first rule.
- [x] Step 1, the survey, always: `restyle <url> --dry-run` to a file, then say
      in plain words what the document holds: threads, pending suggestions,
      chips, tabs, named ranges. If `nothing_to_protect` is true, offer the
      read-note-publish route in one sentence and never take it alone.
- [x] Step 2, the cover: ask whether the house cover is wanted. If yes, propose
      each of the thirteen values from the document and the hub, show them,
      and write the fields file only after Nail confirms. `Title` is never
      invented: it is proposed and confirmed.
- [x] Step 3, the styling run: say what will be sent and ask once more, because
      this is the one direct edit gdoc ever makes. Then `restyle <url> --from
      <survey> [--fields <fields>]`.
- [x] Step 4, the report: read `manual` out loud, each step with its menu path;
      say `verified`; say if the run stopped half way and what that leaves;
      say that a restyle is a moment and not a setting, so the next heading
      Nail types will not carry the house look.
- [x] Never: never run the styling without the survey read in this session,
      never retry a batch Docs refused, never run on a document with more than
      one tab, never invent a cover value, never run git.
- [x] `git commit -m "feat(skill): gdoc-restyle"`

### Task 8: the test over every skill

**Files:**
- Create: `go/cmd/gdoc/skills_test.go`, which reads `../../../skills/*/SKILL.md`

- [x] Test first, `TestEverySkillNamesOnlyCommandsAndFlagsTheBinaryHas`: walk
      `../../skills/*/SKILL.md`, find every `gdoc <words>` and `$GDOC <words>`,
      resolve the words against the table by longest prefix, and assert a
      match; for each `--flag` on the same line, assert the matched command
      takes it. A `--flag` on a line with no command is ignored.
- [x] The test fails when it finds no skill files, so a moved directory is a
      failure rather than an empty pass.
- [x] Run it against the three skills; fix any stale call in `gdoc-review`
      this finds, as its own checkbox added here with ➕. It found none: every
      call in the three skills names a command in the table with flags that
      command takes.
- [x] ➕ The skills wrap an inline call across two lines, so the span reader
      follows a call cut in half by a line break. Outside a code fence only
      `$GDOC` opens a call, because prose puts a warning gdoc printed in
      backticks too.
- [x] ➕ `TestEverySkillNamesOnlyCommandsAndFlagsTheBinaryHas` also fails when a
      skill file carries no call at all, so a rewrite that stops naming the
      binary is a failure rather than an empty pass.
- [x] `cd go && go test -race ./...` passes.
- [x] `git commit -m "test(v2): every skill names only commands and flags the binary has"`

### Task 9: the documents

**Files:**
- Modify: `docs/v2/SPEC.md`, `docs/v2/PLAN.md`, `CLAUDE.md`,
  `go/cmd/gdoc/doc.go`

The DECISIONS.md entry for this milestone is already written, dated
2026-09-16, with its register row, because a SPEC change is a DECISIONS entry
first. This task makes the other documents agree with it.

- [x] SPEC.md "The binary": a paragraph on `help` and `completion`, in the
      present tense, with one sentence each on the output contract holding and
      on completion being a written file.
- [x] SPEC.md "Output contract": unchanged in every bullet. Read it and confirm
      rather than assume, since decision 1 was designed to keep it so.
- [x] SPEC.md "The skills, and how a comment reaches one": four skills, two
      Nail-invoked, and the help-first rule as one sentence.
- [x] PLAN.md: the M7d section under "Done" becomes a row in Task 11. M9's
      line "The skills that do not exist yet land here" and its two bullets go,
      replaced by one sentence that PowerShell completion lands at M9. M9
      already reads that way, so the Done row is all that is left, and that row
      is Task 11's.
- [x] CLAUDE.md "What lives where": `go/cmd/gdoc/` row says twelve commands
      plus help and completion. `skills/` row names three skills. "If you
      touch": the command row adds "its help"; a new row for a skill points at
      `skills/`. The invariant "Skills are linked, not copied" names all three.
      Stays under 300 lines.
- [x] `go/cmd/gdoc/doc.go`: every rule this milestone added names its test.
      `TestEveryPackageHasExactlyOnePackageComment` green.
- [x] `cd go && go test -race ./...` passes, the docs tests included.
- [x] `git commit -m "docs(v2): help, completion and the two skills in SPEC, PLAN and CLAUDE.md"`

### Task 10: verify acceptance criteria

- [x] `make build`; `bin/gdoc help` exits 0 with one object on stdout and the
      prose on stderr, checked with `bin/gdoc help >/dev/null` and
      `bin/gdoc help 2>/dev/null | head -c 1` reading `{`.
      Both hold: exit 0, stdout opens `{`, the fourteen-line list is on stderr.
- [x] `bin/gdoc help publish`, `bin/gdoc restyle --help`, `bin/gdoc -h read`
      each answer. Bare `bin/gdoc` exits 1 with the help on stderr.
      All three exit 0 with the one object on stdout and the usage of that one
      command on stderr. `completion --help` answers the same way. Bare
      `bin/gdoc` exits 1 with the full list on stderr and nothing on stdout.
- [x] `bin/gdoc completion zsh --out /tmp/gdoc.zsh --force`, then in a fresh
      `zsh -f`: `autoload -Uz compinit && compinit && source /tmp/gdoc.zsh`,
      then type `gdoc pub<Tab>` and `gdoc publish --<Tab>`. Record what was
      offered here. This is a person at a keyboard, not a test.
      Driven through a real interactive `zsh -f -i` on a pseudo-terminal
      (`zmodload zsh/zpty`), because this session has no keyboard. What came
      back, verbatim:

      ```
      gdoc pub<Tab>        -> gdoc publish          (unique match, completed)
      gdoc comp<Tab>       -> gdoc completion       (unique match, completed)
      gdoc publish --<Tab> -> --folder-id  -- the Drive folder the document is created in
                              --house      -- a house style file other than the one inside gdoc
                              --md         -- the note to publish
      gdoc <Tab>           -> auth, build, comments, completion, help, probe,
                              propose, publish, read, reply, restyle,
                              suggestions, withdraw, each with its sentence
      ```

      Thirteen names at the top level, `auth` standing for its two words, which
      is the table read back. Nail still does the keyboard pass in
      Post-Completion; this records that the script loads and offers.
- [x] `bin/gdoc help 2>/dev/null | python3 -c 'import json,sys; d=json.load(sys.stdin); print(len(d["data"]["commands"]))'`
      prints 14: twelve commands plus `help` and `completion`. It prints 14.
- [x] `grep -c -- '--' skills/gdoc-publish/SKILL.md skills/gdoc-restyle/SKILL.md`
      shows no flag list: the only `--` occurrences are inside the example
      lines the help-first rule quotes, if any.
      Two each, and both are the front matter fences on lines 1 and 4. Neither
      skill names a single flag.
- [x] `make test`, `make vet`, `make dist` green. CI runs when the branch is
      pushed, which is Nail's to do.
      `make vet` clean, gofmt clean, `go test -race ./...` all packages ok,
      `make dist` wrote the three binaries into `bin/`.
- [x] `wc -l CLAUDE.md` under 300. It is 228.
- [x] ➕ `./install.sh` runs green with all three skills present, which Task 5
      could not check because two of them did not exist yet.
      Green. It wrote `bin/gdoc.zsh`, linked `~/.local/bin/gdoc`, and listed
      `gdoc-review`, `gdoc-publish` and `gdoc-restyle` as symlinks into
      `skills/`. The tree was clean, so no `+ uncommitted changes` line.

### Task 11: close the milestone

- [ ] Move this plan to `docs/plans/completed/`.
- [ ] PLAN.md's done table gains the M7d row, and the M7d section under open
      work goes.
- [ ] `git commit -m "docs(v2): close M7d"`

## Post-Completion

**Manual verification:**

- Add `source ~/src/personal/gdoc/bin/gdoc.zsh` to `.zshrc` by hand, open a
  new terminal, and use the completion for a day. What it offers wrongly, or
  fails to offer, is a backlog item each.
- Start a fresh Claude Code session in the hub and ask for a note to be
  published. Watch whether `gdoc-publish` triggers on the sentence, runs
  `gdoc help publish` first, and reads the report back in plain words.
- Do the same with a link and `gdoc-restyle`, on a copy of a real document,
  through to the `manual` list read out loud.
- Read `bin/gdoc help 2>&1 >/dev/null` as a person and cut every sentence that
  needs translating. Principle 4 is judged by reading, not by a test.

**External:**

- PowerShell completion and the Windows smoke test are M9's.
- `zsh-autosuggestions`, the grey text from history as you type, is an
  oh-my-zsh plugin and not gdoc's. Nail adds it to `plugins=(...)` if he wants
  it.

## What this milestone leaves for later

- Whether `gdoc-review` should also run `gdoc help` before its first call is a
  one-line edit to that skill. It is not done here because that skill is live
  and works; it is a backlog item if the boundary test in Task 8 finds it
  stale.
- Whether `help` should print examples that are real document ids from a
  configured test folder is a question for after a week of use.
