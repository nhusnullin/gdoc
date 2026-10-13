// The facts a chat answer carries beside every comment and reply: five literal
// checks on the words, and the author's domain, which decides nothing.

package chat

import (
	"regexp"
	"strings"
	"unicode"
)

// Facts is what one comment or reply carries beside its words in a chat answer.
//
// Every field is a check on the text and nothing else, so the person hears a
// fact about the item in front of them rather than a judgement: "this comment
// has a link and addresses the AI". Nothing here says whether a comment is
// handled, trustworthy or worth acting on. The hold rules in rules.go read
// HasLink, NamesAI and HiddenChars; nothing reads AuthorDomain.
//
// TestEachFactHasOneCheck holds every check against a fixture that trips it and
// one that does not, and TestTheFactNamesAreTheLiterals holds the names the
// model reads.
type Facts struct {
	HasLink      bool   `json:"has_link"`
	HasEmail     bool   `json:"has_email"`
	NamesAI      bool   `json:"names_ai"`
	HiddenChars  bool   `json:"hidden_chars"`
	AuthorDomain string `json:"author_domain,omitempty"`
}

// Comment is one comment or reply the facts are about: what it says, and the
// domain of whoever wrote it. The id Drive gave it is not here, because no fact
// asks anything of the item beyond its own words.
type Comment struct {
	Text         string
	AuthorDomain string
}

// aiWords is the fixed list that makes NamesAI true, from the specification.
// Case is folded and each has to stand as whole words, so "the maintenance plan"
// does not carry "ai" and "the previous paragraph" does not carry "ignore
// previous".
var aiWords = []string{"AI", "assistant", "Claude", "ignore previous", "approved by"}

// aiPattern is that list as one expression. Whitespace between the words of a
// phrase is any run of it, because a comment written over two lines says the
// same thing as one written over one.
var aiPattern = regexp.MustCompile(`(?i)\b(?:` + strings.Join(phrases(aiWords), "|") + `)\b`)

func phrases(words []string) []string {
	out := make([]string, 0, len(words))
	for _, w := range words {
		parts := strings.Fields(w)
		for i, p := range parts {
			parts[i] = regexp.QuoteMeta(p)
		}
		out = append(out, strings.Join(parts, `\s+`))
	}
	return out
}

// emailPattern is an address. It is deliberately looser than the standard
// allows on the left of the at sign and stricter on the right: what matters is
// that a reader would send mail to it.
var emailPattern = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9](?:[a-z0-9\-]*[a-z0-9])?(?:\.[a-z0-9\-]+)*\.[a-z]{2,}`)

// urlPattern is a link written as a link: a scheme, a host under www, or a host
// with a path on it. Any of the three is unambiguous.
var urlPattern = regexp.MustCompile(`(?i)(?:[a-z][a-z0-9+.\-]*://|\bwww\.[a-z0-9]|\b[a-z0-9](?:[a-z0-9\-]*[a-z0-9])?(?:\.[a-z0-9\-]+)*\.[a-z]{2,}/)`)

// linkTLDs is the list a bare host has to end in.
//
// A dot between two words is usually a sentence ending and not a host, so a bare
// host is read only where the last label is one of these. The alternative, any
// two letters after a dot, reads "ends here.The next" as a link, and a fact that
// fires on ordinary prose is a hold the person learns to wave through. A link
// written with a scheme, with www or with a path is caught by urlPattern
// whatever it ends in.
var linkTLDs = []string{
	"ai", "app", "co", "com", "de", "dev", "edu", "eu", "fr", "gov",
	"info", "io", "me", "net", "nl", "org", "ru", "uk", "xyz",
}

var bareHostPattern = regexp.MustCompile(`(?i)\b[a-z0-9](?:[a-z0-9\-]*[a-z0-9])?(?:\.[a-z0-9\-]+)*\.(?:` + strings.Join(linkTLDs, "|") + `)\b`)

// FactsOf reads one comment or reply. Every answer is a check on the text it
// was handed, so the same words read the same way in every session.
func FactsOf(c Comment) Facts {
	// The addresses come out before the links are looked for, so an address is
	// one fact and not two.
	withoutEmails := emailPattern.ReplaceAllString(c.Text, " ")
	return Facts{
		HasLink:      urlPattern.MatchString(withoutEmails) || bareHostPattern.MatchString(withoutEmails),
		HasEmail:     emailPattern.MatchString(c.Text),
		NamesAI:      aiPattern.MatchString(c.Text),
		HiddenChars:  HasHiddenChars(c.Text),
		AuthorDomain: c.AuthorDomain,
	}
}

// HasHiddenChars is true when the text carries a character a person reading it
// cannot see: the zero-width characters, the bidi controls and the tags are one
// Unicode category between them, Cf, and the tag block is in it.
//
// It is one function because two readings of the same rune tables would drift,
// and two rooms ask it: FactsOf reports it beside a comment, and Rules refuses
// such text outright rather than holding it.
// TestHasHiddenCharsIsTheSameCheckTheFactReports. Variation selectors are not
// here, and that is on purpose: they are how an emoji is written, and 🤖 is the
// one mark gdoc signs with.
func HasHiddenChars(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Cf, r) {
			return true
		}
	}
	return false
}
