package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// The title every fixture under testdata carries, written out here as a literal
// rather than read out of the fixture. What these tests are about is a model
// naming the document a person agreed to, and the one way to check that is to
// spell the name.
const chatTitle = "Supplier register policy"

// pinThread is the comment listing a thread_quote is pinned against. The words
// it opens with carry a curly quote and two double spaces, so a model writing
// the same words with straight quotes and single spaces is naming this thread.
//
// The robot reply is already in it, because the listing is also the read-back
// reply makes of the thread it posted into.
const pinThread = `{"comments":[{"id":"AAAA1111",` +
	`"author":{"displayName":"Ada Lovelace","emailAddress":"ada.lovelace@example.org","me":false},` +
	`"createdTime":"2026-09-06T10:00:00Z","modifiedTime":"2026-09-06T10:30:00Z",` +
	`"content":"ai? whose  ‘register’  is this","resolved":false,` +
	`"quotedFileContent":{"value":"the operations team"},` +
	`"replies":[{"id":"R1",` +
	`"author":{"displayName":"Nail Khusnullin","emailAddress":"nail@example.com","me":true},` +
	`"createdTime":"2026-09-06T10:45:00Z","content":"🤖 the 2026 register"}]}]}`

// chatQuote is the opening words as a model would write them: straight quotes,
// one space between words.
const chatQuote = `ai? whose 'register' is this`

const chatBody = "🤖 the 2026 register"

// pinAnswers is what the wire says to the read every chat write makes of its
// target: the Docs read carrying the title, and the comment listing carrying
// the words each thread opens with.
func pinAnswers(t *testing.T) []*answer {
	t.Helper()
	return []*answer{
		{method: "GET", match: fixtureDocID + "?includeTabsContent", json: readFixture(t, "single-tab.json")},
		{method: "GET", match: "/comments?", json: pinThread},
	}
}

// chatWriteArgs is one call of each write tool, with every argument right.
func chatWriteArgs(title string) map[string]string {
	return map[string]string{
		"reply": `{"url":"` + fixtureDocID + `","title":"` + title + `",` +
			`"comment_id":"AAAA1111","thread_quote":"` + chatQuote + `","body":"` + chatBody + `"}`,
		"annotate": `{"url":"` + fixtureDocID + `","title":"` + title + `",` +
			`"annotations":[{"quoted":"reviewed annually","why":"The 2026 register says quarterly."}]}`,
		"propose": `{"url":"` + fixtureDocID + `","title":"` + title + `",` +
			`"proposals":[{"quoted":"reviewed annually","replacement":"reviewed quarterly","why":"the register says quarterly"}]}`,
	}
}

// titleRefusal is the sentence a wrong title answers with, spelled here so a
// test can tell it from every other refusal a write can carry.
const titleRefusal = "title must be the document's own title"

