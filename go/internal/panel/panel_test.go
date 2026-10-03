package panel

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"gdoc/internal/tty"
)

// plain is a box nobody coloured, which is what a width golden is drawn with.
func plain(width int) Panel { return New(tty.Style{}, width) }

// The three bands and their edges, as literals. Forty-nine is the last width
// that gets no box at all, eighty the first that gets two columns, and a
// hundred the widest box there is.
func TestTheLayoutBandsAreTheirEdges(t *testing.T) {
	for _, c := range []struct {
		width int
		want  Kind
	}{
		{width: 1, want: Plain},
		{width: 49, want: Plain},
		{width: 50, want: Stacked},
		{width: 79, want: Stacked},
		{width: 80, want: TwoColumns},
		{width: 100, want: TwoColumns},
		{width: 160, want: TwoColumns},
	} {
		if got := Layout(c.width); got != c.want {
			t.Errorf("%d columns is %v and the rule says %v", c.width, got, c.want)
		}
	}
}

// A box is drawn at the terminal's width up to a hundred, so a very wide window
// does not stretch a line past what an eye reads in one go.
func TestABoxIsNeverWiderThanAHundred(t *testing.T) {
	for _, c := range []struct{ width, want int }{
		{width: 44, want: 44},
		{width: 80, want: 80},
		{width: 100, want: 100},
		{width: 160, want: 100},
		{width: 1000, want: 100},
	} {
		if got := DrawWidth(c.width); got != c.want {
			t.Errorf("DrawWidth(%d) is %d and the rule says %d", c.width, got, c.want)
		}
		if got := plain(c.width).Width(); got != c.want {
			t.Errorf("a panel built at %d is %d wide and the rule says %d", c.width, got, c.want)
		}
	}
}

// The title sits in the top border with one space each side of it, and a label
// ends one dash before the corner.
func TestTheTitleSitsInTheTopBorder(t *testing.T) {
	lines := plain(40).Box("update", "v2.9.0", []Row{Line("looking")})

	want := []string{
		"┌─ update ──────────────────── v2.9.0 ─┐",
		"│ looking                              │",
		"└──────────────────────────────────────┘",
	}
	if len(lines) != len(want) {
		t.Fatalf("a box with one row is three lines and this is %d: %q", len(lines), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d is\n%q\nand the box is\n%q", i, lines[i], want[i])
		}
	}
}

// A box with no title is a plain rule, and so is one with no label.
func TestABoxWithNoTitleIsAPlainRule(t *testing.T) {
	lines := plain(20).Box("", "", []Row{Line("hi")})

	if lines[0] != "┌──────────────────┐" {
		t.Errorf("the top border is %q", lines[0])
	}
	if lines[1] != "│ hi               │" {
		t.Errorf("the row is %q", lines[1])
	}
	if lines[2] != "└──────────────────┘" {
		t.Errorf("the bottom border is %q", lines[2])
	}
}

// A section is a rule across the box with its name in it.
func TestASectionIsARuleWithItsName(t *testing.T) {
	lines := plain(24).Box("", "", []Row{Line("a"), Section("Read"), Line("b")})

	if lines[2] != "├─ Read ───────────────┤" {
		t.Errorf("the section rule is %q", lines[2])
	}
}

// The joint is a down joint where the column opens, a cross where it carries
// on through a section, and an up joint where it closes.
func TestTheColumnJointsAreDownCrossAndUp(t *testing.T) {
	p := plain(40).WithColumn(12)
	lines := p.Box("gdoc", "", []Row{
		Line("usage"),
		Section("Read"),
		Pair("read", "a document"),
		Section("Write"),
		Pair("reply", "a thread"),
	})

	for _, c := range []struct {
		at   int
		want string
	}{
		{at: 0, want: "─"}, // the top border has no pair under it
		{at: 2, want: "┬"}, // Read opens the column
		{at: 4, want: "┼"}, // Write carries it through
		{at: 6, want: "┴"}, // the bottom closes it
	} {
		if got := string([]rune(lines[c.at])[12]); got != c.want {
			t.Errorf("line %d holds %q at the column and the rule says %q", c.at, got, c.want)
		}
	}
}

// A section name long enough to reach the column takes the joint's own cell.
// The name is what a person reads, so it wins, and a caller that wants its
// joints leaves room for its longest name.
func TestATitleThatReachesTheColumnTakesTheJoint(t *testing.T) {
	lines := plain(40).WithColumn(8).Box("gdoc", "", []Row{
		Section("Write into a doc"),
		Pair("reply", "a thread"),
	})

	if strings.ContainsRune(lines[1], '┬') {
		t.Errorf("the name reaches the column, so the rule carries no joint: %q", lines[1])
	}
	if !strings.Contains(lines[1], "Write into a doc") {
		t.Errorf("the name is what a person reads and the rule is %q", lines[1])
	}
}

