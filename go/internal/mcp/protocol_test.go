// The protocol tests: the handshake, the methods, the ids, the tool list and
// the list-changed notification. The framing tests are in frame_test.go.

package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testInfo is the server identity every test here starts from. The version is
// a literal, not the build's, so a release never moves a test.
func testInfo() Info {
	return Info{Name: "gdoc", Version: "v2.9.0", Instructions: "Call guide first."}
}

// readTool is a read-only tool that answers with the arguments it was handed,
// so a test can see what reached it.
func readTool() Tool {
	return Tool{
		Name:        "read",
		Title:       "Read a Google Doc",
		Description: "Reads the document's text.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"url":{"type":"string"},"tab":{"type":"string"}},"required":["url"]}`),
		ReadOnly:    true,
		Call: func(ctx context.Context, args json.RawMessage) Result {
			return Result{Texts: []string{string(args)}}
		},
	}
}

// replyTool is a write tool that refuses arguments it cannot read, as every
// real tool does: a tool error, never a protocol error.
func replyTool() Tool {
	return Tool{
		Name:        "reply",
		Title:       "Reply to a comment in a Google Doc",
		Description: "Posts one reply.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"url":{"type":"string"},"text":{"type":"string"}},"required":["url","text"]}`),
		Call: func(ctx context.Context, args json.RawMessage) Result {
			var got struct {
				URL  string `json:"url"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal(args, &got); err != nil || got.URL == "" || got.Text == "" {
				return Result{Texts: []string{"reply needs a url and a text"}, IsError: true}
			}
			return Result{Texts: []string{"posted"}}
		},
	}
}

// run feeds the lines to one server, waits for Serve to end, and gives back
// every line it printed, decoded as a map so a test can read a member without
// deciding the order of the rest.
func run(t *testing.T, s *Server, lines ...string) []map[string]json.RawMessage {
	t.Helper()
	var out bytes.Buffer
	in := ""
	for _, line := range lines {
		in += line + "\n"
	}
	if err := s.Serve(context.Background(), strings.NewReader(in), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	return decode(t, out.String())
}

// runRaw is run without the decoding, for a test that pins the bytes.
func runRaw(t *testing.T, s *Server, lines ...string) []string {
	t.Helper()
	var out bytes.Buffer
	in := ""
	for _, line := range lines {
		in += line + "\n"
	}
	if err := s.Serve(context.Background(), strings.NewReader(in), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	printed := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(printed) == 1 && printed[0] == "" {
		return nil
	}
	return printed
}

func decode(t *testing.T, printed string) []map[string]json.RawMessage {
	t.Helper()
	var msgs []map[string]json.RawMessage
	for _, line := range strings.Split(printed, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("a printed line is not a JSON object: %q: %v", line, err)
		}
		msgs = append(msgs, m)
	}
	return msgs
}

// errorCode reads the code out of a response that carries an error member.
func errorCode(t *testing.T, m map[string]json.RawMessage) int {
	t.Helper()
	raw, ok := m["error"]
	if !ok {
		t.Fatalf("the answer carries no error member: %v", m)
	}
	var e struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatal(err)
	}
	if e.Message == "" {
		t.Error("an error answer with no message says nothing to the reader")
	}
	return e.Code
}

func initializeLine(version string) string {
	return `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"` + version + `","capabilities":{},"clientInfo":{"name":"claude-ai","version":"1"}}}`
}

func TestInitializeAnswersEachSupportedVersion(t *testing.T) {
	for _, version := range []string{"2025-11-25", "2025-06-18", "2025-03-26", "2024-11-05"} {
		msgs := run(t, New(testInfo(), []Tool{readTool()}, nil), initializeLine(version))
		if len(msgs) != 1 {
			t.Fatalf("%s: want one answer, got %d", version, len(msgs))
		}
		var got struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(msgs[0]["result"], &got); err != nil {
			t.Fatal(err)
		}
		if got.ProtocolVersion != version {
			t.Errorf("asked %s, answered %s; a supported version is echoed", version, got.ProtocolVersion)
		}
	}
}

func TestAnUnsupportedVersionGetsTheNewest(t *testing.T) {
	lines := []string{
		initializeLine("2024-01-01"),
		initializeLine(""),
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}`,
		`{"jsonrpc":"2.0","id":1,"method":"initialize"}`,
	}
	for _, line := range lines {
		msgs := run(t, New(testInfo(), []Tool{readTool()}, nil), line)
		if len(msgs) != 1 {
			t.Fatalf("%s: want one answer, got %d", line, len(msgs))
		}
		var got struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(msgs[0]["result"], &got); err != nil {
			t.Fatal(err)
		}
		if got.ProtocolVersion != "2025-11-25" {
			t.Errorf("%s: answered %s, want the newest 2025-11-25", line, got.ProtocolVersion)
		}
	}
}

