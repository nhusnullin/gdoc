package guard

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// counting answers every create with a distinct id, so several creates in
// flight at once make the policy learn from several goroutines.
type counting struct {
	mu sync.Mutex
	n  int
}

func (c *counting) RoundTrip(r *http.Request) (*http.Response, error) {
	c.mu.Lock()
	c.n++
	id := fmt.Sprintf("MADE%d", c.n)
	c.mu.Unlock()
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"id":"` + id + `"}`)),
		Request:    r,
	}, nil
}

// TestOneClientIsSafeFromManyGoroutines is the contract NewClient signs by
// returning an *http.Client: the standard library documents a client as safe
// for concurrent use. Learn writes the file set from inside RoundTrip while
// Judge reads it on every request, so without a lock this is a concurrent map
// read and write, which Go turns into a fatal error nothing can recover from.
// Run it under -race.
func TestOneClientIsSafeFromManyGoroutines(t *testing.T) {
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	p.AllowFile("DOC1", LevelSuggest)
	c := NewClient(p, &counting{})

	const each = 8
	var wg sync.WaitGroup
	for i := 0; i < each; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := c.Post("https://www.googleapis.com/drive/v3/files",
				"application/json", bytes.NewReader([]byte(`{"parents":["FOLDER1"]}`)))
			if err != nil {
				t.Error(err)
				return
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := c.Get("https://docs.googleapis.com/v1/documents/DOC1")
			if err != nil {
				t.Error(err)
				return
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("GET", "https://docs.googleapis.com/v1/documents/DOC1", nil)
			next, _ := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files/DOC1", nil)
			_ = c.CheckRedirect(next, []*http.Request{req})
		}()
	}
	wg.Wait()

	if got := len(p.Warnings()); got != 0 {
		t.Fatalf("no create should have gone unlearned: %v", p.Warnings())
	}
}
