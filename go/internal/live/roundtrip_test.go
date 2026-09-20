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
	"regexp"
	"sort"
	"strings"
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
	footnoteDefinition = regexp.MustCompile(`^\[\^[^\]]+\]:`)
	imageLink          = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]*)\)`)
	relativeLink       = regexp.MustCompile(`\[([^\]]*)\]\(([^)]*)\)`)
	houseNumber        = regexp.MustCompile(`^(#{1,6} )[0-9]+(?:\.[0-9]+)*-(\S)`)
	tableSeparator     = regexp.MustCompile(`^[-:]+(\|[-:]+)+$`)
	taskBox            = regexp.MustCompile(`^([-*+] )\[[ xX]\] `)
	blockStarter       = regexp.MustCompile(`^(#{1,6} |[-*+] |[0-9]+[.)] |> |\||` + "```" + `|---$|\*\*\*$)`)
)

// knownDifferences is the list, and alt is the alt text of every picture the
// note links, because the caption the generator writes under a figure is a
// paragraph of the document and those are the words it holds.
func knownDifferences(alt map[string]bool) []knownDifference {
	return []knownDifference{{
		Name:   "front matter and the prelude",
		Reason: "the note's own keys become the cover, the three house tables and the contents list, and the export strips every one of them as the prelude",
		Gone:   func(l noteLine) bool { return l.FrontMatter },
	}, {
		Name:   "code blocks",
		Reason: "the generator writes a fenced block as paragraphs in the code style, so no character of the document says it was fenced and nothing can put the fence back",
		Gone:   func(l noteLine) bool { return l.Code },
	}, {
		Name:   "rules",
		Reason: "a horizontal rule is a paragraph border in the document, so read prints it as [rule] and the note's three hyphens are not in it",
		Gone:   func(l noteLine) bool { return l.Text == "---" || l.Text == "***" },
		Added:  func(s string) bool { return s == "[rule]" },
	}, {
		Name:   "footnotes",
		Reason: "read writes every footnote under a --- rule at the end whatever the note did with it, so the definition moves and the rule is a line the note never had",
		Gone:   func(l noteLine) bool { return footnoteDefinition.MatchString(l.Text) },
		Added:  func(s string) bool { return s == "---" || footnoteDefinition.MatchString(s) },
	}, {
		Name:   "pictures",
		Reason: "the bytes come back from the docx export as a new file under assets/, so the file the export links is not the file the note links, and the caption the generator wrote under a figure is a paragraph of its own",
		Gone:   func(l noteLine) bool { return imageLink.MatchString(l.Text) },
		Added: func(s string) bool {
			return imageLink.MatchString(s) || strings.HasPrefix(s, "<!-- picture:") ||
				s == "[image]" || s == "[drawing]" || alt[s]
		},
	}, {
		Name:   "page breaks",
		Reason: "the house layout breaks the page where the template says, and a note holds no character that could",
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
		Reason: "the generator's smart punctuation is goldmark's typographer, so the straight quotes, the two kinds of dash and the three dots are single characters in the document",
		Note:   typography,
	}, {
		Name:   "inline markup",
		Reason: "bold, italic, code, highlight and strikeout are text styles in the document rather than characters, so the projection prints the words and the delimiters are not there to print",
		Note:   stripDelimiters,
	}, {
		Name:   "relative links as words",
		Reason: "a link to a file in the hub means nothing in Drive, so the generator writes the words and not the address",
		Note:   func(s string) string { return relativeLink.ReplaceAllStringFunc(s, dropRelative) },
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
	list := knownDifferences(alt)
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
func stripDelimiters(s string) string {
	for _, d := range []string{"***", "**", "~~", "==", "*", "_", "`"} {
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
