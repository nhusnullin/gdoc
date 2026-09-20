// This file is the block's own shape: the fields gdoc owns in a note, what a
// Drive id looks like, and Validate, which names the first key a block breaks.
// frontmatter.go is the read and the byte-preserving write, and doc.go holds
// the package comment.

package frontmatter

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Schema is the version this package writes: one note, a list of documents.
const Schema = 2

// SchemaOne is the version publish wrote until 2026-09-19, one document per
// note. It is still read, because those notes are in somebody's vault, and a
// note nothing changed in is written back as it was.
const SchemaOne = 1

// The two words a seen suggestion's kind may be.
const (
	KindInsertion = "insertion"
	KindDeletion  = "deletion"
)

// driveID is what a Drive file id looks like. Refusing anything else here
// keeps a path or a URL fragment out of a field the guard is later opened with.
var driveID = regexp.MustCompile(`^[A-Za-z0-9_-]{20,}$`)

// tabID is what a Docs tab id looks like: t.0 on a document with one tab, and
// t.<id> or kix.<id> beyond that. It is not a Drive id, so it has its own rule,
// and the rule exists for the reason driveID does: a path or a URL fragment
// must not reach a field a later run reads back.
var tabID = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,128}$`)

// Block is what gdoc owns in a markdown note. Everything outside it belongs to
// the author.
//
// Schema is what the note says, so a block read from a schema 1 note carries 1
// and is written back unchanged. Write renders schema 2 the moment anything in
// the block changes.
type Block struct {
	Schema    int     `yaml:"schema"`
	Documents []Entry `yaml:"documents"`
}

// Entry is one document this note is paired with: the document gdoc published
// from it, or a document it was exported from. A note names as many as it has
// been published to, and every writer acts on the one the URL names.
type Entry struct {
	ID              string           `yaml:"id"`
	FolderID        string           `yaml:"folder_id,omitempty"`
	TabID           string           `yaml:"tab_id,omitempty"`
	Published       *Published       `yaml:"published,omitempty"`
	Exported        *Exported        `yaml:"exported,omitempty"`
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

// Exported is what export records: when the document was read into the hub,
// and, in the note export wrote, the note it was read beside. A note carrying
// Note is a copy of somebody else's document rather than the source of one, and
// every writer refuses to act on it.
//
// One field is enough. The stamp on a paired note carries the time alone: the
// note is its own source, so there is no other note to name.
type Exported struct {
	At   time.Time `yaml:"at,omitempty"`
	Note string    `yaml:"note,omitempty"`
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

// Entry returns the entry for the document the URL names. A document the block
// does not name is an error naming every document it does, because the answer
// is always to open one of those instead.
func (b *Block) Entry(id string) (*Entry, error) {
	if b == nil {
		return nil, fmt.Errorf("gdoc front matter: no block")
	}
	for i := range b.Documents {
		if b.Documents[i].ID == id {
			return &b.Documents[i], nil
		}
	}
	return nil, fmt.Errorf("gdoc front matter: no entry names document %s; the block names %s",
		id, strings.Join(b.IDs(), ", "))
}

// IDs is every document the block names, in the order it names them.
func (b *Block) IDs() []string {
	if b == nil {
		return nil
	}
	out := make([]string, 0, len(b.Documents))
	for _, e := range b.Documents {
		out = append(out, e.ID)
	}
	return out
}

// Validate reports the first rule the block breaks, naming the key. A block
// that fails any rule is never written and never returned from a read.
func (b *Block) Validate() error {
	if b == nil {
		return fmt.Errorf("gdoc front matter: no block")
	}
	switch b.Schema {
	case 0:
		return fmt.Errorf("gdoc front matter: schema is required and must be %d", Schema)
	case SchemaOne:
		if len(b.Documents) != 1 {
			return fmt.Errorf("gdoc front matter: schema %d names one document, and this block names %d",
				SchemaOne, len(b.Documents))
		}
	case Schema:
		if len(b.Documents) == 0 {
			return fmt.Errorf("gdoc front matter: documents is required and names at least one document")
		}
	default:
		return fmt.Errorf("gdoc front matter: schema is %d, and this gdoc reads %d and %d", b.Schema, SchemaOne, Schema)
	}
	first := make(map[string]int, len(b.Documents))
	for i := range b.Documents {
		if err := b.validateEntry(i); err != nil {
			return err
		}
		id := b.Documents[i].ID
		if at, ok := first[id]; ok {
			return fmt.Errorf("gdoc front matter: %s names document %s, and %s names it too, so gdoc cannot tell which entry a run means",
				b.field(i, "id"), id, b.field(at, "id"))
		}
		first[id] = i
	}
	return nil
}

// validateEntry is one entry's own rules. Every message goes through field, so
// a schema 1 block names the keys that are in the file and a schema 2 block
// names which entry the key is in.
func (b *Block) validateEntry(i int) error {
	e := &b.Documents[i]
	if e.ID == "" {
		return fmt.Errorf("gdoc front matter: %s is required", b.field(i, "id"))
	}
	if !driveID.MatchString(e.ID) {
		return fmt.Errorf("gdoc front matter: %s %q is not a Drive id", b.field(i, "id"), e.ID)
	}
	if e.FolderID != "" && !driveID.MatchString(e.FolderID) {
		return fmt.Errorf("gdoc front matter: %s %q is not a Drive id", b.field(i, "folder_id"), e.FolderID)
	}
	if e.TabID != "" && !tabID.MatchString(e.TabID) {
		return fmt.Errorf("gdoc front matter: %s %q is not a Docs tab id", b.field(i, "tab_id"), e.TabID)
	}
	if e.Published != nil {
		if e.Published.At.IsZero() {
			return fmt.Errorf("gdoc front matter: %s is required", b.field(i, "published.at"))
		}
		if e.Published.Title == "" {
			return fmt.Errorf("gdoc front matter: %s is required", b.field(i, "published.title"))
		}
	}
	if e.Exported != nil && e.Exported.At.IsZero() && e.Exported.Note == "" {
		return fmt.Errorf("gdoc front matter: %s carries neither at nor note, and one of them says what was exported", b.field(i, "exported"))
	}
	if e.SuggestionsSeen != nil {
		for j, item := range e.SuggestionsSeen.Items {
			if item.ID == "" {
				return fmt.Errorf("gdoc front matter: %s is required", b.field(i, fmt.Sprintf("suggestions_seen.items[%d].id", j)))
			}
			if item.Kind != KindInsertion && item.Kind != KindDeletion {
				return fmt.Errorf("gdoc front matter: %s is %q, and it is %s or %s",
					b.field(i, fmt.Sprintf("suggestions_seen.items[%d].kind", j)), item.Kind, KindInsertion, KindDeletion)
			}
		}
	}
	for j, p := range e.Proposals {
		if p.ID == "" {
			return fmt.Errorf("gdoc front matter: %s is required", b.field(i, fmt.Sprintf("proposals[%d].id", j)))
		}
		if p.CommentID == "" {
			return fmt.Errorf("gdoc front matter: %s is required", b.field(i, fmt.Sprintf("proposals[%d].comment_id", j)))
		}
		if p.At.IsZero() {
			return fmt.Errorf("gdoc front matter: %s is required", b.field(i, fmt.Sprintf("proposals[%d].at", j)))
		}
	}
	return nil
}

// field is the key a refusal names. A schema 1 block names the keys that are in
// the file, because that is what the reader opens and sees; a schema 2 block
// names the entry too, because a note with four documents in it is otherwise
// four places to look.
func (b *Block) field(i int, name string) string {
	if b.Schema == SchemaOne {
		if name == "id" {
			return "document_id"
		}
		return name
	}
	return fmt.Sprintf("documents[%d].%s", i, name)
}
