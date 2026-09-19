# gdoc v2 Milestone 13: export and align, specification

2026-09-19. A Google Doc comes back into the hub as a Markdown note with its
pictures beside it, a note may be published more than once, and the align
skill carries changes in both directions with the person's agreement. The
binary reads, writes files on disk, and decides nothing. The skill judges.

This is the specification. The plan with its task list follows it in a second
document once Nail has read the scenarios and this file. Three roles reviewed
the design in two rounds on 2026-09-19: a senior engineer, a product owner and
a QA engineer, and a third round over the whole file. Their findings are
folded in below.

## Principles

Serves: 3, uncertainty never resolves toward the destructive answer. Export
never overwrites and has no flag that could: a taken name gets the next free
number, always. Every change export finds in a document reaches the file as a
marked fact, and a person says what the note keeps and when. Every change the
skill carries into a document is a suggestion. Also 2, the hub is the working
directory: the pairing lives in the note's own front matter, now as a list of
the documents the note has met, and the pictures in an `assets` folder beside
it, nothing hidden. And 4: the markers are gdoc's, so the skill can undo every
one of them, and the person decides when.

Strains: none. Two standing decisions are superseded rather than strained: the
markdown as the source of truth (2026-08-13) and publish running once
(2026-08-29, restated 2026-09-11 and 2026-09-16). Both are register rows
below.

## Overview

| Piece | Today | After |
|---|---|---|
| a Google Doc into the hub | `read` prints a projection for a session, no file, `[image]` for a picture | `gdoc export` writes one Markdown file per tab and the PNGs in `assets/` beside it |
| a note already published | `publish` refuses it | `publish` makes a new document and adds it to the note's list |
| the `gdoc:` block | one `document_id`, schema 1 | a `documents` list, one entry per document, schema 2; schema 1 still reads |
| which document a command works on | the one `document_id` names, checked against the URL | the URL, checked against the list |
| a picture in a document | `[image]` inline, silence for a floating one | a PNG file in `assets/` beside the note, or the note's own picture when the bytes match |
| link targets and list numbering | lost in `read` | decoded in `internal/docs`, so `read` gains them too |
| changes from the document into the note | by hand | the align skill merges from the export with agreement |
| changes from the note into the same document | `propose` per change, composed by hand | the align skill composes the difference and proposes each change |
| a body carrying gdoc's markers | `publish` prints `{+` as prose | `publish` and `build` refuse it by line |
| a document with two tabs | every writer refuses it; `read` prints both in one stream | `export` writes one file per tab, each naming the document and its tab |
| a live review session on the document | nothing else touches the note meanwhile | export and the merge run beside it; nothing is stopped |

## Decisions Nail took, 2026-09-19

Each is one DECISIONS.md entry or one row in an entry. A task that finds one
wrong stops and says so.

1. **Export creates the note when none exists.** `gdoc export <url> --out
   <path>` writes the file at that path when nothing is there, with a
   `gdoc:` block naming the document. The colleague who asked has a Doc and no
   note, and this is the case that serves them.

2. **Nothing is overwritten, and there is no flag that could.** A taken path
   gets the next free number: `<stem>.2.md`, then `<stem>.3.md`. Pictures go
   into an `assets` folder beside the note, created when it is missing, as
   `assets/<stem>-1.png` onward from the next free number. Nail struck
   `--force` on 2026-09-19: a flag that replaces a note is a false door, and
   a person who wants the old note gone exports to the new file, reads the
   old one, and renames by hand. The reply names every file it wrote.

3. **The export file holds everything the document says, and decides
   nothing.** Pending insertions and deletions are marked the way `read`
   marks them, `{+words+}[s:ID]` and `{-words-}[s:ID]`, comment anchors as
   `[[c:ID]]words[[/c]]`, and every marker is escaped by the rules
   `internal/view` already holds. Beyond `read`, the file carries link
   targets, list numbering, pictures as files, and the house prelude
   stripped. The align skill judges what the note keeps, with the person,
   and when. Nail's correction on 2026-09-19: a person may export and keep
   the file in the hub with its markers for as long as they like, with no
   time to resolve them today. The markers are gdoc's, the skill can undo
   them on any later day. What a marked file cannot do is be published or
   built, or have its text sent into a document: `publish`, `build` and every
   writer that takes text from a note refuse a marker by line. This is the file
   that is the note, the no-note case of decision 1. A copy made beside an
   existing note is not kept: decision 13 deletes a stale one and exports
   again, because export is cheap and the copy holds nothing the document
   does not. A person who wants to keep a copy as a version renames it by
   hand and removes the `note:` line under `exported`, which is what the
   writers' refusal on a copy tells them.

   The same markers must never travel the other way. `annotate` and `reply`
   refuse text carrying a marker in `internal/plaintext`, beside the markdown
   refusal. `propose` refuses one in its read of the `--from` file, because
   document text never passes through `plaintext`. The align skill resolves
   the markers before it proposes anything from a marked note.

4. **The block is a list of documents.** One entry per document the note has
   been published to or exported from, each holding its own facts:
   `published`, `exported`, `folder_id`, `suggestions_seen` and `proposals`.
   Schema becomes 2. A schema 1 block reads as a one-entry list and is
   rewritten as schema 2 on the first write that changes it. Proposals and
   the snapshot sit under their document, because a withdraw must never send
   an id from another document. Any write that changes the block renders it
   as schema 2, and that includes the snapshot `suggestions --md` writes after
   every successful read, so a note published before this milestone flips on
   its first run of any command that writes. The reply says so in one line:
   the block was rewritten to schema 2. A block read and written unchanged
   stays byte for byte as it was, which is the test that already holds.

5. **No side is the source of truth by rule.** Nail's words: only the person
   decides what the source is, and decides it per run. The block records dated
   facts, when a document was published from the note and when it was exported
   into it, and no field says which side was right. The 2026-08-13 decision is
   superseded.

6. **A note may be published more than once, and each publish makes a new
   document.** The old document keeps its URL and its threads, and its entry
   stays in the list untouched. The publish skill says once that a new
   document is being made and that the old one goes stale, then runs.

7. **Publishing into the same document is the skill's judgement, on the
   binary's options.** The binary offers three doors and no fourth: publish
   to a new document, propose a change as a suggestion, export the document
   into the hub. It never replaces a body, which the 2026-08-29 decision and
   the guard hold. When a note is already paired with a document and the
   person asks to publish there, the skill reads that as a merge: it reads the
   note and the document, composes the difference, and proposes each change as
   a native suggestion, in the best way it can, for the document's owner to
   accept or reject.

