package gapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"gdoc/internal/comments"
	"gdoc/internal/docs"
	"gdoc/internal/docx"
	"gdoc/internal/guard"
)

// otherDocID is a document nobody named. It is a real-shaped id, so what
// refuses it is the policy rather than the id parser.
const otherDocID = "9ZzYyXxWwVvUuTtSsRrQqPpOoNn0123456789zzzz"

// neverCalled is the proof, not the assertion. Counting the requests a fake
// wire saw notices the leak after it happened; this transport fails the test
// the moment anything reaches it, so "never reaches the transport" is enforced
// by the transport itself. It is guard/acceptance_test.go's helper, brought to
// the room where a session is what carries a read.
type neverCalled struct{ t *testing.T }

func (n neverCalled) RoundTrip(r *http.Request) (*http.Response, error) {
	n.t.Helper()
	n.t.Fatalf("a refused read reached the wire: %s %s", r.Method, r.URL)
	return nil, nil
}

// TestAcceptanceEveryReadIsBoundedToTheDocumentTheCommandWasGiven is milestone
// 2's acceptance check for the guard, in one place.
//
// The policy is built the way cmd/gdoc's open() builds it: one AllowFile at
// LevelSuggest and nothing else. The URLs are the ones the three read commands
// actually send, taken from the packages that build them rather than retyped,
// so a URL that changes shape changes this test with it.
//
// Every case runs over neverCalled, so the only way to pass is to be refused
// before the wire.
func TestAcceptanceEveryReadIsBoundedToTheDocumentTheCommandWasGiven(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "the Docs read of another document",
			url:  docs.URL(otherDocID),
			want: "was not given to this command",
		},
		{
			name: "the comment listing of another document",
			url:  comments.ListURL(otherDocID, "", ""),
			want: "was not given to this command",
		},
		{
			name: "the docx export of another document",
			url:  docx.ExportURL(otherDocID),
			want: "was not given to this command",
		},
		{
			name: "a walk out of the allowed id to another endpoint",
			url:  "https://www.googleapis.com/drive/v3/files/" + docID + "/../../../about",
			want: `walks through ".."`,
		},
		{
			name: "the permission surface of the allowed id",
			url:  "https://www.googleapis.com/drive/v3/files/" + docID + "/permissions",
			want: "permissions",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tokenFile(t, time.Now().Add(time.Hour))
			p := guard.NewPolicy()
			p.AllowFile(docID, guard.LevelSuggest)

			s, err := Open(p, neverCalled{t})
			if err != nil {
				t.Fatal(err)
			}
			err = s.GetJSON(context.Background(), c.url, &struct{}{})
			if err == nil {
				t.Fatalf("%s was carried", c.name)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error = %q, want it to name %q", err, c.want)
			}
		})
	}
}

// GetBytes is the export's door, and it is the same guard. A read of another
// document through it must not reach the wire either.
func TestAcceptanceGetBytesIsBoundedToo(t *testing.T) {
	tokenFile(t, time.Now().Add(time.Hour))
	p := guard.NewPolicy()
	p.AllowFile(docID, guard.LevelSuggest)

	s, err := Open(p, neverCalled{t})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.GetBytes(context.Background(), docx.ExportURL(otherDocID), docx.MaxExportBytes)
	if err == nil {
		t.Fatal("the export of a document nobody named was carried")
	}
	if !strings.Contains(err.Error(), "was not given to this command") {
		t.Errorf("error = %q, want the guard's refusal", err)
	}
}
