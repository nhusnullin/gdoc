# gdoc storage and iteration, design

Date: 2026-08-14
Status: agreed, not yet built
Supersedes parts of: [2026-08-13-gdoc-ai-agent-design.md](2026-08-13-gdoc-ai-agent-design.md)

Decisions taken on 2026-08-14, after the tool moved into its own repo. Read it
alongside the 2026-08-13 spec, which still describes comment handling correctly
but is wrong about storage, about mirrors, and about git.

## The workflow this must serve

Nail's loop, in his order:

1. Write a markdown file with the content.
2. Generate a document from it in Altery style, using a separate skill.
3. Read and review that document in Google Docs.
4. Leave comments: questions, instructions, improvements. Later, also take
   account of colleagues' comments, not only his own.
5. Run `/gdoc-review`. Expect a reaction in every comment marked for the agent.
6. Reply to the agent's answers in the document, or discuss in the terminal and
   have the agent post. Re-running `/gdoc-review` should answer those replies.
7. Run `/gdoc-apply`. Expect the markdown to change and a new document version
   to be generated for the next round of review.
8. Expect direct edits to the document, and suggestion-mode edits, to reach the
   new version too. Comments are not the only input.

Step 8 is the demanding one. It makes the document an editing surface rather
than a read-only artifact, so content arrives from three places: comment
instructions, direct edits, and suggestions.

## Where files live

### Rooted at the working directory

`--repo-root` defaults to `$PWD`, the directory the skill was launched from. No
path in the tool names any specific repository.

The CLI already defaults `--repo-root` to `"."`. The coupling lives in the two
SKILL.md files, which override it with a hardcoded tool-repo path. Both drop to
a single root variable.

### `.gdoc/` beside the work

Tool-managed files go in `.gdoc/`, next to the source markdown they belong to:

```
<folder holding the source md>/
  2026-08-13-topic.md              the source, hand written
  2026-08-13-topic.docx            generated, distributed
  .gdoc/
    <slug>/
      pending.md                   captured items awaiting apply
      baseline.md                  the document as generated, see below
      out/v<n>.docx                upload intermediate
```

Not `docs/gdoc/`. Once the root can be any directory, `docs/` cannot be assumed
free or appropriate, and a dot-namespace signals tool-owned. It also keeps these
files out of Obsidian's search and graph in vaults where that matters.

Everything under `.gdoc/` is written by the tool and is reproducible or
transient. The source markdown and its pairing frontmatter are the only things
Nail authors, and they stay in the folder's normal structure.

### No git requirement

`Altery-Platform-Hub`, where the source documents live, is not a git repository
and will not become one. The tool must work without git.

Consequences:

- Nothing may refuse to run because git is unavailable.
- `gdoc-apply`'s commit step becomes conditional: commit when inside a repo,
  otherwise skip it and say so plainly.
- `is_dirty` currently raises `MirrorConflict` when git cannot be consulted.
  Without git there is no way to know whether a file holds unsaved work, so the
  safe fallback is to refuse to overwrite an existing file unless `--force` is
  passed. Writes meant to overwrite, such as the baseline after a successful
  generate, pass `--force` themselves.

The 2026-08-13 spec assumed git throughout. That assumption is now wrong.

## The mirror becomes a baseline

`mirror.md` was written by `gdoc-review` at the **end** of a review. That is the
wrong moment: by then the document may already carry Nail's direct edits, so the
snapshot bakes them in and they become invisible to any later comparison.

Step 8 needs one question answered: what changed in the document that is not in
my markdown? That cannot come from comparing the document against the source.
The source goes through pandoc to `.docx`, Drive converts to a native Doc, and
export converts back to markdown. The round trip rewrites heading styles, list
markers and spacing, so the diff would be mostly noise.

It comes from diffing **two exports of the same document**. Both carry the same
round-trip distortion, so it cancels.

The file is therefore kept, renamed and rescheduled:

- Renamed `baseline.md`, because it exists to be compared against.
- Written by `generate`, immediately after a successful upload, when document
  and markdown provably match.
- Never written by `gdoc-review`.

At the next apply:

```
direct edits = diff(baseline.md, export of the document now)
```

### `export` fetches, it does not file

`export` today downloads, derives a slug from the title, checks collisions,
refuses on uncommitted edits, and writes to a path the caller never chose. It
becomes a fetch:

