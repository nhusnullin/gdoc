# gdoc v2 Milestone 16: settings, the mark, and answers an AI can act on. Specification

2026-10-04. Every decision below is Nail's and settled. The task list is
`docs/plans/2026-10-04-gdoc-v2-m16-settings.md`. On 2026-10-13 Nail moved Part
C, the token in the Keychain and signed releases, to M17
(`docs/plans/2026-10-13-gdoc-v2-m17-keychain-spec.md`), so v2.11.0 does not
wait for the certificate. In v2.11.0 the token stays the file it is today, and
nothing is signed. It grew out of the 2026-10-03 brainstorm on how a
colleague's own skill uses gdoc (George's `docs-gdoc-suggestion-edits`). It
was checked against main after M14 and M15 merged (`ed9d269`).

Two parts. Part A is small fixes that change no contract. Part B is the
settings, with the thread mark as the first one.

## Principles

Serves: 4, every word costs a reader's attention. A colleague picks a mark that
is quieter than 🤖, and an AI that reads gdoc's refusals is told what to do
next.

Strains, each needing a DECISIONS.md entry before code:

- "Everything gdoc writes into a thread opens with the robot prefix". The
  prefix becomes the mark in effect, and the mark may be empty.
- "The marker is the only record of authorship there is". With an empty mark
  there is none.

## Decisions taken

Nail, 2026-10-04:

| Rule | Part |
|---|---|
| The thread mark is a setting. 🤖, ⁘, ∴, an empty mark and other values are allowed | B |
| An empty mark is allowed: gdoc then writes the words with nothing in front | B |
| Settings live in a file, and the path to that file can be given as an argument when gdoc starts, so tests run in parallel with their own settings | B |
| Every setting can also be given as a direct argument, which overrides the file | B |
| Any symbol may be set as the mark. gdoc suggests a default | B |
| The default mark, for a colleague who sets nothing, is ⁘ | B |
| `reply --body "<text>"` joins this milestone, and chat's `body` maps to it | B |
| The token is not a setting. It never lives in the settings file | B |
| Part C, the token in the Keychain and signed releases, moves to M17 (2026-10-13) | none |

From the 2026-10-03 brainstorm, taken then:

| Rule | Part |
|---|---|
| George keeps his own skill. gdoc's own answers have to be enough for it | A |
| Removing `propose --folder` stays in its backlog item, as M14 decided | none |

## Part A: answers an AI can act on

No contract changes. Each is one commit, test first.

1. Three refusals stop naming a person: `propose.go:178`, `annotate.go:67`,
   `cmd/gdoc/annotate.go:93` say "the reader has to guess at".
2. `propose/doc.go:306` says "both block routes", and a new test pins that a
   words proposal lands inside a table cell. No offline test pins it today;
   the only evidence is the 2026-09-18 live run in MEASURED.md.
3. An unknown field in a `--from` file is refused with the fields that entry
   takes, decoded entry by entry, still strictly. Today: `json: unknown field
   "reason"`.
4. A deletion is refused with the way out: quote a neighbour word and keep it
   in the replacement.
5. A refusal during the write names the proposal by number and quote.
6. The `--from` summary in help names `why`, the one required field it leaves
   out.
7. `propose` reads the document once before its first write, not twice.

## Part B: settings

### The file and the arguments

- The settings file is `settings.yaml` in the config folder, decoded strictly:
  an unknown key is refused by name. No file means every default.
- `--config <file>` is accepted by every command and by `gdoc mcp`. It names the
  settings file to use instead of the default one. A named file that does not
  exist is refused, never treated as empty.
- Each setting is also an argument of its own. The first is `--mark <value>`.
  An argument overrides the file. `--mark ""` is the empty mark.
- The order: argument, then the file in use, then the built-in default.
- `help --json` carries the settings in effect and where each one came from:
  `settings: {mark: "⁘", mark_from: "file"}`. A fact, not advice.
- The token, its path and the update stamp are not settings.

### The mark

- The mark in effect opens everything gdoc writes into a thread: a reply, a
  proposal's comment, an annotation. Then one space, then the words. An empty
  mark writes the words alone.
- `reply` adds the mark itself, as `propose` and `annotate` already do. Today
  `reply` refuses a body without 🤖, and M14's chat text tells the model to
  type it. Both change. For one release a body that already opens with a known
  mark is accepted with a warning, and the mark is not doubled.
- A value is refused when a thread would show it wrong: a line break,
  markdown, one of gdoc's document markers, or more than eight characters.
- Recognising gdoc's own replies (`comments.Reply.ByGdoc`, chat's
  `robot_not_ours`) checks the list in decision B2, never the account.

### Open decisions for Part B

- B1, settled 2026-10-04 (Nail: the mark serves the review, so its judgement
  belongs to the skill). What a mark means is the skill's business: skipping a
  thread already answered, never taking gdoc's own words as an instruction, and
  naming who asked. The binary keeps two jobs: it writes the mark in effect, and
  it prints `by_gdoc` as a fact, "this text opens with a known mark or the mark
  in effect", because chat has no skill and every skill would otherwise carry
  its own copy of the list. With an empty mark `by_gdoc` is false for that
  person's replies, help and the README say what that costs, and the binary
  keeps no record of the replies it wrote.
- Chat's `robot_not_ours` fact is removed. Since `7fea76a` it flags nothing and
  is only shown. M14 recorded it, so its removal is a DECISIONS.md line.
Settled 2026-10-04, kept here for the reasoning:

- B2. Any value may be set. Each gdoc recognises the known list (🤖, ⁘, ∴)
  plus its own mark. A colleague's mark outside the list is read as a person's
  reply by everybody else's gdoc, which is the price of free choice.
- B3. The default is ⁘. Every colleague's replies change on their next update.
  🤖 stays recognised forever, because old replies keep it.
- B4. `reply --body "<text>"` joins, beside `--body-file`. Chat's `body` maps
  to it, so chat stops writing a temp file per reply.

## Not in this milestone

Each stays a backlog item or an idea until Nail names it:

- A line break hides from `read` (`docs/backlog/a-line-break-hides-from-read-and-propose.md`).
- Help that carries an example input file, as a new field in the help object.
- Picking one cell when a value repeats (`near`).
- A proposal that answers the thread that asked (`reply_to`).
- Stable refusal codes and a feature list for skill authors.
- Removing `propose --folder`.
- The token in the macOS Keychain, and releases signed on Nail's Mac: M17.

## Tests that will be named

Written into DECISIONS.md by the first task, once the decisions above are in:

- `TestNoRefusalNamesAPerson`
- `TestFindSpanPlacesAQuoteInsideATableCell`
- `TestAnUnknownFieldIsRefusedWithTheFieldsTheEntryTakes`
- `TestASettingArgumentOverridesTheFile`
- `TestAConfigFileThatDoesNotExistIsRefused`
- `TestReplyAddsTheMarkInEffect`
- `TestAnEmptyMarkWritesTheWordsAlone`
