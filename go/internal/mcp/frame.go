// The framing: what counts as one message on the way in, what one answer looks
// like on the way out, and the error codes gdoc answers with.

package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
)

// maxLine is the ceiling on one incoming line, in bytes, counting the newline
// that ends it. A client that sends more has gone wrong, and a server that
// keeps appending would grow until the machine stopped.
// TestALineAtTheCeilingIsReadAndOneOverItIsNot states the literal.
const maxLine = 4 << 20

// The JSON-RPC codes. Only these five are ever answered.
const (
	codeParse          = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
)

// rpcError is the error member of one answer.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// response is one answer. ID is raw so the client's own bytes go back
// unchanged, and it has no omitempty because an answer always carries an id,
// null when nothing in the line could be read.
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// notification is one message out that asks for no answer. It has no id, which
// is what makes it a notification.
type notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
}

// message is one message in. hasID separates a request from a notification:
// a message with no id key is a notification, and id null is not, so the key's
// presence is what is recorded rather than the value's emptiness.
type message struct {
	id     json.RawMessage
	hasID  bool
	method string
	params json.RawMessage
}

// readLine reads one line and reports whether it went over the limit. An
// oversize line is thrown away as it is read and the reader lands on the next
// newline, so the caller can answer and go on.
//
// A last line with no newline of its own is still a line: it comes back with a
// nil error, and the call after it reports the end.
func readLine(r *bufio.Reader, limit int) (line []byte, over bool, err error) {
	var buf []byte
	for {
		chunk, rerr := r.ReadSlice('\n')
		if len(chunk) > 0 {
			if !over && len(buf)+len(chunk) > limit {
				over, buf = true, nil
			}
			if !over {
				buf = append(buf, chunk...)
			}
		}
		if rerr == bufio.ErrBufferFull {
			continue
		}
		if rerr != nil {
			if len(buf) == 0 && !over {
				return nil, false, rerr
			}
			return trimEOL(buf), over, nil
		}
		return trimEOL(buf), over, nil
	}
}

// trimEOL takes the newline off, and the carriage return before it, so a
// client that writes CRLF is read like any other.
func trimEOL(b []byte) []byte {
	b = bytes.TrimSuffix(b, []byte("\n"))
	return bytes.TrimSuffix(b, []byte("\r"))
}

// parseMessage reads one line. The error it gives back is the answer to send,
// and the message still carries the id when the id was readable, so a line
// that names no method is still answered to the right request.
func parseMessage(line []byte) (message, *rpcError) {
	if bytes.HasPrefix(bytes.TrimLeft(line, " \t"), []byte("[")) {
		return message{}, &rpcError{codeInvalidRequest, "a batch is not part of this protocol: send one message per line"}
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		if json.Valid(line) {
			return message{}, &rpcError{codeInvalidRequest, "a JSON-RPC message is a JSON object"}
		}
		return message{}, &rpcError{codeParse, "the line is not JSON: " + err.Error()}
	}
	var m message
	m.id, m.hasID = raw["id"]
	m.params = raw["params"]
	rawMethod, ok := raw["method"]
	if !ok {
		return m, &rpcError{codeInvalidRequest, "the message names no method"}
	}
	if err := json.Unmarshal(rawMethod, &m.method); err != nil {
		return m, &rpcError{codeInvalidRequest, "the method is not a string"}
	}
	return m, nil
}

// compactID is one id as bytes that can be compared. The id is echoed byte for
// byte, but a requestId in a cancellation is written by the client a second
// time, so the whitespace inside it is not the same whitespace.
func compactID(id json.RawMessage) []byte {
	trimmed := bytes.TrimSpace(id)
	if len(trimmed) == 0 {
		return nil
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, trimmed); err != nil {
		return trimmed
	}
	return buf.Bytes()
}
