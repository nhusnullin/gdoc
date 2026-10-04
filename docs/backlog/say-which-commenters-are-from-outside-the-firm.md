---
worth: maybe
where: go/internal/comments/comments.go, go/internal/chat/facts.go
added: 2026-10-03
---
# Say which commenters are from outside the firm

The M14 specification's scenario 16 opens a chat review with a one-line risk
summary that names a comment "from outside Altery". It was to rest on
`author_domain`, the domain of the comment author's email address.

Drive does not fill that address. Measured 2026-10-03 (MEASURED.md, "gdoc mcp in
Claude Desktop, the first run"): `author.emailAddress` came back empty on every
comment, the signed-in person's own included. So `author_domain` is left out of
the answer, and nothing can tell an outside commenter from an inside one today.

What could work, each to be measured first: the Drive `permissions` list of the
file, which names the email addresses the document is shared with, matched by
display name; or the People API for a commenter's profile. Both are new reads
through the guard, so each is a decision, not a refactor. Until then the
summary names links, hidden characters and comments that address the AI, which
are facts gdoc does have.
