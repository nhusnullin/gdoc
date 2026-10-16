// The round trip comparison, and the named list of differences it rests on. It
// is beside export_test.go rather than in it because that file is the four live
// tests and this one is the reading of two files that says whether they hold
// the same words.
//
// A note and the export of the document it was published into are two spellings
// of the same words, and the comparison is over lines: the note's paragraphs
// unwrapped, the file's lines as it wrote them, both trimmed and with runs of
// spaces collapsed, because a soft wrap and a table's padding are not
// differences in anything. What is left is compared as a bag rather than a
// sequence: a line that moved is still a line both sides hold, and a reordering
// would show up as the pair of lines that no longer match.
//
// Every remaining difference goes through the named list. A name may say the
// note's line reaches no line of the document at all, how the note's line has to
// be read for the document to hold it, how the file's line has to be read for
// the note to hold it, or that a line in the file is one the note never had. The
// names that fired are logged with their counts, so nothing is excused quietly,
// and a line no name covers fails the run.

package live

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// noteLine is one line of a note as the comparison reads it.
type noteLine struct {
	Text string
	// FrontMatter and Code say where the line stood, because a line's own
	// characters cannot say it was a key or that it was inside a fence.
	FrontMatter bool
	Code        bool
}

// knownDifference is one reason a line of a note is not a line of the export of
// the document that note was published into.
//
// Reason is why, in a sentence, and it is the field that makes the list worth
// keeping: a name with no reason is a difference somebody stopped looking at.
type knownDifference struct {
	Name   string
	Reason string
	// Gone says the note's line reaches no line of the file at all.
	Gone func(l noteLine) bool
	// Note is how the note's line has to be read for the document to hold it,
	// and the line itself when this difference has nothing to say about it.
	Note func(s string) string
	// Export is the same the other way round, for a line of the file.
	Export func(s string) string
	// Added says a line of the file the note never held is this one's.
	Added func(s string) bool
}

var (
	footnoteDefinition = regexp.MustCompile(`^\[\^([^\]]+)\]:`)
	footnoteMark       = regexp.MustCompile(`\[\^[^\]]+\]`)
	imageLink          = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]*)\)`)
	relativeLink       = regexp.MustCompile(`\[([^\]]*)\]\(([^)]*)\)`)
	houseNumber        = regexp.MustCompile(`^(#{1,6} )[0-9]+(?:\.[0-9]+)*-(\S)`)
	tableSeparator     = regexp.MustCompile(`^[-:]+(\|[-:]+)+$`)
	taskBox            = regexp.MustCompile(`^([-*+] )\[[ xX]\] `)
	blockStarter       = regexp.MustCompile(`^(#{1,6} |[-*+] |[0-9]+[.)] |> |\||` + "```" + `|---$|\*\*\*$)`)
	// The pencil's two shapes, as the author writes them: a comment with words
	// of the sentence in front of it, and a comment opening a line.
	inlinePencil = regexp.MustCompile(`(?m)^.*\S.*<!--.*-->`)
	blockPencil  = regexp.MustCompile(`(?m)^<!--`)
)

// The pencil's delimiters, as the author types them and as the export writes
// them back. internal/house states them as the mark's syntax and
// internal/export writes them; they are literals here for the reason every
// house-style test states its value as a literal.
const (
	commentOpener = "<!--"
	commentCloser = "-->"
)

