// The whole panel as values, before anything is drawn.
//
// The view draws this and decides nothing, which is the rule M23 started
// with and this milestone keeps: every word a person can read is a value a
// test can state, and a render adds only whether the pixels came out.
//
// It is built from three things and nothing else: the events, the facts
// about the machine, and what the person did to the panel. The three folds
// are taken here so the shell hands over a list of events and never three
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
  /// A pin, taking it back, and a hide. The first two are the row's fifth
  /// icon since the third look; the hide is the one word of its menu.
  case pinDoc
  case unpinDoc
  case hideDoc
}

/// One word of a row's own menu, and what pressing it means.
///
/// A menu item is not an action button: it has a word and no icon, and no
/// state, because a menu that is open draws only what can be pressed.
public struct MenuItemModel: Sendable, Equatable, Identifiable {
  public let kind: ActionKind
  public let word: String
  public let target: String

  public init(_ kind: ActionKind, word: String, target: String) {
    self.kind = kind
    self.word = word
    self.target = target
  }

  public var id: String { "\(target)|\(kind.rawValue)" }
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
/// One function, so a view that gains a button gains no knowledge of what
/// the button reaches. The shell fills it in; every test and every still
/// leaves it alone.
public struct PanelActions {
  public var perform: (ActionKind, String?) -> Void

  public init(perform: @escaping (ActionKind, String?) -> Void = { _, _ in }) {
    self.perform = perform
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
  /// Whether the person pinned it, which the row draws as the state of its
  /// fifth icon: the tack is filled on a pinned row and hollow on the rest.
  public let isPinned: Bool
  public let actions: [ActionModel]
  /// The row's own menu: Hide from the list, and nothing on a pinned row.
  public let menu: [MenuItemModel]

  public var id: String { doc }
}

/// A question asked in place of a row: the words, what follows from them,
/// and the word on the loud half of the pair. The quiet half is always Keep.
public struct QuestionModel: Sendable, Equatable {
  public let title: String
  public let consequence: String
  public let danger: ButtonWord
}

/// One SETTINGS row, with its words and what sits on its right.
public struct SettingsRowModel: Sendable, Equatable, Identifiable {
  public let kind: SettingsRowKind
  public let value: String
  public let glyph: GlyphKind
  public let tone: GlyphTone
  /// The ring in the glyph's place. The two rings the panel draws are the
  /// header's since 2026-10-18, the sign-in's and the update's, so no row
  /// here draws one today.
  public let waiting: Bool
  public let quiet: String?
  public let actions: [ActionModel]
  public let isOn: Bool?
  /// Pressing the row itself, which Quit and the Login Items row do.
  ///
  /// No row here is ever a question. The one destructive confirm in the
  /// panel is the sign-out, and it is asked in the header since 2026-10-18:
  /// testTheQuestionKeepsTheAccountsGlyphAndSaysNothingElse.
  public let select: ActionKind?

  public var id: String { kind.rawValue }

