# How AI skills should talk to gdoc: a design review

Date: 2026-10-27. Written for Nail, as a review by a senior engineer who builds
agent integrations.

Inputs: the four notes in this folder (`baseline_today.md`,
`cli_mcp_code_debate.md`, `claude_and_mcp_surfaces.md`,
`tool_makers_case_studies.md`), `PRINCIPLES.md`, `CLAUDE.md`,
`skills/gdoc-review/` (`SKILL.md`, `review.md`, `live.md`),
`go/internal/mcp/doc.go`, `go/internal/chat/doc.go` and
`go/internal/propose/doc.go`. Nothing in the repository was changed, and
nothing ran against Drive.

How to read the numbers:

- **Measured** means a number from `baseline_today.md`, with its tag: [PRES]
  the presence record, [TX] Claude Code transcripts, [HELP], [FILES], [MCP].
- **est.** means my estimate. The arithmetic stands next to it.
- Today's rules are not constraints in this review, as asked. Each idea names
  the rules it would break, with these codes:
  - **R1** one JSON object, then exit
  - **R2** the binary prints facts, the skills judge
  - **R3** the binary never prompts
  - **R4** the guard and its grants die with the process
  - **R5** no watcher: the loop starts when Nail starts it
  - **Other** any other invariant in `CLAUDE.md` or principle in `PRINCIPLES.md`

---

## Summary

The time and the tokens are not in gdoc and not in the transport. They are in
the number of model turns, and in how much context each turn re-reads.

- A live window is 75% model time. About half of its turns only fetch facts in
  an order the skill fixes.
- A third of live wake-ups find nothing to do.
- Every turn re-reads 257k to 725k tokens, because the loop runs inside a big
  session.

So the target is not "a daemon" or "an MCP server". It is three things, in
this order of payoff:

1. **Fewer turns per job.** Job-sized commands that hand the model a ready
   work packet, and take its whole answer back in one call.
2. **No turns for nothing.** A watch that wakes the model only for real work.
3. **Small context for routine work.** The live loop and each window run in a
   small, fresh context, not inside the hub session.

| Rank | Idea | Comment to reply, est. | Tokens per live hour, est. | Build |
|---|---|---|---|---|
| today | skills over the one-shot CLI | 55 s | 8.6M | |
| 1 | A. Job-sized commands with work packets | 25 to 35 s | 5.6M | M |
| 2 | K. Live review in its own small session | about 50 s | 5.8M alone; about 1.2M with A, H and I | S |
| 3 | H. A quiet watch through Claude Code monitors and hooks | 50 to 55 s alone | 5.4 to 6.2M | S to M |
| 4 | G. Fast path in Go, the fact-only parts | about 45 s | 6.8M | S to M |
| 5 | C. `gdoc mcp` as the one route, with a watch that pushes | 23 to 33 s with A | 3.6 to 4.8M with A | M to L |
| 6 | F. No open session: a watcher wakes a fresh headless agent | 30 to 40 s with A | about 0.6M | M, needs Nail's decision |
| 7 | I. A small responder subagent per window | about 30 s with A | 6.0M alone | S to M |
| 8 | G. The automatic acknowledgment | first sign at 3 to 5 s | no change | S, needs Nail's decision |
| 9 | E. gdoc runs its own small agent | 15 to 25 s | about 0.2M plus API | XL |
| 10 | J. The document as local files | about 50 s | 8.0M | M |
| 11 | D. Code execution against a gdoc API | 30 to 40 s | 5.6M | L |
| 12 | B. A resident daemon with thin clients | 52 to 54 s | 8.6M | L |

Recommended path: Stage 1 this week (a `--follow` watch under the Monitor tool,
an own-echo filter, a short help, and live review in its own session). Then
job-sized commands, then a small responder, then the MCP route for Claude Code
as well as Desktop. The headless watcher (F) is the step that removes the big
session entirely. It breaks R5 and needs Nail's decision. Section 5 has the
stages and what to measure.

---

## 1. Diagnosis

### 1.1 Two constants explain most of the cost

**T, model time per turn, is about 8 s.**

- Arithmetic: 35.6 s of model time per live window (measured median) divided
  by 4.5 turns per work window (measured, 4 to 5) = 7.9 s.
- It agrees with the measured model time before each gdoc call: medians 6.4 s,
  7.2 s and 9.1 s in three sessions [TX].
- Most of a turn is writing. The median turn writes 384 to 687 output tokens
  [TX]. At an assumed 50 to 80 output tokens per second, 500 tokens take 6 to
  10 s (est.).
- So speed depends on how many turns a job takes and how much each turn
  writes. Model choice matters too: a faster model writes faster.

**C, the context each turn re-reads, is 257k to 725k tokens.**

- Measured medians per API call: 445k (f942, a mixed session: publish, live
  review, align, Jira), 317k (543f) and 258k (fd18), both dedicated live
  reviews [TX]. No-op rounds read 363k to 725k [TX].
- Every turn pays C, whether it drafts an answer or decides that an echo of
  its own reply is not work.
- I use C = 400k as a middle value below, and say where 725k changes a
  conclusion.

**What is not a lever.** gdoc's own bytes are about 1% of what a turn re-reads
(baseline §2). The binary starts in 20 to 60 ms [HELP]. Token refresh happens
about once an hour, not per call (baseline §3).

### 1.2 Live review: one comment, from writing to answer

| Step | Time | Source |
|---|---|---|
| Comment posted; the next poll sees it | 2 to 4 s | est. from 2 s poll interval plus about 1.5 to 2 s for the listing (baseline §4) |
| Turn: read the window, call `read` | 8 s + 1.3 s | T; `read` median 1,310 ms [PRES] |
| Turn: call `suggestions` | 8 s + 1.0 s | T; median 985 ms [PRES] |
| Hub search (Grep, Read) | inside model time | [TX] |
| Turn: write proposals file, call `propose` | 8 s + 6.8 s | median 6,797 ms, p90 17.4 s [PRES] |
| Turn: call `comments` again before posting | 8 s + 2.0 s | review.md rule; median 2,003 ms [PRES] |
| Turn: write the body, call `reply` | 8 s + 1.3 s | median 1,309 ms [PRES] |
| **Wait return to last write** | **52 s median, about 90 s p90** | measured over 13 windows; gdoc 12.9 s (25%), model 35.6 s (75%) |
| **Comment to reply** | **about 55 s median** | 3 s detect (est.) + 52 s |

If the comment lands while the model is busy, add the rest of that window and
the measured 6 to 16 s gap before the next wait starts.

**Half the turns are mechanical.** Of the five gdoc calls in a typical window,
three fetch facts in an order the skill prescribes: `read`, `suggestions` and
the `comments` re-read. They carry no judgement. Only `propose` and `reply`
carry the model's work.

