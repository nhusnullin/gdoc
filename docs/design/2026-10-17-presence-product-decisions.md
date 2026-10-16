# The presence: product decisions

2026-10-17. After the screens pass
([2026-10-16-presence-votes.md](2026-10-16-presence-votes.md), round two)
Nail asked for a product owner's view of the whole journey: overflow, time
horizon, ordering, concurrency, empty states, the action set, discoverability
and interruption. A product-owner seat decided these without asking back,
marking guesses. Nail has not reviewed them yet; the screens pass that
follows draws them so he can.

Scale assumption behind everything: one user, 15 to 20 documents a month, a
few hundred log lines a day at peak. The panel is a glance, not a tool.
Anything that needs reading goes to Claude Code or the hub.

## Overflow

1. DOCUMENTS shows the last 5 by last action; TODAY shows the last 8 (guess
   on the numbers). Nail works on one to three documents a day.
2. "Show more" expands inline inside the same dropdown, and the expanded
   section scrolls inside itself, with the whole panel capped at 600 px.
   The panel is a popover-style window, not a system menu, so it scrolls
   inside; arrow-scrolling menus would break the Stop button and icon rows.
3. No search and no filter in the panel. Twenty documents fit one expanded
   scroll. Search over history is a Claude Code or Obsidian question.
4. NOW never truncates. It holds a handful of items by nature and is why the
   panel exists.

## Time horizon

5. A document stays in DOCUMENTS for 14 days after its last action, then
   drops out of the panel (guess on 14).
6. TODAY rolls at local midnight into a collapsed "Yesterday" header, one
   line: "Yesterday: 12 actions, 2 documents". Click expands it inline.
   Nothing older than yesterday appears in the panel.
7. The log lives as one append-only JSONL file beside the token, written by
   the binary, not by the menu bar app. The binary already prints every fact
   as JSON; appending the same envelope is the smallest honest change, and it
   survives the panel being closed or not running. The hub is not the place:
   the log names documents from other people's folders and sessions that
   never touched a note. The app only tails the file. (This is a technology
   decision Nail deferred; it is recorded here, not decided.)
8. The file rotates monthly and the app never prunes it. Facts are cheap,
   deletion is a decision.

## Ordering and grouping

9. DOCUMENTS sorts by last action, newest first. No state ordering, no
   pinning. State lives in NOW; duplicating it would give two sort rules and
   a document that jumps when a job ends.
10. A document with a running job or hold appears in both NOW and DOCUMENTS.
    NOW is "what needs me", DOCUMENTS is "where do I go". The DOCUMENTS row
    carries a small inline badge so the two agree.
11. TODAY groups by time only, newest first, one line per action with the
    document in grey. Grouping by document would hide interleaving, and
    interleaving is the thing Nail cannot see anywhere else.

## Many at once

12. NOW order: holds first, then stuck, then running jobs, then live
    watches. Waiting on Nail beats waiting on Google; a live watch is
    background.
13. Within a class, oldest first. The one that has waited longest is the one
    most likely forgotten.
14. Two live watches are two cards, each with its own Stop. No "Stop all".
15. Two holds show both cards; the menu bar badge is the single pause glyph,
    no count.

## Empty and first-run states

16. First launch: NOW and TODAY say "Nothing yet". DOCUMENTS shows one line:
    "Publish a note or review a document from Claude Code and it will appear
    here". HEALTH is the point of the first open.
17. Not signed in: the account row says "Not signed in" with a Sign in button
    that runs the login in a terminal window and shows the URL the binary
    prints as a copyable line. The app never opens a browser either. Badge:
    exclamation.
18. gdoc missing from PATH: HEALTH shows "gdoc not found" with the install
    one-liner as copyable text and the path it looked at. Everything else
    greyed, badge exclamation. The app does not install the binary.
19. Desktop extension not installed: a HEALTH row with Install, which runs
    the desktop update. The AI icon on rows is greyed with a tooltip. Not a
    badge: nothing is broken.
20. Hub not found: a Hub row in HEALTH with the vault path, or "not set" with
    Choose. The note icon on rows is greyed.
21. Document not in the hub: note icon greyed, tooltip "No note is paired
    with this document". Expected for restyles and annotations; never a
    badge.
22. Google Doc deleted or unreachable: the row stays its 14 days with the
    title struck through and a tooltip naming the 404. Open in Docs greyed,
    copy link still works. A fact, not an error: Nail may have trashed it.

## The action set

23. Open in Google Docs: opens the URL with the tab of the last action when
    known. Always available unless 22.
24. Open md: opens the paired note through Obsidian's URL scheme, vault and
    file. Vault set but file missing: greyed, tooltip "Note moved or
    deleted".
25. Continue the AI session is one action: resume the last Claude Code
    session that touched this document if the app knows one; otherwise open
    Claude Desktop with the document link prefilled. Right-click offers "New
    Claude Code session" as the one alternative. (Guess that a session id is
    recoverable.)
