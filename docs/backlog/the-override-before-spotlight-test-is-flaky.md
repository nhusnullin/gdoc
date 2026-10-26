---
worth: yes
where: presence/Tests/PresenceAppTests/SessionTests.swift:521
added: 2026-10-24
---
# The override-before-Spotlight test is flaky

`testTheOverrideIsAskedBeforeSpotlight` failed once on PR 94, which changed
nothing under `presence/`, and passed on the rerun. The failure:

```
XCTAssertEqual failed: ("["1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789"]") is not
equal to ("[]") - and Spotlight was never asked about it
```

So on a slow runner Spotlight was asked about the document before the `hubPath`
override answered. Either the app has a real race, where the two routes start
together and the override only wins when it is fast, or the test waits for the
pairing and not for the order. Which one it is decides the fix: a race is an
app bug against the M27 rule that the override is asked first, and a test that
waits wrong is a test fix.
