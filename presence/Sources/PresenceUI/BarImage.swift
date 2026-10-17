// The one thing the menu bar shows.
//
// Six states and never two: a write that stopped, a hold waiting for the
// person, a finite job running, a watch sitting on a document, the five
// seconds after a write read itself back, and nothing happening at all. The
// order between them is the whole of what this file decides, and it is the
// specification's and not this file's reading.
//
// Every image is a template, so macOS draws it in the menu bar's own colour
// in both appearances and in both the normal and the inverted bar. There is
// no colour in the menu bar, ever, which is principle 1. That is why the
// badge is drawn here in one ink at a few opacities rather than reusing the
// panel's `StatusGlyph` and `RingView`, which are the one place colour lives.
//
// The sizes are the `.status-item` rules of
// docs/design/2026-10-17-presence-screens-v3.html: a 15 px mark, a 10 px
// badge, and the gap between them. The shapes are the panel's own, because
// `GlyphGeometry` takes the side it is drawn at.
//
// It knows nothing about a document. The state comes from the same two folds
// the panel is built from, so the bar and the panel cannot disagree: a
// dismissed stuck card clears the exclamation because the fold it came out of
// no longer holds the write. BarTests states every rule and every number.

import AppKit
import PresenceCore
import SwiftUI

/// What the menu bar is showing.
public enum BarState: String, Sendable, Equatable, CaseIterable {
  /// A write that failed or whose process is gone, a token that does not
  /// work, or no gdoc to run at all.
  case stuck
  /// A chat write is held, waiting for a person.
  case hold
  /// A command with an end is running.
  case job
  /// A `comments --wait` is sitting on a document.
  case live
  /// A write read itself back a moment ago.
  case check
  /// Nothing is happening that the record knows about.
  case idle

  /// The badge beside the mark, where the state has one. Live and idle are
  /// the mark alone.
  public var badge: BarBadge? {
    switch self {
    case .stuck: return .glyph(.exclamation)
    case .hold: return .glyph(.pause)
    case .job: return .ring
    case .check: return .glyph(.check)
    case .live, .idle: return nil
    }
  }
}

/// The shape of a badge: one of the panel's glyph marks, or the ring.
public enum BarBadge: Sendable, Equatable {
  case glyph(GlyphKind)
  /// A job's ring. It is one fixed frame rather than the panel's turning
  /// one: a template image in the menu bar has no animation, and the record
  /// carries no steps to turn it through.
  case ring
}

/// How long the check stays, and whether it is showing.
///
/// The check is the one state that is about a moment rather than a state, so
/// it is the one thing the bar remembers. It starts only for a write that
/// arrived while the app was running: every end in the file at launch is a
/// replay, and a replay would flash a check for a write from last Tuesday.
public struct BarCheck: Sendable, Equatable {
  /// Five seconds, which is the banner's life as well.
  public static let lasts: TimeInterval = Timing.checkFlash

  /// The write the check was last considered for, so the same end does not
  /// buy itself a second five seconds on the next redraw.
  private var seen: String?
  private var since: Date?

  public init() {}

  /// A verified write out of the fold, and whether its line arrived live.
  ///
  /// - Returns: whether the check started now, which is what the shell
  ///   hangs its one-shot timer on.
  @discardableResult
  public mutating func saw(_ write: VerifiedWrite?, live: Bool, at now: Date) -> Bool {
    guard let write else { return false }
    let key = "\(write.run)|\(write.call)"
    guard key != seen else { return false }
    seen = key
    guard live else { return false }
    since = now
    return true
  }

  public func isShowing(now: Date) -> Bool {
    guard let since else { return false }
    return now.timeIntervalSince(since) < Self.lasts
  }
}

public enum BarImage {
  /// gdoc's own mark, which is what the bar is built around. It is a literal
  /// here and not the mark in the person's settings: the menu bar is gdoc
  /// saying it is there, and a person who emptied the mark setting still has
  /// a status item to press.
  public static let mark = "⁘"

  /// `.status-item .mark`'s `font-size:15px`.
  public static let markSize: CGFloat = 15

  /// The image's own height. A status item is 22 pt and macOS leaves room
  /// around the image, so 18 is what the bar draws without scaling it.
  public static let height: CGFloat = 18

  /// `.status-item .g`'s and `.ring`'s 10 px.
  public static let badgeSize: CGFloat = 10

  /// `.status-item`'s `gap:3px` and the badge's own `margin-left:2px`.
  public static let badgeGap: CGFloat = 5

