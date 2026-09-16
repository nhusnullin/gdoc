// gdoc completion: the command table as a shell script, written to a file.
//
// The script is a rendering of the same table help prints, so a Tab offers
// what the parser takes and cannot drift from it. It goes into a file and
// never onto stdout, because stdout carries one JSON object and this command
// is not the exception: see the doc.go section that says so.

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"gdoc/internal/atomicfile"
	"gdoc/internal/emit"
)

//go:embed completion_zsh.tmpl
var zshTemplate string

// completionMode is the mode a new script is written with. An existing file
// keeps its own, which is atomicfile.ModeOf's rule.
const completionMode = 0o644

// completionReport is what the command prints: the shell it wrote for, the
// file it wrote, and the one line a person adds to their shell's own file.
// gdoc never edits that file itself.
type completionReport struct {
	Shell      string `json:"shell"`
	Wrote      string `json:"wrote"`
	AddToZshrc string `json:"add_to_zshrc"`
}

// shells names every shell gdoc writes a completion for, in the order a
// refusal lists them. zsh is the one today.
func shells() []string { return []string{"zsh"} }

func cmdCompletion(a *args) emit.Result {
	shell, err := oneShell(a.positional)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	out, err := required(a, "--out")
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	// Before anything is rendered. The file already there is somebody's, and
	// not knowing must never resolve to overwrite. The check is build's.
	if err := freeToWrite(out, a.has("--force")); err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	script, err := zshScript(commands())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	if err := atomicfile.Replace(out, []byte(script), atomicfile.ModeOf(out, completionMode)); err != nil {
		return emit.Result{OK: false, Error: fmt.Sprintf("the completion script could not be written: %v", err)}
	}
	path := absolute(out)
	return emit.Result{OK: true, Data: completionReport{
		Shell:      shell,
		Wrote:      path,
		AddToZshrc: "source " + path,
	}}
}

// oneShell reads the one word this command takes. The count is checked here
// and not by the parser, because the parser's refusal for a missing word names
// a document, and what is missing here is a shell.
func oneShell(words []string) (string, error) {
	known := strings.Join(shells(), ", ")
	if len(words) == 0 {
		return "", fmt.Errorf("this command needs a shell to write for. It writes: %s", known)
	}
	if len(words) > 1 {
		return "", fmt.Errorf("%q is an extra argument: this command takes one shell", words[1])
	}
	for _, s := range shells() {
		if words[0] == s {
			return s, nil
		}
	}
	return "", fmt.Errorf("%q is not a shell gdoc writes a completion for. It writes: %s", words[0], known)
}

// zshData is what the template reads: the usage line as a comment, and the
// commands grouped by their first word, because that is how a person types
// them and how zsh completes them.
type zshData struct {
	Usage  string
	Groups []zshGroup
}

// zshGroup is one word a caller types. A group with Subs is a first word that
// carries commands under it, auth today. A group without is a whole command,
// and Specs is what _arguments is given for it.
type zshGroup struct {
	Word  string
	Entry string
	Subs  []zshGroup
	Specs []string
}

// zshScript renders the table as the zsh completion script.
func zshScript(table []command) (string, error) {
	t, err := template.New("zsh").Parse(zshTemplate)
	if err != nil {
		return "", fmt.Errorf("the zsh completion template could not be read: %w", err)
	}
	var b bytes.Buffer
	if err := t.Execute(&b, zshData{Usage: usageLine(), Groups: zshGroups(table)}); err != nil {
		return "", fmt.Errorf("the zsh completion could not be rendered: %w", err)
	}
	return b.String(), nil
}

// zshGroups gathers the table by first word, keeping the table's order. A
// first word that carries more than one command, or one command whose name is
// two words, becomes a group with a second level under it.
func zshGroups(table []command) []zshGroup {
	order := make([]string, 0, len(table))
	byFirst := map[string][]command{}
	for _, c := range table {
		first := c.nameWords()[0]
		if _, seen := byFirst[first]; !seen {
			order = append(order, first)
		}
		byFirst[first] = append(byFirst[first], c)
	}

	groups := make([]zshGroup, 0, len(order))
	for _, first := range order {
		entries := byFirst[first]
		if len(entries) == 1 && len(entries[0].nameWords()) == 1 {
			c := entries[0]
			groups = append(groups, zshGroup{
				Word:  first,
				Entry: zshEntry(first, c.summary),
				Specs: zshSpecs(c),
			})
			continue
		}
		subs := make([]zshGroup, 0, len(entries))
		seconds := make([]string, 0, len(entries))
		for _, c := range entries {
			second := c.nameWords()[1]
			subs = append(subs, zshGroup{
				Word:  second,
				Entry: zshEntry(second, c.summary),
				Specs: zshSpecs(c),
			})
			seconds = append(seconds, second)
		}
		// A first word is not a command, so it has no sentence of its own in
		// the table. What it offers is what a reader needs to see beside it.
		// No colon in that line: _describe cuts an entry on the first one.
		groups = append(groups, zshGroup{
			Word:  first,
			Entry: zshEntry(first, "one of "+strings.Join(seconds, ", ")),
			Subs:  subs,
		})
	}
	return groups
}

// zshSpecs is what _arguments is handed for one command: a line per flag, and
// last a line for the words it takes. That last line offers nothing, because
// the words are a document URL, a comment id or a suggestion id, and nothing
// on this machine knows one. Without it zsh would offer file names for them.
func zshSpecs(c command) []string {
	specs := make([]string, 0, len(c.flags)+1)
	for _, f := range c.flags {
		specs = append(specs, zshFlagSpec(f))
	}
	return append(specs, `'*: :'`)
}

// zshFlagSpec is one flag as _arguments reads it: the name, the sentence the
// table carries, and for a flag with a value the placeholder and what to offer
// for it. Only a file is offered, because a folder id, a cursor and a wait
// length live nowhere on this machine.
func zshFlagSpec(f flag) string {
	spec := "'" + f.name + "[" + zshText(f.summary) + "]"
	if f.value == kindNone {
		return spec + "'"
	}
	value := strings.Trim(f.value.placeholder(), "<>")
	return spec + ":" + zshText(value) + ":" + zshAction(f.value) + "'"
}

// zshAction is what zsh offers for a flag's value, and the empty string means
// it offers nothing.
func zshAction(k kind) string {
	if k == kindFile {
		return "_files"
	}
	return ""
}

// zshEntry is one line of a _describe array: the word, then what it is.
func zshEntry(word, summary string) string {
	return "'" + word + ":" + zshText(summary) + "'"
}

// zshText escapes what a spec is read for: the quote that would end the
// string, and the three characters _arguments cuts a spec on.
func zshText(s string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`'`, `'\''`,
		`[`, `\[`,
		`]`, `\]`,
		`:`, `\:`,
	).Replace(s)
}
