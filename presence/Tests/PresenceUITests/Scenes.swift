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

import AppKit
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

  /// A document and the note the finder answered for it, as the absolute
  /// path the opener is handed. Northwind and Partner bank are in no note,
  /// which is what greys their md icon.
  static let pairing: [String: String] = [
    chapter4: "/Users/nail/hub/falcon/chapters/04.md",
    chapter5: "/Users/nail/hub/falcon/chapters/05.md",
    policy: "/Users/nail/notes/05-rules/group-channel.md",
    concept: "/Users/nail/hub/falcon/b2b2c-concept.md",
    proposal: "/Users/nail/hub/falcon/proposal-mea-v1.md",
  ]
}

/// The firm's mark as a render may draw it, which is a stand-in and never
/// the file.
///
/// The decision of 2026-10-16 keeps the firm's logo out of this repository,
/// so a test draws a shape of the file's own proportions: a filled rounded
/// rectangle 275 by 100 pixels, which is the wordmark's 2.75 to 1, in black
/// on nothing, so the ink a view puts down is the tint and not the bytes.
/// Its ink fills its frame, which is what lets a measurement read the
/// frame off the drawing.
///
/// Nonisolated because `Scenes.all` is, and the measuring helpers in
/// Shots.swift are on the main actor and cannot be called from it.
enum SceneMark {
  /// The stand-in, as PNG bytes. Nothing from the firm and nothing from
  /// disk.
  static let standIn: Data = drawn(width: 275, height: 100)

  /// Two hundred bytes that are no picture at all, which is what the panel
  /// is shown to draw nothing for.
  static let notAPicture: Data = Data((0..<200).map { UInt8(($0 * 7 + 13) % 251) })

