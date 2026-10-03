// What a pipe gets, frozen. Every byte a skill and a log already read.
//
// A terminal is about to get screens instead, and the whole point of that
// change is that nothing else moves. So the text a pipe gets is written down
// here as files, before any rendering code is touched: stderr and stdout of
// the help spellings, bare gdoc and an unknown command, and the plain lines
// `gdoc update` prints beside them.
//
// From this file on the goldens under testdata/pipe are read only. A later
// task that changes one of these bytes has a bug in it, not a new golden. The
// helper writes a golden that is missing, so the first run of a new case
// records it, and never replaces one that is there.
//
// Nothing here reaches the network: the version, the stamp and the reach are
// all fixed by the test, through notice_test.go's checking.

package main

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"gdoc/internal/emit"
	"gdoc/internal/lastcheck"
)

// frozenVersion is the release the frozen runs were built from, and
// frozenStable and frozenNightly what GitHub had said when they ran. Literals,
// so the heading, the envelope's version and the notice line are the same
// bytes on every machine.
const (
	frozenVersion = "v2.9.0"
	frozenStable  = "v2.10.0"
	frozenNightly = "v2.10.1"
)

// frozenCheckedAt stands in for when gdoc last asked. The real value has to be
// inside the day for the run to make no request, so it cannot be a literal;
// the test replaces it with this one before comparing, and everything else in
// the object is compared as it was printed.
const frozenCheckedAt = "2026-10-03T00:00:00Z"

// checkedAtInObject is the one field whose value is a clock reading.
var checkedAtInObject = regexp.MustCompile(`"checked_at":"[^"]*"`)

// pipeRun is one command run with both streams on buffers, which is what a
// skill, a log and a pipe all are.
func pipeRun(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(context.Background(), args, &out, &errOut)
	return out.String(), errOut.String(), code
}

// freezing fixes everything a run of this binary would otherwise read off the
// machine: the version, the record of the last release check, and the reach,
// which refuses every request. The stamp is dated an hour ago, so the run is
// inside the day and asks GitHub nothing.
func freezing(t *testing.T) {
	t.Helper()
	path := checking(t, frozenVersion, &refusingPlain{t})
	stampedAt(t, path, time.Hour, lastcheck.Stamp{
		LatestStable:  frozenStable,
		LatestNightly: frozenNightly,
	})
}

// frozen compares what a run printed with the file that holds it. A golden
// that is not there yet is written and the test fails saying so, because a
// recorded screen is something a person reads once; a golden that is there is
// never replaced from a run.
func frozen(t *testing.T, name, got string) {
	t.Helper()
	if strings.ContainsRune(got, 0x1b) {
		t.Errorf("%s: a pipe gets no escape byte, and this run wrote one: %q", name, got)
	}
	path := filepath.Join("testdata", "pipe", name)
	want, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Fatalf("%s held nothing, so this run wrote it. Read it, then run the test again: from here on it is read only.", path)
	}
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("%s is\n%q\nand the frozen text is\n%q", name, got, string(want))
	}
}

// Help on a pipe is the text it was, byte for byte: the notice line, the blank
// line under it, the version heading and the words, on stderr, and one JSON
// object on stdout.
func TestHelpOnAPipeIsTodaysTextByteForByte(t *testing.T) {
	for _, c := range []struct {
		name string
		args []string
	}{
		{name: "help", args: []string{"help"}},
		{name: "help-publish", args: []string{"help", "publish"}},
		{name: "help-comments", args: []string{"help", "comments"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			freezing(t)

			stdout, stderr, code := pipeRun(t, c.args...)
			if code != 0 {
				t.Fatalf("%v is an answer and exited %d: %s", c.args, code, stdout)
			}
			frozen(t, c.name+".stderr.golden", stderr)
			frozen(t, c.name+".stdout.golden", checkedAtInObject.ReplaceAllString(stdout, `"checked_at":"`+frozenCheckedAt+`"`))
		})
	}
}

