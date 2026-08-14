# gdoc-apply first run and version bookkeeping, proposal

Date: 2026-08-14
Status: superseded by [2026-08-14-gdoc-template-merge-design.md](2026-08-14-gdoc-template-merge-design.md),
which takes these findings and answers the three open decisions. The findings
below are kept because they are observations from a live run.
Extends: [2026-08-14-gdoc-storage-and-iteration-design.md](2026-08-14-gdoc-storage-and-iteration-design.md)

Findings from the first live run of `/gdoc-apply` against a source markdown that
had never been generated. Everything below was observed, not reasoned about.

## What was run

A note in the Altery hub, never paired, no `.gdoc/` directory anywhere in the
tree:

```
root: /Users/nailkhusnullin/src/altery/Altery-Platform-Hub   (not a git repo)
md:   11-m2-crypto-project-eagle/202608-miguel-asv-assessment/2026-08-12-miguel-kickoff-call.md
ask:  generate it into Drive folder 1CQE-o9R21qsLj89wEk2kps8RjEUYQAox
```

The document was created: `1pVh8SLKm7fMvBA2wR5vcDCEqhnJgE-h6kDVeJVo6KGw`, in
the right folder, confirmed by its Drive `parentId`. Getting there took one
command the skill does not mention and one command the skill does mention that
fails.

## Finding 1: nothing owns the first version

`gdoc-apply` Step 1 resolves the source in two ways, `Source:` in `pending.md`
and `gdoc pair find --doc-id`. Then:

> If neither answers, Nail does not own this document. The output is a note for
> him, not a new document. Say so and stop.

For a markdown that has never been generated, both lookups correctly return
nothing, and the instruction is to stop. But that state is not an outsider's
document. It is step 1 of the loop this tool exists to serve.

The storage spec says step 2 is "Generate a document from it in Altery style,
using a separate skill." That separate skill knows nothing about pairings, so
whoever creates v1 leaves no `gdoc:` frontmatter behind. Nothing in the system
writes the first pairing.

The gap is not "gdoc-apply should generate v1". It is that **v1 is created
somewhere that does not record it**, and every later step assumes the record
exists.

## Finding 2: Step 5 fails on a first run

Step 5 runs `pair add-version` straight after `generate`. On an unpaired file:

```
$ gdoc pair add-version --md <md> --version-id <new id> --created 2026-08-14
{"error": "... is not paired to a document"}
exit=1
```

`cmd_pair_add_version` returns `_fail` when `read_pairing` is `None`, and
`cmd_generate` never writes a pairing. So the two commands the skill puts next
to each other cannot both succeed the first time.

The workaround is `gdoc pair set --md <md> --doc-id <new id>` in between. That
works, and it is what unblocked the run.

## Finding 3: `pair set` is the wrong workaround from v2 onward

`cmd_pair_set` clears `gdoc_versions` whenever the new `doc_id` differs from the
stored one:

```python
else:
    # Different document (or no prior pairing): old versions describe the
    # wrong document and must not be carried forward.
    versions = ()
```

That reasoning is right for repointing a file at an unrelated document. It is
wrong for the normal case, where the new id is the next version of the same
lineage. Writing "use `pair set` when unpaired" into the skill invites an agent
to reach for it on the second run too, and the version history goes silently.

The first run reported `versions_cleared: 0`, so the trap did not fire. It would
have on v2.

## Finding 4: `gdoc:` goes stale from v2 onward

The storage spec defines the frontmatter as:

```yaml
gdoc: <current version id>
gdoc_versions:
  - id: <v0.1 id>
```

`add_version` appends to `versions` and leaves `doc_id` untouched. Step 5 calls
only `add-version`. So after v2, `gdoc:` still names v1 while `gdoc_versions`
holds both, and the key's stated meaning is false.

