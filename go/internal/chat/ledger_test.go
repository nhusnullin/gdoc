package chat

import (
	"go/parser"
	"go/token"
	"sync"
	"testing"
	"time"
)

// A *Ledger is what the facts ask about a robot mark, so the stub of Task 11 is
// gone and the record is the real one.
var _ OwnReplies = (*Ledger)(nil)

// read is the instant the fixtures below are read at, written out rather than
// taken from the clock: what the ledger keeps is the time it was handed, and a
// test that read the clock would hold nothing.
var ledgerAt = time.Date(2026, 10, 3, 11, 0, 0, 0, time.UTC)

// Every read is kept with the document it was of, the title it came back with,
// the instant it was made at, and the words of every comment and reply in it,
// gdoc's own marked as gdoc's.
func TestEveryReadIsRecordedWithItsTitleTimeAndOthersTexts(t *testing.T) {
	l := NewLedger()
	l.RecordRead(Read{
		DocID: "D1", Title: "Supplier register policy", At: ledgerAt,
		Remarks: []Remark{
			{ID: "C1", ThreadID: "C1", Text: "ai? which register is this"},
			{ID: "R1", ThreadID: "C1", Text: "🤖 the 2026 register", ByGdoc: true},
		},
	})
	l.RecordRead(Read{DocID: "D2", Title: "Quarterly board pack", At: ledgerAt.Add(time.Minute)})

	kept := l.Reads()
	if len(kept) != 2 {
		t.Fatalf("the ledger kept %d reads, want 2", len(kept))
	}
	if kept[0].DocID != "D1" || kept[0].Title != "Supplier register policy" {
		t.Errorf("the first read is %q %q, want D1 and the register policy", kept[0].DocID, kept[0].Title)
	}
	if !kept[0].At.Equal(ledgerAt) {
		t.Errorf("the first read is at %s, want %s", kept[0].At, ledgerAt)
	}
	if !kept[1].At.Equal(ledgerAt.Add(time.Minute)) {
		t.Errorf("the second read is at %s, want a minute later", kept[1].At)
	}

	said := l.Remarks("D1")
	if len(said) != 2 {
		t.Fatalf("the ledger kept %d remarks of D1, want 2", len(said))
	}
	if said[0].Text != "ai? which register is this" || said[0].ByGdoc {
		t.Errorf("the comment is kept as %+v, want a stranger's words", said[0])
	}
	if said[1].Text != "🤖 the 2026 register" || !said[1].ByGdoc {
		t.Errorf("the reply is kept as %+v, want gdoc's own", said[1])
	}
	if said[1].ThreadID != "C1" {
		t.Errorf("the reply sits in thread %q, want C1", said[1].ThreadID)
	}
	if other := l.Remarks("D2"); len(other) != 0 {
		t.Errorf("D2's read carried no comments and the ledger kept %d", len(other))
	}

	// What comes back is a copy. A rule walking the ledger cannot change what
	// the next rule reads.
	kept[0].Remarks[0].Text = "rewritten"
	if l.Remarks("D1")[0].Text != "ai? which register is this" {
		t.Error("a caller rewrote what the ledger kept")
	}
}

// Every write is kept with the document and the tool, and the ids of what gdoc
// wrote are kept beside them, because that is the only record of authorship
// there is.
func TestEveryWriteAndOwnReplyIsRecorded(t *testing.T) {
	l := NewLedger()
	l.RecordWrite(Written{DocID: "D1", Tool: "reply", At: ledgerAt})
	l.RecordOwn("R1", "")
	l.RecordWrite(Written{DocID: "D1", Tool: "annotate", At: ledgerAt.Add(time.Second)})
	l.RecordOwn("C9")

	kept := l.Writes()
	if len(kept) != 2 {
		t.Fatalf("the ledger kept %d writes, want 2", len(kept))
	}
	if kept[0].Tool != "reply" || kept[0].DocID != "D1" || !kept[0].At.Equal(ledgerAt) {
		t.Errorf("the first write is kept as %+v", kept[0])
	}
	if kept[1].Tool != "annotate" {
		t.Errorf("the second write is kept as %+v, want annotate", kept[1])
	}
	for _, id := range []string{"R1", "C9"} {
		if !l.Wrote(id) {
			t.Errorf("the ledger does not know it wrote %s", id)
		}
	}
	if l.Wrote("R2") {
		t.Error("the ledger claims a reply nobody here wrote")
	}
	// An id nobody gave is nobody's: an empty string is not a write.
	if l.Wrote("") {
		t.Error("the empty id is somebody's write")
	}

	kept[0].Tool = "propose"
	if l.Writes()[0].Tool != "reply" {
		t.Error("a caller rewrote the writes the ledger kept")
	}
}