**Where the tokens go in a live hour.** A scenario, est.: an hour with three
marked comments, each its own window.

| Kind of turn | Turns | Why | Tokens at C = 400k | At C = 725k |
|---|---|---|---|---|
| Work turns | 13.5 | 3 windows × 4.5 turns | 5.4M | 9.8M |
| Echo wake-ups | 3 | own reply comes back on the first poll | 1.2M | 2.2M |
| Quiet re-arms | 5 | the 9-minute ceiling, about 60/9 minus busy time | 2.0M | 3.6M |
| **Total** | **21.5** | | **8.6M** | **15.6M** |

Cross-check against what was measured: no-op rounds alone were 16% of f942's
cache reads (37 × 725,416 = 26.8M of 166.6M) and 12% of 543f's (18 × 362,928 =
6.5M of 54.2M). Those sessions also did other work, so the share is lower than
in a pure live hour. In the scenario, nothing-turns are 8 of 21.5, or 37%.

**A third of wake-ups find nothing.** 43 of 120 waits came back on the first
poll, and the command before 27 of them was a gdoc write or another wait. 26
ran the full nine minutes [PRES]. Each of those ended in a full-context turn.

### 1.3 One-shot review

- Calls: about 11 gdoc calls for two `ai?`, one `ai!` and one proposal:
  `help`, `comments`, `read`, `suggestions`, 3 × (`comments` + `reply`),
  `propose` (baseline §1). Six of the eleven are mechanical: `help`, `read`,
  `suggestions` and the three re-reads.
- Turns: 15 to 20 (est. from the skill text).
- Time, est.: 17 turns × 8 s = 136 s of model time. gdoc time = 0.03 + 2.0 +
  1.3 + 1.0 + 3 × (2.0 + 1.3) + 6.8 = 21 s. Total about 2.6 minutes, plus hub
  search.
- Tokens, est.: in a fresh session C starts near 50k and grows to about 100k.
  17 turns × 75k average = 1.3M.
- Wasted turns from refusals: `propose` ended `ok:false` at 0 ms in 7 of 34
  runs, `annotate` in 3 of 9 and `publish` in 3 of 8 [PRES]. Each refusal is a
  full turn spent learning the input was wrong.
- Narrowing: 35% to 83% of gdoc calls were piped into `python3 -c` or `head`
  [TX]. The model wrote parsers to cut JSON it did not need.

### 1.4 Publish

- gdoc's part: `publish` median 10.3 s [PRES]. Build and upload start in the
  same second, upload to read-back is 6 to 8 s, read-back to link is 4 to 6 s.
- The skill's part: `help --json`, `build` to a scratch path to read the marks,
  `help publish --json`, `publish`, and `help marks --json` when marks exist.
  That is 4 to 5 gdoc calls, and 4 of them are mechanical.
- Time, est.: 6 to 8 turns × 8 s = 48 to 64 s of model time, plus about 11 s of
  gdoc. About 60 to 75 s from "publish this" to the link on screen. gdoc's
  10 s is the floor, not what Nail waits.
- Measured context: the one publish-only session (a022) spanned 278 s with 23 s
  of gdoc time, person included [PRES]. f942 ran 13 `build` calls for 2
  publishes, as retries while fixing the note [TX].

### 1.5 What the diagnosis says

1. **Speed is turns.** The model is 75% of a live window and about 80% of a
   publish (est.: 56 of 70 s). gdoc is 25% of a window, and most of that is
   `propose`'s three read-backs plus the re-read before each reply.
2. **Tokens are turns × C**, and C is large because the loop lives in a big
   session. Mixed sessions run at 445k to 725k per turn. Dedicated ones at 257k
   to 317k.
3. **Half the turns are mechanical.** They fetch facts in a fixed order.
4. **A third of live wake-ups are empty.**
5. **Transport is not a lever on its own.** A kept connection saves at most
   part of 0.5 to 1 s per call (baseline §3). 2026 re-tests found MCP and CLI
   within a few percent (cli_mcp_code_debate §3).
6. **The safety layer is uneven.** Holds, the ledger and the write memory exist
   only in `gdoc mcp` (baseline §5). The Claude Code route reads the same
   stranger text, holds Bash and Edit, and has none of them.

---

## 2. Eleven architectures

Each idea below has: how it works, a flow you can draw, its effect, the build
cost, the rules it breaks, its risks, and what the industry evidence says.
Effects are against today: **55 s** comment to reply, **60 to 75 s** publish
request to link (10.3 s of it gdoc), **4.5** turns per work window, **8.6M**
tokens per live hour at C = 400k.

### A. Job-sized commands with work packets (the fat CLI)

**How it works.**
1. `gdoc review <url> --work` returns only what needs work: the marked
   threads, each with the paragraph it is anchored to, the pending suggestions
   that overlap that paragraph, and the paired note's path.
2. `comments --wait` windows carry the same packet, so the window is the work.
3. `gdoc respond <url> --from plan.json` takes the model's whole answer for a
   window: replies, receipts, proposals. It re-reads each thread just before
   posting, as a precondition, and runs the read-backs in parallel.
4. `gdoc publish note.md --folder <id>` builds in process and reports the marks
   itself, so no scratch build and no `help` calls.
5. Output is concise by default. `--full` gives everything. A short `help
   --brief` replaces the 9.6 KB preamble.

**Flow.** Person → Docs: writes `ai?`. gdoc wait → Drive: poll every 2 s.
gdoc wait → Docs: one read for the packet. gdoc wait → model: window with
packet. Model → hub: Grep and Read. Model → gdoc respond: plan file. gdoc
respond → Drive: re-read thread, post reply. gdoc respond → Docs: propose,
three read-backs in parallel. gdoc respond → model: one result.

**Effect, est.**
- Turns before the last write: 2 instead of 4 to 5. One turn reads the packet
  and starts the hub search; one writes the plan and calls `respond`. The
  report turn comes after the write.
- Comment to reply: 3 s detect + 1.5 s packet read + 2 × 8 s + 2 s search
  tools + 2 s for the reply inside `respond` = 24.5 s. With a proposal, the
  last write lands about 4 s later. So 25 to 35 s.
- Parallel read-backs: `propose` 6.8 s → about 4 s, if the inline read, the
  preview read and the docx export run at once (the three are independent
  reads per `propose/doc.go`; the export is likely the slowest).
- Tokens per live hour: 3 windows × 3 turns + 0 echo (filtered) + 5 quiet
  re-arms = 14 turns × 400k = 5.6M, down 35%.
- One-shot review: about 4 gdoc calls and 6 to 8 turns instead of 11 and 15
  to 20.
