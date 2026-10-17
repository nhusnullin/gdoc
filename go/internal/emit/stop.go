// This file is the stop code: what a failed run says went wrong, in one of
// seven fixed words rather than in the sentence a person reads.

package emit

// StopCode names why a run failed, as a fixed word. It exists because the
// presence record carries why a run stopped and must never carry a sentence:
// a sentence holds a path, a title or a quote, and the record carries none of
// those. The seven words are the whole set, and a failure that is none of them
// is other rather than a new word: TestTheSevenCodesAreLiterals.
//
// The caller that sets it is cmd/gdoc, which reads the typed failures the
// packages raise through errors.As. internal/presence lifts the word and
// judges nothing, so it imports nothing new.
type StopCode string

// The seven codes the spec names, in its order.
const (
	// StopSVGPicture is a picture the house style cannot embed, which is an
	// SVG in every case met so far. internal/body raises it with the line of
	// the note it sits on.
	StopSVGPicture StopCode = "svg_picture"
	// StopMultiTab is a write that names a range in a document with more than
	// one tab, refused by propose and by annotate.
	StopMultiTab StopCode = "multi_tab"
	// StopNotFound is Drive or Docs answering 404 for the id a command named.
	StopNotFound StopCode = "not_found"
	// StopAuth is the token refused: no sign-in, an expired grant, or a scope
	// the account does not hold.
	StopAuth StopCode = "auth"
	// StopNetwork is a request that never got an answer: a dial, a handshake
	// or a deadline.
	StopNetwork StopCode = "network"
	// StopRefused is the guard refusing to make the request at all.
	StopRefused StopCode = "refused"
	// StopOther is every failure the six above do not name. It is the honest
	// answer, not a gap: a reader of the record learns that the run failed and
	// reads the command's own object for why.
	StopOther StopCode = "other"
)

// stopCodes is the set, declared once so Known and StopCodes cannot disagree.
var stopCodes = [...]StopCode{
	StopSVGPicture, StopMultiTab, StopNotFound, StopAuth, StopNetwork, StopRefused, StopOther,
}

// StopCodes is the whole set in the spec's order, as a copy: a caller that
// edits what it is handed does not edit the next caller's set.
func StopCodes() []StopCode {
	out := make([]StopCode, len(stopCodes))
	copy(out, stopCodes[:])
	return out
}

// Known says c is one of the seven. Anything else, an empty string and a
// sentence included, is not a stop code.
func (c StopCode) Known() bool {
	for _, k := range stopCodes {
		if c == k {
			return true
		}
	}
	return false
}