8. **Pending suggestions are exported as markers, never decided.** gdoc's own
   pending proposals come back with ids the note already records, so the skill
   keeps the note's words for those without asking. A person's pending
   suggestions are shown by id and author, and the person says which the note
   takes. The document is untouched either way.

9. **A colleague on an older binary is refused by name.** The shipped
   decoder runs the strict YAML read before it looks at the schema number, so
   on a schema 2 block it refuses the unknown key `documents` by name and
   never reaches its schema sentence. It says nothing about updating, because
   no shipped binary can be taught a new sentence. The daily notice in `help`
   already says a newer release exists, and that is where the colleague
   learns to run `gdoc update`. Nothing in the note breaks. The plan's Task 2
   pins the exact sentence as a literal, and scenario 18 quotes it.

10. **Two routes, matched by order.** The Docs read carries the text with
    suggestion and comment ids, link targets and list numbering. The docx
    export carries the picture bytes, inline pictures, floating pictures and
    Google Drawings alike, as PNG. Nothing else crosses the two routes, so the
    k-th picture in one is the k-th in the other. A count mismatch means a
    placeholder for every picture and no file written. The `contentUri` in the
    Docs JSON is a `googleusercontent.com` host the guard does not admit, and
    it stays out.

11. **A document with two tabs exports as two files.** Nail's correction on
    2026-09-19, after reading the scenarios: a tab is a document of its own to
    a reader, so one file per tab, never one file for both and never a
    refusal. The first tab lands at `--out`, and each further tab lands beside
    it as `<stem>-<tab title>.md`, slugged the way a heading id is, with the
    numbering rule of decision 2 when a name is taken. Each file's block entry
    names the document and its `tab_id`. Pictures follow their tab. The
    writers keep refusing a target with more than one tab until a live
    measurement says what a write into one tab does, so a note exported from
    a second tab can be published as a new document and cannot yet propose
    into its tab. The 2026-08-29 decision that gdoc defends against tabs holds
    for the writers and is narrowed for this one reader.

12. **The align skill judges logic, not only paragraphs.** Nail's words: a
    document is coherent from beginning to end. A merge that takes a paragraph
    from each side can leave a document that contradicts itself, so the skill
    reads the merged whole once more and names every place where the logic
    broke. Those are conflicts too, shown beside the paragraph conflicts, and
    the person decides each. The skill's own file holds the examples.

13. **A stale copy is never merged from.** Export is read-only and cheap, so
    when the skill finds a copy left by a run that died, it says when the copy
    was made and exports again. When the fresh export's body is the same as
    the stale copy's, the stale copy is deleted. When it differs, the person
    edited the copy or the document moved since, and the skill shows the
    difference and asks before deleting anything. Nail's correction on
    2026-09-19: there is no way to know what the previous run had decided,
    and asking the person to reconstruct it is worse than one more export.
    The comparison is what keeps a copy the person worked on from vanishing.

14. **Nothing stops a live review session.** Export reads, and the merge
    writes the note. A live session, which a colleague may have started, keeps
    running. Every block write in the binary reads the file fresh before it
    writes and replaces it in one rename. The skill edits the note in small
    edits with a tool that refuses to write a file changed since it was read,
    and re-reads on a refusal, so a snapshot the live session wrote in between
    is never dropped. Nothing is promised about a rename on the skill's side.
    Nail's correction on 2026-09-19.

15. **`export` is a command, not a flag on `read`.** Nail's decision on
    2026-09-19. `read` stays a pure read that
    writes nothing on disk, and its help stays short. `export` gets its own
    row in the command table, so `help` and completion follow, and its help
    entry is where the numbering rule, the `assets` folder and the block go.
    The path flag is `--out`, the flag `build` already uses for a file to
    write, and never `--md`, which means "the paired note" on five commands.
    The command table grows from fourteen rows to fifteen, and the tests that
    spell the usage lines out gain one line.

16. **Two skills, `gdoc-export` and `gdoc-align`.** Nail's decision on
    2026-09-19, reading the scenarios: a colleague who says "bring this into
    the hub" has nothing to align, so the skill that answers is not called
    align. `gdoc-export` serves a document with no note. `gdoc-align` serves
    a paired note, in both directions. The marker rules are written once, in
    a file beside `gdoc-export`'s SKILL.md, and `gdoc-align` points at it,
    the way the SVG script sits beside the publish skill. The repo then
    carries five skills: review, publish, restyle, export, align.

17. **Skills for everything that judges, binary commands for everything that
    does not, and no slash commands.** Nail's decision on 2026-09-19, on the
    product owner's proposal. Every skill starts from a colleague's sentence,
    and none carries a by-name-only line, because a session with `gdoc` on
    PATH can run any command by hand, and a skill that does not fire does not
    stop the action, it only skips the asking. So the asking lives inside the
    skills: `gdoc-publish` on a paired note stops and asks, a new document or
    a merge through `gdoc-align`; `gdoc-align` shows the list of paragraphs
    and asks once before its first `propose`, because a merge can send twenty
    suggestions where a review sends one; `gdoc-restyle` already asks through
    its survey. Each skill's first line says which skill took the request,
    since two now answer "publish" and "align", and the plugin form is
    `/altery:gdoc-align`. `needs:` becomes `v2.4.0` on review, publish,
    export and align, because a paired note publishes and the schema 2 block
    reads only on the new binary; restyle takes no note and stays at
    `v2.0.0`.

## The command

`gdoc export <url> --out <path>`

Reads the document through the Docs read and the docx export, and writes one
Markdown file per tab and their PNGs under `assets/` beside it. It sends no request that
can change a document, opens no new host, and needs no grant. It is invoked
from a session by name, and never from an `ai!` comment: the review skill
answers such a comment with one reply saying so.

A fifteenth command, not a flag on `read`, decision 15.

What it does with `--out`, which is the path to write and not a paired note, the flag `build` uses for the same thing:

- The path is free: the file lands there. Its block holds one entry for the
  document, with `exported: {at}`.
- The path holds a note whose list names this document: the note is left as
  it stands except for one stamp, `exported: {at}` on that document's entry,
  through the byte-preserving block write. The export lands beside it at the
  next free number, and its block holds one entry with `exported: {at, note:
  <path>}`, the note it belongs to. Every writer refuses a file whose entry
  names a note, in one sentence pointing at the note.
- The path holds a note whose list does not name this document: refused at
  the door, naming the documents it does name. The same rule every writer
  holds today.
- The path holds a file with broken front matter: refused, naming what is
  broken, as `read --md` refuses it today.
