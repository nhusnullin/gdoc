# gdoc v2 Milestone 1: Foundation Implementation Plan

## Overview

A Go binary `gdoc` that exists, speaks the JSON envelope, owns the network through a guard with write levels, and can log in, refresh and report its OAuth state. It can refuse everything before it can do anything.

The problem it solves: v1 is Python, and its guard was fitted around a client that already existed. v2 starts from the guard, so no outbound request exists before a policy that can refuse it. This milestone builds nothing that talks to Docs yet. It builds the refusal, the envelope, and the credential.

How it integrates: the module lives at `go/`, beside the existing Python package. Nothing under `gdoc/` changes. The Go binary reads v1's token file (`oauth-token.json`, same format), so a person already logged in through v1 is already logged in here.

Spec: `docs/v2/SPEC.md` (agreed 2026-08-29). Master plan: `docs/v2/PLAN.md`.

## Context (from discovery)

- Files and components involved: a new tree at `go/` holding `internal/emit`, `internal/config`, `internal/guard`, `internal/auth`, `internal/auth/loopback`, `cmd/gdoc` and `boundary/`, plus a `Makefile` at the repo root.
- Related patterns found: `gdoc/guard.py` (the reachable set, and its rule that ids enter through exactly two doors), `gdoc/auth.py` (`resolve_auth_mode`, the config and token paths), `gdoc/oauth.py` lines 51-52 (the bundled client constants this milestone copies verbatim).
- Dependencies identified: none. Go 1.27 standard library only.

## Development Approach

- **Testing approach**: TDD. Every task writes the failing test first, runs it to watch it fail, then implements until it passes.
- Complete each task fully before moving to the next. Each task ends in its own commit.
- Make small, focused changes.
- **CRITICAL: every task MUST include new or updated tests** for the code it changes. Tests are a required deliverable of the task, not an optional extra.
- **CRITICAL: all tests must pass before the next task starts.** No exceptions.
- **CRITICAL: update this plan file when scope changes during implementation.**
- Run the validation commands after each change.

## Testing Strategy

- **Unit tests**: required for every task. Table-driven wherever the input space is a set of cases, as in the guard policy.
- **Boundary test**: `go/boundary/` holds an import allowlist test. It is an allowlist in both directions. It fails when `net/http` spreads to a package that may not have it, and it fails when a package that should have it stops importing it.
- **E2E tests**: this project has no UI and no browser e2e suite, so there are none to add. Task 7 ends with a real `auth status` run against the machine's own token instead, which is the closest thing to end to end here.
- Coverage standard: every exported function under `internal/` has a test.

## Validation Commands

- `cd go && go test ./...`
- `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- `cd go && go vet ./...`

## Progress Tracking

- Mark completed items with `[x]` as soon as they are done.
- Add newly discovered tasks with a ➕ prefix.
- Record issues and blockers with a ⚠️ prefix.
- Update the plan if the implementation deviates from the original scope.

## Solution Overview

One module at `go/`. One package, `internal/guard`, owns all outbound HTTP as a policy plus a transport. `internal/auth` handles the token file and its refresh through the guard's client; the loopback login listener under `internal/auth/loopback` is the one other place `net/http` may appear, and the boundary test enforces that. Every command prints exactly one JSON object to stdout.

Key design decisions and why:

- The guard is constructed before anything that could reach the network, so the first request in the program's history has already passed a policy. Fitting a guard on later is how v1 got there, and it is why v1 needs a test to prove `build()` is called in only one module.
- Write levels live in the policy, not at the call site. A handed-in document id is readable and never editable. A create is refused unless it names a folder the run was given.
- The envelope is one package that every command prints through, so "one JSON object on stdout" is a property of a type rather than a convention each command has to remember.
- Login prints the URL to stderr instead of opening a browser. That is what lets v2 ban `os/exec` outright.

## Technical Details

**Principles.** Serves: 1 (static binary, stdlib only), 3 (the guard exists before the first network call, and uncertainty refuses), 4 (stdout carries one machine object, human words go to stderr). Strains: none.

**Tech stack.** Go 1.27, standard library only in this milestone. No `golang.org/x/oauth2`. No `os/exec` anywhere in v2, which is stronger than v1.

**Global constraints:**

- Module path `gdoc`, directory `go/` in this repo. Go `1.27`.
- Third-party dependencies allowed in this milestone: none.
- Every command: exactly one JSON object on stdout, exit 0 iff `ok` is true. Human prose and the login URL go to stderr.
- The binary never prompts and never reads stdin.
- Config dir: `$GDOC_CONFIG_DIR` if set; else `~/.config/gdoc-agent` on darwin; else `%AppData%\gdoc-agent` on windows. Token file name: `oauth-token.json` (v1's file, same format).
- Allowed hosts, exact: `docs.googleapis.com`, `www.googleapis.com`, `oauth2.googleapis.com` (token endpoint only).
- OAuth client id and secret: copy the constant values verbatim from `gdoc/oauth.py` lines 51-52 (`BUNDLED_CLIENT_ID`, `BUNDLED_CLIENT_SECRET`). They are deliberately in version control; see CLAUDE.md.
- Scopes requested by login: `https://www.googleapis.com/auth/drive` and `https://www.googleapis.com/auth/documents` (one browser trip covers both, as v1's `LOGIN_SCOPES`).
- All commits run from the repo root. Test command: `cd go && go test ./...`.
- No em dashes in any text this plan produces.

## What Goes Where

- **Implementation Steps** (`[ ]` checkboxes): everything achievable inside this repo. The Go module, its tests, the Makefile, the documentation updates.
- **Post-Completion** (no checkboxes): the checks that need a real Google account, a real browser trip, or a machine other than this one.

## Implementation Steps

---

### Task 1: Module scaffold and the JSON envelope

**Files:**
- Create: `go/go.mod`
- Create: `go/internal/emit/emit.go`
- Test: `go/internal/emit/emit_test.go`

