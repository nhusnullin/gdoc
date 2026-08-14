# gdoc storage and iteration, Part A

> **For agentic workers:** REQUIRED SUB-SKILL: use superpowers:subagent-driven-development
> or superpowers:executing-plans to work this plan task by task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Detach the tool from any named repository, move its files next to the
work they belong to, and put a baseline in place so a later apply can see what
changed in the document.

**Spec:** [`docs/superpowers/specs/2026-08-14-gdoc-storage-and-iteration-design.md`](../specs/2026-08-14-gdoc-storage-and-iteration-design.md)

**Scope:** Part A only. Part B, merging suggestions, is unblocked but not planned
here. The spike ran on 2026-08-14 and settled how they are read; how they are
merged is a separate plan.

## Global constraints

Copied from the spec. Every task's requirements include these.

- **The tool must work without git.** `Altery-Platform-Hub`, where the source
  documents live, is not a git repository and will not become one. Nothing may
  refuse to run because git is unavailable.
- **No path in the package or the skills names a specific repository.** Not
  `intelligence-hub`, not `~/src/personal/gdoc`. The root is `$PWD`.
- **Tool-managed files go under `.gdoc/`,** beside the source markdown.
- **Immutability:** frozen dataclasses, new objects, never mutate in place.
- **Tests first.** Write the failing test, watch it fail, then implement.
- **Coverage stays at 80% minimum** on the package.
- **Plain English in every user-facing string.** Short sentences, no em dashes.

**Run the suite with:** `~/.config/gdoc-agent/venv/bin/pytest`

**Do not commit anything from `~/.config/gdoc-agent/`.** It holds the service
account key.

## What exists now

| File | Current state |
|---|---|
| `gdoc/mirror.py` | `MIRROR_DIR = Path("docs") / "gdoc"`, `mirror_path`, `write_mirror`, `is_dirty`, `MirrorConflict`, `slugify` |
| `gdoc/pending.py` | imports `MIRROR_DIR` from `gdoc.mirror`; `pending_path` |
| `gdoc/cli.py` | `cmd_export` writes the mirror; `_check_slug_collision`; `--repo-root` defaults to `"."` on `export` and `capture`, and is `required=True` on `pair find` |
| `gdoc/generate.py` | `generate()` converts and uploads. Writes no baseline |
| `skills/*/SKILL.md` | both set `GDOC_REPO="$HOME/src/personal/gdoc"` and pass it as `--repo-root` |

---

### Task 1: Rename the mirror to a baseline

The file is kept, but its name lies about its job. It exists to be diffed
against, not to mirror. Renaming first means every later task works in the final
names.

**Files:**
- Rename: `gdoc/mirror.py` to `gdoc/baseline.py`
- Rename: `tests/test_mirror.py` to `tests/test_baseline.py`
- Modify: `gdoc/pending.py`, `gdoc/cli.py`, `tests/test_cli.py`

**Interfaces:**
- Produces: `write_baseline`, `baseline_path`, `BaselineConflict`, unchanged
  `slugify` and `is_dirty`. The written filename becomes `baseline.md`.
- Consumes: nothing new.

- [ ] **Step 1: Update the tests first**

`git mv tests/test_mirror.py tests/test_baseline.py`, then replace the names.
Assert on the filename explicitly, because that is the part a rename can silently
miss:

```python
def test_the_written_file_is_named_baseline(tmp_path):
    path = write_baseline(tmp_path, "my-doc", "# Title\n")
    assert path.name == "baseline.md"
```

- [ ] **Step 2: Run and confirm failure**

Expected: `ModuleNotFoundError: No module named 'gdoc.baseline'`

- [ ] **Step 3: Rename and update**

`git mv gdoc/mirror.py gdoc/baseline.py`. Inside it, `mirror_path` becomes
`baseline_path`, `write_mirror` becomes `write_baseline`, `MirrorConflict`
becomes `BaselineConflict`, and the filename literal becomes `"baseline.md"`.
Update the module docstring: it writes a snapshot to compare against, it does not
mirror.

Fix the importers: `gdoc/pending.py` and `gdoc/cli.py`.

