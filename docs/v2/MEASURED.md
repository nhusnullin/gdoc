# Measured

What Google does when it accepts. Kept separately from
[DECISIONS.md](DECISIONS.md) on purpose: a decision is a choice, and none of
these are chosen. It is the sibling of [BLOCKED-BY-API.md](BLOCKED-BY-API.md),
which holds what Google refuses. This file holds what it allows, and what shape
the answer comes back in.

Every section carries the date, the test that made the measurement, the verbatim
answer where there is one, and a line saying when to ask again. This file holds
the whole index, "Not measured yet" and the sections dated 2026-10-16 and later;
earlier sections are frozen in [MEASURED-archive.md](MEASURED-archive.md).

| Measured | Section | Entry |
|---|---|---|
| 2026-09-09 | Which styling requests land in place | MEASURED-archive.md |
| 2026-09-10 | In-place styling preserves anchors and pending suggestions | MEASURED-archive.md |
| 2026-09-10 | A named range over a suggested insertion | MEASURED-archive.md |
| 2026-09-10 | A table takes one index of its own at the end | MEASURED-archive.md |
| 2026-09-18 | A proposal into a table cell lands | MEASURED-archive.md |
| 2026-09-27 | A block of new paragraphs, proposed in one SUGGEST batch | MEASURED-archive.md |
| 2026-10-03 | Claude Desktop and a local MCP server | MEASURED-archive.md |
| 2026-10-03 | gdoc mcp in Claude Desktop, the first run | MEASURED-archive.md |
| 2026-10-03 | gdoc mcp in Claude Desktop, the red team and a clean install | MEASURED-archive.md |
| 2026-10-14 | Drive's import of a Word footnote | MEASURED-archive.md |
| 2026-10-14 | Drive's import of run shading and text colour | MEASURED-archive.md |
| 2026-10-15 | Footnotes and the marks, published by gdoc | MEASURED-archive.md |
| 2026-10-16 | How read prints the two kinds of page break | MEASURED.md |
| 2026-10-17 | The presence app built, rendered and unzipped, the prototype's three numbers | MEASURED.md |
| 2026-10-17 | What Gatekeeper says about a bundle gdoc's own code unzipped | MEASURED.md |
| 2026-10-17 | Twenty thousand presence lines from two binaries into one file | MEASURED.md |
| 2026-10-17 | The bundle swapped under a process running out of it | MEASURED.md |
| 2026-10-16 | Drive's import of a heading's bookmark | MEASURED.md |
| 2026-10-17 | What Claude Code puts in a child's environment | MEASURED.md |
| 2026-10-17 | What ImageRenderer draws for a SwiftUI ScrollView | MEASURED.md |
| 2026-10-17 | What LaunchServices says about a `claude://` handler | MEASURED.md |
| 2026-10-17 | What proc_pidpath says after the binary is renamed | MEASURED.md |
| 2026-10-18 | The lock, `open -g`, and a command whose opener cannot start anything | MEASURED.md |
|  | Not measured yet | MEASURED.md |
| 2026-10-18 | What `claude://code/continue` does from outside the app | MEASURED.md |
| 2026-10-18 | Where the screen cuts a panel as tall as its content | MEASURED.md |
| 2026-10-18 | What Spotlight answers for a document id | MEASURED.md |
| 2026-10-18 | What the header line holds, and what a second version costs it | MEASURED.md |

A measurement is never edited to match a later reading; a different answer is a
new section in this file rather than a change in the archive, and the old one
keeps its words.

A citation of the form "measurement N" or "MEASURED N" is a row of the numbered
table in "Claude Desktop and a local MCP server", and one of the form "row N" is
a row of the table in "A block of new paragraphs, proposed in one SUGGEST
batch". The numbers inside those tables are unchanged by the move, and this
index says which file each section is in.

"Not measured yet" names the two sections it leans on rather than saying "the
section above", because the section above it may be in the archive.

## How read prints the two kinds of page break

Measured 2026-10-16 by `TestLivePageBreakShape`
(`go/internal/live/pagebreak_test.go`), one document, built from a note of a
heading, a paragraph, `---` and a paragraph, and trashed on the way out. The
document was `1MmipMWaS96OmV3ONxFioNFmaKhBYHWb7w3jrrF90hic`. It reads through
`docs.URL`, which is the read every command makes, and it asserts nothing about
the answer: the answer is Google's.

**Neither kind comes back as a `pageBreak` element. Both come back as
`pageBreakBefore` on the paragraph's own style, a field `internal/docs` did not
decode on the day of the measurement, so `read` printed nothing at all where
the page turned. Task 5 of the same milestone decodes it.**

The last column is what `read` prints since Task 5 of the same milestone, which
decoded the flag. The column before it is what it printed on the day of the
measurement, and for every release from M13 to v2.12.0.

| Kind | What the Docs answer holds | What `read` printed | What `read` prints now |
|---|---|---|---|
| the body break, where the note wrote `---` | a `NORMAL_TEXT` paragraph whose only element is a `textRun` of `"\n"`, carrying `paragraphStyle.pageBreakBefore: true`. No `pageBreak` element | nothing. One blank line between the two paragraphs, the same blank line any two paragraphs have | `[page break]` on a line of its own between the two paragraphs, with a warning. `export` writes `---` for it |
| the first body heading's break, the house's own | the heading paragraph itself, `HEADING_1`, text `1-Where the page turns\n`, carrying `paragraphStyle.pageBreakBefore: true`. So it rides inside the heading and is not a paragraph before it | `# 1-Where the page turns`, with no marker of any kind | `[page break]` on the line before the heading. `export` leaves that one out, because `build` puts it back |
| the front matter's own two breaks, measured because they were in the same document | two `NORMAL_TEXT` paragraphs of `"\n"` each, both carrying `paragraphStyle.pageBreakBefore: true` | nothing | `[page break]` for each, in `read`. `export` strips both with the prelude they stand in |

The document walked as 69 paragraphs and not one of them held a `page_break`
run. The four `pageBreakBefore` paragraphs above are every break in it.