func TestInitializeCarriesCapabilitiesInfoAndInstructions(t *testing.T) {
	printed := runRaw(t, New(testInfo(), []Tool{readTool()}, nil), initializeLine("2025-11-25"))
	want := `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-11-25","capabilities":{"tools":{"listChanged":true}},"serverInfo":{"name":"gdoc","version":"v2.9.0"},"instructions":"Call guide first."}}`
	if len(printed) != 1 || printed[0] != want {
		t.Errorf("the handshake answer moved.\n got %q\nwant %q", strings.Join(printed, "\n"), want)
	}
}

func TestAnEmptyVersionIsDev(t *testing.T) {
	info := testInfo()
	info.Version = ""
	msgs := run(t, New(info, []Tool{readTool()}, nil), initializeLine("2025-11-25"))
	var got struct {
		ServerInfo struct {
			Version string `json:"version"`
		} `json:"serverInfo"`
	}
	if err := json.Unmarshal(msgs[0]["result"], &got); err != nil {
		t.Fatal(err)
	}
	if got.ServerInfo.Version != "dev" {
		t.Errorf("a checkout build named %q, want dev", got.ServerInfo.Version)
	}
}

func TestAnUnknownMethodIsMethodNotFoundBeforeAndAfterInitialize(t *testing.T) {
	msgs := run(t, New(testInfo(), []Tool{readTool()}, nil),
		`{"jsonrpc":"2.0","id":1,"method":"server/discover"}`,
		initializeLine("2025-11-25"),
		`{"jsonrpc":"2.0","id":3,"method":"server/discover"}`,
		`{"jsonrpc":"2.0","id":4,"method":"resources/list"}`,
		`{"jsonrpc":"2.0","id":5,"method":"prompts/list"}`,
	)
	if len(msgs) != 5 {
		t.Fatalf("want five answers, got %d", len(msgs))
	}
	for _, i := range []int{0, 2, 3, 4} {
		if code := errorCode(t, msgs[i]); code != -32601 {
			t.Errorf("answer %d: code %d, want -32601", i, code)
		}
	}
	if _, ok := msgs[1]["result"]; !ok {
		t.Error("initialize between two unknown methods lost its own answer")
	}
}

func TestAnUnknownToolIsInvalidParams(t *testing.T) {
	msgs := run(t, New(testInfo(), []Tool{readTool()}, nil),
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nope","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call"}`,
	)
	if len(msgs) != 2 {
		t.Fatalf("want two answers, got %d", len(msgs))
	}
	for i, m := range msgs {
		if code := errorCode(t, m); code != -32602 {
			t.Errorf("answer %d: code %d, want -32602", i, code)
		}
	}
}

