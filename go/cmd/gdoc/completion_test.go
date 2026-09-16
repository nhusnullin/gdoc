package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	kinds := map[string]kind{}
	for _, c := range table {
		for _, f := range c.flags {
			if was, seen := kinds[f.name]; seen && was != f.value {
				t.Fatalf("%s carries two kinds, and this test reads the script a line at a time", f.name)
			}
			kinds[f.name] = f.value
		}
	}
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
	kinds := map[string]kind{}
	for _, c := range table {
		for _, f := range c.flags {
			if was, seen := kinds[f.name]; seen && was != f.value {
				t.Fatalf("%s carries two kinds, and this test reads the script an arm at a time", f.name)
			}
			kinds[f.name] = f.value
		}
	}

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
