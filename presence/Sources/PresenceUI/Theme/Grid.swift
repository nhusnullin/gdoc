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
  /// The cap the expanded sections scroll inside, rather than growing past.
  public static let panelMaxHeight: CGFloat = 600

  /// The heights: the header, a section label, and a row in its three forms.
  public static let headerHeight: CGFloat = 36
  /// The gap the header leaves under itself, which the body has to take off
  /// the cap as well: a body that only subtracts the header is laid out this
  /// much past the frame, and the clip takes it off the bottom edge.
  public static let headerGap: CGFloat = 4
  public static let sectionLabelHeight: CGFloat = 28
  public static let rowHeight: CGFloat = 32
  public static let twoLineRowHeight: CGFloat = 44
  public static let compactRowHeight: CGFloat = 28

  /// The document row's third column, wide enough for the four icons.
  public static let actionColumn: CGFloat = 108

  /// The glyph, an action button and the radius they share.
  public static let glyphSize: CGFloat = 16
  /// The pin, and the slot a document row keeps for it in front of the
  /// glyph. Every document row keeps it, pinned or not, so the titles in
  /// the list break at the same place whatever is pinned.
  public static let pinSize: CGFloat = 10
  public static let actionSize: CGFloat = 24
  public static let controlRadius: CGFloat = 6

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

  /// The scroll box: its own cap, and the fade at its foot.
  public static let scrollBoxHeight: CGFloat = 236
  public static let fadeHeight: CGFloat = 28

  /// The badge beside the mark in the menu bar.
  public static let badgeSize: CGFloat = 10
}
