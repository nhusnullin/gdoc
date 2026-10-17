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
// Five scenes draw a fallback rather than what the screens show, because
// the v1 record does not carry the field: r1 has no title for a publish and
// no steps, r2 and the other stuck cards have no cause, r8's failed poll has
// no reason, e6's gone document has no field saying so, and e9's card comes
// from `auth status` alone. M24b adds the five fields and Task 25 puts them
// into these same scenes. Scene e4 is not built at all: releasing a hold
// without writing is in the specification's "what is not built".

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
  static let chapter5 = "1Ch5AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPp"
  static let northwind = "1NwBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQq"
  static let partner = "1PbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRr"
  static let policy = "1GcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSs"
  static let concept = "1B2EeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTt"
  static let proposal = "1PrFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUu"

  /// The hub index: a document, and the note paired with it. Northwind and
  /// Partner bank are in no note, which is what greys their md icon.
  static let pairing: [String: String] = [
    chapter5: "falcon/chapters/05.md",
    policy: "05-rules/group-channel.md",
    concept: "falcon/b2b2c-concept.md",
    proposal: "falcon/proposal-mea-v1.md",
  ]
}

enum Scenes {
  static let account = "nail@example.com"
  static let hub = "/Users/nail/hub"
  static let gdocPath = "/Users/nail/.local/bin/gdoc"
  static let signInURL =
    "https://accounts.google.com/o/oauth2/auth?client_id=…&redirect_uri=http://127.0.0.1:0"

  /// A machine that is signed in and up to date, which most scenes are.
  static func machine(
    auth: AuthWord = .ok,
    account: String? = Scenes.account,
    previousAccount: String? = nil,
    latestStable: String? = nil,
    gdocFound: Bool = true,
    extensionState: ExtensionState = .connected,
    network: NetworkFact? = nil,
    signIn: SignInPhase? = nil,
    signedInAt: Date? = nil
  ) -> Facts {
    Facts(
      account: account,
      auth: auth,
      previousAccount: previousAccount,
      versions: Versions(
        app: "2.14.0", gdoc: gdocFound ? "v2.14.0" : nil, latestStable: latestStable,
        skills: "v2.14.0 · altery plugin"),
      mark: "⁘",
      gdocPath: gdocFound ? gdocPath : nil,
      hub: hub,
      extensionState: extensionState,
      network: network,
      loginItem: .on,
      signIn: signIn,
      pairing: SceneDocuments.pairing,
      signedInAt: signedInAt)
  }

