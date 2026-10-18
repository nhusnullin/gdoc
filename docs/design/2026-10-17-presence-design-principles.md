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
   with a consequence: the hold card's Open chat and Dismiss, and the two
   inline confirms the panel has, "Dismiss this card?" and "Sign out?", each
   answered by one loud word and Keep. Nothing else. Update, Sign in,
   Install, Stop, Cancel, Copy, Show more and Quit are icons or plain rows,
   never pills.
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
  no token, a sign-in in flight, and while a question is asked.

Left as they are, with the reason:

- **NOW and TODAY name the stopped watch the same way.** On scene e9 both
  read "Live stopped · Chapter 5", one over "15:29 · sign-in expired" and
  one over "19 min · nothing written". It reads as one thing said twice,
  which is what it is: NOW says what is waiting on the person and the log
  says what the watch did. One vocabulary for one watch is the rule that
  put the word in both.

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

## Amendments, 2026-10-17, after the second reviewer's pass

A second reviewer who drew nothing read the renders of commit 47034ad
against the checklist above and wrote thirty-three findings: three blocking,
fifteen to fix, fifteen notes. The report itself is not in the tree, so the
findings are named here rather than lettered. Four tasks closed them. These
are the lines the pass changed.

The three blocking ones, and where they were closed:

- **The fixtures told one story each, not one story.** The stopped publish
  was named on the first screen and nameless on the five after it, one run
  changed its source in the middle of the day, and one approved write drew
  two rows. A reviewer reading the day as a day read every one of them as a
  bug in the panel. Closed in Task 1, in the records under
  `Tests/PresenceUITests/Scenes`.

  Half of the first one was closed the wrong way and corrected afterwards.
  The fixtures named the stopped publish by putting a title on its `start`
  line, and `presence.Begin` writes no title on a start: a publish that
  stopped inside the render has no name on any line it wrote, so the card
  and the row say "Publish" alone and the renders now draw that. Naming it
  would be gdoc writing the note's own title into the record, which is a
  decision and not a fixture:
  `docs/backlog/a-publish-that-stopped-in-the-render-has-no-name.md`.
- **A watch whose sign-in expired is stopped, not paused.** "Paused" is the
  panel saying it will carry on, which is a forecast, and nobody has
  measured whether gdoc resumes from the same cursor after a sign-in. Scene
  e9 says "stopped 15:29 · sign-in expired", draws no Stop icon, and its
  caption says the resume is unmeasured. Closed in Task 2.
- **The index page names what is still Nail's call.** A reviewer reads a
  render as the specification drawn and rates a product choice as a bug. The
  page now opens with six lines, each a choice the render makes against a
  rule or a decision: the idle bar's dim mark, Launch at login on by
  default, a publish kept out of DOCUMENTS until it has an id, a trashed
  document's row unstruck, the sign-in countdown, and the DOCUMENTS header.
  Closed in Task 4.

The rules that moved with them:

- **Principle 3 names the confirms that exist.** "Release?" was never a
  string in this build and "Dismiss this hold?" never was either: the card
  asks "Dismiss this card?", because dismissing clears the card and leaves
  the hold in the chat. The other confirm is "Sign out?". Both are answered
  by one loud word and Keep.
- **The DOCUMENTS header has two forms and not three.** "N today" where
  every row listed is from today, "14 days · N" otherwise, and N counts the
  rows behind Show more. The old rule wrote "1 today" over four rows from
  three days.
- **A held write is named by its heading.** "Proposal held by rule Unread
  target", never the record's own `propose`. The panel has one word per
  write and a raw command name is a second vocabulary for one thing.
- **A sub-line wraps before it truncates**, which is the rule the document
  rows already followed, and a card's title drops its tail rather than its
  head. A line nobody can read is worse than a row 12 pt taller.
- **HEALTH's row rule is said once, on the index page.** It is the one
  section whose length changes screen to screen, so a reviewer comparing two
  screens reads a missing row as a bug: Account first, Quit last, Network
  and Claude Desktop only where a fact carries them, and no Skills row at
  all.
