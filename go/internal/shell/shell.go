// Package shell turns the master template into a filled "shell": front matter
// completed, sample body removed, ready for generated content to be spliced in.
//
// Everything the eye recognises as the Altery house document lives in the front
// matter and the page furniture, so none of it is rebuilt here. The template is
// copied and edited in place, which is what keeps the output pixel-identical to
// the master.
package shell

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/beevik/etree"

	"spike/gdocgo/internal/docx"
	"spike/gdocgo/internal/frontmatter"
	"spike/gdocgo/internal/ooxml"
)

// The template's own "highlight as appropriate" mechanism for the Document
// Classification table is cell shading, not a run highlight. Each class has its
// own colour on the label cell; marking a class means painting the description
// cell to match.
var classFills = map[string]string{
	"Confidential (C)": "f4cccc",
	"Restricted (R)":   "fce5cd",
	"Internal (I)":     "fff2cc",
	"Public (P)":       "d9ead3",
}

var versionControlLabels = map[string]func(*frontmatter.Meta) string{
	"Document Owner":          func(m *frontmatter.Meta) string { return m.Owner },
	"Date of Last Approval":   func(m *frontmatter.Meta) string { return m.LastApproval },
	"Review Frequency":        func(m *frontmatter.Meta) string { return m.ReviewFrequency },
	"Board Ratification Date": func(m *frontmatter.Meta) string { return m.BoardRatification },
	"Policy Distribution":     func(m *frontmatter.Meta) string { return m.Distribution },
}

const (
	titlePlaceholder       = "(Name of) Framework/Policy"
	versionPlaceholder     = "Version: 1.0"
	datePlaceholder        = "May 2025"
	runningHeadPlaceholder = "Altery - xxx Policy"
	black                  = "000000"
)

// Error means the master template is not shaped the way this code expects.
type Error struct{ msg string }

func (e *Error) Error() string { return e.msg }

func errorf(format string, args ...any) error {
	return &Error{msg: fmt.Sprintf(format, args...)}
}

// --------------------------------------------------------------- text edits --

// clearPlaceholderMarks strips the yellow highlight and the red instruction
// colour from a run. In the master, yellow marks "a human must fill this in"
// and red marks "this is guidance, not content". Once we have written a real
// value both marks are actively misleading, so they go.
func clearPlaceholderMarks(run *etree.Element) {
	rPr := run.SelectElement("w:rPr")
	if rPr == nil {
		return
	}
	for _, highlight := range rPr.SelectElements("w:highlight") {
		rPr.RemoveChild(highlight)
	}
	for _, color := range rPr.SelectElements("w:color") {
		color.CreateAttr("w:val", black)
	}
}

// ReplaceInParagraph replaces old with new even when it is split across runs.
//
// Google Docs starts a new run wherever formatting changes, so
// "(Name of) Framework/Policy" is really two runs (yellow "(Name of) " plus red
// "Framework/Policy") and "Version: 1.0" is "Version: " plus a highlighted
// "1.0". Matching run by run therefore finds nothing. Match the joined paragraph
// text instead, write the replacement into the first run that overlaps the
// match, and blank the overlap out of the rest. Writing into an existing run is
// what preserves the font, size and centring.
func ReplaceInParagraph(paragraph *etree.Element, old, replacement string) bool {
	runs := ooxml.Runs(paragraph)
	if len(runs) == 0 {
		return false
	}
	texts := make([]string, len(runs))
	var joined strings.Builder
	for i, run := range runs {
		texts[i] = ooxml.RunText(run)
		joined.WriteString(texts[i])
	}
	start := strings.Index(joined.String(), old)
	if start < 0 {
		return false
	}
	end := start + len(old)

	position, written := 0, false
	for i, run := range runs {
		runStart, runEnd := position, position+len(texts[i])
		position = runEnd
		if runEnd <= start || runStart >= end {
			continue
		}
		head := texts[i][:max(0, min(len(texts[i]), start-runStart))]
		tail := ""
		if runEnd > end {
			tail = texts[i][max(0, end-runStart):]
		}
		if !written {
			ooxml.SetRunText(run, head+replacement+tail)
			clearPlaceholderMarks(run)
			written = true
		} else {
			ooxml.SetRunText(run, head+tail)
		}
	}
	return true
}

