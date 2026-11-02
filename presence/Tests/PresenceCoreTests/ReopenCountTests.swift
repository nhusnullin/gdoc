// The five seconds between an update that replaced the bundle and the open,
// as the state machine the notice row reads down.
//
// Every answer is a function of the state, the gate the session learned on
// this ask and the clock, so the whole of the count is reachable from here
// and nothing about it waits on a real second. The rules each have their own
// test below: the count begins on the first clear gate, it reads 5 down to
// 1, it opens once, a held gate under five seconds does not move it, a held
// gate at five sends it back, a refused gate is for good, and the number on
// the row is clamped.

import Foundation
import XCTest

@testable import PresenceCore

final class ReopenCountTests: XCTestCase {
  /// A fixed moment, so every test reads offsets from one number.
  private let t0 = Date(timeIntervalSince1970: 1_792_000_000)

  func testTheCountBeginsOnTheFirstClearGateAndNotBefore() {
    // Arrange: the bundle moved while a gdoc update was still live in the
    // record, which is the gate the first ask almost always sees.
    let held = ReopenCount.held

    // Act and assert: a held gate says nothing and moves nothing, however
    // long it is held for.
    let first = held.asked(.held, now: t0)
    XCTAssertEqual(first.standing, .held)
    XCTAssertFalse(first.opensNow)
    XCTAssertEqual(first.count, .held, "a held gate leaves the count where it was")

    let later = held.asked(.held, now: t0.addingTimeInterval(30))
    XCTAssertEqual(later.standing, .held)
    XCTAssertEqual(later.count, .held, "thirty seconds of holding is still a hold")

    // Act and assert: the first clear gate starts the clock at the moment it
    // cleared, and the row reads the whole five.
    let t1 = t0.addingTimeInterval(30)
    let begun = held.asked(.clear, now: t1)
    XCTAssertEqual(begun.standing, .counting(5))
    XCTAssertFalse(begun.opensNow, "the open is at zero and not at five")
    XCTAssertEqual(begun.count, .counting(began: t1))
  }

  func testTheCountReadsFiveDownToOneAndOpensOnceAtFive() {
    let t1 = t0
    let counting = ReopenCount.counting(began: t1)

    for (elapsed, left) in [(0.0, 5), (0.99, 5), (1.0, 4), (2.0, 3), (3.0, 2), (4.0, 1), (4.99, 1)]
    {
      let step = counting.asked(.clear, now: t1.addingTimeInterval(elapsed))
      XCTAssertEqual(step.standing, .counting(left), "\(elapsed) seconds in, the row reads \(left)")
      XCTAssertFalse(step.opensNow, "nothing opens before the five seconds are spent")
      XCTAssertEqual(step.count, .counting(began: t1), "the count is not moved by being read")
    }

    // Act: the ask at five seconds, with the policy still clear.
    let zero = counting.asked(.clear, now: t1.addingTimeInterval(5))
    XCTAssertEqual(zero.standing, .counting(1), "the row holds at 1 while the open is asked for")
    XCTAssertTrue(zero.opensNow)
    XCTAssertEqual(zero.count, .opening)

    // Assert: every ask after that opens nothing, whatever the gate says,
    // because the open was asked for and its answer is what moves the count.
    for (elapsed, gate) in [(5.5, ReopenGate.clear), (6, .held), (60, .clear)] {
      let step = ReopenCount.opening.asked(gate, now: t1.addingTimeInterval(elapsed))
      XCTAssertEqual(step.standing, .counting(1), "the row says 1 until this instance goes")
      XCTAssertFalse(step.opensNow, "one open per count")
      XCTAssertEqual(step.count, .opening)
    }
  }

  func testAHeldGateBeforeFiveSecondsKeepsCountingAndAtFiveGoesBackToHeld() {
    let t1 = t0
    let counting = ReopenCount.counting(began: t1)

    // A poll child of a few hundred milliseconds holds the gate for one ask.
    // A row that jumped back to the update's words would be the panel losing
    // its place over something the person cannot see.
    let during = counting.asked(.held, now: t1.addingTimeInterval(2))
    XCTAssertEqual(during.standing, .counting(3))
    XCTAssertFalse(during.opensNow)
    XCTAssertEqual(during.count, .counting(began: t1), "a hold under five seconds moves nothing")

    // At five seconds a held gate is the bytes still moving, and a 1 that
    // waited would be a promise. So the row goes back to the hold's words.
    let atFive = counting.asked(.held, now: t1.addingTimeInterval(5))
    XCTAssertEqual(atFive.standing, .held)
    XCTAssertFalse(atFive.opensNow)
    XCTAssertEqual(atFive.count, .held)

    // And the count starts over from the moment the gate cleared, so the row
    // never freezes at 1.
    let again = atFive.count.asked(.clear, now: t1.addingTimeInterval(9))
    XCTAssertEqual(again.standing, .counting(5))
    XCTAssertEqual(again.count, .counting(began: t1.addingTimeInterval(9)))
  }

  func testARefusedGateRefusesForGoodFromAnyState() {
    // A hold the session cannot wait out, a signature that did not check or
    // another identifier on the disk, is the end of the count: the row goes
    // back to asking for a hand.
    for state in [ReopenCount.held, .counting(began: t0), .opening] {
      let step = state.asked(.refused, now: t0.addingTimeInterval(1))
      XCTAssertEqual(step.standing, .refused)
      XCTAssertFalse(step.opensNow)
      XCTAssertEqual(step.count, .refused)
    }

    for (gate, elapsed) in [(ReopenGate.clear, 0.0), (.held, 10), (.refused, 600)] {
      let step = ReopenCount.refused.asked(gate, now: t0.addingTimeInterval(elapsed))
      XCTAssertEqual(step.standing, .refused, "refused is for good")
      XCTAssertFalse(step.opensNow)
      XCTAssertEqual(step.count, .refused)
    }

    XCTAssertTrue(ReopenCount.refused.isRefused)
    XCTAssertFalse(ReopenCount.held.isRefused)
    XCTAssertFalse(ReopenCount.counting(began: t0).isRefused)
    XCTAssertFalse(ReopenCount.opening.isRefused)
  }

  func testTheRowIsClampedToOneThroughFive() {
    let t1 = t0
    let counting = ReopenCount.counting(began: t1)

    // A clock that went back reads the whole five and never a 6.
    let back = counting.asked(.clear, now: t1.addingTimeInterval(-3))
    XCTAssertEqual(back.standing, .counting(5))
    XCTAssertFalse(back.opensNow, "a clock that went back opens nothing either")

    // A count that was never opened because the gate kept holding reads the
    // hold's words and not a negative number.
    let long = counting.asked(.held, now: t1.addingTimeInterval(40))
    XCTAssertEqual(long.standing, .held)
    XCTAssertEqual(long.count, .held)
  }

  func testTheCountIsFiveSeconds() {
    XCTAssertEqual(
      ReopenCount.seconds, 5,
      "the literal, because this is the wait the relauncher's settle used to be")
  }
}
