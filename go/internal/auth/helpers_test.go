package auth

// The fakes and fixtures every test file here shares. They live in one file so
// a test that needs the token endpoint does not grow a fourth stand-in for it.

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixture = `{"token":"OLD","refresh_token":"R1","token_uri":"https://oauth2.googleapis.com/token","client_id":"CID","client_secret":"CS","scopes":["https://www.googleapis.com/auth/drive"],"expiry":"2020-01-01T00:00:00Z"}`

func writeFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	path := filepath.Join(dir, "oauth-token.json")
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// formRT stands in for the token endpoint. It records the form the request
// carried, so a test can prove what went out, and answers with one canned
// body. Nothing here reaches the network.
type formRT struct {
	body   string // the answer; a 200 carrying a token when this is empty
	status int    // 200 when this is zero
	seen   url.Values
}

func (f *formRT) RoundTrip(r *http.Request) (*http.Response, error) {
	b, _ := io.ReadAll(r.Body)
	f.seen, _ = url.ParseQuery(string(b))
	status, body := f.status, f.body
	if status == 0 {
		status = 200
	}
	if body == "" {
		body = `{"access_token":"A","refresh_token":"R","expires_in":3600}`
	}
	return &http.Response{StatusCode: status, Request: r,
		Body: io.NopCloser(strings.NewReader(body))}, nil
}

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// jsonRT answers every token request with one canned body, and looks at
// nothing. It is formRT's other half, for the tests that care about the answer
// rather than the question.
func jsonRT(body string) rtFunc {
	return func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Request: r,
			Body: io.NopCloser(strings.NewReader(body))}, nil
	}
}

// chanWriter hands the login URL to the test the moment Login prints it, so
// the test can play the browser.
type chanWriter struct{ lines chan string }

func (c chanWriter) Write(p []byte) (int, error) {
	c.lines <- string(p)
	return len(p), nil
}

// urlFrom pulls the authorization URL out of the sentence Login prints.
func urlFrom(t *testing.T, printed string) *url.URL {
	t.Helper()
	i := strings.Index(printed, "https://")
	if i < 0 {
		t.Fatalf("no URL in the login message: %q", printed)
	}
	u, err := url.Parse(strings.TrimSpace(printed[i:]))
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// callbackWith builds the redirect the browser would follow.
func callbackWith(t *testing.T, redirect string, q url.Values) string {
	t.Helper()
	u, err := url.Parse(redirect)
	if err != nil {
		t.Fatal(err)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
