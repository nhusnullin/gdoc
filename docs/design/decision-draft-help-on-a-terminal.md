# DRAFT: decision entry, not yet in DECISIONS.md

Status: draft for Nail. Nothing below holds until it is copied into
`docs/v2/DECISIONS.md` with its register row, on the day it is decided.

Idea stage only. No spec and no plan are written before M14 finishes
(Nail, 2026-10-03). This work ships after M14, so it may use what M14 adds,
such as `AllowAccountRead`. When M14 has landed, this draft is reread against M14's own
entry, which narrows the stdin and stdout invariants for `gdoc mcp`.

The pictures behind it: `docs/design/tui-variations.html` (round one, six
directions) and `docs/design/panels-round-two.html` (round two, every screen).

Register row it would add:

| Date | Decision | Status |
|---|---|---|
| 2026-10-03 | On a terminal, a help screen is for the person: Panels on stderr and nothing on stdout, and `--json` always prints the object | holds |

---

## 2026-10-03. On a terminal, a help screen is for the person: Panels on stderr and nothing on stdout, and `--json` always prints the object.

Nail's decision, taken in the TUI brainstorm of 2026-10-03, where he chose the
"Panels" direction and then its details with a designer and an engineer. Serves
principle 4, every word costs a reader's attention: a person who types
`gdoc help` reads a screen built for them, not a 7.5 KB JSON line wrapped under
it. Serves 1: no new module, the styling and the terminal check are standard
library only.

**What was true before.** Every command prints exactly one JSON object on
stdout, always, and the words a person reads go to stderr. The 2026-09-16 entry
"Help is an answer" applied that to `help`: the object on stdout, the prose on
stderr, exit 0. On a terminal both streams reach the same screen, so the person
sees the help and then the raw object under it. A skill reads the object and
never the prose.

**The rule.** On a terminal without `--json`, a help screen replaces the
object. Help screens are two: `help` when it answers, and bare `gdoc`. Every
other result still prints its object, refusals included. In detail:

- The decision is made in `run()` (`main.go`), which holds stdout. A result
  carries an internal marker, set only by a successful `cmdHelp` and by the bare
  branch of `dispatch`. `run()` drops the object only when the marker is set,
  stdout is a terminal, and `--json` was not given. A panic has no marker, so
  its envelope is always printed.
- `help` that answers: Panels on stderr, nothing on stdout, exit 0.
- Bare `gdoc`: Panels on stderr, nothing on stdout, exit 1. No skill calls bare
  `gdoc`, and `gdoc --json` stays an unknown command, so bare `gdoc` has no way
  to print its object on a terminal. Nothing needs one.
- A refusal, such as `gdoc help sing` or an unknown command, keeps its
  `ok: false` object on a terminal, as today.
- The last line of a help screen that dropped its object is one dim line:
  `Add --json to print the JSON object a skill reads.` It is printed on
  stderr, so stdout stays "the object or nothing". It is dim only when colour
  is on, and it is printed under `NO_COLOR` and `TERM=dumb` too, because there
  the object is also gone. It is never on a pipe, which gets the object, and
  never on bare `gdoc`, where `--json` is not a valid call.

**What "terminal" means.** The terminal driver answers for the file descriptor:
the `isatty` question, asked through `syscall` on darwin. Windows is out of
scope for this decision (Nail, 2026-10-03). It is not the
character-device bit that `progress.go` reads today, because `/dev/null` is a
character device too. `progress.go` moves to the same check, so gdoc has one
definition of a terminal.

**`--json` is the safety net.**

- A new flag on `help`. It prints the object on stdout wherever stdout goes,
  and it keeps stderr plain, with no Panels escapes.
- It exists because no check can tell a person from an agent that runs commands
  inside a pseudo-terminal, or one that types into a visible terminal tab. A
  pseudo-terminal is a terminal. Claude Code's shell tool gives gdoc pipes
  (measured 2026-10-03), so it would be safe without the flag; other agents may
  not be.
- It survives every spelling of help. Today `helpWords` (`help.go`) drops what
  is not a command word, so `gdoc help --json`, `gdoc publish --help --json`
  and `gdoc --help --json` must each keep it, with a test per form.
- Every skill passes `--json` on every `help` call, and says that a call with
  no object at all is a failed call.
- Release order, agreed by Nail: the strict parser refuses a flag it does not
  know, so the skills move to `--json` in the sitting that cuts the tag, and
  their `needs` line names that release, the order the 2026-10-02 entry used
  for `--folder`. A skill run against an older gdoc fails loudly, because the
  refusal still carries `version`.

**What a person sees.** All of it on stderr, and only when stderr is a
terminal. A pipe gets today's plain text, byte for byte, and no escape byte ever
reaches a pipe or a file. `NO_COLOR` and `TERM=dumb` turn the colour off.

- Boxes with titles in the border. No status bar: the version is in the top
  border, the channel is in the version, and "signed in" belongs to
  `auth status`. So `help` never reads the token file.
- Width: drawn at the terminal's width, up to 100. Two columns from 80, stacked
  from 50 to 79, plain text under 50. Width comes from `TIOCGWINSZ` on darwin,
  then `COLUMNS`, then 80.
