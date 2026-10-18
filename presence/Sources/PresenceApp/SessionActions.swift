// What a press in the panel means.
//
// One function over the eighteen kinds, and nothing else in the app turns a
// press into anything. Each case is one line of the specification's decision
// table: the two icons that open something, the one that copies, the one that
// dismisses a card, the one signal, sign in and sign out, the update and the
// extension, the switch, quit, and the three words of a document row's own
// menu.
//
// Nothing here judges. A dismissal hides a card and rewrites no row, a stop
// is refused by the process check and not by this file, and a sign-out stops
// the watches before the token goes because a watch that lost its token stops
// answering without saying so.
//
// The questions are the two the panel asks, and both of them are a second
// press of the same kind while the question is up: a hold's Dismiss and the
// Account row's Sign out. A question waits for its answer or for the panel to
// close. Everything else happens at once.

import Foundation
import PresenceCore
import PresenceUI

extension Session {
  /// One press, as the panel hands it over: the kind, and the document,
  /// hold or run it was about.
  public func perform(_ kind: ActionKind, _ target: String?) {
    switch kind {
    case .openDoc:
      opener.openDocument(target)
      sawTheStrip()
    case .openNote:
      openTheNote(target)
      sawTheStrip()
    case .openChat:
      opener.continueWithClaude()
      sawTheStrip()
    case .copyLink:
      copyTheLink(target)
    case .dismiss:
      dismiss(target)
    case .stopLive:
      stopLive(target)
    case .signIn:
      signInWithGoogle()
    case .signOut:
      askThenSignOut()
    case .cancelSignIn:
      Task { [weak self] in await self?.signIn.cancel() }
    case .copySignInLink:
      copyTheSignInLink()
    case .update:
      install()
    case .installExtension:
      installTheExtension()
    case .openLoginItems:
      loginItem.openSettings()
    case .toggleLoginItem:
      switchLaunchAtLogin()
    case .quit:
      quit()
    case .keep:
      putTheQuestionAway()
    case .pinDoc:
      pin(target)
    case .unpinDoc:
      unpin(target)
    case .hideDoc:
      hide(target)
    }
  }

  /// One section opened or closed. A refold never closes it: what is open is
  /// the person's and lives out here.
  public func toggle(_ list: PanelList) {
    ui = ui.toggling(list)
    changed()
  }

  // MARK: the four icons on a row

  /// The four labels under the first document row go after the first click
  /// on any icon and never come back.
  private func sawTheStrip() {
    guard !ui.stripSeen else {
      changed()
      return
    }
    ui = ui.afterAnIconClick()
    preferences.stripSeen = true
    changed()
  }

  /// Open the note, and keep why it did not open where it did not.
  ///
  /// A click that goes nowhere has to say so: the index is rebuilt on a file
  /// event, so a note somebody deleted usually greys itself, but a note
  /// Obsidian refuses is a fault only the opening knows about. The icon
  /// carries it for as long as the panel is up, and `opened()` is what takes
  /// it away: a greyed icon cannot be pressed, so a second click is not a
  /// route out of it: testAFaultGoesWhenThePanelOpensAgain.
  private func openTheNote(_ doc: String?) {
    guard let doc else { return }
    switch opener.openNote(document: doc, hub: preferences.hub, pairing: index.notes) {
    case .success:
      ui.noteFault.removeValue(forKey: doc)
    case .failure(let fault):
      ui.noteFault[doc] = fault
    }
  }

  private func copyTheLink(_ doc: String?) {
    guard let doc, opener.copyLink(doc) != nil else { return }
    ui.copied = doc
    ui.copiedAt = clock()
    revert(after: Timing.doneFlash)
    sawTheStrip()
  }

  /// The sign-in link, as the line under the Account row copies it. It was
  /// held to Google's own sign-in address before it became that line, so
  /// what is copied is what was opened.
  private func copyTheSignInLink() {
    guard let url = phase?.url else { return }
    desk.copy(url)
    changed()
  }

  // MARK: the two words of a row's menu

  /// Pin one document, with the title the list is drawing for it.
  ///
  /// The title is kept because the pin outlives the lines: fourteen days on,
  /// or two month files on, the record says nothing about that document and
  /// the pin is all the row has. It is the one thing of somebody's document
  /// this app writes down, and the decision of 2026-10-18 is where that is
  /// written.
  private func pin(_ doc: String?) {
    guard let doc else { return }
    preferences.pin(doc, title: lastTitle(of: doc), at: clock())
    ui.pinned = preferences.pins
    changed()
  }

  private func unpin(_ doc: String?) {
    guard let doc else { return }
    preferences.unpin(doc)
    ui.pinned = preferences.pins
    changed()
  }

  /// Hide one document from DOCUMENTS. Its rows in TODAY stay, because the
  /// log is the log, and the next action gdoc finishes on it brings the row
  /// back: nothing here unhides one.
  private func hide(_ doc: String?) {
    guard let doc else { return }
    preferences.hide(doc, title: lastTitle(of: doc), at: clock())
    ui.hidden = preferences.hides
    changed()
  }

  /// The title the list is drawing for one document, which is the latest one
  /// the record carries. Nothing where the row is the id itself.
  private func lastTitle(of doc: String) -> String? {
    drawn.documents.rows.first { $0.doc == doc }?.title
  }

  // MARK: the cards

