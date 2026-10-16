// Package update is the arithmetic of `gdoc update`: what a tag says, which
// release of which channel a machine would install, what a run does about the
// difference, and the replacement itself. It holds no wire and no state.
//
// It still holds none after the daily check landed on 2026-09-18: what
// remembers when gdoc last asked GitHub, and what it heard, is the stamp file
// in internal/lastcheck, and this package is handed those versions the same
// way the command hands it a listing.
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
// A version is v2.1.3 and nothing else: no pre-release word, no build
// metadata, no range syntax, because gdoc compares two tags and nothing else.
// A dash is refused by name, because the nightly is the pre-release channel
// and an rc would be a third channel for one use (DECISIONS.md, 2026-09-17).
// The module list stays at three, and a page of code is cheaper than a fourth.
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
// TestTheReplaceSequenceLeavesTheNewBinaryAndKeepsTheOld, and
// TestAnUnreadableReadBackPutsTheOldBinaryBack for the read-back that fails.
//
// Where there was no old one to put back, the failing read-back removes what
// it could not verify instead. There is nothing to restore, and leaving the
// file would leave an unverified binary for the next run to execute.
// TestAFailedReadBackWithNothingToRestoreLeavesNoBinary.
//
// gdoc never runs the binary it just installed. The one program anything under
// go/ runs is the opener that hands the Claude Desktop extension to Claude
// Desktop, and it is in cmd/gdoc rather than here, so the proof that the update
// worked is the next envelope a person sees, and Sum is what this run can
// honestly say about it.
//
// # One more file out of the same zip
//
// FileFrom is what a caller may read out of a release without writing
// anything: one named file, out of bytes that have already matched the
// published checksum. `gdoc update --desktop` reads the Claude Desktop
// manifest template that way, so the template a person's Claude Desktop is
// handed came out of the same verified zip as the binary it names, and no
// second download decides anything. Nothing on disk moves.
// TestFileFromTakesOneFileOutOfTheVerifiedZip,
// TestFileFromRefusesAZipThatDoesNotMatchItsChecksum and
// TestFileFromRefusesAFileTheZipDoesNotHoldByName.
//
// Apply, FileFrom and sumLine are each about one named file, and until the
// presence landed that was the whole of what this package took out of a zip.
// ApplyBundle is the one route that takes a tree, and the section below is
// what that costs.
//
// Rollback is a swap rather than a move: what was installed becomes
// <path>.previous and the earlier binary comes back, so a rollback taken by
// mistake is one more rollback away from where it started.
// TestARollbackRunTwiceIsWhereItStarted.
//
// It is also the one place a failure after the swap moves nothing back, since
// undoing it would take back the rollback a person asked for. It says where
// the binaries ended up instead, and it removes nothing: after the swap the
// binary that was rolled back from is the only copy of that release on the
// machine. TestAFailedKeepLeavesTheRolledBackBinaryOnDisk. TODO(test):
// nothing drives a read-back failure after the swap.
//
// A rollback reports no installed version. The binary it put back is the one
// gdoc is not running, and reading its version would mean running it.
// TestRollbackPutsTheEarlierBinaryBack.
//
// # A folder out of the same zip, and the app it is
//
// ApplyBundle installs the macOS app bundle the darwin zips carry at
// presence/GdocPresence.app. It is a second function rather than a flag on
// Apply because almost nothing about a tree is the same as a file. A folder
// cannot go through internal/atomicfile, which replaces one file's contents.
// A zip holding a tree can hold a path that climbs out of it, a symlink, a
// second entry for the same file, a path that is a different file here and
// the same file on a Mac's case-insensitive disk, and a resource fork, none
// of which a single named file can be. And rename will not replace a folder,
// so the bundle kept from the run before has to be cleared out of the way
// rather than written over.
//
// The order is Apply's order. The zip is verified first, so a zip that fails
// its checksum extracts nothing:
// TestABundleIsTakenOutOfTheVerifiedZipOnly. Then every entry is judged
// before any of them is written. A name no entry of a release ever carries
// is refused wherever it sits in the archive: an absolute path, a .. or .
// segment, a backslash, the __MACOSX folder a Mac packs beside what it
// zipped, and a resource fork. Everything else is judged only under the
// bundle path, and an entry elsewhere in the zip is simply not taken,
// because the same archive holds the binary, the installer and the skills.
// TestOnlyEntriesUnderTheBundlePathAreTaken and
// TestAPathOutsideTheBundleASymlinkAndAnAppleDoubleEntryAreRefusedByName.
//
// Two ceilings, 2,000 entries and 64 MiB expanded, are both far above a
// plist and one universal executable and far below what a zip bomb needs.
// They are added up off the entries' own headers before anything is written,
// and each entry is then read through a limit of the size its header
// declared, so an entry that unpacks past what it promised cannot walk
// through the ceiling that was added up from it.
// TestTooManyEntriesOrTooManyBytesAreRefusedByName.
//
// The last judgement is the bundle's own Info.plist, read strictly with
// encoding/xml: it has to name com.altery.gdoc.presence and carry the
// release's own version, which is the asset name's tag without the v. A zip
// and a stamp that did not come out of the same build are refused by name.
// Reading the plist with encoding/xml and nothing else is also what keeps it
// in the XML format, because a plist converted to the binary one would stop
// reading here. TestAnotherIdentifierOrAnotherVersionIsRefusedByName and
// TestOnlyAnXMLPlistReads, with TestTheReleasesTagIsReadOutOfTheAssetName
// for where the version compared against comes from.
//
// What is already at the destination is judged too, and nothing has moved
// when that refusal is raised. A symlink is refused because the rename would
// replace the link and leave whatever it pointed at; a file because a bundle
// is a folder; a folder whose plist does not read, or names somebody else,
// because not knowing never resolves to overwrite. A person who put
// something else there removes it, and the next run installs.
// TestADestinationThatIsASymlinkAFileOrAnotherAppMovesNothing. A missing
// ~/Applications is made rather than refused:
// TestAMissingApplicationsFolderIsMade.
//
// Then the tree is written under <dest>.new, with a stale stage cleared
// first, read back off the disk and summed, and swapped in by rename. A file
// the zip marked executable is left at 0755 and everything else at 0644,
// whatever the person's umask would have taken off, and every folder at
// 0755. TestModesComeFromTheZip and TestAStaleStageIsRemovedFirst.
//
// TreeSum is what a tree has instead of Sum: one line per file, the path
// under the bundle, a NUL, that file's SHA-256, sorted by path, and the
// SHA-256 of all of it. The NUL is the separator because it is the one byte
// a path cannot hold. TestTheStageIsReadBackAndSummed. TODO(test): nothing
// drives the comparison between the tree that was written and the tree read
// back, because within one call nothing can make the two differ.
//
// The swap is two renames and the clearing before them, and the second
// rename failing undoes the first. TestTheSwapKeepsThePreviousBundle and
// TestAFailedSwapPutsTheOldBundleBack, which calls the swap with nothing
// staged, because ApplyBundle cannot be made to hit it on purpose.
//
// RollbackBundle is the bundle's involution, as Rollback is the binary's:
// TestRollbackOfTheBundleRunTwiceIsWhereItStarted and
// TestRollbackOfTheBundleWithNoPreviousIsRefusedByName. It differs from
// Rollback in one way. A rollback reports no installed binary version,
// because reading it would mean running the binary; a bundle's version is a
// file, so the rollback reads it and says what went back.
//
// Nothing here undoes the binary. The app and the binary are installed one
// after the other and judged apart, so a machine can sit with the two at
// different versions, and the app is the one that says so.
//
// Windows is the same three renames, and os.Rename replaces the file it lands
// on there as it does here. It is unmeasured until the checklist in
// docs/backlog/windows-rollout-checklist.md runs, which is the first time gdoc
// replaces itself on Windows.
package update
