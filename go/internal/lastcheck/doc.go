// Package lastcheck is one file: update-check.json in the config directory,
// holding when gdoc last asked GitHub what is published and what it heard.
//
// It exists because of the 2026-09-18 decision that help checks for a newer
// release by itself, once a day. The whole of what keeps that check cheap is
// this file: a run reads it, and only a run that finds it missing or older
// than Interval opens the wire at all. A failed check is written down too, so
// a machine that cannot reach GitHub pays the two second ceiling once a day
// and not once a run. docs/v2/DECISIONS.md holds the decision and what it
// reverses.
//
// # It holds facts and decides nothing
//
// The stamp says what was published when gdoc asked. Whether that is newer
// than this binary is internal/update's arithmetic, run fresh over the
// versions in the stamp every time, and never written here. Nothing in the
// file says "available", "behind" or "should", and nothing in it is a cached
// verdict: a stamp says what GitHub answered, not what to do about it.
//
// # The shape, and why it is read strictly
//
//	{"checked_at":"2026-09-18T07:12:03Z","latest_stable":"v2.3.0","latest_nightly":"v2.3.4"}
//
// One line, UTC to the second, so a person looking in ~/.config/gdoc-agent can
// read it. TestAWrittenStampReadsBackByteForByte pins those bytes as a
// literal, and TestAFailedCheckKeepsWhatItHeardAndNamesTheCause pins the line
// a failed check writes.
//
// The read is strict and every refusal names what was wrong: a key gdoc does
// not know, a second object behind the first, a file nothing can decode, a
// file with no checked_at in it. This is gdoc's own file, so an unknown key is
// an older binary reading what a newer one wrote, and reading half of it
// silently would hide that. Every refusal is stale rather than fatal, because
// there is always an answer available: ask GitHub again.
// TestMissingCorruptAndOldAreEachStaleAndNamed holds the four causes, and the
// twenty three hour file that is fresh beside them.
//
// # The interval, and what a stale stamp still carries
//
// Interval is a day. TestTheIntervalIsADay states the literal 24h and never
// reads the constant, as a house-style test does. A stamp that decoded and is
// merely old comes back with its versions, because the check that follows may
// fail and then keeps them; a stamp nothing could decode comes back empty,
// because there is nothing in it to keep.
//
// A stamp dated after the clock is stale too. Somebody's clock moved, or the
// file came from another machine, and asking again costs one request.
//
// # The write
//
// Through internal/atomicfile, at 0600 like the token beside it, into a
// directory this package makes when it is not there: the first run on a
// machine is exactly the run with something to record.
// TestAWriteLeavesNoTempFileBehind is the pin over the temp file,
// TestTheStampIsWrittenReadableByItsOwnerAlone over the mode,
// TestWriteMakesTheDirectoryItWritesInto over the missing directory, and
// TestAWriteThatCannotHappenSaysSo over the directory nothing may write into.
// The caller reports that failure as a warning and does not retry in the same
// run; cmd/gdoc's notice.go is where that is decided.
//
// # What is not here
//
// No wire. This package opens nothing and imports no client, so the one room
// that builds a request stays internal/gapi and the one room that judges it
// stays internal/guard. The path is config.LastCheckPath, beside the token's,
// so nothing spells the file name twice.
package lastcheck
