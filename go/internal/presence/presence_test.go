package presence

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gdoc/internal/emit"
)

// The three values every scripted run uses, so a line in a failure message is
// recognisable at a glance.
const (
	version = "v2.13.0"
	docID   = "1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789"
	fixedID = "9f1cb3a7d2e45608"
)

// scene points the config directory at a fresh temporary one and puts every
// seam back the way it was afterwards, so one test's clock is never another
// test's clock.
func scene(t *testing.T) (string, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	return dir, reset(t)
}

// reset puts the seams back after the test and hands back the stderr a run
// writes its one failure line to.
func reset(t *testing.T) *bytes.Buffer {
	t.Helper()
	oldNow, oldSince, oldPID, oldRun, oldWrite := now, since, pidOf, runID, writeTo
	t.Cleanup(func() {
		now, since, pidOf, runID, writeTo = oldNow, oldSince, oldPID, oldRun, oldWrite
		said, swept = new(sync.Once), new(sync.Once)
		calls = atomic.Int64{}
	})
	said, swept = new(sync.Once), new(sync.Once)
	calls = atomic.Int64{}
	now = func() time.Time { return at(t, "2026-10-17T09:15:00Z") }
	pidOf = func() int { return 4242 }
	runID = fixedID
	return &bytes.Buffer{}
}

func at(t *testing.T, s string) time.Time {
	t.Helper()
	when, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return when
}

// written is every line in the one file the run wrote, as text.
func written(t *testing.T, dir, name string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "presence", name))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.TrimSuffix(string(b), "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// decoded is every line of the one file as the events they are.
func decoded(t *testing.T, dir, name string) []Event {
	t.Helper()
	var out []Event
	for _, line := range written(t, dir, name) {
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		out = append(out, e)
	}
	return out
}

func TestOffTouchesNothing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a 0500 directory is not a refusal on Windows")
	}
	locked := t.TempDir()
	if err := os.Chmod(locked, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	cfg := filepath.Join(locked, "cfg")
	t.Setenv("GDOC_CONFIG_DIR", cfg)
	errOut := reset(t)

	ctx := context.Background()
	r := Begin(ctx, false, version, "read", docID, false, false, errOut)
	r.End(emit.Result{OK: true})
	Held(ctx, false, version, "reply", docID, "h_ab12", "echo", now(), errOut)
	Released(false, version, "h_ab12", errOut)
	Expired(false, version, "h_cd34", errOut)
	Landed(false, version, "h_ef56", errOut)
	var none *Run
	none.End(emit.Result{OK: true})

	if _, err := os.Stat(cfg); !os.IsNotExist(err) {
		t.Fatalf("the config directory was touched: %v", err)
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr: %q", errOut.String())
	}
}

func TestTheEventCarriesOnlyItsListedFields(t *testing.T) {
	want := []string{
		"v", "at", "gdoc", "pid", "run", "call", "event", "command", "via",
		"doc", "wait", "title", "ok", "verified", "ms", "hold", "rule", "until",
	}
	var got []string
	ty := reflect.TypeOf(Event{})
	for i := 0; i < ty.NumField(); i++ {
		got = append(got, strings.Split(ty.Field(i).Tag.Get("json"), ",")[0])
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("the event's fields are\n%v\nand the specification lists\n%v", got, want)
	}
}

func TestAnEventCarriesNoWordOfTheEnvelopeButTheTitle(t *testing.T) {
	dir, errOut := scene(t)
	canaries := []string{
		"the body of the document",
		"a comment nobody answered",
		"ai? what do you think",
		"/Users/nail/notes/supplier.md",
		"someone@example.com",
		"the warning about the mark",
		"the quote \"was not found\"",
		"h_ab12_value",
	}
	r := Begin(context.Background(), true, version, "reply", docID, false, true, errOut)
	r.End(emit.Result{
		OK: false,
		Data: map[string]any{
			"title":   "Supplier register",
			"text":    canaries[0],
			"threads": []string{canaries[1], canaries[2]},
			"path":    canaries[3],
			"account": canaries[4],
		},
		Warnings: []string{canaries[5]},
		Error:    canaries[6],
		Version:  version,
	})

	whole := strings.Join(written(t, dir, "2026-10.jsonl"), "\n")
	for _, canary := range canaries {
		if strings.Contains(whole, canary) {
			t.Fatalf("the record carries %q:\n%s", canary, whole)
		}
	}
	if !strings.Contains(whole, "Supplier register") {
		t.Fatalf("the title is the one document word the record keeps:\n%s", whole)
	}
}

