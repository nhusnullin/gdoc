// The stop code: which of the seven words a failed run writes into the record,
// read off the typed failures the packages raise.
//
// The sentence a person reads is not under test here, and barely a test below
// reads one: that is the point of the codes. What is under test is that every
// failure answers one of the seven, that the picture refusal brings its line
// with it, and that a failure nobody typed is other rather than nothing.

package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gdoc/internal/auth"
	"gdoc/internal/body"
	"gdoc/internal/emit"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/propose"
)

// timeoutError is what net/http hands back for a handshake that never
// finished: an error that is a net.Error and says it timed out. The real one
// is unexported, and what a caller matches on is the interface.
type timeoutError struct{}

func (timeoutError) Error() string   { return "net/http: TLS handshake timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

// guardRefusal is the real refusal an unopened policy gives: no document is in
// the reachable set, so a read of one is refused before the wire.
func guardRefusal(t *testing.T, method, rawURL string) error {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	refusal := guard.NewPolicy().Judge(method, u, nil)
	if refusal == nil {
		t.Fatalf("an unopened policy refuses %s %s, and this one did not", method, rawURL)
	}
	return refusal
}

// stopCase is one failure a command can hand up, with the one word the record
// gets for it and the line where the failure has one.
type stopCase struct {
	name string
	err  error
	code emit.StopCode
	line int
}

// stopCases is the whole set, built from the typed failures the packages raise
// rather than from sentences, because a code read out of English would follow
// the words wherever a milestone reworded them.
func stopCases(t *testing.T) []stopCase {
	t.Helper()
	return []stopCase{
		{"a picture the house style cannot carry", &body.PictureError{Line: 5}, emit.StopSVGPicture, 5},
		{"that picture under a sentence of its own",
			fmt.Errorf("the note could not be rendered: %w", &body.PictureError{Line: 12}),
			emit.StopSVGPicture, 12},
		{"a document with more than one tab",
			propose.MoreThanOneTab("the document has %d tabs, and a proposal names one", 3),
			emit.StopMultiTab, 0},
		{"Drive answering 404", &gapi.StatusError{Status: statusNotFound}, emit.StopNotFound, 0},
		{"the token refused twice", &gapi.StatusError{Status: statusUnauthorized}, emit.StopAuth, 0},
		{"no token on this computer", auth.ErrNoToken, emit.StopAuth, 0},
		{"a handshake that never finished", timeoutError{}, emit.StopNetwork, 0},
		{"that handshake inside the url error http wraps it in",
			&url.Error{Op: "Post", URL: "https://docs.googleapis.com/v1/documents", Err: timeoutError{}},
			emit.StopNetwork, 0},
		{"a dial nobody answered",
			&net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			emit.StopNetwork, 0},
		{"the guard refusing the request",
			guardRefusal(t, "GET", "https://www.googleapis.com/drive/v3/files/"+fixtureDocID),
			emit.StopRefused, 0},
		{"a note that is not there",
			&os.PathError{Op: "open", Path: "/Users/nail/notes/supplier.md", Err: errors.New("no such file or directory")},
			emit.StopOther, 0},
		{"a sentence nobody typed", errors.New("the quote was not found exactly once"), emit.StopOther, 0},
	}
}

// Every failure a command hands up answers one of the seven words, and none of
// them answers with a sentence. A code outside the set is the one thing the
// record may not carry: a reader of the record cannot tell a word gdoc meant
// from a word that leaked out of English.
func TestAStopIsOneOfTheSevenCodesAndNeverASentence(t *testing.T) {
	for _, c := range stopCases(t) {
		code, line := stopFor(c.err)
		if !code.Known() {
			t.Errorf("%s answered %q, which is not one of the seven", c.name, code)
		}
		if code != c.code || line != c.line {
			t.Errorf("%s answered %q and line %d, want %q and line %d", c.name, code, line, c.code, c.line)
		}
	}

	// And the envelope a refusal becomes carries the code and never the words:
	// what a caller reads is the sentence, and what the record reads is the
	// word, from the one call that builds both.
	err := errors.New("the folder id /Users/nail/notes is not a Drive id")
	r := failed(err)
	if r.OK || r.Error != err.Error() {
		t.Errorf("the envelope lost the sentence a person reads: %+v", r)
	}
	if r.Stop != emit.StopOther || r.StopLine != 0 {
		t.Errorf("the envelope says stop %q and line %d", r.Stop, r.StopLine)
	}
}

// The one failure that has a line is the picture a note carries, and the line
// is the only way a person finds it in a long note. It reaches the record
// through the envelope, so what is measured here is the whole run.
func TestTheSVGRefusalCarriesItsLine(t *testing.T) {
	theHouseLogoCached(t)
	dir := os.Getenv("GDOC_CONFIG_DIR")
	note := noteWithAnSVG(t)

	var out strings.Builder
	if code := run(context.Background(),
		[]string{"publish", "--md", note, "--folder-id", testFolderID}, &out, io.Discard, presenceOn()); code == 0 {
		t.Fatalf("an SVG is refused, and this publish answered ok: %s", out.String())
	}

	end := endOf(t, presenceEvents(t, dir))
	if end.Stop != string(emit.StopSVGPicture) {
		t.Errorf("the end says stop %q, want %q", end.Stop, emit.StopSVGPicture)
	}
	if want := svgLineOf(t, note); end.Line != want {
		t.Errorf("the end says line %d, want %d", end.Line, want)
	}
	if end.OK == nil || *end.OK {
		t.Errorf("the end of a refused publish says ok %v", end.OK)
	}
	// And the object a caller reads is the object it always was: the code is
	// for the record alone.
	if strings.Contains(out.String(), `"stop"`) || strings.Contains(out.String(), `"line"`) {
		t.Errorf("the stop reached the object a caller reads: %s", out.String())
	}
}

// noteWithAnSVG is the publishable note every other publish test uses with one
// picture appended that the house style cannot carry.
func noteWithAnSVG(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("testdata", "build-note.md"))
	if err != nil {
		t.Fatal(err)
	}
	svg := base64.StdEncoding.EncodeToString([]byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`))
	text := string(src) + "\n![a diagram](data:image/svg+xml;base64," + svg + ")\n"
	path := filepath.Join(t.TempDir(), "supplier-register.md")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// svgLineOf is the line the picture is written on, counted the way a person
// reading the note in an editor counts.
func svgLineOf(t *testing.T, path string) int {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(b), "\n") {
		if strings.Contains(line, "a diagram") {
			return i + 1
		}
	}
	t.Fatalf("%s carries no picture", path)
	return 0
}

// Drive answering 404 for the id a command named is the failure a person fixes
// by looking at the link they pasted.
func TestANotFoundIsNotFound(t *testing.T) {
	code, line := stopFor(&gapi.StatusError{Status: statusNotFound})
	if code != emit.StopNotFound || line != 0 {
		t.Errorf("a 404 answered %q and line %d", code, line)
	}
	// A status outside the three a person meets is other rather than a guess:
	// 500 is Google being unwell, which none of the six name.
	if code, _ := stopFor(&gapi.StatusError{Status: 500}); code != emit.StopOther {
		t.Errorf("a 500 answered %q, want %q", code, emit.StopOther)
	}
}

// A document with more than one tab is the word's own command refusing it.
// `propose` reads the document itself before propose does, so the refusal that
// reaches the record is the one this file builds, and it has to be the typed
// one: a sentence built by hand here would be written down as other, for the
// one failure multi_tab is named after.
func TestAProposeIntoATabbedDocumentStopsAtMultiTab(t *testing.T) {
	dir := recording(t)
	f := stubWire(t, &fakeWire{answers: []*answer{
		{method: "GET", match: proposeDocID + "?includeTabsContent", json: twoTabsOf(proposeDocID)},
	}})
	from := tempFile(t, "proposals.json", oneProposal)

	got, code := runJSONSet(t, presenceOn(), "propose", proposeDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a proposal into two tabs is refused: %v (exit %d)", got, code)
	}
	if sent := f.writes(); len(sent) != 0 {
		t.Errorf("nothing is sent into a document with two tabs: %v", sent)
	}
	end := endOf(t, presenceEvents(t, dir))
	if end.Stop != string(emit.StopMultiTab) {
		t.Errorf("the end says stop %q, want %q", end.Stop, emit.StopMultiTab)
	}
}

// A proposal the wire refused is the word the wire gave. The sentence a person
// reads names the proposal by number, and the number is built here with
// Sprintf, so the failure underneath has to be handed to the record separately
// or a 404 halfway through a file of seven reads as other.
func TestAProposalRefusedOnTheWireKeepsTheWordTheWireGave(t *testing.T) {
	dir := recording(t)
	stubWire(t, &fakeWire{answers: []*answer{
		{method: "GET", match: proposeDocID + "?includeTabsContent",
			json: readFixture(t, "propose-before.json"), once: true},
		{method: "POST", match: proposeDocID + ":batchUpdate",
			err: &gapi.StatusError{Status: statusNotFound}},
	}})
	from := tempFile(t, "proposals.json", oneProposal)

	got, code := runJSONSet(t, presenceOn(), "propose", proposeDocID, "--from", from)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a proposal Drive refused must fail the run: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "proposal 1") {
		t.Errorf("error = %q, want the sentence that names the proposal", msg)
	}
	end := endOf(t, presenceEvents(t, dir))
	if end.Stop != string(emit.StopNotFound) {
		t.Errorf("the end says stop %q, want %q", end.Stop, emit.StopNotFound)
	}
}

// twoTabsOf is the one shape every write that names a range refuses on sight,
// under the id the caller named.
func twoTabsOf(docID string) string {
	return strings.Replace(twoTabDocument, fixtureDocID, docID, 1)
}

// The token refused and no token at all are the same answer to the person
// reading the record: this run needed a sign-in it did not have. A grant the
// token endpoint refused is the third way in, and it arrives wrapped in the
// sentence internal/gapi puts in front of it.
func TestAnAuthRefusalIsAuth(t *testing.T) {
	for _, err := range []error{
		&gapi.StatusError{Status: statusUnauthorized},
		auth.ErrNoToken,
		fmt.Errorf("%w at /x. Run: gdoc auth login", auth.ErrNoToken),
		auth.ErrGrantRefused,
		fmt.Errorf("the access token could not be refreshed: %w", auth.ErrGrantRefused),
	} {
		if code, _ := stopFor(err); code != emit.StopAuth {
			t.Errorf("%v answered %q, want %q", err, code, emit.StopAuth)
		}
	}
}

// A 403 is not the token. Google answers it for a document this account was
// never shared and GitHub for the rate limit on a read that carries no
// credential, so auth would send a person to sign in again over something a
// sign-in cannot touch. Other is the honest word, and the command's own
// sentence is where the reason is.
func TestAForbiddenIsNotTheTokenRefused(t *testing.T) {
	for _, err := range []error{
		&gapi.StatusError{Status: statusForbidden},
		fmt.Errorf("the read failed: %w", &gapi.StatusError{Status: statusForbidden}),
	} {
		if code, _ := stopFor(err); code != emit.StopOther {
			t.Errorf("%v answered %q, want %q", err, code, emit.StopOther)
		}
	}
}

// A request that never got an answer is network, whichever layer gave up: the
// dial, the name, the handshake or the deadline.
func TestAHandshakeTimeoutIsNetwork(t *testing.T) {
	for _, err := range []error{
		timeoutError{},
		&url.Error{Op: "Get", URL: "https://docs.googleapis.com/v1/documents/x", Err: timeoutError{}},
		&net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
		&net.DNSError{Err: "no such host", Name: "docs.googleapis.com"},
		context.DeadlineExceeded,
	} {
		if code, _ := stopFor(err); code != emit.StopNetwork {
			t.Errorf("%v answered %q, want %q", err, code, emit.StopNetwork)
		}
	}
	// A run the person stopped is not a network failure: nothing was wrong
	// with the wire, and the record says the run failed and no more.
	if code, _ := stopFor(context.Canceled); code != emit.StopOther {
		t.Errorf("a cancelled run answered %q, want %q", code, emit.StopOther)
	}
}

// The guard refusing to make a request at all is its own word, and it is read
// before network: net/http wraps a transport error in a *url.Error, which is
// itself a net.Error, so a refusal read in the other order would become a
// timeout nobody had.
func TestAGuardRefusalIsRefused(t *testing.T) {
	refusal := guardRefusal(t, "GET", "https://www.googleapis.com/drive/v3/files/"+fixtureDocID)
	for _, err := range []error{
		refusal,
		fmt.Errorf("the read failed: %w", refusal),
		&url.Error{Op: "Get", URL: "https://www.googleapis.com/drive/v3/files/x", Err: refusal},
	} {
		if code, _ := stopFor(err); code != emit.StopRefused {
			t.Errorf("%v answered %q, want %q", err, code, emit.StopRefused)
		}
	}
}

// A failure the six words do not name is other, and other is written rather
// than nothing: the record says a run failed and that gdoc did not sort the
// reason, where an absent stop would read as a run that failed for no reason.
func TestAnUnknownFailureIsOther(t *testing.T) {
	if code, line := stopFor(errors.New("the quote was not found exactly once")); code != emit.StopOther || line != 0 {
		t.Errorf("a sentence nobody typed answered %q and line %d", code, line)
	}
	if code, _ := stopFor(nil); code != "" {
		t.Errorf("no error answered %q, and a run that worked says nothing about why it stopped", code)
	}

	// And the hook fills the word in for a command that built its refusal by
	// hand, so no command file has to remember it.
	dir := recording(t)
	var out strings.Builder
	if code := run(context.Background(),
		[]string{"restyle", fixtureDocID, "--dry-run", "--from", "survey.json"}, &out, io.Discard, presenceOn()); code == 0 {
		t.Fatalf("restyle takes one of those two flags and not both: %s", out.String())
	}
	end := endOf(t, presenceEvents(t, dir))
	if end.Stop != string(emit.StopOther) {
		t.Errorf("a refusal nobody typed says stop %q, want %q", end.Stop, emit.StopOther)
	}
	if end.Line != 0 {
		t.Errorf("a refusal with no line says line %d", end.Line)
	}
}
