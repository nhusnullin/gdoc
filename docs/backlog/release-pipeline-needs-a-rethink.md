---
worth: yes
where: Makefile, .github/workflows/nightly.yml, .github/workflows/release.yml
added: 2026-09-28
---
# The release pipeline depends on memory, and a version bump cannot pass CI before its merge

Nail, 2026-09-28, after cutting v2.7.0: the release pipeline needs a rethink.
This item collects what went wrong around it, from the sessions of
2026-09-17 to 2026-09-28, so
the redesign starts from the incidents and not from a blank page.

## How it works today

- A skill reaches a colleague from `main` at once, through the plugin with
  auto-update on.
- The binary reaches them only as a GitHub release, and only when they run
  `gdoc update`.
- `make tag VERSION=vX.Y.0`, by hand, writes `.claude-plugin/plugin.json`,
  commits, tags and pushes. The Release workflow builds the zips.
- The nightly cuts `x.y.(z+1)` when `main` moved. It never cuts a minor.
  Nail runs it by hand after every merge, by a standing rule.
- `TestNoSkillNeedsAReleaseNobodyCut` fails when a skill's `needs` line is
  above the version in `plugin.json`.

## What went wrong

1. **A version bump cannot be green before its merge.** PR #76 raised two
   skills to `needs: v2.7.0`. The gate was red, correctly, because v2.7.0 did
   not exist. It could only turn green after the merge, when `make tag` ran on
   `main`. So a PR that raises a floor always merges red, and a real failure
   in the same run would hide behind the expected one.
2. **The obvious workaround switches the gate off.** Bumping `plugin.json`
   inside the PR makes CI green, but then the gate checks "someone typed the
   number", not "a release exists". Forgetting `make tag` would strand every
   colleague again, silently. `make tag` also fails on an already-bumped file,
   with nothing to commit.
3. **A test reached the real GitHub.** `TestTheVersionReachesTheEnvelopeAndTheHelp`
   ran `help` as a tagged build with an empty config dir, so it read the live
   releases listing. It broke on every branch when v2.6.0 shipped, and the
   2026-09-27 plan records the same failure on v2.5.0, "flagged separately"
   and not fixed. Fixed in #71; four other PRs each needed a cherry-pick of it.
   Nothing stops the next test from doing the same.
4. **Releases depend on remembered commands.** `make tag` for a minor, and
   `gh workflow run nightly.yml` after each merge, are both a person's memory.
   The skill-release mismatch of M13 (2026-09-20, `needs: v2.4.0` against a
   newest tag of v2.3.5) happened because the first was forgotten.
5. **Plugin auto-update is not on in practice.** CLAUDE.md and DECISIONS.md
   2026-09-18 say a skill reaches a colleague from `main` with auto-update on.
   On Nail's machine on 2026-09-28 the installed plugin was v2.4.0 while v2.6.0
   and v2.7.0 existed, and the marketplace clone was last pulled on 21 Sep. The
   hub's committed `.claude/settings.json` does not declare the marketplace,
   and the `gdoc` entry in `~/.claude/settings.json` has no `autoUpdate`. The
   same stale-skills symptom was asked about on 2026-09-21 ("why is my skill
   not updated automatically") and came back a week later.
6. **A feature waits for a hand-cut minor.** On 2026-09-18 a hub session could
   not leave a comment because `annotate` existed only in nightly v2.3.2.
   `make tag VERSION=v2.3.2` is refused, because only `x.y.0` is allowed, and
   the installer and `gdoc update` skip nightlies. So a finished command
   reaches colleagues only when somebody remembers a minor tag.
7. **The daily binary is no longer the checkout.** `~/.local/bin/gdoc` was
   meant to link to `bin/gdoc` so `make build` refreshes it. `gdoc update`
   replaced the link with a plain file on 21 Sep, so since then `make build`
   changes nothing Nail runs, and CLAUDE.md's "no reinstall" line is untrue on
   this machine.
8. **`gdoc update` does part of the job.** It leaves the shell completion
   stale (`update-leaves-the-completion-stale.md`) and prints nothing while it
   works (`update-shows-its-progress.md`).

Earlier incidents with the same shape: the first stable release (v2.0.0,
2026-09-17) shipped a `restyle --from` that refused its own `--dry-run` file,
because the round trip was never run before the tag; and the `-rc` tag failed
in CI because the parser accepted what the workflows refused.

## Questions the rethink has to answer

- Should the merge that raises a skill's floor cut the matching `x.y.0`
  itself, so the two roads meet by construction?
- Or should a skill stop naming a version, and ask the binary whether it has
  the commands and flags it uses?
- Should the nightly run on every push to `main` rather than by hand?
- How does auto-update actually get turned on, for Nail and for a colleague,
  and what proves it is on?
- What is the one intended way a checkout build reaches Nail's PATH, now that
  `gdoc update` replaces the link?
- How does CI refuse a test that reaches the network, the way the boundary
  tests refuse a second `http.Client`?

## Related

- The version paragraphs in each `skills/*/SKILL.md`: how a session tells a
  person to update, with the command alone in a `bash` block.
- `update-reads-one-page-of-releases.md`, `update-check-has-no-off-switch.md`,
  `windows-rollout-checklist.md`: the updater side of the same pipeline.
