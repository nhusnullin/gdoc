// The menu bar: one image, one state, no colour.
//
// Every rule under test here is the specification's own: the mark alone for a
// live watch, the mark with a 10 pt badge for the four states that have one,
// the dim mark when nothing is happening, and the priority that picks between
// them. The numbers are literals read off the `.status-item` rules of
// docs/design/2026-10-17-presence-screens-v3.html.
//
// The state is taken from the same folds the panel is built from, so the bar
// and the panel cannot disagree: a dismissed stuck card clears the
// exclamation because the fold it came out of no longer holds the write.
//
// The sheet of all six states is written as a PNG outside the repository, so
// a person looks at the shapes once rather than trusting a pixel count.

import AppKit
import PresenceCore
import SwiftUI
import XCTest

@testable import PresenceUI

@MainActor
final class BarTests: XCTestCase {
  /// `.status-item .mark` with `.g` or `.ring` beside it: one image per
  /// state, each a template so macOS tints it in the bar's own colour.
  func testOneTemplateImagePerState() throws {
    XCTAssertEqual(BarState.allCases.count, 6, "six states, and the bar is always in one of them")

    for state in BarState.allCases {
      let image = BarImage.image(for: state)
      XCTAssertTrue(image.isTemplate, "\(state) is not a template image, so the bar would tint it")
      XCTAssertEqual(image.size.height, BarImage.height, "\(state) is not \(BarImage.height) high")
      XCTAssertGreaterThan(
        try inkedPixels(BarGlyph(state: state), scheme: .dark, side: 32), 8,
        "\(state) drew nothing at all")
    }

    // The mark alone is live; idle is that same mark; the other four add the
    // badge and its gap and nothing else.
    let mark = BarImage.image(for: .live).size.width
    XCTAssertEqual(
      BarImage.image(for: .idle).size.width, mark, accuracy: 0.6,
      "idle is the live mark and not another shape")
    for state in [BarState.stuck, .hold, .job, .check] {
      XCTAssertEqual(
        BarImage.image(for: state).size.width, mark + BarImage.badgeGap + BarImage.badgeSize,
        accuracy: 0.6, "\(state) is the mark and one 10 pt badge")
      XCTAssertNotNil(state.badge, "\(state) draws a badge")
    }
    XCTAssertEqual(
      BarImage.ringFraction, RingView.indeterminateFraction,
      "the bar's ring and the panel's carry the same arc")
    XCTAssertNil(BarState.live.badge, "a live watch is the mark alone")
    XCTAssertNil(BarState.idle.badge, "idle is the mark alone")

    // "The mark at rest is as bright as the mark at work", which is the
    // 2026-10-18 decision answering the one choice principle 1 argued
    // against. Nothing in the file says what idle is drawn at any more.
    let rest = try inkWeight(BarGlyph(state: .idle), scheme: .dark, side: 32)
    let work = try inkWeight(BarGlyph(state: .live), scheme: .dark, side: 32)
    XCTAssertEqual(rest / work, 1, accuracy: 0.02, "the idle mark is not the live mark's own ink")

    try writeSheet()
  }

  /// Principle 1: no colour in the menu bar, ever. The image is drawn in one
  /// ink at a few opacities, so every pixel of it is the same grey.
  func testEveryBarPixelIsMonochrome() throws {
    for state in BarState.allCases {
      let image = BarImage.image(for: state)
      let data = try XCTUnwrap(image.tiffRepresentation, "\(state) has no bitmap")
      let rep = try XCTUnwrap(NSBitmapImageRep(data: data), "\(state) is not a bitmap")
      var inked = 0
      for x in 0..<rep.pixelsWide {
        for y in 0..<rep.pixelsHigh {
          let pixel = try XCTUnwrap(rep.colorAt(x: x, y: y)?.usingColorSpace(.deviceRGB))
          XCTAssertEqual(
            pixel.redComponent, pixel.greenComponent, accuracy: 0.004,
            "\(state) put colour at \(x),\(y)")
          XCTAssertEqual(
            pixel.greenComponent, pixel.blueComponent, accuracy: 0.004,
            "\(state) put colour at \(x),\(y)")
          if pixel.alphaComponent > 0.05 { inked += 1 }
        }
      }
      XCTAssertGreaterThan(inked, 8, "\(state) is an empty bitmap")
    }
  }

