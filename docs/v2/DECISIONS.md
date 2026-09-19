# gdoc v2 decisions

Dated, replaceable. Each names the principle it serves, or says it is a domain
choice with none above it. Same convention as [PRINCIPLES.md](../../PRINCIPLES.md).

v2 is a rewrite in Go. The three principles carry over unchanged. What changes is
which decisions implement them, and this file is where those start.

## The register

Every decision in this file, in date order, with its status stated rather than
inferred. Status is one of `holds`, `rejected`, `superseded YYYY-MM-DD (what
replaced it)`, or `MEASURED.md`. Nothing else.

| Date | Decision | Status |
|---|---|---|
| 2026-08-13 | The markdown is the source, the Google Doc is a rendering | superseded 2026-09-19 (no side is the source by rule; the person decides per run) |
| 2026-08-14 | Skills are symlinked into `~/.claude/skills/`, never copied | holds |
| 2026-08-15 | The client reaches only the files it was given | holds |
| 2026-08-18 | The OAuth client is shipped in git, and the client stays Internal | superseded 2026-09-16 (the secret is injected at build time; the client stays Internal) |
| 2026-08-18 | Nothing runs git, and the agent never commits | holds |
| 2026-08-29 | The next version of gdoc is written in Go | holds |
| 2026-08-29 | gdoc never replaces the body of a document that already exists | holds |
| 2026-08-29 | Never trust a success. Verify with a second, independent probe | holds |
| 2026-08-29 | gdoc marks its own changes by colour, never by highlight | superseded 2026-08-29 (the colour scheme is retired) |
| 2026-08-29 | A comment is a prompt. `align` is a separate question | holds |
| 2026-08-29 | The marker is the trigger. Always | holds |
| 2026-08-29 | gdoc judges whether to ask, rather than always asking | holds |
| 2026-08-29 | There is no `release` command, and gdoc does not export PDFs | holds |
| 2026-08-29 | Anchors are not banked | rejected |
| 2026-08-29 | The Developer Preview arrived, and a third of this file retires | holds |
| 2026-08-29 | gdoc proposes. It never accepts and never rejects | holds |
| 2026-08-29 | MCP does not replace the REST client | holds |
| 2026-08-29 | Tabs are out of scope, and gdoc defends against them | superseded 2026-09-19 (the reader half only: `export` writes one file per tab; every writer still refuses a document with more than one) |
| 2026-08-29 | Amendments from reading the documentation | holds |
| 2026-08-29 | gdoc can retract its own proposal, and must check that it landed | holds |
| 2026-08-29 | The house style is a config file. The .docx becomes provenance | holds |
| 2026-08-29 | The finishing checklist is for `restyle` only | superseded 2026-09-09 (what an in-place restyle changes: the list is reported, not written) |
| 2026-08-29 | `publish` builds a file. `restyle` has two modes | superseded 2026-09-11 (`restyle --new` is not built) |
| 2026-08-29 | A comment is answered with the hub open | holds |
| 2026-08-29 | Emoji reactions: still no API, preview or not | holds |
| 2026-08-29 | The colour scheme is retired, not forgotten. Its "no named ranges" bullet is superseded 2026-09-10 (the house template reaches a document as a suggestion), which gives gdoc one named range as its own marker | holds |
| 2026-08-29 | Alignment runs both ways, and it is not a command | holds |
| 2026-08-29 | The product is three things, and it matters which does what | holds |
| 2026-08-29 | Everything gdoc writes opens with 🤖 | holds |
| 2026-08-29 | gdoc reads back what it proposed, before saying it worked | holds |
| 2026-08-29 | The preview may vanish, and the risk is accepted. Loudly | holds |
| 2026-08-29 | The guard owns the transport, and every id carries a write level. Widened 2026-09-09 (M7 splits) by a third level, `LevelInPlace` | holds |
| 2026-08-29 | Publish runs once. Everything after travels as suggestions | superseded 2026-09-19 (a note may be published more than once, and each publish is a new document) |
| 2026-08-29 | Live review is a session, not a service | holds |
| 2026-09-07 | A replace proposal cannot be fully withdrawn by a delete | holds |
| 2026-09-08 | v1's `gdoc: <id>` stays refused. Publish is the only writer of the block | superseded 2026-09-19 (the one-writer clause only: `export` writes the block too, and the block is a list of documents; the v1 refusal holds) |
| 2026-09-09 | Which styling requests land in place | MEASURED.md |
| 2026-09-09 | The guard carries `files.copy`, for one source | holds |
| 2026-09-09 | Seven paragraph elements were dropped at decode | holds |
| 2026-09-09 | M7 splits. The survey ships without the write | holds |
| 2026-09-09 | What an in-place restyle changes, and what it deliberately does not write | holds |
| 2026-09-10 | In-place styling preserves anchors and pending suggestions | MEASURED.md |
| 2026-09-10 | The house template reaches a document as a suggestion | holds |
| 2026-09-10 | A named range over a suggested insertion | MEASURED.md |
| 2026-09-10 | A table takes one index of its own at the end | MEASURED.md |
| 2026-09-11 | `restyle --new` is not built | holds |
| 2026-09-16 | Help is an answer, completion is a written file, and two skills learn the tool from the tool | holds |
| 2026-09-16 | M8 is deferred to the backlog, and the release goes next | holds |
| 2026-09-16 | The release: `x.y.z` with a nightly, an updater on demand with one read-only guard door, skills as a Claude Code plugin | superseded 2026-09-18 (the "no check, no stamp file" clause only: `help` checks once a day; everything else in the entry holds) |
| 2026-09-16 | The source repository is public, and the client secret is injected at build time | holds |
| 2026-09-16 | What the update door actually reaches: two asset hosts, one bigger page, and a version that is only ever a tag | holds |
| 2026-09-17 | There is no rc. The nightly is the pre-release channel | holds |
| 2026-09-17 | The plugin is named `altery`, and the marketplace stays `gdoc` | holds |
| 2026-09-18 | The wait polls every two seconds | holds |
| 2026-09-18 | The binary notices a release by itself, once a day from `help`, and the plugin carries the stable number | holds |
| 2026-09-18 | annotate: a comment on quoted words, and nothing else | holds |
| 2026-09-18 | Five backlog items closed, and what each decided | holds |
| 2026-09-19 | Export, and the `gdoc:` block as a list of documents | holds |

**An entry is never edited after this, except its status line.** A decision that
changes is a new entry, dated today, with a new row here, and the old entry's
status says what replaced it. That is what makes a status worth reading: it says
the entry below was looked at, rather than that nobody has been back to it.

A row whose status is `MEASURED.md` has its text in
[MEASURED.md](MEASURED.md), under a heading of the same name. It moved because a
measurement is what Google does, not a choice gdoc made.

## 2026-08-13. The markdown is the source, the Google Doc is a rendering.

Written in PRINCIPLES.md on the date above and moved here on 2026-09-15, when
that file became the four principles and nothing else.

Domain choice, no principle above it. A document-wide change is made in the
paired markdown and reaches the document as a new version of it. A reply goes in
the comment thread that asked, because a thread is comment surface and not
content.

In v2 the same rule decides which command does what. `publish` renders the note
into a document. `propose` reaches into a document, and everything it writes
there is a suggestion. `reply` writes into a thread and nowhere else.

## 2026-08-14. Skills are symlinked into `~/.claude/skills/`, never copied.

Written in PRINCIPLES.md on the date above and moved here on 2026-09-15.

Domain choice, no principle above it. A copy drifts silently: an edit to the
skill in this repository would be invisible to the session running the copy.
`~/.claude/skills/gdoc-review` is a symlink into `skills/`, so an edit is live
the moment it is saved and before it is committed. `./install.sh` defends this.
It refuses to replace a real directory whose contents differ, and it is safe to
re-run.

## 2026-08-15. The client reaches only the files it was given.

Written in PRINCIPLES.md on the date above and moved here on 2026-09-15.

Serves principle 3. Under OAuth the credential can reach every file the
signed-in user owns, so the guard holds a set of file ids, the ones the command
was handed plus the ones its own creates returned, and refuses every request
addressing anything else. A listing is refused outright, so the tool cannot
search Drive. An empty set refuses everything, so a command that does not name a
document reaches nothing.

The cost is stated rather than hidden: on a file in the set, the guard bounds
which methods carry, not what a method could do to the words. "The agent cannot
edit a reviewed document" is "it does not", and the decision of 2026-08-13 above
is held up by design rather than by a permission Google enforces.

This is the entry the 2026-08-29 decision "The guard owns the transport" below
is built on: v2 adds the write levels to the same set of ids, and
`internal/guard` is where both live.

## 2026-08-18. The OAuth client is shipped in git, and the client stays Internal.

Serves principle 1. Nobody using gdoc visits a cloud console, so the client id
and secret are two constants in the source and a login is one browser trip.

**The secret belongs in version control.** RFC 8252 section 8.5 says a secret
shipped to many users "should not be treated as confidential" and serves no
purpose "beyond client identification". `gh` ships its own with the comment
"This value is safe to be embedded in version control", and `gcloud` ships a
Google secret in a constant named `CLOUDSDK_CLIENT_NOTSOSECRET`. So moving it to
a config file is not a fix, and a reader who tries should read this entry first.
What protects an account is the per-user token, which never leaves the machine.

**The client stays User type Internal.** gdoc needs the full Drive scope, which
Google classes as restricted. Internal exempts gdoc from verification, from the
unverified-app screen and from the 100-user cap. External would mean a CASA
assessment every 12 months and refresh tokens expiring weekly.

**A file in the config directory wins over the bundled client.** That is an
override for quota, not a setup step, the way gcloud treats `--client-id-file`.
One shared client shares one Google rate limit, which is why rclone is retiring
its shared Drive client during 2026. The code reports where the client came
from, so a status command can say which one is in use.

### The one thing that changes this decision

Making the repository public. GitHub secret scanning carries a **partner**
pattern for `google_oauth_client_id, google_oauth_client_secret`. On a public
repository a hit is reported to Google, who may revoke the client. So publishing
this repository would break every colleague's login at once, without warning and
without a commit to blame. rclone obfuscates its Google secret for exactly this
reason, which is evasion of automated revocation rather than security.

Before this repository is ever public: create a fresh client, distribute it as a
file out of band, and clear the two constants. Do not obfuscate them to get past
the scanner.

## 2026-08-18. Nothing runs git, and the agent never commits.

Written in PRINCIPLES.md on the date above and moved here on 2026-09-15.

Serves principle 1. Committing is Nail's job. A person receiving this tool
should not have to think about git at all, and the skills should not carry a
conditional and a "nothing was committed" sentence for a case that may never
apply to them.

The skills do not commit. They name every file that changed on disk instead, so
Nail commits them himself. Trying the commit and reporting what git said was
considered and rejected: that still makes committing the agent's job.

Nothing asks git anything either, not whether the tree is a repository and not
whether a file is dirty. Outside a repository the answer is always "cannot
tell", which is most of the time, and where the root is a synced folder a
Dropbox or Nextcloud rewrite is exactly what git cannot see, so the check read
as safety while providing none. What replaced it is disk state: an existing
output file is never overwritten without `--force`, in any directory, which is
principle 3 held without a subprocess.

The cost is that a document generated from an uncommitted note is not matched to
a commit. That was only ever true on Nail's own machine, and it was never
checked. `install.sh` is the one exception, and it reads git about this
repository rather than about somebody's documents.

## 2026-08-29. The next version of gdoc is written in Go.

Written in PRINCIPLES.md on the date above and moved here on 2026-09-15. It is
the decision this whole file descends from.

Serves principle 1. Principle 1 drew its line at "a program pip cannot install
does not travel", and pandoc was that program. It survived three specs because
removing it in Python meant writing a Markdown parser and rewriting the AST
walker in the module where the body's pixel fidelity lives. In Go it is an
import, and a Go build is one static binary: no interpreter, no package
manager, no external program that has to already be on the machine. That is
principle 1 satisfied rather than managed, and it is the whole reason for the
decision. Speed is not.

The fidelity question was measured before deciding, because it was the assumed
risk. Six documents were built by both renderers, published to Drive by both,
exported as PDF by Google, rasterised at 300 dpi and compared with a zero
tolerance: 418,385,088 pixels across 48 pages, none different. Run twice on
separate publishes.

It ported exactly for a structural reason worth keeping in mind. Neither
renderer builds a .docx from nothing. Both copy the master and cut into it, so
the cover, the logo, the running head and the coloured tables travel as bytes. A
.docx is a zip of XML, and the body was already written as OOXML by hand against
constants measured out of the template.

**What the decision did not rest on.** The docx *reader* was load-bearing rather
than a fallback: it is the only route that brings a picture out of a Google Doc,
goldmark does not replace it, and it was unmeasured on the day. It is
`internal/docx` now.

## 2026-08-29. gdoc never replaces the body of a document that already exists.

Serves principle 3. Uncertainty never resolves toward the destructive answer, and
here the destruction is silent.

Two ways exist to change a Google Doc gdoc did not create.

**Replace the body.** Export it to `.docx`, edit that, upload it back with
`files.update` and conversion. The document id survives, and this is the only way
to create a suggestion, because a `.docx` carrying Word tracked changes imports as
real accept/reject suggestions.

**Change it in place.** `documents.batchUpdate`. Nothing is replaced.

Measured live on 2026-08-29, not reasoned about:

| | replace the body | in place |
|---|---|---|
| comment anchors | **100% destroyed** | **100% preserved**, 355 of 355 characters across three anchors |
| pending suggestions | destroyed and recreated with new ids | byte-identical ids across 22 batches |
| smart chips | flattened; a date chip loses its date and keeps the string | untouched |
| Google Drawings | survive | untouched |
| footnotes, lists, tables, images | survive | untouched |

The anchor result is the one that decides it. Changing one paragraph destroys the
anchors on paragraphs that were never touched, and **the API reports every one of
them as healthy afterwards**. `comments.list` returns the original `anchor` and
`quotedFileContent` unchanged, because both are stored strings rather than live
pointers. A detached comment does not render in Google Docs, so the threads simply
vanish from the screen while the API insists they are fine. Nail confirmed this by
eye before the mechanism was understood.

So the rule is not a preference. Replacement is unusable on any document a person
has worked in, which is every document worth changing.

**In place is the decision. Replacement is reserved for creating a new document,
where there is nothing to lose.**

### What it costs, stated rather than hidden

Three things cannot be created by any Docs API request, each confirmed with a
verbatim refusal:

- **A first-page header**, which is where the template holds the logo and "For
  internal use only". `HeaderFooterType` is exactly
  `["HEADER_FOOTER_TYPE_UNSPECIFIED", "DEFAULT"]`, and all six header and footer
  id fields answer `Unallowed field` on both `updateDocumentStyle` and
  `updateSectionStyle`. `createPositionedObject` does not exist.
- **A table of contents.** `insertTableOfContents` and every other spelling
  answers `Cannot find field`. The `Request` message has exactly forty members and
  none touches a TOC.
- **A page-number field in a footer.** No request inserts AutoText.

### How the cost is paid

gdoc detects what is missing and **writes a finishing checklist into the document**,
as its own page after the cover, with checkbox bullets and the exact menu path for
each item. The block is marked with a named range so it can be removed in one call.

The general rule this is an instance of:

> **Where gdoc cannot act, it says so in the document, with the exact clicks.
> Never a silent omission.**

That is principle 3 again. A document that quietly lacks its contents list is
"I do not know" resolving to "ship it anyway".

Driving the browser was considered and rejected. It would have worked, and it
would have added a signed-in Chrome to the dependencies of a tool whose first
principle is that it runs where nothing can be installed. The checklist costs the
reader five seconds and cannot break when Google moves a menu.

### Consequences

- `restyle` styles in place and never replaces a body.
- `align` proposes; it does not rebuild the document.
- `publish` may render and upload freely, because the document does not exist yet.
- `gdoc release` refuses to export a PDF while the finishing checklist is still in
  the document. Instructions in the body are only safe if they cannot be sent by
  accident.
