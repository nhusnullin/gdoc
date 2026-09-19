package export

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"gdoc/internal/docs"
	"gdoc/internal/docx"
)

// pictureDir is where the bytes these tests use live. They are the body
// package's own pictures rather than a second copy of them.
const pictureDir = "../body/testdata/docs"

// TestPicturesPairByOrder is decision 10: nothing crosses the two routes, so
// the k-th object of the read is the k-th picture of the export.
func TestPicturesPairByOrder(t *testing.T) {
	objects := []docs.Object{
		{ID: "kix.one", Kind: docs.KindImage},
		{ID: "kix.two", Kind: docs.KindDrawing},
	}
	media := []docx.Medium{
		{Name: "word/media/image1.png", Bytes: []byte("first")},
		{Name: "word/media/image2.jpeg", Bytes: []byte("second")},
	}

	pics, warnings := Pictures(objects, media, nil, Options{})

	if len(warnings) != 0 {
		t.Errorf("two objects and two pictures need no warning, and it warned: %v", warnings)
	}
	if len(pics) != 2 {
		t.Fatalf("two objects give two pictures and it gave %d", len(pics))
	}
	for i, c := range []struct{ id, ext, want string }{
		{"kix.one", ".png", "first"},
		{"kix.two", ".jpeg", "second"},
	} {
		if pics[i].ObjectID != c.id {
			t.Errorf("picture %d names the object %q, and the %d-th object is %q", i, pics[i].ObjectID, i, c.id)
		}
		if pics[i].Ext != c.ext {
			t.Errorf("picture %d carries the extension %q, and its part is a %s", i, pics[i].Ext, c.ext)
		}
		if string(pics[i].Bytes) != c.want {
			t.Errorf("picture %d carries %q, and the %d-th part is %q", i, pics[i].Bytes, i, c.want)
		}
	}
}

