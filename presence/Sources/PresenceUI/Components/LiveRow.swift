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
  /// `Live · Chapter 5 storyline`.
  public let title: String
  /// Since when, how long it has been quiet, what it answered, and whether
  /// the last poll failed. Facts, never a prediction.
  public let sub: String

  public init(title: String, sub: String) {
    self.title = title
    self.sub = sub
  }
}

/// One watch.
public struct LiveRow: View {
  /// The words on the one icon.
  public static let stopTip = "Stop watching"

  public let model: LiveRowModel
  /// `.live`, or `.off` with the words the process check refuses by.
  public let state: ActionState
  /// Whether the signal has gone and the end has not arrived.
  public let pending: Bool
  public let onStop: () -> Void

  public init(
    model: LiveRowModel, stop: ActionState = .live, pending: Bool = false,
    onStop: @escaping () -> Void = {}
  ) {
    self.model = model
    self.state = stop
    self.pending = pending
    self.onStop = onStop
  }

  /// The icons the row carries: the one, or none while it is pending.
  public static func icons(pending: Bool) -> [Icon] {
    pending ? [] : [.stop]
  }

  /// Whether the ring stands where the icon was.
  public static func showsRing(pending: Bool) -> Bool { pending }

  public var body: some View {
    RowView(
      .two, glyph: .dot, tone: .live, title: model.title, sub: model.sub,
      trailing: {
        if Self.showsRing(pending: pending) {
          // The ring is the glyph's 16 pt in the action's 24 pt box, so the
          // row does not change height when the icon goes.
          RingView()
            .metricProbe("live.ring")
            .frame(width: Grid.actionSize, height: Grid.actionSize)
        } else {
          ActionButton(.stop, tip: Self.stopTip, state: state, action: onStop)
            .metricProbe("live.stop")
        }
      })
  }
}
