// The panel with nothing in it: the mark, and one sentence.
//
// `.empty` in docs/design/2026-10-17-presence-screens-v3.html is the one
// centred thing in the panel: the mark at 28 pt and a sentence under it, with
// 28 pt of air above and 20 below. It is scene e5, the first open of a
// machine that has never signed in, and the sentence does one job: it names
// the icon to press. There is no large filled button, because one action is
// one control, and that control is the sign-in icon in the header.
//
// The mark is a setting and may be empty, so the view takes it as a string
// and draws whatever it is handed.
//
// CardTests states the first-run sentence as a literal.

import SwiftUI

/// The empty panel.
public struct EmptyState: View {
  /// The one sentence the first open says. It names where the control is
  /// rather than describing a state, because a person reading it has
  /// nothing to do yet and one thing to find. It said "the Account row
  /// below" until 2026-10-18, when the account moved into the header.
  public nonisolated static let firstRun =
    "Not signed in. Sign in with the Google account that will post replies: "
    + "the Sign in icon at the top."

  /// `.empty{padding:28px 12px 20px}` and `.empty .mark{margin-bottom:8px}`.
  public static let topPadding: CGFloat = 28
  public static let bottomPadding: CGFloat = 20
  public static let markGap: CGFloat = Grid.unit

  /// The mark in effect, which the binary's settings decide.
  public let mark: String
  public let words: String

  @Environment(\.colorScheme) private var scheme

  public init(mark: String, words: String) {
    self.mark = mark
    self.words = words
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return VStack(spacing: 0) {
      if !mark.isEmpty {
        Text(mark)
          .font(Typography.markLarge.font)
          .foregroundStyle(palette.text.color)
          .padding(.bottom, Self.markGap)
          .metricProbe("empty.mark")
      }
      Text(words)
        .font(Typography.rowValue.font)
        .multilineTextAlignment(.center)
        .fixedSize(horizontal: false, vertical: true)
        .foregroundStyle(palette.textSecondary.color)
    }
    .frame(maxWidth: .infinity)
    .padding(.horizontal, Grid.rowInset)
    .padding(.top, Self.topPadding)
    .padding(.bottom, Self.bottomPadding)
    .metricProbe("empty")
  }
}