// A write names its document twice: by id, which a model can carry from a link
// somebody pasted into a comment, and by title, which is what the person in the
// chat said yes to. The title is read from the document itself on every call,
// and a call naming another document is refused with nothing sent.
func TestAWrongTitleIsRefused(t *testing.T) {
	for tool, args := range chatWriteArgs("Quarterly board pack") {
		t.Run(tool, func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
			signedIn(t)
			f := stubWire(t, &fakeWire{answers: pinAnswers(t)})

			code := callCode(t)
			res := mcpRun(context.Background(), mcpToolNamed(t, tool), withCode(t, code, args), nilWriter{}, chatWith(code))
			env := envelopeOf(t, res.Texts)
			if env.OK {
				t.Fatalf("%s answered ok for a call naming another document: %q", tool, env.Error)
			}
			if !strings.Contains(env.Error, titleRefusal) {
				t.Errorf("%s must be refused by the title: %q", tool, env.Error)
			}
			if !strings.Contains(env.Error, chatTitle) {
				t.Errorf("the refusal must say what the title really is: %q", env.Error)
			}
			if !strings.Contains(env.Error, "Quarterly board pack") {
				t.Errorf("the refusal must say what the call said: %q", env.Error)
			}
			if sent := f.writes(); len(sent) != 0 {
				t.Errorf("a refused write sent %v", sent)
			}
		})
	}

	// The title the document really carries is not refused. The three tools do
	// different work past this point, so what is checked here is that none of
	// them is stopped by the title.
	for tool, args := range chatWriteArgs(chatTitle) {
		t.Run(tool+"/right", func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
			signedIn(t)
			stubWire(t, &fakeWire{answers: pinAnswers(t)})

			code := callCode(t)
			res := mcpRun(context.Background(), mcpToolNamed(t, tool), withCode(t, code, args), nilWriter{}, chatWith(code))
			env := envelopeOf(t, res.Texts)
			if strings.Contains(env.Error, titleRefusal) {
				t.Errorf("%s names the document's own title and was refused by it: %q", tool, env.Error)
			}
		})
	}

	// A write with no title at all is refused too: the schema asks for one, and
	// nothing here trusts a schema.
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	stubWire(t, &fakeWire{answers: pinAnswers(t)})
	code := callCode(t)
	res := mcpRun(context.Background(), mcpToolNamed(t, "annotate"), withCode(t, code,
		`{"url":"`+fixtureDocID+`","annotations":[{"quoted":"reviewed annually","why":"b"}]}`), nilWriter{}, chatWith(code))
	if env := envelopeOf(t, res.Texts); env.OK || !strings.Contains(env.Error, "title") {
		t.Errorf("a write with no title must be refused naming it: %v %q", env.OK, env.Error)
	}
}

// reply names its thread twice as well: by id, and by the words the thread opens
// with. The words are compared after curly quotes and runs of whitespace are
// normalised, because a model reading the comments answer and typing the words
// back is not retyping the punctuation. A quote differing by a word is a
// different thread, and the refusal says which words to use.
func TestAThreadQuoteDifferingOnlyInQuotesOrSpacingPasses(t *testing.T) {
	for _, good := range []string{
		chatQuote,
		`ai?  whose ‘register’ is this`,
		"ai? whose\t'register'\nis this",
		`ai? whose 'register' is this document of ours`,
	} {
		t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
		signedIn(t)
		f := stubWire(t, &fakeWire{answers: append(pinAnswers(t), &answer{
			method: "POST", match: "/comments/AAAA1111/replies",
			json: `{"id":"R1","createdTime":"2026-09-06T10:45:00Z","content":"` + chatBody + `"}`,
		})})

		args := `{"url":"` + fixtureDocID + `","title":"` + chatTitle + `",` +
			`"comment_id":"AAAA1111","thread_quote":` + mustJSON(t, good) + `,"body":"` + chatBody + `"}`
		code := callCode(t)
		res := mcpRun(context.Background(), mcpToolNamed(t, "reply"), withCode(t, code, args), nilWriter{}, chatWith(code))
		env := envelopeOf(t, res.Texts)
		if !env.OK {
			t.Errorf("thread_quote %q names the thread and was refused: %q", good, env.Error)
			continue
		}
		if len(f.writes()) != 1 {
			t.Errorf("thread_quote %q: the reply was sent %d times", good, len(f.writes()))
		}
	}

	// A word different is another thread, and nothing is sent.
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	f := stubWire(t, &fakeWire{answers: pinAnswers(t)})
	args := `{"url":"` + fixtureDocID + `","title":"` + chatTitle + `",` +
		`"comment_id":"AAAA1111","thread_quote":"ai? which register is this","body":"` + chatBody + `"}`
	code := callCode(t)
	res := mcpRun(context.Background(), mcpToolNamed(t, "reply"), withCode(t, code, args), nilWriter{}, chatWith(code))
	env := envelopeOf(t, res.Texts)
	if env.OK {
		t.Fatal("a thread_quote naming other words must be refused")
	}
	if !strings.Contains(env.Error, "thread_quote") {
		t.Errorf("the refusal must name the argument: %q", env.Error)
	}
	if !strings.Contains(env.Error, chatQuote) {
		t.Errorf("the refusal must say which words to use: %q", env.Error)
	}
	if sent := f.writes(); len(sent) != 0 {
		t.Errorf("a refused reply sent %v", sent)
	}

	// An id no thread in the document carries is refused by name as well.
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	stubWire(t, &fakeWire{answers: pinAnswers(t)})
	args = `{"url":"` + fixtureDocID + `","title":"` + chatTitle + `",` +
		`"comment_id":"ZZZZ9999","thread_quote":"` + chatQuote + `","body":"` + chatBody + `"}`
	code = callCode(t)
	res = mcpRun(context.Background(), mcpToolNamed(t, "reply"), withCode(t, code, args), nilWriter{}, chatWith(code))
	if env := envelopeOf(t, res.Texts); env.OK || !strings.Contains(env.Error, "comment_id") {
		t.Errorf("a comment_id the document does not carry must be refused naming it: %v %q", env.OK, env.Error)
	}
}

