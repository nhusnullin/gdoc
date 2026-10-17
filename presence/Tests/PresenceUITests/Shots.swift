// What a render test does with a view: draw it, count its ink, write it out.
//
// Four test files draw the same way, so the renderer, the pixel count, the
// measurements and the directory the PNGs go to live here once. The directory is outside the
// repository, always: a render is something a person looks at once, and a
// committed screenshot is a file nobody compares against the code that drew
// it.

import AppKit
import SwiftUI
import XCTest

@testable import PresenceUI

/// One view through the renderer at 2x, in the appearance named.
@MainActor
func png<V: View>(_ view: V, scheme: ColorScheme) throws -> Data {
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

/// How many pixels carry ink. The view is laid out in a square of the size
/// named over nothing, so a transparent pixel is a pixel nothing drew on.
@MainActor
func inkedPixels<V: View>(_ view: V, scheme: ColorScheme, side: CGFloat) throws -> Int {
  var counted = 0
  try eachPixel(view, scheme: scheme, side: side) { alpha in
    if alpha > 0.05 { counted += 1 }
  }
  return counted
}

/// How much ink, rather than how many pixels: the alphas added up, so a view
/// drawn at a lower opacity weighs less than the same view at full.
@MainActor
func inkWeight<V: View>(_ view: V, scheme: ColorScheme, side: CGFloat) throws -> Double {
  var total = 0.0
  try eachPixel(view, scheme: scheme, side: side) { total += $0 }
  return total
}

@MainActor
private func eachPixel<V: View>(
  _ view: V, scheme: ColorScheme, side: CGFloat, _ body: (Double) -> Void
) throws {
  let framed = view.frame(width: side, height: side)
  let renderer = ImageRenderer(content: framed.environment(\.colorScheme, scheme))
  renderer.scale = 2
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
        body(Double(rep.colorAt(x: x, y: y)?.alphaComponent ?? 0))
      }
    }
  }
  if let failure { XCTFail(failure) }
}

/// Where the PNGs go: outside the repository, named by PRESENCE_SHOTS or a
/// directory in the temporary one.
func shotsDirectory() throws -> URL {
  let named = ProcessInfo.processInfo.environment["PRESENCE_SHOTS"]
  let temporary = URL(fileURLWithPath: NSTemporaryDirectory())
    .appendingPathComponent("gdoc-presence-shots")
  let base = named.map { URL(fileURLWithPath: $0) } ?? temporary
  let repository = presencePackageRoot().deletingLastPathComponent().standardizedFileURL.path
  XCTAssertFalse(
    base.standardizedFileURL.path.hasPrefix(repository),
    "PRESENCE_SHOTS names \(base.path), which is inside the repository")
  try FileManager.default.createDirectory(at: base, withIntermediateDirectories: true)
  return base
}

/// This file is Tests/PresenceUITests/Shots.swift, so the package is three
/// directories up from it. The path is the file's own and never the working
/// directory, which swift test does not promise.
func presencePackageRoot() -> URL {
  URL(fileURLWithPath: #filePath)
    .deletingLastPathComponent()
    .deletingLastPathComponent()
    .deletingLastPathComponent()
}

/// What a view measured, by name: the `metricProbe` names under it against
/// the boxes they were drawn in.
///
/// A size in this package is a literal from the screens file, and a literal
/// is worth nothing until the drawing is shown to honour it, so the tests
/// read the numbers back through the preference key rather than off the
/// constant they came from.
@MainActor
func probes<V: View>(_ view: V, scheme: ColorScheme, width: CGFloat = Grid.panelWidth) throws
  -> [String: CGSize]
{
  let collector = ProbeCollector()
  let framed =
    view
    .frame(width: width)
    .environment(\.colorScheme, scheme)
    .onPreferenceChange(MetricProbes.self) { reported in
      MainActor.assumeIsolated { collector.reported = reported }
    }
  let renderer = ImageRenderer(content: framed)
  renderer.scale = 2
  let appearance = NSAppearance(named: scheme == .dark ? .darkAqua : .aqua)
  appearance?.performAsCurrentDrawingAppearance {
    _ = renderer.cgImage
  }
  XCTAssertFalse(collector.reported.isEmpty, "nothing reported a size in \(scheme)")
  var byName: [String: CGSize] = [:]
  for probe in collector.reported where byName[probe.name] == nil {
    byName[probe.name] = probe.size
  }
  return byName
}

/// The box a render writes its measurements into.
@MainActor
final class ProbeCollector {
  var reported: [MetricProbe] = []
}

/// One pixel of a view, by its point in the square the view was laid out in.
///
/// The render is at 2x, so a point is two pixels and the sample is the first
/// of them. A test that asks for a colour asks for it where it drew
/// something, so there is no searching here: the caller names the place.
@MainActor
func pixel<V: View>(
  _ view: V, scheme: ColorScheme, side: CGFloat, x: CGFloat, y: CGFloat
) throws -> NSColor {
  let framed = view.frame(width: side, height: side)
  let renderer = ImageRenderer(content: framed.environment(\.colorScheme, scheme))
  renderer.scale = 2
  var sampled: NSColor?
  let appearance = NSAppearance(named: scheme == .dark ? .darkAqua : .aqua)
  appearance?.performAsCurrentDrawingAppearance {
    guard let image = renderer.cgImage else { return }
    let rep = NSBitmapImageRep(cgImage: image)
    sampled = rep.colorAt(x: Int(x * 2), y: Int(y * 2))?.usingColorSpace(.sRGB)
  }
  return try XCTUnwrap(sampled, "the renderer drew nothing in \(scheme)")
}