func TestBadArgumentsAreAToolErrorNotAProtocolError(t *testing.T) {
	msgs := run(t, New(testInfo(), []Tool{readTool(), replyTool()}, nil),
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"reply","arguments":{"url":"https://docs.google.com/document/d/abc/edit"}}}`,
	)
	if len(msgs) != 1 {
		t.Fatalf("want one answer, got %d", len(msgs))
	}
	if _, ok := msgs[0]["error"]; ok {
		t.Fatal("bad arguments came back as a protocol error; the model cannot correct itself from that")
	}
	var got struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(msgs[0]["result"], &got); err != nil {
		t.Fatal(err)
	}
	if !got.IsError {
		t.Error("the tool said error and the result does not")
	}
	if len(got.Content) != 1 || got.Content[0].Type != "text" || got.Content[0].Text != "reply needs a url and a text" {
		t.Errorf("the tool's own words did not reach the result: %+v", got.Content)
	}
}

func TestEveryTextIsOneContentItemInOrder(t *testing.T) {
	three := Tool{
		Name:   "three",
		Schema: json.RawMessage(`{"type":"object"}`),
		Call: func(ctx context.Context, args json.RawMessage) Result {
			return Result{Texts: []string{"first", "second", "third"}}
		},
	}
	none := Tool{
		Name:   "none",
		Schema: json.RawMessage(`{"type":"object"}`),
		Call: func(ctx context.Context, args json.RawMessage) Result {
			return Result{}
		},
	}
	printed := runRaw(t, New(testInfo(), []Tool{three, none}, nil),
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"three","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"none","arguments":{}}}`,
	)
	want := []string{
		`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"first"},{"type":"text","text":"second"},{"type":"text","text":"third"}],"isError":false}}`,
		`{"jsonrpc":"2.0","id":2,"result":{"content":[],"isError":false}}`,
	}
	if len(printed) != 2 || printed[0] != want[0] || printed[1] != want[1] {
		t.Errorf("the result shape moved.\n got %q\nwant %q", printed, want)
	}
}

func TestANotificationGetsNoAnswer(t *testing.T) {
	printed := runRaw(t, New(testInfo(), []Tool{readTool()}, nil),
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","method":"ping"}`,
		`{"jsonrpc":"2.0","method":"tools/list"}`,
	)
	if len(printed) != 0 {
		t.Errorf("a notification was answered: %q", printed)
	}
}

func TestAnUnknownNotificationIsIgnored(t *testing.T) {
	printed := runRaw(t, New(testInfo(), []Tool{readTool()}, nil),
		`{"jsonrpc":"2.0","method":"notifications/nothing/here"}`,
		`{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":9}}`,
	)
	if len(printed) != 0 {
		t.Errorf("an unknown notification was answered: %q", printed)
	}
}

func TestStringAndNumberIDsAreEchoedExactly(t *testing.T) {
	long := `"` + strings.Repeat("a", 200) + `"`
	for _, id := range []string{`"7"`, `7`, `7.0`, long, `null`, `-3`, `12345678901234567890`} {
		printed := runRaw(t, New(testInfo(), []Tool{readTool()}, nil),
			`{"jsonrpc":"2.0","id":`+id+`,"method":"ping"}`)
		want := `{"jsonrpc":"2.0","id":` + id + `,"result":{}}`
		if len(printed) != 1 || printed[0] != want {
			t.Errorf("id %s was not echoed byte for byte.\n got %q\nwant %q", id, printed, want)
		}
	}
}

func TestAMalformedLineAnOversizeLineAndABatchEachGetOneErrorAndReadingGoesOn(t *testing.T) {
	oversize := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{"pad":"` + strings.Repeat("x", 5<<20) + `"}}`
	msgs := run(t, New(testInfo(), []Tool{readTool()}, nil),
		`{nope`,
		oversize,
		`[{"jsonrpc":"2.0","id":1,"method":"ping"}]`,
		`"a string is not a message"`,
		`{"jsonrpc":"2.0","id":9,"method":"ping"}`,
	)
	if len(msgs) != 5 {
		t.Fatalf("want five answers, got %d: %v", len(msgs), msgs)
	}
	if code := errorCode(t, msgs[0]); code != -32700 {
		t.Errorf("a line that is not JSON: code %d, want -32700", code)
	}
	for _, i := range []int{1, 2, 3} {
		if code := errorCode(t, msgs[i]); code != -32600 {
			t.Errorf("answer %d: code %d, want -32600", i, code)
		}
	}
	for _, i := range []int{0, 1, 2, 3} {
		if string(msgs[i]["id"]) != "null" {
			t.Errorf("answer %d: id %s, want null when the id could not be read", i, msgs[i]["id"])
		}
	}
	if string(msgs[4]["id"]) != "9" || string(msgs[4]["result"]) != "{}" {
		t.Errorf("reading did not go on: %v", msgs[4])
	}
}

func TestToolsListCarriesTitlesSchemasAndHintsInOrder(t *testing.T) {
	printed := runRaw(t, New(testInfo(), []Tool{readTool(), replyTool()}, nil),
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	want := `{"jsonrpc":"2.0","id":2,"result":{"tools":[` +
		`{"name":"read","title":"Read a Google Doc","description":"Reads the document's text.","inputSchema":{"type":"object","properties":{"url":{"type":"string"},"tab":{"type":"string"}},"required":["url"]},"annotations":{"title":"Read a Google Doc","readOnlyHint":true,"destructiveHint":false}},` +
		`{"name":"reply","title":"Reply to a comment in a Google Doc","description":"Posts one reply.","inputSchema":{"type":"object","properties":{"url":{"type":"string"},"text":{"type":"string"}},"required":["url","text"]},"annotations":{"title":"Reply to a comment in a Google Doc","readOnlyHint":false,"destructiveHint":false}}` +
		`]}}`
	if len(printed) != 1 || printed[0] != want {
		t.Errorf("the tool list moved.\n got %q\nwant %q", strings.Join(printed, "\n"), want)
	}
}

func TestAToolWithNoSchemaStillLists(t *testing.T) {
	msgs := run(t, New(testInfo(), []Tool{{Name: "bare", Call: func(context.Context, json.RawMessage) Result { return Result{} }}}, nil),
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	var got struct {
		Tools []struct {
			InputSchema json.RawMessage `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(msgs[0]["result"], &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Tools) != 1 || string(got.Tools[0].InputSchema) != `{"type":"object"}` {
		t.Errorf("a tool with no schema listed as %v, want the empty object schema", got.Tools)
	}
}

