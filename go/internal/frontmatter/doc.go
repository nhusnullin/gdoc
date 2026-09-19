// Package frontmatter is the gdoc: block in a note's YAML front matter, and
// nothing else in the file.
//
// The block carries schema and documents, a list with one entry per document
// the note is paired with. An entry carries id, folder_id, tab_id, a published
// record, an exported record, the suggestions_seen snapshot and proposals. A
// proposal is
// {id, comment_id, at, quoted}: the suggestion id, the comment insertComment
// returned, the time, and the words that were replaced. propose writes those
// entries and withdraw reads them and shortens the list, so the block is the
// only record of what gdoc itself wrote into a document. quoted is optional in
// the decoder, because a note paired before that field existed carries
// proposals without it, and refusing those notes would unpair every note gdoc
// has already written. TestReadDecodesTheFullBlock,
// TestReadDecodesProposalsWithAndWithoutQuoted and
// TestWriteKeepsQuotedThroughARoundTrip are the pins.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # The block is a list, and schema 1 still reads
//
// A note published before 2026-09-19 carries schema 1: one document, its fields
// beside the schema rather than under an entry. That note still reads, as a
// block whose documents list holds one entry, and the Schema field says 1,
// because that is what the file says.
// TestReadAcceptsASchemaOneBlockWrittenBeforeToday and
// TestASchemaOneBlockCarriesItsOldFieldsOnTheEntry are the pins.
//
// Write compares the block it is given against the one already in the file. A
// block that is equal is a write with nothing to write, and the file comes back
// byte for byte, which is what keeps those notes at schema 1 until something in
// them changes. Any other write renders schema 2, whatever the block says,
// because this package has one shape it writes.
// TestWriteAnUnchangedSchemaOneBlockStaysSchemaOne,
// TestAChangingWriteRewritesSchemaOneAsTwo and TestASchemaTwoBlockRoundTrips
// are the pins.
//
// The reason for the list is that a note publishes more than once. Which entry
// a run acts on is the document the URL names, and Entry is that lookup: it
// returns the block's own entry, so a writer that changes it and writes the
// block writes what it changed, and a document the block does not name is a
// refusal naming every document it does. TestEntryFindsTheDocumentTheURLNames
// and TestEntryIsTheBlocksOwnEntry are the pins.
//
// Two entries naming one document are refused, because gdoc could not tell
// which one a run means. An exported record carrying neither at nor note is
// refused, because one of them is what says what was exported. A tab_id that is
// not a Docs tab id is refused, for the reason a document_id that is not a
// Drive id is: a path or a URL fragment must not reach a field a later run
// reads back. Each refusal names the key and the entry it is in.
// TestValidateNamesTheEntryItRefused, TestValidateAcceptsADocsTabID and
// TestValidateAcceptsAnExportedRecord are the pins.
//
// A gdoc from before this milestone reads a schema 2 note and refuses it by
// name: its read is strict before it looks at the schema, so it names the
// documents key it did not know rather than the version.
// TestTheShippedDecoderRefusesADocumentsKeyByName is the pin.
//
// # The publish record is {at, title, house}
//
// When the run happened, the title that went on the cover, and whether the
// style came from the embedded file or from a --house path. There is no
// revision id beside those: reading a Drive revision id needs a route the guard
// does not carry, for a field nothing here reads. Nail's decision, 2026-09-08,
// DECISIONS.md. Validate refuses a published record missing at or title, and a
// block still naming revision_id is refused by name under the strict read, like
// any other unknown key. TestValidateAcceptsAPublishRecord,
// TestValidateNamesTheKeyItRefused, TestWriteKeepsThePublishRecordThroughARoundTrip
// and TestReadRefusesAndNamesWhatIsWrong, over published-revision-id.md, are
// the pins.
//
// # The read is strict
//
// goccy/go-yaml with yaml.Strict(): an unknown key, a key given twice, a
// missing id, an id that is not a Drive id, and a kind that is
// neither insertion nor deletion. Each is refused naming the key, and the file
// is left untouched. A block gdoc half understands is a pairing it may act on
// wrongly. TestReadRefusesAndNamesWhatIsWrong and
// TestValidateNamesTheKeyItRefused are the pins, with
// TestReadRefusesAnEmptyGdocKey over a key holding nothing at all.
//
// The front matter is one YAML document by construction, so nothing checks for
// a second one: it closes at the first --- or ... line, which is where a second
// document would have begun. A delimiter is recognised with trailing spaces or
// tabs after it, and behind a leading byte order mark, because Jekyll,
// python-frontmatter and goldmark-meta all read those as front matter. A note
// gdoc reads as unpaired is a note Write puts a second block in front of,
// demoting the author's keys to prose.
// TestTrailingSpaceOnTheDelimiterIsStillFrontMatter and
// TestAByteOrderMarkDoesNotHideTheFrontMatter are the pins.
//
// The author's own keys are checked too, and on every path. A file being paired
// for the first time has no gdoc: key at all, so checking only when one is
// already there would check every case but the first.
// TestReadRefusesFrontMatterTheAuthorBroke and
// TestWriteRefusesBrokenFrontMatterWithNoGdocKey are the two halves.
//
// # A bare string where a block is expected is refused, and the refusal says
// what to do
//
// A gdoc: key holding a plain string is a pairing this reader does not read. It
// is refused with the string it found named, and the message carries the two
// ways out: write the key as a block stating schema and a documents list whose
// one entry names the id, or take the line out and pair the note again with
// gdoc publish. Nail's decision,
// 2026-09-08, DECISIONS.md.
//
// What makes the refusal safe to keep is that publish is the only command that
// creates the block. suggestions --md, propose --md and withdraw write into
// one, and each of them refuses a note that has none, so no shape this reader
// does not know ever enters it. A key holding any other scalar does not get
// that sentence: telling somebody to rewrite gdoc: 3 as a document id sends
// them the wrong way. TestReadRefusesABareStringPairingAndSaysWhatToDoAboutIt,
// TestWriteRefusesABareStringPairingRatherThanReplacingIt and
// TestANonStringScalarUnderGdocIsNotABareStringPairing are the pins.
//
// # schema is 1 or 2, and nothing else
//
// A block stating another version is refused rather than read on a guess.
// Schema is the constant this package writes and SchemaOne is the one it still
// reads, and adding a third is a decision rather than a refactor, because the
// block is the record a later run reads before it acts. The schema cases in
// TestValidateNamesTheKeyItRefused and TestReadRefusesAndNamesWhatIsWrong, over
// schema3.md, are the pins.
//
// # The write is byte-preserving
//
// Only the gdoc: span changes. The author's keys, their order, the blank lines
// around them, the line endings and the trailing newline come through
// unchanged, and a file with no front matter at all gets the block added with
// new delimiters. The line endings are read off the file's own front matter
// rather than off the whole file, so one CRLF anywhere in the prose does not
// churn the endings of a block gdoc promises to leave alone.
// TestWriteAnUnchangedBlockIsByteIdentical, TestWriteMovesNoLineOutsideTheBlock,
// TestWriteGivesBackTheBlankLineAfterTheBlock, TestWriteKeepsCRLF,
// TestWriteTakesTheLineEndingsFromTheFrontMatterNotTheProse,
// TestWriteAppendsIntoExistingFrontMatter and
// TestWriteGivesAFileWithoutFrontMatterDelimitersAndNothingElse are the pins.
//
// Read and Write share one parse, because Write calls readFrom on the same
// document, so the two can never disagree about where the span is. A file whose
// own block cannot be read is refused rather than overwritten: not knowing what
// is there must never resolve to replacing it.
// TestWriteRefusesAFileItCouldNotRead, TestWriteRefusesAnInvalidBlockAndTouchesNothing
// and TestWriteRefusesANilBlock are the pins.
//
// A file whose opening --- never closes is neither of those cases. Read reports
// no block, because there is no front matter to read, and Write refuses it.
// Writing there would put a second block in front of the author's keys and
// demote their own gdoc: key to prose, which is gdoc pairing a note it had just
// broken. TestReadTreatsAnUnclosedDelimiterAsNoFrontMatter and
// TestWriteRefusesAFileWhoseFrontMatterNeverCloses are the two halves.
//
// # The write is checked against its own parse before it leaves the package
//
// A string carrying a control character is written double quoted, because the
// emitter writes it as a plain scalar the parser reads differently: a tab
// inside one is dropped on the way back in, and a bare carriage return produces
// a block that fails to parse at all. Google Docs puts a tab in a text run
// wherever the author typed one, so this is the snapshot's own words. verify
// then renders the block, reads it back and renders it again, and refuses a
// block whose two renderings differ.
//
// It has to be a refusal rather than a warning, because Write reads the block
// it finds before replacing it: a block gdoc broke is a note gdoc would then
// never touch again. The check runs while the file on disk is still untouched.
// TestControlCharactersSurviveTheRoundTrip and
// TestVerifyRefusesABlockThatDoesNotReadBack are the pins.
//
// # The snapshot is written after a successful read, never before
//
// A read that failed knows nothing about what is pending, and a snapshot taken
// then would report everything that run could not see as gone on the next one.
// Which ids go into it is internal/suggestions' rule, and its doc comment holds
// it. The write goes through internal/atomicfile, a temp file in the same
// directory and a rename, keeping the file's mode, because the markdown is the
// source and gdoc is not its only reader. A file whose block names another
// document is refused rather than repaired.
// TestSuggestionsLeavesTheFileAloneWhenTheReadFailed,
// TestSuggestionsWithMDWritesTheSnapshotAndLeavesEveryOtherLineAlone,
// TestTheSnapshotWriteKeepsTheNotesMode,
// TestSuggestionsRefusesAFilePairedWithAnotherDocument and
// TestSuggestionsRefusesAFileWithNoBlock in cmd/gdoc are the pins.
package frontmatter
