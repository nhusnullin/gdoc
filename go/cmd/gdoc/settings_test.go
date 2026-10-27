// The three setting flags on the line, one of them experimental: taken off it wherever they stand, turned
// into what this run writes, and handed down to every command and to the
// session.
//
// Every mark here is a literal. A test that read plaintext.DefaultMark back
// would follow the default wherever somebody moved it and say nothing about
// what a colleague's line does.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"gdoc/internal/emit"
	"gdoc/internal/settings"
)

// defaultsInEffect is what a run that was told nothing writes, as the tests
// that care about something else hand it to run and to a session. The mark is
// the literal, not the constant: this is the value a colleague who set nothing
// gets, and a test that read the constant would agree with any change to it.
func defaultsInEffect() settings.InEffect {
	return settings.InEffect{Mark: "⁘", MarkFrom: settings.FromDefault}
}

// robotInEffect is a run that writes the robot, named on the line. It is what
// the tests over the recorded comment fixtures hand to run: those fixtures
// carry the robot, because they were recorded when the robot was the only mark
// there was, and a read-back is asked about the string that was sent. So the
// test says which mark it means rather than inheriting the default.
func robotInEffect() settings.InEffect {
	return settings.InEffect{Mark: "🤖", MarkFrom: settings.FromArgument}
}

// quietPlain reaches nothing and says so without failing the test. It is what
// a run that may touch GitHub is handed where the reach is not what is being
// measured: refusingPlain of notice_test fails the test instead, which is the
// right answer for a run that must ask nothing.
type quietPlain struct{}

func (quietPlain) GetJSON(_ context.Context, _ string, _ any) error {
	return errors.New("this test reaches nothing")
}

func (quietPlain) GetBytes(_ context.Context, _ string, _ int64) ([]byte, error) {
	return nil, errors.New("this test reaches nothing")
}

func (quietPlain) Warnings() []string { return nil }

// watchingDispatch stands in for the dispatch under route, so a test can read
// the settings a line resolved to without running a command. It answers an
// envelope that says the run was fine, because what is being measured is the
// value, not the work.
func watchingDispatch(t *testing.T) *watchedDispatch {
	t.Helper()
	w := &watchedDispatch{}
	old := dispatching
	dispatching = func(_ context.Context, args []string, _ io.Writer, _ *emit.Lines, set settings.InEffect) emit.Result {
		w.record(args, set)
		return emit.Result{OK: true}
	}
	t.Cleanup(func() { dispatching = old })
	return w
}

// watchedDispatch is every call that reached the dispatch, under a lock
// because a session dispatches from the server's own goroutine.
type watchedDispatch struct {
	mu    sync.Mutex
	lines [][]string
	sets  []settings.InEffect
}

func (w *watchedDispatch) record(args []string, set settings.InEffect) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lines = append(w.lines, append([]string(nil), args...))
	w.sets = append(w.sets, set)
}

func (w *watchedDispatch) calls() ([][]string, []settings.InEffect) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lines, w.sets
}

// routed runs one line through route with the dispatch watched, and answers the
// settings it resolved to, the line that was left and the object on stdout.
func routed(t *testing.T, args ...string) (settings.InEffect, []string, map[string]any, int) {
	t.Helper()
	w := watchingDispatch(t)
	var out, errOut bytes.Buffer
	code := route(context.Background(), args, strings.NewReader(""), &out, &errOut)
	lines, sets := w.calls()
	if len(sets) != 1 {
		return settings.InEffect{}, nil, decodeOne(t, &out), code
	}
	return sets[0], lines[0], nil, code
}

// Every command takes the three flags, and none of them refuses one as a flag it
// does not take. They are off the line before any command's own parser sees
// it, so the refusal a command gives is its own: a missing document, a flag it
// needs, or no token on this machine.
func TestEveryCommandTakesTheSettingFlags(t *testing.T) {
	file := writeSettingsFile(t, "mark: \"∴\"\n")
	for _, c := range commands() {
		t.Run(c.name, func(t *testing.T) {
			checking(t, "", quietPlain{})
			stubLogin(t, func(io.Writer) error { return errors.New("this test signs nobody in") })
			saved := serveMCP
			t.Cleanup(func() { serveMCP = saved })
			serveMCP = func(context.Context, io.Reader, io.Writer, io.Writer, []string, settings.InEffect) int {
				return 0
			}

			args := append(c.nameWords(), configFlag, file, markFlag, "∴", presenceFlag, "false")
			var out, errOut bytes.Buffer
			route(context.Background(), args, strings.NewReader(""), &out, &errOut)
			said := out.String() + errOut.String()
			for _, name := range []string{configFlag, markFlag, presenceFlag} {
				if strings.Contains(said, name+"\\\" is not a flag") || strings.Contains(said, name+"\" is not a flag") {
					t.Errorf("%s refused %s as a flag of its own: %s", c.name, name, said)
				}
			}
		})
	}
}

