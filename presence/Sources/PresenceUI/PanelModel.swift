// The whole panel as values, before anything is drawn.
//
// The view draws this and decides nothing, which is the rule M23 started
// with and this milestone keeps: every word a person can read is a value a
// test can state, and a render adds only whether the pixels came out.
//
// It is built from three things and nothing else: the events, the facts
// about the machine, and what the person did to the panel. The four folds
// are taken here so the shell hands over a list of events and never four
// lists that could disagree with each other.
//
// Two words are this file's own, and both are the reading the app is allowed
// to make. The state of a document is said in one word (hold, running, live,
// live stopped), because the record has no word for it. And a write that
// stopped with a cause this version has no word for carries one sentence
// that names the moment and where the reason is, because a panel that
// invented a cause would be read as gdoc's own answer.
//
// Nothing here predicts anything, which is principle 5. The job card's ring
// is the part's place in a list of four, which is where the run is and not
// how much of it is left, and no row says when something will finish.

import Foundation
import PresenceCore

/// What pressing something in the panel means. The view never knows what a
/// kind does: it calls `PanelActions.perform` with the kind and the thing it
/// was about, and Tasks 17 to 21 give that function a body.
public enum ActionKind: String, Sendable, Equatable, CaseIterable {
  case openDoc
  case openNote
  case openChat
  case copyLink
  case dismiss
  case stopLive
  case signIn
  case signOut
  case cancelSignIn
  case copySignInLink
  case update
  case installExtension
  case openLoginItems
  case toggleLoginItem
  case quit
  /// The safe half of every confirm, which does nothing but put the
  /// question away.
  case keep
}

/// One pressable thing, as a value: which icon, the words a pointer gets,
/// what state it is in, and what it is about.
public struct ActionModel: Sendable, Equatable, Identifiable {
  public let kind: ActionKind
  public let icon: Icon
  public let tip: String
  public let state: ActionState
  /// The document, hold or run the action is about, where it is about one.
  public let target: String?
  /// Held open with no pointer, which a still of the panel needs.
  public let forcedTooltip: Bool

  public init(
    _ kind: ActionKind, icon: Icon, tip: String, state: ActionState = .live,
    target: String? = nil, forcedTooltip: Bool = false
  ) {
    self.kind = kind
    self.icon = icon
    self.tip = tip
    self.state = state
    self.target = target
    self.forcedTooltip = forcedTooltip
  }

  /// What a forced tooltip names: the thing and the action on it.
  public var id: String { "\(target ?? "")|\(kind.rawValue)" }
}

/// What the panel does when something is pressed.
///
/// One function and one toggle, so a view that gains a button gains no
/// knowledge of what the button reaches. The shell fills these in; every
/// test and every still leaves them alone.
public struct PanelActions {
  public var perform: (ActionKind, String?) -> Void
  public var toggle: (PanelList) -> Void

  public init(
    perform: @escaping (ActionKind, String?) -> Void = { _, _ in },
    toggle: @escaping (PanelList) -> Void = { _ in }
  ) {
    self.perform = perform
    self.toggle = toggle
  }
}

/// One row of NOW, in the four shapes it has.
public enum NowItem: Sendable, Equatable, Identifiable {
  case hold(id: String, card: HoldCardModel, asking: Bool)
  case stuck(id: String, card: StuckCardModel, actions: [ActionModel])
  case job(id: String, card: JobCardModel)
  case live(id: String, row: LiveRowModel, stop: ActionModel?, pending: Bool)

  public var id: String {
    switch self {
    case .hold(let id, _, _): return "hold|\(id)"
    case .stuck(let id, _, _): return "stuck|\(id)"
    case .job(let id, _): return "job|\(id)"
    case .live(let id, _, _, _): return "live|\(id)"
    }
  }
}

