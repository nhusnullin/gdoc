// Package tasks reads the notation a task list item carries: at most one owner
// and at most one due date at the end of its line.
//
// It is one package rather than a rule in two, because internal/body reads the
// notation out of a note on its way to the docx and internal/propose reads the
// same notation out of a block somebody wrote in chat or on the line. Two
// copies of one rule are two rules that drift, and the drift would be a chip on
// the wrong person.
//
// Nothing here reaches the network, and nothing here imports internal/settings.
// Parse answers the owner as it was written, and Owner.Resolve is handed the
// domain in effect by whoever has it. That keeps the whole notation testable
// from literals, and it keeps the setting's reading in the one package that
// owns the file.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # A token counts only at the end of the line, and anywhere else it is words
//
// Parse takes the last whitespace-separated word of the line, and the one
// before it, while each is an owner or a day. The first word that is neither
// ends the reading, and everything up to it is the item's words. So "Ask @anna
// about 2026-01-02 today" is a sentence about a person and a day, and "Fix the
// login @anna 2026-01-02" is a task for Anna by the second of January:
// TestTheTokensAreReadOnlyAtTheEnd and TestOwnerAndDateInEitherOrder.
//
// The two stand in either order, because nobody remembers an order. Each
// token's byte span in the source is answered beside it, so a caller can put a
// chip exactly where the token stood rather than searching the text for it
// again: TestOwnerAndDateInEitherOrder states both spans as literals.
//
// # What gdoc half understands is refused naming the word
//
// Two owners, two dates, a date of the right shape that is no day on a
// calendar, and a word opening with @ that is no address: each is refused
// naming what the line said. A second owner cannot be dropped, because which
// person the task is for is not gdoc's to decide, and 2026-02-30 cannot be
// rounded, because a chip on the wrong day is worse than a refusal a person
// can read and fix: TestTwoOwnersAreRefusedByName, TestTwoDatesAreRefusedByName,
// TestADateThatIsNotADayIsRefused and TestAnOwnerThatIsNotAnAddressIsRefused.
//
// A word of the date's shape is a date somebody meant. That is why 2026-02-30
// is a refusal while 2026-1-2 is words: the first is the notation written
// wrong, and the second is not the notation.
//
// # Parse takes the markdown source, not the rendered inlines
//
// GFM's linkify turns anna@example.org into an autolink, so an item's inlines
// hand the address over in pieces with a link around it. The source text is
// what a person typed, and it is the one reading in which an owner is one
// word.
//
// # A bare name takes the domain setting, and never a guess
//
// @anna is the common case, because a note is written among colleagues. Owner
// carries the name alone, and Resolve finishes it from the domain in effect.
// With no domain in effect it is refused naming the key and the flag, so a
// person is told the line to write before anything is sent anywhere. An
// address somebody wrote out in full is answered as written, whatever the
// setting says: TestABareNameTakesTheDomain,
// TestABareNameWithNoDomainIsRefusedNamingTheSetting and
// TestAFullAddressIgnoresTheDomain.
//
// # CheckDomain is the one rule about a domain
//
// Letters, digits, hyphens and dots, with at least one dot, no empty label and
// no label opening or closing with a hyphen. internal/settings asks it of the
// file and the flag, and readOwner asks it of a domain somebody wrote out, so
// @a@b is refused for the reason domain: b would be: TestADomainIsLettersDigitsHyphensAndDots
// here, and TestAMalformedDomainIsRefusedByName in internal/settings.
//
// # Nothing here judges a task
//
// No field says whether a task matters, whether it is overdue or whether an
// owner is the person signed in. The notation is a fact about what somebody
// wrote, and what to do about it belongs to a skill.
package tasks
