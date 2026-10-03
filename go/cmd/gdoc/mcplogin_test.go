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
	"time"

	"gdoc/internal/auth"
	"gdoc/internal/gapi"
	"gdoc/internal/mcp"
)

const testAuthURL = "https://accounts.google.com/o/oauth2/auth?client_id=not-real&state=abc"

// fakeTrip is a browser trip that opens no port. Wait does whatever the test
// hands it, so a listener that times out, one that panics and one that signs
// somebody in are all the same two lines of setup.
type fakeTrip struct {
	mu     sync.Mutex
	wait   func(ctx context.Context) error
	closed int
}

func (f *fakeTrip) Wait(ctx context.Context) error {
	if f.wait == nil {
		<-ctx.Done()
		return ctx.Err()
	}
	return f.wait(ctx)
}

func (f *fakeTrip) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed++
}

func (f *fakeTrip) closes() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

// trips stands in for the browser trip. It counts the listeners a run opened,
// which is what "one link and one port" is about, and keeps the last one so a
// test can see it closed.
type trips struct {
	mu      sync.Mutex
	started int
	url     string
	wait    func(ctx context.Context) error
	err     error
	last    *fakeTrip
}

func (tr *trips) stub(t *testing.T) {
	t.Helper()
	old := startLogin
	startLogin = func() (loginTrip, string, error) {
		tr.mu.Lock()
		defer tr.mu.Unlock()
		tr.started++
		if tr.err != nil {
			return nil, "", tr.err
		}
		tr.last = &fakeTrip{wait: tr.wait}
		return tr.last, tr.url, nil
	}
	t.Cleanup(func() { startLogin = old })
}

func (tr *trips) count() int {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return tr.started
}

func (tr *trips) trip() *fakeTrip {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return tr.last
}

// stubAccount stands in for the one Drive read that names the signed-in
// account. The read itself, URL, field mask and guard included, is held in
// internal/gapi by TestTheAccountReadNamesTheUserAndIsJudgedByTheGuard.
func stubAccount(t *testing.T, acc gapi.Account, err error) *int {
	t.Helper()
	reads := 0
	old := accountOf
	accountOf = func(context.Context) (gapi.Account, error) {
		reads++
		return acc, err
	}
	t.Cleanup(func() { accountOf = old })
	return &reads
}

// loginAnswer is the login tool's envelope, read back.
type loginAnswer struct {
	OK       bool     `json:"ok"`
	Error    string   `json:"error"`
	Warnings []string `json:"warnings"`
	Data     struct {
		State   string `json:"state"`
		URL     string `json:"url"`
		Account string `json:"account"`
		Name    string `json:"name"`
		Note    string `json:"note"`
	} `json:"data"`
}

func callLogin(t *testing.T, m *mcpLogin) loginAnswer {
	t.Helper()
	res := m.answer(context.Background())
	if len(res.Texts) != 1 {
		t.Fatalf("one answer is one text item: %v", res.Texts)
	}
	var out loginAnswer
	if err := json.Unmarshal([]byte(res.Texts[0]), &out); err != nil {
		t.Fatalf("the answer is not an envelope: %v", err)
	}
	if out.OK == res.IsError {
		t.Errorf("isError must disagree with ok: %v, %+v", res.IsError, out)
	}
	return out
}

