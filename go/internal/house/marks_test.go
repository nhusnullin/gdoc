package house

import (
	"bytes"
	"sort"
	"strings"
	"testing"
)

// Every emoji and every hex in this file is written out. A test that reads
// cfg.Marks for the value it checks is a mirror: change the palette and the
// assertion follows it. See CLAUDE.md, "A house-style test states its value as
// a literal".

// TestTheSevenMarksAreStatedAsLiterals is the palette of M18's part B, the one
// source of what an author may write and what colour it reaches the document
// as. Yellow carries two openers because a plain highlight is yellow and
// Obsidian's own 🟡 says the same thing. The pencil carries three since
// milestone 20: Pandoc's span, which is going, and the two comments that
// replace it, HTML's and Obsidian's.
func TestTheSevenMarksAreStatedAsLiterals(t *testing.T) {
	cfg := load(t)
	cases := []struct {
		name   string
		syntax []string
		kind   string
		color  string
		debt   bool
	}{
		{"yellow", []string{"==", "🟡"}, "highlight", "#FFD500", true},
		{"green", []string{"🟢"}, "highlight", "#3DD67A", true},
		{"purple", []string{"🟣"}, "highlight", "#BA92FF", true},
		{"blue", []string{"🔵"}, "highlight", "#5DB3FF", false},
		{"red", []string{"🔴"}, "highlight", "#FF6B6B", false},
		{"orange", []string{"🟠"}, "highlight", "#FFB85C", false},
		{"pencil", []string{"<!--", "%%", ".pencil"}, "text", "#9A9A9A", false},
	}
	for _, c := range cases {
		mark, ok := cfg.Marks[c.name]
		if !ok {
			t.Errorf("marks.%s is missing", c.name)
			continue
		}
		if strings.Join(mark.Syntax, " ") != strings.Join(c.syntax, " ") {
			t.Errorf("marks.%s.syntax = %q, want %q", c.name, mark.Syntax, c.syntax)
		}
		if mark.Kind != c.kind {
			t.Errorf("marks.%s.kind = %q, want %q", c.name, mark.Kind, c.kind)
		}
		if mark.Color != c.color {
			t.Errorf("marks.%s.color = %q, want %q", c.name, mark.Color, c.color)
		}
		if mark.Debt != c.debt {
			t.Errorf("marks.%s.debt = %v, want %v", c.name, mark.Debt, c.debt)
		}
	}
}

// TestTheMarksAreTheSevenAndNoOther. The seven are what body parses, what
// render writes a character style for and what export reads back, each by
// name, so an eighth nobody reads and a seventh that is gone are both a
// document built silently wrong.
func TestTheMarksAreTheSevenAndNoOther(t *testing.T) {
	cfg := load(t)
	got := make([]string, 0, len(cfg.Marks))
	for name := range cfg.Marks {
		got = append(got, name)
	}
	sort.Strings(got)
	want := "blue green orange pencil purple red yellow"
	if strings.Join(got, " ") != want {
		t.Errorf("marks = %q, want %q", strings.Join(got, " "), want)
	}

	eighth := bytes.Replace(embedded, []byte("  pencil:\n"),
		[]byte("  teal:\n    syntax:\n    - 🟤\n    kind: highlight\n    color: '#549F99'\n    debt: false\n  pencil:\n"), 1)
	if bytes.Equal(eighth, embedded) {
		t.Fatal("marks.pencil was not found to insert an eighth mark before")
	}
	if _, err := LoadFile(write(t, eighth)); err == nil {
		t.Error("an eighth mark was accepted")
	} else if !strings.Contains(err.Error(), "teal") {
		t.Errorf("error %q does not name teal", err)
	}

	src := bytes.Replace(embedded, []byte("  pencil:\n"), []byte("  graphite:\n"), 1)
	if bytes.Equal(src, embedded) {
		t.Fatal("marks.pencil was not found to replace")
	}
	if _, err := LoadFile(write(t, src)); err == nil {
		t.Error("a config without the pencil was accepted")
	} else if !strings.Contains(err.Error(), "pencil") {
		t.Errorf("error %q does not name pencil", err)
	}
}

// TestAMarkWithAnUnknownKindIsRefused. render asks a mark whether it is a
// highlight, which is run shading, or a text colour, which is w:color. A third
// word is a mark it would write neither way.
func TestAMarkWithAnUnknownKindIsRefused(t *testing.T) {
	src := bytes.Replace(embedded,
		[]byte("    kind: text\n"),
		[]byte("    kind: underline\n"), 1)
	if bytes.Equal(src, embedded) {
		t.Fatal("the pencil's kind was not found to replace")
	}
	_, err := LoadFile(write(t, src))
	if err == nil {
		t.Fatal("a mark whose kind is underline was accepted")
	}
	if !strings.Contains(err.Error(), "underline") {
		t.Errorf("error %q does not name underline", err)
	}
	if !strings.Contains(err.Error(), "pencil") {
		t.Errorf("error %q does not name the mark it refused", err)
	}
}

