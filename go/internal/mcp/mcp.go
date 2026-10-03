package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// Tool is one tool the server offers. Schema is the inputSchema, written by
// hand so the order of its properties is the order a card shows them in.
// ReadOnly is the one judgement this package carries about a tool, and it goes
// out as readOnlyHint.
type Tool struct {
	Name        string
	Title       string
	Description string
	Schema      json.RawMessage
	ReadOnly    bool
	Call        func(ctx context.Context, args json.RawMessage) Result
}

// Result is what a tool answers. Each text is one content item, in order, and
// IsError is the tool saying the model should try again rather than the
// protocol saying the client is broken.
type Result struct {
	Texts   []string
	IsError bool
}

// Info is who the server says it is, and what it tells a client that shows
// instructions to its model.
type Info struct {
	Name         string
	Version      string
	Instructions string
}

// versions are the protocol versions gdoc speaks, newest first. A server with
// tools only sends the same shapes under all four, so the list is the whole of
// what version negotiation is.
var versions = []string{"2025-11-25", "2025-06-18", "2025-03-26", "2024-11-05"}

// Server is one stdio session. It knows tools as names and functions, and
// nothing about documents, commands or the wire.
type Server struct {
	info Info
	log  io.Writer

	toolsMu sync.Mutex
	tools   []Tool

	outMu sync.Mutex
	out   io.Writer

	qMu    sync.Mutex
	qCond  *sync.Cond
	queue  []*call
	closed bool
}

// call is one tools/call waiting for the worker. The tool is resolved when the
// line is read, so an unknown name is refused at once rather than behind
// whatever is running.
type call struct {
	id   json.RawMessage
	tool Tool
	args json.RawMessage
}

// New makes a server over the tools given, in their order. An empty version is
// a checkout build, which belongs to no release, and says dev rather than
// naming one.
func New(info Info, tools []Tool, log io.Writer) *Server {
	if info.Version == "" {
		info.Version = "dev"
	}
	s := &Server{info: info, log: log, tools: append([]Tool(nil), tools...)}
	s.qCond = sync.NewCond(&s.qMu)
	return s
}

// Add registers a tool and tells the client the list changed. A tool of the
// same name is replaced where it stands, so the order never moves under a
// client that is reading it.
func (s *Server) Add(t Tool) {
	s.toolsMu.Lock()
	replaced := false
	for i := range s.tools {
		if s.tools[i].Name == t.Name {
			s.tools[i], replaced = t, true
			break
		}
	}
	if !replaced {
		s.tools = append(s.tools, t)
	}
	s.toolsMu.Unlock()
	s.tellListChanged()
}

// Remove takes a tool out and tells the client the list changed. A name that
// is not there changes nothing and tells nobody.
func (s *Server) Remove(name string) {
	s.toolsMu.Lock()
	kept := make([]Tool, 0, len(s.tools))
	gone := false
	for _, t := range s.tools {
		if t.Name == name {
			gone = true
			continue
		}
		kept = append(kept, t)
	}
	s.tools = kept
	s.toolsMu.Unlock()
	if gone {
		s.tellListChanged()
	}
}

