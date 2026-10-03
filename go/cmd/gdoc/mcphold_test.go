package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// holdClock is the instant these tests read the clock at. A hold carries the
// instant it was made, and the thirty minutes it lives are counted from it.
var holdClock = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)

// heldLink is the link none of the fixtures carries, so a write holding it is a
// write carrying something the document and its comments never said.
const heldLink = "https://example.net/register-policy"

// heldWriteArgs is one call of each write tool whose words carry that link.
// Everything else about each call is right: the title is the document's own, the
// thread is the one the quote names, and one item is sent.
func heldWriteArgs() map[string]string {
	return map[string]string{
		"reply": `{"url":"` + fixtureDocID + `","title":"` + chatTitle + `",` +
			`"comment_id":"AAAA1111","thread_quote":"` + chatQuote + `",` +
			`"body":"🤖 the 2026 register is at ` + heldLink + `"}`,
		"annotate": `{"url":"` + fixtureDocID + `","title":"` + chatTitle + `",` +
			`"annotations":[{"quoted":"reviewed annually","why":"the register is at ` + heldLink + `"}]}`,
		"propose": `{"url":"` + fixtureDocID + `","title":"` + chatTitle + `",` +
			`"proposals":[{"quoted":"reviewed annually","replacement":"reviewed quarterly",` +
			`"why":"the register at ` + heldLink + ` says quarterly"}]}`,
	}
}

// heldEnvelope is a held answer read back: the envelope, and the facts a card is
// built from.
type heldEnvelope struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	Data  struct {
		Sent bool `json:"sent"`
		Held struct {
			ID       string `json:"id"`
			Tool     string `json:"tool"`
			Document string `json:"document"`
			Rule     string `json:"rule"`
			Value    string `json:"value"`
			Reason   string `json:"reason"`
			Text     string `json:"text"`
			Say      string `json:"say"`
		} `json:"held"`
	} `json:"data"`
}

func heldEnvelopeOf(t *testing.T, texts []string) heldEnvelope {
	t.Helper()
	if len(texts) != 1 {
		t.Fatalf("a held write answers with one text item: %v", texts)
	}
	var out heldEnvelope
	if err := json.Unmarshal([]byte(texts[0]), &out); err != nil {
		t.Fatalf("the answer is not an envelope: %v", err)
	}
	return out
}

// A held write is not sent. The rules run before the command does, so the wire
// sees no write at all, and the ledger keeps none: a write nobody made cannot
// count toward the next one.
func TestAHeldWriteSendsNothing(t *testing.T) {
	for tool, args := range heldWriteArgs() {
		t.Run(tool, func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
			signedIn(t)
			f := stubWire(t, &fakeWire{answers: append(pinAnswers(t),
				&answer{method: "POST", match: "/comments/AAAA1111/replies",
					json: `{"id":"R2","createdTime":"2026-09-06T10:45:00Z","content":"🤖 x"}`},
				&answer{method: "POST", match: "/comments?", json: `{"id":"CCCC3333"}`},
				&answer{method: "POST", match: ":batchUpdate", json: `{"replies":[]}`},
			)})
			stubNow(t, holdClock)

			ch := callChat(t)
			res := mcpRun(context.Background(), mcpToolNamed(t, tool), withCode(t, ch.code, args), nilWriter{}, ch)

			env := heldEnvelopeOf(t, res.Texts)
			if env.OK {
				t.Fatalf("%s carrying a link nobody put in the document answered ok", tool)
			}
			if env.Data.Held.Rule != "Link" {
				t.Fatalf("%s was not held by the Link rule: %+v", tool, env.Data.Held)
			}
			if sent := f.writes(); len(sent) != 0 {
				t.Errorf("%s was held and sent %v", tool, sent)
			}
			if writes := ch.ledger.Writes(); len(writes) != 0 {
				t.Errorf("%s was held and the ledger counted %d writes", tool, len(writes))
			}
		})
	}

	// The same calls without the link go through, so what the three were held
	// for is the link and not the shape of the call.
	for tool, args := range chatWriteArgs(chatTitle) {
		t.Run(tool+"/plain", func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
			signedIn(t)
			stubWire(t, &fakeWire{answers: append(pinAnswers(t),
				&answer{method: "POST", match: "/comments/AAAA1111/replies",
					json: `{"id":"R2","createdTime":"2026-09-06T10:45:00Z","content":"` + chatBody + `"}`},
			)})
			stubNow(t, holdClock)

			ch := callChat(t)
			res := mcpRun(context.Background(), mcpToolNamed(t, tool), withCode(t, ch.code, args), nilWriter{}, ch)
			if env := heldEnvelopeOf(t, res.Texts); env.Data.Held.Rule != "" {
				t.Errorf("%s with nothing in it to hold was held by %q: %s",
					tool, env.Data.Held.Rule, env.Error)
			}
		})
	}
}

