// Package render puts markdown into a house template and produces a .docx.
//
// The pipeline: parse the front matter, copy the master and fill its front
// matter, splice the rendered body in, then write the contents list.
//
// The master is copied and edited, never rebuilt. That is what keeps the cover,
// logo, running head, footer and coloured tables pixel-identical to the original.
//
// Nothing here calls an external program. Page numbers are the caller's to
// supply, because they do not exist until something lays the document out. A
// caller with none gets a contents list with correct entries and blank page
// numbers, which is honest.
package render

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"spike/gdocgo/internal/body"
	"spike/gdocgo/internal/contents"
	"spike/gdocgo/internal/frontmatter"
	"spike/gdocgo/internal/shell"
)

// Error is anything that should stop the build with a message rather than a stack.
type Error struct{ msg string }

func (e *Error) Error() string { return e.msg }

func errorf(format string, args ...any) error { return &Error{msg: fmt.Sprintf(format, args...)} }

// Result is what one build produced.
type Result struct {
	DocxPath string
	Title    string
	Template string
	Blocks   int
	Entries  int
	Headings []contents.Heading
}

// Options are the caller's choices for one build.
type Options struct {
	Template   string
	Title      string
	Pages      map[string]int
	Hyperlinks bool
}

var (
	nonSlug   = regexp.MustCompile(`[^a-z0-9]+`)
	dashes    = regexp.MustCompile(`-+`)
	asciiOnly = transform.Chain(norm.NFKD, runes.Remove(runes.In(unicodeMarks)))
)

func Slugify(text string) string {
	folded, _, err := transform.String(asciiOnly, text)
	if err != nil {
		folded = text
	}
	var ascii strings.Builder
	for _, r := range folded {
		if r < 128 {
			ascii.WriteRune(r)
		}
	}
	slug := nonSlug.ReplaceAllString(strings.ToLower(ascii.String()), "-")
	return strings.Trim(dashes.ReplaceAllString(slug, "-"), "-")
}

// DefaultOutput names the output YYYY-MM-DD-<title-slug>.docx, beside the source.
func DefaultOutput(meta *frontmatter.Meta, source string) string {
	datePrefix := os.Getenv("DOC_DATE")
	if datePrefix == "" {
		datePrefix = time.Now().Format("2006-01-02")
	}
	stem := Slugify(meta.CoverTitle)
	if stem == "" {
		stem = strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	}
	absolute, err := filepath.Abs(source)
	if err != nil {
		absolute = source
	}
	return filepath.Join(filepath.Dir(absolute), datePrefix+"-"+stem+".docx")
}

// DropTitleHeading removes a leading H1 that repeats the title.
//
// Without this the title prints twice, once on the cover and again above the
// first paragraph. Keyed on the text matching, so it also helps a hand-written
// front matter whose title repeats the note's own heading.
//
// Matches either the raw front matter title or the cover title, because
// CoverTitle can append DocType and a user copying what they see on the cover
// types the longer form.
func DropTitleHeading(bodyMarkdown, title, coverTitle string) string {
	candidates := map[string]bool{strings.TrimSpace(title): true}
	if coverTitle != "" {
		candidates[strings.TrimSpace(coverTitle)] = true
	}
	lines := strings.Split(bodyMarkdown, "\n")
	for index, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		match := frontmatter.H1RE.FindStringSubmatch(line)
		if match != nil && candidates[strings.TrimSpace(match[1])] {
			return strings.TrimLeft(strings.Join(lines[index+1:], "\n"), "\n")
		}
		return bodyMarkdown
	}
	return bodyMarkdown
}

// Build renders one markdown file through the house template.
func Build(mdPath, outPath string, opts Options) (*Result, error) {
	sourcePath, err := filepath.Abs(mdPath)
	if err != nil {
		return nil, errorf("input file not found: %s", mdPath)
	}
	if info, err := os.Stat(sourcePath); err != nil || info.IsDir() {
		return nil, errorf("input file not found: %s", sourcePath)
	}
	if opts.Template == "" {
		return nil, errorf("build needs a template")
	}
	if _, err := os.Stat(opts.Template); err != nil {
		return nil, errorf("template not found: %s", opts.Template)
	}

	markdownBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	meta, bodyMarkdown, err := frontmatter.Parse(string(markdownBytes), opts.Title,
		filepath.Base(sourcePath))
	if err != nil {
		return nil, err
	}
	bodyMarkdown = DropTitleHeading(bodyMarkdown, meta.Title, meta.CoverTitle)
	if strings.TrimSpace(bodyMarkdown) == "" {
		return nil, errorf("the file has front matter but no body content")
	}

	if outPath == "" {
		outPath = DefaultOutput(meta, sourcePath)
	}
	outPath, err = filepath.Abs(outPath)
	if err != nil {
		return nil, err
	}

	built, err := shell.Build(opts.Template, meta)
	if err != nil {
		return nil, err
	}

	blocks, err := body.Render(built.Package, built.Document, built.Body, built.Numbering,
		bodyMarkdown, body.Options{
			HeadingNumbering: meta.HeadingNumbering,
			BaseDir:          filepath.Dir(sourcePath),
			Hyperlinks:       opts.Hyperlinks,
		})
	if err != nil {
		return nil, err
	}

	documentBytes, err := built.Document.WriteToBytes()
	if err != nil {
		return nil, err
	}
	built.Package.Set("word/document.xml", documentBytes)

	numberingBytes, err := built.Numbering.WriteToBytes()
	if err != nil {
		return nil, err
	}
	built.Package.Set("word/numbering.xml", numberingBytes)

	written, err := contents.Rewrite(built.Package, opts.Pages)
	if err != nil {
		return nil, err
	}

	if err := built.Package.Save(outPath); err != nil {
		return nil, err
	}

	return &Result{
		DocxPath: outPath,
		Title:    meta.CoverTitle,
		Template: opts.Template,
		Blocks:   blocks,
		Entries:  len(written.Entries),
		Headings: written.Entries,
	}, nil
}
