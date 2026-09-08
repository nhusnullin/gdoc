// Package cover reads the author's own front matter: the words that reach the
// cover page, the version-control table and the running head.
//
// Only title is required. Anything can go in the house template: a policy, a
// brief, a report, a set of notes. doc_type is free text and may be left out.
// Classification is the one value still validated, because it shades a fixed
// row in the front matter, so an unknown value would silently shade nothing.
//
// The keys are v1's, so a note written for the Python tool publishes here with
// no edits. Two rules follow from that:
//
// A key this package does not read is carried, never refused. The note is the
// author's file and gdoc owns one key in it. In particular the gdoc: block is
// internal/frontmatter's, in either v1's string shape or v2's mapping, and is
// skipped here whatever it holds.
//
// A missing title is a refusal carrying a candidate rather than a title this
// package invented. The skill proposes the candidate and writes it into the
// note once the author agrees.
//
// Nothing here reaches the network and nothing here writes a file.
package cover

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

// Classifications maps what an author may write to what the front matter says.
// The right-hand values are the labels the house file's classification table
// spells out, so a value outside this set would shade no row at all.
var Classifications = map[string]string{
	"confidential": "Confidential (C)",
	"restricted":   "Restricted (R)",
	"internal":     "Internal (I)",
	"public":       "Public (P)",
}

// DefaultVersion is what a note that states no version publishes as.
const DefaultVersion = "1.0"

// DefaultClassification is the key a note that states none is read as.
const DefaultClassification = "internal"

// runningHeadPrefix opens the running head in the page header.
const runningHeadPrefix = "Altery - "

// gdocKey is internal/frontmatter's key. This package skips it and never
// validates it: two readers of one block are two rules that drift.
const gdocKey = "gdoc"

// revisionFields are the seven columns of the revision-history table, in the
// order the columns run.
var revisionFields = []string{"version", "date", "author", "approved_by",
	"approval_date", "section", "change"}

var (
	datePrefixRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)
	headingRE    = regexp.MustCompile(`^#\s+(.+?)\s*#*\s*$`)
	fenceRE      = regexp.MustCompile("^\\s*(```|~~~)")
	isoDateRE    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)
)

// delimiters open and close YAML front matter, the same two internal/frontmatter
// reads: "..." closes a YAML document too, and a file that uses it is still
// front matter. A byte order mark in front of the opening line is an editor's,
// and the front matter behind it is still front matter.
const (
	openDelimiter  = "---"
	closeAlternate = "..."
	bom            = "\uFEFF"
)

// Revision is one row of the revision-history table.
type Revision struct {
	Version      string
	Date         string
	Author       string
	ApprovedBy   string
	ApprovalDate string
	Section      string
	Change       string
}

// Values returns the seven cells in the order the table's columns run.
func (r Revision) Values() []string {
	return []string{r.Version, r.Date, r.Author, r.ApprovedBy, r.ApprovalDate,
		r.Section, r.Change}
}

// Fields is one note's front matter, read and validated.
//
// The three titles the shell needs are methods rather than fields, so a caller
// cannot hand the cover one title and the header another.
type Fields struct {
	Title            string
	AltTitle         string
	DocType          string
	Version          string
	Date             string
	Owner            string
	Classification   string
	HeadingNumbering bool
	Revisions        []Revision
}

// CoverTitle joins the title to its type: "Third Party Risk" plus "Policy".
// A title that already names its own type is left alone, so nobody publishes a
// cover reading "Third Party Risk Policy Policy".
func (f Fields) CoverTitle() string { return join(f.Title, f.DocType) }

// CoverAltTitle is CoverTitle for the alternative title, empty when the note
// states none.
func (f Fields) CoverAltTitle() string { return join(f.AltTitle, f.DocType) }

// RunningHead is what the page header carries: the alternative title when the
// note states one, else the title. A note with no title has no running head,
// because "Altery - " on its own names nothing.
func (f Fields) RunningHead() string {
	head := f.CoverAltTitle()
	if head == "" {
		head = f.CoverTitle()
	}
	if head == "" {
		return ""
	}
	return runningHeadPrefix + head
}

func join(title, docType string) string {
	if title == "" {
		return ""
	}
	if docType == "" || strings.HasSuffix(strings.ToLower(title), strings.ToLower(docType)) {
		return title
	}
	return title + " " + docType
}

// MissingTitle is the note with no title. It carries a candidate rather than
// using it: the cover title is the author's decision, made once and then
// written into the note.
type MissingTitle struct {
	Candidate string
	Source    string // "h1", "filename", or "" when there was nothing to draw on
}

