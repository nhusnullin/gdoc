---
worth: yes
where: skills/gdoc-publish/SKILL.md:83
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
- Chrome is what this Mac happened to have, not a rule. On macOS the
  session needs nothing installed: Safari's engine, WebKit, is reachable
  from `osascript -l JavaScript`, which ships with the OS, and the script
  below renders markers, weights and fonts exactly as Chrome does. Measured
  2026-09-19 on macOS 26.3.1 over all four Bridge diagrams: same pixels as
  the Chrome run, a fifth of a second each, and `gdoc build` embeds the
  result. Two dead ends on the same road: `NSImage` also loads an SVG with
  no install, but CoreSVG drops `marker-end`, so every arrowhead vanished
  and semibold text came out regular; and `safaridriver` needs a one-time
  `sudo safaridriver --enable` and opens a visible Safari window.
- Elsewhere the skill names a detection order and the session takes the
  first one present: `rsvg-convert` (librsvg, the cheapest and the one to
  suggest installing), then a Chromium-family browser by its platform
  path, Chrome, Chromium or Edge, all of which take the same headless
  flags, then `cairosvg` only if it starts. Edge is on every Windows
  machine, so Windows always has a route; Linux may have none.
- When nothing is present, stop and say so: name the picture, name what to
  install (`brew install librsvg`, or the distribution's package), and offer
  the other door, a PNG the author exports by hand and links from the note.
  Never download a browser or a Node package to get one, because the skill
  would then be fetching code onto a colleague's machine on its own.
- Write the PNG beside the SVG with the same stem, repoint the note's link
  from `.svg` to `.png`, and keep the SVG as the master. Say in the reply
  which lines were rewritten, because the hub may not be under git and the
  transcript is then the only record.
- Name the dead ends in one line each, so nobody tries them again:
  QuickLook crops to a square, cairosvg needs libcairo, NSImage drops
  markers, safaridriver needs sudo and a window.

The macOS script, for the skill to carry as a file beside `SKILL.md`. It
uses the legacy in-process `WebView`, deprecated since 10.14 and still
present in 26.3; `WKWebView` renders out of process and its snapshot call
takes a block the bridge cannot pass. If a later macOS drops `WebView`, the
script says so on its first line and the order falls through to the next
renderer.

```
// osascript -l JavaScript svg2png.js in.svg out.png width height [scale]
ObjC.import('Cocoa'); ObjC.import('WebKit');
function run(argv) {
  const inPath=argv[0], outPath=argv[1], w=parseInt(argv[2]), h=parseInt(argv[3]), scale=parseFloat(argv[4]||'2');
  $.NSApplication.sharedApplication;
  const rect=$.NSMakeRect(0,0,w,h);
  const win=$.NSWindow.alloc.initWithContentRectStyleMaskBackingDefer(rect, 0, $.NSBackingStoreBuffered, false);
  const wv=$.WebView.alloc.initWithFrameFrameNameGroupName(rect, $(), $());
  if (wv.isNil()) return 'legacy WebView unavailable';
  win.contentView.addSubview(wv);
  wv.mainFrame.loadRequest($.NSURLRequest.requestWithURL($.NSURL.fileURLWithPath(inPath)));
  const deadline=Date.now()+10000;
  while (wv.isLoading && Date.now()<deadline) $.NSRunLoop.currentRunLoop.runUntilDate($.NSDate.dateWithTimeIntervalSinceNow(0.05));
  if (wv.isLoading) return 'timed out loading';
  $.NSRunLoop.currentRunLoop.runUntilDate($.NSDate.dateWithTimeIntervalSinceNow(0.2));
  const docView=wv.mainFrame.frameView.documentView;
  const pw=Math.round(w*scale), ph=Math.round(h*scale);
  const big=$.NSBitmapImageRep.alloc.initWithBitmapDataPlanesPixelsWidePixelsHighBitsPerSampleSamplesPerPixelHasAlphaIsPlanarColorSpaceNameBytesPerRowBitsPerPixel(null,pw,ph,8,4,true,false,$.NSCalibratedRGBColorSpace,0,0);
  big.setSize($.NSMakeSize(w,h));
  $.NSGraphicsContext.saveGraphicsState;
  $.NSGraphicsContext.setCurrentContext($.NSGraphicsContext.graphicsContextWithBitmapImageRep(big));
  docView.displayRectIgnoringOpacityInContext(rect, $.NSGraphicsContext.currentContext);
  $.NSGraphicsContext.restoreGraphicsState;
  const png=big.representationUsingTypeProperties($.NSBitmapImageFileTypePNG,$.NSDictionary.dictionary);
  return (png.writeToFileAtomically(outPath,true)?'ok ':'write failed ')+pw+'x'+ph;
}
```

Width and height are the SVG's own `width` and `height` attributes, which
the session reads from the file. The other two commands:

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

## How the skill changes, from the experiments

Sized S once the decisions below stand: one new section in `SKILL.md`, one
script file beside it, no binary change, `needs` stays at v2.0.0.

**Where.** A new `## If a picture is refused` section after `## If the
request is a dry run` and before Step 1, in the voice of the two `If`
sections already there. Step 1 is the wrong home: the refusal comes out of
`build` or `publish`, not out of the front matter.

**The one sentence the section must carry about publish-once.** The
picture refusal happens on this machine before anything leaves it: no
document was created and the note is untouched, and the error says so. So
fixing the picture and running again is not a second publish. The section
says that in plain words, then tells the session to run `build` to the
scratchpad after the fix, so the note is proven acceptable with no network,
and to run `publish` once after that. That keeps "Publish is one call"
true as written.

**The script's home.** `skills/gdoc-publish/svg2png.js`, beside `SKILL.md`,
no subfolder. All four routes a colleague gets a skill by carry the folder
whole: the plugin, the zip (`cp -R skills` in `release.yml`), `install.sh
--skills` (`cp -R "$src/skills/$skill"`) and the developer symlink. No
list has to learn the file's name. The tests that read skills touch only
`SKILL.md` (`skills_test.go`, `plugin_test.go`), so they neither break nor
notice; `make test` stays the gate. The skill names the script by its
place, "beside this file, in the base directory Claude Code printed when it
loaded this skill", never by a path, because
`TestNoSkillNamesAPersonOrAMachinesPath` bans `/Users/` and `~/src/` and a
colleague's checkout is not this one.

**The order the section states, short on purpose.** `release/platforms`
lists two darwin pairs and nothing else, so every colleague the release
reaches is on a Mac and the WebKit script is the route. The section names
one fallback, a Chromium-family browser by its app path, for a macOS that
has dropped the legacy `WebView`, and the stop. The Linux and Windows
lines above stay in this item and go into the skill when
`windows-rollout-checklist.md` lands and `release/platforms` grows, not
before.

**What the session does, step by step, as the section will put it.**

1. Read the refusal: it names the line and the file. Read the SVG's root
   element for `width` and `height`; when only `viewBox` is there, take its
   third and fourth numbers.
2. Render with the script at scale 2 to the scratchpad first, and look at
   the PNG before touching the hub: arrowheads present, text at the right
   weight. The two dead ends were both discovered by looking, not by an
   exit code.
3. The PNG goes beside the SVG with the same stem. A PNG already there
   under that name is a stop, not an overwrite, because the session cannot
   know whether it is stale output or somebody's own picture.
4. Repoint the one link on the named line from `.svg` to `.png`. The SVG
   stays; it is the master.
5. Run `build` to the scratchpad. A clean object means publish, once.
6. In the reply, beside `files_changed`, say what the binary cannot: which
   SVG, which PNG, which line was rewritten. The hub may not be under git,
   so the reply is the only record.

**The reply shape in Step 3.** One added bullet: a picture the session
rendered is a change to the hub the binary did not make and cannot list, so
the session lists it itself, file and line.

**Rollout, which is not the skill's problem but is Nail's.** On this Mac
`/altery:gdoc-publish` loads from the plugin cache, which holds v2.1.0
while the latest tag is v2.3.4, and `~/.claude/skills` carries no gdoc
links. So an edit to `skills/` reaches nobody's session, this machine
included, until a release moves the plugin. The skill change ships with the
next `make tag`.

**Done when.** `make test` is green; a scratch note with one SVG is refused
by `build`, the section is followed as written, and `build` then accepts
it; the reply names the file and the line; and `docs/guide/publishing.md:62`
gains one sentence saying the publish skill knows the route.

Open edge, not for the skill to solve: the PNG is derived output living beside
its source, and an edited SVG leaves a stale PNG and a stale document with no
warning. One other hub note links an SVG today, so a check is not worth
building yet.
