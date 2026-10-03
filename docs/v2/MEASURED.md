# Measured

What Google does when it accepts. Kept separately from
[DECISIONS.md](DECISIONS.md) on purpose: a decision is a choice, and none of
these are chosen. It is the sibling of [BLOCKED-BY-API.md](BLOCKED-BY-API.md),
which holds what Google refuses. This file holds what it allows, and what shape
the answer comes back in.

Every section carries the date, the test that made the measurement, the verbatim
answer where there is one, and a line saying when to ask again. A measurement is
never edited to match a later reading. A different answer is a new section with
its own date, and the old one keeps its words.

## Which styling requests land in place

Measured 2026-09-09, by `TestLiveStyleFidelity` in
`go/internal/live/fidelity_test.go`. Corrected the same day, twice, and both
corrections are below with their own words.

Recheck when Google's release notes add a request kind, when `updateNamedStyle`
appears in the discovery document, or before anything is added to
`inPlaceKinds`.


The 2026-08-29 run measured **survival**: what an in-place `batchUpdate` does
not destroy. It never measured **fidelity**, and SPEC's sentence "the house
style is approximate" has stood in for a list nobody had written. Nail's
decision the same day was that M7b's scope comes from a measurement rather than
from reading `house.yaml`. This is the measurement.

`go/internal/live/fidelity_test.go`, run against a document it created in the
test folder and trashed afterwards. Each request kind was sent in its own batch,
so one refusal could not hide the rest, and the document was read back once at
the end. Nine of the first ten landed, and the three kinds the first run skipped
were measured the same day in commit `a5fc6e0`, so the table below is thirteen
rows and ten of them land:

| Request kind | Accepted | Landed |
|---|---|---|
| `updateDocumentStyle` (margins) | yes | yes |
| `updateParagraphStyle` (`namedStyleType`) | yes | yes |
| `updateParagraphStyle` (spacing, indent) | yes | yes |
| `updateTextStyle` (font family, size) | yes | yes |
| `updateTextStyle` (`foregroundColor`) | yes | yes |
| `updateTextStyle` (`backgroundColor`) | yes | yes |
| `createParagraphBullets` | yes | yes |
| `createNamedRange` | yes | yes |
| `updateParagraphStyle` (`borderBottom`) | yes | yes |
| `updateTableCellStyle` (shading, padding) | yes | yes |
| `updateNamedStyle` | **no** | no such request kind |
| `updateParagraphStyle` (`tabStops`) | **no** | read-only in the reference |
| `createHeader` (`FIRST_PAGE_HEADER`) | **no** | `HeaderFooterType` is `UNSPECIFIED` and `DEFAULT` only |

**The limit is durability, not fidelity, and that is the sentence to tell
somebody before a restyle.** `updateNamedStyle` does not exist, so the nine
named styles cannot be redefined. But assigning a paragraph to `HEADING_1`
lands, and overriding its visual properties per paragraph lands, which is
exactly what the master template does: it states a heading colour on the style
and overrides it on every paragraph, which is why eight rows sit in
`drift.Known`. So an in-place restyle reaches the look. What it cannot do is
make the **next** heading the author types inherit it.

**Two things the M7 plan review listed as unreachable are reachable**, and the
plan passed them on without checking:

- The `highlight` house.yaml states as an OOXML name lands through
  `backgroundColor` with an RGB value. It needs a name-to-hex mapping, not a
  deferral.
- Bullets land through `createParagraphBullets`. The `BULLET_DISC_CIRCLE_SQUARE`
  preset is disc, circle, square, which is the `●○■` house.yaml asks for.

**Corrected 2026-09-09, and the correction changed M7b's scope.**
`createParagraphBullets` lands, which is true and incomplete. The reference says
the leading tabs that set a bullet's nesting level "are removed by this
request", so it deletes text the author typed, and it merges a bulleted range
into an adjacent list with a matching preset, renumbering their items. The probe
missed both because the content it wrote had no leading tabs and no neighbouring
list. So bullets are out of `inPlaceKinds`, a restyle leaves every list with
whatever bullets it has, and the milestone's defining property, that nothing it
sends can change a character, is literally true rather than nearly true.

