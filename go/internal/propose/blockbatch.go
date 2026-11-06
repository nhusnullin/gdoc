// This file is the write one block proposal makes: the text it inserts, where
// each of its new paragraphs lands once it has, and the request list that states
// their styles, their bullets and the comment carrying the reason. place.go says
// where the block goes, block.go reads its content, propose.go holds the words
// kind's own batch, and doc.go states each rule with the test that pins it.

package propose

import (
	"encoding/json"
	"strings"
	"time"

	"gdoc/internal/docsreq"
	"gdoc/internal/plaintext"
	"gdoc/internal/tasks"
)

// dateFormat is how a due date is shown in the document, and dueHour the hour
// of the day it is sent at, in UTC. They are internal/publish's two values,
// stated here as literals because that is the house rule for a value a test
// pins: a task proposed into a document and a task published into one show the
// same day in the same words. internal/publish's package comment says why the
// hour is the middle of the day.
const (
	dateFormat = "DATE_FORMAT_MONTH_DAY_YEAR_ABBREVIATED"
	dueHour    = 12
)

// MaxContent is how long a block's content may be, in bytes.
//
// The ceiling exists because of the guard rather than because of Docs. The guard
// reads a batchUpdate body to judge the requests in it, and a body past its peek
// arrives there truncated and is refused rather than carried unread. That
// refusal names the guard and a truncated body, which is nothing a person
// writing a block can act on, so Check refuses a content longer than this first
// and says what a block is for.
//
// The number is a measurement rather than a guess. The batch grows fastest in
// requests per byte on the shortest paragraphs there are, and fastest of all
// when they alternate with list items, because then every paragraph costs a
// named style and a bullet request of its own: "a\n\n- b\n\n" repeated to this
// length builds around 500 kilobytes, which is under half the guard's ceiling.
// TestALargeBlockStaysUnderThePeek builds that shape and three more at exactly
// this length and has the real policy judge each one.
//
// It is a section rather than a document. Eight kilobytes is over a thousand
// words, and a review answer longer than that is a note that wants publishing
// rather than a block that wants proposing.
const MaxContent = 8 << 10

// clearedMarks is the field mask of the one request that clears what the insert
// inherited.
//
// Text inserted at a paragraph's start takes that paragraph's first run style,
// so a block in front of a bold linked sentence would arrive bold and linked.
// The mask names every mark the content can carry and the object it comes with
// is empty, which is how the Docs API is told to put a field back to its
// default: stating bold false would clear a boolean, but nothing states "no
// link" except leaving the field out of an object whose mask names it.
//
// It is the marks the content can carry, and not the face, the size or the
// colour, which are inherited too and are not put right by clearing them: a
// restyled document carries those directly on every paragraph, so a block
// cleared back to the document's defaults would be the one paragraph that
// matches nothing around it. docs/backlog/a-block-inherits-the-look-of-what-it-
// lands-in-front-of.md holds what would settle that, and it is a measurement
// and a decision rather than a wider mask.
const clearedMarks = "bold,italic,underline,strikethrough,link"

// bulletPresets is the preset each list kind takes. The two kinds are two
// presets, which is the whole reason Para carries the kind rather than a flag.
var bulletPresets = map[ListKind]string{
	Bulleted: "BULLET_DISC_CIRCLE_SQUARE",
	Numbered: "NUMBERED_DECIMAL_ALPHA_ROMAN",
	TaskList: "BULLET_CHECKBOX",
}

// paraSpan is one new paragraph's span in the document the insert makes, from
// its first character to one past its paragraph mark.
type paraSpan struct {
	Start int
	End   int
}

// blockLayout is what one block inserts and where its paragraphs land.
//
// It is computed rather than stored, like the placement it is built from: the
// read the write goes out on is the only thing these indexes are true of.
type blockLayout struct {
	// Text is exactly what the insertText request carries.
	Text string
	// Paras is one span per new paragraph, in content order.
	Paras []paraSpan
}

