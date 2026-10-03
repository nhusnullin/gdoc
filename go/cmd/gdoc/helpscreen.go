// The screens a person reads instead of the help's object: the whole table
// grouped by job, one command on its own, and the words bare gdoc opens with.
//
// This is the room that decides which screen a command draws and what in it is
// a key, a title or a muted word. internal/tty answers what one stream is and
// holds the palette; internal/panel draws the boxes. So nothing here names a
// colour, a box-drawing character or an escape code, and nothing there knows
// what a command is.
//
// Every screen goes to stderr, and only where stderr is a terminal wide enough
// for a box. A pipe, a file, a window under fifty columns and a caller that
// asked for the object by name each get today's plain text, byte for byte,
// which the goldens in testdata/pipe hold.

package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gdoc/internal/panel"
	"gdoc/internal/tty"
)

// The four jobs a command does. A screen groups the table by them, because
// eighteen commands in one list is eighteen things to read and four groups is
// four: TestEveryCommandHasAGroup.
const (
	groupRead    = "Read"
	groupWrite   = "Write into a doc"
	groupMake    = "Make a doc"
	groupAccount = "Account and tool"
)

// helpGroups is the order the groups are drawn in: reading first, because that
// is what a session does first, and the account and the tool itself last.
var helpGroups = []string{groupRead, groupWrite, groupMake, groupAccount}

// needsCommand is what bare gdoc says, in the refusal's object and as the
// first line of its screen, so a person reads why they got a help they did not
// ask for: TestBareGdocOpensWithTheWordsItRefusesWith.
const needsCommand = "gdoc needs a command."

// usageAll is the line the whole binary is called by, and helpFoot the one
// that says where the detail is. Both are written once and printed twice, in
// today's text and on the screen, so neither can drift.
const (
	usageAll = "gdoc <command> [words] [flags]"
	helpFoot = "Run gdoc help <command> for the words and flags one takes."
)

// The marks that open a row carrying something a reader has to act on: the
// release notice, and a warning the envelope would have carried where the
// object had not been dropped. One column each, with a blank after them.
const (
	noticeMark = "▸"
	warnMark   = "!"
)

// The cells a column costs beside the longest thing in it. A command name
// takes the three internal/panel leaves a column nobody named: a blank each
// side of the cell and the bar. A group name takes two more, because it is
// drawn in the rule itself: the dash the border opens with, a blank each side
// of the name, the dash after it, and the joint.
const (
	cellPadding = 3
	rulePadding = 5
)

// screenAt is the panel a screen is drawn in and the style it is painted with,
// and whether there is a screen at all.
//
// Three answers mean there is not. A caller that asked for the object by name
// wants stderr plain beside it. A stream no terminal driver owns is a skill, a
// log or a file, and gets the bytes it always got. And a window under fifty
// columns has no room for a box, so it gets the plain text too, which is the
// Plain band of the width rule.
func screenAt(w io.Writer, json bool) (panel.Panel, tty.Style, bool) {
	if json || !isTerminal(w) {
		return panel.Panel{}, tty.Style{}, false
	}
	width := tty.Width(w, os.Getenv)
	if panel.Layout(width) == panel.Plain {
		return panel.Panel{}, tty.Style{}, false
	}
	style := tty.NewStyle(tty.Colour(os.Getenv))
	return panel.New(style, width), style, true
}

// writeHelp is the words of a help: the screen where there is one, and today's
// text everywhere else. The notice line opens the plain text as it always did,
// and inside a screen it is a row of the top box instead, because on a
// terminal the object that carried it is gone.
//
// TestHelpOnAPipeIsTodaysTextByteForByte holds the plain half and
// TestTheHelpScreensAreTheirRecordedBytes the drawn one.
func writeHelp(errOut io.Writer, matched []command, all bool, line string, warns []string, json bool) {
	p, style, ok := screenAt(errOut, json)
	if !ok {
		if line != "" {
			fmt.Fprint(errOut, line+"\n\n")
		}
		fmt.Fprint(errOut, helpProse(matched, all))
		return
	}
	writeLines(errOut, helpScreen(p, style, matched, all, line, warns))
}

// writeBareHelp is the second help screen: the words of the refusal, and the
// whole help under them. Bare gdoc asks GitHub nothing, so it carries no
// notice line, and it takes no flags, so there is never an object to keep.
func writeBareHelp(errOut io.Writer) {
	table := commands()
	p, style, ok := screenAt(errOut, false)
	if !ok {
		fmt.Fprint(errOut, helpProse(table, true))
		return
	}
	fmt.Fprintln(errOut, style.Fail(needsCommand))
	writeLines(errOut, helpScreen(p, style, table, true, "", nil))
}

