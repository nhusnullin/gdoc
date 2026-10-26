// The indexes are the one-hop route from a citation to an entry.
//
// These guards read that route in both directions. A register or index row
// must name the one file that holds its entry, and every entry must have a row.

package boundary

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type indexRow struct {
	date   string
	title  string
	status string
	entry  string
}

type indexedEntry struct {
	date  string
	title string
	file  string
}

var decisionFiles = []string{"DECISIONS.md", "DECISIONS-archive.md"}
var measuredFiles = []string{"MEASURED.md", "MEASURED-archive.md"}

func v2File(name string) string {
	return filepath.Join(repoRoot, "docs", "v2", name)
}

func markdownRows(t *testing.T, file, header string, columns int) []indexRow {
	t.Helper()
	b, err := os.ReadFile(v2File(file))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(b), "\n")
	start := -1
	for i, line := range lines {
		if line == header {
			start = i + 2
			break
		}
	}
	if start < 0 {
		t.Fatalf("docs/v2/%s has no %s table", file, header)
	}
	var rows []indexRow
	for _, line := range lines[start:] {
		if !strings.HasPrefix(line, "|") {
			break
		}
		parts := strings.Split(strings.Trim(line, "|"), "|")
		if len(parts) != columns {
			t.Fatalf("docs/v2/%s has %d cells in %q, want %d", file, len(parts), line, columns)
		}
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if columns == 4 {
			rows = append(rows, indexRow{date: parts[0], title: parts[1], status: parts[2], entry: parts[3]})
		} else {
			rows = append(rows, indexRow{date: parts[0], title: parts[1], entry: parts[2]})
		}
	}
	if len(rows) == 0 {
		t.Fatalf("docs/v2/%s has an empty %s table", file, header)
	}
	return rows
}

func headings(t *testing.T, files []string, dated bool) []indexedEntry {
	t.Helper()
	var entries []indexedEntry
	for _, file := range files {
		b, err := os.ReadFile(v2File(file))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(b), "\n")
		for i, line := range lines {
			if !strings.HasPrefix(line, "## ") {
				continue
			}
			title := strings.TrimPrefix(line, "## ")
			if dated {
				date, rest, ok := strings.Cut(title, ". ")
				if !ok || !validDate(date) {
					continue
				}
				entries = append(entries, indexedEntry{date: date, title: rest, file: file})
				continue
			}
			date := ""
			for _, opening := range lines[i+1:] {
				if strings.TrimSpace(opening) == "" {
					continue
				}
				date = openingDate(opening)
				break
			}
			entries = append(entries, indexedEntry{date: date, title: title, file: file})
		}
	}
	return entries
}

func openingDate(line string) string {
	line = strings.TrimPrefix(line, "Measured ")
	if len(line) < len("2026-10-16") {
		return ""
	}
	date := line[:len("2026-10-16")]
	if !validDate(date) {
		return ""
	}
	return date
}

func validDate(date string) bool {
	parsed, err := time.Parse("2006-01-02", date)
	return err == nil && parsed.Format("2006-01-02") == date
}

func sameTitle(a, b string) bool {
	normalize := func(s string) string {
		s = strings.ToLower(strings.TrimSpace(s))
		return strings.TrimRight(s, " .!?*_\t")
	}
	a, b = normalize(a), normalize(b)
	if len(a) > len(b) {
		a, b = b, a
	}
	if strings.HasPrefix(b, a) {
		return true
	}
	// Two register titles omit a parenthetical phrase or conjunction from the
	// fuller entry heading. After punctuation and conjunctions are discarded,
	// the shorter title's words still appear in order in the longer one. This
	// keeps the rule structural instead of carrying exceptions for those rows.
	words := func(s string) []string {
		return strings.FieldsFunc(s, func(r rune) bool {
			return r == ' ' || r == '\t' || strings.ContainsRune(",.:;!?*`'\"()", r)
		})
	}
	short, long := words(a), words(b)
	i := 0
	for _, word := range long {
		if word == "and" {
			continue
		}
		for i < len(short) && short[i] == "and" {
			i++
		}
		if i < len(short) && short[i] == word {
			i++
		}
	}
	for i < len(short) && short[i] == "and" {
		i++
	}
	return i == len(short)
}

func matchingEntries(row indexRow, entries []indexedEntry) []indexedEntry {
	var matches []indexedEntry
	for _, entry := range entries {
		if row.date != entry.date {
			continue
		}
		if sameTitle(row.title, entry.title) {
			matches = append(matches, entry)
		}
	}
	return matches
}

func TestIndexMatchingRequiresTheSameValidDate(t *testing.T) {
	entries := []indexedEntry{{date: "2026-10-16", title: "same title"}}
	for _, date := range []string{"", "2026-10-17"} {
		if got := matchingEntries(indexRow{date: date, title: "same title"}, entries); len(got) != 0 {
			t.Errorf("date %q matched an entry dated 2026-10-16", date)
		}
	}
	if got := matchingEntries(indexRow{date: "2026-10-16", title: "same title"}, entries); len(got) != 1 {
		t.Errorf("the same valid date matched %d entries, want one", len(got))
	}
	for _, opening := range []string{"Measured 2026-13-01 by hand", "Measured 2026-10-1 by hand", "Not dated"} {
		if got := openingDate(opening); got != "" {
			t.Errorf("openingDate(%q) = %q, want no date", opening, got)
		}
	}
}

