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

  private let calendar: Calendar
  private let tail: LogTail
  private let watcher: BundleWatcher?
  private var events: [Event] = []
  private var ticker: Timer?
  private var onChange: () -> Void = {}

  init(directory: URL, version: String, bundle bundlePath: URL?) {
    self.version = version
    self.calendar = Calendar.current
    self.tail = LogTail(directory: directory)
    self.watcher = bundlePath.map { BundleWatcher(bundle: $0, running: version) }
    self.glance = Fold.glance(
      [], now: Date(), calendar: Calendar.current, alive: Liveness.isAlive)
  }

  /// Start reading. The first pass has already run by the time this returns,
  /// so the first click opens a panel with the record in it.
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
    took(tail.poll())
  }

  func stop() {
    ticker?.invalidate()
    ticker = nil
    tail.stop()
    watcher?.stop()
  }

  /// What the menu bar shows.
  var glyph: GlyphState { Glyph.state(for: glance) }

  /// What the panel draws.
  var model: PanelModel {
    PanelModel.build(
      glance: glance, appVersion: version, bundle: bundle, calendar: calendar)
  }

  private func took(_ records: Records) {
    guard !records.events.isEmpty else {
      if !records.isEmpty { refold() }
      return
    }
    events.append(contentsOf: records.events)
    if events.count > eventCeiling {
      events.removeFirst(events.count - eventCeiling)
    }
    refold()
  }

  private func refold() {
    let before = glance
    glance = Fold.glance(
      events, now: Date(), calendar: calendar, alive: Liveness.isAlive)
    guard glance != before else { return }
    onChange()
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
