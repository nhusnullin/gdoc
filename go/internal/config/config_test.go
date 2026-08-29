package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnvOverrideWins(t *testing.T) {
	// t.TempDir, not a literal path: M9 runs this suite on Windows, where a
	// path spelled for POSIX is not a path at all.
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	d, err := Dir()
	if err != nil || d != dir {
		t.Fatalf("got %q, %v", d, err)
	}
	p, _ := TokenPath()
	if p != filepath.Join(dir, "oauth-token.json") {
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
