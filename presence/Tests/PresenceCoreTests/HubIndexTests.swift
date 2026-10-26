// What the app reads of somebody's notes, and what it refuses to.
//
// The hub is a folder of Markdown. The app reads one block of one note's
// front matter and nothing else, so a note's text never reaches the app at
// all. Fixtures/hub holds four notes: three gdoc published or exported, and
// one that names a document id in its body and under another key, which is
// the thing this index must not see.

import Foundation
import XCTest

@testable import PresenceCore

final class HubIndexTests: XCTestCase {

  private static let supplier = "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd"
  private static let treasury = "2BcDeFgHiJkLmNoPqRsTuVwXyZ0123456789bcde"
  private static let treasuryTab = "3CdEfGhIjKlMnOpQrStUvWxYzA0123456789cdef"
  private static let archived = "4DeFgHiJkLmNoPqRsTuVwXyZaB0123456789defa"
  private static let neverPaired = "5EfGhIjKlMnOpQrStUvWxYzaBc0123456789efab"
  private static let inATextFile = "6FgHiJkLmNoPqRsTuVwXyZaBcD0123456789fabc"

  private func hub() throws -> String {
    let url = try XCTUnwrap(
      Bundle.module.url(forResource: "hub", withExtension: nil, subdirectory: "Fixtures"),
      "Fixtures/hub is not in the test bundle")
    return url.path
  }

  func testTheHubIndexReadsOnlyTheFrontMattersGdocBlock() throws {
    let index = HubIndex.scan(hub: try hub())

    XCTAssertEqual(index.notes[Self.supplier], "notes/supplier-register.md", "schema 1")
    XCTAssertEqual(index.notes[Self.treasury], "notes/treasury-policy.md", "schema 2, first entry")
    XCTAssertEqual(
      index.notes[Self.treasuryTab], "notes/treasury-policy.md", "schema 2, second entry")
    XCTAssertEqual(index.notes[Self.archived], "archive/old-note.md", "a note in a subfolder")

    XCTAssertNil(
      index.notes[Self.neverPaired],
      "an id in the body and under another front-matter key is not a pairing")
    XCTAssertNil(index.notes[Self.inATextFile], "a .txt is not a note")
    XCTAssertEqual(index.notes.count, 4, "four ids over three notes, and nothing else")
    XCTAssertTrue(
      index.notePath(for: Self.supplier)?.hasSuffix("/hub/notes/supplier-register.md") ?? false,
      "and the same note as the absolute path the opener is handed")

    // And the reading itself, over the text, so the rule is stated once.
    XCTAssertEqual(
      HubIndex.documentIDs(
        inFrontMatterOf: """
          ---
          gdoc:
            schema: 1
            document_id: \(Self.supplier)
          ---

          gdoc:
            document_id: \(Self.neverPaired)
          """),
      [Self.supplier], "nothing past the closing fence is front matter")
    XCTAssertEqual(
      HubIndex.documentIDs(inFrontMatterOf: "# No front matter at all\n"), [],
      "a note with no front matter is unpaired")
    XCTAssertEqual(
      HubIndex.documentIDs(inFrontMatterOf: "---\ntitle: unterminated\ngdoc:\n  schema: 1\n"), [],
      "front matter that never closes is front matter the app does not have")
    XCTAssertEqual(
      HubIndex.documentIDs(
        inFrontMatterOf: """
          ---
          gdoc:
            schema: 1
            document_id: ../../etc/passwd
          ---
          """),
      [], "a path in the id field is refused by the id's own shape")
  }

  /// Two notes naming one document is somebody's own mistake, not a crash:
  /// the index takes one of them and takes the same one every time.
  func testOneDocumentTakesTheFirstNoteByPath() throws {
    let hub = try temporaryHub()
    try note(in: hub, at: "b-second.md", id: Self.supplier)
    try note(in: hub, at: "a-first.md", id: Self.supplier)

    XCTAssertEqual(HubIndex.scan(hub: hub).notes[Self.supplier], "a-first.md")
  }

  func testTheIndexRescansOnAFileEvent() throws {
    let hub = try temporaryHub()
    XCTAssertTrue(HubIndex.scan(hub: hub).notes.isEmpty, "an empty hub pairs nothing")

    let rescanned = expectation(description: "the watcher said something changed")
    rescanned.assertForOverFulfill = false
    let latest = Latest()
    let watcher = HubWatcher(hub: hub, debounce: 0.1) {
      let next = HubIndex.scan(hub: hub)
      latest.put(next)
      if next.notes[Self.supplier] != nil { rescanned.fulfill() }
    }
    watcher.start()
    defer { watcher.stop() }

    try note(in: hub, at: "notes/new-note.md", id: Self.supplier)
    wait(for: [rescanned], timeout: 20)

    XCTAssertEqual(latest.read().notes[Self.supplier], "notes/new-note.md")
  }

