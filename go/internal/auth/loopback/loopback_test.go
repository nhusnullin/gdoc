package loopback

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// hit plays the browser: it follows the redirect the authorization server would
// send back to the loopback address.
func hit(t *testing.T, addr string, q url.Values) (string, int) {
	t.Helper()
	u := url.URL{Scheme: "http", Host: addr, Path: "/callback", RawQuery: q.Encode()}
	resp, err := http.Get(u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body), resp.StatusCode
}

func callback(code, state string) url.Values {
	return url.Values{"code": {code}, "state": {state}}
}

func TestWaitCodeReturnsTheCodeAndAnswersTheBrowser(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if !strings.HasPrefix(s.Addr(), "127.0.0.1:") {
		t.Fatalf("the listener must be loopback only: %q", s.Addr())
	}
	page, status := hit(t, s.Addr(), callback("CODE1", "STATE1"))
	if status != http.StatusOK || !strings.Contains(page, "close this tab") {
		t.Fatalf("the browser gets a plain sentence back: %d %q", status, page)
	}

	code, err := s.WaitCode(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if code != "CODE1" {
		t.Fatalf("code: %q", code)
	}
}

// A callback with the wrong state is somebody else's. It must not end this
// login, and the browser must not be told it signed in. Anything on the machine
// can reach this port, so dropping the stray hit and going on waiting is the
// difference between a working login and one anybody can kill.
func TestAStrayCallbackNeitherEndsNorHijacksTheLogin(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	page, status := hit(t, s.Addr(), callback("SOMEBODY-ELSES-CODE", "WRONG-STATE"))
	if status == http.StatusOK {
		t.Fatalf("a stray callback must not be answered with a success page: %q", page)
	}
	if strings.Contains(page, "Signed in") {
		t.Fatalf("the page told the human they signed in: %q", page)
	}

	// The real browser arrives after the stray hit, and the login still works.
	hit(t, s.Addr(), callback("CODE1", "STATE1"))
	code, err := s.WaitCode(time.Second)
	if err != nil {
		t.Fatalf("the real callback must still land: %v", err)
	}
	if code != "CODE1" {
		t.Fatalf("code: %q", code)
	}
}

// The authorization server says why it refused. Reporting "no code" instead
// sends the reader looking for a bug that is not there.
func TestADeniedSignInIsReportedAsSuch(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	page, status := hit(t, s.Addr(), url.Values{
		"state":             {"STATE1"},
		"error":             {"access_denied"},
		"error_description": {"the user said no"},
	})
	if status == http.StatusOK || strings.Contains(page, "Signed in") {
		t.Fatalf("a refusal must not read as a success: %d %q", status, page)
	}
	if _, err := s.WaitCode(time.Second); err == nil {
		t.Fatal("a refused sign-in must fail the login")
	} else if !strings.Contains(err.Error(), "access_denied") {
		t.Fatalf("the error must name what the server said: %v", err)
	}
}

func TestWaitCodeRefusesACallbackWithNoCode(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	page, status := hit(t, s.Addr(), url.Values{"state": {"STATE1"}})
	if status == http.StatusOK || strings.Contains(page, "Signed in") {
		t.Fatalf("a callback with no code must not read as a success: %d %q", status, page)
	}

	if _, err := s.WaitCode(time.Second); err == nil {
		t.Fatal("a callback with no code must be refused")
	}
}

// A second matching hit changes nothing: the first answer stands.
func TestASecondCallbackChangesNothing(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	hit(t, s.Addr(), callback("FIRST", "STATE1"))
	hit(t, s.Addr(), callback("SECOND", "STATE1"))

	code, err := s.WaitCode(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if code != "FIRST" {
		t.Fatalf("the first callback is the one that counts: %q", code)
	}
}

// Nothing but /callback is served.
func TestAnotherPathIsNotTheCallback(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	resp, err := http.Get("http://" + s.Addr() + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("only /callback is served, got %d", resp.StatusCode)
	}
	if _, err := s.WaitCode(20 * time.Millisecond); err == nil {
		t.Fatal("a hit on another path must not end the wait")
	}
}

func TestWaitCodeTimesOut(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	start := time.Now()
	if _, err := s.WaitCode(20 * time.Millisecond); err == nil {
		t.Fatal("waiting for a callback that never comes must end in an error")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("the timeout was not honoured")
	}
}

// Close must stop serving, so a login that failed leaves no port open.
func TestCloseStopsTheListener(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	addr := s.Addr()
	s.Close()

	if resp, err := http.Get("http://" + addr + "/callback"); err == nil {
		resp.Body.Close()
		t.Fatal("the listener still answers after Close")
	}
}
