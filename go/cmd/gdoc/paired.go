// This file is the note a --md flag names, read once and checked once. read.go
// and write.go both go through it, so suggestions, propose and withdraw refuse
// the same things in the same words.

package main

import (
	"fmt"
	"strings"

	"gdoc/internal/frontmatter"
)

// paired reads the note's block and returns the entry for the document this run
// is of. A note names every document it has been published to, and the URL is
// what says which one a run means.
//
// Three things are refused rather than acted on. A note whose front matter does
// not read is frontmatter's own refusal, passed through: a block gdoc half
// understands is a pairing it may act on wrongly. A note that does not name
// this document is the wrong file, and writing this run's observation into it
// would be read by the next run as that document's history. A note whose entry
// carries exported.note is a copy of somebody else's document rather than the
// source of one, so there is nothing in it to record and nothing that would
// ever be published from it.
func paired(path string, src []byte, id string) (*frontmatter.Block, *frontmatter.Entry, error) {
	block, err := frontmatter.Read(src)
	if err != nil {
		return nil, nil, err
	}
	if block == nil {
		return nil, nil, fmt.Errorf("%s carries no gdoc: front matter, so it is not paired with a document", path)
	}
	entry, err := block.Entry(id)
	if err != nil {
		return nil, nil, fmt.Errorf("%s is paired with %s, and this run is of %s", path, pairedWith(block), id)
	}
	if entry.Exported != nil && entry.Exported.Note != "" {
		return nil, nil, fmt.Errorf("%s is a copy gdoc exported from document %s, and the note it was exported beside is %s; run this against that note",
			path, id, entry.Exported.Note)
	}
	return block, entry, nil
}

// pairedWith names the documents a note holds, so a refusal says what to open
// instead of this one.
func pairedWith(block *frontmatter.Block) string {
	ids := block.IDs()
	if len(ids) == 1 {
		return "document " + ids[0]
	}
	return "documents " + strings.Join(ids, " and ")
}

// rewritten is the warning a run carries when its write moved the note from the
// block publish wrote before 2026-09-19 to the one gdoc writes now. It is said
// once, because the next run reads a schema 2 block and has nothing to say.
//
// schemaOne is what the block said when it was read, and the caller adds the
// warning only after it has written: a run that changed nothing wrote nothing,
// and saying the note was rewritten then would be a fact that is not true.
func rewritten(path string, schemaOne bool) []string {
	if !schemaOne {
		return nil
	}
	return []string{fmt.Sprintf("the gdoc: block in %s was rewritten from schema %d to schema %d, which is one entry per document",
		path, frontmatter.SchemaOne, frontmatter.Schema)}
}