**Corrected 2026-09-09.** This entry used to end by naming three things the run
did not test, so M7b would not assume them: `updateTableCellStyle`, tab stops,
and anything touching the first-page header. Commit `a5fc6e0` measured all
three the same day, once `writeBody` learned to lay a table down in a second
batch so a cell had something to be styled in, with the table's start index read
back rather than computed. They are in the table above, and this is what they
said:

- `updateTableCellStyle` **lands**. The shaded, padded header row `house.yaml`
  states for its three front-matter tables is reachable in place, which is
  better than expected and is why the kind is on `inPlaceKinds`.
- `ParagraphStyle.tabStops` is **read-only**, so the running head's tab-stop
  layout cannot be applied in place at all.
- `createHeader` with `FIRST_PAGE_HEADER` is **refused**, confirming the
  2026-08-29 measurement that `HeaderFooterType` is exactly `UNSPECIFIED` and
  `DEFAULT`. That is the reason the finishing checklist was invented, measured
  here rather than trusted from a note.

**The run was blocked for an hour by something unrelated**, recorded here
because the symptom is so misleading. On the office wifi every Go TLS 1.3
handshake times out, to every host, while `openssl s_client -tls1_3` succeeds on
the same network to the same address and Go capped at TLS 1.2 works in 0.1s.
gdoc reports it as `TLS handshake timeout` on the token refresh, which reads
like an expired token. A mobile hotspot fixes it. Never work around it by
letting gdoc fall back to TLS 1.2.

## In-place styling preserves anchors and pending suggestions

Measured 2026-09-10, by `TestLiveRestyleKeepsAnchorsAndSuggestions` in
`go/internal/live/anchors_test.go`. `TestLiveRestylePreservesTenFeatures` in
`restyle_test.go` is the fuller acceptance and has never run, for the reason the
last paragraph gives.

Recheck when a request kind is added to `inPlaceKinds`, because the property
measured here is that none of them changes a character.


Serves principle 3. This is the measurement M7b rests on, made against Google
rather than reasoned about, and it is the first time this code has asked.

**The run.** A copy of a real one-tab policy document, carrying one comment
anchored to its words and one pending suggestion, restyled in place with 181
styling requests in one batch: 49 paragraphs, 125 runs, 75 cells across 6
tables. Before and after, read through the docx export because `comments.list`
reports a destroyed anchor as healthy:

| | threads | anchored | pending |
|---|---|---|---|
| before | 1 | 1 | `suggest.r1tnorocsz3h` |
| after | 1 | 1 | `suggest.r1tnorocsz3h` |

Nothing moved. That is the 2026-08-29 measurement holding from the other side:
replacing a document's body destroyed 100% of comment anchors, 355 of 355
characters across three anchors, and recreated every suggestion id. In-place
styling destroyed none, and the suggestion kept its own id rather than being
recreated under a new one.

**The guard refused before the grant, in the same run.** `restyleCopy` sends one
styling request before `GrantInPlace` and requires a `guard refused` error back.
It got one. So the direct-edit door shut since M1 is really shut until one line
in one command opens it for one id, and that is measured rather than asserted.

**The source never moved.** Same revision id before and after, asserted on every
path out including the failing ones. The 2026-08-29 rule, held.

**The cheap acceptance is the one that gets run, and that is the decision here.**
`TestLiveRestylePreservesTenFeatures` needs a document holding all ten of SPEC
item 5's features, which is a document somebody builds by hand and keeps intact,
and on 2026-09-10 the document it was pointed at held none of them and carried
two tabs, which a restyle refuses outright. So it has never run.
`TestLiveRestyleKeepsAnchorsAndSuggestions` needs one anchored comment and one
pending suggestion, which is any document somebody has reviewed, and it answers
the question the milestone actually rests on. Both stay. An acceptance nobody
can run is not an acceptance, and the ten-feature run is still the fuller
answer for the day somebody rebuilds the document.

## A named range over a suggested insertion

Measured 2026-09-10, by `TestLiveNamedRangeOverSuggestionProbe` in
`go/internal/live/namedrangeprobe_test.go`. The accept row was read back by
`TestLiveNamedRangeAfterAcceptedByHand` in the same file, once Nail had accepted
the suggestion in the browser.

Recheck when a run meets a prelude accepted one paragraph at a time, which is
the shape the last paragraph says nothing here has seen.


Serves principle 3. It is a measurement rather than a decision, and it is here
because M7c's marker rests on it and nobody had asked.

