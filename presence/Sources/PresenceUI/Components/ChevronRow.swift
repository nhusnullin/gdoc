// The row that opens a longer list, and closes it again.
//
// `.row-chevron` in docs/design/2026-10-17-presence-screens-v3.html is the
// quiet 28 pt row of `.row-more` with a 12 pt chevron on the right, turned
// over when the list is open. It is the panel's one disclosure: everything
// else is either on the screen or behind an action.
//
// The chevron is 12 pt and not 16, which is `.row-chevron svg` against
// `.act svg`, and the screens file wins on a pixel.
//
// RowTests states the height and the turn.

import SwiftUI

/// One disclosure row.
public struct ChevronRow: View {
  public let title: String
  public let isOpen: Bool
  public let action: () -> Void

  @Environment(\.colorScheme) private var scheme

  public init(_ title: String, isOpen: Bool = false, action: @escaping () -> Void = {}) {
    self.title = title
    self.isOpen = isOpen
    self.action = action
  }

  /// `.row-chevron.is-open svg{transform:rotate(180deg)}`.
  public static func rotation(isOpen: Bool) -> Angle {
    .degrees(isOpen ? 180 : 0)
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return Button(action: action) {
      RowView(
        .more, title: title,
        leading: { EmptyView() },
        trailing: {
          IconView(.chevron, ink: palette.textTertiary)
            .rotationEffect(Self.rotation(isOpen: isOpen))
            .metricProbe("chevron")
        })
    }
    .buttonStyle(.plain)
    .accessibilityLabel(title)
  }
}
