// Signing in, without a browser and without the real binary.
//
// Two kinds of child are used here: Fixtures/fake-gdoc.sh, which prints the
// sentence and a link the way `gdoc auth login` does, and an object in this
// file for the cases a shell script cannot hold still for, which are the
// cancel and the countdown. Nothing here opens a URL: the opening is a
// closure the test holds.

import Foundation
import PresenceCore
import PresenceUI
import XCTest

@testable import PresenceApp

final class SignInTests: XCTestCase {

  private static let real =
    "https://accounts.google.com/o/oauth2/auth?client_id=fake"

  func testTheURLIsTakenOnlyFromTheChildsStderrAfterTheSentence() async throws {
    // A link printed before the sentence is some other screen's.
    XCTAssertNil(
      SignIn.link(inStandardError: "\(Self.real)\nsomething else entirely\n"),
      "a link with no sentence before it is not the login's link")
    XCTAssertEqual(
      SignIn.link(
        inStandardError: """
          https://accounts.google.com/o/oauth2/auth?client_id=earlier
          \(SignIn.linkAsk)
          \(Self.real)
          """),
      Self.real, "the first non-empty line after the sentence, and only that one")
    XCTAssertEqual(
      SignIn.link(inStandardError: "\(SignIn.linkAsk)\n\n  \(Self.real)  \n"),
      Self.real, "a blank line and the spaces around it are not the link")
    XCTAssertNil(
      SignIn.link(inStandardError: SignIn.linkAsk + "\n"), "a sentence with nothing after it")
    XCTAssertEqual(SignIn.linkAsk, "Open this link in your browser to sign in:")

    // And through a real child, which is the fixture and never gdoc.
    let desk = NoteDesk()
    let runner = try fakeRunner()
    let signIn = SignIn(
      start: { try runner.start(.authLogin) }, open: { desk.open($0) })
    let result = await signIn.run()

    XCTAssertEqual(desk.opened.map(\.absoluteString), [Self.real], "the link the child printed")
    guard case .signedIn(let account, _) = result else {
      return XCTFail("the fixture answers ok, so this is a sign-in: \(result)")
    }
    XCTAssertEqual(account, "someone@example.com")
  }

  func testOnlyGooglesSignInURLIsRefusedFromAChild() async {
    let desk = NoteDesk()
    let child = HeldChild(
      lines: [SignIn.linkAsk, "https://accounts.google.com.evil.example/o/oauth2/auth"],
      envelope: Envelope(ok: true), endsAtOnce: true)
    let signIn = SignIn(start: { child }, open: { desk.open($0) })
    let result = await signIn.run()

    XCTAssertEqual(
      result, .refused("https://accounts.google.com.evil.example/o/oauth2/auth"),
      "a lookalike host is not Google")
    XCTAssertTrue(desk.opened.isEmpty, "and nothing was opened")
  }

  func testTheCountdownIs180Seconds() async {
    XCTAssertEqual(SignIn.countdown, 180)
    XCTAssertEqual(Timing.signInCountdown, 180)

    let began = Date(timeIntervalSince1970: 1_792_000_000)
    XCTAssertEqual(SignIn.secondsLeft(startedAt: began, now: began), 180)
    XCTAssertEqual(SignIn.secondsLeft(startedAt: began, now: began.addingTimeInterval(1)), 179)
    XCTAssertEqual(SignIn.secondsLeft(startedAt: began, now: began.addingTimeInterval(179.5)), 0)
    XCTAssertEqual(SignIn.secondsLeft(startedAt: began, now: began.addingTimeInterval(180)), 0)
    XCTAssertEqual(
      SignIn.secondsLeft(startedAt: began, now: began.addingTimeInterval(600)), 0,
      "a countdown never runs past its end")

    // And the line the panel draws, which is the link and what is left.
    let desk = NoteDesk()
    let child = HeldChild(lines: [SignIn.linkAsk, Self.real], envelope: Envelope(ok: true))
    let signIn = SignIn(start: { child }, open: { desk.open($0) }, clock: { began })
    let running = Task { await signIn.run() }
    await child.printed()
    await tookTheLink(signIn)

    let phase = await signIn.phase(now: began.addingTimeInterval(20))
    XCTAssertEqual(phase, SignInPhase(url: Self.real, secondsLeft: 160))

    await child.finish()
    _ = await running.value
    let after = await signIn.phase(now: began.addingTimeInterval(20))
    XCTAssertNil(after, "a sign-in that ended draws no line")
  }

