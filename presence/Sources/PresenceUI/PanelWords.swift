// Every word the panel says that is not a row's own value.
//
// Three tables and one struct: the tooltip on each icon, the sentences that
// belong to no row, the reading of a note that would not open, and the
// phrases each card and each log row is built out of. They are here rather
// than beside the models so the shapes and the English are two files: a
// wording change touches this one alone.
//
// Nothing here reads the machine or judges a document. Every one of them is
// handed a fold, a moment and the facts, and hands back a string.

import Foundation
import PresenceCore

/// The words on an icon, which are the whole of its label. Each is the
/// screens file's own `data-tip`.
public enum Tips {
  public static let openDoc = "Open in Google Docs"
  public static let noDoc = "No Google Doc yet"
  public static let openNote = "Open the note in Obsidian"
  public static let noNote = "No note is paired with this document"
  /// Two notes name this document, which the app may not pick between. It
  /// is somebody's own mistake and the words say only what was found.
  public static let twoNotes = "Two notes name this document"
  /// The note is not a file at that path any more. It is a fact and never a
  /// complaint about the person who moved it.
  public static let noteMoved = "Note moved or deleted"
  public static let openChat = "Continue with Claude"
  public static let noExtension = "Install the Desktop extension in Settings"
  public static let copyLink = "Copy link"
  /// The link icon's own off word, which the screens file writes beside
  /// "No Google Doc yet": there is no link to copy for the same reason
  /// there is no document to open.
  public static let noLink = "No link yet"
  public static let copied = "Copied"
  /// The fifth icon of a document row, in its two states. They are the words
  /// the menu carried until the third look, on the icon that replaced it.
  public static let pin = "Pin"
  public static let unpin = "Unpin"
  public static let dismiss = ButtonWord.dismiss.rawValue
  public static let cannotStop = "This watch is not one gdoc presence can stop"
  public static let signIn = "Sign in with Google"
  public static let signOut = ButtonWord.signOut.rawValue
  public static let cancel = "Cancel"
  public static let copySignIn = "Copy sign-in link"
  public static let installExtension = "Install extension"
  /// The header's update glyph while nothing newer is known. The press runs
  /// `gdoc update`, which asks GitHub itself, so the words are about asking
  /// and not about a release nobody has heard of.
  public static let checkUpdate = "Check for a new version"
  /// The same glyph for the second after a run that found nothing to
  /// install. It is what the run answered and never a promise about later.
  public static let newestRelease = "This is the newest release"

  /// The glyph where `help --json` listed a newer stable release.
  public static func updateTo(_ version: String) -> String {
    "Update to \(version)"
  }
}

/// The word of a document row's own menu, which is one word since the third
/// look: the pin is the row's fifth icon and Hide is what is left.
public enum Menus {
  /// The one way out of the list. The way back is gdoc doing something on
  /// the document again, which the guide says and no menu word can.
  public static let hide = "Hide from the list"
}

extension NoteFault {
  /// The words the greyed icon carries. Each is a fact about what was found
  /// and none of them is a judgement about the note or about the person who
  /// moved it.
  public var tip: String {
    switch self {
    case .notPaired: Tips.noNote
    case .twoNotes: Tips.twoNotes
    case .movedOrDeleted, .notMarkdown: Tips.noteMoved
    }
  }
}

/// The sentences the panel writes that are nobody's row.
public enum Sentences {
  /// DOCUMENTS with nothing in it, which is the first open of a machine
  /// that has signed in and done nothing yet.
  public static let noDocuments =
    "Publish a note or review a document from Claude Code and it will appear here."
  /// DOCUMENTS while a write that has named no document yet is running.
  public static let nothingYet = "nothing yet"
  /// A token with no address in it. Whose account it is decides nothing, so
  /// this is a fact and not a fault.
  public static let accountNotRead = "signed in, account not read"
  public static let loginItemApproval = "macOS is waiting for you to allow it in Login Items"
  /// What joins the parts of one line: the middle dot with a space either
  /// side. The stuck card breaks its title on it, so the two halves of
  /// "Export stopped · Chapter 4" are the card's two lines.
  public static let separator = " · "
  /// A document row with no note behind it. It named the hub until the
  /// third look, and there is no hub to name.
  public static let noNoteYet = "no note"

