# CLI vs MCP vs code execution: how an agent should call a tool (state as of October 2026)

Scope: what vendors and practitioners measured and argued about three ways for an agent to reach a tool: a shell CLI, an MCP server, or code the agent writes against an API ("code mode", programmatic tool calling). Every claim carries its URL. Dates are given because the numbers moved a lot between mid-2025 and mid-2026. Research date: 2026-10-27.

## 1. Anthropic's guidance and published numbers

### Takeaway
Anthropic publishes no direct "CLI vs MCP" verdict. Its guidance is about context cost and round trips: few, consolidated, high-signal tools; concise outputs; defer tool definitions (tool search); and let code, not the model, chain three or more dependent calls. The headline numbers are 150,000 to 2,000 tokens (98.7%) for code execution with MCP, an 85% token cut from tool search, and 37% fewer tokens from programmatic tool calling.

### Cited Findings

"Writing effective tools for AI agents" (Sep 11, 2025)
- A "concise" Slack response used 72 tokens against 206 for "detailed", about one third. The article recommends a `response_format` enum ("detailed"/"concise"), where detailed includes the ids needed for follow-up calls. [Anthropic](https://www.anthropic.com/engineering/writing-tools-for-agents)
- "we restrict tool responses to 25,000 tokens by default" in Claude Code. [Anthropic](https://www.anthropic.com/engineering/writing-tools-for-agents)
- Guidance: more tools do not mean better outcomes; do not just wrap API endpoints; consolidate multi-step workflows (example: one `schedule_event` that finds availability and books); prefer `search_contacts` over `list_contacts`; namespace tools (`asana_search`); return names, not `uuid`/`mime_type` (resolving UUIDs to names "significantly improves Claude's precision in retrieval tasks by reducing hallucinations"); paginate, filter and truncate with sensible defaults and tell the agent how to narrow; write actionable error messages with examples. [Anthropic](https://www.anthropic.com/engineering/writing-tools-for-agents)
- Evaluation advice: track accuracy, runtime, tool-call count, token use and tool errors per task; many redundant calls point at pagination or limits, many invalid-parameter errors point at descriptions. No exact accuracy figures are given in text. [Anthropic](https://www.anthropic.com/engineering/writing-tools-for-agents)
- The article does not compare CLI and MCP. [Anthropic](https://www.anthropic.com/engineering/writing-tools-for-agents)

"Code execution with MCP" (Nov 4, 2025, Adam Jones and Conor Kelly)
- Presenting MCP tools as files (`./servers/google-drive/getDocument.ts`) the agent lists and reads on demand cut a workflow "from 150,000 tokens to 2,000 tokens", "a time and cost saving of 98.7%". [Anthropic](https://www.anthropic.com/engineering/code-execution-with-mcp)
- A 2-hour meeting transcript passed through the model twice can add about 50,000 tokens; filtering a 10,000-row sheet in code lets the model see 5 rows. [Anthropic](https://www.anthropic.com/engineering/code-execution-with-mcp)
- Conditionals in code save "time to first token" latency; a polling loop in code (its example sleeps 5,000 ms) beats alternating tool calls and sleeps. [Anthropic](https://www.anthropic.com/engineering/code-execution-with-mcp)
- State: intermediate results go to files (`./workspace/leads.csv`) so work can resume; working code plus a SKILL.md becomes a reusable skill. [Anthropic](https://www.anthropic.com/engineering/code-execution-with-mcp)
- Caveat stated by Anthropic: generated code needs "a secure execution environment with appropriate sandboxing, resource limits, and monitoring", and "These infrastructure requirements add operational overhead and security considerations" that direct tool calls avoid. [Anthropic](https://www.anthropic.com/engineering/code-execution-with-mcp)

"Advanced tool use" (Nov 24, 2025): tool search, programmatic tool calling, tool use examples
- Measured tool-definition cost: GitHub 35 tools ~26K tokens; Slack 11 tools ~21K; Sentry 5 tools ~3K; Grafana 5 ~3K; Splunk 2 ~2K; five servers 58 tools ~55K; Jira alone ~17K; Anthropic internal peak 134K tokens before optimisation. [Anthropic](https://www.anthropic.com/engineering/advanced-tool-use)
- Tool Search Tool: ~500 tokens up front plus ~3K for 3-5 discovered tools, ~8.7K total versus ~77K traditional; "85%" token reduction. MCP-eval accuracy rose from 49% to 74% (Opus 4) and from 79.5% to 88.1% (Opus 4.5). Use it when definitions exceed 10K tokens or there are 10+ tools; less useful under 10 tools; keep the "three to five most-used tools always loaded". [Anthropic](https://www.anthropic.com/engineering/advanced-tool-use)
- Programmatic Tool Calling (PTC): average tokens 43,588 to 27,297 (37% fewer) on complex research tasks; internal knowledge retrieval 25.6% to 28.5%; GIA benchmark 46.5% to 51.2%; 20+ calls in one code block remove "19+ inference passes"; each round trip costs "hundreds of milliseconds to seconds" of inference. Most useful for "three or more dependent tool calls", less for "simple single-tool invocations". [Anthropic](https://www.anthropic.com/engineering/advanced-tool-use)
- Tool use examples: complex-parameter accuracy 72% to 90%; recommends "1-5 examples per tool". [Anthropic](https://www.anthropic.com/engineering/advanced-tool-use)

Programmatic tool calling docs (current, GA)
- PTC "reduces latency for multi-tool workflows"; on BrowseComp and DeepSearchQA it improved performance "by an average of 11 percent while using 24 percent fewer input tokens". [Claude docs](https://platform.claude.com/docs/en/agents-and-tools/tool-use/programmatic-tool-calling)
- On a 75-tool project-management benchmark, PTC cut billed input tokens "by roughly 38 percent with no change in task accuracy"; production requests with 10 to 49 tools see "typical token savings of 20 to 40 percent"; "calling 10 tools directly uses ~10x the tokens of calling them programmatically and returning a summary". [Claude docs](https://platform.claude.com/docs/en/agents-and-tools/tool-use/programmatic-tool-calling)
- PTC "trades a small fixed overhead (container startup, script generation)" for savings; good for fan-out ("checking 50 endpoints or looking up 20 records"); not for "Strictly sequential workflows where each call depends on Claude reasoning over the previous result". Tools are exposed as async Python functions and can run in parallel with `asyncio.gather`. [Claude docs](https://platform.claude.com/docs/en/agents-and-tools/tool-use/programmatic-tool-calling)
- Limits: idle containers reclaimed after about 5 minutes; a pending call times out after about 4 minutes (270 s in the example); tools from an MCP connector are not supported; `strict: true` tools not supported. [Claude docs](https://platform.claude.com/docs/en/agents-and-tools/tool-use/programmatic-tool-calling)

Agent Skills (Oct 16, 2025)
- Three levels of progressive disclosure: name and description preloaded; SKILL.md read when relevant; linked files read only as needed. Bundled scripts run "without loading the script or the PDF into context"; code is deterministic and cheaper than token generation (sorting example). No token counts given. MCP integration is described as future work. [Anthropic](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills)
- Claude Code docs: a skill shows only its name and description every session and the full file loads only when the task calls for it. [Claude Code best practices](https://code.claude.com/docs/en/best-practices)

Claude Agent SDK post (Sep 29, 2025)
- Tools are "the primary building blocks of execution" and should be the agent's core, frequent actions; bash is "a general-purpose tool"; code is "precise, composable, and infinitely reusable"; MCP gives "standardized integrations to external services, handling authentication and API calls automatically". No numbers. [Claude blog](https://claude.com/blog/building-agents-with-the-claude-agent-sdk)

Claude Code MCP behaviour (docs, current)
- Tool search is on by default: "MCP tools are deferred and discovered on demand"; only tool names and server instructions load at start; "no fixed per-server tool cap". `ENABLE_TOOL_SEARCH=auto` loads tools up front while they total under 10% of context. [Claude Code MCP docs](https://code.claude.com/docs/en/mcp)
- Output: warning at 10,000 tokens (fixed), default cap 25,000 tokens (`MAX_MCP_OUTPUT_TOKENS`), text results over 50,000 characters saved to a file. A call still running after two minutes moves to a background task. Stdio idle timeout is 30 minutes. [Claude Code MCP docs](https://code.claude.com/docs/en/mcp)
- Server instructions matter more with tool search: they tell Claude when to search for the server's tools; deferred tool descriptions are capped at 16,384 characters, and matching reads only the start, so key words belong first. [Claude Code MCP docs](https://code.claude.com/docs/en/mcp)

### Inferences
- Anthropic's own numbers favour two things regardless of transport: a small, consolidated tool surface and keeping intermediate data out of the model. The transport (CLI or MCP) is not what its numbers measure.
- The 98.7% figure is one illustrative workflow with large intermediate payloads, not a benchmark average. The 37% and 20-40% PTC numbers are closer to what a typical multi-tool workload gets.

### Gaps
- I found no Anthropic page that recommends CLI over MCP or the reverse for a local tool. The Claude Code best-practices page fetched on 2026-10-27 no longer carries a "CLI tools are the most context-efficient" line that some blogs attribute to it; I could not confirm that wording.
- Anthropic gives no wall-clock latency numbers for tool search or PTC, only token and accuracy deltas.

## 2. Cloudflare Code Mode and similar ideas

### Takeaway
Code Mode is a strong token result for very large APIs (about 1,000 tokens for ~2,500 endpoints) but Cloudflare has published no accuracy or latency measurements for it. Its premise that models write code better than they make tool calls is argued, not tested.

### Cited Findings
- First Code Mode post (Sep 26, 2025): convert MCP tools into a TypeScript API and have the model write code; "LLMs are better at writing code to call MCP, than at calling MCP directly". No benchmarks, though it says "the results are striking". Reasoning: tool calls rely on special tokens learned from "synthetic training data", while code comes from "millions of open source projects". Isolates "can start in a handful of milliseconds using only a few megabytes of memory"; `fetch()` and `connect()` are blocked; the supervisor holds credentials. The whole generated API was loaded into context, with on-demand browsing left as future work. [Cloudflare](https://blog.cloudflare.com/code-mode/)
- Code Mode MCP server (Feb 20, 2026, modified Jul 15, 2026): two tools, `search()` and `execute()`, each taking JavaScript; uses "around 1,000 tokens" against "1.17 million tokens" for an equivalent tool-per-endpoint server; "reduces the number of input tokens used by 99.9%"; measured with tiktoken. The page description also says "over 2 million tokens", which the page does not reconcile with 1.17 million. [Cloudflare](https://blog.cloudflare.com/code-mode-mcp/)
- Third-party count: native MCP with only required parameters ~244,047 tokens, Code Mode ~1,069 tokens. [Glama](https://glama.ai/mcp/servers/ubnx8r8i7p)
- Cloudflare's own comparison: client-side code mode "requires the agent to ship with secure sandbox access" (it names Anthropic's PTC and Goose as implementations); CLIs need a shell, "which not every environment provides", and bring "a much broader attack surface than a sandboxed isolate"; dynamic tool search "requires a search function that must be maintained and evaluated" and "each matched tool still uses tokens". Code Mode "solves context costs for a single API" only. No latency or accuracy measurements. [Cloudflare](https://blog.cloudflare.com/code-mode-mcp/)
- Armin Ronacher (Aug 18, 2025): an MCP server exposing "a single tool, that accepts programming code" took his Playwright setup "from ~30 tool definitions to 1"; a `state` variable persists between calls so pagination loops run without inference. In a pexpect demo, the first debugging session took about 45 seconds and ~7 tool calls; re-running the generated script took one call and under 5 seconds. He also says the Playwright version is "not promising enough yet" and does not beat Playwright MCP "in the current form". [Armin Ronacher](https://lucumr.pocoo.org/2025/8/18/code-mcps/)
- Peter Steinberger's mcporter turns any MCP server into a standalone CLI (`generate-cli`) or a typed TypeScript API (`emit-ts`), with a daemon mode for stateful servers; it cites Anthropic's code-execution post as its motivation. [GitHub: steipete/mcporter](https://github.com/steipete/mcporter)

### Inferences
- Code Mode's gain scales with API size. For a tool with a handful of commands (like gdoc's seventeen), the definition cost Code Mode removes is already small, so the remaining argument for code is chaining and filtering, which is the PTC argument.
- The reuse pattern in Ronacher's demo (45 s and ~7 calls first time, under 5 s and 1 call on re-run) is the same idea as a skill that ships a script: pay for discovery once, then run deterministic code.

### Gaps
- No published accuracy or end-to-end latency comparison of Code Mode against direct MCP tool calls from Cloudflare. I found no independent benchmark either.

## 3. The "CLI beats MCP" argument and what was measured

### Takeaway
The 2025 case for CLIs was mostly about MCP definition bloat (GitHub MCP at 23K to 55K tokens). Measured results: ScaleKit found CLI 4x to 32x cheaper in tokens and more reliable (100% vs 72%) on GitHub reads in March 2026, but the MCP failures were all network timeouts to a remote server. By mid-2026, with deferred tool loading and leaner MCP outputs, two independent re-tests found MCP and CLI within a few percent. Zechner's careful 2025 benchmark found protocol choice "doesn't matter as much as you'd think".

### Cited Findings

Mario Zechner, "MCP vs CLI" benchmark (Aug 15, 2025)
- 120 runs (3 tasks x 4 tools x 10 reps) in Claude Code: terminalcp MCP, terminalcp CLI, tmux, screen. terminalcp MCP and CLI both 100% success; screen 67% overall (0% on one task). MCP version "23% faster (51m vs 66m)" and "2.5% cheaper ($19.45 vs $19.95)" than the same tool as a CLI. [Mario Zechner](https://mariozechner.at/posts/2025-08-15-mcp-vs-cli/)
- With Haiku, terminalcp MCP used about 35k tokens against 1.3-2M for the CLI tools, which he attributes to Claude Code's per-bash-call security checks. debug-lldb averages: tmux $0.3729 / 1m 28.7s; terminalcp CLI $0.3865 / 1m 37.2s; terminalcp MCP $0.4804 / 1m 22.2s; screen $0.6003 / 1m 46.2s. [Mario Zechner](https://mariozechner.at/posts/2025-08-15-mcp-vs-cli/)
- Conclusion: the choice "doesn't matter as much as you'd think"; tool design, docs and token efficiency matter more than protocol; if users have a shell, build a good CLI first and add MCP on top later. [Mario Zechner](https://mariozechner.at/posts/2025-08-15-mcp-vs-cli/)

Mario Zechner, "What if you don't need MCP at all?" (Nov 2, 2025)
- "Playwright MCP has 21 tools using 13.7k tokens (6.8% of Claude's context)"; "Chrome DevTools MCP has 26 tools using 18.0k tokens (9.0%)"; his bash-script README "has a whopping 225 tokens". CLI output can go to files, chain in one bash call, or be processed with code, while MCP results pass through context. He prefers explicit `@README.md` references over skill auto-discovery, which he found unreliable. [Mario Zechner](https://mariozechner.at/posts/2025-11-02-what-if-you-dont-need-mcp/)

Armin Ronacher
- Aug 2025: CLI tools are "sometimes platform-dependent, version-dependent, and at times undocumented"; "Doing multiple turns is very hard with CLI tools" because the agent must manage session state; Claude Code's Haiku security preflight slows multi-turn CLI use. [Armin Ronacher](https://lucumr.pocoo.org/2025/8/18/code-mcps/)
- Dec 13, 2025: he moved all MCPs to skills (Sentry last, Playwright replaced by a skill). Loading Sentry MCP cost about 8k tokens up front. Changing tool definitions mid-conversation breaks the cache: "You would basically start from scratch and pay full token rates plus cache write cost". "MCP servers have no desire to maintain API stability". [Armin Ronacher](https://lucumr.pocoo.org/2025/12/13/skills-vs-mcp/)

Simon Willison
- Aug 22, 2025, citing Geoffrey Huntley: the GitHub MCP "defines 93 additional tools and swallows another 55,000" tokens; with `gh` the agent gets the same functions "for a token cost close to zero". [Simon Willison](https://simonwillison.net/2025/Aug/22/too-many-mcps/)
- Oct 16, 2025: GitHub's MCP "famously consumes tens of thousands of tokens of context"; "each skill only takes up a few dozen extra tokens"; "Almost everything I might achieve with an MCP can be handled by a CLI tool instead". [Simon Willison](https://simonwillison.net/2025/Oct/16/claude-skills/)

Peter Steinberger (Oct 2025, "Just talk to it", undated page)
- "Almost all MCPs really should be clis"; MCPs are "a constant cost and garbage in my context"; "Use GitHub's MCP and see 23k tokens gone" (almost 50,000 at launch). A CLI needs no explanation: the model reads `--help` on first use. Still uses chrome-devtools-mcp occasionally. [Peter Steinberger](https://steipete.me/posts/just-talk-to-it)

GitHub `gh` vs GitHub MCP: ScaleKit benchmark (Mar 11, 2026)
- Claude Sonnet 4, five read-only tasks, 75 runs (25 per mode). Median tokens: language and license CLI 1,365 / CLI+Skills 4,724 / MCP 44,026 (32x); PR details 1,648 / 2,816 / 32,279 (20x); repo metadata 9,386 / 12,210 / 82,835 (9x); merged PRs 5,010 / 6,107 / 33,712 (7x); latest release 8,750 / 6,860 / 37,402 (4x). p < 0.05. [ScaleKit](https://scalekit.com/blog/mcp-vs-cli-use)
- Success: CLI 25/25, CLI+Skills 25/25, MCP 18/25 (72%). All seven MCP failures were `ConnectTimeout` to GitHub's remote Copilot MCP server; none were protocol or tool-call errors. A skill (~800 tokens of `gh` tips) cut tool calls and latency by about a third against naive CLI. Projected cost at 10,000 ops/month: CLI ~$3.20, MCP ~$55.20, MCP via a filtering gateway ~$5. ScaleKit still recommends MCP when agents act for other people (per-user OAuth, tool boundaries, audit). [ScaleKit](https://scalekit.com/blog/mcp-vs-cli-use)
- Later counts of the GitHub MCP: 93 tools ~55,000 tokens (full); ~42,000 tokens (another count); 26-tool standard toolset ~4,200 tokens, a 92% cut via `GITHUB_TOOLSETS`. Crossover where MCP composition pays off is "somewhere in the low single digits" of composed operations per turn. The post is promotional for Unblocked, and its tool-search figure ("51,000 to 8,500 (46.9%)") is internally inconsistent. [Unblocked](https://getunblocked.com/blog/github-mcp-token-cost/)

Playwright CLI vs Playwright MCP
- Microsoft's `playwright-cli` README: for coding agents "that is the best fit"; CLI calls "are more token-efficient: they avoid loading large tool schemas and verbose accessibility trees"; MCP fits loops that "benefit from persistent state" and "long-running autonomous workflows". Sessions keep cookies and storage between CLI calls; headless sessions shut down after an hour idle. No numbers. [GitHub: microsoft/playwright-cli](https://github.com/microsoft/playwright-cli)
- Widely cited figure: ~114,000 tokens via MCP against ~26,000-27,000 via CLI for one task; its primary source is unclear. A re-run on `@playwright/mcp@0.0.78` vs `@playwright/cli@0.1.17` (Jul 24, 2026), 7-step login flow, found ~3,580 vs ~3,620 characters returned, "Within ~1% of each other", because current MCP also writes snapshots to disk and returns a path. Remaining difference: MCP definitions ~18k characters vs the CLI skill ~12.6k characters. [DEV Community](https://dev.to/aswani25/playwright-cli-vs-playwright-mcp-which-should-you-use-with-claude-code-1olh)
- Checkly (Jul 30, 2026): Claude Code, Playwright MCP vs `playwright-cli` plus skill, 3 runs each; MCP sessions ended at "48k to 50k tokens of context", CLI at "45k to 48k". Conclusion: "There's no big difference in using CLIs or MCPs these days"; the author uses MCP in Claude Desktop and CLIs in Claude Code. [Checkly](https://www.checklyhq.com/blog/mcp-vs-cli-token-efficiency/)

Vercel d0 (Dec 22, 2025): many tools replaced by bash
- Success 4/5 to 5/5; average time 274.8 s to 77.4 s (3.5x faster); tokens ~102k to ~61k (37% fewer); steps ~12 to ~7; tools 17 to 2 in the code sample (`ExecuteCommand`, `ExecuteSQL`). Small sample (5 tasks). [Vercel](https://vercel.com/blog/we-removed-80-percent-of-our-agents-tools)

### Inferences
- Most of the 2025 "CLI beats MCP" gap was definition cost of large third-party servers plus verbose outputs. Tool search (on by default in Claude Code) and file-path outputs removed much of it, which is why the 2026 re-tests converge.
- The one benchmark where the same tool was offered both ways (Zechner, terminalcp) found MCP slightly faster and cheaper, not slower. For a tool its author controls, output design decides cost more than transport.
- ScaleKit's reliability gap is a remote-server network effect. It says nothing about a local stdio server such as `gdoc mcp`.

### Gaps
- No benchmark compares a small, well-designed local CLI with the same tool as a local stdio MCP server under Claude Code 2026 with tool search on, measuring wall-clock time. Zechner's 2025 run is the closest.
- I did not find a 2026 Willison or Steinberger post that revisits the claim after tool search shipped.

## 4. Cost of tool definitions, tool count, large outputs, and output shape

### Takeaway
Tool count and output size both measurably hurt accuracy. Selection accuracy can be very low with large catalogs (13.62% baseline in RAG-MCP) and drops 7% to 85% as catalogs grow (LongFuncEval). Large tool responses cut answer retrieval by 7% to 91%. Best practice: few consolidated tools, concise default output with ids available on request, pagination and filters, files for bulky payloads.

### Cited Findings
- RAG-MCP (arXiv 2505.03275, May 6, 2025): retrieving tools before selection raised accuracy from 13.62% to 43.13% and cut prompt tokens by "over 50%". [arXiv](https://arxiv.org/abs/2505.03275)
- LongFuncEval (arXiv 2505.10570, IBM, 2025): performance drops 7% to 85% as the number of tools grows; 7% to 91% drop in answer retrieval as tool response length grows; 13% to 40% degradation over longer multi-turn conversations; position of the right tool in the catalog matters more at long context. [arXiv](https://arxiv.org/pdf/2505.10570)
- Anthropic's tool search lifted MCP-eval accuracy 49% to 74% (Opus 4) and 79.5% to 88.1% (Opus 4.5), which shows definition load costs accuracy, not only tokens. [Anthropic](https://www.anthropic.com/engineering/advanced-tool-use)
- Independent measure of Claude Code tool search on a 65-tool setup (Sep 2026, secondary): first request about 64% smaller, full task only 46% smaller, because lazy loading adds a round trip to fetch the tool. [Towards AI, via search summary](https://pub.towardsai.net/claude-code-tool-search-nearly-halves-your-context-bill-a14defdf42f3)
- Output-shape guidance: concise vs detailed (72 vs 206 tokens); 25,000-token default cap; pagination, range selection, filtering, truncation with instructions to narrow; names over UUIDs; actionable errors. [Anthropic](https://www.anthropic.com/engineering/writing-tools-for-agents)
- Claude Code warns at 10,000 tokens of MCP output, caps at 25,000, and saves text over 50,000 characters to a file. [Claude Code MCP docs](https://code.claude.com/docs/en/mcp)
- PTC docs: "Return structured data: JSON or other machine-readable formats work best for programmatic processing". [Claude docs](https://platform.claude.com/docs/en/agents-and-tools/tool-use/programmatic-tool-calling)
- File-path outputs were what closed the Playwright MCP vs CLI token gap (a screenshot inline can add 5,000-8,000 tokens vs ~50 for a path, per secondary summaries). [DEV Community](https://dev.to/aswani25/playwright-cli-vs-playwright-mcp-which-should-you-use-with-claude-code-1olh)

### Inferences
- For gdoc, the measured risk is not its tool count (small) but the size of what `read` and `comments` return. A concise default and a way to ask for detail or a range, or to write bulk to a file, is what the evidence supports.

### Gaps
- Blog claims that accuracy "slips at 10-15 tools" or "at 50 tools" have no clear primary source; I left them out.

## 5. Latency: process start, model turns, parallel calls, batching, streaming, events

### Takeaway
Model turns dominate wall-clock time: each round trip costs "hundreds of milliseconds to seconds", while a local stdio MCP call costs tens of milliseconds and a process start is small for a compiled binary. The levers that cut time are fewer turns (batching, consolidated tools, code chaining), parallel calls, and push instead of poll. Client behaviour matters: Claude Desktop was measured dispatching MCP calls 3-6 s apart while the server answered in 50 ms.

### Cited Findings
- Each model round trip takes "hundreds of milliseconds to seconds"; 20+ calls in one code block remove "19+ inference passes". [Anthropic](https://www.anthropic.com/engineering/advanced-tool-use)
- PTC: "The model is not re-sampled between the tool calls inside one code execution"; fan-out runs in parallel with `asyncio.gather`; no benefit for strictly sequential, reasoning-dependent chains. [Claude docs](https://platform.claude.com/docs/en/agents-and-tools/tool-use/programmatic-tool-calling)
- Vercel: fewer, general tools cut steps ~12 to ~7 and time 274.8 s to 77.4 s. [Vercel](https://vercel.com/blog/we-removed-80-percent-of-our-agents-tools)
- ScaleKit: a ~800-token skill with `gh` tips cut tool calls and latency by about a third against naive CLI. [ScaleKit](https://scalekit.com/blog/mcp-vs-cli-use)
- Claude Desktop issue #76854 (opened Jul 12, 2026, closed "not planned" Sep 3, 2026): two MCP calls in one Desktop turn arrived 3.07 s and later 6.09 s apart; direct HTTP no-op calls min 35 ms / p50 50 ms / p95 69 ms; stdio `tools/call` 54-78 ms (initialize 133 ms); a terminal `claude` session with 3 parallel calls had them 58-155 ms apart. Workarounds: batch operations into one tool call; prefer terminal sessions. [GitHub issue](https://github.com/anthropics/claude-code/issues/76854); [mirror with details](https://claudeissues.com/issue/76854-desktop-mcp-tool-calls-dispatched-3-6-s-apart-while-the-server-answers-in-50-ms)
- Small benchmark: a stdio MCP call ~2 ms vs ~0.01 ms in-process, server start ~570 ms once; negligible next to inference. Single blog, indicative only. [Juejin](https://juejin.cn/post/7662273240987582479)
- Zechner: Claude Code's per-bash-call security checks inflated CLI token use (Haiku: ~35k for MCP vs 1.3-2M for CLI tools); Ronacher names the same Haiku preflight as a multi-turn slowdown. [Mario Zechner](https://mariozechner.at/posts/2025-08-15-mcp-vs-cli/); [Armin Ronacher](https://lucumr.pocoo.org/2025/8/18/code-mcps/)
- Polling in code beats alternating tool calls and sleep commands. [Anthropic](https://www.anthropic.com/engineering/code-execution-with-mcp)
- Push: a Claude Code channel is "an MCP server that pushes events into a Claude Code session so Claude can react to things happening outside the terminal"; one-way or two-way; research preview, Claude Code v2.1.80+, opt in with `--channels`, server declares `claude/channel`; events arrive only while the session is open. [Claude Code channels reference](https://code.claude.com/docs/en/channels-reference); [Claude Code MCP docs](https://code.claude.com/docs/en/mcp)
- On the v2 runtime, a channel server that negotiates MCP revision 2026-07-28 "can't deliver channel messages", so Claude Code does not register it as a channel. [Claude Code MCP docs](https://code.claude.com/docs/en/mcp)
- MCP 2025-11-25 added experimental Tasks ("call-now, fetch-later", states pending/working/input_required/completed/failed/cancelled, polling). [WorkOS](https://workos.com/blog/mcp-2025-11-25-spec-update)
- MCP 2026-07-28 goes stateless: no initialize handshake or `Mcp-Session-Id`, per-request `_meta`, cacheable list responses (`ttlMs`), "Multi Round-Trip Requests" for mid-call input, Tasks and MCP Apps as extensions. [MCP blog, release candidate](https://blog.modelcontextprotocol.io/posts/2026-07-28-release-candidate/); [Flavio Copes](https://flaviocopes.com/mcp-2026-07-28-stateless/)
- Claude Code: a main-conversation MCP call still running after two minutes moves to a background task; stdio idle timeout 30 minutes; `list_changed` notifications refresh tools without reconnecting. [Claude Code MCP docs](https://code.claude.com/docs/en/mcp)

### Inferences
- For gdoc, Go process start is not the bottleneck. The time a person waits is model turns, Google API calls, and client dispatch. A command that does read, act and verify in one call saves more than any transport change.
- In Claude Desktop the issue above makes many small MCP calls in one turn slow; one batched call is the measured workaround. That argues for coarse-grained `gdoc mcp` tools.
- For live reaction, Claude Code channels are the only push path into a session found. A blocking long-poll command (which gdoc already has as `comments --wait`) is the portable alternative. Channels and stateless 2026-07-28 MCP do not currently mix in Claude Code.

### Gaps
- No controlled study of wall-clock time to a user-visible result for CLI vs MCP vs code mode. No measured process-start cost for a Go CLI under Claude Code's bash tool (with its permission and safety checks) was found.
- Whether Claude Desktop still shows the 3-6 s dispatch gap after Sep 2026 is unknown; the issue was closed "not planned" without a cached maintainer reply.

## 6. Where each approach wins, by task shape

### Takeaway
Evidence by shape: CLI plus skill wins for one-shot or few-call work by a developer agent with a shell, especially against large third-party servers. MCP wins where there is no shell (Claude Desktop), where state or a live session must persist, where per-user auth and audit matter, and as the only push channel. Code execution wins for fan-out and for chains of three or more dependent calls with bulky intermediate data. For a small tool the author controls, output design and call granularity matter more than transport.

### Cited Findings
- One-shot reads, developer as user: CLI 4x-32x fewer tokens and 100% vs 72% success on GitHub reads; recommendation "CLI + Skills" when the developer is the user. [ScaleKit](https://scalekit.com/blog/mcp-vs-cli-use)
- Acting for other people: MCP's per-user OAuth, tool boundaries and audit trails. [ScaleKit](https://scalekit.com/blog/mcp-vs-cli-use); MCP handles "authentication and API calls automatically". [Claude blog](https://claude.com/blog/building-agents-with-the-claude-agent-sdk)
- No shell available: CLIs need a shell, "which not every environment provides". [Cloudflare](https://blog.cloudflare.com/code-mode-mcp/)
- Persistent state and long sessions: Playwright says MCP fits loops that "benefit from persistent state" and long autonomous workflows; CLI fits coding agents. [GitHub: microsoft/playwright-cli](https://github.com/microsoft/playwright-cli). Ronacher: multi-turn state is "very hard with CLI tools". [Armin Ronacher](https://lucumr.pocoo.org/2025/8/18/code-mcps/)
- Multi-step and fan-out: PTC for "three or more dependent tool calls" and fan-out over many items; not for single calls or strictly sequential reasoning chains. [Anthropic](https://www.anthropic.com/engineering/advanced-tool-use); [Claude docs](https://platform.claude.com/docs/en/agents-and-tools/tool-use/programmatic-tool-calling)
- Very large APIs: Code Mode, ~1,000 tokens for ~2,500 endpoints. [Cloudflare](https://blog.cloudflare.com/code-mode-mcp/)
- Live/event-driven: channels push events into a Claude Code session. [Claude Code channels reference](https://code.claude.com/docs/en/channels-reference). Polling in code beats tool-call-and-sleep loops. [Anthropic](https://www.anthropic.com/engineering/code-execution-with-mcp)
- Same tool, both transports: near parity (terminalcp MCP 23% faster, 2.5% cheaper). [Mario Zechner](https://mariozechner.at/posts/2025-08-15-mcp-vs-cli/). 2026 re-tests of Playwright: within ~1% to a few percent. [DEV Community](https://dev.to/aswani25/playwright-cli-vs-playwright-mcp-which-should-you-use-with-claude-code-1olh); [Checkly](https://www.checklyhq.com/blog/mcp-vs-cli-token-efficiency/)
- Practice split reported by practitioners: MCP in Claude Desktop, CLIs in Claude Code. [Checkly](https://www.checklyhq.com/blog/mcp-vs-cli-token-efficiency/)
- Write-heavy vs read-heavy: Anthropic recommends MCP tool annotations to flag destructive or open-world actions. [Anthropic](https://www.anthropic.com/engineering/writing-tools-for-agents)

### Inferences
- gdoc's current split (CLI plus skills in Claude Code, `gdoc mcp` in Claude Desktop) matches what practitioners report doing in 2026. The evidence does not suggest switching transport.
- The bigger wins the evidence points to are inside the tool: concise default output with a detail mode, coarse commands that do a whole step (act plus read-back) in one call, files or paths for bulk content, and fewer model turns per user-visible result.
- Code execution would help gdoc only for fan-out (many threads or documents at once). For one comment reply the chain is sequential and depends on model judgement, which is the case where PTC gives no gain.

### Gaps
- I found no evidence specific to write-heavy tools with verification read-backs (like gdoc's) comparing transports.
- No source compares event-driven designs (push channel vs long-poll CLI vs MCP Tasks) on latency to reaction.
