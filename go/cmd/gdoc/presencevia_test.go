// Who ran this, and the one wait a person stopped.
//
// The via of a run is a fact about the process rather than about the line: the
// same `gdoc read` is a terminal run when a person types it and a code run
// when Claude Code's own tool starts it. The environment is where that is
// read, once, in dispatch, and these tests are the only place the variable is
// set, because every other presence test clears it in recording.

package main

import (
	"bytes"
	"context"
	"io"
	"testing"

	"gdoc/internal/gapi"
)

// A run Claude Code started says so, on its start and on its end.
//
// Any value of the variable counts: it is Claude Code's own, and a release
// that writes something other than 1 in it is still Claude Code. An empty
// variable is a person at a terminal, which is what an absent one is too.
func TestViaIsCodeUnderClaudeCode(t *testing.T) {
	for _, c := range []struct {
		name  string
		value string
		want  string
	}{
		{"Claude Code as it sets it today", "1", "code"},
		{"a value this version has not seen", "true", "code"},
		{"the variable cleared", "", "terminal"},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := recording(t)
			t.Setenv(claudeCodeEnv, c.value)
			stubSession(t, docsAndComments(t))

			var out bytes.Buffer
			if code := run(context.Background(), []string{"read", fixtureDocID}, &out, io.Discard, presenceOn()); code != 0 {
				t.Fatalf("read failed: exit %d, %s", code, out.String())
			}

			events := presenceEvents(t, dir)
			if len(events) != 2 {
				t.Fatalf("want a start and an end, got %d: %+v", len(events), events)
			}
			for _, e := range events {
				if e.Via != c.want {
					t.Errorf("the %s event is via %q, want %q", e.Event, e.Via, c.want)
				}
			}
		})
	}
}

// A tool call of a chat session is a chat call even where Claude Code started
// the process, which is exactly how a session under Claude Code's own MCP
// client runs. The session marks its context once and nothing takes that mark
// off it: internal/presence's TestChatWinsOverCode is the same rule over the
// context alone.
func TestViaChatWinsOverCodeInTheMcpSession(t *testing.T) {
	dir := recording(t)
	t.Setenv(claudeCodeEnv, "1")
	signedIn(t)
	stubWire(t, sessionWire(t))
	stubAccount(t, gapi.Account{Email: "ada@example.org", Name: "Ada Lovelace"}, nil)

	d := startSessionWith(t, presenceOn())
	d.request("initialize", `{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"a test","version":"1"}}`)
	d.notify("notifications/initialized")
	code := codeFromGuide(t, d)
	if got := d.call("read", argsWithCode(t, code, `{"url":"`+fixtureDocID+`"}`)); got.IsError {
		t.Fatalf("the read was refused: %v", got.texts())
	}
	d.end()

	events := presenceEvents(t, dir)
	if len(events) != 2 {
		t.Fatalf("one tool call is one start and one end, and wrote %d: %+v", len(events), events)
	}
	for _, e := range events {
		if e.Via != "chat" {
			t.Errorf("a call from a session under Claude Code is via %q", e.Via)
		}
	}
}

