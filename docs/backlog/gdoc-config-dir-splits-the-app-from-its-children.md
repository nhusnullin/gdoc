---
worth: later
where: presence/Sources/PresenceApp/Places.swift:85
added: 2026-10-17
---
# GDOC_CONFIG_DIR splits the app from its children

The app reads one directory and runs children that read another, where
somebody has set `GDOC_CONFIG_DIR` and launched the app from that terminal.

`Places.recordDirectory` honours the variable, which is M23's decision and its
reason: a person pointing gdoc at another directory can point the app at the
same one. `Runner.childEnvironment` drops it, which is M24a's decision and its
reason: the app asks about the token the person's own terminal uses, and a
variable inherited from a launch agent would point the child somewhere else
without saying so. Both are written down, in `docs/guide/presence.md`, in the
M24 spec's acceptance table and in CLAUDE.md's "four variables and no stdin".

They disagree only in one case, and in that case the panel is half right: NOW,
DOCUMENTS and TODAY come from the directory the variable names, while the
account, the mark and the sign-in come from `~/.config/gdoc-agent`. Launched
from Finder or at login, which is every ordinary launch, there is no variable
and the two agree.

Either way out is one rule and a decision, not a refactor:

- Pass `GDOC_CONFIG_DIR` through to children where the parent carries one.
  That is a fifth variable, so CLAUDE.md's invariant, the guide, the spec row
  and `testTheChildEnvironmentIsTheFourVariables` all move with it.
- Or have `Places` ignore it, as the runner does, so the app is always about
  `~/.config/gdoc-agent`. Then a person with a custom directory has no app,
  which is honest but is less than they have today.

Worth `later`: nobody has reported it, because nobody launches the menu bar app
from a terminal with that variable set.
