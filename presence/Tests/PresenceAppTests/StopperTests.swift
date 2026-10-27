// Stopping a live watch, with no real process and no real signal.
//
// The one thing in this app that reaches outside it is the signal a live
// row's Stop sends, so every test here injects both halves of the kernel: the
// facts about a pid, and the sending itself. Nothing in this file signals
// anything, and no test needs a gdoc on the machine.
//
// The table is the M24 specification's: one SIGINT, never repeated, only to a
// pid whose wait passes every check in PresenceCore/ProcessCheck. What a
// refusal looks like to a person is held here too, because the sentence and
// the greyed icon are the whole of what the panel says about it.

import Foundation
import PresenceCore
import PresenceUI
import XCTest

@testable import PresenceApp

/// On the main actor for the two readings of the live row's own view, which
/// is where every view in this package lives.
@MainActor
final class StopperTests: XCTestCase {
  private static let doc = "1Ch5AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private static let other = "1NorAaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  private static let gdoc = "/Users/nail/.local/bin/gdoc"
  private static let title = "Chapter 5. Commercial Model: the storyline"

  // MARK: the signal

  func testStopSendsOneSIGINTOnlyAfterTheProcessCheck() throws {
    let at = try moment("2026-10-17T12:00:00Z")
    let desk = Desk()
    let stopper = armed(desk)

    XCTAssertEqual(Stopper.signal, SIGINT, "the one signal the app sends")
    XCTAssertEqual(Stopper.signal, 2, "which is 2, stated as a literal")

    XCTAssertEqual(
      stopper.standing(doc: Self.doc, in: waiting(at: at)), .ready,
      "every check holds, so Stop is live")
    XCTAssertEqual(
      stopper.stop(doc: Self.doc, in: waiting(at: at)), .signalled,
      "and the signal goes to the pid the record named")
    XCTAssertEqual(desk.sent, [Sent(pid: 4311, signal: SIGINT)], "one signal, and one only")

    // Every refusal, each on its own, and none of them sends anything.
    let table: [(String, StopStanding, [Event], Desk.Facts)] = [
      (
        "the path is another program", .refused(.notGdoc), waiting(at: at),
        .init(path: "/usr/bin/vim")
      ),
      (
        "the kernel named no path, which a dead-but-unreaped pid also gives",
        .refused(.pathUnknown), waiting(at: at), .init(path: nil)
      ),
      ("the pid belongs to another user", .refused(.notOurUser), waiting(at: at), .init(uid: 502)),
      ("the pid is gone", .refused(.gone), waiting(at: at), .init(isAlive: false)),
      (
        "it started eleven seconds before the event", .refused(.outsideTheWindow),
        waiting(at: at), .init(started: at.addingTimeInterval(-11))
      ),
      (
        "the kernel named no start time", .refused(.startUnknown), waiting(at: at),
        .init(started: nil)
      ),
      ("the run says chat", .refused(.fromChat), waiting(at: at, via: "chat"), .init()),
      (
        "the run is another command", .refused(.notAWait), waiting(at: at, command: "propose"),
        .init()
      ),
      ("the line is not a wait", .refused(.notAWait), waiting(at: at, isWait: false), .init()),
      (
        "the wait already has its end", .refused(.ended),
        waiting(at: at) + [end(at: at.addingTimeInterval(4))], .init()
      ),
      ("there is no wait on that document at all", .refused(.notAWait), [], .init()),
    ]

    for (reason, expected, events, facts) in table {
      let room = Desk(facts: facts)
      let asked = armed(room)
      XCTAssertEqual(asked.standing(doc: Self.doc, in: events), expected, reason)
      XCTAssertEqual(asked.stop(doc: Self.doc, in: events), expected, reason)
      XCTAssertTrue(room.sent.isEmpty, "\(reason), so nothing is sent")
    }

    // An app that never found a gdoc has no path to compare the pid's
    // against, so it signals nothing either.
    let blind = Desk()
    let unarmed = Stopper(
      gdoc: nil, user: 501, inspect: { blind.facts(for: $0) }, send: { blind.send($0, $1) },
      resolve: { $0 })
    XCTAssertEqual(unarmed.standing(doc: Self.doc, in: waiting(at: at)), .noGdoc)
    XCTAssertEqual(unarmed.stop(doc: Self.doc, in: waiting(at: at)), .noGdoc)
    XCTAssertTrue(blind.sent.isEmpty, "no binary, no signal")
  }

