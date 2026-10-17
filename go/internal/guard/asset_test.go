package guard

// This file holds one subject: the asset read. What AllowAsset opens, spelled
// exactly, what it refuses on the same id, and what a policy nobody granted it
// refuses there, which is what every policy refused before this door existed.

import (
	"strings"
	"testing"
)

const (
	// assetID is the shape of id the house file names: a Drive file id, here a
	// stand-in for the firm's logo.
	assetID = "1J3uh6poVAneP_V_Pc5FuNI4-Gahlab0s"
	// assetRead is the one request the grant opens, spelled as the asset room
	// builds it. It is a literal here and a literal there, because two packages
	// that read each other's constant drift together and a test would never say
	// so.
	assetRead = "https://www.googleapis.com/drive/v3/files/" + assetID + "?alt=media&supportsAllDrives=true"
	// notGiven is the refusal a Drive file nobody handed in has always got, and
	// the one the grant leaves in place for every id and every shape it does
	// not admit.
	notGiven = "was not given to this command"
)

// The grant carries one method, one id and two parameters. Everything else
// about that id is not judged by it at all, so it falls through to the refusal
// that was there before, which is how a reader can tell a widening from a hole.
func TestTheAssetGrantAdmitsOneMediaReadAndNothingElse(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.AllowAsset(assetID)

	if err := p.Judge("GET", mustURL(t, assetRead), nil); err != nil {
		t.Fatalf("the grant must carry the one asset read: %v", err)
	}

	base := "https://www.googleapis.com/drive/v3/files/" + assetID
	for _, c := range []struct{ name, method, url string }{
		{"no alt at all", "GET", base + "?supportsAllDrives=true"},
		{"a metadata read", "GET", base},
		{"a field mask", "GET", base + "?fields=id,name&supportsAllDrives=true"},
		{"a field mask beside the media read", "GET", base + "?alt=media&supportsAllDrives=true&fields=id"},
		{"another alt", "GET", base + "?alt=json&supportsAllDrives=true"},
		{"alt twice", "GET", base + "?alt=media&alt=media&supportsAllDrives=true"},
		{"shared drives refused", "GET", base + "?alt=media&supportsAllDrives=false"},
		{"the media read without the shared drive", "GET", base + "?alt=media"},
		{"an export of the same id", "GET", base + "/export?alt=media&mimeType=text/plain"},
		{"a copy of it", "POST", base + "/copy"},
		{"a patch of it", "PATCH", base + "?alt=media&supportsAllDrives=true"},
		{"a delete of it", "DELETE", base + "?alt=media&supportsAllDrives=true"},
		{"a post to it", "POST", base + "?alt=media&supportsAllDrives=true"},
		{"an upload of it", "GET", "https://www.googleapis.com/upload/drive/v3/files/" + assetID + "?alt=media&supportsAllDrives=true"},
		{"its comments", "GET", base + "/comments"},
		{"a second id read the same way", "GET", "https://www.googleapis.com/drive/v3/files/ASSET2?alt=media&supportsAllDrives=true"},
		{"the document the command was given, as bytes", "GET", "https://www.googleapis.com/drive/v3/files/DOC1?alt=media&supportsAllDrives=true"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if err := p.Judge(c.method, mustURL(t, c.url), nil); err == nil {
				t.Fatal("want a refusal, got none")
			}
		})
	}
}

// Without the grant the read is as far away as it was before the grant
// existed, and the sentence names the file: a refused call usually means the
// command did not say which document it was for.
func TestWithoutTheAssetGrantTheReadIsRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)

	err := p.Judge("GET", mustURL(t, assetRead), nil)
	if err == nil {
		t.Fatal("a policy nobody granted an asset read must refuse it")
	}
	if !strings.Contains(err.Error(), notGiven) {
		t.Errorf("the refusal must be the one an id nobody handed in gets: %v", err)
	}
	if !strings.Contains(err.Error(), assetID) {
		t.Errorf("the refusal must name the file: %v", err)
	}
}

// The grant opens no document. A run that fetches the logo and was handed no
// document reaches no document, through Drive or through Docs, so the reachable
// set cannot grow through this door: a PNG carries no id to learn.
func TestTheAssetGrantAdmitsNoDocument(t *testing.T) {
	p := NewPolicy()
	p.AllowAsset(assetID)

	if err := p.Judge("GET", mustURL(t, assetRead), nil); err != nil {
		t.Fatalf("the grant alone must carry its own read: %v", err)
	}
	for _, c := range []struct{ name, url string }{
		{"the asset as a document", "https://docs.googleapis.com/v1/documents/" + assetID},
		{"a document read", "https://docs.googleapis.com/v1/documents/DOC1"},
		{"a Drive file read", "https://www.googleapis.com/drive/v3/files/DOC1"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if err := p.Judge("GET", mustURL(t, c.url), nil); err == nil {
				t.Fatal("the asset grant admitted a document")
			}
		})
	}
}

// The grant is per run, it names one object, and nothing writes it down: a
// policy nobody granted it carries none. It is the shape AllowCopy and
// AllowAccountRead already have.
func TestTheAssetGrantDiesWithThePolicy(t *testing.T) {
	p := NewPolicy()
	if p.assetFile() != "" {
		t.Fatal("a fresh policy must read nobody's asset")
	}
	p.AllowAsset(assetID)
	if p.assetFile() != assetID {
		t.Fatalf("the grant opened nothing: %q", p.assetFile())
	}
	if NewPolicy().assetFile() != "" {
		t.Fatal("the grant reached a policy nobody granted it to")
	}
	if err := NewPolicy().Judge("GET", mustURL(t, assetRead), nil); err == nil {
		t.Fatal("a second policy carried the first one's asset read")
	}
}

// An empty id opens nothing. A caller that has just shown it does not know
// which file it means must not be left with a grant that matches a path the
// guard reads as empty.
func TestAnEmptyAssetGrantOpensNothing(t *testing.T) {
	p := NewPolicy()
	p.AllowAsset("")
	if p.assetFile() != "" {
		t.Fatal("an empty grant was written down")
	}
	if err := p.Judge("GET", mustURL(t, "https://www.googleapis.com/drive/v3/files/?alt=media&supportsAllDrives=true"), nil); err == nil {
		t.Fatal("an empty grant admitted a read")
	}
}
