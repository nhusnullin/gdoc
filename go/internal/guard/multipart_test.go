package guard

import (
	"net/http"
	"strings"
	"testing"
)

// This file is Task 1 of M6: the guard learns to read a multipart create, and
// every test here was written as an attack before the parse existed.
//
// The rule the whole file states is one sentence. Drive picks the parser for a
// create body from three signals, the /upload path, the uploadType or
// upload_protocol value, and the media type on the request. The guard has to
// pick its parser from the same three, and refuse the moment they disagree.
// Anything else is a body judged one way and sent another, which is the class
// "Authorization" beside "authorization" already belongs to.

const grantedFolder = "FOLDER1"

// relatedBody builds a multipart/related body the shape Drive's own
// documentation describes: a JSON metadata part first, the file second.
func relatedBody(boundary, metaHeaders, meta, file string) string {
	var b strings.Builder
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString(metaHeaders)
	b.WriteString("\r\n")
	b.WriteString(meta)
	b.WriteString("\r\n--" + boundary + "\r\n")
	b.WriteString("Content-Type: application/vnd.openxmlformats-officedocument.wordprocessingml.document\r\n\r\n")
	b.WriteString(file)
	b.WriteString("\r\n--" + boundary + "--\r\n")
	return b.String()
}

// jsonPart is the metadata part header Google's documentation puts on a
// multipart create, charset and all.
const jsonPart = "Content-Type: application/json; charset=UTF-8\r\n"

const uploadURL = "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart"

// goodMultipart is the request gdoc itself builds: all three signals agreeing,
// one folder named, and it is the granted one.
func goodMultipart() (string, string, string) {
	body := relatedBody("BOUND", jsonPart, `{"name":"Policy","parents":["FOLDER1"],"mimeType":"application/vnd.google-apps.document"}`, "PK\x03\x04docx bytes")
	return uploadURL, "multipart/related; boundary=BOUND", body
}

// refusedBeforeTheWire posts the request and requires a guard refusal that
// never reached the base transport and taught the policy nothing.
func refusedBeforeTheWire(t *testing.T, what, rawURL, contentType, body string) {
	t.Helper()
	f := &fake{status: 200, body: `{"id":"UPLOADED"}`}
	p := NewPolicy()
	p.AllowCreateIn(grantedFolder)
	c := NewClient(p, f)
	_, err := c.Post(rawURL, contentType, strings.NewReader(body))
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("%s must be refused, got %v", what, err)
	}
	if len(f.seen) != 0 {
		t.Fatalf("%s reached the wire", what)
	}
	if _, known := p.files["UPLOADED"]; known {
		t.Fatalf("%s taught the policy an id at full level", what)
	}
}

// TestTheThreeSignalsMustAgree is the attack this whole task exists for. Two
// of these three shapes fail closed today by luck rather than by design: Drive
// refuses the first, so no 2xx comes back and nothing is learned, and the
// guard's own JSON parse refuses the second. The moment the guard can parse
// multipart, a parser chosen from the wrong signal is a body judged one way and
// sent another, so the rule lands in the same change as the parse.
func TestTheThreeSignalsMustAgree(t *testing.T) {
	_, relatedType, related := goodMultipart()
	cases := []struct {
		what        string
		rawURL      string
		contentType string
		body        string
	}{
		{
			// This is TestUploadCreateAlsoTeachesThePolicy as it stood before
			// M6: the parameter says multipart and the body is plain JSON.
			"a JSON body under uploadType=multipart",
			uploadURL,
			"application/json",
			`{"parents":["FOLDER1"]}`,
		},
		{
			"a multipart body with no upload parameter",
			"https://www.googleapis.com/upload/drive/v3/files",
			relatedType,
			related,
		},
		{
			"a multipart body on the plain files path",
			"https://www.googleapis.com/drive/v3/files",
			relatedType,
			related,
		},
		{
			"a JSON body on the /upload path with no upload parameter",
			"https://www.googleapis.com/upload/drive/v3/files",
			"application/json",
			`{"parents":["FOLDER1"]}`,
		},
	}
	for _, c := range cases {
		refusedBeforeTheWire(t, c.what, c.rawURL, c.contentType, c.body)
	}
}

