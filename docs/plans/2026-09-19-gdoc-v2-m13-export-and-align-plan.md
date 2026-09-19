# gdoc v2 Milestone 13: export and align, plan

2026-09-19. The task list for the specification at
`docs/plans/2026-09-19-gdoc-v2-m13-export-and-align.md`. The spec holds the
seventeen decisions Nail took, the shapes, and the twenty-three scenarios that
are the acceptance list. This file holds the order the work lands in, one
commit per task, for ralphex. A task names the decision and the scenarios it
serves. A task that finds the spec wrong stops and says so rather than
choosing.

## Principles

Serves: 3, uncertainty never resolves toward the destructive answer. Nothing
in this plan overwrites a file: export takes the next free number and has no
flag that could replace one (Task 9). Every change the skill carries into a
document is a suggestion, and the binary gains no request kind (Task 10
sends none). Also 2: the pairing stays in the note's front matter, now as a
list (Task 2), and pictures land in `assets/` beside the note, nothing hidden.
And 4: `read` gains link targets and numbering (Tasks 5 and 6), so a session
reads less machinery, and the markers stay gdoc's own so a person never
translates them.

Strains: none. Two decisions are superseded by Task 13's DECISIONS.md entry,
the markdown as the source of truth and publish running once, both Nail's
call on 2026-09-19.

## Overview

| Piece | Today | After |
|---|---|---|
| the `gdoc:` block | one `document_id`, schema 1 | `documents`, one entry per document, schema 2; schema 1 reads and stays byte-identical until a write changes it |
| a paired note and `publish` | refused | a new document, one more entry |
| a marker in a note body | `build` prints `{+` as prose; `propose --from` would send it | `build`, `publish`, `propose`, `annotate` and `reply` refuse it by line |
| `read` | chip labels, bullets for every list, `[image]`, silence for a floating picture | `[words](target)`, `1.` numbering, a placeholder for a floating picture |
| a Google Doc into the hub | nothing | `gdoc export <url> --out <path>`: one file per tab, PNGs in `assets/`, the block, the stamp |
| the skills | review, publish, restyle | plus export and align; publish asks on a paired note; `needs: v2.4.0` on four |
| the command table | fourteen rows | fifteen |

## Decisions

In the spec, numbered 1 to 17 under "Decisions Nail took, 2026-09-19". This
plan repeats none of them. Each task below names the ones it serves.

## Context (from discovery)

- **The block.** `go/internal/frontmatter/schema.go` holds `Block` with
  `DocumentID`, `FolderID`, `Published`, `SuggestionsSeen`, `Proposals` and
  `Schema = 1`; `Validate` at `:84` names the first wrong key. `Read` at
  `frontmatter.go:50`, `Write` at `:129`, `marshal` at `:178` and `render` at
  `:207` render the whole block from the struct; `verify` at `:231` re-parses
  what was written. `TestWriteAnUnchangedBlockIsByteIdentical` at
  `frontmatter_test.go:148` is the pin the migration must keep.
- **Who reads the block.** `cmd/gdoc/read.go:769-786` reads it for
  `suggestions --md`, refuses another id at `:776`, and writes the snapshot at
  `:786`. `cmd/gdoc/write.go:406` reads it for `propose` and `withdraw`.
  `cmd/gdoc/publish.go:160` is `unpaired`, `:199-215` is `pair`, which writes
  the first block. `TestPublishRefusesANoteThatIsAlreadyPaired` at
  `publish_test.go:214`, `TestProposeRefusesANoteThatNamesAnotherDocument` at
  `write_test.go:556` and `TestSuggestionsRefusesAFilePairedWithAnotherDocument`
  at `read_test.go:470` are the shapes the new refusals take.
- **The Docs walk.** `go/internal/docs/walk.go`: `rawParaElement` at `:186`
  carries `inlineObjectElement` and a `bullet` field at `:177` that nothing
  reads into `Paragraph.Bullet` beyond the level; `run` at `:390` reads no
  `textStyle.link`; `blocks` at `:351` handles `paragraph` and `table` and
  drops `tableOfContents`; nothing reads `positionedObjectIds` or the
  `positionedObjects` map; `objectKind` at `:473` names image, drawing,
  equation, object. `docs.go:55-127` are the exported types; `Tab` at `:66`.
  The fixtures under `docs/testdata/` are recorded live reads.
- **The projection.** `go/internal/view/text.go`: `Text` at `:100`,
  `Structure` at `:117`, `prefix` at `:295` prints `- ` for every list,
  `mark` at `:69` prints `[image]`, `writeText` at `:531` and `escapeAt` at
  `:553` hold the escaping, one run at a time, which is the across-runs gap
  in `docs/backlog/escaping-across-run-boundaries.md`;
  `TestEscapingLeavesNoMarkerBehindWhenTwoOverlap` at `text_test.go:504`.
  Five goldens under `view/testdata/`.
- **The docx export.** `go/internal/docx/docx.go`: `ExportURL` at `:118`,
  `Export` at `:128`, `Parse` at `:140` reads `word/comments.xml` and
  `word/document.xml` through `part` at `:185` with a size limit; nothing
  reads `word/media/` or `word/_rels/document.xml.rels`. The guard admits
  `{id}/export` at `policy.go:627` and refuses only PDF at `params.go:271`.
- **Text into a document.** `go/internal/plaintext/plaintext.go`:
  `Markdown` at `:90` over the `markdown` pattern at `:86`; called from
  `reply.go` and `annotate.go` on the words a session wrote. `propose` reads
  its `--from` file in `cmd/gdoc/write.go` and never passes it through
  `plaintext`.
- **The body.** `go/internal/body/body.go`: `bookmarkName` at `:357` is the
  slug rule the tab file name reuses; `headingAnchors` at `:380`;
  `body/image.go` resolves a picture against the note's directory and reads
  PNG and JPEG only. `body/doc.go` says footnotes and HTML are refused or
  dropped. Six notes under `body/testdata/` with goldens.
- **The command table.** `go/cmd/gdoc/commands.go:147` `commands()` with
  fourteen rows; `help`, completion and the usage tests read it
  (`TestTheUsageLineMarksWhatIsOptionalAndWhatIsAnAlternative` at
  `help_test.go:274` spells every line out). `build` at `:281` uses `--out`.
- **The skills.** `skills/gdoc-{review,publish,restyle}/SKILL.md` with
  `name`, `description`, `needs: v2.0.0`. `install.sh:29` lists the three,
  `release/install.sh` copies them, `.claude-plugin/plugin.json` describes
  them, `cmd/gdoc/skills_test.go` reads them.
- **Live tests.** `go/internal/live/doc.go` names the variables;
  `live_test.go:395` `trashSubject`; `GDOC_LIVE_RECORD` saves a Docs read into
  testdata. `docs/v2/MEASURED.md` ends with "Not measured yet" at `:342`.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it
  to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own
  commit, and each ends with the test gate as its last checkbox.
- **CRITICAL: every task MUST include new or updated tests**, success and
  failure paths both.
- **CRITICAL: all tests must pass before the next task starts.** `make test`
  runs `-race`; keep it there.
- **CRITICAL: the guard does not move.** No task touches `go/internal/guard/`
  except to add the one test in Task 10 that export opens only what `read`
  opens. No new host, no new grant, no new request kind.
- **CRITICAL: nothing is overwritten.** No task writes a file over an existing
  one except through `atomicfile.Replace` on a note whose block it is
  stamping. There is no `--force` on export.
- **CRITICAL: facts only in Go.** Export decides nothing: no field says
  whether a difference matters, which side is right, or whether a suggestion
  should be taken.
