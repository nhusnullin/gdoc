// This file is the browser trip. It builds the authorization URL, prints it,
// waits on the loopback listener and exchanges the code. No browser is opened
// and no external program runs: gdoc runs none at all, which is why the URL
// goes to stderr for a person to open.

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gdoc/internal/auth/loopback"
)

// authEndpoint is a page the human opens in their own browser. gdoc never
// requests it, so it is not in the guard's host allowlist.
const authEndpoint = "https://accounts.google.com/o/oauth2/auth"

// loginTimeout is how long the listener waits for the browser to come back.
const loginTimeout = 3 * time.Minute

// loginScopes is what one browser trip asks for: the full Drive scope, and the
// read/write Docs scope.
//
// The Docs scope here is documents and not documents.readonly, because gdoc
// writes suggestions through the Docs API and the read-only scope cannot carry
// a batchUpdate. The two sets are not equal, and the difference runs one way. A
// token granting Drive plus documents.readonly is missing nothing gdoc needs,
// because Drive covers the Docs calls; see coveredBy below. A token this login
// writes records documents and not documents.readonly, so a reader that asks
// for the read-only scope by name asks for its own sign-in. Do not fold the two
// sets into one by calling them equal.
var loginScopes = []string{
	"https://www.googleapis.com/auth/drive",
	"https://www.googleapis.com/auth/documents",
}

// randomToken is the source of both the PKCE verifier and the state. Both must
// be unguessable, and both are fresh per login.
//
// crypto/rand.Read is documented never to fail on any supported platform: it
// fills the slice or the program dies. That is the guarantee this line rests
// on. If the source is ever swapped for one that can fail, this has to return
// an error, because a silent failure hands out an all-zero verifier and state.
func randomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func buildAuthURL(redirect, state string) (verifier, authURL string) {
	verifier = randomToken()
	sum := sha256.Sum256([]byte(verifier))
	q := url.Values{
		"client_id":             {BundledClientID},
		"redirect_uri":          {redirect},
		"response_type":         {"code"},
		"scope":                 {strings.Join(loginScopes, " ")},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
		"access_type":           {"offline"},
		"prompt":                {"consent"},
	}
	return verifier, authEndpoint + "?" + q.Encode()
}

// exchangeCode turns the callback's code into a token. The client comes from
// internal/guard, so this POST is judged like every other request.
func exchangeCode(ctx context.Context, c *http.Client, code, verifier, redirect string) (Token, error) {
	r, err := postTokenForm(ctx, c, TokenURI, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"client_id":     {BundledClientID},
		"client_secret": {BundledClientSecret},
		"redirect_uri":  {redirect},
	}, "code exchange")
	if err != nil {
		return Token{}, err
	}
	// A code exchange with no refresh token is not a login. The access token
	// would work for about an hour, then every refresh would post an empty
	// refresh_token and fail for good, with no way back except a fresh login,
	// and Save would already have written over the token that did work. The
	// request carries access_type=offline with prompt=consent, so a missing
	// refresh token is an anomaly rather than a normal reply.
	if r.RefreshToken == "" {
		return Token{}, fmt.Errorf("code exchange returned 200 with no refresh token, " +
			"so the sign-in would stop working within the hour. Nothing was saved. Try signing in again")
	}
	return Token{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		TokenURI:     TokenURI,
		ClientID:     BundledClientID,
		ClientSecret: BundledClientSecret,
		Scopes:       grantedScopes(r.Scope),
		Expiry:       time.Now().UTC().Add(time.Duration(r.ExpiresIn) * time.Second),
	}, nil
}

// grantedScopes is what the token actually carries, which is what belongs in
// the file. Recording the requested list instead would let auth status report
// scopes the token does not have, and a later 403 would have nothing in the
// file to explain it.
//
// Silence means the grant matched the request: RFC 6749 section 5.1 makes the
// scope field optional only in that case.
func grantedScopes(scope string) []string {
	if granted := strings.Fields(scope); len(granted) > 0 {
		return granted
	}
	return loginScopes
}

// coveredBy names the scopes that stand in for another one. The Docs API
// accepts the full Drive scope on documents.get and documents.batchUpdate, so a
// token holding drive can do everything the documents scope allows. Without
// this, a token that works reports a scope missing that it does not need, which
// is a warning on the working case, and a warning on the working case is one
// people learn to ignore. drive.file is deliberately not here: it reaches only
// files the app itself created.
var coveredBy = map[string][]string{
	"https://www.googleapis.com/auth/documents": {"https://www.googleapis.com/auth/drive"},
}

