// Package frontmatter parses and validates the YAML front matter of an Altery
// document's Markdown file.
//
// Only title is required. Anything can go in the house template: a policy, a
// brief, a report, a set of notes. doc_type is free text and may be left out.
//
// Classification is the one value still validated, because it shades a fixed row
// in the template, so an unknown value would silently shade nothing.
package frontmatter

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var Classifications = map[string]string{
	"confidential": "Confidential (C)",
	"restricted":   "Restricted (R)",
	"internal":     "Internal (I)",
	"public":       "Public (P)",
}

const DefaultVersion = "1.0"

var revisionFields = []string{"version", "date", "author", "approved_by",
	"approval_date", "section", "change"}

var (
	frontMatterRE = regexp.MustCompile(`(?s)\A---[ \t]*\n(.*?)\n---[ \t]*\n`)
	datePrefixRE  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)
	H1RE          = regexp.MustCompile(`^#\s+(.+?)\s*#*\s*$`)
	fenceRE       = regexp.MustCompile("^\\s*(```|~~~)")
)

// Error is anything that should stop the build with a message rather than a stack.
type Error struct{ msg string }

func (e *Error) Error() string { return e.msg }

func errorf(format string, args ...any) error {
	return &Error{msg: fmt.Sprintf(format, args...)}
}

// MissingTitle carries a candidate rather than using it. The tool does not
// invent a cover title silently; the skill proposes this one and writes it into
// the note once the author agrees.
type MissingTitle struct {
	Candidate string
	Source    string // "h1" or "filename"
}

func (e *MissingTitle) Error() string {
	return "no title in front matter, so the cover and running head would be blank"
}

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

// Values returns the seven cells in the order the template's columns run.
func (r Revision) Values() []string {
	return []string{r.Version, r.Date, r.Author, r.ApprovedBy, r.ApprovalDate,
		r.Section, r.Change}
}

// Meta is the front matter as plain strings, ready for the template filler.
type Meta struct {
	Title            string
	AltTitle         string
	DocType          string
	Version          string
	Date             string
	Owner            string
	LastApproval     string
	ReviewFrequency  string
	BoardRatification string
	Distribution     string
	Classification   string
	HeadingNumbering string
	Revisions        []Revision
	CoverTitle       string
	CoverAltTitle    string
	RunningHead      string
}

// TitleCandidate returns a title to propose and where it came from.
// Deterministic, so it can be tested.
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
		if m := H1RE.FindStringSubmatch(line); m != nil {
			return strings.TrimSpace(m[1]), "h1"
		}
	}
	stem := ""
	if sourceName != "" {
		stem = strings.TrimSuffix(filepath.Base(sourceName), filepath.Ext(sourceName))
	}
	words := datePrefixRE.ReplaceAllString(stem, "")
	words = strings.TrimSpace(strings.NewReplacer("-", " ", "_", " ").Replace(words))
	if words == "" {
		return "Untitled", "filename"
	}
	return strings.ToUpper(words[:1]) + words[1:], "filename"
}

// asText renders a YAML scalar as the string a reader expects.
//
// Reading the node rather than a decoded value is what keeps "1.0" from
// becoming "1": the raw scalar text is already exactly what the author typed.
// A date is the one exception, rendered in UK long form rather than ISO.
func asText(node *yaml.Node) string {
	if node == nil || node.Tag == "!!null" {
		return ""
	}
	if node.Tag == "!!timestamp" {
		for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05"} {
			if parsed, err := time.Parse(layout, node.Value); err == nil {
				return fmt.Sprintf("%d %s", parsed.Day(), parsed.Format("January 2006"))
			}
		}
	}
	return node.Value
}

// Split returns the front matter node map and the body markdown.
func Split(markdownText string) (map[string]*yaml.Node, string, error) {
	loc := frontMatterRE.FindStringSubmatchIndex(markdownText)
	if loc == nil {
		return nil, "", errorf("%s", noFrontMatter())
	}
	block := markdownText[loc[2]:loc[3]]

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(block), &root); err != nil {
		return nil, "", errorf("the front matter is not valid YAML: %v", err)
	}
	fields := map[string]*yaml.Node{}
	if len(root.Content) == 0 {
		return fields, markdownText[loc[1]:], nil
	}
	mapping := root.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil, "", errorf("the front matter must be a block of key: value pairs")
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		fields[mapping.Content[i].Value] = mapping.Content[i+1]
	}
	return fields, markdownText[loc[1]:], nil
}

func noFrontMatter() string {
	optional := []string{"alt_title", "doc_type", "version", "date", "owner",
		"last_approval", "review_frequency", "board_ratification", "distribution",
		"classification", "heading_numbering", "revisions"}
	return "no YAML front matter found. The file must start with a '---' line, the " +
		"metadata block, then a closing '---' line. Required: [title]. Optional: [" +
		strings.Join(optional, " ") + "]. The shortest note that publishes is three " +
		"lines: '---', 'title: Some Title', '---'."
}

