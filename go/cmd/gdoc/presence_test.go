// The hook: what a run records when the presence is on, and what it changes
// when it is off.
//
// The first two tests here are the guards over the whole feature. The flag may
// move no byte of the object a caller reads and no exit code, and a record
// that cannot be written may not either. Everything else in this file is about
// what reaches the file: which commands report, what a start and an end carry,
// and which of them may carry a verified.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gdoc/internal/emit"
	"gdoc/internal/gapi"
	"gdoc/internal/presence"
	"gdoc/internal/settings"
)

// presenceOn is the settings of a run that records what it is doing, as route
// resolves them from --presence true.
func presenceOn() settings.InEffect {
	set := defaultsInEffect()
	set.Presence, set.PresenceFrom = true, settings.FromArgument
	return set
}

// recording points the config directory at a fresh temporary one and answers
// where the record will be, so one test's events are never another test's.
//
// It also clears the one variable that would make every run of these tests a
// Claude Code run, because the suite itself is often run from a Claude Code
// session and what a run records may not depend on who started the test:
// TestViaIsCodeUnderClaudeCode sets it back for the one test about it.
func recording(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	t.Setenv(claudeCodeEnv, "")
	return dir
}

// recorded is every event the run wrote, in the order it wrote them. The
// month's file is found rather than named, so a run on the last second of a
// month is read as well as any other.
func presenceEvents(t *testing.T, dir string) []presence.Event {
	t.Helper()
	names, err := filepath.Glob(filepath.Join(dir, "presence", "*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	var out []presence.Event
	for _, name := range names {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
			if line == "" {
				continue
			}
			var e presence.Event
			if err := json.Unmarshal([]byte(line), &e); err != nil {
				t.Fatalf("%q is no event: %v", line, err)
			}
			out = append(out, e)
		}
	}
	return out
}

// nothingRecorded fails the test unless the run touched the record at all.
func nothingRecorded(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, "presence")); !os.IsNotExist(err) {
		t.Fatalf("the run wrote a record: %v, %v", err, presenceEvents(t, dir))
	}
}

// outcome is one run as a caller sees it: the object, the words and the code.
type outcome struct {
	out, errOut string
	code        int
}

// scene is one command run twice, once with the flag off and once with it on.
// setup makes this run's own stubs and files, because the two runs are two
// runs: a note a publish rewrote is not a note the next publish may read.
//
// It answers the paths of this run's own temporary files beside the line,
// because a publish prints the note it changed and each run's note is under a
// directory of its own. Those are blanked out of both answers before they are
// compared: what is being measured is the flag, and a temporary path is not
// something the flag could move.
type scene struct {
	name  string
	setup func(t *testing.T) (context.Context, []string, []string)
}

// presenceScenes are the four commands the guard is measured over: a read, a
// wait that was already stopped, a write that is read back, and the create
// that makes a document of its own.
func presenceScenes() []scene {
	return []scene{
		{
			name: "read",
			setup: func(t *testing.T) (context.Context, []string, []string) {
				stubSession(t, docsAndComments(t))
				return context.Background(), []string{"read", fixtureDocID}, nil
			},
		},
		{
			name: "comments --wait",
			setup: func(t *testing.T) (context.Context, []string, []string) {
				stubSession(t, docsAndComments(t))
				return done(t), []string{"comments", fixtureDocID, "--wait", "9m"}, nil
			},
		},
		{
			name: "reply",
			setup: func(t *testing.T) (context.Context, []string, []string) {
				stubWire(t, &fakeWire{answers: []*answer{
					{method: "POST", match: "/comments/AAAA1111/replies",
						json: `{"id":"R1","createdTime":"2026-09-06T10:45:00Z","content":"🤖 The 2026 register."}`},
					{method: "GET", match: "/comments?", json: readFixture(t, "comments.json")},
				}})
				path := tempFile(t, "reply.txt", "The 2026 register.\n")
				return context.Background(), []string{"reply", fixtureDocID, "AAAA1111", "--body-file", path}, nil
			},
		},
		{
			name: "publish",
			setup: func(t *testing.T) (context.Context, []string, []string) {
				stubWire(t, &fakeWire{answers: publishAnswers(t)})
				_, md := buildNote(t)
				return context.Background(), []string{"publish", "--md", md, "--folder-id", testFolderID}, []string{md}
			},
		},
	}
}

// blank takes this run's own temporary paths out of what it printed, so two
// runs of one command are compared on everything else.
func blank(text string, volatile []string) string {
	for _, path := range volatile {
		text = strings.ReplaceAll(text, path, "<this run's own file>")
	}
	return text
}

