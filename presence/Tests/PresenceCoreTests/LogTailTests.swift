// Following the record while gdoc writes it.
//
// The binary appends one line per event and never tells anyone it did. The
// tail reads what arrived, parses newline-terminated records only, keeps the
// trailing partial bytes in a bounded buffer, reopens a file that was
// replaced or removed, and moves to the next month's file on the boundary.
// Every test here drives the synchronous pass, so nothing waits on a clock
// except the one that asserts the watcher delivers at all.

import Foundation
import XCTest

@testable import PresenceCore

final class LogTailTests: XCTestCase {
  private var dir = URL(fileURLWithPath: "/")

  override func setUpWithError() throws {
    dir = URL(fileURLWithPath: NSTemporaryDirectory())
      .appendingPathComponent("presence-tail-" + UUID().uuidString)
    try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
  }

  override func tearDownWithError() throws {
    try? FileManager.default.removeItem(at: dir)
  }

  func testAnUndecodableRecordIsSkippedAndTailingGoesOn() throws {
    try write("2026-10.jsonl", try Fixture.data("malformed"))
    let tail = LogTail(directory: dir, clock: { try! moment("2026-10-17T12:00:00Z") })

    let first = tail.poll()
    XCTAssertEqual(first.events.map(\.kind), [.start, .end, .start], "three records decoded")
    XCTAssertEqual(
      first.skipped, 3, "a line that is not JSON, one that is not an object, and a bad at")
    XCTAssertFalse(first.newerSchema)

    let more = try Fixture.lines("events-v1")[5] + "\n"
    try append("2026-10.jsonl", more)
    let second = tail.poll()
    XCTAssertEqual(second.events.map(\.kind), [.released], "the tail went on from where it stopped")
    XCTAssertEqual(second.skipped, 0)
  }