// TestACountMismatchWritesNoPictureAndWarns is the refusal the pairing rests
// on: the two routes disagree, so nothing says which bytes belong to which
// object and no file is written at all.
func TestACountMismatchWritesNoPictureAndWarns(t *testing.T) {
	objects := []docs.Object{
		{ID: "kix.one", Kind: docs.KindImage},
		{ID: "kix.two", Kind: docs.KindImage},
		{ID: "kix.three", Kind: docs.KindDrawing},
	}
	media := []docx.Medium{
		{Name: "word/media/image1.png", Bytes: []byte("first")},
		{Name: "word/media/image2.png", Bytes: []byte("second")},
	}

	pics, warnings := Pictures(objects, media, nil, Options{})

	if len(warnings) != 1 {
		t.Fatalf("a count mismatch is one warning and it gave %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "3") || !strings.Contains(warnings[0], "2") {
		t.Errorf("the warning reads %q, and it says three objects against two pictures", warnings[0])
	}
	if len(pics) != 3 {
		t.Fatalf("every object is still a picture, as a placeholder, and it gave %d", len(pics))
	}
	for i, p := range pics {
		if len(p.Bytes) != 0 || p.Matched != "" {
			t.Errorf("picture %d carries %d bytes and matched %q, and a mismatch writes nothing",
				i, len(p.Bytes), p.Matched)
		}
	}
}

// TestTwoIdenticalPicturesKeepTheirOrder: the same bytes twice are two
// pictures, each under its own object, because the pairing is by position and
// never by what the bytes are.
func TestTwoIdenticalPicturesKeepTheirOrder(t *testing.T) {
	same := []byte("one and the same")
	objects := []docs.Object{
		{ID: "kix.one", Kind: docs.KindImage},
		{ID: "kix.two", Kind: docs.KindImage},
	}
	media := []docx.Medium{
		{Name: "word/media/image1.png", Bytes: same},
		{Name: "word/media/image1.png", Bytes: same},
	}

	pics, warnings := Pictures(objects, media, nil, Options{})

	if len(warnings) != 0 {
		t.Errorf("two pictures of one part need no warning, and it warned: %v", warnings)
	}
	if len(pics) != 2 {
		t.Fatalf("two objects give two pictures and it gave %d", len(pics))
	}
	if pics[0].ObjectID != "kix.one" || pics[1].ObjectID != "kix.two" {
		t.Errorf("the pictures name %q and %q, and the objects are kix.one then kix.two",
			pics[0].ObjectID, pics[1].ObjectID)
	}
	if !bytes.Equal(pics[0].Bytes, same) || !bytes.Equal(pics[1].Bytes, same) {
		t.Error("both pictures are the same part, so both carry its bytes")
	}
}

// TestTheHashMatchIsOffUntilMeasured is measurement 2 of the spec: until a
// live read says a published PNG comes back byte for byte, the match against
// the note's own pictures is off, every picture is written, and the reply says
// the match is not trusted yet.
func TestTheHashMatchIsOffUntilMeasured(t *testing.T) {
	png := read(t, "badge.png")
	objects := []docs.Object{{ID: "kix.one", Kind: docs.KindImage}}
	media := []docx.Medium{{Name: "word/media/image1.png", Bytes: png}}
	note := []NotePicture{{Target: "badge.png", Path: "/notes/badge.png", Bytes: png}}

	off, warnings := Pictures(objects, media, note, Options{})

	if len(off) != 1 || off[0].Matched != "" || !bytes.Equal(off[0].Bytes, png) {
		t.Fatalf("with the match off the picture is written: it matched %q and carries %d bytes",
			off[0].Matched, len(off[0].Bytes))
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "match") {
		t.Fatalf("with the match off the reply says so, and the warnings are %v", warnings)
	}

	on, warnings := Pictures(objects, media, note, Options{MatchByHash: true})

	if len(warnings) != 0 {
		t.Errorf("with the match on nothing is warned about, and it said %v", warnings)
	}
	if len(on) != 1 || on[0].Matched != "badge.png" {
		t.Fatalf("with the match on the note's own picture is kept, and it matched %q", on[0].Matched)
	}
	if len(on[0].Bytes) != 0 {
		t.Errorf("a matched picture carries %d bytes, and a matched picture is never written", len(on[0].Bytes))
	}
}

// TestADifferentPictureIsWrittenWhenTheMatchIsOn is scenario 4: the bytes
// match nothing the note holds, so the picture is a file of its own.
func TestADifferentPictureIsWrittenWhenTheMatchIsOn(t *testing.T) {
	objects := []docs.Object{{ID: "kix.one", Kind: docs.KindImage}}
	media := []docx.Medium{{Name: "word/media/image1.png", Bytes: read(t, "diagram.png")}}
	note := []NotePicture{{Target: "badge.png", Path: "/notes/badge.png", Bytes: read(t, "badge.png")}}

	pics, warnings := Pictures(objects, media, note, Options{MatchByHash: true})

	if len(warnings) != 0 {
		t.Errorf("a picture that matched nothing needs no warning, and it said %v", warnings)
	}
	if len(pics) != 1 || pics[0].Matched != "" {
		t.Fatalf("the picture matched %q, and the note holds different bytes", pics[0].Matched)
	}
	if len(pics[0].Bytes) == 0 {
		t.Error("a picture that matched nothing is written, and it carries no bytes")
	}
}

// TestTheNotesPicturesAreFoundThroughItsLinks: the note at --out is read for
// its own pictures, PNG and JPEG alike, resolved against its directory. A
// data: or http destination is not a file and is stepped over, and no second
// note in that directory is read.
func TestTheNotesPicturesAreFoundThroughItsLinks(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatalf("the assets folder was not made: %v", err)
	}
	write(t, filepath.Join(dir, "badge.png"), read(t, "badge.png"))
	write(t, filepath.Join(dir, "assets", "photo.jpg"), read(t, "photo.jpg"))
	write(t, filepath.Join(dir, "elsewhere.png"), read(t, "diagram.png"))
	write(t, filepath.Join(dir, "other.md"), []byte("![](elsewhere.png)\n"))

	src := []byte("# A note\n\n" +
		"![a badge](badge.png)\n\n" +
		"![a photo](assets/photo.jpg)\n\n" +
		"![inline](data:image/png;base64,iVBORw0KGgo=)\n\n" +
		"![remote](https://example.com/one.png)\n")

	found, warnings := NotePictures(src, dir)

	if len(warnings) != 0 {
		t.Errorf("every picture the note names was read, and it warned: %v", warnings)
	}
	if len(found) != 2 {
		t.Fatalf("the note names two files and it found %d: %v", len(found), targets(found))
	}
	if found[0].Target != "badge.png" || found[1].Target != "assets/photo.jpg" {
		t.Errorf("the note's pictures are %v, and it names badge.png then assets/photo.jpg", targets(found))
	}
	if !bytes.Equal(found[0].Bytes, read(t, "badge.png")) {
		t.Error("the badge carries bytes that are not the badge's")
	}
	if !bytes.Equal(found[1].Bytes, read(t, "photo.jpg")) {
		t.Error("the photo carries bytes that are not the photo's")
	}
	for _, p := range found {
		if strings.Contains(p.Target, "elsewhere") {
			t.Errorf("a second note in the directory was read: %v", targets(found))
		}
	}
}

