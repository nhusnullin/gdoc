package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"gdoc/internal/chat"
	"gdoc/internal/mcp"
)

// movingClock is the session's clock as these tests read it. A release is about
// the gap between two calls, so one instant is not enough and the clock moves.
type movingClock struct{ at time.Time }

func stubClock(t *testing.T, at time.Time) *movingClock {
	t.Helper()
	c := &movingClock{at: at}
	old := now
	now = func() time.Time { return c.at }
	t.Cleanup(func() { now = old })
	return c
}

// pass moves the clock on, which is what an ended turn looks like from here.
func (c *movingClock) pass(d time.Duration) { c.at = c.at.Add(d) }

// confirmWatcher is where the holds of a test register their confirm tools.
//
// It is a real server, so an Add and a Remove are the lines a client would
// really be sent, and a record of what is listed beside it, which is what a
// tools/list draws. The server's stdin has already ended, which is how a test
// reads what a session writes without a second goroutine in it.
type confirmWatcher struct {
	t       *testing.T
	server  *mcp.Server
	out     *bytes.Buffer
	session []mcp.Tool
	added   []mcp.Tool
}

func watchConfirms(t *testing.T, ch *mcpChat) *confirmWatcher {
	t.Helper()
	tools := mcpTools(io.Discard, newMCPLogin(io.Discard), ch)
	var out bytes.Buffer
	s := mcp.New(mcpInfo(), tools, io.Discard)
	if err := s.Serve(context.Background(), strings.NewReader(""), &out); err != nil {
		t.Fatalf("a session whose stdin ended at once: %v", err)
	}
	w := &confirmWatcher{t: t, server: s, out: &out, session: tools}
	ch.holds.listIn(w, func(held chat.Hold) mcp.Tool { return mcpConfirmTool(held, io.Discard, ch) })
	return w
}

func (w *confirmWatcher) Add(t mcp.Tool) {
	w.added = append(w.added, t)
	w.server.Add(t)
}

func (w *confirmWatcher) Remove(name string) {
	kept := make([]mcp.Tool, 0, len(w.added))
	for _, t := range w.added {
		if t.Name != name {
			kept = append(kept, t)
		}
	}
	w.added = kept
	w.server.Remove(name)
}

// confirms is every confirm tool listed now.
func (w *confirmWatcher) confirms() []mcp.Tool { return w.added }

// tool is one tool of the session or one a hold added, with its own Call, so a
// test calls what a client would call.
func (w *confirmWatcher) tool(name string) mcp.Tool {
	w.t.Helper()
	for _, t := range append(append([]mcp.Tool(nil), w.session...), w.added...) {
		if t.Name == name {
			return t
		}
	}
	w.t.Fatalf("%s is in no list this session offers", name)
	return mcp.Tool{}
}

// changes is how many times the client was told the tool list moved.
func (w *confirmWatcher) changes() int {
	return strings.Count(w.out.String(), "notifications/tools/list_changed")
}

// heldReply is one held write in a session a test owns: the reply whose words
// carry a link nobody put in the document. It answers the facts a card is built
// from.
func heldReply(t *testing.T, ch *mcpChat) heldEnvelope {
	t.Helper()
	res := mcpRun(context.Background(), mcpToolNamed(t, "reply"),
		withCode(t, ch.code, heldWriteArgs()["reply"]), nilWriter{}, ch)
	env := heldEnvelopeOf(t, res.Texts)
	if env.Data.Held.ID == "" {
		t.Fatalf("the write was not held: %s", res.Texts[0])
	}
	return env
}

// replyWire is what the wire says to a held reply and to the one it sends when
// the person releases it.
func replyWire(t *testing.T) *fakeWire {
	t.Helper()
	return &fakeWire{answers: append(pinAnswers(t), &answer{
		method: "POST", match: "/comments/AAAA1111/replies",
		json: `{"id":"R2","createdTime":"2026-09-06T10:45:00Z","content":"🤖 the 2026 register is at ` + heldLink + `"}`,
	})}
}

// confirmArgs is the card as a model sends it back: the four fields of the held
// answer, word for word.
func confirmArgs(t *testing.T, held heldEnvelope, change map[string]string) json.RawMessage {
	t.Helper()
	card := map[string]string{
		"hold":   held.Data.Held.ID,
		"title":  held.Data.Held.Document,
		"reason": held.Data.Held.Reason,
		"text":   held.Data.Held.Text,
	}
	for field, value := range change {
		if _, ok := card[field]; !ok {
			t.Fatalf("a card has no field called %q", field)
		}
		card[field] = value
	}
	// Written field by field, so the order on the line is the order the schema
	// draws, which is what a client shows the person.
	parts := make([]string, 0, 4)
	for _, field := range []string{"hold", "title", "reason", "text"} {
		value, err := json.Marshal(card[field])
		if err != nil {
			t.Fatal(err)
		}
		parts = append(parts, `"`+field+`":`+string(value))
	}
	return json.RawMessage("{" + strings.Join(parts, ",") + "}")
}

