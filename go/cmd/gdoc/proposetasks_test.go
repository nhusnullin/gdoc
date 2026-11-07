package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"gdoc/internal/settings"
)

// This file is the one rule the two doors of propose answer differently: the
// address a task in a block names. The terminal finishes a bare owner against
// the domain in effect, because a person typing the command can read the
// setting and the note. Chat finishes nothing: the card shows the words the
// model wrote, and a domain filled in behind them would not be on it.
//
// tasks_test.go holds publish's half of the same question, and
// internal/propose/block_test.go the parsing underneath both.

// taskBlock is a proposals file holding one block whose list is a task with a
// bare owner. It is the file both terminal tests below are about.
const taskBlock = `[{"kind":"block","after":"reviewed annually",` +
	`"content":"- [ ] Ask the bank @anna\n","why":"the register names no bank"}]`

// domainInEffect is the mark these tests write under with a domain beside it,
// as a settings file carrying both would give.
func domainInEffect() settings.InEffect {
	return settings.InEffect{
		Mark:       "🤖",
		MarkFrom:   settings.FromArgument,
		Domain:     "example.org",
		DomainFrom: settings.FromArgument,
	}
}

// A bare owner on the terminal is finished against the domain in effect, so the
// chip the batch carries names a whole address. The note and the setting are
// both in front of the person who typed the command, which is what makes the
// completion theirs rather than gdoc's.
//
// What this test measures is the request: the one batch, and the address on it.
// Whether the read-backs then hold is internal/propose's own question, over
// its own fixtures, and TestTheBlockReadBackHoldsWithChips is where it is
// asked; the fixtures here carry the everyday block rather than this task, so
// the envelope reports a change it could not confirm.
func TestTheTerminalResolvesABareOwner(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: blockAnswers(t)})
	from := tempFile(t, "proposals.json", taskBlock)

	got, _ := runJSONSet(t, domainInEffect(), "propose", proposeDocID, "--from", from)
	list, _ := dataOf(t, got)["proposals"].([]any)
	if len(list) != 1 {
		t.Fatalf("proposals = %v, want the one entry", list)
	}
	if one, _ := list[0].(map[string]any); one["sent"] != true {
		t.Fatalf("the proposal was not sent: %v", list[0])
	}
	sent := f.writes()
	if len(sent) != 1 {
		t.Fatalf("writes = %v, want the one batch", sent)
	}
	body := string(sent[0].Body)
	if !strings.Contains(body, "anna@example.org") {
		t.Errorf("the batch carries no chip for the finished address:\n%s", body)
	}
	if strings.Contains(body, "@anna ") || strings.Contains(body, `"@anna"`) {
		t.Errorf("the batch still carries the bare token:\n%s", body)
	}
}

// The same file with no domain in effect is refused, and refused before the
// session is opened: a publish of a bare owner is the same rule, and a chip on
// nobody is the thing neither may write.
func TestTheTerminalRefusesABareOwnerWithNoDomain(t *testing.T) {
	f := stubWire(t, &fakeWire{answers: blockAnswers(t)})
	from := tempFile(t, "proposals.json", taskBlock)

	got, code := runJSONSet(t, robotInEffect(), "propose", proposeDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a bare owner with no domain must stop the run: %v (exit %d)", got, code)
	}
	text, _ := got["error"].(string)
	if !strings.Contains(text, "@anna") {
		t.Errorf("the refusal does not name the token it could not finish: %q", text)
	}
	if !strings.Contains(text, "domain") {
		t.Errorf("the refusal does not name the setting that would finish it: %q", text)
	}
	if len(f.calls) != 0 {
		t.Errorf("nothing may reach Google: %v", f.calls)
	}
}

// chatProposals is one propose call's arguments, with the block's content as
// the model would have written it.
func chatProposals(t *testing.T, content string) string {
	t.Helper()
	return `{"url":"` + fixtureDocID + `","title":"` + chatTitle + `","proposals":[` +
		`{"kind":"block","after":"reviewed annually","content":` + mustJSON(t, content) +
		`,"why":"the register names no bank"}]}`
}

// Chat refuses a bare owner, whatever the domain in effect is. The person in
// the chat agrees to the words on the card, and the card is the call's own
// arguments: an address gdoc finished from a setting nobody is looking at is an
// address they never saw. So the refusal asks for the whole address, and no
// file is written.
func TestChatRefusesABareOwner(t *testing.T) {
	files := &callFiles{}
	defer files.remove()

	argv, err := mcpToolNamed(t, "propose").argv(
		json.RawMessage(chatProposals(t, "- [ ] Ask the bank @anna\n")), files)
	if err == nil {
		t.Fatalf("a bare owner in chat must be refused and became %v", argv)
	}
	if !strings.Contains(err.Error(), "@anna") {
		t.Errorf("the refusal must name the token: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "whole address") {
		t.Errorf("the refusal must ask for the whole address: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "proposals") {
		t.Errorf("the refusal must name the argument it is in: %q", err.Error())
	}
}

// An address written out in full is what chat writes, and it reaches the
// command byte for byte: nothing here needs a setting, and the words on the
// card are the words in the document.
func TestChatTakesAFullAddress(t *testing.T) {
	files := &callFiles{}
	defer files.remove()

	content := "- [ ] Ask the bank @anna@example.com 2026-11-20\n"
	argv, err := mcpToolNamed(t, "propose").argv(
		json.RawMessage(chatProposals(t, content)), files)
	if err != nil {
		t.Fatalf("a whole address in chat: %v", err)
	}
	path := ""
	for _, a := range argv {
		if rest, ok := strings.CutPrefix(a, "--from="); ok {
			path = rest
		}
	}
	if path == "" {
		t.Fatalf("argv = %v, want the proposals file on it", argv)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "@anna@example.com 2026-11-20") {
		t.Errorf("the file does not carry the task as the call wrote it:\n%s", raw)
	}
}