// loginUntil calls login until it reports the state asked for. The sign-in
// finishes in a goroutine, so what a later call says is what a chat would see.
func loginUntil(t *testing.T, m *mcpLogin, state string) loginAnswer {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		got := callLogin(t, m)
		if got.Data.State == state {
			return got
		}
		if time.Now().After(deadline) {
			t.Fatalf("login never reached %q; it says %+v", state, got)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// loginSettled calls login until it reports something other than waiting, and
// answers that. It is loginUntil for a test about which state a trip settles
// into rather than about reaching one: a later call would read the token on its
// own, so only the first answer after the trip ended says anything.
func loginSettled(t *testing.T, m *mcpLogin) loginAnswer {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		got := callLogin(t, m)
		if got.Data.State != "waiting" {
			return got
		}
		if time.Now().After(deadline) {
			t.Fatalf("login never settled; it says %+v", got)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// lockOnDisk is the waiting sign-in as the file holds it, or nil.
func lockOnDisk(t *testing.T) *auth.LoginLock {
	t.Helper()
	dir := os.Getenv("GDOC_CONFIG_DIR")
	b, err := os.ReadFile(filepath.Join(dir, "login-pending.json"))
	if err != nil {
		return nil
	}
	var l auth.LoginLock
	if err := json.Unmarshal(b, &l); err != nil {
		t.Fatalf("the lock does not parse: %v", err)
	}
	return &l
}

// The link comes back from the first call, with the listener still waiting.
// A chat has a person in it who has to go and open a browser, so an answer
// that waited for them would be an answer nobody ever reads.
func TestLoginAnswersTheLinkAtOnce(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	tr := &trips{url: testAuthURL}
	tr.stub(t)
	m := newMCPLogin(io.Discard)
	t.Cleanup(m.close)

	got := callLogin(t, m)
	if !got.OK || got.Data.State != "waiting" {
		t.Fatalf("a started sign-in is waiting: %+v", got)
	}
	if got.Data.URL != testAuthURL {
		t.Errorf("the answer carries %q, want the link", got.Data.URL)
	}
	if !strings.Contains(got.Data.Note, "computer running Claude Desktop") {
		t.Errorf("the answer must say where the link works: %q", got.Data.Note)
	}
	if tr.trip().closes() != 0 {
		t.Error("the listener was closed while it was still waiting")
	}

	lock := lockOnDisk(t)
	if lock == nil {
		t.Fatal("a waiting sign-in writes itself down, so a second process finds it")
	}
	if lock.PID != os.Getpid() || lock.URL != testAuthURL {
		t.Errorf("the lock says %+v", *lock)
	}
}

// Two calls in one process are one listener. A second port would be a second
// state parameter, and the browser trip the person already started would stop
// being the one that finishes.
func TestLoginTwiceInOneProcessGivesOneLinkAndOnePort(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	tr := &trips{url: testAuthURL}
	tr.stub(t)
	m := newMCPLogin(io.Discard)
	t.Cleanup(m.close)

	first := callLogin(t, m)
	second := callLogin(t, m)
	if first.Data.URL != second.Data.URL {
		t.Errorf("two links: %q and %q", first.Data.URL, second.Data.URL)
	}
	if second.Data.State != "waiting" {
		t.Errorf("the second call says %q", second.Data.State)
	}
	if n := tr.count(); n != 1 {
		t.Errorf("%d listeners were opened, want 1", n)
	}
}

// Claude Desktop runs two gdoc processes, one per client (MEASURED 3). The
// second one hands on the link the first is waiting on rather than opening a
// listener of its own, because the person has one browser and the sign-in they
// are halfway through must be the one that finishes.
func TestASecondProcessReusesTheWaitingListener(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	other := "https://accounts.google.com/o/oauth2/auth?client_id=not-real&state=other"
	if err := auth.WriteLoginLock(auth.LoginLock{PID: os.Getpid(), URL: other, Started: time.Now()}); err != nil {
		t.Fatal(err)
	}
	tr := &trips{url: testAuthURL}
	tr.stub(t)
	m := newMCPLogin(io.Discard)
	t.Cleanup(m.close)

	got := callLogin(t, m)
	if !got.OK || got.Data.State != "waiting" {
		t.Fatalf("a sign-in waiting elsewhere is still waiting: %+v", got)
	}
	if got.Data.URL != other {
		t.Errorf("the answer carries %q, want the waiting process's link", got.Data.URL)
	}
	if !strings.Contains(got.Data.Note, "another gdoc") {
		t.Errorf("the answer must say whose link this is: %q", got.Data.Note)
	}
	if n := tr.count(); n != 0 {
		t.Errorf("%d listeners were opened, and the one waiting was enough", n)
	}
}

// A lock a dead process left, or one older than the three minutes a listener
// waits, is taken away and a fresh sign-in started. Reading either as live
// would hand out a link that opens on nothing.
func TestAStaleLockIsRemoved(t *testing.T) {
	for _, c := range []struct {
		name string
		lock auth.LoginLock
	}{
		{"the process is gone", auth.LoginLock{PID: 99999998, URL: testAuthURL, Started: time.Now()}},
		{"older than three minutes", auth.LoginLock{PID: os.Getpid(), URL: testAuthURL, Started: time.Now().Add(-4 * time.Minute)}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
			if err := auth.WriteLoginLock(c.lock); err != nil {
				t.Fatal(err)
			}
			tr := &trips{url: testAuthURL}
			tr.stub(t)
			m := newMCPLogin(io.Discard)
			t.Cleanup(m.close)

			got := callLogin(t, m)
			if got.Data.URL != testAuthURL {
				t.Errorf("the answer carries the stale link %q", got.Data.URL)
			}
			if n := tr.count(); n != 1 {
				t.Errorf("%d listeners were opened, want the one this process started", n)
			}
			lock := lockOnDisk(t)
			if lock == nil || lock.URL != testAuthURL || lock.PID != os.Getpid() {
				t.Errorf("the stale lock was not replaced: %+v", lock)
			}
		})
	}
}

// Three states, and the second call is what reports them: waiting while the
// browser is open, signed in once the token is saved, expired when the
// listener gave up or failed. The reason is in the answer either way, because
// the person reads it in the chat and nowhere else.
func TestTheStateMovesFromWaitingToSignedInOrExpired(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
		tr := &trips{url: testAuthURL, wait: func(context.Context) error {
			return errors.New("no login callback arrived within the timeout")
		}}
		tr.stub(t)
		m := newMCPLogin(io.Discard)
		t.Cleanup(m.close)

		if got := callLogin(t, m); got.Data.State != "waiting" {
			t.Fatalf("the first call is the link: %+v", got)
		}
		got := loginUntil(t, m, "expired")
		if got.OK {
			t.Error("a sign-in that did not finish is not a success")
		}
		if !strings.Contains(got.Error, "timeout") {
			t.Errorf("the answer must carry what the listener said: %q", got.Error)
		}
		if lock := lockOnDisk(t); lock != nil {
			t.Errorf("a sign-in that ended left its lock behind: %+v", *lock)
		}

		// And the next call starts a fresh one, because a link that has
		// expired is one the person cannot use.
		if next := callLogin(t, m); next.Data.State != "waiting" {
			t.Errorf("after an expiry the next call starts again: %+v", next)
		}
		if n := tr.count(); n != 2 {
			t.Errorf("%d listeners were opened, want two", n)
		}
	})

	t.Run("signed in here", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("GDOC_CONFIG_DIR", dir)
		stubAccount(t, gapi.Account{Email: "someone@example.com", Name: "Someone"}, nil)
		tr := &trips{url: testAuthURL, wait: func(context.Context) error {
			writeTokenFile(t, dir, signedInToken)
			return nil
		}}
		tr.stub(t)
		m := newMCPLogin(io.Discard)
		t.Cleanup(m.close)

		callLogin(t, m)
		got := loginUntil(t, m, "signed in")
		if !got.OK || got.Data.Account != "someone@example.com" {
			t.Errorf("the finished sign-in must name the account: %+v", got)
		}
		if lock := lockOnDisk(t); lock != nil {
			t.Errorf("a finished sign-in left its lock behind: %+v", *lock)
		}
	})

	t.Run("signed in in the other process", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("GDOC_CONFIG_DIR", dir)
		stubAccount(t, gapi.Account{Email: "someone@example.com", Name: "Someone"}, nil)
		tr := &trips{url: testAuthURL}
		tr.stub(t)
		m := newMCPLogin(io.Discard)
		t.Cleanup(m.close)

		if err := auth.WriteLoginLock(auth.LoginLock{
			PID: os.Getpid(), URL: testAuthURL, Started: time.Now().Add(-10 * time.Second)}); err != nil {
			t.Fatal(err)
		}
		writeTokenFile(t, dir, signedInToken)

		got := callLogin(t, m)
		if !got.OK || got.Data.State != "signed in" {
			t.Fatalf("a token newer than the waiting sign-in is that sign-in finishing: %+v", got)
		}
		if got.Data.Account != "someone@example.com" {
			t.Errorf("the answer must name the account: %+v", got)
		}
		if n := tr.count(); n != 0 {
			t.Errorf("%d listeners were opened for a sign-in that had already finished", n)
		}
	})
}

