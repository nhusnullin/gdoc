// The three read commands, and the argument parsing they share.
//
// Every one of them does the same four things in the same order: turn the
// argument into a document id, open a policy holding exactly that id, open a
// session on the guard's client, and hand what came back to a pure reader
// package. The readers decide nothing, and neither does anything here: the
// commands print facts and the skill reading the JSON judges.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"gdoc/internal/atomicfile"
	"gdoc/internal/comments"
	"gdoc/internal/docs"
	"gdoc/internal/docx"
	"gdoc/internal/emit"
	"gdoc/internal/frontmatter"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/suggestions"
	"gdoc/internal/view"
)

// session is what a read command needs of an authenticated reach: two GETs and
// the warnings the run produced. *gapi.Session satisfies it.
//
// The interface is named here rather than the struct so that a command test can
// stand in for the wire without naming net/http. A stub built on the concrete
// type would need an http.RoundTripper, which would put cmd/gdoc in the
// boundary test's import allowlist for the sake of a test helper.
type session interface {
	GetJSON(ctx context.Context, rawURL string, into any) error
	GetBytes(ctx context.Context, rawURL string, limit int64) ([]byte, error)
	PostJSON(ctx context.Context, rawURL string, body any, into any) error
	PatchJSON(ctx context.Context, rawURL string, body any, into any) error
	Warnings() []string
}