// MissingScopes names the scopes v2 asks for that a token does not carry. A
// partial grant is reported, never refused: the login worked, and the person
// has to be able to see why the Docs calls will fail.
func MissingScopes(have []string) []string {
	got := make(map[string]bool, len(have))
	for _, s := range have {
		got[s] = true
	}
	var missing []string
	for _, want := range loginScopes {
		if !satisfied(got, want) {
			missing = append(missing, want)
		}
	}
	return missing
}

func satisfied(got map[string]bool, want string) bool {
	if got[want] {
		return true
	}
	for _, wider := range coveredBy[want] {
		if got[wider] {
			return true
		}
	}
	return false
}

// Pending is a login that has been started: the listener is up, the URL is
// built, and nobody has opened it yet. It is the login in two halves, because
// the second front door hands the link to a person in one message and finishes
// the trip in another, with the listener alive in between.
//
// A Pending holds an open port, so every path that makes one closes it.
type Pending struct {
	// URL is the link the person opens in their own browser.
	URL string

	srv      *loopback.Server
	verifier string
	redirect string
	client   *http.Client
}

// StartLogin opens the listener and builds the authorization URL, and returns
// at once: nothing here waits for a browser. The listener is on a port the
// kernel picks on 127.0.0.1. No browser is opened: v2 runs no external
// programs at all.
//
// A build with no client secret is refused here rather than at the exchange
// minutes later, so nothing opens a listener or hands out a link that cannot
// work. TestStartLoginReturnsTheLinkAtOnce and
// TestStartLoginRefusesABuildWithNoClientSecret are the pins.
func StartLogin(c *http.Client) (*Pending, error) {
	if BundledClientSecret == "" {
		return nil, errors.New("this build carries no OAuth client secret, so it cannot sign anyone in. " +
			"A release build carries one; a local build needs GDOC_OAUTH_CLIENT_SECRET set when running make build")
	}
	state := randomToken()
	srv, err := loopback.Listen(state)
	if err != nil {
		return nil, err
	}
	redirect := "http://" + srv.Addr() + "/callback"
	verifier, authURL := buildAuthURL(redirect, state)
	return &Pending{URL: authURL, srv: srv, verifier: verifier, redirect: redirect, client: c}, nil
}

// Wait waits for the browser to come back, exchanges the code and saves the
// token, then tells the browser what happened. It gives up after loginTimeout,
// and the context ends it earlier.
//
// Every path past the callback answers the browser, through the defer: the
// person is looking at that tab, and a dropped connection tells them nothing.
// The page says "Signed in" only once the token is on disk.
// TestWaitExchangesAndSaves, TestThePageSaysSignedInOnlyAfterTheSave and
// TestTheBrowserIsAnsweredOnEveryPath are the pins.
func (p *Pending) Wait(ctx context.Context) (err error) {
	code, cerr := p.srv.WaitCode(ctx, loginTimeout)
	if cerr != nil {
		return cerr // no code arrived, so there is no held request to answer
	}
	defer func() { p.srv.Finish(err) }()
	var tok Token
	if tok, err = exchangeCode(ctx, p.client, code, p.verifier, p.redirect); err != nil {
		return err
	}
	err = Save(tok)
	return err
}

// Close frees the port. It is safe to call twice.
func (p *Pending) Close() { p.srv.Close() }

// Login prints the authorization URL to w (stderr: stdout is reserved for the
// one JSON object) and waits. It is StartLogin and Wait in a row, which is the
// whole of what the CLI's auth login does.
//
// The CLI login has no context of its own: nothing above it can cancel a run
// that is already waiting on a browser, and loginTimeout is what bounds the
// wait. The exchange after it is bounded by the client's timeout.
func Login(c *http.Client, w io.Writer) error {
	p, err := StartLogin(c)
	if err != nil {
		return err
	}
	defer p.Close()
	fmt.Fprint(w, LinkLine(p.URL))
	return p.Wait(context.Background())
}

// LinkAsk is the sentence that goes with the link, and LinkLine the sentence
// and the link together: the two lines every login has printed, and the two a
// pipe and a log still read. cmd/gdoc draws the sentence inside a box on a
// terminal and prints LinkLine itself everywhere else, which is why both are
// exported from here rather than written out twice.
//
// Nothing about a terminal lives in this package: no colour, no box and no
// escape byte. TestTheLinkLineIsTodays states both as the literal they are and
// TestLoginPrintsThroughTheLinkLine holds that Login goes through them.
const LinkAsk = "Open this link in your browser to sign in:"

// LinkLine is LinkAsk, the link under it, and a newline after each.
func LinkLine(url string) string { return LinkAsk + "\n" + url + "\n" }