// mustJSON writes a Go string as the JSON string a model would have sent.
func mustJSON(t *testing.T, text string) string {
	t.Helper()
	raw, err := json.Marshal(text)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// One item per write call, so one yes covers one write. The schema says so, and
// the binary refuses a second item rather than trusting it: a client that sent
// the list anyway would otherwise have one approval cover two writes.
func TestASecondItemIsRefused(t *testing.T) {
	for _, one := range []struct {
		tool string
		args string
	}{
		{"annotate", `{"url":"` + fixtureDocID + `","title":"` + chatTitle + `","annotations":[` +
			`{"quoted":"reviewed annually","why":"a"},{"quoted":"the operations team","why":"b"}]}`},
		{"propose", `{"url":"` + fixtureDocID + `","title":"` + chatTitle + `","proposals":[` +
			`{"quoted":"reviewed annually","replacement":"x","why":"a"},` +
			`{"quoted":"the operations team","replacement":"y","why":"b"}]}`},
	} {
		files := &callFiles{}
		argv, err := mcpToolNamed(t, one.tool).argv(json.RawMessage(one.args), files)
		files.remove()
		if err == nil {
			t.Errorf("%s with two items must be refused and became %v", one.tool, argv)
			continue
		}
		if !strings.Contains(err.Error(), "one yes covers one write") {
			t.Errorf("%s: the refusal must say why one item: %q", one.tool, err.Error())
		}
	}

	// One item is what the tools take.
	for tool, args := range chatWriteArgs(chatTitle) {
		if tool == "reply" {
			continue
		}
		files := &callFiles{}
		_, err := mcpToolNamed(t, tool).argv(json.RawMessage(args), files)
		files.remove()
		if err != nil {
			t.Errorf("%s with one item: %v", tool, err)
		}
	}
}

// assignee is not in chat. It makes Google email the address it names, and a
// chat write already emails everybody on the document: an address a comment
// asked for would be a stranger's words choosing who hears from gdoc.
//
// The field reaches the command through a file written byte for byte, so the
// command's own reader would take it. This server refuses it before the file is
// written, by the one list that says what an item may carry: the schema the card
// shows.
func TestAssigneeIsRefused(t *testing.T) {
	files := &callFiles{}
	defer files.remove()

	args := `{"url":"` + fixtureDocID + `","title":"` + chatTitle + `","proposals":[` +
		`{"quoted":"reviewed annually","replacement":"reviewed quarterly","why":"a",` +
		`"assignee":"ada.lovelace@example.org"}]}`
	argv, err := mcpToolNamed(t, "propose").argv(json.RawMessage(args), files)
	if err == nil {
		t.Fatalf("assignee must be refused and became %v", argv)
	}
	if !strings.Contains(err.Error(), "assignee") {
		t.Errorf("the refusal must name the field: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "proposals") {
		t.Errorf("the refusal must name the argument it is in: %q", err.Error())
	}
}