/// One document's row.
public struct DocumentRowModel: Sendable, Equatable, Identifiable {
  public let doc: String
  public let title: String
  public let sub: String
  public let glyph: GlyphKind
  public let tone: GlyphTone
  /// The document is in the trash. The v1 record never says so, so this is
  /// false until M24b.
  public let gone: Bool
  public let actions: [ActionModel]

  public var id: String { doc }
}

/// One row of TODAY or of Yesterday.
public struct LogRowModel: Sendable, Equatable, Identifiable {
  public let key: String
  public let glyph: GlyphKind
  public let tone: GlyphTone
  public let title: String
  public let value: String
  public let sub: String?

  public var id: String { key }
}

/// A question asked in place of a row: the words, what follows from them,
/// and the word on the loud half of the pair. The quiet half is always Keep.
public struct QuestionModel: Sendable, Equatable {
  public let title: String
  public let consequence: String
  public let danger: ButtonWord
}

/// One HEALTH row, with its words and what sits on its right.
public struct HealthRowModel: Sendable, Equatable, Identifiable {
  public let kind: HealthRowKind
  public let value: String
  public let glyph: GlyphKind
  public let tone: GlyphTone
  /// The ring in the glyph's place, which a sign-in in flight and an update
  /// that is still running draw.
  public let waiting: Bool
  public let quiet: String?
  public let actions: [ActionModel]
  public let isOn: Bool?
  /// Pressing the row itself, which Quit and the Login Items row do.
  public let select: ActionKind?
  /// The question this row has become, for the three seconds it is asked.
  public let question: QuestionModel?

  public var id: String { kind.rawValue }

  public init(
    _ kind: HealthRowKind, value: String = "", glyph: GlyphKind = .blank,
    tone: GlyphTone = .quiet, waiting: Bool = false, quiet: String? = nil,
    actions: [ActionModel] = [], isOn: Bool? = nil, select: ActionKind? = nil,
    question: QuestionModel? = nil
  ) {
    self.kind = kind
    self.value = value
    self.glyph = glyph
    self.tone = tone
    self.waiting = waiting
    self.quiet = quiet
    self.actions = actions
    self.isOn = isOn
    self.select = select
    self.question = question
  }
}

/// DOCUMENTS: its count, its rows, and how much of them is on the screen.
public struct DocumentsSectionModel: Sendable, Equatable {
  public let count: String?
  public let rows: [DocumentRowModel]
  public let collapsed: Int
  public let isOpen: Bool
  /// The one line a panel with no documents says instead.
  public let empty: String?
  /// Whether the four labels are drawn under the first row.
  public let showsStrip: Bool
}

/// TODAY, or Yesterday opened.
public struct LogSectionModel: Sendable, Equatable {
  public let rows: [LogRowModel]
  public let collapsed: Int
  public let isOpen: Bool
  public let empty: String?
}

/// Yesterday: one chevron row that opens in place.
public struct YesterdayModel: Sendable, Equatable {
  public let summary: String
  public let isOpen: Bool
  public let rows: [LogRowModel]
}

/// The address said once at its full size, for five seconds after a sign-in.
public struct BannerModel: Sendable, Equatable {
  public let account: String
}

/// The sections of the panel, in the order they are drawn.
public enum PanelSection: Sendable, Equatable, Identifiable {
  case now([NowItem])
  case documents(DocumentsSectionModel)
  case today(LogSectionModel)
  case yesterday(YesterdayModel)
  case health([HealthRowModel])
  case signedIn(BannerModel)

  /// The label over the section, as the screens file writes it.
  public var label: String {
    switch self {
    case .now: return "Now"
    case .documents: return "Documents"
    case .today: return "Today"
    case .yesterday: return "Yesterday"
    case .health: return "Health"
    case .signedIn: return "Signed in"
    }
  }

  public var id: String { label }
}