26. A document touched only from Desktop chat has no Claude Code session; the
    same icon opens Claude Desktop. One icon, one meaning: talk to the AI
    about this document.
27. Copy link copies the plain Docs URL. Never markdown, never a title.

## Discoverability

28. Tooltips after half a second on every icon, plus a one-time strip under
    the first DOCUMENTS row, "Docs · Note · AI · Link", gone after the first
    click on any icon.

## Notifications and interruption

29. A colleague's new comment is not surfaced anywhere in the menu bar. Nail
    voted no on an inbox; a count in a row is an inbox with fewer pixels.
30. Stop live does not confirm. Stopping is cheap and reversible. A confirm on
    the easy action teaches clicking through on the hard one.
31. Release without writing confirms inline: the button becomes "Release?
    Nothing will be written" for three seconds. The one irreversible choice
    in the panel.

## Unasked

32. Launch at login is off by default, a toggle in HEALTH.
33. The app never talks to Google. It reads the log and HEALTH asks the
    binary. Otherwise it would need a client, and the guard owns the wire.
34. Right-click on the menu bar item opens the same dropdown. One surface.
35. `verified: false` shows in TODAY as a hollow check, never as a failure.
    The write happened; the facts say so.
36. Quit lives at the bottom of HEALTH.

## Screens the mockup set must add

Expanded DOCUMENTS with the inline scroll; Yesterday collapsed and expanded;
not signed in; gdoc not found; a crowded NOW with two holds, a stuck job, a
publish and two live watches in order; first launch; a row with greyed icons
and their tooltips; the release confirmation; a deleted document row; the
first-use icon strip; the right-click on the AI icon.

## Login and logout

Asked by Nail the same evening. Standing rule: the app never touches the
token or Google. Every auth action is a call to the binary, and the panel
shows what the binary printed.

37. Sign in lives in HEALTH, on the account row, and nowhere else. With no
    token present, HEALTH is pinned open at the top and the other sections
    collapse below it, because nothing else works yet.
38. Clicking Sign in runs `gdoc auth login` as a child process and opens the
    printed URL in the default browser; the same URL is shown as a copyable
    line under the button. The binary still runs no program; the app is a
    different program with a different job, and opening the one URL the
    binary chose is what the skill asks the person to do today. Both routes,
    because the browser may be in the wrong Google profile.
39. While waiting the row reads "Waiting for Google… (2:40 left)" with
    Cancel, badge ring. Cancel kills the child and the loopback listener with
    it. Timeout three minutes (guess). After it: "Google did not answer. Try
    again", no badge.
40. Wrong Google account: the app cannot know what wrong is, so it does not
    refuse. After login it shows the email large for five seconds with
    "Not you? Sign out", then settles.
41. Optional expected domain in settings (guess that Nail wants it): when the
    signed-in email differs, a persistent exclamation line "Signed in as
    nail@gmail.com, expected @altery.com". A hint in a panel is not a gate;
    the identity invariant holds.
42. The signed-in row shows the email and a small check. No picture: it
    needs a userinfo read, a scope and the account grant per run, and a face
    on the panel invites treating identity as a gate.
43. Token expiry is never shown as a time. "Refreshes itself at 23:40" is a
    promise the app cannot keep. The row shows one of three: OK, Needs
    sign-in, Missing scopes: <name>.
44. `gdoc auth status` is asked when the dropdown opens and every ten minutes
    while a watch runs, never more often.
45. Refresh failed mid-day: badge exclamation; a NOW card at the top, below
    holds: "Sign-in expired. Live watch on <doc> is paused", with Sign in and
    Stop watch. The app does not kill the watch; if the binary exits on 401
    today, the card says stopped and resume is typing live again (guess on
    current behaviour). The log records the failure and the later sign-in.
46. Sign out exists, in HEALTH beside the email, and calls a new
    `gdoc auth logout`, which removes the token (file or Keychain item) and
    prints one envelope. Deleting a file the app does not own is how two
    programs end up disagreeing about one path. This is a binary change and
    Nail's decision.
47. Sign out confirms inline and names the consequence: "Sign out? 2 live
    watches and 1 hold will stop", three seconds.
48. Sign out stops watches and releases holds without writing, each logged.
    A hold released by a sign-out must never become a write under the next
    account. The log and DOCUMENTS survive untouched; only the token goes.
49. Switching accounts is sign out, then sign in. No switcher, no second
    token. The log records the email at each sign-in; when it differs from
    the previous one, a one-line notice under the account row for the rest
    of the day: "Account changed from A to B". It blocks nothing.
50. `install.sh` stays primary for the first login and runs `gdoc auth login`
    as its last step, printing the URL as today. The panel's Sign in is for
    re-login, Nail's common case.

Screens to add for auth: HEALTH signed out; waiting for Google with the
countdown and the copyable URL; just signed in with "Not you?"; wrong
domain; expired mid-watch; sign out confirm; account changed; missing
scopes.