// ------------------------------------------------------------------- cover ---

// FillCover fills the cover block and drops the unused alternate-title lines.
//
// The master offers two title lines separated by the word "or", so an author can
// show a Framework and the Policy it sits under. Most documents have one title,
// and leaving a stray "or" on the cover looks like a defect, so both extra lines
// are removed unless alt_title is set.
func FillCover(body *etree.Element, meta *frontmatter.Meta) error {
	var titleParagraphs []*etree.Element
	for _, paragraph := range ooxml.Paragraphs(body) {
		if strings.Contains(ooxml.ParagraphText(paragraph), titlePlaceholder) {
			titleParagraphs = append(titleParagraphs, paragraph)
		}
	}
	if len(titleParagraphs) < 2 {
		return errorf("expected two %q cover lines, found %d", titlePlaceholder,
			len(titleParagraphs))
	}

	ReplaceInParagraph(titleParagraphs[0], titlePlaceholder, meta.CoverTitle)

	if meta.CoverAltTitle != "" {
		ReplaceInParagraph(titleParagraphs[1], titlePlaceholder, meta.CoverAltTitle)
	} else {
		ooxml.Remove(titleParagraphs[1])
		for _, paragraph := range ooxml.Paragraphs(body) {
			if strings.TrimSpace(ooxml.ParagraphText(paragraph)) == "or" {
				ooxml.Remove(paragraph)
				break
			}
		}
	}

	for _, paragraph := range ooxml.Paragraphs(body) {
		text := ooxml.ParagraphText(paragraph)
		switch {
		case strings.Contains(text, versionPlaceholder):
			ReplaceInParagraph(paragraph, versionPlaceholder, "Version: "+meta.Version)
		case strings.Contains(text, datePlaceholder):
			ReplaceInParagraph(paragraph, datePlaceholder, meta.Date)
		}
	}
	return nil
}

// ------------------------------------------------------------------ tables ---

// applyMarkFormat gives a run the paragraph mark's font and size where it has
// none of its own.
//
// An empty template cell still holds a bare <w:r/> carrying no w:rPr. Text
// written into it would render at the document default rather than the
// template's, which is visibly smaller. The intended formatting is on the
// paragraph mark (w:pPr/w:rPr), so that is what gets copied down.
func applyMarkFormat(run, paragraph *etree.Element) {
	rPr := run.SelectElement("w:rPr")
	if rPr == nil {
		rPr = etree.NewElement("w:rPr")
		run.InsertChildAt(0, rPr)
	}

	var mark *etree.Element
	if pPr := paragraph.SelectElement("w:pPr"); pPr != nil {
		mark = pPr.SelectElement("w:rPr")
	}

	for _, tag := range []string{"w:sz", "w:szCs"} {
		if rPr.SelectElement(tag) != nil {
			continue
		}
		value := ooxml.BodySz
		if mark != nil {
			if source := mark.SelectElement(tag); source != nil {
				value = source.SelectAttrValue("w:val", ooxml.BodySz)
			}
		}
		ooxml.Sub(rPr, tag, "val", value)
	}

	if rPr.SelectElement("w:rFonts") == nil {
		var source *etree.Element
		if mark != nil {
			source = mark.SelectElement("w:rFonts")
		}
		if source != nil {
			rPr.InsertChildAt(0, source.Copy())
		} else {
			ooxml.SetFonts(rPr, ooxml.BodyFont)
		}
	}
}

// setCellText writes text into a cell, keeping the first run's formatting.
//
// Reusing the existing run rather than making a new one inherits the template's
// font, size and alignment for free. Surplus runs and paragraphs are dropped so
// nothing of the sample data survives.
func setCellText(cell *etree.Element, text string) {
	paragraphs := ooxml.Paragraphs(cell)
	if len(paragraphs) == 0 {
		return
	}
	target := paragraphs[0]
	for _, extra := range paragraphs[1:] {
		ooxml.Remove(extra)
	}

	runs := ooxml.Runs(target)
	if len(runs) == 0 {
		target.CreateElement("w:r")
		runs = ooxml.Runs(target)
	}

	ooxml.SetRunText(runs[0], text)
	clearPlaceholderMarks(runs[0])
	applyMarkFormat(runs[0], target)
	for _, extra := range runs[1:] {
		ooxml.Remove(extra)
	}
}