/// The whole panel.
public struct PanelModel: Sendable, Equatable {
  /// The mark in effect, which the header and the empty state draw.
  public let mark: String
  /// The address in the header, or nothing.
  public let account: String?
  /// The centred first-run words, where there is nothing else at all.
  public let empty: String?
  /// The one line the app says about itself: the bundle on the disk, or a
  /// record from a gdoc this app cannot read.
  public let notice: String?
  public let sections: [PanelSection]
  /// The one action whose words are held open with no pointer.
  public let forcedTooltip: String?

  /// The sign-in URL under the Account row, where a sign-in is in flight.
  public let signInURL: String?

  public var now: [NowItem] {
    for section in sections {
      if case .now(let items) = section { return items }
    }
    return []
  }

  public var health: [HealthRowModel] {
    for section in sections {
      if case .health(let rows) = section { return rows }
    }
    return []
  }

  public var documents: DocumentsSectionModel? {
    for section in sections {
      if case .documents(let model) = section { return model }
    }
    return nil
  }

  public var today: LogSectionModel? {
    for section in sections {
      if case .today(let model) = section { return model }
    }
    return nil
  }

  /// Every sentence the panel writes under a title, which the render tests
  /// hold to the stuck card's eighty characters.
  public var reasons: [String] {
    now.compactMap {
      if case .stuck(_, let card, _) = $0 { return card.reason } else { return nil }
    }
  }
}

extension PanelModel {
  /// The events, the machine and the person into one panel.
  ///
  /// - Parameters:
  ///   - now: the moment the panel is drawn at.
  ///   - calendar: the zone and locale the clock is read in, injected so a
  ///     still renders the same on any machine.
  ///   - alive: the kernel's answer for one pid.
  public static func build(
    events: [Event], facts: Facts, ui: UIState = UIState(), now: Date, calendar: Calendar,
    alive: (Int32) -> Bool
  ) -> PanelModel {
    let nowFold = Now.fold(events, now: now, dismissed: ui.dismissed, alive: alive)
    let sessions = Live.sessions(events, now: now, alive: alive)
    let docs = Documents.fold(events, now: now, calendar: calendar, alive: alive)
    // The cards NOW is keeping are not rows TODAY writes as well: the
    // horizon and the dismissals are the NOW fold's, so it is folded first
    // and its keys are handed over.
    let days = Days.fold(
      events, now: now, calendar: calendar, carded: Set(nowFold.stuck.map(\.key)), alive: alive)

    let words = Phrases(facts: facts, now: now, calendar: calendar)
    let items = nowItems(nowFold, sessions: sessions, facts: facts, ui: ui, words: words)
    let documents = documentsSection(
      docs, facts: facts, ui: ui, words: words, busy: !items.isEmpty)
    let today = todaySection(days, ui: ui, words: words)
    let yesterday = yesterdaySection(days, ui: ui, words: words)
    let health = healthRows(
      facts: facts, ui: ui, now: now, stopping: stopping(items, sessions: sessions))

    // HEALTH is last, where a person looks for it, except on the three
    // screens where it is the whole of what the panel is for: a machine
    // with no token at all, a sign-in in flight, which lives in the Account
    // row with its countdown and its address, and the three seconds a
    // question is asked in it. A question or a countdown that scrolled out
    // of sight is one nobody answers.
    let pinned =
      facts.auth == .signedOut || facts.signIn != nil || ui.isAsking(.signOut, now: now)

    var sections: [PanelSection] = []
    if pinned { sections.append(.health(health)) }
    if !items.isEmpty { sections.append(.now(items)) }
    sections.append(.documents(documents))
    // An empty TODAY under a pinned HEALTH is a label over one grey word,
    // between the person and the row they opened the panel for. NOW is
    // already dropped when it is empty, and this is the same rule.
    if !(pinned && today.rows.isEmpty) { sections.append(.today(today)) }
    if let yesterday { sections.append(.yesterday(yesterday)) }
    if !pinned { sections.append(.health(health)) }
    if let account = facts.account,
      SignedInBanner.isShowing(signedInAt: facts.signedInAt, now: now)
    {
      sections.append(.signedIn(BannerModel(account: account)))
    }

    let bare =
      items.isEmpty && documents.rows.isEmpty && today.rows.isEmpty && yesterday == nil

    return PanelModel(
      mark: facts.mark,
      account: facts.account,
      empty: bare && facts.auth == .signedOut ? EmptyState.firstRun : nil,
      notice: notice(facts),
      sections: sections,
      forcedTooltip: ui.forcedTooltip,
      signInURL: facts.signIn?.url)
  }

