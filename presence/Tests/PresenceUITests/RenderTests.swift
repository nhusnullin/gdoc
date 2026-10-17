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
      let whole = try scene.view(.whole)
      let bar = try scene.bar()
      for scheme in [ColorScheme.dark, .light] {
        let suffix = scheme == .dark ? "dark" : "light"
        let name = "\(scene.id)-\(suffix).png"
        let data = try png(view, scheme: scheme)
        try data.write(to: shots.appendingPathComponent(name))
        XCTAssertGreaterThan(
          data.count, 2_000, "\(name) is too small to be a panel")

        // The same panel with nothing hidden. The reviewer reads this one,
        // because the rows past the cap are rows too.
        let wholeName = "\(scene.id)-whole-\(suffix).png"
        let wholeData = try png(whole, scheme: scheme)
        try wholeData.write(to: shots.appendingPathComponent(wholeName))
        XCTAssertGreaterThan(
          wholeData.count, 2_000, "\(wholeName) is too small to be a panel")

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

  /// The reviewer's copy hides nothing: on the screens the cap cuts, the
  /// whole render is taller, and HEALTH's last row is in it.
  func testTheWholeRenderCarriesWhatTheCapCuts() throws {
    for id in ["e1", "e3", "e9"] {
      let scene = try XCTUnwrap(Scenes.all.first { $0.id == id })
      let capped = try probes(try scene.view(), scheme: .dark)
      let whole = try probes(try scene.view(.whole), scheme: .dark)
      let cappedHeight = try XCTUnwrap(capped["panel"]).height
      let wholeHeight = try XCTUnwrap(whole["panel"]).height

      XCTAssertEqual(cappedHeight, Grid.panelMaxHeight, "\(id) is not at the cap")
      XCTAssertGreaterThan(
        wholeHeight, cappedHeight,
        "\(id) is cut at the cap, so the reviewer's copy has to be taller")
      XCTAssertNotNil(
        whole["health.quit"], "\(id)'s whole render drew no HEALTH at all")
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

  /// HEALTH is pinned first on each of the three screens it is the whole
  /// point of, and on no other. A countdown or a question that scrolled out
  /// of sight is one nobody answers.
  func testHealthIsPinnedWhileASignInIsInFlight() throws {
    let waiting = try XCTUnwrap(Scenes.all.first { $0.id == "e8" }).model()
    XCTAssertEqual(
      waiting.sections.first?.id, PanelSection.health([]).id,
      "the sign-in's countdown and its address live in HEALTH")
    XCTAssertFalse(
      waiting.sections.contains { if case .today = $0 { return true } else { return false } },
      "an empty TODAY under a pinned HEALTH is a label over one grey word")

    let firstRun = try XCTUnwrap(Scenes.all.first { $0.id == "e5" }).model()
    XCTAssertFalse(
      firstRun.sections.contains { if case .today = $0 { return true } else { return false } },
      "nothing has happened on a machine with no token, so TODAY says nothing")
  }

  /// One document is a document.
  func testACountAgreesWithItsNoun() throws {
    XCTAssertEqual(Sentences.count(1, "document"), "1 document")
    XCTAssertEqual(Sentences.count(0, "document"), "0 documents")
    XCTAssertEqual(Sentences.count(12, "action"), "12 actions")

    let midnight = try XCTUnwrap(Scenes.all.first { $0.id == "e2" }).model()
    let summaries = midnight.sections.compactMap { section -> String? in
      if case .yesterday(let model) = section { return model.summary }
      return nil
    }
    XCTAssertEqual(summaries.count, 1, "e2 is the screen Yesterday folds on")
    XCTAssertFalse(summaries[0].contains("1 documents"), summaries[0])
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

  /// The shadow stack belongs under the panel and under nothing inside it.
  ///
  /// SwiftUI gives every layer of a tree its own shadow unless the tree is
  /// flattened first, so a panel that asks for `--shadow` without compositing
  /// draws a grey halo under each row, each glyph and each line of text. The
  /// screens file puts no shadow inside `.panel`: the plate is white and the
  /// ink sits on white.
  ///
  /// The subject is a plate with one solid block on it rather than the panel
  /// itself, because a block's shadow is geometry and a word's is a font.
  func testTheShadowStackFallsUnderThePlateAndNotUnderWhatIsOnIt() throws {
    let plate = ZStack(alignment: .top) {
      Rectangle().fill(Color.white)
      Rectangle().fill(Color.black).frame(width: 60, height: 20).padding(.top, 10)
    }
    .frame(width: 100, height: 100)
    .shadows(Palette.light.panelShadows)

    // 50 pt down the plate: 20 pt under the block, inside the reach of a
    // 36 pt blur offset 12 pt down, and 30 pt clear of the plate's own edge.
    let under = try pixel(plate, scheme: .light, side: 100, x: 50, y: 50)
    XCTAssertEqual(
      under.redComponent, 1, accuracy: 0.004,
      "the block's shadow fell on the plate, so the stack was not composited")
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
          <figure><img src="\(scene.id)-whole-dark.png" width="320"><figcaption>dark, nothing hidden</figcaption></figure>
          <figure><img src="\(scene.id)-whole-light.png" width="320"><figcaption>light, nothing hidden</figcaption></figure>
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
      .pair{display:flex;gap:16px;align-items:flex-start;flex-wrap:wrap}
      figure{margin:0} figcaption{font-size:11px;color:#ebebf566;padding-top:4px}
      img{border-radius:12px;display:block}
    </style>
    <h1>gdoc presence: every screen, both appearances</h1>
    <p>Drawn from the records under Tests/PresenceUITests/Scenes at 2x. The
    first pair of each row is the panel as the person opens it, capped at
    600 pt with the rest under a scroll. The second pair is the same panel
    with nothing hidden, which is the one to review: a PNG cannot be
    scrolled.</p>
    \(entries.joined(separator: "\n"))
    """
  }

  private func escaped(_ text: String) -> String {
    text.replacingOccurrences(of: "&", with: "&amp;")
      .replacingOccurrences(of: "<", with: "&lt;")
      .replacingOccurrences(of: ">", with: "&gt;")
  }
}