func TestOkIsTheEnvelopesOk(t *testing.T) {
	dir, errOut := scene(t)
	ctx := context.Background()
	Begin(ctx, true, version, "read", docID, false, false, errOut).End(emit.Result{OK: true})
	Begin(ctx, true, version, "read", docID, false, false, errOut).End(emit.Result{OK: false})

	got := decoded(t, dir, "2026-10.jsonl")
	if len(got) != 4 {
		t.Fatalf("got %d events, want 4", len(got))
	}
	if got[0].OK != nil || got[2].OK != nil {
		t.Fatalf("a start says nothing about ok: %+v", got)
	}
	if got[1].OK == nil || !*got[1].OK {
		t.Fatalf("the first end: %+v", got[1])
	}
	if got[3].OK == nil || *got[3].OK {
		t.Fatalf("the second end: %+v", got[3])
	}
}

func TestTitleIsLiftedOnlyWhenPresentAsAString(t *testing.T) {
	type named struct {
		Title string `json:"title"`
	}
	cases := []struct {
		name string
		data any
		want string
	}{
		{"a string in a map", map[string]any{"title": "Supplier register"}, "Supplier register"},
		{"a string in a struct", named{Title: "Supplier register"}, "Supplier register"},
		{"a number", map[string]any{"title": 7}, ""},
		{"none", map[string]any{"count": 3}, ""},
		{"no data at all", nil, ""},
		{"data that is no object", []string{"one"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir, errOut := scene(t)
			Begin(context.Background(), true, version, "read", docID, false, false, errOut).
				End(emit.Result{OK: true, Data: c.data})
			got := decoded(t, dir, "2026-10.jsonl")
			if got[1].Title != c.want {
				t.Fatalf("title %q, want %q", got[1].Title, c.want)
			}
		})
	}
}

func TestVerifiedIsLiftedOnlyWhenTheCallerSaysTheCommandReadsBack(t *testing.T) {
	cases := []struct {
		name      string
		readsBack bool
		data      any
		want      *bool
	}{
		{"a read-back writer that verified", true, map[string]any{"verified": true}, boolAt(true)},
		{"a read-back writer that did not", true, map[string]any{"verified": false}, boolAt(false)},
		{"a command that reads nothing back", false, map[string]any{"verified": false}, nil},
		{"a read-back writer with no verified", true, map[string]any{"count": 1}, nil},
		{"a verified that is no boolean", true, map[string]any{"verified": "yes"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir, errOut := scene(t)
			Begin(context.Background(), true, version, "reply", docID, false, c.readsBack, errOut).
				End(emit.Result{OK: true, Data: c.data})
			got := decoded(t, dir, "2026-10.jsonl")[1].Verified
			switch {
			case c.want == nil && got != nil:
				t.Fatalf("verified %v, want none", *got)
			case c.want != nil && got == nil:
				t.Fatalf("no verified, want %v", *c.want)
			case c.want != nil && *got != *c.want:
				t.Fatalf("verified %v, want %v", *got, *c.want)
			}
		})
	}
}

func boolAt(b bool) *bool { return &b }

func TestTheTitleIsCappedAtTwoHundredRunes(t *testing.T) {
	dir, errOut := scene(t)
	long := strings.Repeat("ы", 250)
	Begin(context.Background(), true, version, "read", docID, false, false, errOut).
		End(emit.Result{OK: true, Data: map[string]any{"title": long}})

	got := decoded(t, dir, "2026-10.jsonl")[1].Title
	if n := len([]rune(got)); n != 200 {
		t.Fatalf("the title is %d runes, want 200", n)
	}
	if got != strings.Repeat("ы", 200) {
		t.Fatalf("the title is not the first 200 runes: %q", got)
	}
}