```
gdoc export <url>              markdown to stdout
gdoc export <url> --out FILE   written to FILE
```

Nothing is written unless asked. This makes on-demand drift checking possible
for the first time:

```
diff <(gdoc export <url>) path/to/source.md
```

## Identity: key by document, not by title

`slug = slugify(document title)`, so the directory name tracks a mutable
attribute. Renaming a document in Drive, or bumping a version in its title,
creates a fresh directory and orphans the existing queue, with no warning.

Not an edge case. `/gdoc-apply` generates a new version, and if that changes the
title the next review forks a new queue. It has already happened: `ver-0-1` and
`ver-0-2` are two queues for one document lineage. The dedup guard in
`pending.py` is per file, so a comment carried across versions is captured again
as new.

Fix: keep readable directory names from the title, but look up by document id
first. `find_slug_for_doc(root, doc_id)` scans `.gdoc/*/` for a recorded id.
Found means reuse that directory whatever the title now says. Not found means
create one from the title.

`_check_slug_collision` guards the opposite direction, two documents mapping to
one slug. With directories keyed by id it becomes unnecessary.

## Where generated documents go

`generate` converts markdown to `.docx`, then Drive turns that into a native
Doc. The `.docx` has two roles, and `generate.py` keeps the file precisely so
the failure path has something to hand back:

| Case | What the `.docx` is |
|---|---|
| upload succeeds | throwaway intermediate; the real output is a Drive link |
| no `output_folder_id`, or upload refused | the deliverable, and its path is reported |

One fixed location cannot serve both. `--out` is chosen by intent:

- new Drive version: `.gdoc/<slug>/out/v<n>.docx`, out of sight
- a document Nail asked for: `$PWD`, visible and ready to send
- an explicit destination: that path

The CLI stays dumb; the skill carries the policy. No staging-then-moving step,
which can half-fail and buys nothing over choosing the right `--out`.

PDF is nearly free, since `md_to_docx` runs `pandoc md -o out_path` and pandoc
picks the format from the extension. It needs a PDF engine installed, so that
must be checked before it is offered.

## Merging three inputs

**Direct edits, and edits to the markdown: merge on best effort, ask when it
cannot.** Apply automatically. Stop only when the same passage moved in both
places in a way the tool cannot reconcile, or when a change looks like
round-trip noise rather than intent: punctuation-only, whitespace-only.

**Suggestions: more than a proposal, and arguable.** Four possible responses:

| Response | When |
|---|---|
| accept and merge | it is right, or harmless |
| push back with evidence | it contradicts the corpus; cite plain file paths |
| rephrase | the point is right, the wording is not |
| flag a knock-on effect | accepting it makes another section wrong |

The last needs the whole document in view, not just the suggested paragraph.
This reuses `gdoc-review` Step 4, which already grounds answers in the corpus
and cites paths.

The distinction driving the behaviour: Nail's direct edits are decisions already
taken, so the default is apply. A colleague's suggestion is a claim, so the
default is evaluate.

## Open, and blocking Part B

Whether suggestions can be read at all. The tool uses only the Drive API
(`files().export`). Whether that renders pending suggestions as accepted,
rejected or inline is unverified.

The likely route is the Docs API `documents.get` with its `suggestionsViewMode`
parameter. Two things need confirming before any of step 8 is designed: whether
a Commenter-role service account can read suggestions, and whether that needs an
API scope the credential does not hold.

If suggestions cannot be read, that half of step 8 has no build and needs a
different approach.

## Deliberately not doing

- `gdoc pending list`. `.gdoc/` is a plain directory; globbing and `ls` already
  answer the question. A subcommand to list files in a folder is speculative.
- A central index of documents. `pairing.py` already rejects this, and its
  reasoning holds: an index would live outside every repo and go stale silently
  when a file moves.

## Known gap, not addressed here

The dedup guard lives in two places that never agree. `pending.md` dedups by
comment id, but `/gdoc-apply` deletes each item as it applies it, so a later
review can capture the same comment again as new. In the other direction,
deleting `pending.md` does not let a re-run recover the items, because each
already carries a service-account reply and `gdoc read` puts answered threads
under `skipped`.

Worth fixing. Out of scope for Part A.
