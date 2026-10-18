// The token sheet, against the stylesheet it came from.
//
// Every value here was read by hand out of the `<style>` block of
// docs/design/2026-10-17-presence-screens-v3.html and written as a literal.
// The test never reads the constant it checks: a test that does follows the
// constant wherever somebody moves it, and the thing under test is whether
// the Swift says what the screens say.
//
// The CSS variable each value came from is named beside it, so the next
// person can grep the stylesheet rather than trust this file.

import PresenceUI
import SwiftUI
import XCTest

final class ThemeTests: XCTestCase {
  // MARK: the dark values, the `:root` block

  func testTheDarkPaletteIsTheScreensFileLiterals() {
    let dark = Palette.dark

    // --s0, --s1, --s2 and the mock's --wall
    XCTAssertEqual(dark.surface, Swatch(0x1C_1C_1E))
    XCTAssertEqual(dark.raised, Swatch(0x24_24_26))
    XCTAssertEqual(dark.hover, Swatch(0x2C_2C_2E))
    XCTAssertEqual(dark.wall, Swatch(0x3B_43_52))

    // --t1, --t2, --t3
    XCTAssertEqual(dark.text, Swatch(0xF2_F2_F7))
    XCTAssertEqual(dark.textSecondary, Swatch(0xEB_EB_F5, alpha: 0.60))
    XCTAssertEqual(dark.textTertiary, Swatch(0xEB_EB_F5, alpha: 0.32))

    // .act's color, which the dark block leaves at --t3
    XCTAssertEqual(dark.actionIcon, Swatch(0xEB_EB_F5, alpha: 0.32))

    // --hair, --hair-strong
    XCTAssertEqual(dark.hairline, Swatch(0xFF_FF_FF, alpha: 0.08))
    XCTAssertEqual(dark.hairlineStrong, Swatch(0xFF_FF_FF, alpha: 0.14))

    // --ok, --hold, --bad
    XCTAssertEqual(dark.ok, Swatch(0x5C_C4_89))
    XCTAssertEqual(dark.hold, Swatch(0xD9_A0_3C))
    XCTAssertEqual(dark.bad, Swatch(0xE5_48_4D))

    // .hold and .stuck: the tint and the half-pixel edge inside it
    XCTAssertEqual(dark.holdTint, Swatch(0xD9_A0_3C, alpha: 0.07))
    XCTAssertEqual(dark.holdTintEdge, Swatch(0xD9_A0_3C, alpha: 0.22))
    XCTAssertEqual(dark.holdInk, Swatch(0xF2_F2_F7))
    XCTAssertEqual(dark.holdInkQuiet, Swatch(0xEB_EB_F5, alpha: 0.32))
    XCTAssertEqual(dark.stuckTint, Swatch(0xE5_48_4D, alpha: 0.07))
    XCTAssertEqual(dark.stuckTintEdge, Swatch(0xE5_48_4D, alpha: 0.22))

    // .ring, .switch, .scrollbox, .btn-primary, .btn.confirm, .btn-danger
    XCTAssertEqual(dark.ringTrack, Swatch(0xFF_FF_FF, alpha: 0.10))
    XCTAssertEqual(dark.switchTrack, Swatch(0xFF_FF_FF, alpha: 0.24))
    XCTAssertEqual(dark.switchKnob, Swatch(0xFF_FF_FF))
    XCTAssertEqual(dark.scrollThumb, Swatch(0xFF_FF_FF, alpha: 0.22))
    XCTAssertEqual(dark.primaryFill, Swatch(0xFF_FF_FF, alpha: 0.92))
    XCTAssertEqual(dark.primaryInk, Swatch(0x11_11_11))
    XCTAssertEqual(dark.confirmFill, Swatch(0xE5_48_4D, alpha: 0.10))
    XCTAssertEqual(dark.dangerEdge, Swatch(0x00_00_00, alpha: 0))

    // [data-tip]::after, which is the tooltip the app draws itself
    XCTAssertEqual(dark.tooltipSurface, Swatch(0x2C_2C_2E))
    XCTAssertEqual(dark.tooltipEdge, Swatch(0xFF_FF_FF, alpha: 0.12))
    XCTAssertEqual(dark.tooltipShadow, Shadow(Swatch(0x00_00_00, alpha: 0.4), y: 2, blur: 8))

    // .panel: a half-pixel edge, an inset highlight and two drop shadows
    XCTAssertEqual(dark.panelEdge, Swatch(0xFF_FF_FF, alpha: 0.14))
    XCTAssertEqual(dark.panelHighlight, Swatch(0xFF_FF_FF, alpha: 0.06))
    XCTAssertEqual(
      dark.panelShadows,
      [
        Shadow(Swatch(0x00_00_00, alpha: 0.45), y: 12, blur: 40),
        Shadow(Swatch(0x00_00_00, alpha: 0.4), y: 1, blur: 2),
      ])

    // .glyph's border-width, which the light block is the one to override
    XCTAssertEqual(dark.glyphStroke, 1.25)
  }

