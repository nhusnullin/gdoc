// The five labels under the first document row, once.
//
// `.strip` in docs/design/2026-10-17-presence-screens-v3.html is words in
// 10 pt, each 24 pt wide, sitting under the icons of the document row above.
// It is the whole of this app's onboarding: one person does not need a tour,
// he needs to not guess once. It goes after the first click on any icon and
// never comes back, which the app remembers.
//
// The labels are an enum, because one word under each icon is a thing that
// stays as long as the icons do. It was four words until the third look
// added the pin (docs/v2/DECISIONS.md, 2026-10-18).
//
// CardTests states the words and the cell width.

import SwiftUI

/// The five words, in the order the icons stand in.
public enum StripLabel: String, Sendable, CaseIterable {
  /// Open in Google Docs.
  case doc = "Doc"
  /// Open the note in Obsidian.
  case note = "Note"
  /// Continue with Claude.
  case chat = "Chat"
  /// Copy link.
  case link = "Link"
  /// Pin the document, or take the pin back.
  case pin = "Pin"
}

/// The strip.
public struct IconStrip: View {
  /// `.strip span{width:24px;text-align:center}`: one cell per action
  /// button, so a word sits under its own icon.
  public static let cell: CGFloat = Grid.actionSize
  /// `.strip{gap:4px}`, which is the gap between the icons it labels.
  public static let gap: CGFloat = Grid.actionGap
  /// `padding:0 12px 6px 44px`.
  public static let leadingInset: CGFloat = Grid.quietInset
  public static let bottomPadding: CGFloat = 6

  @Environment(\.colorScheme) private var scheme

  public init() {}

  public var body: some View {
    let palette = Palette.of(scheme)
    return HStack(spacing: Self.gap) {
      Spacer(minLength: 0)
      ForEach(StripLabel.allCases, id: \.rawValue) { label in
        Text(label.rawValue)
          .font(Typography.strip.font)
          .lineLimit(1)
          .multilineTextAlignment(.center)
          .frame(width: Self.cell)
          .metricProbe("strip.\(label.rawValue)")
      }
    }
    .foregroundStyle(palette.textTertiary.color)
    .frame(maxWidth: .infinity, alignment: .trailing)
    .metricProbe("strip")
    .padding(.leading, Self.leadingInset)
    .padding(.trailing, Grid.rowInset)
    .padding(.bottom, Self.bottomPadding)
  }
}
