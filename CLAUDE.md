# Notes for AI assistants

Read [PRINCIPLES.md](PRINCIPLES.md) before proposing any design. It is three
constraints and the decisions that currently implement them, and it is short.

Read [README.md](README.md) for what the tool does and how to run it.

This file holds the specifics: what lives where, what each rule means in code,
and what never to do. The reasons live in PRINCIPLES.md, and they live there
only.

## What lives where

| Path | Holds |
|---|---|
| `gdoc/` | the package. Imports are `from gdoc.x import y` |
| `gdoc/guard.py` | the reachable set. Which files a client may touch, under either credential |
| `gdoc/marker.py` | the `[gdoc]` label on gdoc's own replies |
| `tests/` | pytest suite. Every module has a matching test file |
| `skills/` | `gdoc-review` and `gdoc-apply`. Symlinked into `~/.claude/skills/`, so edits are live |
| `docs/superpowers/` | the implementation plans and the design specs |
| `.gdoc/<slug>/` | `pending.md`, `baseline.md`, generated `out/*.docx`. **Not in this repo:** it sits beside the source markdown being reviewed |

Secrets and the venv live in `~/.config/gdoc-agent/`, never in this repo.

## One root, and it is never this repo

Principle 2. Read it there.

In code: `--repo-root` is the tree scanned for markdown with `gdoc:` frontmatter,
and the tree `.gdoc/` is written into. In practice it is the folder holding the
source document, in a vault such as `Altery-Platform-Hub`.

## Nothing runs git

Principles 1 and 3, and the decision dated 2026-08-18. This is the constraint an
agent is most likely to break, usually by adding a helpful commit.

- No module and no skill runs git. Not to commit, not to check whether the root
  is a repository, not to ask whether a file is dirty. `gdoc-apply` names every
  file it changed and stops there.
- `write_baseline` refuses to overwrite an existing `baseline.md` unless `force`
  says so. It asks nothing and nobody: not knowing must never resolve to
  "overwrite", and in a synced vault git could not have answered anyway.
- `generate` is the only caller that writes a baseline, and it passes `force` at
  the one moment the document and the markdown provably match.
- `install.sh` still reads git, and that is about this repository rather than
  about the user's documents.

## No external programs

Principle 1. The rule is scoped to `gdoc/render/`, the publish path.

pandoc remains, in three places, and each is tracked separately:

- `gdoc/render/body.py`, as the markdown parser `build` depends on. Load-bearing.
  Removing it means a pure-Python parser plus a rewritten AST walker, in the
  module where the document body's pixel fidelity lives, so it needs its own
  spec.
- `gdoc/export.py`, as a markdown fallback. Drive exports `text/markdown`
  natively, verified against the live account, so this one looks removable on
  its own.
- `gdoc/generate.py`, for the plain non-template path, which is what
  `--template none` selects. Kept on purpose: it is the fallback when the house
  template is not wanted, and keeping it narrowed the blast radius of wiring the
  template in.

`tests/test_no_external_programs.py` enforces this. The five removed program
names must not appear in code under `gdoc/render/`, and the set of modules
there importing `subprocess` must be exactly `{body.py}`. That is an allowlist,
not a ban: it fails if `subprocess` spreads to another module, and it fails just
as loudly if body.py's own dependency vanishes without the test being updated.

Page numbers are the one real cost. They do not exist until something lays the
document out. The intended flow makes Google the layout engine: upload once with
blank numbers, read which page each heading landed on out of the PDF export, write
those numbers in, then upload the version that gets published.

`gdoc generate` runs that flow. It builds with blank page numbers, uploads that
copy, exports it as PDF, reads the pages back, builds again with the numbers in,
uploads the version that gets published, and trashes the measuring copy. The
published copy is measured too, so a contents list that disagrees with its own
document comes back in the result as `drift` rather than passing quietly.

The template comes from `--template`, or from `template` in the config, which
defaults to the bundled profile. `--template none` keeps the plain
`pandoc md -o docx` path.

`tests/test_contents_integration.py` still composes the two passes against live
Drive, and is opt-in behind `GDOC_LIVE_PUBLISH_TEST=1`. It is the only test that
creates real documents.

`gdoc.render.build` on its own has no pagination to offer, so it leaves the page
numbers blank. Any desktop refresh fills them in, and a wrong number would be
worse than a blank one.

## The client reaches only the files it was given

Principle 3, and the decision dated 2026-08-15. Read it there.

In code: `drive_service(doc_ids=...)` wraps the transport in
`gdoc.guard.GuardedHttp`, which carries a request only when the file it
addresses is in its set. The set is seeded from the CLI, where `read`, `reply`,
`export` and `capture` each pass the document they were pointed at and
`generate` passes the output folder, and it grows only when a create the guard
itself carried comes back with an id.

Two rules an agent is likely to break:

- The set has exactly two doors, the ids passed in and the ids learned from a
  create. Never add a third, and never widen it to make a test pass. A refused
  call usually means the command did not say which document it was for.
- `gdoc/auth.py` is the only module that may call `build()`.
  `tests/test_guard_is_installed.py` enforces it, as an allowlist: it fails if
  the call spreads, and it fails if it moves.

## Which credential, and who decides

`gdoc.auth.resolve_auth_mode` is the one place. It is a pure function of the mode
the config states plus which credential files exist, so it never reads the config
itself and a test can hand it either case.

- A stated mode wins outright. Nothing on disk may overrule the author's word,
  and a mode it does not recognise is refused rather than inferred past. A typo
  must never resolve to oauth, which is the credential with the wider reach.
- An unstated mode is inferred: a token means oauth, a key with no token means
  service_account, neither means oauth.