  // MARK: the light values, the `.screen.light` block

  func testTheLightPaletteIsTheScreensFileLiterals() {
    let light = Palette.light

    XCTAssertEqual(light.surface, Swatch(0xFF_FF_FF))
    XCTAssertEqual(light.raised, Swatch(0xDF_DF_E4))
    XCTAssertEqual(light.hover, Swatch(0xEC_EC_EF))
    XCTAssertEqual(light.wall, Swatch(0x7E_93_AD))

    XCTAssertEqual(light.text, Swatch(0x11_11_13))
    XCTAssertEqual(light.textSecondary, Swatch(0x6B_6B_70))
    XCTAssertEqual(light.textTertiary, Swatch(0x96_96_9C))

    // .screen.light .act: the one grey in the light block that is not --t3
    XCTAssertEqual(light.actionIcon, Swatch(0x7B_7B_81))

    // #0000001a and #00000029: the alpha is a byte in the stylesheet
    XCTAssertEqual(light.hairline, Swatch(0x00_00_00, alpha8: 0x1A))
    XCTAssertEqual(light.hairlineStrong, Swatch(0x00_00_00, alpha8: 0x29))

    XCTAssertEqual(light.ok, Swatch(0x1B_9E_4B))
    XCTAssertEqual(light.hold, Swatch(0xE6_87_0A))
    XCTAssertEqual(light.bad, Swatch(0xE0_31_2E))

    // the light hold card is a solid tint with its own ink, --hold-ink
    XCTAssertEqual(light.holdTint, Swatch(0xFF_F6_E5))
    XCTAssertEqual(light.holdTintEdge, Swatch(0xE6_87_0A, alpha8: 0x66))
    XCTAssertEqual(light.holdInk, Swatch(0x8A_55_00))
    XCTAssertEqual(light.holdInkQuiet, Swatch(0x8A_55_00))
    XCTAssertEqual(light.stuckTint, Swatch(0xC9_38_38, alpha: 0.07))
    XCTAssertEqual(light.stuckTintEdge, Swatch(0xC9_38_38, alpha: 0.28))

    XCTAssertEqual(light.ringTrack, Swatch(0x00_00_00, alpha8: 0x1F))
    XCTAssertEqual(light.switchTrack, Swatch(0x00_00_00, alpha: 0.12))
    XCTAssertEqual(light.switchKnob, Swatch(0xFF_FF_FF))
    XCTAssertEqual(light.scrollThumb, Swatch(0x00_00_00, alpha: 0.22))
    XCTAssertEqual(light.primaryFill, Swatch(0x11_11_13))
    XCTAssertEqual(light.primaryInk, Swatch(0xFF_FF_FF))
    XCTAssertEqual(
      light.confirmFill, Swatch(0xE5_48_4D, alpha: 0.10),
      "the stylesheet leaves .btn.confirm's fill alone in the light block")
    XCTAssertEqual(light.dangerEdge, Swatch(0xE0_31_2E, alpha8: 0x33))

    XCTAssertEqual(light.tooltipSurface, Swatch(0xF2_F2_F3))
    XCTAssertEqual(light.tooltipEdge, Swatch(0x00_00_00, alpha: 0.14))
    XCTAssertEqual(light.tooltipShadow, Shadow(Swatch(0x00_00_00, alpha: 0.14), y: 2, blur: 8))

    // the light panel has no border: the half-pixel ring is its first shadow
    XCTAssertEqual(light.panelEdge, Swatch(0x00_00_00, alpha8: 0x29))
    XCTAssertEqual(light.panelHighlight, Swatch(0xFF_FF_FF))
    XCTAssertEqual(
      light.panelShadows,
      [
        Shadow(Swatch(0x00_00_00, alpha8: 0x1F), y: 1, blur: 2),
        Shadow(Swatch(0x00_00_00, alpha8: 0x29), y: 12, blur: 36),
      ])

    XCTAssertEqual(light.glyphStroke, 1.5)
  }

