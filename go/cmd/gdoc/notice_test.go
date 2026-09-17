// The daily release check: when help asks GitHub, when it does not, what it
// writes down, and the one line a person reads.
//
// Nothing here reaches GitHub. The reach is a stub that answers a literal
// listing, records every URL it was handed and records the deadline the
// context carried, so a test that says nothing was fetched can prove it and a
// test about the ceiling can measure it.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"gdoc/internal/config"
	"gdoc/internal/guard"
	"gdoc/internal/lastcheck"
)

// checkPlain is the reach the check is handed. It answers the listing and
// refuses every download, because a check reads one page and installs nothing.
type checkPlain struct {
	listing   string
	err       error
	got       []string
	asked     []time.Time
	deadlines []time.Time
}

func (c *checkPlain) GetJSON(ctx context.Context, rawURL string, into any) error {
	c.got = append(c.got, rawURL)
	c.asked = append(c.asked, time.Now())
	if d, ok := ctx.Deadline(); ok {
		c.deadlines = append(c.deadlines, d)
	}
	if c.err != nil {
		return c.err
	}
	return json.Unmarshal([]byte(c.listing), into)
}

func (c *checkPlain) GetBytes(_ context.Context, rawURL string, _ int64) ([]byte, error) {
	return nil, fmt.Errorf("a check downloads nothing, and this one asked for %s", rawURL)
}

func (c *checkPlain) Warnings() []string { return nil }

// refusingPlain fails the test the moment anything reaches it. It is what a
// run that must not touch GitHub is handed.
type refusingPlain struct{ t *testing.T }

func (r *refusingPlain) GetJSON(_ context.Context, rawURL string, _ any) error {
	r.t.Errorf("nothing in this run may reach %s", rawURL)
	return errors.New("this run reaches nothing")
}

func (r *refusingPlain) GetBytes(_ context.Context, rawURL string, _ int64) ([]byte, error) {
	r.t.Errorf("nothing in this run may reach %s", rawURL)
	return nil, errors.New("this run reaches nothing")
}

func (r *refusingPlain) Warnings() []string { return nil }

