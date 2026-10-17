// The status glyph: a 16 px circle with one mark inside it.
//
// This is the one place colour lives, which is principle 2, so the glyph is
// seven shapes by six tones and nothing else in the panel is coloured. It is
// drawn with Path rather than a system symbol, because the shapes are the
// `.glyph` rules of docs/design/2026-10-17-presence-screens-v3.html and a
// symbol would be whatever the platform ships this year.
//
// The CSS numbers are kept as CSS numbers. Every rule there is written for a
// 16 px border-box, so the geometry below is in those units and the one
// conversion is the final scale. The stroke of the circle is the appearance's
// (1.25 on black, 1.5 on white), and because `.screen.light` overrides only
// `.glyph`'s own border-width, the marks inside keep their 1.25 in both.
//
// GlyphTests states every colour rule and renders all forty-two pairs.

import SwiftUI

/// The seven shapes, each the stylesheet's class.
public enum GlyphKind: String, Sendable, CaseIterable {
  /// `.glyph` with no kind class: the circle alone, in the tone's colour.
  case plain
  /// `.glyph-empty`: the circle alone, always tertiary, for a step that has
  /// not happened and a document with nothing yet.
  case empty
  /// `.glyph-check`: the check. Green when a write was read back, quiet when
  /// it was not, which is the hollow check of principle 5.
  case check
  /// `.glyph-excl`: the bar and its dot, which is the only thing that is red.
  case exclamation
  /// `.glyph-pause`: the two bars of a hold waiting for the person.
  case pause
  /// `.glyph-dot`: the filled dot of a live watch, with no circle around it.
  case dot
  /// `.glyph` with its border-color transparent: nothing at all, so a row
  /// with no state keeps the column the others are aligned on.
  case blank

  /// Whether the circle is drawn.
  var hasCircle: Bool {
    switch self {
    case .plain, .empty, .check, .exclamation, .pause: return true
    case .dot, .blank: return false
    }
  }
}

/// The six tones: the five `.glyph-*` colour classes and the untoned default.
public enum GlyphTone: String, Sendable, CaseIterable {
  /// `.glyph`'s own `color:var(--t2)`, which no class sets and the hollow
  /// check draws in.
  case quiet
  /// `.glyph-ok`
  case ok
  /// `.glyph-hold`
  case hold
  /// `.glyph-bad`
  case bad
  /// `.glyph-live`, which is --t1: a live watch is the brightest thing in the
  /// panel without being a colour.
  case live
  /// `.glyph-dim`, which is --t3: something that is over.
  case dim

  public func swatch(in palette: Palette) -> Swatch {
    switch self {
    case .quiet: return palette.textSecondary
    case .ok: return palette.ok
    case .hold: return palette.hold
    case .bad: return palette.bad
    case .live: return palette.text
    case .dim: return palette.textTertiary
    }
  }
}

/// One status glyph.
public struct StatusGlyph: View {
  /// `.glyph`'s width and height.
  public static let size: CGFloat = Grid.glyphSize

  public let kind: GlyphKind
  public let tone: GlyphTone

  @Environment(\.colorScheme) private var scheme

  public init(_ kind: GlyphKind, tone: GlyphTone = .quiet) {
    self.kind = kind
    self.tone = tone
  }

  /// `.glyph`'s border-width, which the light appearance is the one override
  /// of, so a hairline circle on white reads as it does on black.
  public static func stroke(for scheme: ColorScheme) -> CGFloat {
    Palette.of(scheme).glyphStroke
  }

  /// What the glyph is drawn in, or nothing at all.
  ///
  /// The tone never reaches `.glyph-empty`, which sets its own border colour
  /// in the stylesheet, and never reaches a blank, which draws nothing.
  public static func ink(kind: GlyphKind, tone: GlyphTone, in palette: Palette) -> Swatch? {
    switch kind {
    case .blank: return nil
    case .empty: return palette.textTertiary
    default: return tone.swatch(in: palette)
    }
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    let stroke = Self.stroke(for: scheme)
    let geometry = GlyphGeometry(side: Self.size, stroke: stroke)
    return ZStack {
      if let ink = Self.ink(kind: kind, tone: tone, in: palette) {
        if kind.hasCircle {
          geometry.circle.stroke(ink.color, lineWidth: stroke)
        }
        if let filled = geometry.filled(kind) {
          filled.fill(ink.color)
        }
        if let stroked = geometry.stroked(kind) {
          stroked.stroke(
            ink.color,
            style: StrokeStyle(
              lineWidth: geometry.markStroke, lineCap: .square, lineJoin: .miter))
        }
      }
    }
    .frame(width: Self.size, height: Self.size)
    .accessibilityHidden(true)
  }
}

