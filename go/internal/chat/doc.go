// Package chat is what a chat adds to a gdoc command, and nothing the terminal
// needs.
//
// A terminal has a person reading every line of it. A chat has a model between
// the person and the document, and a comment in that document is text a
// stranger wrote which the model reads. So the checks here are the ones that
// only make sense where that is true: the code that proves the rules arrived,
// the labels and the facts a read answer carries, the record of what this
// process read and wrote, and the rules that hold a risky write until the
// person approves it themselves.
//
// It knows no command, no document and no wire. It imports no net/http, runs
// nothing and writes nothing to disk. cmd/gdoc is what joins it to a command.
// docs/v2/DECISIONS.md holds the 2026-10-03 decisions, and
// docs/plans/2026-10-02-gdoc-v2-m14-chat.md the specification.
//
// # The guide code
//
// Every tool but guide and login takes a code argument, and guide is the only
// place that code is given out. It is sixteen hex characters from crypto/rand,
// made once when the process starts: TestACodeIsSixteenHexCharacters and
// TestTheCodeAProcessGaveIsTaken.
//
// It is reliability and not safety. What it proves is that the rules entered
// the model's context before the first comment was read, which matters because
// Claude Desktop does not show a server's instructions to the person and may
// not keep them in front of the model (docs/v2/MEASURED.md, measurement 4).
// What stands between a comment and a write is never the code: it is the hold
// rules and the person's own approval.
//
// Nothing writes a code down, so a code from an earlier process, or from the
// other process Claude Desktop runs, is refused like any other wrong one:
// TestACodeFromAnotherProcessIsStale.
//
// A missing or wrong code gets RetrySentence and nothing else, so a model
// reads it, calls guide and makes the same call again without asking the person
// about a code they never typed: TestAMissingCodeIsRefusedWithTheRetrySentence,
// and TestEveryToolButGuideAndLoginRefusesAMissingOrStaleCode in cmd/gdoc for
// the same sentence through every tool.
//
// # What is not here yet
//
// The labels and the facts on a read answer, the ledger, the hold rules, the
// confirm tools, the write memory and the trusted domains are tasks 11 to 18 of
// docs/plans/2026-10-03-gdoc-v2-m14b-server.md.
package chat
