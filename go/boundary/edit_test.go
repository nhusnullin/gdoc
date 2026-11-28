package boundary

import "testing"

// TestGrantEditHasTwoCallers holds the edit door to the two commands Nail
// decided on, 2026-11-28: `propose --edit` and `seen`. GrantEdit is called in
// one function, openEdit, and that function is called from those two. A third
// caller of either is a second way into a handed-in document's words, and it is
// a decision rather than a refactor.
func TestGrantEditHasTwoCallers(t *testing.T) {
	grants, err := callsOf("GrantEdit")
	if err != nil {
		t.Fatal(err)
	}
	if len(grants) != 1 || grants[0].file != "cmd/gdoc/edit.go" || grants[0].fn != "openEdit" {
		t.Errorf("GrantEdit is called once, from openEdit in cmd/gdoc/edit.go, and this tree calls it from %v", grants)
	}
	opens, err := callsOf("openEdit")
	if err != nil {
		t.Fatal(err)
	}
	fns := map[string]bool{}
	for _, c := range opens {
		fns[c.fn] = true
	}
	if len(opens) != 2 || !fns["cmdProposeEdit"] || !fns["cmdSeen"] {
		t.Errorf("openEdit is called by cmdProposeEdit and cmdSeen, and this tree calls it from %v", opens)
	}
	spans, err := callsOf("AllowEditSpans")
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 1 || spans[0].file != "cmd/gdoc/edit.go" {
		t.Errorf("AllowEditSpans is called once, from cmd/gdoc/edit.go, and this tree calls it from %v", spans)
	}
}
