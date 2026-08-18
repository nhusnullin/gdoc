# A document that is not in the hub: restyle it, or adopt it

2026-08-19. Brainstorm for issue #9. Nothing here is built.

## Principles

Serves: principle 2, the root is where the user stands. A document nobody in the
hub owns the source of has no queue directory and no paired markdown, and the
tool currently stops rather than guessing where to put one.

Strains: principle 3, in one place, and that place is the whole decision below.
Publishing into the folder that holds the original means reaching a folder Nail
never named, derived from a document he did name. `gdoc/guard.py` has two doors
today and CLAUDE.md says never add a third.

## What the issue asks for

Two different jobs are written in one issue, and separating them is most of the
work.

**Restyle.** Point `/gdoc-apply` at a Google Doc URL. Get a new document in the
house style, in the same Drive folder as the original, with nothing else changed.
The comments come across "keeping original authors if possible, and if not
possible then skip copying comments".

**Adopt.** The same document, but the point is to work on it: pull it into
markdown, that markdown becomes the source of truth, and from then on it is an
ordinary hub document with the ordinary review loop. Nail calls this "the
opposite process", and it is: today the markdown makes the document, here the
document makes the markdown.

They share exactly one step, the pull, and after that they diverge completely.
Restyle throws the markdown away. Adopt keeps it and makes it authoritative.

## What is already true tonight