M7c proposes the house prelude rather than writing it, so the named range that
marks gdoc's own prelude is created over text that exists only as a pending
insertion. `TestLiveNamedRangeOverSuggestionProbe` asked Docs, one fresh
document per case, for the reason the suggested-insert probe learned the hard
way: a probe that measures its own leftovers answers about itself.

| Question | Answer |
|---|---|
| created directly over a pending insertion? | yes, id `kix.wi79lhqfq91l` |
| what it covers while the insertion is pending | exactly the proposed line, `[1,23)` |
| after the suggestion is **rejected** | the marker is gone, no named range at all |
| after the suggestion is **accepted** | it survives: same id, same range, now over the accepted text |

The accept row was read separately, on 2026-09-10, and the reason it had to be
is the part worth keeping. **gdoc cannot accept its own suggestion, and that is
the guard working rather than a gap.** `judgeRequests` refuses every request kind
whose name carries "suggestion" before it looks at the level, so a document the
probe created a second ago is refused like anybody else's. `rejectSuggestion`
has exactly one door, `AllowReject`, which `withdraw` already opens from the
note's own record, so the reject case ran through it with the probe's own id.
`acceptSuggestion` has no door. Opening one so that a probe could measure itself
would be widening the guard for the tail rather than the dog, so the probe left
its document in the test folder, printed the URL, and Nail accepted it in the
browser the way he will accept a real prelude. The read-back is
`TestLiveNamedRangeAfterAcceptedByHand`, which reads and writes nothing.

**What it settles.** M7c's Task 6 is the replace-in-place design rather than the
fallback, and `AllowMarker` has a production caller. A second run meets three
shapes and each has an answer: a marker, which it replaces; no marker, which is
a document gdoc never touched or one whose prelude was rejected, and both are
proposed into cleanly; and a marker over a prelude still pending, which is
refused, because accepting or rejecting the one already in front of him is
Nail's and not gdoc's.

**What it does not settle.** Whether the marker survives an accept the author
makes one paragraph at a time, rather than the whole prelude at once. The probe
accepted a single insertion. A prelude is many, and Docs numbers a partial
accept differently. The milestone that meets a half-accepted prelude measures
that; until then a marker whose range no longer covers a whole prelude is a
shape nothing here has seen.

## A table takes one index of its own at the end

Measured 2026-09-10, by `TestLiveTableIndexProbe` in
`go/internal/live/tableindexprobe_test.go`, after a failed live run of
`TestLivePreludeIsProposedNotWritten`. The verbatim refusal that started it is
below.

Recheck when `internal/prelude`'s index arithmetic changes, because every index
it computes is read from this map and nothing reads it back.


**Found by a failed live run, not by review.**
`TestLivePreludeIsProposedNotWritten` sent the whole house prelude at the
document Nail named, and Docs refused the batch whole:

```
Invalid requests[106].insertText: The insertion index must be inside the bounds
of an existing paragraph. You can still create new paragraphs by inserting
newlines.
```

Nothing was written. A batch Docs refuses is refused whole, everything in it was
a suggestion in any case, and the source document's revision never moved. The
guard was never in question: the requests it carried are the ones
`internal/propose` sends every day.

Request 106 is the spacer newline between two front-matter tables, at the index
`internal/prelude` computed as one past the first table's last cell.

**The measurement.** `TestLiveTableIndexProbe` in `internal/live`, one throwaway
document per case, everything in SUGGEST mode, which is the mode the prelude
sends in. A 2x2 table of empty cells asked for at index 1 reads back as:

| Element | Range |
|---|---|
| paragraph, the newline `insertTable` writes in front | [1,2) |
| **table** | **[2,14)** |
| row 0 | [3,8) |
| cell 0.0, and its paragraph | [4,6), [5,6) |
| cell 0.1, and its paragraph | [6,8), [7,8) |
| row 1 | [8,13) |
| cell 1.0, and its paragraph | [9,11), [10,11) |
| cell 1.1, and its paragraph | [11,13), [12,13) |
| paragraph, what follows the table | [14,15) |

The last cell ends at 13 and the table ends at 14. So a table takes one index of
its own at the end that no row, no cell and no paragraph mark accounts for, and
the paragraph behind a table begins at the table's own `endIndex`. An empty
table is twelve units for a 2x2, which is one for the table, one per row, one
per cell plus one for that cell's paragraph mark, **and one for the table's own
end**, with a thirteenth unit for the newline in front of it.

