package main

// Spike: can ledongthuc/pdf's Content() give per-page LINES from a real
// Google Docs PDF export? Group by Y, sort by X, collapse whitespace.

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

var ws = regexp.MustCompile(`\s+`)

func pageLines(p pdf.Page) []string {
	texts := p.Content().Text
	if len(texts) == 0 {
		return nil
	}
	type row struct {
		y     float64
		items []pdf.Text
	}
	var rows []*row
	for _, t := range texts {
		placed := false
		for _, r := range rows {
			if math.Abs(r.y-t.Y) < 2.0 { // 2pt tolerance
				r.items = append(r.items, t)
				placed = true
				break
			}
		}
		if !placed {
			rows = append(rows, &row{y: t.Y, items: []pdf.Text{t}})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].y > rows[j].y })
	var out []string
	for _, r := range rows {
		sort.SliceStable(r.items, func(i, j int) bool { return r.items[i].X < r.items[j].X })
		var b strings.Builder
		for _, it := range r.items {
			b.WriteString(it.S)
		}
		s := strings.TrimSpace(ws.ReplaceAllString(b.String(), " "))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func main() {
	f, r, err := pdf.Open(os.Args[1])
	if err != nil {
		fmt.Println("open error:", err)
		return
	}
	defer f.Close()
	fmt.Printf("pages: %d\n", r.NumPage())
	for i := 1; i <= r.NumPage() && i <= 4; i++ {
		lines := pageLines(r.Page(i))
		fmt.Printf("\n--- page %d (%d lines) ---\n", i, len(lines))
		for j, l := range lines {
			if j >= 12 {
				fmt.Printf("  ... %d more\n", len(lines)-12)
				break
			}
			fmt.Printf("  %s\n", l)
		}
	}
}
