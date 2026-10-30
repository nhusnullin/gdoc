// The one read of the firm's mark, held to one file and one shape.
//
// Every case here is a real file in a temporary directory, because the three
// checks are about what the kernel says and not about what a path looks
// like. The firm's own file is not in this repository by the decision of
// 2026-10-16, so the passing cases are made here: eight signature bytes and
// then filler.

import Foundation
import XCTest

@testable import PresenceCore

final class MarkFileTests: XCTestCase {
  /// The eight bytes a PNG opens with.
  private static let signature: [UInt8] = [0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A]

  func testAPNGComesBackAndNothingElseDoes() throws {
    let directory = try temporaryDirectory()

    let png = directory.appendingPathComponent("logo.png")
    let bytes = pngOf(200)
    try bytes.write(to: png)
    XCTAssertEqual(MarkFile.read(at: png.path), bytes)

    // A JPEG is a picture and still not the one shape this app draws, so it
    // comes back as nothing rather than as bytes the view would have to
    // guess at.
    let jpeg = directory.appendingPathComponent("logo.jpg")
    try Data([0xFF, 0xD8, 0xFF, 0xE0] + [UInt8](repeating: 0x20, count: 196)).write(to: jpeg)
    XCTAssertNil(MarkFile.read(at: jpeg.path))

    // A file too short to carry the signature carries no signature.
    let stub = directory.appendingPathComponent("stub.png")
    try Data(Self.signature[0..<4]).write(to: stub)
    XCTAssertNil(MarkFile.read(at: stub.path))
  }

  func testTheCeilingIsOneMegabyteAndOneByteOverIsRefused() throws {
    let directory = try temporaryDirectory()

    let atTheCeiling = directory.appendingPathComponent("big.png")
    let allowed = pngOf(MarkFile.ceiling)
    try allowed.write(to: atTheCeiling)
    XCTAssertEqual(MarkFile.read(at: atTheCeiling.path)?.count, MarkFile.ceiling)

    let overIt = directory.appendingPathComponent("bigger.png")
    try pngOf(MarkFile.ceiling + 1).write(to: overIt)
    XCTAssertNil(MarkFile.read(at: overIt.path))
  }

  func testADirectoryAMissingPathAndALinkAreAllNothing() throws {
    let directory = try temporaryDirectory()

    XCTAssertNil(MarkFile.read(at: directory.path))
    XCTAssertNil(MarkFile.read(at: directory.appendingPathComponent("gone.png").path))

    // The file named is the file read: a link to a passing PNG is refused,
    // so nobody can point the one path this app reads at a file it never
    // named.
    let real = directory.appendingPathComponent("real.png")
    try pngOf(200).write(to: real)
    let link = directory.appendingPathComponent("link.png")
    try FileManager.default.createSymbolicLink(at: link, withDestinationURL: real)
    XCTAssertEqual(MarkFile.read(at: real.path)?.count, 200)
    XCTAssertNil(MarkFile.read(at: link.path))
  }

  func testAPipeWithNoWriterComesBackRatherThanWaiting() throws {
    let directory = try temporaryDirectory()

    // A pipe nobody is writing to never answers a blocking open, and this
    // read runs inside the poller: the answer must be nothing, now.
    let pipe = directory.appendingPathComponent("logo.png")
    XCTAssertEqual(mkfifo(pipe.path, 0o600), 0)

    let answered = expectation(description: "the read came back")
    var bytes: Data? = pngOf(200)
    Thread.detachNewThread {
      bytes = MarkFile.read(at: pipe.path)
      answered.fulfill()
    }
    wait(for: [answered], timeout: 2)
    XCTAssertNil(bytes)
  }

  // MARK: helpers

  /// A file of exactly `count` bytes that opens with the PNG signature.
  private func pngOf(_ count: Int) -> Data {
    var bytes = Data(Self.signature)
    bytes.append(Data(repeating: 0x7A, count: count - bytes.count))
    return bytes
  }

  private func temporaryDirectory() throws -> URL {
    let directory = URL(fileURLWithPath: NSTemporaryDirectory(), isDirectory: true)
      .appendingPathComponent("markfile-\(UUID().uuidString)", isDirectory: true)
    try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
    addTeardownBlock { try? FileManager.default.removeItem(at: directory) }
    return directory
  }
}