// A listener that gave up does not make a signed-in person signed out.
//
// The person can finish the sign-in anywhere: a terminal running gdoc auth
// login writes the same token file this listener was waiting for. So a trip
// that timed out asks the token the same question the waiting answer asks, and
// a token newer than this trip started is this sign-in finishing. Without that
// the next call tells a model the sign-in did not finish and to ask for a fresh
// link, and the call after it says signed in.
func TestATimedOutListenerSeesATokenThatArrivedMeanwhile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	stubAccount(t, gapi.Account{Email: "someone@example.com", Name: "Someone"}, nil)
	tr := &trips{url: testAuthURL, wait: func(context.Context) error {
		return errors.New("no login callback arrived within the timeout")
	}}
	tr.stub(t)
	m := newMCPLogin(io.Discard)
	t.Cleanup(m.close)

	if got := callLogin(t, m); got.Data.State != "waiting" {
		t.Fatalf("the first call is the link: %+v", got)
	}
	// Signed in somewhere else while this listener was still open.
	writeTokenFile(t, dir, signedInToken)

	// The first answer after the listener gave up is the one a chat reads, so
	// it is the one measured: a later call would find the token on its own.
	got := loginSettled(t, m)
	if !got.OK || got.Data.State != "signed in" {
		t.Fatalf("a token newer than the trip is that sign-in finishing: %+v", got)
	}
	if got.Data.Account != "someone@example.com" {
		t.Errorf("the answer must name the account: %+v", got)
	}
}