  // MARK: NOW

  private static func nowItems(
    _ fold: NowFold, sessions: [LiveSession], facts: Facts, ui: UIState, words: Phrases
  ) -> [NowItem] {
    var items: [NowItem] = []

    // The sign-in card is first of everything but a hold: a watch that
    // cannot reach Google is paused until somebody signs in, and that is
    // the one thing the person can do about any of the rows under it.
    let expired = signInCard(facts: facts, sessions: sessions)

    for entry in fold.entries {
      switch entry {
      case .hold(let hold):
        items.append(
          .hold(
            id: hold.id, card: words.holdCard(hold),
            asking: ui.isAsking(.dismissHold(hold.id), now: words.now)))
      case .stuck(let write):
        items.append(
          .stuck(
            id: write.key, card: words.stuckCard(write),
            actions: stuckActions(write, facts: facts, ui: ui, now: words.now)))
      case .job(let run):
        items.append(.job(id: run.key, card: words.jobCard(run)))
      case .watch:
        // A watch is drawn from its session rather than from its one poll,
        // because what a person calls a live watch is the whole stretch.
        continue
      }
    }

    if let expired {
      let holds = items.prefix { if case .hold = $0 { return true } else { return false } }
      items.insert(expired, at: holds.count)
    }

    for session in sessions where session.isOpen {
      items.append(
        .live(
          id: session.doc, row: words.liveRow(session),
          stop: stopAction(session, facts: facts, ui: ui),
          pending: ui.pendingStop.contains(session.doc)))
    }
    return items
  }

  /// The card a token that stopped working raises, which is the one card in
  /// NOW that comes from the machine and not from the record.
  private static func signInCard(facts: Facts, sessions: [LiveSession]) -> NowItem? {
    // A sign-in the app started is the Account row with its countdown, and
    // the last `auth status` still says the token does not work until that
    // child answers. A card asking for a sign-in under a sign-in in flight
    // is the same request twice, and its button is the one call
    // `signInWithGoogle` refuses while one is running.
    guard facts.signIn == nil else { return nil }
    guard facts.auth == .needsSignIn || facts.auth == .missingScopes else { return nil }
    // A watch whose token Google refused is stopped: the skill's next wait
    // cannot start, so the sentence asks for the sign-in and for the watch
    // to be started again. It never says the watch will carry on by itself,
    // which is a forecast nobody measured.
    let stopped = sessions.filter(\.isOpen).count
    let reason =
      stopped == 0
      ? "Nothing can reach Google until you sign in."
      : stopped == 1
        ? "1 live watch stopped. Sign in and start it again."
        : "\(stopped) live watches stopped. Sign in and start them again."
    return .stuck(
      id: "auth",
      card: StuckCardModel(title: facts.auth.rawValue, reason: reason),
      actions: [
        ActionModel(.signIn, icon: .signin, tip: Tips.signIn, target: "auth")
      ])
  }

  private static func stuckActions(
    _ write: StuckWrite, facts: Facts, ui: UIState, now: Date
  ) -> [ActionModel] {
    var actions: [ActionModel] = []
    if let doc = write.doc {
      actions.append(openDoc(doc))
      actions.append(openNote(doc, facts: facts, ui: ui))
      actions.append(openChat(doc, facts: facts))
      actions.append(copyLink(doc, ui: ui, now: now))
    } else {
      // A write that created nothing has nothing to open or to copy, and a
      // dead icon is what made this card read as buttons again.
      actions.append(
        ActionModel(
          .openNote, icon: .md, tip: Tips.openNote, state: .off(Tips.noNote),
          target: write.key, forcedTooltip: ui.forcedTooltip == "\(write.key)|openNote"))
      actions.append(openChat(write.doc, facts: facts, target: write.key))
    }
    actions.append(
      ActionModel(.dismiss, icon: .xmark, tip: Tips.dismiss, target: write.key))
    return actions
  }