// helpScreen is a drawn help: the whole table grouped by job, or one box for
// each command the words matched, with a blank line between two boxes.
func helpScreen(p panel.Panel, style tty.Style, matched []command, all bool, line string, warns []string) []string {
	if all {
		return tableScreen(p, style, matched, line, warns)
	}
	var out []string
	for i, c := range matched {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, commandScreen(p, style, c, line, warns)...)
		line, warns = "", nil
	}
	return out
}

// tableScreen is the screen bare `gdoc help` and bare `gdoc` draw: one box
// holding the usage line, whatever the daily check had to say, and every
// command under the job it does, with the line that says where the detail is
// under the box.
//
// The groups are the rules across the box, so the column has to clear the
// longest group name: a name that reaches it takes the joint's own cell, and
// the screen would lose the joints it draws the groups with.
func tableScreen(p panel.Panel, style tty.Style, matched []command, line string, warns []string) []string {
	groups := groupsOf(matched)
	p = p.WithColumn(commandColumn(groups))
	rows := []panel.Row{panel.Line(style.Key("Usage") + "  " + usageAll)}
	rows = append(rows, noticeRows(p, style, line, warns)...)
	for _, g := range groups {
		rows = append(rows, panel.Section(g.name))
		for _, c := range g.cmds {
			rows = append(rows, pairRows(p, c.name, c.summary)...)
		}
	}
	out := p.Box(gdocTitle(), fmt.Sprintf("%d commands", len(matched)), rows)
	for _, l := range panel.Wrap(helpFoot, p.Width()) {
		out = append(out, style.Dim(l))
	}
	return out
}

// commandScreen is one command in full: what it does, the line a person types,
// a row per flag, and the example under the box, flush left and on one line,
// so a copy runs in any shell.
func commandScreen(p panel.Panel, style tty.Style, c command, line string, warns []string) []string {
	p = p.WithColumn(flagColumn(c))
	rows := lineRows(p, c.summary)
	rows = append(rows, noticeRows(p, style, line, warns)...)
	rows = append(rows, panel.Section("Usage"))
	rows = append(rows, usageRows(p, style, c)...)
	if len(c.flags) > 0 {
		rows = append(rows, panel.Section("Flags"))
		for _, f := range c.flags {
			rows = append(rows, pairRows(p, style.Key(flagWords(f)), f.summary)...)
		}
	}
	return append(p.WithFooter("Example").Box(c.name, gdocLabel(), rows), c.example)
}

// gdocTitle is the name and the release, which is what the top border of the
// whole help carries so that a screenshot of something odd names the build
// that drew it. An untagged build is the name on its own.
func gdocTitle() string {
	if v := releaseVersion(); v != "" {
		return "gdoc " + v
	}
	return "gdoc"
}

// gdocLabel is that same build at the right end of one command's border, and
// nothing at all for a build from a checkout: a border saying only "gdoc" in a
// box the command's own name titles tells a reader nothing it did not know.
func gdocLabel() string {
	if v := releaseVersion(); v != "" {
		return "gdoc " + v
	}
	return ""
}

// commandGroup is one job and the commands that do it.
type commandGroup struct {
	name string
	cmds []command
}

// groupsOf is the matched commands by the job they do, in the groups' order and
// in the table's order inside each group, so a reader of the screen and a
// reader of the object meet them in one order. A group nothing matched is not
// drawn at all.
func groupsOf(matched []command) []commandGroup {
	out := make([]commandGroup, 0, len(helpGroups))
	for _, name := range helpGroups {
		var cmds []command
		for _, c := range matched {
			if c.group == name {
				cmds = append(cmds, c)
			}
		}
		if len(cmds) > 0 {
			out = append(out, commandGroup{name: name, cmds: cmds})
		}
	}
	return out
}

// commandColumn is where the column sits in the grouped screen: past the
// longest group name, and past the longest command name.
func commandColumn(groups []commandGroup) int {
	widest := 0
	for _, g := range groups {
		widest = max(widest, len(g.name)+rulePadding)
		for _, c := range g.cmds {
			widest = max(widest, len(c.name)+cellPadding)
		}
	}
	return widest
}

// flagColumn is where the column sits in one command's box: past its longest
// flag, which is what internal/panel would choose itself. It is named here
// because the summaries are wrapped to the cell before the box is drawn.
func flagColumn(c command) int {
	widest := 0
	for _, f := range c.flags {
		widest = max(widest, len(flagWords(f))+cellPadding)
	}
	return widest
}

