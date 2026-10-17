// Where this machine keeps the four things the app needs of it.
//
// The record's directory, this app's own bundle, the version and identifier
// that bundle names, the home directory the binary is looked for under, and
// the arguments this instance came up with. Every one of them is a fact about
// the machine, read once at launch and handed over as a value, so a session
// in a test is a session over a directory the test made.
//
// The shell reads none of these itself. It asks for this machine's places and
// hands them to the session, which is the whole of what the shell knows about
// the file system.

import Foundation
import PresenceCore

/// The paths and the two strings off the plist.
public struct Places: Sendable, Equatable {
  /// Where gdoc appends the record.
  public let records: URL
  /// This app's own bundle, when it is running from one. A bare executable
  /// from `swift run` has none, and then nothing watches a bundle and
  /// nothing relaunches.
  public let bundle: URL?
  /// The version the bundle on the disk named at launch.
  public let version: String
  /// This app's own identifier, as the running app knows it.
  public let identifier: String
  /// The home directory `~/.local/bin/gdoc` is under.
  public let home: String
  /// The arguments this instance came up with, where the relaunch's own is.
  public let arguments: [String]

  /// The version a bundle with no plist to read is taken to be at. It
  /// compares equal to nothing a release names, so the reopen row is never
  /// shown for it.
  public static let noVersion = "0.0.0"

  public init(
    records: URL,
    bundle: URL? = nil,
    version: String = Places.noVersion,
    identifier: String = "",
    home: String = "",
    arguments: [String] = []
  ) {
    self.records = records
    self.bundle = bundle
    self.version = version
    self.identifier = identifier
    self.home = home
    self.arguments = arguments
  }

  /// This machine, as the shell finds it at launch.
  ///
  /// The version is read off the plist on the disk rather than out of the
  /// cached bundle, because that is the one number the bundle watcher
  /// compares against later and a cached answer would never move.
  public static func ofThisMachine(
    environment: [String: String] = ProcessInfo.processInfo.environment,
    bundle: URL? = Places.runningBundle(),
    identifier: String? = Bundle.main.bundleIdentifier,
    arguments: [String] = ProcessInfo.processInfo.arguments,
    home: String = FileManager.default.homeDirectoryForCurrentUser.path
  ) -> Places {
    Places(
      records: recordDirectory(environment: environment, home: home),
      bundle: bundle,
      version: bundle.flatMap(BundleWatcher.versionOnDisk(of:)) ?? noVersion,
      identifier: identifier ?? "",
      home: home,
      arguments: arguments)
  }

  /// The bundle this process is running out of, where it is running out of
  /// one at all.
  public static func runningBundle() -> URL? {
    let url = Bundle.main.bundleURL
    return url.pathExtension == "app" ? url : nil
  }

  /// Where gdoc keeps the record. GDOC_CONFIG_DIR first, which is the one
  /// override the binary itself honours, so a person pointing gdoc at
  /// another directory can point the app at the same one.
  public static func recordDirectory(environment: [String: String], home: String) -> URL {
    let named = environment["GDOC_CONFIG_DIR"] ?? ""
    let base =
      named.isEmpty
      ? URL(fileURLWithPath: home).appendingPathComponent(".config/gdoc-agent")
      : URL(fileURLWithPath: named)
    return base.appendingPathComponent("presence")
  }
}