// writeSettingsFile is a settings file of this test's own, named by --config.
func writeSettingsFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// The flags stand wherever a person put them: before the command, between its
// words, or after everything. What is left of the line is the command and its
// own arguments, in the order they were written.
func TestTheSettingFlagsAreTakenAnywhereOnTheLine(t *testing.T) {
	file := writeSettingsFile(t, "mark: \"🤖\"\n")
	for _, args := range [][]string{
		{markFlag, "∴", configFlag, file, "read", "https://docs.google.com/document/d/1AbC/edit"},
		{"read", markFlag, "∴", "https://docs.google.com/document/d/1AbC/edit", configFlag, file},
		{"read", "https://docs.google.com/document/d/1AbC/edit", markFlag + "=∴", configFlag + "=" + file},
		{presenceFlag, "true", "read", markFlag, "∴", "https://docs.google.com/document/d/1AbC/edit", configFlag + "=" + file},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

			set, line, _, code := routed(t, args...)
			if code != 0 {
				t.Fatalf("%v reached no command: exit %d", args, code)
			}
			if set.Mark != "∴" {
				t.Errorf("the argument is the mark in effect: got %q, want %q", set.Mark, "∴")
			}
			if set.MarkFrom != settings.FromArgument {
				t.Errorf("the mark came from the line: got %q", set.MarkFrom)
			}
			if set.File != file {
				t.Errorf("the file in use is the one --config named: got %q, want %q", set.File, file)
			}
			want := []string{"read", "https://docs.google.com/document/d/1AbC/edit"}
			if len(line) != len(want) || !equalWords(line, want) {
				t.Errorf("what is left of the line is %v, want %v", line, want)
			}
		})
	}
}

// --mark "" and --mark= are the empty mark, which is a colleague's choice and
// not a flag nobody typed.
func TestAnEmptyMarkArgumentIsTheEmptyMark(t *testing.T) {
	for _, args := range [][]string{
		{"read", markFlag, "", "https://docs.google.com/document/d/1AbC/edit"},
		{"read", markFlag + "=", "https://docs.google.com/document/d/1AbC/edit"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

			set, _, _, code := routed(t, args...)
			if code != 0 {
				t.Fatalf("%v reached no command: exit %d", args, code)
			}
			if set.Mark != "" {
				t.Errorf("the empty mark was asked for and the mark in effect is %q", set.Mark)
			}
			if set.MarkFrom != settings.FromArgument {
				t.Errorf("the empty mark came from the line: got %q", set.MarkFrom)
			}
		})
	}
}

// A setting flag given twice is refused naming it, as a repeated flag is
// everywhere else: which one counts is not decided here.
func TestASettingFlagGivenTwiceIsRefused(t *testing.T) {
	file := writeSettingsFile(t, "")
	for _, c := range []struct {
		args []string
		says string
	}{
		{[]string{"read", markFlag, "∴", markFlag, "🤖", "1AbC"}, markFlag},
		{[]string{"read", markFlag, "∴", markFlag + "=🤖", "1AbC"}, markFlag},
		{[]string{"read", configFlag, file, configFlag + "=" + file, "1AbC"}, configFlag},
		{[]string{"read", presenceFlag, "true", presenceFlag + "=false", "1AbC"}, presenceFlag},
	} {
		t.Run(strings.Join(c.args, " "), func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

			_, _, got, code := routed(t, c.args...)
			if code != 1 {
				t.Fatalf("%v is refused and exits 1, and this run exited %d", c.args, code)
			}
			if got["ok"] != false {
				t.Fatalf("the object must say the line was refused: %v", got)
			}
			if say, _ := got["error"].(string); !strings.Contains(say, c.says) || !strings.Contains(say, "twice") {
				t.Errorf("the refusal must name %s and say it was given twice: %q", c.says, say)
			}
		})
	}
}

