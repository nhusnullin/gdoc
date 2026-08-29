// Package guard is the network policy. Principle 3: the client reaches only
// the files it was given, and every id carries a write level. This file is
// pure judgment; transport.go carries requests through it.
package guard

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
)

type Level int

const (
	LevelSuggest Level = 1 // handed in: read, comment, suggest. Never direct-edit.
	LevelFull    Level = 2 // created by gdoc, or explicitly granted in-place.
)

// Policy is the reachable set plus the one folder a create may target.
//
// It is guarded by a mutex because NewClient hands out an *http.Client, which
// the standard library documents as safe for concurrent use. Learn writes the
// set from inside RoundTrip while Judge reads it on every request and
// CheckRedirect reads it from another goroutine, so without the lock two
// parallel requests are a concurrent map read and write, which is a fatal
// error rather than a recoverable one.
type Policy struct {
	mu       sync.RWMutex
	files    map[string]Level
	createIn string   // folder id a create may target; empty means no creates
	warnings []string // things the guard could not do quietly, for the command to report
}

func NewPolicy() *Policy { return &Policy{files: map[string]Level{}} }

// AllowFile puts a handed-in id in the set at the level it was handed in at.
func (p *Policy) AllowFile(id string, lvl Level) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.files[id] = lvl
}

// AllowCreateIn names the one folder a create may target. The folder id is not
// added to the file set: creating in a folder is not a licence to read it, and
// the set has exactly two doors, the ids passed in and the ids learned from a
// create.
func (p *Policy) AllowCreateIn(folderID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.createIn = folderID
}

// GrantInPlace upgrades an id already in the set. It is not a third door.
func (p *Policy) GrantInPlace(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, known := p.files[id]; known {
		p.files[id] = LevelFull
	}
}

// Learn is the second door: an id that came back from a create the guard
// itself carried.
func (p *Policy) Learn(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.files[id] = LevelFull
}

// level reports the level of an id, and whether it is in the set at all.
func (p *Policy) level(id string) (Level, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	lvl, known := p.files[id]
	return lvl, known
}

// createFolder is the folder a create may target, empty when none was named.
func (p *Policy) createFolder() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.createIn
}

// note records something the guard could not do quietly. The alternative was
// silence: an id the guard failed to learn turns into "file was not given to
// this command" on the next request, which points the reader at the wrong
// problem.
func (p *Policy) note(format string, a ...any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.warnings = append(p.warnings, fmt.Sprintf(format, a...))
}

// Warnings returns what the guard could not do quietly, for the command to put
// on the envelope.
func (p *Policy) Warnings() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.warnings) == 0 {
		return nil
	}
	out := make([]string, len(p.warnings))
	copy(out, p.warnings)
	return out
}

func refuse(format string, a ...any) error {
	return fmt.Errorf("guard refused: "+format, a...)
}

func (p *Policy) Judge(method string, u *url.URL, body []byte) error {
	if u.Scheme != "https" {
		return refuse("scheme %q", u.Scheme)
	}
	// Userinfo in the URL becomes an Authorization header on the wire. gdoc
	// authenticates with its own token and nothing else, so a URL carrying
	// credentials is a URL the guard did not build.
	if u.User != nil {
		return refuse("the URL carries credentials")
	}
	if err := plainPath(u); err != nil {
		return err
	}
	switch u.Host {
	case "oauth2.googleapis.com":
		if method == "POST" && u.Path == "/token" {
			return nil
		}
		return refuse("%s %s on the token host", method, u.Path)
	case "docs.googleapis.com":
		return p.judgeDocs(method, u, body)
	case "www.googleapis.com":
		return p.judgeDrive(method, u)
	}
	return refuse("host %q", u.Host)
}

