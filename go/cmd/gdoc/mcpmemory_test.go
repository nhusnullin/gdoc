package main

import (
	"context"
	"testing"
	"time"
)

// memoryClock is the instant these tests read the clock at.
var memoryClock = time.Date(2026, 10, 3, 13, 0, 0, 0, time.UTC)

// writeCall is one write tool with the wire that answers it, for a test that
// makes the same call twice.
type writeCall struct {
	tool string
	doc  string
	args string
	wire func(t *testing.T) *fakeWire
}

// writeCalls is one plain call of each write tool: every argument right, nothing
// in the words a rule holds.
func writeCalls() []writeCall {
	return []writeCall{
		{
			tool: "reply", doc: fixtureDocID, args: chatWriteArgs(chatTitle)["reply"],
			wire: func(t *testing.T) *fakeWire {
				return stubWire(t, &fakeWire{answers: append(pinAnswers(t), &answer{
					method: "POST", match: "/comments/AAAA1111/replies",
					json: `{"id":"R2","createdTime":"2026-09-06T10:45:00Z","content":"` + chatBody + `"}`,
				})})
			},
		},
		{
			tool: "annotate",
			doc:  annotateDocID,
			args: `{"url":"` + annotateDocID + `","title":"` + chatTitle + `",` +
				`"annotations":[{"quoted":"reviewed annually","why":"The 2026 register says quarterly."}]}`,
			wire: func(t *testing.T) *fakeWire {
				return stubWire(t, &fakeWire{answers: annotateAnswers(t)})
			},
		},
		{
			tool: "propose",
			doc:  proposeDocID,
			args: `{"url":"` + proposeDocID + `","title":"` + chatTitle + `","proposals":` + oneProposal + `}`,
			wire: func(t *testing.T) *fakeWire {
				// The pin's read is one more of the same at the front: propose
				// reads the document three times in order, each answer spent
				// before the next is reached.
				answers := append([]*answer{{method: "GET", match: proposeDocID + "?includeTabsContent",
					json: readFixture(t, "propose-before.json"), once: true}}, proposeAnswers(t, true)...)
				return stubWire(t, &fakeWire{answers: answers})
			},
		},
	}
}

// The same write twice is written once. A client that gave up on a call and sent
// it again has no way of knowing whether the first one reached the document, so
// the session answers with the answer the first call gave and the wire sees
// nothing at all.
func TestTheSameWriteInsideTenMinutesGetsTheKeptAnswer(t *testing.T) {
	for _, one := range writeCalls() {
		t.Run(one.tool, func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
			signedIn(t)
			f := one.wire(t)
			clock := stubClock(t, memoryClock)

			ch := callChat(t)
			looked(ch, one.doc)
			first := mcpRun(context.Background(), mcpToolNamed(t, one.tool),
				withCode(t, ch.code, one.args), nilWriter{}, ch)
			if env := envelopeOf(t, first.Texts); !env.OK {
				t.Fatalf("%s was refused: %s", one.tool, env.Error)
			}
			sent := len(f.writes())
			if sent == 0 {
				t.Fatalf("%s wrote nothing, so there is nothing to keep", one.tool)
			}

			clock.pass(9 * time.Minute)
			again := mcpRun(context.Background(), mcpToolNamed(t, one.tool),
				withCode(t, ch.code, one.args), nilWriter{}, ch)

			if len(f.writes()) != sent {
				t.Errorf("%s asked twice sent %d requests, want the %d of the one write",
					one.tool, len(f.writes()), sent)
			}
			if len(again.Texts) != len(first.Texts) {
				t.Fatalf("the kept answer carries %d text items, want %d", len(again.Texts), len(first.Texts))
			}
			for i := range first.Texts {
				if again.Texts[i] != first.Texts[i] {
					t.Errorf("the kept answer is not the answer the write gave:\n  %s\nwant\n  %s",
						again.Texts[i], first.Texts[i])
				}
			}
			if again.IsError != first.IsError {
				t.Errorf("the kept answer is an error = %v, and the write's was %v", again.IsError, first.IsError)
			}
			// And the ledger counted one write, because one write happened: a
			// retry nobody wrote cannot count toward the next one.
			if writes := ch.ledger.Writes(); len(writes) != 1 {
				t.Errorf("the ledger counted %d writes for the one that happened", len(writes))
			}
		})
	}
}

