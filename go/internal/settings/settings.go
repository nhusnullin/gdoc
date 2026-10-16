package settings

import (
	"errors"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"

	"gdoc/internal/plaintext"
)

// The two keys the file takes. They are named here because the refusals quote
// them: the unfinished line, a mark the file names that a thread would show
// wrong, and a presence that is neither true nor false.
const (
	key         = "mark"
	presenceKey = "presence"
)

// presenceArg is the flag the same value comes in on, named here because two
// refusals quote it and no file is in either of them.
const presenceArg = "--presence"

// presenceTakes is what both sources take, quoted by every refusal of a value
// that is neither. The argument takes these two words and nothing else, while
// the file takes whatever YAML reads as a boolean, `True` among them: the
// argument is a line somebody typed once and the file is a line they keep.
const presenceTakes = "true or false"

// The three answers MarkFrom gives. They are a fact about where the value came
// from, printed by help --json and never advice about what to do with it.
const (
	FromArgument = "argument"
	FromFile     = "file"
	FromDefault  = "default"
)

// Given is what the command line said, and nothing else. A nil pointer is a
// flag nobody typed, which is why each field is a pointer: --mark "" is the
// empty mark a colleague asked for, not the absence of an argument.
type Given struct {
	Config *string
	Mark   *string
	// Presence is the experimental flag as it was typed, "true" or "false"
	// and nothing else. It is a string rather than a *bool because the
	// refusal has to quote what the line said, and a bool that failed to
	// parse carries none of it.
	Presence *string
}

// InEffect is what this run writes, and where each value came from. File is
// the settings file that was read, or the empty string when there was none,
// so a caller can say which file a value came out of.
type InEffect struct {
	Mark     string
	MarkFrom string
	File     string
	// Presence is whether this run records what it is doing, and
	// PresenceFrom where that came from. PresenceFrom is the empty string
	// whenever Presence is false, including where a file or an argument said
	// so in as many words: while the presence is off there is nothing in
	// effect to name a source for, and an experimental flag says nothing
	// about itself in anybody else's output.
	// TestPresenceFromIsSetOnlyWhenTheFlagIsOn.
	Presence     bool
	PresenceFrom string
}

// file is the settings file's shape. Mark is a pointer so that a key with no
// value behind it can be told apart from a key nobody wrote, which present
// answers.
type file struct {
	Mark     *string `yaml:"mark"`
	Presence *bool   `yaml:"presence"`
}

// Resolve reads the settings file in use and lays the arguments over it. The
// order is the argument, then the file, then the built-in default, and the
// file is read whether or not an argument overrides what it says: a file gdoc
// cannot make sense of is one a colleague has to fix either way.
//
// defaultPath is the file to read when --config named none, and its absence is
// every default. A named --config that is not there is refused.
func Resolve(given Given, defaultPath string) (InEffect, error) {
	path, named := defaultPath, false
	if given.Config != nil {
		path, named = *given.Config, true
	}
	fromFile, read, err := readFile(path, named)
	if err != nil {
		return InEffect{}, err
	}
	in := InEffect{Mark: plaintext.DefaultMark, MarkFrom: FromDefault}
	if read {
		in.File = path
	}
	if fromFile.Mark != nil {
		if err := plaintext.CheckMark(*fromFile.Mark); err != nil {
			return InEffect{}, fmt.Errorf("%s, %s: %w", path, key, err)
		}
		in.Mark, in.MarkFrom = *fromFile.Mark, FromFile
	}
	if given.Mark != nil {
		if err := plaintext.CheckMark(*given.Mark); err != nil {
			return InEffect{}, fmt.Errorf("--mark: %w", err)
		}
		in.Mark, in.MarkFrom = *given.Mark, FromArgument
	}
	if fromFile.Presence != nil {
		in.Presence, in.PresenceFrom = *fromFile.Presence, FromFile
	}
	if given.Presence != nil {
		on, err := presenceArgument(*given.Presence)
		if err != nil {
			return InEffect{}, err
		}
		in.Presence, in.PresenceFrom = on, FromArgument
	}
	if !in.Presence {
		in.PresenceFrom = ""
	}
	return in, nil
}

// presenceArgument reads the one argument that is a boolean. It takes the two
// words and nothing else: `--presence yes` and `--presence True` are refused
// naming the flag, because a line somebody typed is a line they can retype,
// and a value gdoc half understood is the one thing a flag may never be.
func presenceArgument(value string) (bool, error) {
	switch value {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("%s takes %s, and %q is neither", presenceArg, presenceTakes, value)
}

// readFile reads one settings file strictly. It answers the mark the file
// names, whether there was a file at all, and the refusal when there was one
// gdoc could not read.
//
// A missing file is every default when nobody named it, and a refusal when
// somebody did: a mistyped path that quietly became every default would hand a
// colleague the wrong mark and say nothing.
func readFile(path string, named bool) (file, bool, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if named {
			return file{}, false, fmt.Errorf("--config names %s, and there is no file there; write the file or leave --config out to use the default settings", path)
		}
		return file{}, false, nil
	}
	if err != nil {
		return file{}, false, fmt.Errorf("settings: %w", err)
	}
	var f file
	if err := yaml.UnmarshalWithOptions(b, &f, yaml.Strict()); err != nil {
		if bad := badPresence(b); bad != nil {
			return file{}, true, fmt.Errorf("%s: %w", path, bad)
		}
		return file{}, true, fmt.Errorf("%s: %w", path, err)
	}
	if f.Mark == nil && present(b, key) {
		return file{}, true, fmt.Errorf(`%s: %s: has no value behind it; write %s: "" for the empty mark, or a mark of your own`, path, key, key)
	}
	if f.Presence == nil && present(b, presenceKey) {
		return file{}, true, fmt.Errorf("%s: %s: has no value behind it; write %s: true to record what gdoc is doing, or leave the key out", path, presenceKey, presenceKey)
	}
	return f, true, nil
}

// badPresence is the refusal for a presence the file wrote that is no boolean,
// in gdoc's words. It is asked only where the strict decode already failed,
// and it answers nil for every other reason that decode had.
//
// It exists because go-yaml names the Go field it was decoding into rather
// than the key the file wrote, so `presence: yes` would be refused with a
// sentence about a field called Presence, which is no line a colleague can go
// and change: TestABadPresenceValueIsRefusedNamingWhereItCameFrom.
func badPresence(b []byte) error {
	var loose map[string]any
	if err := yaml.Unmarshal(b, &loose); err != nil {
		return nil
	}
	v, ok := loose[presenceKey]
	if !ok {
		return nil
	}
	if _, isBool := v.(bool); isBool || v == nil {
		return nil
	}
	return fmt.Errorf("%s: takes %s, and %q is neither", presenceKey, presenceTakes, fmt.Sprint(v))
}

// present answers whether the file wrote a key at all, which the strict decode
// into file cannot say: a key with nothing behind it and a key nobody wrote
// both leave a nil pointer. The loose read is safe here because the strict one
// above has already refused everything this file may not hold.
func present(b []byte, name string) bool {
	var loose map[string]any
	if err := yaml.Unmarshal(b, &loose); err != nil {
		return false
	}
	_, ok := loose[name]
	return ok
}
