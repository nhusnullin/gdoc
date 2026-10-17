// The nine pieces the panel's sections are made of, and the words in them.
//
// Every number below is a literal read out of
// docs/design/2026-10-17-presence-screens-v3.html by hand: `.hold`, `.stuck`,
// `.row-live`, `.now-job`, `.empty`, `.bigmail`, `.urlline`, `.strip` and the
// `HROW` rows of HEALTH. The words are literals too, because a sentence a
// person reads in the menu bar is a decision and not a string somebody
// improved.
//
// Three of these tests are about words rather than pixels. The hold card
// never says Release, the strip says four of them, and the health rows keep
// one order on every screen: each is a thing the panel would still draw
// wrongly with every size right.

import AppKit
import SwiftUI
import XCTest

@testable import PresenceUI

@MainActor
final class CardTests: XCTestCase {
  /// The width a card is drawn at inside the panel: 320 less the panel's own
  /// 8 pt either side.
  private let cardWidth = Grid.panelWidth - 2 * Grid.unit

  // `.hold{padding:10px 12px;border-radius:8px}` on the amber tint, with the
  // two buttons the specification's decision table names. The screens file
  // labels the second one "Release without writing"; the decision table says
  // the panel never says Release and carries Dismiss instead, and a decision
  // wins over a label.
  func testTheHoldCardSaysOpenChatAndDismissAndNeverRelease() throws {
    XCTAssertEqual(HoldCard.words, [.openChat, .dismiss])
    XCTAssertEqual(HoldCard.words.map(\.rawValue), ["Open chat", "Dismiss"])
    for word in HoldCard.words {
      XCTAssertFalse(word.rawValue.contains("Release"))
    }
    // The word itself is nowhere in the file that would have drawn it.
    let source = try componentSource("HoldCard.swift")
    XCTAssertFalse(
      source.contains("Release without"),
      "HoldCard.swift carries the screens file's label instead of the decision's")

    // The plate is the amber tint and its half-pixel edge, and never the
    // amber block of the second version.
    for palette in [Palette.dark, Palette.light] {
      XCTAssertEqual(HoldCard.tint(in: palette), palette.holdTint)
      XCTAssertEqual(HoldCard.edge(in: palette), palette.holdTintEdge)
    }
    XCTAssertEqual(Grid.cardRadius, 8)
    XCTAssertEqual(Grid.cardInset, 10)
    XCTAssertEqual(HoldCard.quietTop, 6)
    XCTAssertEqual(HoldCard.actionTop, 10)
    XCTAssertEqual(HoldCard.actionGap, 6)
    XCTAssertEqual(HoldCard.bodyInset, 24, "the glyph's 16 pt and the row's 8 pt gap")

    let measured = try probes(HoldCard(model: hold()), scheme: .dark, width: cardWidth)
    XCTAssertNotNil(measured["hold.card"])
    XCTAssertNotNil(measured["hold.quiet"], "the card says where the text is waiting")
    XCTAssertEqual(measured["hold.open-chat"]?.height, 24, "`.btn{height:24px}`")
    XCTAssertEqual(measured["hold.dismiss"]?.height, 24)

    // Pressed once, the card asks, and the pair that answers is the inline
    // confirm the panel already has.
    let asking = try probes(HoldCard(model: hold(), asking: true), scheme: .dark, width: cardWidth)
    XCTAssertNotNil(asking["confirm.danger"])
    XCTAssertNotNil(asking["confirm.keep"])
    XCTAssertNil(asking["hold.open-chat"], "a card that is asking carries no other action")
  }

