// The finder, over the four answers Spotlight can lead to.
//
// No test here runs a real metadata query: the search is handed over, so
// what each case states is which candidates came back and what their front
// matter says. The one case that is not about a count is the document map,
// a file that names the id in a sentence: Spotlight returns it and the
// front-matter check throws it away, which is the measurement of 2026-10-18
// written as a test.

import PresenceCore
import XCTest

@testable import PresenceApp

final class NoteFinderTests: XCTestCase {
  private static let doc = "1gH4y7uvT4QgpZBCHHlHz9g5L9R0R7dzONs9cJOTcY0c"
  private static let other = "1x8UY2HbBY2Moo7pkzQ2yiH49EoPtZlLthaVYsGqv_Bg"

  /// A note whose `gdoc:` block names this document.
  private static func note(_ id: String) -> String {
    """
    ---
    title: a note
    gdoc:
      schema: 1
      document_id: \(id)
    ---

    Body.
    """
  }

  /// A file that names the id in a sentence, which is what Nail's document
  /// map does.
  private static func map(_ id: String) -> String {
    "# Document map\n\nThe paper lives at \(id) in Drive.\n"
  }

  /// A front matter that names the id under somebody else's key.
  private static func decoy(_ id: String) -> String {
    "---\ntitle: a decoy\nother:\n  document_id: \(id)\n---\n\nBody.\n"
  }

  func testOneNoteNamingTheDocumentIsTheAnswer() async {
    let finder = NoteFinder(
      search: StatedSearch([Self.doc: ["/notes/paper.md"]]),
      read: { _ in Self.note(Self.doc) })

    let found = await finder.note(for: Self.doc)
    XCTAssertEqual(try? found.get(), "/notes/paper.md")
  }

  func testNoCandidateAtAllIsNotPaired() async {
    let finder = NoteFinder(search: StatedSearch([:]), read: { _ in "" })

    let found = await finder.note(for: Self.doc)
    XCTAssertEqual(found.fault, .notPaired)
    XCTAssertEqual(NoteFault.notPaired.tip, "No note is paired with this document")
  }

  /// Two notes is a sentence of its own and never a pick: there is no folder
  /// and no order here worth trusting, so a click that opened one of the two
  /// at random would be worse than a click that says what is wrong.
  func testTwoNotesNamingOneDocumentIsAFaultOfItsOwn() async {
    let finder = NoteFinder(
      search: StatedSearch([Self.doc: ["/notes/a.md", "/notes/b.md"]]),
      read: { _ in Self.note(Self.doc) })

    let found = await finder.note(for: Self.doc)
    XCTAssertEqual(found.fault, .twoNotes)
    XCTAssertEqual(NoteFault.twoNotes.tip, "Two notes name this document")
  }

  /// Spotlight answers on the text of a file, so a document map and a note
  /// with the id under another key both come back. The front-matter check is
  /// what tells the note from them, and it is the same reading the hub scan
  /// did: testTheHubIndexReadsOnlyTheFrontMattersGdocBlock.
  func testAFileThatMerelyMentionsTheIdIsNotTheNote() async {
    let texts = [
      "/notes/map.md": Self.map(Self.doc),
      "/notes/decoy.md": Self.decoy(Self.doc),
      "/notes/paper.md": Self.note(Self.doc),
    ]
    let finder = NoteFinder(
      search: StatedSearch([Self.doc: ["/notes/map.md", "/notes/decoy.md", "/notes/paper.md"]]),
      read: { texts[$0] ?? "" })

    let found = await finder.note(for: Self.doc)
    XCTAssertEqual(try? found.get(), "/notes/paper.md")

    // And with no note among them the two candidates pair nothing.
    let mapAlone = NoteFinder(
      search: StatedSearch([Self.doc: ["/notes/map.md", "/notes/decoy.md"]]),
      read: { texts[$0] ?? "" })
    let alone = await mapAlone.note(for: Self.doc)
    XCTAssertEqual(alone.fault, .notPaired)
  }