- Never call `deleteHeader` on a first-page header. It succeeds, and the header can
  never be recreated: `createHeader` then answers "Default header already exists"
  and `firstPageHeaderId` is gone for good. One-way door, no warning.

### The one thing that changes this decision

`writeControl.writeMode: SUGGEST` on `documents.batchUpdate` would make a
suggestion surgically, with no replacement and none of the damage above. It is
gated on the calling Cloud project being allowlisted for the Google Workspace
Developer Preview Program, and on a project that is not allowlisted it returns
**HTTP 200 and performs a direct edit**, with nothing in the response to say so.

If project `4326046141` is enrolled, suggestions come back and this decision keeps
its shape: still no replacement, and now proposals in the document rather than in
the terminal. Until then, approval happens in the terminal before anything is
written.

Never send `writeMode: SUGGEST` and trust the 200. Probe the capability against a
throwaway document first, once, and cache the answer.

## 2026-08-29. Never trust a success. Verify with a second, independent probe.

Serves principle 3.

Four Google APIs reported success while doing something else, in one day:

1. `writeControl.writeMode: SUGGEST` returns 200 and makes a direct edit.
2. `comments.list` reports detached, invisible comments as healthy and anchored.
3. `files.export?revisionId=N` returns the head and ignores the parameter.
4. `revisions.update keepForever` returns 200 and never persists.

So a status code is not evidence. Where an operation matters, gdoc confirms it by a
different route than the one that performed it. The `.docx` export is the honest
witness where the JSON API is not: comment attachment is only truthfully readable
from `word/comments.xml` and the `commentRangeStart` and `commentRangeEnd` markers.

## 2026-08-29. gdoc marks its own changes by colour, never by highlight.

Domain choice, no principle above it, but it exists only because of the decision
above: gdoc cannot create a real suggestion, so it has to show its work some other
way.

**Insertions are green. Deletions are red and struck through. Nothing is
highlighted.**

| | `foregroundColor` | also |
|---|---|---|
| proposed insertion | `rgb(0.18, 0.60, 0.35)` | nothing. No underline, no background |
| proposed deletion | `rgb(0.80, 0.25, 0.22)` | `strikethrough: true` |

Nail chose this from three shades on
`gdoc PROPOSAL MARKING styles`, judged in real sentences at body size rather than
on a swatch. Two paler variants exist in that document if this pair reads too
loud in a long paragraph.

The reason it is colour and not something else is that **it imitates how Google
Docs renders a real suggestion**. Reviewers already read green-and-underlined as
"proposed" and red-and-struck as "proposed for deletion". gdoc is borrowing a
vocabulary the reader already has, so nothing has to be explained in the document.

### What was rejected, and why

- **Background highlight.** Nail's call. A background colour dominates a page,
  reads as marker pen rather than as a proposal, and survives into a PDF looking
  like a mistake.
- **Emoji reactions.** No API at all. `"reaction"` appears zero times in both the
  Docs and the Drive discovery documents. It is a UI-only feature.
- **An anchored comment on the changed text.** This is what we actually wanted, and
  Drive will not do it: the documented anchor JSON is accepted, echoed back by
  `comments.list` forever, and produces no `commentRangeStart` in the export. An
  anchor can be *reused* from an existing comment, never *minted*.
- **A footnote.** Visible and unobtrusive, but there is no `deleteFootnote`
  request. A marker gdoc cannot remove is a one-way door.
- **A paragraph change bar** (`borderLeft`). Kept in reserve for a whole rewritten
  paragraph, where styling every word would be noise.

### Rules that come with it

- **A named range underneath every proposal**, `gdoc:proposal:<id>` or
  `gdoc:proposal-delete:<id>`. The colour is for the human; the range is how gdoc
  finds its own work again. `createNamedRange` returns only an id, and
  `documents.get` reads the current span back.
- **Never act on a stored index.** A named range is not anchored to text, so its
  span drifts if somebody edits across its boundary. Before accepting or rejecting,
  re-read the range and confirm the content inside it still matches what gdoc
  wrote. Deleting on a remembered index would eat somebody's paragraph.
- **Accepting clears the styling**, leaving text indistinguishable from text gdoc
  never touched. Rejecting deletes the range's content. A proposed deletion is the
  inverse: accepting deletes, rejecting removes the strikethrough.
- **`gdoc release` refuses to export a PDF while any proposal is outstanding.**
  Coloured text in a regulator's copy is the one failure this design can produce,
  so it is made impossible rather than unlikely.
- **The document shows what changed. The terminal says why.** gdoc cannot attach a
  comment to its own edit, so the explanation lives in `gdoc proposals` output,
  not in the file.

### The one thing that changes this decision

Developer Preview access to `writeMode: SUGGEST`. Then gdoc makes real suggestions,
with real accept and reject chips, and all of the above is deleted: no colours, no
named ranges, no stored-index rule, no release gate. This decision is scaffolding
for an allowlist that has not arrived.

### Measured 2026-08-29, after the decision above

The mechanism was tested against a live document before being written down. What
was learned changes three of the rules.

**Named ranges track edits exactly.** Inserting 200 characters before a range moved
it 232 to 432 with the covered text byte-identical; deleting 49 moved it back. Zero
drift. Typing inside expands it, typing at the start shifts it without swallowing
the new text, typing at the end leaves it alone, and a paragraph style change does
not disturb it. So the range is a real anchor, not a hopeful one.

**Drift is lossy, never destructive.** An edit *across* a boundary makes the range
**shrink to a subset** of the proposed words. It never slides onto text gdoc did
not write. So the re-read-and-compare rule catches every case, and the worst
outcome is a proposal gdoc declines to act on. It happened unprompted during the
test run, so it is not theoretical.

**The key is `namedRangeId`, not the name.** Duplicate names are allowed and
coexist, and `deleteNamedRange` **by name deletes all of them at once**. Store the
id.

**Clearing is clean.** `{"textStyle": {}, "fields": "foregroundColor,underline"}`
returns a run to `{}`, byte-identical to one that was never styled. No explicit
black is needed, and accepting leaves no residue to accumulate over months.

**`insertText` inherits the preceding run's style.** Two inserts in the test run
silently became green strikethrough, one of them a heading. **Every insert needs a
style reset after it**, naming every field the proposal styling can set.

**A named range does not survive a docx round trip**, though it survives
`files.copy`. So proposals must be resolved before any export-based step, and
`gdoc release` refusing while proposals are outstanding covers that too.

**The release check is cheap.** `documents.get(fields="namedRanges")` returns 982
bytes against 12,907 for the whole document. One call, filter names by the
`gdoc:proposal` prefix.

**Rejected alternatives, measured rather than reasoned.** An invisible range with
an explanatory comment and no styling **prints as ordinary prose**: the proposed
sentence sits in the PDF with nothing marking it. That is a failure, not a
trade-off. `quotedFileContent` is settable but Google silently coerces its mimeType
from `text/plain` to `text/html`.

A paragraph `borderLeft` **does** render as a margin bar, confirmed in a PDF, at
3PT with padding. Kept in reserve for a wholly rewritten paragraph.

## 2026-08-29. A comment is a prompt. `align` is a separate question.

Domain choice, no principle above it. Nail's model, recorded after he corrected
mine.

**Writing `ai!` in a document does exactly what typing the same instruction to
Claude in a terminal does.** Same agent, same judgement, same effect on the hub,
in the same moment. The comment thread is one more place the agent is spoken to,
not a queue feeding a later command. If the instruction needs a note in the hub
changed or a new file written, that happens then, not at some reconciliation step.

This is why `pending.md` is gone and nothing replaces it. v1 captured work in a
file because a session ended before the work was done. Nothing is captured now
because nothing waits.

**`align` is not the thing that processes comments.** It answers a different
question, asked when Nail wants it answered: *how far has this document drifted
from what the hub now says, and where?* It runs because he ran it.

It exists because the two sides move independently and nothing should keep them in
step on its own. He edits notes in the hub. Decisions land in other files in the
same folder. People edit the document. "How aligned is this with what we now
think" is a judgement about meaning, so it lives in the skill. The binary supplies
both sides and the differences between them, and says nothing about what they mean.

## 2026-08-29. The marker is the trigger. Always.

Serves principle 3.

A thread stays open once it has been addressed, so the agent keeps the
conversation in mind. **But gdoc acts only on a comment carrying an `ai` marker.**
An unmarked follow-up is reported and never executed.

Both halves matter. v1's failure was silence: a second comment in an answered
thread was ignored with nothing said, and an instruction was lost. Reporting fixes
that. The other half fixes a problem v1 never had: without it, anyone replying in
a thread Nail opened would inherit the authority to drive the agent, and
"agreed, but check the custody wording" would be executed as an instruction
against a regulatory document nobody asked to change.

So stickiness carries **context**, never **authority**.

## 2026-08-29. gdoc judges whether to ask, rather than always asking.

Domain choice. Nail's instruction, and it corrects an earlier decision here.

The earlier version said the binary never prompts and Claude puts every choice to
Nail. The first half stands. The second was wrong: it made asking the default,
and a tool that asks permission for everything teaches its user to say yes without
reading.

Instead, each action is weighed for **safety and sensitivity**. Routine and
reversible, gdoc does it. Outward-facing, hard to undo, or touching something
sensitive, it stops and asks. The weighing belongs in the skill and the prompt,
because this is not only a command-line tool: the binary, the skill and the prompt
are one product, and judgement is the part only the agent can supply.

The binary still never prompts. It has no terminal to prompt from and must run
headless.

## 2026-08-29. There is no `release` command, and gdoc does not export PDFs.

Domain choice. Nail's call.

He downloads a PDF from the browser, which he already does. So `gdoc release` is
gone, and with it the rule that refused to export while a finishing checklist or an
unresolved proposal was still in the document. That rule protected the release
command and had nothing else to protect.

The command surface is `publish` and `restyle`, plus the primitives the skills
compose: read the comments, reply, propose, apply, show the differences. The two
named workflows, a review session and an alignment check, are skills rather than
commands, because both are judgement.

## 2026-08-29. Anchors are not banked. *Rejected the day it was proposed.*

It worked. It is not being built.

The idea: a comment anchor cannot be minted, but a document born from a file gdoc
wrote can carry as many as gdoc likes. So publish would seed one anchor per
sentence, `files.copy` would shed the placeholder comments while keeping every
anchor, and gdoc could then attach genuine anchored comments in place for the life
of that document. Measured and confirmed: 1,000 anchors imported intact for 1.8
seconds of publish time, surviving edits, copies and a full round trip.

**Nail rejected it.** Seeding an invisible marker on every sentence of every
document is a large, permanent, load-bearing trick bought for a capability that
only ever works on text gdoc itself wrote. A sentence he types in the browser
could never have one. The complexity is paid on every document forever; the
benefit is partial by construction.

So: **gdoc cannot attach a comment to a specific sentence, and will not pretend
to.** The colour marking stands as written, and the explanation stays in the
terminal rather than in the margin. If the Developer Preview arrives,
`insertComment` does this properly and none of the above is needed.

The measurements are kept in `BLOCKED-BY-API.md` so nobody rediscovers the idea
and proposes it again.


## 2026-08-29. The Developer Preview arrived, and a third of this file retires.

The application went in on the morning of 2026-08-29 and Google granted it the
same day. Cloud project `4326046141`. Measured within the hour, with the ordinary
OAuth token:

| | |
|---|---|
| `writeControl.writeMode: SUGGEST` | **200, and genuine suggestions.** Text absent from `PREVIEW_WITHOUT_SUGGESTIONS`, present with `suggestedInsertionIds` under `SUGGESTIONS_INLINE`, `w:ins` in the export |
| `insertComment` | **200, and genuinely anchored.** The export enclosed exactly the word targeted |
| `commentsViewMode` | 200, comments returned with real character ranges |
| `acceptSuggestion` | 200 |

The `insertComment` payload is undocumented. It is:

```json
{"insertComment": {"range": {"startIndex": N, "endIndex": M}, "content": "..."}}
```

Not `anchor`, not `quotedRange`, not `comment`. Every other shape is
`Cannot find field`.

The answer is undocumented too, and it was measured on 2026-09-07, on a
throwaway document in the test folder, because the first live write test
guessed it wrong and lost the comment id. The `insertComment` reply carries a
`commentThread`, and the id is one level down:

```json
{"replies": [{}, {}, {"insertComment": {"commentThread": {
   "commentId": "AAAC…", "anchorId": "kix.…",
   "headPost": {"postId": "AAAC…", "content": "🤖 …", "author": {"me": true}, "…": "…"},
   "status": "OPEN", "plainTextQuote": "the inserted words"}}}],
 "suggestionResponses": [{"createdSuggestionIds": ["suggest.…"]},
                         {"updatedSummarySuggestionIds": ["suggest.…"]},
                         {"updatedSummarySuggestionIds": ["suggest.…"]}],
 "commentUpdateState": "ALL_SAVED", "writeControl": {"requiredRevisionId": "…"}}
```

`go/internal/propose/testdata/batch-saved-measured.json` is that answer with
the ids replaced. `propose` reads `commentThread.commentId` first and the flat
`insertComment.commentId` it had guessed as a fallback.

The day's behaviour is worth recording, because it explains three contradictory
readings: that morning `SUGGEST` returned **200 and silently made a direct edit**;
by midday it returned **400 Unsupported WriteControl mode**; by evening it worked.
An enrolment landing in stages. Nothing in the response ever said which state it
was in, which is "never trust a success" earning its place twice over.

### What this deletes

Colour marking of proposals. The named ranges that tracked them. The
never-act-on-a-stored-index rule. Explaining changes in the terminal instead of
the document. Anchor banking, already rejected. The Commenter-role idea. The
Chrome extension question. All of it existed to work around a gate that is open.

### What it does not touch

**Layout.** Re-probed with the preview active: `insertTableOfContents`,
`refreshTableOfContents`, `createPositionedObject`, `insertAutoText` and
`insertPageNumber` are all `Cannot find field`; `createHeader` still takes only
`DEFAULT`; `firstPageHeaderId` is still `Unallowed field`. So the finishing
checklist survives, cut to three items: the first-page header with the logo, the
contents list, the footer page numbers. All free when a document is born from the
template, all impossible on one that already exists.

### The cost nobody expected: attribution

**Authorship cannot be set.** Every field refused: `writeControl.author`,
`writeControl.suggestionAuthor`, a top-level `author`, `insertComment.author`,
`insertComment.authorDisplayName`. Everything gdoc writes is authored **Nail
Khusnullin**, in the UI and in the export.

The retired docx route *did* carry a custom `w:author`. So on attribution the
preview is **worse** than the mechanism it replaces. It is still the right choice,
because that route replaced the body and destroyed the review, but the cost is
real and it is paid every time.

The only remedy is the text itself: **a comment gdoc writes opens with `gdoc:`**,
and a suggestion is explained by a comment that does. Nail's call, and the only
option available.

*Superseded the same day: the marker is 🤖, not a `gdoc:` prefix. See
"Everything gdoc writes opens with 🤖."*

## 2026-08-29. gdoc proposes. It never accepts and never rejects.

Nail's rule. `acceptSuggestion` and `rejectSuggestion` work and will not be used.
Resolving a proposal is his hand, in the document, and is not delegated.

*Narrowed 2026-09-07, Nail's call: `rejectSuggestion` is used on gdoc's own
proposals, and on nothing else, because it is the only request that withdraws a
replace proposal whole. The guard carries it for one granted id per run. See
"A replace proposal cannot be fully withdrawn" below.*

This raised a question and answered it. An earlier decision says the hub markdown
changes when a proposal is **accepted**. If gdoc never accepts, how does it know?

**By reading.** An accepted suggestion leaves the pending list and its text becomes
ordinary text; a rejected one leaves and the text does not. So `align` reads the
document, sees which proposals were resolved and how, and offers the accepted ones
to the markdown. gdoc learns the decision rather than making it.

Measured 2026-08-29 that this is safe to build on: twelve pending suggestions,
twelve distinct ids, and resolving three left the other nine byte-identical and
still pending. Ids are stable, so gdoc can list what is outstanding, quote what
each proposes, and notice what changed since it last looked.

