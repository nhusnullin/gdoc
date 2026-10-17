// What a render test does with a view: draw it, count its ink, write it out.
//
// Three test files draw the same way, so the renderer, the pixel count and
// the directory the PNGs go to live here once. The directory is outside the
// repository, always: a render is something a person looks at once, and a
// committed screenshot is a file nobody compares against the code that drew
// it.

import AppKit
import SwiftUI
import XCTest

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
