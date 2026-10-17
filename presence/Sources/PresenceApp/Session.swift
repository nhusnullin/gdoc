// Everything the app holds between two clicks.
//
// One tail over the record, the events it has delivered, the facts the two
// polls answered, what the person did to the panel, and the fold of all of it
// at the moment somebody looks. The shell holds this object and nothing else:
// it is the room where the pollers, the runner, the opener, the stopper, the
// stores and the one signal meet, and it is in the app library rather than in
// the executable so a test can reach every one of them.
//
// Nothing here reads a word of a document and nothing here judges one. The
// panel's words are PanelModel's, the schedule is the poller's, every child
// is the runner's, and what is written down is the preferences'. This file
// decides when each of them is asked and what the answer becomes.
//
// Three clocks run over it: the five-second refold, which is how a pid that
// died without writing its end is noticed; one second instead while a child
// of the app's own is up, so the ring and the sign-in's countdown move; and
// the one-shot timers that take a green check, a copied icon and a question
// back off the screen.

import Foundation
import PresenceCore
import PresenceUI

/// How many events are kept in memory. A day of ordinary use is a few
/// hundred lines, so this is far above it, and the oldest go first rather
/// than the list growing for as long as the app is open.
private let eventCeiling = 20_000

/// How often the fold is taken again with no new line. A pid that died
/// without writing its end changes the glyph, and nothing on the disk says
/// so, so the only way to notice is to ask the kernel again.
private let refoldEvery: TimeInterval = 5

/// How often while a child of the app's own is up. The ring and the sign-in
/// countdown are the two things on the screen that move, and both of them
/// move in seconds.
private let whileAChildIsUp: TimeInterval = 1

@MainActor
public final class Session {
  /// The version the app launched at, read off the bundle on the disk at
  /// launch and never from the cached bundle afterwards.
  public let version: String

  // MARK: the outside, all of it injected

  let places: Places
  let calendar: Calendar
  let preferences: Preferences
  let desk: Desk
  let opener: Opener
  let runner: Runner
  let polls: HealthPoller
  let updates: UpdateAction
  let signIn: SignIn
  let stopper: Stopper
  let loginItem: LoginItem
  let relauncher: Relauncher?
  let tail: LogTail
  let watcher: BundleWatcher?
  let alive: (Int32) -> Bool
  let clock: () -> Date
  let quit: () -> Void

  // MARK: what it holds

  var events: [Event] = []
  var ui: UIState
  var facts: Facts
  var glance: Glance
  var bundle: BundleState = .same
  var index = HubIndex.empty
  var hubWatcher: HubWatcher?

  /// Whether the record has held a line from a gdoc at a schema this app does
  /// not read. It is sticky: the line is skipped and never kept, so the one
  /// thing left to say is that it was there.
  var newerGdoc = false

  /// The sign-in in flight, as the panel draws it, and whether the app's own
  /// `auth login` child is up.
  var phase: SignInPhase?
  var signingIn = false
  var signedInAt: Date?

  /// Whether the app's own `update` child is up. It is the flag and not the
  /// runner's answer, because it is set by the press and the child is
  /// launched a moment later: the ring on the gdoc row is the actor's own
  /// standing, and this is what makes the clock tick fast enough to show it.
  var installing = false

  /// The menu bar's check, and whether what the tail has delivered so far is
  /// the file as it stood at launch. Every line in it is a replay, and a
  /// replay flashes no check.
  var check = BarCheck()
  var replayed = false

  var ticker: Timer?
  var tickingEvery: TimeInterval = refoldEvery
  var checkTimer: Timer?
  var revertTimer: Timer?
  var onChange: () -> Void = {}

  public init(
    places: Places,
    calendar: Calendar = .current,
    preferences: Preferences = Preferences(),
    desk: Desk = SystemDesk(),
    loginService: LoginService = SystemLoginService(),
    pad: Launchpad = SystemLaunchpad(),
    environment: [String: String] = ProcessInfo.processInfo.environment,
    alive: @escaping (Int32) -> Bool = Liveness.isAlive,
    clock: @escaping () -> Date = Date.init,
    quit: @escaping () -> Void = {}
  ) {
    let location = Runner.locate(home: places.home, bundle: places.bundle)
    let runner = Runner(location: location, environment: environment)
    let relaunched = RelaunchPolicy.launcher(in: places.arguments) != nil

    self.places = places
    self.version = places.version
    self.calendar = calendar
    self.preferences = preferences
    self.desk = desk
    self.opener = Opener(desk: desk)
    self.runner = runner
    self.polls = HealthPoller(
      location: location,
      answer: { try await runner.run($0) },
      memory: AccountMemory(seen: preferences.lastAccount))
    self.updates = UpdateAction(answer: { try await runner.run($0) })
    self.signIn = SignIn(
      start: { try runner.start(.authLogin, timeout: SignIn.childTimeout) },
      open: { desk.open($0) },
      clock: { Date() })
    self.stopper = Stopper(gdoc: location.path)
    self.loginItem = LoginItem(
      service: loginService, preferences: preferences, relaunched: relaunched)
    self.relauncher =
      places.bundle.flatMap { bundle in
        places.identifier.isEmpty
          ? nil : Relauncher(bundle: bundle, identifier: places.identifier, pad: pad)
      }
    self.tail = LogTail(directory: places.records)
    self.watcher = places.bundle.map { BundleWatcher(bundle: $0, running: places.version) }
    self.alive = alive
    self.clock = clock
    self.quit = quit

    self.ui = UIState(
      dismissed: preferences.dismissals(now: clock()), stripSeen: preferences.stripSeen)
    self.facts = Facts.launching(appVersion: places.version)
    self.glance = Fold.glance([], now: clock(), calendar: calendar, alive: alive)
  }

