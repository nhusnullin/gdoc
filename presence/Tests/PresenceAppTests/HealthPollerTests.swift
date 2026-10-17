// When the app asks gdoc about the machine, and what it does with the answer.
//
// No test here starts a program: the poller is handed a closure that answers
// a call, so every schedule is decided by a clock the test holds and every
// fact comes from an object written in this file. The schedule itself is a
// pure function, and its two gaps are literals here.

import Foundation
import PresenceCore
import PresenceUI
import XCTest

@testable import PresenceApp

final class HealthPollerTests: XCTestCase {

  // MARK: the schedule

  func testThePollIsOnOpenAtMostOnceAMinuteAndEveryTenMinutesWhileLive() async {
    XCTAssertEqual(PollSchedule.openGap, 60)
    XCTAssertEqual(PollSchedule.liveGap, 600)

    let open = Date(timeIntervalSince1970: 1_000_000)

    XCTAssertTrue(
      PollSchedule.asksAuth(.panelOpened, live: false, now: open, lastAsked: nil),
      "a panel that has asked nothing yet asks")
    XCTAssertFalse(
      PollSchedule.asksAuth(
        .panelOpened, live: false, now: open.addingTimeInterval(59), lastAsked: open),
      "opening the panel twice in a minute is one question")
    XCTAssertTrue(
      PollSchedule.asksAuth(
        .panelOpened, live: false, now: open.addingTimeInterval(60), lastAsked: open))

    XCTAssertFalse(
      PollSchedule.asksAuth(
        .timer, live: false, now: open.addingTimeInterval(3600), lastAsked: open),
      "nothing is live, so nothing is asked on a timer")
    XCTAssertFalse(
      PollSchedule.asksAuth(
        .timer, live: true, now: open.addingTimeInterval(599), lastAsked: open))
    XCTAssertTrue(
      PollSchedule.asksAuth(
        .timer, live: true, now: open.addingTimeInterval(600), lastAsked: open),
      "a live session is a token that has to keep working")

    // And through the poller, which is the thing that counts the children.
    let fake = FakeGdoc()
    let poller = HealthPoller(location: Self.found, answer: fake.answer)
    await poller.poll(.panelOpened, live: false, now: open)
    await poller.poll(.panelOpened, live: false, now: open.addingTimeInterval(30))
    await poller.poll(.timer, live: false, now: open.addingTimeInterval(1200))
    let once = await fake.calls
    XCTAssertEqual(once, [.help, .authStatus], "one open and nothing else is one of each")

    await poller.poll(.timer, live: true, now: open.addingTimeInterval(1200))
    let again = await fake.calls
    XCTAssertEqual(again, [.help, .authStatus, .authStatus], "ten minutes later, while live")
  }

  func testHelpIsAskedOnOpenAndNeverOnATimer() async {
    let fake = FakeGdoc()
    let poller = HealthPoller(location: Self.found, answer: fake.answer)
    let open = Date(timeIntervalSince1970: 2_000_000)

    await poller.poll(.panelOpened, live: true, now: open)
    let onOpen = await fake.calls
    XCTAssertEqual(onOpen, [.help, .authStatus])

    // `help --json` carries gdoc's own daily check of what is published, so
    // asking it on a timer would be the app asking GitHub on gdoc's behalf.
    for minutes in [10, 20, 30, 60, 120] {
      await poller.poll(
        .timer, live: true, now: open.addingTimeInterval(Double(minutes) * 60))
    }
    let onTimers = await fake.calls.filter { $0 == .help }
    XCTAssertEqual(onTimers, [.help], "help is asked once, on the open")

    await poller.poll(.panelOpened, live: true, now: open.addingTimeInterval(7200))
    let onTheNextOpen = await fake.calls.filter { $0 == .help }
    XCTAssertEqual(onTheNextOpen.count, 2, "the next open asks again")
  }

  /// A sign-in, a sign-out or an update makes the last answer wrong, so the
  /// app asks both questions again however little time has passed.
  func testAChangeTheAppMadeItselfAsksBothWhateverTheGapsSay() async {
    let open = Date(timeIntervalSince1970: 1_500_000)
    XCTAssertTrue(PollSchedule.asksAuth(.changed, live: false, now: open, lastAsked: open))
    XCTAssertTrue(PollSchedule.asksHelp(.changed, now: open, lastAsked: open))

    let fake = FakeGdoc()
    let poller = HealthPoller(location: Self.found, answer: fake.answer)
    await poller.poll(.panelOpened, live: false, now: open)
    await poller.poll(.changed, live: false, now: open.addingTimeInterval(1))
    let calls = await fake.calls
    XCTAssertEqual(
      calls, [.help, .authStatus, .help, .authStatus],
      "the open asked both, and the change asked both again a second later")
  }

  // MARK: what the answers become