/// The shapes, in the stylesheet's own units.
///
/// A CSS absolute child sits inside the border, so every number below is in
/// the padding box and the circle's stroke is added back. `side` is there so
/// the menu bar's smaller glyph can take the same shapes.
struct GlyphGeometry {
  let side: CGFloat
  let stroke: CGFloat

  /// 16, the box every number here was written for.
  private static let authoredSide: CGFloat = 16

  /// The border-width of a mark, which the light appearance leaves alone.
  private static let authoredMarkStroke: CGFloat = 1.25

  private var scale: CGFloat { side / Self.authoredSide }

  /// The stroke of the check, scaled with the box.
  var markStroke: CGFloat { Self.authoredMarkStroke * scale }

  /// The width and height of the padding box, which the marks are placed in.
  private var inner: CGFloat { Self.authoredSide - 2 * authoredStroke }

  /// The circle's border-width in authored units.
  private var authoredStroke: CGFloat { stroke / scale }

  /// `border-radius:50%` with the border drawn inside the box.
  var circle: Path {
    let inset = authoredStroke / 2
    return scaled(
      Path(
        ellipseIn: CGRect(
          x: inset, y: inset, width: Self.authoredSide - 2 * inset,
          height: Self.authoredSide - 2 * inset)))
  }

  /// The marks that are a background rather than a border: the exclamation's
  /// bar and dot, the pause's two bars, and the live dot.
  func filled(_ kind: GlyphKind) -> Path? {
    switch kind {
    case .exclamation:
      var path = bar(x: 6.4, y: 3)
      path.addEllipse(in: padded(CGRect(x: 6.4, y: 9.5, width: 1.25, height: 1.25)))
      return scaled(path)
    case .pause:
      var path = bar(x: 5, y: 4.5)
      // `right:5px` is the right edge of the padding box, so the left edge
      // moves with the circle's border-width.
      path.addPath(bar(x: inner - 5 - 1.25, y: 4.5))
      return scaled(path)
    case .dot:
      // `inset:5px` inside the padding box, which is where the transparent
      // border leaves it.
      return scaled(
        Path(ellipseIn: padded(CGRect(x: 5, y: 5, width: inner - 10, height: inner - 10))))
    case .plain, .empty, .check, .blank:
      return nil
    }
  }

  /// The marks that are a border: the check, which the stylesheet draws as
  /// the right and bottom edges of a 3 by 6 box turned 45 degrees.
  func stroked(_ kind: GlyphKind) -> Path? {
    guard kind == .check else { return nil }
    let box = CGRect(x: 4.5, y: 2.5, width: 3, height: 6)
    let inset = Self.authoredMarkStroke / 2
    var path = Path()
    path.move(to: CGPoint(x: box.maxX - inset, y: box.minY))
    path.addLine(to: CGPoint(x: box.maxX - inset, y: box.maxY - inset))
    path.addLine(to: CGPoint(x: box.minX, y: box.maxY - inset))
    let centre = CGPoint(x: box.midX, y: box.midY)
    let turned = path.applying(
      CGAffineTransform(translationX: centre.x, y: centre.y)
        .rotated(by: .pi / 4)
        .translatedBy(x: -centre.x, y: -centre.y))
    return scaled(turned.applying(paddingOffset))
  }

  /// One rounded bar: 1.25 by 5 with `border-radius:1px`.
  private func bar(x: CGFloat, y: CGFloat) -> Path {
    Path(
      roundedRect: padded(CGRect(x: x, y: y, width: 1.25, height: 5)),
      cornerRadius: 1)
  }

  /// A rectangle written in padding-box coordinates, moved into the box.
  private func padded(_ rect: CGRect) -> CGRect {
    rect.applying(paddingOffset)
  }

  private var paddingOffset: CGAffineTransform {
    CGAffineTransform(translationX: authoredStroke, y: authoredStroke)
  }

  private func scaled(_ path: Path) -> Path {
    scale == 1 ? path : path.applying(CGAffineTransform(scaleX: scale, y: scale))
  }
}