  // `.stuck .row-sub{-webkit-line-clamp:2}`, and the reason rule of scene r2:
  // the cause with its place first, then the state of the world, in 80
  // characters or fewer. Two lines at the card's own width is what 80
  // characters of ordinary words come to, which is measured here rather than
  // assumed.
  func testTheStuckCardReasonFitsTwoLinesUnderEightyCharacters() throws {
    XCTAssertEqual(StuckCard.reasonCeiling, 80)
    XCTAssertEqual(StuckCard.reasonLines, 2)
    XCTAssertEqual(
      StuckCard.reasonColumn, 256,
      "320 less the panel's 8 either side, the card's 12 either side, the glyph and its gap")

    // The two reasons the screens file writes, and one at the ceiling.
    let reasons = [
      "SVG picture at line 41. Nothing was created; the note is unchanged.",
      "Live watch on Chapter 5 is paused. Sign in to resume; the cursor is kept.",
      "The second tab has no heading. Nothing was written; the document is unchanged.",
    ]
    for reason in reasons {
      XCTAssertTrue(StuckCard.fits(reason), "\(reason.count) characters is past the ceiling")
      XCTAssertLessThanOrEqual(
        lines(reason, role: Typography.caption, width: StuckCard.reasonColumn),
        StuckCard.reasonLines, "“\(reason)” does not fit two lines")
    }
    XCTAssertFalse(
      StuckCard.fits(String(repeating: "x", count: 81)), "81 characters is one too many")

    let measured = try probes(
      StuckCard(model: stuck(), actions: [note(), branch()], onDismiss: {}),
      scheme: .dark, width: cardWidth)
    XCTAssertNotNil(measured["stuck.card"])
    XCTAssertEqual(
      measured["stuck.reason"]?.width, StuckCard.reasonColumn,
      "the reason spans the card past the glyph, which is `grid-column:2/4`")
    XCTAssertLessThanOrEqual(
      measured["stuck.reason"]?.height ?? 0, 2 * Typography.caption.lineHeight + 1,
      "a longer reason is clamped rather than drawn")
  }

  // `.stuck .acts` holds Open md, Continue with Claude and the bare cross,
  // in that order: the one that closes the card is the last, so a pointer
  // moving right never crosses it on the way to something else. Scene e9's
  // card has one action and no cross at all, because a sign-in that expired
  // is not something to dismiss.
  func testTheStuckCardDismissIsTheLastIcon() throws {
    XCTAssertEqual(StuckCard.dismissTip, "Dismiss")
    XCTAssertEqual(
      StuckCard.icons([note(), branch()], dismissable: true), [.md, .branch, .xmark])
    XCTAssertEqual(StuckCard.icons([note(), branch()], dismissable: false), [.md, .branch])
    XCTAssertEqual(
      StuckCard.icons([IconAction(.signin, tip: "Sign in with Google")], dismissable: false),
      [.signin], "scene e9: one action, and nothing to dismiss")

    var dismissed = 0
    let card = StuckCard(
      model: stuck(), actions: [note(), branch()], onDismiss: { dismissed += 1 })
    XCTAssertEqual(card.actions.map(\.icon), [.md, .branch, .xmark])
    card.actions.last?.perform()
    XCTAssertEqual(dismissed, 1, "the last icon is the one that closes the card")
  }

  // `LIVE()` in the screens file: one action, Stop watching. While the
  // SIGINT is in flight the icon becomes the small ring, because a stop that
  // can be pressed twice is a stop nobody trusts; the end arriving takes the
  // row away.
  func testTheLiveRowHasOneStopIconThatBecomesARingWhilePending() throws {
    XCTAssertEqual(LiveRow.stopTip, "Stop watching")
    XCTAssertEqual(LiveRow.icons(pending: false), [.stop])
    XCTAssertEqual(LiveRow.icons(pending: true), [], "the icon is gone while it is pending")
    XCTAssertFalse(LiveRow.showsRing(pending: false))
    XCTAssertTrue(LiveRow.showsRing(pending: true))

    let row = LiveRow(model: live())
    let measured = try probes(row, scheme: .dark, width: cardWidth)
    XCTAssertEqual(measured["row.two"]?.height, 44)
    XCTAssertEqual(measured["live.stop"]?.height, Grid.actionSize)
    XCTAssertNil(measured["live.ring"])

    let pending = try probes(
      LiveRow(model: live(), pending: true), scheme: .dark, width: cardWidth)
    XCTAssertNil(pending["live.stop"])
    XCTAssertEqual(pending["live.ring"], CGSize(width: Grid.glyphSize, height: Grid.glyphSize))

    // Off with its own words, which is the check in Task 18 refusing: the
    // row still draws one icon and never two.
    let off = LiveRow(model: live(), stop: .off("This watch is not one gdoc presence can stop"))
    XCTAssertEqual(off.state, .off("This watch is not one gdoc presence can stop"))
    XCTAssertEqual(LiveRow.icons(pending: false).count, 1)
  }