- [ ] **Step 4: Prove nothing still refers to the old names**

```bash
grep -rn "mirror\|Mirror" gdoc tests skills
```

Expected: no matches, or only prose that is genuinely about the old design.

- [ ] **Step 5: Run the suite**

Expected: everything that passed before still passes.

- [ ] **Step 6: Commit**

```bash
git commit -am "refactor: rename the mirror to a baseline

It exists to be diffed against a later export, not to mirror the document.
The name drove the wrong write timing, which the next tasks fix."
```

---

### Task 2: Move the tool's files to `.gdoc/`, rooted at `$PWD`

**Files:**
- Modify: `gdoc/baseline.py`, `gdoc/cli.py`
- Modify: `tests/test_baseline.py`, `tests/test_pending.py`, `tests/test_cli.py`

**Interfaces:**
- Produces: `GDOC_DIR = Path(".gdoc")`, replacing `MIRROR_DIR`. `--repo-root`
  defaults to `"."` on every command that takes it, `pair find` included.

- [ ] **Step 1: Write the failing tests**

In `tests/test_baseline.py`:

```python
def test_files_go_in_a_dot_gdoc_directory(tmp_path):
    path = baseline_path(tmp_path, "my-doc")
    assert path == tmp_path / ".gdoc" / "my-doc" / "baseline.md"
```

In `tests/test_cli.py`, assert the default rather than the passed value, since
the default is the whole point:

```python
def test_pair_find_defaults_to_the_working_directory():
    args = build_parser().parse_args(["pair", "find", "--doc-id", "1AbC"])
    assert args.repo_root == "."
```

- [ ] **Step 2: Run and confirm failure**

Expected: the path test fails on `docs/gdoc`, and `pair find` fails with
`error: the following arguments are required: --repo-root`.

- [ ] **Step 3: Implement**

In `gdoc/baseline.py`, `MIRROR_DIR = Path("docs") / "gdoc"` becomes
`GDOC_DIR = Path(".gdoc")`. Update the import in `gdoc/pending.py`.

In `gdoc/cli.py`, drop `required=True` from `pair find`'s `--repo-root` and give
it `default="."`.

Why a dot directory: once the root can be any directory, `docs/` cannot be
assumed free or appropriate, and a dot-namespace signals tool-owned. It also
keeps these files out of Obsidian's search and graph.

- [ ] **Step 4: Run the suite, then commit**

```bash
git commit -am "refactor: keep tool files in .gdoc/ under the working directory

The root is now \$PWD everywhere, so nothing points at a named repo."
```

---

### Task 3: Work without git

`is_dirty` raises `BaselineConflict` when git cannot be consulted. That is right
when git exists and refuses to answer. It is wrong when there is no git at all,
which is now the normal case: the hub holding the source documents is not a
repository.

**Files:**
- Modify: `gdoc/baseline.py`
- Modify: `tests/test_baseline.py`

**Interfaces:**
- Produces: `has_git(path: Path) -> bool`. `write_baseline` keeps its signature.

- [ ] **Step 1: Write the failing tests**

`tmp_path` is outside any repository, which is exactly the case to pin:

```python
def test_writes_a_new_file_outside_a_git_repo(tmp_path):
    path = write_baseline(tmp_path, "my-doc", "# Title\n")
    assert path.read_text() == "# Title\n"


def test_refuses_to_overwrite_outside_a_git_repo(tmp_path):
    write_baseline(tmp_path, "my-doc", "# First\n")
    with pytest.raises(BaselineConflict, match="force"):
        write_baseline(tmp_path, "my-doc", "# Second\n")


def test_force_overwrites_outside_a_git_repo(tmp_path):
    write_baseline(tmp_path, "my-doc", "# First\n")
    path = write_baseline(tmp_path, "my-doc", "# Second\n", force=True)
    assert path.read_text() == "# Second\n"
```

The existing in-repo tests must keep passing unchanged. Where git IS available,
behaviour does not change: a clean tracked file is still overwritten without
`force`.

- [ ] **Step 2: Run and confirm failure**

Expected: the first test fails with `BaselineConflict: git could not be
consulted`.