So a `pageBreak` element is a Docs-native break, the one a person inserts with
Insert, Break, Page break, and the shape
`go/internal/docs/testdata/elements.json` was built from. A break that reached
a document through a docx import is not that element, and nothing in the read
path saw it until this measurement. The round trip's `page breaks` entry had
been excusing a `[page break]` line that a published document never produced,
and Task 5 rewrote that entry to the house's own breaks.

This is the gate Task 4 of the M19 plan set, and it did not hold. The
specification's export rule assumed a body break comes back as a paragraph
holding one `pageBreak` and no text. It comes back as a paragraph style, so
`export` as it stood could not see a break at all. Three ways out were written
down, and the first was taken:

- **Taken, 2026-10-16, by Claude under Nail's standing "do it for me" of the
  same day and in his absence.** `internal/docs` decodes `pageBreakBefore` as
  a fact on a `Paragraph`, and `view` and `export` read it. That is the
  smallest change and it touches the one room every read goes through. Task 5
  of the plan was rewritten from it, and the specification's `view` and
  `export` rows with it: `read` is no longer unchanged by this milestone.
- Declined. `render` writes the break paragraph as a run holding
  `w:br w:type="page"`, the docx spelling of an inline break, instead of
  `w:pageBreakBefore` on the paragraph. Whether Drive's import turns that one
  into a `pageBreak` element is itself unmeasured.
- Declined. M19 ships without the export half, and a document's page breaks
  stay invisible to `read` the way they have been since M13.

Nail can reverse to either of the two declined. Reversing means the third
column of the table above again, and `Paragraph.PageBreakBefore` with no
reader.

Recheck when Google's docx import changes, or before anything downstream of
`read` is asked about a page break.

## The presence app built, rendered and unzipped, the prototype's three numbers

Measured 2026-10-17 on Nail's machine: macOS 26.3.1 (build 25D771280a),
Xcode 26.6 (17F113), Apple Swift 6.3.3, no signing identity. These are the
three numbers the design record of the same night
(`docs/design/2026-10-17-presence-mechanism.md`, item 10) claimed from a
prototype, taken again from the committed package. Not Google: this file holds
what a platform does when it accepts, and macOS is the platform here.

**A universal bundle builds in 14 seconds from a clean clone, with no Xcode
project.** Taken as `rm -rf presence/.build bin/GdocPresence.app` and then
`time make app`, which runs `presence/bundle.sh dev bin/GdocPresence.app`,
which runs `swift build -c release --arch arm64 --arch x86_64`. The answer was
14.286 seconds wall, 22.47 user and 8.53 system, at 216% CPU. The prototype
said 13 seconds. The bundle that came out is `x86_64 arm64` by `lipo -archs`
and `version 0.0.0`, which is what a build from no release stamps.

**The panel renders to PNG headless, in both schemes, from `swift test`.**
Taken as `PRESENCE_SHOTS=/tmp/m23-shots swift test --filter PresenceUITests`
in a non-interactive session with no window server session of its own. Five
tests, 0 failures, 0.111 seconds. Two files came out, `panel-light.png` 84,460
bytes and `panel-dark.png` 85,771 bytes, both `PNG image data, 640 x 937,
8-bit/color RGBA, non-interlaced` by `file -b`. No Screen Recording permission
was asked for and none is held: the render goes through an off-screen bitmap
and never reads the screen.

**The binary `gdoc update` installed carries one attribute, and it is not
quarantine.** Taken as `xattr -l ~/.local/bin/gdoc` on the v2.12.1 release
binary that `gdoc update` put there on 2026-10-16. The answer is
`com.apple.provenance` and nothing else. The bundle half of the same question
is the section below.

Recheck when the Swift toolchain or the macOS major version changes, or when a
signing identity appears on the machine, because that is what M17 would change.

## What Gatekeeper says about a bundle gdoc's own code unzipped

Measured 2026-10-17, on a bundle built here, zipped the way the release zips it
and unzipped by `internal/update`'s own `ApplyBundle` rather than by `unzip` or
by Archive Utility. The point of the measurement is the route: quarantine is
set by the program that downloads, and the program that downloads here is Go.

How it was taken, in four steps:

1. `presence/bundle.sh v9.9.9 /tmp/m23-measure/stage/presence/GdocPresence.app`,
   so the plist version is the tag the asset name will carry.
2. `ditto -c -k --norsrc --noextattr --keepParent presence
   /tmp/m23-measure/gdoc-v9.9.9-darwin-arm64.zip`, the release job's own
   command, and `shasum -a 256` into a `SHA256SUMS` line beside it.
3. `ApplyBundle(archive, sums, "gdoc-v9.9.9-darwin-arm64.zip",
   "presence/GdocPresence.app", dest)` from a throwaway test in
   `go/internal/update`, with `dest` in a temp folder. It answered
   `version=9.9.9`, `previous=""` and
   `treesum=fe811d76e61109db26c951790fd7315da076d0b97dd0a8ce63711f3b19ef5138`.
4. `xattr -lr`, `codesign --verify --deep --strict -vv`, `codesign -dvv`,
   `spctl -a -t exec -vv` and `lipo -archs` on what landed.

| Command | Answer |
|---|---|
| `find` | seven paths: the bundle, `Contents`, `Contents/Info.plist`, `Contents/MacOS`, the executable, `Contents/_CodeSignature` and its `CodeResources`. Nothing else |
| `xattr -lr` | `com.apple.provenance` on every one of the seven, with an empty value, and **no `com.apple.quarantine` anywhere** |
| `codesign --verify --deep --strict -vv` | `valid on disk`, `satisfies its Designated Requirement` |
| `codesign -dvv` | `Identifier=com.altery.gdoc.presence`, `Signature=adhoc`, `flags=0x2(adhoc)`, `TeamIdentifier=not set`, `app bundle with Mach-O universal (x86_64 arm64)` |
| `spctl -a -t exec -vv` | `rejected` |
| `lipo -archs` | `x86_64 arm64` |
| `ls -l` on the two files | the executable `0755`, 1,150,096 bytes; the plist `0644`, 765 bytes. The modes came off the zip |

**`spctl` rejects every ad-hoc bundle, and that is not the verdict on whether
the app opens.** `spctl -a` asks what Gatekeeper would say about an assessment,
and an assessment is only made on a file carrying `com.apple.quarantine`. The
bundle above carries none, so nothing assesses it, and the `rejected` line is
`spctl` answering a question nobody asked it. The control was run as well: the
same bundle copied with `ditto` and given
`xattr -w -r com.apple.quarantine "0083;68f1c0a0;Safari;"` still verifies under
`codesign` and still says `rejected` under `spctl`, so the two answers do not
tell the two bundles apart. Only a launch does.