  // `.now-job` is the large ring, the title and one quiet line. The step
  // rows under it are the record's `step` events, which M24b adds: with none
  // of them the card is the ring and the two lines, and never four grey rows
  // the app invented.
  func testTheJobCardShowsStepRowsOnlyWhenStepsAreKnown() throws {
    let blind = JobCardModel(title: "Publishing", sub: nil, progress: nil, steps: [])
    XCTAssertFalse(JobCard.showsSteps(blind))
    let measured = try probes(JobCard(model: blind), scheme: .dark, width: cardWidth)
    XCTAssertNotNil(measured["job.card"])
    XCTAssertEqual(measured["job.ring"], CGSize(width: Grid.ringLarge, height: Grid.ringLarge))
    XCTAssertTrue(
      measured.keys.filter { $0.hasPrefix("job.step.") }.isEmpty,
      "nothing is known about the steps, so the card draws none")

    // Scene r1: four steps, two done, one running with its fraction, one
    // that has not happened.
    let known = job()
    XCTAssertTrue(JobCard.showsSteps(known))
    let drawn = try probes(JobCard(model: known), scheme: .dark, width: cardWidth)
    XCTAssertEqual(drawn.keys.filter { $0.hasPrefix("job.step.") }.count, 4)
    XCTAssertEqual(drawn["job.step.0"]?.height, Grid.compactRowHeight)
    XCTAssertEqual(JobCard.ringProgress(known), 0.5, "the card's own fraction, not a step's")
    XCTAssertNil(JobCard.ringProgress(blind), "an indeterminate ring, which turns")
  }

  // `.empty` is the first open of a machine that never signed in: the mark
  // at 28 pt and one sentence that names the row to press. Nothing else in
  // the panel is centred.
  func testTheEmptyStateIsTheFirstRunText() throws {
    XCTAssertEqual(
      EmptyState.firstRun,
      "Not signed in. Sign in with the Google account that will post replies: "
        + "the Account row below.")
    XCTAssertEqual(Typography.markLarge.size, 28, "`.empty .mark{font-size:28px}`")
    XCTAssertEqual(EmptyState.topPadding, 28)
    XCTAssertEqual(EmptyState.bottomPadding, 20)
    XCTAssertEqual(EmptyState.markGap, 8)

    for scheme in [ColorScheme.dark, .light] {
      let measured = try probes(
        EmptyState(mark: "⁘", words: EmptyState.firstRun), scheme: scheme, width: cardWidth)
      XCTAssertNotNil(measured["empty"])
      XCTAssertNotNil(measured["empty.mark"])
      XCTAssertGreaterThan(
        try inkedPixels(
          EmptyState(mark: "⁘", words: EmptyState.firstRun), scheme: scheme,
          side: Grid.panelWidth), 500, "the empty state drew nothing in \(scheme)")
    }
  }

  // Scene e8: the address at 15 pt for five seconds after a sign-in, with
  // "Not you?" beside it, because one glance catching a personal account is
  // cheaper than a reply carrying its name into a document.
  func testTheBannerLivesFiveSeconds() throws {
    XCTAssertEqual(Timing.checkFlash, 5)
    XCTAssertEqual(SignedInBanner.life, Timing.checkFlash)
    XCTAssertEqual(
      SignedInBanner.words, "Signed in just now. Not you? Sign out and sign in again.")
    XCTAssertEqual(SignedInBanner.signOutTip, "Sign out")
    XCTAssertEqual(Typography.bigMail.size, 15, "`.bigmail{font:500 15px/20px}`")

    let signedIn = Date(timeIntervalSince1970: 1_000)
    XCTAssertTrue(SignedInBanner.isShowing(signedInAt: signedIn, now: signedIn))
    XCTAssertTrue(
      SignedInBanner.isShowing(signedInAt: signedIn, now: signedIn.addingTimeInterval(4.99)))
    XCTAssertFalse(
      SignedInBanner.isShowing(signedInAt: signedIn, now: signedIn.addingTimeInterval(5)))
    XCTAssertFalse(
      SignedInBanner.isShowing(signedInAt: nil, now: signedIn), "nobody signed in")

    let measured = try probes(
      SignedInBanner(account: "nail@example.com"), scheme: .dark, width: cardWidth)
    XCTAssertGreaterThanOrEqual(
      measured["banner"]?.height ?? 0, SignedInBanner.minHeight,
      "`min-height:48px`: the address is a line taller than a title")
    XCTAssertLessThanOrEqual(
      measured["banner"]?.height ?? 0, SignedInBanner.minHeight + Typography.caption.lineHeight,
      "and the sentence under it wraps to two lines at most")
    XCTAssertEqual(measured["banner.sign-out"]?.height, Grid.actionSize)
  }

