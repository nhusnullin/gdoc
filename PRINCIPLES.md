# Principles

Read this before proposing any design. It is short on purpose.

Two layers. **Principles** are durable constraints on the world the tool runs
in. They have a reason that does not expire, and there are four, the fourth
added 2026-08-29. They are this file. **Decisions** are today's implementation:
dated, naming the principle they serve, and retiring one is an ordinary edit.
They live in [docs/v2/DECISIONS.md](docs/v2/DECISIONS.md).

When a change arrives, the layer tells you what kind of change it is. A decision
you edit. A principle you argue about first.

## 1. It runs on someone else's machine

The CLI and the skills get distributed. Some of those machines are isolated
environments with no package manager. Every dependency is one more thing that has
to already be there, and a missing one is not a degraded tool, it is a tool that
does not run.

The line that decides: **if it is not inside the binary, it does not travel.**
gdoc is one static Go binary. Nothing has to already be on the machine, pandoc
and every other external program included. The line used to be written in pip's
language; it was restated when the tool changed language (changed 2026-08-29,
`docs/v2/DECISIONS.md`).

So:

- A third-party module still needs a stated reason in the spec that adds it.
  Three today, each named in `allowedModules` with its reason, and both
  directions fail: a module that is required and not listed, and a module that
  is listed and not required.
- An external program is the expensive kind, so there are none. Nothing under
  `go/` runs one, which is what lets the login print a sign-in link instead of
  opening a browser.
- git is not a dependency at all. The vault holding the source documents is not
  a git repository and will not become one. Nothing in the binary and nothing in
  the skills runs git, so there is no conditional step and nothing to say out
  loud about one. See the decision dated 2026-08-18 in `docs/v2/DECISIONS.md`.

## 2. The root is where the user stands, and it is never this repo

The source documents live in a vault that is not this repository. A default that
names a repository writes files into the wrong tree, and that failure is quiet.

The restatement, once the root became the place the agent acts from as well as
the place files land: **the hub is the working directory, the memory and the
context, and nothing lives hidden beside it** (changed 2026-08-29,
`docs/v2/DECISIONS.md`).

So:

- The root means one thing for every command: the directory the tool works in,
  `$PWD` by default. No path in the binary and none in the skills names a
  specific repository.
- What gdoc knows about a document lives in the front matter of the markdown
  that describes it, under a single `gdoc:` key that gdoc owns and never writes
  beyond. There is no hidden directory and no queue.
- A comment in a document is answered with the hub open, because the answer
  usually needs what the hub holds and not only what the document says.

## 3. Uncertainty never resolves toward the destructive answer

The tool cannot always tell an edit from a stale copy, or a real instruction from
a stray comment. "I do not know" must never become "overwrite" or "act anyway".

So:

- An existing output file is never overwritten without `--force`. Outside a git
  repository there is no way to tell an edit from a stale copy, so not knowing
  resolves to refusing.
- Every change inside a document somebody handed gdoc is a suggestion, which is
  the guard's write levels holding it rather than a rule somebody remembers. The
  one door in that wall is the grant below.
- Nothing trusts a success. The capability probe runs every time, and every
  write is read back through a route it did not go out on, because four 200s
  lied in one day.
- The marker is the trigger, so an unmarked comment is reported and never
  executed, and a comment the tool cannot confidently classify is reported, not
  acted on.
- A write target with more than one tab stops the command rather than guessing.

*Amended 2026-09-09, for M7b. Nail's decision.* One clause above stops being
true. A handed-in document can now be edited directly, for one run, when the
caller grants it: `gdoc restyle <url> --from survey.json` raises exactly one id
to `LevelInPlace` and puts that document into the house style where it stands.
The principle is why the door is shaped the way it is, not a reason it stayed
shut.

- The grant names one id, is opened by one line in one command, and dies with
  the process. Nothing writes it down, and no flag turns it on for every
  document.
- The level carries four request kinds and no others: `updateDocumentStyle`,
  `updateParagraphStyle`, `updateTextStyle` and `updateTableCellStyle`. None of
  them can change a character of what the author wrote. A kind nobody has heard
  of is refused, which inverts the rule every other level holds, because
  `writeMode` is no longer there to bound it.
