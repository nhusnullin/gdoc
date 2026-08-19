# Restyle a document gdoc knows nothing about

2026-08-19. Design for issue #9, restyle half only. Decisions taken with Nail
this evening. The brainstorm this narrows is
[2026-08-19-gdoc-foreign-document-brainstorm.md](2026-08-19-gdoc-foreign-document-brainstorm.md).

## What it does

Nail points `/gdoc-apply` at a Google Doc. Nothing under the root knows that
document: no `.gdoc` directory, no queue, no markdown paired to it. Today the
skill says so and stops. From now on it pulls the document, wraps it in the
house template, publishes a new document, and copies the open comment threads
across.

The original is not touched. Nothing lands in the root. No pairing, no queue, no
baseline, no version record.

## Decisions

| Question | Answer |
|---|---|
| Which folder | `--folder-id` if the command was given one, otherwise `output_folder_id` from the config. The original's parent is never read |
| The pulled markdown | throwaway. A temp directory, deleted when the run succeeds |
| Comments | copied as real Drive comments on the new document, unanchored, each message prefixed with its original author's name |
| Which comments | open threads only. Resolved threads are dropped. Every message in a copied thread comes across, replies included |
| Adopt | not in this change. The pulled markdown never becomes a note |
| Merge | not in this change. See "What this is not" |

The folder decision matters most, and it is the cheap one. Deriving the
original's parent would have meant a third door into `gdoc/guard.py`, which
CLAUDE.md forbids. Taking the folder from the command line or the config means
every id the run touches is one it was handed or one it made. The guard is
untouched by this change.

## What this is not

Two neighbouring jobs are out of scope, on purpose.

**Adopt.** Pulling a foreign document into a note that becomes the source of
truth. Shares only the pull step. Not built.

**Merge.** A document that *is* paired, where the markdown moved and the
document moved too. Nail's rule for it, recorded here so it is not lost: read
all three of the paired markdown, `baseline.md` and the live document, treat a
direct edit in the document as the higher priority, judge, and ask when in
doubt. That needs its own spec, because it needs a conflict review loop. This
change does not alter what happens to a paired document.

## The command

One new command, `gdoc restyle`, rather than a sequence the skill composes.

```bash
$GDOC restyle --doc "<document URL>" \
  [--folder-id "<folder URL>"] \
  [--title "<text>"] \
  [--template none] \
  [--no-comments] \
  [--dry-run]
```

`--dry-run` reports the document's name, the folder id and how many threads are
open or resolved, and creates nothing. The confirmation below cannot be written
without it: it names counts that only a read can produce, and it has to be on
screen before a document exists.

One command for one reason: the guard. Restyle reads the source document,
creates a new one, and then writes comments onto the new one. Inside a single
client the source and the folder are ids the command was given, and the new
document's id is learned from the create that made it. Split across three CLI
runs, the third would have to be handed the new document's id from outside,
which works but puts the burden on the skill to carry an id between steps it
could get wrong. One client, one run, no new door.

`gdoc/restyle.py` composes what already exists. `export.export_with_media` for
the pull, `generate.generate` for the publish, and a new `gdoc/comments.py` for
the copy. Nothing in `gdoc/render/` changes, so the external-programs rule is
untouched.

## The steps, in order

0. **Check the folder, then the title.** Before the document is even read. A run
   that cannot publish should not spend a download and a pandoc conversion
   finding that out, and a document with no name has nothing to put on a cover.
1. **Read the source.** `files.get(fields=name)` for the document's name, and
   `fetch_threads` for its comments. Both before anything is created.
2. **Pull it.** `export_with_media` into a temp directory: `source.md` and
   `media/`, with `restyled.docx` alongside once it is built. Any picture that could not be carried is collected as a
   warning, reported, and does not stop the run.
