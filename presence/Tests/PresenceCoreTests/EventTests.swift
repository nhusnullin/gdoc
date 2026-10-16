// What one line of the record decodes to, and what it refuses.
//
// Every line is forgeable data: it arrives from a file any process with the
// owner's rights can write, so the decoder's refusals are the app's only
// defence. A record it cannot read is skipped and never guessed at.

import Foundation
import XCTest

@testable import PresenceCore

final class EventTests: XCTestCase {
  func testAVersionOneLineDecodesAndAnUnknownFieldIsIgnored() throws {
    let lines = try Fixture.lines("events-v1")
    XCTAssertEqual(lines.count, 10, "the shared fixture is ten records")

    let widened = try withUnknownField(lines[0])
    let event = try Event.decode(widened)

    XCTAssertEqual(event.schema, 1)
    XCTAssertEqual(event.at, try moment("2026-10-17T09:15:00Z"))
    XCTAssertEqual(event.gdoc, "v2.13.0")
    XCTAssertEqual(event.pid, 4242)
    XCTAssertEqual(event.run, "9f1cb3a7d2e45608")
    XCTAssertEqual(event.call, 1)
    XCTAssertEqual(event.kind, .start)
    XCTAssertEqual(event.command, "comments")
    XCTAssertEqual(event.via, "terminal")
    XCTAssertEqual(event.doc, "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789")
    XCTAssertTrue(event.wait)
    XCTAssertNil(event.title)
    XCTAssertNil(event.ok)

    let end = try Event.decode(lines[1])
    XCTAssertEqual(end.kind, .end)
    XCTAssertEqual(end.title, "Supplier register")
    XCTAssertEqual(end.ok, true)
    XCTAssertNil(end.verified, "a read writes no verified")
    XCTAssertEqual(end.ms, 540_000)

    let write = try Event.decode(lines[3])
    XCTAssertEqual(write.verified, true)

    let held = try Event.decode(lines[4])
    XCTAssertEqual(held.kind, .held)
    XCTAssertEqual(held.hold, "h_ab12")
    XCTAssertEqual(held.rule, "echo")
    XCTAssertEqual(held.until, try moment("2026-10-17T09:45:00Z"))
    XCTAssertNil(held.call, "a hold is not a numbered call")

    let kinds = try lines.map { try Event.decode($0).kind }
    XCTAssertEqual(
      kinds,
      [.start, .end, .start, .end, .held, .released, .expired, .landed, .start, .end]
    )
  }

  func testAnUnknownVersionIsRefusedAsNewer() throws {
    let lines = try Fixture.lines("future")

    XCTAssertThrowsError(try Event.decode(lines[1])) { error in
      XCTAssertEqual(error as? RecordFault, .newerSchema(2))
    }

    let scan = Event.scan(try Fixture.data("future"))
    XCTAssertEqual(scan.events.count, 2, "the two records this app knows still decode")
    XCTAssertTrue(scan.newerSchema, "one line said gdoc is newer than this app")
    XCTAssertEqual(scan.skipped, 0, "a newer schema is not an undecodable record")
  }

  func testAMissingRequiredFieldSkipsTheRecord() throws {
    let line = try Fixture.lines("events-v1")[0]
    for key in ["v", "at", "pid", "run", "event"] {
      XCTAssertThrowsError(try Event.decode(try without(key, line)), "\(key) is required") {
        error in
        guard case .undecodable = error as? RecordFault else {
          return XCTFail("a record missing \(key) is undecodable, and this said \(error)")
        }
      }
    }

    XCTAssertThrowsError(try Event.decode(try without("command", line)), "a start names a command")

    let held = try Fixture.lines("events-v1")[4]
    XCTAssertThrowsError(try Event.decode(try without("hold", held)), "a hold names itself")

    let released = try Fixture.lines("events-v1")[5]
    XCTAssertThrowsError(try Event.decode(try without("hold", released)))
  }

  /// One key nobody has heard of, in front of every key that matters, so the
  /// test reads a line a newer gdoc could really have written.
  private func withUnknownField(_ line: String) throws -> String {
    let opening = try XCTUnwrap(line.firstIndex(of: "{"))
    let unknown = "{" + #""nobody":"has heard of this","#
    return line.replacingCharacters(in: opening...opening, with: unknown)
  }

  /// The same line with one key taken out, through the only decoder that is
  /// not under test here.
  private func without(_ key: String, _ line: String) throws -> String {
    let data = Data(line.utf8)
    var object = try XCTUnwrap(
      try JSONSerialization.jsonObject(with: data) as? [String: Any],
      "the fixture line is a JSON object"
    )
    XCTAssertNotNil(object.removeValue(forKey: key), "\(key) was in the line to begin with")
    let out = try JSONSerialization.data(withJSONObject: object)
    return String(decoding: out, as: UTF8.self)
  }
}