`auth._config()` returns the defaults for a **missing** config only. A config that
exists and cannot be understood reaches the caller, so every command fails naming
the problem. `gdoc auth status` is the one place that catches it, reports
`auth_mode: null` with source `unknown`, and never guesses.

`Config.auth_mode` is `None` when the file does not say, and that is the point.
Defaulting it to oauth in `load_config` would flip every working service_account
install the moment it upgraded, because a config written before the key existed
cannot mention it. Never give that field a default.

Two consequences an agent is likely to break:

- `gdoc auth login` writes `auth_mode: oauth` through `config.write_auth_mode`,
  after the browser flow returns and **before** the account lookup. Later would
  mean a network failure leaves the person signed in with the old credential
  configured. `--token` skips the write entirely and says so, because nothing
  else reads a custom token path. `gdoc auth use <mode>` is the same write, in
  either direction, so no setup step is ever a hand edit of JSON.
- `write_auth_mode` carries every other key over, `display_name` included, and
  refuses a file it could not parse rather than replacing it. It writes through a
  temporary file and `os.replace`, so a failed write cannot leave an empty config.
  Truncating first lost settings that were readable a moment earlier, and the
  empty file then blocked the next write too.

`tests/conftest.py` redirects the config and both credential paths into a tmp
directory for every non-integration test, autouse. Without it a test that runs
`auth login` edits the developer's real config, and resolving an unstated mode
would answer differently per machine. Integration tests are exempt, because the
real credential is their point.

## The OAuth client is shipped, and stays Internal

`gdoc/oauth.py` holds `BUNDLED_CLIENT_ID` and `BUNDLED_CLIENT_SECRET`, so nobody
using gdoc visits a cloud console. Both rules matter:

- **The secret belongs in version control.** RFC 8252 section 8.5: a secret
  shipped to many users "should not be treated as confidential" and serves no
  purpose "beyond client identification". `gh` ships its own with the comment
  "This value is safe to be embedded in version control", and `gcloud` ships a
  Google secret in a constant named `CLOUDSDK_CLIENT_NOTSOSECRET`. Do not
  "fix" this by moving it to `~/.config/gdoc-agent/`. What protects an account
  is the per-user token, which never leaves the machine.
- **The client must stay User type Internal.** gdoc needs the full Drive scope,
  which Google classes as restricted. Internal exempts gdoc from verification,
  the unverified-app screen and the 100-user cap. External would mean a CASA
  assessment every 12 months, and refresh tokens expiring weekly.

A file at `~/.config/gdoc-agent/oauth-client.json` wins over the bundled client.
That is an override for quota, not a setup step, the way gcloud treats
`--client-id-file`. One shared client shares one Google rate limit, which is why
rclone is retiring its shared Drive client during 2026.

`oauth.client_config` is the one place that decides, and it returns where the
client came from so `gdoc auth status` can report it.

### The one thing that changes this decision: making the repo public

Nail decided on 2026-08-18 to keep the client in git, on the RFC and on the `gh`
and `gcloud` precedent. That decision assumed a private repo, and one condition
would break it.

GitHub secret scanning carries a **partner** pattern for
`google_oauth_client_id, google_oauth_client_secret`. On a public repository it is
reported to Google, who may revoke the client. So publishing this repo would
break every colleague's login at once, without warning and without a commit to
blame. rclone obfuscates its Google secret for exactly this reason, which is
evasion of automated revocation rather than security.

So before this repo is ever made public: create a fresh client, distribute it as
a file out of band, and clear these two constants. Do not obfuscate them to get
past the scanner.

## Identity is never a gate

Drive's `author.me` means the service account under `auth_mode: service_account`
and Nail under `oauth`. Nothing may branch on it to decide whether a comment is
work: that would skip every comment Nail writes. The marker decides.

`[gdoc]` on the last line of a reply is how gdoc recognises its own replies.
`Reply.by_agent`, which is `me`, is kept in `has_agent_reply` for one reason
only: threads the service account answered before the marker existed carry no
marker, and dropping it would answer them twice.

## Skills are linked, not copied

`~/.claude/skills/gdoc-review` and `gdoc-apply` are symlinks into `skills/` in
this repo. There is one copy of each SKILL.md, so editing it here changes what
Claude Code loads. No copy step, and no way for the skill to drift from the CLI
it calls.

Two consequences worth holding:

- An edit is live the moment it is saved, before it is committed. Nothing warns
  you. `./install.sh` prints `+ uncommitted changes` when the tree is dirty.
- Deleting or moving `skills/` breaks the installed skills. Re-run
  `./install.sh` after any move.

`install.sh` is safe to re-run. It refuses to replace a real
`~/.claude/skills/<name>` directory whose contents differ from this repo, so an
older copy-based install cannot be destroyed silently.

## Testing

TDD. Write the failing test first.

```bash
~/.config/gdoc-agent/venv/bin/pytest
```

`tests/test_access_integration.py` calls Drive and skips without credentials.
Never make a test pass by loosening an assertion about what the credential can do.

### A house-style test must never read the constant it tests

`assert style.spacing == HEADING_LINE_SPACING` is worth nothing. It is a mirror:
set the constant back to the master's single spacing and the assertion follows it
and still passes. Write the house value as a literal instead, so the test says
what the house style is. `tests/test_render_shell.py` does this on purpose, and
where a value has a safe range rather than one right answer it asserts the floor
instead, as with the 40-twip cell margin.

## Never

- Never edit a reviewed Google Doc. Under `service_account` the credential
  cannot. Under `oauth` it could: `gdoc/guard.py` bounds which files are
  reachable, not what may be done inside one. Nothing in gdoc edits a document,
  and nothing may start.
- Never commit anything from `~/.config/gdoc-agent/`.
- Never post markdown into a comment thread. The CLI refuses it for a reason.

## Writing style

Plain, short English. No em dashes.
