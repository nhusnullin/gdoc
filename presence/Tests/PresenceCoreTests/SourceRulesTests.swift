// What no file in this package may contain.
//
// The app reads one directory of forgeable lines and its own plist, and runs
// six argv lists at a path it resolved itself. It opens no socket, so a line
// somebody planted in the record has no wire to reach even if the app
// believed every word of it; and it starts a program in one file only, so
// what it can run is a list somebody can read in one sitting. The rules are
// source scans rather than a review, because the next person to add a
// convenience here will not have read the specification.
//
// The second scan is the one sentence M23 reversed: that app launched
// nothing at all, and this one launches the six.
//
// The rest are about what the app does to the machine and what it puts in
// front of somebody: the system is asked through one file, an application is
// opened by one file, this machine's login items are touched by one file, and
// the only addresses any of them can build are two schemes and two hosts. A
// line of the record is forgeable, so a click that came out of one must have
// nowhere else to go.

import Foundation
import XCTest

final class SourceRulesTests: XCTestCase {
  /// Where the one room that starts a program is, relative to the package.
  private static let runner = "Sources/PresenceApp/Runner.swift"

  /// Where the one room that asks the system to open or copy something is.
  private static let opener = "Sources/PresenceApp/Opener.swift"

  /// Where the one room that sends a signal is.
  private static let stopper = "Sources/PresenceApp/Stopper.swift"

  /// Where the one room that writes anything down is.
  private static let preferences = "Sources/PresenceApp/Preferences.swift"

  /// Where the one room that opens an application is, and the one
  /// application it opens is this app's own bundle.
  private static let relauncher = "Sources/PresenceApp/Relauncher.swift"

  /// Where the one room that touches this machine's login items is.
  private static let loginItem = "Sources/PresenceApp/LoginItem.swift"

  /// The question that signals nothing: a zero asks whether a process is
  /// there and is the whole of Liveness. Any other number is a signal.
  private static let asking = "0"

  /// The schemes the app may name. `https` is a document and a sign-in;
  /// `obsidian` is a note in the vault the person chose. A third is a
  /// decision in `docs/v2/DECISIONS.md`.
  private static let schemes = ["https", "obsidian"]

  /// The hosts it may name, both Google's.
  private static let hosts = ["docs.google.com", "accounts.google.com"]

  func testNoSourceFileReachesTheWire() throws {
    // Spelled in pieces so this file is not its own failure.
    let session = "URL" + "Session"
    let network = "import " + "Network"

    var seen = 0
    try eachSwiftFile { path, text in
      seen += 1
      XCTAssertFalse(text.contains(session), "\(path) reaches the wire")
      XCTAssertFalse(text.contains(network), "\(path) reaches the wire")
    }
    XCTAssertGreaterThan(seen, 5, "no Swift files were read, so this test guards nothing")
  }

  func testProcessAppearsOnlyInTheRunner() throws {
    let child = "Proc" + "ess"

    var sawTheRunner = false
    try eachSwiftFile { path, text in
      let named = text.split(separator: "\n").filter { word(child, isIn: String($0)) }
      guard path != Self.runner else {
        sawTheRunner = true
        XCTAssertFalse(named.isEmpty, "\(path) is the one file that starts a program")
        return
      }
      for line in named {
        XCTFail("\(path) names a \(child): \(line.trimmingCharacters(in: .whitespaces))")
      }
    }
    XCTAssertTrue(sawTheRunner, "\(Self.runner) is not there, so this test guards nothing")
  }

