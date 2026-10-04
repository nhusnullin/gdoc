package chat

import (
	"regexp"
	"strings"
	"testing"
)

// The code is sixteen hex characters. The number is stated here as a literal,
// because a test that read the constant would follow it wherever somebody moved
// it, and the length is the whole of what makes a code hard to arrive at by
// accident.
func TestACodeIsSixteenHexCharacters(t *testing.T) {
	// Arrange
	hex := regexp.MustCompile(`^[0-9a-f]{16}$`)

	// Act
	code, err := NewCode()

	// Assert
	if err != nil {
		t.Fatalf("a code could not be made: %v", err)
	}
	if !hex.MatchString(code.Value()) {
		t.Errorf("the code is %q, want sixteen lowercase hex characters", code.Value())
	}
}

// The code a process gave out is the code it takes back, and nothing else a
// model might send instead.
func TestTheCodeAProcessGaveIsTaken(t *testing.T) {
	// Arrange
	code, err := NewCode()
	if err != nil {
		t.Fatal(err)
	}

	// Act
	err = code.Check(code.Value())

	// Assert
	if err != nil {
		t.Errorf("the code this session gave out was refused: %v", err)
	}
	if err := code.Check(strings.ToUpper(code.Value())); err == nil {
		t.Error("the code is sixteen lowercase characters, and another spelling of it is not the code")
	}
}

// Claude Desktop runs two gdoc processes, and a chat that was open before an
// update is a chat holding a code the process that answers it never made. Both
// are the same thing: a code is good for the one session that made it.
func TestACodeFromAnotherProcessIsStale(t *testing.T) {
	// Arrange
	first, err := NewCode()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewCode()
	if err != nil {
		t.Fatal(err)
	}

	// Act and assert
	if first.Value() == second.Value() {
		t.Fatal("two processes made the same code, so nothing here is random")
	}
	if err := first.Check(second.Value()); err == nil {
		t.Error("one process took the other process's code")
	}
	if err := second.Check(first.Value()); err == nil {
		t.Error("one process took the other process's code")
	}
}

// The refusal is the one sentence, stated here as a literal. A model reads it,
// calls guide and makes the same call again: there is nothing in it for the
// person to do, because the code is not theirs.
func TestAMissingCodeIsRefusedWithTheRetrySentence(t *testing.T) {
	// Arrange
	const want = "call guide first, then retry this same call with the code it gives"
	code, err := NewCode()
	if err != nil {
		t.Fatal(err)
	}

	// Act and assert
	for _, got := range []string{"", " ", "not the code", code.Value() + "0"} {
		err := code.Check(got)
		if err == nil {
			t.Errorf("%q was taken for the code", got)
			continue
		}
		if err.Error() != want {
			t.Errorf("%q was refused with %q, want %q", got, err.Error(), want)
		}
	}

	// A session with no code of its own refuses every call, and says it is a
	// fault rather than telling the model to call guide for a code nothing can
	// hand out.
	var none *Code
	err = none.Check("anything")
	if err == nil {
		t.Fatal("a session with no code must refuse every call")
	}
	if strings.Contains(err.Error(), want) {
		t.Errorf("a session with no code tells the model to retry, which it cannot: %q", err.Error())
	}
}
