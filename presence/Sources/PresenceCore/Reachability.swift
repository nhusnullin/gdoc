// Whether Google answered the last command that tried.
//
// The one fact SETTINGS takes from the record, and the only thing this app can
// know about the network: it opens no connection itself, so the answer is
// what gdoc wrote down. `stop: "network"` is gdoc's own word for a request
// that did not get there, and a later command that worked is the answer to
// it.
//
// It is here and not in the panel's own Facts because every fold of the
// record lives in this module, beside Now, LiveSession, Days and Documents.
// This one hands back a plain Bool, and the panel words it.

import Foundation

/// The record's own word about reaching Google.
public enum Reachability {

  /// The commands whose end says nothing about Google.
  ///
  /// `auth logout` removes a file and opens no connection at all, so a
  /// logout that worked is no answer to a reply that could not get out.
  /// `update` reads GitHub and never Google, so its `stop: "network"` is
  /// GitHub unreachable and a row naming Google would be a guess. Every
  /// other reporting command carries a Google request:
  /// testTheNetworkRowIsTheRecordsOwnWord.
  static let offGoogle: Set<String> = ["auth logout", "update"]

  /// The one failure that is Google answering. `not_found` is Drive or Docs
  /// saying 404 for the id a command named, so the request got there; the
  /// other six words are either the wire itself or something gdoc refused
  /// before it made a request, and `auth` covers a computer with no token,
  /// which never reaches Google either.
  static let answered = "not_found"

  /// What the newest end that said anything says about reaching Google, and
  /// nothing where no command has said anything yet:
  /// testTheNetworkRowIsTheRecordsOwnWord.
  public static func latest(in events: [Event]) -> Bool? {
    for event in Fold.inOrder(events).reversed() {
      guard event.kind == .end, let ok = event.ok else { continue }
      guard !offGoogle.contains(event.command ?? "") else { continue }
      if event.stop == "network" { return false }
      if ok || event.stop == answered { return true }
    }
    return nil
  }
}
