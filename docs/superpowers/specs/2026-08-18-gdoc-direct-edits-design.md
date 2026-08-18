# Direct edits in the document, carried back into the markdown

2026-08-18

## Principles

Serves: principle 3, uncertainty never resolves toward the destructive answer.
The document already carries Nail's edits, and today's `generate` writes over
them without a word. Reading them first turns a silent loss into either an
applied change or a question. It also serves principle 1: the Docs API is a new
API surface on a package that is already a dependency, not a new program.

Strains: principle 1, slightly. One more OAuth scope means one re-login for
anyone already installed, and the run that needs it fails until they do. The
failure names the fix and no other command is affected, which is why it is
acceptable.

## The problem

`baseline.md` is written at the one moment the document and the markdown provably
match. Nothing reads it. So an edit Nail makes in the browser, in editing mode or
in suggesting mode, is lost the next time `generate` runs, and nothing says so.

`gdoc-review` deliberately writes no snapshot for this reason. The mechanism was
built and the step that uses it was never written.

## Decided with Nail, 2026-08-18 (issue #29)

1. It runs in `gdoc-apply`, before the pending items. Apply is the only place
   that edits markdown and republishes.
2. Suggesting mode is in scope. It is how Nail actually reviews, and Drive's
   markdown export hides pending suggestions completely, so leaving it out would
   ship the silent failure this work exists to remove.
3. The agent judges and asks only when in doubt. Twenty small wording fixes are
   one summary, not twenty questions.
4. Moves and unlocatable hunks follow the same rule: work it out, ask when unsure.

## Shape

A new command, `gdoc edits`, that reads and reports. It writes nothing to the
markdown: porting a change into hand-written prose is judgment, and judgment is
the skill's job, not the CLI's.

```
gdoc edits <doc url> [--baseline-root .] [--source notes.md]
```

It returns JSON:

```json
{
  "doc_id": "...",
  "slug": "...",
  "source": "notes.md",
  "baseline_path": ".gdoc/<slug>/baseline.md",
  "baseline": "ok",
  "hunks": [
    {"kind": "change", "heading": "3. Routes", "before": "...", "after": "..."}
  ],
  "suggestions": [
    {"kind": "insertion", "heading": "3. Routes", "text": "...", "author": "Nail"}
  ],
  "counts": {"hunks": 1, "suggestions": 1}
}
```

### Two sources, because one of them lies

- **The export**, diffed against `baseline.md`. Both sides carry the same export
  distortion, so it cancels. This sees edits made in editing mode.
- **The Docs API**, `documents.get` with `suggestionsViewMode=SUGGESTIONS_INLINE`.
  The export renders a suggested document as if no suggestion existed, so without
  this a document reviewed entirely in suggesting mode diffs to nothing at all.

They are reported separately, because they mean different things. An edit is
settled. A suggestion is Nail thinking out loud, and it is still pending in the
document after the markdown is changed.

### The baseline has to be the right one

Refuse rather than diff against a snapshot that does not describe this document.
`generate` now writes `.gdoc/<slug>/baseline.json` beside the baseline, recording
the document id it was taken from. Every version is a new document, so a mismatch
is exactly the case where the baseline is stale.

- No `baseline.md`: `"baseline": "missing"`, no hunks, and the suggestions are
  still reported, because they do not need one.
- `baseline.json` names another document: `"baseline": "stale"`, same treatment.
- No `baseline.json` at all, from a version published before this change:
  `"baseline": "unverified"`. The hunks are reported and labelled, because a
  baseline that is probably right is worth reading when the alternative is
  losing the edits silently.

The command never fails on any of these. It reports which case it is, and the
pending items go ahead either way.

### Scopes

`documents.readonly`, and only where it is needed. `drive_service` keeps the
Drive scope alone, so every existing command works on the token Nail already has.
`docs_service` asks for the Docs scope, so `gdoc edits` is the one command that
fails until he runs `gdoc auth login` once. `auth login` requests both.

Read only, so the Commenter-era decision survives in spirit: gdoc reads a
suggestion as an intention and applies it to the markdown. It never accepts one
in the document.

### The guard already covers it

`gdoc/guard.py` matches `/v1/documents/{id}` on `docs.googleapis.com`, and the
check sits in the transport, so the Docs client is confined the same way the
Drive client is. `gdoc/auth.py` stays the only module that calls `build()`.

## What the skill does with it

A new step in `gdoc-apply`, between finding the queue and working the items:

1. Run `gdoc edits`. Say which baseline case came back.
2. Port each hunk into the paired source markdown, by heading and surrounding
   text rather than by line number, because the source is hand-written and not
   export-shaped.
3. Apply what is clear. Ask about what is not: a hunk that cannot be located, a
   delete-plus-insert pair that may be one moved section, a change that collides
   with a queued item.
4. Report what was applied, in one summary, and say plainly that suggestions are
   still pending in the document and gdoc did not accept them.

## Not in this change

- Accepting or rejecting a suggestion in the document. gdoc does not edit
  documents, and the credential is read only here.
- Images and drawings, which is issue #28.
- Anything about who suggested what beyond the name the API returns.