// Serve reads one message per line until the reader ends, and writes one
// compact line per answer. It returns once the calls that arrived have been
// answered: a call that reached the queue was asked for, and the end of stdin
// is not a reason to pretend it never came.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	s.outMu.Lock()
	s.out = out
	s.outMu.Unlock()

	worked := make(chan struct{})
	go func() {
		defer close(worked)
		s.work(ctx)
	}()
	defer func() {
		s.qMu.Lock()
		s.closed = true
		s.qMu.Unlock()
		s.qCond.Broadcast()
		<-worked
	}()

	r := bufio.NewReader(in)
	for {
		line, over, err := readLine(r, maxLine)
		switch {
		case over:
			s.answerError(nil, rpcError{codeInvalidRequest,
				fmt.Sprintf("the line is over the %d byte ceiling and was skipped", maxLine)})
		case len(bytes.TrimSpace(line)) > 0:
			s.handle(line)
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// work runs the queued calls one at a time, in the order their lines arrived.
// gapi.Session is not safe for concurrent use and the test seams in cmd/gdoc
// are package variables, so serial is what keeps every command as it is.
//
// It ends when the session is closed and the queue is empty, so the last call
// to arrive is answered rather than dropped.
func (s *Server) work(ctx context.Context) {
	for {
		s.qMu.Lock()
		for len(s.queue) == 0 && !s.closed {
			s.qCond.Wait()
		}
		if len(s.queue) == 0 {
			s.qMu.Unlock()
			return
		}
		c := s.queue[0]
		s.queue = s.queue[1:]
		s.qMu.Unlock()
		s.answerCall(ctx, c)
	}
}

// handle answers one line. Everything but a tool call is answered here, in the
// reader, because none of it waits on anything.
func (s *Server) handle(line []byte) {
	m, rerr := parseMessage(line)
	if rerr != nil {
		s.answerError(m.id, *rerr)
		return
	}
	if !m.hasID {
		// A notification is never answered, known or unknown. There is nothing
		// to answer it to.
		return
	}
	switch m.method {
	case "initialize":
		s.answerResult(m.id, s.initialize(m.params))
	case "ping":
		s.answerResult(m.id, json.RawMessage(`{}`))
	case "tools/list":
		s.answerResult(m.id, s.toolsList())
	case "tools/call":
		s.startCall(m)
	default:
		s.answerError(m.id, rpcError{codeMethodNotFound, "gdoc knows no method " + m.method})
	}
}

// initializeResult is the handshake answer. The field order here is the order
// it goes out in, and TestInitializeCarriesCapabilitiesInfoAndInstructions
// pins the bytes.
type initializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    capabilities `json:"capabilities"`
	ServerInfo      serverInfo   `json:"serverInfo"`
	Instructions    string       `json:"instructions,omitempty"`
}

type capabilities struct {
	Tools toolsCapability `json:"tools"`
}

// toolsCapability says the list can change while the session runs, which is
// what lets a hold register its own one-time confirm tool.
type toolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// initialize echoes a version gdoc speaks and answers the newest otherwise.
// Params that will not decode are params that named no version.
func (s *Server) initialize(params json.RawMessage) json.RawMessage {
	asked := ""
	if len(params) > 0 {
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(params, &p); err == nil {
			asked = p.ProtocolVersion
		}
	}
	answer := versions[0]
	for _, v := range versions {
		if v == asked {
			answer = v
			break
		}
	}
	return s.encode(initializeResult{
		ProtocolVersion: answer,
		Capabilities:    capabilities{Tools: toolsCapability{ListChanged: true}},
		ServerInfo:      serverInfo{Name: s.info.Name, Version: s.info.Version},
		Instructions:    s.info.Instructions,
	})
}

// toolJSON is one tool as the client reads it. Title goes out twice, at the
// top level for the versions that have it and in annotations for the ones that
// do not, so all four versions get the same words.
type toolJSON struct {
	Name        string          `json:"name"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
	Annotations annotations     `json:"annotations"`
}

// annotations carries the two hints. destructiveHint is always false because
// nothing gdoc does in chat takes anything away: a reply, a comment and a
// suggestion are each something added.
type annotations struct {
	Title           string `json:"title,omitempty"`
	ReadOnlyHint    bool   `json:"readOnlyHint"`
	DestructiveHint bool   `json:"destructiveHint"`
}

func (s *Server) toolsList() json.RawMessage {
	s.toolsMu.Lock()
	tools := append([]Tool(nil), s.tools...)
	s.toolsMu.Unlock()

	out := make([]toolJSON, 0, len(tools))
	for _, t := range tools {
		schema := t.Schema
		if !json.Valid(schema) {
			if len(schema) > 0 {
				s.logf("mcp: the schema of tool %s is not JSON; listing the empty object instead", t.Name)
			}
			schema = json.RawMessage(`{"type":"object"}`)
		}
		out = append(out, toolJSON{
			Name:        t.Name,
			Title:       t.Title,
			Description: t.Description,
			InputSchema: schema,
			Annotations: annotations{Title: t.Title, ReadOnlyHint: t.ReadOnly},
		})
	}
	return s.encode(struct {
		Tools []toolJSON `json:"tools"`
	}{out})
}

// callParams is what a tools/call carries.
type callParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// callResult is what one tool call answers with.
type callResult struct {
	Content []content `json:"content"`
	IsError bool      `json:"isError"`
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// startCall resolves the tool and queues the call. An unknown name is invalid
// params, answered at once, so a model that guessed a name hears it back
// without waiting for whatever is running.
func (s *Server) startCall(m message) {
	var p callParams
	if len(m.params) > 0 {
		if err := json.Unmarshal(m.params, &p); err != nil {
			s.logf("mcp: the params of a tools/call would not decode: %v", err)
		}
	}
	s.toolsMu.Lock()
	var found *Tool
	for i := range s.tools {
		if s.tools[i].Name == p.Name {
			t := s.tools[i]
			found = &t
			break
		}
	}
	s.toolsMu.Unlock()
	if found == nil {
		s.answerError(m.id, rpcError{codeInvalidParams, "gdoc offers no tool named " + p.Name})
		return
	}

	s.qMu.Lock()
	defer s.qMu.Unlock()
	if s.closed {
		return
	}
	s.queue = append(s.queue, &call{id: m.id, tool: *found, args: p.Arguments})
	s.qCond.Signal()
}

// answerCall runs one tool and answers with what it said.
func (s *Server) answerCall(ctx context.Context, c *call) {
	res := c.tool.Call(ctx, c.args)
	items := make([]content, 0, len(res.Texts))
	for _, text := range res.Texts {
		items = append(items, content{Type: "text", Text: text})
	}
	s.answerResult(c.id, s.encode(callResult{Content: items, IsError: res.IsError}))
}

func (s *Server) answerResult(id json.RawMessage, result json.RawMessage) {
	s.send(response{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *Server) answerError(id json.RawMessage, e rpcError) {
	s.send(response{JSONRPC: "2.0", ID: id, Error: &e})
}

// tellListChanged says the list moved. Before Serve there is no writer, so an
// Add while the server is being built tells nobody and simply lists.
func (s *Server) tellListChanged() {
	s.send(notification{JSONRPC: "2.0", Method: "notifications/tools/list_changed"})
}

// send writes one compact line. Every writer in the process goes through here,
// so the reader and the worker never interleave halves of two messages.
//
// HTML escaping is off: the text gdoc puts in a result carries the angle
// brackets of a label and the ampersands of a document, and a reader of the
// log should see them as they are.
func (s *Server) send(v any) {
	s.outMu.Lock()
	defer s.outMu.Unlock()
	if s.out == nil {
		return
	}
	enc := json.NewEncoder(s.out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		s.logf("mcp: one line could not be written: %v", err)
	}
}

// encode turns a result into raw JSON for the response to carry. A result that
// will not encode is a bug in the caller, and an empty object keeps the
// session alive to report it.
func (s *Server) encode(v any) json.RawMessage {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		s.logf("mcp: a result would not encode: %v", err)
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(bytes.TrimSuffix(buf.Bytes(), []byte("\n")))
}

// logf writes one line to the log, where there is one. The log is never
// stdout: stdout carries JSON-RPC and nothing else.
func (s *Server) logf(format string, args ...any) {
	if s.log == nil {
		return
	}
	fmt.Fprintf(s.log, format+"\n", args...)
}
