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
// angle the renderer reached, and the next render would catch another.
//
// GlyphTests states the two sizes and the three cases that do not spin.

import SwiftUI

/// Whether the view tree is being drawn into an image rather than onto a
/// screen. A render has one frame, so nothing in it animates.
private struct PresenceRenderModeKey: EnvironmentKey {
  static let defaultValue = false
}

extension EnvironmentValues {
  public var presenceRenderMode: Bool {
    get { self[PresenceRenderModeKey.self] }
    set { self[PresenceRenderModeKey.self] = newValue }
  }
}

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
  @Environment(\.presenceRenderMode) private var renderMode
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

  public var body: some View {
    let palette = Palette.of(scheme)
    let fraction = min(max(progress ?? Self.indeterminateFraction, 0), 1)
    let spinning = progress == nil && Self.spins(reduceMotion: reduceMotion, renderMode: renderMode)
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
        .rotationEffect(.degrees(angle))
    }
    .frame(width: size.diameter, height: size.diameter)
    .accessibilityHidden(true)
    .onAppear {
      guard spinning else { return }
      withAnimation(.linear(duration: Self.turn).repeatForever(autoreverses: false)) {
        angle = 360
      }
    }
  }
}
