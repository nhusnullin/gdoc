// The notation itself: one task item's line read apart into words, an owner
// and a day. The package comment in doc.go holds why each rule is the rule.

package tasks

import (
	"fmt"
	"strings"
	"time"
)

// DomainKey and DomainFlag are the two lines a person writes a domain on, named
// here because the refusal of a bare owner quotes both. internal/settings owns
// the reading of either, and this package owns what the value may be.
const (
	DomainKey  = "domain"
	DomainFlag = "--domain"
)

// DayLayout is the one date form a task takes, the calendar day as the ISO
// order writes it. Nothing here reads a time of day: a due date is a day, and
// what a chip makes of it is internal/publish's, which asks this package for
// the layout rather than spelling a second copy of it.
// TestADateThatIsNotADayIsRefused and TestADateIsNoonUTCOnItsDay.
const DayLayout = "2006-01-02"

// BallotEmpty and BallotCrossed are what a task item opens with where there is
// no box to tick: the character and the space after it. internal/body writes
// one of the two into the docx, because a file opened in Word has no box, and
// internal/publish reads the same two off the front of the paragraph that came
// back, because that is what it takes away when the document gets a real box.
// Two readers and one writer, so the three cannot drift.
const (
	BallotEmpty   = "\u2610 "
	BallotCrossed = "\u2612 "
)

// nameRunes are the characters an owner's name may hold beside letters and
// digits. They are the ones an address's local part commonly carries, and a
// dot may not open or close the name or stand beside another.
const nameRunes = "._-+"

// Task is one task list item's line, read apart. Words is the line with the
// tokens and the spaces in front of them taken off, which is what the item
// says; Owner and Due are what it was tagged with, each absent where nobody
// wrote one.
type Task struct {
	Words string
	Owner Owner
	Due   Due
}

// Owner is the address a task names, as it was written. Name and Domain are
// the two halves, and Domain is the empty string for a bare name, which
// Resolve finishes from the setting. Start and End are the token's byte span
// in the source handed to Parse, so a caller can put a chip where the token
// stood.
type Owner struct {
	Text   string
	Name   string
	Domain string
	Start  int
	End    int
}

// Present answers whether anybody wrote an owner at all.
func (o Owner) Present() bool { return o.Text != "" }

// Resolve answers the whole address. A name somebody wrote out in full is
// answered as written, whatever the domain in effect is, and a bare name takes
// the domain. A bare name with no domain in effect is refused naming the
// setting and the flag, because a guess at somebody's address is the one thing
// this package may never make.
//
// An owner nobody wrote resolves to the empty address and no refusal, so a
// caller may ask it of every task.
// TestABareNameTakesTheDomain, TestABareNameWithNoDomainIsRefusedNamingTheSetting
// and TestAFullAddressIgnoresTheDomain.
func (o Owner) Resolve(domain string) (string, error) {
	if !o.Present() {
		return "", nil
	}
	if o.Domain != "" {
		return o.Name + "@" + o.Domain, nil
	}
	if domain == "" {
		return "", fmt.Errorf("%s names a person and no domain, and no %s setting is in effect; write %s: in settings.yaml, pass %s, or write the whole address in the note", o.Text, DomainKey, DomainKey, DomainFlag)
	}
	if err := CheckDomain(domain); err != nil {
		return "", err
	}
	return o.Name + "@" + domain, nil
}

// Due is the day a task is due, as it was written, with its byte span in the
// source. It is a day and never a time: Text is the calendar day, and what a
// chip makes of it is the caller's.
type Due struct {
	Text  string
	Start int
	End   int
}

// Present answers whether anybody wrote a date at all.
func (d Due) Present() bool { return d.Text != "" }

// Parse reads one task list item's line: the words, at most one owner and at
// most one day, each only where it ends the line, in either order. What it
// cannot read it refuses naming the word, because a token gdoc half understood
// is a chip on the wrong person or the wrong day.
//
// source is the item's own text with no list marker and no ballot box in front
// of it. It is the markdown source rather than the rendered inlines, because
// GFM's linkify turns anna@example.org into an autolink and the address would
// arrive in pieces.
func Parse(source string) (Task, error) {
	rest := strings.TrimRight(source, " \t")
	var task Task
	for rest != "" {
		start := strings.LastIndexAny(rest, " \t") + 1
		word := rest[start:]
		switch {
		case strings.HasPrefix(word, "@"):
			if task.Owner.Present() {
				return Task{}, fmt.Errorf("a task takes at most one owner, and this one names %s and %s", word, task.Owner.Text)
			}
			owner, err := ReadOwner(word)
			if err != nil {
				return Task{}, err
			}
			owner.Start, owner.End = start, len(rest)
			task.Owner = owner
		case looksLikeADay(word):
			if task.Due.Present() {
				return Task{}, fmt.Errorf("a task takes at most one date, and this one names %s and %s", word, task.Due.Text)
			}
			if _, err := time.Parse(DayLayout, word); err != nil {
				return Task{}, fmt.Errorf("%s is no day on the calendar; a task's date is a day, written %s", word, DayLayout)
			}
			task.Due = Due{Text: word, Start: start, End: len(rest)}
		default:
			task.Words = rest
			return task, nil
		}
		rest = strings.TrimRight(rest[:start], " \t")
	}
	task.Words = rest
	return task, nil
}

