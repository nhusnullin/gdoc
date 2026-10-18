// The folds over M24b's new fields.
//
// Five of the six fields are words the panel writes, and those are held in
// PresenceUITests/FieldsTests. The one that moves a row from nowhere to
// somewhere is `made`: a publish names no document until it has created
// one, so without the field every publish that worked left no row in
// DOCUMENTS at all.

import Foundation
import XCTest

@testable import PresenceCore

final class FieldsTests: XCTestCase {
  private let chapter4 = "1Ch4BbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQq"
  private let policy = "1GcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSs"

  // `made` is the document a publish created, and it is the only route a
  // publish has to a row: the start names nothing and the end names nothing
  // either until the upload came back with an id. A rollback leaves the end
  // without it, and then there is nothing to open, which is what the row
  // says.
  func testAPublishedRowTakesItsIdFromMade() throws {
    let at = try moment("2026-10-17T15:00:00Z")
    let published = [
      start(at: "2026-10-17T14:29:00Z", command: "publish", run: "p1", doc: nil),
      end(
        at: "2026-10-17T14:31:00Z", command: "publish", run: "p1", doc: nil, ok: true,
        verified: true, title: "Chapter 4", made: chapter4),
    ]

    let documents = Documents.fold(published, now: at, alive: { _ in false })
    XCTAssertEqual(documents.rows.map(\.doc), [chapter4], "the created document is the row")
    XCTAssertEqual(documents.rows.first?.title, "Chapter 4")
    XCTAssertEqual(documents.rows.first?.state, .done)

    let now = Now.fold(published, now: at, alive: { _ in false })
    XCTAssertEqual(
      now.lastVerified?.doc, chapter4, "so the check is about a document somebody can open")

    let rolledBack = [
      start(at: "2026-10-17T14:29:00Z", command: "publish", run: "p2", doc: nil),
      end(
        at: "2026-10-17T14:31:00Z", command: "publish", run: "p2", doc: nil, ok: false,
        stop: "network"),
    ]
    XCTAssertTrue(
      Documents.fold(rolledBack, now: at, alive: { _ in false }).rows.isEmpty,
      "a rollback created nothing, so there is no document to put in the list")
  }

  // A document remembers the Claude Code session its latest `via: code` run
  // came from, and remembers none where no run named one.
  //
  // The latest wins, as a title does: a document worked on from two sessions
  // belongs to the one a hand went to last. A chat run and a terminal run
  // name no session at all, which is what every document carried before gdoc
  // wrote the field.
  func testARowRemembersTheSessionOfItsLatestCodeRun() throws {
    let at = try moment("2026-10-17T15:00:00Z")
    let first = "0199c3d4-5e6f-7a8b-9c0d-1e2f3a4b5c6d"
    let second = "0199dead-beef-4a8b-9c0d-1e2f3a4b5c6d"

    let events =
      [
        start(
          at: "2026-10-17T10:00:00Z", command: "propose", run: "r1", doc: chapter4, via: "code",
          session: first),
        end(
          at: "2026-10-17T10:00:20Z", command: "propose", run: "r1", doc: chapter4, via: "code",
          ok: true, verified: true, title: "Chapter 4"),
        start(
          at: "2026-10-17T13:00:00Z", command: "reply", run: "r2", doc: chapter4, via: "code",
          session: second),
        end(
          at: "2026-10-17T13:00:20Z", command: "reply", run: "r2", doc: chapter4, via: "code",
          ok: true, verified: true, title: "Chapter 4"),
      ]
      + pair(
        at: "2026-10-17T11:00:00Z", command: "reply", run: "r3", doc: policy, via: "chat",
        verified: true, title: "The policy")

    let rows = Documents.fold(events, now: at, alive: { _ in false }).rows
    XCTAssertEqual(
      rows.first(where: { $0.doc == chapter4 })?.session, second,
      "the session of the latest Claude Code run is the one the row remembers")
    XCTAssertNil(
      rows.first(where: { $0.doc == policy })?.session, "a chat run names no session")
  }

  // A document gdoc published from Claude Code remembers that session too.
  //
  // A publish is handed no document, so its start names none and its session
  // sits on a line with no document on it; the id it brought into the world
  // is `made`, on the end. The session is read per call for that reason, as
  // a title is: a document gdoc has just made is the one a hand is most
  // likely to go back to the session for.
  func testAPublishedDocumentRemembersTheSessionItWasPublishedFrom() throws {
    let at = try moment("2026-10-17T15:00:00Z")
    let session = "0199c3d4-5e6f-7a8b-9c0d-1e2f3a4b5c6d"

    let events = [
      start(
        at: "2026-10-17T12:00:00Z", command: "publish", run: "p1", doc: nil, via: "code",
        title: "Q3 plan", session: session),
      end(
        at: "2026-10-17T12:00:40Z", command: "publish", run: "p1", doc: nil, via: "code",
        ok: true, verified: true, title: "Q3 plan", made: chapter4),
    ]

    let rows = Documents.fold(events, now: at, alive: { _ in false }).rows
    XCTAssertEqual(rows.map(\.doc), [chapter4], "the publish made the one row")
    XCTAssertEqual(
      rows.first?.session, session,
      "and the document it made remembers the session it was published from")
  }

  // A hold the sign-in changed under is dropped, which is a fourth ending
  // and not a fourth kind of row: the card goes, and the document stops
  // being held.
  func testADroppedHoldLeavesNoCardAndNoRow() throws {
    let at = try moment("2026-10-17T17:00:00Z")
    let events = [
      hold(at: "2026-10-17T16:20:00Z", run: "h1", doc: policy, id: "h_1"),
      holdEnd(at: "2026-10-17T16:41:00Z", kind: .dropped, run: "h1", doc: policy, id: "h_1"),
    ]

    XCTAssertTrue(
      Now.fold(events, now: at, alive: { _ in true }).holds.isEmpty,
      "the hold is gone, so the card is gone")
    XCTAssertTrue(
      Documents.fold(events, now: at, alive: { _ in true }).rows.isEmpty,
      "a hold that ended is no open action, and a hold alone is no other action either")

    XCTAssertEqual(
      HoldEnding.allCases, [.released, .expired, .landed, .dropped],
      "the four ways a hold stops waiting, which are the record's own four kinds")
  }
}