// A setting flag with nothing after it, and one whose value is the other
// setting flag, are both refused naming the flag. Without the second refusal
// `--mark --config f` would write "--config" in front of gdoc's words and read
// the file nobody named.
func TestASettingFlagWithNoValueIsRefused(t *testing.T) {
	for _, c := range []struct {
		args []string
		says string
	}{
		{[]string{"read", "1AbC", markFlag}, "none follows it"},
		{[]string{"read", "1AbC", configFlag}, "none follows it"},
		{[]string{"read", "1AbC", markFlag, configFlag, "f"}, "another flag"},
		{[]string{"read", "1AbC", presenceFlag}, "none follows it"},
		{[]string{"read", "1AbC", presenceFlag, markFlag, "∴"}, "another flag"},
		{[]string{"read", "1AbC", presenceFlag, "yes"}, "true or false"},
		{[]string{"read", "1AbC", presenceFlag + "="}, "true or false"},
		{[]string{"read", "1AbC", configFlag, ""}, "empty value"},
		{[]string{"read", "1AbC", configFlag + "="}, "empty value"},
	} {
		t.Run(strings.Join(c.args, " "), func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

			_, _, got, code := routed(t, c.args...)
			if code != 1 {
				t.Fatalf("%v is refused and exits 1, and this run exited %d", c.args, code)
			}
			if say, _ := got["error"].(string); !strings.Contains(say, c.says) {
				t.Errorf("the refusal must say %q: %q", c.says, say)
			}
		})
	}
}

// A settings file gdoc cannot make sense of, and a mark a thread would show
// wrong, are one envelope on stdout with exit 1: the same answer every other
// refusal gives, so a skill reads what was wrong with the line it built.
func TestABadSettingsFileIsOneEnvelope(t *testing.T) {
	for _, c := range []struct {
		name string
		body string
		args func(path string) []string
		says string
	}{
		{name: "unknown key", body: "colour: red\n",
			args: func(p string) []string { return []string{"read", "1AbC", configFlag, p} },
			says: "colour"},
		{name: "mark with no value", body: "mark:\n",
			args: func(p string) []string { return []string{"read", "1AbC", configFlag, p} },
			says: "mark"},
		{name: "markdown in the file", body: "mark: \"**\"\n",
			args: func(p string) []string { return []string{"read", "1AbC", configFlag, p} },
			says: "**"},
		{name: "markdown on the line", body: "",
			args: func(p string) []string { return []string{"read", "1AbC", configFlag, p, markFlag, "**"} },
			says: markFlag},
		{name: "presence with no value", body: "presence:\n",
			args: func(p string) []string { return []string{"read", "1AbC", configFlag, p} },
			says: "presence"},
		{name: "a presence the file wrote that is no boolean", body: "presence: yes\n",
			args: func(p string) []string { return []string{"read", "1AbC", configFlag, p} },
			says: "true or false"},
		{name: "a presence the line wrote that is no boolean", body: "",
			args: func(p string) []string {
				return []string{"read", "1AbC", configFlag, p, presenceFlag, "True"}
			},
			says: presenceFlag},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
			path := writeSettingsFile(t, c.body)

			var out, errOut bytes.Buffer
			code := route(context.Background(), c.args(path), strings.NewReader(""), &out, &errOut)
			if code != 1 {
				t.Fatalf("%s is refused and exits 1, and this run exited %d", c.name, code)
			}
			got := decodeOne(t, &out)
			if got["ok"] != false {
				t.Fatalf("the object must say the run was refused: %v", got)
			}
			if say, _ := got["error"].(string); !strings.Contains(say, c.says) {
				t.Errorf("the refusal must name %q: %q", c.says, say)
			}
		})
	}
}

// A --config that names no file is refused on the line, because a mistyped
// path read as every default would hand a colleague the wrong mark and say
// nothing.
func TestAConfigFileThatDoesNotExistIsRefusedOnTheLine(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	missing := filepath.Join(t.TempDir(), "nowhere", "settings.yaml")

	var out, errOut bytes.Buffer
	code := route(context.Background(),
		[]string{"read", "1AbC", configFlag, missing}, strings.NewReader(""), &out, &errOut)
	if code != 1 {
		t.Fatalf("a --config naming nothing is refused and exits 1, and this run exited %d", code)
	}
	got := decodeOne(t, &out)
	if got["ok"] != false {
		t.Fatalf("the object must say the run was refused: %v", got)
	}
	say, _ := got["error"].(string)
	if !strings.Contains(say, missing) {
		t.Errorf("the refusal must name the path that is not there: %q", say)
	}
}

