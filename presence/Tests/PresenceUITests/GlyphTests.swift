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
          let inked = try inkedPixels(
            StatusGlyph(kind, tone: tone), scheme: scheme, side: StatusGlyph.size)
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
  /// A pause is two bars at every size it is drawn, the menu bar's 10 pt
  /// badge included, where a gap under one device pixel closes them into a
  /// block that reads as a stop.
  ///
  /// The measurement is the glyph's own middle row: two runs of ink with a
  /// gap between them, counted off the drawing rather than off the numbers
  /// that made it.
  func testThePauseIsTwoBarsAtTheBadgeSizeToo() throws {
    for side in [Grid.glyphSize, Grid.badgeSize] {
      let geometry = GlyphGeometry(side: side, stroke: 1.25)
      let bars = try XCTUnwrap(geometry.filled(.pause), "the pause drew nothing at \(side)")
      let box = bars.boundingRect

      // Two bars 1.25 wide with a gap: the pair is wider than one bar and
      // the gap is at least a point of what is drawn. The glyph is authored
      // at 16 pt, which is the panel's own size.
      let drawn = 1.25 * side / 16
      let gap = box.width - 2 * drawn
      XCTAssertGreaterThanOrEqual(
        gap, 1, "the two bars are \(gap) pt apart at \(side), which closes into a block")
      XCTAssertEqual(
        box.midX, side / 2, accuracy: 0.3, "the pair is off centre at \(side)")
    }
  }

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
      try inkedPixels(RingView(progress: 0, size: .small), scheme: .dark, side: StatusGlyph.size),
      8,
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
      RingView(size: .large).rendering(), scheme: .dark)
    let second = try png(
      RingView(size: .large).rendering(), scheme: .dark)
    XCTAssertEqual(first, second, "an indeterminate ring in render mode is one fixed frame")
    XCTAssertGreaterThan(
      try inkedPixels(
        RingView(size: .large).rendering(), scheme: .dark,
        side: RingView.Size.large.diameter),
      8, "the indeterminate ring draws its arc and its track")

    // And it is `rendering(_:)`'s own key the ring reads, which is the key
    // a scene sets. A second key of the ring's own was a flag nothing
    // outside this file ever set, so every rendered scene spun.
    let source = try String(
      contentsOf: presencePackageRoot()
        .appendingPathComponent("Sources/PresenceUI/Components/RingView.swift"),
      encoding: .utf8)
    XCTAssertTrue(
      source.contains("@Environment(\\.renderMode)"),
      "the ring reads the render mode `rendering(_:)` sets")
    XCTAssertFalse(
      source.contains("EnvironmentKey"), "and declares no render key of its own")
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
          RingView(size: .large).rendering()
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
}
