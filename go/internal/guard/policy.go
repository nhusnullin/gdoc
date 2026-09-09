// Package guard is the network policy. Principle 3: the client reaches only
// the files it was given, and every id carries a write level. This file is
// pure judgment; transport.go carries requests through it.
package guard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
)

// Level is how much a request may do to one file. It starts at 1 rather than
// at iota's 0 so that the zero value is no level at all: a Level nobody set
// must not read as permission to do anything.
type Level int

// The levels are names, not a ladder. Every comparison in this file is ==,
// never >=, so a level carries exactly what its own rules say and nothing it
// happens to sort above. That is why LevelInPlace, which is the newest and the
// one a restyle opens, still gets no Drive file PATCH: LevelFull has that rule
// and LevelInPlace is not a bigger LevelFull. Renumbering these constants must
// not change what any of them may do.
const (
	LevelSuggest Level = 1 // handed in: read, comment, suggest. Never direct-edit.
	LevelFull    Level = 2 // created by gdoc. The only door to it is Learn.
	LevelInPlace Level = 3 // handed in and granted for one run: direct edit, styling kinds only. The only door to it is GrantInPlace.
)

// String names a level for a refusal message. A reader who has to translate a
// number is a reader who has not been told which rule refused them.
func (l Level) String() string {
	switch l {
	case LevelSuggest:
		return "suggest"
	case LevelFull:
		return "full"
	case LevelInPlace:
		return "in place"
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
	createIn string          // folder id a create may target; empty means no creates
	copyFrom string          // the one file files.copy may duplicate; empty means no copies
	rejects  map[string]bool // suggestion ids a rejectSuggestion may name; empty means none
	warnings []string        // things the guard could not do quietly, for the command to report
}

// NewPolicy returns a policy that refuses everything. A command opens it one id
// at a time, so a command that forgot to say which document it is for gets a
// refusal rather than the whole of Drive.
func NewPolicy() *Policy { return &Policy{files: map[string]Level{}, rejects: map[string]bool{}} }

// AllowReject names one suggestion a rejectSuggestion request may act on.
//
// Nail's decision, 2026-09-07: gdoc may reject a suggestion the note records as
// its own, because that is the only request that retracts a whole replace
// proposal. The guard cannot tell whose a suggestion is, so the permission is
// provenance, read from the note by the withdraw command, and this is how the
// command hands it in for one run. It is one id, not the verb: acceptSuggestion
// and deleteSuggestion stay refused whatever id they name, and a
// rejectSuggestion naming any other id is refused too. Nothing else gdoc does
// grants this, and the grant dies with the process.
func (p *Policy) AllowReject(suggestionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if suggestionID != "" {
		p.rejects[suggestionID] = true
	}
}

// mayReject reports whether a rejectSuggestion may name this id.
func (p *Policy) mayReject(suggestionID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.rejects[suggestionID]
}

// AllowFile puts a handed-in id in the set at the level it was handed in at,
// and it cannot put one in at LevelInPlace.
//
// That refusal is the point rather than a tidiness. GrantInPlace is where the
// decision to direct-edit somebody's document is made, and it holds an
// invariant: it upgrades an id that is already in the set and admits none. A
// function taking any Level is a second way to write the in-place level into
// the set, reaching it without that invariant and without the one line a
// reviewer reads. So the level is refused here.
//
// It fails closed rather than loudly: the id is admitted at no level at all, so
// every request naming it is refused, and the refusal is recorded for the
// command to put on the envelope. A panic was the other option and would have
// been the wrong direction to be wrong in, because cmd/gdoc recovers a panic
// into the same JSON object; failing closed with a warning says which line to
// call instead, and it says it before any request is judged.
func (p *Policy) AllowFile(id string, lvl Level) {
	if lvl == LevelInPlace {
		p.note("a file may not be handed in at the in-place level; nothing was added to the reachable set for %q. Direct edit is opened by GrantInPlace on an id already in the set, and that is the one line that opens it", id)
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.files[id] = lvl
}

// GrantInPlace upgrades an id already in the set to LevelInPlace, which permits
// a batchUpdate without SUGGEST on that one document, and only for the request
// kinds inPlaceKinds names. It is not a third door: an id nobody handed in is
// not admitted by asking to style it.
//
// The grant is one id and one run. Nothing writes it down, nothing reads it
// from a file, and no flag turns it on for every document: it dies with the
// process, in the shape AllowReject and AllowCreateIn already have. Nail's
// decision of 2026-09-09, M7b. It is also the widest thing gdoc can be asked to
// do, so the caller is the most security-relevant line in the milestone.
//
// What it does not raise is the Drive file. judgeDrive carries a PATCH at
// LevelFull and compares levels by name, so a restyle cannot trash or rename
// the document it is styling.
func (p *Policy) GrantInPlace(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, known := p.files[id]; known {
		p.files[id] = LevelInPlace
	}
}

// AllowCopy names the one file files.copy may duplicate, into the folder
// AllowCreateIn named. It is per-run and one source, in the shape AllowReject
// and AllowCreateIn already have: nothing writes it down, nothing reads it from
// a file, and it dies with the process. A second call replaces the first,
// because the grant is one source and a caller naming two has made a mistake
// the guard must not turn into two reachable duplicates.
//
// Nail's decision, 2026-09-09, and it is recorded as a decision rather than as
// M2's production-caller rule satisfied. Two things about it are worth reading
// before it is widened.
//
// Its only caller is the live acceptance test, not a command. M2's rule is that
// a guard door with no production caller is deleted rather than carried, which
// is why GrantInPlace went and came back with its own caller. This one is kept
// on the strength of the measurement it makes possible: the ten-feature
// preservation run needs a document holding an anchored comment, a pending
// suggestion, an image and a Drawing, and a throwaway probe showed the first
// two can be built from scratch through the API while the last two cannot. The
// probe was tools/copyprobe, deleted at M7b once it had answered; the
// measurement it made is in docs/v2/BLOCKED-BY-API.md. Copying
// an ideal document is how the run stops being a thing somebody does by hand in
// a browser.
//
// And a copy is the widest reach a handed-in id has ever produced. It takes a
// full duplicate of somebody's document into gdoc's own folder, comments and
// pending suggestions included, where learnFromCreate puts it at LevelFull.
// Nothing about the source changes, and the guard's whole claim is about what
// is reachable, so this is a decision to record rather than a detail.
func (p *Policy) AllowCopy(sourceID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.copyFrom = sourceID
}

// mayCopy reports whether files.copy may duplicate this file. An empty id is
// never granted, so a policy nobody called AllowCopy on carries no copy.
func (p *Policy) mayCopy(id string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return id != "" && p.copyFrom == id
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
// The body is read for two things: to tell a suggestion from a direct edit, and
// to see the request kinds a batchUpdate carries. An absent body is judged as
// no suggestion and as a batchUpdate the guard could not read, and both of
// those refuse rather than carry.
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
			// The exchange is a form in the body. Nothing gdoc sends puts any
			// of it in the query.
			return checkQuery(u, noParams)
		}
		return refuse("%s %s is not carried on the token host, where the one call gdoc makes is POST /token", method, u.Path)
	case "docs.googleapis.com":
		return p.judgeDocs(method, u, body)
	case "www.googleapis.com":
		return p.judgeDrive(method, u, body)
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
		return checkQuery(u, docsReadParams)
	case method == "POST" && verb == "batchUpdate":
		if err := p.judgeRequests(body, lvl); err != nil {
			return err
		}
		if lvl == LevelFull || lvl == LevelInPlace || isSuggestMode(body) {
			return checkQuery(u, noParams)
		}
		return refuse("direct edit of %q, which was handed in at the suggest level; only SUGGEST is allowed there, and a direct edit needs the in-place grant this run did not open", id)
	}
	return refuse("%s %s is not a call gdoc makes on a document. It reads with GET and writes with POST {id}:batchUpdate, and those are the two", method, u.Path)
}

