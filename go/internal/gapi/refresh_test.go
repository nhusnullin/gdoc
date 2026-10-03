package gapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"gdoc/internal/auth"
	"gdoc/internal/guard"
)

// refreshWire is the fake wire the token race needs. It keeps the form of every
// token POST, so a test can read which refresh token was sent, and it runs
// onPost while a refresh is in flight, which is how a login landing between the
// refresh answer and the save is staged.
type refreshWire struct {
	mu      sync.Mutex
	forms   []url.Values
	bearers []string
	access  []string // the access token each refresh answers with, in order
	rotated []string // the refresh token each answer carries, in order; "" for none
	doc     []int    // the status each read answers, in order; 200 past the end
	onPost  func()   // runs after the form is recorded and before the answer
}

func (w *refreshWire) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host == "oauth2.googleapis.com" {
		raw, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(raw))
		w.mu.Lock()
		w.forms = append(w.forms, form)
		n := len(w.forms) - 1
		w.mu.Unlock()
		if w.onPost != nil {
			w.onPost()
		}
		tok := "NEW"
		if n < len(w.access) {
			tok = w.access[n]
		}
		// Google omits the refresh token on most refreshes and rotates it on
		// some, and the two are different answers to the session.
		rotated := ""
		if n < len(w.rotated) {
			rotated = `,"refresh_token":"` + w.rotated[n] + `"`
		}
		return reply(r, 200, `{"access_token":"`+tok+`","expires_in":3600`+rotated+`}`), nil
	}
	w.mu.Lock()
	w.bearers = append(w.bearers, r.Header.Get("Authorization"))
	n := len(w.bearers) - 1
	w.mu.Unlock()
	status := 200
	if n < len(w.doc) {
		status = w.doc[n]
	}
	return reply(r, status, `{"documentId":"`+docID+`"}`), nil
}

func (w *refreshWire) tokenPosts() []url.Values {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]url.Values{}, w.forms...)
}

func (w *refreshWire) reads() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string{}, w.bearers...)
}

// aLogin is the token file a `gdoc auth login` leaves behind.
func aLogin(access, refreshToken string, expiry time.Time) auth.Token {
	return auth.Token{
		AccessToken:  access,
		RefreshToken: refreshToken,
		TokenURI:     auth.TokenURI,
		ClientID:     "CID",
		ClientSecret: "CS",
		Scopes:       []string{"https://www.googleapis.com/auth/drive"},
		Expiry:       expiry,
	}
}

func writeToken(t *testing.T, path string, tok auth.Token) {
	t.Helper()
	b, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func fileBytes(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func saidNewerLogin(warnings []string) bool {
	for _, w := range warnings {
		if strings.Contains(w, "newer sign-in") {
			return true
		}
	}
	return false
}

// TestARefreshAfterANewerLoginSavesNothing is the race the whole rule exists
// for: a session opened before a re-login would otherwise write the old
// account over the new one an hour later.
func TestARefreshAfterANewerLoginSavesNothing(t *testing.T) {
	path := tokenFile(t, time.Now().Add(-time.Hour))
	w := &refreshWire{}
	s := openOver(t, w) // the session holds A, expired

	writeToken(t, path, aLogin("B", "R2", time.Now().Add(time.Hour)))
	before := fileBytes(t, path)

	if err := s.GetJSON(context.Background(), docURL(), &struct{}{}); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}

	if posts := w.tokenPosts(); len(posts) != 0 {
		t.Errorf("sent %d refreshes, want none: the file already held a newer login", len(posts))
	}
	if got := w.reads(); len(got) != 1 || got[0] != "Bearer B" {
		t.Errorf("the read carried %v, want the newer login's access token", got)
	}
	if after := fileBytes(t, path); !bytes.Equal(before, after) {
		t.Errorf("the token file was rewritten:\n%s", after)
	}
	if !saidNewerLogin(s.Warnings()) {
		t.Errorf("Warnings() = %v, want the newer sign-in said", s.Warnings())
	}
	for _, warn := range s.Warnings() {
		if strings.Contains(warn, "refreshed and saved") {
			t.Errorf("Warnings() = %v, want nothing claiming a save", s.Warnings())
		}
	}
}

// TestARefreshAfterANewerExpiredLoginRefreshesThatOne is the same race where
// the newer login is itself past use: the session refreshes the file's token
// and never its own, which belongs to the account that was signed out.
func TestARefreshAfterANewerExpiredLoginRefreshesThatOne(t *testing.T) {
	path := tokenFile(t, time.Now().Add(-time.Hour))
	w := &refreshWire{access: []string{"FRESH"}}
	s := openOver(t, w)

	writeToken(t, path, aLogin("B", "R2", time.Now().Add(-time.Hour)))

	if err := s.GetJSON(context.Background(), docURL(), &struct{}{}); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}

	posts := w.tokenPosts()
	if len(posts) != 1 {
		t.Fatalf("sent %d refreshes, want 1: %v", len(posts), posts)
	}
	if got := posts[0].Get("refresh_token"); got != "R2" {
		t.Errorf("refreshed %q, want R2, the newer login's refresh token", got)
	}
	if got := w.reads(); len(got) != 1 || got[0] != "Bearer FRESH" {
		t.Errorf("the read carried %v, want the refreshed token", got)
	}
	saved := savedToken(t, path)
	if saved.AccessToken != "FRESH" || saved.RefreshToken != "R2" {
		t.Errorf("saved token = %q/%q, want FRESH/R2", saved.AccessToken, saved.RefreshToken)
	}
}

