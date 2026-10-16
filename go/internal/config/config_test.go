package config

import (
	"errors"
	"os"
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
	c, _ := LastCheckPath()
	if c != filepath.Join(dir, "update-check.json") {
		t.Fatalf("last check path: %q", c)
	}
	l, _ := LoginPendingPath()
	if l != filepath.Join(dir, "login-pending.json") {
		t.Fatalf("login pending path: %q", l)
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
	default: // darwin and everything else keep ~/.config/gdoc-agent
		if filepath.Base(filepath.Dir(d)) != ".config" || filepath.Base(d) != "gdoc-agent" {
			t.Fatalf("unix dir: %q", d)
		}
	}
}

// The Windows branch decides where the OAuth token is written on a machine
// nobody here runs the suite on, so it is checked as a decision rather than as
// a path. filepath.Join spells the separator of the platform this test is
// compiled for, which is not the one being described, so the assertion is on
// the parts and never on a literal path.
func TestDirForEachCase(t *testing.T) {
	homeGone := errors.New("$HOME is not defined")
	for _, c := range []struct {
		name string
		in   outside
		want string
		err  bool
	}{
		{
			name: "the override wins on unix",
			in:   outside{goos: "darwin", configDir: "/tmp/elsewhere", home: "/Users/nail"},
			want: "/tmp/elsewhere",
		},
		{
			name: "the override wins on windows too, before AppData is looked at",
			in:   outside{goos: "windows", configDir: "/tmp/elsewhere"},
			want: "/tmp/elsewhere",
		},
		{
			name: "windows puts it under AppData",
			in:   outside{goos: "windows", appData: filepath.Join("C:", "Users", "nail", "AppData", "Roaming")},
			want: filepath.Join("C:", "Users", "nail", "AppData", "Roaming", "gdoc-agent"),
		},
		{
			name: "windows without AppData is a failure, never a guess",
			in:   outside{goos: "windows"},
			err:  true,
		},
		{
			name: "unix keeps ~/.config/gdoc-agent",
			in:   outside{goos: "darwin", home: filepath.Join("/Users", "nail")},
			want: filepath.Join("/Users", "nail", ".config", "gdoc-agent"),
		},
		{
			name: "a home the OS cannot name is a failure, never a guess",
			in:   outside{goos: "linux", homeErr: homeGone},
			err:  true,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := dirFor(c.in)
			if c.err {
				if err == nil {
					t.Fatalf("dirFor(%+v) = %q, want an error", c.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("dirFor(%+v): %v", c.in, err)
			}
			if got != c.want {
				t.Errorf("dirFor(%+v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// The error the home lookup gave is the error the caller sees. A message this
// package wrote over it would hide which of the two lookups failed.
func TestDirForCarriesTheHomeErrorItself(t *testing.T) {
	homeGone := errors.New("$HOME is not defined")
	_, err := dirFor(outside{goos: "darwin", homeErr: homeGone})
	if !errors.Is(err, homeGone) {
		t.Errorf("dirFor() error = %v, want the one the home lookup gave", err)
	}
}

// TokenPath fails when Dir does. A token path built over an empty directory
// would be a relative path in whatever directory the command happened to run
// in, which is a token file written where nobody looks for it.
func TestTokenPathFailsWhenTheDirectoryCannotBeNamed(t *testing.T) {
	t.Setenv("GDOC_CONFIG_DIR", "")
	if runtime.GOOS == "windows" {
		t.Setenv("AppData", "")
	} else {
		t.Setenv("HOME", "")
	}
	p, err := TokenPath()
	if err == nil {
		t.Fatalf("TokenPath() = %q, want the failure Dir gave", p)
	}
	if p != "" {
		t.Errorf("TokenPath() = %q on failure, want the empty string", p)
	}
}

// The settings file sits beside the token, because both are per-user state this
// machine keeps. It is never a credential, and internal/settings decides what
// may be in it.
func TestSettingsPathIsInTheConfigFolder(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	p, err := SettingsPath()
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join(dir, "settings.yaml") {
		t.Errorf("SettingsPath() = %q, want settings.yaml in the config folder", p)
	}
}

func TestPresenceDirIsInTheConfigFolder(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GDOC_CONFIG_DIR", dir)
	p, err := PresenceDir()
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join(dir, "presence") {
		t.Errorf("PresenceDir() = %q, want presence in the config folder", p)
	}
	// Naming the directory creates nothing: the first event is what makes it,
	// and a command with the flag off names it never.
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("PresenceDir() made the directory: %v", err)
	}
}

func TestPresenceDirFailsWhenTheDirectoryCannotBeNamed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("%AppData% is the Windows branch and TestDirForEachCase holds it")
	}
	t.Setenv("GDOC_CONFIG_DIR", "")
	t.Setenv("HOME", "")
	if _, err := PresenceDir(); err == nil {
		t.Error("PresenceDir() guessed a path with no home to put it under")
	}
}
