// Package pagination asks Google which page each heading landed on.
//
// A contents list needs page numbers, and page numbers do not exist until
// something lays the document out. Google Docs is that something, and we are
// uploading to it anyway. So the publish flow uploads once with blank page
// numbers, exports the result as PDF, reads which page each heading actually
// sits on, writes those numbers into the contents list, and uploads the version
// it publishes. Google is the layout engine, which means the page numbers
// describe the document the reader is holding.
//
// The numbers come from the PDF's own outline, not from scraping its text.
// Google's export carries one outline entry per heading, each pointing at the
// page object the heading sits on, so the answer is read rather than inferred.
//
// Scraping was tried first and is a trap. A PDF holds glyphs at coordinates, not
// lines, so reassembling text means guessing where the lines were: the two
// libraries tried returned "Contents" as "d4 gnOia1-ehne eC stntno" and merged
// two headings into one row. Matching a heading against that either fails or,
// worse, matches the contents entry for the same heading and reports the
// contents page as the heading's own. The outline has no such failure mode.
package pagination

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"spike/gdocgo/internal/contents"
)

// Error means the rendered document could not be read, so page numbers are
// unavailable.
type Error struct{ msg string }

func (e *Error) Error() string { return e.msg }

func errorf(format string, args ...any) error { return &Error{msg: fmt.Sprintf(format, args...)} }

var whitespaceRE = regexp.MustCompile(`\s+`)

func Normalise(text string) string {
	return strings.TrimSpace(whitespaceRE.ReplaceAllString(text, " "))
}

// Entry is one outline entry: the heading Google saw, and the page it is on.
type Entry struct {
	Title string
	Page  int
}

// Resolve maps heading text to the 1-based page its heading sits on.
//
// Matched with a cursor that only moves forward, because both lists are in
// document order. That is what makes a document with two sections called
// "Overview" resolve to two different pages rather than twice to the first.
// A heading the outline never mentions is left out of the result rather than
// guessed at.
func Resolve(outline []Entry, headings []contents.Heading) map[string]int {
	found := map[string]int{}
	cursor := 0
	for _, heading := range headings {
		target := Normalise(heading.Text)
		for i := cursor; i < len(outline); i++ {
			if Normalise(outline[i].Title) != target {
				continue
			}
			found[heading.Text] = outline[i].Page
			cursor = i + 1
			break
		}
	}
	return found
}

// Outline reads the PDF's outline, flattened into document order.
func Outline(path string) ([]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errorf("no such file: %s", path)
	}
	defer file.Close()

	marks, err := api.Bookmarks(file, model.NewDefaultConfiguration())
	if err != nil {
		return nil, errorf("could not read the outline of %s: %v", path, err)
	}
	var out []Entry
	var walk func([]pdfcpu.Bookmark)
	walk = func(list []pdfcpu.Bookmark) {
		for _, mark := range list {
			out = append(out, Entry{Title: mark.Title, Page: mark.PageFrom})
			walk(mark.Kids)
		}
	}
	walk(marks)
	if len(out) == 0 {
		return nil, errorf("%s carries no outline, so which page each heading "+
			"sits on cannot be read", path)
	}
	return out, nil
}

func PageCount(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, errorf("no such file: %s", path)
	}
	defer file.Close()
	count, err := api.PageCount(file, model.NewDefaultConfiguration())
	if err != nil {
		return 0, errorf("could not read %s as a PDF: %v", path, err)
	}
	return count, nil
}

func FromPDF(path string, headings []contents.Heading) (map[string]int, error) {
	outline, err := Outline(path)
	if err != nil {
		return nil, err
	}
	return Resolve(outline, headings), nil
}

// Drift lists headings whose published page differs from the number we wrote.
//
// Empty means the contents list describes the document it sits in. Anything else
// means another pass is needed, and saying so is the whole point.
func Drift(written, published map[string]int) map[string][2]int {
	moved := map[string][2]int{}
	keys := map[string]bool{}
	for key := range written {
		keys[key] = true
	}
	for key := range published {
		keys[key] = true
	}
	for key := range keys {
		if written[key] != published[key] {
			moved[key] = [2]int{written[key], published[key]}
		}
	}
	return moved
}
