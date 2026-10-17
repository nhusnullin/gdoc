// The panel's own line: the mark, the name, the account.
//
// `.panel-header` in docs/design/2026-10-17-presence-screens-v3.html is
// 36 pt tall and holds three things and no action: the mark in effect, the
// word gdoc, and the signed-in address pushed to the right in the quietest
// grey in the panel. It is the only place the account is said at this size,
// because whose account it is decides nothing: identity is never a gate.
//
// The mark is a setting, ⁘ by default, and may be empty, so the header takes
// it as a string and draws whatever it is handed.
//
// RowTests states the height and the three pieces.

import SwiftUI

/// The header line.
public struct PanelHeader: View {
  /// The mark in effect, which the binary's settings decide and this view
  /// never guesses.
  public let mark: String
  /// `.panel-header .name`, which is the product's name and nothing else.
  public let name: String
  /// The address, or nothing when there is no token to read one from.
  public let account: String?

  @Environment(\.colorScheme) private var scheme

  public init(mark: String, name: String = "gdoc", account: String? = nil) {
    self.mark = mark
    self.name = name
    self.account = account
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return HStack(spacing: Grid.columnGap) {
      if !mark.isEmpty {
        Text(mark)
          .font(Typography.mark.font)
          .foregroundStyle(palette.text.color)
          .metricProbe("header.mark")
      }
      Text(name)
        .font(Typography.titleEmphasis.font)
        .foregroundStyle(palette.text.color)
        .metricProbe("header.name")
      Spacer(minLength: Grid.columnGap)
      if let account {
        Text(account)
          .font(Typography.caption.font)
          .lineLimit(1)
          .truncationMode(.tail)
          .foregroundStyle(palette.textTertiary.color)
          .metricProbe("header.account")
      }
    }
    .padding(.horizontal, Grid.rowInset)
    .frame(maxWidth: .infinity, minHeight: Grid.headerHeight, alignment: .leading)
    .padding(.bottom, 4)
    .metricProbe("header")
  }
}
