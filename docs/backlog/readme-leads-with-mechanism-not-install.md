---
worth: yes
where: README.md:1
added: 2026-09-17
---
# The README explains the mechanism before it sells the tool or installs it

`README.md` is 842 lines. It opens with the credential model and the guard,
then spends 60 lines on the checkout install before a colleague without a
checkout finds the one-line installer. The three things a new user wants,
publish, restyle and live review, sit under `## What it does` at line 84 and
below, and each is described as a walk through the binary's output rather
than as a result the person gets.

That is the wrong document for the audience the repository now has. With the
plugin in `.claude-plugin/` and the installer in `release/`, the README is the
page a colleague lands on from a PR link, and it should get them working with
gdoc in minutes and make them want to.

## What the rewrite does

- **Title and first screen sell.** One paragraph on what gdoc does for you,
  then the three features as three short pitches: publish a note into Drive in
  the house style, restyle a document that is already there, review a
  document live from the terminal with every edit as a suggestion you accept
  or reject.
- **Install is one block, near the top.** The `curl | bash` line from
  `release/README.md`, then the plugin marketplace line, then `gdoc auth
  login`. Nothing about the checkout route, `make build` or the zsh completion
  before that block.
- **Technical detail moves out, with links left behind.** The credential and
  guard model, the checkout install, the per-command walkthroughs, the
  `gdoc:` block, and the escaping rules go to a documentation page (a
  `docs/` file or a section the README links to), each one linked from the
  spot the README mentions it. `release/README.md` already carries the
  short install; decide whether the top-level README absorbs it or links to
  it so the two stop drifting.
- **The planned section goes.** `## What is planned` at line 833 points at
  the plan; a link is enough.

`TestCLAUDEmdIsUnderTheCeiling` guards `CLAUDE.md`, not the README, so nothing
pins the README's length today. Adding a ceiling test for it is part of this
item if the rewrite wants one.
