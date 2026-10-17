// What the menu bar shows, and what the panel reads.
//
// Two things are under test here. The glyph is one of four states, chosen by
// priority from one glance, and every one of them is monochrome, because the
// menu bar carries no colour. The panel is a model of rows built from the
// same glance, so every row a person can see is a fixture and not a
// screenshot somebody remembers.
//
// The renders are the one place pixels are made. They go to a directory
// outside the repository, named by PRESENCE_SHOTS, so a look at them is a
// person opening two files and never a committed image nobody compares.

import AppKit
import PresenceCore
import SwiftUI
import XCTest

@testable import PresenceUI

@MainActor
final class RenderTests: XCTestCase {
  // The glance the panel renders from: the shared day fixture, folded at a
  // moment inside it, with one pid dead so a stopped row is there.
  private let dayMoment = "2026-10-17T12:00:00Z"
  private let deadPid: Int32 = 4299

  func testTheGlyphIsOneOfFourStatesByPriority() throws {
    let waiting = try event(.held, at: "2026-10-17T11:00:00Z", pid: 11, run: "r1", hold: "h_1")
    let running = try event(.start, at: "2026-10-17T11:01:00Z", pid: 12, run: "r2", command: "read")
    let watching = try event(
      .start, at: "2026-10-17T11:02:00Z", pid: 13, run: "r3", command: "comments", wait: true)
    let stopped = try event(
      .start, at: "2026-10-17T11:03:00Z", pid: 99, run: "r4", command: "publish")
    let alive: (Int32) -> Bool = { $0 != 99 }

    XCTAssertEqual(
      Glyph.state(for: try fold([waiting, running, watching], alive: alive)), .hold,
      "a hold is the one row waiting for the person, so it is over everything")
    XCTAssertEqual(
      Glyph.state(for: try fold([running, watching], alive: alive)), .running,
      "a finite job is over a watch that may sit there all day")
    XCTAssertEqual(Glyph.state(for: try fold([watching], alive: alive)), .watching)
    XCTAssertEqual(Glyph.state(for: try fold([], alive: alive)), .idle)
    XCTAssertEqual(
      Glyph.state(for: try fold([stopped], alive: alive)), .idle,
      "a start whose pid is gone is a row in the panel and never a glyph in the menu bar")

    for state in [GlyphState.hold, .running, .watching] {
      let image = try XCTUnwrap(Glyph.image(for: state), "\(state) has no glyph")
      XCTAssertTrue(
        image.isTemplate, "the menu bar is monochrome, so \(state) must be a template image")
      XCTAssertGreaterThan(image.size.width, 0)
      XCTAssertFalse(Glyph.tooltip(for: state).isEmpty)
    }
    XCTAssertNil(Glyph.image(for: .idle), "idle shows nothing at all")
  }

  // A hold leaves the menu bar when its life has run out, the way a start
  // leaves it when its pid is gone. The binary writes the line that says a
  // hold expired on its next tool call, so a chat window somebody closed with
  // a card open writes none at all: the hold's own `until` is the whole of
  // what the record can say, and a pause glyph that stays lit for a card
  // nobody can press any more is the icon nobody believes.
  //
  // The row stays in the panel, saying it ran out, because what happened is
  // still worth reading.
  func testAHoldThatRanOutLeavesTheMenuBar() throws {
    let waiting = try event(
      .held, at: "2026-10-17T11:50:00Z", pid: 11, run: "r1", hold: "h_1",
      until: "2026-10-17T12:20:00Z")
    let ranOut = try event(
      .held, at: "2026-10-17T11:00:00Z", pid: 12, run: "r2", hold: "h_2",
      until: "2026-10-17T11:30:00Z")
    let alive: (Int32) -> Bool = { _ in true }

    XCTAssertEqual(
      Glyph.state(for: try fold([waiting], alive: alive)), .hold,
      "a hold inside its life is the one row waiting for the person")
    let gone = try fold([ranOut], alive: alive)
    XCTAssertEqual(
      Glyph.state(for: gone), .idle,
      "a hold past its until is waiting for nobody, so the menu bar says nothing")
    XCTAssertEqual(
      Glyph.state(for: try fold([waiting, ranOut], alive: alive)), .hold,
      "one hold that ran out does not take the glyph off a hold that has not")

    let row = try XCTUnwrap(
      PanelModel.build(
        glance: gone, appVersion: "2.13.0", bundle: .same, calendar: try amsterdam()
      ).now.first, "the hold is still a row in the panel")
    XCTAssertEqual(row.kind, .hold)
    XCTAssertEqual(row.value, "ran out")
  }

