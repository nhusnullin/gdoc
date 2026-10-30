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
      XCTAssertEqual(panel.width, Grid.panelWidth, "\(scene.id) is not 360 wide")

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

  /// Nothing scrolls, and the panel is its parts added up.
  ///
  /// The cap, the middle box, the inner scrolls and Show more went on
  /// 2026-10-18: the panel is as tall as its content, whatever that comes
  /// to. Two things are read here. The sources carry no scroll view at all,
  /// which is the one claim a render cannot make: a box nobody put in a
  /// tree draws nothing in any state, and the words that built the old box
  /// are gone with it. And scene e3, the screen with every part on it, is
  /// the header plus its sections plus the panel's own padding, with
  /// nothing taken off anywhere and nothing spare.
  func testNothingScrollsAndThePanelIsItsPartsAddedUp() throws {
    for file in try uiSourceFiles() {
      let source = try String(contentsOf: file, encoding: .utf8)
      let name = file.lastPathComponent
      for gone in ["ScrollView", "ScrollBox", "ShowMore", "panelMaxHeight"] {
        XCTAssertFalse(source.contains(gone), "\(name) still names \(gone)")
      }
    }

    let scene = try XCTUnwrap(Scenes.all.first { $0.id == "e3" })
    for scheme in [ColorScheme.dark, .light] {
      let drawn = try probeList(try scene.view(), scheme: scheme)
      XCTAssertNil(
        drawn.first { $0.name.hasPrefix("scrollbox") }, "e3 drew a box in \(scheme)")

      let panel = try XCTUnwrap(drawn.first { $0.name == "panel" })
      let header = try XCTUnwrap(drawn.first { $0.name == "header" })
      let sections = drawn.filter { $0.name == "section" }
      XCTAssertEqual(sections.count, 3, "e3 is NOW, DOCUMENTS and SETTINGS")
      XCTAssertEqual(
        header.size.height + sections.map(\.size.height).reduce(0, +) + 2 * Grid.unit,
        panel.size.height,
        "e3's \(sections.count) sections under a \(header.size.height) pt header are not "
          + "the \(panel.size.height) pt panel")
      XCTAssertGreaterThan(
        panel.size.height, 600,
        "e3 is taller than the 600 pt the panel was capped at until 2026-10-18, which is "
          + "the price Nail chose over a scroll")
      XCTAssertNotNil(drawn.first { $0.name == "settings.quit" }, "Quit is drawn in \(scheme)")
    }
  }

  /// The question is asked on the header's own line, and the panel grows
  /// for it.
  ///
  /// A row states a minimum and not a height, so a header inside a capped
  /// panel stretched into whatever was spare: scene e10's header was 131 pt
  /// with "Sign out?" drawn in the middle of a hole, and DOCUMENTS and
  /// TODAY had 18 pt between them. The header takes its own height, every
  /// section takes its own, and the panel is the sum: a question adds the
  /// line it is asked on and moves nothing else.
  func testTheQuestionLeavesTheHeaderAtThirtySix() throws {
    let scene = try XCTUnwrap(Scenes.all.first { $0.id == "e10" })
    for scheme in [ColorScheme.dark, .light] {
      let drawn = try probeList(try scene.view(), scheme: scheme)

      let line = try XCTUnwrap(drawn.first { $0.name == "header.line" })
      XCTAssertEqual(line.size.height, Grid.headerHeight, "the question is one 36 pt line")

      let header = try XCTUnwrap(drawn.first { $0.name == "header" })
      let sections = drawn.filter { $0.name == "section" }
      let panel = try XCTUnwrap(drawn.first { $0.name == "panel" })
      XCTAssertEqual(
        header.size.height + sections.map(\.size.height).reduce(0, +) + 2 * Grid.unit,
        panel.size.height,
        "e10's header and its \(sections.count) sections are not the panel")

      let now = try XCTUnwrap(drawn.first { $0.name == "panel.now" }, "e10 draws NOW")
      let settings = try XCTUnwrap(drawn.first { $0.name == "panel.settings" })
      XCTAssertLessThan(header.origin.y, now.origin.y, "the header is above NOW")
      XCTAssertLessThan(now.origin.y, settings.origin.y, "SETTINGS is under it")
    }
  }

  /// Every row of every scene starts on the panel's one left edge, and the
  /// pin is drawn at the trailing edge with the other icons.
  ///
  /// A document row kept 8 pt in front of its glyph for the pin until the
  /// third look, so DOCUMENTS stood right of NOW, of SETTINGS and of its own
  /// section label on every screen in the set. The pin is the row's fifth
  /// icon now: the leading column is the glyph's own 16 pt in every row the
  /// panel draws, and the pinned scenes say so too.
  func testEveryRowInEverySceneStartsOnTheOneLeftEdge() throws {
    for scene in Scenes.all {
      let columns = try probeList(try scene.view(), scheme: .dark)
        .filter { $0.name == "row.leading" }
      for column in columns {
        XCTAssertEqual(
          column.size.width, Grid.glyphSize,
          "\(scene.id) draws a row that does not start where the others do")
      }
    }

    for id in ["e1", "e2"] {
      let pinned = try XCTUnwrap(Scenes.all.first { $0.id == id })
      let rows = try XCTUnwrap(try pinned.model().documents).rows
      XCTAssertTrue(rows[0].isPinned, "\(id)'s first row is the pinned one")
      XCTAssertEqual(
        rows[0].actions.last?.icon, .pinOn, "\(id) draws the filled tack on it")
      for row in rows.dropFirst() {
        XCTAssertEqual(
          row.actions.last?.icon, .pin, "and the outline tack on every row under it")
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

  func testSettingsIsPinnedFirstWhenSignedOut() throws {
    let first = try XCTUnwrap(Scenes.all.first { $0.id == "e5" }).model()
    XCTAssertEqual(
      first.sections.first?.id, PanelSection.settings([]).id,
      "nothing works without a sign-in, so SETTINGS is above everything")
    XCTAssertEqual(first.empty, EmptyState.firstRun, "the first run says where to press")

    let signedIn = try XCTUnwrap(Scenes.all.first { $0.id == "r4" }).model()
    XCTAssertEqual(
      signedIn.sections.last?.id, PanelSection.settings([]).id,
      "a working token puts SETTINGS last, where a person looks for it")
    XCTAssertNil(signedIn.empty)
  }

  /// SETTINGS is pinned first on each of the two screens it is the whole
  /// point of, and on no other. A countdown that scrolled out of sight is
  /// one nobody waits for.
  ///
  /// The sign-out question was the third of those screens until the second
  /// morning. It is asked in the header now, which is above every section
  /// on every screen, so the group goes back to the foot while the question
  /// is out: testTheSignOutQuestionPinsNothing.
  func testSettingsIsPinnedWhileASignInIsInFlight() throws {
    let waiting = try XCTUnwrap(Scenes.all.first { $0.id == "e8" }).model()
    let empty = DocumentsSectionModel(rows: [], empty: nil, showsStrip: false, forcedCard: nil)
    XCTAssertEqual(
      waiting.sections.map(\.id),
      [PanelSection.settings([]).id, PanelSection.documents(empty).id],
      "the group first, and then the list: nothing is running on that machine")

    let firstRun = try XCTUnwrap(Scenes.all.first { $0.id == "e5" }).model()
    XCTAssertEqual(
      firstRun.sections.first?.id, PanelSection.settings([]).id,
      "nothing works without a sign-in, so the group is above everything there too")
  }

  /// The one screen the second morning took off the pinned list: a token
  /// that works, and Sign out asking its question in the header. Nothing
  /// about the group is what that screen is for, so it stays at the foot
  /// and the documents keep the space above it.
  func testTheSignOutQuestionPinsNothing() throws {
    let asking = try XCTUnwrap(Scenes.all.first { $0.id == "e10" }).model()
    XCTAssertNotNil(asking.header.question, "e10 is the screen the question is out on")
    XCTAssertEqual(
      asking.sections.last?.id, PanelSection.settings([]).id,
      "the question is in the header, so SETTINGS is where it always is")
    XCTAssertEqual(
      asking.sections.first?.id, PanelSection.now([]).id,
      "and NOW keeps the top for a question that pins nothing")
  }

  /// The rows of SETTINGS never move, whichever screen they are on.
  func testSettingsKeepsItsOrderOnEveryScene() throws {
    let places = Dictionary(
      uniqueKeysWithValues: SettingsRowKind.allCases.enumerated().map { ($1, $0) })
    for scene in Scenes.all {
      let rows = try scene.model().settings
      XCTAssertFalse(rows.isEmpty, "\(scene.id) drew no SETTINGS at all")
      let order = rows.map { places[$0.kind] ?? -1 }
      XCTAssertEqual(order, order.sorted(), "\(scene.id) moved a SETTINGS row")
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

  /// Scene e1's list: six rows that are still six names once the middle is
  /// cut out of them.
  ///
  /// A row draws the head with an ellipsis after it and the tail whole, so
  /// two titles that differ only in the middle are one string on the screen
  /// and the list reads as the same document six times. The fixture's six
  /// differ at the head and at the tail, and this counts the strings the
  /// rows draw rather than the titles they came from.
  func testTheSixRowsOfE1DrawSixDifferentStrings() throws {
    let scene = try XCTUnwrap(Scenes.all.first { $0.id == "e1" })
    let rows = try XCTUnwrap(scene.model().documents).rows
    XCTAssertEqual(rows.count, 6, "e1 is the pinned note and the five gdoc touched last")
    XCTAssertEqual(
      Set(try scene.events().compactMap(Fold.subject)).count, 12,
      "and its caption says how many the record holds")

    let drawn = rows.map { drawnTitle($0.title) }
    XCTAssertEqual(
      Set(drawn).count, 6,
      "the rows draw \(Set(drawn).count) different strings: \(drawn)")
  }

  /// Scene e2: the pin is what holds a document past the fourteen days.
  ///
  /// The wallet note was last written to twenty days before that morning,
  /// so the horizon has let it go; the pin keeps its row, with the title
  /// and the moment of that last action and no state a pin invented. The
  /// four under it are the fortnight's own, newest first.
  func testThePinOfE2OutlivesTheFortnight() throws {
    let scene = try XCTUnwrap(Scenes.all.first { $0.id == "e2" })
    let documents = try XCTUnwrap(scene.model().documents)
    XCTAssertEqual(documents.rows.count, 5, "the pin and the four inside the fortnight")

    let pinned = try XCTUnwrap(documents.rows.first)
    XCTAssertEqual(pinned.doc, SceneDocuments.wallet)
    XCTAssertTrue(pinned.isPinned)
    XCTAssertEqual(pinned.title, "Wallet note: the partner shortlist")
    XCTAssertEqual(
      pinned.sub, "proposed 28 Sep · no note",
      "the last action it had, in the words every other row says one in")

    // The same scene with nothing pinned: the document is gone, and the
    // four that are left are the ones the fortnight carries.
    let bare = PanelModel.build(
      events: try scene.events(), facts: scene.facts, now: try scene.now(),
      calendar: scene.calendar, alive: { _ in false })
    XCTAssertEqual(
      bare.documents?.rows.map(\.doc), documents.rows.dropFirst().map(\.doc),
      "without the pin the list is the fortnight alone")
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
      let header = try scene.model().header
      XCTAssertEqual(header.version, "v2.14.0", "\(id) forgot which binary is installed")
      let update = try XCTUnwrap(header.update, "\(id) drew no way to take the release")
      XCTAssertEqual(
        update.tip, "Update to v2.15.0",
        "\(id)'s header forgets the release r8 found")
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

    guard case .live(_, let row, let stop, _, _)? = model.now.last else {
      return XCTFail("e9 draws its watch as a live row")
    }
    // The title carries the word and the sub-line carries the moment and
    // the cause, which is the stuck card's own rule: "Live" over a line
    // that said the watch had stopped was the row saying both.
    XCTAssertEqual(row.title, "Live stopped · Ch 5. Commercial Model: review draft v0.1")
    XCTAssertEqual(row.sub, "15:29 · sign-in expired")
    XCTAssertTrue(row.stopped, "the dot goes dim with the word")
    XCTAssertNil(stop, "nothing is polling, so there is nothing to stop")

    guard case .stuck(_, let card, _, _)? = model.now.first else {
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

  /// The same watch in both sections that name it.
  ///
  /// A reader of one screen compares them, so a watch the sign-in stopped is
  /// stopped in DOCUMENTS as well as in NOW, and no Stop icon is drawn
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
      watched.sub, "live stopped 15:29 · ~/hub/falcon/chapters/05.md",
      "DOCUMENTS said live while NOW said stopped")
    XCTAssertEqual(
      watched.glyph, .blank,
      "the row's words carry the state: the mark is a failed run and nothing else since "
        + "the fourth look")

    XCTAssertEqual(
      model.now.compactMap { item -> String? in
        if case .live(_, let row, _, _, _) = item { return row.title } else { return nil }
      },
      ["Live stopped · Ch 5. Commercial Model: review draft v0.1"],
      "and NOW says the same word over the same document")

    for scheme in [ColorScheme.dark, .light] {
      let drawn = try probes(try scene.view(), scheme: scheme)
      XCTAssertNil(drawn["live.stop"], "e9 draws a Stop icon in \(scheme)")
      XCTAssertNil(drawn["live.ring"], "and nothing is pending")
    }
  }

  /// A watch's counts, in the words the rest of the panel uses for those
  /// writes. DOCUMENTS says replied, so "answered" would be a second word
  /// for one thing, and the specification's own sentence names the two it
  /// allows: "since, replied and proposed counts".
  func testAWatchCountsItsWritesInTheWordsForThem() throws {
    let scene = try XCTUnwrap(Scenes.all.first { $0.id == "r4" })
    guard case .live(_, let row, _, _, _)? = try scene.model().now.last else {
      return XCTFail("r4 draws a live row")
    }
    XCTAssertEqual(row.sub, "since 13:10 · quiet 38 min · replied 1")
    XCTAssertFalse(row.stopped, "r4's watch is polling")
    XCTAssertEqual(Words.verb("reply")?.past, "replied", "the word the list says it in")
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
  /// so a label over them read NOW over a log row and a SETTINGS row, and
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
  /// or the decision it goes against. The second morning's own pass added
  /// two: where SETTINGS sits on the first-run screen, which the plan says
  /// two things about, and the light tooltip's plate. It also closed one,
  /// the dim mark, which is a decision now and is said as `markRule`. The
  /// third look closed the DOCUMENTS header count, which Nail settled on
  /// 2026-10-18 by asking for the label alone.
  func testTheIndexOpensWithWhatIsStillYourCall() throws {
    XCTAssertEqual(Scenes.stillYourCall.count, 7, "the pass left seven choices to Nail")

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
      "decision 32", "decision 10", "trash",
      "decision 39", "faintest control",
      "pinned above DOCUMENTS", "tooltip's plate in light",
    ]
    for subject in subjects {
      XCTAssertTrue(
        Scenes.stillYourCall.contains { $0.contains(subject) },
        "no line is about \(subject)")
    }

    // The mark is not one of them any more: Nail answered it on 2026-10-18.
    XCTAssertFalse(
      Scenes.stillYourCall.contains { $0.contains("as bright as the mark at work") },
      "the dim mark is a decision now and not a choice left open")

    // Neither is the DOCUMENTS header count, which the third look took off
    // the label: no render draws it, so no choice is left about its form.
    for settled in ["14 days", "N today"] {
      XCTAssertFalse(
        Scenes.stillYourCall.contains { $0.contains(settled) },
        "the DOCUMENTS header count is settled and not a choice left open: \(settled)")
    }

    // Nor is the nameless publish, which the fourth look answered: gdoc
    // writes the note's cover title on the end of a run that stopped in the
    // render, and r2's card reads it.
    XCTAssertFalse(
      Scenes.stillYourCall.contains { $0.contains("nameless") },
      "the publish that stopped in the render has a name now and no choice is left")
  }

  /// The bright idle mark is said once, as a decision and not a choice.
  ///
  /// It left "Still your call" with this milestone, and a reviewer who
  /// compares the bar of r9 with the mockup, which draws nothing at rest,
  /// needs the sentence that says the mockup was overruled. Said once,
  /// because a rule in two places is two rules.
  func testTheMarkRuleIsSaidOnceOnTheIndex() throws {
    let page = html([])
    let said = page.components(separatedBy: escaped(Scenes.markRule)).count - 1
    XCTAssertEqual(said, 1, "the mark rule is on the page \(said) times")

    for part in ["the mark at work", "2026-10-18", "four states apart and not five"] {
      XCTAssertTrue(Scenes.markRule.contains(part), "the rule does not say \(part)")
    }
  }

  /// SETTINGS is the one section whose length changes screen to screen, so
  /// the rule is on the page once and no reviewer reads a missing row as a
  /// bug. It is said once, because a rule in two places is two rules.
  func testTheSettingsRowRuleIsSaidOnceOnTheIndex() throws {
    let page = html([])
    let said = page.components(separatedBy: escaped(Scenes.settingsRowRule)).count - 1
    XCTAssertEqual(said, 1, "the SETTINGS rule is on the page \(said) times")

    for part in ["The group is SETTINGS", "ends in Quit", "not rows at all", "no Skills row"] {
      XCTAssertTrue(Scenes.settingsRowRule.contains(part), "the rule does not say \(part)")
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
          <figure><img src="\(scene.id)-dark.png" width="\(Int(Grid.panelWidth))"><figcaption>dark</figcaption></figure>
          <figure><img src="\(scene.id)-light.png" width="\(Int(Grid.panelWidth))"><figcaption>light</figcaption></figure>
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
          <img src="\(page.name)-dark.png" width="\(Int(Grid.panelWidth))">
          <img src="\(page.name)-light.png" width="\(Int(Grid.panelWidth))">
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
      <p>\(escaped(Scenes.settingsRowRule))</p>
      <p>\(escaped(Scenes.markRule))</p>
    </section>
    <p>Drawn from the records under Tests/PresenceUITests/Scenes at 2x. One
    image per scene and appearance, which is the whole panel: nothing in it
    scrolls and nothing is capped since 2026-10-18, so the panel the person
    opens and the panel the reviewer reads are the same drawing. A panel
    taller than the screen is cut at the screen's edge, which is the price
    of that decision and is in docs/v2/MEASURED.md.</p>
    <p>The mark at the right end of the DOCUMENTS label is a stand-in shape
    the test draws, of the wordmark's own proportions, and not the firm's
    logo: this repository carries no copy of that file. The app draws the
    file gdoc cached, and a machine with none draws the words alone, which
    is what scene e5 shows: the mark is cached after a sign-in, and the first
    run has never had one. A click on the mark opens the picture's own page
    in Google Drive, and the address behind it in these renders is a stand-in
    of the right shape, as the picture is.</p>
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
