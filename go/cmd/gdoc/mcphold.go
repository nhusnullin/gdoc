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
// And it keeps the hold for thirty minutes, so the confirm tool in
// mcprelease.go has something to release: TestAHoldLivesThirtyMinutes. One write
// is one hold there: a call the rules stopped and a model sent again finds the
// card already open rather than a second one, because two cards for one write
// could both be released once the write memory's ten minutes have passed:
// TestARetriedHeldWriteKeepsOneHold. And one write is at most one hold: a write
// that reaches the document another way takes its card with it, so nobody
// presses a card for a reply that is already posted:
// TestAHoldGoesWhenTheWriteItHeldLands.
//
// # A hold dies with the sign-in it was held under
//
// A held chat write waits up to thirty minutes for a person, and in that time
// somebody may sign out and sign in as somebody else. Releasing the held call
// then would post it under the next account, which nobody in that chat asked
// for. So mcpHolds keeps auth.SignInMark beside the hold, a domain-prefixed
// SHA-256 of the refresh token and no credential, and compares it at the
// release and in the sweep. Another mark or no token at all drops the hold,
// removes its confirm tool, writes the record's dropped line and answers that
// the sign-in changed and nothing was sent; a token that cannot be read keeps
// the hold and sends nothing, because not knowing may not become a release:
// TestAHoldRemembersTheSignInItWasHeldUnder,
// TestAReleaseAfterTheSignInChangedSendsNothingAndDropsTheHold,
// TestTheSweepDropsAHoldWhoseSignInChanged and
// TestAnUnreadableTokenAtReleaseKeepsTheHoldAndSendsNothing.
//
// The mark is read and never shown. It is in no answer, no tool schema and no
// line of the record, and no email is compared anywhere: a dropped hold says
// the sign-in is a different one and never whose it is, which is the identity
// invariant in this one place:
// TestTheSignInMarkReachesNoAnswerNoToolAndNoLine.
//
// One strain, accepted and written down. This process's own refresh of the
// refresh token is a new mark too, although internal/gapi does not count it
// as a newer sign-in, so a hold raised before such a rotation is dropped and
// the person is asked again. It fails towards nothing being sent, which is
// the side to fail on: TestARotatedRefreshTokenRefusesTheRelease.
//

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"gdoc/internal/auth"
	"gdoc/internal/chat"
	"gdoc/internal/docs"
	"gdoc/internal/emit"
	"gdoc/internal/mcp"
	"gdoc/internal/presence"
	"gdoc/internal/propose"
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

// The three sentences a release gets when the sign-in is not the one the hold
// was raised under. They are fixed, like the held sentence above and for the
// same reason, and none of them names an account: the mark says the sign-in is
// another one and never whose it is.
//
// Two of them say the card is gone, because it is: the hold was dropped the
// moment the sign-in changed, and asking the person to approve it again under
// the account they are signed in as now is the honest step. The third says the
// card is still open, because not knowing never resolves to sending and never
// resolves to taking a person's card away either.
const (
	mcpSignInChanged = "the sign-in on this computer is not the one gdoc held this write under, so nothing was sent and the card is gone. " +
		"Tell the person gdoc was signed in again since this write was held, and make the original call again if they still want this write."
	mcpSignedOutSince = "nobody is signed in to Google on this computer any more, so nothing was sent and the card is gone. " +
		"Tell the person they were signed out, and call the login tool before making the original call again."
	mcpSignInUnread = "Nothing was sent and the card is still open."
)

// mcpLister is the little of a server a hold needs: one tool registered, one
// taken away, each told to the client as a list that moved. internal/mcp's
// Server is what fills it, and a test watches what it was asked for.
type mcpLister interface {
	Add(mcp.Tool)
	Remove(string)
}

// mcpHolds is every hold one session is keeping.
//
// Per process and in memory, like the ledger beside it, and guarded because the
// login listener answers beside the worker. A session that ends takes its holds
// with it, which is why a client restart loses a pending card and posts nothing.
type mcpHolds struct {
	mu   sync.Mutex
	byID map[string]keptHold

	// on is whether this session records what it is doing, and errOut where
	// the one line goes when the record cannot be written. A hold is the one
	// thing in a chat a person is waiting on without the chat saying so, so
	// each of the four ways a hold begins and leaves writes one line:
	// TestAHeldWriteRecordsTheRuleAndNoneOfItsWords and
	// TestEveryWayAHoldLeavesIsExactlyOneLine.
	on     bool
	errOut io.Writer

	// lister is where a hold's confirm tool goes, and confirm is what builds
	// one. Both are set once the session has a server to tell, because the
	// tools are built before the server is. A session that wired neither keeps
	// its holds and registers nothing.
	lister  mcpLister
	confirm func(chat.Hold) mcp.Tool
}

