// A live session: the one thing the record never writes down, stitched out of
// the waits it does write.
//
// live-day.jsonl is one day on two documents: a watch on Chapter 5 from 11:10
// to 16:55 UTC, twelve waits long, with four replies and two proposals
// between them and one poll that failed in the middle, and a watch on the
// Northwind document that is still in flight at 17:30. The screens file's
// TODAY row for that first watch reads "5 h 45 min · answered 4 · proposed 2",
// which is the same day in the panel's own zone.
//
// Every rule here is a product decision from the M24 specification's table.
// The moment is injected, so no test reads this machine's clock.

import Foundation
import XCTest

@testable import PresenceCore

final class LiveSessionTests: XCTestCase {
  /// 19:30 in the panel's zone: after the Chapter 5 watch stopped and while
  /// the Northwind one is still waiting.
  private func halfPastSeven() throws -> Date { try moment("2026-10-17T17:30:00Z") }

  private func day() throws -> [LiveSession] {
    let scan = Event.scan(try Fixture.data("live-day"))
    XCTAssertEqual(scan.skipped, 0, "the day fixture is clean")
    return Live.sessions(scan.events, now: try halfPastSeven(), alive: { _ in true })
  }

  func testWaitsWithinThreeMinutesAreOneSession() throws {
    XCTAssertEqual(Live.gap, 3 * 60, "three minutes, as a duration")

    let sessions = try day()
    XCTAssertEqual(sessions.count, 2, "one session per document, oldest first")
    XCTAssertEqual(sessions.map(\.doc), [chapterFive, northwind])

    let five = sessions[0]
    XCTAssertEqual(five.since, try moment("2026-10-17T11:10:00Z"), "the first wait's start")
    XCTAssertEqual(
      five.ended, try moment("2026-10-17T16:55:00Z"),
      "the last wait's end: twelve waits with gaps under three minutes are one watch")
    XCTAssertFalse(five.isOpen)
    XCTAssertEqual(five.title, "Chapter 5. Commercial Model: the storyline")

    // The gap is measured from one wait's end to the next wait's start.
    let near = [
      wait(from: "2026-10-17T17:00:00Z", to: "2026-10-17T17:10:00Z", run: "aaaa"),
      wait(from: "2026-10-17T17:12:59Z", to: nil, run: "bbbb"),
    ].flatMap { $0 }
    XCTAssertEqual(
      Live.sessions(near, now: try halfPastSeven(), alive: { _ in true }).count, 1,
      "two minutes and 59 seconds")

    let far = [
      wait(from: "2026-10-17T17:00:00Z", to: "2026-10-17T17:10:00Z", run: "aaaa"),
      wait(from: "2026-10-17T17:13:00Z", to: nil, run: "bbbb"),
    ].flatMap { $0 }
    let split = Live.sessions(far, now: try halfPastSeven(), alive: { _ in true })
    XCTAssertEqual(split.count, 2, "three minutes exactly is a second watch")
    XCTAssertEqual(
      split.map(\.since),
      [
        try moment("2026-10-17T17:00:00Z"), try moment("2026-10-17T17:13:00Z"),
      ])

    // One document's waits never join another's, however close they are.
    let apart =
      wait(from: "2026-10-17T17:00:00Z", to: "2026-10-17T17:10:00Z", run: "aaaa")
      + wait(from: "2026-10-17T17:10:10Z", to: nil, run: "bbbb", doc: northwind)
    XCTAssertEqual(
      Live.sessions(apart, now: try halfPastSeven(), alive: { _ in true }).map(\.doc),
      [chapterFive, northwind])
  }