It also survives the awkward case: if a suggestion is **edited** before being
accepted, its id tells gdoc nothing, but the document's text does, and that is what
`align` reconciles from anyway.

## 2026-08-29. MCP does not replace the REST client. The binary stays a REST client.

Assessed against Google's own documentation rather than guessed at.

The Docs MCP server is Google-hosted at `https://docsmcp.googleapis.com/mcp/v1`,
itself in Developer Preview, and exposes **exactly two tools**: `read_doc` and
`update_doc`. `update_doc` is a pass-through to `documents.batchUpdate`.

Four reasons it cannot be gdoc's transport:

- **No Drive.** gdoc creates documents, copies the template, exports, reads
  folders and checks write access. None of that exists in two tools. There is a
  separate Drive MCP server, also in preview, which is one more dependency rather
  than a simplification.
- **It is built for AI hosts, not for a CLI.** The documented redirect URIs are
  `claude.ai` and `antigravity.google`. A Go binary speaking MCP over HTTP with
  OAuth is *more* work than calling REST, not less.
- **The guard could not exist.** gdoc's safety property is that every request
  passes through a transport that refuses any file it was not given. Talking to
  Google's server directly leaves nothing to wrap and no way to bound what is
  reached. That is principle 3, not a preference.
- **It is preview too.** No stability is gained.

Where MCP is genuinely useful is outside gdoc: Nail can connect Claude Code to it
for ad-hoc work on documents gdoc has nothing to do with. His enrolment also lit up
the Drive, Gmail, Calendar, Sheets, Slides, Chat, People and Universal Search MCP
servers. None changes this design.

## 2026-08-29. Tabs are out of scope, and gdoc defends against them.

Amended after review: the earlier title read as an intention to support tabs.
There is none. **gdoc has no tab features.** It creates none, styles none, and
addresses none, and this does not change if Google pushes tabs harder.

The entry stays because tabs are **GA**, not preview, so a colleague can put one
in a document gdoc is pointed at, and one of Nail's own test documents already
had one. Both failure modes are silent, on somebody else's document.

Google's wording, verbatim:

> When `includeTabsContent` is not set, "the text fields in the Document Resource
> (e.g. `document.body`) will be populated with content **from the first tab
> only**. The `document.tabs` field will be empty and content from other tabs won't
> be returned."

And for writes: a request without a `tabId` "will in most cases be applied to the
first tab" — **except** `ReplaceAllTextRequest`, `DeleteNamedRangeRequest` and
`ReplaceNamedRangeContentRequest`, which **apply to every tab**.

Two defensive rules, and they are defences rather than features:

- **Always read with `includeTabsContent=true`.** Reading `document.body` without
  it sees one tab and reports the rest as absent, silently. gdoc must never be
  blind to content that exists. `commentsViewMode` requires it anyway.
- **If a document gdoc is asked to write to has more than one tab, gdoc stops and
  reports that tabs are unsupported.** It never guesses which tab was meant and
  never writes anyway. Not knowing must not resolve to "the first one, probably".

## 2026-08-29. Amendments from reading the documentation.

Recorded because these were found by **reading**, after several were missed by
probing. Nail's instruction, and he was right.

**`insertComment` takes an assignee.** The documented shape is:

```json
"insertComment": {
  "content": "...",
  "assigneeEmailAddress": "user@example.com",
  "range": {"startIndex": N, "endIndex": M}
}
```

Probing found `range` and `content` and would never have found
`assigneeEmailAddress`. It means gdoc can raise a question **and assign it to the
person who should answer it**, which is a better shape than a comment nobody owns.

**`addCommentReply` is nested, which is why the probe failed.**

```json
"addCommentReply": {
  "commentId": "...",
  "post": {
    "content": "...",
    "commentAction": "RESOLVE" | "REOPEN",
    "assigneeEmail": "..."
  }
}
```

Note `commentAction`. **gdoc can now resolve and reopen threads.** v1's rule that
it must never resolve a comment was enforced by an API that had no way to. It is
now a choice, and it stays: **resolving means the answer was accepted, and only
Nail accepts.** Same reasoning as never accepting a suggestion.

**Also documented, and confirming the probes:** authorship is not settable
anywhere; `commentsViewMode: COMMENTS_VIEW_MODE_INCLUDED` requires
`includeTabsContent=true`; and comment or suggestion writes can **partially fail**,
committing the text while losing the comment, reported in `commentUpdateState`
as `NO_UPDATES_REQUESTED`, `ALL_SAVED` or `ALL_FAILED_UNKNOWN_REASON`. **Check that
field, not the status code.**

### The rule this whole entry exists to state

**Read the documentation before probing.** Probing finds what you thought to ask
for. It found `range` and `content` and missed the assignee, missed the nesting on
replies, and would have missed `commentUpdateState` entirely, which is a silent
data-loss path. Probing is for confirming what the documentation claims, and for
the cases where the documentation is wrong, which today it was three times.

## 2026-08-29. gdoc can retract its own proposal, and must check that it landed.

Two findings from the first end-to-end run of the whole design, on a copy of the
ideal test document.

**A suggestion can be withdrawn.** A `deleteContentRange` issued in `SUGGEST` mode
over gdoc's own pending insertion removes it cleanly and comes back with
`deletedSuggestionIds`. So a proposal gdoc got wrong can be taken back without
touching the document and without asking Nail to reject it. This matters more than
it sounds: it is the difference between a tool that can correct itself and one whose
mistakes are the reader's problem.

It does not weaken the rule that gdoc never accepts or rejects. Withdrawing its own
unaccepted proposal is not resolving Nail's decision; there was no decision yet.

**An insertion point is off by one until proved otherwise.** The first attempt of
that run put text one character before a full stop, producing
`…this line Keep one idea in each item..`, and `batchUpdate` returned **200**. The
error was visible only in the rendered PDF. Index arithmetic is the hazard of this
whole API, not the requests.

So: after inserting a suggestion, **read back the span and confirm it reads as
intended** before telling anyone it worked. The same "never trust a success" rule,
now with a third instance behind it.

### What the run proved, which is the whole design in one document

Style applied in place, and every one of ten features survived: person chip, date
chip with its timestamp **and** locale **and** format, calendar rich link, the
Google Drawing, three footnotes, both lists, the table with its pinned shaded
header, the inline image byte-identical, a hand-made anchored comment still
enclosing exactly its original words, and a hand-made pending suggestion still
pending.

The date chip is the sharpest evidence. Every docx route flattened it to the string
"13 Aug 2026" with no date behind it. In place, untouched.

The source document was verified byte-identical, same revisionId, before and after.
Only the copy was written to, per Nail's rule.

## 2026-08-29. The house style is a config file. The .docx becomes provenance.

Nail's decision, and the reason he gave for it is the whole argument.

`house.yaml` is the house style: 1,110 lines, 30,866 bytes, including the logo as
base64. It holds page geometry, all nine named styles, the cover, the header and
footer, three tables cell by cell, the contents field, heading numbering and the
logo. A generator reads it plus markdown and emits a .docx. **Nothing reads the
master at runtime.**

Measured 2026-08-29 against a document built the old way, by copying the template:
160 items compared, **137 identical**. Of the 23 differences, 21 are the same value
stated two ways, because the template declares a heading colour and then overrides
it on every paragraph while the config states the effective one. Two are real: a
0.001pt rounding on the logo offset, and a page count caused by a stale contents
list. On the rendered PDFs the **worst position offset anywhere is 2.5pt**.

The logo comes out **positioned, not inline**: a `wp:anchor` in the header part
behind `<w:titlePg/>` survives Google's import as a real floating object at the
template's exact offset. The Docs API cannot create one; the docx can, so nothing
needs the API to.

### Why not the hybrid, which was the earlier recommendation

Because v1 already ran that experiment and we know how it ends.

v1's template was a .docx. The style improvements Nail made afterwards, table
changes and paragraph spacing among them, **could not go into a Word file**, so
they went into code. That is what `render/shell.py` is: 574 lines, roughly forty
accumulated decisions, three to five weeks to reproduce. **The code became the real
template and the .docx stopped describing the output.**

The hybrid rebuilds that trap the first time an improvement touches a table, which
is exactly where Nail's changes went.

### The reason that actually decided it

Nail does not make these changes by hand. In his words: *"I didn't do the changes
in the template by myself because I just didn't know what exactly I need to
improve. I asked AI to analyse and understand better and give me advice."*

If the house style improves by **an agent reading it, reasoning about it and
proposing a change**, then it has to be something an agent can read and change, and
a change has to be something Nail can review. `space_after_pt: 6 → 8` is a line in
a diff he can judge. The same change buried in a surgery function is not. An agent
cannot meaningfully edit a .docx; it writes code that pokes at the XML, which is
v1 again.

### The risk, and the thing that answers it

The house look lives in details nobody would think to write down: which blank
paragraph is 18pt and which 29pt, that two of three tables have zero cell padding
while the third has 5pt, that Word insets a table by its first cell's left margin.
Each was invisible until measured.

So **the 160-item comparison stays as a test.** It renders from the config, renders
from the master, and fails when any value drifts. That test is what makes this
decision safe rather than brave.

The master `.docx` stays in the repo as provenance and as that test's fixture.
Nothing else reads it.

## 2026-08-29. The finishing checklist is for `restyle` only.

Clarified because it was easy to misread, and Nail did.

The Developer Preview solved comments and suggestions. **It did not touch layout.**
Re-probed with the preview active: no request creates a table of contents, a
first-page header or a page-number field, and `createPositionedObject` does not
exist.

But those three are **free on a document gdoc creates**, because they come from the
.docx it writes: a `wp:anchor` behind `<w:titlePg/>` gives a positioned logo in a
first-page header, a Word TOC field imports as a live refreshable contents list,
and a PAGE field gives footer numbers. All three verified.

| | born from gdoc's docx | already exists |
|---|---|---|
| first-page header with positioned logo | **free** | no |
| live contents list | **free** | no |
| footer page numbers | **free** | no |

So `publish` needs no checklist at all. **Only `restyle` does**, when a colleague's
document arrives and gdoc styles it in place rather than replacing it, which is the
one case where the document was not born from gdoc's own file.

One caveat that misled once already: a document that happens to already carry an
empty first-page header segment **can** have a logo written into it with
`insertInlineImage` at that `segmentId`. That is not a general capability. It works
only where the segment already exists, and gdoc cannot create one.

## 2026-08-29. `publish` builds a file. `restyle` has two modes, and defaults to the careful one.

Nail's reading, confirmed against everything measured today.

### `publish` renders a .docx and uploads it

Not a hundred API calls. The document does not exist yet, so there is nothing to
protect, and the file gives what the API refuses:

- a **positioned logo** in a real first-page header, from a `wp:anchor` behind
  `<w:titlePg/>`
- a **live, refreshable contents list**, from a Word TOC field
- **footer page numbers**, from a PAGE field

All three verified. There is no reason to build a new document any other way.

### `restyle` has two modes and they are a genuine trade

*Superseded 2026-09-11: the `--new` column is not built. See that entry.*

| | **in place** (default) | **as a new document** (`--new`) |
|---|---|---|
| how | `batchUpdate` on the original | render its content to a .docx, upload as new |
| house style | approximate | **exact** |
| comment threads, with real authors and anchors | **kept** | lost |
| pending suggestions | **kept** | lost |
| **smart chips** | **kept, with their data** | **flattened** |
| Google Drawings | kept | kept |
| the URL | **unchanged** | new link, old one still live |

The trade is sharper than "layout against review". Going to a new document costs
the **chips** as well, because the content has to pass through a file to get there,
and that is where a date chip becomes the string "13 Aug 2026" with no date behind
it.

### In place is the default

`restyle` exists for a colleague's document that already has comments on it. That
is exactly when in-place is worth most and a new document costs most. The three
things in-place cannot do are three clicks on the finishing checklist, once, on a
document Nail will have open anyway.

### The exception gdoc can detect

A fresh draft with **no comments, no suggestions and no smart chips** has nothing
to protect. Rebuilding it costs nothing and gives the exact house style.

So gdoc checks before it styles, and when there is nothing to lose it **says so and
offers the better route**. It does not take it silently: a new document means a new
URL, and that is the kind of thing a person should choose. Uncertainty resolves to
asking, and certainty about a better option resolves to offering it.

## 2026-08-29. A comment is answered with the hub open, not against the document alone.

Nail's emphasis, added because the earlier entry understated it.

"A comment is a prompt" says how the instruction arrives. This says what the agent
has in its hands when it acts on one.

**The same context a terminal session would have.** The markdown the document was
generated from, and the folder around it: the notes, the decision log, the meeting
records, the raw material. The hub exists to be the context an agent reasons over,
and a comment in a Google Doc is simply another place Nail speaks to that agent.

So:

- An `ai?` question is answered **from what he actually knows**, not from the words
  on the page. "Why did we take this position" is answered out of the decision log,
  not paraphrased back out of the paragraph it was asked about.
- An `ai!` instruction is carried out **against his material**, not against the
  document in isolation.
- If carrying it out needs a note in the hub changed or a new file written, that
  happens in the same moment, exactly as it would in a terminal session.

The rule underneath: **there is no reduced mode.** Working through comments is not
a lesser kind of session with only the document loaded. It is the ordinary agent,
with the ordinary context, reached through a different door.

## 2026-08-29. Emoji reactions: still no API, preview or not.

Re-probed after the Developer Preview was granted, because the earlier finding
predated it and should not have been assumed to hold.

`insertReaction`, `addReaction`, `createReaction`, `insertEmoji`,
`addEmojiReaction`, `createEmojiReaction`, `insertEmojiReaction`, `reactToRange`
and `addTextReaction` all return `Cannot find field`. The word **`reaction`
appears zero times** in the entire Docs discovery document. The only `emoji` in the
API is `iconEmoji`, which sets the icon on a document **tab** and has nothing to do
with reacting to text.

Emoji as ordinary characters work everywhere, in the body and inside a comment,
because they are just Unicode. So gdoc can write `gdoc: ✅ applied`. It cannot
attach a reaction to a span.

Nothing in the design wanted this. Recorded so it is not probed a third time.

## 2026-08-29. The colour scheme is retired, not forgotten.

Marking proposals in green and explaining them in the terminal was decided earlier
on **the same day**, and it is now withdrawn. It existed for one reason: at the
time gdoc could not create a suggestion or attach a comment to text, so it had to
signal a change some other way.

The Developer Preview removed that constraint. So:

- **no green, no red, no strikethrough** as a marking convention
- **no named ranges** tracking gdoc's own proposals, and no stored-index rule
- **no explanation stranded in a terminal** where the people reading the document
  cannot see it

Changes arrive as **native Google suggestions**, with the accept and reject chips
colleagues already understand, and the reason arrives as a **real comment anchored
to the exact words**.

Recorded as a withdrawal rather than deleted, because the reasoning was sound for
the constraint that existed, and someone reading this in six months should be able
to tell a retired workaround from an oversight. If the preview is ever withdrawn,
this is the fallback and it is written down.

## 2026-08-29. Alignment runs both ways, and it is not a command.

Nail's correction. The earlier entry said `align` asks how far the document has
drifted from the hub. That is half of it.

**Both sides move, and the second direction is the dangerous one.**

- The **document** drifts when somebody edits it in the browser. Visible, and the
  kind of drift anyone would think to look for.
- The **hub** moves ahead when a decision lands in the folder, a meeting note
  changes a position, or the source markdown is edited in another session. The
  document is then quietly **wrong** while looking untouched, and nobody is
  prompted to check.

So alignment reports in both directions, and it exists for the next iteration
rather than as a tidying step.

### It is not a command-line operation

The binary can say what each side contains and where the two differ. It cannot say
whether a difference **matters**, and it must not pretend to: "this document no
longer reflects the position we took in the decision log" is a judgement about
meaning.

So alignment lives in the **skill**. The binary supplies both sides and the
differences. This is the same split as everywhere else in the design: facts below,
judgement above.

### And it never deletes

Something present only in the hub is **work not yet published**, not drift to be
removed. Alignment offers what the document has that the notes lack, reports what
the notes have that the document lacks, and reports a conflict where both moved.
It writes to the markdown only with agreement, and never removes from it.

