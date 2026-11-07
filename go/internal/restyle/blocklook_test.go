package restyle

import (
	"net/url"
	"strings"
	"testing"

	"gdoc/internal/guard"
	"gdoc/internal/propose"
)

// TestBlockLookIsTheHouseBodyLook is Nail's rule of 2026-11-07 on the side that
// supplies the look: a paragraph propose adds is body text at 12pt, justified,
// in Calibri, with 12pt below it, which is what build writes from a note and
// what a restyle writes over a Normal paragraph. The values are literals, the
// house rule for a style test, so moving one in house.yaml fails here.
func TestBlockLookIsTheHouseBodyLook(t *testing.T) {
	para, text := BlockLook(embeddedHouse(t))("NORMAL_TEXT", false, false)
	if para == nil || text == nil {
		t.Fatal("NORMAL_TEXT has no look, and it is the house's body text")
	}
	if got := para.Set["alignment"]; got != "JUSTIFIED" {
		t.Errorf("alignment = %v, want JUSTIFIED", got)
	}
	if got := magnitude(t, para.Set, "spaceAbove"); got != 0 {
		t.Errorf("spaceAbove = %v, want 0", got)
	}
	if got := magnitude(t, para.Set, "spaceBelow"); got != 12 {
		t.Errorf("spaceBelow = %v, want 12", got)
	}
	if got := magnitude(t, text.Set, "fontSize"); got != 12 {
		t.Errorf("fontSize = %v, want 12", got)
	}
	font, _ := text.Set["weightedFontFamily"].(map[string]any)
	if font["fontFamily"] != "Calibri" {
		t.Errorf("weightedFontFamily = %v, want Calibri", text.Set["weightedFontFamily"])
	}
}

// TestBlockLookSpacesAListByTheListGap is the list half. A list item that is
// not the last has 3pt below it, and the last has the body's 12pt, so the list
// ends the way a paragraph does: the rule restyle and build already share.
func TestBlockLookSpacesAListByTheListGap(t *testing.T) {
	look := BlockLook(embeddedHouse(t))
	item, _ := look("NORMAL_TEXT", true, false)
	if got := magnitude(t, item.Set, "spaceBelow"); got != 3 {
		t.Errorf("a list item has %vpt below it, want 3", got)
	}
	last, text := look("NORMAL_TEXT", true, true)
	if got := magnitude(t, last.Set, "spaceBelow"); got != 12 {
		t.Errorf("the last list item has %vpt below it, want 12", got)
	}
	if got := magnitude(t, text.Set, "fontSize"); got != 12 {
		t.Errorf("a list item is %vpt, want 12", got)
	}
}

// TestBlockLookGivesAHeadingItsOwnSize is the reason the look is per named
// style rather than one body look: a heading in a block is a house heading.
func TestBlockLookGivesAHeadingItsOwnSize(t *testing.T) {
	_, text := BlockLook(embeddedHouse(t))("HEADING_2", false, false)
	if got := magnitude(t, text.Set, "fontSize"); got != 14 {
		t.Errorf("a Heading 2 is %vpt, want 14", got)
	}
}

// TestBlockLookLeavesAStyleWithNoHouseLookAlone is the same rule a restyle
// holds: a style outside the house file gets nothing rather than the nearest
// look.
func TestBlockLookLeavesAStyleWithNoHouseLookAlone(t *testing.T) {
	para, text := BlockLook(embeddedHouse(t))("SOMETHING_ELSE", false, false)
	if para != nil || text != nil {
		t.Errorf("an unknown style got a look: %v, %v", para, text)
	}
}

// TestTheWorstBlockWithTheHouseLookStaysUnderThePeek is propose's
// TestALargeBlockStaysUnderThePeek asked with the look a command really hands.
// It lives here because propose cannot import restyle. A heading's text look is
// not the body's, so headings alternating with body text group nothing, and that
// is the shape the house look makes largest: the review of 2026-11-07 found it
// past the peek, where a test look that was the same for every style had hidden
// it. The ceiling is a literal, the house rule for a value a test pins.
func TestTheWorstBlockWithTheHouseLookStaysUnderThePeek(t *testing.T) {
	const maxPeek = 1 << 20
	look := BlockLook(embeddedHouse(t))
	for name, unit := range map[string]string{
		"headings alternating with body text":   "# a\n\nb\n\n",
		"headings alternating with list items":  "# a\n\n- b\n\n",
		"two heading levels alternating":        "# a\n\n## b\n\n",
		"body text alternating with list items": "a\n\n- b\n\n",
		"the shortest paragraphs there are":     "a\n\n",
	} {
		content := strings.Repeat(unit, propose.MaxContent/len(unit))
		paras, err := propose.ParseContent(content)
		if err != nil {
			t.Fatalf("%s: parsing the content: %v", name, err)
		}
		// An index of seven digits, so every range is as long as it gets in a
		// document of a few hundred pages.
		raw := propose.BlockBatch(propose.Placement{Insert: 1_000_000}, paras, look, "why", "", "⁘")
		if len(raw) >= maxPeek {
			t.Errorf("%s at the content limit builds %d bytes, and the guard reads only the first %d", name, len(raw), maxPeek)
		}
		p := guard.NewPolicy()
		p.AllowFile("doc", guard.LevelSuggest)
		u, err := url.Parse(propose.BatchURL("doc"))
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Judge("POST", u, raw); err != nil {
			t.Errorf("%s: the guard refused a batch of %d bytes: %v", name, len(raw), err)
		}
	}
}
