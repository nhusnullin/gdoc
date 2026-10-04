---
worth: yes
where: release/install.sh:521
added: 2026-10-03
---
# The installer leaves the sign-in to the colleague

The one-line installer ends by printing `gdoc auth status` under the line
"next: gdoc auth login, if this says signed out". The colleague has to read a
JSON answer, decide whether it means signed out, and type the next command. On
the clean install of 2026-10-03 the summary said "all good" and then handed this
step back. It must not be painful for colleagues.

Nail's design, 2026-10-03: the installer checks, and when the check fails it
signs in. It runs `gdoc auth status`. When there is no token, it runs `gdoc auth
login`, which prints the sign-in link. The installer opens that link in the
default browser and also prints it, with a line saying to paste it into the
right browser profile when the person has several, because the default profile
may be signed in to another Google account. `auth login` waits for the browser
to come back. The installer then runs `auth status` again to check, and the
summary ends with "signed in as <account>", or with the one command to retry if
the sign-in did not finish.

What stands in the way today, and why it can move:

- The installer's own header says "the token is not read, written or removed
  here. Signing in is `gdoc auth login`." That is a choice in `install.sh`, not
  an invariant. Running `gdoc auth login` leaves the token to the binary, as it
  is today.
- `auth login` prints the link to stderr and keeps stdout to one JSON object,
  so the installer reads the link from stderr, prints it, and runs `open` on it.
- The binary never opens a browser: nothing under `go/` runs an external
  program but one `open` in `desktop.go` (CLAUDE.md invariants). The installer
  is a shell script outside `go/`, and it already opens `gdoc.mcpb` the same way.
- The installer must still work where nobody is at the keyboard (CI, a pipe
  with no terminal). Sign in only when stdout is a terminal, and print the
  command otherwise.

A Claude Desktop colleague does not need the terminal sign-in at all: chat's
`login` tool gives the link when the first tool says nobody is signed in
(measured 2026-10-03). The installer can say that instead of signing in when it
was run with `--desktop` only.