  func testRepliedAndProposedCountTheDocumentsOkEnds() throws {
    let five = try day()[0]
    XCTAssertEqual(five.replied, 4, "the four replies between its waits")
    XCTAssertEqual(five.proposed, 2, "and the two proposals")

    let northwindSession = try day()[1]
    XCTAssertEqual(northwindSession.replied, 0)
    XCTAssertEqual(northwindSession.proposed, 0)

    // A write that failed is not a reply the session counts, nor is one on
    // another document, nor one before the watch began.
    let before = write(at: "2026-10-17T16:59:00Z", command: "reply", ok: true)
    let failed = write(at: "2026-10-17T17:05:00Z", command: "reply", ok: false)
    let elsewhere = write(at: "2026-10-17T17:05:00Z", command: "reply", ok: true, doc: northwind)
    let counted = write(at: "2026-10-17T17:05:00Z", command: "propose", ok: true)
    let events =
      wait(from: "2026-10-17T17:00:00Z", to: nil, run: "aaaa")
      + [before, failed, elsewhere, counted]
    let session = try XCTUnwrap(
      Live.sessions(events, now: try halfPastSeven(), alive: { _ in true }).first)
    XCTAssertEqual(session.replied, 0, "a failed reply answered nobody")
    XCTAssertEqual(session.proposed, 1)
  }

  /// The shape the Go writer really writes: `wait` on the start, and nothing
  /// about waiting on the end. An end held to that field pairs with nothing,
  /// and then every poll of a watch is its own open session for ever.
  func testAnEndWithoutTheWaitFieldStillClosesItsStart() throws {
    let started = try moment("2026-10-17T17:00:00Z")
    let stopped = try moment("2026-10-17T17:02:00Z")
    let line = { (kind: EventKind, at: Date, ok: Bool?) in
      Event(
        schema: 1, at: at, gdoc: "v2.14.0", pid: 7100, run: "ffff", call: 1, kind: kind,
        command: "comments", via: "terminal", doc: self.chapterFive, wait: kind == .start,
        title: nil, ok: ok, verified: nil, ms: nil, hold: nil, rule: nil, until: nil)
    }
    let events = [line(.start, started, nil), line(.end, stopped, true)]

    let open = try XCTUnwrap(
      Live.sessions(events, now: try moment("2026-10-17T17:03:00Z"), alive: { _ in true })
        .first)
    XCTAssertNil(open.flight, "the end paired, so no poll is in flight")
    XCTAssertTrue(open.isOpen, "still inside the three-minute gap")

    let later = Live.sessions(
      events, now: try moment("2026-10-17T17:10:00Z"), alive: { _ in true })
    XCTAssertEqual(later.count, 1, "one wait is one session, not one per half of it")
    XCTAssertEqual(later[0].ended, stopped, "and it closed at the end the record wrote")
  }

  /// A wait a person stopped is the end of the watch, and not three minutes
  /// spent waiting to see whether another poll follows. No poll ever does:
  /// the skill reads `waited.interrupted`, prints its totals and stops.
  ///
  /// Without this the panel kept a watch somebody had just stopped live for
  /// three minutes, with no process to signal, so Stop was greyed with "This
  /// watch is not one gdoc presence can stop" over the thing that had worked,
  /// the menu bar still said live, and TODAY wrote no row for the stop.
  func testAnInterruptedWaitEndsTheWatchAtOnce() throws {
    let signalled = wait(
      from: "2026-10-17T17:00:00Z", to: "2026-10-17T17:04:00Z", run: "aaaa", interrupted: true)
    let stopped = try XCTUnwrap(
      Live.sessions(signalled, now: try moment("2026-10-17T17:04:10Z"), alive: { _ in true })
        .first)
    XCTAssertFalse(stopped.isOpen, "ten seconds after the signal the watch is over")
    XCTAssertEqual(
      stopped.ended, try moment("2026-10-17T17:04:00Z"), "at the end the record wrote")
    XCTAssertTrue(stopped.stoppedByYou)
    XCTAssertNil(stopped.flight, "and there is no process left to signal")

    // The three minutes are still what closes a watch nobody stopped: there
    // the next poll really may be a second away.
    let ran = wait(from: "2026-10-17T17:00:00Z", to: "2026-10-17T17:04:00Z", run: "bbbb")
    let between = try XCTUnwrap(
      Live.sessions(ran, now: try moment("2026-10-17T17:04:10Z"), alive: { _ in true }).first)
    XCTAssertTrue(between.isOpen, "a poll that ended by itself is inside the gap")
    XCTAssertFalse(between.stoppedByYou)
  }