  /// The argv set is closed at six, in the source and not only in a run.
  ///
  /// A seventh case is a seventh thing the app can do to somebody's
  /// documents, so it is a decision in `docs/v2/DECISIONS.md` and a line in
  /// CLAUDE.md's invariant, never a case added while fixing something else.
  func testTheArgvEnumIsClosedAtSix() throws {
    let text = try String(
      contentsOf: packageRoot().appendingPathComponent(Self.runner), encoding: .utf8)
    let opening = try XCTUnwrap(
      text.range(of: "public enum Call"), "the argv enum is not in \(Self.runner)")
    let body = text[opening.upperBound...]
    let closing = try XCTUnwrap(body.range(of: "\n}"), "the argv enum is never closed")
    let declared = body[..<closing.lowerBound]
      .split(separator: "\n")
      .map { $0.trimmingCharacters(in: .whitespaces) }
      .filter { $0.hasPrefix("case ") && !$0.hasPrefix("case .") }
    XCTAssertEqual(
      declared,
      [
        "case help", "case authStatus", "case authLogin", "case authLogout", "case update",
        "case updateDesktop",
      ])
  }

  /// One file sends a signal, and it sends one kind.
  ///
  /// This is the only thing the app does that reaches a program it did not
  /// start, so it is held twice: to one file, and to one signal. A second
  /// signal to a wait would be a stop nobody trusts, and a stronger one
  /// would take away the end that makes Stop usable at all. Liveness's
  /// question is exempt by its zero, which signals nothing.
  func testASignalIsSentOnlyByTheStopper() throws {
    let send = "kil" + "l("

    var sawTheStopper = false
    try eachSwiftFile { path, text in
      let named = text.split(separator: "\n").filter { $0.contains(send) }
      guard path != Self.stopper else {
        sawTheStopper = true
        XCTAssertFalse(named.isEmpty, "\(path) is the one file that sends a signal")
        return
      }
      for line in named {
        XCTAssertTrue(
          signalless(String(line), call: send),
          "\(path) sends a signal: \(line.trimmingCharacters(in: .whitespaces))")
      }
    }
    XCTAssertTrue(sawTheStopper, "\(Self.stopper) is not there, so this test guards nothing")

    let text = try String(
      contentsOf: packageRoot().appendingPathComponent(Self.stopper), encoding: .utf8)
    XCTAssertTrue(text.contains("SIGINT"), "the one signal is SIGINT")
    for other in ["SIGKIL" + "L", "SIGTER" + "M", "SIGHUP", "SIGQUIT", "SIGSTOP", "SIGUSR"] {
      XCTAssertFalse(text.contains(other), "\(Self.stopper) names \(other), and SIGINT is the one")
    }
  }

  /// Whether the call on this line asks rather than signals: its last
  /// argument is the zero.
  private func signalless(_ line: String, call: String) -> Bool {
    guard let opening = line.range(of: call) else { return true }
    let rest = line[opening.upperBound...]
    guard let closing = rest.firstIndex(of: ")") else { return false }
    let arguments = rest[..<closing].split(separator: ",").map {
      $0.trimmingCharacters(in: .whitespaces)
    }
    return arguments.last == Self.asking
  }

  /// One file writes anything down, so "nothing else persists" is a rule
  /// somebody can read rather than a review.
  ///
  /// The app's own six keys are the whole of what outlives the process. A
  /// seventh key, or a second file writing one, is a decision in
  /// `docs/v2/DECISIONS.md`: everything else this app knows comes out of the
  /// record, which is gdoc's file and not the app's.
  func testOnlyThePreferencesPersist() throws {
    let store = "User" + "Defaults"

    var sawThePreferences = false
    try eachSwiftFile { path, text in
      guard path.hasPrefix("Sources/") else { return }
      let named = text.split(separator: "\n").filter { word(store, isIn: String($0)) }
      guard path != Self.preferences else {
        sawThePreferences = true
        XCTAssertFalse(named.isEmpty, "\(path) is the one file that writes something down")
        return
      }
      for line in named {
        XCTFail("\(path) persists something: \(line.trimmingCharacters(in: .whitespaces))")
      }
    }
    XCTAssertTrue(
      sawThePreferences, "\(Self.preferences) is not there, so this test guards nothing")
  }

