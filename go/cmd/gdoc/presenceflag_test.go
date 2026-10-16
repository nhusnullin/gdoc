// The experimental setting flag: in the table, and in nothing a reader is
// shown.
//
// This file is the pin over that rule. The help screens, both completion
// scripts and the help object are what every other colleague reads, and a
// flag one person is trying out may move no byte of them. The screens have
// their own recorded goldens next door; what is read here is those files and
// the two scripts, by the one name the flag has.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The help object a checkout build printed before this flag existed, taken
// from `gdoc help --json` on 2026-10-17 with the reach refusing everything, by
// length and by sha256. Both literals, so a byte that moves is named here and
// nowhere else.
const (
	helpObjectBytes  = 8921
	helpObjectSHA256 = "40a28f2a2ff65037099cfeee3109f701e57b6de95e44569fc57c1bf5ca2c0a46"
)

// One flag of the three is experimental, it is --presence, and it carries the
// sentence that says what it does. A flag in the table with no sentence is a
// flag nobody can be told about once it stops being experimental.
func TestTheTableNamesTheOneExperimentalFlag(t *testing.T) {
	var experimental []string
	var summary string
	for _, f := range settingFlags() {
		if !f.experimental {
			continue
		}
		experimental = append(experimental, f.name)
		summary = f.summary
	}
	if len(experimental) != 1 || experimental[0] != presenceFlag {
		t.Fatalf("the experimental flags are %v, and the one is %s", experimental, presenceFlag)
	}
	if !strings.Contains(summary, "true or false") {
		t.Errorf("the sentence must say what the flag takes: %q", summary)
	}
	if !strings.Contains(summary, "presence file") {
		t.Errorf("the sentence must say what the flag writes: %q", summary)
	}
	for _, f := range shownSettingFlags() {
		if f.name == presenceFlag {
			t.Errorf("%s is offered, and it is experimental", presenceFlag)
		}
	}
}

// The flag reaches no screen, no script and no help object while it is off.
// The fifteen help screens are read off disk as they stand, because a golden
// this task rewrote would prove nothing.
func TestAnExperimentalFlagIsInTheTableAndInNoScreen(t *testing.T) {
	t.Run("the fifteen help screens", func(t *testing.T) {
		names, err := filepath.Glob(filepath.Join("testdata", "screens", "*.golden"))
		if err != nil {
			t.Fatal(err)
		}
		screens := 0
		for _, name := range names {
			base := filepath.Base(name)
			if !strings.HasPrefix(base, "help") && !strings.HasPrefix(base, "bare") {
				continue
			}
			screens++
			b, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			// The flag has a hyphen in it, which no screen wraps on, so the
			// whole word is looked for and so is the key the file takes.
			for _, word := range []string{presenceFlag, "presence"} {
				if strings.Contains(string(b), word) {
					t.Errorf("%s names %q, and the flag is experimental", base, word)
				}
			}
		}
		if screens != 15 {
			t.Errorf("the help screens are %d, and there are fifteen of them", screens)
		}
	})

	t.Run("the words a person reads", func(t *testing.T) {
		prose := helpProse(commands(), true) + helpProse(helpMatches([]string{"publish"}), false)
		if strings.Contains(prose, "presence") {
			t.Errorf("the help a person reads names the presence: %q", prose)
		}
	})

	t.Run("both completion scripts", func(t *testing.T) {
		zsh, err := zshScript(commands())
		if err != nil {
			t.Fatal(err)
		}
		bash, err := bashScript(commands())
		if err != nil {
			t.Fatal(err)
		}
		for name, script := range map[string]string{"zsh": zsh, "bash": bash} {
			if strings.Contains(script, "presence") {
				t.Errorf("the %s script names the presence: %q", name, script)
			}
		}
	})

	t.Run("the help object with the flag off", func(t *testing.T) {
		got := helpObject(t)
		if len(got) != helpObjectBytes {
			t.Errorf("the help object is %d bytes, and it was %d before the flag existed", len(got), helpObjectBytes)
		}
		sum := sha256.Sum256(got)
		if hex.EncodeToString(sum[:]) != helpObjectSHA256 {
			t.Errorf("the help object is\n%s\nand its sha256 is %s, not %s", got, hex.EncodeToString(sum[:]), helpObjectSHA256)
		}
	})
}

// With the flag on, the object carries the two fields and nothing else moves:
// a caller already using the flag is the one caller that asks the help about
// it.
func TestTheHelpObjectCarriesThePresenceOnlyWhenItIsOn(t *testing.T) {
	for _, c := range []struct {
		name string
		args []string
		from string
	}{
		{name: "off", args: []string{"help", "--json"}},
		{name: "on from the argument", args: []string{presenceFlag, "true", "help", "--json"}, from: "argument"},
		{name: "off in as many words", args: []string{presenceFlag, "false", "help", "--json"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			set := settingsOf(t, helpObject(t, c.args...))
			on, said := set["presence"]
			if c.from == "" {
				if said {
					t.Errorf("the object carries presence = %v while the flag is off", on)
				}
				if _, said := set["presence_from"]; said {
					t.Errorf("the object carries presence_from while the flag is off: %v", set["presence_from"])
				}
				return
			}
			if on != true {
				t.Errorf("presence = %v, want true", on)
			}
			if set["presence_from"] != c.from {
				t.Errorf("presence_from = %v, want %q", set["presence_from"], c.from)
			}
			if set["mark"] != "⁘" {
				t.Errorf("mark = %v, and the presence flag moved it", set["mark"])
			}
		})
	}
}

// helpObject is the bytes `gdoc help --json` writes to stdout, through the
// whole line the way a shell hands it over, with the version empty and the
// reach refusing everything so nothing is asked of GitHub.
func helpObject(t *testing.T, args ...string) []byte {
	t.Helper()
	if len(args) == 0 {
		args = []string{"help", "--json"}
	}
	t.Setenv("GDOC_CONFIG_DIR", t.TempDir())
	checking(t, "", quietPlain{})
	var out, errOut strings.Builder
	code := route(context.Background(), args, strings.NewReader(""), &out, &errOut)
	if code != 0 {
		t.Fatalf("%v is an answer and exited %d: %s", args, code, errOut.String())
	}
	return []byte(out.String())
}

// settingsOf is the settings object inside one help envelope.
func settingsOf(t *testing.T, object []byte) map[string]any {
	t.Helper()
	var got struct {
		Data struct {
			Settings map[string]any `json:"settings"`
		} `json:"data"`
	}
	if err := json.Unmarshal(object, &got); err != nil {
		t.Fatalf("the help object does not decode: %v: %s", err, object)
	}
	return got.Data.Settings
}