// driveShape names the Drive method a path under a known file id addresses, and
// returns "" for a path that is not one of them. The plan's grammar is `{id}`,
// `{id}/export` and `{id}/comments*`, and everything else is refused:
// /permissions exposes collaborator identities, and /revisions is a second way
// to read a document's history.
//
// The segment count is part of the name, and that is the whole point of this
// function. Reading only the first sub-segment judges `{id}/export/anything` as
// files.export and `{id}/comments/C1/permissions` as the comment surface, so a
// known id walks to a path nobody decided about while the guard reads the
// prefix it recognises.
//
// An empty segment is a segment. `{id}//permissions` splits into three parts
// with the middle one empty, and a check that reads an empty sub-resource as
// "none given" judges it as a plain metadata read. What Drive's router does
// with the doubled slash is not a question the guard answers by guess, so every
// empty segment is refused below before this is called.
func driveShape(parts []string) string {
	switch {
	case len(parts) == 1:
		return "file"
	case len(parts) == 2 && parts[1] == "export":
		return "export"
	case len(parts) == 2 && parts[1] == "copy":
		return "copy" // files.copy, and only under the grant AllowCopy opens
	case len(parts) == 2 && parts[1] == "comments":
		return "comments" // comments.list
	case len(parts) == 3 && parts[1] == "comments":
		return "comment" // comments.get, update, delete
	case len(parts) == 4 && parts[1] == "comments" && parts[3] == "replies":
		return "replies" // replies.list, create
	case len(parts) == 5 && parts[1] == "comments" && parts[3] == "replies":
		return "reply" // replies.get, update, delete
	}
	return ""
}

