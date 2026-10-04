// The tests for what happens around a tool call: the deadline it is given, the
// order the calls run in, a cancellation before and after it starts, and a
// tool that panics.

package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// clock hands out the times given, in order, and repeats the last one. The
// server asks for the time once per tool call, when the line is read, so the
// list reads as one time per call.
type clock struct {
	mu    sync.Mutex
	times []time.Time
	at    int
}

func (c *clock) next() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := c.times[c.at]
	if c.at < len(c.times)-1 {
		c.at++
	}
	return t
}

// callLine is one tools/call for the tool named, with no arguments.
func callLine(id int, name string) string {
	return `{"jsonrpc":"2.0","id":` + strconv.Itoa(id) + `,"method":"tools/call","params":{"name":"` + name + `","arguments":{}}}`
}

// cancelLine is the notification a client sends to take a call back.
func cancelLine(requestID int) string {
	return `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":` + strconv.Itoa(requestID) + `,"reason":"the person changed their mind"}}`
}

// resultTexts reads the texts out of one tool answer, in their order.
func resultTexts(t *testing.T, m map[string]json.RawMessage) []string {
	t.Helper()
	raw, ok := m["result"]
	if !ok {
		t.Fatalf("the answer carries no result: %v", m)
	}
	var got struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	texts := make([]string, 0, len(got.Content))
	for _, item := range got.Content {
		texts = append(texts, item.Text)
	}
	return texts
}

