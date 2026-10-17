// The URLs the app may build, and the one it may accept.
//
// Opening a URL is the one outbound act of this app that is not a fixed argv
// at a path it resolved itself, so what a URL may be is decided here, in a
// file that opens nothing. One host for a document, one host and one path for
// a sign-in, and a document id that is a Drive id or no URL at all.
//
// The id shape is `go/internal/frontmatter/schema.go:31`'s, and for its
// reason: a path or a URL fragment must never reach a field that is later
// read back as an id. A record line is forgeable, so the id in one is checked
// here before it is put between a scheme and a slash.
//
// No title and no account ever reaches a URL. A title is somebody's document
// text, and the only thing the panel needs for a link is the id.

import Foundation

/// The links the panel has, judged.
public enum LinkLine {
  /// Where a document is opened and linked. One host, so a line of the
  /// record cannot send a click anywhere else.
  public static let documentPrefix = "https://docs.google.com/document/d/"
  /// What is opened: the document, not a preview and not an export.
  public static let documentSuffix = "/edit"
  /// The host `gdoc auth login` prints a link to.
  public static let signInHost = "accounts.google.com"
  /// The path it prints, which is `go/internal/auth/login.go:25`'s own.
  public static let signInPath = "/o/oauth2/auth"

  /// The shortest a Drive id is. Twenty is the floor the Go side holds.
  public static let idFloor = 20

  /// Whether a string is a Drive file id and nothing else.
  public static func isDocumentID(_ id: String) -> Bool {
    guard id.count >= idFloor else { return false }
    return id.allSatisfy { character in
      character.isASCII
        && (character.isLetter || character.isNumber || character == "_" || character == "-")
    }
  }

  /// The document's URL, or nothing where there is no id to build one from.
  public static func documentURL(for id: String?) -> String? {
    guard let id, isDocumentID(id) else { return nil }
    return documentPrefix + id + documentSuffix
  }

  /// The sign-in link, if what the child printed is one.
  ///
  /// The scheme, the host and the path are compared whole. A lookalike host,
  /// a credential before the host and another path on the same host are each
  /// a different page, so each is nothing.
  public static func signInURL(_ text: String) -> String? {
    let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
    guard let url = URL(string: trimmed) else { return nil }
    guard url.scheme?.lowercased() == "https" else { return nil }
    guard url.user == nil, url.password == nil else { return nil }
    guard url.host?.lowercased() == signInHost else { return nil }
    guard url.path == signInPath else { return nil }
    return trimmed
  }
}
