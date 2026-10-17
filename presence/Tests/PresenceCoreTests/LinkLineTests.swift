// The two URLs the app may build, and the one it may accept.
//
// Every value here is a literal. A URL the app opens is the one outbound act
// of this app that is not a fixed argv, so what it may be is stated in this
// file rather than inferred from a parser's behaviour.

import Foundation
import XCTest

@testable import PresenceCore

final class LinkLineTests: XCTestCase {

  /// A Drive id, the shape `go/internal/frontmatter/schema.go:31` holds.
  private static let id = "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd"

  func testTheDocsURLIsBuiltFromAValidIdOnly() {
    XCTAssertEqual(
      LinkLine.documentURL(for: Self.id),
      "https://docs.google.com/document/d/1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd/edit")

    XCTAssertNil(LinkLine.documentURL(for: nil), "no id, no URL")
    XCTAssertNil(LinkLine.documentURL(for: ""), "an empty id is not an id")
    XCTAssertNil(LinkLine.documentURL(for: "short"), "under twenty characters is not a Drive id")
    XCTAssertNil(
      LinkLine.documentURL(for: "../../etc/passwd0123456789abcd"),
      "a path is never an id, whatever its length")
    XCTAssertNil(
      LinkLine.documentURL(for: "1AbCdEfGhIjKlMnOpQrSt/edit?x=1"),
      "a URL fragment is never an id")
    XCTAssertNil(
      LinkLine.documentURL(for: "1AbCdEfGhIjKlMnOpQr Tt0123456789"),
      "a space is not in the shape")

    // Exactly twenty is the floor, and nineteen is not.
    XCTAssertTrue(LinkLine.isDocumentID(String(repeating: "a", count: 20)))
    XCTAssertFalse(LinkLine.isDocumentID(String(repeating: "a", count: 19)))
    XCTAssertTrue(LinkLine.isDocumentID("Ab-9_0123456789abcdefgh"), "dash and underscore are in")
  }

  /// The sign-in link is taken from a child's output, so it is judged before
  /// it is opened: one scheme, one host, one path, and no credential in it.
  func testOnlyGooglesSignInURLIsAccepted() {
    XCTAssertEqual(LinkLine.signInHost, "accounts.google.com")
    XCTAssertEqual(LinkLine.signInPath, "/o/oauth2/auth")

    let real =
      "https://accounts.google.com/o/oauth2/auth?client_id=1&redirect_uri=http%3A%2F%2F127.0.0.1%3A1"
    XCTAssertEqual(LinkLine.signInURL(real), real, "the link the login prints, query and all")
    XCTAssertEqual(
      LinkLine.signInURL("  " + real + "  "), real, "a line with its spaces still trimmed")

    for refused in [
      "http://accounts.google.com/o/oauth2/auth",
      "https://accounts.google.com.evil.example/o/oauth2/auth",
      "https://accounts.google.com@evil.example/o/oauth2/auth",
      "https://evil.example/o/oauth2/auth",
      "javascript:alert(1)",
      "https://accounts.google.com/v2/auth?client_id=1",
      "https://accounts.google.com/o/oauth2/auth/../../wherever",
      "not a URL at all",
      "",
    ] {
      XCTAssertNil(LinkLine.signInURL(refused), "\(refused) is not a sign-in link")
    }
  }
}
