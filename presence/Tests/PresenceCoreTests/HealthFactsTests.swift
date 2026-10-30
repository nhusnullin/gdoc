// What the two health commands say, and what the app remembers itself.
//
// Every object here is a literal, so the parsers are held to the shapes
// `gdoc help --json` and `gdoc auth status` print rather than to a machine
// that happens to be signed in. The memory is a pure value for the same
// reason: "Account changed" is a sentence about two addresses the app saw,
// and this file is where that is proved to come from nowhere else.

import Foundation
import XCTest

@testable import PresenceCore

final class HealthFactsTests: XCTestCase {

  // MARK: help --json

  func testHelpJSONGivesTheMarkTheVersionAndTheLatest() throws {
    let help = try HelpFacts.decode(
      Data(
        #"""
        {"commands":[{"name":"read"}],
         "settings":{"mark":"⁘","mark_from":"default"},
         "update":{"installed":"v2.14.0","latest_stable":"v2.14.1",
                   "latest_nightly":"v2.15.0","checked_at":"2026-10-17T09:00:00Z"}}
        """#.utf8))
    XCTAssertEqual(help.mark, "⁘")
    XCTAssertEqual(help.markFrom, "default")
    XCTAssertEqual(help.installed, "v2.14.0")
    XCTAssertEqual(help.latestStable, "v2.14.1")

    // The mark is a setting and may be empty, so an empty one is a mark and
    // never a missing field.
    let silent = try HelpFacts.decode(
      Data(#"{"settings":{"mark":"","mark_from":"settings.yaml"}}"#.utf8))
    XCTAssertEqual(silent.mark, "")
    XCTAssertEqual(silent.markFrom, "settings.yaml")

    // A checkout build names no release, so there is nothing to offer.
    let checkout = try HelpFacts.decode(Data(#"{"settings":{"mark":"⁘"}}"#.utf8))
    XCTAssertNil(checkout.installed)
    XCTAssertNil(checkout.latestStable)

    // A field a newer gdoc prints is read past, as the record's decoder
    // reads past one.
    let newer = try HelpFacts.decode(
      Data(#"{"settings":{"mark":"⁘","tomorrow":7},"tomorrow":{"deep":1}}"#.utf8))
    XCTAssertEqual(newer.mark, "⁘")

    let refusals: [(String, String)] = [
      (#"{}"#, "no settings"),
      (#"{"settings":{}}"#, "no settings.mark"),
      (#"{"settings":{"mark":7}}"#, "no settings.mark"),
      (#"{"settings":"⁘"}"#, "no settings"),
      (#"{"settings":{"mark":"⁘"},"update":{"installed":2}}"#, "no update.installed"),
    ]
    for (text, named) in refusals {
      do {
        _ = try HelpFacts.decode(Data(text.utf8))
        XCTFail("\(text) was read as a help answer")
      } catch {
        XCTAssertEqual(error as? HealthFault, .undecodable(named), "for \(text)")
      }
    }
  }

  /// The path the help names for the house logo, and a machine with no
  /// picture.
  ///
  /// The field is absent on a machine that has never built a document, so a
  /// help answer without it is an ordinary answer and not a fault. A
  /// `logo_path` that is not a string is refused by name, which is this
  /// file's strict rule: something other than gdoc wrote that object.
  func testHelpNamesTheLogoPathAndItMayBeAbsent() throws {
    let named = try HelpFacts.decode(
      Data(
        #"""
        {"settings":{"mark":"⁘","mark_from":"default"},
         "logo_path":"/Users/someone/.config/gdoc-agent/assets/1J3uh6po"}
        """#.utf8))
    XCTAssertEqual(named.logoPath, "/Users/someone/.config/gdoc-agent/assets/1J3uh6po")
    XCTAssertEqual(named.mark, "⁘")

    let silent = try HelpFacts.decode(
      Data(
        #"""
        {"settings":{"mark":"⁘","mark_from":"default"},
         "update":{"installed":"v2.26.0","latest_stable":"v2.26.0"}}
        """#.utf8))
    XCTAssertNil(silent.logoPath)
    XCTAssertEqual(silent.mark, "⁘")
    XCTAssertEqual(silent.markFrom, "default")
    XCTAssertEqual(silent.installed, "v2.26.0")
    XCTAssertEqual(silent.latestStable, "v2.26.0")

    do {
      _ = try HelpFacts.decode(
        Data(#"{"settings":{"mark":"⁘"},"logo_path":7}"#.utf8))
      XCTFail("a numeric logo_path was read as a path")
    } catch {
      XCTAssertEqual(error as? HealthFault, .undecodable("no logo_path"))
    }
  }

  // MARK: auth status

  func testAuthStatusGivesTheThreeWordsAndTheEmail() throws {
    let working = try AuthFacts.decode(
      Data(
        #"""
        {"auth_mode":"oauth","token_path":"~/.config/gdoc-agent/token.json",
         "client_source":"built in","token_present":true,"expired":false,
         "scopes":["https://www.googleapis.com/auth/documents"],
         "account":"nail@example.com","account_name":"Nail","version":"v2.14.0"}
        """#.utf8))
    XCTAssertEqual(working.state, .ok)
    XCTAssertEqual(working.account, "nail@example.com")
    XCTAssertEqual(working.version, "v2.14.0")
    XCTAssertTrue(working.tokenPresent)

    let expired = try AuthFacts.decode(
      Data(#"{"token_present":true,"expired":true,"account":"nail@example.com"}"#.utf8))
    XCTAssertEqual(expired.state, .needsSignIn)
    XCTAssertEqual(expired.account, "nail@example.com", "whose token it is is still known")

    let short = try AuthFacts.decode(
      Data(
        #"""
        {"token_present":true,"expired":true,
         "missing_scopes":["https://www.googleapis.com/auth/drive.file"]}
        """#.utf8))
    XCTAssertEqual(
      short.state, .missingScopes,
      "an expired token refreshes itself; a scope it never carried needs a new consent")
    XCTAssertEqual(short.missingScopes.count, 1)

    let out = try AuthFacts.decode(Data(#"{"token_present":false}"#.utf8))
    XCTAssertEqual(out.state, .signedOut)
    XCTAssertNil(out.account)

    XCTAssertEqual(AuthState.allCases.count, 4, "four states, and the panel words three of them")

    let refusals: [(String, String)] = [
      (#"{}"#, "no token_present"),
      (#"{"token_present":"yes"}"#, "no token_present"),
      (#"{"token_present":true,"expired":"no"}"#, "no expired"),
      (#"{"token_present":true,"account":7}"#, "no account"),
      (#"{"token_present":true,"missing_scopes":"drive"}"#, "no missing_scopes"),
    ]
    for (text, named) in refusals {
      do {
        _ = try AuthFacts.decode(Data(text.utf8))
        XCTFail("\(text) was read as an auth status")
      } catch {
        XCTAssertEqual(error as? HealthFault, .undecodable(named), "for \(text)")
      }
    }
  }

  func testATokenWithNoAccountReadsSignedInAccountNotRead() throws {
    // `auth status` leaves the account out when the read did not answer: the
    // token is the fact, and whose it is was what could not be read.
    let unread = try AuthFacts.decode(
      Data(#"{"token_present":true,"expired":false,"version":"v2.14.0"}"#.utf8))
    XCTAssertEqual(unread.state, .ok, "the token works, so nothing is asked of the person")
    XCTAssertNil(unread.account)
    XCTAssertTrue(unread.isAccountUnread, "which is the row's 'signed in, account not read'")

    let known = try AuthFacts.decode(
      Data(#"{"token_present":true,"expired":false,"account":"nail@example.com"}"#.utf8))
    XCTAssertFalse(known.isAccountUnread)

    let out = try AuthFacts.decode(Data(#"{"token_present":false}"#.utf8))
    XCTAssertFalse(out.isAccountUnread, "no token is not an account that could not be read")
  }

  // MARK: the expected domain

  func testTheExpectedDomainIsAHintThatBlocksNothing() throws {
    XCTAssertEqual(DomainHint.unset.expected, "", "empty by default")
    XCTAssertNil(
      DomainHint.unset.sentence(for: "nail@gmail.example"),
      "no expectation, no hint, whatever the address is")

    let firm = DomainHint(expected: "altery.example")
    XCTAssertNil(firm.sentence(for: "nail@altery.example"))
    XCTAssertNil(
      firm.sentence(for: "NAIL@Altery.Example"), "a domain is not read letter by letter")
    XCTAssertEqual(
      firm.sentence(for: "nail@gmail.example"), "the expected domain is altery.example")
    XCTAssertNil(firm.sentence(for: nil), "an account that could not be read is not a mismatch")
    XCTAssertNil(firm.sentence(for: "not-an-address"), "and neither is a word with no domain in it")
    XCTAssertEqual(
      DomainHint(expected: "@altery.example").expected, "altery.example",
      "a preference typed with the at sign is the same preference")

    // It blocks nothing: the state is the token's, and the hint changes no
    // word of it.
    let elsewhere = try AuthFacts.decode(
      Data(#"{"token_present":true,"expired":false,"account":"nail@gmail.example"}"#.utf8))
    XCTAssertEqual(elsewhere.state, .ok)
    XCTAssertNotNil(firm.sentence(for: elsewhere.account))
  }

  // MARK: the app's own memory

  func testAccountChangedIsTheAppsOwnMemory() throws {
    let fresh = AccountMemory()
    XCTAssertNil(fresh.seen)
    XCTAssertNil(
      fresh.change(to: "nail@example.com"), "the first address the app ever sees is not a change")

    let known = fresh.noting("nail@example.com")
    XCTAssertEqual(known.seen, "nail@example.com")
    XCTAssertNil(known.change(to: "nail@example.com"))
    XCTAssertEqual(known.change(to: "other@example.com"), "nail@example.com")
    XCTAssertNil(
      known.change(to: nil), "an account that could not be read is not a sign-in as somebody else")
    XCTAssertEqual(known.noting(nil), known, "and it forgets nothing")
    XCTAssertEqual(known.noting("other@example.com").seen, "other@example.com")

    // Never the log. A line that plants an address is a record like any
    // other, and the address is read past: nothing the app folds can put a
    // second account in front of a person.
    let planted = "planted@example.com"
    let line = """
      {"v":1,"at":"2026-10-17T09:00:00Z","pid":401,"run":"r1","event":"start",\
      "command":"read","account":"\(planted)"}
      """
    let event = try Event.decode(line)
    let said = Mirror(reflecting: event).children.compactMap { $0.value as? String }
    XCTAssertFalse(said.contains(planted), "the record carries no account")
    XCTAssertEqual(
      AccountMemory(seen: planted).change(to: planted), nil,
      "and an app that had seen it would still call it the same account")
  }
}