// keptHold is one hold and the sign-in it was raised under.
//
// The two are kept together because a release is only an approval of this
// write under that sign-in: a person who approved a reply as one account did
// not approve it as the next one. internal/chat knows nothing of this, by
// design, because a hold rule is about words and this is about the token.
type keptHold struct {
	held chat.Hold

	// mark is auth.SignInMark of the token the hold was raised under: a hash
	// and never the token, the account or an email. Nothing carries it out of
	// this process: TestTheSignInMarkReachesNoAnswerNoToolAndNoLine.
	//
	// An empty mark is a hold raised when no token could be read, and it
	// matches no sign-in at all, so such a hold is dropped rather than sent.
	// That is the safe way round, and it cannot happen on the ordinary route:
	// every write tool reads the token before the rules are asked.
	mark string
}

// newMCPHolds is one session's own.
func newMCPHolds(on bool, errOut io.Writer) *mcpHolds {
	return &mcpHolds{byID: map[string]keptHold{}, on: on, errOut: errOut}
}

// mcpSignInNow is which sign-in is on this computer now, and nothing about
// whose it is: the mark of the token at the path, an empty mark where there is
// no token at all, and the error where there is one nobody can read.
//
// The three answers are three different things to a hold. Another mark is
// another sign-in, and the hold goes. No token is nobody signed in, and the
// hold goes too, because whoever signs in next is not who approved the write.
// A token that cannot be read is not knowing, and not knowing never resolves
// to sending: the hold stays and nothing goes out.
func mcpSignInNow() (string, error) {
	tok, err := auth.Load()
	if errors.Is(err, auth.ErrNoToken) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return auth.SignInMark(tok), nil
}

// marksOneSignIn is whether a hold raised under was may still be sent under
// is. An empty mark on either side is not a match: a hold that remembers no
// sign-in, and a computer nobody is signed in on, are both "not the sign-in
// this write was approved against".
func marksOneSignIn(was, is string) bool { return was != "" && was == is }

// listIn is where this session's confirm tools are listed. serveMCP calls it
// once, with the server it is about to run.
func (h *mcpHolds) listIn(l mcpLister, confirm func(chat.Hold) mcp.Tool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lister, h.confirm = l, confirm
}

// keep puts one hold in the session's record, under the id its answer named,
// registers the one tool that can release it, and answers the hold to report.
//
// A write this session is already holding is held once. The held answer is not
// remembered, on purpose, so the same call again is judged again; and a second
// hold for it would be a second card for one write, which the write memory
// covers for its ten minutes and not for the thirty a hold lives. Past that the
// person could approve the same words twice. So the hold already open is
// answered instead, under the id and the words the card in front of them
// carries: TestARetriedHeldWriteKeepsOneHold.
//
// The same write is chat.WriteKey's reading of it, which is the write memory's
// own, so the two cannot disagree about the order of the properties.
func (h *mcpHolds) keep(ctx context.Context, held chat.Hold) chat.Hold {
	key := chat.WriteKey(held.Tool, held.Args)
	// The sign-in this hold is raised under, read before the lock, because a
	// release under another one sends nothing: sameSignIn below.
	mark, _ := mcpSignInNow()
	h.mu.Lock()
	for _, open := range h.byID {
		if chat.WriteKey(open.held.Tool, open.held.Args) == key {
			h.mu.Unlock()
			return open.held
		}
	}
	h.byID[held.ID] = keptHold{held: held, mark: mark}
	lister, confirm := h.lister, h.confirm
	h.mu.Unlock()
	// One hold is one held line, so the retry above is already gone: it is the
	// hold that is open and not a second one. What travels is the rule's name
	// and never the value it tripped on, the reason or the words, because
	// every one of those is text somebody else wrote.
	presence.Held(ctx, h.on, releaseVersion(), held.Tool, held.DocID, held.ID, held.Rule,
		held.Created.Add(mcpHoldLife), h.errOut)
	if lister == nil || confirm == nil {
		return held
	}
	lister.Add(confirm(held))
	return held
}

