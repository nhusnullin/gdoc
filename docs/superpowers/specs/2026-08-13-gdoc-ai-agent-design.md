# Google Docs review agent

Status: design, not built
Last updated: 2026-08-13
Owner: Nail

> **Partly superseded on 2026-08-14** by
> [2026-08-14-gdoc-storage-and-iteration-design.md](2026-08-14-gdoc-storage-and-iteration-design.md).
>
> Three things below are now wrong. Storage: files go in `.gdoc/` beside the
> source markdown, not in `docs/gdoc/` in the tool's own repo, and the root is
> `$PWD`. The mirror: it is renamed `baseline.md` and written by `generate`
> after a successful upload, never at the end of a review. Git: this spec
> assumes a git repository throughout, and the hub holding the source documents
> is not one, so nothing may refuse to run without git.
>
> Everything else here still holds, including the comment marker rules, the
> Commenter guarantee, and what section 10 records as verified.

A global skill you invoke by hand. You give it a Google Doc link. It reads the
`ai:` comments you left, answers the local ones in their threads, and tells you
plainly which ones need a new version of the whole document.

v1 is manual on purpose. See section 11 for what that removes.

---

## 1. What it is

```
You review a doc in Google Docs, leaving ai: comments as you go.
Then, in a terminal session, in whatever hub gives the right context:

  /gdoc-review https://docs.google.com/document/d/1AbC.../edit

  Found 6 comments. 4 are yours, 2 are from William.
  Of your 4: 3 look local, 1 needs a new document version.
  Proceed? y

  posted   para 3   answered: yes, matches CBC art 12(2)
  posted   para 7   rephrased, ready to paste
  posted   para 11  answered: no source in the hub, flagged
  captured para 2   global: renumber sections after the new 4.3

  1 global item saved to docs/gdoc/safeguarding-v3/pending.md
```

You copy the text you want, resolve the comments yourself, and come back next
session for the global work. You stay in control of the document at all times.

## 2. What v1 does not do

- **No automatic trigger.** No polling, no watcher, no launchd, no Pub/Sub. You run it.
- **No edits to the reviewed document.** The service account holds Commenter, not
  Editor, so it cannot change the document even if a comment tells it to.
- **No global rewrites during a review run.** Those are captured and named, never
  attempted. See section 4.
- **No resolving comments.** You resolve them, because resolving means you accepted
  the text.
- **No vector index.** Grep over the hub, for the reasons kept from the earlier design.

## 3. How you use it

The skill is global, at `~/.claude/skills/gdoc-review/`, laptop only. Global because
the context you want changes per document, and the cwd is how you choose it. Running
it from `intelligence-hub` gives one corpus, from `Altery.Platform-hub` another. That
choice is yours per run, which is why no sandbox is needed.

```
/gdoc-review <url>                  review, post replies in threads
/gdoc-review <url> --md <path>      state the paired markdown file explicitly
/gdoc-review <url> --terminal-only  print answers here, post nothing
```

Replies are posted on every document, including documents you do not own. Your
decision, and it keeps one behaviour instead of two.

**The author label is the raw service account address**, verified on 2026-08-13:

```
nail-ai@doc-agent-505414.iam.gserviceaccount.com
```

The display name set in the Cloud console is a GCP-side label and Drive ignores it.
Accepted as is. A friendly "Nail AI" label would need a real Google identity, which
is the Cloud Identity Free upgrade in section 10, not a v1 concern. Tell colleagues
what the address is before they find it themselves.

`--terminal-only` stays available for the cases where you would rather paste the
answer as yourself. It is a flag you reach for, not a default.

## 4. Local versus global

This is the central distinction, and the agent must never blur it.

| | Local | Global |
|---|---|---|
| Examples | A question. Rephrase this sentence. Is this term right? Add a missing clause | Renumber sections. Restructure. Apply a term change everywhere. Split a section |
| Test | The change fits inside the quoted span | The change touches text the comment does not quote |
| Output | A reply in the thread with text you can paste | A captured item plus a short reply saying so |
| When | Now, in this run | A later session, from a markdown file |

When a comment is global, the reply is short and fixed in shape:

```
Understood. This needs changes across the document, so I am not
proposing text here. Captured as item 2. I will handle it in a
terminal session and produce a new version of the document.
```

