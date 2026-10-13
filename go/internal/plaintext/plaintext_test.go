package plaintext

import (
	"strings"
	"testing"
)

// TestMarkdownNamesWhatDocsWouldRenderLiterally is the rule both writers hold.
// A thread renders what it is given, so each of these arrives as typed and
// somebody strips it by hand.
func TestMarkdownNamesWhatDocsWouldRenderLiterally(t *testing.T) {
	for _, c := range []struct{ body, want string }{
		{"🤖 use **six months**", "**"},
		{"🤖 the `annually` wording", "`"},
		{"🤖 see [the policy](https://example.com)", "[the policy](https://example.com)"},
		{"🤖 a line\n# Heading", "#"},
		{"🤖 a line\n   ### Heading", "###"},
	} {
		if got := Markdown(c.body); got != c.want {
			t.Errorf("Markdown(%q) = %q, want %q", c.body, got, c.want)
		}
	}
}

// TestMarkdownLeavesPlainWordsAlone is the other direction, and it is the half
// that keeps the rule usable. A hyphen bullet reads as a list in plain text, and
// a number sign in the middle of a sentence is a number sign somebody typed:
// refusing either would cost a reply nothing was wrong with.
func TestMarkdownLeavesPlainWordsAlone(t *testing.T) {
	for _, body := range []string{
		"🤖 The 2026 register.",
		"🤖 Two points:\n- the date\n- the owner",
		"🤖 See issue #28 for the rest.",
		"🤖 The cost is 5 * 3 units.",
	} {
		if got := Markdown(body); got != "" {
			t.Errorf("Markdown(%q) = %q, and there is no markdown in it", body, got)
		}
	}
}

// TestPlaintextRefusesAMarker is decision 3's other direction. gdoc's markers
// travel out of a document into a file in the hub; none of them may travel back
// in. A reply or a comment carrying "{+words+}[s:AAA]" puts a suggestion's
// brackets into a thread as prose, and the thread renders them as typed, which
// is the same failure the markdown rule exists for.
func TestPlaintextRefusesAMarker(t *testing.T) {
	for _, c := range []struct{ body, want string }{
		{"🤖 the sentence reads {+six months+}[s:AAA] now", "{+"},
		{"🤖 the words {-annually-}[s:AAA] are gone", "{-"},
		{"🤖 see [[c:AAA]]the operations team[[/c]]", "[[c:"},
		{"🤖 that suggestion is [s:AAA]", "[s:"},
	} {
		if got := Marker(c.body); got != c.want {
			t.Errorf("Marker(%q) = %q, want %q", c.body, got, c.want)
		}
	}
}

// TestMarkerLeavesPlainWordsAlone is the half that keeps a reply writable. A
// reply talking about the markers writes them escaped, the way `read` does, and
// an ordinary sentence with a bracket in it is nobody's marker.
func TestMarkerLeavesPlainWordsAlone(t *testing.T) {
	for _, body := range []string{
		"🤖 The 2026 register.",
		`🤖 the export writes \{+words\+} for an insertion`,
		"🤖 See item [1] of the schedule.",
	} {
		if got := Marker(body); got != "" {
			t.Errorf("Marker(%q) = %q, and there is no marker of gdoc's in it", body, got)
		}
	}
}

