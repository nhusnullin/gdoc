// The command table: every command gdoc answers, described once.
//
// Nothing else in this package lists a command or a flag. The dispatcher walks
// this table, the usage line is joined from it, and the parser each command is
// handed is built from the flags its entry names. A thirteenth command cannot
// answer a caller without being here, and a flag cannot be taken without being
// written down beside the sentence that says what it is for.

package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"gdoc/internal/emit"
)

// kind is what a flag's value is, and it has two readers. Help prints the
// placeholder for it, and the parser learns from it whether the flag carries a
// value at all. One field, so the two cannot drift apart.
type kind int

const (
	kindNone kind = iota
	kindFile
	kindFolderID
	kindCursor
	kindSeconds
)

// placeholder is the value as a reader sees it, or the empty string for a flag
// that takes none.
func (k kind) placeholder() string {
	switch k {
	case kindFile:
		return "<file>"
	case kindFolderID:
		return "<folder id>"
	case kindCursor:
		return "<cursor>"
	case kindSeconds:
		return "<seconds>"
	}
	return ""
}

// flag is one flag of one command: what a caller types, what it carries, and
// one sentence saying what it is for.
type flag struct {
	name    string
	value   kind
	summary string
}

// command is one entry in the table. words are the positional arguments in the
// order they are typed, named the way a reader sees them, and their count is
// what the parser insists on.
type command struct {
	name     string
	words    []string
	anyWords bool
	flags    []flag
	summary  string
	example  string
	run      func(ctx context.Context, a *args, errOut io.Writer) emit.Result
}

// anyCount is the word count of a command that takes however many words it is
// handed, and whose own function says what it will take. Two commands carry
// it. help takes another command's name, which is two words, or one, or none
// at all. completion takes one shell, and counts the words itself so that a
// missing one is refused naming a shell rather than naming a document, which
// is what the parser's refusal for one missing word says.
const anyCount = -1

// wants is the number of words the parser insists on, in both directions.
func (c command) wants() int {
	if c.anyWords {
		return anyCount
	}
	return len(c.words)
}

// nameWords is the command as typed, split. "auth status" is two words.
func (c command) nameWords() []string { return strings.Fields(c.name) }

// flagSet is what the parser is handed: the entry's flags, name against
// whether it carries a value. Derived rather than written twice, so a flag this
// command takes is a flag the help prints.
func (c command) flagSet() flagSet {
	spec := make(flagSet, len(c.flags))
	for _, f := range c.flags {
		spec[f.name] = f.value != kindNone
	}
	return spec
}

