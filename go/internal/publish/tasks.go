package publish

// The one batch that turns the task lines a note held into a checklist in the
// document publish has just made. The rule and the reason for each step are in
// the package comment, under "the tasks become a checklist after the upload".

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"gdoc/internal/docs"
	"gdoc/internal/docsreq"
	"gdoc/internal/tasks"
)

// dateFormat is how a due date is shown in the document: the abbreviated month
// with the day and the year, which is the form the probe of 2026-11-05
// measured a date chip reading back as.
const dateFormat = "DATE_FORMAT_MONTH_DAY_YEAR_ABBREVIATED"

// dueHour is the hour of the day a due date is sent at, in UTC. A task's date
// is a day and the chip takes a moment, so the moment is the middle of that
// day: noon UTC is the same calendar day in every zone from UTC-11 to UTC+11,
// and a midnight would be the day before for half the world.
const dueHour = 12

// Task is one task list item of the note, as publish sends it: the words the
// document shows, the whole address of the owner, the day it is due, whether
// the note ticked it, and the tokens as the docx carries them.
//
// Owner is already resolved. A bare "@anna" in a note is finished against the
// domain in effect by the caller, before the upload, because a publish that
// could not finish one must send nothing at all: cmd/gdoc's
// TestABareOwnerWithNoDomainIsRefusedBeforeTheUpload is the pin.
//
// Nothing here says whether a task matters or who will do it. The address is
// the one a person wrote, and the day is the day they wrote.
type Task struct {
	// Words is the item's own words, with no ballot character and no tokens.
	Words string
	// Owner is the whole address the person chip carries, and empty where the
	// item names nobody.
	Owner string
	// Due is the day the date chip carries, written YYYY-MM-DD, and empty
	// where the item names no day.
	Due string
	// Done is whether the note ticked the box. The docx has no box, so a done
	// item arrives with the crossed character and struck words.
	Done bool
	// Tokens is the owner and the day as the docx carries them, the spaces in
	// front of them included.
	Tokens string
}

// Text is the paragraph the uploaded document carries for this task: the
// ballot character, the words, and the tokens. It is what a paragraph of the
// read is matched against, so a document that is not the note's own render is
// found before anything is sent.
func (t Task) Text() string { return t.ballot() + t.Words + t.Tokens }

// ballot is the character this item opens with in the docx, which is the only
// thing there marking it as a task.
func (t Task) ballot() string {
	if t.Done {
		return tasks.BallotCrossed
	}
	return tasks.BallotEmpty
}

// chips is how many of each kind this task gets.
func (t Task) chips() (people, dates int) {
	if t.Owner != "" {
		people = 1
	}
	if t.Due != "" {
		dates = 1
	}
	return people, dates
}

// taskPara is one paragraph of the read that carries a task: where it sits in
// the body, its own span, the text of its runs joined, and where each of those
// runs starts on both sides, so a byte of the text can be named as an index.
type taskPara struct {
	at    int
	start int
	end   int
	text  string
	spans []textSpan
}

// textSpan is one text run: where its text begins in the paragraph's joined
// text, and the document index that byte sits at.
type textSpan struct {
	at    int
	index int
	text  string
}