// TestALoginBetweenTheRefreshAndTheSaveWins closes the window the first test
// leaves: the file is read again just before the save, so a login that landed
// while the refresh was in flight is not written over either.
func TestALoginBetweenTheRefreshAndTheSaveWins(t *testing.T) {
	path := tokenFile(t, time.Now().Add(-time.Hour))
	w := &refreshWire{}
	var before []byte
	w.onPost = func() {
		writeToken(t, path, aLogin("C", "R3", time.Now().Add(time.Hour)))
		before = fileBytes(t, path)
	}
	s := openOver(t, w)

	if err := s.GetJSON(context.Background(), docURL(), &struct{}{}); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}

	posts := w.tokenPosts()
	if len(posts) != 1 {
		t.Fatalf("sent %d refreshes, want 1: %v", len(posts), posts)
	}
	if got := posts[0].Get("refresh_token"); got != "R1" {
		t.Errorf("refreshed %q, want R1: the login landed after the request went out", got)
	}
	if got := w.reads(); len(got) != 1 || got[0] != "Bearer C" {
		t.Errorf("the read carried %v, want the login that landed mid-refresh", got)
	}
	if after := fileBytes(t, path); !bytes.Equal(before, after) {
		t.Errorf("the token file was rewritten:\n%s", after)
	}
}

// TestARefreshNeverWritesAnEmptyRefreshToken pins the one save that must never
// happen whatever the answer held. The shape cannot come out of auth.Load,
// which refuses a file carrying no refresh token, so the test builds the
// session: the guard sits on the save because the save is what cannot be taken
// back.
func TestARefreshNeverWritesAnEmptyRefreshToken(t *testing.T) {
	path := tokenFile(t, time.Now().Add(-time.Hour))
	before := fileBytes(t, path)
	w := &refreshWire{}
	p := guard.NewPolicy()
	p.AllowFile(docID, guard.LevelSuggest)
	s := &Session{
		policy: p,
		client: guard.NewClient(p, w),
		token: auth.Token{AccessToken: "OLD", TokenURI: auth.TokenURI,
			ClientID: "CID", ClientSecret: "CS"},
		loaded: "R1", // what the file holds, so no newer login is seen
	}

	err := s.refresh(context.Background())
	if err == nil {
		t.Fatal("a token with no refresh token was accepted")
	}
	if !strings.Contains(err.Error(), "refresh token") {
		t.Errorf("error = %q, want it to name the missing refresh token", err)
	}
	if after := fileBytes(t, path); !bytes.Equal(before, after) {
		t.Errorf("the token file was rewritten:\n%s", after)
	}
}

// TestTwoRefreshesOfOneLoginBothSave is the ordinary path, and it is here
// because the rule above must not reach it: Google omits the refresh token on a
// refresh, so both saves carry the login's own, and neither looks like somebody
// else's.
func TestTwoRefreshesOfOneLoginBothSave(t *testing.T) {
	path := tokenFile(t, time.Now().Add(-time.Hour))
	w := &refreshWire{access: []string{"NEW1", "NEW2"}, doc: []int{http.StatusUnauthorized}}
	s := openOver(t, w)

	if err := s.GetJSON(context.Background(), docURL(), &struct{}{}); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}

	posts := w.tokenPosts()
	if len(posts) != 2 {
		t.Fatalf("sent %d refreshes, want 2: the expired token then the 401", len(posts))
	}
	for i, post := range posts {
		if got := post.Get("refresh_token"); got != "R1" {
			t.Errorf("refresh %d sent %q, want R1", i+1, got)
		}
	}
	if got := w.reads(); len(got) != 2 || got[0] != "Bearer NEW1" || got[1] != "Bearer NEW2" {
		t.Errorf("the reads carried %v, want NEW1 then NEW2", got)
	}
	saved := savedToken(t, path)
	if saved.AccessToken != "NEW2" || saved.RefreshToken != "R1" {
		t.Errorf("saved token = %q/%q, want NEW2/R1", saved.AccessToken, saved.RefreshToken)
	}
	if saidNewerLogin(s.Warnings()) {
		t.Errorf("Warnings() = %v, want no newer sign-in on the ordinary path", s.Warnings())
	}
}

