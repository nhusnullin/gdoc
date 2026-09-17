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
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"syscall"
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

// session is what the commands need of an authenticated reach: the reads, the
// writes, the upload a publish is, and the warnings the run produced.
// *gapi.Session satisfies it.
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
	PostMultipart(ctx context.Context, rawURL string, meta any, part []byte, partType string, into any) error
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

// now is this run's clock, read in three places: it stamps the suggestions
// snapshot, it dates a proposal in the note, and it dates a baseline cursor
// when the listing carried no instant of its own. It is behind a variable so a
// test can fix the instant all three are measured from.
//
// The cursor is the consequential one. A wrong snapshot stamp writes a wrong
// line into a note; a wrong cursor instant is compared at Drive as
// startModifiedTime, and a cursor that never moves backwards loses every
// comment written behind it. Read cursorFloor for the rest of that.
var now = time.Now

// docURLPatterns are the URL shapes Nail pastes. Three shapes, matched by two
// patterns: the editor URL is the usual one, with or without the account
// segment below, and the ?id= shape is what an older Drive link and a shared
// link carry.
//
// The optional u/<n>/ segment is the account the browser is signed in as, and
// it is in the address bar of anybody signed into more than one Google
// account. A pattern that does not allow for it comes back as "not a Google
// Docs URL" for a URL that is one.
var docURLPatterns = []*regexp.Regexp{
	regexp.MustCompile(`/document/(?:u/\d+/)?d/([A-Za-z0-9_-]+)`),
	regexp.MustCompile(`[?&]id=([A-Za-z0-9_-]+)`),
}

// driveID is what a Drive id looks like. The length floor is what stops a path
// fragment or a word from being opened as a document: the read would come back
// as a 404 naming neither mistake.
var driveID = regexp.MustCompile(`^[A-Za-z0-9_-]{20,}$`)

// folderURLPattern is the shape a Drive folder URL has. Opening the folder and
// copying the address bar is what a person has at hand, so it is what the tool
// takes.
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
// fewer than that. The count is checked in parseArgsN, so a caller reading a
// word it asked for always gets one, except under anyCount, where the command
// counts its own words and this returns the empty string for a word it did not
// get.
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