// knownDifferences is the list. alt is the alt text of every picture the note
// links, because the caption the generator writes under a figure is a
// paragraph of the document and those are the words it holds, and notes is the
// note's own footnote labels numbered the way the document numbered them.
func knownDifferences(alt map[string]bool, notes footnoteNumbering) []knownDifference {
	return []knownDifference{{
		Name:   "front matter and the prelude",
		Reason: "the note's own keys become the cover, the three house tables and the contents list, and the export strips every one of them as the prelude",
		Gone:   func(l noteLine) bool { return l.FrontMatter },
	}, {
		Name:   "code blocks",
		Reason: "the generator writes a fenced block as paragraphs in the code style, so no character of the document says it was fenced and nothing can put the fence back",
		Gone:   func(l noteLine) bool { return l.Code },
	}, {
		Name:   "page breaks the note wrote",
		Reason: "a thematic break in the note is a page break in the document since M19, and the export writes a break of its own back as `---`, so both sides spell it the same way and no line needs excusing; a note's `***` or `___` is that same break and comes home as `---`, and a break the build left out, at the end of a note or before its footnote definitions, is a line the document never held",
		Note:   thematicBreak,
	}, {
		Name:   "footnotes",
		Reason: "the numbers are the document's: a label the note wrote is the number Word gave the mark it carried, read writes every note under a --- rule at the end whatever the note did with it, and a label the note marked twice is two notes under the page, so the second definition is a line the note never had",
		Note:   notes.numbered,
		Added:  func(s string) bool { return s == "---" || notes.repeated(s) },
	}, {
		Name:   "pictures",
		Reason: "the bytes come back from the docx export as a new file under assets/, so the file the export links is not the file the note links, and the caption the generator wrote under a figure is a paragraph of its own",
		Gone:   func(l noteLine) bool { return imageLink.MatchString(l.Text) },
		Added: func(s string) bool {
			return imageLink.MatchString(s) || strings.HasPrefix(s, "<!-- picture:") ||
				s == "[image]" || s == "[drawing]" || alt[s]
		},
	}, {
		Name:   "the house's own breaks",
		Reason: "the house layout breaks the page after the cover and before the contents list, both inside the prelude the strip takes out, and before the first body heading, which Drive rides inside the heading paragraph and the export leaves out because build puts it back; a column break is the layout's too, and a page break still marked is one inside a sentence that no line of markdown could carry",
		Added:  func(s string) bool { return s == "[page break]" || s == "[column break]" },
	}, {
		Name:   "heading numbers",
		Reason: "house.yaml numbers a heading with literal text, 1-Scope, and the export takes the number back off; a number reaching this comparison is one the export did not recognise",
		Export: func(s string) string { return houseNumber.ReplaceAllString(s, "$1$2") },
	}, {
		Name:   "escaping",
		Reason: "read escapes the markup a document happens to contain, so the file reads back as the same words, and the note wrote the character plainly",
		Export: func(s string) string { return unescapeMarkup(s) },
	}, {
		Name:   "table separators",
		Reason: "the note's separator row is written for a person and read prints one of its own, and neither is a row of the table",
		Note:   func(s string) string { return canonicalSeparator(s) },
		Export: func(s string) string { return canonicalSeparator(s) },
	}, {
		Name:   "typography",
		Reason: "the generator's smart punctuation is goldmark's typographer, so the straight quotes, the two kinds of dash and the three dots are single characters in the document. A pencil's words are not its business: goldmark hands a comment over as one piece of raw HTML, so the typographer never sees inside it and both sides hold what the author typed",
		Note:   func(s string) string { return outsideComments(s, typography) },
	}, {
		Name:   "inline markup",
		Reason: "bold, italic, code and strikeout are text styles in the document rather than characters, so the projection prints the words and the delimiters are not there to print. The highlight is not in that list any more: a mark is a colour the export recognises and writes back, so both sides spell it the same way and a difference in one is a difference",
		Note:   func(s string) string { return outsideComments(s, stripDelimiters) },
	}, {
		Name:   "relative links as words",
		Reason: "a link to a file in the hub means nothing in Drive, so the generator writes the words and not the address",
		Note: func(s string) string {
			return outsideComments(s, func(out string) string {
				return relativeLink.ReplaceAllStringFunc(out, dropRelative)
			})
		},
	}, {
		Name:   "blockquotes",
		Reason: "the generator writes a quote as a paragraph in the quote style, so no character of the document says it is quoted",
		Note:   func(s string) string { return strings.TrimPrefix(strings.TrimPrefix(s, "> "), ">") },
	}, {
		Name:   "task lists",
		Reason: "a task box is a checkbox character in the note and a plain bullet in the house style",
		Note:   func(s string) string { return taskBox.ReplaceAllString(s, "$1") },
	}}
}