  func testThePanelRendersNowAndTodayInLightAndDark() throws {
    let model = try dayModel()
    XCTAssertFalse(model.now.isEmpty, "the day fixture holds a hold, a watch and a live run")
    XCTAssertFalse(model.today.isEmpty, "the day fixture holds three commands that ended today")

    let shots = try shotsDirectory()
    for scheme in [ColorScheme.light, .dark] {
      let name = scheme == .light ? "panel-light.png" : "panel-dark.png"
      let png = try XCTUnwrap(
        render(model, scheme: scheme), "the panel did not render in \(name)")
      let file = shots.appendingPathComponent(name)
      try png.write(to: file)
      XCTAssertGreaterThan(
        png.count, 2_000, "\(name) is too small to be a panel: \(file.path)")
    }
  }

  func testAStoppedRunWithUnknownOutcomeIsItsOwnRow() throws {
    let model = try dayModel()
    let stopped = model.now.filter { $0.kind == .stopped }
    XCTAssertEqual(stopped.count, 1, "one start in the day fixture has a dead pid")
    XCTAssertEqual(stopped.first?.leading, "propose")
    XCTAssertEqual(
      stopped.first?.detail, "stopped, outcome unknown",
      "the record cannot say whether it worked, so the row says exactly that")
    XCTAssertEqual(
      stopped.first?.mark, RowMark.none, "nothing here is a judgement about the outcome")

    let live = model.now.filter { $0.kind == .running || $0.kind == .watching }
    XCTAssertEqual(live.count, 2, "the other two starts in the fixture are alive")
  }

  func testTheVersionRowsNameTheAppAndTheLastGdoc() throws {
    let model = try dayModel()
    XCTAssertEqual(model.versions.count, 2)
    XCTAssertEqual(model.versions[0].kind, .version)
    XCTAssertEqual(model.versions[0].leading, "gdoc presence")
    XCTAssertEqual(
      model.versions[0].value, "2.13.0", "the app says the version it launched at")
    XCTAssertEqual(model.versions[1].leading, "gdoc")
    XCTAssertEqual(
      model.versions[1].value, "v2.13.0",
      "the binary's version is the last one a record named, because the two may differ")

    let quiet = try built(of: [], appVersion: "0.0.0", bundle: .same)
    XCTAssertEqual(
      quiet.versions[1].value, "checkout build",
      "a gdoc built from a checkout names no release, and the row says so")
  }

  func testTheReopenRowAppearsWhenThePlistVersionDiffers() throws {
    XCTAssertNil(
      try dayModel(bundle: .same).notice, "nothing is said while the disk holds what is running")

    let updated = try dayModel(bundle: .reopenNeeded("2.14.0"))
    XCTAssertEqual(updated.notice?.kind, .notice)
    XCTAssertEqual(updated.notice?.leading, "Updated to 2.14.0. Quit and reopen")

    let rolled = try dayModel(bundle: .reopenNeeded("2.12.0"))
    XCTAssertEqual(
      rolled.notice?.leading, "Updated to 2.12.0. Quit and reopen",
      "a rollback is a difference like any other and the row names the version that is there")

    XCTAssertEqual(try dayModel(bundle: .gone).notice?.leading, "gdoc presence was removed")
  }

  // MARK: the fixture

  /// The shared day fixture folded into a model, which is what every test
  /// here reads except the glyph's.
  private func dayModel(bundle: BundleState = .same) throws -> PanelModel {
    let data = try Data(contentsOf: fixture("day"))
    return try built(of: Event.scan(data).events, appVersion: "2.13.0", bundle: bundle)
  }

