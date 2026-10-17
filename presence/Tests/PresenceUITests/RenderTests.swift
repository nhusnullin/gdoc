// Every screen of the screens file, drawn from a record.
//
// This is the one place in the package where whole panels are made. Each
// scene is a fixture, a moment and a set of facts, so what comes out is the
// panel the app would draw and never a mock: a row that reads wrong here is
// wrong in the app.
//
// The PNGs and the index page go outside the repository, under
// PRESENCE_SHOTS, because a render is something a person looks at once and
// a committed screenshot is a file nobody compares against the code that
// drew it. Task 23 is the reviewer's pass over that index page.
//
// The sizes are read back off the drawing through the metric probes rather
// than off the constants they came from, because a row that states 32 and
// draws 36 passes every test that reads the constant.

import AppKit
import PresenceCore
import SwiftUI
import XCTest

@testable import PresenceUI

@MainActor
final class RenderTests: XCTestCase {
  func testEveryScreenRendersInBothAppearances() throws {
    let shots = try shotsDirectory()
    var page: [String] = []

    for scene in Scenes.all {
      let view = try scene.view()
      let bar = try scene.bar()
      for scheme in [ColorScheme.dark, .light] {
        let suffix = scheme == .dark ? "dark" : "light"
        let name = "\(scene.id)-\(suffix).png"
        let data = try png(view, scheme: scheme)
        try data.write(to: shots.appendingPathComponent(name))
        XCTAssertGreaterThan(
          data.count, 2_000, "\(name) is too small to be a panel")

        // The menu bar the panel hangs from, in the same appearance. It is
        // one picture of the template image macOS tints, so a reviewer sees
        // the state the caption names.
        let barName = "\(scene.id)-bar-\(suffix).png"
        let barData = try png(BarPreview(state: bar), scheme: scheme)
        try barData.write(to: shots.appendingPathComponent(barName))
        XCTAssertGreaterThan(barData.count, 200, "\(barName) is too small to be a menu bar")
      }
      page.append(entry(for: scene, bar: bar))
    }

    let index = shots.appendingPathComponent("index.html")
    try html(page).write(to: index, atomically: true, encoding: .utf8)
    XCTAssertEqual(Scenes.all.count, 18, "the screens file holds eighteen scenes that are built")
  }

  /// Scene e4 is the hold released without writing, which the
  /// specification's "what is not built" rules out: the panel never says
  /// Release, so there is no screen to draw.
  func testReleaseWithoutWritingIsNotAScene() {
    XCTAssertFalse(
      Scenes.all.contains { $0.id == "e4" },
      "e4 is releasing a hold from the panel, which is not built")
    XCTAssertFalse(
      ButtonWord.allCases.contains { $0.rawValue.contains("Release") },
      "the panel never says Release")
  }

  func testTheMetricProbesHoldOnEveryScene() throws {
    for scene in Scenes.all {
      let sizes = try probes(try scene.view(), scheme: .dark, width: Grid.panelWidth)
      let panel = try XCTUnwrap(sizes["panel"], "\(scene.id) drew no panel")
      XCTAssertEqual(panel.width, Grid.panelWidth, "\(scene.id) is not 320 wide")

      if let header = sizes["header"] {
        XCTAssertGreaterThanOrEqual(
          header.height, Grid.headerHeight, "\(scene.id)'s header is under 36")
      }
      if let label = sizes["section-label"] {
        XCTAssertEqual(label.height, Grid.sectionLabelHeight, "\(scene.id)'s label is not 28")
      }
      if let row = sizes["row.doc"] {
        XCTAssertGreaterThanOrEqual(
          row.height, Grid.twoLineRowHeight, "\(scene.id)'s document row is under 44")
      }
      if let strip = sizes["strip.Doc"] {
        XCTAssertEqual(
          strip.width, Grid.actionSize, "\(scene.id)'s strip cell is not an icon wide")
      }
    }
  }

  /// The two busiest screens. The cap is the panel's own frame: a section a
  /// person opens scrolls inside itself and never grows the window.
  func testThePanelNeverPassesSixHundred() throws {
    for id in ["e1", "e3"] {
      let scene = try XCTUnwrap(Scenes.all.first { $0.id == id })
      for scheme in [ColorScheme.dark, .light] {
        let sizes = try probes(try scene.view(), scheme: scheme)
        let panel = try XCTUnwrap(sizes["panel"])
        XCTAssertLessThanOrEqual(
          panel.height, Grid.panelMaxHeight,
          "\(id) drew \(panel.height) pt in \(scheme), past the 600 the panel is capped at")
      }
    }
  }

