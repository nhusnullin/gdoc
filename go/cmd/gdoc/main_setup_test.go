// The two things this package's tests set up before any of them runs: nobody
// reaches the network to ask who the token signs in as, and nobody opens the
// app of the machine the suite is on.
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
	"path/filepath"
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
	accountOf = func(context.Context) (gapi.Account, []string, error) {
		return gapi.Account{Email: testAccountEmail, Name: testAccountName}, nil, nil
	}
	// The app gdoc would open is a path under a temporary directory that is
	// not there, so openPresenceApp's first question about the bundle answers
	// no in every test that does not point it at a bundle of its own. Without
	// this, a run under the presence setting on the machine this suite is
	// developed on would open the real menu bar app: appAt and the opener's
	// own table test are the two places that say otherwise.
	presenceBundle = func() (string, error) {
		return filepath.Join(os.TempDir(), "gdoc-tests-no-app-here", "GdocPresence.app"), nil
	}
	os.Exit(m.Run())
}
