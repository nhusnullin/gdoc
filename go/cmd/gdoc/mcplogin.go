// The login tool: one sign-in per process, the lock that keeps two gdoc
// processes from opening two listeners, and the account the answer names.
//
// Claude Desktop runs two gdoc processes, one per client (docs/v2/MEASURED.md,
// measurement 3). Every other piece of chat state is per process on purpose.
// The sign-in cannot be: the person has one browser, and a second listener
// would be a second state parameter, so the trip they are halfway through
// would stop being the one that finishes. The lock file beside the token is
// how the second process finds the first one's link instead of opening its
// own.

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"gdoc/internal/auth"
	"gdoc/internal/emit"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/mcp"
)

// The three states a login call reports. A listener that gave up and one that
// failed are both "expired", with the reason in the error: what the person
// does about either is the same, which is to ask for a fresh link.
const (
	loginWaiting  = "waiting"
	loginSignedIn = "signed in"
	loginExpired  = "expired"
)

// What the answer says beside the state. The link is the one thing in gdoc
// that only works on one computer, so every waiting answer says so: a voice
// session is held on a phone and the listener is on the Mac
// (docs/v2 milestone 14 specification, scenario 3).
const (
	mcpLoginSay = "Give the person this link and ask them to open it in a browser on the computer running Claude Desktop, " +
		"which is the only place it works. It is good for three minutes."
	mcpLoginElsewhere = "This is the link another gdoc process is already waiting on, rather than a new one."
	mcpLoginHere      = "gdoc is signed in. Tell the person which account this is, so an account they did not mean to use is noticed."
	mcpLoginGone      = "That listener is closed, so the link it gave out cannot be used."
)

// loginTrip is the half of a browser trip that outlives the call that started
// it: the wait on the loopback listener, and the port it holds. *auth.Pending
// satisfies it.
type loginTrip interface {
	Wait(ctx context.Context) error
	Close()
}

// startLogin opens the listener and builds the link. It is behind a variable
// for the reason the CLI's login is: a test stands in for the browser trip,
// and this one has to stand in for a listener that times out, one that panics
// and one that signs somebody in.
var startLogin = func() (loginTrip, string, error) {
	p, err := auth.StartLogin(guard.NewClient(guard.NewPolicy(), nil))
	if err != nil {
		return nil, "", err
	}
	return p, p.URL, nil
}

// accountOf reads the Google account the token signs in as. It is the one
// caller of guard.AllowAccountRead in this binary: the policy is built here,
// for this read, and dies with it, so no other command and no other tool gains
// that reach. Decision 4 with the DECISIONS.md entry of 2026-10-03.
var accountOf = func(ctx context.Context) (gapi.Account, error) {
	p := guard.NewPolicy()
	p.AllowAccountRead()
	s, err := gapi.Open(p, nil)
	if err != nil {
		return gapi.Account{}, err
	}
	return s.Account(ctx)
}

// mcpLoginData is what the login tool answers with. The state is a fact and so
// is the account: nothing here says whether the person should do anything,
// beyond the note that says where the link works.
type mcpLoginData struct {
	State   string `json:"state"`
	URL     string `json:"url,omitempty"`
	Account string `json:"account,omitempty"`
	Name    string `json:"name,omitempty"`
	Note    string `json:"note"`
}

// mcpLogin is the sign-in one session holds. The mutex is what makes two login
// calls one listener: the tool runs on the server's worker, one call at a time,
// but the wait runs in its own goroutine and writes the state a later call
// reads.
type mcpLogin struct {
	log io.Writer

	mu      sync.Mutex
	trip    loginTrip
	cancel  context.CancelFunc
	url     string
	started time.Time
	done    bool
	err     error
}

func newMCPLogin(log io.Writer) *mcpLogin { return &mcpLogin{log: log} }

