// The marks on the way home: the syntax export writes for a colour the house
// claims, the words it writes for a colour nobody claims, and the read, which
// writes neither.
//
// Every hex, emoji and delimiter here is a literal. A test that read the
// palette out of house.yaml would follow the file wherever somebody moved it
// and agree with a document gdoc never wrote.

package export

import (
	"strings"
	"testing"

	"gdoc/internal/docs"
	"gdoc/internal/house"
	"gdoc/internal/view"
)

// palette is the house marks as the projection writes them. It goes through
// Palette rather than a list of its own, because what this file is testing is
// that the seven in house.yaml reach a file as Obsidian's syntax.
func palette(t *testing.T) []view.Mark {
	t.Helper()
	cfg, err := house.Load()
	if err != nil {
		t.Fatalf("house.Load: %v", err)
	}
	return Palette(cfg)
}

// marked is the marks fixture projected with the palette passed, which is what
// export does, and the body of its one file.
func marked(t *testing.T) (string, map[string]int) {
	t.Helper()
	files, _, other, _ := Project(fixture(t, "marks.json"), nil, palette(t))
	if len(files) != 1 {
		t.Fatalf("marks.json gave %d files, want 1", len(files))
	}
	return files[0].Body, other
}

// The six highlights come home as Obsidian's ==, with the colour emoji at the
// start of the words and the one space internal/body takes off again, and the
// pencil comes home as HTML's comment. Yellow is the plain highlight and
// carries no emoji: an author writing ==words== gets ==words== back.
func TestExportWritesObsidiansSyntaxForEachPaletteColour(t *testing.T) {
	body, _ := marked(t)
	want := "==🔴 red== ==🟠 orange== ==yellow== ==🟢 green== ==🔵 blue== ==🟣 purple== <!-- pencil -->"
	if !strings.Contains(body, want) {
		t.Errorf("the seven marks came home as\n%s\nwant the line\n%s", body, want)
	}
}

// A colour the house claims no mark for is the words as they stand, because
// gdoc did not write it and has no syntax that means it. It is counted by hex
// so the session reading the file can say how much of the document it is
// bringing home as plain prose, and adjacent runs of one colour are one count:
// Docs splits a run wherever the formatting changes, and three counts for one
// highlighted phrase would say the document holds three.
//
// #FFFF00 is Word's named yellow, which the cover's own placeholders carry and
// which the mark probe measured as a different colour from the mark yellow
// #FFD500. Telling the two apart is what this count rests on.
func TestAColourOutsideThePaletteIsPlainWordsAndCounted(t *testing.T) {
	body, other := marked(t)

	if !strings.Contains(body, "stale words and more.") {
		t.Errorf("the unclaimed colour did not come home as plain words:\n%s", body)
	}
	if got := other["#FFFF00"]; got != 2 {
		t.Errorf("other[#FFFF00] = %d, want 2", got)
	}
	if len(other) != 1 {
		t.Errorf("other = %v, want #FFFF00 alone", other)
	}
}

// Docs splits a run wherever the formatting changes, so a mark over a phrase
// that holds bold or code arrives as several runs sharing one background. They
// are one mark in the document and one mark in the file: a pair of == around
// each run would say the author marked the words twice.
func TestAMarkSplitOverSeveralRunsComesHomeWhole(t *testing.T) {
	body, _ := marked(t)
	if !strings.Contains(body, "==🟢 a claim to confirm==") {
		t.Errorf("the split green mark came home as\n%s", body)
	}
	if strings.Contains(body, "==🟢 a claim== ==🟢 to confirm==") {
		t.Errorf("the split green mark came home as two marks:\n%s", body)
	}
}

// read is unchanged. The words of a marked phrase reach a reviewing session
// and the mark around them does not, because a session quotes what it reads
// and a quote carrying == is not text the document holds.
func TestReadPrintsNoMark(t *testing.T) {
	text, _ := view.Text(fixture(t, "marks.json"))
	want := "The figure and a claim to confirm, pencilled, stale words and more.\n\n" +
		"red orange yellow green blue purple pencil\n"
	if text != want {
		t.Errorf("read printed\n%q\nwant\n%q", text, want)
	}
}