- **A sheet draws no word of its own.** The two component sheets are the
  pages with no record behind them, so every string on them now comes out of
  a scene through `SheetPieces`, and a test holds it:
  `testEverySheetWordIsAWordAPanelDraws`. The first draft of them had
  invented "Watching", "In the trash", "Proposed 2 · verified" and
  "Nothing since 09:05.", none of which gdoc can write.

Left to Nail, on the index page rather than in a task: the six lines of
"Still your call". A reviewer who rates one of them blocking is rating a
decision, not a drawing.

## Amendments, 2026-10-17, after the third reviewer's pass

Five reviewers who drew nothing read every rendered screen and every
component sheet of this branch against the checklist above: four scenes
each, and one on the sheets. Three findings were blocking, fifteen were to
fix, the rest were notes. Every blocking and should-fix one is closed in the
pass's own commit except two, which are at the foot of this section with the
reason. These are the lines that moved.

- **A model that says there is nothing to stop is a row that draws no Stop.**
  The panel's one dead-looking live control was a view filling an absent
  state in with `.live`: scene e9 drew the brightest Stop icon in the panel
  for a watch no process is polling. A row is handed the state or nothing,
  and nothing draws nothing.
- **A watch Google refused the token on is over in every section.** It is
  the end of the watch and not a watch in trouble, because the skill's next
  wait cannot start without a sign-in, so DOCUMENTS says "live stopped" and
  TODAY carries the stopped row. NOW keeps it while the token is still
  broken, because signing in does not start it again, and the panel's own
  NOW is where what waits on the person lives. Once the token works the row
  leaves: by then the log is the whole of it.
- **A card drops a dead icon; a row dims one.** Principle 4's second
  sentence is the rule and the render was the exception: the stuck card of a
  publish that created nothing carried a dimmed note icon that can never
  resolve a note, because no field of the record names the file a publish
  was built from. A row in a list keeps the column the same on every row and
  says why an icon is off. A card is one plate about one write.
- **One word per write, everywhere.** A watch's counts are "replied 1" and
  "proposed 2", which is what TODAY and DOCUMENTS call those writes and what
  the specification's own sentence says. "answered" was a third word for the
  one thing.
- **A row written in the present tense says so only while it is true.** The
  row that raised a hold said "waiting in Desktop chat" under the row that
  approved it twenty-one minutes later. It now says what it is: the moment
  the write was held, and by which rule.
- **One right edge for every value in the panel.** A row with nothing at its
  trailing edge draws no column there. An empty box is not nothing: its
  8 pt gap put every version, mark and keystroke in HEALTH 8 pt left of
  every time in TODAY.
- **A one-line row meets its columns in the middle.** The two-line shapes
  hang from the top, so a glyph sits against the title. A one-line row has
  nothing to hang from, and a 24 pt control beside a 16 pt line of words put
  the chevron of every Show more 6 pt under the words it opens.
- **A card's title is cut by the row's own rule.** The second line was left
  to the system's middle truncation, which knows where the middle is and not
  where a word ends: scene e3 drew "Proposal: Expansio…her regions, draft
  v1", and "her regions" is a word in nobody's title.
- **A confirm draws no line it has no words for.** The question is one line
  and its consequence is a line of the card's, so the row is the one-line
  shape. The two-line shape reserved a sub-line it never drew and left 21 pt
  of nothing, which reads as a line that failed to render.
- **A row inside a tint is written in that plate's ink.** Principle 8 gives
  the light hold tint its own ink, and the card's question and consequence
  were in the panel's neutral greys: two ink systems on one colour, one card
  apart.
- **There are three sheets, not two, and a sheet draws no section.** The
  controls sheet had four strings typed into it by hand, outside the rule
  and outside its test; it now takes its question from scene e10 and its two
  tooltips from scene e6's rows, and the test reads all three files. A
  section label over borrowed rows is a claim about them: the rows sheet
  said "14 days · 12" over two rows and a Show more, and put a log row and a
  HEALTH row under NOW. The labels are drawn as the pieces they are.
