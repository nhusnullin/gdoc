// The confirm tool: the one tool a hold registers, the card the person reads on
// it, and the write their own approval sends.
//
// A held write is answered and gone (mcphold.go). What brings it back is not a
// yes typed in the chat, because that passes through the model: it is a tool
// this hold registered, named after this hold, which the client draws as a card
// with every argument on it. The person approves that card or nothing happens.
//
// Three things are asked of the call that releases a hold, and all three are
// asked by the binary. The card's four words have to be the hold's own, byte for
// byte, so the person cannot be shown a paraphrase of why the write was stopped:
// internal/chat's Release is where that is judged. Five seconds of quiet have to
// sit behind the call, because the one shape that is not an honest approval is a
// model releasing its own write inside the turn that was held. And the hold has
// to still be there: thirty minutes, or until one write went out under it.
//
// What goes out is the call as it arrived, kept on the hold, and never anything
// the release sends: TestAReleasedHoldPostsExactlyTheHeldTextOnce. The rules are
// not asked again, because they were asked and the person answered them.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"gdoc/internal/chat"
	"gdoc/internal/emit"
	"gdoc/internal/mcp"
)

// mcpConfirmPrefix is what a confirm tool's name opens with. The rest of it is
// the hold's own id, so the name never existed before this hold: always-allow is
// stored per tool name, and it cannot be granted in advance to a name nobody has
// seen. TestAHoldRegistersOneConfirmToolAndRemovesItOnRelease.
const mcpConfirmPrefix = "confirm_"

// The four arguments a release carries, in the order the card draws them: which
// hold, and the three sentences the person reads.
const (
	holdProp   = "hold"
	reasonProp = "reason"
	textProp   = "text"
)

// mcpConfirmSchema is the card. Nothing on it is optional, because the whole of
// it is the point: what the person approves is the document, the reason and the
// words, as gdoc wrote them:
// TestTheConfirmSchemaListsHoldTitleReasonText.
const mcpConfirmSchema = objectTop +
	`"hold":{"type":"string","description":"The hold id, exactly as the held answer gave it."},` +
	`"title":{"type":"string","description":"The document's title, exactly as the held answer gave it."},` +
	`"reason":{"type":"string","description":"Why gdoc held this write, exactly as the held answer gave it."},` +
	`"text":{"type":"string","description":"The words that would be written, exactly as the held answer gave them."}` +
	`},"required":["hold","title","reason","text"],"additionalProperties":false}`

// mcpConfirmTool is the one-time tool one hold registers.
//
// It is not wrapped the way the session's own tools are: a release records no
// instant for the quiet gap, because an approval is the person's and not the
// session's own work, and a second card waiting on the first would help nobody.
// The sweep it needs happens where it looks its hold up.
func mcpConfirmTool(held chat.Hold, errOut io.Writer, ch *mcpChat) mcp.Tool {
	return mcp.Tool{
		Name:  mcpConfirmPrefix + held.ID,
		Title: "Send the " + held.Tool + " gdoc held",
		Description: "gdoc held this " + held.Tool + " and sent nothing, because " + held.Reason + ". " +
			"Calling this tool asks the person to approve that one write. " +
			"It sends exactly the words gdoc held, into the document the card names, and nothing else. " +
			"Send the hold, the title, the reason and the text exactly as the held answer gave them. " +
			"Tell the person the reason and end your turn first: a call made in the same breath as the hold is refused and the hold kept.",
		Schema: json.RawMessage(mcpConfirmSchema),
		Call: func(ctx context.Context, args json.RawMessage) mcp.Result {
			return mcpRelease(ctx, held.ID, args, errOut, ch)
		},
	}
}

// mcpHoldGone is what a release of a hold this session is no longer keeping
// answers. Thirty minutes passed, or one write already went out under it. Either
// way the sentence says to ask again rather than to try again, because the words
// have to be judged against the document as it stands now.
const mcpHoldGone = "this hold is no longer open: it was released already, or the thirty minutes it lives ran out. " +
	"Nothing was sent. Say the write again to the person and make the original call once more."

