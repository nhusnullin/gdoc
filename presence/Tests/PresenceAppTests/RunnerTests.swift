// What the one room that starts a program is held to.
//
// Every test here runs Fixtures/fake-gdoc.sh, which answers the six argv
// lists and tells the test what reached it. The real binary is never run: it
// would read somebody's token, ask Google about it, and make the suite depend
// on a machine.
//
// The resolution, the argv set and the envelope are pure functions of their
// arguments, so those tests state every value as a literal.

import Foundation
import XCTest

@testable import PresenceApp

final class RunnerTests: XCTestCase {

  // MARK: where the binary is

  func testGdocIsResolvedAtLocalBinThenBesideTheBundleNeverFromPATH() throws {
    let home = "/Users/someone"
    let bundle = URL(fileURLWithPath: "/Users/someone/Applications/GdocPresence.app")
    let localBin = "/Users/someone/.local/bin/gdoc"
    let besideBundle = "/Users/someone/Applications/gdoc"
    let onPath = "/usr/local/bin/gdoc"

    let both = Runner.locate(
      home: home, bundle: bundle,
      exists: { [localBin, besideBundle, onPath].contains($0) })
    XCTAssertEqual(both.candidates, [localBin, besideBundle], "two places, in that order")
    XCTAssertEqual(both.path, localBin, "the installed binary comes first")

    let beside = Runner.locate(home: home, bundle: bundle, exists: { $0 == besideBundle })
    XCTAssertEqual(beside.path, besideBundle, "an unzipped release is the second place")

    let onlyOnPath = Runner.locate(home: home, bundle: bundle, exists: { $0 == onPath })
    XCTAssertNil(onlyOnPath.path, "a gdoc on PATH is not a gdoc this app runs")
    XCTAssertFalse(onlyOnPath.found)
    XCTAssertEqual(
      onlyOnPath.tried, "~/.local/bin/gdoc or ~/Applications/gdoc",
      "the not-found fact names both places, with the home directory as a tilde")

    // The checkout route: a link at the local bin, resolved before it is run,
    // so the path the app launches is the path the kernel will name for it.
    let linked = Runner.locate(
      home: home, bundle: bundle, exists: { $0 == localBin },
      resolve: { $0 == localBin ? "/Users/someone/src/gdoc/bin/gdoc" : $0 })
    XCTAssertEqual(linked.path, "/Users/someone/src/gdoc/bin/gdoc")

    let unbundled = Runner.locate(home: home, bundle: nil, exists: { _ in false })
    XCTAssertEqual(unbundled.candidates, [localBin], "no bundle, no second place")
  }

  // MARK: what it runs

  func testTheArgvSetIsExactlyTheSix() {
    XCTAssertEqual(Call.allCases.count, 6)
    XCTAssertEqual(Call.help.arguments, ["help", "--json"])
    XCTAssertEqual(Call.authStatus.arguments, ["auth", "status"])
    XCTAssertEqual(Call.authLogin.arguments, ["auth", "login"])
    XCTAssertEqual(Call.authLogout.arguments, ["auth", "logout"])
    XCTAssertEqual(Call.update.arguments, ["update"])
    XCTAssertEqual(Call.updateDesktop.arguments, ["update", "--desktop"])

    // Two flags in the whole set, and each is a flag the command has for
    // everybody. No setting flag, and no flag only the app would pass.
    let flags = Call.allCases.flatMap(\.arguments).filter { $0.hasPrefix("-") }
    XCTAssertEqual(flags.sorted(), ["--desktop", "--json"])
  }

  func testTheChildEnvironmentIsTheFourVariables() async throws {
    let parent = [
      "HOME": "/Users/someone",
      "PATH": "/usr/bin:/bin",
      "LANG": "en_GB.UTF-8",
      "GDOC_CONFIG_DIR": "/tmp/somewhere-else",
      "TERM": "xterm-256color",
      "SSH_AUTH_SOCK": "/private/tmp/socket",
    ]
    let reduced = Runner.childEnvironment(from: parent)
    XCTAssertEqual(Set(reduced.keys), ["HOME", "PATH", "LANG", "NO_COLOR"])
    XCTAssertEqual(reduced["HOME"], "/Users/someone")
    XCTAssertEqual(reduced["PATH"], "/usr/bin:/bin")
    XCTAssertEqual(reduced["LANG"], "en_GB.UTF-8")
    XCTAssertEqual(reduced["NO_COLOR"], "1", "the app reads JSON, so colour is off")

    // And over the fake, with the variable set in this test's own
    // environment: what the app is pointed at is never what the child reads.
    setenv("GDOC_CONFIG_DIR", "/tmp/gdoc-agent-under-test", 1)
    defer { unsetenv("GDOC_CONFIG_DIR") }
    XCTAssertNotNil(
      ProcessInfo.processInfo.environment["GDOC_CONFIG_DIR"], "the test holds it itself")

    let fromTheFake = try child(of: await runner().run(.help))
    XCTAssertEqual(fromTheFake["config_dir"], "", "the config directory never reaches the child")
    XCTAssertEqual(fromTheFake["no_color"], "1")
    XCTAssertEqual(fromTheFake["path_set"], "yes")
  }

  func testStdinIsTheNullDevice() async throws {
    // This test's own standard input is a pipe with a line waiting in it, so
    // a child that inherited it would read that line and say so.
    let planted = Pipe()
    planted.fileHandleForWriting.write(Data("planted\n".utf8))
    try planted.fileHandleForWriting.close()
    let saved = dup(0)
    XCTAssertEqual(dup2(planted.fileHandleForReading.fileDescriptor, 0), 0)
    defer {
      dup2(saved, 0)
      close(saved)
    }

    let seen = try child(of: await runner().run(.help))
    XCTAssertEqual(seen["stdin"], "", "the child read nothing")
    XCTAssertEqual(seen["stdin_eof"], "yes", "the child's standard input was at its end at once")
  }

