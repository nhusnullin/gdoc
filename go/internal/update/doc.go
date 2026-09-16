// Package update is the arithmetic of `gdoc update`: what a tag says, which
// release of which channel a machine would install, what a run does about the
// difference, and the replacement itself. It holds no wire and no state.
//
// Everything but the replacement is a function over values. The command in
// cmd/gdoc opens the policy, fetches through internal/gapi and carries the
// decision out; this package is handed what came back and answers. That is
// what makes the policy table in the M9 plan a test over literals rather than
// a test over a network, and it is what leaves Apply with a temp directory as
// its whole world.
//
// # Three integers, and not semver
//
// A version is v2.1.3, with an optional pre-release word after a dash. There
// is no build metadata, no range syntax and no comparison of dot-separated
// pre-release fields, because gdoc compares two tags and nothing else. The
// module list stays at three, and a page of code is cheaper than a fourth.
// TestAVersionIsThreeNumbersAfterAV and
// TestAVersionThatIsNotThreeNumbersIsRefusedByName pin what parses, and
// TestCompareOrdersTheVersionsThePolicyReadsAbout pins the order.
//
// A tag nothing can parse is skipped rather than refused, because it comes
// from somebody else's listing: a moving tag on the releases page is GitHub's
// business, and it is not a version to compare against a person's binary.
// TestChooseSkipsADraftAndATagThatIsNotAVersion.
//
// # The channel is the patch number
//
// `make tag` cuts x.y.0 and the nightly cuts x.y.(z+1), so the patch number is
// the whole of the difference between the two channels. GitHub's own
// prerelease flag decides nothing here: a nightly is an ordinary release of
// this repository, and a flag on the release page is a second place for the
// two to disagree. A draft is skipped, because its assets are not public.
// TestChoosePicksTheHighestOfTheChannelForThePlatform.
//
// The highest release of the channel is the only candidate. A latest release
// carrying no zip for this platform is a refusal naming the platform, never a
// quiet fall back to the release before it: an update that installs something
// other than the newest is worse than one that says what is missing. The same
// holds for the checksum file, because a release nothing can be verified
// against is a release nothing may be replaced from.
// TestChooseRefusesWhenThePlatformHasNoAssetInTheLatestRelease and
// TestChooseRefusesAChecksumFileThatIsMissing.
//
// # Nothing goes down, and nothing crosses a major unasked
//
// Decide is the policy table, row for row, in TestThePolicyTable. A target at
// or below what is installed is up to date whatever was typed, a target across
// a major boundary is named and not taken without --major, and --check turns
// whatever was decided into an answer with nothing written.
// TestOnlyAnUpdatedDecisionInstalls is the one field the command reads before
// it downloads anything.
//
// The decision says what to do and never why it matters. Whether a colleague
// should take a nightly, and whether a major is worth the afternoon, is for
// the person reading the object and the skills beside them.
//
// # Nothing moves before it is verified
//
// Apply checks the zip against its line of the release's checksum file first,
// and only then touches the disk. A download cut short, a byte changed on the
// way and an asset the checksum file says nothing about are all one refusal
// with the old binary exactly where it was.
// TestVerifyRefusesByNameWhatDoesNotMatch and
// TestAWrongChecksumReplacesNothing.
//
// Then three renames: the new binary is written beside the old as <path>.new,
// the old is renamed to <path>.previous, and the new is renamed into place. A
// running executable cannot be written through, so a rename is the only way a
// binary replaces itself, and every failure after the first rename puts the
// old one back. Sum is hashed off the file at its final path rather than off
// the bytes that were about to be written, because what a person runs
// tomorrow is the file, not the download.
// TestTheReplaceSequenceLeavesTheNewBinaryAndKeepsTheOld.
//
// gdoc never runs the binary it just installed. Nothing under go/ runs an
// external program, so the proof that the update worked is the next envelope
// a person sees, and Sum is what this run can honestly say about it.
//
// Rollback is a swap rather than a move: what was installed becomes
// <path>.previous and the earlier binary comes back, so a rollback taken by
// mistake is one more rollback away from where it started.
// TestARollbackRunTwiceIsWhereItStarted.
//
// Windows is the same three renames, and os.Rename replaces the file it lands
// on there as it does here. It is unmeasured until the Windows checklist in
// the M9 plan runs, which is the first time gdoc replaces itself on Windows.
package update