  /// The system is asked for something in two files, so what this app can
  /// put in front of a person is a list one person can read.
  ///
  /// The opener is the four verbs a row's icons have; the relauncher opens
  /// this app's own bundle and nothing else. Both names are here rather than
  /// in a review, because the next person to add a convenience will not have
  /// read the specification.
  func testTheSystemIsAskedOnlyThroughTheOpenerAndTheRelauncher() throws {
    let workspace = "NSWork" + "space"
    let board = "NSPaste" + "board"
    let running = "NSRunning" + "Application"

    var asking: Set<String> = []
    try eachSwiftFile { path, text in
      guard path.hasPrefix("Sources/") else { return }
      let named = text.split(separator: "\n").filter { line in
        [workspace, board, running].contains { word($0, isIn: String(line)) }
      }
      guard [Self.opener, Self.relauncher].contains(path) else {
        for line in named {
          XCTFail("\(path) asks the system: \(line.trimmingCharacters(in: .whitespaces))")
        }
        return
      }
      if !named.isEmpty { asking.insert(path) }
    }
    XCTAssertEqual(
      asking, [Self.opener, Self.relauncher],
      "both files are there and both ask the system, so this test guards something")
  }

  /// One file starts an application, and the application is this app.
  ///
  /// Every other route to the system is a URL or an activation of an app a
  /// person installed themselves. Opening a bundle at a path is the one call
  /// that could start anything at all, so it is held to the file that opens
  /// this app's own bundle, with the new instance and the no-activation the
  /// specification names.
  func testOnlyTheRelauncherOpensAnApplication() throws {
    let start = "openApp" + "lication"

    var sawTheRelauncher = false
    try eachSwiftFile { path, text in
      guard path.hasPrefix("Sources/") else { return }
      let named = text.split(separator: "\n").filter { $0.contains(start) }
      guard path != Self.relauncher else {
        sawTheRelauncher = true
        XCTAssertFalse(named.isEmpty, "\(path) is the one file that opens an application")
        return
      }
      for line in named {
        XCTFail("\(path) opens an application: \(line.trimmingCharacters(in: .whitespaces))")
      }
    }
    XCTAssertTrue(
      sawTheRelauncher, "\(Self.relauncher) is not there, so this test guards nothing")

    let text = try String(
      contentsOf: packageRoot().appendingPathComponent(Self.relauncher), encoding: .utf8)
    XCTAssertTrue(
      text.contains("createsNewApplicationInstance = true"),
      "the bundle on the disk is opened as a new instance")
    XCTAssertTrue(
      text.contains("activates = false"), "a relaunch is not something a person asked to look at")
  }

  /// One file registers this app at login, and nothing else names the
  /// register at all.
  ///
  /// Starting by itself is the one thing this app arranges on somebody's
  /// machine without being asked, so both halves of it are held here: the
  /// register is reached from one file, and the settings pane that file opens
  /// is opened through the register's own call rather than a settings URL,
  /// which would be a third scheme in the next test.
  func testLoginItemsAreTouchedOnlyByTheLoginItem() throws {
    let register = "SMApp" + "Service"

    var sawTheLoginItem = false
    try eachSwiftFile { path, text in
      guard path.hasPrefix("Sources/") else { return }
      let named = text.split(separator: "\n").filter { word(register, isIn: String($0)) }
      guard path != Self.loginItem else {
        sawTheLoginItem = true
        XCTAssertFalse(named.isEmpty, "\(path) is the one file that touches the login items")
        return
      }
      for line in named {
        XCTFail("\(path) touches the login items: \(line.trimmingCharacters(in: .whitespaces))")
      }
    }
    XCTAssertTrue(sawTheLoginItem, "\(Self.loginItem) is not there, so this test guards nothing")

    let text = try String(
      contentsOf: packageRoot().appendingPathComponent(Self.loginItem), encoding: .utf8)
    XCTAssertTrue(
      text.contains("SMAppService.mainApp"), "the bundle registered is this app's own")
    XCTAssertTrue(
      text.contains("SMAppService.openSystemSettingsLoginItems()"),
      "the approval row opens the pane through the register and not through a URL")
  }

