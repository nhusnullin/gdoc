# Checklist tasks from a note

Asked by Nail on 2026-11-05: a note line `- [ ] task` should arrive in the
Google Doc as a checklist item, with whom and by when on it. He writes the
questions from a source (a PDF) into a note as tasks and publishes the note.
He also asked for the same in chat (`gdoc mcp`).

Status: research done, waiting on Nail's five answers. No decision yet, so
nothing here is in SPEC.md.

## What gdoc does today

- `publish`: goldmark parses the task list, and `internal/body` writes the
  ballot character into the text, `☐ ` or `☒ `
  (`TestTaskListsCarryTheirBallotCharacter`). The document shows a character,
  not a box anyone can tick.
- `propose`: a block proposal refuses a task list by name ("a task list is
  not something a block proposes").
- `read` and `export`: a checklist in a document reads as an unordered list,
  `- item`. Nothing marks it as a checklist.
- Chat has `read`, `comments`, `suggestions`, `reply`, `annotate` and
  `propose`. It has no `publish`.
- An assigned comment works today from the CLI (`annotate`, `propose`,
  `assignee`), and chat refuses it on purpose (`TestAssigneeIsRefused`).

## What Google's API offers

Read from the Docs discovery document, revision 20261028, and the Tasks
discovery document, revision 20261103. The Docs `Request` now has 53 kinds,
not the 40 that BLOCKED-BY-API.md counted. `insertPerson`, `insertDate`,
`insertRichLink` and the dropdown kinds are new since then.

| Want | API | Answer |
|---|---|---|
| A checklist item | `createParagraphBullets` with `BULLET_CHECKBOX` | Yes |
| A person chip | `insertPerson` with an email | Yes |
| A date chip | `insertDate` with a timestamp and a format | Yes |
| A ticked box | none | No field in the schema, to read or to write |
| A Docs task with an assignee and a due date, the kind that lands in Google Tasks | none in Docs. Tasks `assignmentInfo` is output only | No |
| An assigned comment, which emails the person | `insertComment.assigneeEmailAddress` | Yes, and gdoc already uses it |

So the real "Assign as a task" button in Docs stays a person's click. What a
program can build is a checklist line with a name chip and a date chip, which
looks like a task and does not notify anyone, plus an optional assigned
comment, which does.

## Measured 2026-11-05

`TestLiveChecklistProbe` in `go/internal/live/checklistprobe_test.go`. How
taken: the real token through the real guard, four throwaway documents the
guard's own create made in the Drive test folder, each trashed after. The
person chip names Nail's own address.

| Case | Result |
|---|---|
| Direct, checklist only | Lands. The list level reads `glyphType: GLYPH_TYPE_UNSPECIFIED`, `glyphFormat: "%0"` and no `glyphSymbol` |
| Direct, checklist with a person chip and a date chip | Lands. The person reads back with his name filled in by Google. The date reads `displayText: "Nov 20, 2026"` |
| SUGGEST, checklist only | Lands as one suggestion: one id on each paragraph's `suggestedBulletChanges` and on the list's `suggestedInsertionId` |
| SUGGEST, person chip and date chip | Land as suggestions, one id per insert, on the chip and on the text before it |

A plain disc bullet reads with `glyphSymbol: "●"` in the fixtures, so the tell
for a checklist is a level with no number and no symbol. This tell is inferred
and not documented by Google.

The guard needs no change. A document gdoc created is at the full level. On a
handed-in document, SUGGEST mode carries kinds the guard does not know, and
these three are not in any refused family.

Measured the same day, `TestLiveChecklistProbeOverBullets`, same conditions:

| Case | Result |
|---|---|
| SUGGEST, one batch: two new paragraphs, a checklist over them, a person chip on the first | One suggestion id on everything, the text, the chip, the list. `withdraw` can take it back whole |
| Direct, a disc list of three, a checklist over the first two | The whole list turns into checkboxes, the third item too. A list's look belongs to the list, not the paragraph |
| Direct, the same with two requests, one per paragraph | The same: one list, all three checkboxes |
| Direct, bullets taken off the first two, then a checklist over them | Two lists: the two tasks are a checklist, the third item stays a disc bullet |
| Direct, bullets taken off the middle one, then a checklist over it | Three runs: disc, checklist, disc, the two disc items in one list |

So a published list that mixes tasks and plain items must take the bullets
off the task paragraphs before it makes them a checklist. Taking bullets off
moved no index.

## Still to measure, by hand

**Whether a person chip mails or shares.** The trust guardian vetoed the plan
until this is measured: every chip so far named the signed-in account, and
nobody is notified about themselves. Google's own coverage of body mentions
(2020 and 2021) says a mention notifies nobody and shares nothing, and that
the person typing in the browser gets a prompt to share. No current help
page says so. The test: a chip naming an address Nail reads, that is not the
signed-in account and has no access to the document, once direct and once
in SUGGEST mode. He then says whether a mail arrived and whether the
document's sharing list changed.

Taken 2026-11-06 by `TestLivePersonChipToAnotherAddress`: two documents
kept in the test folder, a chip naming Nail's personal address (not the
signed-in account, no access to the folder), one written directly at
05:18:07Z and one suggested at 05:18:11Z. Sharing, read through the Drive
permissions list of each: the address was not added, and both lists are the
shared drive's members alone. The inbox: Nail read it, spam included, and answered "No email".

