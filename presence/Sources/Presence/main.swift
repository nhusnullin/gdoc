// gdoc presence: the app in the menu bar.
//
// It reads one directory of lines gdoc appended and shows what is happening
// now and what happened today. It runs six argv lists of gdoc's own at a path
// it resolved itself, sends one SIGINT to a wait a person asked it to stop,
// and never reaches the wire, never writes into that directory and never
// touches the token itself. Everything it knows came out of a file gdoc
// wrote, and every line in that file is treated as something a person could
// have planted there.
//
// There is no Dock tile and no window: an accessory application with one
// status item, which is the whole of it.
//
// This file is the shell, and it holds no reading and no wording. The session
// in PresenceApp knows the record, the children, the stores and what a press
// means; here there is the status item, the panel's window, and the one
// signal between them: something changed, so the glyph and an open panel are
// redrawn.

import AppKit
import PresenceApp
import PresenceUI
import SwiftUI

@MainActor
final class Shell: NSObject, NSApplicationDelegate {
  private var session: Session?
  private var item: StatusItem?
  private let panel = Panel()

  func applicationDidFinishLaunching(_ notification: Notification) {
    let session = Session(places: .ofThisMachine(), quit: { NSApp.terminate(nil) })
    let item = StatusItem { [weak self] in self?.clicked() }
    self.session = session
    self.item = item

    session.start { [weak self] in self?.changed() }
    changed()
  }

  func applicationWillTerminate(_ notification: Notification) {
    session?.stop()
  }

  /// The one click: the panel opens, or closes if it is already open. An
  /// open is also the one moment the app asks gdoc about this machine.
  private func clicked() {
    guard let session else { return }
    panel.toggle(view(session.screen.model, of: session), under: item?.button)
    if panel.isOpen { session.opened() }
  }

  /// Something changed. The glyph follows it always; the panel only while it
  /// is open.
  ///
  /// One `screen` and not the glyph's two folds and then the panel's four:
  /// every fold walks every line the app holds, and these two surfaces are
  /// over one record at one moment.
  private func changed() {
    guard let session else { return }
    let screen = session.screen
    item?.show(screen.bar, progress: screen.progress)
    panel.refresh(view(screen.model, of: session))
  }

  private func view(_ model: PanelModel, of session: Session) -> some View {
    PanelView(
      model: model,
      actions: PanelActions(perform: session.perform, toggle: session.toggle))
  }
}

let application = NSApplication.shared
let shell = Shell()
application.delegate = shell
application.setActivationPolicy(.accessory)
application.run()
