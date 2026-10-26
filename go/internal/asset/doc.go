// Package asset is one file in Drive, read as bytes, held to what the caller
// says of it, and cached beside the token. It knows nothing about logos: it is
// handed an id and a statement of mime, so the second asset gdoc
// ever needs is the same call with another id.
//
// It exists because of the M22 decision that the firm's logo is the firm's
// file in the firm's Drive rather than bytes in a public tree. The whole of
// what makes that affordable is here: a stamp, so a build asks Drive once a
// day rather than once a run, and a cache, so a build that cannot reach Drive
// still carries the mark. docs/v2/DECISIONS.md holds the decision.
//
// # It is handed its reach and builds nothing
//
// Fetch takes a Drive, which is one method: the bytes of one file by id,
// within a ceiling. internal/gapi's session is one, because the request, the
// bearer and the refresh rule belong there, and internal/guard owns the client
// underneath it. A test hands its own. So this room opens nothing, and the
// grant that admits the read is the caller's to make: guard.AllowAsset names
// one id for one run, admits GET with alt=media and supportsAllDrives=true,
// and admits nothing else on it.
//
// A nil Drive is a run with no sign-in. That is not an error here, because a
// cache written yesterday is still an answer: TestNoTokenAndACacheUsesTheCache.
//
// # It holds facts and decides nothing
//
// Bytes carries the file, where it came from, when it was fetched, how old
// that is and why the cache was used. Nothing in it says the run should go on
// or stop, and nothing is a cached verdict. Whether a person hears about the
// age is the caller's decision, in cmd/gdoc's envelope and in chat.
//
// # The order, which is the whole rule
//
// A stamp written within Interval, about bytes that are still there and still
// the file the caller states, is the answer, and the wire stays shut:
// TestAFreshStampMakesNoRequest. Anything else asks Drive exactly once, and
// every way a stamp can fail to say "asked today" is one of them:
// TestAStaleStampFetchesOnce holds the eight, from no stamp at all to a stamp
// whose hash is another file's.
//
// A read that worked is checked, cached and returned as FromDrive. A read that
// was refused, timed out or never left falls back to the cache, with the
// status or the error as Reason: TestARefusedReadUsesTheCacheAndNamesTheReason.
// No reach at all is the same case with the sign-in as the reason.
//
// Nothing usable in the cache and no answer from Drive is a refusal naming
// what happened, and the sign-in when that is what is missing:
// TestNoTokenAndNoCacheRefusesNamingTheLogin. This room never returns bytes it
// did not check and never returns none.
//
// # A wrong file refuses even when the cache is good
//
// A file in Drive that is not a PNG within the side ceiling stops the run, and the
// cache is left exactly as it was: TestABadFileRefusesEvenWithAGoodCache. The
// firm's file is wrong and a refusal is the only message that reaches whoever
// can fix it. Falling back there would hide a broken file for as long as a
// cache survives. The same check runs over the cached bytes, so a cache
// somebody corrupted is no cache: a file gdoc half understands reaches a
// document by no route. The check reads the header first and judges both sides
// against MaxSide, because Go's decoder allocates the whole picture from what
// the header says: TestAHugeStatedSizeIsRefusedWithoutDecoding,
// TestASideOverTheCeilingIsRefusedBeforeDecoding and TestTheSideCeilingIs4096
// hold that. It then
// decodes the picture whole, because a header states a size a file cut off
// after it does not carry: TestAHalfPNGIsRefusedByBothRoutes holds both
// routes.
//
// Want is the caller's mime statement, and image/png is the only mime this room
// checks, because it is the only one it can decode. A statement naming
// anything else is refused rather than accepted unread.
//
// # The two files, and how they are written
//
// config.AssetPath and config.AssetStampPath: assets/<id> and assets/<id>.json
// in the config directory, beside the token, which is where this machine's
// per-user state lives. The id is one path segment and is checked as one, so
// an id out of a house file somebody wrote by hand never becomes a path:
// config's TestAnAssetPathRefusesAnIdThatIsNotOne and this package's
// TestAnIdThatIsNotOneIsRefusedBeforeAnythingIsRead.
//
// Both files go through internal/atomicfile at 0600, in a directory this
// package makes on first write: TestTheCacheIsWrittenAtomically. The bytes are
// only replaced when they changed, while the stamp moves either way, because
// the stamp records that the question was asked:
// TestUnchangedBytesLeaveTheCacheFileAlone. A cache that cannot be written
// still builds, with the failure named in CacheError:
// TestACacheWriteFailureStillBuilds.
//
// The stamp is internal/lastcheck's shape with its own two keys, fetched_at
// and sha256, one line in UTC to the second:
// TestTheStampIsOneLineAPersonCanRead. It is read strictly and every refusal
// is stale rather than fatal, for lastcheck's reasons: an unknown key is an
// older binary reading what a newer one wrote, and there is always an answer
// available, which is to ask Drive again. A stamp that decoded and is merely
// old keeps its time, so a fetch that then fails can report the age of the
// bytes it fell back on.
//
// Interval is a day and TestTheIntervalIsADay states the literal. MaxBytes is
// a megabyte, far above the five kilobytes a logo is, because it is there to
// stop an answer nothing bounds: TestTheReadIsBounded. MaxSide is 4096, which
// bounds decoded allocation before a whole PNG is read:
// TestTheSideCeilingIs4096.
package asset
