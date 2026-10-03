package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// mcp is the one command that is not one JSON object, so it is the one command
// routed before run sees the line. The routing helper is what main calls, and
// this is its witness: the word reaches serveMCP, nothing is printed through
// the envelope, and the exit code is the session's own.
//
// --help and -h are not routed. They are the same question about mcp that they
// are about every other command, and they are answered by the table, so
// `gdoc mcp --help` prints the usage line and not a server waiting on a pipe
// nobody is going to fill.
func TestMcpIsRoutedBeforeRun(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	saved := serveMCP
	t.Cleanup(func() { serveMCP = saved })

	var got []string
	var read string
	serveMCP = func(_ context.Context, in io.Reader, out, errOut io.Writer, args []string) int {
		got = append([]string(nil), args...)
		b, _ := io.ReadAll(in)
		read = string(b)
		return 7
	}

	// What follows the word reaches serveMCP as it was given, because serveMCP
	// is what refuses it: TestMcpTakesNoWordsAndNoFlags.
	var out, errOut bytes.Buffer
	code := route(context.Background(), []string{"mcp", "serve"},
		strings.NewReader("a line\n"), &out, &errOut)

	if code != 7 {
		t.Errorf("the exit code is the session's own: got %d, want 7", code)
	}
	if want := []string{"serve"}; !equalWords(got, want) || len(got) != len(want) {
		t.Errorf("serveMCP was handed %v, want %v", got, want)
	}
	if read != "a line\n" {
		t.Errorf("serveMCP reads the stream main was given: got %q", read)
	}
	if out.Len() != 0 {
		t.Errorf("the routing helper prints no envelope for mcp: %q", out.String())
	}

	// The two spellings of the same question, each answered by the table.
	for _, args := range [][]string{{"mcp", "--help"}, {"mcp", "-h"}} {
		got = nil
		var hout, herr bytes.Buffer
		code := route(context.Background(), args, strings.NewReader(""), &hout, &herr)
		if code != 0 {
			t.Errorf("%v is a question answered: exit %d", args, code)
		}
		if got != nil {
			t.Errorf("%v must not reach serveMCP, and it was handed %v", args, got)
		}
		if _, err := onlyJSONObject(&hout); err != nil {
			t.Errorf("%v prints the one object like every other help: %v", args, err)
		}
		if !strings.Contains(herr.String(), "Usage: gdoc mcp") {
			t.Errorf("%v prints the usage line for mcp: %q", args, herr.String())
		}
	}
}

// mcp takes no words and no flags. The Claude Desktop extension has no settings
// since 2026-10-03, so its line is the word mcp and nothing after it, and
// --trusted-email-domains, the flag it used to carry, is refused like any other.
//
// Everything after the word is refused on stderr with exit 1 and nothing on
// stdout: a server that started anyway would be a session built on a line
// nobody meant.
func TestMcpTakesNoWordsAndNoFlags(t *testing.T) {
	for _, bad := range []struct {
		args []string
		says string
	}{
		{[]string{"serve"}, "serve"},
		{[]string{"--stdio"}, "--stdio"},
		{[]string{"--trusted-email-domains="}, "--trusted-email-domains="},
		{[]string{"--trusted-email-domains=example.com"}, "--trusted-email-domains=example.com"},
		{[]string{"--trusted-email-domains", "example.com"}, "--trusted-email-domains"},
	} {
		t.Run(strings.Join(bad.args, " "), func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := serveMCP(context.Background(), strings.NewReader(""), &out, &errOut, bad.args)
			if code != 1 {
				t.Errorf("%v must be refused with exit 1: got %d", bad.args, code)
			}
			if out.Len() != 0 {
				t.Errorf("nothing reaches stdout on a refusal: %q", out.String())
			}
			if !strings.Contains(errOut.String(), bad.says) {
				t.Errorf("the refusal must name %q: %q", bad.says, errOut.String())
			}
		})
	}
}

// Everything on stdout is a JSON-RPC message, and nothing else is: not a
// warning, not a stack trace, not a log line. The client parses that stream a
// line at a time and one stray word ends the session.
//
// The session here is the whole handshake a client makes, through serveMCP
// itself rather than through a server a test built, so the wiring is what is
// measured. Later tasks add their tools to it.
func TestStdoutCarriesOnlyJSONRPC(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	in := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"v"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"guide","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"ping"}`,
		`not json at all`,
		"",
	}, "\n")

	var out, errOut bytes.Buffer
	if code := serveMCP(context.Background(), strings.NewReader(in), &out, &errOut, nil); code != 0 {
		t.Fatalf("stdin closing ends the session: exit %d, stderr %q", code, errOut.String())
	}

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("five messages were asked for and %d came back: %q", len(lines), out.String())
	}
	for _, line := range lines {
		var m map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("stdout carries a line that is not JSON: %q", line)
			continue
		}
		if string(m["jsonrpc"]) != `"2.0"` {
			t.Errorf("stdout carries a line that is not JSON-RPC: %q", line)
		}
		if _, ok := m["id"]; !ok {
			t.Errorf("every answer here is to a request and carries its id: %q", line)
		}
	}

	// The handshake itself, so a session that parses as JSON-RPC and says
	// nothing useful still fails here.
	if !strings.Contains(lines[0], `"2025-11-25"`) {
		t.Errorf("the first answer names the version the client asked for: %q", lines[0])
	}
	if !strings.Contains(lines[0], "guide") {
		t.Errorf("the instructions say to call guide first: %q", lines[0])
	}
	for _, name := range []string{"guide", "login"} {
		if !strings.Contains(lines[1], `"`+name+`"`) {
			t.Errorf("tools/list must list %s: %q", name, lines[1])
		}
	}
}

// The session offers the six table commands beside the two of its own, in the
// specification's own order, which is the order a review runs in.
func TestTheSixTableCommandsAreOffered(t *testing.T) {
	var names []string
	for _, tool := range mcpTools(io.Discard, newMCPLogin(io.Discard), callChat(t)) {
		names = append(names, tool.Name)
	}
	want := []string{"read", "comments", "suggestions", "reply", "annotate", "propose", "login", "guide"}
	if strings.Join(names, " ") != strings.Join(want, " ") {
		t.Errorf("the tools are %v, want %v", names, want)
	}
}

// help and completion are the two readers of the table, and mcp is in it, so a
// person asking what gdoc takes is told about the server and Tab offers it.
// The usage line is spelled out here rather than joined from the table, the
// way the other usage-line test does it.
func TestHelpAndCompletionNameMcp(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	got, prose, code := runHelp(t, "help")
	if code != 0 {
		t.Fatalf("gdoc help: exit %d", code)
	}
	found := false
	for _, name := range helpNames(t, got) {
		if name == "mcp" {
			found = true
		}
	}
	if !found {
		t.Errorf("gdoc help names every command, and mcp is not among them: %v", helpNames(t, got))
	}
	if !strings.Contains(prose, "mcp") {
		t.Errorf("the help a person reads names mcp: %q", prose)
	}

	_, prose, code = runHelp(t, "help", "mcp")
	if code != 0 {
		t.Fatalf("gdoc help mcp: exit %d", code)
	}
	const usage = "Usage: gdoc mcp"
	if !strings.Contains(prose, usage+"\n") {
		t.Errorf("gdoc help mcp must print\n  %s\nand printed\n%s", usage, prose)
	}

	for name, render := range map[string]func([]command) (string, error){"zsh": zshScript, "bash": bashScript} {
		script, err := render(commands())
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(withoutComments(script), "mcp") {
			t.Errorf("the %s script offers every command, and mcp is not in it", name)
		}
	}
}