## 2026-08-29. The product is three things, and it matters which does what.

Recorded because it is the shape everything else hangs on, and it was implicit
rather than written.

- **The binary** deals in facts. What is in the document, what is in the hub, where
  they differ, what was changed. It never decides what any of it means. It never
  prompts. It emits structured output and exits.
- **The skill** is the judgement. What a comment is asking for. Whether a
  difference matters. What to propose. Whether an action is safe enough to take
  without asking.
- **A comment in the document** is another door into that same agent, alongside the
  terminal. Not a lesser mode, not a queue: the same agent, the same context, the
  same behaviour.

This is why the binary is opaque and why it never prompts, and it is why "a comment
is a prompt" is true rather than a slogan. It is also the answer whenever something
new is proposed: **facts belong below, judgement belongs above**, and anything that
needs to weigh meaning does not go in the binary.

## 2026-08-29. Everything gdoc writes opens with 🤖.

Nail chose it from six candidates seen side by side in a real comment sidebar:
🤖 robot, ✨ sparkle, 🪄 wand, ✍️ pencil, a plain `gdoc:` prefix, and the robot with
the name spelled out.

**The rule: every comment gdoc leaves begins with the robot, and nothing else.** No
name, no colon, no prefix text.

### Why a marker is needed at all

Authorship cannot be set. Measured 2026-08-29: `writeControl.author`,
`writeControl.suggestionAuthor`, a top-level `author`, `insertComment.author` and
`insertComment.authorDisplayName` are all `Cannot find field`. Every comment and
every suggestion gdoc makes is signed **Nail Khusnullin**, with his photo, in the
UI and in the export.

So the text is the only place the truth can go.

### Why the robot rather than the others

The marker has **one** job, and it is narrower than "this is AI". The comment
already carries Nail's name and face. The job is to say **this was not written by a
person**.

- 🤖 says exactly that and nothing more.
- ✨ says "AI-assisted", which is a softer and different claim, and it is Google's
  own Gemini mark, so it would read as a Google feature rather than as Nail's tool.
- A `gdoc:` prefix is more text in a column already full of text. It does not carry
  at a glance.
- The robot plus the name was the safe option, and was rejected as noise: the name
  repeats in every thread and adds nothing once a reader has learned the symbol.

Emoji survive the API as ordinary Unicode, in the document body and inside a
comment, both verified.

## 2026-08-29. gdoc reads back what it proposed, before saying it worked.

Kept, narrowed, and settled by Nail: "I don't care at all how many calls you need."
Google's write quota is 60 a minute per user and nothing in this design comes near
it, so the extra round trip is free in practice.

**The rule: after gdoc makes a proposal, it reads the document back and confirms
the change arrived as a suggestion and not as an edit, before reporting success.**

This is narrower than the "never trust a success" version it replaces. Checking
`commentUpdateState` rather than the HTTP status, and similar care elsewhere, is
ordinary correct code and does not need a decision record. This one does, because
it is a **choice that costs something** and a reasonable implementer would skip it:
the API returns 200 and gives no hint that anything is wrong.

The reason it is not optional, measured on the morning of 2026-08-29: that exact
call returned **200 and silently rewrote a reviewed document** instead of proposing
anything. Without the read-back gdoc would have reported a proposal while having
edited somebody's document, and nobody would have known until they read it.

The detector is cheap and unambiguous: a genuine suggestion is **absent** from the
document read with `PREVIEW_WITHOUT_SUGGESTIONS`; a direct edit is present.

The four incidents that produced this rule are recorded as facts about Google's
behaviour in `BLOCKED-BY-API.md`, not here.

## 2026-08-29. The preview may vanish, and the risk is accepted. Loudly.

Nail's call, on review. If Google withdraws the Developer Preview, gdoc loses
proposing in the document: suggestions and anchored comments. Replies in existing
threads survive, because they are plain Drive API. That loss is accepted.

What is not built: fallback code. The colour scheme stays on paper, in the
withdrawal entry above, and is rebuilt only if the day comes. Maintaining a live
second path for a retired design is the cost that was refused.

What is not negotiable: the loss must be loud. The capability probe against a
throwaway document and the read-back after every proposal stay, because an
unenrolled project answers 200 while silently direct-editing. The feature may
vanish. It must never vanish as a silent edit to a reviewed document.

## 2026-08-29. The guard owns the transport, and every id carries a write level.

Serves principle 3, and carries the v1 guard into Go stronger than it was.

In v1 the guard wrapped a client library's transport. v2 has no client library,
and the whole preview surface is hand-rolled JSON, which is exactly the kind of
request that walks around a bolted-on wrapper. So the guard is not a wrapper any
more: **one package owns the network**, nothing else in the binary can make an
HTTP request, and a test fails if HTTP construction appears anywhere else.

The set keeps v1's two doors exactly: ids handed in on the command line, ids
learned from a create the guard itself carried. New in v2, each id carries a
write level:

- an id learned from a create is fully writable, because gdoc made it
- an id handed in is **read-and-suggest only**, never direct-editable
- `restyle` in place is the one exception, granted explicitly per run, never
  inherited

A rule the skill follows is advisory. A rule in the transport is physics. The
morning incident, a 200 that direct-edited a reviewed document, becomes locally
impossible to cause and the read-back catches the remote half.

## 2026-08-29. Publish runs once. Everything after travels as suggestions.

Colleagues keep one URL for the life of a document, and gdoc never replaces an
existing body. Together those close the question of how a big hub update lands:
there is **no republish mechanism**, in any form.

Scale is handled by judgement, in the skill, like everything else:

- small drift arrives as inline suggestions
- a structural rewrite arrives as **one suggestion per affected section**, each
  with a 🤖 comment saying what changed and why, so accepting is one chip per
  section rather than a wall of green
- when a document's review has run its course, the escape hatch already exists:
  `restyle --new` builds a fresh document from the hub, exact style, new URL,
  said out loud and chosen by a person

One more line from the same review: the prompts behind the skills, alignment
above all, are part of the product. They live in the repo and a change to one is
reviewed like code, because a wrong alignment judgement writes a wrong suggestion
into a document colleagues read.

## 2026-08-29. Live review is a session, not a service.

Nail's requirement, added after the spec was first written: once a review is
launched, the conversation moves into the document and stays there. He writes
`ai!` in a comment and the suggestion and the 🤖 reply arrive within seconds,
for as long as the session runs. Review is live until he stops it.

What does not change: nothing runs on its own. The live loop exists only while
a session Nail launched is running. Closing the terminal ends it. The 2026-08-17
standing watcher stays dead.

**The mechanism is session-scoped polling.** The skill loops; every 5 to 15
seconds the binary makes one cheap call, "any comment activity since this
cursor", and exits. The binary stays one-shot and gains a `--since` cursor.
Liveness is the skill's, which is where judgement already lives.

Two alternatives were reviewed on Nail's ask, and both lose mechanically:

- **Apps Script** cannot push to a machine with no public endpoint, so the poll
  never goes away; it would only add a component in front of it. And its own
  timers bottom out at one minute, six times slower than the poll it would
  replace. Plus the costs already priced in BLOCKED-BY-API.md: a second
  execution surface that sees less of the API than the binary does.
- **A Chrome extension** can push for real, via native messaging, but only
  while the document is open in a browser, only until Google moves the DOM, and
  only by putting a signed-in Chrome inside the trust boundary of a tool whose
  first principle is one file you copy. Driving the browser was already
  rejected once, for the checklist, on the same grounds.

**A colleague's `ai!` acts.** Nail's call, made knowing what it widens: during
a live session, anyone who can comment on the document can drive the agent,
without Nail between them. The marker stays the trigger and identity stays not
a gate, consistent with everything above. The guard caps the blast radius: on a
handed-in document the agent can only suggest and reply, so the worst a
colleague's instruction can produce is a suggestion Nail rejects. A colleague
who can already edit the whole document was always trusted with more than this.

## 2026-09-07. A replace proposal cannot be fully withdrawn by a delete. Decided: reject gdoc's own.

Found by the first live write test, on throwaway documents in the test folder,
after M3's revmux review had passed. The 2026-08-29 finding that a
`deleteContentRange` in `SUGGEST` mode "removes it cleanly and comes back with
`deletedSuggestionIds`" was measured on a **pure insertion**. A `propose` is a
**replace**: one suggestion id covering a suggested deletion of the quoted words
and a suggested insertion of the replacement. Five things were measured:

| Sent, on gdoc's own replace proposal | What Docs did |
|---|---|
| `deleteContentRange` in `SUGGEST` over the insertion half (what `withdraw` sends) | 200. The inserted words are gone. The answer carries `suggestionResponses[].updatedSummarySuggestionIds: [id]` and **no `deletedSuggestionIds`**. The read-back still carries the quoted words as a suggested deletion under the same id. **Half the proposal stays pending.** |
| a second `deleteContentRange` in `SUGGEST` over the still-suggested-deleted words | 200, `updatedSummarySuggestionIds` again, and nothing changed. |
| one `deleteContentRange` in `SUGGEST` over both halves at once | 200. The inserted words become suggested-inserted **and** suggested-deleted under the same id. Worse than before. |
| `rejectSuggestion {suggestionId}`, in `SUGGEST` mode and plain | **Refused by the guard before it left the machine**: any request kind whose name carries "suggestion" is refused, which is the 2026-08-29 rule "gdoc never accepts and never rejects" in code. Not measured against Google. |
| the 🤖 comment, after the insertion half is deleted | Survives, `deleted: false`, still anchored by id, quoting words that are no longer in the document. The skill can still reply into it. |

So today `withdraw` reports `verified: false` with two warnings on every
replace proposal, leaves the entry in the note, and the document keeps a
suggested deletion gdoc cannot take back. The live write test fails on exactly
this, and it is left failing on purpose.

The one route that would retract the whole suggestion is `rejectSuggestion` on
gdoc's own id, and the guard refuses it by Nail's rule. Widening that rule to
"gdoc may reject a suggestion the note records as its own" is his decision, not
a refactor: the guard cannot tell whose a suggestion is, so the door would have
to be a per-run grant the `withdraw` command seeds from `proposals[]`, the way
`AllowCreateIn` names one folder. The alternatives are to leave `withdraw` as a
half-retraction that says so, or to make withdrawing a thing Nail does by hand.

**Decided the same day. Nail: "allow reject on own ids; merge after the fix."**
Measured before it was built, through the new door and not around the guard:
`rejectSuggestion {suggestionId}` inside a `writeMode: SUGGEST` batch returns
200, the document reads exactly as it did before the proposal, the answer
carries `suggestionResponses[].rejectedSuggestionIds: [id]` with
`commentUpdateState: ALL_SAVED`, and the 🤖 comment survives, `deleted: false`,
still anchored by id. The same request without `writeMode` behaves the same, so
the guard's SUGGEST requirement on a handed-in document stays as it was.

What changed: `Policy.AllowReject(id)` is the per-run grant, seeded by
`cmdWithdraw` from the note's `proposals[]` after `Mine` has answered; the guard
carries a `rejectSuggestion` only when it is spelled exactly, carries exactly
`{"suggestionId": <that id>}`, and the id is the granted one, while
`acceptSuggestion`, `deleteSuggestion`, any other id and any second field stay
refused; `withdraw` sends that one request, checks that the document still
carries the id on either side before it writes, and is verified when
`rejectedSuggestionIds` names the id and a fresh read carries no run under it.
A document the old delete half-retracted is repaired by running `withdraw`
again, because a run still suggested-deleted under the id counts as pending.
The multi-tab refusal went with the range. The live write test passes end to
end. The rule "gdoc never accepts, rejects or deletes anyone else's suggestion"
is unchanged and is now guard-enforced rather than a family ban.

## 2026-09-08. v1's `gdoc: <id>` stays refused. Publish is the only writer of the block.

v1's `gdoc generate` writes the pairing as a plain string, `gdoc: 1AbC...`. v2's
front matter is a block, `gdoc: {schema: 1, document_id, folder_id, ...}`, read
under `yaml.Strict()`. So until M6 a note v1 published could not be handed to
v2's `propose --md` or `withdraw`: the reader refused the shape, and the run had
no provenance record, which is the permission to withdraw later. Reading,
answering and proposing on the link alone always worked; only the note pairing
was lost.

Two ways out were on the table. The reader could accept the v1 string as a
schema-0 shape and upgrade it on the first write, which keeps notes published
that week usable with no hand edit. Or M6's publish becomes the only writer of
the block and v1's `generate` is retired for v2 notes.

**Decided: the second.** No schema-0 shape enters the reader. A note v1
published gets its `gdoc:` line rewritten by hand once, or the line is taken out
and the note is published again by v2.

The reason is what the first option costs for what it buys. It buys one hand
edit on the handful of notes published in one week. It costs a second shape in
the one reader that decides whether gdoc is allowed to act on a note, for ever,
plus an upgrade-on-write path that rewrites somebody's front matter from a shape
gdoc only half understands. `frontmatter`'s whole rule is that a block gdoc half
understands is a pairing it may act on wrongly, and a schema-0 branch is that
rule with an exception in it. There is also no folder in a v1 string, so an
upgraded block would carry a `document_id` and no `folder_id`, which is a
pairing that reads and is not the one publish writes.

What that puts on the refusal is the whole migration story, because there is no
other one. So it is not the parser's own sentence. A `gdoc:` key holding a plain
string comes back naming the shape it found, the id inside it, and the two ways
out: write it as a block carrying `schema: 1` and that `document_id`, or take
the line out and publish again. `v1Pairing` and `v1Refusal` in
`internal/frontmatter/frontmatter.go` are where that lives, and they are asked
only after the strict read has already refused, so a block that reads pays
nothing. A key holding some other scalar, `gdoc: 3`, is not v1's pairing and
does not get v1's sentence: sending somebody to rewrite a number as a document
id is the wrong way.

`docs/backlog/v1-frontmatter-migration.md` is deleted by this entry. It asked
the question and said Nail decides in M6.

## 2026-09-09. The guard carries `files.copy`, for one source, and it is a decision rather than a rule satisfied.

Serves principle 3, and stretches it further than anything before it. Nail's
decision, recorded here because M2's rule does not cover it.

M2's rule is that a guard door with no **production** caller is deleted rather
than carried. That is why `GrantInPlace` went, and why it came back at M7b
beside the line that calls it. `Policy.AllowCopy` has no production caller and
is not going to get one: its only caller is `TestLiveRestylePreservesTenFeatures`,
the ten-feature preservation run. So the rule says delete it, and the decision is
to keep it anyway.

**What buys it is the measurement it makes possible.** The acceptance for an
in-place restyle needs a document holding all ten features, an anchored comment,
a pending suggestion, a smart chip, an image and a Google Drawing among them.
A throwaway probe showed on the same day that the first several can be built
from scratch through the API, and that the image and the Drawing cannot. It was
`tools/copyprobe`, deleted at M7b once it had answered, and what it measured is
in BLOCKED-BY-API.md. Without
a copy route the ideal document is made by hand in a browser before every run,
which is a test nobody runs, which is an acceptance that does not exist.
`files.copy?copyComments=true` carries the threads still anchored and the
pending suggestions with them, measured, so the run is unattended.

**What it costs is worth naming rather than discovering.** A copy is the widest
reach a handed-in id has ever produced. Every other route out of `LevelSuggest`
reads: the export hands back the bytes, and that is all. A copy takes a full
duplicate of somebody's document, comments and pending suggestions included,
into gdoc's own folder, where `learnFromCreate` puts it at `LevelFull`, which is
trash-and-rename. Nothing about the source changes and nothing about the source
becomes more reachable, so principle 3's claim still holds as written. It is
still the largest thing a handed-in id has ever produced, and a reader should
meet that as a decision somebody took rather than as a door somebody opened.

Four rules hold it down, and each is pinned by a test in
`go/internal/guard/copy_test.go`:

- **The grant is one source and one run.** `AllowCopy(id)` in the shape
  `AllowReject` and `AllowCreateIn` already have. A second call replaces the
  first, nothing writes it to disk, and no flag turns it on for every document.
