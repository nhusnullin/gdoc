package comments

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func at(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestCursorRoundTripsThroughItsString(t *testing.T) {
	c := &Cursor{At: at("2026-09-06T11:00:00Z")}
	back, err := ParseCursor(c.String())
	if err != nil {
		t.Fatal(err)
	}
	if !back.At.Equal(c.At) {
		t.Errorf("round trip = %v, want %v", back.At, c.At)
	}
}

func TestCursorIsBase64URLWithoutPadding(t *testing.T) {
	// The cursor travels through a JSON field and back through an argument, so
	// it carries no character a shell or a URL treats as its own.
	s := (&Cursor{At: at("2026-09-06T11:00:00Z")}).String()
	if strings.ContainsAny(s, "+/=") {
		t.Errorf("cursor = %q, and + / = are not base64url", s)
	}
	body, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("the cursor does not decode as base64url: %v", err)
	}
	if !strings.Contains(string(body), `"v":1`) {
		t.Errorf("the cursor body %s does not carry its version", body)
	}
	if !strings.Contains(string(body), `"2026-09-06T11:00:00Z"`) {
		t.Errorf("the cursor body %s does not carry the instant in UTC", body)
	}
}

func TestCursorStringIsUTCWhateverZoneItWasGiven(t *testing.T) {
	zone := time.FixedZone("CET", 2*60*60)
	local := &Cursor{At: at("2026-09-06T11:00:00Z").In(zone)}
	if local.String() != (&Cursor{At: at("2026-09-06T11:00:00Z")}).String() {
		t.Errorf("the same instant in two zones made two cursors: %q", local.String())
	}
}

func TestParseCursorRefusesWhatItCannotRead(t *testing.T) {
	v2 := base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"t":"2026-09-06T11:00:00Z"}`))
	noTime := base64.RawURLEncoding.EncodeToString([]byte(`{"v":1}`))
	badTime := base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"t":"yesterday"}`))
	notJSON := base64.RawURLEncoding.EncodeToString([]byte(`not json`))
	for name, in := range map[string]string{
		"a later version":  v2,
		"no instant":       noTime,
		"an unread time":   badTime,
		"not JSON inside":  notJSON,
		"not base64":       "not base64!!",
		"the empty string": "",
	} {
		// A cursor the binary cannot decode is an error naming the problem,
		// never a silent read from the beginning: that would answer a poll with
		// every thread in the document and read as news.
		if _, err := ParseCursor(in); err == nil {
			t.Errorf("ParseCursor accepted %s (%q)", name, in)
		}
	}
}

func TestNextCursorPicksTheNewestAcrossCommentsAndReplies(t *testing.T) {
	threads := []Thread{
		{Modified: "2026-09-06T10:30:00Z", Replies: []Reply{{Created: "2026-09-06T10:45:00Z"}}},
		{Modified: "2026-09-06T09:00:00Z"},
	}
	got := NextCursor(nil, threads)
	if got == nil {
		t.Fatal("NextCursor returned nothing for threads it could read")
	}
	if want := at("2026-09-06T10:45:00Z"); !got.At.Equal(want) {
		t.Errorf("cursor = %v, want the reply's time %v", got.At, want)
	}
}

func TestNextCursorFallsBackToThePreviousOne(t *testing.T) {
	prev := &Cursor{At: at("2026-09-06T11:00:00Z")}
	if got := NextCursor(prev, nil); got != prev {
		t.Errorf("a run that saw nothing returned %v, want the cursor it was given", got)
	}
	// And a run that saw only older activity keeps the newer cursor: a cursor
	// that goes backwards makes the next poll report what this one just did.
	older := []Thread{{Modified: "2026-09-05T08:00:00Z"}}
	got := NextCursor(prev, older)
	if got == nil || !got.At.Equal(prev.At) {
		t.Errorf("cursor = %v, want it to stay at %v", got, prev.At)
	}
}

func TestNextCursorWithNothingAtAllIsNothing(t *testing.T) {
	if got := NextCursor(nil, nil); got != nil {
		t.Errorf("NextCursor(nil, nil) = %v, want nothing to print", got)
	}
}

func TestNextCursorStepsOverATimeItCannotRead(t *testing.T) {
	threads := []Thread{{Modified: "not a time", Replies: []Reply{{Created: "2026-09-06T10:45:00Z"}}}}
	got := NextCursor(nil, threads)
	if got == nil || !got.At.Equal(at("2026-09-06T10:45:00Z")) {
		t.Errorf("cursor = %v, want the one time that read", got)
	}
}

// Padded base64url is what an encoder that does not know the raw form writes.
// Reading it costs a line, and the branch that does has a case rather than a
// comment saying why it is there.
func TestPaddedBase64IsRead(t *testing.T) {
	at := time.Date(2026, 9, 6, 11, 30, 0, 0, time.UTC)
	raw := []byte(`{"v":1,"t":"` + at.Format(time.RFC3339) + `"}`)
	padded := base64.URLEncoding.EncodeToString(raw)
	if !strings.HasSuffix(padded, "=") {
		t.Fatalf("the fixture %q carries no padding, so it tests nothing", padded)
	}
	got, err := ParseCursor(padded)
	if err != nil {
		t.Fatalf("ParseCursor(%q) = %v", padded, err)
	}
	if !got.At.Equal(at) {
		t.Errorf("ParseCursor() = %v, want %v", got.At, at)
	}
}

// TestCursorKeepsTheMillisecondsDriveSent is the one that decides whether a
// poll ever finishes. Drive writes modifiedTime with milliseconds, so a cursor
// that formats to whole seconds hands the next poll a floor up to 999 ms below
// the activity it just reported. Drive then returns that thread again, and
// NextCursor truncates it back to the same floor, so the thread is news on
// every poll for ever.
func TestCursorKeepsTheMillisecondsDriveSent(t *testing.T) {
	c := &Cursor{At: at("2026-09-06T11:00:00.789Z")}

	back, err := ParseCursor(c.String())
	if err != nil {
		t.Fatal(err)
	}
	if !back.At.Equal(c.At) {
		t.Errorf("round trip = %v, want %v", back.At, c.At)
	}
	if got, want := c.since(), "2026-09-06T11:00:00.789Z"; got != want {
		t.Errorf("since() = %q, want %q", got, want)
	}
}

// TestNextCursorDoesNotReReportTheThreadItJustSaw is the same fact from the
// caller's side: the cursor this run emits must not select the thread this run
// already reported.
func TestNextCursorDoesNotReReportTheThreadItJustSaw(t *testing.T) {
	threads := []Thread{{Modified: "2026-09-06T11:00:00.789Z"}}

	emitted := NextCursor(nil, threads).String()
	next, err := ParseCursor(emitted)
	if err != nil {
		t.Fatal(err)
	}
	if next.At.Before(at("2026-09-06T11:00:00.789Z")) {
		t.Errorf("the emitted cursor is %v, which is before the activity it saw", next.At)
	}
}
