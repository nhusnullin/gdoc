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
login`, which prints the sign-in link; the person clicks it, and `auth login`
waits for the browser to come back. The installer then runs `auth status` again
to check, and the summary ends with "signed in as <account>", or with the one
command to retry if the sign-in did not finish. The installer does not open the
browser itself: the person clicks the link.

What stands in the way today, and why it can move:

- The installer's own header says "the token is not read, written or removed
  here. Signing in is `gdoc auth login`." That is a choice in `install.sh`, not
  an invariant. Running `gdoc auth login` leaves the token to the binary, as it
  is today.
- `auth login` prints the link to stderr and keeps stdout to one JSON object,
  so the link reaches the terminal as it does when a person types the command.
- The installer must still work where nobody is at the keyboard (CI, a pipe
  with no terminal). Sign in only when stdout is a terminal, and print the
  command otherwise.

A Claude Desktop colleague does not need the terminal sign-in at all: chat's
`login` tool gives the link when the first tool says nobody is signed in
(measured 2026-10-03). The installer can say that instead of signing in when it
was run with `--desktop` only.
