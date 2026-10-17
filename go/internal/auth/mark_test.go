package auth

// The sign-in mark: one short string that tells one sign-in from another, and
// carries nothing of the token it was taken from. The rules are in doc.go.

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// The mark is a SHA-256 over a fixed prefix and the refresh token, and the
// test states both as literals: a test that read the constants would follow
// them wherever somebody moved them.
func TestTheSignInMarkIsTheDomainPrefixedHashOfTheRefreshToken(t *testing.T) {
	sum := sha256.Sum256([]byte("gdoc sign-in mark v1\x00" + "R1"))
	want := hex.EncodeToString(sum[:])

	got := SignInMark(Token{RefreshToken: "R1"})
	if got != want {
		t.Fatalf("the mark of R1 is %q, want %q", got, want)
	}
	if len(got) != 64 {
		t.Errorf("the mark is %d characters, want the 64 of a hex SHA-256", len(got))
	}
}

// The mark says which sign-in and never what the token is: a record or a log
// that carried it still carries no credential, and nothing can be signed in
// with it.
func TestTheSignInMarkCarriesNoWordOfTheToken(t *testing.T) {
	secret := "1//0gSECRETrefreshTOKENvalue"
	mark := SignInMark(Token{
		AccessToken:  "AT",
		RefreshToken: secret,
		ClientSecret: "CS",
		Account:      "someone@example.com",
	})
	for _, word := range []string{secret, "AT", "CS", "someone@example.com"} {
		if strings.Contains(mark, word) {
			t.Errorf("the mark carries %q: %s", word, mark)
		}
	}
}

// Two sign-ins are two marks, and one sign-in read twice is one mark. That is
// the whole of what a hold asks of it.
func TestTheSameRefreshTokenMarksTheSameAndAnotherOneDoesNot(t *testing.T) {
	first := SignInMark(Token{RefreshToken: "R1", AccessToken: "A1"})
	// The same refresh token, every other field different: an access token
	// refreshed in flight is the same sign-in and must mark the same, or a
	// hold would be dropped by an ordinary refresh.
	again := SignInMark(Token{RefreshToken: "R1", AccessToken: "A2", Account: "other@example.com"})
	if first != again {
		t.Errorf("one sign-in marked %q and then %q", first, again)
	}
	if other := SignInMark(Token{RefreshToken: "R2"}); other == first {
		t.Errorf("two sign-ins share the mark %q", other)
	}
}

// A token with no refresh token marks nothing at all, rather than marking the
// prefix alone. An empty mark is never equal to a real one, so a hold that
// remembers nothing is a hold nothing releases.
func TestATokenWithNoRefreshTokenHasNoMark(t *testing.T) {
	if got := SignInMark(Token{}); got != "" {
		t.Errorf("a token with no refresh token marked %q", got)
	}
	if got := SignInMark(Token{AccessToken: "AT"}); got != "" {
		t.Errorf("a token with no refresh token marked %q", got)
	}
}