// A sign-in that finished elsewhere ends this process's listener too.
//
// The answer is the same either way, but the port and the lock are not: a
// listener left waiting holds the loopback port for three more minutes, and the
// link it handed out would still exchange a code and write a token over the one
// that just arrived.
func TestASignInFinishedElsewhereClosesThisListenerAndItsLock(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	stubAccount(t, gapi.Account{Email: "someone@example.com", Name: "Someone"}, nil)
	// No wait function: the listener waits until somebody closes it.
	tr := &trips{url: testAuthURL}
	tr.stub(t)
	m := newMCPLogin(io.Discard)
	t.Cleanup(m.close)

	if got := callLogin(t, m); got.Data.State != "waiting" {
		t.Fatalf("the first call is the link: %+v", got)
	}
	writeTokenFile(t, dir, signedInToken)

	got := callLogin(t, m)
	if !got.OK || got.Data.State != "signed in" {
		t.Fatalf("a token newer than the trip is that sign-in finishing: %+v", got)
	}
	if tr.trip().closes() == 0 {
		t.Error("the listener waiting on a finished sign-in kept its port")
	}
	if lock := lockOnDisk(t); lock != nil {
		t.Errorf("the finished sign-in left its lock behind: %+v", *lock)
	}
	if next := callLogin(t, m); !next.OK || next.Data.State != "signed in" {
		t.Errorf("the call after it says %+v", next)
	}
	if n := tr.count(); n != 1 {
		t.Errorf("%d listeners were opened, want one", n)
	}
}

// Signed in already, nothing waiting: the answer is the account, read through
// the one Drive request the guard's account grant opens. Decision 4: a swapped
// account has to be visible in the chat.
func TestTheSignedInAnswerNamesTheAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	writeTokenFile(t, dir, signedInToken)
	reads := stubAccount(t, gapi.Account{Email: "someone@example.com", Name: "Someone"}, nil)
	tr := &trips{url: testAuthURL}
	tr.stub(t)
	m := newMCPLogin(io.Discard)
	t.Cleanup(m.close)

	got := callLogin(t, m)
	if !got.OK || got.Data.State != "signed in" {
		t.Fatalf("a token and nothing waiting is signed in: %+v", got)
	}
	if got.Data.Account != "someone@example.com" || got.Data.Name != "Someone" {
		t.Errorf("the answer says %q, %q", got.Data.Account, got.Data.Name)
	}
	if *reads != 1 {
		t.Errorf("the account was read %d times, want once", *reads)
	}
	if n := tr.count(); n != 0 {
		t.Errorf("%d listeners were opened for somebody already signed in", n)
	}
}