// answer is one login call, in the order the four cases have to be asked in:
// the listener this process holds, the one another process holds, a token with
// nothing waiting at all, and then a trip nobody has started yet.
func (m *mcpLogin) answer(ctx context.Context) mcp.Result {
	if res, held := m.ownTrip(ctx); held {
		return res
	}
	if res, elsewhere := m.otherProcess(ctx); elsewhere {
		return res
	}
	_, err := auth.Load()
	switch {
	case err == nil:
		return m.signedIn(ctx)
	case !errors.Is(err, auth.ErrNoToken):
		// A token file that is there and cannot be read is named, never signed
		// in over: the login that followed would write across whatever is in
		// it. It is the rule mcpSignedIn holds for the other six tools.
		return mcpEnvelope(emit.Result{OK: false, Error: err.Error()})
	}
	return m.start(ctx)
}

// ownTrip reports the state of the listener this process opened, and whether
// there is one. A trip that has ended is dropped as it is reported, so the next
// call starts a fresh one rather than handing out a link nobody can use.
func (m *mcpLogin) ownTrip(ctx context.Context) (mcp.Result, bool) {
	m.mu.Lock()
	if m.trip == nil {
		m.mu.Unlock()
		return mcp.Result{}, false
	}
	done, err, url, started, cancel := m.done, m.err, m.url, m.started, m.cancel
	if done {
		m.trip, m.cancel, m.url = nil, nil, ""
	}
	m.mu.Unlock()

	if done {
		if cancel != nil {
			cancel()
		}
		if err == nil {
			return m.signedIn(ctx), true
		}
		// A listener that gave up is asked the same question a waiting one is,
		// because the person can finish the sign-in anywhere: a terminal
		// running gdoc auth login writes the token this one was waiting for.
		// Without this, the answer tells a model who is signed in that the
		// sign-in did not finish, and the call after it says signed in:
		// TestATimedOutListenerSeesATokenThatArrivedMeanwhile.
		if auth.SignedInSince(started) {
			return m.signedIn(ctx), true
		}
		return mcpEnvelope(emit.Result{OK: false,
			Error: "the sign-in did not finish: " + err.Error() + ". Call login again for a fresh link",
			Data:  mcpLoginData{State: loginExpired, Note: mcpLoginGone},
		}), true
	}
	// The other process may have finished the sign-in while this listener was
	// still waiting, and a token newer than this trip started is that. The
	// listener ends here, port and record together: one left waiting holds the
	// loopback port for the rest of its three minutes, and the link it handed
	// out would still exchange a code and write a token over the one that just
	// arrived.
	// TestASignInFinishedElsewhereClosesThisListenerAndItsLock.
	if auth.SignedInSince(started) {
		m.close()
		return m.signedIn(ctx), true
	}
	return waitingEnvelope(url, "", nil), true
}

// otherProcess reports the sign-in another gdoc process is waiting on, and
// whether there is one. A lock that cannot be read is logged and treated as
// nothing waiting: this process then starts its own trip, which replaces the
// file it could not read.
func (m *mcpLogin) otherProcess(ctx context.Context) (mcp.Result, bool) {
	lock, err := auth.ReadLoginLock(now())
	if err != nil {
		m.logf("gdoc mcp: %v", err)
		return mcp.Result{}, false
	}
	if lock == nil {
		return mcp.Result{}, false
	}
	if auth.SignedInSince(lock.Started) {
		return m.signedIn(ctx), true
	}
	return waitingEnvelope(lock.URL, mcpLoginElsewhere, nil), true
}

