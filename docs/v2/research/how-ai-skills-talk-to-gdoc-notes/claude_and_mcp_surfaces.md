# Claude Code, Claude Desktop and MCP: surfaces for connecting a local tool to an agent (as of 2026-10-27)

Scope note: every claim below carries a URL. Docs pages on code.claude.com are undated living docs; where a page names a minimum Claude Code version, that version is quoted instead of a date. Claims from secondary blogs are marked as such.

## 1. Claude Code plugins: what a plugin can bundle, and how skills reach scripts and tools

### Takeaway
A plugin can bundle skills, commands, subagents, hooks, MCP servers (including `.mcpb` bundles), LSP servers, output styles, themes, workflows, monitors (experimental), channels, a `bin/` folder put on PATH, and two default settings. For gdoc the important pieces are `bin/` (the binary on PATH with no install step), `mcpServers` (gdoc as a stdio server started by the plugin), `experimental.monitors` (a persistent background watcher whose stdout lines reach Claude), and `channels` (an MCP server that pushes into the session).

### Cited Findings
- Manifest `plugin.json` fields: `skills`, `commands`, `agents`, `hooks`, `mcpServers`, `lspServers`, `outputStyles`, `workflows`, `channels`, `userConfig`, `settings`, `dependencies`, and an `experimental` object holding `themes`, `monitors` and `evals` "whose manifest shape may still change" ([Plugin manifest reference](https://code.claude.com/docs/en/plugins-reference)).
- Standard layout: `skills/`, `commands/`, `agents/`, `hooks/hooks.json`, `.mcp.json`, `.lsp.json`, `output-styles/`, `workflows/`, `themes/`, `monitors/monitors.json`, `bin/`, `settings.json` ([Plugin manifest reference](https://code.claude.com/docs/en/plugins-reference)).
- `bin/`: "Files here are on the Bash tool's `PATH` while the plugin is enabled, so Claude runs them as bare commands." Plugin `bin/` comes after the user's own PATH entries, so it cannot shadow system commands. "claude.ai and Cowork don't install a plugin that has this directory" ([Plugin manifest reference](https://code.claude.com/docs/en/plugins-reference); [Plugin components](https://code.claude.com/docs/en/plugins/components)).
- `mcpServers` accepts a JSON file, an inline map, or an MCP bundle path or `https://` URL ending `.mcpb`/`.dxt`, which Claude Code extracts into `.mcpb-cache/` ([Plugin manifest reference](https://code.claude.com/docs/en/plugins-reference)). Plugin MCP tools are named `mcp__plugin_<plugin>_<server>__<tool>`; the server shows as `plugin:<plugin>:<server>` ([Plugin components](https://code.claude.com/docs/en/plugins/components)).
- Monitors: each entry has `name`, `command`, `description`, optional `when` (`"always"` at session start, or `"on-skill-invoke:<skill>"` to start the first time a skill runs). "A monitor is a shell command that runs in the background for the whole session. What it prints reaches Claude as notifications." Monitors do not start with `-p`, run outside the sandbox with full user permissions, cannot reference `${user_config.*}`, and keep running if the plugin is disabled mid-session ([Plugin components](https://code.claude.com/docs/en/plugins/components); [Plugin manifest reference](https://code.claude.com/docs/en/plugins-reference)).
- Channels in a plugin: a `channels` entry binds to one of the plugin's MCP servers by `server` key and can prompt for its own `userConfig` ([Plugin manifest reference](https://code.claude.com/docs/en/plugins-reference)).
- Settings: only `agent` and `subagentStatusLine` take effect from a plugin; `agent` runs one of the plugin's agents as the main thread ([Plugin components](https://code.claude.com/docs/en/plugins/components)).
- Path variables: `${CLAUDE_PLUGIN_ROOT}` (install dir, changes on update), `${CLAUDE_PLUGIN_DATA}` (`~/.claude/plugins/data/<id>/`, survives updates), `${CLAUDE_PROJECT_DIR}`. In skill, command and agent Markdown, `${...}` is substituted inline when content loads; the variables are NOT in the environment of commands Claude runs through Bash ([Plugin manifest reference](https://code.claude.com/docs/en/plugins-reference)).
- A `CLAUDE.md` at the plugin root is not loaded; instructions belong in a skill ([Plugin components](https://code.claude.com/docs/en/plugins/components)).
- A local stdio server in a plugin runs in Claude Code and in Cowork on the user's machine in Claude Desktop, but not on claude.ai; to reach claude.ai, reference a remote `https://` server ([Plugin components](https://code.claude.com/docs/en/plugins/components)).

### Inferences
- gdoc already ships through `.claude-plugin/`. Adding `bin/gdoc` to the plugin would remove the separate install step for Claude Code, but would block install through claude.ai or Cowork org sync.
- A plugin monitor with `when: "on-skill-invoke:gdoc-review"` is the closest documented fit for "live review starts a watcher only when the review skill runs".

### Gaps
- No skill-specific variable such as a skill directory path was confirmed in the pages read; only `${CLAUDE_PLUGIN_ROOT}` substitution in skill Markdown is documented here.
- Whether plugin monitors have a deadline like the Monitor tool (5 to 30 minutes) is not stated; the docs say "for the whole session".

## 2. Hooks: events, blocking, context injection, latency

### Takeaway
There are about 34 hook events. Many can block, and many can add `additionalContext` that Claude reads on the next model request. Hooks can be shell commands, HTTP POSTs, a direct MCP tool call, a prompt, or an agent. The one documented way a background hook can wake an idle session is `asyncRewake: true`.

### Cited Findings
- Events (from the hooks reference): SessionStart, Setup, UserPromptSubmit, UserPromptExpansion, PreToolUse, PermissionRequest, PermissionDenied, PostToolUse, PostToolUseFailure, PostToolBatch, Notification, MessageDisplay, SubagentStart, SubagentStop, TaskCreated, TaskCompleted, Stop, StopFailure, TeammateIdle, InstructionsLoaded, ConfigChange, CwdChanged, DirectoryAdded, FileChanged, WorktreeCreate, WorktreeRemove, PreCompact, PostCompact, PreModelSwitch, PostModelSwitch, Elicitation, ElicitationResult, SessionEnd ([Hooks reference](https://code.claude.com/docs/en/hooks)).
- Blocking: PreToolUse (`permissionDecision: "deny"`), UserPromptSubmit, Stop (keeps Claude working), SubagentStop, PostToolBatch, TaskCreated, TaskCompleted, PreCompact, ConfigChange, Elicitation and others can block; Notification, SessionEnd, FileChanged, CwdChanged cannot ([Hooks reference](https://code.claude.com/docs/en/hooks)).
- Context: `hookSpecificOutput.additionalContext` "adds text to Claude's context at the point where the hook fired. Claude reads it on the next model request." SessionStart can also return `initialUserMessage`, `watchPaths`, `sessionTitle`, `reloadSkills`. `systemMessage` shows a warning to the user, not to Claude ([Hooks reference](https://code.claude.com/docs/en/hooks)).
- Hook types: `command`, `http` (JSON POST to a URL), `mcp_tool` (calls a tool on a configured MCP server), `prompt`, `agent` (experimental). SessionStart supports only `command` and `mcp_tool` ([Hooks reference](https://code.claude.com/docs/en/hooks)).
- Default timeouts: command/http/mcp_tool 600 s; prompt 30 s; agent 60 s; command/http/mcp_tool drop to 30 s on UserPromptSubmit and model-switch events, 10 s on MessageDisplay; SessionEnd shares a 1.5 s budget (up to 60 s) ([Hooks reference](https://code.claude.com/docs/en/hooks)).
- `async: true` runs a command hook in the background without enforcing its timeout. `asyncRewake: true` "runs in the background and wakes Claude when it exits with code 2. Stderr (or stdout if stderr is empty) reaches Claude as a system reminder." The hooks page summary identifies this as the only documented way for a background hook to wake a running session ([Hooks reference](https://code.claude.com/docs/en/hooks)).
- Plugin hooks are registered when a session loads the plugin and fire on their events from then on, not only when the plugin's skill runs ([Plugin components](https://code.claude.com/docs/en/plugins/components)).

### Inferences
- An `mcp_tool` hook lets a hook call `gdoc mcp` tools without a Bash round trip and without the model spending a turn.
- `asyncRewake` could implement a single "wake me when a new comment arrives" step: a hook runs `gdoc comments --wait` and exits 2 with the new threads on stderr. It wakes Claude once per exit, so it needs re-arming after each event.

### Gaps
- Measured latency of a hook invocation (process spawn cost per event) is not published; only timeouts are.
- Which events honour `asyncRewake` (all events or only some) was not confirmed in the part of the page read (the page is 282 KB; only the first 100 KB was read).

## 3. Ways to push something into a running Claude Code session

### Takeaway
There are six documented push routes, and their status differs. Monitor tool, background Bash and backgrounded MCP calls are general availability and push a notification when output or completion arrives. Plugin monitors are experimental. Channels are a research preview with an allowlist. Cross-session messaging also offers a local inbox socket that a session's own child process can post to. Each route has its own time limit, opt-in and permission rules.

### Cited Findings
- **Monitor tool**: "Runs a command in the background and feeds each output line back to Claude, so it can react to log entries, file changes, or polled status mid-conversation. Can also open a WebSocket and treat each incoming message as an event." "You keep working in the same session and Claude interjects when an event arrives." Every watch has a deadline: "5 minutes by default, at most 30 minutes, and at most 10 minutes" in a `-p` run; at the deadline Claude gets one notice and can restart it ([Tools reference](https://code.claude.com/docs/en/tools-reference)).
- Monitor WebSocket source: each text message is one event; binary frames become a placeholder; messages over 1 MiB end the watch. Claude Code "denies URLs that point at a private, link-local, or cloud-metadata address, including hostnames that resolve to one" ([Tools reference](https://code.claude.com/docs/en/tools-reference)).
- **Plugin monitors**: start automatically at session start or on first skill use and run for the whole session; not in `-p` ([Plugin components](https://code.claude.com/docs/en/plugins/components)).
- **Background Bash**: `run_in_background: true` starts a background task; Claude continues working. Interactive local sessions (terminal, desktop app, VS Code) have no time limit on background commands; unattended sessions (`-p`, Agent SDK, CI, cloud) get 30 minutes by default and at most 2 hours (requires v2.1.285+) ([Tools reference](https://code.claude.com/docs/en/tools-reference)).
- **Backgrounded MCP calls**: "An MCP tool call in the main conversation that is still running after two minutes moves to a background task instead of blocking the session. Claude receives the task ID immediately and keeps working, and the result arrives as a task notification when the call settles." Threshold set by `CLAUDE_CODE_MCP_AUTO_BACKGROUND_MS`; not applied to subagent calls or non-interactive runs by default; does not survive exiting the session ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- MCP idle timeout: a call with no response and no progress notification aborts after 30 minutes on stdio (5 minutes for HTTP), set by `CLAUDE_CODE_MCP_TOOL_IDLE_TIMEOUT`; `MCP_TOOL_TIMEOUT` defaults to about 28 hours ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- **Channels** (research preview): "A channel is an MCP server that pushes events into your running Claude Code session." The server declares `capabilities.experimental['claude/channel']: {}`, connects over stdio, and emits `notifications/claude/channel` with `content` (string) and optional `meta` (string map; keys letters, digits, underscore). The model sees `<channel source="..." key="...">content</channel>`. "Claude Code doesn't acknowledge notifications"; events from a server not loaded as a channel are dropped silently. "If several notifications arrive while Claude is busy, they're delivered together on the next turn" ([Channels reference](https://code.claude.com/docs/en/channels-reference)).
- Channel opt-in: a server must be named in `--channels` at startup; being in `.mcp.json` is not enough. During the preview `--channels` accepts only an Anthropic allowlist (the plugins in `claude-plugins-official`) or an org's `allowedChannelPlugins`; custom channels need `--dangerously-load-development-channels plugin:<name>@<marketplace>` or `server:<name>`, which shows a confirmation dialog and is ignored in `-p` and the Agent SDK. claude.ai Team and Enterprise orgs have channels blocked until an Owner sets `channelsEnabled`. Neither flag appears in `claude --help` ([Channels](https://code.claude.com/docs/en/channels); [Channels reference](https://code.claude.com/docs/en/channels-reference)).
- Channels require claude.ai or Console authentication and are not available on Bedrock, Google Cloud Agent Platform or Foundry ([Channels](https://code.claude.com/docs/en/channels)).
- Channels can be two-way (expose a normal MCP `reply` tool) and can relay permission prompts with `claude/channel/permission` capability, `notifications/claude/channel/permission_request` out and `notifications/claude/channel/permission` (`request_id`, `behavior: allow|deny`) back (v2.1.234+ sends these only to registered channels) ([Channels reference](https://code.claude.com/docs/en/channels-reference)).
- Channels and the new MCP revision: "On the v2 runtime, a channel server that negotiates MCP protocol revision 2026-07-28 can't deliver channel messages, so Claude Code doesn't register it as a channel." Workaround: `MCP_PROTOCOL_NEGOTIATION=legacy` ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- Channels launch date: March 20, 2026, Claude Code v2.1.80, per secondary sources ([The Decoder](https://the-decoder.com/anthropic-turns-claude-code-into-an-always-on-ai-agent-with-new-channels-feature/); [Towards AI](https://pub.towardsai.net/claude-code-channels-message-your-ai-coding-agent-from-telegram-and-discord-2026-5f263ccc4b9c)). Not confirmed from an Anthropic changelog in this research.
- **Cross-session messaging** (`SendMessage`, `ListAgents`): requires v2.1.224+ on macOS and Linux; on with nothing to enable. "The receiving Claude reads the message between tool calls during an active turn... When the receiving session is idle, Claude Code starts a new turn with the message." Plain text only, about one million characters cap, at most 50 queued accepted messages ([Cross-session messaging](https://code.claude.com/docs/en/cross-session-messaging)).
- **The session inbox socket**: each session binds a Unix socket, exported to hooks and Bash commands as `CLAUDE_CODE_MESSAGING_SOCKET` with a token in `CLAUDE_CODE_MESSAGING_TOKEN`. The docs name the case "when you want a script or hook to post into a session". When no `crossSessionInbound` setting applies, "Claude Code delivers a message it verifies came from the session's own child processes, such as a hook or Bash command posting back to its own session's socket." A connection must send a complete line within 30 seconds ([Cross-session messaging](https://code.claude.com/docs/en/cross-session-messaging)).
- `notify_when_idle`: a session can ask another local session for one notice when it next goes idle (v2.1.236+), expiring after 12 hours ([Cross-session messaging](https://code.claude.com/docs/en/cross-session-messaging)).
- **Session-scoped scheduled tasks**: `CronCreate` "Schedules a recurring or one-shot prompt within the current session"; `ScheduleWakeup` reschedules a self-paced `/loop` between one minute and one hour out ([Tools reference](https://code.claude.com/docs/en/tools-reference)). The channels page positions these as the way "to poll on a timer instead of reacting to pushed events" ([Channels](https://code.claude.com/docs/en/channels)).
- Other surfaces compared on the channels page: cloud sessions, Claude in Slack (spawns a cloud session), standard MCP ("nothing is pushed to the session"), Remote Control (drive a local session from claude.ai or mobile) ([Channels](https://code.claude.com/docs/en/channels)).
- `PushNotification` tool sends a desktop notification and, with Remote Control, a phone push: this goes to the person, not into the session ([Tools reference](https://code.claude.com/docs/en/tools-reference)).

### Inferences
- For gdoc live review, the GA route with no flags is a Monitor (or plugin monitor) running `gdoc comments --wait` in a loop that prints one line per new thread. Its cost per event is one notification plus whatever Claude reads, not a poll turn. The tool-started Monitor's 30-minute ceiling means the skill must re-arm it; a plugin monitor avoids that.
- Monitor's WebSocket source may refuse `ws://127.0.0.1` because loopback may count as "private". The docs do not say, so a gdoc WebSocket daemon should not rely on it without a test.
- Channels are the cleanest push design (one long-lived MCP server, events tagged with `meta` such as `doc_id` and `thread_id`). But today they need either the dev flag with a startup dialog on every launch, or an Anthropic or org allowlist entry, and Altery's org setting (`channelsEnabled`) is unknown. gdoc's hand-written MCP (`go/internal/mcp/`) could emit `notifications/claude/channel` directly, since the contract is a JSON-RPC notification over stdio; the docs only mention the TypeScript SDK.
- The inbox socket is a documented, flag-free push route for a process the session started. A backgrounded `gdoc` started from Bash would inherit `CLAUDE_CODE_MESSAGING_SOCKET`. This would be an external program talking to a socket, which is new ground for gdoc's guard and "one JSON object to stdout" invariants.

### Gaps
- The wire format of a message posted to the inbox socket (beyond the optional `{"type":"auth","token":...}` first line) was not in the pages read.
- The ship dates of the Monitor tool, plugin monitors and `asyncRewake` were not found in a primary changelog.
- Whether Claude Desktop (chat, not Code) supports channels or any push into a chat: no source found.

## 4. MCP features beyond tools, and what Claude Code and Claude Desktop support

### Takeaway
MCP revision 2026-07-28 made the protocol stateless: no sessions, no `initialize`, server-initiated requests replaced by multi round-trip results, resource subscriptions replaced by `subscriptions/listen`, tasks moved to an extension, and roots, sampling and logging deprecated. Claude Code supports resources (@-mentions), prompts (as slash commands), elicitation (form and URL), roots, list_changed, tool search, and the tasks extension in public beta on the v2 runtime. MCP Apps (interactive UI) work on claude.ai and are documented by Anthropic for claude.ai, while Claude Code hides `ui://` resources. Claude Code docs do not describe sampling or resource subscription support.

### Cited Findings
- 2026-07-28 major changes: remove protocol sessions and `Mcp-Session-Id`; remove `initialize`, each request carries version and capabilities in `_meta`; add `server/discover`; replace HTTP GET and `resources/subscribe` with `subscriptions/listen` (one long-lived POST response stream, opt-in types `toolsListChanged`, `promptsListChanged`, `resourcesListChanged`, `resourceSubscriptions`); remove `ping`, `logging/setLevel`, `notifications/roots/list_changed`; move tasks to extension `io.modelcontextprotocol/tasks` (polling via `tasks/get`, `tasks/update` for input); Multi Round-Trip Requests (`resultType: "input_required"`) replace server-sent `roots/list`, `sampling/createMessage`, `elicitation/create`; remove SSE resumability ([MCP 2026-07-28 changelog](https://modelcontextprotocol.io/specification/2026-07-28/changelog)).
- 2026-07-28 minor: `extensions` field in capabilities; `ttlMs` and `cacheScope` on list and read results to reduce polling; deterministic `tools/list` order "to improve LLM prompt cache hit rates"; any JSON Schema 2020-12 in `inputSchema`/`outputSchema` and any JSON in `structuredContent` ([MCP 2026-07-28 changelog](https://modelcontextprotocol.io/specification/2026-07-28/changelog)).
- Deprecated in 2026-07-28: Roots, Sampling, Logging ("integrate directly with LLM provider APIs instead of Sampling"), HTTP+SSE transport, Dynamic Client Registration ([MCP 2026-07-28 changelog](https://modelcontextprotocol.io/specification/2026-07-28/changelog)).
- Claude Code client runtimes: v1 on MCP TypeScript SDK 1.x; v2 on SDK 2.0 adding revision 2026-07-28. v2 is used from v2.1.232 where flags are fetched, and by default from v2.1.274 otherwise. v2 asks each server whether it supports the new revision. Override with `MCP_SDK_GENERATION` ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- Transports in Claude Code: stdio, HTTP (recommended; `streamable-http` alias), SSE (deprecated), WebSocket (`"type": "ws"`) ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- Resources: `@server:protocol://resource/path` mentions, fetched as attachments; Claude Code adds list and read resource tools. MCP Apps UI resources (`ui://` or `text/html;profile=mcp-app`) "don't appear in the `@` suggestions or in the resource list tool's results" ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- Prompts: listed as `/servername:promptname (MCP)`; results injected into the conversation ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- Elicitation: Claude Code shows a dialog; declares `elicitation: {form: {}, url: {}}` on 2026-07-28 connections and bare `elicitation: {}` (form only) on others; an `Elicitation` hook can auto-respond ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- Roots: Claude Code answers `roots/list` with the launch directory plus added directories ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- list_changed: interactive sessions refresh tools, prompts and resources; `-p` and Agent SDK refresh only tools ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- Progress: per-server `timeout` is wall clock and "progress notifications from the server don't extend it"; progress does reset the idle timeout ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- MCP tasks extension: "public beta in the Claude Code CLI and require Claude Code v2.1.304 or later", only on 2026-07-28 connections, in sessions that fetch feature flags, from the main conversation. A server returns `resultType: "task"` with `taskId`, `status`, `statusMessage`, `pollIntervalMs` ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- Output limits: warning above 10,000 tokens, cap 25,000 tokens (`MAX_MCP_OUTPUT_TOKENS`); text over 50,000 characters is saved to a file unless the tool sets `_meta["anthropic/maxResultSizeChars"]` (ceiling 500,000) ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- Claude-specific tool `_meta`: `anthropic/requiresUserInteraction` forces a prompt on every call; `anthropic/alwaysLoad` exempts a tool from deferral ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- `claude mcp serve` runs Claude Code itself as a stdio MCP server exposing its tools ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- MCP Apps: launched in Claude on January 26, 2026, the first official MCP extension to reach production status ([The Register](https://www.theregister.com/2026/01/26/claude_mcp_apps_arrives); [Alpic](https://alpic.ai/blog/mcp-apps-goes-official-claude-chatgpt-support)). Tools link UI through `_meta.ui.resourceUri` with MIME `text/html;profile=mcp-app`, rendered in a sandboxed iframe ([WorkOS](https://workos.com/blog/2026-01-27-mcp-apps); [MCP Apps API](https://apps.extensions.modelcontextprotocol.io/api/)). Anthropic's connector overview lists claude.ai with "full remote MCP support & MCP Apps" and Claude Desktop with "full MCP support and local desktop extensions" ([Claude connectors overview](https://claude.com/docs/connectors/overview)).

### Inferences
- For "kept state" under 2026-07-28, the spec wants explicit server-minted handles passed as tool arguments. That matches gdoc's existing cursor and ledger design better than relying on MCP sessions.
- `subscriptions/listen` is a standard server-to-client change stream, but it carries list-changed and resource-updated notifications to the client, not a prompt to the model. The docs read do not show Claude Code turning a resource update into a model turn. Channels remain the only documented "event becomes a turn" MCP route.
- A long `wait_for_comments` MCP tool in Claude Code becomes a background task after 2 minutes and notifies on completion, which gives a push-like loop without channels. It needs progress notifications or a result within 30 minutes to avoid the stdio idle abort.
- Sampling is deprecated, so a design where gdoc asks the client's model to judge a comment through MCP should not be adopted.

### Gaps
- Claude Code's support for sampling, `resources/subscribe` or `subscriptions/listen`, and `outputSchema` validation was not found in the pages read (the MCP page is 151 KB and was searched, not read in full).
- Whether Claude Desktop renders MCP Apps from a local `.mcpb` server is not stated by Anthropic; the overview credits MCP Apps to claude.ai only.
- Which MCP features Claude Desktop's chat client supports (elicitation, prompts, resources, list_changed, tasks) is not documented in the sources found.

## 5. Claude Agent SDK: custom tools, hooks, sessions, and a tool that ships its own agent

### Takeaway
The Agent SDK (TypeScript and Python) runs the Claude Code agent loop in a program. It takes in-process MCP tools, permission callbacks and hooks in code. A tool can ship a small agent built on it, but that program then pays for its own model calls and is a separate session from the one the user is in.

### Cited Findings
- Custom tools: `tool()` (TypeScript, Zod schema) or `@tool` (Python), wrapped in `createSdkMcpServer` / `create_sdk_mcp_server`, which "runs in-process inside your application, not as a separate process", passed via `mcpServers` to `query()` ([Agent SDK custom tools](https://code.claude.com/docs/en/agent-sdk/custom-tools)).
- Tool search is on by default for SDK MCP tools; `alwaysLoad: true` keeps a schema in the initial prompt ([Agent SDK custom tools](https://code.claude.com/docs/en/agent-sdk/custom-tools)).
- `readOnlyHint: true` lets Claude call tools in parallel; `permission_policy: "always_ask"` (TypeScript, SDK v0.3.301+) routes calls to the `canUseTool` callback ([Agent SDK custom tools](https://code.claude.com/docs/en/agent-sdk/custom-tools)).
- `structuredContent`: when set, Claude receives the JSON plus image or resource blocks, and text blocks are not forwarded. The Python `@tool` decorator forwards only `content` and `is_error` ([Agent SDK custom tools](https://code.claude.com/docs/en/agent-sdk/custom-tools)).
- In SDK and `-p` sessions, background Bash commands have a time limit (30 minutes default, 2 hours max); plugin monitors and the dev channels flag do not run there ([Tools reference](https://code.claude.com/docs/en/tools-reference); [Plugin components](https://code.claude.com/docs/en/plugins/components); [Channels reference](https://code.claude.com/docs/en/channels-reference)).
- A `-p` session binds an inbox socket and can receive cross-session messages; to take them unattended set `crossSessionInbound: accept` in its `--settings` ([Cross-session messaging](https://code.claude.com/docs/en/cross-session-messaging)).

### Inferences
- gdoc is Go, and the SDK is TypeScript or Python. Shipping an SDK agent would add a second runtime and a model bill outside the user's session, and would conflict with gdoc's three-module rule.
- A plugin subagent (`agents/*.md`) gives "skills delegate to a gdoc agent" inside the user's own session at no extra runtime cost. Plugin agents cannot declare their own hooks or MCP servers, so those stay at plugin level ([Plugin components](https://code.claude.com/docs/en/plugins/components)).

### Gaps
- The SDK's sessions, resume and streaming-input pages were not read; billing (API key versus subscription) for SDK use was not confirmed.

## 6. Desktop extensions (.mcpb), connectors, and one local server for several clients

### Takeaway
Claude Desktop runs local servers as stdio subprocesses installed from `.mcpb` bundles. claude.ai reaches only remote HTTPS connectors through Anthropic's infrastructure. Claude Code takes stdio, HTTP or WebSocket, and can load a `.mcpb` from a plugin. No Anthropic document describes one long-lived local server shared by several clients. The spec's move to stateless HTTP makes a localhost HTTP daemon easier, but each Claude client today starts its own stdio process.

### Cited Findings
- MCPB: "a zip archive containing a local MCP server and a `manifest.json`, which Claude Desktop installs in a single click"; runs "on the user's machine over stdio"; "Works offline"; "No OAuth required"; Claude Desktop runs on macOS and Windows; Node.js is recommended because it ships with Claude Desktop ([Build a desktop extension with MCPB](https://claude.com/docs/connectors/building/mcpb)).
- `user_config` in the manifest makes Claude Desktop generate a settings UI ([Build a desktop extension with MCPB](https://claude.com/docs/connectors/building/mcpb)).
- "Desktop extension listings in the directory are deprecated, and the directory no longer accepts MCPB submissions. To distribute a local MCP server through the directory, include it in a plugin." Install is per user ([Build a desktop extension with MCPB](https://claude.com/docs/connectors/building/mcpb)).
- Remote connectors "run on your servers via HTTPS and are accessed through Anthropic's infrastructure" ([Build a desktop extension with MCPB](https://claude.com/docs/connectors/building/mcpb)).
- Claude Code loads claude.ai connectors automatically when signed in with a claude.ai subscription, and can import servers from Claude Desktop ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- A plugin's `mcpServers` can point at a `.mcpb` or `.dxt`, so one bundle can serve Claude Desktop and Claude Code ([Plugin manifest reference](https://code.claude.com/docs/en/plugins-reference)).
- 2026-07-28 removes `Mcp-Session-Id` and the handshake; list endpoints no longer vary per connection; cross-call state uses explicit handles ([MCP 2026-07-28 changelog](https://modelcontextprotocol.io/specification/2026-07-28/changelog)).
- Channels must be stdio and spawned by Claude Code as a subprocess ([Channels reference](https://code.claude.com/docs/en/channels-reference)).

### Inferences
- "One route shared by Claude Code, Claude Desktop and the app" can be one gdoc binary with one MCP tool set, started as a stdio subprocess by each client, with shared state kept on disk (as today) rather than in one process. A single daemon with thin stdio shims is possible, but no Anthropic doc describes it and it would add a second long-lived process.
- A localhost HTTP gdoc server could serve Claude Code (`--transport http`) but not claude.ai, since claude.ai connectors go through Anthropic's servers.

### Gaps
- Current MCPB `manifest_version` (a third-party page says 0.3; unverified) and whether Claude Desktop's chat surfaces elicitation, MCP Apps or prompts from a `.mcpb` server.

## 7. Tool search, deferred loading and programmatic tool calling

### Takeaway
Claude Code defers MCP tool definitions by default and loads them on demand, so the cost of exposing many gdoc tools is mostly names plus server instructions. Programmatic tool calling is an API feature that keeps intermediate tool results out of context; it is not documented as a Claude Code feature.

### Cited Findings
- "Tool search keeps MCP context usage low by deferring tool definitions until Claude needs them. Only tool names and server instructions load at session start." Enabled by default; `ENABLE_TOOL_SEARCH` values `true`, `auto` (load upfront under 10% of context), `auto:N`, `false` ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- With tool search, the server `instructions` field helps Claude know when to search for the server's tools; a deferred tool's description may be up to 16,384 characters ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- Per-tool `_meta["anthropic/alwaysLoad"]`, per-server `alwaysLoad`; `alwaysLoad: true` makes startup wait for that server, capped at 5 seconds ([Claude Code MCP](https://code.claude.com/docs/en/mcp)).
- API launch: Tool Search Tool, Programmatic Tool Calling and Tool Use Examples published November 24, 2025, beta header `advanced-tool-use-2025-11-20` ([Anthropic engineering: advanced tool use](https://www.anthropic.com/engineering/advanced-tool-use)).
- Tool search measured 85% fewer tokens (about 77K to 8.7K before work starts) and MCP eval accuracy from 79.5% to 88.1% on Opus 4.5 ([Anthropic engineering: advanced tool use](https://www.anthropic.com/engineering/advanced-tool-use)).
- Programmatic tool calling: Claude writes Python in code execution that calls tools (opt-in `allowed_callers`); average tokens fell 37% (43,588 to 27,297) on complex research tasks; useful for three or more dependent calls ([Anthropic engineering: advanced tool use](https://www.anthropic.com/engineering/advanced-tool-use)).
- Tool use examples (`input_examples`) raised complex parameter accuracy from 72% to 90% ([Anthropic engineering: advanced tool use](https://www.anthropic.com/engineering/advanced-tool-use)).
- 2026-07-28 asks servers to return `tools/list` in a deterministic order to improve prompt cache hits ([MCP 2026-07-28 changelog](https://modelcontextprotocol.io/specification/2026-07-28/changelog)).

### Inferences
- Exposing gdoc's commands as MCP tools in Claude Code costs little context until used, so the token argument against "many tools" is weak there. The per-call costs that remain are the tool result size (25K token cap, 50K character file spill) and the round trip.
- The Bash route costs a Bash tool call per command plus the JSON object; the MCP route saves shell parsing and permission prompts per command pattern, but adds a ToolSearch call the first time a deferred tool is used.

### Gaps
- No source states whether programmatic tool calling is available inside Claude Code or Claude Desktop sessions.
- The shorter description limit for upfront-loaded tools in current versions was not captured (it was 2,048 characters before v2.1.296).
