// Signing in, as the app runs it.
//
// `gdoc auth login` prints one link to standard error and then waits three
// minutes for the browser to come back on its own loopback listener. The app
// runs that command as its own child, reads the link off its standard error,
// judges it, opens it, and shows it as a line somebody can copy with the
// child's own countdown beside it. Cancel terminates the child; nothing else
// here signals anything.
//
// The link is taken from the child and from nowhere else, off the first
// non-empty line after the sentence the command prints before it, and off no
// other line. A line that came earlier is some other screen, and one that
// came later has missed its turn. What is taken is then held to one scheme,
// one host and one path in PresenceCore/LinkLine, so a child that printed
// something else opens nothing at all.
//
// The lines reach this actor through one stream with one reader, so they
// arrive in the order the child wrote them. The command prints its sentence
// and its link in a single write, and a task per line may arrive either way
// round, which would drop the link.
//
// This file holds no window. The stage is a value the panel reads, and the
// opening is the Opener's one verb handed in as a closure.

import Foundation
import PresenceCore
import PresenceUI

/// What a sign-in needs of a child. The runner's own `Child` is one; a test
/// hands over an object of its own, so no test here waits for a browser.
public protocol SignInChild: AnyObject, Sendable {
  func onStandardErrorLine(_ sink: @escaping @Sendable (String) -> Void)
  func outcome() async throws -> Outcome
  func terminate()
}

extension Child: SignInChild {}

/// How a sign-in ended.
public enum SignInResult: Sendable, Equatable {
  /// The envelope said `ok`. The address is the one the command printed,
  /// which may be absent: a token with no address in it is still a token.
  case signedIn(account: String?, at: Date)
  /// The child printed something that is not Google's sign-in link. The
  /// string is what it printed, for a test and never for a person.
  case refused(String)
  /// The child ended without printing a link at all.
  case noLink
  case cancelled
  /// The child answered, and its answer was not `ok`.
  case failed(String)
}