  /// Every scene, in the order the screens file tells the day.
  static let all: [Scene] = [
    Scene(
      id: "r1", title: "Publishing Chapter 4",
      caption:
        "A publish is running. The v1 record names no document for a publish, so the card "
        + "is the ring and the word, with no title and no steps.",
      fixture: "r1", moment: "2026-10-17T09:04:00Z", alive: [7101],
      facts: machine(), ui: UIState(), appearance: nil),
    Scene(
      id: "r2", title: "The publish stopped",
      caption:
        "The write failed. The v1 record carries no cause, so the reason names the moment "
        + "and where the reason is, in one sentence under eighty characters.",
      fixture: "r2", moment: "2026-10-17T09:05:30Z", alive: [],
      facts: machine(), ui: UIState(), appearance: nil),
    Scene(
      id: "r3", title: "A comment and a reply, both read back",
      caption:
        "Two writes landed and were verified. The stuck card stays until the publish is "
        + "run again, and Copy link on the first row is a second into its green check.",
      fixture: "r3", moment: "2026-10-17T11:40:30Z", alive: [],
      facts: machine(), ui: UIState(copied: SceneDocuments.chapter5), appearance: nil),
    Scene(
      id: "r3l", title: "The same screen in the light appearance",
      caption:
        "The panel follows the system and nothing else: the same three surfaces, three "
        + "text levels and hairlines in their light values.",
      fixture: "r3", moment: "2026-10-17T11:40:30Z", alive: [],
      facts: machine(), ui: UIState(copied: SceneDocuments.chapter5), appearance: .light),
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
        "The last poll failed. The record says that it failed and not why, so the row "
        + "says the moment alone; HEALTH carries what the machine knows.",
      fixture: "r8", moment: "2026-10-17T17:05:00Z", alive: [],
      facts: machine(
        latestStable: "v2.15.0",
        network: NetworkFact(
          reachable: false, detail: "the internet is up, the handshake with Google fails")),
      ui: UIState(), appearance: nil),
    Scene(
      id: "r9", title: "You stopped the watch",
      caption:
        "Nothing is running, so NOW is not drawn at all. The day's log is behind Show "
        + "more and yesterday is one row.",
      fixture: "r9", moment: "2026-10-17T18:59:00Z", alive: [],
      facts: machine(), ui: UIState(), appearance: nil),
    Scene(
      id: "e1", title: "More than five documents",
      caption:
        "DOCUMENTS opened: the list scrolls inside its own box with the thumb and the "
        + "fade, and the panel stays under 600 pt.",
      fixture: "e1", moment: "2026-10-17T18:10:00Z", alive: [],
      facts: machine(), ui: UIState(expanded: [.documents]), appearance: nil),
    Scene(
      id: "e2", title: "Midnight: yesterday folds into one row",
      caption:
        "The local day decides. Yesterday is one chevron row with its two counts, and "
        + "nothing older is in the panel at all.",
      fixture: "e2", moment: "2026-10-18T09:12:00Z", alive: [],
      facts: machine(), ui: UIState(), appearance: nil),
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
        "Nothing works without a sign-in, so HEALTH is pinned above everything and the "
        + "one sentence names the row to press.",
      fixture: nil, moment: "2026-10-17T10:00:00Z", alive: [],
      facts: machine(
        auth: .signedOut, account: nil, gdocFound: true, extensionState: .notInstalled),
      ui: UIState(), appearance: nil),
    Scene(
      id: "e6", title: "Icons with nothing to open, and the first hint",
      caption:
        "An unpaired document greys its note icon and says why. The strip of four labels "
        + "is shown once under the first row. A document in the trash keeps its row; the "
        + "v1 record cannot say it is gone, so it is not struck through yet.",
      fixture: "e6", moment: "2026-10-17T11:00:00Z", alive: [],
      facts: machine(),
      ui: UIState(
        forcedTooltip: "\(SceneDocuments.northwind)|openNote", stripSeen: false),
      appearance: nil),
    Scene(
      id: "e8", title: "Waiting for Google, and just signed in",
      caption:
        "The sign-in is the child's own three minutes, with the URL as a line that can be "
        + "copied. The address is said once at its full size for five seconds.",
      fixture: nil, moment: "2026-10-17T09:01:00Z", alive: [],
      facts: machine(
        signIn: SignInPhase(url: signInURL, secondsLeft: 163),
        signedInAt: Date(timeIntervalSince1970: 1_792_911_659)),
      ui: UIState(), appearance: nil),
    Scene(
      id: "e9", title: "The sign-in expired in the middle of a watch",
      caption:
        "The card comes from `auth status` and not from the record. The watch is not "
        + "killed: after a sign-in it carries on from the same cursor.",
      fixture: "e9", moment: "2026-10-17T15:30:00Z", alive: [],
      facts: machine(auth: .needsSignIn), ui: UIState(), appearance: nil),
    Scene(
      id: "e10", title: "Signing out, and changing account",
      caption:
        "The confirm names what will stop, in place and for three seconds. A changed "
        + "address is the app's own memory and never a line of the log.",
      fixture: "e10", moment: "2026-10-17T18:00:00Z", alive: [7200, 7210],
      facts: machine(previousAccount: "nail@gmail.example"),
      ui: UIState(
        question: .signOut, questionAt: Date(timeIntervalSince1970: 1_792_944_000)),
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
      now: fold, sessions: Live.sessions(events, now: at), facts: facts,
      checkShowing: check.isShowing(now: at))
  }

  /// This scene as a view, in the mode the renderer needs.
  ///
  /// `.clipped` is the panel the person opens, capped at 600 pt with the
  /// rest under a scroll. `.whole` is the same panel with nothing hidden,
  /// which is what the reviewer's pass reads: a PNG cannot be scrolled, so
  /// a screen whose HEALTH is past the cap is a screen nobody checked.
  @MainActor
  func view(_ mode: RenderMode = .clipped) throws -> some View {
    PanelView(model: try model()).rendering(mode)
  }
}
