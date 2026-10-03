package gapi

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptrace"
	"testing"
	"time"

	"gdoc/internal/guard"
)

// This file is the bound on the third case the package comment names: a request
// that was written and whose answer never came. The mark is what tells it from a
// request that never left the machine, and the difference is the whole of what a
// writer package can say about the document afterwards.

// wasUnknown asks the way the writer packages ask, by behaviour rather than by
// naming this package's own type.
func wasUnknown(err error) bool {
	var u interface{ Unknown() bool }
	return errors.As(err, &u) && u.Unknown()
}

// openOver builds a session over any transport with the one document allowed.
// open takes the canned *wire, and these tests need a transport that fails in
// one named way each.
func openOver(t *testing.T, rt http.RoundTripper) *Session {
	t.Helper()
	p := guard.NewPolicy()
	p.AllowFile(docID, guard.LevelSuggest)
	s, err := Open(p, rt)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TestAFiveHundredAfterTheWriteIsMarkedUnknown is Docs answering that something
// went wrong on its side. The request was written, so the batch may have been
// applied before the failure, and a caller told the write never happened is a
// caller that writes it again on top of the first.
func TestAFiveHundredAfterTheWriteIsMarkedUnknown(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	w := &wire{answer: func(n int, r *http.Request) (int, string) {
		return 500, `{"error":{"message":"Internal error encountered."}}`
	}}
	s := openOver(t, w)

	err := s.PostJSON(context.Background(), batchURL(), suggestBatch(), &struct{}{})
	if err == nil {
		t.Fatal("a 500 came back as success")
	}
	if !wasUnknown(err) {
		t.Errorf("error %q is not marked unknown, and the request was written before Docs failed", err)
	}
	// Unknown is not Sent. Sent says Docs took the batch, and a 500 says the
	// opposite as far as anything can be read from it.
	if wasSent(err) {
		t.Errorf("a 500 is marked as sent, and nothing in it says the batch was applied: %q", err)
	}
}

// TestADropAfterTheWriteIsMarkedUnknown is the connection going away once the
// bytes are out. The fake fires the hook http.Transport fires at that moment,
// which is the only fact gdoc has about which side of the write the failure fell
// on.
func TestADropAfterTheWriteIsMarkedUnknown(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	dropped := rtFunc(func(r *http.Request) (*http.Response, error) {
		if tr := httptrace.ContextClientTrace(r.Context()); tr != nil && tr.WroteRequest != nil {
			tr.WroteRequest(httptrace.WroteRequestInfo{})
		}
		return nil, errors.New("read: connection reset by peer")
	})
	s := openOver(t, dropped)

	err := s.PostJSON(context.Background(), batchURL(), suggestBatch(), &struct{}{})
	if err == nil {
		t.Fatal("a dropped connection came back as success")
	}
	if !wasUnknown(err) {
		t.Errorf("error %q is not marked unknown, and the request was written before the drop", err)
	}
}

// TestNothingBeforeTheWriteIsMarkedUnknown is the half that matters most. A
// refusal the guard made inside the process, a connection that was never made
// and a handshake that never finished all mean the document was not touched, and
// a 4xx is Docs rejecting the batch whole. None of them may read as a change
// that may be in there.
//
// The dial and the handshake run through a real http.Transport, so what is being
// pinned is that the transport itself does not report a request as written on
// either path. A fake standing in for it would be pinning the fake.
func TestNothingBeforeTheWriteIsMarkedUnknown(t *testing.T) {
	refused := func(t *testing.T) (*Session, map[string]any) {
		t.Helper()
		body := suggestBatch()
		// Without writeControl the batch is a direct edit, and the guard refuses
		// it before anything leaves the machine.
		delete(body, "writeControl")
		return openOver(t, &wire{}), body
	}
	dialFailed := func(t *testing.T) (*Session, map[string]any) {
		t.Helper()
		rt := &http.Transport{DialContext: func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("dial tcp 142.250.0.0:443: connect: connection refused")
		}}
		return openOver(t, rt), suggestBatch()
	}
	handshakeFailed := func(t *testing.T) (*Session, map[string]any) {
		t.Helper()
		rt := &http.Transport{DialContext: func(context.Context, string, string) (net.Conn, error) {
			// One end of a pipe whose other end is already closed: the client
			// hello goes nowhere and the handshake fails before any request.
			ours, theirs := net.Pipe()
			theirs.Close()
			return ours, nil
		}}
		return openOver(t, rt), suggestBatch()
	}
	refusedByDocs := func(t *testing.T) (*Session, map[string]any) {
		t.Helper()
		w := &wire{answer: func(n int, r *http.Request) (int, string) {
			return 400, `{"error":{"message":"Invalid requests[0].insertText"}}`
		}}
		return openOver(t, w), suggestBatch()
	}

	for name, build := range map[string]func(*testing.T) (*Session, map[string]any){
		"the guard refused it":         refused,
		"the connection was refused":   dialFailed,
		"the handshake never finished": handshakeFailed,
		"Docs refused the batch":       refusedByDocs,
	} {
		t.Run(name, func(t *testing.T) {
			tokenFile(t, time.Now().Add(time.Hour))
			s, body := build(t)

			err := s.PostJSON(context.Background(), batchURL(), body, &struct{}{})
			if err == nil {
				t.Fatal("a failure came back as success")
			}
			if wasUnknown(err) {
				t.Errorf("error %q is marked unknown, and the request was never written", err)
			}
		})
	}
}

