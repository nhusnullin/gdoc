// This file is the one read whose answer is neither a document nor JSON: the
// bytes of one file in Drive. internal/asset is what asks for it, and the
// house logo is the one file there is to ask about today.

package gapi

import (
	"context"
	"fmt"
	"net/url"
)

// assetURL is the whole of that read, spelled exactly as guard.isAssetRead
// reads it: files.get on one id, `alt=media`, `supportsAllDrives=true`, and no
// third parameter. The guard spells the same literals on its own side, so a
// parameter added here that the guard has not agreed to is a refusal rather
// than a wider read.
//
// `alt=media` is what turns files.get from a metadata read into the file's
// bytes. `supportsAllDrives=true` is there because the file sits in a Shared
// Drive, and Drive answers 404 on one of those without it.
const assetURL = "https://www.googleapis.com/drive/v3/files/%s?alt=media&supportsAllDrives=true"

// FileBytes reads one Drive file whole and returns at most limit bytes. It is
// what satisfies asset.Drive, and the request needs guard.AllowAsset on that
// id, which only the logo fetch in cmd/gdoc grants, so every other policy in
// this binary is refused by the guard before anything leaves.
//
// The id goes into the path, so an id carrying anything a path segment cannot
// is refused here rather than escaped into a request for a file somebody else
// meant. The guard would refuse it too; refusing it by name is the better
// message.
func (s *Session) FileBytes(ctx context.Context, fileID string, limit int64) ([]byte, error) {
	if fileID == "" || fileID != url.PathEscape(fileID) {
		return nil, fmt.Errorf("%q is not a Drive file id, so nothing was read", fileID)
	}
	return s.get(ctx, fmt.Sprintf(assetURL, fileID), "*/*", limit)
}
