// Package comments is the threads on a document, as facts.
//
// Two sources, because neither one carries the whole answer. Drive's
// comments.list is documented and stable and carries the replies, the authors,
// resolved and modifiedTime, but its anchor is an opaque string. The Docs read
// carries the character range each thread sits on, keyed by the same comment
// id, and carries nothing about the replies. So this package joins them, and
// reports the id of a thread the Docs read did not place rather than failing
// the whole listing over one anchor.
//
// Nothing here decides whether a thread is answered. SPEC.md was corrected on
// 2026-09-06 for exactly this: "a 🤖 reply newer than the comment" is a
// heuristic, and a wrong one when the reply was an acknowledgment or the
// follow-up changed the question. So a Thread carries every reply, its author,
// its time and whether it opens with 🤖, and the skill reading the JSON decides
// what still needs an answer. There is no Handled field, and there must not be.
package comments

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"gdoc/internal/docs"
)

// pageSize is the largest page Drive's comments.list serves. Fewer, larger
// pages is fewer round trips through the guard for the same threads.
const pageSize = "100"

// maxPages bounds the listing. 100 pages of 100 is ten thousand comments, which
// is more than a document under review has; past it the answer is a report, not
// a document, and a listing still handing back tokens is a Drive that is not
// advancing rather than a document that is very large.
const maxPages = 100

// fieldMask is exactly what a thread is made of, and nothing else. Asking for
// every field would carry the permission surface the guard refuses to reach by
// its own path, so the mask is written out rather than left off.
const fieldMask = "nextPageToken,comments(id,author(displayName,me),createdTime,modifiedTime," +
	"content,resolved,quotedFileContent(value),replies(id,author(displayName,me),createdTime,content))"

// The markers a comment can carry. The match is exact: `AI:` is not one of
// them, and neither is `ai:x`. A marker is the trigger for gdoc to act, so a
// loose match is gdoc acting on a sentence nobody addressed to it.
const (
	MarkerNone = "none"
	robot      = "🤖"
)

var markers = map[string]bool{"ai:": true, "ai?": true, "ai!": true}

// RawComment is one entry of Drive's comments.list answer, in Drive's own
// field names. It is what Fetch returns, so a caller that wants the wire's
// words has them, and Threads is a pure function of it.
type RawComment struct {
	ID                string     `json:"id"`
	Author            Author     `json:"author"`
	CreatedTime       string     `json:"createdTime"`
	ModifiedTime      string     `json:"modifiedTime"`
	Content           string     `json:"content"`
	Resolved          bool       `json:"resolved"`
	QuotedFileContent *Quoted    `json:"quotedFileContent"`
	Replies           []RawReply `json:"replies"`
}

// RawReply is one reply of a thread, as Drive answers it.
type RawReply struct {
	ID          string `json:"id"`
	Author      Author `json:"author"`
	CreatedTime string `json:"createdTime"`
	Content     string `json:"content"`
}

// Author is who wrote a comment. `me` is kept because Drive sends it, and not
// used for anything: under one credential it means gdoc and under another it
// means Nail, so branching on it would skip every comment Nail wrote. The 🤖
// prefix is what says gdoc wrote a reply.
type Author struct {
	DisplayName string `json:"displayName"`
	Me          bool   `json:"me"`
}

// Quoted is the document text a comment was attached to, as Drive saw it when
// the comment was made. It is a snapshot, not a live read: the words may have
// been edited since, which is what the Docs range and the docx witness are for.
type Quoted struct {
	Value string `json:"value"`
}

// Thread is one comment and its replies, with the range the Docs read placed it
// at. Witness is empty here and filled in by internal/docx when the caller
// asked for it: whether the export still carries the comment attached to text
// is a second read, and most runs do not make it.
type Thread struct {
	ID       string      `json:"id"`
	Author   string      `json:"author"`
	Created  string      `json:"created"`
	Modified string      `json:"modified"`
	Content  string      `json:"content"`
	Marker   string      `json:"marker"`
	Resolved bool        `json:"resolved"`
	Quoted   string      `json:"quoted"`
	Range    *docs.Range `json:"range"`
	Replies  []Reply     `json:"replies"`
	Witness  string      `json:"witness,omitempty"`
}

// Reply is one reply, as the skill reads it. ByGdoc is a fact about the text,
// not about the credential: see Author.
type Reply struct {
	ID      string `json:"id"`
	Author  string `json:"author"`
	Created string `json:"created"`
	Content string `json:"content"`
	ByGdoc  bool   `json:"by_gdoc"`
}

// Reader is the one thing this package needs of a session: a JSON GET. A
// *gapi.Session satisfies it, and so does a fake in a test. The interface is
// named here rather than the struct so that net/http stays out of this room and
// out of its tests, which is what the boundary test asks of every package but
// the four that build requests.
type Reader interface {
	GetJSON(ctx context.Context, rawURL string, into any) error
}

