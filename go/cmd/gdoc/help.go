// gdoc help: the command table as one JSON object, and as words a person reads.
//
// Both renderings walk the same entries, so what a skill parses and what a
// person reads cannot disagree, and neither can drift from what the parser
// takes. The object goes to stdout like every other command's, and the words go
// to stderr where the login URL already goes.

package main

import (
	"fmt"
	"io"
	"strings"

	"gdoc/internal/emit"
)

// helpReport is what a caller reads. One shape for both forms, so a skill
// written against `gdoc help publish` reads `gdoc help` without a second path.
type helpReport struct {
	Commands []helpCommand `json:"commands"`
}

// helpCommand is one entry of the table as JSON. Words and Flags are always
// there, empty rather than absent, because a skill telling "takes none" from
// "said nothing" is the whole reason to print the help at all.
type helpCommand struct {
	Name    string     `json:"name"`
	Words   []string   `json:"words"`
	Flags   []helpFlag `json:"flags"`
	Summary string     `json:"summary"`
	Example string     `json:"example"`
}

// helpFlag carries the placeholder as help prints it rather than the kind it
// came from: a reader cares what to type, not what the table calls it. A flag
// that takes no value carries the empty string.
type helpFlag struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Summary string `json:"summary"`
}

// cmdHelp answers with every command, or with the ones whose name opens with
// the words it was given. Words that match nothing are refused the way an
// unknown command is, because that is what they are.
func cmdHelp(words []string, errOut io.Writer) emit.Result {
	matched := helpMatches(words)
	if len(matched) == 0 {
		return unknownCommand(words)
	}
	fmt.Fprint(errOut, helpProse(matched, len(words) == 0))
	return emit.Result{OK: true, Data: helpReport{Commands: helpEntries(matched)}}
}

// helpMatches is every command whose name opens with these words, so `help
// auth` is both auth commands and `help publish` is one. No words at all is
// every command, which is what bare `gdoc help` asks for.
func helpMatches(words []string) []command {
	table := commands()
	if len(words) == 0 {
		return table
	}
	var matched []command
	for _, c := range table {
		name := c.nameWords()
		if len(words) <= len(name) && equalWords(name[:len(words)], words) {
			matched = append(matched, c)
		}
	}
	return matched
}

// helpEntries is the table's own values copied into the shape that goes out, so
// nothing a caller is handed points back into the table.
func helpEntries(matched []command) []helpCommand {
	entries := make([]helpCommand, 0, len(matched))
	for _, c := range matched {
		flags := make([]helpFlag, 0, len(c.flags))
		for _, f := range c.flags {
			flags = append(flags, helpFlag{
				Name:    f.name,
				Value:   f.value.placeholder(),
				Summary: f.summary,
			})
		}
		entries = append(entries, helpCommand{
			Name:    c.name,
			Words:   append([]string{}, c.words...),
			Flags:   flags,
			Summary: c.summary,
			Example: c.example,
		})
	}
	return entries
}

// helpProse is the same entries as words. The list is what a person who typed
// the wrong thing needs, and the detail is what a person who knows the command
// but not its flags needs.
func helpProse(matched []command, all bool) string {
	if all {
		return helpList(matched)
	}
	var b strings.Builder
	for i, c := range matched {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(helpOne(c))
	}
	return b.String()
}

// helpList is every command, one line each, name and sentence aligned.
func helpList(matched []command) string {
	var b strings.Builder
	b.WriteString("Usage: gdoc <command> [words] [flags]\n\n")
	width := 0
	for _, c := range matched {
		if len(c.name) > width {
			width = len(c.name)
		}
	}
	for _, c := range matched {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, c.name, c.summary)
	}
	b.WriteString("\nRun gdoc help <command> for the words and flags one takes.\n")
	return b.String()
}

// helpOne is one command in full: the line a person types, the sentence saying
// what it does, a line per flag, and one example to copy.
func helpOne(c command) string {
	var b strings.Builder
	b.WriteString("Usage: gdoc " + c.name)
	for _, w := range c.words {
		b.WriteString(" " + w)
	}
	for _, f := range c.flags {
		b.WriteString(" " + flagWords(f))
	}
	fmt.Fprintf(&b, "\n  %s\n", c.summary)
	if len(c.flags) > 0 {
		b.WriteString("\n")
		width := 0
		for _, f := range c.flags {
			if n := len(flagWords(f)); n > width {
				width = n
			}
		}
		for _, f := range c.flags {
			fmt.Fprintf(&b, "  %-*s  %s\n", width, flagWords(f), f.summary)
		}
	}
	fmt.Fprintf(&b, "\n  Example: %s\n", c.example)
	return b.String()
}

// flagWords is the flag as it is typed: the name, and the placeholder after it
// when it carries a value.
func flagWords(f flag) string {
	if p := f.value.placeholder(); p != "" {
		return f.name + " " + p
	}
	return f.name
}

// helpAsked reports whether --help or -h stands anywhere on the line, and gives
// back what is left once they are taken off.
func helpAsked(args []string) ([]string, bool) {
	rest := make([]string, 0, len(args))
	asked := false
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			asked = true
			continue
		}
		rest = append(rest, arg)
	}
	return rest, asked
}

// helpWords turns what is left into the words help answers over. A command
// found in the rest is the question, whatever else was typed beside it, so
// `gdoc restyle --from x --help` asks about restyle rather than about three
// words that name no command. What matches nothing is passed through, so it is
// refused by name.
func helpWords(rest []string) []string {
	if c := match(rest); c != nil {
		return c.nameWords()
	}
	return rest
}