// commentWrites are the writes the guard carries on the comment surface, by
// shape and method. Creating is a POST on the collection, and the same call
// repeats one level down for replies. A method Drive does not define on a
// shape is refused: the guard carries the calls gdoc makes, and nothing else.
var commentWrites = map[string]map[string]bool{
	"comments": {"POST": true},
	"replies":  {"POST": true},
}

// commentItemWrites are the two methods Drive defines on one comment or one
// reply, and the guard carries neither. That is finding 3 of the M1 review.
//
// The reason is what the guard can see. A path names a comment id, and nothing
// in the id says who wrote it, so PATCH on C1 is as likely to rewrite somebody
// else's words as gdoc's own. No command gdoc has changes or removes a comment:
// a proposal withdrawn leaves its own comment in place and replies to it, which
// is a POST. So the surface stays as narrow as its callers, and a milestone
// that needs either method adds it back here beside the caller that needs it,
// the way GrantInPlace came back at M7b beside cmdRestyle's grant.
var commentItemWrites = map[string]bool{"PATCH": true, "DELETE": true}

// driveReadParamsFor is the query allowlist for one read shape. Each Drive
// method carries its own parameters, so one shared list would put paging on a
// metadata read and an export format on a comment listing, and neither is a
// call Drive has.
func driveReadParamsFor(shape string) map[string]bool {
	switch shape {
	case "file":
		return driveGetParams
	case "export":
		return driveExportParams
	case "comments":
		return driveCommentListParams
	case "replies":
		return driveReplyListParams
	case "comment", "reply":
		return driveCommentGetParams
	}
	return nil
}

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

// filesCopy names the file a Drive path asks to duplicate, and reports whether
// the path is files.copy at all. It is filesCollection's rule for the second
// create shape, and it exists for the same reason: judgeDrive reads it through
// driveShape to decide a POST is a copy, and isCreate reads it to decide the
// parent check and the id learning run. A path only one of them called a create
// would reach Drive with a parent nobody checked and come back with an id
// nobody learned, so the duplicate would be unreachable and the run could not
// even trash it.
//
// The /upload prefix is stripped here because judgeDrive strips it too. A
// reader that stripped it in one place only is exactly the disagreement this
// function exists to prevent.
func filesCopy(path string) (string, bool) {
	p := strings.TrimPrefix(path, "/upload")
	rest, ok := strings.CutPrefix(p, "/drive/v3/files/")
	if !ok {
		return "", false
	}
	id, tail, found := strings.Cut(rest, "/")
	if !found || id == "" || tail != "copy" {
		return "", false
	}
	return id, true
}