  /// A path as a row writes it: with the home directory as a tilde, or
  /// whole where it is somewhere else.
  ///
  /// The home is handed in rather than read here, which is the rule for
  /// every file in this target, and it is why a render of a scene says the
  /// same thing on every machine. A path outside that home is written in
  /// full: a tilde there would name a file that is not the one found.
  public static func shortPath(_ path: String, home: String) -> String {
    guard !home.isEmpty, path == home || path.hasPrefix(home + "/") else { return path }
    return "~" + path.dropFirst(home.count)
  }
}

/// Every sentence the panel builds out of one fold and one set of facts.
///
/// It is a struct rather than a pile of functions because each of them needs
/// the same three things: the moment, the calendar and the facts.
struct Phrases {
  let facts: Facts
  let now: Date
  let calendar: Calendar

  // MARK: the cards

  func holdCard(_ hold: PresenceCore.HoldCard) -> HoldCardModel {
    let what = Words.verb(hold.command)?.heading ?? hold.command
    let sub = sentence([what, hold.title, hold.rule.map { "rule \($0)" }])

    if hold.ranOut {
      return HoldCardModel(
        title: "Ran out · \(clock(hold.until ?? hold.held))",
        sub: sub,
        quiet: "Nothing was written. The text is still in the chat.",
        question: "Dismiss this card?",
        consequence: "Nothing is written either way.")
    }

    return HoldCardModel(
      title: "Waiting for you · \(Words.duration(now.timeIntervalSince(hold.held)))",
      sub: sub,
      quiet: postsLine(),
      question: "Dismiss this card?",
      consequence: "The hold stays in the chat until it runs out. Nothing is written.")
  }

  /// Where the words are, and under whose name they would be posted. The
  /// account is said because a reply carries it into somebody's document;
  /// it is not a gate, which is why no rule reads it.
  private func postsLine() -> String {
    let who = facts.account.map { "Posts from \($0)" } ?? "Posts"
    let mark = facts.mark.isEmpty ? "" : " with the \(facts.mark) mark"
    return "\(who)\(mark). The text is in the chat."
  }

  func stuckCard(_ write: StuckWrite) -> StuckCardModel {
    let what = Words.verb(write.command)?.heading ?? write.command
    return StuckCardModel(
      title: sentence(["\(what) stopped", write.title]),
      reason: stuckReason(write))
  }

  /// Where the reason is, by the record's own `via`. All three the record
  /// writes are here: a run from Claude Code left nothing in a terminal, so
  /// sending a person to one would be a sentence that is not true.
  /// testAStoppedWriteNamesThePlaceItRanFrom.
  private static let places: [String: String] = [
    "terminal": "terminal",
    "chat": "Desktop chat",
    "code": "Claude Code session",
  ]

  /// What the world looks like after a write stopped, in the words of the
  /// thing that write was handed: a publish was handed a note, and the
  /// three writes that are one request each were handed a document.
  ///
  /// A restyle is a dozen batches, a propose is a suggestion and then its
  /// own comment, and an export writes a file for every tab, so none of
  /// the three can say nothing was written: a restyle that stopped at the
  /// twelfth batch left a part-styled document. internal/restyle holds
  /// that rule on gdoc's own side, where
  /// TestAFailedRequestNeverSaysTheDocumentIsAsItWas pins it, and the
  /// panel is not the place it stops being true:
  /// testAStoppedWriteClaimsTheWorldOnlyWhereItCan.
  private static let worlds: [String: String] = [
    "publish": "Nothing was created; the note is unchanged.",
    "reply": "Nothing was written; the document is unchanged.",
    "annotate": "Nothing was written; the document is unchanged.",
    "withdraw": "Nothing was written; the document is unchanged.",
  ]