**Interfaces:**
- Produces: `emit.Result{OK bool, Data any, Error string, Warnings []string}`; `emit.Print(w io.Writer, r Result) error` writes one JSON object plus newline; `emit.ExitCode(r Result) int` (0 iff OK). Later tasks build every command's output through these.

- [x] **Step 1: Create the module**

```bash
mkdir -p go/internal/emit && cd go && go mod init gdoc
```

- [x] **Step 2: Write the failing test**

```go
package emit

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPrintSuccessShape(t *testing.T) {
	var buf bytes.Buffer
	r := Result{OK: true, Data: map[string]any{"n": 1}, Warnings: []string{"w"}}
	if err := Print(&buf, r); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not one JSON object: %v", err)
	}
	if got["ok"] != true || got["error"] != nil {
		t.Fatalf("wrong envelope: %v", got)
	}
	if ExitCode(r) != 0 {
		t.Fatal("success must exit 0")
	}
}

func TestPrintFailureShape(t *testing.T) {
	var buf bytes.Buffer
	r := Result{OK: false, Error: "no token at /x. Run: gdoc auth login"}
	if err := Print(&buf, r); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(buf.Bytes(), &got)
	if got["ok"] != false || got["error"] == "" {
		t.Fatalf("failure must carry ok=false and a message: %v", got)
	}
	if ExitCode(r) != 1 {
		t.Fatal("failure must exit non-zero")
	}
}
```

- [x] **Step 3: Run to verify it fails**

Run: `cd go && go test ./internal/emit/`
Expected: FAIL, `Result` undefined.

- [x] **Step 4: Implement**

```go
// Package emit is the one output contract: every command prints exactly one
// JSON object to stdout and exits 0 iff ok. Spec: "Output contract".
package emit

import (
	"encoding/json"
	"io"
)

type Result struct {
	OK       bool     `json:"ok"`
	Data     any      `json:"data,omitempty"`
	Error    string   `json:"error,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func Print(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(r)
}

func ExitCode(r Result) int {
	if r.OK {
		return 0
	}
	return 1
}
```

- [x] **Step 5: Run to verify it passes, then commit**

Run: `cd go && go test ./internal/emit/` (expect PASS), then:

```bash
git add go/go.mod go/internal/emit/
git commit -m "feat(v2): go module and the JSON output envelope"
```

---

### Task 2: Config paths per platform

**Files:**
- Create: `go/internal/config/config.go`
- Test: `go/internal/config/config_test.go`

**Interfaces:**
- Produces: `config.Dir() (string, error)`, `config.TokenPath() (string, error)`. Honors `GDOC_CONFIG_DIR`; darwin gets `~/.config/gdoc-agent`, windows `%AppData%\gdoc-agent`. Task 6 consumes both.

- [x] **Step 1: Write the failing test**

```go
package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnvOverrideWins(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", "/tmp/gdoc-test-conf")
	d, err := Dir()
	if err != nil || d != "/tmp/gdoc-test-conf" {
		t.Fatalf("got %q, %v", d, err)
	}
	p, _ := TokenPath()
	if p != filepath.Join("/tmp/gdoc-test-conf", "oauth-token.json") {
		t.Fatalf("token path: %q", p)
	}
}

func TestPlatformDefault(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", "")
	d, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	switch runtime.GOOS {
	case "windows":
		if filepath.Base(d) != "gdoc-agent" {
			t.Fatalf("windows dir: %q", d)
		}
	default: // darwin and everything else keep v1's path
		if filepath.Base(filepath.Dir(d)) != ".config" || filepath.Base(d) != "gdoc-agent" {
			t.Fatalf("unix dir: %q", d)
		}
	}
}
```

- [x] **Step 2: Run to verify it fails** (`go test ./internal/config/`, FAIL: undefined)

- [x] **Step 3: Implement**

```go
// Package config decides where gdoc's per-user files live. v1's macOS path
// (~/.config/gdoc-agent) is preserved on purpose; %AppData%\gdoc-agent on
// Windows. GDOC_CONFIG_DIR overrides both, which is also what the test suite
// uses so tests never touch the real config.
package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

const dirName = "gdoc-agent"

func Dir() (string, error) {
	if d := os.Getenv("GDOC_CONFIG_DIR"); d != "" {
		return d, nil
	}
	if runtime.GOOS == "windows" {
		appData := os.Getenv("AppData")
		if appData == "" {
			return "", errors.New("config: %AppData% is not set")
		}
		return filepath.Join(appData, dirName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", dirName), nil
}

func TokenPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "oauth-token.json"), nil
}
```

- [x] **Step 4: Run tests, commit**

```bash
git add go/internal/config/
git commit -m "feat(v2): per-platform config paths, env override for tests"
```

---

### Task 3: The guard policy (pure judgment, no network)

**Files:**
- Create: `go/internal/guard/policy.go`
- Test: `go/internal/guard/policy_test.go`

**Interfaces:**
- Produces:
  - `type Level int` with `LevelSuggest Level = 1` (handed in: read, comment, suggest) and `LevelFull Level = 2` (created by gdoc, or in-place grant).
  - `NewPolicy() *Policy`; `(*Policy).AllowFile(id string, lvl Level)`; `(*Policy).AllowCreateIn(folderID string)`; `(*Policy).GrantInPlace(id string)` (upgrades a handed-in id for this process only); `(*Policy).Learn(id string)` (adds a created id at LevelFull).
  - `(*Policy).Judge(method string, u *url.URL, body []byte) error`. `nil` means carry the request. Task 4 wraps this in the transport.
- The path grammar Judge understands (everything else is refused):
  - `docs.googleapis.com`: `GET /v1/documents/{id}`, `POST /v1/documents/{id}:batchUpdate`.
  - `www.googleapis.com`: `POST /drive/v3/files` and `POST /upload/drive/v3/files*` (create; needs `AllowCreateIn`), `GET|PATCH /drive/v3/files/{id}`, `GET /drive/v3/files/{id}/export`, `GET|POST /drive/v3/files/{id}/comments*` and deeper reply paths.
  - `oauth2.googleapis.com`: `POST /token` only.
  - `GET /drive/v3/files` with no id (listing) is refused outright.

- [ ] **Step 1: Write the failing table test**

```go
package guard

