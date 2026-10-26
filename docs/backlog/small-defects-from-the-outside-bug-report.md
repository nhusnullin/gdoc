---
worth: later
added: 2026-10-23
---
# Small defects from the outside bug report

An outside review of commit `7be00df` named sixteen bugs. The team checked
each one with a test against the current code. Nail chose to fix the render
fields, the Russian heading ids, the null mask in the guard and the two export
heading bugs. These are the rest. Each one is real, and none reaches a person's
document today.

- **drift, a header run is not resolved through its styles** (`drift/docx.go`
  `Segment()`). A run with no `w:sz` or `w:color` of its own reads as nil. gdoc's
  own render always writes both, so only another producer would trip it. drift
  is a test gate only: `internal/live/publish_test.go` is its one importer.
- **drift, `<w:u w:val="none"/>` reads as underlined** (`onOff`). Nothing reads
  `Style.Underline` today.
- **drift, a body paragraph's `jc` and spacing use the prefix from
  `styles.xml`** (`applyPara`). It only matters when `document.xml` binds
  WordprocessingML under a prefix other than `w:`. No known producer does
  that. `reference()` hard-codes `r:id` the same way.
- **render, `contentTypes` writes `image/jpg` and `image/svg`**. Unreachable,
  because `body/image.go` accepts only PNG and JPEG and names the file `.jpeg`.
  It becomes real the day `identify` accepts more.
- **body, an uppercase `DATA:` picture is refused** (`decodeDataURI`). The
  `TrimPrefix` is case-sensitive while `isDataURI` is not.
- **propose, the page-break refusal names the wrong line** after a setext
  heading or a closed ATX heading (`unpositionedLine`). Only the message is
  wrong.
- **view, the warning for a thread inside the contents list says the range
  "is not in the text of the tab"**. Refusing to place it is right, because
  `read` prints no contents list. The wording is wrong.

Not fixed by decision: `AllowFile` accepts any level, `LevelFull` included,
while the doc says "the only door to it is Learn". All four production callers
pass `guard.LevelSuggest` as a constant.

Worth `later`: each fix is a line or two. They wait for a milestone that
touches the same file anyway.