// The account read failing does not make a signed-in person signed out. The
// token is the fact; who it belongs to is what could not be read, and that is
// a warning.
func TestASignedInAnswerWhoseAccountReadFailedStillSaysSignedIn(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	writeTokenFile(t, dir, signedInToken)
	stubAccount(t, gapi.Account{}, errors.New("the account read failed"))
	tr := &trips{url: testAuthURL}
	tr.stub(t)
	m := newMCPLogin(io.Discard)
	t.Cleanup(m.close)

	got := callLogin(t, m)
	if !got.OK || got.Data.State != "signed in" {
		t.Fatalf("a read that failed is not a sign-out: %+v", got)
	}
	if got.Data.Account != "" {
		t.Errorf("an account nobody could read must not be named: %q", got.Data.Account)
	}
	if len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "account read failed") {
		t.Errorf("the warnings must say why: %v", got.Warnings)
	}
}

// The listener runs in a goroutine, so a panic in it takes the process down
// and the session dies mid sentence unless it has its own recover. The panic
// becomes the state a later call reports.
func TestAPanicInTheListenerDoesNotEndTheServer(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	var log bytes.Buffer
	tr := &trips{url: testAuthURL, wait: func(context.Context) error {
		panic("something impossible in the listener")
	}}
	tr.stub(t)
	m := newMCPLogin(&log)
	t.Cleanup(m.close)

	if got := callLogin(t, m); got.Data.State != "waiting" {
		t.Fatalf("the first call is the link: %+v", got)
	}
	got := loginUntil(t, m, "expired")
	if !strings.Contains(got.Error, "something impossible") {
		t.Errorf("the answer must carry what happened: %q", got.Error)
	}
	if !strings.Contains(log.String(), "something impossible") {
		t.Errorf("the panic belongs in the log too: %q", log.String())
	}
	if lock := lockOnDisk(t); lock != nil {
		t.Errorf("the lock outlived the listener: %+v", *lock)
	}
}

// Claude Desktop closing ends the session, and the port and the lock go with
// it. A listener left holding a port after the process that answers for it has
// stopped reading is a sign-in nobody can finish, and a lock left behind makes
// the next process hand out its dead link.
func TestStdinClosingClosesAWaitingListenerAndItsLock(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	tr := &trips{url: testAuthURL}
	tr.stub(t)

	in := strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"login","arguments":{}}}` + "\n")
	var out bytes.Buffer
	if code := serveMCP(context.Background(), in, &out, io.Discard, nil); code != 0 {
		t.Fatalf("the session ended %d: %s", code, out.String())
	}

	if !strings.Contains(out.String(), testAuthURL) {
		t.Fatalf("the session never answered the login call: %s", out.String())
	}
	trip := tr.trip()
	if trip == nil {
		t.Fatal("no listener was opened")
	}
	if trip.closes() == 0 {
		t.Error("stdin closed and the listener kept its port")
	}
	if lock := lockOnDisk(t); lock != nil {
		t.Errorf("stdin closed and the lock stayed: %+v", *lock)
	}
}

// The tool itself: the name, the words and the empty schema a card draws a
// button for.
func TestTheLoginToolIsListedAsItself(t *testing.T) {
	m := newMCPLogin(io.Discard)
	var tool mcp.Tool
	for _, got := range mcpTools(io.Discard, m, callChat(t)) {
		if got.Name == "login" {
			tool = got
		}
	}
	if tool.Name != "login" {
		t.Fatal("the session offers no login tool")
	}
	if !tool.ReadOnly {
		t.Error("signing in changes no document")
	}
	if string(tool.Schema) != noArguments {
		t.Errorf("login takes no arguments: %s", tool.Schema)
	}
}