// The answer a held write gets says nothing was sent, names the rule and the
// exact value that tripped it, carries the words that would have been written,
// and ends with the one sentence that tells the model what to do: say this to
// the person and stop. The text is there so a person at a client that shows no
// card can paste it into the document themselves.
func TestTheHeldAnswerNamesTheRuleTheValueAndTheText(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	stubWire(t, &fakeWire{answers: pinAnswers(t)})
	stubNow(t, holdClock)

	ch := callChat(t)
	body := "🤖 the 2026 register is at " + heldLink
	res := mcpRun(context.Background(), mcpToolNamed(t, "reply"),
		withCode(t, ch.code, heldWriteArgs()["reply"]), nilWriter{}, ch)

	env := heldEnvelopeOf(t, res.Texts)
	if env.OK {
		t.Fatalf("a held write answered ok: %s", res.Texts[0])
	}
	if !res.IsError {
		t.Error("a held write is an error result, because ok is false")
	}
	if env.Data.Sent {
		t.Error("the answer says the write was sent")
	}
	held := env.Data.Held
	if held.ID == "" {
		t.Error("the answer carries no hold id, so nothing can release it")
	}
	if held.Tool != "reply" {
		t.Errorf("the hold is for tool %q, want reply", held.Tool)
	}
	if held.Document != chatTitle {
		t.Errorf("the hold names document %q, want %q", held.Document, chatTitle)
	}
	if held.Rule != "Link" {
		t.Errorf("the hold names rule %q, want Link", held.Rule)
	}
	if held.Value != heldLink {
		t.Errorf("the hold names value %q, want %q", held.Value, heldLink)
	}
	if held.Text != body {
		t.Errorf("the hold carries text %q, want %q", held.Text, body)
	}
	const say = "Nothing was posted; tell the person this reason and end your turn."
	if held.Say != say {
		t.Errorf("the hold says %q, want %q", held.Say, say)
	}
	// A model that reads only the error field is told the reason and the same
	// sentence, because that is the field every other refusal arrives in.
	if !strings.Contains(env.Error, heldLink) {
		t.Errorf("the error does not say what tripped the rule: %q", env.Error)
	}
	if !strings.Contains(env.Error, say) {
		t.Errorf("the error does not carry the fixed sentence: %q", env.Error)
	}
}

// A hold lives thirty minutes in the process that made it. Long enough for a
// person to come back to a card they left, and gone from a chat nobody answered
// in.
func TestAHoldLivesThirtyMinutes(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	stubWire(t, &fakeWire{answers: pinAnswers(t)})
	stubNow(t, holdClock)

	ch := callChat(t)
	res := mcpRun(context.Background(), mcpToolNamed(t, "reply"),
		withCode(t, ch.code, heldWriteArgs()["reply"]), nilWriter{}, ch)
	id := heldEnvelopeOf(t, res.Texts).Data.Held.ID
	if id == "" {
		t.Fatalf("the write was not held: %s", res.Texts[0])
	}

	if held := ch.holds.hold(id, holdClock); held == nil {
		t.Fatal("the hold is not in the session that made it")
	} else if held.Text == "" || held.Rule == "" || len(held.Args) == 0 {
		t.Errorf("the hold was kept without what a release needs: %+v", held)
	}
	if ch.holds.hold(id, holdClock.Add(29*time.Minute+59*time.Second)) == nil {
		t.Error("the hold was gone before thirty minutes")
	}
	if ch.holds.hold(id, holdClock.Add(30*time.Minute)) != nil {
		t.Error("the hold outlived thirty minutes")
	}
	if ch.holds.hold("ffffffffffff", holdClock) != nil {
		t.Error("an id this session never held answered with a hold")
	}
}