  public init(
    _ kind: SettingsRowKind, value: String = "", glyph: GlyphKind = .blank,
    tone: GlyphTone = .quiet, waiting: Bool = false, quiet: String? = nil,
    actions: [ActionModel] = [], isOn: Bool? = nil, select: ActionKind? = nil
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
  }
}

/// DOCUMENTS: its rows, every one of which is drawn.
public struct DocumentsSectionModel: Sendable, Equatable {
  public let rows: [DocumentRowModel]
  /// The one line a panel with no documents says instead.
  public let empty: String?
  /// Whether the five labels are drawn under the first row.
  public let showsStrip: Bool
}

/// The address said once at its full size, for five seconds after a sign-in.
public struct BannerModel: Sendable, Equatable {
  public let account: String
}

/// How the header's update glyph is drawn, which is the whole of what the
/// panel says about what is published.
///
/// Running is not here: the ring stands in the glyph's place while a child
/// of the app's own is up, which is `versionWaiting`, and there is nothing
/// to press. The one-second check after a run that installed nothing is the
/// glyph's own `ActionState`, as a copied link's is.
public enum UpdateLook: String, Sendable, Equatable, CaseIterable {
  /// Nothing newer is known. The icon is the caption grey, and pressing it
  /// is how a person asks before the six hours are up.
  case quiet
  /// `help --json` listed a newer stable release: the accent ink, the dot,
  /// and the published version in the quiet line under the header.
  case available
}

/// The panel's own line, as values: the mark and the name, the version with
/// the one action on it, and the account with its own.
///
/// Both of those were HEALTH rows until 2026-10-18, at the foot of a panel
/// that scrolls. The account is the first thing a person checks before a
/// reply goes out under their name, and the version is which gdoc this is,
/// so the two of them are what the header is about.
public struct HeaderModel: Sendable, Equatable {
  /// The mark in effect, which the binary's settings decide. It may be
  /// empty, and the header draws whatever it is handed.
  public let mark: String
  /// The product's name, which is the one word here that never changes.
  public let name: String
  /// The version `help --json` answered with, "not found" where no binary
  /// was there to ask, or nothing before the answer arrived.
  public let version: String?
  /// The version's ink: the quietest grey, or the bad one where there is no
  /// binary to name a version of.
  public let versionTone: GlyphTone
  /// The ring beside the version, which an update child of the app's own
  /// draws while it is up.
  public let versionWaiting: Bool
  /// The update glyph, which is in the header whenever there is a binary to
  /// update and no child of the app's own up. Pressing it runs `gdoc
  /// update` whatever state it is in, because that command is the one that
  /// asks GitHub.
  public let update: ActionModel?
  /// Which of the two the glyph is drawn as.
  public let updateLook: UpdateLook
  /// The address, or nothing where no token has been read.
  public let account: String?
  public let glyph: GlyphKind
  public let tone: GlyphTone
  /// The ring in the glyph's place, which nobody-has-asked-yet and a sign-in
  /// in flight draw.
  public let waiting: Bool
  /// The account's own actions: Sign out, Sign in, or the sign-in's Copy
  /// and Cancel.
  public let actions: [ActionModel]
  /// The question asked in place of the line, while it is asked.
  public let question: QuestionModel?
  /// The lines under the line, in the order they are drawn: the account's
  /// own first, because the countdown belongs under the countdown, and the
  /// version's after it.
  public let quiet: [String]
  /// The sign-in URL, as a line that can be read and copied.
  public let signInURL: String?
  /// The address at its full size, for five seconds after a sign-in.
  public let banner: BannerModel?

  public init(
    mark: String = "",
    name: String = "gdoc",
    version: String? = nil,
    versionTone: GlyphTone = .dim,
    versionWaiting: Bool = false,
    update: ActionModel? = nil,
    updateLook: UpdateLook = .quiet,
    account: String? = nil,
    glyph: GlyphKind = .blank,
    tone: GlyphTone = .quiet,
    waiting: Bool = false,
    actions: [ActionModel] = [],
    question: QuestionModel? = nil,
    quiet: [String] = [],
    signInURL: String? = nil,
    banner: BannerModel? = nil
  ) {
    self.mark = mark
    self.name = name
    self.version = version
    self.versionTone = versionTone
    self.versionWaiting = versionWaiting
    self.update = update
    self.updateLook = updateLook
    self.account = account
    self.glyph = glyph
    self.tone = tone
    self.waiting = waiting
    self.actions = actions
    self.question = question
    self.quiet = quiet
    self.signInURL = signInURL
    self.banner = banner
  }
}

/// The sections of the panel, in the order they are drawn.
public enum PanelSection: Sendable, Equatable, Identifiable {
  case now([NowItem])
  case documents(DocumentsSectionModel)
  case settings([SettingsRowModel])

  /// The label over the section, as the screens file writes it.
  public var label: String {
    switch self {
    case .now: return "Now"
    case .documents: return "Documents"
    case .settings: return "Settings"
    }
  }

