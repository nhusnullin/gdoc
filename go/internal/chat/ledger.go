// The ledger: what this process read and what it wrote, kept in memory for the
// one thing it is for, which is holding or refusing a write.

package chat

import (
	"sync"
	"time"
)

// Remark is one comment or reply the ledger kept: its id, the thread it sits in,
// what it says, and whether gdoc wrote it.
//
// ByGdoc is the robot prefix on the text and never the account that posted it,
// because identity is never a gate: internal/comments decides it and the ledger
// carries the answer through.
type Remark struct {
	ID       string
	ThreadID string
	Text     string
	ByGdoc   bool
}

// Read is one read this process made of one document: which document, the title
// it came back with, when it was made, the document's own words where the read
// fetched them, and every comment and reply it carried.
//
// A read that fetched no text leaves Text empty, which is what a comment listing
// is, and a read that fetched no comments leaves Remarks empty, which is what a
// text read is. The ledger joins them: Text and Remarks answer for the document
// and not for one read of it.
//
// ForWrite marks the read a write makes of its own target before it goes out.
// Its remarks are the ones the Dictated rule asks about, so it is kept like any
// other read, but it is the binary reading and never the model looking: nobody
// in the chat saw a word of it. Both halves of the Focus rule therefore step
// over it. "Has this session read the target" does, or a write into a document
// the model never opened would answer that question with its own read:
// TestAWriteIntoAnUnreadTargetIsHeldThoughItsOwnPinReadIt. "Was another
// document read" does too, or a refused write would leave a document behind
// that holds every later one:
// TestAPinReadOfAnotherDocumentIsNotAttentionElsewhere.
type Read struct {
	DocID    string
	Title    string
	At       time.Time
	Text     string
	Remarks  []Remark
	ForWrite bool
}

// Written is one write this process made: which document, which tool, and when.
// What was written is not here, because the rules that read this ask how often
// and how recently and never what about.
type Written struct {
	DocID string
	Tool  string
	At    time.Time
}

// Ledger is this process's record of what it read and what it wrote.
//
// It exists because every hold rule is a question about this session and not
// about the document: whether a run of words came out of a comment a stranger
// left, whether the model has been reading another document, how many writes
// have gone into this one in the last minute, how much of the document one
// proposal takes out. None of those can be answered by the call in front of it.
//
// It is per process and nothing is written to disk:
// TestTheLedgerIsPerProcessAndWritesNothingToDisk. A session that ends takes its
// record with it, which is the same rule a grant lives by. Every method is safe
// on a nil Ledger, which is a process that has recorded nothing, and safe from
// two goroutines at once, because the login listener answers beside the worker:
// TestTheLedgerTakesTwoRecordersAtOnce.
type Ledger struct {
	mu     sync.Mutex
	reads  []Read
	writes []Written
	own    map[string]bool
}

// NewLedger is the record one session keeps.
func NewLedger() *Ledger {
	return &Ledger{own: map[string]bool{}}
}

// RecordRead keeps one read, as the caller read it. The time is handed in rather
// than taken from the clock here, so the one clock a session reads is the one
// cmd/gdoc reads: TestEveryReadIsRecordedWithItsTitleTimeAndOthersTexts.
//
// The remarks are copied, so the slice the caller built is not the slice the
// ledger holds.
func (l *Ledger) RecordRead(r Read) {
	if l == nil {
		return
	}
	r.Remarks = append([]Remark(nil), r.Remarks...)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reads = append(l.reads, r)
}

// RecordWrite keeps one write: TestEveryWriteAndOwnReplyIsRecorded.
func (l *Ledger) RecordWrite(w Written) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writes = append(l.writes, w)
}

// RecordOwn keeps the ids of what gdoc wrote: the reply a reply made, the
// comment an annotate or a propose made. An empty id is dropped, because a write
// whose answer carried no id is a write nothing can be matched against later.
func (l *Ledger) RecordOwn(ids ...string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.own == nil {
		l.own = map[string]bool{}
	}
	for _, id := range ids {
		if id != "" {
			l.own[id] = true
		}
	}
}

// Wrote answers whether this process wrote the comment or reply with this id,
// which is what the robot_not_ours fact asks: TestRobotNotOursUsesTheLedger.
func (l *Ledger) Wrote(id string) bool {
	if l == nil || id == "" {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.own[id]
}

// Reads is every read this process made, oldest first, as copies.
func (l *Ledger) Reads() []Read {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Read, 0, len(l.reads))
	for _, r := range l.reads {
		r.Remarks = append([]Remark(nil), r.Remarks...)
		out = append(out, r)
	}
	return out
}

// Writes is every write this process made, oldest first, as copies.
func (l *Ledger) Writes() []Written {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Written(nil), l.writes...)
}

// Text is the document's own words, as the newest read that fetched them gave
// them. A comment listing fetches none, and it does not take away what a text
// read fetched: TestAReadKeepsTheDocumentsBodyText.
func (l *Ledger) Text(docID string) string {
	if l == nil {
		return ""
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := len(l.reads) - 1; i >= 0; i-- {
		if l.reads[i].DocID == docID && l.reads[i].Text != "" {
			return l.reads[i].Text
		}
	}
	return ""
}

// Remarks is every comment and reply this process has read in one document, in
// the order it first saw them, each as the newest read of it said it.
func (l *Ledger) Remarks(docID string) []Remark {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []Remark
	at := map[string]int{}
	for _, r := range l.reads {
		if r.DocID != docID {
			continue
		}
		for _, said := range r.Remarks {
			if i, seen := at[said.ID]; seen {
				out[i] = said
				continue
			}
			at[said.ID] = len(out)
			out = append(out, said)
		}
	}
	return out
}
