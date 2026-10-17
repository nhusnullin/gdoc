// The ring: the panel's one moving part.
//
// `.ring` in the screens file is a conic gradient behind a radial mask, which
// in SwiftUI is a stroked circle with a trim. Two sizes and no third: the
// small one sits in the glyph column of a step row, the large one in the job
// card. A caller passes a fraction it knows, or nothing at all, and nothing
// in between.
//
// A ring with no fraction turns, at the stylesheet's 28 percent and 0.9
// seconds a turn. It stops for a person who asked for less motion, and it
// stops in a render: a PNG of a turning ring is a frame caught at whatever
// angle the renderer reached, and the next render would catch another. It
// stops again the moment a fraction arrives, which is a job whose first step
// line reaches a ring already on the screen: the turn is read from where the
// ring stands and not from the angle its animation is at.
//
// GlyphTests states the two sizes and the three cases that do not spin.

import SwiftUI

/// A progress ring, or an indeterminate one.
public struct RingView: View {
  /// `.ring` and `.ring.lg`.
  public enum Size: String, Sendable, CaseIterable {
    case small
    case large

    /// `--size`. The small ring stands in the 16 px glyph column; the large
    /// one is the job card's own.
    public var diameter: CGFloat {
      switch self {
      case .small: return Grid.glyphSize
      case .large: return Grid.ringLarge
      }
    }

    /// `--stroke`.
    public var stroke: CGFloat {
      switch self {
      case .small: return 1.5
      case .large: return 2
      }
    }
  }

  /// `.ring.wait`'s `--p:28`: how much of the circle an indeterminate ring
  /// carries while it turns.
  public static let indeterminateFraction: Double = 0.28

  /// `animation:spin .9s linear infinite`: one turn, in seconds.
  public static let turn: Double = 0.9

  /// A fraction from 0 to 1, or nothing for the indeterminate ring.
  public let progress: Double?
  public let size: Size

  @Environment(\.colorScheme) private var scheme
  /// How this tree is drawn. It is the one render mode `rendering(_:)` sets,
  /// so a scene's PNG and a scene's panel agree about it: a second key of
  /// its own was a flag nothing outside a test ever set.
  @Environment(\.renderMode) private var mode
  @Environment(\.accessibilityReduceMotion) private var reduceMotion
  @State private var angle: Double = 0

  public init(progress: Double? = nil, size: Size = .small) {
    self.progress = progress
    self.size = size
  }

  /// The `@media (prefers-reduced-motion:reduce)` block, and the renderer.
  public static func spins(reduceMotion: Bool, renderMode: Bool) -> Bool {
    !reduceMotion && !renderMode
  }

  /// How far round the arc is drawn: the turn where the ring turns, and
  /// nothing where it has a fraction to stand at.
  ///
  /// The turn is a repeating animation started once, and one ring outlives
  /// the reason it was indeterminate: a publish whose first `step` line
  /// arrives after its start keeps its place in the job card's tree. So the
  /// rotation is read from where the ring stands now rather than from the
  /// angle the animation reached: testARingWithAFractionCarriesNoRotation.
  public static func rotation(angle: Double, spinning: Bool) -> Double {
    spinning ? angle : 0
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    let fraction = min(max(progress ?? Self.indeterminateFraction, 0), 1)
    let spinning =
      progress == nil && Self.spins(reduceMotion: reduceMotion, renderMode: mode != .live)
    let line = StrokeStyle(lineWidth: size.stroke, lineCap: .butt)
    return ZStack {
      Circle()
        .inset(by: size.stroke / 2)
        .stroke(palette.ringTrack.color, style: line)
      Circle()
        .inset(by: size.stroke / 2)
        .trim(from: 0, to: fraction)
        .stroke(palette.text.color, style: line)
        // The conic gradient starts at twelve o'clock; a trim starts at three.
        .rotationEffect(.degrees(-90))
        .rotationEffect(.degrees(Self.rotation(angle: angle, spinning: spinning)))
    }
    .frame(width: size.diameter, height: size.diameter)
    .accessibilityHidden(true)
    .onAppear { spin(spinning) }
    .onChange(of: spinning) { _, now in spin(now) }
  }

  /// Start the turn, or take it off the angle again.
  ///
  /// A fraction that arrives after the ring is up needs the repeating
  /// animation stopped and not merely hidden: an animation left running
  /// keeps redrawing a view that no longer moves.
  private func spin(_ on: Bool) {
    guard on else {
      var still = Transaction(animation: nil)
      still.disablesAnimations = true
      withTransaction(still) { angle = 0 }
      return
    }
    withAnimation(.linear(duration: Self.turn).repeatForever(autoreverses: false)) {
      angle = 360
    }
  }
}