- **It admits no id.** A copy of a file nobody handed in is refused by the
  file check, before the grant is read. Two doors, still.
- **The copy lands in the one folder the run named.** A `files.copy` that omits
  `parents` lands in the **source's** own parent, a folder gdoc was never given.
  So the copy is a create in the transport's grammar too: `filesCopy` is
  `filesCollection`'s rule for the second create shape, `isCreate` reads it, and
  the same one-parent check and the same id learning run on it. A path only one
  of the two called a create would carry an unparented duplicate and learn
  nothing from the answer.
- **Three parameters.** `copyComments`, `supportsAllDrives` and `fields`.
  `ocr`, `keepRevisionForever`, `ignoreDefaultVisibility`, `enforceSingleParent`
  and `includePermissionsForView` are refused with everything else the allowlist
  does not name.

## 2026-09-09. Seven paragraph elements were dropped at decode. The fixture is built from the reference, and the live check is outstanding.

Serves principle 3, at one remove: a reader that drops what it does not
recognise makes every decision downstream on a document that is not the one on
screen.

`ParagraphElement` is a union of eleven members. `run()` in
`go/internal/docs/walk.go` reads four of them, and its `default` arm returns
`(Run{}, false)`, so the other seven vanish at decode:

| Member | Carries |
|---|---|
| `person` | `personId`, `personProperties.name`, `.email` |
| `richLink` | `richLinkId`, `richLinkProperties.title`, `.uri`, `.mimeType` |
| `dateElement` | `dateId`, `dateElementProperties` with `timestamp`, `timeZoneId`, `locale`, `dateFormat`, `timeFormat` and the output-only `displayText` |
| `autoText` | `type`, which is `PAGE_NUMBER` or `PAGE_COUNT` |
| `pageBreak` | a page break |
| `columnBreak` | a column break |
| `horizontalRule` | a rule |

Every one of them also carries `suggestedInsertionIds` and
`suggestedDeletionIds`, like every other element in a paragraph.

Vanishing is worse than being unreadable. An inline object gdoc cannot classify
still prints `[object]` with a warning, so a reader is told there is something
there. These seven reach neither `view`'s placeholders nor its warnings: a
person chip, a date chip and a calendar link are simply absent from what `read`
prints, and a page break and a rule with them. Every review session since M2 has
been reading documents with holes in them and being told nothing.

**The fixture is built, not measured.** The reference documents every field of
all seven, so `go/internal/docs/testdata/elements.json` is written from it, with
placeholder text throughout: nobody's document is in it. That is a departure
from how `anchors.json` was made, and the reason is that the `commentAnchors`
shape is undocumented while this one is published.

**So the fixture is a hypothesis until a real document agrees with it, and that
check is outstanding.** A reference and a server can differ, and the field this
is least certain about is `dateElement`, which is the newest of the seven. The
check is Post-Completion work in the M7 plan: a real document holding a person
chip, a date chip and a calendar link, read through `documents.get`, compared
against `elements.json`. If it disagrees, the difference is recorded here as its
own entry and the fixture is corrected. It is not a test loosened.

The decoder is Task 2's, and the wider fix travels with it: the `default` arm
reports rather than vanishes, so the eighth member Google adds shows up as a
placeholder with a warning instead of silently absent.

## 2026-09-09. M7 splits. The survey ships without the write, and the guard's third level travels with the write.

Serves principle 3, and M2's rule about doors with no callers.

M7 was one milestone in PLAN.md: the survey, the in-place styling, the guard's
third write level, the finishing checklist, the nothing-to-protect offer and
`--new`. The review of the draft plan found two things, and either one on its
own would have been enough.

**The survey rested on a decoder that was throwing chips away.** `run()` in
`internal/docs` dropped seven of the eleven `ParagraphElement` members, the
three chips among them, so a survey counting chips would have reported a number
nobody could have trusted, and `read` had been printing documents with holes in
them since M2. That is a live defect in a shipped command, and it does not wait
behind a styling engine.

**The write half shares no risk with the survey.** It carries a new guard level,
a styling engine, a fidelity measurement against a throwaway document, a
read-recompute-send write loop and a live ten-feature run. None of that is
needed to read a document and print what is in it, and the survey is the half
the review skill benefits from immediately.

So the survey, the decoder fix and the named-range read are M7, and the write is
M7b. Nothing in M7 writes to a Google Doc.

### The guard's third level is not added early

`LevelInPlace` is M7b's, beside the command that sends the first request under
it. M2's rule is that a guard door with no production caller is deleted rather
than carried, which is why `GrantInPlace` was deleted with its tests; adding a
level a milestone before its caller would repeat exactly that, and a level with
no caller is a level no test can exercise honestly.

**The allowlist Nail chose for it.** At that level a `batchUpdate` carries the
styling request kinds and nothing else, and `deleteHeader`,
`deleteContentRange`, `replaceAllText` and `deletePositionedObject` are refused
by name. `deleteHeader` on a first-page header is a one-way door this file
already records. This is the one place in the guard where an allowlist over
request kinds is right rather than wrong: `judgeRequests`' own comment says an
allowlist would be empty and would refuse the SUGGEST write the guard exists to
allow, and that argument expires the moment SUGGEST stops bounding what an
unknown kind can do. Under `LevelInPlace` there is no SUGGEST to bound it, so
the list has to name what carries.

The call sites are `policy.go`'s level check and its refusal text,
`Level.String()`, and `AllowFile`, which takes any level and is therefore the
side door around the grant's own invariant.

### The styling is scoped from a measurement, not from `house.yaml`

What in-place styling attempts is decided by a probe that applies each candidate
request kind to a throwaway document and records what actually lands. It is not
derived from reading `house.yaml` and assuming the API can express it. The
2026-08-29 run measured **survival**, which is what the original document keeps,
and that is a different question from **fidelity**, which is what the styled
document achieves. Nothing has measured the second one.

That matters because the honest answer is already known to be a short list. The
nine named styles cannot be redefined at all, so styling means applying
paragraph and text style over every paragraph one at a time, and the next
heading the author types is not house style. `highlight` is an OOXML name with
no Docs equivalent, bullet glyphs and number formats are a fixed enum, tab stops
are read-only, and the Docs API accepts no image bytes, so an inline logo is
closed too. M7b writes that list down rather than shipping the word
"approximate".

### What this entry does not do

It does not touch the Never lists. PRINCIPLES.md, CLAUDE.md and SPEC.md all say
a handed-in document is never direct-edited, and that is true for every line of
M7. M7b amends all three, and SPEC.md's acceptance item 1 with them, which says
a direct edit on a handed-in id is refused.

`--new` stays deferred, decided earlier the same day: it needs a markdown
export, media extraction and a markdown writer, none of which exist in Go.

## 2026-09-09. What an in-place restyle changes, what bounds it, and what it deliberately does not write.

Serves principle 3, and stretches it further than any decision before it. Nail's
decisions, taken with the fidelity measurement above in hand and recorded
together because each one narrows the same door. M7b is where they ship, and
this is the entry the amended Never lists point at.

**The scope came from the measurement, not from reading `house.yaml`.** What a
restyle applies is what a request kind was measured landing on a real document.
The measurement is not the allowlist either: two kinds landed and are refused
anyway, `createParagraphBullets` because it removes leading tabs and
`createNamedRange` because nothing here writes a checklist to mark.

**Typography only. gdoc changes no text at all.** Page geometry, paragraph and
text styling, and table cell appearance. No cover, no front-matter tables, no
legend, no contents list, no list styling, and no heading numbering. Numbering
means writing into the author's prose, and a restyle does not do that.

**The restyle keeps the structure and changes the look.** A paragraph already
marked `HEADING_1` stays `HEADING_1` and gets the house look; body stays body.
gdoc never infers structure from text and never restructures somebody's
document. `build` maps markdown headings to styles and a restyle has nothing to
map from, so this is the answer to what would otherwise be an unasked question.
A named style the house has no look for is reported and left alone rather than
mapped to the nearest one.

**The limit is durability, not fidelity**, and that is the sentence the skill
tells Nail before a restyle. `updateNamedStyle` does not exist, so the nine
named styles cannot be redefined. A paragraph can still be assigned to
`HEADING_1` and have its look overridden per paragraph, which is exactly what
the master template does and why eight rows sit in `drift.Known`. So the
document ends up looking right. The next heading the author types will not.

**What a restyle overwrites, stated rather than discovered.** Applying the house
look to a paragraph replaces the run styling the author chose there. That is
what a restyle is for, and it is still worth a person knowing before they run
one: a hand-bolded phrase inside a body paragraph does not survive a body-text
pass. The report says which paragraphs were restyled.

**The `fields` mask is a rule the guard holds, not a note somebody wrote down.**
The allowlist bounds the request kind and cannot bound the mask, and the mask is
where this level's danger lives. The reference: "To reset a property to its
default value, include its field name in the field mask but leave the field
itself unset." So an `updateTextStyle` carrying `fields: "*"` over a range
resets every property it does not set, bold, italic, links, colours and
highlights, permanently, **with every character intact**. Saying the text is
unreachable while leaving the mask unbounded would be reassuring about the wrong
thing: what this level can destroy is everything except the text.
`checkInPlaceMask` refuses a star, by itself or inside a path, refuses a mask
that names nothing, because Google reads an empty mask as every field, and
refuses a mask naming `useFirstPageHeaderFooter` or `useEvenPageHeaderFooter`,
because switching either off hides the first-page header carrying the logo. The
builder never relies on being refused: every request names exactly what it sets,
and a test walks the whole plan in both directions. The guard holds that same
rule since the M7b review: `checkMaskIsSet` walks every path in the mask into
the request's own style object and refuses one the request leaves unset, because
`fields: "*"` and the fields enumerated one at a time destroy exactly the same
properties, and refusing only the star bounds a spelling rather than the
behaviour.

**SPEC's finishing checklist is deliberately not built.** SPEC has gdoc write a
page after the cover listing the three things the API cannot create, with
checkbox bullets and a named range so it is removable in one call. gdoc reports
those three instead, with the exact menu path for each, and the skill tells
Nail. Two reasons, and the second is the stronger. It is the M2 line held: the
binary prints facts and the skill judges. And writing the page needs `insertText`
and `createParagraphBullets` back on the allowlist, which is the whole of what
keeps a restyle unable to change a character. If a terminal report proves too
easy to lose, writing the page is its own plan and it reopens that question on
purpose.

**A restyle has no capability probe, and the read-back stands alone.** A
proposal is probed because what makes it a suggestion is `writeMode`, a field
gdoc supplies, absent from the public discovery document, and Google returned
200 on it once while making a direct edit. `LevelInPlace` asks for nothing of
that kind: it makes no claim to the server that a probe could test, so a probe
here would be a throwaway document created to answer no question. What replaces
the second bar is reading the document back, and it is both halves.

- **The preservation half.** Thread counts and per-thread witness, pending
  suggestion ids, and chips, before and after. The before comes from M7's
  survey. The witness is the docx export and never Drive: `comments.list`
  reports a destroyed anchor as healthy, returning the original `anchor` and
  `quotedFileContent`, because both are stored strings rather than live
  pointers. A witness that reads `unmatched` is reported as unwitnessed and
  never as a lost anchor, and it keeps `verified` false all the same.
- **The landed half.** A margin, a paragraph's spacing, a run's font and a
  cell's padding, read back to see whether the style is actually there. The
  fidelity run's whole point was the accepted-but-not-landed row, and a
  `verified: true` printed over an invisible change would be that failure
  exactly. The check is made against the requests that were sent rather than
  against `house.yaml`, because a check written from the house style asks the
  question the builder already answers and the two drift the first time a
  builder stops setting a field.

**There is no rollback, and what a failed run leaves behind is written down
rather than discovered.** A run that stops at batch twelve leaves a half-styled
document, and the recovery is the document's own version history, by hand. No
text was touched, so nothing the author wrote is lost, but their own run
formatting inside the paragraphs that were restyled is. That sentence reaches
the envelope's warnings on every path that stops early. A batch Docs refused on
a moved revision is never retried: retrying against a fresh revision would be
gdoc styling a document somebody is editing, which is the exact case
`writeControl.requiredRevisionId` exists to refuse.

## 2026-09-10. The house template reaches a document as a suggestion, not as a direct edit.

Serves principle 3, and it is Nail's idea rather than a compromise found while
implementing one. It removes the milestone's whole danger instead of bounding
it.

**The problem it answers.** `restyle --from` gives a document the house look and
cannot give it the house *template*: no cover, no front-matter tables, no
legend. All of those need gdoc to write text, and being unable to write text is
what M7b's security argument rests on. The design that was on the table before
this widened `LevelInPlace` from four request kinds to eight, added two new
per-run grants to bound where an insert could land, and replaced the property
"none of the four kinds can change a character" with a longer sentence about
adding material before the body. It was buildable and it was worse.

**Nail's question, and the measurement that answered it.** "Table of content,
versioning table or other content might be added in suggested mode, isn't it?"
`TestLiveSuggestedInsertProbe` in `internal/live` asked Google, one request kind
per case, each on a document nobody had suggested anything in:

| Request kind | Asked in SUGGEST mode | What Docs recorded |
|---|---|---|
| `insertText` | accepted | `suggestedInsertionIds` |
| `insertPageBreak` | accepted | `suggestedInsertionIds` |
| `insertTable` | accepted | `suggestedInsertionIds`, 12 marks for one 2x2 table |
| `insertInlineImage` | accepted | `suggestedInsertionIds` |
| `updateParagraphStyle` | accepted | `suggestedParagraphStyleChanges` |
| `updateTextStyle` | accepted | `suggestedTextStyleChanges` |
| `createParagraphBullets` | accepted | `suggestedBulletChanges` |
| `deleteContentRange` | accepted | `suggestedDeletionIds` |
| `updateTableCellStyle` | accepted | `suggestedTableCellStyleChanges` |
| `createNamedRange` | **refused** | `Request does not support application as suggestion.` |

Nine of ten. So the entire template can be proposed rather than written, Nail
accepts it in the browser the way he accepts any suggestion, and gdoc keeps its
inability to change a character on its own authority. A suggested
`deleteContentRange` is the same answer for removing an empty page: a proposal
that can be rejected.

**The probe's own three wrong answers are why it is shaped the way it is**, and
they are recorded because the next person to measure something here will be
tempted by the same shortcuts. Its first run walked only `suggestedInsertionIds`
and `suggestedDeletionIds`, so nine style-change marks were invisible and six
accepted requests read as silent direct edits. Its second run reused one
document, so by the fifth case every index named content the earlier cases had
already suggested: two kinds were refused for a stale index and read as Docs
limits, and two more folded into the suggestion already there and read as direct
edits again. Its third run flattened the tab for `tableStart` and then counted
every mark twice. Only a clean document per case, a walk over every `suggested*`
field, and a count taken from the tab alone tell the truth. A probe that
measures its own leftovers answers about itself.

**Decisions this settles**, all Nail's, 2026-09-10:

- **The comments, the pending suggestions, the chips and the URL must survive**,
  so the template goes into the live document rather than into a new one.
  SPEC's `--new` remains the other mode and remains deferred.
- **Heading numbering is out of M7c, and its own milestone starts from
  suggested mode.** Nail asked on 2026-09-10 whether it could be suggested too,
  and it can: `house.yaml` says `mechanism: literal`, so the number is
  `insertText` in front of the heading's own words, and that is the row the
  probe measured first. His recommendation is that it stays that way when it is
  built, because a number shown as a pending insertion is the clearest that
  change can be made. So the reason it waits is not that it writes into prose. A
  suggestion is not a write, and saying otherwise here would leave a reason on
  record that the probe has already answered. It waits because it has no second
  run and no placement rule. Nothing marks a number gdoc wrote, and
  `numberedHeadingRE` does not recognise the house's own format, since the
  separator is `-` and the pattern wants `.`, `)` or a space, so
  "1-Introduction" reads as unnumbered and a second run makes it
  "1-1-Introduction". A named range per heading is the alternative, and it asks
  the named-range-over-a-suggestion question once per heading rather than once.
  Placement is the other half: the prelude is a single insertion at index 1,
  while every number names a position inside the author's own paragraph, which
  is the thing `propose`'s "a proposal names text, never an index" rule refuses.