// TestAMarkWhoseColourIsNotHexIsRefused. A mark's colour is a hex value, the
// way every other colour in this file is: render writes it into w:shd w:fill
// and w:color, which take six hex digits and nothing else, and export matches
// the Docs answer against it hex for hex.
func TestAMarkWhoseColourIsNotHexIsRefused(t *testing.T) {
	cases := []struct {
		name string
		put  string
	}{
		{"a name rather than a colour", "lightGray"},
		{"a hex value with no hash", "FFD500"},
		{"three digits", "#FD5"},
		{"digits that are not hex", "#GGHHII"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := bytes.Replace(embedded,
				[]byte("    color: '#FFD500'\n"),
				[]byte("    color: '"+c.put+"'\n"), 1)
			if bytes.Equal(src, embedded) {
				t.Fatal("the yellow mark's colour was not found to replace")
			}
			_, err := LoadFile(write(t, src))
			if err == nil {
				t.Fatalf("a mark whose colour is %q was accepted", c.put)
			}
			if !strings.Contains(err.Error(), c.put) {
				t.Errorf("error %q does not name %q", err, c.put)
			}
			if !strings.Contains(err.Error(), "yellow") {
				t.Errorf("error %q does not name the mark it refused", err)
			}
		})
	}
}

// TestTwoMarksCannotShareASyntax. body resolves what the author wrote to one
// mark. Two marks claiming 🟢 is a highlight whose colour depends on which of
// them a map walk reached first.
func TestTwoMarksCannotShareASyntax(t *testing.T) {
	src := bytes.Replace(embedded,
		[]byte("    - 🔵\n"),
		[]byte("    - 🟢\n"), 1)
	if bytes.Equal(src, embedded) {
		t.Fatal("the blue mark's emoji was not found to replace")
	}
	_, err := LoadFile(write(t, src))
	if err == nil {
		t.Fatal("two marks sharing an emoji were accepted")
	}
	if !strings.Contains(err.Error(), "green") || !strings.Contains(err.Error(), "blue") {
		t.Errorf("error %q does not name both marks", err)
	}
}

// TestAMarkWithNoSyntaxIsRefused. A mark no author can write is a colour
// export would read back out of a document and no route could ever put there.
func TestAMarkWithNoSyntaxIsRefused(t *testing.T) {
	src := bytes.Replace(embedded,
		[]byte("    syntax:\n    - 🟠\n"),
		[]byte("    syntax: []\n"), 1)
	if bytes.Equal(src, embedded) {
		t.Fatal("the orange mark's syntax was not found to replace")
	}
	_, err := LoadFile(write(t, src))
	if err == nil {
		t.Fatal("a mark with no syntax was accepted")
	}
	if !strings.Contains(err.Error(), "orange") {
		t.Errorf("error %q does not name orange", err)
	}
}

// TestAPaletteWhereNoMarkClaimsTheBareHighlightIsRefused. "==" with no colour
// emoji on it is what an author writes when they mean nothing by the colour,
// and internal/body resolves it to a mark by name: a palette claiming it
// nowhere resolves to no name, and every ==highlight== in the note then reached
// the document with no colour on it and nothing said so.
func TestAPaletteWhereNoMarkClaimsTheBareHighlightIsRefused(t *testing.T) {
	src := bytes.Replace(embedded,
		[]byte("    syntax:\n    - '=='\n    - \U0001F7E1\n"),
		[]byte("    syntax:\n    - \U0001F7E1\n"), 1)
	if bytes.Equal(src, embedded) {
		t.Fatal("the yellow mark's bare delimiter was not found to remove")
	}
	_, err := LoadFile(write(t, src))
	if err == nil {
		t.Fatal("a palette where nothing claims == was accepted")
	}
	if !strings.Contains(err.Error(), "==") {
		t.Errorf("error %q does not name the delimiter nobody claims", err)
	}
}

// TestTheBareHighlightBelongsToAHighlight. The delimiter is read as shading,
// so the mark claiming it has to be one: given to the pencil, every
// ==highlight== would come out as grey text.
func TestTheBareHighlightBelongsToAHighlight(t *testing.T) {
	src := bytes.Replace(embedded,
		[]byte("    syntax:\n    - '<!--'\n"),
		[]byte("    syntax:\n    - '<!--'\n    - '=='\n"), 1)
	if bytes.Equal(src, embedded) {
		t.Fatal("the pencil mark's syntax was not found to extend")
	}
	src = bytes.Replace(src,
		[]byte("    syntax:\n    - '=='\n    - \U0001F7E1\n"),
		[]byte("    syntax:\n    - \U0001F7E1\n"), 1)
	_, err := LoadFile(write(t, src))
	if err == nil {
		t.Fatal("a palette giving == to the pencil was accepted")
	}
	if !strings.Contains(err.Error(), "pencil") {
		t.Errorf("error %q does not name the pencil", err)
	}
}
