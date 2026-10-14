package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// write puts a settings file in a directory of this test's own and hands back
// its path. Every test names its own file, so no test can see another's.
func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "settings.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// gone is a path in a directory that exists with no file at it, which is what
// a default settings file looks like on a machine nobody has configured.
func gone(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "settings.yaml")
}

// No argument and no file is the four-dot mark, from the default, and no file
// was read. The mark is stated as a literal, never read back from the constant
// this package asks for.
func TestNoFileIsEveryDefault(t *testing.T) {
	in, err := Resolve(Given{}, gone(t))
	if err != nil {
		t.Fatal(err)
	}
	if in.Mark != "⁘" {
		t.Errorf("Mark = %q, want the four-dot mark", in.Mark)
	}
	if in.MarkFrom != "default" {
		t.Errorf("MarkFrom = %q, want %q", in.MarkFrom, "default")
	}
	if in.File != "" {
		t.Errorf("File = %q, want the empty string when no file was read", in.File)
	}
}

func TestTheFileSetsTheMark(t *testing.T) {
	path := write(t, "mark: \"∴\"\n")
	in, err := Resolve(Given{}, path)
	if err != nil {
		t.Fatal(err)
	}
	if in.Mark != "∴" {
		t.Errorf("Mark = %q, want the three-dot mark the file names", in.Mark)
	}
	if in.MarkFrom != "file" {
		t.Errorf("MarkFrom = %q, want %q", in.MarkFrom, "file")
	}
	if in.File != path {
		t.Errorf("File = %q, want %q", in.File, path)
	}
}

// A zero-byte file is every default, and it is still the file in use.
func TestAFileOfNoBytesIsEveryDefault(t *testing.T) {
	path := write(t, "")
	in, err := Resolve(Given{}, path)
	if err != nil {
		t.Fatal(err)
	}
	if in.Mark != "⁘" || in.MarkFrom != "default" {
		t.Errorf("Resolve() = %q from %q, want the four-dot mark from the default", in.Mark, in.MarkFrom)
	}
	if in.File != path {
		t.Errorf("File = %q, want the file that was read", in.File)
	}
}

func TestASettingArgumentOverridesTheFile(t *testing.T) {
	path := write(t, "mark: \"∴\"\n")
	mark := "🤖"
	in, err := Resolve(Given{Config: &path, Mark: &mark}, gone(t))
	if err != nil {
		t.Fatal(err)
	}
	if in.Mark != "🤖" {
		t.Errorf("Mark = %q, want the robot the argument named", in.Mark)
	}
	if in.MarkFrom != "argument" {
		t.Errorf("MarkFrom = %q, want %q", in.MarkFrom, "argument")
	}
	if in.File != path {
		t.Errorf("File = %q, want the file that was read even though the argument won", in.File)
	}
}

// A named file that is not there is refused. Treating it as empty would hand a
// colleague who mistyped the path every default and say nothing.
func TestAConfigFileThatDoesNotExistIsRefused(t *testing.T) {
	path := gone(t)
	_, err := Resolve(Given{Config: &path}, gone(t))
	if err == nil {
		t.Fatal("Resolve() with a --config file that is not there, want a refusal")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("refusal %q, want the path it looked for", err)
	}
	if !strings.Contains(err.Error(), "--config") {
		t.Errorf("refusal %q, want the flag that named it", err)
	}
}

func TestAnUnknownKeyIsRefusedByName(t *testing.T) {
	path := write(t, "colour: red\n")
	_, err := Resolve(Given{}, path)
	if err == nil {
		t.Fatal("Resolve() with an unknown key, want a refusal")
	}
	if !strings.Contains(err.Error(), "colour") {
		t.Errorf("refusal %q, want the key it did not know", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("refusal %q, want the file it read", err)
	}
}

func TestAnEmptyMarkInTheFileIsTheEmptyMark(t *testing.T) {
	path := write(t, "mark: \"\"\n")
	in, err := Resolve(Given{}, path)
	if err != nil {
		t.Fatal(err)
	}
	if in.Mark != "" {
		t.Errorf("Mark = %q, want the empty mark", in.Mark)
	}
	if in.MarkFrom != "file" {
		t.Errorf("MarkFrom = %q, want %q", in.MarkFrom, "file")
	}
}

// The empty mark is a choice, and `mark:` with nothing behind it is a line
// somebody did not finish. The refusal says how to ask for the empty mark.
func TestAMarkKeyWithNoValueIsRefused(t *testing.T) {
	path := write(t, "mark:\n")
	_, err := Resolve(Given{}, path)
	if err == nil {
		t.Fatal("Resolve() with mark: and no value, want a refusal")
	}
	if !strings.Contains(err.Error(), `mark: ""`) {
		t.Errorf("refusal %q, want the way to write the empty mark", err)
	}
}

func TestABadMarkIsRefusedNamingWhereItCameFrom(t *testing.T) {
	t.Run("the file and its key", func(t *testing.T) {
		path := write(t, "mark: \"**\"\n")
		_, err := Resolve(Given{}, path)
		if err == nil {
			t.Fatal("Resolve() with markdown in the file's mark, want a refusal")
		}
		if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "mark:") {
			t.Errorf("refusal %q, want the file and the key it came from", err)
		}
		if !strings.Contains(err.Error(), "**") {
			t.Errorf("refusal %q, want what it found", err)
		}
	})
	t.Run("the argument", func(t *testing.T) {
		mark := "**"
		_, err := Resolve(Given{Mark: &mark}, gone(t))
		if err == nil {
			t.Fatal("Resolve() with markdown in --mark, want a refusal")
		}
		if !strings.Contains(err.Error(), "--mark") {
			t.Errorf("refusal %q, want the flag it came from", err)
		}
		if strings.Contains(err.Error(), "settings.yaml") {
			t.Errorf("refusal %q names a file, and the argument is where it came from", err)
		}
	})
}

func TestMarkFromSaysWhichSourceWon(t *testing.T) {
	empty := ""
	for _, c := range []struct {
		name  string
		given Given
		body  string
		file  bool
		mark  string
		from  string
	}{
		{name: "nothing anywhere", mark: "⁘", from: "default"},
		{name: "the file alone", body: "mark: \"∴\"\n", file: true, mark: "∴", from: "file"},
		{
			name:  "the argument alone",
			given: Given{Mark: &empty},
			mark:  "",
			from:  "argument",
		},
		{
			name:  "both, and the argument wins",
			given: Given{Mark: &empty},
			body:  "mark: \"∴\"\n",
			file:  true,
			mark:  "",
			from:  "argument",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			path := gone(t)
			if c.file {
				path = write(t, c.body)
			}
			in, err := Resolve(c.given, path)
			if err != nil {
				t.Fatal(err)
			}
			if in.Mark != c.mark || in.MarkFrom != c.from {
				t.Errorf("Resolve() = %q from %q, want %q from %q", in.Mark, in.MarkFrom, c.mark, c.from)
			}
		})
	}
}

// The token and its path are not settings. They are a credential, and a file a
// colleague edits by hand is not where a credential's location is decided.
// Both read as unknown keys, which is the same refusal any other word gets.
func TestTheTokenIsNotASetting(t *testing.T) {
	for _, key := range []string{"token", "token_path"} {
		t.Run(key, func(t *testing.T) {
			path := write(t, key+": x\n")
			_, err := Resolve(Given{}, path)
			if err == nil {
				t.Fatalf("Resolve() with %s: in the file, want the refusal any unknown key gets", key)
			}
			if !strings.Contains(err.Error(), key) {
				t.Errorf("refusal %q, want the key it did not know", err)
			}
		})
	}
}