  /// Hide one card. A hold asks first and a stuck write does not: a hold
  /// lives on in the chat until its own `until` after the card goes, which
  /// is worth one question, and a write that failed is over.
  private func dismiss(_ key: String?) {
    guard let key else { return }
    if isAHold(key), !ui.isAsking(.dismissHold(key), now: clock()) {
      asksFirst(.dismissHold(key))
      return
    }
    preferences.dismiss(key, at: clock())
    ui.dismissed.insert(key)
    putTheQuestionAway()
  }

  private func isAHold(_ id: String) -> Bool {
    Now.fold(events, now: clock(), dismissed: ui.dismissed, alive: alive)
      .holds.contains { $0.id == id }
  }

  // MARK: the one signal

  /// Stop one watch: the check, then one SIGINT, both the stopper's. A
  /// signal that went leaves the row a ring until its end is in the record.
  private func stopLive(_ doc: String?) {
    guard let doc else { return }
    if case .signalled = stopper.stop(doc: doc, in: events) {
      ui.pendingStop.insert(doc)
    }
    changed()
  }

  // MARK: the token

  private func signInWithGoogle() {
    guard !signingIn else { return }
    signingIn = true
    retick()
    Task { [weak self] in
      guard let self else { return }
      let result = await self.signIn.run()
      self.signingIn = false
      self.phase = nil
      self.retick()
      if case .signedIn(_, let at) = result {
        self.signedInAt = at
        // The banner is five seconds of the clock and no fact of its own,
        // so nothing else would take it off: a refold that did not move
        // redraws nothing, and the next fact may be minutes away. It is its
        // own slot and not the revert's, which a copied icon or a question
        // inside those five seconds would take:
        // testTheSignedInBannerTakesItselfOff.
        self.retireTheBanner(after: SignedInBanner.life)
      }
      await self.ask(.changed)
    }
  }

  /// Sign out, once. The first press is the question, which waits for its
  /// answer or for the panel to close; the second is the sign-out.
  private func askThenSignOut() {
    guard ui.isAsking(.signOut, now: clock()) else {
      asksFirst(.signOut)
      return
    }
    putTheQuestionAway()
    Task { [weak self] in await self?.signOut() }
  }

  private func signOut() async {
    // The watches first: each of them is a wait that would go on asking
    // Google with a token that is no longer there.
    for session in Live.sessions(events, now: clock(), alive: alive) where session.isOpen {
      if case .signalled = stopper.stop(doc: session.doc, in: events) {
        ui.pendingStop.insert(session.doc)
      }
    }
    changed()
    _ = try? await runner.run(.authLogout)
    signedInAt = nil
    await ask(.changed)
    // One redraw whatever the facts did: the banner's clock moved, and a
    // read that came back the same (a logout that was refused) is still the
    // moment the row goes back from its question to its icon.
    changed()
  }

  // MARK: the two runs the panel offers

  /// `gdoc update`, with no flag: what the person would have typed. The row
  /// draws a ring until the child's end, and the version beside it moves
  /// only once `help --json` says what is installed.
  private func install() {
    guard !installing else { return }
    installing = true
    retick()
    Task { [weak self] in
      guard let self else { return }
      _ = await self.updates.update()
      self.installing = false
      self.retick()
      await self.ask(.changed)
    }
  }

  /// `gdoc update --desktop`. The row it is pressed from stays where the run
  /// did not hand the extension to Claude Desktop, because the file the row
  /// otherwise reads is written before the handing over and left behind when
  /// it fails: testARunThatOpenedNothingLeavesTheRowWhereItWas.
  ///
  /// It takes the same slot the update takes, because both argv lists are
  /// `gdoc update` and `--desktop` installs the release it writes the
  /// extension of: two of them at once are two processes staging and
  /// renaming one binary, which can leave the rollback copy pointing at the
  /// new binary instead of the old one. The runner's own rule is per argv
  /// list and would let these two through:
  /// testTheExtensionAndTheUpdateCannotRunAtOnce.
  ///
  /// The redraw is unconditional, unlike a refold's: a run a person started
  /// may end with no fact moved at all, and the screen is still owed the
  /// end of it.
  private func installTheExtension() {
    guard !installing else { return }
    installing = true
    retick()
    Task { [weak self] in
      guard let self else { return }
      self.extensionHandedOver = await self.updates.installExtension()
      self.installing = false
      self.retick()
      await self.recompose()
      self.changed()
    }
  }

  // MARK: the switch

  private func switchLaunchAtLogin() {
    // macOS took the registration and is waiting for the person to allow
    // it, which the row draws as an off switch because the item is not
    // running yet. The choice is written down already, so the one thing a
    // press may do here is open the pane that allows it: registering again
    // is a second notice in somebody's menu bar for an answer nobody
    // changed, and LoginItem's own page says it is never retried:
    // testTheSwitchInTheApprovalStateOpensThePaneAndRegistersNothingAgain.
    guard facts.loginItem != .needsApproval else {
      loginItem.openSettings()
      return
    }
    _ = loginItem.set(facts.loginItem == .on ? .off : .on)
    Task { [weak self] in await self?.recompose() }
  }

  // MARK: the question on the screen

  private func asksFirst(_ question: PanelQuestion) {
    ui.question = question
    ui.questionAt = clock()
    changed()
  }

  private func putTheQuestionAway() {
    ui.question = nil
    ui.questionAt = nil
    changed()
  }
}
