package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The exclusion lists, as literals. They are the whole of what a reader has to
// take on trust about the mapping: everything else is checked against the table
// and against the schema.
//
// flagsNeverOffered are the flags a chat caller never fills. --wait would hold
// a chat turn open for nine minutes; --md and --folder reach a hub and a Drive
// folder, and nothing in chat does either; --quote and --body-file are
// annotate's by-hand form, where chat sends the list instead. reply's
// --body-file is on the list because no property is named after it: the body
// property becomes it.
var flagsNeverOffered = map[string][]string{
	"read":        nil,
	"comments":    {"--wait"},
	"suggestions": {"--md"},
	"reply":       {"--body-file"},
	"annotate":    {"--quote", "--body-file"},
	"propose":     {"--md", "--folder"},
}

// propertiesThatMapToNoFlag are the properties that fill nothing on the line,
// because they are a check this server makes and no command of the terminal has.
// code is the guide code, read before the line is built. Task 12 adds title and
// thread_quote.
var propertiesThatMapToNoFlag = map[string][]string{
	"read":        {"code"},
	"comments":    {"code"},
	"suggestions": {"code"},
	"reply":       {"code"},
	"annotate":    {"code"},
	"propose":     {"code"},
}

// schemaProperties is the property names of a tool's schema, in the order the
// schema writes them, which is the order a card draws them.
func schemaProperties(t *testing.T, schema string) []string {
	t.Helper()
	var shape struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal([]byte(schema), &shape); err != nil {
		t.Fatalf("the schema is not JSON: %v", err)
	}
	out := make([]string, 0, len(shape.Properties))
	for name := range shape.Properties {
		out = append(out, name)
	}
	return out
}

// Every property of every schema fills a word or a flag of that tool's own
// table entry, and every word and flag of that entry is filled or is on the
// tool's own exclusion list. Both directions, because a mapping is wrong in
// either: a property nothing reads is a field a card shows and gdoc drops, and
// a flag nothing fills is a thing the terminal can do that chat silently
// cannot.
func TestEverySchemaPropertyMapsToAWordOrFlagAndBack(t *testing.T) {
	for _, c := range mcpCommands() {
		entry := match(strings.Fields(c.tool))
		if entry == nil {
			t.Errorf("%s is offered as a tool and is no command of the table", c.tool)
			continue
		}

		filled := map[string]bool{}
		mapped := map[string]bool{}
		for _, prop := range c.words {
			mapped[prop] = true
		}
		for _, f := range c.flags {
			mapped[f.prop] = true
			filled[f.flag] = true
		}

		// Forward: every property is read.
		for _, prop := range schemaProperties(t, c.schema) {
			if mapped[prop] {
				continue
			}
			if contains(propertiesThatMapToNoFlag[c.tool], prop) {
				continue
			}
			t.Errorf("%s's schema carries %q, and nothing on the line is filled from it", c.tool, prop)
		}
		// And back: every mapping names a property the schema really has.
		have := map[string]bool{}
		for _, prop := range schemaProperties(t, c.schema) {
			have[prop] = true
		}
		for prop := range mapped {
			if !have[prop] {
				t.Errorf("%s fills the line from %q, which its schema does not offer", c.tool, prop)
			}
		}

		// The words are the entry's own, in its own number and order.
		if len(c.words) != len(entry.words) {
			t.Errorf("%s takes %d words and the tool fills %d: %v", c.tool, len(entry.words), len(c.words), c.words)
		}

		// Every flag of the entry is filled or named as never offered.
		for _, f := range entry.flags {
			if filled[f.name] || contains(flagsNeverOffered[c.tool], f.name) {
				continue
			}
			t.Errorf("%s takes %s and chat neither fills it nor says it is never offered", c.tool, f.name)
		}
		// And back: the exclusion list names flags the command really takes, so
		// a flag somebody renames is caught here rather than left on a list
		// nothing reads.
		for _, name := range flagsNeverOffered[c.tool] {
			if !entryHasFlag(entry, name) {
				t.Errorf("%s is on %s's never-offered list and %s takes no such flag", name, c.tool, c.tool)
			}
		}
		// And the property exclusion list names properties the schema has.
		for _, prop := range propertiesThatMapToNoFlag[c.tool] {
			if !have[prop] {
				t.Errorf("%s is on %s's property list and its schema does not offer it", prop, c.tool)
			}
		}
	}

	// The lists themselves name tools that exist, so one left behind by a
	// renamed tool fails here.
	tools := map[string]bool{}
	for _, c := range mcpCommands() {
		tools[c.tool] = true
	}
	for _, lists := range []map[string][]string{flagsNeverOffered, propertiesThatMapToNoFlag} {
		for name := range lists {
			if !tools[name] {
				t.Errorf("%q is on an exclusion list and is no tool", name)
			}
		}
	}
}