  /// Two schemes and two hosts in the whole of the sources.
  ///
  /// Only the sources: a test that holds a refusal has to write the refused
  /// address down, and this is the file that says so.
  func testTheOnlySchemesAndHostsAreGooglesAndTheVaults() throws {
    var seenSchemes: Set<String> = []
    var seenHosts: Set<String> = []

    try eachSwiftFile { path, text in
      guard path.hasPrefix("Sources/") else { return }
      for scheme in schemesNamed(in: text) {
        XCTAssertTrue(Self.schemes.contains(scheme), "\(path) names the scheme \(scheme)")
        seenSchemes.insert(scheme)
      }
      for host in hostsNamed(in: text) {
        XCTAssertTrue(Self.hosts.contains(host), "\(path) names the host \(host)")
        seenHosts.insert(host)
      }
    }

    XCTAssertEqual(seenSchemes, Set(Self.schemes), "both schemes are named, and no third")
    XCTAssertEqual(seenHosts, Set(Self.hosts), "both hosts are named, and no third")
  }

  /// Every `<scheme>://` in a file, as the scheme alone.
  private func schemesNamed(in text: String) -> [String] {
    var found: [String] = []
    var search = text.startIndex..<text.endIndex
    while let mark = text.range(of: "://", range: search) {
      let scheme = String(
        text[..<mark.lowerBound].reversed().prefix(while: {
          $0.isLetter || $0.isNumber || $0 == "+" || $0 == "-" || $0 == "."
        }).reversed())
      found.append(scheme)
      search = mark.upperBound..<text.endIndex
    }
    return found
  }

  /// Every host under Google's own domain a file names, whole.
  private func hostsNamed(in text: String) -> [String] {
    var found: [String] = []
    var search = text.startIndex..<text.endIndex
    while let mark = text.range(of: "google.com", range: search) {
      let before = String(
        text[..<mark.lowerBound].reversed().prefix(while: {
          $0.isLetter || $0.isNumber || $0 == "-" || $0 == "."
        }).reversed())
      found.append(before + "google.com")
      search = mark.upperBound..<text.endIndex
    }
    return found
  }

  /// Every Swift file in the package, as its path relative to the package
  /// root and its text.
  private func eachSwiftFile(_ body: (String, String) throws -> Void) throws {
    let root = packageRoot()
    for directory in ["Sources", "Tests"] {
      let base = root.appendingPathComponent(directory)
      let walker = try XCTUnwrap(
        FileManager.default.enumerator(atPath: base.path), "\(directory) is not there")
      for case let name as String in walker where name.hasSuffix(".swift") {
        let text = try String(
          contentsOf: base.appendingPathComponent(name), encoding: .utf8)
        try body("\(directory)/\(name)", text)
      }
    }
  }

  /// The word on its own, so the one type whose name begins with it and reads
  /// nothing but this machine's own pid is not a failure.
  private func word(_ needle: String, isIn line: String) -> Bool {
    var search = line.startIndex..<line.endIndex
    while let found = line.range(of: needle, range: search) {
      let before =
        found.lowerBound == line.startIndex ? nil : line[line.index(before: found.lowerBound)]
      let after = found.upperBound == line.endIndex ? nil : line[found.upperBound]
      let bounded = !isWordish(before) && !isWordish(after)
      if bounded { return true }
      search = found.upperBound..<line.endIndex
    }
    return false
  }

  private func isWordish(_ character: Character?) -> Bool {
    guard let character else { return false }
    return character.isLetter || character.isNumber || character == "_"
  }

  /// This file is Tests/PresenceCoreTests/SourceRulesTests.swift, so the
  /// package is three directories up from it. The path is the test file's own
  /// and never the working directory, which swift test does not promise.
  private func packageRoot() -> URL {
    URL(fileURLWithPath: #filePath)
      .deletingLastPathComponent()
      .deletingLastPathComponent()
      .deletingLastPathComponent()
  }
}