- [ ] **Step 3: Implement**

Add `has_git(path)`, returning whether `git rev-parse --git-dir` succeeds in that
directory. In `write_baseline`, when the file already exists and `force` is
false:

- git available: keep today's behaviour, refuse when `is_dirty`.
- no git: refuse. Without git there is no way to tell an edit from a stale copy,
  and not knowing must never resolve to "overwrite".

Keep `BaselineConflict` raised by `is_dirty` for the case where git exists but
the command fails. That is a real fault and must stay loud.

The error message must name `--force` and say plainly that git is unavailable, so
the reader knows the refusal is not about uncommitted work.

- [ ] **Step 4: Run the suite, then commit**

```bash
git commit -am "feat: work without git

The hub holding the source documents is not a repository. Without git we cannot
tell an edit from a stale copy, so an existing file is never overwritten unless
--force says so."
```

---

### Task 4: `export` fetches, it does not file

`export` downloads, derives a slug, checks collisions, refuses on uncommitted
edits, and writes to a path the caller never chose. Split the fetch from the
filing so a drift check is possible.

**Files:**
- Modify: `gdoc/cli.py`
- Modify: `tests/test_cli.py`

**Interfaces:**
- Produces: `gdoc export <url>` prints markdown to stdout;
  `gdoc export <url> --out FILE` writes it. `--repo-root`, `--slug` and
  `--force` are removed from this subcommand.

- [ ] **Step 1: Write the failing tests**

```python
def test_export_prints_markdown_to_stdout(capsys, ...):
    ...
    assert capsys.readouterr().out == "# Title\n\nBody.\n"


def test_export_writes_to_the_named_file(tmp_path, ...):
    out = tmp_path / "fetched.md"
    main(["export", URL, "--out", str(out)])
    assert out.read_text() == "# Title\n\nBody.\n"


def test_export_writes_nothing_when_out_is_absent(tmp_path, ...):
    main(["export", URL])
    assert list(tmp_path.iterdir()) == []
```

Note that stdout is now raw markdown, not the JSON envelope every other command
emits. That is deliberate: the output is meant to be piped into `diff`. Assert
it, so nobody "fixes" it back into JSON later.

- [ ] **Step 2: Run and confirm failure**

- [ ] **Step 3: Implement**

Rewrite `cmd_export` to fetch, then print or write. Delete `_check_slug_collision`
and its tests: it guards two documents mapping to one slug, and Task 6 keys
directories by document id, which removes the collision entirely.

`export` no longer touches `write_baseline`. Nothing is written unless asked.

- [ ] **Step 4: Confirm the drift check works**

```bash
gdoc export <url> | diff - path/to/source.md
```

Expect a real diff, dominated by round-trip formatting noise. That noise is the
reason Task 5 exists: only two exports of the same document cancel it.

- [ ] **Step 5: Run the suite, then commit**

```bash
git commit -am "refactor: export fetches markdown instead of filing it

Prints to stdout, or writes to --out. Nothing is written unless asked, which
makes an on-demand drift check possible."
```

---

### Task 5: `generate` writes the baseline

The snapshot has to be taken when the document and the markdown provably match.
That moment is right after a successful upload, and nowhere else.

**Files:**
- Modify: `gdoc/cli.py`
- Modify: `tests/test_cli.py`

**Interfaces:**
- Produces: the `generate` subcommand gains `--baseline-root DIR`. When it is
  given and the upload succeeds, the new document is exported and written to
  `<root>/.gdoc/<slug>/baseline.md` with `force=True`.
- The `generate()` function in `gdoc/generate.py` is unchanged. It stays a
  converter and uploader; the baseline is a CLI concern, which keeps the seam
  testable.

- [ ] **Step 1: Write the failing tests**

