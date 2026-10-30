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

    XCTAssertEqual(handedOver, .handedOver, "Desktop was handed it")
    XCTAssertEqual(
      notHandedOver, .unchanged(""),
      "the file is there and Desktop was offered nothing")
    XCTAssertNil(handedOver.miss, "so the row has nothing to say")
    XCTAssertEqual(notHandedOver.miss, .handedNothing)
  }

  /// Each run of either button is read by `ok` and the action word alone,
  /// and what comes out is one of the cases the panel has a sentence for.
  /// The child's own sentence reaches neither: the warning gdoc writes
  /// about a declined major is a paragraph about this machine, and the
  /// refusal's error is the terminal's.
  func testTheRunIsReadByOkAndTheActionWordAlone() async {
    let declined = UpdateAction(
      answer: FakeUpdater(
        data: #"{"installed":"v2.14.0","action":"major_available","to":"v3.0.0"}"#,
        warning: "--desktop writes the extension of the gdoc that is installed"
      ).answer)
    let unreachable = UpdateAction(
      answer: FakeUpdater(data: #"{"installed":"v2.14.0","action":"unreachable"}"#).answer)
    let refused = UpdateAction(answer: FakeUpdater(ok: false, error: "the zip is short").answer)
    let unreadable = UpdateAction(answer: FakeUpdater(data: #"{"action":42}"#).answer)
    let rehearsed = UpdateAction(
      answer: FakeUpdater(data: #"{"installed":"v2.14.0","action":"checked"}"#).answer)

    let declinedUpdate = await declined.update().miss
    let unreachableUpdate = await unreachable.update().miss
    let refusedUpdate = await refused.update().miss
    let unreadableUpdate = await unreadable.update().miss
    let rehearsedUpdate = await rehearsed.update().miss
    XCTAssertEqual(declinedUpdate, .major)
    XCTAssertEqual(unreachableUpdate, .unreachable)
    XCTAssertEqual(refusedUpdate, .stopped)
    XCTAssertEqual(unreadableUpdate, .stopped, "an answer gdoc would not print")
    XCTAssertEqual(
      rehearsedUpdate, .stopped,
      "the app passes no --check, so a run that printed one is not this button's")

    let declinedExtension = await declined.installExtension().miss
    let unreachableExtension = await unreachable.installExtension().miss
    let refusedExtension = await refused.installExtension().miss
    let unreadableExtension = await unreadable.installExtension().miss
    XCTAssertEqual(declinedExtension, .major)
    XCTAssertEqual(
      unreachableExtension, .handedNothing,
      "the extension is written from a release, and this run read none")
    XCTAssertEqual(refusedExtension, .stopped)
    XCTAssertEqual(unreadableExtension, .stopped)
  }

  /// A machine already on the newest release and a run that installed
  /// something both have their own answer on the screen, so neither writes
  /// a sentence.
  func testARunThatDidSomethingWritesNoSentence() async {
    let newest = UpdateAction(
      answer: FakeUpdater(data: #"{"installed":"v2.14.0","action":"up_to_date"}"#).answer)
    let installed = UpdateAction(answer: FakeUpdater().answer)

    let onTheNewest = await newest.update().miss
    let replaced = await installed.update().miss
    XCTAssertNil(onTheNewest, "the check for a second is the answer")
    XCTAssertNil(replaced, "and the new version is")
  }

  func testTheGdocRowShowsTheRingUntilTheEndThenTheNewVersion() async {
    let gate = Gate()
    let fake = FakeUpdater(gate: gate)
    let action = UpdateAction(answer: fake.answer)
    let facts = Self.facts(installed: "v2.14.0", latest: "v2.15.0")

    // Before: the row offers the update it was told about.
    let before = Self.gdocHeader(Self.facts(installed: "v2.14.0", latest: "v2.15.0"))
    XCTAssertEqual(before.version, "v2.14.0")
    XCTAssertEqual(before.update?.tip, "Update to v2.15.0")
    XCTAssertFalse(before.versionWaiting)

    // While the child is up: the ring stands in the glyph's place, and the
    // row offers nothing to press a second time.
    let running = Task { await action.update() }
    await gate.entered()
    let mid = await action.standing
    XCTAssertEqual(mid, .running)
    let ringing = Self.gdocHeader(await action.applied(to: facts))
    XCTAssertTrue(
      ringing.versionWaiting, "the header draws a ring while the update is up")
    XCTAssertEqual(
      ringing.version, "v2.14.0", "nothing says the new version before it is there")
    XCTAssertNil(ringing.update)

    // After the end: the version it installed, and no update left to take.
    await gate.open()
    let ended = await running.value
    XCTAssertEqual(ended, .installed("v2.15.0"))
    let after = Self.gdocHeader(await action.applied(to: facts))
    XCTAssertFalse(after.versionWaiting)
    XCTAssertEqual(after.version, "v2.15.0")
    XCTAssertEqual(
      after.updateLook, .quiet, "there is nothing newer to take, so the glyph goes quiet")
    XCTAssertTrue(after.quiet.isEmpty, "and no line names a release")
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
    XCTAssertFalse(
      standing.isCurrent,
      "a major this app declined to install is not a machine on the newest release")
    let row = Self.gdocHeader(await action.applied(to: facts))
    XCTAssertEqual(row.version, "v2.14.0")
    XCTAssertEqual(row.update?.tip, "Update to v3.0.0")
  }

  /// The version this app installed moves the row, and stops moving it the
  /// moment a reading says something else.
  ///
  /// The override is there for the second between the child's end and the
  /// next `help --json`, where the last reading still names the version that
  /// was replaced. It is bounded by that version: a later reading naming
  /// anything else is newer than this run, which is what a `gdoc update` run
  /// in a terminal after this one leaves behind. Without the bound the row
  /// would name this run's version for the life of the process, and offer an
  /// update for a release already installed.
  func testTheInstalledVersionStopsOverridingALaterReading() async {
    let action = UpdateAction(answer: FakeUpdater().answer)

    let standing = await action.update()
    XCTAssertEqual(standing, .installed("v2.15.0"))
    XCTAssertFalse(standing.isCurrent, "a run that installed something shows no check")

    // The stale reading: the poll has not answered yet, so it still says the
    // version this run replaced.
    let stale = Self.gdocHeader(
      await action.applied(to: Self.facts(installed: "v2.14.0", latest: "v2.15.0")))
    XCTAssertEqual(stale.version, "v2.15.0", "the child's end is newer than the last poll")

    // A terminal updated gdoc again. The reading wins, and no update is
    // offered for a release that is already there.
    let later = Self.gdocHeader(
      await action.applied(to: Self.facts(installed: "v2.16.0", latest: "v2.16.0")))
    XCTAssertEqual(later.version, "v2.16.0", "a reading this run did not write stands")
    XCTAssertEqual(later.updateLook, .quiet, "and offers nothing to install")
  }

  /// Every fact this room was handed comes back, the one picture included.
  ///
  /// `applied(to:)` rebuilds the facts field by field, and the session runs
  /// it after the poller's own, so a field it forgot would be a mark that
  /// went out every time a person pressed Update.
  func testTheLogoSurvivesTheUpdateActionsRebuild() async {
    let action = UpdateAction(answer: FakeUpdater().answer)
    let bytes = Data([0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x7A, 0x7A])

    let kept = await action.applied(
      to: Facts(versions: Versions(app: "2.14.0"), logo: bytes))
    XCTAssertEqual(kept.logo, bytes, "the bytes the poller read are the bytes that stay")

    let none = await action.applied(to: Facts(versions: Versions(app: "2.14.0")))
    XCTAssertNil(none.logo, "and a machine with no picture invents none")
  }

  /// The page the mark opens comes back too.
  ///
  /// It rides beside the bytes for the same reason: a field this rebuild
  /// forgot would be a press that died every time a person pressed Update.
  /// The id is a stand-in of the right shape.
  func testTheLogosPageSurvivesTheUpdateActionsRebuild() async {
    let action = UpdateAction(answer: FakeUpdater().answer)
    let page = "https://drive.google.com/file/d/1MaRkStAnDiN0PaGe0iD00000000/view"

    let kept = await action.applied(
      to: Facts(versions: Versions(app: "2.14.0"), logoURL: page))
    XCTAssertEqual(kept.logoURL, page, "the address the poller took is the one that stays")

    let none = await action.applied(to: Facts(versions: Versions(app: "2.14.0")))
    XCTAssertNil(none.logoURL, "and a house with no page invents none")
  }

  // MARK: the panel's own reading

  private static func facts(installed: String, latest: String) -> Facts {
    Facts(
      account: "someone@example.com", auth: .ok,
      versions: Versions(app: "2.14.0", gdoc: installed, latestStable: latest),
      gdocPath: "/Users/someone/.local/bin/gdoc")
  }

  private static func gdocHeader(_ facts: Facts) -> HeaderModel {
    PanelModel.build(
      events: [], facts: facts, now: Date(timeIntervalSince1970: 1_792_000_000),
      calendar: Calendar(identifier: .gregorian), alive: { _ in false }
    ).header
  }
}

/// The child, as a closure, with what it printed.
private actor FakeUpdater {
  private(set) var calls: [Call] = []
  private let gate: Gate?
  private let data: String
  private let ok: Bool
  private let error: String?
  private let warning: String?

  init(
    gate: Gate? = nil,
    data: String = #"{"installed":"v2.14.0","action":"updated","to":"v2.15.0","verified":true}"#,
    ok: Bool = true,
    error: String? = nil,
    warning: String? = nil
  ) {
    self.gate = gate
    self.data = data
    self.ok = ok
    self.error = error
    self.warning = warning
  }

  nonisolated var answer: @Sendable (Call) async throws -> Outcome {
    { [self] call in try await said(call) }
  }

  private func said(_ call: Call) async throws -> Outcome {
    calls.append(call)
    await gate?.enter()
    return Outcome(
      call: call,
      envelope: Envelope(
        ok: ok, data: Data(data.utf8), error: error, warnings: [warning].compactMap { $0 },
        version: "v2.15.0"),
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
