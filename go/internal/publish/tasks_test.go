package publish

import (
	"context"
	"strings"
	"testing"

	"gdoc/internal/docs"
)

// theTasks is the two task lines published-tasks.json came from, with the
// owner already resolved. The note's list holds a third item that is no task,
// so every test below is over a mixed list.
//
// Literals, every one of them, because what is asserted is what a person
// really wrote in a note.
func theTasks() []Task {
	return []Task{
		{
			Words:  "Ask the bank",
			Owner:  "anna@example.com",
			Due:    "2026-01-02",
			Tokens: " @anna@example.com 2026-01-02",
		},
		{Words: "Confirm the scheme", Done: true},
	}
}

// readOf is one fixture decoded the way a read-back decodes it.
func readOf(t *testing.T, name string) *docs.Document {
	t.Helper()
	d, err := docs.Parse(fixture(t, name))
	if err != nil {
		t.Fatalf("the fixture %s does not read as a document: %v", name, err)
	}
	return d
}

// kinds is the request kind of each request in a batch, in order.
func kinds(reqs []map[string]any) []string {
	out := make([]string, 0, len(reqs))
	for _, r := range reqs {
		for k := range r {
			out = append(out, k)
		}
	}
	return out
}

// fields is one request's own object, whatever kind it is.
func fields(t *testing.T, req map[string]any) map[string]any {
	t.Helper()
	for _, v := range req {
		got, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("the request carries no object: %v", req)
		}
		return got
	}
	t.Fatalf("the request names no kind: %v", req)
	return nil
}

// spanOf is a request's range as two numbers.
func spanOf(t *testing.T, req map[string]any) (int, int) {
	t.Helper()
	rng, ok := fields(t, req)["range"].(map[string]any)
	if !ok {
		t.Fatalf("the request carries no range: %v", req)
	}
	return rng["startIndex"].(int), rng["endIndex"].(int)
}

// locationOf is where an insert puts what it inserts.
func locationOf(t *testing.T, req map[string]any) int {
	t.Helper()
	loc, ok := fields(t, req)["location"].(map[string]any)
	if !ok {
		t.Fatalf("the insert names no location: %v", req)
	}
	return loc["index"].(int)
}

func requestsFor(t *testing.T, read string, want []Task) []map[string]any {
	t.Helper()
	reqs, err := taskRequests(readOf(t, read).Tabs[0], want)
	if err != nil {
		t.Fatalf("the batch could not be built: %v", err)
	}
	return reqs
}

// The order is the whole of the index arithmetic. Bullets move no character,
// so they go first at the read's own indexes; the text that follows walks
// backwards, so an index still to be used names the character the read named.
func TestTheBulletsComeFirstAndTheTextFromTheLastTaskToTheFirst(t *testing.T) {
	reqs := requestsFor(t, "published-tasks.json", theTasks())

	want := []string{
		"deleteParagraphBullets",
		"createParagraphBullets",
		// The done task, which is the last one and carries no token.
		"deleteContentRange",
		// The first task, from its date back to its ballot character.
		"deleteContentRange", "insertDate",
		"deleteContentRange", "insertPerson",
		"deleteContentRange",
	}
	got := kinds(reqs)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("the batch is\n  %v\nwant\n  %v", got, want)
	}
	// Every index after the first two requests descends, which is what says
	// nothing in the batch moves what a later request names.
	last := -1
	for i := 2; i < len(reqs); i++ {
		at := locationOrStart(t, reqs[i])
		if last >= 0 && at > last {
			t.Errorf("request %d names index %d after a request at %d, so the batch walks forward and the indexes it carries have moved", i, at, last)
		}
		last = at
	}
}

// locationOrStart is where a request begins, whichever of the two it names.
func locationOrStart(t *testing.T, req map[string]any) int {
	t.Helper()
	if _, ok := fields(t, req)["location"]; ok {
		return locationOf(t, req)
	}
	start, _ := spanOf(t, req)
	return start
}

