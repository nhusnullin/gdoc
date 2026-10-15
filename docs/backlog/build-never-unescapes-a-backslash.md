---
worth: yes
added: 2026-10-15
---
# `build` never unescapes a backslash, so every escape gdoc writes reaches the document as a backslash

Found in the M18 review on 2026-10-15.

`internal/body` hands goldmark's text through as it stands. `decodeEntities` resolves entity
references, because goldmark leaves those to its HTML renderer, and nothing resolves the other half
goldmark leaves to that renderer: a backslash in front of ASCII punctuation. Measured:

| note | document |
|---|---|
| `A \[bracket\] here.` | `A \[bracket\] here.` |
| `A \{+ marker here.` | `A \{+ marker here.` |
| `A \*star\* here.` | `A \*star\* here.` |

Both pandoc and CommonMark drop the backslash. So does goldmark, in `util.UnescapePunctuations`,
which its own `html.Writer.Write` calls and this package does not.

Two routes depend on the escape being read back, and both are documented as working:

- `internal/view` escapes `[` and `]` inside a link's words, so a document whose link text holds a
  bracket exports to `[a\]b](url)` and publishes back with the backslash in it.
- `internal/markers` states the parity rule, and `docs/guide/exporting.md` and
  `skills/gdoc-export/markers.md` both tell a reader to write `\{+` for a marker of their own. That
  file publishes with `\{+` in the document.
  `docs/backlog/the-suggestion-marker-is-refused-but-not-escaped.md` names the same expectation.

**Why it was not fixed in the M18 review.** It is one line to call `util.UnescapePunctuations`, and
the one line is wrong: goldmark resolves the backslash and the entity in a single pass, so `\&amp;`
is the literal `&amp;` there and is `&` whichever order two passes run in. Doing it properly is a
small single-pass writer in `decodeEntities`, it changes what every note carrying a backslash
publishes, and that is a decision about whether gdoc's markdown is CommonMark's on this point. Nail's
call, not a refactor.

**What a fix looks like.** One pass over the text node: a backslash before ASCII punctuation writes
the punctuation and drops the backslash, an `&` begins an entity, anything else is itself. Then
`internal/view`'s escaping round-trips and the two guide pages are true.

Pins to write with the fix: a note holding `\[a\]` publishes the words `[a]`; a document whose link
text holds a bracket survives export and publish unchanged; a note holding `\{+` publishes `{+` and
one holding a bare `{+` is still refused.
