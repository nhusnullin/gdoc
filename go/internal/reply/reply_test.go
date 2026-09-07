package reply

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"

	"gdoc/internal/comments"
	"gdoc/internal/guard"
)

const (
	testDocID     = "1RePlY000000000000000000000000000000000"
	testCommentID = "AAAA"
	testReplyID   = "AAAB"
	testCreated   = "2026-09-07T10:00:00Z"
	goodBody      = "🤖 The 2026 register."
)

// call is one request the fake was asked to make, with the bytes the session
// would have marshalled, so a test reads what the wire would carry.
type call struct {
	method string
	url    string
	body   json.RawMessage
}

// fakeSession answers by call shape rather than by position, so a test that
// changes how far the run gets does not have to re-count the answers behind it.
// Nothing here names net/http: this is not a room the boundary test lists.
type fakeSession struct {
	calls    []call
	created  []byte        // the replies.create answer
	listing  []byte        // the comments.list answer
	failAt   map[int]error // fail the nth call, counted from zero
	postSeen int
}

func (f *fakeSession) do(method, rawURL string, body any, into any) error {
	raw := json.RawMessage(nil)
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		raw = b
	}
	f.calls = append(f.calls, call{method: method, url: rawURL, body: raw})
	if method == "POST" {
		f.postSeen++
	}
	if err := f.failAt[len(f.calls)-1]; err != nil {
		return err
	}
	answer := f.listing
	if method == "POST" {
		answer = f.created
	}
	if into == nil {
		return nil
	}
	if len(answer) == 0 {
		answer = []byte(`{}`)
	}
	return json.Unmarshal(answer, into)
}

func (f *fakeSession) GetJSON(_ context.Context, rawURL string, into any) error {
	return f.do("GET", rawURL, nil, into)
}

func (f *fakeSession) PostJSON(_ context.Context, rawURL string, body any, into any) error {
	return f.do("POST", rawURL, body, into)
}

// listingWith is Drive's comments.list answer carrying one thread whose replies
// are the ones named.
func listingWith(replies ...comments.RawReply) []byte {
	answer := struct {
		Comments []comments.RawComment `json:"comments"`
	}{
		Comments: []comments.RawComment{{
			ID:      testCommentID,
			Content: "ai: is this still annual?",
			Replies: replies,
		}},
	}
	b, err := json.Marshal(answer)
	if err != nil {
		panic(err)
	}
	return b
}

// script is a Drive that accepts the reply and shows it in the thread.
func script() *fakeSession {
	return &fakeSession{
		created: []byte(`{"id":"` + testReplyID + `","createdTime":"` + testCreated + `","content":` + quote(goodBody) + `}`),
		listing: listingWith(comments.RawReply{
			ID:          testReplyID,
			CreatedTime: testCreated,
			Content:     goodBody,
		}),
		failAt: map[int]error{},
	}
}

func quote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestCheckAcceptsAPlainRobotReply(t *testing.T) {
	for _, body := range []string{
		goodBody,
		"🤖 Done. The register is now reviewed every six months.",
		"🤖 Two lines.\nThe second one says more.",
		"🤖 A hyphen bullet reads fine as plain text:\n- one\n- two",
	} {
		if err := Check(body); err != nil {
			t.Errorf("Check(%q) refused a plain reply: %v", body, err)
		}
	}
}

func TestCheckRefusesEachBadShapeByName(t *testing.T) {
	cases := []struct {
		name string
		body string
		says string
	}{
		{"empty", "", "empty"},
		{"whitespace only", "   \n\t ", "empty"},
		{"no robot at all", "The 2026 register.", "🤖 "},
		{"a space before the robot", " 🤖 The 2026 register.", "🤖 "},
		{"the robot with no space", "🤖The 2026 register.", "🤖 "},
		{"the robot alone", "🤖 ", "empty"},
		{"bold", "🤖 The **2026** register.", "**"},
		{"a backtick", "🤖 Run `gdoc read` on it.", "`"},
		{"a heading on a later line", "🤖 Two things.\n## Second", "#"},
		{"a fenced block", "🤖 Like this:\n```\ngdoc read\n```", "`"},
		{"a link in brackets", "🤖 See [the register](https://example.com).", "["},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Check(c.body)
			if err == nil {
				t.Fatalf("Check(%q) accepted it", c.body)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal does not name %q: %v", c.says, err)
			}
		})
	}
}

func TestPostSendsTheBodyVerbatimToTheRepliesURL(t *testing.T) {
	f := script()

	if _, err := Post(context.Background(), f, testDocID, testCommentID, goodBody); err != nil {
		t.Fatal(err)
	}

	if len(f.calls) == 0 {
		t.Fatal("nothing was sent")
	}
	post := f.calls[0]
	if post.method != "POST" {
		t.Errorf("the first call is %s, not the reply", post.method)
	}
	u, err := url.Parse(post.url)
	if err != nil {
		t.Fatal(err)
	}
	wantPath := "/drive/v3/files/" + testDocID + "/comments/" + testCommentID + "/replies"
	if u.Path != wantPath {
		t.Errorf("the reply went to %q, want %q", u.Path, wantPath)
	}
	// Only fields. Every other parameter on a write is one nobody decided
	// about, and the guard's driveWriteParams is wider than this call needs.
	for name := range u.Query() {
		if name != "fields" {
			t.Errorf("the reply URL carries %q as well as fields", name)
		}
	}
	var sent struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(post.body, &sent); err != nil {
		t.Fatal(err)
	}
	if sent.Content != goodBody {
		// v1 appended [gdoc]. v2's mark is the prefix the skill already wrote,
		// so anything added here is text gdoc put in Nail's document unasked.
		t.Errorf("the body sent is %q, want it verbatim: %q", sent.Content, goodBody)
	}
}

