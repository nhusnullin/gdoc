# Principles

Read this before proposing any design. It is short on purpose.

Two layers. **Principles** are durable constraints on the world the tool runs in.
They have a reason that does not expire, and there are four, the fourth
added 2026-08-29. **Decisions** are
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
  Two of the three go away by changing language rather than by writing a parser;
  see the decision dated 2026-08-29 below.
- git is not a dependency at all. The vault holding the source documents is not
  a git repository and will not become one. Nothing in the tool or the skills
  runs git, so there is no conditional step and nothing to say out loud about
  one. See the decision dated 2026-08-18 below.

*Amended 2026-08-29, for v2.* The reason holds and v2 makes it stronger, but
the line above is written in pip's language and v2 has no pip: it is one static
Go binary with no external programs at all, pandoc included. The line that
decides becomes: **if it is not inside the binary, it does not travel.** The
bullets above describe v1 and retire with it.

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

*Amended 2026-08-29, for v2.* The principle stands; two of the three
consequences retire. v2 has no `.gdoc/` and no queue: what gdoc knows lives in
the front matter of the markdown it describes, under a single `gdoc:` key that
gdoc owns and never beyond it. And the root gains a second job the old wording
never mentions: it is not only where files land, it is the context the agent
acts from, so a comment in a document is answered with the hub open. The
restatement: **the hub is the working directory, the memory and the context,
and nothing lives hidden beside it.**

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

*Amended 2026-08-29, for v2.* The text stands word for word. It is the one
principle v2 needed more of. The examples above are v1's: `baseline.md` and
`pending.md` do not exist in v2, so they retire with it. The v2 instances: the
guard's write levels, under which a handed-in document can never be
direct-edited; the capability probe and the read-back after every proposal,
because four 200s lied in one day; the marker is the trigger, so an unmarked
comment is reported and never executed; and a write target with more than one
tab stops the command rather than guessing. A comment the tool cannot
confidently classify is still reported, not acted on.

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

Dated, replaceable. Each names the principle it serves, or says it is a domain
choice with none above it. Retired entries stay, marked retired, with the reason.

**2026-08-13. The markdown is the source, the Google Doc is a rendering.**
Domain choice, no principle above it. Document-wide changes are applied to the
paired markdown and republished as a new version. Replies go in comment threads,
because a thread is comment surface and not content.

**2026-08-13. The credential is Commenter-only.** *Retired 2026-08-15.* It
enforced the decision above by permission rather than by discipline, and it was
Google's to enforce. Under OAuth the usable scope is full Drive, so keeping this
would have meant keeping the per-document sharing step forever. Replaced by the
decision below.

*Corrected 2026-08-18.* This entry used to say Drive has no scope that reads
comments and writes replies without full Drive access. Not true: both accept
`drive.file`, which is non-sensitive. But a file enters `drive.file` scope only
when the app created it or the user handed it over through Google's file picker,
and a terminal has no picker, so pasting a URL still means full Drive. The
conclusion holds; the reason was wrong.

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

**2026-08-18. An unstated `auth_mode` is inferred, never defaulted.** Serves
principle 1, that the tool keeps working. A config written before `auth_mode`
existed cannot mention it, so reading a missing key as oauth would flip every
working service_account install the moment it upgraded, and every command would
then fail on a token that was never created. `Config.auth_mode` is `None` when
the file does not say, and `gdoc.auth.resolve_auth_mode` decides: a stated mode
wins outright, a token means oauth, a key with no token means service_account,
and neither means oauth, which is the new install.

Inference is for silence only. A stated mode that is not one of the two is
refused, and a config that cannot be read reaches the caller, because resolving
either quietly would pick oauth on a machine holding a token, which is the wider
credential and not the one the author asked for.

The consequence is that a credential is chosen partly by which files exist, so
`gdoc auth status` reports whether the mode was stated or inferred, and reports
`null` rather than a guess when the config is broken. `gdoc auth
login` and `gdoc auth use` write the mode down, which both makes a login take
effect and settles the question for good on that machine. No setup step is a hand
edit of the config file.

**2026-08-18. Nothing runs git, and the agent never commits.** Serves principle
1. Committing is Nail's job. A person receiving this tool should not have to
think about git at all, and the skills should not carry a conditional and a
"nothing was committed" sentence for a case that may never apply to them.

Two halves, and they are separate decisions that happen to land together.

