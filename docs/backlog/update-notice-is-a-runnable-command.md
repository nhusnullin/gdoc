---
worth: yes
where: skills/gdoc-review/SKILL.md:44
added: 2026-09-28
---
# A skill names `gdoc update` as a runnable command, not inside a sentence

When `help` says a newer gdoc is published, each of the five skills tells the
session to "name what installs it". Sessions do that inside prose: "Also,
gdoc v2.6.0 is out and you are on v2.4.0 (`gdoc update`)." Nail, 2026-09-28:
that is text to read and retype. The command should arrive as its own row
the person can run with one click, like the fenced `bash` block the desktop
app gives a Run button.

The fix is words in the same paragraph of all five `SKILL.md` files (review,
publish, restyle, export, align), and in the matching `needs` paragraph that
stops a session on an older binary:

- One short sentence says what is published and what is installed.
- The command follows in its own fenced block tagged `bash`, one command,
  no `$` prompt: `gdoc update`, or `gdoc update --major` for a major.
- Nothing else goes in that block, so the Run button runs exactly the
  install.

The notice `help` prints on stderr in `go/cmd/gdoc/notice.go:181` is for a
terminal and stays as it is. A test in `go/boundary/` that reads the five
skills and finds the fenced block would stop one skill drifting from the
others, the way the `needs` line is already checked across them.

Related: `gdoc update` draws its steps on stderr since 2026-09-28
(`go/cmd/gdoc/progress.go`). A session that runs it itself gets the plain-line
form, one line per finished step.
