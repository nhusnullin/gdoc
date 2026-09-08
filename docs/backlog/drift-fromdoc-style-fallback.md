---
worth: yes
added: 2026-09-08
---
# `drift`'s Docs reader resolves no style, so an inherited value reads nil

`drift.Docx` resolves a value through docDefaults, `basedOn` and the paragraph
style, which is what makes `body text size` and `body font` answer on a master
whose runs state neither. `drift.Doc` reads only what the Docs answer states on
the run or the paragraph, and the Docs API leaves an inherited run property out
of `textRun.textStyle` altogether.

So on a converted document whose body runs inherit NORMAL_TEXT, `body text
size`, `body font` and the two `body Hn run colour` rows read nil on the Doc
side. Against a master that answers, that is a DIFFERENT row nobody wants; on
the live gate, which reads two Docs answers, both sides read nil and the row
comes back IDENTICAL measuring nothing. The heading colour rows are already in
`Known` for the export quirk, so neither gate would measure them at all.

`doc_test.go` hides it: the fixture sets those properties explicitly on its own
runs, and `TestFromDocReadsEveryItem` compares one answer against itself, so
every row is IDENTICAL by construction. 78 of the 169 items read nil from that
fixture and nothing states which ones are expected to. `Docx` has that pin, in
`unstated`.

Found by the M5 review, 2026-09-08. It costs nothing today, because the live
gate cannot run until M6 teaches the guard to read the first MIME part of a
multipart create and gives `gapi` a multipart write. It has to be settled with
that upload, not after it.

The work is: give `Doc` the same resolution chain `Docx` has, run to paragraph
style and then to the document's own defaults; normalise an absent colour the
way `Docx.Style` does; add fixture cases where the run inherits rather than
states; and pin the set of items the fixture is expected to answer, the way
`unstated` pins the docx half.
