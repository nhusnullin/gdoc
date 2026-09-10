---
worth: yes
added: 2026-09-10
---
# GrantInPlace takes capability away from an id that is already at LevelFull

`Policy.GrantInPlace` writes `LevelInPlace` over whatever level the id already carried, and every
level comparison in the guard is `==` on purpose, so the levels are names and not a ladder. Handed an
id that `Learn` put in the set at `LevelFull`, from a create the guard itself carried, the grant
therefore *removes* reach:

- `judgeDrive` matches `method == "PATCH" && shape == "file" && lvl == LevelFull`, so `drive.Trash`
  on that document stops being carried.
- `judgeRequests` starts enforcing `inPlaceKinds`, so an `insertText` that was legal one line earlier
  is refused.

Nothing warns. `AllowFile`, immediately above it, refuses the analogous mistake and records the
reason through `p.note`, which is the shape this one is missing.

No production caller reaches it today: `cmdRestyle` grants an id it handed in at `LevelSuggest`, and
that is the only caller. It is already costing something in the tests, though.
`internal/live/restyle_test.go` has to run the copy's trash on the first policy and the restyle on a
second one, with a comment explaining why, and any later flow that creates a document, styles it in
place and then trashes it fails on the cleanup step of a run that has already created a file. The
message names the level and not the line that caused the demotion.

Two ways out, and the choice is Nail's, because it is guard semantics:

- Refuse and `p.note` when the current level is not `LevelSuggest`, which is `AllowFile`'s own rule
  one function up: the grant is for a handed-in document, and an id at `LevelFull` was never one.
- Keep the higher level and note that the grant was unnecessary. This is the friendlier direction and
  it makes the levels partly a ladder, which the constant block's own comment says they are not.

Either way the caller learns which line changed the reach. Found in the M7b external review,
2026-09-10.