  func testStopNeverSendsAnotherSignal() throws {
    let at = try moment("2026-10-17T12:00:00Z")
    let desk = Desk()
    let stopper = armed(desk)
    let events = waiting(at: at)

    XCTAssertEqual(stopper.stop(doc: Self.doc, in: events), .signalled)
    XCTAssertEqual(
      stopper.standing(doc: Self.doc, in: events), .signalled,
      "the wait has been signalled and its end has not arrived")
    XCTAssertEqual(
      stopper.stop(doc: Self.doc, in: events), .signalled,
      "a second press is the same standing, not a second signal")
    XCTAssertEqual(stopper.stop(doc: Self.doc, in: events), .signalled)
    XCTAssertEqual(desk.sent.count, 1, "one wait, one SIGINT, however many presses")

    // The memory is of one wait and not of one document: the next poll on the
    // same document is a process of its own, and stoppable.
    let next =
      events + [end(at: at.addingTimeInterval(6))]
      + waiting(at: at.addingTimeInterval(8), run: "b2c3d4e5f6071829", pid: 4322)
    XCTAssertEqual(stopper.standing(doc: Self.doc, in: next), .ready, "a new wait, not the old one")
    XCTAssertEqual(stopper.stop(doc: Self.doc, in: next), .signalled)
    XCTAssertEqual(
      desk.sent, [Sent(pid: 4311, signal: SIGINT), Sent(pid: 4322, signal: SIGINT)],
      "one signal per wait, to the pid that wait named")
  }

  /// The Go writer puts `wait` on the start and nothing about waiting on the
  /// end, so an end held to that field would never be found: the wait would
  /// look open for ever and a signalled one would keep its ring for ever.
  func testAnEndWithoutTheWaitFieldEndsTheWait() throws {
    let at = try moment("2026-10-17T12:00:00Z")
    let closed = waiting(at: at) + [end(at: at.addingTimeInterval(6))]

    let found = try XCTUnwrap(Stopper.latestWait(doc: Self.doc, in: closed))
    XCTAssertTrue(found.1, "the end paired with its start")
    XCTAssertTrue(Stopper.openWaits(in: closed).isEmpty, "so no wait is open")

    // And the memory of a signal is let go once that end is in the record.
    let desk = Desk()
    let stopper = armed(desk)
    XCTAssertEqual(stopper.stop(doc: Self.doc, in: waiting(at: at)), .signalled)
    XCTAssertEqual(
      stopper.pendingDocs(in: waiting(at: at)), [Self.doc], "the ring is on until the end")
    XCTAssertTrue(stopper.pendingDocs(in: closed).isEmpty, "and off once the end arrived")
  }

  /// A `comments --follow` holds one process open for half an hour, so the
  /// wait Stop reaches is 29 minutes old instead of two. Nothing in the
  /// check is a function of now: it compares the kernel's start time against
  /// the event's own, so an old wait passes every check a fresh one does and
  /// the row the panel draws for it still offers Stop.
  func testStopReachesALongWaitAfterTheProcessCheck() throws {
    let at = try moment("2026-10-17T12:00:00Z")
    let halfAnHourOn = at.addingTimeInterval(29 * 60)
    let desk = Desk()
    let stopper = armed(desk)
    let events = waiting(at: at)

    XCTAssertEqual(
      stopper.standing(doc: Self.doc, in: events), .ready,
      "a wait 29 minutes old is the same wait the kernel agreed about")
    XCTAssertEqual(stopper.unstoppable(in: events), [], "so no row is greyed")

    let live = try panel(events, at: halfAnHourOn)
    guard case .live(_, _, let offered, let pending, _) = try first(of: live) else {
      return XCTFail("a follow still running draws a live row")
    }
    XCTAssertEqual(offered?.state, .live, "and Stop can be pressed half an hour in")
    XCTAssertFalse(pending, "nothing has been signalled yet")

    XCTAssertEqual(stopper.stop(doc: Self.doc, in: events), .signalled)
    XCTAssertEqual(
      desk.sent, [Sent(pid: 4311, signal: SIGINT)],
      "one SIGINT, to the pid the record named, and only after the check")
    XCTAssertEqual(
      stopper.pendingDocs(in: events), [Self.doc], "the ring is on until the follow's end")
  }