// The one rule the whole feature stands on: with the flag on, a caller reads
// exactly what they read with it off. The same object on stdout, the same
// words on stderr, the same exit code. The only difference a run makes is the
// file beside the token.
func TestTheFlagChangesNoByteOfTheObject(t *testing.T) {
	for _, sc := range presenceScenes() {
		t.Run(sc.name, func(t *testing.T) {
			var got [2]outcome
			for i, set := range []settings.InEffect{defaultsInEffect(), presenceOn()} {
				t.Run(map[bool]string{false: "off", true: "on"}[set.Presence], func(t *testing.T) {
					dir := recording(t)
					ctx, args, volatile := sc.setup(t)
					var out, errOut bytes.Buffer
					code := run(ctx, args, &out, &errOut, set)
					got[i] = outcome{out: blank(out.String(), volatile), errOut: blank(errOut.String(), volatile), code: code}

					if !set.Presence {
						nothingRecorded(t, dir)
						return
					}
					events := presenceEvents(t, dir)
					if len(events) < 2 || events[0].Event != "start" || events[len(events)-1].Event != "end" {
						t.Fatalf("a reporting command writes a start and an end, and wrote %d: %+v", len(events), events)
					}
					// And between them, only the parts of the one command that
					// has parts. A publish writes four; every other command
					// writes none, so its two lines are next to each other.
					for _, e := range events[1 : len(events)-1] {
						if e.Event != "step" {
							t.Fatalf("the run wrote a %q between its start and its end: %+v", e.Event, e)
						}
					}
				})
			}
			if got[0].out != got[1].out {
				t.Errorf("the object moved with the flag on:\noff %s\non  %s", got[0].out, got[1].out)
			}
			if got[0].errOut != got[1].errOut {
				t.Errorf("stderr moved with the flag on:\noff %q\non  %q", got[0].errOut, got[1].errOut)
			}
			if got[0].code != got[1].code {
				t.Errorf("the exit code moved with the flag on: off %d, on %d", got[0].code, got[1].code)
			}
		})
	}
}

