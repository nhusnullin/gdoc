package gapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"gdoc/internal/guard"
)

// The read goes out as the guard's own rule spells it, which is why this test
// builds a real policy rather than a permissive stub: a URL this package spells
// differently from the way the guard reads it is a read that never leaves.
func TestTheAccountReadNamesTheUserAndIsJudgedByTheGuard(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(int, *http.Request) (int, string) {
		return 200, `{"user":{"emailAddress":"someone@example.com","displayName":"Someone"}}`
	}}
	p := guard.NewPolicy()
	p.AllowAccountRead()
	s, err := Open(p, w)
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.Account(context.Background())
	if err != nil {
		t.Fatalf("the account read failed: %v", err)
	}
	if got.Email != "someone@example.com" || got.Name != "Someone" {
		t.Fatalf("read back %+v", got)
	}

	reqs := w.requests()
	if len(reqs) != 1 {
		t.Fatalf("sent %d requests, want 1: %+v", len(reqs), reqs)
	}
	if reqs[0].Method != "GET" || reqs[0].Host != "www.googleapis.com" || reqs[0].Path != "/drive/v3/about" {
		t.Errorf("the request is %s %s%s", reqs[0].Method, reqs[0].Host, reqs[0].Path)
	}
	if got := reqs[0].Auth; len(got) != 1 || !strings.HasPrefix(got[0], "Bearer ") {
		t.Errorf("Authorization = %v, want one bearer: it is a read of the token's own account", got)
	}
}

// Without the grant the same call is refused by the guard and nothing leaves.
// The account read is the one request in this package that needs a grant of
// its own, so a caller that forgot it hears so rather than getting an empty
// account.
func TestTheAccountReadWithoutTheGrantSendsNothing(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{}
	s, err := Open(guard.NewPolicy(), w)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.Account(context.Background()); err == nil {
		t.Fatal("want the guard's refusal, got none")
	}
	if n := len(w.requests()); n != 0 {
		t.Fatalf("%d requests went out: %+v", n, w.requests())
	}
}

// An answer with no user in it is named rather than handed back as an account
// with no address. A swapped account is the whole reason the read exists, so an
// empty answer must not read as one.
func TestAnAnswerWithNoUserIsNamed(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(int, *http.Request) (int, string) { return 200, `{}` }}
	p := guard.NewPolicy()
	p.AllowAccountRead()
	s, err := Open(p, w)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.Account(context.Background()); err == nil {
		t.Fatal("an answer naming no account must be an error")
	}
}
