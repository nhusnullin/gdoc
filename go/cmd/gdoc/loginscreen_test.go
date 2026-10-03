// The login screen: the link in a box, and the one line under it that waits.
//
// Every run here stands in for the browser trip through startLogin, which is
// the variable the MCP login tool's tests already stub, so no listener is
// opened and nothing reaches the network. The waiting line is recorded with
// the spinner turned down to one frame, so a screen is the same bytes on
// every machine.

package main

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"gdoc/internal/tty"
)

// loginScreenURL is the link every run below hands out. A literal, because a
// recorded screen is as wide as the link in it.
const loginScreenURL = "https://accounts.google.com/o/oauth2/auth?client_id=not-real.apps.googleusercontent.com&state=hQ2p"

// cursorUp is the one escape code a waiting line may not write: it would move
// off the line and over the link above it, and a Ctrl-C in the middle of the
// wait would leave the screen half drawn.
var cursorUp = regexp.MustCompile(`\x1b\[[0-9]*A`)

// signingIn stands in for the browser trip: the link is handed out at once and
// the wait ends with whatever err says.
func signingIn(t *testing.T, err error) *trips {
	t.Helper()
	tr := &trips{url: loginScreenURL, wait: func(context.Context) error { return err }}
	tr.stub(t)
	return tr
}

// oneFrame turns the spinner down so a recorded login is the first frame and
// nothing else, however long the machine takes between the two writes.
func oneFrame(t *testing.T) {
	t.Helper()
	old := spinEvery
	spinEvery = time.Hour
	t.Cleanup(func() { spinEvery = old })
}

// loginRun is `gdoc auth login` with both streams a terminal, at eighty
// columns with the colour off, and a trip that signs somebody in.
func loginRun(t *testing.T, width int, err error) (stderr string, code int) {
	t.Helper()
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	t.Setenv("COLUMNS", strconv.Itoa(width))
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "xterm")
	oneFrame(t)
	signingIn(t, err)

	var out, errOut bytes.Buffer
	terminals(t, &out, &errOut)
	code = run(context.Background(), []string{"auth", "login"}, &out, &errOut)
	return errOut.String(), code
}

// A pipe gets the two lines internal/auth has always printed and nothing
// else: a skill and a log read them, and the only record of a sign-in in
// somebody's terminal history is those bytes.
func TestThePipeLoginLineIsTodays(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signingIn(t, nil)

	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"auth", "login"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("a login that worked exits 0, and this run exited %d: %s", code, out.String())
	}
	want := "Open this link in your browser to sign in:\n" + loginScreenURL + "\n"
	if errOut.String() != want {
		t.Errorf("a pipe read\n%q\nand the two lines it has always read are\n%q", errOut.String(), want)
	}
	if got := decodeOne(t, &out); got["ok"] != true {
		t.Fatalf("envelope: %v", got)
	}
}

// A window with no room for a box gets those same two lines: the Plain band of
// the width rule is plain text, not a box squeezed into forty-four columns.
func TestANarrowWindowGetsTheLoginLineAndNoBox(t *testing.T) {
	stderr, code := loginRun(t, 44, nil)
	if code != 0 {
		t.Fatalf("a login that worked exits 0, and this run exited %d", code)
	}
	want := "Open this link in your browser to sign in:\n" + loginScreenURL + "\n"
	if stderr != want {
		t.Errorf("a narrow window read\n%q\nand the plain lines are\n%q", stderr, want)
	}
}

// The waiting line rewrites itself with a carriage return and nothing else.
// No cursor move and no auto-wrap change: `gdoc auth login` traps no signal,
// so a Ctrl-C while the browser is open has to leave the terminal as it found
// it, and a line that only ever rewrote itself does.
func TestTheLoginSpinnerMovesNoCursorButCarriageReturn(t *testing.T) {
	stderr, code := loginRun(t, 80, nil)
	if code != 0 {
		t.Fatalf("a login that worked exits 0, and this run exited %d", code)
	}
	if !strings.Contains(stderr, "\r") {
		t.Errorf("the waiting line is redrawn with a carriage return: %q", stderr)
	}
	if at := cursorUp.FindString(stderr); at != "" {
		t.Errorf("a login moves the cursor over nothing, and this run wrote %q: %q", at, stderr)
	}
	for what, code := range map[string]string{"wrap-off": tty.WrapOff, "wrap-on": tty.WrapOn} {
		if strings.Contains(stderr, code) {
			t.Errorf("a login writes no %s: %q", what, stderr)
		}
	}
	if want := "\r⠋ waiting for the browser to come back\x1b[K"; !strings.Contains(stderr, want) {
		t.Errorf("the waiting line is %q, and the screen is %q", want, stderr)
	}
	if want := "\r✓ signed in\x1b[K\n"; !strings.HasSuffix(stderr, want) {
		t.Errorf("the line a person is left reading is %q, and the screen ends %q", want, stderr)
	}
	if !strings.Contains(stderr, loginScreenURL+"\n") {
		t.Errorf("the link is one line of its own, so a copy gets all of it: %q", stderr)
	}
}

// A wait that failed ends the same line with a cross and the reason, and the
// object on stdout says why, as it always did.
func TestAFailedWaitEndsTheLineWithACrossAndTheObjectSaysWhy(t *testing.T) {
	const reason = "no login callback arrived within the timeout"
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	t.Setenv("COLUMNS", "80")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "xterm")
	oneFrame(t)
	signingIn(t, errors.New(reason))

	var out, errOut bytes.Buffer
	terminals(t, &out, &errOut)
	code := run(context.Background(), []string{"auth", "login"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("a login that failed exits 1, and this run exited %d", code)
	}
	if want := "\r✗ " + reason + "\x1b[K\n"; !strings.HasSuffix(errOut.String(), want) {
		t.Errorf("the line a person is left reading is %q, and the screen ends %q", want, errOut.String())
	}
	got := decodeOne(t, &out)
	if got["ok"] != false {
		t.Fatalf("a sign-in that did not finish is not a success: %v", got)
	}
	if msg, _ := got["error"].(string); msg != reason {
		t.Errorf("the object says why, and it says %q", msg)
	}
}

// The listener is closed however the wait ended: a port held open after the
// command printed its object is a sign-in nobody can finish.
func TestTheLoginClosesItsListenerEitherWay(t *testing.T) {
	for name, err := range map[string]error{
		"signed in": nil,
		"timed out": errors.New("no login callback arrived within the timeout"),
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
			oneFrame(t)
			tr := signingIn(t, err)

			var out, errOut bytes.Buffer
			run(context.Background(), []string{"auth", "login"}, &out, &errOut)
			if tr.count() != 1 {
				t.Fatalf("one login opens one listener, and this run opened %d", tr.count())
			}
			if got := tr.trip().closes(); got != 1 {
				t.Errorf("the listener is closed once, and this one was closed %d times", got)
			}
		})
	}
}

// The screen a person is left reading, recorded byte for byte: the link in its
// box, the waiting line, and the line it ended as.
func TestTheLoginScreensAreTheirRecordedBytes(t *testing.T) {
	for name, err := range map[string]error{
		"login-80.golden":        nil,
		"login-failed-80.golden": errors.New("no login callback arrived within the timeout"),
	} {
		t.Run(name, func(t *testing.T) {
			stderr, _ := loginRun(t, 80, err)
			recorded(t, name, stderr)
		})
	}
}