- **CRITICAL: the schema 1 pin stays.** `TestWriteAnUnchangedBlockIsByteIdentical`
  keeps passing on a schema 1 block read and written unchanged.
- **CRITICAL: a test states its value as a literal**, never reading the
  constant it checks.
- **CRITICAL: the backlog file is `git rm`'d in the commit that lands its
  fix**, named in the task.
- **CRITICAL: new tests go in new files** where the existing test file is
  over 800 lines (`write_test.go`, `read_test.go`, `body_test.go`).
- **CRITICAL: no doc.go rule without its test named.** Every package comment
  paragraph a task adds names the test that pins it.
- **CRITICAL: update this plan file when scope changes during implementation.**
- No em dashes in anything this plan produces. Plain English.

## Testing Strategy

- **Unit, `frontmatter`**: schema 1 reads as one entry and writes back
  byte-identical when unchanged; a changing write renders schema 2 with a
  `documents` list; every entry field round-trips; an unknown key under an
  entry is named; `exported.note` reads and writes; `tab_id` reads and writes;
  a block with two entries naming one id is refused.
- **Unit, `cmd/gdoc`**: every writer acts on the entry the URL names and
  refuses a URL the list does not hold; a proposal is recorded under its own
  document; `withdraw` never sends an id from another document; `GoneSince`
  reads the snapshot of the document read; every writer refuses a file whose
  entry carries `exported.note`, naming the note; `publish` accepts a paired
  note and appends; the reply carries the schema 2 line once; `export`'s
  usage line, flags, envelope, refusals and files.
- **Unit, `docs` and `view`**: links with url, heading and bookmark targets;
  bullets with glyph and level from the lists map; a `tableOfContents`
  element; a positioned object per paragraph; goldens for `[words](target)`,
  `1.`, the placeholder; escaping across two runs leaves no marker.
- **Unit, `export`**: golden Markdown per fixture (Docs JSON plus docx zip)
  with expected PNG hashes; the prelude strip for a publish document, a
  restyle document and none; an edited cover; a count mismatch; the numbering
  of files and pictures; the stamp; the door checks; two tabs.
- **Unit, `body` and `plaintext`**: a marker line is refused by `build` and
  by `propose --from`; `annotate` and `reply` refuse a marker.
- **Boundary**: all green; `TestNoThirdPartyDependencies` unchanged, no
  module added.
- **Live, opt-in**: the three measurements; publish, export, hash equality;
  propose then export carries the id; the round trip over the six body notes
  against a named list of known differences.
- Coverage standard: every changed function has a test; no package below 80%.

## Validation Commands

- `cd go && go test -race ./...`
- `cd go && test -z "$(gofmt -l .)" && go vet ./...`
- `make build`, then `bin/gdoc help export` prints the usage line
  `export <url> --out <file>` and names `assets`, the numbering rule and the
  block.
- `bin/gdoc help | grep -c '^  '` prints one more line than on main.
- `ls skills/` prints five folders, and `ls skills/gdoc-export/` prints
  `SKILL.md` and `markers.md`.
- `grep -c 'needs: v2.4.0' skills/*/SKILL.md` prints four ones and one zero
  (restyle).
- `ls docs/backlog/ | wc -l` prints 20, four fewer than the 24 on main.
- `wc -l CLAUDE.md` under 300.
- With `GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1 GDOC_LIVE_FOLDER_ID=<folder>` and
  `GDOC_LIVE_EXPORT_DOC_ID=<the fixture document>`:
  `cd go && go test -race -run 'TestLiveExport' ./internal/live/` passes and
  prints the three measurements.

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.

## Solution Overview

**Task 1** writes the live measurement test and nothing else, so Nail can run
it early. It reads one hand-made fixture document holding an inline picture, a
Google Drawing, a floating picture and a second tab, through both routes, and
prints body order on each side, the PNG bytes' hash against a known upload,
and what the docx carries for the second tab. Its three answers are MEASURED.md
rows Task 13 writes.

**Task 2** changes the block and its readers before any new command
exists: the `documents` list, the per-entry facts, the schema 1 read, the URL
rule in every writer, the `exported.note` refusal, and `publish` appending.
After Task 2 the binary is one that a colleague could ship without export.

**Task 4** closes the marker door in the other direction, so nothing later in
the plan can push a marker into a document.

**Tasks 5 and 6** teach `docs` and `view` what export needs and `read` was
missing: links, numbering, the contents element, positioned objects, and the
across-runs escaping.

**Tasks 7 to 10** are the export: the Markdown projection with the prelude
stripped and pictures placed (8), the picture bytes from the docx matched by
order with the hash match behind a switch (9), the files and the block on disk
(10), and the command row with its envelope (11).

**Task 11** is the live round trip, written for Nail to run.

**Task 12** is the five skills. **Task 13** is the documents, the decisions
and the backlog. **Tasks 14 and 15** close.

## Technical Details

**The block, schema 2.** `Block{Schema int; Documents []Entry}` with
`Entry{ID, FolderID string; Published *Published; Exported *Exported;
SuggestionsSeen *SuggestionsSeen; Proposals []Proposal; TabID string}` and
`Exported{At time.Time; Note string}`. `Read` accepts schema 1 by decoding
the old shape into one `Entry` and remembering, on the returned value, that
the source was schema 1. `Write` on a block that `reflect.DeepEqual`s its
source's decoded value writes the source bytes back, which is how the schema
1 pin keeps holding; any other write renders schema 2. A helper
`(*Block).Entry(id string) (*Entry, error)` returns the entry the URL names or
an error naming the ids the block holds. `Validate` refuses two entries with
one id, a `tab_id` that is not a Drive id shape, and `exported` with neither
field.

**The URL rule.** Every place that compared `block.DocumentID` with the URL's
id calls `block.Entry(id)` and works on that entry. `pair` in `publish.go`
appends. The snapshot write in `read.go` and the proposal write in `write.go`
write into the entry. A refusal on `exported.note` is one check in a shared
helper in `cmd/gdoc`, `paired(path, src, id) (*Block, *Entry, error)`, that
every `--md` reader calls.

**The marker refusal.** One exported function `markers.Find(line string)
(string, bool)` in a new tiny package `internal/markers` that both `body` and
`plaintext` and `propose` call, holding the four openers `{+`, `{-`, `[[c:`,
`[s:` and their closers as literals; `TestMarkersNamesEachOpener` pins them.
`body` refuses a line carrying one the way it refuses a footnote, naming the
line. `plaintext.Markdown` gains the check. `propose` checks the `--from`
file's quote and replacement.

**Docs decoding.** `Run` gains `Link *Link` with `URL, HeadingID, BookmarkID,
TabID string`; `Paragraph` gains `HeadingID string` and `Positioned []string`;
`Bullet` gains `ListID string`, `Ordered bool` and `Glyph string` read from
`lists[id].listProperties.nestingLevels[level].glyphType` where a glyph type
other than `GLYPH_TYPE_UNSPECIFIED` and empty means ordered; a new
`Block.TOC *TOC` for `tableOfContents` with its content walked like a body;
`Tab` gains `Positioned map[string]Object`. `view` prints a link as
`[words](target)` where target is the URL, `#<heading words slugged>` for a
heading id or bookmark, and the label alone when the target is a tab; a
numbered item as `N. ` counted per list id and level; a positioned object as
`<!-- picture: floating, <id> -->` after its paragraph. Escaping across runs:
`writeText` takes the previous run's last rune and escapes the pair by
writing a backslash before the second half, Nail's convention choice recorded
in the DECISIONS entry of Task 13.