  // `.strip` is shown once, under the first document row, and says what the
  // four icons above it are. It is the whole of the onboarding: one person
  // does not need a tour, he needs to not guess once.
  func testTheStripLabelsAreDocNoteChatLink() throws {
    XCTAssertEqual(StripLabel.allCases.map(\.rawValue), ["Doc", "Note", "Chat", "Link"])
    XCTAssertEqual(StripLabel.allCases.count, 4, "one label per icon of the document row")
    XCTAssertEqual(IconStrip.cell, 24, "`.strip span{width:24px}`")
    XCTAssertEqual(IconStrip.gap, Grid.actionGap)
    XCTAssertEqual(IconStrip.leadingInset, Grid.quietInset, "`padding-left:44px`")
    XCTAssertEqual(Typography.strip.size, 10)

    let measured = try probes(IconStrip(), scheme: .dark, width: cardWidth)
    XCTAssertNotNil(measured["strip"])
    for label in StripLabel.allCases {
      XCTAssertEqual(
        measured["strip.\(label.rawValue)"]?.width, IconStrip.cell,
        "\(label.rawValue) sits over its own icon")
    }
  }

  // HEALTH is the same list on every screen, with the rows that have nothing
  // to say left out: account, network, gdoc, skills, Claude Desktop, the
  // thread mark, the switch, Quit. A section whose rows move is a section
  // nobody learns.
  func testHealthRowsKeepTheirOrderOnEveryScreen() throws {
    XCTAssertEqual(
      HealthRowKind.allCases,
      [.account, .network, .gdoc, .skills, .desktop, .mark, .launchAtLogin, .quit])
    XCTAssertEqual(
      HealthRowKind.allCases.map(\.label),
      [
        "Account", "Network", "gdoc", "Skills", "Claude Desktop", "Thread mark",
        "Launch at login", "Quit",
      ])
    XCTAssertEqual(HealthRows.quitShortcut, "⌘Q")

    // The five screens the blueprint draws HEALTH on, each shuffled and put
    // back in order.
    let screens: [(String, [HealthRowKind])] = [
      ("r8", [.account, .network, .gdoc, .skills, .desktop, .mark, .launchAtLogin, .quit]),
      ("e5", [.account, .gdoc, .skills, .desktop, .launchAtLogin, .quit]),
      ("e8", [.account, .gdoc, .launchAtLogin, .quit]),
      ("e9", [.account, .gdoc, .desktop, .launchAtLogin, .quit]),
      ("e10", [.account, .gdoc, .desktop, .launchAtLogin, .quit]),
    ]
    for (screen, order) in screens {
      let rows = order.map { HealthRow($0, value: $0.label) }
      XCTAssertEqual(
        HealthRows.ordered(rows.shuffled()).map(\.kind), order,
        "HEALTH on \(screen) is out of order")
    }

    // The quiet line under a row is that row's, so it never floats to the
    // foot of the section.
    let withQuiet = [
      HealthRow(.gdoc, value: "v2.12.0"),
      HealthRow(
        .network, value: "Google not reachable", glyph: .exclamation, tone: .bad,
        quiet: "the internet is up, the handshake with Google fails"),
      HealthRow(.account, value: "nail@example.com", glyph: .check, tone: .ok),
    ]
    let ordered = HealthRows.ordered(withQuiet)
    XCTAssertEqual(ordered.map(\.kind), [.account, .network, .gdoc])
    XCTAssertEqual(ordered[1].quiet, "the internet is up, the handshake with Google fails")

    let measured = try probes(HealthRows(rows: withQuiet), scheme: .dark, width: cardWidth)
    XCTAssertEqual(measured["health.account"]?.height, Grid.compactRowHeight)
    XCTAssertNotNil(measured["health.network.quiet"])
    let switched = try probes(
      HealthRows(rows: [HealthRow(.launchAtLogin, value: "", isOn: true)]), scheme: .dark,
      width: cardWidth)
    XCTAssertEqual(switched["switch"], CGSize(width: SwitchView.width, height: SwitchView.height))
  }

  // MARK: the render

