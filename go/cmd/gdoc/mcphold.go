// The hold: the rules asked of a chat write before anything is sent, the answer
// a held write gets, and the thirty minutes the hold stays in the session.
//
// Every chat write passes through here, between the read it makes of its own
// target and the command that would send it, so a held write reaches no wire at
// all: TestAHeldWriteSendsNothing. What the rules are and why each is there
// belongs to internal/chat; this file is the wiring, which is three things.
//
// It builds the one write a rule judges out of the call's own arguments: the
// document the pin resolved, the title the document itself carries, the thread a
// reply names, the words that would be written, and for a propose the count of
// characters it takes out. Nothing here judges any of them.
//
// It answers a held write with the envelope a card is built from: ok false, sent
// false, and the hold's id, rule, value and text, with one fixed sentence that
// tells the model to say the reason and stop:
// TestTheHeldAnswerNamesTheRuleTheValueAndTheText.
//
// And it keeps the hold for thirty minutes, so the confirm tool of the next task
// has something to release: TestAHoldLivesThirtyMinutes.

package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"gdoc/internal/chat"
	"gdoc/internal/emit"
	"gdoc/internal/mcp"
)

// mcpHoldLife is how long a hold stays releasable in the process that made it.
//
// Thirty minutes is long enough for a person to come back to a card they left
// and short enough that nobody releases a write they have forgotten the reason
// for. A hold nobody answers in that time is gone, and the model has to ask
// again: TestAHoldLivesThirtyMinutes.
const mcpHoldLife = 30 * time.Minute

// mcpHeldSentence is the sentence every held answer ends with, word for word.
//
// It is fixed and it is here rather than left to the model, because a model
// writing its own words for a refusal writes words that lead back to trying
// again. This one says what happened and what to do about it, and nothing else.
const mcpHeldSentence = "Nothing was posted; tell the person this reason and end your turn."

// mcpHolds is every hold one session is keeping.
//
// Per process and in memory, like the ledger beside it, and guarded because the
// login listener answers beside the worker. A session that ends takes its holds
// with it, which is why a client restart loses a pending card and posts nothing.
type mcpHolds struct {
	mu   sync.Mutex
	byID map[string]chat.Hold
}

// newMCPHolds is one session's own.
func newMCPHolds() *mcpHolds {
	return &mcpHolds{byID: map[string]chat.Hold{}}
}

// keep puts one hold in the session's record, under the id its answer named.
func (h *mcpHolds) keep(held chat.Hold) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.byID[held.ID] = held
}

// hold is the hold with this id as it stands now, or nothing: an id this session
// never held, and one it held more than mcpHoldLife ago, answer the same way.
//
// Every expired hold is dropped on the way past, so a session that ran all day
// keeps the holds somebody may still answer and nothing else.
func (h *mcpHolds) hold(id string, now time.Time) *chat.Hold {
	h.mu.Lock()
	defer h.mu.Unlock()
	for other, held := range h.byID {
		if !held.Created.After(now.Add(-mcpHoldLife)) {
			delete(h.byID, other)
		}
	}
	held, ok := h.byID[id]
	if !ok {
		return nil
	}
	return &held
}

// mcpJudge is every hold rule over one call, before the command that would send
// it runs. It answers the envelope to send back and whether this call stops
// here.
//
// Three things stop a call: arguments the write cannot be read out of, a text
// carrying characters nobody can see, which is refused outright, and a rule that
// trips, which is held. Everything else goes on to the command.
func mcpJudge(c mcpCommand, args json.RawMessage, target mcpTarget, ch *mcpChat) (mcp.Result, bool) {
	w, err := mcpWriteOf(c, args, target, ch.ledger)
	if err != nil {
		return mcpEnvelope(emit.Result{OK: false, Error: err.Error()}), true
	}
	// The trusted domains are not wired yet: task 18 of the milestone 14 run 2
	// plan reads the flag and hands them in here.
	held, err := chat.Rules(w, ch.ledger, nil, now())
	if err != nil {
		return mcpEnvelope(emit.Result{OK: false, Error: err.Error()}), true
	}
	if held == nil {
		return mcp.Result{}, false
	}
	// The call as it arrived, so a release posts exactly what was held rather
	// than whatever the model sends with the release.
	held.Args = args
	ch.holds.keep(*held)
	return mcpHeldAnswer(*held), true
}

// holdData is what a held write answers with: nothing was sent, and the facts a
// card is built from.
//
// It carries no judgement about the person who wrote the comment or about the
// model that made the call. It is which rule tripped, the value that tripped it,
// and the words that would have gone out, which are there so a person at a
// client that shows no card can paste them into the document themselves.
type holdData struct {
	Sent bool      `json:"sent"`
	Held holdFacts `json:"held"`
}

type holdFacts struct {
	ID       string `json:"id"`
	Tool     string `json:"tool"`
	Document string `json:"document"`
	Rule     string `json:"rule"`
	Value    string `json:"value"`
	Text     string `json:"text"`
	Say      string `json:"say"`
}

