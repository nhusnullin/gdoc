// The write memory: the answer each write of this session gave, kept for ten
// minutes, so the same write asked twice is written once.

package chat

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// memoryLife is how long one write's answer is kept.
//
// Ten minutes is the specification's number. It is longer than any client
// timeout, which is what this is for: a client that gave up on a call and sent
// it again has no way of knowing whether the first one reached the document, and
// the only answer that is true for both is the first call's own. It is shorter
// than the thirty minutes a hold lives, because a write nobody retried in ten
// minutes is a write the model has moved on from.
const memoryLife = 10 * time.Minute

// Memory is the answers this session's writes gave, under the call each came
// from.
//
// It exists because a retry is indistinguishable from a second write: the same
// tool, the same arguments, arriving again. Asking the document would not settle
// it either, since a reply posted twice is two replies and both are there. So
// the session remembers what it answered, and answers that again.
//
// T is whatever the caller calls an answer: cmd/gdoc keeps the one result a tool
// call returns, so a retry gets the first call's answer byte for byte and
// nothing about it says it is a second call. Anything else would tell the model
// a write happened that did not.
//
// It is per process and nothing reaches disk, like the ledger beside it:
// TestTheMemoryIsPerProcessAndWritesNothingToDisk. Every method is safe on a nil
// Memory, which is a session that remembers nothing, and safe from two
// goroutines at once: TestTheMemoryTakesTwoCallersAtOnce.
type Memory[T any] struct {
	mu   sync.Mutex
	kept map[string]remembered[T]
}

// remembered is one answer and when it was given.
type remembered[T any] struct {
	answer T
	at     time.Time
}

// NewMemory is one session's own.
func NewMemory[T any]() *Memory[T] {
	return &Memory[T]{kept: map[string]remembered[T]{}}
}

// Recall is the answer this session already gave to this call, where it gave one
// inside the last ten minutes: TestAKeptAnswerIsRecalledForTheSameCall.
//
// The clock is handed in, as it is everywhere else here, so the one clock a
// session reads is the one cmd/gdoc reads.
func (m *Memory[T]) Recall(tool string, args json.RawMessage, now time.Time) (T, bool) {
	var zero T
	if m == nil {
		return zero, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forget(now)
	one, ok := m.kept[WriteKey(tool, args)]
	if !ok {
		return zero, false
	}
	return one.answer, true
}

// Keep remembers the answer one write gave. The caller keeps only an answer a
// write really made: a held write is not kept, because the person has not
// answered the card yet and the same call again has to be judged again, which is
// TestAHeldAnswerIsNotKept in cmd/gdoc.
func (m *Memory[T]) Keep(tool string, args json.RawMessage, answer T, now time.Time) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.kept == nil {
		m.kept = map[string]remembered[T]{}
	}
	m.forget(now)
	m.kept[WriteKey(tool, args)] = remembered[T]{answer: answer, at: now}
}

// forget drops every answer older than memoryLife. It runs under the lock, on
// every call either way, so a session that ran for an afternoon holds the last
// ten minutes of it and no more.
func (m *Memory[T]) forget(now time.Time) {
	for key, one := range m.kept {
		if !one.at.After(now.Add(-memoryLife)) {
			delete(m.kept, key)
		}
	}
}

// WriteKey is the tool and its arguments as one short string: the identity of
// one write, which is what "the same write twice" is asked against.
//
// The arguments are canonicalised before they are hashed, by decoding and
// encoding them again, which sorts the properties and drops the spacing. So the
// same write with its properties in another order is the same write, which is
// what it is: TestArgumentsInAnotherOrderAreTheSameWrite. Arguments that will
// not decode are hashed as they arrived, because a call the server could not
// read is a call it cannot say anything about.
//
// It is a hash and not the arguments themselves because a propose carries
// paragraphs, and a session holding ten minutes of them would hold them twice.
//
// It is exported because two rooms ask the same question of it, this memory and
// cmd/gdoc's record of the holds a session is keeping, and two readings of
// "the same write" would disagree about the order of the properties: one of them
// would then make a second card for a write the other had already held.
// TestARetriedHeldWriteKeepsOneHold in cmd/gdoc is the pin on that use.
func WriteKey(tool string, args json.RawMessage) string {
	canon := []byte(args)
	var shape any
	if err := json.Unmarshal(args, &shape); err == nil {
		if again, err := json.Marshal(shape); err == nil {
			canon = again
		}
	}
	sum := sha256.Sum256(append([]byte(tool+"\x00"), canon...))
	return hex.EncodeToString(sum[:])
}
