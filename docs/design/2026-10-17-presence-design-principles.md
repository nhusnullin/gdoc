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

Every scene is drawn once in each appearance, and that one image is the
whole panel. It was two images until 2026-10-18: the panel as the person
opened it, capped at 600 pt with the rest under a scroll, and the same panel
with nothing hidden. The third look took the cap and the scroll away, so the
two are one drawing and the panel the person opens is the panel the reviewer
reads. A panel taller than the screen is cut at the screen's edge, which is
measured in `docs/v2/MEASURED.md` and is the price Nail chose over a scroll.

## Amendments, 2026-10-18, after the third look's pass

Five reviewers who drew nothing read every rendered screen and every
component sheet of milestone 27 against the checklist above: seventeen
scenes between four of them, and one on the six sheets. Three findings were
blocking, nineteen were to fix, the rest were notes. Every blocking and
should-fix one is closed in the pass's own commit except the ones at the
foot of this section. These are the lines that moved.

- **A page that says what is drawn says it after the drawing changes.** The
  index named a Hub row in SETTINGS, a pin before the state glyph, "Pin,
  Unpin and Hide" in a row's menu, a hint strip of four labels and a
  first-run screen of 409 pt. The milestone had taken the Hub row out, moved
  the pin to the row's fifth icon, left Hide alone in the menu, made the
  strip five and the screen 381 pt. Three reviewers read the same paragraph
  and two of them rated it blocking, which is what a sentence costs when it
  outlives its drawing. Five source comments said "the four labels" as well.
- **One right edge means the switch too.** A row pulls its trailing column
  4 pt out, because an action's 16 pt of artwork sits in a 24 pt box and the
  artwork is what meets the edge. A 26 pt pill has no such margin inside it,
  so the one control in the panel stood 4 pt right of every icon, every
  value and the ⌘Q under it. Four reviewers measured it at 303.5 against
  299. It predates this milestone and was invisible until Launch at login
  went off by default and put the switch on nearly every screen:
  `testTheSwitchStopsWhereTheKeystrokeUnderItStops`.
- **A label sits on the icon it names, not 4 pt off it.** The hint strip is
  the panel's whole onboarding, and it kept a row's 12 pt where the icons
  above it are pulled 4 pt out, so every word was half a letter left of its
  own icon. The third look made it plain by adding a fifth: "Pin" under the
  gap beside the tack is the hint failing at the one icon Nail could not
  find: `testEachLabelIsCentredOnTheIconItNames`.
- **An icon is one mark in one ink.** The pinned tack is a filled head with
  the outline's stroke over it, which is how both states keep one
  silhouette. Drawn as two layers of a translucent ink it composites to
  `1-(1-a)²`: 0.49 where the action ink is 0.29, and 0.61 where the dimmed
  one is 0.38. The pinned row carried the brightest icon in the panel and a
  dimmed pin outshone a live icon beside it. The shapes are drawn opaque now
  and the ink's own alpha goes on the pair:
  `testThePinnedTackIsOneLayerOfItsOwnInk`.
- **Every state this milestone added is on a screen, and two of them had no
  day to be on.** The update glyph is drawn four ways. Two belong to moments
  of a day and are on scenes, r8 and e1; the check after a press that
  installed nothing and the ring while the child is up last as long as a
  press and belong to no moment, so they were drawn nowhere at all. The rows
  sheet draws the header four times now, which is the one page a state with
  no day can be on: `testTheRowSheetDrawsTheUpdateGlyphInEveryDrawingItHas`.
- **Principle 2 gains the update glyph.** Three reviewers found the accent
  ink in the header against "colour lives in the 16 px status glyph and
  nowhere else". It is the fifth exception, with the green check after a
  press that installed nothing for the sixth, which is the Copy check's own
  rule in another place. The accent is Task 5's decision and the principle
  is what had not caught up.
- **A caption names what the render draws**, five more times: r3 named the
  source of each write where a DOCUMENTS row draws the word and the moment
  alone, r9 called its list the fortnight where the list is the last five,
  r8 and r9 said nothing about the blue dot and the published line that are
  this milestone's own new header, e5 said the grey note icon is on no
  screen where it is on three, and e3 explained how a publish clears a stuck
  card under a card that is an export.

Left as they are, with the reason:

- **The question is the one header with no update glyph, and that is the
  rule.** Task 5 says the glyph is always in the header; the second
  morning's amendment says the question takes the header's line. Both are
  written down and they meet on one screen, e10. The question wins: a line
  asking whether to sign out is not a line to offer an install on, and the
  version, the glyph and the memory of the changed address all come back
  when it is answered. e10's caption says so now.