// failingBody is a response body that answers one read with an error, which is
// the connection going away while the server's own error body streams.
type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("read: connection reset by peer") }
func (failingBody) Close() error             { return nil }

// TestAFiveHundredWhoseBodyFailsIsStillUnknown is the 500 above read one step
// later: the status arrived, so the request was written, and the body carrying
// Google's own message did not finish. The status is the fact that matters, and a
// read that failed after it is no reason to report the batch as never sent.
func TestAFiveHundredWhoseBodyFailsIsStillUnknown(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	cut := rtFunc(func(r *http.Request) (*http.Response, error) {
		if tr := httptrace.ContextClientTrace(r.Context()); tr != nil && tr.WroteRequest != nil {
			tr.WroteRequest(httptrace.WroteRequestInfo{})
		}
		return &http.Response{StatusCode: 500, Request: r, Header: http.Header{}, Body: failingBody{}}, nil
	})
	s := openOver(t, cut)

	err := s.PostJSON(context.Background(), batchURL(), suggestBatch(), &struct{}{})
	if err == nil {
		t.Fatal("a 500 with an unreadable body came back as success")
	}
	if !wasUnknown(err) {
		t.Errorf("error %q is not marked unknown, and the request was written before Docs failed", err)
	}
	if wasSent(err) {
		t.Errorf("error %q is marked sent, and nothing in a 500 says the batch was applied", err)
	}
}

// TestAFourHundredWhoseBodyFailsIsNotUnknown is the other half. Docs rejecting
// the batch whole is a change that did not happen, and a body that failed to
// read afterwards does not turn it into one that may be in the document.
func TestAFourHundredWhoseBodyFailsIsNotUnknown(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	cut := rtFunc(func(r *http.Request) (*http.Response, error) {
		if tr := httptrace.ContextClientTrace(r.Context()); tr != nil && tr.WroteRequest != nil {
			tr.WroteRequest(httptrace.WroteRequestInfo{})
		}
		return &http.Response{StatusCode: 400, Request: r, Header: http.Header{}, Body: failingBody{}}, nil
	})
	s := openOver(t, cut)

	err := s.PostJSON(context.Background(), batchURL(), suggestBatch(), &struct{}{})
	if err == nil {
		t.Fatal("a 400 with an unreadable body came back as success")
	}
	if wasUnknown(err) {
		t.Errorf("error %q is marked unknown, and a 400 is Docs refusing the batch whole", err)
	}
	if wasSent(err) {
		t.Errorf("error %q is marked sent, and a 400 is a change that did not happen", err)
	}
}
