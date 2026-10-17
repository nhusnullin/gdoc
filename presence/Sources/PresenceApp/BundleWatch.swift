// Whether the bundle under the running app is still the bundle it launched
// from.
//
// gdoc update stages the new app beside the old one and swaps it by rename,
// which is two changes in the parent directory and none the running code can
// see. So one source watches the bundle itself and one its parent, both
// debounced, and the plist is read off the disk with
// PropertyListSerialization every time. Bundle.main is never asked: it cached
// what it read at launch, which is the one answer this is looking for a
// change against.
//
// The reading is `BundleVersion.check`, which is the core's and pure. This
// file is the two sources and the debounce, and it decides nothing.

import Foundation
import PresenceCore

/// How long the watcher waits after a change before it reads the plist. A
/// rename arrives as several events and the plist is only worth reading once
/// they have stopped.
let bundleDebounce: TimeInterval = 0.75

@MainActor
public final class BundleWatcher {
  private let bundle: URL
  private let running: String
  private var sources: [DispatchSourceFileSystemObject] = []
  private var pending: DispatchWorkItem?
  private var onChange: ((BundleState) -> Void)?

  public init(bundle: URL, running: String) {
    self.bundle = bundle
    self.running = running
  }

  public func start(onChange: @escaping (BundleState) -> Void) {
    self.onChange = onChange
    watch(bundle)
    watch(bundle.deletingLastPathComponent())
  }

  public func stop() {
    pending?.cancel()
    pending = nil
    for source in sources { source.cancel() }
    sources = []
    onChange = nil
  }

  /// The version the plist on the disk names now, or nothing where there is
  /// no plist to read. One read of one file, so it belongs to no actor.
  public nonisolated static func versionOnDisk(of bundle: URL) -> String? {
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