**The launch is Nail's step, and so is the negative control.** Nothing here
opens a GUI app. His two commands, in this order:

```sh
# The positive case: the bundle gdoc update wrote, no quarantine, ad-hoc signed.
open ~/Applications/GdocPresence.app          # expected: the glyph appears

# The negative control: the same bundle with a download's attribute on it.
ditto ~/Applications/GdocPresence.app /tmp/quarantined/GdocPresence.app
xattr -w -r com.apple.quarantine "0083;00000000;Safari;" /tmp/quarantined/GdocPresence.app
open /tmp/quarantined/GdocPresence.app        # expected: macOS blocks it
```

If the first one is blocked, the ad-hoc signature is not enough on this macOS
and M17 moves up: that would be a new section here, not an edit to this one.
If the second one opens, quarantine is not what blocks, and the reasoning above
is wrong in a way worth a section of its own.

Recheck on every macOS major version, on the first CI-built bundle, which is
signed by the same `codesign -s -` on a different machine, and on the day a
Developer ID appears.

**The launch, measured 2026-10-17 at 09:17 on Nail's machine, macOS 26.3.1.**
`gdoc update` under `presence: true` installed `~/Applications/GdocPresence.app`
(2.14.0, universal, ad-hoc, `codesign --verify --deep --strict` valid, no
`com.apple.quarantine` on any file) out of the v2.14.0 darwin-arm64 zip, and
Nail opened it with `open ~/Applications/GdocPresence.app`. It launched with
no dialog; `pgrep -x GdocPresence` showed one process out of that path, and
`log show --last 2m --predicate 'process == "syspolicyd"'` carried no line
about it. So the Go route holds on this machine: a bundle gdoc's own code
wrote is not assessed on first launch, and the ad-hoc signature is enough.
The browser route (a quarantined copy) was not run on this machine; the
negative control from the section above stands as it was measured.

## Twenty thousand presence lines from two binaries into one file

Measured 2026-10-17. The channel's one promise is that two gdoc processes may
append to the same file with no lock and no torn line. The suite holds that
with 16 writers in one process (`TestTwoWritersNeverTearALine`); this is the
same question asked of separate binaries, separate processes and 20,000 lines.

How it was taken:

```sh
cd go && go build -o /tmp/m23-run/gdoc-a ./cmd/gdoc
cd go && go build -o /tmp/m23-run/gdoc-b ./cmd/gdoc
printf 'presence: true\n' > /tmp/m23-cfg/settings.yaml
# loop.sh runs $1 $2 times: GDOC_CONFIG_DIR=/tmp/m23-cfg "$1" read <id>
(./loop.sh /tmp/m23-run/gdoc-a 5000 & ./loop.sh /tmp/m23-run/gdoc-b 5000 & wait)
```

The command is `read` against a document id with no token in that config
directory, so each run fails at the token, reaches no network, and writes its
two lines: a `start` from the hook in `dispatch` and an `end` from the same
hook on the way out. 5,000 runs is 10,000 lines per binary and 20,000 in the
one file. The wall time was 155 seconds, so about 15.5 milliseconds per
process, both loops running at once.

| What was asked | Command | Answer |
|---|---|---|
| how many lines | `wc -l` | 20,000 |
| how many bytes | `wc -c` | 3,860,000 |
| is every line one object | `grep -c '^{"v":1,'` and `cut -c1 \| sort -u` | 20,000, and the only character a line starts with is `{` |
| how long is a line | `awk` over the file | min 184, max 200, mean 192.0 bytes |
| does the reader read them all | `Event.scan` from `presence/Sources/PresenceCore`, the app's own reader, over the whole file | `events=20000 skipped=0 newerSchema=false` |
| do the pairs survive | the same scan | 10,000 `start`, 10,000 `end`, 10,000 distinct `run` ids, 10,000 distinct `pid`s, 10,000 distinct `run|call` pairings |

**Not one line of 20,000 was torn, and the app's own decoder skipped none.**
The longest line is 200 bytes, well under the 512 bytes of `PIPE_BUF` that the
atomicity of a single `write` to an `O_APPEND` file descriptor rests on, which
is why one open, one write and one close is the whole of the writer.

Two things the numbers say in passing. One run id and one pid per process, with
`call` counting inside the process, is what tells 10,000 runs apart: nothing in
the file is a sequence number across processes. And the events were written at
`2026-10-16T23:59Z` and later while the machine's local date was 2026-10-17, so
the file they landed in is `2026-10.jsonl` by the event's UTC month, as the
package says.

Recheck when the writer stops being one open and one write, or when a line can
grow past 512 bytes, which the 200-rune title cap is what holds today. A config
directory on a network or a synced volume is out of scope: the local
filesystem's promise is the only one measured.

## The bundle swapped under a process running out of it

Measured 2026-10-17. After `gdoc update` replaces the app, the app on disk is
a different bundle from the one the person is looking at, and the design has it
notice and reopen itself. This measures the half that can be measured with no
GUI app: that the swap does not kill a process running out of the old bundle,
and that the plist at the destination names the new version afterwards.

How it was taken. A bundle-shaped folder was made at
`/tmp/m23-swap/Applications/GdocPresence.app`: `Contents/MacOS/GdocPresence` a
Go program whose whole body is `time.Sleep(900 * time.Second)`, ad-hoc signed,
and a `Contents/Info.plist` naming `com.altery.gdoc.presence` at `9.9.9`. It
was started with `nohup`, confirmed alive with `kill -0` and confirmed by `ps`
to be running from that path. Then `ApplyBundle` was called from a throwaway
test in `go/internal/update` with a real `v9.9.10` zip, built by `bundle.sh`
and packed by `ditto`, at the same destination.

