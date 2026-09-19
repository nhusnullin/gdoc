---
worth: yes
where: go/internal/body/build.go:351
added: 2026-09-19
---
# Every heading carries a bookmark, and Google Docs shows each one as a flag

The M12 anchor-link fix (7e5fee8, 2026-09-18) writes a `w:bookmarkStart`/
`w:bookmarkEnd` pair around the runs of every heading that has words, whether
or not any `#` link in the note names it. Word hides bookmarks. Google Docs
keeps them on import and shows a blue flag on the heading whenever the cursor
lands there, so a published note with no internal links at all, such as the
Bridge sales-deck note published 2026-09-19, shows a flag on every heading for
nothing.

The rule in `body/doc.go` says every heading carries one "because a note is
edited after it is published". That does not hold: a later edit goes out
through a fresh publish, which rebuilds the docx and its bookmarks from the
note as it stands, so a bookmark nothing links to today buys nothing.

The fix is small. `headingAnchors` in `body.go:144` already pre-walks the note
and collects every heading id that will carry a bookmark. Collect the `#`
destinations the note's links name on the same pass, and have `heading` write
the pair only when its id is in that set. A note with no `#` link then
publishes with no bookmarks; a note that links to a heading keeps the one
bookmark that jump lands on. There is no way to keep the jump and hide that
one flag. The six golden files under `body/testdata/golden/` change, and
`TestEveryHeadingCarriesABookmark` in `anchors_test.go` becomes a test that a
linked heading carries one and an unlinked heading does not.

Nail's call, 2026-09-19, after seeing the flags in the Bridge document.