import (
	"net/url"
	"testing"
)

func mustURL(t *testing.T, s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestJudge(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest) // handed in on the command line
	p.AllowCreateIn("FOLDER1")
	p.Learn("MADE1") // came back from a create gdoc carried

	suggestBody := []byte(`{"requests":[],"writeControl":{"writeMode":"SUGGEST"}}`)
	editBody := []byte(`{"requests":[]}`)

	cases := []struct {
		name, method, url string
		body              []byte
		ok                bool
	}{
		{"read known doc", "GET", "https://docs.googleapis.com/v1/documents/DOC1", nil, true},
		{"read unknown doc", "GET", "https://docs.googleapis.com/v1/documents/EVIL", nil, false},
		{"suggest on handed-in", "POST", "https://docs.googleapis.com/v1/documents/DOC1:batchUpdate", suggestBody, true},
		{"direct edit on handed-in refused", "POST", "https://docs.googleapis.com/v1/documents/DOC1:batchUpdate", editBody, false},
		{"direct edit on created ok", "POST", "https://docs.googleapis.com/v1/documents/MADE1:batchUpdate", editBody, true},
		{"files.list refused", "GET", "https://www.googleapis.com/drive/v3/files?q=x", nil, false},
		{"create allowed with folder", "POST", "https://www.googleapis.com/drive/v3/files", nil, true},
		{"export known", "GET", "https://www.googleapis.com/drive/v3/files/DOC1/export?mimeType=x", nil, true},
		{"comments on handed-in", "POST", "https://www.googleapis.com/drive/v3/files/DOC1/comments?fields=x", nil, true},
		{"trash created", "PATCH", "https://www.googleapis.com/drive/v3/files/MADE1", []byte(`{"trashed":true}`), true},
		{"patch handed-in refused", "PATCH", "https://www.googleapis.com/drive/v3/files/DOC1", []byte(`{"trashed":true}`), false},
		{"token endpoint", "POST", "https://oauth2.googleapis.com/token", nil, true},
		{"wrong host", "GET", "https://evil.example.com/v1/documents/DOC1", nil, false},
		{"http not https", "GET", "http://docs.googleapis.com/v1/documents/DOC1", nil, false},
		{"unknown path shape", "GET", "https://www.googleapis.com/drive/v3/about", nil, false},
	}
	for _, c := range cases {
		err := p.Judge(c.method, mustURL(t, c.url), c.body)
		if (err == nil) != c.ok {
			t.Errorf("%s: got err=%v, want ok=%v", c.name, err, c.ok)
		}
	}
}

func TestGrantInPlaceUpgrades(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	edit := []byte(`{"requests":[]}`)
	u := mustURL(t, "https://docs.googleapis.com/v1/documents/DOC1:batchUpdate")
	if p.Judge("POST", u, edit) == nil {
		t.Fatal("edit must be refused before the grant")
	}
	p.GrantInPlace("DOC1")
	if err := p.Judge("POST", u, edit); err != nil {
		t.Fatalf("edit must be allowed after the grant: %v", err)
	}
}

func TestEmptyPolicyRefusesEverything(t *testing.T) {
	p := NewPolicy()
	u := mustURL(t, "https://docs.googleapis.com/v1/documents/ANY")
	if p.Judge("GET", u, nil) == nil {
		t.Fatal("an empty set must refuse everything")
	}
}
```

- [ ] **Step 2: Run to verify it fails** (`go test ./internal/guard/`, FAIL)

- [ ] **Step 3: Implement**

```go
// Package guard is the network policy. Principle 3: the client reaches only
// the files it was given, and every id carries a write level. This file is
// pure judgment; transport.go carries requests through it.
package guard

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type Level int

const (
	LevelSuggest Level = 1 // handed in: read, comment, suggest. Never direct-edit.
	LevelFull    Level = 2 // created by gdoc, or explicitly granted in-place.
)

type Policy struct {
	files    map[string]Level
	createIn string // folder id a create may target; empty means no creates
}

func NewPolicy() *Policy { return &Policy{files: map[string]Level{}} }

func (p *Policy) AllowFile(id string, lvl Level) { p.files[id] = lvl }
func (p *Policy) AllowCreateIn(folderID string) {
	p.createIn = folderID
	p.files[folderID] = LevelSuggest
}
func (p *Policy) GrantInPlace(id string) {
	if _, known := p.files[id]; known {
		p.files[id] = LevelFull
	}
}
func (p *Policy) Learn(id string) { p.files[id] = LevelFull }

func refuse(format string, a ...any) error {
	return fmt.Errorf("guard refused: "+format, a...)
}

func (p *Policy) Judge(method string, u *url.URL, body []byte) error {
	if u.Scheme != "https" {
		return refuse("scheme %q", u.Scheme)
	}
	switch u.Host {
	case "oauth2.googleapis.com":
		if method == "POST" && u.Path == "/token" {
			return nil
		}
		return refuse("%s %s on the token host", method, u.Path)
	case "docs.googleapis.com":
		return p.judgeDocs(method, u, body)
	case "www.googleapis.com":
		return p.judgeDrive(method, u)
	}
	return refuse("host %q", u.Host)
}

func (p *Policy) judgeDocs(method string, u *url.URL, body []byte) error {
	rest, ok := strings.CutPrefix(u.Path, "/v1/documents/")
	if !ok || rest == "" {
		return refuse("docs path %q", u.Path)
	}
	id, verb, _ := strings.Cut(rest, ":")
	lvl, known := p.files[id]
	if !known {
		return refuse("document %q was not given to this command", id)
	}
	switch {
	case method == "GET" && verb == "":
		return nil
	case method == "POST" && verb == "batchUpdate":
		if lvl == LevelFull || isSuggestMode(body) {
			return nil
		}
		return refuse("direct edit of %q, which was handed in; only SUGGEST is allowed", id)
	}
	return refuse("%s %s", method, u.Path)
}

