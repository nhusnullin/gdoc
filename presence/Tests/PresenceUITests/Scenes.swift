// The scenes of the screens file, as records and facts.
//
// docs/design/2026-10-17-presence-screens-v3.html is nineteen stills of one
// day. Each of them is a moment, a record up to that moment, and a machine
// in some state, so each of them is a fixture here: the record under
// Scenes/<id>.jsonl, the moment and the facts below, and nothing invented
// inside a view.
//
// The titles and the addresses are the screens file's own invented
// stand-ins, of the same lengths, because the firm's documents stay out of
// this tree.
//
// Two things the screens draw are still not in a scene, because the record
// does not carry them and this milestone did not add them: a document in
// the trash, which e6 draws as an ordinary row, and the counts beside a
// publish's steps, which the specification put on the list of what is never
// written down. Everything else the screens show is a field: M24b's `made`,
// `step`, `stop`, `line`, `via: code` and `interrupted` reach these scenes.
// Scene e4 is not built at all: releasing a hold without writing is in the
// specification's "what is not built".

import Foundation
import PresenceCore
import SwiftUI
import XCTest

@testable import PresenceUI

/// One still: what it is called, what the reviewer is looking at, and
/// everything it takes to draw it.
struct Scene {
  let id: String
  /// The reviewer's heading, in English.
  let title: String
  /// What the scene is for, which is the caption under it on the index page.
  let caption: String
  /// The record, under Tests/PresenceUITests/Scenes. Two scenes have none:
  /// a first run has no record at all.
  let fixture: String?
  /// The moment the panel is drawn at, as the record writes one.
  let moment: String
  /// The pids the kernel still knows about. Everything else is gone.
  let alive: Set<Int32>
  let facts: Facts
  let ui: UIState

  /// The one appearance a scene is about, where it is about one. Every
  /// scene still renders in both; this is what the index page says.
  let appearance: ColorScheme?
}

/// The document ids and their titles, which are the screens file's
/// stand-ins at the same lengths.
enum SceneDocuments {
  static let chapter4 = "1Ch4BbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQq"
  static let chapter5 = "1Ch5AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  static let northwind = "1NwBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQq"
  static let partner = "1PbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRr"
  static let policy = "1GcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSs"
  static let concept = "1B2EeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTt"
  static let proposal = "1PrFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUu"
  /// The oldest of scene e1's twelve, eight days behind the rest of them.
  /// It is the one a person pinned, which is why it is drawn first.
  static let wallet = "1Ex06GgHhIiJjKkLlMmNnOoPpQqRrSsTt"

  /// The hub index: a document, and the note paired with it. Northwind and
  /// Partner bank are in no note, which is what greys their md icon.
  static let pairing: [String: String] = [
    chapter4: "falcon/chapters/04.md",
    chapter5: "falcon/chapters/05.md",
    policy: "05-rules/group-channel.md",
    concept: "falcon/b2b2c-concept.md",
    proposal: "falcon/proposal-mea-v1.md",
  ]
}

