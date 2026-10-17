// This file is the sign-in mark: one string that tells one sign-in from
// another and is no credential. The rules it holds are in the package comment
// in doc.go.

package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

// signInMarkDomain is what the hash is taken over before the token itself.
//
// It is there so this digest is this question's own: a hash of the refresh
// token with no prefix would equal a hash somebody else took of the same
// string for another purpose, and two answers that happen to match would then
// be read as one. The v1 is in it because a mark is compared only against
// another mark of the same running process, so the day the input changes the
// prefix changes with it and nothing has to be migrated.
const signInMarkDomain = "gdoc sign-in mark v1\x00"

// SignInMark is which sign-in this token is, as a string that carries nothing
// of it: the hex SHA-256 of a fixed prefix and the refresh token.
//
// The refresh token is what it reads, because that is the one field that
// changes when somebody signs in as another account, and the one field an
// ordinary access-token refresh leaves alone. internal/gapi already treats a
// changed refresh token as a newer sign-in, so a hold borrowing this reading
// cannot disagree with a session about which login is in the file.
//
// A token with no refresh token has no mark at all, and an empty mark is never
// equal to a real one. So a hold that remembers no sign-in is a hold nothing
// releases, which is the safe way round: TestATokenWithNoRefreshTokenHasNoMark.
//
// No account and no email is compared anywhere. A mark says the sign-in is a
// different one, and never whose it is: TestTheSignInMarkCarriesNoWordOfTheToken.
func SignInMark(t Token) string {
	if t.RefreshToken == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(signInMarkDomain + t.RefreshToken))
	return hex.EncodeToString(sum[:])
}