**Export projection.** `view`'s emitter is unexported and `writeText`
escapes `[`, so `view` gains one exported entry beside `Text`:
`Project(d *docs.Document, o Options) (string, []string)` with
`Options{Picture func(objectID string) string; Skip func(docs.Block) bool}`.
A non-nil `Picture` result is written raw, on its own line, never through
`writeText`; `Skip` drops a block before it is projected. `Text` is
`Project(d, Options{})`, so the five goldens do not move. `internal/export`
calls `Project` over the tree with the prelude blocks skipped and every
picture object named. Strip: for a document whose
`NamedRanges` carry `prelude.MarkerName`, remove the blocks inside that range;
otherwise, when the blocks before the first `HEADING_1` paragraph hold three
or more tables and a `TOC` block, remove from the first block to the `TOC`
block inclusive; otherwise strip nothing. The removed blocks' plain text goes
on the result as `Stripped []Piece{Kind, Text}`. Heading numbers: a heading
whose text starts with digits and a hyphen loses that prefix, recorded as a
piece. Footnotes as `view` prints them, with the warning.

**Pictures.** `docx` gains `Media(f *File) []MediaFile{Name string; Bytes
[]byte}` in body order: walk `word/document.xml` for `a:blip r:embed` in
order, map each through `word/_rels/document.xml.rels` to `word/media/<n>`,
read the part under the existing size limit. `export` pairs the k-th media
file with the k-th object in the Docs tree, inline objects first in body
order and positioned objects after their paragraph, and stops matching on a
count mismatch. The hash match: `sha256` of each media file against `sha256`
of every picture the note at `--out` links, found by parsing the note with
goldmark and resolving as `body/image.go` does; behind a field
`Options.MatchByHash`, never a package variable, which every caller in this
milestone passes as `false`. Nail sets it to `true` by hand, in a commit of
its own after the merge, when measurement 2 says the bytes are equal, with
the MEASURED.md row in the same commit. The reply names the state either way.

**Files on disk.** `export.Write(dir, stem string, files []TabFile, pics
[]Picture) (Written, error)`: for each tab file, the free path or the next
number; for each picture, `assets/<stem>-<n>.png` from the next free `n`;
`assets/` created with 0755; every write through `atomicfile.Replace` to a
path that does not exist. The stamp: when `--out` holds a note whose entry
matches, `frontmatter.Write` with `Exported.At` set on that entry. The copy's
block: one entry with `Exported{At, Note: <relative path to the note>}`.

**The command.** A row in `commands()`: `name: "export"`, a `<url>`
argument, `--out` of the file kind, no other flag; `run` in a new
`cmd/gdoc/export.go`; the envelope `exportData{Files []FileOut{Path, TabID,
TabTitle}; Pictures []PictureOut{Path, Matched string}; Pending int; Own *int;
Threads int; Stripped []Piece; SchemaRewritten bool}` plus warnings. The
refusals in the order the spec lists them.

## Implementation Steps

### Task 1: the live measurement test

Serves decision 10 and the spec's "Measurements before any matching code".
Test only; Nail runs it by hand at any point, and Task 13 writes its answers.

**Files:**
- Create: `go/internal/live/export_measure_test.go`
- Modify: `go/internal/live/doc.go`

- [x] `doc.go` names `GDOC_LIVE_EXPORT_DOC_ID`: a document Nail makes by hand
      in the test folder holding, in this order, one inline PNG picture
      uploaded from `body/testdata/`'s pictures note, one Google Drawing, one
      floating picture, and a second tab titled `Appendix` with one picture.
      The read test creates nothing.
- [x] `TestLiveExportMeasurements`, gated on `GDOC_LIVE_TEST` and the id:
      reads the Docs JSON with `includeTabsContent`, lists every inline and
      positioned object id in body order per tab; fetches the docx export,
      lists every `r:embed` in `word/document.xml` order with the media part
      name and its sha256; prints both lists side by side; prints whether the
      first media file's sha256 equals the sha256 of the uploaded PNG on
      disk; prints the count of `w:drawing` in the docx against the count of
      objects in the JSON's first tab and second tab. It fails only when the
      export or the read fails; the facts are printed, not asserted, because
      they are the measurement.
- [x] `TestLiveExportMeasurements` also saves the Docs read and the docx zip
      under `go/internal/export/testdata/fixture-measured/` when
      `GDOC_LIVE_RECORD=1`, the way the read test records.
- [x] `doc.go` also names `GDOC_LIVE_PUBLISHED_DOC_ID` and
      `GDOC_LIVE_RESTYLED_DOC_ID`: a document `publish` made and one `restyle`
      styled with `--fields`, both in the test folder. With `GDOC_LIVE_RECORD=1`
      the test saves their Docs reads under
      `go/internal/export/testdata/publish-prelude/` and `restyle-prelude/`,
      so Task 7's strip goldens rest on real documents.
- [x] `cd go && go test -race ./internal/live/` compiles and skips without the
      variables.
- [x] `git commit -m "test(live): the three export measurements, printed not asserted"`

### Task 2: the block is a list of documents, and every writer acts on the entry the URL names

Serves decisions 4, 9 and 13. Scenarios 7, 11, 15, 17, 18, 23. One commit,
because the block's new type and its readers must land together for the tree
to compile, and ralphex runs `make test` after every task.

**Files:**
- Modify: `go/internal/frontmatter/schema.go`, `go/internal/frontmatter/frontmatter.go`,
  `go/internal/frontmatter/doc.go`, `go/internal/frontmatter/testdata/` (new
  fixtures, old ones kept)
- Create: `go/internal/frontmatter/schema2_test.go`
- Create: `go/cmd/gdoc/paired.go`, `go/cmd/gdoc/paired_test.go`
- Modify: `go/cmd/gdoc/read.go`, `go/cmd/gdoc/write.go`, `go/cmd/gdoc/publish.go`,
  `go/cmd/gdoc/annotate.go`, `go/cmd/gdoc/doc.go`

The block first, in `frontmatter`:

- [x] Test first, `TestReadAcceptsASchemaOneBlockWrittenBeforeToday`: every
      existing schema 1 fixture reads as one `Entry` with the old fields in
      place. Watch it fail on the new type.
- [x] Test, `TestWriteAnUnchangedSchemaOneBlockStaysSchemaOne`: read a schema 1
      fixture, write it back unchanged, bytes identical (the existing pin,
      restated on the new type, and the old test kept).
- [x] Test, `TestAChangingWriteRewritesSchemaOneAsTwo`: read schema 1, set
      `Exported`, write; the output carries `schema: 2` and `documents:` with
      one entry, and reads back equal.
- [x] Test, `TestASchemaTwoBlockRoundTrips`: two entries, one with
      `published`, one with `exported {at, note}` and `tab_id`, each with its
      own `proposals` and `suggestions_seen`; write then read is equal; an
      unchanged write is byte-identical.
- [x] Test, `TestValidateNamesTheEntryItRefused`: two entries with one id;
      `exported` with no fields; a `tab_id` that is a path; an unknown key
      inside an entry. Each refusal names the key and the entry index.
- [x] Test, `TestEntryFindsTheDocumentTheURLNames`: `Entry("1AbC")` returns
      the entry; `Entry("nope")` returns an error naming every id the block
      holds.
- [x] Test, `TestTheShippedDecoderRefusesADocumentsKeyByName`: the sentence
      the binary on main prints on a schema 2 block, pinned as a literal by
      running the schema 1 decoder from `git show main:` in a testdata copy,
      or by reading the strict yaml error's shape: the read at
      `frontmatter.go:80` runs `yaml.Strict()` before `Validate`, so the old
      binary names the unknown field `documents` and never reaches the
      schema sentence. The literal goes into this plan as a ➕ note for Task
      13, which writes decision 9 and scenario 18 to match.