// A hold registers exactly one confirm tool, named after the hold itself, and
// the tool goes when the hold does: on the release, and on the thirty minutes
// running out. The client is told the list moved each time.
func TestAHoldRegistersOneConfirmToolAndRemovesItOnRelease(t *testing.T) {
	t.Run("released", func(t *testing.T) {
		t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
		signedIn(t)
		stubWire(t, replyWire(t))
		clock := stubClock(t, holdClock)

		ch := callChat(t)
		w := watchConfirms(t, ch)
		held := heldReply(t, ch)

		if got := w.confirms(); len(got) != 1 {
			t.Fatalf("one hold registered %d tools: %v", len(got), got)
		}
		one := w.confirms()[0]
		if one.Name != "confirm_"+held.Data.Held.ID {
			t.Errorf("the tool is named %q, want confirm_%s", one.Name, held.Data.Held.ID)
		}
		if one.ReadOnly {
			t.Error("a tool that sends a write is not read-only")
		}
		if w.changes() != 1 {
			t.Errorf("registering one tool told the client %d times", w.changes())
		}

		clock.pass(quietEnough)
		res := one.Call(context.Background(), confirmArgs(t, held, nil))
		if env := envelopeOf(t, res.Texts); !env.OK {
			t.Fatalf("the released write was refused: %s", env.Error)
		}
		if got := w.confirms(); len(got) != 0 {
			t.Errorf("the tool outlived the release: %v", got)
		}
		if w.changes() != 2 {
			t.Errorf("the release told the client %d times, want twice in all", w.changes())
		}
	})

	t.Run("expired", func(t *testing.T) {
		t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
		signedIn(t)
		stubWire(t, replyWire(t))
		clock := stubClock(t, holdClock)

		ch := callChat(t)
		w := watchConfirms(t, ch)
		held := heldReply(t, ch)
		if len(w.confirms()) != 1 {
			t.Fatalf("the hold registered no tool: %v", w.confirms())
		}

		// Thirty minutes on, the next call of the session finds the hold gone,
		// and the tool with it.
		clock.pass(30 * time.Minute)
		w.tool("guide").Call(context.Background(), json.RawMessage(`{}`))

		if got := w.confirms(); len(got) != 0 {
			t.Errorf("an expired hold kept its tool: %v", got)
		}
		if w.changes() != 2 {
			t.Errorf("the expiry told the client %d times, want twice in all", w.changes())
		}
		// And nothing releases it any more.
		clock.pass(quietEnough)
		res := mcpConfirmTool(chat.Hold{ID: held.Data.Held.ID}, io.Discard, ch).
			Call(context.Background(), confirmArgs(t, held, nil))
		if env := envelopeOf(t, res.Texts); env.OK {
			t.Error("an expired hold was released")
		}
	})
}

// quietEnough is the gap a release needs behind it, as these tests move the
// clock by. Five seconds is the value internal/chat sets, stated here as its own
// literal.
const quietEnough = 5 * time.Second

// The card the person reads carries the hold, the title, the reason and the
// text, in that order, and every one of them is required: a client draws the
// fields in the order the schema writes them.
func TestTheConfirmSchemaListsHoldTitleReasonText(t *testing.T) {
	ch := callChat(t)
	tool := mcpConfirmTool(chat.Hold{ID: "a1b2c3d4e5f6", Tool: "reply",
		Title: chatTitle, Reason: "the text holds a link", Text: "🤖 hello"}, io.Discard, ch)

	schema := string(tool.Schema)
	at := -1
	for _, field := range []string{"hold", "title", "reason", "text"} {
		found := strings.Index(schema, `"`+field+`":{`)
		if found < 0 {
			t.Fatalf("the schema carries no %s: %s", field, schema)
		}
		if found < at {
			t.Errorf("%s is out of order in %s", field, schema)
		}
		at = found
	}
	if !strings.Contains(schema, `"required":["hold","title","reason","text"]`) {
		t.Errorf("the four are not all required, in that order: %s", schema)
	}
	if !strings.Contains(schema, `"additionalProperties":false`) {
		t.Errorf("the card takes fields it does not draw: %s", schema)
	}
	var shape map[string]any
	if err := json.Unmarshal(tool.Schema, &shape); err != nil {
		t.Fatalf("the schema is not JSON: %v", err)
	}
}