  /// An answer is given once and kept, and a Spotlight notice is what throws
  /// it away: without that the panel would query for every document on every
  /// redraw, and with no way to throw it away a note written a minute ago
  /// would never be found.
  func testAnAnswerIsKeptUntilTheNoticeThrowsItAway() async {
    let search = StatedSearch([Self.doc: []])
    let forgotten = Counter()
    let finder = NoteFinder(
      search: search, read: { _ in Self.note(Self.doc) },
      forgot: { _ in forgotten.add() })

    let first = await finder.note(for: Self.doc)
    XCTAssertEqual(first.fault, .notPaired)
    XCTAssertEqual(search.asked.count, 1)
    let again = await finder.note(for: Self.doc)
    XCTAssertEqual(again.fault, .notPaired, "the answer is kept")
    XCTAssertEqual(search.asked.count, 1, "and no second query was run")

    search.willAnswer([Self.doc: ["/notes/paper.md"]])
    await finder.forget(Self.doc)
    XCTAssertEqual(forgotten.count, 1, "the session is told its copy is stale")
    let afterwards = await finder.note(for: Self.doc)
    XCTAssertEqual(try? afterwards.get(), "/notes/paper.md")
    XCTAssertEqual(search.asked.count, 2)
  }

  /// Each document is its own answer: a notice about one throws away one.
  func testOneNoticeThrowsAwayOneDocumentsAnswer() async {
    let search = StatedSearch([
      Self.doc: ["/notes/paper.md"], Self.other: ["/notes/africa.md"],
    ])
    let finder = NoteFinder(
      search: search,
      read: { path in
        Self.note(path == "/notes/paper.md" ? Self.doc : Self.other)
      })

    _ = await finder.note(for: Self.doc)
    _ = await finder.note(for: Self.other)
    XCTAssertEqual(search.asked.count, 2)

    await finder.forget(Self.doc)
    _ = await finder.note(for: Self.other)
    XCTAssertEqual(search.asked.count, 2, "the other document's answer is untouched")
    _ = await finder.note(for: Self.doc)
    XCTAssertEqual(search.asked.count, 3)
  }

  /// The query string is the word form over Markdown, written out, so the
  /// measurement in `docs/v2/MEASURED.md` is the query this app runs.
  func testTheQueryIsTheWordFormOverMarkdown() {
    XCTAssertEqual(
      SpotlightSearch.queryString(for: "1gH4y7uvT4QgpZBCHHlHz9g5L9R0R7dzONs9cJOTcY0c"),
      "kMDItemTextContent = \"1gH4y7uvT4QgpZBCHHlHz9g5L9R0R7dzONs9cJOTcY0c\"cdw"
        + " && kMDItemFSName = \"*.md\"c")
  }

  /// An id that is not a Drive id never reaches a query string, because the
  /// search refuses it before one is written. Every id in this app arrives
  /// in a line of the record, which any process with the owner's rights can
  /// write.
  ///
  /// No query runs here: the refusal is the first line of `candidates`.
  func testAnIdThatIsNotADriveIdNeverReachesAQuery() async {
    let planted = "\" || kMDItemFSName = \"*"
    XCTAssertFalse(LinkLine.isDocumentID(planted))

    let found = await SpotlightSearch().candidates(naming: planted) { _ in }
    XCTAssertEqual(found, [], "nothing was asked of Spotlight at all")
  }
}

/// A search that answers from a literal and writes down what it was asked.
/// It is the whole of why no test in this target runs a real Spotlight
/// query.
final class StatedSearch: NoteSearch, @unchecked Sendable {
  private let lock = NSLock()
  private var answers: [String: [String]]
  private var questions: [String] = []

  init(_ answers: [String: [String]] = [:]) {
    self.answers = answers
  }

  /// Every document asked about, in order, so a test can say how many
  /// queries one draw ran.
  var asked: [String] {
    lock.lock()
    defer { lock.unlock() }
    return questions
  }

  /// What the next query comes back with.
  func willAnswer(_ answers: [String: [String]]) {
    lock.lock()
    self.answers = answers
    lock.unlock()
  }

  func candidates(
    naming id: String, changed: @escaping @Sendable (String) -> Void
  ) async -> [String] {
    take(id)
  }

  /// The answer, and one more ask written down. It is its own function
  /// because a lock may not be held across an `await`.
  private func take(_ id: String) -> [String] {
    lock.lock()
    defer { lock.unlock() }
    questions.append(id)
    return answers[id] ?? []
  }
}

/// How many times the finder said it had thrown an answer away.
private final class Counter: @unchecked Sendable {
  private let lock = NSLock()
  private var seen = 0

  func add() {
    lock.lock()
    seen += 1
    lock.unlock()
  }

  var count: Int {
    lock.lock()
    defer { lock.unlock() }
    return seen
  }
}
