// Package drive is the Drive file calls gdoc makes on a file it created itself:
// the two URLs, and the one operation that has to be confirmed before it may be
// reported as having happened.
//
// This comment holds why the package refuses what it refuses. What it reports
// is in the code beside it.
//
// # The trash is believed on its confirming read, never on the PATCH
//
// Trash is three steps: PATCH the file, read it again, and believe the read.
// Drive answering the PATCH says the request was taken, and the question the
// caller has is whether the file is gone, which only the read answers. So a
// PATCH that failed claims nothing at all about where the file is, a
// confirmation that could not be read is a failure, and Drive saying the file
// is not trashed is a failure rather than a success.
// TestTrashPatchesThenConfirms, TestAFailedPatchClaimsNothingAboutWhereTheFileIs,
// TestAConfirmationThatCouldNotBeReadIsAFailure and
// TestDriveSayingItIsNotTrashedIsAFailureRatherThanASuccess are the four pins,
// one per answer.
//
// # It exists because two callers believe that rule
//
// internal/probe puts its throwaway document away after asking its question,
// and internal/publish takes its document back when the pairing could not be
// recorded. Two copies of those three steps are two chances for one of them to
// start reporting a document as gone that is still there, so there is one copy
// and both callers ask it.
//
// The policy around the answer is each caller's, and they differ on purpose.
// The probe carries a warning and answers its question anyway, because a
// document left in the folder does not make its measurement wrong. A publish
// reports a rollback that did not hold with the live id and the steps to take,
// because there the document is the thing that must not be left behind.
package drive

import (
	"context"
	"errors"
	"fmt"
)

// Trasher is what this package needs of a session: the PATCH Drive spells
// trashing as, and the GET that confirms it. A *gapi.Session satisfies it, and
// so does a fake in a test. Naming the interface here keeps net/http out of
// this room, which is what the boundary test asks of every package but the four
// that build requests.
type Trasher interface {
	GetJSON(ctx context.Context, rawURL string, into any) error
	PatchJSON(ctx context.Context, rawURL string, body any, into any) error
}

// FileURL is files.update, which is how Drive spells trashing.
func FileURL(id string) string {
	return "https://www.googleapis.com/drive/v3/files/" + id + "?supportsAllDrives=true"
}

// TrashedURL is files.get asking the one question the confirmation has.
func TrashedURL(id string) string {
	return "https://www.googleapis.com/drive/v3/files/" + id + "?fields=trashed&supportsAllDrives=true"
}

// Trash puts the file away and confirms it went. It answers nil only when Drive
// itself says the file is in the trash, so a PATCH that was accepted and a
// read-back that says otherwise is a failure here rather than a success the
// caller has to notice.
//
// Each of the three failures reads differently on purpose. A failed PATCH
// claims nothing about where the file is, because it holds three things and only
// two of them are Drive turning the request down: a guard refusal never left the
// machine and a 4xx is a refusal, but a 5xx or a dropped connection is written
// and may have been applied, which is the case gapi's own mark cannot resolve
// either. An unread confirmation says nothing either way. A read-back reporting
// the file as not trashed is Drive contradicting the write, and it is the one
// failure here that knows where the file is. A caller prefixes what the file is,
// because this package does not know.
func Trash(ctx context.Context, s Trasher, id string) error {
	if err := s.PatchJSON(ctx, FileURL(id), map[string]any{"trashed": true}, nil); err != nil {
		return fmt.Errorf("the trash request failed: %w", err)
	}
	var answer struct {
		Trashed bool `json:"trashed"`
	}
	if err := s.GetJSON(ctx, TrashedURL(id), &answer); err != nil {
		return fmt.Errorf("Drive took the trash and could not be asked to confirm it: %w", err)
	}
	if !answer.Trashed {
		return errors.New("Drive took the trash and still reports it as not trashed")
	}
	return nil
}