// hold is the hold with this id as it stands now, or nothing: an id this session
// never held, and one it held more than mcpHoldLife ago, answer the same way.
func (h *mcpHolds) hold(id string, now time.Time) *chat.Hold {
	h.sweep(now)
	h.mu.Lock()
	defer h.mu.Unlock()
	kept, ok := h.byID[id]
	if !ok {
		return nil
	}
	return &kept.held
}

// signIn is the mark the hold with this id was raised under, and empty where
// this session keeps no such hold. It is here for the tests that ask what a
// hold remembers; nothing in the server reads it.
func (h *mcpHolds) signIn(id string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.byID[id].mark
}

// sameSignIn answers why the hold with this id cannot be sent under the
// sign-in on this computer now, and drops it where that sign-in is not the one
// it was raised under.
//
// A hold this session does not keep is nobody's business here: the lookup that
// follows says so in its own words. A token nobody can read keeps the hold and
// sends nothing. Anything else is a hold that will never be sendable again, so
// it goes with its tool and its one line:
// TestAReleaseAfterTheSignInChangedSendsNothingAndDropsTheHold.
func (h *mcpHolds) sameSignIn(id string) error {
	h.mu.Lock()
	kept, ok := h.byID[id]
	h.mu.Unlock()
	if !ok {
		return nil
	}
	mark, err := mcpSignInNow()
	if err != nil {
		return fmt.Errorf("%w. %s", err, mcpSignInUnread)
	}
	if marksOneSignIn(kept.mark, mark) {
		return nil
	}
	h.drop([]string{id})
	if mark == "" {
		return errors.New(mcpSignedOutSince)
	}
	return errors.New(mcpSignInChanged)
}

// dropChangedSignIn takes out every hold raised under a sign-in that is not
// the one on this computer now. The sweep calls it, so a card whose account
// changed leaves the client in its own time rather than waiting for somebody
// to press it: TestTheSweepDropsAHoldWhoseSignInChanged.
//
// A session keeping no hold asks the disk nothing, and a token nobody can read
// keeps every hold.
func (h *mcpHolds) dropChangedSignIn() {
	h.mu.Lock()
	open := len(h.byID)
	h.mu.Unlock()
	if open == 0 {
		return
	}
	mark, err := mcpSignInNow()
	if err != nil {
		return
	}
	h.mu.Lock()
	var changed []string
	for id, kept := range h.byID {
		if !marksOneSignIn(kept.mark, mark) {
			changed = append(changed, id)
		}
	}
	h.mu.Unlock()
	h.drop(changed)
}

// drop takes these holds out, writes the one line each that says the sign-in
// they were held under is gone, and takes their confirm tools off the list.
func (h *mcpHolds) drop(ids []string) {
	h.mu.Lock()
	gone := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := h.byID[id]; !ok {
			continue
		}
		delete(h.byID, id)
		gone = append(gone, id)
	}
	lister := h.lister
	h.mu.Unlock()
	h.left(gone, lister, presence.Dropped)
}

// left is the tail every way a hold leaves shares: the ids in one order, one
// line each saying how they left, and their confirm tools off the list. The
// caller has already taken them out of byID under the lock and handed in the
// lister it read there, because what leaves is the caller's own question and
// this is only what happens afterwards.
//
// say is the one thing that differs: presence.Dropped, Expired or Landed, all
// three of one shape. It is a parameter rather than three copies of these
// lines, so the order, the record and the list cannot be fixed in one place
// and left wrong in the other two: TestEveryWayAHoldLeavesIsExactlyOneLine.
func (h *mcpHolds) left(gone []string, lister mcpLister, say func(bool, string, string, io.Writer)) {
	// Sorted, so two leaving together leave in the same order every run, in
	// the record as well as on the list.
	slices.Sort(gone)
	for _, id := range gone {
		say(h.on, releaseVersion(), id, h.errOut)
	}
	if lister == nil {
		return
	}
	for _, id := range gone {
		lister.Remove(mcpConfirmPrefix + id)
	}
}

// sweep drops every hold older than mcpHoldLife and takes its tool off the list.
// It runs at the start of every tool call and whenever a hold is looked up, so a
// card nobody answered leaves the client in its own time rather than waiting for
// somebody to press it: TestAHoldRegistersOneConfirmToolAndRemovesItOnRelease.
//
// It drops the holds whose sign-in changed too, through dropChangedSignIn,
// which reads the token file once while this session keeps any hold at all and
// nothing at all while it keeps none: TestTheSweepDropsAHoldWhoseSignInChanged.
// So a card whose sign-in is no longer the one it was raised under leaves on
// the next tool call rather than on the press.
func (h *mcpHolds) sweep(now time.Time) {
	defer h.dropChangedSignIn()
	h.mu.Lock()
	var gone []string
	for id, kept := range h.byID {
		if !kept.held.Created.After(now.Add(-mcpHoldLife)) {
			delete(h.byID, id)
			gone = append(gone, id)
		}
	}
	lister := h.lister
	h.mu.Unlock()
	h.left(gone, lister, presence.Expired)
}