// looksLikeADay answers whether a word is shaped like the notation's date,
// ten characters with digits and two hyphens where the layout has them. A word
// of this shape is a date somebody meant, so one that is no calendar day is
// refused rather than read as a word; anything of another shape is a word.
func looksLikeADay(word string) bool {
	if len(word) != len(DayLayout) {
		return false
	}
	for i := 0; i < len(word); i++ {
		digit := word[i] >= '0' && word[i] <= '9'
		if i == 4 || i == 7 {
			if word[i] != '-' {
				return false
			}
			continue
		}
		if !digit {
			return false
		}
	}
	return true
}

// ReadOwner reads one owner token as a note wrote it, "@name" or
// "@name@domain", into the two halves a caller resolves. The name is refused by
// the rule below, and a domain somebody wrote out goes through CheckDomain, so
// @a@b is refused for the one reason the setting would be: it is no domain.
//
// Parse asks it of a word that ends an item's line, and cmd/gdoc asks it of a
// token internal/body already read, because a publish resolves every owner
// against the domain in effect before it sends anything. The span is the
// caller's to fill in: a token with no source behind it has none.
// TestAnOwnerThatIsNotAnAddressIsRefused and
// TestAnOwnerTokenReadsBackIntoItsTwoHalves.
func ReadOwner(word string) (Owner, error) {
	if !strings.HasPrefix(word, "@") {
		return Owner{}, fmt.Errorf("%q is no owner: a task's owner opens with @, as @anna does", word)
	}
	name, domain, written := strings.Cut(strings.TrimPrefix(word, "@"), "@")
	if err := checkName(name); err != nil {
		return Owner{}, fmt.Errorf("%s is no address: %w", word, err)
	}
	if written {
		if err := CheckDomain(domain); err != nil {
			return Owner{}, fmt.Errorf("%s is no address: %w", word, err)
		}
	}
	return Owner{Text: word, Name: name, Domain: domain}, nil
}

// checkName refuses what an address's local part may not be: nothing at all, a
// character outside the letters, digits and the four marks, a dot at either
// end, and two dots together.
func checkName(name string) error {
	if name == "" {
		return fmt.Errorf("it names nobody in front of the domain")
	}
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") || strings.Contains(name, "..") {
		return fmt.Errorf("a name takes no dot at either end and no two dots together")
	}
	for _, r := range name {
		if isLetterOrDigit(r) || strings.ContainsRune(nameRunes, r) {
			continue
		}
		return fmt.Errorf("a name takes letters, digits and %s, and %q is none of them", nameRunes, r)
	}
	return nil
}

// CheckDomain is the one rule about what a domain may be, asked by this package
// of an address somebody wrote out and by internal/settings of the file and the
// flag. It takes letters, digits, hyphens and dots, with at least one dot, and
// refuses everything else naming the value, because a domain gdoc half
// understood is a chip on nobody.
// TestADomainIsLettersDigitsHyphensAndDots.
func CheckDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("a domain is letters, digits, hyphens and dots with at least one dot, and this one is empty")
	}
	for _, r := range domain {
		if isLetterOrDigit(r) || r == '-' || r == '.' {
			continue
		}
		return fmt.Errorf("%q is no domain: it takes letters, digits, hyphens and dots, and %q is none of them", domain, r)
	}
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return fmt.Errorf("%q is no domain: it needs at least one dot, as example.com has", domain)
	}
	for _, label := range labels {
		if label == "" {
			return fmt.Errorf("%q is no domain: it has a dot with nothing on one side of it", domain)
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return fmt.Errorf("%q is no domain: %q opens or closes with a hyphen", domain, label)
		}
	}
	return nil
}

// isLetterOrDigit is the ASCII letters and digits, which is what an address
// gdoc sends may hold. A name in another script is refused by checkName rather
// than sent to a chip that would be nobody's.
func isLetterOrDigit(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
