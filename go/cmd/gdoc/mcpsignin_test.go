package main

// A hold and the sign-in it was raised under. An approval is an approval of
// one write as one account, so a card raised under one sign-in sends nothing
// under the next: it is dropped, with its tool and one line. The rules are in
// mcphold.go, and these tests are here rather than in mcphold_test.go because
// that file is already at its length.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/auth"
	"gdoc/internal/presence"
)

// anotherRefreshToken is the same token file with another refresh token in it,
// which is what a sign-in as somebody else leaves behind. Nothing else of the
// file changes, so what the mark reads is the one field that moved.
const anotherRefreshToken = `{"token":"ya29.not-a-real-token",` +
	`"refresh_token":"1//not-a-real-refresh-either",` +
	`"client_id":"not-a-real-client.apps.googleusercontent.com",` +
	`"client_secret":"not-a-real-secret",` +
	`"token_uri":"https://oauth2.googleapis.com/token",` +
	`"scopes":["https://www.googleapis.com/auth/drive"],` +
	`"expiry":"2099-01-01T00:00:00Z"}`

// refreshedAccessToken is the same sign-in after an ordinary refresh: a new
// access token and a new expiry, the refresh token untouched.
const refreshedAccessToken = `{"token":"ya29.a-newer-not-real-token",` +
	`"refresh_token":"1//not-a-real-refresh",` +
	`"client_id":"not-a-real-client.apps.googleusercontent.com",` +
	`"client_secret":"not-a-real-secret",` +
	`"token_uri":"https://oauth2.googleapis.com/token",` +
	`"scopes":["https://www.googleapis.com/auth/drive"],` +
	`"expiry":"2099-01-01T00:00:00Z"}`

// signedInAs writes another token file over the one signedIn wrote, which is
// what a gdoc auth login, or another process's refresh, leaves at the path.
func signedInAs(t *testing.T, body string) {
	t.Helper()
	writeTokenFile(t, os.Getenv("GDOC_CONFIG_DIR"), body)
}

// signedOut removes the token file, the way gdoc auth logout does.
func signedOut(t *testing.T) {
	t.Helper()
	if err := os.Remove(filepath.Join(os.Getenv("GDOC_CONFIG_DIR"), "oauth-token.json")); err != nil {
		t.Fatal(err)
	}
}

// markNow is the mark of the sign-in on this computer as the test's own
// reading of it, so an assertion about what a hold remembers is not the
// implementation agreeing with itself.
func markNow(t *testing.T) string {
	t.Helper()
	tok, err := auth.Load()
	if err != nil {
		t.Fatalf("the test's own token: %v", err)
	}
	mark := auth.SignInMark(tok)
	if mark == "" {
		t.Fatal("the test's own token has no mark")
	}
	return mark
}

// A hold keeps the sign-in it was raised under, in this process and nowhere
// else, as a mark and never as a token or an account.
func TestAHoldRemembersTheSignInItWasHeldUnder(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	stubWire(t, replyWire(t))
	stubClock(t, holdClock)

	ch := callChat(t)
	watchConfirms(t, ch)
	held := heldReply(t, ch)

	if got := ch.holds.signIn(held.Data.Held.ID); got != markNow(t) {
		t.Errorf("the hold remembers %q, want the mark of the token it was raised under", got)
	}
	if ch.holds.signIn("ffffffffffff") != "" {
		t.Error("a hold this session never kept remembers a sign-in")
	}
}

// A release after somebody signed in as another account sends nothing, and the
// card is gone rather than kept: that write will never be sendable under the
// approval the person gave, and leaving the card would ask them to press it
// again as an account they never agreed to.
func TestAReleaseAfterTheSignInChangedSendsNothingAndDropsTheHold(t *testing.T) {
	dir := recording(t)
	signedIn(t)
	f := stubWire(t, replyWire(t))
	clock := stubClock(t, holdClock)

	ch := chatRecording(t)
	w := watchConfirms(t, ch)
	held := heldInChat(t, ch)
	id := held.Data.Held.ID

	// Somebody signs in as somebody else while the card is open.
	signedInAs(t, anotherRefreshToken)
	clock.pass(quietEnough)

	res := w.tool(mcpConfirmPrefix+id).
		Call(presence.WithChat(context.Background()), confirmArgs(t, held, nil))

	env := envelopeOf(t, res.Texts)
	if env.OK {
		t.Fatalf("a release under another sign-in went through: %s", res.Texts[0])
	}
	if !strings.Contains(env.Error, "the sign-in on this computer is not the one gdoc held this write under") {
		t.Errorf("the refusal does not say the sign-in changed: %q", env.Error)
	}
	if sent := f.writes(); len(sent) != 0 {
		t.Errorf("a release under another sign-in sent %v", sent)
	}
	if ch.holds.hold(id, clock.at) != nil {
		t.Error("the hold survived a sign-in it cannot be sent under")
	}
	if got := w.confirms(); len(got) != 0 {
		t.Errorf("the tool outlived the hold: %v", got)
	}
	leftOnce(t, presenceEvents(t, dir), "dropped")
}

