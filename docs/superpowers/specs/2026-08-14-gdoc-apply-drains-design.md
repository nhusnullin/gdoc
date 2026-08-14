# gdoc-apply drains the document

Status: design agreed 2026-08-14. Not implemented.

## 1. Why

The pipeline works only if the order is kept by hand. `gdoc-review` fills
`pending.md`, `gdoc-apply` reads `pending.md`, and anything that happens between
the two is lost without a word. This sequence is normal and it silently drops
work:

1. `/gdoc-review` captures two items.
2. Nail reads the document again and leaves three more marked comments.
3. A colleague leaves one.
4. Nail answers the agent's reply on a fourth thread.
5. Nail fixes two sentences in the document directly.
6. `/gdoc-apply` applies the two items from step 1, generates v2, reports
   success.

Six pieces of feedback existed. Two reached the new version. The other four sit
on v1, which is now superseded, and nothing said so.

The expectation this design serves is one sentence: **when Nail runs
`/gdoc-apply`, everything anyone has said about the document reaches the new
version.**

The alternative, considered and rejected, is the check in the OAuth spec's
section 8: apply notices unreviewed comments and refuses until a review has run.
That is honest but it makes the tool's correctness depend on the operator
remembering an order. It also catches only one of the four losses above, because
a thread Nail has replied into is not `addressed`, and a direct edit is not a
comment at all.

## 2. What changes

`gdoc-apply` stops being a queue consumer and becomes the thing that collects
every change since the last version and produces the next one.

`pending.md` stays. It is no longer only a capture queue written by review; it is
the working set, and apply writes to it as well as reading it.

`gdoc-review` keeps its job but the line between it and apply moves. The split
stops being about the kind of comment and becomes about the outcome: **review
never makes a new document, apply always does.**

Local and global stop existing. Both skills run one test instead, in section 7,
and it decides the only thing that still needs deciding: is this comment an item
or a question.

## 3. What `gdoc-apply` does, in order

| Step | What |
|---|---|
| 1 | Load `pending.md` and run `gdoc read` on the document it names |
| 2 | Merge the two by comment id, and append what section 7's test calls an item |
| 3 | Fold direct edits to the document back into the source markdown |
| 4 | Work the items, one at a time, diff and approve |
| 5 | Commit, generate, record the version, clear the applied items |
| 6 | Post closure into every handled thread, and one note on the old version |

Steps 4 and 5 are what the skill does today. Steps 1, 2, 3 and 6 are the change.

Step 3 runs before step 4 and that ordering is load bearing. Every item's edit is
written against the current source markdown, so if a direct edit is folded in
afterwards it lands on text the items have already moved.

## 4. The merge

### The dedup key is the comment id

`pending.md` records `Comment id:` for every item, and `gdoc read` returns the
thread id. Nothing else is stable: the text is edited, the anchor moves, the
author is a display name.

### Three buckets

| Bucket | Meaning | What apply does |
|---|---|---|
| In `pending.md` and in `addressed` | Captured, and someone has replied since | Work it, reading the whole thread, not the captured text alone |
| In `pending.md` only | Captured during a review, nothing has happened since | Work it, as today |
| In `addressed` only | Left after the last review, or never reviewed | Apply the test in section 7. If it changes the text, append to `pending.md` and work it. If not, answer it in the thread and capture nothing |

That last row is where the two skills have to agree. A comment that reaches apply
without a review is not automatically an item. It gets the same one-sentence test
review would have given it, at a later moment. Without that, a question someone
asked in a thread that a reply re-opened becomes an item with nothing to edit.

Nothing is appended silently. Apply prints the merged list, marks which items are
new, and waits, the same gate `gdoc-review` step 2 has. A comment reaching apply
without a review is the point of this design, but it is still the first time Nail
has seen it in this session.

A fourth case is not a bucket but a stop. An item in `pending.md` whose id is not
in the fetched threads at all means the comment was deleted. An item whose thread
is now `resolved` means someone marked it done in the document. Both are reported
and neither is guessed at:

```
Item 3 (comment 1a2b3c) is in the queue but its comment is gone from the
document. It may have been deleted. The captured text is:

  > renumber the annex to follow the new section order

Apply it, or drop it?
```

### Why `addressed` is the right input

A captured item carries the agent's reply, so it is `skipped` on a re-read, not
`addressed`. That is correct and it is why the file is still needed: `addressed`
alone would never return a captured item. The union of the file and `addressed`
is exactly "accepted work plus everything nobody has looked at".

This depends on section 6 below. Without it, a thread Nail replied into after the
agent answered stays `skipped` forever and never enters the merge.

