package plaintext

import "testing"

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
