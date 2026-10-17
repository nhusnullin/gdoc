// The panel's delays, every one of them a decision.
//
// A number here is a thing a person waits for, so each is the product
// decision that set it and never a number a view felt like. Nothing here
// animates for its own sake: the panel's one moving part is the ring.

import Foundation

/// The six delays the panel is built on.
public enum Timing {
  /// How long a pointer rests before the app draws a tooltip.
  public static let tooltipDelay: TimeInterval = 0.5
  /// How long a copied icon stays green before it goes back.
  public static let doneFlash: TimeInterval = 1
  /// How long an inline confirm waits before it reverts itself, so a press
  /// nobody followed up on undoes itself.
  public static let confirmRevert: TimeInterval = 3
  /// The green check in the menu bar after a verified write that arrived
  /// live, and the life of the signed-in banner.
  public static let checkFlash: TimeInterval = 5
  /// The sign-in countdown, which is the child's own patience.
  public static let signInCountdown: TimeInterval = 180
  /// At most one self-relaunch per on-disk version in ten minutes.
  public static let relaunchGuard: TimeInterval = 600
}
