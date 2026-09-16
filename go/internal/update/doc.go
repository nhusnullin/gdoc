// Package update is the arithmetic of `gdoc update`: what a tag says, which
// release of which channel a machine would install, and what a run does about
// the difference. It holds no wire, no file and no state.
//
// Everything here is a function over values. The command in cmd/gdoc opens the
// policy, fetches through internal/gapi and carries the decision out; this
// package is handed what came back and answers. That is what makes the policy
// table in the M9 plan a test over literals rather than a test over a network.
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
package update