// mcpHeldAnswer is the one envelope a held write gets.
//
// The reason and the fixed sentence are in the error field as well as in the
// facts, because the error is the field every other refusal arrives in and a
// model that reads only that one is still told what to say.
func mcpHeldAnswer(held chat.Hold) mcp.Result {
	return mcpEnvelope(emit.Result{
		OK:    false,
		Error: held.Reason + ". " + mcpHeldSentence,
		Data: holdData{Sent: false, Held: holdFacts{
			ID:       held.ID,
			Tool:     held.Tool,
			Document: held.Title,
			Rule:     held.Rule,
			Value:    held.Value,
			Text:     held.Text,
			Say:      mcpHeldSentence,
		}},
	})
}

// mcpItem is one item of a write's list, with every field the two list tools
// draw on their cards. A field a tool's own schema does not carry is absent from
// its calls, because fileBody refuses an item that holds one.
type mcpItem struct {
	Quoted      string `json:"quoted"`
	Why         string `json:"why"`
	Replacement string `json:"replacement"`
	Content     string `json:"content"`
	ReplaceFrom string `json:"replace_from"`
	ReplaceTo   string `json:"replace_to"`
}

// mcpWriteOf is the one write the rules judge, read out of the call.
//
// The property the words arrive in is the command's own flag property from the
// table above, rather than a name written again here, so the two cannot drift:
// reply's is body, annotate's annotations, propose's proposals. A write tool has
// exactly one such flag, and a tool that grew a second would be refused here
// rather than judged on half its words.
func mcpWriteOf(c mcpCommand, args json.RawMessage, target mcpTarget, led *chat.Ledger) (chat.Write, error) {
	given, err := mcpArguments(args)
	if err != nil {
		return chat.Write{}, err
	}
	w := chat.Write{Tool: c.tool, DocID: target.docID, Title: target.title}
	if slices.Contains(c.checks, quoteProp) {
		if w.ThreadID, err = mcpWord(given, threadProp); err != nil {
			return chat.Write{}, err
		}
	}
	if len(c.flags) != 1 {
		return chat.Write{}, fmt.Errorf("%s carries %d flags, and the words a rule judges would be half of them",
			c.tool, len(c.flags))
	}
	f := c.flags[0]
	raw, ok := given[f.prop]
	if !ok {
		return chat.Write{}, fmt.Errorf("%s is needed and was not given", f.prop)
	}
	if !f.raw {
		w.Text, err = mcpText(raw, f.prop)
		return w, err
	}
	item, err := mcpOneItem(raw, f.prop)
	if err != nil {
		return chat.Write{}, err
	}
	w.Text = mcpItemWords(item)
	// What a propose takes out is counted here, because the count is about the
	// document and the item together, and the rule is about the number alone.
	if c.tool == "propose" {
		w.Removed = mcpRemoved(item, led.Text(w.DocID))
	}
	return w, nil
}

// mcpOneItem is the one item a write carries. argv has already refused a call
// sending two, so the first is the write.
func mcpOneItem(raw json.RawMessage, prop string) (mcpItem, error) {
	var list []mcpItem
	if err := json.Unmarshal(raw, &list); err != nil {
		return mcpItem{}, fmt.Errorf("%s must be a list: %w", prop, err)
	}
	if len(list) == 0 {
		return mcpItem{}, fmt.Errorf("%s carries nothing, so there is nothing to write", prop)
	}
	return list[0], nil
}

// mcpItemWords is everything of an item a rule reads: the words that would reach
// the document, and the comment said about them.
//
// An annotate item carries only the comment, because the words it is anchored to
// are the document's own and it puts nothing new on the page. A propose carries
// the replacement or the new paragraphs as well.
func mcpItemWords(item mcpItem) string {
	parts := make([]string, 0, 3)
	for _, part := range []string{item.Replacement, item.Content, item.Why} {
		if strings.TrimSpace(part) != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "\n")
}

// mcpRemoved is how many characters a proposal takes out of the document.
//
// A words change takes out the words it quotes, whatever it puts back: a
// paragraph rewritten in full is a paragraph the person should read before it
// goes. A block change takes out the run from the first words it names to the
// last, measured in this session's own read of the document.
//
// Where those words are not in that read, the count is the length of what the
// call named, which is the least it can be. Nothing is lost by the
// understatement: the command underneath refuses a quote it cannot find in the
// document exactly once.
func mcpRemoved(item mcpItem, text string) int {
	if item.Quoted != "" {
		return len([]rune(item.Quoted))
	}
	if item.ReplaceFrom == "" {
		return 0
	}
	last := item.ReplaceTo
	if last == "" {
		last = item.ReplaceFrom
	}
	start := strings.Index(text, item.ReplaceFrom)
	if start < 0 {
		return len([]rune(item.ReplaceFrom))
	}
	end := strings.Index(text[start:], last)
	if end < 0 {
		return len([]rune(item.ReplaceFrom))
	}
	return len([]rune(text[start : start+end+len(last)]))
}