**Ticks and Docs tasks.**

`TestLiveChecklistForHands` left one document in the test folder:
https://docs.google.com/document/d/1IoTXlEWi2VUtkqEnVQstn_Rj9kXqraeMWSdDXZumPPg/edit

Nail ticks the first box and presses "Assign as a task" on the second. Then
`GDOC_PROBE_READ_ID=<that id>` with the same test reads it back. That answers
whether a tick or a Docs task shows in the read at all. It decides what `read`
and `export` can say about a ticked item.

## The notation, proposed

One line, the task first, then optional tokens at the end:

```
- [ ] Which bank holds the safeguarding account? @anna 2026-11-20
- [ ] Do we need a CASP licence in the Netherlands? @mark@example.com
- [ ] Is the card programme in scope?
- [x] Confirm the scheme.
```

- `@name` at the end of the line is the owner. A bare name takes the domain of
  the signed-in account, so `@anna` is Anna at the firm's own domain for Nail,
  and right for any colleague in another domain. A full email is also accepted. One owner per
  line, because a Docs task has one.
- An ISO date `YYYY-MM-DD` at the end of the line is the due date. ISO, because
  it is the one form that is not ambiguous between British and American readers
  and that a strict parser can refuse when it is wrong.
- Tokens are read only at the end of the line, after the words, so an `@` or a
  date inside a sentence stays text.
- Strict, as every input is: two owners, a date that does not exist, or a
  token in the middle are refused by name, before anything is sent.

Plain ASCII on purpose: typeable on any keyboard and in any editor. Obsidian
Tasks writes the due date as `📅 2026-11-20`; that could be accepted as well
later, and it is not needed to start.

## The shape, proposed

- `publish`: the docx stays as it is. After the upload, one more batch on the
  document gdoc just created turns each task paragraph into a checklist item,
  takes the ballot character out, and inserts the person and date chips. Then
  the read-back checks every task line is a checklist item with the chips it
  should carry.
- `propose`, CLI and chat: a block proposal may hold a task list. It lands as
  a suggested checklist with suggested chips, which is what was measured.
- `read`, CLI and chat: a checklist item prints as `- [ ] words`, with the
  person as `@email` and the date as `YYYY-MM-DD` at the end, so a read and a
  note use the same notation. A tick prints only if the hand test shows it.
- `export` and `align`: the same notation into the note.
- Chat stays without `publish`. The chat card for a proposal must show the
  owner's email, because a chip names a person, and the hold rules judge that
  write like any other.
- An assigned comment per task is a choice for Nail, because it emails people.