enum Scenes {
  /// What the render may have got wrong, and what nobody but Nail can
  /// settle. The index page opens with these, so a choice nobody has agreed
  /// to is never read as agreed.
  ///
  /// Each line is one choice the panel makes, the rule or the decision it
  /// goes against, and where it can be seen. They are product choices and
  /// not findings: a reviewer who rates one of them blocking is rating
  /// Nail's call, which is why they are on the page rather than in a task.
  static let stillYourCall: [String] = [
    "Launch at login is off until you turn it on, which is product decision 32. The "
      + "2026-10-18 decision withdrew the registration M24 made on the first run. A gdoc "
      + "command now opens the app when it is not running.",
    "The off switch in light is the faintest control the panel draws: a 12 percent black "
      + "pill on white, with the knob's shadow for its only edge. The mockup names that "
      + "fill. No render drew the switch off before 2026-10-18, so every light screen now "
      + "carries a control that is nearly not there.",
    "A publish is kept out of DOCUMENTS until a run names a document, which is why r1's "
      + "DOCUMENTS is empty. Product decision 10 puts a running job in both sections and "
      + "the mockup draws that row dimmed.",
    "A document in the trash keeps an ordinary row, which is e6's last one. No field of "
      + "the record says a document is gone, so nothing is struck through and nothing "
      + "says why.",
    "The header counts the sign-in's three minutes down, as product decision 39 asks. "
      + "Principle 5 allows facts and not promises, and the countdown is the one number in "
      + "the panel about something that has not happened.",
    "The DOCUMENTS header is \"N today\" where every row is from today and \"14 days · N\" "
      + "otherwise, counting the rows behind Show more. The mockup wrote three forms of it "
      + "on different screens. Since the second morning it counts a pinned row as well, "
      + "however old that document is, and counts a hidden one not at all: the number is "
      + "the list, and a pin and a hide are what the list is.",
    "SETTINGS is pinned above DOCUMENTS on the two screens the group is the whole of the "
      + "panel on, no token and a sign-in in flight, which is e5 and e8. It is the one "
      + "place Quit is not the last row in the panel, and the plan of the second morning "
      + "says both things: Task 3 keeps the pinned-first rule for those two screens and "
      + "Task 4 makes SETTINGS the foot of every panel. The code does what Task 3 says. "
      + "Dropping the rule costs nothing on the first-run screen, which is 409 pt, so "
      + "which of the two sentences wins is yours.",
    "The tooltip's plate in light is rgb(242,242,243) on the white panel, about 1.1 to 1, "
      + "with a one-pixel rgb(208,208,209) edge and its shadow for the rest. In dark it is "
      + "44 on 28. It is the second-faintest surface the light panel draws, after the off "
      + "switch, and it is the mockup's own `[data-tip]::after`.",
    "The publish that stopped in the render is nameless, in r2's card and in every row "
      + "after it, where the mockup draws \"Publish stopped · Chapter 4\". gdoc writes no "
      + "title on a start and none on the end of a publish that created nothing, so the "
      + "note's name is in no line of that run: naming it is a new fact in the record, "
      + "which docs/backlog/a-publish-that-stopped-in-the-render-has-no-name.md holds.",
  ]

  /// The mark at rest, said once on the index page as a decision rather
  /// than a choice. It was the first line of "Still your call" until
  /// 2026-10-18, when Nail answered it; the sentence stays on the page
  /// because the mockup draws nothing at rest and a reviewer comparing the
  /// two needs to know which of them is current.
  static let markRule =
    "The mark at rest is the mark at work. Since the decision of 2026-10-18 the idle bar "
    + "carries the same ink as a live watch, because Nail asked to see it: principle 1's "
    + "\"idle is nothing at all\" and the mockup's r9, which draws nothing, were both "
    + "overruled that day. What follows from it is in the bar of every screen: a live "
    + "watch and an idle machine are the same image, so the menu bar tells four states "
    + "apart and not five."

  /// How SETTINGS's rows are chosen, said once on the index page because it
  /// is the one section whose length changes from screen to screen, and a
  /// reviewer comparing two screens reads a missing row as a bug.
  static let settingsRowRule =
    "The group is SETTINGS, not HEALTH: what is left in it is two facts about the machine, "
    + "the two settings the panel has and Quit. The Hub row is one of the two: it says the folder the "
    + "notes are in, or offers to choose one, and pressing it opens the system's folder "
    + "panel. It ends in Quit on every screen, and the rows "
    + "above it keep their places. Network and Claude Desktop are drawn only where a fact "
    + "carries them: the record's own network stop, and no extension beside the binary. "
    + "The account, the version and the thread mark are not rows at all since the second "
    + "morning: the first two are the header, with Sign out and the update beside them, "
    + "and the mark is the header's own mark. There is no Skills row either, because "
    + "nothing gdoc prints names a version of the skills."