func entryHasFlag(c *command, name string) bool {
	for _, f := range c.flags {
		if f.name == name {
			return true
		}
	}
	return false
}

// Every schema is JSON a client can parse, and it says what a card needs: an
// object, with the document required.
func TestEverySchemaIsAnObjectThatRequiresTheDocument(t *testing.T) {
	for _, c := range mcpCommands() {
		var shape struct {
			Type       string          `json:"type"`
			Required   []string        `json:"required"`
			Properties json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal([]byte(c.schema), &shape); err != nil {
			t.Errorf("%s's schema is not JSON: %v", c.tool, err)
			continue
		}
		if shape.Type != "object" {
			t.Errorf("%s's schema is %q, want an object", c.tool, shape.Type)
		}
		if !contains(shape.Required, "url") {
			t.Errorf("%s names a document and its schema does not require url: %v", c.tool, shape.Required)
		}
	}
}

// A string that becomes a word on the line, or a flag value, is refused when it
// starts with a dash. The parser reads any such word as a flag, so a url of
// "--md=x" would be read as a flag rather than as a document, and a document
// nobody meant is what the whole guard exists to stop.
//
// A body and a why go to a file and never reach the line, so a dash there is
// just a dash.
func TestAWordStringThatStartsWithADashIsRefused(t *testing.T) {
	byName := map[string]mcpCommand{}
	for _, c := range mcpCommands() {
		byName[c.tool] = c
	}

	for _, bad := range []struct {
		tool string
		args string
		says string
	}{
		{"read", `{"url":"--md=x"}`, "url"},
		{"read", `{"url":"-structure"}`, "url"},
		{"comments", `{"url":"--witness"}`, "url"},
		{"comments", `{"url":"` + fixtureDocID + `","since":"-1"}`, "since"},
		{"reply", `{"url":"` + fixtureDocID + `","comment_id":"--since=x","body":"🤖 hi"}`, "comment_id"},
	} {
		files := &callFiles{}
		argv, err := byName[bad.tool].argv(json.RawMessage(bad.args), files)
		files.remove()
		if err == nil {
			t.Errorf("%s %s must be refused and became %v", bad.tool, bad.args, argv)
			continue
		}
		if !strings.Contains(err.Error(), bad.says) {
			t.Errorf("%s %s must be refused naming %q: %q", bad.tool, bad.args, bad.says, err.Error())
		}
	}

	// The same dash, in the two values that go to a file.
	for _, good := range []struct {
		tool string
		args string
	}{
		{"reply", `{"url":"` + fixtureDocID + `","comment_id":"AAAA1111","body":"-- the register, 2026"}`},
		{"annotate", `{"url":"` + fixtureDocID + `","annotations":[{"quoted":"a","why":"--maybe not"}]}`},
	} {
		files := &callFiles{}
		_, err := byName[good.tool].argv(json.RawMessage(good.args), files)
		files.remove()
		if err != nil {
			t.Errorf("%s %s goes to a file and must be taken: %v", good.tool, good.args, err)
		}
	}
}

// Every flag value is passed joined, --flag=value, so nothing on the line can
// be read as the next flag's name. A flag that carries no value is passed bare,
// which is what its table entry says it is.
func TestFlagValuesArePassedJoined(t *testing.T) {
	byName := map[string]mcpCommand{}
	for _, c := range mcpCommands() {
		byName[c.tool] = c
	}

	for _, one := range []struct {
		tool  string
		args  string
		wants []string
	}{
		{"read", `{"url":"` + fixtureDocID + `","structure":true}`, []string{"read", fixtureDocID, "--structure"}},
		{"read", `{"url":"` + fixtureDocID + `","structure":false}`, []string{"read", fixtureDocID}},
		{"comments", `{"url":"` + fixtureDocID + `","since":"2026-10-03T09:00:00Z","witness":true}`,
			[]string{"comments", fixtureDocID, "--since=2026-10-03T09:00:00Z", "--witness"}},
		{"suggestions", `{"url":"` + fixtureDocID + `"}`, []string{"suggestions", fixtureDocID}},
	} {
		files := &callFiles{}
		argv, err := byName[one.tool].argv(json.RawMessage(one.args), files)
		files.remove()
		if err != nil {
			t.Errorf("%s %s: %v", one.tool, one.args, err)
			continue
		}
		if strings.Join(argv, " ") != strings.Join(one.wants, " ") {
			t.Errorf("%s %s became %v, want %v", one.tool, one.args, argv, one.wants)
		}
	}

	// The three flags that carry a path are joined to it, and the path is in
	// the directory this call made.
	for _, one := range []struct {
		tool string
		args string
		flag string
		file string
	}{
		{"reply", `{"url":"` + fixtureDocID + `","comment_id":"AAAA1111","body":"🤖 yes"}`, "--body-file", "body.txt"},
		{"annotate", `{"url":"` + fixtureDocID + `","annotations":[{"quoted":"a","why":"b"}]}`, "--from", "annotations.json"},
		{"propose", `{"url":"` + fixtureDocID + `","proposals":` + oneProposal + `}`, "--from", "proposals.json"},
	} {
		files := &callFiles{}
		argv, err := byName[one.tool].argv(json.RawMessage(one.args), files)
		if err != nil {
			files.remove()
			t.Errorf("%s %s: %v", one.tool, one.args, err)
			continue
		}
		want := one.flag + "=" + filepath.Join(files.dir, one.file)
		if !contains(argv, want) {
			t.Errorf("%s became %v, want it to carry %q", one.tool, argv, want)
		}
		files.remove()
	}
}

// A property the schema does not carry is refused by name rather than dropped.
// A tool that quietly ignores an argument tells its caller it did something it
// did not.
func TestAnArgumentNoSchemaCarriesIsRefusedByName(t *testing.T) {
	var read mcpCommand
	for _, c := range mcpCommands() {
		if c.tool == "read" {
			read = c
		}
	}
	files := &callFiles{}
	defer files.remove()
	_, err := read.argv(json.RawMessage(`{"url":"`+fixtureDocID+`","md":"note.md"}`), files)
	if err == nil {
		t.Fatal("an argument read does not take must be refused")
	}
	if !strings.Contains(err.Error(), "md") {
		t.Errorf("the refusal names the argument: %q", err.Error())
	}
}

// A tool answer carries the envelope the terminal prints, with nothing added
// and nothing taken away, and ok: false is isError. Six commands, each against
// the same fakes twice: once through the tool and once through run.
//
// Which item the envelope is depends on the tool. A write answers with the
// envelope alone. A read that succeeded answers three items, the envelope
// second, with the fixed line before it and the wrapped copy after it:
// mcpview.go holds why, and mcplabel_test.go holds the other two items. A read
// that was refused carries no text out of a document, so it is one item again.
func TestTheSameAnswerAsTheCLI(t *testing.T) {
	byName := map[string]mcpCommand{}
	for _, c := range mcpCommands() {
		byName[c.tool] = c
	}

	for _, one := range []struct {
		tool string
		args string
		// cli is the line, with @FILE@ standing where the path of the file the
		// tool wrote goes, so the two lines are the same line.
		cli  []string
		body string
		ok   bool
		// items is how many text items the answer carries, and at is where the
		// envelope sits among them.
		items int
		at    int
		// wire says which fake the command needs.
		wire func(t *testing.T)
	}{
		{
			tool: "read", args: `{"url":"` + fixtureDocID + `"}`,
			cli: []string{"read", fixtureDocID}, ok: true, items: 3, at: 1,
			wire: func(t *testing.T) { stubSession(t, docsAndComments(t)) },
		},
		{
			tool: "comments", args: `{"url":"` + fixtureDocID + `"}`,
			cli: []string{"comments", fixtureDocID}, ok: true, items: 3, at: 1,
			wire: func(t *testing.T) { stubSession(t, docsAndComments(t)) },
		},
		{
			tool: "suggestions", args: `{"url":"` + fixtureDocID + `"}`,
			cli: []string{"suggestions", fixtureDocID}, ok: true, items: 3, at: 1,
			wire: func(t *testing.T) { stubSession(t, docsAndComments(t)) },
		},
		{
			tool: "reply", args: `{"url":"` + fixtureDocID + `","comment_id":"AAAA1111","body":"🤖 The 2026 register."}`,
			cli:  []string{"reply", fixtureDocID, "AAAA1111", "--body-file=@FILE@"},
			body: "🤖 The 2026 register.", ok: true, items: 1,
			wire: func(t *testing.T) {
				stubWire(t, &fakeWire{answers: []*answer{
					{method: "POST", match: "/comments/AAAA1111/replies", json: `{"id":"R1","createdTime":"2026-09-06T10:45:00Z","content":"🤖 The 2026 register."}`},
					{method: "GET", match: "/comments?", json: readFixture(t, "comments.json")},
				}})
			},
		},
		{
			tool: "annotate",
			args: `{"url":"` + annotateDocID + `","annotations":[{"quoted":"reviewed annually","why":"The 2026 register says quarterly."}]}`,
			cli:  []string{"annotate", annotateDocID, "--from=@FILE@"},
			body: `[{"quoted":"reviewed annually","why":"The 2026 register says quarterly."}]`, ok: true, items: 1,
			wire: func(t *testing.T) { stubWire(t, &fakeWire{answers: annotateAnswers(t)}) },
		},
		{
			tool: "propose", args: `{"url":"` + proposeDocID + `","proposals":` + oneProposal + `}`,
			cli:  []string{"propose", proposeDocID, "--from=@FILE@"},
			body: oneProposal, ok: true, items: 1,
			wire: func(t *testing.T) { stubWire(t, &fakeWire{answers: proposeAnswers(t, true)}) },
		},
		{
			// The refusal too: a document nobody can read answers the same way
			// through both doors.
			tool: "read", args: `{"url":"not a document"}`,
			cli: []string{"read", "not a document"}, ok: false, items: 1,
			wire: func(t *testing.T) { stubSession(t, docsAndComments(t)) },
		},
	} {
		t.Run(one.tool+"/"+boolWord(one.ok), func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
			signedIn(t)

			one.wire(t)
			guideCode := callCode(t)
			res := mcpRun(context.Background(), byName[one.tool], withCode(t, guideCode, one.args), nilWriter{}, guideCode)
			if len(res.Texts) != one.items {
				t.Fatalf("the answer carries %d text items, want %d: %v", len(res.Texts), one.items, res.Texts)
			}
			if res.IsError == one.ok {
				t.Errorf("isError = %v for an answer that is ok = %v", res.IsError, one.ok)
			}

			one.wire(t)
			line := make([]string, 0, len(one.cli))
			var path string
			if one.body != "" {
				path = tempFile(t, "body", one.body)
			}
			for _, word := range one.cli {
				line = append(line, strings.ReplaceAll(word, "@FILE@", path))
			}
			var buf strings.Builder
			code := run(context.Background(), line, &buf, nilWriter{})
			if (code == 0) != one.ok {
				t.Fatalf("the CLI exited %d for an answer that is ok = %v: %s", code, one.ok, buf.String())
			}

			if got, want := normalisePaths(res.Texts[one.at], path), normalisePaths(buf.String(), path); got != want {
				t.Errorf("the tool answered\n  %s\nand the CLI answered\n  %s", got, want)
			}
		})
	}
}