```python
def test_generate_writes_the_baseline_after_a_successful_upload(tmp_path, ...):
    main(["generate", "--md", str(md), "--name", "My Doc",
          "--out", str(out), "--baseline-root", str(tmp_path)])
    assert (tmp_path / ".gdoc" / "my-doc" / "baseline.md").exists()


def test_generate_writes_no_baseline_when_the_upload_failed(tmp_path, ...):
    # folder_id absent, so generate() returns a Result with doc_id None
    main(["generate", "--md", str(md), "--name", "My Doc",
          "--out", str(out), "--baseline-root", str(tmp_path)])
    assert not (tmp_path / ".gdoc").exists()


def test_generate_overwrites_an_existing_baseline(tmp_path, ...):
    # two runs; the second must not raise BaselineConflict
```

The failure case matters most. A baseline written for a document that was never
created would make the next apply diff against a document that does not exist.

- [ ] **Step 2: Run and confirm failure**

- [ ] **Step 3: Implement**

In `cmd_generate`, after `generate()` returns: if `--baseline-root` was given and
`result.doc_id` is set, call `export_markdown(drive, result.doc_id)` and
`write_baseline(root, slug, markdown, force=True)`.

`force=True` is correct here and only here: this write is meant to replace the
previous baseline, and refusing would break the loop on the second version.

Include the baseline path in the JSON output, so the skill can report it.

- [ ] **Step 4: Run the suite, then commit**

```bash
git commit -am "feat: write the baseline after a successful upload

Taken at the one moment the document and the markdown provably match. A later
apply diffs this against a fresh export to see direct edits, and the two exports
cancel the pandoc round-trip noise."
```

---

### Task 6: Key directories by the source markdown file

`slug = slugify(document title)`, so the directory tracks a mutable attribute
and each new version forks the queue. That is not an edge case, it is the normal
pattern: every iteration raises a new version.

The document id is not the anchor either. `generate` calls `files.create`, so
each version is a new Google Doc with a new id. Verified on 2026-08-14: the
`ver-0-1` queue holds `1E8-xKq...` and `ver-0-2` holds `1zIyYZO...`, two
different documents.

The anchor is the source markdown file. It survives every iteration, and it
already carries the lineage in `gdoc:` plus `gdoc_versions[]`.

**Files:**
- Modify: `gdoc/pairing.py`
- Modify: `gdoc/cli.py`
- Modify: `tests/test_pairing.py`, `tests/test_cli.py`

**Interfaces:**
- Modifies: `find_by_doc_id(root, doc_id)` also matches `gdoc_versions[].id`,
  not only the current `gdoc:` value.
- Produces: `slug_for_source(md_path: Path) -> str`, the file stem.
- No `doc_id` file, no new index. The frontmatter is the record.

- [ ] **Step 1: Write the failing tests**

In `tests/test_pairing.py`, using the shape the live file already has:

```python
LINEAGE = """\
---
title: Screening proposal
gdoc: 1CurrentVersion
gdoc_versions:
  - id: 1FirstVersion
    created: 2026-08-13
  - id: 1CurrentVersion
    created: 2026-08-14
---

Body.
"""


def test_finds_the_source_by_the_current_version_id(tmp_path):
    (tmp_path / "proposal.md").write_text(LINEAGE)
    assert find_by_doc_id(tmp_path, "1CurrentVersion").name == "proposal.md"


def test_finds_the_source_by_a_previous_version_id(tmp_path):
    """Reviewing v0.1 after v0.2 exists must still land on one source file."""
    (tmp_path / "proposal.md").write_text(LINEAGE)
    assert find_by_doc_id(tmp_path, "1FirstVersion").name == "proposal.md"


def test_unknown_id_still_returns_none(tmp_path):
    (tmp_path / "proposal.md").write_text(LINEAGE)
    assert find_by_doc_id(tmp_path, "1Unrelated") is None
```

The second test is the whole point of this task. It fails today.

- [ ] **Step 2: Run and confirm failure**

Expected: `test_finds_the_source_by_a_previous_version_id` fails, returning
`None`. The other two already pass, which is the proof that only the widening is
missing.

- [ ] **Step 3: Implement**

In `find_by_doc_id`, match `pairing.doc_id` first, then any `id` in
`pairing.versions`. Current version first, so the common case does no extra work.

Add `slug_for_source(md_path)` returning `md_path.stem`. The stem is already
date-prefixed and hand-chosen, so it needs no slugify pass, but run it through
`slugify` anyway to guarantee a safe directory name.

