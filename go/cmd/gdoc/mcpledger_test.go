package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ledgerClock is the instant these tests read the clock at, written out so the
// time the ledger kept is a value a test states rather than one it parses back.
var ledgerClock = time.Date(2026, 10, 3, 11, 0, 0, 0, time.UTC)

// Every read a tool makes is kept: which document, the title it came back with,
// when it was made, and the words of the document and of every comment and reply
// in it. The rules of tasks 14 to 17 ask the ledger and never the call.
func TestEachReadToolRecordsItsReadInTheLedger(t *testing.T) {
	for _, tool := range readTools {
		t.Run(tool, func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
			signedIn(t)
			stubSession(t, docsAndComments(t))
			stubNow(t, ledgerClock)

			ch := callChat(t)
			res := mcpRun(context.Background(), mcpToolNamed(t, tool),
				withCode(t, ch.code, `{"url":"`+fixtureDocID+`"}`), nilWriter{}, ch)
			if res.IsError {
				t.Fatalf("%s answered an error: %v", tool, res.Texts)
			}

			reads := ch.ledger.Reads()
			if len(reads) != 1 {
				t.Fatalf("%s left %d reads in the ledger, want one", tool, len(reads))
			}
			if reads[0].DocID != fixtureDocID {
				t.Errorf("the read is of %q, want %q", reads[0].DocID, fixtureDocID)
			}
			if !reads[0].At.Equal(ledgerClock) {
				t.Errorf("the read is at %s, want %s", reads[0].At, ledgerClock)
			}
		})
	}
}

// The text read keeps the document's own words, and the comments read keeps every
// comment and reply with gdoc's own marked as gdoc's. Which read carries which is
// the point: the Link rule asks what the document already says, and the Dictated
// rule asks what a stranger wrote in it.
func TestAReadKeepsTheWordsItsOwnToolFetched(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	stubSession(t, docsAndComments(t))
	stubNow(t, ledgerClock)
	ch := callChat(t)

	if res := mcpRun(context.Background(), mcpToolNamed(t, "read"),
		withCode(t, ch.code, `{"url":"`+fixtureDocID+`"}`), nilWriter{}, ch); res.IsError {
		t.Fatalf("read answered an error: %v", res.Texts)
	}
	if got := ch.ledger.Text(fixtureDocID); !strings.Contains(got, "The supplier register is reviewed") {
		t.Errorf("the ledger holds %q as the document's text", got)
	}
	if reads := ch.ledger.Reads(); reads[0].Title != chatTitle {
		t.Errorf("the read came back titled %q, want %q", reads[0].Title, chatTitle)
	}
	if said := ch.ledger.Remarks(fixtureDocID); len(said) != 0 {
		t.Errorf("the text read fetched no comments and the ledger kept %d", len(said))
	}

	stubSession(t, docsAndComments(t))
	if res := mcpRun(context.Background(), mcpToolNamed(t, "comments"),
		withCode(t, ch.code, `{"url":"`+fixtureDocID+`"}`), nilWriter{}, ch); res.IsError {
		t.Fatalf("comments answered an error: %v", res.Texts)
	}
	said := ch.ledger.Remarks(fixtureDocID)
	if len(said) != 3 {
		t.Fatalf("the ledger kept %d remarks, want the two comments and the one reply: %+v", len(said), said)
	}
	byID := map[string]bool{}
	for _, one := range said {
		byID[one.ID] = one.ByGdoc
		if one.Text == "" {
			t.Errorf("remark %s was kept with no words", one.ID)
		}
	}
	for id, want := range map[string]bool{"AAAA1111": false, "BBBB2222": false, "R1": true} {
		got, kept := byID[id]
		if !kept {
			t.Errorf("the ledger kept nothing for %s", id)
			continue
		}
		if got != want {
			t.Errorf("%s is kept as by_gdoc = %v, want %v", id, got, want)
		}
	}
	for _, one := range said {
		if one.ID == "R1" && one.ThreadID != "AAAA1111" {
			t.Errorf("the reply sits in thread %q, want AAAA1111", one.ThreadID)
		}
	}
	// The comments read fetched no text of its own, and it did not take away what
	// the text read fetched.
	if got := ch.ledger.Text(fixtureDocID); !strings.Contains(got, "The supplier register is reviewed") {
		t.Errorf("the comments read took the document's text away: %q", got)
	}
}

