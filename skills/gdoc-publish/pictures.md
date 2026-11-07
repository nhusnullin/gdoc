# Rendering an SVG for a publish

This is the SVG recipe `gdoc-publish` points at. Read it when a build or a
publish refuses a picture, and not before: it is here rather than in the
SKILL.md because most notes hold no SVG and the SKILL.md loads on every run.

`build` and `publish` read PNG and JPEG only. An SVG is refused, and the refusal
names the line and the format. Google Docs does not import SVG out of a docx, so
the PNG has to come from this session.

The refusal happens on this machine before anything leaves it: no document was
created and the note is untouched. So rendering the picture and running again is
not a second publish.

What to do, in order:

1. Read the refusal. It names the file and the line. Read the SVG's root element
   for `width` and `height`; when only `viewBox` is there, take its third and
   fourth numbers.
2. Render it at scale 2, which lands at about 400 ppi once gdoc scales the picture
   to the text column. Not higher: it only makes the docx bigger.
3. Render to the session's scratchpad first and look at the PNG before anything
   goes near the hub. Arrowheads present, text at the right weight. Both dead
   ends below were found by looking, not by an exit code.
4. The PNG goes beside the SVG with the same stem. A PNG already there under that
   name is a stop, not an overwrite: this session cannot tell stale output from
   somebody's own picture.
5. Repoint the one link on the named line from `.svg` to `.png`. The SVG stays.
   It is the master.
6. Run `build` to the scratchpad. A clean object means publish, once.

### What renders it

On macOS nothing has to be installed. `svg2png.js` sits beside this file, in the
base directory Claude Code printed when it loaded the skill, and it drives
WebKit, which ships with the OS:

```bash
osascript -l JavaScript svg2png.js in.svg out.png <width> <height> 2
```

It prints `ok <pixels>` or the reason it could not. Measured on 2026-09-19 over
four diagrams: the same pixels a headless Chrome run gave, a fifth of a second
each, and `build` embedded the result.

When it says the legacy WebView is unavailable, which a later macOS may do, fall
back to a Chromium-family browser by its app path:

```bash
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new \
  --disable-gpu --hide-scrollbars --force-device-scale-factor=2 \
  --window-size=<width>,<height> --screenshot=out.png "file://$PWD/in.svg"
```

`rsvg-convert --zoom 2 in.svg -o out.png` is cheaper still where librsvg is
installed, and `brew install librsvg` is what installs it.

When none of them is present, stop and say so: name the picture, name what to
install, and offer the other door, a PNG you export by hand and link from the
note. Never download a browser or a package to get one. That would be this
session fetching code onto your machine on its own.

Four dead ends, one line each, so nobody spends the calls again: QuickLook
(`qlmanage -t`) crops to a square, `cairosvg` needs libcairo, `NSImage` drops
`marker-end` so every arrowhead vanishes and semibold text comes out regular, and
`safaridriver` needs a one-time `sudo` and opens a visible window.
