// The panel `gdoc auth status` draws on stderr: what the report says, in
// words, for the person who typed the command.
//
// It sits beside helpscreen.go and loginscreen.go for the same reason: this is
// the room that knows what a command draws, and internal/tty and
// internal/panel are the rooms that know a colour and a box. So no colour
// name, no box-drawing character and no escape code is written here.
//
// Unlike a help screen this one replaces nothing. The object still goes to
// stdout, because for this command the object is the answer and the panel is
// the same facts for a reader who is not a program. So nothing here can drop a
// field, and every row is read out of the data that was printed.

package main

import (
	"io"
	"strings"

	"gdoc/internal/panel"
	"gdoc/internal/tty"
)

// The three states a reader can be in, as the panel says them. They come from
// the report's own fields and no minutes come with them: when a token expires
// is a field of the object, and the question a person asks at a prompt is
// whether they are signed in at all.
const (
	stateSignedIn      = "signed in"
	stateSignedOut     = "signed out"
	stateScopesMissing = "scopes missing"
)

// statusColumn is the cell the keys and the values part at, which is where the
// pictures draw it: panels-round-two.html, statusScreen. The keys are short, so
// the column is the one thing in this panel that is not measured from them.
const statusColumn = 16

// writeStatusScreen draws the panel where there is one to draw: a terminal
// wide enough for a box. A pipe, a file and a narrow window read nothing here,
// as they always did.
func writeStatusScreen(errOut io.Writer, d statusData) {
	if d.StatusReport == nil {
		return
	}
	p, style, ok := screenAt(errOut, false)
	if !ok {
		return
	}
	writeLines(errOut, statusScreen(p.WithColumn(statusColumn), style, d))
}

// statusScreen is the panel: the state, the account where one was read, the
// file the token is in, the scopes it carries and the ones it does not, and
// the two facts that say which credential this is.
func statusScreen(p panel.Panel, style tty.Style, d statusData) []string {
	rows := pairRows(p, style.Key("status"), stateOf(style, d))
	if d.Account != "" {
		rows = append(rows, pairRows(p, style.Key("account"), accountWords(d))...)
	}
	rows = append(rows, pairRows(p, style.Key("file"), d.TokenPath)...)
	if len(d.Scopes) > 0 {
		rows = append(rows, pairRows(p, style.Key("scopes"), scopeNames(d.Scopes))...)
	}
	if len(d.MissingScopes) > 0 {
		rows = append(rows, pairRows(p, style.Key("missing"), style.Warn(scopeNames(d.MissingScopes)))...)
	}
	rows = append(rows, pairRows(p, style.Key("client"), d.ClientSource)...)
	rows = append(rows, pairRows(p, style.Key("mode"), d.AuthMode)...)
	return p.Box("auth status", gdocLabel(), rows)
}

// accountWords is the account row: the person's name beside the address when
// Google said who it is, and the address alone when it did not.
// TestThePanelNamesThePersonBesideTheAddress.
func accountWords(d statusData) string {
	if d.AccountName == "" {
		return d.Account
	}
	return d.AccountName + " · " + d.Account
}

// stateOf is the first row: a mark and the words for it. The marks are the
// login screen's own tick and cross, and the warning mark the help screen
// opens a warning row with, so one terminal says one thing three ways and not
// three things.
func stateOf(style tty.Style, d statusData) string {
	switch {
	case !d.TokenPresent:
		return style.Fail(failedMark) + " " + stateSignedOut
	case len(d.MissingScopes) > 0:
		return style.Warn(warnMark) + " " + stateScopesMissing
	default:
		return style.OK(doneMark) + " " + stateSignedIn
	}
}

// scopeNames is what a scope is called, which is its last path element: the
// whole URL is four rows of a panel and says nothing the last word does not.
// The object carries the URLs, and a reader comparing the two reads the same
// order.
func scopeNames(scopes []string) string {
	names := make([]string, 0, len(scopes))
	for _, s := range scopes {
		names = append(names, s[strings.LastIndex(s, "/")+1:])
	}
	return strings.Join(names, ", ")
}
