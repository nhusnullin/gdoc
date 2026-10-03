// The hold rules: the five questions the binary asks of a chat write before it
// goes out, and the one text it refuses outright.

package chat

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// The names a hold carries, which are what the person reads on the card and
// what a test names. They are words rather than codes because the sentence a
// model repeats to the person is built around them.
const (
	RuleDictated      = "Dictated"
	RuleFocus         = "Focus"
	RuleBurst         = "Burst"
	RuleFlaggedThread = "Flagged thread"
	RuleLargeRemoval  = "Large removal"
)

// The thresholds, each one number in one place.
//
// They are not tuned. They are the specification's values, chosen so that the
// ordinary shape of a review, read a document, reply to a few comments, passes
// without a card, and the shapes that cost something, a stranger's words
// repeated back, a burst of writes, stop for the person.
const (
	dictatedRun       = 12               // words in a row shared with a stranger's comment
	focusWindow       = 30 * time.Minute // how long another document stays in view
	burstWindow       = 60 * time.Second // the window the third write is counted in
	burstWrites       = 3                // the write that trips it
	documentWindow    = time.Hour        // the window writes into one document are counted in
	documentWrites    = 26               // the write into one document that trips it
	largestRemoval    = 300              // characters a propose may take out without a card
	holdIDBytes       = 6                // twelve hex characters, enough for one session's holds
	copiedRunInReason = 12               // words of a copied run the Focus reason names
)

// ErrHiddenChars is the one text that is refused rather than held. There is
// nothing a person could sensibly approve: they cannot see what they would be
// approving. TestHiddenCharactersAreRefusedOutright.
var ErrHiddenChars = errors.New("this text carries characters a person reading it cannot see, so gdoc refuses it rather than asking: say the words plainly instead")

// Write is the one write a rule judges.
//
// It carries no author and no account, because identity is never a gate:
// TestAuthorDomainChangesNoOutcome.
type Write struct {
	Tool     string // reply, annotate or propose
	DocID    string // the document it goes into
	Title    string // that document's title, as the write's own read came back with it
	ThreadID string // the thread a reply goes under, empty for the other two
	Text     string // the words that would be written: a reply body, a why, a replacement
	Removed  int    // characters a propose takes out, 0 for the other two
}

// Hold is one write stopped, with everything the card a person sees is built
// from. Nothing here is a judgement about the person or the commenter: it is
// which rule tripped, the value that tripped it, and the words that would have
// been written.
//
// Args is the call as it arrived, kept so a release posts exactly what was
// held and not a paraphrase of it. Rules leaves it empty; the caller that owns
// the call fills it in.
type Hold struct {
	ID      string
	Tool    string
	DocID   string
	Title   string
	Rule    string
	Value   string
	Reason  string
	Text    string
	Args    json.RawMessage
	Created time.Time
}

// Rules is every hold rule, in the table's order, over one write.
//
// It answers a hold or nothing, and an error only where the text is refused
// outright. The first rule that trips is the one named, because a card naming
// five reasons is a card nobody reads: TestTheFirstRuleThatTripsIsTheOneNamed.
//
// Every question it asks is about this session rather than about the call in
// front of it, which is why it takes the ledger. The clock is handed in, so the
// one clock a session reads is the one cmd/gdoc reads.
func Rules(w Write, l *Ledger, now time.Time) (*Hold, error) {
	if HasHiddenChars(w.Text) {
		return nil, ErrHiddenChars
	}

	for _, rule := range []func(Write, *Ledger, time.Time) (string, string, string){
		dictatedRule, focusRule, burstRule, flaggedThreadRule, largeRemovalRule,
	} {
		name, value, reason := rule(w, l, now)
		if name == "" {
			continue
		}
		id, err := newHoldID()
		if err != nil {
			return nil, err
		}
		return &Hold{
			ID: id, Tool: w.Tool, DocID: w.DocID, Title: w.Title,
			Rule: name, Value: value, Reason: reason, Text: w.Text, Created: now,
		}, nil
	}
	return nil, nil
}

