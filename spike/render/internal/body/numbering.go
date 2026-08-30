package body

import (
	"regexp"
	"strconv"
	"strings"
)

// Headings that name themselves are never auto-numbered; "Appendix 2" would
// otherwise come out as "13-Appendix 2".
var unnumberedHeadingRE = regexp.MustCompile(
	`(?i)^\s*(appendix|appendices|addendum|addenda|annex(e|es|ure)?|` +
		`contents|glossary|schedule)\b`)

// Headings that number themselves are never auto-numbered either. "1. Key terms"
// would otherwise come out as "1-1. Key terms", and the two numbers disagree the
// moment the document has one unnumbered heading above the numbered ones. The
// author's numbers win, because those are the ones the prose cross-references.
//
// A separator is required, so a heading is only self-numbered when it says so:
// "1." and "1)" and "3.1" are numbers, "2026 plan" and "1.5x throughput" are not.
var numberedHeadingRE = regexp.MustCompile(`^\s*(\d+(?:\.\d+)+|\d+)([.)])?(\s|$)`)

// authoredNumber is the number a heading gives itself, or nil.
//
// A lone number with no separator is not one: "10 things we learned" is a
// heading, "10." is a section.
func authoredNumber(headingText string) []int {
	match := numberedHeadingRE.FindStringSubmatch(headingText)
	if match == nil {
		return nil
	}
	number, separator := match[1], match[2]
	if separator == "" && !strings.Contains(number, ".") {
		return nil
	}
	var parts []int
	for _, piece := range strings.Split(number, ".") {
		value, err := strconv.Atoi(piece)
		if err != nil {
			return nil
		}
		parts = append(parts, value)
	}
	return parts
}

// headingNumberer produces the template's "1-", "1.1-", "1.1.1-" prefixes.
//
// The master writes these as literal text rather than as Word list numbering, so
// we do the same. Reproducing the template beats being clever here: a real
// numbered-heading list would renumber differently in Word and Google Docs.
type headingNumberer struct {
	enabled  bool
	topLevel int
	counters [9]int
}

func newHeadingNumberer(enabled bool, topLevel int) *headingNumberer {
	return &headingNumberer{enabled: enabled, topLevel: topLevel}
}

// styleLevel maps a Markdown heading level onto a template Heading style.
//
// The shallowest heading in the document becomes Heading1, so a file whose top
// level is "##" still gets a 16pt Heading1 rather than a 14pt Heading2. This is
// independent of numbering: it applies even when numbering is off.
func (n *headingNumberer) styleLevel(level int) int {
	if v := level - n.topLevel + 1; v > 1 {
		return v
	}
	return 1
}

func (n *headingNumberer) prefix(level int, headingText string) string {
	if !n.enabled || unnumberedHeadingRE.MatchString(headingText) {
		return ""
	}
	index := level - n.topLevel
	if index < 0 {
		index = 0
	}
	if index >= len(n.counters) {
		index = len(n.counters) - 1
	}
	if authored := authoredNumber(headingText); authored != nil {
		n.follow(index, authored)
		return ""
	}
	n.counters[index]++
	n.clearBelow(index)
	parts := make([]string, 0, index+1)
	for _, counter := range n.counters[:index+1] {
		parts = append(parts, strconv.Itoa(counter))
	}
	return strings.Join(parts, ".") + "-"
}

// follow carries on from the author's number, so the next computed one follows
// it. Without this, "1-Introduction" then a hand-written "2. Scope" is followed
// by a computed "2-", and the document holds two headings numbered 2.
//
// A number that does not fit the depth it sits at is not this document's
// sequence: "3.1 GHz band" as a top-level heading is a frequency. It is left
// unnumbered, and it moves nothing.
func (n *headingNumberer) follow(index int, authored []int) {
	if len(authored) != index+1 {
		return
	}
	copy(n.counters[:index+1], authored)
	n.clearBelow(index)
}

func (n *headingNumberer) clearBelow(index int) {
	for deeper := index + 1; deeper < len(n.counters); deeper++ {
		n.counters[deeper] = 0
	}
}