// A record that cannot be written changes nothing either. The path where the
// record goes is a file, so the directory cannot be made: the object, the exit
// code and the work are what they were, and one line on stderr says so.
//
// The line is once per run and run() starts the run's own, so the two runs
// below and every other test in this package each get their own: without that
// this test would pass on the first run of a process and fail on the second.
func TestAnUnwritableLogChangesNoByteOfTheObject(t *testing.T) {
	var got [2]outcome
	for i, set := range []settings.InEffect{defaultsInEffect(), presenceOn()} {
		dir := recording(t)
		if err := os.WriteFile(filepath.Join(dir, "presence"), []byte("not a directory\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		stubSession(t, docsAndComments(t))
		var out, errOut bytes.Buffer
		code := run(context.Background(), []string{"read", fixtureDocID}, &out, &errOut, set)
		got[i] = outcome{out: out.String(), errOut: errOut.String(), code: code}
	}
	if got[0].out != got[1].out {
		t.Errorf("the object moved when the record could not be written:\noff %s\non  %s", got[0].out, got[1].out)
	}
	if got[0].code != got[1].code {
		t.Errorf("the exit code moved: off %d, on %d", got[0].code, got[1].code)
	}
	if got[0].errOut != "" {
		t.Errorf("the run with the flag off said %q", got[0].errOut)
	}
	// One line, because a session that cannot write would otherwise say the
	// same sentence on every call, and the first one said everything.
	lines := strings.Split(strings.TrimSuffix(got[1].errOut, "\n"), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], "the presence record was not written") {
		t.Fatalf("stderr carries %d lines, want the one naming the failure: %q", len(lines), got[1].errOut)
	}
	if strings.Contains(got[1].errOut, "Supplier register policy") {
		t.Errorf("the line carries the document's words: %q", got[1].errOut)
	}
}

// A reporting command writes two lines: the document at the start, and the
// document, the title and the ok at the end. A terminal run says so.
func TestAReportingCommandWritesAStartAndAnEnd(t *testing.T) {
	dir := recording(t)
	stubSession(t, docsAndComments(t))

	var out bytes.Buffer
	if code := run(context.Background(), []string{"read", fixtureDocID}, &out, io.Discard, presenceOn()); code != 0 {
		t.Fatalf("read failed: exit %d, %s", code, out.String())
	}

	events := presenceEvents(t, dir)
	if len(events) != 2 {
		t.Fatalf("want a start and an end, got %d: %+v", len(events), events)
	}
	start, end := events[0], events[1]
	if start.Event != "start" || end.Event != "end" {
		t.Fatalf("the two events are %q and %q", start.Event, end.Event)
	}
	for _, e := range events {
		if e.Command != "read" {
			t.Errorf("the event names command %q", e.Command)
		}
		if e.Via != "terminal" {
			t.Errorf("a run from the line is via %q", e.Via)
		}
		if e.Doc != fixtureDocID {
			t.Errorf("the event names document %q, want the one the line named", e.Doc)
		}
		if e.Call == 0 {
			t.Error("the event carries no call number, so nothing pairs it")
		}
	}
	if start.Call != end.Call {
		t.Errorf("the start is call %d and the end is call %d", start.Call, end.Call)
	}
	if start.OK != nil || start.MS != nil || start.Title != "" {
		t.Errorf("a start says nothing about the outcome: %+v", start)
	}
	if end.OK == nil || !*end.OK {
		t.Errorf("the end does not carry the envelope's ok: %+v", end)
	}
	if end.Title != "Supplier register policy" {
		t.Errorf("the end carries title %q, want the document's own", end.Title)
	}
	if end.MS == nil {
		t.Error("the end carries no duration")
	}
}

// The verified is lifted for the five commands whose caller says they read
// their own write back, and for nothing else. update and restyle print
// verified: false to mean no write happened, and that must never become a
// failed read-back in a record meant to last.
func TestOnlyTheReadBackWritersCarryVerified(t *testing.T) {
	// The table first: the two bits, as literals, against the names.
	readBack := map[string]bool{"reply": true, "propose": true, "withdraw": true, "annotate": true, "publish": true}
	reports := map[string]bool{
		"read": true, "comments": true, "suggestions": true, "export": true, "restyle": true,
		"reply": true, "propose": true, "withdraw": true, "annotate": true, "publish": true,
		"update": true, "auth login": true, "auth logout": true,
	}
	for _, c := range commands() {
		if c.readsBack != readBack[c.name] {
			t.Errorf("%s reads back %t, want %t", c.name, c.readsBack, readBack[c.name])
		}
		if c.report != reports[c.name] {
			t.Errorf("%s reports %t, want %t", c.name, c.report, reports[c.name])
		}
		if c.readsBack && !c.report {
			t.Errorf("%s reads back and reports nothing, so the verified reaches no record", c.name)
		}
	}

	// And the two runs: a reply, whose read-back found the words, and a restyle
	// survey, whose envelope says verified: false because it wrote nothing.
	t.Run("reply carries it", func(t *testing.T) {
		dir := recording(t)
		stubWire(t, &fakeWire{answers: []*answer{
			{method: "POST", match: "/comments/AAAA1111/replies",
				json: `{"id":"R1","createdTime":"2026-09-06T10:45:00Z","content":"🤖 The 2026 register."}`},
			{method: "GET", match: "/comments?", json: readFixture(t, "comments.json")},
		}})
		path := tempFile(t, "reply.txt", "The 2026 register.\n")
		set := presenceOn()
		set.Mark, set.MarkFrom = "🤖", settings.FromArgument

		var out bytes.Buffer
		if code := run(context.Background(),
			[]string{"reply", fixtureDocID, "AAAA1111", "--body-file", path}, &out, io.Discard, set); code != 0 {
			t.Fatalf("reply failed: exit %d, %s", code, out.String())
		}
		end := endOf(t, presenceEvents(t, dir))
		if end.Verified == nil || !*end.Verified {
			t.Errorf("the end of a read-back writer carries no verified: %+v", end)
		}
	})

	t.Run("a check carries none", func(t *testing.T) {
		// An update that installed nothing prints verified: false, meaning no
		// write happened. That is the one the record has to leave out: kept, it
		// would read a year later as a read-back that failed.
		installedAt(t, "v2.0.0", []byte("old"), &stubPlain{
			listing: listing("v2.1.0"),
			files:   published(t, "v2.1.0", []byte("new")),
		})
		dir := os.Getenv("GDOC_CONFIG_DIR")

		var out bytes.Buffer
		if code := run(context.Background(), []string{"update", "--check"}, &out, io.Discard, presenceOn()); code != 0 {
			t.Fatalf("the check failed: exit %d, %s", code, out.String())
		}
		if !bytes.Contains(out.Bytes(), []byte(`"verified":false`)) {
			t.Fatalf("a check no longer prints verified: false, so this test proves nothing: %s", out.String())
		}
		if end := endOf(t, presenceEvents(t, dir)); end.Verified != nil {
			t.Errorf("a check's end carries verified %v, and the command reads nothing back", *end.Verified)
		}
	})
}

// endOf is the one end event among what a run wrote.
func endOf(t *testing.T, events []presence.Event) presence.Event {
	t.Helper()
	for _, e := range events {
		if e.Event == "end" {
			return e
		}
	}
	t.Fatalf("the run wrote no end: %+v", events)
	return presence.Event{}
}

// `gdoc auth status` writes no event. It is the command a person runs to ask
// what they have, and a question about the token is not work on a document.
func TestAnAuthStatusRunWritesNoEvent(t *testing.T) {
	dir := recording(t)
	var out bytes.Buffer
	run(context.Background(), []string{"auth", "status"}, &out, io.Discard, presenceOn())
	nothingRecorded(t, dir)
}

// The help writes no event either, and it is the command that asks GitHub
// unasked: a person reading the help is not a run to watch.
func TestAHelpRunWritesNoEvent(t *testing.T) {
	dir := recording(t)
	var out bytes.Buffer
	if code := run(context.Background(), []string{"help", "--json"}, &out, io.Discard, presenceOn()); code != 0 {
		t.Fatalf("help failed: exit %d, %s", code, out.String())
	}
	nothingRecorded(t, dir)
}

// A call that came from Claude Desktop says so. The session marks its context
// once, and every command under it is via: chat.
func TestAChatCallSaysViaChat(t *testing.T) {
	dir := recording(t)
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
			t.Errorf("a call from a session is via %q", e.Via)
		}
		if e.Command != "read" {
			t.Errorf("the event names command %q", e.Command)
		}
	}
}

