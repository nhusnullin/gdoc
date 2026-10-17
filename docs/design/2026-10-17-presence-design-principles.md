# The presence: design principles, the team, and the review before a show

2026-10-17. Nail's instruction after three rounds of screens: write down the
principles so every screen is consistent, keep the team roster so nobody is
forgotten, and review before showing. This file is that. It is read before
any screen is drawn and before any screen is shown.

## Principles for the panel

1. **One icon, one meaning, by priority.** The menu bar shows one monochrome
   state: exclamation (stuck), pause (a hold waits), ring (a finite job),
   the mark (a live watch), nothing (idle). A green check for five seconds
   after a verified write, over any state. No colour in the menu bar, ever.
2. **Colour lives in the 16 px status glyph and nowhere else.** Titles,
   values, times, rows, borders and labels stay neutral. Two exceptions: the
   hold card tint and the stuck card tint, both at 7 percent with a half-pixel
   ring.
3. **Actions are icons; questions are words.** A row's actions are 16 px line
   icons at its trailing edge, always visible, tertiary grey, tooltips after
   half a second. A text button appears only where the panel asks a question
   with a consequence: a hold (Open chat, Release), an inline confirmation
   (Release? Sign out?), and nothing else. Update, Sign in, Install, Stop,
   Cancel, Copy, Show more and Quit are icons or plain rows, never pills.
4. **Dead controls are not shown as live.** An icon with nothing to open is
   dimmed with a tooltip that says why. A card never carries an icon that
   would do nothing.
5. **Facts, not promises.** The panel says what happened and what state
   things are in. It never predicts ("refreshes at 23:40") and never judges
   ("important", "handled"). `verified: false` is a hollow check, not a
   failure.
6. **No dismiss on a problem.** A stuck card leaves when the next run of that
   command starts. An × invites closing a problem without opening the note.
7. **The reason line is two short sentences, under 80 characters:** the cause
   with its location first, then the state of the world. The title and the
   glyph already say the command and the word "stopped".
8. **Dark and light are two palettes, not one inverted.** Light uses a pure
   white panel with a drawn edge, opaque greys for secondary and tertiary
   text, saturated system status colours, a browner amber with its own ink
   for text on the tint, and a three-layer shadow. Hover goes darker, never
   lighter.
9. **Proportions on the 4/8 pt grid.** Panel 320 px, 8 px inset, 12 px
   radius; rows 32/44/28 px; header 36 px; two type weights only; tabular
   figures right-aligned; captions and section labels at 11 px.
10. **One surface.** Right-click opens the same dropdown. No second menu, no
    context menus except the one alternative on the AI icon.
11. **The app reads and asks; it never writes to Google and never touches the
    token.** Every auth action is a call to the binary; the panel shows what
    the binary printed. (A technology note, recorded while technology is
    deferred.)

## The team, by seat

Call the seat when its question comes up. Do not draw first and ask later.

| Seat | Owns |
|---|---|
| Visual designer (premium, restrained) | the panel system: grid, type, colour, components, cards; the action vocabulary |
| macOS platform designer | menu bar and status item metrics, NSMenu versus popover conventions, system colours, SF Symbols names |
| Iconographer | icon candidates by meaning, three per action, confusion risks, SVG at 16 px |
| Light-appearance designer | the light palette as its own set, not an inversion |
| Product owner | the journey: overflow, time horizon, ordering, concurrency, empty states, the action set, discoverability, interruption, login and logout |
| Recognition tester | guesses icon meanings without labels before an icon ships |
| Trust guardian | what the presence must never leak or do: titles on a shared screen, approve outside the card, a feed that judges |
| Skeptic | principle 4, every word costs attention: which signals survive |
| Colleague without a terminal | what the person in the document sees and never sees |

## Review before showing Nail

Run before every publish that Nail will look at. A reviewer who did not draw
the screens reads the screenshots and checks each line.

- Every action in every screen follows principle 3: icons for actions, words
  only for questions. List every text button on the page and justify each.
- No control is dead and live-looking (principle 4).
- Every reason line and every secondary line fits without wrapping past two
  lines at 320 px; no title truncates to under eight characters.
- The menu bar item is monochrome in every screen; the badge matches the
  state the caption describes.