  /// The world clause for one stopped write, or none where the record does
  /// not carry the claim.
  ///
  /// `network` is gdoc's word for a request that never got an answer, which
  /// internal/emit says is a dial, a handshake or a deadline: the first two
  /// never reached Google and the last may have been taken and applied, and
  /// the record cannot tell them apart. So a request with no answer behind
  /// it is a write nobody can call undone, whatever the command was.
  private static func world(_ write: StuckWrite) -> String? {
    guard write.stop != "network" else { return nil }
    return worlds[write.command]
  }

  /// One sentence, under eighty characters: the cause with its place, then
  /// the state of the world where the record carries it. That is scene r2's
  /// rule, and the halves are all the record's: `stop` is one of gdoc's
  /// seven words and `line` is where the cause sits.
  ///
  /// A run whose process went away wrote neither, and so does a code this
  /// version has never seen: then the panel says where the reason is and
  /// nothing else, because the one thing it must not do is invent a cause.
  /// The card's own title already says the write stopped, so a sentence
  /// that said it again would be the third place one screen says one thing.
  ///
  /// The moment is on the card, because nothing else on the screen carries
  /// it: a write that stopped is one card and no row anywhere, and NOW
  /// keeps such a card for fourteen days:
  /// testACardWithNoRowBehindItCarriesTheMoment.
  private func stuckReason(_ write: StuckWrite) -> String {
    let place = Self.places[write.via ?? ""] ?? "terminal"
    let look = "The \(place) that ran it has the reason."
    guard let cause = Words.cause(write.stop, line: write.line) else {
      guard write.cause == .stopped else { return look }
      return "Started \(clockInSentence(write.at)). \(look)"
    }
    guard let world = Self.world(write) else { return "\(Words.opening(cause)). \(look)" }
    return "\(Words.opening(cause)). \(world)"
  }

  /// A running job, with the part it is on.
  ///
  /// A publish is handed no title, so the card is "Publishing" alone until
  /// the render names the cover; the steps are the four gdoc always takes,
  /// and the record says which of them began last. The ring is that part's
  /// place in the list, which is a position and not a forecast.
  func jobCard(_ run: NowRun) -> JobCardModel {
    let place = run.index.flatMap { index in run.count.map { "step \(index) of \($0)" } }
    let title = titled(Words.running(run.command) ?? run.command, run.title)
    return JobCardModel(
      prefix: title.prefix,
      name: title.name,
      sub: sentence([place, "started \(clock(run.started))", Words.via(run.via)]),
      progress: run.progress,
      steps: steps(run))
  }

  /// The parts of a publish as rows: what the record says began last is
  /// running, everything before it happened, everything after it has not.
  /// A run the record named no part of draws no rows at all.
  private func steps(_ run: NowRun) -> [JobStep] {
    guard let now = run.step, let at = Words.steps.firstIndex(where: { $0.name == now })
    else { return [] }
    return Words.steps.enumerated().map { place, words in
      if place < at { return JobStep(title: words.done, state: .done) }
      if place == at { return JobStep(title: words.running, state: .running(nil)) }
      return JobStep(title: words.waiting, state: .waiting)
    }
  }

  /// A watch's one line of facts.
  ///
  /// A watch that is not getting through says so and nothing else: the row
  /// is one line at 320 pt, and since and quiet pushed the one thing that
  /// is wrong off the end of it. Scene r8 is that line, and it carries no
  /// word for why, because the record has none.
  func liveRow(_ session: LiveSession) -> LiveRowModel {
    // The watch's own ending and never the machine's word: a token short a
    // scope stops one watch and leaves another polling, and a fact about
    // the computer written over every row would take since, quiet and the
    // two counts off a watch that is still live and still has its Stop:
    // testAnOpenWatchKeepsItsFactsWhenAnotherOneLostTheSignIn.
    let expired = session.stoppedBySignIn ? "sign-in expired" : nil
    let failed = session.lastPollFailed.map { moment in
      // A poll Google refused the token on is the end of the watch and not
      // a watch in trouble: the skill's next wait cannot start without a
      // sign-in, so nothing is polling that document any more. "Paused"
      // would be the panel saying it will carry on, which is a forecast and
      // nobody has measured whether gdoc resumes from the same cursor.
      // That is scene e9.
      //
      // The word is the title's, as it is on a stuck card: the row says
      // "Live stopped" above and the moment and the cause below, rather
      // than "stopped" twice on one row.
      session.lastPollStop == Live.signInStop
        ? clock(moment) : "last poll failed \(clock(moment))"
    }

    let parts: [String?] =
      failed == nil && expired == nil
      ? [
        "since \(clock(session.since))",
        "quiet \(Words.duration(now.timeIntervalSince(session.quietSince)))",
        count(session.replied, of: "reply"),
        count(session.proposed, of: "propose"),
      ]
      : [failed, expired ?? Words.cause(session.lastPollStop)]

    // A watch that is over says so in its title, which is the word
    // DOCUMENTS uses for it. The row kept opening with "Live" over a
    // sub-line that said the watch had stopped:
    // testE9SaysTheWatchIsStoppedInEverySectionAndDrawsNoStop.
    let title = titled(
      session.isOpen ? "Live" : "Live stopped", session.title ?? session.doc)
    return LiveRowModel(
      prefix: title.prefix,
      name: title.name,
      sub: sentence(parts),
      stopped: !session.isOpen)
  }