- **The quiet glyph is at the action ink, which in light is a step darker
  than the version beside it.** Task 5's sentence is "the icon in the
  caption grey", and in dark the two inks measure the same. In light they do
  not, and the glyph is drawn in the ink every pressable icon in the panel
  uses. Making it the fainter grey would leave the one action in the header
  fainter than every action under it, which is principle 4 read backwards.
- **A bubble ends on the icon column however many icons the row has.** With
  the pin, a Copy tooltip ends 24 pt right of the icon it belongs to and
  over the tack. The rule is the second morning's and it is deliberate: a
  bubble that moves with its icon gives one row five bubble positions, and
  the icon it hangs under is what says which one it is about. It reads
  differently at five icons than it did at four, so it is named here for
  Nail rather than changed.
- **e5 carries a hairline above SETTINGS and e8 does not, because one has a
  block above it.** The rule is `isFirst`: the section under the header
  carries no rule, and a panel with a notice or an empty state above it has
  no first section. e5's first-run sentence is a block of the body; e8's
  account banner belongs to the header. One reviewer read the difference as
  a drift between two screens of the same shape.
- **"Still your call" keeps the DOCUMENTS header line until Task 9.** Four
  reviewers found it, and all four were right that no render draws the count
  any more. The plan gives that list to the milestone's last task, which is
  where the lines this plan decided leave it.
- **The fifth icon costs the title column 28 pt, and the titles say so.**
  Every reviewer who read a document row measured it: "Ch 5. Commercial
  Model… v0.1" in NOW against "Ch 5. Com… v0.1" in DOCUMENTS on one screen,
  and sub-lines that now wrap to two. Every one of them clears the
  eight-character head floor, so the rule holds and the room is what the pin
  was bought with. The lever is the strip, not the truncation, and it is
  Nail's.
- **The pin's artwork is the smallest in the strip**, 10 × 11.5 pt where its
  neighbours are 11–14 × 13–14. A tack seen from the side is a bar and a
  needle, and it is drawn inside the same 16 pt slot as the rest. Making it
  larger is the iconographer's seat and a drawing Nail approved.

And the thing the milestone is named for, which every one of the five was
asked for separately and every one of them answered the same way: **no
render shows a scroll thumb, a fade, a Show more row or any other sign of a
cap**, in either appearance, on any of the seventeen scenes or the six
sheets. Four of them checked by measurement as well as by eye, scanning the
strip inside each panel's right edge for a vertical run and finding only
card tints, the panel's own edge and ordinary text. The heights are the
other half of it: the panels come out between 321 and 1088 pt, so the cap is
gone rather than merely unreached.

## Amendments, 2026-10-26, the fourth look

Nail read the panel of v2.20.0 and sent the sentences nine backlog items
held. Three of them change a rule written above.

- **The mark before a document means one thing, a run that failed.**
  Principle 1 is about the menu bar, and the document row's own 16 px glyph
  grew to five marks under it: a green check for a run that worked, a dot
  for a live watch, a dim dot for one that stopped, a pause for a hold, the
  red exclamation for a failure. Nail read the five on the panel and could
  name none of them, and the row's second line was already saying the same
  thing in words. So the exclamation stays and every other state draws the
  blank glyph, which keeps the column and the one left edge every title
  starts on. The words follow the mark: a failed run said "stopped Tue",
  which is what a watch somebody stopped says, and it names the write now,
  "reply failed Tue" against "live stopped 12:26":
  `StateTests.testADocumentRowIsMarkedOnlyWhenItsLastRunFailed` and
  `testAFailedRunDoesNotReadAsAStoppedOne`.
- **The eight-character head floor counts the name and not the verb in
  front of it.** This reverses the M25 amendment above, which kept the floor
  on the verb and the name together and sent the row to the backlog. A title
  is handed over as two pieces now, so `MiddleTitle` draws the verb and its
  dot whole and measures the floor and the middle cut on the name alone, in
  the room the verb leaves. `Live stopped by you · Ch 5.… v0.1` passed the
  floor at 27 characters while the name inside it was cut to five, which is
  four characters and a full stop to tell a document by. A title with no
  verb in front of it measures exactly as it did:
  `RowTests.testAPrefixTakesItsRoomAndTheFloorCountsTheName`.
- **A held press opens the system's own menu, never a second window.**
  Principle 10 says right-click opens the same dropdown; it did not say what
  draws it. A SwiftUI popover is a window of its own, and the panel is a
  borderless key panel whose delegate closes it the moment it loses key, so
  the one menu the panel has was drawn by the one thing that could close the
  panel under it. The 2026-10-18 probe never got the app active and never
  reached the question. Both ways in pop up a real `NSMenu` from an anchor
  behind the row's title, over the same `MenuItemModel` list, and no view
  under `Sources` opens a window at all:
  `SourceRulesTests.testNoViewOpensASecondWindow` and
  `CardTests.testTheHeldPressOpensTheSystemMenuWithTheRowsOwnWords`.

