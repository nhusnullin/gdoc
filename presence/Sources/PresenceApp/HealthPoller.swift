// When gdoc is asked about the machine, and what the answers become.
//
// Two of the six argv lists are questions rather than actions: `help --json`
// and `auth status`. This file holds when each is asked and what its answer
// does to the panel's facts, and nothing else. It starts nothing itself: the
// child comes from a closure, which is the runner's own `run` in the app and
// an object written in a test under test.
//
// The schedule is a pure function of a reason, a clock and what was asked
// before, so a person opening and closing the panel ten times in a minute
// asks gdoc once. `help --json` is asked on open and never on a timer,
// because that command carries gdoc's own once-a-day check of what is
// published: a timer here would be this app asking GitHub on gdoc's behalf.
// `auth status` is the one a live session needs to keep working, so it is
// asked again every ten minutes while one is open.

import Foundation
import PresenceCore
import PresenceUI

/// Why a poll is happening.
public enum PollReason: Sendable, Equatable {
  case panelOpened
  case timer
}

/// When each question is asked. Both gaps are counted from the last time the
/// question was asked, never from the last answer: a command that is slow is
/// still a command that was asked.
public enum PollSchedule {
  /// The floor under opening the panel. Opening it twice inside a minute is
  /// one question.
  public static let openGap: TimeInterval = 60
  /// How often a live session's token is checked again.
  public static let liveGap: TimeInterval = 600

  public static func asksAuth(
    _ reason: PollReason, live: Bool, now: Date, lastAsked: Date?
  ) -> Bool {
    guard let lastAsked else { return reason == .panelOpened || live }
    let since = now.timeIntervalSince(lastAsked)
    switch reason {
    case .panelOpened: return since >= openGap
    case .timer: return live && since >= liveGap
    }
  }

  public static func asksHelp(_ reason: PollReason, now: Date, lastAsked: Date?) -> Bool {
    guard reason == .panelOpened else { return false }
    guard let lastAsked else { return true }
    return now.timeIntervalSince(lastAsked) >= openGap
  }
}

/// What the last answers said, as the panel's facts are built from them.
public struct HealthReading: Sendable, Equatable {
  public let help: HelpFacts?
  public let auth: AuthFacts?
  /// Whether the last `auth status` came back saying it could not read the
  /// token file. It is not an auth fact: there was no token to describe.
  public let authUnreadable: Bool
  /// The address the app saw before this one, which is its own memory.
  public let previousAccount: String?
  /// The line about the expected domain, where the preference is set and the
  /// account is at another one. It blocks nothing.
  public let domainHint: String?

  public init(
    help: HelpFacts? = nil,
    auth: AuthFacts? = nil,
    authUnreadable: Bool = false,
    previousAccount: String? = nil,
    domainHint: String? = nil
  ) {
    self.help = help
    self.auth = auth
    self.authUnreadable = authUnreadable
    self.previousAccount = previousAccount
    self.domainHint = domainHint
  }

  /// The one word the account row says, or nothing before the first answer.
  /// A token file that could not be read asks for a sign-in: the command
  /// answers `ok: true` with `token_present: false` for a machine that is
  /// signed out, so an object that says `ok: false` is a fault in the file
  /// itself, and the person has one thing to do about it either way.
  public var state: AuthState? {
    if authUnreadable { return .needsSignIn }
    return auth?.state
  }
}

