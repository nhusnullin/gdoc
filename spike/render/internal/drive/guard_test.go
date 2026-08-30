package drive

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

type recorder struct {
	calls    []string
	response *http.Response
}

func (r *recorder) RoundTrip(request *http.Request) (*http.Response, error) {
	r.calls = append(r.calls, request.URL.Path)
	if r.response != nil {
		return r.response, nil
	}
	return &http.Response{StatusCode: 200, Header: http.Header{},
		Body: io.NopCloser(strings.NewReader("{}"))}, nil
}

func get(t *testing.T, guard *Guard, path string) error {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, "https://www.googleapis.com"+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = guard.RoundTrip(request)
	return err
}

func TestAFileTheClientWasGivenIsReached(t *testing.T) {
	inner := &recorder{}
	guard := NewGuard(inner, []string{"FOLDER"})

	if err := get(t, guard, "/drive/v3/files/FOLDER"); err != nil {
		t.Fatalf("a file in the set must be reachable: %v", err)
	}
	if len(inner.calls) != 1 {
		t.Errorf("calls = %v, want the request carried", inner.calls)
	}
}

func TestAFileTheClientWasNeverGivenIsRefused(t *testing.T) {
	// A refused call usually means the command did not say which document it
	// was for. The set has exactly two doors, and this is neither.
	inner := &recorder{}
	guard := NewGuard(inner, []string{"FOLDER"})

	err := get(t, guard, "/drive/v3/files/SOMEONE-ELSES-DOC")

	if err == nil {
		t.Fatal("a file outside the set must be refused")
	}
	if !strings.Contains(err.Error(), "SOMEONE-ELSES-DOC") {
		t.Errorf("the refusal must name the file, got %q", err)
	}
	if len(inner.calls) != 0 {
		t.Errorf("the request reached the network anyway: %v", inner.calls)
	}
}

func TestACreateNamesNoFileSoItIsCarried(t *testing.T) {
	inner := &recorder{}
	guard := NewGuard(inner, []string{"FOLDER"})
	request, _ := http.NewRequest(http.MethodPost,
		"https://www.googleapis.com/upload/drive/v3/files", nil)

	if _, err := guard.RoundTrip(request); err != nil {
		t.Fatalf("a create addresses no file, so it must be carried: %v", err)
	}
}

func TestTheSecondDoorIsACreateComingBack(t *testing.T) {
	// The set grows only when a create the guard itself carried comes back with
	// an id. That is the second door, and the only other one.
	body := `{"id":"NEW-DOC","webViewLink":"https://docs.google.com/x"}`
	inner := &recorder{response: &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
	}}
	guard := NewGuard(inner, []string{"FOLDER"})
	request, _ := http.NewRequest(http.MethodPost,
		"https://www.googleapis.com/upload/drive/v3/files", nil)

	response, err := guard.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}

	// The body must still be readable by the caller downstream.
	read, _ := io.ReadAll(response.Body)
	if string(read) != body {
		t.Errorf("body = %q, want it restored after the peek", read)
	}
	if !guard.permitted("NEW-DOC") {
		t.Fatal("the created document did not join the set")
	}
	if err := get(t, guard, "/drive/v3/files/NEW-DOC"); err != nil {
		t.Errorf("the created document is not reachable: %v", err)
	}
}

func TestAnExportOfAFileOutsideTheSetIsRefused(t *testing.T) {
	inner := &recorder{}
	guard := NewGuard(inner, []string{"FOLDER"})

	if err := get(t, guard, "/drive/v3/files/OTHER/export"); err == nil {
		t.Fatal("an export names the file it exports, so it must be checked")
	}
}
