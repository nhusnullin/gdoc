---
worth: later
where: go/cmd/gdoc/main.go:359
added: 2026-10-18
---
# A code run started from a subagent names the subagent's session

`claudeSessionEnv` is `CLAUDE_CODE_SESSION_ID`, read unconditionally, and that
is the session of the process that started gdoc. Inside a Claude Code subagent
that is the child's own session, which nobody can sit in. Measured 2026-10-18
in a subagent's environment on Nail's machine:

| Variable | Value |
|---|---|
| `CLAUDE_CODE_SESSION_ID` | `e477e5e3-0c9f-487f-a704-aa5e2b6dec24`, 36 runes |
| `CLAUDE_CODE_HOST_SESSION_ID` | `local_67637bff-b596-4f9b-a6d9-a4c307a59215`, 42 runes |
| `CLAUDE_CODE_CHILD_SESSION` | `1` |

So a run from a subagent writes `session` on its start line, and the id it
writes is not the session a person could go back to. The field's one stated
purpose is that a person can, `go/internal/presence/doc.go` and
`PresenceCore/Documents.swift`.

**The obvious fix is not obviously right.** `CLAUDE_CODE_HOST_SESSION_ID` is a
different shape, `local_` and then a uuid, and what `claude://code/continue`
accepts was measured with a bare uuid, MEASURED.md, 2026-10-18. Taking the host
variable would put a differently shaped id in the record for a link that was
measured to switch nothing today, so it trades one unusable value for another
of unknown use. The third answer is to write no session at all where
`CLAUDE_CODE_CHILD_SESSION` is set, which is the honest line: the run came from
a session a person cannot be handed.

What costs nothing today: nothing opens the link, so no press is wrong. What
costs something: the record is filling with child ids now, and the panel keeps
them.

What would settle it, and all three belong in MEASURED.md with the
`claude://code/continue` rows, which today record only the URL forms: whether
`claude://code/continue?session=local_<uuid>` is a form Claude Desktop knows at
all, and whether a host session id resumes anything a hand can see. Then Nail
decides which variable the record names, which is a DECISIONS.md entry and not
a refactor.

One small thing to fix whichever way it goes: `claudeSessionEnv`'s comment says
"the measurement is in docs/v2/MEASURED.md", and MEASURED.md has no entry for
the variable. The branch measured the link and not the environment.
