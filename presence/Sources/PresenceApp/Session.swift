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
  var drawn: Drawn
  var bundle: BundleState = .same
  /// Whether a reopen was held by one of the two guards that clear by
  /// themselves, so the next tick asks again.
  ///
  /// `bundleMoved` asks once per change of `BundleState`, and `gdoc update`
  /// swaps the bundle before it writes its run's end, so the ask the change
  /// makes lands while the update is still live in the record almost every
  /// time. Without a second ask the person is left with "Quit and reopen",
  /// which is the fallback and not what the app decided to do:
  /// testAHeldReopenCostsNothingAndTheNextAskOpensTheNewInstance.
  var reopenHeld = false
  var index = HubIndex.empty
  var hubWatcher: HubWatcher?
  /// Which hub scan was started last. A scan that comes back and is not it
  /// is an older walk of the folder, and it is dropped.
  var hubScans = 0

  /// Whether the record has held a line from a gdoc at a schema this app does
  /// not read. It is sticky: the line is skipped and never kept, so the one
  /// thing left to say is that it was there.
  var newerGdoc = false

  /// The sign-in in flight, as the panel draws it, and whether the app's own
  /// `auth login` child is up.
  var phase: SignInPhase?
  var signingIn = false
  var signedInAt: Date?

  /// Whether an `update` child of the app's own is up, either list: plain
  /// `update` or `update --desktop`, which installs the release it writes
  /// the extension of. It is one flag for the two because two of them at
  /// once are two processes replacing one binary.
  ///
  /// It is the flag and not the runner's answer, because it is set by the
  /// press and the child is launched a moment later: the ring on the gdoc
  /// row is the actor's own standing, and this is what makes the clock tick
  /// fast enough to show it.
  var installing = false
  /// Whether the last Install extension of this process handed the extension
  /// to Claude Desktop. True until a run says otherwise, because the file
  /// beside the binary is what answers before any run of this app's.
  var extensionHandedOver = true

  /// The menu bar's check, and whether what the tail has delivered so far is
  /// the file as it stood at launch. Every line in it is a replay, and a
  /// replay flashes no check.
  var check = BarCheck()
  var replayed = false

  var ticker: Timer?
  var tickingEvery: TimeInterval = refoldEvery
  var checkTimer: Timer?
  var revertTimer: Timer?
  /// The signed-in banner's own five seconds. It is a slot of its own and
  /// not the revert's, because the banner is no question: a Copy link or a
  /// question raised inside those five seconds would otherwise take the
  /// banner's end away and leave it on the screen until some unrelated fact
  /// moved: testTheSignedInBannerTakesItselfOff.
  var bannerTimer: Timer?
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
    self.drawn = Drawn([], now: clock(), dismissed: [], calendar: calendar, alive: alive)
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
    bannerTimer?.invalidate()
    bannerTimer = nil
    tail.stop()
    watcher?.stop()
    hubWatcher?.stop()
    hubWatcher = nil
  }

  /// The panel is on the screen. It is the one moment the app asks gdoc
  /// about the machine, at most once a minute, because `help --json` carries
  /// gdoc's own daily check of what is published.
  ///
  /// It is also where a failed Open md is forgotten. The fault greys the
  /// icon, and a greyed icon cannot be pressed, so the click that would
  /// clear it can never happen: the panel going away is what ends it, and a
  /// hub chosen or a note renamed in between is a live icon on the next
  /// open: testAFaultGoesWhenThePanelOpensAgain.
  public func opened() {
    if !ui.noteFault.isEmpty {
      ui.noteFault.removeAll()
      changed()
    }
    Task { [weak self] in await self?.ask(.panelOpened) }
  }

  // MARK: what the shell draws

  /// What the menu bar shows, out of the same two folds the panel is built
  /// from, so the item and an open panel never say different things.
  public var bar: BarState {
    let moment = clock()
    return BarImage.state(
      now: Now.fold(events, now: moment, dismissed: ui.dismissed, alive: alive),
      sessions: Live.sessions(events, now: moment, alive: alive),
      facts: facts,
      checkShowing: check.isShowing(now: moment))
  }

  /// Where the menu bar's ring stands, which is the running job's own
  /// place in its steps. Nothing for every other state, which draws no
  /// ring at all.
  public var barProgress: Double? {
    BarImage.ringProgress(
      now: Now.fold(events, now: clock(), dismissed: ui.dismissed, alive: alive))
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
    checkTimer = redraw(after: BarCheck.lasts)
    return true
  }

  /// The folds taken again, and whether any of them moved. A screen that
  /// would be drawn the same way is not drawn again.
  ///
  /// It is the four folds the panel and the menu bar are actually built
  /// from, rather than one summary of them. Two things only those four
  /// notice: a `step` line, which moves the ring and the job card and
  /// nothing else: testAStepLineRedrawsTheJobCard, and a watch going quiet,
  /// which is three minutes passing and no line at all:
  /// testAQuietWatchRedrawsWithNoNewLine.
  ///
  /// The waits a signal has gone to are read back here as well, because what
  /// forgets one is its end arriving in the record.
  @discardableResult
  private func refolded() -> Bool {
    let before = drawn
    let pending = ui.pendingStop
    ui.pendingStop = stopper.pendingDocs(in: events)
    drawn = Drawn(
      events, now: clock(), dismissed: ui.dismissed, calendar: calendar, alive: alive)
    return drawn != before || ui.pendingStop != pending
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
  /// own version, the bundle under it, the hub, the switch, the extension,
  /// what the record last said about reaching Google, and the waits it may
  /// not signal.
  private func machine() -> Facts {
    Facts(
      versions: Versions(app: version),
      hub: preferences.hub,
      extensionState: extensionState(),
      network: Reachability.latest(in: events).map(NetworkFact.init(reachable:)),
      loginItem: loginItem.state,
      signIn: phase,
      pairing: index.notes,
      signedInAt: signedInAt,
      bundle: bundle,
      newerRecord: newerGdoc,
      unstoppable: stopper.unstoppable(in: events),
      // The press and not the actor's standing, so the extension's own run
      // closes both buttons too: the actor moves no version for that one.
      updating: installing)
  }

  /// Whether gdoc has ever written the Claude Desktop extension here. The
  /// one file the app looks at outside the record and its own plist, in the
  /// directory it already resolved the binary in, and the only thing it can
  /// say is that there is none: whether Desktop loaded one is not a
  /// question this app has a way to ask.
  /// testTheExtensionRowAppearsOnlyUntilGdocHasWrittenOne.
  ///
  /// A run of this process that wrote the file and could not hand it over is
  /// the one case the file itself would get wrong: `update --desktop` writes
  /// before it opens and leaves the file where it is when the open fails, so
  /// a machine with no Claude Desktop would end up with the row and its
  /// button gone and nothing said. The run's own answer wins while this
  /// process lives: testARunThatOpenedNothingLeavesTheRowWhereItWas.
  private func extensionState() -> ExtensionState {
    guard let file = runner.location.extensionFile else { return .unknown }
    guard extensionHandedOver else { return .notInstalled }
    return FileManager.default.fileExists(atPath: file) ? .written : .notInstalled
  }

  /// Whether a watch is open, which is what makes the token a question
  /// every ten minutes.
  func isLive() -> Bool {
    Live.sessions(events, now: clock(), alive: alive).contains(where: \.isOpen)
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
    reaskTheReopen()
    Task { [weak self] in await self?.ask(.timer) }
  }

  /// The reopen asked again, once the guard that held it is gone.
  ///
  /// The two cheap questions come before the fold, and the flag before both,
  /// so an ordinary tick asks nothing: a reopen is held for a few seconds in
  /// the life of an app. The policy's own ten-minute stamp bounds what
  /// follows, so this asks at most once per hold.
  private func reaskTheReopen() {
    guard reopenHeld, !aChildIsUp(), !updateIsRunning() else { return }
    reopenHeld = false
    Task { [weak self] in await self?.reopenIfTheBundleMoved() }
  }

  /// The sign-in's countdown, read off the child's own start. It is a value
  /// on this object rather than a question the panel asks, because the panel
  /// is drawn synchronously and the sign-in is an actor.
  private func readThePhase() {
    guard signingIn else { return }
    Task { [weak self] in
      guard let self else { return }
      let now = await self.signIn.phase(now: self.clock())
      guard self.tookThePhase(now) else { return }
      await self.recompose()
    }
  }

  /// The phase a read came back with, kept only while the sign-in it asked
  /// about is still running.
  ///
  /// The read hops to an actor and back, and the sign-in may finish inside
  /// that hop: its own closure puts the flag down and the row away. Writing
  /// the phase that read asked about back after that would leave a frozen
  /// countdown and a Cancel the child is already gone from, and nothing
  /// would clear it, because every later read returns at the guard above:
  /// testAPhaseThatCameBackAfterTheSignInEndedIsDropped.
  func tookThePhase(_ read: SignInPhase?) -> Bool {
    guard signingIn, read != phase else { return false }
    phase = read
    return true
  }

  /// One redraw, that many seconds from now. It is what takes a copied icon,
  /// a question and a green check back off the screen.
  ///
  /// The later of the two moments wins, because the slot is shared by two
  /// things that are not the same length: the copied check lasts a second
  /// and a question three. A Copy link pressed inside those three seconds
  /// would otherwise move the one redraw forward to its own second, and
  /// nothing would be scheduled for the moment the question ran out: the
  /// expired question would sit there until some fact moved, and a press on
  /// it would ask again instead of acting:
  /// testACopiedCheckDoesNotCutTheQuestionsRedrawShort.
  func revert(after seconds: TimeInterval) {
    let waiting = (revertTimer?.isValid ?? false) ? revertTimer?.fireDate : nil
    guard Session.takesTheSlot(waiting: waiting, due: Date().addingTimeInterval(seconds)) else {
      return
    }
    revertTimer?.invalidate()
    revertTimer = redraw(after: seconds)
  }

  /// Whether a redraw due at this moment takes the slot from the one already
  /// waiting in it. The later moment wins, and an empty slot is taken.
  static func takesTheSlot(waiting: Date?, due: Date) -> Bool {
    guard let waiting else { return true }
    return due >= waiting
  }

  /// One redraw when the signed-in banner's five seconds are up. The banner
  /// is the clock and no fact, and it is nobody's question, so it takes a
  /// slot of its own: testTheSignedInBannerTakesItselfOff.
  func retireTheBanner(after seconds: TimeInterval) {
    bannerTimer?.invalidate()
    bannerTimer = redraw(after: seconds)
  }

  /// One redraw, that many seconds from now, on a timer the caller holds.
  /// The three one-shots are the same line, so what differs between them is
  /// the slot and not what firing does.
  private func redraw(after seconds: TimeInterval) -> Timer {
    Timer.scheduledTimer(withTimeInterval: seconds, repeats: false) { _ in
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

  /// One scan, numbered, because only the newest of them may stand.
  private func rescanTheHub(_ hub: String) {
    hubScans += 1
    let scan = hubScans
    Task.detached(priority: .utility) { [weak self] in
      let scanned = HubIndex.scan(hub: hub)
      await MainActor.run { [weak self] in
        guard let self, self.tookTheScan(scan, scanned) else { return }
        Task { await self.recompose() }
      }
    }
  }

  /// The pairing a scan came back with, kept only where that scan is still
  /// the newest one asked for.
  ///
  /// A scan walks a vault off the main thread, so a burst of file events
  /// starts several and they come back in whatever order they finish in. A
  /// slow walk of the hub as it was, landing after a fast walk of the hub as
  /// it is, would leave Open md on yesterday's note until the next file
  /// event: testAnOlderHubScanDoesNotReplaceANewerOne.
  func tookTheScan(_ scan: Int, _ scanned: HubIndex) -> Bool {
    guard scan == hubScans, scanned != index else { return false }
    index = scanned
    return true
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
    let updateRunning = updateIsRunning()
    let childAlive = aChildIsUp()
    let after = await relauncher.reopen(
      bundle: bundle, updateRunning: updateRunning, childAlive: childAlive,
      remembering: preferences)
    // These two are the only guards that clear on their own: every other
    // one is about what the bundle on the disk is, and a bundle that is not
    // this app does not become it while the app waits. A hold on either is
    // the tick's to ask about again.
    reopenHeld = after == .keepTheReopenRow && (updateRunning || childAlive)
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