- ➕ The literal, for Task 13. The schema 1 decoder, which is what a gdoc from
      before this milestone runs, refuses a schema 2 block with
      `gdoc front matter: [3:3] unknown field "documents"` followed by the
      four lines of YAML it was reading, the third marked. It never reaches
      the schema sentence, so decision 9 and scenario 18 say the old binary
      names the key rather than the version.
- [x] Implement `Entry`, `Exported`, the schema 1 decode into one entry, the
      source-equality check in `Write`, `Schema = 2`, the `Validate` rules.
- [x] `doc.go`: a section "The block is a list, and schema 1 still reads",
      naming the tests above; the "publish is its only writer" sentence goes.

Then the readers, in `cmd/gdoc`:

- [x] Test, `TestEveryWriterActsOnTheURLAndRefusesAnIDOutsideTheList`:
      a note with two entries; `suggestions --md`, `propose`, `withdraw` and
      `annotate` each succeed on either URL and refuse a third, naming both
      ids.
- ➕ `annotate` takes no `--md`, so the three commands that read a note are
      `suggestions`, `propose` and `withdraw`, and the test covers those. The
      note's entries are written with the run's own document second, so a
      reader that took the first entry would fail. `annotate.go` is therefore
      untouched by this task, and the files list above says otherwise.
- [x] Test, `TestAProposalIsRecordedUnderItsOwnDocument`: propose on document
      B of a two-entry note; B's `proposals` grows, A's is untouched.
- [x] Test, `TestWithdrawNeverSendsAnIDFromAnotherDocument`: a proposal id
      under A; `withdraw` with B's URL refuses naming A.
- [x] Test, `TestGoneSinceReadsTheSnapshotOfTheDocumentRead`: snapshots under
      A and B; `suggestions` with B's URL compares against B's.
- [x] Test, `TestEveryWriterRefusesACopyThatNamesANote`: an entry with
      `exported.note`; every `--md` command refuses in one sentence naming
      the note path.
- [x] Test, `TestTheReplySaysOnceWhenTheBlockWasRewritten`: a schema 1 note
      through `suggestions --md`; the envelope carries one warning saying the
      block in `<path>` was rewritten to schema 2; a second run carries none.
- [x] `paired.go`: `paired(path string, src []byte, id string) (*frontmatter.Block, *frontmatter.Entry, error)`
      with the three refusals (other ids, broken front matter, `exported.note`);
      every reader in `read.go` and `write.go` calls it; the snapshot and
      proposal writes go into the entry; the schema line.
- ➕ Three signatures moved with the entry, because a proposal under one
      document says nothing about another: `propose.Record` takes the document
      id, `withdraw.Mine` and `withdraw.Run` take the entry, and
      `withdraw.Forget` takes the document id beside the suggestion id.
- ➕ `freshNote`'s refusal now opens with "the note changed while the run was
      under way", because `paired` writes the door check's own sentence and the
      re-read needs to say which of the two it was.
      `TestProposeWarnsWhenTheNoteStopsNamingThisDocumentMidRun` moved with it.
- [x] `doc.go`: the paragraph on `--md` says the URL picks the entry, and
      names the tests.
- [x] `cd go && go test -race ./...` passes; `make vet` passes.
- [x] `git commit -m "feat(frontmatter): the gdoc block is a list of documents, and every writer acts on the one the URL names"`

### Task 3: publish accepts a paired note

Serves decisions 6 and 17. Scenarios 10, 12, 14.

**Files:**
- Modify: `go/cmd/gdoc/publish.go`, `go/cmd/gdoc/publish_test.go`, `go/cmd/gdoc/doc.go`
- Remove: `docs/backlog/publish-a-paired-note-again.md`

- [x] Test first, `TestPublishAppendsAnEntryToAPairedNote`: a note with one
      entry; publish with a stubbed session; the block holds two entries, the
      first untouched byte for byte within its span, the second with
      `published` and `folder_id`; the envelope names the new id. Watch it
      fail on `unpaired`.
- [x] Delete `TestPublishRefusesANoteThatIsAlreadyPaired`. Keep the
      "a block appeared during the run" refusal in `pair`, restated as "an
      entry for this id appeared".
- [x] `unpaired` goes; `pair` appends.
- [x] `doc.go`: the `publish` paragraph says a paired note publishes again
      and names the test; the "open that document or take the block out"
      sentence goes.
- [x] `cd go && go test -race ./cmd/...` passes.
- [x] `git rm docs/backlog/publish-a-paired-note-again.md`
- [x] `git commit -m "feat(publish): a paired note publishes again, to a new document"`

### Task 4: a marker never reaches a document

Serves decision 3, last paragraph. Scenarios 13, 22.

**Files:**
- Create: `go/internal/markers/markers.go`, `go/internal/markers/markers_test.go`
- Modify: `go/internal/body/body.go`, `go/internal/body/doc.go`,
  `go/internal/plaintext/plaintext.go`, `go/internal/plaintext/doc.go` (or the
  package comment file), `go/cmd/gdoc/write.go`
- Create: `go/internal/body/markers_test.go`, `go/cmd/gdoc/propose_markers_test.go`

- [x] Test first, `TestMarkersNamesEachOpener`: the four openers and their
      closers as literals; `Find` returns the first one on a line and false on
      a line with an escaped one (`\{+`). Watch it fail.
- [x] Test, `TestBuildRefusesAMarkerByLine`: a note with `{+words+}[s:1]` on
      line 7; `body.Render` refuses naming line 7 and the marker; the same for
      `[[c:1]]`.
- [x] Test, `TestPlaintextRefusesAMarker`: `plaintext.Markdown` names a marker
      the way it names markdown.
- [x] Test, `TestProposeRefusesAMarkerInTheFile`: a `--from` file whose
      replacement carries `{-x-}`; `propose` refuses before any request,
      naming the field.
- [x] Implement the package, the three call sites, and the package comment
      of `markers` saying why a marker is refused everywhere text goes out.
- [x] `body/doc.go` and `plaintext`'s comment name the new tests.
- [x] `cd go && go test -race ./internal/markers/ ./internal/body/ ./internal/plaintext/ ./cmd/...` passes.
- [x] `git commit -m "feat: a gdoc marker is refused on every route into a document"`

### Task 5: the Docs read carries links, numbering, the contents element and floating objects

Serves decision 10 and the spec's "The file". Scenarios 4, 9, 19.

**Files:**
- Modify: `go/internal/docs/walk.go`, `go/internal/docs/docs.go`, `go/internal/docs/doc.go`
- Create: `go/internal/docs/links_test.go`, `go/internal/docs/testdata/links.json`,
  `go/internal/docs/testdata/lists.json`, `go/internal/docs/testdata/toc.json`,
  `go/internal/docs/testdata/positioned.json`

- [x] Test first, `TestARunCarriesItsLinkTarget`: fixtures written from the
      API reference for `url`, `headingId`, `bookmarkId` and `tabId`; each
      lands on `Run.Link`; a run with no link has nil. Watch it fail.
- [x] Test, `TestABulletCarriesItsListAndGlyph`: a `lists` map with an
      ordered and an unordered list; `Bullet.ListID`, `Ordered`, `Glyph`,
      `Level` per paragraph; the map is read per tab.
- [x] Test, `TestAParagraphCarriesItsHeadingID`.
- [x] Test, `TestATableOfContentsIsABlock`: the element lands as `Block.TOC`
      with its paragraphs, and `plainText` includes them.
