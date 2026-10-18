---
worth: soon
where: presence/Sources/PresenceUI/PanelView.swift:355
added: 2026-10-18
---
# The held press may never reach a document row's menu

`RowMenu` has two ways in. The secondary click is `.contextMenu`, which tracks
an `NSMenu` and is fine. The held press sets `held = true` and presents a
SwiftUI `.popover`. A popover is its own window, and the panel it would sit
over is a borderless `KeyPanel` whose delegate closes the panel the moment it
loses key, `presence/Sources/Presence/Panel.swift`:

```swift
func windowDidResignKey(_ notification: Notification) {
  close()
}
```

If the popover's window takes key, the panel resigns key, `close()` runs, and
the popover goes with the host view it hung from. The words would then be
unreachable by the one gesture `docs/guide/presence.md` tells a colleague to
use: "hold a press on it for half a second".

What was measured, 2026-10-18, by a throwaway bundled app that rebuilt the
panel's window exactly, borderless, `.nonactivatingPanel`, `.popUpMenu`, the
same delegate, and presented the same `.popover`:

| Asked | Answer |
|---|---|
| is the popover drawn under the panel | no, `_NSPopoverWindow` is a child window of the panel at the same level 101 |
| did the panel resign key when the popover opened | no event fired |
| can the popover's window take key | `canBecomeKey` is `true` |

**The main claim is not measured.** The probe never got the app active, so the
panel was never key and the one question, whether a key panel resigns key to
its own popover, was not reached: `NSApp.activate` did nothing from a terminal
session, bundled and ad-hoc signed alike. So the mechanism stands as a risk and
not as a fact, and the thing that would settle it is a hand check on the built
app, which is Nail's: open the panel, hold a press on a document row for half a
second, and say whether the two words appear and stay.

Three answers if the hand check says the menu never opens:

- drop the popover, make the secondary click the one way in, and change the
  guide and the plan's sentence to match;
- open a real `NSMenu` with `popUp(positioning:at:in:)` from the panel's own
  view on long press, so no second window is created;
- hold `windowDidResignKey`'s `close()` while a row menu is up, which means the
  panel learns about a menu it does not own.

`worth: soon` because the guide already promises the gesture. Nothing is
reported as broken: the secondary click works either way, and no test covers
the gesture, `testTheRowMenuSaysPinOrUnpinAndHide` asserts the model's `menu`
array alone.

A smaller thing in the same place: the long press is attached to the whole
`RowView`, action strip included, so a slow click on Copy link both copies the
link and raises the menu. Whichever answer wins, the gesture belongs on the
row's title area and not over its four icons.
