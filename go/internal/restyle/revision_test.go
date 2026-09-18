package restyle

// RevisionOf, read on its own. The apply loop never calls it, so the scripted
// session in apply_test.go fails every read by design and no test there can
// reach this function. It is still live production code on one path, the
// prelude phase boundary in cmd/gdoc/restyle.go, and that path only runs when a
// marker batch came back naming no revision, so a live run would not reliably
// catch a break. Each of its four behaviours is pinned here against a session
// of this file's own.

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"gdoc/internal/docs"
)

// reader is one document read: the body it answers with, or the error it fails
// with, and the URL it was asked for.
type reader struct {
	body string
	err  error
	url  string
}

func (r *reader) GetJSON(ctx context.Context, rawURL string, into any) error {
	r.url = rawURL
	if r.err != nil {
		return r.err
	}
	return json.Unmarshal([]byte(r.body), into)
}

func (r *reader) PostJSON(ctx context.Context, rawURL string, body any, into any) error {
	return errors.New("RevisionOf must not write: it posted to " + rawURL)
}

// TestRevisionOfReadsTheNamedRangesURLAndReturnsTheRevision is the happy path,
// and it pins the route as well as the answer: the read is the named ranges
// URL, which is the smallest read that carries a revision id.
func TestRevisionOfReadsTheNamedRangesURLAndReturnsTheRevision(t *testing.T) {
	s := &reader{body: `{"revisionId":"rev-7"}`}

	got, err := RevisionOf(context.Background(), s, "doc-1")

	if err != nil {
		t.Fatalf("RevisionOf failed: %v", err)
	}
	if got != "rev-7" {
		t.Errorf("the revision is %q, want %q", got, "rev-7")
	}
	if want := docs.NamedRangesURL("doc-1"); s.url != want {
		t.Errorf("the read went to %q, want %q", s.url, want)
	}
}

// TestRevisionOfPassesTheTransportErrorThrough: the read failing is the
// caller's to report, and wrapping it here would say this room knew something
// about it that it does not.
func TestRevisionOfPassesTheTransportErrorThrough(t *testing.T) {
	boom := errors.New("the network went away")
	s := &reader{err: boom}

	_, err := RevisionOf(context.Background(), s, "doc-1")

	if !errors.Is(err, boom) {
		t.Errorf("the error is %v, want the transport's own %v", err, boom)
	}
}

// TestRevisionOfNamesADecodeFailure: an answer this room cannot read is
// refused by name rather than treated as an answer carrying no revision.
func TestRevisionOfNamesADecodeFailure(t *testing.T) {
	s := &reader{body: `{"revisionId":42}`}

	_, err := RevisionOf(context.Background(), s, "doc-1")

	if err == nil {
		t.Fatal("an answer that does not decode was accepted")
	}
	if !strings.Contains(err.Error(), "did not decode") {
		t.Errorf("the error is %q, want it to say the read did not decode", err)
	}
}

// TestRevisionOfRefusesAnAnswerCarryingNoRevision: an empty revision id is
// refused rather than returned, because a batch is never sent without one and
// an empty string would be sent as one.
func TestRevisionOfRefusesAnAnswerCarryingNoRevision(t *testing.T) {
	s := &reader{body: `{}`}

	got, err := RevisionOf(context.Background(), s, "doc-1")

	if err == nil {
		t.Fatalf("an answer carrying no revision id was accepted as %q", got)
	}
	if !strings.Contains(err.Error(), "no revision id") {
		t.Errorf("the error is %q, want it to say the read carried no revision id", err)
	}
}