func (e *MissingTitle) Error() string {
	if e.Candidate == "" {
		return "no title in front matter, so the cover and the running head would be blank"
	}
	return fmt.Sprintf("no title in front matter, so the cover and the running head "+
		"would be blank. The %s suggests %q", e.Source, e.Candidate)
}

// TitleCandidate returns a title to propose and where it came from, "h1" or
// "filename". It is deterministic, so it can be tested and so two runs over one
// note propose the same words.
//
// A heading inside a fenced code block is not a candidate: it is somebody's
// example, not the document's name.
func TitleCandidate(bodyMarkdown, sourceName string) (string, string) {
	inFence := false
	for _, line := range strings.Split(bodyMarkdown, "\n") {
		if fenceRE.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if m := headingRE.FindStringSubmatch(line); m != nil {
			return strings.TrimSpace(m[1]), "h1"
		}
	}
	if sourceName == "" {
		return "", ""
	}
	stem := strings.TrimSuffix(filepath.Base(sourceName), filepath.Ext(sourceName))
	words := datePrefixRE.ReplaceAllString(stem, "")
	words = strings.TrimSpace(strings.NewReplacer("-", " ", "_", " ").Replace(words))
	if words == "" {
		return "Untitled", "filename"
	}
	return strings.ToUpper(words[:1]) + words[1:], "filename"
}

// Read returns the note's fields and the markdown behind its front matter.
//
// The body comes back even when the title is missing, so the caller, which
// knows the file's name, can ask TitleCandidate for a better proposal than one
// drawn from the body alone.
func Read(src []byte) (Fields, []byte, error) {
	block, body := split(string(src))

	fields, err := mapping(block)
	if err != nil {
		return Fields{}, []byte(body), err
	}

	title := strings.TrimSpace(text(fields["title"]))
	if title == "" {
		candidate, source := TitleCandidate(body, "")
		return Fields{}, []byte(body), &MissingTitle{Candidate: candidate, Source: source}
	}

	classification, err := readClassification(fields)
	if err != nil {
		return Fields{}, []byte(body), err
	}
	numbering, err := readNumbering(fields)
	if err != nil {
		return Fields{}, []byte(body), err
	}
	revisions, err := readRevisions(fields["revisions"])
	if err != nil {
		return Fields{}, []byte(body), err
	}

	f := Fields{
		Title:            title,
		AltTitle:         strings.TrimSpace(text(fields["alt_title"])),
		DocType:          strings.TrimSpace(text(fields["doc_type"])),
		Version:          strings.TrimSpace(text(fields["version"])),
		Date:             strings.TrimSpace(text(fields["date"])),
		Owner:            strings.TrimSpace(text(fields["owner"])),
		Classification:   classification,
		HeadingNumbering: numbering,
		Revisions:        revisions,
	}
	if f.Version == "" {
		f.Version = DefaultVersion
	}
	return f, []byte(body), nil
}

// readClassification turns what the author wrote into the label the front
// matter shades. "Restricted (R)" written out in full is accepted, because that
// is what the table itself says and somebody will copy it.
func readClassification(fields map[string]ast.Node) (string, error) {
	written := strings.TrimSpace(text(fields["classification"]))
	key := written
	if key == "" {
		key = DefaultClassification
	}
	key = strings.ToLower(key)
	key, _, _ = strings.Cut(key, " (")
	label, ok := Classifications[key]
	if !ok {
		return "", fmt.Errorf("classification %q is not one of %v", written, names())
	}
	return label, nil
}