- Colour: one palette for dark and light terminals, because gdoc cannot ask
  which one it is (that needs a stdin read). Text keeps the terminal's own
  colour; only accents are coloured, at mid brightness. On a 16-colour
  terminal, chips become reverse video.
- `auth status` gets a panel on a terminal, saying signed in, signed out, or
  scopes missing, and the account it signs in as. Its object stays below it,
  and carries the account too.
- The account is read live, on every `auth status`, through M14's
  `AllowAccountRead` (`GET /drive/v3/about`, field mask
  `user(emailAddress,displayName)`). This work ships after M14, so the read
  exists. `auth status` becomes its second caller beside M14's login, which is
  a widening of who may hold that grant and is written into the guard's
  `doc.go` with its test. It is read live rather than stored, because the token
  file is shared with Google's own library and its `account` field is that
  library's, and because a live read sees a swapped token. Offline, the panel
  says signed in and that the account could not be read; `auth status` does not
  fail.
- `auth login` prints the link, then one spinner line redrawn with a carriage
  return only, then `✓ signed in`. No wrap-off and no cursor-up, because login
  has no signal handler and a Ctrl-C must leave the terminal as it was.
- `update` keeps its step list and spinner in the Panels style. No download
  progress bar.
- An example is printed flush left under its box, as one line the terminal
  wraps, so a copied example runs as it is, with no line-break logic.
- A long word, such as a hash in a checksum error, is cut where the line ends.
  Nothing is shortened.
- The daily release notice and its warnings are shown in the panels, because
  on a terminal the object that carried the warnings is gone.

**What this narrows.** The invariant "One JSON object reaches stdout, always"
in `CLAUDE.md` becomes: one JSON object reaches stdout, always, except a help
screen on a terminal without `--json`, which writes nothing there. Its second
half becomes: the exit code is 0 if and only if that object says `ok`, and when
a help screen replaced the object, 0 for `help` and 1 for bare `gdoc`. The
SPEC.md line "Every command writes exactly one JSON object to stdout and exits"
gets the same exception, in the same words. After M14 lands, the rule is
written once in `CLAUDE.md`, naming this exception and M14's.

**What was rejected.**

- Detecting an AI by its environment, for example `CLAUDECODE=1`. Claude Code
  sets it and other agents do not, and identity is never a gate in gdoc.
- Hiding the object on a terminal for every command. For most commands the
  object is the answer.
- The dim hint on stdout. It would be the only non-JSON line gdoc writes there.
- `--json` on every command as a no-op, for uniformity. It is a flag accepted
  and ignored, which the strict parser exists to refuse.
- A `GDOC_JSON=1` environment variable instead of the flag. A misspelt name is
  ignored in silence. Left for later.
- Treating a terminal stdin as the sign of a person. Under a pseudo-terminal,
  stdin is the same terminal.
- A status bar, and "signed in" in the top border. Both make `help` read the
  token file and add a way for it to fail, to answer a question nobody asks
  `help`.
- A panel for an unknown command. A typo does not need a screen, and the words
  would appear twice.
- A download progress bar. It needs a counting reader for a few-MB download
  that takes seconds. On the backlog if anyone asks.
- A backslash at the end of a wrapped example. One line the terminal wraps is
  simpler and copies the same.
- Lip Gloss and Bubble Tea. Lip Gloss adds about ten modules; Bubble Tea is an
  interactive framework and gdoc never reads stdin. A stdlib-only package,
  `internal/tty`, of about 300 lines draws everything.

**Tests it needs.**

- `TestHelpOnATerminalWritesNothingToStdout`.
- `TestBareGdocOnATerminalWritesNothingToStdoutAndExitsOne`.
- `TestARefusalKeepsItsObjectOnATerminal`.
- `TestHelpWithJSONPrintsTheObjectOnATerminal`, one per spelling of help.
- `TestTheHintIsTheLastLineOnlyWhenTheObjectWasDropped`.
- `TestHelpOnAPipeIsTodaysTextByteForByte`, against a golden file.
- `TestNoEscapeByteReachesAPipe`, for every command, in the boundary package.
- `TestDevNullIsNotATerminal`.
- `TestHelpNeverReadsTheToken`.
- `TestAuthStatusNamesTheAccountItReadsLive`, and
  `TestAuthStatusOfflineStillAnswersWithoutTheAccount`.
- The guard's test that holds `AllowAccountRead` to its callers names the
  second one.
- `TestEverySkillPassesJSONToHelp`, beside the existing SKILL.md call-line test.
- `TestHelpIsOneObjectAndTheProseIsOnStderr` keeps holding for the pipe case,
  and its doc comment names the terminal exception.

## Still open

0. **Bare `gdoc`, agreed by Nail, 2026-10-03.** Its screen
   opens with the object's own words, `gdoc needs a command.`, so a person sees
   why they got help. It carries no `--json` hint, because `gdoc --json` is not
   a valid call and the screen is about a slip, not about skills.
1. **Unknown command on a terminal.** It keeps its object, as today. Whether it
   should drop it like bare `gdoc` is left for later.
2. **Reread after M14**, as the status line at the top says.

Windows is out of scope (Nail, 2026-10-03). Its escape-code question stays where
it already is, item 10 of `docs/backlog/windows-rollout-checklist.md`.
