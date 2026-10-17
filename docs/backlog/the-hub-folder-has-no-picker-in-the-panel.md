---
worth: later
where: presence/Sources/PresenceApp/Preferences.swift:105
added: 2026-10-17
---
# The hub folder has no picker in the panel

The M24 specification says Open md "reads the hub the person chose in HEALTH (a
folder, kept in preferences)". The preference is there and everything above it
works: `HubIndex` scans the folder, the index rescans on a file event, and the
md icon opens the paired note. What is missing is the choosing. HEALTH has no
hub row, and the screens file draws none either, so the only way to name the
folder is from a terminal:

    defaults write com.altery.gdoc.presence hubPath -string ~/path/to/hub

`docs/guide/presence.md` says exactly that, so nobody is stuck. But a colleague
who never opens a terminal has no route to the md icon at all, and the one
sentence in the specification reads as though they do.

Two ways out, both small:

- A HEALTH row "Hub" with the folder's last component and a picker behind it,
  which is one `NSOpenPanel` and a seventh name in the opener's source scan.
- A row that appears only while no hub is set, with the same picker, so a
  machine that has one keeps the short HEALTH list.

Worth `later` because the panel is Nail's alone until he shows it to anybody,
and the `defaults write` line is one line. It becomes `yes` the first time a
colleague installs the app.
