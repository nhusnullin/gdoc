// The firm's mark, read from the one file the help names.
//
// `gdoc help --json` says where `internal/asset` keeps the house logo on this
// machine, and this room reads that path and no other. A machine that has
// built a document has the file; a machine that has not has nothing to draw,
// which is not an error and writes no line anywhere.
//
// Three checks, in this order. The path is opened once with O_NOFOLLOW, so a
// link at that name is refused by the kernel: the file named is the file
// read, and nobody can point the one path this app opens at a file it never
// named. O_NONBLOCK rides along, because the open itself must answer: a pipe
// at that name with nobody writing to it would hold a blocking open forever,
// and this read runs inside the poller. The flag changes nothing about a
// regular file. The open descriptor is then asked what it is, so the answer
// is about the bytes that will be read and not about a name a swap could
// outrun: a regular file, of at most `ceiling` bytes. Last, the bytes
// themselves must open with the eight PNG bytes, because the panel draws one
// shape and a view that had to guess would be guessing about somebody's
// screen.
//
// Any failure is nothing. The tests are `MarkFileTests.swift`, and
// `SourceRulesTests.testAByteIsReadOnlyInTheFiveReaders` holds that this is
// one of the five files in the app that reads a byte at all.

import Foundation

/// The one read of the house logo's cached file.
public enum MarkFile {
  /// The most this app will read for a mark, in bytes. The house file is a
  /// wordmark of a few kilobytes; a megabyte is room for one drawn far
  /// larger than the panel needs and still a bound a view can hold.
  public static let ceiling = 1_048_576

  /// The eight bytes every PNG opens with.
  private static let signature: [UInt8] = [0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A]

  /// The file's bytes, or nothing where there is no PNG of a sane size at
  /// that exact path.
  public static func read(at path: String) -> Data? {
    let descriptor = Darwin.open(path, O_RDONLY | O_NOFOLLOW | O_NONBLOCK)
    guard descriptor >= 0 else { return nil }
    defer { Darwin.close(descriptor) }

    var about = Darwin.stat()
    guard Darwin.fstat(descriptor, &about) == 0,
      (about.st_mode & S_IFMT) == S_IFREG,
      about.st_size >= Int64(signature.count),
      about.st_size <= Int64(ceiling)
    else { return nil }

    guard let bytes = bytes(from: descriptor), bytes.count <= ceiling else { return nil }
    guard bytes.starts(with: signature) else { return nil }
    return bytes
  }

  /// At most one byte over the ceiling, so a file that grew between the
  /// question and the read is still refused rather than half drawn.
  private static func bytes(from descriptor: Int32) -> Data? {
    var out = Data()
    let wanted = 1 << 16
    var buffer = [UInt8](repeating: 0, count: wanted)
    while out.count <= ceiling {
      let taken = buffer.withUnsafeMutableBytes { raw in
        Darwin.read(descriptor, raw.baseAddress, wanted)
      }
      if taken < 0 { return nil }
      if taken == 0 { return out }
      out.append(contentsOf: buffer[0..<taken])
    }
    return out
  }
}
