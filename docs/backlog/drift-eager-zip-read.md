---
worth: no
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

`worth: no`, settled 2026-09-18. The question the item asked was whether this
reader ever opens a file that did not come from the master in `testdata/` or
from the builder in the same test process. It does not: `internal/drift` is
imported by nothing under `go/` outside its own tests, so no command can hand
it a file somebody sent. A ceiling on a reader nobody can reach from the
binary is code with no caller to protect. This stays as a `no` so the next
review of `OpenDocx` does not file it again. If a command ever imports
`drift` to read a file from outside the repo, read the parts by name or check
`f.UncompressedSize64` against a running total before decompressing, and
delete this file in that commit.

Found by the M5 review, 2026-09-08.
