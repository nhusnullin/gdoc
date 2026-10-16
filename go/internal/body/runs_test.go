package body

// The run's own properties: the character style a run may name, and the order
// the properties are written in. Both parts of this milestone need them: a
// footnote reference names FootnoteReference and a marked run names its
// colour's style.

import (
	"testing"

	"github.com/beevik/etree"
)

// TestARunMayNameItsCharacterStyle. w:rStyle is the first element of the run
// properties, so a run that names one and states nothing else still writes an
// rPr: runOpts.empty() counts the style.
func TestARunMayNameItsCharacterStyle(t *testing.T) {
	p := etree.NewElement("w:p")
	textRun(p, "pencilled", runOpts{Style: "Pencil"})

	rPr := p.FindElement("w:r/w:rPr")
	if rPr == nil {
		t.Fatal("a run naming a character style carries no w:rPr")
	}
	first := rPr.ChildElements()[0]
	if first.FullTag() != "w:rStyle" {
		t.Fatalf("the run's first property is %s, want w:rStyle", first.FullTag())
	}
	if got := first.SelectAttrValue("w:val", ""); got != "Pencil" {
		t.Errorf("w:rStyle = %q, want Pencil", got)
	}
}

// TestAddRunsNamesTheStyleItWasHanded. addRuns is what every paragraph writes
// its words through, and the style has to survive the copy it makes of the
// base formatting.
func TestAddRunsNamesTheStyleItWasHanded(t *testing.T) {
	r := &renderer{cfg: config(t), anchors: map[string]bool{}, deadAnchors: map[string]bool{}}
	p := etree.NewElement("w:p")
	r.addRuns(p, []Run{{Text: "pencilled"}}, runOpts{Style: "Pencil"})

	style := p.FindElement("w:r/w:rPr/w:rStyle")
	if style == nil {
		t.Fatal("addRuns wrote no w:rStyle")
	}
	if got := style.SelectAttrValue("w:val", ""); got != "Pencil" {
		t.Errorf("w:rStyle = %q, want Pencil", got)
	}
}
