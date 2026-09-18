---
worth: later
added: 2026-09-18
---
# The three skills for a colleague running OpenAI Codex

Nail's note on 2026-09-18: support OpenAI Codex. The binary is agent-neutral,
one JSON object per command and no prompt, so the gap is the skills alone.
`skills/gdoc-review`, `gdoc-publish` and `gdoc-restyle` reach a colleague
through the Claude Code plugin in `.claude-plugin/`, or through
`release/install.sh --skills`, which copies them into `~/.claude/skills` or
`./.claude/skills`. Neither route puts anything where Codex looks.

What settles the value decision, and why it is `later`: whether any colleague
runs Codex against these documents, and what Codex reads a skill from at the
time the work starts, since that has moved. The likely shape is a third
`--skills` target in `release/install.sh` and a Codex-facing copy of the
three SKILL.md files, or an AGENTS.md that points at them. The wording of the
skills assumes Claude Code in places, the live review loop in `gdoc-review`
most of all, and that is where the real cost sits rather than in the install
line.