  /// Stop, or the words that say why this watch is not one the app can
  /// stop. Between two polls there is no process to signal; the rest is the
  /// process check's answer, which arrives as a fact in `Facts`.
  private static func stopAction(_ session: LiveSession, facts: Facts, ui: UIState)
    -> ActionModel?
  {
    guard !ui.pendingStop.contains(session.doc) else { return nil }
    // A watch whose last poll Google refused the token on draws no Stop at
    // all, not even a dim one: between two ordinary polls the next wait is
    // a second away and the dim icon says why it cannot be signalled, but
    // here there is no next wait until somebody signs in. Scene e9:
    // testTheExpiredSignInOfE9StopsTheWatch.
    guard session.lastPollStop != "auth" else { return nil }
    let cannot = session.flight == nil || facts.unstoppable.contains(session.doc)
    let state: ActionState = cannot ? .off(Tips.cannotStop) : .live
    return ActionModel(
      .stopLive, icon: .stop, tip: LiveRow.stopTip, state: state, target: session.doc)
  }

  // MARK: DOCUMENTS

  private static func documentsSection(
    _ fold: DocumentsFold, facts: Facts, ui: UIState, words: Phrases, busy: Bool
  ) -> DocumentsSectionModel {
    let rows = fold.rows.map { row in
      DocumentRowModel(
        doc: row.doc,
        title: row.title ?? row.doc,
        sub: words.documentSub(row),
        glyph: glyph(for: row.state).kind,
        tone: glyph(for: row.state).tone,
        gone: false,
        actions: [
          openDoc(row.doc, forced: ui.forcedTooltip),
          openNote(row.doc, facts: facts, ui: ui, forced: ui.forcedTooltip),
          openChat(row.doc, facts: facts, forced: ui.forcedTooltip),
          copyLink(row.doc, ui: ui, now: words.now),
        ])
    }
    return DocumentsSectionModel(
      count: count(fold),
      rows: rows,
      collapsed: Documents.defaultRows,
      isOpen: ui.isOpen(.documents),
      // The hint is the first run's: it tells a person who has never
      // published how to start. With a write already running it tells
      // them to do the thing they are doing, so the line is the quiet one
      // until that write names a document.
      empty: rows.isEmpty ? (busy ? Sentences.nothingYet : Sentences.noDocuments) : nil,
      showsStrip: !ui.stripSeen && !rows.isEmpty)
  }

  /// One rule, and one number in it: `3 today` where every row in the list
  /// is from today, and `14 days · 12` where one of them is older.
  ///
  /// The number is the list's and never the screen's, because a row behind
  /// "Show more" is a document the person has. `1 today` over four rows
  /// from three days, which is what the old rule drew on scene e2, is a
  /// count of one thing over a list of another:
  /// testTheDocumentsHeaderCountsTheWholeList.
  private static func count(_ fold: DocumentsFold) -> String? {
    guard !fold.rows.isEmpty else { return nil }
    if fold.today == fold.rows.count { return "\(fold.today) today" }
    return "14 days · \(fold.rows.count)"
  }

  private static func glyph(for state: DocumentState) -> (kind: GlyphKind, tone: GlyphTone) {
    switch state {
    case .hold: return (.pause, .hold)
    case .running: return (.empty, .quiet)
    case .live: return (.dot, .live)
    case .liveStopped: return (.dot, .dim)
    case .done: return (.check, .ok)
    case .failed: return (.exclamation, .bad)
    case .unknown: return (.empty, .quiet)
    }
  }