func (p *Policy) judgeDrive(method string, u *url.URL) error {
	path := strings.TrimPrefix(u.Path, "/upload")
	rest, ok := strings.CutPrefix(path, "/drive/v3/files")
	if !ok {
		return refuse("drive path %q", u.Path)
	}
	if rest == "" || rest == "/" { // the collection itself
		if method == "POST" && p.createIn != "" {
			return nil // create, into the one named folder; transport verifies parent
		}
		return refuse("%s on the files collection (listing and unparented creates)", method)
	}
	parts := strings.Split(strings.TrimPrefix(rest, "/"), "/")
	id := parts[0]
	lvl, known := p.files[id]
	if !known {
		return refuse("file %q was not given to this command", id)
	}
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	switch {
	case method == "GET":
		return nil // metadata, export, comments, replies: reading is level 1
	case (method == "POST" || method == "PATCH" || method == "DELETE") && (sub == "comments" || sub == "replies"):
		return nil // the comment surface is part of LevelSuggest
	case method == "PATCH" && sub == "" && lvl == LevelFull:
		return nil // e.g. trashing a document gdoc created
	}
	return refuse("%s %s at level %d", method, u.Path, lvl)
}

func isSuggestMode(body []byte) bool {
	var probe struct {
		WriteControl struct {
			WriteMode string `json:"writeMode"`
		} `json:"writeControl"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return false // a body the guard cannot read is not a suggestion
	}
	return probe.WriteControl.WriteMode == "SUGGEST"
}
```

- [ ] **Step 4: Run tests (PASS), commit**

```bash
git add go/internal/guard/
git commit -m "feat(v2): guard policy with write levels and the two doors"
```

---

### Task 4: The guard transport (the only wire)

**Files:**
- Create: `go/internal/guard/transport.go`
- Test: `go/internal/guard/transport_test.go`

**Interfaces:**
- Produces: `guard.NewClient(p *Policy, base http.RoundTripper) *http.Client`. `base == nil` means real HTTPS. The client: judges every request (body read via `GetBody`), refuses redirects to any other origin, and after a `POST` create on the files collection reads the response JSON for `"id"` and calls `p.Learn(id)`, restoring the response body. Every later milestone gets its `*http.Client` from here and nowhere else.

- [ ] **Step 1: Write the failing tests over a fake transport**

```go
package guard

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fake struct {
	status int
	body   string
	seen   []*http.Request
}

func (f *fake) RoundTrip(r *http.Request) (*http.Response, error) {
	f.seen = append(f.seen, r)
	return &http.Response{
		StatusCode: f.status,
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    r,
	}, nil
}

func TestRefusedRequestNeverReachesTheWire(t *testing.T) {
	f := &fake{status: 200, body: `{}`}
	p := NewPolicy()
	c := NewClient(p, f)
	_, err := c.Get("https://docs.googleapis.com/v1/documents/EVIL")
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("want a guard refusal, got %v", err)
	}
	if len(f.seen) != 0 {
		t.Fatal("the refused request reached the transport")
	}
}

func TestCreateTeachesThePolicy(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"NEWDOC","name":"x"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)
	resp, err := c.Post("https://www.googleapis.com/drive/v3/files",
		"application/json", bytes.NewReader([]byte(`{"parents":["FOLDER1"]}`)))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body) // body must survive the guard's peek
	if !strings.Contains(string(got), "NEWDOC") {
		t.Fatalf("response body was consumed: %q", got)
	}
	if p.files["NEWDOC"] != LevelFull {
		t.Fatal("the created id was not learned at LevelFull")
	}
}

func TestCreateOutsideTheNamedFolderIsRefused(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"X"}`}
	p := NewPolicy()
	p.AllowCreateIn("FOLDER1")
	c := NewClient(p, f)
	_, err := c.Post("https://www.googleapis.com/drive/v3/files",
		"application/json", bytes.NewReader([]byte(`{"parents":["OTHER"]}`)))
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("create aimed at a folder that was never named must be refused, got %v", err)
	}
	if len(f.seen) != 0 {
		t.Fatal("it reached the wire")
	}
}

func TestCrossOriginRedirectRefused(t *testing.T) {
	p := NewPolicy()
	p.AllowFile("DOC1", LevelSuggest)
	c := NewClient(p, &fake{status: 200, body: `{}`})
	req, _ := http.NewRequest("GET", "https://docs.googleapis.com/v1/documents/DOC1", nil)
	via := []*http.Request{req}
	next, _ := http.NewRequest("GET", "https://evil.example.com/", nil)
	if c.CheckRedirect(next, via) == nil {
		t.Fatal("a redirect off the allowed hosts must be refused")
	}
}
```

- [ ] **Step 2: Run to verify failure** (`go test ./internal/guard/`, FAIL: `NewClient` undefined)

- [ ] **Step 3: Implement**

```go
package guard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type transport struct {
	policy *Policy
	base   http.RoundTripper
}

func NewClient(p *Policy, base http.RoundTripper) *http.Client {
	if base == nil {
		base = http.DefaultTransport
	}
	return &http.Client{
		Transport: &transport{policy: p, base: base},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if err := p.Judge(req.Method, req.URL, nil); err != nil {
				return fmt.Errorf("redirect refused: %w", err)
			}
			return nil
		},
	}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := peekBody(req)
	if err != nil {
		return nil, fmt.Errorf("guard refused: unreadable request body: %w", err)
	}
	if err := t.policy.Judge(req.Method, req.URL, body); err != nil {
		return nil, err
	}
	if isCreate(req.URL, req.Method) {
		if err := t.checkParent(body); err != nil {
			return nil, err
		}
	}
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if isCreate(req.URL, req.Method) && resp.StatusCode < 300 {
		t.learnFromCreate(resp)
	}
	return resp, nil
}

func peekBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	if req.GetBody != nil {
		rc, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(b))
	return b, nil
}

func isCreate(u *url.URL, method string) bool {
	if method != "POST" || u.Host != "www.googleapis.com" {
		return false
	}
	p := strings.TrimPrefix(u.Path, "/upload")
	p = strings.TrimSuffix(p, "/")
	return p == "/drive/v3/files"
}

func (t *transport) checkParent(body []byte) error {
	// A create must name exactly the folder this command was given. For
	// multipart uploads the metadata part is the first JSON object; M6 sets
	// GetBody so the peek sees it. A create whose parents the guard cannot
	// read is refused: not knowing never resolves to carrying it.
	var meta struct {
		Parents []string `json:"parents"`
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&meta); err != nil || len(meta.Parents) != 1 || meta.Parents[0] != t.policy.createIn {
		return fmt.Errorf("guard refused: create must name exactly the folder %q", t.policy.createIn)
	}
	return nil
}

func (t *transport) learnFromCreate(resp *http.Response) {
	b, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(b))
	if err != nil {
		return
	}
	var created struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(b, &created) == nil && created.ID != "" {
		t.policy.Learn(created.ID)
	}
}
```

- [ ] **Step 4: Run tests (PASS), commit**

```bash
git add go/internal/guard/
git commit -m "feat(v2): guard transport: judge, parent check, learn from create, redirect refusal"
```

---

### Task 5: The import-boundary test

**Files:**
- Create: `go/boundary/boundary_test.go`

**Interfaces:**
- Produces: a test that fails the build when `net/http` is imported anywhere in the module outside the allowlist `{internal/guard, internal/auth/loopback}`. It is an allowlist, not a ban, exactly like v1's `test_guard_is_installed`: it fails if the import spreads, and it fails just as loudly if an allowlisted package stops importing it without this test being updated.

- [ ] **Step 1: Write the test (it passes now and starts failing the moment anyone strays; also add a canary)**

```go
// Package boundary enforces that gdoc has exactly one wire. Only the guard
// (and the login loopback listener, which serves and never dials) may import
// net/http. This is the v1 allowlist test, ported: it fails in BOTH
// directions, on spread and on silent disappearance.
package boundary

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

var allowed = map[string]bool{
	"internal/guard":         true,
	"internal/auth/loopback": true,
}

func TestNetHTTPStaysInItsTwoRooms(t *testing.T) {
	moduleRoot := ".."
	found := map[string]bool{}
	err := filepath.WalkDir(moduleRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		f, perr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if perr != nil {
			return perr
		}
		for _, imp := range f.Imports {
			if strings.Trim(imp.Path.Value, `"`) != "net/http" {
				continue
			}
			rel, _ := filepath.Rel(moduleRoot, filepath.Dir(path))
			rel = filepath.ToSlash(rel)
			if rel == "boundary" {
				continue // this package's own canary below
			}
			if !allowed[rel] {
				t.Errorf("%s imports net/http; only %v may", path, keys())
			}
			found[rel] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for pkg := range allowed {
		if !found[pkg] {
			t.Errorf("allowlisted package %s no longer imports net/http; update the allowlist deliberately", pkg)
		}
	}
}

func keys() []string {
	var k []string
	for a := range allowed {
		k = append(k, a)
	}
	return k
}
```

- [ ] **Step 2: Run it** (`go test ./boundary/`). Expected now: FAIL on the second half, because `internal/auth/loopback` does not exist yet. That is the allowlist working. Temporarily it documents Task 7's obligation; leave it failing only if Task 7 lands in the same session, otherwise trim the allowlist to `internal/guard` and expand it in Task 7. Choose the trim: the test must be green at every commit.

- [ ] **Step 3: Commit**

```bash
git add go/boundary/
git commit -m "test(v2): net/http import allowlist, both directions"
```

---

### Task 6: Auth: token load, refresh, crash-safe write, `auth status`, CLI skeleton

**Files:**
- Create: `go/internal/auth/auth.go`
- Create: `go/cmd/gdoc/main.go`
- Test: `go/internal/auth/auth_test.go`

**Interfaces:**
- Consumes: `config.TokenPath()`, `guard.NewClient` (a bare policy: the token endpoint needs no file ids).
- Produces:
  - `auth.Token{AccessToken, RefreshToken, ClientID, ClientSecret, TokenURI string, Scopes []string, Expiry time.Time}` reading v1's `oauth-token.json` (google-auth "authorized user" JSON: fields `token`, `refresh_token`, `token_uri`, `client_id`, `client_secret`, `scopes`, `expiry`).
  - `auth.Load() (Token, error)`; `(Token).Refresh(c *http.Client) (Token, error)` posting `grant_type=refresh_token` form data to `TokenURI`, keeping the old refresh token when the response omits one; `auth.Save(t Token) error` via temp file in the same directory plus `os.Rename` (crash-safe; Windows rename semantics are documented here and smoke-tested in M9).
  - `auth.Status() map[string]any` for the command: `{token_present, token_path, client_source, scopes, expired}`.
  - `cmd/gdoc`: dispatch `gdoc auth status`; unknown commands emit a failing envelope naming the command.
- Constants: copy `BUNDLED_CLIENT_ID` and `BUNDLED_CLIENT_SECRET` values verbatim from `gdoc/oauth.py:51-52` into `auth.go`, with v1's comment about RFC 8252 carried over. `client_source` is `"file"` when `oauth-client.json` exists in the config dir, else `"bundled"`.

- [ ] **Step 1: Write the failing tests** (temp config dir via `t.Setenv("GDOC_CONFIG_DIR", t.TempDir())`; write a fixture token JSON; assert Load round-trips; assert Refresh posts the right form and keeps the old refresh token when the response has none, over a fake `RoundTripper`; assert Save then Load is identical and that a failed Save leaves the original file intact)

```go
package auth

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixture = `{"token":"OLD","refresh_token":"R1","token_uri":"https://oauth2.googleapis.com/token","client_id":"CID","client_secret":"CS","scopes":["https://www.googleapis.com/auth/drive"],"expiry":"2020-01-01T00:00:00Z"}`

func writeFixture(t *testing.T) string {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	path := filepath.Join(dir, "oauth-token.json")
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadReadsV1Format(t *testing.T) {
	writeFixture(t)
	tok, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if tok.RefreshToken != "R1" || tok.ClientID != "CID" {
		t.Fatalf("bad load: %+v", tok)
	}
	if !tok.Expired() {
		t.Fatal("a 2020 expiry is expired")
	}
}

type fakeRT struct{ body string }

func (f *fakeRT) RoundTrip(r *http.Request) (*http.Response, error) {
	b, _ := io.ReadAll(r.Body)
	if !strings.Contains(string(b), "grant_type=refresh_token") ||
		!strings.Contains(string(b), "refresh_token=R1") {
		return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(`{"error":"bad form"}`)), Request: r}, nil
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(f.body)), Request: r}, nil
}

