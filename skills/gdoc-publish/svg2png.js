// Render an SVG to a PNG with WebKit, which ships with macOS.
//
//   osascript -l JavaScript svg2png.js in.svg out.png width height [scale]
//
// Width and height are the SVG's own width and height attributes, or the third
// and fourth numbers of its viewBox. Scale defaults to 2, which lands at about
// 400 ppi once gdoc scales the picture to the text column.
//
// It prints "ok <pixels>" when the file is written, and the reason otherwise.
// The legacy in-process WebView is what it uses: WKWebView renders out of
// process and its snapshot call takes a block this bridge cannot pass. WebView
// has been deprecated since 10.14 and is still present in 26.3. On a macOS that
// drops it the first line of the answer says so, and the skill falls through to
// the next renderer.
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
