package guard

// This file holds one subject: the update door. What a policy without the
// grant refuses on GitHub's hosts, what AllowUpdateFrom opens and for which
// repository, and why no request to those hosts may carry a credential.

import (
	"net/http"
	"strings"
	"testing"
)

const (
	testRepo    = "nhusnullin/gdoc"
	testListing = "https://api.github.com/repos/nhusnullin/gdoc/releases"
	testAsset   = "https://github.com/nhusnullin/gdoc/releases/download/v2.0.0/gdoc-v2.0.0-darwin-arm64.zip"
	testObject  = "https://objects.githubusercontent.com/github-production-release-asset/1/2?X-Amz-Signature=abc"
)

// A policy nobody granted an update refuses the three hosts exactly as it did
// before this door existed, and it says which rule refused. This is the first
// half of the grant: without it, GitHub is as far away as any other host.
func TestWithoutTheUpdateGrantGitHubIsRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)

	for _, u := range []string{testListing, testAsset, testObject} {
		err := p.Judge("GET", mustURL(t, u), nil)
		if err == nil {
			t.Fatalf("want a refusal for %s, got none", u)
		}
		if !strings.Contains(err.Error(), "nothing granted one") {
			t.Errorf("the refusal must say no update was granted: %v", err)
		}
	}
}

// What the grant opens: the releases listing of the one repository, the
// download path under it, and the asset host the download redirects to. GET
// only, for the run's length.
func TestTheUpdateGrantCarriesTheListingTheDownloadAndTheAsset(t *testing.T) {
	p := NewPolicy()
	p.AllowUpdateFrom(testRepo)

	for _, u := range []string{testListing, testAsset, testObject} {
		if err := p.Judge("GET", mustURL(t, u), nil); err != nil {
			t.Errorf("the update grant must carry GET %s: %v", u, err)
		}
	}
}

// The attack surface of one read-only door: another method, another
// repository, another path on either host, a query the listing does not
// carry, and the Google hosts, which the grant does not touch.
func TestTheUpdateGrantOpensNothingBesideThoseThreeReads(t *testing.T) {
	p := NewPolicy()
	p.AllowUpdateFrom(testRepo)

	cases := []struct{ name, method, url, want string }{
		{"a write to the listing", "POST", testListing, "reads and never writes"},
		{"a write to the download", "POST", testAsset, "reads and never writes"},
		{"a write to the asset host", "PUT", testObject, "reads and never writes"},
		{"another repository's listing", "GET", "https://api.github.com/repos/someone/else/releases", "is not the releases listing"},
		{"another path on the API host", "GET", "https://api.github.com/repos/nhusnullin/gdoc/issues", "is not the releases listing"},
		{"the user endpoint", "GET", "https://api.github.com/user", "is not the releases listing"},
		{"a query the listing does not carry", "GET", testListing + "?per_page=100", "is not one this call carries"},
		{"another repository's download", "GET", "https://github.com/someone/else/releases/download/v1/x.zip", "release downloads of"},
		{"another path on the download host", "GET", "https://github.com/nhusnullin/gdoc/archive/main.zip", "release downloads of"},
		{"a download naming no asset", "GET", "https://github.com/nhusnullin/gdoc/releases/download/v2.0.0", "names a tag and an asset"},
		{"a download walking past the asset", "GET", "https://github.com/nhusnullin/gdoc/releases/download/v2.0.0/a/b", "names a tag and an asset"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := p.Judge(c.method, mustURL(t, c.url), nil)
			if err == nil {
				t.Fatal("want a refusal, got none")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("the refusal must say %q: %v", c.want, err)
			}
		})
	}
}

