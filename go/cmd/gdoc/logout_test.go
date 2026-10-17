// The sign-out as a command: what it answers, what it records, and the one
// front door it is not behind.

package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tokenFile is where the token sits in a config dir a test owns. signedIn,
// which mcpauth_test.go holds, is what puts it there.
func tokenFile(dir string) string { return filepath.Join(dir, "oauth-token.json") }

// The command answers the two facts and the file is gone. Being signed out
// already is the same answer with removed false, because the question has
// been answered either way.
func TestTheLogoutCommandAnswersWhatItRemoved(t *testing.T) {
	t.Run("a token was there", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("GDOC_CONFIG_DIR", dir)
		signedIn(t)
		path := tokenFile(dir)

		got, code := runJSON(t, "auth", "logout")
		if code != 0 || got["ok"] != true {
			t.Fatalf("a sign-out is an answer: %v (exit %d)", got, code)
		}
		data, _ := got["data"].(map[string]any)
		if data["removed"] != true {
			t.Errorf("the object says removed %v: %v", data["removed"], data)
		}
		if data["token_path"] != path {
			t.Errorf("the object names %v, want %q", data["token_path"], path)
		}
		if _, err := os.Lstat(path); err == nil {
			t.Errorf("%s is still there", path)
		}
	})

	t.Run("nobody was signed in", func(t *testing.T) {
		t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

		got, code := runJSON(t, "auth", "logout")
		if code != 0 || got["ok"] != true {
			t.Fatalf("no token is an answer: %v (exit %d)", got, code)
		}
		data, _ := got["data"].(map[string]any)
		if data["removed"] != false {
			t.Errorf("the object says removed %v, and there was no token: %v", data["removed"], data)
		}
	})
}

// The sentence a reader meets says what a sign-out does not do. The token is
// this computer's; the grant is Google's, and it is still there afterwards.
func TestTheLogoutSummarySaysTheGrantStaysAtGoogle(t *testing.T) {
	entry := match([]string{"auth", "logout"})
	if entry == nil {
		t.Fatal("the table has no auth logout")
	}
	for _, word := range []string{"Google", "account settings"} {
		if !strings.Contains(entry.summary, word) {
			t.Errorf("the summary must name %q: %q", word, entry.summary)
		}
	}
	if !entry.report {
		t.Error("a sign-out is a run somebody is waiting on, and it reports nothing")
	}
	if entry.readsBack {
		t.Error("a sign-out writes into no document, so it reads nothing back")
	}
}

// TestLogoutIsNoChatTool holds the one front door this command is not behind.
// A model that could sign the person out of Google could stop every other
// tool of its own session working, and nobody in the chat asked it to. The
// list a session offers is read as the client reads it.
func TestLogoutIsNoChatTool(t *testing.T) {
	tools := mcpTools(io.Discard, newMCPLogin(io.Discard), callChat(t))
	if len(tools) == 0 {
		t.Fatal("the session offers no tools, so this test is measuring nothing")
	}
	for _, tool := range tools {
		if strings.Contains(tool.Name, "logout") {
			t.Errorf("the session offers %q, and a chat may not sign anybody out", tool.Name)
		}
		if strings.Contains(strings.ToLower(tool.Description), "sign out") {
			t.Errorf("%q describes signing out: %q", tool.Name, tool.Description)
		}
	}
	for _, c := range mcpCommands() {
		if strings.Contains(c.tool, "logout") {
			t.Errorf("the chat commands name %q", c.tool)
		}
	}
}

// TestLogoutWritesAStartAndAnEndWithNoDoc is the record's half. A sign-out is
// a run the panel shows, and it names no document: there is none, and the
// record carries no path and no account either.
func TestLogoutWritesAStartAndAnEndWithNoDoc(t *testing.T) {
	dir := recording(t)
	signedIn(t)

	var out bytes.Buffer
	if code := run(context.Background(), []string{"auth", "logout"}, &out, io.Discard, presenceOn()); code != 0 {
		t.Fatalf("a sign-out failed: exit %d, %s", code, out.String())
	}
	events := presenceEvents(t, dir)
	if len(events) != 2 {
		t.Fatalf("a sign-out wrote %d events: %+v", len(events), events)
	}
	for _, e := range events {
		if e.Command != "auth logout" || e.Doc != "" {
			t.Errorf("a sign-out event names %q and document %q", e.Command, e.Doc)
		}
	}
	end := endOf(t, events)
	if end.OK == nil || !*end.OK {
		t.Errorf("the end does not carry the envelope's ok: %+v", end)
	}
	if end.Title != "" {
		t.Errorf("the end carries title %q, and a sign-out names no document", end.Title)
	}
	if end.Verified != nil {
		t.Errorf("the end carries verified %v, and a sign-out reads nothing back", *end.Verified)
	}
}
