// Package contents writes the contents list itself, instead of leaving it to a
// Word field.
//
// A .docx contents list is a Word field. It carries a cached result, the text
// some application computed the last time it laid the document out. Google Docs
// never refreshes that cached result on import: two documents left untouched
// kept the master template's placeholder contents list for over thirteen
// minutes, and only changed when a human clicked "Update table of contents".
//
// The master's cached result lists the template's own placeholder sections, with
// page numbers running past the end of the document. So a published document
// keeps a contents page describing a different document, indefinitely. The body
// is correct. Only the contents page lies, which makes it the worst kind of
// defect.
//
// So this package replaces the field's cached result, and keeps the field.
// Delete the field and Google imports plain text: the published document then
// has no table of contents at all, only text shaped like one.
//
// Page numbers come from the caller. The source is Google itself: upload once,
// read the pagination back out of Google's own PDF export, then write it in. A
// caller with no pagination to offer gets blank page numbers, because a blank is
// honest and a wrong number is not.
package contents

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"spike/gdocgo/internal/docx"
	"spike/gdocgo/internal/ooxml"
)

const (
	maxLevel = 3
	// Bookmark names must be unique and free of spaces. Prefixed so ours are
	// recognisable and so a rewrite never collides with the template's own.
	bookmarkPrefix = "_gdoc_toc"
)

// Explicit spacing on the entry styles, so a refresh does not visibly move them.
//
// Google imposes its own paragraph spacing when a reader refreshes the contents
// list, and only the font is inherited. Measured on a published document:
// entries sit at a 16 to 17pt pitch after a refresh. Declaring the tighter value
// up front means the page looks the same before and after.
const entrySpacing = `<w:spacing w:before="0" w:after="20" w:line="276" w:lineRule="auto"/>`

// Google leaves 3pt above the first entry only, not above every one. Word and
// Google both add space-before to space-after rather than collapsing them, so
// putting this on the style would widen every gap instead.
const firstEntrySpacing = `<w:spacing w:before="60" w:after="20" w:line="276" w:lineRule="auto"/>`

// The right tab is the text edge, which is what puts the page number at the
// margin. That is the page width less both margins, so it is UsableTwips by
// definition. Imported rather than repeated: a page size change has to move the
// tab with it or the page numbers stop sitting at the margin.
const rightTab = ooxml.UsableTwips

// One nesting step each, in twips. 360 and 720 are what Google's own refresh
// uses, measured off a published document. The house template used 283 and 567,
// which is close but visibly different, and a refresh would move the entries.
var levelIndents = [3]int{0, 360, 720}

func levelStyle(level, indent int) string {
	return fmt.Sprintf(
		`<w:style w:type="paragraph" w:styleId="TOC%d">`+
			`<w:name w:val="toc %d"/><w:basedOn w:val="Index"/><w:pPr><w:tabs>`+
			`<w:tab w:val="clear" w:pos="720"/>`+
			`<w:tab w:val="right" w:pos="%d" w:leader="dot"/></w:tabs>`+
			`%s<w:ind w:hanging="0" w:left="%d"/>`+
			`</w:pPr><w:rPr/></w:style>`,
		level, level, rightTab, entrySpacing, indent)
}

// tocStyles were measured off a LibreOffice-produced document, then frozen here.
// The entry styles declare no font and no weight of their own, so they inherit
// the document default, Calibri. Left to its own defaults Google renders a
// contents page in Arial with bold level-one entries, neither of which is house
// style.
func tocStyles() [][2]string {
	return [][2]string{
		{"Index", `<w:style w:type="paragraph" w:styleId="Index"><w:name w:val="Index"/>` +
			`<w:basedOn w:val="Normal"/><w:qFormat/>` +
			`<w:pPr><w:suppressLineNumbers/></w:pPr>` +
			`<w:rPr><w:rFonts w:cs="Arial Unicode MS"/></w:rPr></w:style>`},
		{"TOC1", levelStyle(1, levelIndents[0])},
		{"TOC2", levelStyle(2, levelIndents[1])},
		{"TOC3", levelStyle(3, levelIndents[2])},
		// Empty on purpose. It overrides Word's Hyperlink style, which would
		// otherwise render every entry blue and underlined.
		{"IndexLink", `<w:style w:type="character" w:styleId="IndexLink">` +
			`<w:name w:val="Index Link"/><w:qFormat/><w:rPr/></w:style>`},
	}
}