3. **Front matter.** The pulled markdown has none, and `frontmatter.split`
   refuses a file without it even when `--title` is given. So restyle prepends
   the shortest block that publishes:

   ```
   ---
   title: <the document's name, or --title>
   ---
   ```

   Nothing else is invented. No `doc_type`, no `owner`, no `classification`:
   the template's defaults are honest about not knowing, and a guessed owner on
   a cover page is worse than a blank one.

   Built with `yaml.safe_dump`, not an f-string. The title is a Drive
   document's name, written by somebody who has never seen a line of YAML, and
   a colon in it is the ordinary case. `title: Q3: Roadmap` is not a mapping,
   so the run would die parsing its own front matter before publishing
   anything, on a document whose only crime was being called "Q3: Roadmap".
4. **Publish.** `generate` into the folder, with the house template, `--out`
   inside the temp directory. No `--baseline-root`: there is no note to write
   `gdoc:` into and no baseline that would mean anything.
5. **Copy the comments.** Only if step 4 returned a `doc_id`. Each open thread
   becomes one unanchored comment on the new document.
6. **Clean up.** Delete the temp directory when the document exists. Keep it
   otherwise, because then the pulled markdown is all the run produced: reported
   as `workdir` when the publish was refused, and named in the error message
   when something raised instead. The one exception is a failure before the pull
   landed, where the directory is empty and is deleted.

## How a copied comment reads

A thread on the original:

```
quoted: "the fee is 2%"
William Mejia: this number is wrong
  Anna: fixed in the table
  Nail: thanks
```

arrives on the new document as one comment:

```
On "the fee is 2%":

William Mejia: this number is wrong
Anna: fixed in the table
Nail: thanks
```

Four rules, and each one has a reason.

- **The quote line comes first, when the thread had one.** Anchors cannot
  survive. The original comment points at a span of text; the new document is
  the house template, so its structure is different by construction and there is
  no anchor to compute. Quoting the text is the only pointer left, and it is one
  a human can follow. A thread with no quote gets no quote line.
- **Every message keeps its author's name.** Drive writes every comment as the
  authenticated user, and there is no field for saying otherwise. Under `oauth`
  all of these arrive authored by Nail. The name prefix is what stops the new
  document from claiming he said all of it.
- **No `[gdoc]` marker.** The marker means "gdoc wrote this". These are other
  people's words, and stamping them would claim authorship of text gdoc only
  carried. The marker's real job, telling `has_agent_reply` which replies to
  skip, is about replies and is unaffected.
- **No markdown check.** `reply.assert_plain_text` refuses `**` and `#` because
  gdoc's own replies render literally in a comment. A copied comment is somebody
  else's text, verbatim. Refusing to carry it because a human typed an asterisk
  would lose the comment to protect its formatting.

Comments are created one at a time. A thread that fails is recorded in
`comment_errors` and the rest keep going, for the same reason `generate` records
`baseline_error` rather than throwing: the document is already real, and a run
that dies here would hide it.

The copy as a whole gets the same treatment one level up. `copy_comments` guards
each create, so a failure that escapes it is the loop itself, which is exactly
the kind of fault a narrow catch lets through. `restyle` catches it, records it
in `comment_errors`, and still returns the document. `cli.cmd_generate` defends
its baseline and pairing writes this way and says why at length; the rule is the
same here. Once Drive has created the document, nothing below may withhold its
id.

## The JSON

```json
{
  "source_doc_id": "1abc...",
  "source_name": "MC incentive routes",
  "title": "MC incentive routes",
  "folder_id": "0By...",
  "doc_id": "1xyz...",
  "link": "https://docs.google.com/document/d/1xyz.../edit",
  "comments_copied": 4,
  "comments_skipped_resolved": 3,
  "comment_errors": [],
  "image_warnings": [],
  "drift": null,
  "reason": null,
  "workdir": null
}
```

`doc_id` decides whether a document exists, exactly as in `generate`. Every other
key is a fact about a document that already exists, or a reason none does.

- `workdir` is null on success and the temp directory's path when anything
  failed, so Nail can get at the pulled markdown.
- `comments_skipped_resolved` is reported rather than silent. A run that copies
  four of seven threads should say where the other three went.
- `image_warnings` carries `export_with_media`'s warnings verbatim. Issue #28
  showed that a lost picture is only noticed when somebody reads the new
  document, which is too late.
