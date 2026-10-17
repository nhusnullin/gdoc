// The one room that asks the system to open or copy something.
//
// Four verbs: open a document in a browser, copy its link, open the note
// paired with it in Obsidian, and bring Claude Desktop to the front. Every
// NSWorkspace and NSPasteboard use in this package is here, which a source
// scan in PresenceCoreTests holds, so what this app can put in front of
// somebody is a list one person can read.
//
// Nothing here decides a URL. A document's URL is built in
// PresenceCore/LinkLine from an id held to the Drive shape, and a note's is
// built in PresenceCore/HubIndex from a path proven to be a regular `.md`
// inside the hub the person chose. This file hands the result to the system
// and reports what happened.
//
// No title and no account reaches a URL or the pasteboard. A title is
// somebody's document text and a link needs only the id, so the two are
// never in the same sentence here.
//
// Claude Desktop is activated and nothing more. A `claude:` scheme is
// registered on this machine, but what a URL under it means is not written
// down anywhere and was not measured, so there is no prefill: the panel
// brings the app to the front and the person types. The measurement is in
// `docs/v2/MEASURED.md`.

import AppKit
import Foundation
import PresenceCore
import PresenceUI

/// What the app may ask of the system, as three verbs.
///
/// A protocol so every test states what was asked without a browser opening:
/// the app hands over `SystemDesk`, and a test hands over an object that
/// writes down what it was given.
public protocol Desk: Sendable {
  /// Open a URL with whatever is registered for it.
  @discardableResult func open(_ url: URL) -> Bool
  /// Bring an application to the front, launching it if it is not up.
  @discardableResult func activate(_ bundleIdentifier: String) -> Bool
  /// Replace the pasteboard with one string.
  func copy(_ text: String)
}

/// The system itself.
public struct SystemDesk: Desk {
  public init() {}

  @discardableResult public func open(_ url: URL) -> Bool {
    NSWorkspace.shared.open(url)
  }

  /// A running copy is activated where there is one, and the bundle is
  /// opened where there is not. Neither route hands the application an
  /// argument, a file or a URL.
  @discardableResult public func activate(_ bundleIdentifier: String) -> Bool {
    let running = NSRunningApplication.runningApplications(withBundleIdentifier: bundleIdentifier)
    if let already = running.first {
      return already.activate(options: [])
    }
    guard
      let bundle = NSWorkspace.shared.urlForApplication(withBundleIdentifier: bundleIdentifier)
    else { return false }
    return NSWorkspace.shared.open(bundle)
  }

  public func copy(_ text: String) {
    let board = NSPasteboard.general
    board.clearContents()
    board.setString(text, forType: .string)
  }
}

/// The four things a row's icons do.
public struct Opener: Sendable {
  /// Claude Desktop's own identifier, read off its bundle.
  public static let claudeDesktop = "com.anthropic.claudefordesktop"

  private let desk: Desk

  public init(desk: Desk = SystemDesk()) {
    self.desk = desk
  }

  /// Open the document in Google Docs. Nothing happens where there is no id
  /// to build a URL from, which is a greyed icon in the panel and never a
  /// click that goes somewhere else.
  @discardableResult public func openDocument(_ id: String?) -> Bool {
    guard let text = LinkLine.documentURL(for: id), let url = URL(string: text) else {
      return false
    }
    return desk.open(url)
  }

  /// Copy the document's link, and hand back exactly what was copied so a
  /// test can say the pasteboard holds the plain URL and nothing else.
  @discardableResult public func copyLink(_ id: String?) -> String? {
    guard let text = LinkLine.documentURL(for: id) else { return nil }
    desk.copy(text)
    return text
  }

  /// Open the note paired with a document, or say why there is none.
  @discardableResult public func openNote(
    document id: String?, hub: String?, pairing: [String: String]
  ) -> Result<URL, NoteFault> {
    guard let hub, !hub.isEmpty else { return .failure(.noHub) }
    guard let id, let relative = pairing[id] else { return .failure(.notPaired) }
    let built = HubIndex.noteURL(hub: hub, relative: relative)
    guard case .success(let url) = built else { return built }
    guard desk.open(url) else { return .failure(.movedOrDeleted) }
    return .success(url)
  }

  /// Bring Claude Desktop to the front. The hold card's Open chat is this
  /// and the document row's branch icon is this: one activation, no prefill.
  @discardableResult public func continueWithClaude() -> Bool {
    desk.activate(Self.claudeDesktop)
  }
}

extension NoteFault {
  /// The words the greyed icon carries. The panel has no sentence of its own
  /// for a hub nobody chose: without a hub there is no pairing, and the
  /// person reads the same thing either way.
  public var tip: String {
    switch self {
    case .noHub, .notPaired: Tips.noNote
    case .movedOrDeleted, .outsideTheHub, .notMarkdown: Tips.noteMoved
    }
  }
}
