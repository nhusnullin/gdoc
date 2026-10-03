// The framing tests: the 4 MiB ceiling, a blank line, a last line with no
// newline of its own, and one compact line per message on the way out.

package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// pingOfLength builds a valid ping whose line, with the newline that ends it,
// is exactly n bytes.
func pingOfLength(t *testing.T, n int) string {
	t.Helper()
	head := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{"pad":"`
	tail := `"}}`
	pad := n - 1 - len(head) - len(tail)
	if pad < 0 {
		t.Fatalf("%d bytes is too few for a ping", n)
	}
	line := head + strings.Repeat("x", pad) + tail
	if len(line)+1 != n {
		t.Fatalf("built %d bytes, wanted %d", len(line)+1, n)
	}
	if !json.Valid([]byte(line)) {
		t.Fatal("the built line is not JSON")
	}
	return line
}

func TestALineAtTheCeilingIsReadAndOneOverItIsNot(t *testing.T) {
	at := pingOfLength(t, 4194304)
	msgs := run(t, New(testInfo(), []Tool{readTool()}, nil), at)
	if len(msgs) != 1 || string(msgs[0]["id"]) != "1" || string(msgs[0]["result"]) != "{}" {
		t.Fatalf("a line of exactly 4194304 bytes was not read: %v", msgs)
	}

	over := pingOfLength(t, 4194305)
	msgs = run(t, New(testInfo(), []Tool{readTool()}, nil),
		over,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
	)
	if len(msgs) != 2 {
		t.Fatalf("want one refusal and one answer, got %d", len(msgs))
	}
	if code := errorCode(t, msgs[0]); code != -32600 {
		t.Errorf("one byte over the ceiling: code %d, want -32600", code)
	}
	if string(msgs[0]["id"]) != "null" {
		t.Errorf("an oversize line answered with id %s; nothing in it could be read", msgs[0]["id"])
	}
	if string(msgs[1]["id"]) != "2" {
		t.Errorf("reading did not go on past the oversize line: %v", msgs[1])
	}
}

func TestABlankLineIsSkipped(t *testing.T) {
	msgs := run(t, New(testInfo(), []Tool{readTool()}, nil),
		``,
		"   ",
		"\t",
		`{"jsonrpc":"2.0","id":1,"method":"ping"}`,
	)
	if len(msgs) != 1 || string(msgs[0]["id"]) != "1" {
		t.Errorf("a blank line was answered: %v", msgs)
	}
}

func TestALastLineWithoutANewlineIsStillRead(t *testing.T) {
	s := New(testInfo(), []Tool{readTool()}, nil)
	var out strings.Builder
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n" + `{"jsonrpc":"2.0","id":2,"method":"ping"}`)
	if err := s.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	msgs := decode(t, out.String())
	if len(msgs) != 2 || string(msgs[1]["id"]) != "2" {
		t.Errorf("the last line without a newline was dropped: %v", msgs)
	}
}

func TestACarriageReturnBeforeTheNewlineIsNotPartOfTheMessage(t *testing.T) {
	s := New(testInfo(), []Tool{readTool()}, nil)
	var out strings.Builder
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\r\n")
	if err := s.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	msgs := decode(t, out.String())
	if len(msgs) != 1 || string(msgs[0]["result"]) != "{}" {
		t.Errorf("a CRLF line was refused: %v", msgs)
	}
}

func TestOneCompactLinePerMessage(t *testing.T) {
	printed := runRaw(t, New(testInfo(), []Tool{readTool()}, nil),
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read","arguments":{"url":"https://docs.google.com/document/d/abc/edit"}}}`)
	if len(printed) != 1 {
		t.Fatalf("one answer printed %d lines: %q", len(printed), printed)
	}
	if strings.Contains(printed[0], "\n") || strings.Contains(printed[0], "  ") {
		t.Errorf("the answer is not one compact line: %q", printed[0])
	}
}

func TestATextWithANewlineInItStaysOneLine(t *testing.T) {
	printed := runRaw(t, New(testInfo(), []Tool{{
		Name:   "read",
		Schema: json.RawMessage(`{"type":"object"}`),
		Call: func(context.Context, json.RawMessage) Result {
			return Result{Texts: []string{"one\ntwo\nthree"}}
		},
	}}, nil), `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read","arguments":{}}}`)
	if len(printed) != 1 {
		t.Fatalf("a text with two newlines in it printed %d lines: %q", len(printed), printed)
	}
	if !strings.Contains(printed[0], `one\ntwo\nthree`) {
		t.Errorf("the newlines were not escaped: %q", printed[0])
	}
}
