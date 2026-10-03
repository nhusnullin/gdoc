// The one thing this package's tests set up before any of them runs: nobody
// reaches the network to ask who the token signs in as.
//
// `gdoc auth status` reads the account live, so every test that runs it would
// otherwise build a real client and make a real request from whatever machine
// the suite is on. The fake answers the same two fields the real read does, so
// a test that cares about them says so with stubAccount and every other test
// is simply offline.

package main

import (
	"context"
	"os"
	"testing"

	"gdoc/internal/gapi"
)

// The account every test reads unless it stands in for its own. The domain is
// example.com, which is nobody's.
const (
	testAccountEmail = "name@example.com"
	testAccountName  = "Example Person"
)

func TestMain(m *testing.M) {
	accountOf = func(context.Context) (gapi.Account, error) {
		return gapi.Account{Email: testAccountEmail, Name: testAccountName}, nil
	}
	os.Exit(m.Run())
}
