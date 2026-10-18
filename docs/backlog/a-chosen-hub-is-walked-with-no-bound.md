---
worth: soon
where: presence/Sources/PresenceCore/HubIndex.swift:70
added: 2026-10-18
---
# A chosen hub is walked with no bound

`HubIndex.scan` enumerates the whole tree under the hub with no depth, no
file count and no time ceiling. It skips dot folders and reads up to 64 KB of
every `.md` it finds, and that is the whole of the restraint. `HubWatcher`
then runs an FSEvents stream over the same root with file events, and each
debounced burst runs `rescanIfWatched`, which is a fresh full walk;
`rescanTheHub` numbers its scans so a stale answer is dropped, but it never
cancels the walk that produced it, so overlapping `Task.detached` walks pile
up.

Until this branch the hub could only be named with `defaults write`, which is
a path somebody typed on purpose. The new Hub row hands the person an
`NSOpenPanel` rooted anywhere, and `~`, `/` or a mounted volume is one click
away. The result is a menu bar app walking a file system on a utility queue
and re-walking it on nearly every file change on the machine, with nothing in
the panel saying so.

What a fix needs is a number, which is why this is not a review's to pick: a
ceiling on directories or files visited, a depth, or a pre-count before
`preferences.hub` is written, and a refusal or a warning on a folder over it.
Cancelling the previous walk before starting the next is the smaller half and
needs no decision.