**Accepted is not the answer, and that is the trap.** The sweep asked six
indexes, each on its own document, each in one batch with the table:

| Index | Accepted | Where it went |
|---|---|---|
| 11 | no | inside no paragraph |
| 12 | **yes** | **inside the last cell**: the table then spans [2,15) |
| 13 | no | the table's own end, inside no paragraph |
| 14 | yes | behind the table, which still spans [2,14) |
| 15, 16 | no | past the end of the body |

12 is the answer that reads like success and is not one. The probe's first run
reported the first accepted index and stopped there, so it said 12, and a second
table asked for at 13 was nested inside that cell rather than put behind the
first. Every accepted candidate is read back now, and the verdict is where the
insert really landed. A probe that stops at the status code answers a different
question from the one it was asked.

Two tables with one paragraph between them, spaced at 14, land as two top-level
tables at [2,14) and [16,28), which is the shape the house front matter has.

**The fix is one line and one comment**, `b.at = at + 1` at the end of
`prelude.builder.table`, with `TestTheIndexBehindATableIsTheTablesOwnEnd`
stating the measured map as numbers. Nothing else moved: the per-cell
arithmetic was already right, and the two facts measured on 2026-09-10 still
hold, that a 2x7 table inserted at 279 puts the first cell's content at 283 and
that a cell holding ten characters puts the next cell's content at 295.

**"Twelve marks for one 2x2" was not a measurement of indexes, and reading it as
one is what put the bug there.** The suggested-insert probe counted the elements
Docs recorded a `suggestedInsertionIds` on, and that count agrees with the
arithmetic by coincidence: twelve marks, and twelve index units for the table
plus one for its newline. A count of marks says nothing about where a table
ends. The rule this leaves is the project's own: a number that reaches the code
is measured against the question the code asks, or it is a guess wearing a
measurement's clothes.

**The live acceptance passes.** Re-run the same evening with the fix in:
250 requests in one batch, 70 paragraphs, 3 tables and 34 cells all carrying
suggestion ids, nothing written, the marker over [1,1243), the author's own text
character for character what it was, and the source document still on the
revision it started on.

## A proposal into a table cell lands

Measured 2026-09-18, by hand with the released command, on a document
`gdoc publish` created in the test folder from `release/example/first-note.md`
and its two-column table (`1jkKjzNSu5EzaTQZ-eYy4UZQEcucN6S7mwTfDKIK02PA`, left
in the folder with the suggestion pending so it can be looked at).

Recheck on the document Nail saw it fail on, 2026-09-08, or on a table with
merged cells, a nested table, or a quote that runs across a cell boundary. None
of those was measured here.

`docs/backlog/propose-inside-tables.md` records Nail's finding from the M4
acceptance run that a suggestion into words inside a table did not land, and
lists three suspects. The first step it asks for is a reproduction, and this is
it. The quote `The system the account reaches` sits alone in one cell of the
example note's table. The same command was run twice, once into that cell and
once into an ordinary paragraph as a control, each with its own probe:

| Target | `sent` | `verified` | `suggestions_inline` | `preview_without_suggestions` | `docx_anchored` |
|---|---|---|---|---|---|
| ordinary paragraph | true | true | true | true | true |
| table cell | true | true | true | true | true |

`gdoc read` afterwards prints the cell as
`System | {+[[c:AAACHTigeJI]]The system the account can reach[[/c]]+}[s:...]{-The system the account reaches-}[s:...]`,
so the suggestion, its comment and the docx witness all agree that it landed
inside the cell. None of the three suspects in the backlog item fired on this
shape: the comment range stayed inside one cell, `deleteContentRange` took the
cell text, and the witness found the anchor under `w:tbl`.

What this does not say is why the 2026-09-08 run failed. That document, its
table's shape and the envelope it printed were not kept, so the defect is
unreproduced rather than absent.

## A block of new paragraphs, proposed in one SUGGEST batch

Measured 2026-09-27 by `TestLiveBlockProposalProbe`
(`go/internal/live/blockprobe_test.go`), one throwaway document per case in
the test folder, each set up as five paragraphs with the fourth a real bullet.
Asked for `docs/backlog/propose-cannot-add-paragraphs.md`. Recheck when a
block proposal comes back with more than one suggestion id on text it only
inserted.

