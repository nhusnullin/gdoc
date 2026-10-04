package guard

// This file holds one subject: the account read. What AllowAccountRead opens,
// spelled exactly, what it refuses on the same path, and what a policy nobody
// granted it refuses there, which is what every policy refused before this
// door existed.

import (
	"strings"
	"testing"
)

const (
	// accountRead is the one request the grant opens, spelled as gapi builds
	// it. It is a literal here and a literal there: two packages that read
	// each other's constant drift together and a test would never say so.
	accountRead = "https://www.googleapis.com/drive/v3/about?fields=user(emailAddress,displayName)"
	// outsideFiles is the refusal every Drive path but the files collection
	// has always got, and the one the grant leaves in place for everything it
	// does not admit.
	outsideFiles = "outside /drive/v3/files"
)

// The grant carries one method, one path and one field mask. Everything else
// about `about` is not judged by it at all, so it falls through to the refusal
// that was there before, which is how a reader can tell a widening from a hole.
func TestTheAboutReadIsAdmittedWithItsFieldsAndNothingElse(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	p.AllowAccountRead()

	if err := p.Judge("GET", mustURL(t, accountRead), nil); err != nil {
		t.Fatalf("the grant must carry the one account read: %v", err)
	}

	for _, c := range []struct{ name, method, url string }{
		{"a narrower mask", "GET", "https://www.googleapis.com/drive/v3/about?fields=user"},
		{"every field", "GET", "https://www.googleapis.com/drive/v3/about?fields=*"},
		{"the storage quota as well", "GET", "https://www.googleapis.com/drive/v3/about?fields=user(emailAddress,displayName),storageQuota"},
		{"no mask at all", "GET", "https://www.googleapis.com/drive/v3/about"},
		{"a second parameter", "GET", "https://www.googleapis.com/drive/v3/about?fields=user(emailAddress,displayName)&alt=json"},
		{"the mask twice", "GET", "https://www.googleapis.com/drive/v3/about?fields=user(emailAddress,displayName)&fields=user(emailAddress,displayName)"},
		{"a path under about", "GET", "https://www.googleapis.com/drive/v3/about/user?fields=user(emailAddress,displayName)"},
		{"about as a prefix", "GET", "https://www.googleapis.com/drive/v3/aboutDOC1?fields=user(emailAddress,displayName)"},
		{"a post", "POST", accountRead},
		{"a patch", "PATCH", accountRead},
		{"a delete", "DELETE", accountRead},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := p.Judge(c.method, mustURL(t, c.url), nil)
			if err == nil {
				t.Fatal("want a refusal, got none")
			}
			if !strings.Contains(err.Error(), outsideFiles) {
				t.Errorf("the refusal must read as the one every other Drive path gets: %v", err)
			}
		})
	}
}

// Without the grant the read is as far away as it was before the grant
// existed, and the sentence is the same one. TestARefusalNamesTheRuleItApplied
// and TestJudge use this very request as their "outside the files collection"
// case, so this is the half that keeps them true.
func TestWithoutTheAccountGrantTheAboutReadIsRefusedAsBefore(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)

	err := p.Judge("GET", mustURL(t, accountRead), nil)
	if err == nil {
		t.Fatal("a policy nobody granted an account read must refuse it")
	}
	if !strings.Contains(err.Error(), outsideFiles) {
		t.Errorf("the refusal must be unchanged: %v", err)
	}
}

// The grant is per run, it opens no document, and nothing writes it down: a
// policy nobody granted it carries none. It is the shape AllowCopy and
// AllowUpdateFrom already have.
func TestTheAccountGrantDiesWithThePolicy(t *testing.T) {
	p := NewPolicy()
	if p.mayReadAccount() {
		t.Fatal("a fresh policy must read nobody's account")
	}
	p.AllowAccountRead()
	if !p.mayReadAccount() {
		t.Fatal("the grant opened nothing")
	}
	if NewPolicy().mayReadAccount() {
		t.Fatal("the grant reached a policy nobody granted it to")
	}

	// It is not a door into the reachable set. A run that reads the account
	// and was handed no document reaches no document.
	bare := NewPolicy()
	bare.AllowAccountRead()
	if err := bare.Judge("GET", mustURL(t, "https://www.googleapis.com/drive/v3/files/DOC1"), nil); err == nil {
		t.Fatal("the account grant admitted a document")
	}
	if err := bare.Judge("GET", mustURL(t, "https://docs.googleapis.com/v1/documents/DOC1"), nil); err == nil {
		t.Fatal("the account grant admitted a Docs read")
	}
}