- **The contents list stays a manual step**, and that is not a choice:
  `BLOCKED-BY-API.md` already records that `insertTableOfContents` and four
  other spellings answer `Cannot find field`. The `Request` message has no
  member that makes one.
- **The cover's values come from a file the skill proposes and Nail confirms**,
  in `--from survey.json`'s shape, read strictly. A restyle has no note to read
  front matter from, and `internal/cover` already refuses to invent a title and
  hands back a candidate instead: this is that pattern one step further out.
- **A named range marks gdoc's own prelude**, so a second run replaces it rather
  than adding a second cover, **and it is the one thing written directly.**
  `createNamedRange` cannot be a suggestion, and it adds and removes no text, so
  the narrow permission it needs cannot touch a character either. Guessing which
  cover is gdoc's own was the alternative, and guessing is what this project
  refuses everywhere else.

**What this leaves open, and it is the next thing to settle.** The guard gates
every `batchUpdate` on an id granted `LevelInPlace` by the four-kind allowlist,
**whatever `writeMode` says**, and CLAUDE.md gives the reason: an allowlist hung
off the direct-edit branch alone would let a granted document take an
`insertText` under SUGGEST. That rule was written when suggested inserts were
something to prevent. They are now the design. So either a run styles and
proposes through two policies, or that rule changes on purpose. It is a decision
rather than a detail, and it is not made here.

## 2026-09-11. `restyle --new` is not built. Decided, not deferred.

Nail's decision. SPEC's second restyle mode, the one that renders an existing
document to a docx and uploads it as new, is dropped. It had been deferred since
2026-09-09 for what it needs, a markdown export with media extraction and a
markdown writer, and the question this entry answers is whether it is worth
building at all. It is not.

**What it was for, and what covers each case without it.**

- A document with nothing to protect, wanting the exact house style: `read`
  the document, put the text in a note with front matter, `publish --md`. Two
  steps, with the skill as the markdown writer. What the two steps lose is
  pictures, because `read` prints `[image]`, and that is the one thing `--new`
  would have added. A markdown exporter that carries pictures is not worth
  building for that.
- A document whose review has run its course: the note is the source of truth
  in v2, so the note is published again. `publish` refuses a paired note, and
  its own refusal already says what to do: take the `gdoc:` block out by hand
  if it names a document that has gone. Anything more than that sentence is
  M8's, with alignment.
- A document that lives only in Google Docs: the same two steps as the first
  case.

**Why the value went.** On 2026-08-29 the gap between in place and new was the
whole house template. Since M7c the in-place restyle proposes the cover, the
front-matter tables and the legend as suggestions, so the gap is the `manual`
list: the first-page header with the logo, the contents list, the footer page
numbers, tab stops, durable named styles, lists and column widths. For a
document with comments on it that list is cheaper than losing the comments, the
suggestions and the chips. For a document without, `read` plus `publish` is the
same result.

**What changes.**

- SPEC's `restyle` section has one mode. The "detected exception" stays as a
  fact: `restyle --dry-run` still reports `nothing_to_protect`, and the skill
  reads it. What the skill offers over it is the two-step route, not a flag.
- PLAN.md M6's "a second version of a note" and M8's unpairing constraint no
  longer wait on `--new`. The refusal sentence in `publish` is the answer until
  M8 says more.
- `NothingToProtect` in `internal/restyle` is unchanged. It was always a fact
  about five counts, and nothing in the binary offered anything over it.

Nothing in `go/` moves. No flag existed, so no flag is removed.

## 2026-09-16. Help is an answer, completion is a written file, and two skills learn the tool from the tool.

Nail's decision, taken in the brainstorm that produced the M7d plan. Serves
principle 4, every word costs a reader's attention: a person at the terminal
and a session in Claude Code both learn gdoc from gdoc, in the words it uses,
and neither opens Go source to find a flag. Serves 1: nothing new has to be on
the machine.

**What was true before.** `gdoc --help` was `ok: false` and exit 1, and
`cmd/gdoc/doc.go` said why: readable help would have to reach stdout beside
the object, or exit 0 on a run that did no work. There was no completion. One
skill existed, `gdoc-review`, and `build`, `publish` and `restyle` were learned
by reading `doc.go`.

**What changes.**

- `gdoc help` and `gdoc help <words>` print one JSON object on stdout and the
  human text on stderr, where the login URL already goes, and exit 0. Help is
  an answer to a question, the way `auth status` with no token is. The output
  contract does not move: one object, prose on stderr, exit 0 if and only if
  the object says `ok`. `--help` and `-h` are aliases anywhere on the line,
  recognised before the strict parser runs, so no parser refusal changes.
- Bare `gdoc` still fails. It prints the help to stderr, keeps its `ok: false`
  object, and exits 1. A run with no command did no work.
- One command table in `cmd/gdoc/commands.go` is the only description of a
  command. The dispatcher, the usage line, the help and the completion all
  read it. A new command cannot exist without help and completion for it,
  because the only way to be dispatched is to be in the table.
- `gdoc completion zsh --out <path>` and `gdoc completion bash --out <path>`
  write a shell completion script and report what they wrote. The script is
  never printed to stdout, because that would be the one command whose stdout
  is not an object. An existing path is refused without `--force`, the rule
  `build --out` already holds. `install.sh` attempts the zsh script on every
  run: it prints the `source` line when the write landed, and a warning naming
  why it did not when it failed. It never edits `.zshrc`. PowerShell waits
  for M9 and the Windows smoke test.
- Two new skills, `gdoc-publish` and `gdoc-restyle`. Two rather than one
  router, because they start from different things, a note and a link, and a
  skill triggers on its description. `gdoc-apply` stays retired and the name
  is not reused.
- Each flag in the table says how it stands in the call beside what it carries:
  required, optional, or one of a set the command needs exactly one of. The
  usage line brackets what may be left out and puts a bar between alternatives,
  and the object carries the same as a word under `need`. Without it the line
  was every flag run together, which told a reader that `publish` wants a
  `--house` file nobody has and that `restyle` takes `--dry-run --from` in one
  call, which the binary refuses. A skill builds its call from what help
  printed, so that line has to be a call the binary would accept. The binary is
  its own witness rather than the table being trusted:
  `TestEveryRequiredFlagIsOneTheCommandRefusesToRunWithout` runs each command
  with one required flag missing and every other one given, and a flag marked
  wrong in either direction fails there.
- A skill never holds a flag list. Before the first call of a command in a
  session it runs `gdoc help <command>` and reads the words and flags from the
  binary it is about to run. The two new skills are written this way today;
  `gdoc-review` still names its flags and is converted when it is next opened,
  because rewriting the longest skill was not this milestone's work. A test over
  every `SKILL.md` holds one direction meanwhile: on a call line, the line
  opening with `gdoc` or `$GDOC`, every command and every flag exists in the
  table. Prose and continuation lines are not read, so the test narrows the
  blast radius of a rename rather than closing it.
- No fourth module. Cobra would give the help and the completion, and it would
  replace a strict parser whose refusals are each pinned by a test.
  `text/template` in the standard library renders the scripts.

**What was rejected.** An MCP server inside the binary. Grants die with the
process, and an MCP server is a long-lived process, so the per-run grant
model would have to be argued about first. Every tool schema loads into every
session against roughly forty tokens for an unopened skill. It does nothing
for a person at the terminal. The 2026-08-29 entry above already keeps MCP on
the other side of the wire for a different reason.

**What this retires.** The `doc.go` paragraph "The binary never prompts, and
--help is a failure", and the tests `TestTheUsageLineNamesEveryCommand` in its
`--help` form and `TestEveryCommandDispatchReachesIsInTheUsageLine`, which read
a switch that no longer exists. Their replacements are named in the M7d plan.

## 2026-09-16. M8 is deferred to the backlog, and the release goes next.

Nail's decision, the same day M7d merged. Alignment, the align skill, the diff
question and the two things folded into M8 move whole into
`docs/backlog/m8-alignment-and-the-align-skill.md`, and M9, the release, is the
next milestone. A domain choice with no principle above it.

**Why.** The tool is complete enough to hand to the team: a person and a
session both learn it from `gdoc help`, and the review, publish and restyle
loops all run. What the team says after using it is better evidence for what
to build next than the plan's own guess, and alignment is the one open
milestone whose value nobody has asked for yet. Building it first would spend
an evening on a guess.

**What changes.** PLAN.md's M8 section becomes one pointer at the backlog
item, and its standing fact about `sergi/go-diff` says the candidate is
deferred with it. SPEC.md keeps its two alignment sections as the description
of a deferred thing, with each "arrives at M8" changed to say so. Nothing in
`go/` moves. The backlog item names the unknown that would bring M8 back: a
person on the team asking for a document's edits to come back into the hub.

## 2026-09-16. The release: `x.y.z` with a nightly, an updater on demand with one read-only guard door, skills as a Claude Code plugin.

Nail's decisions, taken in the brainstorm that produced the M9 plan,
`docs/plans/completed/2026-09-16-gdoc-v2-m9-release.md`. Serves principle 1: a
colleague gets one zip, one installer and one binary that keeps itself current,
with nothing else on the machine. Strains 3 in one bounded place, below.

**Releases live on this repository.** Nail made the source repository public
later the same day, which is the entry below, so the separate releases
repository this entry first named is not needed: the GitHub Releases of
`nhusnullin/gdoc` are the store, colleagues download from them with no
credential, and the updater fetches from them. The assessment that preceded
it stands: the zip carries no secret beyond the Internal OAuth client, which
only an `altery.com` sign-in can use, and no document; what is public is the
Altery logo, the template's shape inside the binary, and the skills' wording.

**Versions are `x.y.z`, and the number is the channel.** Nail tags `x.y.0` by
hand. A nightly job tags `x.y.(z+1)` when main has moved since the last tag.
`z == 0` is stable, `z > 0` is nightly, and nothing else records which is
which. The first tag is `v2.0.0`, because the tool is already gdoc v2.

**Updates are on demand.** `gdoc update` runs when a person types it and
never otherwise: no check when a session starts, no scheduler, no stamp
file. Nail's revision the same evening; the first draft had every skill
running it before its first call and a minor version applying itself, and
he withdrew that. What it applies when run: same `x`, higher `y`, by
default; a higher `x` only with `--major`; a higher `z` only with
`--nightly`; never down. Every download is verified against its checksum
before a byte is replaced, the previous binary is kept, and `gdoc update
--rollback` swaps it back. With nothing moving unasked, the strain on
principle 3 the first draft carried is gone.

**One read-only door in the guard.** `Policy.AllowUpdateFrom` names one
repository for one run and admits GET on `api.github.com`,
`github.com` and the asset host for that repository's releases, and nothing
else on them. It carries no Authorization header, because the only bearer
gdoc holds is Google's. Opened by `gdoc update` alone, in the shape of
`AllowCreateIn`. The Google rules do not move, and the request is still built
in `internal/gapi`, so the wire's four rooms stay four.

**The skills travel as a Claude Code plugin.** This repository carries
`.claude-plugin/plugin.json` and `.claude-plugin/marketplace.json`, so it is
both the plugin and the marketplace, and `skills/` stays where it is. A
colleague installs with `/plugin marketplace add nhusnullin/gdoc` and
`/plugin install gdoc@gdoc`, chooses global or per-project in Claude Code's
own terms, and updates through Claude Code's per-marketplace toggle. The
first draft of this entry had the installer copying skill folders with a
version marker and the updater replacing them; Claude Code already does that
job for every plugin, so gdoc does not do it twice. Two fallbacks exist
because a managed Claude Code can refuse a marketplace, and with
`strictPluginOnlyCustomization` can refuse personal and project skills too:
where a marketplace is refused but local skills load, the zip carries
`skills/` and `install.sh --skills global|local` copies them, marked so a
re-run replaces only what it wrote; where nothing but plugins and managed
settings load, the administrator names this marketplace in
`extraKnownMarketplaces` and turns the plugin on in `enabledPlugins`, two
lines in `managed-settings.json`, which is the reason the plugin format is
worth having even for a team that could copy folders. The 2026-08-14
decision, skills are linked and never copied, holds for a checkout; a
release copy exists only on the fallback route. The plugin's
version is the tag's, and a release whose two versions disagree is refused
by the workflow. A skill names the binary version it needs in its front
matter and, when `gdoc help` reports an older one, says so and names `gdoc
update`.

**Page breaks are a block in the house style.** `- block: page_break`, before
the version control label and after the classification table, read by the
docx renderer and by the prelude. Nail found the version control table on the
title page of a published document on 2026-09-16; the master pushes its
tables apart with blank lines, which Google's conversion spaces differently.
The offline drift gate reports nothing new: measured on 2026-09-16, the same
169 rows with the same 22 differences, because no item in `drift.Items` reads
a page break or counts a front-matter paragraph. So `drift.Known` gains
nothing, and the two rows this entry expected were never there to explain.

**Builds are trimmed and stripped**, so a home path is not shipped and a
tagged build is the same bytes everywhere. Windows ships under its own tag
when a colleague has run its checklist; `v2.0.0` is macOS.

## 2026-09-16. The source repository is public, and the client secret is injected at build time.

Nail made `nhusnullin/gdoc` public on 2026-09-16, after the assessment
recorded in the release entry above found no secret beyond the OAuth client
and no document anywhere in the tree or its history. Serves principle 1 one
step further: a colleague installs with one line from the repository itself,
and there is no second repository to keep in step.

**What the 2026-08-18 entry foresaw, and what was done about it.** GitHub
scans every public repository for partner patterns whether or not the owner
turned alerts on, and Google is the partner for OAuth client secrets, so the
secret that had been in `auth.go` and in the v1 history was treated as
reported the moment the repository turned public. Two things happened the
same day: the secret left the source, and Nail rotated it in the Google
Cloud console so the reported one is dead.

**Where the secret lives now.** `BundledClientSecret` is an empty variable
in the source, set by the linker from `GDOC_OAUTH_CLIENT_SECRET` in `make
build` and `make dist`, and in the release workflow from a repository
secret. A release build carries it, so principle 1 still holds for a
colleague: one binary, nothing else. A build without it can refresh a token
it already holds, because the token file carries the secret it was issued
with, but cannot sign anyone in, and `Login` refuses before it opens a
listener or prints a URL. `TestLoginRefusesABuildWithNoClientSecret` pins
that. `TestNoGoogleClientSecretInTheTree` in `go/boundary` refuses the shape
of a Google client secret in any file of the tree, so the next person who
"fixes" a local login by pasting one in finds out before the commit does.

**What does not change.** The client id stays a constant: it is public in
every sign-in URL. The client stays User type Internal. RFC 8252's point
stands, the per-user token is what protects an account, and it never leaves
the machine.

**One more door closed.** The Claude workflow in `.github/workflows` answers
only the repository owner's own comments now, because on a public repository
anyone can write `@claude` in an issue and spend the token.

**What this costs colleagues once.** A token issued under the old secret
refreshes until it expires and then fails, so each person signs in again
once with `gdoc auth login` after the rotation. The release README says so.

## 2026-09-16. What the update door actually reaches: two asset hosts, one bigger page, and a version that is only ever a tag.

Three corrections to the release entry above, found by review on the day M9
closed and before any of it had run against a real second release. None of them
widens what the update may do; two widen where it may look, and the third takes
a number out of the envelope. Serves principle 1: the point of `gdoc update` is
that a colleague never assembles anything by hand, and all three of these leave
them doing exactly that.

**The redirect lands on a second host, and the grant names both.** The release
entry named one asset host, `objects.githubusercontent.com`, which is where a
release download redirected for years. Measured on 2026-09-16 against a public
release: GitHub now answers a download with a 302 to
`release-assets.githubusercontent.com`. The guard re-judges a redirect target,
so every `gdoc update` would have refused its own first download, at the
checksum file, before a single byte of a zip. The install path had never been
run, because it needs a second published release, so nothing had said so. Both
hosts are named now: which one a redirect picks is GitHub's to change, a stale
host costs nothing, and neither is reachable without the grant, carries a
credential, or escapes the checksum that is what actually bounds the read.

