// Package boundary holds the tests that keep the wire in one room, keep every
// external program out of the tree, keep the dependency list to three modules,
// and keep the documentation the shape this repository agreed on.
//
// Nothing here is production code. The package has no declarations at all
// outside its test files, and that is the point: every rule below is a property
// of the whole tree, so it belongs to no single package and has to be asked
// from outside all of them. A rule nobody measures is a rule that decays
// quietly.
//
// This comment holds why each rule refuses what it refuses, with the test that
// pins it. What the rules are is in the code beside them.
//
// # Two allowlists over the wire, and the difference is the point
//
// Naming net/http and dialing with it are not the same thing, so there are two
// lists rather than one.
//
// The import allowlist says who may name the type. Four rooms: internal/guard,
// internal/auth, internal/auth/loopback and internal/gapi. internal/auth is on
// it because Refresh and Login take the guard's client as a parameter.
// internal/gapi is on it because it builds the *http.Request every read goes
// out as and sets the bearer on it, while the *http.Client it sends them on is
// a parameter. Keeping request building in one room is what stops the bearer,
// the Accept header and the refresh rule from being written three slightly
// different ways in three reader packages. It is in the import allowlist and
// not the builder one, and that difference is the whole reason for having two
// lists. TestNetHTTPStaysInItsRooms is the pin.
//
// The builder allowlist says who may construct an outbound client or reach a
// package-level dialer such as http.Get. That is internal/guard alone, because
// the guard builds its client out of a policy, so the first request in the
// program's history has already been judged. Serving is not building:
// internal/auth/loopback runs an http.Server, which answers a request somebody
// else made, so it stays out of this set. TestOnlyTheGuardBuildsTheWire is the
// pin, and TestScannerTellsNamingFromBuilding is what states the serving case
// rather than leaving it to luck.
//
// # Both lists fail in both directions
//
// They fail when an import or a builder spreads, and they fail when an
// allowlisted room stops holding what it was listed for. A list naming a room
// that no longer owns the wire has stopped describing the tree, and it is a
// door standing open for no reason.
//
// The disappearance half reads production files only. A _test.go that fakes the
// wire must never stand in for the room that owns the wire, so httpImporters
// returns the two sets apart and TestScannerFindsAStrayImport asks for both: a
// package whose only net/http import is in a test counts as a stray and does
// not count as a room.
//
// # The builder scanner knows eight ways to make a wire
//
// A composite literal, new(http.Client), a zero-value declaration such as
// var c http.Client, the package-level dialers, a type declaration that renames
// the wire (type C = http.Client, or the same without the equals sign), a
// struct holding one by value, a container holding one by value (make([]http.Client, 1)
// and every shape holdsWire walks), and a function handing one back by value
// (func Build() (c http.Client) { return }). holdsWire recurses, so it also
// sees a wire that is only a generic type argument. A struct field counts
// whether it is embedded or named: struct{ C http.Client } is the same
// zero-value client as struct{ http.Client }, reached through one extra word.
//
// That is not thoroughness for its own sake. A scanner that assumes the import
// name is always http and looks only for composite literals is walked around by
// import nh "net/http", by import . "net/http", by new(...), by a zero-value
// declaration, by type C = http.Client; var _ = &C{}, by
// type T struct{ http.Client }, by a slice, and by a named return. Every one of
// those builds a wire outside the guard while passing both checks.
// TestScannerTellsNamingFromBuilding carries a case for each, and each case was
// watched failing against the scanner that missed it.
//
// The two type shapes are flagged on the declaration, not on the values built
// from it. Following an alias would mean resolving names across a package, and
// a package outside the guard has no reason to give the wire a second name.
// Embedding a pointer is not flagged: that is holding a client somebody else
// made, the same as taking one as a parameter.
//
// # No external programs at all
//
// No os/exec anywhere under go/, and nothing may add one. What makes the rule
// possible is that auth login prints the authorization URL instead of opening a
// browser, and opening a browser is the one thing a CLI usually shells out for.
// A binary that runs nothing is a binary that is the whole dependency.
// syscall/js is refused beside os/exec, since it reaches a host that can run
// anything. TestNothingRunsAnExternalProgram is the pin.
//
// # Three modules, named in allowedModules, and the rest refused
//
// allowedModules names what go.mod may require, each against its reason. Every
// other require line is refused, and so is every other module in go.sum,
// because the sum file is what proves something was really fetched.
//
// The three are github.com/goccy/go-yaml for the gdoc: front matter and
// house.yaml, github.com/beevik/etree because encoding/xml rewrites namespace
// prefixes and drops the attribute order Word reads, so a part that went
// through it comes back as a document Word repairs, and github.com/yuin/goldmark
// for the markdown the generator parses. That is all SPEC.md agreed. A
// milestone that first needs a module adds its path here and nothing else: it
// does not delete the test, and it does not widen it to "whatever go.mod says".
// A fourth module needs its reason in SPEC.md before its line in the map.
// TestNoThirdPartyDependencies is the pin.
//
// TestAllowedModulesAreReallyRequired is the other direction, the same
// disappearance rule the wire lists hold: a path in the map that go.mod no
// longer requires fails too, because an allowlist naming something that is not
// there stops describing the tree.
//
// Once a module is both listed and required, a test that reads only the real
// files passes whether the refusal still works or not, so
// TestAllowedModulesStillRefusesAnUnlistedPath judges scratch text instead, and
// TestRequiredModulesReadsBothSpellings asks the same of the go.mod parser,
// which has to read a single require line and a require block alike.
//
// # Four guards over the docs' shape
//
// docs_test.go asks the same kind of question about the documentation, and it
// is here for the same reason the wire checks are: a prose rule nobody measures
// is one that decays quietly. That is how CLAUDE.md grew to thousands of lines
// of per-package essay no reader asked for and every session paid for.
//
// TestCLAUDEmdIsUnderTheCeiling is a ceiling on the file every session loads.
// The number is claudeCeiling, and raising it is a decision somebody explains
// in the commit message, not a number somebody nudges to make a test green.
//
// TestEveryPackageHasExactlyOnePackageComment counts them: one package comment
// per package, in a file that is not a test. A package comment on a test file
// is one go doc never prints, so the reader who went looking for the reasons
// finds nothing and writes them down a third time. Two comments are the quieter
// failure, since go/doc joins them and neither author sees the pair. The known
// map is empty today, and it works in both directions the way drift.Known does:
// a listed package that already holds the rule fails too, so the task that
// fixes a package deletes its row rather than leaving a name that stopped
// meaning anything.
//
// TestOnlyThePackageCommentReachesGoDoc is the other half of that rule, and it
// is the half a reader notices first. A comment block sitting directly above
// the package clause is that file's Doc to go/parser, and go/doc concatenates
// every file's Doc into the package's documentation, in filename order, so a
// paragraph saying what one file is for does not stay in that file. The rule is
// not that a file may carry no paragraph: it is that a blank line separates the
// paragraph from the package clause, which leaves it an ordinary comment in the
// file and leaves go doc printing the package comment alone. The counting guard
// cannot catch this, because pkgComment deliberately does not count a file
// paragraph, and counting them there would refuse the convention outright.
//
// TestTheTaskMapNamesFilesThatExist reads the task map in CLAUDE.md and stats
// every path in it. A missing heading is a failure rather than an empty pass: a
// map that is not there is the same silence as a map full of dead rows, and the
// map is the one thing in that file a reader uses to decide where to go next.
//
// # A make target that writes into bin/ makes bin/ first
//
// bin/ is ignored and nothing tracks it, so a fresh clone does not have one. A
// target that writes a binary there without creating the directory fails before
// it produces anything, and it fails only on the machine that has never built
// before, which is the machine somebody is trying gdoc on for the first time.
// TestEveryTargetThatWritesIntoBinMakesIt is the pin.
package boundary