// Bare gdoc on a pipe reads the whole help on stderr and still refuses on
// stdout, with the exit code that goes with the refusal.
func TestBareGdocOnAPipeIsTodaysTextByteForByte(t *testing.T) {
	freezing(t)

	stdout, stderr, code := pipeRun(t)
	if code != 1 {
		t.Fatalf("bare gdoc named no command, so it fails and exits 1, and this run exited %d", code)
	}
	frozen(t, "bare.stderr.golden", stderr)
	frozen(t, "bare.stdout.golden", stdout)
}

// A word gdoc does not answer to is refused by name, on stdout, with nothing
// on stderr at all.
func TestAnUnknownCommandOnAPipeIsTodaysTextByteForByte(t *testing.T) {
	freezing(t)

	stdout, stderr, code := pipeRun(t, "frobnicate")
	if code != 1 {
		t.Fatalf("an unknown command fails and exits 1, and this run exited %d", code)
	}
	frozen(t, "unknown.stderr.golden", stderr)
	frozen(t, "unknown.stdout.golden", stdout)
}

// The step list on a pipe is one plain line per step that ended, in the order
// they ended, and nothing for a step that was planned and never ran. A step
// that is still running has printed nothing yet either.
func TestUpdateStepsOnAPipeAreTodaysLinesByteForByte(t *testing.T) {
	var buf bytes.Buffer
	p := newProgress(&buf, "update")
	p.Plan(stepRead, stepChoose, stepChecksums, stepDownload, stepVerify, stepReplace, stepReadBack)
	p.Start(stepRead, "")
	p.Done("nhusnullin/gdoc, 3 listed")
	p.Start(stepChoose, "")
	p.Done(frozenStable + ", stable, darwin-arm64, from " + frozenVersion)
	p.Start(stepChecksums, "SHA256SUMS-"+frozenStable)
	p.Done("")
	p.Start(stepDownload, "gdoc-darwin-arm64.zip, 6.1 MB")
	p.Done("")
	p.Start(stepVerify, "")
	p.Done("sha256 matches")
	p.Start(stepReplace, "")
	p.Finish("gdoc " + frozenStable + " installed. gdoc update --rollback goes back.")

	frozen(t, "update-steps.golden", buf.String())
}

// A failed step carries the cross and its reason on the line under it, the
// steps after it are never drawn, and a failed run prints no result line.
func TestUpdateFailureOnAPipeIsTodaysLinesByteForByte(t *testing.T) {
	var buf bytes.Buffer
	p := newProgress(&buf, "update")
	p.Plan(stepChecksums, stepDownload, stepVerify, stepReplace, stepReadBack)
	p.Start(stepChecksums, "SHA256SUMS-"+frozenStable)
	p.Done("")
	p.Start(stepDownload, "gdoc-darwin-arm64.zip, 6.1 MB")
	p.Fail(errors.New("gdoc-darwin-arm64.zip could not be downloaded, so nothing was replaced"))
	p.Finish(resultLine(emit.Result{OK: false, Error: "the download failed"}))

	frozen(t, "update-steps-failed.golden", buf.String())
}

// The result line under the steps is the words it was, for an install on its
// own and for the two ways a --desktop run names the extension it wrote.
func TestTheUpdateResultLineIsTodaysWordsByteForByte(t *testing.T) {
	installed := updateData{
		Installed: frozenVersion,
		Action:    "updated",
		To:        frozenStable,
		Path:      "/usr/local/bin/gdoc",
		Verified:  true,
		Previous:  frozenVersion,
	}
	withExtension := installed
	withExtension.Extension = "/Users/person/Library/Application Support/gdoc/gdoc.mcpb"
	opened := withExtension
	opened.ExtensionOpened = true

	for name, d := range map[string]updateData{
		"update-result.golden":                  installed,
		"update-result-extension.golden":        withExtension,
		"update-result-extension-opened.golden": opened,
	} {
		frozen(t, name, resultLine(emit.Result{OK: true, Data: d})+"\n")
	}
}