| What was asked | Answer |
|---|---|
| the version before | `9.9.9`, read by `BundleVersion`, which reads the plist |
| the version after | `9.9.10` |
| what was kept | `GdocPresence.app.previous`, whose plist reads `9.9.9` |
| is the process alive | `kill -0` yes, and `ps -o comm=` names the same path it was started from |
| which file is the process actually running | `lsof -p <pid>` `txt` line: `/private/tmp/m23-swap/Applications/GdocPresence.app.previous/Contents/MacOS/GdocPresence`, 1,893,936 bytes |
| what is at the live path now | the release's executable, a different sha256 from the running one |

**The swap keeps the process running, and the kernel follows it into
`.previous`.** The running executable is an open file, and renaming a folder it
sits in does not touch the open file. So the app that is running after an
update is the previous bundle's binary, reachable only through `.previous`,
which is exactly why it has to notice and reopen its own path rather than
re-exec the file it came from.

**One thing worth writing down: a bundle already carrying a plist refused to be
written into.** `touch`, `cp` and `go build -o` into
`GdocPresence.app/Contents/MacOS/` all failed with `Operation not permitted`
once the folder held an `Info.plist`, from this session, while the same writes
into the same folder before the plist existed succeeded. The cause was not
isolated here: it is either macOS App Management protection over a folder it
reads as an app, or this session's own sandbox. It does not reach the updater's
route either way, which never writes into an existing bundle: it writes
`<dest>.new`, a folder that is not an app, reads it back, sums it, and swaps by
rename. Both the rename sequence and the `.previous` rename worked in the
measurement above.

Recheck the live half, which is not measured and is Nail's: with the app
running, `gdoc update` into a newer release, then watch for the reopen row and
a single instance afterwards. If the app does not reopen itself, the swap is
still safe and the person restarts it by hand, and the self-relaunch is M24's.

**The live half, measured 2026-10-17 at 09:25.** The installed 2.14.0 was
running (pid 84014, started 09:24:42). A 2.14.1 bundle built by
`presence/bundle.sh` replaced it by two renames, `GdocPresence.app` to
`GdocPresence.app.previous` and the new tree into place, at 09:25:01. Eight
seconds later the old process was still alive, `pgrep` counted one instance,
and `plutil -extract CFBundleShortVersionString raw` on the path read 2.14.1
while the process out of it was 2.14.0. The unified log carried no fault for
the process. So a rename swap under a running instance is safe, and an
instance that reads its own path's plist sees the new version: that is the
fact a self-relaunch rests on, and M24 builds it on this measurement. The
2.14.0 bundle was put back and the app quit afterwards.

## Drive's import of a heading's bookmark

Measured 2026-10-16 by hand, from one probe note of three headings, one `#`
link and one page break, built with `gdoc build` and uploaded with conversion
into the test folder six times, once per variant. Nothing was trashed: the six
documents are `toc-headingid-probe-a` to `toc-headingid-probe-f` there, and
they are the evidence this section points at. The reads are `documents.get`
through `docs.URL`, the read every command makes.

The question came from a document of Nail's, "Embedded retail card concept
paper v0.9". After a refresh of the contents list in Google Docs, every entry's
hover card said "Tab 1" and a click landed on the title page rather than on the
heading the entry names.

Recheck when Google's docx import changes, when Word's treatment of a hidden
bookmark changes, and before anything else rests on a contents entry opening
its heading.

**First, the survey: every document gdoc has published has it, and it is not
new with M19.**

| Document | Headings | With a heading id | Contents entries | Entry link shape |
|---|---|---|---|---|
| v0.9, `1AbUOmAgEJdyCZdHFKdSlX8FrbCfQlEpTWSRAloaIHV0` | 28 | 0 | 28 | `heading: {tabId: t.0}`, no `id` |
| v0.5, 13 October | 19 | 0 | 19 | the same |
| v0.4, 13 October | 16 | 0 | 16 | the same |
| `toc-font-experiment-copy`, 19 September | 12 | 0 | 12 | the same |
| Nail's "Group Application Development Policy V4.0", written in Docs | 85 | 85 | 20 | `heading: {id: h.…, tabId: t.0}` |

A contents entry is a link to a heading by its id. gdoc's headings come back
from Drive's import with no `headingId` at all, so the refresh writes a link
with the tab and no id, and Docs resolves that to the start of the tab. A
document written in Docs has an id on every heading, so its entries open.

**Then the cause, one variant at a time.** Drive's import mints a heading id
for a heading paragraph only when the paragraph carries a bookmark whose name
opens with `_`. That is the convention Google's own docx export uses: every
heading gets `<w:bookmarkStart w:name="_2csi3wz2phe8"/><w:bookmarkEnd/>` before
its first run, and a link to the heading is
`<w:hyperlink w:anchor="_2csi3wz2phe8">`. Word treats a bookmark opening with
`_` as hidden, so it shows in no bookmark list there either.

| Variant | What each heading carried | Heading ids | The `#` link came back as |
|---|---|---|---|
| a, as gdoc built it until this milestone (`1ODMFgBjoDJktG7DtbCUroUhtEPLcju-EEB5UvMoH9FE`) | nothing, except `h_who_decides` wrapping the linked heading | none | `link.bookmark` |
| b | an empty `_probeNNN` pair before the runs, on the headings with no bookmark | the two with the pair; the linked heading still none | `link.bookmark` |
| c | an empty `_probeNNN` pair on every heading, beside the `h_` one | all three | `link.bookmark` |
| d | an empty `probeNNN` pair, no underscore | none | `link.bookmark` |
| e | `_probeNNN` wrapping the runs, like the `h_` one | the two that carried it | `link.bookmark` |
| f, the shape M21 builds (`118is97pMYpL14kgtolRM3JuBlFup5B8rTZD5Pr8Ah5s`) | one empty `_h_<slug>` pair before the runs, on every heading, and the `#` link anchored at it | all three | `link.heading` with that heading's id |

So the leading underscore is the trigger, the range does not matter, the name
is not kept (Google mints its own `h.` id), and a jump anchored at the hidden
bookmark becomes a heading link, which `internal/docs` already decodes and
`internal/view` already prints as `#` and the heading's words.

**An existing document cannot be repaired through the API.** Re-applying
`namedStyleType: HEADING_1` on variant a's headings through
`updateParagraphStyle` mints no id: the read after it carries the same 0 of 3.
The id is minted at import and nowhere else, so the route for a document
already published is a fresh publish from its note, which creates a new
document.

