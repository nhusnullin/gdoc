// The boxes themselves: the width rule, the rules with their titles and
// joints, and the rows between them.

package panel

import (
	"strings"

	"gdoc/internal/tty"
)

// The box drawing. Every one of these is one column wide on a terminal, which
// tty.VisibleWidth is what holds.
const (
	topLeft     = "┌"
	topRight    = "┐"
	sideLeft    = "├"
	sideRight   = "┤"
	bottomLeft  = "└"
	bottomRight = "┘"
	horizontal  = "─"
	vertical    = "│"
	jointDown   = "┬"
	jointCross  = "┼"
	jointUp     = "┴"
)

// The three bands of the width rule, and the widest box there is.
const (
	// twoColumnWidth is the first width that fits a name beside its
	// description.
	twoColumnWidth = 80
	// stackedWidth is the first width that fits a box at all. Under it a
	// caller draws plain lines and no box.
	stackedWidth = 50
	// maxWidth is how wide a box is drawn in a window wider than that. A line
	// past a hundred columns is more than an eye reads in one go.
	maxWidth = 100
	// The cells a box spends on its own drawing: a bar, a blank, the content,
	// a blank, a bar.
	rowFurniture = 4
	// cellFurniture is what a second cell costs on top of that: a bar and a
	// blank each side of it.
	cellFurniture = 3
)

// Kind is which of the three layouts a width gets.
type Kind int

const (
	// Plain is under fifty columns: no box at all, plain lines.
	Plain Kind = iota
	// Stacked is fifty to seventy-nine: a box, a name on its own line, its
	// words indented under it.
	Stacked
	// TwoColumns is eighty and over: a box with a name beside its words.
	TwoColumns
)

// String names a layout, so a test and a golden say which one they mean.
func (k Kind) String() string {
	switch k {
	case TwoColumns:
		return "two columns"
	case Stacked:
		return "stacked"
	default:
		return "plain"
	}
}

// Layout is the layout for a terminal width columns wide:
// TestTheLayoutBandsAreTheirEdges states every edge as a literal.
func Layout(width int) Kind {
	switch {
	case width >= twoColumnWidth:
		return TwoColumns
	case width >= stackedWidth:
		return Stacked
	default:
		return Plain
	}
}

// DrawWidth is how wide a box in a terminal width columns wide is drawn: the
// window itself up to a hundred columns, and a hundred past that.
// TestABoxIsNeverWiderThanAHundred holds it, and New puts every Panel through
// it, so no Panel can be wider.
func DrawWidth(width int) int {
	if width > maxWidth {
		return maxWidth
	}
	return width
}

// rowKind is what one row of a box is.
type rowKind int

const (
	// rowLine is one line across the whole box.
	rowLine rowKind = iota
	// rowPair is a name in the left cell and words in the right one.
	rowPair
	// rowSection is a rule across the box with a name in it.
	rowSection
)

// A Row is one line inside a box, or the rule that opens a section of it.
// Build one with Line, Pair or Section.
//
// The text in a Row is the caller's, coloured or not: a box pads it by its
// visible width and paints only its own drawing. So cmd/gdoc decides that a
// flag is a key and a placeholder is a title, and this package decides what a
// box looks like.
type Row struct {
	kind  rowKind
	left  string
	right string
}

// Line is one line across the whole box.
func Line(text string) Row { return Row{kind: rowLine, left: text} }

// Pair is a name and the words beside it, in the two cells the column divides.
// In a stacked screen a caller builds Lines instead, which is what the width
// goldens show.
func Pair(left, right string) Row { return Row{kind: rowPair, left: left, right: right} }

// Section is a rule across the box with its name in it:
// TestASectionIsARuleWithItsName.
func Section(name string) Row { return Row{kind: rowSection, left: name} }

// Panel draws boxes at one width, in one style. It knows no command: it takes
// strings and hands back lines.
//
// The zero Panel writes nothing anybody wants, so build one with New.
type Panel struct {
	style  tty.Style
	width  int
	column int
	footer string
}

// New is a Panel for a terminal width columns wide, painted with style. The
// width goes through DrawWidth, so a Panel is never wider than a hundred.
func New(style tty.Style, width int) Panel {
	return Panel{style: style, width: DrawWidth(width)}
}

// WithColumn is a copy of p whose column sits at that offset, which is the
// cell the joints and the second cell of every Pair line up on. Nothing is
// changed in p itself.
//
// A Panel nobody named a column for puts it three past its widest left cell:
// TestTheColumnFitsTheWidestLeft.
func (p Panel) WithColumn(column int) Panel {
	p.column = column
	return p
}