  /// How many writes of one kind a watch has to its name, in the word the
  /// rest of the panel uses for that write.
  ///
  /// A count of nothing is left out. The word is the command's own past,
  /// which is the word DOCUMENTS uses for that write: a third word here
  /// would be a second vocabulary for one thing, which is what the
  /// specification's own sentence says ("since, replied and proposed
  /// counts"):
  /// testAWatchCountsItsWritesInTheWordsForThem.
  private func count(_ many: Int, of command: String) -> String? {
    guard many > 0 else { return nil }
    return "\(Words.verb(command)?.past ?? command) \(many)"
  }

  // MARK: the lists

  /// A document row's second line: what happened to it, and where its note
  /// is. The state of an open action is one word, because the record has
  /// none and a person reading the list wants the state and not the verb.
  func documentSub(_ row: DocumentRow) -> String {
    let note = facts.note(for: row.doc).map { Sentences.shortPath($0, home: facts.home) }
    return sentence([state(row), note ?? Sentences.noNoteYet])
  }

  /// What the row says about the last thing that happened, or nothing at
  /// all for a row that is only in the list because it is pinned: the pin is
  /// not an action, and a sentence about one would be invented.
  private func state(_ row: DocumentRow) -> String? {
    switch row.state {
    case .hold: return "hold"
    case .running: return (Words.running(row.command) ?? "running").lowercased()
    case .live: return "live"
    case .liveStopped: return "live stopped \(clockInSentence(row.at))"
    case .done:
      return "\(Words.verb(row.command)?.past ?? row.command) \(clockInSentence(row.at))"
    case .failed: return "stopped \(clockInSentence(row.at))"
    case .unknown: return "outcome unknown · \(clockInSentence(row.at))"
    case .pinnedOnly: return nil
    }
  }

  // MARK: the two small pieces

  func clock(_ moment: Date) -> String {
    Words.time(moment, now: now, calendar: calendar)
  }

  /// The same moment inside a sentence.
  ///
  /// Yesterday is the one word the clock gives back that is not a proper
  /// noun: a weekday and a month keep their capital between other words,
  /// and "live stopped Yesterday" reads as a name. It stays capital where
  /// it stands alone in the value column, which is where `clock` is used.
  func clockInSentence(_ moment: Date) -> String {
    let written = clock(moment)
    return written == "Yesterday" ? "yesterday" : written
  }

  func sentence(_ parts: [String?]) -> String {
    parts.compactMap { $0 }.filter { !$0.isEmpty }.joined(separator: Sentences.separator)
  }

  /// A row's title as the two pieces it is drawn in: the verb in front, and
  /// the document's name.
  ///
  /// The verb is drawn whole and the name is measured on its own, so a verb
  /// as long as "Live stopped" does not eat the name behind it. Where the
  /// record names no document the verb is the whole title and there is no
  /// prefix: a line of one word is measured as any other line of one word.
  func titled(_ verb: String, _ name: String?) -> (prefix: String?, name: String) {
    guard let name, !name.isEmpty else { return (nil, verb) }
    return (verb, name)
  }
}
