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
// front of somebody: the system is asked through two files, one application
// is opened at a path by one of them, this machine's login items are touched
// by one file, and the only addresses any of them can build are two schemes
// and two hosts. A line of the record is forgeable, so a click that came out
// of one must have nowhere else to go.

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

  /// The executable target, which is the shell: the status item, the
  /// window, and the one signal between them.
  private static let shell = "Presence"

  /// What a file of the shell may import. AppKit and SwiftUI because a
  /// status item and a window are made of them, Foundation for a date, and
  /// the two libraries it is wired out of.
  private static let shellImports = [
    "AppKit", "Foundation", "PresenceApp", "PresenceUI", "SwiftUI",
  ]

  /// What no declaration in the shell may be about. An event and a record
  /// are the core's, a call and an outcome are a child the app library
  /// started, and a URL is a path or an address somebody else resolved.
  private static let notTheShells = ["Event", "Records", "Glance", "Call", "Outcome", "URL"]

  /// The question that signals nothing: a zero asks whether a process is
  /// there and is the whole of Liveness. Any other number is a signal.
  private static let asking = "0"

  /// The schemes the app may name. `https` is a document and a sign-in;
  /// `obsidian` is a note in the vault the person chose; `claude` is the one
  /// path of Claude Desktop's own scheme that names a session, built in
  /// LinkLine and opened by nothing, because the measurement of 2026-10-18
  /// found that it switches no window from outside the app. A fourth is a
  /// decision in `docs/v2/DECISIONS.md`, as this third one is.
  private static let schemes = ["https", "obsidian", "claude"]

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
  /// The app's own eight keys are the whole of what outlives the process. A
  /// ninth key, or a second file writing one, is a decision in
  /// `docs/v2/DECISIONS.md`: everything else this app knows comes out of the
  /// record, which is gdoc's file and not the app's.
  ///
  /// Eight since the second morning, and the two that were added are the
  /// reason this count is written down: the pins and the hides name
  /// somebody's documents, which every other key refuses to, and a pin
  /// keeps the last title as well, because a pinned document whose lines
  /// are past the horizon has its name nowhere else. That is the decision of
  /// 2026-10-18, and a ninth key is the next one.
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
  /// The opener is the four verbs the panel has, which are what a row's
  /// icons do. The relauncher opens this app's own bundle and nothing else.
  /// Both names are here rather than in a review, because the next person
  /// to add a convenience will not have read the specification.
  ///
  /// The folder panel was the fifth verb until the third look. The note is
  /// found rather than configured now, so this app opens no window of the
  /// system's at all, and `NSOpenPanel` is named in neither file.
  func testTheSystemIsAskedOnlyThroughTheOpenerAndTheRelauncher() throws {
    let workspace = "NSWork" + "space"
    let board = "NSPaste" + "board"
    let running = "NSRunning" + "Application"
    let chooser = "NSOpen" + "Panel"

    var asking: Set<String> = []
    try eachSwiftFile { path, text in
      guard path.hasPrefix("Sources/") else { return }
      XCTAssertFalse(
        word(chooser, isIn: text), "\(path) opens a window of the system's")
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

  /// One file opens an application at a path, and the application is this
  /// app.
  ///
  /// The scan is of `openApplication`, which takes a bundle URL the caller
  /// built and is the one call that could start anything at all, so it is
  /// held to the file that opens this app's own bundle, with the new
  /// instance and the no-activation the specification names. The opener can
  /// also bring an application up, by the identifier of Claude Desktop and
  /// no other: `SystemDesk.activate` asks the workspace for that bundle's
  /// own URL rather than naming a path, which is held by the scheme and
  /// host test below and by the identifier being a constant.
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

  /// The shell wires and holds nothing.
  ///
  /// An executable target cannot be imported by a test target, so anything
  /// in `Sources/Presence` is code no test can reach. The rule that keeps
  /// that from mattering is this one: the shell imports the two libraries
  /// and AppKit, and declares nothing over an event, a record, a call or a
  /// URL. Every reading is the core's, every child and every store is the
  /// app library's, and what is left here is three objects and one signal.
  func testTheShellOnlyWiresAndHoldsNoLogic() throws {
    var imported: Set<String> = []
    var files = 0

    try eachSwiftFile { path, text in
      guard path.hasPrefix("Sources/\(Self.shell)/") else { return }
      files += 1
      for module in importsNamed(in: text) {
        XCTAssertTrue(
          Self.shellImports.contains(module), "\(path) imports \(module)")
        imported.insert(module)
      }
      for line in text.split(separator: "\n") where declares(String(line)) {
        for subject in Self.notTheShells {
          XCTAssertFalse(
            word(subject, isIn: String(line)),
            "\(path) declares something over a \(subject): "
              + line.trimmingCharacters(in: .whitespaces))
        }
      }
    }

    XCTAssertGreaterThan(files, 0, "the shell is not there, so this test guards nothing")
    XCTAssertTrue(imported.contains("PresenceApp"), "the shell is made of the app library")
    XCTAssertTrue(imported.contains("PresenceUI"), "and of the views")
    XCTAssertFalse(
      imported.contains("PresenceCore"),
      "the shell reads nothing itself, so it never names the core")
  }

  /// Whether this line declares a function or an initialiser, which is where
  /// a shell that had started reading something would say so.
  private func declares(_ line: String) -> Bool {
    let trimmed = line.trimmingCharacters(in: .whitespaces)
    return word("func", isIn: trimmed) || trimmed.contains("init(")
  }

  /// The modules a file imports, each as its own name.
  private func importsNamed(in text: String) -> [String] {
    text.split(separator: "\n").compactMap { line in
      let trimmed = line.trimmingCharacters(in: .whitespaces)
      guard trimmed.hasPrefix("import ") else { return nil }
      return String(trimmed.dropFirst("import ".count))
        .split(separator: ".").first.map(String.init)?
        .trimmingCharacters(in: .whitespaces)
    }
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