// outsideComments reads one of the note's lines through a rule, and hands the
// rule the words around each comment rather than the whole line. A comment is
// left as it stands, delimiters and all.
//
// Three of the note-side rules rewrite the middle of a line: the smart
// punctuation, the emphasis delimiters and the link to a file in the hub. None
// of the three happened inside a pencil, because goldmark hands the comment
// over as one piece of raw HTML and neither the typographer nor the emphasis
// parsers see inside it, so the document holds the characters the author typed
// and so does the file the export wrote. A rule that retyped them would turn a
// line both sides spell the same way into a difference no name covers.
//
// An unclosed `<!--` is left alone with the rest of the line, which is the one
// shape the build refuses to carry anyway.
func outsideComments(s string, rule func(string) string) string {
	var b strings.Builder
	for {
		open := strings.Index(s, commentOpener)
		if open < 0 {
			break
		}
		shut := strings.Index(s[open:], commentCloser)
		if shut < 0 {
			break
		}
		shut += open + len(commentCloser)
		b.WriteString(rule(s[:open]))
		b.WriteString(s[open:shut])
		s = s[shut:]
	}
	b.WriteString(rule(s))
	return b.String()
}

// thematicBreak is a note's thematic break in the one spelling the export
// writes. goldmark reads `---`, `***` and `___` as one node, so all three are
// the same page break in the document, and the export has one line for it.
func thematicBreak(s string) string {
	if s == "***" || s == "___" {
		return "---"
	}
	return s
}

// footnoteNumbering is the note's own footnote labels read as the numbers the
// document gives them. It cannot be a per-line function, because a line does
// not know how many marks stood before it in the note, and the number a mark
// takes is its place in the whole note's reading order. So the whole note is
// read once and the two answers the footnotes difference needs are written
// down: how a line of the note has to be read, and which definition lines of
// the file the note never had.
//
// One shape is out of its reach, and no fixture holds it: a mark on an
// indented continuation line of another definition. internal/body warns about
// such a mark and leaves it out, so the document has no note for it, and this
// reading would number it.
type footnoteNumbering struct {
	// lines is a line of the note as the comparison reads it, mapped to the
	// same line with every label replaced by its number.
	lines map[string]string
	// extra is every number that belongs to a second or later mark of one
	// label. The note wrote one definition and the document holds a note per
	// mark, so read writes a definition line for each of them.
	extra map[string]bool
}

// numbered is the Note rule: the line as the document holds it, or the line
// itself where no mark stood in it.
func (f footnoteNumbering) numbered(s string) string {
	if out, ok := f.lines[s]; ok {
		return out
	}
	return s
}

// repeated is the Added rule: a definition line of the file whose number is a
// second mark of a label the note defined once.
func (f footnoteNumbering) repeated(s string) bool {
	m := footnoteDefinition.FindStringSubmatch(s)
	return m != nil && f.extra[m[1]]
}

// numberFootnotes reads the note once: every mark in the prose takes the next
// number, in the order the marks appear, which is the order Word numbers them
// in and the order read prints them back in. A definition then takes the first
// number its own label was given, and the rest of that label's numbers are
// definition lines the file holds and the note never wrote.
func numberFootnotes(src []byte) footnoteNumbering {
	out := footnoteNumbering{lines: map[string]string{}, extra: map[string]bool{}}
	numbers := map[string][]string{}
	var definitions []string
	n := 0

	for _, l := range noteLines(src) {
		if l.FrontMatter || l.Code {
			continue
		}
		if footnoteDefinition.MatchString(l.Text) {
			definitions = append(definitions, l.Text)
			continue
		}
		if !footnoteMark.MatchString(l.Text) {
			continue
		}
		line := footnoteMark.ReplaceAllStringFunc(l.Text, func(mark string) string {
			label := mark[2 : len(mark)-1]
			n++
			at := strconv.Itoa(n)
			numbers[label] = append(numbers[label], at)
			return "[^" + at + "]"
		})
		if _, seen := out.lines[l.Text]; !seen {
			out.lines[l.Text] = line
		}
	}

	for _, text := range definitions {
		label := footnoteDefinition.FindStringSubmatch(text)[1]
		took := numbers[label]
		if len(took) == 0 {
			// internal/body refuses a definition nothing marks, so the note
			// this runs over cannot hold one. Left alone rather than guessed
			// at, so a note that somehow holds one shows up as a difference.
			continue
		}
		if _, seen := out.lines[text]; !seen {
			out.lines[text] = footnoteDefinition.ReplaceAllString(text, "[^"+took[0]+"]:")
		}
		for _, at := range took[1:] {
			out.extra[at] = true
		}
	}
	return out
}