// taskRequests is the whole batch, or the reason nothing is sent.
//
// It is built from the read the verify step already took, so a publish costs
// no extra read before it writes. Every index in it is that read's, which is
// why the bullets go first: a bullet is not a character, and taking one off or
// putting one on moves nothing (MEASURED.md, 2026-11-05).
//
// The text comes after, from the last task to the first and inside a paragraph
// from the end back, so every index still to be used names the character it
// named when the read was taken.
// TestTheBulletsComeFirstAndTheTextFromTheLastTaskToTheFirst,
// TestTheBulletsComeOffBeforeTheChecklistGoesOn and
// TestEachTokenBecomesItsChipWhereItStood.
func taskRequests(tab docs.Tab, want []Task) ([]map[string]any, error) {
	found := taskParagraphs(tab)
	if len(found) != len(want) {
		return nil, fmt.Errorf("the note holds %d task lines and the document came back with %d paragraphs opening with a ballot character, so nothing was sent: the document is the note's words and its tasks are not boxes anybody can tick", len(want), len(found))
	}
	for i, p := range found {
		if p.text != want[i].Text() {
			return nil, fmt.Errorf("task %d reads %q in the note and %q in the document, so nothing was sent: the document is the note's words and its tasks are not boxes anybody can tick", i+1, want[i].Text(), p.text)
		}
	}
	var reqs []map[string]any
	// The bullets first, one run of adjacent task paragraphs at a time. A
	// checklist request over part of a list turns the whole list, so the
	// bullets come off before the boxes go on and a plain item beside a task
	// stays the bullet it was.
	for _, run := range runsOf(found) {
		start, end := run[0].start, run[len(run)-1].end
		reqs = append(reqs, docsreq.DeleteBullets(start, end), checklistOver(start, end))
	}
	for i := len(found) - 1; i >= 0; i-- {
		one, err := chipRequests(found[i], want[i])
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, one...)
	}
	return reqs, nil
}

// checklistOver is the one request that gives a stretch of paragraphs a box
// each. BULLET_CHECKBOX is the preset Docs names for it.
func checklistOver(start, end int) map[string]any {
	return map[string]any{"createParagraphBullets": map[string]any{
		"range":        map[string]any{"startIndex": start, "endIndex": end},
		"bulletPreset": "BULLET_CHECKBOX",
	}}
}

// chipRequests is one task paragraph's text: each token deleted and its chip
// put where the token stood, from the end of the paragraph back, and then the
// ballot character and the space after it taken off the front.
func chipRequests(p taskPara, want Task) ([]map[string]any, error) {
	read, err := tasks.Parse(want.Words + want.Tokens)
	if err != nil {
		return nil, fmt.Errorf("the task %q no longer reads as one: %w", want.Text(), err)
	}
	ballot := len(want.ballot())
	type token struct {
		start, end int
		chip       map[string]any
	}
	var list []token
	if want.Owner != "" && read.Owner.Present() {
		list = append(list, token{ballot + read.Owner.Start, ballot + read.Owner.End, nil})
		list[len(list)-1].chip = personChip(want.Owner)
	}
	if want.Due != "" && read.Due.Present() {
		chip, err := dateChip(want.Due)
		if err != nil {
			return nil, err
		}
		list = append(list, token{ballot + read.Due.Start, ballot + read.Due.End, chip})
	}
	// Descending, so a token earlier in the line still sits where the read
	// said it did when its turn comes.
	sort.Slice(list, func(i, j int) bool { return list[i].start > list[j].start })

	var reqs []map[string]any
	for _, t := range list {
		from, ok := p.indexAt(t.start)
		to, alsoOk := p.indexAt(t.end)
		if !ok || !alsoOk {
			return nil, fmt.Errorf("the task %q is split across runs this read cannot name an index in, so nothing was sent", want.Text())
		}
		reqs = append(reqs, deleteRange(from, to), at(t.chip, from))
	}
	end, ok := p.indexAt(ballot)
	if !ok {
		return nil, fmt.Errorf("the task %q carries no ballot character this read can name an index for, so nothing was sent", want.Text())
	}
	return append(reqs, deleteRange(p.start, end)), nil
}

// personChip is a person chip naming one address, without the index it goes
// at: at() puts it there, so the location is written in one place.
func personChip(email string) map[string]any {
	return map[string]any{"insertPerson": map[string]any{
		"personProperties": map[string]any{"email": email},
	}}
}

// dateChip is a date chip for one calendar day, sent as noon UTC on that day
// for the reason dueHour states.
func dateChip(day string) (map[string]any, error) {
	d, err := time.Parse(tasks.DayLayout, day)
	if err != nil {
		return nil, fmt.Errorf("%s is no day on the calendar, so nothing was sent: %w", day, err)
	}
	stamp := time.Date(d.Year(), d.Month(), d.Day(), dueHour, 0, 0, 0, time.UTC)
	return map[string]any{"insertDate": map[string]any{
		"dateElementProperties": map[string]any{
			"timestamp":  stamp.Format(time.RFC3339),
			"dateFormat": dateFormat,
		},
	}}, nil
}