// A call the session refused before any command ran writes nothing: there was
// no run to record. A wrong title and a missing code are the two shapes.
func TestAChatRefusalBeforeDispatchWritesNothing(t *testing.T) {
	t.Run("a title that is not the document's", func(t *testing.T) {
		dir := recording(t)
		signedIn(t)
		stubWire(t, &fakeWire{answers: pinAnswers(t)})
		ch := chatWithSettings(callCode(t), presenceOn())
		args := `{"url":"` + fixtureDocID + `","title":"Some other document",` +
			`"comment_id":"AAAA1111","thread_quote":"` + chatQuote + `","body":"The 2026 register."}`
		res := mcpRun(presence.WithChat(context.Background()), mcpToolNamed(t, "reply"),
			withCode(t, ch.code, args), nilWriter{}, ch)
		if !res.IsError {
			t.Fatalf("a call naming another document was accepted: %v", res.Texts)
		}
		nothingRecorded(t, dir)
	})

	t.Run("no code", func(t *testing.T) {
		dir := recording(t)
		signedIn(t)
		stubSession(t, docsAndComments(t))
		ch := chatWithSettings(callCode(t), presenceOn())
		res := mcpRun(presence.WithChat(context.Background()), mcpToolNamed(t, "read"),
			json.RawMessage(`{"url":"`+fixtureDocID+`"}`), nilWriter{}, ch)
		if !res.IsError {
			t.Fatalf("a call without the code was accepted: %v", res.Texts)
		}
		nothingRecorded(t, dir)
	})
}

// A command that crashed still ends its run. safeDispatch turns the panic into
// an envelope, and the end written on the way out says ok: false.
func TestAPanicInACommandStillWritesAnEndWithOkFalse(t *testing.T) {
	dir := recording(t)
	stubLogin(t, func(io.Writer) error { panic("something impossible") })

	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"auth", "login"}, &out, &errOut, presenceOn()); code != 1 {
		t.Fatalf("a crash must exit 1, got %d: %s", code, out.String())
	}
	events := presenceEvents(t, dir)
	if len(events) != 2 {
		t.Fatalf("the crashed run wrote %d events, want its start and its end: %+v", len(events), events)
	}
	end := endOf(t, events)
	if end.OK == nil || *end.OK {
		t.Errorf("the end of a crashed run says ok %v", end.OK)
	}
	if end.Command != "auth login" {
		t.Errorf("the end names command %q", end.Command)
	}
}

