// Whether this tree is on the screen or on its way into an image.
//
// One thing in the panel moves, the ring, and a render has one frame: an
// indeterminate ring drawn through `ImageRenderer` without this flag comes
// out at whatever angle the animation was at, so two renders of one scene
// are two different PNGs and a reviewer comparing them is comparing noise.
// `rendering()` says so once at the top of a tree, and `RingView` is the one
// view that reads it.
//
// It was three cases until 2026-10-18: live, clipped and whole, because the
// panel was capped at 600 pt and a reviewer needed the uncapped copy of a
// screen whose foot was under the cut. Nothing is capped now and nothing
// scrolls, so the two images were the same image and one of them went with
// the cap.
//
// GlyphTests reads this file's own key in the ring's source, so a second
// key of the ring's own is a failure rather than a scene that spins.

import SwiftUI

/// How a tree is being drawn.
public enum RenderMode: Sendable {
  /// On the screen, where the ring turns.
  case live
  /// Into an image, which is one frame: nothing animates.
  case image
}

private struct RenderModeKey: EnvironmentKey {
  static let defaultValue = RenderMode.live
}

extension EnvironmentValues {
  /// How this tree is drawn.
  public var renderMode: RenderMode {
    get { self[RenderModeKey.self] }
    set { self[RenderModeKey.self] = newValue }
  }

  /// True while a render is in progress.
  public var isRendering: Bool { renderMode != .live }
}

extension View {
  /// Draw this tree for an image: nothing in it animates.
  public func rendering() -> some View {
    environment(\.renderMode, .image)
  }
}
