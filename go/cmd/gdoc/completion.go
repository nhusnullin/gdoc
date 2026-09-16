// gdoc completion: the command table as a shell script, written to a file.
//
// The script is a rendering of the same table help prints, so a Tab offers
// what the parser takes and cannot drift from it. It goes into a file and
// never onto stdout, because stdout carries one JSON object and this command
// is not the exception: see the doc.go section that says so.
//
// Two shells today, zsh and bash. They group the table the same way, because
// both complete the way a person types, and they differ only in the language
// each says it in.

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

//go:embed completion_bash.tmpl
var bashTemplate string

// completionMode is the mode a new script is written with. An existing file
// keeps its own, which is atomicfile.ModeOf's rule.
const completionMode = 0o644

// shell is one shell gdoc writes a completion for: the word a caller types,
// the rendering of the table for it, and the key the report says the line to
// add under, because .zshrc and .bashrc are two different files.
type shell struct {
	name   string
	rcKey  string
	render func(table []command) (string, error)
}

// shells names every shell gdoc writes a completion for, in the order a
// refusal lists them.
func shells() []shell {
	return []shell{
		{"zsh", "add_to_zshrc", zshScript},
		{"bash", "add_to_bashrc", bashScript},
	}
}

// shellNames is the shells as a refusal reads them out.
func shellNames() string {
	names := make([]string, 0, len(shells()))
	for _, s := range shells() {
		names = append(names, s.name)
	}
	return strings.Join(names, ", ")
}

// completionReport is what the command prints: the shell it wrote for, the
// file it wrote, and the one line a person adds to their shell's own file.
// gdoc never edits that file itself. The key that line comes under names the
// file it goes in, so a bash user is never handed a line about .zshrc.
func completionReport(sh shell, path string) map[string]string {
	return map[string]string{
		"shell":  sh.name,
		"wrote":  path,
		sh.rcKey: "source " + path,
	}
}

func cmdCompletion(a *args) emit.Result {
	sh, err := oneShell(a.positional)
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
	script, err := sh.render(commands())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	if err := atomicfile.Replace(out, []byte(script), atomicfile.ModeOf(out, completionMode)); err != nil {
		return emit.Result{OK: false, Error: fmt.Sprintf("the completion script could not be written: %v", err)}
	}
	path := absolute(out)
	return emit.Result{OK: true, Data: completionReport(sh, path)}
}

// oneShell reads the one word this command takes. The count is checked here
// and not by the parser, because the parser's refusal for a missing word names
// a document, and what is missing here is a shell.
func oneShell(words []string) (shell, error) {
	known := shellNames()
	if len(words) == 0 {
		return shell{}, fmt.Errorf("this command needs a shell to write for. It writes: %s", known)
	}
	if len(words) > 1 {
		return shell{}, fmt.Errorf("%q is an extra argument: this command takes one shell", words[1])
	}
	for _, s := range shells() {
		if words[0] == s.name {
			return s, nil
		}
	}
	return shell{}, fmt.Errorf("%q is not a shell gdoc writes a completion for. It writes: %s", words[0], known)
}

// wordGroup is the table gathered by first word, which is the shape both
// renderings need: a person types one word, and then either a second word or
// a flag. entries keeps the table's order.
type wordGroup struct {
	word    string
	entries []command
}

// whole is the one command this group is, or nil when the group is a first
// word with commands under it.
func (g wordGroup) whole() *command {
	if len(g.entries) == 1 && len(g.entries[0].nameWords()) == 1 {
		return &g.entries[0]
	}
	return nil
}

// seconds is the second word of every command under this group, in order.
func (g wordGroup) seconds() []string {
	words := make([]string, 0, len(g.entries))
	for _, c := range g.entries {
		words = append(words, c.nameWords()[1])
	}
	return words
}