func (p *Policy) judgeDrive(method string, u *url.URL, body []byte) error {
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
			return checkQuery(u, driveCreateParams)
		}
		return refuse("%s on the files collection is not carried: gdoc never lists Drive, and it carries a create only into the folder the command named", method)
	}
	parts := strings.Split(strings.TrimPrefix(rest, "/"), "/")
	for _, seg := range parts {
		if seg == "" {
			return refuse("the path %q carries an empty segment, and the guard reads a doubled slash as nothing rather than guessing what Drive's router makes of it", u.Path)
		}
	}
	id := parts[0]
	lvl, known := p.level(id)
	if !known {
		return refuse("file %q was not given to this command", id)
	}
	shape := driveShape(parts)
	switch {
	case method == "GET" && driveReadParamsFor(shape) != nil:
		// metadata, export, comments, replies: reading is level 1. What comes
		// back is decided by the query as much as by the path, so the query is
		// judged too, per shape.
		return checkQuery(u, driveReadParamsFor(shape))
	case commentWrites[shape][method]:
		// the comment surface, replies included, is part of LevelSuggest
		if err := checkCommentWrite(body); err != nil {
			return err
		}
		return checkQuery(u, driveWriteParams)
	case commentItemWrites[method] && (shape == "comment" || shape == "reply"):
		return refuse("%s on one %s is not carried: nothing in the path says whose comment this is, so the guard carries no comment or reply PATCH or DELETE at all. A later milestone adds it back beside the caller that needs it", method, shape)
	case method == "POST" && shape == "copy":
		// files.copy, under the two grants it needs and not otherwise. The
		// source is the one AllowCopy named, and the duplicate lands in the one
		// folder AllowCreateIn named, which the transport's parent check
		// enforces on the body. Neither grant is a door into the set: the id
		// above was already known, and the copy's own id is learned from the
		// answer the way every other create's is.
		if !p.mayCopy(id) {
			return refuse("copying %q is not something this command was granted. A handed-in file is read, commented on and suggested on; duplicating one takes its comments and its pending suggestions into gdoc's own folder, so it needs the per-run grant naming exactly that source", id)
		}
		if p.createFolder() == "" {
			return refuse("copying %q was granted but no folder was named for the copy to land in, and a copy with nowhere to go lands beside the original, in a folder this command was never given", id)
		}
		return checkQuery(u, driveCopyParams)
	case method == "PATCH" && shape == "file" && lvl == LevelFull:
		// e.g. trashing a document gdoc created
		return checkQuery(u, driveWriteParams)
	}
	return refuse("%s %s is not allowed at the %s level. A file handed in may be read, commented on and suggested on; the in-place grant reaches the styling inside one document and never its Drive file, so only a file gdoc created may be patched here", method, u.Path, lvl)
}

// checkCommentWrite refuses a write that closes or reopens somebody's thread.
// SPEC.md's Never list: "Never resolve or reopen a comment thread." The whole
// comment surface is writable at LevelSuggest, so the path cannot tell a reply
// from a resolve and the body is the only thing that can.
//
// Drive spells both as one field, `action` on a reply
// (https://developers.google.com/workspace/drive/api/reference/rest/v3/replies),
// and the refusal is the whole field rather than the two values. gdoc sets no
// action at all, so an action nobody has decided about is refused the way an
// unknown query parameter is. `resolved` on the comment itself is output only:
// Drive sets it from the reply action, and there is nothing to check there.
//
// The field is refused by its presence, not by its value. `{"action":""}` and
// `{"action":null}` are the field, spelled two ways that a check on the decoded
// string reads as absent, and what Drive does with either is a question the
// guard would be answering by guess. The body is decoded into raw fields for
// that reason: a string field cannot tell an absent key from a present empty
// one. The names are compared with case folded, because encoding/json matches
// them that way, so `{"Action":"resolve"}` decodes into the same field.
//
// A body that cannot be read is refused rather than carried. A comment write is
// above a read, so not knowing must not resolve to sending it. That covers a
// body longer than the transport's peek, which arrives here truncated, and a
// body that is not an object at all.
// One more rule on this surface was considered and left out on purpose, so it
// is written down here rather than found missing later. SPEC.md's Never list
// also says "Never write a comment or reply that does not open with the robot
// emoji." The guard could read `content` and check the prefix. It does not, and
// the rule belongs where the reply body is built.
//
// The reason is the difference between that field and this one. `action` is a
// field gdoc never sets, so refusing its presence can never be wrong. `content`
// is a field gdoc always sets, and the check would be on its value, in human
// prose. That is where a guard can refuse real work, and it has an exception
// the guard cannot see: text gdoc did not author. v1 learned it, and CLAUDE.md
// records it. gdoc/comments.py carries a copied comment verbatim and unmarked,
// because the marker means "gdoc wrote this" and stamping it on somebody else's
// words claims authorship of text gdoc only moved. v2 plans no comment copy
// today, checked against SPEC.md's restyle section and PLAN.md's M7b: in-place
// restyle leaves the threads alone and `--new` loses them. But a prefix rule
// enforced here would have to be unpicked the day that changes, and the guard
// would be the last place anybody looked.
//
// The two rules also differ in what they cost when broken. Accepting somebody's
// suggestion is irreversible harm to their work, which is why judgeRequests
// refuses it here. A reply missing its marker is a breach of a reader
// convention, fixable by the next reply. The guard holds the first kind.
func checkCommentWrite(body []byte) error {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil // no body at all, and so no action
	}
	if err := hasDuplicateKeys(body); err != nil {
		return err
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(body, &probe); err != nil {
		return refuse("the body of this comment write cannot be read, so a reply cannot be told from a resolve: %v", err)
	}
	for name := range probe {
		if strings.EqualFold(name, "action") {
			return refuse("%q on a comment thread is refused: gdoc never resolves or reopens somebody's thread, so it sends no action field at all", name)
		}
	}
	return nil
}

