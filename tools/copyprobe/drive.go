package main

// The calls. Every one carries supportsAllDrives=true, because the test folder
// is on a shared drive and Drive answers a flat 404 without it.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (c *client) do(method, url string, body any) ([]byte, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.bearer)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("%s answered %d: %s", url, resp.StatusCode, oneLine(string(out)))
	}
	return out, nil
}

func (c *client) getJSON(url string, into any) error {
	b, err := c.do("GET", url, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, into)
}

func (c *client) postJSON(url string, body, into any) error {
	b, err := c.do("POST", url, body)
	if err != nil {
		return err
	}
	if into == nil {
		return nil
	}
	return json.Unmarshal(b, into)
}

// buildSource makes the document the whole measurement rests on: a sentence,
// a comment anchored to two words inside it, and a pending suggestion.
//
// The comment is created through the Docs API's insertComment, in the same
// batch that the anchoring range is known. Drive's comments.create cannot do
// this: Google documents that the Workspace editors treat its custom anchors as
// unanchored, so a comment made that way would prove nothing.
func (c *client) buildSource(folder string) (string, error) {
	var created struct {
		ID string `json:"id"`
	}
	err := c.postJSON("https://www.googleapis.com/drive/v3/files?fields=id&supportsAllDrives=true",
		map[string]any{
			"name":     "gdoc COPY PROBE source - does copyComments carry an anchor",
			"mimeType": "application/vnd.google-apps.document",
			"parents":  []string{folder},
		}, &created)
	if err != nil {
		return "", fmt.Errorf("the source document could not be created: %w", err)
	}
	if created.ID == "" {
		return "", fmt.Errorf("Drive accepted the create and returned no id")
	}
	id := created.ID

	if err := c.batch(id, nil, map[string]any{
		"insertText": map[string]any{
			"location": map[string]any{"index": 1},
			"text":     sentence,
		},
	}); err != nil {
		return id, fmt.Errorf("the sentence could not be written: %w", err)
	}

	// The anchored comment. Index 1 is the start of the body, so the words sit
	// at 1+offset, counted in UTF-16 units, which for this sentence is bytes.
	start := 1 + strings.Index(sentence, anchored)
	end := start + len(anchored)
	if err := c.batch(id, nil, map[string]any{
		// content, not text, and range beside it at the top of the request.
		// That shape is measured rather than documented: every other spelling
		// answers "Cannot find field" (DECISIONS.md, 2026-08-29), and
		// internal/propose sends exactly this.
		"insertComment": map[string]any{
			"range":   map[string]any{"startIndex": start, "endIndex": end},
			"content": "copyprobe: this comment is anchored to the word above.",
		},
	}); err != nil {
		return id, fmt.Errorf("the anchored comment could not be made: %w", err)
	}

	// The pending suggestion, written in SUGGEST mode so it stays pending.
	if err := c.batch(id, map[string]any{"writeMode": "SUGGEST"}, map[string]any{
		"insertText": map[string]any{
			"location": map[string]any{"index": end},
			"text":     suggest,
		},
	}); err != nil {
		return id, fmt.Errorf("the pending suggestion could not be made: %w", err)
	}
	return id, nil
}

func (c *client) batch(id string, writeControl map[string]any, requests ...map[string]any) error {
	body := map[string]any{"requests": requests}
	if writeControl != nil {
		body["writeControl"] = writeControl
	}
	return c.postJSON("https://docs.googleapis.com/v1/documents/"+id+":batchUpdate", body, nil)
}

// copy is the call the whole program exists to test. param is either
// "&copyComments=true" or empty, and the empty one is the control.
func (c *client) copy(src, folder, param string) (string, error) {
	var out struct {
		ID string `json:"id"`
	}
	url := "https://www.googleapis.com/drive/v3/files/" + src +
		"/copy?fields=id&supportsAllDrives=true" + param
	err := c.postJSON(url, map[string]any{
		"name":    "gdoc COPY PROBE copy" + param,
		"parents": []string{folder},
	}, &out)
	if err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", fmt.Errorf("Drive accepted the copy and returned no id")
	}
	return out.ID, nil
}

func (c *client) exportDocx(id string) ([]byte, error) {
	return c.do("GET", "https://www.googleapis.com/drive/v3/files/"+id+
		"/export?mimeType=application%2Fvnd.openxmlformats-officedocument.wordprocessingml.document", nil)
}

func (c *client) trash(id string) error {
	req, err := http.NewRequest("PATCH", "https://www.googleapis.com/drive/v3/files/"+id+
		"?supportsAllDrives=true", strings.NewReader(`{"trashed":true}`))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.bearer)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("Drive answered %d", resp.StatusCode)
	}
	return nil
}
