package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/guard"
)

// signedInToken is a token file in the shape auth.Load accepts: the three
// fields google-auth insists on, and an expiry far enough away that nothing
// tries to refresh it. No test here reaches Google, so the values are words.
const signedInToken = `{"token":"ya29.not-a-real-token",` +
	`"refresh_token":"1//not-a-real-refresh",` +
	`"client_id":"not-a-real-client.apps.googleusercontent.com",` +
	`"client_secret":"not-a-real-secret",` +
	`"token_uri":"https://oauth2.googleapis.com/token",` +
	`"scopes":["https://www.googleapis.com/auth/drive"],` +
	`"expiry":"2099-01-01T00:00:00Z"}`

// signedIn writes that token into the config directory the test already set,
// so a tool call gets as far as the command. Every chat tool reads the token
// file before it sends anything, and a test about the mapping or the temp
// files is not a test about being signed out.
func signedIn(t *testing.T) {
	t.Helper()
	dir := os.Getenv("GDOC_CONFIG_DIR")
	if dir == "" {
		t.Fatal("signedIn needs GDOC_CONFIG_DIR set to a directory of this test's own")
	}
	writeTokenFile(t, dir, signedInToken)
}

func writeTokenFile(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "oauth-token.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// opens counts the sessions a run opened. A tool that refused before dispatch
// opened none, which is the half of "nothing is sent" a fake wire cannot show:
// a session nobody opened made no request to record.
type opens struct{ n int }

func (o *opens) stub(t *testing.T) {
	t.Helper()
	old := openSession
	openSession = func(p *guard.Policy) (session, error) {
		o.n++
		return docsAndComments(t), nil
	}
	t.Cleanup(func() { openSession = old })
}

// mcpCallArgs is one usable call for each of the six, so a test about the
// token can call every tool the way a model would.
func mcpCallArgs() map[string]string {
	return map[string]string{
		"read":        `{"url":"` + fixtureDocID + `"}`,
		"comments":    `{"url":"` + fixtureDocID + `"}`,
		"suggestions": `{"url":"` + fixtureDocID + `"}`,
		"reply":       `{"url":"` + fixtureDocID + `","comment_id":"AAAA1111","body":"🤖 The 2026 register."}`,
		"annotate":    `{"url":"` + fixtureDocID + `","annotations":[{"quoted":"reviewed annually","why":"The 2026 register says quarterly."}]}`,
		"propose":     `{"url":"` + fixtureDocID + `","proposals":[{"quoted":"reviewed annually","replacement":"reviewed quarterly","why":"The 2026 register says quarterly."}]}`,
	}
}

// envelopeOf reads the one text item a tool answers with back as the envelope
// it is.
func envelopeOf(t *testing.T, texts []string) struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
} {
	t.Helper()
	var out struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if len(texts) != 1 {
		t.Fatalf("one answer is one text item: %v", texts)
	}
	if err := json.Unmarshal([]byte(texts[0]), &out); err != nil {
		t.Fatalf("the answer is not an envelope: %v", err)
	}
	return out
}

// With no token file, every one of the six says so and sends nothing. A chat
// has no terminal in it, so the answer names the one thing that can fix it,
// which is the login tool, and never a command somebody would have to type.
func TestNoTokenMakesEveryGoogleToolAnswerTheLoginHint(t *testing.T) {
	args := mcpCallArgs()
	for _, c := range mcpCommands() {
		t.Run(c.tool, func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
			var o opens
			o.stub(t)

			code := callCode(t)
			res := mcpRun(context.Background(), c, withCode(t, code, args[c.tool]), nilWriter{}, code)
			env := envelopeOf(t, res.Texts)
			if env.OK {
				t.Errorf("%s answered ok with nobody signed in", c.tool)
			}
			if !res.IsError {
				t.Errorf("%s answered isError = false, and the model has to see this one", c.tool)
			}
			if !strings.Contains(env.Error, "signed in to Google") {
				t.Errorf("%s does not say the person is not signed in: %q", c.tool, env.Error)
			}
			if !strings.Contains(env.Error, "login tool") {
				t.Errorf("%s does not say to call the login tool: %q", c.tool, env.Error)
			}
			if o.n != 0 {
				t.Errorf("%s opened %d sessions, and a tool with no token sends nothing", c.tool, o.n)
			}
		})
	}
}

// A token file that is there and cannot be read is a fault to report, never
// "signed out". Telling the person to sign in again would have them answer a
// browser prompt for a file gdoc never looked past, and the login would write
// over whatever is there.
func TestABrokenTokenFileIsNamedNotTreatedAsSignedOut(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	writeTokenFile(t, dir, "this is not JSON")
	var o opens
	o.stub(t)

	var read mcpCommand
	for _, c := range mcpCommands() {
		if c.tool == "read" {
			read = c
		}
	}
	code := callCode(t)
	res := mcpRun(context.Background(), read, withCode(t, code, `{"url":"`+fixtureDocID+`"}`), nilWriter{}, code)
	env := envelopeOf(t, res.Texts)
	if env.OK {
		t.Error("a token file nothing can read answered ok")
	}
	if !strings.Contains(env.Error, "cannot be parsed") {
		t.Errorf("the answer does not name what is wrong with the file: %q", env.Error)
	}
	if !strings.Contains(env.Error, filepath.Join(dir, "oauth-token.json")) {
		t.Errorf("the answer does not name the file: %q", env.Error)
	}
	if strings.Contains(env.Error, "login tool") {
		t.Errorf("a broken file was answered as being signed out: %q", env.Error)
	}
	if o.n != 0 {
		t.Errorf("%d sessions opened, and a token nothing can read sends nothing", o.n)
	}
}