// The end of a wait somebody stopped says so, from the command's own answer.
// A wait that was not stopped says nothing: the field is a fact and not a
// flag with two values, so there is no false in the record for the app to
// draw.
func TestAStoppedWaitsEndSaysSo(t *testing.T) {
	t.Run("a person stopped it", func(t *testing.T) {
		dir := recording(t)
		stubSession(t, docsAndComments(t))

		var out bytes.Buffer
		if code := run(done(t), []string{"comments", fixtureDocID, "--since", waitSince, "--wait", "9m"}, &out, io.Discard, presenceOn()); code != 0 {
			t.Fatalf("a stopped wait is still an answer: exit %d, %s", code, out.String())
		}

		events := presenceEvents(t, dir)
		if len(events) != 2 {
			t.Fatalf("want a start and an end, got %d: %+v", len(events), events)
		}
		start, end := events[0], events[1]
		if !start.Wait {
			t.Errorf("the start of a wait does not say it waits: %+v", start)
		}
		if start.Interrupted {
			t.Errorf("a run that has begun has not been stopped: %+v", start)
		}
		if !end.Interrupted {
			t.Errorf("the end of the stopped wait says nothing: %+v", end)
		}
	})

	t.Run("nobody stopped it", func(t *testing.T) {
		dir := recording(t)
		stubSession(t, docsAndComments(t))

		var out bytes.Buffer
		if code := run(context.Background(), []string{"comments", fixtureDocID, "--since", waitSince}, &out, io.Discard, presenceOn()); code != 0 {
			t.Fatalf("the listing failed: exit %d, %s", code, out.String())
		}

		for _, e := range presenceEvents(t, dir) {
			if e.Interrupted {
				t.Errorf("a listing that never waited says a person stopped it: %+v", e)
			}
		}
	})
}

// The session of a code run is on its start line, from the one variable
// Claude Code hands a child beside CLAUDECODE, and nowhere else.
//
// It is the record's one word about the person's own session, and the panel
// has it so Open chat can offer the session a document was worked on from. A
// terminal run names none, and a chat call names none either, even where
// Claude Code started the process: the session is a fact about the run, and a
// tool call of gdoc mcp is a chat run.
func TestTheSessionOfACodeRunIsOnItsStart(t *testing.T) {
	const session = "0199c3d4-5e6f-7a8b-9c0d-1e2f3a4b5c6d"

	t.Run("a run Claude Code started", func(t *testing.T) {
		dir := recording(t)
		t.Setenv(claudeCodeEnv, "1")
		t.Setenv(claudeSessionEnv, session)
		stubSession(t, docsAndComments(t))

		var out bytes.Buffer
		if code := run(context.Background(), []string{"read", fixtureDocID}, &out, io.Discard, presenceOn()); code != 0 {
			t.Fatalf("read failed: exit %d, %s", code, out.String())
		}

		events := presenceEvents(t, dir)
		if len(events) != 2 {
			t.Fatalf("want a start and an end, got %d: %+v", len(events), events)
		}
		if events[0].Session != session {
			t.Errorf("the start says session %q, want %q", events[0].Session, session)
		}
		if events[1].Session != "" {
			t.Errorf("the end says session %q, and the start already said it", events[1].Session)
		}
	})

	t.Run("a person at a terminal", func(t *testing.T) {
		dir := recording(t)
		t.Setenv(claudeSessionEnv, session)
		stubSession(t, docsAndComments(t))

		var out bytes.Buffer
		if code := run(context.Background(), []string{"read", fixtureDocID}, &out, io.Discard, presenceOn()); code != 0 {
			t.Fatalf("read failed: exit %d, %s", code, out.String())
		}

		for _, e := range presenceEvents(t, dir) {
			if e.Session != "" {
				t.Errorf("a terminal run names a session: %+v", e)
			}
		}
	})

	t.Run("a tool call of the chat session", func(t *testing.T) {
		dir := recording(t)
		t.Setenv(claudeCodeEnv, "1")
		t.Setenv(claudeSessionEnv, session)
		signedIn(t)
		stubWire(t, sessionWire(t))
		stubAccount(t, gapi.Account{Email: "ada@example.org", Name: "Ada Lovelace"}, nil)

		d := startSessionWith(t, presenceOn())
		d.request("initialize", `{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"a test","version":"1"}}`)
		d.notify("notifications/initialized")
		code := codeFromGuide(t, d)
		if got := d.call("read", argsWithCode(t, code, `{"url":"`+fixtureDocID+`"}`)); got.IsError {
			t.Fatalf("the read was refused: %v", got.texts())
		}
		d.end()

		for _, e := range presenceEvents(t, dir) {
			if e.Session != "" {
				t.Errorf("a chat call names a session: %+v", e)
			}
		}
	})
}
