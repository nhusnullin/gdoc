package guard

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

// neverCalled is the proof, not the assertion. A refusal that reaches the wire
// and is only noticed afterwards has already leaked the request. This base
// transport fails the test the moment anything reaches it, so "never reaches
// the transport" is enforced by the transport itself.
type neverCalled struct{ t *testing.T }

func (n neverCalled) RoundTrip(r *http.Request) (*http.Response, error) {
	n.t.Helper()
	n.t.Fatalf("a refused request reached the wire: %s %s", r.Method, r.URL)
	return nil, nil
}

// TestAcceptanceGuardRefusals is milestone 1's acceptance check, in one place.
// Each case builds the client over neverCalled, so the only way to pass is to
// be refused before the wire.
func TestAcceptanceGuardRefusals(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*Policy)
		do    func(*http.Client) error
		want  string
	}{
		{
			name:  "an id outside the set never reaches the transport",
			setup: func(p *Policy) { p.AllowFile("DOC1", LevelSuggest) },
			do: func(c *http.Client) error {
				_, err := c.Get("https://docs.googleapis.com/v1/documents/EVIL")
				return err
			},
			want: `document "EVIL" was not given to this command`,
		},
		{
			name:  "a direct edit on a handed-in id is refused",
			setup: func(p *Policy) { p.AllowFile("DOC1", LevelSuggest) },
			do: func(c *http.Client) error {
				_, err := c.Post("https://docs.googleapis.com/v1/documents/DOC1:batchUpdate",
					"application/json", bytes.NewReader([]byte(`{"requests":[]}`)))
				return err
			},
			want: `direct edit of "DOC1"`,
		},
		{
			name:  "a create aimed at an unnamed folder is refused",
			setup: func(p *Policy) { p.AllowCreateIn("FOLDER1") },
			do: func(c *http.Client) error {
				_, err := c.Post("https://www.googleapis.com/drive/v3/files",
					"application/json", bytes.NewReader([]byte(`{"parents":["SOMEONE_ELSES_FOLDER"]}`)))
				return err
			},
			want: `create must name exactly the folder "FOLDER1"`,
		},
		{
			name:  "an id in the set may not walk to an endpoint the guard refuses",
			setup: func(p *Policy) { p.AllowFile("DOC1", LevelSuggest) },
			do: func(c *http.Client) error {
				_, err := c.Get("https://www.googleapis.com/drive/v3/files/DOC1/../../../about")
				return err
			},
			want: `walks through ".."`,
		},
		{
			name:  "a create when no folder was named at all is refused",
			setup: func(p *Policy) { p.AllowFile("DOC1", LevelSuggest) },
			do: func(c *http.Client) error {
				_, err := c.Post("https://www.googleapis.com/drive/v3/files",
					"application/json", bytes.NewReader([]byte(`{"parents":["ANY"]}`)))
				return err
			},
			want: "files collection",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPolicy()
			tc.setup(p)
			err := tc.do(NewClient(p, neverCalled{t}))
			if err == nil {
				t.Fatal("want a refusal, got none")
			}
			if !strings.Contains(err.Error(), "guard refused") {
				t.Errorf("a refusal must say so: %v", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("a refusal must name the reason %q: %v", tc.want, err)
			}
			t.Logf("refused: %v", err)
		})
	}
}

// TestAcceptanceTheAllowedRequestDoesReachTheWire is the other direction. A
// guard that refused everything would pass every case above and be useless.
func TestAcceptanceTheAllowedRequestDoesReachTheWire(t *testing.T) {
	f := &fake{status: 200, body: `{"documentId":"DOC1"}`}
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	resp, err := NewClient(p, f).Get("https://docs.googleapis.com/v1/documents/DOC1")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(f.seen) != 1 {
		t.Fatalf("an allowed read must reach the wire exactly once, got %d", len(f.seen))
	}
}
