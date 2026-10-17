---
worth: later
where: presence/Sources/PresenceCore/HealthFacts.swift:143
added: 2026-10-17
---
# The expected domain has nowhere to be set

The M24 specification's account row says "the expected domain is an app
preference, empty by default, a hint row that blocks nothing". The rule is
built as a value and reaches no screen.

`DomainHint` parses the domain and writes the sentence, and
`testTheExpectedDomainIsAHintThatBlocksNothing` holds it. What is missing is
every step after it:

- `Session` builds `HealthPoller` with no `domain:`, so the hint is
  `DomainHint.unset` for the life of the app.
- `HealthPoller.applied(to:)` carries nineteen fields into `Facts` and drops
  `reading.domainHint`. `Facts` has no field for it, and the account row's
  quiet line is built from `previousAccount` alone.
- There is no seventh `Preferences.Key`, so nobody can set a domain at all.

The last one is why this is not a wiring fix. Row 89 of the specification
fixes the persisted state at six keys and says "Nothing else persists", so a
seventh key is a decision and not a refactor. Two ways out:

- A seventh key, written down as a decision, and `domainHint` carried through
  `Facts` into the account row's quiet line. The panel has no text field, so
  the domain would be set in `settings.yaml` or in the defaults by hand.
- Or cut `DomainHint` and `HealthReading.domainHint`, and drop the row from
  the specification.

Worth `later`: whose account it is decides nothing anywhere in gdoc, so a
hint that blocks nothing is the quietest row in HEALTH. "Account changed"
already says the thing a person would look for.
