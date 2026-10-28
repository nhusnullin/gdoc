// When the card is up, as a value rather than as a view's private mood.
//
// The card is a rest and not an entry, which is the tooltip's own rule: a
// pointer crossing a row on its way to SETTINGS is not asking what that row
// says. It is also the one thing in the panel that two other gestures take
// away, because both of them put something else over the same words: a
// held press opens the system's menu, and a click opens a browser tab.
//
// So the question "is the card drawn" has five inputs and no clock of its
// own, and it is written here as a value a test feeds events to, in the
// shape UIState is written in: no view decides it, and a still of any
// moment of it is a literal.
//
// After a held press or a click the card stays away until the pointer
// leaves and comes back. A card that reappeared under a hand that had just
// pressed through it would be the panel arguing with the person.
//
// ControlTests.testTheCardShowsAfterTheRestAndGoesOnLeaveHeldPressOrClick
// walks the five.

import Foundation

/// The five things that happen to a row's middle column, as far as the card
/// is concerned.
public enum CardRestEvent: String, Sendable, Equatable, CaseIterable {
  /// The pointer arrived.
  case enter
  /// It has been still for `Tooltip.delay`.
  case rested
  /// It left.
  case leave
  /// A press was held long enough to be the menu's.
  case heldPress
  /// A press was released before that, which opens the document.
  case click
}

/// Whether this row's card is drawn, and what got it there.
public struct CardRest: Sendable, Equatable {
  /// The pointer is over the words.
  private var inside = false
  /// A press happened under this pointer, so nothing shows until it leaves.
  private var pressed = false
  /// What the carrier asks.
  public private(set) var isShowing = false

  public init() {}

  /// The same rest, one event later.
  public mutating func after(_ event: CardRestEvent) {
    switch event {
    case .enter:
      inside = true
      pressed = false
      isShowing = false
    case .rested:
      isShowing = inside && !pressed
    case .leave:
      inside = false
      isShowing = false
    case .heldPress, .click:
      pressed = true
      isShowing = false
    }
  }
}
