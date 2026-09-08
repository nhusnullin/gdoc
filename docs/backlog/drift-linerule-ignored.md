---
worth: yes
where: go/internal/drift/docx.go:412
added: 2026-09-08
---
# `drift` reads `w:line` without `w:lineRule`, so an exact line height reads as a percentage

`applyPara` turns `w:line` into a percentage unconditionally,
`st.LineSpacing = round3(n / 240 * 100)`. That is only what the attribute means
under `w:lineRule="auto"`. Under `exact` or `atLeast` the same number is twips,
so a paragraph set to an 18pt line reads as 150% spacing.

Measured by the M5 review on 2026-09-08: `lineRule="exact" w:line="360"` reads
`lineSpacing 150`, and 360 twips is 18pt. `line="276" lineRule="auto"` and
`line="276" lineRule="exact"` both read 115.0 and compare IDENTICAL.

Latent today, and the reason is worth writing down rather than trusting: every
`w:lineRule` in the master is `auto` (49 in `styles.xml`, 185 in
`document.xml`), and everything `internal/render` writes is `auto` too
(`render/xml.go`, `render/styles.go`). So the two documents cannot disagree
about the rule yet. `lineSpacing` is also the one item carrying `exactTol`, a
tolerance of zero, so the day they do disagree the row reports a difference of
the wrong kind or hides one.

The fix is to read `w:lineRule` and report `nil` for anything but `auto`, so the
row goes MISSING rather than carrying a number in the wrong unit. Converting an
exact height to points instead would need its own item name, because a
percentage and a length are not one measurement and `Doc` has no counterpart to
read it from.
