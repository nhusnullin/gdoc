package comments

import (
	"encoding/json"
	"strings"
	"testing"
)

// Drive leaves a comment author's emailAddress empty, the signed-in person's own
// included (MEASURED.md, "gdoc mcp in Claude Desktop, the first run"). An empty
// author_domain read as a fact made a model call the author "probably a personal
// account". So the key is left out when Drive gave no address, and an absent key
// says only that Google did not say.
func TestAnUnknownAuthorDomainIsLeftOut(t *testing.T) {
	thread := Thread{ID: "AAAA1111", Author: "Nail", Replies: []Reply{{ID: "R1", Author: "Nail"}}}
	b, err := json.Marshal(thread)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "author_domain") {
		t.Errorf("a thread and a reply with no known domain still carry author_domain: %s", b)
	}

	thread.AuthorDomain = "example.com"
	thread.Replies[0].AuthorDomain = "example.org"
	b, err = json.Marshal(thread)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"author_domain":"example.com"`, `"author_domain":"example.org"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("a known domain is not reported as %s: %s", want, b)
		}
	}
}