  func testOneChildPerKindAtATime() async throws {
    let runner = runner()
    let slow = try runner.start(.update)
    XCTAssertTrue(runner.isRunning(.update))

    do {
      _ = try runner.start(.update)
      XCTFail("a second update was launched")
    } catch {
      XCTAssertEqual(error as? RunFailure, .busy(.update))
    }

    // Another kind is another child, and runs while the first one is up.
    let other = try runner.start(.help)
    let seen = try child(of: await other.outcome())
    XCTAssertEqual(seen["no_color"], "1")

    slow.terminate()
    _ = try? await slow.outcome()
    XCTAssertFalse(runner.isRunning(.update), "the kind is free again")
  }

  // MARK: what comes back

  func testTheEnvelopeIsDecodedStrictly() async throws {
    let good = try Envelope.decode(
      Data(#"{"ok":true,"data":{"a":1},"version":"v2.14.0","warnings":["careful"]}"#.utf8))
    XCTAssertTrue(good.ok)
    XCTAssertEqual(good.version, "v2.14.0")
    XCTAssertEqual(good.warnings, ["careful"])
    XCTAssertNotNil(good.data)

    let failed = try Envelope.decode(Data(#"{"ok":false,"error":"the quote is not there"}"#.utf8))
    XCTAssertFalse(failed.ok)
    XCTAssertEqual(failed.error, "the quote is not there")
    XCTAssertNil(failed.data)
    XCTAssertEqual(failed.warnings, [])

    // A field a newer gdoc prints is read past, as the record's own decoder
    // reads past one: an app that refused it would break on a release.
    let newer = try Envelope.decode(Data(#"{"ok":true,"tomorrow":{"deep":7}}"#.utf8))
    XCTAssertTrue(newer.ok)

    let refusals: [(String, EnvelopeFault)] = [
      ("", .notJSON),
      ("not json at all", .notJSON),
      (#"{"ok":true} and then some"#, .notJSON),
      (#"{"ok":true}{"ok":false}"#, .notJSON),
      (#"[{"ok":true}]"#, .notAnObject),
      (#"{"data":{}}"#, .noOK),
      (#"{"ok":"yes"}"#, .badOK),
      (#"{"ok":1}"#, .badOK),
      (#"{"ok":true,"error":3}"#, .badError),
      (#"{"ok":true,"warnings":"careful"}"#, .badWarnings),
      (#"{"ok":true,"warnings":["careful",3]}"#, .badWarnings),
      (#"{"ok":true,"version":2}"#, .badVersion),
      (#"{"ok":true,"data":7}"#, .badData),
    ]
    for (text, fault) in refusals {
      do {
        _ = try Envelope.decode(Data(text.utf8))
        XCTFail("\(text) was read as an envelope")
      } catch {
        XCTAssertEqual(error as? RunFailure, .envelope(fault), "for \(text)")
      }
    }

    // And the fake's own object, through the whole runner.
    let outcome = try await runner().run(.authStatus)
    XCTAssertTrue(outcome.envelope.ok)
    XCTAssertEqual(outcome.envelope.version, "v2.14.0")
    XCTAssertEqual(outcome.status, 0)
    XCTAssertEqual(outcome.call, .authStatus)

    // `auth login` prints its link on stderr, which is where sign-in reads it.
    let login = try await runner().run(.authLogin)
    XCTAssertTrue(
      login.standardError.contains("https://accounts.google.com/o/oauth2/auth"),
      "the link reached the app: \(login.standardError)")
  }

  func testATimeoutTerminatesTheChild() async throws {
    let runner = runner(timeout: 0.3)
    let slow = try runner.start(.update)
    do {
      _ = try await slow.outcome()
      XCTFail("the slow child answered")
    } catch {
      XCTAssertEqual(error as? RunFailure, .timedOut(after: 0.3))
    }
    XCTAssertFalse(slow.isRunning, "the child was terminated")
    XCTAssertFalse(runner.isRunning(.update), "the kind is free again")
  }

  func testAMissingBinaryIsRefusedByName() async throws {
    let nowhere = Runner(
      location: Location(
        path: nil, candidates: ["/nowhere/gdoc"], tried: "~/.local/bin/gdoc"),
      timeout: 1)
    do {
      _ = try await nowhere.run(.help)
      XCTFail("a runner with no binary ran something")
    } catch {
      XCTAssertEqual(error as? RunFailure, .notFound(tried: "~/.local/bin/gdoc"))
    }
  }

  // MARK: the fake

  /// The fake is this file's neighbour, read at its own path rather than out
  /// of a test bundle, so the mode bits git holds are the mode bits it runs
  /// with.
  private static let fake = URL(fileURLWithPath: #filePath)
    .deletingLastPathComponent()
    .appendingPathComponent("Fixtures")
    .appendingPathComponent("fake-gdoc.sh")
    .path

  private func runner(timeout: TimeInterval = 10) -> Runner {
    Runner(
      location: Location(path: Self.fake, candidates: [Self.fake], tried: Self.fake),
      timeout: timeout)
  }

  /// What the child said about itself, out of the envelope's data.
  private func child(of outcome: Outcome) throws -> [String: String] {
    let bytes = try XCTUnwrap(outcome.envelope.data, "the fake printed no data")
    let object = try JSONSerialization.jsonObject(with: bytes) as? [String: Any]
    return try XCTUnwrap(
      object?["child"] as? [String: String], "no child field in \(bytes.count) bytes")
  }
}