  func testTheLastWriteTimeIsTheQuietAnchor() throws {
    let five = try day()[0]
    XCTAssertEqual(
      five.lastWrite, try moment("2026-10-17T16:40:20Z"), "the last reply, not the last poll")
    XCTAssertEqual(five.quietSince, five.lastWrite, "quiet is measured from the last write")

    let northwindSession = try day()[1]
    XCTAssertNil(northwindSession.lastWrite, "nothing was written on this one yet")
    XCTAssertEqual(
      northwindSession.quietSince, northwindSession.since,
      "with no write, the watch has been quiet since it began")

    // Any of the seven writes anchors it, not only a reply or a proposal.
    let events =
      wait(from: "2026-10-17T17:00:00Z", to: nil, run: "aaaa")
      + [
        write(at: "2026-10-17T17:05:00Z", command: "reply", ok: true),
        write(at: "2026-10-17T17:09:00Z", command: "annotate", ok: true),
        write(at: "2026-10-17T17:12:00Z", command: "read", ok: true),
      ]
    let session = try XCTUnwrap(
      Live.sessions(events, now: try halfPastSeven(), alive: { _ in true }).first)
    XCTAssertEqual(
      session.lastWrite, try moment("2026-10-17T17:09:00Z"), "a read changed nothing")
  }

  func testAFailedPollIsAFact() throws {
    let five = try day()[0]
    XCTAssertNil(
      five.lastPollFailed, "the poll that failed at 14:20 was answered by the next one")

    let events = wait(
      from: "2026-10-17T17:00:00Z", to: "2026-10-17T17:04:00Z", run: "aaaa", ok: false)
    let fresh = try XCTUnwrap(
      Live.sessions(events, now: try moment("2026-10-17T17:05:00Z"), alive: { _ in true })
        .first)
    XCTAssertEqual(
      fresh.lastPollFailed, try moment("2026-10-17T17:04:00Z"), "when it failed, and nothing else")
    XCTAssertTrue(fresh.isOpen, "the next poll is still inside the gap")

    let cold = try XCTUnwrap(
      Live.sessions(events, now: try moment("2026-10-17T17:08:00Z"), alive: { _ in true })
        .first)
    XCTAssertFalse(cold.isOpen, "no poll followed it inside three minutes")
    XCTAssertEqual(
      cold.lastPollFailed, try moment("2026-10-17T17:04:00Z"), "the fact stays after the watch")
    XCTAssertEqual(cold.ended, try moment("2026-10-17T17:04:00Z"))
  }

  /// A poll Google refused the token on ends the watch where it ended, so
  /// NOW, DOCUMENTS and TODAY say the same thing about it on the one screen
  /// where it happens. "Paused" would be the panel saying it will carry on,
  /// which nobody has measured.
  func testAPollGoogleRefusedTheTokenOnEndsTheWatch() throws {
    XCTAssertEqual(Live.signInStop, "auth", "gdoc's own word for it")

    let refused = wait(
      from: "2026-10-17T17:00:00Z", to: "2026-10-17T17:04:00Z", run: "aaaa", ok: false,
      stop: "auth")
    let session = try XCTUnwrap(
      Live.sessions(refused, now: try moment("2026-10-17T17:05:00Z"), alive: { _ in true })
        .first)
    XCTAssertEqual(
      session.ended, try moment("2026-10-17T17:04:00Z"),
      "the watch is over at the poll the token was refused on")
    XCTAssertFalse(session.isOpen, "one minute on, and it is not waiting out the gap")
    XCTAssertTrue(session.stoppedBySignIn)

    // Every other failure is a watch in trouble and not a watch over: the
    // next poll may be a second away.
    let network = wait(
      from: "2026-10-17T17:00:00Z", to: "2026-10-17T17:04:00Z", run: "bbbb", ok: false,
      stop: "network")
    let waiting = try XCTUnwrap(
      Live.sessions(network, now: try moment("2026-10-17T17:05:00Z"), alive: { _ in true })
        .first)
    XCTAssertTrue(waiting.isOpen, "a poll that could not reach Google is still a watch")
    XCTAssertFalse(waiting.stoppedBySignIn)
  }