// checking points a run at a config dir of its own, gives it a version, and
// hands it the reach. It answers the path the stamp lives at.
func checking(t *testing.T, ver string, pl plain) string {
	t.Helper()
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	oldVersion, oldOpen := version, openPlain
	version = ver
	openPlain = func(*guard.Policy) plain { return pl }
	t.Cleanup(func() { version, openPlain = oldVersion, oldOpen })
	path, err := config.LastCheckPath()
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// stampedAt writes a stamp dated age ago, which is how a test says the last
// check was yesterday without waiting a day.
func stampedAt(t *testing.T, path string, age time.Duration, s lastcheck.Stamp) {
	t.Helper()
	s.CheckedAt = time.Now().Add(-age)
	if err := lastcheck.Write(path, s); err != nil {
		t.Fatal(err)
	}
}

// stampOnDisk reads the file back as the next run would.
func stampOnDisk(t *testing.T, path string) lastcheck.Stamp {
	t.Helper()
	s, _, why := lastcheck.Read(path, time.Now())
	if s.CheckedAt.IsZero() {
		t.Fatalf("no stamp was written at %s: %s", path, why)
	}
	return s
}

// updateFactsOf is the update block out of the object, as a skill reads it.
func updateFactsOf(t *testing.T, got map[string]any) map[string]any {
	t.Helper()
	facts, ok := dataOf(t, got)["update"].(map[string]any)
	if !ok {
		t.Fatalf("data.update is not an object: %v", got["data"])
	}
	return facts
}

// No record of an earlier check, so help asks once, writes down what it heard,
// prints the four facts a skill reads and opens the words a person reads with
// one line naming the release and the command that takes it.
func TestAStaleStampMakesHelpAskOnce(t *testing.T) {
	pl := &checkPlain{listing: listing("v2.3.4", "v2.3.0", "v2.2.0")}
	path := checking(t, "v2.2.0", pl)

	got, prose, code := runHelp(t, "help")
	if code != 0 || got["ok"] != true {
		t.Fatalf("help is an answer: %v (exit %d)", got, code)
	}
	if len(pl.got) != 1 {
		t.Fatalf("a stale stamp is one request and this run made %d: %v", len(pl.got), pl.got)
	}
	facts := updateFactsOf(t, got)
	if facts["installed"] != "v2.2.0" || facts["latest_stable"] != "v2.3.0" || facts["latest_nightly"] != "v2.3.4" {
		t.Errorf("the object carries what is here and what is published: %v", facts)
	}
	if _, ok := facts["checked_at"].(string); !ok {
		t.Errorf("the object says when gdoc asked: %v", facts)
	}
	if _, ok := facts["error"]; ok {
		t.Errorf("the check answered, so it names no cause: %v", facts)
	}
	s := stampOnDisk(t, path)
	if s.LatestStable != "v2.3.0" || s.LatestNightly != "v2.3.4" || s.Error != "" {
		t.Errorf("the stamp holds what was heard: %+v", s)
	}
	want := "gdoc v2.3.0 is published and this is v2.2.0. `gdoc update` installs it."
	if !strings.HasPrefix(prose, want) {
		t.Errorf("the words a person reads open with the notice:\nwant prefix %q\n got %q", want, prose)
	}
}

// A check younger than the interval is the answer. Nothing is fetched, and
// the object still carries the facts, out of the file.
func TestAFreshStampMakesNoRequest(t *testing.T) {
	path := checking(t, "v2.2.0", &refusingPlain{t: t})
	stampedAt(t, path, 23*time.Hour, lastcheck.Stamp{LatestStable: "v2.3.0", LatestNightly: "v2.3.4"})

	got, prose, code := runHelp(t, "help")
	if code != 0 || got["ok"] != true {
		t.Fatalf("help is an answer: %v (exit %d)", got, code)
	}
	facts := updateFactsOf(t, got)
	if facts["latest_stable"] != "v2.3.0" || facts["latest_nightly"] != "v2.3.4" {
		t.Errorf("the facts come out of the stamp: %v", facts)
	}
	if !strings.Contains(prose, "gdoc v2.3.0 is published") {
		t.Errorf("a stamped release is still worth one line: %q", prose)
	}
}

// A build from a checkout names no release, so there is nothing to compare and
// nothing to ask. It reaches GitHub never, writes no stamp, and says nothing.
func TestACheckoutBuildNeverChecks(t *testing.T) {
	path := checking(t, "dev", &refusingPlain{t: t})

	got, prose, code := runHelp(t, "help")
	if code != 0 || got["ok"] != true {
		t.Fatalf("help is an answer: %v (exit %d)", got, code)
	}
	if _, ok := dataOf(t, got)["update"]; ok {
		t.Errorf("a checkout build has nothing to compare, so it claims no update facts: %v", got["data"])
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("a checkout build writes no stamp, and %s is there: %v", path, err)
	}
	if strings.Contains(prose, "is published") {
		t.Errorf("a checkout build says nothing about a release: %q", prose)
	}
}

// The ceiling is two seconds, so a machine that cannot reach GitHub costs a
// help that much and no more.
func TestTheCheckIsBoundedByTwoSeconds(t *testing.T) {
	pl := &checkPlain{listing: listing("v2.3.0")}
	checking(t, "v2.2.0", pl)

	if _, _, code := runHelp(t, "help"); code != 0 {
		t.Fatalf("help is an answer: exit %d", code)
	}
	if len(pl.deadlines) != 1 {
		t.Fatalf("the check makes one request under a deadline, and made %d: %v", len(pl.deadlines), pl.deadlines)
	}
	// Measured from the moment the request was made, because the deadline is
	// set just before it: two seconds is the whole of what is left to wait.
	if d := pl.deadlines[0].Sub(pl.asked[0]); d > 2*time.Second || d <= 0 {
		t.Errorf("the check is bounded by two seconds and this one had %v", d)
	}
}

// GitHub not answering is a fact about today. Help still answers, the cause
// reaches a warning and the stamp, and the next run in the same day asks
// nothing: that is what bounds the cost of a network that refuses GitHub.
func TestAnUnreachableGitHubIsStampedAndHelpStillAnswers(t *testing.T) {
	for _, cause := range []string{
		"context deadline exceeded",
		"connect: connection refused",
		"answered 503: Service Unavailable",
		"answered 403: API rate limit exceeded for 1.2.3.4",
	} {
		t.Run(cause, func(t *testing.T) {
			pl := &checkPlain{err: fmt.Errorf("the request to %s failed: %s", releasesURL, cause)}
			path := checking(t, "v2.2.0", pl)
			stampedAt(t, path, 25*time.Hour, lastcheck.Stamp{LatestStable: "v2.2.0", LatestNightly: "v2.2.4"})

			got, _, code := runHelp(t, "help")
			if code != 0 || got["ok"] != true {
				t.Fatalf("%q must be an answer: %v (exit %d)", cause, got, code)
			}
			if !hasWarning(warningsOf(t, got), cause) {
				t.Errorf("%q must reach a warning: %v", cause, got["warnings"])
			}
			facts := updateFactsOf(t, got)
			if msg, _ := facts["error"].(string); !strings.Contains(msg, cause) {
				t.Errorf("the object names the cause: %v", facts)
			}
			if facts["latest_stable"] != "v2.2.0" {
				t.Errorf("a failed check keeps what it heard before: %v", facts)
			}
			s := stampOnDisk(t, path)
			if !strings.Contains(s.Error, cause) || s.LatestStable != "v2.2.0" {
				t.Errorf("the failure is written down with what was heard before: %+v", s)
			}

			if _, _, code := runHelp(t, "help"); code != 0 {
				t.Fatalf("the second help must answer too: exit %d", code)
			}
			if len(pl.got) != 1 {
				t.Errorf("a stamped failure costs one request a day, and this cost %d: %v", len(pl.got), pl.got)
			}
		})
	}
}

// A config dir nothing can be written into is one warning and not a failure.
// The check still happened, so it is not tried a second time in this run.
func TestAStampThatCannotBeWrittenIsOneWarning(t *testing.T) {
	pl := &checkPlain{listing: listing("v2.3.0")}
	path := checking(t, "v2.2.0", pl)
	dir, err := config.Dir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	got, _, code := runHelp(t, "help")
	if code != 0 || got["ok"] != true {
		t.Fatalf("an unwritable stamp is not a failed help: %v (exit %d)", got, code)
	}
	if ws := warningsOf(t, got); len(ws) != 1 || !strings.Contains(ws[0], path) {
		t.Errorf("one warning, naming the file it could not write: %v", ws)
	}
	if len(pl.got) != 1 {
		t.Errorf("the check ran once and is not tried again in the same run: %v", pl.got)
	}
	if facts := updateFactsOf(t, got); facts["latest_stable"] != "v2.3.0" {
		t.Errorf("what was heard is still reported: %v", facts)
	}
}

// The policy the check opens is the fifth grant and nothing else, judged the
// way TestTheUpdateRunReachesTheReleasesAndNothingElse judges the update's.
func TestTheHelpCheckOpensThePolicyTheUpdateOpens(t *testing.T) {
	pl := &checkPlain{listing: listing("v2.3.0")}
	checking(t, "v2.2.0", pl)
	var opened *guard.Policy
	openPlain = func(p *guard.Policy) plain { opened = p; return pl }

	if _, _, code := runHelp(t, "help"); code != 0 {
		t.Fatalf("help is an answer: exit %d", code)
	}
	if opened == nil {
		t.Fatal("the check opened no policy")
	}
	judge := func(method, rawURL string) error {
		u, err := url.Parse(rawURL)
		if err != nil {
			t.Fatal(err)
		}
		return opened.Judge(method, u, nil)
	}
	if err := judge("GET", "https://api.github.com/repos/nhusnullin/gdoc/releases"); err != nil {
		t.Errorf("the listing is the one call the check makes: %v", err)
	}
	for _, refused := range [][2]string{
		{"GET", "https://api.github.com/repos/someone/else/releases"},
		{"POST", "https://api.github.com/repos/nhusnullin/gdoc/releases"},
		{"GET", "https://docs.googleapis.com/v1/documents/1AbCdEfGhIjKlMnOpQrStUvWxYz012345"},
		{"GET", "https://www.googleapis.com/drive/v3/files/1AbCdEfGhIjKlMnOpQrStUvWxYz012345"},
	} {
		if err := judge(refused[0], refused[1]); err == nil {
			t.Errorf("%s %s must be refused by the check's policy", refused[0], refused[1])
		}
	}
}

// A release across a major boundary is named with the flag that takes it,
// because a major is where something a person relies on may have gone. And a
// binary that is already the newest stable is told nothing at all.
func TestTheLineNamesMajorForAMajor(t *testing.T) {
	path := checking(t, "v2.2.0", &refusingPlain{t: t})
	stampedAt(t, path, time.Hour, lastcheck.Stamp{LatestStable: "v3.0.0", LatestNightly: "v3.0.0"})

	_, prose, code := runHelp(t, "help")
	if code != 0 {
		t.Fatalf("help is an answer: exit %d", code)
	}
	want := "gdoc v3.0.0 is published and this is v2.2.0. It is a major release: `gdoc update --major` installs it."
	if !strings.HasPrefix(prose, want) {
		t.Errorf("a major is named with the flag that takes it:\nwant prefix %q\n got %q", want, prose)
	}

	path = checking(t, "v2.2.0", &refusingPlain{t: t})
	stampedAt(t, path, time.Hour, lastcheck.Stamp{LatestStable: "v2.2.0", LatestNightly: "v2.2.0"})
	_, prose, code = runHelp(t, "help")
	if code != 0 {
		t.Fatalf("help is an answer: exit %d", code)
	}
	if strings.Contains(prose, "is published") {
		t.Errorf("there is nothing newer, so there is no line: %q", prose)
	}
}

// Every command but help is exactly as offline from GitHub as it was. A read
// with a stale stamp and a reach that fails the test makes no request at all.
func TestReadNeverReachesTheCheck(t *testing.T) {
	path := checking(t, "v2.2.0", &refusingPlain{t: t})
	stampedAt(t, path, 30*24*time.Hour, lastcheck.Stamp{LatestStable: "v2.3.0"})
	stubSession(t, docsAndComments(t))

	got, code := runJSON(t, "read", fixtureDocID)
	if code != 0 || got["ok"] != true {
		t.Fatalf("read: %v (exit %d)", got, code)
	}
	if _, ok := dataOf(t, got)["update"]; ok {
		t.Errorf("read says nothing about releases: %v", got["data"])
	}
	before := stampOnDisk(t, path)
	if before.LatestStable != "v2.3.0" {
		t.Errorf("read touched the stamp: %+v", before)
	}
}
