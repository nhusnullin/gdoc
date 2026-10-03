// The guide code: the one argument every tool call carries that no command of
// the terminal has, and the sentence a call without it gets back.

package chat

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// codeBytes is how many random bytes one code is made of. Eight bytes is
// sixteen hex characters: short enough for a model to carry in every call, and
// far past anything that collides with another process's by accident.
const codeBytes = 8

// RetrySentence is what a tool answers when the code is missing or stale. It
// names the one thing that fixes it and nothing else, so a model reads it and
// calls guide without bothering the person about a code they never typed.
const RetrySentence = "call guide first, then retry this same call with the code it gives"

// Code is the code one server process hands out through guide. It is made once
// when the process starts and never written down, so a code is good for exactly
// the session it was made in: TestACodeFromAnotherProcessIsStale.
type Code struct {
	value string
}

// NewCode makes the code for one process. The error is the random source
// failing, which a session cannot start without: a code nothing can make is a
// server where every tool refuses, and saying so at the start is the only place
// a person can read it.
func NewCode() (*Code, error) {
	b := make([]byte, codeBytes)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("the guide code could not be made, so this session cannot start: %w", err)
	}
	return &Code{value: hex.EncodeToString(b)}, nil
}

// Value is the code itself, which guide answers with and nothing else prints.
func (c *Code) Value() string {
	if c == nil {
		return ""
	}
	return c.value
}

// Check judges the code a call carried. It is an exact comparison and not a
// secret kept from anybody: the code proves the rules entered the model's
// context, which is reliability rather than safety, and the holds are what
// stand between a comment and a write.
func (c *Code) Check(got string) error {
	if c == nil || c.value == "" {
		return errors.New("this session never made a guide code, so no tool can run. It is a fault in gdoc, not something the person can fix")
	}
	if got != c.value {
		return errors.New(RetrySentence)
	}
	return nil
}