func TestTheDurationIsMonotonic(t *testing.T) {
	dir, errOut := scene(t)
	// The clock the event's at comes from walks backwards an hour between the
	// start and the end. The duration must not follow it.
	clock := []time.Time{at(t, "2026-10-17T10:15:00Z"), at(t, "2026-10-17T09:15:00Z")}
	now = func() time.Time {
		when := clock[0]
		clock = clock[1:]
		return when
	}
	Begin(context.Background(), true, version, "read", docID, false, false, errOut).
		End(emit.Result{OK: true})

	ms := decoded(t, dir, "2026-10.jsonl")[1].MS
	if ms == nil {
		t.Fatal("the end carries no ms")
	}
	if *ms < 0 {
		t.Fatalf("ms is %d, and a duration taken off the wall clock is what that means", *ms)
	}
}

func TestTheMonthNamesTheFile(t *testing.T) {
	dir, errOut := scene(t)
	// A time late enough in the day that the local month and the UTC month
	// differ wherever the test runs.
	now = func() time.Time { return at(t, "2026-12-31T23:30:00Z").In(time.FixedZone("plus", 2*60*60)) }
	Released(true, version, "h_ab12", errOut)

	if _, err := os.Stat(filepath.Join(dir, "presence", "2026-12.jsonl")); err != nil {
		t.Fatalf("the UTC month names the file: %v", err)
	}
	got := decoded(t, dir, "2026-12.jsonl")
	if got[0].At != "2026-12-31T23:30:00Z" {
		t.Fatalf("at is %q, and it is written in UTC", got[0].At)
	}
}

func TestTheFileIsReadableByItsOwnerAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no such mode")
	}
	dir, errOut := scene(t)
	Released(true, version, "h_ab12", errOut)

	d, err := os.Stat(filepath.Join(dir, "presence"))
	if err != nil {
		t.Fatal(err)
	}
	if d.Mode().Perm() != 0o700 {
		t.Fatalf("the directory is %v, want 0700", d.Mode().Perm())
	}
	f, err := os.Stat(filepath.Join(dir, "presence", "2026-10.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if f.Mode().Perm() != 0o600 {
		t.Fatalf("the file is %v, want 0600", f.Mode().Perm())
	}
}

func TestASymlinkOrANonRegularFileAtThePathIsRefusedAndNothingIsWritten(t *testing.T) {
	t.Run("a symlink", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("a symlink needs a privilege on Windows")
		}
		dir, errOut := scene(t)
		elsewhere := filepath.Join(t.TempDir(), "target")
		if err := os.WriteFile(elsewhere, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "presence"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(elsewhere, filepath.Join(dir, "presence", "2026-10.jsonl")); err != nil {
			t.Fatal(err)
		}

		Released(true, version, "h_ab12", errOut)

		b, err := os.ReadFile(elsewhere)
		if err != nil {
			t.Fatal(err)
		}
		if len(b) != 0 {
			t.Fatalf("the symlink's target was written: %q", b)
		}
		if !strings.Contains(errOut.String(), "2026-10.jsonl") {
			t.Fatalf("stderr does not name the path: %q", errOut.String())
		}
	})

	t.Run("a directory", func(t *testing.T) {
		dir, errOut := scene(t)
		if err := os.MkdirAll(filepath.Join(dir, "presence", "2026-10.jsonl"), 0o700); err != nil {
			t.Fatal(err)
		}

		Released(true, version, "h_ab12", errOut)

		if !strings.Contains(errOut.String(), "2026-10.jsonl") {
			t.Fatalf("stderr does not name the path: %q", errOut.String())
		}
	})
}

func TestAShortWriteIsAFailure(t *testing.T) {
	dir, errOut := scene(t)
	writeTo = func(f *os.File, b []byte) (int, error) {
		short := b[:len(b)-1]
		n, err := f.Write(short)
		return n, err
	}
	Released(true, version, "h_ab12", errOut)

	if !strings.Contains(errOut.String(), "2026-10.jsonl") {
		t.Fatalf("a short write says nothing: %q", errOut.String())
	}
	if len(written(t, dir, "2026-10.jsonl")) != 1 {
		t.Fatal("the half line is still on disk, which is why the failure is reported")
	}
}

func TestTwoWritersNeverTearALine(t *testing.T) {
	dir, errOut := scene(t)
	var guarded sync.Mutex
	safe := &lockedBuffer{to: errOut, mu: &guarded}

	var wg sync.WaitGroup
	for w := 0; w < 16; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				Released(true, version, "h_ab12", safe)
			}
		}()
	}
	wg.Wait()

	lines := written(t, dir, "2026-10.jsonl")
	if len(lines) != 16*500 {
		t.Fatalf("got %d lines, want %d", len(lines), 16*500)
	}
	for _, line := range lines {
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("a torn line: %q: %v", line, err)
		}
	}
}