// openSession is the session factory behind a variable, as login is. The base
// transport is nil, which means the real wire, and the client it goes out on is
// still the guard's.
var openSession = func(p *guard.Policy) (session, error) {
	s, err := gapi.Open(p, nil)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// now is the clock the snapshot is stamped with, behind a variable so a test
// states the line it expects rather than parsing one back.
var now = time.Now

// docURLPatterns are the URL shapes Nail pastes, ported from v1's
// gdoc/docid.py. Three shapes, matched by two patterns: the editor URL is the
// usual one, with or without the account segment below, and the ?id= shape is
// what an older Drive link and a shared link carry.
//
// The optional u/<n>/ segment is the account the browser is signed in as, and
// it is in the address bar of anybody signed into more than one Google account.
// v1's two patterns miss it, so the paste comes back as "not a Google Docs
// URL" for a URL that is one.
var docURLPatterns = []*regexp.Regexp{
	regexp.MustCompile(`/document/(?:u/\d+/)?d/([A-Za-z0-9_-]+)`),
	regexp.MustCompile(`[?&]id=([A-Za-z0-9_-]+)`),
}

// driveID is what a Drive id looks like. The length floor is what stops a path
// fragment or a word from being opened as a document: the read would come back
// as a 404 naming neither mistake.
var driveID = regexp.MustCompile(`^[A-Za-z0-9_-]{20,}$`)

// folderURLPattern is the shape a Drive folder URL has, ported from v1's
// gdoc/docid.py. Opening the folder and copying the address bar is what a
// person has at hand, so it is what the tool takes.
var folderURLPattern = regexp.MustCompile(`/folders/([A-Za-z0-9_-]+)`)

// documentURLPattern is a document URL, recognised here only so that handing
// one to --folder is refused by name. A document URL carries a valid-looking
// id, so passing it through would create the probe document nowhere and come
// back as a 404 naming neither mistake.
var documentURLPattern = regexp.MustCompile(`/document/(?:u/\d+/)?d/[A-Za-z0-9_-]+`)

// folderID turns the argument into a Drive folder id: a folder URL, or a bare
// id. It is documentID's twin, and the two are deliberately not one function:
// a folder and a document are different things to be pointed at, and the whole
// value of the check is that each refuses the other's URL by name.
func folderID(arg string) (string, error) {
	text := strings.TrimSpace(arg)
	if m := folderURLPattern.FindStringSubmatch(text); m != nil {
		if !driveID.MatchString(m[1]) {
			return "", fmt.Errorf("%q names %q, which is too short to be a Drive folder id", text, m[1])
		}
		return m[1], nil
	}
	if documentURLPattern.MatchString(text) {
		return "", fmt.Errorf("%q is a document, not a folder", text)
	}
	if driveID.MatchString(text) {
		return text, nil
	}
	if text == "" {
		return "", errors.New("this command needs a folder: a Drive folder URL or a folder id")
	}
	return "", fmt.Errorf("%q is not a Drive folder URL or a folder id", text)
}

// documentID turns the argument into a document id. A bare id is accepted
// because it costs one line, and anything else is refused quoting the input:
// every caller needs an id, and a silent empty one surfaces later as a 404.
func documentID(arg string) (string, error) {
	text := strings.TrimSpace(arg)
	for _, re := range docURLPatterns {
		m := re.FindStringSubmatch(text)
		if m == nil {
			continue
		}
		if !driveID.MatchString(m[1]) {
			return "", fmt.Errorf("%q names %q, which is too short to be a Drive document id", text, m[1])
		}
		return m[1], nil
	}
	if driveID.MatchString(text) {
		return text, nil
	}
	if text == "" {
		return "", errors.New("this command needs a document: a Google Docs URL or a document id")
	}
	return "", fmt.Errorf("%q is not a Google Docs URL or a document id", text)
}

// flagSet is the flags one command takes: the name against whether it carries a
// value. It is an allowlist, so a flag the command does not know is refused
// rather than ignored.
type flagSet map[string]bool

// names lists the flags for a refusal, in a stable order.
func (fs flagSet) names() string {
	if len(fs) == 0 {
		return "no flags"
	}
	out := make([]string, 0, len(fs))
	for name := range fs {
		out = append(out, name)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// args is one command's arguments after parsing: the words it was pointed at,
// in the order they were written, and the flags it was given.
type args struct {
	positional []string
	flags      map[string]string
}

// target is the first positional argument, which every command that names a
// document takes first.
func (a *args) target() string { return a.at(0) }

// at is one positional argument, or the empty string when the command took
// fewer than that. The count is checked in parseArgs, so a caller reading a
// word it asked for always gets one.
func (a *args) at(i int) string {
	if i >= len(a.positional) {
		return ""
	}
	return a.positional[i]
}

// has reports whether the flag was given at all, whatever its value.
func (a *args) has(name string) bool {
	_, given := a.flags[name]
	return given
}

// parseArgs is strict, as dispatch already is. An unknown flag, a repeated
// flag, a missing value and an extra positional argument each fail naming the
// offender. Nothing is accepted and ignored: a command that quietly drops what
// it did not understand tells the caller it did something it did not.
func parseArgs(raw []string, spec flagSet) (*args, error) {
	return parseArgsN(raw, spec, 1)
}

// parseArgsN is parseArgs for a command that takes a different number of words
// before its flags: none for probe, which names a folder with a flag, and two
// for reply and withdraw, which name a document and then a thing inside it.
//
// want is exact in both directions. One word too many is refused because a
// command that ignores what it did not understand tells the caller it did
// something it did not, and one too few is refused because the missing word is
// what the command is about.
func parseArgsN(raw []string, spec flagSet, want int) (*args, error) {
	a := &args{flags: map[string]string{}}
	for i := 0; i < len(raw); i++ {
		arg := raw[i]
		if !strings.HasPrefix(arg, "-") {
			if len(a.positional) == want {
				return nil, fmt.Errorf("%q is an extra argument: this command takes %s", arg, words(want))
			}
			a.positional = append(a.positional, arg)
			continue
		}
		name, value, joined := strings.Cut(arg, "=")
		takesValue, known := spec[name]
		if !known {
			return nil, fmt.Errorf("%q is not a flag this command takes. It takes: %s", name, spec.names())
		}
		if _, twice := a.flags[name]; twice {
			return nil, fmt.Errorf("%s is given twice, and which one counts is not decided here", name)
		}
		switch {
		case !takesValue && joined:
			return nil, fmt.Errorf("%s takes no value, and %q gives it one", name, arg)
		case !takesValue:
			a.flags[name] = ""
		case joined && value == "":
			return nil, fmt.Errorf("%s was given an empty value", name)
		case joined:
			a.flags[name] = value
		case i+1 >= len(raw):
			return nil, fmt.Errorf("%s needs a value, and none follows it", name)
		case knows(spec, raw[i+1]):
			// The next argument is another flag this command takes, so it is not
			// this one's value. Swallowing it read `--since --witness` as a
			// cursor and failed naming the cursor, which is the wrong problem.
			return nil, fmt.Errorf("%s needs a value, and %q is another flag", name, raw[i+1])
		case raw[i+1] == "":
			return nil, fmt.Errorf("%s was given an empty value", name)
		default:
			i++
			a.flags[name] = raw[i]
		}
	}
	if len(a.positional) < want {
		if want == 1 {
			return nil, errors.New("this command needs a document: a Google Docs URL or a document id")
		}
		return nil, fmt.Errorf("this command takes %s, and %d were given", words(want), len(a.positional))
	}
	return a, nil
}

// words says how many arguments a command takes, in a sentence.
func words(n int) string {
	switch n {
	case 0:
		return "no arguments"
	case 1:
		return "one document"
	default:
		return fmt.Sprintf("%d arguments", n)
	}
}

// knows reports whether the argument names a flag of this command. It is the
// map lookup rather than the value, because a flag that takes no value is a
// false in the same map.
func knows(spec flagSet, arg string) bool {
	_, ok := spec[arg]
	return ok
}

// reach is one command's opened reach: the id it was given, and the session on
// the guard's client that holds exactly that id.
type reach struct {
	id      string
	session session
}

// open turns the argument into a reach. The policy is opened at LevelSuggest,
// which is what a handed-in document gets: read, comment and suggest, and never
// a direct edit.
func open(target string) (*reach, error) {
	id, err := documentID(target)
	if err != nil {
		return nil, err
	}
	p := guard.NewPolicy()
	p.AllowFile(id, guard.LevelSuggest)
	s, err := openSession(p)
	if err != nil {
		return nil, err
	}
	return &reach{id: id, session: s}, nil
}

// warnings is the session's warnings, which are the policy's followed by its
// own, plus whatever the command has to add. A new slice, so nothing the
// session holds is written to.
func (r *reach) warnings(own ...string) []string {
	if r == nil {
		return own
	}
	out := append([]string{}, r.session.Warnings()...)
	return append(out, own...)
}

// readData is what `gdoc read` prints. The text is what the AI reads; the
// structure is what a write milestone places a proposal into, and it is absent
// unless it was asked for.
type readData struct {
	DocumentID string `json:"document_id"`
	Title      string `json:"title"`
	RevisionID string `json:"revision_id"`
	Tabs       int    `json:"tabs"`
	MultiTab   bool   `json:"multi_tab"`
	Text       string `json:"text"`
	Structure  any    `json:"structure,omitempty"`
}

func cmdRead(raw []string) emit.Result {
	a, err := parseArgs(raw, flagSet{"--structure": false})
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	r, err := open(a.target())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	d, err := docs.Fetch(context.Background(), r.session, r.id)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings()}
	}
	text, notes := view.Text(d)
	notes = append(unplacedWarnings(d), notes...)
	data := readData{
		DocumentID: d.ID,
		Title:      d.Title,
		RevisionID: d.RevisionID,
		Tabs:       len(d.Tabs),
		MultiTab:   d.MultiTab(),
		Text:       text,
	}
	if a.has("--structure") {
		data.Structure = view.Structure(d)
	}
	return emit.Result{OK: true, Data: data, Warnings: r.warnings(notes...)}
}

// unplacedWarnings names the comments the Docs read returned with no range this
// binary could read. `read` marks a range in the text, so a comment with no
// range is one the text cannot show, and silence there reads as a document with
// no such comment in it.
func unplacedWarnings(d *docs.Document) []string {
	var out []string
	for _, id := range d.Unplaced {
		out = append(out, fmt.Sprintf(
			"comment %s: the Docs read placed no range for it, so the text carries no marker for it", id))
	}
	return out
}

// commentsData is what `gdoc comments` prints. The cursor is what the next poll
// hands back; it is absent when this run has no instant to report.
type commentsData struct {
	DocumentID string            `json:"document_id"`
	Title      string            `json:"title"`
	Tabs       int               `json:"tabs"`
	MultiTab   bool              `json:"multi_tab"`
	Cursor     string            `json:"cursor,omitempty"`
	Threads    []comments.Thread `json:"threads"`
}

func cmdComments(raw []string) emit.Result {
	a, err := parseArgs(raw, flagSet{"--since": true, "--witness": false})
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	var since *comments.Cursor
	if a.has("--since") {
		// Before the session, because a cursor nobody can read is a mistake in
		// the call rather than something a read could fix.
		since, err = comments.ParseCursor(a.flags["--since"])
		if err != nil {
			return emit.Result{OK: false, Error: err.Error()}
		}
	}
	r, err := open(a.target())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	ctx := context.Background()
	// The Docs read first: it carries the ranges the threads are joined to, and
	// the title and the tab count the envelope reports.
	d, err := docs.Fetch(ctx, r.session, r.id)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings()}
	}
	raws, err := comments.Fetch(ctx, r.session, r.id, since)
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings()}
	}
	threads, unplaced := comments.Threads(raws, d)

	var own []string
	for _, id := range unplaced {
		own = append(own, fmt.Sprintf(
			"comment %s: the Docs read gave it no usable range, so the thread comes back with its quoted text and no position", id))
	}
	if a.has("--witness") {
		var witnessed []comments.Thread
		witnessed, own = witness(ctx, r, threads, own)
		threads = witnessed
	}
	return emit.Result{OK: true, Warnings: r.warnings(own...), Data: commentsData{
		DocumentID: d.ID,
		Title:      d.Title,
		Tabs:       len(d.Tabs),
		MultiTab:   d.MultiTab(),
		Cursor:     comments.NextCursor(since, threads).String(),
		Threads:    threads,
	}}
}

