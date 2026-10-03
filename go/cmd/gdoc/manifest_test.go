package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// The Claude Desktop extension's manifest template, judged from here.
//
// release/mcpb/manifest.json is the one file in this tree that a program
// outside it reads: Claude Desktop installs the extension and starts the
// command the manifest names. Nothing in the binary reads the template, so
// without these tests a field renamed in the manifest is found by a colleague
// whose connector failed with no reason.
//
// The template is judged rather than rendered: both installers and
// `gdoc update --desktop` fill @BIN@ and @VERSION@, and release/test-desktop.sh
// is what watches the filling, because a Go test cannot run a shell script.

// manifestTemplate is the template's path from this package.
const manifestTemplate = "../../../release/mcpb/manifest.json"

// mcpbManifest is the manifest as this file reads it. Only the fields these
// tests judge are here: a field Claude Desktop reads and gdoc does not is still
// the manifest's own business.
type mcpbManifest struct {
	ManifestVersion string `json:"manifest_version"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	Description     string `json:"description"`
	Author          struct {
		Name string `json:"name"`
	} `json:"author"`
	Server struct {
		Type       string `json:"type"`
		EntryPoint string `json:"entry_point"`
		MCPConfig  struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcp_config"`
	} `json:"server"`
	Tools []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"tools"`
	Compatibility struct {
		Platforms []string `json:"platforms"`
	} `json:"compatibility"`
	UserConfig map[string]struct {
		Type        string  `json:"type"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Required    *bool   `json:"required"`
		Default     *string `json:"default"`
	} `json:"user_config"`
}

// readManifest is the template as bytes and as the shape above.
func readManifest(t *testing.T) (string, mcpbManifest) {
	t.Helper()
	b, err := os.ReadFile(manifestTemplate)
	if err != nil {
		t.Fatal(err)
	}
	var m mcpbManifest
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("%s is not JSON Claude Desktop could read: %v", manifestTemplate, err)
	}
	return string(b), m
}

// TestTheManifestTemplateParses holds the shape of the file and its two
// placeholders. @BIN@ is twice over, because Claude Desktop reads the command
// out of mcp_config and the entry point out of server, and a manifest naming
// the binary in one place and a placeholder in the other starts nothing.
// @VERSION@ is once, in the version field.
//
// The values are literals here, never read from the code that fills them: a
// test that read the filler would follow it wherever somebody moved it.
func TestTheManifestTemplateParses(t *testing.T) {
	raw, m := readManifest(t)

	if n := strings.Count(raw, "@BIN@"); n != 2 {
		t.Errorf("%s names @BIN@ %d times, want 2: the entry point and the command", manifestTemplate, n)
	}
	if n := strings.Count(raw, "@VERSION@"); n != 1 {
		t.Errorf("%s names @VERSION@ %d times, want 1: the version field", manifestTemplate, n)
	}

	if m.ManifestVersion != "0.3" {
		t.Errorf("manifest_version is %q, want %q", m.ManifestVersion, "0.3")
	}
	if m.Name != "gdoc" {
		t.Errorf("name is %q, want %q", m.Name, "gdoc")
	}
	if m.Version != "@VERSION@" {
		t.Errorf("version is %q, want the placeholder %q", m.Version, "@VERSION@")
	}
	// A page with no sentence on it is an extension a person installs without
	// knowing what it reaches, and an author is who they ask about it.
	if m.Description == "" {
		t.Error("the manifest carries no description, and that sentence is what a person reads before they install it")
	}
	if m.Author.Name == "" {
		t.Error("the manifest names no author")
	}
	if m.Server.Type != "binary" {
		t.Errorf("server.type is %q, want %q", m.Server.Type, "binary")
	}
	if m.Server.EntryPoint != "@BIN@" {
		t.Errorf("server.entry_point is %q, want the placeholder %q", m.Server.EntryPoint, "@BIN@")
	}
	if m.Server.MCPConfig.Command != "@BIN@" {
		t.Errorf("server.mcp_config.command is %q, want the placeholder %q", m.Server.MCPConfig.Command, "@BIN@")
	}
	// The word mcp and the one flag, in that order. The flag is joined to its
	// value, because Claude Desktop fills the field whether the person typed
	// anything in it or not and an empty value has to reach the parser as one
	// argument.
	want := []string{"mcp", "--trusted-email-domains=${user_config.trusted_email_domains}"}
	if strings.Join(m.Server.MCPConfig.Args, " ") != strings.Join(want, " ") {
		t.Errorf("server.mcp_config.args is %v, want %v", m.Server.MCPConfig.Args, want)
	}
	if strings.Join(m.Compatibility.Platforms, " ") != "darwin" {
		t.Errorf("compatibility.platforms is %v, want [darwin]: the extension is written and opened on macOS alone", m.Compatibility.Platforms)
	}
}

// TestTheManifestListsTheToolsToolsListLists is the drift this file exists for.
// The manifest's list is what Claude Desktop shows on the extension's page, and
// tools/list is what the session answers with. A tool in one and not the other
// is a page that lies about what the connector does.
//
// The descriptions are compared whole, so the words on the page are the words
// the model is given.
func TestTheManifestListsTheToolsToolsListLists(t *testing.T) {
	_, m := readManifest(t)
	tools := mcpTools(io.Discard, newMCPLogin(io.Discard), callChat(t))
	if len(tools) == 0 {
		t.Fatal("the session offers no tools")
	}
	if len(m.Tools) != len(tools) {
		t.Fatalf("the manifest lists %d tools and the session offers %d", len(m.Tools), len(tools))
	}
	for i, tool := range tools {
		listed := m.Tools[i]
		if listed.Name != tool.Name {
			t.Errorf("the manifest's tool %d is %q and the session's is %q; the order is the session's", i, listed.Name, tool.Name)
			continue
		}
		if listed.Description != tool.Description {
			t.Errorf("the manifest describes %s as\n  %q\nand the session as\n  %q", tool.Name, listed.Description, tool.Description)
		}
	}
}

// TestTheManifestHasOneOptionalSettingWithAnEmptyDefault holds the one field
// the extension shows. It is not required and its default is empty, so a person
// installs the extension by pressing install and never reads it: decision 17.
// A required field would make every colleague decide something about addresses
// before they have reviewed a document.
func TestTheManifestHasOneOptionalSettingWithAnEmptyDefault(t *testing.T) {
	_, m := readManifest(t)
	if len(m.UserConfig) != 1 {
		t.Fatalf("the manifest shows %d settings, want 1", len(m.UserConfig))
	}
	field, ok := m.UserConfig["trusted_email_domains"]
	if !ok {
		t.Fatalf("the manifest's one setting is not trusted_email_domains: %v", m.UserConfig)
	}
	if field.Type != "string" {
		t.Errorf("the setting's type is %q, want %q", field.Type, "string")
	}
	if field.Required == nil || *field.Required {
		t.Error("the setting is required or says nothing about it; it is optional, and said so out loud")
	}
	if field.Default == nil || *field.Default != "" {
		t.Errorf("the setting's default is %v, want the empty string written out", field.Default)
	}
	if field.Title != "Email domains that need no approval" {
		t.Errorf("the setting's title is %q", field.Title)
	}
	if field.Description != "Advanced and optional. Leave empty." {
		t.Errorf("the setting's description is %q", field.Description)
	}
}