func TestEachTokenBecomesItsChipWhereItStood(t *testing.T) {
	reqs := requestsFor(t, "published-tasks.json", theTasks())

	// "☐ Ask the bank @anna@example.com 2026-01-02" opens at 66, so the owner
	// stands at [81,98) and the day at [99,109).
	for _, c := range []struct {
		name       string
		del, chip  int
		start, end int
	}{
		{"the date", 3, 4, 99, 109},
		{"the owner", 5, 6, 81, 98},
	} {
		start, end := spanOf(t, reqs[c.del])
		if start != c.start || end != c.end {
			t.Errorf("%s is taken out of [%d,%d), want [%d,%d)", c.name, start, end, c.start, c.end)
		}
		if at := locationOf(t, reqs[c.chip]); at != c.start {
			t.Errorf("%s chip goes in at %d, want %d, where the token stood", c.name, at, c.start)
		}
	}
	if email := personEmail(t, reqs[6]); email != "anna@example.com" {
		t.Errorf("the person chip names %q, want the address the note resolved to", email)
	}
	// The ballot character and the space after it, off the front of each task.
	for _, c := range []struct {
		name       string
		req        int
		start, end int
	}{
		{"the done task", 2, 110, 112},
		{"the first task", 7, 66, 68},
	} {
		start, end := spanOf(t, reqs[c.req])
		if start != c.start || end != c.end {
			t.Errorf("%s loses [%d,%d), want [%d,%d), which is the ballot character and its space", c.name, start, end, c.start, c.end)
		}
	}
}

func personEmail(t *testing.T, req map[string]any) string {
	t.Helper()
	props, ok := fields(t, req)["personProperties"].(map[string]any)
	if !ok {
		t.Fatalf("the person chip carries no properties: %v", req)
	}
	return props["email"].(string)
}

// A checklist request over part of a list turns the whole list, so the bullets
// come off the task paragraphs before the boxes go on them. Measured
// 2026-11-05, TestLiveChecklistProbeOverBullets.
func TestTheBulletsComeOffBeforeTheChecklistGoesOn(t *testing.T) {
	reqs := requestsFor(t, "published-tasks.json", theTasks())

	if _, ok := reqs[0]["deleteParagraphBullets"]; !ok {
		t.Fatalf("the first request is %v, want the bullets taken off", kinds(reqs[:1]))
	}
	if _, ok := reqs[1]["createParagraphBullets"]; !ok {
		t.Fatalf("the second request is %v, want the boxes put on", kinds(reqs[1:2]))
	}
	offStart, offEnd := spanOf(t, reqs[0])
	onStart, onEnd := spanOf(t, reqs[1])
	if offStart != onStart || offEnd != onEnd {
		t.Errorf("the bullets come off [%d,%d) and the boxes go on [%d,%d), and the two are one stretch", offStart, offEnd, onStart, onEnd)
	}
	if preset := fields(t, reqs[1])["bulletPreset"]; preset != "BULLET_CHECKBOX" {
		t.Errorf("the preset is %v, want the one that draws a box", preset)
	}
}

// The third item of the list is no task. The stretch the batch names stops
// before it, so it stays the bullet the note wrote.
func TestAMixedListKeepsItsPlainItems(t *testing.T) {
	reqs := requestsFor(t, "published-tasks.json", theTasks())

	start, end := spanOf(t, reqs[1])
	if start != 66 || end != 131 {
		t.Errorf("the checklist covers [%d,%d), want [66,131), which is the two tasks and not the plain item at 131", start, end)
	}
	for i, r := range reqs {
		s, e := requestSpan(t, r)
		if s >= 131 || e > 131 {
			t.Errorf("request %d (%v) reaches [%d,%d), and the plain item begins at 131", i, kinds(reqs[i:i+1]), s, e)
		}
	}
}

// requestSpan is what a request touches, whichever way it names it.
func requestSpan(t *testing.T, req map[string]any) (int, int) {
	t.Helper()
	if _, ok := fields(t, req)["location"]; ok {
		at := locationOf(t, req)
		return at, at
	}
	return spanOf(t, req)
}