### Apply reads the thread, not the captured text

For every item that has a live thread, apply reads the whole thread. The captured
text in `pending.md` is what was said when it was captured. A reply since then may
narrow the ask, withdraw it, or answer a question the agent asked. Working from
the file alone would act on a stale version of the instruction.

## 5. Direct edits are folded back

### The problem

`generate` builds v2 from the source markdown. Anything Nail typed into v1 by hand
is not in the markdown, so it is not in v2. No warning, no diff, no trace.

### The diff

`baseline.md` is the export of v1, taken at the moment the document and the source
provably matched. Comparing it to a fresh export of the document now shows what
was edited directly. Both sides went through the same pandoc round trip, so the
formatting distortion cancels and only real edits stand out. `gdoc/baseline.py`
already says this is what the file is for. Nothing has ever done it.

### The hunks do not apply

`baseline.md` and the source markdown are different text. The source went
`markdown -> docx -> Google Doc -> markdown` to become the baseline, and that trip
is lossy. So a hunk from the diff cannot be patched into the source. Each direct
edit has to be **translated** into the source, by the agent, one at a time, with a
diff shown and approval waited for.

PR #5 adds a wrinkle. A templated document exports with a cover page, three
control tables and a contents list that exist nowhere in the source. Both sides
of the baseline-against-export diff carry them, so they cancel and never reach
the translation step. This holds only while the diff stays baseline against
export. Diffing the baseline against the source would surface the whole template
as spurious edits, so nothing may do that.

### What Nail sees

```
You edited v1 in 4 places since it was generated.

  section 2.1   "within five working days" -> "within three working days"
  section 4     one sentence added after the first paragraph
  section 6.2   the second bullet removed
  annex A       a row added to the fees table

I will translate each into the source markdown before touching any item,
because every item's edit is written against the result.

Fold them in, or continue without them?
```

Continuing without them is allowed and it is a real answer, because sometimes the
edit was a note to self. Whatever Nail chooses is said out loud in the final
report, so a discarded edit is never silent.

### Terminal-only

The diff needs an export, which is a read. It runs under `--terminal-only`.

## 6. `has_agent_reply` becomes last-reply

`gdoc/model.py` defines it as any reply by the agent, and `gdoc/filters.py`
skips any thread where it is true. So the check asks "has the agent ever
replied", not "who spoke last".

The consequence is that answering the agent does nothing. Nail replies "no, the
other sense of the word" under the agent's answer, and the thread is `skipped` on
every future run, forever.

The rule becomes: a thread needs action when the agent has not replied, **or when
someone replied after the agent's last reply.** The property is renamed to say
what it now means.

One thing this must not do is re-open a thread the agent itself closed. Under
OAuth the agent's replies are Nail's replies as far as Drive is concerned, which
is why the `[gdoc]` marker exists in the OAuth branch. The last-reply test uses
the marker, not identity, for the same reason every other rule in that branch
does.

This is the smallest change in this document and the one most likely to be
noticed, because it changes what a second `gdoc read` returns on documents that
already exist.

## 7. One test, in both skills

### Local and global are retired

Today `gdoc-review` classifies every comment as local, meaning the change fits
inside the quoted span, or global, meaning it does not. Local ones are answered in
the thread. Global ones are captured.

That was right when Nail edited the document by hand. It is wrong now, twice.

A local answer carrying replacement text is a change to the document that the
markdown never receives. Review posts the new sentence in the thread, Nail pastes
it into the document, and it becomes a direct edit that section 5 has to catch
later. The tool proposes work and then loses it.

And once apply is also deciding what to do with a comment, the term has to be
carried by both skills while only one acts on it. Two SKILL.md files have no way
to share text, so anything both must know is duplicated by hand and drifts.

### The test

**Does this comment change the document's text?**

| | Yes | No |
|---|---|---|
| Examples | Rephrase this. Add a missing clause. Renumber the annex. Apply a term change everywhere | Is this the right term? Why does this section say five days? Which PDR decided this? |
| Action | It is an item. Capture it | It is a question. Answer it in the thread |

One sentence, one table, and both skills run it. Review runs it when it reads the
document. Apply runs it on anything live that is not already in `pending.md`. The
same comment gets the same answer whichever one sees it first, which is what makes
running them in either order safe.

Whether the reply also carries proposed text is a judgement, not a category. Show
the text when there is useful text to show. A rephrasing has some. A renumbering
does not.

### What this replaces