  func testOnlyNewlineTerminatedRecordsAreParsedAndTheTailIsBounded() throws {
    try write("2026-10.jsonl", try Fixture.data("torn"))
    let tail = LogTail(directory: dir, clock: { try! moment("2026-10-17T12:00:00Z") })

    let batch = tail.poll()
    XCTAssertEqual(
      batch.events.map(\.pid), [4242, 4250],
      "the record a dead writer left half written, with the next one glued on, is one skipped line")
    XCTAssertEqual(batch.skipped, 1)
    XCTAssertEqual(tail.poll().events.count, 0, "the unterminated last line is not a record yet")

    try append("2026-10.jsonl", "ate\",\"via\":\"chat\"}\n")
    XCTAssertEqual(
      tail.poll().events.map(\.command), ["annotate"], "its newline arrived and so did it")

    // Bounded: a partial line longer than the buffer is dropped, and reading
    // picks up again at the next newline rather than growing without end.
    var reader = LineReader(maxLine: 64)
    XCTAssertEqual(reader.take(Data(repeating: UInt8(ascii: "x"), count: 300)).count, 0)
    XCTAssertEqual(reader.dropped, 1, "one oversized partial was let go")
    XCTAssertEqual(reader.pending, 0, "and nothing of it is still held")
    let lines = reader.take(Data("junk\n{\"v\":1}\n".utf8))
    XCTAssertEqual(lines.map { String(decoding: $0, as: UTF8.self) }, [#"{"v":1}"#])
  }

  func testTheTailReadsAppendsWaitsForTheNewlineAndFollowsRotation() throws {
    let lines = try Fixture.lines("events-v1")
    try write("2026-10.jsonl", Data((lines[0] + "\n").utf8))
    let clock = Clock(now: try moment("2026-10-17T12:00:00Z"))
    let tail = LogTail(directory: dir, clock: clock.read)

    XCTAssertEqual(tail.poll().events.map(\.kind), [.start])

    try append("2026-10.jsonl", String(lines[1].prefix(40)))
    XCTAssertEqual(tail.poll().events.count, 0, "half a line is not a record")
    try append("2026-10.jsonl", String(lines[1].dropFirst(40)) + "\n")
    XCTAssertEqual(tail.poll().events.map(\.kind), [.end], "the rest arrived and the line closed")

    clock.set(try moment("2026-11-02T09:00:00Z"))
    try write("2026-11.jsonl", Data((lines[2] + "\n").utf8))
    let rotated = tail.poll()
    XCTAssertEqual(rotated.events.map(\.call), [2], "the new month is followed")
    XCTAssertEqual(tail.files.map(\.lastPathComponent), ["2026-10.jsonl", "2026-11.jsonl"])

    // The watcher itself: an append after start() reaches the callback.
    let arrived = expectation(description: "the watcher delivered an appended record")
    let watching = LogTail(directory: dir, clock: clock.read, rescan: 0.05)
    watching.start { batch in
      if batch.events.contains(where: { $0.kind == .landed }) { arrived.fulfill() }
    }
    defer { watching.stop() }
    try append("2026-11.jsonl", lines[7] + "\n")
    wait(for: [arrived], timeout: 10)
  }

  func testTheTailReopensWhenTheFileIsReplacedOrRemoved() throws {
    let lines = try Fixture.lines("events-v1")
    try write("2026-10.jsonl", Data((lines[0] + "\n").utf8))
    let tail = LogTail(directory: dir, clock: { try! moment("2026-10-17T12:00:00Z") })
    XCTAssertEqual(tail.poll().events.count, 1)

    // Replaced: another file renamed over the path, so the same name is a new
    // inode and the offset the tail held means nothing in it.
    let other = dir.appendingPathComponent("incoming")
    try Data((lines[5] + "\n" + lines[6] + "\n").utf8).write(to: other)
    try FileManager.default.replaceItem(
      at: dir.appendingPathComponent("2026-10.jsonl"), withItemAt: other,
      backupItemName: nil, resultingItemURL: nil)
    XCTAssertEqual(
      tail.poll().events.map(\.kind), [.released, .expired],
      "a replaced file is read from its first byte")

    // Removed: nothing is read, nothing throws, and the next file at that
    // name is read from its own beginning.
    try FileManager.default.removeItem(at: dir.appendingPathComponent("2026-10.jsonl"))
    XCTAssertEqual(tail.poll().events.count, 0)
    try write("2026-10.jsonl", Data((lines[7] + "\n").utf8))
    XCTAssertEqual(tail.poll().events.map(\.kind), [.landed])
  }

  func testTheTailReadsThisAndLastMonthAtLaunchAndRescansAtTheMonthBoundary() throws {
    let lines = try Fixture.lines("events-v1")
    try write("2026-08.jsonl", Data((lines[0] + "\n").utf8))
    try write("2026-09.jsonl", Data((lines[2] + "\n").utf8))
    try write("2026-10.jsonl", Data((lines[4] + "\n").utf8))

    let clock = Clock(now: try moment("2026-10-17T12:00:00Z"))
    let tail = LogTail(directory: dir, clock: clock.read)
    XCTAssertEqual(
      tail.files.map(\.lastPathComponent), ["2026-09.jsonl", "2026-10.jsonl"],
      "this month and the one before it, and August is not reopened")
    XCTAssertEqual(tail.poll().events.map(\.kind), [.start, .held])

    clock.set(try moment("2026-11-01T00:00:00Z"))
    try write("2026-11.jsonl", Data((lines[7] + "\n").utf8))
    XCTAssertEqual(tail.poll().events.map(\.kind), [.landed], "the boundary brought November in")
    XCTAssertEqual(
      tail.files.map(\.lastPathComponent), ["2026-10.jsonl", "2026-11.jsonl"],
      "October is still the previous month, and September is let go")

    XCTAssertEqual(
      MonthFile.name(for: try moment("2026-01-31T23:59:59Z")), "2026-01.jsonl",
      "the month is the event's UTC month, so nothing is ever renamed")
    XCTAssertEqual(
      MonthFile.names(at: try moment("2026-01-01T00:00:00Z")),
      ["2025-12.jsonl", "2026-01.jsonl"], "December is January's previous month")
  }

  private func write(_ name: String, _ data: Data) throws {
    try data.write(to: dir.appendingPathComponent(name))
  }

  private func append(_ name: String, _ text: String) throws {
    let handle = try FileHandle(forWritingTo: dir.appendingPathComponent(name))
    try handle.seekToEnd()
    try handle.write(contentsOf: Data(text.utf8))
    try handle.close()
  }
}

/// A clock a test moves by hand. The tail reads the moment through it, so the
/// month boundary is a line in a test and never a wait.
final class Clock: @unchecked Sendable {
  private let lock = NSLock()
  private var now: Date

  init(now: Date) { self.now = now }

  func read() -> Date {
    lock.lock()
    defer { lock.unlock() }
    return now
  }

  func set(_ when: Date) {
    lock.lock()
    now = when
    lock.unlock()
  }
}