// difference is one line the comparison could not match, with the side it came
// from, and named is one entry of the list with how many lines it covered.
type (
	difference struct {
		side string
		text string
	}
	namedCount struct {
		name   string
		reason string
		count  int
	}
	roundTripReport struct {
		named   []namedCount
		unnamed []difference
	}
)

// roundTrip compares one note with the body of the file the export wrote for
// the document that note was published into.
func roundTrip(note []byte, body string) roundTripReport {
	alt := map[string]bool{}
	for _, m := range imageLink.FindAllStringSubmatch(string(note), -1) {
		if s := normaliseLine(m[1]); s != "" {
			alt[s] = true
		}
	}
	list := knownDifferences(alt, numberFootnotes(note))
	counts := map[string]int{}

	// The file's lines first, each read back through every Export rule, so the
	// note is compared against what the document holds rather than against how
	// read spells it.
	have := map[string]int{}
	var order []string
	for _, raw := range strings.Split(body, "\n") {
		s := normaliseLine(raw)
		if s == "" {
			continue
		}
		for _, k := range list {
			if k.Export == nil {
				continue
			}
			if out := k.Export(s); out != s {
				counts[k.Name]++
				s = out
			}
		}
		if s == "" {
			continue
		}
		if have[s] == 0 {
			order = append(order, s)
		}
		have[s]++
	}

	var unnamed []difference
	for _, l := range noteLines(note) {
		if name, ok := goneBy(list, l); ok {
			counts[name]++
			continue
		}
		if matchLine(list, l.Text, have, counts) {
			continue
		}
		unnamed = append(unnamed, difference{side: "in the note and in no line of the file", text: l.Text})
	}

	for _, s := range order {
		for ; have[s] > 0; have[s]-- {
			if name, ok := addedBy(list, s); ok {
				counts[name]++
				continue
			}
			unnamed = append(unnamed, difference{side: "in the file and in no line of the note", text: s})
		}
	}
	return roundTripReport{named: report(list, counts), unnamed: unnamed}
}

// matchLine takes one note line through the Note rules in order, stopping at
// the first reading the file holds. The names that changed the line up to that
// point are the reasons it differs, and they are counted there rather than
// speculatively: a rule that fired after the match was already found would be
// naming a difference that was not in the way.
func matchLine(list []knownDifference, text string, have map[string]int, counts map[string]int) bool {
	if have[text] > 0 {
		have[text]--
		return true
	}
	var used []string
	s := text
	for _, k := range list {
		if k.Note == nil {
			continue
		}
		out := normaliseLine(k.Note(s))
		if out == s {
			continue
		}
		used = append(used, k.Name)
		s = out
		if s == "" || have[s] > 0 {
			if s != "" {
				have[s]--
			}
			for _, n := range used {
				counts[n]++
			}
			return true
		}
	}
	return false
}

// goneBy and addedBy are the two one-sided rules, asked in list order.
func goneBy(list []knownDifference, l noteLine) (string, bool) {
	for _, k := range list {
		if k.Gone != nil && k.Gone(l) {
			return k.Name, true
		}
	}
	return "", false
}

func addedBy(list []knownDifference, s string) (string, bool) {
	for _, k := range list {
		if k.Added != nil && k.Added(s) {
			return k.Name, true
		}
	}
	return "", false
}

// report is the names that fired, in the list's own order, so two runs read the
// same way.
func report(list []knownDifference, counts map[string]int) []namedCount {
	var out []namedCount
	for _, k := range list {
		if counts[k.Name] > 0 {
			out = append(out, namedCount{name: k.Name, reason: k.Reason, count: counts[k.Name]})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].count > out[j].count })
	return out
}