// The ledger is this process and nothing else: two of them share nothing, and
// the file that holds one names no way to reach a disk.
func TestTheLedgerIsPerProcessAndWritesNothingToDisk(t *testing.T) {
	first, second := NewLedger(), NewLedger()
	first.RecordRead(Read{DocID: "D1", Title: "Supplier register policy", At: ledgerAt})
	first.RecordWrite(Written{DocID: "D1", Tool: "reply", At: ledgerAt})
	first.RecordOwn("R1")

	if len(second.Reads()) != 0 || len(second.Writes()) != 0 || second.Wrote("R1") {
		t.Error("the second ledger knows what the first was told")
	}
	// A ledger nobody made is a process that has recorded nothing, which is the
	// honest state of one that has just started.
	var none *Ledger
	none.RecordRead(Read{DocID: "D1", At: ledgerAt})
	none.RecordWrite(Written{DocID: "D1", Tool: "reply", At: ledgerAt})
	none.RecordOwn("R1")
	if none.Wrote("R1") || len(none.Reads()) != 0 || len(none.Writes()) != 0 || none.Text("D1") != "" {
		t.Error("a nil ledger answered as though it had kept something")
	}

	// Nothing in the ledger reaches a disk, and the way that is held is the
	// imports of the file itself: a package that cannot name os or a path
	// cannot write one. TestTheLedgerIsPerProcessAndWritesNothingToDisk.
	allowed := map[string]bool{`"sync"`: true, `"time"`: true}
	file, err := parser.ParseFile(token.NewFileSet(), "ledger.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range file.Imports {
		if !allowed[imp.Path.Value] {
			t.Errorf("ledger.go imports %s, and the ledger writes nothing to disk", imp.Path.Value)
		}
	}
}

// Two goroutines recording at once is what a session does: the worker runs the
// calls and the listener answers beside it. The race detector is the assertion.
func TestTheLedgerTakesTwoRecordersAtOnce(t *testing.T) {
	l := NewLedger()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			l.RecordRead(Read{DocID: "D1", Title: "Supplier register policy", At: ledgerAt,
				Remarks: []Remark{{ID: "C1", ThreadID: "C1", Text: "which register"}}})
			l.RecordWrite(Written{DocID: "D1", Tool: "reply", At: ledgerAt})
			l.RecordOwn("R1")
			_ = l.Remarks("D1")
			_ = l.Wrote("R1")
		}(i)
	}
	wg.Wait()
	if len(l.Reads()) != 4 || len(l.Writes()) != 4 {
		t.Errorf("the ledger kept %d reads and %d writes, want 4 of each", len(l.Reads()), len(l.Writes()))
	}
}

// The robot mark with no receipt is the one fact that is not about the text
// alone, and the ledger is what answers it now.
func TestRobotNotOursUsesTheLedger(t *testing.T) {
	const text = "🤖 the 2026 register"
	l := NewLedger()
	if !FactsOf(Comment{ID: "R1", Text: text}, l).RobotNotOurs {
		t.Error("a mark is ours before this process wrote anything")
	}
	l.RecordOwn("R1")
	if FactsOf(Comment{ID: "R1", Text: text}, l).RobotNotOurs {
		t.Error("the reply this process wrote is reported as somebody else's")
	}
	if !FactsOf(Comment{ID: "R2", Text: text}, l).RobotNotOurs {
		t.Error("a marked reply this process never wrote is reported as ours")
	}
}

// A read keeps the document's own words, and a later read that fetched none does
// not take them away: the Focus rule names the run a write copies out of what a
// document says, and the comments read that follows a text read carries no text.
func TestAReadKeepsTheDocumentsBodyText(t *testing.T) {
	const body = "The supplier register is reviewed annually.\nSee example.net for the list."
	l := NewLedger()
	l.RecordRead(Read{DocID: "D1", Title: "Supplier register policy", At: ledgerAt, Text: body})
	l.RecordRead(Read{DocID: "D1", Title: "Supplier register policy", At: ledgerAt.Add(time.Minute),
		Remarks: []Remark{{ID: "C1", ThreadID: "C1", Text: "which register"}}})

	if got := l.Text("D1"); got != body {
		t.Errorf("the ledger holds %q as D1's text, want the words the read fetched", got)
	}
	if got := l.Text("D2"); got != "" {
		t.Errorf("the ledger holds %q as the text of a document nobody read", got)
	}
	// A later read that fetched the text again replaces it, because the document
	// may have changed between the two.
	l.RecordRead(Read{DocID: "D1", Title: "Supplier register policy", At: ledgerAt.Add(2 * time.Minute),
		Text: "The supplier register is reviewed quarterly."})
	if got := l.Text("D1"); got != "The supplier register is reviewed quarterly." {
		t.Errorf("the ledger holds %q, want the newest words", got)
	}
	// And the comment from the middle read is still there: a remark is kept once
	// by its id, however many reads carried it.
	l.RecordRead(Read{DocID: "D1", At: ledgerAt.Add(3 * time.Minute),
		Remarks: []Remark{{ID: "C1", ThreadID: "C1", Text: "which register, the 2026 one?"}}})
	said := l.Remarks("D1")
	if len(said) != 1 {
		t.Fatalf("the ledger kept %d remarks of D1, want one: %+v", len(said), said)
	}
	if said[0].Text != "which register, the 2026 one?" {
		t.Errorf("the remark is %q, want what the newest read of it said", said[0].Text)
	}
}
