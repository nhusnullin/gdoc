package generate

import "google.golang.org/api/googleapi"

// googleContentType names the upload's own type, which is separate from the
// mimeType asked for on the file: the first says what is being sent, the second
// says what Drive should convert it into.
func googleContentType(mime string) googleapi.MediaOption {
	return googleapi.ContentType(mime)
}