// Two reporting commands name no document, because neither is about one: an
// update is about this binary, and a login is about the token.
func TestAnUpdateAndALoginWriteNoDoc(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		installedAt(t, "v2.0.0", []byte("old"), &stubPlain{
			listing: listing("v2.1.0"),
			files:   published(t, "v2.1.0", []byte("new")),
		})
		dir := os.Getenv("GDOC_CONFIG_DIR")
		var out bytes.Buffer
		run(context.Background(), []string{"update", "--check"}, &out, io.Discard, presenceOn())
		events := presenceEvents(t, dir)
		if len(events) != 2 {
			t.Fatalf("an update wrote %d events: %+v", len(events), events)
		}
		for _, e := range events {
			if e.Command != "update" || e.Doc != "" {
				t.Errorf("an update event names %q and document %q", e.Command, e.Doc)
			}
		}
	})

	t.Run("auth login", func(t *testing.T) {
		dir := recording(t)
		stubLogin(t, func(io.Writer) error { return nil })
		var out bytes.Buffer
		run(context.Background(), []string{"auth", "login"}, &out, io.Discard, presenceOn())
		events := presenceEvents(t, dir)
		if len(events) != 2 {
			t.Fatalf("a login wrote %d events: %+v", len(events), events)
		}
		for _, e := range events {
			if e.Command != "auth login" || e.Doc != "" {
				t.Errorf("a login event names %q and document %q", e.Command, e.Doc)
			}
		}
	})
}

// The one command that creates a document names it. Every other command was
// handed the document it worked on, and that id is already the event's doc:
// made is the id a run brought into the world.
func TestAPublishEndNamesTheDocumentItCreated(t *testing.T) {
	theHouseLogoCached(t)
	dir := os.Getenv("GDOC_CONFIG_DIR")
	stubWire(t, &fakeWire{answers: publishAnswers(t)})
	_, md := buildNote(t)

	var out bytes.Buffer
	if code := run(context.Background(),
		[]string{"publish", "--md", md, "--folder-id", testFolderID}, &out, io.Discard, presenceOn()); code != 0 {
		t.Fatalf("publish failed: exit %d, %s", code, out.String())
	}

	events := presenceEvents(t, dir)
	if len(events) != 6 {
		t.Fatalf("a publish wrote %d events, want its start, its four steps and its end: %+v", len(events), events)
	}
	start, end := events[0], endOf(t, events)
	if start.Made != "" {
		t.Errorf("the start names a document nothing had created yet: %q", start.Made)
	}
	if end.Made != publishDocID {
		t.Errorf("the end names made %q, want the id the create answered with", end.Made)
	}
	if end.Doc != "" {
		t.Errorf("a publish was handed no document, and its event names doc %q", end.Doc)
	}
}

// A publish whose pairing failed takes the document back, and a document that
// is not there any more is not one to name: the envelope clears the id and the
// record carries none either.
func TestARolledBackPublishNamesNoDocument(t *testing.T) {
	theHouseLogoCached(t)
	dir := os.Getenv("GDOC_CONFIG_DIR")
	_, md := buildNote(t)
	answers := publishAnswers(t)
	answers[0].before = func() {
		if err := os.Remove(md); err != nil {
			t.Fatal(err)
		}
	}
	stubWire(t, &fakeWire{answers: append(answers, trashAnswers()...)})

	var out bytes.Buffer
	if code := run(context.Background(),
		[]string{"publish", "--md", md, "--folder-id", testFolderID}, &out, io.Discard, presenceOn()); code == 0 {
		t.Fatalf("a note that cannot be paired is not a clean publish: %s", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte(`"rolled_back":true`)) {
		t.Fatalf("the document was not taken back, so this test proves nothing: %s", out.String())
	}

	end := endOf(t, presenceEvents(t, dir))
	if end.Made != "" {
		t.Errorf("the end names made %q, and the document it names has gone", end.Made)
	}
	if end.OK == nil || *end.OK {
		t.Errorf("the end of a rolled back publish says ok %v", end.OK)
	}
}

// made is publish's alone. Every other command is handed its document, so an
// answer that carried a document_id by any other route still writes none.
func TestOnlyPublishCarriesMade(t *testing.T) {
	res := emit.Result{OK: true, Data: publishData{DocumentID: publishDocID}}
	for _, c := range commands() {
		made := presenceMade(c, res)
		if c.name == "publish" {
			if made != publishDocID {
				t.Errorf("publish's end names made %q, want the id in its data", made)
			}
			continue
		}
		if made != "" {
			t.Errorf("%s's end names made %q, and it creates no document", c.name, made)
		}
	}

	// And nothing that is not a Drive id reaches the record, whatever a
	// command's data carries under that key.
	for _, odd := range []any{
		nil,
		map[string]any{"document_id": "/Users/nail/notes/supplier.md"},
		publishData{DocumentID: "no"},
		publishData{},
	} {
		if made := presenceMade(publishCommand(t), emit.Result{OK: false, Data: odd}); made != "" {
			t.Errorf("data %v put %q in the record", odd, made)
		}
	}
}

// publishCommand is the table's own publish entry.
func publishCommand(t *testing.T) command {
	t.Helper()
	for _, c := range commands() {
		if c.name == "publish" {
			return c
		}
	}
	t.Fatal("the table carries no publish")
	return command{}
}
