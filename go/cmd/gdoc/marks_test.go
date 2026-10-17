// The marks and the footnotes where a caller meets them: the two envelopes
// that carry what the walker counted, the help topic that says what the seven
// marks are, and the two notes a build refuses before it writes a file.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// debtNote is a note carrying one yellow mark, one purple, no green, and one
// footnote. The title is the one the publish read-back fixture carries, so the
// two commands can be run over the same note without a title warning.
const debtNote = `---
title: Supplier Register Policy
---

# Purpose

A ==figure== to replace, and a ==🟣question== for one reviewer.[^a]

[^a]: The note under the page.
`

// debtCounts is what both envelopes must carry for that note, written out as
// literals. The three debts and no fourth: blue, red and orange are marks the
// binary writes and does not count, and a key for one of them here would be
// the binary judging that an aside is worth reporting.
var debtCounts = map[string]float64{"yellow": 1, "green": 0, "purple": 1}

// The envelope of both commands that turn a note into a document carries what
// the walker counted, and the three debts are in it with a zero where the note
// has none: a note somebody has been through is not the same fact as a note
// nobody has read. Whether a document carrying a debt may be published is the
// skill's to say, which is why the count is a number and not a refusal.
func TestBuildAndPublishCarryTheDebtCounts(t *testing.T) {
	theHouseLogoCached(t)
	t.Run("build", func(t *testing.T) {
		noSession(t)
		md := writeNote(t, "register.md", debtNote)
		out := filepath.Join(filepath.Dir(md), "register.docx")

		got, code := runJSON(t, "build", "--md", md, "--out", out)
		if code != 0 || got["ok"] != true {
			t.Fatalf("build: %v (exit %d)", got, code)
		}
		assertDebts(t, got)
	})

	t.Run("publish", func(t *testing.T) {
		stubWire(t, &fakeWire{answers: publishAnswers(t)})
		md := writeNote(t, "register.md", debtNote)

		got, code := runJSON(t, "publish", "--md", md, "--folder-id", testFolderID)
		if code != 0 || got["ok"] != true {
			t.Fatalf("publish: %v (exit %d)", got, code)
		}
		assertDebts(t, got)
	})
}

// assertDebts reads body.marks and body.footnotes off an envelope.
func assertDebts(t *testing.T, got map[string]any) {
	t.Helper()
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	counts, ok := data["body"].(map[string]any)
	if !ok {
		t.Fatalf("no body counts: %v", data)
	}
	if counts["footnotes"] != float64(1) {
		t.Errorf("body.footnotes is %v, want the note's one", counts["footnotes"])
	}
	marks, ok := counts["marks"].(map[string]any)
	if !ok {
		t.Fatalf("body.marks is not an object: %v", counts["marks"])
	}
	if len(marks) != len(debtCounts) {
		t.Errorf("body.marks carries %d entries, want the three debts: %v", len(marks), marks)
	}
	for name, want := range debtCounts {
		if marks[name] != want {
			t.Errorf("body.marks.%s is %v, want %v", name, marks[name], want)
		}
	}
}

// markTopicRow is one mark as `gdoc help marks` must print it, every field a
// literal. The file house.yaml is read, and this table is not: a test that
// read the constant would follow the palette wherever somebody moved it.
type markTopicRow struct {
	name   string
	syntax []string
	kind   string
	colour string
	fate   string
}

// theSevenMarks is the palette as a colleague reads it out of the binary.
var theSevenMarks = []markTopicRow{
	{"yellow", []string{"==", "🟡"}, "highlight", "#FFD500", "counted"},
	{"green", []string{"🟢"}, "highlight", "#3DD67A", "counted"},
	{"purple", []string{"🟣"}, "highlight", "#BA92FF", "counted"},
	{"blue", []string{"🔵"}, "highlight", "#5DB3FF", "carried"},
	{"red", []string{"🔴"}, "highlight", "#FF6B6B", "carried"},
	{"orange", []string{"🟠"}, "highlight", "#FFB85C", "carried"},
	{"pencil", []string{"<!--", "%%"}, "text", "#9A9A9A", "carried"},
}

