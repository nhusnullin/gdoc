# PRINCIPLES.md: what gdoc will not do, and why

Date: 2026-08-15

## Principles

Serves: this spec creates the principles, so it is the one document with
nothing above it. It defines the gate that every later spec passes.

Strains: none.

## The problem

The reasoning behind gdoc's design is written down, but it is scattered across
7,100 lines of specs and plans, plus README.md, plus CLAUDE.md, plus commit
messages. A new session reads none of that. It reads CLAUDE.md, opens the code,
and proposes something.

The failure this causes is concrete. A feature needs a document converter, so a
session reaches for LibreOffice. It does not know that the CLI and the skills
get distributed to other people, and that some of those machines are isolated
environments where a heavy external binary is not present and cannot be
installed. The constraint that rules LibreOffice out is real and settled. It is
simply not written anywhere a session will look.

The same shape repeats. `--repo-root` defaulting to a named repository writes
Altery notes into the wrong tree. A step that resolves "I cannot tell" into
"overwrite" destroys the baseline. Each of these has been decided once and can
be undecided by any session that never saw the reason.

## What this is not

This is not governance. There is no approval step, no review board, no sign-off.
One person works on this repo, with Claude sessions doing the typing. A process
built for a team would cost more than it returns.

It is also not a style guide or a coding standard. Those live in the global rules
already.

It is a short list of decisions that are already settled, each written with the
reason that settled it, in one file a session can read in under a minute.

## Two layers

The document has two layers, and the split is what makes it survive contact with
change.

**Principles** are durable. They are constraints on the world the tool runs in,
not choices about how the tool models its domain. They have a why that does not
expire. There are three, and adding a fourth is a real decision.

**Decisions** are today's implementation. They are dated, replaceable, and each
one names the principle it serves, or states plainly that it is a domain choice
with no principle above it. Retiring a decision is an ordinary edit.

The test that produced this split: "the marker decides" reads like a principle,
but it stops being true as soon as unmarked comments get handled. So it was never
a principle. It is a filter, and filters get replaced. Same for the Commenter-only
credential: it is what enforces the markdown-is-source model today, and Editor
access would be a change to argue about, not a line that can never be crossed.

Sorting a statement into the right layer is the main work this document does.
When a change arrives, the layer tells you what kind of change it is. A decision
you edit. A principle you discuss.

## The three principles

These are written out in full here because the wording is the deliverable. The
implementation copies this text into `PRINCIPLES.md`.

### 1. It runs on someone else's machine

The CLI and the skills get distributed. Some of those machines are isolated
environments with no package manager. Every dependency is one more thing that has
to already be there, and a missing one is not a degraded tool, it is a tool that
does not run.

The line that decides: a dependency pip can install travels with the tool. A
program pip cannot install does not. That is why PyYAML is fine and LibreOffice
is not.

So:

- A new third-party package still needs a stated reason in the spec that adds it.
  Six today.
- An external program is the expensive kind. LibreOffice and poppler were removed
  for this reason, 833MB that cannot be copied into an isolated environment.
  `tests/test_no_external_programs.py` enforces this for `gdoc/render/`, the
  publish path, as an allowlist rather than a ban.
- pandoc is the one that remains, tracked in three places. It must degrade with a
  clear message, never crash, and never be found at a hardcoded absolute path.
- git is a dependency the tool does not have. The vault holding the source
  documents is not a git repository and will not become one. Nothing refuses to
  run for lack of git, and a step skipped because git is absent is always said
  out loud.

### 2. The root is where the user stands, and it is never this repo

The source documents live in a vault that is not this repository. A default that
names a repository writes files into the wrong tree, and that failure is quiet.

So:

- `--repo-root` means one thing for every command: the directory the tool works
  in, `$PWD` by default. No path in the package or in the skills names a specific
  repository.
- Tool-managed files go in `.gdoc/`, beside the source markdown they belong to.
- The queue directory is keyed by the source markdown file. Not the document
  title, not the document id. Both change on every iteration. The file does not.

### 3. Uncertainty never resolves toward the destructive answer

The tool cannot always tell an edit from a stale copy, or a real instruction from
a stray comment. "I do not know" must never become "overwrite" or "act anyway".

So:

- An existing `baseline.md` is never overwritten without `--force`. Outside a git
  repository there is no way to tell an edit from a stale copy, so not knowing
  resolves to refusing.
- A captured item is never silently dropped. It stays in `pending.md` until it is
  applied or removed on purpose.