  /// A watch Google refused the token on is waiting on the person only
  /// while nothing has got through since. The record says when something
  /// did: an end that worked, or one Google itself answered 404 to, is a
  /// token Google took, and from there that watch is the day's log and not
  /// a row NOW keeps. Without it every watch a refusal ever ended would
  /// come back the next time a token stopped working.
  func testAWatchTheSignInOutlivedLeavesNow() throws {
    let refused = wait(
      from: "2026-10-17T09:00:00Z", to: "2026-10-17T09:04:00Z", run: "aaaa", ok: false,
      stop: "auth")
    let at = try moment("2026-10-17T17:05:00Z")

    let alone = try XCTUnwrap(Live.sessions(refused, now: at, alive: { _ in true }).first)
    XCTAssertTrue(alone.stoppedBySignIn)
    XCTAssertFalse(alone.reachedSince, "nothing reached Google after it")
    XCTAssertTrue(alone.waitsForTheSignIn, "so NOW keeps the row")

    // The person signed in again and a reply went out. The watch is over
    // and nobody has started another, which is the day's log.
    let worked = refused + [write(at: "2026-10-17T10:00:00Z", command: "reply", ok: true)]
    let outlived = try XCTUnwrap(Live.sessions(worked, now: at, alive: { _ in true }).first)
    XCTAssertTrue(outlived.stoppedBySignIn, "the record still says why it stopped")
    XCTAssertTrue(outlived.reachedSince)
    XCTAssertFalse(outlived.waitsForTheSignIn, "and it is no longer waiting on anybody")

    // Two commands say nothing about Google: a logout opens no connection
    // and an update reads GitHub.
    for command in ["auth logout", "update"] {
      let quiet = refused + [write(at: "2026-10-17T10:00:00Z", command: command, ok: true)]
      let still = try XCTUnwrap(Live.sessions(quiet, now: at, alive: { _ in true }).first)
      XCTAssertFalse(still.reachedSince, "\(command) is no word about the token")
      XCTAssertTrue(still.waitsForTheSignIn)
    }

    // A watch still going says nothing of the kind.
    let open = wait(from: "2026-10-17T17:00:00Z", to: nil, run: "bbbb")
    let going = try XCTUnwrap(Live.sessions(open, now: at, alive: { _ in true }).first)
    XCTAssertTrue(going.isOpen)
    XCTAssertFalse(going.reachedSince, "there is no after yet")
    XCTAssertFalse(going.waitsForTheSignIn)
  }

  func testTheSessionKeepsPidRunAndCallForStop() throws {
    let sessions = try day()

    let flight = try XCTUnwrap(sessions[1].flight, "the wait in flight is the one Stop signals")
    XCTAssertEqual(flight.pid, 6022)
    XCTAssertEqual(flight.run, "2000000000000022")
    XCTAssertEqual(flight.call, 1)
    XCTAssertEqual(flight.started, try moment("2026-10-17T17:26:00Z"))
    XCTAssertTrue(sessions[1].isOpen)
    XCTAssertNil(sessions[1].ended)

    XCTAssertNil(
      sessions[0].flight, "a watch that stopped has no process, so there is nothing to signal")

    // Between two waits a session is open and still has no pid.
    let between = wait(from: "2026-10-17T17:00:00Z", to: "2026-10-17T17:29:00Z", run: "aaaa")
    let session = try XCTUnwrap(
      Live.sessions(between, now: try halfPastSeven(), alive: { _ in true }).first)
    XCTAssertTrue(session.isOpen, "a minute after the last poll, inside the gap")
    XCTAssertNil(session.flight, "and between waits there is no process to signal")
  }

