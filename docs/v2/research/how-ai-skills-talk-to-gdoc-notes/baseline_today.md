# Baseline: how the AI skills talk to gdoc today, and what it costs

Scope: the five skills under `skills/`, the `gdoc` CLI they call through the Claude Code Bash tool, and `gdoc mcp` for Claude Desktop. All numbers below were measured on Nail's machine on 2026-10-27, read only. Nothing was written to Drive.

How each number was measured (referred to below by its tag):

- **[FILES]** `wc -c` on the repository files at commit f2b6f5a. Tokens are estimated as bytes / 4.
- **[HELP]** `gdoc help <command> --json` run from `~/.local/bin/gdoc` (installed binary is v2.20.0, main is v2.21.0), stdout saved and counted with `wc -c`, wall time with `/usr/bin/time -p`.
- **[MCP]** `gdoc mcp` fed four JSON-RPC lines on stdin (initialize, initialized, tools/list, tools/call guide). Each reply line counted. The guide call is local and reaches no network.
- **[PRES]** `~/.config/gdoc-agent/presence/2026-10.jsonl`, 699 lines from 2026-10-17T01:14Z to 2026-10-26T08:26Z: 339 runs, each a start line and an end line carrying `ms` (monotonic elapsed time, `go/internal/presence/doc.go:170`). Start and end joined on `run`+`call`. `wait: true` is on the start line only. `via` is `code` (Claude Code, 493 lines) or `terminal` (206 lines). No `chat` (Desktop) line exists in the month. `help` and `build` do not write presence lines. Python scripts in the session scratchpad computed medians and p90 by linear interpolation.
- **[TX]** Claude Code transcripts matched to presence by the `session` field (Claude Code's own session id, `go/internal/presence/doc.go:92-101`). Three of the six presence sessions have transcripts:
  - `f942157f` in `~/.claude/projects/-Users-nailkhusnullin-src-altery-Altery-Platform-hub/`: publish, live review, align and Jira, 2026-10-17 to 10-19, 353 model API calls.
  - `543fde4f` and `fd1845e6` in `...-202609-embedded-retail-card/`: live reviews of the concept paper (2026-10-20 to 10-22) and the legal actions doc (2026-10-26), with 185 and 125 API calls.
  - From each transcript I took the Bash `tool_use` timestamps, the `tool_result` timestamps, the result text length, `permissionDecision`, and each assistant message's `usage` (deduplicated by message id).
  - A scan of all 12 transcripts under `~/.claude/projects/` that call gdoc found 360 Bash calls that run a gdoc command.

---

## 1. Which gdoc commands each skill runs, in order, and how many shell calls and model turns a typical task takes

### Takeaway
Every skill opens with `gdoc help --json`. Most also call `gdoc help <command> --json` before each command. After that they run 3 to 12 single-shot commands, one Bash call each. A one-shot review of a document with two `ai?` and one `ai!` comment and one proposal is about 11 gdoc calls and 15 to 20 model turns. In the transcripts the model wraps 35% to 83% of gdoc calls in pipes or ad hoc Python to cut the JSON down.

### Cited Findings
- **Setup in every skill.** `GDOC=gdoc; ROOT="$PWD"; $GDOC help --json`. `help` is "the first call of every session", because it carries `version` and `update` (skills/gdoc-review/SKILL.md:27-54). The same block is in skills/gdoc-publish/SKILL.md:24-60, skills/gdoc-export/SKILL.md:38-75, skills/gdoc-restyle/SKILL.md:25-62 and skills/gdoc-align/SKILL.md:37-75.
- **"Learn the command before calling it".** Publish, export, restyle and align each tell the session to run `$GDOC help <command> --json` before a command (skills/gdoc-publish/SKILL.md:69-84; skills/gdoc-export/SKILL.md:81-95; skills/gdoc-restyle/SKILL.md:70-86; skills/gdoc-align/SKILL.md:80-95). Review does not.
- **Review, one shot** (skills/gdoc-review/SKILL.md):
  - Step 1 runs `comments <url>` then `read <url>` (lines 95-100).
  - Step 2 runs `suggestions <url>` and reads `review.md` (lines 131-147).
  - Step 5: per `ai?`, the session writes a body file with a heredoc, then runs `reply <url> <id> --body-file` (lines 194-199).
  - review.md says to read the thread again just before every post (skills/gdoc-review/review.md:69-87). In practice that adds one `comments` call per reply: presence shows `comments` then `reply` as a pair in every live window (see question 4).
  - Step 7: per proposal, the session writes a proposals JSON file, then runs `propose <url> --from <file> --md <note>`. It reads `propose.md` first (SKILL.md:269-294; skills/gdoc-review/propose.md:20-28).
  - A block replace first runs `suggestions` and `comments` again (SKILL.md:275-282).
  - Count for 2 `ai?`, 1 `ai!` and 1 proposal: help 1, comments 1, read 1, suggestions 1, then 3 × (comments + reply), then propose 1. That is 11 gdoc calls, plus 2 Reads (review.md, propose.md), file writes, hub search and edit calls. (Inference from the skill text, and consistent with [PRES] below.)
- **Live review.**
  - Everything above runs once.
  - Then the loop `comments <url> --since "$CURSOR" --wait 9m` runs with the Bash `timeout` set to 600000 (skills/gdoc-review/live.md:14-49).
  - Each window that holds threads runs Steps 2 to 8 again (live.md:36-38).
  - Before acting on a marked comment that may be old, the session runs `suggestions` again (live.md:97-102).
- **Publish** (skills/gdoc-publish/SKILL.md):
  - `help --json`, then `build` to a scratch path to read `body.marks` (lines 233-257).
  - Then `help publish --json` and one `publish` (lines 259-271).
  - `help marks --json` when marks are present (line 218).
  - Typical total: 4 to 5 gdoc calls. [TX] session f942 used 13 `build` calls and 2 `publish` calls across two publishes, and the `build` calls were retries while fixing the note.
- **Export** (skills/gdoc-export/SKILL.md):
  - `help --json`, then ask for a file name (Step 1, lines 116-131).
  - `help export --json`, then `export <url> --out <path>` (lines 132-143).
  - `comments <url>` when threads exist (lines 182-197).
  - Marker resolution with the person, reading markers.md (Step 5).
  - Typical total: 3 to 4 gdoc calls.
- **Restyle** (skills/gdoc-restyle/SKILL.md):
  - `help --json`, `help restyle --json`, then the survey `restyle <url> --dry-run`, saved to a file (Step 1, lines 112-155).
  - Ask about the cover and write the fields file (Step 2).
  - Ask once more, then `restyle <url> --from <survey> [--fields <file>]` (Step 3, lines 200-211).
  - Flags measured from [HELP]: `--dry-run`, `--from`, `--fields`.
  - Typical total: 4 gdoc calls and 2 questions to the person.
- **Align** (skills/gdoc-align/SKILL.md):
  - Step 0 may export again per stale copy (lines 117-146).
  - `help export --json` and `export <url> --out <note>.md` (lines 148-160).
  - `comments <url>` when threads exist (line 206).
  - `read <url>` (line 218), then `suggestions` and `comments` before a block replace (line 242).
  - One question for the whole list, then `propose --from ... --md` (lines 248-270).
  - Resolve-only mode runs `suggestions <url> --md <note>.md` (line 325).
  - Typical total: 6 to 8 gdoc calls.
- **[TX] gdoc calls per transcript.**
  - f942: 129 Bash calls running gdoc. Subcommands: comments 74, propose 18, reply 17, read 15, build 13, help 6, suggestions 4, publish 2, export 1.
  - 543f: 66 Bash calls. comments 28, read 17, propose 17, reply 10, withdraw 5, annotate 3, suggestions 3, help 2.
  - fd18: 52 Bash calls. comments 17, read 13, help 7, suggestions 7, propose 6, build 4, update 1, publish 1, withdraw 1, reply 1.
- **[TX] The model chains several gdoc commands in one Bash call** to save turns: 21 of 129 calls (f942), 14 of 66 (543f) and 6 of 52 (fd18).
- **[TX] Calls piped or redirected** into `python3 -c`, `head -c`, regex extractors and similar: 107 of 129 (83%) in f942, 23 of 66 (35%) in 543f, 41 of 52 (79%) in fd18. Examples from f942:
  - `gdoc comments ... 2>/dev/null | python3 -c "import json,sys; d=json.load(sys.stdin); ..."`
  - `gdoc read ... | python3 -c "...re.finditer(r'A merchant refund[^\n]{0,260}',t)..."`
- **[PRES] Calls per Claude Code session:**

  | Session | gdoc runs | Span | Non-wait gdoc time | Wait time |
  |---|---|---|---|---|
  | f942 | 57 | 4,711 s | 98.3 s | 3,933 s |
  | 543f | 87 | 150,657 s (across 2 days) | 364.8 s | 1,206 s |
  | 13ab | 23 | 3,996 s | 20.3 s | 2,791 s |
  | fd18 | 51 | 3,018 s | 106.1 s | 1,382 s |
  | a022 (publish only) | 3 | 278 s | 23.0 s | 0 s |

### Inferences
- A typical one-shot task is 4 to 12 gdoc processes, each a separate Bash tool call and so a separate model turn, unless the model chains them.
- The skills' "learn the command first" rule adds one `help <command>` turn per distinct command.
- The heavy piping shows that the model, not the binary, does the narrowing of large JSON. That costs output tokens to write parsers. It also risks the cursor rule in live.md:69-71: never keep the cursor of a listing read through a narrowing pipe.

### Gaps
- No transcript exists for sessions 13ab738d and a0221dcb, so their model turns cannot be counted.
- No transcript covers a full `restyle` or `export` session, so the call counts for those two are read from the skill text, not observed.

---

## 2. How large the inputs and the JSON replies are, in bytes and tokens

### Takeaway
The skill text a session loads is 6k to 13k tokens per task. The `help --json` envelope is 9.6 KB, about 2.4k tokens. Replies are small, under 1.5k chars at the median, except `read`, which carries the whole document: 55,825 chars of text for the concept paper, about 14k tokens. The real token cost, though, is not these payloads. It is that every model turn re-reads the session's whole cached context: a median of 257k to 445k tokens per call in the three transcripts.

### Cited Findings
- **[FILES] Skill files:**

  | File | Bytes | Tokens |
  |---|---|---|
  | gdoc-review/SKILL.md | 15,017 | ~3.8k |
  | gdoc-review/review.md | 9,438 | ~2.4k |
  | gdoc-review/propose.md | 12,717 | ~3.2k |
  | gdoc-review/live.md | 6,147 | ~1.5k |
  | gdoc-publish/SKILL.md | 19,256 | ~4.8k |
  | gdoc-export/SKILL.md | 13,110 | ~3.3k |
  | gdoc-export/markers.md | 11,089 | ~2.8k |
  | gdoc-restyle/SKILL.md | 14,938 | ~3.7k |
  | gdoc-align/SKILL.md | 18,877 | ~4.7k |

- **[HELP] Help output and timing.**
  - `help --json` is 9,631 B. It holds 19 commands, `settings` and `update`.
  - Per-command help, in bytes: read 740, comments 1,048, suggestions 751, export 783, restyle 1,020, reply 849, propose 1,278, withdraw 769, annotate 1,018, build 2,088, publish 906, marks 680, auth status 259.
  - Wall time was 0.02 to 0.03 s each, with a fresh update stamp.
  - In [TX] f942 a `help --json` result reached 12,709 chars.
- **Per-task skill load (bytes / 4):**

  | Task | What it loads | Bytes | Tokens |
  |---|---|---|---|
  | Review, one shot | SKILL + review.md + help | 34,086 | ~8.5k |
  | Review, with a proposal | adds propose.md | 46,803 | ~11.7k |
  | Review, live | adds live.md | 52,950 | ~13.2k |
  | Publish | SKILL + help + help publish + help build | 31,881 | ~8.0k |
  | Export | SKILL + markers.md + help + help export | 34,613 | ~8.7k |
  | Restyle | SKILL + help + help restyle | 25,589 | ~6.4k |
  | Align | SKILL + help + help export + help propose + propose.md | 43,286 | ~10.8k |

- **Duplicate skill listing.** This session's own skill listing shows each gdoc skill twice, `gdoc-review` and `altery:gdoc-review` and so on, once from the `~/.claude/skills/` symlink and once from the plugin. CLAUDE.md "Skills are linked, not copied" describes both routes. That is 10 descriptions of about 300 chars each in every session's system prompt.
- **[TX] Reply sizes as the model received them** (median / p90 chars; most were already narrowed by pipes, so these are lower bounds on the raw size):
  - `comments --wait`: 973 / 2,373 (f942, n=55); 504 / 1,432 (543f, n=22)
  - `read`: 864 / 1,669 (f942); 1,024 / 2,058 (543f); 1,103 / 2,266 (fd18)
  - `propose`: 170 / 801 (f942); 163 / 583 (543f)
  - `reply`: 20 to 44
  - `publish`: 799 to 990
  - `export`: 3,581
- **[TX] Unpiped raw outputs across all 12 transcripts** (median chars): `comments --wait` 1,268 (n=27, p90 2,818, max 3,225); `export` 5,845 (n=2); `comments` 795 (n=1); `annotate` 311 (n=1).
- **[TX] 543f: the size of the document text alone.** One call printed `len(data.text)` = **55,825** chars for the concept paper. The session then made 17 `read` calls, each fetching the whole document from Google and cutting a slice out with a Python regex.
- **[MCP] Desktop route:** initialize 979 B (instructions 792 B); tools/list 7,805 B (~2k tokens) for 8 tools; guide 15,588 B (text 15,178 B, ~3.8k tokens). Per-tool card size (description + schema): read 751, comments 979, suggestions 596, reply 1,438, annotate 1,474, propose 2,073, login 438, guide 362.
- **[TX] Context re-read per model API call** (input + cache read + cache write; median / p90): f942 445,311 / 767,359; 543f 317,219 / 431,492; fd18 257,703 / 302,472. Output tokens per call: median 384 to 687. Session totals:
  - f942: cache read 166.6M, cache write 3.15M, output 355k.
  - 543f: cache read 54.2M, cache write 1.0M, output 163k.
  - fd18: cache read 30.7M, cache write 0.28M, output 79k.

### Inferences
- Per turn, gdoc's own bytes (a few hundred chars to a few KB) are 1% or less of what the model re-reads. A change that cuts the number of model turns saves far more tokens than one that shrinks JSON.
- `read` is the one reply that is large on a real document, about 14k tokens for a 56k-char paper. Skills re-fetch it to grep a slice rather than keeping it, which costs round trips and Google calls more than tokens.
- The duplicate skill listing is a small fixed cost per session. It is also a source of ambiguity about which copy is invoked: [TX] 543f and fd18 invoked `altery:gdoc-review`, while f942 invoked `gdoc-review`.

### Gaps
- No unpiped full `read` JSON or full `comments` JSON on a large document was found in the transcripts, so the raw envelope size for those (text plus JSON escaping plus thread fields) is not measured. I did not run `read` or `comments` myself, to avoid Google calls and new presence lines.
- Token counts are bytes / 4, not tokenizer counts. Russian text and JSON punctuation tokenize worse than that.

---

## 3. Speed: how long each command takes, and where a session's time goes

### Takeaway
gdoc itself is fast on reads: about 1 to 2 s per call, almost all of it Google round trips. The binary starts in about 20 to 60 ms. Writes are slower because of the read-backs: `propose` about 7 s at the median, `publish` about 10 to 12 s. Across a working session, the model's own time between calls is 4 to 16 times gdoc's non-wait time. In a live session the blocked `--wait` calls are the largest share of wall time, and that time is idle.

### Cited Findings
- **[PRES] `ms` per command, Claude Code (`via: code`), non-wait runs:**

  | Command | n | min | median | p90 | max |
  |---|---|---|---|---|---|
  | read | 40 | 559 | 1,310 | 1,780 | 11,897 |
  | reply | 21 | 1,030 | 1,309 | 1,494 | 1,530 |
  | suggestions | 16 | 590 | 985 | 1,405 | 1,663 |
  | withdraw | 8 | 1,226 | 2,413 | 3,544 | 3,638 |
  | annotate | 9 | 0 | 4,259 | 5,082 | 5,739 |
  | propose | 34 | 0 | 6,797 | 17,388 | 87,682 |
  | publish | 8 | 0 | 10,256 | 11,984 | 13,574 |
  | update | 2 | 4,502 | | | 6,505 |

  - Runs of 0 ms are refusals before any request: 7 propose, 3 annotate and 3 publish ended `ok:false`.
- **[PRES] `comments` without `--wait`, all callers:** n=36, min 1,402, median 2,003, p90 2,988, max 4,327 ms. By caller: code median 2,163 / p90 3,597; terminal median 1,894.
- **[PRES] Terminal (person typing) for comparison:** reply median 1,297 (n=12); read 1,191 (n=7); propose 5,448 (n=11); publish 12,333 (n=3); export 3,508 (n=1); `auth login` 6,919 (n=1).
- **[PRES] Publish steps** (step lines, 1 s resolution, five complete runs): build and upload start in the same second. upload to read_back is 6 to 8 s. read_back to link is 4 to 6 s. Total 9.8 to 13.6 s. Publish runs three read-backs (go/internal/publish, CLAUDE.md "What lives where").
- **Propose makes extra calls.** Every proposal is checked by three read-backs, and the run stops at the first one they cannot confirm (go/internal/propose/doc.go:1-14). This is why `propose` costs about 5x a `reply`.
- **[HELP] Process start:** `gdoc help --json` takes 0.02 to 0.03 s wall. [MCP] a whole 4-message `gdoc mcp` session takes 0.06 s wall.
- **Each command opens its own guard, session and HTTP transport.**
  - `open()` builds a new `guard.NewPolicy()` and calls `openSession(p)` (go/cmd/gdoc/read.go:346-361).
  - `gapi.Open` loads the token from disk and builds `guard.NewClient` (go/internal/gapi/session.go:184-191).
  - `NewClient` makes a new `http.Transport` when no base is given (go/internal/guard/transport.go:113-133).
  - So no TCP or TLS connection is reused across commands. This holds even inside `gdoc mcp`, which runs each tool through `safeDispatch` (go/cmd/gdoc/mcptools.go:502).
- **Token refresh is not per process.** A refresh is saved back to the token file, and a later process reads it (go/internal/gapi/session.go:530-560). Expiry has one minute of skew (go/internal/auth/auth.go:46-48, 101-103). So a new process pays a refresh about once an hour, not on every call.
- **[TX] Where the wall time goes**, from Bash tool durations and the gap from a user or tool_result line to the next assistant message, gaps under 600 s:

  | Session | gdoc `--wait` | Model | Other tools | gdoc non-wait |
  |---|---|---|---|---|
  | f942 | 93.5 min (57.8%) | 54.5 min (33.7%) | 10.5 min (6.5%) | 3.4 min (2.1%) |
  | 543f | 20.4 min (41.8%) | 17.1 min (35.0%) | 7.2 min (14.8%) | 4.1 min (8.4%) |
  | fd18 | 2.1 min (11.3%) | 11.0 min (58.2%) | 3.2 min (17.0%) | 2.5 min (13.5%) |

  In fd18 the waits ran in the background, so their time is mostly not in the wait bucket.
- **[TX] Model time before each gdoc call** (previous tool_result to the gdoc tool_use; median / p90): 7.2 / 14.8 s (f942), 9.1 / 192.4 s (543f), 6.4 / 19.7 s (fd18).
- **[TX] Permission prompts:** 246 of 247 gdoc Bash calls in the three transcripts carry `permissionDecision: accept, source: config`, an allowlist rule, and 1 carries `reject, source: config`. None waited on a human click.

### Inferences
- Model thinking is 4x to 16x gdoc's own non-wait time: 54.5 vs 3.4 min, 17.1 vs 4.1 min, 11.0 vs 2.5 min. Process start (about 20 ms) is negligible.
- Google round trips dominate gdoc's time: about 0.6 to 1.3 s for one read, more for writes with read-backs. A fresh TLS handshake per process is part of each call. It is not isolated here, but `read`'s minimum of 559 ms bounds it from above.
- A persistent process with a kept connection would save at most part of that 0.5 to 1 s per call, and only on calls that are not model-bound.
- For Nail, permission prompts cost nothing today because an allowlist rule accepts gdoc calls. A colleague without that rule would get a prompt on each of the 4 to 12+ Bash calls per task. That cost is not measured here.

### Gaps
- The record has no Desktop (`via: chat`) runs at all this month, so MCP-path latency is not measured.
- The presence `at` timestamps have 1 s resolution, so sub-second step splits inside a command are not available.
- Google API time versus TLS setup versus JSON work inside one call is not split anywhere in the record.

---

## 4. How live review works today, and how long a comment takes to get an answer

### Takeaway
Live review is a skill-level loop over a blocking CLI call. `comments --since CURSOR --wait 9m` polls Drive every 2 s inside one process and returns on the first activity or after 9 minutes. The model then reads the window and works. From the moment the wait returns to the reply being in the thread is a median of about 52 s: about 13 s of gdoc calls and about 36 s of model time. Many waits return within 2 s with nothing to do, because the session's own reply comes back as news. Each such return still costs a full model turn.

### Cited Findings
- **The loop.** `$GDOC comments <url> --since "$CURSOR" --wait 9m` with Bash timeout 600000 (skills/gdoc-review/live.md:18-27). The binary polls Drive every two seconds and returns on the first activity after the cursor, or at the deadline with an empty window (live.md:29-32). Four outcomes are handled by the model (live.md:34-44).
- **Why nine minutes.** The Bash tool gives up at ten minutes (live.md:46-49). `maxWait` is one hour, chosen as "the bar that stops a wait from becoming the watcher principle 1 refuses to have" (go/cmd/gdoc/read.go:432-435).
- **Poll cost.** `waitInterval = 2 * time.Second`, deliberately not a flag (go/cmd/gdoc/read.go:425-430). A quiet poll reads the Drive listing only. "A quiet nine-minute wait is therefore one document read and 270 listings" (go/internal/comments/doc.go:79-85).
- **Design rule in the spec.** "There is no watcher, and the loop starts when Nail starts it" (docs/v2/SPEC.md:982-989). "A live session keeps the polling inside the call ... A call per poll would make the latency a minute rather than seconds ... every idle tick would spend a model turn on a quiet document" (docs/v2/SPEC.md:991-997). "a daemon or watcher" is out of scope (docs/v2/SPEC.md:43).
- **Own replies come back.** A window with only gdoc's own replies is not work, but it is still a returned call (live.md:76-83).
- **Suggestions are not watched.** A live session does not see a suggestion accepted or rejected (docs/backlog/a-live-session-does-not-watch-suggestions.md).
- **[PRES] All 120 `--wait` runs:**
  - 26 ran the full 9 min (about 540,000 ms), 2 were interrupted by the person, and 94 returned early.
  - Of the 94 early returns, 43 came back in 1.3 to 2.2 s, which is the first poll.
  - The command just before a wait that returned in under 2.5 s, in the same session, was: `reply` 14, another wait 10, `propose` 2, `suggestions` 1.
  - The command just after a wait that returned between 2.5 s and 535 s was: another wait 25, `read` 10, `propose` 3, `reply` 1.
- **[PRES] Wait return to the last write of the window**, 13 windows where the wait ran in the foreground (sessions f942, 543f, 13ab; three fd18/13ab windows dropped because the wait ran in the background and overlapped the work):
  - Span: 12, 13, 33, 35, 41, 46, 52, 65, 66, 72, 76, 91 and 98 s. Median 52 s.
  - gdoc time inside the span: median 12.9 s, total 174 of 700 s, which is 25%.
  - Model time (span minus gdoc): median 35.6 s, range 10.5 to 70.8 s.
  - The typical window runs read, (suggestions), propose, comments, reply. Example from f942 on 2026-10-18: the wait returned at 16:22:31, read took 1.5 s, suggestions 1.2 s, propose 13.0 s, comments 2.2 s and reply 1.4 s. The reply landed at 16:23:43, so 72 s, of which 19.2 s was gdoc.
- **[PRES] Gap between the end of a window's work and the next wait starting:** 6 to 16 s (median about 10 s). A comment written in that gap is seen on the next wait's first poll.
- **[TX] Backgrounded waits.**
  - In fd18 Claude Code moved the `--wait` Bash call to the background: "Command was moved to the background ... so that a message that arrived while it was running can reach you".
  - Results then arrived as 8 `<task-notification>` messages, each pointing at an output file the model has to read with a further tool call.
- **[TX] Token cost of a quiet window.**
  - In f942 there were 37 "no-op" rounds: a wait returned, the model took 1 turn, and it called wait again with no other tool. Each read a median of 725,416 context tokens (from cache) and wrote a median of 543 output tokens.
  - In 543f there were 18 no-op rounds: a median of 362,928 context tokens and 607 output tokens each.
  - Windows with work had a median of 4 to 5 turns, 3,954 to 5,330 output tokens and 1.36M to 1.80M context tokens read.

### Inferences
- **Comment to answer** is roughly: up to 2 s to detect, plus about 1.5 to 2 s for the poll that finds it, plus a median of about 52 s from the wait's return to the reply (p90 about 90 s). That makes about 55 s at the median when the session is already waiting. It adds up to about 10 s more when the comment lands while the model is between waits.
- About 75% of that time is the model. gdoc's share is mostly `propose`, with its three read-backs, and the re-read of the thread before each reply.
- **Each gdoc write tends to cause one extra round trip.** The write lands, the next wait's first poll sees gdoc's own reply as activity, and the model spends a full-context turn deciding it is not work. This is consistent with 27 of the 43 fast returns following a write or another wait. Whether each such window held only gdoc's own replies was not checked thread by thread.
- **A quiet hour costs about 6.7 model turns** (60 / 9 min), each re-reading the whole context. At 725k tokens that is roughly 4.8M cache-read tokens per quiet hour for f942's context size. This is an estimate from the measured median, not a measured hour.

### Gaps
- Drive's own delay between a person posting a comment and the comments listing showing it was not measured. The comment `created` times inside wait results were not compared with wait return times, because most results were narrowed by pipes.
- How many fast-return windows held only gdoc's own replies, versus an unmarked or resolved-thread change, needs the raw wait JSON, which the pipes dropped.

---

## 5. What `gdoc mcp` exposes, how it differs from the CLI, and what the two paths share

### Takeaway
`gdoc mcp` is one long-lived stdio process with 8 tools. Six are commands (read, comments, suggestions, reply, annotate, propose), plus `guide` and `login`. It runs the very same command code in-process through `safeDispatch`. It adds state the CLI cannot keep: a per-process guide code, a ledger of what was read and written, hold rules on risky writes, and a 10-minute memory of writes. It has no `--wait`, no export, publish, restyle, withdraw or build. Each tool call still builds a fresh guard and HTTP transport.

### Cited Findings
- **[MCP] tools/list returns 8 tools.** read, comments and suggestions are read-only. reply, annotate and propose are writes. login and guide are read-only.
- **Six command tools "in the order a review runs".** "withdraw, export, restyle, build, publish, probe and update are not here. Nothing in chat writes into a hub..." (go/cmd/gdoc/mcptools.go:276-283).
- **No wait in chat.** The `comments` tool takes `since` and `witness` but no `wait` (go/cmd/gdoc/mcptools.go:296-311).
- **Same code path.** `mcpSend` turns the arguments into an argv, pins the document, asks the hold rules, then calls `safeDispatch(ctx, argv, errOut, ch.set)`, the dispatcher the terminal uses (go/cmd/gdoc/mcptools.go:467-533, call at line 502; `safeDispatch` in go/cmd/gdoc/main.go:416). "the envelope the terminal would have printed as one text item".
- **Extra round trip on writes.** For a write, `mcpPin` reads the document to check the title and thread quote before sending (go/cmd/gdoc/mcptools.go:488-495, 656-770).
- **Per-process state:**
  - The guide code is 16 hex chars, made once per process. It is required on every tool but guide and login (go/internal/chat/doc.go:17-37).
  - The ledger of reads and writes is per process, and "nothing reaches disk" (go/internal/chat/doc.go:96-133).
  - A held write stays in the process for 30 minutes (go/cmd/gdoc/doc.go:106).
  - A write repeated within 10 minutes gets the kept answer (go/cmd/gdoc/mcptools.go:468-477).
- **Protocol** (go/internal/mcp/doc.go):
  - Written by hand, with 4 methods plus ping (lines 37-42).
  - One worker runs tool calls serially, because `gapi.Session` is not safe for concurrent use (lines 80-98).
  - Each call has a 200 s deadline, because Claude Desktop gives up at 240 s (lines 100-108).
  - `listChanged` is true, because a hold registers a one-time confirm tool (lines 30-31).
- **Instructions.** "Call guide before any other tool. It gives the rules for a review and the code every other tool needs." (go/cmd/gdoc/mcp.go:116-119).
- **Update check.** The mcp process may ask GitHub for updates once per process, bounded at 2 s (CLAUDE.md invariants; go/cmd/gdoc/mcp.go:172-199).
- **What the two paths share:** the command table, the `dispatch` code, the envelope (`internal/emit`), the guard per command, the token file, the settings and the presence record.
- **What differs:**
  - Transport: a process per call under Bash, versus one stdio process for a Desktop session.
  - The rules: SKILL.md files in Claude Code, versus the `guide` tool text (15.2 KB) plus tool cards in Desktop.
  - Chat-only safety: code, ledger, holds and write memory exist only in mcp.
  - The menu bar app talks to neither. It runs six fixed argv lists (`help --json`, `auth status`, `auth login`, `auth logout`, `update`, `update --desktop`), sends one SIGINT to a `comments --wait`, and reads the presence file (CLAUDE.md invariant "The menu bar app runs six argv lists...").
- **The review rules exist twice.** The review rules are embedded for chat from `skills/gdoc-review/review.md` (CLAUDE.md "Skills are linked, not copied"; `TestTheEmbeddedCoreIsTheSkillsCore`).
- **[PRES] No Desktop usage this month:** there are no `via: chat` lines at all.

### Inferences
- The MCP path already shows that "one long process, many calls" works inside the repository's rules: grants and state die with the process, and the guard is still per command.
- It does not save the per-call connection setup, because each dispatch builds a new transport.
- It is also not a live route today: it cannot wait.
- The three front ends (Bash CLI, MCP and the app's argv) share the command layer but not the transport or the rules text. This is the gap a single shared route would have to close.

### Gaps
- No measurement of MCP tool-call latency in Desktop exists in the record, so the Desktop path's speed is not compared with the Bash path.

---

## 6. Which repository rules shape the interface, and what each costs in speed or tokens

### Takeaway
The interface is shaped on purpose by the rules: one object and exit, never prompt, a guard and grants per process, facts not judgement, nothing trusts a success, and no daemon or watcher. Each rule is cheap in gdoc time. Their cost shows up as more model turns and more re-fetching, because no state survives between calls except what the model carries in its context.

### Cited Findings
- **One JSON object, then exit.** "Exactly one JSON object reaches stdout ... the exit code is 0 if and only if that object says ok" (go/cmd/gdoc/doc.go:11-14; CLAUDE.md invariant "One JSON object reaches stdout").
  - Cost: one Bash call and one model turn per command. No streaming of progress, which the Desktop path also lacks.
- **The binary never prompts.** "Nothing here reads stdin. A command missing something fails and says what is missing, and does not ask." (go/cmd/gdoc/doc.go:199-206; CLAUDE.md invariant).
  - Cost: every question to the person is a model turn in the skill: the cover question in restyle, the marks question in publish, the replace question in review and align.
- **Guard per process, and grants die with the process.**
  - `open()` builds a fresh policy per command (go/cmd/gdoc/read.go:337-361).
  - "A grant names one object and dies with the process" (CLAUDE.md invariants). Restyle's in-place grant is "opened by one line in one command, and dies with the process" (PRINCIPLES.md:90-92).
  - Cost: no connection reuse (go/internal/guard/transport.go:113-133) and a token-file read per call. Small in ms. It also rules out a shared long-lived session holding grants across calls.
- **Facts, not judgement.** "The binary prints facts and the skills judge" (CLAUDE.md invariants). The wait has "no field saying whether what came back was worth anything" (go/cmd/gdoc/read.go:437-440).
  - Cost: the model must judge every window, including echoes of its own replies (live.md:78-83), which is one full-context turn each.
  - The one exception is the chat hold list (go/internal/chat/doc.go:138-202).
- **Nothing trusts a success.** "Every write is read back through a route it did not go out on" (PRINCIPLES.md:75-77; CLAUDE.md invariants). The extra read-backs are visible in timing: `propose` median 6.8 s against `reply` 1.3 s, and publish's read_back step 4 to 6 s.
- **No daemon or watcher; the loop starts when Nail starts it** (docs/v2/SPEC.md:43, 982-997; go/cmd/gdoc/read.go:432-435).
  - Cost: live mode needs an open Claude Code session and re-enters the model every 9 minutes or on any activity.
  - Nothing answers while no session runs. "A comment written today waits, visible in the document, until a session looks" (SPEC.md:982-984).
- **No local record of handled comments.** "gdoc's reply in the thread is the receipt, and the document is the ledger. Every run recomputes the open work from the threads alone." (skills/gdoc-review/SKILL.md:69-75).
  - Cost: every run re-reads comments, document and suggestions, and the check before an old comment re-reads suggestions (live.md:97-102).
- **Read the thread again before posting** (skills/gdoc-review/review.md:69-87). Cost: one extra `comments` call, about 2 s, per reply.
- **Two messages per thread, never a progress feed** (skills/gdoc-review/review.md:52-67). The guard refuses PATCH and DELETE on comments, so a live "working on it" line is a reversal, not a patch (docs/backlog/live-review-shows-progress-in-the-thread.md).
- **Strict inputs through files.** Reply bodies and proposals go through files the skill writes first (SKILL.md:194-199; propose.md:20-28). Cost: one heredoc write per write, often chained in the same Bash call.
- **Every word costs a reader's attention** (PRINCIPLES.md:161-176). This keeps machinery out of documents, but not out of the tool answers that the model reads.

### Inferences
- The rules move cost from gdoc to the model. gdoc stays stateless, safe and fast, about 1 to 2 s per read. The model pays in turns and context re-reads to rebuild state each time and to judge facts that the binary will not judge.
- Any alternative that keeps a long-lived process must keep these properties: a per-command guard, grants that die with their scope, one object per answer and no judgement. Otherwise it is a DECISIONS.md change (CLAUDE.md "Breaking one is a decision...").

### Gaps
- None for this question. All rules were read in the repository.

---

## 7. Concrete friction points

### Takeaway
The biggest frictions are model-side, not gdoc-side:
- full-context turns spent on empty or echo windows in live mode;
- ad hoc Python narrowing of JSON in most calls;
- `read` re-fetched to grep slices;
- the help preamble every session;
- the backgrounded-wait detour;
- no watching at all outside an open session.

gdoc-side frictions are small: a fresh TLS connection per call, and the three read-backs on writes.

### Cited Findings
1. **Echo windows.** 43 of 120 waits came back on the first poll (1.3 to 2.2 s). The command just before was a gdoc write or another wait in 27 cases ([PRES]). Each costs one model turn re-reading 363k to 725k tokens ([TX] no-op round medians).
2. **Quiet-hour turns.** A quiet document still costs a turn every 9 minutes (live.md:46-49). 26 full 9-minute waits in the record each ended in a turn ([PRES]).
3. **Ad hoc narrowing.** 35% to 83% of gdoc Bash calls are piped through `python3 -c` or `head` ([TX]). This means output tokens spent writing parsers and fragile one-off code. It also risks the cursor rule (live.md:69-71). In f942, `2>/dev/null` was used on most calls, which hides gdoc's stderr words.
4. **Repeated full reads.** 17 `read` calls in 543f on a document of 55,825 text chars, each about 1.3 s and a full Docs fetch, to extract one passage each ([TX], [PRES] read median 1,310 ms).
5. **Re-read before reply.** One `comments` call (about 2 s) before every `reply` (review.md:69-87). It is visible as a `comments`, `reply` pair in every live window ([PRES] sequences).
6. **Help preamble.** `help --json` (9.6 KB) at every skill start, plus `help <command> --json` before each command in four skills ([FILES], [HELP]). It is about 2.4k to 3.5k tokens, and the `update` check inside it can fail: the measured envelope carried `"error": "...context deadline exceeded"` and a stale `latest_stable` v2.19.0 against installed v2.20.0.
7. **Backgrounded waits.** Claude Code moved `--wait` calls to the background in fd18, so each result needed a notification and then a file read ([TX]). That is an extra round trip, and the wait is no longer the session's single blocking call.
8. **No connection reuse.** Each command, including each MCP tool call, makes a new `http.Transport` (go/internal/guard/transport.go:113-133; go/cmd/gdoc/mcptools.go:502). The cost is a new TLS handshake per call. Its size is not isolated here.
9. **Write latency.** `propose` median 6.8 s, p90 17.4 s, max 87.7 s; publish about 10 to 12 s ([PRES]). This is driven by the read-backs (go/internal/propose/doc.go:1-14), and it is the largest gdoc share of comment-to-answer time.
10. **No live route outside Claude Code.** MCP `comments` has no `wait` (go/cmd/gdoc/mcptools.go:296-311). The menu bar app only reads the presence file and runs fixed argv lists. Suggestion accept and reject events are never watched (docs/backlog/a-live-session-does-not-watch-suggestions.md).
11. **Duplicate skill registration.** The same five skills are listed twice (local symlink and `altery:` plugin), as seen in this session's skill list. Different sessions invoked different copies ([TX]).
12. **Permission prompts.** These cost nothing for Nail: 246 of 247 calls were accepted by a config rule ([TX]). A colleague without an allowlist rule would get a prompt per Bash call (inference).
13. **Token refresh is not a friction.** It is saved to disk and shared across processes (go/internal/gapi/session.go:530-560).

### Inferences
- In order of measured cost: model turns on no-op and echo windows (tokens) come first. Model time per window (about 36 s median, speed) comes second. gdoc write read-backs (about 7 to 13 s per window) come third. Connection setup and process start come last.
- An alternative that cuts turns, for example by filtering the session's own echoes, returning only actionable windows, or keeping the read in memory between calls, would move the numbers more than a faster transport.
- Filtering echoes inside the binary touches the "facts, not judgement" invariant unless it is framed as a fact, for example "only gdoc's own replies changed".

### Gaps
- Not measured: Drive comment propagation delay; the TLS handshake share of a call; MCP latency in Desktop; the Claude Code permission cost for a colleague; token counts with a real tokenizer.
