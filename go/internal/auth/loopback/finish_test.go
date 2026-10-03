package loopback

// The held request: the browser waits for Finish, and it is answered whatever
// happens to the login that is holding it.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// page plays the browser from its own goroutine, because the request is held
// open until Finish and the test has to go on meanwhile. It carries the error
// rather than failing from a goroutine, which testing forbids.
type page struct {
	body   string
	status int
	err    error
}

func hitAsync(addr string, q url.Values) <-chan page {
	out := make(chan page, 1)
	u := url.URL{Scheme: "http", Host: addr, Path: "/callback", RawQuery: q.Encode()}
	go func() {
		resp, err := http.Get(u.String())
		if err != nil {
			out <- page{err: err}
			return
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		out <- page{body: string(b), status: resp.StatusCode, err: err}
	}()
	return out
}

// quiet fails the test if the browser has already been answered. The callback
// holds its request open, and that is what lets the login say "signed in" only
// once the token is on disk.
func quiet(t *testing.T, got <-chan page) {
	t.Helper()
	select {
	case p := <-got:
		t.Fatalf("the browser was answered before Finish: %+v", p)
	case <-time.After(100 * time.Millisecond):
	}
}

func take(t *testing.T, got <-chan page) page {
	t.Helper()
	select {
	case p := <-got:
		if p.err != nil {
			t.Fatalf("the browser got no page: %v", p.err)
		}
		return p
	case <-time.After(5 * time.Second):
		t.Fatal("the browser was never answered")
		return page{}
	}
}

// Finish writes the page, and nothing before it does. The order is the whole
// point: a page saying "Signed in" that goes out before the token is saved
// tells the person something gdoc does not know yet.
func TestFinishWritesThePageAndNothingBeforeItDoes(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	got := hitAsync(s.Addr(), callback("CODE1", "STATE1"))
	code, err := s.WaitCode(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if code != "CODE1" {
		t.Fatalf("code: %q", code)
	}
	quiet(t, got)

	s.Finish(nil)
	p := take(t, got)
	if p.status != http.StatusOK || !strings.Contains(p.body, "Signed in") {
		t.Fatalf("Finish(nil) says the person is signed in: %d %q", p.status, p.body)
	}
	// Close straight after Finish must not drop the page: Finish returned only
	// once the handler had written it.
	s.Close()
	if !strings.Contains(p.body, "close this tab") {
		t.Fatalf("the page tells the person what to do next: %q", p.body)
	}
}

// A login that broke says so on the page, with the reason. The person is
// looking at the browser, not at the terminal.
func TestFinishWithAnErrorSaysTheSignInFailed(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	got := hitAsync(s.Addr(), callback("CODE1", "STATE1"))
	if _, err := s.WaitCode(context.Background(), time.Second); err != nil {
		t.Fatal(err)
	}
	s.Finish(errors.New("the token could not be written"))

	p := take(t, got)
	if !strings.Contains(p.body, "Sign-in failed") {
		t.Fatalf("the page must say the sign-in failed: %q", p.body)
	}
	if !strings.Contains(p.body, "the token could not be written") {
		t.Fatalf("the page must carry the reason: %q", p.body)
	}
	if strings.Contains(p.body, "Signed in") {
		t.Fatalf("a failed sign-in must not read as a success: %q", p.body)
	}
}

// A Finish that never comes must not leave the browser spinning for ever. The
// held request answers on its own and says gdoc does not know what happened,
// which is the honest answer.
func TestAFinishThatNeverComesStillAnswersTheBrowser(t *testing.T) {
	if finishWait != 30*time.Second {
		t.Fatalf("a held request waits half a minute for Finish, got %v", finishWait)
	}
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.finishWait = 50 * time.Millisecond

	got := hitAsync(s.Addr(), callback("CODE1", "STATE1"))
	if _, err := s.WaitCode(context.Background(), time.Second); err != nil {
		t.Fatal(err)
	}

	p := take(t, got)
	if !strings.Contains(p.body, "could not confirm the sign-in") {
		t.Fatalf("the page must say gdoc does not know: %q", p.body)
	}
	if strings.Contains(p.body, "Signed in") {
		t.Fatalf("an unconfirmed sign-in must not read as a success: %q", p.body)
	}
	// Finish arriving late changes nothing and does not block.
	s.Finish(nil)
}

// A second callback while one is held is answered at once and changes nothing:
// the first code is the one the login uses.
func TestASecondCallbackWhileOneIsHeldChangesNothing(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	first := hitAsync(s.Addr(), callback("FIRST", "STATE1"))
	code, err := s.WaitCode(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if code != "FIRST" {
		t.Fatalf("code: %q", code)
	}

	second := take(t, hitAsync(s.Addr(), callback("SECOND", "STATE1")))
	if !strings.Contains(second.body, "already being finished") {
		t.Fatalf("the second browser must be told a sign-in is already being finished: %q", second.body)
	}
	if strings.Contains(second.body, "Signed in") {
		t.Fatalf("the second callback must not read as a success: %q", second.body)
	}

	s.Finish(nil)
	p := take(t, first)
	if !strings.Contains(p.body, "Signed in") {
		t.Fatalf("the held page is still the one that gets the answer: %q", p.body)
	}
	// Nothing else is waiting: the second callback never reached the waiter.
	if _, err := s.WaitCode(context.Background(), 20*time.Millisecond); err == nil {
		t.Fatal("the second code must never reach the waiter")
	}
}

// Close is called from a defer and again by hand on the paths that end early,
// so it has to be safe twice.
func TestCloseIsSafeTwice(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	addr := s.Addr()
	s.Close()
	s.Close()

	if resp, err := http.Get("http://" + addr + "/callback"); err == nil {
		resp.Body.Close()
		t.Fatal("the listener still answers after Close")
	}
}

// Finish with nothing held is a no-op: the login failed before a code arrived,
// and there is no browser request to answer.
func TestFinishWithNothingHeldDoesNothing(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	done := make(chan struct{})
	go func() { s.Finish(errors.New("no callback arrived")); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Finish blocked with no request held")
	}
}

// A code already in hand wins over a context that has already ended. The
// person consented, and throwing that away would make them do it again. The
// exchange is what fails on the dead context, and because the request is held
// the browser hears why.
func TestACodeAlreadyInHandWinsOverADeadContext(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, _, held := s.hold(result{code: "CODE1"}); !held {
		t.Fatal("nothing is held yet")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	code, err := s.WaitCode(ctx, time.Minute)
	if err != nil {
		t.Fatalf("the code that arrived must still be handed over: %v", err)
	}
	if code != "CODE1" {
		t.Fatalf("code: %q", code)
	}
}

// A context that ends stops the wait. The CLI hands a background context; a
// second front door hands one that can be cancelled, and a login nobody is
// waiting for any more must not hold the wait for three minutes.
func TestACancelledContextEndsTheWait(t *testing.T) {
	s, err := Listen("STATE1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.WaitCode(ctx, time.Minute); err == nil {
		t.Fatal("a cancelled context must end the wait")
	}
}
