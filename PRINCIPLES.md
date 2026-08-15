# Principles

Read this before proposing any design. It is short on purpose.

Two layers. **Principles** are durable constraints on the world the tool runs in.
They have a reason that does not expire, and there are three. **Decisions** are
today's implementation. They are dated, they name the principle they serve, and
retiring one is an ordinary edit.

When a change arrives, the layer tells you what kind of change it is. A decision
you edit. A principle you argue about first.

## 1. It runs on someone else's machine

The CLI and the skills get distributed. Some of those machines are isolated
environments with no package manager. Every dependency is one more thing that has
to already be there, and a missing one is not a degraded tool, it is a tool that
does not run.

The line that decides: **a dependency pip can install travels with the tool. A
program pip cannot install does not.** That is why PyYAML is fine and LibreOffice
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

## 2. The root is where the user stands, and it is never this repo

The source documents live in a vault that is not this repository. A default that
names a repository writes files into the wrong tree, and that failure is quiet.

So:

- `--repo-root` means one thing for every command: the directory the tool works
  in, `$PWD` by default. No path in the package or in the skills names a specific
  repository.
- Tool-managed files go in `.gdoc/`, beside the source markdown they belong to.
- The queue directory is keyed by the source markdown file. Not the document
  title, not the document id. Both change on every iteration. The file does not.

## 3. Uncertainty never resolves toward the destructive answer

The tool cannot always tell an edit from a stale copy, or a real instruction from
a stray comment. "I do not know" must never become "overwrite" or "act anyway".

So:

- An existing `baseline.md` is never overwritten without `--force`. Outside a git
  repository there is no way to tell an edit from a stale copy, so not knowing
  resolves to refusing.
- A captured item is never silently dropped. It stays in `pending.md` until it is
  applied or removed on purpose.
- A comment the tool cannot confidently classify is reported, not acted on.

## Decisions

Dated, replaceable. Each names the principle it serves, or says it is a domain
choice with none above it. Retired entries stay, marked retired, with the reason.

**2026-08-13. The markdown is the source, the Google Doc is a rendering.**
Domain choice, no principle above it. Document-wide changes are applied to the
paired markdown and republished as a new version. Replies go in comment threads,
because a thread is comment surface and not content.

**2026-08-13. The credential is Commenter-only.** *Retired 2026-08-15.* It
enforced the decision above by permission rather than by discipline, and it was
Google's to enforce. OAuth has no scope that reads comments and writes replies
without full Drive access, so keeping this would have meant keeping the
per-document sharing step forever. Replaced by the decision below.

**2026-08-15. The client reaches only the files it was given.** Serves principle
3. Under OAuth the credential can reach every file the signed-in user owns, so
`gdoc/guard.py` holds a set of file ids, the ones the command was handed plus
the ones its own creates returned, and refuses every request addressing anything
else. `files.list` is refused outright, so the tool cannot search Drive. An
empty set refuses everything, so a command that does not name a document reaches
nothing.

The cost is stated rather than hidden: on a file in the set, every method is
permitted, including an edit to a reviewed document. "The agent cannot edit a
reviewed document" becomes "it does not". Nothing in gdoc edits one, and the
markdown-is-the-source decision above is now held up by design rather than by
permission. `tests/test_guard.py` and `tests/test_guard_is_installed.py` are
what keep it honest.

**2026-08-13. Only `ai:`-marked, unresolved, unanswered comments are actioned.**
Domain choice, no principle above it. The author name is a label, not a gate. Drive returns no email address for
comment authors and display names are editable, so any identity check is a guess
the skill would have to disclaim every run. Pointing the skill at a document is
the trust decision. Handling unmarked comments is planned, and it changes this
line only.

**2026-08-14. Skills are symlinked into `~/.claude/skills/`, never copied.**
Domain choice, no principle above it. A copy drifts silently. `install.sh` defends this and refuses to replace a real
directory whose contents differ.

### Open violations

**2026-08-15. `gdoc/export.py` hardcodes an absolute path to pandoc.**
*Closed 2026-08-15.* The parallel branch merged. `find_pandoc` now resolves
pandoc at call time, and the Homebrew path is a last-resort fallback rather than
the answer. `tests/test_pandoc_path.py` covers it.

None open.

## The gate

Every spec and plan under `docs/superpowers/` carries this block, right after the
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