  func testCancelTerminatesTheChild() async {
    let desk = NoteDesk()
    let child = HeldChild(lines: [SignIn.linkAsk, Self.real], envelope: Envelope(ok: true))
    let signIn = SignIn(start: { child }, open: { desk.open($0) })
    let running = Task { await signIn.run() }
    await child.printed()
    await tookTheLink(signIn)

    await signIn.cancel()
    let result = await running.value
    XCTAssertEqual(result, .cancelled, "a cancelled sign-in is cancelled and not a failure")
    let signals = await child.terminations
    XCTAssertEqual(
      signals, 1, "one terminate, and nothing sent twice or of another kind")
  }

  func testTheBannerFollowsAnOkEnvelope() async throws {
    let moment = Date(timeIntervalSince1970: 1_792_000_000)
    let desk = NoteDesk()
    let runner = try fakeRunner()
    let signIn = SignIn(
      start: { try runner.start(.authLogin) }, open: { desk.open($0) }, clock: { moment })

    guard case .signedIn(let account, let at) = await signIn.run() else {
      return XCTFail("the fixture answers ok")
    }
    XCTAssertEqual(account, "someone@example.com")
    XCTAssertEqual(at, moment)

    XCTAssertTrue(SignedInBanner.isShowing(signedInAt: at, now: at.addingTimeInterval(4)))
    XCTAssertFalse(
      SignedInBanner.isShowing(signedInAt: at, now: at.addingTimeInterval(6)),
      "the banner lives five seconds")

    // An envelope that is not ok is not a sign-in, whatever the child printed.
    let refused = HeldChild(
      lines: [SignIn.linkAsk, Self.real],
      envelope: Envelope(ok: false, error: "the browser never came back"), endsAtOnce: true)
    let second = SignIn(start: { refused }, open: { desk.open($0) }, clock: { moment })
    let answered = await second.run()
    XCTAssertEqual(answered, .failed("the browser never came back"))
  }

  /// Wait until the sign-in has taken the link off the child, so the test
  /// looks at the panel's line after the line exists.
  private func tookTheLink(_ signIn: SignIn) async {
    for _ in 0..<400 {
      if await signIn.linkInFlight() != nil { return }
      try? await Task.sleep(nanoseconds: 5_000_000)
    }
    XCTFail("the sign-in never took the link the child printed")
  }

  // MARK: the fixture, run at its own path

  private func fakeRunner() throws -> Runner {
    let script = URL(fileURLWithPath: #filePath)
      .deletingLastPathComponent()
      .appendingPathComponent("Fixtures/fake-gdoc.sh")
    XCTAssertTrue(
      FileManager.default.isExecutableFile(atPath: script.path),
      "Fixtures/fake-gdoc.sh is not executable, so nothing here runs")
    return Runner(
      location: Location(path: script.path, candidates: [script.path], tried: script.path),
      timeout: 20, environment: ProcessInfo.processInfo.environment)
  }
}

/// A child that prints when the test says and ends when the test says.
///
/// An actor because the sign-in reads it from its own, and because the test
/// drives both sides in order: print, then look, then end.
actor HeldChild: SignInChild {
  private let lines: [String]
  private let envelope: Envelope
  /// Whether the child is already finished when it is asked for its
  /// outcome, which is what a command that printed everything in one breath
  /// looks like.
  private let endsAtOnce: Bool
  private var sink: (@Sendable (String) -> Void)?
  private var ending: CheckedContinuation<Outcome, Error>?
  private var ended = false
  private(set) var terminations = 0

  init(lines: [String], envelope: Envelope, endsAtOnce: Bool = false) {
    self.lines = lines
    self.envelope = envelope
    self.endsAtOnce = endsAtOnce
  }

  nonisolated func onStandardErrorLine(_ sink: @escaping @Sendable (String) -> Void) {
    Task { await take(sink) }
  }

  private func take(_ sink: @escaping @Sendable (String) -> Void) {
    self.sink = sink
  }

  /// Write every line to whoever is listening. Waits for the sink, because
  /// registering it is the sign-in's first act and arrives on its own actor.
  func printed() async {
    while sink == nil { await Task.yield() }
    for line in lines { sink?(line) }
  }

  nonisolated func outcome() async throws -> Outcome {
    try await withCheckedThrowingContinuation { continuation in
      Task { await hold(continuation) }
    }
  }

  private func hold(_ continuation: CheckedContinuation<Outcome, Error>) {
    guard !ended, !endsAtOnce else {
      continuation.resume(returning: answer())
      return
    }
    ending = continuation
  }

  /// End the child the way one whose browser came back ends.
  func finish() {
    ended = true
    ending?.resume(returning: answer())
    ending = nil
  }

  nonisolated func terminate() {
    Task { await signalled() }
  }

  private func signalled() {
    terminations += 1
    finish()
  }

  private func answer() -> Outcome {
    Outcome(
      call: .authLogin, envelope: envelope,
      standardError: lines.joined(separator: "\n") + "\n", status: 0)
  }
}
