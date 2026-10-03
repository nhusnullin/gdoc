// The Claude Desktop extension `gdoc update --desktop` writes, and the one
// program gdoc runs.
//
// Nothing here starts a program and nothing here reaches GitHub. The opener is
// a variable a test replaces, so what is measured is the argv gdoc would hand
// it, and the release is the stub listing update_test.go already builds.

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The template as a release carries it, short enough to read in one line and
// carrying both placeholders. The real one is release/mcpb/manifest.json, and
// manifest_test.go is what judges that file.
const (
	newTemplate = `{"name":"gdoc","version":"@VERSION@","server":{"mcp_config":{"command":"@BIN@","args":["mcp","--trusted-email-domains=${user_config.trusted_email_domains}"]}}}`
	oldTemplate = `{"name":"gdoc","version":"@VERSION@","server":{"mcp_config":{"command":"@BIN@","args":["mcp"]}}}`
)

// opened is what the one program gdoc may run was handed, call by call. A test
// that says nothing was run proves it from this.
type opened struct {
	calls [][2]string
}

// desktopSeams points the opener at a recorder and the operating system at the
// one named, so a test can ask what a Linux run would do from a Mac.
func desktopSeams(t *testing.T, osName string) *opened {
	t.Helper()
	rec := &opened{}
	oldRun, oldOS := runProgram, desktopOS
	runProgram = func(name, arg string) error {
		rec.calls = append(rec.calls, [2]string{name, arg})
		return nil
	}
	desktopOS = osName
	t.Cleanup(func() { runProgram, desktopOS = oldRun, oldOS })
	return rec
}

// mcpbBeside writes the extension file a person already has, carrying the
// manifest handed in. It is the zip with one entry that a .mcpb is.
func mcpbBeside(t *testing.T, binary, manifest string) string {
	t.Helper()
	path := filepath.Join(filepath.Dir(binary), "gdoc.mcpb")
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(manifest)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// manifestIn reads the one manifest out of the extension file at path.
func manifestIn(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no extension was written at %s: %v", path, err)
	}
	r, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("%s is not a zip Claude Desktop could open: %v", path, err)
	}
	if len(r.File) != 1 || r.File[0].Name != "manifest.json" {
		t.Fatalf("%s must hold exactly one manifest.json: %d entries", path, len(r.File))
	}
	rc, err := r.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	var into map[string]any
	if err := json.NewDecoder(rc).Decode(&into); err != nil {
		t.Fatalf("the manifest in %s is not JSON: %v", path, err)
	}
	return into
}

// command is the command the manifest tells Claude Desktop to start.
func startedCommand(t *testing.T, manifest map[string]any) string {
	t.Helper()
	server, ok := manifest["server"].(map[string]any)
	if !ok {
		t.Fatalf("the manifest names no server: %v", manifest)
	}
	config, ok := server["mcp_config"].(map[string]any)
	if !ok {
		t.Fatalf("the manifest names no mcp_config: %v", manifest)
	}
	return config["command"].(string)
}

// Plain `gdoc update` is the run it has always been. The release carries the
// extension template and the run leaves it in the zip: nothing is written
// beside the binary, and no program is started.
func TestPlainUpdateWritesNoExtensionAndRunsNothing(t *testing.T) {
	pl := &stubPlain{
		listing: listing("v2.1.0"),
		files:   publishedWith(t, "v2.1.0", []byte("new"), zipEntry{"mcpb/manifest.json", []byte(newTemplate)}),
	}
	path := installedAt(t, "v2.0.0", []byte("old"), pl)
	rec := desktopSeams(t, "darwin")

	got, code := runJSON(t, "update")
	if code != 0 || got["ok"] != true {
		t.Fatalf("the update must answer: %v (exit %d)", got, code)
	}
	data := updateObject(t, got)
	if data["action"] != "updated" || data["extension"] != nil {
		t.Errorf("a plain update installs the binary and writes no extension: %v", data)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "gdoc.mcpb")); err == nil {
		t.Error("a plain update wrote an extension file")
	}
	if len(rec.calls) != 0 {
		t.Errorf("a plain update starts no program: %v", rec.calls)
	}
}