| Case | Suggestion ids | What the read-back shows |
|---|---|---|
| Heading, body and two list items inserted at the start of the next paragraph, each style stated, one run bold | 1 | insertion, text style, heading style and the new bullets all under the one id. No separate paragraph-style or bullet id |
| The same block, plus `deleteContentRange` over two whole paragraphs behind it | 1 | the deleted paragraphs carry the same id as the insertion. A replace is one suggestion |
| Text inserted at a list item's end, before its newline, nothing stated | 1 | every new paragraph is bulleted |
| The same, plus `deleteParagraphBullets` over the new text | 2 | the new paragraph with a new mark loses its bullet under the insertion id. The last one owns the item's old mark, keeps its bullet on screen, and carries a second id for the bullet change |
| Whole paragraphs inserted at a list item's start, nothing stated | 1 | every new paragraph is bulleted |
| The same, plus `deleteParagraphBullets` over the new paragraphs | 1 | no bullets, and nothing beyond the insertion id |
| A block after the last paragraph, before the body's final newline, a heading style stated | 1 | the anchor paragraph gets a new mark under the insertion id, and the last new paragraph owns the old final mark with no id of its own, because nothing restated it |

What follows for a block proposal:

- **Insert at the start of the paragraph after the anchor, never at the end of
  the anchor.** Then every new paragraph has a mark of its own, and every style,
  bullet and bullet removal folds into the one insertion id. Inserting at an
  end hands the anchor's old mark to the last new paragraph, and a change to
  that mark is a second suggestion.
- **Text takes the list membership of the paragraph it lands in front of.**
  Stating `namedStyleType` does not clear a bullet. `deleteParagraphBullets`
  over the new paragraphs does, as a suggestion under the same id. This also
  answered the open question a backlog item held about the house prelude,
  which now sends the same request over its own range (`go/internal/prelude/doc.go`).
- **After the last paragraph there is no paragraph to go in front of.** The one
  case measured came back with one id because the last new paragraph was not
  restyled. A restyled last paragraph there was not measured.

## Claude Desktop and a local MCP server

Measured 2026-10-03 by Nail on his Mac, with a throwaway stub server outside
the tree and not committed: one Go file speaking MCP over stdio, installed as a
thin `.mcpb` whose command is the stub's absolute path. It logged every line
Claude Desktop sent and every answer, and offered four tools: a read, a write,
a sleep and a held write that registers a one-time `confirm_<id>` tool. These
are the twelve measurements the M14 specification
(`docs/plans/2026-10-02-gdoc-v2-m14-chat.md`) puts before any server code.
Recheck before run 2 ships if Claude Desktop has been updated since, and
whenever a release note mentions extensions, tool permissions or voice.

| # | Question | Answer |
|---|---|---|
| 1 | Does a `.mcpb` start a command outside the bundle, and does the setting show as optional? | Yes, it installs and starts. The one `user_config` string reaches the server as `--trusted-email-domains=` when empty. Settings shows the field with its description, which also fills the placeholder, and a Save button. Nothing marks it optional or advanced |
| 2 | After the binary is replaced, does a restart run the new one? What does a missing binary show? | Toggling the extension restarted the chat process on the new binary; the agent-mode process kept the old one until Claude Desktop was quit. A missing binary shows a toast, "MCP gdoc spike: No executable file at <path>. It may be missing, a directory, or not marked executable. Check the server's command or remove the server", with "Open developer settings"; the tools vanish from chat |
| 3 | What comes first, and which protocol version? | `initialize`, never `server/discover`, asking `2025-11-25`. Two clients each start their own process at the same moment: `claude-ai` (chat) and `local-agent-mode-<extension name>`, which also declares `roots`. Both stay running |
| 4 | Does the model see the server instructions? | Not in chat: in two new chats the model did not know a word that appeared only in the instructions, and reached for a tool to find it. A Claude Code session in the same app that has the extension does show them |
| 5 | The tool timeout | 240 seconds, counted from when the call reaches the server, which is after the card is approved. A 120-second call finished; a 300-second call got `notifications/cancelled` with reason "SdkError: Request timed out" at 240.0 s, and the chat showed "No result received from client-side tool execution after waiting 4 minutes" |
| 6 | Are two calls sent before the first is answered? | Not observed. Asked for two at once, Claude Desktop sent the second 10 s after the first answered, each behind its own card. Not measured with both tools on always-allow |
| 7 | A card before a write tool? Elicitation? | A card before every tool by default, read-only ones too. Settings sorts tools by `readOnlyHint` into "Read-only tools" and "Write/delete tools", each with allow, approval and block per tool and per group. The client declares no `elicitation` and no `sampling`, and does declare an `io.modelcontextprotocol/ui` extension |
| 8 | Are local tools offered in voice? | On the Mac, yes: a voice session called the read tool once. Speech recognition turned spoken tool names into other words ("Calls by Creed", "GDoc expansion"), so later turns never reached the tool. The phone was not measured |
| 9 | Does a tool added mid-chat get a fresh card, also after always-allow? | Yes. `notifications/tools/list_changed` made the client fetch the list again within 3 ms. `confirm_<id>` got its own card at "Needs approval", and again when the whole "Write/delete tools" group was on always-allow: the group was switched before the hold created the tool, and the new name still got a card |
| 10 | What does the card show? | Every argument, in full, labelled with its property name. A 1,526-character text scrolled with its paragraph breaks kept. The order is the order the model sent the arguments, once schema order and once alphabetical, not the order the server declared. The model copied the long text back byte for byte. The phone was not measured |
| 11 | The quiet gap between a hold and its release | A person releasing after reading the card: 95 s and 121 s. The server sees the confirm call only after the card is approved. The model ended its turn after a hold both times it was tried, once refusing an explicit ask to release in the same reply |
| 12 | Does a comment that names an email address notify that address? | Not measured; Nail skipped it on 2026-10-03 |

