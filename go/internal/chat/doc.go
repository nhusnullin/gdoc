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
// # The six hold rules, decided here and never by the model
//
// Rules runs over one Write and answers a Hold or nothing. The binary decides
// that a write is risky, by a fixed list, and the person releases it. That is a
// judgement in Go, which the invariant "the binary prints facts and the skills
// judge" otherwise forbids: decision 16 of the specification accepted it,
// because a model asked to judge whether a comment is steering it is the thing
// being steered.
//
// The first rule that trips is the one named, in this order, because a card
// naming six reasons is a card nobody reads:
// TestTheFirstRuleThatTripsIsTheOneNamed.
//
//   - Link: the text carries a link or an email address this session has not
//     seen in the target document or its comments. A link in a write the
//     document never held came from somewhere, and the only somewheres are a
//     stranger's comment and another document:
//     TestTheLinkRuleTripsOnWhatIsNotAlreadyThere. An address at a domain the
//     person listed as trusted is exempt and a link never is:
//     TestATrustedDomainsAddressIsNotHeldAndALinkStillIs.
//   - Dictated: twelve words in a row shared with a comment gdoc did not write.
//     This is the attack the whole design is for, a stranger typing the reply
//     they want and the model posting it under the firm's name. Twelve is long
//     enough that an agreement of phrasing is not it:
//     TestTheDictatedRuleTripsOnTwelveWordsInARow.
//   - Focus: another document was read in the last thirty minutes, or this
//     session never read the target. A write into a document nobody looked at is
//     a write nobody can check:
//     TestTheFocusRuleTripsOnAnotherDocumentOrAnUnreadTarget. Copied text is
//     held and never refused, on Nail's call of 2026-10-03, because quoting the
//     firm's own policy can be the job, and the reason names the document and
//     the run of words: TestTextCopiedFromAnotherDocumentIsHeldNotRefused. A
//     write that went into the target resets the window, and a held write
//     records nothing, so the reset means a write the person released.
//   - Burst: the third write in sixty seconds, or the twenty-sixth into one
//     document in an hour. A review goes at the speed of a person reading:
//     TestTheBurstRuleTripsOnTheThirdWriteAndTheTwentySixth.
//   - Flagged thread: a reply under a comment carrying has_link, names_ai,
//     robot_not_ours or hidden_chars, each of which is a comment written at the
//     model rather than at a colleague:
//     TestTheFlaggedThreadRuleTripsOnAReplyIntoAFlaggedThread.
//   - Large removal: a propose taking out more than three hundred characters,
//     which is no longer a correction:
//     TestTheLargeRemovalRuleTripsOverThreeHundredCharacters.
//
// Text carrying a hidden character is refused outright rather than held, because
// there is nothing a person could sensibly approve: they cannot see what they
// would be approving. ErrHiddenChars is that refusal:
// TestHiddenCharactersAreRefusedOutright.
//
// Every rule asks about this session rather than about the call in front of it,
// which is why Rules takes the ledger, and the clock is handed in, so the one
// clock a session reads is cmd/gdoc's. A Hold carries everything the card is
// built from, the rule, the exact value, and the words themselves:
// TestAHoldCarriesTheWriteItIsFor. Whose account wrote a comment reaches none of
// it: the ledger carries no author, and rules.go never names the fact:
// TestAuthorDomainChangesNoOutcome.
//
// # Only the person releases a hold
//
// A hold is answered by a card, never by a yes said in the chat, because a yes
// said in the chat passes through the model. Release is what judges the one call
// that would send a held write, and it asks two things.
//
// The words are the hold's own. Card carries the hold id, the document's title,
// the reason and the text, and all four are compared byte for byte, so the
// person cannot be shown a paraphrase of why the write was stopped:
// TestTheCardsWordsArePinnedToTheByte, and
// TestAByteDifferentTitleReasonOrTextIsRefused in cmd/gdoc through the tool
// itself. The field that differs is named, because the words to send are in the
// held answer and which one is wrong is a fact about the call.
//
// And the turn ended. quietGap is five seconds of silence behind the call,
// measured from the session's last tool call or from the hold itself, whichever
// is later: TestTheQuietGapIsFiveSeconds and
// TestTheGapIsMeasuredFromTheLastCallOrTheHoldItself. The number is measurement
// 11 of docs/v2/MEASURED.md: the two people measured took 95 and 121 seconds
// over the card, so five seconds costs an honest release nothing, and the one
// shape it does cost is a model releasing its own write inside the turn that was
// held, which is what both runs of that measurement showed it doing.
//
// Both refusals keep the hold, because both are answerable: one by sending the
// card's own words, and the other by waiting. Nothing here registers a tool,
// keeps a hold or sends a write: cmd/gdoc's mcprelease.go is where a hold
// becomes one tool with one name, and where a released call goes out as the
// arguments the hold kept.
//
// # What is not here yet
//
// The write memory and the trusted-domains setting are tasks 17 and 18 of
// docs/plans/2026-10-03-gdoc-v2-m14b-server.md. Rules already takes the trusted
// list; nothing parses the person's setting into it yet.
package chat