func boolWord(ok bool) string {
	if ok {
		return "ok"
	}
	return "refused"
}

// normalisePaths takes the two temp paths out of an envelope: the one the tool
// made for the call and the one the test wrote for the CLI. Everything else has
// to be the same byte for byte.
func normalisePaths(text, cliPath string) string {
	if cliPath != "" {
		text = strings.ReplaceAll(text, cliPath, "<file>")
		text = strings.ReplaceAll(text, filepath.Dir(cliPath), "<dir>")
	}
	return text
}

// nilWriter is stderr thrown away. The log of a session goes there and says
// nothing about the answer.
type nilWriter struct{}

func (nilWriter) Write(p []byte) (int, error) { return len(p), nil }

// Each call makes its own directory, under a name carrying this process's id
// and a mode only this user can read, and the directory is gone when the call
// is: after an answer, after a refusal and after a panic.
func TestTempFilesAreMadeForTheCallAndGone(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)

	var reply mcpCommand
	for _, c := range mcpCommands() {
		if c.tool == "reply" {
			reply = c
		}
	}

	files := &callFiles{}
	argv, err := reply.argv(json.RawMessage(`{"url":"`+fixtureDocID+`","comment_id":"AAAA1111","body":"🤖 yes"}`), files)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(files.dir)
	if err != nil {
		t.Fatalf("the call's directory: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Errorf("the call's directory is mode %o, want 700", got)
	}
	if base := filepath.Base(files.dir); !strings.HasPrefix(base, mcpTempPrefix()) {
		t.Errorf("the call's directory is %q, want it to open with %q", base, mcpTempPrefix())
	}
	// The file name is fixed and never built from an argument.
	if !contains(argv, "--body-file="+filepath.Join(files.dir, "body.txt")) {
		t.Errorf("argv = %v, want the fixed file name", argv)
	}
	dir := files.dir
	files.remove()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the call's directory outlived the call: %v", err)
	}

	// Through a whole call, three ways out. The directory each one made is
	// found by watching how many are there afterwards.
	before := countCallDirs(t)
	stubWire(t, &fakeWire{answers: []*answer{
		{method: "POST", match: "/comments/AAAA1111/replies", json: `{"id":"R1","createdTime":"2026-09-06T10:45:00Z","content":"🤖 yes"}`},
		{method: "GET", match: "/comments?", json: readFixture(t, "comments.json")},
	}})
	code := callCode(t)
	mcpRun(context.Background(), reply, withCode(t, code, `{"url":"`+fixtureDocID+`","comment_id":"AAAA1111","body":"🤖 yes"}`), nilWriter{}, code)
	stubWire(t, &fakeWire{})
	mcpRun(context.Background(), reply, withCode(t, code, `{"url":"`+fixtureDocID+`","comment_id":"AAAA1111","body":"not the robot"}`), nilWriter{}, code)
	if after := countCallDirs(t); after != before {
		t.Errorf("%d call directories before and %d after, so a call left one behind", before, after)
	}

	// A panic is the third way out. The remove is deferred, so it runs while the
	// panic is on its way up, which is how mcpRun survives one.
	before = countCallDirs(t)
	func() {
		defer func() { _ = recover() }()
		files := &callFiles{}
		defer files.remove()
		if _, err := files.write("body.txt", "body", "x"); err != nil {
			t.Fatal(err)
		}
		panic("the call failed in a way nothing expected")
	}()
	if after := countCallDirs(t); after != before {
		t.Errorf("%d call directories before the panic and %d after", before, after)
	}
}