**Nail's gate.** The one step the measuring session could not take was the
click in the Docs UI, because the browser pane there was not signed in. Nail
opened variant f on 2026-10-16, refreshed the contents list and checked it: an
entry names its heading and opens it, no bookmark flag sits beside any heading,
and the `#` jump lands. The decision is DECISIONS.md, 2026-10-16, "Every
heading carries one hidden bookmark".

## What Claude Code puts in a child's environment

Measured 2026-10-17. The record says who ran a command, and the panel draws
the answer. A terminal run and a chat call were already told apart by a mark on
the context; a run Claude Code's own Bash tool starts is neither, and the only
way to know it from inside the process is the environment it was handed.

How it was taken. One line through the Bash tool of a Claude Code session, the
desktop app at 2.42304.0 running the CLI at 2.1.289, from the M24b worktree:
`env | grep -i -E "claude|anthropic"`, with the values of anything token-shaped
dropped before they were read. The session was the ralphex run of this
milestone, so the child was a plain `/bin/zsh` the tool started, not a shell a
person had typed in.

| What was asked | Answer |
|---|---|
| is `CLAUDECODE` in the child's environment | yes, `CLAUDECODE=1` |
| what else names Claude Code | `CLAUDE_CODE_ENTRYPOINT=sdk-cli`, `CLAUDE_CODE_SESSION_ID`, `CLAUDE_CODE_EXECPATH`, `AI_AGENT=claude-code_2-1-289_agent` and a dozen more `CLAUDE_CODE_*` |
| is the account in there | yes, `CLAUDE_CODE_USER_EMAIL`, which gdoc reads nothing from and never will: no account reaches the record |
| is a token in there | `CLAUDE_CODE_MESSAGING_TOKEN` and `CLAUDE_CODE_OAUTH_SCOPES` are, so nothing but the one variable is read |

**`CLAUDECODE` is the one variable gdoc reads, and any value of it counts.**
It is the shortest of the set and the only one whose whole meaning is "Claude
Code started this", so a release that renames an entrypoint or bumps a version
does not move it. gdoc compares it with the empty string rather than with `1`:
the variable is Claude Code's own, and a record that said terminal because the
value had become `true` would be wrong about the one thing the field is for.
`cmd/gdoc`'s `TestViaIsCodeUnderClaudeCode` states both halves, and
`internal/presence` reads no environment at all.

**What this does not say.** It is one session on one machine on one day. A
Claude Code release that stops setting the variable makes every run a terminal
run again, which is the state before M24b and costs the panel one word, not a
wrong one. Nothing gdoc does depends on the answer: the variable decides one
field of one line of a file the binary never reads back.
## What ImageRenderer draws for a SwiftUI ScrollView

Measured 2026-10-17 while building the panel's scroll box (M24a Task 7). The
panel is capped at 600 pt and an opened section scrolls inside itself, so the
box is a `ScrollView`; the panel's tests and its scene renders all go through
`ImageRenderer`. The question is whether a rendered panel shows the rows inside
such a box at all.

How it was taken. A throwaway `XCTestCase` in `presence/Tests/PresenceUITests`
built forty rows of `Text` in a `VStack`, each 200 by 20 pt over a red
background, and drew them four ways through `ImageRenderer` at scale 1 in the
dark appearance: in a `ScrollView` with no height given, in a plain stack with
no height given, in a `ScrollView` framed at 200 by 236, and in a plain stack
framed at 200 by 236 with `.clipped()`. A fifth draw put a single row in the
framed `ScrollView`. Each image was read back as an `NSBitmapImageRep` and its
pixels with an alpha over 0.05 counted. The file was deleted after the reading.

| What was drawn | Image | Inked pixels |
|---|---|---|
| `ScrollView` of forty rows, width 200, no height | 200x1600 | 0 |
| the same forty rows in a plain `VStack` | 200x1600 | 160,000, which is every pixel of the window counted |
| `ScrollView` of forty rows, framed 200x236 | 200x236 | 0 |
| `ScrollView` of one row, framed 200x236 | 200x236 | 0 |
| forty rows framed 200x236 and clipped | 200x236 | 47,200, which is every pixel |

At scale 2 the same two 236 pt boxes came back as a 23,910-byte PNG for the
stack and a 3,467-byte PNG for the scroller.

**`ImageRenderer` lays a `ScrollView` out and draws it empty.** The box takes
the size it should, and its content takes the content's size, so every
measurement through a preference key still holds; nothing inside it reaches the
canvas. One row is as empty as forty, so this is not clipping or a scroll
offset: the renderer does not draw the scroller's content at all.

So the scroll box carried a drawn mode: with `\.isRendering` on it clipped a
plain stack instead of scrolling, and the scenes turned that on with one
`rendering()` at the top of the tree.

The box itself went on 2026-10-18 with the third look, and with it the
assertion that watched this answer. Nothing in the panel is a `ScrollView`
any more, so the measurement is a record of what `ImageRenderer` does and no
longer something the panel rests on. `rendering()` stays, for the one frame a
render is: the ring reads it so a scene does not come out at whatever angle
the animation was at. The day a scroller is wanted again, this section is
what says to measure it first.

## What LaunchServices says about a `claude://` handler

2026-10-17, M24a Task 17. The panel's "Continue with Claude" is one
activation of Claude Desktop and no URL, because the spec holds the prefill
back until a scheme is measured. This is the measurement, taken by reading the
LaunchServices database and launching nothing:

```
/System/Library/Frameworks/CoreServices.framework/Frameworks/\
LaunchServices.framework/Support/lsregister -dump
```

macOS 26.3.1 (25D771280a), Claude 2.42304.0 at `/Applications/Claude.app`,
bundle identifier `com.anthropic.claudefordesktop`.

| What | Answer |
|---|---|
| Is a `claude:` scheme claimed? | yes, by Claude.app: `claimed schemes: claude:, msauth.com.anthropic.claudefordesktop:` |
| Its rank and role | `rank: Default`, `roles: Viewer (0000000000000002)`, `flags: url-type` |
| Its bindings | one, `claude:` |
| What the bundle's own plist says | `CFBundleURLTypes` holds two entries, `Claude` with scheme `claude` and `MSAL` with scheme `msauth.com.anthropic.claudefordesktop` |
| Does anything say what a `claude:` URL means? | no: the plist names the scheme and nothing else, and no host, path or parameter of it is documented |