- **A caption names what the render draws.** Scene r3 was drawn one second
  after Copy link was pressed, which is the moment the check goes back, under
  a caption that said it was green: it is half a second now. Scene r2's
  caption said one sentence where the principle asks for two. Scene e2's said
  nothing older than yesterday is in the panel, where DOCUMENTS keeps its own
  fourteen days.
- **One day, one set of facts.** The check found a release at 17:05, so the
  gdoc row says so on every later still of that day. A fact that comes and
  goes between two screens reads as the panel losing it.
- **The index page carries the six component sheets and says the e screens
  are each their own record.** The sheets were written beside the scenes and
  named on no page, so the one reviewer who read them found them by
  guessing; and two counts from two e scenes are two days, not a sum.
- **A watch that is over says so in its title too.** The row read "Live ·
  Chapter 5" over a sub-line that said the watch had stopped. The title
  carries the word, as a stuck card's does, and the sub-line carries the
  moment and the cause: "Live stopped · Chapter 5" over "15:29 · sign-in
  expired", with the dot dim.
- **A sheet of a tooltip shows where the panel puts one.** The page drew one
  bubble over two icons, which is both halves of the thing going wrong: the
  layer draws one bubble at a time, and a layer with no room under the
  anchor flips the bubble on top of it. So the dead control the words were
  explaining was the one thing hidden. One icon per band, each with its own
  layer and the room for it, and the icon at the trailing edge where a row
  keeps its actions: the bubble hangs off its icon's own right edge, so an
  icon drawn on the left of a sheet gets a bubble 173 pt away from it.

Left as they are, with the reason:

- **The bar of scene e8 is the ring and not the exclamation.** The scene's
  facts say the token works and a sign-in is in flight, which is a job; the
  specification's "not signed in counts as stuck" is about `auth`, and that
  is scene e5, which draws the exclamation. The reviewer read the Account
  row's "Waiting for Google" as a machine with no token, which is the other
  half of the two moments that render stacks.
- **On white, the confirm's two halves keep the panel's surface behind
  them.** The screens file gives every text button the surface colour on
  light, and the code's reason is the amendment's own: on the hold card's
  tint a transparent button would read as a hole. On the tint the two
  plates do read as two buttons, which is what "borderless" was for, so the
  visual designer's seat has one call to make here. Neither reading is a
  drawing error.
- **The stuck card's reason stays in the panel's tertiary grey on its
  tint.** Principle 8 names one tinted ink, the amber's, because the amber
  is the tint a neutral grey disappears into. The red tint is 7 percent and
  the grey reads on it, and giving every tint its own ink is a palette
  decision rather than a fix.
- **The hold card names what Dismiss leaves behind when it is pressed, not
  before.** The offer says where the text is ("The text is in the chat."),
  and the question that follows the press says the rest: "The hold stays in
  the chat until it runs out. Nothing is written." A consequence on the offer
  as well is the same sentence twice on one plate.

## Amendments, 2026-10-18, after the M25 pass

Two reviewers who drew nothing read the renders of the branch that turned
Launch at login off by default. The switch is now drawn off on every screen
but one, where every earlier render drew it on, so the off state is new
surface even though no line of its code changed. These are the rulings; the
findings themselves are in the commit.

- **The off switch in light is the mockup's own fill, so its weight is
  Nail's call and not a defect.** `.screen.light .switch` is
  `rgba(0,0,0,.12)`, which measures 1.3 to 1 against the white panel, and
  the knob's drop shadow is the only edge it has. That is the faintest
  control the panel draws, and from this milestone it is on nearly every
  screen. It is on the index page's "Still your call" list with its
  measurement rather than changed here, because changing it is changing a
  drawing Nail approved.
- **A HEALTH sub-line sits 8 pt right of its own title, and that is
  drawn.** `.quiet` is `padding-left:44px` in the screens file where a
  row's title starts at 36, so the Account row's countdown and its URL
  field both hang in from the title. Every sub-line in DOCUMENTS and TODAY
  is flush, because those are a different element. Two readings of one
  page, not a drift.
