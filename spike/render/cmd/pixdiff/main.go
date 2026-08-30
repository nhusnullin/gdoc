// Command pixdiff compares two directories of page images and reports how many
// pixels differ, per page and overall.
//
// The comparison is exact by default. A tolerance exists because two PDF
// rasterisations of the same page can differ by one level in the last bit of a
// subpixel-antialiased glyph edge, and counting those as differences would bury
// the ones that matter.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
)

type pageReport struct {
	Page      int     `json:"page"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Different int     `json:"different_pixels"`
	Percent   float64 `json:"percent"`
	Note      string  `json:"note,omitempty"`
}

type report struct {
	Left       string       `json:"left"`
	Right      string       `json:"right"`
	LeftPages  int          `json:"left_pages"`
	RightPages int          `json:"right_pages"`
	Different  int          `json:"different_pixels"`
	Total      int          `json:"total_pixels"`
	Percent    float64      `json:"percent"`
	Identical  bool         `json:"identical"`
	Pages      []pageReport `json:"pages"`
}

func main() {
	tolerance := flag.Int("tolerance", 0, "per-channel difference treated as equal")
	out := flag.String("diff-dir", "", "write a diff image per differing page here")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: pixdiff [--tolerance n] [--diff-dir d] <leftDir> <rightDir>")
		os.Exit(2)
	}
	left, right := flag.Arg(0), flag.Arg(1)

	leftPages, err := pages(left)
	check(err)
	rightPages, err := pages(right)
	check(err)

	result := report{Left: left, Right: right,
		LeftPages: len(leftPages), RightPages: len(rightPages), Identical: true}

	count := len(leftPages)
	if len(rightPages) > count {
		count = len(rightPages)
	}
	if *out != "" {
		check(os.MkdirAll(*out, 0o755))
	}

	for i := 0; i < count; i++ {
		page := pageReport{Page: i + 1}
		if i >= len(leftPages) || i >= len(rightPages) {
			page.Note = "page present on one side only"
			result.Identical = false
			result.Pages = append(result.Pages, page)
			continue
		}
		a, err := load(leftPages[i])
		check(err)
		b, err := load(rightPages[i])
		check(err)
		if a.Bounds() != b.Bounds() {
			page.Note = fmt.Sprintf("different raster size: %v against %v", a.Bounds(), b.Bounds())
			result.Identical = false
			result.Pages = append(result.Pages, page)
			continue
		}
		bounds := a.Bounds()
		page.Width, page.Height = bounds.Dx(), bounds.Dy()
		var diff *image.RGBA
		if *out != "" {
			diff = image.NewRGBA(bounds)
		}
		different := 0
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				if same(a.At(x, y), b.At(x, y), *tolerance) {
					if diff != nil {
						grey := lighten(a.At(x, y))
						diff.Set(x, y, grey)
					}
					continue
				}
				different++
				if diff != nil {
					diff.Set(x, y, color.RGBA{255, 0, 0, 255})
				}
			}
		}
		page.Different = different
		total := page.Width * page.Height
		page.Percent = percent(different, total)
		result.Different += different
		result.Total += total
		if different > 0 {
			result.Identical = false
			if diff != nil {
				write(filepath.Join(*out, fmt.Sprintf("page-%02d.png", i+1)), diff)
			}
		}
		result.Pages = append(result.Pages, page)
	}
	result.Percent = percent(result.Different, result.Total)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	check(encoder.Encode(result))
}

func percent(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) * 100 / float64(whole)
}

func same(a, b color.Color, tolerance int) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	limit := uint32(tolerance) * 257
	return within(ar, br, limit) && within(ag, bg, limit) &&
		within(ab, bb, limit) && within(aa, ba, limit)
}

func within(a, b, limit uint32) bool {
	if a > b {
		return a-b <= limit
	}
	return b-a <= limit
}

// lighten fades the unchanged pixels, so the red of a difference reads at a
// glance against a ghost of the page it sits on.
func lighten(c color.Color) color.RGBA {
	r, g, b, _ := c.RGBA()
	fade := func(v uint32) uint8 { return uint8(255 - (255-v/257)/4) }
	return color.RGBA{fade(r), fade(g), fade(b), 255}
}

func pages(dir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.png"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

func load(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return png.Decode(file)
}

func write(path string, img image.Image) {
	file, err := os.Create(path)
	check(err)
	defer file.Close()
	check(png.Encode(file, img))
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "pixdiff:", err)
		os.Exit(1)
	}
}