- Both palettes are present where the screen exists in light; status colours
  are the light set on light.
- Numbers right-aligned and tabular; times in the value column; nothing
  predicts a time.
- Section labels, header, account row, Quit and Launch at login are in the
  same place on every screen.
- The caption under the screen names what Nail has not decided, and nothing
  else is presented as decided.
- The Russian explanations use no em dashes and no banned words.
- One screenshot per changed screen was taken and looked at. If the preview
  could not be rendered, the message to Nail says so.

## Amendments, 2026-10-17 evening

- Principle 2 has one more exception: after Copy link, the icon shows a
  green check for about a second with the tooltip "Copied", then returns.
  Nail asked what it meant and did not object once told.
- Principle 6 is reversed by Nail: a stuck card carries a dismiss × as the
  last icon in its row. Dismissing clears the card and the menu bar
  exclamation; the log row stays.
- NOW is hidden entirely when nothing runs (Nail: do not spend space on an
  empty section). Order stays NOW, DOCUMENTS, TODAY, HEALTH.
- The Account row is one row: the email as the value, the glyph carrying the
  sign-in state, the trailing icon sign-out or sign-in by state.
- Copy link uses the two-rectangles glyph, Nail's choice over the chain.
- Long titles: one line with a middle ellipsis, the tail keeping the last
  word group under 14 characters, the full title as the row's tooltip.

## Amendments, 2026-10-17, after the first reviewer's pass

Two reviewers who drew nothing read every rendered screen against the
checklist above. These are the lines the pass changed. The findings it fixed
in code are in the commit; these are the rules that had to move.

- **Principle 3's hold pair is "Open chat" and "Dismiss".** The principle
  still said "Release", which the specification's decision table had already
  replaced: the panel never says Release, because only the chat can say what
  it would write. Both halves of an inline confirm are borderless, so a pair
  reads as one pair.
- **Principle 2 gains the confirm's red word.** "Sign out" is drawn in the
  bad colour. It is the fourth exception, after the hold tint, the stuck tint
  and the Copy check, and it is there because a question with a consequence
  is the one place the panel asks for a decision.
- **A title keeps eight characters of head including its ellipsis.** The
  checklist line said eight characters; the measurement forgot that the three
  dots are drawn inside the head's own room, so a title could show seven. A
  tail that cannot leave eight characters and the dots is dropped, and the
  title goes in whole.
- **A card's title may take two lines.** A card is a plate rather than a row,
  and a stuck card with five live icons beside a one-line title stops naming
  the document that stopped.
- **The scroll thumb is as much of the track as the box is of the list.** The
  screens file draws 88 pt, which is that fraction for the one list it drew.
  A fixed 88 pt under any other list states a number that is not true, which
  principle 5 does not allow.
- **The pause badge holds a point of gap at every size.** At the menu bar's
  10 pt the screens file's one-point gap falls under a device pixel and the
  two bars close into a block that reads as a stop. Principle 1 asks for five
  states a person tells apart at a glance.
- **An empty TODAY is not drawn under a pinned HEALTH**, which is the rule
  the evening's amendment gave NOW. HEALTH is pinned first on three screens:
  no token, a sign-in in flight, and the three seconds a question is asked.

Left as they are, with the reason:

- **The live dot is 3.5 pt, not 6.** The screens file sets `box-sizing:
  border-box`, so `inset:5px` is measured inside the 13.5 pt padding box. The
  file's own value is what is drawn.
- **The panel's corner is 12 pt continuous.** The value is the file's; the
  curve is the platform's, which is why a ruler finds about 9.5 pt.
- **Continue with Claude is live on every row.** It activates Claude Desktop
  and nothing more, so there is no row it would do nothing on. The screens
  file dims it on a document restyled from chat, which is a session the
  specification does not build.
- **The menu bar shows the exclamation in scene e3.** The priority is stuck,
  hold, job, live, idle, and the specification's table wins over the file's
  own drawing of that scene.

## The render for review

Every scene is drawn twice: the panel as the person opens it, capped at
600 pt with the rest under a scroll, and the same panel with nothing hidden.
The second is the one to review. A PNG cannot be scrolled, so a screen whose
HEALTH sat past the cap was a screen nobody had checked.