Not fatal today: `_matches` checks `gdoc_versions[].id` as well, so a review of
v2 still resolves to the right source file. It is a correctness debt, not an
outage. It matters the moment anything reads `gdoc` as "the current document",
which is what the spec says it is.

## Finding 5: the slug is decided in two places

`slug_for_source` is the CLI's answer, `slugify(md_path.stem)`, and
`_write_baseline_for` uses it. The skill instead tells the agent to pick
`<slug>` itself for `.gdoc/<slug>/out/v<n>.docx`. Following both produced two
sibling directories for one source file:

```
.gdoc/2026-08-12-miguel-kickoff-call/baseline.md   written by the CLI
.gdoc/miguel-kickoff-call/out/v1.docx              chosen by the agent
```

`capture` already has `--slug` documented as "queue directory name, default the
paired source's stem". `generate` is the one command with no slug concept, so
the agent has to reinvent it and can only get it right by accident.

## Finding 6: two things the skill never mentions

**`--folder-id`.** `generate` accepts it and falls back to
`config.output_folder_id`. The skill's command block has neither, so an agent
told "put it in this folder" has no documented way to comply. It has to read the
CLI help to find the flag.

**Where `n` comes from.** Both `--name "<doc name> v<n>"` and
`--out .../v<n>.docx` use `n`, and nothing says how to compute it. It is
derivable as `len(pair show → versions) + 1`, but that is inference, and two
agents will not infer the same thing.

## Proposal

Make `generate` the single place that closes the loop, because it is already the
only place that knows the upload succeeded.

After a successful upload, `cmd_generate` would, in the same block that writes
the baseline:

1. Set `gdoc` to the new document id.
2. Append `{id, created}` to `gdoc_versions`, preserving what is there.
3. Report `slug` and `version` in the JSON alongside `baseline_path`.

Then Step 5 of the skill disappears, `pair set` goes back to being the manual
repointing tool it reads like, first run and iteration become the same command,
and `n` comes from the tool rather than from an agent's arithmetic.

### On "the CLI stays dumb"

CLAUDE.md and the storage spec both say policy belongs in the skill. This does
not breach that. Which folder, which filename, which version label are policy
and stay in the skill. That a document with this id was created from this
markdown on this date is a fact, and the baseline write already establishes that
recording facts at this exact moment is `generate`'s job.

### Guard

Pair writing should be opt-in, not automatic, so a one-off document Nail asked
for does not stamp frontmatter on a source file. `--baseline-root` already marks
"this is a tracked version of a tracked file" and could carry it, or a separate
`--pair` flag could, which is more explicit. Worth deciding rather than assuming.

### Open, needs a decision before building

- Which flag gates the pair write, `--baseline-root` or a new `--pair`.
- Whether `add-version` should also move `doc_id`, or be left alone once
  `generate` owns the bookkeeping. Leaving it alone means it keeps its current
  sharp edge for anyone calling it by hand.
- What `gdoc-apply` Step 1 should say instead of "stop" when the source is
  unpaired. Two candidates: generate v1 and record it, or refuse and name the
  skill that should have.

### Tests to write first

| Test | Asserts |
|---|---|
| `generate` on an unpaired md | `gdoc` set, `gdoc_versions` has one entry |
| `generate` on a paired md with one version | two entries, first one unchanged, `gdoc` moved |
| `generate` when upload fails | frontmatter untouched, no version recorded |
| `generate` without the gating flag | frontmatter untouched |
| slug reported by `generate` | equals `slug_for_source(md)` |

The failure-path test is the one that matters. "Never write a `gdoc_versions`
entry for a document that failed to upload" is currently enforced by the skill
telling the agent to skip Step 5. Moving the write into the CLI moves that
guarantee into code, where it can be tested.

## Not proposed

- Changing `pair set`. Its clearing behaviour is right for what it is. The fix
  is to stop the skill from routing the common case through it.
- A `gdoc slug` subcommand. If `generate` reports the slug, nothing needs to ask
  for it separately.
