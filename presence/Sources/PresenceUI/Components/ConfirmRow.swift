// An inline confirm: the question, and the pair that answers it.
//
// `.confirm` in docs/design/2026-10-17-presence-screens-v3.html is two short
// buttons 2 pt apart where a row's actions would be, and the row's own title
// becomes the question. There is no sheet and no alert: a menu bar panel that
// puts a modal window on the screen has lost the thing it was for.
//
// It reverts itself after three seconds. A panel left open with a question in
// it is a panel waiting for an answer nobody is going to give, and the safe
// half of every pair here is the one that does nothing, so letting the time
// run out is the same as pressing Keep.
//
// Two confirms exist: Sign out, and the hold card's Dismiss. Stop live has
// none, because stopping a watch is cheap and reversible.
//
// ControlTests states the three seconds, the gap and the two heights.

import SwiftUI

/// One inline confirm.
public struct ConfirmRow: View {
  /// `.confirm{gap:2px}`.
  public static let gap: CGFloat = 2

  public let title: String
  public let sub: String?
  public let glyph: GlyphKind
  public let tone: GlyphTone
  public let danger: ButtonWord
  /// The question's colour where it is asked inside a tint, which is the
  /// hold card's: the plate has its own ink on white, and the panel's
  /// neutral greys on it are two ink systems on one tint.
  public let titleInk: Swatch?
  public let onConfirm: () -> Void
  public let onKeep: () -> Void

  public init(
    title: String,
    sub: String? = nil,
    glyph: GlyphKind = .empty,
    tone: GlyphTone = .quiet,
    danger: ButtonWord,
    titleInk: Swatch? = nil,
    onConfirm: @escaping () -> Void = {},
    onKeep: @escaping () -> Void = {}
  ) {
    self.title = title
    self.sub = sub
    self.glyph = glyph
    self.tone = tone
    self.danger = danger
    self.titleInk = titleInk
    self.onConfirm = onConfirm
    self.onKeep = onKeep
  }

  /// Whether the question is still on the screen.
  ///
  /// The clock is the caller's, so the revert is a value a test can state
  /// rather than a timer it has to wait for.
  /// A question asked in the future is no question on the screen: a clock
  /// that stepped backwards would otherwise leave it up for ever.
  public nonisolated static func isShowing(askedAt: Date?, now: Date) -> Bool {
    guard let askedAt else { return false }
    let since = now.timeIntervalSince(askedAt)
    return since >= 0 && since < Timing.confirmRevert
  }

  public var body: some View {
    // A question with no second line is a one-line row. The two-line shape
    // reserved the sub-line it was not drawing, which left a 21 pt hole
    // between the hold card's question and the consequence under it:
    // testTheConfirmDrawsNoLineItHasNoWordsFor.
    RowView(
      sub == nil ? .plain : .two, glyph: glyph, tone: tone, title: title, sub: sub,
      titleInk: titleInk,
      trailing: {
        HStack(spacing: Self.gap) {
          TextButton(danger, kind: .danger, size: .compact, action: onConfirm)
            .metricProbe("confirm.danger")
          TextButton(.keep, kind: .plain, size: .compact, action: onKeep)
            .metricProbe("confirm.keep")
        }
      })
  }
}
