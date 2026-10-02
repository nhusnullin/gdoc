---
worth: later
added: 2026-10-03
---
# A full Claude Desktop extension for colleagues without a terminal

M14 installs gdoc for Claude Desktop through `install.sh --desktop`: a thin
`gdoc.mcpb` whose command points at the one binary in `~/.local/bin`, so the
terminal, Claude Code and chat run the same file and `gdoc update` updates
all three. Nail chose that on 2026-10-03 because every current colleague has
a terminal and a technical background.

A colleague without a terminal would need a full bundle that carries its own
binary, downloaded from the release and opened in Claude Desktop. The M14
review designed it:

- one universal Mac binary, made by a fat Mach-O writer of about 60 lines of
  Go using `debug/macho`, run by the release job on Ubuntu, with the arm64
  slice aligned at 2^14 and Go's ad-hoc signature left inside its slice;
- the asset named `gdoc.mcpb`, a fixed name, so a permanent
  `releases/latest/download/gdoc.mcpb` link works and `SHA256SUMS` still
  covers it;
- `gdoc update` refusing to run from inside the bundle, because Claude
  Desktop manages that file; the bundle updates by opening a newer one;
- the update line in chat naming the download link instead of `gdoc update`;
- the quarantine question, since no installer strips the mark: allow once in
  System Settings, Privacy & Security, or sign the binary with an Apple
  Developer ID.

The cost is two copies of gdoc updated two ways for anyone who uses both
chat and the terminal, which is why it was not built.

Nail's idea of the same day belongs here too: `gdoc` packing itself as an
extension, so the manifest's tool list comes from the binary that answers
`tools/list`. It cannot make the universal binary from one chip's build, so
it serves a person making a bundle for their own Mac, not the release.

What settles the value decision, and why it is `later`: a colleague who
reviews documents and cannot or will not use a terminal.