// pairRows is a name and its words: beside each other where the window has
// eighty columns, and the words indented under the name below that. The wrap
// happens before the colour, because a wrap counts what it can see.
func pairRows(p panel.Panel, name, words string) []panel.Row {
	if panel.Layout(p.Width()) != panel.TwoColumns {
		return append([]panel.Row{panel.Line(name)}, indentRows(p, words)...)
	}
	var rows []panel.Row
	for i, l := range panel.Wrap(words, p.ColumnWidth()) {
		if i == 0 {
			rows = append(rows, panel.Pair(name, l))
			continue
		}
		rows = append(rows, panel.Pair("", l))
	}
	return rows
}

// lineRows is a paragraph across the whole box, and indentRows the same thing
// two columns in.
func lineRows(p panel.Panel, text string) []panel.Row {
	var rows []panel.Row
	for _, l := range panel.Wrap(text, p.TextWidth()) {
		rows = append(rows, panel.Line(l))
	}
	return rows
}

func indentRows(p panel.Panel, text string) []panel.Row {
	var rows []panel.Row
	for _, l := range panel.Wrap(text, p.TextWidth()-2) {
		rows = append(rows, panel.Line("  "+l))
	}
	return rows
}

// noticeRows is what the daily check had to say, and then every warning the
// envelope would have carried. They are rows of the box rather than lines
// above it because on a terminal the object that carried them is dropped, and
// a warning nobody reads is a warning nobody was given.
//
// The backticks go: the line is written for a reader who is about to run the
// command it names, and a backtick is markdown, which a terminal is not.
func noticeRows(p panel.Panel, style tty.Style, line string, warns []string) []panel.Row {
	var rows []panel.Row
	if line != "" {
		rows = append(rows, markedRows(p, style, noticeMark, strings.ReplaceAll(line, "`", ""))...)
	}
	for _, w := range warns {
		rows = append(rows, markedRows(p, style, warnMark, w)...)
	}
	return rows
}

// markedRows is one marked paragraph: the mark on the first line, and the
// lines under it indented to where its words start.
func markedRows(p panel.Panel, style tty.Style, mark, text string) []panel.Row {
	var rows []panel.Row
	for i, l := range panel.Wrap(text, p.TextWidth()-2) {
		if i == 0 {
			rows = append(rows, panel.Line(style.Warn(mark)+" "+l))
			continue
		}
		rows = append(rows, panel.Line("  "+l))
	}
	return rows
}

// usageRows is the line a person types, broken only between its parts: a flag
// keeps its placeholder and an optional part keeps its brackets, because a
// usage line split inside one of those teaches a call the binary refuses. A
// second line is indented two columns, so it reads as the first one carried on.
func usageRows(p panel.Panel, style tty.Style, c command) []panel.Row {
	parts := append([]string{"gdoc", c.name}, c.words...)
	parts = append(parts, usageFlags(c)...)

	var rows []panel.Row
	var line []string
	width := p.TextWidth()
	taken := 0
	for _, part := range parts {
		if len(line) > 0 && taken+1+len(part) > width {
			rows = append(rows, panel.Line(paintUsage(style, line)))
			line, taken, width = nil, 2, p.TextWidth()-2
		}
		if taken > 0 && len(line) > 0 {
			taken++
		}
		line = append(line, part)
		taken += len(part)
	}
	indent := ""
	if len(rows) > 0 {
		indent = "  "
	}
	return append(rows, panel.Line(indent+paintUsage(style, line)))
}

// paintUsage paints one line of a usage: a flag is a key and a placeholder is a
// title, which is the same pair of roles the flag rows under it use. The
// brackets, the bar and the command's own words keep the terminal's colour.
func paintUsage(style tty.Style, parts []string) string {
	painted := make([]string, 0, len(parts))
	for _, part := range parts {
		words := strings.Fields(part)
		for i, w := range words {
			words[i] = usageWord(style, w)
		}
		painted = append(painted, strings.Join(words, " "))
	}
	return strings.Join(painted, " ")
}

// usageWord is one word of a usage line, with whatever brackets it carries
// left as they were.
func usageWord(style tty.Style, word string) string {
	open, close := "", ""
	if strings.HasPrefix(word, "[") {
		open, word = "[", strings.TrimPrefix(word, "[")
	}
	if strings.HasSuffix(word, "]") {
		close, word = "]", strings.TrimSuffix(word, "]")
	}
	switch {
	case strings.HasPrefix(word, "--"):
		word = style.Key(word)
	case strings.HasPrefix(word, "<"):
		word = style.Title(word)
	}
	return open + word + close
}

// writeLines is a drawn screen on its stream, one line at a time.
func writeLines(w io.Writer, lines []string) {
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
}