// names lists the classification keys in a fixed order, so the refusal reads
// the same on every run.
func names() []string {
	out := make([]string, 0, len(Classifications))
	for name := range Classifications {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// readNumbering reads heading_numbering. v1 wrote "auto" and "none"; a boolean
// is the shape a person guesses at, so both are read and anything else is
// refused naming what was written.
func readNumbering(fields map[string]ast.Node) (bool, error) {
	written := strings.TrimSpace(text(fields["heading_numbering"]))
	switch strings.ToLower(written) {
	case "", "auto", "true", "yes", "on":
		return true, nil
	case "none", "false", "no", "off":
		return false, nil
	}
	return false, fmt.Errorf("heading_numbering %q is not one of [auto none true false]", written)
}

// readRevisions reads the revision-history rows. A row needs its version, which
// is the row's own name; every other cell may be blank. An unknown field is
// refused, because a misspelled column is a cell that silently never prints.
func readRevisions(node ast.Node) ([]Revision, error) {
	if node == nil || isNull(node) {
		return nil, nil
	}
	seq, ok := node.(*ast.SequenceNode)
	if !ok {
		return nil, fmt.Errorf("'revisions' must be a list of entries")
	}
	allowed := map[string]bool{}
	for _, name := range revisionFields {
		allowed[name] = true
	}
	var out []Revision
	for index, entry := range seq.Values {
		row, err := values(entry)
		if err != nil {
			return nil, fmt.Errorf("revision %d must be a block of key: value pairs", index+1)
		}
		var unknown []string
		cells := map[string]string{}
		for name, value := range row {
			if !allowed[name] {
				unknown = append(unknown, name)
				continue
			}
			cells[name] = strings.TrimSpace(text(value))
		}
		if len(unknown) > 0 {
			sort.Strings(unknown)
			return nil, fmt.Errorf("revision %d has unknown field(s) %v. Allowed: %v",
				index+1, unknown, revisionFields)
		}
		if cells["version"] == "" {
			return nil, fmt.Errorf("revision %d has no 'version', so the row has no name", index+1)
		}
		out = append(out, Revision{
			Version: cells["version"], Date: cells["date"], Author: cells["author"],
			ApprovedBy: cells["approved_by"], ApprovalDate: cells["approval_date"],
			Section: cells["section"], Change: cells["change"],
		})
	}
	return out, nil
}

// split cuts a markdown file into its front-matter block and its body. A file
// with no front matter, or one that opens a block and never closes it, is all
// body: writing a second block in front of the author's keys is what refusing
// the second case avoids, and that is internal/frontmatter's rule too.
func split(src string) (block, body string) {
	rest := strings.TrimPrefix(src, bom)
	lines := splitLines(rest)
	if len(lines) == 0 || !isDelimiter(lines[0], openDelimiter) {
		return "", src
	}
	for i := 1; i < len(lines); i++ {
		if isDelimiter(lines[i], openDelimiter) || isDelimiter(lines[i], closeAlternate) {
			return strings.Join(lines[1:i], ""), strings.Join(lines[i+1:], "")
		}
	}
	return "", src
}

// splitLines cuts text into lines that each carry their own terminator, so
// joining a slice of them reproduces the bytes exactly.
func splitLines(text string) []string {
	var out []string
	for len(text) > 0 {
		at := strings.IndexByte(text, '\n')
		if at < 0 {
			out = append(out, text)
			break
		}
		out = append(out, text[:at+1])
		text = text[at+1:]
	}
	return out
}

func isDelimiter(line, want string) bool {
	return strings.TrimRight(line, " \t\r\n") == want
}

// mapping parses the front-matter block into its top-level keys. An empty block
// is a note with no keys rather than an error, so a file opening with "---\n---"
// reads as a note with no title.
func mapping(block string) (map[string]ast.Node, error) {
	if strings.TrimSpace(block) == "" {
		return map[string]ast.Node{}, nil
	}
	file, err := parser.ParseBytes([]byte(block), 0)
	if err != nil {
		return nil, fmt.Errorf("the front matter is not valid YAML: %w", err)
	}
	if len(file.Docs) == 0 || file.Docs[0].Body == nil {
		return map[string]ast.Node{}, nil
	}
	out, err := values(file.Docs[0].Body)
	if err != nil {
		return nil, fmt.Errorf("the front matter must be a block of key: value pairs")
	}
	delete(out, gdocKey)
	return out, nil
}

// values reads a mapping node into its keys. goccy gives a single-pair mapping
// its own node type, so both shapes are read here rather than at four call
// sites.
func values(node ast.Node) (map[string]ast.Node, error) {
	out := map[string]ast.Node{}
	switch n := node.(type) {
	case *ast.MappingNode:
		for _, pair := range n.Values {
			out[pair.Key.GetToken().Value] = pair.Value
		}
	case *ast.MappingValueNode:
		out[n.Key.GetToken().Value] = n.Value
	default:
		return nil, fmt.Errorf("not a mapping")
	}
	return out, nil
}

func isNull(node ast.Node) bool {
	_, ok := node.(*ast.NullNode)
	return ok
}

// text renders a scalar as the string a reader expects.
//
// The token's own text is read rather than a decoded value, which is what keeps
// version 1.0 from publishing as "1". A date is the one value rewritten: YAML
// reads 2026-09-08 as a date, and the cover states it in UK long form.
func text(node ast.Node) string {
	if node == nil || isNull(node) {
		return ""
	}
	switch node.(type) {
	case *ast.MappingNode, *ast.MappingValueNode, *ast.SequenceNode:
		return ""
	}
	token := node.GetToken()
	if token == nil {
		return ""
	}
	return longDate(token.Value)
}

// longDate turns an ISO date into "8 September 2026" and leaves everything else
// as the author wrote it.
func longDate(value string) string {
	if !isoDateRE.MatchString(value) {
		return value
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return fmt.Sprintf("%d %s", parsed.Day(), parsed.Format("January 2006"))
		}
	}
	return value
}
