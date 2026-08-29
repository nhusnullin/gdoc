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
// at most one callback, and a second hit changes nothing.
type Server struct {
	ln    net.Listener
	codes chan result
	srv   *http.Server
}

type result struct {
	code, state string
}

// Listen binds 127.0.0.1 on a free port and starts serving the callback.
func Listen() (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{ln: ln, codes: make(chan result, 1)}
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		select {
		case s.codes <- result{code: q.Get("code"), state: q.Get("state")}:
		default: // a second hit changes nothing
		}
		fmt.Fprint(w, "Signed in. You can close this tab.")
	})
	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = s.srv.Serve(ln) }()
	return s, nil
}

// Addr is the host:port the redirect must point at.
func (s *Server) Addr() string { return s.ln.Addr().String() }

// WaitCode blocks until the browser arrives, the state fails to match, or the
// timeout runs out. A callback that does not carry this run's state is somebody
// else's, so its code is refused rather than used.
func (s *Server) WaitCode(state string, timeout time.Duration) (string, error) {
	select {
	case r := <-s.codes:
		if r.state != state {
			return "", errors.New("login callback carried the wrong state; refusing the code")
		}
		if r.code == "" {
			return "", errors.New("login callback carried no code")
		}
		return r.code, nil
	case <-time.After(timeout):
		return "", errors.New("no login callback arrived within the timeout")
	}
}

// Close stops serving and frees the port.
func (s *Server) Close() { _ = s.srv.Close() }
