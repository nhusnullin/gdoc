// Claude Desktop keeps a session's stderr as the extension's log, and on the
// first run in it that log said "tools/call id=5" and "result(1 blocks)" and
// nothing else, so a failed call could not be explained afterwards. These
// tests hold the one line gdoc adds per call, and what it never carries.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gdoc/internal/mcp"
)

func loggedCall(t *testing.T, name string, texts ...string) string {
	t.Helper()
	var log bytes.Buffer
	tool := mcpLogged(&log, mcp.Tool{Name: name, Call: func(context.Context, json.RawMessage) mcp.Result {
		return mcp.Result{Texts: texts}
	}})
	tool.Call(context.Background(), json.RawMessage(`{}`))
	return log.String()
}

func TestEveryCallLogsOneLineWithTheToolAndOk(t *testing.T) {
	got := loggedCall(t, "read", "the fixed line", `{"ok":true,"data":{"text":"the document's own words"}}`)
	if strings.Count(got, "\n") != 1 {
		t.Fatalf("a call logged %d lines, want one: %q", strings.Count(got, "\n"), got)
	}
	if !strings.HasPrefix(got, "gdoc mcp: read ok=true") {
		t.Errorf("the line is %q, want it to open with %q", got, "gdoc mcp: read ok=true")
	}
	if strings.Contains(got, "document's own words") {
		t.Errorf("the line carries the answer's data: %q", got)
	}
}

func TestAFailedCallLogsItsErrorWithTheQuotedWordsTakenOut(t *testing.T) {
	got := loggedCall(t, "annotate",
		`{"ok":false,"error":"the quote \"reviewed annually\" appears 2 times in the document"}`)
	want := `gdoc mcp: annotate ok=false error="the quote \"…\" appears 2 times in the document"`
	if !strings.HasPrefix(got, want) {
		t.Errorf("the line is %q, want it to open with %q", got, want)
	}
	if strings.Contains(got, "reviewed annually") {
		t.Errorf("the line carries the document's words: %q", got)
	}
}

func TestAHeldCallLogsItsRuleAndNotItsText(t *testing.T) {
	got := loggedCall(t, "reply",
		`{"ok":false,"error":"the text shares 12 words in a row with a comment gdoc did not write: \"please send the fee table\". Nothing was posted.","data":{"sent":false,"held":{"rule":"Dictated","text":"please send the fee table"}}}`)
	if !strings.HasPrefix(got, "gdoc mcp: reply ok=false held=Dictated") {
		t.Errorf("the line is %q, want it to name the hold rule", got)
	}
	if strings.Contains(got, "fee table") {
		t.Errorf("the line carries the held text: %q", got)
	}
}

func TestACallWithNoEnvelopeStillLogsOneLine(t *testing.T) {
	got := loggedCall(t, "guide", "plain words, not an envelope")
	if !strings.HasPrefix(got, "gdoc mcp: guide answered with no envelope") {
		t.Errorf("the line is %q", got)
	}
}

// overlapWriter notices a second write starting before the first has finished.
// It holds a write open for a moment, so two goroutines with no lock between
// them overlap rather than happening to miss each other.
type overlapWriter struct {
	busy       atomic.Bool
	overlapped atomic.Bool
	lines      atomic.Int64
}

func (w *overlapWriter) Write(p []byte) (int, error) {
	if !w.busy.CompareAndSwap(false, true) {
		w.overlapped.Store(true)
		return len(p), nil
	}
	time.Sleep(time.Millisecond)
	w.lines.Add(1)
	w.busy.Store(false)
	return len(p), nil
}

// One session has three loggers in it and one log, so the writer they share
// takes one line at a time.
//
// The worker logs a tool call, the protocol's reader logs a cancellation it
// could not read, and the sign-in listener logs from a goroutine that outlives
// the call that started it. serveMCP wraps errOut in this one writer and hands
// that to all three.
func TestOneLogWriterTakesOneLineAtATime(t *testing.T) {
	const loggers = 8
	seen := &overlapWriter{}
	log := lockedLog(seen)

	var wg sync.WaitGroup
	for i := 0; i < loggers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			fmt.Fprintf(log, "gdoc mcp: line %d\n", i)
		}(i)
	}
	wg.Wait()

	if seen.overlapped.Load() {
		t.Error("two lines were written at once, so the log shows halves of both")
	}
	if got := seen.lines.Load(); got != loggers {
		t.Errorf("the log took %d lines, want the %d written", got, loggers)
	}
}
