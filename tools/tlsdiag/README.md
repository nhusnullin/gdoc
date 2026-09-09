# tlsdiag

Is this network interfering with TLS?

One command, no dependencies, no configuration, nothing written anywhere. It
opens some connections, times them, and tells you what it found.

```bash
cd tools/tlsdiag && go run . -v
```

Or build it once and hand the binary to somebody:

```bash
cd tools/tlsdiag && go build -o tlsdiag .
```

## Why it exists

On 2026-09-09 every Go program on one machine, `gdoc` included, timed out after
ten seconds on any HTTPS call. The error was:

```
Post "https://oauth2.googleapis.com/token": net/http: TLS handshake timeout
```

That reads like an expired credential or a broken tool. It was neither. curl
worked. openssl worked. The browser worked. A mobile hotspot worked. The office
wifi did not.

An hour went into finding that, most of it on wrong theories. What would have
found it in twenty seconds is the comparison this tool makes: **TLS 1.2 against
TLS 1.3, to more than one host, with the timings side by side.**

## What it checks

For each host, three things separately, because separating them is what makes
the answer readable:

1. **TCP connect.** If this fails, nothing below it means anything.
2. **A handshake capped at TLS 1.2.**
3. **A handshake required to be TLS 1.3.**

A TCP connection that succeeds while a handshake stalls says the packets arrive
and something dislikes what is in them.

It checks three hosts on three independent operators by default, because one
host failing is that host's problem and three failing together is the path. Pass
your own if you like:

```bash
go run . example.com:443 internal.corp:8443
```

Then, only when Go's TLS 1.3 failed, it runs `openssl` and `curl` against the
same host at the same moment. That part matters more than it looks. "TLS 1.3 is
blocked" gets checked with a browser, the browser works, and the ticket closes.
The accurate claim is narrower and stranger: **Go's handshake stalls where
openssl's succeeds.** Nobody believes that without seeing both, so the tool
shows both rather than asserting it.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Nothing found. TLS 1.3 completed to every reachable host. |
| 1 | TLS 1.3 is being interfered with, or the pattern is mixed and needs reading. |
| 2 | Nothing was reachable. Not a TLS question: check the connection, DNS and any VPN. |

## For a network administrator

If this reports TLS 1.3 failing while TLS 1.2 succeeds, something between the
machine and the internet is inspecting or filtering TLS handshakes. TLS 1.3
encrypts the certificate exchange, so appliances that want to see certificates
sometimes drop it rather than fail it, which is why it stalls for ten seconds
instead of returning an error.

Two things worth knowing before dismissing it:

- **The machine may still browse the web perfectly.** Browsers and curl can be
  treated differently from other clients, so "the internet works here" does not
  rule this out.
- **The failure is silent and slow**, so it looks like the application's fault.
  Every developer on that network will lose an hour to it separately.

## What it does not do

No writes, no uploads, no credentials, no configuration files read. It reads the
clock, the sockets it opens, and with `-v` the default route, any proxy
environment variables and the list of tunnel interfaces. It runs `openssl` and
`curl` only when a Go handshake has already failed, and only against a host that
was already being tested.

It is standard library only and about three hundred lines, so it can be read
before it is run. That is the point of a tool you hand to somebody else.
