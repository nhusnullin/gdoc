package drift

import (
	"archive/zip"
	"bytes"
	"math"
	"strings"
	"testing"
)

// TestABadDocxIsAnErrorNamingWhatIsWrong. The gate opens two files, and a file
// that is not a document has to say so rather than read as a document with
// nothing in it: every row would then be MISSING and the report would blame the
// house style for a file that never opened.
func TestABadDocxIsAnErrorNamingWhatIsWrong(t *testing.T) {
	if _, err := OpenDocx([]byte("this is not a zip")); err == nil ||
		!strings.Contains(err.Error(), "not a docx") {
		t.Errorf("bytes that are not a zip came back with %v", err)
	}
	if _, err := OpenDocxFile("testdata/there-is-no-such-file.docx"); err == nil ||
		!strings.Contains(err.Error(), "there-is-no-such-file.docx") {
		t.Errorf("a missing file came back with %v, and the error names the path", err)
	}

	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	w, err := z.Create("word/document.xml")
	if err != nil {
		t.Fatalf("the fixture zip could not be written: %v", err)
	}
	if _, err := w.Write([]byte(`<w:document xmlns:w="` + wordNS + `"><w:body/></w:document>`)); err != nil {
		t.Fatalf("the fixture zip could not be written: %v", err)
	}
	if err := z.Close(); err != nil {
		t.Fatalf("the fixture zip could not be closed: %v", err)
	}
	if _, err := OpenDocx(buf.Bytes()); err == nil || !strings.Contains(err.Error(), "word/styles.xml") {
		t.Errorf("a package with no styles came back with %v, and the error names the missing part", err)
	}
}

// TestTheLastDefinitionOfARepeatedStyleWins states the rule the master forced.
//
// Google's docx export writes Heading1 six times, and the first of the six
// carries #0041D3 with no bold. Word applies the last, so that first block is a
// statement nobody has ever seen on the page. Reading it would have this
// package report a colour that is in no document.
func TestTheLastDefinitionOfARepeatedStyleWins(t *testing.T) {
	got := openMaster(t).Style("Heading1")
	if got.Colour == "#0041D3" {
		t.Fatal("Heading 1 read the master's first definition, and Word applies the last")
	}
	if got.Colour != "#06436E" {
		t.Errorf("Heading 1 is declared %v in the master, and its last definition is #06436E", got.Colour)
	}
	if !got.Bold {
		t.Error("Heading 1's last definition in the master is bold")
	}
}

// TestAnAbsentColourIsBlackRatherThanNothing. OOXML's "auto" and no w:color at
// all both mean the colour a reader sees is black. Reporting nil would put a
// MISSING against a file that spells the same black out, which is a difference
// nobody can see and a row nobody can act on.
func TestAnAbsentColourIsBlackRatherThanNothing(t *testing.T) {
	if got := hexColour(""); got != nil {
		t.Errorf("an attribute that is not there read %v, and this helper answers about a value it was given", got)
	}
	if got := hexColour("auto"); got != "#000000" {
		t.Errorf("auto read %v, and a reader sees black", got)
	}
	if got := hexColour("22265f"); got != "#22265F" {
		t.Errorf("a lower case colour read %v, and the two halves compare it in upper case", got)
	}
	if got := openMaster(t).Style("Title").Colour; got != "#000000" {
		t.Errorf("the master's Title states no colour and read %v, and a reader sees black", got)
	}
}

// TestANumberIsReadWhicheverWayTheFileWroteIt. Google's export writes a row
// height as 487.96875 and a Word file writes 488. A reader that took only whole
// numbers would report the first as no height at all.
func TestANumberIsReadWhicheverWayTheFileWroteIt(t *testing.T) {
	if got := twipsPt("487.96875"); got != 24.398 {
		t.Errorf("487.96875 twips read as %v points, and it is 24.398", got)
	}
	if got := twipsPt("488"); got != 24.4 {
		t.Errorf("488 twips read as %v points, and it is 24.4", got)
	}
	if got := twipsPt(""); got != nil {
		t.Errorf("an attribute that is not there read %v, and the document says nothing", got)
	}
	if got := twipsPt("wide"); got != nil {
		t.Errorf("a value that is not a number read %v", got)
	}
	if got := emuPt("1857375"); got != 146.25 {
		t.Errorf("1857375 EMU read as %v points, and it is 146.25", got)
	}
}

