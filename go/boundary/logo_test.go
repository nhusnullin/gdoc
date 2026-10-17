package boundary

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// realLogoSHA256 is the SHA-256 of the firm's own logo, the PNG that was
// house.yaml's logo.base64 until M22. It is a hash and not the bytes for the
// reason companyHostSHA256 is a hash and not the domain: the rule is that the
// logo is not in this repository, and a test carrying it would be the first
// file to break the rule it states. The logo lives in one file in the firm's
// Drive now, and house.yaml names it by id.
const realLogoSHA256 = "a0c4fc99472c7a8fb069d387b8755dd73f622c2ce261c35a34871e5f1af2b3ef"

// scanForBytes names every place in the tree holding a file whose SHA-256 is
// want: the path on its own for an ordinary file, and path:member for a member
// of a .docx, which is a zip and hides its members from any scan that reads
// bytes straight off the disk.
//
// It hashes whole files rather than looking for the bytes inside a larger one,
// because the two ways the logo travels are as a file and as a part of a
// document. A re-encoding of the same picture is a different file and this
// scan does not know it; what it holds is that nobody puts this file back.
//
// Base64 inside a larger file is the one re-encoding that matters, because it
// is the form the logo had until M22. There is one file in the tree written to
// carry a logo that way, internal/house/testdata/house-base64.yaml, and
// TestTheBase64HouseCarriesThePlaceholderAndNothingElse holds its bytes to the
// stand-in. This scan covers everything else.
func scanForBytes(root string, want [32]byte) ([]string, error) {
	var found []string
	err := walkTracked(root, func(rel, path string) error {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if sha256.Sum256(raw) == want {
			found = append(found, rel)
		}
		if !strings.HasSuffix(rel, ".docx") {
			return nil
		}
		members, err := zipMembers(raw, want)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		for _, name := range members {
			found = append(found, rel+":"+name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}

// zipMembers names every member of one zip whose bytes hash to want. A zip it
// cannot open is an error rather than an empty answer: a document this scan
// half read would pass while carrying anything.
func zipMembers(raw []byte, want [32]byte) ([]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, fmt.Errorf("a .docx this scan cannot open as a zip: %w", err)
	}
	var found []string
	for _, member := range zr.File {
		if member.FileInfo().IsDir() {
			continue
		}
		rc, err := member.Open()
		if err != nil {
			return nil, fmt.Errorf("member %s: %w", member.Name, err)
		}
		sum := sha256.New()
		_, err = io.Copy(sum, rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("member %s: %w", member.Name, err)
		}
		var got [32]byte
		copy(got[:], sum.Sum(nil))
		if got == want {
			found = append(found, member.Name)
		}
	}
	return found, nil
}

// The canary. The scan is watched finding the bytes bare and inside a zip
// before it is trusted to find none, and what it is handed is a planted
// payload of its own rather than the firm's file.
func TestTheLogoScanFindsAPlantedPNG(t *testing.T) {
	const planted = "a planted logo\n"
	root := t.TempDir()
	write(t, filepath.Join(root, "clean.txt"), "nothing here\n")
	write(t, filepath.Join(root, "sub", "bare.png"), planted)
	writeZip(t, filepath.Join(root, "note.docx"), map[string]string{
		"word/document.xml":     "<w:document/>",
		"word/media/image1.png": planted,
	})
	writeZip(t, filepath.Join(root, "clean.docx"), map[string]string{
		"word/media/image1.png": "another picture\n",
	})

	found, err := scanForBytes(root, wantHost(t, "c5d2eceb2e8592fd87dd420bf7a64f57d50b01b990601002f8e6a40dd5d989a3"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"note.docx:word/media/image1.png", "sub/bare.png"}
	if strings.Join(found, " ") != strings.Join(want, " ") {
		t.Errorf("found = %v, want %v", found, want)
	}
}

// The rule itself. The firm's logo is in the firm's Drive and in no file here,
// as a picture, as a fixture or inside a document.
func TestTheLogoIsNotInTheTree(t *testing.T) {
	found, err := scanForBytes(repoRoot, wantHost(t, realLogoSHA256))
	if err != nil {
		t.Fatal(err)
	}
	for _, where := range found {
		t.Errorf("%s is the firm's own logo; house.yaml names it by its Drive id, and a fixture carries the placeholder", where)
	}
}

// writeZip writes one zip of the given members, in sorted name order so the
// scan's answer is the same on every run.
func writeZip(t *testing.T, path string, members map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for _, name := range sortedNames(members) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(members[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func sortedNames(members map[string]string) []string {
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// logoWordPages are the pages that promise a reader where the logo comes from:
// the two skills that run a build or a restyle, and the two guide pages behind
// them. A person whose run refused because Drive was unreachable reads one of
// these, so each has to say the whole of it in one place.
var logoWordPages = []string{
	"skills/gdoc-publish/SKILL.md",
	"skills/gdoc-restyle/SKILL.md",
	"docs/guide/publishing.md",
	"docs/guide/restyle.md",
}

// logoWords are the four facts about the logo, in the words each page says
// them in. Where it lives, how often gdoc asks, where the answer is kept, and
// what a run that cannot get it does. The last is the one that matters on a
// bad day: a reader who knows the run refuses looks for the network, and a
// reader who does not goes looking for a missing picture in the tree.
var logoWords = []string{
	"one file in the firm's Drive",
	"once a day",
	"cached beside the token",
	"refuses",
}

// The words. Each page says where the house logo comes from, because since M22
// the tree carries no copy of it and a build can refuse for a reason no older
// page explains.
func TestTheWordsSayWhereTheLogoLives(t *testing.T) {
	for _, page := range logoWordPages {
		b, err := os.ReadFile(filepath.Join(repoRoot, page))
		if err != nil {
			t.Fatalf("%s is named by this test and is not there: %v", page, err)
		}
		// Folded, so a rewrap of the paragraph is not a difference: these
		// pages are written to about eighty columns and a phrase lands
		// across two lines as often as not.
		text := strings.Join(strings.Fields(string(b)), " ")
		for _, want := range logoWords {
			if !strings.Contains(text, want) {
				t.Errorf("%s never says %q, so a reader whose build refused cannot tell why", page, want)
			}
		}
	}
}

// The README's one line. The binary and these pages are the code's; the
// picture and the marks they fetch are the firm's, and a colleague who forks
// this repository gets the one and not the other.
func TestTheREADMESaysTheLogoIsNotLicensedWithTheCode(t *testing.T) {
	text := strings.Join(strings.Fields(releaseREADME(t)), " ")
	for _, want := range []string{
		"The firm's logo and marks are the firm's",
		"not licensed with the code",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("README.md never says %q, and a colleague forking this repository cannot tell what is theirs to use", want)
		}
	}
}