  /// The specification's order, which is the whole of what the bar decides: a
  /// write that stopped is first because it is the one thing that went wrong,
  /// then a hold waiting for the person, then a finite job over a watch that
  /// may sit there all day, then nothing.
  func testThePriorityIsStuckHoldJobLiveIdle() throws {
    let hold = try event(.held, at: "2026-10-17T11:00:00Z", pid: 11, run: "h", hold: "h_1")
    let stuck = try event(
      .end, at: "2026-10-17T11:01:00Z", pid: 12, run: "s", command: "reply", ok: false)
    let job = try event(
      .start, at: "2026-10-17T11:02:00Z", pid: 13, run: "j", command: "publish")
    let watch = try event(
      .start, at: "2026-10-17T11:03:00Z", pid: 14, run: "w", command: "comments", wait: true)

    XCTAssertEqual(try bar([hold, stuck, job, watch]), .stuck)
    XCTAssertEqual(try bar([hold, job, watch]), .hold)
    XCTAssertEqual(try bar([job, watch]), .job)
    XCTAssertEqual(try bar([watch]), .live)
    XCTAssertEqual(try bar([]), .idle)

    // The app's own sign-in is a job too: scene e8 draws the ring while the
    // child waits for Google, and the record carries no line for it.
    let signingIn = Scenes.machine(
      signIn: SignInPhase(url: Scenes.signInURL, secondsLeft: 163))
    XCTAssertEqual(try bar([], facts: signingIn), .job)
    XCTAssertEqual(try bar([hold], facts: signingIn), .hold, "a hold is still first")
    XCTAssertEqual(
      try bar([], facts: Scenes.machine(auth: .signedOut, signIn: nil)), .stuck,
      "a machine with no token is stuck whatever is in flight")

    // A start whose process is gone: a write is stuck, and anything else is
    // a line in TODAY and never an icon in the menu bar.
    XCTAssertEqual(
      try bar([job], alive: { _ in false }), .stuck,
      "a publish whose process is gone is a write nobody answered")
    let read = try event(
      .start, at: "2026-10-17T11:04:00Z", pid: 15, run: "r", command: "read")
    XCTAssertEqual(
      try bar([read], alive: { _ in false }), .idle,
      "a read whose process is gone changed nothing, so the bar says nothing")
  }

  /// A hold leaves the menu bar when its life has run out, the way a start
  /// leaves it when its pid is gone. The row stays in the panel, because what
  /// happened is still worth reading.
  func testAHoldThatRanOutLeavesTheMenuBar() throws {
    let waiting = try event(
      .held, at: "2026-10-17T11:50:00Z", pid: 11, run: "a", hold: "h_1",
      until: "2026-10-17T12:20:00Z")
    let ranOut = try event(
      .held, at: "2026-10-17T11:00:00Z", pid: 12, run: "b", hold: "h_2",
      until: "2026-10-17T11:30:00Z")

    XCTAssertEqual(try bar([waiting]), .hold)
    XCTAssertEqual(
      try bar([ranOut]), .idle,
      "a hold past its until is waiting for nobody, so the menu bar says nothing")
    XCTAssertEqual(
      try bar([waiting, ranOut]), .hold,
      "one hold that ran out does not take the badge off a hold that has not")
  }

  /// The three facts about the machine that are stuck without a line in the
  /// record: no token, a token that stopped working, and no binary to run.
  /// Each of them is the same red exclamation the HEALTH row draws.
  func testNotSignedInExpiredAndGdocNotFoundAreStuck() throws {
    for word in [AuthWord.signedOut, .needsSignIn, .missingScopes] {
      XCTAssertEqual(
        try bar([], facts: Scenes.machine(auth: word)), .stuck,
        "\(word.rawValue) is a row a person has to answer")
    }
    XCTAssertEqual(
      try bar([], facts: Scenes.machine(gdocFound: false)), .stuck,
      "an app that cannot find gdoc can do nothing at all")
    XCTAssertEqual(try bar([], facts: Scenes.machine()), .idle)

    // And they are over every state the record can be in, which is what
    // being first in the order means.
    let watch = try event(
      .start, at: "2026-10-17T11:03:00Z", pid: 14, run: "w", command: "comments", wait: true)
    XCTAssertEqual(try bar([watch], facts: Scenes.machine(auth: .needsSignIn)), .stuck)
  }