// landed drops the hold for a write that reached the document another way, and
// takes its confirm tool off the list.
//
// A held write the model sends again, after doing what the reason asked, goes
// out with no card at all. The card left behind is then a card for a write that
// already happened: the write memory answers a retry of it for ten minutes, and
// a hold lives thirty, so past the ten the person pressing it would post the
// same words a second time: TestAHoldGoesWhenTheWriteItHeldLands.
//
// The same write is chat.WriteKey's reading of it, the write memory's own, so
// the hold and the answer kept for that call cannot disagree about which call
// they are for.
func (h *mcpHolds) landed(tool string, args json.RawMessage) {
	key := chat.WriteKey(tool, args)
	h.mu.Lock()
	var gone []string
	for id, open := range h.byID {
		if chat.WriteKey(open.held.Tool, open.held.Args) == key {
			delete(h.byID, id)
			gone = append(gone, id)
		}
	}
	lister := h.lister
	h.mu.Unlock()
	h.left(gone, lister, presence.Landed)
}

// release takes one hold out and its tool off the list, which is what makes a
// confirm tool one-time: a second call of the same card finds no hold.
func (h *mcpHolds) release(id string) {
	h.mu.Lock()
	_, ok := h.byID[id]
	delete(h.byID, id)
	lister := h.lister
	h.mu.Unlock()
	if !ok {
		return
	}
	presence.Released(h.on, releaseVersion(), id, h.errOut)
	if lister == nil {
		return
	}
	lister.Remove(mcpConfirmPrefix + id)
}