// at puts an insert request's location on it. Both chip kinds name where they
// go the same way, and a location written twice is a location that drifts.
func at(req map[string]any, index int) map[string]any {
	for _, v := range req {
		if fields, ok := v.(map[string]any); ok {
			fields["location"] = map[string]any{"index": index}
		}
	}
	return req
}

// deleteRange takes one stretch of characters out.
func deleteRange(start, end int) map[string]any {
	return map[string]any{"deleteContentRange": map[string]any{
		"range": map[string]any{"startIndex": start, "endIndex": end},
	}}
}

// taskParagraphs is every top-level paragraph of the body whose text opens
// with a ballot character, in reading order.
//
// The ballot character is the only thing marking a task in the uploaded
// document: the docx carries no style and no bookmark on a task item, so this
// is what there is to find them by. A paragraph inside a table is not one: a
// task in a table cell is out of scope and would be refused by internal/body
// before a document was ever made.
func taskParagraphs(tab docs.Tab) []taskPara {
	var found []taskPara
	for i, b := range tab.Body {
		if b.Paragraph == nil {
			continue
		}
		text, spans := joinRuns(b.Paragraph)
		if !strings.HasPrefix(text, tasks.BallotEmpty) && !strings.HasPrefix(text, tasks.BallotCrossed) {
			continue
		}
		found = append(found, taskPara{
			at:    i,
			start: b.Paragraph.StartIndex,
			end:   b.Paragraph.EndIndex,
			text:  text,
			spans: spans,
		})
	}
	return found
}

// joinRuns is a paragraph's text as one string, with the newline that closes
// it taken off, and where each run's text begins on both sides.
//
// Only the text runs. A chip or a picture carries no text and is one unit of
// the document's own count, so a paragraph holding one joins to something the
// note's own line cannot equal, and that paragraph is a task gdoc does not
// recognise rather than one it half writes.
func joinRuns(p *docs.Paragraph) (string, []textSpan) {
	var b strings.Builder
	var spans []textSpan
	for _, r := range p.Runs {
		if r.Kind != docs.KindText {
			continue
		}
		spans = append(spans, textSpan{at: b.Len(), index: r.StartIndex, text: r.Text})
		b.WriteString(r.Text)
	}
	return strings.TrimSuffix(b.String(), "\n"), spans
}

// indexAt is the document index of one byte of the paragraph's joined text.
// The runs are walked rather than counted from the paragraph's start, because
// a done task's words are struck and a strike is a run of its own.
func (p taskPara) indexAt(off int) (int, bool) {
	for _, s := range p.spans {
		if off < s.at || off > s.at+len(s.text) {
			continue
		}
		return s.index + docsreq.Len(s.text[:off-s.at]), true
	}
	return 0, false
}

// runsOf groups the task paragraphs into stretches that sit next to each other
// in the body. A plain item between two tasks ends a stretch, because the
// bullets come off and the boxes go on one stretch at a time.
func runsOf(found []taskPara) [][]taskPara {
	var out [][]taskPara
	for i, p := range found {
		if i > 0 && found[i-1].at == p.at-1 {
			out[len(out)-1] = append(out[len(out)-1], p)
			continue
		}
		out = append(out, []taskPara{p})
	}
	return out
}