- The path holds any other file: the export lands beside it at the next free
  number, and the reply says the path was taken.

A document with more than one tab writes one file per tab, decision 11. The
envelope lists every file with the tab it came from. Each tab's path takes
the same checks as `--out`: a note there naming other documents, or broken
front matter, refuses the whole run before any file is written. The tab slug
is the rule `body` uses for a bookmark name, lower case, every run of
characters outside letters and digits turned into one hyphen, and nothing
else: an empty title gives `tab-<n>` by position, two tabs with one title
take the numbering rule of decision 2, and a child tab is a file like any
other, in document order, slugged from its own title.

What it refuses: a note at `--out` naming other documents, broken front matter
at `--out`, and a URL that is not a Google Doc. Nothing else.

The envelope carries: the files written, each with its tab, the pictures as a list of `{file,
matched}` where `matched` is the note's own picture when the bytes matched and
empty otherwise, the count of pending suggestions, and beside it how many are
gdoc's own when `--out` holds a note whose entry for this document lists
proposals and no such count otherwise, the count of open threads, what was
stripped as a list of the prelude pieces recognised with the text each held,
so an edit inside a cover table is in the envelope and the skill can compare
it with the note's keys, one line when the note's block was rewritten to
schema 2, and one warning per thing it could not carry: a floating picture's
placeholder, a footnote flattened, a Drawing with no PNG, a count mismatch.

## The file

The body, in this order of rules:

- **Text** as `read` prints it: headings, paragraphs, lists, tables, with the
  escaping `internal/view` holds. The across-runs gap in that escaping,
  `docs/backlog/escaping-across-run-boundaries.md`, is a blocker for this
  milestone and not a backlog item any more, because the skill must undo the
  escaping exactly.
- **Suggestions and comments** as markers with their ids, as decision 3 says.
- **Links** as `[words](target)`. A heading link becomes `#slug` from the
  heading's words, the reverse of what `body` writes. An external link keeps
  its URL. A chip keeps its label and its target.
- **Lists** with their numbering. `internal/docs` gains the `lists` map and
  the paragraph's bullet, so a numbered list is numbered and a nested one is
  nested. `read` gains the same, under `--structure` too: a numbered item
  prints `1.` and a link prints `[words](target)`. The five golden files
  under `view/testdata` change, and the marker table in `view/doc.go` names
  the two new shapes. That closes a line in `docs/guide/reading.md`.
- **Pictures** as `![](assets/<stem>-<n>.png)` on their own line, or the
  note's own link when the bytes matched. The `assets` folder sits beside the
  note and is created when missing. A picture already there under that name
  is never replaced; the next free number is taken. The note's own pictures
  are found by parsing the file at `--out` with goldmark, taking every image
  destination, resolving it against the note's directory the way `body`
  resolves one, and reading PNG and JPEG only; a `data:` or `http`
  destination is skipped. Only the file at `--out` is read, never a second
  note naming the same document. The match is off until measurement 2
  passes, and until then every picture is written and the reply says so. A floating picture lands as a placeholder
  comment after its anchoring paragraph, plus the file. A Drawing is a
  picture. An equation or an embedded object is a placeholder comment and a
  warning, as today.
- **Footnotes** as `[^n]` with the note at the end of the file, and a
  warning, because `publish` refuses them and the skill must say so before
  the file becomes a note. Two known losses ride with them and are warned
  about by name: a suggestion inside a footnote exports as plain text, and a
  comment anchored in a header, a footer or a footnote gets no mark
  (`docs/backlog/suggestions-inside-footnotes.md` and
  `comment-anchors-in-headers-and-footnotes.md`).
- **The house prelude stripped.** A document `publish` made carries the cover,
  the three tables, the legend and the contents list in the house layout
  order, and `{n}-` heading numbers as text. A document `restyle` made carries
  the same under gdoc's named range. Export strips by layout position and
  lists what it removed, with the text each piece held. As code: a restyled
  document is stripped over its `MarkerName` range. A published document has
  no marker, so the span is from the body's start to the end of the contents
  element, and only when the span before the first level-one heading holds
  three or more tables; `internal/docs` gains the `tableOfContents`
  structural element, which it drops today. An edited cover title still
  matches; an added heading inside the cover breaks the match, nothing is
  stripped, and the warning names what stood there. What it cannot recognise
  stays in the file and is named in a warning, and the skill shows it before
  merging. The author's front-matter keys the cover was built from are never
  written by export.
- **Comments never reach a note body.** The skill removes the anchors and
  lists the threads. They live in Drive.

The front matter of a file export creates holds the `gdoc:` block and nothing
else. The skill adds `title` when it makes the file a note, because a note
without one cannot build.

## The block

```yaml
gdoc:
  schema: 2
  documents:
    - id: 1AbC...
      folder_id: 0Xyz...
      published: {at: 2026-09-19T10:00:00Z, title: ..., house: embedded}
      exported: {at: 2026-09-19T14:00:00Z}
      suggestions_seen: {at: ..., items: [...]}
      proposals: [{id, comment_id, at, quoted}]
    - id: 1DeF...
      published: {at: 2026-09-19T15:00:00Z, title: ..., house: embedded}
```

- An entry may carry `published`, `exported` or both, and `tab_id` when the
  file came from one tab of a document with several, decision 11.
- Order is the order of writing. Nothing reads it as "newest". The command
  works on the entry whose id the URL names, and a URL naming no entry is
  refused, the rule every writer already holds against a different id.
- `exported.note` appears only in a copy written beside an existing note, as
  a path relative to the copy's own directory, and it makes every writer
  refuse the file: `publish`, `propose`, `withdraw`, `annotate`, `reply`,
  `comments --md` and `suggestions --md`. It is the one field that is not a
  dated fact, and its job is to keep proposals out of a stray copy. The
  refusal says how to adopt the copy as a note: delete the `note:` line
  under `exported`, by hand, the way today's refusal says to take a block
  out by hand. Nothing in the binary removes it, because the binary cannot
  know the person meant to keep the copy.
- A schema 1 block is read as one entry. `WriteAnUnchangedBlockIsByteIdentical`
  keeps holding: a schema 1 block read and written unchanged stays schema 1,
  and the rewrite to schema 2 happens only on a write that changes something.
- Two notes may name one document, since `publish` no longer refuses a paired
  note and export can create a second. The binary cannot search the hub, so
  it does not look. The align skill looks beside the note for a copy naming
  it, and the reply of any writer names the file it wrote into.

## The export skill