// FillVersionControl fills the five-row Document Owner / approval-dates table.
func FillVersionControl(table *etree.Element, meta *frontmatter.Meta) {
	for _, row := range ooxml.Rows(table) {
		cells := ooxml.Cells(row)
		if len(cells) < 2 {
			continue
		}
		label := strings.TrimSpace(ooxml.CellText(cells[0]))
		read, ok := versionControlLabels[label]
		if !ok {
			continue
		}
		if value := read(meta); value != "" {
			setCellText(cells[1], value)
		}
	}
}

// FillRevisions rebuilds the revision-history rows from front matter.
//
// Row 0 is the header and row 1 is the sample row, which doubles as the
// formatting prototype for however many revisions the author declared.
func FillRevisions(table *etree.Element, revisions []frontmatter.Revision) {
	if len(revisions) == 0 {
		return
	}
	rows := ooxml.Rows(table)
	if len(rows) < 2 {
		return
	}
	prototype := rows[1].Copy()

	for _, row := range rows[1:] {
		ooxml.Remove(row)
	}

	for _, revision := range revisions {
		newRow := prototype.Copy()
		table.AddChild(newRow)
		cells := ooxml.Cells(newRow)
		for i, value := range revision.Values() {
			if i < len(cells) {
				setCellText(cells[i], value)
			}
		}
	}
}

// MarkClassification shades the chosen classification row the way the master
// does it.
func MarkClassification(table *etree.Element, classification string) {
	rows := ooxml.Rows(table)
	if len(rows) < 2 {
		return
	}
	for _, row := range rows[1:] {
		cells := ooxml.Cells(row)
		if len(cells) < 2 {
			continue
		}
		label := strings.TrimSpace(ooxml.CellText(cells[0]))
		tcPr := cells[1].SelectElement("w:tcPr")
		if tcPr == nil {
			continue
		}
		for _, shd := range tcPr.SelectElements("w:shd") {
			tcPr.RemoveChild(shd)
		}
		if label != classification {
			continue
		}
		fill, ok := classFills[classification]
		if !ok {
			fill = "fff2cc"
		}
		tcPr.AddChild(ooxml.El("w:shd", "val", "clear", "color", "auto", "fill", fill))
	}
}

// -------------------------------------------------------------------- body ---

// AddPageBreakBefore starts the paragraph whose text is text on a new page.
func AddPageBreakBefore(body *etree.Element, text string) bool {
	for _, paragraph := range ooxml.Paragraphs(body) {
		if strings.TrimSpace(ooxml.ParagraphText(paragraph)) != text {
			continue
		}
		ooxml.SetPPrChild(ooxml.GetOrAddPPr(paragraph), "w:pageBreakBefore", "val", "1")
		return true
	}
	return false
}

// RemoveEmptyParagraphsBefore drops blank filler paragraphs sitting immediately
// above text.
//
// The master pads its sections apart with empty paragraphs. Once the section
// starts on a page of its own those spacers have nothing to space, and if they
// happen to cross the page boundary they produce an entirely blank page ahead of
// the break.
func RemoveEmptyParagraphsBefore(body *etree.Element, text string) int {
	paragraphs := ooxml.Paragraphs(body)
	target := -1
	for i, paragraph := range paragraphs {
		if strings.TrimSpace(ooxml.ParagraphText(paragraph)) == text {
			target = i
			break
		}
	}
	if target < 0 {
		return 0
	}
	removed := 0
	for i := target - 1; i >= 0; i-- {
		paragraph := paragraphs[i]
		if strings.TrimSpace(ooxml.ParagraphText(paragraph)) != "" {
			break
		}
		if paragraph.SelectElement("w:tbl") != nil {
			break
		}
		ooxml.Remove(paragraph)
		removed++
	}
	return removed
}

var headingStyleRE = regexp.MustCompile(`^Heading([1-6])$`)

type headingStyle struct {
	level int
	style *etree.Element
}