**A handler exists and a grammar does not.** The scheme is registered, so a
`claude://...` URL would reach Claude Desktop rather than nothing. What it
would then do is unknown: nothing in the bundle, and nothing published, says
which host or path opens a chat, let alone one with text already in it. A URL
built on a guess is a URL that may do something else on the next release.

So "Continue with Claude" stays one activation: `NSRunningApplication.activate`
where it is up, and `NSWorkspace.open` on its bundle where it is not, in
`presence/Sources/PresenceApp/Opener.swift`, with no argument, no file and no
URL. `OpenerTests.testContinueWithClaudeOnlyActivatesTheApp` holds that, and
`SourceRulesTests.testTheOnlySchemesAndHostsAreGooglesAndTheVaults` holds that
`claude` is not one of the two schemes the sources may name.

Recheck when Anthropic documents the scheme, or when a release note names a
URL that opens a chat. The prefill is then one decision entry and one line in
the opener, and the scheme becomes the third the source scan admits.

## What proc_pidpath says after the binary is renamed

Measured 2026-10-17 while building the panel's Stop (M24a Task 18). Stop may
signal a pid only when the kernel says that pid is running the gdoc the app
knows, and `gdoc update` renames the running binary to `gdoc.previous` before
it writes the new one. So the question is what `proc_pidpath` answers for a
process whose own file was renamed under it: the old name, the new one, or
nothing.

How it was taken. A throwaway Swift script outside the repository, run with
`swift measure2.swift`, compiled a four-line C program that sleeps thirty
seconds, started it with `posix_spawn` from
`/tmp/pidpath-measure/room2/waiter`, waited 400 ms, read `proc_pidpath` and
`proc_pidinfo(PROC_PIDTBSDINFO)`, renamed the file to `waiter.previous`, and
read both again at once and two seconds later. The subject is ad-hoc
linker-signed (`codesign -dv`: `flags=0x20002(adhoc,linker-signed)`), which is
what `~/.local/bin/gdoc` is too. macOS 26.3.1 (25D771280a), arm64. The script
and its directory were deleted after the reading.

| When | `kill(pid, 0)` | `proc_pidpath` |
|---|---|---|
| before the rename | alive | 41 bytes, `/private/tmp/pidpath-measure/room2/waiter` |
| 400 ms after the rename | alive | 50 bytes, `/private/tmp/pidpath-measure/room2/waiter.previous` |
| two seconds after the rename | alive | the same 50 bytes |

**The kernel answers the file's name now, not the name it was started from.**
The process keeps running, and `proc_pidinfo` keeps answering uid 501 and the
same start time. So a wait that began before an update reads as
`<path>/gdoc.previous`, which is exactly the one extra name
`ProcessCheck.previousSuffix` admits, and nothing wider is needed: the suffix
is compared whole, so `gdoc.previously` is still refused
(`ProcessCheckTests.testPreviousCountsAsGdocAndPathsAreResolvedBeforeComparing`).

Two things fell out of the same run and are written down because the check
rests on them:

- A copy of `/bin/sleep` cannot be the subject. `posix_spawn` of a copied
  platform binary returned 0 and the process was then killed by the kernel
  with signal 9, so the first reading of it came back `ESRCH`. That is why the
  measurement compiles its own binary.
- A process that has died and not been reaped still answers `kill(pid, 0)`
  with yes while `proc_pidpath` answers `ESRCH`. The check reads an absent
  path as `pathUnknown` and refuses, so a zombie is never signalled; the
  waits the panel sees are never this app's children, so their own parent
  reaps them.

Recheck on a macOS upgrade, or if `gdoc update` ever stops renaming the
running binary. `StopperTests` holds the reading that depends on this: the
kernel's path is compared, never the record's `gdoc` field.

## The lock, `open -g`, and a command whose opener cannot start anything

Measured 2026-10-18, by hand on Nail's Mac (Darwin 25.3.0, arm64), against the
bundle this branch builds and the binary this branch builds: `make app` wrote
`bin/GdocPresence.app`, version 0.0.0, universal and ad-hoc signed, and `make
build` wrote `bin/gdoc`. The installed bundle at
`~/Applications/GdocPresence.app` is v2.15.0 and takes no lock, so it could not
be the subject. It was stopped with `kill` for these readings and opened again
afterwards, and the one trace left in the real presence directory is the empty
`app.lock` the new app creates. No byte of the record was read.

Every reading came from one `python3` process driving `/usr/bin/open`, `pgrep`,
`/usr/bin/lsappinfo` and `flock` through `fcntl`, so the timings are off one
clock. The app side first:

| What was asked | The answer |
|---|---|
| `open -g <bundle>` with the app not running | exit 0, nothing on stdout or stderr, and the process at `<bundle>/Contents/MacOS/GdocPresence` alive 0.3 seconds later |
| the lock after that launch | `presence/app.lock` where no file was, 0 bytes, mode 0644, and a non-blocking `flock` for it from another process refused with `Resource temporarily unavailable` |
| the front application across the launch | `loginwindow` before and `loginwindow` after, and `lsappinfo` names no GdocPresence as a front application at all, which is what `LSUIElement` in the bundle's `Info.plist` buys |
| `open -g <bundle>` again while it runs | exit 0, the same single pid, the front application unchanged |
| `open <bundle>` without the flag while it runs | exit 0, the same pid, and the front application unchanged as well |
| a second instance started past LaunchServices, by running the executable directly | exit 0 after 0.1 seconds with no output, and the first instance untouched |
| the lock after `kill -9` of the app | free 0.032 seconds after the signal, with the file still there and still empty |

Two of those rows need a sentence. The display was locked while this was
measured, so `loginwindow` was the front application throughout and the flag's
effect cannot be told apart from no flag by this reading. What the two rows do
say is that `open -g` started the app and brought nothing forward, and that an
app marked `LSUIElement` is not something the window server will put in front.
The second-instance row is `AppLock.decide` on a real kernel: an instance that
does not get the lock leaves at once rather than drawing a second glyph.