- `gdoc export --out note.md --media-dir note-media` pulls a document into
  markdown with its pictures, and says which pictures it could not carry (#28).
  Before tonight the diagrams were lost, which would have made restyle produce a
  document that had lost the part that mattered most.
- `gdoc generate` already writes the first `gdoc:` and `gdoc_versions` into a
  note that has never been published, so adopt needs no new pairing code.
- `gdoc edits` reads what changed in a document against its baseline (#29). It is
  no use here: a foreign document has no baseline, and `baseline_state` returns
  `missing` for exactly this case.

So restyle is: pull, add front matter, generate. Adopt is: pull, add front matter,
ask where the markdown lives, then hand over to the normal loop.

## The decision that has to come first: which folder

Nail asked for the original's folder. That means `files.get(fileId, fields=parents)`
on the document, then creating in the parent it names.

The guard permits the read, because the document is in its set, and it permits the
create, because a create names no file. So this works today with no change to
`gdoc/guard.py`. That is the problem: it works by accident rather than by decision.

Three ways to resolve it, and this is Nail's call, not the implementer's.

1. **A third door, written down.** The set may grow by the parent of a file
   already in it, learned from that file's own metadata. Narrow, derived, and
   auditable: no folder is reachable unless a document Nail named lives in it.
   Cost: CLAUDE.md says the set has exactly two doors and must never gain a
   third, so this is an edit to a stated rule, and the rule exists because
   widening a reachable set is the kind of change that looks harmless every time.
2. **Ask every time.** The tool reads the parent, tells Nail which folder it is
   ("this document lives in Shared drives/Legal/Reviews. Publish the restyled
   copy there?"), and creates only after he confirms or names another. One extra
   turn per restyle, and the reach stays exactly where it is.
3. **Never derive it.** Always require `--folder-id`. Simplest, and it throws
   away the thing that made the request convenient.

Recommendation: **2**. It gives Nail what he asked for, the answer is on screen
rather than guessed, and it leaves the guard's rule intact. The confirmation is
one line and it is the same line that protects against publishing a restyled copy
of someone else's document into someone else's folder, which is the failure worth
being slow about. If it turns out to be tiresome in practice, 1 is a small edit
away, and by then there will be evidence for it.

## Comments: the honest answer is "no", and here is why

Nail wrote "keeping original authors (if possible, but not necessarily)". It is
not possible.

- Drive creates every comment as the authenticated user. There is no field for
  writing one as somebody else, and no scope that grants it. Under `oauth` every
  copied comment would arrive as Nail, on a document his colleagues will read.
  A comment that says "William Mejia" but was written by Nail's credential is
  worse than a missing comment.
- Anchoring is the second wall. A comment is useful because it points at a span
  of text. Anchoring a new comment needs an anchor computed against the new
  document's own structure, and the restyled document has different structure by
  construction: it is the house template. Unanchored copies arrive as a pile of
  remarks at the top of the document, in no order, attached to nothing.

So the issue's own fallback is the answer: **do not copy comments.** Say why, in
one sentence, and give the link to the original so the thread is one click away.

Worth offering, not by default: a **comment appendix**. An "Original comments"
section at the end of the restyled document, each comment as text with its author
name and the sentence it quoted. It is honest, it is anchored by quotation rather
than by API, and it survives being read by someone who cannot open the original.
Offer it, do not do it silently.

## Where the markdown goes

Restyle needs a markdown file only as an intermediate, so it goes in a temporary
directory and is deleted. Nothing in the hub, no pairing, no queue.

Adopt needs it to live somewhere, and only Nail knows where. The rule that fits
principle 2: propose, never assume.

```
This document is not paired to anything under /Users/nail/hub.
Adopting it will write:
  ./2026-08-19-mc-incentive-routes.md      (from the document)
  ./2026-08-19-mc-incentive-routes-media/  (3 pictures)
Adopt it here? Or give me a path.
```

The name comes from the document title, slugified, with today's date in front,
because that is the convention every other note in the hub follows.

## What adopt has to record that publish does not

An adopted note came from somewhere, and six months later that matters.

```yaml
gdoc: <id of the new document, as usual>
gdoc_versions: [...]
gdoc_adopted_from: <id of the original document>
gdoc_adopted_on: 2026-08-19
```

`gdoc_adopted_from` is not a pairing and must never be treated as one: the tool
must not publish over the original, and `pair find` must not return it. It is
provenance, for a human reading the note later. Whether it earns its keep is worth
one round of argument before it is built.

## The two shapes, end to end

**Restyle**

1. `gdoc export --out <tmp>/doc.md --media-dir <tmp>/doc-media`.
2. Report any picture that could not be carried, before anything is created.
3. Add front matter: the title from the document, nothing else invented.
4. Read the original's parent folder, show it, confirm or ask.
5. `gdoc generate --folder-id <that folder>`.
6. Report the link, and say plainly: no comments were copied, the original is
   untouched, and nothing in the hub changed.

**Adopt**

1 to 3 as above, into the path Nail approved rather than a temp directory.
4. Publish as usual, which writes `gdoc:` and the first version record.
5. Say that the markdown is now the source, and that the original is not linked
   and will not be updated.
6. From here it is an ordinary hub document. `/gdoc-review` works on the new one.

## What could go wrong, and what the tool must say

- **The credential cannot create in that folder.** Under `service_account` this
  is the normal case, because the folder was never shared with it. Say which
  folder and which credential, and ask for a folder URL. This is the issue's own
  requirement, and it is the one branch that must be tested against a real
  refusal rather than assumed.
- **The original is huge, or full of tables Google's export mangles.** The pull
  is a conversion, so the restyled document is not the original. Say that once,
  plainly, before creating anything: "this is a conversion, not a copy".
- **Somebody restyles the same document twice.** Two new documents, no pairing,
  no way to tell them apart later except by their creation date. Adopt does not
  have this problem, because the note records the version list. Worth deciding
  whether restyle should refuse a second run, or simply not care. Not caring
  looks right: restyle is a throwaway by design.

## Open, for Nail

1. Folder derivation: option 1, 2 or 3 above. The recommendation is 2.
2. Comment appendix: offer it, or leave comments out entirely.
3. `gdoc_adopted_from`: provenance worth carrying, or noise?
4. Does restyle deserve its own command name rather than living inside
   `/gdoc-apply`? It shares no step with applying a queue, and the skill is
   already long. `/gdoc-restyle` would say what it does.
