// The action icons: the screens file's SVG, drawn as Path.
//
// Ten of the twelve shapes are the `IC` object of
// docs/design/2026-10-17-presence-screens-v3.html, kept as the `d` strings
// that file holds and turned into a Path here. The last two are the pin,
// which the file has no shape for: it is a pair, the outline and the filled,
// and both are written here in the same grammar so they can be read beside
// the others. They are not SF Symbols: four of them stand side by side in a
// document row, their weights have to match, and a system symbol is whatever
// the platform ships this year.
//
// Keeping the `d` string means a shape can be compared with the line it came
// from by eye. The cost is the small parser below, which understands the
// subset those strings use: M, L, H, V, C and A in both cases, Z, and the
// implicit repeat. No exponent and no rotated ellipse, because the file has
// neither; an input it does not understand draws nothing rather than
// something wrong, and IconTests fails on an empty icon.
//
// IconTests states the sizes, the stroke and the caps, and renders the sheet.

import SwiftUI

/// The twelve icons the panel draws.
///
/// The `IC` object holds more than these: the ones Nail did not choose are
/// not ported, so a name here is an icon some screen uses.
public enum Icon: String, Sendable, CaseIterable {
  /// Open the note in Obsidian.
  case md
  /// Continue with Claude.
  case branch
  /// Copy link.
  case link
  /// The check Copy link becomes for a second, and nothing else.
  case done
  /// Stop watching.
  case stop
  /// Update to the version the row names.
  case down
  /// Sign in.
  case signin
  /// Sign out.
  case signout
  /// Cancel, in a circle: the sign-in a person gives up on.
  case x
  /// Dismiss, bare: the stuck card's last icon.
  case xmark
  /// Pin this document: the last icon of a document row, with its head
  /// hollow because the document is not pinned.
  case pin
  /// Unpin it: the same tack with its head filled, which is the whole of
  /// what the row says about being pinned.
  case pinOn

  /// The CSS width of this icon, which is also its viewBox.
  ///
  /// `.act svg` is 16, and every icon the panel draws is one of those since
  /// 2026-10-18: the 12 pt one was `.row-chevron svg`, and the row that
  /// opened a longer list went with the lists that had one.
  public var side: CGFloat { 16 }

  /// The elements this icon fills rather than strokes, which is the pinned
  /// tack's head and nothing else in the set.
  ///
  /// A pair of shapes, one outlined and one filled, is how a row says pinned
  /// without a word and without moving anything. Filling is the pin's own
  /// because an action icon is a 1.25 pt line everywhere else, and a line is
  /// the one weight the four icons of a row share:
  /// testThePinIsOutlinedAndFilledAndNeitherIsACircle.
  var solidParts: [IconPart] {
    guard self == .pinOn else { return [] }
    return [.roundedRect(x: 4, y: 3, width: 8, height: 3, radius: 1.5)]
  }

