// What the person did to the panel, as plain values.
//
// The panel has no state of its own inside a view. A dismissed card, a
// question waiting for its second click and a tooltip held open are all
// here, outside every view, for two reasons: a refold must not lose what a
// person did, and a still of any of those states must be renderable from a
// literal.
//
// Six of these outlive the process, in the app's own store, which is
// PresenceApp/Preferences and the one room that writes anything down: the
// dismissals, whether the icon strip has been seen, whether the card's hint
// has been seen, the hub, and the pinned and hidden documents. The rest die
// with the panel, which is what they are for.

import Foundation
import PresenceCore

/// The one question on the screen, where there is one. Both wait for their
/// answer, or for the panel to close; the clock never takes one away.
public enum PanelQuestion: Sendable, Equatable {
  /// The header's confirm, which names what will stop.
  case signOut
  /// A hold card asking before it dismisses itself, by the hold's id.
  case dismissHold(String)
}

/// Everything the panel remembers between two redraws.
public struct UIState: Sendable, Equatable {
  /// The hold ids and the `run|call` keys a person hid. Only the card
  /// goes; nothing is written either way.
  public var dismissed: Set<String>
  /// The documents whose Stop has been signalled and whose end has not
  /// arrived, so the row draws a ring instead of a second Stop.
  public var pendingStop: Set<String>
  /// The document whose Copy link is green, and the moment it was pressed.
  /// The moment is here rather than in a view so the second is a value a
  /// test states.
  public var copied: String?
  public var copiedAt: Date?
  /// The question on the screen, and when it was asked.
  public var question: PanelQuestion?
  public var questionAt: Date?
  /// One action's words, held open with no pointer. It is an `ActionModel`
  /// id, which is the target and the kind.
  public var forcedTooltip: String?
  /// The document whose card is held open with no pointer, by id. It is
  /// `forcedTooltip`'s shape on the other thing the layer draws, and
  /// nothing in the app ever sets it: a still of the card is the one
  /// reason it exists.
  public var forcedCard: String?
  /// Whether the four labels under the first document row have been seen.
  /// They go after the first click on any icon and never come back.
  public var stripSeen: Bool
  /// Whether the card's one hint has been seen: the line that says a click
  /// on a document row opens the document. It goes after the person's own
  /// first click on a row and never comes back.
  ///
  /// It is the strip's rule on the press the fifth look made: the row is
  /// the way into a document now, and the words for it are the card's. True
  /// here, as `stripSeen` is, so a card built from a literal says nothing
  /// about a hint nobody asked for; the store is where the false lives,
  /// until the first press writes it.
  public var cardHintSeen: Bool
  /// Why the last Open md on a document did not open anything, by document.
  /// A click that went nowhere is the one route to "Note moved or deleted":
  /// the index is rebuilt on a file event, but a note Obsidian would not
  /// open is a fault nothing else in the panel can see. It greys the icon,
  /// so the click that would clear it cannot be made: it dies with the panel
  /// instead, and the next open of the panel is a live icon again.
  public var noteFault: [String: NoteFault]
  /// The documents a person pinned, and the ones they hid. Both are the
  /// app's own preferences, each one a document's id with the moment and the
  /// title the list was drawing, and neither is ever a line in the record.
  /// The fold reads them: a pin is first in DOCUMENTS and never falls out of
  /// it, and a hide leaves the list until gdoc finishes something on that
  /// document again.
  public var pinned: [DocumentChoice]
  public var hidden: [DocumentChoice]

  public init(
    dismissed: Set<String> = [],
    pendingStop: Set<String> = [],
    copied: String? = nil,
    copiedAt: Date? = nil,
    question: PanelQuestion? = nil,
    questionAt: Date? = nil,
    forcedTooltip: String? = nil,
    forcedCard: String? = nil,
    stripSeen: Bool = true,
    cardHintSeen: Bool = true,
    noteFault: [String: NoteFault] = [:],
    pinned: [DocumentChoice] = [],
    hidden: [DocumentChoice] = []
  ) {
    self.dismissed = dismissed
    self.pendingStop = pendingStop
    self.copied = copied
    self.copiedAt = copiedAt
    self.question = question
    self.questionAt = questionAt
    self.forcedTooltip = forcedTooltip
    self.forcedCard = forcedCard
    self.stripSeen = stripSeen
    self.cardHintSeen = cardHintSeen
    self.noteFault = noteFault
    self.pinned = pinned
    self.hidden = hidden
  }

  /// The same state with the icon strip gone.
  ///
  /// The four labels under the first document row are there to be read once.
  /// The first click on any icon is the proof they were read, so they go
  /// then and never come back, and the app's store keeps the flag so a later
  /// launch does not show them again.
  public func afterAnIconClick() -> UIState {
    guard !stripSeen else { return self }
    var next = self
    next.stripSeen = true
    return next
  }

  /// The same state with the card's hint gone.
  ///
  /// The hint is there to be read once. The person's own first press of a
  /// document row is the proof they read it, so it goes then and never
  /// comes back, and the app's store keeps the flag so a later launch does
  /// not show it again. It is `afterAnIconClick`'s rule on the other press.
  public func afterARowOpened() -> UIState {
    guard !cardHintSeen else { return self }
    var next = self
    next.cardHintSeen = true
    return next
  }

  /// Whether this question is still on the screen. The clock is the
  /// caller's, as every other revert in the panel is.
  public func isAsking(_ asked: PanelQuestion, now: Date) -> Bool {
    guard question == asked else { return false }
    return ConfirmRow.isShowing(askedAt: questionAt, now: now)
  }
}
