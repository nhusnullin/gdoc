// The update screen: the step list a person reads on a terminal, recorded
// byte for byte.
//
// Every run here draws over a buffer through newLiveProgress, which is the
// terminal form with no terminal involved, at eighty columns with the colour
// off, and with a spinner that never turns. So a recorded screen is the same
// bytes on every machine and in CI. The plain lines a pipe gets are frozen
// next door in pipe_test.go, and nothing here may move them.

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"gdoc/internal/emit"
	"gdoc/internal/tty"
	"gdoc/internal/update"
)

// The run every recorded screen is drawn from, as literals: the gdoc that is
// running, the one it is taking, and the machine it is on. A platform read off
// runtime would make the recording the machine's rather than the screen's.
const (
	screenPlatform = "darwin-arm64"
	screenAsset    = "gdoc-" + screenPlatform + ".zip"
	screenSums     = "SHA256SUMS-" + frozenStable
	screenPath     = "/usr/local/bin/gdoc"
	screenMcpb     = "/Users/person/Library/Application Support/gdoc/gdoc.mcpb"
)

// liveUpdate is the step list drawn as a terminal gets it. The spinner is
// stopped before it can start, by handing the list the two channels it would
// have made itself with the stopped one already closed, so the frame in a
// recorded screen is the first one and no goroutine writes under the test.
func liveUpdate(t *testing.T, w io.Writer) *progress {
	t.Helper()
	t.Setenv("COLUMNS", "80")
	p := newLiveProgress(w, "update", tty.NewStyle(tty.NoColour))
	p.stop, p.stopped = make(chan struct{}), make(chan struct{})
	close(p.stopped)
	return p
}

// readAndChoose is the two steps every run takes, drawn as a run that found
// the release it is taking.
func readAndChoose(p *progress) {
	p.Label(frozenVersion)
	p.Plan(stepRead, stepChoose)
	p.Start(stepRead, "")
	p.Done(updateRepo + ", 30 listed")
	p.Start(stepChoose, "")
	p.Done(frozenStable + ", stable, " + screenPlatform + ", from " + frozenVersion)
	p.Label(frozenVersion + " › " + frozenStable)
}

// drawInstall is the five steps a run that takes a release draws, up to the
// one named by stopAt, which is where it fails. An empty stopAt runs them all.
func drawInstall(p *progress, stopAt string, reason error) {
	for _, s := range []struct{ name, detail, done string }{
		{stepChecksums, screenSums, ""},
		{stepDownload, screenAsset + ", 6.1 MB", ""},
		{stepVerify, "", "matches " + screenSums},
		{stepReplace, screenPath, screenPath + ", the old one kept beside it"},
		{stepReadBack, "", "sha256 4f2a9c1be30d"},
	} {
		p.Start(s.name, s.detail)
		if s.name == stopAt {
			p.Fail(reason)
			return
		}
		p.Done(s.done)
	}
}

// installedData is the object a run that installed printed, which is what the
// lines under the box restate.
func installedData() updateData {
	return updateData{
		Installed:     frozenVersion,
		LatestStable:  frozenStable,
		LatestNightly: frozenNightly,
		Action:        "updated",
		To:            frozenStable,
		Path:          screenPath,
		Verified:      true,
		SHA256:        "4f2a9c1be30d",
		Previous:      screenPath + ".previous",
	}
}