// isSuggestMode reads the one field that keeps a handed-in document
// read-and-suggest only.
//
// Read this before trusting it. The bar is a field the client itself supplies,
// and docs/v2/BLOCKED-BY-API.md records the measurement: writeMode is absent
// from the public Docs discovery document, and one morning this exact call
// returned 200 and silently made a direct edit instead of a suggestion.
// Measured again on 2026-09-07, on a throwaway document: an insertText at
// index 30 in SUGGEST mode came back 200 and the read-back carried
// suggest.xyh4cb4emh7y, so the project is enrolled today. Enrolled today is not
// a guarantee for tomorrow, and the earlier measurement is what says so. A body
// that says SUGGEST is a statement of intent; what makes a write trustworthy is
// the capability probe before it and the read-back after it, both of which live
// above this package.
//
// The two keys are read exactly, and that is finding 1 of the M1 review.
// Google's proto-JSON is case-sensitive, so `WRITEMODE` is not the field the
// server reads. encoding/json matches field names with case folded, so the
// struct this used to unmarshal into read that body as a suggestion while Docs
// would have read it as a direct edit. The guard must never be broader than the
// server on the one field that permits a write.
//
// Two keys that fold to the same name are refused for the same reason: which
// one the server takes is not decided here. A key repeated exactly is refused
// one layer up, by hasDuplicateKeys.
//
// Widening or narrowing what this permits is a spec decision, not a refactor.
func isSuggestMode(body []byte) bool {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		return false // a body the guard cannot read is not a suggestion
	}
	control, ok := exactKey(top, "writeControl")
	if !ok {
		return false
	}
	var inner map[string]json.RawMessage
	if err := json.Unmarshal(control, &inner); err != nil {
		return false
	}
	raw, ok := exactKey(inner, "writeMode")
	if !ok {
		return false
	}
	var mode string
	if err := json.Unmarshal(raw, &mode); err != nil {
		return false
	}
	return mode == "SUGGEST"
}

// exactKey returns the value stored under exactly name. It reports false when
// the key is absent, and when another key in the same object folds to the same
// name: the server reads one of them and the guard would be reading the other.
func exactKey(m map[string]json.RawMessage, name string) (json.RawMessage, bool) {
	v, ok := m[name]
	if !ok {
		return nil, false
	}
	for k := range m {
		if k != name && strings.EqualFold(k, name) {
			return nil, false
		}
	}
	return v, true
}

// hasDuplicateKeys refuses a body that names the same key twice inside one
// object. It is finding 2 of the M1 review, and the reason is the invariant the
// whole package is built on: the guard must judge the bytes the server acts on.
//
// encoding/json keeps the last copy of a repeated key and drops the rest, so
// `{"requests":[],"requests":[{"deleteSuggestion":{}}]}` reads to the guard's
// own parse as one list and may read to the server as the other. The same trick
// hides a second `parents` from the create check and a second `action` from the
// comment check. Nothing gdoc builds repeats a key, so refusing every repeat
// costs nothing.
//
// The names are folded, because that is how encoding/json matches them: two
// spellings of one field are one field to the parse the guard is protecting.
//
// A body this cannot walk is refused here rather than left to the caller, and
// the decoder reads numbers as json.Number so that the walk can reach the end
// of any body a caller would accept. Both halves are one bug. json.Decoder
// decodes a number into a float64 by default, so a literal out of that range
// ends the walk with an error; the callers unmarshal into json.RawMessage and a
// []string, neither of which parses the number, so they accept the body and
// keep the last copy of the repeat. Swallowing the walk's error then carried
// exactly the bodies this exists to refuse, with the padding number as the key.
//
// With UseNumber the walk ends early only on a body that is not valid JSON,
// which every caller refuses on the line after this one, so failing closed here
// costs a message rather than a request.
func hasDuplicateKeys(body []byte) error {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	name, err := duplicateKey(dec)
	if err != nil {
		return refuse("the body could not be read as JSON, so the guard cannot tell whether it names a key twice: %v", err)
	}
	if name != "" {
		return refuse("the body names %q twice inside one object, and encoding/json keeps the last copy while the server may read the first", name)
	}
	return nil
}