// The default settings file is read where nobody named one, and its absence is
// every default.
func TestTheDefaultSettingsFileIsReadFromTheConfigFolder(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)

	set, _, _, code := routed(t, "read", "1AbC")
	if code != 0 {
		t.Fatalf("a run with no settings file reaches its command: exit %d", code)
	}
	if set.Mark != "⁘" || set.MarkFrom != settings.FromDefault {
		t.Errorf("no file is every default: got %q from %q", set.Mark, set.MarkFrom)
	}

	if err := os.WriteFile(filepath.Join(dir, "settings.yaml"), []byte("mark: \"∴\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	set, _, _, code = routed(t, "read", "1AbC")
	if code != 0 {
		t.Fatalf("a run with a settings file reaches its command: exit %d", code)
	}
	if set.Mark != "∴" || set.MarkFrom != settings.FromFile {
		t.Errorf("the file in the config folder is read: got %q from %q", set.Mark, set.MarkFrom)
	}
}

// mcp takes the setting flags and nothing else. The flags come off the
// line before mcp is routed, so what is left for the session to refuse is what
// it always refused, on stderr with exit 1.
func TestMcpTakesTheSettingFlagsAndNothingElse(t *testing.T) {
	t.Run("the flags start a session", func(t *testing.T) {
		t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
		file := writeSettingsFile(t, "mark: \"🤖\"\n")

		saved := serveMCP
		t.Cleanup(func() { serveMCP = saved })
		var got settings.InEffect
		var rest []string
		served := false
		serveMCP = func(_ context.Context, _ io.Reader, _, _ io.Writer, args []string, set settings.InEffect) int {
			served, rest, got = true, append([]string(nil), args...), set
			return 0
		}

		var out, errOut bytes.Buffer
		code := route(context.Background(),
			[]string{"mcp", markFlag, "∴", configFlag, file, presenceFlag, "true"}, strings.NewReader(""), &out, &errOut)
		if code != 0 {
			t.Fatalf("gdoc mcp with the setting flags starts, and this run exited %d: %s", code, errOut.String())
		}
		if !served {
			t.Fatal("the session was never served")
		}
		if len(rest) != 0 {
			t.Errorf("nothing is left of the line for the session to refuse: %v", rest)
		}
		if got.Mark != "∴" || got.MarkFrom != settings.FromArgument {
			t.Errorf("the session was handed %q from %q", got.Mark, got.MarkFrom)
		}
		if !got.Presence || got.PresenceFrom != settings.FromArgument {
			t.Errorf("the session was handed the presence %v from %q", got.Presence, got.PresenceFrom)
		}
	})

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
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

			var out, errOut bytes.Buffer
			code := route(context.Background(), append([]string{"mcp"}, bad.args...),
				strings.NewReader(""), &out, &errOut)
			if code != 1 {
				t.Errorf("mcp %v must be refused with exit 1: got %d", bad.args, code)
			}
			if out.Len() != 0 {
				t.Errorf("nothing reaches stdout on a refusal: %q", out.String())
			}
			if !strings.Contains(errOut.String(), bad.says) {
				t.Errorf("the refusal must name %q: %q", bad.says, errOut.String())
			}
		})
	}

	// A setting flag the reading itself refuses is the same line on stderr. The
	// flags come off before the word mcp is looked for, so a refusal that threw
	// the line away would leave route with nothing to recognise and the reason
	// would go out as a JSON envelope on the protocol stream, which the host
	// reads as a message and a person never reads at all.
	//
	// The last three put a second setting flag between the refused one and the
	// word mcp. A reading that stopped at the first refusal would leave that
	// flag in front of the command, and the word mcp would never be found.
	for _, bad := range [][]string{
		{"mcp", markFlag},
		{"mcp", markFlag, "∴", markFlag, "🤖"},
		{"mcp", configFlag + "="},
		{"mcp", markFlag, configFlag, "settings.yaml"},
		{configFlag + "=", "mcp"},
		{configFlag + "=", markFlag, "∴", "mcp"},
		{markFlag, configFlag, "settings.yaml", "mcp"},
		{markFlag, "∴", markFlag, "🤖", configFlag + "=", "mcp"},
	} {
		t.Run(strings.Join(bad, " "), func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

			var out, errOut bytes.Buffer
			code := route(context.Background(), bad, strings.NewReader(""), &out, &errOut)
			if code != 1 {
				t.Errorf("gdoc %v must be refused with exit 1: got %d", bad, code)
			}
			if out.Len() != 0 {
				t.Errorf("a session prints no envelope, so nothing reaches stdout: %q", out.String())
			}
			if !strings.Contains(errOut.String(), markFlag) && !strings.Contains(errOut.String(), configFlag) {
				t.Errorf("the refusal must name the flag: %q", errOut.String())
			}
			if lines := strings.Count(strings.TrimRight(errOut.String(), "\n"), "\n"); lines != 0 {
				t.Errorf("a session's refusal is one line: %q", errOut.String())
			}
		})
	}

	t.Run("a bad mark is one line on stderr", func(t *testing.T) {
		t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

		var out, errOut bytes.Buffer
		code := route(context.Background(), []string{"mcp", markFlag, "**"}, strings.NewReader(""), &out, &errOut)
		if code != 1 {
			t.Fatalf("a mark a thread shows wrong is refused with exit 1: got %d", code)
		}
		if out.Len() != 0 {
			t.Errorf("a session prints no envelope, so nothing reaches stdout: %q", out.String())
		}
		if !strings.Contains(errOut.String(), markFlag) {
			t.Errorf("the refusal must name the flag: %q", errOut.String())
		}
	})
}

