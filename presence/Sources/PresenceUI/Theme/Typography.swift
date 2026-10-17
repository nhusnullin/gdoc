// The twelve type roles of the panel.
//
// Two weights carry the whole panel, 400 and 500, and the light appearance
// asks for a third on one label only. A role is a size, a weight, a line
// height and a tracking read off the screens file's stylesheet; a view names
// the role and never a number.

import SwiftUI

/// The weights the panel uses. The third is the light section label's alone.
public enum TypeWeight: Sendable, Equatable {
  case regular
  case medium
  case semibold

  public var font: Font.Weight {
    switch self {
    case .regular: return .regular
    case .medium: return .medium
    case .semibold: return .semibold
    }
  }
}

/// One role: the stylesheet's `font` shorthand, kept whole.
///
/// `tracking` is in em as the CSS writes it, and `letterSpacing` is the same
/// value in points, which is what SwiftUI takes.
public struct TypeRole: Sendable, Equatable {
  public let size: CGFloat
  public let weight: TypeWeight
  public let lineHeight: CGFloat
  public let tracking: CGFloat
  public let monospaced: Bool
  public let uppercase: Bool

  public init(
    size: CGFloat, weight: TypeWeight, lineHeight: CGFloat, tracking: CGFloat = 0,
    monospaced: Bool = false, uppercase: Bool = false
  ) {
    self.size = size
    self.weight = weight
    self.lineHeight = lineHeight
    self.tracking = tracking
    self.monospaced = monospaced
    self.uppercase = uppercase
  }

  public var font: Font {
    .system(size: size, weight: weight.font, design: monospaced ? .monospaced : .default)
  }

  public var letterSpacing: CGFloat { tracking * size }

  /// The same role in another weight, which the light section label is the
  /// one caller of.
  public func weighted(_ weight: TypeWeight) -> TypeRole {
    TypeRole(
      size: size, weight: weight, lineHeight: lineHeight, tracking: tracking,
      monospaced: monospaced, uppercase: uppercase)
  }
}

/// The roles, named for what they carry.
public enum Typography {
  /// The mark in the panel header, `.panel-header .mark`.
  public static let mark = TypeRole(size: 16, weight: .medium, lineHeight: 16)
  /// The mark of the empty state, `.empty .mark`.
  public static let markLarge = TypeRole(size: 28, weight: .medium, lineHeight: 28)
  /// The header's name, a card's title and a job's title.
  public static let titleEmphasis = TypeRole(size: 13, weight: .medium, lineHeight: 16)
  /// A row's title, which is the panel's own font.
  public static let rowTitle = TypeRole(
    size: 13, weight: .regular, lineHeight: 16, tracking: -0.006)
  /// A section's label. The light appearance takes it a weight heavier.
  public static let sectionLabel = TypeRole(
    size: 11, weight: .medium, lineHeight: 11, tracking: 0.06, uppercase: true)
  /// A row's value on the right, and the size the quiet line and the more
  /// row share with it.
  public static let rowValue = TypeRole(size: 12, weight: .regular, lineHeight: 16)
  /// A sub-line, the header's account and a tooltip's words.
  public static let caption = TypeRole(size: 11, weight: .regular, lineHeight: 14)
  /// A button's label, the only text a person presses.
  public static let button = TypeRole(
    size: 12, weight: .medium, lineHeight: 12, tracking: -0.005)
  /// The menu bar badge beside the mark.
  public static let badge = TypeRole(size: 11, weight: .medium, lineHeight: 11)
  /// The sign-in URL, the one monospaced line in the panel.
  public static let mono = TypeRole(
    size: 11, weight: .regular, lineHeight: 14, monospaced: true)
  /// The icon strip's labels under a row.
  public static let strip = TypeRole(size: 10, weight: .regular, lineHeight: 12)
  /// The signed-in account, said once at its full size.
  public static let bigMail = TypeRole(size: 15, weight: .medium, lineHeight: 20, tracking: -0.01)

  /// Every role by name, so a test can count them and a reviewer can read
  /// the sheet in one place.
  public static let roles: [String: TypeRole] = [
    "mark": mark,
    "markLarge": markLarge,
    "titleEmphasis": titleEmphasis,
    "rowTitle": rowTitle,
    "sectionLabel": sectionLabel,
    "rowValue": rowValue,
    "caption": caption,
    "button": button,
    "badge": badge,
    "mono": mono,
    "strip": strip,
    "bigMail": bigMail,
  ]

  /// The section label, which is the one role the appearance changes: a
  /// medium label on black reads as a semibold label does on white.
  public static func sectionLabel(in scheme: ColorScheme) -> TypeRole {
    scheme == .dark ? sectionLabel : sectionLabel.weighted(.semibold)
  }
}