// noteLines is the note as the comparison reads it: the front matter marked,
// fenced blocks marked, and every soft-wrapped paragraph joined back into the
// one line the document holds for it.
func noteLines(src []byte) []noteLine {
	var out []noteLine
	var buf string
	flush := func() {
		if s := normaliseLine(buf); s != "" {
			out = append(out, noteLine{Text: s})
		}
		buf = ""
	}
	front, fence := false, false
	for i, raw := range strings.Split(string(src), "\n") {
		line := strings.TrimSpace(raw)
		if i == 0 && line == "---" {
			front = true
			continue
		}
		if front {
			if line == "---" {
				front = false
				continue
			}
			out = append(out, noteLine{Text: line, FrontMatter: true})
			continue
		}
		if strings.HasPrefix(line, "```") {
			flush()
			fence = !fence
			out = append(out, noteLine{Text: line, Code: true})
			continue
		}
		if fence {
			out = append(out, noteLine{Text: raw, Code: true})
			continue
		}
		if line == "" {
			flush()
			continue
		}
		if blockStarter.MatchString(line) {
			flush()
			buf = line
			continue
		}
		if buf == "" {
			buf = line
			continue
		}
		buf += " " + line
	}
	flush()
	return out
}

// normaliseLine is what neither side is asked to account for: the space around
// a line, a run of spaces inside it, and how a pipe table is written. A note
// pads its cells and fences the row with an outer pipe, and read writes the
// cells joined by one pipe and no outer one, so both come down to the cells
// with a pipe between them. None of it is a difference in anything.
func normaliseLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, "| ", "|")
	s = strings.ReplaceAll(s, " |", "|")
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "|") || strings.HasSuffix(s, "|") {
		s = strings.Trim(s, "|")
	}
	return s
}

// canonicalSeparator reduces a pipe table's separator row to one shape, so the
// note's alignment colons and read's own dashes are the same row.
func canonicalSeparator(s string) string {
	if !tableSeparator.MatchString(s) {
		return s
	}
	return strings.TrimSuffix(strings.Repeat("-|", strings.Count(s, "|")+1), "|")
}

// typography is the ten substitutions internal/body asks goldmark's typographer
// for. The characters are spelled as escapes so the list reads as the code
// points it is rather than as ten shapes a font decides.
func typography(s string) string {
	for _, pair := range [][2]string{
		{"---", "—"}, {"--", "–"}, {"...", "…"},
		{"<<", "«"}, {">>", "»"},
	} {
		s = strings.ReplaceAll(s, pair[0], pair[1])
	}
	return quotes(s)
}

// quotes turns the straight quotes into the curly ones, opening where the
// character before is a space or nothing and closing everywhere else, which is
// the rule the typographer follows.
func quotes(s string) string {
	var b strings.Builder
	prev := ' '
	for _, r := range s {
		open := prev == ' ' || prev == '(' || prev == '[' || prev == 0
		switch {
		case r == '"' && open:
			b.WriteRune('“')
		case r == '"':
			b.WriteRune('”')
		case r == '\'' && open:
			b.WriteRune('‘')
		case r == '\'':
			b.WriteRune('’')
		default:
			b.WriteRune(r)
		}
		prev = r
	}
	return b.String()
}

// stripDelimiters takes the emphasis characters off, longest first, so ** is
// never read as two single stars.
//
// The highlight's == is not taken off. It was until M18: the colour reached
// the document and nothing brought it home, so the note's delimiters had to go
// before the two sides could be compared. From M18 the export writes the mark
// syntax back, so the note's ==🟢 a claim== is the file's ==🟢 a claim==, and
// taking the == off one side would hide a mark that came home as the wrong
// colour or did not come home at all.
func stripDelimiters(s string) string {
	for _, d := range []string{"***", "**", "~~", "*", "_", "`"} {
		s = strings.ReplaceAll(s, d, "")
	}
	return s
}

// dropRelative writes a link that points at a file in the hub as its own words,
// and leaves a link with a scheme alone: the document carries that one and read
// prints it back.
func dropRelative(link string) string {
	m := relativeLink.FindStringSubmatch(link)
	if m == nil {
		return link
	}
	target := m[2]
	if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "#") {
		return link
	}
	return m[1]
}

