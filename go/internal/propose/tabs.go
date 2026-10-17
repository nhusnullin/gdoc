// This file is the one refusal a document with more than one tab gets from
// every write that names a range: propose's own two, and annotate's, which
// calls MoreThanOneTab for the same reason.

package propose

import (
	"errors"
	"fmt"
)

// ErrMoreThanOneTab is what errors.Is matches on every refusal of a document
// with more than one tab. A write that names a range means nothing without
// saying which tab the range is in, and refusing is the honest answer until a
// milestone decides how a proposal names a tab.
//
// It is a value a caller matches rather than a sentence it reads, because
// cmd/gdoc names a stop code for the presence record from it and a code read
// out of English would follow the words wherever somebody reworded them.
// TestMoreThanOneTabIsATypedRefusal is the pin, in this package and in
// annotate.
var ErrMoreThanOneTab = errors.New("the document has more than one tab")

// MoreThanOneTab is that refusal with the sentence the caller wants on it. The
// sentence names the count and what the write was, because the person reading
// it has to know which of their documents was refused and why; the match is
// ErrMoreThanOneTab underneath, which the sentence never changes.
func MoreThanOneTab(format string, a ...any) error {
	return tabRefusal{msg: fmt.Sprintf(format, a...)}
}

// tabRefusal is the sentence and nothing else. Unwrap, rather than a %w in the
// sentence, is why the words a person reads stayed exactly what they were.
type tabRefusal struct{ msg string }

func (e tabRefusal) Error() string { return e.msg }
func (e tabRefusal) Unwrap() error { return ErrMoreThanOneTab }