// A sign-out is the same answer in its own words: there is nobody to send the
// write as, and whoever signs in next did not approve it.
func TestAReleaseAfterASignOutSendsNothingAndDropsTheHold(t *testing.T) {
	dir := recording(t)
	signedIn(t)
	f := stubWire(t, replyWire(t))
	clock := stubClock(t, holdClock)

	ch := chatRecording(t)
	w := watchConfirms(t, ch)
	held := heldInChat(t, ch)
	id := held.Data.Held.ID

	signedOut(t)
	clock.pass(quietEnough)

	res := w.tool(mcpConfirmPrefix+id).
		Call(presence.WithChat(context.Background()), confirmArgs(t, held, nil))

	env := envelopeOf(t, res.Texts)
	if env.OK {
		t.Fatalf("a release with nobody signed in went through: %s", res.Texts[0])
	}
	if !strings.Contains(env.Error, "nobody is signed in to Google on this computer any more") {
		t.Errorf("the refusal does not say the person was signed out: %q", env.Error)
	}
	if !strings.Contains(env.Error, "login tool") {
		t.Errorf("the refusal does not say to call the login tool: %q", env.Error)
	}
	if sent := f.writes(); len(sent) != 0 {
		t.Errorf("a release with no token sent %v", sent)
	}
	if ch.holds.hold(id, clock.at) != nil {
		t.Error("the hold survived the sign-out")
	}
	if got := w.confirms(); len(got) != 0 {
		t.Errorf("the tool outlived the hold: %v", got)
	}
	leftOnce(t, presenceEvents(t, dir), "dropped")
}

// The sign-in a hold was raised under is the refresh token's and nothing
// else's. A token file rewritten with the same refresh token is the same
// sign-in, whatever else moved in it, so an ordinary access-token refresh
// between the hold and the release changes nothing.
func TestAReleaseUnderTheSameSignInStillSends(t *testing.T) {
	for name, body := range map[string]string{
		"the same file":            signedInToken,
		"a refreshed access token": refreshedAccessToken,
	} {
		t.Run(name, func(t *testing.T) {
			dir := recording(t)
			signedIn(t)
			f := stubWire(t, replyWire(t))
			clock := stubClock(t, holdClock)

			ch := chatRecording(t)
			w := watchConfirms(t, ch)
			held := heldInChat(t, ch)

			signedInAs(t, body)
			clock.pass(quietEnough)

			res := w.tool(mcpConfirmPrefix+held.Data.Held.ID).
				Call(presence.WithChat(context.Background()), confirmArgs(t, held, nil))
			if env := envelopeOf(t, res.Texts); !env.OK {
				t.Fatalf("the release was refused: %s", env.Error)
			}
			if sent := f.writes(); len(sent) != 1 {
				t.Errorf("the released write sent %d requests, want one: %v", len(sent), sent)
			}
			leftOnce(t, presenceEvents(t, dir), "released")
		})
	}
}

// A token that is there and cannot be read is not knowing, and not knowing
// never resolves to sending. It does not resolve to taking the person's card
// away either: nothing is sent, the hold stays, and the sentence names the file
// so somebody can go and look.
func TestAnUnreadableTokenAtReleaseKeepsTheHoldAndSendsNothing(t *testing.T) {
	dir := recording(t)
	signedIn(t)
	f := stubWire(t, replyWire(t))
	clock := stubClock(t, holdClock)

	ch := chatRecording(t)
	w := watchConfirms(t, ch)
	held := heldInChat(t, ch)
	id := held.Data.Held.ID

	signedInAs(t, "this is not JSON")
	clock.pass(quietEnough)

	res := w.tool(mcpConfirmPrefix+id).
		Call(presence.WithChat(context.Background()), confirmArgs(t, held, nil))

	env := envelopeOf(t, res.Texts)
	if env.OK {
		t.Fatalf("a release gdoc could not read a token for went through: %s", res.Texts[0])
	}
	if !strings.Contains(env.Error, "cannot be parsed") {
		t.Errorf("the refusal does not name what is wrong with the file: %q", env.Error)
	}
	if !strings.Contains(env.Error, mcpSignInUnread) {
		t.Errorf("the refusal does not say the card is still open: %q", env.Error)
	}
	if sent := f.writes(); len(sent) != 0 {
		t.Errorf("a release gdoc could not read a token for sent %v", sent)
	}
	if ch.holds.hold(id, clock.at) == nil {
		t.Error("the hold was dropped over a token nobody could read")
	}
	if got := w.confirms(); len(got) != 1 {
		t.Errorf("the tool went with a release that was refused: %v", got)
	}
	for _, e := range presenceEvents(t, dir) {
		if e.Event != "held" {
			t.Errorf("the hold is still open and the record says it left: %+v", e)
		}
	}

	// And the hold is still sendable once the file is readable again.
	signedInAs(t, signedInToken)
	clock.pass(quietEnough)
	res = w.tool(mcpConfirmPrefix+id).
		Call(presence.WithChat(context.Background()), confirmArgs(t, held, nil))
	if env := envelopeOf(t, res.Texts); !env.OK {
		t.Fatalf("the hold did not survive the unreadable token: %s", env.Error)
	}
}