// tasksLanded is what the read after the batch does not show, one sentence
// each. An empty answer is every task on the page as a box with the chips it
// was given.
//
// It counts and compares; it judges nothing. A task that did not land is a
// fact about a document that exists, so it is a warning and a check that did
// not hold, never a failed publish.
// TestAMissingChipIsTasksFalseAndAWarning.
func tasksLanded(tab docs.Tab, want []Task) []string {
	got := checklistParagraphs(tab)
	if len(got) != len(want) {
		return []string{taskWarning(fmt.Sprintf(
			"the note holds %d task lines and the document reads back with %d checklist items", len(want), len(got)))}
	}
	var warns []string
	for i, p := range got {
		text, _ := joinRuns(p)
		if words := strings.TrimRight(text, " "); words != want[i].Words {
			warns = append(warns, taskWarning(fmt.Sprintf(
				"task %d reads %q in the note and %q in the document", i+1, want[i].Words, words)))
		}
		people, dates := countChips(p)
		wantPeople, wantDates := want[i].chips()
		if people != wantPeople {
			warns = append(warns, taskWarning(fmt.Sprintf(
				"task %d names %d people in the note and reads back with %d name chips", i+1, wantPeople, people)))
		}
		if dates != wantDates {
			warns = append(warns, taskWarning(fmt.Sprintf(
				"task %d names %d days in the note and reads back with %d date chips", i+1, wantDates, dates)))
		}
	}
	return warns
}

// checklistParagraphs is every top-level paragraph the read says is drawn with
// a box, in reading order. Every checklist in the document is one this batch
// made, because the note it was rendered from had none.
func checklistParagraphs(tab docs.Tab) []*docs.Paragraph {
	var found []*docs.Paragraph
	for _, b := range tab.Body {
		if b.Paragraph == nil || b.Paragraph.Bullet == nil || !b.Paragraph.Bullet.Checklist {
			continue
		}
		found = append(found, b.Paragraph)
	}
	return found
}

// countChips is how many name chips and how many date chips one paragraph
// carries.
func countChips(p *docs.Paragraph) (people, dates int) {
	for _, r := range p.Runs {
		switch r.Kind {
		case docs.KindPerson:
			people++
		case docs.KindDate:
			dates++
		}
	}
	return people, dates
}

// taskWarning is one reason the tasks are not boxes, with what that costs. It
// is the shape unmarked has, and for the same reason: the document is there
// and its words are right, and what is missing is named with its consequence.
func taskWarning(why string) string {
	return fmt.Sprintf("the document's task lines are not a checklist, because %s. The document is there and its words are right, but its tasks read as the characters ☐ and ☒ rather than boxes anybody can tick", why)
}

// convert turns the note's task lines into a checklist and reads the document
// back to see it.
//
// It raises nothing, for the reason verify raises nothing: the document is in
// the folder whatever happens here, and a document whose tasks are characters
// rather than boxes is still the note's words. It answers the read the caller
// should go on with, which is the one taken after the batch where there was a
// batch: every index after the cover has moved.
//
// A note with no task is the whole of the common case today, and it costs
// nothing: no request of any kind, the read unchanged, and the check true
// because nothing was asked for.
func convert(ctx context.Context, s Session, id string, d *docs.Document, want []Task) (*docs.Document, bool, []string) {
	if len(want) == 0 {
		return d, true, nil
	}
	if d == nil || len(d.Tabs) != 1 {
		return d, false, []string{taskWarning("the document could not be read back as one tab, so there was nothing to find the task lines in")}
	}
	reqs, err := taskRequests(d.Tabs[0], want)
	if err != nil {
		return d, false, []string{taskWarning(err.Error())}
	}
	body := map[string]any{"requests": reqs}
	if err := s.PostJSON(ctx, batchURL(id), body, nil); err != nil && !sentAnyway(err) {
		return d, false, []string{taskWarning(fmt.Sprintf("the batch failed: %v", err))}
	}
	// A batch whose answer was lost may have landed, so the read-back decides
	// that case too rather than the error.
	after, err := docs.Fetch(ctx, s, id)
	if err != nil {
		return d, false, []string{taskWarning(fmt.Sprintf("the batch was sent and the document could not be read back to see it: %v", err))}
	}
	if len(after.Tabs) != 1 {
		return d, false, []string{taskWarning(fmt.Sprintf("the batch was sent and the document came back with %d tabs, so nothing could be counted", len(after.Tabs)))}
	}
	if warns := tasksLanded(after.Tabs[0], want); len(warns) > 0 {
		return after, false, warns
	}
	return after, true, nil
}
