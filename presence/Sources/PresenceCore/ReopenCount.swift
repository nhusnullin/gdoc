// The five seconds between an update that replaced the bundle and the app
// opening itself again, as the person reads them down on the notice row.
//
// `gdoc update` swaps this app's bundle by rename while the app is running,
// and the app then opens the new bundle itself. For the seconds in between
// the row used to say "Quit and reopen", which asked for a hand the app did
// not need and was about to not need. So the row counts instead, and this is
// the clock behind it.
//
// It is pure: the session asks PresenceCore/RelaunchPolicy and reads its own
// clock, hands both over as one gate and one moment, and this decides. So
// every answer, including the one open, is reachable from a test without a
// second passing.
//
// A held gate during the count does not restart it. The two holds that clear
// by themselves are a live `gdoc update` and a child of this app's own, and
// a poll child is a few hundred milliseconds: a row that jumped back to the
// update's words and then to 5 again would be the panel losing its place
// over something the person cannot see. A held gate at five seconds does
// restart it, because the bytes may still be arriving, and a 1 that sat
// there waiting would be the promise this count replaced.
//
// `.opening` is a state because the open is asked for in a Task and the next
// tick lands before that Task answers. The row holds at 1 and the count
// opens nothing again; the session is what turns it into `.refused` if the
// open did not happen.
//
// This is the wait the relauncher used to sleep: `RelaunchPolicy.settle` is
// gone and the open is asked for at zero, which is why there is no
// two-second wait anywhere else.

import Foundation

/// What the session learned about the policy on this ask.
///
/// It comes straight out of `RelaunchPolicy.decide`: clear where it said
/// relaunch, held on the two holds that clear by themselves, and refused on
/// every other hold.
public enum ReopenGate: Sendable, Equatable {
  case clear, held, refused
}

/// What the notice row says about the reopen.
public enum ReopenStanding: Sendable, Equatable {
  /// Something is holding the reopen, and it will clear by itself.
  case held
  /// This many seconds are left, 5 down to 1.
  case counting(Int)
  /// The app is not going to reopen itself, so the row asks for a hand.
  case refused
}

/// Where the reopen of this bundle has got to.
public enum ReopenCount: Sendable, Equatable {
  /// Nothing is counting: the policy is holding, or no bundle moved.
  case held
  /// The count that began at this moment.
  case counting(began: Date)
  /// The open has been asked for and has not answered.
  case opening
  /// The app is not reopening itself.
  case refused

  /// How long the count is, which is the whole of the wait before the open.
  public static let seconds = 5

  /// Whether the count is over for good.
  public var isRefused: Bool {
    if case .refused = self { return true }
    return false
  }

  /// The count moved on, what the row says, and whether this is the one
  /// moment to open.
  public struct Step: Equatable {
    public let count: ReopenCount
    public let standing: ReopenStanding
    public let opensNow: Bool
  }

  /// This ask, as a new value: nothing here is mutated.
  public func asked(_ gate: ReopenGate, now: Date) -> Step {
    let refusal = Step(count: ReopenCount.refused, standing: .refused, opensNow: false)
    let holding = Step(count: ReopenCount.held, standing: .held, opensNow: false)

    switch (self, gate) {
    case (.refused, _), (_, .refused):
      return refusal
    case (.opening, _):
      return Step(count: .opening, standing: .counting(1), opensNow: false)
    case (.held, .clear):
      return Step(count: .counting(began: now), standing: .counting(Self.seconds), opensNow: false)
    case (.held, _):
      return holding
    case (.counting(let began), _):
      guard now.timeIntervalSince(began) >= Double(Self.seconds) else {
        return Step(count: self, standing: Self.left(since: began, now: now), opensNow: false)
      }
      guard gate == .clear else { return holding }
      return Step(count: .opening, standing: .counting(1), opensNow: true)
    }
  }

  /// The seconds on the row, clamped to 1 through 5, so a clock that went
  /// back reads 5 and a count nobody opened never reads a negative number.
  private static func left(since began: Date, now: Date) -> ReopenStanding {
    let elapsed = Int(now.timeIntervalSince(began))
    return .counting(min(seconds, max(1, seconds - elapsed)))
  }
}
