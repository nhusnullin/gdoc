// The one switch in the panel: Launch at login.
//
// `.switch` in docs/design/2026-10-17-presence-screens-v3.html is 26 by 16
// with a 12 pt knob inset 2, and it is drawn here rather than handed to
// SwiftUI's own toggle, which is 51 pt wide, blue, and the native look that
// started this milestone. The on state is `var(--t1)`, the panel's brightest
// grey: a setting is not a status, so it is never green.
//
// It presses, because the row it sits in is a setting and the switch is the
// setting's control. What it records is the app's business; this file draws
// the state it is handed.
//
// ControlTests states the geometry, the two fills and that no file in the
// package hands a switch to the platform.

import SwiftUI

/// One drawn switch.
public struct SwitchView: View {
  /// `.switch{width:26px;height:16px;border-radius:8px}`.
  public static let width: CGFloat = 26
  public static let height: CGFloat = 16
  public static let radius: CGFloat = 8
  /// `.switch::after{width:12px;height:12px;top:2px;left:2px}`.
  public static let knob: CGFloat = 12
  public static let inset: CGFloat = 2
  /// `.switch.is-on::after{left:12px}`.
  public static let knobTravel: CGFloat = 12
  /// `box-shadow:0 1px 2px rgba(0,0,0,.3)` under the knob, inline in the
  /// stylesheet rather than a token, so it is a literal beside its rule.
  public static let knobShadow = Shadow(Swatch(0x00_00_00, alpha: 0.3), y: 1, blur: 2)

  public let isOn: Bool
  public let action: () -> Void

  @Environment(\.colorScheme) private var scheme

  public init(isOn: Bool, action: @escaping () -> Void = {}) {
    self.isOn = isOn
    self.action = action
  }

  /// Where the knob's left edge sits.
  public static func knobOffset(isOn: Bool) -> CGFloat {
    isOn ? knobTravel : inset
  }

  /// `.switch{background:rgba(255,255,255,.24)}` and
  /// `.switch.is-on{background:var(--t1)}`.
  public static func track(isOn: Bool, in palette: Palette) -> Swatch {
    isOn ? palette.text : palette.switchTrack
  }

  /// `.switch::after{background:#fff}` and
  /// `.switch.is-on::after{background:var(--s0)}`: on a bright track the
  /// knob becomes the panel's own surface, so the switch reads as a hole.
  public static func knobInk(isOn: Bool, in palette: Palette) -> Swatch {
    isOn ? palette.surface : palette.switchKnob
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return Button(action: action) {
      ZStack(alignment: .topLeading) {
        RoundedRectangle(cornerRadius: Self.radius, style: .circular)
          .fill(Self.track(isOn: isOn, in: palette).color)
        Circle()
          .fill(Self.knobInk(isOn: isOn, in: palette).color)
          .frame(width: Self.knob, height: Self.knob)
          .shadow(
            color: Self.knobShadow.swatch.color, radius: Self.knobShadow.radius,
            x: Self.knobShadow.x, y: Self.knobShadow.y
          )
          .offset(x: Self.knobOffset(isOn: isOn), y: Self.inset)
      }
      .frame(width: Self.width, height: Self.height)
      .contentShape(RoundedRectangle(cornerRadius: Self.radius, style: .circular))
      .metricProbe("switch")
    }
    .buttonStyle(.plain)
  }
}