// The boundary comes from the header, and a header the guard cannot read is a
// body it cannot find the metadata in.
func TestTheBoundaryComesFromTheHeaderAndMustBeUsable(t *testing.T) {
	_, _, related := goodMultipart()
	cases := []struct {
		what        string
		contentType string
	}{
		{"a multipart create with no boundary parameter", "multipart/related"},
		{"a boundary the body does not use", "multipart/related; boundary=OTHER"},
		// mime.ParseMediaType refuses a duplicate parameter. A hand-rolled
		// split would take one of the two and leave the server the other.
		{"a boundary given twice", "multipart/related; boundary=BOUND; boundary=OTHER"},
		{"a Content-Type that is not a media type at all", "multipart/related; boundary"},
	}
	for _, c := range cases {
		refusedBeforeTheWire(t, c.what, uploadURL, c.contentType, related)
	}
}

// The metadata part is small and first, so a request gdoc builds cannot reach
// this. A request somebody adds later can, and the refusal is how they find
// out.
func TestAMetadataPartOutsideThePeekIsRefused(t *testing.T) {
	pad := strings.Repeat("x", maxPeek)
	body := relatedBody("BOUND", jsonPart, `{"note":"`+pad+`","parents":["FOLDER1"]}`, "docx")
	refusedBeforeTheWire(t, "a metadata part the guard cannot read whole", uploadURL,
		"multipart/related; boundary=BOUND", body)
}

// The first part is the metadata, so its own media type has to say so. A part
// the guard reads as JSON while Drive reads it as something else is the same
// mismatch one layer in.
func TestTheFirstPartMustBeJSON(t *testing.T) {
	cases := []struct {
		what    string
		headers string
		meta    string
	}{
		{"a first part with no Content-Type", "", `{"parents":["FOLDER1"]}`},
		{"a first part that is not JSON", "Content-Type: text/plain\r\n", `{"parents":["FOLDER1"]}`},
		{"a first part that is not valid JSON", jsonPart, `{"parents":["FOLDER1"`},
		{"a first part carrying two Content-Type headers",
			"Content-Type: application/json\r\nContent-Type: text/plain\r\n", `{"parents":["FOLDER1"]}`},
		// The guard reads the part's raw bytes and Drive would decode them, so
		// an encoding is a body the two read differently.
		{"a first part carrying Content-Transfer-Encoding",
			jsonPart + "Content-Transfer-Encoding: base64\r\n", `{"parents":["FOLDER1"]}`},
	}
	for _, c := range cases {
		body := relatedBody("BOUND", c.headers, c.meta, "docx")
		refusedBeforeTheWire(t, c.what, uploadURL, "multipart/related; boundary=BOUND", body)
	}
}

// The parents check is the plain create's, run on the bytes of the first part.
// It refuses what it has always refused.
func TestTheFirstPartsParentsAreCheckedLikeAPlainCreate(t *testing.T) {
	cases := []struct {
		what string
		meta string
	}{
		{"a metadata part naming parents twice", `{"parents":["OTHER"],"parents":["FOLDER1"]}`},
		{"a metadata part naming another folder", `{"parents":["OTHER"]}`},
		{"a metadata part naming two folders", `{"parents":["FOLDER1","OTHER"]}`},
		{"a metadata part naming no folder at all", `{"name":"Policy"}`},
	}
	for _, c := range cases {
		body := relatedBody("BOUND", jsonPart, c.meta, "docx")
		refusedBeforeTheWire(t, c.what, uploadURL, "multipart/related; boundary=BOUND", body)
	}
}