// The session resolves its settings once, at the start, and every tool call
// goes out under the same value. A session cannot be handed a new mark
// halfway: there is no flag on a tool call to carry one.
func TestTheMcpSessionHandsItsSettingsToEveryCall(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	w := watchingDispatch(t)

	d := startSessionWith(t, settings.InEffect{Mark: "∴", MarkFrom: settings.FromArgument})
	d.request("initialize", `{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"a test","version":"1"}}`)
	d.notify("notifications/initialized")
	code := codeFromGuide(t, d)

	for _, url := range []string{fixtureDocID, "1BcD" + strings.Repeat("0", 20)} {
		d.call("read", argsWithCode(t, code, `{"url":"`+url+`"}`))
	}
	d.end()

	_, sets := w.calls()
	if len(sets) != 2 {
		t.Fatalf("two reads reached the dispatch, and %d did", len(sets))
	}
	for i, set := range sets {
		if set.Mark != "∴" || set.MarkFrom != settings.FromArgument {
			t.Errorf("call %d went out under %q from %q", i+1, set.Mark, set.MarkFrom)
		}
	}
}

// codeFromGuide is the session's code, which every other tool needs.
func codeFromGuide(t *testing.T, d *mcpDriver) string {
	t.Helper()
	guide := d.call("guide", `{}`)
	if guide.IsError {
		t.Fatalf("guide was refused: %v", guide.texts())
	}
	var guided struct {
		Data struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(guide.texts()[0]), &guided); err != nil {
		t.Fatalf("the guide answer is no envelope: %v", err)
	}
	if guided.Data.Code == "" {
		t.Fatalf("guide gave no code: %s", guide.texts()[0])
	}
	return guided.Data.Code
}

// Every setting flag a reader is offered is in the help a person reads and in
// both completion scripts, so a flag every command takes is a flag every
// command offers. An experimental flag is offered nowhere and is skipped here;
// TestAnExperimentalFlagIsInTheTableAndInNoScreen is the other half, and it
// holds that it reaches no screen and no script at all.
func TestEverySettingFlagIsInHelpAndInBothScripts(t *testing.T) {
	zsh, err := zshScript(commands())
	if err != nil {
		t.Fatal(err)
	}
	bash, err := bashScript(commands())
	if err != nil {
		t.Fatal(err)
	}
	zshBody, bashBody := withoutComments(zsh), withoutComments(bash)
	prose := helpProse(commands(), true) + helpProse(helpMatches([]string{"publish"}), false)

	for _, f := range settingFlags() {
		if f.experimental {
			continue
		}
		if !strings.Contains(prose, f.name) {
			t.Errorf("the help must name %s: %q", f.name, prose)
		}
		if !strings.Contains(prose, f.summary) {
			t.Errorf("the help must carry the sentence for %s", f.name)
		}
		if !strings.Contains(zshBody, f.name+"[") {
			t.Errorf("the zsh script must offer %s after a command", f.name)
		}
		if !strings.Contains(bashBody, f.name) {
			t.Errorf("the bash script must offer %s after a command", f.name)
		}
	}
	if !strings.Contains(prose, settingHeading) {
		t.Errorf("the help must say the flags are not one command's own: %q", prose)
	}
}

// The --mark sentence says what the empty mark costs, because a colleague who
// turns the mark off is turning off the only record of authorship a thread has.
func TestTheMarkFlagSaysWhatAnEmptyMarkCosts(t *testing.T) {
	var found bool
	for _, f := range settingFlags() {
		if f.name != markFlag {
			continue
		}
		found = true
		if !strings.Contains(f.summary, "an empty mark writes the words alone") {
			t.Errorf("the sentence must say what an empty mark writes: %q", f.summary)
		}
		if !strings.Contains(f.summary, "nothing in the thread says which words are gdoc's") {
			t.Errorf("the sentence must say what the empty mark costs: %q", f.summary)
		}
	}
	if !found {
		t.Fatalf("%s is no setting flag", markFlag)
	}
}
