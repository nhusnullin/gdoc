package main

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"gdoc/internal/lastcheck"
)

// The script is the command table in another language, so the table is what
// the test asks. A command added to the table and not to the script is a Tab
// that stays silent, and a flag whose kind says <file> must offer file names
// where one whose kind says <folder id> offers none: nothing on this machine
// knows a Drive id, a cursor or a wait length.
func TestTheZshScriptNamesEveryCommandAndEveryFlag(t *testing.T) {
	table := commands()
	if len(table) == 0 {
		t.Fatal("the command table is empty, so this test is measuring nothing")
	}
	script, err := zshScript(table)
	if err != nil {
		t.Fatal(err)
	}

	// zsh reads the first line to learn what the file completes, and the last
	// line is what makes `source <path>` enough, with no fpath arrangement.
	if !strings.HasPrefix(script, "#compdef gdoc\n") {
		t.Errorf("the script must open with #compdef gdoc: %q", head(script))
	}
	if !strings.HasSuffix(script, "compdef _gdoc gdoc\n") {
		t.Errorf("the script must end by registering itself: %q", tail(script))
	}

	// The body, never the header. The header is one comment line carrying the
	// usage line, which names every command in the table, so a test that reads
	// the whole file would pass on the comment alone and say nothing about what
	// Tab offers.
	body := withoutComments(script)
	for _, c := range table {
		// Every word a caller types is a _describe entry, which zshEntry writes
		// as 'word:what it is. A two-word command is two of them, the first word
		// at the top level and the second under it.
		for _, word := range c.nameWords() {
			if !strings.Contains(body, "'"+word+":") {
				t.Errorf("the script must offer %q as an entry of %q", word, c.name)
			}
		}
		for _, f := range c.flags {
			if !strings.Contains(body, f.name+"[") {
				t.Errorf("%s takes %s, and the script offers it nowhere", c.name, f.name)
			}
		}
	}

	// One kind per flag name, so the lines below can be read one at a time.
	kinds := flagKinds(t, table)
	for _, line := range strings.Split(body, "\n") {
		spec := strings.TrimSpace(line)
		if !strings.HasPrefix(spec, "'--") {
			continue
		}
		name, _, ok := strings.Cut(strings.TrimPrefix(spec, "'"), "[")
		if !ok {
			t.Errorf("a flag line carries no description: %q", spec)
			continue
		}
		k, known := kinds[name]
		if !known {
			t.Errorf("the script offers %q, which no command takes: %q", name, spec)
			continue
		}
		if files := strings.Contains(spec, "_files"); files != (k == kindFile) {
			t.Errorf("%s carries %q, and the script %s file names for it: %q",
				name, k.placeholder(), map[bool]string{true: "offers", false: "offers no"}[files], spec)
		}
	}
}

// withoutComments is the script with its comment lines dropped, which is what
// every assertion about what Tab offers reads. Both scripts open with a comment
// carrying the usage line, and the usage line names every command in the table,
// so a command missing from the completion logic would still be found by a
// search over the whole file.
func withoutComments(script string) string {
	var kept []string
	for _, line := range strings.Split(script, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// flagArms is every arm of the bash script's case over the word before the
// cursor, each one whole: the line that opens it, and every line after it up to
// and including the ;; that closes it. An arm opens with a dash, which is what
// tells a flag arm from a command arm.
func flagArms(script string) []string {
	var arms []string
	var current []string
	for _, line := range strings.Split(script, "\n") {
		text := strings.TrimSpace(line)
		if current == nil && !strings.HasPrefix(text, "--") {
			continue
		}
		current = append(current, text)
		if strings.Contains(text, ";;") {
			arms = append(arms, strings.Join(current, " "))
			current = nil
		}
	}
	return arms
}

// hasArm reports whether the bash script answers for this word, which is a line
// opening an arm of a case: the word, then the closing parenthesis. A flag arm
// opens with a dash, so it is never mistaken for a command word.
func hasArm(script, word string) bool {
	for _, line := range strings.Split(script, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), word+")") {
			return true
		}
	}
	return false
}