  /// Open md is built from an absolute path and needs a regular `.md` at
  /// the end of it.
  ///
  /// It is the `path` form of Obsidian's scheme since the third look. A note
  /// Spotlight found is in whatever folder the person keeps it in, and there
  /// is no vault name to write: Obsidian resolves the path against the
  /// vaults it has open. The proving is what is left of the old inside-the-
  /// hub check, and it is now about the file and not about a folder.
  func testOpenMdNeedsAnAbsolutePathToARegularMdFile() throws {
    let hub = try temporaryHub()
    try note(in: hub, at: "notes/supplier register.md", id: Self.supplier)
    try FileManager.default.createDirectory(
      at: URL(fileURLWithPath: hub).appendingPathComponent("notes/folder.md"),
      withIntermediateDirectories: true)
    let inside = { (relative: String) in
      URL(fileURLWithPath: hub).appendingPathComponent(relative).path
    }

    XCTAssertEqual(
      try HubIndex.noteURL(absolute: inside("notes/supplier register.md")).get().absoluteString,
      "obsidian://open?path=" + encoded(inside("notes/supplier register.md")),
      "the absolute path, percent-encoded, and no line parameter")

    XCTAssertEqual(HubIndex.noteURL(absolute: inside("notes/gone.md")).fault, .movedOrDeleted)
    XCTAssertEqual(
      HubIndex.noteURL(absolute: inside("notes/folder.md")).fault, .movedOrDeleted,
      "a directory named .md is not a note")
    XCTAssertEqual(
      HubIndex.noteURL(absolute: "notes/supplier register.md").fault, .notMarkdown,
      "a relative path is not a path at all here")
    XCTAssertEqual(
      HubIndex.noteURL(absolute: inside("notes/supplier register.txt")).fault, .notMarkdown)
    XCTAssertEqual(
      HubIndex.noteURL(absolute: "/etc/passwd").fault, .notMarkdown,
      "a file that is not Markdown is refused wherever it is")
  }

  /// The one escaping rule, written here rather than read off the source:
  /// everything but the unreserved set is percent-encoded, the separators
  /// included.
  private func encoded(_ path: String) -> String {
    let unreserved = CharacterSet(charactersIn: "-._~").union(.alphanumerics)
    return path.addingPercentEncoding(withAllowedCharacters: unreserved) ?? ""
  }

  /// A dot folder is somebody's machinery and is skipped whole. A dot file
  /// is one file: Finder drops `.DS_Store` into every folder a person opens,
  /// and it sorts before the letters, so skipping the rest of the folder
  /// after one would hide most of a vault.
  func testADotFileHidesNothingAndADotFolderIsSkippedWhole() throws {
    let hub = try temporaryHub()
    try note(in: hub, at: "notes/supplier-register.md", id: Self.supplier)
    try note(in: hub, at: ".obsidian/plugins/cached-note.md", id: Self.treasury)
    try dotFile(in: hub, at: "notes/.DS_Store")

    let index = HubIndex.scan(hub: hub)

    XCTAssertEqual(
      index.notes[Self.supplier], "notes/supplier-register.md",
      "a .DS_Store beside a note hides nothing: it is one file, not a folder")
    XCTAssertNil(index.notes[Self.treasury], "a note inside a dot folder is not the hub's")
  }

  /// A folder with more entries than the walk reads is walked no further,
  /// and the index says why rather than carrying half a pairing.
  ///
  /// Half a pairing would be the worse answer: the notes under the first
  /// few hundred entries would pair, every note past them would not, and
  /// nothing on the screen would say which half a grey icon was in. The
  /// number here is three, so the tree is one a reader can count; the one
  /// the app runs with is `HubIndex.entryCeiling`.
  func testAHubOverTheCeilingIsWalkedNoFurther() throws {
    let hub = try temporaryHub()
    try note(in: hub, at: "a.md", id: Self.supplier)
    try note(in: hub, at: "b.md", id: Self.treasury)
    try note(in: hub, at: "c.md", id: Self.archived)

    XCTAssertEqual(HubIndex.entryCeiling, 20_000, "the number the app walks with")

    let under = HubIndex.scan(hub: hub, ceiling: 3)
    XCTAssertEqual(under.notes.count, 3, "three entries under a ceiling of three, all read")
    XCTAssertNil(under.stop, "and the walk saw the whole tree")

    let over = HubIndex.scan(hub: hub, ceiling: 2)
    XCTAssertEqual(over.stop, .overTheCeiling, "the third entry is one too many")
    XCTAssertEqual(over.notes, [:], "so there is no pairing at all, not the first two")
    XCTAssertEqual(over.hub, hub, "and the folder it stopped in is named")
  }