// The answers OpensWithRobot gives. It is true where the mark opens the text
// and false everywhere else, and a mark in the middle of a sentence is not the
// text opening with it.
//
// The leading whitespace cases are the reason the check exists at all: a reply
// whose first line is blank still opens with the mark.
//
// That three readers share this one check rather than writing it out again is
// the other half, and it is read off the tree by
// TestTheRobotMarkIsReadInOnePlace in go/boundary.
func TestOpensWithRobotIsTrueOnlyWhereTheMarkOpensTheText(t *testing.T) {
	for _, c := range []struct {
		text string
		want bool
	}{
		{"🤖 the register is reviewed", true},
		{"🤖the register is reviewed", true},
		{"\n\n🤖 the register is reviewed", true},
		{"  \t🤖 the register is reviewed", true},
		{"the register is reviewed", false},
		{"", false},
		{"Please ask 🤖 about it", false},
	} {
		if got := OpensWithRobot(c.text); got != c.want {
			t.Errorf("OpensWithRobot(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}

// The known list, as literals. It is the robot every old reply carries and the
// two quieter marks beside it, and it is read back rather than read off the
// constant, because a test that reads the constant follows it wherever
// somebody moves it.
//
// The second half is the reason KnownMarks is a function: a caller handed the
// one slice could sort it, trim it or write over an entry, and every later
// reader of the list would inherit that. A new slice on every call is the
// cheapest way to say the list is not anybody's to change.
func TestTheKnownMarksAreTheRobotAndTheTwoDots(t *testing.T) {
	want := []string{"🤖", "⁘", "∴"}
	got := KnownMarks()
	if len(got) != len(want) {
		t.Fatalf("KnownMarks() = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("KnownMarks()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	got[0] = "Bot:"
	if again := KnownMarks(); again[0] != "🤖" {
		t.Errorf("after a caller wrote over the list, KnownMarks()[0] = %q, want %q", again[0], "🤖")
	}
}

// The default, as a literal. ⁘ is decision B3 of the M16 specification: the
// robot stays known forever, because old replies carry it, and what a new run
// writes is the quieter mark.
func TestTheDefaultMarkIsTheFourDot(t *testing.T) {
	if DefaultMark != "⁘" {
		t.Errorf("DefaultMark = %q, want %q", DefaultMark, "⁘")
	}
}

// What a mark may not be, and each refusal names what it found. A thread
// renders what it is given, so a mark with a line break in it opens the words
// on the second line, a mark Docs reads as markdown arrives as typed, and a
// mark carrying one of gdoc's own markers says a thing about a document that
// the thread turns into prose.
//
// The markdown rule is asked of the mark with words behind it, because the
// heading arm is anchored to a line start: "#" alone is a number sign, and
// "# the register" is a heading. A mark is what the words open with, so the
// question is about the line the reader sees.
func TestCheckMarkRefusesWhatAThreadWouldShowWrong(t *testing.T) {
	for _, c := range []struct{ name, mark, names string }{
		{"a line break", "⁘\n", "line break"},
		{"bold", "**", "**"},
		{"code", "`", "`"},
		{"a heading", "#", "#"},
		{"a marker of gdoc's", "{+", "{+"},
		{"nine characters", "123456789", "eight"},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := CheckMark(c.mark)
			if err == nil {
				t.Fatalf("CheckMark(%q) = nil, and a thread would show that mark wrong", c.mark)
			}
			if !strings.Contains(err.Error(), c.names) {
				t.Errorf("CheckMark(%q) = %q, and it does not name %q", c.mark, err, c.names)
			}
		})
	}
}

// The other direction, which is the half that keeps the setting usable. Each
// known mark passes, the empty mark passes because writing the words alone is
// a choice a colleague may make, a person's own word passes because decision
// B2 says any value may be set, and eight characters pass because the ceiling
// is counted in runes: eight four-dot marks are eight characters and
// thirty-two bytes.
func TestCheckMarkAcceptsEachKnownMarkTheEmptyMarkAndEightCharacters(t *testing.T) {
	for _, mark := range []string{"🤖", "⁘", "∴", "", "Bot:", "12345678", "⁘⁘⁘⁘⁘⁘⁘⁘"} {
		if err := CheckMark(mark); err != nil {
			t.Errorf("CheckMark(%q) = %v, and that is a mark a thread shows as typed", mark, err)
		}
	}
}

// The empty mark writes the words alone, with no space in front of them. It is
// what a colleague who wants no mark at all gets, and the price is in the
// specification: their own replies read as a person's to everybody else's
// gdoc.
func TestAnEmptyMarkPrefixesNothing(t *testing.T) {
	if got, want := Prefixed("", "the register is reviewed"), "the register is reviewed"; got != want {
		t.Errorf("Prefixed(%q, ...) = %q, want %q", "", got, want)
	}
	if got, want := Prefixed("⁘", "the register is reviewed"), "⁘ the register is reviewed"; got != want {
		t.Errorf("Prefixed(%q, ...) = %q, want %q", "⁘", got, want)
	}
}

// The answers OpensWithMark gives. Every known mark is gdoc's, whichever
// colleague's run wrote it, and the mark in effect is gdoc's as well when a
// person set one of their own. A mark in the middle of a sentence is not the
// text opening with it, and leading whitespace is stepped over, because a
// reply whose first line is blank still opens with the mark.
//
// "Bot:" is decision B2's price in one line: it is this person's mark and
// nobody else's, so the same text is gdoc's with that mark in effect and a
// person's reply without it.
func TestOpensWithMarkKnowsEveryKnownMarkAndTheMarkInEffect(t *testing.T) {
	for _, c := range []struct {
		text, inEffect string
		want           bool
	}{
		{"🤖 the register is reviewed", "⁘", true},
		{"⁘ the register is reviewed", "⁘", true},
		{"∴ the register is reviewed", "⁘", true},
		{"⁘the register is reviewed", "⁘", true},
		{"\n\n⁘ the register is reviewed", "⁘", true},
		{"  \t🤖 the register is reviewed", "⁘", true},
		{"⁘ the register is reviewed", "", true},
		{"Bot: the register is reviewed", "Bot:", true},
		{"Bot: the register is reviewed", "⁘", false},
		{"Bot: the register is reviewed", "", false},
		{"the register is reviewed", "⁘", false},
		{"", "⁘", false},
		{"Please ask ⁘ about it", "⁘", false},
	} {
		if got := OpensWithMark(c.text, c.inEffect); got != c.want {
			t.Errorf("OpensWithMark(%q, %q) = %v, want %v", c.text, c.inEffect, got, c.want)
		}
	}
}

// MarkOpening names the character it found, which is what a writer refusing a
// reason that already carries a mark puts in the refusal. A refusal naming the
// mark this run would have written instead is one a colleague cannot act on:
// the reason they typed opens with the robot and the run writes the four dots.
//
// The longest match wins, for the reason CutMark takes the longest off: a
// person whose own mark opens with a known one is signed with their own.
func TestMarkOpeningNamesTheMarkItFound(t *testing.T) {
	for _, c := range []struct {
		text, inEffect, want string
		found                bool
	}{
		{"🤖 the register is reviewed", "⁘", "🤖", true},
		{"⁘ the register is reviewed", "⁘", "⁘", true},
		{"∴ the register is reviewed", "⁘", "∴", true},
		{"\n  Bot: the register", "Bot:", "Bot:", true},
		{"⁘⁘ the register", "⁘⁘", "⁘⁘", true},
		{"Bot: the register", "⁘", "", false},
		{"the register is reviewed", "", "", false},
		{"", "⁘", "", false},
	} {
		got, found := MarkOpening(c.text, c.inEffect)
		if got != c.want || found != c.found {
			t.Errorf("MarkOpening(%q, %q) = %q, %v, want %q, %v", c.text, c.inEffect, got, found, c.want, c.found)
		}
	}
}

// One mark and one space come off, and no more. A body carrying two marks
// keeps the second, because the caller asked for a mark to be taken off and
// not for the words to be cleaned up, and a second mark is a thing the writer
// then signs once rather than a thing that quietly disappears.
//
// Leading whitespace is stepped over for the same reason OpensWithMark steps
// over it: the two are asked together, and a pair that disagreed about a body
// whose first line is blank would say the mark is there and then leave it on.
func TestCutMarkTakesOneMarkAndOneSpace(t *testing.T) {
	for _, c := range []struct {
		text, inEffect, want string
		cut                  bool
	}{
		{"⁘ the register", "⁘", "the register", true},
		{"🤖 the register", "⁘", "the register", true},
		{"∴ the register", "", "the register", true},
		{"⁘the register", "⁘", "the register", true},
		{"⁘  the register", "⁘", " the register", true},
		{"\n⁘ the register", "⁘", "the register", true},
		{"⁘ ⁘ twice", "⁘", "⁘ twice", true},
		{"Bot: the register", "Bot:", "the register", true},
		{"Bot: the register", "⁘", "Bot: the register", false},
		{"the register", "⁘", "the register", false},
		{"", "⁘", "", false},
	} {
		got, cut := CutMark(c.text, c.inEffect)
		if got != c.want || cut != c.cut {
			t.Errorf("CutMark(%q, %q) = %q, %v, want %q, %v", c.text, c.inEffect, got, cut, c.want, c.cut)
		}
	}
}