  /// The six pages of components, by the name they are written under and
  /// what each of them is. They are drawn by the files that measure those
  /// components, so a run of the whole suite writes all six beside the
  /// scenes; a reviewer who had to find them in the directory by hand is a
  /// reviewer who did not read them.
  static let sheets: [(name: String, about: String)] = [
    ("bar-states", "The menu bar: the five states and the check, monochrome, at the item's size."),
    ("glyphs", "The 16 px status glyph in every kind and tone, and the ring in its two sizes."),
    (
      "icons",
      "The action icons at 16 px, live and dimmed, and the Copy check; then the pin, "
        + "which is no action: it is drawn in the column it sits in, before a state "
        + "glyph, beside the empty slot the other rows of that list keep and beside a "
        + "row from a list with no pin at all."
    ),
    (
      "rows",
      "The header with the version, the update icon, the account and Sign out; then the "
        + "five row shapes, the chevron, the quiet line, and the two labels as pieces."
    ),
    (
      "cards",
      "The hold card asking and offering, the stuck card, the live row, the job card; "
        + "then the hint strip of four labels, the sign-in URL on its plate, the "
        + "signed-in account with its one sentence, and the SETTINGS rows, the Hub row "
        + "among them."
    ),
    (
      "controls",
      "The switch, the two text buttons, the one inline confirm, and two tooltips: an "
        + "icon's own words, and the reason an icon with nothing to open gives instead."
    ),
  ]

  static let account = "nail@example.com"
  static let hub = "/Users/nail/hub"
  /// The home the Hub row writes its tilde against, so every render says
  /// `~/hub` whoever's machine drew it.
  static let home = "/Users/nail"
  static let gdocPath = "/Users/nail/.local/bin/gdoc"
  static let signInURL =
    "https://accounts.google.com/o/oauth2/auth?client_id=…&redirect_uri=http://127.0.0.1:0"

  /// Half a second before r3's own moment, 2026-10-17T11:40:30Z, which is
  /// what its caption claims: Copy link half a second into its green check.
  /// A document with no moment is nothing pressed, so the moment is the
  /// whole of the state.
  ///
  /// The check lasts one second, and a second before the moment is the
  /// moment the check goes back: the render drew the link icon under a
  /// caption that said it was green.
  /// testTheCopyCheckOfR3IsOnTheScreen.
  static let copiedOnR3 = Date(timeIntervalSince1970: 1_792_237_229.5)

  /// A machine that is signed in and up to date, which most scenes are.
  static func machine(
    auth: AuthWord = .ok,
    account: String? = Scenes.account,
    previousAccount: String? = nil,
    latestStable: String? = nil,
    gdocFound: Bool = true,
    extensionState: ExtensionState = .written,
    network: NetworkFact? = nil,
    loginItem: LoginItemState = .off,
    signIn: SignInPhase? = nil,
    signedInAt: Date? = nil,
    hub: String? = Scenes.hub
  ) -> Facts {
    Facts(
      account: account,
      auth: auth,
      previousAccount: previousAccount,
      versions: Versions(
        app: "2.14.0", gdoc: gdocFound ? "v2.14.0" : nil, latestStable: latestStable),
      mark: "⁘",
      gdocPath: gdocFound ? gdocPath : nil,
      hub: hub,
      home: home,
      extensionState: extensionState,
      network: network,
      loginItem: loginItem,
      signIn: signIn,
      pairing: SceneDocuments.pairing,
      signedInAt: signedInAt)
  }

