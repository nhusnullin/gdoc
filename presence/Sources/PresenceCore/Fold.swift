// The three walks every fold of the record shares.
//
// The record is forgeable and best-effort: a line may be there twice, a start
// may have lost its end, two month files may arrive in either order, and an
// end may have arrived whose start was in a month the binary has since
// removed. None of that may make a fold throw or lose a row, so each of the
// four sections walks the events through these first.
//
// What each section then makes of them is its own file: NOW is Now.swift,
// the live watches are LiveSession.swift, TODAY and Yesterday are Days.swift
// and DOCUMENTS is Documents.swift. HEALTH takes one fact from the record and
// the rest from two commands, and that one is Reachability.swift. Nothing
// judges here.

import Foundation

public enum Fold {
  /// What makes one record the same record as another: a start or an end and
  /// its run and call, a step and the part it names, or a hold's ending and
  /// its id. A line the writer wrote twice, or a file a person
  /// concatenated, is then one row and not two.
  private static func key(_ event: Event) -> String {
    switch event.kind {
    case .start, .end:
      return "\(event.kind.rawValue)|\(event.pairing)"
    case .step:
      // A run has four of these, so the part's own name is in the key: without
      // it a file somebody concatenated would lose three of the four.
      return "\(event.kind.rawValue)|\(event.pairing)|\(event.step ?? "")"
    case .held, .released, .expired, .landed, .dropped:
      return "\(event.kind.rawValue)|\(event.hold ?? "")"
    }
  }
}

extension Fold {
  /// The events once each, oldest first.
  ///
  /// A line the writer wrote twice, or a file a person concatenated, is one
  /// event; the index breaks a tie, so two events at the same moment keep the
  /// order the record had them in.
  public static func inOrder(_ events: [Event]) -> [Event] {
    var seen: Set<String> = []
    let once = events.filter { seen.insert(key($0)).inserted }
    return once.enumerated()
      .sorted { left, right in
        left.element.at == right.element.at
          ? left.offset < right.offset : left.element.at < right.element.at
      }
      .map(\.element)
  }

  /// The document a line is about.
  ///
  /// It is the one the command was handed, and the one a publish created
  /// where it was handed none: a publish names no document until it has
  /// made one, so `made` is that line's document everywhere the panel
  /// groups by one. Nothing else writes `made`.
  public static func subject(_ event: Event) -> String? {
    if let doc = event.doc, !doc.isEmpty { return doc }
    guard let made = event.made, !made.isEmpty else { return nil }
    return made
  }

  /// Every document's latest title, by document id.
  ///
  /// A start names no title and a held line names none either, so a card for
  /// a document the record has read before takes the title from that read.
  /// The latest wins, because that is the name the person sees in Drive.
  public static func latestTitles(_ events: [Event]) -> [String: String] {
    var titles: [String: String] = [:]
    var when: [String: Date] = [:]
    for event in events {
      guard let doc = subject(event), let title = event.title, !title.isEmpty else { continue }
      if let had = when[doc], had > event.at { continue }
      titles[doc] = title
      when[doc] = event.at
    }
    return titles
  }
}

extension Fold {
  /// Every call's latest title, by the run and the call its events pair on.
  ///
  /// A publish is handed no document, so nothing `latestTitles` holds names
  /// one: the name on the cover arrives on a step line, or on the end that
  /// says what it made, and the start carries it where gdoc read the note
  /// first. A call is named here from whichever of its own lines carried a
  /// title, the latest winning, so a card and a row read the same whichever
  /// of those lines the record still holds.
  ///
  /// The key is the pairing and not the run alone, because the record stamps
  /// one run on every event of a process: a chat session is one run and a
  /// dozen calls, and the title of one of its writes is not the name of
  /// another.
  public static func latestCallTitles(_ events: [Event]) -> [String: String] {
    var titles: [String: String] = [:]
    var when: [String: Date] = [:]
    for event in events {
      guard let title = event.title, !title.isEmpty else { continue }
      if let had = when[event.pairing], had > event.at { continue }
      titles[event.pairing] = title
      when[event.pairing] = event.at
    }
    return titles
  }
}