// TestWordsJustificationIsReadUnderTheDocsNames, so one item reports the same
// word whichever half read it.
func TestWordsJustificationIsReadUnderTheDocsNames(t *testing.T) {
	for word, want := range map[string]string{
		"left": "START", "start": "START", "center": "CENTER",
		"both": "JUSTIFIED", "right": "END", "end": "END",
	} {
		if got := alignment(word); got != want {
			t.Errorf("Word's %q read as %v, and Docs calls it %v", word, got, want)
		}
	}
}

// TestWordsBooleanIsTrueWhenTheElementIsThere. `<w:b/>` with no value is bold,
// which is what the schema says and what every reader of one expects.
func TestWordsBooleanIsTrueWhenTheElementIsThere(t *testing.T) {
	for value, want := range map[string]bool{
		"": true, "1": true, "true": true, "on": true,
		"0": false, "false": false, "off": false,
	} {
		if got := onOff(value); got != want {
			t.Errorf("w:val=%q read as %v, and it is %v", value, got, want)
		}
	}
}

// TestARoundedNumberIsTheSameOnEveryArchitecture. round3 cast through int64,
// and Go leaves a float to int64 conversion out of range implementation
// dependent: NaN read as 0 on darwin/arm64 and as -9.2e15 on darwin/amd64.
// `make dist` ships both, so one document measured two ways is one gate giving
// two answers. NaN has to stay NaN, so the row goes DIFFERENT rather than
// carrying a number nobody wrote.
func TestARoundedNumberIsTheSameOnEveryArchitecture(t *testing.T) {
	if got := round3(math.NaN()); !math.IsNaN(got) {
		t.Errorf("NaN rounded to %v, and a value that is not a number stays one", got)
	}
	if got := round3(math.Inf(1)); !math.IsInf(got, 1) {
		t.Errorf("+Inf rounded to %v", got)
	}
	if got := round3(24.3984375); got != 24.398 {
		t.Errorf("24.3984375 rounded to %v, and three decimals is 24.398", got)
	}
	if got := round3(-24.3984375); got != -24.398 {
		t.Errorf("-24.3984375 rounded to %v, and three decimals is -24.398", got)
	}
}

// TestAStyleThatIsNotInTheFileCarriesNothing, which is what Style's own doc
// comment promises. docDefaults were folded in before the chain was walked, so
// an absent style came back 11pt Calibri at 115%: drop a named style from
// house.yaml and six of its eleven rows read the defaults on both sides and
// compare IDENTICAL. The gate would pass on a style that no longer exists.
func TestAStyleThatIsNotInTheFileCarriesNothing(t *testing.T) {
	got := openMaster(t).Style("ThisStyleIsNotInTheMaster")
	for _, f := range []struct {
		name string
		v    any
	}{
		{"fontSize", got.FontSize}, {"font", got.Font}, {"colour", got.Colour},
		{"lineSpacing", got.LineSpacing}, {"spaceAbove", got.SpaceAbove},
		{"spaceBelow", got.SpaceBelow}, {"indentStart", got.IndentStart},
		{"alignment", got.Alignment},
	} {
		if f.v != nil {
			t.Errorf("a style the master does not carry read %s as %v, and the file says nothing about it", f.name, f.v)
		}
	}
	if got.Bold || got.Italic || got.KeepWithNext {
		t.Errorf("a style the master does not carry read bold %v, italic %v, keepWithNext %v", got.Bold, got.Italic, got.KeepWithNext)
	}
	// The defaults are still folded in for a style that is there, which is what
	// Word does and what every other row rests on.
	if openMaster(t).Style("Heading1").FontSize == nil {
		t.Error("Heading 1 is in the master and read no font size")
	}
}
