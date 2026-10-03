// Package tty holds the facts about one output stream and the environment
// around it: whether a person is reading it, how much colour it takes, and how
// wide it is. It also holds the one palette and every escape code gdoc writes.
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
// variable: TestTheIoctlAnswerIsTheAnswer. cmd/gdoc asks the question through
// one variable of its own, so the step list `gdoc update` draws answers to this
// definition and to nothing else: TestOnlyATerminalDriverMakesATerminal.
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
// # One palette, by role and by depth
//
// Style writes colour around text. Its methods are roles and not colours:
// Title, Key, Border, Dim, OK, Fail, Warn and Chip. A caller says what a piece
// of text is for and this package says what that looks like at the depth
// Colour answered with, so internal/panel draws boxes without naming a
// colour.
//
// Text itself has no role. A command name, a description and a JSON line keep
// the terminal's own foreground, which the person already tuned for their
// background. That, and accents at middle brightness, are what let one palette
// read on a dark terminal and on a light one, which gdoc needs because it
// cannot ask a terminal what its background is. The values are palette 4A of
// docs/design/panels-round-two.html, stated as literals in
// TestTrueColourIsTheHexOfPaletteFourA, TestTwoFiftySixIsTheNearestIndex and
// TestSixteenIsTheBasicAnsiCode.
//
// At NoColour every role hands the text straight back, so a pipe, a file and
// every non-darwin run see no escape byte: TestNoColourWritesNoEscapeByte,
// with TestAnEmptyRoleIsStillPlainAtNoColour on the empty string a box draws
// for a missing label. The zero Style is that one, because NoColour is the
// zero Depth: TestAStyleNobodySetWritesNothing. Every coloured role closes
// with the reset, so nothing leaks onto the shell prompt under the screen:
// TestEveryRoleEndsWithTheReset.
//
// Chip is the role that is a background rather than a foreground: a version in
// a top border, a channel beside it. At sixteen colours no pair of colours
// reads on both backgrounds, so a chip asks the terminal to swap its own two:
// TestAChipIsReverseVideoOnSixteen.
//
// # Every escape code is here
//
// Up, ClearBelow, ClearLine, WrapOff and WrapOn are the cursor and line codes,
// the same at every depth because they are not colour: a list that redraws
// itself needs them even where NO_COLOR is set.
// TestTheLineCodesAreTheirBytes pins each as the bytes it is, and
// TestUpIsNothingForNoRows holds the first draw, which has nothing above it to
// move over and must therefore write no code at all.
//
// VisibleWidth is how wide a drawn line is: every rune outside an escape
// sequence counts as one column, so a coloured line and a plain one measure
// the same, and the box-drawing and braille a panel uses count as one each:
// TestVisibleWidthSkipsEscapes and
// TestVisibleWidthCountsBoxDrawingAndBrailleAsOne.
//
// # What is not here
//
// No layout and no box. No read of stdin. No network. And no second room
// writing an escape byte: boundary's TestNoEscapeLiteralOutsideTTY reads the
// syntax tree of every non-test file under go/ and fails on a literal holding
// one outside this package.
package tty