- **The eight-character head floor counts the head, and a TODAY line's head
  is its verb and its document name together.** So
  `Live stopped by you · Ch 5.… v0.1` passes the floor at 27 characters
  while the name inside it is cut to five. The rule is kept as it is and
  the row goes to
  `docs/backlog/a-verb-prefix-eats-the-document-name-in-a-row.md`, because
  fixing it means a TODAY row hands over two pieces instead of one string.
- **A sign-in card is not a stuck card and carries no dismiss ×.** The
  evening amendment that reversed principle 6 is about a write that
  stopped: the next run of that command clears it. Nothing the person does
  clears `Needs sign-in` except signing in, so the card carries the sign-in
  arrow alone, which is principle 4. The code has always done this; the
  amendment did not say it.

## Amendments, 2026-10-18, after Nail's first sign-out

- **A question waits for its answer.** The inline confirm reverted itself
  after three seconds, by principle 3's "a press nobody followed up on undoes
  itself". Nail pressed Sign out, read the sentence under "Sign out?", pressed
  again and found the icon back under the pointer: the question had gone and
  the press asked again, so `auth logout` never ran. The question now goes
  with Keep, with the confirm, or with the panel closing, which is the Keep
  nobody pressed, and never with the clock. The safe half still wins by
  default: a panel closed on a question has answered Keep.

## Amendments, 2026-10-18, after the second morning's pass

Six reviewers who drew nothing read every rendered screen and every
component sheet of milestone 26 against the checklist above: eighteen scenes
between five of them, and one on the six sheets. Two findings were blocking,
sixteen were to fix, the rest were notes. Every blocking and should-fix one
is closed in the pass's own commit except the two at the foot of this
section. These are the lines that moved.

- **The panel's header takes its own height, as SETTINGS does.** A row states
  a minimum and not a height, so the header stretched into whatever a capped
  panel had spare: on the sign-out question it took 190 pt, drew "Sign out?"
  in the middle of a hole and left DOCUMENTS and TODAY 18 pt between them,
  with the scroll thumb hanging over SETTINGS. The middle is the one part
  that gives way, which is what the second morning pinned the edges for.
- **A reserved column is only kept where something is in it.** Every document
  row kept the pin's slot, pinned or not, so DOCUMENTS stood 10 pt right of
  NOW, of TODAY, of SETTINGS and of its own section label on every screen,
  with nothing drawn in the room it took. The slot belongs to the list: a
  list with a pin keeps it on every row, so one pin does not move the titles
  under it, and a list with no pin keeps none. This is the leading-edge half
  of the third pass's "an empty box is not nothing".
- **The icons and not their boxes meet the panel's right edge.** An action's
  artwork is 16 pt centred in a 24 pt box, so every time in TODAY and every
  value in SETTINGS ended 4 pt right of every icon under it. The mockup
  answers it in one rule, `.acts{margin-right:-4px}`, which the Swift had
  only on the stuck card.
- **An action's tooltip ends where the icons end.** The bubble ended on the
  panel's own 8 pt padding, which is 12 pt right of the icon it belongs to
  and over the scroll thumb. A bubble about a row may cover the row under
  it, which its plate and shadow are for, and may not cover the one thing
  that says there is more of the list.
- **A question keeps the glyph of the thing it is about.** "Sign out?" drew
  the empty ring, which is the panel's "nobody has asked" and the glyph
  beside "Claude Desktop, not installed", so the panel stopped saying whose
  account it is at the one moment a person is deciding about it. The hold
  card's question keeps the hold's pause; the header's keeps the account's
  check.
- **A question is the only thing said under itself.** The quiet line about a
  changed address was drawn under "Sign out?" in the same ink and at the
  same indent as the consequence, so it read as a second consequence: press
  this and the account changes. It is the app's memory of something that
  already happened, and it comes back when the question is answered.