// frame is one open container in the token walk below: an object tracks the
// names it has seen and whether the next token in it is a key, and an array
// tracks neither.
type frame struct {
	object bool
	seen   map[string]bool
	key    bool
}

// duplicateKey walks a JSON value as a token stream and returns the first key
// an object repeats, or "" when none does. The walk is iterative rather than
// recursive: the body reaching here is capped at maxPeek, and a megabyte of
// nested brackets would otherwise be a megabyte of stack frames.
func duplicateKey(dec *json.Decoder) (string, error) {
	var stack []*frame
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		top := (*frame)(nil)
		if n := len(stack); n > 0 {
			top = stack[n-1]
		}
		// Inside an object, the next token is a key or the closing brace.
		if top != nil && top.object && top.key {
			if d, ok := tok.(json.Delim); ok && d == '}' {
				stack = stack[:len(stack)-1]
				continue
			}
			key, ok := tok.(string)
			if !ok {
				return "", fmt.Errorf("an object key is not a string")
			}
			folded := strings.ToLower(key)
			if top.seen[folded] {
				return key, nil
			}
			top.seen[folded] = true
			top.key = false
			continue
		}
		// Anything else is a value, or the end of an array.
		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '{':
				markValueRead(top)
				stack = append(stack, &frame{object: true, seen: map[string]bool{}, key: true})
				continue
			case '[':
				markValueRead(top)
				stack = append(stack, &frame{})
				continue
			default: // '}' or ']'
				stack = stack[:len(stack)-1]
				continue
			}
		}
		markValueRead(top)
	}
}

// markValueRead tells an enclosing object that the value for its current key is
// consumed, so the next token there is a key again. An array has no keys, so it
// records nothing.
func markValueRead(f *frame) {
	if f != nil && f.object {
		f.key = true
	}
}