- [x] Test, `TestAPositionedObjectIsCarriedOnItsParagraph`: the paragraph
      lists the id; the tab carries the object with its kind.
- [x] Implement the four decodes. Nothing else in the tree changes yet
      (`view` prints nothing new until Task 6).
- [x] `doc.go`: a section per new fact, each saying it is written from the
      reference until a live read agrees, naming the test, the way the
      seven-elements fixture is described.
- [x] `cd go && go test -race ./internal/docs/ ./internal/view/` passes with
      the five goldens unchanged.
- [x] `git commit -m "feat(docs): links, list numbering, the contents element and floating objects"`

### Task 6: read prints links and numbers, and escapes across runs

Serves decision 3 and the spec's "The file". Closes the across-runs blocker.

**Files:**
- Modify: `go/internal/view/text.go`, `go/internal/view/doc.go`,
  `go/internal/view/testdata/*.golden`, `docs/guide/reading.md`
- Create: `go/internal/view/links_test.go`, `go/internal/view/testdata/links.golden`,
  `go/internal/view/testdata/lists.golden`, `go/internal/view/testdata/positioned.golden`
- Remove: `docs/backlog/escaping-across-run-boundaries.md`

- [x] Test first, golden `links.golden`: `[words](https://...)`,
      `[words](#the-heading-words)` for a heading id, the label alone for a
      tab link; the same under `--structure` as a `link` field. Watch it
      fail.
- [x] Test, golden `lists.golden`: `1.`, `2.`, nested `   1.`, a second list
      starting at `1.` again, bullets unchanged.
- [x] Test, golden `positioned.golden`: `<!-- picture: floating, kix.p1 -->`
      after the anchoring paragraph, and one warning.
- [x] Test, `TestEscapingHoldsAcrossTwoRuns`: `["` in one run and `[` in the
      next gives `[\[` in the text; `{` then `+` gives `{\+`; a run ending
      in `[` before a comment-open marker gives `[\[[c:ID]]`.
- [x] Implement in `text.go`: `prefix` reads `Bullet`; `span` writes the link
      form; `writeText` carries the previous rune; the placeholder after a
      paragraph with `Positioned`.
- [x] `doc.go`: the marker table gains the link and number shapes; the
      escaping paragraph states the across-runs rule and names the test.
      `docs/guide/reading.md` drops the two lines that said links and
      numbering are lost.
- [x] Test, `TestUnescapeAfterEscapeIsIdentity`: a test-only inverse in
      `links_test.go` that undoes the escaping rules `doc.go` states; over
      every fixture and over a marker split across two runs, unescape after
      escape gives the source text back.
- [x] Re-record the five goldens that change, and paste the `git diff` of
      each golden into this plan as a ➕ note under this task, so revmux
      reads what moved: only link targets, numbers and placeholders.
- [x] `cd go && go test -race ./internal/view/ ./cmd/...` passes.
- [x] `git rm docs/backlog/escaping-across-run-boundaries.md`
- [x] `git commit -m "feat(view): read prints link targets and numbering, and escapes across runs"`

➕ **None of the five goldens moved.** `git status` lists no change under
`view/testdata/` beside the three new files. None of the five fixtures holds a
link, a numbered list or a floating object, and none holds a character that the
new escaping reaches: the literal markers in `single-tab.json` sit inside one run
and are already escaped on their first half, and no fixture has a document
character against one of gdoc's markers. `TestGolden` now reads eight fixtures
rather than five.

⚠️ **The escaping convention is the first half, not the second.** The plan said
`writeText` carries the previous run's last rune and escapes the second half, so
`[` then `[` would read `[\[`. That convention cannot spell the plan's own third
case: a run ending in `[` before a comment-open marker would put the backslash in
front of gdoc's marker, and a reader then takes the marker as the document's text
and loses it. So the projection holds the document's last character back until it
knows what follows, and escapes the first half everywhere: two runs give `\[[`,
and a character against a marker gives `\[[[c:ID]]`. One convention, not two, and
the reader's rule is one sentence: a backslash makes the one character after it
the document's own. That is the parity `internal/markers` already reads, and
`TestUnescapeAfterEscapeIsIdentity` is the reader written out. Nail's line in
Task 13's DECISIONS entry should record this wording rather than the plan's.

➕ **A bracket inside a link's words is escaped.** The link form makes one
bracket markup where it was not before, because a reader takes the words up to
the first `]`. Left alone, a document holding `Q3 [draft] plan` under a link
would cut the link short and leave its target standing as prose, which is the
defect the escaping exists to stop. `TestABracketInsideALinksWordsIsEscaped` is
the pin. One bracket inside a chip's label is still not escaped, which is the
older question `docs/backlog/escaping-across-run-boundaries.md` mentioned in
passing; `view/doc.go` now states it in place of the file, and the backlog count
stays at 22 here, 20 after Task 13.

➕ **The placeholder names the object's kind.** The plan wrote
`<!-- picture: floating, kix.p1 -->`. The fixture's second floating object is a
Google Drawing, and calling it a picture is a false fact of exactly the kind this
tree refuses, so the word before the colon is the kind the walk read:
`<!-- image: floating, kix.posone -->` and
`<!-- drawing: floating, kix.postwo -->`. Task 7's `positioned.md` golden and the
export warning should follow it.

➕ **`lists.json` gained a third tab.** The fixture Task 5 wrote holds one
numbered item, one nested item and one bullet, so it could not show `2.` or a
second list starting at `1.` again. Tab `t.2` holds two items of one list and one
item of another. Task 5's `TestABulletCarriesItsListAndGlyph` reads tabs 0 and 1
and is unchanged.

➕ **A numbered item counts even when it prints nothing.** An item holding no
text is skipped as every empty paragraph is, and still advances its count, so the
numbers say what the document shows rather than closing the gap over an item
nobody typed into.

➕ **A contents block still prints nothing.** `view` walks paragraphs and tables
and steps over `Block.TOC`, as it did before Task 5 made the element a block.
Task 7 owns the prelude and the strip, so nothing here prints one.

### Task 7: the export projection

Serves decisions 3, 8, 11 and the spec's "The file". Scenarios 1, 6, 7, 8, 9.

**Files:**
- Create: `go/internal/export/doc.go`, `go/internal/export/project.go`,
  `go/internal/export/strip.go`, `go/internal/export/project_test.go`,
  `go/internal/export/testdata/` (Docs JSON fixtures and `.md` goldens)
- Modify: `go/internal/view/text.go`, `go/internal/view/doc.go`
- Create: `go/internal/view/project_test.go`

- [x] Test first, in `view`, `TestProjectSkipsAndNamesPictures`: `Skip`
      drops a block, `Picture` writes its name raw on its own line, and
      `Text` equals `Project` with zero options on every golden. Watch it
      fail; implement `Project` and name the test in `view/doc.go`.
- [x] Test, golden `plain.md` from the `elements` fixture: the same
      text `read` prints, with markers, plus a front matter holding only a
      `gdoc:` block with one entry. Watch it fail.
- [x] Before the next two: `ls go/internal/export/testdata/publish-prelude/
      restyle-prelude/`. If Task 1's recordings are there, they are the
      fixtures. If not, build each from `internal/prelude`'s own request
      list rendered into the Docs JSON shape by a test helper, and write a
      ⚠️ note here saying the strip is pinned against gdoc's own idea of the
      prelude, not against a document Google made, until the recordings
      exist.
- [x] Test, golden `publish-prelude.md`: opens at the first body heading;
      `Stripped` lists cover, three tables, legend, contents, and the `{n}-`
      prefixes with their text.
