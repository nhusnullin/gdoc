// This file is the query surface, decided once and in one place.
//
// A Google request carries two kinds of query parameter. The method's own,
// listed in the API's discovery document
// (https://www.googleapis.com/discovery/v1/apis/drive/v3/rest), and the system
// parameters every Google API accepts
// (https://docs.cloud.google.com/apis/docs/system-parameters). The second set
// is the one that keeps growing another spelling for the same thing: `fields`
// is also `$fields` and also the header X-Goog-FieldMask, `uploadType` has the
// sibling `upload_protocol`, `key` is also `$key` and the header
// X-Goog-Api-Key, and an empty field mask means every field rather than none.
//
// So the rule here is an allowlist, the shape every other rule in this guard
// already has. A request carries only the parameters named below, and anything
// else is refused. A list of blocked spellings needs a new patch each time
// somebody finds another one, and every miss is a live hole until then. An
// allowlist is wrong in the direction of a refusal the next milestone widens on
// purpose. The header half of the same rule lives in checkWireMatchesJudgment.
//
// Some system parameters are left out on purpose, and the reason is the same
// for all of them: `access_token`, `oauth_token` and `key` put a second
// credential on a request gdoc authenticates itself, `callback` turns the
// answer into JSONP, and `prettyPrint`, `quotaUser` and `$.xgafv` change
// nothing gdoc reads. None of them appears in a call gdoc builds.
package guard

import (
	"net/url"
	"strings"
)

// noParams is the empty allowlist: this call carries no query at all.
var noParams = map[string]bool{}

// docsReadParams are the parameters a Docs read may carry. suggestionsViewMode
// is how gdoc sees pending suggestions, which Drive's export renders as though
// nothing had been suggested. commentsViewMode is how it reads comment threads
// with real character ranges, which is M2's whole reason for the Docs read; it
// needs includeTabsContent, so the two travel together.
//
// These are the three documents.get parameters
// (https://developers.google.com/workspace/docs/api/reference/rest/v1/documents/get)
// plus the two system parameters gdoc sets, so a later milestone adds nothing
// here.
var docsReadParams = map[string]bool{
	"alt":                 true,
	"fields":              true,
	"suggestionsViewMode": true,
	"includeTabsContent":  true,
	"commentsViewMode":    true,
}

// The Drive reads, one allowlist per call shape. The path is only half of what
// a GET asks for: the path names one file and the level says read, and the
// query decides how much of that file comes back.
//
// One list across all of them was wrong in both directions. It put paging on a
// metadata read and an export format on a comment listing, neither of which is
// a call Drive has, and it put `alt` on the bare files.get, where `alt=media`
// stops being a metadata read and hands back the file's bytes. Each set below
// is that method's own parameters, from the Drive v3 reference, so a request
// carries only what its own shape needs.

// driveGetParams are the parameters files.get may carry. No `alt`: a metadata
// read asks for metadata, and gdoc reads bytes through /export, where
// checkExportMime decides the format. What else is left out is left out on
// purpose: `includePermissionsForView` is the permission surface /permissions
// and checkFields already refuse, and `acknowledgeAbuse` overrides a warning
// gdoc has no business overriding.
var driveGetParams = map[string]bool{
	"fields":            true, // narrowed further by checkFields
	"supportsAllDrives": true,
}

// driveExportParams are the parameters files.export may carry. This is the one
// read that asks for bytes, so `alt` belongs here and nowhere else. No `fields`:
// the answer is a file, and a field mask has nothing to select in it. No
// `supportsAllDrives` either: files.export does not define it, measured against
// the live Drive v3 discovery document on 2026-09-06, and an allowlist naming a
// parameter the method does not have permits a request nothing should send.
var driveExportParams = map[string]bool{
	"alt":      true,
	"mimeType": true, // which export format, narrowed further by checkExportMime
}

// driveCommentListParams are the parameters comments.list may carry.
// startModifiedTime is the --since cursor, activity after this instant, and it
// is on comments.list alone.
var driveCommentListParams = map[string]bool{
	"fields":            true,
	"pageSize":          true, // comments come back a page at a time
	"pageToken":         true,
	"includeDeleted":    true,
	"startModifiedTime": true,
}

// driveReplyListParams are the parameters replies.list may carry: the comment
// list's paging without the cursor, which replies.list does not define.
var driveReplyListParams = map[string]bool{
	"fields":         true,
	"pageSize":       true,
	"pageToken":      true,
	"includeDeleted": true,
}

