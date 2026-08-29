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