  public var id: String { label }
}

/// The whole panel.
public struct PanelModel: Sendable, Equatable {
  /// The panel's own line and everything drawn under it.
  public let header: HeaderModel
  /// The centred first-run words, where there is nothing else at all.
  public let empty: String?
  /// The one line the app says about itself: the bundle on the disk, or a
  /// record from a gdoc this app cannot read.
  public let notice: String?
  public let sections: [PanelSection]
  /// The one action whose words are held open with no pointer.
  public let forcedTooltip: String?

  /// The mark in effect, which the header and the empty state draw.
  public var mark: String { header.mark }
  /// The address in the header, or nothing.
  public var account: String? { header.account }
  /// The sign-in URL under the header, where a sign-in is in flight.
  public var signInURL: String? { header.signInURL }

  public var now: [NowItem] {
    for section in sections {
      if case .now(let items) = section { return items }
    }
    return []
  }

  public var settings: [SettingsRowModel] {
    for section in sections {
      if case .settings(let rows) = section { return rows }
    }
    return []
  }

  public var documents: DocumentsSectionModel? {
    for section in sections {
      if case .documents(let model) = section { return model }
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
    let docs = Documents.fold(
      events, now: now, pinned: ui.pinned, hidden: ui.hidden, alive: alive)

    return build(
      now: nowFold, sessions: sessions, documents: docs, facts: facts, ui: ui,
      at: now, calendar: calendar)
  }

  /// The same panel out of folds the caller already took.
  ///
  /// Each of those three is a walk and a sort of every line the app holds,
  /// and the app takes all three on every tick to decide whether anything
  /// moved. Reading them again here made one redraw seven of those walks
  /// where three would do, on the main thread, at one second apiece while
  /// the ring or the sign-in countdown is moving. The folds and the words
  /// also share one reading of the clock this way, so the glyph and the
  /// panel beside it cannot describe two different moments.
  public static func build(
    now nowFold: NowFold, sessions: [LiveSession], documents docs: DocumentsFold,
    facts: Facts, ui: UIState = UIState(), at now: Date, calendar: Calendar
  ) -> PanelModel {
    let words = Phrases(facts: facts, now: now, calendar: calendar)
    let items = nowItems(nowFold, sessions: sessions, facts: facts, ui: ui, words: words)
    let documents = documentsSection(
      docs, facts: facts, ui: ui, words: words, busy: !items.isEmpty)
    let settings = settingsRows(facts: facts)
    let header = self.header(
      facts: facts, ui: ui, now: now, stopping: stopping(nowFold, sessions: sessions))

    // SETTINGS is last, where a person looks for it, except on the two
    // screens where the group is the whole of what the panel is for: a
    // machine with no token at all, and a sign-in in flight. A countdown
    // that scrolled out of sight is one nobody waits for.
    //
    // The third screen was the sign-out question, and it is not one any
    // more: the question is asked in the header since 2026-10-18, which is
    // above everything on every screen and cannot scroll away.
    let pinned = facts.auth == .signedOut || facts.signIn != nil

    var sections: [PanelSection] = []
    if pinned { sections.append(.settings(settings)) }
    if !items.isEmpty { sections.append(.now(items)) }
    sections.append(.documents(documents))
    if !pinned { sections.append(.settings(settings)) }

    let bare = items.isEmpty && documents.rows.isEmpty

    return PanelModel(
      header: header,
      empty: bare && facts.auth == .signedOut ? EmptyState.firstRun : nil,
      notice: notice(facts),
      sections: sections,
      forcedTooltip: ui.forcedTooltip)
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

    for session in watched(sessions, facts: facts) {
      items.append(
        .live(
          id: session.doc, row: words.liveRow(session),
          stop: stopAction(session, facts: facts, ui: ui),
          pending: ui.pendingStop.contains(session.doc)))
    }
    return items
  }

  /// The watches NOW draws: the open ones, and the ones a sign-in that
  /// stopped working ended, while it is still not working.
  ///
  /// A watch Google refused the token on is over, and DOCUMENTS says so,
  /// but it is the one ending nobody asked for: signing in does not
  /// start it again, and NOW is where the panel keeps what is waiting on the
  /// person. Once the token works the row leaves, because then the day's log
  /// is the whole of it: testE9SaysTheWatchIsStoppedInEverySectionAndDrawsNoStop.
  ///
  /// Two things have to hold, not one. The token has to be refused now,
  /// which is the machine's own word, and nothing may have got through
  /// since that watch stopped, which is the record's: a refusal last week
  /// that a sign-in answered is a watch nobody has run for days, and
  /// without the second half every one of them would come back into NOW
  /// the next time a token stopped working.
  private static func watched(_ sessions: [LiveSession], facts: Facts) -> [LiveSession] {
    sessions.filter { $0.isOpen || ($0.waitsForTheSignIn && facts.auth?.isWorking == false) }
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
    guard let auth = facts.auth, auth == .needsSignIn || auth == .missingScopes else {
      return nil
    }
    // A watch whose token Google refused is stopped: the skill's next wait
    // cannot start, so the sentence asks for the sign-in and for the watch
    // to be started again. It never says the watch will carry on by itself,
    // which is a forecast nobody measured.
    //
    // It counts the watches this sign-in stopped and not the rows NOW draws:
    // `watched` keeps the open ones too, and a watch still polling under a
    // token that is only short a scope would have the card call it stopped
    // while the row beside it drew it live with its Stop icon:
    // testTheSignInCardCountsOnlyTheWatchesItStopped.
    let stopped = sessions.filter(\.waitsForTheSignIn).count
    let reason =
      stopped == 0
      ? "Nothing can reach Google until you sign in."
      : stopped == 1
        ? "1 live watch stopped. Sign in and start it again."
        : "\(stopped) live watches stopped. Sign in and start them again."
    return .stuck(
      id: "auth",
      card: StuckCardModel(title: auth.rawValue, reason: reason),
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
      // A write that created nothing has nothing to open and nothing to
      // copy, so the card carries neither: a row in a list dims an icon and
      // says why, because the row is one of many and the column is the same
      // on all of them, but a card never carries an icon that would do
      // nothing. The note is not an exception, because no field of the
      // record names the file a publish was built from:
      // testACardCarriesNoIconThatWouldDoNothing.
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
    guard !session.stoppedBySignIn else { return nil }
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
        isPinned: row.isPinned,
        actions: [
          openDoc(row.doc, forced: ui.forcedTooltip),
          openNote(row.doc, facts: facts, ui: ui, forced: ui.forcedTooltip),
          openChat(row.doc, facts: facts, forced: ui.forcedTooltip),
          copyLink(row.doc, ui: ui, now: words.now),
          pin(row),
        ],
        menu: menu(row))
    }
    return DocumentsSectionModel(
      rows: rows,
      // The hint is the first run's: it tells a person who has never
      // published how to start. With a write already running it tells
      // them to do the thing they are doing, so the line is the quiet one
      // until that write names a document.
      empty: rows.isEmpty ? (busy ? Sentences.nothingYet : Sentences.noDocuments) : nil,
      showsStrip: !ui.stripSeen && !rows.isEmpty)
  }