Wire both into `capture`: derive the slug from the paired source file rather than
from the document title, and make `--slug` optional.

Keep `_check_slug_collision`, retargeted. Two source files in different folders
can still share a stem, and that collision is now the only one possible.

- [ ] **Step 4: Migrate the two existing queues by hand**

They live at
`~/src/altery/Altery-Platform-Hub/11-m2-crypto-project-eagle/202607-kyt-travel-rule/.gdoc/`,
beside their source file
`2026-08-13-non-custodial-crypto-screening-control-proposal.md`, whose
frontmatter already lists both document ids.

Both queues belong to that one source, so they become one directory named after
its stem. Merging them means concatenating two `pending.md` files: 12 items from
`ver-0-1` and 1 from `ver-0-2`.

Renumber the merged items sequentially and keep every comment id, since that is
what the dedup guard matches on. Show Nail the merged file before deleting
either original.

- [ ] **Step 5: Run the suite, then commit**

```bash
git commit -am "feat: key queue directories by the source markdown file

Every iteration raises a new document with a new id, so neither the title nor
the document id is stable. The source file is, and its frontmatter already
records the whole lineage. find_by_doc_id now matches past versions too, so
reviewing an old version lands in the one right directory."
```

---

### Task 7: Rewrite the two skills

**Files:**
- Modify: `skills/gdoc-review/SKILL.md`
- Modify: `skills/gdoc-apply/SKILL.md`

Both are symlinked into `~/.claude/skills/`, so every edit is live the moment it
is saved, before it is committed. Nothing warns you.

- [ ] **Step 1: Collapse two roots into one**

Both files currently set `GDOC_REPO="$HOME/src/personal/gdoc"` alongside
`VAULT="$PWD"`. Delete `GDOC_REPO`. One root, `$PWD`, used for the corpus search
and for `.gdoc/` alike. Every `--repo-root "$GDOC_REPO"` goes away, since `$PWD`
is now the default.

- [ ] **Step 2: Move the baseline write out of gdoc-review**

`gdoc-review` Step 7 writes the mirror at the end of a review. Delete that step.
By then the document may already carry direct edits, and baking them into the
snapshot makes them invisible to the next apply. `generate` owns the write now.

- [ ] **Step 3: Repoint gdoc-apply at the baseline**

`gdoc-apply` reads `mirror.md` in the same folder. That becomes `baseline.md`.
The paired markdown stays the file that gets edited: never the baseline.

- [ ] **Step 4: Choose `--out` by intent**

| Case | `--out` |
|---|---|
| a new Drive version | `.gdoc/<slug>/out/v<n>.docx` |
| a document Nail asked for | `$PWD` |
| an explicit destination | that path |

`gdoc-apply` generates a new version, so it uses the first row. The CLI stays
dumb and the skill carries the policy.

- [ ] **Step 5: Make the commit step conditional**

`gdoc-apply` runs `git add` and `git commit` on the paired markdown. Guard both
with a repository check, and say plainly when the step is skipped. A silent skip
would read as a successful commit.

- [ ] **Step 6: Read both files end to end**

Check that no path names a repository, that no step assumes git, and that the
two files agree on `.gdoc/` and `baseline.md`.

- [ ] **Step 7: Commit**

```bash
git commit -am "refactor: detach both skills from any named repo

One root, \$PWD. The baseline write moves to generate, where the document and
the markdown provably match. The commit step is conditional, because the hub is
not a git repository."
```

---

### Task 8: Update the documentation

**Files:**
- Modify: `README.md`, `CLAUDE.md`
- Modify: `docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md`

- [ ] **Step 1: README**

The "Two roots" section is now wrong: there is one root. Replace it with what
`.gdoc/` holds and where it sits. Update the layout block, the `Use` examples and
the direct CLI examples, all of which pass `--repo-root ~/src/personal/gdoc`.

- [ ] **Step 2: CLAUDE.md**

"Two roots, do not conflate them" goes. The `docs/gdoc/<slug>/` table row becomes
`.gdoc/<slug>/` beside the source markdown. Add a line that the tool must work
without git, since that is the constraint an agent is most likely to break.