  // The complaint that started this milestone was the native look, so no
  // panel colour is a semantic system colour and no setting picks a palette:
  // the appearance SwiftUI is drawing in is the whole of the choice.
  func testThePaletteIsTheColourSchemeAndNothingElse() {
    XCTAssertEqual(Palette.of(.dark), Palette.dark)
    XCTAssertEqual(Palette.of(.light), Palette.light)
    XCTAssertNotEqual(Palette.dark, Palette.light)
  }

  // A CSS blur is twice SwiftUI's radius, so a stack copied across keeps the
  // stylesheet's own numbers and converts at the one place it is drawn.
  func testACSSBlurIsTwiceTheRadius() {
    XCTAssertEqual(Shadow(Swatch(0x00_00_00, alpha: 0.45), y: 12, blur: 40).radius, 20)
  }

  func testASwatchIsItsStylesheetChannels() {
    let tertiary = Swatch(0xEB_EB_F5, alpha: 0.32)
    XCTAssertEqual(tertiary.red, 235.0 / 255)
    XCTAssertEqual(tertiary.green, 235.0 / 255)
    XCTAssertEqual(tertiary.blue, 245.0 / 255)
    XCTAssertEqual(tertiary.alpha, 0.32)
    XCTAssertEqual(Swatch(0x00_00_00, alpha8: 0x1A).alpha, 26.0 / 255)
  }

  // MARK: the grid

  func testTheGridIsTheScreens() {
    XCTAssertEqual(Grid.panelWidth, 320)
    XCTAssertEqual(Grid.unit, 8)
    XCTAssertEqual(Grid.panelRadius, 12)
    XCTAssertEqual(Grid.panelMaxHeight, 600)
    XCTAssertEqual(Grid.headerHeight, 36)
    XCTAssertEqual(Grid.sectionLabelHeight, 28)
    XCTAssertEqual(Grid.rowHeight, 32)
    XCTAssertEqual(Grid.twoLineRowHeight, 44)
    XCTAssertEqual(Grid.compactRowHeight, 28)
    XCTAssertEqual(Grid.actionColumn, 108)
    XCTAssertEqual(Grid.glyphSize, 16)
    XCTAssertEqual(Grid.actionSize, 24)
    XCTAssertEqual(Grid.controlRadius, 6)
  }

  // The rest of the stylesheet's layout numbers, each on the grid.
  func testTheLayoutNumbersAreTheStylesheets() {
    XCTAssertEqual(Grid.rowInset, 12)
    XCTAssertEqual(Grid.columnGap, 8)
    XCTAssertEqual(Grid.actionGap, 4)
    XCTAssertEqual(Grid.cardRadius, 8)
    XCTAssertEqual(Grid.cardInset, 10)
    XCTAssertEqual(Grid.ringLarge, 28)
    XCTAssertEqual(Grid.hairline, 0.5)
    XCTAssertEqual(Grid.quietInset, 44)
    XCTAssertEqual(Grid.scrollBoxHeight, 236)
    XCTAssertEqual(Grid.fadeHeight, 28)
    XCTAssertEqual(Grid.badgeSize, 10)
  }

