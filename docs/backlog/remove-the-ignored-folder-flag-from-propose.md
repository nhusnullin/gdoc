---
worth: yes
where: go/cmd/gdoc/commands.go:252
added: 2026-10-03
---
# Remove the ignored --folder flag from propose

M14 took the capability probe out of `propose`, so its `--folder` no longer
does anything. To keep M14 a minor release, v2.8.0, the flag stays one
release: accepted, ignored, with a warning that it will be removed. Nothing
a v2.7 caller sends breaks, so plain `gdoc update` takes the release without
`--major`.

Remove it in a later release, once the skills that passed it, `gdoc-review`'s
`propose.md` and `gdoc-align`, have shipped without it and colleagues have
updated their plugin. Removing a flag a caller may still send is a breaking
change, so it goes either in a major release or after a release whose notes
said it was coming. The table entry, the usage line in `help_test.go`, and
the warning test go with it.