// driveCommentGetParams are the parameters comments.get and replies.get may
// carry. One comment is not a page of them, so no paging and no cursor.
var driveCommentGetParams = map[string]bool{
	"fields":         true,
	"includeDeleted": true,
}

// driveWriteParams are the parameters a comment write or an in-place patch may
// carry. `fields` picks what the answer says about the thing just written.
var driveWriteParams = map[string]bool{
	"alt":               true,
	"fields":            true,
	"supportsAllDrives": true,
}

// driveCreateParams are the parameters a create may carry. The two upload
// parameters are the same choice spelled two ways, and checkUploadShape reads
// both.
var driveCreateParams = map[string]bool{
	"alt":               true,
	"fields":            true,
	"supportsAllDrives": true,
	"uploadType":        true,
	"upload_protocol":   true,
}

// driveCopyParams are the parameters files.copy may carry, and they are three.
// copyComments is the one the 2026-09-09 measurement turned on, and it is what
// carries the source's threads and its pending suggestions into the duplicate;
// supportsAllDrives is what a folder on a shared drive needs, without which
// Drive answers a flat 404; and `fields` narrows the answer the new id is read
// out of, checkFields narrowing it further.
//
// What is left out is left out on purpose. `alt` has no place here, because a
// copy answers with metadata and `alt=media` would ask for bytes the guard then
// searches for an id in. The two upload parameters are not files.copy's: it
// uploads nothing, so a shape naming one is a request the guard would be
// reading one way and Drive another. `ocr` runs somebody's document through
// text recognition, `keepRevisionForever` pins a revision in the owner's
// storage quota, `ignoreDefaultVisibility` and `enforceSingleParent` change who
// reaches the duplicate, and `includePermissionsForView` is the permission
// surface /permissions and checkFields already refuse.
var driveCopyParams = map[string]bool{
	"copyComments":      true,
	"supportsAllDrives": true,
	"fields":            true, // narrowed further by checkFields
}

// checkQuery refuses a query carrying anything the allowlist does not name, and
// then checks the values of the parameters that decide what comes back.
//
// The query is parsed here rather than through u.Query(), which drops a pair it
// cannot read and returns the rest. A query the guard cannot read whole is a
// query it cannot judge, and Google may still read the part that was dropped.
//
// One value per parameter, always. gdoc repeats none of them, and a repeated
// parameter is a request whose meaning depends on which copy the server takes.
func checkQuery(u *url.URL, allowed map[string]bool) error {
	vals, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return refuse("the query %q cannot be read, so it cannot be judged: %v", u.RawQuery, err)
	}
	for name, list := range vals {
		if !allowed[name] {
			return refuse("the query parameter %q is not one this call carries", name)
		}
		if len(list) != 1 {
			return refuse("the query parameter %q is given %d times, and which copy the server takes is not decided here", name, len(list))
		}
		if err := checkParamValue(name, list[0]); err != nil {
			return err
		}
	}
	return checkUploadShape(vals)
}

// altValues are the response formats gdoc reads: JSON metadata, and the bytes
// of an export.
var altValues = map[string]bool{"json": true, "media": true}

func checkParamValue(name, v string) error {
	switch name {
	case "fields":
		return checkFields(v)
	case "alt":
		if !altValues[v] {
			return refuse("alt=%q is not a response format gdoc reads", v)
		}
	case "mimeType":
		return checkExportMime(v)
	case "includeTabsContent":
		// SPEC.md: "Always read with includeTabsContent=true. Reading without
		// it silently sees one tab." The guard decides the value here, and it
		// does not require the parameter to be present. Two reasons, and both
		// matter.
		//
		// A read that says `false` is asking for the blind read on purpose, and
		// nothing gdoc does wants that, so it is refused like any other value
		// the guard has not decided about. But requiring the parameter would
		// make the guard order every Docs read to fetch every tab, including a
		// metadata-only read such as `fields=documentId,title`, which needs no
		// tab content at all. That is a call-site decision about what to ask
		// for, and it belongs where the call is built, in M2's reader, beside
		// the multi-tab check the same read feeds.
		//
		// So the limit is worth naming: a Docs read that omits this reaches
		// Drive, sees one tab, and the guard does not stop it.
		if v != "true" {
			return refuse("includeTabsContent=%q is not a read gdoc makes: without a plain true the answer covers one tab and says nothing about the rest", v)
		}
	}
	return nil
}

