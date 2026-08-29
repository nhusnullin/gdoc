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
// nothing had been suggested.
var docsReadParams = map[string]bool{
	"alt":                 true,
	"fields":              true,
	"suggestionsViewMode": true,
	"includeTabsContent":  true,
}

// driveReadParams are the parameters a Drive read may carry. The path is only
// half of what a GET asks for: the path names one file and the level says read,
// and the query decides how much of that file comes back.
var driveReadParams = map[string]bool{
	"alt":               true, // export asks for the bytes rather than the metadata
	"mimeType":          true, // which export format
	"fields":            true, // narrowed further by checkFields
	"pageSize":          true, // comments come back a page at a time
	"pageToken":         true,
	"includeDeleted":    true, // comments list
	"supportsAllDrives": true,
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
	}
	return nil
}

// uploadShapes are the create shapes the guard can check, under either
// spelling. `multipart` and `resumable` both put the metadata where the parent
// check can read it: a multipart body opens with the metadata part, and a
// resumable start is the metadata on its own. No upload parameter at all is a
// plain JSON create, which the parent check reads directly.
//
// The refused shape is the one that makes the whole body the file's content:
// `uploadType=media`, and the same thing under its newer name,
// `upload_protocol=raw`. There `{"parents":["FOLDER1"]}` is bytes to Drive and
// metadata to the parent check, so the file lands unparented and the guard then
// learns its id at full level. Nothing here may learn an id from a create it
// could not verify.
var uploadShapes = map[string]bool{"multipart": true, "resumable": true}

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
		return refuse("%s=%q is not a create the guard can check; it reads parents out of multipart and resumable metadata only", name, strings.Join(shape, ","))
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
