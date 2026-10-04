// The release line a chat hears: once in a process, built from the versions
// the daily stamp holds, and never from a checkout build.
//
// Nothing here reaches GitHub. The reach is notice_test.go's own stub, so a
// test that says nothing was fetched can prove it, and the stamp lives in a
// config directory of the test's own.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"gdoc/internal/guard"
	"gdoc/internal/lastcheck"
)

// guideCalls runs one session of n guide calls and answers the text items of
// each answer, in order. guide is the tool it calls because guide reaches no
// token and no document, so what is measured is the line and nothing else.
func guideCalls(t *testing.T, n int) [][]string {
	t.Helper()
	lines := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"v"}}}`,
	}
	for i := 0; i < n; i++ {
		lines = append(lines, `{"jsonrpc":"2.0","id":`+strconv.Itoa(2+i)+`,"method":"tools/call","params":{"name":"guide","arguments":{}}}`)
	}
	lines = append(lines, "")

	var out, errOut strings.Builder
	if code := serveMCP(context.Background(), strings.NewReader(strings.Join(lines, "\n")), &out, &errOut, nil); code != 0 {
		t.Fatalf("the session exited %d: %s", code, errOut.String())
	}
	answers := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(answers) != n+1 {
		t.Fatalf("%d answers were asked for and %d came back: %q", n+1, len(answers), out.String())
	}

	var texts [][]string
	for _, answer := range answers[1:] {
		var parsed struct {
			Result struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(answer), &parsed); err != nil {
			t.Fatalf("an answer is not JSON: %v", err)
		}
		var items []string
		for _, c := range parsed.Result.Content {
			items = append(items, c.Text)
		}
		texts = append(texts, items)
	}
	return texts
}

// No record of an earlier check, so the first tool answer of the session asks
// once, writes down what it heard, and carries the line as one more text item.
// The second answer carries nothing: a chat hears this once.
func TestAStaleStampAsksOnceAndTheFirstAnswerCarriesTheLine(t *testing.T) {
	pl := &checkPlain{listing: listing("v2.9.0", "v2.8.0")}
	path := checking(t, "v2.8.0", pl)

	texts := guideCalls(t, 2)
	if len(pl.got) != 1 {
		t.Fatalf("a session asks once and this one asked %d times: %v", len(pl.got), pl.got)
	}
	if len(texts[0]) != 2 {
		t.Fatalf("the first answer is the envelope and the line, and it has %d items: %v", len(texts[0]), texts[0])
	}
	// The literal, because this is the whole of what a person hears about a
	// release from inside a chat.
	want := "gdoc v2.9.0 is published and this is v2.8.0. Run gdoc update in a terminal, then quit Claude Desktop and open it again."
	if texts[0][1] != want {
		t.Errorf("the line is\n got %q\nwant %q", texts[0][1], want)
	}
	if len(texts[1]) != 1 {
		t.Errorf("the second answer says it again: %v", texts[1])
	}
	s := stampOnDisk(t, path)
	if s.LatestStable != "v2.9.0" || s.Error != "" {
		t.Errorf("the stamp holds what was heard: %+v", s)
	}
}

// A check younger than the interval asks nothing, and the line still comes,
// out of the file, on the first answer and only there. The 24 hours and the
// stamp are the ones help uses, so a chat and a terminal cost one request a
// day between them.
func TestAFreshStampShowingANewerReleaseStillGivesTheLineOncePerProcess(t *testing.T) {
	path := checking(t, "v2.8.0", &refusingPlain{t: t})
	stampedAt(t, path, 23*time.Hour, lastcheck.Stamp{LatestStable: "v2.9.0", LatestNightly: "v2.9.0"})

	texts := guideCalls(t, 3)
	if len(texts[0]) != 2 || !strings.Contains(texts[0][1], "gdoc v2.9.0 is published and this is v2.8.0") {
		t.Fatalf("the first answer carries the line out of the stamp: %v", texts[0])
	}
	for i, answer := range texts[1:] {
		if len(answer) != 1 {
			t.Errorf("answer %d says it again: %v", i+2, answer)
		}
	}
}

// A build from a checkout names no release, so there is nothing to compare,
// nothing to ask and nothing to say. It writes no stamp either.
func TestACheckoutBuildNeverAsksFromMcp(t *testing.T) {
	path := checking(t, "dev", &refusingPlain{t: t})

	texts := guideCalls(t, 2)
	for i, answer := range texts {
		if len(answer) != 1 {
			t.Errorf("answer %d carries more than the envelope: %v", i+1, answer)
		}
		if strings.Contains(strings.Join(answer, " "), "is published") {
			t.Errorf("a checkout build says nothing about a release: %v", answer)
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("a checkout build writes no stamp, and %s is there: %v", path, err)
	}
}

// The ceiling is two seconds, the one help's check has, so the first tool
// answer of a session on a network that cannot reach GitHub is that much late
// and no more.
func TestTheMcpCheckIsBoundedByTwoSeconds(t *testing.T) {
	pl := &checkPlain{listing: listing("v2.9.0")}
	checking(t, "v2.8.0", pl)

	guideCalls(t, 1)
	if len(pl.deadlines) != 1 {
		t.Fatalf("the check makes one request under a deadline, and made %d: %v", len(pl.deadlines), pl.deadlines)
	}
	if d := pl.deadlines[0].Sub(pl.asked[0]); d > 2*time.Second || d <= 0 {
		t.Errorf("the check is bounded by two seconds and this one had %v", d)
	}
}

// The policy the session's check opens is the fifth grant and nothing else:
// the same one help's check and the update open, judged the same way. A
// session that read a document under this policy would be a widening of the
// wire by a command nobody typed.
func TestTheMcpCheckOpensThePolicyTheUpdateOpens(t *testing.T) {
	pl := &checkPlain{listing: listing("v2.9.0")}
	checking(t, "v2.8.0", pl)
	var opened *guard.Policy
	openPlain = func(p *guard.Policy) plain { opened = p; return pl }

	guideCalls(t, 1)
	if opened == nil {
		t.Fatal("the check opened no policy")
	}
	judge := func(method, rawURL string) error {
		u, err := url.Parse(rawURL)
		if err != nil {
			t.Fatal(err)
		}
		return opened.Judge(method, u, nil)
	}
	if err := judge("GET", "https://api.github.com/repos/nhusnullin/gdoc/releases"); err != nil {
		t.Errorf("the listing is the one call the check makes: %v", err)
	}
	for _, refused := range [][2]string{
		{"GET", "https://api.github.com/repos/someone/else/releases"},
		{"POST", "https://api.github.com/repos/nhusnullin/gdoc/releases"},
		{"GET", "https://docs.googleapis.com/v1/documents/1AbCdEfGhIjKlMnOpQrStUvWxYz012345"},
		{"GET", "https://www.googleapis.com/drive/v3/files/1AbCdEfGhIjKlMnOpQrStUvWxYz012345"},
		{"GET", "https://www.googleapis.com/drive/v3/about?fields=user(emailAddress,displayName)"},
	} {
		if err := judge(refused[0], refused[1]); err == nil {
			t.Errorf("%s %s must be refused by the session's check policy", refused[0], refused[1])
		}
	}
}

// What the line tells a person to do, and what it never says. A toggle of the
// connector restarts the chat process and leaves agent mode on the old binary
// (docs/v2/MEASURED.md, measurement 2), so the words are quit and open again.
// A major is named with the flag that installs it, because `gdoc update`
// alone would not.
func TestTheLineSaysQuitAndOpenAgainNeverToggle(t *testing.T) {
	for _, c := range []struct {
		name, installed, stable, want string
	}{
		{
			name: "a minor", installed: "v2.8.0", stable: "v2.9.0",
			want: "gdoc v2.9.0 is published and this is v2.8.0. Run gdoc update in a terminal, then quit Claude Desktop and open it again.",
		},
		{
			name: "a major", installed: "v2.8.0", stable: "v3.0.0",
			want: "gdoc v3.0.0 is published and this is v2.8.0. It is a major release: run gdoc update --major in a terminal, then quit Claude Desktop and open it again.",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			path := checking(t, c.installed, &refusingPlain{t: t})
			stampedAt(t, path, time.Hour, lastcheck.Stamp{LatestStable: c.stable, LatestNightly: c.stable})

			texts := guideCalls(t, 1)
			if len(texts[0]) != 2 {
				t.Fatalf("the first answer carries the line: %v", texts[0])
			}
			line := texts[0][1]
			if line != c.want {
				t.Errorf("the line is\n got %q\nwant %q", line, c.want)
			}
			for _, never := range []string{"toggle", "Toggle", "restart", "Restart", "reconnect"} {
				if strings.Contains(line, never) {
					t.Errorf("the line says %q, and a %s does not give a chat the new binary: %q", never, never, line)
				}
			}
		})
	}

	// Nothing newer is no line at all, in a chat as in a terminal.
	path := checking(t, "v2.9.0", &refusingPlain{t: t})
	stampedAt(t, path, time.Hour, lastcheck.Stamp{LatestStable: "v2.9.0", LatestNightly: "v2.9.0"})
	texts := guideCalls(t, 1)
	if len(texts[0]) != 1 {
		t.Errorf("there is nothing newer, so there is nothing to say: %v", texts[0])
	}
}

// ctxPlain answers the listing unless the context it was handed is already
// done. It is the one thing a stub of the wire has to copy here, because the
// whole question is whose cancellation the check rides on.
type ctxPlain struct {
	listing string
	got     []string
}

func (c *ctxPlain) GetJSON(ctx context.Context, rawURL string, into any) error {
	c.got = append(c.got, rawURL)
	if err := ctx.Err(); err != nil {
		return err
	}
	return json.Unmarshal([]byte(c.listing), into)
}

func (c *ctxPlain) GetBytes(_ context.Context, rawURL string, _ int64) ([]byte, error) {
	return nil, fmt.Errorf("a check downloads nothing, and this one asked for %s", rawURL)
}

func (c *ctxPlain) Warnings() []string { return nil }

// A client that stops the turn its first tool call is in cancels that call's
// context. The check is not that call's work: it rides on it, and a failure
// stamped from a cancellation costs the day's check for both processes Claude
// Desktop started, and the once is spent for the life of this one. So the check
// is handed a context of its own, which keeps the two-second ceiling and leaves
// the stamp out of the call's hands.
func TestACancelledFirstCallStillLeavesTheDaysCheckUnspent(t *testing.T) {
	pl := &ctxPlain{listing: listing("v2.9.0", "v2.8.0")}
	path := checking(t, "v2.8.0", pl)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var errOut strings.Builder
	line := (&mcpNotice{}).first(ctx, &errOut)
	if line == "" {
		t.Fatalf("the check ran and said nothing: %q", errOut.String())
	}
	s := stampOnDisk(t, path)
	if s.Error != "" {
		t.Errorf("a cancelled call must not stamp a failure: %q", s.Error)
	}
	if s.LatestStable != "v2.9.0" {
		t.Errorf("the check read the listing, so the stamp holds it: %+v", s)
	}
}