  func testThePolledFactsReachTheFacts() async throws {
    let fake = FakeGdoc()
    let poller = HealthPoller(
      location: Self.found, answer: fake.answer, domain: DomainHint(expected: "example.com"))
    let open = Date(timeIntervalSince1970: 3_000_000)
    await poller.poll(.panelOpened, live: false, now: open)

    let facts = await poller.applied(to: Facts.launching(appVersion: "2.14.0"))
    XCTAssertEqual(facts.account, "nail@example.com")
    XCTAssertEqual(facts.auth, .ok)
    XCTAssertNil(facts.previousAccount, "the first address the app sees is not a change")
    XCTAssertEqual(facts.mark, "⁘")
    XCTAssertEqual(facts.versions.app, "2.14.0", "the app's own version is the app's own")
    XCTAssertEqual(facts.versions.gdoc, "v2.14.0")
    XCTAssertEqual(facts.versions.latestStable, "v2.14.1")
    XCTAssertEqual(facts.versions.updateAvailable, "v2.14.1")
    XCTAssertEqual(facts.gdocPath, Self.found.path)
    XCTAssertEqual(facts.gdocPathTried, Self.found.tried)
    XCTAssertTrue(facts.gdocFound)
    let atHome = await poller.reading.domainHint
    XCTAssertNil(atHome, "the account is at the expected domain")

    // The second account the app sees is the one the row says changed.
    await fake.answers(as: "someone@gmail.example")
    await poller.poll(.panelOpened, live: false, now: open.addingTimeInterval(120))
    let second = await poller.applied(to: Facts.launching(appVersion: "2.14.0"))
    XCTAssertEqual(second.account, "someone@gmail.example")
    XCTAssertEqual(second.previousAccount, "nail@example.com")
    let hint = await poller.reading.domainHint
    XCTAssertEqual(hint, "the expected domain is example.com", "a hint, and the word stays OK")
    XCTAssertEqual(second.auth, .ok)
  }

  func testAGdocThatIsNotThereIsAFactAndNotACrash() async {
    let fake = FakeGdoc()
    let poller = HealthPoller(location: Self.missing, answer: fake.answer)
    await poller.poll(.panelOpened, live: false, now: Date(timeIntervalSince1970: 4_000_000))

    let none = await fake.calls
    XCTAssertEqual(none, [], "nothing to run")
    let facts = await poller.applied(to: Facts.launching(appVersion: "2.14.0"))
    XCTAssertFalse(facts.gdocFound)
    XCTAssertNil(facts.versions.gdoc)
    XCTAssertEqual(facts.gdocPathTried, Self.missing.tried)
  }

  func testAFailedAuthStatusAsksForASignInAndGuessesNothingElse() async {
    let fake = FakeGdoc(broken: [.authStatus])
    let poller = HealthPoller(location: Self.found, answer: fake.answer)
    await poller.poll(.panelOpened, live: false, now: Date(timeIntervalSince1970: 5_000_000))

    let facts = await poller.applied(to: Facts.launching(appVersion: "2.14.0"))
    XCTAssertEqual(
      facts.auth, .needsSignIn, "a token file that cannot be read is a sign-in, never an OK")
    XCTAssertNil(facts.account)
    XCTAssertEqual(facts.versions.gdoc, "v2.14.0", "and help still answered")
  }

  // MARK: the two places, as the runner resolves them

  private static let found = Location(
    path: "/Users/someone/.local/bin/gdoc",
    candidates: ["/Users/someone/.local/bin/gdoc"],
    tried: "~/.local/bin/gdoc")

  private static let missing = Location(
    path: nil, candidates: ["/Users/someone/.local/bin/gdoc"],
    tried: "~/.local/bin/gdoc or ~/Applications/gdoc")
}

/// The gdoc the poller's tests ask, which is an object and never a program.
private actor FakeGdoc {
  private(set) var calls: [Call] = []
  private var account: String
  private let broken: Set<Call>

  init(broken: Set<Call> = [], account: String = "nail@example.com") {
    self.broken = broken
    self.account = account
  }

  func answers(as email: String) { account = email }

  /// The closure the poller is built with. It is `nonisolated` so the poller
  /// holds a plain function, as it will hold the runner's own.
  nonisolated var answer: @Sendable (Call) async throws -> Outcome {
    { [self] call in try await said(call) }
  }

  private func said(_ call: Call) throws -> Outcome {
    calls.append(call)
    if broken.contains(call) {
      return Outcome(
        call: call,
        envelope: Envelope(ok: false, error: "the token file could not be read"),
        standardError: "", status: 1)
    }
    switch call {
    case .help:
      return outcome(
        call,
        #"{"settings":{"mark":"⁘"},"update":{"installed":"v2.14.0","latest_stable":"v2.14.1"}}"#)
    case .authStatus:
      return outcome(
        call,
        #"{"token_present":true,"expired":false,"account":"\#(account)","version":"v2.14.0"}"#)
    default:
      return outcome(call, "{}")
    }
  }

  private func outcome(_ call: Call, _ data: String) -> Outcome {
    Outcome(
      call: call,
      envelope: Envelope(ok: true, data: Data(data.utf8), version: "v2.14.0"),
      standardError: "", status: 0)
  }
}
