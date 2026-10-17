// The title split, against the screens file's own function.
//
// `TT()` in docs/design/2026-10-17-presence-screens-v3.html is eight lines
// of JavaScript, and this file walks it case by case over invented titles of
// 20, 40, 60 and 80 characters, the lengths the hub really holds. Every
// number below is a literal read off that line.

import XCTest

@testable import PresenceCore

final class TitlesTests: XCTestCase {
  /// The four lengths, invented so no real document's name is in the tree.
  private let twenty = "Pricing one-pager v2"
  private let forty = "Partner onboarding checklists for Munich"
  private let sixty = "Treasury reconciliation runbook for the quarterly close call"
  private let eighty =
    "Second-quarter liquidity and settlement reviews for the Amsterdam trading office"

  func testTheSplitIsTheScreensRule() {
    XCTAssertEqual(Titles.wholeLimit, 26)
    XCTAssertEqual(Titles.tailLimit, 14)
    XCTAssertEqual(Titles.headFloor, 8)

    // The lengths are what they say they are, so the cases below are the
    // cases they claim to be.
    XCTAssertEqual(twenty.count, 20)
    XCTAssertEqual(forty.count, 40)
    XCTAssertEqual(sixty.count, 60)
    XCTAssertEqual(eighty.count, 80)

    // Twenty characters: `t.length<=26`, so the head is the whole title and
    // there is no tail.
    XCTAssertEqual(Titles.split(twenty), Titles.Split(head: twenty))

    // Forty: the last space, a tail of seven counting the space.
    XCTAssertEqual(
      Titles.split(forty),
      Titles.Split(head: "Partner onboarding checklists for", tail: " Munich"))
    XCTAssertEqual(Titles.split(forty).tail?.count, 7)

    // Sixty: the same rule, a tail of five.
    XCTAssertEqual(
      Titles.split(sixty),
      Titles.Split(
        head: "Treasury reconciliation runbook for the quarterly close", tail: " call"))

    // Eighty: a tail of seven, and the head is the other seventy-three.
    XCTAssertEqual(
      Titles.split(eighty),
      Titles.Split(
        head: "Second-quarter liquidity and settlement reviews for the Amsterdam trading",
        tail: " office"))

    // Every split puts the title back together, which is what the tooltip
    // shows.
    for title in [twenty, forty, sixty, eighty] {
      XCTAssertEqual(Titles.split(title).whole, title)
    }

    // The boundaries, one character either side of each number.
    let twentySix = String(repeating: "a", count: 21) + " tail"
    XCTAssertEqual(twentySix.count, 26)
    XCTAssertEqual(
      Titles.split(twentySix), Titles.Split(head: twentySix),
      "26 is whole, because the rule is `<=26`")

    let twentySeven = String(repeating: "a", count: 22) + " tail"
    XCTAssertEqual(twentySeven.count, 27)
    XCTAssertEqual(
      Titles.split(twentySeven),
      Titles.Split(head: String(repeating: "a", count: 22), tail: " tail"),
      "27 splits")

    // A tail of fourteen is the longest the rule takes: the space and
    // thirteen characters.
    let thirteenWord = String(repeating: "a", count: 20) + " " + String(repeating: "b", count: 13)
    XCTAssertEqual(Titles.split(thirteenWord).tail?.count, 14)
    XCTAssertEqual(Titles.split(thirteenWord).head, String(repeating: "a", count: 20))

    let fourteenWord = String(repeating: "a", count: 20) + " " + String(repeating: "b", count: 14)
    XCTAssertEqual(
      Titles.split(fourteenWord), Titles.Split(head: fourteenWord),
      "a tail of fifteen leaves the title whole, head only")

    // A long title with no space at all: one piece, and no last character
    // taken for a tail.
    let noSpace = String(repeating: "x", count: 40)
    XCTAssertEqual(Titles.split(noSpace), Titles.Split(head: noSpace))

    // A trailing space is its own tail of one, which is `TT()` to the
    // letter and draws as the title with nothing after it.
    let trailing = String(repeating: "a", count: 30) + " "
    XCTAssertEqual(
      Titles.split(trailing),
      Titles.Split(head: String(repeating: "a", count: 30), tail: " "))

    // The empty title is whole and empty, which a row with no title draws as
    // nothing.
    XCTAssertEqual(Titles.split(""), Titles.Split(head: ""))
  }
}
