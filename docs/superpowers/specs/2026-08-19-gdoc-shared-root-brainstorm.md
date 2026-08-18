# A shared root: whose state is `.gdoc/`?

2026-08-19. Brainstorm for issue #25. Nothing here is built.

## Principles

Serves: principle 3. Two people writing one queue, or one publish overwriting
another's baseline, is "I do not know whose this is" resolving to "overwrite".

Strains: principle 2, mildly. Anything that keys state by identity puts a name in
a path under the root, and the root is meant to be the user's own tree with as
little tool furniture in it as possible.

## The failure, stated once

`.gdoc/` sits beside the markdown, inside a vault that syncs. So it syncs.

- **`pending.md`.** Two reviewers append. `next_item_number` reads the file and
  adds one, so two captures pick the same number, and the sync client resolves it
  by leaving `pending (conflicted copy).md`, which nothing reads. One person on a
  laptop and a desktop hits the same race alone.
- **`baseline.md`.** `generate` replaces it with `force`. Somebody else's publish
  therefore destroys the snapshot your next apply depends on, and since tonight
  there is no git check pretending otherwise (#24). `baseline.json` (#29) can now
  *detect* this, because it records which document the snapshot came from, but
  detecting is not recovering.
- **`out/*.docx`.** Two runs write the same paths. Harmless, they are
  intermediates, but they are noise in a conflict list.

## What is not the answer

**A lock file.** A lock that has not synced yet is not a lock. Two machines can
both take it before either sees the other, and a stale one cannot be told from a
live one. It also does nothing for the case where two people genuinely both have
items.

**Detecting a sync folder and warning.** It misses custom locations, it reassures
people it should not, and it prevents nothing. A warning is not a guard.

**Declaring a synced root unsupported.** A shared vault is where these documents
live. Writing that down would be honest about today and wrong about the tool.

## The shape to build

### 1. The queue is per identity

`.gdoc/<slug>/queues/<identity>/pending.md`.

Two people no longer share a list, which is the failure that loses work. Reading
someone else's queue becomes a deliberate act with a path in it, which is right:
their queue is their intention, not yours.

### 2. The identity is chosen, not derived

Put it in the machine-local config, `~/.config/gdoc-agent/config.json`, as
`state_identity`. Suggest it during setup from the signed-in account, and never
take it from the credential at run time.

Four reasons, and each one is a real case:

- Rotating a token must not move a queue.
- Switching between `oauth` and `service_account` must not move a queue. That is
  one person, one intention, two credentials.
- Two people sharing one service account key must not share a queue. Deriving
  from the credential gives them the same one, which is the bug.
- Two machines belonging to one person must share a queue, and only a chosen name
  can say so.

If it is unset and a command would write into `.gdoc/`, refuse and print the
command that sets it. Guessing here picks somebody's queue at random, and
`auth_mode` already establishes that inference is for silence, never for a
question whose wrong answer is destructive.

### 3. The baseline is shared, and keyed by document

A baseline is a fact about a document, not about a person. Copying it per
identity would make several supposedly authoritative snapshots of one document,
which is worse than sharing one.

But one mutable `baseline.md` is also wrong, because `generate` replaces it and
somebody else's publish is indistinguishable from your own. Key it by the
document it describes and never replace it:

```
.gdoc/<slug>/baselines/<doc-id>/baseline.md
.gdoc/<slug>/baselines/<doc-id>/baseline.json
```

Every version is a new document, so every publish writes a new directory and
nothing is ever overwritten. `gdoc edits` already asks "is this baseline this
document's?", and under this layout the question answers itself by the path.
Same content on a rerun is success. Different content under the same id is a
refusal, not a merge.

Cost: the directory grows one snapshot per version. A snapshot is a few tens of
kilobytes and a document gets a handful of versions, so this is not a problem
worth engineering around. If it ever is, deleting old ones is a chore for a
`gdoc prune`, not a reason to overwrite.

### 4. Item numbers stop being identity

`next_item_number` reading the file and adding one is the collision. Numbers are
for talking to Nail ("item 2"), so keep them for display and give each item a
stable id of its own, the comment id it came from, which Drive already made
unique. Two captures then cannot collide even in the same file.

That alone fixes most of the one-person-two-machines case, and it is small.

## What Codex argued for, and why I would not start there

Asked the same question, Codex (gpt-5.6-sol) went further: make the queue an
append-only log of immutable events with tombstones for applied items, checksum
every baseline, refuse on any mismatch, and ship an explicit
`gdoc state migrate` command.

That is the correct answer for a system where writers genuinely cannot see each
other. It is heavier than this tool has earned. gdoc is used by one person and
occasionally a colleague, the queue is read by a human every time before anything
happens, and an event log turns "open pending.md and read it" into "run a tool to
render your own queue", which is a real cost in a vault people browse in Obsidian.

Two of its points survive that filter completely, and they are in the shape above:
the identity must not come from the credential, and the baseline must be
immutable and keyed by document.

Its strongest point, which no design here fixes: writes to the **source markdown**
race too. `write_pairing` is read-modify-replace, so two publishes of the same
note can lose a `gdoc_versions` entry. That is outside `.gdoc/` and outside this
issue, and it should be its own issue rather than be smuggled into this one.

## The old layout

Refuse and ask. A `.gdoc/<slug>/pending.md` in the current shape belongs to
somebody, and the tool cannot tell whom: under a shared service account key it
may be two people's items in one file. Assigning it silently to whoever runs the
upgrade is the destructive resolution of not knowing.

So: on finding the old layout, stop, say what was found, and print one command
that moves it into a named identity. Keep the original files until the person
deletes them.

## What has to hold for any of this to survive a sync client

State the limit rather than pretend to close it.

- Every write stays atomic through a temp file and `os.replace`, as
  `write_baseline` already does. That survives a crash. It does not survive a sync
  client rewriting the file a second later.
- Anything malformed, conflicted or half-arrived is refused, never merged.
- `(conflicted copy)` files are recognised by name and reported, because today
  they are invisible and they are the sync client's own report of a lost write.
- Two people editing the same source markdown at the same moment cannot be made
  safe by a local CLI. Say so in the README, once, plainly.

## Open, for Nail

1. `state_identity`: your call on the default. Refuse until set, or fall back to
   the account name and say so every run?
2. Baseline per document id: the layout change is the largest part of this, and
   it moves a path that `gdoc-apply` prints. Worth doing in the same change as
   the queue move, or separately?
3. Do you want `gdoc state migrate`, or is "move the folder yourself, here is
   where" enough for a tool with two users?
4. The source markdown race is a separate issue. Shall I file it?
