// The panel's sizes, on the 8 pt grid.
//
// Named Grid because PanelView still holds M23's `Metrics` until the plain
// panel goes. Every number is the screens file's stylesheet, so a layout
// names a field here and never writes a number of its own.

import CoreGraphics

/// The sizes the panel is laid out on.
public enum Grid {
  /// `.panel`'s width, its padding and its corner.
  public static let panelWidth: CGFloat = 320
  public static let unit: CGFloat = 8
  public static let panelRadius: CGFloat = 12

  /// The heights: the header, a section label, and a row in its three forms.
  public static let headerHeight: CGFloat = 36
  /// The gap the header leaves under itself, which is part of what the
  /// header's own probe reports.
  public static let headerGap: CGFloat = 4
  public static let sectionLabelHeight: CGFloat = 28
  public static let rowHeight: CGFloat = 32
  public static let twoLineRowHeight: CGFloat = 44
  public static let compactRowHeight: CGFloat = 28

  /// The document row's third column, wide enough for the four icons: the
  /// three that open or copy something and the pin at the end of them.
  ///
  /// It was 136 until the fifth look (docs/v2/DECISIONS.md, 2026-10-28).
  /// The Google Docs icon went when the row itself became that press, and
  /// the 28 pt it held went to the title:
  /// testTheDocRowActionColumnIs108.
  public static let actionColumn: CGFloat = 108

  /// The glyph, an action button and the radius they share.
  public static let glyphSize: CGFloat = 16
  public static let actionSize: CGFloat = 24
  public static let controlRadius: CGFloat = 6
  /// `.acts{margin-right:-4px}`: an action's artwork is 16 pt centred in a
  /// 24 pt box, so a strip whose boxes end on the content edge draws its
  /// icons 4 pt inside it. The strip is pulled back by that much, and the
  /// panel keeps one right edge for its values, its times and its icons:
  /// testAnActionStripMeetsTheColumnEveryValueEndsOn.
  public static let actionOverhang: CGFloat = (actionSize - glyphSize) / 2

  /// A row's own inset and the gaps inside it.
  public static let rowInset: CGFloat = 12
  public static let columnGap: CGFloat = 8
  public static let actionGap: CGFloat = 4

  /// A card: the hold's and the stuck's plate.
  public static let cardRadius: CGFloat = 8
  public static let cardInset: CGFloat = 10

  /// The large ring, which a running job carries.
  public static let ringLarge: CGFloat = 28

  /// One device pixel on a retina screen, which every hairline is.
  public static let hairline: CGFloat = 0.5

  /// The left inset of a line under a row, past the glyph column.
  public static let quietInset: CGFloat = 44

  /// The badge beside the mark in the menu bar.
  public static let badgeSize: CGFloat = 10

  /// The dot on the header's update glyph while a release is there to take.
  /// It is the menu bar badge's own size taken down to what reads beside a
  /// 16 pt icon, and it is drawn in the accent ink the icon takes:
  /// testTheUpdateGlyphCarriesItsDotOnlyWhenThereIsSomethingToTake.
  public static let updateDotSize: CGFloat = 5
}
