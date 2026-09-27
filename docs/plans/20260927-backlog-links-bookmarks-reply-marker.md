# Three backlog items: note links, heading bookmarks, the reply marker

## Overview

Three items from `docs/backlog/` whose fix and decision are already written
down. None needs a new decision, a live document or a new dependency.

- `relative-link-to-another-note-is-dead.md`: a link to another hub note
  publishes as a hyperlink that opens nothing in Drive. Nail decided on
  2026-09-19 that it becomes its own words, with no hyperlink.
- `bookmark-only-linked-headings.md`: every heading carries a bookmark, and
  Google Docs shows each one as a blue flag. Nail decided on 2026-09-19 that
  only a heading some `#` link names keeps one.
- `a-marked-reply-carries-no-marker.md`: a reply has no `marker` field, so a
  marked reply is the easiest instruction in a listing to drop. The fix is the
  field plus the missing line in the review skill's two report shapes.

Each item is one task and one commit, and that commit `git rm`s its backlog
file.

## Context (from discovery)

- `go/internal/body/inline.go:118`: the `*ast.Link` case sets `Run.Link` to
  the destination as written. `*ast.AutoLink` is a separate case and always
  carries an address or `mailto:`.
- `go/internal/body/build.go:255` `addRuns`: `#` goes to a `w:anchor`,
  anything else to a relationship through `r.linkID`.
- `go/internal/body/body.go:389` `headingAnchors`: the pre-walk that collects
  every heading id that will carry a bookmark. `headingBookmark` (body.go:673)
  names the bookmark for every heading, and `heading` (build.go:351) writes the
  pair when the name is not empty.
- `go/internal/body/doc.go`: the anchor-link section states "every heading ...
  carries one ... because a note is edited after it is published", which the
  bookmark item says does not hold.
- `go/internal/comments/comments.go:101` `Reply` and `replies()`;
  `markerOf` already exists and is exact. `comments/doc.go` lists the fact
  fields under "Facts, never verdicts".
- `skills/gdoc-review/SKILL.md`: Step 1 lists the reply fields, Step 3 and
  Step 8 print the report shapes.
- No SPEC.md text lists the reply fields or the bookmark rule. DECISIONS.md
  2026-09-18 does state the bookmark rule, so Task 2 supersedes it there.
- None of the six golden notes under `body/testdata/docs/` carries a relative
  link or a `#` link, so every heading in the goldens loses its bookmark.

## Development Approach

- **testing approach**: TDD. The failing test first, then the smallest change
  that passes it.
- One task at a time, all tests green before the next.
- A `doc.go` rule names its test (CLAUDE.md).
- `make test` and `make vet` after each task.

## Testing Strategy

- Unit tests in the package that changes. The golden bodies are rewritten with
  `go test ./internal/body -update` and the diff is read, not trusted.
- No e2e: the live tests need Nail's variables and are not in scope.

## Progress Tracking

- `[x]` when done, ➕ for a task found on the way, ⚠️ for a blocker.

## Solution Overview

**Relative links.** Decide in `inline.go`, at the `*ast.Link` case: a
destination with no URL scheme that does not open with `#` is not a link, so
the run carries no `Link` and `addRuns` writes the words alone. No warning,
because the note is correct. Autolinks never reach this branch.

**Bookmarks.** The pre-walk also collects the `#` destinations of every
`*ast.Link` in the note. A heading keeps its bookmark only when its id is in
that set and it is a heading the walk will emit. `r.anchors` stays what a link
resolves against (every heading that could carry one), so a dead `#` still
warns. `headingBookmark` returns empty for an unlinked heading.

**Reply marker.** `Reply` grows `Marker string json:"marker"`, set from
`markerOf`. The skill lists the field and its Step 3 and Step 8 shapes grow a
line for an answered thread that now carries a marked reply. The skill's
`needs` line is not raised: an older binary leaves the field out, and Step 2
already tells the session to read the text, so nothing dead-ends
(`a-skill-can-need-a-release-nobody-cut.md`).

## Implementation Steps