// WithFooter is a copy of p whose bottom border carries name, the way the top
// one carries a title. It is for a caller that prints something of its own
// under the box, so the bare line is not a line nobody introduced:
// TestTheFooterNameSitsInTheBottomBorder. Nothing is changed in p itself.
func (p Panel) WithFooter(name string) Panel {
	p.footer = name
	return p
}

// Width is how wide every line this Panel draws is.
func (p Panel) Width() int { return p.width }

// TextWidth is how wide the content of a Line row is, which is what a caller
// wraps its words to before handing them over.
func (p Panel) TextWidth() int { return p.width - rowFurniture }

// ColumnWidth is how wide the right cell of a Pair row is, for a Panel that
// names its column. A caller wraps a description to it.
func (p Panel) ColumnWidth() int { return p.width - p.columnAt(nil) - rowFurniture }

// Box draws rows inside a box, with title in the top border and an optional
// label at its right end: TestTheTitleSitsInTheTopBorder. Either may be empty,
// and then the border is a plain rule: TestABoxWithNoTitleIsAPlainRule. A
// title too wide for the border, or a label that would run into the title, is
// dropped rather than drawn over a corner:
// TestATitleTooWideForTheBorderIsDropped and
// TestALabelThatWouldRunIntoTheTitleIsDropped.
//
// The joints follow the pairs: the border above the first Pair is a down
// joint, a section with pairs each side of it is a cross, the one below the
// last pair is an up joint, and a box with no pair in it has none at all:
// TestTheColumnJointsAreDownCrossAndUp and TestABoxWithNoPairHasNoJoint.
func (p Panel) Box(title, label string, rows []Row) []string {
	column := p.columnAt(rows)
	segments := segmentsOf(rows)
	out := make([]string, 0, len(rows)+len(segments)+1)
	for i, s := range segments {
		above := i > 0 && hasPair(segments[i-1].rows)
		below := hasPair(s.rows)
		if i == 0 {
			out = append(out, p.drawRule(rule{
				left:      topLeft,
				right:     topRight,
				joint:     jointFor(false, below),
				title:     title,
				titleRole: roleTitle,
				label:     label,
			}, column))
		} else {
			out = append(out, p.drawRule(rule{
				left:      sideLeft,
				right:     sideRight,
				joint:     jointFor(above, below),
				title:     s.name,
				titleRole: roleKey,
			}, column))
		}
		for _, r := range s.rows {
			if r.kind == rowPair {
				out = append(out, p.pairRows(r.left, r.right, column)...)
				continue
			}
			out = append(out, p.lineRows(r.left)...)
		}
	}
	last := segments[len(segments)-1]
	out = append(out, p.drawRule(rule{
		left:      bottomLeft,
		right:     bottomRight,
		joint:     jointFor(hasPair(last.rows), false),
		title:     p.footer,
		titleRole: roleKey,
	}, column))
	return out
}

// columnAt is where the column sits for these rows: the offset the caller
// named, or three past the widest left cell. Either way it leaves at least one
// column in each cell, so a column nobody could honour is pulled back inside
// the box rather than drawn through its wall.
func (p Panel) columnAt(rows []Row) int {
	column := p.column
	if column == 0 {
		widest := 0
		for _, r := range rows {
			if r.kind != rowPair {
				continue
			}
			if w := tty.VisibleWidth(r.left); w > widest {
				widest = w
			}
		}
		column = widest + cellFurniture
	}
	if lowest := cellFurniture + 1; column < lowest {
		column = lowest
	}
	if highest := p.width - rowFurniture - 1; column > highest {
		column = highest
	}
	return column
}

// segment is the rows between two rules, and the name of the section that
// opens it. The first segment has no name: the box's own title opens it.
type segment struct {
	name string
	rows []Row
}

// segmentsOf splits rows at the sections. There is always at least one
// segment, so a box with no row at all is still two rules.
func segmentsOf(rows []Row) []segment {
	out := []segment{{}}
	for _, r := range rows {
		if r.kind == rowSection {
			out = append(out, segment{name: r.left})
			continue
		}
		last := len(out) - 1
		out[last] = segment{name: out[last].name, rows: append(out[last].rows, r)}
	}
	return out
}

// hasPair reports whether any of these rows is two cells, which is what
// decides whether the rules around them carry a joint.
func hasPair(rows []Row) bool {
	for _, r := range rows {
		if r.kind == rowPair {
			return true
		}
	}
	return false
}

