// gdoc presence: the app in the menu bar.
//
// It reads one directory of lines gdoc appended and shows what is happening
// now and what happened today. It never writes to that directory, never
// reaches the wire, never starts another program and never touches the token.
// Everything it knows came out of a file gdoc wrote, and every line in that
// file is treated as something a person could have planted there.
//
// There is no Dock tile and no window: an accessory application with one
// status item, which is the whole of it.

import AppKit
import PresenceUI
import SwiftUI

/// The shell. It holds the three objects and wires the one signal between
/// them: the record changed, so the glyph and an open panel are redrawn.
@MainActor
final class Shell: NSObject, NSApplicationDelegate {
  private var state: AppState?
  private var item: StatusItem?
  private let panel = Panel()

  func applicationDidFinishLaunching(_ notification: Notification) {
    let bundle = ownBundle()
    let state = AppState(
      directory: recordDirectory(),
      version: bundle.flatMap(BundleWatcher.versionOnDisk(of:)) ?? "0.0.0",
      bundle: bundle)
    let item = StatusItem { [weak self] in self?.clicked() }
    self.state = state
    self.item = item

    state.start { [weak self] in self?.changed() }
    changed()
  }

  func applicationWillTerminate(_ notification: Notification) {
    state?.stop()
  }

  /// The one click: the panel opens, or closes if it is already open.
  private func clicked() {
    guard let state else { return }
    panel.toggle(view(of: state), under: item?.button)
  }

  /// The record moved. The glyph follows it always; the panel only while it
  /// is open.
  private func changed() {
    guard let state else { return }
    item?.show(state.glyph)
    panel.refresh(view(of: state))
  }

  private func view(of state: AppState) -> some View {
    PanelView(
      model: state.model,
      actions: PanelActions(perform: { kind, _ in
        // The one action this milestone's shell performs. Tasks 17 to 21
        // give the rest of the kinds a body, and Task 22 wires them here.
        if kind == .quit { NSApp.terminate(nil) }
      }))
  }

  /// The app's own bundle, when it is running from one. A bare executable
  /// from `swift run` has no plist to compare against, and then the reopen
  /// row is never shown.
  private func ownBundle() -> URL? {
    let url = Bundle.main.bundleURL
    return url.pathExtension == "app" ? url : nil
  }

  /// Where gdoc keeps the record. GDOC_CONFIG_DIR first, which is the one
  /// override the binary itself honours, so a person pointing gdoc at another
  /// directory can point the app at the same one.
  private func recordDirectory() -> URL {
    let named = ProcessInfo.processInfo.environment["GDOC_CONFIG_DIR"] ?? ""
    let base =
      named.isEmpty
      ? FileManager.default.homeDirectoryForCurrentUser
        .appendingPathComponent(".config/gdoc-agent")
      : URL(fileURLWithPath: named)
    return base.appendingPathComponent("presence")
  }
}

let application = NSApplication.shared
let shell = Shell()
application.delegate = shell
application.setActivationPolicy(.accessory)
application.run()
