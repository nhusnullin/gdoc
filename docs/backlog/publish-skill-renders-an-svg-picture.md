---
worth: yes
where: skills/gdoc-publish/SKILL.md:112
added: 2026-09-19
---
# The publish skill says nothing about a note whose picture is an SVG

`gdoc build` and `gdoc publish` read only PNG and JPEG, and refuse an SVG
picture naming the line and the format (`go/internal/body/image.go:67`). The
refusal is right: Google Docs does not import SVG from a docx, Word's own SVG
extension always carries a PNG fallback, and the binary neither runs an
external program nor takes a fourth module, so rasterising in Go is a
`DECISIONS.md` entry with worse text rendering than a browser gives. So the
PNG has to come from the session. The skill does not say so, and its Step 1
names two refusals, no `gdoc:` block and no title, and not this one.

Seen 2026-09-19 publishing the Bridge sales-deck note, whose four diagrams
were SVG. The session found the route by trial: cairosvg fails on this Mac
because libcairo is not installed, QuickLook (`qlmanage -t`) renders a square
thumbnail and crops the diagram, and headless Chrome worked on the third try.
Five tool calls that the next session will spend again, because neither the
repo skill nor the plugin copy records any of it.

## What the skill should say

One section beside the title refusal, in the same voice:

- When the refusal names an SVG, render it at the SVG's own `width` and
  `height` at device scale factor 2, which lands at about 400 ppi once gdoc
  scales the picture to the text column. Not higher: it only inflates the
  docx.
- Chrome is what this Mac happened to have, not a rule. The skill names a
  detection order and the session takes the first one present: `rsvg-convert`
  (librsvg, the cheapest and the one to suggest installing), then a
  Chromium-family browser by its platform path, Chrome, Chromium or Edge, all
  of which take the same headless flags, then `cairosvg` only if it starts.
  Edge is on every Windows machine, so Windows always has a route; macOS and
  Linux may have none.
- When nothing is present, stop and say so: name the picture, name what to
  install (`brew install librsvg`, or the distribution's package), and offer
  the other door, a PNG the author exports by hand and links from the note.
  Never download a browser or a Node package to get one, because the skill
  would then be fetching code onto a colleague's machine on its own.
- Write the PNG beside the SVG with the same stem, repoint the note's link
  from `.svg` to `.png`, and keep the SVG as the master. Say in the reply
  which lines were rewritten, because the hub may not be under git and the
  transcript is then the only record.
- Name the two dead ends in one line each, so nobody tries them again.

The two commands, for the skill to carry:

```
rsvg-convert --zoom 2 diagram.svg -o out.png

"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new \
  --disable-gpu --hide-scrollbars --force-device-scale-factor=2 \
  --window-size=1400,620 --screenshot=out.png "file://$PWD/diagram.svg"
```

The browser paths by platform: the macOS app bundle above, `google-chrome`,
`chromium` or `microsoft-edge` on PATH on Linux, and
`C:\Program Files\Google\Chrome\Application\chrome.exe` or
`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe` on
Windows. The rsvg line was not run on 2026-09-19 because librsvg is not on
this Mac; check its flags when writing the skill.

Open edge, not for the skill to solve: the PNG is derived output living beside
its source, and an edited SVG leaves a stale PNG and a stale document with no
warning. One other hub note links an SVG today, so a check is not worth
building yet.