  /// `.status-item .g`'s `border-width:1px`.
  public static let badgeStroke: CGFloat = 1

  /// `.status-item .g`'s `opacity:.8` and `.ring`'s `opacity:.9`.
  public static let glyphBadgeOpacity: Double = 0.8
  public static let ringBadgeOpacity: Double = 0.9

  /// `.ring`'s `--stroke:1.25px` at the badge's size.
  public static let ringStroke: CGFloat = 1.25

  /// How much of the ring is drawn where the run says nothing about where
  /// it is, which is `.ring.wait`'s `--p:28` and the panel's own
  /// indeterminate fraction. BarTests holds it to `RingView`'s.
  public static let ringFraction: Double = 0.28

  /// How much of the badge's ring one job draws: the part it is on out of
  /// the parts there are, and the indeterminate fraction for a run the
  /// record names no part of. It is a place in a list and never a guess at
  /// what is left, which is the same rule the panel's own ring follows.
  public static func ringFraction(for progress: Double?) -> Double {
    guard let progress else { return ringFraction }
    return min(max(progress, 0), 1)
  }

  /// The fraction the bar turns at for one fold: the first running job that
  /// says where it is. Nothing else in the panel moves the badge.
  public static func ringProgress(now fold: NowFold) -> Double? {
    fold.jobs.compactMap(\.progress).first
  }

  /// What idle is drawn at. The mark stays there so the panel is reachable
  /// when nothing is happening; the vote's "nothing at all" waits for Nail.
  public static let idleOpacity: Double = 0.45

  /// The state one moment is in.
  ///
  /// The check is first because it is over any state: a write that read
  /// itself back is the one thing a person wants said at the moment it
  /// happens. Under it the order is the specification's: stuck, hold, job,
  /// live, idle.
  ///
  /// A hold whose life ran out is not waiting for anybody, so it is a row in
  /// the panel and never a badge here, for the same reason a start whose pid
  /// is gone is not one. And `facts.network` is never read: a watch that
  /// cannot reach Google is still a watch.
  public static func state(
    now fold: NowFold, sessions: [LiveSession], facts: Facts, checkShowing: Bool
  ) -> BarState {
    if checkShowing { return .check }
    if !fold.stuck.isEmpty || !facts.auth.isWorking || !facts.gdocFound { return .stuck }
    if fold.holds.contains(where: { !$0.ranOut }) { return .hold }
    // A sign-in in flight is a job as well, which is scene e8 of the screens
    // file: the app's own child is a command with an end, and the record
    // carries no line for it. It stays under stuck, because a machine with no
    // token is stuck whatever is in flight.
    if !fold.jobs.isEmpty || facts.signIn != nil { return .job }
    if !fold.watches.isEmpty || sessions.contains(where: \.isOpen) { return .live }
    return .idle
  }

  /// What the whole image is drawn at.
  public static func opacity(for state: BarState) -> Double {
    state == .idle ? idleOpacity : 1
  }

  /// What the pointer resting on the item says. The tooltip is the one
  /// always-visible word, so it names the state and nothing else: no title,
  /// no address and no id, because a menu bar is on screen in every
  /// screen-share.
  public static func tooltip(for state: BarState) -> String {
    switch state {
    case .stuck: return "gdoc is stuck on something"
    case .hold: return "gdoc is holding a write for you"
    case .job: return "gdoc is running"
    case .live: return "gdoc is watching a document"
    case .check: return "gdoc wrote and read it back"
    case .idle: return "gdoc"
    }
  }

  /// The template image for a state.
  ///
  /// Drawn once per state and kept: the item is set again on every refold,
  /// and the six images are the same six pictures every time.
  @MainActor
  public static func image(for state: BarState, progress: Double? = nil) -> NSImage {
    let key = "\(state.rawValue)|\(ringFraction(for: progress))"
    if let drawn = cache[key] { return drawn }
    let renderer = ImageRenderer(content: BarGlyph(state: state, progress: progress))
    renderer.scale = 2
    let image: NSImage
    if let bitmap = renderer.cgImage {
      image = NSImage(
        cgImage: bitmap,
        size: NSSize(
          width: CGFloat(bitmap.width) / renderer.scale,
          height: CGFloat(bitmap.height) / renderer.scale))
    } else {
      image = NSImage(size: NSSize(width: height, height: height))
    }
    image.isTemplate = true
    cache[key] = image
    return image
  }