// oneYellow is a document of one paragraph whose whole text is one run marked
// with the yellow mark's hex, which is the mark an author may write bare.
func oneYellow(text string) *docs.Document {
	return &docs.Document{
		ID: "1yellow", Title: "One yellow run",
		Tabs: []docs.Tab{{ID: "t.0", Body: []docs.Block{{Paragraph: &docs.Paragraph{
			Style: "NORMAL_TEXT",
			Runs: []docs.Run{{Kind: docs.KindText, Text: text + "\n",
				StartIndex: 1, EndIndex: 1 + len([]rune(text)) + 1,
				Background: "#FFD500"}},
		}}}}},
	}
}

// A highlight's delimiter is read by flanking: "==" with a space before it
// does not close a highlight and "==" with a space after it does not open one.
// A run in Docs can begin or end with a space, which is what an edit leaves
// behind, so the space is written outside the delimiters. Written inside them,
// the next build printed "==" into the document and the debt stopped being
// counted.
func TestAMarkedRunsOwnSpacesAreWrittenOutsideTheDelimiters(t *testing.T) {
	files, _, _, _ := Project(oneYellow(" to fill in "), nil, palette(t))

	if want := " ==to fill in== "; !strings.Contains(files[0].Body, want) {
		t.Errorf("the marked run came home as\n%q\nwant %q", files[0].Body, want)
	}
}

// The flanking test is goldmark's, and goldmark reads whitespace as
// unicode.IsSpace: a soft line break, which is what Docs writes for
// Shift+Enter and which it keeps inside the run, and a non-breaking space are
// both whitespace there. A narrower set wrote "==to fill in\v==", whose
// closing delimiter is not right-flanking, so the mark came home as two pairs
// of equals signs in the words.
func TestAMarksOwnSoftBreakAndNonBreakingSpaceAreWrittenOutsideTheDelimiters(t *testing.T) {
	for _, c := range []struct{ name, text, want string }{
		{"a soft line break", "to fill in\v", "==to fill in==\v"},
		{"a non-breaking space", " to fill in", " ==to fill in=="},
	} {
		files, _, _, _ := Project(oneYellow(c.text), nil, palette(t))

		if !strings.Contains(files[0].Body, c.want) {
			t.Errorf("%s: the marked run came home as\n%q\nwant %q", c.name, files[0].Body, c.want)
		}
	}
}

// internal/body reads the first rune of a highlight as its colour, so words of
// the author's own opening with a colour emoji cannot be written after the
// bare delimiter: the mark would come home a different colour with the emoji
// dropped from the words. The colour is written out instead.
func TestAYellowMarkWhoseWordsOpenWithAnEmojiIsWrittenWithItsColour(t *testing.T) {
	files, _, _, _ := Project(oneYellow("🟢 go"), nil, palette(t))

	if want := "==🟡 🟢 go=="; !strings.Contains(files[0].Body, want) {
		t.Errorf("the marked run came home as\n%q\nwant %q", files[0].Body, want)
	}
}

// oneMarked is a document of one paragraph whose whole text is the run passed,
// so a test can mark any words with any colour and read back what the file
// holds.
func oneMarked(r docs.Run) *docs.Document {
	r.Kind = docs.KindText
	r.StartIndex = 1
	r.EndIndex = 1 + len([]rune(r.Text))
	return &docs.Document{
		ID: "1mark", Title: "One marked run",
		Tabs: []docs.Tab{{ID: "t.0", Body: []docs.Block{{Paragraph: &docs.Paragraph{
			Style: "NORMAL_TEXT",
			Runs: []docs.Run{r,
				{Kind: docs.KindText, Text: "\n", StartIndex: r.EndIndex, EndIndex: r.EndIndex + 1}},
		}}}}},
	}
}

