---
worth: later
where: go/internal/drift/docx.go:75
added: 2026-09-08
---
# `drift`'s docx reader decompresses every member, with no ceiling on the total

`OpenDocx` walks every entry in the archive and decompresses it into
`d.parts`. `readEntry` caps one member at `maxPart`, 32 MB, and nothing caps how
many members there are or what they come to together. So an archive of N
high-ratio members costs N x 32 MB of memory before the first value is read.

`internal/docx`, the export reader, does not have this shape: it reads members
by name, so it touches only the four or five parts it looks at. This reader
loads the logo PNG, every media file and every part it never opens.

Two things it is not. There is no zip-slip here, because nothing is written to
disk. Go's `encoding/xml` does not expand DTD entities, so a billion-laughs
archive is not reachable either.

`worth: later` because the value decision is unresolved rather than the work.
The two inputs are the master committed in `testdata/` and a document this same
process wrote a moment earlier, both in an offline test. What would settle it is
whether this reader ever opens a file that did not come from one of those two:
the live gate at M6 reads Docs answers rather than archives, so the answer today
looks like no. If it stays no, the honest close is a `no` recording that the
reader is test-only, not a fix. If any command ever hands it a file somebody
sent, read the parts by name or check `f.UncompressedSize64` and a running total
before decompressing.

Found by the M5 review, 2026-09-08.