func TestEveryRegisterRowNamesTheFileThatHoldsIt(t *testing.T) {
	rows := markdownRows(t, "DECISIONS.md", "| Date | Decision | Status | Entry |", 4)
	entries := append(headings(t, decisionFiles, true), headings(t, measuredFiles, false)...)
	for _, row := range rows {
		matches := matchingEntries(row, entries)
		if len(matches) != 1 {
			t.Errorf("register row %s %q matches %d entries, want exactly one", row.date, row.title, len(matches))
			continue
		}
		if matches[0].file != row.entry {
			t.Errorf("register row %s %q names %s, but its entry is in %s", row.date, row.title, row.entry, matches[0].file)
		}
	}
}

func TestEveryDecisionEntryHasItsRegisterRow(t *testing.T) {
	rows := markdownRows(t, "DECISIONS.md", "| Date | Decision | Status | Entry |", 4)
	for _, entry := range headings(t, decisionFiles, true) {
		matches := matchingEntries(indexRow{date: entry.date, title: entry.title}, rowsAsEntries(rows))
		if len(matches) != 1 {
			t.Errorf("%s entry %s %q matches %d register rows, want exactly one", entry.file, entry.date, entry.title, len(matches))
		}
	}
}

func TestDecisionRegisterStatusesDescribeTheirEntries(t *testing.T) {
	rows := markdownRows(t, "DECISIONS.md", "| Date | Decision | Status | Entry |", 4)
	for _, row := range rows {
		if !validDate(row.date) {
			t.Errorf("register row %q has invalid date %q", row.title, row.date)
		}
		valid := row.status == "holds" || row.status == "rejected" || row.status == "measured"
		if rest, ok := strings.CutPrefix(row.status, "superseded "); ok && len(rest) >= len("2026-10-16") {
			valid = validDate(rest[:len("2026-10-16")])
		}
		if !valid {
			t.Errorf("register row %s %q has invalid status %q", row.date, row.title, row.status)
		}
		measured := row.entry == "MEASURED.md" || row.entry == "MEASURED-archive.md"
		if measured && row.status != "measured" {
			t.Errorf("register row %s %q names %s but has status %q, want measured", row.date, row.title, row.entry, row.status)
		}
		if !measured && row.status == "measured" {
			t.Errorf("register row %s %q has status measured but names %s", row.date, row.title, row.entry)
		}
	}
}

func rowsAsEntries(rows []indexRow) []indexedEntry {
	entries := make([]indexedEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, indexedEntry{date: row.date, title: row.title, file: row.entry})
	}
	return entries
}

func TestEveryMeasurementIsIndexed(t *testing.T) {
	rows := markdownRows(t, "MEASURED.md", "| Measured | Section | Entry |", 3)
	entries := headings(t, measuredFiles, false)
	for _, row := range rows {
		if row.title == "Not measured yet" {
			if row.date != "" {
				t.Errorf("measured index row %q has date %q, want none", row.title, row.date)
			}
		} else if !validDate(row.date) {
			t.Errorf("measured index row %q has invalid date %q", row.title, row.date)
		}
		matches := matchingEntries(row, entries)
		if len(matches) != 1 {
			t.Errorf("measured index row %q matches %d sections, want exactly one", row.title, len(matches))
			continue
		}
		if matches[0].file != row.entry {
			t.Errorf("measured index row %q names %s, but its section is in %s", row.title, row.entry, matches[0].file)
		}
	}
	for _, entry := range entries {
		if entry.title == "Not measured yet" {
			if entry.date != "" {
				t.Errorf("%s section %q has date %q, want none", entry.file, entry.title, entry.date)
			}
		} else if !validDate(entry.date) {
			t.Errorf("%s section %q has invalid opening date %q", entry.file, entry.title, entry.date)
		}
		matches := matchingEntries(indexRow{date: entry.date, title: entry.title}, rowsAsEntries(rows))
		if len(matches) != 1 {
			t.Errorf("%s section %q matches %d index rows, want exactly one", entry.file, entry.title, len(matches))
		}
	}
}

func TestArchivedEntriesStayFrozen(t *testing.T) {
	want := map[string]string{
		"DECISIONS-archive.md": "cd9b401b39b06e39039c9034f72c692ec994169b6c31711c789efa7ea38c918e",
		"MEASURED-archive.md":  "5625e9729341930ab7a07cdfb65d13ed516381d1b73f9817a11ded5f70d880b0",
	}
	for file, wantSum := range want {
		b, err := os.ReadFile(v2File(file))
		if err != nil {
			t.Fatal(err)
		}
		_, entries, ok := strings.Cut(string(b), "\n## ")
		if !ok {
			t.Fatalf("docs/v2/%s has no archived entries", file)
		}
		got := fmt.Sprintf("%x", sha256.Sum256([]byte("## "+entries)))
		if got != wantSum {
			t.Errorf("docs/v2/%s entries changed: SHA-256 is %s, want %s", file, got, wantSum)
		}
	}
}

func TestLiveIndexPreamblesNameTheirArchives(t *testing.T) {
	tests := []struct {
		live    string
		archive string
		end     string
	}{
		{live: "DECISIONS.md", archive: "DECISIONS-archive.md", end: "## The register"},
		{live: "MEASURED.md", archive: "MEASURED-archive.md", end: "| Measured | Section | Entry |"},
	}
	for _, tt := range tests {
		t.Run(tt.live, func(t *testing.T) {
			b, err := os.ReadFile(v2File(tt.live))
			if err != nil {
				t.Fatal(err)
			}
			preamble, _, ok := strings.Cut(string(b), tt.end)
			if !ok {
				t.Fatalf("docs/v2/%s has no %q after its preamble", tt.live, tt.end)
			}
			if !strings.Contains(preamble, "2026-10-16") {
				t.Errorf("docs/v2/%s preamble does not name the live-entry cutoff", tt.live)
			}
			if !strings.Contains(preamble, "]("+tt.archive+")") {
				t.Errorf("docs/v2/%s preamble does not link to %s", tt.live, tt.archive)
			}
		})
	}
}