Two other facts the log showed:

- **Changing a tool permission restarts the chat's server process.** A hold
  waiting in it is lost, and nothing is posted.
- **Only the chat process restarts on a toggle.** The agent-mode process keeps
  running, so per-process state lives twice and a toggle never reaches both.

What follows for run 2 is a decision, not a measurement, and goes to
DECISIONS.md: the spec assumed one process (3) and a 60-second timeout with a
45-second deadline (5), and both answers differ.

## Not measured yet

Two of the M14 measurements above stayed open on 2026-10-03: what the phone
shows in voice mode, for a local tool and for its card (8 and 10), and whether
Google emails an address named in a comment that gdoc posts (12). Recheck the
first when gdoc's server first runs in voice on a phone. Recheck the second
with one `gdoc annotate` on a document in the test folder whose text names an
address the person can read, before the Link hold's rule for email addresses
is changed.

The seven paragraph elements `internal/docs` decodes are a fixture built from
the reference, not from a document. Recheck when somebody reads a real document
holding a person chip, a date chip and a calendar link through `documents.get`
and compares the answer against `go/internal/docs/testdata/elements.json`; a
difference is recorded as its own section here, and the fixture is corrected
rather than a test loosened. The decision is DECISIONS.md, 2026-09-09, "Seven
paragraph elements were dropped at decode".

Three things `export` rests on are not measured either. The fixture document
Task 1 of the M13 plan describes has not been made, so
`TestLiveExportMeasurements` has not run: it is written, gated on
`GDOC_LIVE_TEST` with `GDOC_LIVE_EXPORT_DOC_ID`, and it prints its answers
rather than asserting them, because they are the measurement.

| What | Why it matters | What the test prints |
|---|---|---|
| Whether the Docs read and the docx export list a document's pictures in the same order | the pairing between the two routes is order and nothing else, DECISIONS.md 2026-09-19 | every inline and positioned object id per tab in body order, beside every `r:embed` in `word/document.xml` order with its media part name |
| Whether a PNG that `publish` uploaded comes back from the docx export byte for byte | `Options.MatchByHash` keeps a note's own picture file, so the SVG it was rendered from stays the master. It ships `false` until this holds | the sha256 of the first media file against the sha256 of the uploaded PNG on disk |
| What the docx export carries for a document's second tab | a tabbed document exports one file per tab, and a picture has to follow its own tab | the count of `w:drawing` in the docx against the count of objects in the first tab and in the second |

Recheck by making the fixture document in the test folder: one inline PNG
picture, one Google Drawing, one floating picture, and a second tab titled
`Appendix` with one picture, in that order. A difference is its own section
here. If the second answer says the bytes are equal, `MatchByHash` is flipped to
`true` in one commit with that section, and `TestLiveExportHashEquality` turns
from a print into an assertion.