// byFirstWord gathers the table by first word, keeping the table's order. A
// first word that carries more than one command, or one command whose name is
// two words, becomes a group with a second level under it.
func byFirstWord(table []command) []wordGroup {
	order := make([]string, 0, len(table))
	byFirst := map[string][]command{}
	for _, c := range table {
		first := c.nameWords()[0]
		if _, seen := byFirst[first]; !seen {
			order = append(order, first)
		}
		byFirst[first] = append(byFirst[first], c)
	}
	groups := make([]wordGroup, 0, len(order))
	for _, first := range order {
		groups = append(groups, wordGroup{word: first, entries: byFirst[first]})
	}
	return groups
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

// zshGroups is the table's grouping said in zsh.
func zshGroups(table []command) []zshGroup {
	groups := make([]zshGroup, 0, len(table))
	for _, g := range byFirstWord(table) {
		if c := g.whole(); c != nil {
			groups = append(groups, zshGroup{
				Word:  g.word,
				Entry: zshEntry(g.word, c.summary),
				Specs: zshSpecs(*c),
			})
			continue
		}
		subs := make([]zshGroup, 0, len(g.entries))
		for i, c := range g.entries {
			subs = append(subs, zshGroup{
				Word:  g.seconds()[i],
				Entry: zshEntry(g.seconds()[i], c.summary),
				Specs: zshSpecs(c),
			})
		}
		// A first word is not a command, so it has no sentence of its own in
		// the table. What it offers is what a reader needs to see beside it.
		// No colon in that line: _describe cuts an entry on the first one.
		groups = append(groups, zshGroup{
			Word:  g.word,
			Entry: zshEntry(g.word, "one of "+strings.Join(g.seconds(), ", ")),
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

// bashData is what the bash template reads. bash has no per-flag description,
// so a flag is a word in a list and what follows it is a case over the word
// before the cursor: the file flags in one arm, the flags carrying a value
// nothing here knows in the other.
type bashData struct {
	Usage       string
	FileFlags   string
	OpaqueFlags string
	Words       string
	Groups      []bashGroup
}

// bashGroup is one word a caller types. Words is what compgen -W is handed
// under it: the second words for a first word that carries commands, and the
// flag names for a whole command. The words a command takes are a document
// URL, a comment id or a suggestion id, so none of them are offered.
type bashGroup struct {
	Word  string
	Words string
	Subs  []bashGroup
}

// bashScript renders the table as the bash completion script.
func bashScript(table []command) (string, error) {
	t, err := template.New("bash").Parse(bashTemplate)
	if err != nil {
		return "", fmt.Errorf("the bash completion template could not be read: %w", err)
	}
	files, opaque := bashFlagArms(table)
	data := bashData{
		Usage:       usageLine(),
		FileFlags:   files,
		OpaqueFlags: opaque,
		Words:       strings.Join(bashFirstWords(table), " "),
		Groups:      bashGroups(table),
	}
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", fmt.Errorf("the bash completion could not be rendered: %w", err)
	}
	return b.String(), nil
}

// bashGroups is the table's grouping said in bash.
func bashGroups(table []command) []bashGroup {
	groups := make([]bashGroup, 0, len(table))
	for _, g := range byFirstWord(table) {
		if c := g.whole(); c != nil {
			groups = append(groups, bashGroup{Word: g.word, Words: bashFlagWords(*c)})
			continue
		}
		subs := make([]bashGroup, 0, len(g.entries))
		for i, c := range g.entries {
			subs = append(subs, bashGroup{Word: g.seconds()[i], Words: bashFlagWords(c)})
		}
		groups = append(groups, bashGroup{
			Word:  g.word,
			Words: strings.Join(g.seconds(), " "),
			Subs:  subs,
		})
	}
	return groups
}

// bashFirstWords is every first word a caller can type, in the table's order.
func bashFirstWords(table []command) []string {
	groups := byFirstWord(table)
	words := make([]string, 0, len(groups))
	for _, g := range groups {
		words = append(words, g.word)
	}
	return words
}

// bashFlagWords is the flags of one command, as compgen -W reads them.
func bashFlagWords(c command) string {
	names := make([]string, 0, len(c.flags))
	for _, f := range c.flags {
		names = append(names, f.name)
	}
	return strings.Join(names, " ")
}

// bashFlagArms is the two arms of the case over the word before the cursor,
// each a bash pattern: the flags that carry a file, which offer file names,
// and the flags that carry a folder id, a cursor or a wait length, which offer
// nothing because nothing on this machine knows one. A flag that carries no
// value is in neither arm, because nothing follows it.
func bashFlagArms(table []command) (files, opaque string) {
	var withFile, withOpaque []string
	seen := map[string]bool{}
	for _, c := range table {
		for _, f := range c.flags {
			if f.value == kindNone || seen[f.name] {
				continue
			}
			seen[f.name] = true
			if f.value == kindFile {
				withFile = append(withFile, f.name)
				continue
			}
			withOpaque = append(withOpaque, f.name)
		}
	}
	return strings.Join(withFile, "|"), strings.Join(withOpaque, "|")
}
