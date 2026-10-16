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

// The presence is off unless somebody turns it on, and nothing was read.
// PresenceFrom is the empty string there, because where a value nobody set
// came from is a question with no answer worth printing.
func TestPresenceIsOffByDefault(t *testing.T) {
	in, err := Resolve(Given{}, gone(t))
	if err != nil {
		t.Fatal(err)
	}
	if in.Presence {
		t.Error("Presence is on with nothing anywhere that asked for it")
	}
	if in.PresenceFrom != "" {
		t.Errorf("PresenceFrom = %q, want the empty string while the presence is off", in.PresenceFrom)
	}
}

// The file turns it on, in either spelling go-yaml reads as a boolean. `True`
// is one of them, which is the difference from the argument below.
func TestTheFileTurnsPresenceOn(t *testing.T) {
	for _, body := range []string{"presence: true\n", "presence: True\n"} {
		t.Run(body, func(t *testing.T) {
			in, err := Resolve(Given{}, write(t, body))
			if err != nil {
				t.Fatal(err)
			}
			if !in.Presence {
				t.Error("Presence is off, and the file turned it on")
			}
			if in.PresenceFrom != "file" {
				t.Errorf("PresenceFrom = %q, want %q", in.PresenceFrom, "file")
			}
		})
	}
	t.Run("presence: false\n", func(t *testing.T) {
		in, err := Resolve(Given{}, write(t, "presence: false\n"))
		if err != nil {
			t.Fatal(err)
		}
		if in.Presence || in.PresenceFrom != "" {
			t.Errorf("Resolve() = %v from %q, want it off from nowhere", in.Presence, in.PresenceFrom)
		}
	})
}

func TestThePresenceArgumentOverridesTheFile(t *testing.T) {
	t.Run("on over a file that says off", func(t *testing.T) {
		on := "true"
		in, err := Resolve(Given{Presence: &on}, write(t, "presence: false\n"))
		if err != nil {
			t.Fatal(err)
		}
		if !in.Presence || in.PresenceFrom != "argument" {
			t.Errorf("Resolve() = %v from %q, want it on from the argument", in.Presence, in.PresenceFrom)
		}
	})
	t.Run("off over a file that says on", func(t *testing.T) {
		off := "false"
		in, err := Resolve(Given{Presence: &off}, write(t, "presence: true\n"))
		if err != nil {
			t.Fatal(err)
		}
		if in.Presence {
			t.Error("Presence is on, and the argument turned it off")
		}
		if in.PresenceFrom != "" {
			t.Errorf("PresenceFrom = %q, want the empty string while the presence is off", in.PresenceFrom)
		}
	})
}

// Every refusal names where the value came from and what gdoc takes, in
// gdoc's words: go-yaml names the Go field rather than the key, which is no
// line a colleague can go and change.
func TestABadPresenceValueIsRefusedNamingWhereItCameFrom(t *testing.T) {
	for _, body := range []string{"presence: yes\n", "presence: 1\n", "presence: \"\"\n"} {
		t.Run(body, func(t *testing.T) {
			path := write(t, body)
			_, err := Resolve(Given{}, path)
			if err == nil {
				t.Fatalf("Resolve() with %q, want a refusal", body)
			}
			if !strings.Contains(err.Error(), path) {
				t.Errorf("refusal %q, want the file it read", err)
			}
			if !strings.Contains(err.Error(), "presence") {
				t.Errorf("refusal %q, want the key it came from", err)
			}
			if !strings.Contains(err.Error(), "true or false") {
				t.Errorf("refusal %q, want what the key takes", err)
			}
			// The path is cut off first: a temp directory named after this
			// test carries the word the assertion looks for.
			if said := strings.TrimPrefix(err.Error(), path); strings.Contains(said, "Presence") {
				t.Errorf("refusal %q names the Go field, and a colleague edits a key", said)
			}
		})
	}
	for _, value := range []string{"True", "yes", "1", ""} {
		t.Run("--presence "+value, func(t *testing.T) {
			given := value
			_, err := Resolve(Given{Presence: &given}, gone(t))
			if err == nil {
				t.Fatalf("Resolve() with --presence %q, want a refusal", value)
			}
			if !strings.Contains(err.Error(), "--presence") {
				t.Errorf("refusal %q, want the flag it came from", err)
			}
			if !strings.Contains(err.Error(), "true or false") {
				t.Errorf("refusal %q, want what the flag takes", err)
			}
			if strings.Contains(err.Error(), "settings.yaml") {
				t.Errorf("refusal %q names a file, and the argument is where it came from", err)
			}
		})
	}
}

