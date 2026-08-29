package shell

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/beevik/etree"

	"spike/gdocgo/internal/frontmatter"
	"spike/gdocgo/internal/ooxml"
)

// masterPath is the bundled template, reached where it really lives rather than
// through a copy. The surgery finds the cover by placeholder text and the tables
// by their first-column labels, so testing it against anything else would test a
// different contract, and a copy would drift out of that contract in silence.
func masterPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..",
		"gdoc", "templates", "altery-group-policy-v1.0", "template.docx")
}

func meta(t *testing.T, text string) *frontmatter.Meta {
	t.Helper()
	parsed, _, err := frontmatter.Parse(text, "", "note.md")
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func buildShell(t *testing.T, front string) *Shell {
	t.Helper()
	built, err := Build(masterPath(t), meta(t, front))
	if err != nil {
		t.Fatal(err)
	}
	return built
}

func bodyText(body *etree.Element) string {
	var b strings.Builder
	for _, t := range body.FindElements("//w:t") {
		b.WriteString(t.Text())
		b.WriteString("\n")
	}
	return b.String()
}

const fullFront = `---
title: Third Party and Outsourcing
doc_type: Policy
version: 3.1
date: 2026-08-29
owner: Chief Risk Officer
last_approval: 2026-08-14
review_frequency: Annually
board_ratification: 2026-08-20
distribution: All staff
classification: confidential
revisions:
  - version: 3.1
    date: 2026-08-29
    author: N K
    approved_by: Board
    approval_date: 2026-08-20
    section: 4
    change: Reissued
---
body
`

func TestTheCoverCarriesTheTitleAndTheVersion(t *testing.T) {
	built := buildShell(t, fullFront)

	text := bodyText(built.Body)
	if !strings.Contains(text, "Third Party and Outsourcing Policy") {
		t.Error("the cover title was not written")
	}
	if strings.Contains(text, "(Name of) Framework/Policy") {
		t.Error("a title placeholder survived")
	}
	if !strings.Contains(text, "Version: 3.1") {
		t.Error("the version was not written")
	}
	if strings.Contains(text, "May 2025") {
		t.Error("the date placeholder survived")
	}
}

func TestTheAlternateTitleLinesGoWhenThereIsOnlyOneTitle(t *testing.T) {
	// Leaving a stray "or" on the cover looks like a defect.
	built := buildShell(t, "---\ntitle: Access Control\n---\nbody\n")

	for _, paragraph := range ooxml.Paragraphs(built.Body) {
		if strings.TrimSpace(ooxml.ParagraphText(paragraph)) == "or" {
			t.Fatal("the cover still carries the alternate-title 'or'")
		}
	}
}

func TestAnAlternateTitleIsKeptWhenThereIsOne(t *testing.T) {
	built := buildShell(t,
		"---\ntitle: Third Party\nalt_title: Supplier Management\ndoc_type: Policy\n---\nbody\n")

	text := bodyText(built.Body)
	if !strings.Contains(text, "Supplier Management Policy") {
		t.Error("the alternate title was dropped")
	}
}

func TestTheVersionControlTableIsFilledByItsLabels(t *testing.T) {
	built := buildShell(t, fullFront)

	text := bodyText(built.Body)
	for _, want := range []string{"Chief Risk Officer", "All staff"} {
		if !strings.Contains(text, want) {
			t.Errorf("%q is not in the control tables", want)
		}
	}
}

func TestTheChosenClassificationIsTheOnlyShadedRow(t *testing.T) {
	built := buildShell(t, fullFront)
	tables := ooxml.Tables(built.Body)

	shaded := map[string]string{}
	for _, row := range ooxml.Rows(tables[2])[1:] {
		cells := ooxml.Cells(row)
		if len(cells) < 2 {
			continue
		}
		tcPr := cells[1].SelectElement("w:tcPr")
		if tcPr == nil {
			continue
		}
		if shd := tcPr.SelectElement("w:shd"); shd != nil {
			shaded[strings.TrimSpace(ooxml.CellText(cells[0]))] =
				shd.SelectAttrValue("w:fill", "")
		}
	}

	if len(shaded) != 1 {
		t.Fatalf("shaded rows = %#v, want only the chosen class", shaded)
	}
	if shaded["Confidential (C)"] != "f4cccc" {
		t.Errorf("shading = %#v, want Confidential in its own colour", shaded)
	}
}

func TestTheRevisionRowsAreRebuiltFromTheFrontMatter(t *testing.T) {
	built := buildShell(t, fullFront)
	tables := ooxml.Tables(built.Body)

	rows := ooxml.Rows(tables[1])
	if len(rows) != 2 {
		t.Fatalf("revision rows = %d, want a header and one revision", len(rows))
	}
	if !strings.Contains(ooxml.CellText(ooxml.Cells(rows[1])[6]), "Reissued") {
		t.Errorf("the change column reads %q", ooxml.CellText(ooxml.Cells(rows[1])[6]))
	}
}

func TestTheSampleBodyIsGoneButThePageSetupStays(t *testing.T) {
	// sectPr carries the page size, margins, titlePg flag and the header wiring,
	// so removing it would take the logo and the running head with it.
	built := buildShell(t, fullFront)

	if built.Body.SelectElement("w:sectPr") == nil {
		t.Fatal("the sectPr went with the body, so the page furniture is lost")
	}
	if strings.Contains(bodyText(built.Body), "details how Altery manages") {
		t.Error("the template's sample body survived")
	}
	if built.Removed == 0 {
		t.Error("nothing was removed, so the body boundary was never found")
	}
}

func TestTheControlBlockAndTheContentsEachStartAPage(t *testing.T) {
	built := buildShell(t, fullFront)

	found := map[string]bool{}
	for _, paragraph := range ooxml.Paragraphs(built.Body) {
		text := strings.TrimSpace(ooxml.ParagraphText(paragraph))
		if text != "Version Control" && text != "Contents" {
			continue
		}
		pPr := paragraph.SelectElement("w:pPr")
		if pPr != nil && pPr.SelectElement("w:pageBreakBefore") != nil {
			found[text] = true
		}
	}

	for _, label := range []string{"Version Control", "Contents"} {
		if !found[label] {
			t.Errorf("%q does not start a page of its own", label)
		}
	}
}

func TestTheTemplatesReviewCommentsDoNotTravel(t *testing.T) {
	// Left in place they reappear as live Google Docs comments on a document
	// that is supposed to be freshly issued.
	built := buildShell(t, fullFront)

	for _, name := range built.Package.Names() {
		if strings.Contains(name, "comments") {
			t.Errorf("the package still carries %s", name)
		}
	}
	if len(built.Body.FindElements("//w:commentRangeStart")) > 0 {
		t.Error("comment anchors survived in the body")
	}
	if len(built.Body.FindElements("//w:commentReference")) > 0 {
		t.Error("comment reference runs survived in the body")
	}
	types, _ := built.Package.Get("[Content_Types].xml")
	if strings.Contains(string(types), "comments.xml") {
		t.Error("the content-type override for the comment part survived")
	}
	rels, _ := built.Package.Get("word/_rels/document.xml.rels")
	if strings.Contains(string(rels), "comments.xml") {
		t.Error("the relationship to the comment part survived")
	}
}

func TestTheRunningHeadIsPatchedInTheHeaderParts(t *testing.T) {
	built := buildShell(t, fullFront)

	found := false
	for _, name := range built.Package.Names() {
		if !strings.HasPrefix(name, "word/header") {
			continue
		}
		data, _ := built.Package.Get(name)
		if strings.Contains(string(data), "Altery - Third Party and Outsourcing Policy") {
			found = true
		}
		if strings.Contains(string(data), "Altery - xxx Policy") {
			t.Errorf("%s still carries the placeholder running head", name)
		}
	}
	if !found {
		t.Error("no header part carries the running head")
	}
}

func TestTheHeadingStylesAreFlattenedAndEvened(t *testing.T) {
	// A direct w:ind on the paragraph clears this for Heading1 and Heading2 but
	// is silently ignored for Heading3, which then renders 90pt from the margin.
	built := buildShell(t, fullFront)
	styles, _ := built.Package.Get("word/styles.xml")
	document := etree.NewDocument()
	if err := document.ReadFromBytes(styles); err != nil {
		t.Fatal(err)
	}

	for _, entry := range headingStyles(document.SelectElement("w:styles")) {
		pPr := entry.style.SelectElement("w:pPr")
		if pPr == nil {
			t.Fatalf("Heading%d has no pPr", entry.level)
		}
		ind := pPr.SelectElement("w:ind")
		if ind == nil || ind.SelectAttrValue("w:left", "") != "0" ||
			ind.SelectAttrValue("w:hanging", "") != "0" {
			t.Errorf("Heading%d indent = %v, want it flat", entry.level, ind)
		}
		spacing := pPr.SelectElement("w:spacing")
		if spacing == nil || spacing.SelectAttrValue("w:after", "") == "0" {
			t.Errorf("Heading%d has no room beneath it", entry.level)
		}
		if pPr.SelectElement("w:keepNext") == nil || pPr.SelectElement("w:keepLines") == nil {
			t.Errorf("Heading%d can be stranded at the foot of a page", entry.level)
		}
	}
}

func TestATemplateWithNoHeading1IsRefusedByName(t *testing.T) {
	body := etree.NewElement("w:body")
	body.CreateElement("w:p")

	if _, err := StripBody(body); err == nil {
		t.Fatal("a template with no Heading1 must be refused, not silently emptied")
	}
}

func TestAPlaceholderSplitAcrossRunsIsStillReplaced(t *testing.T) {
	// Google Docs starts a new run wherever formatting changes, so
	// "(Name of) Framework/Policy" is really two runs.
	paragraph := etree.NewElement("w:p")
	ooxml.TextRun(paragraph, "(Name of) ", ooxml.RunOpts{})
	ooxml.TextRun(paragraph, "Framework/Policy", ooxml.RunOpts{})

	if !ReplaceInParagraph(paragraph, "(Name of) Framework/Policy", "Access Control") {
		t.Fatal("the split placeholder was not found")
	}

	if got := ooxml.ParagraphText(paragraph); got != "Access Control" {
		t.Errorf("paragraph = %q, want %q", got, "Access Control")
	}
}
