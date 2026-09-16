package gapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"gdoc/internal/guard"
)

const (
	testRepo    = "nhusnullin/gdoc"
	testListing = "https://api.github.com/repos/nhusnullin/gdoc/releases"
	testZip     = "https://github.com/nhusnullin/gdoc/releases/download/v2.0.0/gdoc-v2.0.0-darwin-arm64.zip"
)

// openPlain builds an unauthenticated reach over the fake wire, with the
// update grant open.
func openPlain(t *testing.T, w *wire) *Plain {
	t.Helper()
	p := guard.NewPolicy()
	p.AllowUpdateFrom(testRepo)
	return OpenPlain(p, w)
}

func TestAPlainGetCarriesNoCredentialAtAll(t *testing.T) {
	// No token file is written and none is read: a Plain never loads one, so
	// there is no bearer to put on a request even by mistake.
	w := &wire{answer: func(int, *http.Request) (int, string) {
		return 200, `[{"tag_name":"v2.1.0"}]`
	}}
	pl := openPlain(t, w)

	var into []struct {
		Tag string `json:"tag_name"`
	}
	if err := pl.GetJSON(context.Background(), testListing, &into); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if len(into) != 1 || into[0].Tag != "v2.1.0" {
		t.Errorf("decoded %+v, want the one release the listing held", into)
	}
	reqs := w.requests()
	if len(reqs) != 1 {
		t.Fatalf("sent %d requests, want 1: %+v", len(reqs), reqs)
	}
	if len(reqs[0].Auth) != 0 {
		t.Errorf("the request carried %v, want no Authorization header", reqs[0].Auth)
	}
	if reqs[0].Host != "api.github.com" || reqs[0].Method != http.MethodGet {
		t.Errorf("sent %s %s, want GET api.github.com", reqs[0].Method, reqs[0].Host)
	}
	if reqs[0].Accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", reqs[0].Accept)
	}
	if len(pl.Warnings()) != 0 {
		t.Errorf("warnings = %v, want none", pl.Warnings())
	}
}

func TestTheListingIsBoundedByFiveSeconds(t *testing.T) {
	// A person typed `gdoc update` and is waiting at a terminal. GitHub not
	// answering is an answer, and five seconds is how long it takes to get it.
	var deadline time.Time
	var had bool
	w := &wire{answer: func(_ int, r *http.Request) (int, string) {
		deadline, had = r.Context().Deadline()
		return 200, `[]`
	}}
	pl := openPlain(t, w)
	if err := pl.GetJSON(context.Background(), testListing, &[]struct{}{}); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if !had {
		t.Fatal("the listing request carried no deadline")
	}
	if left := time.Until(deadline); left <= 0 || left > ListingTimeout {
		t.Errorf("the deadline was %v away, want at most %v", left, ListingTimeout)
	}
}

func TestADownloadRunsOnTheCallersOwnDeadline(t *testing.T) {
	// The zip is megabytes on whatever connection the machine has, so the
	// download is bounded by the caller and not by the listing's five seconds.
	var deadline time.Time
	var had bool
	w := &wire{answer: func(_ int, r *http.Request) (int, string) {
		deadline, had = r.Context().Deadline()
		return 200, "PK\x03\x04"
	}}
	pl := openPlain(t, w)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	body, err := pl.GetBytes(ctx, testZip, 1<<20)
	if err != nil {
		t.Fatalf("GetBytes: %v", err)
	}
	if string(body) != "PK\x03\x04" {
		t.Errorf("read %q, want the bytes the release served", body)
	}
	if !had {
		t.Fatal("the download carried no deadline; the caller set one")
	}
	if left := time.Until(deadline); left <= ListingTimeout {
		t.Errorf("the deadline was %v away, want the caller's minute rather than the listing's %v", left, ListingTimeout)
	}
	if got := w.requests()[0].Accept; got != "*/*" {
		t.Errorf("Accept = %q, want */* on a zip", got)
	}
}

func TestTheGuardJudgesAPlainGetLikeEveryOtherRequest(t *testing.T) {
	// No grant, no release: the client is the guard's, so an ungranted run
	// never reaches GitHub at all.
	w := &wire{}
	pl := OpenPlain(guard.NewPolicy(), w)
	err := pl.GetJSON(context.Background(), testListing, &[]struct{}{})
	if err == nil {
		t.Fatal("a policy with no update grant read the releases listing")
	}
	if !strings.Contains(err.Error(), "api.github.com") {
		t.Errorf("the refusal says %q, want it to name the host", err)
	}
	if len(w.requests()) != 0 {
		t.Errorf("%d requests left the machine, want none", len(w.requests()))
	}
}

func TestAPlainGetReportsWhatGitHubAnswered(t *testing.T) {
	w := &wire{answer: func(int, *http.Request) (int, string) {
		return 403, `{"message":"API rate limit exceeded"}`
	}}
	pl := openPlain(t, w)
	err := pl.GetJSON(context.Background(), testListing, &[]struct{}{})
	if err == nil {
		t.Fatal("a 403 read as a listing")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("the failure says %q, want it to carry the status", err)
	}
	// GitHub puts its message at the top level rather than under an error
	// object, and the rate limit is the one a person actually meets.
	if !strings.Contains(err.Error(), "API rate limit exceeded") {
		t.Errorf("the failure says %q, want it to carry GitHub's own message", err)
	}
}

func TestAPlainGetNamesABodyThatIsNotJSON(t *testing.T) {
	w := &wire{answer: func(int, *http.Request) (int, string) {
		return 200, "<html>a proxy sign-in page</html>"
	}}
	pl := openPlain(t, w)
	err := pl.GetJSON(context.Background(), testListing, &[]struct{}{})
	if err == nil {
		t.Fatal("HTML decoded as the releases listing")
	}
	if !strings.Contains(err.Error(), "not JSON") {
		t.Errorf("the failure says %q, want it to name the body", err)
	}
}

func TestAPlainGetRefusesABodyOverItsLimit(t *testing.T) {
	w := &wire{answer: func(int, *http.Request) (int, string) {
		return 200, strings.Repeat("x", 64)
	}}
	pl := openPlain(t, w)
	_, err := pl.GetBytes(context.Background(), testZip, 16)
	if err == nil {
		t.Fatal("a body over the limit was read whole")
	}
	if !strings.Contains(err.Error(), "16 bytes") {
		t.Errorf("the failure says %q, want it to name the limit", err)
	}
}
