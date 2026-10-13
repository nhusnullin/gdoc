package settings

import (
	"errors"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"

	"gdoc/internal/plaintext"
)

// key is the one key the file takes. It is named here because two refusals
// quote it: the unfinished line, and a mark the file names that a thread would
// show wrong.
const key = "mark"

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
}

// InEffect is what this run writes, and where each value came from. File is
// the settings file that was read, or the empty string when there was none,
// so a caller can say which file a value came out of.
type InEffect struct {
	Mark     string
	MarkFrom string
	File     string
}

// file is the settings file's shape. Mark is a pointer so that a key with no
// value behind it can be told apart from a key nobody wrote, which present
// answers.
type file struct {
	Mark *string `yaml:"mark"`
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
	if fromFile != nil {
		if err := plaintext.CheckMark(*fromFile); err != nil {
			return InEffect{}, fmt.Errorf("%s, %s: %w", path, key, err)
		}
		in.Mark, in.MarkFrom = *fromFile, FromFile
	}
	if given.Mark != nil {
		if err := plaintext.CheckMark(*given.Mark); err != nil {
			return InEffect{}, fmt.Errorf("--mark: %w", err)
		}
		in.Mark, in.MarkFrom = *given.Mark, FromArgument
	}
	return in, nil
}

// readFile reads one settings file strictly. It answers the mark the file
// names, whether there was a file at all, and the refusal when there was one
// gdoc could not read.
//
// A missing file is every default when nobody named it, and a refusal when
// somebody did: a mistyped path that quietly became every default would hand a
// colleague the wrong mark and say nothing.
func readFile(path string, named bool) (*string, bool, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if named {
			return nil, false, fmt.Errorf("--config names %s, and there is no file there; write the file or leave --config out to use the default settings", path)
		}
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("settings: %w", err)
	}
	var f file
	if err := yaml.UnmarshalWithOptions(b, &f, yaml.Strict()); err != nil {
		return nil, true, fmt.Errorf("%s: %w", path, err)
	}
	if f.Mark == nil && present(b, key) {
		return nil, true, fmt.Errorf(`%s: %s: has no value behind it; write %s: "" for the empty mark, or a mark of your own`, path, key, key)
	}
	return f.Mark, true, nil
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