// A mark whose words cannot carry its syntax comes home as the words alone,
// with the loss named. The next build reads the syntax back, and in each of
// these shapes it cannot: a highlight's delimiter inside the words closes it
// early or lengthens it, and a comment's closer inside them shuts the comment
// and leaves the words after it as prose.
//
// Written anyway, the first two brought the document's own "==" home as the
// author's syntax, and the third would put half a grey aside into the file. A
// backslash is no answer: internal/body never unescapes a punctuation mark, so
// one written here reaches the document as a backslash.
//
// A bracket, a caret and a link are not on this list any more: they were the
// span's refusals, and a comment holds all three as the characters they are.
// TestAPencilRunComesHomeAsAnHTMLComment and
// TestFormattingOnAPencilIsDroppedAndNamed are where those words go now.
func TestAMarkWhoseWordsCannotCarryItsSyntaxComesHomeAsWordsAndIsNamed(t *testing.T) {
	for _, c := range []struct {
		name, want string
		run        docs.Run
	}{
		{"a highlight holding the delimiter", "a==b",
			docs.Run{Text: "a==b", Background: "#FFD500"}},
		{"a highlight ending in one half of it", "a=",
			docs.Run{Text: "a=", Background: "#FFD500"}},
		{"a pencil holding the comment's closer", "a--> b",
			docs.Run{Text: "a--> b", Colour: "#9A9A9A"}},
	} {
		files, _, _, warnings := Project(oneMarked(c.run), nil, palette(t))

		if got := files[0].Body; !strings.Contains(got, c.want+"\n") {
			t.Errorf("%s came home as\n%q\nwant the words %q alone", c.name, got, c.want)
		}
		if strings.Contains(files[0].Body, "<!--") || strings.Contains(files[0].Body, "==a") {
			t.Errorf("%s came home carrying a mark the next build cannot read:\n%q", c.name, files[0].Body)
		}
		if len(warnings) != 1 {
			t.Errorf("%s raised %v, want the one sentence naming the loss", c.name, warnings)
		}
	}
}

// The cover's own placeholders are Word's named yellow, #FFFF00, which a
// published document carries on every field the author has not filled in yet.
// They are the prelude: they leave the file with the rest of it, and counting
// them would say a document gdoc wrote came home with marks nobody made. The
// count rests on the caller's skip, so this asks a document whose prelude the
// layout recognised, with a placeholder carrying that yellow.
func TestThePreludesOwnYellowIsNotCounted(t *testing.T) {
	files, pieces, other, _ := Project(fixture(t, "publish-prelude.json"), nil, palette(t))

	if len(pieces) == 0 {
		t.Fatal("no prelude was stripped, so this document does not ask the question")
	}
	// The same walk with no skip finds it, so this cannot pass on a fixture
	// that holds no yellow at all.
	bare := map[string]int{}
	otherColours(fixture(t, "publish-prelude.json").Tabs[0].Body, nil, palette(t), bare)
	if bare["#FFFF00"] == 0 {
		t.Fatalf("the prelude carries no #FFFF00, so this test asks nothing: %v", bare)
	}
	if len(other) != 0 {
		t.Errorf("other = %v, want nothing: the only colour is the cover's own", other)
	}
	if strings.Contains(files[0].Body, "The Board") {
		t.Errorf("the placeholder is still in the file, so it was never the prelude:\n%s", files[0].Body)
	}
}

// A marked run holding no words of its own is written as it stands. A space
// an author highlighted on its own, or the highlight an edit left on the
// trailing space a paragraph's newline shares, has nothing for a pair of
// delimiters to sit around, and "== ==" would not read as a mark anyway.
//
// It used to be the group the walk could not take the whitespace off: the
// guard counted the space as a word and the cut took it away, leaving the cut
// nothing to index, and the export of a document anybody could write ended as
// a panic recovered into a failed envelope.
func TestAMarkedRunOfOnlyWhitespaceComesHomeAsItStands(t *testing.T) {
	// The whole paragraph is one highlighted space, so the run holds the
	// space and the paragraph's own newline and nothing else.
	files, _, _, _ := Project(oneYellow(" "), nil, palette(t))

	if got := files[0].Body; !strings.Contains(got, " \n") || strings.Contains(got, "==") {
		t.Errorf("the highlighted space came home as\n%q\nwant the space with no delimiters", got)
	}
}

// The same group between words, which is the shape an edit inside a highlight
// leaves behind: the space carries the mark and the words on either side do
// not.
func TestAMarkedSpaceBetweenPlainWordsComesHomeAsItStands(t *testing.T) {
	d := &docs.Document{
		ID: "1space", Title: "A marked space",
		Tabs: []docs.Tab{{ID: "t.0", Body: []docs.Block{{Paragraph: &docs.Paragraph{
			Style: "NORMAL_TEXT",
			Runs: []docs.Run{
				{Kind: docs.KindText, Text: "Hello", StartIndex: 1, EndIndex: 6},
				{Kind: docs.KindText, Text: " ", StartIndex: 6, EndIndex: 7, Background: "#FFD500"},
				{Kind: docs.KindText, Text: "world\n", StartIndex: 7, EndIndex: 13},
			},
		}}}}},
	}

	files, _, _, _ := Project(d, nil, palette(t))

	if want := "Hello world"; !strings.Contains(files[0].Body, want) {
		t.Errorf("the marked space came home as\n%q\nwant %q", files[0].Body, want)
	}
}