// headingStyles yields each Heading1..Heading6 style definition.
func headingStyles(styles *etree.Element) []headingStyle {
	var out []headingStyle
	for _, style := range styles.SelectElements("w:style") {
		match := headingStyleRE.FindStringSubmatch(style.SelectAttrValue("w:styleId", ""))
		if match == nil {
			continue
		}
		level, _ := strconv.Atoi(match[1])
		out = append(out, headingStyle{level: level, style: style})
	}
	return out
}

// stylePPr returns the style's w:pPr, created in the right place if it has none.
// w:pPr must precede w:rPr inside w:style or Word rejects the style.
func stylePPr(style *etree.Element) *etree.Element {
	if pPr := style.SelectElement("w:pPr"); pPr != nil {
		return pPr
	}
	pPr := etree.NewElement("w:pPr")
	rPr := style.SelectElement("w:rPr")
	if rPr == nil {
		style.AddChild(pPr)
	} else {
		style.InsertChildAt(rPr.Index(), pPr)
	}
	return pPr
}

func styleRPr(style *etree.Element) *etree.Element {
	if rPr := style.SelectElement("w:rPr"); rPr != nil {
		return rPr
	}
	return style.CreateElement("w:rPr")
}

// setRPrFlag sets or clears a boolean run property such as w:b or w:i.
// Removing the element is not enough for a style that inherits the flag, so it
// is written as val="0" rather than dropped.
func setRPrFlag(rPr *etree.Element, tag string, on bool) {
	value := "0"
	if on {
		value = "1"
	}
	for _, name := range []string{tag, tag + "Cs"} {
		for _, existing := range rPr.SelectElements("w:" + name) {
			rPr.RemoveChild(existing)
		}
		ooxml.Sub(rPr, "w:"+name, "val", value)
	}
}

// NormaliseHeadingEmphasis makes weight fall as heading depth grows, for
// Heading3..Heading6.
//
// Two defects in the master, and the second is the one readers notice:
// Heading4..Heading6 inherit Normal's 11pt while body text is 12pt, so a
// fourth-level heading renders smaller than the prose underneath it; and
// Heading3 is regular while Heading4 and Heading6 are bold, so a child heading
// looks stronger than its parent.
func NormaliseHeadingEmphasis(styles *etree.Element) int {
	changed := 0
	for _, entry := range headingStyles(styles) {
		emphasis, ok := ooxml.HeadingEmphasis[entry.level]
		if !ok {
			continue // Heading1 and Heading2 keep the template's 16pt and 14pt
		}
		rPr := styleRPr(entry.style)
		for _, tag := range []string{"w:sz", "w:szCs"} {
			for _, existing := range rPr.SelectElements(tag) {
				rPr.RemoveChild(existing)
			}
			ooxml.Sub(rPr, tag, "val", ooxml.BodySz)
		}
		setRPrFlag(rPr, "b", emphasis.Bold)
		setRPrFlag(rPr, "i", emphasis.Italic)
		changed++
	}
	return changed
}

// NormaliseHeadingSpacing gives Heading1..Heading6 1.15 line spacing and room
// beneath them. The master's heading styles carry lineRule="auto" with no
// w:line, so they fall back to single spacing while the prose around them is
// 1.15, and they set after="0", so the heading sits hard against the first line
// of its own section.
func NormaliseHeadingSpacing(styles *etree.Element) int {
	changed := 0
	for _, entry := range headingStyles(styles) {
		ooxml.SetPPrChild(stylePPr(entry.style), "w:spacing",
			"before", ooxml.HeadingSpaceBefore, "after", ooxml.HeadingSpaceAfter,
			"line", ooxml.HeadingLineSpacing, "lineRule", "auto")
		changed++
	}
	return changed
}

// NormaliseHeadingKeeps stops a heading being stranded at the foot of a page, or
// split in two.
func NormaliseHeadingKeeps(styles *etree.Element) int {
	changed := 0
	for _, entry := range headingStyles(styles) {
		pPr := stylePPr(entry.style)
		ooxml.SetPPrChild(pPr, "w:keepNext", "val", "1")
		ooxml.SetPPrChild(pPr, "w:keepLines", "val", "1")
		changed++
	}
	return changed
}