// A write reads its target on the call, and that read is recorded: the Link rule
// asks whether a link in the write was already in the document, and a write whose
// target was never read would have nothing to ask.
func TestAWriteReadsItsTargetFreshAndRecordsIt(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	stubWire(t, &fakeWire{answers: append(pinAnswers(t), &answer{
		method: "POST", match: "/comments/AAAA1111/replies",
		json: `{"id":"R2","createdTime":"2026-09-06T10:45:00Z","content":"` + chatBody + `"}`,
	})})
	stubNow(t, ledgerClock)

	ch := callChat(t)
	looked(ch, fixtureDocID)
	args := `{"url":"` + fixtureDocID + `","title":"` + chatTitle + `",` +
		`"comment_id":"AAAA1111","thread_quote":"` + chatQuote + `","body":"` + chatBody + `"}`
	res := mcpRun(context.Background(), mcpToolNamed(t, "reply"), withCode(t, ch.code, args), nilWriter{}, ch)
	if env := envelopeOf(t, res.Texts); !env.OK {
		t.Fatalf("the reply was refused: %q", env.Error)
	}

	// The read the pin made, with the document's own words and the thread it is
	// about to write into.
	if len(ch.ledger.Reads()) == 0 {
		t.Fatal("the write made no read the ledger kept")
	}
	if got := ch.ledger.Text(fixtureDocID); !strings.Contains(got, "The supplier register is reviewed") {
		t.Errorf("the write's own read kept %q as the document's text", got)
	}
	said := ch.ledger.Remarks(fixtureDocID)
	if len(said) == 0 {
		t.Fatal("the write's own read kept no comment of the thread it wrote into")
	}
	var found bool
	for _, one := range said {
		if one.ID == "AAAA1111" && strings.Contains(one.Text, "register") {
			found = true
		}
	}
	if !found {
		t.Errorf("the thread's own comment is not in the ledger: %+v", said)
	}

	// And the write itself, with the tool that made it and the reply it wrote.
	writes := ch.ledger.Writes()
	if len(writes) != 1 {
		t.Fatalf("the ledger kept %d writes, want one", len(writes))
	}
	if writes[0].Tool != "reply" || writes[0].DocID != fixtureDocID {
		t.Errorf("the write is kept as %+v", writes[0])
	}
	if !writes[0].At.Equal(ledgerClock) {
		t.Errorf("the write is at %s, want %s", writes[0].At, ledgerClock)
	}
	if !ch.ledger.Wrote("R2") {
		t.Error("the ledger does not know gdoc wrote R2")
	}
}

// The reply this session wrote is not a mark with no receipt, and the one a
// stranger left is. It is the whole reason the ledger is wired to the view: the
// marker is the only record of authorship there is, and the receipt is this
// process's own.
func TestAReplyThisSessionWroteIsNotRobotNotOurs(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	stubSession(t, docsAndComments(t))
	stubNow(t, ledgerClock)
	ch := callChat(t)

	ours := func() bool {
		t.Helper()
		stubSession(t, docsAndComments(t))
		res := mcpRun(context.Background(), mcpToolNamed(t, "comments"),
			withCode(t, ch.code, `{"url":"`+fixtureDocID+`"}`), nilWriter{}, ch)
		if res.IsError {
			t.Fatalf("comments answered an error: %v", res.Texts)
		}
		view := chatViewOf(t, res.Texts[2])
		for _, thread := range view.Threads {
			for _, reply := range thread.Replies {
				if reply.ID == "R1" {
					flagged, _ := reply.Facts["robot_not_ours"].(bool)
					return !flagged
				}
			}
		}
		t.Fatal("the wrapped copy carries no reply R1")
		return false
	}

	if ours() {
		t.Error("a robot reply this session never wrote is reported as ours")
	}
	ch.ledger.RecordOwn("R1")
	if !ours() {
		t.Error("the reply this session wrote is still reported as somebody else's")
	}
}
