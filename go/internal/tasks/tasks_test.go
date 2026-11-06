// The notation, read from one task item's line. Every assertion states the
// words, the address and the day as literals, so a test that passes says what
// a person typed and what gdoc made of it.

package tasks

import (
	"strings"
	"testing"
)

// A token counts only where it ends the line. An address or a date in the
// middle of a sentence is a sentence, because a person writing "Ask @anna
// about 2026-01-02 today" is writing about a person and a day, not assigning
// anything.
func TestTheTokensAreReadOnlyAtTheEnd(t *testing.T) {
	// Arrange
	const source = "Ask @anna about 2026-01-02 today"

	// Act
	task, err := Parse(source)

	// Assert
	if err != nil {
		t.Fatalf("Parse(%q) refused it: %v", source, err)
	}
	if task.Words != "Ask @anna about 2026-01-02 today" {
		t.Errorf("Words = %q, want the whole line", task.Words)
	}
	if task.Owner.Present() {
		t.Errorf("Owner = %q, want no owner", task.Owner.Text)
	}
	if task.Due.Present() {
		t.Errorf("Due = %q, want no date", task.Due.Text)
	}
}

// At the end of the line the two may stand in either order, because nobody
// remembers an order. Each token's span is where it stood, which is where its
// chip goes.
func TestOwnerAndDateInEitherOrder(t *testing.T) {
	for _, c := range []struct {
		source     string
		ownerStart int
		ownerEnd   int
		dueStart   int
		dueEnd     int
	}{
		{"Fix the login @anna 2026-01-02", 14, 19, 20, 30},
		{"Fix the login 2026-01-02 @anna", 25, 30, 14, 24},
	} {
		// Act
		task, err := Parse(c.source)

		// Assert
		if err != nil {
			t.Fatalf("Parse(%q) refused it: %v", c.source, err)
		}
		if task.Words != "Fix the login" {
			t.Errorf("Parse(%q) Words = %q, want %q", c.source, task.Words, "Fix the login")
		}
		if task.Owner.Text != "@anna" {
			t.Errorf("Parse(%q) Owner = %q, want %q", c.source, task.Owner.Text, "@anna")
		}
		if task.Owner.Name != "anna" || task.Owner.Domain != "" {
			t.Errorf("Parse(%q) owner name %q domain %q, want %q and the empty domain", c.source, task.Owner.Name, task.Owner.Domain, "anna")
		}
		if task.Due.Text != "2026-01-02" {
			t.Errorf("Parse(%q) Due = %q, want %q", c.source, task.Due.Text, "2026-01-02")
		}
		if task.Owner.Start != c.ownerStart || task.Owner.End != c.ownerEnd {
			t.Errorf("Parse(%q) owner span %d to %d, want %d to %d", c.source, task.Owner.Start, task.Owner.End, c.ownerStart, c.ownerEnd)
		}
		if task.Due.Start != c.dueStart || task.Due.End != c.dueEnd {
			t.Errorf("Parse(%q) date span %d to %d, want %d to %d", c.source, task.Due.Start, task.Due.End, c.dueStart, c.dueEnd)
		}
		if got := c.source[task.Owner.Start:task.Owner.End]; got != "@anna" {
			t.Errorf("Parse(%q) owner span holds %q", c.source, got)
		}
		if got := c.source[task.Due.Start:task.Due.End]; got != "2026-01-02" {
			t.Errorf("Parse(%q) date span holds %q", c.source, got)
		}
	}
}

// Two owners are refused naming both, because which of them the task is for is
// not a thing gdoc decides.
func TestTwoOwnersAreRefusedByName(t *testing.T) {
	// Arrange
	const source = "Fix the login @anna @bob"

	// Act
	_, err := Parse(source)

	// Assert
	if err == nil {
		t.Fatalf("Parse(%q) took two owners", source)
	}
	for _, want := range []string{"@anna", "@bob", "one owner"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must name %q: %v", want, err)
		}
	}
}

// Two dates, the same rule.
func TestTwoDatesAreRefusedByName(t *testing.T) {
	// Arrange
	const source = "Fix the login 2026-01-02 2026-01-03"

	// Act
	_, err := Parse(source)

	// Assert
	if err == nil {
		t.Fatalf("Parse(%q) took two dates", source)
	}
	for _, want := range []string{"2026-01-02", "2026-01-03", "one date"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must name %q: %v", want, err)
		}
	}
}

// A date shaped like the notation that is no day on a calendar is refused by
// name rather than carried to a chip that would land on some other day.
func TestADateThatIsNotADayIsRefused(t *testing.T) {
	// Arrange
	const source = "Fix the login 2026-02-30"

	// Act
	_, err := Parse(source)

	// Assert
	if err == nil {
		t.Fatalf("Parse(%q) took a day that is not one", source)
	}
	if !strings.Contains(err.Error(), "2026-02-30") {
		t.Errorf("the refusal must name the date: %v", err)
	}
}

