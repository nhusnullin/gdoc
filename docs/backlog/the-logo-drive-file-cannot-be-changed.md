---
worth: yes
where: go/internal/house/house.yaml:1163
added: 2026-10-21
---
# The logo's Drive file cannot be changed without a new binary

The house logo comes from one Drive file, named by id in `logo.drive` in
`house.yaml`. That file is embedded in the binary, so pointing gdoc at another
logo, or at another folder of assets, means a commit and a release. Nail asked
for a way to change where gdoc reads its assets from.

Today there is no folder at all: the reach is one file id, and
`guard.AllowAsset` admits one media read of that one id per run. `settings.yaml`
holds the mark and `presence` and nothing about assets.

What the fix has to decide first:

- A setting that names a file id, or one that names a folder. A folder means
  listing it, which is a Drive call the guard does not admit today and a new
  door into the reachable set. A file id keeps the one-id grant as it is.
- Whether the setting overrides `logo.drive` or replaces it, and what happens
  to the cache under `assets/<id>` when the id changes. The cache is keyed by
  id, so an old entry would sit there unused.
- The stated mime and pixels in `house.yaml` still hold the file to its size,
  so a new logo of another size is refused until that changes too.

Either way it is a `docs/v2/DECISIONS.md` entry before code, because the M22
decision made the firm's file in the firm's Drive the one source.