// An error naming the file the server wrote names the argument instead. The
// path is a path nobody typed and nobody can open, and the argument is what the
// model has to fix.
func TestAnErrorNamingTheTempPathNamesTheArgument(t *testing.T) {
	files := &callFiles{}
	defer files.remove()
	path, err := files.write("annotations.json", "annotations", "[]")
	if err != nil {
		t.Fatal(err)
	}
	got := files.name(path + " carries no annotations, so there is nothing to write")
	if strings.Contains(got, path) {
		t.Errorf("the path is still in the sentence: %q", got)
	}
	if !strings.HasPrefix(got, "annotations ") {
		t.Errorf("the sentence must open with the argument: %q", got)
	}
}

// countCallDirs is how many call directories this process has in the temp
// directory now. Another test's directory would be another process's.
func countCallDirs(t *testing.T) int {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), mcpTempPrefix()+"*"))
	if err != nil {
		t.Fatal(err)
	}
	return len(matches)
}

// Each call opens its own policy, so a grant one call carried is absent from
// the next. Nothing is remembered between calls: the policy is built inside the
// command and dies with it, which is the invariant a grant names one object and
// dies with the process, read a call at a time.
func TestAGrantFromOneCallIsAbsentFromTheNext(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)

	var read mcpCommand
	for _, c := range mcpCommands() {
		if c.tool == "read" {
			read = c
		}
	}
	const otherDocID = "1OtHeRdOcUmEnT0000000000000000000000000000"

	code := callCode(t)
	first := stubSession(t, docsAndComments(t))
	mcpRun(context.Background(), read, withCode(t, code, `{"url":"`+fixtureDocID+`"}`), nilWriter{}, code)
	second := stubSession(t, docsAndComments(t))
	mcpRun(context.Background(), read, withCode(t, code, `{"url":"`+otherDocID+`"}`), nilWriter{}, code)

	if first.policy == nil || second.policy == nil {
		t.Fatal("both calls open a policy")
	}
	if first.policy == second.policy {
		t.Fatal("the two calls shared one policy")
	}
	allowed := mustParse(t, "https://docs.googleapis.com/v1/documents/"+fixtureDocID+"?includeTabsContent=true")
	if err := first.policy.Judge("GET", allowed, nil); err != nil {
		t.Errorf("the first call reaches the document it named: %v", err)
	}
	if err := second.policy.Judge("GET", allowed, nil); err == nil {
		t.Error("the second call must not reach the document the first one named")
	}
}

// An array argument reaches the file as its own JSON, byte for byte. The
// commands' readers are strict, and a value the server re-encoded would be
// judged as the server's JSON rather than as what the model wrote.
func TestArraysPassThroughAsTheirRawJSON(t *testing.T) {
	var propose mcpCommand
	for _, c := range mcpCommands() {
		if c.tool == "propose" {
			propose = c
		}
	}
	files := &callFiles{}
	defer files.remove()

	const list = `[{"quoted":"reviewed annually","replacement":"reviewed every six months","why":"the policy says twice a year"}]`
	if _, err := propose.argv(json.RawMessage(`{"url":"`+proposeDocID+`","proposals":`+list+`}`), files); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(files.dir, "proposals.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != list {
		t.Errorf("the file holds\n  %s\nand the argument was\n  %s", raw, list)
	}

	// An empty list is refused before any session is opened: a run that writes
	// nothing is not a run worth opening somebody's document for.
	empty := &callFiles{}
	defer empty.remove()
	if _, err := propose.argv(json.RawMessage(`{"url":"`+proposeDocID+`","proposals":[]}`), empty); err == nil {
		t.Error("an empty list must be refused")
	}
}
