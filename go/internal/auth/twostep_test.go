package auth

// The login in two calls: StartLogin hands back the link, Wait finishes the
// trip. The CLI runs the two in a row, and a second front door can hold the
// pending login between two messages.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// authQuery is the query of the link StartLogin handed back.
func authQuery(t *testing.T, p *Pending) url.Values {
	t.Helper()
	u, err := url.Parse(p.URL)
	if err != nil {
		t.Fatal(err)
	}
	return u.Query()
}

// browserPage plays the browser from its own goroutine. The callback request is
// held open until the save is done, so the test cannot make it inline.
type browserPage struct {
	body string
	err  error
}

func playBrowser(t *testing.T, q url.Values, code string) <-chan browserPage {
	t.Helper()
	cb := callbackWith(t, q.Get("redirect_uri"), url.Values{"code": {code}, "state": {q.Get("state")}})
	out := make(chan browserPage, 1)
	go func() {
		resp, err := http.Get(cb)
		if err != nil {
			out <- browserPage{err: err}
			return
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		out <- browserPage{body: string(b), err: err}
	}()
	return out
}

func waitForPage(t *testing.T, got <-chan browserPage) string {
	t.Helper()
	select {
	case p := <-got:
		if p.err != nil {
			t.Fatalf("the browser got no page: %v", p.err)
		}
		return p.body
	case <-time.After(5 * time.Second):
		t.Fatal("the browser was never answered")
		return ""
	}
}

// StartLogin listens and builds the link, and hands it back without waiting for
// anybody. That is what lets a chat answer with the link in the same turn it
// was asked for the sign-in.
func TestStartLoginReturnsTheLinkAtOnce(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())

	start := time.Now()
	p, err := StartLogin(&http.Client{Transport: refuseAll{}})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if time.Since(start) > 2*time.Second {
		t.Fatalf("StartLogin waited for something: %v", time.Since(start))
	}

	q := authQuery(t, p)
	redirect := q.Get("redirect_uri")
	if !strings.HasPrefix(redirect, "http://127.0.0.1:") {
		t.Fatalf("the link must name the loopback listener: %q", redirect)
	}
	if q.Get("state") == "" {
		t.Fatal("the link must carry this run's state")
	}
	// The listener named in the link is up: a callback carrying somebody
	// else's state reaches it and is refused.
	resp, err := http.Get(callbackWith(t, redirect, url.Values{"code": {"C"}, "state": {"WRONG"}}))
	if err != nil {
		t.Fatalf("the listener in the link is not up: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("a stray callback must not be answered with a success page: %d", resp.StatusCode)
	}
}

func TestWaitExchangesAndSaves(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	rt := &formRT{}
	p, err := StartLogin(&http.Client{Transport: rt})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	done := make(chan error, 1)
	go func() { done <- p.Wait(context.Background()) }()
	got := playBrowser(t, authQuery(t, p), "CODE7")

	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if rt.seen.Get("code") != "CODE7" {
		t.Fatalf("the code from the callback did not reach the exchange: %q", rt.seen.Get("code"))
	}
	saved, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.RefreshToken != "R" {
		t.Fatalf("Wait did not save the token: %+v", saved)
	}
	if page := waitForPage(t, got); !strings.Contains(page, "Signed in") {
		t.Fatalf("the browser must hear it worked: %q", page)
	}
}

// The page is the only place the person is looking, so it must say what gdoc
// knows and not a word more. Nothing is written until the token is on disk, and
// a save that failed says so.
func TestThePageSaysSignedInOnlyAfterTheSave(t *testing.T) {
	t.Run("nothing is written until the save is done", func(t *testing.T) {
		t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
		release := make(chan struct{})
		held := rtFunc(func(r *http.Request) (*http.Response, error) {
			<-release
			return &http.Response{StatusCode: 200, Request: r, Body: io.NopCloser(
				strings.NewReader(`{"access_token":"A","refresh_token":"R","expires_in":3600}`))}, nil
		})
		p, err := StartLogin(&http.Client{Transport: held})
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		done := make(chan error, 1)
		go func() { done <- p.Wait(context.Background()) }()
		got := playBrowser(t, authQuery(t, p), "CODE7")

		// The exchange is held, so nothing is saved and nothing is said.
		select {
		case page := <-got:
			t.Fatalf("the browser was answered before the save: %+v", page)
		case <-time.After(100 * time.Millisecond):
		}
		if _, err := Load(); err == nil {
			t.Fatal("nothing can be on disk yet")
		}

		close(release)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		if page := waitForPage(t, got); !strings.Contains(page, "Signed in") {
			t.Fatalf("the page must say the person is signed in: %q", page)
		}
		if _, err := Load(); err != nil {
			t.Fatalf("the page went out and the token is not on disk: %v", err)
		}
	})

	t.Run("a save that failed says so", func(t *testing.T) {
		t.Setenv("GDOC_CONFIG_DIR", unwritableDir(t))
		p, err := StartLogin(&http.Client{Transport: &formRT{}})
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()

		done := make(chan error, 1)
		go func() { done <- p.Wait(context.Background()) }()
		got := playBrowser(t, authQuery(t, p), "CODE7")

		err = <-done
		if err == nil {
			t.Fatal("a save that cannot be made must fail the login")
		}
		page := waitForPage(t, got)
		if !strings.Contains(page, "Sign-in failed") {
			t.Fatalf("the page must say the sign-in failed: %q", page)
		}
		if strings.Contains(page, "Signed in") {
			t.Fatalf("a failed save must not read as a success: %q", page)
		}
		if !strings.Contains(page, err.Error()) {
			t.Fatalf("the page must carry the reason %q: %q", err, page)
		}
	})
}

// unwritableDir names a config dir that cannot be created: its parent is a
// regular file. A read-only directory would not do, because root writes
// through one and the test would have to skip itself.
func unwritableDir(t *testing.T) string {
	t.Helper()
	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(blocked, "config")
}

// Once a code has arrived the browser is answered on every path, because the
// person is watching that tab and a dropped connection tells them nothing.
func TestTheBrowserIsAnsweredOnEveryPath(t *testing.T) {
	cases := []struct {
		name      string
		configDir func(t *testing.T) string
		transport func(cancel context.CancelFunc) http.RoundTripper
		want      string
	}{
		{
			name: "the exchange fails",
			transport: func(context.CancelFunc) http.RoundTripper {
				return rtFunc(func(r *http.Request) (*http.Response, error) {
					return nil, errors.New("the wire broke")
				})
			},
			want: "the wire broke",
		},
		{
			name:      "the save fails",
			configDir: unwritableDir,
			transport: func(context.CancelFunc) http.RoundTripper { return &formRT{} },
			want:      "Sign-in failed",
		},
		{
			name: "the context ends between the code and the token",
			transport: func(cancel context.CancelFunc) http.RoundTripper {
				return rtFunc(func(r *http.Request) (*http.Response, error) {
					cancel()
					return nil, context.Canceled
				})
			},
			want: "context canceled",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			if c.configDir != nil {
				dir = c.configDir(t)
			}
			t.Setenv("GDOC_CONFIG_DIR", dir)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p, err := StartLogin(&http.Client{Transport: c.transport(cancel)})
			if err != nil {
				t.Fatal(err)
			}

			done := make(chan error, 1)
			go func() { done <- p.Wait(ctx) }()
			got := playBrowser(t, authQuery(t, p), "CODE7")

			if err := <-done; err == nil {
				t.Fatal("this path must fail the login")
			}
			page := waitForPage(t, got)
			if !strings.Contains(page, c.want) {
				t.Fatalf("the page must say %q: %q", c.want, page)
			}
			if strings.Contains(page, "Signed in") {
				t.Fatalf("a login that failed must not read as a success: %q", page)
			}
			// Close straight after the page went out never drops it, and it is
			// safe twice.
			p.Close()
			p.Close()
		})
	}
}

// A build with no secret refuses before it opens a listener, and StartLogin is
// where that now happens: a chat tool that asked for a link must get the
// refusal instead of a link that cannot work.
func TestStartLoginRefusesABuildWithNoClientSecret(t *testing.T) {
	saved := BundledClientSecret
	BundledClientSecret = ""
	t.Cleanup(func() { BundledClientSecret = saved })

	p, err := StartLogin(&http.Client{Transport: refuseAll{}})
	if err == nil {
		p.Close()
		t.Fatal("a build with no client secret must refuse to start a login")
	}
	if p != nil {
		t.Fatal("a refused start must hand back no pending login")
	}
	for _, want := range []string{"client secret", "GDOC_OAUTH_CLIENT_SECRET"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must say %q: %q", want, err)
		}
	}
}
