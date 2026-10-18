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

  /// The parts add up to the cap. `.frame(maxHeight:)` with a clip under it
  /// reads 600 whatever was laid out inside, so a body one gap too tall is
  /// 600 on the probe and a bottom edge, corner and fade cut off the screen.
  func testTheHeaderAndTheBodyAddUpToTheCap() throws {
    for id in ["e1", "e3"] {
      let scene = try XCTUnwrap(Scenes.all.first { $0.id == id })
      let drawn = try probeList(try scene.view(), scheme: .dark)
      let header = try XCTUnwrap(drawn.first { $0.name == "header" }).size.height
      let body = try XCTUnwrap(
        drawn.filter { $0.name == "scrollbox" }.map { $0.size.height }.max(),
        "\(id) drew no scroll box")
      XCTAssertEqual(
        header + body + 2 * Grid.unit, Grid.panelMaxHeight,
        "\(id) laid out \(header + body + 2 * Grid.unit) pt inside the 600 pt frame")
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

  /// Scene e1's list: six names that are still six names once the middle is
  /// cut out of them.
  ///
  /// A row draws the head with an ellipsis after it and the tail whole, so
  /// two titles that differ only in the middle are one string on the screen
  /// and the list reads as the same document six times. The fixture's six
  /// differ at the head and at the tail, and this counts the strings the
  /// rows draw rather than the titles they came from.
  func testTheSixWorkingNotesOfE1DrawSixDifferentRows() throws {
    let scene = try XCTUnwrap(Scenes.all.first { $0.id == "e1" })
    let rows = try XCTUnwrap(scene.model().documents).rows
    let notes = rows.filter { $0.title.contains("note") }
    XCTAssertEqual(notes.count, 6, "e1 is six working notes under the day's own documents")

    let drawn = notes.map { drawnTitle($0.title) }
    XCTAssertEqual(
      Set(drawn).count, 6,
      "the rows draw \(Set(drawn).count) different strings: \(drawn)")
  }

  /// Scene r7's one row: the hold the person approved in the chat and the
  /// reply it approved are one thing that happened.
  func testTheApprovedHoldOfR7IsOneRow() throws {
    let scene = try XCTUnwrap(Scenes.all.first { $0.id == "r7" })
    let rows = try XCTUnwrap(scene.model().today).rows
    let approved = rows.filter { $0.sub?.contains("approved by you in chat") == true }
    XCTAssertEqual(approved.count, 1, "one row and not a Hold closed beside it")
    XCTAssertEqual(approved.first?.title, "Reply · Altery Group Channel Management Policy")
    XCTAssertEqual(approved.first?.value, "16:41")
    XCTAssertEqual(approved.first?.sub, "approved by you in chat · verified")
    XCTAssertFalse(
      rows.contains { $0.title.hasPrefix("Hold closed") }, "the write carries the ending")
  }

  /// Scene e2's DOCUMENTS header: four rows from three days, so the header
  /// is the stretch and the whole count. The old rule drew "1 today" over
  /// those four rows, which counted one thing and listed another.
  func testTheHeaderOfE2IsTheStretchAndTheWholeCount() throws {
    let scene = try XCTUnwrap(Scenes.all.first { $0.id == "e2" })
    let documents = try XCTUnwrap(scene.model().documents)
    XCTAssertEqual(documents.rows.count, 4)
    XCTAssertEqual(documents.count, "14 days · 4")
  }

  /// The green check of scene r3 is on the screen the caption names.
  ///
  /// The check lasts one second, and the scene was drawn exactly one second
  /// after the press: the render showed the plain link icon under a caption
  /// that said it was a second into its green check. A caption that names
  /// something the render does not draw is the one thing on the index page
  /// a reviewer cannot check against anything.
  func testTheCopyCheckOfR3IsOnTheScreen() throws {
    for id in ["r3", "r3l"] {
      let scene = try XCTUnwrap(Scenes.all.first { $0.id == id })
      let documents = try XCTUnwrap(try scene.model().documents, "\(id) drew no DOCUMENTS")
      let row = try XCTUnwrap(
        documents.rows.first { $0.doc == SceneDocuments.chapter5 }, "\(id) lost the document")
      let copy = try XCTUnwrap(
        row.actions.first { $0.kind == .copyLink }, "\(id) drew no Copy link")
      XCTAssertEqual(copy.state, .done, "\(id)'s check is not on the screen")
      XCTAssertEqual(copy.tip, Tips.copied)
      XCTAssertTrue(
        scene.caption.contains("green check") || id == "r3l",
        "the caption no longer names the check")
    }
  }

  /// One day, one update notice.
  ///
  /// The check found v2.15.0 at 17:05, which is scene r8. A release is not
  /// unpublished by 18:59 with the binary still at v2.14.0 and no update in
  /// the log, so the row says the same thing on every later still of that
  /// day: a fact that comes and goes reads as the panel losing it.
  func testTheUpdateNoticeStaysOnTheRestOfTheDay() throws {
    for id in ["r8", "r9"] {
      let scene = try XCTUnwrap(Scenes.all.first { $0.id == id })
      let gdoc = try XCTUnwrap(
        try scene.model().health.first { $0.kind == .gdoc }, "\(id) drew no gdoc row")
      XCTAssertTrue(
        gdoc.value.contains("v2.15.0"),
        "\(id)'s gdoc row says “\(gdoc.value)”, which forgets the release r8 found")
      XCTAssertTrue(
        gdoc.actions.contains { $0.kind == .update }, "\(id) drew no way to take it")
    }
  }

  /// Scene e9: the sign-in expired in the middle of a watch.
  ///
  /// The last poll came back with gdoc's `auth` code, so no poll follows it
  /// until somebody signs in: the watch is stopped and not paused, the card
  /// says what to do about it, and there is no Stop icon because there is no
  /// process to signal and never will be one.
  func testTheExpiredSignInOfE9StopsTheWatch() throws {
    let scene = try XCTUnwrap(Scenes.all.first { $0.id == "e9" })
    let model = try scene.model()

    guard case .live(_, let row, let stop, _)? = model.now.last else {
      return XCTFail("e9 draws its watch as a live row")
    }
    // The title carries the word and the sub-line carries the moment and
    // the cause, which is the stuck card's own rule: "Live" over a line
    // that said the watch had stopped was the row saying both.
    XCTAssertEqual(row.title, "Live stopped · Ch 5. Commercial Model: review draft v0.1")
    XCTAssertEqual(row.sub, "15:29 · sign-in expired")
    XCTAssertTrue(row.stopped, "the dot goes dim with the word")
    XCTAssertNil(stop, "nothing is polling, so there is nothing to stop")

    guard case .stuck(_, let card, _)? = model.now.first else {
      return XCTFail("e9 opens with the card `auth status` raised")
    }
    XCTAssertEqual(card.title, "Needs sign-in")
    XCTAssertEqual(card.reason, "1 live watch stopped. Sign in and start it again.")
    XCTAssertTrue(StuckCard.fits(card.reason), "it fits the card's two lines")

    XCTAssertTrue(
      scene.caption.contains("unmeasured"),
      "whether gdoc carries on from the same cursor after a sign-in is nobody's measurement yet")
    XCTAssertFalse(scene.caption.contains("paused"), "the caption says what the row says")
  }

  /// The same watch in the three sections that name it.
  ///
  /// A reader of one screen compares them, so a watch the sign-in stopped is
  /// stopped in DOCUMENTS and in TODAY as well, and no Stop icon is drawn
  /// anywhere on the screen: the model saying there is nothing to stop and
  /// the view drawing a live icon anyway is the one bug a probe catches and
  /// a field test does not.
  func testE9SaysTheWatchIsStoppedInEverySectionAndDrawsNoStop() throws {
    let scene = try XCTUnwrap(Scenes.all.first { $0.id == "e9" })
    let model = try scene.model()

    let documents = try XCTUnwrap(model.documents, "e9 drew no DOCUMENTS")
    let watched = try XCTUnwrap(
      documents.rows.first { $0.doc == SceneDocuments.chapter5 }, "the watched document is gone")
    XCTAssertEqual(
      watched.sub, "live stopped 15:29 · falcon/chapters/05.md",
      "DOCUMENTS said live while NOW said stopped")
    XCTAssertEqual(watched.tone, .dim, "and drew the live dot for a watch that is over")

    let today = try XCTUnwrap(model.today, "e9 drew no TODAY")
    let stops = today.rows.filter { $0.title.hasPrefix("Live stopped") }
    XCTAssertEqual(
      stops.map(\.value), ["15:29", "14:40"],
      "the watch the sign-in stopped is a row of the day like any other")

    for scheme in [ColorScheme.dark, .light] {
      let drawn = try probes(try scene.view(.whole), scheme: scheme)
      XCTAssertNil(drawn["live.stop"], "e9 draws a Stop icon in \(scheme)")
      XCTAssertNil(drawn["live.ring"], "and nothing is pending")
    }
  }

  /// A watch's counts, in the words the rest of the panel uses for those
  /// writes. TODAY says Reply and DOCUMENTS says replied, so "answered" was
  /// a third word for one thing, and the specification's own sentence names
  /// the two it allows: "since, replied and proposed counts".
  func testAWatchCountsItsWritesInTheWordsForThem() throws {
    let scene = try XCTUnwrap(Scenes.all.first { $0.id == "r4" })
    guard case .live(_, let row, _, _)? = try scene.model().now.last else {
      return XCTFail("r4 draws a live row")
    }
    XCTAssertEqual(row.sub, "since 13:10 · quiet 38 min · replied 1")
    XCTAssertFalse(row.stopped, "r4's watch is polling")

    // And the same words where the watch is over, on the row that says so.
    let stopped = try XCTUnwrap(Scenes.all.first { $0.id == "r9" })
    let today = try XCTUnwrap(try stopped.model().today, "r9 drew no TODAY")
    let last = try XCTUnwrap(
      today.rows.first { $0.title.hasPrefix("Live stopped") }, "r9 drew no stopped watch")
    XCTAssertEqual(last.sub, "5 h 30 min · replied 1")
  }

  /// A hold that stopped waiting is a moment and not a state.
  ///
  /// The row that raised it is written in the present tense, and scene r7
  /// draws it under the write the person approved 21 minutes later: two rows
  /// about one hold, one of them saying it is still waiting.
  func testAHoldThatClosedStopsSayingItIsWaiting() throws {
    let waiting = try XCTUnwrap(Scenes.all.first { $0.id == "r6" })
    let open = try XCTUnwrap(
      try waiting.model().today?.rows.first { $0.title.hasPrefix("Hold") },
      "r6 drew no hold row")
    XCTAssertEqual(open.sub, "Reply held by rule Dictated · waiting in Desktop chat")

    let approved = try XCTUnwrap(Scenes.all.first { $0.id == "r7" })
    let rows = try XCTUnwrap(try approved.model().today, "r7 drew no TODAY").rows
    let closed = try XCTUnwrap(
      rows.first { $0.title.hasPrefix("Hold") }, "r7 drew no hold row")
    XCTAssertEqual(closed.sub, "Reply held by rule Dictated")
    XCTAssertEqual(
      rows.first?.sub, "approved by you in chat · verified",
      "the row above it is the one that says how it ended")
  }

  /// One title as a row draws it: the head cut to what the column shows,
  /// the ellipsis where it was cut, and the tail whole. `MiddleTitle` draws
  /// exactly this, and a test that compared the titles themselves would
  /// pass on six rows that read the same.
  private func drawnTitle(_ title: String) -> String {
    let split = MiddleTitle.fitted(title)
    let shown = MiddleTitle.headCharacters(title)
    let head =
      shown < split.head.count
      ? String(split.head.prefix(shown)) + MiddleTitle.ellipsis : split.head
    return head + (split.tail ?? "")
  }

  /// Scene e8 is two moments on one panel, which the mockup drew that way
  /// to save a scene: the sign-in still waiting at the top, and the five
  /// seconds after one at the bottom. A caption that said neither would let
  /// a reviewer read the panel as one moment that cannot happen.
  func testTheE8CaptionSaysItStacksTwoMoments() throws {
    let scene = try XCTUnwrap(Scenes.all.first { $0.id == "e8" })
    XCTAssertTrue(
      scene.caption.contains("two moments"),
      "e8's caption does not say the render stacks two moments: \(scene.caption)")
    XCTAssertTrue(
      scene.caption.contains("as the mockup did"),
      "and does not say where the stacking comes from")
  }

  /// The two sheets are the one page in the package with no record behind
  /// it: a component after a component, with its words written by hand. A
  /// string invented there is a string Nail reads as the app's own, so the
  /// sheets say what the scenes say and the six the first draft invented
  /// are gone from the source itself.
  func testTheSheetsSayNothingNoSceneSays() throws {
    let invented = [
      "Watching", "\"Mark\"", "In the trash", "Proposed 2", "Nothing since",
      "14 days · 3 today",
    ]
    for file in Self.sheetFiles {
      let source = try testSource(file)
      for word in invented {
        XCTAssertFalse(
          source.contains(word), "\(file) still draws \(word), which no scene says")
      }
    }
  }

  /// The three files that draw a page with no record behind it. A fourth
  /// sheet is a fourth page nobody is holding to the scenes' words.
  static let sheetFiles = ["RowTests.swift", "CardTests.swift", "ControlTests.swift"]

  /// The six sheets are on the page.
  ///
  /// They are written by the files that measure their components, so a run
  /// of the whole suite leaves all six beside the scenes. The page did not
  /// name them at all, and a reviewer who has to find a PNG in a directory
  /// and guess what it is of is a reviewer reading something nobody wrote
  /// down.
  func testTheIndexCarriesTheComponentSheets() throws {
    XCTAssertEqual(Scenes.sheets.count, 6)
    let page = html([])
    for sheet in Scenes.sheets {
      XCTAssertTrue(page.contains("\(sheet.name)-dark.png"), "the page misses \(sheet.name)")
      XCTAssertTrue(page.contains("\(sheet.name)-light.png"), "in light as well")
      XCTAssertTrue(page.contains(escaped(sheet.about)), "and says nothing about it")
    }
  }

  /// A sheet puts no label over rows that are not that section's.
  ///
  /// A section label is a claim about what is under it: the count says how
  /// many rows the list has, and the name says what they are. The sheets
  /// borrow their rows from the scene that has the shape they want to show,
  /// so a label over them read NOW over a log row and a HEALTH row, and
  /// "14 days · 12" over two rows and a Show more. The labels are drawn as
  /// what they are, components with nothing under them.
  func testASheetPutsNoLabelOverRowsFromAnotherScene() throws {
    for file in Self.sheetFiles {
      let drawing = try sheetFunction(in: file)
      XCTAssertFalse(
        drawing.contains("SectionView("),
        "\(file)'s sheet draws a section, which claims the rows under it are its own")
    }
  }

  /// The body of one file's sheet-drawing function, which is the page with
  /// no record behind it. The rest of the file measures components, and a
  /// section measured is not a section drawn on a page Nail reads.
  private func sheetFunction(in file: String) throws -> String {
    let source = try testSource(file)
    let opening = try XCTUnwrap(
      source.range(of: "SheetRenders() throws {"), "\(file) draws no sheet")
    let rest = source[opening.upperBound...]
    let closing = try XCTUnwrap(rest.range(of: "\n  }"), "\(file)'s sheet never ends")
    return String(rest[..<closing.lowerBound])
  }

  /// One test file's own text, read from this file's path.
  private func testSource(_ name: String) throws -> String {
    let file =
      presencePackageRoot()
      .appendingPathComponent("Tests/PresenceUITests/\(name)")
    return try String(contentsOf: file, encoding: .utf8)
  }

  /// The index page opens with what nobody has agreed to yet.
  ///
  /// A reviewer who reads a render as the specification drawn rates a
  /// product choice as a bug, and a choice Nail has not made is not a bug.
  /// The eight are on the page, above every screen, and each names the rule
  /// or the decision it goes against.
  func testTheIndexOpensWithWhatIsStillYourCall() throws {
    XCTAssertEqual(Scenes.stillYourCall.count, 8, "the pass left eight choices to Nail")

    let page = html(["<section>a screen</section>"])
    let heading = try XCTUnwrap(page.range(of: "Still your call"), "the page does not open with it")
    let first = try XCTUnwrap(page.range(of: "<section>a screen</section>"))
    XCTAssertLessThan(
      heading.lowerBound, first.lowerBound, "the list is under the screens instead of over them")

    for line in Scenes.stillYourCall {
      XCTAssertTrue(page.contains(escaped(line)), "the page does not carry: \(line)")
    }

    // What each of them is about, so a line that is quietly dropped is a
    // failure rather than a shorter page.
    let subjects = [
      "45 percent", "decision 32", "decision 10", "trash", "decision 39", "14 days · N",
      "nameless", "faintest control",
    ]
    for subject in subjects {
      XCTAssertTrue(
        Scenes.stillYourCall.contains { $0.contains(subject) },
        "no line is about \(subject)")
    }
  }

  /// HEALTH is the one section whose length changes screen to screen, so
  /// the rule is on the page once and no reviewer reads a missing row as a
  /// bug. It is said once, because a rule in two places is two rules.
  func testTheHealthRowRuleIsSaidOnceOnTheIndex() throws {
    let page = html([])
    let said = page.components(separatedBy: escaped(Scenes.healthRowRule)).count - 1
    XCTAssertEqual(said, 1, "the HEALTH rule is on the page \(said) times")

    for part in ["Account first", "Quit last", "no Skills row"] {
      XCTAssertTrue(Scenes.healthRowRule.contains(part), "the rule does not say \(part)")
    }
  }

  /// Every word on either sheet is a word the panel draws.
  ///
  /// The two sheets are the pages with no record behind them: a component
  /// after a component, with their words handed in. The vocabulary is every
  /// string the scenes draw, plus the few that belong to no row at all, so
  /// a sentence somebody typed onto a sheet fails here rather than reaching
  /// Nail as the app's own.
  func testEverySheetWordIsAWordAPanelDraws() throws {
    let vocabulary = try panelVocabulary()
    XCTAssertGreaterThan(vocabulary.count, 100, "the scenes drew almost nothing")

    let words = try SheetPieces.words()
    XCTAssertGreaterThan(words.count, 30, "the sheets drew almost nothing")
    for word in words {
      XCTAssertTrue(
        vocabulary.contains(word),
        "a sheet draws “\(word)”, which no scene says and no row of the panel has")
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
          <figure><img src="\(scene.id)-whole-dark.png" width="320"><figcaption>dark, nothing hidden</figcaption></figure>
          <figure><img src="\(scene.id)-whole-light.png" width="320"><figcaption>light, nothing hidden</figcaption></figure>
        </div>
        <p class="about">drawn at \(scene.moment) · the scene is about the \(appearance) one</p>
      </section>
      """
  }

  /// One component sheet on the page, in both appearances.
  private func sheet(_ page: (name: String, about: String)) -> String {
    """
      <figure>
        <figcaption>\(escaped(page.name)) · \(escaped(page.about))</figcaption>
        <div class="pair">
          <img src="\(page.name)-dark.png" width="320">
          <img src="\(page.name)-light.png" width="320">
        </div>
      </figure>
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
      .calls{max-width:760px;margin:0 0 40px}
      .calls ul{margin:0 0 12px;padding-left:20px} .calls li{margin:0 0 6px}
    </style>
    <h1>gdoc presence: every screen, both appearances</h1>
    <section class="calls">
      <h2>Still your call</h2>
      <ul>
    \(Scenes.stillYourCall.map { "    <li>\(escaped($0))</li>" }.joined(separator: "\n"))
      </ul>
      <p>Each line is a choice the render makes against a rule or a
      decision. Nothing else on this page is open: the rest is the
      specification drawn.</p>
      <p>\(escaped(Scenes.healthRowRule))</p>
    </section>
    <p>Drawn from the records under Tests/PresenceUITests/Scenes at 2x. The
    first pair of each row is the panel as the person opens it, capped at
    600 pt with the rest under a scroll. The second pair is the same panel
    with nothing hidden, which is the one to review: a PNG cannot be
    scrolled.</p>
    <p>One record per scene, and each of them its own day: the nine r
    screens are one Saturday told nine times and agree with each other, and
    every e screen is a record of its own, written for the one thing it
    shows. Two counts from two e scenes are two days and not a sum.</p>
    \(entries.joined(separator: "\n"))
    <section>
      <h2>The component sheets</h2>
      <p>Six pages of pieces rather than panels. Every word on them comes
      out of a scene, so a sheet says what the app says and nothing else.</p>
    \(Scenes.sheets.map(sheet).joined(separator: "\n"))
    </section>
    """
  }

  private func escaped(_ text: String) -> String {
    text.replacingOccurrences(of: "&", with: "&amp;")
      .replacingOccurrences(of: "<", with: "&lt;")
      .replacingOccurrences(of: ">", with: "&gt;")
  }
}