**Why global work is refused in-thread rather than attempted.** A comment reply can
only carry one span of text. A change touching six sections cannot be reviewed as six
separate replies, because you would have to hold consistency in your head while
pasting. The markdown file is where consistency is checkable, and `git diff` is how
you check it.

If a comment is ambiguous, the agent asks you in the terminal rather than guessing.
Guessing global wastes a session. Guessing local posts text you cannot use.

## 5. Only your comments

The skill acts on a comment only if both are true: you wrote it, and it starts
with the marker.

**The marker.** A comment addressed to the agent starts with `ai` and a sign:

| Written | Means |
|---|---|
| `ai:` | you decide whether this is local or global |
| `ai?` | answer it in the thread |
| `ai!` | this one is global, capture it |

It starts with a letter, not `@`. Typing `@` in a Google Docs comment opens the
people picker from your address book, which is a nuisance when the thing you are
addressing is not a person. A letter opens nothing.

The sign is required. A comment that just begins with the word "AI" is ordinary
prose and is left alone. The older `@ai` form still works, with or without a
sign, so comments written before this change are not silently ignored.

Author identity from the Drive API is not fully reliable. `author.emailAddress` is not
always populated, and display names are editable. Rather than build an allowlist,
**the skill shows what it found and asks before posting anything**:

```
Yours (will act):      para 3, para 7, para 11, para 2
Others (context only): para 5 (William), para 9 (William)
```

You are already at the keyboard, so a confirmation is both cheaper and stronger than
any identity check. Other people's comments are still read, because they are context
for answering yours, but they never get a reply.

The marker is still needed even though you start the run by hand. Not every comment
you leave is for the agent. Some are notes to yourself or to a colleague.

## 6. Reply format

Google Docs comment replies are plain text. Markdown is not rendered, so a fenced
block arrives with the backticks still in it and you have to strip them.

Rules:
- Proposed replacement text comes **first**, so the first thing you see is the thing
  you paste.
- No markdown syntax. No `**`, no backticks, no bullets that depend on rendering.
- One short reason line after the text, only when the reason is not obvious.
- Sources as plain paths: `domains/regulatory/cbc-emi.md`.
- If the hub has no source for the answer, say so in the reply. Never fill the gap
  with a plausible sentence.

Answers to questions use the same shape: answer first, then the paths it came from.

## 7. The paired markdown file

A reviewed document falls into one of three cases, and the skill settles which before
doing anything:

| Case | How it is resolved |
|---|---|
| The doc came from a markdown file you own | `--md <path>`, or a `gdoc:` key in that file's frontmatter, or the skill asks |
| The doc is yours but has no markdown source | On first run the skill mirrors it to markdown and records the pairing |
| The doc is not yours | No pairing. Review only. Global items become a note, not a rewrite |

The pairing lives in the markdown file's own frontmatter, not in a separate index:

```yaml
---
gdoc: 1AbC...
gdoc_synced: 2026-08-13
gdoc_versions:
  - id: 1XyZ...
    created: 2026-08-14
---
```

Frontmatter rather than a central mapping file because the skill is global and runs in
different repos. A central index would have to live outside them all, and would go
stale silently when a file moves. Frontmatter moves with the file.

**Mirroring follows the pairing.** A document is mirrored to markdown only when a
pairing exists or is being created, which means cases 1 and 2 above. **A document you
do not own is never mirrored into the hub.** Counsel drafts, partner documents and
anything else you only review stay in Drive. You still get answers on them, and
global items are still captured as a note, but no copy lands in a repo you push.

## 8. Files it writes

```
<paired md file>               frontmatter gains gdoc, gdoc_synced, gdoc_versions
docs/gdoc/<slug>/pending.md    captured global items, in the cwd repo
docs/gdoc/<slug>/mirror.md     the doc as markdown, only when paired
docs/gdoc/<slug>/out/v3.docx   pandoc output, kept only if the Drive upload failed
```

In the cwd repo, because you chose that repo for its context and that is where the
global work will happen. `pending.md` and `mirror.md` are committed: "what did I ask
for, and when" is useful history. `out/` is gitignored, since it is a build product of
the markdown.

No queue, no database, no state directory. The run is synchronous and you are watching
it.

## 9. The next session, for global items

```
/gdoc-apply docs/gdoc/safeguarding-v3/pending.md
```

