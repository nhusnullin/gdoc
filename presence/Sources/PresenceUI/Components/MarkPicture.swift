// The firm's mark as a bitmap of the size it is drawn at.
//
// v2.29.0 handed the cached file's bytes to the compositor as an image and
// let the view scale them: 509 by 185 pixels down to 14 pt, a shrink of six
// and a half. Nail looked at it within the hour and doubted the resolution,
// and the measurement agreed: the letters came out ragged beside words the
// same screen had anti-aliased (docs/v2/DECISIONS.md, 2026-10-30, "The
// mark, sharp and a press"). What is made here instead is one bitmap of
// exactly the pixels the mark will occupy, averaged down from the source in
// one draw, so the reader sees the wordmark and not the compositor's work
// on it.
//
// It is made once per bytes, height and scale and then kept. A view's body
// runs on every evaluation, and a shrink of half a million pixels there is
// a shrink a frame.
//
// What comes back is a bitmap and never a wrapper around one. The plan's
// reviewer measured that a wrapper built at a stated point size reports the
// main screen's pixel count and not the bitmap's, and that drawing one
// under a renderer at another scale resamples a second time; the bitmap
// itself, handed to SwiftUI with its scale, draws one for one at 1x, 2x and
// 3x.
//
// Nothing here reaches the file. The bytes arrive from the facts, already in
// hand, and a draw over a picture already in memory is not a read of this
// machine.
//
// RowTests states the pixel size, the once and the averaging.

import AppKit
import SwiftUI

/// The mark, drawn down to the pixels it will be shown at.
@MainActor
public enum MarkPicture {
  /// The bitmap for these bytes at this height and this screen scale, or
  /// nothing where the bytes are no picture or the numbers are no size.
  ///
  /// The width is the source's own aspect, so no proportion of the firm's
  /// wordmark is written down in this package.
  public static func make(_ data: Data, height: CGFloat, scale: CGFloat) -> CGImage? {
    guard height > 0, scale > 0 else { return nil }

    let key = Key(bytes: data, height: height, scale: scale)
    if let made = made[key] { return made }

    // An image with no size is the answer AppKit gives for a file that is
    // half written, and there is nothing to shrink.
    guard let image = NSImage(data: data) else { return nil }
    var proposed = CGRect(origin: .zero, size: image.size)
    guard let source = image.cgImage(forProposedRect: &proposed, context: nil, hints: nil),
      source.width > 0, source.height > 0
    else { return nil }

    let pixelHeight = Int((height * scale).rounded())
    let aspect = CGFloat(source.width) / CGFloat(source.height)
    let pixelWidth = Int((CGFloat(pixelHeight) * aspect).rounded())
    guard pixelHeight > 0, pixelWidth > 0 else { return nil }

    // Premultiplied, because the mark is a shape on nothing and the
    // transparency is what the template tint reads. High interpolation,
    // because the point of this room is the averaging: the cheaper settings
    // sample the source and leave its edges standing.
    guard let space = CGColorSpace(name: CGColorSpace.sRGB),
      let context = CGContext(
        data: nil, width: pixelWidth, height: pixelHeight, bitsPerComponent: 8,
        bytesPerRow: 0, space: space,
        bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue)
    else { return nil }
    context.interpolationQuality = .high
    context.draw(source, in: CGRect(x: 0, y: 0, width: pixelWidth, height: pixelHeight))
    guard let bitmap = context.makeImage() else { return nil }

    made[key] = bitmap
    return bitmap
  }

  /// The bytes, the height and the scale together. The height is in the key
  /// because a caller at another height would otherwise be handed the
  /// bitmap of the one before it.
  private struct Key: Hashable {
    let bytes: Data
    let height: CGFloat
    let scale: CGFloat
  }

  /// What has been drawn so far. One picture at one height on one screen is
  /// one entry, and a machine has few of either.
  private static var made: [Key: CGImage] = [:]
}
