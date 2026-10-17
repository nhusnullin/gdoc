// A document title cut in the middle: a head that may be shortened and a
// tail that never is.
//
// A panel 320 px wide has 140 pt for a document's title, which is about
// twenty characters, and the names in the hub end in the part that tells
// them apart: "…quarterly close call" and "…quarterly close plan" differ in
// their last word. Cutting the end off would hide the one word a person is
// looking for, so the last word is kept and the middle is what goes.
//
// The rule is `TT()` in docs/design/2026-10-17-presence-screens-v3.html,
// ported case by case: a title of 26 characters or fewer is left whole; a
// longer one is split at its last space when the tail, counting that space,
// is 14 characters or fewer; and a longer last word than that leaves the
// title whole, because a tail that wide would leave no head.
//
// This is the whole of the rule a character count can state. The panel adds
// one more, measured rather than counted: `MiddleTitle` drops a tail whose
// width would leave fewer than eight characters of head at the narrowest
// title column. TitlesTests holds this half and RowTests holds that one.

import Foundation

/// The title split, and the three numbers it is made of.
public enum Titles {
  /// `if(t.length<=26)`: a title this short is one piece.
  public static let wholeLimit = 26

  /// `tail.length<=14`, the space counted, so the longest tail the rule
  /// admits is a space and thirteen characters.
  public static let tailLimit = 14

  /// The fewest characters of head the panel ever shows. Measured, not
  /// counted, so it lives in the view; it is stated here because it is part
  /// of the same rule.
  public static let headFloor = 8

  /// What a row draws: the head, and the tail when there is one.
  ///
  /// The tail carries its own leading space, as the stylesheet's
  /// `white-space:pre` tail does, so the two pieces put back together are
  /// the title again.
  public struct Split: Equatable, Sendable {
    public let head: String
    public let tail: String?

    public init(head: String, tail: String? = nil) {
      self.head = head
      self.tail = tail
    }

    /// The title the split came from, for the tooltip, which always carries
    /// the whole of it.
    public var whole: String { head + (tail ?? "") }
  }

  /// The split, as the screens file states it.
  ///
  /// A title with no space at all is left whole. `TT()` would take its last
  /// character as the tail, which is JavaScript's `slice(-1)` rather than a
  /// decision anybody made; one character is not a word and the panel has
  /// no such title, so the case is answered here rather than copied.
  public static func split(_ title: String) -> Split {
    if title.count <= wholeLimit { return Split(head: title) }
    guard let space = title.lastIndex(of: " ") else { return Split(head: title) }
    let tail = String(title[space...])
    guard tail.count <= tailLimit else { return Split(head: title) }
    return Split(head: String(title[title.startIndex..<space]), tail: tail)
  }
}