func resultIsError(t *testing.T, m map[string]json.RawMessage) bool {
	t.Helper()
	var got struct {
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(m["result"], &got); err != nil {
		t.Fatal(err)
	}
	return got.IsError
}

// deadlineTool answers with the deadline its context carries, so a test can
// read what the server gave it.
func deadlineTool() Tool {
	return Tool{
		Name:   "deadline",
		Schema: json.RawMessage(`{"type":"object"}`),
		Call: func(ctx context.Context, args json.RawMessage) Result {
			at, ok := ctx.Deadline()
			if !ok {
				return Result{Texts: []string{"none"}, IsError: true}
			}
			return Result{Texts: []string{at.UTC().Format(time.RFC3339Nano)}}
		},
	}
}

func TestEveryCallGetsADeadlineFromTheMomentItsLineWasRead(t *testing.T) {
	first := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	second := first.Add(60 * time.Second)
	s := New(testInfo(), []Tool{deadlineTool()}, nil)
	s.now = (&clock{times: []time.Time{first, second}}).next

	msgs := run(t, s, callLine(1, "deadline"), callLine(2, "deadline"))
	if len(msgs) != 2 {
		t.Fatalf("want two answers, got %d: %v", len(msgs), msgs)
	}
	// The literal is the ceiling the plan sets: 200 seconds, counted from the
	// moment the line was read and not from the moment the call started.
	want := []string{
		first.Add(200 * time.Second).Format(time.RFC3339Nano),
		second.Add(200 * time.Second).Format(time.RFC3339Nano),
	}
	for i, m := range msgs {
		texts := resultTexts(t, m)
		if len(texts) != 1 || texts[0] != want[i] {
			t.Errorf("call %d: deadline %v, want %s", i+1, texts, want[i])
		}
	}
}

func TestCallsRunOneAtATimeInArrivalOrder(t *testing.T) {
	var mu sync.Mutex
	var order []string
	inFlight, most := 0, 0
	counting := Tool{
		Name:   "counting",
		Schema: json.RawMessage(`{"type":"object"}`),
		Call: func(ctx context.Context, args json.RawMessage) Result {
			mu.Lock()
			inFlight++
			if inFlight > most {
				most = inFlight
			}
			mu.Unlock()
			time.Sleep(2 * time.Millisecond)
			mu.Lock()
			order = append(order, string(args))
			inFlight--
			mu.Unlock()
			return Result{Texts: []string{string(args)}}
		},
	}
	lines := make([]string, 0, 4)
	for _, which := range []string{"a", "b", "c", "d"} {
		lines = append(lines, `{"jsonrpc":"2.0","id":"`+which+`","method":"tools/call","params":{"name":"counting","arguments":{"which":"`+which+`"}}}`)
	}
	msgs := run(t, New(testInfo(), []Tool{counting}, nil), lines...)
	if len(msgs) != 4 {
		t.Fatalf("want four answers, got %d", len(msgs))
	}
	mu.Lock()
	defer mu.Unlock()
	if most != 1 {
		t.Errorf("%d calls ran at once; the worker runs one at a time", most)
	}
	for i, which := range []string{"a", "b", "c", "d"} {
		if !strings.Contains(order[i], `"`+which+`"`) {
			t.Errorf("call %d was %q, want the one that named %q; the queue is arrival order", i, order[i], which)
		}
		if string(msgs[i]["id"]) != `"`+which+`"` {
			t.Errorf("answer %d carries id %s, want %q", i, msgs[i]["id"], which)
		}
	}
}

// gated is a tool that waits for the test to let it finish, so a test can hold
// the worker and see what the reader still answers.
func gated(gate chan struct{}, started chan struct{}) Tool {
	return Tool{
		Name:   "gated",
		Schema: json.RawMessage(`{"type":"object"}`),
		Call: func(ctx context.Context, args json.RawMessage) Result {
			select {
			case started <- struct{}{}:
			default:
			}
			select {
			case <-gate:
			case <-time.After(5 * time.Second):
			}
			return Result{Texts: []string{"finished"}}
		},
	}
}

// opener closes a gate exactly once, so a test can both release it and defer
// releasing it without closing a closed channel.
func opener(gate chan struct{}) func() {
	var once sync.Once
	return func() { once.Do(func() { close(gate) }) }
}

func TestPingIsAnsweredWhileACallRuns(t *testing.T) {
	gate, started := make(chan struct{}), make(chan struct{}, 1)
	open := opener(gate)
	defer open()

	s := New(testInfo(), []Tool{gated(gate, started)}, nil)
	c := live(t, s)
	c.send(t, callLine(1, "gated"))
	<-started
	c.send(t, `{"jsonrpc":"2.0","id":2,"method":"ping"}`)
	if got := c.next(t); string(got["id"]) != "2" {
		t.Fatalf("the first line back carries id %s, want the ping answered while the call runs", got["id"])
	}
	open()
	if got := c.next(t); string(got["id"]) != "1" {
		t.Errorf("the call answered with id %s, want 1", got["id"])
	}
}

func TestACancelledCallThatHasNotStartedIsDropped(t *testing.T) {
	gate, started := make(chan struct{}), make(chan struct{}, 1)
	open := opener(gate)
	defer open()

	var mu sync.Mutex
	var ran []string
	counting := Tool{
		Name:   "counting",
		Schema: json.RawMessage(`{"type":"object"}`),
		Call: func(ctx context.Context, args json.RawMessage) Result {
			mu.Lock()
			ran = append(ran, string(args))
			mu.Unlock()
			return Result{Texts: []string{"ran"}}
		},
	}
	s := New(testInfo(), []Tool{gated(gate, started), counting}, nil)
	c := live(t, s)

	c.send(t, callLine(1, "gated"))
	<-started
	c.send(t, callLine(2, "counting"))
	c.send(t, cancelLine(2))
	// The reader reads one line at a time, so an answered ping is the proof
	// that it has already passed the cancellation. Without it the gate could
	// open first and the worker could take the call before it was cancelled.
	c.send(t, `{"jsonrpc":"2.0","id":3,"method":"ping"}`)
	if got := c.next(t); string(got["id"]) != "3" {
		t.Fatalf("the first line back carries id %s, want the ping answered in the reader", got["id"])
	}
	open()

	if got := c.next(t); string(got["id"]) != "1" {
		t.Fatalf("the next answer carries id %s, want the call that was not cancelled", got["id"])
	}
	c.send(t, `{"jsonrpc":"2.0","id":4,"method":"ping"}`)
	if got := c.next(t); string(got["id"]) != "4" {
		t.Errorf("the cancelled call was answered: id %s came back, want 4 next", got["id"])
	}
	mu.Lock()
	defer mu.Unlock()
	if len(ran) != 0 {
		t.Errorf("the cancelled call ran anyway: %v", ran)
	}
}

func TestACancelledCallThatStartedHasItsAnswerDiscarded(t *testing.T) {
	started := make(chan struct{}, 1)
	waiting := Tool{
		Name:   "waiting",
		Schema: json.RawMessage(`{"type":"object"}`),
		Call: func(ctx context.Context, args json.RawMessage) Result {
			started <- struct{}{}
			select {
			case <-ctx.Done():
			case <-time.After(5 * time.Second):
				return Result{Texts: []string{"the context was never cancelled"}, IsError: true}
			}
			return Result{Texts: []string{"stopped where it was"}}
		},
	}
	s := New(testInfo(), []Tool{waiting}, nil)
	c := live(t, s)

	c.send(t, callLine(1, "waiting"))
	<-started
	c.send(t, cancelLine(1))
	c.send(t, `{"jsonrpc":"2.0","id":2,"method":"ping"}`)
	if got := c.next(t); string(got["id"]) != "2" {
		t.Errorf("the cancelled call still answered: id %s came back, want 2", got["id"])
	}
}

func TestAPanicInAToolIsAnErrorResultAndTheServerKeepsRunning(t *testing.T) {
	breaking := Tool{
		Name:   "breaking",
		Schema: json.RawMessage(`{"type":"object"}`),
		Call: func(ctx context.Context, args json.RawMessage) Result {
			panic("the document id was nil")
		},
	}
	var log bytes.Buffer
	msgs := run(t, New(testInfo(), []Tool{breaking, readTool()}, &log),
		callLine(1, "breaking"),
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"read","arguments":{"url":"https://docs.google.com/document/d/abc/edit"}}}`,
	)
	if len(msgs) != 2 {
		t.Fatalf("want two answers, got %d: %v", len(msgs), msgs)
	}
	if _, ok := msgs[0]["error"]; ok {
		t.Fatal("a panic in a tool came back as a protocol error; it is the tool that failed, not the client")
	}
	if !resultIsError(t, msgs[0]) {
		t.Error("the answer to a panicking tool does not say isError")
	}
	texts := resultTexts(t, msgs[0])
	if len(texts) != 1 || !strings.Contains(texts[0], "the document id was nil") {
		t.Errorf("the answer does not name the panic: %v", texts)
	}
	if !strings.Contains(texts[0], "breaking") {
		t.Errorf("the answer does not name the tool: %v", texts)
	}
	if string(msgs[1]["id"]) != "2" {
		t.Errorf("the call after the panic answered with id %s, want 2; the server keeps running", msgs[1]["id"])
	}
	logged := log.String()
	if !strings.Contains(logged, "the document id was nil") || !strings.Contains(logged, "mcp.") {
		t.Errorf("the trace did not reach the log: %q", logged)
	}
}