func TestATextWithAngleBracketsIsNotEscaped(t *testing.T) {
	labelled := Tool{
		Name:   "read",
		Schema: json.RawMessage(`{"type":"object"}`),
		Call: func(ctx context.Context, args json.RawMessage) Result {
			return Result{Texts: []string{"<<doc-text ab12>>hello & goodbye<<end ab12>>"}}
		},
	}
	printed := runRaw(t, New(testInfo(), []Tool{labelled}, nil),
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read","arguments":{}}}`)
	if len(printed) != 1 || !strings.Contains(printed[0], `<<doc-text ab12>>hello & goodbye<<end ab12>>`) {
		t.Errorf("the label was escaped on the way out: %q", printed)
	}
}

// conn is one live session: lines in, lines out, while the test holds both
// ends. Both pipes are the operating system's, which buffers, so an Add from
// the test goroutine does not wait for the test to read its notification.
type conn struct {
	in   *os.File
	out  *bufio.Reader
	done chan error
}

func live(t *testing.T, s *Server) *conn {
	t.Helper()
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	c := &conn{in: inW, out: bufio.NewReader(outR), done: make(chan error, 1)}
	go func() {
		serr := s.Serve(context.Background(), inR, outW)
		outW.Close()
		c.done <- serr
	}()
	t.Cleanup(func() {
		inW.Close()
		<-c.done
		outR.Close()
		inR.Close()
	})
	return c
}

func (c *conn) send(t *testing.T, line string) {
	t.Helper()
	if _, err := io.WriteString(c.in, line+"\n"); err != nil {
		t.Fatalf("writing %q: %v", line, err)
	}
}

func (c *conn) next(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	type read struct {
		line []byte
		err  error
	}
	got := make(chan read, 1)
	go func() {
		line, err := c.out.ReadBytes('\n')
		got <- read{line, err}
	}()
	select {
	case r := <-got:
		if r.err != nil {
			t.Fatalf("reading one line: %v", r.err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(r.line, &m); err != nil {
			t.Fatalf("a printed line is not a JSON object: %q: %v", r.line, err)
		}
		return m
	case <-time.After(5 * time.Second):
		t.Fatal("no line came back in five seconds")
		return nil
	}
}

// toolNames reads the names out of a tools/list answer, in their order.
func toolNames(t *testing.T, m map[string]json.RawMessage) []string {
	t.Helper()
	var got struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(m["result"], &got); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(got.Tools))
	for _, tool := range got.Tools {
		names = append(names, tool.Name)
	}
	return names
}

func confirmTool(name string) Tool {
	return Tool{
		Name:   name,
		Title:  "Confirm this write",
		Schema: json.RawMessage(`{"type":"object"}`),
		Call: func(ctx context.Context, args json.RawMessage) Result {
			return Result{Texts: []string{"done"}}
		},
	}
}

func TestAddAndRemoveSendListChanged(t *testing.T) {
	s := New(testInfo(), []Tool{readTool()}, nil)
	c := live(t, s)

	c.send(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	if names := toolNames(t, c.next(t)); len(names) != 1 || names[0] != "read" {
		t.Fatalf("first list: %v", names)
	}

	s.Add(confirmTool("confirm_a1b2"))
	told := c.next(t)
	if _, ok := told["id"]; ok {
		t.Error("a list-changed notification carries an id")
	}
	var method string
	if err := json.Unmarshal(told["method"], &method); err != nil {
		t.Fatal(err)
	}
	if method != "notifications/tools/list_changed" {
		t.Errorf("Add sent %q", method)
	}

	c.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if names := toolNames(t, c.next(t)); len(names) != 2 || names[1] != "confirm_a1b2" {
		t.Fatalf("after Add: %v", names)
	}

	s.Remove("confirm_a1b2")
	told = c.next(t)
	if err := json.Unmarshal(told["method"], &method); err != nil {
		t.Fatal(err)
	}
	if method != "notifications/tools/list_changed" {
		t.Errorf("Remove sent %q", method)
	}

	c.send(t, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`)
	if names := toolNames(t, c.next(t)); len(names) != 1 || names[0] != "read" {
		t.Fatalf("after Remove: %v", names)
	}

	s.Remove("confirm_a1b2")
	c.send(t, `{"jsonrpc":"2.0","id":4,"method":"ping"}`)
	if got := c.next(t); string(got["id"]) != "4" {
		t.Errorf("removing a name that is not there told the client something: %v", got)
	}
}

func TestAddBeforeServeTellsNobodyAndStillLists(t *testing.T) {
	s := New(testInfo(), []Tool{readTool()}, nil)
	s.Add(confirmTool("confirm_c3d4"))
	msgs := run(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	if len(msgs) != 1 {
		t.Fatalf("want one answer and no notification, got %d: %v", len(msgs), msgs)
	}
	if names := toolNames(t, msgs[0]); len(names) != 2 || names[1] != "confirm_c3d4" {
		t.Errorf("the tool added before Serve is not listed: %v", names)
	}
}

func TestAddReplacesAToolOfTheSameName(t *testing.T) {
	s := New(testInfo(), []Tool{readTool(), replyTool()}, nil)
	replacement := readTool()
	replacement.Title = "Read it again"
	s.Add(replacement)
	msgs := run(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	names := toolNames(t, msgs[0])
	if len(names) != 2 || names[0] != "read" || names[1] != "reply" {
		t.Errorf("a replacement moved the list: %v", names)
	}
}

func TestStdinClosingEndsServe(t *testing.T) {
	s := New(testInfo(), []Tool{readTool()}, nil)
	c := live(t, s)
	c.send(t, `{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	c.next(t)
	c.in.Close()
	select {
	case err := <-c.done:
		c.done <- err
		if err != nil {
			t.Errorf("Serve ended with %v, want nil when stdin simply closed", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not end five seconds after stdin closed")
	}
}

// overlapWriter is a writer that notices a second write starting before the
// first has finished. It holds the write open for a moment, so two goroutines
// writing without a lock between them overlap rather than happening to miss
// each other.
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

// Two goroutines logging at once write two lines, never two halves of two.
//
// A session logs from both sides: the reader says a cancellation would not
// decode while the worker says a tool panicked. os.Stderr takes one Fprintf as
// one write, which is why a real session has never shown this, but Serve takes
// any writer and every test here hands it a buffer.
func TestTwoGoroutinesLoggingDoNotOverlap(t *testing.T) {
	const loggers = 8
	w := &overlapWriter{}
	s := New(Info{Name: "gdoc", Version: "v1.2.3"}, nil, w)

	var wg sync.WaitGroup
	for i := 0; i < loggers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.logf("mcp: the tool %d panicked", i)
		}(i)
	}
	wg.Wait()

	if w.overlapped.Load() {
		t.Error("two log lines were written at once, so a reader sees halves of both")
	}
	if got := w.lines.Load(); got != loggers {
		t.Errorf("the log took %d lines, want the %d written", got, loggers)
	}
}