// checkExportMime refuses the one export format SPEC.md's Never list names:
// "Never export a PDF. Nail downloads it from the browser." mimeType decides
// what /export returns, so this is where that rule can be applied.
//
// The comparison folds case and drops anything after the `;`, because a media
// type is case-insensitive and its parameters do not change which type it is.
// A `;` makes url.ParseQuery refuse the whole query today, so that half is
// belt and braces rather than the only stop.
func checkExportMime(v string) error {
	mime, _, _ := strings.Cut(v, ";")
	if strings.EqualFold(strings.TrimSpace(mime), "application/pdf") {
		return refuse("mimeType=%q asks for a PDF export, and gdoc never exports one", v)
	}
	return nil
}

// uploadShapes are the create shapes the guard carries, under either spelling.
// `multipart` puts the metadata where the parent check can read it: the body
// opens with the metadata part. No upload parameter at all is a plain JSON
// create, which the parent check reads directly.
//
// The first refused shape is the one that makes the whole body the file's
// content: `uploadType=media`, and the same thing under its newer name,
// `upload_protocol=raw`. There `{"parents":["FOLDER1"]}` is bytes to Drive and
// metadata to the parent check, so the file lands unparented and the guard then
// learns its id at full level. Nothing here may learn an id from a create it
// could not verify.
//
// `resumable` is the second, and dropping it is finding 4 of the M1 review. A
// resumable create is two legs: this one, and a PUT to a location URL carrying
// `upload_id`. The guard carries neither the method nor the parameter, so the
// shape could never finish, and a half-permitted route reads as a working one
// to whoever tries it next. The milestone that needs resumable uploads adds
// both legs together or neither.
//
// M6 made `multipart` reachable rather than merely named. Until then the
// transport's own parse refused every multipart body, so this entry permitted a
// shape nothing could finish; transport.checkParent now reads the first MIME
// part, and multipartCreate there requires this value, the /upload path and a
// multipart/related media type to agree before it does.
var uploadShapes = map[string]bool{"multipart": true}

// uploadShape reports the create shape a query names, and "" when it names
// none. checkQuery has already refused a query naming both spellings, or
// naming a shape uploadShapes does not carry, so whichever value is here is one
// the guard carries. It is the second of the three signals transport reads
// before it picks a parser for a create body.
func uploadShape(u *url.URL) string {
	vals, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return ""
	}
	if v := vals.Get("uploadType"); v != "" {
		return v
	}
	return vals.Get("upload_protocol")
}

// checkUploadShape refuses a create whose body the parent check cannot read.
// The two parameters are the same choice spelled two ways, so a request naming
// both is refused as well: which one Drive obeys is not decided here.
func checkUploadShape(vals url.Values) error {
	legacy, hasLegacy := vals["uploadType"]
	modern, hasModern := vals["upload_protocol"]
	if hasLegacy && hasModern {
		return refuse("uploadType and upload_protocol are the same choice spelled two ways, and a request naming both leaves the guard reading the one Drive may not obey")
	}
	shape := legacy
	name := "uploadType"
	if hasModern {
		shape, name = modern, "upload_protocol"
	}
	if len(shape) == 0 {
		return nil
	}
	if len(shape) != 1 || !uploadShapes[shape[0]] {
		return refuse("%s=%q is not a create the guard carries; it reads parents out of multipart metadata and out of a plain JSON create, and out of nothing else", name, strings.Join(shape, ","))
	}
	return nil
}

// blockedFields are the field names a read may not ask for. The guard refuses
// /permissions because it names who else can reach the document, and `fields`
// reaches the same data through a plain GET of the file.
var blockedFields = map[string]bool{"permissions": true, "permissionids": true}

// checkFields reads a fields expression as its bare names. Drive separates
// them with commas, slashes and parentheses, so splitting on everything that
// is not a name character gives the list to check, and `permissions/role`
// cannot hide inside a sub-selection.
func checkFields(v string) error {
	// An empty field mask is not an empty answer. Google's system-parameter
	// documentation reads a mask with nothing in it as every field, so `fields=`
	// asks for the same thing `fields=*` does, by a spelling that names no
	// blocked token at all.
	if strings.TrimSpace(v) == "" {
		return refuse("fields= is an empty field mask, which asks for every field, the same as fields=*")
	}
	if strings.Contains(v, "*") {
		return refuse("fields=%q asks for every field, and that includes the permission surface /permissions is refused to protect", v)
	}
	for _, name := range strings.FieldsFunc(v, notFieldRune) {
		if blockedFields[strings.ToLower(name)] {
			return refuse("fields=%q names %q, and who else can reach a document is refused however it is asked for", v, name)
		}
	}
	return nil
}

func notFieldRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		return false
	}
	return true
}