  /// The badge at login. Both questions are asked when the panel opens, so
  /// between the launch and the first click the app has no version of gdoc
  /// and the binary is there all the same: the path is what being found
  /// means, or the app would badge itself red on a healthy machine every
  /// morning and say gdoc was missing when the panel opened.
  func testTheBarIsNotStuckBeforeTheFirstAnswer() throws {
    let launching = Facts(
      auth: .ok, versions: Versions(app: "2.14.0"), gdocPath: Scenes.gdocPath)
    XCTAssertTrue(launching.gdocFound, "the binary is at a path, which is the location's fact")
    XCTAssertEqual(try bar([], facts: launching), .idle, "nothing is wrong yet")

    let row = PanelModel.gdoc(facts: launching)
    XCTAssertTrue(row.waiting, "the ring says the question is out")
    XCTAssertEqual(row.value, "", "and no version is claimed")
    XCTAssertNil(row.quiet)

    let missing = Facts(auth: .ok, versions: Versions(app: "2.14.0"))
    XCTAssertEqual(
      PanelModel.gdoc(facts: missing).value, "not found",
      "no path is the only not found there is")
  }

  /// A watch that cannot reach Google is still a watch: the decision table's
  /// own words, and the one fact the bar is handed and never reads.
  func testNetworkTroubleDoesNotChangeTheIcon() throws {
    let down = NetworkFact(reachable: false)
    let watch = try event(
      .start, at: "2026-10-17T11:03:00Z", pid: 14, run: "w", command: "comments", wait: true)

    XCTAssertEqual(try bar([watch], facts: Scenes.machine()), .live)
    XCTAssertEqual(
      try bar([watch], facts: Scenes.machine(network: down)), .live,
      "Google not answering is a HEALTH row and never an icon")
    XCTAssertEqual(try bar([], facts: Scenes.machine(network: down)), .idle)
  }

  /// The check is the one state that is over everything, and the one that is
  /// about a moment rather than a state: five seconds after a write read
  /// itself back, and only for a line that arrived while the app was running.
  /// Every end in the file at launch is a replay, and a replay flashes
  /// nothing: the app would show a check for a write from last Tuesday.
  func testTheCheckLastsFiveSecondsAfterALiveVerifiedEndAndNeverForAReplay() throws {
    XCTAssertEqual(BarCheck.lasts, 5)
    XCTAssertEqual(BarCheck.lasts, Timing.checkFlash, "the bar and the banner are one delay")

    let wrote = try event(
      .start, at: "2026-10-17T11:40:00Z", pid: 21, run: "v", command: "reply")
    let landed = try event(
      .end, at: "2026-10-17T11:40:02Z", pid: 21, run: "v", command: "reply", ok: true,
      verified: true)
    let fold = Now.fold([wrote, landed], now: try moment(), alive: { _ in false })
    let verified = try XCTUnwrap(fold.lastVerified, "the record holds a write that was read back")
    let at = try moment()

    var replayed = BarCheck()
    XCTAssertFalse(replayed.saw(verified, live: false, at: at), "a replayed line starts nothing")
    XCTAssertFalse(replayed.isShowing(now: at))

    var live = BarCheck()
    XCTAssertTrue(live.saw(verified, live: true, at: at), "the write arrived, so the check starts")
    XCTAssertTrue(live.isShowing(now: at))
    XCTAssertTrue(live.isShowing(now: at.addingTimeInterval(4.9)))
    XCTAssertFalse(live.isShowing(now: at.addingTimeInterval(5)))

    // The same end again is the same end: a redraw two seconds later does
    // not buy it another five.
    XCTAssertFalse(
      live.saw(verified, live: true, at: at.addingTimeInterval(4)),
      "the bar has shown this write already")
    XCTAssertFalse(live.isShowing(now: at.addingTimeInterval(5)))

    // Over every other state, which is principle 1's "over any state".
    let stuck = try event(
      .end, at: "2026-10-17T11:01:00Z", pid: 12, run: "s", command: "reply", ok: false)
    XCTAssertEqual(try bar([stuck], checkShowing: true), .check)
    XCTAssertEqual(try bar([], facts: Scenes.machine(auth: .signedOut), checkShowing: true), .check)
  }