// ListURL is one page of comments.list.
//
// supportsAllDrives is deliberately absent, and this is the one Drive read
// where that is true. comments.list does not define it: the parameter belongs
// to the files collection, and the guard's driveCommentListParams, written from
// the Drive v3 reference, names five parameters and not that one. A comment
// read on a shared-drive file works without it because the file is addressed by
// id and the comment collection hangs off the file.
//
// startModifiedTime is the --since cursor: activity after this instant. It is
// added only when there is a cursor, because Drive reads an empty value as a
// timestamp it cannot parse rather than as no filter at all.
func ListURL(id, pageToken, since string) string {
	q := url.Values{
		"pageSize":       {pageSize},
		"includeDeleted": {"false"},
		"fields":         {fieldMask},
	}
	if pageToken != "" {
		q.Set("pageToken", pageToken)
	}
	if since != "" {
		q.Set("startModifiedTime", since)
	}
	// Encoded, not concatenated. A page token is Drive's opaque string, and one
	// carrying an & would end the parameter and start another: the request on
	// the wire would then not be the request the guard judged.
	return "https://www.googleapis.com/drive/v3/files/" + id + "/comments?" + q.Encode()
}

// listPage is one answer from comments.list.
type listPage struct {
	NextPageToken string       `json:"nextPageToken"`
	Comments      []RawComment `json:"comments"`
}

// Fetch reads every page of the comment listing through the session's guarded
// client. A document the command was not given is refused inside the process,
// before anything reaches a wire, and that refusal comes back as this
// function's error unwrapped: it is the sentence the caller has to read.
func Fetch(ctx context.Context, s Reader, id string, since *Cursor) ([]RawComment, error) {
	var out []RawComment
	token := ""
	for page := 0; page < maxPages; page++ {
		var answer listPage
		if err := s.GetJSON(ctx, ListURL(id, token, since.since()), &answer); err != nil {
			return nil, err
		}
		out = append(out, answer.Comments...)
		if answer.NextPageToken == "" {
			return out, nil
		}
		if answer.NextPageToken == token {
			return nil, fmt.Errorf("the comment listing repeated its page token after %d comments, so it is not advancing", len(out))
		}
		token = answer.NextPageToken
	}
	// The loop is bounded because the exit is Drive's to give. A listing that
	// keeps handing back a token is a hang with no output and no exit code,
	// which is the one thing the output contract promises cannot happen.
	return nil, fmt.Errorf("the comment listing did not end after %d pages of %s comments each", maxPages, pageSize)
}

// Threads joins the Drive listing to the Docs ranges, in the order Drive
// answered. The second return is the ids no range was found for: the thread
// still comes back, with its quoted text, and the command puts the ids in
// warnings. A missing anchor is a fact about one thread, never a reason to fail
// the read of all of them.
//
// A nil document is no ranges rather than a crash: a run that reads threads
// without a Docs read behind it is a run where every thread is unplaced.
func Threads(raw []RawComment, d *docs.Document) ([]Thread, []string) {
	var ranges map[string]docs.Range
	if d != nil {
		ranges = d.CommentRanges
	}
	threads := make([]Thread, 0, len(raw))
	var unplaced []string
	for _, c := range raw {
		t := Thread{
			ID:       c.ID,
			Author:   c.Author.DisplayName,
			Created:  c.CreatedTime,
			Modified: c.ModifiedTime,
			Content:  c.Content,
			Marker:   markerOf(c.Content),
			Resolved: c.Resolved,
			Replies:  replies(c.Replies),
		}
		if c.QuotedFileContent != nil {
			t.Quoted = c.QuotedFileContent.Value
		}
		if r, ok := ranges[c.ID]; ok {
			// A copy, so the caller cannot reach into the document's map.
			t.Range = &r
		} else {
			unplaced = append(unplaced, c.ID)
		}
		threads = append(threads, t)
	}
	return threads, unplaced
}

// replies is the reply list, never nil: a null in the JSON is a shape the
// caller has to special-case, and an empty list says the same thing.
func replies(raw []RawReply) []Reply {
	out := make([]Reply, 0, len(raw))
	for _, r := range raw {
		out = append(out, Reply{
			ID:      r.ID,
			Author:  r.Author.DisplayName,
			Created: r.CreatedTime,
			Content: r.Content,
			ByGdoc:  byGdoc(r.Content),
		})
	}
	return out
}

// markerOf is the first whitespace-separated token of a comment, when that
// token is exactly one of the three markers. Everything else is MarkerNone,
// including `AI:` and `ai:no-space`: the marker is what makes gdoc act, and a
// loose match is gdoc acting on a sentence nobody addressed to it.
func markerOf(content string) string {
	// Fields, so the sentence is cut the way the doc comment says it is. Cutting
	// on one separator at a time left `ai:<tab>answer this` as a single token,
	// which read as no marker at all.
	fields := strings.Fields(content)
	if len(fields) > 0 && markers[fields[0]] {
		return fields[0]
	}
	return MarkerNone
}

// byGdoc is true when a reply opens with the robot. SPEC.md: every reply gdoc
// writes opens with 🤖 and nothing else, so this is the receipt. Leading
// whitespace is stepped over, because a reply that starts on its second line is
// still a reply that opens with the robot.
func byGdoc(content string) bool {
	return strings.HasPrefix(strings.TrimLeft(content, " \t\r\n"), robot)
}
