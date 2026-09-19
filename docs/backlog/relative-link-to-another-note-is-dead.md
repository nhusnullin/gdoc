---
worth: yes
where: go/internal/body/build.go:295
added: 2026-09-19
---
# A relative link to another note publishes as an address that opens nothing

`[bybit](2026-08-20-bybit-card-regional-setups.md)` in a note's Related
section reaches `word/_rels/document.xml.rels` as an external relationship
whose target is the bare path, the same shape the `#` anchor links had before
M12. Word resolves it against the document's own location; in Drive that file
does not exist, so clicking the link opens nothing and nothing warns. Every
hub note ends with a Related section written this way, so every published
note carries a few dead links at the bottom.

Seen 2026-09-19 in the Bridge sales-deck note, whose four Related links all
point at other hub notes by relative path.

## Decision, Nail, 2026-09-19

Keep the file name as plain text. Related is hub navigation: a colleague
reading in Drive is not meant to follow it, only to know which notes sit
beside this one, and a note name is what they can search the hub for. So a
link whose destination is a relative path, no scheme and no leading `#`, is
written as its words with no hyperlink, no relationship and no `Media` entry,
and no warning, because nothing is wrong with the note.

Rejected: refusing it with a warning the way a dead `#` anchor is refused,
because the note is correct and the author has nothing to fix; and rewriting
it to the other note's Drive document from its `gdoc:` block, because it
makes publish order matter and needs a rule for a note not yet published.

The one open edge: a relative path that is not a note, such as a folder
(`202608-eagle-money-flow/`). Same rule, plain words, since the docx cannot
reach it either.

`TestALinkIsAHyperlinkWithARelationship` in `body/body_test.go:323` states
the current behaviour, and `TestAnEmailAutolinkCarriesTheMailtoScheme` beside
it is the shape the test for a scheme-less destination takes.