Then the gdoc side, five runs of `bin/gdoc read <a url whose id is no
document>` under `presence: true`, with `HOME` and `GDOC_CONFIG_DIR` pointed at
a temporary directory so the opener's three questions could be set one at a
time. That command fails at the token before it reaches the network, which is
what makes it a cheap subject: its answer is one line, and the same line every
time.

| The three questions | The answer |
|---|---|
| no bundle under the temporary home | nothing started, and stdout was the same line as the same run with the setting off, stderr empty, exit 1 |
| a bundle `open` cannot run, an `.app` directory with no executable | the opener was issued and the child failed: `The application cannot be opened because its executable is missing` and exit 1 from `open`. gdoc printed the same line, left stderr empty and exited 1 |
| the bundle there and no lock file | the app was running 0.2 seconds after the command returned, out of the temporary home's copy, and the answer was the same line again |
| the bundle there and the lock held by another process | nothing started in the two seconds after, and the same line again |
| the same run once the holder let go | the app started again |

One sentence of the mechanism is not measured here. Remote Login is off on this
machine and the account has no administrator rights, so neither `ssh localhost`
nor `launchctl asuser` could be run, and a shell with no window session was not
reached. The broken-bundle row is what stands in for it: an `open` that fails
costs the command nothing, because the child is started with no stderr of its
own and its exit status is never read. Recheck the real thing with Remote Login
on, by `ssh localhost` running one reporting command under the setting and
comparing the object and stderr against the same command in a terminal.

Nothing measured above contradicts the mechanism this milestone is built on.

Recheck all of it on a macOS upgrade, if `open` ever changes what `-g` does, or
if the app stops taking the lock as its first act.

## Not measured yet

One of the M14 measurements stayed open on 2026-10-03: whether Google emails an
address named in a comment that gdoc posts (12). Recheck it with one `gdoc
annotate` on a document in the test folder whose text names an address the
person can read, before anything gdoc says to a person promises or denies an
email. The phone halves of 8 and 10 in "Claude Desktop and a local MCP server"
are answered by "gdoc mcp in Claude Desktop, the first run": voice mode does
not reach the extension, so there is no card on a phone to measure.

The seven paragraph elements `internal/docs` decodes are a fixture built from
the reference, not from a document. Recheck when somebody reads a real document
holding a person chip, a date chip and a calendar link through `documents.get`
and compares the answer against `go/internal/docs/testdata/elements.json`; a
difference is recorded as its own section here, and the fixture is corrected
rather than a test loosened. The decision is DECISIONS.md, 2026-09-09, "Seven
paragraph elements were dropped at decode".

Three things `export` rests on are not measured either. The fixture document
Task 1 of the M13 plan describes has not been made, so
`TestLiveExportMeasurements` has not run: it is written, gated on
`GDOC_LIVE_TEST` with `GDOC_LIVE_EXPORT_DOC_ID`, and it prints its answers
rather than asserting them, because they are the measurement.

| What | Why it matters | What the test prints |
|---|---|---|
| Whether the Docs read and the docx export list a document's pictures in the same order | the pairing between the two routes is order and nothing else, DECISIONS.md 2026-09-19 | every inline and positioned object id per tab in body order, beside every `r:embed` in `word/document.xml` order with its media part name |
| Whether a PNG that `publish` uploaded comes back from the docx export byte for byte | `Options.MatchByHash` keeps a note's own picture file, so the SVG it was rendered from stays the master. It ships `false` until this holds | the sha256 of the first media file against the sha256 of the uploaded PNG on disk |
| What the docx export carries for a document's second tab | a tabbed document exports one file per tab, and a picture has to follow its own tab | the count of `w:drawing` in the docx against the count of objects in the first tab and in the second |

Recheck by making the fixture document in the test folder: one inline PNG
picture, one Google Drawing, one floating picture, and a second tab titled
`Appendix` with one picture, in that order. A difference is its own section
here. If the second answer says the bytes are equal, `MatchByHash` is flipped to
`true` in one commit with that section, and `TestLiveExportHashEquality` turns
from a print into an assertion.

## What `claude://code/continue` does from outside the app

2026-10-18, M26. The strings of `/Applications/Claude.app/Contents/Resources/app.asar`
(Claude 2.42304.0) carry three Code links: `claude://code/continue?session=last&source=desktop_action`,
`claude://code/new?source=desktop_action` and
`claude://code/needs-input?source=desktop_action`. Whether the first opens a
named session when another process issues it was measured by Nail watching
the window while this session ran `open` on each form, Claude Desktop
already running and showing a Code session:

| Link issued | What the window did |
|---|---|
| `claude://code/continue?session=<this session's id>&source=gdoc` | came to the front, stayed on the session it showed |
| `claude://code/continue?session=<the publishing session's id>&source=desktop_action` | came to the front, stayed |
| `claude://code/continue?session=last&source=desktop_action` | came to the front, stayed |

**From outside the app the path does nothing but activate it.** The link is
the desktop app's own, raised from its notifications and menus, and whatever
makes it switch there is not the URL alone. So Open chat keeps the
activation it has, the `session` field still goes into the record under
`via: code`, and the day a form is found that switches, the opener has the
id. Recheck on a Claude Desktop update by the same three lines.

## Where the screen cuts a panel as tall as its content

Measured 2026-10-18 on Nail's machine, M27 Task 1. Nothing in the panel
scrolls since that evening and it has no cap, so a busy day draws a panel
taller than the screen. The decision says that cut is the price Nail chose
over any scroll, and this is the height it happens at.

How it was taken. A throwaway `XCTestCase` in `presence/Tests/PresenceUITests`
built scene e3's panel with the hold card of scene r6 repeated 1, 4, 8 and 12
times in NOW, read each panel's height back through the `panel` metric probe
and off `NSHostingView.fittingSize`, and asked `NSScreen.main` for its frames.
It then placed a borderless `NSPanel` holding the twelve-card panel by the
arithmetic `presence/Sources/Presence/Panel.swift` uses, with the status
item's button taken as the top right of the screen. The file was deleted after
the reading.

