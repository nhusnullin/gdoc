// The words of a reply: on the line, or in a file, and exactly one of the two.
//
// A chat reply is a sentence the model just wrote, and writing it to a file so
// the command can read it back made a directory for one string. The flag it
// now goes through is the terminal's as well, because a colleague replying by
// hand has the same sentence in their head and no file to put it in.

package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// Both routes carry the same words to Drive, and the mark in effect goes in
// front of them once. The body on the line is trimmed at the end exactly as a
// file's is: a read-back compares what was sent.
func TestReplyTakesTheBodyOnTheLineOrInAFile(t *testing.T) {
	for _, one := range []struct {
		name string
		flag string
		give func(t *testing.T) string
	}{
		{"on the line", "--body", func(*testing.T) string { return "The 2026 register.  " }},
		{"in a file", "--body-file", func(t *testing.T) string {
			return tempFile(t, "reply.txt", "The 2026 register.\n")
		}},
	} {
		t.Run(one.name, func(t *testing.T) {
			f := stubWire(t, &fakeWire{answers: []*answer{
				{method: "POST", match: "/comments/AAAA1111/replies", json: `{"id":"R1","createdTime":"2026-09-06T10:45:00Z","content":"🤖 The 2026 register."}`},
				{method: "GET", match: "/comments?", json: readFixture(t, "comments.json")},
			}})

			got, code := runJSONSet(t, robotInEffect(), "reply", fixtureDocID, "AAAA1111", one.flag, one.give(t))
			if code != 0 || got["ok"] != true {
				t.Fatalf("reply %s: %v (exit %d)", one.flag, got, code)
			}
			if data := dataOf(t, got); data["reply_id"] != "R1" || data["verified"] != true {
				t.Errorf("data = %v, want the reply id and a read-back that found it", data)
			}
			posts := f.writes()
			if len(posts) != 1 {
				t.Fatalf("one reply is one write: %v", posts)
			}
			var body map[string]any
			if err := json.Unmarshal(posts[0].Body, &body); err != nil {
				t.Fatal(err)
			}
			if body["content"] != "🤖 The 2026 register." {
				t.Errorf("content = %q, want the words under the mark and nothing trailing", body["content"])
			}
		})
	}
}

// Both at once is a call where which one counts is not decided here, and the
// refusal names the two flags so the caller drops one.
func TestReplyRefusesBothBodyAndBodyFile(t *testing.T) {
	f := stubWire(t, &fakeWire{})
	path := tempFile(t, "reply.txt", "The 2026 register.")

	got, code := runJSON(t, "reply", proposeDocID, "AAAA1111", "--body", "The 2026 register.", "--body-file", path)
	if code == 0 || got["ok"] != false {
		t.Fatalf("both flags must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "--body") || !strings.Contains(msg, "--body-file") {
		t.Errorf("the refusal must name both flags: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may reach Drive for a call that was refused: %v", f.calls)
	}
}

// Neither is the same refusal from the other side: the words are what the
// command is about, and the caller is told both ways of giving them.
func TestReplyRefusesNeitherBodyNorBodyFile(t *testing.T) {
	f := stubWire(t, &fakeWire{})

	got, code := runJSON(t, "reply", proposeDocID, "AAAA1111")
	if code == 0 || got["ok"] != false {
		t.Fatalf("reply without the words must fail: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "--body") || !strings.Contains(msg, "--body-file") {
		t.Errorf("the refusal must name both flags: %q", msg)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may reach Drive for a call that was refused: %v", f.calls)
	}
}

// toolNamed is one tool of the chat table, by name.
func toolNamed(t *testing.T, name string) mcpCommand {
	t.Helper()
	for _, c := range mcpCommands() {
		if c.tool == name {
			return c
		}
	}
	t.Fatalf("%s is no tool of the table", name)
	return mcpCommand{}
}

// A chat reply puts no file on disk: the words go on the line joined to the
// flag, so the call makes no directory for one sentence.
func TestAChatReplyWritesNoFile(t *testing.T) {
	const args = `{"url":"` + fixtureDocID + `","comment_id":"AAAA1111","body":"yes, the 2026 register"}`

	files := &callFiles{}
	defer files.remove()
	argv, err := toolNamed(t, "reply").argv(json.RawMessage(args), files)
	if err != nil {
		t.Fatal(err)
	}
	if files.dir != "" {
		t.Errorf("the call made a directory at %q, and a reply needs no file", files.dir)
	}
	if want := "--body=yes, the 2026 register"; !contains(argv, want) {
		t.Errorf("argv = %v, want it to carry %q", argv, want)
	}
}

// A reply opening with a hyphen is plain text the binary takes, so the dash
// refusal is not asked of the body: the value is joined to the flag, which the
// parser cannot read as a flag of its own.
func TestAChatReplyOpeningWithAHyphenIsSent(t *testing.T) {
	const hyphen = "- the register says quarterly"
	const args = `{"url":"` + fixtureDocID + `","comment_id":"AAAA1111","body":"` + hyphen + `"}`

	files := &callFiles{}
	defer files.remove()
	argv, err := toolNamed(t, "reply").argv(json.RawMessage(args), files)
	if err != nil {
		t.Fatalf("a reply opening with a hyphen must be sent: %v", err)
	}
	if want := "--body=" + hyphen; !contains(argv, want) {
		t.Errorf("argv = %v, want it to carry %q", argv, want)
	}

	// And the line it built is a line the command runs: the parser reads the
	// value off the join, and plaintext takes a hyphen bullet.
	f := stubWire(t, &fakeWire{answers: []*answer{
		{method: "POST", match: "/comments/AAAA1111/replies", json: `{"id":"R1","createdTime":"2026-09-06T10:45:00Z","content":"🤖 ` + hyphen + `"}`},
		{method: "GET", match: "/comments?", json: readFixture(t, "comments.json")},
	}})
	got, code := runJSONSet(t, robotInEffect(), append([]string{"reply"}, argv[1:]...)...)
	if code != 0 || got["ok"] != true {
		t.Fatalf("the line chat built was refused: %v (exit %d)", got, code)
	}
	var body map[string]any
	if err := json.Unmarshal(f.writes()[0].Body, &body); err != nil {
		t.Fatal(err)
	}
	if body["content"] != "🤖 "+hyphen {
		t.Errorf("content = %q, want the hyphen as the model typed it", body["content"])
	}
}

// The dash exemption belongs to one argument. Anywhere else a value opening
// with a dash is a flag nobody meant, and the refusal stays: the forward half
// of that is TestAWordStringThatStartsWithADashIsRefused.
func TestOnlyTheReplyBodyMayOpenWithADash(t *testing.T) {
	for _, c := range mcpCommands() {
		for _, f := range c.flags {
			if !f.dashAllowed {
				continue
			}
			if c.tool != "reply" || f.prop != "body" {
				t.Errorf("%s's %s may open with a dash, and only reply's body may", c.tool, f.prop)
			}
		}
	}
}
