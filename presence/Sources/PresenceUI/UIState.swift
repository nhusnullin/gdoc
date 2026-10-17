// What the person did to the panel, as plain values.
//
// The panel has no state of its own inside a view. An expanded section, a
// dismissed card, a question waiting for its second click and a tooltip held
// open are all here, outside every view, for two reasons: a refold must not
// close a section somebody opened, and a still of any of those states must
// be renderable from a literal.
//
// Three of these outlive the process and are written to UserDefaults in
// Task 19: the dismissals, whether the icon strip has been seen, and the
// hub. The rest die with the panel, which is what they are for.

import Foundation

/// The three lists a person can open.
public enum PanelList: String, Sendable, Equatable, CaseIterable {
  case documents
  case today
  case yesterday
}

/// The one question on the screen, where there is one. Both revert after
/// three seconds, which the clock outside decides.
public enum PanelQuestion: Sendable, Equatable {
  /// The HEALTH row's confirm, which names what will stop.
  case signOut
  /// A hold card asking before it dismisses itself, by the hold's id.
  case dismissHold(String)
}

/// Everything the panel remembers between two redraws.
public struct UIState: Sendable, Equatable {
  /// The hold ids and the `run|call` keys a person hid. The row in TODAY
  /// stays; only the card goes.
  public var dismissed: Set<String>
  /// The sections that are open.
  public var expanded: Set<PanelList>
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
  /// Whether the four labels under the first document row have been seen.
  /// They go after the first click on any icon and never come back.
  public var stripSeen: Bool

  public init(
    dismissed: Set<String> = [],
    expanded: Set<PanelList> = [],
    pendingStop: Set<String> = [],
    copied: String? = nil,
    copiedAt: Date? = nil,
    question: PanelQuestion? = nil,
    questionAt: Date? = nil,
    forcedTooltip: String? = nil,
    stripSeen: Bool = true
  ) {
    self.dismissed = dismissed
    self.expanded = expanded
    self.pendingStop = pendingStop
    self.copied = copied
    self.copiedAt = copiedAt
    self.question = question
    self.questionAt = questionAt
    self.forcedTooltip = forcedTooltip
    self.stripSeen = stripSeen
  }

  /// Whether a list is open.
  public func isOpen(_ list: PanelList) -> Bool { expanded.contains(list) }

  /// The same state with the icon strip gone.
  ///
  /// The four labels under the first document row are there to be read once.
  /// The first click on any icon is the proof they were read, so they go
  /// then and never come back; Task 19 writes the flag down so a later
  /// launch does not show them again.
  public func afterAnIconClick() -> UIState {
    guard !stripSeen else { return self }
    var next = self
    next.stripSeen = true
    return next
  }

  /// The same state with one list the other way round.
  public func toggling(_ list: PanelList) -> UIState {
    var next = self
    if next.expanded.contains(list) {
      next.expanded.remove(list)
    } else {
      next.expanded.insert(list)
    }
    return next
  }

  /// Whether this question is still on the screen. The clock is the
  /// caller's, as every other revert in the panel is.
  public func isAsking(_ asked: PanelQuestion, now: Date) -> Bool {
    guard question == asked else { return false }
    return ConfirmRow.isShowing(askedAt: questionAt, now: now)
  }
}
