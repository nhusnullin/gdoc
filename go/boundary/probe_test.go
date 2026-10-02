// The guard over the sentence the capability probe left behind.
//
// Until 2026-10-02 every proposal made a throwaway document first and asked
// Google whether a suggestion was honoured that day. No command does that now:
// the three read-backs and the stop at the first unconfirmed proposal are what
// catch a SUGGEST Google did not honour. The claim was written into the pages a
// colleague reads, into the command's own reasons and into the live tests, so
// this guard asks every one of them on every commit. A page saying a proposal
// probes first sends a reader looking for a folder flag that does nothing.
//
// gdoc probe is still a command a person types, and the live tests still run it
// as their own precondition, so the scan refuses one claim rather than the
// word: what it looks for is a proposal, a proposal file or the command said to
// run the probe, in either word order.

package boundary

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// probeClaimScanned are the files and trees the claim was written into: the
// file every session loads, the colleague's pages, the command's reasons, the
// writer's own, and the live tests, which are the one place left that calls
// the probe.
var probeClaimScanned = []string{
	"CLAUDE.md",
	"README.md",
	"docs/guide",
	"go/cmd/gdoc/doc.go",
	"go/internal/propose/doc.go",
	"go/internal/live",
}

// probeClaims are the two word orders the claim arrives in, each held inside
// one sentence: the subject before the verb, as in "propose runs the probe",
// and the probe first, as in "it is what the command runs before it writes".
// Sentence punctuation ends a match, so a denial standing in its own sentence,
// such as "No command runs that probe any more", is not one of these: that one
// says "that probe" rather than "the probe", and it is the sentence this guard
// exists to keep true.
var probeClaims = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(propose|proposal|the command|the writer)\b[^.;:!?]{0,80}\bruns?\s+(the\s+|its\s+|a\s+)?(capability\s+)?probe\b`),
	regexp.MustCompile(`(?i)\bprobe\b[^.;:!?]{0,80}\b(propose|the command|the writer)\s+runs\b`),
}

func TestNoDocumentSaysAProposalRunsTheProbe(t *testing.T) {
	var found []string
	for _, path := range probeClaimFiles(t) {
		b, err := os.ReadFile(filepath.Join(repoRoot, path))
		if err != nil {
			t.Fatal(err)
		}
		flat, starts := flattenForClaims(b)
		for _, re := range probeClaims {
			for _, loc := range re.FindAllStringIndex(flat, -1) {
				line := sort.SearchInts(starts, loc[0]+1)
				quote := strings.Join(strings.Fields(flat[loc[0]:loc[1]]), " ")
				found = append(found, path+":"+strconv.Itoa(line)+": "+quote)
			}
		}
	}
	if len(found) > 0 {
		t.Errorf("no command runs the capability probe any more, and these sentences still say one does:\n%s", strings.Join(found, "\n"))
	}
}

// TestTheProbeClaimScanFindsAPlantedSentence watches the scanner find something
// before it is trusted to find nothing. The claim is planted in both word
// orders and split across two lines, which is how it was written in a comment.
func TestTheProbeClaimScanFindsAPlantedSentence(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"subject first", "So `propose` runs the probe every time, on a document of its own.", true},
		{"probe first", "// The probe first, because it is what the command runs\n// before it writes.", true},
		{"split across lines", "measurements. So propose runs\nthe probe every time.", true},
		{"the denial stands", "No command runs that probe any more: the read-backs catch it.", false},
		{"the probe command itself", "`probe` creates a throwaway document and says whether Docs honours a suggestion today.", false},
		{"a live test running it itself", "The probe first, because an unenrolled project makes every assertion below meaningless.", false},
		{"the stop", "That stop is the bound on no command running the capability probe any more.", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			flat, _ := flattenForClaims([]byte(c.text))
			var got bool
			for _, re := range probeClaims {
				if re.MatchString(flat) {
					got = true
				}
			}
			if got != c.want {
				t.Errorf("matched = %v, want %v, for %q", got, c.want, c.text)
			}
		})
	}
}

// probeClaimFiles expands the scanned list into the paths really read: a file
// as itself, a directory as every .md and .go file under it. A named path that
// is not there fails, because a scan of nothing passes quietly.
func probeClaimFiles(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, entry := range probeClaimScanned {
		full := filepath.Join(repoRoot, entry)
		info, err := os.Stat(full)
		if err != nil {
			t.Fatalf("%s is named by the scan and is not there: %v", entry, err)
		}
		if !info.IsDir() {
			out = append(out, entry)
			continue
		}
		err = filepath.WalkDir(full, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			switch filepath.Ext(p) {
			case ".md", ".go":
				rel, err := filepath.Rel(repoRoot, p)
				if err != nil {
					return err
				}
				out = append(out, rel)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(out)
	return out
}

// flattenForClaims lays a file out as one string so a sentence split across two
// lines still reads as one, and returns the offset each line starts at so a
// match can be named by line. The markers go out through replacements of the
// same byte length, which is what keeps those offsets true.
func flattenForClaims(b []byte) (string, []int) {
	lines := strings.Split(string(b), "\n")
	starts := make([]int, len(lines))
	var sb strings.Builder
	for i, line := range lines {
		starts[i] = sb.Len()
		sb.WriteString(claimMarkers.Replace(line))
		sb.WriteString(" ")
	}
	return sb.String(), starts
}

// claimMarkers takes the comment slashes, the backticks and the emphasis stars
// out of a line. Every replacement is as long as what it replaces.
var claimMarkers = strings.NewReplacer("//", "  ", "`", " ", "*", " ")
