package prelude

import (
	"fmt"

	"gdoc/internal/cover"
	"gdoc/internal/house"
)

// Cover is the house cover as requests, starting at the index it is given.
//
// It is the same cover internal/render writes into a docx, read out of the
// same house.Cover: the leading blanks, the lines, the trailing blanks, and
// then the page break that puts what follows at the top of the next page. A
// line whose placeholder the fields filled in prints their words; a line whose
// placeholder they left empty prints the template's own, which are the
// highlighted words a person fills in by hand.
//
// start is where the prelude goes, which is index 1 for a document whose body
// gdoc is putting a cover in front of. It is a parameter rather than a
// constant because the front matter that follows in the same phase begins
// where this ends.
//
// Nothing is sent here. What comes back is a list of requests for the caller
// to send in writeMode SUGGEST, and a range for the marker.
func Cover(cfg *house.Config, f cover.Fields, start int) (Result, error) {
	if cfg == nil {
		return Result{}, fmt.Errorf("prelude: no house style")
	}
	if start < 1 {
		return Result{}, fmt.Errorf("prelude: a document's body begins at index 1, and the cover was asked for at %d", start)
	}
	b := &builder{cfg: cfg, fields: f, at: start}
	b.cover()
	b.pageBreak()
	if b.err != nil {
		return Result{}, fmt.Errorf("prelude: %w", b.err)
	}
	return Result{Requests: b.requests, Start: start, End: b.at, Paragraphs: b.count}, nil
}

// cover writes the first page, in the order house.yaml states it.
func (b *builder) cover() {
	c := b.cfg.Cover
	for _, blank := range c.LeadingBlanks {
		align := blank.Align
		if align == "" {
			align = c.Align
		}
		b.blank(align, blank.SizePt, blank.SpaceBeforePt, blank.SpaceAfterPt, blank.LineSpacing)
	}
	for _, line := range c.Lines {
		b.coverLine(c.Align, line)
	}
	for _, blank := range c.TrailingBlanks {
		b.blank(blank.Align, blank.SizePt, blank.SpaceBeforePt, blank.SpaceAfterPt, blank.LineSpacing)
	}
}

// coverLine is one line of the cover.
//
// A line that depends on a cover field the fields left out is not written at
// all. The master offers the title twice either side of an "or", for a person
// filling the cover in by hand to pick one, and writing both published a cover
// reading the title, then "or", then the template's own highlighted
// placeholder.
func (b *builder) coverLine(align string, line house.CoverLine) {
	if line.With != "" {
		if _, filled := b.placeholder(line.With); !filled {
			return
		}
	}
	value, filled := b.placeholder(line.Placeholder)
	look := b.paragraphLook(align, line.SpaceBeforePt, line.SpaceAfterPt, line.LineSpacing)

	switch {
	case filled:
		// The fields' own words, and none of the template's marks: the yellow
		// means "a person fills this in" and the red means "this is guidance",
		// and both are misleading once the value is there.
		b.paragraph([]run{{
			text: value,
			look: b.textLook(mark{sizePt: line.SizePt, bold: line.Bold}),
		}}, look)
	case len(line.Runs) > 0:
		// The template's own runs, marks and all. The line states the size and
		// the weight and each run states its own colour, which is the split
		// the docx writer makes: "Version: " is a run of its own with no
		// placeholder on it, so the label survives while the number beside it
		// is replaced.
		runs := make([]run, 0, len(line.Runs))
		for _, r := range line.Runs {
			text, m := b.runText(r, mark{
				sizePt: line.SizePt, bold: line.Bold,
				color: r.Color, highlight: r.Highlight,
			})
			runs = append(runs, run{text: text, look: b.textLook(m)})
		}
		b.paragraph(runs, look)
	case line.Text != "":
		b.paragraph([]run{{
			text: line.Text,
			look: b.textLook(mark{sizePt: line.SizePt, bold: line.Bold, highlight: line.Highlight}),
		}}, look)
	default:
		// A line with nothing to say is a blank paragraph at the line's own
		// height, which is what the master's own empty cover lines are.
		b.blank(align, line.SizePt, line.SpaceBeforePt, line.SpaceAfterPt, line.LineSpacing)
	}
}

// runText is one configured run's words and the marks it carries, with the
// fields' own value where they filled the run's placeholder in. It is
// internal/render's rule in this writer's units: a filled run loses the yellow
// and the red, because those mean "not filled in yet".
func (b *builder) runText(r house.Run, m mark) (string, mark) {
	value, filled := b.placeholder(r.Placeholder)
	if !filled {
		return r.Text, m
	}
	m.color, m.highlight = "", ""
	return value, m
}

// placeholder is the fields' own value for a named cover field, and whether
// they filled it in. The names are internal/cover's, so this writer and the
// docx writer cannot read one placeholder two ways.
func (b *builder) placeholder(name string) (string, bool) {
	value, filled, err := b.fields.Placeholder(name)
	if err != nil {
		b.fail("%s", err)
		return "", false
	}
	return value, filled
}

// pageBreak ends the cover with a page break inside a paragraph of gdoc's own,
// so what follows starts at the top of the next page.
//
// Inside gdoc's own paragraph, rather than at the start of the author's first
// one, for two reasons. The author's paragraph is theirs, and a break put at
// the front of it is a change to their text rather than an addition in front of
// it. And the marker a second run reads covers what gdoc proposed: a break
// living in the author's paragraph would sit outside that range, so a rejected
// prelude would leave it behind.
//
// The break takes one index unit and the paragraph mark takes another, so the
// paragraph spans two. That is the reference's own accounting for a PageBreak
// element, and the live acceptance is what confirms it against a real document.
func (b *builder) pageBreak() {
	start := b.at
	b.request("insertText", map[string]any{
		"text":     "\n",
		"location": map[string]any{"index": start},
	})
	// The break goes in before the mark that was just written, which is what
	// inserting at the paragraph's own start index means.
	b.request("insertPageBreak", map[string]any{
		"location": map[string]any{"index": start},
	})
	b.at = start + 2
	b.count++
	b.style("updateParagraphStyle", "paragraphStyle", span(start, b.at),
		b.paragraphLook("", nil, nil, nil))
	b.style("updateTextStyle", "textStyle", span(start, b.at), b.textLook(mark{}))
}