// `presence:` with nothing behind it is a line somebody did not finish, and
// the strict decode cannot tell it from a key nobody wrote. present answers it.
func TestAPresenceKeyWithNoValueIsRefused(t *testing.T) {
	path := write(t, "presence:\n")
	_, err := Resolve(Given{}, path)
	if err == nil {
		t.Fatal("Resolve() with presence: and no value, want a refusal")
	}
	if !strings.Contains(err.Error(), "presence: true") {
		t.Errorf("refusal %q, want the way to turn the presence on", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("refusal %q, want the file it read", err)
	}
}

// PresenceFrom is a fact only while the presence is on. Off is off from
// everywhere, and an experimental flag says nothing about itself until it is
// in effect.
func TestPresenceFromIsSetOnlyWhenTheFlagIsOn(t *testing.T) {
	off, on := "false", "true"
	for _, c := range []struct {
		name  string
		given Given
		body  string
		file  bool
	}{
		{name: "nothing anywhere"},
		{name: "the file says off", body: "presence: false\n", file: true},
		{name: "the argument says off", given: Given{Presence: &off}},
		{name: "the argument says off over a file that says on", given: Given{Presence: &off}, body: "presence: true\n", file: true},
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
			if in.PresenceFrom != "" {
				t.Errorf("PresenceFrom = %q while the presence is off", in.PresenceFrom)
			}
		})
	}
	t.Run("the argument says on", func(t *testing.T) {
		in, err := Resolve(Given{Presence: &on}, gone(t))
		if err != nil {
			t.Fatal(err)
		}
		if in.PresenceFrom != "argument" {
			t.Errorf("PresenceFrom = %q, want %q", in.PresenceFrom, "argument")
		}
	})
}

func TestPresenceFromSaysWhichSourceWon(t *testing.T) {
	on := "true"
	for _, c := range []struct {
		name  string
		given Given
		body  string
		file  bool
		from  string
	}{
		{name: "the file alone", body: "presence: true\n", file: true, from: "file"},
		{name: "the argument alone", given: Given{Presence: &on}, from: "argument"},
		{
			name:  "both, and the argument wins",
			given: Given{Presence: &on},
			body:  "presence: true\n",
			file:  true,
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
			if !in.Presence || in.PresenceFrom != c.from {
				t.Errorf("Resolve() = %v from %q, want it on from %q", in.Presence, in.PresenceFrom, c.from)
			}
		})
	}
}

// The mark and the presence are read out of one file and laid over by one
// order, and neither one's refusal hides the other's value.
func TestTheMarkAndThePresenceComeOutOfOneFile(t *testing.T) {
	path := write(t, "mark: \"∴\"\npresence: true\n")
	in, err := Resolve(Given{}, path)
	if err != nil {
		t.Fatal(err)
	}
	if in.Mark != "∴" || in.MarkFrom != "file" {
		t.Errorf("Mark = %q from %q, want the three-dot mark from the file", in.Mark, in.MarkFrom)
	}
	if !in.Presence || in.PresenceFrom != "file" {
		t.Errorf("Presence = %v from %q, want it on from the file", in.Presence, in.PresenceFrom)
	}
}