  /// A row's own menu: the hide, on a row that is not pinned.
  ///
  /// Pin left it at the third look. It was the one way to pin a document and
  /// Nail never found it, so it is the row's fifth icon now, and Hide is
  /// what the menu is for.
  ///
  /// A pinned row offers nothing, so it opens no menu. A pin is the louder
  /// answer and is never away, whatever a hide of it says, so the press
  /// would do nothing on the screen and then take the row away at the unpin,
  /// which is not what either press asked for. Unpin first, and the row
  /// offers Hide.
  ///
  /// There is no Unhide: a hidden document has no row to open a menu on, and
  /// what brings it back is gdoc doing something on it again.
  private static func menu(_ row: DocumentRow) -> [MenuItemModel] {
    guard !row.isPinned else { return [] }
    return [MenuItemModel(.hideDoc, word: Menus.hide, target: row.doc)]
  }

  /// The pin, in the state the row is in: the outline tack that pins, or the
  /// filled one that takes the pin back.
  ///
  /// It is live on every row there is. A pin is the person's own answer
  /// about a document and nothing about the document can refuse it: a row
  /// whose id Drive could not have written still pins, because the row is
  /// there and the pin is what keeps it there.
  private static func pin(_ row: DocumentRow) -> ActionModel {
    ActionModel(
      row.isPinned ? .unpinDoc : .pinDoc,
      icon: row.isPinned ? .pinOn : .pin,
      tip: row.isPinned ? Tips.unpin : Tips.pin,
      target: row.doc)
  }

