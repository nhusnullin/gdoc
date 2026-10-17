// The status glyph and the ring, against the stylesheet they came from.
//
// The glyph is the one place colour lives, so what is under test is which
// colour each kind and tone draws in, and that every one of the forty-two
// pairs puts ink on the canvas, except the blank which is there to hold a
// column open. The numbers are literals read by hand out of the `.glyph` and
// `.ring` rules of docs/design/2026-10-17-presence-screens-v3.html.
//
// The sheet of all forty-two is written as a PNG outside the repository, so a
// person can look at the shapes once rather than trust a pixel count.

import AppKit
import SwiftUI
import XCTest

@testable import PresenceUI

@MainActor
final class GlyphTests: XCTestCase {
  // Every kind in every tone draws, and each pair is a shape rather than an
  // empty box. The blank is the exception on purpose: a HEALTH row with
  // nothing to say in its glyph column still keeps the column, which the
  // screens file does with `border-color:transparent`.
  func testEveryGlyphKindAndToneRenders() throws {
    XCTAssertEqual(GlyphKind.allCases.count, 7, "the blueprint says seven kinds")
    XCTAssertEqual(GlyphTone.allCases.count, 6, "the blueprint says six tones")

    // `.glyph` is 16 px square; `.screen.light .glyph` is the one rule that
    // changes its border-width.
    XCTAssertEqual(StatusGlyph.size, 16)
    XCTAssertEqual(StatusGlyph.stroke(for: .dark), 1.25)
    XCTAssertEqual(StatusGlyph.stroke(for: .light), 1.5)

    for scheme in [ColorScheme.dark, .light] {
      for kind in GlyphKind.allCases {
        for tone in GlyphTone.allCases {
          let inked = try inkedPixels(StatusGlyph(kind, tone: tone), scheme: scheme)
          if kind == .blank {
            XCTAssertEqual(inked, 0, "a blank glyph draws nothing, in \(scheme) and \(tone)")
          } else {
            XCTAssertGreaterThan(inked, 8, "\(kind) in \(tone) drew nothing in \(scheme)")
          }
        }
      }
    }

    try writeSheet()
  }

  // Principle 5: a write that could not be read back is a hollow check, not a
  // failure. The shape is the same check the green one draws, and the tone is
  // the panel's own secondary text, which is what `.glyph` is before any
  // `.glyph-*` tone class touches it.
  func testTheHollowCheckIsNeutral() {
    XCTAssertEqual(StatusGlyph(.check).tone, .quiet, "a glyph with no tone named is the quiet one")

    for palette in [Palette.dark, Palette.light] {
      let hollow = try? XCTUnwrap(StatusGlyph.ink(kind: .check, tone: .quiet, in: palette))
      XCTAssertEqual(hollow, palette.textSecondary, "the hollow check is --t2, the CSS's default")
      XCTAssertNotEqual(hollow, palette.ok, "a hollow check is not a green one")
      XCTAssertNotEqual(hollow, palette.bad, "and it is not a failure either")

      XCTAssertEqual(
        StatusGlyph.ink(kind: .check, tone: .ok, in: palette), palette.ok,
        "a verified write is the green check")
      XCTAssertEqual(
        StatusGlyph.ink(kind: .empty, tone: .ok, in: palette), palette.textTertiary,
        "`.glyph-empty` sets its own border colour, so the tone never reaches it")
      XCTAssertNil(
        StatusGlyph.ink(kind: .blank, tone: .bad, in: palette),
        "a blank glyph has no ink at all, whatever tone it is handed")
    }

    // The five tone classes of the stylesheet, and the untoned default.
    XCTAssertEqual(GlyphTone.ok.swatch(in: .dark), Palette.dark.ok)
    XCTAssertEqual(GlyphTone.hold.swatch(in: .dark), Palette.dark.hold)
    XCTAssertEqual(GlyphTone.bad.swatch(in: .dark), Palette.dark.bad)
    XCTAssertEqual(GlyphTone.live.swatch(in: .dark), Palette.dark.text)
    XCTAssertEqual(GlyphTone.dim.swatch(in: .dark), Palette.dark.textTertiary)
    XCTAssertEqual(GlyphTone.quiet.swatch(in: .dark), Palette.dark.textSecondary)
  }

  // `.ring` and `.ring.lg`, and nothing between them: a job's ring is the
  // large one, a step row's is the small one, and no caller passes a number.
  func testTheRingIsTwoSizes() throws {
    XCTAssertEqual(RingView.Size.allCases.count, 2, "two sizes, not a free diameter")
    XCTAssertEqual(RingView.Size.small.diameter, 16)
    XCTAssertEqual(RingView.Size.small.stroke, 1.5)
    XCTAssertEqual(RingView.Size.large.diameter, 28)
    XCTAssertEqual(RingView.Size.large.stroke, 2)

    let small = try inkedPixels(
      RingView(progress: 0.66, size: .small), scheme: .dark, side: RingView.Size.large.diameter)
    let large = try inkedPixels(
      RingView(progress: 0.66, size: .large), scheme: .dark, side: RingView.Size.large.diameter)
    XCTAssertGreaterThan(large, small, "the large ring covers more of the canvas than the small")

    // The track is drawn whatever the progress, so an empty ring is still a
    // ring and never a hole in the row.
    XCTAssertGreaterThan(
      try inkedPixels(RingView(progress: 0, size: .small), scheme: .dark), 8,
      "a ring at nought still draws its track")
  }

