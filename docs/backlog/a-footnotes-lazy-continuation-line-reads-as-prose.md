---
worth: later
added: 2026-10-15
---
# A footnote definition's lazy continuation line reads as prose

Found in the M18 review on 2026-10-15.

`scan.go` marks a line as a definition's own when it is the opener or is indented to the
definition's content indent. goldmark continues a definition's paragraph lazily as well: a line
indented nought to three spaces straight after `[^a]: the note` is still inside note `a`.

So in

```
Some text[^a].

[^a]: the note
  continued [^b] here

[^b]: other note
```

goldmark parses `[^b]` as a child of note `a`'s paragraph, and `proseLines` reports the line as the
document's own words. `checkFootnotes` then counts `[^b]` as a prose mark, so the refusal
"the footnote [^b] is defined and nothing in this file's own words marks it" never fires.

**Why it is not urgent.** Nothing is silent. The walk drops the mark because Word has one level of
note, and `footnoteBlocks` warns by line: `a footnote inside a footnote is not rendered in the house
style and was left out`. What is lost is the stronger answer, a refusal, for words that reach no
page.

**What a fix looks like.** Track whether the definition's paragraph is still open, which is one more
piece of state beside `base`: a non-blank line after the opener continues it at any indent, and a
blank line ends it. Then `line.inside` is true for the lazy line too and the refusal fires.

Pins to write with the fix: the note above is refused by the line the definition of `[^b]` is on, and
a definition followed by a blank line and a line of prose at the margin is still prose.