// commands is the whole of gdoc. The order is the order a reader meets them:
// the credential, then reading, then writing, then the two that make a document
// out of a note, and last the one that describes the rest.
//
// It is a function and not a variable because help is an entry in the table and
// reads the table, and a variable that refers to itself through a function is an
// initialization cycle Go refuses. Each caller is handed its own slice, so
// nothing can keep a pointer into a table somebody else is reading.
func commands() []command {
	return []command{
		{
			name:    "auth status",
			summary: "Say whether gdoc has a token, when it expires, and what it may reach.",
			example: "gdoc auth status",
			run: func(_ context.Context, _ *args, _ io.Writer) emit.Result {
				return authStatus()
			},
		},
		{
			name:    "auth login",
			summary: "Print the Google sign-in URL and wait for the browser to come back.",
			example: "gdoc auth login",
			run: func(_ context.Context, _ *args, errOut io.Writer) emit.Result {
				return authLogin(errOut)
			},
		},
		{
			name:  "read",
			words: []string{"<url>"},
			flags: []flag{
				{"--structure", kindNone, "print the tree of tabs, headings and tables instead of the text"},
			},
			summary: "Print the document as text, with the ids a comment or a suggestion is named by.",
			example: "gdoc read https://docs.google.com/document/d/1AbC.../edit",
			run: func(_ context.Context, a *args, _ io.Writer) emit.Result {
				return cmdRead(a)
			},
		},
		{
			name:  "comments",
			words: []string{"<url>"},
			flags: []flag{
				{"--since", kindCursor, "only what is newer than this cursor, which an earlier run printed"},
				{"--witness", kindNone, "export the document and say which threads the export still shows"},
				{"--wait", kindSeconds, "wait this long for something new before answering"},
			},
			summary: "List the comment threads with their ranges, their replies and the next cursor.",
			example: "gdoc comments https://docs.google.com/document/d/1AbC.../edit --wait 60",
			run: func(ctx context.Context, a *args, _ io.Writer) emit.Result {
				return cmdComments(ctx, a)
			},
		},
		{
			name:  "suggestions",
			words: []string{"<url>"},
			flags: []flag{
				{"--md", kindFile, "the note paired with this document, to say which of gdoc's own proposals have gone"},
			},
			summary: "List the suggestions pending in the document.",
			example: "gdoc suggestions https://docs.google.com/document/d/1AbC.../edit --md note.md",
			run: func(_ context.Context, a *args, _ io.Writer) emit.Result {
				return cmdSuggestions(a)
			},
		},
		{
			name:  "restyle",
			words: []string{"<url>"},
			flags: []flag{
				{"--dry-run", kindNone, "survey the document and write nothing into it"},
				{"--from", kindFile, "the survey a dry run wrote, which says the document has not moved since"},
				{"--fields", kindFile, "the cover values, which propose the house template as well as the styling"},
			},
			summary: "Survey a document against the house style, or apply that style where the document stands.",
			example: "gdoc restyle https://docs.google.com/document/d/1AbC.../edit --dry-run",
			run: func(_ context.Context, a *args, _ io.Writer) emit.Result {
				return cmdRestyle(a)
			},
		},
		{
			name: "probe",
			flags: []flag{
				{"--folder", kindFolderID, "the Drive folder the throwaway document is created in"},
			},
			summary: "Create a throwaway document and say whether Docs honours a suggestion today.",
			example: "gdoc probe --folder 1AbC...",
			run: func(_ context.Context, a *args, _ io.Writer) emit.Result {
				return cmdProbe(a)
			},
		},
		{
			name:  "reply",
			words: []string{"<url>", "<comment id>"},
			flags: []flag{
				{"--body-file", kindFile, "the file holding the words of the reply"},
			},
			summary: "Write one reply into a comment thread, under the robot prefix.",
			example: "gdoc reply https://docs.google.com/document/d/1AbC.../edit AAAA1234 --body-file reply.txt",
			run: func(_ context.Context, a *args, _ io.Writer) emit.Result {
				return cmdReply(a)
			},
		},
		{
			name:  "propose",
			words: []string{"<url>"},
			flags: []flag{
				{"--from", kindFile, "the file holding the changes to propose"},
				{"--folder", kindFolderID, "the Drive folder the working copy is created in"},
				{"--md", kindFile, "the note to record the proposals in, so they can be taken back later"},
			},
			summary: "Propose changes as native suggestions, each with the comment that says why.",
			example: "gdoc propose https://docs.google.com/document/d/1AbC.../edit --from changes.json --folder 1AbC... --md note.md",
			run: func(_ context.Context, a *args, _ io.Writer) emit.Result {
				return cmdPropose(a)
			},
		},
		{
			name:  "withdraw",
			words: []string{"<url>", "<suggestion id>"},
			flags: []flag{
				{"--md", kindFile, "the note that records which suggestions gdoc wrote"},
			},
			summary: "Take back one pending suggestion, and only one gdoc proposed itself.",
			example: "gdoc withdraw https://docs.google.com/document/d/1AbC.../edit suggest.abc123 --md note.md",
			run: func(_ context.Context, a *args, _ io.Writer) emit.Result {
				return cmdWithdraw(a)
			},
		},
		{
			name: "build",
			flags: []flag{
				{"--md", kindFile, "the note to build"},
				{"--out", kindFile, "the docx file to write"},
				{"--house", kindFile, "a house style file other than the one inside gdoc"},
				{"--force", kindNone, "replace the out file if something is already there"},
			},
			summary: "Build the note as a house-style docx on this machine, with no network at all.",
			example: "gdoc build --md note.md --out note.docx",
			run: func(_ context.Context, a *args, _ io.Writer) emit.Result {
				return cmdBuild(a)
			},
		},
		{
			name: "publish",
			flags: []flag{
				{"--md", kindFile, "the note to publish"},
				{"--folder-id", kindFolderID, "the Drive folder the document is created in"},
				{"--house", kindFile, "a house style file other than the one inside gdoc"},
			},
			summary: "Build the note as a house-style document and upload it into one folder.",
			example: "gdoc publish --md note.md --folder-id 1AbC...",
			run: func(_ context.Context, a *args, _ io.Writer) emit.Result {
				return cmdPublish(a)
			},
		},
		{
			name:     "help",
			words:    []string{"<command>"},
			anyWords: true,
			summary:  "Print every command gdoc takes, or the words and flags of one.",
			example:  "gdoc help publish",
			run: func(_ context.Context, a *args, errOut io.Writer) emit.Result {
				return cmdHelp(a.positional, errOut)
			},
		},
		{
			name:     "completion",
			words:    []string{"<shell>"},
			anyWords: true,
			flags: []flag{
				{"--out", kindFile, "the file to write the completion script to"},
				{"--force", kindNone, "replace the out file if something is already there"},
			},
			summary: "Write the completion script for one shell, and say the line that turns it on.",
			example: "gdoc completion zsh --out ~/.gdoc-completion.zsh",
			run: func(_ context.Context, a *args, _ io.Writer) emit.Result {
				return cmdCompletion(a)
			},
		},
	}
}

// usageLine names every command that exists, in the table's order. It is what
// a refusal prints, so a caller who typed a word gdoc does not know reads the
// words it does know.
func usageLine() string {
	table := commands()
	names := make([]string, 0, len(table))
	for _, c := range table {
		names = append(names, c.name)
	}
	return "Commands: " + strings.Join(names, ", ")
}

// unknownCommand is the one refusal for a word gdoc does not answer to,
// wherever the word came from: typed as a command, or handed to help.
func unknownCommand(args []string) emit.Result {
	return emit.Result{OK: false,
		Error: fmt.Sprintf("unknown command %q. %s", strings.Join(args, " "), usageLine())}
}

// match finds the entry whose name the arguments open with, longest first, so
// "auth status" is found where a bare "auth" would match nothing. It returns
// nil when no entry matches, which is the unknown command.
func match(args []string) *command {
	table := commands()
	var found *command
	var longest int
	for i := range table {
		words := table[i].nameWords()
		if len(words) > len(args) || len(words) <= longest {
			continue
		}
		if !equalWords(args[:len(words)], words) {
			continue
		}
		found, longest = &table[i], len(words)
	}
	return found
}

func equalWords(got, want []string) bool {
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