  // `.ring.wait` spins at 28 percent, and the stylesheet's reduced-motion
  // block stops it. A render is the same stop: a PNG of a spinning ring is a
  // ring frozen at whatever angle the renderer caught, which is a screenshot
  // nobody can compare with the next one.
  func testAnIndeterminateRingIsStaticUnderReduceMotionAndInRenderMode() throws {
    XCTAssertEqual(RingView.indeterminateFraction, 0.28)
    XCTAssertEqual(RingView.turn, 0.9)

    XCTAssertTrue(RingView.spins(reduceMotion: false, renderMode: false))
    XCTAssertFalse(
      RingView.spins(reduceMotion: true, renderMode: false),
      "a person who asked for less motion gets none")
    XCTAssertFalse(
      RingView.spins(reduceMotion: false, renderMode: true),
      "a render draws one frame, so there is nothing to spin")
    XCTAssertFalse(RingView.spins(reduceMotion: true, renderMode: true))

    // Twice through the renderer, byte for byte the same, so a scene's PNG
    // does not change under a reviewer who opened it a second time.
    let first = try png(
      RingView(size: .large).environment(\.presenceRenderMode, true), scheme: .dark)
    let second = try png(
      RingView(size: .large).environment(\.presenceRenderMode, true), scheme: .dark)
    XCTAssertEqual(first, second, "an indeterminate ring in render mode is one fixed frame")
    XCTAssertGreaterThan(
      try inkedPixels(
        RingView(size: .large).environment(\.presenceRenderMode, true), scheme: .dark,
        side: RingView.Size.large.diameter),
      8, "the indeterminate ring draws its arc and its track")
  }

  // MARK: the renders

  /// The sheet: seven kinds down, six tones across, in both appearances, so
  /// the shapes are looked at once rather than counted.
  private func writeSheet() throws {
    let shots = try shotsDirectory()
    for scheme in [ColorScheme.dark, .light] {
      let palette = Palette.of(scheme)
      let sheet = VStack(alignment: .leading, spacing: 12) {
        ForEach(GlyphKind.allCases, id: \.self) { kind in
          HStack(spacing: 12) {
            ForEach(GlyphTone.allCases, id: \.self) { tone in
              StatusGlyph(kind, tone: tone)
            }
            RingView(progress: 0.66, size: .small)
          }
        }
        HStack(spacing: 12) {
          RingView(progress: 0, size: .large)
          RingView(progress: 0.5, size: .large)
          RingView(progress: 1, size: .large)
          RingView(size: .large).environment(\.presenceRenderMode, true)
        }
      }
      .padding(16)
      .background(palette.surface.color)

      let data = try png(sheet, scheme: scheme)
      let name = scheme == .dark ? "glyphs-dark.png" : "glyphs-light.png"
      try data.write(to: shots.appendingPathComponent(name))
      XCTAssertGreaterThan(data.count, 1_000, "\(name) is too small to be the sheet")
    }
  }

  /// One view through the renderer at 2x, in the appearance named.
  private func png<V: View>(_ view: V, scheme: ColorScheme) throws -> Data {
    let renderer = ImageRenderer(content: view.environment(\.colorScheme, scheme))
    renderer.scale = 2
    var data: Data?
    let appearance = NSAppearance(named: scheme == .dark ? .darkAqua : .aqua)
    appearance?.performAsCurrentDrawingAppearance {
      guard let image = renderer.cgImage else { return }
      data = NSBitmapImageRep(cgImage: image).representation(using: .png, properties: [:])
    }
    return try XCTUnwrap(data, "the renderer drew nothing in \(scheme)")
  }

  /// How many pixels carry ink. The view is laid out in a square of its own
  /// size over nothing, so a transparent pixel is a pixel nothing drew on.
  private func inkedPixels<V: View>(
    _ view: V, scheme: ColorScheme, side: CGFloat = StatusGlyph.size
  ) throws -> Int {
    let framed = view.frame(width: side, height: side)
    let renderer = ImageRenderer(content: framed.environment(\.colorScheme, scheme))
    renderer.scale = 2
    var counted = 0
    var failure: String?
    let appearance = NSAppearance(named: scheme == .dark ? .darkAqua : .aqua)
    appearance?.performAsCurrentDrawingAppearance {
      guard let image = renderer.cgImage else {
        failure = "the renderer drew nothing"
        return
      }
      let rep = NSBitmapImageRep(cgImage: image)
      for x in 0..<rep.pixelsWide {
        for y in 0..<rep.pixelsHigh {
          if let colour = rep.colorAt(x: x, y: y), colour.alphaComponent > 0.05 { counted += 1 }
        }
      }
    }
    if let failure { XCTFail(failure) }
    return counted
  }

  /// Where the PNGs go: outside the repository, always, because a committed
  /// screenshot is a file nobody compares against the code that drew it.
  private func shotsDirectory() throws -> URL {
    let named = ProcessInfo.processInfo.environment["PRESENCE_SHOTS"]
    let temporary = URL(fileURLWithPath: NSTemporaryDirectory())
      .appendingPathComponent("gdoc-presence-shots")
    let base = named.map { URL(fileURLWithPath: $0) } ?? temporary
    let repository = packageRoot().deletingLastPathComponent().standardizedFileURL.path
    XCTAssertFalse(
      base.standardizedFileURL.path.hasPrefix(repository),
      "PRESENCE_SHOTS names \(base.path), which is inside the repository")
    try FileManager.default.createDirectory(at: base, withIntermediateDirectories: true)
    return base
  }

  /// This file is Tests/PresenceUITests/GlyphTests.swift, so the package is
  /// three directories up from it.
  private func packageRoot() -> URL {
    URL(fileURLWithPath: #filePath)
      .deletingLastPathComponent()
      .deletingLastPathComponent()
      .deletingLastPathComponent()
  }
}
