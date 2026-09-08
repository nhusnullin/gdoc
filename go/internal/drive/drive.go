// Package drive is the Drive file calls gdoc makes on a file it created itself:
// the two URLs, and the one operation that has to be confirmed before it may be
// reported as having happened.
//
// It exists because two callers do that operation. The probe puts its throwaway
// document away after asking its question, and a publish that could not record
// the pairing takes its document back. Both are the same three steps, PATCH the
// file, read it again, and believe the read rather than the PATCH, and a second
// copy of those steps is a second chance for the two to disagree about whether
// an unconfirmed trash counts as a trash. The policy around them is each
// caller's: the probe carries a warning and answers its question anyway, and a
// publish reports a rollback that did not hold with the live id and the steps.
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
// Each of the three failures reads differently on purpose. A refused PATCH left
// the file where it was; an unread confirmation says nothing either way; and a
// read-back reporting the file as not trashed is Drive contradicting the write.
// A caller prefixes what the file is, because this package does not know.
func Trash(ctx context.Context, s Trasher, id string) error {
	if err := s.PatchJSON(ctx, FileURL(id), map[string]any{"trashed": true}, nil); err != nil {
		return fmt.Errorf("Drive refused to trash it, so it is still where it was: %w", err)
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