The skills no longer commit. `gdoc-apply` used to run `git rev-parse` before
three commits: the edits, the front matter the publish wrote back, and the
cleared queue. All three are gone, replaced by naming every file that changed on
disk so Nail can commit them himself. Trying the commit and reporting what git
said was considered and rejected: that still makes committing the agent's job.

`gdoc/baseline.py` no longer consults git either. It used to ask whether
`baseline.md` held uncommitted work and overwrite it when git said no. Outside a
repository the answer was always "cannot tell", which was most of the time. And
where the root is a synced folder, a Dropbox or Nextcloud rewrite is exactly
what git cannot see, so the check read as safety while providing none. Now an
existing baseline is never overwritten without `force`, in any directory, which
is principle 3 held by disk state rather than by a subprocess. `generate` passes
`force` at the one moment the document and the markdown provably match, and it
is the only caller that writes one.

The cost is that a document generated from an uncommitted note is no longer
matched to a commit. That was only ever true on Nail's own machine, and it was
never checked.

**2026-08-13. Only `ai:`-marked, unresolved, unanswered comments are actioned.**
Domain choice, no principle above it. The author name is a label, not a gate. Drive returns no email address for
comment authors and display names are editable, so any identity check is a guess
the skill would have to disclaim every run. Pointing the skill at a document is
the trust decision.

*Amended 2026-08-15.* Two things. Under `oauth` this stops being a preference:
Drive reports the credential as the author of everything the signed-in user
writes, so an author check would skip every marked comment Nail leaves himself.
And `gdoc read --all` now exists, so unmarked comments are actionable when Nail
asks for them and picks each one. The default is unchanged.

**2026-08-14. Skills are symlinked into `~/.claude/skills/`, never copied.**
Domain choice, no principle above it. A copy drifts silently. `install.sh` defends this and refuses to replace a real
directory whose contents differ.

**2026-08-29. The next version of gdoc is written in Go.** Serves principle 1.

Principle 1 draws its line at "a program pip cannot install does not travel".
pandoc is that program. It has survived three specs because removing it in Python
means writing a Markdown parser and rewriting the AST walker in the module where
the body's pixel fidelity lives. In Go it is an import. A Go build is one static
binary: no interpreter, no venv, no pandoc, nothing that has to already be on the
machine. That is principle 1 satisfied rather than managed, and it is the whole
reason for the decision. Speed is not.

The fidelity question was measured before deciding, because it was the assumed
risk. Six documents were built by both renderers, published to Drive by both,
exported as PDF by Google, rasterised at 300 dpi and compared with a zero
tolerance: 418,385,088 pixels across 48 pages, none different. Run twice on
separate publishes. The spike is
`docs/superpowers/specs/2026-08-29-go-render-spike.md`, and the code is on branch
`spike/go-render`.

It ports exactly for a structural reason worth keeping in mind. Neither renderer
builds a .docx. Both copy the master and cut into it, so the cover, the logo, the
running head and the coloured tables travel as bytes. A .docx is a zip of XML, and
the body was already written as OOXML by hand against constants measured out of
the template. None of that was ever Python.

**What the decision does not rest on.** `gdoc/export.py` uses pandoc as a docx
*reader*, and that use is load-bearing rather than a fallback: it is the only
route that brings a picture out of a Google Doc. goldmark does not replace it, so
that reader has to be written. It is unmeasured. The estimate is that it is the
size of the body renderer, because it only has to read Google's own docx export
rather than arbitrary Word files, and because it is the same `document.xml` shape
the renderer already writes. An estimate is not a measurement, and proving this is
the first thing the port should do.

So "pandoc in three places" becomes "one docx reader to write", and that reader is
a job this repository has in either language.

**Scope.** Next version. The Python tool stays until the Go one covers what it
covers, and nothing is deleted on the strength of this entry. Roughly a third of
the package is ported; the rest, and a 652-test suite, is not.

**One thing the port found that belongs in Python either way.** Page numbers come
from Google's PDF export. The Python renderer reads them by extracting the PDF's
text and matching it line by line, which is guesswork over glyph coordinates: it
can match a heading against its own contents entry and report the contents page as
the heading's page. Google's export carries a document outline, one entry per
heading pointing at the page it sits on. Reading that is exact and has no failure
of this kind. Adopt it in Python whatever happens to the port.

Porting also found four defects in `gdoc/render/`, each of which loses content in
silence. They are fixed already, so that work survives even if the port stalls.
See the commit "fix: four things the renderer was dropping in silence".

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