  /// Every scene, in the order the screens file tells the day.
  static let all: [Scene] = [
    Scene(
      id: "r1", title: "Publishing Chapter 4",
      caption:
        "A publish is running. The steps are the record's own four, the one written last "
        + "is the one running, and the ring is its place in the list. Nothing beside a "
        + "step says how much it is: gdoc writes no counts and the panel invents none. "
        + "DOCUMENTS is empty because a publish has no document until it creates one.",
      fixture: "r1", moment: "2026-10-17T09:04:00Z", alive: [7101],
      facts: machine(), ui: UIState(), appearance: nil),
    Scene(
      id: "r2", title: "The publish stopped",
      caption:
        "The write failed. The reason is the cause with its place first and the state of "
        + "the world second, two short sentences under eighty characters, out of the "
        + "record's own stop code and the line it names. The card carries the two icons "
        + "something can be done with: a publish that created nothing has no document to "
        + "open and no link to copy.",
      fixture: "r2", moment: "2026-10-17T09:05:30Z", alive: [],
      facts: machine(), ui: UIState(), appearance: nil),
    Scene(
      id: "r3", title: "A comment and a reply, both read back",
      caption:
        "Three writes landed and were verified: the restyle of 10:12 from the terminal, "
        + "the comment of 11:38 from Claude Code and the reply of 11:40 from the "
        + "terminal, which is why DOCUMENTS says three. The stuck card stays until the "
        + "publish is run again, and Copy link on the first row is half a second into "
        + "its green check.",
      fixture: "r3", moment: "2026-10-17T11:40:30Z", alive: [],
      facts: machine(),
      ui: UIState(copied: SceneDocuments.chapter5, copiedAt: copiedOnR3), appearance: nil),
    Scene(
      id: "r3l", title: "The same screen, and the light appearance is what it is about",
      caption:
        "The same record and the same moment as r3, drawn again because the light "
        + "appearance is the thing to read here: the same three surfaces, three text "
        + "levels and hairlines in their light values. Every scene on this page is drawn "
        + "in both appearances, so the four images in this row are r3's four.",
      fixture: "r3", moment: "2026-10-17T11:40:30Z", alive: [],
      facts: machine(),
      ui: UIState(copied: SceneDocuments.chapter5, copiedAt: copiedOnR3), appearance: .light),
    Scene(
      id: "r4", title: "A watch is live while the card is stuck",
      caption:
        "The watch on Chapter 5 has been going since 13:10. The live row carries one "
        + "action, Stop watching, which is live while a poll is in flight.",
      fixture: "r4", moment: "2026-10-17T14:02:00Z", alive: [7200],
      facts: machine(), ui: UIState(), appearance: nil),
    Scene(
      id: "r5", title: "A job and a watch at once",
      caption: "A finite job is over a watch that may sit there all day, in NOW as in the bar.",
      fixture: "r5", moment: "2026-10-17T14:30:00Z", alive: [7200, 7102],
      facts: machine(), ui: UIState(), appearance: nil),
    Scene(
      id: "r6", title: "A hold in chat over the watch",
      caption:
        "The hold is the one row that is a debt to the person, so it is first. The card "
        + "says Open chat and Dismiss, and never Release.",
      fixture: "r6", moment: "2026-10-17T16:34:00Z", alive: [7200],
      facts: machine(), ui: UIState(), appearance: nil),
    Scene(
      id: "r7", title: "The hold was approved in chat",
      caption: "The card is gone, the reply is a row in TODAY, and the watch carries on.",
      fixture: "r7", moment: "2026-10-17T16:41:30Z", alive: [7200],
      facts: machine(), ui: UIState(), appearance: nil),
    Scene(
      id: "r8", title: "Google did not answer",
      caption:
        "The last poll failed, and the record says what kind of trouble it was in one "
        + "word. The row says the moment and that word, and SETTINGS says the same thing "
        + "about the machine: the app opens no connection, so Google not answering is "
        + "only ever something gdoc wrote down.",
      fixture: "r8", moment: "2026-10-17T17:05:00Z", alive: [],
      facts: machine(
        latestStable: "v2.15.0",
        network: NetworkFact(reachable: false)),
      ui: UIState(), appearance: nil),
    Scene(
      id: "r9", title: "You stopped the watch",
      caption:
        "Nothing is running, so NOW is not drawn at all. The watch says you stopped it, "
        + "because the wait's end said so, and the publish of 14:31 is a document because "
        + "its end named the one it created. The day's log is behind Show more and "
        + "yesterday is one row.",
      fixture: "r9", moment: "2026-10-17T18:59:00Z", alive: [],
      facts: machine(latestStable: "v2.15.0"), ui: UIState(), appearance: nil),
    Scene(
      id: "e1", title: "More than five documents, one of them pinned",
      caption:
        "DOCUMENTS opened: the list scrolls inside its own box with the thumb and the "
        + "fade, and the panel stays under 600 pt. The middle of the panel scrolls as "
        + "well on this screen, and its own indicator is hidden while the inner box "
        + "carries one: two thumbs over one list is the panel saying which of them a "
        + "drag would move, which it cannot. The first row is the wallet note, pinned "
        + "two days ago: a pin puts a document first however old its last action is, "
        + "the pin is the tack before its state glyph, and every row of a list with a "
        + "pin in it keeps that slot, so the titles under it break where they would "
        + "anyway. Pin, Unpin and Hide from the list are the row's own menu, which no "
        + "still can show.",
      fixture: "e1", moment: "2026-10-17T18:10:00Z", alive: [],
      facts: machine(),
      ui: UIState(
        expanded: [.documents],
        pinned: [
          DocumentChoice(
            doc: SceneDocuments.wallet, at: Date(timeIntervalSince1970: 1_792_141_200),
            title: "Wallet note: the partner shortlist")
        ]),
      appearance: nil),
    Scene(
      id: "e2", title: "Midnight: yesterday folds into one row",
      caption:
        "The local day decides. Yesterday is one chevron row with its two counts, and "
        + "nothing older is in the log at all. DOCUMENTS keeps its own fourteen days, "
        + "which is why two rows there are older than yesterday. This is the next "
        + "morning, and the one screen whose Launch at login is on: the person turned "
        + "the switch on the evening before, which is the only way it is ever on.",
      fixture: "e2", moment: "2026-10-18T09:12:00Z", alive: [],
      facts: machine(loginItem: .on), ui: UIState(), appearance: nil),
    Scene(
      id: "e3", title: "Everything at once: the order of NOW",
      caption:
        "Holds first, then what is stuck, then running jobs, then watches, oldest first "
        + "inside each class. What is stuck here is an export: a failed publish names no "
        + "document, so any later publish clears it, and this screen has one running.",
      fixture: "e3", moment: "2026-10-17T17:20:00Z", alive: [7200, 7210, 7103],
      facts: machine(), ui: UIState(), appearance: nil),
    Scene(
      id: "e5", title: "The first run: no token at all",
      caption:
        "Nothing works without a sign-in, so the header says so and the one sentence "
        + "names the icon to press. Nothing else on this machine is set up either: the "
        + "Hub row offers to choose a folder, and this is the one screen that offer is "
        + "drawn on. No document has run here yet, so the grey note icon a missing hub "
        + "makes is on no screen at all. SETTINGS "
        + "is still pinned above everything, because on this machine it is the whole of "
        + "what the panel is for.",
      fixture: nil, moment: "2026-10-17T10:00:00Z", alive: [],
      facts: machine(
        auth: .signedOut, account: nil, gdocFound: true, extensionState: .notInstalled,
        hub: nil),
      ui: UIState(), appearance: nil),
    Scene(
      id: "e6", title: "Icons with nothing to open, and the first hint",
      caption:
        "An unpaired document greys its note icon and says why. The strip of four labels "
        + "is shown once under the first row. A document in the trash keeps its row: no "
        + "field says it is gone, so it is not struck through.",
      fixture: "e6", moment: "2026-10-17T11:00:00Z", alive: [],
      facts: machine(),
      ui: UIState(
        forcedTooltip: "\(SceneDocuments.northwind)|openNote", stripSeen: false),
      appearance: nil),
    Scene(
      id: "e8", title: "Waiting for Google, and just signed in",
      caption:
        "The sign-in is the child's own three minutes, with the URL as a line that can be "
        + "copied. Both of them are under the header now, where the account is. The "
        + "address is said once at its full size for five seconds. The render stacks two "
        + "moments on one panel, waiting for Google above and just signed in below, as "
        + "the mockup did: on a real machine one of them is always gone.",
      fixture: nil, moment: "2026-10-17T09:01:00Z", alive: [],
      facts: machine(
        signIn: SignInPhase(url: signInURL, secondsLeft: 163),
        signedInAt: Date(timeIntervalSince1970: 1_792_227_659)),
      ui: UIState(), appearance: nil),
    Scene(
      id: "e9", title: "The sign-in expired in the middle of a watch",
      caption:
        "The card comes from `auth status` and not from the record. The watch is stopped: "
        + "its last poll came back with gdoc's own auth code, so nothing is polling that "
        + "document and there is no Stop to press. Whether gdoc carries on from the same "
        + "cursor once somebody signs in is unmeasured, so the card asks for the watch to "
        + "be started again.",
      fixture: "e9", moment: "2026-10-17T15:30:00Z", alive: [],
      facts: machine(auth: .needsSignIn), ui: UIState(), appearance: nil),
    Scene(
      id: "e10", title: "Signing out, and changing account",
      caption:
        "The confirm names what will stop, in place of the header's own line, and waits "
        + "for the answer. A changed address is the app's own memory and never a line of "
        + "the log.",
      fixture: "e10", moment: "2026-10-17T18:00:00Z", alive: [7200, 7210],
      facts: machine(previousAccount: "nail@gmail.example"),
      ui: UIState(
        question: .signOut, questionAt: Date(timeIntervalSince1970: 1_792_259_999)),
      appearance: nil),
  ]
}

