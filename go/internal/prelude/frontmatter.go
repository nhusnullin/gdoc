package prelude

import (
	"fmt"

	"gdoc/internal/cover"
	"gdoc/internal/house"
)

// FrontMatter is the whole house prelude as requests, starting at the index it
// is given: the cover, the two labels, the three tables, the legend and the
// blanks between them, in the order house.yaml's own front_matter states them.
//
// The order is read, never decided. internal/render walks the same list to
// build the same template into a docx, and a second copy of the order here
// would be two documents that drift.
//
// One block cannot be written at all. The Docs API has no request that makes a
// contents list, measured and recorded in docs/v2/BLOCKED-BY-API.md, so the toc
// block is reported in Manual with the menu path and the heading above it is
// still written: a person inserting the list needs somewhere to put it.
//
// start is where the prelude goes, which is index 1 for a document gdoc is
// putting a template in front of. Nothing is sent here, and nothing here
// reaches the network.
func FrontMatter(cfg *house.Config, f cover.Fields, start int) (Result, error) {
	if cfg == nil {
		return Result{}, fmt.Errorf("prelude: no house style")
	}
	if start < 1 {
		return Result{}, fmt.Errorf("prelude: a document's body begins at index 1, and the front matter was asked for at %d", start)
	}
	b := &builder{cfg: cfg, fields: f, at: start}
	for i, block := range cfg.FrontMatter {
		b.block(i, block)
	}
	if b.tables > 0 {
		// house.yaml states every column width and every row height, and no
		// request kind here sets either: updateTableColumnProperties and
		// updateTableRowStyle were not among the nine the 2026-09-10 probe
		// measured as suggestible, and a request Docs refuses takes the whole
		// batch with it.
		b.manualStep(
			fmt.Sprintf("the column widths and row heights of the %d front-matter table(s)", b.tables),
			"click in the table, then Format > Table > Table properties")
	}
	if b.err != nil {
		return Result{}, fmt.Errorf("prelude: %w", b.err)
	}
	return Result{
		Requests:   b.requests,
		Start:      start,
		End:        b.at,
		Paragraphs: b.count,
		Tables:     b.tables,
		Cells:      b.cells,
		Manual:     b.manual,
	}, nil
}

// block writes one front_matter entry. A kind this writer does not have is
// refused naming it, rather than skipped: a front page silently missing its
// version control table is a document nobody would think to check.
func (b *builder) block(i int, block house.Block) {
	switch block.Block {
	case "cover":
		b.cover()
	case "page_break":
		// The break the cover used to write itself is a block of its own, so
		// house.yaml states where the page turns and both writers read it
		// there. internal/render makes the same block an empty paragraph
		// carrying w:pageBreakBefore.
		b.pageBreak()
	case "label":
		b.label(block.Ref)
	case "table":
		b.table(block.Ref)
	case "blank":
		count := block.Count
		if count == 0 {
			count = 1
		}
		for n := 0; n < count; n++ {
			b.blank(block.Align, block.SizePt, block.SpaceBeforePt, block.SpaceAfterPt, block.LineSpacing)
		}
	case "legend":
		b.legend()
	case "toc":
		b.manualStep("the contents list", "Insert > Table of contents")
	default:
		b.fail("front_matter[%d]: %q is not a block kind", i, block.Block)
	}
}

// label is one heading-like line in the front matter, named by a block's ref.
func (b *builder) label(ref string) {
	l, ok := b.cfg.Labels[ref]
	if !ok {
		b.fail("labels: no label named %q", ref)
		return
	}
	b.paragraph(
		[]run{{text: l.Text, look: b.textLook(mark{
			sizePt: l.SizePt, bold: l.Bold, italic: l.Italic, color: l.Color,
		})}},
		b.look(paraSpec{
			align: l.Align, before: l.SpaceBeforePt, after: l.SpaceAfterPt,
			line: l.LineSpacing, keepWithNext: l.KeepWithNext,
			keepLinesTogether: l.KeepLinesTogether,
		}),
	)
}

// legend is the key under the revision history: a bold word, a tab, and the
// sentence behind it.
//
// A tab is a character here, where the docx writer emits a w:tab element. The
// Docs API strips U+0000 to U+0008 and U+000C to U+001F out of an insert and
// leaves U+0009 alone, so the tab arrives as the author of the master typed it.
//
// The house legend carries no bullets, which is worth saying because the plan
// this was built from expected some. The lines are a bold word and a sentence,
// and the master's own markup has no numbering on them, so nothing here sends
// createParagraphBullets. If a later block needs one, the reason it may be sent
// here and is refused at LevelInPlace is the same one either way:
// createParagraphBullets removes the leading tabs that set a bullet's nesting
// level, so at LevelInPlace it would delete text an author typed, while here it
// would land on text gdoc itself proposed a moment earlier, where there are no
// author tabs to remove. The 2026-09-10 probe measured it accepted as a
// suggestion.
func (b *builder) legend() {
	g := b.cfg.Legend
	zero := 0.0
	for i, line := range g.Lines {
		if len(line) != 2 {
			b.fail("legend.lines[%d] has %d parts, want the word and the sentence", i, len(line))
			continue
		}
		b.paragraph([]run{
			{text: line[0] + "\t", look: b.textLook(mark{sizePt: g.SizePt, bold: true, color: g.Color})},
			{text: line[1], look: b.textLook(mark{sizePt: g.SizePt, color: g.Color})},
		}, b.look(paraSpec{before: &zero, after: &zero, line: g.LineSpacing}))
	}
}
