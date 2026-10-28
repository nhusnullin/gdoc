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
// A line may open with a verb: NOW's watch is "Live stopped · <name>" and
// its publish is "Publishing · <name>". The verb and the dot after it are
// one piece that is never cut, and the floor and the middle cut are
// measured on the name alone, in the room the verb leaves. Measured on the
// whole line, "Live stopped by you · " was 22 characters before the name
// started and the name was cut to five, while the floor passed on the verb:
// docs/backlog/a-verb-prefix-eats-the-document-name-in-a-row.md. A line
// with no verb in front of it is measured exactly as it was.
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

  /// The room a NOW row's title has, which is what a line with a verb in
  /// front of it is measured against: 320 less the panel's 8 either side,
  /// the row's 12 either side, the 16 pt glyph, the two 8 pt gaps and the
  /// one 24 pt action a live row carries, that action's 4 pt of overhang
  /// given back. 228 pt.
  ///
  /// A watch that cannot be stopped draws no action and has 24 pt more than
  /// this. It is measured at the same 228 all the same: a name that read one
  /// way while Stop was there and another way once it went would be the
  /// panel saying two things about one document.
  public static let nowColumn: CGFloat =
    Grid.panelWidth - 2 * Grid.unit - 2 * Grid.rowInset - Grid.glyphSize - 2 * Grid.columnGap
    - Grid.actionSize + Grid.actionOverhang

  /// The name, which is the piece the floor and the cut are measured on.
  public let title: String
  /// The verb in front of the name, drawn whole with the separator after
  /// it. `nil` is a line that is a name and nothing else.
  public let prefix: String?
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
    _ title: String, prefix: String? = nil, role: TypeRole = Typography.rowTitle,
    column: CGFloat = MiddleTitle.narrowestColumn,
    gone: Bool = false, quietInk: Swatch? = nil
  ) {
    self.title = title
    self.prefix = prefix
    self.role = role
    self.column = column
    self.gone = gone
    self.quietInk = quietInk
  }

  /// The verb as it is drawn: the word and the separator after it, one
  /// piece the cut never touches.
  ///
  /// These two put the pieces back together rather than measure anything,
  /// and a model holds them: `LiveRowModel.title` is read where the panel is
  /// built, which is off the main actor, and a view's own members are on it.
  public nonisolated static func lede(_ prefix: String) -> String {
    prefix + Sentences.separator
  }

  /// The whole line, which is what a tooltip and a reader of the label get.
  public nonisolated static func whole(prefix: String?, name: String) -> String {
    guard let prefix, !prefix.isEmpty else { return name }
    return lede(prefix) + name
  }

  /// The room the name has: the line's own column, less the verb and the
  /// separator where there is one.
  public static func roomAfter(
    _ prefix: String?, in column: CGFloat, role: TypeRole = Typography.rowTitle
  ) -> CGFloat {
    guard let prefix, !prefix.isEmpty else { return column }
    return column - width(lede(prefix), role: role)
  }

  /// The split a row draws: the counted rule, then the measured floor.
  ///
  /// When the floor refuses the tail the whole title becomes the head, which
  /// is what `TT()` does with a tail of its own that is too long.
  public static func fitted(
    _ title: String, prefix: String? = nil, role: TypeRole = Typography.rowTitle,
    column: CGFloat = MiddleTitle.narrowestColumn
  ) -> Titles.Split {
    let column = roomAfter(prefix, in: column, role: role)
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
    _ title: String, prefix: String? = nil, role: TypeRole = Typography.rowTitle,
    column: CGFloat = MiddleTitle.narrowestColumn
  ) -> Int {
    let column = roomAfter(prefix, in: column, role: role)
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
      .accessibilityLabel(Self.whole(prefix: prefix, name: title))
  }

  /// The verb, the head and the tail, which is what one line of a row gets.
  private var split: some View {
    let pieces = Self.fitted(title, prefix: prefix, role: role, column: column)
    return HStack(spacing: 0) {
      if let prefix, !prefix.isEmpty {
        Text(Self.lede(prefix))
          .lineLimit(1)
          .fixedSize(horizontal: true, vertical: false)
      }
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