// NormaliseHeadingIndents flattens the left indent out of the Heading1..Heading6
// style definitions.
//
// The master indents its heading styles, which is a leftover from the Google
// Docs numbered-heading list. A direct w:ind on the paragraph clears this for
// Heading1 and Heading2 but is silently ignored for Heading3, which then renders
// 90pt from the margin while its siblings sit flush.
func NormaliseHeadingIndents(styles *etree.Element) int {
	changed := 0
	for _, entry := range headingStyles(styles) {
		ooxml.SetPPrChild(stylePPr(entry.style), "w:ind",
			"left", "0", "right", "0", "hanging", "0", "firstLine", "0")
		changed++
	}
	return changed
}

// StripBody deletes everything from the first Heading1 to the end of the body.
//
// sectPr is deliberately left in place: it carries the page size, margins,
// titlePg flag and the header/footer wiring, so removing it would take the logo
// and the running head with it.
func StripBody(body *etree.Element) (int, error) {
	children := body.ChildElements()
	start := -1
	for index, element := range children {
		if element.FullTag() != "w:p" {
			continue
		}
		pPr := element.SelectElement("w:pPr")
		if pPr == nil {
			continue
		}
		style := pPr.SelectElement("w:pStyle")
		if style != nil && style.SelectAttrValue("w:val", "") == "Heading1" {
			start = index
			break
		}
	}
	if start < 0 {
		return 0, errorf("no Heading1 found in the template, so the body " +
			"boundary cannot be located")
	}

	removed := 0
	for _, element := range children[start:] {
		if element.FullTag() == "w:sectPr" {
			continue
		}
		body.RemoveChild(element)
		removed++
	}
	return removed, nil
}

// ----------------------------------------------------------------- headers ---

// FillRunningHead patches the running head, which lives in the header parts, not
// the body. Every header part is visited rather than the three a section names,
// which reaches the same runs by a shorter route.
func FillRunningHead(pkg *docx.Package, runningHead string) error {
	for _, name := range pkg.Names() {
		if !strings.HasPrefix(name, "word/header") || !strings.HasSuffix(name, ".xml") {
			continue
		}
		data, _ := pkg.Get(name)
		document := etree.NewDocument()
		if err := document.ReadFromBytes(data); err != nil {
			return fmt.Errorf("parse %s: %w", name, err)
		}
		changed := false
		for _, t := range document.FindElements("//w:t") {
			if strings.Contains(t.Text(), runningHeadPlaceholder) {
				t.SetText(strings.ReplaceAll(t.Text(), runningHeadPlaceholder, runningHead))
				changed = true
			}
		}
		if !changed {
			continue
		}
		out, err := document.WriteToBytes()
		if err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
		pkg.Set(name, out)
	}
	return nil
}

// ---------------------------------------------------------------- comments ---

var commentParts = []string{"word/comments.xml", "word/commentsExtended.xml"}

// Attribute order is not fixed, and no writer agrees on it. The bundled master
// is a Google Docs export and writes ContentType before PartName, so a pattern
// anchored on PartName coming first matches nothing and leaves an Override
// pointing at a part that no longer exists.
var (
	contentTypeOverrideRE = regexp.MustCompile(`<Override[^>]*PartName="/word/comments(Extended)?\.xml"[^>]*/>`)
	commentRelationshipRE = regexp.MustCompile(`<Relationship[^>]*Target="comments(Extended)?\.xml"[^>]*/>`)
)

// stripCommentAnchors removes the template's tracked comments from the body.
//
// The master carries 46KB of review comments. Left in place they travel into
// every generated document and, once uploaded, reappear as live Google Docs
// comments on a document that is supposed to be freshly issued.
//
// Done structurally rather than by regex: a run carrying a comment reference has
// to be found by looking inside it, and Go's regexp engine has no lookahead to
// express "a run that is not yet closed".
func stripCommentAnchors(root *etree.Element) {
	for _, tag := range []string{"//w:commentRangeStart", "//w:commentRangeEnd"} {
		for _, element := range root.FindElements(tag) {
			ooxml.Remove(element)
		}
	}
	for _, reference := range root.FindElements("//w:commentReference") {
		run := reference.Parent()
		for run != nil && run.FullTag() != "w:r" {
			run = run.Parent()
		}
		if run != nil {
			ooxml.Remove(run)
		}
	}
}

