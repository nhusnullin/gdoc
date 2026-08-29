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
func hit(t *testing.T, addr, code, state string) string {
	t.Helper()
	u := url.URL{
		Scheme:   "http",
		Host:     addr,
		Path:     "/callback",
		RawQuery: url.Values{"code": {code}, "state": {state}}.Encode(),
	}
	resp, err := http.Get(u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestWaitCodeReturnsTheCodeAndAnswersTheBrowser(t *testing.T) {
	s, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if !strings.HasPrefix(s.Addr(), "127.0.0.1:") {
		t.Fatalf("the listener must be loopback only: %q", s.Addr())
	}
	page := hit(t, s.Addr(), "CODE1", "STATE1")
	if !strings.Contains(page, "close this tab") {
		t.Fatalf("the browser gets a plain sentence back: %q", page)
	}

	code, err := s.WaitCode("STATE1", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if code != "CODE1" {
		t.Fatalf("code: %q", code)
	}
}

func TestWaitCodeRefusesAWrongState(t *testing.T) {
	s, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	hit(t, s.Addr(), "CODE1", "SOMEBODY-ELSE")

	if _, err := s.WaitCode("STATE1", time.Second); err == nil {
		t.Fatal("a callback with another state must be refused")
	} else if !strings.Contains(err.Error(), "state") {
		t.Fatalf("the error must say what was wrong: %v", err)
	}
}

func TestWaitCodeRefusesACallbackWithNoCode(t *testing.T) {
	s, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	hit(t, s.Addr(), "", "STATE1")

	if _, err := s.WaitCode("STATE1", time.Second); err == nil {
		t.Fatal("a callback with no code must be refused")
	}
}

func TestWaitCodeTimesOut(t *testing.T) {
	s, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	start := time.Now()
	if _, err := s.WaitCode("STATE1", 20*time.Millisecond); err == nil {
		t.Fatal("waiting for a callback that never comes must end in an error")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("the timeout was not honoured")
	}
}

// Close must stop serving, so a login that failed leaves no port open.
func TestCloseStopsTheListener(t *testing.T) {
	s, err := Listen()
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