// The six ways a run ends, each recorded as the screen a person is left
// reading. A golden that is not there is written and the test fails saying
// so, because a recorded screen is something a person reads once.
func TestTheUpdateScreensAreTheirRecordedBytes(t *testing.T) {
	for _, c := range []struct {
		name string
		draw func(t *testing.T, p *progress)
	}{
		{name: "update-running", draw: func(t *testing.T, p *progress) {
			readAndChoose(p)
			p.Plan(installSteps(false)...)
			p.Start(stepChecksums, screenSums)
			p.Done("")
			p.Start(stepDownload, screenAsset+", 6.1 MB")
		}},
		{name: "update-installed", draw: func(t *testing.T, p *progress) {
			readAndChoose(p)
			p.Plan(installSteps(false)...)
			drawInstall(p, "", nil)
			p.Finish(resultLines(emit.Result{OK: true, Data: installedData()}, true)...)
		}},
		{name: "update-uptodate", draw: func(t *testing.T, p *progress) {
			p.Label(frozenVersion)
			p.Plan(stepRead, stepChoose)
			p.Start(stepRead, "")
			p.Done(updateRepo + ", 30 listed")
			p.Start(stepChoose, "")
			p.Done(frozenVersion + ", stable, " + screenPlatform + ", from " + frozenVersion)
			d := updateData{Installed: frozenVersion, LatestStable: frozenVersion, Action: "checked", Path: screenPath}
			p.Finish(resultLines(emit.Result{OK: true, Data: d}, true)...)
		}},
		{name: "update-offline", draw: func(t *testing.T, p *progress) {
			p.Label(frozenVersion)
			p.Plan(stepRead, stepChoose)
			p.Start(stepRead, "")
			p.Fail(errors.New("Get \"https://api.github.com/repos/" + updateRepo + "/releases\": dial tcp: lookup api.github.com: no such host"))
			d := updateData{Installed: frozenVersion, Action: string(update.Unreachable), Path: screenPath}
			p.Finish(resultLines(emit.Result{OK: true, Data: d}, true)...)
		}},
		{name: "update-failed", draw: func(t *testing.T, p *progress) {
			readAndChoose(p)
			p.Plan(installSteps(false)...)
			drawInstall(p, stepVerify, errors.New(screenAsset+" hashes to "+strings.Repeat("a", 64)+" and the release said "+strings.Repeat("b", 64)+", so it is not the file the release published"))
			p.Finish(resultLines(emit.Result{OK: false, Error: "it is not the file the release published"}, true)...)
		}},
		{name: "update-extension", draw: func(t *testing.T, p *progress) {
			readAndChoose(p)
			p.Plan(installSteps(true)...)
			p.Start(stepChecksums, screenSums)
			p.Done("")
			p.Start(stepDownload, screenAsset+", 6.1 MB")
			p.Done("")
			p.Start(stepVerify, "")
			p.Done("matches " + screenSums)
			p.Start(stepTemplate, "gdoc.mcpb.template")
			p.Done("")
			p.Start(stepReplace, screenPath)
			p.Done(screenPath + ", the old one kept beside it")
			p.Start(stepReadBack, "")
			p.Done("sha256 4f2a9c1be30d")
			p.Start(stepExtension, screenMcpb)
			p.Done(screenMcpb + ", handed to Claude Desktop")
			d := installedData()
			d.Extension, d.ExtensionOpened = screenMcpb, true
			p.Finish(resultLines(emit.Result{OK: true, Data: d}, true)...)
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var buf strings.Builder
			p := liveUpdate(t, &buf)
			c.draw(t, p)
			recorded(t, c.name+"-80.golden", lastScreen(buf.String()))
		})
	}
}

// Each mark is the role its state is: the spinner a title, a done step the
// ok green, a failed one the fail red and a step that has not run yet the
// muted grey of the pending ring. The bytes are stated here as literals, so a
// mark that silently changed colour fails.
func TestTheStepMarksAreTheirRolesOnATerminal(t *testing.T) {
	t.Setenv("COLUMNS", "80")
	var buf strings.Builder
	p := newLiveProgress(&buf, "update", sixteen)
	p.stop, p.stopped = make(chan struct{}), make(chan struct{})
	close(p.stopped)
	p.Plan(stepRead, stepChoose, stepChecksums)
	p.Start(stepRead, "")
	p.Done("")
	p.Start(stepChoose, "")

	screen := lastScreen(buf.String())
	for what, want := range map[string]string{
		"a done step":    litGreen + "✓" + litReset,
		"a running step": litBlue + "⠋" + litReset,
		"a pending step": litDim + "○" + litReset,
	} {
		if !strings.Contains(screen, want) {
			t.Errorf("%s is drawn %q, and the screen is %q", what, want, screen)
		}
	}

	p.Start(stepChecksums, "")
	p.Fail(errors.New("offline"))
	if want := litRed + "✗" + litReset; !strings.Contains(lastScreen(buf.String()), want) {
		t.Errorf("a failed step is drawn %q, and the screen is %q", want, lastScreen(buf.String()))
	}
}