var (
	paragraphRE = regexp.MustCompile(`(?s)<w:p\b.*?</w:p>`)
	// Strict: <w:t> or <w:t attr=...>, never <w:instrText>, which a looser
	// pattern matches by accident and which would drop a field instruction into
	// the text.
	textRE    = regexp.MustCompile(`<w:t(?:\s[^>]*)?>([^<]*)</w:t>`)
	headingRE = regexp.MustCompile(`w:pStyle w:val="Heading(\d)"`)
	// Which field is the contents field, and where it stops.
	instructionRE = regexp.MustCompile(`(?s)<w:instrText[^>]*>(.*?)</w:instrText>`)
	fldCharRE     = regexp.MustCompile(`w:fldCharType="(begin|end)"`)
	runRE         = regexp.MustCompile(`(?s)<w:r\b.*?</w:r>`)

	// Attribute order is not fixed, and no writer agrees on it. Go's regexp
	// engine has no lookahead, so the tag is matched whole and its attributes
	// are read out of the match. A pattern anchored on one attribute order
	// finds none of the template's bookmarks, which then makes every id we pick
	// a duplicate of one of theirs.
	bookmarkStartRE = regexp.MustCompile(`<w:bookmarkStart\b[^>]*/>`)
	bookmarkEndRE   = regexp.MustCompile(`<w:bookmarkEnd\b[^>]*/>`)
	attrIDRE        = regexp.MustCompile(`\bw:id="(\d+)"`)
	attrNameRE      = regexp.MustCompile(`\bw:name="([^"]*)"`)
)

const tocInstruction = "TOC"
const beginMarker = `w:fldCharType="begin"`

// Error means the contents list could not be written, and guessing would be worse.
type Error struct{ msg string }

func (e *Error) Error() string { return e.msg }

func errorf(format string, args ...any) error { return &Error{msg: fmt.Sprintf(format, args...)} }

// Heading is one entry, as it will appear on the contents page.
type Heading struct {
	Level int
	Text  string
}

// Result is what a rewrite changed.
type Result struct {
	Entries             []Heading
	ReplacedParagraphs  int
}

// Escape XML-escapes a heading. Ampersands first, or the escapes get escaped.
func Escape(text string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(text)
}

// Unescape undoes XML escaping, so a heading is held as the characters a reader
// sees. Ampersand last, or an escape gets un-escaped twice: "&amp;lt;" would
// come back as "<" instead of "&lt;".
func Unescape(text string) string {
	for _, pair := range [][2]string{{"&lt;", "<"}, {"&gt;", ">"}, {"&quot;", `"`},
		{"&apos;", "'"}, {"&amp;", "&"}} {
		text = strings.ReplaceAll(text, pair[0], pair[1])
	}
	return text
}

// Headings lists every heading in document order.
//
// The text is unescaped here, because a <w:t> body is already escaped and the
// write site escapes again. Escaping twice puts "&amp;" on the contents page
// under a heading reading "&". It also breaks the page number: this text is the
// key the pagination lookup uses, and the PDF carries a literal ampersand.
func Headings(documentXML string) []Heading {
	var found []Heading
	for _, paragraph := range paragraphRE.FindAllString(documentXML, -1) {
		match := headingRE.FindStringSubmatch(paragraph)
		if match == nil {
			continue
		}
		var joined strings.Builder
		for _, piece := range textRE.FindAllStringSubmatch(paragraph, -1) {
			joined.WriteString(piece[1])
		}
		text := strings.TrimSpace(Unescape(joined.String()))
		if text == "" {
			continue
		}
		level, _ := strconv.Atoi(match[1])
		found = append(found, Heading{Level: level, Text: text})
	}
	return found
}

// field is where the contents field is, and the two runs that must survive a
// rewrite.
type field struct {
	start      int
	end        int
	beginRun   string
	endRun     string
	paragraphs int
}