func TestPostReportsTheReplyAndVerifiesItFromTheThread(t *testing.T) {
	f := script()

	got, err := Post(context.Background(), f, testDocID, testCommentID, goodBody)
	if err != nil {
		t.Fatal(err)
	}

	if got.ReplyID != testReplyID {
		t.Errorf("ReplyID is %q, want %q", got.ReplyID, testReplyID)
	}
	if got.Created != testCreated {
		t.Errorf("Created is %q, want %q", got.Created, testCreated)
	}
	if !got.Verified {
		t.Errorf("the reply is in the thread and Verified is false: %v", got.Warnings)
	}
	if len(f.calls) != 2 {
		t.Fatalf("the run made %d calls, want the write and one read-back", len(f.calls))
	}
	if f.calls[1].method != "GET" {
		t.Errorf("the read-back is a %s", f.calls[1].method)
	}
}

func TestPostReportsTheReplyIDWhenTheThreadDoesNotShowIt(t *testing.T) {
	f := script()
	f.listing = listingWith() // Drive answered the write and shows no reply

	got, err := Post(context.Background(), f, testDocID, testCommentID, goodBody)
	if err != nil {
		t.Fatal(err)
	}

	if got.Verified {
		t.Error("the thread does not carry the reply and Verified is true")
	}
	if got.ReplyID != testReplyID {
		// The write happened. Losing its id is how somebody posts it twice.
		t.Errorf("ReplyID is %q, want it reported anyway", got.ReplyID)
	}
	if len(got.Warnings) == 0 {
		t.Error("an unverified reply carries no warning saying so")
	}
}

func TestPostDoesNotVerifyAReplyWhoseTextCameBackDifferent(t *testing.T) {
	f := script()
	f.listing = listingWith(comments.RawReply{
		ID:          testReplyID,
		CreatedTime: testCreated,
		Content:     "🤖 something else entirely",
	})

	got, err := Post(context.Background(), f, testDocID, testCommentID, goodBody)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verified {
		t.Error("the thread carries other words under that id and Verified is true")
	}
}

func TestPostRefusesMarkdownBeforeAnythingIsSent(t *testing.T) {
	f := script()

	_, err := Post(context.Background(), f, testDocID, testCommentID, "🤖 The **2026** register.")
	if err == nil {
		t.Fatal("a markdown body was posted")
	}
	if !strings.Contains(err.Error(), "**") {
		t.Errorf("the refusal does not name the offending text: %v", err)
	}
	if len(f.calls) != 0 {
		t.Errorf("the run made %d calls after refusing the body", len(f.calls))
	}
}

func TestPostCarriesAGuardRefusalToTheCaller(t *testing.T) {
	f := script()
	refusal := errors.New("file DOC9 was not given to this command")
	f.failAt[0] = refusal

	_, err := Post(context.Background(), f, testDocID, testCommentID, goodBody)
	if !errors.Is(err, refusal) {
		t.Fatalf("the refusal did not reach the caller: %v", err)
	}
}

func TestPostSaysSoWhenDriveAnswersWithNoReplyID(t *testing.T) {
	f := script()
	f.created = []byte(`{}`)

	got, err := Post(context.Background(), f, testDocID, testCommentID, goodBody)
	if err != nil {
		// An error here would make the skill repost, and Drive may well have
		// taken the first one. Unverified is the honest answer.
		t.Fatalf("an answer with no id came back as an error: %v", err)
	}
	if got.Verified {
		t.Error("a reply with no id was reported as verified")
	}
	if len(got.Warnings) == 0 {
		t.Error("an answer with no reply id carries no warning")
	}
}

func TestAReadBackThatFailedIsAWarningRatherThanAnError(t *testing.T) {
	f := script()
	f.failAt[1] = errors.New("the network went away")

	got, err := Post(context.Background(), f, testDocID, testCommentID, goodBody)
	if err != nil {
		t.Fatalf("a failed read-back came back as an error: %v", err)
	}
	if got.Verified {
		t.Error("a read-back that never happened verified the reply")
	}
	if len(got.Warnings) == 0 {
		t.Error("a failed read-back carries no warning")
	}
}

// TestTheGuardCarriesBothOfThePostsRequests is the check that costs nothing to
// run and everything to be missing. A URL that is right for Drive and wrong for
// the guard fails inside the process, on Nail's machine, on the first real run.
func TestTheGuardCarriesBothOfThePostsRequests(t *testing.T) {
	f := script()
	if _, err := Post(context.Background(), f, testDocID, testCommentID, goodBody); err != nil {
		t.Fatal(err)
	}

	p := guard.NewPolicy()
	p.AllowFile(testDocID, guard.LevelSuggest)
	for i, c := range f.calls {
		u, err := url.Parse(c.url)
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		if err := p.Judge(c.method, u, c.body); err != nil {
			t.Errorf("request %d, %s %s: %v", i, c.method, c.url, err)
		}
	}
}

// TestTheGuardRefusesAReplyOnADocumentNobodyNamed is the other half. A reply is
// a write into somebody's document, and the only documents reachable are the
// ones the command was given.
func TestTheGuardRefusesAReplyOnADocumentNobodyNamed(t *testing.T) {
	p := guard.NewPolicy()
	p.AllowFile(testDocID, guard.LevelSuggest)

	u, err := url.Parse(CreateURL("1OtHeR00000000000000000000000000000000", testCommentID))
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Judge("POST", u, []byte(`{"content":"🤖 hello"}`)); err == nil {
		t.Fatal("a reply was carried into a document the command was not given")
	}
}