// A pending ring is a terminal's own: a pipe gets the blank it always got,
// because a plain line is one step per line and nothing is waiting on it.
func TestThePendingRingIsTheTerminalsAlone(t *testing.T) {
	var buf strings.Builder
	stubTerminal(t, false)
	p := newProgress(&buf, "update")
	p.Plan(stepRead, stepChoose)
	p.Start(stepRead, "")
	p.Done("")
	p.Finish("")

	if strings.Contains(buf.String(), pendingMark) {
		t.Errorf("a pipe gets no pending ring: %q", buf.String())
	}
}

// A reason too long for its cell is broken into lines, and a hash with no
// space in it is cut where the line ends rather than shortened: the pieces
// read off the screen and put back together are the sentence again.
func TestALongReasonWrapsInsideTheBoxAndIsNeverShortened(t *testing.T) {
	reason := screenAsset + " hashes to " + strings.Repeat("a", 64) + " and the release said " + strings.Repeat("b", 64) + ", so it is not the file the release published"
	var buf strings.Builder
	p := liveUpdate(t, &buf)
	p.Plan(stepVerify)
	p.Start(stepVerify, "")
	p.Fail(errors.New(reason))
	p.Finish("")

	screen := lastScreen(buf.String())
	if strings.Contains(screen, "…") {
		t.Errorf("nothing on the screen is shortened, and this one carries the mark of a cut: %q", screen)
	}
	var cells []string
	for _, line := range strings.Split(strings.TrimRight(screen, "\n"), "\n") {
		parts := strings.Split(line, "│")
		if len(parts) != 4 {
			continue
		}
		cells = append(cells, strings.TrimSpace(parts[2]))
	}
	// A line broken at a space loses that space and a word cut in the middle
	// loses nothing, so what every line holds together is the reason with its
	// blanks taken out. Nothing is dropped and nothing stands in for anything.
	blankless := func(s string) string { return strings.ReplaceAll(s, " ", "") }
	if got := blankless(strings.Join(cells, "")); got != blankless(reason) {
		t.Errorf("the lines put back together are\n%q\nand the reason is\n%q", got, blankless(reason))
	}
	for _, line := range strings.Split(strings.TrimRight(screen, "\n"), "\n") {
		if got := tty.VisibleWidth(line); got != 80 {
			t.Errorf("every line of the box is eighty columns, and this one is %d: %q", got, line)
		}
	}
}

// The extension line is a second thing for a person to do, so on a terminal it
// is a line of its own under the box. On a pipe it is the one line it always
// was, which the frozen goldens hold.
func TestTheExtensionIsItsOwnLineOnATerminalAndOneLineOnAPipe(t *testing.T) {
	d := installedData()
	d.Extension, d.ExtensionOpened = screenMcpb, true
	r := emit.Result{OK: true, Data: d}

	live := resultLines(r, true)
	if len(live) != 2 {
		t.Fatalf("a terminal reads the run and the extension on two lines, and this is %q", live)
	}
	if live[0] != actionLine(d) || live[1] != extensionLine(d) {
		t.Errorf("the two lines are the run and the extension, and they are %q", live)
	}

	piped := resultLines(r, false)
	if len(piped) != 1 || piped[0] != resultLine(r) {
		t.Errorf("a pipe reads the one line it always read, and this is %q", piped)
	}
}

// A run that failed says nothing under the box: the cross and its reason are
// already the last thing on the screen.
func TestAFailedRunHasNoLineUnderTheBox(t *testing.T) {
	if lines := resultLines(emit.Result{OK: false, Error: "the download failed"}, true); len(lines) != 0 {
		t.Errorf("a failure writes no result line, and this run wrote %q", lines)
	}
}

