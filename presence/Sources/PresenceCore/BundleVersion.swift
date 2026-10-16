// Whether the app on the disk is still the app that is running.
//
// gdoc update stages a new bundle beside the old one and swaps it by rename,
// which the running app does not notice: its code is already mapped and
// Bundle.main has cached what it read at launch. So the shell reads
// Contents/Info.plist off the disk again and hands the two strings here. The
// comparison is a pure function so the reading and the deciding are not the
// same thing, and so every answer is reachable from a test.
//
// This milestone never relaunches itself. Whether that swap is safe while the
// app runs is M24's question, asked after it is measured.

import Foundation

/// What the app shows about the bundle under it.
public enum BundleState: Sendable, Equatable {
  /// The disk holds the version that is running. Nothing is said.
  case same
  /// The disk holds another version, named here, so the row reads
  /// "Updated to X. Quit and reopen".
  case reopenNeeded(String)
  /// There is no bundle, or none that names a version, so the row reads that
  /// gdoc presence was removed.
  case gone
}

public enum BundleVersion {
  /// The running version against the one on the disk now.
  ///
  /// A difference in either direction is a reopen: a rollback is as much a
  /// change as an update, and the row names the version that is there rather
  /// than calling it newer.
  public static func check(running: String, onDisk: String?) -> BundleState {
    guard let onDisk, !onDisk.isEmpty else { return .gone }
    return onDisk == running ? .same : .reopenNeeded(onDisk)
  }
}