// A task's date is a day, and a chip takes a moment. Noon UTC is the same
// calendar day everywhere a colleague reads it.
func TestADateIsNoonUTCOnItsDay(t *testing.T) {
	reqs := requestsFor(t, "published-tasks.json", theTasks())

	props, ok := fields(t, reqs[4])["dateElementProperties"].(map[string]any)
	if !ok {
		t.Fatalf("the date chip carries no properties: %v", reqs[4])
	}
	if props["timestamp"] != "2026-01-02T12:00:00Z" {
		t.Errorf("the date chip carries %v, want noon UTC on the day the note named", props["timestamp"])
	}
	if props["dateFormat"] != "DATE_FORMAT_MONTH_DAY_YEAR_ABBREVIATED" {
		t.Errorf("the date chip is formatted %v, want the form the probe measured", props["dateFormat"])
	}
}

// A document that is not the note's own render is a document this batch would
// write the wrong indexes into. So the count and the words are checked first,
// and a disagreement sends nothing at all.
func TestADifferentTaskCountSendsNothing(t *testing.T) {
	tab := readOf(t, "published-tasks.json").Tabs[0]

	for _, c := range []struct {
		name string
		want []Task
	}{
		{"one task fewer", theTasks()[:1]},
		{"words that do not match", []Task{
			{Words: "Ask the baker", Owner: "anna@example.com", Due: "2026-01-02",
				Tokens: " @anna@example.com 2026-01-02"},
			{Words: "Confirm the scheme", Done: true},
		}},
	} {
		reqs, err := taskRequests(tab, c.want)
		if err == nil {
			t.Errorf("%s built a batch of %d requests, want nothing sent", c.name, len(reqs))
		}
		if reqs != nil {
			t.Errorf("%s answered requests beside its refusal: %v", c.name, kinds(reqs))
		}
	}
}

// A chip that is not in the read is a fact about a document that exists, so it
// is a warning and a check that did not hold.
func TestAMissingChipIsTasksFalseAndAWarning(t *testing.T) {
	f := scriptWithTasks(t, "published-tasks.json", "published-tasks-no-person.json")

	rep, err := Run(context.Background(), f, optionsWithTasks())

	if err != nil {
		t.Fatal(err)
	}
	if rep.Checks.Tasks {
		t.Error("checks.tasks is true on a document that came back without the name chip")
	}
	if rep.Verified {
		t.Error("the publish is verified on a document whose tasks did not land")
	}
	if !strings.Contains(strings.Join(rep.Warnings, " "), "name chips") {
		t.Errorf("no warning names the chip that is missing: %v", rep.Warnings)
	}
	if rep.DocumentID != testDocID {
		t.Error("the document is reported whatever the tasks did, because it is there")
	}
}

// TestATabBeforeATokenIsNotATaskThatDidNotLand is the separator the batch
// leaves behind. A note may write the tab internal/tasks splits a token off
// on, and the chip goes in where the token stood, so the tab stays at the end
// of the line the read brings back. Taking only the space off called that
// landed task a task whose words had changed, and the warning beside it said
// the boxes never arrived, which was false twice over.
func TestATabBeforeATokenIsNotATaskThatDidNotLand(t *testing.T) {
	want := []Task{{
		Words:  "Ask the bank",
		Owner:  "anna@example.com",
		Tokens: "\t@anna@example.com",
	}}
	// The paragraph as the read brings it back: the words with the tab still
	// on them, and the chip where the token stood.
	landed := docs.Tab{Body: []docs.Block{{Paragraph: &docs.Paragraph{
		Bullet: &docs.Bullet{Checklist: true, ListID: "kix.check"},
		Runs: []docs.Run{
			{Kind: docs.KindText, Text: "Ask the bank\t", StartIndex: 1, EndIndex: 14},
			{Kind: docs.KindPerson, StartIndex: 14, EndIndex: 15},
			{Kind: docs.KindText, Text: "\n", StartIndex: 15, EndIndex: 16},
		},
	}}}}

	if warns := tasksLanded(landed, want); len(warns) > 0 {
		t.Errorf("a task whose token stood after a tab reads as one that did not land: %v", warns)
	}
}