// layOutBlock is the block's text and the span of each of its paragraphs.
//
// In the ordinary case the text is the paragraphs each with its own newline, and
// the first one starts where the insert does. After the document's last
// paragraph there is nothing to go in front of, so the text opens with a newline
// and carries no trailing one: the newline becomes the anchor's new paragraph
// mark, every new paragraph begins one unit further on, and the block's last
// paragraph owns the body's old final mark, which is why its span reaches one
// unit past the text. MEASURED.md row 7 is the measurement, and that row came
// back with one suggestion id because nothing restated that paragraph.
// BlockBatch restates nothing on it either, and PlaceAfter refuses an anchor
// whose style or bullet the block would then be stuck with.
//
// The caller has content that ParseContent read, so there is at least one
// paragraph and each one has words in it.
func layOutBlock(place Placement, content []Para) blockLayout {
	var body strings.Builder
	for _, p := range content {
		body.WriteString(p.Text())
		body.WriteString("\n")
	}

	out := blockLayout{Text: body.String()}
	at := place.Insert
	if place.AtEnd {
		out.Text = "\n" + strings.TrimSuffix(out.Text, "\n")
		at++
	}
	for _, p := range content {
		end := at + utf16Len(p.Text()) + 1
		out.Paras = append(out.Paras, paraSpan{Start: at, End: end})
		at = end
	}
	return out
}

