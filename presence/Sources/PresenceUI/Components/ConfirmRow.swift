// An inline confirm: the question, and the pair that answers it.
//
// `.confirm` in docs/design/2026-10-17-presence-screens-v3.html is two short
// buttons 2 pt apart where a row's actions would be, and the row's own title
// becomes the question. There is no sheet and no alert: a menu bar panel that
// puts a modal window on the screen has lost the thing it was for.
//
// It waits for its answer. The question goes with Keep, with the confirm, or
// with the panel closing, which is the Keep nobody pressed; the clock never
// takes it away. Until 2026-10-18 it reverted after three seconds, and a
// person reading the sentence under "Sign out?" found the question gone and
// the icon back under the pointer, so the second press asked again and the
// sign-out never ran.
//
// Two confirms exist: Sign out, and the hold card's Dismiss. Stop live has
// none, because stopping a watch is cheap and reversible.
//
// ControlTests states the wait, the gap and the two heights.

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

  /// Whether the question is on the screen: asked, and not in the future.
  ///
  /// A question asked in the future is no question on the screen: a clock
  /// that stepped backwards is not a person asking. Nothing else ends it but
  /// an answer or the panel closing: testAConfirmRowWaitsForItsAnswer.
  public nonisolated static func isShowing(askedAt: Date?, now: Date) -> Bool {
    guard let askedAt else { return false }
    return now.timeIntervalSince(askedAt) >= 0
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
