// A document's title in a row: the end kept, the middle cut.
//
// `.row-title` in docs/design/2026-10-17-presence-screens-v3.html is a flex
// box of two pieces: `.head`, which may be shortened with an ellipsis, and
// `.tail`, which is `flex:none;white-space:pre` and so is never touched.
// `Titles.split` decides where the two meet.
//
// The one rule this file adds to that one is measured: a tail wide enough to
// leave fewer than eight characters of head is dropped, and the title goes
// in whole with an ellipsis, because two characters and a word is not a
// title anybody can read. The width it is measured against is the narrowest
// title column in the panel, the document row's, so one title reads the same
// in every section it appears in.
//
// One line of a row is the whole of this file. Where a card's title takes
// two lines, which is StuckCard's own rule, each of them is drawn by this
// one at its own width: a line left to the system's middle truncation cut
// inside a word and showed "her regions" of "further regions".
//
// RowTests holds the floor; TitlesTests holds the split.

import AppKit
import PresenceCore
import SwiftUI

/// One row's title.
public struct MiddleTitle: View {
  /// The narrowest title column in the panel: the document row's middle
  /// cell. 320 less the panel's 8 either side, less the row's 12 either
  /// side, less the 16 pt glyph, the two 8 pt gaps and the 108 pt action
  /// column.
  public static let narrowestColumn: CGFloat =
    Grid.panelWidth - 2 * Grid.unit - 2 * Grid.rowInset - Grid.glyphSize - 2 * Grid.columnGap
    - Grid.actionColumn

  public let title: String
  public let role: TypeRole
  /// The room the line has. A row's is the narrowest column in the panel,
  /// so one title reads the same in every section; a card's two lines are
  /// each their own width.
  public let column: CGFloat
  public let gone: Bool
  /// The colour the title is drawn in when it is not gone. `nil` is
  /// `.row-title`'s own `--t1`; the quiet rows hand over `--t3`.
  public let quietInk: Swatch?

  @Environment(\.colorScheme) private var scheme

  public init(
    _ title: String, role: TypeRole = Typography.rowTitle,
    column: CGFloat = MiddleTitle.narrowestColumn,
    gone: Bool = false, quietInk: Swatch? = nil
  ) {
    self.title = title
    self.role = role
    self.column = column
    self.gone = gone
    self.quietInk = quietInk
  }

  /// The split a row draws: the counted rule, then the measured floor.
  ///
  /// When the floor refuses the tail the whole title becomes the head, which
  /// is what `TT()` does with a tail of its own that is too long.
  public static func fitted(
    _ title: String, role: TypeRole = Typography.rowTitle,
    column: CGFloat = MiddleTitle.narrowestColumn
  ) -> Titles.Split {
    let split = Titles.split(title)
    guard let tail = split.tail else { return split }
    let room = column - width(tail, role: role)
    if width(split.head, role: role) <= room { return split }

    // The head is cut, so the floor is the floor plus the three dots after
    // it, and one character more. The dots are drawn inside the head's own
    // room, and the row's two pieces are laid out in whole points, so a
    // floor that fits by a hair is drawn one character short: scene r9's
    // "B2B2C payment card concept: the frozen vocabulary" measured eight and
    // drew seven.
    let floor = String(split.head.prefix(Titles.headFloor + 1)) + ellipsis
    guard width(floor, role: role) <= room else { return Titles.Split(head: title) }
    return split
  }

  /// One string's width in the panel's own font, tracking counted.
  static func width(_ text: String, role: TypeRole) -> CGFloat {
    let font = NSFont.systemFont(ofSize: role.size, weight: role.weight.appKit)
    let attributed = NSAttributedString(
      string: text, attributes: [.font: font, .kern: role.letterSpacing])
    return attributed.size().width
  }

  /// The three dots a truncated head ends with. They are drawn inside the
  /// head's own room, so every measurement of what fits counts them.
  static let ellipsis = "…"

  /// How many characters of the head a column shows, which is what the
  /// floor is a floor on.
  ///
  /// A head that fits whole shows all of itself. A head that does not is
  /// drawn with the ellipsis after it, and the ellipsis takes room from the
  /// characters, so it is measured with them.
  static func headCharacters(
    _ title: String, role: TypeRole = Typography.rowTitle,
    column: CGFloat = MiddleTitle.narrowestColumn
  ) -> Int {
    let split = fitted(title, role: role, column: column)
    let room = column - width(split.tail ?? "", role: role)
    if width(split.head, role: role) <= room { return split.head.count }

    var shown = 0
    for count in 1...max(split.head.count, 1) {
      let drawn = String(split.head.prefix(count)) + ellipsis
      guard width(drawn, role: role) <= room else { break }
      shown = count
    }
    return shown
  }

  /// `.row-title.gone{text-decoration:line-through;color:var(--t2)}`: a
  /// document that is in the trash keeps its row and loses its title.
  public static func ink(gone: Bool, quiet: Swatch? = nil, in palette: Palette) -> Swatch {
    if gone { return palette.textSecondary }
    return quiet ?? palette.text
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return
      split
      .font(role.font)
      .tracking(role.letterSpacing)
      .strikethrough(gone)
      .foregroundStyle(Self.ink(gone: gone, quiet: quietInk, in: palette).color)
      .accessibilityLabel(title)
  }

  /// The head and the tail, which is what one line of a row gets.
  private var split: some View {
    let pieces = Self.fitted(title, role: role, column: column)
    return HStack(spacing: 0) {
      Text(pieces.head)
        .lineLimit(1)
        .truncationMode(.tail)
      if let tail = pieces.tail {
        Text(tail)
          .lineLimit(1)
          .fixedSize(horizontal: true, vertical: false)
      }
    }
  }
}

extension TypeWeight {
  /// The same weight as AppKit states it, for the one measurement this
  /// package makes.
  var appKit: NSFont.Weight {
    switch self {
    case .regular: return .regular
    case .medium: return .medium
    case .semibold: return .semibold
    }
  }
}