// The guard reads the first part and stops. It does not scan the body for
// something that looks like a boundary, which is what would let the file's own
// bytes move where the guard thinks the metadata ends.
func TestTheGuardReadsTheFirstPartOnly(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"UPLOADED"}`}
	p := NewPolicy()
	p.AllowCreateIn(grantedFolder)
	c := NewClient(p, f)
	file := "PK\x03\x04 --BOUND\r\nContent-Type: application/json\r\n\r\n{\"parents\":[\"OTHER\"]}\r\n"
	body := relatedBody("BOUND", jsonPart, `{"parents":["FOLDER1"]}`, file)
	resp, err := c.Post(uploadURL, "multipart/related; boundary=BOUND", strings.NewReader(body))
	if err != nil {
		t.Fatalf("the file's own bytes are opaque to the guard: %v", err)
	}
	resp.Body.Close()
	if p.files["UPLOADED"] != LevelFull {
		t.Fatal("the create was carried but its id was not learned")
	}
}

// Naming a folder is the grant. Without one there is no create at all, in any
// shape.
func TestAMultipartCreateWithNoGrantIsRefused(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"UPLOADED"}`}
	p := NewPolicy()
	c := NewClient(p, f)
	rawURL, ct, body := goodMultipart()
	_, err := c.Post(rawURL, ct, strings.NewReader(body))
	if err == nil || !strings.Contains(err.Error(), "guard refused") {
		t.Fatalf("a create with no folder granted must be refused, got %v", err)
	}
	if len(f.seen) != 0 {
		t.Fatal("it reached the wire")
	}
}

// The carry, and the other half of the door: the id comes back from a create
// the guard itself judged, so it is learned at full level.
//
// This replaces TestAMultipartCreateIsRefusedForNow, whose pin said the shape
// stayed refused until something read the first MIME part. M6 is that
// something, so the pin is spent.
func TestAMultipartCreateNamingTheGrantedFolderIsCarried(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"UPLOADED"}`}
	p := NewPolicy()
	p.AllowCreateIn(grantedFolder)
	c := NewClient(p, f)
	rawURL, ct, body := goodMultipart()
	resp, err := c.Post(rawURL, ct, strings.NewReader(body))
	if err != nil {
		t.Fatalf("a multipart create naming exactly the granted folder is carried: %v", err)
	}
	resp.Body.Close()
	if len(f.seen) != 1 {
		t.Fatalf("want one request on the wire, got %d", len(f.seen))
	}
	if p.files["UPLOADED"] != LevelFull {
		t.Fatal("the new id was not learned at LevelFull")
	}
}

// The whole body still goes out, the file part included. The guard reads the
// front of it and hands the rest on untouched.
func TestTheWholeMultipartBodyReachesTheWire(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"UPLOADED"}`}
	p := NewPolicy()
	p.AllowCreateIn(grantedFolder)
	c := NewClient(p, f)
	rawURL, ct, body := goodMultipart()
	resp, err := c.Post(rawURL, ct, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	sent := readAll(t, f.seen[0])
	if sent != body {
		t.Fatalf("the bytes on the wire are not the bytes that were judged:\nwant %q\ngot  %q", body, sent)
	}
}

// A plain JSON create is untouched by any of this: no upload path, no upload
// parameter, no multipart media type, and the parents read straight out of the
// body.
func TestAPlainJSONCreateStillWorks(t *testing.T) {
	f := &fake{status: 200, body: `{"id":"CREATED"}`}
	p := NewPolicy()
	p.AllowCreateIn(grantedFolder)
	c := NewClient(p, f)
	resp, err := c.Post("https://www.googleapis.com/drive/v3/files", "application/json",
		strings.NewReader(`{"parents":["FOLDER1"],"name":"probe"}`))
	if err != nil {
		t.Fatalf("a plain JSON create must work exactly as it did: %v", err)
	}
	resp.Body.Close()
	if p.files["CREATED"] != LevelFull {
		t.Fatal("the plain create did not teach the policy")
	}
}

func readAll(t *testing.T, req *http.Request) string {
	t.Helper()
	if req.Body == nil {
		return ""
	}
	var b strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := req.Body.Read(buf)
		b.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return b.String()
}
