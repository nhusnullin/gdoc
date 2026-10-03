// One whole session, end to end through the wiring: the handshake, the guide,
// the three reads, the three writes, one of them released through its card, the
// sign-in, a call with no code, a command that panics, and stdin closing.
//
// Every other mcp test here calls one tool's own function. This one drives the
// server the way a client does, over a pipe, so what it holds is the wiring
// between the pieces: that a code guide handed out reaches a read, that a hold
// made in one call is released by a tool registered in another, that a panic
// under a command is one error answer and the next call still works, and that
// the only thing on stdout is JSON-RPC.
//
// Nothing here reaches Google or GitHub. The wire is the package's own fake and
// the build names no release, so the daily check never asks.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"gdoc/internal/chat"
	"gdoc/internal/gapi"
)

// panicDocID is the document whose read panics. It is a document id in shape, so
// the panic happens under the command rather than in the argument reading.
const panicDocID = "1PaNiC0000000000000000000000000000000000"

// sessionClock is the session's clock, moved by the test while the server reads
// it from its own goroutine. movingClock of mcprelease_test is the same idea
// without the lock, which is enough where the calls and the clock are in one
// goroutine and not enough here.
type sessionClock struct {
	mu sync.Mutex
	at time.Time
}

func stubSessionClock(t *testing.T, at time.Time) *sessionClock {
	t.Helper()
	c := &sessionClock{at: at}
	old := now
	now = func() time.Time {
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.at
	}
	t.Cleanup(func() { now = old })
	return c
}

// pass moves the clock on, which is what an ended turn looks like from here.
func (c *sessionClock) pass(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = c.at.Add(d)
}

// syncWriter is the session's log, written by the server's goroutines and read
// by the test. The holds and the notice both write there.
type syncWriter struct {
	mu sync.Mutex
	b  strings.Builder
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

func (w *syncWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.String()
}

// rpcMessage is one line of stdout, as little of it as a test needs: which
// request it answers, or which notification it is.
type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

// toolAnswer is a tools/call result: the text items, in order, and whether the
// tool refused.
type toolAnswer struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError"`
}

func (a toolAnswer) texts() []string {
	out := make([]string, 0, len(a.Content))
	for _, item := range a.Content {
		out = append(out, item.Text)
	}
	return out
}

// mcpDriver is one running session a test writes requests into and reads
// answers out of, one at a time, because what the next request says depends on
// what the last answer gave: the code comes out of guide and the hold id out of
// the held write.
type mcpDriver struct {
	t      *testing.T
	in     *io.PipeWriter
	dec    *json.Decoder
	done   chan int
	errOut *syncWriter
	// notices is every notification the session sent, in order.
	notices []string
	next    int
}

func startSession(t *testing.T, args ...string) *mcpDriver {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	errOut := &syncWriter{}
	done := make(chan int, 1)
	go func() {
		code := serveMCP(context.Background(), inR, outW, errOut, args)
		outW.Close()
		done <- code
	}()
	return &mcpDriver{t: t, in: inW, dec: json.NewDecoder(outR), done: done, errOut: errOut, next: 1}
}

// request writes one request and reads the answer to it, keeping whatever
// notifications arrived first.
func (d *mcpDriver) request(method, params string) rpcMessage {
	d.t.Helper()
	id := d.next
	d.next++
	line := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":%q`, id, method)
	if params != "" {
		line += `,"params":` + params
	}
	d.write(line + "}")
	return d.answerTo(id)
}

// notify writes one notification, which is answered by nothing.
func (d *mcpDriver) notify(method string) {
	d.t.Helper()
	d.write(fmt.Sprintf(`{"jsonrpc":"2.0","method":%q}`, method))
}

func (d *mcpDriver) write(line string) {
	d.t.Helper()
	if _, err := io.WriteString(d.in, line+"\n"); err != nil {
		d.t.Fatalf("writing %s to the session: %v", line, err)
	}
}

// call is one tools/call, with the arguments as the test wrote them.
func (d *mcpDriver) call(tool, args string) toolAnswer {
	d.t.Helper()
	m := d.request("tools/call", fmt.Sprintf(`{"name":%q,"arguments":%s}`, tool, args))
	if len(m.Error) != 0 {
		d.t.Fatalf("%s answered a protocol error: %s", tool, m.Error)
	}
	var out toolAnswer
	if err := json.Unmarshal(m.Result, &out); err != nil {
		d.t.Fatalf("%s answered something that is no tool result: %v", tool, err)
	}
	if len(out.Content) == 0 {
		d.t.Fatalf("%s answered no text at all", tool)
	}
	return out
}

// answerTo reads stdout until the answer to this id, judging every line on the
// way: a session's stdout is JSON-RPC and nothing else.
func (d *mcpDriver) answerTo(id int) rpcMessage {
	d.t.Helper()
	for {
		m, err := d.read()
		if err != nil {
			d.t.Fatalf("waiting for the answer to %d: %v, log %q", id, err, d.errOut.String())
		}
		if m.Method != "" && len(m.ID) == 0 {
			d.notices = append(d.notices, m.Method)
			continue
		}
		if string(m.ID) != fmt.Sprint(id) {
			d.t.Fatalf("the answer to %d is out of order: id %s", id, m.ID)
		}
		return *m
	}
}