- The `fields` mask is bounded too. A mask resets what it names and leaves
  unset, so `*` is refused, an empty mask is refused, and so is a mask naming
  either header toggle.
- The level is a name, not a rung. It does not carry the file `PATCH` that
  `LevelFull` carries, so a restyle cannot trash or rename the document it is
  styling.
- There is no rollback, so a run that stops half way leaves a half-styled
  document and says so. A batch Docs refused on a moved revision is never
  retried: retrying would be gdoc styling a document somebody is editing, which
  is uncertainty resolving toward the destructive answer.

*Amended 2026-09-10, for M7c. Nail's idea.* The clause above stands as it is,
and this milestone deliberately did not widen it. `gdoc restyle --fields` adds
the house cover, the front-matter tables and the legend to that same run, and
every character of them is a **suggestion**, sent on a policy that granted
nothing at all, while the styling goes out on a second one that holds the
grant. Nail accepts them in the browser or rejects them, and a
rejection leaves the document as it was. Not knowing whether the template is
wanted therefore resolves to asking rather than to writing.

- Two phases with two permissions, never one permission that does both. A
  granted document still cannot take an `insertText`, and an ungranted one still
  cannot take a direct edit.
- One request kind was added to the level, and it is `createNamedRange`, bounded
  by a second per-run grant naming the one range it may make. It marks gdoc's
  own prelude so that a second run replaces it rather than adding a second
  cover, and it adds and removes no character, so the sentence above about the
  four still holds.
- The marker is the answer to a second run, and it was measured rather than
  assumed: a named range over a suggested insertion survives the accept and
  vanishes with the reject. A prelude still pending is refused instead, because
  replacing it would propose deleting text that has never been written.

*Amended 2026-09-18, for M11. Nail's decision.* One clause above overstates the
probe. It runs before every proposal, and two writers do not run it at all, the
first of them since M7b. The read-back half of that clause holds for both, with
no exception.

- `gdoc restyle` sends its phase 1 unprobed. The probe needs a folder to create
  its throwaway document in and restyle takes none, and giving it one would be a
  second create door on a command that writes to the one document it was handed.
  `prelude.Verify`'s `Written` count, read back after the write, stands in its
  place. `go/cmd/gdoc/restyle.go` holds that reasoning beside the call.
- `gdoc annotate` runs no probe at all. The batch it sends holds one
  `insertComment` and nothing beside it, and no `insertComment` can move a
  character whatever the write mode does, so the probe has no question to answer
  here. The 2026-09-18 entry in `docs/v2/DECISIONS.md` holds that decision.

## 4. Every word costs a reader's attention

Added 2026-08-29, Nail's principle, wording settled with a second opinion.

gdoc writes in a terminal Nail chose to open and in documents read by people
who did not choose it. Its readers have limited attention. They should not
have to understand gdoc's machinery to understand its work.

gdoc says what happened, why it matters, and what the reader must decide or
do, in terms the reader uses. Calls, indexes, status codes and other machinery
stay out of documents. They appear in the terminal only when Nail asks for
diagnostics or needs them to recover from a failure.

The line that decides: **if the reader must translate it, rewrite it. If the
reader does not need it, remove it.**

## Decisions

The decisions live in [docs/v2/DECISIONS.md](docs/v2/DECISIONS.md), each dated,
each naming the principle it serves, with its status in the register at the top
of that file. This file holds the four principles and nothing else.

Five decisions were written here first and moved there on 2026-09-15, with their
dates intact: the markdown is the source (2026-08-13), skills are symlinked
(2026-08-14), the client reaches only the files it was given (2026-08-15),
nothing runs git (2026-08-18), and the next version of gdoc is written in Go
(2026-08-29). The ones that described the retired Python tool went with it.

## The gate

Every spec and plan under `docs/plans/` carries this block, right after the
title and date:

```
## Principles

Serves: <which principle, and how>

Strains: <which principle, and why that is acceptable>
```

`Strains: none` is a valid answer and will be the common one. The text is not the
point. The point is that a spec cannot be written without opening this file.

Nothing blocks, nothing approves, nothing is automated. This is not governance.

Existing specs and plans are not backfilled. The gate applies going forward.