// stripCommentParts removes the comment parts, their relationships and their
// content-type overrides. Four things must stay consistent, which is why they
// are done together.
func stripCommentParts(pkg *docx.Package) {
	for _, part := range commentParts {
		pkg.Remove(part)
	}
	if data, ok := pkg.Get("[Content_Types].xml"); ok {
		pkg.Set("[Content_Types].xml", contentTypeOverrideRE.ReplaceAll(data, nil))
	}
	if data, ok := pkg.Get("word/_rels/document.xml.rels"); ok {
		pkg.Set("word/_rels/document.xml.rels", commentRelationshipRE.ReplaceAll(data, nil))
	}
}

// --------------------------------------------------------------- entrypoint --

// Shell is the opened package plus the parsed parts the body renderer needs to
// keep working on. Holding them parsed is what avoids a serialise-and-reparse
// round trip between every stage.
type Shell struct {
	Package   *docx.Package
	Document  *etree.Document
	Body      *etree.Element
	Numbering *etree.Document
	Removed   int
}

// Build copies the master, fills its front matter, and clears the sample body.
func Build(templatePath string, meta *frontmatter.Meta) (*Shell, error) {
	pkg, err := docx.Open(templatePath)
	if err != nil {
		return nil, err
	}

	documentXML, err := pkg.MustGet("word/document.xml")
	if err != nil {
		return nil, err
	}
	document := etree.NewDocument()
	if err := document.ReadFromBytes(documentXML); err != nil {
		return nil, fmt.Errorf("parse word/document.xml: %w", err)
	}
	root := document.SelectElement("w:document")
	if root == nil {
		return nil, errorf("word/document.xml has no w:document root")
	}
	body := root.SelectElement("w:body")
	if body == nil {
		return nil, errorf("word/document.xml has no w:body")
	}

	stylesXML, err := pkg.MustGet("word/styles.xml")
	if err != nil {
		return nil, err
	}
	stylesDoc := etree.NewDocument()
	if err := stylesDoc.ReadFromBytes(stylesXML); err != nil {
		return nil, fmt.Errorf("parse word/styles.xml: %w", err)
	}
	styles := stylesDoc.SelectElement("w:styles")
	if styles == nil {
		return nil, errorf("word/styles.xml has no w:styles root")
	}

	tables := ooxml.Tables(body)
	if len(tables) < 3 {
		return nil, errorf("expected at least 3 front-matter tables in the template, found %d",
			len(tables))
	}

	if err := FillCover(body, meta); err != nil {
		return nil, err
	}
	FillVersionControl(tables[0], meta)
	FillRevisions(tables[1], meta.Revisions)
	MarkClassification(tables[2], meta.Classification)

	NormaliseHeadingIndents(styles)
	NormaliseHeadingEmphasis(styles)
	NormaliseHeadingSpacing(styles)
	NormaliseHeadingKeeps(styles)

	// Give the cover, the document-control block and the contents list a page
	// each. The master runs them together, which leaves the Version Control
	// table split across the cover and page 2.
	for _, label := range []string{"Version Control", "Contents"} {
		RemoveEmptyParagraphsBefore(body, label)
		if !AddPageBreakBefore(body, label) {
			return nil, errorf("could not find the %q paragraph to break the page before", label)
		}
	}

	removed, err := StripBody(body)
	if err != nil {
		return nil, err
	}

	stripCommentAnchors(root)
	stripCommentParts(pkg)

	if err := FillRunningHead(pkg, meta.RunningHead); err != nil {
		return nil, err
	}

	styleBytes, err := stylesDoc.WriteToBytes()
	if err != nil {
		return nil, err
	}
	pkg.Set("word/styles.xml", styleBytes)

	numberingXML, err := pkg.MustGet("word/numbering.xml")
	if err != nil {
		return nil, err
	}
	numberingDoc := etree.NewDocument()
	if err := numberingDoc.ReadFromBytes(numberingXML); err != nil {
		return nil, fmt.Errorf("parse word/numbering.xml: %w", err)
	}

	return &Shell{
		Package:   pkg,
		Document:  document,
		Body:      body,
		Numbering: numberingDoc,
		Removed:   removed,
	}, nil
}
