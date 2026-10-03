// The one setting this server takes, wired: a value it could not read, the
// addresses it exempted, and the fact a write answer states about them.
//
// Nothing here suggests the setting. The field is in Claude Desktop's own
// settings for a person who went looking for it, and what the binary says about
// it is a fact: decision 17 of the milestone 14 specification, and
// TestNothingSuggestsTheSetting.

package main

import (
	"context"
	"encoding/json"
	"strings"

	"gdoc/internal/chat"
	"gdoc/internal/emit"
	"gdoc/internal/mcp"
)

// mcpSettingChecked is one tool that refuses while the setting is unreadable.
//
// A value gdoc half understands does not stop the server, because a server that
// does not start says nothing to the person: Claude Desktop shows them a
// connector that failed and no reason. So the session runs, the reason is on the
// log, and every tool but guide answers it instead of running:
// TestAMalformedValueMakesEveryToolButGuideNameIt.
//
// guide is left alone because it reaches nothing and because it is where the
// person reads what this session is running with, the bad value included. A
// confirm tool needs no wrapper: a hold exists only where a write ran, and no
// write runs while this is unread.
func mcpSettingChecked(ch *mcpChat, t mcp.Tool) mcp.Tool {
	if t.Name == "guide" || ch.trustedErr == nil {
		return t
	}
	refusal := ch.trustedErr.Error()
	t.Call = func(context.Context, json.RawMessage) mcp.Result {
		return mcpEnvelope(emit.Result{OK: false, Error: refusal})
	}
	return t
}

// mcpExempted is the addresses in one write that sit at a domain the person
// listed, read before the command runs so the answer states them whatever the
// write came back with.
//
// A read carries none, because the setting is about what gdoc writes. Arguments
// no write can be read out of carry none either: that call is refused above this
// line and the refusal is about the arguments.
func mcpExempted(c mcpCommand, args json.RawMessage, target mcpTarget, ch *mcpChat) []string {
	if c.readOnly || len(ch.trusted) == 0 {
		return nil
	}
	w, err := mcpWriteOf(c, args, target, ch.ledger)
	if err != nil {
		return nil
	}
	return chat.Exempted(w.Text, ch.trusted)
}

// mcpWriteAnswer is what a write tool answers: the envelope the terminal would
// have printed, and where the setting applied, one more text item saying so.
//
// The loosening is said out loud because it is the only one in gdoc, and a check
// nobody can see was loosened is a check nobody can audit. It is a fact and no
// judgement: which addresses were not asked about, and that a link is never one
// of them: TestTheWriteAnswerNamesTheExemption.
func mcpWriteAnswer(r emit.Result, exempt []string) mcp.Result {
	out := mcpEnvelope(r)
	if len(exempt) == 0 {
		return out
	}
	texts := make([]string, 0, len(out.Texts)+1)
	texts = append(texts, out.Texts...)
	texts = append(texts, mcpExemptionLine(exempt))
	return mcp.Result{Texts: texts, IsError: out.IsError}
}

// mcpExemptionLine is that one item. It names the addresses as they were
// written, because that is what the document carries.
func mcpExemptionLine(exempt []string) string {
	quoted := make([]string, 0, len(exempt))
	for _, address := range exempt {
		quoted = append(quoted, `"`+address+`"`)
	}
	return "This text holds " + strings.Join(quoted, ", ") +
		", at an email domain this computer's gdoc is set to need no approval for, so the person was not asked about it. " +
		"A link is never exempt, whatever its domain."
}