  // MARK: the type

  func testTheTypeRolesAreTwelveStylesheetLiterals() {
    XCTAssertEqual(Typography.roles.count, 12, "the blueprint lists twelve roles")

    // .panel-header .mark and .empty .mark
    XCTAssertEqual(Typography.mark, TypeRole(size: 16, weight: .medium, lineHeight: 16))
    XCTAssertEqual(Typography.markLarge, TypeRole(size: 28, weight: .medium, lineHeight: 28))

    // .panel-header .name, .hold .row-title, .now-job .jt
    XCTAssertEqual(Typography.titleEmphasis, TypeRole(size: 13, weight: .medium, lineHeight: 16))

    // .panel itself: 400 13px/16px with -.006em
    XCTAssertEqual(
      Typography.rowTitle,
      TypeRole(size: 13, weight: .regular, lineHeight: 16, tracking: -0.006))

    // .section-label: 500 11px/1, uppercase, +.06em
    XCTAssertEqual(
      Typography.sectionLabel,
      TypeRole(
        size: 11, weight: .medium, lineHeight: 11, tracking: 0.06, uppercase: true))

    // .row-value, which the quiet line and the more row share the size of
    XCTAssertEqual(Typography.rowValue, TypeRole(size: 12, weight: .regular, lineHeight: 16))

    // .row-sub, .panel-header .account and the tooltip, all 11/14
    XCTAssertEqual(Typography.caption, TypeRole(size: 11, weight: .regular, lineHeight: 14))

    // .btn: 500 12px/1 with -.005em
    XCTAssertEqual(
      Typography.button,
      TypeRole(size: 12, weight: .medium, lineHeight: 12, tracking: -0.005))

    // .status-item .badge: 500 11px/1, the one role the menu bar uses
    XCTAssertEqual(Typography.badge, TypeRole(size: 11, weight: .medium, lineHeight: 11))

    // .urlline: the monospaced role, 400 11px/14px
    XCTAssertEqual(
      Typography.mono, TypeRole(size: 11, weight: .regular, lineHeight: 14, monospaced: true))

    // .strip
    XCTAssertEqual(Typography.strip, TypeRole(size: 10, weight: .regular, lineHeight: 12))

    // .bigmail: 500 15px/20px with -.01em
    XCTAssertEqual(
      Typography.bigMail,
      TypeRole(size: 15, weight: .medium, lineHeight: 20, tracking: -0.01))
  }

  // Two weights everywhere, and the third only where the light block asks
  // for it: a 500 label on white is lighter than the same label on black.
  func testTheLightSectionLabelIsTheThirdWeight() {
    XCTAssertEqual(Typography.sectionLabel(in: .dark).weight, .medium)
    XCTAssertEqual(Typography.sectionLabel(in: .light).weight, .semibold)
    XCTAssertEqual(
      Typography.sectionLabel(in: .light).size, 11, "only the weight changes with the appearance")
  }

  // The stylesheet writes tracking in em; SwiftUI takes points.
  func testTrackingIsEmTurnedIntoPoints() {
    XCTAssertEqual(Typography.rowTitle.letterSpacing, 13 * -0.006)
    XCTAssertEqual(Typography.caption.letterSpacing, 0)
  }

  // MARK: the timings

  func testTheTimingsAreTheDecisions() {
    XCTAssertEqual(Timing.tooltipDelay, 0.5)
    XCTAssertEqual(Timing.doneFlash, 1)
    XCTAssertEqual(Timing.checkFlash, 5)
    XCTAssertEqual(Timing.signInCountdown, 180)
    XCTAssertEqual(Timing.relaunchGuard, 600)
  }
}