  /// The SVG elements of this icon, in the order the file writes them.
  var parts: [IconPart] {
    switch self {
    case .md:
      return [
        .path("M4.5 2h7.5a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1H4.5a1 1 0 0 1-1-1V3a1 1 0 0 1 1-1z"),
        .path("M2.5 5h3M2.5 8h3M2.5 11h3"),
        .path("M7.5 6.5h3M7.5 9.5h3"),
      ]
    case .branch:
      return [
        .path(
          "M4 2.5h8a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2H7.5L4.5 14v-2.5H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2z"
        ),
        .path("M8 4.25C8 6.25 9 7 11 7C9 7 8 7.75 8 9.75C8 7.75 7 7 5 7C7 7 8 6.25 8 4.25z"),
      ]
    case .link:
      return [
        .path(
          "M5.5 5.5V3.5a1.5 1.5 0 0 1 1.5-1.5h5.5A1.5 1.5 0 0 1 14 3.5V9a1.5 1.5 0 0 1-1.5 1.5h-2"),
        .path(
          "M3.5 5.5h5.5A1.5 1.5 0 0 1 10.5 7v5.5A1.5 1.5 0 0 1 9 14H3.5A1.5 1.5 0 0 1 2 12.5V7a1.5 1.5 0 0 1 1.5-1.5z"
        ),
      ]
    case .done:
      return [
        .circle(x: 8, y: 8, radius: 6.25),
        .path("M5.25 8.25l1.9 1.9 3.6-4"),
      ]
    case .stop:
      return [
        .circle(x: 8, y: 8, radius: 6.25),
        .roundedRect(x: 5.5, y: 5.5, width: 5, height: 5, radius: 1),
      ]
    case .down:
      return [
        .circle(x: 8, y: 8, radius: 6.25),
        .path("M8 4.75v6.5M5.25 8.5 8 11.25l2.75-2.75"),
      ]
    case .signin:
      return [
        .path("M9 2.75h3a1.5 1.5 0 0 1 1.5 1.5v7.5a1.5 1.5 0 0 1-1.5 1.5H9"),
        .path("M2.5 8h7M7 5.5 9.5 8 7 10.5"),
      ]
    case .signout:
      return [
        .path("M7 2.75H4a1.5 1.5 0 0 0-1.5 1.5v7.5a1.5 1.5 0 0 0 1.5 1.5h3"),
        .path("M6.5 8h7M11 5.5 13.5 8 11 10.5"),
      ]
    case .x:
      return [
        .circle(x: 8, y: 8, radius: 6.25),
        .path("M5.75 5.75l4.5 4.5M10.25 5.75l-4.5 4.5"),
      ]
    case .xmark:
      return [.path("M4.5 4.5l7 7M11.5 4.5l-7 7")]
    case .pin, .pinOn:
      // A tack seen from the side: a flat head and the needle under it. Two
      // shapes are all that survive at this size, where a thumbtack's three
      // pieces close into a blot, and the head is a bar rather than a ring
      // because a ring on a stem reads as a magnifying glass.
      //
      // Both states draw this, and the pinned one fills the head as well.
      // The outline is kept under the fill so the two tacks have the one
      // silhouette: a press darkens the head and moves nothing else on the
      // row: testThePinIsOutlinedAndFilledAndNeitherIsACircle.
      return [
        .roundedRect(x: 4, y: 3, width: 8, height: 3, radius: 1.5),
        .path("M8 6V13"),
      ]
    }
  }
}

/// One element of an icon's SVG.
enum IconPart {
  case path(String)
  case circle(x: CGFloat, y: CGFloat, radius: CGFloat)
  case roundedRect(x: CGFloat, y: CGFloat, width: CGFloat, height: CGFloat, radius: CGFloat)

  var path: Path {
    switch self {
    case .path(let d):
      return SVGPath.parse(d) ?? Path()
    case .circle(let x, let y, let radius):
      return Path(
        ellipseIn: CGRect(
          x: x - radius, y: y - radius, width: radius * 2, height: radius * 2))
    case .roundedRect(let x, let y, let width, let height, let radius):
      return Path(
        roundedRect: CGRect(x: x, y: y, width: width, height: height), cornerRadius: radius)
    }
  }
}

/// One icon as a shape, in the box it is given: its stroked half, or its
/// filled one where it has any.
public struct IconShape: Shape {
  /// `.act svg`'s stroke-width, the same in both appearances.
  public static let stroke: CGFloat = 1.25

  /// `stroke-linecap:round;stroke-linejoin:round`.
  public static let style = StrokeStyle(lineWidth: stroke, lineCap: .round, lineJoin: .round)

  public let icon: Icon
  /// Whether this is the icon's filled half. Only the pinned tack has one,
  /// and it is empty for every other icon, which draws nothing.
  let solid: Bool

  public init(_ icon: Icon) {
    self.init(icon, solid: false)
  }

  init(_ icon: Icon, solid: Bool) {
    self.icon = icon
    self.solid = solid
  }

  public func path(in rect: CGRect) -> Path {
    var whole = Path()
    for part in solid ? icon.solidParts : icon.parts {
      whole.addPath(part.path)
    }
    let scale = min(rect.width, rect.height) / icon.side
    let drawn = icon.side * scale
    return whole.applying(
      CGAffineTransform(
        translationX: rect.minX + (rect.width - drawn) / 2,
        y: rect.minY + (rect.height - drawn) / 2
      ).scaledBy(x: scale, y: scale))
  }
}