// parseArgsN is strict, as dispatch already is. An unknown flag, a repeated
// flag, a missing value and an extra positional argument each fail naming the
// offender. Nothing is accepted and ignored: a command that quietly drops what
// it did not understand tells the caller it did something it did not.
//
// want is the number of words the command takes before its flags: one for the
// commands that name a document, none for probe, which names a folder with a
// flag, and two for reply and withdraw, which name a document and then a thing
// inside it. It is exact in both directions. One word too many is refused
// because a command that ignores what it did not understand tells the caller it
// did something it did not, and one too few is refused because the missing word
// is what the command is about.
//
// anyCount is the one exception, and it is the table's word rather than this
// function's: neither guard fires, so the command is handed however many words
// it was given and counts them itself. help and completion carry it, because
// the refusal a missing word deserves there names a command or a shell, and the
// refusal below names a document.
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
//
// grants are applied to the policy before the session is built, so the first
// request in the program's history is judged against them. The one caller
// today is withdraw, handing in AllowReject for the suggestion the note records
// as gdoc's own. A grant is per run and names one thing; nothing here widens
// the level.
func open(target string, grants ...func(*guard.Policy)) (*reach, error) {
	id, err := documentID(target)
	if err != nil {
		return nil, err
	}
	p := guard.NewPolicy()
	p.AllowFile(id, guard.LevelSuggest)
	for _, grant := range grants {
		grant(p)
	}
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

func cmdRead(a *args) emit.Result {
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

// waitInterval is how long a wait sleeps between polls. It is two seconds, a
// constant the spec names, and a package variable only so a test can drive two
// polls in no wall time. It is deliberately not a flag: a caller that could set
// it could poll Drive as fast as it liked, and nothing on the wire would say
// the interval had changed. TestTheWaitIntervalIsTwoSeconds pins the literal.
var waitInterval = 2 * time.Second

// maxWait is the longest one call will look. The skill asks for nine minutes,
// because the tool that runs the command waits ten at most; the hour is the bar
// that stops a wait from becoming the watcher principle 1 refuses to have.
const maxWait = time.Hour

// waitedData is what one wait did, and every field is a fact. How many times it
// looked, how long it took, and whether the person stopped it. There is no
// field saying whether what came back was worth anything: that is the skill's,
// reading the same threads a one-shot listing prints.
type waitedData struct {
	Polls       int  `json:"polls"`
	Seconds     int  `json:"seconds"`
	Interrupted bool `json:"interrupted"`
}

// commentsData is what `gdoc comments` prints. The cursor is what the next poll
// hands back, and every listing prints one: a run whose threads carried no
// instant is dated from this run's clock instead, which is startCursor below.
// So the field carries no omitempty, because there is no listing it would fire
// on and a reader handling an absent cursor would be handling a state the
// binary cannot produce. Waited is absent unless --wait was given: reporting
// one poll on a call that never waited would tell a reader the binary polls
// when it does not.
type commentsData struct {
	DocumentID string            `json:"document_id"`
	Title      string            `json:"title"`
	Tabs       int               `json:"tabs"`
	MultiTab   bool              `json:"multi_tab"`
	Cursor     string            `json:"cursor"`
	Threads    []comments.Thread `json:"threads"`
	Waited     *waitedData       `json:"waited,omitempty"`
}

// cursorFloor is how far back a baseline cursor is dated when the listing gave
// no instant to date it from. What it covers is the offset between this
// machine's clock and Drive's, because the instant is read here and compared
// there: Drive applies it as startModifiedTime and withholds everything older.
// A clock five minutes fast would otherwise put the cursor five minutes into
// Drive's future and lose every comment written in that window, permanently,
// because the cursor never moves backwards.
//
// So the floor is generous rather than tight. It is not a round trip, and it is
// not paying for the reads either: startCursor is dated from the instant before
// the poll, not the instant after it. Being early costs nothing here, because
// this is only reached when the listing carried no readable instant at all,
// which is a document with no comments in it; being late loses a comment
// outright.
const cursorFloor = 5 * time.Minute

// startCursor is the cursor a listing hands the next call, and it is where a
// live session on a quiet document starts.
//
// NextCursor answers nil when nothing it saw carried an instant, which is every
// document that has no comments in it yet. That is the honest answer to "what
// is the newest activity here", and it is unusable as a starting point: --wait
// needs --since, and there would be no cursor to give it, so live mode could
// not start on the one document it is most often started on.
//
// So a listing with no instant of its own is dated from this run's clock,
// biased backwards. Backwards rather than forwards because a clock a little
// ahead of Drive's would otherwise tell the next poll to withhold a comment
// written in the meantime, and losing a comment is the wrong direction to be
// wrong in. It is the same argument Cursor.narrow makes at the boundary.
//
// at is read before the poll rather than after it, and the caller passes it in
// for that reason. Read afterwards, the reads themselves spend the floor: two
// slow reads put the cursor after the moment the listing went out, and a
// comment written while the baseline was being read falls into a window no poll
// ever asks for again.
func startCursor(c *comments.Cursor, at time.Time) *comments.Cursor {
	if c != nil {
		return c
	}
	return &comments.Cursor{At: at.Add(-cursorFloor)}
}

// parseWait reads the --wait value. Every refusal quotes what it was given: a
// caller reading "not a duration" without its own word back cannot tell which
// argument it wrote wrongly.
func parseWait(text string) (time.Duration, error) {
	d, err := time.ParseDuration(text)
	if err != nil {
		return 0, fmt.Errorf("--wait %q is not a length of time. Write it as 9m or 90s", text)
	}
	if d <= 0 {
		return 0, fmt.Errorf("--wait %q is not a length of time to look for", text)
	}
	if d > maxWait {
		return 0, fmt.Errorf("--wait %q is longer than the %s this command will look for", text, maxWait)
	}
	return d, nil
}

func cmdComments(ctx context.Context, a *args) emit.Result {
	var since *comments.Cursor
	if a.has("--since") {
		// Before the session, because a cursor nobody can read is a mistake in
		// the call rather than something a read could fix.
		parsed, err := comments.ParseCursor(a.flags["--since"])
		if err != nil {
			return emit.Result{OK: false, Error: err.Error()}
		}
		since = parsed
	}
	var deadline time.Duration
	if a.has("--wait") {
		// A wait with no cursor would hand back every thread in the document at
		// once, which is the one-shot read under another name and reads to a
		// session as news. The first read is the baseline and takes no --wait.
		if since == nil {
			return emit.Result{OK: false,
				Error: "--wait needs --since: the cursor is what makes a window, and a wait without one answers with the whole document"}
		}
		parsed, err := parseWait(a.flags["--wait"])
		if err != nil {
			return emit.Result{OK: false, Error: err.Error()}
		}
		deadline = parsed
	}
	r, err := open(a.target())
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}

	// One poll is the two reads this command has always made: the comment
	// listing, then the Docs read that carries the ranges the threads are
	// joined to and the title and the tab count the envelope reports.
	//
	// The listing goes first, and the order is the whole of it. A comment
	// written between the two reads is in whichever of them ran second. Listed
	// first, it is a comment the Docs read has not got to yet and the next poll
	// reports it with its range. Read first, it is a comment in the listing with
	// no anchor in a document read a moment before it existed, so the thread
	// comes back placed nowhere, the cursor moves past it, and the skill is told
	// a thread it could have proposed into cannot be. A wait ends on exactly the
	// poll that first sees a new comment, which is the poll the race is live on.
	//
	// The last document read is kept for the envelope's fields, which
	// comments.Waited does not carry and does not need to.
	var d *docs.Document
	poll := func(ctx context.Context) (*docs.Document, []comments.RawComment, error) {
		raws, err := comments.Fetch(ctx, r.session, r.id, since)
		if err != nil {
			return nil, nil, err
		}
		got, err := docs.Fetch(ctx, r.session, r.id)
		if err != nil {
			return nil, nil, err
		}
		d = got
		return got, raws, nil
	}

	if !a.has("--wait") {
		// Before the poll, because it is what a baseline cursor is dated from
		// and the reads must not spend the floor it is biased by.
		at := now()
		got, raws, err := poll(ctx)
		if err != nil {
			return emit.Result{OK: false, Error: err.Error(), Warnings: r.warnings()}
		}
		threads, unplaced := comments.Threads(raws, got)
		return commentsResult(ctx, r, d, threads, unplaced,
			startCursor(comments.NextCursor(since, threads), at), a.has("--witness"), nil)
	}

	// The wait hears Ctrl-C, and only the wait. A wait is the one run with an
	// answer already in hand when the person stops it: the envelope this call
	// already has, one JSON object, ok, no threads, and the cursor it was
	// handed. A default SIGINT there takes that answer with the process.
	//
	// It is installed here rather than in main because signal.Notify takes the
	// default kill away from the whole process for as long as it is on. Every
	// other command, `propose` included, keeps dying on the first Ctrl-C the way
	// it always did.
	//
	// The trap is on the wait's own context, not on the caller's, so what runs
	// after the wait cannot be quietly cancelled by it. defer is the safety net;
	// the stop that matters is the one on the line after the wait.
	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	w, err := comments.Wait(sigCtx, since, comments.WaitOptions{
		Interval: waitInterval,
		Deadline: deadline,
		Fetch:    poll,
	})
	// Off the moment the wait is over. What follows is a --witness export, and
	// a Ctrl-C during that must kill the run the way it kills every other
	// command. stop is what puts the default kill back. Left installed, the
	// handler holds that kill off until this function returns, while the export
	// below runs on the caller's context, which the signal never reaches: the
	// Ctrl-C then does nothing at all, and the run answers as though nobody had
	// pressed it.
	stop()
	waited := &waitedData{
		Polls: w.Polls,
		// Rounded, because the caller reads it to know how long the document was
		// quiet and not to measure the binary.
		Seconds:     int((w.Waited + time.Second/2) / time.Second),
		Interrupted: w.Interrupted,
	}
	if err != nil {
		// A failed poll is the one ending that is a failure, and the polls it
		// made still go out: the skill says so, waits, and calls again with the
		// cursor it already had.
		data := commentsBase(r, d, since)
		data.Waited = waited
		return emit.Result{OK: false, Error: err.Error(), Data: data, Warnings: r.warnings()}
	}
	// The deadline bounds the call, and a --witness export is part of the call.
	// Left unbounded it runs on the guard's own client timeout, minutes on top
	// of the wait, so a `--wait 9m --witness` answering at fourteen minutes is
	// killed by the harness that waits ten and prints nothing at all. An export
	// cut short by what is left is a warning naming it, on an envelope that
	// still carries the threads.
	rest, cancel := context.WithTimeout(ctx, deadline-w.Waited)
	defer cancel()
	return commentsResult(rest, r, d, w.Threads, w.Unplaced, w.Cursor, a.has("--witness"), waited)
}

// commentsBase is the envelope's document fields. The document is nil whenever
// no poll finished both of its reads: the id the run was given is still the
// truthful answer there, and title, tabs and multi_tab come back as their zero
// values, on an envelope that carries no threads either.
//
// They are not omitted, because the skill reads multi_tab on every listing and
// dropping it from the single-tab case would be a bigger change to the
// envelope than the empty title is worth.
//
// polls is not what tells those cases apart, and reading it that way is the
// mistake to avoid. There are four of them:
//
//   - a first poll cut short by the signal: polls 1, ok true, interrupted true;
//   - a first poll that failed at either read: polls 1, ok false;
//   - a wait whose context was already done before it looked: polls 0, ok true,
//     interrupted true;
//   - a first poll cut short by the deadline: polls 1, ok true, interrupted
//     false, which is the same shape as an ordinary quiet window and is told
//     from one by tabs: 0, because a window that polled to the end carries the
//     document's fields.
//
// So ok and interrupted separate the first three, and the nil document itself
// is what separates the fourth from a window that found nothing.
func commentsBase(r *reach, d *docs.Document, cursor *comments.Cursor) commentsData {
	data := commentsData{
		DocumentID: r.id,
		Cursor:     cursor.String(),
		Threads:    []comments.Thread{},
	}
	if d != nil {
		data.DocumentID = d.ID
		data.Title = d.Title
		data.Tabs = len(d.Tabs)
		data.MultiTab = d.MultiTab()
	}
	return data
}

// commentsResult is the successful listing, whether one call made it or a wait
// did. The two paths share it so a window cannot come back described one way
// and a one-shot listing another.
func commentsResult(ctx context.Context, r *reach, d *docs.Document, threads []comments.Thread,
	unplaced []string, cursor *comments.Cursor, wantWitness bool, waited *waitedData) emit.Result {
	var own []string
	for _, id := range unplaced {
		own = append(own, fmt.Sprintf(
			"comment %s: the Docs read gave it no usable range, so the thread comes back with its quoted text and no position", id))
	}
	// An empty window has nothing to witness, and the export would be one more
	// request for no question. An interrupted wait is that same case rather than
	// a second one: it comes back with no threads, so the export is skipped
	// without the interrupt having to be read here.
	if wantWitness && len(threads) > 0 {
		threads, own = witness(ctx, r, threads, own)
	}
	data := commentsBase(r, d, cursor)
	data.Threads = threads
	data.Waited = waited
	return emit.Result{OK: true, Warnings: r.warnings(own...), Data: data}
}

// witness reads the docx export and sets Witness on every thread. An export
// that could not be read is a warning and every thread unmatched, not a failed
// listing: the threads are the answer, and the witness is a second read on top
// of them.
func witness(ctx context.Context, r *reach, threads []comments.Thread, own []string) ([]comments.Thread, []string) {
	f, err := exportFile(ctx, r)
	if err != nil {
		return docx.Match(threads, nil), append(own, fmt.Sprintf(
			"the docx export could not be read, so every thread is reported unmatched: %v", err))
	}
	return docx.Match(threads, f), own
}

// exportFile is the export read and parsed, and it is the one export path in
// this binary. Both callers that want a witness go through it: `comments
// --witness` above, and the survey. Two export paths would be two chances for
// one of them to ask Drive for a different document, or to read the answer to a
// different ceiling.
func exportFile(ctx context.Context, r *reach) (*docx.File, error) {
	b, err := docx.Export(ctx, r.session, r.id)
	if err != nil {
		return nil, err
	}
	return docx.Parse(b)
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

func cmdSuggestions(a *args) emit.Result {
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
