// Package loopback is the one-shot localhost listener the login flow parks a
// browser redirect on. It serves exactly one callback and never makes a
// request, which is why the boundary test allowlists the import and still keeps
// this package out of the builder set: nothing here dials.
package loopback

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Server is a listener bound to 127.0.0.1 on a port the kernel picks. It holds
// at most one answer, and a second matching hit changes nothing.
type Server struct {
	ln    net.Listener
	state string
	codes chan result
	srv   *http.Server
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
	s := &Server{ln: ln, state: state, codes: make(chan result, 1)}
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.callback)
	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = s.srv.Serve(ln) }()
	return s, nil
}

// callback answers the browser honestly. It says "signed in" only when a code
// this run asked for actually arrived.
//
// The page is written and flushed before the answer is delivered. Delivering
// first ends the login, the caller closes the listener, and the browser gets a
// dropped connection instead of the sentence explaining what happened.
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
		s.deliver(w, result{err: fmt.Errorf("the sign-in was refused: %s", msg)})
		return
	}
	code := q.Get("code")
	if code == "" {
		http.Error(w, "Sign-in failed: the callback carried no code.", http.StatusBadRequest)
		s.deliver(w, result{err: errors.New("login callback carried no code")})
		return
	}
	fmt.Fprint(w, "Signed in. You can close this tab.")
	s.deliver(w, result{code: code})
}

func (s *Server) deliver(w http.ResponseWriter, r result) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	select {
	case s.codes <- r:
	default: // a second hit changes nothing
	}
}

// Addr is the host:port the redirect must point at.
func (s *Server) Addr() string { return s.ln.Addr().String() }

// WaitCode blocks until the browser arrives with this run's state, the
// authorization server refuses, or the timeout runs out.
func (s *Server) WaitCode(timeout time.Duration) (string, error) {
	select {
	case r := <-s.codes:
		if r.err != nil {
			return "", r.err
		}
		return r.code, nil
	case <-time.After(timeout):
		return "", errors.New("no login callback arrived within the timeout")
	}
}

// Close stops serving and frees the port.
func (s *Server) Close() { _ = s.srv.Close() }