  /// The sheet: the nine pieces in both appearances, so they are looked at
  /// once rather than trusted from a pixel count.
  func testTheCardSheetRenders() throws {
    let shots = try shotsDirectory()
    for scheme in [ColorScheme.dark, .light] {
      let palette = Palette.of(scheme)
      let sheet = VStack(alignment: .leading, spacing: Grid.unit) {
        HoldCard(model: hold())
        HoldCard(model: hold(), asking: true)
        StuckCard(model: stuck(), actions: [note(), branch()], onDismiss: {})
        LiveRow(model: live())
        LiveRow(model: live(), pending: true)
        JobCard(model: job())
        IconStrip()
        URLLine(url: "https://accounts.google.com/o/oauth2/auth?client_id=…")
        SignedInBanner(account: "nail@example.com")
        HealthRows(rows: healthScreen())
        EmptyState(mark: "⁘", words: EmptyState.firstRun)
      }
      .frame(width: cardWidth)
      .padding(Grid.unit)
      .background(palette.surface.color)

      let data = try png(sheet, scheme: scheme)
      let name = scheme == .dark ? "cards-dark.png" : "cards-light.png"
      try data.write(to: shots.appendingPathComponent(name))
      XCTAssertGreaterThan(data.count, 2_000, "\(name) is too small to be the sheet")
    }
  }

  // MARK: the fixtures
  //
  // Every value a piece draws is a plain value stated here, which is the
  // rule for this whole task: no process, no file and no machine behind any
  // of them.

  private func hold() -> HoldCardModel {
    HoldCardModel(
      title: "Waiting for you · 14 min",
      sub: "Reply to a thread · Group Channel Policy · rule Dictated",
      quiet: "Posts from nail@example.com with the ⁘ mark. The text is in the chat.",
      question: "Dismiss this hold?",
      consequence: "Nothing will be written. The hold stays in the chat.")
  }

  private func stuck() -> StuckCardModel {
    StuckCardModel(
      title: "Publish stopped · Chapter 4",
      reason: "SVG picture at line 41. Nothing was created; the note is unchanged.")
  }

  private func live() -> LiveRowModel {
    LiveRowModel(title: "Live · Chapter 5 storyline", sub: "since 13:10 · quiet 2 h")
  }

  private func job() -> JobCardModel {
    JobCardModel(
      title: "Publishing · Chapter 4", sub: "step 2 of 4 · started 09:03 · Falcon / Drafts",
      progress: 0.5,
      steps: [
        JobStep(title: "Built the docx", value: "7 pages · 3 pictures", state: .done),
        JobStep(title: "Uploaded", value: "1.2 MB", state: .done),
        JobStep(title: "Reading back", value: "2 of 3", state: .running(0.66)),
        JobStep(title: "Link into the note", value: nil, state: .waiting),
      ])
  }

  private func healthScreen() -> [HealthRow] {
    [
      HealthRow(
        .account, value: "nail@example.com", glyph: .check, tone: .ok,
        action: IconAction(.signout, tip: "Sign out")),
      HealthRow(
        .network, value: "Google not reachable", glyph: .exclamation, tone: .bad,
        quiet: "the internet is up, the handshake with Google fails"),
      HealthRow(
        .gdoc, value: "v2.12.0 · v2.13.0 available", glyph: .check, tone: .ok,
        action: IconAction(.down, tip: "Update to v2.13.0")),
      HealthRow(.skills, value: "v2.12.0 · altery plugin", glyph: .check, tone: .ok),
      HealthRow(.desktop, value: "extension connected", glyph: .check, tone: .ok),
      HealthRow(.mark, value: "⁘"),
      HealthRow(.launchAtLogin, value: "", isOn: true),
      HealthRow(.quit, value: HealthRows.quitShortcut),
    ]
  }

  private func note() -> IconAction {
    IconAction(.md, tip: "Open note at line 41")
  }

  private func branch() -> IconAction {
    IconAction(.branch, tip: "Continue with Claude")
  }

  // MARK: the measurements

  /// How many lines a sentence takes in the panel's own font at a width,
  /// which is the one thing a clamp cannot be read off the code for.
  private func lines(_ text: String, role: TypeRole, width: CGFloat) -> Int {
    let font = NSFont.systemFont(ofSize: role.size, weight: role.weight.appKit)
    let box = NSAttributedString(string: text, attributes: [.font: font]).boundingRect(
      with: CGSize(width: width, height: .greatestFiniteMagnitude),
      options: [.usesLineFragmentOrigin, .usesFontLeading])
    return Int((box.height / role.lineHeight).rounded(.up))
  }

  /// One component source by name, read from this file's own path.
  private func componentSource(_ name: String) throws -> String {
    let file =
      presencePackageRoot()
      .appendingPathComponent("Sources/PresenceUI/Components/\(name)")
    return try String(contentsOf: file, encoding: .utf8)
  }
}
