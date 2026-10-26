// A live watch, as a row with one action.
//
// `.row-live` in docs/design/2026-10-17-presence-screens-v3.html is the
// two-line row with the filled dot in the panel's brightest grey, the
// document it watches, a sub-line of facts, and one icon: Stop watching.
// There is no "stop all" and no confirm. One person with two documents is
// two clicks, and stopping a watch is cheap and reversible, which is the one
// place in the panel where that is true.
//
// While the SIGINT is in flight the icon becomes the small ring: a stop that
// can be pressed twice is a stop nobody trusts, and the app sends exactly
// one signal. The end arriving takes the row away, which is the fold's doing
// and not this view's.
//
// CardTests states the one icon, the ring and the row's height.

import SwiftUI

/// What a live row draws.
public struct LiveRowModel: Equatable, Sendable {
  /// The verb in front of the name, "Live" or "Live stopped". It is drawn
  /// whole and the name is measured in the room it leaves, so a watch over
  /// `Ch 5. Commercial Model v0.1` keeps a name a person can read:
  /// testAPrefixTakesItsRoomAndTheFloorCountsTheName.
  public let prefix: String?
  /// The document's own name, `Chapter 5 storyline`.
  public let name: String
  /// Since when, how long it has been quiet, what it replied, and whether
  /// the last poll failed. Facts, never a prediction.
  public let sub: String
  /// Whether the watch is over. NOW keeps a watch the sign-in stopped,
  /// because signing in does not start it again, and the dot goes dim with
  /// the word: a live dot beside the word stopped is the panel saying two
  /// things about one watch. Scene e9.
  public let stopped: Bool

  public init(prefix: String? = nil, name: String, sub: String, stopped: Bool = false) {
    self.prefix = prefix
    self.name = name
    self.sub = sub
    self.stopped = stopped
  }

  /// `Live · Chapter 5 storyline`: the line a reader sees, which is the two
  /// pieces put back together.
  public var title: String { MiddleTitle.whole(prefix: prefix, name: name) }
}

/// One watch.
public struct LiveRow: View {
  /// The words on the one icon.
  public nonisolated static let stopTip = "Stop watching"

  public let model: LiveRowModel
  /// `.live`, or `.off` with the words the process check refuses by.
  /// Absent where there is nothing to stop at all, which is a watch the
  /// sign-in ended: no process is polling and no later one will be.
  public let state: ActionState?
  /// Whether the signal has gone and the end has not arrived.
  public let pending: Bool
  public let onStop: () -> Void

  public init(
    model: LiveRowModel, stop: ActionState? = .live, pending: Bool = false,
    onStop: @escaping () -> Void = {}
  ) {
    self.model = model
    self.state = stop
    self.pending = pending
    self.onStop = onStop
  }

  /// The icons the row carries: the one, none while it is pending, and none
  /// where there is nothing to stop.
  ///
  /// A row handed no state drew the liveliest icon in the panel for a watch
  /// nobody can stop, because the view filled the absence in with `.live`:
  /// testE9SaysTheWatchIsStoppedInEverySectionAndDrawsNoStop.
  public static func icons(pending: Bool, stoppable: Bool = true) -> [Icon] {
    pending || !stoppable ? [] : [.stop]
  }

  /// Whether the ring stands where the icon was.
  public static func showsRing(pending: Bool) -> Bool { pending }

  public var body: some View {
    RowView(
      .two, glyph: .dot, tone: model.stopped ? .dim : .live, title: model.name,
      titlePrefix: model.prefix, titleColumn: MiddleTitle.nowColumn, sub: model.sub,
      trailing: {
        if Self.showsRing(pending: pending) {
          // The ring is the glyph's 16 pt in the action's 24 pt box, so the
          // row does not change height when the icon goes.
          RingView()
            .metricProbe("live.ring")
            .frame(width: Grid.actionSize, height: Grid.actionSize)
        } else if let state {
          ActionButton(.stop, tip: Self.stopTip, state: state, action: onStop)
            .metricProbe("live.stop")
        }
      })
  }
}