// judgeRequests reads the request kinds inside a batchUpdate body. SPEC.md's
// Never list: "Never accept, reject or delete anyone else's suggestion." Docs
// spells all three as request kinds inside `requests[]`
// (https://developers.google.com/workspace/docs/api/reference/rest/v1/documents/request),
// so neither the path nor the write level can see them, and the body is the
// only thing that can.
//
// The rule holds at both levels. A document gdoc created can still hold
// somebody else's suggestion, and the Never list names no level.
//
// Unknown kinds are carried, and that is the one rule in this guard that is not
// an allowlist. The reason is the shape of what it guards. A batchUpdate
// request kind is one entry in a documented schema, and the spec names the ones
// that are dangerous; the query surface is the opposite, a set Google keeps
// giving new spellings for the same capability, which is why params.go could
// only be an allowlist. An allowlist here would also be empty today, because M1
// has no batchUpdate call site, and an empty one refuses the SUGGEST write the
// guard exists to allow.
//
// So the family is refused rather than the three names: a kind whose name
// carries "suggestion" acts on one, and a fourth spelling of the same idea is
// refused before anybody has read about it. Every ordinary request kind a later
// milestone needs carries unchanged.
//
// The one door in that wall is AllowReject, Nail's decision of 2026-09-07. A
// rejectSuggestion is carried when it is spelled exactly, carries exactly one
// field, suggestionId spelled exactly, and that id is one the policy was
// granted for this run. Exactly, because Google's proto-JSON is case-sensitive
// while encoding/json is not, and the guard must never be broader than the
// server on the one field that permits a write: isSuggestMode holds the same
// rule on writeMode. A second field beside the id is refused because nobody
// here has read what it does. Everything else in the family, the two other
// verbs included, is refused whatever id it names.
//
// A body this cannot read is refused, at every level. That covers a body past
// the transport's peek, which arrives here truncated: a batchUpdate longer than
// maxPeek is refused rather than carried unread, and a milestone that needs a
// bigger one raises the cap on purpose.
//
// At LevelInPlace the unknown-kinds rule is inverted, and that inversion is
// M7b's whole point. The argument above rests on writeMode: a kind nobody has
// read about was still a suggestion, so carrying it cost a suggestion somebody
// could reject. LevelInPlace removes that bound, so inPlaceKinds stands in its
// place and a kind that is not on it is refused whatever it is called. The
// allowlist is scoped to that level alone: applied everywhere it would refuse
// the probe's direct insertText at LevelFull and every propose batch at
// LevelSuggest.
func (p *Policy) judgeRequests(body []byte, lvl Level) error {
	if err := hasDuplicateKeys(body); err != nil {
		return err
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		return refuse("the body of this batchUpdate cannot be read, so the requests in it cannot be judged: %v", err)
	}
	// The list is found by folding case, because encoding/json matches field
	// names that way and Docs may too, so `Requests` is the same list under a
	// spelling a check on the exact key reads as absent. Two spellings at once
	// are refused: which list the server takes is not decided here.
	var raw json.RawMessage
	found := false
	for name, v := range top {
		if !strings.EqualFold(name, "requests") {
			continue
		}
		if found {
			return refuse("this batchUpdate names its request list more than once, and which list the server reads is not decided here")
		}
		raw, found = v, true
	}
	if !found {
		return nil // a batchUpdate naming no requests changes nothing
	}
	var reqs []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &reqs); err != nil {
		return refuse("the request list of this batchUpdate cannot be read as a list of requests: %v", err)
	}
	for _, req := range reqs {
		// A Docs request names exactly one kind. A request naming none, or two,
		// is one the guard cannot say what it does, and that must not resolve
		// to sending it.
		if len(req) != 1 {
			return refuse("a request in this batchUpdate names %d kinds, and a request names one", len(req))
		}
		for kind, raw := range req {
			if kind == "rejectSuggestion" {
				// The one door through the allowlist below, and it is the door
				// AllowReject already opened: one id the note records as
				// gdoc's own. No command opens both grants, so this is written
				// down rather than relied on.
				if err := p.checkGrantedReject(raw); err != nil {
					return err
				}
				continue
			}
			if strings.Contains(strings.ToLower(kind), "suggestion") {
				return refuse("%q acts on a suggestion, and gdoc never accepts, rejects or deletes anyone else's", kind)
			}
			if lvl != LevelInPlace {
				continue
			}
			if !inPlaceKinds[kind] {
				return refuse("%q is not one of the four styling requests a restyle may send. At the in-place level the allowlist is the bound, so a request kind that is not on it is refused whatever it does: %s", kind, inPlaceKindList)
			}
			if err := checkInPlaceMask(kind, raw); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkGrantedReject judges the body of one rejectSuggestion request against
// the run's grant. The shape is exact: one key, suggestionId, a string, and an
// id AllowReject named. Read judgeRequests for why exact.
func (p *Policy) checkGrantedReject(raw json.RawMessage) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return refuse("the rejectSuggestion in this batchUpdate cannot be read: %v", err)
	}
	if len(fields) != 1 {
		return refuse("a rejectSuggestion carries exactly one field, suggestionId, and this one carries %d", len(fields))
	}
	idRaw, ok := fields["suggestionId"]
	if !ok {
		return refuse("a rejectSuggestion names the suggestion in suggestionId, spelled exactly, and this one does not")
	}
	var id string
	if err := json.Unmarshal(idRaw, &id); err != nil || id == "" {
		return refuse("the suggestionId of a rejectSuggestion is one non-empty string")
	}
	if !p.mayReject(id) {
		return refuse("rejectSuggestion names %q, which is not one of gdoc's own proposals in this run, and gdoc never accepts, rejects or deletes anyone else's", id)
	}
	return nil
}