// BlockBatch is the whole write for one block proposal: one request list, in the
// order it must be applied, inside one SUGGEST batchUpdate.
//
// The order is the reason it works, and it is measured rather than guessed
// (MEASURED.md, "A block of new paragraphs, proposed in one SUGGEST batch"):
//
//   - insertText, at the start of the paragraph behind the anchor, so every new
//     paragraph owns a mark of its own and every request below folds into the
//     one suggestion id;
//   - updateParagraphStyle per new paragraph, because an inserted paragraph
//     takes the named style of the paragraph it landed in and a list item in
//     front of somebody's Heading 1 would be a heading. After the document's
//     last paragraph the block's own last one is left out, because it owns the
//     body's old final mark and MEASURED.md row 7 restated nothing there;
//   - one updateTextStyle clearing what the insert inherited, before the block's
//     own marks, so a run that really is bold is written bold again after it;
//   - updateTextStyle per marked run;
//   - createParagraphBullets per stretch of list items of one kind, with a
//     deleteParagraphBullets in front of it on a task stretch, because a
//     checkbox preset over the list the insert inherited is a second
//     suggestion;
//   - deleteParagraphBullets per stretch of new paragraphs that are not list
//     items, because stating a named style does not clear an inherited bullet,
//     and with the same last paragraph left out at the end of a document;
//   - deleteContentRange for a replace, at the indexes the insert left the old
//     paragraphs at;
//   - insertComment on the words of the first new paragraph;
//   - insertPerson and insertDate for a task list's chips, last of all.
//
// The chips are last because each one is a unit the document did not have, so
// every index behind it moves: sent earlier, a chip would move the old
// paragraphs under a replace's deletion and the words under the comment's
// anchor. They go from the last paragraph to the first, and inside a paragraph
// from the end back, so no index a later chip names has moved either. Each one
// goes in the space Para.Text left for it.
//
// Every index is counted in UTF-16 code units, which is what the Docs API
// counts. One batch, because each request is built at the indexes the ones
// before it leave behind: two batches would mean computing the second from a
// document the first had already changed.
func BlockBatch(place Placement, content []Para, why, assignee, mark string) []byte {
	l := layOutBlock(place, content)
	var reqs []any
	add := func(kind string, body map[string]any) {
		reqs = append(reqs, map[string]any{kind: body})
	}

	add("insertText", map[string]any{
		"location": map[string]any{"index": place.Insert},
		"text":     l.Text,
	})
	for i, p := range content {
		if place.AtEnd && i == len(content)-1 {
			// That paragraph owns the body's old final mark, and MEASURED.md
			// row 7 came back with one suggestion id because nothing restated
			// it. PlaceAfter has already refused an anchor that is not plain
			// body text, so the style this paragraph inherits from that mark is
			// the style the block asked for.
			continue
		}
		add("updateParagraphStyle", map[string]any{
			"range":          spanOf(l.Paras[i].Start, l.Paras[i].End),
			"paragraphStyle": map[string]any{"namedStyleType": p.Style},
			"fields":         "namedStyleType",
		})
	}
	add("updateTextStyle", map[string]any{
		"range":     spanOf(place.Insert, place.Insert+utf16Len(l.Text)),
		"textStyle": map[string]any{},
		"fields":    clearedMarks,
	})
	for i, p := range content {
		at := l.Paras[i].Start
		for _, r := range p.Runs {
			end := at + utf16Len(r.Text)
			if set, mask := runMarks(r); mask != "" {
				add("updateTextStyle", map[string]any{
					"range":     spanOf(at, end),
					"textStyle": set,
					"fields":    mask,
				})
			}
			at = end
		}
	}
	for _, s := range listStretches(content) {
		from, to := l.Paras[s.from].Start, l.Paras[s.to].End
		if content[s.from].List == TaskList {
			// The bullets come off before the boxes go on. New text takes the
			// list membership of the paragraph it lands in front of, and a
			// checkbox preset over an inherited list is a change to that
			// list's own properties, which Docs records as a second
			// suggestion: TestLiveProposeSuggestsATaskList measured two ids on
			// 2026-11-07 for a task block placed in front of a disc bullet,
			// and a block withdraw cannot take back whole is not one this
			// package will make. Off the list first, the boxes make a list of
			// their own inside the one insertion, which is the shape
			// MEASURED.md's 2026-11-05 row came back with one id on.
			reqs = append(reqs, docsreq.DeleteBullets(from, to))
		}
		add("createParagraphBullets", map[string]any{
			"range":        spanOf(from, to),
			"bulletPreset": bulletPresets[content[s.from].List],
		})
	}
	for _, s := range plainStretches(content, place.AtEnd) {
		reqs = append(reqs, docsreq.DeleteBullets(l.Paras[s.from].Start, l.Paras[s.to].End))
	}
	if place.Replaces() {
		// The old paragraphs are where the insert left them: it went in at the
		// start of the first of them, so both ends moved along by its length.
		shift := utf16Len(l.Text)
		add("deleteContentRange", map[string]any{
			"range": spanOf(place.Delete.Start+shift, place.Delete.End+shift),
		})
	}

	// The comment is anchored on the words of the first new paragraph, without
	// its paragraph mark: a range carrying a mark anchors across into the
	// paragraph behind it, and every comment gdoc writes is on words. The
	// insertComment shape is measured rather than documented, and it is the one
	// propose.Batch sends (DECISIONS.md, 2026-08-29).
	comment := map[string]any{
		"range":   spanOf(l.Paras[0].Start, l.Paras[0].End-1),
		"content": plaintext.Prefixed(mark, why),
	}
	if assignee != "" {
		comment["assigneeEmailAddress"] = assignee
	}
	add("insertComment", comment)

	for i := len(content) - 1; i >= 0; i-- {
		for _, c := range chipRequests(content[i], l.Paras[i].Start) {
			reqs = append(reqs, c)
		}
	}

	body := map[string]any{
		"requests":     reqs,
		"writeControl": map[string]any{"writeMode": "SUGGEST"},
	}
	// The body is built from maps with no cycles and no unsupported types, so
	// the marshal cannot fail. Bytes rather than a map, so the caller cannot
	// change what the guard has already judged.
	raw, _ := json.Marshal(body)
	return raw
}

// runMarks is one run's style object and the mask that names it, built in one
// pass so a mark cannot be stated in the object and missing from the mask, or
// the other way round, which is how a property gets reset by a request that
// meant to leave it alone. It is the rule docsreq.Fields holds for the two
// packages that write the house look, in the three fields this one has.
//
// A run with no marks states nothing and sends no request: the clearing request
// above has already put it back to the document's own body style.
func runMarks(r Run) (map[string]any, string) {
	set := map[string]any{}
	var names []string
	put := func(name string, v any) {
		set[name] = v
		names = append(names, name)
	}
	if r.Bold {
		put("bold", true)
	}
	if r.Italic {
		put("italic", true)
	}
	if r.Link != "" {
		put("link", map[string]any{"url": r.Link})
	}
	return set, strings.Join(names, ",")
}

