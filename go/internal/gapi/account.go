// This file is the one read that names no file: who the token signs in as.
// Every other read in this package is about a document.

package gapi

import (
	"context"
	"errors"
)

// accountURL is the whole of that read, spelled exactly as guard.isAccountRead
// reads it: one path, one field mask, no other parameter. The guard spells the
// same two literals on its own side, so a change here that the guard has not
// agreed to is a refusal rather than a wider read.
//
// The mask is the narrowest answer that names an account: the address, and the
// name a person recognises. Drive's about read returns the storage quota, the
// import formats and the rest of what the installation can do when nobody says
// otherwise, and none of it is anybody's business here.
const accountURL = "https://www.googleapis.com/drive/v3/about?fields=user(emailAddress,displayName)"

// Account is who a token signs in as. It is the answer gdoc mcp's login tool
// reads out, so a person in a chat sees which Google account a write would be
// made by, and the account `gdoc auth status` names, so a person at a prompt
// reads the same fact.
type Account struct {
	Email string
	Name  string
}

// Account reads it. The request needs guard.AllowAccountRead, which only
// accountOf in cmd/gdoc grants, for the login tool of gdoc mcp and for
// `gdoc auth status` and nothing else, so every other caller in this binary is
// refused by the guard before anything leaves:
// TestTheAccountReadWithoutTheGrantSendsNothing.
//
// An answer carrying no address is an error rather than an empty Account. The
// read exists so that a swapped account is seen, and an account with no name to
// read out would be reported as though it had been seen:
// TestAnAnswerWithNoUserIsNamed.
func (s *Session) Account(ctx context.Context) (Account, error) {
	var answer struct {
		User struct {
			EmailAddress string `json:"emailAddress"`
			DisplayName  string `json:"displayName"`
		} `json:"user"`
	}
	if err := s.GetJSON(ctx, accountURL, &answer); err != nil {
		return Account{}, err
	}
	if answer.User.EmailAddress == "" {
		return Account{}, errors.New("the account read came back with no address in it, so which Google account this is cannot be said")
	}
	return Account{Email: answer.User.EmailAddress, Name: answer.User.DisplayName}, nil
}
