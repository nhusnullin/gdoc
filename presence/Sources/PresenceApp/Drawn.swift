// What the screen is made of, as far as it comes out of the record.
//
// The session redraws when this changes and not otherwise, so what is in here
// decides what a person sees move. It is the four folds the menu bar and the
// panel are really built from, taken together: anything one of them reads is
// something a redraw must notice, and anything none of them reads is not.
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

/// The record and the clock, folded the four ways the screen reads them.
struct Drawn: Equatable {
  let now: NowFold
  let sessions: [LiveSession]
  let days: DaysFold
  let documents: DocumentsFold

  init(
    _ events: [Event], now moment: Date, dismissed: Set<String>, calendar: Calendar,
    alive: (Int32) -> Bool
  ) {
    self.now = Now.fold(events, now: moment, dismissed: dismissed, alive: alive)
    self.sessions = Live.sessions(events, now: moment)
    self.days = Days.fold(events, now: moment, calendar: calendar, alive: alive)
    self.documents = Documents.fold(events, now: moment, calendar: calendar, alive: alive)
  }
}
