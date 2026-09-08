---
worth: yes
added: 2026-09-08
---
# A second numbered list carries on from the first

`word/numbering.xml` defines two lists, `numId` 1 for bullets and `numId` 2 for numbers, and
`internal/render/numbering.go` writes exactly those two. `internal/body` names them, so every numbered list
in a note points at the same `numId`. Word and Google both read that as one list, so a document with two
numbered lists numbers the second one 4, 5, 6 rather than 1, 2, 3, and a list written as `5.` in the
markdown starts wherever the previous one stopped.

Found while porting the body walker in M5 Task 4, 2026-09-08. Bulleted lists do not care, because a bullet
carries no count.

The fix is a `numId` per list, which means the body has to tell the shell how many lists it found before
`numbering.xml` is written: today `render.Build` writes that part from the config alone and never sees the
body. Two shapes are possible, and choosing between them is a design decision rather than a patch:

- `body.Result` carries the lists it opened, and `render.Build` writes one `w:num` per list, all pointing at
  the same two abstract lists. Cheap, and it makes the numbering part depend on the body.
- The body writes list paragraphs with an explicit `w:numPr` override per item. Keeps the parts independent
  and duplicates the start value on every item.

`TestANumberedListNamesTheNumberedList` in `internal/body/body_test.go` is where the current behaviour is
stated, and the place to extend.