And the thing that is not a drawing: **every row in NOW is a press that
opens its document**, which is principle 3 read the other way round. An
action is an icon, and a row that is about one document is a link to it. A
hold, a write that stopped, a job and a watch each carry the id they open,
held to the shape Drive writes, and the controls on them keep their own
press: Stop stops and opens nothing, and a hold asking its question opens
nothing while it asks. The gesture itself is Nail's hand check, since no
test sends a real click: `StateTests.testEveryNowRowThatNamesADocumentOpensIt`
and `CardTests.testTheRowOpensAndOnlyStopStops`.

## Amendments, 2026-10-28, the fifth look

Nail looked at the panel of v2.24.0 and sent four sentences: details when
the pointer is on a document, a click that opens it, a wider panel, and the
firm's mark in the empty space beside the DOCUMENTS label. He answered the
lead's five questions the same evening, "ok, run with the recommendation: 1
yes, 2 a, 3 360, 4 yes, 5 b after". Three of them change a rule written
above, and the fourth is a rule about a drawing nobody has drawn yet.

- **A row that is about one document is the press that opens it, and one
  action has one control.** Principle 3 says an action is an icon, and the
  fourth look had already read it the other way round for NOW. DOCUMENTS
  followed it here: the title, the second line and the gaps around them are
  the row's own press, the Google Docs icon went with it, and the strip is
  four icons and 108 pt instead of five and 136. Two controls for the same
  opening is one too many, and the one that cost nothing to lose is the one
  the row can do by itself. A row whose id is not one Drive could have
  written carries no press, which is principle 4 for a whole row:
  `StateTests.testADocumentRowOpensItselfAndCarriesFourIcons`,
  `CardTests.testADocumentRowOpensAndItsIconsKeepTheirOwnPress` and
  `RowTests.testTheDocRowActionColumnIs108`.
- **The card is the row uncut, and never the row plus.** Principle 5 says
  facts and not promises; the card says which facts. Every line on it is a
  line the row already draws, with the cutting taken off: the whole title,
  the same state words with the moment written out, the same note path. The
  one line that is not about the document is the hint, and it goes after the
  person's own first click on a row and never comes back. A new fact on the
  card, a word the row does not already say, is a decision in
  `docs/v2/DECISIONS.md` and not a line somebody adds to a model:
  `StateTests.testTheCardIsTheRowUncutAndNeverTheRowPlus` and
  `WordsTests.testTheFullMomentCarriesTheDayDateAndClock`. It is a rest and
  not an entry, on the tooltip's own plate and placed by the one layer that
  places tooltips, so an icon's words win over it and a held press or a
  click takes it away before the menu or the tab arrives:
  `ControlTests.testTheCardShowsAfterTheRestAndGoesOnLeaveHeldPressOrClick`,
  `testTheCardSpansFromTheTitleToTheValueColumn` and
  `testAnIconsTooltipWinsOverTheCard`.
- **The panel is 360 points wide.** Principle 9 said 320, which is the
  number the first screens were drawn at. 360 is Nail's own answer of the
  three offered, and the grid is the one place it is written: the window
  reads `Grid.panelWidth` and holds no width of its own, so a width is
  changed in one line or not at all. Nothing was loosened to reach it. The
  premises that were true only at 320, the head floor, the counted rule and
  the header's crowding, were each re-measured and restated at 360:
  `RowTests.testThePanelIs360PointsWide` and
  `SourceRulesTests.testThePanelWindowTakesItsWidthFromTheGrid`.
- **The firm's mark is decided for the foot of the panel, and not drawn
  until its own milestone.** Nail's answer was "5 b after": the mark goes
  below Settings and not beside the DOCUMENTS label, where it would sit in
  the one column the fourth look keeps blank so every title starts on the
  same left edge. An empty column is not a space for a picture; it is what
  makes a list readable. Nothing in this branch draws or reads a picture,
  and the day the mark is drawn is a milestone of its own with a decision
  entry in front of it.

## Amendments, 2026-10-29, the stuck card's open icon

The design reviewer read the renders of v2.25.0 and named two should-fix
items. Nail answered the first: "Yes, drop the open icon from the stuck-run
card in the next release; the row already opens the document." The second is
the lead's how and cost one line. The decision is in `docs/v2/DECISIONS.md`,
2026-10-29, "The stuck card loses its open icon".

- **The stuck card's open icon went, because one action has one control.**
  The fifth look took the Google Docs icon off a DOCUMENTS row when the row
  itself became the press; the stuck card in NOW kept the same icon beside
  the same press, which is two controls for one opening. The card now
  carries the note, the chat, the link and the cross, and a click on its
  words opens the document. With four icons the title column beside them is
  184 pt instead of 156. The glyph is gone from the sheet, which draws
  twelve icons now, and the words that said "No Google Doc yet" went with
  it: `CardTests.testTheStuckCardCarriesNoOpenIcon`,
  `CardTests.testACardCutsATitleByTheRowsOwnRule` and
  `IconTests.testTheIconSheetRenders`.