// read is one message of stdout, or the error that ended the stream.
func (d *mcpDriver) read() (*rpcMessage, error) {
	d.t.Helper()
	var m rpcMessage
	if err := d.dec.Decode(&m); err != nil {
		return nil, err
	}
	if m.JSONRPC != "2.0" {
		d.t.Errorf("stdout carries a message that is not JSON-RPC: %+v", m)
	}
	return &m, nil
}

// end closes stdin, drains whatever is left of stdout and answers the exit code.
// Draining before the wait is what keeps a session that still had something to
// say from blocking on a pipe nobody is reading.
func (d *mcpDriver) end() int {
	d.t.Helper()
	d.in.Close()
	for {
		m, err := d.read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			d.t.Fatalf("draining stdout: %v", err)
		}
		if m.Method != "" && len(m.ID) == 0 {
			d.notices = append(d.notices, m.Method)
			continue
		}
		d.t.Errorf("stdout carries an answer nobody asked for: %+v", m)
	}
	return <-d.done
}

// sessionWire is what the wire says to every call of the session: the document
// and its comment listing, as often as they are asked for, one reply posted, and
// a panic under the one document whose read is meant to crash.
func sessionWire(t *testing.T) *fakeWire {
	t.Helper()
	return &fakeWire{answers: []*answer{
		{method: "GET", match: panicDocID, before: func() { panic("the fake wire failed in a way nothing expected") }},
		{method: "GET", match: fixtureDocID + "?includeTabsContent", json: readFixture(t, "single-tab.json")},
		{method: "GET", match: "/comments?", json: strangerThreads},
		{method: "POST", match: "/comments/AAAA1111/replies", once: true,
			json: `{"id":"R2","createdTime":"2026-09-06T10:45:00Z","content":"` + heldBody + `"}`},
	}}
}

// argsWithCode is an arguments object with this session's code in front of it,
// which is how a model sends one.
func argsWithCode(t *testing.T, code, args string) string {
	t.Helper()
	rest := strings.TrimSpace(strings.TrimSpace(args)[1:])
	opening := `{"code":"` + code + `"`
	if rest == "}" {
		return opening + "}"
	}
	return opening + "," + rest
}