// head and tail are the ends of the script, for a failure a person can read
// without the whole file in the way.
func head(script string) string {
	first, _, _ := strings.Cut(script, "\n")
	return first
}

func tail(script string) string {
	lines := strings.Split(strings.TrimRight(script, "\n"), "\n")
	return lines[len(lines)-1]
}

// The script is a file the command writes, never something on stdout: stdout
// carries the one object, here as everywhere. The object says where the file
// went and the line to add, because a script nobody sources completes nothing.
func TestCompletionWritesTheFileAndReportsTheLineToAdd(t *testing.T) {
	noSession(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "gdoc.zsh")

	got, code := runJSON(t, "completion", "zsh", "--out", out)
	if code != 0 || got["ok"] != true {
		t.Fatalf("completion zsh: %v (exit %d)", got, code)
	}
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	if data["shell"] != "zsh" {
		t.Errorf("shell: %v", data["shell"])
	}
	if data["wrote"] != out {
		t.Errorf("wrote: got %v, want %s", data["wrote"], out)
	}
	if want := "source " + out; data["add_to_zshrc"] != want {
		t.Errorf("add_to_zshrc: got %v, want %q", data["add_to_zshrc"], want)
	}

	written, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	script := string(written)
	if !strings.HasPrefix(script, "#compdef gdoc\n") || !strings.HasSuffix(script, "compdef _gdoc gdoc\n") {
		t.Errorf("the file is the script: %q ... %q", head(script), tail(script))
	}
	if !strings.Contains(script, "publish") {
		t.Errorf("the file must carry the commands: %q", script)
	}

	// The path is resolved, so the line to add works from any directory. A
	// person runs this from wherever they stand and pastes the line into a
	// file read from home.
	t.Chdir(dir)
	got, code = runJSON(t, "completion", "zsh", "--out", "relative.zsh")
	if code != 0 || got["ok"] != true {
		t.Fatalf("a relative --out: %v (exit %d)", got, code)
	}
	data, _ = got["data"].(map[string]any)
	wrote, _ := data["wrote"].(string)
	if !filepath.IsAbs(wrote) || !strings.HasSuffix(wrote, "relative.zsh") {
		t.Errorf("wrote must be the absolute path: %v", data["wrote"])
	}
}