`gdoc-export`, for a document with no note, decision 16. It runs `gdoc
export`, names the file from the document's title, slugged, and asks once
before writing. It adds `title` to the front matter. It resolves the markers
with the person, or leaves them when the person has no time, and says in one
line that any later session can resolve them. It reports every file the
binary wrote. It runs `gdoc export` and `gdoc comments`, for a thread's
author and first line, and nothing else. When the document has tabs it says
how many files came out and where. It never publishes, never proposes, and
never runs from an `ai!`.

The marker rules, what each marker means, how it is escaped, and how a
session undoes it, live in one file beside this skill's SKILL.md, and
`gdoc-align` reads the same file.

## The align skill

`gdoc-align`, for a paired note, two directions, and the person's sentence is
the same for both. The skill reads the state and says which direction it is
taking before it does anything. A `--out` path with no note is not this
skill's case: it says so and names `gdoc-export`.

**Document into note.** Runs `gdoc export`, reads the copy and the note, and
shows three lists: only in the document, only in the note, changed on both
sides. It merges what the person agrees to, never deletes what only the note
holds, resolves the markers the person wants resolved, removes the comment
anchors, and deletes the copy when the run ends. gdoc's own pending proposals
are not asked about: the note already holds those words. Open conflicts stay
as the note had them and are named in the reply. A person who has no time
today says so, and the file stays in the hub with its markers until they come
back; the skill can pick it up on any later day, because the markers are its
own.

**Note into document.** Reads the note and `gdoc read <url>`, composes the
difference, shows the list of paragraphs it would change and asks once,
decision 17, then runs `gdoc propose` once per changed paragraph, each with
its 🤖 comment saying why. A paragraph the note lost arrives as a suggested
deletion. The document's owner accepts or rejects in the browser.

**Both moved.** Both directions in one run, the first before the second. The
skill compares paragraph by paragraph, merges what only one side changed
without asking, and shows only the conflicts: the paragraphs both sides
changed, and the places where the merged whole stopped being coherent,
decision 12. The person decides each.

**Resolve only.** A third run, over a marked file already in the hub and
with no export: the skill runs `gdoc suggestions <url> --md <note>` first, so
a suggestion accepted or rejected in the document since the export is
resolved from that answer without a question, then asks the person about each
one still pending, and removes the comment anchors.

The commands the skill runs, and nothing else: `gdoc export`, `gdoc read`,
`gdoc comments` for a thread's author and first line, `gdoc suggestions` for
what is still pending, and `gdoc propose`. The Docs API never says who wrote
a suggestion, so a pending suggestion is listed by id and words, never by
author.

The skill promises: every pending suggestion and open thread is reported by
id; the note ends up saying what the document says where the person agreed,
and nothing already in the note is deleted; every copy it made is gone when
the run ends, or named in the reply when the run stopped; before it proposes
from a note it looks in the note's folder for another file naming the same
document and names it, because two files can hold proposals for one document
and the binary cannot see the folder. Two sentences it says when they apply:
that a document gdoc did not publish has no house style and `/gdoc-restyle`
is the route to one, and that a new document made from an exported note
carries none of the original's threads.

The skill refuses: writing the note before showing the merge; deciding
anyone's pending suggestion; touching author keys beyond adding a missing
`title`; running from an `ai!`; publishing; deleting anything but its own
copies; running git.

Before any run it looks for every copy beside the note whose entry names the
note. For each one it says when it was made, exports again, deletes it when
the fresh body matches, and asks when it does not, decision 13. A live review
session on the same document is left running, decision 14.

Every scenario the skill is in names the skill and the binary commands it
runs, in the order it runs them, so a reader can tell what the binary did and
what the skill judged.

## What changes in publish and the other writers

- `publish` accepts a paired note. `unpaired` and its refusal go, with
  `TestPublishRefusesANoteThatIsAlreadyPaired`. It appends an entry to the
  list and touches no other entry. The publish skill says once that a new
  document is being made.
- `publish` and `build` refuse a body carrying gdoc's markers, `{+`, `{-`,
  `[[c:` and their closers, by line, the way they refuse a footnote. That is
  what stops a forgotten marker from publishing as prose.
- Every writer reads the entry the URL names and writes its facts there:
  `propose` records under its document, `withdraw` reads only its document's
  proposals, `suggestions --md` compares against its document's snapshot.
- Every writer refuses a file whose entry carries `exported.note`.
- The refusal sentences change. "Open that document or take the block out"
  is gone from `publish`. A URL naming no entry says which documents the note
  does name.

## Measurements before any matching code

Three live reads on one fixture document holding an inline picture, a Google
Drawing and a floating picture, each recorded in `docs/v2/MEASURED.md` before
a line of the picture code is written:

1. **Body order in both routes.** That the docx export lists `w:drawing`
   elements in the order the Docs JSON lists inline objects, and where a
   floating picture falls in each.
2. **Byte identity.** That a PNG `publish` uploaded comes back from the docx
   export with the same bytes. All three roles said this was measured for an
   in-place restyle and not for this route. Until it passes, the hash match
   is off and every picture is written as a file.
3. **A two-tab document's docx.** What the export carries for the second tab
   and in what order, so each tab's pictures can be told from the other's.
   If the docx carries one tab only, the second tab's pictures are
   placeholders and the envelope says why.

## Tests

The plan names each test beside its task. The list the roles gave:

- Golden: recorded Docs JSON and docx zip per fixture, the expected `.md` and
  the expected PNG hashes, deterministic like `view`'s. Fixtures: `view`'s
  five, plus links, numbering, two identical pictures, a floating picture, a
  publish prelude, a restyle prelude, no prelude.
- The file path: free, `.2`, `.3`; pictures under `assets/` from the next
  free number, the folder created when missing; nothing is ever replaced and
  no flag exists that could; the reply names every file written.
- Refusals: a note naming another document, broken front matter, a copy
  whose entry names a note, refused by every writer.
- Two tabs: two files, each entry naming its `tab_id`, pictures under their
  own tab; a tab title that collides with the note's stem takes the next
  free number.
- The inverse reader over every fixture: unescape after escape is identity,
  including a marker split across two runs.
- A picture count mismatch gives placeholders and writes nothing.
- `publish` and `build` refuse a marker by line.
- The block: a schema 1 block written before today reads as one entry and
  writes back byte-identical; a schema 2 block round-trips; an unknown key
  under an entry is named; a proposal is recorded under its own document; a
  withdraw never sends an id from another document; `GoneSince` reads the
  snapshot of the document read.
