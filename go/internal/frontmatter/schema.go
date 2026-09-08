// Package frontmatter reads and writes the gdoc: block in a markdown note's
// YAML front matter, and nothing else in the file.
//
// Two rules shape every function here. The read is strict: an unknown key, a
// duplicate key or a schema this version does not know is refused with the key
// named, because a block gdoc half understands is a pairing it may act on
// wrongly. The write is byte-preserving: only the gdoc: key's span changes, and
// the author's keys, their line endings and the trailing newline come through
// untouched.
package frontmatter

import (
	"fmt"
	"regexp"
	"time"
)

// Schema is the version this package reads and writes. A block stating any
// other version is refused rather than read on a guess.
const Schema = 1

// The two words a seen suggestion's kind may be.
const (
	KindInsertion = "insertion"
	KindDeletion  = "deletion"
)

// driveID is what a Drive file id looks like. Refusing anything else here
// keeps a path or a URL fragment out of a field the guard is later opened with.
var driveID = regexp.MustCompile(`^[A-Za-z0-9_-]{20,}$`)

// Block is what gdoc owns in a markdown note. Everything outside it belongs to
// the author.
type Block struct {
	Schema          int              `yaml:"schema"`
	DocumentID      string           `yaml:"document_id"`
	FolderID        string           `yaml:"folder_id,omitempty"`
	Published       *Published       `yaml:"published,omitempty"`
	SuggestionsSeen *SuggestionsSeen `yaml:"suggestions_seen,omitempty"`
	Proposals       []Proposal       `yaml:"proposals,omitempty"`
}

// Published is what publish records about the document it made: when it was
// made, the title that went on its cover, and where the house style came from,
// either "embedded" or the path a --house run named.
//
// There is no revision id. Reading one back would need a Drive route the guard
// does not carry, for a field nothing here reads, so a block carrying
// revision_id is refused by name under the strict read.
type Published struct {
	At    time.Time `yaml:"at"`
	Title string    `yaml:"title"`
	House string    `yaml:"house,omitempty"`
}

// SuggestionsSeen is the snapshot `gdoc suggestions --md` writes after a
// successful read. It is what the next run compares against to say which
// suggestions stopped being pending.
type SuggestionsSeen struct {
	At    time.Time        `yaml:"at"`
	Items []SuggestionSeen `yaml:"items"`
}

// SuggestionSeen is one pending suggestion as it looked at snapshot time.
type SuggestionSeen struct {
	ID      string `yaml:"id"`
	Kind    string `yaml:"kind"`
	Section string `yaml:"section"`
	Text    string `yaml:"text"`
}

// Proposal is one of gdoc's own suggestions. M3 writes these; M2 only carried
// them through a read and a write unchanged.
//
// Quoted is the text the proposal replaced. It is what a later run shows Nail
// when it says which proposal it is about to withdraw, and it is optional in
// the decoder because a note paired under M2 carries proposals without it: a
// strict read there would unpair every note gdoc has already written.
type Proposal struct {
	ID        string    `yaml:"id"`
	CommentID string    `yaml:"comment_id"`
	At        time.Time `yaml:"at"`
	Quoted    string    `yaml:"quoted,omitempty"`
}

// Validate reports the first rule the block breaks, naming the key. A block
// that fails any rule is never written and never returned from a read.
func (b *Block) Validate() error {
	if b == nil {
		return fmt.Errorf("gdoc front matter: no block")
	}
	if b.Schema == 0 {
		return fmt.Errorf("gdoc front matter: schema is required and must be %d", Schema)
	}
	if b.Schema != Schema {
		return fmt.Errorf("gdoc front matter: schema is %d, and this gdoc reads %d", b.Schema, Schema)
	}
	if b.DocumentID == "" {
		return fmt.Errorf("gdoc front matter: document_id is required")
	}
	if !driveID.MatchString(b.DocumentID) {
		return fmt.Errorf("gdoc front matter: document_id %q is not a Drive id", b.DocumentID)
	}
	if b.FolderID != "" && !driveID.MatchString(b.FolderID) {
		return fmt.Errorf("gdoc front matter: folder_id %q is not a Drive id", b.FolderID)
	}
	if b.Published != nil {
		if b.Published.At.IsZero() {
			return fmt.Errorf("gdoc front matter: published.at is required")
		}
		if b.Published.Title == "" {
			return fmt.Errorf("gdoc front matter: published.title is required")
		}
	}
	if b.SuggestionsSeen != nil {
		for i, item := range b.SuggestionsSeen.Items {
			if item.ID == "" {
				return fmt.Errorf("gdoc front matter: suggestions_seen.items[%d].id is required", i)
			}
			if item.Kind != KindInsertion && item.Kind != KindDeletion {
				return fmt.Errorf("gdoc front matter: suggestions_seen.items[%d].kind is %q, and it is %s or %s",
					i, item.Kind, KindInsertion, KindDeletion)
			}
		}
	}
	for i, p := range b.Proposals {
		if p.ID == "" {
			return fmt.Errorf("gdoc front matter: proposals[%d].id is required", i)
		}
		if p.CommentID == "" {
			return fmt.Errorf("gdoc front matter: proposals[%d].comment_id is required", i)
		}
		if p.At.IsZero() {
			return fmt.Errorf("gdoc front matter: proposals[%d].at is required", i)
		}
	}
	return nil
}