**The listing asks for a hundred, and `per_page` is the one parameter it may
carry.** The release entry said the listing takes no query at all. GitHub
answers thirty releases when nobody says otherwise, and the nightly cuts one
most nights main moved, so about a month after each hand-cut `x.y.0` that
release stops being on the page. `Choose` would then find no stable candidate,
and a bare `gdoc update` would tell every colleague there is no stable release
while `--nightly` still worked. The guard admits `per_page` on the listing
alone, holds it to GitHub's maximum of 100, and refuses a value it cannot read
as a plain number in that range. `page` is deliberately not admitted: walking
pages is a read count the server decides, and a run with no bound of its own is
not a run this guard can state the shape of. A hundred moves the ceiling to
about three months rather than removing it, which is `docs/backlog/` work with
its reason written there, not a thing to slip in here.

**A version is a clean tag or nothing.** The spec says a binary built from a
checkout is `dev` and prints no version at all, and the skills gate on exactly
that: an older binary than a skill needs stops the run and names `gdoc update`,
and no version at all is a source build, which is not an error. The Makefile
stamped `git describe --tags --always --dirty`, which never returns empty: with
no tag it gives a bare commit hash, and over a tag with uncommitted work it
gives `v2.0.0-dirty`, which parses as a pre-release *below* `v2.0.0`. So the
source-build branch was unreachable and the maintainer's own daily binary
carried a number naming no release. The stamp is
`git describe --tags --exact-match --dirty` now, filtered so a dirty tag falls
back to `dev` as well: a version rides in the envelope when there is a release
behind it and never otherwise. `TestTheVersionStampNamesOnlyATag` in
`go/boundary` reads the assignment, the way the tag target's refusal is read
rather than run.

**Two smaller things travelled with them.** The updater looks for `gdoc.exe` at
the top of a Windows zip, which is the name `release.yml` already packs there;
the single name would have failed after downloading and verifying the whole
archive, and the line that adds `windows-amd64` to `release/platforms` should
not also have to find that. And a major declined on the nightly channel now
names `gdoc update --major --nightly`, because the command without the channel
is a different run and would have taken something else, or nothing.

**What did not change.** The door is still GET only, still one repository,
still per-run, still credential-free, and still opened by `gdoc update` alone.

## 2026-09-17. There is no rc. The nightly is the pre-release channel.

The M9 plan accepted the release by cutting `v2.0.0-rc1`, installing it in a
scratch home, cutting `v2.0.0-rc2` and updating from one to the other. Run for
real on 2026-09-17, the first tag failed in CI before a build: the boundary
test on `plugin.json` holds the version to `vX.Y.Z`, `release.yml` refuses a
tag whose manifest disagrees, and `nightly.yml` stops on a last tag that is not
`vX.Y.Z`. Only `update.Version` accepted `-rc1`, and `IsStable` still called it
stable because its patch is zero, so a real rc would have reached every
colleague on a plain `gdoc update`. Four places said one thing and one said
another, and the one was the rc.

**Decision.** There is no rc. The 2026-09-16 release entry made the number the
channel: patch zero is stable and patch above zero is nightly. A nightly is a
release on the page that nobody gets until they type `gdoc update --nightly` or
install it by name with `--tag`, which is everything an rc is for. An rc would
be a third channel for one use, the day-one acceptance, and that use is covered
by throwaway plain versions: the acceptance ran on `v0.1.0` and `v0.2.0`, both
deleted the same morning. From `v2.0.0` on, every release gets its rehearsal
for free: main moves, 02:00 UTC cuts `x.y.(z+1)`, somebody tries it with
`--nightly`, and when it holds Nail cuts `x.(y+1).0`.

**What changed.** `Version` is three integers and nothing else. `Parse`
refuses a dash by name and says why. `Compare` is the three numbers in order.
The rc rows in `version_test.go` and `policy_test.go` are gone, and the refusal
table gains `v2.0.0-rc1` and `v2.0.0-dirty`. Nothing in the workflows, the
installer or the guard moves, because none of them ever accepted an rc.

**What the run found on the way.** The one-line install, the zip install with
`--skills global`, the plugin install at project scope, a publish into the test
folder with the release binary, its read-back, `update --check`, `update` and
`update --rollback` all held on the first try. The nightly dry run on a main
with no tag says so and cuts nothing. Serves principle 4: a colleague learns
two channels, and the words `rc` and `pre-release` appear nowhere they read.

## 2026-09-17. The plugin is named `altery`, and the marketplace stays `gdoc`.

Installed from the marketplace, the three skills showed in Claude Code's
picker as `gdoc:gdoc-review`, `gdoc:gdoc-publish` and `gdoc:gdoc-restyle`. The
prefix is the plugin's name from `plugin.json`, the rest is the skill's own
name, and both said gdoc. Nail chose the prefix over the skills: the plugin is
now `altery`, and the picker shows `altery:gdoc-review`, `altery:gdoc-publish`
and `altery:gdoc-restyle`. The skills keep their names, so the symlinks under
`~/.claude/skills/`, the `--skills` copies and every document that names one
stay as they were.

**What changed.** `plugin.json` names `altery`, and the one plugin listed in
`marketplace.json` names the same. The install line is `/plugin install
altery@gdoc` everywhere it is printed or written: both READMEs, both
installers, SPEC.md, and the boundary test that pins the manifests and the
release README. The 2026-09-16 release entry keeps the old line, because it
records what was true that day.