// TestANotePictureThatCannotBeReadIsNamed: nothing here is silent. A link to a
// file that is not there, and a file that is neither PNG nor JPEG, are both
// facts the session acts on.
func TestANotePictureThatCannotBeReadIsNamed(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "notes.txt"), []byte("not a picture"))

	found, warnings := NotePictures([]byte("![](gone.png)\n\n![](notes.txt)\n"), dir)

	if len(found) != 0 {
		t.Errorf("neither file is a picture this read carries, and it found %v", targets(found))
	}
	if len(warnings) != 2 {
		t.Fatalf("two pictures could not be read and it gave %d warnings: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "gone.png") || !strings.Contains(warnings[1], "notes.txt") {
		t.Errorf("the warnings read %v, and they name gone.png and notes.txt", warnings)
	}
}

// TestObjectsAreTheTabsPicturesInBodyOrder: the list the pairing counts. An
// inline picture stands where its run does, a floating one after the paragraph
// it is anchored to, which is where internal/view prints its placeholder, and
// an object that is neither a picture nor a drawing is not counted at all.
func TestObjectsAreTheTabsPicturesInBodyOrder(t *testing.T) {
	tab := docs.Tab{
		ID: "t.0",
		Body: []docs.Block{
			{Paragraph: &docs.Paragraph{
				Runs: []docs.Run{
					{Kind: docs.KindText, Text: "words"},
					{Kind: docs.KindImage, Detail: &docs.Detail{ID: "kix.inline"}},
					{Kind: docs.KindEquation},
				},
				Positioned: []string{"kix.float"},
			}},
			{Table: &docs.Table{Rows: [][]docs.Cell{{{Blocks: []docs.Block{
				{Paragraph: &docs.Paragraph{Runs: []docs.Run{
					{Kind: docs.KindDrawing, Detail: &docs.Detail{ID: "kix.cell"}},
				}}},
			}}}}}},
		},
		Positioned: map[string]docs.Object{
			"kix.float": {ID: "kix.float", Kind: docs.KindDrawing},
		},
	}

	got := Objects(tab)

	want := []string{"kix.inline", "kix.float", "kix.cell"}
	if len(got) != len(want) {
		t.Fatalf("the tab holds %d pictures and it found %d: %v", len(want), len(got), ids(got))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Errorf("picture %d is %q, and body order puts %q there", i, got[i].ID, id)
		}
	}
}

func ids(objects []docs.Object) []string {
	out := make([]string, 0, len(objects))
	for _, o := range objects {
		out = append(out, o.ID)
	}
	return out
}

func targets(pics []NotePicture) []string {
	out := make([]string, 0, len(pics))
	for _, p := range pics {
		out = append(out, p.Target)
	}
	return out
}

func read(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(pictureDir, name))
	if err != nil {
		t.Fatalf("%s could not be read: %v", name, err)
	}
	return b
}

func write(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("%s could not be written: %v", path, err)
	}
}

// TestANotePictureIsOnlyEverAReadOfARegularFile: the ceiling this read states
// is a fact about a regular file's size, and about nothing else. A note that
// links a device node or a pipe would otherwise read until the machine runs
// out of memory, or block the command for ever, on a stat that said nothing.
func TestANotePictureIsOnlyEverAReadOfARegularFile(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "pipe.png")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("this filesystem makes no fifo: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, warnings := NotePictures([]byte("![](pipe.png)\n"), dir)
		if len(warnings) != 1 || !strings.Contains(warnings[0], "pipe.png") {
			t.Errorf("the warnings read %v, and one of them names pipe.png", warnings)
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the read of a fifo did not come back: it is waiting for a writer that will never come")
	}
}