  /// Scene r2's rule, which every stuck card obeys: the cause with its
  /// place first, then the state of the world, in two lines at this width.
  func testEveryReasonLineIsUnderEightyCharacters() throws {
    for scene in Scenes.all {
      for reason in try scene.model().reasons {
        XCTAssertTrue(
          StuckCard.fits(reason),
          "\(scene.id) wrote a reason of \(reason.count) characters: \(reason)")
      }
    }
  }

  func testNowIsHiddenWhenEmpty() throws {
    let quiet = try XCTUnwrap(Scenes.all.first { $0.id == "r9" }).model()
    XCTAssertTrue(quiet.now.isEmpty, "nothing is running in r9")
    XCTAssertFalse(
      quiet.sections.contains { if case .now = $0 { return true } else { return false } },
      "an empty NOW spends the panel's space on nothing, so it is not drawn")

    let busy = try XCTUnwrap(Scenes.all.first { $0.id == "e3" }).model()
    XCTAssertEqual(busy.now.count, 6, "e3 is two holds, one stuck, one job and two watches")
    XCTAssertEqual(
      busy.sections.first?.id, PanelSection.now([]).id, "NOW is first when it is there")
  }

  func testHealthIsPinnedFirstWhenSignedOut() throws {
    let first = try XCTUnwrap(Scenes.all.first { $0.id == "e5" }).model()
    XCTAssertEqual(
      first.sections.first?.id, PanelSection.health([]).id,
      "nothing works without a sign-in, so HEALTH is above everything")
    XCTAssertEqual(first.empty, EmptyState.firstRun, "the first run says where to press")

    let signedIn = try XCTUnwrap(Scenes.all.first { $0.id == "r4" }).model()
    XCTAssertEqual(
      signedIn.sections.last?.id, PanelSection.health([]).id,
      "a working token puts HEALTH last, where a person looks for it")
    XCTAssertNil(signedIn.empty)
  }

  /// The rows of HEALTH never move, whichever screen they are on.
  func testHealthKeepsItsOrderOnEveryScene() throws {
    let places = Dictionary(
      uniqueKeysWithValues: HealthRowKind.allCases.enumerated().map { ($1, $0) })
    for scene in Scenes.all {
      let rows = try scene.model().health
      XCTAssertFalse(rows.isEmpty, "\(scene.id) drew no HEALTH at all")
      let order = rows.map { places[$0.kind] ?? -1 }
      XCTAssertEqual(order, order.sorted(), "\(scene.id) moved a HEALTH row")
      XCTAssertEqual(rows.last?.kind, .quit, "\(scene.id) does not end in Quit")
    }
  }

  // MARK: the index page

  private func entry(for scene: Scene, bar: BarState) -> String {
    let appearance = scene.appearance == .light ? "light" : "dark"
    return """
      <section>
        <h2>\(scene.id) · \(escaped(scene.title))</h2>
        <p>\(escaped(scene.caption))</p>
        <div class="pair">
          <figure><img src="\(scene.id)-bar-dark.png" width="140"><figcaption>the bar: \(bar.rawValue)</figcaption></figure>
          <figure><img src="\(scene.id)-bar-light.png" width="140"><figcaption>light</figcaption></figure>
        </div>
        <div class="pair">
          <figure><img src="\(scene.id)-dark.png" width="320"><figcaption>dark</figcaption></figure>
          <figure><img src="\(scene.id)-light.png" width="320"><figcaption>light</figcaption></figure>
        </div>
        <p class="about">drawn at \(scene.moment) · the scene is about the \(appearance) one</p>
      </section>
      """
  }

  private func html(_ entries: [String]) -> String {
    """
    <!doctype html>
    <meta charset="utf-8">
    <title>gdoc presence: every screen</title>
    <style>
      body{margin:0;padding:32px;background:#3b4352;color:#f2f2f7;
        font:400 14px/1.5 -apple-system,system-ui,sans-serif}
      h1{font-size:20px} h2{font-size:15px;margin:0 0 4px}
      section{margin:0 0 40px;max-width:760px}
      p{margin:0 0 12px;color:#ebebf599} .about{font-size:12px;color:#ebebf566}
      .pair{display:flex;gap:16px;align-items:flex-start}
      figure{margin:0} figcaption{font-size:11px;color:#ebebf566;padding-top:4px}
      img{border-radius:12px;display:block}
    </style>
    <h1>gdoc presence: every screen, both appearances</h1>
    <p>Drawn from the records under Tests/PresenceUITests/Scenes at 2x.</p>
    \(entries.joined(separator: "\n"))
    """
  }

  private func escaped(_ text: String) -> String {
    text.replacingOccurrences(of: "&", with: "&amp;")
      .replacingOccurrences(of: "<", with: "&lt;")
      .replacingOccurrences(of: ">", with: "&gt;")
  }
}