// A word that opens with @ at the end of a task line is somebody's address or
// it is nothing, and nothing is refused naming the word.
func TestAnOwnerThatIsNotAnAddressIsRefused(t *testing.T) {
	for _, word := range []string{"@a@b", "@", "@anna@", "@anna."} {
		source := "Fix the login " + word

		// Act
		_, err := Parse(source)

		// Assert
		if err == nil {
			t.Fatalf("Parse(%q) took %q as an address", source, word)
		}
		if !strings.Contains(err.Error(), word) {
			t.Errorf("the refusal of %q must name it: %v", word, err)
		}
	}
}

// A bare name takes the domain in effect, so a person writes @anna and the
// chip carries anna@example.com.
func TestABareNameTakesTheDomain(t *testing.T) {
	// Arrange
	task, err := Parse("Fix the login @anna")
	if err != nil {
		t.Fatal(err)
	}

	// Act
	address, err := task.Owner.Resolve("example.com")

	// Assert
	if err != nil {
		t.Fatalf("Resolve refused it: %v", err)
	}
	if address != "anna@example.com" {
		t.Errorf("address = %q, want %q", address, "anna@example.com")
	}
}

// With no domain in effect a bare name is refused naming the setting, before
// anything is sent anywhere.
func TestABareNameWithNoDomainIsRefusedNamingTheSetting(t *testing.T) {
	// Arrange
	task, err := Parse("Fix the login @anna")
	if err != nil {
		t.Fatal(err)
	}

	// Act
	_, err = task.Owner.Resolve("")

	// Assert
	if err == nil {
		t.Fatal("Resolve answered an address with no domain in effect")
	}
	for _, want := range []string{"@anna", "domain", "--domain"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must name %q: %v", want, err)
		}
	}
}

// A whole address is the whole address. The setting is for the names that are
// short of one, and it never rewrites one somebody wrote out.
func TestAFullAddressIgnoresTheDomain(t *testing.T) {
	// Arrange
	task, err := Parse("Fix the login @anna@example.org")
	if err != nil {
		t.Fatal(err)
	}

	// Act
	address, err := task.Owner.Resolve("example.com")

	// Assert
	if err != nil {
		t.Fatalf("Resolve refused it: %v", err)
	}
	if address != "anna@example.org" {
		t.Errorf("address = %q, want %q", address, "anna@example.org")
	}
	if task.Owner.Domain != "example.org" {
		t.Errorf("Domain = %q, want %q", task.Owner.Domain, "example.org")
	}
}

// A domain is letters, digits, hyphens and dots with at least one dot, and
// everything else is refused naming the value. The setting asks this package,
// so one rule serves the file, the flag and an address somebody wrote out.
func TestADomainIsLettersDigitsHyphensAndDots(t *testing.T) {
	for _, good := range []string{"example.com", "mail.example.co.uk", "my-firm.example"} {
		if err := CheckDomain(good); err != nil {
			t.Errorf("CheckDomain(%q) refused it: %v", good, err)
		}
	}
	for _, bad := range []string{"", "example", "example.com@", "example com", "-example.com", "example..com", "example.com."} {
		err := CheckDomain(bad)
		if err == nil {
			t.Errorf("CheckDomain(%q) took it", bad)
			continue
		}
		if bad != "" && !strings.Contains(err.Error(), bad) {
			t.Errorf("the refusal of %q must name it: %v", bad, err)
		}
	}
}

// A token internal/body already read goes back through ReadOwner, because a
// publish resolves every owner against the domain in effect and the note kept
// only what the author typed. The same word through Parse and through
// ReadOwner is the same owner.
func TestAnOwnerTokenReadsBackIntoItsTwoHalves(t *testing.T) {
	for _, c := range []struct {
		token, name, domain string
	}{
		{"@anna", "anna", ""},
		{"@anna@example.com", "anna", "example.com"},
		{"@a.b-c_d+e@sub.example.co.uk", "a.b-c_d+e", "sub.example.co.uk"},
	} {
		owner, err := ReadOwner(c.token)
		if err != nil {
			t.Fatalf("%s is an owner Parse takes and ReadOwner refused: %v", c.token, err)
		}
		if owner.Text != c.token || owner.Name != c.name || owner.Domain != c.domain {
			t.Errorf("ReadOwner(%q) = %+v, want the name %q and the domain %q", c.token, owner, c.name, c.domain)
		}
		if owner.Start != 0 || owner.End != 0 {
			t.Errorf("ReadOwner(%q) named a span %d to %d, and a token with no source behind it has none", c.token, owner.Start, owner.End)
		}
	}
	if _, err := ReadOwner("anna"); err == nil {
		t.Error("a word with no @ in front of it was read as an owner")
	}
}
