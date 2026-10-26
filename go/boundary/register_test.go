// The indexes are the one-hop route from a citation to an entry.
//
// These guards read that route in both directions. A register or index row
// must name the one file that holds its entry, and every entry must have a row.

package boundary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type indexRow struct {
	date  string
	title string
	entry string
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
			rows = append(rows, indexRow{date: parts[0], title: parts[1], entry: parts[3]})
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
				if !ok || len(date) != len("2026-10-16") {
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
	for i, want := range "0000-00-00" {
		if want == '-' {
			if date[i] != '-' {
				return ""
			}
			continue
		}
		if date[i] < '0' || date[i] > '9' {
			return ""
		}
	}
	return date
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
		if row.date != "" && entry.date != "" && row.date != entry.date {
			continue
		}
		if sameTitle(row.title, entry.title) {
			matches = append(matches, entry)
		}
	}
	return matches
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
	b, err := os.ReadFile(v2File("DECISIONS.md"))
	if err != nil {
		t.Fatal(err)
	}
	header, columns := "| Date | Decision | Status |", 3
	if strings.Contains(string(b), "| Date | Decision | Status | Entry |") {
		header, columns = "| Date | Decision | Status | Entry |", 4
	}
	rows := markdownRows(t, "DECISIONS.md", header, columns)
	for _, entry := range headings(t, decisionFiles, true) {
		matches := matchingEntries(indexRow{date: entry.date, title: entry.title}, rowsAsEntries(rows))
		if len(matches) != 1 {
			t.Errorf("%s entry %s %q matches %d register rows, want exactly one", entry.file, entry.date, entry.title, len(matches))
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
		matches := matchingEntries(indexRow{title: entry.title}, rowsAsEntries(rows))
		if len(matches) != 1 {
			t.Errorf("%s section %q matches %d index rows, want exactly one", entry.file, entry.title, len(matches))
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
