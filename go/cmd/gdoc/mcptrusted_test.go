package main

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

// trustedClock is the instant these tests read the clock at.
var trustedClock = time.Date(2026, 10, 3, 15, 0, 0, 0, time.UTC)

// trustedBody is a reply carrying an address at the domain the session below
// trusts. It is a reply and not a comment because a colleague writing their own
// firm's address into a thread is the shape the setting exists for.
const trustedBody = "🤖 write to registry@example.com"

// trustedReplyArgs is that reply as a model sends it.
func trustedReplyArgs() string {
	return `{"url":"` + fixtureDocID + `","title":"` + chatTitle + `",` +
		`"comment_id":"AAAA1111","thread_quote":"` + chatQuote + `","body":"` + trustedBody + `"}`
}

// A value gdoc half understands does not stop the server: it starts, says the
// reason on its log, and every tool but guide answers that reason and runs
// nothing. The person reads it in the chat and fixes the field in Claude
// Desktop, which is the only place they can fix it from.
//
// guide still answers, because guide is where the person is told what this
// session is running with and it reaches no document.
func TestAMalformedValueMakesEveryToolButGuideNameIt(t *testing.T) {
	// A wildcard, a web address, an email address and a word with no dot.
	for _, raw := range []string{"*.example.com", "https://example.com", "registry@example.com", "example"} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
			signedIn(t)

			var log strings.Builder
			ch, err := newMCPChat(raw)
			if err != nil {
				t.Fatalf("a malformed setting stopped the session from starting: %v", err)
			}
			if ch.trustedErr == nil {
				t.Fatalf("%q was read as a list of domains, want it refused", raw)
			}
			if len(ch.trusted) != 0 {
				t.Errorf("%q was refused and left %v trusted", raw, ch.trusted)
			}

			tools := mcpTools(&log, newMCPLogin(io.Discard), ch)
			if len(tools) == 0 {
				t.Fatal("the session offers no tools")
			}
			for _, tool := range tools {
				res := tool.Call(context.Background(), withCode(t, ch.code, `{}`))
				env := envelopeOf(t, res.Texts)
				if tool.Name == "guide" {
					if !env.OK {
						t.Errorf("guide refused with a malformed setting: %s", env.Error)
					}
					continue
				}
				if env.OK {
					t.Errorf("%s ran with a malformed setting", tool.Name)
					continue
				}
				if !strings.Contains(env.Error, "Email domains that need no approval") {
					t.Errorf("%s does not name the setting: %s", tool.Name, env.Error)
				}
				if !strings.Contains(env.Error, raw) {
					t.Errorf("%s does not name the value %q: %s", tool.Name, raw, env.Error)
				}
			}
		})
	}
}

// The setting is the one loosening of a check in gdoc, so a write it applied to
// says so. It is a fact beside the envelope: which address was not asked about,
// and that a link is never exempt whatever its domain.
func TestTheWriteAnswerNamesTheExemption(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	f := stubWire(t, &fakeWire{answers: append(pinAnswers(t), &answer{
		method: "POST", match: "/comments/AAAA1111/replies",
		json: `{"id":"R2","createdTime":"2026-09-06T10:45:00Z","content":"` + trustedBody + `"}`,
	})})
	stubClock(t, trustedClock)

	ch := callChat(t)
	looked(ch, fixtureDocID)
	ch.trusted = []string{"example.com"}
	res := mcpRun(context.Background(), mcpToolNamed(t, "reply"),
		withCode(t, ch.code, trustedReplyArgs()), nilWriter{}, ch)

	if len(res.Texts) != 2 {
		t.Fatalf("the write answered %d text items, want the envelope and the exemption: %v",
			len(res.Texts), res.Texts)
	}
	if len(f.writes()) != 1 {
		t.Fatalf("the wire saw %d writes, want the one reply", len(f.writes()))
	}
	fact := res.Texts[1]
	if !strings.Contains(fact, "registry@example.com") {
		t.Errorf("the exemption does not name the address: %q", fact)
	}
	if !strings.Contains(fact, "example.com") || !strings.Contains(strings.ToLower(fact), "link") {
		t.Errorf("the exemption does not say the domain was trusted and a link is not: %q", fact)
	}
}

// Nothing gdoc says suggests the setting. It is an advanced field for a person
// who went looking for it, and a model that offers it to a colleague is a model
// teaching them to loosen a check they never asked about: decision 17.
func TestNothingSuggestsTheSetting(t *testing.T) {
	// The phrases that would be naming it. "trust" alone is not one: the review
	// core says never to trust a status code, which is the opposite advice.
	phrases := []string{
		"trusted_email_domains", "trusted-email-domains", "trusted email", "trusted domain",
		"need no approval", "needs no approval", "email domains that",
	}
	code := callCode(t)
	tools := mcpTools(io.Discard, newMCPLogin(io.Discard), chatWith(code))
	if len(tools) == 0 {
		t.Fatal("the session offers no tools")
	}

	named := map[string]string{
		"the server instructions": mcpInstructions,
		"the chat header":         mcpChatHeader,
		"the review core":         mcpReviewCore,
	}
	for _, tool := range tools {
		named["the "+tool.Name+" tool's title"] = tool.Title
		named["the "+tool.Name+" tool's description"] = tool.Description
	}
	for where, text := range named {
		lower := strings.ToLower(text)
		for _, phrase := range phrases {
			if strings.Contains(lower, phrase) {
				t.Errorf("%s says %q, and nothing gdoc says suggests that setting", where, phrase)
			}
		}
	}
}