extension Scene {
  /// The record this scene is drawn from.
  func events() throws -> [Event] {
    guard let fixture else { return [] }
    let url = try XCTUnwrap(
      Bundle.module.url(forResource: fixture, withExtension: "jsonl", subdirectory: "Scenes"),
      "Scenes/\(fixture).jsonl is not in the test bundle")
    let records = Event.scan(try Data(contentsOf: url))
    XCTAssertEqual(records.skipped, 0, "\(fixture).jsonl holds a line the app cannot read")
    XCTAssertFalse(records.events.isEmpty, "\(fixture).jsonl is empty")
    return records.events
  }

  /// The moment the panel is drawn at.
  func now() throws -> Date {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return try XCTUnwrap(formatter.date(from: moment), "\(moment) is not a moment")
  }

  /// The one calendar every scene is folded and read in. UTC, so a still
  /// renders the same on any machine and a record's own clock is the
  /// panel's clock.
  var calendar: Calendar {
    var calendar = Calendar(identifier: .gregorian)
    calendar.timeZone = TimeZone(secondsFromGMT: 0) ?? .gmt
    calendar.locale = Locale(identifier: "en_US_POSIX")
    return calendar
  }

  /// This scene as the panel's model.
  @MainActor
  func model() throws -> PanelModel {
    let alive = self.alive
    return PanelModel.build(
      events: try events(), facts: facts, ui: ui, now: try now(), calendar: calendar,
      alive: { alive.contains($0) })
  }

