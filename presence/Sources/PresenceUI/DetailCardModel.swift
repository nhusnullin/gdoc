// The card a rested pointer brings up over a document row.
//
// It is the row uncut and never the row plus, which is the 2026-10-28
// decision's own sentence. The panel is 360 points wide and a row has one
// line for its name and one for its state and its note, so both are cut on
// the documents a person works on. The card is where those two lines are
// whole: the same facts, in the same words, with nothing measured off them.
//
// Four fields, and the test that holds the count is
// StateTests.testTheCardIsTheRowUncutAndNeverTheRowPlus. A fifth would be
// the card saying something the row does not, which is the thing this file
// is here to refuse: nothing in it is read from Drive, nothing is counted,
// and nothing is judged.
//
// The one line that is not about the document is `hint`, and it retires
// itself: it says that a click opens the document until the person's own
// first click on a row, which is Preferences' ninth key.

import Foundation

/// What the card draws, as the four strings it is.
public struct DetailCardModel: Sendable, Equatable {
  /// The document's name, whole: the title the record last gave it, or its
  /// id where the record has never named it. It is the row's own title.
  public let title: String
  /// What happened to it and when: the second line's own state words, with
  /// the moment written whole where that line has one. It is nothing for a
  /// row that is in the list only because somebody pinned it, because the
  /// pin is not an action and a sentence about one would be invented.
  public let when: String?
  /// Where the note is, with the home as a tilde, or the row's own words
  /// for a document no note is paired with.
  public let note: String
  /// The one line that is not a fact about the document, until the first
  /// click on a row takes it away for good.
  public let hint: String?

  public init(title: String, when: String?, note: String, hint: String?) {
    self.title = title
    self.when = when
    self.note = note
    self.hint = hint
  }
}