// jointFor is the joint character for a rule with pairs above it, below it,
// both or neither.
func jointFor(above, below bool) string {
	switch {
	case above && below:
		return jointCross
	case below:
		return jointDown
	case above:
		return jointUp
	default:
		return ""
	}
}

// role is what one piece of a drawn line is for. The box paints its own
// drawing and its own titles, and hands a caller's text through untouched.
type role int

const (
	roleBorder role = iota
	roleTitle
	roleKey
	roleDim
)

// paint is one role's colour around text, through the one palette.
func (p Panel) paint(r role, text string) string {
	switch r {
	case roleTitle:
		return p.style.Title(text)
	case roleKey:
		return p.style.Key(text)
	case roleDim:
		return p.style.Dim(text)
	default:
		return p.style.Border(text)
	}
}

// rule is one of the three rules a box is made of: its corners, the joint
// where the column crosses it, a title at its left and a label at its right.
type rule struct {
	left      string
	right     string
	joint     string
	title     string
	titleRole role
	label     string
}

// cell is one column of a rule, and the role it is painted in. A rule is built
// a column at a time and then painted in runs, so the title keeps its own
// colour inside the border's.
type cell struct {
	text string
	role role
}

// drawRule draws one rule. The dashes run corner to corner, the joint replaces
// the dash at the column, the title block starts two columns in, and the label
// block ends one dash before the far corner.
func (p Panel) drawRule(r rule, column int) string {
	cells := make([]cell, p.width)
	cells[0] = cell{text: r.left}
	for i := 1; i < p.width-1; i++ {
		cells[i] = cell{text: horizontal}
	}
	cells[p.width-1] = cell{text: r.right}
	if r.joint != "" && column > 0 && column < p.width-1 {
		cells[column] = cell{text: r.joint}
	}
	// The title starts past the first dash, with a blank each side of it.
	titleEnd := 2
	if block := blockOf(r.title); len(block) > 0 && 2+len(block) <= p.width-2 {
		for i, text := range block {
			cells[2+i] = cell{text: text, role: r.titleRole}
		}
		titleEnd = 2 + len(block)
	}
	// The label ends one dash before the corner, and is dropped where the
	// title already reaches that far.
	if block := blockOf(r.label); len(block) > 0 {
		if at := p.width - 2 - len(block); at >= titleEnd {
			for i, text := range block {
				cells[at+i] = cell{text: text, role: roleDim}
			}
		}
	}
	return p.join(cells)
}

// blockOf is a title or a label as columns, with one blank each side of it, and
// nothing at all for the empty string, which is a border nobody titled.
func blockOf(s string) []string {
	if s == "" {
		return nil
	}
	out := []string{" "}
	for _, r := range s {
		out = append(out, string(r))
	}
	return append(out, " ")
}

// join paints the cells in runs of one role, so a rule is a handful of escape
// sequences rather than one per column.
func (p Panel) join(cells []cell) string {
	var line strings.Builder
	run := strings.Builder{}
	current := cells[0].role
	for _, c := range cells {
		if c.role != current {
			line.WriteString(p.paint(current, run.String()))
			run.Reset()
			current = c.role
		}
		run.WriteString(c.text)
	}
	line.WriteString(p.paint(current, run.String()))
	return line.String()
}

// lineRows is one Line row, and the lines it takes where its text is wider
// than the box. A caller wraps its own words to TextWidth, so the wrap here is
// what keeps a box square when somebody forgets.
func (p Panel) lineRows(text string) []string {
	bar := p.paint(roleBorder, vertical)
	width := p.TextWidth()
	var out []string
	for _, line := range Wrap(text, width) {
		out = append(out, bar+" "+pad(line, width)+" "+bar)
	}
	return out
}

// pairRows is one Pair row: the name in the left cell, the words in the right
// one, and a line for each where either runs past its cell.
func (p Panel) pairRows(left, right string, column int) []string {
	bar := p.paint(roleBorder, vertical)
	leftWidth := column - cellFurniture
	rightWidth := p.width - column - rowFurniture
	leftLines := Wrap(left, leftWidth)
	rightLines := Wrap(right, rightWidth)
	rows := max(len(leftLines), len(rightLines))
	out := make([]string, 0, rows)
	for i := range rows {
		l, r := "", ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		out = append(out, bar+" "+pad(l, leftWidth)+" "+bar+" "+pad(r, rightWidth)+" "+bar)
	}
	return out
}
