// The fixtures every test here reads, and the two helpers that find them.
//
// events-v1.jsonl is the Go writer's own output, held byte for byte equal to
// go/internal/presence/testdata/events-v1.jsonl by a Go test, so a field that
// changes meaning in one language fails the other language's suite.

import Foundation
import XCTest

enum Fixture {
  /// The bytes of one fixture, as the app would read them off the disk.
  static func data(_ name: String) throws -> Data {
    let url = try XCTUnwrap(
      Bundle.module.url(forResource: name, withExtension: "jsonl", subdirectory: "Fixtures"),
      "fixture \(name).jsonl is not in the test bundle"
    )
    return try Data(contentsOf: url)
  }

  /// The newline-terminated lines of one fixture, as strings. A trailing
  /// partial record is one of them, because torn.jsonl ends on one.
  static func lines(_ name: String) throws -> [String] {
    let text = String(decoding: try data(name), as: UTF8.self)
    return text.split(separator: "\n", omittingEmptySubsequences: true).map(String.init)
  }
}

/// A date written the way the record writes one, so a test states its moment
/// as a literal and never as an offset from the machine's clock.
func moment(_ text: String) throws -> Date {
  let formatter = ISO8601DateFormatter()
  formatter.formatOptions = [.withInternetDateTime]
  return try XCTUnwrap(formatter.date(from: text), "\(text) is not a moment")
}

/// The calendar every fold test folds in: one zone, written out, because the
/// local day is the thing under test and the machine's own zone is not it.
func amsterdam() throws -> Calendar {
  var calendar = Calendar(identifier: .gregorian)
  calendar.timeZone = try XCTUnwrap(TimeZone(identifier: "Europe/Amsterdam"))
  return calendar
}