  /// The tooltip is the one word that is always there, so it names the state
  /// and nothing else. No title, no address and no id: a menu bar is on
  /// screen in every screen-share.
  func testTheTooltipNamesTheStateAndNoDocument() {
    let words: [BarState: String] = [
      .stuck: "gdoc is stuck on something",
      .hold: "gdoc is holding a write for you",
      .job: "gdoc is running",
      .live: "gdoc is watching a document",
      .check: "gdoc wrote and read it back",
      .idle: "gdoc",
    ]
    for state in BarState.allCases {
      XCTAssertEqual(BarImage.tooltip(for: state), words[state], "\(state) says something else")
    }
    XCTAssertEqual(
      Set(words.values).count, BarState.allCases.count, "two states say the same thing")

    // Nothing about a document reaches the tooltip, which is why the
    // function is handed a state and never a glance.
    let names = [
      SceneDocuments.chapter5, SceneDocuments.northwind, "Chapter 5", "Northwind", Scenes.account,
    ]
    for state in BarState.allCases {
      let said = BarImage.tooltip(for: state)
      for name in names {
        XCTAssertFalse(said.contains(name), "\(state) named \(name)")
      }
    }
  }

  // MARK: the record, the folds and the renders

  /// One moment, which every fixture in this file is read at.
  private func moment(_ text: String = "2026-10-17T12:00:00Z") throws -> Date {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return try XCTUnwrap(formatter.date(from: text), "\(text) is not a moment")
  }

  /// The bar for a record, through the same two folds the panel is built
  /// from.
  private func bar(
    _ events: [Event], facts: Facts = Scenes.machine(),
    alive: (Int32) -> Bool = { _ in true }, checkShowing: Bool = false
  ) throws -> BarState {
    let at = try moment()
    return BarImage.state(
      now: Now.fold(events, now: at, alive: alive),
      sessions: Live.sessions(events, now: at, alive: alive),
      facts: facts,
      checkShowing: checkShowing)
  }

  /// One event, written as a line and decoded, so a test states the record
  /// and never the struct the record happens to decode into.
  private func event(
    _ kind: EventKind, at: String, pid: Int32, run: String, call: Int64? = 1,
    command: String? = nil, wait: Bool = false, ok: Bool? = nil, verified: Bool? = nil,
    hold: String? = nil, until: String? = nil
  ) throws -> Event {
    var fields: [String] = [
      "\"v\":1", "\"at\":\"\(at)\"", "\"pid\":\(pid)", "\"run\":\"\(run)\"",
      "\"event\":\"\(kind.rawValue)\"",
    ]
    if let call { fields.append("\"call\":\(call)") }
    if let command { fields.append("\"command\":\"\(command)\"") }
    if wait { fields.append("\"wait\":true") }
    if let ok { fields.append("\"ok\":\(ok)") }
    if let verified { fields.append("\"verified\":\(verified)") }
    if let hold { fields.append("\"hold\":\"\(hold)\"") }
    if let until { fields.append("\"until\":\"\(until)\"") }
    return try Event.decode("{" + fields.joined(separator: ",") + "}")
  }

  /// The sheet: the six states as the menu bar draws them, in both
  /// appearances, so the shapes are looked at once rather than counted.
  private func writeSheet() throws {
    let shots = try shotsDirectory()
    for scheme in [ColorScheme.dark, .light] {
      let sheet = VStack(alignment: .leading, spacing: 0) {
        ForEach(BarState.allCases, id: \.self) { state in
          BarPreview(state: state)
        }
      }
      let data = try png(sheet.environment(\.colorScheme, scheme), scheme: scheme)
      let name = scheme == .dark ? "bar-states-dark.png" : "bar-states-light.png"
      try data.write(to: shots.appendingPathComponent(name))
      XCTAssertGreaterThan(data.count, 500, "\(name) is too small to be the sheet")
    }
  }
}
