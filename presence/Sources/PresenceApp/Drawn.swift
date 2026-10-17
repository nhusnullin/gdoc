// What the screen is made of, as far as it comes out of the record.
//
// The session redraws when this changes and not otherwise, so what is in here
// decides what a person sees move. It is the four folds the menu bar and the
// panel are really built from, taken together: anything one of them reads is
// something a redraw must notice, and anything none of them reads is not.
//
// One thing in here is the clock and not a line: the minutes the rows that
// count one are showing, where a hold or a live watch is on the screen. Those
// two rows say how long something has been going, which moves with no event
// behind it.
//
// It is not a summary. A summary is where the two bugs this replaces came
// from: a fold that skipped `step` lines left a publish's ring and job card
// frozen at the first part, and one that knew nothing about the
// three-minute gap left the menu bar live long after the last poll. Both are
// time or lines that only these four read.
//
// Nothing of the machine is here. The facts have their own route: every one
// of them arrives through `recompose`, which redraws when they move.

import Foundation
import PresenceCore
import PresenceUI

/// The record and the clock, folded the four ways the screen reads them.
struct Drawn: Equatable {
  let now: NowFold
  let sessions: [LiveSession]
  let days: DaysFold
  let documents: DocumentsFold
  /// The whole minutes the screen's own stretches are showing, and nothing
  /// where nothing on it counts one.
  let counting: [Int]?

  init(
    _ events: [Event], now moment: Date, dismissed: Set<String>, calendar: Calendar,
    alive: (Int32) -> Bool
  ) {
    let fold = Now.fold(events, now: moment, dismissed: dismissed, alive: alive)
    let watches = Live.sessions(events, now: moment, alive: alive)
    self.now = fold
    self.sessions = watches
    self.days = Days.fold(
      events, now: moment, calendar: calendar, carded: Set(fold.stuck.map(\.key)), alive: alive)
    self.documents = Documents.fold(events, now: moment, calendar: calendar, alive: alive)
    self.counting = Drawn.counting(
      moment,
      from: fold.holds.filter { !$0.ranOut }.map(\.held)
        + watches.filter(\.isOpen).map(\.quietSince))
  }

  /// The whole minutes the stretches on the screen are showing, and nothing
  /// where nothing on it counts one.
  ///
  /// A hold says how long it has been waiting and a live row how long it has
  /// been quiet, and both of those are the clock against a line rather than a
  /// line of their own: "quiet 2 h" becomes "quiet 2 h 1 min" with no event
  /// behind it.
  ///
  /// It is each stretch's own minute and never the clock's. A hold raised at
  /// 10:00:30 shows its first minute at 10:01:30, so a key cut at the wall
  /// clock would have left the panel a minute behind its own rule for up to
  /// 59 seconds. `Words.minutes` is the number the row says, so the key and
  /// the words cannot drift. A screen with neither row on it redraws as
  /// rarely as it did before:
  /// testAQuietMinuteRedrawsAHoldAndAWatchAndNothingElse.
  private static func counting(_ moment: Date, from stretches: [Date]) -> [Int]? {
    guard !stretches.isEmpty else { return nil }
    return stretches.map { Words.minutes(moment.timeIntervalSince($0)) }
  }
}