// After ten minutes the same call is a new write. The window is what a client
// retry lives inside, and a model asking again later is asking for another
// write.
func TestAfterTenMinutesItIsANewWrite(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	f := stubWire(t, &fakeWire{answers: append(pinAnswers(t), &answer{
		method: "POST", match: "/comments/AAAA1111/replies",
		json: `{"id":"R2","createdTime":"2026-09-06T10:45:00Z","content":"` + chatBody + `"}`,
	})})
	clock := stubClock(t, memoryClock)

	ch := callChat(t)
	looked(ch, fixtureDocID)
	args := withCode(t, ch.code, chatWriteArgs(chatTitle)["reply"])
	if env := envelopeOf(t, mcpRun(context.Background(), mcpToolNamed(t, "reply"), args, nilWriter{}, ch).Texts); !env.OK {
		t.Fatalf("the first reply was refused: %s", env.Error)
	}
	sent := len(f.writes())

	clock.pass(10 * time.Minute)
	if env := envelopeOf(t, mcpRun(context.Background(), mcpToolNamed(t, "reply"), args, nilWriter{}, ch).Texts); !env.OK {
		t.Fatalf("the reply ten minutes later was refused: %s", env.Error)
	}
	if len(f.writes()) != 2*sent {
		t.Errorf("the second reply sent %d requests in all, want the %d of two writes",
			len(f.writes()), 2*sent)
	}
}

// A call differing in one argument is a different write. What is remembered is
// the call, so changing the words changes nothing about whether they go.
func TestADifferentArgumentIsADifferentWrite(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	f := stubWire(t, &fakeWire{answers: append(pinAnswers(t), &answer{
		method: "POST", match: "/comments/AAAA1111/replies",
		json: `{"id":"R2","createdTime":"2026-09-06T10:45:00Z","content":"` + chatBody + `"}`,
	})})
	clock := stubClock(t, memoryClock)

	ch := callChat(t)
	looked(ch, fixtureDocID)
	reply := func(body string) {
		t.Helper()
		args := `{"url":"` + fixtureDocID + `","title":"` + chatTitle + `",` +
			`"comment_id":"AAAA1111","thread_quote":"` + chatQuote + `","body":"` + body + `"}`
		res := mcpRun(context.Background(), mcpToolNamed(t, "reply"), withCode(t, ch.code, args), nilWriter{}, ch)
		if env := envelopeOf(t, res.Texts); !env.OK {
			t.Fatalf("the reply saying %q was refused: %s", body, env.Error)
		}
	}

	reply(chatBody)
	sent := len(f.writes())
	// A minute on, so the two writes are not a burst, which is a rule of its
	// own and not what this test is about.
	clock.pass(time.Minute)
	reply("🤖 the 2027 register")

	if len(f.writes()) != 2*sent {
		t.Errorf("two different replies sent %d requests in all, want the %d of two writes",
			len(f.writes()), 2*sent)
	}
}

// A held write is not an answer, so nothing is kept and the same call is judged
// again. The person has not seen the card yet, and the rules are about the
// session as it stands rather than about the moment the call first arrived.
//
// It is measured by changing what the rules would say between the two calls: the
// first is held because nobody in this chat has read the target, the read the
// person then asks for settles that, and the same call again goes out. A session
// that had kept the held answer would hand it back and send nothing.
//
// What the second call must not do either is make a second card: that is
// TestARetriedHeldWriteKeepsOneHold, where nothing between the two calls
// changed.
func TestAHeldAnswerIsNotKept(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	f := stubWire(t, replyWire(t))
	stubClock(t, memoryClock)

	ch := callChat(t)
	args := withCode(t, ch.code, chatWriteArgs(chatTitle)["reply"])

	first := heldEnvelopeOf(t, mcpRun(context.Background(), mcpToolNamed(t, "reply"), args, nilWriter{}, ch).Texts)
	if first.Data.Held.Rule != "Focus" {
		t.Fatalf("a reply into a document nobody read was not held by Focus: %+v", first.Data.Held)
	}
	if sent := f.writes(); len(sent) != 0 {
		t.Fatalf("the held write sent %v", sent)
	}

	// The read the rule was waiting for.
	looked(ch, fixtureDocID)

	env := envelopeOf(t, mcpRun(context.Background(), mcpToolNamed(t, "reply"), args, nilWriter{}, ch).Texts)
	if !env.OK {
		t.Fatalf("the same call, judged again against a session that has read the document: %s", env.Error)
	}
	if sent := f.writes(); len(sent) != 1 {
		t.Errorf("the second call sent %d writes, want the one it was judged clear for: %v", len(sent), sent)
	}
}
