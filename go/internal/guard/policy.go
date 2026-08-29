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
