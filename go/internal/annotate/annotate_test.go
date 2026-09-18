package annotate

import (
	"strings"
	"testing"
)

// TestCheckRefusesEachBadShapeByName is the rule asked of every entry before
// the first one leaves the machine. Each refusal names what is wrong, because
// the caller is a skill writing a file and the file is what has to change.
func TestCheckRefusesEachBadShapeByName(t *testing.T) {
	for _, c := range []struct {
		name  string
		a     Annotation
		names string
	}{
		{
			name:  "an empty quote",
			a:     Annotation{Quoted: "", Why: "The 2026 register says quarterly."},
			names: "quotes no text",
		},
		{
			name:  "a quote carrying a line break",
			a:     Annotation{Quoted: "reviewed annually\nby the board", Why: "The 2026 register says quarterly."},
			names: "line break",
		},
		{
			name:  "an empty why",
			a:     Annotation{Quoted: "reviewed annually", Why: ""},
			names: "no reason",
		},
		{
			name:  "a why of nothing but spaces",
			a:     Annotation{Quoted: "reviewed annually", Why: "   \n  "},
			names: "no reason",
		},
		{
			name:  "a why already opening with the robot",
			a:     Annotation{Quoted: "reviewed annually", Why: "🤖 The 2026 register says quarterly."},
			names: "already opens with",
		},
		{
			name:  "a why carrying markdown",
			a:     Annotation{Quoted: "reviewed annually", Why: "The register says **quarterly**."},
			names: "**",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := c.a.Check()
			if err == nil {
				t.Fatalf("Check() of %+v gave no error, and %s is refused", c.a, c.name)
			}
			if !strings.Contains(err.Error(), c.names) {
				t.Errorf("Check() said %q, which does not name %q", err, c.names)
			}
		})
	}
}

// TestCheckAcceptsAPlainAnnotation is the other direction. An assignee is
// optional and decides nothing about the shape.
func TestCheckAcceptsAPlainAnnotation(t *testing.T) {
	for _, a := range []Annotation{
		{Quoted: "reviewed annually", Why: "The 2026 register says quarterly."},
		{Quoted: "the Cyprus entity", Why: "Named twice with two spellings.", Assignee: "x@altery.com"},
		{Quoted: "5 * 3 units", Why: "See issue #28 for the rest."},
	} {
		if err := a.Check(); err != nil {
			t.Errorf("Check() of %+v said %q, and there is nothing wrong with it", a, err)
		}
	}
}

// TestBodyIsTheRobotAndTheWhy states the prefix as a literal rather than
// reading the constant, because a test that reads the constant follows it
// wherever somebody moves it. The mark is the only record of authorship a
// comment has.
func TestBodyIsTheRobotAndTheWhy(t *testing.T) {
	if got, want := Body("The register."), "🤖 The register."; got != want {
		t.Errorf("Body(%q) = %q, want %q", "The register.", got, want)
	}
}
