---
worth: yes
where: go/internal/view/text.go:1031
added: 2026-10-03
---
# A line break hides from `read`, so a quote copied from it is refused

Found on 2026-10-03 while thinking through George's step 8 (change words in a
table cell as a suggestion). It was confirmed offline with a throwaway test that
built the document in the Docs API's own shape and ran it through `view.Text`
and `propose.FindSpan`. No live document was used.

A cell holding `£20` and `per month` on two lines:

| How the author broke the line | What `gdoc read` prints | Quote `£20 per month` | Quote `£20` |
|---|---|---|---|
| Enter: two paragraphs | `Business \| £20 per month` | refused | found |
| Shift+Enter: one paragraph, U+000B | `Business \| £20\vper month` | refused | found |

- With Enter, `text.go:1031` joins a cell's paragraphs with a space. The AI
  copies what it sees, and the quote crosses a paragraph mark, which
  `FindSpan` never matches.
- With Shift+Enter, `read` prints the raw U+000B. This happens in an ordinary
  paragraph too, not only in a table. The AI reads it as a space or drops it.
  Sent exactly, with the U+000B inside, the quote is found.
- Both are refused with "not found in the document as written; it may have
  changed, or it may be inside a pending suggestion". Neither cause it names is
  the real one, so the caller has nothing to act on.

This is a strong candidate for the failing shape in
`docs/backlog/propose-inside-tables.md`, which waits on exactly that.

## The obvious fix opens a new hole

Printing the break as `<br>` in `read` fixes a caller that changes one line,
and nothing else:

- A caller that changes the whole cell copies `<br>` into its quote. `FindSpan`
  finds no `<br>` in the document, so the quote is refused again.
- A caller that also writes `<br>` into the replacement gets the literal text
  `<br>` written into someone's document. Today `Proposal.Check` accepts a
  replacement holding U+000B, and nobody has measured what Docs does with that
  inside a suggestion.
- An author who typed `<br>` as text becomes ambiguous with a real break.

So the fix is four parts and two live checks, not one change to `read`.

1. `read` prints every break as `<br>`, in tables and in ordinary paragraphs,
   and escapes an author's literal `<br>` the way it escapes `|`.
2. `propose` reads `<br>` in a quote. A U+000B break places the change in one
   paragraph. A paragraph break inside a cell is refused by name: one proposal
   per line, because a block is not proposed inside a table.
3. `propose` refuses `<br>` in a replacement by name, until the live check
   below says what Docs does with a break inside a suggestion.
4. The not-found refusal names a line break as a cause, for a caller that
   copied the quote from somewhere other than `read`.

Two live checks in the test folder come first:

- What `gdoc comments` prints as a thread's anchored text when the range covers
  a two-line cell. A caller copying the quote from the thread is not helped by
  part 1.
- Whether a suggestion whose replacement holds U+000B lands and verifies. This
  decides when part 3 can be lifted.

Changing `read`'s output touches its golden files and any skill that quotes
from it, so the five skills are checked for that before part 1 lands.