// Not knowing never resolves to overwrite, and the file already there is
// somebody's. The rule is build --out's, and this command calls the same
// check rather than writing a second one.
func TestCompletionRefusesAnExistingOutUnlessForced(t *testing.T) {
	noSession(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "already.zsh")
	if err := os.WriteFile(out, []byte("not a completion"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, code := runJSON(t, "completion", "zsh", "--out", out)
	if code == 0 || got["ok"] != false {
		t.Fatalf("an existing file must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, out) || !strings.Contains(msg, "--force") {
		t.Errorf("the error must name the file and the way to say yes: %q", msg)
	}
	if kept, _ := os.ReadFile(out); string(kept) != "not a completion" {
		t.Fatalf("the refused run must leave the file alone: %q", kept)
	}

	got, code = runJSON(t, "completion", "zsh", "--out", out, "--force")
	if code != 0 || got["ok"] != true {
		t.Fatalf("--force replaces it: %v (exit %d)", got, code)
	}
	if replaced, _ := os.ReadFile(out); !strings.HasPrefix(string(replaced), "#compdef gdoc\n") {
		t.Errorf("the file was replaced by the script: %q", head(string(replaced)))
	}

	// A directory is refused whatever the flag says. --force is somebody
	// agreeing to replace a file, not to replace a folder.
	got, code = runJSON(t, "completion", "zsh", "--out", dir, "--force")
	if code == 0 || got["ok"] != false {
		t.Fatalf("a directory must be refused: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "directory") {
		t.Errorf("the refusal must say what is there: %q", msg)
	}
}

// completion is parsed as strictly as everything else: it takes one shell,
// one --out, and --force. Each refusal names what was wrong, and a refused
// run writes nothing.
func TestCompletionArgumentsAreStrict(t *testing.T) {
	noSession(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "gdoc.zsh")

	for _, tc := range []struct {
		args []string
		says string
	}{
		{[]string{"completion", "--out", out}, "shell"},
		{[]string{"completion", "fish", "--out", out}, "fish"},
		{[]string{"completion", "zsh"}, "--out"},
		{[]string{"completion", "zsh", "bash", "--out", out}, "bash"},
		{[]string{"completion", "zsh", "--out", out, "--md", "note.md"}, "--md"},
	} {
		got, code := runJSON(t, tc.args...)
		if code == 0 || got["ok"] != false {
			t.Errorf("%v must be refused: %v (exit %d)", tc.args, got, code)
			continue
		}
		msg, _ := got["error"].(string)
		if !strings.Contains(msg, tc.says) {
			t.Errorf("%v must be refused naming %q: %q", tc.args, tc.says, msg)
		}
		if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%v was refused and something was written to %s anyway", tc.args, out)
		}
	}
}

// The bash script is the same table in a third language, and the test asks
// the table for the same reason: a command added without a line here is a Tab
// that stays silent. What follows a flag is the same rule as zsh's, said the
// way bash says it: a file offers file names, and a Drive id, a cursor and a
// wait length offer nothing.
func TestTheBashScriptNamesEveryCommandAndEveryFlag(t *testing.T) {
	table := commands()
	if len(table) == 0 {
		t.Fatal("the command table is empty, so this test is measuring nothing")
	}
	script, err := bashScript(table)
	if err != nil {
		t.Fatal(err)
	}

	// bash has no #compdef line. What it needs is the last line, which is what
	// makes `source <path>` enough.
	if !strings.HasPrefix(script, "# bash completion for gdoc") {
		t.Errorf("the script must open by saying what it is: %q", head(script))
	}
	if !strings.HasSuffix(script, "complete -o filenames -F _gdoc gdoc\n") {
		t.Errorf("the script must end by registering itself: %q", tail(script))
	}

	// The body, never the header, for the reason the zsh test says: the header
	// carries the usage line, which names every command already.
	body := withoutComments(script)
	for _, c := range table {
		// Every word a caller types is an arm of a case. A two-word command is
		// two of them, the first word at the top level and the second under it.
		for _, word := range c.nameWords() {
			if !hasArm(body, word) {
				t.Errorf("the script must answer for %q, which is a word of %q", word, c.name)
			}
		}
		for _, f := range c.flags {
			if !strings.Contains(body, f.name) {
				t.Errorf("%s takes %s, and the script offers it nowhere", c.name, f.name)
			}
		}
	}

	// One kind per flag name, so the arms below can be read one at a time.
	kinds := flagKinds(t, table)

	// Every arm of the case over the word before the cursor. A flag with a
	// value is in one of them, and a flag without a value is in none, because
	// nothing follows it. An arm is read whole, up to and including the ;; that
	// closes it, because what it offers may stand on a line of its own.
	armed := map[string]bool{}
	for _, arm := range flagArms(body) {
		names, _, ok := strings.Cut(arm, ")")
		if !ok {
			t.Errorf("a flag arm is not an arm: %q", arm)
			continue
		}
		files := strings.Contains(arm, "compgen -f")
		for _, name := range strings.Split(names, "|") {
			k, known := kinds[name]
			if !known {
				t.Errorf("the script answers for %q, which no command takes: %q", name, arm)
				continue
			}
			if k == kindNone {
				t.Errorf("%s carries no value, so nothing follows it: %q", name, arm)
				continue
			}
			armed[name] = true
			if files != (k == kindFile) {
				t.Errorf("%s carries %q, and the script %s file names after it: %q",
					name, k.placeholder(), map[bool]string{true: "offers", false: "offers no"}[files], arm)
			}
		}
	}
	for name, k := range kinds {
		if k != kindNone && !armed[name] {
			t.Errorf("%s carries %q, and the script says nothing about what follows it", name, k.placeholder())
		}
	}
}

// The file arm's shape, not just that it calls compgen. An unquoted
// $(compgen -f) is split on IFS, so ~/My Notes/ comes back as "My" and
// "Notes/", two candidates that are each a path to nothing. The read loop is
// what keeps one path one candidate, and the one-liner it replaced contains
// compgen -f too, so the assertion above would pass on the bug. The loop runs
// in the calling shell, which is an interactive shell, so the variable it
// assigns has to be declared local or it leaks into the person's session.
func TestTheBashFileArmKeepsAPathWithASpaceWhole(t *testing.T) {
	script, err := bashScript(commands())
	if err != nil {
		t.Fatal(err)
	}
	body := withoutComments(script)

	var file string
	for _, arm := range flagArms(body) {
		if strings.Contains(arm, "compgen -f") {
			file = arm
		}
	}
	if file == "" {
		t.Fatal("no arm of the case offers file names, so this test is measuring nothing")
	}
	if !strings.Contains(file, "while IFS= read -r candidate") {
		t.Errorf("the file arm must read one candidate a line, so a path holding a space stays one: %q", file)
	}
	if strings.Contains(file, "COMPREPLY=( $(compgen") {
		t.Errorf("an unquoted $(compgen) is split on IFS, and splits a path holding a space: %q", file)
	}

	// Every variable the body assigns is declared, because the read loop runs
	// in the shell that sourced this file.
	assigned := []string{"cur", "prev", "candidate"}
	local := ""
	for _, line := range strings.Split(body, "\n") {
		if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, "local ") {
			local = trimmed
		}
	}
	if local == "" {
		t.Fatal("the function declares nothing local, so every variable it assigns leaks into the shell")
	}
	for _, name := range assigned {
		if !strings.Contains(local, " "+name) {
			t.Errorf("%s is assigned in the function and not declared: %q", name, local)
		}
		if !strings.Contains(body, name) {
			t.Errorf("%s is declared and assigned nowhere, so this test names a variable the script dropped", name)
		}
	}
}

// The line to add is reported under the name of the file it goes in. A bash
// user handed add_to_zshrc would be gdoc's mistake and not theirs.
func TestCompletionBashWritesTheFileAndNamesBashrc(t *testing.T) {
	noSession(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "gdoc.bash")

	got, code := runJSON(t, "completion", "bash", "--out", out)
	if code != 0 || got["ok"] != true {
		t.Fatalf("completion bash: %v (exit %d)", got, code)
	}
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	if data["shell"] != "bash" {
		t.Errorf("shell: %v", data["shell"])
	}
	if data["wrote"] != out {
		t.Errorf("wrote: got %v, want %s", data["wrote"], out)
	}
	if want := "source " + out; data["add_to_bashrc"] != want {
		t.Errorf("add_to_bashrc: got %v, want %q", data["add_to_bashrc"], want)
	}
	if _, says := data["add_to_zshrc"]; says {
		t.Errorf("the bash report must not name .zshrc: %v", data)
	}

	written, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	script := string(written)
	if !strings.HasPrefix(script, "# bash completion for gdoc") || !strings.HasSuffix(script, "complete -o filenames -F _gdoc gdoc\n") {
		t.Errorf("the file is the script: %q ... %q", head(script), tail(script))
	}
	if !strings.Contains(script, "publish") {
		t.Errorf("the file must carry the commands: %q", script)
	}
}

// A text flag carries the words a caller typed out of somebody's document, and
// nothing on this machine holds a list of those. So both scripts must offer
// nothing after it: no file names, no folder ids, no words at all.
//
// The case is built rather than read out of the table, because the first
// command carrying this kind arrives after the kind does. Both scripts are
// written from the same built table, so what Tab offers is read from the real
// generators.
func TestATextFlagIsOpaqueToBothScripts(t *testing.T) {
	quote := flag{"--quote", kindText, needRequired, "the words to leave the comment on"}
	table := []command{{
		name:    "annotate",
		words:   []string{"<url>"},
		flags:   []flag{quote},
		summary: "leave a comment on the words a caller quoted",
		example: "gdoc annotate <url> --quote text",
	}}

	spec := zshFlagSpec(quote)
	if !strings.Contains(spec, ":text:") {
		t.Errorf("zsh must name the value after a text flag: %q", spec)
	}
	if !strings.HasSuffix(spec, ":text:'") {
		t.Errorf("zsh must offer nothing after a text flag: %q", spec)
	}
	if strings.Contains(spec, "_files") {
		t.Errorf("zsh must not offer file names after a text flag: %q", spec)
	}

	zsh, err := zshScript(table)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(withoutComments(zsh), spec) {
		t.Errorf("the zsh script must carry the spec the flag makes: %q", zsh)
	}

	// Both offered setting flags are in every script, because every command
	// takes them: --config is the one file flag of this table, and --mark is a
	// text flag like --quote and stands in the same arm. The experimental
	// third is in neither script, which is what
	// TestAnExperimentalFlagIsInTheTableAndInNoScreen holds.
	files, opaque := bashFlagArms(table)
	if files != configFlag {
		t.Errorf("the only file flag here is the settings file: %q", files)
	}
	if want := "--quote|" + markFlag; opaque != want {
		t.Errorf("a text flag belongs in the opaque arm: %q, want %q", opaque, want)
	}

	bash, err := bashScript(table)
	if err != nil {
		t.Fatal(err)
	}
	body := withoutComments(bash)
	if !strings.Contains(body, "--quote|"+markFlag+") return ;;") {
		t.Errorf("the bash script must answer for --quote with nothing: %q", bash)
	}
	for _, arm := range flagArms(body) {
		if !strings.Contains(arm, "compgen -f") {
			continue
		}
		if names, _, _ := strings.Cut(arm, ")"); names != configFlag {
			t.Errorf("file names are offered after %q, and only the settings file carries one: %q", names, arm)
		}
	}
}

// flagKinds is one kind per flag name, over the table and the setting flags
// both scripts offer, which is what both scripts are written from. An
// experimental flag is in neither script, so it is in neither this map. A
// name carrying two kinds would make the assertions below unreadable, so it
// stops the test instead.
func flagKinds(t *testing.T, table []command) map[string]kind {
	t.Helper()
	kinds := map[string]kind{}
	all := append(append([]command{}, table...), command{flags: shownSettingFlags()})
	for _, c := range all {
		for _, f := range c.flags {
			if was, seen := kinds[f.name]; seen && was != f.value {
				t.Fatalf("%s carries two kinds, and this test reads the script one line at a time", f.name)
			}
			kinds[f.name] = f.value
		}
	}
	return kinds
}

// --loader writes a loader instead of the table: a few lines of zsh that ask
// the gdoc on PATH for its own completion at every new shell. The gdoc running
// an update is the old binary and cannot know the commands the new one takes,
// so the refresh comes from the installed binary at the first moment it can.
//
// The first line is fixed, because it is one of the two ways `gdoc update`
// knows the file at that path is gdoc's own and not somebody else's.
func TestTheLoaderOpensWithItsOwnFirstLine(t *testing.T) {
	noSession(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "completion.zsh")

	got, code := runJSON(t, "completion", "zsh", "--loader", "--out", out)
	if code != 0 || got["ok"] != true {
		t.Fatalf("completion zsh --loader: %v (exit %d)", got, code)
	}
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data object: %v", got)
	}
	if data["shell"] != "zsh" {
		t.Errorf("shell: %v", data["shell"])
	}
	if data["wrote"] != out {
		t.Errorf("wrote: got %v, want %s", data["wrote"], out)
	}
	if want := "source " + out; data["add_to_zshrc"] != want {
		t.Errorf("add_to_zshrc: got %v, want %q", data["add_to_zshrc"], want)
	}

	written, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if first := head(string(written)); first != "# gdoc completion loader" {
		t.Errorf("the loader must open with its own line: %q", first)
	}
	if strings.Contains(string(written), "#compdef") {
		t.Errorf("the loader is not the script: %q", written)
	}
}

// loaderText is the loader as a shell gets it, read back off the file the
// command wrote. Every assertion below reads it from there rather than from
// the constant, because the file is what is sourced.
func loaderText(t *testing.T) string {
	t.Helper()
	noSession(t)
	out := filepath.Join(t.TempDir(), "completion.zsh")
	got, code := runJSON(t, "completion", "zsh", "--loader", "--out", out)
	if code != 0 || got["ok"] != true {
		t.Fatalf("completion zsh --loader: %v (exit %d)", got, code)
	}
	written, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return string(written)
}

// everyFlagName is every flag name a caller can type, over the table and the
// setting flags, the experimental one included: a loader naming any of them
// but --out and --force would go stale the day that flag moved.
func everyFlagName() []string {
	var names []string
	for _, c := range commands() {
		for _, f := range c.flags {
			names = append(names, f.name)
		}
	}
	for _, f := range settingFlags() {
		names = append(names, f.flag.name)
	}
	return names
}

// The loader is version-free and command-free on purpose: it is written once
// and sourced by every shell from then on, so anything in it that a later
// release moves is a line that goes stale where nobody is looking. The one
// call it makes is `gdoc completion zsh`, with --out and --force.
func TestTheLoaderNamesNoVersionAndNoOtherCommand(t *testing.T) {
	text := loaderText(t)

	if v := regexp.MustCompile(`v[0-9]+\.[0-9]+`).FindString(text); v != "" {
		t.Errorf("the loader names %q, and a version in it goes stale: %q", v, text)
	}
	for _, c := range commands() {
		for _, word := range c.nameWords() {
			if word == "completion" {
				continue
			}
			if regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`).MatchString(text) {
				t.Errorf("the loader names the command word %q: %q", word, text)
			}
		}
	}
	for _, name := range everyFlagName() {
		if name == "--out" || name == "--force" {
			continue
		}
		if strings.Contains(text, name) {
			t.Errorf("the loader names %s, and the one call it makes takes --out and --force: %q", name, text)
		}
	}
	if !strings.Contains(text, "gdoc completion zsh --out") {
		t.Errorf("the loader must ask gdoc for its own completion: %q", text)
	}
}

// A new shell is not a place to print, and a shell with no gdoc on PATH is a
// shell where nothing happens. Two guards, and they are separate on purpose: a
// .zshrc that adds ~/.local/bin to PATH after the source line still sources
// the completion it already had.
//
// Read as bytes, never run: no file under go/ starts zsh or any program, which
// TestNothingRunsAnExternalProgram holds over the test files too. Sourcing it
// in a real shell is release/test-completion.sh.
func TestTheLoaderIsQuietWithoutGdoc(t *testing.T) {
	text := loaderText(t)

	var call, sourced string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if strings.Contains(line, "gdoc completion zsh") {
			call = line
		}
		if strings.Contains(line, "source ") {
			sourced = line
		}
	}
	if call == "" {
		t.Fatalf("the loader asks gdoc nothing: %q", text)
	}
	if sourced == "" {
		t.Fatalf("the loader sources nothing: %q", text)
	}
	if !strings.Contains(call, "$+commands[gdoc]") {
		t.Errorf("the call must be guarded by a check that gdoc is on PATH: %q", call)
	}
	if !strings.Contains(call, ">/dev/null 2>&1") {
		t.Errorf("both streams go to /dev/null: a new shell is not a place to print: %q", call)
	}
	if !strings.Contains(sourced, "[[ -r ") {
		t.Errorf("the source must be guarded by a readable-file check: %q", sourced)
	}
	if strings.Contains(sourced, "commands[gdoc]") {
		t.Errorf("sourcing must not depend on gdoc being on PATH: %q", sourced)
	}
}

// The loader is zsh, and the refusal says so by naming the shell that was
// asked for. bash has no %x and no commands array, and a loader written for it
// would be a file that does nothing.
func TestTheLoaderIsRefusedForBash(t *testing.T) {
	noSession(t)
	out := filepath.Join(t.TempDir(), "completion.bash")

	got, code := runJSON(t, "completion", "bash", "--loader", "--out", out)
	if code == 0 || got["ok"] != false {
		t.Fatalf("a bash loader must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "bash") || !strings.Contains(msg, "zsh") {
		t.Errorf("the refusal must name the shell asked for and the one it writes: %q", msg)
	}
	if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the refused run wrote %s anyway", out)
	}
}

// The loader is a file like the script is, so --out is required for it too.
func TestTheLoaderNeedsOut(t *testing.T) {
	noSession(t)

	got, code := runJSON(t, "completion", "zsh", "--loader")
	if code == 0 || got["ok"] != false {
		t.Fatalf("--loader without --out must be refused: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "--out") {
		t.Errorf("the refusal must name the missing flag: %q", msg)
	}
}

// And the file already there is somebody's, whichever of the two this run
// writes. Not knowing never resolves to overwrite.
func TestTheLoaderRefusesAnExistingOutUnlessForced(t *testing.T) {
	noSession(t)
	out := filepath.Join(t.TempDir(), "already.zsh")
	if err := os.WriteFile(out, []byte("not a loader"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, code := runJSON(t, "completion", "zsh", "--loader", "--out", out)
	if code == 0 || got["ok"] != false {
		t.Fatalf("an existing file must be refused: %v (exit %d)", got, code)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, out) || !strings.Contains(msg, "--force") {
		t.Errorf("the error must name the file and the way to say yes: %q", msg)
	}
	if kept, _ := os.ReadFile(out); string(kept) != "not a loader" {
		t.Fatalf("the refused run must leave the file alone: %q", kept)
	}

	got, code = runJSON(t, "completion", "zsh", "--loader", "--out", out, "--force")
	if code != 0 || got["ok"] != true {
		t.Fatalf("--force replaces it: %v (exit %d)", got, code)
	}
	if replaced, _ := os.ReadFile(out); head(string(replaced)) != "# gdoc completion loader" {
		t.Errorf("the file was replaced by the loader: %q", head(string(replaced)))
	}
}

// completion runs at every new shell now, so it must stay as offline from
// GitHub as every command but help and mcp: a stale stamp and a reach that
// fails the test make no request and leave the stamp where it was. The twin is
// TestReadNeverReachesTheCheck, and the reason this one exists is that nothing
// else in the suite runs this command once a terminal.
func TestCompletionNeverReachesTheCheck(t *testing.T) {
	noSession(t)
	path := checking(t, "v2.2.0", &refusingPlain{t: t})
	stampedAt(t, path, 30*24*time.Hour, lastcheck.Stamp{LatestStable: "v2.3.0"})
	out := filepath.Join(t.TempDir(), "completion.zsh")

	got, code := runJSON(t, "completion", "zsh", "--loader", "--out", out)
	if code != 0 || got["ok"] != true {
		t.Fatalf("completion zsh --loader: %v (exit %d)", got, code)
	}
	if _, ok := dataOf(t, got)["update"]; ok {
		t.Errorf("completion says nothing about releases: %v", got["data"])
	}
	if after := stampOnDisk(t, path); after.LatestStable != "v2.3.0" {
		t.Errorf("completion touched the stamp: %+v", after)
	}
}
