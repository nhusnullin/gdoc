package main

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"gdoc/internal/annotate"
	"gdoc/internal/propose"
)

// TestAnUnknownFieldIsRefusedWithTheFieldsTheEntryTakes is what a session
// with no memory file needs: the name it guessed, the entry it guessed it in,
// and the names gdoc takes, so the second attempt is the right one.
func TestAnUnknownFieldIsRefusedWithTheFieldsTheEntryTakes(t *testing.T) {
	f := stubWire(t, &fakeWire{})
	from := tempFile(t, "proposals.json",
		`[{"quoted":"reviewed annually","replacement":"reviewed quarterly","reason":"the 2026 register"}]`)

	got, code := runJSON(t, "propose", proposeDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("an unknown field must stop the run: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	for _, want := range []string{
		`proposals[0] has a field gdoc does not take: "reason"`,
		"A words entry takes quoted, replacement, why and assignee.",
		"A block entry takes kind, after or replace_from and replace_to, content, why and assignee.",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal must say %q:\n%s", want, msg)
		}
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may reach Google: %v", f.calls)
	}
}

// TestAnUnknownFieldNamesTheEntryItIsIn: the second entry is the one with the
// stray field, and the refusal says so.
func TestAnUnknownFieldNamesTheEntryItIsIn(t *testing.T) {
	stubWire(t, &fakeWire{})
	from := tempFile(t, "proposals.json",
		`[{"quoted":"a","replacement":"b","why":"c"},{"quoted":"d","replacement":"e","why":"f","note":"g"}]`)

	got, _ := runJSON(t, "propose", proposeDocID, "--from", from)
	if msg, _ := got["error"].(string); !strings.Contains(msg, `proposals[1] has a field gdoc does not take: "note"`) {
		t.Errorf("the refusal must name the second entry: %q", msg)
	}
}

// TestAnUnknownAnnotationFieldIsRefusedWithTheFieldsItTakes is the same
// answer for annotate --from.
func TestAnUnknownAnnotationFieldIsRefusedWithTheFieldsItTakes(t *testing.T) {
	f := stubWire(t, &fakeWire{})
	from := tempFile(t, "annotations.json",
		`[{"quoted":"reviewed annually","reason":"The 2026 register says quarterly."}]`)

	got, code := runJSON(t, "annotate", annotateDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("an unknown field must stop the run: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	for _, want := range []string{
		`annotations[0] has a field gdoc does not take: "reason"`,
		"An entry takes quoted, why and assignee.",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal must say %q:\n%s", want, msg)
		}
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may reach Google: %v", f.calls)
	}
}

// TestTheFieldListsNameEveryTag keeps each list honest against its struct:
// every JSON name the decoder takes is a word of the sentence that lists them.
func TestTheFieldListsNameEveryTag(t *testing.T) {
	for _, c := range []struct {
		list string
		of   any
	}{
		{propose.Fields, propose.Proposal{}},
		{annotate.Fields, annotate.Annotation{}},
	} {
		words := map[string]bool{}
		for _, w := range regexp.MustCompile(`[a-z_]+`).FindAllString(c.list, -1) {
			words[w] = true
		}
		ty := reflect.TypeOf(c.of)
		for i := 0; i < ty.NumField(); i++ {
			name, _, _ := strings.Cut(ty.Field(i).Tag.Get("json"), ",")
			if name != "" && name != "-" && !words[name] {
				t.Errorf("%s takes %q, and the list a refusal gives does not name it: %q", ty.Name(), name, c.list)
			}
		}
	}
}