| What was measured | Height |
|---|---|
| this screen, `NSScreen.main.frame` | 1728 x 1117 |
| the same screen's `visibleFrame`, which is what the panel is placed in | 1675 x 1084 |
| e3 with one hold card | 1092 pt |
| e3 with four | 1512 pt |
| e3 with eight | 2072 pt |
| e3 with twelve | 2632 pt |
| the tallest scene as the scenes stand, e3 | 1552 pt |
| the next two, e1 and r9 | 1184 pt and 1124 pt |
| the shortest, e8 | 372 pt |

The twelve-card panel asked for 320 x 2632 and the window kept all of it:
`setContentSize` is not clamped for a borderless panel. The placement is what
decides which end is lost. It hangs the panel under the button and then
raises the origin to `visibleFrame.minY + 8`, so the foot sits 8 pt above the
screen's bottom edge, 1076 pt of the panel are on the screen and **1556 pt
are above the screen's top edge**, where the window server shows nothing.

**A panel taller than the screen loses its head, not its foot.** SETTINGS and
Quit stay on the screen because the bottom is pinned; the header, the account
and NOW go off the top. A panel of 1084 pt or less is whole, which every scene
in the set is except e3 at 1552, and a day with one hold card in it is already
past that on this screen.

The window server's own bounds for that window could not be read from the test
process: `CGWindowListCopyWindowInfo` reported 0 x 0 and no on-screen flag,
because a `swift test` process composites no window. So the cut above is
AppKit's geometry and the screen's frame, not a screenshot.

Recheck on a screen of another size, or if the placement in `Panel.swift`
changes which edge it pins. A later decision may cap NOW, which the 2026-10-18
entry leaves open; that would be the one number to measure again.

## What Spotlight answers for a document id

Measured 2026-10-18 on Nail's machine, M27 Task 4. The hub folder was one
setting for a thing that differs by document, so the note is found now: the
app asks Spotlight for the Markdown files whose text holds the id and reads
the `gdoc:` front-matter block of each. This is the query it runs, what it
takes, and what comes back that is not the note.

How it was taken. Three `mdfind -onlyin ~` runs of each form under
`/usr/bin/time -p`, over the real vault at
`~/src/altery/Altery-Platform-Hub`, for the id
`1gH4y7uvT4QgpZBCHHlHz9g5L9R0R7dzONs9cJOTcY0c`, which one note's front matter
names. Then a throwaway `swiftc` program ran the chosen form through
`NSMetadataQuery` with `operationQueue` set and no run loop, to prove the app
can ask the same question off the main actor. Both the program and the files
below were deleted after the reading.

| Query form | Seconds | Answered |
|---|---|---|
| `<id>`, mdfind's own parser | 0.10–0.14 | the three files |
| `kMDItemTextContent == "*<id>*"` | 1.51–1.53 | the three files |
| `kMDItemTextContent == "*<id>*" && kMDItemFSName == "*.md"` | 1.62–1.67 | the three files |
| `kMDItemTextContent = "<id>"cdw` | 0.10–0.11 | the three files |
| `kMDItemTextContent = "<id>"cdw && kMDItemFSName = "*.md"c` | 0.24–0.26 | the three files |
| the last form through `NSMetadataQuery`, no run loop | 0.23 | the three files |

**The word form is six times the substring form's speed over the same index.**
`cdw` matches the id as a word, which the index can answer from; `*<id>*` is a
substring scan. The app runs the last form, which is the one with the Markdown
clause, and `SpotlightSearch.queryString` writes it:
testTheQueryIsTheWordFormOverMarkdown.

**Three files, and the front-matter check keeps one.** Two of the three were
written for this measurement in `~/m27-spotlight-measure`: `document-map.md`,
which names the id in a sentence the way Nail's own document maps do, and
`decoy-key.md`, whose front matter holds `document_id` under a key that is not
`gdoc:`. Spotlight returns both, because it answers on a file's text. The
reading `HubIndex.documentIDs(inFrontMatterOf:)` already did for a whole hub
throws both away and keeps
`11-m2-crypto-project-eagle/2026-09-15-eagle-roles-and-responsibilities.md`:
testAFileThatMerelyMentionsTheIdIsNotTheNote.

**Not every Markdown file under the vault is indexed.** `mdfind -onlyin <vault>
'kMDItemFSName == "*.md"'` answered 727 files where `find` counts 1764. Some of
the difference is dot folders, which `find` walks and Spotlight does not, and
the rest was not chased. It is the reason the `hubPath` override stays: a
person whose notes Spotlight has not indexed still has a folder to name.

The ceiling is two seconds, `NoteFinder.ceiling`, which is eight times the
measured query. Nothing waits on it but one greyed icon, and nothing is written
down for a query that hit it. The clock asks about that document again a minute
later, and a panel somebody opens asks at once.

Recheck after a Spotlight reindex, or on a vault an order of magnitude bigger
than 727 files. The timings are one machine's and the shape of the answer is
what matters: a handful of candidates, one of which the front matter proves.

## What the header line holds, and what a second version costs it

Measured 2026-10-18 on this machine, M27 Task 5, through the metric probes
`RowTests` already reads. The decision of the third look asks for the newer
version written after the current one, `v2.18.0 → v2.19.0`, on the header
line. The line does not have the room.

How it was taken. `RowTests.testTheHeaderIs36WithTheMarkTheNameTheVersionAndTheAccount`
renders `PanelHeader` at the width the panel gives it, 320 less 8 either
side, and reads the address's own box back twice: once drawn alone at the
width it asks for, and once on the header line. The two numbers below are
that comparison, and both are assertions in that test rather than a reading
somebody wrote down.

| The header line | The address's width |
|---|---|
| mark, name, one version, the update glyph, the account's glyph, its address and Sign out | 98 pt, which is the whole of `nail@example.com` |
| the same line with `v2.16.0 → v2.17.0` in the version's place | 49.5 pt, which is `nail@ex…` |

**So the published version is the line under the header and not a second
number on it.** The address is the piece that gives, because it is the one
thing on the line drawn without `fixedSize`, and an address cut in half is
what the morning of the same day was about. The glyph keeps its accent ink
and its dot, which is what catches an eye; the number is in the quiet line
directly under the installed one, as `v2.15.0 is published`.

Nail may still want both on the line. That is a new decision entry and one
of these: a shorter address, no mark, or the name dropped. Dropping the
leading `v` from both versions buys 12 pt of the 49 needed, so it is not
one of them.
