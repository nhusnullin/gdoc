package chat

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"sync"
	"testing"
	"time"
)

// memoryAt is the instant these tests remember at.
var memoryAt = time.Date(2026, 10, 3, 13, 0, 0, 0, time.UTC)

// replyArgs is one call of one write tool, as a model sends it.
const replyArgs = `{"url":"1AbC","title":"Supplier register policy","comment_id":"AAAA1111","body":"🤖 the 2026 register"}`

// The answer a write gave is the answer the same write gets again. What is
// remembered is the tool and the arguments together, so the same arguments under
// another tool are another write.
func TestAKeptAnswerIsRecalledForTheSameCall(t *testing.T) {
	m := NewMemory[string]()
	m.Keep("reply", json.RawMessage(replyArgs), "posted R2", memoryAt)

	got, ok := m.Recall("reply", json.RawMessage(replyArgs), memoryAt.Add(time.Minute))
	if !ok {
		t.Fatal("the same write was not recalled")
	}
	if got != "posted R2" {
		t.Errorf("the memory answered %q, want the answer the write gave", got)
	}
	if _, ok := m.Recall("annotate", json.RawMessage(replyArgs), memoryAt); ok {
		t.Error("the same arguments under another tool were recalled as the same write")
	}
	if _, ok := m.Recall("reply", json.RawMessage(`{"url":"1AbC"}`), memoryAt); ok {
		t.Error("another call was recalled as this one")
	}
}

// A write whose properties arrive in another order, or with other spacing, is
// the same write. A model writing the same call twice does not write the same
// bytes twice, and what the memory is for is the call and not the bytes.
func TestArgumentsInAnotherOrderAreTheSameWrite(t *testing.T) {
	m := NewMemory[string]()
	m.Keep("reply", json.RawMessage(replyArgs), "posted R2", memoryAt)

	const reordered = `{ "body":"🤖 the 2026 register", "comment_id":"AAAA1111",
		"title":"Supplier register policy", "url":"1AbC" }`
	if _, ok := m.Recall("reply", json.RawMessage(reordered), memoryAt); !ok {
		t.Error("the same call written in another order was not recalled")
	}
	// A call the server cannot decode is remembered as it arrived, so nothing
	// here depends on a shape it never read.
	m.Keep("reply", json.RawMessage(`not json`), "refused", memoryAt)
	if _, ok := m.Recall("reply", json.RawMessage(`not json`), memoryAt); !ok {
		t.Error("arguments that would not decode were not recalled by their own bytes")
	}
}

// Ten minutes, and then the write is a new one. The number is the
// specification's, stated here as its own literal.
func TestNothingIsRecalledAfterTenMinutes(t *testing.T) {
	m := NewMemory[string]()
	m.Keep("reply", json.RawMessage(replyArgs), "posted R2", memoryAt)

	if _, ok := m.Recall("reply", json.RawMessage(replyArgs), memoryAt.Add(9*time.Minute+59*time.Second)); !ok {
		t.Error("the answer was forgotten before ten minutes")
	}
	if _, ok := m.Recall("reply", json.RawMessage(replyArgs), memoryAt.Add(10*time.Minute)); ok {
		t.Error("the answer outlived ten minutes")
	}
}

// The memory is one session's own and nothing in it reaches a disk, which is the
// rule a grant lives by as well. A memory nobody made remembers nothing, which
// is the honest state of a session that has just started.
func TestTheMemoryIsPerProcessAndWritesNothingToDisk(t *testing.T) {
	first, second := NewMemory[string](), NewMemory[string]()
	first.Keep("reply", json.RawMessage(replyArgs), "posted R2", memoryAt)
	if _, ok := second.Recall("reply", json.RawMessage(replyArgs), memoryAt); ok {
		t.Error("the second memory knows what the first was told")
	}

	var none *Memory[string]
	none.Keep("reply", json.RawMessage(replyArgs), "posted R2", memoryAt)
	if _, ok := none.Recall("reply", json.RawMessage(replyArgs), memoryAt); ok {
		t.Error("a nil memory answered as though it had kept something")
	}

	// The way that is held is the imports of the file itself: a package that
	// cannot name os or a path cannot write one.
	allowed := map[string]bool{
		`"crypto/sha256"`: true, `"encoding/hex"`: true, `"encoding/json"`: true,
		`"sync"`: true, `"time"`: true,
	}
	file, err := parser.ParseFile(token.NewFileSet(), "memory.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range file.Imports {
		if !allowed[imp.Path.Value] {
			t.Errorf("memory.go imports %s, and the memory writes nothing to disk", imp.Path.Value)
		}
	}
}

// Two goroutines at once is what a session does: the worker runs the calls and
// the login listener answers beside it. The race detector is the assertion.
func TestTheMemoryTakesTwoCallersAtOnce(t *testing.T) {
	m := NewMemory[string]()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Keep("reply", json.RawMessage(replyArgs), "posted R2", memoryAt)
			_, _ = m.Recall("reply", json.RawMessage(replyArgs), memoryAt)
		}()
	}
	wg.Wait()
}