- `--no-comments` sets `comments_copied` to null rather than 0, so "you asked me
  not to" and "there were none" stay different answers.

Refusals exit 1 with `error` as the only key, as everywhere else. Three of them:
no folder anywhere, naming both the flag and the config key; a document with no
name and no `--title`; and a document URL passed to `--folder-id`, which
`extract_folder_id` already refuses by name.

A publish that was refused by Drive is not one of them. It exits 0 with `doc_id`
null and `reason` set, the same as `generate`, so the skill reads the keys rather
than the exit code.

## What the skill does

`skills/gdoc-apply/SKILL.md`, Step 1. The table gains a row, and the paragraph
that currently refuses is replaced.

Today:

> If neither answers **and he gave you a document link**, there is no source
> markdown for it here. That is a document he does not own the source of, so the
> output is a note for him, not a new document. Say so and stop.

That paragraph goes. In its place: this is a restyle, and it runs after one
confirmation. The confirmation is one screen, not a yes/no on nothing:

```
"MC incentive routes" is not paired to anything under this root.
Restyle it: a new document in the house template, in folder 0AFolderId
  (from your config).
  7 comment threads, 4 open. The 4 open ones come across, authored by you,
  with the original names in the text. The 3 resolved ones do not.
  This is a conversion, not a copy: tables and layout may come across differently.
  Nothing is written under this root. The original is untouched.
Go ahead? y
```

One confirmation, not none, and it is not ceremony. It creates a document other
people can read, in a folder, and there is no undo. The line that earns its
place is the folder: publishing a restyled copy of somebody else's document into
the wrong folder is the failure worth being slow about. It is the folder id
rather than the folder name, because nothing resolves a folder's name and one
more Drive call to pretty-print a confirmation is not worth the reach.

Steps 2, 3, 4 and 5 of the skill are skipped: no items to work through, no
markdown edited, nothing for `generate` to do, and no queue to clear. The single
`restyle` call has already published by then.

`--terminal-only` refuses a restyle rather than adapting it. The flag means
"change nothing in Drive", and restyle uploads before it has anything to show.
`--dry-run` stays available, because it creates nothing.

## Tests

TDD, and the failing test comes first in each case.

`tests/test_comments.py`, new:

- a thread with replies becomes one body, author name on every line, in order
- a quoted thread gets the `On "..."` line, an unquoted one does not
- a resolved thread is dropped
- a body containing `**bold**` is copied unchanged, not refused
- a document name that is not plain YAML, a colon in it above all, still
  publishes and keeps its title
- a copy that fails as a whole still returns the document, with the failure in
  `comment_errors`
- `comments.create` is called with the new file's id and no anchor
- one thread failing does not stop the next, and lands in `comment_errors`

`tests/test_restyle.py`, new, against a fake Drive:

- the happy path: pulled, published, comments copied, temp directory gone
- the document name becomes the title, and `--title` overrides it
- front matter is prepended, and it is three lines
- a failed publish keeps the temp directory and reports it as `workdir`
- a failed pull deletes it instead: the directory is still empty, and leaving it
  would leak one per failure, unnamed and unreported
- a failure after the pull but before the document exists, which is what a
  template build error on a foreign document looks like, keeps the markdown and
  names the directory in the error, because a raise carries no `workdir` key
- `--no-comments` reports null rather than 0, and creates no comments
- no folder anywhere is a refusal naming both the flag and the config key
- nothing is written under the root

`tests/test_cli.py`, extended: argument parsing, folder resolution from the
config, and the exit code for each outcome.

`tests/test_guard.py` needs nothing new. No new door, so there is no new rule to
test. That is the point of the folder decision.

## Open

Nothing blocking. Two things worth a sentence when it is built:

1. Restyling the same document twice makes two documents, and there is no record
   linking either to the original. That is what throwaway means, and the
   brainstorm already concluded not caring is right.
2. Under `service_account`, the target folder has to be shared with the service
   account as Content manager, and a refused create means that share is missing.
   The existing message covers it.