  /// A dot folder's own entries are not counted against the ceiling: they
  /// are not walked at all, so a vault with a large `.obsidian` cache is
  /// the folder the person keeps notes in and nothing else.
  func testADotFolderCostsTheCeilingNothing() throws {
    let hub = try temporaryHub()
    try note(in: hub, at: "a.md", id: Self.supplier)
    for which in 0..<5 {
      try note(in: hub, at: ".obsidian/cache-\(which).md", id: Self.treasury)
    }

    let index = HubIndex.scan(hub: hub, ceiling: 2)
    XCTAssertNil(index.stop, "one note and one dot folder is two entries")
    XCTAssertEqual(index.notes[Self.supplier], "a.md")
  }

  /// A walk that is asked to let go lets go between two entries, and
  /// carries no pairing: whoever asked it to stop wanted a newer one.
  func testAWalkThatIsCancelledCarriesNoPairing() throws {
    let hub = try temporaryHub()
    try note(in: hub, at: "a.md", id: Self.supplier)
    try note(in: hub, at: "b.md", id: Self.treasury)

    let index = HubIndex.scan(hub: hub, isCancelled: { true })
    XCTAssertEqual(index.stop, .cancelled, "it let go before the first entry")
    XCTAssertEqual(index.notes, [:], "and read nothing")

    var entries = 0
    let partway = HubIndex.scan(
      hub: hub,
      isCancelled: {
        entries += 1
        return entries > 1
      })
    XCTAssertEqual(partway.stop, .cancelled, "and in the middle of a tree the same")
    XCTAssertEqual(partway.notes, [:])
  }

  /// The watcher is handed to a C stream that calls back on a queue of its
  /// own, so the stream holds it up: a callback already in flight never
  /// lands on memory the owner freed.
  func testTheStreamHoldsTheWatcherUpWhileItRuns() throws {
    let hub = try temporaryHub()
    var strong: HubWatcher? = HubWatcher(hub: hub, debounce: 0.1) {}
    weak var seen = strong
    strong?.start()

    strong = nil
    let held = try XCTUnwrap(seen, "the running stream holds the watcher up")
    held.stop()
  }

  // MARK: a hub of this test's own

  private func temporaryHub() throws -> String {
    let base = URL(fileURLWithPath: NSTemporaryDirectory())
      .appendingPathComponent("hub-\(UUID().uuidString)")
      .appendingPathComponent("hub")
    try FileManager.default.createDirectory(at: base, withIntermediateDirectories: true)
    addTeardownBlock {
      try? FileManager.default.removeItem(at: base.deletingLastPathComponent())
    }
    // Resolved, because a temporary directory on this machine is reached
    // through a symlink and the paths under test are compared whole.
    return base.resolvingSymlinksInPath().path
  }

  private func dotFile(in hub: String, at relative: String) throws {
    let url = URL(fileURLWithPath: hub).appendingPathComponent(relative)
    try FileManager.default.createDirectory(
      at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
    try Data("\0\u{1}".utf8).write(to: url)
  }

  private func note(in hub: String, at relative: String, id: String) throws {
    let url = URL(fileURLWithPath: hub).appendingPathComponent(relative)
    try FileManager.default.createDirectory(
      at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
    try "---\ngdoc:\n  schema: 1\n  document_id: \(id)\n---\n\n# A note\n"
      .write(to: url, atomically: true, encoding: .utf8)
  }
}

/// The last index a watcher handed over, where a thread that is not this
/// test's own can put it.
final class Latest: @unchecked Sendable {
  private let lock = NSLock()
  private var index = HubIndex.empty

  func put(_ next: HubIndex) {
    lock.lock()
    index = next
    lock.unlock()
  }

  func read() -> HubIndex {
    lock.lock()
    defer { lock.unlock() }
    return index
  }
}

/// Why a result failed, where it did. One line so a table of refusals reads
/// as a table.
extension Result {
  var fault: Failure? {
    guard case .failure(let why) = self else { return nil }
    return why
  }
}