// unescapeMarkup is read's escaping read backwards: a backslash in front of a
// character read would otherwise have written as markup.
func unescapeMarkup(s string) string {
	var b strings.Builder
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		if rs[i] == '\\' && i+1 < len(rs) && strings.ContainsRune("\\`*_{}[]()#+-.!|<>~=", rs[i+1]) {
			i++
		}
		b.WriteRune(rs[i])
	}
	return b.String()
}

// TestTheComparisonLeavesAPencilsWordsAsTyped runs in `make test`, with
// neither variable set.
//
// A pencil's words reach the document as the author typed them: goldmark hands
// the comment over as one piece of raw HTML, so the typographer and the
// emphasis parsers never see inside it, and internal/body's
// TestTheWordsInsideAPencilAreLiteral says so from the build's side. The
// reading above has to leave them alone for the same reason. It did not when
// M20 first measured a live round trip: the smart punctuation rule read the
// `--` of `<!--` as an en dash, so a pencil on a line that also carried a
// quote or a bold word came home spelled the same way on both sides and was
// still reported as a difference no name covered.
func TestTheComparisonLeavesAPencilsWordsAsTyped(t *testing.T) {
	note := "The author's aside <!-- don't -- **retype** [this](./other.md) -->, and *stress* on the words around it."
	file := "The author’s aside <!-- don't -- **retype** [this](./other.md) -->, and stress on the words around it."

	list := knownDifferences(map[string]bool{}, numberFootnotes(nil))
	have := map[string]int{normaliseLine(file): 1}
	counts := map[string]int{}
	if !matchLine(list, normaliseLine(note), have, counts) {
		t.Errorf("the note's line reaches no line of the file, and the only difference between them is in the words around the comment:\n  note %q\n  file %q",
			note, file)
	}
}

// TestTheRoundTripNoteCarriesAFootnoteAndEveryMark runs in `make test`, with
// neither variable set.
//
// The round trip above is the one test that measures a footnote and a mark
// over four translations, and it measures them only if one of the six notes it
// publishes holds them. The kitchen sink does. A note somebody edits back to
// the shapes of M17 would otherwise leave the live run green and the question
// unasked, so the seven mark openers and the footnote are pinned here, each as
// the literal an author types.
//
// The pencil is pinned twice, because since M20 it has two shapes rather than
// one spelling: a comment inside a sentence is a grey run of that sentence,
// and a comment on lines of its own is a grey paragraph. They travel home by
// different routes, so a note holding only the first would leave the second
// measured by no live run at all.
func TestTheRoundTripNoteCarriesAFootnoteAndEveryMark(t *testing.T) {
	named := false
	for _, name := range roundTripNotes {
		if name == kitchenSinkNote {
			named = true
		}
	}
	if !named {
		t.Fatalf("%s is not one of the notes the round trip publishes: %v", kitchenSinkNote, roundTripNotes)
	}

	src, err := os.ReadFile(filepath.Join(bodyNotes, kitchenSinkNote))
	if err != nil {
		t.Fatalf("the note %s could not be read: %v", kitchenSinkNote, err)
	}
	note := string(src)

	for _, c := range []struct{ syntax, mark string }{
		{"==", "yellow"},
		{"🟢", "green"},
		{"🟣", "purple"},
		{"🔵", "blue"},
		{"🔴", "red"},
		{"🟠", "orange"},
		{"<!--", "pencil"},
	} {
		if !strings.Contains(note, c.syntax) {
			t.Errorf("%s holds no %q, so the round trip measures no %s mark", kitchenSinkNote, c.syntax, c.mark)
		}
	}
	if !inlinePencil.MatchString(note) {
		t.Errorf("%s holds no comment inside a sentence, so the round trip measures no pencil run", kitchenSinkNote)
	}
	if !blockPencil.MatchString(note) {
		t.Errorf("%s holds no comment on a line of its own, so the round trip measures no grey paragraph", kitchenSinkNote)
	}
	if !footnoteDefinition.MatchString(note) && !strings.Contains(note, "\n[^") {
		t.Errorf("%s defines no footnote, so the round trip measures none", kitchenSinkNote)
	}
	if n := len(footnoteMark.FindAllString(note, -1)); n == 0 {
		t.Errorf("%s marks no footnote in its prose", kitchenSinkNote)
	}
}
