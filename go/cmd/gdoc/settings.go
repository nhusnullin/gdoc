// The two setting flags, taken off the line before anything else reads it.
//
// This file is the reading, and internal/settings is the deciding: what the
// order is, what a mark may be, and what the file may hold are its rules. What
// is here is the line a person typed turned into the pair of questions that
// package answers, and the one place the default file's path is named.

package main

import (
	"fmt"
	"strings"

	"gdoc/internal/config"
	"gdoc/internal/settings"
)

// takeSettings lifts the two setting flags off the line, wherever they stand,
// and answers what is left beside what they said.
//
// Wherever they stand, and before anything else reads the line, because every
// other reading of it would otherwise have to know them: `gdoc --mark ∴ help`
// is a help, `gdoc mcp --mark ∴` is a session rather than a refusal, and
// `gdoc read <url> --mark ∴` is a read whose own parser never hears of the
// flag. TestTheSettingFlagsAreTakenAnywhereOnTheLine and
// TestEveryCommandTakesTheSettingFlags.
//
// Both spellings, `--mark x` and `--mark=x`, because the parser every other
// flag goes through takes both. `--mark ""` and `--mark=` are the empty mark a
// colleague asked for, which is why settings.Given holds pointers:
// TestAnEmptyMarkArgumentIsTheEmptyMark.
//
// The refusals are parseArgsN's own, for the same reasons. A flag given twice
// is refused rather than resolved, because which one counts is not decided
// here. A flag with nothing after it is refused naming it. So is one whose
// value is the other setting flag, because `--mark --config f` would otherwise
// write "--config" in front of gdoc's words and read the file nobody named.
// And `--config` with an empty value is refused, because the empty string
// names no file, while the empty mark is a mark.
// TestASettingFlagGivenTwiceIsRefused and TestASettingFlagWithNoValueIsRefused.
//
// A refusal still gives the line back, and the reading finishes it: the first
// reason is the one kept, and every setting flag comes off the line whatever
// stood before it. Nothing runs on that line, because route has an error in
// hand, and the one question left to ask of it is whether the line was a
// session. A line that stopped at the first refusal would answer that question
// with a flag where the command should be, so `gdoc --config= --mark ⁘ mcp`
// would print a JSON envelope into somebody's protocol stream instead of one
// line on stderr. TestMcpTakesTheSettingFlagsAndNothingElse.
func takeSettings(args []string) ([]string, settings.Given, error) {
	rest := make([]string, 0, len(args))
	var given settings.Given
	var refusal error
	// refuse keeps the first reason and reads on. The reading goes to the end
	// of the line either way, so what is left is the command and its own
	// arguments whichever flag was wrong, and nothing the caller is handed came
	// from a flag the line refused.
	refuse := func(err error) {
		if refusal == nil {
			refusal = err
		}
	}
	for i := 0; i < len(args); i++ {
		name, value, joined := strings.Cut(args[i], "=")
		if name != configFlag && name != markFlag {
			rest = append(rest, args[i])
			continue
		}
		if !joined {
			if i+1 >= len(args) {
				refuse(fmt.Errorf("%s needs a value, and none follows it", name))
				continue
			}
			if next := args[i+1]; next == configFlag || next == markFlag {
				refuse(fmt.Errorf("%s needs a value, and %q is another flag", name, next))
				continue
			}
			i++
			value = args[i]
		}
		held := &given.Mark
		if name == configFlag {
			if value == "" {
				refuse(fmt.Errorf("%s was given an empty value", name))
				continue
			}
			held = &given.Config
		}
		if *held != nil {
			refuse(fmt.Errorf("%s is given twice, and which one counts is not decided here", name))
			continue
		}
		*held = &value
	}
	if refusal != nil {
		return rest, settings.Given{}, refusal
	}
	return rest, given, nil
}

// resolveSettings is what this run writes: the file in use with the line laid
// over it. internal/settings decides that order and refuses what it cannot
// read.
//
// The default path is settings.yaml in the config folder, and it is asked for
// only where --config named no file of its own, so a machine whose config
// folder cannot be named still runs `gdoc build --config <file>`. Where it is
// asked for and cannot be answered, the run is refused rather than read as
// every default: a settings file that was never found holds a mark somebody
// set and gdoc would not be writing.
func resolveSettings(given settings.Given) (settings.InEffect, error) {
	path := ""
	if given.Config == nil {
		p, err := config.SettingsPath()
		if err != nil {
			return settings.InEffect{}, err
		}
		path = p
	}
	return settings.Resolve(given, path)
}
