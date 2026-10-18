---
worth: soon
where: presence/Sources/PresenceApp/Opener.swift:97
added: 2026-10-18
---
# The Hub row's folder picker closes the panel it was pressed in

`SystemDesk.chooseFolder` runs `panel.runModal()` on an `NSOpenPanel`, which
is a window of this application's own and takes key while it is up. The panel
it was pressed in is a borderless `KeyPanel` whose delegate closes it the
moment it loses key, `presence/Sources/Presence/Panel.swift`:

```swift
func windowDidResignKey(_ notification: Notification) {
  close()
}
```

So the panel closes, `onClose` runs `session.closed()`, and `chooseTheHub`
then writes the hub, reads the notes and calls `changed()` into a window that
is gone: `Panel.refresh` returns at its `guard isOpen`. The choice is saved
and the md icons are lit, so nothing is lost and the next open shows all of
it. What the person sees is the panel disappearing under the press they made
in it, which is the one thing the function's own comment was written against
("a folder chosen and no icon lit is a row that looks like it did nothing").

This is the same mechanism as
`the-held-press-may-never-reach-a-rows-menu.md`, and a different window: the
measurement there found a popover is a *child* window at the panel's own
level and fired no resign-key event, which says nothing about a modal of the
application's own. Nobody has measured this one either, and the hand check is
one press: open the panel, press Hub, and say whether the panel is still
there when the picker comes up.

Why it is not fixed in place. Holding `close()` while `NSApp.modalWindow` is
up leaves the panel non-key after the picker goes, and a non-key panel fires
no further resign-key event: it would then stay open until the status item is
pressed again, which is the always-visible surface the design principles rule
out. A correct hold has to hand key back when the modal ends, so the panel
has to learn that a modal of somebody else's opened and closed. That is a new
seam between `Presence` and `PresenceApp`, with the source scan in
PresenceCoreTests and `Opener.swift`'s own page to follow, and which shape it
takes is Nail's call:

- the panel learns about a modal, holds its close and takes key back after;
- the Hub row closes the panel on purpose before the picker, and the guide
  says the panel steps aside while the system asks;
- the picker is raised from the shell rather than from the session, so the
  one room that owns the window owns the modal too.

A secondary thing in the same call: the app is `.accessory` and nothing
activates it before `runModal`, so the picker can come up behind the
frontmost application.
