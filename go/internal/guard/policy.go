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

// Level is how much a request may do to one file. It starts at 1 rather than
// at iota's 0 so that the zero value is no level at all: a Level nobody set
// must not read as permission to do anything.
type Level int

const (
	LevelSuggest Level = 1 // handed in: read, comment, suggest. Never direct-edit.
	LevelFull    Level = 2 // created by gdoc, or explicitly granted in-place.
)

// String names a level for a refusal message. A reader who has to translate a
// number is a reader who has not been told which rule refused them.
func (l Level) String() string {
	switch l {
	case LevelSuggest:
		return "suggest"
	case LevelFull:
		return "full"
	}
	return fmt.Sprintf("unknown(%d)", int(l))
}

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

// NewPolicy returns a policy that refuses everything. A command opens it one id
// at a time, so a command that forgot to say which document it is for gets a
// refusal rather than the whole of Drive.
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

// Judge is the whole policy in one function: it answers whether this method on
// this URL, carrying this body, may go out. Every request passes it, including
// the ones a redirect produces, and a refusal is an error the caller reports
// rather than a request it retries. Nothing else in gdoc decides what is
// reachable.
//
// The body is read only to tell a suggestion from a direct edit. An absent
// body is judged as no suggestion, which refuses rather than carries.
func (p *Policy) Judge(method string, u *url.URL, body []byte) error {
	if u.Scheme != "https" {
		return refuse("the URL scheme is %q, and gdoc makes https requests only. This request was not built by gdoc", u.Scheme)
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
		return refuse("%s %s is not carried on the token host, where the one call gdoc makes is POST /token", method, u.Path)
	case "docs.googleapis.com":
		return p.judgeDocs(method, u, body)
	case "www.googleapis.com":
		return p.judgeDrive(method, u)
	}
	return refuse("the host %q is not one gdoc talks to. It reaches docs.googleapis.com, www.googleapis.com and the token host, and nothing else", u.Host)
}

// plainPath refuses a path the guard would read differently from the way the
// transport sends it. Three shapes, one reason. It is the URL-level half of the
// invariant that checkWireMatchesJudgment holds at the request level.
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
// An opaque path is the third door. url.URL.RequestURI returns Opaque when it
// is set, so the transport writes Opaque and never looks at Path. The rules
// below read Path, so a known id in Path would carry a request to whatever
// Opaque names.
//
// None of the three occurs in a real Docs or Drive URL. Ids are [A-Za-z0-9_-],
// so refusing all of them costs nothing and closes the gap between what is
// judged and what is sent.
func plainPath(u *url.URL) error {
	if u.Opaque != "" {
		return refuse("the URL carries an opaque path %q, which is sent instead of %q", u.Opaque, u.Path)
	}
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
		return refuse("the path %q names no document. On the Docs host gdoc reaches /v1/documents/{id}, and nothing else", u.Path)
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
	return refuse("%s %s is not a call gdoc makes on a document. It reads with GET and writes with POST {id}:batchUpdate, and those are the two", method, u.Path)
}

// driveReads are the sub-resources a level-1 read may name under a known id.
// The plan's grammar is `{id}`, `{id}/export` and `{id}/comments*`, and
// everything else is refused: /permissions exposes collaborator identities,
// and /revisions is a second way to read a document's history.
var driveReads = map[string]bool{"": true, "export": true, "comments": true}

// filesCollection reports whether a Drive path names the files collection
// itself rather than a file under it. It is one grammar with two readers, and
// they must agree: judgeDrive reads it to decide a POST is a create, and
// isCreate reads it to decide the parent check runs. A path only one of them
// called a create would reach Drive with a parent nobody checked, which is one
// of principle 3's two doors left open.
func filesCollection(path string) bool {
	p := strings.TrimPrefix(path, "/upload")
	return p == "/drive/v3/files" || p == "/drive/v3/files/"
}