// plainPath refuses a path the guard would read differently from the way the
// transport sends it. Two shapes, one reason.
//
// A `..` segment lets an id that is in the set walk to an endpoint the rules
// below refuse outright: `/drive/v3/files/DOC1/../../../about` is judged as a
// read of DOC1 and arrives at the server as drive.about.get, which the guard
// declines when it is asked plainly. Go does not clean dot segments out of a
// URL, so the walk reaches the wire intact.
//
// A percent-encoded separator does the same by a different door. u.Path is
// decoded and u.EscapedPath is what goes out, so `DOC1%2F..%2Fabout` is one
// segment to the guard and three to whoever decodes it next.
//
// Neither shape occurs in a real Docs or Drive URL. Ids are [A-Za-z0-9_-], so
// refusing both costs nothing and closes the gap between what is judged and
// what is sent.
func plainPath(u *url.URL) error {
	if u.RawPath != "" {
		return refuse("path %q is percent-encoded; the guard judges plain paths only", u.EscapedPath())
	}
	for _, seg := range strings.Split(u.Path, "/") {
		if seg == "." || seg == ".." {
			return refuse("path %q walks through %q", u.Path, seg)
		}
	}
	return nil
}

func (p *Policy) judgeDocs(method string, u *url.URL, body []byte) error {
	rest, ok := strings.CutPrefix(u.Path, "/v1/documents/")
	if !ok || rest == "" {
		return refuse("docs path %q", u.Path)
	}
	id, verb, _ := strings.Cut(rest, ":")
	lvl, known := p.level(id)
	if !known {
		return refuse("document %q was not given to this command", id)
	}
	switch {
	case method == "GET" && verb == "":
		return nil
	case method == "POST" && verb == "batchUpdate":
		if lvl == LevelFull || isSuggestMode(body) {
			return nil
		}
		return refuse("direct edit of %q, which was handed in; only SUGGEST is allowed", id)
	}
	return refuse("%s %s", method, u.Path)
}

// driveReads are the sub-resources a level-1 read may name under a known id.
// The plan's grammar is `{id}`, `{id}/export` and `{id}/comments*`, and
// everything else is refused: /permissions exposes collaborator identities,
// and /revisions is a second way to read a document's history.
var driveReads = map[string]bool{"": true, "export": true, "comments": true}

func (p *Policy) judgeDrive(method string, u *url.URL) error {
	path := strings.TrimPrefix(u.Path, "/upload")
	rest, ok := strings.CutPrefix(path, "/drive/v3/files")
	if !ok {
		return refuse("drive path %q", u.Path)
	}
	// CutPrefix does not know about segment boundaries, so `/drive/v3/filesX`
	// would otherwise be judged as file X. The guard must not judge a path it
	// has misread.
	if rest != "" && !strings.HasPrefix(rest, "/") {
		return refuse("drive path %q", u.Path)
	}
	if rest == "" || rest == "/" { // the collection itself
		if method == "POST" && p.createFolder() != "" {
			return nil // create, into the one named folder; transport verifies parent
		}
		return refuse("%s on the files collection (listing and unparented creates)", method)
	}
	parts := strings.Split(strings.TrimPrefix(rest, "/"), "/")
	id := parts[0]
	lvl, known := p.level(id)
	if !known {
		return refuse("file %q was not given to this command", id)
	}
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	switch {
	case method == "GET" && driveReads[sub]:
		return nil // metadata, export, comments, replies: reading is level 1
	case (method == "POST" || method == "PATCH" || method == "DELETE") && sub == "comments":
		return nil // the comment surface, replies included, is part of LevelSuggest
	case method == "PATCH" && sub == "" && lvl == LevelFull:
		return nil // e.g. trashing a document gdoc created
	}
	return refuse("%s %s at level %d", method, u.Path, lvl)
}

// isSuggestMode reads the one field that keeps a handed-in document
// read-and-suggest only.
//
// Read this before trusting it. The bar is a field the client itself supplies,
// and docs/v2/BLOCKED-BY-API.md records the measurement: writeMode is absent
// from the public Docs discovery document, and one morning this exact call
// returned 200 and silently made a direct edit instead of a suggestion. So the
// server is not known to honour the field, and a body that says SUGGEST is a
// statement of intent rather than a guarantee.
//
// The capability probe the spec relies on, which would ask the server what it
// will do before the write goes out, does not exist yet. Until it does, the
// level-1 write bar rests on a client-supplied field. Widening or narrowing
// what this permits is a spec decision, not a refactor.
func isSuggestMode(body []byte) bool {
	var probe struct {
		WriteControl struct {
			WriteMode string `json:"writeMode"`
		} `json:"writeControl"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return false // a body the guard cannot read is not a suggestion
	}
	return probe.WriteControl.WriteMode == "SUGGEST"
}