// TestARotatedRefreshTokenIsNotASecondLogin is the save the session has to
// recognise as its own. Google usually omits the refresh token on a refresh and
// sometimes rotates it, and a session that remembered only the string the file
// held when it opened would read its own rotated save as somebody signing in
// again: it would warn about a newer sign-in that never happened, and then
// refuse to refresh a token it believed belonged to another account.
func TestARotatedRefreshTokenIsNotASecondLogin(t *testing.T) {
	path := tokenFile(t, time.Now().Add(-time.Hour)) // the session holds R1, expired
	w := &refreshWire{
		access:  []string{"NEW1", "NEW2"},
		rotated: []string{"R2"}, // the first refresh rotates, the second does not
		doc:     []int{http.StatusUnauthorized},
	}
	s := openOver(t, w)

	if err := s.GetJSON(context.Background(), docURL(), &struct{}{}); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}

	posts := w.tokenPosts()
	if len(posts) != 2 {
		t.Fatalf("sent %d refreshes, want 2: the expired token then the 401: %v", len(posts), posts)
	}
	if got := posts[0].Get("refresh_token"); got != "R1" {
		t.Errorf("refresh 1 sent %q, want R1", got)
	}
	if got := posts[1].Get("refresh_token"); got != "R2" {
		t.Errorf("refresh 2 sent %q, want R2, the rotated token this session saved itself", got)
	}
	if got := w.reads(); len(got) != 2 || got[0] != "Bearer NEW1" || got[1] != "Bearer NEW2" {
		t.Errorf("the reads carried %v, want NEW1 then NEW2", got)
	}
	saved := savedToken(t, path)
	if saved.AccessToken != "NEW2" || saved.RefreshToken != "R2" {
		t.Errorf("saved token = %q/%q, want NEW2/R2", saved.AccessToken, saved.RefreshToken)
	}
	if saidNewerLogin(s.Warnings()) {
		t.Errorf("Warnings() = %v, want no newer sign-in: the rotation was this session's own save", s.Warnings())
	}
}

// TestAnAdoptedExpiredLoginSaysOnlyOneThingAboutTheSave is the envelope read by
// a person. Adopting the file's token writes nothing, and refreshing an adopted
// token that is itself expired does write, so one run reaches both sentences. Two
// warnings disagreeing about whether the token file was written is the envelope
// saying it does not know what it just did.
func TestAnAdoptedExpiredLoginSaysOnlyOneThingAboutTheSave(t *testing.T) {
	path := tokenFile(t, time.Now().Add(-time.Hour))
	w := &refreshWire{access: []string{"FRESH"}}
	s := openOver(t, w)

	writeToken(t, path, aLogin("B", "R2", time.Now().Add(-time.Hour)))

	if err := s.GetJSON(context.Background(), docURL(), &struct{}{}); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}

	if !saidNewerLogin(s.Warnings()) {
		t.Errorf("Warnings() = %v, want the newer sign-in said", s.Warnings())
	}
	if !hasWarning(s.Warnings(), "refreshed and saved") {
		t.Errorf("Warnings() = %v, want the save said: the adopted token was refreshed and written", s.Warnings())
	}
	for _, warn := range s.Warnings() {
		if strings.Contains(warn, "nothing was saved") {
			t.Errorf("Warnings() = %v, want nothing denying a save: %q, and the file was written", s.Warnings(), warn)
		}
	}
	if saved := savedToken(t, path); saved.AccessToken != "FRESH" {
		t.Errorf("saved access token = %q, want FRESH", saved.AccessToken)
	}
}

// TestATokenFileWithNoRefreshTokenIsRefusedRatherThanAdopted is why newerLogin
// compares the two refresh tokens and nothing else. An empty one in the file
// would compare unequal to the one this session holds and so read as somebody's
// newer sign-in, except that auth.Load refuses a token missing a field it needs
// before the comparison is ever made. The refresh stops, names the field and
// names the fix, and claims no sign-in it did not see.
func TestATokenFileWithNoRefreshTokenIsRefusedRatherThanAdopted(t *testing.T) {
	path := tokenFile(t, time.Now().Add(-time.Hour))
	w := &refreshWire{access: []string{"FRESH"}}
	s := openOver(t, w) // the session holds OLD, expired, with R1

	// Hand-edited, or written by something that dropped the field.
	writeToken(t, path, aLogin("B", "", time.Now().Add(-time.Hour)))

	err := s.GetJSON(context.Background(), docURL(), &struct{}{})
	if err == nil {
		t.Fatal("a token file gdoc cannot vouch for is not a file it refreshes over")
	}
	if !strings.Contains(err.Error(), "refresh_token") || !strings.Contains(err.Error(), "gdoc auth login") {
		t.Errorf("err = %v, want the missing field and the fix both named", err)
	}
	if posts := w.tokenPosts(); len(posts) != 0 {
		t.Errorf("sent %d refreshes, want none: the file was never vouched for", len(posts))
	}
	if saidNewerLogin(s.Warnings()) {
		t.Errorf("Warnings() = %v, want no sign-in claimed for a file with none in it", s.Warnings())
	}
}
