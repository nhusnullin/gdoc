// What no file in this package may contain.
//
// The app reads one directory of forgeable lines and its own plist, and runs
// six argv lists at a path it resolved itself. It opens no socket, so a line
// somebody planted in the record has no wire to reach even if the app
// believed every word of it; and it starts a program in one file only, so
// what it can run is a list somebody can read in one sitting. The rules are
// source scans rather than a review, because the next person to add a
// convenience here will not have read the specification.
//
// The second scan is the one sentence M23 reversed: that app launched
// nothing at all, and this one launches the six.

import Foundation
import XCTest

final class SourceRulesTests: XCTestCase {
  /// Where the one room that starts a program is, relative to the package.
  private static let runner = "Sources/PresenceApp/Runner.swift"

  func testNoSourceFileReachesTheWire() throws {
    // Spelled in pieces so this file is not its own failure.
    let session = "URL" + "Session"
    let network = "import " + "Network"

    var seen = 0
    try eachSwiftFile { path, text in
      seen += 1
      XCTAssertFalse(text.contains(session), "\(path) reaches the wire")
      XCTAssertFalse(text.contains(network), "\(path) reaches the wire")
    }
    XCTAssertGreaterThan(seen, 5, "no Swift files were read, so this test guards nothing")
  }

  func testProcessAppearsOnlyInTheRunner() throws {
    let child = "Proc" + "ess"

    var sawTheRunner = false
    try eachSwiftFile { path, text in
      let named = text.split(separator: "\n").filter { word(child, isIn: String($0)) }
      guard path != Self.runner else {
        sawTheRunner = true
        XCTAssertFalse(named.isEmpty, "\(path) is the one file that starts a program")
        return
      }
      for line in named {
        XCTFail("\(path) names a \(child): \(line.trimmingCharacters(in: .whitespaces))")
      }
    }
    XCTAssertTrue(sawTheRunner, "\(Self.runner) is not there, so this test guards nothing")
  }

  /// The argv set is closed at six, in the source and not only in a run.
  ///
  /// A seventh case is a seventh thing the app can do to somebody's
  /// documents, so it is a decision in `docs/v2/DECISIONS.md` and a line in
  /// CLAUDE.md's invariant, never a case added while fixing something else.
  func testTheArgvEnumIsClosedAtSix() throws {
    let text = try String(
      contentsOf: packageRoot().appendingPathComponent(Self.runner), encoding: .utf8)
    let opening = try XCTUnwrap(
      text.range(of: "public enum Call"), "the argv enum is not in \(Self.runner)")
    let body = text[opening.upperBound...]
    let closing = try XCTUnwrap(body.range(of: "\n}"), "the argv enum is never closed")
    let declared = body[..<closing.lowerBound]
      .split(separator: "\n")
      .map { $0.trimmingCharacters(in: .whitespaces) }
      .filter { $0.hasPrefix("case ") && !$0.hasPrefix("case .") }
    XCTAssertEqual(
      declared,
      [
        "case help", "case authStatus", "case authLogin", "case authLogout", "case update",
        "case updateDesktop",
      ])
  }

  /// Every Swift file in the package, as its path relative to the package
  /// root and its text.
  private func eachSwiftFile(_ body: (String, String) throws -> Void) throws {
    let root = packageRoot()
    for directory in ["Sources", "Tests"] {
      let base = root.appendingPathComponent(directory)
      let walker = try XCTUnwrap(
        FileManager.default.enumerator(atPath: base.path), "\(directory) is not there")
      for case let name as String in walker where name.hasSuffix(".swift") {
        let text = try String(
          contentsOf: base.appendingPathComponent(name), encoding: .utf8)
        try body("\(directory)/\(name)", text)
      }
    }
  }

  /// The word on its own, so the one type whose name begins with it and reads
  /// nothing but this machine's own pid is not a failure.
  private func word(_ needle: String, isIn line: String) -> Bool {
    var search = line.startIndex..<line.endIndex
    while let found = line.range(of: needle, range: search) {
      let before =
        found.lowerBound == line.startIndex ? nil : line[line.index(before: found.lowerBound)]
      let after = found.upperBound == line.endIndex ? nil : line[found.upperBound]
      let bounded = !isWordish(before) && !isWordish(after)
      if bounded { return true }
      search = found.upperBound..<line.endIndex
    }
    return false
  }

  private func isWordish(_ character: Character?) -> Bool {
    guard let character else { return false }
    return character.isLetter || character.isNumber || character == "_"
  }

  /// This file is Tests/PresenceCoreTests/SourceRulesTests.swift, so the
  /// package is three directories up from it. The path is the test file's own
  /// and never the working directory, which swift test does not promise.
  private func packageRoot() -> URL {
    URL(fileURLWithPath: #filePath)
      .deletingLastPathComponent()
      .deletingLastPathComponent()
      .deletingLastPathComponent()
  }
}