// The note everybody publishes today has no task line. It costs no request of
// any kind, and the check is true because nothing was asked for.
func TestANoteWithNoTaskSendsNothingNew(t *testing.T) {
	f := script(t, "published-contents.json")

	rep, err := Run(context.Background(), f, options())

	if err != nil {
		t.Fatal(err)
	}
	if !rep.Checks.Tasks {
		t.Error("checks.tasks is false on a note that named no task")
	}
	if !rep.Verified {
		t.Errorf("a note with no task is not verified: %+v %v", rep.Checks, rep.Warnings)
	}
	// The create, the read, the export, the marker batch and the read that
	// confirms it. Nothing else.
	if len(f.calls) != 5 {
		t.Errorf("the run made %d requests, want the five a publish with no task makes: %v", len(f.calls), urls(f.calls))
	}
	for _, c := range f.calls {
		if strings.HasSuffix(c.url, ":batchUpdate") && !strings.Contains(string(c.body), "createNamedRange") {
			t.Errorf("a note with no task sent a batch that is not the marker: %s", c.body)
		}
	}
}

func urls(calls []call) []string {
	out := make([]string, 0, len(calls))
	for _, c := range calls {
		out = append(out, c.method+" "+c.url)
	}
	return out
}

// Verified is the four routes together, and the tasks are one of them, so it
// is computed after the batch rather than before it.
func TestVerifiedIsComputedAfterTheTasks(t *testing.T) {
	f := scriptWithTasks(t, "published-tasks.json", "published-tasks-done.json")

	rep, err := Run(context.Background(), f, optionsWithTasks())

	if err != nil {
		t.Fatal(err)
	}
	if !rep.Checks.Tasks {
		t.Fatalf("checks.tasks is false on a document whose tasks landed: %v", rep.Warnings)
	}
	if !rep.Verified {
		t.Errorf("the publish is not verified: %+v %v", rep.Checks, rep.Warnings)
	}
	bad := scriptWithTasks(t, "published-tasks.json", "published-tasks-no-person.json")
	if rep, _ := Run(context.Background(), bad, optionsWithTasks()); rep.Verified {
		t.Error("a publish whose tasks did not land is verified, so verified was computed before them")
	}
}

// The batch moves every index after the cover, so the marker is written from
// the read taken after it rather than from the one the batch was built on.
func TestMarkIsHandedTheReadAfterTheTasks(t *testing.T) {
	f := scriptWithTasks(t, "published-tasks.json", "published-tasks-done.json")

	rep, err := Run(context.Background(), f, optionsWithTasks())

	if err != nil {
		t.Fatal(err)
	}
	var batches []call
	for _, c := range f.calls {
		if strings.HasSuffix(c.url, ":batchUpdate") {
			batches = append(batches, c)
		}
	}
	if len(batches) != 2 {
		t.Fatalf("the run sent %d batches, want the tasks and then the marker: %v", len(batches), urls(f.calls))
	}
	if strings.Contains(string(batches[0].body), "createNamedRange") {
		t.Error("the marker went out before the tasks, so it names indexes the tasks then moved")
	}
	if !strings.Contains(string(batches[1].body), "createNamedRange") {
		t.Errorf("the second batch is not the marker: %s", batches[1].body)
	}
	if !rep.Marked {
		t.Errorf("the marker did not read back: %v", rep.Warnings)
	}
	// The read the marker was built from is the one after the task batch: the
	// fixture before it ends its contents list in the same place, so what this
	// asserts is the order of the calls.
	var sawRead bool
	for _, c := range f.calls {
		if c.method == "GET" && strings.Contains(c.url, "includeTabsContent") && sawRead {
			return
		}
		if c.method == "GET" && strings.Contains(c.url, "includeTabsContent") {
			sawRead = true
		}
	}
	t.Error("the run took no second Docs read, so the marker was written from the read the batch was built on")
}

// optionsWithTasks is a publish of the note published-tasks.json came from.
func optionsWithTasks() Options {
	o := options()
	o.Tasks = theTasks()
	return o
}

// scriptWithTasks is a Google that answers the read before the task batch with
// one fixture and the read after it with another, and the read after the
// marker with the marked one.
func scriptWithTasks(t *testing.T, before, after string) *fakeSession {
	t.Helper()
	f := script(t, before)
	f.taskRead = fixture(t, after)
	return f
}