// mcpRelease is one call of one confirm tool.
//
// The order is: the card read, the hold found, the words and the gap judged, the
// sign-in, and only then the hold taken out and the write sent. The hold goes
// before the write runs, so one approval sends one write whatever the write
// itself answers: a send that failed on the wire is a write the person asks for
// again, not a card they can press twice.
func mcpRelease(ctx context.Context, id string, args json.RawMessage, errOut io.Writer, ch *mcpChat) mcp.Result {
	card, err := mcpCard(args)
	if err != nil {
		return mcpEnvelope(emit.Result{OK: false, Error: err.Error()})
	}
	at := now()
	held := ch.holds.hold(id, at)
	if held == nil {
		return mcpEnvelope(emit.Result{OK: false, Error: mcpHoldGone})
	}
	if err := held.Release(card, ch.lastCall(), at); err != nil {
		return mcpKeptAnswer(*held, err.Error())
	}
	c, err := mcpWriteNamed(held.Tool)
	if err != nil {
		return mcpEnvelope(emit.Result{OK: false, Error: err.Error()})
	}
	// The token before the hold is spent: a release nobody is signed in for
	// could send nothing, and burning the approval for it would cost the person
	// the card as well as the write.
	if err := mcpSignedIn(); err != nil {
		return mcpEnvelope(emit.Result{OK: false, Error: err.Error()})
	}
	ch.holds.release(id)
	return mcpSend(ctx, c, held.Args, errOut, ch, false)
}

// mcpCard is the four words a release sends back, read out of the call.
//
// Every one of them is required and read here rather than taken on trust from
// the schema, the way every other argument of this server is.
func mcpCard(args json.RawMessage) (chat.Card, error) {
	given, err := mcpArguments(args)
	if err != nil {
		return chat.Card{}, err
	}
	var card chat.Card
	fields := []struct {
		prop string
		into *string
	}{
		{holdProp, &card.Hold},
		{titleProp, &card.Title},
		{reasonProp, &card.Reason},
		{textProp, &card.Text},
	}
	known := map[string]bool{}
	for _, f := range fields {
		known[f.prop] = true
		value, err := mcpRequired(given, f.prop)
		if err != nil {
			return chat.Card{}, err
		}
		*f.into = value
	}
	for prop := range given {
		if !known[prop] {
			return chat.Card{}, fmt.Errorf("a confirm tool takes no argument called %q", prop)
		}
	}
	return card, nil
}

// mcpWriteNamed is the table tool the hold was made for. A hold naming a tool
// this binary does not offer is a hold nothing can send, and saying so is better
// than sending something else.
func mcpWriteNamed(tool string) (mcpCommand, error) {
	for _, c := range mcpCommands() {
		if c.tool == tool {
			return c, nil
		}
	}
	return mcpCommand{}, fmt.Errorf("the hold is for %q, which this gdoc has no tool for, so nothing was sent", tool)
}

// mcpTimed is one tool with the session's clock around it: the holds it keeps
// swept before the call, so a card whose thirty minutes ran out leaves the list
// even where nobody tries to release it, and the instant the call ended recorded
// after it, which is what the quiet gap is measured from.
//
// Every tool of the session goes through here, the reads included, because what
// the gap asks is whether the session was silent and not whether it was silent
// of one kind of call: TestAReleaseInsideTheQuietGapIsRefusedAndTheHoldKept.
func mcpTimed(ch *mcpChat, t mcp.Tool) mcp.Tool {
	call := t.Call
	t.Call = func(ctx context.Context, args json.RawMessage) mcp.Result {
		ch.holds.sweep(now())
		defer func() { ch.touch(now()) }()
		return call(ctx, args)
	}
	return t
}