- Live, under `GDOC_LIVE_WRITE`: publish the pictures fixture, export it, the
  PNG hashes are equal; propose, export again, the id is in the file; publish
  the paired note again, two entries; trash both.
- Round trip: the six `body` notes published, exported and compared against a
  named list of known differences with reasons, in the drift gate's shape. An
  unnamed difference fails.
- The sentences the third review found no test for, each with the test that
  pins it: export opens only what the read policy opens
  (`TestExportOpensOnlyThePolicyReadOpens`); the stamp on an existing note
  changes no other byte (`TestExportStampsTheNoteAndChangesNoOtherByte`); a
  URL that is not a document is refused
  (`TestExportRefusesAFileThatIsNotADocument`); the envelope's counts and the
  stripped list (`TestExportEnvelopeCountsPendingOwnThreadsAndStripped`); a
  heading link round-trips to its slug over the ampersand and accent headings
  in `05-edge-cases.md` (`TestAHeadingLinkRoundTripsToItsSlug`); a chip keeps
  label and target (`TestAChipExportsLabelAndTarget`); a footnote is written
  and warned (`TestAFootnoteIsWrittenAndWarned`); an edited prelude stays and
  warns (`TestAnEditedPreludeStaysAndWarns`); `annotate` and `reply` refuse a
  marker (`TestPlaintextRefusesAMarker`) and `propose` refuses one in its
  `--from` file (`TestProposeRefusesAMarkerInTheFile`); a changing write
  rewrites schema 1 as 2 and the reply says so
  (`TestAChangingWriteRewritesSchemaOneAsTwo`); a tab path takes the door
  checks (`TestATabPathIsCheckedLikeOut`).
- What no test can pin, said plainly: the skill's judgement in scenarios 3,
  7, 13, 15 and 20, and decisions 12 and 13. `skills_test.go` reads the
  skill's text only. Those are held by the scenarios and by reading the
  skill.

## Documents that change

- `README.md`: a fourth recipe, paste a Doc link and get a note.
- `docs/guide/how-it-works.md`: the source-of-truth line becomes the person's
  choice per run, and one line on PNG files beside the note.
- `docs/guide/publishing.md`: the block example in its list shape, the
  publish-many rule, and one sentence that the publish skill knows the SVG
  route.
- A guide page for export: the numbering rule, the PNG names, what the
  markers mean to a person who opens the raw file, one file per tab.
- `skills/gdoc-publish/SKILL.md`: Step 1 no longer says a block means
  published; the once-only sentence goes; on a paired note it stops and asks,
  a new document or a merge through `gdoc-align`; the SVG section from
  `docs/backlog/publish-skill-renders-an-svg-picture.md` lands; `needs:
  v2.4.0`.
- `skills/gdoc-review/SKILL.md` and the two new skills: `needs: v2.4.0`, and
  a first line naming the skill that took the request. Restyle stays at
  `v2.0.0`.
- `skills/gdoc-review/SKILL.md`: an `ai!` may not run export.
- `skills/gdoc-export/SKILL.md`: new, with the marker rules in a file beside
  it.
- `skills/gdoc-align/SKILL.md`: new.
- `.claude-plugin/`, `release/install.sh --skills` and `install.sh`: two more
  skill folders in the lists that carry them.
- `docs/v2/SPEC.md`: `publish` runs more than once; the diff section
  describes the skill as built; a new `export` section. Each change is a
  DECISIONS.md entry the same day.

## DECISIONS.md entries and register rows

One entry dated 2026-09-19 with these rows, and the register updated:

| Row | Status change |
|---|---|
| The markdown is the source, the Google Doc is a rendering (2026-08-13) | superseded 2026-09-19 (no side is the source by rule; the person decides per run) |
| Publish runs once. Everything after travels as suggestions (2026-08-29, restated 2026-09-11 and 2026-09-16) | superseded 2026-09-19 (a note may be published more than once; each publish is a new document) |
| The publish record: publish is its only writer (2026-09-08) | superseded 2026-09-19 (export writes the block too, and the block is a list of documents) |
| gdoc never replaces the body of a document that already exists (2026-08-29) | holds, and is what makes decision 7 a merge rather than a replace |
| gdoc proposes. It never accepts and never rejects (2026-08-29) | holds; decision 8 rests on it |
| Export has no `--force`, and a taken name takes the next free number | new, serves 3; the one output file that cannot be replaced |
| Two routes matched by order, `contentUri` stays out | new, serves 1 and the guard |
| Tabs are out of scope, and gdoc defends against them (2026-08-29) | holds for the writers; narrowed 2026-09-19 for `export`, which writes one file per tab |

## Backlog items this milestone touches

| Item | What happens |
|---|---|
| `export-a-document-as-markdown-with-its-pictures.md` | closed by this milestone |
| `read-pictures-and-drawings.md` | the bytes route lands here; the floating-picture placeholder lands here; `read` gains links and numbering |
| `publish-a-paired-note-again.md` | closed, in the shape decision 6 states |
| `m8-alignment-and-the-align-skill.md` | the skill lands here without a diff command; the hub-wide live session and `sergi/go-diff` stay deferred, and the item is rewritten to hold only those |
| `escaping-across-run-boundaries.md` | a blocker, fixed in this milestone |
| `publish-skill-renders-an-svg-picture.md` | the skill section lands here, unchanged |
| `relative-link-to-another-note-is-dead.md`, `bookmark-only-linked-headings.md` | unchanged; export cannot restore a Related link the note published as words, and the merge keeps the note's own links |

## Scenarios

Twenty-three scenarios, written by the product owner on 2026-09-19 against this
file and read by Nail the same day. They are the acceptance list: the plan
names the scenario each task serves, and a scenario no task serves is a hole.

gdoc publishes a note from the hub as a Google Doc. Export is the reverse.
`gdoc export <url> --out <path>` reads one Doc and writes one Markdown file per tab. Its pictures land as PNG files in `assets/` beside it.
The file holds everything the Doc says, marked the way `gdoc read` marks it. A pending suggestion is `{+words+}[s:ID]` or `{-words-}[s:ID]`. A comment anchor is `[[c:ID]]words[[/c]]`.
Export changes nothing in Drive, replaces nothing on disk, and decides nothing. A taken name takes the next free number.
Two skills are the judgement. `/gdoc-export` brings a Doc with no note into the hub and resolves the markers with you. `/gdoc-align` runs export on a paired note, compares the copy with the note, and merges with your agreement.
It resolves the markers, removes the comment anchors, and never deletes what only the note holds. It checks the merged whole for logic breaks, then deletes the copy.
It also runs the other way. It reads the note and the Doc and proposes the note's changes into the Doc as suggestions.
No side is the source of truth by rule. You decide per run. The `gdoc:` block records dated facts only.
The block is a list, `documents`, one entry per document the note has met. Publishing a paired note again makes a new document and appends an entry.
Every skill runs `gdoc help` first; the scenarios leave that out. A marked file may stay in the hub as long as you like. Only `publish`, `build` and a writer that takes its text refuse it.