// The grant is not a door into the reachable set. A document nobody handed in
// is no more reachable for a run that is also allowed to look at releases, and
// the Google rules read exactly as they did.
func TestTheUpdateGrantAdmitsNoDocument(t *testing.T) {
	p := NewPolicy()
	p.AllowUpdateFrom(testRepo)

	cases := []struct{ name, method, url, want string }{
		{"a docs read", "GET", "https://docs.googleapis.com/v1/documents/DOC1", "was not given to this command"},
		{"a drive read", "GET", "https://www.googleapis.com/drive/v3/files/DOC1", "was not given to this command"},
		{"the repository as a document", "GET", "https://docs.googleapis.com/v1/documents/nhusnullin", "was not given to this command"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := p.Judge(c.method, mustURL(t, c.url), nil)
			if err == nil {
				t.Fatal("want a refusal, got none")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("the refusal must say %q: %v", c.want, err)
			}
		})
	}
}

// The grant is one repository, in AllowCopy's shape. A second call replaces
// the first, because a caller naming two repositories has made a mistake the
// guard must not turn into two reachable ones.
func TestASecondUpdateGrantReplacesTheFirst(t *testing.T) {
	p := NewPolicy()
	p.AllowUpdateFrom("someone/else")
	p.AllowUpdateFrom(testRepo)

	if err := p.Judge("GET", mustURL(t, testListing), nil); err != nil {
		t.Errorf("the second grant must be the one that stands: %v", err)
	}
	err := p.Judge("GET", mustURL(t, "https://api.github.com/repos/someone/else/releases"), nil)
	if err == nil {
		t.Fatal("the replaced grant still carries")
	}
	if !strings.Contains(err.Error(), "is not the releases listing") {
		t.Errorf("the refusal must name the listing rule: %v", err)
	}
}

// A grant the guard cannot read opens nothing, and says so on the envelope.
// That is AllowMarker's rule: a caller that cannot name a repository has made
// a mistake, and resolving it to a wider reach is the wrong direction to be
// wrong in. A refused second call takes the first back too.
func TestAnUnreadableUpdateGrantOpensNothing(t *testing.T) {
	for _, repo := range []string{"", "gdoc", "nhusnullin/gdoc/extra", "nhusnullin/", "/gdoc", "nhusnullin/gd oc", "../../etc"} {
		t.Run(repo, func(t *testing.T) {
			p := NewPolicy()
			p.AllowUpdateFrom(testRepo)
			p.AllowUpdateFrom(repo)

			if err := p.Judge("GET", mustURL(t, testListing), nil); err == nil {
				t.Fatal("the earlier grant stood after a grant the guard could not read")
			}
			if len(p.Warnings()) == 0 {
				t.Error("a grant the guard could not read is not on the envelope")
			}
		})
	}
}

// The only bearer gdoc holds is Google's, and it has no business on GitHub.
// A request to any of the three hosts carrying an Authorization header is
// refused before it leaves, granted or not, because the header allowlist runs
// above the policy.
func TestAnUpdateRequestCarriesNoBearer(t *testing.T) {
	for _, u := range []string{testListing, testAsset, testObject} {
		t.Run(u, func(t *testing.T) {
			f := &fake{status: 200, body: `{}`}
			p := NewPolicy()
			p.AllowUpdateFrom(testRepo)
			c := NewClient(p, f)

			req, err := http.NewRequest("GET", u, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "Bearer token")

			resp, err := c.Do(req)
			if err == nil {
				resp.Body.Close()
				t.Fatal("a credential reached GitHub")
			}
			if !strings.Contains(err.Error(), "carries no credential") {
				t.Errorf("the refusal must name the credential rule: %v", err)
			}
			if len(f.seen) != 0 {
				t.Fatal("the refused request reached the transport")
			}
		})
	}
}

// The three reads go out through the one client like every other request, and
// they go out bare. This is the success path of the test above.
func TestTheUpdateReadsGoOutWithNoCredential(t *testing.T) {
	f := &fake{status: 200, body: `[]`}
	p := NewPolicy()
	p.AllowUpdateFrom(testRepo)
	c := NewClient(p, f)

	resp, err := c.Get(testListing)
	if err != nil {
		t.Fatalf("the listing must carry: %v", err)
	}
	resp.Body.Close()
	if len(f.seen) != 1 {
		t.Fatalf("want one request on the wire, got %d", len(f.seen))
	}
	if got := f.seen[0].Header.Get("Authorization"); got != "" {
		t.Errorf("the update read carried a credential: %q", got)
	}
}