- [ ] **Step 3: Repoint the spec pointer in gdoc-review**

`skills/gdoc-review/SKILL.md:15` points at the 2026-08-13 spec. Point it at the
2026-08-14 spec, which supersedes the storage parts.

- [ ] **Step 4: Check the whole tree for stale paths**

```bash
grep -rn "docs/gdoc\|intelligence-hub\|GDOC_REPO\|mirror" README.md CLAUDE.md gdoc skills tests
```

Only the two spec files and this plan should mention the old names, and only as
history.

- [ ] **Step 5: Run the suite one last time, check coverage, then commit**

```bash
~/.config/gdoc-agent/venv/bin/pytest --cov=gdoc
git commit -am "docs: one root, .gdoc/, and no git requirement"
```

---

### Task 9: The marker decides, not the author name

Today `partition` splits actionable threads into `mine` and `others` by comparing
`thread.author_name` to `config.display_name`. That split exists to answer "is
this really Nail?", and it cannot: Drive returns no email for comment authors and
display names are editable. So the skill has to print a caveat every run saying
the answer is a guess.

Nail's decision, 2026-08-14: he does not want proof. Launching the skill is the
trust decision. The `ai:` marker is the instruction, and a marker inside a
document Nail chose to point the skill at is enough to act on.

So the author name stops being a gate and becomes a label. `needs_action` already
holds the real rules: marker present, not resolved, not the agent's own comment,
no agent reply yet. That is the whole test after this task.

**Note on ordering:** this task edits `skills/gdoc-review/SKILL.md` Step 2 and
the `Never` list, which Task 7 also rewrites, and it edits `README.md`, which
Task 8 rewrites. The sections do not overlap, so either order works. If Task 7
has not started, doing this task first is cheaper.

**Files:**
- Modify: `gdoc/filters.py`, `gdoc/cli.py`, `gdoc/config.py`
- Modify: `tests/test_filters.py`, `tests/test_cli.py`, `tests/test_config.py`
- Modify: `skills/gdoc-review/SKILL.md`, `README.md`

**Interfaces:**
- Modifies: `partition(threads) -> tuple[tuple[Thread, ...], tuple[Thread, ...]]`,
  returning `(addressed, skipped)`. The `display_name` argument is gone.
- Modifies: `gdoc read` JSON. `mine` and `others` are replaced by one
  `addressed` list. Each item keeps its `author` field, so the report can still
  say who wrote a comment.
- Modifies: `Config` loses `display_name`. `output_folder_id` is the only field
  left, and an old config file that still carries `display_name` loads fine
  because the extra key is ignored.
- Unchanged: `Thread.author_name`, `is_addressed`, `forced_kind`, `needs_action`.

- [ ] **Step 1: Write the failing tests**

In `tests/test_filters.py`, the point of the task is the second test. It asserts
the behaviour that is wrong today:

```python
def test_a_marked_comment_from_anyone_is_actionable():
    """The marker is the instruction. Who typed it does not change the work."""
    threads = (thread(author_name="Nail Khusnullin"), thread(author_name="William Mejia"))
    addressed, skipped = partition(threads)
    assert len(addressed) == 2
    assert skipped == ()


def test_an_unmarked_comment_is_still_skipped():
    addressed, skipped = partition((thread(content="Looks fine to me"),))
    assert addressed == ()
    assert len(skipped) == 1


def test_an_already_answered_comment_is_still_skipped():
    """Dropping the author check must not weaken the idempotence guard."""
    answered = thread(replies=(Reply(id="r1", content="done", by_agent=True),))
    addressed, skipped = partition((answered,))
    assert addressed == ()
```

Delete the two tests that assert the `mine` / `others` split. They pin the
behaviour being removed, so keeping them adapted would be the loosening that
`CLAUDE.md` forbids.

In `tests/test_cli.py`, pin the JSON shape, because the skill reads it:

```python
def test_read_returns_one_addressed_list_with_authors(...):
    payload = read_json(main(["read", URL]))
    assert "mine" not in payload
    assert "others" not in payload
    assert [t["author"] for t in payload["addressed"]] == ["Nail Khusnullin", "William Mejia"]
```