- **A still of a forced card draws its row hovered, because a still states
  what the app shows.** Principle 5 is facts and not promises. A card is
  only ever up on a hovered row, so a scene that forces the card and leaves
  the row unshaded draws a panel the app never draws, and a reader cannot
  tell which row the plate belongs to. `RowView.isFilled` now reads
  `hovering || pointerInside || forcedCard`, and the e1 caption says the
  plate sits under the second row and over the two below it:
  `RowTests.testAForcedCardRowDrawsTheHoverFill`.

The "five live icons" line under the first reviewer's pass stays as the
record of its day.

## Amendments, 2026-10-29, the firm's mark beside DOCUMENTS

The visual designer gave the mark's place and its numbers, the lead drew it
on the v2.25.0 render, and Nail answered: "Да, логотип стоит правильно.
Запиши это как решение вместо «внизу панели» и готовь веху." The decision is
in `docs/v2/DECISIONS.md`, 2026-10-29, "The firm's mark beside DOCUMENTS".

- **The firm's mark sits at the right end of the DOCUMENTS label, 14 pt
  tall.** Its right edge lands on the content edge every value in the panel
  ends on, the 28 pt label centres it, so its top is at 7 pt, and its width
  is the picture's own aspect and no number in the package. Only DOCUMENTS
  is handed it. The fifth look's bullet two sections up, "The firm's mark is
  decided for the foot of the panel", is superseded in its place by Nail's
  word of 2026-10-29 and stays as the record of its day: the empty column
  the fourth look keeps blank is still blank, because the label's line is
  not a row and no title starts on it. The tests are
  `RowTests.testTheDocumentsLabelDrawsTheMarkFourteenPointsTallOnTheContentEdge`,
  `RowTests.testTheMarkTakesTheLabelsOwnInk` and
  `RowTests.testNoMarkDrawsNothingAndMovesNothing`.
- **The mark is one read of one file, and nothing where there is no file.**
  Principle 5 is facts and not promises, and principle 4 of
  `PRINCIPLES.md` is that nothing is built for a machine that has no
  picture: no placeholder, no fetch, no watcher. `gdoc help --json` names
  where `internal/asset` cached the house logo, the app opens that one path
  with `O_NOFOLLOW` and takes it only as a regular PNG of at most a
  megabyte, and a missing file, a link, a swap or bytes that are not a
  picture all draw nothing and write no line. The app holds no Drive id and
  asks no host. The tests are `MarkFileTests` and
  `SourceRulesTests.testAByteIsReadOnlyInTheFiveReaders`.

## Amendments, 2026-10-30, the mark sharp and a press

Nail looked at v2.29.0 on his own screen within the hour: "logo is good in
general but im doubt about its resolution. also, when i click i expect
default browser with picture from gdrive to be opened". The lead measured
the first half and wrote the second half up. The decision is in
`docs/v2/DECISIONS.md`, 2026-10-30, "The mark, sharp and a press".

- **The mark is drawn from a bitmap made once at its drawn pixel size, so
  its edges are anti-aliased like the words beside it, and the view scales
  nothing.** A 509 by 185 picture handed to the compositor at 14 pt is
  scaled down by six and a half at draw time, which leaves the letters
  ragged beside anti-aliased words. The bitmap is made at the drawn size
  times the screen's scale, once per bytes, height and scale, and the
  label draws that bitmap and no other picture. The tests are
  `RowTests.testTheMarkIsABitmapAtItsOwnPixelSize` and
  `RowTests.testTheBitmapAveragesWhatItShrinks`, which holds that
  one-pixel stripes shrink to a flat mid grey; whether it looks sharp on
  a screen is Nail's own check, because the renderer averages already and
  a test can only say that it did.
- **The mark is a press that opens the picture's page in Drive.** The
  tooltip after half a second says "Open in Google Drive", a hover plate
  goes darker behind it, and there is no press where there is no file or
  no address, because principle 4 is that dead controls are not shown as
  live. The sentence "The app holds no Drive id and asks no host" in the
  section above stays true: the page's address comes from the binary on
  the same help answer as the path, and the app takes it only in that one
  shape and opens no other. The tests are
  `RowTests.testTheMarkWithAPageIsAPressThatOpensIt`,
  `OpenerTests.testOpenMarkOpensTheDrivePageAndNoOtherShape` and
  `SourceRulesTests.testTheOnlySchemesAndHostsAreGooglesAndTheVaults`.

The 2026-10-29 decision's "It is not a press and has no tooltip" clause is
superseded in its place and stays as the record of its day.