// chipRequests is one paragraph's chips, from the last to the first, each at
// the index of the space Para.Text left for it.
//
// A name chip is written before a date chip whichever order the item wrote the
// two tokens in, because both land at the end of the words rather than where
// the token stood: internal/publish keeps the written order, because there the
// tokens are in the document and each chip replaces one.
func chipRequests(p Para, start int) []map[string]any {
	chips := p.chips()
	if chips == 0 {
		return nil
	}
	// The first space is one unit behind the words, and each chip goes in the
	// space of its own.
	at := start + utf16Len(p.words()) + chips
	var out []map[string]any
	if p.Due != "" {
		out = append(out, dateChip(p.Due, at))
		at--
	}
	if p.Owner != "" {
		out = append(out, personChip(p.Owner, at))
	}
	return out
}

// personChip is a name chip for one whole address, at one index.
func personChip(email string, at int) map[string]any {
	return map[string]any{"insertPerson": map[string]any{
		"location":         map[string]any{"index": at},
		"personProperties": map[string]any{"email": email},
	}}
}

// dateChip is a date chip for one calendar day, at one index.
//
// The day is sent as noon UTC on that day, in the format the probe of
// 2026-11-05 measured a date chip reading back in. internal/publish says why
// the hour is noon, and both packages ask internal/tasks for the layout rather
// than spelling a second copy of it. A day internal/tasks has already read is
// a day on the calendar, so a parse that fails here cannot happen and the chip
// is left out rather than guessed: the read-back then says the chip is
// missing.
func dateChip(day string, at int) map[string]any {
	d, err := time.Parse(tasks.DayLayout, day)
	if err != nil {
		return map[string]any{}
	}
	stamp := time.Date(d.Year(), d.Month(), d.Day(), dueHour, 0, 0, 0, time.UTC)
	return map[string]any{"insertDate": map[string]any{
		"location": map[string]any{"index": at},
		"dateElementProperties": map[string]any{
			"timestamp":  stamp.Format(time.RFC3339),
			"dateFormat": dateFormat,
		},
	}}
}

// stretch is a run of neighbouring new paragraphs, both ends inclusive.
type stretch struct {
	from int
	to   int
}

// listStretches is the block's list items, grouped into the stretches one
// createParagraphBullets can cover: neighbours of the same kind. A bulleted list
// behind a numbered one is two requests, because the two kinds are two presets.
func listStretches(content []Para) []stretch {
	var out []stretch
	for i := 0; i < len(content); {
		if content[i].List == NotAList {
			i++
			continue
		}
		j := i
		for j < len(content) && content[j].List == content[i].List {
			j++
		}
		out = append(out, stretch{from: i, to: j - 1})
		i = j
	}
	return out
}

// plainStretches is the block's paragraphs that are not list items, grouped the
// same way.
//
// They are grouped because the removal is one request per stretch rather than
// one per paragraph, and they are separate stretches because a paragraph behind
// a list is not beside the heading in front of it, and a request covering the
// list between them would take its bullets off too.
//
// atEnd leaves the block's last paragraph out, because there it owns the body's
// old final mark and the batch restates nothing on that mark: MEASURED.md row 7,
// and the same reason the named style is left off it. PlaceAfter has already
// refused an anchor carrying a bullet, so the mark has none to clear. The last
// paragraph is always in a plain stretch there, because PlaceAfter refuses a
// block whose own last paragraph is a list item.
func plainStretches(content []Para, atEnd bool) []stretch {
	last := len(content)
	if atEnd {
		last--
	}
	var out []stretch
	for i := 0; i < last; {
		if content[i].List != NotAList {
			i++
			continue
		}
		j := i
		for j < last && content[j].List == NotAList {
			j++
		}
		out = append(out, stretch{from: i, to: j - 1})
		i = j
	}
	return out
}

// spanOf is one range as the Docs API takes it.
func spanOf(start, end int) map[string]any {
	return map[string]any{"startIndex": start, "endIndex": end}
}
