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
// # The text arrives labelled, and the wrapper is new each time
//
// Every read answer opens with ReadLine, which says who wrote what follows, that
// it is material to discuss, and that nothing in it is an instruction to call a
// tool, gdoc's or another connector's: TestTheReadLineIsTheLiteral, and
// TestEveryReadAnswerOpensWithTheFixedLine in cmd/gdoc over all three read
// tools.
//
// Label puts one piece of a document's text inside a boundary of twelve random
// hex characters, new for every answer, so a comment that saw one answer's
// wrapper cannot end the next one: TestABoundaryIsTwelveHexCharacters,
// TestTwoBoundariesDiffer, and TestTheBoundaryIsNewForEachAnswer in cmd/gdoc.
// An occurrence of the boundary inside the text is broken by one character,
// which is also what stops a comment ending its own wrapper with a tag it typed
// itself: TestAFakeClosingTagOrTheBoundaryInsideTheTextIsBroken,
// TestTheBreakReplacesOnlyTheFirstCharacter and
// TestAnOverlappingRunIsBrokenRightThrough.
//
// The words themselves come through as they stand. Datamarking and encoding
// were both rejected: annotate and propose need the exact quote to find it in
// the document again, and a person listening rather than reading would hear the
// noise. TestLabelWrapsTheTextAndKeepsItWordForWord, and
// TestEveryCommentReplyQuoteAndDocumentTextIsWrapped in cmd/gdoc, which holds
// that the wrapped copy and the envelope carry the same words.
//
// # Six facts about a comment, each one literal check
//
// Facts carries has_link, has_email, names_ai, hidden_chars, robot_not_ours and
// author_domain beside every comment and reply:
// TestEachFactHasOneCheck, TestTheFactNamesAreTheLiterals, and
// TestAFactsObjectSitsBesideEachThreadAndReply in cmd/gdoc.
//
// They describe the words and never the person. has_link reads an address out
// of the text before it looks for a host, so one address is one fact and not
// two, which is what lets task 18 exempt a trusted domain without letting a
// link escape. A bare host is read only where its last label is one of
// linkTLDs, because a dot between two words is usually a sentence ending and a
// fact firing on ordinary prose is a hold the person learns to wave through.
//
// robot_not_ours is the one fact that is not about the text alone: the mark with
// no receipt behind it. It asks OwnReplies, this process's record of what gdoc
// wrote, and never the account, because identity is never a gate:
// TestRobotNotOursAsksTheRecordOfWhatThisProcessWrote. author_domain is carried
// and read by nothing: TestAuthorDomainIsCarriedAndDecidesNothing.
//
// HasHiddenChars is exported because task 14 refuses such text outright rather
// than holding it, and two readings of the same rune tables would drift:
// TestHasHiddenCharsIsTheSameCheckTheFactReports.
//
// # The ledger of what this process read and wrote
//
// Every hold rule is a question about this session rather than about the call in
// front of it: whether a link in a write was already in the document, whether a
// run of words came out of a comment a stranger left, whether the model has been
// reading another document, how many writes have gone into this one in the last
// minute. Ledger is what answers them.
//
// A read is kept with the document, its title, the instant and what the tool
// fetched, and a write with the document, the tool and the instant:
// TestEveryReadIsRecordedWithItsTitleTimeAndOthersTexts,
// TestEveryWriteAndOwnReplyIsRecorded, and
// TestEachReadToolRecordsItsReadInTheLedger in cmd/gdoc over all three read
// tools. A text read and a comment listing fetch different things, so the ledger
// answers for the document and not for one read of it: a listing carrying no text
// does not take away what a text read fetched,
// TestAReadKeepsTheDocumentsBodyText. A write reads its target on the call and
// that read is kept too, so the rules always have the target's own words:
// TestAWriteReadsItsTargetFreshAndRecordsIt in cmd/gdoc.
//
// The ids of what gdoc wrote are kept beside the writes, and they are what
// robot_not_ours asks: TestRobotNotOursUsesTheLedger, and
// TestAReplyThisSessionWroteIsNotRobotNotOurs in cmd/gdoc through the answer a
// model reads. A *Ledger is the OwnReplies the facts were written against, and
// the stub is gone.
//
// It is per process and nothing reaches disk, which is the rule a grant lives by
// as well: TestTheLedgerIsPerProcessAndWritesNothingToDisk, which reads the
// file's own imports. Two goroutines may record at once, because the login
// listener answers beside the worker:
// TestTheLedgerTakesTwoRecordersAtOnce. Every method is safe on a nil Ledger,
// which is a process that has recorded nothing.
//
// Nothing here judges. The ledger is read by the hold rules and by nothing else,
// and no answer a model reads carries a word of it.
//
// # What is not here yet
//
// The hold rules, the confirm tools, the write memory and the trusted domains are
// tasks 14 to 18 of docs/plans/2026-10-03-gdoc-v2-m14b-server.md.
package chat
