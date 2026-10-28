# The presence: the fifth look

2026-10-28. Nail looked at the panel of v2.24.0 and sent four sentences: when
the pointer is on a document he wants to see its details, the path and the
date, and whatever else belongs there; a click on the document should open
it; the panel should be a little wider; and could the firm's mark sit in the
empty space beside the DOCUMENTS label. He asked the product owner for the
journey and the designers for the empty column and the mark.

Decided the same day, in Nail's words: "ok, run with the recommendation: 1
yes, 2 a, 3 360, 4 yes, 5 b after". It was built by
`docs/plans/completed/2026-10-28-gdoc-v2-m32c-the-fifth-look.md`.

The seats answered on the same day. This page is their answers and the
lead's reading of them. The screens are
`2026-10-28-presence-fifth-look.html`. Nothing here was built when it was
written, and the decision was Nail's: it is in `docs/v2/DECISIONS.md` in his
words.

## The journey of one pointer on a DOCUMENTS row

The product owner's seven steps:

1. Rest. The row as today: the blank column, the cut title, the cut second
   line, the icons.
2. Hover. The row fills grey at once. After the half second the tooltips
   already wait, a detail card appears under the row, inside the panel. The
   pointer is on the title or the second line, not on an icon.
3. Click on the row. The document opens in Google Docs. The card goes with
   the press. The panel stays open.
4. Click on an icon. The icon does its own thing. While the pointer is in
   the icon column the card is never shown; the icon's tooltip is.
5. Right-click. The system menu with Hide. The card goes.
6. Held press, half a second. The same menu. The press takes the card away
   first, so the menu and the card are never up together.
7. Leave. The card goes the moment the pointer leaves the row.

A release before half a second is a click and opens; half a second down is
the hold and opens the menu; the two gestures the panel already has keep
them apart.

## The detail card

The tooltip's plate and not a tinted card, because tint means a question or
a problem. It sits under the hovered row, leading-aligned with the title,
panel width less the row inset (320 pt on a 360 pt panel), 8 pt padding, the
control radius, a hairline edge and the tooltip's shadow. It takes no clicks
and carries no controls. It flips above the row when there is no room under.

Three lines, each a fact the row already draws and cuts:

1. The whole title, wrapped.
2. The last action with its full moment: "published Wed 28 Oct 2026, 12:08",
   in the second line's own verb.
3. The whole note path, home shortened to `~`, or "no note".

Until the first click on any row, a fourth quiet line says "Click opens in
Google Docs", the way the labels under the icons teach the icons once.

Left off, and why: the Drive link, because copy link gives it and the row
opens it; pinned-since, because the row does not hold it and the filled tack
says pinned; the Claude Code session, because nothing opens it today. The
rule for adding a line: the card is the row uncut, never the row plus. A new
fact on it is a decision entry, not a line.

The guardian has no veto. It notes that the card shows a full home path
without a click, which matters on a shared screen, and the half-second rest
is the answer to that. One layer draws one bubble, so an icon's tooltip wins
over the card when both could show.

## The row click, and the Google Docs icon

Nail's word of today supersedes the fourth look's "a DOCUMENTS row gains no
row click". The row is the press, down the one path a NOW row already takes
to open its document. The product owner's call, which the lead takes: the
Google Docs icon goes. An icon that does what the row under it does is a
second control for one action, and NOW rows open with no icon. The strip
becomes four labels: Note, Chat, Link, Pin. The title column gets the icon's
28 pt back.

The alternative, kept for Nail: the row opens and the icon stays as the
labelled control. It costs nothing but the duplicate.

## The blank leading column

Keep it, with nothing in it but the red mark of a failed run. It is what
makes every title in the panel start on one edge; NOW draws its ring and
its marks there. Dropping it gains 24 pt and leaves the failure mark nowhere
honest. Filling it on every row says nothing: every row is a document, a
tack would say pin twice beside the fifth icon, and a kind mark repeats the
second line. The visual designer's read: the column looks empty because the
panel is dark and narrow, and the width fixes most of that feeling.

## The width

360 pt, from 320. On the 8 pt grid 344 is barely felt and 384 starts to
look like a window. At 360 the title column goes from 112 pt to 152 with
five icons and 180 with four; the second line shows about seven more
characters. Control Center and the Wi-Fi menu are about 320; 1Password's
panel sits near 360 to 380. The one cost is that the panel slides 40 pt
further left near the right edge of the screen, which the window already
does.

## The firm's mark

Three places were drawn. Beside DOCUMENTS, in Nail's red box: a 14 pt grey
wordmark reads as "Altery's documents", which is only half true, and puts a
picture in a row that has no colour and no action. In the header beside the
account: fits the 36 pt line but competes with the account, the one thing
the header is for. At the foot: a 28 pt row under the last hairline with a
12 pt monochrome mark in tertiary ink, the way an About line sits, with no
claim about any row above it. The designers recommend the foot, drawn as a
template image in one grey, and nothing at all, no placeholder and no gap,
on a machine that has no file.

What it costs, whichever place. The app reads three things by invariant;
the mark is a fourth. The trust guardian vetoes the app reading the cache
on its own (it would need its own copy of the Drive id, and go blank when
the logo moves) and vetoes the binary handing over bytes (a seventh argv
list or base64 in every skill's output). It signs one route: `help --json`,
which the app already runs, names the cached file's path; the app opens
that one path read-only, accepts only a regular file under a megabyte that
decodes as a PNG, and draws nothing otherwise. `help` never fetches the
logo. The entry's sentence: "The app reads one more file, the path
`help --json` names, and draws nothing when it is absent or not a PNG." A Go
change, a Swift change, a CLAUDE.md line, and a decision. The lead's
recommendation is a milestone of its own after the row work, so the row
work ships this week on its own.

## Out of scope, named

Reading Drive for the card (owner, editors, counts): the app never talks to
Google. Note contents beyond the path. Buttons or links inside the card. A
hover card on NOW rows. A placeholder, a "no logo yet" line or a file
watcher for the mark.