### Getting a document into the hub

#### 1. A Doc and no note

**Who and where they start.** A colleague has a Doc link. The hub has no note for it.

**What they do.** "Bring this into the hub" with the link.
- Skill `gdoc-export`, which asks for the file name.
- `gdoc export <url> --out <name>.md`
- Skill: lists suggestions and threads, asks which suggestions the note keeps, resolves the markers, removes the anchors, adds `title`.

**What they see.** The text, the pending suggestions by id and author, the open threads, the pictures, the stripped prelude pieces.

**What changed.** Outside the block: `<name>.md` body and `title`, and `assets/<name>-1.png` onward. In the block of `<name>.md`: one entry with `id` and `exported: {at}`. Drive: nothing.

**What they do next.** Edit the note like any other.

#### 2. A note already paired, the Doc edited by others

**Who and where they start.** The note was published. Colleagues changed the Doc in the browser.

**What they do.** "Align this note with its document."
- Skill `gdoc-align`.
- `gdoc export <url> --out <note>.md`: the path holds the note, so the copy lands at `<note>.2.md`.
- Skill: shows the Doc's additions, merges what is agreed, resolves the markers, deletes the copy.

**What they see.** The differences section by section, one question each. Nothing only in the note is offered for deletion.

**What changed.** In the block of `<note>.md`: `exported: {at}` on this document's entry, stamped by export. Outside the block: `<note>.md` body where agreed; `<note>.2.md` written, then deleted. Drive: nothing.

**What they do next.** Nothing, or scenario 11.

#### 3. Both edited

**Who and where they start.** The note moved in the hub. The Doc moved in the browser.

**What they do.** "Align this note with its document."
- Skill `gdoc-align`, both directions, document first.
- `gdoc export <url> --out <note>.md`, landing at `<note>.2.md`.
- Skill: merges what one side changed, shows what both changed, then names every logic break in the merged whole.
- `gdoc read <url>`
- `gdoc propose <url> --from <file> --folder <test folder> --md <note>.md`, once per paragraph to push.

**What they see.** The conflicts, one question each. Then the logic breaks, such as a term defined one way and used another.

**What changed.** Outside the block: `<note>.md` body where agreed; `<note>.2.md` deleted. In the block of `<note>.md`: `exported: {at}` and `proposals` on the entry. Drive: suggestions with 🤖 comments.

**What they do next.** The Doc owner accepts or rejects.

#### 4. A picture somebody added in the Doc

**Who and where they start.** A paired note. A colleague pasted a chart into the Doc.

**What they do.** "Align this note with its document."
- Skill `gdoc-align`.
- `gdoc export <url> --out <note>.md`: export hashes each picture against the note's pictures; this one matched nothing.
- Skill: offers the picture line for the note.

**What they see.** `assets/<note>-1.png` with `matched` empty, and the line `![](assets/<note>-1.png)` offered.

**What changed.** Outside the block: `assets/<note>-1.png`, one line in `<note>.md` when agreed, the copy deleted. In the block of `<note>.md`: `exported: {at}`. Drive: nothing.

**What they do next.** Rename the PNG if wanted; fix the link.

#### 5. A picture that was an SVG in the note

**Who and where they start.** The note links `diagram.png`, rendered from `diagram.svg`. The Doc holds those bytes.

**What they do.** "Align this note with its document."
- Skill `gdoc-align`.
- `gdoc export <url> --out <note>.md`: the bytes matched `diagram.png`, so `matched` names it and no file is written.
- Skill: nothing to merge for this picture.

**What they see.** The copy links `diagram.png`. A replaced picture is scenario 4. Until measurement 2 has passed, the match is off: the picture is written as `assets/<note>-1.png` and the reply says the match is not yet trusted.

**What changed.** Nothing for this picture, outside or inside the block. Drive: nothing. The SVG stays the master.

**What they do next.** Nothing.

#### 6. A Doc with pending suggestions from a person

**Who and where they start.** A reviewer left suggestions in the Doc. Nobody accepted them.

**What they do.** "Align this note with its document."
- Skill `gdoc-align`.
- `gdoc export <url> --out <note>.md`: each suggestion as `{+ +}` or `{- -}` with its id.
- Skill: proposes the text a reader sees today and asks which suggestions the note takes.

**What they see.** The list by id and words, one question per suggestion. No author: the Docs API never says who wrote a suggestion.

**What changed.** Outside the block: `<note>.md` body, the chosen text. In the block of `<note>.md`: `exported: {at}`. Drive: nothing accepted or rejected.

**What they do next.** Accept or reject in the browser.

#### 7. A Doc with gdoc's own pending proposals

**Who and where they start.** A review session earlier proposed three changes from the note into the Doc, still pending. The note holds those words; its block lists the ids under `proposals`.

**What they do.** "Align this note with its document."
- Skill `gdoc-align`.
- `gdoc export <url> --out <note>.md`: the copy shows the three as `{+words+}[s:ID]`. The envelope says three pending, all gdoc's own.
- Skill: reads the ids in `proposals`, sees the note has the words, keeps them, asks nothing.

**What they see.** One line: the three are gdoc's own, still waiting in the Doc. No question.

**What changed.** Nothing outside the block for those paragraphs. In the block of `<note>.md`: `exported: {at}`; `proposals` unchanged. Drive: nothing.

**What they do next.** Nothing. The Doc owner accepts or rejects.

#### 8. A Doc with open comment threads

**Who and where they start.** Colleagues wrote comments in the Doc. The threads are open.

**What they do.** "Align this note with its document."
- Skill `gdoc-align`.
- `gdoc export <url> --out <note>.md`: the copy wraps each comment's words as `[[c:ID]]words[[/c]]`. The comment text stays out.
- `gdoc comments <url>`, for author and first line per thread.
- Skill: removes the `[[c:ID]]` marks, keeps the words, lists the threads.

**What they see.** The words stay. A list of threads by id, author and first line. The comments stay in Drive.

**What changed.** Outside the block: `<note>.md` body, without the marks. In the block of `<note>.md`: `exported: {at}`. Drive: nothing.

**What they do next.** Run `/gdoc-review` on the marked threads.

