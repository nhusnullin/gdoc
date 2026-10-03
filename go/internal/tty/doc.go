// Package tty holds the facts about one output stream and the environment
// around it: whether a person is reading it, how much colour it takes, and how
// wide it is.
//
// It exists because of the 2026-10-03 decision that on a terminal a help
// screen is for the person. That decision rests on one question, asked once:
// is this stream a terminal. Before this package the tree asked it twice, in
// two different ways, and the second way was wrong: progress.go read the
// character-device bit, and /dev/null is a character device too.
// docs/v2/DECISIONS.md holds the decision and what it narrows.
//
// # It knows no command and no layout
//
// Nothing here knows what a help screen looks like, what gdoc's commands are,
// or where a box is drawn. internal/panel takes a Depth and a width from here
// and returns lines; cmd/gdoc decides which screen to draw. So the question
// "is this a terminal" has one answer in the tree, and the answer is not mixed
// up with what to do about it.
//
// # The driver answers, not the file's mode
//
// IsTerminal is true only for an *os.File the terminal driver answers for. The
// question is asked with the TIOCGETA ioctl on darwin, which is the isatty
// question, and asking it needs no module: syscall is the standard library.
//
// A buffer has no descriptor, so it is refused before the ioctl is reached:
// TestABufferIsNotATerminal. A pipe, which is what every skill hands gdoc, has
// one that no driver owns: TestAPipeIsNotATerminal. /dev/null has one too, and
// that is the case the old character-device check got wrong:
// TestDevNullIsNotATerminal. IsTerminal adds nothing of its own to the
// driver's answer, which is what lets every test above this package stub one
// variable: TestTheIoctlAnswerIsTheAnswer. TODO(test): Task 6 of the M15 plan
// moves the progress list onto this question and adds
// TestOnlyATerminalDriverMakesATerminal in cmd/gdoc.
//
// # Every platform but darwin answers no
//
// terminal_darwin.go sets the two ioctls; terminal_other.go sets both
// questions to false. So a run on linux prints today's plain text and the JSON
// object, which is the safe answer rather than the pretty one. Windows is out
// of scope, Nail's call of 2026-10-03. Neither file returns from IsTerminal or
// Width itself: each sets a variable those two always go through, so one stub
// in a test covers both platforms, and CI on ubuntu still exercises the rule.
//
// # The depth is read from the environment, never from the terminal
//
// Asking a terminal what it can do means writing a query and reading the reply
// on stdin, and gdoc leaves stdin alone: only `mcp` reads there. So Colour
// reads four rows of environment, in order.
//
// NO_COLOR set to anything but the empty string, or TERM=dumb, is NoColour,
// and it wins over everything under it:
// TestNoColourAndADumbTerminalTurnTheColourOff. The empty string is unset,
// which is no-color.org's own wording: TestAnEmptyNoColourIsNotSet. Then
// COLORTERM naming truecolor or 24bit: TestColortermNamesTrueColour. Then a
// TERM ending -256color: TestATermEndingTwoFiftySixColourIsTwoFiftySix. Then
// Sixteen, the floor, where a chip becomes reverse video:
// TestEverythingElseIsSixteen.
//
// NoColour is the zero value on purpose, so a Depth nobody set writes no
// escape byte.
//
// # The width
//
// The window itself first, through TIOCGWINSZ, so a person who resizes the
// terminal gets the next screen at the new width: TestTheIoctlWidthWins. A
// measurement of zero columns is no measurement, which a pseudo-terminal with
// no window size attached really does return. Then a positive COLUMNS, the
// only answer available where there is no ioctl:
// TestColumnsAnswersWhenTheIoctlCannot, with
// TestANonPositiveColumnsIsNoAnswer over a zero, a negative number and a word.
// Then eighty: TestEightyIsTheLastAnswer. A writer that is not a file is never
// measured, so a golden test sets its own width:
// TestABufferHasNoDescriptorSoColumnsAnswers.
//
// # What is not here
//
// No layout and no box. No read of stdin. No network. TODO(test): Task 5 of
// the M15 plan puts the palette and every escape code here, and Task 6 adds
// boundary's TestNoEscapeLiteralOutsideTTY, which holds that no escape byte is
// written anywhere else by reading the syntax tree of every non-test file
// under go/.
package tty