// The top border says which gdoc is running and which one the run is taking,
// once it has chosen one. A checkout build names no version, so it says
// nothing at all.
func TestTheTopBorderNamesTheVersionsOfTheRun(t *testing.T) {
	v := func(s string) update.Version {
		parsed, err := update.Parse(s)
		if err != nil {
			t.Fatal(err)
		}
		return parsed
	}
	for _, c := range []struct{ installed, to, want string }{
		{installed: frozenVersion, to: frozenStable, want: frozenVersion + " › " + frozenStable},
		{installed: frozenVersion, to: frozenVersion, want: frozenVersion},
		{installed: frozenVersion, want: frozenVersion},
		{to: frozenStable, want: frozenStable},
	} {
		var installed, to update.Version
		if c.installed != "" {
			installed = v(c.installed)
		}
		if c.to != "" {
			to = v(c.to)
		}
		if got := takingLabel(installed, to); got != c.want {
			t.Errorf("a run from %q to %q is labelled %q, and must be %q", c.installed, c.to, got, c.want)
		}
	}

	var buf strings.Builder
	p := liveUpdate(t, &buf)
	readAndChoose(p)
	if want := frozenVersion + " › " + frozenStable; !strings.Contains(lastScreen(buf.String()), want) {
		t.Errorf("the top border must carry %q: %q", want, lastScreen(buf.String()))
	}
}

// lastScreen is the frame a person is left reading: everything the run wrote
// after the last clear, which is the final draw of the box and the lines under
// it. The frames before it are the redraw rubbing itself out.
func lastScreen(s string) string {
	at := strings.LastIndex(s, tty.ClearBelow)
	if at < 0 {
		return s
	}
	return s[at+len(tty.ClearBelow):]
}

// The label on a real run is the run's own, which the half above cannot see:
// its helper hands the list the words itself. This one runs the command
// through the live list over a buffer and reads the top border of the frame a
// person is left with, so deleting either Label call in update.go fails here.
// It reads that one line rather than the whole frame, because the versions are
// in the choose row too, and a frame-wide match would pass with no label at
// all. The first call is the only one an unreachable run and an up-to-date run
// reach, and the second is what a run that installs ends with.
func TestTheBorderOfARunCarriesTheVersionsTheRunChose(t *testing.T) {
	for _, c := range []struct {
		name        string
		installed   string
		unreachable bool
		want        string
		absent      string
	}{
		{name: "a run that installs", installed: "v2.0.0", want: "v2.0.0 › v2.1.0"},
		{name: "a run already on the newest", installed: "v2.1.0", want: "v2.1.0", absent: "›"},
		{name: "a run that could not read the listing", installed: "v2.0.0", unreachable: true, want: "v2.0.0", absent: "›"},
	} {
		t.Run(c.name, func(t *testing.T) {
			pl := &stubPlain{listing: listing("v2.1.0", "v2.0.0"), files: published(t, "v2.1.0", []byte("new"))}
			if c.unreachable {
				pl = &stubPlain{listErr: errors.New("connect: connection refused")}
			}
			installedAt(t, c.installed, []byte("old"), pl)
			t.Setenv("COLUMNS", "80")
			was := openProgress
			openProgress = func(w io.Writer, name string) *progress {
				return newLiveProgress(w, name, tty.NewStyle(tty.NoColour))
			}
			t.Cleanup(func() { openProgress = was })

			var out bytes.Buffer
			errOut := &syncBuffer{}
			if code := run(context.Background(), []string{"update"}, &out, errOut); code != 0 {
				t.Fatalf("the update must answer, and this run exited %d: %s", code, out.String())
			}
			border := topBorder(lastScreen(errOut.String()))
			if want := c.want + " ─┐"; !strings.HasSuffix(border, want) {
				t.Errorf("the top border must end with %q: %q", want, border)
			}
			if c.absent != "" && strings.Contains(border, c.absent) {
				t.Errorf("a run that takes nothing names one version, and the border carries %q: %q", c.absent, border)
			}
		})
	}
}

// topBorder is the first line of a frame, which is the box's top border and
// the only line the label is written on.
func topBorder(frame string) string {
	line, _, _ := strings.Cut(frame, "\n")
	return line
}