  // MARK: starting and stopping

  /// Start reading. The tail's first pass runs inside start, and what it
  /// found reaches this object through the same route every later pass does,
  /// so the batches arrive in the order the file holds them. There is no
  /// second, synchronous read here: it would take the newer batch first and
  /// an end seen before its start puts one run in NOW and in TODAY at once.
  public func start(onChange: @escaping () -> Void) {
    self.onChange = onChange
    loginItem.settle()
    readTheHub()

    tail.start { records in
      Task { @MainActor [weak self] in self?.took(records) }
    }
    watcher?.start { [weak self] state in self?.bundleMoved(to: state) }
    retick()

    // The tail's first pass ran inside start, so its batch is already on this
    // actor's queue ahead of this one: everything delivered after it arrived
    // while the app was running.
    Task { @MainActor [weak self] in self?.replayed = true }
    Task { [weak self] in await self?.settleTheLaunch() }
    Task { [weak self] in await self?.recompose() }
  }

  public func stop() {
    ticker?.invalidate()
    ticker = nil
    checkTimer?.invalidate()
    checkTimer = nil
    revertTimer?.invalidate()
    revertTimer = nil
    tail.stop()
    watcher?.stop()
    hubWatcher?.stop()
    hubWatcher = nil
  }

  /// The panel is on the screen. It is the one moment the app asks gdoc
  /// about the machine, at most once a minute, because `help --json` carries
  /// gdoc's own daily check of what is published.
  public func opened() {
    Task { [weak self] in await self?.ask(.panelOpened) }
  }

  // MARK: what the shell draws

  /// What the menu bar shows, out of the same two folds the panel is built
  /// from, so the item and an open panel never say different things.
  public var bar: BarState {
    let moment = clock()
    return BarImage.state(
      now: Now.fold(events, now: moment, dismissed: ui.dismissed, alive: alive),
      sessions: Live.sessions(events, now: moment),
      facts: facts,
      checkShowing: check.isShowing(now: moment))
  }

  /// What the panel draws: the events, the machine and the person.
  public var model: PanelModel {
    PanelModel.build(
      events: events, facts: facts, ui: ui, now: clock(), calendar: calendar, alive: alive)
  }

  // MARK: the record

  private func took(_ records: Records) {
    guard !records.isEmpty else { return }
    var said = false
    if records.newerSchema, !newerGdoc {
      newerGdoc = true
      said = true
    }
    if !records.events.isEmpty {
      events.append(contentsOf: records.events)
      if events.count > eventCeiling {
        events.removeFirst(events.count - eventCeiling)
      }
    }
    if noticedAWrite() { said = true }
    if refolded() || said { changed() }
  }

  /// A write that read itself back starts the menu bar's check, for five
  /// seconds, and only where its line arrived live. The one-shot timer is
  /// what takes the check off again: the five-second refold would not,
  /// because a glance that did not change redraws nothing.
  private func noticedAWrite() -> Bool {
    let moment = clock()
    let fold = Now.fold(events, now: moment, dismissed: ui.dismissed, alive: alive)
    guard check.saw(fold.lastVerified, live: replayed, at: moment) else { return false }
    checkTimer?.invalidate()
    checkTimer = Timer.scheduledTimer(withTimeInterval: BarCheck.lasts, repeats: false) { _ in
      Task { @MainActor [weak self] in self?.changed() }
    }
    return true
  }

  /// The fold taken again, and whether it moved. A glance that did not
  /// change redraws nothing.
  ///
  /// The waits a signal has gone to are read back here as well, because what
  /// forgets one is its end arriving in the record.
  @discardableResult
  private func refolded() -> Bool {
    let before = glance
    let pending = ui.pendingStop
    glance = Fold.glance(events, now: clock(), calendar: calendar, alive: alive)
    ui.pendingStop = stopper.pendingDocs(in: events)
    return glance != before || ui.pendingStop != pending
  }

  // MARK: the facts

  /// Ask the two questions this reason asks for, and draw what came back.
  func ask(_ reason: PollReason) async {
    await polls.poll(reason, live: isLive(), now: clock())
    if let account = await polls.reading.auth?.account { preferences.lastAccount = account }
    await recompose()
  }