// inPlaceKinds is the whole of what a restyle may send. Four request kinds,
// each measured landing on a real document on 2026-09-09 by
// internal/live/fidelity_test.go, and none of them able to change a single
// character of what the author wrote.
//
// The measurement is not the allowlist, and two kinds show why.
// createParagraphBullets lands, and the reference says the leading tabs that
// set a bullet's nesting level "are removed by this request", so it deletes
// text; the probe missed that because its content had no leading tabs.
// createNamedRange lands too and is left out because M7b writes no checklist
// and needs no range. Both are refused here.
//
// Adding a fifth kind is a decision for Nail, and it answers
// TestNothingAtLevelInPlaceCanChangeACharacter rather than this list.
var inPlaceKinds = map[string]bool{
	"updateDocumentStyle":  true,
	"updateParagraphStyle": true,
	"updateTextStyle":      true,
	"updateTableCellStyle": true,
}

// inPlaceKindList is the same four in a refusal, so a reader is told what was
// available rather than only what was refused. It is written out rather than
// built from the map, because a map has no order and a refusal that reads
// differently on each run is a refusal nobody can search for.
const inPlaceKindList = "updateDocumentStyle, updateParagraphStyle, updateTextStyle, updateTableCellStyle"

// refusedMaskFields are the two document-style fields a restyle may never name,
// even in a mask that sets them. Switching either off hides the first-page
// header, which carries the logo, and that header is the one thing this
// milestone reports as unreachable rather than writing. defaultHeaderId and
// firstPageHeaderId are read-only in the reference, so they need no rule.
var refusedMaskFields = map[string]bool{
	"useFirstPageHeaderFooter": true,
	"useEvenPageHeaderFooter":  true,
}

// checkInPlaceMask reads the fields mask of one styling request, and it is the
// rule the allowlist above cannot carry on its own.
//
// The allowlist bounds the kind. It does not bound the mask, and that is where
// the danger of this level lives: the Docs reference says "To reset a property
// to its default value, include its field name in the field mask but leave the
// field itself unset." So an updateTextStyle carrying `fields: "*"` over a
// range resets everything it does not set, bold, italic, links, colours and
// highlights, permanently, with every character intact. Saying the text is
// unreachable while leaving the mask unbounded would be reassuring about the
// wrong thing.
//
// Three refusals, and one of them is an absence. A star, by itself or inside a
// path. A mask that names nothing, because Google reads an empty mask as every
// field, which is checkFields' rule on a query read one layer out. And a mask
// naming either header toggle.
//
// The key is read exactly, the way isSuggestMode reads writeMode. Google's
// proto-JSON is case-sensitive while encoding/json is not, so a request whose
// mask sits under "Fields" is a request whose mask the server never reads: the
// guard must not judge a field the server is not acting on. Two spellings at
// once are refused because which one the server takes is not decided here;
// hasDuplicateKeys already refuses that body a layer out, and the rule is held
// here as well so this function reads honestly on its own.
func checkInPlaceMask(kind string, raw json.RawMessage) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return refuse("the %q in this batchUpdate cannot be read, so its fields mask cannot be judged: %v", kind, err)
	}
	maskRaw, ok := fields["fields"]
	if !ok {
		for name := range fields {
			if strings.EqualFold(name, "fields") {
				return refuse("%q names its fields mask %q, and the server reads only \"fields\" spelled exactly, so the guard would be judging a mask nothing acts on", kind, name)
			}
		}
		return refuse("%q carries no fields mask, and a styling request the guard cannot read the mask of may be resetting every property it does not set", kind)
	}
	for name := range fields {
		if name != "fields" && strings.EqualFold(name, "fields") {
			return refuse("%q names its fields mask twice, as %q and \"fields\", and which one the server reads is not decided here", kind, name)
		}
	}
	var mask string
	if err := json.Unmarshal(maskRaw, &mask); err != nil {
		return refuse("the fields mask of %q is one string, and this one is not: %v", kind, err)
	}
	named := 0
	for _, path := range strings.Split(mask, ",") {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		named++
		for _, seg := range strings.Split(path, ".") {
			if strings.Contains(seg, "*") {
				return refuse("the fields mask of %q names %q, and a mask carrying * resets every property this request does not set", kind, path)
			}
			if refusedMaskFields[seg] {
				return refuse("the fields mask of %q names %q, which switches off the first-page header carrying the logo. A restyle reports that header as something it cannot write, and never hides it", kind, seg)
			}
		}
	}
	if named == 0 {
		return refuse("the fields mask of %q is empty, and Google reads an empty mask as every field", kind)
	}
	return nil
}
