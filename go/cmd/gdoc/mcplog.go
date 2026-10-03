// One line per tool call on stderr, which Claude Desktop keeps as the
// extension's log.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sync"
	"time"

	"gdoc/internal/mcp"
)

// mcpQuoted is a double-quoted run inside an error sentence. gdoc's errors
// quote what they are about, and what they are about is often a document's own
// words or the text a person meant to post, so the log keeps the sentence and
// drops the quote.
var mcpQuoted = regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)

// mcpLogged wraps one tool so every call leaves one line in the log: the tool,
// whether its envelope said ok, the hold rule where the write was held, the
// error with its quoted words taken out, and how long it took.
//
// Claude Desktop's own log records that a call happened and how many text
// items came back, and nothing else, so on the first run in it a failed call
// could not be explained after the fact. The line never carries a document's
// text, a reply's words, a token or an argument:
// TestEveryCallLogsOneLineWithTheToolAndOk,
// TestAFailedCallLogsItsErrorWithTheQuotedWordsTakenOut and
// TestAHeldCallLogsItsRuleAndNotItsText.
func mcpLogged(log io.Writer, t mcp.Tool) mcp.Tool {
	call := t.Call
	name := t.Name
	t.Call = func(ctx context.Context, args json.RawMessage) mcp.Result {
		start := time.Now()
		res := call(ctx, args)
		fmt.Fprintf(log, "%s in %dms\n", mcpLogLine(name, res.Texts), time.Since(start).Milliseconds())
		return res
	}
	return t
}

// mcpLogLine reads the envelope out of a tool's answer, which is the first text
// item that is a JSON object carrying ok: a read answer opens with a fixed line
// before it, and the release line may follow it.
func mcpLogLine(name string, texts []string) string {
	for _, text := range texts {
		var env struct {
			OK    *bool  `json:"ok"`
			Error string `json:"error"`
			Data  struct {
				Held *struct {
					Rule string `json:"rule"`
				} `json:"held"`
			} `json:"data"`
		}
		if json.Unmarshal([]byte(text), &env) != nil || env.OK == nil {
			continue
		}
		line := fmt.Sprintf("gdoc mcp: %s ok=%t", name, *env.OK)
		if env.Data.Held != nil && env.Data.Held.Rule != "" {
			return line + " held=" + env.Data.Held.Rule
		}
		if env.Error != "" {
			line += " error=" + fmt.Sprintf("%q", mcpQuoted.ReplaceAllString(env.Error, `"…"`))
		}
		return line
	}
	return "gdoc mcp: " + name + " answered with no envelope"
}

// syncLog is the one log writer a session's goroutines share.
//
// Three of them write to it: the worker that runs a tool, the protocol's reader,
// and the sign-in listener, which outlives the call that started it. os.Stderr
// takes one Fprintf as one write, so a real session has never shown halves of
// two lines, and every writer a test hands in would:
// TestOneLogWriterTakesOneLineAtATime.
type syncLog struct {
	mu sync.Mutex
	to io.Writer
}

// lockedLog is one writer wrapped so that one line reaches it at a time.
func lockedLog(to io.Writer) io.Writer { return &syncLog{to: to} }

func (s *syncLog) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.to.Write(p)
}
