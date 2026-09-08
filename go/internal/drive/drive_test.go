package drive

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const testFileID = "1PuBl15h0000000000000000000000000000000"

// call is one request the fake was asked to make.
type call struct {
	method string
	url    string
	body   json.RawMessage
}

// fakeSession answers by call shape rather than by position, the way the fakes
// in probe and propose do, so a test that changes how far a run gets does not
// have to re-count the answers behind it.
type fakeSession struct {
	calls   []call
	trashed []byte
	failAt  map[int]error
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
	if err := f.failAt[len(f.calls)-1]; err != nil {
		return err
	}
	if into == nil {
		return nil
	}
	answer := f.trashed
	if len(answer) == 0 {
		answer = []byte(`{}`)
	}
	return json.Unmarshal(answer, into)
}

func (f *fakeSession) GetJSON(_ context.Context, rawURL string, into any) error {
	return f.do("GET", rawURL, nil, into)
}

func (f *fakeSession) PatchJSON(_ context.Context, rawURL string, body any, into any) error {
	return f.do("PATCH", rawURL, body, into)
}

func script() *fakeSession {
	return &fakeSession{trashed: []byte(`{"trashed":true}`), failAt: map[int]error{}}
}

func TestTrashPatchesThenConfirms(t *testing.T) {
	f := script()

	if err := Trash(context.Background(), f, testFileID); err != nil {
		t.Fatal(err)
	}

	want := []call{
		{method: "PATCH", url: "https://www.googleapis.com/drive/v3/files/" + testFileID + "?supportsAllDrives=true"},
		{method: "GET", url: "https://www.googleapis.com/drive/v3/files/" + testFileID + "?fields=trashed&supportsAllDrives=true"},
	}
	if len(f.calls) != len(want) {
		t.Fatalf("the fake saw %d requests, want %d: a trash is the write and the read that confirms it", len(f.calls), len(want))
	}
	for i, w := range want {
		if f.calls[i].method != w.method || f.calls[i].url != w.url {
			t.Errorf("call %d = %s %s, want %s %s", i, f.calls[i].method, f.calls[i].url, w.method, w.url)
		}
	}
	if got := string(f.calls[0].body); got != `{"trashed":true}` {
		t.Errorf("the write carried %s, want the one field Drive spells trashing as", got)
	}
}

// A 500 is written and may have been applied, so the failed PATCH claims nothing
// about where the file is. Saying it is still where it was would send somebody
// looking for a document Drive has already trashed.
func TestAFailedPatchClaimsNothingAboutWhereTheFileIs(t *testing.T) {
	f := script()
	f.failAt[0] = errors.New("files.update answered 500")

	err := Trash(context.Background(), f, testFileID)

	if err == nil {
		t.Fatal("a refused trash came back as success")
	}
	if !strings.Contains(err.Error(), "the trash request failed") {
		t.Errorf("the error does not name the failed request: %v", err)
	}
	if strings.Contains(err.Error(), "still where it was") {
		t.Errorf("the error claims the file was left in place, and a 500 cannot say that: %v", err)
	}
	if !strings.Contains(err.Error(), "answered 500") {
		t.Errorf("the error lost what Drive said: %v", err)
	}
	if len(f.calls) != 1 {
		t.Errorf("the fake saw %d requests, want one: nothing is confirmed after a refused write", len(f.calls))
	}
}

func TestAConfirmationThatCouldNotBeReadIsAFailure(t *testing.T) {
	f := script()
	f.failAt[1] = errors.New("files.get answered 503")

	err := Trash(context.Background(), f, testFileID)

	if err == nil {
		t.Fatal("a trash nobody could confirm came back as success")
	}
	if !strings.Contains(err.Error(), "confirm") {
		t.Errorf("the error does not name the confirmation as the step that failed: %v", err)
	}
}

func TestDriveSayingItIsNotTrashedIsAFailureRatherThanASuccess(t *testing.T) {
	f := script()
	f.trashed = []byte(`{"trashed":false}`)

	err := Trash(context.Background(), f, testFileID)

	if err == nil {
		t.Fatal("Drive contradicted the write and Trash answered nil; the read-back is what is believed, never the PATCH")
	}
	if !strings.Contains(err.Error(), "not trashed") {
		t.Errorf("the error does not say what Drive reported: %v", err)
	}
}