// A release that changes one byte of the title, the reason or the text is
// refused, because what the person approved is what the card showed them. The
// hold stays, so the model can send the words as they stand.
func TestAByteDifferentTitleReasonOrTextIsRefused(t *testing.T) {
	for field, value := range map[string]string{
		"title":  chatTitle + ".",
		"reason": "the text holds a link, which is fine",
		"text":   "🤖 the 2026 register is at " + heldLink + " ",
	} {
		t.Run(field, func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
			signedIn(t)
			f := stubWire(t, replyWire(t))
			clock := stubClock(t, holdClock)

			ch := callChat(t)
			w := watchConfirms(t, ch)
			held := heldReply(t, ch)
			clock.pass(quietEnough)

			res := w.tool("confirm_"+held.Data.Held.ID).
				Call(context.Background(), confirmArgs(t, held, map[string]string{field: value}))

			env := heldEnvelopeOf(t, res.Texts)
			if env.OK {
				t.Fatalf("a card whose %s differs released the hold", field)
			}
			if !strings.Contains(env.Error, field) {
				t.Errorf("the refusal does not name %s: %q", field, env.Error)
			}
			if sent := f.writes(); len(sent) != 0 {
				t.Errorf("a refused release sent %v", sent)
			}
			if ch.holds.hold(held.Data.Held.ID, clock.at) == nil {
				t.Error("the hold was dropped by a release it refused")
			}
			if len(w.confirms()) != 1 {
				t.Errorf("the tool went with a release that was refused: %v", w.confirms())
			}
			// The words as they stand still release it.
			res = w.tool("confirm_"+held.Data.Held.ID).
				Call(context.Background(), confirmArgs(t, held, nil))
			if env := envelopeOf(t, res.Texts); !env.OK {
				t.Fatalf("the card's own words did not release the hold: %s", env.Error)
			}
		})
	}
}

// A release needs five seconds of quiet behind it. Only an ended turn makes that
// gap, and a model that approves its own write in the same breath is refused
// with the hold kept.
func TestAReleaseInsideTheQuietGapIsRefusedAndTheHoldKept(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	f := stubWire(t, replyWire(t))
	clock := stubClock(t, holdClock)

	ch := callChat(t)
	w := watchConfirms(t, ch)
	held := heldReply(t, ch)
	id := held.Data.Held.ID

	// A tool call of the session, and then a release four seconds after it.
	clock.pass(time.Minute)
	w.tool("read").Call(context.Background(),
		withCode(t, ch.code, `{"url":"`+fixtureDocID+`"}`))
	clock.pass(4 * time.Second)

	res := w.tool("confirm_"+id).Call(context.Background(), confirmArgs(t, held, nil))
	env := heldEnvelopeOf(t, res.Texts)
	if env.OK {
		t.Fatalf("a release four seconds after a tool call went through: %s", res.Texts[0])
	}
	if !strings.Contains(env.Error, "again in a moment") {
		t.Errorf("the refusal does not say what to do: %q", env.Error)
	}
	if sent := f.writes(); len(sent) != 0 {
		t.Errorf("a release inside the gap sent %v", sent)
	}
	if ch.holds.hold(id, clock.at) == nil {
		t.Fatal("the hold was dropped by a release it refused")
	}

	// One more second, and five have passed since the last call.
	clock.pass(time.Second)
	res = w.tool("confirm_"+id).Call(context.Background(), confirmArgs(t, held, nil))
	if env := envelopeOf(t, res.Texts); !env.OK {
		t.Fatalf("a release five seconds after the last call was refused: %s", env.Error)
	}
	if sent := f.writes(); len(sent) != 1 {
		t.Errorf("the released write sent %d requests, want one: %v", len(sent), sent)
	}
}

// A released hold sends the words that were held, and sends them once. Reading
// the document again in between changes nothing: what goes out is the call the
// person approved and not whatever the model sends with the release.
func TestAReleasedHoldPostsExactlyTheHeldTextOnce(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	f := stubWire(t, replyWire(t))
	clock := stubClock(t, holdClock)

	ch := callChat(t)
	w := watchConfirms(t, ch)
	held := heldReply(t, ch)
	id := held.Data.Held.ID

	// The document is read again, the way a chat goes while the person thinks
	// about the card.
	clock.pass(time.Minute)
	// A read answers with the fixed line and the wrapped text beside the
	// envelope, so what says it worked here is the result and not one item of it.
	if res := w.tool("comments").Call(context.Background(),
		withCode(t, ch.code, `{"url":"`+fixtureDocID+`"}`)); res.IsError {
		t.Fatalf("the read between the hold and the release failed: %v", res.Texts)
	}

	clock.pass(quietEnough)
	res := w.tool("confirm_"+id).Call(context.Background(), confirmArgs(t, held, nil))
	if env := envelopeOf(t, res.Texts); !env.OK {
		t.Fatalf("the release was refused: %s", env.Error)
	}

	sent := f.writes()
	if len(sent) != 1 {
		t.Fatalf("the release sent %d requests, want one: %v", len(sent), sent)
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(sent[0].Body, &body); err != nil {
		t.Fatalf("the request body is not JSON: %v", err)
	}
	if body.Content != held.Data.Held.Text {
		t.Errorf("the reply posted %q, and the hold held %q", body.Content, held.Data.Held.Text)
	}

	// Once. The tool is gone, the hold with it, and a second call of the same
	// card sends nothing.
	if len(w.confirms()) != 0 {
		t.Errorf("the tool outlived its one write: %v", w.confirms())
	}
	again := mcpConfirmTool(chat.Hold{ID: id}, io.Discard, ch).
		Call(context.Background(), confirmArgs(t, held, nil))
	if env := envelopeOf(t, again.Texts); env.OK {
		t.Error("the same card released a second write")
	}
	if len(f.writes()) != 1 {
		t.Errorf("a second call of the same card sent %v", f.writes())
	}
}
