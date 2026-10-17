// Whether the app on the disk is still the app that is running.
//
// gdoc update swaps the bundle by rename while the app may be open. The app
// cannot tell the person anything useful about that unless it compares the
// version it launched with against the one on the disk now, so the comparison
// is a pure function of two strings and the file reading lives in the shell.

import Foundation
import XCTest

@testable import PresenceCore

final class BundleVersionTests: XCTestCase {
  func testTheBundleVersionCheckIsAPureFunction() {
    XCTAssertEqual(BundleVersion.check(running: "2.13.0", onDisk: "2.13.0"), .same)
    XCTAssertEqual(BundleVersion.check(running: "0.0.0", onDisk: "0.0.0"), .same)
    XCTAssertEqual(
      BundleVersion.check(running: "2.13.0", onDisk: "2.14.0"), .reopenNeeded("2.14.0"),
      "the person quits and reopens; this milestone never relaunches itself")
    XCTAssertEqual(
      BundleVersion.check(running: "2.14.0", onDisk: "2.13.0"), .reopenNeeded("2.13.0"),
      "a rollback is a difference like any other, and the app says which version is there")
    XCTAssertEqual(BundleVersion.check(running: "2.13.0", onDisk: nil), .gone)
    XCTAssertEqual(
      BundleVersion.check(running: "2.13.0", onDisk: ""), .gone,
      "a plist with no version in it is the same as no bundle to the person reading the row")
  }
}