// One session, every tool, against the fakes: the handshake, the guide and its
// code, a call with no code refused, the three reads, a reply held and released
// through the card the hold registered, the other two writes held, the sign-in,
// a command that panics, a call after the panic, and stdin closing with exit 0.
//
// What it is for is the wiring between those, which no test of one tool's own
// function reaches: the code one call handed out reaching the next, the card a
// hold registered in one call releasing the write in another, and the server
// still answering after a command crashed under it.
func TestASessionAgainstFakesRunsEveryTool(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	f := stubWire(t, sessionWire(t))
	reads := stubAccount(t, gapi.Account{Email: "ada@example.org", Name: "Ada Lovelace"}, nil)
	clock := stubSessionClock(t, holdClock)

	d := startSession(t)

	// The handshake. The client names a version it speaks and is answered with
	// that one, and with the instructions saying to call guide first.
	hello := d.request("initialize", `{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"a test","version":"1"}}`)
	if len(hello.Error) != 0 {
		t.Fatalf("initialize was refused: %s", hello.Error)
	}
	if !strings.Contains(string(hello.Result), `"2025-11-25"`) {
		t.Errorf("initialize answers the version the client asked for: %s", hello.Result)
	}
	if !strings.Contains(string(hello.Result), "Call guide before any other tool") {
		t.Errorf("the instructions say to call guide first: %s", hello.Result)
	}
	d.notify("notifications/initialized")

	// The eight tools, in the order a review runs.
	listed := d.request("tools/list", "")
	for _, name := range []string{"read", "comments", "suggestions", "reply", "annotate", "propose", "login", "guide"} {
		if !strings.Contains(string(listed.Result), `"name":"`+name+`"`) {
			t.Errorf("tools/list does not list %s: %s", name, listed.Result)
		}
	}

	// A read before the rules arrived. The answer is the one sentence that fixes
	// it, and nothing was sent.
	refused := d.call("read", `{"code":"0000000000000000","url":"`+fixtureDocID+`"}`)
	if !refused.IsError {
		t.Errorf("a call with a code of nobody's is an error: %v", refused.texts())
	}
	if !strings.Contains(refused.texts()[0], "call guide first") {
		t.Errorf("the refusal must name guide: %s", refused.texts()[0])
	}

	// guide, which reaches nothing and gives the code every other tool needs.
	guide := d.call("guide", `{}`)
	if guide.IsError {
		t.Fatalf("guide was refused: %v", guide.texts())
	}
	var guided struct {
		OK   bool `json:"ok"`
		Data struct {
			Code string `json:"code"`
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(guide.texts()[0]), &guided); err != nil {
		t.Fatalf("the guide answer is no envelope: %v", err)
	}
	if !guided.OK || guided.Data.Code == "" {
		t.Fatalf("guide gave no code: %s", guide.texts()[0])
	}
	if !strings.Contains(guided.Data.Text, "gdoc") {
		t.Errorf("guide gave no rules to read: %q", guided.Data.Text)
	}
	code := guided.Data.Code

	// The three reads. Each answers the fixed line, the envelope and the wrapped
	// copy.
	for _, tool := range []string{"read", "comments", "suggestions"} {
		clock.pass(time.Second)
		got := d.call(tool, argsWithCode(t, code, `{"url":"`+fixtureDocID+`"}`))
		if got.IsError {
			t.Fatalf("%s was refused: %v", tool, got.texts())
		}
		texts := got.texts()
		if len(texts) != 3 {
			t.Fatalf("%s answered %d items, want the line, the envelope and the wrapped copy: %v", tool, len(texts), texts)
		}
		if texts[0] != chat.ReadLine {
			t.Errorf("%s does not open with the fixed line: %q", tool, texts[0])
		}
		if !strings.Contains(texts[1], `"ok":true`) {
			t.Errorf("%s answered an envelope that is not ok: %s", tool, texts[1])
		}
	}

	// A write repeating twelve words of a stranger's comment, which the comments
	// read above carried. It is held, and the hold registers one card.
	clock.pass(time.Second)
	held := d.call("reply", argsWithCode(t, code, heldWriteArgs()["reply"]))
	if held.IsError {
		t.Errorf("a held reply is marked isError, which the chat shows as failed: %v", held.texts())
	}
	env := heldEnvelopeOf(t, held.texts())
	if env.Data.Held.ID == "" {
		t.Fatalf("the held answer names no hold: %s", held.texts()[0])
	}
	if env.Data.Sent {
		t.Error("a held write says nothing was sent")
	}
	confirm := mcpConfirmPrefix + env.Data.Held.ID
	withCard := d.request("tools/list", "")
	if !strings.Contains(string(withCard.Result), `"name":"`+confirm+`"`) {
		t.Fatalf("the hold registered no card: %s", withCard.Result)
	}
	if len(d.notices) == 0 || d.notices[len(d.notices)-1] != "notifications/tools/list_changed" {
		t.Errorf("the client was not told the list moved: %v", d.notices)
	}

	// The person reads the card and approves it, a turn later. What goes out is
	// the write that was held.
	clock.pass(quietEnough)
	sent := d.call(confirm, string(confirmArgs(t, env, nil)))
	if sent.IsError {
		t.Fatalf("the released write was refused: %v", sent.texts())
	}
	if !strings.Contains(sent.texts()[0], `"ok":true`) {
		t.Errorf("the released write answered an envelope that is not ok: %s", sent.texts()[0])
	}
	// And the card is gone, because one approval sends one write.
	after := d.request("tools/list", "")
	if strings.Contains(string(after.Result), confirm) {
		t.Errorf("the card outlived the release: %s", after.Result)
	}

	// The other two writes, each repeating the same stranger's words. Both are held, so the session has called every tool it offers and
	// the wire has still seen one write.
	for _, tool := range []string{"annotate", "propose"} {
		clock.pass(time.Minute)
		got := d.call(tool, argsWithCode(t, code, heldWriteArgs()[tool]))
		if got.IsError {
			t.Errorf("a held %s is marked isError, which the chat shows as failed: %v", tool, got.texts())
		}
		if one := heldEnvelopeOf(t, got.texts()); one.Data.Held.ID == "" {
			t.Errorf("the %s answer names no hold: %s", tool, got.texts()[0])
		}
	}

	// The sign-in, for somebody who is signed in already: the state and the
	// account, and no listener opened.
	clock.pass(time.Second)
	who := d.call("login", `{}`)
	if who.IsError {
		t.Fatalf("login was refused: %v", who.texts())
	}
	if !strings.Contains(who.texts()[0], "signed in") || !strings.Contains(who.texts()[0], "ada@example.org") {
		t.Errorf("login names neither the state nor the account: %s", who.texts()[0])
	}

	// A command that crashes. The answer is one error and the server is still
	// there, which the call after it shows.
	clock.pass(time.Second)
	crashed := d.call("read", argsWithCode(t, code, `{"url":"`+panicDocID+`"}`))
	if !crashed.IsError {
		t.Fatalf("a panic under a command is an error answer: %v", crashed.texts())
	}
	if !strings.Contains(crashed.texts()[0], "gdoc crashed") {
		t.Errorf("the answer does not say what happened: %s", crashed.texts()[0])
	}
	clock.pass(time.Second)
	if again := d.call("guide", `{}`); again.IsError {
		t.Errorf("the session died with the panic: %v", again.texts())
	}

	if code := d.end(); code != 0 {
		t.Fatalf("stdin closing ends the session with exit %d, log %q", code, d.errOut.String())
	}

	// One write reached the wire in the whole session: the one the person
	// approved. The held call sent nothing and the retried card sends nothing
	// twice.
	if got := f.writes(); len(got) != 1 {
		t.Errorf("the session sent %d writes, want the one released: %v", len(got), got)
	}
	// And the account was read once, for the one login call.
	if *reads != 1 {
		t.Errorf("the account was read %d times, want once", *reads)
	}
}
