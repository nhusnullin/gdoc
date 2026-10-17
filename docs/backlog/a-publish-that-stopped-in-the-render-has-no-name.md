---
worth: soon
where: go/internal/presence/event.go:119
added: 2026-10-17
---
# A publish that stopped in the render has no name in the panel

The screens file draws "Publish stopped · Chapter 4" and "Publishing ·
Chapter 6". The panel cannot draw either, and the renders now say "Publish
stopped" and "Publishing" alone.

What the record holds about a publish is this:

- `presence.Begin` writes a start with no title at all. It runs before the
  command does, so there is nothing read yet to take a name from.
- `presence.Step` carries the title the render put on the cover, so the
  build step carries none and the three steps after it do.
- `Run.End` lifts the title out of the answer's own data, and `publishData`
  carries one only where the read-back named the document: that is the
  rule in `go/cmd/gdoc/publish.go`, so a title asked for is never printed
  as a title that exists.

So a publish that fails inside the render, which is scene r2's
`svg_picture`, wrote one step with no title and one end with none, and the
card naming it would be the panel inventing a fact. The fixtures used to
put a title on the start line, which made the renders show a name the
running app cannot draw; that is corrected and the renders are nameless.

Naming it needs gdoc to write the note's own title into the record. Two
shapes, both decisions:

- A new fact on the run: `rec.Named(title)` after `cover.Read` succeeds,
  kept on the `Run` and written on the end beside the read-back's title.
  The record's `title` then means "the name this line knows", which is a
  wider meaning than it has now, and the panel's own card would say it.
- The build step carries the title once the cover is read, which means a
  second step line for one part, or a step written after its part began.
  Both break what a step is: the line is written immediately before the
  part it names.

Nail's call, because it changes what a field in a file that lasts means.
Worth `soon` only because the mockups show the name and a reviewer reads
its absence as a bug: `docs/design/2026-10-17-presence-design-principles.md`
records that.