/// One icon drawn in the colour the caller names, at its own size.
///
/// The ink is handed in rather than taken from the palette, because an
/// action's own grey is not every grey a 16 pt mark is drawn in: the
/// header's update glyph has two of its own.
public struct IconView: View {
  public let icon: Icon
  public let ink: Swatch

  public init(_ icon: Icon, ink: Swatch) {
    self.icon = icon
    self.ink = ink
  }

  public var body: some View {
    // The two shapes are drawn opaque and the ink's own alpha is put on the
    // pair of them, because an icon is one mark in one ink. Drawn as two
    // translucent layers the filled tack's head composites to
    // `1-(1-a)²`, which is 0.49 where the action ink is 0.29: the pinned
    // row carried the brightest icon in the panel, and the dimmed tack was
    // brighter than a live icon beside it:
    // testThePinnedTackIsOneLayerOfItsOwnInk.
    ZStack {
      IconShape(icon).stroke(opaque.color, style: IconShape.style)
      if !icon.solidParts.isEmpty {
        IconShape(icon, solid: true).fill(opaque.color)
      }
    }
    .compositingGroup()
    .opacity(ink.alpha)
    .frame(width: icon.side, height: icon.side)
    .accessibilityHidden(true)
  }

  /// The ink with its alpha taken off, to be put back on the group.
  private var opaque: Swatch { Swatch(ink.hex) }
}

// MARK: the parser

/// The slice of the SVG path grammar the screens file writes.
enum SVGPath {
  /// The `d` attribute as a Path, or nothing if a character in it was not
  /// one of the commands this understands.
  static func parse(_ d: String) -> Path? {
    var scanner = Scanner(d)
    var path = Path()
    var current = CGPoint.zero
    var subpathStart = CGPoint.zero
    var command: Character?

    while !scanner.atEnd() {
      if scanner.nextIsNumber() {
        guard let repeated = command else { return nil }
        // A close takes no numbers, so a number after one repeats a command
        // that would consume nothing and the loop would never end. Nothing
        // drawn is the answer to a path this does not understand, and a
        // hang is worse than nothing: testANumberAfterACloseDrawsNothing.
        guard repeated != "Z", repeated != "z" else { return nil }
        // A second pair after a moveto is a lineto, which is how
        // "M12.5 2.5 8 7" draws its diagonal.
        command = repeated == "M" ? "L" : (repeated == "m" ? "l" : repeated)
      } else {
        guard let letter = scanner.letter() else { return nil }
        command = letter
      }
      guard let letter = command else { return nil }

      switch letter {
      case "M", "m":
        guard let point = scanner.point(from: current, relative: letter == "m") else {
          return nil
        }
        current = point
        subpathStart = point
        path.move(to: point)
      case "L", "l":
        guard let point = scanner.point(from: current, relative: letter == "l") else {
          return nil
        }
        current = point
        path.addLine(to: point)
      case "H", "h":
        guard let value = scanner.number() else { return nil }
        current = CGPoint(x: letter == "h" ? current.x + value : value, y: current.y)
        path.addLine(to: current)
      case "V", "v":
        guard let value = scanner.number() else { return nil }
        current = CGPoint(x: current.x, y: letter == "v" ? current.y + value : value)
        path.addLine(to: current)
      case "C", "c":
        let relative = letter == "c"
        guard let one = scanner.point(from: current, relative: relative),
          let two = scanner.point(from: current, relative: relative),
          let end = scanner.point(from: current, relative: relative)
        else { return nil }
        path.addCurve(to: end, control1: one, control2: two)
        current = end
      case "A", "a":
        guard let rx = scanner.number(), let ry = scanner.number(),
          scanner.number() != nil,
          let large = scanner.number(), let sweep = scanner.number(),
          let end = scanner.point(from: current, relative: letter == "a")
        else { return nil }
        addArc(
          &path, from: current, to: end, rx: rx, ry: ry, largeArc: large != 0, sweep: sweep != 0)
        current = end
      case "Z", "z":
        path.closeSubpath()
        current = subpathStart
      default:
        return nil
      }
    }
    return path
  }

