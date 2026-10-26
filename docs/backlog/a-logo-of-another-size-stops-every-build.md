---
worth: now
where: go/internal/house/house.yaml:1148
added: 2026-10-23
---
# A logo of another size stops every build

Nail replaced the house logo in Drive with a bigger picture. Since then no note
can be built or published. Measured on 2026-10-23 with v2.19.0:

```
gdoc build --md release/example/first-note.md --out example.docx
{"ok":false,"error":"file 1J3uh6poVAneP_V_Pc5FuNI4-Gahlab0s in Drive is not the
file it is stated to be: it is stated as 195x97 px and the image is 509x185"}
```

The cache under `assets/<id>` already holds the new 509x185 bytes, so there is
no older copy to fall back on.

The cause is the pin, not the size. `house.yaml` states `px: [195, 97]`, and
`internal/asset` and `house.Validate` refuse any image of other pixels. The
size the page shows is set apart from that, by `width_pt: 146.25` and
`height_pt: 72.75`. A docx extent is in points, so bigger pixels would print in
the same box and need no change at all.

The ratio does change. The old picture was 2.01 to 1, and the new one is 2.75
to 1. Drawn into the old box as it stands, it would be squashed. Nail fixes this
by hand in Google Docs by dragging a corner, which keeps the ratio.

What the fix has to decide:

- What `px` means after the change: an exact pin, a ceiling, or gone. The pin
  exists so a wrong file is refused rather than printed, and the mime and the
  byte ceiling still do part of that job.
- How the picture fits the box: keep the width and let the height follow
  (146.25 x 53.2 pt for this logo), or fit inside both. Either way the anchor
  offsets in `logo.anchor` stay as they are.
- Whether the Docs-side cover in `internal/prelude` needs the same rule, since
  `restyle` proposes the cover there.

The pin was a decision of M22 (`docs/v2/DECISIONS.md`), so this is an entry
before code. It sits next to `the-logo-drive-file-cannot-be-changed.md`, whose
last point named this very refusal.

Until the fix ships, the one thing that unblocks builds is the old 195x97
picture back in that Drive file, from the file's version history.