In `tests/test_config.py`, replace `test_missing_display_name_is_rejected` with:

```python
def test_a_config_without_display_name_loads(tmp_path):
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"output_folder_id": "0AFolderId"}))
    assert load_config(path) == Config(output_folder_id="0AFolderId")


def test_an_old_config_with_display_name_still_loads(tmp_path):
    """Nail's live config has the key. Loading must not start failing on it."""
    path = tmp_path / "config.json"
    path.write_text(json.dumps({"display_name": "Nail Khusnullin", "output_folder_id": None}))
    assert load_config(path).output_folder_id is None
```

- [ ] **Step 2: Run and confirm failure**

Expected: `TypeError: partition() missing 1 required positional argument:
'display_name'`, and the config test fails with `ValueError: display_name is
required`.

- [ ] **Step 3: Implement**

In `gdoc/filters.py`: drop the `display_name` parameter, drop the `mine` and
`others` lists, return `(addressed, skipped)`. Rewrite the module docstring. The
first paragraph currently explains why authorship cannot be verified; replace it
with why that no longer matters. Keep the explanation of the marker as it is.

In `gdoc/cli.py` `cmd_read`: call `partition(threads)`, emit `addressed` and
`skipped`. Drop the `load_config()` call, which was only there for the name.

In `gdoc/config.py`: remove `display_name` from `Config`, its validation and the
error message. Keep the missing-file error, and keep it naming a valid example.

- [ ] **Step 4: Prove the author name is gone as a gate**

```bash
grep -rn "display_name\|author_name ==" gdoc skills
```

Expected: `author_name` only where a thread is described or printed, and
`display_name` nowhere in `gdoc/` or `skills/`.

- [ ] **Step 5: Rewrite the skill's Step 2**

In `skills/gdoc-review/SKILL.md`:

- Delete the display-name caveat, the first bullet under Step 2.
- Keep the `permissions.list` caveat and the `Proceed?` gate. That gate is not
  about authorship: a reply is public to everyone on the document and cannot be
  taken back, so the confirmation still earns its place.
- Replace the `Yours` / `Others` sample block with one list. Show the author name
  next to each item, so an unexpected name is visible without being a blocker:

```
Will act:         para 3, para 7, para 11 (Nail), para 5 (William Mejia)
Already answered: para 2

I cannot see who else has access to this document. Replies will be
visible to everyone on it, under the service account address.

Proceed?
```

- In the `Never` list, `Never act on a comment that is not Nail's` becomes
  `Never act on a comment without the marker`. That is the rule the code now
  enforces, and the old line would read as a promise the tool no longer makes.

- [ ] **Step 6: Update the README**

Wherever the README explains the `mine` / `others` split or the display-name
config field, say instead that the marker decides and that `read` returns one
actionable list. State the trade plainly in one line: a marked comment from
anyone in the document is acted on, and Nail chooses the document.

- [ ] **Step 7: Run the suite, check coverage, then commit**

```bash
~/.config/gdoc-agent/venv/bin/pytest --cov=gdoc
git commit -am "feat: act on the marker, not on the author name

Drive gives no email for comment authors, so the display-name split was a guess
dressed as a check, and the skill had to disclaim it every run. Pointing the
skill at a document is the trust decision. The marker is the instruction, so
partition now returns one actionable list and config drops display_name."
```

---

## Not in this plan

- **Part B, merging suggestions.** No longer blocked. The 2026-08-14 spike
  proved a Commenter-role service account reads them through the Docs API
  `suggestionsViewMode`, with no new scope and no Editor role. What is left is
  the merge design, plus one new open question: the response carries no author,
  so suggestions cannot be attributed the way comments can. That needs its own
  plan.
- **The two-sided dedup gap.** `pending.md` dedups by comment id, but
  `/gdoc-apply` deletes each item as it applies it, so a later review captures
  the same comment again. Real, worth fixing, out of scope here.
- **`gdoc pending list`** and a central document index. Both dropped in the
  spec, with reasons.
