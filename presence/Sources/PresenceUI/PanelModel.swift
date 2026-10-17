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
// stopped carries one sentence that names the moment and where the reason
// is, because the v1 record carries no cause: M24b adds `stop`, and Task 25
// replaces the sentence with it.
//
// Nothing here predicts anything, which is principle 5. The job card's ring
// has no fraction until the record carries steps, and no row says when
// something will finish.

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
    let sessions = Live.sessions(events, now: now)
    let docs = Documents.fold(events, now: now, calendar: calendar, alive: alive)
    let days = Days.fold(events, now: now, calendar: calendar, alive: alive)

    let words = Phrases(facts: facts, now: now, calendar: calendar)
    let items = nowItems(nowFold, sessions: sessions, facts: facts, ui: ui, words: words)
    let documents = documentsSection(
      docs, facts: facts, ui: ui, words: words, busy: !items.isEmpty)
    let today = todaySection(days, words: words)
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
            asking: ui.question == .dismissHold(hold.id)))
      case .stuck(let write):
        items.append(
          .stuck(
            id: write.key, card: words.stuckCard(write),
            actions: stuckActions(write, facts: facts, ui: ui)))
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
    guard facts.auth == .needsSignIn || facts.auth == .missingScopes else { return nil }
    let paused = sessions.filter(\.isOpen).count
    let reason =
      paused == 0
      ? "Nothing can reach Google until you sign in."
      : "\(paused) live \(paused == 1 ? "watch is" : "watches are") paused. "
        + "Sign in to carry on."
    return .stuck(
      id: "auth",
      card: StuckCardModel(title: facts.auth.rawValue, reason: reason),
      actions: [
        ActionModel(.signIn, icon: .signin, tip: Tips.signIn, target: "auth")
      ])
  }

  private static func stuckActions(_ write: StuckWrite, facts: Facts, ui: UIState)
    -> [ActionModel]
  {
    var actions: [ActionModel] = []
    if let doc = write.doc {
      actions.append(openDoc(doc))
      actions.append(openNote(doc, facts: facts))
      actions.append(openChat(doc, facts: facts))
      actions.append(copyLink(doc, ui: ui))
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
          openNote(row.doc, facts: facts, forced: ui.forcedTooltip),
          openChat(row.doc, facts: facts, forced: ui.forcedTooltip),
          copyLink(row.doc, ui: ui),
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

  /// `14 days · 12` once the list is longer than what is on the screen, and
  /// the day's own count otherwise. The label is the one place a number
  /// says how much is not on the screen.
  private static func count(_ fold: DocumentsFold) -> String? {
    guard !fold.rows.isEmpty else { return nil }
    if fold.rows.count > Documents.defaultRows { return "14 days · \(fold.rows.count)" }
    return fold.today > 0 ? "\(fold.today) today" : nil
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

  private static func todaySection(_ fold: DaysFold, words: Phrases) -> LogSectionModel {
    LogSectionModel(
      rows: fold.today.enumerated().map { words.logRow($1, index: $0) },
      collapsed: Days.defaultRows,
      isOpen: false,
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

  // MARK: HEALTH

  /// The rows, in the order `HealthRowKind` lists them.
  ///
  /// A row with nothing to say is left out and the rest keep their places.
  /// When every fact is good the list is the short one, because eight green
  /// rows on every screen is eight rows nobody reads; the two that are not
  /// facts stay, so Quit is always one click away.
  private static func healthRows(
    facts: Facts, ui: UIState, now: Date, stopping: Stopping
  ) -> [HealthRowModel] {
    var rows: [HealthRowModel] = [account(facts: facts, ui: ui, now: now, stopping: stopping)]

    if let network = facts.network, !network.reachable {
      rows.append(
        HealthRowModel(
          .network, value: "Google not reachable", glyph: .exclamation, tone: .bad,
          quiet: network.detail))
    }

    rows.append(gdoc(facts: facts))

    let quiet = facts.auth.isWorking && facts.network?.reachable != false && facts.gdocPath != nil
    if !quiet {
      if let skills = facts.versions.skills {
        rows.append(HealthRowModel(.skills, value: skills, glyph: .check, tone: .ok))
      }
      rows.append(contentsOf: desktop(facts: facts))
    }

    rows.append(HealthRowModel(.mark, value: facts.mark))
    rows.append(loginItem(facts: facts))
    rows.append(HealthRowModel(.quit, value: HealthRows.quitShortcut, select: .quit))
    return rows
  }

  /// What signing out would stop, which is what the confirm names.
  struct Stopping: Equatable {
    let live: Int
    let holds: Int

    /// The sentence under the question. Nothing running is said as plainly
    /// as two watches are: a person pressing Sign out is owed the
    /// consequence either way.
    var sentence: String {
      var parts: [String] = []
      if live > 0 { parts.append("\(live) live \(live == 1 ? "watch" : "watches")") }
      if holds > 0 { parts.append("\(holds) \(holds == 1 ? "hold" : "holds")") }
      guard !parts.isEmpty else { return "Nothing is running. Only the token goes." }
      return parts.joined(separator: " and ") + " will stop."
    }
  }

  private static func stopping(_ items: [NowItem], sessions: [LiveSession]) -> Stopping {
    let holds = items.filter { if case .hold = $0 { return true } else { return false } }
    return Stopping(live: sessions.filter(\.isOpen).count, holds: holds.count)
  }

  private static func account(
    facts: Facts, ui: UIState, now: Date, stopping: Stopping
  ) -> HealthRowModel {
    if let phase = facts.signIn {
      return HealthRowModel(
        .account, value: "Waiting for Google", waiting: true,
        quiet: "\(countdown(phase.secondsLeft)) left of the three minutes gdoc waits",
        actions: [
          ActionModel(.copySignInLink, icon: .link, tip: Tips.copySignIn, target: "auth"),
          ActionModel(.cancelSignIn, icon: .x, tip: Tips.cancel, target: "auth"),
        ])
    }

    let changed = facts.previousAccount.map { "Account changed: \($0) → \(facts.account ?? "")" }

    if ui.isAsking(.signOut, now: now) {
      return HealthRowModel(
        .account, glyph: .empty, quiet: changed,
        question: QuestionModel(
          title: "Sign out?", consequence: stopping.sentence, danger: .signOut))
    }

    guard facts.auth.isWorking else {
      return HealthRowModel(
        .account, value: facts.account ?? facts.auth.rawValue, glyph: .exclamation, tone: .bad,
        quiet: changed,
        actions: [ActionModel(.signIn, icon: .signin, tip: Tips.signIn, target: "auth")])
    }

    return HealthRowModel(
      .account, value: facts.account ?? Sentences.accountNotRead, glyph: .check, tone: .ok,
      quiet: changed,
      actions: [ActionModel(.signOut, icon: .signout, tip: Tips.signOut, target: "auth")])
  }

  /// The sign-in's own countdown, as a clock rather than as a number of
  /// seconds: 2:43 is read at a glance and 163 s is counted.
  private static func countdown(_ seconds: Int) -> String {
    let left = max(0, seconds)
    return String(format: "%d:%02d", left / 60, left % 60)
  }

  private static func gdoc(facts: Facts) -> HealthRowModel {
    guard facts.gdocFound, let version = facts.versions.gdoc else {
      return HealthRowModel(
        .gdoc, value: "not found", glyph: .exclamation, tone: .bad,
        quiet: "looked for it at \(facts.gdocPathTried)")
    }
    // While the update's own child is up the row is a ring and the version
    // it names is the one still installed: there is nothing to press until
    // the end arrives, and nothing honest to say about what will be there.
    if facts.updating {
      return HealthRowModel(.gdoc, value: version, waiting: true)
    }
    guard let latest = facts.versions.updateAvailable else {
      return HealthRowModel(.gdoc, value: version, glyph: .check, tone: .ok)
    }
    return HealthRowModel(
      .gdoc, value: "\(version) · \(latest) available", glyph: .check, tone: .ok,
      actions: [
        ActionModel(.update, icon: .down, tip: "Update to \(latest)", target: "gdoc")
      ])
  }

  private static func desktop(facts: Facts) -> [HealthRowModel] {
    switch facts.extensionState {
    case .connected:
      return [HealthRowModel(.desktop, value: "extension connected", glyph: .check, tone: .ok)]
    case .notInstalled:
      return [
        HealthRowModel(
          .desktop, value: "not installed", glyph: .empty,
          actions: [
            ActionModel(
              .installExtension, icon: .down, tip: Tips.installExtension, target: "desktop")
          ])
      ]
    case .unknown:
      return []
    }
  }

  private static func loginItem(facts: Facts) -> HealthRowModel {
    switch facts.loginItem {
    case .on, .off:
      return HealthRowModel(
        .launchAtLogin, isOn: facts.loginItem == .on, select: .toggleLoginItem)
    case .needsApproval:
      return HealthRowModel(
        .launchAtLogin, quiet: Sentences.loginItemApproval, isOn: false,
        select: .openLoginItems)
    }
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

  private static func openNote(_ doc: String, facts: Facts, forced: String? = nil) -> ActionModel {
    let paired = facts.note(for: doc) != nil
    return ActionModel(
      .openNote, icon: .md, tip: Tips.openNote, state: paired ? .live : .off(Tips.noNote),
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

  private static func copyLink(_ doc: String, ui: UIState) -> ActionModel {
    let done = ui.copied == doc
    return ActionModel(
      .copyLink, icon: .link, tip: done ? Tips.copied : Tips.copyLink,
      state: done ? .done : .live, target: doc)
  }
}

/// The words on an icon, which are the whole of its label. Each is the
/// screens file's own `data-tip`.
public enum Tips {
  public static let openDoc = "Open in Google Docs"
  public static let noDoc = "No Google Doc yet"
  public static let openNote = "Open the note in Obsidian"
  public static let noNote = "No note is paired with this document"
  /// The index named a note that is not there any more. It is a fact about
  /// the hub and never a complaint about the person who moved it.
  public static let noteMoved = "Note moved or deleted"
  public static let openChat = "Continue with Claude"
  public static let noExtension = "Install the Desktop extension in Health"
  public static let copyLink = "Copy link"
  public static let copied = "Copied"
  public static let dismiss = ButtonWord.dismiss.rawValue
  public static let cannotStop = "This watch is not one gdoc presence can stop"
  public static let signIn = "Sign in with Google"
  public static let signOut = ButtonWord.signOut.rawValue
  public static let cancel = "Cancel"
  public static let copySignIn = "Copy sign-in link"
  public static let installExtension = "Install extension"
}

/// The sentences the panel writes that are nobody's row.
public enum Sentences {
  /// DOCUMENTS with nothing in it, which is the first open of a machine
  /// that has signed in and done nothing yet.
  public static let noDocuments =
    "Publish a note or review a document from Claude Code and it will appear here."
  /// TODAY with nothing in it.
  public static let nothingYet = "nothing yet"
  /// A token with no address in it. Whose account it is decides nothing, so
  /// this is a fact and not a fault.
  public static let accountNotRead = "signed in, account not read"
  public static let loginItemApproval = "macOS is waiting for you to allow it in Login Items"

  /// A count and the thing counted, with the noun agreeing with it. One
  /// document is a document, and the panel is read by somebody whose first
  /// language is not English, so "1 documents" is a sentence that stops them.
  public static func count(_ number: Int, _ noun: String) -> String {
    "\(number) \(noun)\(number == 1 ? "" : "s")"
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
        consequence: "The row in TODAY stays. Nothing is written either way.")
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

  /// One sentence, under eighty characters: when it stopped, and where the
  /// reason is. The v1 record carries no cause at all, so the panel says
  /// where to look rather than guessing why. M24b adds `stop`, and Task 25
  /// puts the cause and its line here.
  private func stuckReason(_ write: StuckWrite) -> String {
    let place = write.via == "chat" ? "Desktop chat" : "terminal"
    return "Stopped at \(clock(write.at)). The \(place) that ran it has the reason."
  }

  func jobCard(_ run: NowRun) -> JobCardModel {
    JobCardModel(
      title: sentence([Words.running(run.command) ?? run.command, run.title]),
      sub: sentence(["started \(clock(run.started))", Words.via(run.via)]),
      progress: nil,
      steps: [])
  }

  /// A watch's one line of facts.
  ///
  /// A watch that is not getting through says so and nothing else: the row
  /// is one line at 320 pt, and since and quiet pushed the one thing that
  /// is wrong off the end of it. Scene r8 is that line, and it carries no
  /// word for why, because the record has none.
  func liveRow(_ session: LiveSession) -> LiveRowModel {
    let expired = facts.auth.isWorking ? nil : "sign-in expired"
    let failed = session.lastPollFailed.map { "last poll failed \(clock($0))" }

    let parts: [String?] =
      failed == nil && expired == nil
      ? [
        "since \(clock(session.since))",
        "quiet \(Words.duration(now.timeIntervalSince(session.quietSince)))",
        session.replied > 0 ? "answered \(session.replied)" : nil,
        session.proposed > 0 ? "proposed \(session.proposed)" : nil,
      ]
      : [failed, expired]

    return LiveRowModel(
      title: sentence(["Live", session.title ?? session.doc]),
      sub: sentence(parts))
  }

  // MARK: the lists

  /// A document row's second line: what happened to it, and where its note
  /// is. The state of an open action is one word, because the record has
  /// none and a person reading the list wants the state and not the verb.
  func documentSub(_ row: DocumentRow) -> String {
    let note = facts.note(for: row.doc) ?? "not in the hub"
    return sentence([state(row), note])
  }

  private func state(_ row: DocumentRow) -> String {
    switch row.state {
    case .hold: return "hold"
    case .running: return (Words.running(row.command) ?? "running").lowercased()
    case .live: return "live"
    case .liveStopped: return "live stopped \(clockInSentence(row.at))"
    case .done:
      return "\(Words.verb(row.command)?.past ?? row.command) \(clockInSentence(row.at))"
    case .failed: return "stopped \(clockInSentence(row.at))"
    case .unknown: return "outcome unknown · \(clockInSentence(row.at))"
    }
  }

  func logRow(_ row: DayRow, index: Int) -> LogRowModel {
    let what = Words.verb(row.command)?.heading ?? row.command ?? "gdoc"
    let title = row.title ?? row.doc

    switch row.kind {
    case .action:
      let ok = row.ok ?? false
      return LogRowModel(
        key: "\(index)",
        glyph: ok ? .check : .exclamation,
        tone: ok ? (row.verified == false ? .quiet : .ok) : .bad,
        title: sentence([what, title]),
        value: clock(row.at),
        sub: sentence([Words.via(row.via), Words.verified(row.verified), ok ? nil : "stopped"]))
    case .outcomeUnknown:
      return LogRowModel(
        key: "\(index)", glyph: .empty, tone: .quiet, title: sentence([what, title]),
        value: clock(row.at), sub: sentence([Words.via(row.via), "outcome unknown"]))
    case .holdRaised:
      return LogRowModel(
        key: "\(index)", glyph: .pause, tone: .hold, title: sentence(["Hold", title]),
        value: clock(row.at),
        sub: sentence([
          row.rule.map { "\(row.command ?? "write") held by rule \($0)" },
          "waiting in Desktop chat",
        ]))
    case .holdClosed:
      return LogRowModel(
        key: "\(index)", glyph: row.ending == .landed ? .check : .dot,
        tone: row.ending == .landed ? .ok : .dim,
        title: sentence(["Hold closed", title]), value: clock(row.at),
        sub: ending(row.ending))
    case .liveStarted:
      return LogRowModel(
        key: "\(index)", glyph: .dot, tone: .dim, title: sentence(["Live started", title]),
        value: clock(row.at), sub: nil)
    case .liveStopped:
      return LogRowModel(
        key: "\(index)", glyph: .dot, tone: .dim, title: sentence(["Live stopped", title]),
        value: clock(row.at), sub: totals(row.totals))
    }
  }

  /// What the record said a hold came to, in the panel's words and with no
  /// reading on top.
  private func ending(_ ending: HoldEnding?) -> String? {
    switch ending {
    case .landed: return "approved by you in chat"
    case .released: return "released without writing"
    case .expired: return "ran out"
    case nil: return nil
    }
  }

  /// What a watch came to. A count of nothing is left out and the two of
  /// them together are said once: "answered 0 · proposed 0" is three words
  /// for one fact.
  private func totals(_ totals: LiveTotals?) -> String? {
    guard let totals else { return nil }
    let wrote = totals.replied + totals.proposed
    return sentence([
      Words.duration(totals.ran),
      wrote == 0 ? "nothing written" : nil,
      totals.replied > 0 ? "answered \(totals.replied)" : nil,
      totals.proposed > 0 ? "proposed \(totals.proposed)" : nil,
    ])
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
    parts.compactMap { $0 }.filter { !$0.isEmpty }.joined(separator: " · ")
  }
}