Loads the pending items, the mirror, and the paired markdown file. Works through the
items with you, editing the markdown. You review the diff in git. **The markdown is
what you approve.** The Google Doc is generated from it, never edited directly.

Then the new version, in four steps:

```
1. markdown       the approved file, after your review
2. pandoc         md -> docx
3. Drive create   upload the docx into the Shared Drive folder,
                  converting to a native Google Doc
4. frontmatter    append the new id and date to gdoc_versions
```

The original document is never touched. It keeps its comments and its history, and the
new version is a separate file, so nothing you reviewed can be lost by a bad generation.

**One converter, two destinations.** If step 3 fails, for any reason including no
network or no folder rights, the `.docx` from step 2 stays at
`docs/gdoc/<slug>/out/v<n>.docx` and the run says so. The fallback is not separate
code, it is the same pipeline stopping one step early.

Why `.docx` and not the Docs API: Drive converts an uploaded `.docx` into a native
Google Doc on create, so headings, tables and lists survive without building the
document element by element through `batchUpdate`. Less code, and it does not drift
when you restructure the markdown.

Requires `pandoc`, present at `/opt/homebrew/bin/pandoc`, version 3.3.

## 10. Access model

One service account, `nail-ai@doc-agent-505414.iam.gserviceaccount.com`, display name
`Nail AI`. Scope `https://www.googleapis.com/auth/drive`. Key at
`~/.config/gdoc-agent/sa-key.json`, mode 600, never in a repo.

**Access is sharing, not configuration.** The service account owns nothing and sees
only documents shared with it. To let the agent work on a document you share it; to
stop it you unshare. There is no list of document ids to maintain, and no way for a run
to widen its own reach.

**Commenter, not Editor, on reviewed documents.** This is the load-bearing choice. It
is not a policy the agent follows, it is a permission Google enforces. A prompt
injection inside a document cannot make the agent edit that document, because the
credential holds no such right.

**Write rights exist in exactly one place: the output folder.** You add the service
account as **Content manager** on one folder in a Shared Drive. That is the only
location it can create anything. The folder is `1w0SresizE9Kr810VZRJwX4JtDBF4OqNr`,
named "test folder", on Shared Drive `0AA2s2yLSmVTFUk9PVA`. It is a test folder for
now; swapping it later is one line in `~/.config/gdoc-agent/config.json`. Two
properties follow, and both are worth having:

- Files created in a Shared Drive are **owned by the Shared Drive**, not by the service
  account. You find them in your normal Drive interface, and they do not disappear into
  a non-human account's space or hit its storage limits.
- The blast radius of the credential is one folder for writes and a list of shared
  documents for comments. Nothing else in Drive exists for it.

So the permission picture is small enough to hold in your head:

| What | Right |
|---|---|
| Documents under review | Commenter, per document, by sharing |
| One output folder in a Shared Drive | Content manager |
| Everything else in Drive | No access. It does not exist for this account |

Two limits follow, both real:

- **A document you cannot share, the agent cannot see.** If you hold only Commenter on
  someone else's document, you cannot add the service account. The skill must fail with
  a clear message naming the address to ask the owner for, not a raw 404.
- **Replies are attributed to the raw address** everywhere, including on documents you
  do not own. Not "Nail AI": Drive shows
  `nail-ai@doc-agent-505414.iam.gserviceaccount.com`. Accepted deliberately, for one
  consistent behaviour. A friendly label needs a Cloud Identity Free user, which is a
  later upgrade and changes nothing else in this design.

**The one control that survives from the automatic design, and who now performs it.**
A reply is visible to everyone on the document, and your session carries your whole
hub, so an outside collaborator is the one real leak path. The original plan had the
skill check the document's permissions and warn you about anyone outside `altery.com`.

It cannot. Under Commenter, `permissions.list` returns **403** (verified 2026-08-13).
The agent has no way to see the audience.

So the check moves to you. The skill states plainly, every run, that it cannot see who
else has access, and the confirmation prompt in section 5 is where you apply what you
know about the document. This is weaker than an automatic check, and saying so is
better than implying a gate that does not exist.

### Verified on 2026-08-13

Probes run against a throwaway document, so the access model is proven rather than
assumed. Build order steps 1 and 2 are done.