  private static func drawn(width: Int, height: Int) -> Data {
    guard
      let rep = NSBitmapImageRep(
        bitmapDataPlanes: nil, pixelsWide: width, pixelsHigh: height,
        bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false,
        colorSpaceName: .deviceRGB, bytesPerRow: width * 4, bitsPerPixel: 32),
      let context = NSGraphicsContext(bitmapImageRep: rep)
    else { return Data() }
    NSGraphicsContext.saveGraphicsState()
    NSGraphicsContext.current = context
    NSColor.black.setFill()
    NSBezierPath(
      roundedRect: NSRect(x: 0, y: 0, width: width, height: height),
      xRadius: 12, yRadius: 12
    ).fill()
    context.flushGraphics()
    NSGraphicsContext.restoreGraphicsState()
    return rep.representation(using: .png, properties: [:]) ?? Data()
  }
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
  ///
  /// A line leaves the list on the day Nail settles it. The third look took
  /// off the one about the DOCUMENTS header count, which asked which form
  /// of "N today" and "14 days · N" was right: the answer of 2026-10-18 is
  /// the label on its own, so no render draws a count and no choice is left.
  ///
  /// The fourth look took off the one about the nameless publish: the answer
  /// of 2026-10-26 is that gdoc keeps the note's cover title on a run that
  /// stopped in the render and writes it on the end, which r2's card now
  /// reads.
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
    "SETTINGS is pinned above DOCUMENTS on the two screens the group is the whole of the "
      + "panel on, no token and a sign-in in flight, which is e5 and e8. It is the one "
      + "place Quit is not the last row in the panel, and the plan of the second morning "
      + "says both things: Task 3 keeps the pinned-first rule for those two screens and "
      + "Task 4 makes SETTINGS the foot of every panel. The code does what Task 3 says. "
      + "Dropping the rule costs nothing on the first-run screen, which is 381 pt, so "
      + "which of the two sentences wins is yours.",
    "The tooltip's plate in light is rgb(242,242,243) on the white panel, about 1.1 to 1, "
      + "with a one-pixel rgb(208,208,209) edge and its shadow for the rest. In dark it is "
      + "44 on 28. It is the second-faintest surface the light panel draws, after the off "
      + "switch, and it is the mockup's own `[data-tip]::after`.",
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
    + "the one setting the panel has and Quit. The Hub row went with the third look: the "
    + "note is found through Spotlight, so there is no folder to choose and no row that "
    + "chooses one. It ends in Quit on every screen, and the rows "
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
      "The action icons at 16 px, live and dimmed, and the Copy check. The last two "
        + "are the pin, the fourth and last icon of a document row since the fifth look "
        + "took the Google Docs one away: the "
        + "outline a document carries while it is not pinned, and the filled tack it "
        + "carries while it is."
    ),
    (
      "rows",
      "The header with the version, the account and Sign out, drawn four times for the "
        + "four ways the update glyph is: a release to take, which is r8's own header, so "
        + "the line under it is that screen's press of Update, and the version is in the "
        + "glyph's tooltip; nothing newer known; the second after a press that "
        + "installed nothing; and the ring while the child is up. Then the four row "
        + "shapes, the quiet line, and the two labels as pieces."
    ),
    (
      "cards",
      "The hold card asking and offering, the stuck card, the live row, the job card; "
        + "then the hint strip of four labels, the sign-in URL on its plate, the "
        + "signed-in account with its one sentence, and the SETTINGS rows, which are "
        + "the machine's own facts, Launch at login and Quit."
    ),
    (
      "controls",
      "The switch, the two text buttons, the one inline confirm, and two tooltips: an "
        + "icon's own words, and the reason an icon with nothing to open gives instead."
    ),
  ]

  static let account = "nail@example.com"
  /// The home a note's path is written with a tilde against, so every
  /// render says `~/hub/...` whoever's machine drew it.
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
    pairing: [String: String] = SceneDocuments.pairing,
    updateChecked: Date? = nil,
    updating: Bool = false,
    updateRun: UpdateRun? = nil,
    extensionRun: ExtensionRun? = nil,
    logo: Data? = SceneMark.standIn
  ) -> Facts {
    Facts(
      account: account,
      auth: auth,
      previousAccount: previousAccount,
      versions: Versions(
        app: "2.14.0", gdoc: gdocFound ? "v2.14.0" : nil, latestStable: latestStable),
      mark: "⁘",
      logo: logo,
      gdocPath: gdocFound ? gdocPath : nil,
      home: home,
      extensionState: extensionState,
      network: network,
      loginItem: loginItem,
      signIn: signIn,
      pairing: pairing,
      signedInAt: signedInAt,
      updating: updating,
      updateChecked: updateChecked,
      updateRun: updateRun,
      extensionRun: extensionRun)
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
        + "record's own stop code and the line it names. The card names the note, "
        + "\"Publish stopped \u{00B7} Chapter 4\": since 2026-10-26 a publish that stopped in "
        + "the render writes the note's cover title on its end, so the card is no longer "
        + "nameless. It carries the two icons something can be done with: a publish that "
        + "created nothing has no document to open and no link to copy.",
      fixture: "r2", moment: "2026-10-17T09:05:30Z", alive: [],
      facts: machine(), ui: UIState(), appearance: nil),
    Scene(
      id: "r3", title: "A comment and a reply, both read back",
      caption:
        "Three writes landed and were verified: the restyle of 10:12, the comment of "
        + "11:38 and the reply of 11:40, which is the three documents DOCUMENTS lists. "
        + "A row says the word and the moment and not where the run came from: the "
        + "source is a sub-line of NOW's, which this screen has none of. The stuck card "
        + "stays until the publish is run again, and Copy link on the first row is half "
        + "a second into its green check.",
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
      caption:
        "The card is gone, the reply the person approved is what the policy's row now "
        + "says, and the watch carries on.",
      fixture: "r7", moment: "2026-10-17T16:41:30Z", alive: [7200],
      facts: machine(), ui: UIState(), appearance: nil),
    Scene(
      id: "r8", title: "Google did not answer",
      caption:
        "The last poll failed, and the record says what kind of trouble it was in one "
        + "word. The row says the moment and that word, and SETTINGS says the same thing "
        + "about the machine: the app opens no connection, so Google not answering is "
        + "only ever something gdoc wrote down. The check of 17:05 found v2.15.0, so the "
        + "update glyph is in the accent ink with a dot, and v2.15.0 is in its tooltip. The "
        + "quiet line under the header is the press of 17:04, whose answer was a miss: "
        + "GitHub did not answer it either, and a button pressed is owed what the run did. It stays there until the next "
        + "press of Update or of Install extension.",
      fixture: "r8", moment: "2026-10-17T17:05:00Z", alive: [],
      facts: machine(
        latestStable: "v2.15.0",
        network: NetworkFact(reachable: false),
        updateRun: UpdateRun(miss: .unreachable, at: Date(timeIntervalSince1970: 1_792_256_640))),
      ui: UIState(), appearance: nil),
    Scene(
      id: "r9", title: "You stopped the watch",
      caption:
        "Nothing is running, so NOW is not drawn at all. The panel is the header, the "
        + "last five documents and SETTINGS, which is every screen with nothing going "
        + "on. The publish of 14:31 is one of those five because its end named the one "
        + "it created, and the restyle of 10:12 is the one it pushed out. The header "
        + "still carries the release the check found at 17:05.",
      fixture: "r9", moment: "2026-10-17T18:59:00Z", alive: [],
      facts: machine(latestStable: "v2.15.0"), ui: UIState(), appearance: nil),
    Scene(
      id: "e1", title: "Twelve documents, the pinned one and the last five",
      caption:
        "The record holds twelve documents and the panel draws six: the pin, and then "
        + "the five gdoc touched last. Nothing scrolls and no row unfolds, so the list "
        + "is the whole of what DOCUMENTS has to say. The first row is the wallet note, "
        + "pinned two days ago and worked on eight days ago: without the pin it would "
        + "be one of the five the list leaves out. The pin is the fourth and last icon "
        + "of every row: the filled tack on the pinned one, the outline on the five "
        + "under it, "
        + "and pressing it is how a document is pinned or let go. Hide is the row's "
        + "own menu, which no still can show. One row carries a mark: the reply on "
        + "Chapter 5 failed at 11:40, and a run that failed is the one thing the mark "
        + "means since 2026-10-26. The five rows around it say what their last run was "
        + "and draw nothing in front of it. NOW is empty because the person dismissed "
        + "that failure's card, and a dismissal hides a card and changes no row. "
        + "The plate under the second row, over the two below it, is the card, which "
        + "a pointer resting half a second on a document's name brings up and which "
        + "goes when it leaves, is held down or is clicked. The second row is shaded "
        + "because the card is only ever up on a hovered row. "
        + "It is the row uncut and never the row plus: its "
        + "first line is the whole name, its second what last happened and when, "
        + "written out with the day and the year, and its third where the note is. "
        + "The fourth line, Click opens in Google Docs, is the one line that is not "
        + "about the document: it goes after the person's own first click on a row "
        + "and never comes back, so it is on this screen and on no later one.",
      fixture: "e1", moment: "2026-10-17T18:10:00Z", alive: [],
      facts: machine(),
      ui: UIState(
        dismissed: ["3000000000000003|1"],
        // The first of the five under the pin, which is the export of
        // 18:10 that the moment of this scene is.
        forcedCard: SceneDocuments.concept,
        cardHintSeen: false,
        pinned: [
          DocumentChoice(
            doc: SceneDocuments.wallet, at: Date(timeIntervalSince1970: 1_792_141_200),
            title: "Wallet note: the partner shortlist")
        ]),
      appearance: nil),
    Scene(
      id: "e2", title: "The next morning: a pin outlives the fortnight",
      caption:
        "The first row is a note nobody has touched for twenty days. A document leaves "
        + "the list fourteen days after its last action, and a pin is what holds one "
        + "there: the row keeps the title and the moment of that last action, and says "
        + "nothing a pin invented. The four under it are the fortnight's own, newest "
        + "first, across three days. This is also the one screen whose Launch at login "
        + "is on: the person turned the switch on the evening before, which is the only "
        + "way it is ever on. The firm's mark is at the right end of DOCUMENTS: 14 pt "
        + "tall, on the one edge every value in the panel ends on, in the label's own "
        + "grey, and nothing presses it.",
      fixture: "e2", moment: "2026-10-18T09:12:00Z", alive: [],
      facts: machine(loginItem: .on),
      ui: UIState(
        pinned: [
          DocumentChoice(
            doc: SceneDocuments.wallet, at: Date(timeIntervalSince1970: 1_792_152_000),
            title: "Wallet note: the partner shortlist")
        ]),
      appearance: nil),
    Scene(
      id: "e3", title: "Everything at once: the order of NOW",
      caption:
        "Holds first, then what is stuck, then running jobs, then watches, oldest first "
        + "inside each class. What is stuck here is an export, so the publish running "
        + "under it clears nothing: a stuck card leaves when the next run of its own "
        + "command starts. The card carries four icons, the note, the chat, the link and "
        + "the cross, and a click on its words opens the document.",
      fixture: "e3", moment: "2026-10-17T17:20:00Z", alive: [7200, 7210, 7103],
      facts: machine(), ui: UIState(), appearance: nil),
    Scene(
      id: "e5", title: "The first run: no token at all",
      caption:
        "Nothing works without a sign-in, so the header says so and the one sentence "
        + "names the icon to press. Nothing else on this machine is set up either: no "
        + "extension has been written, so Claude Desktop carries the one thing to do "
        + "about it. No document has run here yet, so there is no DOCUMENTS row on this "
        + "screen and no note icon of any tone; the grey one is on e6, e9 and e10. "
        + "SETTINGS is still pinned above everything, because on this machine it is the "
        + "whole of what the panel is for.",
      fixture: nil, moment: "2026-10-17T10:00:00Z", alive: [],
      facts: machine(
        auth: .signedOut, account: nil, gdocFound: true, extensionState: .notInstalled,
        pairing: [:]),
      ui: UIState(), appearance: nil),
    Scene(
      id: "e6", title: "Icons with nothing to open, and the first hint",
      caption:
        "An unpaired document greys its note icon and says why. The strip of four labels "
        + "is shown once under the first row. A document in the trash keeps its row: no "
        + "field says it is gone, so it is not struck through. The Claude Desktop row "
        + "carries the other half of the fourth look: Install extension was pressed at "
        + "10:58, the release gdoc found was a major one it declines without the flag, "
        + "so nothing was written and the row says which command takes it. That is also "
        + "why Continue with Claude is grey on every row here: no extension, no chat.",
      fixture: "e6", moment: "2026-10-17T11:00:00Z", alive: [],
      facts: machine(
        extensionState: .notInstalled,
        extensionRun: ExtensionRun(miss: .major, at: Date(timeIntervalSince1970: 1_792_234_680))),
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
        + "be started again. This is also the one screen of a machine with no picture of "
        + "the firm's mark cached, so DOCUMENTS is its words alone.",
      fixture: "e9", moment: "2026-10-17T15:30:00Z", alive: [],
      facts: machine(auth: .needsSignIn, logo: nil), ui: UIState(), appearance: nil),
    Scene(
      id: "e10", title: "Signing out, and changing account",
      caption:
        "The confirm names what will stop, in place of the header's own line, and waits "
        + "for the answer. The line it replaced is the whole line: this is the one "
        + "screen with no version on it and the one with no update glyph, and the quiet "
        + "line about the address that changed is off it too. All four come back when "
        + "the question is answered. A changed address is the app's own memory and never "
        + "a line of the record.",
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

  /// This scene as a view, drawn for an image.
  ///
  /// One view and not two. Until 2026-10-18 a scene was rendered twice, once
  /// capped at 600 pt as the person opened it and once whole for the
  /// reviewer, because a PNG cannot be scrolled. Nothing is capped now, so
  /// the two images are one image.
  @MainActor
  func view() throws -> some View {
    PanelView(model: try model()).rendering()
  }
}
