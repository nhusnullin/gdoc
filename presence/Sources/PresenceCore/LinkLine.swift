// The URLs the app may build, and the two it may accept.
//
// Opening a URL is the one outbound act of this app that is not a fixed argv
// at a path it resolved itself, so what a URL may be is decided here, in a
// file that opens nothing. One host for a document, one host and one path for
// a sign-in, one path of Claude Desktop's own scheme for a session, and a
// document id that is a Drive id or no URL at all. A third host is the Drive
// page of the firm's mark, which arrives as the one address the binary's own
// help answer names for it and is held to its one shape here before a press
// can open it.
//
// The session link is built and not opened. Measured 2026-10-18, MEASURED.md:
// `claude://code/continue` activates Claude Desktop from outside the app and
// switches nothing, so Open chat keeps the activation it has. The builder and
// its test stay for the day a form switches, because by then the record
// already carries the id.
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
  /// Where the firm's mark is opened: the file's own page in Drive. The app
  /// holds no Drive id, so the address arrives whole from the binary's help
  /// and this is the only shape of it the app will take.
  public static let markPagePrefix = "https://drive.google.com/file/d/"
  /// What is opened: the picture's page, not a download and not a preview.
  public static let markPageSuffix = "/view"

  /// The host `gdoc auth login` prints a link to.
  public static let signInHost = "accounts.google.com"
  /// The path it prints, which is `go/internal/auth/login.go:25`'s own.
  public static let signInPath = "/o/oauth2/auth"

  /// The one link of Claude Desktop's own scheme the app may build: the Code
  /// tab, continuing a named session. The three forms the bundle carries were
  /// read out of it on 2026-10-18 and are in MEASURED.md; this is the only one
  /// that names a session.
  public static let codeSessionPrefix = "claude://code/continue?session="

  /// Where the link says it came from, which is this app and never the
  /// desktop action the bundle's own links name.
  public static let codeSessionSource = "&source=gdoc"

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

  /// The page of the firm's mark, if what the help printed is one.
  ///
  /// The prefix and the suffix are compared whole and with their case, and
  /// what lies between them is a Drive id or nothing. The answer is built
  /// again from the parts rather than handed back as it arrived, so nothing
  /// can ride between them into the system's hands.
  public static func markPageURL(_ text: String?) -> String? {
    guard let text else { return nil }
    let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
    guard trimmed.hasPrefix(markPagePrefix), trimmed.hasSuffix(markPageSuffix) else { return nil }
    let id = String(trimmed.dropFirst(markPagePrefix.count).dropLast(markPageSuffix.count))
    guard isDocumentID(id) else { return nil }
    return markPagePrefix + id + markPageSuffix
  }

  /// Whether a string is a Claude Code session id and nothing else.
  ///
  /// A session id is a UUID: measured 2026-10-18 from this session's own
  /// `CLAUDE_CODE_SESSION_ID`. The shape is held here for the reason the
  /// Drive id's is: the value arrives in a line of the record, which any
  /// process with the owner's rights can write, and a query or a path in it
  /// must never reach a URL somebody clicks.
  public static func isSessionID(_ session: String) -> Bool {
    UUID(uuidString: session) != nil
  }

  /// The link that continues the Claude Code session a document was last
  /// worked on from, or nothing where the record named no session or named
  /// something that is not one.
  public static func codeSessionURL(for session: String?) -> String? {
    guard let session, isSessionID(session) else { return nil }
    return codeSessionPrefix + session + codeSessionSource
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
