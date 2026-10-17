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
    XCTAssertEqual(lines.count, 23, "the shared fixture is twenty-three records")

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
      [
        .start, .end, .start, .end, .held, .released, .expired, .landed, .dropped, .start, .end,
        .start, .end, .start, .end, .start, .step, .step, .step, .step, .end, .start, .end,
      ]
    )
  }

  // The one command that creates a document names it on its end, and every
  // other line carries no made at all, so a row the app draws from one of them
  // has nothing to link to but the doc it was handed.
  func testMadeDecodesAndDefaultsToNil() throws {
    let lines = try Fixture.lines("events-v1")

    let end = try Event.decode(lines[12])
    XCTAssertEqual(end.kind, .end)
    XCTAssertEqual(end.command, "publish")
    XCTAssertEqual(end.made, "1PuBl15h9876543210987654321098765432")
    XCTAssertNil(end.doc, "a publish is handed no document")

    let start = try Event.decode(lines[11])
    XCTAssertNil(start.made, "nothing was created when the run began")
    for index in 0..<lines.count where index != 12 {
      XCTAssertNil(try Event.decode(lines[index]).made, "line \(index) names a document it made")
    }
  }

  // A hold whose sign-in changed leaves the record as dropped, which is a
  // hold ending like the other three: it names its hold, it names no command,
  // and it says nothing about the account, because gdoc puts no account in the
  // record at all.
  func testADroppedLineDecodesAsAHoldEnding() throws {
    let line = try Fixture.lines("events-v1")[8]
    let event = try Event.decode(line)

    XCTAssertEqual(event.kind, .dropped)
    XCTAssertEqual(event.hold, "h_gh78")
    XCTAssertNil(event.command, "a hold ending is not a command")
    XCTAssertNil(event.call, "a hold is not a numbered call")
    XCTAssertNil(event.doc)

    XCTAssertThrowsError(try Event.decode(try without("hold", line)), "a dropped hold names itself")

    // And it closes the hold it names, the way a release does.
    let held = try Event.decode(try Fixture.lines("events-v1")[4])
    let renamed = line.replacingOccurrences(of: "h_gh78", with: "h_ab12")
    let fold = Now.fold(
      [held, try Event.decode(renamed)], now: try moment("2026-10-17T09:20:00Z"),
      alive: { _ in true })
    XCTAssertTrue(fold.holds.isEmpty, "the dropped card is gone from NOW")
  }

  // A run that failed says why in one of gdoc's seven words, and the picture
  // the house style cannot carry is the one cause that brings a line with it.
  // Both are absent everywhere else: a run that worked stopped for no reason,
  // and a start has not stopped at all.
  func testStopAndLineDecodeAndDefaultToNil() throws {
    let lines = try Fixture.lines("events-v1")

    let end = try Event.decode(lines[14])
    XCTAssertEqual(end.kind, .end)
    XCTAssertEqual(end.command, "publish")
    XCTAssertEqual(end.ok, false)
    XCTAssertEqual(end.stop, "svg_picture")
    XCTAssertEqual(end.line, 42)

    let start = try Event.decode(lines[13])
    XCTAssertNil(start.stop, "a run that has begun has not stopped")
    XCTAssertNil(start.line)

    for index in 0..<lines.count where index != 14 {
      let event = try Event.decode(lines[index])
      XCTAssertNil(event.stop, "line \(index) says why it stopped")
      XCTAssertNil(event.line, "line \(index) names a line of a note")
    }

    // A word this app has not heard of is still the word gdoc wrote: nothing
    // here guesses at what an unknown code means, and nothing refuses the line
    // for carrying one.
    let unknown = lines[14].replacingOccurrences(of: "svg_picture", with: "quota")
    XCTAssertEqual(try Event.decode(unknown).stop, "quota")
  }

  // A publish says which of its four parts it is on. The name is one of
  // gdoc's four words, the two numbers are that part's place in the list and
  // the length of the list, and the part that follows the render is the first
  // line of the run that can carry a title. No other line is a step, and a
  // step says nothing about an outcome: the end of the run says that.
  func testAStepLineDecodesWithItsNameIndexAndCount() throws {
    let lines = try Fixture.lines("events-v1")
    let steps = try (16...19).map { try Event.decode(lines[$0]) }

    XCTAssertEqual(steps.map(\.kind), [.step, .step, .step, .step])
    XCTAssertEqual(steps.map(\.step), ["build", "upload", "read_back", "link"])
    XCTAssertEqual(steps.map(\.index), [1, 2, 3, 4])
    XCTAssertEqual(steps.map(\.count), [4, 4, 4, 4])
    for step in steps {
      XCTAssertEqual(step.command, "publish")
      XCTAssertEqual(step.call, 6, "a step belongs to the run it is a part of")
      XCTAssertNil(step.ok, "a step is not an outcome")
      XCTAssertNil(step.ms)
      XCTAssertNil(step.made)
    }
    XCTAssertNil(steps[0].title, "the render had not read the note when the build began")
    XCTAssertEqual(steps[1].title, "Supplier register")

    // A step names its command, as a start and an end do.
    XCTAssertThrowsError(
      try Event.decode(try without("command", lines[16])), "a step names its command")

    // And every other line of the fixture is no step at all.
    for index in 0..<lines.count where !(16...19).contains(index) {
      let event = try Event.decode(lines[index])
      XCTAssertNil(event.step, "line \(index) names a step")
      XCTAssertNil(event.index, "line \(index) carries a place in a list")
      XCTAssertNil(event.count, "line \(index) carries a count")
    }
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

  // A schema under the one this app knows is not a gdoc that is newer. It is
  // a line nobody can place, so it is skipped like any other, and the one
  // notice that says gdoc is ahead of the app is kept for when that is true.
  func testAVersionUnderTheKnownOneIsSkippedRatherThanCalledNewer() throws {
    let line = try Fixture.lines("events-v1")[0]
      .replacingOccurrences(of: "\"v\":1", with: "\"v\":0")

    XCTAssertThrowsError(try Event.decode(line)) { error in
      guard case .undecodable = error as? RecordFault else {
        return XCTFail("a v under the known one is undecodable, and this said \(error)")
      }
    }

    var records = Records()
    records.add(Data(line.utf8))
    XCTAssertTrue(records.events.isEmpty, "the line is not a record")
    XCTAssertEqual(records.skipped, 1)
    XCTAssertFalse(records.newerSchema, "nothing here says gdoc is newer than this app")
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

  // Who ran a command is one of gdoc's three words, and a run Claude Code
  // started says code. The wait that run was watching was stopped by a person,
  // which its end says and no other line does: a wait that ran to its deadline
  // writes nothing there, so the app has a fact and never a flag with two
  // values.
  func testViaCodeAndInterruptedDecode() throws {
    let lines = try Fixture.lines("events-v1")

    let start = try Event.decode(lines[21])
    XCTAssertEqual(start.kind, .start)
    XCTAssertEqual(start.command, "comments")
    XCTAssertEqual(start.via, "code")
    XCTAssertTrue(start.wait)
    XCTAssertFalse(start.interrupted, "a run that has begun has not been stopped")

    let end = try Event.decode(lines[22])
    XCTAssertEqual(end.kind, .end)
    XCTAssertEqual(end.via, "code")
    XCTAssertTrue(end.interrupted)
    XCTAssertEqual(end.ok, true, "a wait a person stopped is still an answer")

    for index in 0..<lines.count where index != 22 {
      XCTAssertFalse(
        try Event.decode(lines[index]).interrupted, "line \(index) says a person stopped it")
    }

    // The three words are gdoc's, and a word this version has not heard of is
    // kept as the word gdoc wrote: nothing here guesses at what it means.
    XCTAssertEqual(
      Set(try lines.compactMap { try Event.decode($0).via }), ["terminal", "chat", "code"])
    let unknown = lines[21].replacingOccurrences(of: "\"via\":\"code\"", with: "\"via\":\"cron\"")
    XCTAssertEqual(try Event.decode(unknown).via, "cron")
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