- A comment the tool cannot confidently classify is reported, not acted on.

## The Decisions section

Same file, below the principles. Each entry is one dated line or short paragraph,
naming the principle it serves.

Initial entries, drawn from what is already true:

- **2026-08-13. The markdown is the source, the Google Doc is a rendering.**
  Domain choice, no principle above it. Document-wide changes are applied to the
  paired markdown and republished as a new version. Replies go in comment
  threads, because a thread is comment surface and not content.
- **2026-08-13. The credential is Commenter-only.** Enforces the decision above
  by permission rather than by discipline. Taking Editor access later would be a
  change to argue about, not a line that cannot be crossed.
- **2026-08-13. Only `ai:`-marked, unresolved, unanswered comments are actioned.**
  The author name is a label, not a gate. Drive returns no email address for
  comment authors and display names are editable, so any identity check is a
  guess the skill would have to disclaim every run. Pointing the skill at a
  document is the trust decision. Handling unmarked comments is planned, and it
  changes this line only.
- **2026-08-14. Skills are symlinked into `~/.claude/skills/`, never copied.**
  A copy drifts silently. `install.sh` defends this and refuses to replace a real
  directory whose contents differ.

Open violations get recorded here too, so they are not lost:

- **Open, 2026-08-15. `gdoc/export.py` hardcodes an absolute path to pandoc.**
  Violates principle 1: the path only exists on Apple Silicon with Homebrew, so
  `export` and `generate` both break on any other machine. This file is being
  changed in a parallel branch with an open PR. Reassess after that merges. No
  code change is made as part of this spec.

Retired decisions stay in place, marked retired, with the date and the reason.
The history is the point. A decision that was reversed once tends to come back.

## The gate

Every spec and plan under `docs/superpowers/` gains a short block near the top,
directly after the title and date:

```
## Principles

Serves: <which principle, and how>

Strains: <which principle, and why that is acceptable>
```

`Strains: none` is a valid answer and will be the common one.

The text is not the point. The point is that a spec cannot be written without
opening `PRINCIPLES.md` first. That is the entire enforcement mechanism. Nothing
blocks, nothing approves, nothing is automated.

Existing specs and plans are not backfilled. The gate applies going forward.

## Pointers, and removing the duplication

`PRINCIPLES.md` sits at the repository root.

`CLAUDE.md` gains one line near the top, before "What lives where":

> Read [PRINCIPLES.md](PRINCIPLES.md) before proposing any design. It is short.

`CLAUDE.md` then loses the reasoning it currently duplicates, and only the
reasoning. In the sections "One root, and it is never this repo" and "The tool
must work without git", the paragraphs explaining why are replaced by a pointer,
because that content moves into principles 1 and 2. The specifics stay, because
`PRINCIPLES.md` does not carry them: `is_dirty` raising `BaselineConflict` only
when git exists and the command fails, `write_baseline` refusing without `force`,
and the commit steps in `gdoc-apply` being conditional. The "Never" list stays
whole, because it is an instruction rather than a reason.

`README.md` gains a link in the same place it introduces the design, and keeps
its explanations. README explains the tool to a person using it. PRINCIPLES.md
explains the constraints to a person changing it. The overlap is small, and where
it exists, PRINCIPLES.md holds the why and README refers to it.

The rule going forward: the why lives in one place. If a reason appears in two
files, one of them is a pointer.

## Length

`PRINCIPLES.md` fits on one screen, roughly 100 lines. This is a constraint, not
an estimate. A principles document that has to be skimmed does not get read, and
the whole value is that a session reads all of it before it starts.

If a fourth principle earns its place, something else has to justify staying.

## Testing

There is no code, so there are no unit tests. Verification is by reading:

1. `PRINCIPLES.md` is under 120 lines.
2. Each of the three principles states its reason before its rules.
3. Every Decisions entry names the principle it serves, or says it is a domain
   choice with none.
4. The pandoc violation is recorded with its parallel-branch context.
5. `CLAUDE.md` and `README.md` link to it, and neither restates a reason that
   `PRINCIPLES.md` now holds.
6. This spec's own `## Principles` block is present, demonstrating the gate.

## Out of scope

- Fixing the hardcoded pandoc path. It is recorded, and it belongs to a branch
  already in flight.
- Backfilling the gate block into existing specs and plans.
- Any change to the skills, the CLI, or the tests.
- Handling unmarked comments. That is a later change to one Decisions line.