func runCarrying(paragraphXML, marker string) (string, error) {
	for _, run := range runRE.FindAllString(paragraphXML, -1) {
		if strings.Contains(run, marker) {
			return run, nil
		}
	}
	return "", errorf("the contents field has no run carrying %s", marker)
}

// tocField finds the paragraph carrying the TOC instruction, and where its field
// opens.
//
// Chosen by the instruction naming TOC, never by being the first field in the
// body. A document re-saved from Word can carry a PAGE field earlier on, and
// splicing the entries over that leaves the real contents list where it is,
// gives the reader two contents lists, and reports success.
func tocField(paragraphs [][]int, documentXML string) (int, int) {
	for index, span := range paragraphs {
		text := documentXML[span[0]:span[1]]
		for _, loc := range instructionRE.FindAllStringSubmatchIndex(text, -1) {
			if !strings.Contains(text[loc[2]:loc[3]], tocInstruction) {
				continue
			}
			head := text[:loc[0]]
			offset := strings.LastIndex(head, beginMarker)
			if offset < 0 {
				offset = 0
			}
			return index, offset
		}
	}
	return -1, 0
}

// closingParagraph finds where the field opened at offset closes.
//
// Counted by begin and end nesting depth, not by the first end marker after the
// instruction. A cached result can hold fields of its own: Word writes a PAGEREF
// field into every entry whenever a human refreshes the contents list. Taking
// the first end then stops the field at the first entry, so the rest of the
// stale list survives the splice and the reader gets both lists.
func closingParagraph(paragraphs [][]int, documentXML string, first, offset int) int {
	depth := 0
	for index := first; index < len(paragraphs); index++ {
		text := documentXML[paragraphs[index][0]:paragraphs[index][1]]
		if index == first {
			text = text[offset:]
		}
		for _, match := range fldCharRE.FindAllStringSubmatch(text, -1) {
			if match[1] == "begin" {
				depth++
			} else {
				depth--
			}
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

// locateField finds the contents field.
//
// The master's field is malformed: the begin marker, the instruction and the
// separator all sit in a single run, where the format wants one run each. So the
// field cannot be found by matching runs. Find the paragraph carrying the TOC
// instruction instead, then run forward to the paragraph where the field closes.
//
// Google accepts the malformed run and still builds a real contents list from
// it, so it is carried across untouched rather than repaired.
func locateField(documentXML string) (field, error) {
	paragraphs := paragraphRE.FindAllStringIndex(documentXML, -1)
	first, offset := tocField(paragraphs, documentXML)
	if first < 0 {
		return field{}, errorf("no TOC field instruction found in this document, so " +
			"there is nothing to replace. Was the contents list already written?")
	}
	last := closingParagraph(paragraphs, documentXML, first, offset)
	if last < 0 {
		return field{}, errorf("the contents field is never closed: its begin and end " +
			"markers do not balance, so where the cached result stops is unknown")
	}
	beginRun, err := runCarrying(documentXML[paragraphs[first][0]:paragraphs[first][1]],
		`w:fldCharType="begin"`)
	if err != nil {
		return field{}, err
	}
	endRun, err := runCarrying(documentXML[paragraphs[last][0]:paragraphs[last][1]],
		`w:fldCharType="end"`)
	if err != nil {
		return field{}, err
	}
	return field{
		start:      paragraphs[first][0],
		end:        paragraphs[last][1],
		beginRun:   beginRun,
		endRun:     endRun,
		paragraphs: last - first + 1,
	}, nil
}

// bookmarked wraps a heading paragraph in a bookmark, so an entry can link to it.
func bookmarked(paragraphXML, name string, bookmarkID int) string {
	start := fmt.Sprintf(`<w:bookmarkStart w:id="%d" w:name="%s"/>`, bookmarkID, name)
	end := fmt.Sprintf(`<w:bookmarkEnd w:id="%d"/>`, bookmarkID)
	var out string
	if index := strings.Index(paragraphXML, "</w:pPr>"); index >= 0 {
		cut := index + len("</w:pPr>")
		out = paragraphXML[:cut] + start + paragraphXML[cut:]
	} else {
		cut := strings.Index(paragraphXML, ">") + 1
		out = paragraphXML[:cut] + start + paragraphXML[cut:]
	}
	closing := strings.LastIndex(out, "</w:p>")
	return out[:closing] + end + "</w:p>"
}

type span struct{ start, end int }

// stripOurBookmarks removes every bookmark this package wrote on an earlier pass.
//
// A rewrite must replace our own bookmarks rather than add to them. Bookmark
// names are unique document-wide, and a second write without this would leave
// two <w:bookmarkStart> tags carrying the same name, which is invalid OOXML.
// Each start tag is removed together with the end tag it pairs with. Anything
// not carrying our prefix, including the template's own bookmarks, is left alone.
func stripOurBookmarks(documentXML string) string {
	ends := map[string][]span{}
	for _, loc := range bookmarkEndRE.FindAllStringIndex(documentXML, -1) {
		tag := documentXML[loc[0]:loc[1]]
		if match := attrIDRE.FindStringSubmatch(tag); match != nil {
			ends[match[1]] = append(ends[match[1]], span{loc[0], loc[1]})
		}
	}

	var cuts []span
	claimed := map[span]bool{}
	for _, loc := range bookmarkStartRE.FindAllStringIndex(documentXML, -1) {
		tag := documentXML[loc[0]:loc[1]]
		name := attrNameRE.FindStringSubmatch(tag)
		id := attrIDRE.FindStringSubmatch(tag)
		if name == nil || !strings.HasPrefix(name[1], bookmarkPrefix) {
			continue
		}
		cuts = append(cuts, span{loc[0], loc[1]})
		if id == nil {
			continue
		}
		// Paired by position, never by a document-wide string replace. Ids are
		// supposed to be unique, but an earlier pass may have handed one of ours
		// the same number as one of the template's.
		for _, candidate := range ends[id[1]] {
			if candidate.start >= loc[1] && !claimed[candidate] {
				claimed[candidate] = true
				cuts = append(cuts, candidate)
				break
			}
		}
	}

	// Backwards, so each cut leaves earlier offsets untouched.
	sort.Slice(cuts, func(i, j int) bool { return cuts[i].start > cuts[j].start })
	for _, cut := range cuts {
		documentXML = documentXML[:cut.start] + documentXML[cut.end:]
	}
	return documentXML
}

// nextBookmarkID is one past the highest bookmark id in the document, ours or
// anyone else's. Ends are read as well as starts, so an id used only by a stray
// end tag still counts. A duplicate id is invalid OOXML, and the file still
// parses, so nothing would report it.
func nextBookmarkID(documentXML string) int {
	highest := 0
	for _, pattern := range []*regexp.Regexp{bookmarkStartRE, bookmarkEndRE} {
		for _, tag := range pattern.FindAllString(documentXML, -1) {
			if match := attrIDRE.FindStringSubmatch(tag); match != nil {
				if id, err := strconv.Atoi(match[1]); err == nil && id > highest {
					highest = id
				}
			}
		}
	}
	return highest + 1
}

// withBookmarks bookmarks every heading, returning the document and the anchors.
func withBookmarks(documentXML string, count int) (string, []string, error) {
	documentXML = stripOurBookmarks(documentXML)
	nextID := nextBookmarkID(documentXML)

	type edit struct {
		start, end  int
		replacement string
	}
	var edits []edit
	var anchors []string
	for _, loc := range paragraphRE.FindAllStringIndex(documentXML, -1) {
		paragraph := documentXML[loc[0]:loc[1]]
		if !headingRE.MatchString(paragraph) {
			continue
		}
		var joined strings.Builder
		for _, piece := range textRE.FindAllStringSubmatch(paragraph, -1) {
			joined.WriteString(piece[1])
		}
		if strings.TrimSpace(joined.String()) == "" {
			continue
		}
		name := fmt.Sprintf("%s%d", bookmarkPrefix, len(anchors)+1)
		anchors = append(anchors, name)
		edits = append(edits, edit{loc[0], loc[1],
			bookmarked(paragraph, name, nextID+len(anchors)-1)})
	}

	if len(anchors) != count {
		return "", nil, errorf("bookmarked %d headings but found %d, refusing to link them up",
			len(anchors), count)
	}
	// Backwards, so each splice leaves earlier offsets untouched.
	for i := len(edits) - 1; i >= 0; i-- {
		documentXML = documentXML[:edits[i].start] + edits[i].replacement +
			documentXML[edits[i].end:]
	}
	return documentXML, anchors, nil
}

// entry is one contents line: a styled paragraph, a tab, then the page number.
//
// No font and no weight are declared. That is deliberate: anything declared here
// overrides the template, and anything left undeclared inherits the document
// default. Google supplies Arial and bold only when nothing else does.
//
// prefix and suffix carry the field's begin and end runs on the first and last
// entries, which is what keeps the contents list a field rather than plain text.
//
// The paragraph clears the style's own right tab and sets its own one twip short
// of it. That is copied from observed LibreOffice output: LibreOffice writes the
// direct tab a twip inside the style's, and the numbers line up. A twip is
// 1/1440 of an inch, so the offset is invisible; it is kept because it is what
// was measured against live Google rendering, not because the reason is
// understood.
func entry(level int, text, page, anchor, prefix, suffix string, first bool) string {
	if level > maxLevel {
		level = maxLevel
	}
	if level < 1 {
		level = 1
	}
	spacing := ""
	if first {
		spacing = firstEntrySpacing
	}
	return fmt.Sprintf(
		`<w:p><w:pPr><w:pStyle w:val="TOC%d"/><w:tabs>`+
			`<w:tab w:val="clear" w:pos="%d"/>`+
			`<w:tab w:val="right" w:pos="%d" w:leader="dot"/>`+
			`</w:tabs>%s<w:rPr/></w:pPr>`+
			`%s`+
			`<w:hyperlink w:anchor="%s">`+
			`<w:r><w:rPr><w:rStyle w:val="IndexLink"/><w:webHidden/></w:rPr>`+
			`<w:t xml:space="preserve">%s</w:t>`+
			`<w:tab/><w:t>%s</w:t></w:r></w:hyperlink>`+
			`%s</w:p>`,
		level, rightTab, rightTab-1, spacing, prefix, anchor, Escape(text), page, suffix)
}

func withStyles(stylesXML string) string {
	for _, pair := range tocStyles() {
		if strings.Contains(stylesXML, `w:styleId="`+pair[0]+`"`) {
			continue
		}
		stylesXML = strings.Replace(stylesXML, "</w:styles>", pair[1]+"</w:styles>", 1)
	}
	return stylesXML
}

// Rewrite replaces the contents field in a package with real entries.
//
// pages maps heading text to a page number. A heading missing from it gets a
// blank page number rather than a guess.
func Rewrite(pkg *docx.Package, pages map[string]int) (*Result, error) {
	documentBytes, err := pkg.MustGet("word/document.xml")
	if err != nil {
		return nil, err
	}
	stylesBytes, err := pkg.MustGet("word/styles.xml")
	if err != nil {
		return nil, err
	}
	document := string(documentBytes)
	styles := string(stylesBytes)

	found := Headings(document)
	if len(found) == 0 {
		return nil, errorf("this document has no headings, so a contents list would be " +
			"empty. Refusing to write one.")
	}

	document, anchors, err := withBookmarks(document, len(found))
	if err != nil {
		return nil, err
	}
	located, err := locateField(document)
	if err != nil {
		return nil, err
	}

	var entries strings.Builder
	last := len(found) - 1
	for index, heading := range found {
		page := ""
		if number, ok := pages[heading.Text]; ok {
			page = strconv.Itoa(number)
		}
		prefix, suffix := "", ""
		if index == 0 {
			prefix = located.beginRun
		}
		if index == last {
			suffix = located.endRun
		}
		entries.WriteString(entry(heading.Level, heading.Text, page, anchors[index],
			prefix, suffix, index == 0))
	}

	document = document[:located.start] + entries.String() + document[located.end:]
	pkg.Set("word/document.xml", []byte(document))
	pkg.Set("word/styles.xml", []byte(withStyles(styles)))

	return &Result{Entries: found, ReplacedParagraphs: located.paragraphs}, nil
}