// A box with no pair in it has no column and no joint anywhere.
func TestABoxWithNoPairHasNoJoint(t *testing.T) {
	lines := plain(30).Box("gdoc", "", []Row{Line("a"), Section("Read"), Line("b")})

	for i, line := range lines {
		if strings.ContainsAny(line, "┬┼┴") {
			t.Errorf("line %d holds a joint and nothing in the box is two columns: %q", i, line)
		}
	}
}

// The column sits three past the widest left cell when nobody named one, so the
// names fit and the descriptions start together.
func TestTheColumnFitsTheWidestLeft(t *testing.T) {
	lines := plain(40).Box("", "", []Row{Pair("read", "one"), Pair("suggestions", "two")})

	at := strings.IndexRune(lines[0], '┬')
	if at < 0 {
		t.Fatalf("the top border has a pair under it and no joint: %q", lines[0])
	}
	if got := len([]rune(lines[0][:at])); got != 14 {
		t.Errorf("the joint is at column %d and the widest left is eleven, so the rule says 14", got)
	}
}

// A row the caller coloured is padded by what it takes on a screen and not by
// how many bytes it holds, so a coloured line and a plain one are one width.
func TestAStyledRowIsPaddedByItsVisibleWidth(t *testing.T) {
	st := tty.NewStyle(tty.Sixteen)
	lines := New(st, 30).Box("", "", []Row{Line(st.Key("read")), Pair(st.Key("reply"), "a thread")})

	for i, line := range lines {
		if !strings.ContainsRune(line, 0x1b) {
			t.Errorf("line %d is drawn at sixteen colours and holds no escape: %q", i, line)
		}
		if got := tty.VisibleWidth(line); got != 30 {
			t.Errorf("line %d measures %d columns and the box is 30: %q", i, got, line)
		}
	}
}

// Wrap hands back a line that fits, unchanged, which is what keeps a caller's
// own two spaces of indent in a stacked screen.
func TestWrapKeepsALineThatFits(t *testing.T) {
	got := Wrap("  a document, read", 40)

	if len(got) != 1 || got[0] != "  a document, read" {
		t.Errorf("a line that fits is itself and this is %q", got)
	}
}

// Wrap breaks at the spaces, never inside a word that fits.
func TestWrapBreaksAtSpaces(t *testing.T) {
	got := Wrap("one two three four five", 9)

	want := []string{"one two", "three", "four five"}
	if len(got) != len(want) {
		t.Fatalf("nine columns of those words is %q and the rule says %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d is %q and the rule says %q", i, got[i], want[i])
		}
	}
}

// A word longer than the line, which is what a checksum in a reason is, starts
// a line of its own and is cut where that line ends. Nothing is shortened: the
// pieces put back together are the word, and no ellipsis stands for anything.
func TestAHashIsCutNotShortened(t *testing.T) {
	hash := strings.Repeat("ab", 20)

	got := Wrap("want "+hash, 10)

	if len(got) != 5 {
		t.Fatalf("a forty-character word after a short one is five lines of ten and this is %q", got)
	}
	if got[0] != "want" {
		t.Errorf("the long word starts its own line, so the first is %q", got[0])
	}
	if joined := strings.Join(got[1:], ""); joined != hash {
		t.Errorf("the pieces are %q and the word was %q", joined, hash)
	}
	for i, line := range got {
		if strings.ContainsRune(line, '…') {
			t.Errorf("line %d was shortened: %q", i, line)
		}
	}
}

// A box asked for at a width too narrow for its title keeps the box: the title
// is dropped rather than drawn over the corner.
func TestATitleTooWideForTheBorderIsDropped(t *testing.T) {
	lines := plain(8).Box("suggestions", "", []Row{Line("x")})

	if lines[0] != "┌──────┐" {
		t.Errorf("the top border is %q", lines[0])
	}
}

// A label that would run into the title is dropped, for the same reason.
func TestALabelThatWouldRunIntoTheTitleIsDropped(t *testing.T) {
	lines := plain(20).Box("auth status", "gdoc v2.9.0", []Row{Line("x")})

	if lines[0] != "┌─ auth status ────┐" {
		t.Errorf("the top border is %q", lines[0])
	}
}