// TestAMarkWhoseWordsHoldAnotherMarksSyntaxComesHomeAsWords: a mark written as
// a delimiter pair is checked against every mark in the palette and not
// against its own syntax alone.
//
// A highlight's words holding "<!--" came home as "==a <!-- b --> c==", and
// the next build reads the comment inside the highlight as a mark inside a
// mark and refuses it by line. The colour is dropped and the loss is named.
//
// Dropping the colour is not enough to make that file build: the words still
// hold the opener, and internal/body reads a comment whether or not a mark was
// written around it. So the case carries a second sentence, ownSyntax's,
// naming the words the next build misreads. The two say different things, and
// also is which case expects which.
//
// The pencil is not on this list. A highlight inside a comment is words, not a
// mark inside a mark: TestAPencilWhoseWordsHoldAHighlightComesHomeInTheComment
// is that half.
func TestAMarkWhoseWordsHoldAnotherMarksSyntaxComesHomeAsWords(t *testing.T) {
	for _, c := range []struct {
		name, words, never, also string
		run                      docs.Run
	}{
		{"a highlight whose words hold a comment", "a <!-- b --> c", "==", "<!--",
			docs.Run{Text: "a <!-- b --> c", Background: "#FFD500"}},
	} {
		files, _, _, warnings := Project(oneMarked(c.run), nil, palette(t))

		if got := files[0].Body; !strings.Contains(got, c.words+"\n") {
			t.Errorf("%s came home as\n%q\nwant the words %q alone", c.name, got, c.words)
		}
		if strings.Contains(files[0].Body, c.never) {
			t.Errorf("%s came home carrying %q, which the next build cannot read:\n%q",
				c.name, c.never, files[0].Body)
		}
		want := 1
		if c.also != "" {
			want = 2
		}
		if len(warnings) != want {
			t.Errorf("%s raised %v, want %d sentences naming the loss", c.name, warnings, want)
		}
		dropped := false
		for _, w := range warnings {
			if strings.Contains(w, "is printed as the words alone") {
				dropped = true
			}
		}
		if !dropped {
			t.Errorf("%s raised %v, and none of it names the colour that was dropped", c.name, warnings)
		}
		if c.also == "" {
			continue
		}
		named := false
		for _, w := range warnings {
			if strings.Contains(w, c.also) && strings.Contains(w, "the next build reads") {
				named = true
			}
		}
		if !named {
			t.Errorf("%s raised %v, and none of it names %q as syntax the next build reads",
				c.name, warnings, c.also)
		}
	}
}

// pencilled is a document of one paragraph whose middle run is grey and whose
// words either side are plain, which is a pencil inside a sentence.
func pencilled(before, words, after string) *docs.Document {
	at := 1 + len([]rune(before))
	end := at + len([]rune(words))
	return &docs.Document{
		ID: "1aside", Title: "A pencilled aside",
		Tabs: []docs.Tab{{ID: "t.0", Body: []docs.Block{{Paragraph: &docs.Paragraph{
			Style: "NORMAL_TEXT",
			Runs: []docs.Run{
				{Kind: docs.KindText, Text: before, StartIndex: 1, EndIndex: at},
				{Kind: docs.KindText, Text: words, StartIndex: at, EndIndex: end, Colour: "#9A9A9A"},
				{Kind: docs.KindText, Text: after + "\n", StartIndex: end,
					EndIndex: end + len([]rune(after)) + 1},
			},
		}}}}},
	}
}

// aLineOfItsOwn says the body holds this line with nothing else on it.
func aLineOfItsOwn(body, want string) bool {
	for _, line := range strings.Split(body, "\n") {
		if line == want {
			return true
		}
	}
	return false
}

// A pencil comes home as the comment an author writes it as from milestone 20
// on: HTML's "<!-- words -->", which Obsidian dims while editing and hides
// while reading. One space inside each delimiter is the delimiter's, the way
// the space after a highlight's colour emoji is, so a phrase published from a
// note comes home spelled the way it left.
func TestAPencilRunComesHomeAsAnHTMLComment(t *testing.T) {
	files, _, _, warnings := Project(pencilled("a ", "quiet", " b"), nil, palette(t))

	if want := "a <!-- quiet --> b\n"; !strings.Contains(files[0].Body, want) {
		t.Errorf("the pencilled run came home as\n%q\nwant the sentence %q", files[0].Body, want)
	}
	if len(warnings) != 0 {
		t.Errorf("the pencil raised %v, want nothing: a comment carries these words", warnings)
	}
}