// A release whose template differs from the extension this machine has is the
// one case plain update says anything about Claude Desktop: one line naming the
// command that would refresh it. It still writes nothing and runs nothing.
func TestPlainUpdateHintsWhenTheTemplateChanged(t *testing.T) {
	pl := &stubPlain{
		listing: listing("v2.1.0"),
		files:   publishedWith(t, "v2.1.0", []byte("new"), zipEntry{"mcpb/manifest.json", []byte(newTemplate)}),
	}
	path := installedAt(t, "v2.0.0", []byte("old"), pl)
	rec := desktopSeams(t, "darwin")
	mcpb := mcpbBeside(t, path, strings.NewReplacer("@BIN@", path, "@VERSION@", "2.0.0").Replace(oldTemplate))
	before := fileText(t, mcpb)

	got, code := runJSON(t, "update")
	if code != 0 || got["ok"] != true {
		t.Fatalf("the update must answer: %v (exit %d)", got, code)
	}
	warns := warningsOf(t, got)
	if !hasWarning(warns, "gdoc update --desktop") || !hasWarning(warns, "Claude Desktop") {
		t.Errorf("the hint must name Claude Desktop and the command that refreshes it: %q", warns)
	}
	if fileText(t, mcpb) != before {
		t.Error("the hint is a line, and this run rewrote the extension")
	}
	if len(rec.calls) != 0 {
		t.Errorf("a plain update starts no program: %v", rec.calls)
	}
}

// Two runs that say nothing. With no extension beside the binary there is
// nobody to tell, and a template that changed only its version number changed
// nothing a person has to do anything about.
func TestPlainUpdateIsSilentWithNoMcpbOrNoChange(t *testing.T) {
	for _, run := range []struct {
		name string
		mcpb func(t *testing.T, path string)
	}{
		{name: "no extension beside the binary"},
		{
			name: "the same template, an older version in it",
			mcpb: func(t *testing.T, path string) {
				mcpbBeside(t, path, strings.NewReplacer("@BIN@", path, "@VERSION@", "2.0.0").Replace(newTemplate))
			},
		},
	} {
		pl := &stubPlain{
			listing: listing("v2.1.0"),
			files:   publishedWith(t, "v2.1.0", []byte("new"), zipEntry{"mcpb/manifest.json", []byte(newTemplate)}),
		}
		path := installedAt(t, "v2.0.0", []byte("old"), pl)
		desktopSeams(t, "darwin")
		if run.mcpb != nil {
			run.mcpb(t, path)
		}

		got, code := runJSON(t, "update")
		if code != 0 || got["ok"] != true {
			t.Fatalf("%s: the update must answer: %v (exit %d)", run.name, got, code)
		}
		if warns := warningsOf(t, got); hasWarning(warns, "--desktop") {
			t.Errorf("%s: this run has nothing to say about Claude Desktop: %q", run.name, warns)
		}
	}
}

// --desktop fills the template out of the zip it verified with this machine's
// own path and the version it installed, and writes it beside the binary.
func TestDesktopWritesTheMcpbFromTheZipsTemplate(t *testing.T) {
	pl := &stubPlain{
		listing: listing("v2.1.0"),
		files:   publishedWith(t, "v2.1.0", []byte("new"), zipEntry{"mcpb/manifest.json", []byte(newTemplate)}),
	}
	path := installedAt(t, "v2.0.0", []byte("old"), pl)
	desktopSeams(t, "darwin")

	got, code := runJSON(t, "update", "--desktop")
	if code != 0 || got["ok"] != true {
		t.Fatalf("the update must answer: %v (exit %d)", got, code)
	}
	data := updateObject(t, got)
	mcpb := filepath.Join(filepath.Dir(path), "gdoc.mcpb")
	if data["extension"] != mcpb {
		t.Errorf("the object must name the extension it wrote: %v", data)
	}
	if data["extension_opened"] != true {
		t.Errorf("on macOS the extension is handed to Claude Desktop: %v", data)
	}
	manifest := manifestIn(t, mcpb)
	if got := startedCommand(t, manifest); got != path {
		t.Errorf("the manifest must start the binary this run installed, and names %q", got)
	}
	// The tag without its v, which is what a manifest version is.
	if manifest["version"] != "2.1.0" {
		t.Errorf("the manifest version must be the release's, without the v: %v", manifest["version"])
	}
	if fileText(t, path) != "new" {
		t.Errorf("the binary must be the one that was published: %q", fileText(t, path))
	}
}

// The one program gdoc runs, and the whole of its argv: the opener by its full
// path, and the file that was just written. Nothing else, and once.
func TestDesktopCallsTheRunnerWithOpenAndThePathOnly(t *testing.T) {
	pl := &stubPlain{
		listing: listing("v2.1.0"),
		files:   publishedWith(t, "v2.1.0", []byte("new"), zipEntry{"mcpb/manifest.json", []byte(newTemplate)}),
	}
	path := installedAt(t, "v2.0.0", []byte("old"), pl)
	rec := desktopSeams(t, "darwin")

	got, code := runJSON(t, "update", "--desktop")
	if code != 0 || got["ok"] != true {
		t.Fatalf("the update must answer: %v (exit %d)", got, code)
	}
	want := [2]string{"/usr/bin/open", filepath.Join(filepath.Dir(path), "gdoc.mcpb")}
	if len(rec.calls) != 1 || rec.calls[0] != want {
		t.Errorf("the one program gdoc runs is %v, and this run asked for %v", want, rec.calls)
	}
}