func (p *Policy) judgeDrive(method string, u *url.URL) error {
	path := strings.TrimPrefix(u.Path, "/upload")
	rest, ok := strings.CutPrefix(path, "/drive/v3/files")
	if !ok {
		return refuse("the path %q is outside /drive/v3/files, which is the only Drive collection gdoc reaches", u.Path)
	}
	// CutPrefix does not know about segment boundaries, so `/drive/v3/filesX`
	// would otherwise be judged as file X. The guard must not judge a path it
	// has misread, and the refusal says which of the two path rules stopped it.
	if rest != "" && !strings.HasPrefix(rest, "/") {
		return refuse("the path %q begins with /drive/v3/files but does not end that segment, so the guard would be reading it as a file it is not", u.Path)
	}
	if filesCollection(u.Path) { // the collection itself
		if method == "POST" && p.createFolder() != "" {
			// create, into the one named folder; transport verifies parent,
			// but only for the upload shapes it can read metadata out of.
			return checkUploadType(u)
		}
		return refuse("%s on the files collection is not carried: gdoc never lists Drive, and it carries a create only into the folder the command named", method)
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
		// metadata, export, comments, replies: reading is level 1. What comes
		// back is decided by the query as much as by the path, so the query is
		// judged too.
		return checkReadQuery(u)
	case (method == "POST" || method == "PATCH" || method == "DELETE") && sub == "comments":
		return nil // the comment surface, replies included, is part of LevelSuggest
	case method == "PATCH" && sub == "" && lvl == LevelFull:
		return nil // e.g. trashing a document gdoc created
	}
	return refuse("%s %s is not allowed at the %s level. A file handed in may be read, commented on and suggested on, and only a file gdoc created may be changed in place", method, u.Path, lvl)
}

// uploadTypes are the create shapes the guard can check. `multipart` and
// `resumable` both put the metadata where the parent check can read it: a
// multipart body opens with the metadata part, and a resumable start is the
// metadata on its own. An absent uploadType is a plain JSON create.
//
// `media` is the shape that is refused. It makes the whole body the file's
// content, so `{"parents":["FOLDER1"]}` is bytes to Drive and metadata to the
// parent check: the file lands unparented and the guard then learns its id at
// full level. Nothing here may learn an id from a create it could not verify,
// so an upload shape the guard cannot read is refused rather than carried.
var uploadTypes = map[string]bool{"multipart": true, "resumable": true}

// checkUploadType refuses a create whose body the parent check cannot read.
// The check is on the create only. An upload against a file already in the set
// creates nothing and teaches the guard nothing, so its level decides it.
func checkUploadType(u *url.URL) error {
	vals, present := u.Query()["uploadType"]
	if !present {
		return nil
	}
	if len(vals) != 1 || !uploadTypes[vals[0]] {
		return refuse("uploadType=%q is not a create the guard can check; it reads parents out of multipart and resumable metadata only", strings.Join(vals, ","))
	}
	return nil
}

// driveReadParams are the query parameters a Drive read may carry. It is an
// allowlist because the path is only half of what a GET asks for: the path
// names one file and the level says read, and the query decides how much of
// that file comes back.
var driveReadParams = map[string]bool{
	"alt":               true, // export asks for the bytes rather than the metadata
	"mimeType":          true, // which export format
	"fields":            true, // narrowed further by checkFields
	"pageSize":          true, // comments come back a page at a time
	"pageToken":         true,
	"includeDeleted":    true, // comments list
	"supportsAllDrives": true,
}

// blockedFields are the field names a read may not ask for. The guard refuses
// /permissions because it names who else can reach the document, and `fields`
// reaches the same data through a plain GET of the file.
var blockedFields = map[string]bool{"permissions": true, "permissionids": true}

func checkReadQuery(u *url.URL) error {
	for name, vals := range u.Query() {
		if !driveReadParams[name] {
			return refuse("the query parameter %q is not one a read carries", name)
		}
		if name != "fields" {
			continue
		}
		for _, v := range vals {
			if err := checkFields(v); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkFields reads a fields expression as its bare names. Drive separates
// them with commas, slashes and parentheses, so splitting on everything that
// is not a name character gives the list to check, and `permissions/role`
// cannot hide inside a sub-selection.
func checkFields(v string) error {
	if strings.Contains(v, "*") {
		return refuse("fields=%q asks for every field, and that includes the permission surface /permissions is refused to protect", v)
	}
	for _, name := range strings.FieldsFunc(v, notFieldRune) {
		if blockedFields[strings.ToLower(name)] {
			return refuse("fields=%q names %q, and who else can reach a document is refused however it is asked for", v, name)
		}
	}
	return nil
}

func notFieldRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		return false
	}
	return true
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