The local and global table in `skills/gdoc-review/SKILL.md` step 3 is deleted, not
edited. The terms also appear in `2026-08-13-gdoc-ai-agent-design.md` and
`2026-08-14-gdoc-storage-and-iteration-design.md`, and this section supersedes
them there. Neither file is rewritten; a spec records what was decided when.

`forced_kind` is unaffected. `ai?` still means answer in the thread and `ai!`
still means it needs a new version. Neither decides capture, because a comment can
be both an answer and an item.

## 8. What `pending.md` holds

The file's contents do not change. Its lifecycle does.

| | Today | After |
|---|---|---|
| Written by | `gdoc capture`, run by review | `gdoc capture`, run by review **or by apply** |
| Read by | apply | apply |
| Cleared by | apply, per item | apply, per item |
| Holds | global items | every accepted change |

Apply appending to it is what makes a half-finished session resumable. Items 1 to
3 are approved and cleared, item 4 is left, Nail walks away. Tomorrow's run reads
the file, finds item 4, and does not redo the first three.

`gdoc capture` is unchanged. Apply calls the same command review does.

## 9. Closure

After a successful generate, apply posts one reply into every thread whose item
was applied:

```
[gdoc] Applied. This is in v2: https://docs.google.com/document/d/<id>/edit
```

And one top-level comment on the old version:

```
[gdoc] Superseded by v2: https://docs.google.com/document/d/<id>/edit
```

One comment, not one per thread, so anyone still reading v1 knows where to go.

An item Nail dropped rather than applied gets no reply. Apply did not do it, and
saying why is Nail's to write if he wants to. The final report names every drop,
so it is not silent to him.

Nothing is posted before generate succeeds. A reply saying "applied" against a
version that does not exist is a lie the thread keeps. If generate fails, the
markdown edits are still saved and committed, the `.docx` path is reported, and
the threads are left alone so the next run posts the truth.

Under `--terminal-only` these are printed, not posted, like every other reply.

## 10. What this takes out of the OAuth branch

Two things in `gdoc-oauth` are superseded and should be removed from it rather
than merged and then deleted:

| Where | What |
|---|---|
| Spec, section 8, "The order is review, then apply, and nothing enforces it" | The unreviewed-comments check |
| Plan, Task 11, "Check nothing was left behind" | The same check, written into the skill |

Task 11's second edit, the one that makes Step 2 name the item's `Author:`, stays.
So does Task 8, "The captured item records who asked", because `pending.md`
survives and the author field is needed either way.

Section 8's first two paragraphs stay too. They describe what `gdoc-apply` reads
today, and they are true until this design lands.

## 11. Dependencies

This design assumes both open branches land first.

**`gdoc-oauth`** supplies the `[gdoc]` marker. Section 6's last-reply test and
section 9's closure replies both use it. Without the marker they would use
identity, which OAuth breaks.

**`gdoc-template`** (PR #5) supplies `generate` writing the pairing into the
markdown frontmatter. Section 4 needs the source path, and today the only record
is `pending.md`'s own header, which is circular for a file apply may be creating.

## 12. Out of scope

- Copying comments forward onto the new version. Threads live on the version they
  were left on. Section 9's link is how you get from one to the other.
- Re-anchoring anything. Drive anchors are per document and v2 is a new document.
- Any change to `generate`, `pair`, or the template work. This design consumes
  them.
- Resolving threads. Still never done, for the reason the review skill already
  gives.

## 13. Testing

Unit, at the boundaries this adds:

- The merge, over the three buckets plus the deleted-comment and resolved-comment
  stops. Table driven, no network.
- `needs_action` under last-reply: agent replied last, human replied after,
  human replied then agent replied again, no replies, marker present and absent.
- The direct-edit diff, over a baseline and an export that differ in known ways,
  including one that differs only in pandoc noise and must produce nothing.

Integration, credential gated, skipped without one, same as
`tests/test_access_integration.py` today:

- A document with a captured item and a comment left after the capture. The merge
  returns both.

By hand, once, because no test can prove it:

- The full sequence from section 1, all six pieces of feedback, ending in a v2
  that carries all six.

## 14. Order of work

1. `has_agent_reply` becomes last-reply. Smallest, and the merge depends on it.
2. The merge, and apply appending to `pending.md`.
3. The direct-edit diff and fold-back.
4. Closure replies.
5. `gdoc-review`'s test changes from local-or-global to changes-the-text.
6. Both SKILL.md files, README, CLAUDE.md.

Steps 1 to 4 are `gdoc-apply`. Step 5 is `gdoc-review` and it is last on purpose:
until apply can drain the document, review capturing more is capturing into a
queue that still loses things.