// Claude Desktop is a macOS application here, so off macOS the file is written
// and nothing is started: the person carries it to a Mac and opens it there.
func TestDesktopOffMacOSWritesAndRunsNothing(t *testing.T) {
	pl := &stubPlain{
		listing: listing("v2.1.0"),
		files:   publishedWith(t, "v2.1.0", []byte("new"), zipEntry{"mcpb/manifest.json", []byte(newTemplate)}),
	}
	path := installedAt(t, "v2.0.0", []byte("old"), pl)
	rec := desktopSeams(t, "linux")

	got, code := runJSON(t, "update", "--desktop")
	if code != 0 || got["ok"] != true {
		t.Fatalf("the update must answer: %v (exit %d)", got, code)
	}
	data := updateObject(t, got)
	mcpb := filepath.Join(filepath.Dir(path), "gdoc.mcpb")
	if data["extension"] != mcpb {
		t.Errorf("the extension is written on every platform: %v", data)
	}
	if data["extension_opened"] != nil {
		t.Errorf("off macOS nothing was opened: %v", data)
	}
	if _, err := os.Stat(mcpb); err != nil {
		t.Errorf("the file must be there for somebody to carry: %v", err)
	}
	if len(rec.calls) != 0 {
		t.Errorf("off macOS gdoc starts no program: %v", rec.calls)
	}
}

// A release with no template in its zip cannot write an extension, and the run
// asked for one. It is refused before the binary is replaced, so a person who
// wanted gdoc in their chat is left with the gdoc they had rather than a new
// binary and a warning they scrolled past.
func TestAZipWithoutTheTemplateIsRefusedBeforeTheBinaryIsReplaced(t *testing.T) {
	pl := &stubPlain{
		listing: listing("v2.1.0"),
		files:   published(t, "v2.1.0", []byte("new")),
	}
	path := installedAt(t, "v2.0.0", []byte("old"), pl)
	rec := desktopSeams(t, "darwin")

	got, code := runJSON(t, "update", "--desktop")
	if code == 0 || got["ok"] != false {
		t.Fatalf("a release with no template must be refused: %v (exit %d)", got, code)
	}
	if msg, _ := got["error"].(string); !strings.Contains(msg, "mcpb/manifest.json") {
		t.Errorf("the refusal must name the file the zip has not got: %q", msg)
	}
	nothingMoved(t, path, "old")
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "gdoc.mcpb")); err == nil {
		t.Error("nothing was replaced, so no extension may have been written")
	}
	if len(rec.calls) != 0 {
		t.Errorf("the run was refused, so no program may have started: %v", rec.calls)
	}
}

// The extension follows the release, not the binary, so a machine already
// running the newest gdoc still gets the newest extension. The zip is
// downloaded for the template alone and no binary moves.
func TestDesktopWhenAlreadyNewestStillRefreshesTheExtension(t *testing.T) {
	pl := &stubPlain{
		listing: listing("v2.1.0"),
		files:   publishedWith(t, "v2.1.0", []byte("new"), zipEntry{"mcpb/manifest.json", []byte(newTemplate)}),
	}
	path := installedAt(t, "v2.1.0", []byte("current"), pl)
	rec := desktopSeams(t, "darwin")

	got, code := runJSON(t, "update", "--desktop")
	if code != 0 || got["ok"] != true {
		t.Fatalf("the update must answer: %v (exit %d)", got, code)
	}
	data := updateObject(t, got)
	mcpb := filepath.Join(filepath.Dir(path), "gdoc.mcpb")
	if data["action"] != "up_to_date" || data["extension"] != mcpb {
		t.Errorf("nothing was installed and the extension was refreshed: %v", data)
	}
	manifest := manifestIn(t, mcpb)
	if startedCommand(t, manifest) != path || manifest["version"] != "2.1.0" {
		t.Errorf("the manifest must name this binary and this release: %v", manifest)
	}
	nothingMoved(t, path, "current")
	if len(rec.calls) != 1 {
		t.Errorf("the extension was written, so it was opened once: %v", rec.calls)
	}
}