func TestRefreshKeepsOldRefreshToken(t *testing.T) {
	writeFixture(t)
	tok, _ := Load()
	c := &http.Client{Transport: &fakeRT{body: `{"access_token":"NEW","expires_in":3600}`}}
	got, err := tok.Refresh(c)
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "NEW" || got.RefreshToken != "R1" {
		t.Fatalf("refresh must keep the old refresh token: %+v", got)
	}
	if got.Expired() {
		t.Fatal("a fresh token is not expired")
	}
}

func TestSaveIsCrashSafe(t *testing.T) {
	path := writeFixture(t)
	tok, _ := Load()
	tok.AccessToken = "SAVED"
	if err := Save(tok); err != nil {
		t.Fatal(err)
	}
	again, _ := Load()
	if again.AccessToken != "SAVED" {
		t.Fatal("save did not persist")
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatalf("temp files left behind: %v", entries)
	}
}
```

- [ ] **Step 2: Run to verify failure**, then **Step 3: implement** `auth.go`:

```go
// Package auth holds the OAuth token and the bundled client. The secret is
// deliberately in version control; see CLAUDE.md and RFC 8252 section 8.5.
// Refresh is one form POST, which is why x/oauth2 is not a dependency.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gdoc/internal/config"
)

const (
	BundledClientID     = "" // copy verbatim from gdoc/oauth.py:51
	BundledClientSecret = "" // copy verbatim from gdoc/oauth.py:52
	TokenURI            = "https://oauth2.googleapis.com/token"
)

