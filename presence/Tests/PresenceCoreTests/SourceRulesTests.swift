// What no file in this package may contain.
//
// The app reads one directory of forgeable lines and its own plist. It opens
// no socket and starts no program, so a line somebody planted in the record
// has nothing to reach even if the app believed every word of it. The rule is
// a source scan rather than a review, because the next person to add a
// convenience here will not have read the specification.

import Foundation
import XCTest

final class SourceRulesTests: XCTestCase {
  func testNoSourceFileImportsNetworkOrLaunchesAProcess() throws {
    // Spelled in pieces so this file is not its own failure.
    let session = "URL" + "Session"
    let network = "import " + "Network"
    let child = "Proc" + "ess"

    let root = packageRoot()
    var seen = 0
    for directory in ["Sources", "Tests"] {
      let base = root.appendingPathComponent(directory)
      let walker = try XCTUnwrap(
        FileManager.default.enumerator(atPath: base.path), "\(directory) is not there")
      for case let name as String in walker where name.hasSuffix(".swift") {
        let path = base.appendingPathComponent(name)
        let text = try String(contentsOf: path, encoding: .utf8)
        seen += 1
        XCTAssertFalse(text.contains(session), "\(directory)/\(name) reaches the wire")
        XCTAssertFalse(text.contains(network), "\(directory)/\(name) reaches the wire")
        for line in text.split(separator: "\n") where word(child, isIn: String(line)) {
          XCTFail(
            "\(directory)/\(name) names a \(child): \(line.trimmingCharacters(in: .whitespaces))")
        }
      }
    }
    XCTAssertGreaterThan(seen, 5, "no Swift files were read, so this test guards nothing")
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
