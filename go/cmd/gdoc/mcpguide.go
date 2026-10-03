// The guide tool: the rules a chat reviews a document by, and the code every
// other tool needs.
//
// Two files are embedded here. chatheader.md is the chat's own page: tools
// instead of a shell, short spoken lists, no live mode, and what a marker means
// where there is no hub. review.md is the review core, byte for byte the file
// the gdoc-review skill reads, so a session through either front door judges a
// thread the same way: TestTheEmbeddedCoreIsTheSkillsCore.
//
// It is a copy and not a link, because go:embed cannot reach outside the
// package directory and the binary a colleague installs carries no skills
// folder. The test is what keeps the two from drifting.

package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"

	"gdoc/internal/emit"
	"gdoc/internal/mcp"
)

//go:embed chatheader.md
var mcpChatHeader string

//go:embed review.md
var mcpReviewCore string

// mcpGuideData is what guide answers with.
//
// The code is the one every other tool but login needs. The trusted email
// domains are the setting this process is running with, read from the running
// server and never from the repository: a person who typed one into Claude
// Desktop can see here that gdoc took it. It is a fact and nothing suggests
// setting it, which is decision 17 of the milestone 14 specification and what
// TestNothingSuggestsTheSetting holds.
type mcpGuideData struct {
	Code string `json:"code"`
	// TrustedEmailDomains is the setting as this session parsed it: whole
	// lowercased domains, and an empty list where the field was empty, which is
	// the ordinary case.
	TrustedEmailDomains []string `json:"trusted_email_domains"`
	// TrustedEmailDomainsProblem is why a value was not read, where it was not.
	// guide is the one tool that still answers then, so it is the one place the
	// person can read it from: TestAMalformedValueMakesEveryToolButGuideNameIt.
	TrustedEmailDomainsProblem string `json:"trusted_email_domains_problem,omitempty"`
	Text                       string `json:"text"`
}

// mcpGuideTool is the one tool a session starts with. It reaches nothing: no
// token, no wire and no document, so it answers whether or not anybody is
// signed in, which is what lets the rules arrive before the sign-in does.
func mcpGuideTool(ch *mcpChat) mcp.Tool {
	return mcp.Tool{
		Name:        "guide",
		Title:       "How to review a Google Doc with gdoc",
		Description: "Call this first. It gives the rules for reviewing a Google Doc with gdoc, and the code every other tool needs.",
		Schema:      json.RawMessage(noArguments),
		ReadOnly:    true,
		Call: func(context.Context, json.RawMessage) mcp.Result {
			return mcpGuideAnswer(ch)
		},
	}
}

// mcpGuideAnswer is the envelope guide prints, the same shape every other tool
// answers with: the header and the core as one text, so the model reads them in
// the order they are written.
func mcpGuideAnswer(ch *mcpChat) mcp.Result {
	problem := ""
	if ch.trustedErr != nil {
		problem = ch.trustedErr.Error()
	}
	return mcpEnvelope(emit.Result{OK: true, Data: mcpGuideData{
		Code:                       ch.code.Value(),
		TrustedEmailDomains:        ch.trusted,
		TrustedEmailDomainsProblem: problem,
		Text:                       mcpGuideText(),
	}})
}

// mcpGuideText is the header and the core, in that order. The header says what
// this chat can do and the core says how a thread is judged, and the header is
// what tells the model to read on.
func mcpGuideText() string {
	return strings.TrimRight(mcpChatHeader, "\n") + "\n\n" + mcpReviewCore
}