- [x] Test, golden `restyle-prelude.md`: stripped over the named range.
- [x] Test, `TestAnEditedPreludeStaysAndWarns`: a cover with an added
      heading; nothing stripped, one warning naming what stood there.
- [x] Test, `TestAHeadingLinkRoundTripsToItsSlug` over the ampersand and
      accent headings of `body/testdata/05-edge-cases.md`'s shapes.
- [x] Test, `TestAChipExportsLabelAndTarget`.
- [x] Test, `TestAFootnoteIsWrittenAndWarned`.
- [x] Test, `TestTwoTabsGiveTwoProjections` with the tab title and id on
      each.
- [x] Implement `Project(d *docs.Document, pics PictureNames) ([]TabFile, []Piece, []string)`
      on `view`'s emitter over a stripped tree.
- [x] `doc.go`: the package comment says it projects and decides nothing,
      the strip rule as the spec states it, and names every test.
- [x] `cd go && go test -race ./internal/export/` passes.
- [x] `git commit -m "feat(export): the Markdown projection with the prelude stripped"`


⚠️ **The two prelude fixtures are gdoc's own idea of the prelude.** Task 1's
recordings were not in `go/internal/export/testdata/`, so `publish-prelude.json`
and `restyle-prelude.json` are written by hand in the shape a Docs read comes
back in, following house.yaml's own `front_matter` order: the cover lines, the
label, the three tables, the legend, the classification label and the contents
list, with the body numbered the way `body` numbers it. The strip is therefore
pinned against what gdoc believes it wrote, not against a document Google made.
The live recording replaces both files, and the goldens beside them say what
changed when it does.

➕ **A picture is written where it stands, not on a line of its own.** The spec
says a picture lands on its own line, and a picture in a paragraph of its own is
one: that is where `publish` puts one, and it is what `body` writes. A picture in
the middle of a sentence stays in the middle of that sentence, because breaking
the paragraph around it would cut the sentence into three chunks nothing joins
back up. `objects.json` is the fixture that holds one, and
`TestAPictureIsWrittenWhereItStands` is the pin.

➕ **`view.Options` gained a third field, `ChipTargets`.** The spec's file rules
say a chip keeps its label and its target, and `read` prints the label alone
because the target is in `--structure`. A file in the hub has no second read to
go back to, so the export asks for the target and gets
`[link: Title](https://...)`, the placeholder kept so nothing turns a chip into
an ordinary link behind a person's back. A target carrying a bracket, a
parenthesis or a space is left out, because the form ends at the first `)`.
`TestAChipCarriesItsTargetWhenAskedFor` and `TestAChipExportsLabelAndTarget` are
the two halves, and `plain.md` is therefore not byte-for-byte what `read` prints:
it is that text with the chip targets on it.

➕ **An inline object carries its own id, in `docs.Run.Detail.ID`.** `Picture` is
keyed by object id, and until now only a floating object had one: an inline
picture's run carried its kind and nothing else, so nothing could name it. The
decode is one line in `walk.go`, the run carries no label with it so every
placeholder reads as it did, and `TestAnInlineObjectCarriesItsObjectID` is the
pin. Task 8's pairing of the k-th media file with the k-th object names the
object by this id.

➕ **`view.slug` is exported as `view.Slug`, and a heading that loses its number
takes its links with it.** A heading `publish` wrote as `1-Scope` is linked to as
`#1-scope`, and the file holds `# Scope`, so the link would land nowhere. The
export moves every `](#1-scope)` to `](#scope)` through the one slug rule rather
than a second copy of it. `TestAHeadingLosesItsHouseNumberAndKeepsItsLinks` is
the pin. The heading number piece holds the heading as the document wrote it,
prefix and words, so a reader sees both what came off and what it stood on.

➕ **The contents label and the list under it are one piece.** The `Contents`
heading and the entries below it are what one page showed, and two pieces would
be something a reader has to put back together. A run of paragraphs holding no
text at all is not a piece: the house layout is half blank paragraphs.

### Task 8: the picture bytes, matched by order

Serves decision 10. Scenarios 4, 5.

**Files:**
- Modify: `go/internal/docx/docx.go`, including its package comment
- Create: `go/internal/docx/media_test.go`, `go/internal/docx/testdata/pictures.docx`
- Create: `go/internal/export/pictures.go`, `go/internal/export/pictures_test.go`

- [x] Before writing: `ls go/internal/export/testdata/fixture-measured/`. If
      Task 1's recording is there, the fixtures below are those files. If
      not, they are built from `body/testdata`'s pictures note through
      `render`, and write ⚠️ here saying so: a docx gdoc rendered pins gdoc's
      own decoder against gdoc's own writer, and says nothing about the order
      Google's export uses, which only the recording can.
- [x] Test first, `TestMediaComesInBodyOrder`: a docx with two pictures;
      `Media` returns them in `document.xml` order with names and bytes.
      Watch it fail.
- [x] Test, `TestPicturesPairByOrder`: two objects, two media files; each
      picture names its object id and bytes.
- [x] Test, `TestACountMismatchWritesNoPictureAndWarns`: three objects, two
      media; every picture is a placeholder, one warning.
- [x] Test, `TestTwoIdenticalPicturesKeepTheirOrder`.
- [x] Test, `TestTheHashMatchIsOffUntilMeasured`: with `Options.MatchByHash`
      false a matching PNG beside the note still gets a file and the result
      says the match is off; with it true the note's link is kept. No
      package variable: the option is a field, so `-race` sees no shared
      state.
- [x] Test, `TestTheNotesPicturesAreFoundThroughItsLinks`: PNG and JPEG
      resolved against the note's directory; `data:` and `http` skipped; a
      second note is never read.
- [x] Implement `docx.Media` and `export.Pictures`.
- [x] `cd go && go test -race ./internal/docx/ ./internal/export/` passes.
- [x] `git commit -m "feat(export): picture bytes from the docx export, matched by order"`

⚠️ **The docx fixtures are gdoc's own writer, not Google's export.** Task 1's
recording is not in `go/internal/export/testdata/fixture-measured/`, so
`TestMediaComesInBodyOrder` renders `badge.png` and `diagram.png` through
`body` and `render` and reads that package back. It pins gdoc's decoder against
gdoc's writer and says nothing about the order Google's export uses, which only
measurement 1 can. Until that recording lands, the guard that matters is the
count: the two routes disagreeing means every picture is a placeholder and
nothing is written.

➕ **No `pictures.docx` on disk.** The plan named a committed fixture. The test
builds the package in memory through the generator instead, the way
`drift_test.go` builds the note it measures, so there is no binary blob in the
tree whose provenance a reader has to take on trust. The other four cases are
hand-written zips through `docx_test.go`'s own `buildDocx`, because what they
pin is a malformed export and the generator cannot write one.

➕ **Only `a:blip` is read, and never VML.** Word writes a floating picture
twice, as DrawingML inside `mc:Choice` and as VML inside `mc:Fallback`, so
reading both would count one picture as two and put the pairing out by one. A
picture that reaches the body as VML alone is therefore missing, and the count
guard catches it: a missing picture is a placeholder, never a wrong file.

➕ **A picture `Media` cannot resolve is refused, never skipped.** A relationship
the part list does not hold, one marked `External`, and a part the zip does not
carry are all errors naming what is missing. Skipping one would move every
picture after it one place along, and the pairing is by position, so every
picture behind it would get the wrong bytes under the right name.
`TestMediaRefusesWhatItCannotResolve` is the pin.

➕ **The "match is off" warning needs something to have matched.** The spec says
the reply says the match is not trusted yet. It says so when the note at `--out`
holds pictures of its own, and says nothing when it holds none: there was
nothing the match could have kept, so the sentence would be noise on every
export into a new file.