  // MARK: what the panel says

  func testAFailedCheckGreysStopWithItsTooltip() throws {
    let at = try moment("2026-10-17T12:00:00Z")
    let events = waiting(at: at)

    let holds = armed(Desk())
    XCTAssertEqual(
      holds.unstoppable(in: events), [], "the check holds, so no row is greyed")

    let refuses = armed(Desk(facts: .init(path: "/usr/bin/vim")))
    XCTAssertEqual(
      refuses.unstoppable(in: events), [Self.doc],
      "the pid is not a gdoc, so this document's Stop is greyed")
    XCTAssertEqual(
      refuses.standing(doc: Self.doc, in: events), .refused(.notGdoc),
      "and the reason stays a fact for a test, never a sentence for a person")

    XCTAssertEqual(
      ProcessCheck.refusalLine, "This watch is not one gdoc presence can stop",
      "one sentence for every refusal")
    XCTAssertEqual(
      Tips.cannotStop, "This watch is not one gdoc presence can stop",
      "and the panel's tooltip is that same sentence")

    // The panel's own reading of the fact the stopper hands it.
    let greyed = try panel(events, at: at, unstoppable: refuses.unstoppable(in: events))
    guard case .live(_, _, let stop, let pending, _) = try first(of: greyed) else {
      return XCTFail("an open watch draws a live row")
    }
    XCTAssertEqual(stop?.state, .off(Tips.cannotStop), "the icon is off, with the one sentence")
    XCTAssertFalse(pending, "nothing was signalled, so there is no ring")

    let live = try panel(events, at: at, unstoppable: [])
    guard case .live(_, _, let offered, _, _) = try first(of: live) else {
      return XCTFail("an open watch draws a live row")
    }
    XCTAssertEqual(offered?.state, .live, "the check holds, so Stop can be pressed")
  }

  func testTheCardShowsARingUntilTheEndArrives() throws {
    let at = try moment("2026-10-17T12:00:00Z")
    let desk = Desk()
    let stopper = armed(desk)
    let events = waiting(at: at)

    XCTAssertEqual(stopper.pendingDocs(in: events), [], "nothing is pending before a press")
    XCTAssertEqual(stopper.stop(doc: Self.doc, in: events), .signalled)
    XCTAssertEqual(
      stopper.pendingDocs(in: events), [Self.doc],
      "the SIGINT is in flight, so the row waits for the end")

    let ringing = try panel(
      events, at: at.addingTimeInterval(2), pendingStop: stopper.pendingDocs(in: events))
    guard case .live(_, _, let stop, let pending, _) = try first(of: ringing) else {
      return XCTFail("an open watch draws a live row")
    }
    XCTAssertTrue(pending, "a ring, until the end arrives")
    XCTAssertNil(stop, "and no second Stop to press while it is in flight")
    XCTAssertEqual(LiveRow.icons(pending: true), [], "the icon is the ring's place")
    XCTAssertTrue(LiveRow.showsRing(pending: true), "and the ring stands where it was")

    // The end arrives. The wait is paired, so nothing is pending any more.
    let ended = events + [end(at: at.addingTimeInterval(5))]
    XCTAssertEqual(
      stopper.pendingDocs(in: ended), [], "the end is written, so the ring goes")
    XCTAssertEqual(stopper.standing(doc: Self.doc, in: ended), .refused(.ended))

    let after = try panel(
      ended, at: at.addingTimeInterval(7), pendingStop: stopper.pendingDocs(in: ended))
    guard case .live(_, _, let stopAfter, let pendingAfter, _) = try first(of: after) else {
      return XCTFail("the watch is still open between two polls")
    }
    XCTAssertFalse(pendingAfter, "no ring once the end is in the record")
    XCTAssertEqual(
      stopAfter?.state, .off(Tips.cannotStop),
      "and between two polls there is no process to signal")
  }

