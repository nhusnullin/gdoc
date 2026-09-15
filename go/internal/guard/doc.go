// Package guard is the network policy, and the only place a client is built.
//
// Principle 3: the client reaches only the files it was given, and every id
// carries a write level. policy.go is pure judgment, a function of the method,
// the URL and the body. transport.go carries requests through it and is the
// only room that touches the wire.
//
// This comment holds why the package refuses what it refuses, with the test
// that pins each rule. What the rules are is in the code beside them.
//
// # The guard exists before any client
//
// NewClient is the only place an *http.Client is made, and it is made from a
// *Policy. So the first request in the program's history has already been
// judged: there is no window in which a client exists and no policy does.
// A guard fitted around a client somebody else built cannot make that claim,
// and has to prove instead that nobody builds a second one.
//
// The builder allowlist in go/boundary/boundary_test.go is what keeps the
// claim true as the tree grows: this package is the only room permitted to
// construct an outbound client or reach a package-level dialer.
//
// # The levels are names, not a ladder
//
// A write level lives in the policy, never at the call site. LevelSuggest is
// what a handed-in id gets: read, comment, suggest, and never a direct edit.
// LevelFull is what a create returned. LevelInPlace is one handed-in id
// raised for one run, so a restyle can style that document where it stands.
//
// Every comparison is ==, never >=. That is why LevelInPlace does not inherit
// the Drive file PATCH that LevelFull carries: a restyle cannot trash or
// rename the document it is styling. Renumbering the constants must not change
// what any of them may do. TestTheInPlaceGrantDoesNotReachTheDriveFile is the
// pin, and TestALevelNamesItselfInWords covers the refusal message.
//
// A call site cannot widen its own reach by phrasing a request differently,
// because the policy reads the body as well as the method and the URL. A
// batchUpdate on a handed-in document is refused inside the process unless the
// body says SUGGEST, or the document is the one id this run was granted.
// TestAHandedInDocumentIsNeverDirectlyEditedWithoutTheGrant is the pin.
//
// # Two doors into the set, and four grants beside it
//
// Ids reach the reachable set two ways and no more. AllowFile is the id a
// command was pointed at. Learn is the id a create the guard itself carried
// came back with, read off that create's own answer. TestCreateTeachesThePolicy
// and TestFailedCreateTeachesNothing are the pins: a create that did not
// succeed teaches nothing.
//
// Four grants sit beside the set. Each names one object, opens one shape, and
// dies with the process. None of them is a level and none is a third door.
//
//   - AllowCreateIn names the one folder a create may target. A create naming
//     any other parent is refused, and naming the folder does not put the
//     folder itself in the set. TestAllowCreateInDoesNotAdmitTheFolder and
//     TestCreateOutsideTheNamedFolderIsRefused are the pins.
//   - AllowReject names one suggestion id. The guard refuses every batchUpdate
//     request kind whose name carries "suggestion", and this opens exactly one
//     shape through that wall: a rejectSuggestion spelled exactly, carrying
//     exactly {"suggestionId": <that id>} and nothing beside it. gdoc never
//     accepts, rejects or deletes anyone else's suggestion, and the permission
//     here is provenance: cmdWithdraw seeds the grant from the note's
//     proposals[], which is the only record of what gdoc itself wrote. Nail's
//     decision, 2026-09-07, DECISIONS.md. The pin is
//     TestAGrantedRejectSuggestionCarriesAndNothingElseInTheFamilyDoes.
//   - AllowCopy names the one file files.copy may duplicate into the folder
//     AllowCreateIn named. It is the grant with no production caller, and that
//     is agreed rather than overlooked: its only caller is
//     TestLiveRestylePreservesTenFeatures, which copies a document before it
//     restyles the copy, so the ten-feature acceptance can run unattended.
//     Nail's decision, 2026-09-09, DECISIONS.md. driveCopyParams is its query
//     allowlist. Deleting either reopens that decision. The pins are in
//     copy_test.go, TestTheCopyGrantIsOneSource among them.
//   - AllowMarker names the one named range a createNamedRange may make, by
//     name and by span, at LevelInPlace alone. A second call replaces the
//     first, because a caller naming two ranges has made a mistake the guard
//     must not turn into two markers. A grant the policy cannot read opens
//     nothing and takes back the grant standing before it: a caller that has
//     just shown it cannot compute a range must not be left with an earlier
//     one live. TestASecondAllowMarkerReplacesTheFirst and
//     TestARefusedSecondMarkerGrantTakesTheFirstBack are the pins.
//
// GrantInPlace is the fifth of that shape and the only door to LevelInPlace.
// It upgrades an id already in files and admits nothing new, so it is not a
// third door into the set. AllowFile refuses to hand that level out at all,
// because taking any level there was the side door around the invariant.
// TestGrantInPlaceNeverAdmitsAnUnknownID and TestAllowFileCannotOpenTheInPlaceDoor
// are the pins.
//
// A create whose response carries no readable id is recorded on the policy and
// readable through Warnings. Silence there turns into "file was not given to
// this command" on the next request, which names the wrong problem.
// TestACreateWithNoReadableIDIsRecorded is the pin.
//
// # Read this before trusting the SUGGEST bar
//
// What keeps a handed-in document read-and-suggest only is
// writeControl.writeMode == "SUGGEST" in the request body, which is a field
// the client itself supplies. It is absent from the public Docs discovery
// document, and one morning the same call returned 200 and silently made a
// direct edit. Measured again 2026-09-07 on a throwaway document: a SUGGEST
// insert came back 200 and the read-back carried a suggestion id, so the
// project is enrolled today. BLOCKED-BY-API.md "Gated behind the Developer
// Preview" holds both measurements.
//
// Enrolled today is not a guarantee for tomorrow, and the earlier measurement
// is what says so. The field is a statement of intent rather than a guarantee,
// and what makes a write trustworthy is the capability probe before it and the
// read-back after it, neither of which lives here. Widening or narrowing what
// isSuggestMode permits is Nail's decision, not a refactor. That bar is the one
// LevelInPlace removes for a single id, which is why an allowlist of request
// kinds stands in its place there.
//
// # One body, read exactly, and read once
//
// isSuggestMode reads both keys exactly, and refuses a body where two keys fold
// to either name. Google's proto-JSON is case-sensitive, so WRITEMODE is not
// the field the server reads, while encoding/json matched it. The guard must
// never be broader than the server on the one field that permits a write.
// TestSuggestModeIsReadExactly is the pin.
//
// hasDuplicateKeys is the same rule one layer out. It walks the body as a token
// stream and refuses any object that names a key twice, folding case, and
// judgeRequests, checkCommentWrite and checkParentMetadata each call it first.
// encoding/json keeps the last copy of a repeated key and drops the rest, so a
// body carrying two requests lists, two parents lists or two action fields is
// judged on one copy and may be served on the other.
// TestARepeatedKeyIsRefused is the pin.
//
// The walk reads numbers as json.Number, and a body it cannot walk is refused.
// Both halves are one bug. json.Decoder.Token decodes a number into a float64,
// so a literal out of that range such as 1e999 ended the walk with an error,
// while the callers, which unmarshal into json.RawMessage and a []string, never
// parse the number and accept the same body. Swallowing that error carried a
// batchUpdate naming requests twice, with a deleteSuggestion in the copy the
// guard never read. With UseNumber the walk ends early only on a body that is
// not valid JSON, which every caller refuses on the line after, so failing
// closed there costs a message rather than a request.
// TestANumberTheWalkCannotParseDoesNotHideARepeat is the pin.
//
// commentWrites carries POST and nothing else: no PATCH and no DELETE, on a
// comment or on a reply. Nothing in a comment id says who wrote it, so the
// guard cannot tell gdoc's own comment from somebody else's, and no command
// needs either method. The milestone that needs one adds it back beside its
// caller. TestNoCommentPatchOrDelete is the pin.
//
// # The guard judges the request it actually sends
//
// X-HTTP-Method-Override and its two cousins are refused, and so is a _method
// query parameter, because Google's REST stack performs the overridden method:
// a GET the guard allowed would arrive as a DELETE it never saw.
// TestMethodOverrideIsRefused, TestMethodQueryOverrideIsRefused and
// TestAMethodOverrideInTwoCasingsIsRefused are the pins.
//
// A create whose parents the guard cannot read is refused. That no longer means
// every multipart upload: the transport reads the first MIME part and takes the
// parents out of it. What is still refused is a body whose parents the guard
// cannot reach, a resumable create among them. Failing closed is the right
// direction to be wrong in. TestACreateTheGuardCannotCheckIsRefused and
// TestCreateWithUnreadableParentsIsRefused are the pins.
//
// Three more shapes belong to that same rule, and each closes a way the request
// on the wire differed from the one that was judged.
//
//   - The upload parameter decides what the body is. With uploadType=media the
//     body IS the file's content, so {"parents":["FOLDER1"]} reads as bytes to
//     Drive and as metadata to the parent check: the file lands unparented and
//     the guard then learns its id at LevelFull. checkUploadShape permits the
//     shapes the parent check can read, multipart and an absent parameter, and
//     refuses the rest, resumable included. It reads upload_protocol too, which
//     is the same choice under Google's newer name, where raw is what media
//     was. A resumable create is refused for a second reason of its own: it is
//     two legs, the guard carries neither the PUT nor upload_id, so the shape
//     could never finish and a half-permitted route reads as a working one. A
//     request naming both parameters is refused rather than guessed at: the
//     guard would be reading the one Drive may not obey. Nothing may learn an
//     id from a create it could not verify. TestAMediaUploadIsRefusedBeforeTheWire,
//     TestResumableUploadIsRefused and TestUploadProtocolIsTheSameChoiceAsUploadType
//     are the pins.
//   - fields reaches the permission surface. A GET is judged on its path, and
//     fields=* or fields=permissions(...) returns exactly what refusing
//     /permissions was for. checkFields refuses a star, refuses permissions by
//     any spelling, and refuses an empty mask, because Google reads a mask with
//     nothing in it as every field. TestAReadMayNotAskForThePermissionSurface
//     and TestAnEmptyFieldMaskIsRefused are the pins.
//   - The base transport carries no proxy. http.DefaultTransport reads
//     HTTPS_PROXY, so a nil base would send an unjudged CONNECT to whatever
//     host the environment named, with the credential following it there.
//     baseTransport sets Proxy: nil for that reason. A proxy that is ever
//     wanted has to be judged, not inherited from the environment.
//     TestTheGuardDoesNotDialThroughAProxy is the pin.
//
// TestTheGuardJudgesTheBodyItSends and TestTheReplayIsTheBodyTheGuardJudged sit
// over the whole rule: what the transport writes is the bytes the policy read.
//
// # Two allowlists over one request, the query and the headers
//
// This is the guard's broadest rule, and it is easy to read past because
// neither half names a single attack.
//
// Google gives one capability several spellings. fields is also $fields and
// also the header X-Goog-FieldMask. uploadType has the sibling upload_protocol.
// key is also $key and the header X-Goog-Api-Key. A list of blocked spellings
// needs a patch each time somebody finds another one, and every miss is a live
// hole until then. So both halves are allowlists. They are wrong in the
// direction of a refusal the next milestone widens on purpose.
//
// The query. params.go holds one allowlist per call shape, not one across all
// of them, with noParams for a call that carries no query at all. One list for
// everything was wrong in both directions: it put paging on a metadata read and
// an export format on a comment listing, neither of which is a call Drive has,
// and it put alt on the bare files.get, where alt=media stops being a metadata
// read and hands back the file's bytes. checkQuery parses the raw query itself
// rather than through u.Query(), which drops a pair it cannot read and returns
// the rest, and it refuses a parameter given twice.
// TestEachDriveReadCarriesOnlyItsOwnParameters, TestAltMediaOnAPlainFileGetIsRefused
// and TestAnUnreadableQueryIsRefused are the pins, with
// TestTheCallsGdocMakesStillCarry in the other direction.
//
// The headers. allowedHeaders in transport.go names seven, in lower case, and
// every other header is refused. A Drive read whose query carries no fields can
// still ask for permissions(...) in X-Goog-FieldMask, so a guard that judges
// only the query judges half the request. referer is on the list because
// http.Client.Do builds a redirect hop itself and sets it, and leaving it out
// refused every redirect the policy allows. cookie is deliberately absent:
// NewClient sets no jar, so a Cookie is a second credential nobody decided
// about. A milestone needing another header adds it here on purpose.
// TestAnUnknownHeaderIsRefused and TestTheHeadersGdocSendsAreCarried are the
// pins.
//
// Both loops walk the raw header map and fold case themselves. Header.Get
// canonicalises the key it looks up, so it finds nothing stored under
// X-HTTP-METHOD-OVERRIDE, and two keys that fold to the same name are two lines
// on the wire. Counting one key at a time read each of them as the only one,
// which is how Authorization beside authorization put two credentials on a
// judged request and left the server to pick.
// TestTwoSpellingsOfOneHeaderAreCountedTogether is the pin.
//
// # What checkAuthorization cannot say
//
// checkAuthorization is the most the guard can honestly say about the
// credential, and the limit is worth holding. It checks one value, the "Bearer "
// scheme, and a token after it. That refuses a second credential, another
// scheme carrying another principal, and an empty grant.
//
// It cannot check whose token it is. The guard is built from a policy and a
// base transport and never sees the token, and pinning a literal would refuse
// the request after every refresh. So a caller that swaps in another person's
// bearer token runs the judged operation as that person. What the guard still
// bounds is which files are reachable and what may be done to them, which is
// principle 3's actual claim. Nothing here is a claim about identity. The
// refusal never prints the header value, because a refusal goes into the JSON
// the caller reports. TestTheAuthorizationHeaderMustBeABearerCredential is the
// pin.
//
// # A third write level, and the four kinds it carries
//
// LevelInPlace permits a batchUpdate without SUGGEST on one document, and
// GrantInPlace is the only door to it. Nail's decision of 2026-09-09,
// DECISIONS.md "M7 splits", which also holds the history of the door being
// deleted for having no production caller and then coming back beside the one
// line that calls it. It is the widest thing gdoc can be asked to do, so every
// part of it is narrow. Widening it, by another request kind or by a second
// call site, is Nail's decision and not a refactor.
//
// inPlaceKinds is four request kinds, and it is the bound writeMode used to be:
// updateDocumentStyle, updateParagraphStyle, updateTextStyle and
// updateTableCellStyle. judgeRequests carries a request kind nobody has heard of
// at every other level, and its own comment justifies that on writeMode: an
// unknown kind was still a suggestion somebody could reject. Nothing bounds it
// here, so the rule is inverted and a kind that is not on the list is refused
// whatever it is called. deleteHeader goes that way, and it is why that matters:
// it is a one-way door, and createHeader cannot put a first-page header back.
// TestTheOneWayDoorsAndTheUnknownKindAreRefusedInPlace is the pin.
//
// None of the four can change a character.
// TestNothingAtLevelInPlaceCanChangeACharacter is the pin, and a fifth kind
// answers that test rather than the list. createParagraphBullets is the case
// that shows the difference: it was measured landing on a real document and is
// refused all the same, because the reference says the leading tabs that set a
// bullet's nesting level "are removed by this request". Measured 2026-09-09,
// MEASURED.md "Which styling requests land in place".
//
// The allowlist gates every batchUpdate on a granted id, whatever writeMode
// says. judgeDocs reads lvl == LevelFull || isSuggestMode, so an allowlist hung
// off the direct-edit branch alone would let a granted document take an
// insertText under SUGGEST, and the property above would be false on exactly the
// id it protects. It is scoped to this level alone for the mirror reason:
// applied everywhere it refuses the probe's direct insertText at LevelFull and
// every propose batch at LevelSuggest.
// TestASuggestBatchOnAGrantedIDMeetsTheAllowlistToo and
// TestTheAllowlistIsScopedToTheInPlaceLevel are the pins.
//
// The fields mask is bounded too, and that is where the real danger is. The
// reference: "To reset a property to its default value, include its field name
// in the field mask but leave the field itself unset." So an updateTextStyle
// carrying fields: "*" resets bold, italic, links, colours and highlights over
// its range, permanently, with every character intact. What this level can
// destroy is everything except the text.
//
// checkInPlaceMask reads the mask the way isSuggestMode reads writeMode. It
// refuses a star, an empty mask, which Google reads as every field, and a mask
// naming useFirstPageHeaderFooter or useEvenPageHeaderFooter, because switching
// either off hides the first-page header that carries the logo, the one thing
// a restyle reports as unreachable. The key is read exactly and the two names
// are not, and that split is the point. The key decides whether the server acts
// on the mask at all, so reading it loosely would judge a field nothing acts on.
// The names are a denylist, so maskKey trims each segment, folds case and drops
// the underscores before the lookup: a path trimmed as a whole left the space in
// "documentStyle. useFirstPageHeaderFooter", and a field mask is defined in
// proto, where that camelCase names use_first_page_header_footer. That is
// checkFields' rule one layer out, and its refusal already says it: refused
// however it is asked for. TestAnInPlaceFieldMaskIsBounded is the pin, with
// TestAnOrdinaryMaskCarries in the other direction.
//
// A mask may name only what the request sets, and that is the same rule the
// star refusal is. A star and the fields written out one at a time destroy the
// same properties, so a guard that refuses one and carries the other bounds a
// spelling rather than the behaviour. checkMaskIsSet walks every path into the
// request's own style object, to the leaf, and refuses a path the request leaves
// unset, a style object that is absent, one spelled under another case, and one
// the walk cannot look inside. That is the builders' own rule held at the wire:
// a field named in a mask and forgotten in the style object clears that property
// on every paragraph, run or cell the request addressed. The segments are read
// exactly, so the underscore spelling of a path whose camelCase is set is
// refused here and would have been carried by Google. That direction is
// deliberate: no builder writes it, and the refusal is a run that failed loudly
// against a loss that is permanent. TestAMaskMayNameOnlyWhatTheRequestSets is
// the pin, with TestAMaskNamingExactlyWhatItSetsCarries in the other direction.
//
// A fifth kind carries here, and it is bounded by a grant rather than by the
// list. createNamedRange carries when AllowMarker named that exact range. It is
// not on inPlaceKinds because the list is the four styling kinds and this is not
// one of them: it sets no property, so no field mask bounds it. It writes gdoc's
// own marker over gdoc's own proposed prelude, and it adds and removes no
// character, so everything above still holds word for word. It is written rather
// than suggested because Docs refuses to apply it as a suggestion, in its own
// words: "Request does not support application as suggestion". Measured
// 2026-09-10, MEASURED.md "A named range over a suggested insertion".
// TestAGrantedMarkerCarriesAndNothingElseDoes and
// TestTheMarkerIsRefusedWithoutItsOwnGrant are the pins, with
// TestThePreludeNeedsNoGrantAtAll stating the other half: the prelude itself is
// an ordinary SUGGEST batch on a document with no grant of any kind.
//
// There is no capability probe at this level, and the read-back stands alone. A
// proposal is probed because what makes it a suggestion is writeMode, a field
// gdoc supplies and Google has ignored once. LevelInPlace makes no claim of that
// kind to the server, so there is nothing for a probe to test. What replaces the
// second bar is reading the document back. That sentence is about this level and
// not about the whole run: the prelude phase in front of it does make the
// writeMode claim, and internal/prelude says why it is unprobed too.
//
// # The multipart create is three signals that have to agree
//
// Drive picks the parser for a create body from three things: the /upload path
// prefix, the uploadType or upload_protocol value, and the media type on the
// request. multipartCreate reads the same three and refuses the request unless
// all three say multipart or none of them does. A guard reading JSON where Drive
// reads multipart, or the reverse, is judging a request it is not sending, which
// is the class Authorization beside authorization belongs to.
// TestTheThreeSignalsMustAgree is the pin.
//
//   - The boundary comes off the header, never out of the body.
//     mime.ParseMediaType reads it, which is the same value Google's own parser
//     uses, and it refuses a boundary given twice where a hand-rolled split
//     would take one and leave the server the other. A guard that scanned the
//     body for something boundary-shaped would let the file's own bytes move
//     where it thinks the metadata ends, so a test sends a file carrying the
//     boundary string inside it.
//     TestTheBoundaryComesFromTheHeaderAndMustBeUsable is the pin.
//   - The first part only, and it must be JSON. metadataPart reads one part with
//     mime/multipart and hands its bytes to the same duplicate-key and parents
//     check a plain JSON create goes through. Everything behind that part is
//     opaque. A part whose media type is not JSON is refused: the guard reading
//     it as JSON while Drive reads it as something else is the same mismatch one
//     layer in. TestTheFirstPartMustBeJSON, TestTheGuardReadsTheFirstPartOnly and
//     TestTheFirstPartsParentsAreCheckedLikeAPlainCreate are the pins.
//   - NextRawPart, and no Content-Transfer-Encoding. NextPart decodes a
//     quoted-printable part, and the guard must read the bytes Drive is sent
//     rather than a decoding of them. So the read is raw and an encoded part is
//     refused instead. allowedPartHeaders is content-type alone, an allowlist
//     for the reason the request's own header rule is one, and a header given
//     twice is refused because which one the server reads is not decided here.
//   - The part is read out of the peek. A metadata part whose closing boundary
//     sits past maxPeek ends the read short and is refused rather than judged
//     half-read. Nothing gdoc builds can reach that: the metadata part is small
//     and first. Something somebody adds later can, and the refusal is how they
//     find out. TestAMetadataPartOutsideThePeekIsRefused is the pin.
//
// A multipart create with no AllowCreateIn grant is refused like any other
// create, and a well-formed one naming exactly the granted folder is carried,
// with the new id learned at LevelFull off the create's own answer.
// TestAMultipartCreateWithNoGrantIsRefused,
// TestAMultipartCreateNamingTheGrantedFolderIsCarried and
// TestTheWholeMultipartBodyReachesTheWire are the pins.
//
// # The guard judges plain paths only
//
// A path whose escaping differs from its plain form (%2F, %2E and the like), or
// a . or .. segment, is refused before the host is looked at. A harmless %20
// passes: Go only sets RawPath when the escaped form differs from the default
// encoding of Path. A URL carrying credentials is refused too, because those
// become an Authorization header gdoc did not build.
//
// The reason is not tidiness. u.Path is decoded, u.EscapedPath() is what goes
// out, and Go cleans no dot segments out of a URL. Without the rule the guard
// reads one URL and the transport sends another:
// /drive/v3/files/DOC1/../../../about passes as a read of DOC1 and arrives as
// drive.about.get, an endpoint the guard refuses when it is asked plainly.
// DOC1%2F..%2Fabout is the same walk through a second door. Real Docs and Drive
// ids are [A-Za-z0-9_-], so refusing both shapes costs nothing.
// TestAKnownIDMayNotWalkToAnEndpointTheGuardRefuses is the pin, with
// TestAPlainPathIsStillCarried in the other direction and
// TestHostEdgeCasesAreRefused over the credentials.
//
// # The policy is mutex-guarded
//
// NewClient hands out an *http.Client, which the standard library documents as
// safe for concurrent use, and Learn writes the set from inside RoundTrip while
// Judge reads it and CheckRedirect reads it from another goroutine. Without the
// lock two parallel requests are a concurrent map read and write, which is a
// fatal error rather than a recoverable one. make test runs -race; keep it
// there. TestOneClientIsSafeFromManyGoroutines is the pin.
package guard