  /// A wait killed outright writes no end, so the record alone says the watch
  /// is still going for as long as it keeps the line. The kernel's answer is
  /// the bound: no pid, no watch, and the watch ended where that wait began.
  func testAWaitWhoseProcessWentAwayEndsTheWatch() throws {
    let scan = Event.scan(try Fixture.data("live-day"))
    let flying: Int32 = 6022

    let open = try XCTUnwrap(
      Live.sessions(scan.events, now: try halfPastSeven(), alive: { _ in true }).last)
    XCTAssertEqual(open.doc, northwind)
    XCTAssertTrue(open.isOpen, "the fixture's last wait is in flight")

    let gone = try XCTUnwrap(
      Live.sessions(scan.events, now: try halfPastSeven(), alive: { $0 != flying }).last)
    XCTAssertEqual(gone.doc, northwind)
    XCTAssertFalse(gone.isOpen, "a wait whose process is not there is no watch")
    XCTAssertEqual(
      gone.ended, try moment("2026-10-17T17:26:00Z"),
      "the watch ends where its last wait began, which is the last fact about it")
    XCTAssertNil(gone.flight, "there is no pid left to signal")

    // The one in flight is the only poll the kernel is asked about: a watch
    // whose earlier polls all ended is closed by its own ends.
    let stopped = try XCTUnwrap(
      Live.sessions(scan.events, now: try halfPastSeven(), alive: { _ in false }).first)
    XCTAssertEqual(stopped.ended, try moment("2026-10-17T16:55:00Z"), "its last end stands")
  }

  func testNoSessionSaysWaitingForTheAuthor() throws {
    let reading = [write(at: "2026-10-17T17:00:00Z", command: "read", ok: true)]
    XCTAssertTrue(
      Live.sessions(reading, now: try halfPastSeven(), alive: { _ in true }).isEmpty,
      "a read is no watch")
    XCTAssertTrue(Live.sessions([], now: try halfPastSeven(), alive: { _ in true }).isEmpty)

    // The specification leaves the thread's own state out: gdoc prints no
    // field for it and the app may not guess one. The scan is over Sources
    // only, which is why this file may spell the words.
    let phrase = "waiting for the author"
    let sources = packageRoot().appendingPathComponent("Sources")
    let walker = try XCTUnwrap(
      FileManager.default.enumerator(atPath: sources.path), "Sources is not there")
    var seen = 0
    for case let name as String in walker where name.hasSuffix(".swift") {
      let text = try String(contentsOf: sources.appendingPathComponent(name), encoding: .utf8)
      seen += 1
      XCTAssertFalse(
        text.lowercased().contains(phrase), "Sources/\(name) judges a thread the record never does")
    }
    XCTAssertGreaterThan(seen, 5, "no Swift files were read, so this test guards nothing")
  }

  // MARK: - The pieces a test states

  private let chapterFive = "1Ch5AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private let northwind = "1NorAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"

  /// This file is Tests/PresenceCoreTests/LiveSessionTests.swift, so the
  /// package is three directories up from it.
  private func packageRoot() -> URL {
    URL(fileURLWithPath: #filePath)
      .deletingLastPathComponent()
      .deletingLastPathComponent()
      .deletingLastPathComponent()
  }

  /// One `comments --wait` call: its start, and its end where it has one.
  private func wait(
    from: String, to: String?, run: String, ok: Bool = true, doc: String? = nil,
    interrupted: Bool = false, stop: String? = nil
  ) -> [Event] {
    let on = doc ?? chapterFive
    var events = [
      event(at: from, kind: .start, command: "comments", run: run, doc: on, wait: true)
    ]
    if let to {
      events.append(
        event(
          at: to, kind: .end, command: "comments", run: run, doc: on, ok: ok,
          interrupted: interrupted, stop: stop))
    }
    return events
  }

  /// One command that ended. The run is made of every field the line names,
  /// because the fold reads a line with the run and call of another as the
  /// same line written twice.
  private func write(at: String, command: String, ok: Bool, doc: String? = nil) -> Event {
    let on = doc ?? chapterFive
    return event(
      at: at, kind: .end, command: command, run: "\(at)|\(command)|\(on)|\(ok)",
      doc: on, ok: ok)
  }

  private func event(
    at text: String, kind: EventKind, command: String, run: String, doc: String?,
    wait: Bool = false, ok: Bool? = nil, interrupted: Bool = false, stop: String? = nil
  ) -> Event {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return Event(
      schema: 1, at: formatter.date(from: text) ?? Date(), gdoc: "v2.14.0", pid: 7000, run: run,
      call: 1, kind: kind, command: command, via: "terminal", doc: doc, wait: wait,
      interrupted: interrupted, title: nil,
      ok: ok, verified: nil, stop: stop, ms: ok == nil ? nil : 1200, hold: nil, rule: nil,
      until: nil)
  }
}