  /// `.mbar`'s own background, which is the mock's menu bar and not the
  /// panel's palette. It is here because the only thing that draws it is the
  /// render of the bar for a reviewer.
  public static func barSurface(_ scheme: ColorScheme) -> Swatch {
    scheme == .dark ? Swatch(0x1F1F1F) : Swatch(0xFFFFFF, alpha8: 0xB8)
  }

  /// `.mbar`'s `color`, which is what the template image is tinted in.
  public static func barInk(_ scheme: ColorScheme) -> Swatch {
    scheme == .dark ? Swatch(0xFFFFFF, alpha8: 0xD9) : Swatch(0x000000, alpha8: 0xD9)
  }

  /// The images drawn so far, by state and fraction. A job's badge is one
  /// picture per step, which is four for a publish and never more.
  @MainActor private static var cache: [String: NSImage] = [:]
}

/// The bar's image as a view: the mark, and the badge where there is one.
///
/// The ink is a parameter rather than the appearance's, because the image is
/// a template drawn in black and the render for a reviewer is the same shapes
/// tinted the way the menu bar tints them.
public struct BarGlyph: View {
  public let state: BarState
  public let ink: Color
  /// Where the one running job is, for the badge that is a ring.
  public let progress: Double?

  public init(state: BarState, ink: Color = .black, progress: Double? = nil) {
    self.state = state
    self.ink = ink
    self.progress = progress
  }

  public var body: some View {
    HStack(spacing: 0) {
      Text(BarImage.mark)
        .font(.system(size: BarImage.markSize, weight: .medium))
        .foregroundStyle(ink)
      if let badge = state.badge {
        BarBadgeView(badge: badge, ink: ink, progress: progress)
          .padding(.leading, BarImage.badgeGap)
      }
    }
    .frame(height: BarImage.height)
    .opacity(BarImage.opacity(for: state))
    .accessibilityHidden(true)
  }
}

/// One 10 pt badge, in the one ink a template image has.
struct BarBadgeView: View {
  let badge: BarBadge
  let ink: Color
  var progress: Double?

  var body: some View {
    shape
      .frame(width: BarImage.badgeSize, height: BarImage.badgeSize)
  }

  @ViewBuilder private var shape: some View {
    switch badge {
    case .glyph(let kind):
      glyph(kind).opacity(BarImage.glyphBadgeOpacity)
    case .ring:
      ring.opacity(BarImage.ringBadgeOpacity)
    }
  }

  /// The panel's own shapes at the badge's side, which is what
  /// `GlyphGeometry`'s `side` is for.
  private func glyph(_ kind: GlyphKind) -> some View {
    let geometry = GlyphGeometry(side: BarImage.badgeSize, stroke: BarImage.badgeStroke)
    return ZStack {
      if kind.hasCircle {
        geometry.circle.stroke(ink, lineWidth: BarImage.badgeStroke)
      }
      if let filled = geometry.filled(kind) {
        filled.fill(ink)
      }
      if let stroked = geometry.stroked(kind) {
        stroked.stroke(
          ink,
          style: StrokeStyle(
            lineWidth: geometry.markStroke, lineCap: .square, lineJoin: .miter))
      }
    }
  }

  /// The ring at the job's own step, or at one fixed frame where the record
  /// names no step. The track is the same ink at a quarter, so the image
  /// stays one colour.
  private var ring: some View {
    ZStack {
      Circle()
        .inset(by: BarImage.ringStroke / 2)
        .stroke(ink.opacity(0.25), lineWidth: BarImage.ringStroke)
      Circle()
        .inset(by: BarImage.ringStroke / 2)
        .trim(from: 0, to: BarImage.ringFraction(for: progress))
        .stroke(ink, lineWidth: BarImage.ringStroke)
        .rotationEffect(.degrees(-90))
    }
  }
}

/// A cut-out of the menu bar with one state in it, which is what a reviewer
/// looks at. It is drawn and never shown: the app's own item is the template
/// image, tinted by macOS.
public struct BarPreview: View {
  public let state: BarState

  @Environment(\.colorScheme) private var scheme

  public init(state: BarState) {
    self.state = state
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return HStack(spacing: 0) {
      Spacer(minLength: 0)
      BarGlyph(state: state, ink: BarImage.barInk(scheme).color)
        .padding(.horizontal, 8)
    }
    .frame(width: 140, height: 24)
    .background(BarImage.barSurface(scheme).color)
    .background(palette.wall.color)
  }
}