// `marks` is a word help answers and no command gdoc takes, so the help
// branches on it before the table is walked, which would otherwise refuse it
// the way an unknown command is refused. The seven come out of house.yaml, so
// a colour changed in the file is a colour the help prints, and nothing here
// or in a skill carries a second copy of the palette.
func TestHelpMarksPrintsTheSevenFromTheBinary(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, prose, code := runHelp(t, "help", "marks", "--json")
	if code != 0 || got["ok"] != true {
		t.Fatalf("gdoc help marks must be answered: %v (exit %d)", got, code)
	}
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	list, ok := data["marks"].([]any)
	if !ok {
		t.Fatalf("data.marks is not a list: %v", data)
	}
	if len(list) != len(theSevenMarks) {
		t.Fatalf("the help prints %d marks, want the seven: %v", len(list), list)
	}
	for i, want := range theSevenMarks {
		entry, ok := list[i].(map[string]any)
		if !ok {
			t.Fatalf("entry %d is not an object: %v", i, list[i])
		}
		if entry["name"] != want.name {
			t.Fatalf("entry %d is %v, want %s: the order is the palette's", i, entry["name"], want.name)
		}
		if entry["kind"] != want.kind {
			t.Errorf("%s: kind is %v, want %s", want.name, entry["kind"], want.kind)
		}
		if entry["color"] != want.colour {
			t.Errorf("%s: color is %v, want %s", want.name, entry["color"], want.colour)
		}
		if entry["fate"] != want.fate {
			t.Errorf("%s: fate is %v, want %s", want.name, entry["fate"], want.fate)
		}
		syntax, _ := entry["syntax"].([]any)
		if len(syntax) != len(want.syntax) {
			t.Fatalf("%s: syntax is %v, want %v", want.name, entry["syntax"], want.syntax)
		}
		for j, s := range want.syntax {
			if syntax[j] != s {
				t.Errorf("%s: syntax %d is %v, want %s", want.name, j, syntax[j], s)
			}
		}
	}
	// The command table is not the answer to this question, so none of it is
	// in the object: a skill reading data.commands here would read the help of
	// a command called marks, which there is none of.
	if _, said := data["commands"]; said {
		t.Errorf("the marks topic carries no command table: %v", data)
	}

	// The words a person reads carry every mark too, so a colleague who typed
	// the topic without --json is not sent to the object for the answer.
	for _, want := range theSevenMarks {
		if !strings.Contains(prose, want.name) || !strings.Contains(prose, want.colour) {
			t.Errorf("the words a person reads leave out %s: %q", want.name, prose)
		}
	}
}

// TestHelpMarksPrintsBothPencilSpellings. The grey mark has two spellings
// from milestone 20 on and no third: HTML's comment and Obsidian's, both
// read, with the Pandoc span that was the one spelling until then refused by
// line. A colleague asks the binary which to type, so the two come out of
// `gdoc help marks` and the old one does not.
func TestHelpMarksPrintsBothPencilSpellings(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, prose, code := runHelp(t, "help", "marks", "--json")
	if code != 0 || got["ok"] != true {
		t.Fatalf("gdoc help marks must be answered: %v (exit %d)", got, code)
	}
	data, _ := got["data"].(map[string]any)
	list, ok := data["marks"].([]any)
	if !ok {
		t.Fatalf("data.marks is not a list: %v", data)
	}
	var syntax []any
	for _, entry := range list {
		mark, _ := entry.(map[string]any)
		if mark["name"] == "pencil" {
			syntax, _ = mark["syntax"].([]any)
		}
	}
	want := []any{"<!--", "%%"}
	if len(syntax) != len(want) {
		t.Fatalf("the pencil's syntax is %v, want the two comment spellings %v", syntax, want)
	}
	for i, s := range want {
		if syntax[i] != s {
			t.Errorf("the pencil's syntax %d is %v, want %v", i, syntax[i], s)
		}
	}
	for _, screen := range []string{fmt.Sprint(syntax), prose} {
		if strings.Contains(screen, ".pencil") {
			t.Errorf("the help offers the span gdoc refuses: %q", screen)
		}
	}
}

