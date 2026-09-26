---
worth: yes
where: go/internal/restyle/style.go:222
added: 2026-09-25
---
# restyle flattens the cover of a document gdoc published

## What happened

Hub session `fdf1cd3b-d6d7-4528-a0cf-882ea655b601`, 2026-09-25. The public API v2 notes were
published with `gdoc publish`, so the document opened with the house cover: "Altery Group " at 29pt
bold, centred, and the title under it. Later the session ran `restyle --dry-run` and `restyle --from`
on the same document. After it, the cover lines read, from the Docs API:

| Index | Text | namedStyleType | Alignment | Run style |
|---|---|---|---|---|
| 6 | `Altery Group ` | NORMAL_TEXT | JUSTIFIED | 12pt, nothing else |
| 21 | `Public API v2: terminology review notes` | NORMAL_TEXT | JUSTIFIED | 12pt, nothing else |

That is the house body look, exactly. The cover's size, centring and font are gone, which is what
Nail saw as "the title font, size and position changed".

## Root cause

`restyle` reads structure from `namedStyleType` alone and never from text (`style.go`, "The
structure is read, never decided"). The docx `publish` uploads carries the cover as ordinary
paragraphs with direct formatting, and Drive's conversion turns them into NORMAL_TEXT. So to the
walk, the cover is body prose, and `paragraph()` sends it the body paragraph look and text look.

The one thing that makes the walk step around gdoc's own words is the skip span, and the span comes
only from the `gdoc:house-prelude` named range. `restyle --fields` writes that range. `publish` does
not, so a published document has no named range (the survey said `named_ranges: []`) and nothing
protects its cover.

The skill was also used outside its scope. `gdoc-restyle` says it is for "a Google Doc gdoc did not
write", and this one gdoc wrote. But the binary did not refuse either, and it could not tell, because
nothing in the document says gdoc made it.

## Chosen fix (Nail, 2026-09-26)

Two decisions taken, both on the recommended option:

- **publish marks its cover, ending at the contents list.** After the upload, `publish` writes a named
  range over [1, end of the one top-level table of contents). That is the boundary `export` already
  strips to (`byLayout` in `internal/export/strip.go`). None or two contents lists: no marker, and a
  warning. The document is at the full level, so the guard needs no new grant (`createNamedRange` carries
  there today). Read the marker back, and report it as `marked` beside `verified`, not inside it: an
  unmarked document is still correct.
- **A second marker name, `gdoc:house-published`, and `restyle --fields` refuses it.** The published
  cover is already the house cover, and replacing it would propose deleting the contents list too. So
  `prelude.Decide` reads the published marker first and refuses, saying to run without `--fields`.

Found while sizing it, and part of the same fix:

- **A plain `restyle --from` reads no marker at all today.** `styleDocument` in `cmd/gdoc/restyle.go`
  only skips the span its own phase 1 just proposed. So marking in `publish` alone changes nothing: the
  plain run has to read both markers and skip every span they cover, which means
  `restyle.TabRequestsExcept` takes a list of spans rather than one. This also fixes an accepted
  `--fields` prelude, which a later plain run flattens the same way.
- `docs.TOC` decodes no index today and needs `StartIndex`/`EndIndex`.
- SPEC.md describes the marker, so this starts with a DECISIONS.md entry.

Not reached: documents published before the fix carry no marker. The `gdoc-restyle` skill should say to
stop on a document it knows `publish` made when the survey shows no `gdoc:house-published`.

Tests to write: TOC span decoded; publish marks to the contents end, skips with no contents list,
reports an unread-back marker and a failed marker batch as warnings; `Decide` refuses the published
marker; the two marker readers read only their own name; the walk skips two spans; a plain restyle walks
past a published cover and past an accepted prelude; `--fields` on a published document writes nothing.
The live publish test should assert `marked`, because only Drive's real conversion proves where the
contents list ends.

A first cut exists as an unreviewed reference, not merged: the patch outside the repo at
`~/src/personal/gdoc-publish-marks-its-cover.patch` (22 files, all unit tests green on 2026-09-26, no
live run, stopped before review). The plan may use it or start over.
