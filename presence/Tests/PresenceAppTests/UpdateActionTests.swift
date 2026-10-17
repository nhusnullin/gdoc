// What Update does, and what it never does.
//
// The app never asks GitHub what is published: gdoc's own daily check does
// that, and the button says "Update" rather than "Check for updates" for
// that reason. So this room runs one argv list with no flag at all, and the
// gdoc row draws a ring until the child's end arrives. Install extension is
// the second list, `update --desktop`, and it is not an update of gdoc, so it
// moves no version of its own. It takes the same slot: both lists are `gdoc
// update`, so while either child is up neither row offers a button.
//
// No test here starts a program: the call is answered by a closure, and the
// one that holds the ring is answered by a gate this file opens.

import Foundation
import PresenceCore
import PresenceUI
import XCTest

@testable import PresenceApp

final class UpdateActionTests: XCTestCase {

  func testUpdateNeverPassesAFlag() async {
    let fake = FakeUpdater()
    let action = UpdateAction(answer: fake.answer)

    _ = await action.update()

    let calls = await fake.calls
    XCTAssertEqual(calls, [.update])
    XCTAssertEqual(Call.update.arguments, ["update"], "no flag, and none the app invented")
  }

  func testInstallExtensionIsUpdateDesktop() async {
    let fake = FakeUpdater()
    let action = UpdateAction(answer: fake.answer)

    _ = await action.installExtension()

    let calls = await fake.calls
    XCTAssertEqual(calls, [.updateDesktop])
    XCTAssertEqual(Call.updateDesktop.arguments, ["update", "--desktop"])

    // The extension is not a new gdoc, so the gdoc row is left where it was.
    let standing = await action.standing
    XCTAssertEqual(standing, .idle)
  }

  /// What the extension run hands back is `extension_opened`, and nothing
  /// else. gdoc writes the file before it opens it and leaves it there when
  /// the open refused, so the file is no answer; the field is the only one,
  /// and it is absent on the run that opened nothing.
  func testTheExtensionRunSaysWhetherDesktopWasHandedIt() async {
    let opened = UpdateAction(
      answer: FakeUpdater(
        data: #"{"installed":"v2.14.0","extension":"gdoc.mcpb","extension_opened":true}"#
      ).answer)
    let refused = UpdateAction(
      answer: FakeUpdater(data: #"{"installed":"v2.14.0","extension":"gdoc.mcpb"}"#).answer)

    let handedOver = await opened.installExtension()
    let notHandedOver = await refused.installExtension()

    XCTAssertTrue(handedOver, "Desktop was handed it")
    XCTAssertFalse(notHandedOver, "the file is there and Desktop was offered nothing")
  }

  func testTheGdocRowShowsTheRingUntilTheEndThenTheNewVersion() async {
    let gate = Gate()
    let fake = FakeUpdater(gate: gate)
    let action = UpdateAction(answer: fake.answer)
    let facts = Self.facts(installed: "v2.14.0", latest: "v2.15.0")

    // Before: the row offers the update it was told about.
    let before = Self.gdocRow(Self.facts(installed: "v2.14.0", latest: "v2.15.0"))
    XCTAssertEqual(before.value, "v2.14.0 · v2.15.0 available")
    XCTAssertFalse(before.waiting)

    // While the child is up: the ring stands in the glyph's place, and the
    // row offers nothing to press a second time.
    let running = Task { await action.update() }
    await gate.entered()
    let mid = await action.standing
    XCTAssertEqual(mid, .running)
    let ringing = Self.gdocRow(await action.applied(to: facts))
    XCTAssertTrue(ringing.waiting, "the gdoc row draws a ring while the update is up")
    XCTAssertEqual(ringing.value, "v2.14.0", "nothing says the new version before it is there")
    XCTAssertTrue(ringing.actions.isEmpty)

    // After the end: the version it installed, and no update left to take.
    await gate.open()
    let ended = await running.value
    XCTAssertEqual(ended, .installed("v2.15.0"))
    let after = Self.gdocRow(await action.applied(to: facts))
    XCTAssertFalse(after.waiting)
    XCTAssertEqual(after.value, "v2.15.0")
    XCTAssertTrue(after.actions.isEmpty, "there is nothing newer to take")
  }

  /// A run that installed nothing moves no version. `gdoc update` declines a
  /// major without `--major`, and the app passes no flag, so the version it
  /// names is one this app did not install.
  func testOnlyAnUpdatedActionWritesANewVersion() async {
    let fake = FakeUpdater(
      data: #"{"installed":"v2.14.0","action":"major_available","to":"v3.0.0"}"#)
    let action = UpdateAction(answer: fake.answer)
    let facts = Self.facts(installed: "v2.14.0", latest: "v3.0.0")

    let standing = await action.update()

    XCTAssertEqual(standing, .unchanged("major_available"))
    let row = Self.gdocRow(await action.applied(to: facts))
    XCTAssertEqual(row.value, "v2.14.0 · v3.0.0 available")
  }

  // MARK: the panel's own reading

  private static func facts(installed: String, latest: String) -> Facts {
    Facts(
      account: "someone@example.com", auth: .ok,
      versions: Versions(app: "2.14.0", gdoc: installed, latestStable: latest),
      gdocPath: "/Users/someone/.local/bin/gdoc")
  }

  private static func gdocRow(_ facts: Facts) -> HealthRowModel {
    let model = PanelModel.build(
      events: [], facts: facts, now: Date(timeIntervalSince1970: 1_792_000_000),
      calendar: Calendar(identifier: .gregorian), alive: { _ in false })
    return model.health.first { $0.kind == .gdoc }!
  }
}

/// The child, as a closure, with what it printed.
private actor FakeUpdater {
  private(set) var calls: [Call] = []
  private let gate: Gate?
  private let data: String

  init(
    gate: Gate? = nil,
    data: String = #"{"installed":"v2.14.0","action":"updated","to":"v2.15.0","verified":true}"#
  ) {
    self.gate = gate
    self.data = data
  }

  nonisolated var answer: @Sendable (Call) async throws -> Outcome {
    { [self] call in try await said(call) }
  }

  private func said(_ call: Call) async throws -> Outcome {
    calls.append(call)
    await gate?.enter()
    return Outcome(
      call: call, envelope: Envelope(ok: true, data: Data(data.utf8), version: "v2.15.0"),
      standardError: "", status: 0)
  }
}

/// A child that is still up until this test lets it finish.
private actor Gate {
  private var waiting: [CheckedContinuation<Void, Never>] = []
  private var arrivals: [CheckedContinuation<Void, Never>] = []
  private var isOpen = false
  private var hasArrived = false

  /// Called by the child: it stops here until the test opens the gate.
  func enter() async {
    hasArrived = true
    for arrival in arrivals { arrival.resume() }
    arrivals = []
    guard !isOpen else { return }
    await withCheckedContinuation { waiting.append($0) }
  }

  /// Called by the test: it waits for the child to be inside.
  func entered() async {
    guard !hasArrived else { return }
    await withCheckedContinuation { arrivals.append($0) }
  }

  func open() {
    isOpen = true
    for one in waiting { one.resume() }
    waiting = []
  }
}