  // MARK: the kernel, faked

  private func armed(_ desk: Desk) -> Stopper {
    Stopper(
      gdoc: Self.gdoc, user: 501, inspect: { desk.facts(for: $0) },
      send: { desk.send($0, $1) }, resolve: { $0 })
  }

  // MARK: the record, written here

  /// One `comments --wait` that is still running.
  private func waiting(
    at: Date, run: String = "a1b2c3d4e5f60718", pid: Int32 = 4311, command: String = "comments",
    via: String? = "terminal", isWait: Bool = true
  ) -> [Event] {
    [
      Event(
        schema: 1, at: at, gdoc: "v2.14.0", pid: pid, run: run, call: 3, kind: .start,
        command: command, via: via, doc: Self.doc, wait: isWait, title: Self.title, ok: nil,
        verified: nil, ms: nil, hold: nil, rule: nil, until: nil)
    ]
  }

  private func end(at: Date, run: String = "a1b2c3d4e5f60718") -> Event {
    Event(
      schema: 1, at: at, gdoc: "v2.14.0", pid: 4311, run: run, call: 3, kind: .end,
      command: "comments", via: "terminal", doc: Self.doc, wait: false, title: Self.title, ok: true,
      verified: nil, ms: 4000, hold: nil, rule: nil, until: nil)
  }

  // MARK: the panel

  private func panel(
    _ events: [Event], at: Date, unstoppable: Set<String> = [], pendingStop: Set<String> = []
  ) throws -> PanelModel {
    var calendar = Calendar(identifier: .gregorian)
    calendar.timeZone = TimeZone(secondsFromGMT: 0) ?? .gmt
    return PanelModel.build(
      events: events,
      facts: Facts(
        account: "nail@example.com", versions: Versions(app: "1.0", gdoc: "v2.14.0"),
        gdocPath: Self.gdoc, unstoppable: unstoppable),
      ui: UIState(pendingStop: pendingStop), now: at, calendar: calendar, alive: { _ in true })
  }

  /// The one NOW item an open watch draws.
  private func first(of panel: PanelModel) throws -> NowItem {
    try XCTUnwrap(panel.now.first, "NOW holds the live row")
  }
}

/// One signal, as the fake desk wrote it down.
private struct Sent: Equatable {
  let pid: Int32
  let signal: Int32
}

/// One machine's answers, and what was sent to it. At file scope, so the
/// facts below are literals and not values the main actor hands out.
private final class Desk {
  /// What the kernel says about the one pid a test asks about. Every field
  /// is stated here, so a refusal is reachable from this file alone.
  struct Facts {
    var path: String? = "/Users/nail/.local/bin/gdoc"
    var uid: uid_t = 501
    var started: Date? = try? moment("2026-10-17T11:59:58Z")
    var isAlive = true
  }

  private(set) var sent: [Sent] = []
  private let answers: Facts

  init(facts: Facts = Facts()) { self.answers = facts }

  func facts(for pid: Int32) -> RunningProcess {
    RunningProcess(
      path: answers.path, uid: answers.uid, started: answers.started, isAlive: answers.isAlive)
  }

  func send(_ pid: Int32, _ signal: Int32) -> Bool {
    sent.append(Sent(pid: pid, signal: signal))
    return true
  }
}

/// A moment this file states in full, so no test reads this machine's clock.
private func moment(_ text: String) throws -> Date {
  let formatter = ISO8601DateFormatter()
  formatter.formatOptions = [.withInternetDateTime]
  return try XCTUnwrap(formatter.date(from: text), "\(text) is not a moment")
}