func field(fields map[string]*yaml.Node, name string) string {
	return asText(fields[name])
}

func fieldOr(fields map[string]*yaml.Node, name, fallback string) string {
	if node, ok := fields[name]; ok && node.Tag != "!!null" {
		return asText(node)
	}
	return fallback
}

func validateRevisions(node *yaml.Node) ([]Revision, error) {
	if node == nil || node.Tag == "!!null" {
		return nil, nil
	}
	if node.Kind != yaml.SequenceNode {
		return nil, errorf("'revisions' must be a list of entries")
	}
	allowed := map[string]bool{}
	for _, name := range revisionFields {
		allowed[name] = true
	}
	var out []Revision
	for index, entry := range node.Content {
		if entry.Kind != yaml.MappingNode {
			return nil, errorf("revision %d must be a block of key: value pairs", index+1)
		}
		values := map[string]string{}
		var unknown []string
		for i := 0; i+1 < len(entry.Content); i += 2 {
			key := entry.Content[i].Value
			if !allowed[key] {
				unknown = append(unknown, key)
				continue
			}
			values[key] = asText(entry.Content[i+1])
		}
		if len(unknown) > 0 {
			sort.Strings(unknown)
			return nil, errorf("revision %d has unknown field(s) %v. Allowed: %v",
				index+1, unknown, revisionFields)
		}
		out = append(out, Revision{
			Version: values["version"], Date: values["date"], Author: values["author"],
			ApprovedBy: values["approved_by"], ApprovalDate: values["approval_date"],
			Section: values["section"], Change: values["change"],
		})
	}
	return out, nil
}

// Parse validates the front matter and returns the meta and the body markdown.
//
// sourceName is only used to suggest a title when there is none, so a caller
// with nothing to offer can leave it out and still get the suggestion drawn
// from the first heading.
func Parse(markdownText, titleOverride, sourceName string) (*Meta, string, error) {
	fields, body, err := Split(markdownText)
	if err != nil {
		return nil, "", err
	}
	title := strings.TrimSpace(field(fields, "title"))
	if titleOverride != "" {
		title = strings.TrimSpace(titleOverride)
	}
	if title == "" {
		candidate, source := TitleCandidate(body, sourceName)
		return nil, "", &MissingTitle{Candidate: candidate, Source: source}
	}

	// Free text, and optional. Whatever the author writes reaches the cover as
	// written, so "PRD" or "Board Submission" is not mangled into "Prd".
	docType := strings.TrimSpace(field(fields, "doc_type"))

	key := strings.ToLower(strings.TrimSpace(fieldOr(fields, "classification", "Internal")))
	key, _, _ = strings.Cut(key, " (")
	classification, ok := Classifications[key]
	if !ok {
		names := make([]string, 0, len(Classifications))
		for name := range Classifications {
			names = append(names, name)
		}
		sort.Strings(names)
		return nil, "", errorf("classification %q is not recognised. Use one of %v",
			field(fields, "classification"), names)
	}

	numbering := strings.ToLower(strings.TrimSpace(fieldOr(fields, "heading_numbering", "auto")))
	if numbering != "auto" && numbering != "none" {
		return nil, "", errorf("heading_numbering must be 'auto' or 'none'")
	}

	revisions, err := validateRevisions(fields["revisions"])
	if err != nil {
		return nil, "", err
	}

	version := strings.TrimSpace(field(fields, "version"))
	if version == "" {
		version = DefaultVersion
	}
	date := strings.TrimSpace(field(fields, "date"))
	if date == "" {
		date = time.Now().Format("January 2006")
	}

	meta := &Meta{
		Title:             title,
		AltTitle:          strings.TrimSpace(field(fields, "alt_title")),
		DocType:           docType,
		Version:           version,
		Date:              date,
		Owner:             field(fields, "owner"),
		LastApproval:      field(fields, "last_approval"),
		ReviewFrequency:   fieldOr(fields, "review_frequency", "Annually"),
		BoardRatification: field(fields, "board_ratification"),
		Distribution:      field(fields, "distribution"),
		Classification:    classification,
		HeadingNumbering:  numbering,
		Revisions:         revisions,
	}
	meta.CoverTitle = coverTitle(meta.Title, docType)
	if meta.AltTitle != "" {
		meta.CoverAltTitle = coverTitle(meta.AltTitle, docType)
	}
	meta.RunningHead = "Altery - " + meta.CoverTitle
	return meta, body, nil
}

// coverTitle joins a title to its type: "Third Party Risk" + "Policy".
// A title that already names its own type is left alone, so nobody ends up with
// a cover reading "Third Party Risk Policy Policy".
func coverTitle(title, docType string) string {
	if title == "" {
		return ""
	}
	if docType == "" || strings.HasSuffix(strings.ToLower(title), strings.ToLower(docType)) {
		return title
	}
	return title + " " + docType
}
