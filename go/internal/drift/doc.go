// Package drift is the one list of measured values, read out of a docx and out
// of a Docs answer.
//
// The house style became a config file and the master docx became provenance.
// What answers the risk in that trade is one comparison: read the same list of
// values out of a document built from house.yaml and out of the master, and
// fail when a value moves. DECISIONS.md says it plainly, "the 160-item
// comparison stays as a test. It renders from the config, renders from the
// master, and fails when any value drifts."
//
// Nothing here judges. A Row carries what each side said and a verdict about
// the two values, and whether a difference matters is Nail's, in Word or in
// Drive. Known is the list of differences already looked at, and each entry
// carries its explanation rather than the word "expected".
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # The item list is written once, and read two ways
//
// Items in items.go is the list, ported from compare.py: each item has a name,
// a tolerance and one reader, and Source gives that reader both a docx and a
// Docs answer to read from. Two extractors written out by hand would be two
// chances for one row to mean one thing offline and another thing live, and
// the two gates would then disagree without either of them failing. The names
// are compare.py's on both sides, so a row that fails in one gate is findable
// in the other and in the 2026-08-29 report. TestFromDocxReadsTheMaster and
// TestFromDocReadsTheDocsAnswer are the pins, and each states the house value
// as a literal rather than reading house.yaml: a test that reads the constant
// it checks is a mirror, and it follows that constant wherever somebody moves
// it.
//
// Both halves resolve, and each resolves in its own vocabulary. The docx half
// walks docDefaults and the basedOn chain; the Docs half walks the run, the
// paragraph's named style and NORMAL_TEXT. Neither may report only what the
// nearest message states, because the Docs API leaves an inherited property out
// of the message altogether. TestARunThatStatesNothingReadsTheStyleBehindIt and
// TestTheHeadingsColourComesFromTheStyleWhenTheRunIsSilent are the pins.
//
// # The gate runs both ways
//
// Offline, in make test. TestTheOfflineGate builds 03-policy.md from the
// embedded config and reads the master docx beside it. 169 items, and every
// DIFFERENT or MISSING row is named in Known with the reason it is there. A
// CLOSE row is never in Known and cannot be, because Unexplained only ever asks
// for an entry on the two verdicts that fail the gate.
//
// TestTheGateReadsTheWholeList is the other direction: a gate that compared
// nothing would pass, so that test says the list is read whole, that no two
// rows share a name, and that the rows neither document states anything about
// are exactly the ones written out in unstated. That is what makes "no new
// differences" mean anything. TestFromDocReadsEveryItem asks the same of the
// Docs half.
//
// Live, behind GDOC_LIVE_TEST=1 GDOC_LIVE_WRITE=1. TestLiveDrift in
// internal/live builds the same note, uploads it and the master with
// conversion, reads both through the Docs API, runs this same list, prints the
// table and fails on any verdict that is not IDENTICAL, CLOSE or a name in
// Known. That is the measurement that means something, because Google's import
// is part of the result: a row that differs there and not offline is Drive's
// import, and a row that differs in both is the generator. It carries SPEC
// acceptance item 3 as well, asking the positioned logo, the contents field and
// the footer page numbers twice each. The value says the built document really
// carries the thing, so a generator that stopped emitting it fails rather than
// matching a master that lost it too, and the verdict says the master agrees,
// so a value that survived the import as something else fails as well.
//
// # The live read is not docs.URL
//
// docs.URL asks for includeTabsContent=true, which moves the content into tabs
// and leaves the legacy body, headers and footers empty. Those legacy fields
// are exactly what Doc reads, and they carry the first tab, which is the whole
// of a document converted from one docx. Read through docs.URL the live gate
// would answer nil for every body, header and footer row and pass on a document
// it never looked inside. So readForDrift in internal/live spells a bare
// documents.get with no query at all, which the guard's docsReadParams
// allowlist carries because an empty query names no parameter.
//
// # Known explains a difference between the two documents, never a fault here
//
// Known holds twenty-five names, where the 2026-08-29 report counted two real
// differences. Neither number is wrong, and what separates them is what the
// offline gate reads. It reads XML, where the master declares a heading colour
// and a heading indent on the style and then overrides both on every paragraph
// that uses them, while house.yaml states the effective one a reader sees; that
// is eight rows. The two documents also hold different words, the master being
// the template with "xxx" where a title goes, so every row that reads text
// rather than a measurement differs for that reason and only that reason. Seven
// entries are the front matter the note fills in: the master leaves the owner
// and the approval dates blank, keeps its "xx" revision row and a blank row
// behind it, and was captured with Internal marked. The live gate reads what
// Docs resolved, so most of those rows would answer IDENTICAL there. Adding a
// name to Known is a decision somebody writes down with its reason, not a test
// somebody loosens, and two tests hold the list in both directions:
// TestEveryKnownDifferenceNamesARealItem says every entry names a row the list
// produces, and TestEveryKnownDifferenceStillDiffers says every entry still
// names a row that really disagrees. An entry whose row went IDENTICAL exempts
// that row for ever.
//
// Row.Bug is the split between a difference and a fault. A reading that did not
// line up, by name or by length, and two halves of one row answering in
// different types are all faults in this package, so they set it, and
// Unexplained returns a row carrying it whatever Known says about that name.
// Ten of the twenty-five names carry rows that can go wrong that way, so
// filtering on the name alone dropped them and the gate passed in silence. The
// type check is the arm that found it: two halves of one row are one question
// asked two ways, so they answer in one type, and the %v fallback behind them
// read "1" against 1.0 as IDENTICAL. TestABugRowIsNeverExplainedAway and
// TestATypeMismatchIsABugRatherThanAnAnswer are the pins, with
// TestCompareSaysSoWhenTheTwoReadingsDoNotLineUp over the other two faults.
//
// # Two readers had a hole each, and both are closed
//
// Docx.Style folded the document defaults in before it walked the style chain,
// and the chain is empty when the style is absent, so a style that is in no
// file at all came back stating 11pt Calibri at 115% while its own doc comment
// promised nothing. The master states the same defaults, so dropping a named
// style from house.yaml put six of that style's eleven rows IDENTICAL on both
// sides and the gate passed on a style that no longer existed. found is what
// closed it, and TestAStyleThatIsNotInTheFileCarriesNothing is the pin.
//
// round3 is the other. It cast through int64, and Go leaves that conversion out
// of range implementation dependent, so NaN read as 0 on darwin/arm64 and as
// -9.2e15 on darwin/amd64. make dist ships both, so one document measured two
// ways gave one gate two answers. It goes through math.Round now, which hands
// NaN back and puts the row DIFFERENT rather than inventing a number.
// TestARoundedNumberIsTheSameOnEveryArchitecture is the pin.
//
// # The three PDF items are dropped
//
// compare.py read a PDF export for a page count, a page-1 size and a page-1
// image count. SPEC's Never list says gdoc never exports a PDF, so neither gate
// does, and nothing here opens one. The page count was one of the two real
// differences the 2026-08-29 report counted, and it came from a stale contents
// list in the master rather than from the style. 160 was compare.py's count
// with those three in it; 157 without them, plus bold and italic for all nine
// named styles, is the 169 here. Every item compare.py printed is findable
// under compare.py's own name.
package drift