// mcpJudge is every hold rule over one call, before the command that would send
// it runs. It answers the envelope to send back and whether this call stops
// here.
//
// Three things stop a call: arguments the write cannot be read out of, a text
// carrying characters nobody can see, which is refused outright, and a rule that
// trips, which is held. Everything else goes on to the command.
func mcpJudge(ctx context.Context, c mcpCommand, args json.RawMessage, target mcpTarget, ch *mcpChat) (mcp.Result, bool) {
	w, err := mcpWriteOf(c, args, target, ch.ledger)
	if err != nil {
		return mcpEnvelope(emit.Result{OK: false, Error: err.Error()}), true
	}
	held, err := chat.Rules(w, ch.ledger, now())
	if err != nil {
		return mcpEnvelope(emit.Result{OK: false, Error: err.Error()}), true
	}
	if held == nil {
		return mcp.Result{}, false
	}
	// The call as it arrived, so a release posts exactly what was held rather
	// than whatever the model sends with the release.
	held.Args = args
	// And the answer is the hold the session kept, which is this one unless the
	// same write is already held: keep says which.
	return mcpHeldAnswer(ch.holds.keep(ctx, *held)), true
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

// holdFacts carries the reason as its own field as well as inside the error,
// because the reason is one of the four words a release has to send back byte for
// byte, and reading it out of a sentence is not something a model should have to
// do: mcprelease.go, and TestAByteDifferentTitleReasonOrTextIsRefused.
type holdFacts struct {
	ID       string `json:"id"`
	Tool     string `json:"tool"`
	Document string `json:"document"`
	Rule     string `json:"rule"`
	Value    string `json:"value"`
	Reason   string `json:"reason"`
	Text     string `json:"text"`
	Say      string `json:"say"`
}

// mcpHeldAnswer is the one envelope a held write gets.
//
// The reason and the fixed sentence are in the error field as well as in the
// facts, because the error is the field every other refusal arrives in and a
// model that reads only that one is still told what to say.
//
// This is the one answer that says ok: false and is not an MCP error. Claude
// Desktop shows isError as "failed" in red, and a hold is not a failure: the
// person's next step is the card, and a red word on the step that protects them
// reads as a fault in the tool. DECISIONS.md, 2026-10-03, "What the first run
// in Claude Desktop changed", and
// TestTheHeldAnswerNamesTheRuleTheValueAndTheText.
func mcpHeldAnswer(held chat.Hold) mcp.Result {
	res := mcpHoldEnvelope(held, held.Reason)
	res.IsError = false
	return res
}

// mcpKeptAnswer is a release that released nothing: the same facts the hold
// answered with, and why this call did not send it. The hold is still there, so
// the answer is the card's own words again rather than a bare refusal.
//
// It stays an MCP error, the way every other ok: false is, because both of
// Release's refusals are the model's to put right. A card whose words differ is
// put right by sending the held words back:
// TestAByteDifferentTitleReasonOrTextIsRefused. An approval that came less than
// the quiet gap after the last call is put right by ending the turn, so the
// person can approve again in a moment: TestTheQuietGapIsFiveSeconds. The person
// already chose in both; what went wrong is the call.
func mcpKeptAnswer(held chat.Hold, why string) mcp.Result {
	return mcpHoldEnvelope(held, why)
}

// mcpHoldEnvelope is the envelope both of those are: nothing sent, the facts a
// card is built from, and one sentence saying why, ending in the fixed one. It
// is an error like any other ok: false, and mcpHeldAnswer is the one caller that
// takes that back.
func mcpHoldEnvelope(held chat.Hold, why string) mcp.Result {
	return mcpEnvelope(emit.Result{
		OK:    false,
		Error: why + ". " + mcpHeldSentence,
		Data: holdData{Sent: false, Held: holdFacts{
			ID:       held.ID,
			Tool:     held.Tool,
			Document: held.Title,
			Rule:     held.Rule,
			Value:    held.Value,
			Reason:   held.Reason,
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
	// An item with nothing said in it is refused here rather than judged. The
	// command underneath refuses it too, so nothing is lost; what is gained is
	// that no card is made of it. A hold whose text is empty registers a confirm
	// tool whose text argument cannot be sent back, because an empty text is
	// refused, so the person would be shown a card nothing can release:
	// TestAWriteWithNoWordsIsRefusedRatherThanHeld.
	if strings.TrimSpace(w.Text) == "" {
		return chat.Write{}, fmt.Errorf("the one item of %s says nothing, so there is nothing to write", f.prop)
	}
	// What a propose takes out is counted here, because the count is about the
	// document and the item together, and the rule is about the number alone.
	if c.tool == "propose" {
		w.Removed = mcpRemoved(item, target.doc)
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
// goes.
//
// A block change takes out whole paragraphs, from the start of the one holding
// replace_from to the end of the one holding replace_to. So the count runs to
// the ends of those paragraphs and not to the ends of the quotes: a call
// quoting six words at each end of two long paragraphs takes both paragraphs
// out, and counting the quotes would put a four-hundred-character deletion
// under the line.
//
// It is asked of the document the pin read rather than of the text projection
// of it, through the same propose.PlaceReplace the command underneath will
// place the change with, because the projection is not the document's
// characters: a link prints there as [words](target) and commented text inside
// [[c:ID]] markers, and skills/gdoc-review/propose.md tells the model to quote
// the bare words. Those words are in the document and not in its projection, so
// a search of the projection would miss a quote the command then finds and
// count a multi-paragraph deletion as a few dozen characters.
//
// The number is the delete range Docs is given, in the UTF-16 units a Docs
// index counts. Every branch counts in that one unit, because every branch is
// compared against the one Large removal line: a words change counted in runes
// would let through a deletion the same removal written as a block replace
// holds. Where PlaceReplace refuses the pair, the count is the length of what
// the call named, which is the least it can be, and nothing is lost by the
// understatement: that same refusal is what the command answers with, so
// nothing is written either way.
// TestABlockReplaceIsCountedInWholeParagraphs and
// TestBothKindsOfRemovalAreCountedInUTF16Units.
func mcpRemoved(item mcpItem, d *docs.Document) int {
	if item.Quoted != "" {
		return utf16Units(item.Quoted)
	}
	if item.ReplaceFrom == "" {
		return 0
	}
	last := item.ReplaceTo
	if last == "" {
		last = item.ReplaceFrom
	}
	place, err := propose.PlaceReplace(d, item.ReplaceFrom, last)
	if err != nil {
		return utf16Units(item.ReplaceFrom)
	}
	return place.Delete.End - place.Delete.Start
}

// utf16Units is the length of a string in the units a Docs index counts. A
// character outside the basic plane is two of them, which is why a rune count
// will not do.
func utf16Units(s string) int {
	return len(utf16.Encode([]rune(s)))
}