- **The pin is not a circle, and it is no action.** Its head was a ring on a
  stem in a column full of rings, where it read as a magnifying glass beside
  a state; it is a tack seen from the side now, a bar and a needle, which is
  the one silhouette in that column that is not round. And it was drawn in
  the icons sheet's action rows, live and dimmed, where it said the panel had
  a pin to press and a dead one to explain: Pin is a word in the row's menu,
  and the glyph is a mark in the leading column, so the sheet draws it there.
- **A fixture of one day carries the whole day.** Scene r8 showed a poll
  Google did not answer at 17:04; r9, the same Saturday at 18:59, had the
  same poll ending well. The record now carries the network stop and the
  wait that started again six minutes later, inside the three-minute gap, so
  the watch is one stretch and the day is one day.
- **A sheet draws a piece in the shape the panel draws it in.** The controls
  sheet handed the consequence in as the confirm row's own sub-line, which
  made a two-line row of a one-line question: the pair dropped 17 pt below
  the words it answers and the sentence wrapped where the panel runs it full
  width under the row.
- **Every state this milestone added is on a screen.** No render drew a pin,
  and none drew the Hub row's "Choose…": the new surface of two tasks had
  been looked at by nobody. Scene e1 pins its oldest document and e5, the
  first run, has no hub. A hide is still on no screen, because a hidden
  document is a row that is not there, and a still cannot show an absence.
- **A caption names what the render draws**, four more times: r3 said two
  writes over three, the index said three settings over two and Quit, the
  cards and rows sheets named four pieces each and drew the milestone's own
  header, hint strip, URL field, account block and SETTINGS rows besides, and
  r3l's caption did not say it is r3 drawn again for its appearance. The
  index now also says why a capped still carries a thumb nobody has scrolled.
- **Principle 7 asks for the facts the record carries.** A reason line is the
  cause with its location first and the state of the world second, which is
  two sentences where the record holds two: an export that stopped with no
  stop code says where the cause is and stops, and a second sentence would be
  the panel writing one.
- **The screens file asks no question the milestone answered.** Scenes r9 and
  e1 still asked Nail what the menu bar should show in the quiet, which Task
  1 settled, and r8's story still sent him to HEALTH for the reason.

Left as they are, with the reason:

- **Where SETTINGS sits on the first-run screen is Nail's call, not this
  pass's.** The plan of the second morning says both things: Task 3 keeps
  the pinned-first rule for the two screens the group is the whole of the
  panel on, and Task 4 makes SETTINGS the foot of every panel. The code does
  what Task 3 says, so e5 and e8 are the two screens where Quit is not the
  last row. It is on the index page's "Still your call" list with its
  measurement rather than changed here.
- **A watch between two polls says live and dims its Stop.** Scene r8's last
  poll failed at 17:04 and the panel is drawn at 17:05, inside the
  three-minute gap, so the watch is a watch and the Stop has no process to
  signal and says so under the pointer. The third pass's "a watch that is
  over says so in its title" is about the token being refused, which no
  next poll can get past.
- **The confirm's words stop 3.5 pt short of the icon column.** A compact
  button carries the mockup's own 8 pt of padding and an action's box 4 pt,
  so the two cannot both put ink on the same column; the plates are where
  the icons were.
- **A quiet line sits 8 pt right of the banner's sub-line under the same
  header.** `.quiet` is the screens file's 44 pt and the banner's text column
  is a row's 36 pt. Both are the mockup's, now drawn as neighbours; making
  them one is a drawing Nail approved.
- **The dimmed icon stays at its own ink.** It measures 1.37 to 1 in dark,
  which is faint for a control that has to be hovered before it says why a
  hub is missing, and `.act.off{opacity:.38}` is the mockup's number.
  Lifting it is a palette decision.
- **The two-line row centres its actions and hangs its glyph.** The third
  pass moved the glyph to the top so it sits against the title; the strip
  stayed centred, as the mockup draws it, because four icons hung from the
  title line read as a second title.

## The render for review

Every scene is drawn twice: the panel as the person opens it, capped at
600 pt with the rest under a scroll, and the same panel with nothing hidden.
The second is the one to review. A PNG cannot be scrolled, so a screen whose
HEALTH sat past the cap was a screen nobody had checked.