// The width goldens: the same screen at a hundred, eighty, sixty and
// forty-four columns, each built the way its own layout says.
func TestTheScreenAtEveryWidth(t *testing.T) {
	for _, width := range []int{100, 80, 60, 44} {
		golden(t, screenName(width, tty.NoColour), sampleScreen(tty.Style{}, width))
	}
}

// The depth goldens: one screen per depth at eighty columns, so the bytes a
// terminal gets are written down for each.
func TestTheScreenAtEveryDepth(t *testing.T) {
	for _, d := range []tty.Depth{tty.Sixteen, tty.TwoFiftySix, tty.TrueColour} {
		golden(t, screenName(80, d), sampleScreen(tty.NewStyle(d), 80))
	}
}

// Every line of every golden measures at most the width it was drawn at, with
// the escapes not counted, and every border line measures it exactly.
func TestNoLineIsWiderThanItsBox(t *testing.T) {
	names, err := filepath.Glob(filepath.Join("testdata", "*.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if len(names) < 7 {
		t.Fatalf("there are four width goldens and three depth goldens, and testdata holds %d", len(names))
	}
	for _, name := range names {
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		width := widthOf(t, filepath.Base(name))
		for i, line := range strings.Split(strings.TrimRight(string(body), "\n"), "\n") {
			got := tty.VisibleWidth(line)
			if got > width {
				t.Errorf("%s line %d measures %d columns and the box is %d: %q", name, i, got, width, line)
			}
			if strings.HasPrefix(line, "┌") || strings.HasPrefix(line, "├") || strings.HasPrefix(line, "└") {
				if got != width {
					t.Errorf("%s line %d is a border and measures %d of %d: %q", name, i, got, width, line)
				}
			}
		}
	}
}

// screenName is the file one golden lives in: the width, and the depth where it
// is not the plain one.
func screenName(width int, d tty.Depth) string {
	suffix := map[tty.Depth]string{
		tty.NoColour:    "",
		tty.Sixteen:     "-16",
		tty.TwoFiftySix: "-256",
		tty.TrueColour:  "-truecolor",
	}[d]
	return "screen-" + strconv.Itoa(width) + suffix + ".golden"
}

// widthOf reads the width back out of a golden's name, which is what the width
// assertion measures against.
func widthOf(t *testing.T, name string) int {
	t.Helper()
	part := strings.TrimPrefix(name, "screen-")
	part = strings.SplitN(part, "-", 2)[0]
	part = strings.TrimSuffix(part, ".golden")
	n, err := strconv.Atoi(part)
	if err != nil {
		t.Fatalf("%s does not name its width: %v", name, err)
	}
	return DrawWidth(n)
}

// sample is the commands a golden screen is drawn from: two groups, and
// descriptions long enough to wrap at sixty columns and at forty-four.
var sample = []struct {
	group string
	cmds  [][2]string
}{
	{group: "Read", cmds: [][2]string{
		{"read", "a document as text, one tab or all of them, with the suggestions marked"},
		{"comments", "the threads, their quotes and who wrote them"},
	}},
	{group: "Write into a doc", cmds: [][2]string{
		{"propose", "a change to quoted words as a native suggestion, with a comment beside it"},
		{"reply", "one robot reply into a thread nobody has closed"},
	}},
}

// sampleScreen draws the sample at one width, the way a caller does: two
// columns from eighty, and a name with its description under it below that.
func sampleScreen(st tty.Style, width int) string {
	p := New(st, width).WithColumn(21)
	rows := []Row{Line("Usage: gdoc <command> [flags]")}
	two := Layout(width) == TwoColumns
	for _, g := range sample {
		rows = append(rows, Section(g.group))
		for _, c := range g.cmds {
			if two {
				for i, line := range Wrap(c[1], p.ColumnWidth()) {
					if i == 0 {
						rows = append(rows, Pair(st.Key(c[0]), line))
						continue
					}
					rows = append(rows, Pair("", line))
				}
				continue
			}
			rows = append(rows, Line(st.Key(c[0])))
			for _, line := range Wrap(c[1], p.TextWidth()-2) {
				rows = append(rows, Line("  "+line))
			}
		}
	}
	return strings.Join(p.Box("gdoc", "v2.9.0", rows), "\n") + "\n"
}

// golden compares a drawn screen with the file that holds it. A golden that is
// not there yet is written and the test fails saying so, because a recorded
// screen is something a person reads once.
func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	want, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Fatalf("%s held nothing, so this run wrote it. Read it, then run the test again.", path)
	}
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("%s is\n%s\nand the recorded screen is\n%s", name, got, string(want))
	}
}