➕ **A matched picture carries no bytes.** `Matched` and `Bytes` are never both
set, so "no file is written" is the shape of the value rather than a rule Task 9
has to remember. `Ext` comes off the media part's own name, so the writer names
a file what it is without reading it.

➕ **Two more tests than the plan named.**
`TestADifferentPictureIsWrittenWhenTheMatchIsOn` is scenario 4 with the match
on, which nothing else covered, and `TestANotePictureThatCannotBeReadIsNamed`
holds the rule that a broken picture link in the note is named rather than
dropped: the export is the one read that could have told the session about it.

➕ **`Objects` is the list the pairing counts, and it lives here.** An inline
picture stands where its run does and a floating one after the paragraph it is
anchored to, which is where `internal/view` prints its placeholder. An equation
or an object this read cannot name is a placeholder and never a file, so it is
not counted: counting it would put the pairing one place out.
`TestObjectsAreTheTabsPicturesInBodyOrder` is the pin.

### Task 9: the files on disk, the stamp and the copy

Serves decisions 1, 2, 4, 11, 13, 14. Scenarios 1, 2, 15, 16, 19, 21, 22, 23.

**Files:**
- Create: `go/internal/export/write.go`, `go/internal/export/write_test.go`
- Modify: `go/internal/export/doc.go`
- Modify: `go/internal/atomicfile/atomicfile.go`, `go/internal/atomicfile/atomicfile_test.go`

- [x] Test first, `TestAFreePathIsTaken`: `<stem>.md` and `assets/<stem>-1.png`
      written, the folder created, the block holds one entry with `exported`.
      Watch it fail.
- [x] Test, `TestATakenPathGetsTheNextFreeNumber`: `.2`, then `.3`; pictures
      from the next free number; nothing existing is touched, checked by
      hash.
- [x] Test first in `atomicfile`, `TestCreateRefusesAPathThatExists`:
      `Replace` ends in `os.Rename`, which always replaces, so export cannot
      use it for a new file. Add `Create(path string, b []byte) error`: write
      the temp file in the same directory, then `os.Link` it to `path`, which
      fails when `path` exists, and remove the temp file either way. A path
      that appears between the free-name check and the link is refused, not
      replaced. The package comment says why there are two verbs.
- [x] Test, `TestNothingCanReplaceAFile`: the export package exposes no force
      parameter, every new file goes through `atomicfile.Create`, and only
      the stamp on a note goes through `Replace`.
- [x] Test, `TestExportStampsTheNoteAndChangesNoOtherByte`: a paired note at
      the path; the copy lands at `.2` with `exported.note` relative to its
      directory; the note's bytes differ only inside the block, by the
      `exported` line.
- [x] Test, `TestTheDoorChecks`: a note naming other documents, broken front
      matter, and the same two on a tab's path; refused before any file is
      written.
- [x] Test, `TestATabPathIsCheckedLikeOut` and `TestTabSlugsCollide`: the
      slug rule on literals, an empty title, two tabs with one title.
- [x] Implement `Write`.
- [x] `doc.go` names every test.
- [x] `cd go && go test -race ./internal/export/ ./internal/atomicfile/` passes.
- [x] `git commit -m "feat(export): files on disk, the stamp on the note, and the copy beside it"`


➕ **Two halves, `Plan` and `Write`.** The plan named one function. A picture's
line in the body names the file that picture lands in, so every path has to be
decided before the text exists, and the text is what `Write` is handed. `Plan`
takes every path and answers both door checks without writing a byte;
`(*Layout).PictureNames` is the handover into `Project`; `Write` writes.
`TestTheNamesReachTheBody` is that seam, and the gap between the two is closed
by `atomicfile.Create` refusing a path that appeared in between rather than
replacing it.

➕ **`atomicfile.Create` takes no mode, and `NewMode` names the one it uses.**
There is no file at the path to take a mode from, so the package states 0644
as a constant rather than leaving a new file to whatever umask the temp file
was born under. `Replace` still takes the mode, because there it is the mode of
the file being replaced.

➕ **A number under `assets/` is taken whatever extension wrote it.** A PNG at
`<stem>-1.png` means 1 is gone, so a JPEG cannot land as `<stem>-1.jpg` beside
it and leave a reader guessing which is which.

➕ **The stamp carries an `exported.note` through.** Only a person takes that
line out, by hand, so a stamp on a file that has one sets the date and leaves
the line where it is.

➕ **Three tests beyond the plan's list.** `TestTheNamesReachTheBody` above;
`TestASchemaOneNoteIsRewrittenAndSaidSo`, which is scenario 23 and is what
every note on the team does on its first stamp; and, in `atomicfile`,
`TestCreateWritesAFileThatWasNotThere` and
`TestCreateRefusesADirectoryThatExists` beside the refusal the plan named.

### Task 10: the export command

Serves decisions 10, 15. Every export scenario.

**Files:**
- Create: `go/cmd/gdoc/export.go`, `go/cmd/gdoc/export_test.go`,
  `go/internal/guard/export_test.go`
- Modify: `go/cmd/gdoc/commands.go`, `go/cmd/gdoc/doc.go`,
  `go/cmd/gdoc/help_test.go`, `go/cmd/gdoc/completion_test.go`

- [x] Test first, in `help_test.go`: the fifteenth usage line
      `export <url> --out <file>` in
      `TestTheUsageLineMarksWhatIsOptionalAndWhatIsAnAlternative`; completion
      lists it. Watch it fail.
- [x] Test, `TestExportOpensOnlyThePolicyReadOpens` in `guard`: the requests
      export makes, the Docs read with tabs and the docx export, are the ones
      `read` and `comments --witness` already make; no other host, no other
      method.
- [x] Test, `TestExportRefusesAFileThatIsNotADocument`.
- [x] Test, `TestExportEnvelopeCountsPendingOwnThreadsAndStripped`: the
      counts, `own` present only when `--out` holds a note listing proposals
      for this document, the stripped pieces with text, the files with tabs,
      the pictures with `matched`, the schema line.
- [x] Test, `TestExportSendsNothingThatWrites`: over a scripted session, no
      POST, PATCH or DELETE.
- [x] Implement the row, the flag, `run`, and the `doc.go` entry for `help
      export`, which names `assets`, the numbering rule and the block.
- [x] `cd go && go test -race ./cmd/... ./internal/guard/` passes.
- [x] `git commit -m "feat(gdoc): export, a Google Doc into the hub as Markdown with its pictures"`

### Task 11: the live round trip

Serves the spec's Tests. Test only; Nail runs it.

**Files:**
- Create: `go/internal/live/export_test.go`
- Modify: `go/internal/live/doc.go`

- [ ] `TestLiveExportRoundTrip`, gated on `GDOC_LIVE_WRITE`: publishes each
      of the six `body/testdata` notes into the test folder, exports each,
      compares the export body against the note against a named list of
      known differences with reasons (front matter, the prelude, footnotes,
      code blocks, relative links as words, heading numbers), fails on an
      unnamed difference, trashes every document.
- [ ] `TestLiveExportCarriesAProposal`: publish, propose, export; the
      suggestion id is in the file; withdraw; trash.
- [ ] `TestLiveExportHashEquality`: publish the pictures note, export, print
      whether the media hash equals the upload; assert it only when the
      `MatchByHash` option is on in the binary, so the test is the
      measurement first and the pin later.
- [ ] `TestLivePublishAgainAppendsAnEntry`: publish a note, publish it
      again; the block holds two entries, both documents exist; trash both.
- [ ] `doc.go` names the three and what each creates.
- [ ] `cd go && go test -race ./internal/live/` compiles and skips.
- [ ] `git commit -m "test(live): the export round trip over the six body notes"`