/// A sign-in in flight, and the one that just ended.
///
/// An actor because the child's standard error arrives on a thread of its
/// own while the panel reads the stage on the main one.
public actor SignIn {
  /// How a child is made. The app hands over the runner's own start; a test
  /// hands over an object.
  public typealias Start = @Sendable () throws -> SignInChild

  /// The countdown, which is the child's own patience and never a guess at
  /// how long Google will take.
  public static let countdown: TimeInterval = Timing.signInCountdown

  /// How long the child is waited for: its three minutes, and a little for
  /// the end to arrive after them.
  public static let childTimeout: TimeInterval = Timing.signInCountdown + 10

  /// The sentence `gdoc auth login` prints before the link. It is
  /// `go/internal/auth/login.go:241`'s own words.
  public static let linkAsk = "Open this link in your browser to sign in:"

  private let start: Start
  private let open: @Sendable (URL) -> Bool
  private let clock: @Sendable () -> Date

  private var child: SignInChild?
  private var startedAt: Date?
  private var link: String?
  private var said: [String] = []
  private var cancelled = false

  public init(
    start: @escaping Start,
    open: @escaping @Sendable (URL) -> Bool,
    clock: @escaping @Sendable () -> Date = { Date() }
  ) {
    self.start = start
    self.open = open
    self.clock = clock
  }

  /// The line the panel draws: the link and what is left of the three
  /// minutes, or nothing when no sign-in is running.
  public func phase(now: Date) -> SignInPhase? {
    guard let link, let startedAt else { return nil }
    return SignInPhase(url: link, secondsLeft: Self.secondsLeft(startedAt: startedAt, now: now))
  }

  /// The link, once the child has printed one.
  public func linkInFlight() -> String? { link }

  /// Run one sign-in from end to end.
  public func run() async -> SignInResult {
    let began = clock()
    startedAt = began
    link = nil
    said = []
    cancelled = false

    let running: SignInChild
    do {
      running = try start()
    } catch {
      startedAt = nil
      return .failed(String(describing: error))
    }
    child = running

    // The lines go through one stream with one reader, so they reach this
    // actor in the order the child wrote them. A Task per line would not:
    // the command prints its sentence and its link in a single write, and
    // two unstructured tasks may arrive either way round, which would drop
    // the link and leave the person with no line to open and no countdown.
    // testTheLinkIsTakenHoweverTheLinesAreDelivered.
    let (lines, writing) = AsyncStream<String>.makeStream()
    running.onStandardErrorLine { line in writing.yield(line) }
    let reading = Task { [weak self] in
      for await line in lines { await self?.printed(line) }
    }

    defer {
      writing.finish()
      reading.cancel()
      child = nil
      startedAt = nil
      link = nil
    }

    let outcome: Outcome
    do {
      outcome = try await running.outcome()
    } catch {
      return cancelled ? .cancelled : .failed(String(describing: error))
    }
    // Every line the sink delivered is read before the child's own standard
    // error is, so the two routes never interleave.
    writing.finish()
    await reading.value
    if cancelled { return .cancelled }

    // The whole of what the child said is read once more at the end: a child
    // that printed its link and its object in the same breath may have been
    // waited on before the live sink was ever called.
    if link == nil {
      said = []
      await printed(all: outcome.standardError)
    }

    guard outcome.envelope.ok else {
      return .failed(outcome.envelope.error ?? "auth login did not say why")
    }
    guard link != nil else { return refusal() ?? .noLink }
    return .signedIn(account: Self.account(in: outcome.envelope.data), at: clock())
  }

  /// Stop the sign-in. One terminate to the child this app started, and no
  /// other signal to anything.
  public func cancel() {
    guard let running = child else { return }
    cancelled = true
    running.terminate()
  }

  // MARK: what the child said

  private func printed(all text: String) async {
    for line in text.split(separator: "\n", omittingEmptySubsequences: false) {
      await printed(String(line))
    }
  }

  /// One line, as it arrives. The candidate is recomputed over everything
  /// said so far rather than read off this line, so the live path and the
  /// pure reading below are the one rule: the first non-empty line after the
  /// sentence, and only that line. A line that arrives after a candidate has
  /// already been judged is some other screen's and opens nothing.
  private func printed(_ line: String) async {
    said.append(line)
    guard link == nil, let candidate = Self.candidate(in: said) else { return }
    guard candidate == line.trimmingCharacters(in: .whitespaces) else { return }
    guard let accepted = LinkLine.signInURL(candidate) else { return }
    link = accepted
    guard let url = URL(string: accepted) else { return }
    _ = open(url)
  }

  /// Whether the line after the sentence was a link this app would not open.
  /// It is the difference between a child that printed nothing and one that
  /// printed somewhere else.
  private func refusal() -> SignInResult? {
    Self.candidate(in: said).map(SignInResult.refused)
  }

  // MARK: the two pure readings

  /// The one line a link may come off: the first non-empty line after the
  /// sentence. Everything before the sentence is some other screen's, and
  /// everything after that line has already had its turn.
  static func candidate(in lines: [String]) -> String? {
    guard let ask = lines.firstIndex(where: { $0.contains(linkAsk) }) else { return nil }
    return lines[(ask + 1)...]
      .map { $0.trimmingCharacters(in: .whitespaces) }
      .first(where: { !$0.isEmpty })
  }

  /// The link in everything a child wrote, or nothing. The first non-empty
  /// line after the sentence, and only that line.
  public static func link(inStandardError text: String) -> String? {
    let lines = text.split(separator: "\n", omittingEmptySubsequences: false).map(String.init)
    guard let after = candidate(in: lines) else { return nil }
    return LinkLine.signInURL(after)
  }

  /// What is left of the three minutes, in whole seconds, never under zero.
  public static func secondsLeft(startedAt: Date, now: Date) -> Int {
    let left = countdown - now.timeIntervalSince(startedAt)
    return max(0, Int(left.rounded(.down)))
  }

  /// The address in the object `auth login` printed, where it carries one.
  static func account(in data: Data?) -> String? {
    guard let data, let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any]
    else { return nil }
    return object["account"] as? String
  }
}