  // MARK: TODAY and Yesterday

  private static func todaySection(_ fold: DaysFold, ui: UIState, words: Phrases)
    -> LogSectionModel
  {
    LogSectionModel(
      rows: fold.today.enumerated().map { words.logRow($1, index: $0) },
      collapsed: Days.defaultRows,
      isOpen: ui.isOpen(.today),
      empty: fold.today.isEmpty ? Sentences.nothingYet : nil)
  }

  private static func yesterdaySection(_ fold: DaysFold, ui: UIState, words: Phrases)
    -> YesterdayModel?
  {
    guard !fold.yesterday.isEmpty else { return nil }
    let summary = fold.yesterdaySummary
    return YesterdayModel(
      summary: "\(Sentences.count(summary.actions, "action")) · "
        + Sentences.count(summary.documents, "document"),
      isOpen: ui.isOpen(.yesterday),
      rows: fold.yesterday.enumerated().map { words.logRow($1, index: $0 + 1_000) })
  }

  // MARK: the one line about the app itself

  private static func notice(_ facts: Facts) -> String? {
    switch facts.bundle {
    case .same: break
    case .reopenNeeded(let version): return "Updated to \(version). Quit and reopen"
    case .gone: return "gdoc presence was removed"
    }
    // The app on the disk comes first, because that is the one a person can
    // do something about; a record from a newer gdoc is said next, because
    // every line of it was let go and the panel would otherwise be quiet
    // with nothing to explain it.
    return facts.newerRecord ? "gdoc is newer than this app" : nil
  }

  // MARK: the four document icons

  private static func openDoc(_ doc: String, forced: String? = nil) -> ActionModel {
    ActionModel(
      .openDoc, icon: .gdocs, tip: Tips.openDoc, target: doc,
      forcedTooltip: forced == "\(doc)|openDoc")
  }

  /// Open md, greyed with the reason where there is one.
  ///
  /// The reason is the last click's own fault before the pairing, because a
  /// note the index still names may be one Obsidian would not open: that is
  /// the only route to "Note moved or deleted", and without it a click that
  /// went nowhere would leave the icon live and say nothing:
  /// testAFailedOpenGreysTheIconWithItsReason.
  private static func openNote(
    _ doc: String, facts: Facts, ui: UIState, forced: String? = nil
  ) -> ActionModel {
    let paired = facts.note(for: doc) != nil
    let state: ActionState =
      if let fault = ui.noteFault[doc] { .off(fault.tip) } else if paired { .live } else {
        .off(Tips.noNote)
      }
    return ActionModel(
      .openNote, icon: .md, tip: Tips.openNote, state: state,
      target: doc, forcedTooltip: forced == "\(doc)|openNote")
  }

  private static func openChat(
    _ doc: String?, facts: Facts, target: String? = nil,
    forced: String? = nil
  ) -> ActionModel {
    let name = target ?? doc ?? ""
    let state: ActionState =
      facts.extensionState == .notInstalled ? .off(Tips.noExtension) : .live
    return ActionModel(
      .openChat, icon: .branch, tip: Tips.openChat, state: state, target: name,
      forcedTooltip: forced == "\(name)|openChat")
  }

  /// Copy link, green for the one second after the press.
  ///
  /// The moment is read as well as the document: a `.done` button holds its
  /// own tooltip open, so a check that never went back would keep "Copied"
  /// on the screen and no other icon in the panel could show its words
  /// again: testTheCopiedCheckGoesBackAfterItsSecond.
  private static func copyLink(_ doc: String, ui: UIState, now: Date) -> ActionModel {
    let done = ui.copied == doc && ActionButton.isDone(copiedAt: ui.copiedAt, now: now)
    return ActionModel(
      .copyLink, icon: .link, tip: done ? Tips.copied : Tips.copyLink,
      state: done ? .done : .live, target: doc)
  }
}
