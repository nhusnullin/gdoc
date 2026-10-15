---
worth: yes
where: go/internal/restyle/restyle.go
added: 2026-10-15
---
# An in-place restyle turns every pencil mark black

Measured on 2026-10-14 by `TestLiveMarkImportProbe`, written up in
`docs/v2/MEASURED.md`, "Drive's import of run shading and text colour". A run
sent as `w:color w:val="9A9A9A"` comes home from the import as foreground
#9A9A9A, and after one `gdoc restyle` over the same document Docs reports no
colour on that run at all.

What happens today: `restyle.TabRequests` writes one `updateTextStyle` with
`foregroundColor` from `styles.normal.color`, which house.yaml states as
#000000, over every body run. The run then matches the named style it
inherits from, so the grey is gone. It is the styling allowlist working as
written, because a text colour is not a character and `LevelInPlace` can
change no character. The six highlight colours are untouched: they are run
backgrounds, and nothing on the allowlist writes a background.

So a person who restyles a document that holds pencil gets ordinary black
prose where the grey was, and `export` then brings those words home without
the `[words]{.pencil}` span, because export recognises a mark by its exact
colour. Nothing says so: the restyle answer comes back `verified: true` and
`ManualSteps` names five things, none of them this one.

Nail chose that strain on 2026-10-15 rather than dropping the pencil from the
seven or stopping part B. `publish` and `export` are unaffected, and a
document nobody restyles keeps its pencil.

The fix: restyle splits its `foregroundColor` request around the runs that
carry the pencil hex, so the body goes black and the pencil stays grey. That
needs the run colours a read keeps, which M18's task 11 adds to
`internal/docs` for export's sake, so the data will be there. Two things to
decide with it:

- **The hex is the only handle.** A run is pencil because its foreground is
  #9A9A9A, the same rule export uses. A person who typed that grey by hand in
  the Doc is treated as having pencilled it, which is the rule the whole of
  part B rests on and not a new one.
- **A conditional `ManualStep` is the cheaper half.** Counting the pencil runs
  a restyle is about to flatten and saying so is one fact on the envelope, and
  it closes the silence without widening what the in-place grant writes to.
  That half could ship on its own.