### Task 1: A link to another note publishes as its words

**Files:**
- Modify: `go/internal/body/inline.go`
- Modify: `go/internal/body/body_test.go`
- Modify: `go/internal/body/doc.go`
- Delete: `docs/backlog/relative-link-to-another-note-is-dead.md`

- [x] write `TestARelativeLinkIsItsWordsAndNoHyperlink`: a note link and a
  folder link come out as their words, with no `w:hyperlink`, no `Media`
  entry and no warning. Watch it fail
- [x] write a guard case: `https:`, `mailto:` and `#` links are unchanged
- [x] in `inline.go`, leave `link` empty for a destination with no scheme
  and no leading `#`
- [x] state the rule in `body/doc.go`, naming the test
- [x] `make test`, `make vet`; commit with `git rm` of the item

### Task 2: Only a heading a link names carries a bookmark

**Files:**
- Modify: `go/internal/body/body.go`
- Modify: `go/internal/body/build.go` (only if the call site changes)
- Modify: `go/internal/body/anchors_test.go`
- Modify: `go/internal/body/doc.go`
- Modify: `go/internal/body/testdata/golden/*.xml` (regenerated)
- Delete: `docs/backlog/bookmark-only-linked-headings.md`

- [x] replace `TestEveryHeadingCarriesABookmark` with
  `TestOnlyALinkedHeadingCarriesABookmark`: a linked heading has the pair, an
  unlinked one has none, and ids count from 0 over the bookmarks written.
  Watch it fail
- [x] collect the `#` destinations in the pre-walk, and have
  `headingBookmark` return empty for a heading no link names
- [x] check the existing anchor tests still pass: the forward jump, the dead
  anchor warning, the figure-only heading and the footnote heading
- [x] regenerate the goldens with `-update` and read the diff: only bookmark
  pairs go away
- [x] rewrite the anchor section of `body/doc.go` and the
  `headingBookmark` comment, naming the new test
- [x] `make test`, `make vet`; commit with `git rm` of the item
- ➕ [x] DECISIONS.md entry and register row: the 2026-09-18 bookmark rule is
  superseded, and the note-link rule is recorded beside it
- ⚠️ `TestTheVersionReachesTheEnvelopeAndTheHelp` in `cmd/gdoc` fails on
  `origin/main` too: it reaches GitHub live and v2.5.0 is published. Not this
  plan's; flagged separately

### Task 3: A reply carries its marker

**Files:**
- Modify: `go/internal/comments/comments.go`
- Modify: `go/internal/comments/comments_test.go`
- Modify: `go/internal/comments/doc.go`
- Modify: `skills/gdoc-review/SKILL.md`
- Delete: `docs/backlog/a-marked-reply-carries-no-marker.md`

- [ ] write `TestAReplyCarriesItsMarker`: `ai!` reply gets `ai!`, a sentence
  that mentions `ai!` later gets `none`, and the JSON key is `marker`. Watch
  it fail
- [ ] add `Marker` to `Reply`, set from `markerOf` in `replies()`
- [ ] name the field in `comments/doc.go` beside `by_gdoc`, with the test
- [ ] skill: list `marker` in the Step 1 reply fields, and add the marked-reply
  line to the Step 3 and Step 8 report shapes
- [ ] `make test`, `make vet`; commit with `git rm` of the item

### Task 4: Verify acceptance criteria

- [ ] each of the three backlog items is gone and its fix is in the same commit
- [ ] `make test` (raced) and `make vet` pass
- [ ] `make build`, then publish nothing: `gdoc build` on a note with a
  relative link and a `#` link, and read `word/document.xml` in the output

### Task 5: [Final] Update documentation

- [ ] README and `docs/guide/` checked: neither describes bookmarks, note
  links or reply fields, so nothing changes
- [ ] move this plan to `docs/plans/completed/`

## Post-Completion

- A live publish of a hub note with a Related section, to see the plain names
  and no flags in Drive.
- Push and PR are Nail's call. After a merge into main, run the nightly per
  the standing rule.