  /// What the menu bar shows at this moment.
  ///
  /// Out of the same two folds the panel is built from, and the check out of
  /// the record's own last verified write: a still is a picture of a running
  /// app, so a line in its record is a line that arrived live, and the five
  /// seconds decide. That is why r3 and r7 draw the exclamation and the pause
  /// where the screens file drew a check: both stills are half a minute after
  /// the write they are about, and the check is five seconds long.
  @MainActor
  func bar() throws -> BarState {
    let alive = self.alive
    let at = try now()
    let events = try self.events()
    let fold = Now.fold(events, now: at, dismissed: ui.dismissed, alive: { alive.contains($0) })
    var check = BarCheck()
    check.saw(fold.lastVerified, live: true, at: fold.lastVerified?.ended ?? at)
    return BarImage.state(
      now: fold, sessions: Live.sessions(events, now: at, alive: { alive.contains($0) }),
      facts: facts,
      checkShowing: check.isShowing(now: at))
  }

  /// This scene as a view, in the mode the renderer needs.
  ///
  /// `.clipped` is the panel the person opens, capped at 600 pt with the
  /// rest under a scroll. `.whole` is the same panel with nothing hidden,
  /// which is what the reviewer's pass reads: a PNG cannot be scrolled, so
  /// a screen whose SETTINGS is past the cap is a screen nobody checked.
  @MainActor
  func view(_ mode: RenderMode = .clipped) throws -> some View {
    PanelView(model: try model()).rendering(mode)
  }
}