**What did not change.** The marketplace is still `gdoc`, because it is the
name a colleague's Claude Code registered when they ran `/plugin marketplace
add nhusnullin/gdoc`, and renaming it would make their update a reinstall.
A colleague who already installed `gdoc@gdoc` sees a new plugin called
`altery` on that marketplace's screen after the next version, and removes the
old one by hand. Serves principle 4: the prefix names whose skills these are,
and the skill names still say what they do.

## 2026-09-18. The wait polls every two seconds.

Nail's decision, made with the quota page open. `comments --wait` polled every
ten seconds, inside the five to fifteen the 2026-08-29 live-review decision
named. It now polls every two, and the range is retired.

**The quota is per user per project, and it is far away.** Google's Docs limits
page states 300 read requests a minute per user per project, 3,000 per project,
and separate pools of 60 and 600 for writes. "Per user" is any one particular
user in the Cloud project, so it is the OAuth token: one colleague's polling
never counts against another's. A poll is one Docs read and one Drive listing,
and Drive is metered in units that a listing barely touches.

| Interval | Docs reads a minute, one user | Users in live review the project pool holds |
|---|---|---|
| 10s | 6 | 500 |
| 5s | 12 | 250 |
| 2s | 30 | 100 |

Two seconds is a tenth of one person's read quota, and a hundred colleagues in
live review at once would still fit the project. Reads and writes are separate
pools, so the poll never eats into the 60 writes a proposal and its reply use.

**What does not change.** The interval is still a constant and never a flag,
the first poll still happens at once, the deadline still bounds the call, and
the loop is still the skill's. `TestTheWaitIntervalIsTwoSeconds` pins the
literal, and a change to it is an entry here first.

A domain choice with no principle above it, by the same argument as the
2026-08-29 live-review decision: the comment and the 🤖 reply arrive within
seconds, and two is closer to that than ten.

## 2026-09-18. The binary notices a release by itself, once a day from `help`, and the plugin carries the stable number.

Nail's decisions, taken in the brainstorm that produced the M10 plan,
`docs/plans/2026-09-18-gdoc-v2-m10-update-notice.md`. Serves principle 1: a
colleague who never opens the releases page hears about a release from the tool
itself. Serves 4: one line, once a day, absent when there is nothing to say.
Strains 3 in one bounded place, below.

**One clause of the 2026-09-16 entry is reversed.** That entry said "no check
when a session starts, no scheduler, no stamp file", and recorded that Nail
withdrew a first draft where every skill ran the updater before its first call
and a minor version applied itself. What comes back is the smallest part of that
draft: a check, and a file that remembers it. What stays withdrawn is
everything that moved: no skill runs `gdoc update`, no version applies itself,
and `gdoc update` still installs only when a person types it.

**`help` checks, and nothing else does.** `help` is the first call of every
skill session, it already carries `version`, and no document is open in front
of it. A stamp, `update-check.json` beside the token file, holds when gdoc last
asked GitHub and what it heard. `help` refreshes it when it is missing or older
than 24 hours, under a two-second ceiling, on the same read-only grant
`gdoc update` opens, with no credential and no document id. A failed check is
stamped too, so a network that refuses GitHub costs two seconds a day. Every
other command is as fast and as offline from GitHub as it was, and a build from
a checkout, which carries no version, never checks at all.

**The object carries facts and the person gets one line.** Under `update`:
`installed`, `latest_stable`, `latest_nightly`, `checked_at`, `error`. No
field says "available" or "behind". The one line on stderr, beside the help
prose, names the newer stable and `gdoc update`, or `--major` across a major.
The skills read the facts, say one line, and carry on; `needs` stays the only
hard gate, and it names a stable `x.y.0`.

**The plugin carries the stable number only.** Claude Code pins a plugin to
the `version` string in `plugin.json` and delivers it when the string changes,
which is its own update mechanism. The nightly stops bumping that file, so a
colleague's skills move when `make tag` moves them. There is no nightly
channel for skills: Nail runs the symlinked checkout, which is ahead of nightly,
and nobody else asked. A `stable` branch was considered and dropped, because
one pin serves one channel and one channel is all there is. The release check
that the plugin carries the tag applies to `x.y.0` tags.

**The hub declares the marketplace.** `extraKnownMarketplaces.gdoc` with
`autoUpdate: true` and `enabledPlugins["altery@gdoc"]` in the hub's committed
`.claude/settings.json`. Read from Claude Code's own bundle on 2026-09-18: that
flag in user or project settings is copied into `known_marketplaces.json` on
startup, so a colleague gets marketplace, plugin and auto-update after one trust
prompt. Nail's machine turns `altery@gdoc` off in the hub's local settings, so
the plugin copy never loads beside the symlinks, and the 2026-08-14 decision
holds for a checkout.

**Not built.** A nudge in the skill about the plugin itself, because the hub
turns auto-update on for everyone. An off switch for the check, because a
stamped failure already bounds its cost; the backlog holds it.

**Measured on this machine, 2026-09-18**, with a binary stamped `v2.2.0` and
the stamp file removed before each run. A cold `help`, the one that asks
GitHub, takes 0.6 to 0.8 seconds, and the first run of a freshly built binary
took 2.0 seconds because macOS was checking the binary itself rather than
because of the network. A `help` with a stamp already there takes 0.05 seconds,
which is what every run but one in 24 hours costs. The two-second ceiling was
never reached with the network working.

The strain on principle 3 is that a command reaches a host a person did not
name, and its bound is the whole of the design above: one command, once a day,
two seconds, one file of gdoc's own, and nothing replaced.

## 2026-09-18. annotate: a comment on quoted words, and nothing else.

Nail's decisions, taken in the brainstorm that produced the M11 plan,
`docs/plans/2026-09-18-gdoc-v2-m11-annotate.md`. Serves principle 3: a comment
is the least a writer can do inside somebody's document, and the batch that
carries it cannot move a character whatever Google does with the write mode.
Serves 2: the skill reads the document, decides in the hub, and hands gdoc the
words and the reason. Bends the probe clause of principle 3 in one bounded
place, below.

**A new command and a new package, not a fourth shape of `propose`.** A
proposal with an empty replacement is refused today on purpose, `propose` runs
the probe and records itself in the note, and its comment id is what lets
`withdraw` find its suggestion later. A comment with no suggestion shares none
of that, so it sits beside those writers rather than inside them.

**The name is `annotate`.** `comment` sits one letter from `comments`, the read
command, and a typo in either direction would be silent: a read where a write
was meant, or a write where a read was meant.

**Both input forms, and they are exclusive.** `--quote` with `--body-file`
leaves one comment by hand. `--from` reads a file of many. Giving both is
refused by name, and so is `--quote` without its body file and the reverse,
before any request leaves the machine.

**The binary adds the prefix.** The file carries the reason and gdoc writes
`🤖 ` in front of it, as `propose` does. A reason that already opens with the
robot is refused, so a comment can never carry two marks. The 🤖 stays the only
record of authorship there is, because Google records every comment under the
operator's own account, and a doubled mark would pass both read-backs: they
compare against the string that was sent.

**No probe, no folder, no note.** The probe answers one question, whether
SUGGEST makes a real suggestion today or a silent direct edit, and this batch
makes no suggestion: a request that inserts a comment cannot edit a character
even when the mode is ignored. So this writer does not run it, which is the
bend in the probe clause, and the bend is one writer wide. Nothing is written
into a note either, because nothing here can be withdrawn. Nothing in the guard
moves: its write levels, its comment routes and its suggestion rules are
untouched, and two tests beside the writer pin that the shape this command
sends is already carried and its unsuggested twin already refused.

**Two read-backs, and nothing raises after the write.** Drive's comment listing
must carry the id with the body that was sent, and the docx export must wrap
the quoted words in a comment range. Neither is enough alone: the export
carries no Drive comment id, and the listing keeps reporting the text a
destroyed anchor used to hold, which is this writer's failure. Both holding is
`verified: true`. Anything less is `verified: false` with a warning naming the
route, reported and never raised, because the comment is in the document and a
caller told the run failed writes it a second time.

**The limits are stated, not lifted.** A document with more than one tab is
refused, as `propose` refuses it. Body text only: headers, footers and
footnotes cannot be quoted. Tables are walked and the live test says whether
the comment-only shape lands in one. A wrong comment is removed by a person in
the document, because deleting a comment is a write the guard does not carry.
Lifting the tab rule is its own small change and not part of this one.

**A run stops at the first entry that cannot be sent, as `propose` does.** The
envelope is then `ok: false`, and the report still carries one entry per
annotation in the file, each answering `sent` for itself, so a stop in the
middle names what landed and what never left. `ok: true` over a run where an
entry failed would give the envelope's `ok` a second meaning, and the exit code
follows `ok`.

## 2026-09-18. Five backlog items closed, and what each decided.

Nail's decisions, taken in the brainstorm that produced the M12 plan,
`docs/plans/2026-09-18-gdoc-v2-m12-five-backlog-items.md`. Five items from
`docs/backlog/` whose value was already agreed, each with one design choice
left. Nothing here opens a door, reaches a new host or adds a command. Two
narrow what the binary does, one makes a test stricter, and two fix the docx
the generator writes. Serves principle 3 in the first three and principle 1 in
the last two.

**The grant never narrows.** `Policy.GrantInPlace` on an id already at
`LevelFull`, which is the level a create the guard itself carried gives a
document, used to write `LevelInPlace` over it: the Drive `PATCH` stopped being
carried and the styling allowlist started to bind, both in silence. Now it
leaves the id where it is and notes through `p.note` that the grant changed
nothing, naming the id and the level. This keeps the levels names rather than
rungs: nothing is compared with `<` or `>`, the one case is matched with `==`
on `LevelFull`. A second grant on an id already at `LevelInPlace` changes
nothing and says nothing. `TestGrantInPlaceLeavesACreatedDocumentAtFull` and
`TestASecondInPlaceGrantIsQuiet` pin both. No command reaches the branch today,
because both production callers grant on ids handed in at `LevelSuggest`.

**The apply loop stops on a quiet answer.** A styling batch Docs accepted whose
answer names no `requiredRevisionId` used to send gdoc back to the document to
read the current revision, and the next batch went out against it. That adopts
whatever a colleague typed in the gap, which is uncertainty resolving toward
the destructive answer. Now a quiet answer with a batch still to send ends the
run the way a refused batch does: the batches that landed stay, `leftBehind`
says the document is half styled, `RevisionUnconfirmed` is set and the error
names the batch. The loop reads nothing between batches at all.
`TestAnAnswerCarryingNoRevisionMidRunStopsTheRun` pins the mid-run stop and
`TestTheLastAnswerCarryingNoRevisionIsAWarningAndNotARead` keeps the last-batch
case, which was already a warning. `RevisionOf` stays exported for its one
remaining caller, the prelude phase boundary in `cmd/gdoc/restyle.go`, whose
own comment says what that read costs.

**The offline drift gate pins the measured pair.** `drift.Known` is a map of
name to reason, and a row named in it used to pass on any difference in either
direction, so the gate could no longer tell a known difference from a house
value somebody moved. `Known` stays as it is, shared by both gates, and the
offline gate gains a second map beside it, `knownOffline` in
`internal/drift/pins_test.go`, from name to the two values as `show` prints
them, every one a literal. A `Known` row whose values move now fails until
somebody re-records the pair with its reason, which is the golden-file
discipline the gate already has. `TestEveryKnownDifferenceHasItsPairPinned`
holds the two maps to each other in both directions and
`TestAKnownRowWhoseValuesMovedFailsTheGate` holds the check itself. The live
gate is left alone, because no person has read its rows yet, so there is no
measured pair to pin there.

**One numbered list definition per numbered list.** The generator wrote one
`w:num` for the whole document, so a second numbered list in a note printed 4,
5, 6. The body walker now hands a fresh list id to every ordered list that is
not nested inside an ordered list, counts them on `Result`, and `render.Build`
writes that many `w:num` entries, all pointing at the one numbered abstract
list, each stating `w:startOverride` 1 on all nine of its levels. The override
is the restart said in the file rather than assumed of the reader: a fresh id
alone rests on Word keeping its count per instance and not per abstract list,
which nothing here measures, and pandoc and python-docx both write the override.
The override is correct under either reading of ECMA-376 17.9.27 and costs nine
empty elements per list. The Word master restarts a third way, one
`w:abstractNum` per list, which needs no override and costs a definition per
list instead. A nested ordered list names its parent's
id, so level restarts work as they did. The arithmetic lives in `render` beside
`numberingPart`, which is why `NumberNumID` became a function, and
`NumberNumID(1)` is still `"2"`, so a body paragraph in a document with one
list names the list it always did. The "carries on from the one above" warning
is gone, because it is no longer true. The "starts at N in the note and at 1 in
the document" warning stays: honouring an author's start number is that same
override carrying their number, and it is a decision nobody has taken.
`TestOneNumberedListDefinitionPerNumberedList`,
`TestEveryNumberedListOverridesItsStart`,
`TestASecondNumberedListStartsAgain`, `TestANestedNumberedListNamesItsParent`,
`TestANumberedListUnderABulletOpensItsOwn` and
`TestAListThatStartsElsewhereStillSaysSo` pin it.

**An anchor link is a jump to a bookmark every heading with words carries.**
`[see below](#scope)` used to become an external relationship to the literal
string `#scope`, and Word opened nothing. Every heading that emits a paragraph
now carries a `w:bookmarkStart`/`w:bookmarkEnd` pair around its runs, named
from goldmark's auto heading id through one function: `h_` and then the id with
every character outside `[A-Za-z0-9_]` replaced by `_`, and when that is longer
than 40 characters, which is Word's ceiling on a bookmark name, the first 32
characters, then `_`, then the first 7 hex digits of the FNV-1a 32-bit hash of
the id as it arrived. The rule is deliberately wider than what goldmark emits.
A link whose destination opens with `#` becomes `<w:hyperlink w:anchor>` with
no relationship and no `Media` entry, through the same function. A `#` link
naming no heading is warned about with its line and printed as plain text,
which is how a code block is already refused: a dead jump in Word is worse than
words that do not jump. The walk collects the heading ids before it renders, so
a link to a heading further down resolves, and it collects only the headings
that will carry a bookmark, so a link to a figure-only heading is a dead anchor
like any other. The line the warning names is the walker's current line, set on
every path that emits runs: a heading, a paragraph, a table and a block quote,
which handles a top-level paragraph itself rather than through
`paragraphBlock`. A table names the table's own line rather than the cell's,
because a cell has no line the author would recognise.
`TestBookmarkNameIsWordSafe`, `TestEveryHeadingCarriesABookmark`,
`TestAnAnchorLinkIsAJumpAndNotARelationship`,
`TestAnAnchorToNoHeadingWarnsAndPrintsPlainText`,
`TestADeadAnchorInsideABlockQuoteNamesItsOwnLine`,
`TestADeadAnchorInsideATableNamesTheTablesOwnLine` and
`TestAFigureOnlyHeadingCarriesNoBookmark` pin it. What Google's import does with
a `w:anchor` hyperlink and a `w:bookmarkStart` is unmeasured, and if the jump
does not survive it that is a MEASURED.md row, not a reason to revert the docx
side, which Word reads.

## 2026-09-19. Export, and the `gdoc:` block as a list of documents.

Nail's decisions, taken in the brainstorm that produced the M13 spec,
`docs/plans/2026-09-19-gdoc-v2-m13-export-and-align.md`, which holds all
seventeen of them with the twenty-three scenarios they were read against. This
entry holds what changed in this file. Serves principle 3 in the first row
below, because the one new output is the one thing that can never be replaced,
and principle 2 in the second, because the pairing stays in the note's front
matter and the pictures land beside it.

| Row | Status change |
|---|---|
| The markdown is the source, the Google Doc is a rendering (2026-08-13) | superseded 2026-09-19. No side is the source by rule. The person decides per run |
| Publish runs once. Everything after travels as suggestions (2026-08-29, restated 2026-09-11 and 2026-09-16) | superseded 2026-09-19. A note may be published more than once, and each publish is a new document |
| v1's `gdoc: <id>` stays refused. Publish is the only writer of the block (2026-09-08) | superseded 2026-09-19 in its one-writer clause. `export` writes the block too, and the block is a list of documents. The v1 refusal holds |
| Tabs are out of scope, and gdoc defends against them (2026-08-29) | superseded 2026-09-19 in its reader half. `export` writes one file per tab. Every writer still refuses a document with more than one |
| gdoc never replaces the body of a document that already exists (2026-08-29) | holds, and is what makes a merge into the same document a set of suggestions rather than a replace |
| gdoc proposes. It never accepts and never rejects (2026-08-29) | holds. A pending suggestion is exported as a marker and never decided |
| Export has no `--force`, and a taken name takes the next free number | new, serves principle 3 |
| Two routes matched by order, and `contentUri` stays out | new, serves principle 1 and the guard |

**No side is the source of truth by rule.** The 2026-08-13 decision said the
markdown is the source and the document is a rendering of it. That was true
while a note was published once and every later change travelled as a
suggestion. It stops being true the moment a document can come back into the
hub, because then either side may hold the newer words and no rule in the
binary can say which. So the rule goes and the person takes it: they decide
per run, and the skill asks. The block records dated facts only, when a
document was published from the note and when it was exported into it, and no
field in it says which side was right.

**A note may be published more than once, and each publish is a new document.**
`publish` used to refuse a paired note and tell the person to take the `gdoc:`
block out by hand. That refusal was the whole of the answer to "publish this
again", and it made the note forget the first document. Now `publish` appends
an entry. The old document keeps its URL, its threads and its own entry
untouched, and it goes stale, which the skill says once before it runs.
Publishing into the same document is still refused, because gdoc never replaces
a body: that request is a merge, and a merge is suggestions through
`gdoc-align`. `TestPublishAppendsAnEntryToAPairedNote` is the pin.

**The block is a list of documents, and schema 1 still reads.**
`gdoc: {schema: 2, documents: [...]}`, one entry per document the note has met,
each holding its own `published`, `exported`, `folder_id`, `suggestions_seen`
and `proposals`. A proposal lives under the document it was made in, because a
`withdraw` that sent an id from another document would be a reject against the
wrong file. Every writer takes the entry the URL names and refuses a URL the
list does not hold, naming the ids it does. A schema 1 block reads as a
one-entry list and is written back byte for byte while nothing changes it; the
first write that changes anything renders schema 2, and the reply says so once.
`TestWriteAnUnchangedSchemaOneBlockStaysSchemaOne`,
`TestAChangingWriteRewritesSchemaOneAsTwo`,
`TestEveryWriterActsOnTheURLAndRefusesAnIDOutsideTheList` and
`TestWithdrawNeverSendsAnIDFromAnotherDocument` are the pins.

**A colleague on an older binary is refused by name, and told nothing about
updating.** The shipped read is strict before it looks at the schema number, so
a gdoc from before this milestone meets a schema 2 block with
`gdoc front matter: [3:3] unknown field "documents"` and the four lines of YAML
it was reading, the third marked. It never reaches its schema sentence, and it
cannot be taught one, because it is already on somebody's machine. The daily
notice in `help` is where a colleague learns a newer release exists. Nothing in
the note breaks and nothing is written.
`TestTheShippedDecoderRefusesADocumentsKeyByName` is the pin.

**Export has no `--force`, and a taken name takes the next free number.**
Nail struck the flag on 2026-09-19. A flag that replaces a note is a false
door: the value of an export is that it costs nothing to run again, and the
cost of one wrong run under `--force` is a person's own writing. So a taken
path gets `<stem>.2.md`, then `<stem>.3.md`, pictures take the next free number
under `assets/`, and the reply names every file it wrote. A person who wants
the old note gone exports to the new file, reads the old one and renames by
hand. The one thing export writes into a file somebody else owns is the date,
`exported: {at}` on that document's entry, through the byte-preserving block
write. `TestNothingCanReplaceAFile` reads this package's own source: the word
`force` does not appear in it, and the one call to `atomicfile.Replace` is the
stamp. It rests on `atomicfile.Create`, which ends in a link rather than a
rename, so a path that appeared between the plan and the write is refused
rather than replaced: `TestCreateRefusesAPathThatExists`.

**Two routes matched by order, and `contentUri` stays out.** A picture's bytes
are not in the Docs answer. The `contentUri` it carries is a
`googleusercontent.com` host the guard does not admit, and admitting one for a
read of somebody's picture is a wider door than this earns. So the bytes come
from the docx export, which is a read gdoc already makes for the comment
witness, through the same grant. Nothing crosses the two routes: the read has
object ids the export never mentions, and the export has part names the read
never mentions. So the k-th picture of one is the k-th of the other, and order
is the whole pairing. When the counts disagree every picture is a placeholder
and no file is written, because a pairing one place out writes the wrong bytes
under the right name and nobody reading the note afterwards can see it.
`TestPicturesPairByOrder`, `TestObjectsAreTheTabsPicturesInBodyOrder` and
`TestACountMismatchWritesNoPictureAndWarns` are the pins. Keeping the note's own
picture when the bytes match rests on a published PNG coming back byte for
byte, which is not measured, so `Options.MatchByHash` ships off and the reply
says the match was off: `TestTheHashMatchIsOffUntilMeasured`.

**A document with more than one tab exports as one file per tab.** Nail's
correction on 2026-09-19, reading the scenarios: a tab is a document of its own
to a reader, so one file each, never one file for both and never a refusal. The
first tab lands at `--out` and each further tab beside it as
`<stem>-<tab title>.md`, slugged the way a heading id is, under the same
numbering rule when a name is taken. Each file's entry names the document and
its `tab_id`. This narrows the 2026-08-29 tabs decision for one reader and
leaves it whole for every writer: `propose`, `annotate`, `publish` and the rest
still refuse a document with more than one tab, until a live measurement says
what a write into one tab does. `TestTwoTabsGiveTwoProjections` and
`TestTabSlugsCollide` are the pins.

**A marker never reaches a document, in either direction.** A file export wrote
carries `{+words+}[s:ID]`, `{-words-}[s:ID]` and `[[c:ID]]words[[/c]]`, and a
person may keep it in the hub with its markers for as long as they like: there
is no deadline to resolve them, because the markers are gdoc's own and any
later session can undo them. What a marked file cannot do is become a document.
`build` and `publish` refuse a marker by line, `plaintext` refuses one in
anything `reply` or `annotate` writes into a thread, and `propose` refuses one
in its `--from` file, which never passes through `plaintext`. One package holds
the four openers as literals so the four call sites cannot drift apart:
`TestMarkersNamesEachOpener`, `TestBuildRefusesAMarkerByLine`,
`TestPlaintextRefusesAMarker` and `TestProposeRefusesAMarkerInTheFile`.

**A backslash makes the one character after it the document's own.** The
escaping in `internal/view` used to run one text run at a time, so a document
whose own `[` ended one run and whose `[` opened the next reached the text bare
and read as gdoc's marker. The fix needed a convention, and the two candidates
were escaping the first half of the pair or the second. The second cannot spell
the third case: a run ending in `[` in front of one of gdoc's own comment
markers would put the backslash in front of the marker, and a reader would then
take the marker as the document's words and lose it. So the projection holds the
document's last character back until it knows what follows and escapes the first
half everywhere. Two runs give `\[[`, and a document character against a marker
gives `\[[[c:ID]]`. One convention, not two, and the reader's rule is the one
sentence above, which is the parity `internal/markers` already reads.
`TestEscapingHoldsAcrossTwoRuns` and `TestUnescapeAfterEscapeIsIdentity` are the
pins, the second being the reader written out as a test-only inverse. A bracket
inside a link's words is escaped too, because the link form makes a bracket
markup where it was not before: `TestABracketInsideALinksWordsIsEscaped`. This
closed `docs/backlog/escaping-across-run-boundaries.md`, which was a blocker for
export rather than a backlog item, because a skill that merges an exported file
into a note has to undo the escaping exactly.

**The house prelude is stripped by position, and never by its words.** A
document `publish` made carries the cover, the three tables, the legend and the
contents list, and a document `restyle` made carries the same under gdoc's own
named range. Export takes them out and lists what it removed with the text each
piece held, so an edit somebody made inside a cover table is in the envelope
and the skill can compare it with the note's keys. The named range is the
answer where there is one. Where there is none the answer is the layout, from
the start of the body to the end of the contents list, and only when the span
in front of the first level-one heading holds the three house tables. Nothing
else is stripped: an edited cover title still matches, because no rule reads a
word of it, and a heading somebody added inside the cover does not, so the whole
document stays in the file with a warning naming what stood there. Stripping on
a guess would take somebody's own front page out of their note, and nothing puts
it back. `TestARestyledPreludeIsStrippedOverItsNamedRange`,
`TestAPublishedPreludeIsStrippedByLayout` and `TestAnEditedPreludeStaysAndWarns`
are the pins.

**`export` is a command, not a flag on `read`.** `read` stays a pure read that
writes nothing on disk and keeps its short help. The command table grows from
fourteen rows to fifteen, so `help` and completion follow it, and the entry for
`export` is where the numbering rule, the `assets` folder and the block are
written. The path flag is `--out`, which is what `build` already calls a file to
write, and never `--md`, which means "the paired note" on five commands.

**Two skills, and the asking lives inside them.** `gdoc-export` serves a
document with no note, because a colleague who says "bring this into the hub"
has nothing to align. `gdoc-align` serves a paired note, in both directions:
the document into the note as a merge the person agrees to line by line, and
the note into the document as suggestions. It reads the merged whole once more
and names every place where the logic broke, because a merge that takes a
paragraph from each side can leave a document that contradicts itself, and
those are conflicts too. A stale copy left by a run that died is never merged
from: the skill says when it was made, exports again, and deletes the stale copy
only when the fresh body matches it. There are no slash-command-only rules,
because a session with `gdoc` on PATH can run any command by hand, so a skill
that does not fire does not stop the action. It only skips the asking. So the
asking is in the skills: `gdoc-publish` on a paired note stops and asks, a new
document or a merge; `gdoc-align` shows the list of paragraphs and asks once
before its first `propose`. The marker rules are written once, in a file beside
`gdoc-export`'s SKILL.md, and `gdoc-align` reads the same file. `needs:` becomes
`v2.4.0` on review, publish, export and align, because a paired note publishes
and a schema 2 block reads only on the new binary. Restyle takes no note and
stays at `v2.0.0`.

**An `ai!` comment may not run an export.** Export starts from a session by
name. The review skill answers such a comment with one reply saying so, which
keeps the rule that a comment is a prompt into the hub and not a way to write
files somebody did not ask for.