| Checked | Result |
|---|---|
| `files.get` | Works. Name, mime type and owner returned |
| `comments.list` | Works. Three threads read |
| `quotedFileContent` | Present on every anchored comment, so the agent gets the exact selected text without you pasting it |
| `author.emailAddress` | **`None` on all three comments.** An email-based author allowlist is not possible. Section 5's interactive confirmation is the answer, and it is now evidence-backed |
| `author.me` | Correctly `False` for your comments. Reliable enough for "never reply to my own replies", which is what prevents a loop |
| `replies.create` | Works. Three replies posted, one per comment class |
| Reply author label | The raw service account address, not the console display name |

### Verified after the downgrade to Commenter, same day

The share was changed from Editor to Commenter and the probes were re-run.

| Checked | Result |
|---|---|
| `capabilities.canEdit` | **`false`**. Google's own answer: this credential cannot edit the document |
| `capabilities.canComment` | `true` |
| `comments.list` | Still works |
| `replies.create`, then `replies.delete` of the agent's own reply | Both work. Commenting is untouched by the downgrade |
| `permissions.list` | **403 refused.** The agent cannot see the audience. The control above moved to you because of this |
| Output folder `1w0Sresiz...` | `canAddChildren: true`, `driveId` present, so it is a real Shared Drive folder |
| Folder `1KmUrVKL...` in My Drive | `canAddChildren: true` as well, but files there would be owned by the service account. Not used |

### Verified in Task 3 (2026-08-13)

Integration tests in `tools/gdoc/tests/test_access_integration.py` ran against the live API with `-m integration`. All 8 passed.

| Checked | Result |
|---|---|
| `files.update` rename to `<original> + " PROBE"` | **403 refused.** Google enforces the Commenter boundary on metadata writes |
| `docs.documents().batchUpdate` insert text at index 1 | **403 refused.** Google enforces the Commenter boundary on document body writes |
| `comments.delete` | **403 refused.** Confirmed |
| `permissions.list` | **403 refused.** Confirmed again |
| `files.get` read and `comments.list` read | Still work |
| Output folder `canAddChildren` | `true`, `driveId` present |

One earlier probe result is worth not repeating: a `files.update` that set the name to
its existing value returned 200 under Commenter. That was a no-op patch, not a
permission grant. A write test has to change something.

### Verified in Task 15 (2026-08-13), after the CLI and both skills shipped

These ran through the shipped CLI, not through probe scripts, so they test what
Nail will actually run.

| Checked | Result |
|---|---|
| `gdoc read` on the test document | Works. Returned the name, the derived slug `test-doc`, and all three comments with their quoted text and `anchored: true` |
| Idempotency, live | All three comments came back under `skipped`, because the service account had already replied to them. Nothing was posted. This is the property that makes a second run safe, and it is now proven against the API rather than only in unit tests |
| Author identity through the CLI | Display name present, no email address. Matches the earlier probe |
| `gdoc generate` upload failure | `--folder-id 1invalidFolderIdForTesting` returned `doc_id: null`, a reason naming the folder, and a real 10661-byte `.docx` on disk. The fallback is not theoretical |
| `gdoc generate` real upload | Created a document in output folder `1w0Sresiz...` and returned a link |
| pandoc round trip | The uploaded document was exported back to markdown. The heading survived as `#` and the list survived as `*`. The conversion holds |

The upload probe document was trashed after the check, so nothing was left in the
shared drive.

### Still unproven, and why

Four behaviours need Nail at the keyboard. They are listed as a checklist in
section 13 rather than claimed here.

1. **The new marker in the field.** `ai:`, `ai?` and `ai!` were chosen after the
   probes ran. The live document still carries only the legacy `@ai` form. The
   marker rules are covered by unit tests but no real comment has used them.
2. **The decoy.** A comment starting `AI tools are changing this` must be ignored.
   The marker now starts with a plain letter, so this is worth one real run.
3. **Unanchored comments.** All three test comments were anchored, so the no
   `quotedFileContent` path is still untested against the API.
4. **The agent's own behaviour.** Answering, capture, and the refusal reply are
   the skill's judgement, not the CLI's. Only a real run exercises them.

## 11. What the manual trigger removes

The earlier automatic design is in git at `1615eb2`. It carried a watcher, a poll loop,
a dedicated macOS user per tier, a separate clone per tier, a `sudoers` entry, a handoff
directory, a fail-closed author allowlist, and a `policy.yaml`.