// witness reads the docx export and sets Witness on every thread. An export
// that could not be read is a warning and every thread unmatched, not a failed
// listing: the threads are the answer, and the witness is a second read on top
// of them.
func witness(ctx context.Context, r *reach, threads []comments.Thread, own []string) ([]comments.Thread, []string) {
	var f *docx.File
	b, err := docx.Export(ctx, r.session, r.id)
	if err == nil {
		f, err = docx.Parse(b)
	}
	if err != nil {
		return docx.Match(threads, nil), append(own, fmt.Sprintf(
			"the docx export could not be read, so every thread is reported unmatched: %v", err))
	}
	return docx.Match(threads, f), own
}

// suggestionsData is what `gdoc suggestions` prints. Gone is a pointer so that
// a run with --md and nothing gone prints an empty list, which says "nothing
// stopped being pending", and a run without --md omits the field, which says
// there was nothing to compare against.
type suggestionsData struct {
	DocumentID   string                `json:"document_id"`
	Tabs         int                   `json:"tabs"`
	MultiTab     bool                  `json:"multi_tab"`
	Pending      []suggestions.Pending `json:"pending"`
	Gone         *[]suggestions.Gone   `json:"gone_since_last_look,omitempty"`
	FilesChanged []string              `json:"files_changed,omitempty"`
}

