// What a view measured, reported upwards so a test can read it.
//
// Every size in this package is a literal from the screens file, and a
// literal in a layout is worth nothing until the layout is shown to honour
// it: a row that states 32 and draws 36 passes every test that reads the
// constant. So each piece of the panel reports the box it ended up with
// under a name, through one preference key, and the tests assert the numbers
// off the drawing rather than off the code.
//
// The probe is an overlay of a clear colour, which takes the size it is
// given and changes no layout. It is in the shipping view and not in the
// test target on purpose: the tests of the scenes read the inside of a
// whole panel, which they cannot reach from outside.
//
// RowTests reads these for the five row shapes, the section label and the
// header.

import SwiftUI

/// One measurement: what reported it, and the box it was drawn in.
public struct MetricProbe: Equatable, Sendable {
  public let name: String
  public let size: CGSize

  public init(name: String, size: CGSize) {
    self.name = name
    self.size = size
  }
}

/// Every measurement under a view, in the order they were drawn.
public struct MetricProbes: PreferenceKey {
  public static let defaultValue: [MetricProbe] = []

  public static func reduce(value: inout [MetricProbe], nextValue: () -> [MetricProbe]) {
    value.append(contentsOf: nextValue())
  }
}

extension View {
  /// Report this view's own box under a name.
  public func metricProbe(_ name: String) -> some View {
    overlay(
      GeometryReader { geometry in
        Color.clear.preference(
          key: MetricProbes.self,
          value: [MetricProbe(name: name, size: geometry.size)])
      }
      .allowsHitTesting(false)
    )
  }
}