// start opens the listener, writes the lock and answers the link at once. The
// wait is a goroutine with its own recover, because a chat has a person in it
// who has to go and open a browser: an answer that waited for them is an
// answer nobody reads.
func (m *mcpLogin) start(ctx context.Context) mcp.Result {
	if err := auth.RemoveLoginLock(); err != nil {
		m.logf("gdoc mcp: the stale sign-in record could not be removed: %v", err)
	}
	trip, url, err := startLogin()
	if err != nil {
		return mcpEnvelope(emit.Result{OK: false, Error: err.Error()})
	}
	started := now()
	waitCtx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	m.trip, m.cancel, m.url, m.started, m.done, m.err = trip, cancel, url, started, false, nil
	m.mu.Unlock()

	var warnings []string
	if err := auth.WriteLoginLock(auth.LoginLock{PID: os.Getpid(), URL: url, Started: started}); err != nil {
		// A record nobody could write is no reason to refuse a sign-in the
		// person can still finish. All it costs is that the other gdoc process
		// may open a second listener, so it is said rather than swallowed.
		warnings = append(warnings, "the waiting sign-in could not be written down, so another gdoc process may open a second one: "+err.Error())
	}
	go m.wait(waitCtx, trip)
	return waitingEnvelope(url, "", warnings)
}

// wait is the browser trip's second half, with the recover the whole session
// depends on: a panic in this goroutine takes the process down and the person
// is told nothing.
func (m *mcpLogin) wait(ctx context.Context, trip loginTrip) {
	defer func() {
		if p := recover(); p != nil {
			m.logf("gdoc mcp: the sign-in listener panicked: %v\n%s", p, debug.Stack())
			m.finish(trip, fmt.Errorf("the sign-in listener failed: %v", p))
		}
	}()
	m.finish(trip, trip.Wait(ctx))
}

// finish is the one end of a sign-in, however it ended: the port freed, the
// record taken away, and the state a later login call reports.
func (m *mcpLogin) finish(trip loginTrip, err error) {
	trip.Close()
	m.mu.Lock()
	ours := m.trip == trip
	if ours {
		m.done, m.err = true, err
	}
	m.mu.Unlock()
	if !ours {
		return // close has already taken this one down, record and all
	}
	if rerr := auth.RemoveLoginLock(); rerr != nil {
		m.logf("gdoc mcp: the finished sign-in's record could not be removed: %v", rerr)
	}
}

// close ends a sign-in this session is still waiting on. stdin closing is
// Claude Desktop going away: a listener holding a port for a session nobody
// reads any more is a sign-in nobody can finish, and the record of it left
// behind is a dead link the next process hands out.
func (m *mcpLogin) close() {
	m.mu.Lock()
	trip, cancel, waiting := m.trip, m.cancel, m.trip != nil && !m.done
	m.trip, m.cancel = nil, nil
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if trip != nil {
		trip.Close()
	}
	if waiting {
		if err := auth.RemoveLoginLock(); err != nil {
			m.logf("gdoc mcp: the waiting sign-in's record could not be removed: %v", err)
		}
	}
}

// signedIn is the answer that names the account. The read failing does not
// make a signed-in person signed out: the token is the fact, and who it belongs
// to is what could not be read, so that is a warning.
func (m *mcpLogin) signedIn(ctx context.Context) mcp.Result {
	data := mcpLoginData{State: loginSignedIn, Note: mcpLoginHere}
	var warnings []string
	acc, err := accountOf(ctx)
	if err != nil {
		warnings = append(warnings, "the token is there and which Google account it belongs to could not be read: "+err.Error())
	} else {
		data.Account, data.Name = acc.Email, acc.Name
	}
	return mcpEnvelope(emit.Result{OK: true, Data: data, Warnings: warnings})
}

// waitingEnvelope is the answer while a browser is open. extra is said before
// the standing sentence, and it is where the link's owner is named when it
// belongs to the other process.
func waitingEnvelope(url, extra string, warnings []string) mcp.Result {
	note := mcpLoginSay
	if extra != "" {
		note = extra + " " + note
	}
	return mcpEnvelope(emit.Result{OK: true, Warnings: warnings,
		Data: mcpLoginData{State: loginWaiting, URL: url, Note: note}})
}

func (m *mcpLogin) logf(format string, args ...any) {
	if m.log == nil {
		return
	}
	fmt.Fprintf(m.log, format+"\n", args...)
}