// lockedBuffer is stderr as the suite's goroutines share it, since the race
// detector counts two goroutines in one bytes.Buffer as the bug it is.
type lockedBuffer struct {
	to *bytes.Buffer
	mu *sync.Mutex
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.to.Write(p)
}

func TestAFailedWriteSaysSoOnceOnStderrAndReturnsNothing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a 0500 directory is not a refusal on Windows")
	}
	locked := t.TempDir()
	if err := os.Chmod(locked, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	t.Setenv("GDOC_CONFIG_DIR", filepath.Join(locked, "cfg"))
	errOut := reset(t)

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		Begin(ctx, true, version, "read", docID, false, false, errOut).End(emit.Result{OK: true})
		Released(true, version, "h_ab12", errOut)
	}

	if n := strings.Count(errOut.String(), "\n"); n != 1 {
		t.Fatalf("%d lines on stderr, want 1:\n%s", n, errOut.String())
	}
	if !strings.Contains(errOut.String(), "presence") {
		t.Fatalf("the line does not say what failed: %q", errOut.String())
	}
}

func TestOnlyTheCurrentAndPreviousMonthSurviveAndOnlyExactNamesAreRemoved(t *testing.T) {
	dir, errOut := scene(t)
	home := filepath.Join(dir, "presence")
	if err := os.MkdirAll(filepath.Join(home, "2026-07.jsonl"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"2025-12.jsonl", "2026-08.jsonl", "2026-09.jsonl", "notes.txt", "2026-01.jsonl.bak", "2026-9.jsonl"} {
		if err := os.WriteFile(filepath.Join(home, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	Released(true, version, "h_ab12", errOut)

	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	want := []string{"2026-01.jsonl.bak", "2026-07.jsonl", "2026-09.jsonl", "2026-10.jsonl", "2026-9.jsonl", "notes.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("the directory holds\n%v\nwant\n%v", got, want)
	}
}

func TestOffRemovesNothing(t *testing.T) {
	dir, errOut := scene(t)
	home := filepath.Join(dir, "presence")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(home, "2025-12.jsonl")
	if err := os.WriteFile(old, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	Begin(context.Background(), false, version, "read", docID, false, false, errOut).End(emit.Result{OK: true})
	Released(false, version, "h_ab12", errOut)

	if _, err := os.Stat(old); err != nil {
		t.Fatalf("with the flag off nothing is removed: %v", err)
	}
}

func TestTheFixtureIsWhatTheWriterWrites(t *testing.T) {
	dir, errOut := scene(t)
	clock := at(t, "2026-10-17T09:15:00Z")
	now = func() time.Time {
		when := clock
		clock = clock.Add(time.Second)
		return when
	}
	elapsed := []time.Duration{540 * time.Second, 1200 * time.Millisecond, 0}
	since = func(time.Time) time.Duration {
		d := elapsed[0]
		elapsed = elapsed[1:]
		return d
	}

	ctx := context.Background()
	chat := WithChat(ctx)

	watch := Begin(ctx, true, version, "comments", docID, true, false, errOut)
	watch.End(emit.Result{OK: true, Data: map[string]any{"title": "Supplier register"}, Version: version})

	write := Begin(chat, true, version, "reply", docID, false, true, errOut)
	write.End(emit.Result{OK: true, Data: map[string]any{"title": "Supplier register", "verified": true}, Version: version})

	Held(chat, true, version, "reply", docID, "h_ab12", "echo", at(t, "2026-10-17T09:45:00Z"), errOut)
	Released(true, version, "h_ab12", errOut)
	Expired(true, version, "h_cd34", errOut)
	Landed(true, version, "h_ef56", errOut)

	upgrade := Begin(ctx, true, version, "update", "", false, false, errOut)
	upgrade.End(emit.Result{OK: false, Error: "nothing newer than v2.13.0 is published", Version: version})

	if errOut.Len() != 0 {
		t.Fatalf("stderr: %q", errOut.String())
	}
	got, err := os.ReadFile(filepath.Join(dir, "presence", "2026-10.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "events-v1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("the writer wrote\n%s\nand testdata/events-v1.jsonl holds\n%s", got, want)
	}
}
