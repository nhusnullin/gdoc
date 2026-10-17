// This file is the one reader of the typed failures the packages raise: a
// failed run's stop code, and the line of the note the cause sits on where it
// has one.
//
// It is here rather than in internal/presence because the record may not know
// what a Drive answer or a guard refusal is: presence lifts the word and
// judges nothing. And it is one function rather than a method on each package's
// error, because the seven words are the record's vocabulary and not the
// packages'.

package main

import (
	"context"
	"errors"
	"net"

	"gdoc/internal/auth"
	"gdoc/internal/body"
	"gdoc/internal/emit"
	"gdoc/internal/gapi"
	"gdoc/internal/guard"
	"gdoc/internal/propose"
)

// The three statuses a person meets, as literals, two of them a word of their
// own and the third read and left as other. cmd/gdoc may not import
// net/http, which is the guard's own room and four others':
// boundary's TestNetHTTPStaysInItsRooms. internal/gapi puts the status on the
// error it raises, and these are the numbers that answer which word it is.
const (
	statusUnauthorized = 401
	statusForbidden    = 403
	statusNotFound     = 404
)

// failed is a refusal as the envelope: the sentence a person reads, and the
// word the record reads, from one call. Every command that answers with
// nothing but the error it was handed goes through it.
func failed(err error) emit.Result {
	return withStop(emit.Result{OK: false, Error: err.Error()}, err)
}

// withStop is the same for a refusal that carries its data and its warnings
// too, because what a run earned before it failed is still the caller's to
// read.
func withStop(r emit.Result, err error) emit.Result {
	r.Stop, r.StopLine = stopFor(err)
	return r
}

// stopFor is the word and the line for one failure.
//
// The order is read before the codes are: a guard refusal and a Drive status
// are looked for before the network, because net/http wraps a transport error
// in a *url.Error, which is itself a net.Error, so a refusal read the other
// way round would be written down as a timeout nobody had.
// TestAGuardRefusalIsRefused is the pin.
//
// Nothing here reads a sentence. Every match is a value a package raises, so a
// milestone that rewords a refusal moves no code: TestAStopIsOneOfTheSevenCodesAndNeverASentence.
func stopFor(err error) (emit.StopCode, int) {
	if err == nil {
		return "", 0
	}
	var picture *body.PictureError
	if errors.As(err, &picture) {
		return emit.StopSVGPicture, picture.Line
	}
	if errors.Is(err, propose.ErrMoreThanOneTab) {
		return emit.StopMultiTab, 0
	}
	if errors.Is(err, guard.ErrRefused) {
		return emit.StopRefused, 0
	}
	if errors.Is(err, auth.ErrNoToken) || errors.Is(err, auth.ErrGrantRefused) {
		return emit.StopAuth, 0
	}
	var status *gapi.StatusError
	if errors.As(err, &status) {
		switch status.Status {
		case statusNotFound:
			return emit.StopNotFound, 0
		case statusUnauthorized:
			return emit.StopAuth, 0
		case statusForbidden:
			// A 403 is not the token. Google answers it for a document this
			// account was never shared, for a comment Drive will not let even
			// the owner delete, and GitHub answers it for the rate limit on a
			// read that carries no credential at all. None of those is a
			// sign-in to do again, and auth in the record would send a person
			// to do one: TestAForbiddenIsNotTheTokenRefused.
			return emit.StopOther, 0
		}
		// Every other status is Google answering something none of the six
		// words name, and a guess here would be a word a reader of the record
		// cannot check: TestANotFoundIsNotFound.
		return emit.StopOther, 0
	}
	// A deadline is the one network failure that is also a context error, and
	// it is read before the interface so a cancelled run, which is the person
	// stopping gdoc rather than a wire that failed, stays other.
	if errors.Is(err, context.Canceled) {
		return emit.StopOther, 0
	}
	var wire net.Error
	if errors.As(err, &wire) || errors.Is(err, context.DeadlineExceeded) {
		return emit.StopNetwork, 0
	}
	return emit.StopOther, 0
}