  /// An elliptical arc as cubic curves, which is the one conversion here.
  ///
  /// The endpoint form the file writes is turned into a centre and two
  /// angles the way the SVG specification sets out, then cut into pieces of
  /// a quarter turn or less, each of which a cubic follows closely enough.
  /// The x-axis rotation is not read: no icon in the file turns one.
  private static func addArc(
    _ path: inout Path, from start: CGPoint, to end: CGPoint, rx: CGFloat, ry: CGFloat,
    largeArc: Bool, sweep: Bool
  ) {
    var rx = abs(rx)
    var ry = abs(ry)
    guard rx > 0, ry > 0, start != end else {
      path.addLine(to: end)
      return
    }

    let halfX = (start.x - end.x) / 2
    let halfY = (start.y - end.y) / 2
    let oversize = halfX * halfX / (rx * rx) + halfY * halfY / (ry * ry)
    if oversize > 1 {
      let grow = sqrt(oversize)
      rx *= grow
      ry *= grow
    }

    let denominator = rx * rx * halfY * halfY + ry * ry * halfX * halfX
    guard denominator > 0 else {
      path.addLine(to: end)
      return
    }
    let numerator = max(0, rx * rx * ry * ry - denominator)
    let sign: CGFloat = largeArc == sweep ? -1 : 1
    let factor = sign * sqrt(numerator / denominator)
    let centre = CGPoint(
      x: factor * rx * halfY / ry + (start.x + end.x) / 2,
      y: -factor * ry * halfX / rx + (start.y + end.y) / 2)

    let from = atan2((start.y - centre.y) / ry, (start.x - centre.x) / rx)
    let to = atan2((end.y - centre.y) / ry, (end.x - centre.x) / rx)
    var sweptAngle = to - from
    if sweep, sweptAngle < 0 { sweptAngle += 2 * .pi }
    if !sweep, sweptAngle > 0 { sweptAngle -= 2 * .pi }

    let pieces = max(1, Int(ceil(abs(sweptAngle) / (.pi / 2))))
    let step = sweptAngle / CGFloat(pieces)
    let pull = 4.0 / 3.0 * tan(step / 4)
    var angle = from
    for _ in 0..<pieces {
      let next = angle + step
      let head = CGPoint(x: centre.x + rx * cos(angle), y: centre.y + ry * sin(angle))
      let tail = CGPoint(x: centre.x + rx * cos(next), y: centre.y + ry * sin(next))
      path.addCurve(
        to: tail,
        control1: CGPoint(x: head.x - pull * rx * sin(angle), y: head.y + pull * ry * cos(angle)),
        control2: CGPoint(x: tail.x + pull * rx * sin(next), y: tail.y - pull * ry * cos(next)))
      angle = next
    }
  }

  /// A reader over the `d` string: separators, numbers and command letters.
  private struct Scanner {
    private let characters: [Character]
    private var index: Int

    init(_ text: String) {
      characters = Array(text)
      index = 0
    }

    mutating func atEnd() -> Bool {
      skipSeparators()
      return index >= characters.count
    }

    mutating func nextIsNumber() -> Bool {
      skipSeparators()
      guard index < characters.count else { return false }
      let character = characters[index]
      return character.isNumber || character == "-" || character == "+" || character == "."
    }

    mutating func letter() -> Character? {
      skipSeparators()
      guard index < characters.count, characters[index].isLetter else { return nil }
      defer { index += 1 }
      return characters[index]
    }

    mutating func number() -> CGFloat? {
      skipSeparators()
      var text = ""
      if index < characters.count, characters[index] == "-" || characters[index] == "+" {
        text.append(characters[index])
        index += 1
      }
      while index < characters.count {
        let character = characters[index]
        if character.isNumber {
          text.append(character)
        } else if character == ".", !text.contains(".") {
          text.append(character)
        } else {
          break
        }
        index += 1
      }
      guard let value = Double(text) else { return nil }
      return CGFloat(value)
    }

    mutating func point(from current: CGPoint, relative: Bool) -> CGPoint? {
      guard let x = number(), let y = number() else { return nil }
      return relative ? CGPoint(x: current.x + x, y: current.y + y) : CGPoint(x: x, y: y)
    }

    private mutating func skipSeparators() {
      while index < characters.count,
        characters[index] == " " || characters[index] == "," || characters[index] == "\n"
          || characters[index] == "\t"
      {
        index += 1
      }
    }
  }
}