// newHoldID is the id one hold is known by, and the hex in the name of the
// confirm tool it registers.
func newHoldID() (string, error) {
	b := make([]byte, holdIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("the hold id could not be made, so this write is neither sent nor held: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// dictatedRule holds a write that repeats a run of words out of a comment gdoc
// did not write.
//
// This is the shape of the attack the whole chat design is for: a stranger types
// the reply they want into a comment, and the model posts it under the firm's
// name. Twelve words in a row is long enough that an agreement of phrasing is
// not it, and short enough that one sentence of dictation is caught:
// TestTheDictatedRuleTripsOnTwelveWordsInARow.
//
// gdoc's own replies are left out, because a session quoting what it said
// earlier in the same thread is the ordinary way a review reads. Its own means
// the mark and the receipt together: the ledger has to hold the id as one this
// process wrote. The mark alone is a character anybody can type, so skipping
// every marked remark would hand the attack a one-emoji way past the rule:
// TestAMarkedRemarkThisProcessDidNotWriteIsStillAStrangers.
func dictatedRule(w Write, l *Ledger, _ time.Time) (string, string, string) {
	var strangers []string
	for _, said := range l.Remarks(w.DocID) {
		if said.ByGdoc && l.Wrote(said.ID) {
			continue
		}
		strangers = append(strangers, said.Text)
	}
	run := sharedRun(w.Text, strangers, dictatedRun)
	if run == "" {
		return "", "", ""
	}
	return RuleDictated, run, fmt.Sprintf(
		"the text shares %d words in a row with a comment gdoc did not write: %q", dictatedRun, run)
}

// focusRule holds a write made while the session's attention was somewhere else,
// or into a document the model has never read. A write's own read of its target
// is not the model reading it, in either half of the rule: everRead steps over
// such a read, and so does the search for a document read elsewhere. A refused
// write leaves that read behind, and counting it would hold every later write
// with the name of a document nobody in the chat ever saw:
// TestAPinReadOfAnotherDocumentIsNotAttentionElsewhere.
//
// A write into a document nobody looked at is a write nobody can check, and a
// write made minutes after reading somebody else's document is where text
// crosses between them. Copied text is held and never refused, on Nail's call of
// 2026-10-03: quoting the firm's own policy can be the job, so the reason names
// the document the words came from and the words themselves:
// TestTheFocusRuleTripsOnAnotherDocumentOrAnUnreadTarget and
// TestTextCopiedFromAnotherDocumentIsHeldNotRefused.
//
// A write that went into the target resets the window to the target. A held
// write sends nothing and so records nothing, which is what makes the reset mean
// a write the person released.
func focusRule(w Write, l *Ledger, now time.Time) (string, string, string) {
	since := resetPoint(l, w.DocID)
	var elsewhere *Read
	for _, r := range l.Reads() {
		read := r
		if read.DocID == w.DocID || read.DocID == "" || read.ForWrite {
			continue
		}
		if read.At.Before(since) || !read.At.After(now.Add(-focusWindow)) {
			continue
		}
		if elsewhere == nil || read.At.After(elsewhere.At) {
			elsewhere = &read
		}
	}
	if elsewhere != nil {
		title := elsewhere.Title
		if title == "" {
			title = elsewhere.DocID
		}
		reason := fmt.Sprintf(
			"another document was read in this session in the last %d minutes: %q",
			int(focusWindow/time.Minute), title)
		if run := sharedRun(w.Text, documentWordsOf(l, *elsewhere), copiedRunInReason); run != "" {
			reason += fmt.Sprintf(", and this text shares a run of words with it: %q", run)
		}
		return RuleFocus, title, reason
	}

	if !everRead(l, w.DocID) {
		title := w.Title
		if title == "" {
			title = w.DocID
		}
		return RuleFocus, title, fmt.Sprintf(
			"this session has not read %q, so there is nothing to check this text against", title)
	}
	return "", "", ""
}

// resetPoint is the instant the Focus window starts at: the newest write this
// session made into the target document, or the zero time where it made none.
func resetPoint(l *Ledger, docID string) time.Time {
	var at time.Time
	for _, written := range l.Writes() {
		if written.DocID == docID && written.At.After(at) {
			at = written.At
		}
	}
	return at
}

// everRead answers whether this session read the document at all, which is a
// question about the whole session and not about the Focus window.
//
// A write's own read of its target does not count. That read is the binary's,
// made to check the title and to give the other rules the document's words,
// and no part of it reached the chat: counting it would answer this question
// yes for every write, which is the rule never firing at all.
func everRead(l *Ledger, docID string) bool {
	for _, r := range l.Reads() {
		if r.DocID == docID && !r.ForWrite {
			return true
		}
	}
	return false
}

// documentWordsOf is one read's document as the ledger holds it, its own words
// and its comments, for the run of copied words the Focus reason names.
func documentWordsOf(l *Ledger, r Read) []string {
	out := []string{l.Text(r.DocID)}
	for _, said := range l.Remarks(r.DocID) {
		out = append(out, said.Text)
	}
	return out
}

// burstRule holds a write where this session has been writing fast.
//
// A review goes at the speed of a person reading, so a run of writes is either a
// loop or a model working through a list somebody else wrote. Two windows,
// because the two failures look different: a few writes in seconds, and a
// patient one grinding through a document all hour:
// TestTheBurstRuleTripsOnTheThirdWriteAndTheTwentySixth.
func burstRule(w Write, l *Ledger, now time.Time) (string, string, string) {
	writes := l.Writes()

	recent := 0
	for _, written := range writes {
		if written.At.After(now.Add(-burstWindow)) {
			recent++
		}
	}
	if recent+1 >= burstWrites {
		value := fmt.Sprintf("%d writes in %d seconds", recent+1, int(burstWindow/time.Second))
		return RuleBurst, value, "this session is writing fast: " + value
	}

	intoDocument := 0
	for _, written := range writes {
		if written.DocID == w.DocID && written.At.After(now.Add(-documentWindow)) {
			intoDocument++
		}
	}
	if intoDocument+1 >= documentWrites {
		value := fmt.Sprintf("%d writes to this document in an hour", intoDocument+1)
		return RuleBurst, value, "this session has written into this document many times: " + value
	}
	return "", "", ""
}

// flaggedFacts is the four facts on a thread's comment that make a reply into it
// a write the person approves, in the order the reason names them. A link, the
// AI addressed, a robot mark with no receipt behind it, a character nobody can
// see: each is a comment written at the model rather than at a colleague.
var flaggedFacts = []struct {
	name string
	of   func(Facts) bool
}{
	{"has_link", func(f Facts) bool { return f.HasLink }},
	{"names_ai", func(f Facts) bool { return f.NamesAI }},
	{"robot_not_ours", func(f Facts) bool { return f.RobotNotOurs }},
	{"hidden_chars", func(f Facts) bool { return f.HiddenChars }},
}

// flaggedThreadRule holds a reply going under a comment that carries one of
// those four facts: TestTheFlaggedThreadRuleTripsOnAReplyIntoAFlaggedThread.
//
// It is about a reply and nothing else. annotate and propose put words on the
// document rather than under a stranger's, so the thread is not what they
// answer.
func flaggedThreadRule(w Write, l *Ledger, _ time.Time) (string, string, string) {
	if w.Tool != "reply" {
		return "", "", ""
	}
	opener, found := threadOpener(l, w.DocID, w.ThreadID)
	if !found {
		return "", "", ""
	}
	facts := FactsOf(Comment{ID: opener.ID, Text: opener.Text}, l)
	var fired []string
	for _, f := range flaggedFacts {
		if f.of(facts) {
			fired = append(fired, f.name)
		}
	}
	if len(fired) == 0 {
		return "", "", ""
	}
	value := strings.Join(fired, ", ")
	return RuleFlaggedThread, value, fmt.Sprintf(
		"the comment this reply goes under carries %s", value)
}

// threadOpener is the comment a thread starts with, as this session read it.
// Drive gives a thread the id of that comment, so the id is tried first, and the
// thread a remark names is the fallback.
func threadOpener(l *Ledger, docID, threadID string) (Remark, bool) {
	if threadID == "" {
		return Remark{}, false
	}
	said := l.Remarks(docID)
	for _, r := range said {
		if r.ID == threadID {
			return r, true
		}
	}
	for _, r := range said {
		if r.ThreadID == threadID {
			return r, true
		}
	}
	return Remark{}, false
}

// largeRemovalRule holds a propose that takes out more than a paragraph.
//
// A suggestion is reversible and visible, which is why propose is the one write
// that changes a document at all. Taking out three hundred characters is no
// longer a correction, and the person should read what goes:
// TestTheLargeRemovalRuleTripsOverThreeHundredCharacters.
func largeRemovalRule(w Write, _ *Ledger, _ time.Time) (string, string, string) {
	if w.Tool != "propose" || w.Removed <= largestRemoval {
		return "", "", ""
	}
	value := fmt.Sprintf("%d characters", w.Removed)
	return RuleLargeRemoval, value, fmt.Sprintf(
		"this suggestion takes out %s, and anything over %d needs your approval",
		value, largestRemoval)
}

// sharedRun is the first run of n words the text shares with any of the others,
// as the text itself writes it.
//
// Words are matched folded, lowercased and stripped of the punctuation at their
// edges, so a run quoted into the middle of a sentence is still the same run.
// The run returned is the text's own words, because that is what the person
// reads on the card.
func sharedRun(text string, others []string, n int) string {
	mine := strings.Fields(text)
	if n <= 0 || len(mine) < n {
		return ""
	}
	seen := map[string]bool{}
	for _, other := range others {
		words := strings.Fields(other)
		for i := 0; i+n <= len(words); i++ {
			seen[runKey(words[i:i+n])] = true
		}
	}
	for i := 0; i+n <= len(mine); i++ {
		if seen[runKey(mine[i:i+n])] {
			return strings.Join(mine[i:i+n], " ")
		}
	}
	return ""
}

// runKey is one run of words as it is compared.
func runKey(words []string) string {
	folded := make([]string, 0, len(words))
	for _, w := range words {
		folded = append(folded, fold(w))
	}
	return strings.Join(folded, " ")
}

// fold is one word as it is compared: lowercased, without the punctuation at
// either edge. The punctuation inside it stays, because "same-day" and "same
// day" are not the same words.
func fold(word string) string {
	return strings.TrimFunc(strings.ToLower(word), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}