type Token struct {
	AccessToken  string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	TokenURI     string    `json:"token_uri"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	Scopes       []string  `json:"scopes"`
	Expiry       time.Time `json:"expiry"`
}

func (t Token) Expired() bool {
	// A minute of skew: an almost-expired token is treated as expired so a
	// long-running upload never crosses the line mid-flight.
	return !t.Expiry.After(time.Now().Add(time.Minute))
}

func Load() (Token, error) {
	path, err := config.TokenPath()
	if err != nil {
		return Token{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return Token{}, fmt.Errorf("no OAuth token at %s. Run: gdoc auth login", path)
	}
	var t Token
	if err := json.Unmarshal(b, &t); err != nil {
		return Token{}, fmt.Errorf("the token at %s cannot be parsed: %w", path, err)
	}
	return t, nil
}

func (t Token) Refresh(c *http.Client) (Token, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {t.RefreshToken},
		"client_id":     {t.ClientID},
		"client_secret": {t.ClientSecret},
	}
	resp, err := c.Post(t.TokenURI, "application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != 200 {
		return Token{}, fmt.Errorf("token refresh failed (%d): %s", resp.StatusCode, summarize(body))
	}
	var r struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.AccessToken == "" {
		return Token{}, errors.New("token refresh returned 200 with no access token")
	}
	out := t
	out.AccessToken = r.AccessToken
	if r.RefreshToken != "" {
		out.RefreshToken = r.RefreshToken
	}
	out.Expiry = time.Now().UTC().Add(time.Duration(r.ExpiresIn) * time.Second)
	return out, nil
}

func summarize(body []byte) string {
	var e struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error != "" {
		return e.Error
	}
	return "unreadable error body"
}

func Save(t Token) error {
	path, err := config.TokenPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".oauth-token-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	// Same-directory rename. Not guaranteed atomic on Windows; documented in
	// the spec and covered by the M9 Windows smoke test.
	return os.Rename(tmpName, path)
}
```

And `cmd/gdoc/main.go`:

```go
// gdoc v2. Facts in, JSON out, exit. The skill does the talking.
package main

import (
	"fmt"
	"os"

	"gdoc/internal/auth"
	"gdoc/internal/config"
	"gdoc/internal/emit"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) >= 2 && args[0] == "auth" && args[1] == "status" {
		return finish(authStatus())
	}
	return finish(emit.Result{OK: false,
		Error: fmt.Sprintf("unknown command %q. Commands: auth status", joined(args))})
}

func finish(r emit.Result) int {
	if err := emit.Print(os.Stdout, r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return emit.ExitCode(r)
}

func joined(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func authStatus() emit.Result {
	path, err := config.TokenPath()
	if err != nil {
		return emit.Result{OK: false, Error: err.Error()}
	}
	tok, err := auth.Load()
	data := map[string]any{"auth_mode": "oauth", "token_path": path}
	if err != nil {
		data["token_present"] = false
		return emit.Result{OK: true, Data: data}
	}
	data["token_present"] = true
	data["expired"] = tok.Expired()
	data["scopes"] = tok.Scopes
	return emit.Result{OK: true, Data: data}
}
```

- [ ] **Step 4: Fill in the two constants from `gdoc/oauth.py:51-52`, run all tests** (`cd go && go test ./...`), and try it for real: `cd go && go run ./cmd/gdoc auth status` should print one JSON object reporting your actual v1 token as present.

- [ ] **Step 5: Commit**

```bash
git add go/internal/auth/ go/cmd/
git commit -m "feat(v2): token load and refresh on stdlib, auth status, CLI skeleton"
```

---

### Task 7: `auth login` (loopback + PKCE) and the build matrix

**Files:**
- Create: `go/internal/auth/loopback/loopback.go`
- Create: `go/internal/auth/login.go`
- Modify: `go/cmd/gdoc/main.go` (add the `auth login` arm)
- Modify: `go/boundary/boundary_test.go` (allowlist gains `internal/auth/loopback`, from Task 5's trim)
- Create: `Makefile` (repo root)
- Test: `go/internal/auth/login_test.go`

**Interfaces:**
- Consumes: `auth.Save`, `guard.NewClient` (token exchange POST), `emit`.
- Produces:
  - `loopback.Listen() (*loopback.Server, error)`: binds `127.0.0.1:0`, exposes `Addr() string`, `WaitCode(state string, timeout time.Duration) (code string, err error)`, `Close()`. Serves one request, checks `state`, answers a plain "You can close this tab." page. This package serves and never dials; that is why it may import `net/http`.
  - `auth.Login(c *http.Client, w io.Writer) error`: builds the authorization URL (endpoint `https://accounts.google.com/o/oauth2/auth`, `code_challenge_method=S256`, verifier from `crypto/rand`, scopes from Global Constraints, `redirect_uri` = the loopback address), **prints it to `w` (stderr)** with one plain sentence, waits for the code, exchanges it at `TokenURI` (form fields `grant_type=authorization_code`, `code`, `code_verifier`, `client_id`, `client_secret`, `redirect_uri`), saves via `auth.Save`. No browser is opened: v2 runs no external programs at all.
- Note for the implementer: `accounts.google.com` is a page the human opens in their own browser; gdoc itself never requests it, so the guard's host allowlist does not change.

- [ ] **Step 1: Write the failing test for the URL and the exchange** (fake RoundTripper asserts the exchange form carries `grant_type=authorization_code`, `code_verifier` matching the challenge in the printed URL, and the fixture code; assert the printed URL carries `code_challenge_method=S256`, the two scopes, and the loopback redirect)

```go
package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type exchangeRT struct{ seenVerifier, seenCode string }

func (e *exchangeRT) RoundTrip(r *http.Request) (*http.Response, error) {
	b, _ := io.ReadAll(r.Body)
	form, _ := url.ParseQuery(string(b))
	e.seenVerifier = form.Get("code_verifier")
	e.seenCode = form.Get("code")
	return &http.Response{StatusCode: 200, Request: r, Body: io.NopCloser(strings.NewReader(
		`{"access_token":"A","refresh_token":"R","expires_in":3600}`))}, nil
}

func TestAuthURLCarriesPKCE(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	var sb strings.Builder
	verifier, authURL := buildAuthURL("http://127.0.0.1:9999/cb", "STATE1")
	sb.WriteString(authURL)
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("code_challenge_method") != "S256" {
		t.Fatal("PKCE method missing")
	}
	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if q.Get("code_challenge") != want {
		t.Fatal("challenge does not match verifier")
	}
	if !strings.Contains(q.Get("scope"), "auth/drive") || !strings.Contains(q.Get("scope"), "auth/documents") {
		t.Fatalf("scopes: %q", q.Get("scope"))
	}
}

func TestExchangeSendsVerifierAndSaves(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	rt := &exchangeRT{}
	c := &http.Client{Transport: rt}
	tok, err := exchangeCode(c, "CODE7", "VERIF7", "http://127.0.0.1:9999/cb")
	if err != nil {
		t.Fatal(err)
	}
	if rt.seenCode != "CODE7" || rt.seenVerifier != "VERIF7" {
		t.Fatalf("exchange form wrong: %+v", rt)
	}
	if tok.RefreshToken != "R" {
		t.Fatalf("token: %+v", tok)
	}
}
```

- [ ] **Step 2: Run to verify failure, then implement.** `login.go`:

```go
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gdoc/internal/auth/loopback"
)

const authEndpoint = "https://accounts.google.com/o/oauth2/auth"

var loginScopes = []string{
	"https://www.googleapis.com/auth/drive",
	"https://www.googleapis.com/auth/documents",
}

func randomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func buildAuthURL(redirect, state string) (verifier, authURL string) {
	verifier = randomToken()
	sum := sha256.Sum256([]byte(verifier))
	q := url.Values{
		"client_id":             {BundledClientID},
		"redirect_uri":          {redirect},
		"response_type":         {"code"},
		"scope":                 {strings.Join(loginScopes, " ")},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
		"access_type":           {"offline"},
		"prompt":                {"consent"},
	}
	return verifier, authEndpoint + "?" + q.Encode()
}

func exchangeCode(c *http.Client, code, verifier, redirect string) (Token, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"client_id":     {BundledClientID},
		"client_secret": {BundledClientSecret},
		"redirect_uri":  {redirect},
	}
	resp, err := c.Post(TokenURI, "application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != 200 {
		return Token{}, fmt.Errorf("code exchange failed (%d): %s", resp.StatusCode, summarize(body))
	}
	var r struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := unmarshalToken(body, &r); err != nil {
		return Token{}, err
	}
	return Token{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		TokenURI:     TokenURI,
		ClientID:     BundledClientID,
		ClientSecret: BundledClientSecret,
		Scopes:       loginScopes,
		Expiry:       time.Now().UTC().Add(time.Duration(r.ExpiresIn) * time.Second),
	}, nil
}

// Login prints the authorization URL to w (stderr: stdout is reserved for the
// one JSON object) and waits. No browser is opened: v2 runs no external
// programs at all.
func Login(c *http.Client, w io.Writer) error {
	srv, err := loopback.Listen()
	if err != nil {
		return err
	}
	defer srv.Close()
	state := randomToken()
	redirect := "http://" + srv.Addr() + "/callback"
	verifier, authURL := buildAuthURL(redirect, state)
	fmt.Fprintf(w, "Open this link in your browser to sign in:\n%s\n", authURL)
	code, err := srv.WaitCode(state, 3*time.Minute)
	if err != nil {
		return err
	}
	tok, err := exchangeCode(c, code, verifier, redirect)
	if err != nil {
		return err
	}
	return Save(tok)
}
```

Add the tiny helper to `auth.go` so both exchanges share it:

```go
func unmarshalToken(body []byte, r any) error {
	if err := json.Unmarshal(body, r); err != nil {
		return errors.New("token endpoint returned 200 with an unreadable body")
	}
	return nil
}
```

And `loopback/loopback.go`, the one other room where `net/http` may live (it serves, never dials):

```go
// Package loopback is the one-shot localhost listener the login flow parks a
// browser redirect on. It serves exactly one callback and never makes a
// request, which is why the boundary test allowlists it.
package loopback

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Server struct {
	ln    net.Listener
	codes chan result
	srv   *http.Server
}

type result struct {
	code, state string
}

func Listen() (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{ln: ln, codes: make(chan result, 1)}
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		select {
		case s.codes <- result{code: q.Get("code"), state: q.Get("state")}:
		default: // a second hit changes nothing
		}
		fmt.Fprint(w, "Signed in. You can close this tab.")
	})
	s.srv = &http.Server{Handler: mux}
	go s.srv.Serve(ln)
	return s, nil
}

func (s *Server) Addr() string { return s.ln.Addr().String() }

func (s *Server) WaitCode(state string, timeout time.Duration) (string, error) {
	select {
	case r := <-s.codes:
		if r.state != state {
			return "", errors.New("login callback carried the wrong state; refusing the code")
		}
		if r.code == "" {
			return "", errors.New("login callback carried no code")
		}
		return r.code, nil
	case <-time.After(timeout):
		return "", errors.New("no login callback arrived within the timeout")
	}
}

func (s *Server) Close() { s.srv.Close() }
```

- [ ] **Step 3: Restore the boundary allowlist to its two rooms** (Task 5's trim is reverted: `internal/auth/loopback` is back). Run `go test ./...`: everything green, boundary test included.

- [ ] **Step 4: Wire `auth login` into `cmd/gdoc` (URL to stderr, one JSON object to stdout at the end), and write the Makefile**

```makefile
GO := cd go && go

test:
	$(GO) test ./...

build:
	$(GO) build -o ../bin/gdoc ./cmd/gdoc

dist:
	cd go && CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o ../bin/gdoc-darwin-arm64 ./cmd/gdoc
	cd go && CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -o ../bin/gdoc-darwin-amd64 ./cmd/gdoc
	cd go && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ../bin/gdoc-windows-amd64.exe ./cmd/gdoc

.PHONY: test build dist
```

- [ ] **Step 5: Run the acceptance checks**

Run: `make test && make dist && ls bin/`
Expected: all tests pass; three binaries exist. Then the live check (your machine, real config): `./bin/gdoc-darwin-arm64 auth status` prints one JSON object with `token_present: true`.

- [ ] **Step 6: Commit**

```bash
git add go/ Makefile
git commit -m "feat(v2): auth login with loopback and PKCE, cross-build matrix"
```

---

### Task 8: Verify acceptance criteria

**Files:**
- Modify: none expected. Fix whatever the checks below break.

**Interfaces:**
- Consumes: everything Tasks 1 to 7 produced. Produces: proof that the milestone acceptance in `docs/v2/SPEC.md` item 1 holds.

- [ ] verify every requirement in the Overview is implemented: the binary exists, prints the envelope, owns the network through the guard, and can log in, refresh and report OAuth state
- [ ] verify the guard refuses, by running the four cases in one go and reading the output: a request for an id outside the set never reaches the transport; a direct edit on a handed-in id is refused; a create aimed at an unnamed folder is refused; the boundary test holds in both directions
- [ ] run the full test suite: `cd go && go test ./...`
- [ ] run the formatter check: `cd go && gofmt -l . && test -z "$(gofmt -l .)"`
- [ ] run the vet check: `cd go && go vet ./...`
- [ ] verify coverage: every exported function under `go/internal/` has a test. `cd go && go test ./... -cover` and read the per-package numbers; add tests for anything uncovered rather than lowering the bar
- [ ] verify `make dist` produces the three platform binaries with `CGO_ENABLED=0`, and that `file bin/*` reports the expected architectures
- [ ] commit any fixes this task made

### Task 9: [Final] Update documentation

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `docs/v2/PLAN.md`

**Interfaces:**
- Produces: the repo's own account of what now exists in Go, so the next milestone starts from documentation that matches the tree.

- [ ] update `README.md`: say that `go/` exists, what `gdoc auth status` and `gdoc auth login` do, and how to build with `make build` and `make dist`
- [ ] update `CLAUDE.md` with the patterns this milestone established: the guard owns all outbound HTTP and is built before any client, `net/http` is allowlisted by the boundary test to `internal/guard` and `internal/auth/loopback`, and every command prints through `internal/emit`
- [ ] mark Milestone 1 done in `docs/v2/PLAN.md`
- [ ] run the full test suite one more time: `cd go && go test ./...`
- [ ] commit the documentation updates
- [ ] move this plan to `docs/plans/completed/`

## Post-Completion

*Items needing a real account, a browser, or another machine. No checkboxes: these are informational.*

**Manual verification:**

- `gdoc auth status` on the real machine reports the v1 token, and has modified nothing. Compare `oauth-token.json` before and after by checksum.
- `gdoc auth login` end to end: the URL goes to stderr, the browser trip completes on the loopback listener, and the token that lands still works for a following `auth status`.
- The windows and amd64 binaries from `make dist` actually start on those platforms. Cross-compiling proves they link, not that they run.

**External system updates:**

- None. This milestone adds a second binary beside v1 and changes nothing that v1 or the skills read.
