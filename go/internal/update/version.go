package update

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a release tag read as three numbers and an optional pre-release
// word: v2.1.3, or v2.0.0-rc1.
//
// It is not semver. There is no build metadata, no comparison of dot-separated
// pre-release fields, and no range syntax, because gdoc compares two tags and
// nothing else. Three integers and a comparison is a page of code with no
// dependency, and the module list stays at three.
type Version struct {
	Major int
	Minor int
	Patch int

	// Pre is the text after the dash, without it, and empty for a release.
	// It exists because a milestone is accepted by cutting v2.0.0-rc1 and
	// updating from it to v2.0.0-rc2 on a real machine.
	Pre string
}

// Parse reads a tag. Everything about the shape is refused by name, because
// the one caller that hands it something malformed is a listing from GitHub
// with a moving tag in it, and a version nobody can read is not a version to
// compare against a person's binary.
func Parse(s string) (Version, error) {
	if s == "" {
		return Version{}, fmt.Errorf("the version is empty")
	}
	rest, ok := strings.CutPrefix(s, "v")
	if !ok {
		return Version{}, fmt.Errorf("the version %q opens with a v and this one does not", s)
	}
	pre := ""
	if numbers, tail, cut := strings.Cut(rest, "-"); cut {
		if err := checkPre(s, tail); err != nil {
			return Version{}, err
		}
		rest, pre = numbers, tail
	}
	fields := strings.Split(rest, ".")
	if len(fields) != 3 {
		return Version{}, fmt.Errorf("the version %q is three numbers after the v, and this one has %d", s, len(fields))
	}
	out := Version{Pre: pre}
	into := []*int{&out.Major, &out.Minor, &out.Patch}
	for i, f := range fields {
		n, err := number(s, f)
		if err != nil {
			return Version{}, err
		}
		*into[i] = n
	}
	return out, nil
}

// number reads one field of the three. strconv.Atoi alone takes "+2", " 2" and
// "-1", each of which would make two tags that are not the same text compare
// as the same version.
func number(whole, field string) (int, error) {
	if field == "" {
		return 0, fmt.Errorf("the version %q is three numbers after the v, and one of them is empty", whole)
	}
	for _, r := range field {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("the version %q is three numbers after the v, and %q is not one", whole, field)
		}
	}
	if len(field) > 1 && field[0] == '0' {
		return 0, fmt.Errorf("the version %q carries a leading zero in %q, and no tag gdoc cuts does", whole, field)
	}
	n, err := strconv.Atoi(field)
	if err != nil {
		return 0, fmt.Errorf("the version %q holds %q, which is not a number: %w", whole, field, err)
	}
	return n, nil
}

// checkPre holds the pre-release word to letters, numbers and dots, so that a
// tag with a space or an underscore in it is refused here rather than turning
// into an asset name nothing on the releases page carries.
func checkPre(whole, pre string) error {
	if pre == "" {
		return fmt.Errorf("the version %q ends in a dash with no pre-release word after it", whole)
	}
	for _, r := range pre {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '.':
		default:
			return fmt.Errorf("the pre-release word %q in %q holds %q, and a pre-release is letters, numbers and dots", pre, whole, r)
		}
	}
	return nil
}

// String prints the tag back, and prints nothing at all for the zero Version,
// which is how a channel that holds no release is said.
func (v Version) String() string {
	if v.IsZero() {
		return ""
	}
	s := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		s += "-" + v.Pre
	}
	return s
}

// IsZero reports the value that means nothing was found: a channel with no
// release in it, and a listing gdoc could not read.
//
// v0.0.0 parses to that same value, and nothing distinguishes the two. gdoc's
// first release is v2.0.0 and tags only go up, so the one tag that could be
// confused with nothing found is a tag that will never exist. A flag inside
// the struct to tell them apart would make every Version literal a test
// writes mean something other than what it says.
// TestTheZeroVersionIsHowNothingFoundIsSaid pins it.
func (v Version) IsZero() bool { return v == Version{} }

// IsStable reports whether the tag is one a person cut by hand. `make tag`
// refuses anything but x.y.0, and the nightly cuts x.y.(z+1), so the patch
// number is the whole of the difference between the two channels.
func (v Version) IsStable() bool { return v.Patch == 0 }

// Compare orders a against b: -1, 0 or 1.
//
// A pre-release sorts below the release it names, which is the one rule this
// takes from semver and the reason an rc can be updated from. Two pre-releases
// are compared as text, so rc10 sorts below rc2. gdoc cuts rc1 and rc2 by hand
// in one sitting, and reading the digits out would be a semver implementation
// arriving one function at a time.
func Compare(a, b Version) int {
	for _, pair := range [][2]int{{a.Major, b.Major}, {a.Minor, b.Minor}, {a.Patch, b.Patch}} {
		if pair[0] != pair[1] {
			return sign(pair[0] - pair[1])
		}
	}
	switch {
	case a.Pre == b.Pre:
		return 0
	case a.Pre == "":
		return 1
	case b.Pre == "":
		return -1
	}
	return sign(strings.Compare(a.Pre, b.Pre))
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}