All of it defended one thing: **an automatic trigger is written by whoever can comment
on the document.** Anyone with comment access could have made the agent run with a
prompt of their choosing, so the agent's reach had to be small enough that a hostile
prompt could not hurt.

Invoking by hand removes that premise. You pick the document, the cwd and the moment.
The agent's context is your context, deliberately, which is the point of the feature.
Untrusted text still arrives, as other people's comments and the document body, but it
can only shape a reply you are watching, from a credential that cannot edit anything.

Promote v2 when the manual step starts to annoy you. The pieces are specified and still
valid.

## 12. Failure handling

| Failure | Behaviour |
|---|---|
| Service account has no access | Stop. Print the address to share with, and the role needed |
| Doc may have external collaborators | The agent cannot check: `permissions.list` is refused under Commenter. Say that in the confirmation prompt, so the decision is yours and you know it is yours |
| Comment is ambiguous, local or global | Ask in the terminal. Never guess |
| No hub source for an answer | Say so in the reply. Never write a plausible sentence to fill the gap |
| Reply posting fails midway | Report which comments got replies and which did not. Never retry silently, or you double-post |
| Doc changed during the run | Ignore for v1. The run is short and you are the only editor |
| Paired md file missing | Ask. Do not create a pairing silently |
| Doc is not yours | Answer and reply as normal, but do not mirror it into the hub |
| `pandoc` missing | Stop before editing anything. Name the install command |
| Drive upload of the docx fails | Keep the `.docx`, print its path, and say the Google Doc was not created. Never leave the frontmatter claiming a version that does not exist |
| No write rights on the output folder | Same as above. Fall back to the local `.docx` and name the folder and role needed |
| A comment already has a reply from Nail AI | Skip it and say so. You resolve threads, so an answered unresolved thread means you are still reading it |

## 13. Testing

The API layer is a small script and is testable. The skill's judgement is not, so it is
exercised by hand.

- **Unit, on the script:** doc id extracted from every Google Docs URL shape. Comment
  list parsed, including unanchored comments with no `quotedFileContent`. Reply body
  sent as plain text with no markdown escaping. The marker fires on `ai:`, `ai?`, `ai!`
  and `@ai`, and does not fire on a sentence that merely starts with the word AI.
- **Access, and these matter most:**
  - With Commenter only, an attempted document edit fails. Assert the API rejects it,
    not that the agent chose not to try. The edit must be a real change: a rename to the
    same name returns 200 and proves nothing.
  - Both a metadata write (`files.update`) and a content write (Docs `batchUpdate`) are
    refused. The second is the one the design actually promises.
  - A document not shared with the service account gives the clear message, not a 404.
- **By hand, on one real document:** a question, a rephrase, and a global request in one
  run. Assert the global one is captured and not answered with text.
- **The regression that matters:** a second run over the same document posts nothing new.
  Done. See the Task 15 table in section 10: a re-read returned every comment as
  `skipped`.

### The by-hand run, still to do

Everything the CLI does is proven. What is not proven is the skill's judgement and
the new marker in a real comment. This takes about ten minutes.

1. On the test document, resolve the three existing `@ai` threads, then leave five
   fresh comments:

| Comment | What it tests |
|---|---|
| `ai? <a question about the quoted text>` | forced question, answered in the thread |
| `ai: <asks for better wording>` | the agent decides, and should answer locally |
| `ai! renumber the sections` | forced instruction, captured not answered |
| an unanchored `ai: ...` on the whole document | the last untested API path |
| `AI tools are changing this` | the decoy. Must be ignored |

2. Run `/gdoc-review https://docs.google.com/document/d/1tyVhOTw9-bJ99IJTBZoTfWAT6fm3rjGIzfpHQX91hkw/edit`

3. Check each of these:

- [ ] The confirmation prompt appeared before anything was posted
- [ ] The question was answered with a file path cited
- [ ] The rephrase reply put the replacement text first
- [ ] The structural request was captured, not answered with text
- [ ] The unanchored comment was handled without crashing
- [ ] The decoy comment was left alone
- [ ] `pending.md` exists with the right item number
- [ ] No thread was resolved
- [ ] The document itself is unchanged

4. Run `/gdoc-apply docs/gdoc/test-doc/pending.md` and check the new document opens.

The decoy is the row worth caring about. The marker now starts with a plain letter,
so a real run is the only place that test means anything.

## 14. Build order