// The accepted strain, written down rather than worked around.
//
// Google rotates a refresh token on some refreshes, and gdoc saves what it is
// given. A session's own rotation is not a second sign-in to internal/gapi,
// which remembers what it wrote; a hold compares against the file, so a
// rotation between the hold and the release reads there as another sign-in and
// the card goes. That costs a person one card and one retry, and it fails
// safe, which is the side to fail on. The spec of 2026-10-17 names it.
func TestARotatedRefreshTokenRefusesTheRelease(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	signedIn(t)
	f := stubWire(t, replyWire(t))
	clock := stubClock(t, holdClock)

	ch := callChat(t)
	w := watchConfirms(t, ch)
	held := heldReply(t, ch)
	id := held.Data.Held.ID

	// The same account, refreshed, with Google handing back a new refresh
	// token. Nothing about the person changed.
	signedInAs(t, anotherRefreshToken)
	clock.pass(quietEnough)

	res := w.tool(mcpConfirmPrefix+id).Call(context.Background(), confirmArgs(t, held, nil))
	if env := envelopeOf(t, res.Texts); env.OK {
		t.Fatalf("the release went through after the refresh token rotated: %s", res.Texts[0])
	}
	if sent := f.writes(); len(sent) != 0 {
		t.Errorf("the release sent %v", sent)
	}
	if ch.holds.hold(id, clock.at) != nil {
		t.Error("the hold survived, and this test is the strain being accepted")
	}
}

// The sweep asks the same question the release does, so a card whose account
// changed leaves the client in its own time rather than waiting for somebody
// to press it. One line says it left, however many sweeps run.
func TestTheSweepDropsAHoldWhoseSignInChanged(t *testing.T) {
	dir := recording(t)
	signedIn(t)
	stubWire(t, replyWire(t))
	stubClock(t, holdClock)

	ch := chatRecording(t)
	w := watchConfirms(t, ch)
	held := heldInChat(t, ch)
	id := held.Data.Held.ID

	// A sweep under the sign-in the hold was raised under keeps it.
	ch.holds.sweep(now())
	if ch.holds.hold(id, now()) == nil {
		t.Fatal("the sweep dropped a hold whose sign-in had not changed")
	}

	signedInAs(t, anotherRefreshToken)
	ch.holds.sweep(now())
	ch.holds.sweep(now())

	if ch.holds.hold(id, now()) != nil {
		t.Error("the sweep kept a hold whose sign-in changed")
	}
	if got := w.confirms(); len(got) != 0 {
		t.Errorf("the tool outlived the sweep: %v", got)
	}
	leftOnce(t, presenceEvents(t, dir), "dropped")
}

// The mark is this process's own. It reaches no answer a model reads, no tool
// the client lists, and no line of the record: a hash of a credential is still
// a thing about a credential, and the record's allowlist says the account and
// the token are never in it.
func TestTheSignInMarkReachesNoAnswerNoToolAndNoLine(t *testing.T) {
	dir := recording(t)
	signedIn(t)
	stubWire(t, replyWire(t))
	clock := stubClock(t, holdClock)

	ch := chatRecording(t)
	w := watchConfirms(t, ch)
	held := heldInChat(t, ch)
	id := held.Data.Held.ID
	mark := markNow(t)
	// The card's own tool, read while the hold is open: the release below
	// takes it off the list, and a list read after that is an empty list,
	// which carries no word of anything.
	cards := w.confirms()
	if len(cards) == 0 {
		t.Fatal("the hold registered no confirm tool, and the tool is half of what a tools/list shows")
	}

	// Every answer the three routes give: the held one, a release under
	// another sign-in, and the sweep's own line.
	answers := []string{held.Error, held.Data.Held.Reason, held.Data.Held.Text}
	signedInAs(t, anotherRefreshToken)
	clock.pass(quietEnough)
	res := w.tool(mcpConfirmPrefix+id).
		Call(presence.WithChat(context.Background()), confirmArgs(t, held, nil))
	answers = append(answers, res.Texts...)

	// And every word of every tool this session lists, which is what a
	// tools/list puts in front of the model.
	for _, tool := range append(mcpTools(nilWriter{}, newMCPLogin(nilWriter{}), ch), cards...) {
		answers = append(answers, tool.Name, tool.Title, tool.Description, string(tool.Schema))
	}

	for _, secret := range []string{mark, "1//not-a-real-refresh", "not-a-real-secret"} {
		for _, said := range answers {
			if strings.Contains(said, secret) {
				t.Errorf("an answer or a tool carries %q: %s", secret, said)
			}
		}
		if line := presenceLine(t, dir); strings.Contains(line, secret) {
			t.Errorf("the record carries %q: %s", secret, line)
		}
	}
	// The one thing the record does say is that the hold is gone.
	leftOnce(t, presenceEvents(t, dir), "dropped")
}
