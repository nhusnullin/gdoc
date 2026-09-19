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

Open edge, not for the skill to solve: the PNG is derived output living beside
its source, and an edited SVG leaves a stale PNG and a stale document with no
warning. One other hub note links an SVG today, so a check is not worth
building yet.