### Task 12: the five skills

Serves decisions 7, 12, 13, 14, 16, 17. Scenarios 1 to 3, 12, 13, 15, 20, 22.

**Files:**
- Create: `skills/gdoc-export/SKILL.md`, `skills/gdoc-export/markers.md`,
  `skills/gdoc-align/SKILL.md`
- Modify: `skills/gdoc-publish/SKILL.md`, `skills/gdoc-review/SKILL.md`,
  `install.sh`, `release/install.sh`, `.claude-plugin/plugin.json`,
  `go/cmd/gdoc/skills_test.go`
- Remove: `docs/backlog/publish-skill-renders-an-svg-picture.md`

- [ ] Test first, in `skills_test.go`: five skills, each with `name`,
      `description` and `needs`; four say `v2.4.0`, restyle `v2.0.0`; no
      skill carries `disable-model-invocation`; no skill names a person or a
      machine's path; `gdoc-export/markers.md` exists and `gdoc-align/SKILL.md`
      names it. Watch it fail.
- [ ] `gdoc-export/SKILL.md`: the export skill section of the spec as steps;
      the first line naming the skill; `needs: v2.4.0`.
- [ ] `gdoc-export/markers.md`: each marker, its meaning, its escaping, how
      a session undoes it, and the rule that gdoc's own pending proposals are
      not asked about.
- [ ] `gdoc-align/SKILL.md`: the three runs, the paragraph and logic
      conflicts, the ask before the first `propose`, the stale copies, the
      folder check for a second file naming the document, the two sentences,
      the promises and refusals.
- [ ] `gdoc-publish/SKILL.md`: Step 1 no longer says a block means
      published; the once-only sentence goes; the paired-note ask; the SVG
      section and `svg2png.js` from the backlog item; `needs: v2.4.0`.
- [ ] `gdoc-review/SKILL.md`: `ai!` may not run export, one reply says so;
      `needs: v2.4.0`; the first line.
- [ ] `install.sh:29` lists five; `release/install.sh` copies five;
      `plugin.json` description names export and align.
- [ ] `cd go && go test -race ./cmd/...` passes.
- [ ] `git rm docs/backlog/publish-skill-renders-an-svg-picture.md`
- [ ] `git commit -m "feat(skills): gdoc-export and gdoc-align, and publish asks on a paired note"`

### Task 13: the documents, the decisions and the backlog

Serves decisions 5 and 9, and the spec's "Documents that change",
"DECISIONS.md entries" and "Backlog items".

**Files:**
- Modify: `README.md`, `docs/guide/how-it-works.md`, `docs/guide/publishing.md`,
  `docs/v2/SPEC.md`, `docs/v2/DECISIONS.md`, `docs/v2/MEASURED.md`,
  `docs/v2/PLAN.md`, `CLAUDE.md`, `docs/backlog/m8-alignment-and-the-align-skill.md`,
  `docs/backlog/read-pictures-and-drawings.md`,
  `docs/plans/2026-09-19-gdoc-v2-m13-export-and-align.md`
- Create: `docs/guide/exporting.md`
- Remove: `docs/backlog/export-a-document-as-markdown-with-its-pictures.md`

- [ ] `DECISIONS.md`: one entry dated 2026-09-19 with the register rows the
      spec lists, plus the across-runs escaping convention from Task 6, and
      the register updated. Decision 5 is the row that supersedes
      2026-08-13.
- [ ] The spec's decision 9 and scenario 18, and the export guide: the
      sentence the shipped binary prints on a schema 2 block is the literal
      Task 2 pinned and left here as a ➕ note, a strict-yaml refusal naming
      the unknown key `documents`, not a schema sentence. Rewrite all three
      to that literal.
- [ ] `MEASURED.md`: if Nail has run Task 1 and written its three answers as
      ➕ notes in this plan, the three rows go in as measured; otherwise the
      three rows go under "Not measured yet". The `MatchByHash` option ships
      `false` either way; flipping it is Nail's commit after the merge, named
      in Post-Completion. No Go file changes in this task.
- [ ] `SPEC.md`: `publish` runs more than once; a new `export` section; the
      diff section describes the two skills as built; each change dated.
- [ ] `README.md`: the fourth recipe. `how-it-works.md`: the source-of-truth
      line and the `assets` line. `publishing.md`: the block in its list
      shape, publish-many, the SVG sentence. `exporting.md`: the numbering
      rule, the PNG names, what the markers mean, one file per tab.
- [ ] `CLAUDE.md`: `go/internal/export/` and `go/internal/markers/` in the
      table, five skills in the `skills/` row, and the invariant list gains
      "Nothing is overwritten by export" naming `TestNothingCanReplaceAFile`.
      Under 300 lines.
- [ ] `PLAN.md`: M13 recorded, the next milestone open.
- [ ] Backlog: `git rm` the export item; rewrite the M8 item to hold only the
      hub-wide live session and `sergi/go-diff`; rewrite the pictures item to
      hold only what is still not read (the bytes in `read` itself, if Nail
      wants them there).
- [ ] `cd go && go test -race ./boundary/ ./cmd/...` passes (CLAUDE.md
      ceiling, task map, skills).
- [ ] `git commit -m "docs(v2): M13 export and align, decisions, guides and backlog"`

### Task 14: Verify acceptance criteria

- [ ] `make test`, `make vet`, `make build` and `make dist` pass.
- [ ] Every Validation Command above gives the answer it states; record the
      counts here as ➕ notes.
- [ ] Every scenario in the spec names a task above that serves it, checked
      by reading the list; a scenario no task serves is a ➕ task.
- [ ] `git diff main...HEAD -- go/internal/guard/ | grep '^-' | grep -v '^---'`
      is empty: nothing left the guard.
- [ ] `grep -rn 'force' go/internal/export/ go/cmd/gdoc/export.go` prints
      nothing.
- [ ] Nothing to commit: this task changes no file but this plan.

### Task 15: Update documentation

- [ ] Move this plan and the spec to `docs/plans/completed/`.
- [ ] `cd go && go test -race ./boundary/` passes.
- [ ] `git commit -m "docs(v2): M13 export and align, completed"`

## Post-Completion

**By hand, Nail, as early as possible**: make the fixture document Task 1
describes in the test folder, set `GDOC_LIVE_EXPORT_DOC_ID`, run
`TestLiveExportMeasurements`, and paste its three answers into this plan as
➕ notes under Task 1. Task 13 reads them from here. If the run reaches Task 13
before the answers exist, the rows stay under "Not measured yet", which the
spec allows. The hash match ships off in every case: when measurement 2 says
the bytes are equal, Nail flips `MatchByHash` to `true` in one commit after
the merge, with the MEASURED.md row and `TestLiveExportHashEquality` turned
from a print into an assertion.

**By hand, Nail, before the merge**: run the three live tests of Task 11 with
`GDOC_LIVE_WRITE=1`; publish the pictures note, export it, open both in the
browser and check the picture; export a document with two tabs and read both
files; run `/gdoc-align` on a note whose document a colleague edited, and
`/gdoc-export` on a Doc with no note.

**After the merge**: `make tag` with `VERSION=v2.4.0`, then
`gh workflow run nightly.yml --ref main`, and tell the team the skills need
`gdoc update` before an exported note reads on their machine.

**Not in this milestone, and named so nobody rediscovers it**: a diff command
and `sergi/go-diff`; the hub-wide live session; the picture bytes inside
`read` itself; Windows and Linux lines for the SVG route; proposing into one
tab of a tabbed document, which stays refused until measured.