/// The two questions, asked on a schedule, and the answers as facts.
///
/// An actor because the answers arrive off the main thread and the panel
/// reads the reading on it: the memory of the last account and the two
/// stamps are the state, and they are the actor's.
public actor HealthPoller {
  /// How a call is answered. The app hands over the runner's own `run`; a
  /// test hands over an object.
  public typealias Answer = @Sendable (Call) async throws -> Outcome

  public private(set) var reading = HealthReading()

  private let location: Location
  private let answer: Answer
  private let domain: DomainHint
  private var memory: AccountMemory
  private var helpAsked: Date?
  private var authAsked: Date?

  public init(
    location: Location,
    answer: @escaping Answer,
    domain: DomainHint = .unset,
    memory: AccountMemory = AccountMemory()
  ) {
    self.location = location
    self.answer = answer
    self.domain = domain
    self.memory = memory
  }

  /// Ask what the schedule says to ask, and hand back the calls that were
  /// made. A machine with no gdoc on it asks nothing: there is nothing to
  /// run, and the not-found fact is the location's own.
  @discardableResult
  public func poll(_ reason: PollReason, live: Bool, now: Date = Date()) async -> [Call] {
    guard location.found else { return [] }
    var made: [Call] = []

    if PollSchedule.asksHelp(reason, now: now, lastAsked: helpAsked) {
      helpAsked = now
      made.append(.help)
      await askedHelp()
    }
    if PollSchedule.asksAuth(reason, live: live, now: now, lastAsked: authAsked) {
      authAsked = now
      made.append(.authStatus)
      await askedAuth()
    }
    return made
  }

  /// The panel's facts with what the polls know written into them. Every
  /// other field is the one it was handed: this room knows the account, the
  /// versions, the mark and where the binary is, and nothing else.
  public func applied(to facts: Facts) -> Facts {
    Facts(
      account: reading.auth?.account ?? facts.account,
      auth: reading.state.map(word(for:)) ?? facts.auth,
      previousAccount: reading.previousAccount ?? facts.previousAccount,
      versions: Versions(
        app: facts.versions.app,
        gdoc: reading.help?.installed ?? reading.auth?.version ?? facts.versions.gdoc,
        latestStable: reading.help?.latestStable ?? facts.versions.latestStable,
        skills: facts.versions.skills),
      mark: reading.help?.mark ?? facts.mark,
      gdocPath: location.path,
      gdocPathTried: location.tried,
      hub: facts.hub,
      extensionState: facts.extensionState,
      network: facts.network,
      loginItem: facts.loginItem,
      signIn: facts.signIn,
      pairing: facts.pairing,
      signedInAt: facts.signedInAt,
      bundle: facts.bundle,
      newerRecord: facts.newerRecord)
  }

  // MARK: the two questions

  private func askedHelp() async {
    guard let outcome = try? await answer(.help), outcome.envelope.ok,
      let data = outcome.envelope.data, let help = try? HelpFacts.decode(data)
    else { return }
    reading = HealthReading(
      help: help, auth: reading.auth, authUnreadable: reading.authUnreadable,
      previousAccount: reading.previousAccount, domainHint: reading.domainHint)
  }

  /// What the token is, or that the file holding it could not be read.
  ///
  /// A child that did not answer at all leaves the last reading alone: an app
  /// that could not ask knows nothing new. An object that came back and could
  /// not be read is the file's own fault, which the reading carries as what
  /// it is rather than as an auth fact nobody printed.
  private func askedAuth() async {
    guard let outcome = try? await answer(.authStatus) else { return }
    guard outcome.envelope.ok, let data = outcome.envelope.data,
      let auth = try? AuthFacts.decode(data)
    else {
      reading = HealthReading(
        help: reading.help, auth: nil, authUnreadable: true,
        previousAccount: reading.previousAccount, domainHint: reading.domainHint)
      return
    }
    let changed = memory.change(to: auth.account)
    memory = memory.noting(auth.account)
    reading = HealthReading(
      help: reading.help, auth: auth, authUnreadable: false,
      previousAccount: changed ?? reading.previousAccount,
      domainHint: domain.sentence(for: auth.account))
  }

  /// The core's state as the panel's word. Two enums rather than one,
  /// because the core holds no sentence a person reads.
  private func word(for state: AuthState) -> AuthWord {
    switch state {
    case .ok: .ok
    case .needsSignIn: .needsSignIn
    case .missingScopes: .missingScopes
    case .signedOut: .signedOut
    }
  }
}