#### 9. A Doc with a house prelude

**Who and where they start.** Three Docs. One made by `publish` opens with the cover, three tables, the legend and the contents list. One restyled carries the same under gdoc's named range. One gdoc never touched has no cover.

**What they do.** "Bring this into the hub" or "Align this note."
- Skill `gdoc-export` when there is no note, `gdoc-align` when there is.
- `gdoc export <url> --out <path>`: strips the cover pieces by position and lists each. An unrecognised cover stays, with a warning.
- Skill: shows anything left over before merging.

**What they see.** The copy opens at the first body heading, and the reply lists the stripped pieces. Third Doc: nothing stripped.

**What changed.** None of the author's own keys change. Outside the block: the copy body, from the first heading. In the block: `exported: {at}` only. Drive: nothing.

**What they do next.** Nothing.

### Getting hub changes into a document

#### 10. A note and no Doc

**Who and where they start.** A note in the hub with `title`. No `gdoc:` block.

**What they do.** `/gdoc-publish <note> into <folder url>`.
- Skill `gdoc-publish`.
- `gdoc publish --md <note>.md --folder-id <id>`
- Skill: reads `verified`, names the three things to look at.

**What they see.** The URL, `verified` with three checks, three things to look at.

**What changed.** In the block of `<note>.md`: `schema: 2` and one entry with `id`, `folder_id` and `published: {at, title, house}`. Outside the block: nothing. Drive: a new document in the folder.

**What they do next.** Share the URL.

#### 11. A paired note edited in the hub

**Who and where they start.** Colleagues read the Doc. The note gained two paragraphs and lost one.

**What they do.** "Propose the note's changes into the document."
- Skill `gdoc-align`, note into document.
- `gdoc read <url>`
- Skill: composes the difference, one entry per changed paragraph.
- `gdoc propose <url> --from <file> --folder <test folder> --md <note>.md`

**What they see.** One suggestion per paragraph with a 🤖 comment, `verified` per proposal. The lost paragraph arrives as a suggested deletion.

**What changed.** In the block of `<note>.md`: `proposals` on this document's entry. Outside the block: nothing. Drive: suggestions and comments.

**What they do next.** The Doc owner accepts or rejects.

#### 12. Export, edit, publish to a new document

**Who and where they start.** Scenario 1 happened. The colleague rewrote the note and wants a fresh house-style document.

**What they do.** `/gdoc-publish <note> into <folder url>`.
- Skill `gdoc-publish`: says once that a new document is made and the old one goes stale.
- `gdoc publish --md <note>.md --folder-id <id>`

**What they see.** The new URL and the checks. A body still carrying a marker is refused by line.

**What changed.** In the block of `<note>.md`: a second entry appended, with `id`, `folder_id` and `published`. The first entry is untouched. Outside the block: nothing. Drive: a second document. The old one keeps its threads and URL.

**What they do next.** Tell colleagues which URL is current.

#### 13. Export, edit, propose into the same document

**Who and where they start.** Scenario 1 happened. Three sentences changed; the same URL should follow.

**What they do.** "Publish my changes into the document."
- Skill `gdoc-align`: reads this as a merge, note into document.
- `gdoc read <url>`
- `gdoc propose <url> --from <file> --folder <test folder> --md <note>.md`

**What they see.** Three suggestions with 🤖 comments. The Doc has no house style; `/gdoc-restyle` is the route.

**What changed.** In the block of `<note>.md`: `proposals` on the entry. Outside the block: nothing. Drive: three suggestions.

**What they do next.** Accept them in the browser.

#### 14. A house-style copy of a Doc they only have the link for

**Who and where they start.** A colleague wants the Altery look on somebody's Doc, as a new file.

**What they do.** "Bring this into the hub", then `/gdoc-publish <note> into <folder url>`.
- Skill `gdoc-align`: `gdoc export <url> --out <name>.md`, markers resolved, `title` added.
- Skill `gdoc-publish`: `gdoc publish --md <name>.md --folder-id <id>`.

**What they see.** Scenario 1, then a new URL. The copy has none of the original's threads; `/gdoc-restyle` keeps them in place.

**What changed.** Outside the block: `<name>.md` and `assets/`. In the block of `<name>.md`: two entries, the original with `exported`, the new one with `published`. Drive: one new document; the original untouched.

**What they do next.** Say which document colleagues should read.

### Things that go wrong

#### 15. A session dies between export and merge

**Who and where they start.** `<note>.2.md` sits beside the note, its entry carrying `exported: {at, note: <note>.md}`. The session that made it died.

**What they do.** "Align this note with its document" in a new session.
- Skill `gdoc-align`: finds `<note>.2.md`, says when it was made, deletes it.
- `gdoc export <url> --out <note>.md`: a fresh `<note>.2.md`.
- Skill: the merge, as in scenario 2.

**What they see.** One line about the stale copy and its date, then the merge questions.

**What changed.** Outside the block: stale copy deleted, fresh copy written then deleted, `<note>.md` body where agreed. In the block of `<note>.md`: `exported: {at}` stamped again. Drive: nothing.

**What they do next.** Nothing.

#### 16. An export run twice

**Who and where they start.** `<note>.2.md` from a run by hand is still there.

**What they do.** `gdoc export <url> --out <note>.md` by hand again, no skill.

**What they see.** `<note>.md` and `<note>.2.md` are taken, so the copy is `<note>.3.md`. Pictures take the next free number.

**What changed.** Outside the block: `<note>.3.md` and its PNGs. In the block of `<note>.md`: `exported: {at}` stamped again. Drive: nothing.

**What they do next.** Delete the extra copies, or let `/gdoc-align` do it: it names both, exports fresh, deletes each copy whose body matches, and asks about one that does not.

#### 17. A live review session while an export happens

**Who and where they start.** One session watches the Doc with `/gdoc-review live`. Another exports it.

**What they do.** "Align this note with its document."
- Skill `gdoc-align`.
- `gdoc export <url> --out <note>.md`
- Skill: merges into `<note>.md`; the live session may write `proposals` into the same block.

**What they see.** The merge as usual; nothing is stopped. The binary's block writes read the file fresh and replace it in one rename. The skill's edits refuse a file changed since it was read, and the skill re-reads.

**What changed.** Outside the block: `<note>.md` body, the copy deleted. In the block of `<note>.md`: `exported: {at}` from this session, `proposals` from the other. Drive: nothing.

**What they do next.** Nothing.

#### 18. A colleague on an old binary opens an exported note

**Who and where they start.** The note carries `schema: 2`. Their gdoc predates it.

