// Everything the shell holds between two clicks.
//
// One tail over the record, the events it has delivered, and the fold of
// them at the moment somebody looks. Nothing is written anywhere: the app has
// no state of its own in this milestone, so there is nothing to persist and
// nothing to migrate.
//
// The watcher on the app's own bundle is here too, because the one question
// it answers is part of the same glance: whether the app running is the app
// on the disk.

import AppKit
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

/// How long the bundle watcher waits after a change before it reads the
/// plist. A rename arrives as several events and the plist is only worth
/// reading once they have stopped.
private let bundleDebounce: TimeInterval = 0.75

@MainActor
final class AppState {
  /// The version the app launched at, read off the bundle on the disk at
  /// launch and never from the cached bundle afterwards.
  let version: String

  private(set) var glance: Glance
  private(set) var bundle: BundleState = .same

  /// Whether the record has held a line from a gdoc at a schema this app does
  /// not read. It is sticky: the line is skipped and never kept, so the one
  /// thing left to say is that it was there.
  private(set) var newerGdoc = false

  private let calendar: Calendar
  private let tail: LogTail
  private let watcher: BundleWatcher?
  private var events: [Event] = []
  private var ticker: Timer?
  private var onChange: () -> Void = {}

  /// The menu bar's check, and whether what the tail has delivered so far is
  /// the file as it stood at launch. Every line in it is a replay, and a
  /// replay flashes no check.
  private var check = BarCheck()
  private var replayed = false
  private var checkTimer: Timer?

  init(directory: URL, version: String, bundle bundlePath: URL?) {
    self.version = version
    self.calendar = Calendar.current
    self.tail = LogTail(directory: directory)
    self.watcher = bundlePath.map { BundleWatcher(bundle: $0, running: version) }
    self.glance = Fold.glance(
      [], now: Date(), calendar: Calendar.current, alive: Liveness.isAlive)
  }

  /// Start reading. The tail's first pass runs inside start, and what it
  /// found reaches this object through the same route every later pass does,
  /// so the batches arrive in the order the file holds them. There is no
  /// second, synchronous read here: it would take the newer batch first and
  /// an end seen before its start puts one run in NOW and in TODAY at once.
  func start(onChange: @escaping () -> Void) {
    self.onChange = onChange
    tail.start { records in
      Task { @MainActor [weak self] in self?.took(records) }
    }
    watcher?.start { [weak self] state in
      guard let self, state != self.bundle else { return }
      self.bundle = state
      self.onChange()
    }
    ticker = Timer.scheduledTimer(withTimeInterval: refoldEvery, repeats: true) { _ in
      Task { @MainActor [weak self] in self?.refold() }
    }
    // The tail's first pass ran inside start, so its batch is already on this
    // actor's queue ahead of this one: everything delivered after it arrived
    // while the app was running.
    Task { @MainActor [weak self] in self?.replayed = true }
  }

  func stop() {
    ticker?.invalidate()
    ticker = nil
    checkTimer?.invalidate()
    checkTimer = nil
    tail.stop()
    watcher?.stop()
  }

  /// What the menu bar shows, out of the same two folds the panel is built
  /// from, so the item and an open panel never say different things.
  var bar: BarState {
    let moment = Date()
    return BarImage.state(
      now: Now.fold(events, now: moment, alive: Liveness.isAlive),
      sessions: Live.sessions(events, now: moment),
      facts: facts,
      checkShowing: check.isShowing(now: moment))
  }

  /// What the panel draws.
  ///
  /// The facts are the three this milestone's shell knows: its own version,
  /// the one the record last named, and whether the bundle under it moved.
  /// Task 22 hands over the polled ones and what the person did.
  var model: PanelModel {
    PanelModel.build(
      events: events, facts: facts, ui: UIState(), now: Date(), calendar: calendar,
      alive: Liveness.isAlive)
  }

  /// What the app knows about the machine, before any of it is asked for.
  var facts: Facts {
    Facts(
      versions: Versions(app: version, gdoc: glance.gdoc), bundle: bundle,
      newerRecord: newerGdoc)
  }

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
    if refolded() || said { onChange() }
  }

  /// A write that read itself back starts the menu bar's check, for five
  /// seconds, and only where its line arrived live. The one-shot timer is
  /// what takes the check off again: the five-second refold would not,
  /// because a glance that did not change redraws nothing.
  private func noticedAWrite() -> Bool {
    let moment = Date()
    let fold = Now.fold(events, now: moment, alive: Liveness.isAlive)
    guard check.saw(fold.lastVerified, live: replayed, at: moment) else { return false }
    checkTimer?.invalidate()
    checkTimer = Timer.scheduledTimer(withTimeInterval: BarCheck.lasts, repeats: false) { _ in
      Task { @MainActor [weak self] in self?.onChange() }
    }
    return true
  }

  private func refold() {
    if refolded() { onChange() }
  }

  /// The fold taken again, and whether it moved. A glance that did not change
  /// redraws nothing.
  private func refolded() -> Bool {
    let before = glance
    glance = Fold.glance(
      events, now: Date(), calendar: calendar, alive: Liveness.isAlive)
    return glance != before
  }
}

/// Whether the bundle under the running app is still the bundle it launched
/// from.
///
/// gdoc update stages the new app beside the old one and swaps it by rename,
/// which is two changes in the parent directory and none the running code can
/// see. So one source watches the bundle itself and one its parent, both
/// debounced, and the plist is read off the disk with
/// PropertyListSerialization every time. Bundle.main is never asked: it
/// cached what it read at launch, which is the one answer this is looking for
/// a change against.
@MainActor
final class BundleWatcher {
  private let bundle: URL
  private let running: String
  private var sources: [DispatchSourceFileSystemObject] = []
  private var pending: DispatchWorkItem?
  private var onChange: ((BundleState) -> Void)?

  init(bundle: URL, running: String) {
    self.bundle = bundle
    self.running = running
  }

  func start(onChange: @escaping (BundleState) -> Void) {
    self.onChange = onChange
    watch(bundle)
    watch(bundle.deletingLastPathComponent())
  }

  func stop() {
    pending?.cancel()
    pending = nil
    for source in sources { source.cancel() }
    sources = []
    onChange = nil
  }

  /// The version the plist on the disk names now, or nothing where there is
  /// no plist to read.
  static func versionOnDisk(of bundle: URL) -> String? {
    let plist = bundle.appendingPathComponent("Contents/Info.plist")
    guard let data = try? Data(contentsOf: plist) else { return nil }
    guard
      let values = try? PropertyListSerialization.propertyList(
        from: data, options: [], format: nil) as? [String: Any]
    else { return nil }
    return values["CFBundleShortVersionString"] as? String
  }

  private func watch(_ url: URL) {
    let descriptor = open(url.path, O_EVTONLY)
    guard descriptor >= 0 else { return }
    let source = DispatchSource.makeFileSystemObjectSource(
      fileDescriptor: descriptor,
      eventMask: [.write, .delete, .rename, .revoke],
      queue: .main)
    source.setEventHandler { [weak self] in self?.settle() }
    source.setCancelHandler { close(descriptor) }
    sources.append(source)
    source.resume()
  }

  /// One read per burst of events, not one per event.
  private func settle() {
    pending?.cancel()
    let work = DispatchWorkItem { [weak self] in
      guard let self else { return }
      let state = BundleVersion.check(
        running: self.running, onDisk: BundleWatcher.versionOnDisk(of: self.bundle))
      self.onChange?(state)
    }
    pending = work
    DispatchQueue.main.asyncAfter(deadline: .now() + bundleDebounce, execute: work)
  }
}
