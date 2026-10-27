# How tool makers connect their own tool to AI agents (case studies, as of October 2026)

Scope: what vendors of developer and productivity tools actually shipped for agents, why, and what they measured. Read for gdoc: a Go CLI that returns one JSON object per call, plus `gdoc mcp` for Claude Desktop. Sources are dated where the page gives a date. Secondary sources are marked "(secondary)".

## 1. Tools that ship both a CLI and an MCP server: how they split the work, and what they recommend

### Takeaway
By late 2026 almost no vendor picks one route. The common shape is three parts: a CLI (or SDK) for the shell, a hosted or local MCP server for clients without a shell or for live account data, and a skill pack that teaches the agent the workflow and points it at one of the two. Two vendors moved the other way for coding agents (Playwright added a CLI next to its MCP; Google's Workspace CLI removed its MCP mode), both for context cost.

### Cited Findings

Playwright (Microsoft)
- Microsoft ships `@playwright/cli` next to Playwright MCP. The README calls the CLI "the best fit" for coding agents because it "is more token-efficient" and avoids "loading large tool schemas and verbose accessibility trees into the model context" ([microsoft/playwright-cli README](https://github.com/microsoft/playwright-cli)).
- The same README keeps MCP for "specialized agentic loops that benefit from persistent state, rich introspection, and iterative reasoning", such as exploratory automation and self-healing tests ([playwright-cli README](https://github.com/microsoft/playwright-cli)).
- Design detail: after each command the CLI writes the page snapshot to a file (`.playwright-cli/page-<timestamp>.yml`) instead of into the model's context; `find` searches a snapshot. The browser lives across calls in a named session (`-s=`), in memory by default, `--persistent` saves the profile to disk, and a headless session shuts down after one hour idle. Skills install with `playwright-cli install --skills`; agents can also work from `--help` alone. The example timestamp is 2026-02-14 ([playwright-cli README](https://github.com/microsoft/playwright-cli)).

Chrome DevTools (Google)
- `chrome-devtools-mcp` "is available as a Model-Context-Protocol (MCP) server and as a CLI", and the CLI lets you "control the browser from your terminal without MCP" (ends with `chrome-devtools stop`, which implies a background browser across calls). A `--slim` flag "exposes a reduced set of tools" for basic tasks. The package ships seven skills, including `chrome-devtools` (MCP) and `chrome-devtools-cli` (shell scripts), and warns that "outdated skills can tell your agent to use tools or parameters that changed" ([ChromeDevTools/chrome-devtools-mcp README](https://github.com/ChromeDevTools/chrome-devtools-mcp)).
- The changelog lists `--slim` as a v0.18.0 feature "for maximum token savings" ([changelog mirror on Glama](https://glama.ai/mcp/servers/@ChromeDevTools/chrome-devtools-mcp/blob/505089c2884428ef7d040a36dd0b50ee30f0e623/CHANGELOG.md), secondary mirror).

GitHub (gh and github-mcp-server)
- github-mcp-server runs remote (hosted at `api.githubcopilot.com/mcp/`, OAuth or PAT) or local (Docker or binary). Tools are grouped into toolsets; the default is `context`, `repos`, `issues`, `pull_requests`, `users`. The README says "Enabling only the toolsets that you need can help the LLM with tool choice and reduce the context size." It has `--read-only` (wins over `--tools`) and an `--insiders` mode ([github/github-mcp-server README](https://github.com/github/github-mcp-server)).
- A `dynamic` toolset starts with three tools (`list_available_toolsets`, `get_toolset_tools`, `enable_toolset`) so the client enables toolsets at runtime, behind `--dynamic-toolsets`, in beta ([instagit deep-dive, Feb 2026](https://instagit.com/github/github-mcp-server/dynamic-toolsets-on-demand-discovery-mcp-server.md), secondary).
- GitHub's own agent, Copilot CLI, ships with the GitHub MCP server built in: it "is already available without any additional configuration" and can be turned off with `--disable-builtin-mcps` ([GitHub Docs: Adding MCP servers for Copilot CLI](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers)). So GitHub's first-party agent uses MCP, not `gh`, for GitHub data.
- Practitioner critique: Mario Zechner argues the GitHub MCP server "reimplements functionality that's already available in the GitHub CLI" ([Zechner, MCP vs CLI, 2025-08-15](https://mariozechner.at/posts/2025-08-15-mcp-vs-cli/)).

Google Workspace CLI (`gws`), the closest analogue to gdoc
- `gws` is a CLI for all Workspace APIs (Drive, Docs, Gmail, Chat...), built "with agents as the primary audience" by Justin Poehnelt, Google ([Poehnelt, "You need to rewrite your CLI for AI agents", 2026-03-04](https://justin.poehnelt.com/posts/rewrite-your-cli-for-ai-agents/)). The README says "This is not an officially supported Google product." ([googleworkspace/cli README](https://github.com/googleworkspace/cli)).
- It had a `gws mcp --services drive,gmail` mode that exposed all commands as JSON-RPC tools over stdio ([Poehnelt](https://justin.poehnelt.com/posts/rewrite-your-cli-for-ai-agents/)). Release v0.8.0 (7 March 2026) lists "Remove `mcp` command" with no stated reason ([googleworkspace/cli releases](https://github.com/googleworkspace/cli/releases?q=mcp&expanded=true)). A third-party wrapper says the cut was because the server exposed 200–400 tools and bloated client context ([socket.dev: gws-mcp-server](https://socket.dev/npm/package/gws-mcp-server), secondary, unconfirmed). Google's separate hosted Workspace MCP servers are a different product, in developer preview (same secondary source).
- Today the README says all output (success, errors, download metadata) is structured JSON, offers `--page-all` (NDJSON, one line per page), `gws schema drive.files.list` for introspection, `--dry-run`, and `--sanitize` through Model Armor to "scan API responses for prompt injection before they reach your agent" ([googleworkspace/cli README](https://github.com/googleworkspace/cli)). The README states both "40+ agent skills" and "100+ Agent Skills (SKILL.md files)", which conflict ([README](https://github.com/googleworkspace/cli)).

Stripe
- Stripe's agent page leads with "skills and plugins". Quickstart: `npm install -g @stripe/cli` then `stripe agent setup`. The agent plugin "bundles the Stripe MCP server and agent skills and automatically updates them"; if the agent has no plugins, "combine the Stripe MCP server and agent skills yourself". Remote MCP at `https://mcp.stripe.com`; append `.md` to any docs URL for Markdown; a machine-readable skills catalog at `/.well-known/skills/index.json`; `stripe docs` reads the docs in the terminal (CLI v1.43.3+) ([Stripe: Agents and AI on Stripe](https://docs.stripe.com/building-with-llms)).

Supabase
- Supabase docs list a `supabase` skill that triggers on Supabase products, the CLI or the MCP server, plus a Postgres best-practices skill, installable with `npx skills add supabase/agent-skills` or together with the MCP server via the Supabase plugin ([Supabase docs: AI skills](https://supabase.com/docs/guides/getting-started/ai-skills), via search summary).
- Supabase's position (2026-04-09): "MCP alone is not enough." The MCP-only agent "never called `search_docs`" even though it was available; "The bottleneck is context, not capability." ([Supabase blog: agent skills](https://supabase.com/blog/supabase-agent-skills)). Numbers in section 6.

Vercel
- Vercel's setup for coding agents is three steps: install the CLI once per machine, add guidance (Vercel plugin or standalone skills), connect the remote MCP (`npx -y add-mcp https://mcp.vercel.com -g`, OAuth). Vercel advises keeping human confirmation on for every MCP write, since the agent gets the authorizing user's access ([Vercel docs: Vercel MCP](https://vercel.com/docs/ai-tooling/vercel-mcp), via search summary). Vercel also built `npx skills` (formerly add-skill) to install skills across agents ([same search summary](https://vercel.com/docs/ai-tooling/vercel-mcp)).

Sentry
- Sentry ships a new CLI described as "Built by humans and agents, for humans and agents", with natural-language input, "Structured JSON output for scripts and pipelines", and project auto-detection from `.env` and the codebase ([cli.sentry.dev](https://cli.sentry.dev/)). npm package `sentry`, v0.45.0 at time of listing ([socket.dev](https://socket.dev/npm/package/sentry/overview/0.45.0), secondary). A skill is listed at [skills.sh/getsentry/cli](https://skills.sh/getsentry/cli/sentry-cli).
- Sentry's own split (2026-10-02): Seer is the in-house agent (root cause, plan, draft PR); the MCP server (via the agent plugin) brings Sentry data into the coding agent and can call Seer first; the CLI is for agents with a shell: "Agents that can run shell commands get all of this without needing MCP support." Sentry makes no token or speed claim for either ([Sentry blog: Seer, MCP, CLI or coding agent](https://blog.sentry.io/seer-mcp-cli-or-coding-agent/)).

Linear
- Linear ships only a hosted MCP server (`https://mcp.linear.app/mcp`, read-only at `/mcp/readonly`, Streamable HTTP, OAuth 2.1 or API key) with tools for issues, projects and comments; it documents `claude mcp add` and `codex mcp add`, and has no first-party CLI ([Linear docs: MCP](https://linear.app/docs/mcp)). Its agent route is separate: agents as app users fed by webhooks (section 5).

Notion
- Notion moved from a local open-source server to a hosted remote MCP because setup was "too high" a barrier, and hosting means "Notion can quickly ship improvements without requiring users to download updated packages." It rewrote `create-pages` and `update-page` around "Notion-flavored Markdown" because hierarchical JSON blocks needed several requests; Markdown gives better content density per token and fewer tool calls. No measurements given (2025-07-15) ([Notion blog: hosted MCP inside look](https://www.notion.com/blog/notions-hosted-mcp-server-an-inside-look)).
- Community servers claim large savings over Notion's official MCP (e.g. "92% fewer tokens"), self-measured with tiktoken and not verified ([easy-notion-mcp listing](https://mcpservers.org/servers/grey-iris/easy-notion-mcp), secondary).

Cloudflare
- Cloudflare's Code Mode MCP server exposes its whole API (2,500+ endpoints) through two tools, `search()` over the OpenAPI spec and `execute()` of code, in about 1,000 tokens regardless of API size. Its comparison: native MCP with full schemas would be 2,594 tools and about 1,170,000 tokens; required-params-only about 244,000; Code Mode cuts input tokens 99.9% (tiktoken, Cloudflare's own measurement, 2026-02-20) ([Cloudflare blog: Code Mode MCP](https://blog.cloudflare.com/code-mode-mcp/), via search summary; [InfoQ, April 2026](https://infoq.com/news/2026/04/cloudflare-code-mode-mcp-server)).

Anthropic (as protocol owner)
- Anthropic's own pattern: present MCP servers as code APIs in a filesystem, load only needed definitions, filter data in the sandbox. Measured: about 150,000 tokens down to 2,000 (98.7%) for one workflow (2025-11-04) ([Anthropic: Code execution with MCP](https://www.anthropic.com/engineering/code-execution-with-mcp)).

### Inferences
- The vendors whose main agent users are coding agents with a shell (Playwright, Chrome DevTools, Google Workspace, Sentry) now ship a CLI as a first-class agent route. Vendors whose users are mostly in chat apps or hosted agents (Linear, Notion, Stripe, Vercel) centre on a hosted remote MCP. gdoc has both audiences (Claude Code skills and Claude Desktop), which matches the "both, plus skills" pattern rather than either extreme.
- `gws` is the nearest peer to gdoc (Go-free but same domain, same Google APIs). It kept the CLI, JSON-only output, dry run and skills, and dropped its auto-generated MCP. Its failure mode was tool count (one tool per API method), not MCP as such. gdoc's MCP has a small hand-written tool set, so the `gws` lesson is "do not generate hundreds of tools", not "do not ship MCP".
- Playwright and Chrome DevTools both pair a CLI with a long-lived process holding state (a browser). The CLI call is short, the state is not. This is the same split as "one JSON object and exit" plus a daemon.

### Gaps
- No vendor in this list publishes a measured CLI-versus-MCP comparison of its own product (Playwright, Chrome DevTools, GitHub, Sentry, Stripe give reasons, not numbers). Third-party numbers are in section 6.
- I found no official GitHub statement recommending `gh` over the MCP server for agents, or the reverse.
- I did not confirm the gws MCP removal reason from Google; only a third-party wrapper states it.
- Vercel and Supabase doc claims come from search summaries; I did not open those two pages directly.

## 2. "Agent-native" CLI design

### Takeaway
The published guidance converges on: JSON always (or when stdout is not a terminal), a way to shrink output (field masks, concise mode, snapshot to file), runtime schema introspection, dry run on writes, input hardening against agent mistakes, consolidated "do the job" commands over thin API wrappers, and a skill or context file that states the invariants. There is a live counter-argument against replacing flat flags with one JSON payload.

### Cited Findings
- Poehnelt's rules for `gws` (2026-03-04): accept the raw API payload via `--json`/`--params` and keep human flags too; `--output json`, `OUTPUT_FORMAT=json`, or NDJSON when stdout is not a TTY; runtime introspection (`gws schema ...`) "rather than relying on static docs in a system prompt"; field masks, with the context file saying "ALWAYS use field masks when listing"; "Agents pay per token and lose reasoning capacity with every irrelevant field"; NDJSON pagination; reject path traversal, control characters below 0x20, `?`/`#` in ids and `%` in names, "Treat all agent input as adversarial"; `--dry-run` for mutations; ship SKILL.md files that encode invariants (dry-run first, confirm before writes, add fields to list calls); expose MCP last. Seven retrofit steps in this order: JSON output, input validation, schema/`--describe`, field masks, dry run, context or skill files, MCP surface ([Poehnelt](https://justin.poehnelt.com/posts/rewrite-your-cli-for-ai-agents/)).
- Counterpoint: Microsoft's Waldek Mastykarz, "Don't rewrite your CLI for agents", argues flat arguments can be easier for agents than nested JSON ([feedbagel summary](https://feedbagel.com/post/don8217t-rewrite-your-cli-for-agents), secondary). Speakeasy describes retrofitting its CLI after human-oriented features "became obstacles" ([Speakeasy: engineering an agent-friendly CLI](https://speakeasy.com/blog/engineering-agent-friendly-cli), via search summary).
- Anthropic's tool-design guidance (2025-09-11): a `response_format` enum, "concise" vs "detailed"; in a Slack example detailed was 206 tokens and concise 72, about a third. Detailed keeps the ids (`thread_ts`, `channel_id`) later calls need. Consolidate: one `schedule_event` instead of `list_users` + `list_events` + `create_event`; `search_logs` instead of `read_logs`; `get_customer_context` instead of three getters. Claude Code caps tool responses at 25,000 tokens by default. Resolving UUIDs into meaningful names or 0-indexed ids "improved Claude's retrieval precision by reducing hallucinations" ([Anthropic: Writing effective tools for agents](https://www.anthropic.com/engineering/writing-tools-for-agents)).
- Output to files, not context: Playwright CLI writes each page snapshot to a YAML file and returns a path ([playwright-cli](https://github.com/microsoft/playwright-cli)); Anthropic recommends agents write intermediate results to files so later runs resume ([Anthropic: Code execution with MCP](https://www.anthropic.com/engineering/code-execution-with-mcp)).
- Context auto-detection removes arguments: Sentry CLI detects the project from `.env` and the codebase ([cli.sentry.dev](https://cli.sentry.dev/)); Sentry's blog mentions selectors like `@latest` ([Sentry blog](https://blog.sentry.io/seer-mcp-cli-or-coding-agent/)).
- Prompt-injection hygiene on read output: `gws --sanitize` (Model Armor, warn or block) ([googleworkspace/cli](https://github.com/googleworkspace/cli)).
- MCP spec 2025-11-25 says input validation errors should come back as tool execution errors, not protocol errors, "to enable model self-correction" ([MCP 2025-11-25 changelog](https://modelcontextprotocol.io/specification/2025-11-25/changelog)).

### Inferences
- gdoc already meets most of this list (one JSON object, strict decoding, refusal by name, `verified` read-backs, text quotes instead of indexes). The items it may not have in the published sense: a compact versus full output switch, runtime schema introspection (`help --json` partly covers it), and field masks on large reads.
- "Do the whole job" commands are the consolidation Anthropic recommends; gdoc's `publish` (build, upload, three read-backs) is already that shape. The cost of many small commands is round trips through the model, which section 6 quantifies.

### Gaps
- No vendor publishes an A/B of compact versus full CLI output beyond Anthropic's single Slack example.
- I did not open Mastykarz's or Speakeasy's posts directly.

## 3. Long-lived local daemons that agents and CLIs talk to

### Takeaway
A very common pattern for state-heavy tools is "thin CLI, one call per command, talking to a long-lived local process over a socket". The CLI keeps the agent-facing contract simple (one call, one result) while the daemon keeps auth, caches, open connections and live state. Nobody I found publishes latency numbers for this split; the gains are stated as state, auth sharing and liveness.

### Cited Findings
- agterm (umputun): `agtermctl` "drives a running agterm over a local unix socket, one command per invocation"; `tree --json` dumps the whole model; "Terminal output is not streamed", so the agent calls `session text` to read. Agents report state ("active, blocked, or completed") onto their session row through installable status hooks. A skill "teaches Claude Code or Codex the control model and the agtermctl commands", installed from the app menu or as a plugin (`claude plugin marketplace add umputun/agterm`). The page lists "events" among covered areas but does not describe subscription ([umputun/agterm README](https://github.com/umputun/agterm)).
- Obsidian: the official CLI (installer 1.12.7+) "requires the Obsidian app to be running" and "connects to the running Obsidian instance"; the first command launches the app if closed. "Anything you can do in Obsidian you can do from the command line", including `eval` of JavaScript inside the app and `dev:screenshot` returning base64 PNG; developer commands "allow agentic coding tools to automatically test and debug" ([Obsidian help: CLI](https://obsidian.md/help/cli)). Third-party listings say it shipped in v1.12, February 2026, over IPC ([skills.sh listing](https://www.skills.sh/openclaw/skills/obsidian-cli), secondary). Obsidian's CEO kepano publishes an obsidian-skills pack that teaches agents the CLI ([x-cmd listing](https://x-cmd.com/install/obsidian-skills), secondary).
- 1Password: the CLI talks to the desktop app over XPC on macOS (`NSXPCConnection` via a helper), a Unix socket on Linux (group `onepassword-cli` checked), a named pipe on Windows (Authenticode checked both ways). Authorization is per account and per terminal session, extends to sub-shells on macOS and Linux, expires after "10 minutes of inactivity" and has "a hard limit of 12 hours" ([1Password: app integration security](https://www.1password.dev/cli/app-integration-security.md)). Gains stated: biometric unlock and "Seamlessly sign to the 1Password accounts you've added to the app"; the app logs each CLI command, time and calling application ([1Password: app integration](https://www.1password.dev/cli/app-integration/)).
- Tailscale: the `tailscale` CLI talks to `tailscaled`; the CLI has a `--socket=<path>` flag ([Tailscale CLI reference](https://tailscale.com/docs/reference/tailscale-cli.md)). The LocalAPI is HTTP over that local socket; third-party ports of the Go client (`tslocal`) use the Unix socket, while on macOS one client connects over localhost with a random port and generated password ([tslocal on docs.rs](https://docs.rs/crate/tslocal/0.3.1), secondary; [Tailscale forum](https://forum.tailscale.com/t/how-to-discover-other-tailscale-devices-on-android-and-ios/1139), secondary).
- Playwright CLI keeps a browser process alive across commands in named sessions, closing after one idle hour ([playwright-cli](https://github.com/microsoft/playwright-cli)). Chrome DevTools CLI has a `stop` command, which implies the same ([chrome-devtools-mcp](https://github.com/ChromeDevTools/chrome-devtools-mcp)).
- Language servers: Claude Code has a built-in LSP tool (goToDefinition, findReferences, documentSymbol, hover, getDiagnostics), language servers distributed as plugins with a `.lsp.json`; one guide says it arrived in v2.0.74 behind `ENABLE_LSP_TOOL=1` ([claudelog: LSP tool](https://claudelog.com/faqs/what-is-lsp-tool-in-claude-code/), secondary; [claude-contrib/claude-languages](https://github.com/claude-contrib/claude-languages)). An LSP server is itself a long-lived process that keeps the index warm between queries.
- Claude Code channels run as a long-lived MCP subprocess that can also listen on a local HTTP port (example: 127.0.0.1:8788) and push events into the session ([Claude Code: channels reference](https://code.claude.com/docs/en/channels-reference)). Details in section 5.

### Inferences
- The daemon pattern buys three things relevant to gdoc: warm auth (token refreshed once, not per call), warm caches (document, revision, thread cursor), and a place to hold a wait (comments --wait) without the skill re-launching a process. The CLI contract can stay "one call, one JSON object" while a background process does this, as agterm, Obsidian and 1Password show.
- Every daemon example above ships with an access check on the socket (process group, code signature, generated password, per-terminal authorization). A gdoc daemon would need the same, and it would be a new door next to the guard.
- `gdoc mcp` already is a long-lived process. One option the case studies point to is letting the CLI become a client of a running `gdoc mcp` or sibling daemon when one exists, and falling back to the direct call when not (Obsidian launches the app if closed). This is an inference, not a vendor recommendation.

### Gaps
- None of these vendors publish per-call latency for "CLI talking to daemon" versus "CLI doing the work itself".
- Docker, Raycast and Hammerspoon were in scope; I did not research them within the time budget. Docker CLI to dockerd over a Unix socket is well known but I have no fetched source here.
- agterm's socket path, wire protocol and event subscription are documented at agterm.com/commands, which I did not fetch.

## 4. Vendor skill packs, and skills plus scripts versus an MCP server

### Takeaway
Skills became the standard "how to use our tool" layer in 2026 (Stripe, Supabase, Vercel, Google Workspace, Chrome DevTools, Sentry, Playwright, Obsidian, agterm all ship them). The measured evidence says skills mostly fix *whether and when* the agent uses the tool, not the tool's ability. Skills that are not invoked do nothing, and Vercel found a passive AGENTS.md index beat an on-demand skill.

### Cited Findings
- Anthropic Agent Skills (2025-10-16; open standard 2025-12-18): three levels of progressive disclosure, metadata at startup, SKILL.md body when relevant, linked files "only as needed". Scripts run as tools: Claude "can run this script without loading either the script or the PDF into context", which makes the workflow "consistent and repeatable". Anthropic frames skills as complementing MCP, "teaching agents more complex workflows that involve external tools" ([Anthropic: Equipping agents with Agent Skills](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills)).
- Anthropic's own docx skill is a routing table plus bundled scripts and external CLIs: create with docx (npm), edit by unzip then XML edit then zip, read with `pandoc -t markdown`; bundled scripts `scripts/office/soffice.py`, `scripts/office/validate.py`, `scripts/merge_runs.py`, `scripts/accept_changes.py`, `scripts/comment.py`; verify by rendering to PDF and then images ([anthropics/skills docx SKILL.md](https://github.com/anthropics/skills/blob/main/skills/docx/SKILL.md)). No server; every step is a shell call.
- Vercel (2026-01-27), Next.js 16 API evals: baseline 53%, skill default 53%, skill with explicit instructions 79%, AGENTS.md docs index 100%. "The skill was never invoked in 56% of eval cases." The index was compressed from about 40KB to about 8KB ([Vercel: AGENTS.md outperforms skills](https://vercel.com/blog/agents-md-outperforms-skills-in-our-agent-evals)).
- Supabase (2026-04-09), six scenarios per condition, LLM judge, small sample: Claude Code Opus 4.6 58% baseline, 50% MCP only, 67% MCP + skill; Sonnet 4.6 46%, 58%, 71%; Codex GPT-5.4 71%, 71%, 88%; GPT-5.4 Mini 42%, 63%, 71% ([Supabase: agent skills](https://supabase.com/blog/supabase-agent-skills)).
- Supabase Evals (2026-07-31), real MCP server and CLI in containers: in the Build stage Opus 5 and Kimi K3 hit 100% with no skill; skills lifted Sonnet 5 78% to 100%, GPT-5.6 Sol 89% to 100%, GPT-5.4 mini 78% to 89%. Claude Code checks docs in under 40% of scenarios even with skills; Codex reads about 8 doc pages per scenario versus about 2 for Claude Code. The Postgres skill loaded in about 1 in 10 sessions until its description was rewritten, then 60% ([Supabase: Introducing Supabase Evals](https://supabase.com/blog/introducing-supabase-evals)).
- Chrome DevTools ships separate skills for the MCP route and the CLI route, and says skills must be updated with each server or CLI update ([chrome-devtools-mcp](https://github.com/ChromeDevTools/chrome-devtools-mcp)). Stripe exposes a machine-readable skills catalog and an auto-updating plugin ([Stripe](https://docs.stripe.com/building-with-llms)).
- Mario Zechner's benchmark conclusion: tool design, documentation quality and token efficiency matter more than MCP versus CLI; "for a new tool where users already have a shell", build a good CLI first, then add MCP on top if needed ([Zechner 2025-08-15](https://mariozechner.at/posts/2025-08-15-mcp-vs-cli/)).

### Inferences
- For gdoc, the skill is where judgement lives (the repo already says so). The evidence says the skill's description and its first lines decide whether the right command gets called at all; gdoc's skills should be measured on activation, not only on outcome.
- A skill plus CLI is the same mechanism Anthropic uses for its own docx skill; an MCP server adds value where there is no shell (Claude Desktop) or where state must live across calls.

### Gaps
- No vendor compared "skill + CLI" against "skill + MCP" for the same tool and the same tasks. Supabase used MCP and CLI together; Vercel tested docs, not tools.
- Cursor rules and the AGENTS.md convention as such were not researched beyond the Vercel eval.

## 5. Event-driven tools that wake an agent

### Takeaway
There are three wake-up shapes: webhook into a hosted agent (Linear agents, Sentry Seer, GitHub Actions running Claude Code), push into an already open local session (Claude Code channels), and polling (MCP tasks, SSE polling). Vendors state acknowledgment deadlines (Linear: 10 seconds), not end-to-end times. The only end-to-end figure found is "a few minutes" for Seer root cause.

### Cited Findings
- Claude Code channels (research preview, v2.1.80+, launched about 2026-03-20): a channel is an MCP server spawned over stdio that declares `capabilities.experimental['claude/channel']` and emits `notifications/claude/channel` with `content` and `meta`; the event lands in context as `<channel source="..." ...>`. A server can also listen on a local HTTP port so "anything that can send an HTTP POST, like a CI pipeline, a monitoring alert, or a curl command, can push events to Claude". "Claude Code doesn't acknowledge notifications"; if not loaded as a channel, events are dropped silently. "Events queue into the session and are processed in order"; several arriving while Claude is busy are delivered together on the next turn. Two-way channels add a reply tool; `claude/channel/permission` relays tool approvals with a five-letter id. Custom channels need `--dangerously-load-development-channels` or an org allowlist; Team and Enterprise must enable channels ([Claude Code: channels reference](https://code.claude.com/docs/en/channels-reference)). Launch date and version from secondary coverage ([lilting.ch](https://lilting.ch/en/articles/claude-code-channels-mcp-event-push), secondary).
- Linear agents: delegation or @mention starts a session; the agent receives an `AgentSessionEvent` webhook (`created`) and must "Emit a `thought` activity within 10 seconds to acknowledge the session has begun"; session state updates from the activities the agent emits ([Linear developers: agents](https://linear.app/developers/agents)).
- Sentry Seer: Autofix triggers automatically on issues with 10+ events in 14 days above a fixability threshold, and hands root cause and plan to Cursor, Claude or Copilot cloud agents, which return a branch or draft PR ([Sentry blog, 2026-10-02](https://blog.sentry.io/seer-mcp-cli-or-coding-agent/)). Via MCP, Seer root cause "takes a few minutes to complete, depending on how many issues you send" ([Sentry cookbook: Cursor](https://sentry.io/cookbook/fix-issues-sentry-mcp-cursor), via search summary).
- GitHub Actions: `claude-code-action` activates on "@claude mentions, issue assignments, or executing automation tasks with explicit prompts", runs "entirely on your own GitHub runner", and shows "Visual progress indicators with checkboxes that dynamically update" in the thread. No latency statements ([anthropics/claude-code-action](https://github.com/anthropics/claude-code-action)).
- MCP spec 2025-11-25 added experimental tasks "to enable tracking durable requests with polling and deferred result retrieval" (SEP-1686), and lets servers close SSE streams at will so clients poll and resume (SEP-1699) ([MCP 2025-11-25 changelog](https://modelcontextprotocol.io/specification/2025-11-25/changelog)).

### Inferences
- Channels are the closest fit to "live reaction" for gdoc in Claude Code: a gdoc watcher could push "new ai? comment in doc X" into the open session instead of the skill polling with `comments --wait`. The limits are real: research preview, allowlist or dangerous flag, no delivery acknowledgment, events only while the session runs, and the sender-gating rule (an ungated channel is a prompt-injection vector). For gdoc the "sender" is a Google Docs comment author, which the repo already treats as untrusted.
- The Linear 10-second acknowledgment is a useful benchmark for "the person sees something happened": an early visible marker matters more than total time.

### Gaps
- No vendor published event-to-answer latency (p50 or p95) for Linear agents, Slack agents, Copilot coding agent, claude-code-action or Seer.
- Slack agents were not researched within the time budget.

## 6. Measured outcomes (speed, tokens, success)

### Takeaway
Measured results disagree because they measure different things. For single, simple calls a CLI is cheaper in tokens (no schemas). For multi-step work, raw MCP beats a CLI and code-mode beats both by a wide margin, because the cost driver is the number of model round trips. Well-designed tools win regardless of transport.

### Cited Findings
- Port of Context (2026-03-16), 12 Stripe tasks, Claude Sonnet 4.6, all passed 12/12: CLI (shell tool around `stripe`) $2.22, 711,555 tokens (59,296 per task); raw `@stripe/mcp` $1.60, 506,970 (42,248); Code Mode MCP $0.98, 294,924 (24,577). Create-invoice: CLI 19 steps, $1.52, 497,556 tokens; raw MCP 12 steps, $0.53, 168,480; Code Mode 4 steps, $0.13, 38,847. Simple balance read: CLI 3,001 tokens versus raw MCP 19,172. Conclusion: the client's use of the protocol is the main cost driver, not transport ([Port of Context](https://portofcontext.com/blog/cli-vs-mcp-vs-code-mode)).
- Smithery study, 756 runs, Haiku 4.5, GPT 5.4, Sonnet 4.6, on GitHub REST, Linear GraphQL and a bus API: GitHub, native MCP 91.7% success, 28,838 median tokens, 10.4s median latency; "CLI tree + descriptions" 83.3%, 82,942 tokens, 24.9s. All 826 GitHub tools: MCP 100.0%, 76,101 tokens, 21.4s; CLI + search 87.5%, 79,375, 26.1s. Raw API 53.0%, raw API + specs 75.8% ([Nordic APIs summary, 2026-05-13](https://nordicapis.com/mcp-vs-cli-which-is-better-for-agentic-ai/), secondary; Smithery's own post [smithery.ai/blog/mcp-vs-cli-is-the-wrong-fight](https://smithery.ai/blog/mcp-vs-cli-is-the-wrong-fight) returned 404 when fetched). Caveat: the CLI arm was a generated CLI with descriptions, not `gh`.
- Mario Zechner (2025-08-15), terminal control, 120 runs in Claude Code: MCP and CLI versions of the same tool both 100% success; MCP "23% faster (51m vs 66m)" and "2.5% cheaper ($19.45 vs $19.95)"; he attributes part of the CLI overhead to Claude Code's per-call bash security checks (his interpretation) ([Zechner](https://mariozechner.at/posts/2025-08-15-mcp-vs-cli/)).
- Anthropic: code execution over MCP, about 150,000 to 2,000 tokens (98.7%) ([Anthropic, 2025-11-04](https://www.anthropic.com/engineering/code-execution-with-mcp)); concise versus detailed response, 72 versus 206 tokens ([Anthropic, 2025-09-11](https://www.anthropic.com/engineering/writing-tools-for-agents)).
- Cloudflare: about 1,000 tokens for 2,500+ endpoints versus about 1,170,000 for full schemas (99.9% fewer input tokens) ([Cloudflare blog](https://blog.cloudflare.com/code-mode-mcp/), via search summary).
- Skill effects on success: Vercel 53% to 100% with an AGENTS.md index ([Vercel](https://vercel.com/blog/agents-md-outperforms-skills-in-our-agent-evals)); Supabase MCP-only to MCP + skill +8 to +17 points ([Supabase](https://supabase.com/blog/supabase-agent-skills)).
- mcp-vs-cli-bench v1.0.0 (2026-08-08, UPC, Salamanca, UPV/EHU): seven scaffoldings including Claude Code and Codex, five models, a six-operation workflow on a private GitHub repo, MCP arm versus shell-only arm, scored by reading repo state back; results are in the paper, not on the record page ([record](https://produccioncientifica.usal.es/documentos/6a776d7c5afd4024cfca2c42)).

### Inferences
- For gdoc's goals: tokens and round trips are dominated by how many model turns a job takes. A skill that runs `read`, then `comments`, then `reply`, then `propose` as separate model turns pays the per-turn context each time. A command that does the whole job in one call (or a code-mode style script the skill runs once) cuts that most, more than moving between CLI and MCP.
- The simple-call result (CLI 3,001 versus MCP 19,172 tokens) favours the CLI for gdoc's one-shot skills in Claude Code, where no MCP schema needs loading. MCP's advantage shows up in long sessions with many calls and in clients without a shell.

### Gaps
- No study measured time-to-visible-result for a person (e.g. "reply appears in the document"), which is gdoc's main speed metric.
- I could not read Smithery's primary post (404) or the mcp-vs-cli-bench paper's numbers.