// --rollback puts an earlier binary back and --check writes nothing at all, and
// --desktop writes a file naming a version. Each pair names two runs, so it is
// refused naming both, before anything is fetched.
func TestDesktopIsRefusedWithRollbackOrCheck(t *testing.T) {
	for _, want := range []struct {
		args  []string
		names []string
	}{
		{[]string{"update", "--desktop", "--rollback"}, []string{"--desktop", "--rollback"}},
		{[]string{"update", "--desktop", "--check"}, []string{"--desktop", "--check"}},
	} {
		pl := &stubPlain{listing: listing("v2.1.0")}
		path := installedAt(t, "v2.0.0", []byte("old"), pl)
		rec := desktopSeams(t, "darwin")

		got, code := runJSON(t, want.args...)
		if code == 0 || got["ok"] != false {
			t.Errorf("%v must be refused: %v (exit %d)", want.args, got, code)
			continue
		}
		msg, _ := got["error"].(string)
		for _, name := range want.names {
			if !strings.Contains(msg, name) {
				t.Errorf("%v must be refused naming %q: %q", want.args, name, msg)
			}
		}
		if len(pl.got) != 0 {
			t.Errorf("%v was refused, so nothing may be fetched: %v", want.args, pl.got)
		}
		if len(rec.calls) != 0 {
			t.Errorf("%v was refused, so no program may have started: %v", want.args, rec.calls)
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(path), "gdoc.mcpb")); err == nil {
			t.Errorf("%v was refused, so no extension may have been written", want.args)
		}
	}
}

// A path that would end the JSON string it is written into is refused naming
// the character, because a manifest Claude Desktop cannot read is a connector
// that failed for no reason a person can see.
func TestAPathThatWouldBreakTheManifestIsRefusedByName(t *testing.T) {
	for _, path := range []string{`/tmp/a"b/gdoc`, `/tmp/a\b/gdoc`} {
		_, err := fillManifest([]byte(newTemplate), path, "v2.9.0")
		if err == nil {
			t.Errorf("%s was written into a manifest", path)
			continue
		}
		if !strings.Contains(err.Error(), path) {
			t.Errorf("the refusal must name the path: %v", err)
		}
	}
}

// The version in the manifest is the release's number without its v, and a
// build naming no release gets the one number a manifest accepts for no number.
func TestTheManifestVersionIsTheTagWithoutItsV(t *testing.T) {
	for tag, want := range map[string]string{
		"v2.9.0": "2.9.0",
		"2.9.0":  "2.9.0",
		"":       "0.0.0-dev",
	} {
		got, err := fillManifest([]byte(newTemplate), "/tmp/gdoc", tag)
		if err != nil {
			t.Fatalf("%q: %v", tag, err)
		}
		var into map[string]any
		if err := json.Unmarshal(got, &into); err != nil {
			t.Fatal(err)
		}
		if into["version"] != want {
			t.Errorf("%q must fill the manifest version with %q, and filled %v", tag, want, into["version"])
		}
	}
}

// The two extension steps are drawn where they happen, and the line under them
// says what the person does next. Installing an extension restarts the chat
// process and leaves agent mode on the binary it already started, so the words
// are quit and open again, never toggle.
func TestADesktopRunDrawsTheTwoExtensionStepsAndSaysQuitAndOpenAgain(t *testing.T) {
	pl := &stubPlain{
		listing: listing("v2.1.0"),
		files:   publishedWith(t, "v2.1.0", []byte("new"), zipEntry{"mcpb/manifest.json", []byte(newTemplate)}),
	}
	path := installedAt(t, "v2.0.0", []byte("old"), pl)
	desktopSeams(t, "darwin")

	got, _, stderr, code := runUpdateCapturing(t, "update", "--desktop")
	if code != 0 || got["ok"] != true {
		t.Fatalf("the update must answer: %v (exit %d)", got, code)
	}
	// The template is read between the check and the replacement, because a
	// release that cannot write an extension is refused while the binary on
	// this machine is still untouched.
	order := []string{"verify checksum", "read extension", "replace binary", "read back", "write extension"}
	at := 0
	for _, step := range order {
		i := strings.Index(stderr[at:], "✓ "+step)
		if i < 0 {
			t.Fatalf("the steps must be drawn in the order %v: %s", order, stderr)
		}
		at += i
	}
	if !strings.Contains(stderr, filepath.Join(filepath.Dir(path), "gdoc.mcpb")) {
		t.Errorf("the extension step must name the file it wrote: %s", stderr)
	}
	if !strings.Contains(stderr, "quit Claude Desktop and open it again") {
		t.Errorf("the line must say to quit Claude Desktop and open it again: %s", stderr)
	}
	if strings.Contains(stderr, "toggle") {
		t.Errorf("a toggle restarts the chat process alone, so the line never says it: %s", stderr)
	}
}