// Tab offers the topic where a command would be offered, in both shells.
// Without it `gdoc help ma<Tab>` completes to nothing, and the one word that
// is not a command is the one word nobody can guess is there.
func TestBothCompletionScriptsListTheMarksTopic(t *testing.T) {
	for _, sh := range shells() {
		t.Run(sh.name, func(t *testing.T) {
			script, err := sh.render(commands())
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(script, markTopic) {
				t.Errorf("the %s script offers no %q after help", sh.name, markTopic)
			}
		})
	}
}

// A note the walker refuses is a note no document is written for. Both
// refusals name the line, and the file at --out is never made: a half a
// document under the name somebody asked for is worse than none.
func TestARefusedNoteWritesNothing(t *testing.T) {
	for _, c := range []struct {
		what string
		body string
		says string
	}{
		{
			what: "a dangling footnote",
			body: "A claim.[^a]\n",
			says: "[^a]",
		},
		{
			what: "a mark inside a mark",
			body: "A ==figure <!-- words --> to replace== here.\n",
			says: "a mark inside a mark",
		},
	} {
		t.Run(c.what, func(t *testing.T) {
			noSession(t)
			md := writeNote(t, "register.md", "---\ntitle: A Policy\n---\n\n# Purpose\n\n"+c.body)
			out := filepath.Join(filepath.Dir(md), "register.docx")

			got, code := runJSON(t, "build", "--md", md, "--out", out)
			if code == 0 || got["ok"] == true {
				t.Fatalf("%s must be refused: %v (exit %d)", c.what, got, code)
			}
			said, _ := got["error"].(string)
			if !strings.Contains(said, "line ") {
				t.Errorf("the refusal must name the line: %q", said)
			}
			if !strings.Contains(said, c.says) {
				t.Errorf("the refusal must name what is wrong: %q, want %q in it", said, c.says)
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Errorf("a refused note wrote a file at --out: %v", err)
			}
		})
	}
}

// A note carries two things an author has to be told about, and neither
// belongs to a flag: a footnote is set in the document, and the seven marks
// are syntax gdoc reads. So `gdoc help build` names each in one line and
// sends the question about the colours to `gdoc help marks`, which answers it
// out of house.yaml. The lines are in the object as well as on the screen,
// because a skill building a note is the other reader of this help.
func TestHelpBuildNamesFootnotesAndMarks(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, prose, code := runHelp(t, "help", "build", "--json")
	if code != 0 || got["ok"] != true {
		t.Fatalf("gdoc help build must be answered: %v (exit %d)", got, code)
	}
	list := helpCommands(t, got)
	if len(list) != 1 {
		t.Fatalf("gdoc help build is one entry and %d came back: %v", len(list), list)
	}
	entry, ok := list[0].(map[string]any)
	if !ok {
		t.Fatalf("the entry is not an object: %v", list[0])
	}
	notes, ok := entry["notes"].([]any)
	if !ok {
		t.Fatalf("build's entry carries no notes: %v", entry)
	}
	if len(notes) != 3 {
		t.Fatalf("the three lines are the footnote's, the marks' and the page break's, and %d came back: %v", len(notes), notes)
	}
	footnoteLine, _ := notes[0].(string)
	markLine, _ := notes[1].(string)
	if !strings.Contains(footnoteLine, "footnote") {
		t.Errorf("the first line is the footnote's: %q", footnoteLine)
	}
	if !strings.Contains(markLine, "gdoc help marks") {
		t.Errorf("the marks line must point at the topic that lists the seven: %q", markLine)
	}
	// The words a person reads carry both lines, so a colleague who typed
	// the help without --json is not sent to the object for them.
	for _, want := range []string{footnoteLine, markLine} {
		if !strings.Contains(prose, want) {
			t.Errorf("the words a person reads leave out %q: %q", want, prose)
		}
	}
}