**What they do.** Any command with `--md <note>.md`.

**What they see.** One line from the old binary: the `gdoc:` block in `<note>.md` carries a key it does not read, `documents`. The daily notice in `gdoc help` already says a newer release exists.

**What changed.** Nothing, outside or inside the block.

**What they do next.** `gdoc update`, then run again.

### Things at the edge

#### 19. A Doc with two tabs

**Who and where they start.** The Doc has a second tab named Appendix.

**What they do.** "Bring this into the hub."
- Skill `gdoc-export`.
- `gdoc export <url> --out <name>.md`: two files, `<name>.md` and `<name>-appendix.md`, pictures following their tab.
- Skill: resolves the markers in both.

**What they see.** Two files, one per tab. The second tab's note can be published as a new document. It cannot propose into its tab yet; every writer still refuses two tabs.

**What changed.** Outside the block: two files and `assets/`. In the block of each file: one entry with `id`, `tab_id` and `exported`. Drive: nothing.

**What they do next.** Edit either note.

#### 20. An `ai!` comment asking for an export

**Who and where they start.** A colleague writes `ai! export this to the hub` in the Doc during a review session.

**What they do.** Nothing; the review session reads it.
- Skill `gdoc-review`.
- `gdoc comments <url>`
- `gdoc reply <url> <comment id> --body-file <file>`

**What they see.** A 🤖 reply saying export starts from a session by name, never from a comment.

**What changed.** Nothing outside or inside the block. Drive: one reply.

**What they do next.** Ask a session: "Bring this into the hub."

#### 21. Replace the old note by hand

**Who and where they start.** A paired note is out of date. The colleague wants the Doc's text as the whole note.

**What they do.** "Export this document. I will replace the note myself."
- Skill `gdoc-align`.
- `gdoc export <url> --out <note>.md`, landing at `<note>.2.md`.
- Skill: resolves the markers in `<note>.2.md` with the person, then stops.

**What they see.** `<note>.md` was taken, so the copy is `<note>.2.md`. No flag replaces a note.

**What changed.** Outside the block: `<note>.2.md`. Then by hand: the old note moved away, the copy renamed to `<note>.md`. In the old note's block: `exported: {at}`. In the copy's block: `note: <note>.md`, which every writer refuses until removed. Drive: nothing.

**What they do next.** Remove the `note:` line under `exported` from the renamed file, by hand. The refusal says exactly that.

#### 22. No time to resolve the markers today

**Who and where they start.** Scenario 1, and the person has ten minutes.

**What they do.** "Export it. I will look later."
- Skill `gdoc-export`.
- `gdoc export <url> --out <name>.md`
- Skill: adds `title` and stops.

**What they see.** The file with its markers and the counts. One line: the markers are gdoc's, any later session can resolve them. Only `publish`, `build` and a writer that takes its text refuse the file.

**What changed.** Outside the block: `<name>.md` with markers, `assets/`. In the block of `<name>.md`: one entry with `exported: {at}`. Drive: nothing.

**What they do next.** On another day: "Resolve the markers in `<name>.md`." The skill runs `gdoc suggestions <url> --md <name>.md` first, resolves what the Doc has decided since without asking, asks per suggestion still pending, and removes the anchors, with no new export.

#### 23. A note published before this milestone

**Who and where they start.** The note was published in August. Its block is `schema: 1` with one `document_id`. Every note on the team looks like this today.

**What they do.** "Align this note with its document." Or any command that writes the block, such as `gdoc suggestions --md`.
- Skill `gdoc-align`.
- `gdoc export <url> --out <note>.md`: reads the block as a one-entry list, stamps `exported: {at}` on that entry, and writes the block as `schema: 2`.
- Skill: the merge as in scenario 2.

**What they see.** One extra line in the reply: the block in `<note>.md` was rewritten to schema 2. Every colleague on an older binary is now in scenario 18 for this note.

**What changed.** In the block of `<note>.md`: `schema: 2`, `documents` with one entry carrying what the old block carried, plus `exported: {at}`. Nothing outside the block that scenario 2 does not change. Drive: nothing.

**What they do next.** Tell colleagues to run `gdoc update` if they work on this note.

### Summary

| No. | Scenario | Skill | Commands | Written | Inside the block |
|---|---|---|---|---|---|
| 1 | Doc, no note | export | `export` | `<name>.md`, `assets/` | `exported` |
| 2 | Doc edited by others | align | `export` | note body | `exported` |
| 3 | Both edited | align | `export`, `read`, `propose` | note body, suggestions | `exported`, `proposals` |
| 4 | Picture added in Doc | align | `export` | `assets/<note>-1.png`, one line | `exported` |
| 5 | SVG picture | align | `export` | nothing, link kept | `exported` |
| 6 | Person's suggestions | align | `export` | note, chosen text | `exported` |
| 7 | gdoc's own proposals | align | `export` | nothing | `exported` |
| 8 | Open threads | align | `export`, `comments` | note without marks | `exported` |
| 9 | House prelude | export or align | `export` | copy from first heading | `exported` |
| 10 | Note, no Doc | publish | `publish` | new document | `published` |
| 11 | Note edited in hub | align | `read`, `propose` | suggestions | `proposals` |
| 12 | Publish again | publish | `publish` | new document | entry appended |
| 13 | Propose into same Doc | align | `read`, `propose` | suggestions | `proposals` |
| 14 | House-style copy | align, publish | `export`, `publish` | note, new document | two entries |
| 15 | Session died | align | `export` again | note body | `exported` again |
| 16 | Export twice | none | `export` | `<note>.3.md` | `exported` again |
| 17 | Live review running | align, review | `export`, `propose` | note body | `exported`, `proposals` |
| 18 | Old binary | any | any `--md` | nothing | nothing |
| 19 | Two tabs | export | `export` | one file per tab | `tab_id` |
| 20 | `ai!` export | review | `comments`, `reply` | one reply | nothing |
| 21 | Replace by hand | align | `export` | `<note>.2.md` | `exported`, `note:` |
| 22 | No time today | export | `export` | marked file kept | `exported` |
| 23 | Note from before this milestone | align | `export` | note body | block rewritten to schema 2 |

## Out of scope

- A diff command in the binary, and `sergi/go-diff`. The skill reads two
  files.
- The hub-wide live session.
- Tracing a PNG back to an SVG. Google never gives a vector back.
- Any request that changes a document from export.
- A `--force` flag on export, or any flag that replaces a file.
- Windows. The SVG route's Linux and Windows lines stay in the backlog item
  until `release/platforms` grows.