// A paragraph whose only run is grey is an aside of its own, and it comes home
// as a comment on a line of its own, which is the shape internal/body reads
// back as a grey paragraph.
func TestAGreyParagraphComesHomeAsACommentOfItsOwn(t *testing.T) {
	files, _, _, warnings := Project(oneMarked(docs.Run{Text: "a quiet aside", Colour: "#9A9A9A"}), nil, palette(t))

	if want := "<!-- a quiet aside -->"; !aLineOfItsOwn(files[0].Body, want) {
		t.Errorf("the grey paragraph came home as\n%q\nwant %q on a line of its own", files[0].Body, want)
	}
	if len(warnings) != 0 {
		t.Errorf("the grey paragraph raised %v, want nothing", warnings)
	}
}

// The one word a pencil cannot hold is its closer: "-->" inside the words
// shuts the comment early, and the grey words after it would reach the file as
// prose. The words go home as the document holds them and the loss is named,
// which is what a mark that cannot carry its syntax has always done.
func TestAPencilWhoseWordsHoldTheCloserIsNamedAsALoss(t *testing.T) {
	files, _, _, warnings := Project(oneMarked(docs.Run{Text: "a --> b", Colour: "#9A9A9A"}), nil, palette(t))

	if want := "a --> b\n"; !strings.Contains(files[0].Body, want) {
		t.Errorf("the words came home as\n%q\nwant %q alone", files[0].Body, want)
	}
	if strings.Contains(files[0].Body, "<!--") {
		t.Errorf("the words came home inside a comment the next build cannot read:\n%q", files[0].Body)
	}
	if len(warnings) != 1 {
		t.Errorf("the pencil raised %v, want the one sentence naming the loss", warnings)
	}
}

// Nothing inside a comment is markdown, so the markup gdoc would otherwise
// write around a pencil's words is dropped and the loss is named. A link is
// the one piece of it the read carries on a run that can also carry the grey:
// written as "[words](url)" inside the comment it would reach the next
// document as those characters, grey, and the address nobody typed with them.
func TestFormattingOnAPencilIsDroppedAndNamed(t *testing.T) {
	files, _, _, warnings := Project(oneMarked(docs.Run{Text: "the grey link", Colour: "#9A9A9A",
		Link: &docs.Link{URL: "https://example.com/x"}}), nil, palette(t))

	if want := "<!-- the grey link -->"; !strings.Contains(files[0].Body, want) {
		t.Errorf("the grey link came home as\n%q\nwant %q", files[0].Body, want)
	}
	if strings.Contains(files[0].Body, "https://example.com/x") {
		t.Errorf("the address came home inside the comment, where it is characters rather than a link:\n%q", files[0].Body)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "link") {
		t.Errorf("the dropped link raised %v, want the one sentence naming it", warnings)
	}
}

// A highlight inside a pencil is words. The next build reads nothing inside a
// comment as markdown, so the equals signs are the characters the document
// shows and there is no mark inside a mark to refuse.
func TestAPencilWhoseWordsHoldAHighlightComesHomeInTheComment(t *testing.T) {
	files, _, _, warnings := Project(oneMarked(docs.Run{Text: "a ==b== c", Colour: "#9A9A9A"}), nil, palette(t))

	if want := "<!-- a ==b== c -->"; !strings.Contains(files[0].Body, want) {
		t.Errorf("the pencil holding a highlight came home as\n%q\nwant %q", files[0].Body, want)
	}
	if len(warnings) != 0 {
		t.Errorf("the pencil raised %v, want nothing: the words are literal", warnings)
	}
}

// Two spellings, one mark, and export writes one of them. Obsidian's "%%" is
// the spelling an author may type and never the spelling a file comes home in:
// a note gdoc wrote holds HTML's comment, which every editor dims the same
// way, and the house style states the one that is written first.
func TestTheObsidianSpellingIsNeverWritten(t *testing.T) {
	html := false
	for _, m := range palette(t) {
		if strings.Contains(m.Open, "%%") || strings.Contains(m.Shut, "%%") {
			t.Errorf("the palette writes a mark as %q ... %q, and Obsidian's spelling never comes home", m.Open, m.Shut)
		}
		if m.Open == "<!-- " && m.Shut == " -->" {
			html = true
		}
	}
	if !html {
		t.Error("no mark in the palette is written as an HTML comment, and the pencil is one")
	}
}
