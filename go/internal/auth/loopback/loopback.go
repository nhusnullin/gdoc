// Package loopback is the one-shot localhost listener the login flow parks a
// browser redirect on. It serves exactly one callback and never makes a
// request, which is why the boundary test allowlists the import and still keeps
// this package out of the builder set: nothing here dials.
//
// The callback does not answer the browser itself. It hands the code to
// whoever is waiting and holds the request open until Finish writes the page,
// so the sentence the person reads is the one gdoc can stand behind: "Signed
// in" only once the token is on disk. TestFinishWritesThePageAndNothingBeforeItDoes
// and TestFinishWithAnErrorSaysTheSignInFailed are the pins. A held request
// that gets no Finish within finishWait answers on its own, because a browser
// spinning for ever says nothing at all:
// TestAFinishThatNeverComesStillAnswersTheBrowser.
package loopback

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// finishWait is how long a held browser request waits for Finish before it
// answers on its own. Half a minute is longer than a code exchange and a file
// write take, and short enough that nobody sits in front of a blank tab
// wondering.
const finishWait = 30 * time.Second

const (
	signedInPage    = "Signed in. You can close this tab."
	unconfirmedPage = "gdoc could not confirm the sign-in; look at the terminal or the chat that started it."
	alreadyPage     = "A sign-in is already being finished here. Look at the tab that started it."
)

// Server is a listener bound to 127.0.0.1 on a port the kernel picks. It holds
// at most one answer, and a second matching hit changes nothing.
type Server struct {
	ln    net.Listener
	state string
	// answers, not codes: a refusal from the authorization server comes back
	// the same way a code does, and the caller has to tell the two apart.
	answers chan result
	srv     *http.Server

	// finishWait is the constant above, per server so a test need not wait
	// half a minute for the unconfirmed page.
	finishWait time.Duration

	mu sync.Mutex
	// taken is "an answer already stands". It guards the send to answers, so
	// a second callback never displaces the first and never blocks.
	taken bool
	// page carries the sentence Finish wants written; wrote is closed by the
	// held handler once that sentence is out. Both are nil when no request is
	// being held.
	page  chan string
	wrote chan struct{}
}

type result struct {
	code string
	err  error
}

// Listen binds 127.0.0.1 on a free port and starts serving the callback. The
// state is checked in the handler, not afterwards: any local process, or any
// page in the user's browser, can reach this port, and a callback that does not
// carry this run's state must not be able to end the login.
func Listen(state string) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{ln: ln, state: state, answers: make(chan result, 1), finishWait: finishWait}
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.callback)
	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = s.srv.Serve(ln) }()
	return s, nil
}

// callback answers the browser honestly. A refusal and a callback with no code
// are answered on the spot, because nothing gdoc does next can change what to
// say. A code is different: whether the person is signed in is not known yet,
// so the request is held until Finish.
func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("state") != s.state {
		http.Error(w, "This callback does not belong to the sign-in running here.", http.StatusBadRequest)
		return // keep waiting: somebody else's callback must not end this login
	}
	if e := q.Get("error"); e != "" {
		msg := e
		if d := q.Get("error_description"); d != "" {
			msg += ": " + d
		}
		http.Error(w, "Sign-in failed: "+msg, http.StatusBadRequest)
		s.flushThenAnswer(w, result{err: fmt.Errorf("the sign-in was refused: %s", msg)})
		return
	}
	code := q.Get("code")
	if code == "" {
		http.Error(w, "Sign-in failed: the callback carried no code.", http.StatusBadRequest)
		s.flushThenAnswer(w, result{err: errors.New("login callback carried no code")})
		return
	}
	page, wrote, held := s.hold(result{code: code})
	if !held {
		fmt.Fprint(w, alreadyPage)
		return // a second hit changes nothing
	}
	// wrote closes last, after the page has been written and flushed, which is
	// what lets Finish promise the page is out before it returns.
	defer close(wrote)
	body := unconfirmedPage
	select {
	case body = <-page:
	case <-time.After(s.finishWait):
	}
	fmt.Fprint(w, body)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// hold registers this request as the one being held and hands r to whoever is
// waiting, both under the one lock. The two have to happen together: a Finish
// that lands between them would find nothing held and the browser would wait
// out finishWait for a login that already finished.
func (s *Server) hold(r result) (page chan string, wrote chan struct{}, held bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.taken {
		return nil, nil, false
	}
	s.taken = true
	page, wrote = make(chan string, 1), make(chan struct{})
	s.page, s.wrote = page, wrote
	s.answers <- r // buffered, and taken is why this never blocks
	return page, wrote, true
}

// flushThenAnswer pushes the page out to the browser, then hands the result to
// whoever is waiting. The order is the whole point: answering first ends the
// login, the caller closes the listener, and the browser gets a dropped
// connection instead of the sentence explaining what happened.
func (s *Server) flushThenAnswer(w http.ResponseWriter, r result) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.taken {
		return // a second hit changes nothing
	}
	s.taken = true
	s.answers <- r
}

// Finish writes the page the held browser request is waiting for and returns
// only once that page is out. nil is "signed in"; anything else is the reason
// the sign-in failed, in the person's own tab.
//
// It is a no-op when nothing is held, which is every path that failed before a
// code arrived, and it is safe to call twice.
func (s *Server) Finish(err error) {
	s.mu.Lock()
	page, wrote := s.page, s.wrote
	s.page, s.wrote = nil, nil
	s.mu.Unlock()
	if page == nil {
		return
	}
	page <- pageFor(err)
	<-wrote
}

func pageFor(err error) string {
	if err == nil {
		return signedInPage
	}
	return "Sign-in failed: " + err.Error()
}

// Addr is the host:port the redirect must point at.
func (s *Server) Addr() string { return s.ln.Addr().String() }

// WaitCode blocks until the browser arrives with this run's state, the
// authorization server refuses, the context ends, or the timeout runs out.
//
// A code that has already arrived wins over a context that has already ended:
// the person consented, and throwing that away would make them do it again.
// The exchange after this is the one that fails on the dead context, and the
// browser hears why, because a code is a held request.
func (s *Server) WaitCode(ctx context.Context, timeout time.Duration) (string, error) {
	select {
	case r := <-s.answers:
		return r.code, r.err
	default:
	}
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case r := <-s.answers:
		return r.code, r.err
	case <-ctx.Done():
		return "", fmt.Errorf("the wait for the login callback ended: %w", ctx.Err())
	case <-t.C:
		return "", errors.New("no login callback arrived within the timeout")
	}
}

// Close stops serving and frees the port. It is safe to call twice, because the
// login closes it from a defer and again by hand on the paths that end early.
func (s *Server) Close() { _ = s.srv.Close() }