func cmdSuggestions(raw []string) emit.Result {
	a, err := parseArgs(raw, flagSet{"--md": true})
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	r, err := open(a.target())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	d, err := docs.Fetch(context.Background(), r.session, r.id)
	if err != nil {
		// The read failed, so the snapshot is not written. A snapshot taken now
		// would report whatever this run could not see as gone on the next one.
		return emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings()}
	}
	pending := suggestions.List(d)
	if pending == nil {
		pending = []suggestions.Pending{}
	}
	data := suggestionsData{
		DocumentID: d.ID,
		Tabs:       len(d.Tabs),
		MultiTab:   d.MultiTab(),
		Pending:    pending,
	}
	if !a.has("--md") {
		return emit.Result{OK: true, Data: data, Warnings: r.warnings()}
	}
	path := a.flags["--md"]
	gone, err := recordSnapshot(path, r.id, suggestions.All(d), suggestions.IDs(d))
	if err != nil {
		return emit.Result{OK: false, Error: err.Error(), Data: data, Warnings: r.warnings()}
	}
	data.Gone = &gone
	data.FilesChanged = []string{path}
	return emit.Result{OK: true, Data: data, Warnings: r.warnings()}
}

// recordSnapshot compares the file's last snapshot against what is pending now
// and writes the new one. It is the only write the read commands make, and it
// happens only after a read that fully succeeded.
//
// A file paired with another document is refused: writing this document's
// observation into it would be the wrong file, and the next run would read the
// snapshot as this document's history.
// all is every pending suggestion, the whitespace-only ones the printed listing
// drops included, and nowIDs is their ids. What left is a question about ids: a
// suggestion the author edited down to a space has not left, and a snapshot
// that forgot it cannot say so when it does.
func recordSnapshot(path, id string, all []suggestions.Pending, nowIDs []string) ([]suggestions.Gone, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("the markdown file could not be read: %w", err)
	}
	block, err := frontmatter.Read(src)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("%s carries no gdoc: front matter, so it is not paired with a document", path)
	}
	if block.DocumentID != id {
		return nil, fmt.Errorf("%s is paired with document %s, and this read was of %s", path, block.DocumentID, id)
	}
	gone := suggestions.GoneSince(block.SuggestionsSeen, nowIDs)
	if gone == nil {
		gone = []suggestions.Gone{}
	}
	// A copy, so the block that was read is not written to.
	updated := *block
	updated.SuggestionsSeen = suggestions.Snapshot(all, now())
	out, err := frontmatter.Write(src, &updated)
	if err != nil {
		return nil, err
	}
	if err := writeFile(path, out); err != nil {
		return nil, err
	}
	return gone, nil
}

// writeFile replaces the note, keeping the mode it had. A failed write must not
// leave a note truncated: the markdown is the source, and gdoc is not its only
// reader.
func writeFile(path string, b []byte) error {
	return atomicfile.Replace(path, b, atomicfile.ModeOf(path, 0o644))
}