  /// The facts rebuilt out of what every room knows, and a redraw where they
  /// moved.
  @discardableResult
  func recompose() async -> Bool {
    var next = await polls.applied(to: machine())
    next = await updates.applied(to: next)
    guard next != facts else { return false }
    facts = next
    changed()
    return true
  }

  /// What the app knows about the machine before the polls have spoken: its
  /// own version, the bundle under it, the hub, the switch, and the waits it
  /// may not signal.
  private func machine() -> Facts {
    Facts(
      auth: .ok,
      versions: Versions(app: version),
      hub: preferences.hub,
      loginItem: loginItem.state,
      signIn: phase,
      pairing: index.notes,
      signedInAt: signedInAt,
      bundle: bundle,
      newerRecord: newerGdoc,
      unstoppable: stopper.unstoppable(in: events))
  }

  /// Whether a watch is open, which is what makes the token a question
  /// every ten minutes.
  func isLive() -> Bool {
    Live.sessions(events, now: clock()).contains(where: \.isOpen)
  }

  // MARK: the clocks

  /// The repeating tick, at the interval the moment calls for. It is
  /// recreated only when that interval changes, so an ordinary minute is
  /// twelve ticks and not sixty.
  func retick() {
    let wanted = signingIn || installing ? whileAChildIsUp : refoldEvery
    guard ticker == nil || wanted != tickingEvery else { return }
    ticker?.invalidate()
    tickingEvery = wanted
    ticker = Timer.scheduledTimer(withTimeInterval: wanted, repeats: true) { _ in
      Task { @MainActor [weak self] in self?.tick() }
    }
  }

  private func tick() {
    if refolded() { changed() }
    readThePhase()
    Task { [weak self] in await self?.ask(.timer) }
  }

  /// The sign-in's countdown, read off the child's own start. It is a value
  /// on this object rather than a question the panel asks, because the panel
  /// is drawn synchronously and the sign-in is an actor.
  private func readThePhase() {
    guard signingIn else { return }
    Task { [weak self] in
      guard let self else { return }
      let now = await self.signIn.phase(now: self.clock())
      guard now != self.phase else { return }
      self.phase = now
      await self.recompose()
    }
  }

  /// One redraw, that many seconds from now. It is what takes a copied icon,
  /// a question and a green check back off the screen, and there is one of
  /// them at a time because there is one question at a time.
  func revert(after seconds: TimeInterval) {
    revertTimer?.invalidate()
    revertTimer = Timer.scheduledTimer(withTimeInterval: seconds, repeats: false) { _ in
      Task { @MainActor [weak self] in self?.changed() }
    }
  }

  // MARK: the hub

  /// The pairing, read once at start and again on every burst of changes in
  /// the hub. The scan walks a vault, so it runs off the main thread and the
  /// index it built arrives back on it.
  func readTheHub() {
    guard let hub = preferences.hub, !hub.isEmpty else {
      index = HubIndex.empty
      hubWatcher?.stop()
      hubWatcher = nil
      return
    }
    rescanTheHub(hub)
    guard hubWatcher == nil else { return }
    let watcher = HubWatcher(hub: hub) { [weak self] in
      Task { @MainActor [weak self] in self?.rescanTheHub(hub) }
    }
    watcher.start()
    hubWatcher = watcher
  }

  private func rescanTheHub(_ hub: String) {
    Task.detached(priority: .utility) { [weak self] in
      let scanned = HubIndex.scan(hub: hub)
      await MainActor.run { [weak self] in
        guard let self, scanned != self.index else { return }
        self.index = scanned
        Task { await self.recompose() }
      }
    }
  }

  // MARK: the bundle under the app

  private func bundleMoved(to state: BundleState) {
    guard state != bundle else { return }
    bundle = state
    Task { [weak self] in
      guard let self else { return }
      await self.recompose()
      await self.reopenIfTheBundleMoved()
    }
  }

  /// The app opens its own bundle again where every guard holds, and quits
  /// only once that open succeeded. Every guard is the policy's.
  private func reopenIfTheBundleMoved() async {
    guard let relauncher else { return }
    let after = await relauncher.reopen(
      bundle: bundle, updateRunning: updateIsRunning(), childAlive: aChildIsUp(),
      remembering: preferences)
    if after == .quit { quit() }
  }

  /// What this instance does about the one that launched it: wait for it to
  /// go, then quit if somebody else of this identifier is still up.
  private func settleTheLaunch() async {
    guard let relauncher else { return }
    if await relauncher.settle(arguments: places.arguments) == .quit { quit() }
  }

  /// Whether a `gdoc update` is live in the record, which may be writing the
  /// bytes a relaunch would run.
  private func updateIsRunning() -> Bool {
    Now.fold(events, now: clock(), dismissed: ui.dismissed, alive: alive)
      .jobs.contains { $0.command == Call.update.rawValue }
  }

  /// Whether a child this app started is still up. Quitting would take it
  /// with us.
  private func aChildIsUp() -> Bool {
    Call.allCases.contains { runner.isRunning($0) }
  }

  // MARK: the one signal out of this object

  func changed() {
    onChange()
  }
}