0. **Dependencies.** `pip install google-auth google-api-python-client` in a venv.
   `pandoc` is already installed. Create the service account per section 10.
1. **Auth and read.** Share one doc, list its comments, print them. Proves the access
   model and settles the scope question. Everything else depends on this.
2. **Post one reply.** Confirms Commenter is enough to reply, which is the assumption the
   whole access model rests on.
3. **Create one doc in the output folder.** Upload a two-line `.docx` and confirm it
   converts and lands where you can see it. Settles ownership before it matters.
4. **Mirror a paired doc to markdown.** Useful alone: the corpus grows and you get diffs.
5. **The skill, local only.** Fetch, filter, confirm, classify, answer, report. Global
   items are named at this stage, not stored.
6. **Capture global items.** `pending.md` and the fixed reply.
7. **`/gdoc-apply`.** The next-session path over the markdown file.
8. **Generate the new version.** pandoc, upload, frontmatter, `.docx` fallback.

Steps 1 to 3 are an afternoon and remove every assumption that could invalidate the
design: can it read, can it reply, can it create where you want. Do them before writing
any of the skill.

## 15. Decisions taken

| Decision | Rationale |
|---|---|
| Manual invocation for v1 | Removes the untrusted trigger, and with it the entire sandbox. You already work session by session |
| Global skill, cwd chooses the corpus | Which context is right changes per document. cwd is the control, and it is yours |
| Service account, sharing as the access list | No seat, no admin, no password. Sharing is the on and off switch |
| Commenter, not Editor | Google enforces it. The agent cannot damage a document it is reviewing, whatever a comment says |
| Local answered, global refused and captured | A comment reply carries one span. Cross-document consistency is only checkable in a diff |
| You resolve the threads | Resolving means you accepted the text. That is your judgement, not the agent's |
| Confirm before posting, rather than an author allowlist | You are at the keyboard, and Drive's author fields are not reliable enough to be an access control |
| Pairing in frontmatter | The skill runs in many repos. A central index would go stale when a file moves |
| Plain text replies, paste-ready text first | Docs comments do not render markdown |
| No local state beyond two committed files | The run is synchronous and you are watching it |
| New versions into a Shared Drive folder | The folder owns the file, not the service account, so you find it normally and storage limits never come up |
| Write rights on one folder only | Comments everywhere it is shared, creation in exactly one place |
| `.docx` via pandoc as both the upload format and the fallback | One converter, two destinations. The fallback is the same pipeline stopping early, not a second code path |
| The markdown is what you approve, the doc is generated | A diff is reviewable. A regenerated document is not |
| Replies always from the service account, `--terminal-only` as a flag | One behaviour everywhere. Nail accepted the raw address as the author label |
| `--terminal-only` is a skill argument, not a CLI flag | It means one thing in both skills: nothing leaves the machine. In `/gdoc-review` no reply is posted. In `/gdoc-apply` the run stops after the markdown commit, because `generate` always attempts an upload and a local-only mode would need a CLI flag the design does not ask for |
| Keep project id `doc-agent-505414` | The generated suffix is visible in every reply. Nail decided the long name is not an issue, so it stays |
| Mirror only paired documents | Documents you do not own never land in a repo you push |
| Marker is `ai:` and not `@ai` | `@` opens the Docs people picker, so every marker meant picking a name out of an address book. A letter opens nothing. `@ai` still works so old comments are not silently dropped |
| You check the audience, not the agent | `permissions.list` is refused under Commenter. The agent cannot see who else has access, so the skill says so instead of implying a check it cannot perform |
| Mirror refuses to overwrite uncommitted edits | The markdown is what Nail approves. git already knows whether local work would be lost, so no extra state is needed |

## 16. Open questions

1. **Unanchored comments.** A comment on the whole document has no `quotedFileContent`.
   The code handles it: `anchored: false` and no quote, treated as global unless it is
   plainly a question. That path has unit tests but has never seen a real comment. It is
   row 4 of the by-hand run in section 13.
2. **Suggestions mode.** You review documents where others left suggestions. Whether the
   agent should read pending suggestions as context is undecided.

Question 4, which Shared Drive folder, is settled: `1w0SresizE9Kr810VZRJwX4JtDBF4OqNr`
on drive `0AA2s2yLSmVTFUk9PVA`, verified writable on 2026-08-13. It is named "test
folder", so expect to point this at something permanent later.