  private func built(
    of events: [Event], appVersion: String, bundle: BundleState
  ) throws -> PanelModel {
    PanelModel.build(
      glance: try fold(events, alive: { [deadPid] in $0 != deadPid }),
      appVersion: appVersion, bundle: bundle, calendar: try amsterdam())
  }

  private func fold(_ events: [Event], alive: (Int32) -> Bool) throws -> Glance {
    Fold.glance(
      events, now: try moment(dayMoment), calendar: try amsterdam(), alive: alive)
  }

  /// One event, written as a line and decoded, so a test states the record
  /// and never the struct the record happens to decode into.
  private func event(
    _ kind: EventKind, at: String, pid: Int32, run: String, call: Int64? = 1,
    command: String? = nil, wait: Bool = false, hold: String? = nil, until: String? = nil
  ) throws -> Event {
    var fields: [String] = [
      "\"v\":1", "\"at\":\"\(at)\"", "\"pid\":\(pid)", "\"run\":\"\(run)\"",
      "\"event\":\"\(kind.rawValue)\"",
    ]
    if let call { fields.append("\"call\":\(call)") }
    if let command { fields.append("\"command\":\"\(command)\"") }
    if wait { fields.append("\"wait\":true") }
    if let hold { fields.append("\"hold\":\"\(hold)\"") }
    if let until { fields.append("\"until\":\"\(until)\"") }
    return try Event.decode("{" + fields.joined(separator: ",") + "}")
  }

  /// The fixture both languages read, taken off the disk rather than out of a
  /// bundle, because this target copies nothing: the one copy of day.jsonl is
  /// the core's.
  private func fixture(_ name: String) -> URL {
    packageRoot()
      .appendingPathComponent("Tests/PresenceCoreTests/Fixtures/\(name).jsonl")
  }

  // MARK: the renders

  private func render(_ model: PanelModel, scheme: ColorScheme) -> Data? {
    let renderer = ImageRenderer(
      content: PanelView(model: model, onQuit: {}).environment(\.colorScheme, scheme))
    renderer.scale = 2
    var png: Data?
    let appearance = NSAppearance(named: scheme == .dark ? .darkAqua : .aqua)
    appearance?.performAsCurrentDrawingAppearance {
      guard let image = renderer.cgImage else { return }
      png = NSBitmapImageRep(cgImage: image).representation(using: .png, properties: [:])
    }
    return png
  }

  /// Where the PNGs go. Outside the repository, always: a render is something
  /// a person looks at once, and a committed screenshot is a file nobody
  /// compares against the code that drew it.
  private func shotsDirectory() throws -> URL {
    let named = ProcessInfo.processInfo.environment["PRESENCE_SHOTS"]
    let temporary = URL(fileURLWithPath: NSTemporaryDirectory())
      .appendingPathComponent("gdoc-presence-shots")
    let base = named.map { URL(fileURLWithPath: $0) } ?? temporary
    let repository = packageRoot().deletingLastPathComponent().standardizedFileURL.path
    XCTAssertFalse(
      base.standardizedFileURL.path.hasPrefix(repository),
      "PRESENCE_SHOTS names \(base.path), which is inside the repository")
    try FileManager.default.createDirectory(at: base, withIntermediateDirectories: true)
    return base
  }

  /// This file is Tests/PresenceUITests/RenderTests.swift, so the package is
  /// three directories up from it.
  private func packageRoot() -> URL {
    URL(fileURLWithPath: #filePath)
      .deletingLastPathComponent()
      .deletingLastPathComponent()
      .deletingLastPathComponent()
  }
}

/// A moment as the record writes one, so a test names its time as a literal.
private func moment(_ text: String) throws -> Date {
  let formatter = ISO8601DateFormatter()
  formatter.formatOptions = [.withInternetDateTime]
  return try XCTUnwrap(formatter.date(from: text), "\(text) is not a moment")
}

/// The one zone every test here folds and formats in. The machine's own zone
/// is not the thing under test.
private func amsterdam() throws -> Calendar {
  var calendar = Calendar(identifier: .gregorian)
  calendar.timeZone = try XCTUnwrap(TimeZone(identifier: "Europe/Amsterdam"))
  calendar.locale = Locale(identifier: "en_US_POSIX")
  return calendar
}