  /// The one mark a document row carries: its last run failed.
  ///
  /// The column drew five marks until the fourth look (docs/v2/DECISIONS.md,
  /// 2026-10-26): a green check for a run that worked, a dot for a live
  /// watch, a dim dot for one that stopped, a pause for a hold, and the
  /// exclamation. Nail read a list of grey dots and green checks and
  /// nothing on the panel said which was which, and the row's second line
  /// already says what the last run was. So the mark means one thing a
  /// person can name, and every other state draws nothing.
  ///
  /// Blank and not absent: the column stays, so every title in the list
  /// starts on the one left edge.
  /// testADocumentRowIsMarkedOnlyWhenItsLastRunFailed.
  private static func glyph(for state: DocumentState) -> (kind: GlyphKind, tone: GlyphTone) {
    state == .failed ? (.exclamation, .bad) : (.blank, .quiet)
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

  // MARK: the five document icons

  /// Open in Google Docs, greyed where the id is not one Drive could have
  /// written.
  ///
  /// `Opener` refuses to build a URL out of it, so the press opens nothing
  /// either way; what the state adds is the row saying why before the click
  /// rather than after it. A line carrying an id of that shape is a line
  /// gdoc did not write: testAnIdDriveCouldNotHaveWrittenGreysItsTwoIcons.
  private static func openDoc(_ doc: String, forced: String? = nil) -> ActionModel {
    ActionModel(
      .openDoc, icon: .gdocs, tip: Tips.openDoc, state: documentState(doc, missing: Tips.noDoc),
      target: doc,
      forcedTooltip: forced == "\(doc)|openDoc")
  }

  /// Whether there is a document behind an id at all, in the word of the
  /// icon that is asking.
  private static func documentState(_ doc: String, missing: String) -> ActionState {
    LinkLine.isDocumentID(doc) ? .live : .off(missing)
  }

  /// Open md, greyed with the reason where there is one.
  ///
  /// The reason is the last click's own fault before the pairing, because a
  /// note the index still names may be one Obsidian would not open: that is
  /// the only route to "Note moved or deleted", and without it a click that
  /// went nowhere would leave the icon live and say nothing:
  /// testAFailedOpenGreysTheIconWithItsReason.
  ///
  /// Where no note was found the reason is the finder's own: no note names
  /// this document, or two do and it will not pick between them. Nothing
  /// sends the person to SETTINGS for either since the third look, because
  /// there is no hub to choose:
  /// testTheMdIconIsGreyOnlyWhenNoNoteNamesTheDocument and
  /// testTwoNotesNamingOneDocumentSaySoOnTheIcon.
  private static func openNote(
    _ doc: String, facts: Facts, ui: UIState, forced: String? = nil
  ) -> ActionModel {
    let state: ActionState =
      if let fault = ui.noteFault[doc] {
        .off(fault.tip)
      } else if facts.note(for: doc) != nil {
        .live
      } else {
        .off((facts.noteFaults[doc] ?? .notPaired).tip)
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
      state: done ? .done : documentState(doc, missing: Tips.noLink), target: doc)
  }
}