- Publish: turn 1 calls `publish` (8 s), publish runs (8 to 10 s), turn 2
  prints the link (8 s). About 25 to 30 s instead of 60 to 75 s.

**Build.** M. Three new command shapes on the existing packages, a concise
renderer, and a session per read-back so they can run concurrently
(`gapi.Session` is not safe for concurrent use, per `mcp/doc.go`).

**Rules broken.**
- R2, mildly: the echo filter decides that a window holding only gdoc's own
  replies is not worth returning. It can be stated as a fact ("every item opens
  with gdoc's mark"), and a reply that opens with a mark cannot also open with
  a marker, so nothing actionable is dropped. Still a filter in Go.
- Other: none. `publish` already does build, upload and read-back in one call.

**Risks.**
- The workflow moves into Go. A skill edit is live at once; a command change
  needs a release.
- One call can post three wrong replies. Mitigate with `--dry-run` and the
  rule that every posted text is printed in full.
- The packet may leave out what the model needed. It falls back to `read`,
  which is today's cost.
- Injection: no change from today. Packet text is still stranger text.

**Evidence.**
- For: Anthropic says consolidate multi-step workflows into one tool
  (`schedule_event`) and default to concise output (72 vs 206 tokens)
  (cli_mcp_code_debate §1). Vercel cut steps from about 12 to about 7 and time
  from 274.8 s to 77.4 s (§3). Port of Context: creating an invoice took 19
  steps by CLI, 12 by raw MCP, 4 by code mode (tool_makers §6). Notion rewrote
  its tools around Markdown to cut calls (tool_makers §1). Poehnelt's
  agent-native CLI rules (tool_makers §2).
- Against: Mastykarz argues flat flags beat nested JSON payloads for agents
  (tool_makers §2, secondary). That is an argument about the plan file's shape,
  not about coarse commands.

### B. A resident daemon with thin clients

**How it works.**
1. `gdocd` runs per user, started by launchd or on first use, listening on a
   Unix socket in the config directory.
2. It keeps the token, HTTP/2 connections to Google, a document cache keyed by
   revision, and each watch's cursor.
3. The CLI, `gdoc mcp` and the menu bar app become thin clients. Each call is
   still one request and one JSON object back.
4. The guard runs per request inside the daemon, and a grant names its request,
   not its process.
5. With no daemon running, the CLI does the work itself, as today.

**Flow.** Model → gdoc CLI: one call. gdoc CLI → gdocd: socket request. gdocd
→ Google: on a warm connection, or from cache. gdocd → gdoc CLI → model: one
object. Menu bar app ↔ gdocd: status.

**Effect, est.**
- TLS and connection setup saved: about 0.1 to 0.3 s per connection, about 6
  connections per window, so 0.6 to 1.8 s per window. The measured `read`
  minimum of 559 ms bounds this from above (baseline §3).
- Cached `read` when the revision is unchanged: about 1.3 s → 0.4 s (one
  metadata check). Big win only for repeated reads, such as 543f's 17 reads of
  one 56k-character document.
- Comment to reply: 52 to 54 s. Publish: about 0.5 s faster.
- Tokens and turns: unchanged.

**Build.** L. Socket, access control on the socket, lifecycle and crash
recovery, version skew after `gdoc update`, and the guard re-scoped to requests.

**Rules broken.** R4 (grants outlive one command unless re-scoped), R5 if it
polls in the background. Other: a long-lived listener next to the one-shot
login listener; the menu bar app's six argv lists if it talks to the socket;
the presence record's writer changes.

**Risks.** A stale cache under a proposal. A local process that reaches the
socket reaches the Drive token: every daemon in the case studies checks the
caller (1Password checks code signatures, Tailscale uses a socket or a password).
A wedged daemon breaks every client at once.

**Evidence.**
- For: agterm, Obsidian, 1Password, Tailscale and Playwright's named sessions
  all keep state in a long process behind a short CLI call (tool_makers §3).
- Against: none of them publishes a latency gain (tool_makers §3, Gaps), and
  the baseline shows gdoc's share of time is 25%, of which connection setup is
  a fraction. It fixes the wrong cost.

### C. `gdoc mcp` as the one route, with a watch that pushes

**How it works.**
1. The plugin's `mcpServers` starts `gdoc mcp` for Claude Code too, as Claude
   Desktop already does. Claude Code stops calling gdoc through Bash for review
   work.
2. The tools are the job-sized ones from A. The guide, the hold rules, the
   ledger and the write memory now cover Claude Code as well.
3. A `watch` tool returns only actionable windows, with the packet. In Claude
   Code a call still running after two minutes becomes a background task, and
   its result arrives as a notification. A progress notification every minute
   keeps it inside the 30-minute stdio idle limit.
4. Where an org allows it, the same server declares `claude/channel` and pushes
   each window as an event, with no tool call outstanding.
5. Desktop gets the same job tools, and no watch or a watch bounded at 200 s.
   The menu bar app keeps reading the presence record.

**Flow.** Person → Docs: writes `ai?`. gdoc mcp (watch) → Drive: poll. gdoc mcp
→ Claude Code: task notification or channel event with the packet. Model →
hub: Grep, Read. Model → gdoc mcp: `respond` tool. gdoc mcp hold rules →
person: a card when a write trips a rule. gdoc mcp → Docs: writes and
read-backs.

**Effect, est.**
- Comment to reply: same turn count as A. Zechner measured the same tool 23%
  faster as MCP than as a CLI in Claude Code, which he puts down to per-call
  Bash checks. If a fifth of that shows up, 23 to 33 s.
- Quiet hours: 0 turns, because the watch call lasts until there is work.
- Background watch: each watch call costs one small turn when it moves to the
  background. So per live hour 3 × (3 + 1) = 12 turns = 4.8M. With a channel,
  9 turns = 3.6M.
- First use in a session costs one ToolSearch round trip, because MCP tools
  are deferred (claude_and_mcp_surfaces §7).

**Build.** M to L. The server exists. The work is the job tools, the watch
with progress, the plugin wiring, and the channel experiment.

**Rules broken.**
- R1 for Claude Code: review work moves from one-shot processes to a session
  process. The CLI stays for the terminal.
- R5, arguably: the polling lives as long as the session, with no nine-minute
  bound. It is still started by Nail and dies with the session, which is the
  spirit of the 2026-09-07 decision.
- Other: none. The guard per command stays as it is inside `gdoc mcp` today.

**Risks.**
- Channels are a research preview. A custom channel needs a development flag
  with a dialog at each launch, or an org allowlist, and Team and Enterprise
  owners must set `channelsEnabled` (claude_and_mcp_surfaces §3).
- A channel server that negotiates MCP 2026-07-28 cannot deliver events. gdoc
  speaks up to 2025-11-25 today, so this works, but it is a trap for a later
  upgrade.
- The auto-background behaviour is documented but not measured here.
- Claude Desktop was measured dispatching calls 3 to 6 s apart (issue #76854,
  cli_mcp_code_debate §5). Job tools matter even more there.
- The model loses Bash's piping and chaining. That is fine if outputs are
  concise; it is a loss if they are not.

**Evidence.**
- For: same tool, both transports, MCP slightly faster and cheaper (Zechner).
  Practitioners use MCP in Desktop and CLIs in Code, and 2026 re-tests show
  near parity (Checkly; DEV). Playwright keeps MCP for "persistent state" and
  long loops. MCP is the only documented route where an event becomes a turn
  (channels).
- Against: ScaleKit found CLI 4x to 32x cheaper on simple reads, though its MCP
  failures were remote timeouts. Ronacher moved his MCPs to skills.
  Steinberger: "Almost all MCPs really should be clis". Their point is
  definition bloat from large servers. gdoc's eight tools cost about 2k tokens
  and are deferred, so it barely applies.

### D. Code execution against a gdoc API

**How it works.**
1. `gdoc script` runs a short program the model writes, in an embedded
   language with no file or network access, such as Starlark.
2. Each gdoc command is a function that returns the parsed object.
3. The model writes one script to gather (comments since the cursor, one read,
   slices around each anchor, overlapping suggestions) and one to act (post
   the replies and proposals).
4. Variables can persist across scripts in a state file, so a cursor or a read
   is kept without passing through the model.
5. The guard and the read-backs stay, because every call goes through the
   command code.

**Flow.** Model → gdoc script: gather program. gdoc script → Drive and Docs:
the reads. gdoc script → model: one small result. Model → hub: search. Model
→ gdoc script: act program. gdoc script → Docs: writes and read-backs.

**Effect, est.**
- Turns per window: 3, as in A, but each script turn writes 300 to 600 more
  tokens, which is 4 to 8 s more at 50 to 80 tokens per second. Comment to
  reply 30 to 40 s.
- Tokens per live hour: about 14 turns, 5.6M, as in A.
- Big gains only on fan-out: 543f's 17 reads to cut slices become one script;
  all threads across all paired documents become one script.

**Build.** L. An interpreter as a fourth module, or a small language written
by hand, plus a documented API and error reporting inside scripts.

**Rules broken.** Other: three modules (a fourth), or "nothing runs an external
program" if the script is Python; "a file gdoc half understands never reaches a
document" now has a program in front of it. R1 holds: one script, one object.

**Risks.** Script errors cost retry turns. A script that posts in a loop
multiplies one mistake, so the Burst hold must work inside scripts. Nail can
audit a posted reply more easily than a program that posted it.

**Evidence.**
- For: Anthropic's 150,000 to 2,000 tokens (98.7%) on one workflow with large
  intermediate data; programmatic tool calling 37% fewer tokens, 20 to 40% in
  production; Port of Context 4 steps against 19 (cli_mcp_code_debate §1;
  tool_makers §6). Today the model already does this badly: 35 to 83% of gdoc
  calls are piped into ad hoc Python [TX].
- Against: the docs say programmatic calling does not help "strictly
  sequential workflows where each call depends on Claude reasoning over the
  previous result", which is a review window. Cloudflare's gain scales with API
  size, and gdoc's API is small. Ronacher found his code-mode Playwright "not
  promising enough yet". A gets most of the gain with fixed, reviewed commands.

### E. gdoc runs its own small agent

**How it works.**
1. On an actionable window, gdoc builds a small prompt: the review rules (2.4k
   tokens), the packet (2k to 4k), and nothing from the hub session.
2. It calls the Messages API with four tools: `hub_search`, `hub_read`,
   `reply` and `propose`, and loops for at most four turns with a Sonnet-class
   model.
3. `ai!` (hub edits) and any held write go to the person, not to this agent.
4. It runs inside `gdoc mcp`'s watch, or in a sidecar process, and writes its
   result to the presence record for the menu bar app.
5. The agent loop and the tool use are written by hand in Go, as the MCP layer
   was.

**Flow.** Person → Docs: writes `ai?`. gdoc watch → Drive: poll. gdoc → Docs:
packet. gdoc → Anthropic API: prompt and tools. API → gdoc: tool calls. gdoc →
hub: search and read. gdoc → Docs: reply, read-back. gdoc → presence: line.
Menu bar app → person: done, or a hold to approve.

**Effect, est.**
- Tokens per comment: 3 turns × 15k to 25k = 45k to 75k, against 4.5 × 400k
  = 1.8M today. 25x to 40x fewer.
- Speed: a small-context turn of about 300 output tokens at about 100 tokens
  per second, plus about 1 s to the first token, is 2 to 4 s. 3 turns = 6 to
  12 s, plus 6 to 8 s of gdoc, plus 2 to 3 s to detect: 15 to 25 s.
- A visible acknowledgment can land at about 3 s if wanted (see G).
- Tokens per live hour: no big session at all, about 3 × 60k = 0.2M on the API.

**Build.** XL. The loop, tool use, hub search in Go, an API key in the
Keychain, cost caps, hold approval in the app, and an evaluation of answer
quality before anything posts for real.

**Rules broken.** R2: gdoc now hosts the judging. The model still judges, but
gdoc decides what it sees. R5 if it answers with no session open. Other: a
sixth host (`api.anthropic.com`) carrying a credential, beside the "fifth
grant" rule that update requests carry none; a second secret beside the Google
token.

**Risks.**
- Injection with less oversight: a smaller model, a stranger's comment, and
  nobody reading the terminal. Every write must pass the hold rules, and a held
  write must wait for a person.
- API billing sits outside the Max subscription.
- Answers from a narrow context may be shallower than the hub session's.
- One more model integration to keep current.

**Evidence.**
- For: Sentry runs its own agent (Seer) and offers MCP and a CLI beside it.
  Linear wakes agents by webhook and asks for a sign within 10 s.
  `claude-code-action` answers `@claude` in a thread on a runner
  (tool_makers §5).
- Against: the Agent SDK is TypeScript or Python, which principle 1 rules out
  (claude_and_mcp_surfaces §5). MCP sampling, the protocol's way to borrow the
  client's model, is deprecated in 2026-07-28 (§4). No vendor publishes
  event-to-answer latency, so the speed figure is mine alone.

### F. No open session: a watcher wakes a fresh headless agent per comment

**How it works.**
1. A launchd agent, or the menu bar app, runs `gdoc watch --hub <dir>`. It
   finds the paired documents from the notes' `gdoc:` front matter and polls
   them.
2. On an actionable window it starts `claude -p` in the hub with a fixed
   prompt ("handle threads X in document Y"), a Sonnet-class model, a narrow
   tool allowlist and a turn cap.
3. The headless run uses the job commands from A, so it needs 3 to 4 turns.
4. It answers `ai?` only. `ai!` and holds are listed for Nail in the menu bar
   app.
5. A lease in the presence record stops it when a live session is already
   watching that document.

**Flow.** Person → Docs: writes `ai?`. gdoc watch → Drive: poll. gdoc watch →
launcher: event. Launcher → `claude -p`: prompt plus packet. `claude -p` → hub:
Grep, Read. `claude -p` → gdoc respond: plan. gdoc → Docs: writes. gdoc →
presence → menu bar app: done, or waiting for Nail.

**Effect, est.**
- Tokens per comment: 3 to 4 turns × 40k to 80k (fresh system prompt, project
  CLAUDE.md, skill, packet) = 150k to 300k. Per live hour about 0.6M. Quiet
  hours cost nothing: Go polls, no model runs.
- Speed: `claude -p` start-up 3 to 8 s (unmeasured), 3 turns × 5 to 6 s with a
  faster model and small context, 7 s of gdoc, 2 s to detect: 30 to 40 s.
- It is the only idea that answers while no session is open, as long as the
  laptop is awake.

**Build.** M for the watch and the launcher. L if the menu bar app hosts it
and shows holds.

**Rules broken.** R5 directly: "There is no watcher, and the loop starts when
Nail starts it" (SPEC.md). Other: principle 2's "There is no hidden directory
and no queue", because held items wait somewhere; the app's six argv lists if
it launches `claude`; "nothing runs an external program" if gdoc launches it
rather than launchd or the app.

**Risks.**
- Replies under the firm's name to a stranger's text, with nobody watching.
  This is the largest risk in this review.
- Cost runs away on a busy or hostile document. It needs caps per document and
  per hour.
- Two answers if a live session and the watcher both act. The lease must hold.
- A sleeping laptop answers nothing, and nobody notices.
- Which billing pool `claude -p` draws from (see open questions).

**Evidence.**
- For: `claude-code-action`, Linear agents and Sentry Seer all wake an agent
  per event rather than keep a session open (tool_makers §5).
- Against: plugin monitors and development channels do not run in `-p`
  (claude_and_mcp_surfaces §3, §5). No vendor publishes end-to-end latency for
  this shape.

### G. A deterministic fast path in Go

**How it works.** gdoc does what needs no judgement, and leaves the rest to the
model.
1. The wait never wakes on gdoc's own echoes: it skips activity where every new
   item opens with gdoc's mark.
2. Optional: on a new marked comment, gdoc posts a fixed acknowledgment within
   seconds ("Looking into this now.").
3. `reply --if-unchanged <modified>` refuses when the thread changed since the
   draft was read, which replaces the separate re-read call.
4. `propose` runs its three read-backs in parallel.
5. Refusals the model would hit (a multi-tab document, a null range) are listed
   in the packet before it drafts.

**Flow.** Person → Docs: writes `ai?`. gdoc wait → Drive: poll. gdoc → Docs:
acknowledgment (optional). gdoc wait → model: window. Model → gdoc reply: body
with precondition. gdoc → Docs: write, read-back.

**Effect, est.**
- First visible sign: 2 s poll + 1.3 s reply = about 3 to 5 s, with the
  acknowledgment.
- Answer: 55 s minus about 2 s (no re-read call), 3 s (parallel read-backs)
  and one turn (8 s) = about 45 s.
- Tokens per live hour: 3 × 4 + 0 echo + 5 quiet = 17 turns = 6.8M, down 21%.

**Build.** S to M.

**Rules broken.** R2 for the acknowledgment: gdoc writes into a thread on its
own decision. It also reverses review.md's "write the acknowledgment only when
the work will take noticeably long", and a session that dies after an
automatic acknowledgment leaves the dangling "on it" that the two-message rule
exists to prevent. The echo filter strains R2 mildly, as in A.

**Risks.** An automatic acknowledgment under a comment the model would have
held (a flagged thread). It carries no content, so the harm is small, but it
tells a stranger the trigger worked.

**Evidence.** Linear asks agents to show a sign within 10 s (tool_makers §5).
Anthropic: conditions in code save time to first token (cli_mcp_code_debate
§1). The baseline's 43 first-poll returns.

### H. A quiet watch through Claude Code monitors and hooks

**How it works.**
1. `gdoc comments <url> --follow` keeps its cursor in the process and prints
   one line per actionable window, nothing while quiet, and skips own echoes.
2. The skill runs it under the Monitor tool (generally available; up to 30
   minutes, then re-armed), or a plugin monitor with `when:
   "on-skill-invoke:gdoc-review"` runs it for the whole session.
3. A SessionStart hook adds one line (version, update, auth) as
   `additionalContext`, so no `help --json` turn.
4. A PreToolUse hook runs the chat hold rules on Bash calls to `gdoc reply`,
   `propose` and `annotate`, which brings Desktop's safety to Claude Code.
5. A PostToolUse hook records the ids gdoc wrote, for the echo filter.

**Flow.** Person → Docs: writes `ai?`. gdoc follow → Drive: poll. gdoc follow
→ Monitor → model: one line. Model → gdoc reply (Bash). PreToolUse hook →
gdoc hold check → allow, or ask the person. gdoc → Docs: write.

**Effect, est.**
- Per live hour: 3 × 4.5 + 0 echo + 2 re-arms (Monitor tool) = 15.5 turns =
  6.2M, down 28%. With a plugin monitor, 0 re-arms: 13.5 turns, 5.4M.
- Speed: unchanged on its own, apart from the help turn. With A it is A.
- No backgrounded-wait detour: in fd18 each backgrounded wait cost a
  notification and then a file read [TX].

**Build.** S to M.

**Rules broken.** R1: `--follow` prints a stream of lines. R5, arguably, as in
C: a session-scoped watch with no nine-minute bound.

**Risks.** Plugin monitors are experimental and their manifest "may still
change". Whether a Monitor line wakes an idle session at once is documented
("Claude interjects when an event arrives") but not measured here. A
PreToolUse hook adds latency to every Bash call. Claude Code only.

**Evidence.** claude_and_mcp_surfaces §2 and §3 for the mechanics. Baseline
frictions 1, 2, 6 and 7 for the costs it removes. Anthropic: a polling loop in
code beats alternating tool calls and sleeps (cli_mcp_code_debate §1).

### I. A small responder per window, inside the session

**How it works.**
1. The big session only wakes and dispatches. Each actionable window goes to
   a plugin subagent, `gdoc-responder`.
2. The subagent runs a Sonnet-class model in a fresh context, with the gdoc
   job commands, Grep and Read on the hub, and no Edit.
3. It returns at most ten lines: what it posted, in full, and what verified.
4. An `ai!` that needs a hub edit comes back to the big session.
5. Several windows can run in parallel.

**Flow.** gdoc watch → big session: window. Big session → subagent: packet.
Subagent → hub: Grep, Read. Subagent → gdoc respond. gdoc → Docs: writes.
Subagent → big session: the posted text and the checks. Big session → Nail:
report.

**Effect, est.**
- Big-session turns per window: 2 (dispatch, report) instead of 3 to 4.5.
- The big context grows by 1k to 2k per window instead of 10k to 30k, so C
  stays near where it started. This is the larger effect over a long session.
- Subagent: 4 turns × about 30k = 120k per window.
- Speed: spawn 2 to 5 s (unmeasured), 2 small turns at 4 to 5 s. With A:
  3 + 8 + 3 + 10 + 4 + 2 = 30 s. Without A it is not faster than today.
- Per live hour with A and H: 3 × 2 + 2 = 8 big turns. At a kept C of 100k,
  0.8M plus 3 × 120k = 1.2M.

**Build.** S to M. An agent file and skill text. No binary change.

**Rules broken.** None in the binary.

**Risks.**
- Answer quality: a smaller model, without what Nail said earlier in the
  session, posting under the firm's name.
- Each subagent pays the skill load again.
- Nail sees only its summary, so the "print every reply in full" rule must be
  part of its output.
- Claude Code only.

**Evidence.** A plugin subagent gives "skills delegate to a gdoc agent" at no
extra runtime cost (claude_and_mcp_surfaces §5). Skills lift smaller models:
Supabase measured Sonnet from 78% to 100% with a skill (tool_makers §4).
Anthropic's code-execution post: keep intermediate data out of the main
context (cli_mcp_code_debate §1).

### J. The document as local files

**How it works.**
1. `gdoc mirror <url>` writes the document text with its anchors, the threads
   and the pending suggestions as files under the config directory, keyed by
   revision.
2. The model reads them with its own Read and Grep, in parallel and at no gdoc
   cost.
3. A wait refreshes the mirror before it returns.
4. Writes still go through gdoc and carry a revision or modified-time
   precondition, so a stale mirror cannot produce a wrong write.
5. The mirror is a cache, never a record of what was handled.

**Flow.** gdoc mirror → Docs and Drive: one read. gdoc → disk: files. Model →
disk: Grep, Read. Model → gdoc reply or propose: with precondition. gdoc →
Docs: write.

**Effect, est.**
- 543f's 17 reads of a 56k-character document, 1.3 s and one turn each,
  become one mirror call plus Grep.
- Comment to reply: about 1 to 2 s of gdoc and up to one turn less, so about
  50 s. Tokens per live hour about 20 turns = 8.0M.

**Build.** M.

**Rules broken.** Other: principle 2 says nothing lives hidden beside the hub.
Under the config directory it is not beside the hub, but it is a hidden local
copy of a document, against the stance that the document is the only ledger.

**Risks.** Staleness. Confidential text on disk, though it already sits in
transcripts. Cleanup.

**Evidence.** Playwright CLI writes each page snapshot to a file and searches
it with `find` (tool_makers §1). File-path outputs closed the Playwright MCP vs
CLI gap in 2026 (cli_mcp_code_debate §4). Anthropic: write intermediate results
to files.

### K. Live review in its own small session

**How it works.**
1. Live review runs in a session of its own, not inside the hub session where
   Nail publishes and aligns.
2. The skill says so on the word "live", and gives the one-line command that
   opens it, or the desktop app starts it.
3. Auto-compaction runs at a low threshold in that session.
4. The hub session keeps its own work and context.
5. With A and I, the live session's context stays near its start for hours.

**Flow.** Nail → hub session: "live review <url>". Hub session → Nail: the
command for a new session. New session → gdoc watch. Window → new session →
gdoc.

**Effect, est.**
- Alone: C falls from the mixed-session 445k to 725k to the dedicated-session
  257k to 317k that was measured. At 21.5 turns × 270k = 5.8M per live hour,
  down 33% from 8.6M and 63% from the C = 725k case (15.6M).
- With A, H and I: 8 big turns × 100k + 0.36M = 1.2M, down 86% (est.).
- Speed: a smaller uncached prefix may save 0.5 to 1.5 s per turn
  (unmeasured).

**Build.** S. Skill text only.

**Rules broken.** None.

**Risks.** Two sessions to keep in mind. The live session lacks what Nail said
in the hub session. A dedicated session still grows: 543f was dedicated and
still ran at 317k, mostly from repeated reads, so K needs A or J to stay small.

**Evidence.** The baseline itself: f942 (mixed) at 445k median and 725k on
no-op rounds, against 543f and fd18 (dedicated) at 317k and 258k [TX].

### Considered and dropped

- **Drive push notifications** (`changes.watch`). They need a public HTTPS
  endpoint, so a relay outside the laptop. Detection is already 2 to 4 s of a
  55 s path. Not the bottleneck.
- **MCP sampling**, so gdoc borrows the client's model. Deprecated in MCP
  2026-07-28 (claude_and_mcp_surfaces §4).
- **An Apps Script trigger on comments.** As far as I know Docs offers no
  trigger on a new comment, and it would put code inside documents gdoc does
  not own.
- **MCP Apps cards.** Useful for holds in a chat, not a speed or token lever.
- **Tool search tuning.** gdoc's MCP is eight tools and about 2k tokens. The
  85% saving applies to servers ten times that size.

---

## 3. Side by side

Estimates throughout, from the arithmetic above. "Tokens" is per live hour in
the three-comment scenario, at C = 400k unless the idea changes C.

| Idea | Comment to reply | Publish to link | Big turns per window | Turns per quiet hour | Tokens | Build | Breaks |
|---|---|---|---|---|---|---|---|
| Today | 55 s | 60 to 75 s | 4.5 | 6.7 | 8.6M | | |
| A Job commands | 25 to 35 s | 25 to 30 s | 3 | 6.7 | 5.6M | M | R2 mild |
| B Daemon | 52 to 54 s | 59 to 74 s | 4.5 | 6.7 | 8.6M | L | R4, R5, Other |
| C One MCP route | 23 to 33 s with A | 25 to 30 s with A | 3 to 4 | 0 | 3.6 to 4.8M | M to L | R1, R5 arguably |
| D Code execution | 30 to 40 s | 30 to 35 s | 3 | 6.7 | 5.6M | L | Other |
| E Own agent | 15 to 25 s | as A | 0 | 0 | about 0.2M + API | XL | R2, R5, Other |
| F Headless per comment | 30 to 40 s with A | as A | 0 | 0 | about 0.6M | M | R5, Other |
| G Fast path | 45 s; sign at 3 to 5 s | 57 to 72 s | 4 | 6.7 | 6.8M | S to M | R2 |
| H Quiet watch | 50 to 55 s | 52 to 67 s | 4.5 | 0 to 2 | 5.4 to 6.2M | S to M | R1, R5 arguably |
| I Small responder | about 30 s with A | n/a | 2 | as today | 6.0M alone, 1.2M with A, H, K | S to M | none |
| J Local files | about 50 s | n/a | 4 | 6.7 | 8.0M | M | Other |
| K Own live session | about 50 s | n/a | 4.5 | 6.7 | 5.8M alone | S | none |

How each scores on the stated weights. Scores are my judgement, 0 to 5;
weights follow the order given (speed 30, tokens 20, turns 15, live reaction
10, state 10, reliability 10, one route 5).

| Idea | Speed | Tokens | Turns | Live | State | Reliability | One route | Weighted, of 500 |
|---|---|---|---|---|---|---|---|---|
| E Own agent | 5 | 5 | 5 | 3 | 3 | 3 | 4 | 435 |
| C One MCP route (with A) | 4 | 4 | 4 | 4 | 4 | 4 | 4 | 400 |
| F Headless | 3 | 4 | 4 | 5 | 2 | 3 | 3 | 355 |
| A Job commands | 3 | 3 | 4 | 1 | 2 | 4 | 3 | 295 |
| G Fast path | 3 | 3 | 3 | 2 | 2 | 4 | 4 | 290 |
| I Small responder | 2 | 4 | 3 | 0 | 1 | 3 | 1 | 230 |
| H Quiet watch | 1 | 3 | 3 | 4 | 1 | 3 | 1 | 220 |
| J Local files | 2 | 2 | 2 | 1 | 4 | 3 | 2 | 220 |
| K Own live session | 1 | 4 | 0 | 0 | 0 | 3 | 1 | 175 |
| D Code execution | 2 | 3 | 3 | 0 | 2 | 2 | 2 | 215 |
| B Daemon | 1 | 1 | 0 | 2 | 4 | 2 | 3 | 145 |

The weighted score ignores build cost and the rules. The ranking in section 5
does not.

---

## 4. Red team

### The favourite

My favourite target is **"one engine, a quiet watch, a small worker"**:

- **Engine:** job-sized commands with work packets (A), plus the fact-only
  parts of the fast path (G: echo filter, precondition, parallel read-backs).
  One command table, served as the CLI and as `gdoc mcp` (C).
- **Watch:** wakes the model only for real work, and costs no turns when quiet
  (H now, C's watch later, a channel where allowed).
- **Worker:** routine windows run in a small, fresh context (K and I now; F
  headless later if Nail lifts R5). The big session dispatches and reports, or
  is not involved at all.
- **Menu bar app:** reads the presence record for the watch and the holds, as
  it does now.

### The strongest case against it

1. **It optimises the wrong number.** The person who writes `ai?` in a
   document is not watching a stopwatch. The 18 August leak (review.md) shows
   the expensive failure is a wrong or unsafe answer, not a slow one. Moving
   answers to a smaller model in a smaller context buys seconds and tokens
   with the one thing that has burned the team before.
2. **The responder does not pay for itself inside a session.** With A and H
   in place, I takes big turns per window from 3 to 2. The wake turn and the
   report turn still re-read C. At C = 400k the hour goes from 11 big turns
   (3 × 3 + 2 re-arms = 4.4M) to 8 big turns plus small ones (3.2M + 0.36M =
   3.56M), 19% (est.). And the speed barely moves, because spawning costs most
   of what the smaller model saves.
3. **The only real token win needs R5 broken.** Everything inside Claude Code
   still wakes a big session per event. Only F and E take the big session out,
   and both answer strangers with nobody watching. If Nail will not accept
   that, the "small worker" is mostly decoration.
4. **It leans on surfaces that are not stable.** Plugin monitors are
   experimental. Channels are a research preview behind a dialog. MCP
   auto-backgrounding is a Claude Code behaviour, not a protocol promise. A
   solo maintainer pays for each of these when it changes.
5. **The workflow moves into Go.** Today a skill edit is live the moment it is
   saved. Job commands put the order of a review in the binary, behind a
   release, and a fixed order is the opposite of letting the model judge.
6. **"One route" is a slogan.** Desktop has no push and will not get one soon.
   The menu bar app runs no model. The honest shared layer is the command table
   and the presence record, which already exist.

### What survives, and the decision

- Point 2 is right, with one correction. In a long session the responder's
  value is that C stops growing: each window adds 1k to 2k to the big context
  instead of 10k to 30k (est.). Over ten windows that keeps C near 100k instead
  of letting it climb by 100k to 300k. But that value is cheaper to get from K
  (a dedicated live session) and A (packets instead of full reads). **So I
  demote I.** It becomes an experiment in Stage 3, kept only if a blind check
  finds its answers as good.
- Point 1 is right about priorities. **The answer-quality bar is the gate on
  every stage that changes who writes the answer** (I, F, E). Speed work that
  does not change the writer (A, G's fact parts, H, K) carries no quality risk
  and goes first.
- Point 3 is right, and it is Nail's call. I keep F as the stage that removes
  the big session, gated on his decision, and I put E after F's data, because
  F reuses Claude Code's tools and skills and costs M where E costs XL.
- Point 4: Stage 1 uses the Monitor tool, which is generally available, not
  plugin monitors or channels. Channels stay an experiment.
- Point 5 is a real cost. The mitigation: the job commands do mechanical steps
  only (fetch, pre-join, precondition, verify). Judgement stays in the plan
  file the model writes. Every decision that is the model's today stays the
  model's.
- Point 6: agreed. "One route" in this review means one command table, one
  rules text and one event record, not one transport.

The decision: **keep the target, reorder it.** Fewer turns and no empty turns
first (A, G's fact parts, H, K). The MCP route second (C), for safety parity
and colleagues more than speed. Changes to who writes the answer (I, F, E) only
behind a measured quality bar and Nail's decision.

---

## 5. Ranking and the path from today

### Ranking by value for its cost and risk

1. **A. Job-sized commands with work packets.** The biggest speed lever that
   does not change who writes. Helps every surface, Desktop most of all.
2. **K. Live review in its own session.** Near-zero cost, a third or more off
   live tokens at once.
3. **H. A quiet watch.** Removes the empty turns. Small build.
4. **G, fact parts only.** Echo filter, precondition, parallel read-backs.
   Small, and on the critical path.
5. **C. `gdoc mcp` as the route for Claude Code too.** Holds and the ledger in
   Code, no Bash checks, a push-capable watch, one rules text.
6. **F. Headless worker per comment.** The only way to answer with no session
   open. Behind Nail's decision on R5 and a quality bar.
7. **I. Small responder subagent.** An experiment, kept on evidence.
8. **G, the automatic acknowledgment.** Nail's call; it reverses a written
   rule.
9. **E. gdoc's own agent.** Fastest on paper. Build it only if F's data shows
   `claude -p` start-up or its context is the bottleneck.
10. **J. Local files.** Worth it for long documents read many times; A's
    packets cover the common case.
11. **D. Code execution.** Revisit for hub-wide fan-out ("all open `ai?`
    across every paired document"). A covers a single window better.
12. **B. Daemon.** It fixes the 25% that is gdoc, and only a slice of that.
    Build it only as a host for F or E if one is ever needed.

### The staged path

**Stage 1: stop paying for nothing (this week).**
- Binary: `comments --follow` (one line per actionable window, cursor kept in
  process, own echoes skipped, comment `created` time on each line);
  `help --brief` (version, update and auth in about 200 bytes).
- Skill: live mode runs `--follow` under the Monitor tool and re-arms at the
  30-minute notice; it tells Nail to run live review in its own session; the
  setup block uses `help --brief`.
- Expected, est.: per live hour 21.5 → 15.5 turns (no echo wake-ups, 2 re-arms
  instead of about 5), and C from 445k to 725k down to about 270k. 8.6M to 15.6M
  becomes about 4.2M. Comment to reply unchanged at about 55 s; this stage is
  for tokens.
- Prove it with:
  - wake-ups per live hour: count follow lines and Monitor re-arms in the
    transcript (target: 0 echo wake-ups; at most 2 re-arms per quiet hour);
  - cache-read tokens per live hour from transcript `usage` (target: below
    half of f942's rate);
  - comment to reply, measured from Drive itself: reply `created` minus
    comment `created`, for every gdoc reply in the document. This is the
    end-to-end number the baseline could not measure, and it needs no
    transcript.

**Stage 2: fewer turns per job (weeks 2 and 3).**
- `review --work` and window packets; `respond --from plan.json` with the
  precondition and parallel read-backs; `publish` that folds the build and the
  marks; concise default output.
- Expected, est.: comment to reply 55 s → 25 to 35 s; publish 60 to 75 s → 25
  to 30 s; one-shot review 11 gdoc calls → about 4.
- Prove it with: turns before the last write per window (target 2, at most 3);
  comment to reply median and p90 from Drive timestamps (target median under
  35 s); `propose` median in presence (target under 4.5 s); gdoc calls per
  one-shot review; refusals per 100 calls (`ok:false` at 0 ms, today 7 of 34
  for `propose`); share of gdoc calls piped into Python (today 35 to 83%,
  target under 10%).

**Stage 3: small context for routine work (week 4).**
- A `gdoc-responder` subagent for `ai?` windows, in the dedicated live
  session. Kept only if it passes the quality check.
- Prove it with: big-session context growth per window (target under 5k);
  a blind check where Nail rates 20 answers from the big session and 20 from
  the responder without knowing which is which (target: no worse); cache-read
  tokens per live hour.

**Stage 4: one engine for both chat surfaces (weeks 5 to 7).**
- The plugin starts `gdoc mcp` in Claude Code. The job tools land in MCP. The
  hold rules and ledger apply in Code. A `watch` tool replaces `--follow` if it
  measures as well. A channel mode behind the development flag, as an
  experiment.
- Prove it with: an A/B on one document, the same five windows through Bash
  and through MCP (model time per turn, turns per window); holds that fired in
  Code; Desktop call gaps (issue #76854); permission prompts per task for a
  colleague with no allowlist (target 0).

**Stage 5: no session needed (only after Nail decides on R5).**
- `gdoc watch --hub` under launchd or the menu bar app; `claude -p` per
  window, for `ai?` only, on documents Nail opts in; a lease against a live
  session; caps per document and per hour; `ai!` and holds listed in the app.
- Prove it with: comment to reply with no session open (target p50 under
  40 s, p90 under 90 s); cost per comment; wrong or held answers per 100
  comments; double answers (target 0).

**Stage 6: gdoc's own agent (only if Stage 5's data asks for it).**
- Build E if `claude -p` start-up or its fixed context dominates Stage 5's
  time or cost.

---

## 6. Open questions only Nail can answer

1. **Unattended answers.** Is an answer posted while no session is open
   acceptable at all? If yes: `ai?` only? Internal documents only? Which
   documents opt in?
2. **The automatic acknowledgment.** Is a sign in the thread within seconds
   worth reversing "acknowledge only when the work will take long", and the
   risk of a dangling "on it"?
3. **Who may write the answer.** Is a Sonnet-class model acceptable for
   replies under the firm's name, or must it be the session's model?
4. **Billing.** Which pool may `claude -p`, the Agent SDK or an API key draw
   from? The June 2026 split between Max and SDK use matters here.
5. **Channels at Altery.** Will the claude.ai org enable `channelsEnabled`, or
   allowlist a gdoc plugin? Is the development-flag dialog at each launch
   acceptable meanwhile?
6. **What "fast" means to you.** Is 25 to 35 s for an answer the target, or is
   a first visible sign within 10 s what matters?
7. **Live review in Claude Desktop.** Does it matter, or is Desktop one-shot
   only? It decides whether a bounded watch belongs in `gdoc mcp`.
8. **The menu bar app as a control surface.** May it start and stop a watcher
   and release holds? That breaks "six argv lists".
9. **Workflow in Go.** Are you comfortable with the order of a review living
   in job commands behind a release, while judgement stays in the skill?
10. **The echo filter as a fact.** Do you accept a wait that does not return on
    a window made only of gdoc's own replies?
11. **Colleagues.** Is the first target your machine alone? A colleague with
    no allowlist gets a prompt per Bash call, which makes the MCP route
    (Stage 4) matter much more.
12. **Two sessions.** Will you actually run live review in a second session,
    or should the skill start one for you?

---

## 7. What I could not verify

- Output tokens per second for the session's model. T = 8 s is derived from
  measured model time, but the split into thinking, writing and time to first
  token is not.
- Whether a Monitor line wakes an idle Claude Code session at once, and how
  late it arrives.
- What a backgrounded MCP tool call's notification carries: the result, or a
  pointer that costs another call.
- `claude -p` start-up time and its fixed context on Nail's machine.
- Subagent spawn time.
- Drive's delay between a comment being posted and the listing showing it.
  Stage 1's Drive-timestamp metric includes it, which is the point.
- How much of `propose`'s 6.8 s the docx export takes, which decides how much
  parallel read-backs save.
