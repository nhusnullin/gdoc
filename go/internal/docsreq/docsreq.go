// Package docsreq holds the shapes a Google Docs request is made of: a
// measurement in points, a colour, an alignment, a length in the units the API
// counts, and a style object carrying the field mask that names it.
//
// Two packages build requests. internal/restyle states the house look on a
// document where it stands, and internal/prelude proposes the house template
// into one. Each rule below was decided once and costs something when it
// drifts: the line spacing rounding is what makes a request comparable with the
// value in house.yaml, the colour conversion is what keeps Docs from rejecting
// a batch, and a mask built beside the object it names is what stops a field
// being reset because one of the two was edited and the other was not.
//
// The house style stores every measurement in points and the Docs API's only
// unit is PT, so nothing here converts anything: the twips, half-points and EMU
// the docx writer computes live in internal/render and stay there.
//
// Nothing here reaches the network and nothing here reads a file.
package docsreq

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf16"
)

// alignments maps the house style's own word onto the Docs alignment. A word
// outside this map is not written, the way every other unstated value is not.
var alignments = map[string]string{
	"left": "START", "center": "CENTER", "centre": "CENTER",
	"right": "END", "justify": "JUSTIFIED",
}

// Alignment is one house alignment as the Docs API spells it, and whether the
// API has it at all. An empty word is not an alignment: it is a file saying
// nothing, and nothing is what gets written.
func Alignment(word string) (string, bool) {
	v, ok := alignments[strings.ToLower(strings.TrimSpace(word))]
	return v, ok
}

// Points is one Docs Dimension.
func Points(v float64) map[string]any {
	return map[string]any{"magnitude": v, "unit": "PT"}
}

// Percent is a house line spacing as the Docs API takes it: the file states a
// multiplier and the API a percentage of normal. It is rounded to three places
// because 1.15 times 100 is 114.99999999999999 in binary floating point, and a
// request nobody can match to the value in the file is one nobody can check.
func Percent(multiplier float64) float64 {
	return math.Round(multiplier*100*1000) / 1000
}

// Color is one house colour as the Docs API takes it, wrapped as the
// OptionalColor every colour field on a style is. A value that is not a
// six-digit hex colour is not written: house.yaml states every colour that way,
// and writing a field the guard would carry but Docs would reject costs the
// whole batch.
func Color(hex string) (map[string]any, bool) {
	h := strings.TrimPrefix(hex, "#")
	if len(h) != 6 {
		return nil, false
	}
	channels := make([]float64, 3)
	for i := range channels {
		v, err := strconv.ParseUint(h[i*2:i*2+2], 16, 8)
		if err != nil {
			return nil, false
		}
		channels[i] = float64(v) / 255.0
	}
	return map[string]any{"color": map[string]any{"rgbColor": map[string]any{
		"red": channels[0], "green": channels[1], "blue": channels[2],
	}}}, true
}

// Len is how long a string is in the units the Docs API counts, which are
// UTF-16 code units. A rune outside the basic plane is two of them, so a
// caller that counted bytes or runes would name a position the server reads
// somewhere else.
func Len(s string) int { return len(utf16.Encode([]rune(s))) }

// Fields is a style object and the mask that names it, built together.
//
// Together, because the two are one statement. A mask assembled from the map
// afterwards has no order, so a request would read differently on each run and
// nobody could compare one against a log; and a mask written out by hand beside
// the map is the drift the mask rule is most likely to grow, where a field
// added to one is forgotten in the other and the property it names is reset.
type Fields struct {
	// Set is the style object itself, keyed the way the API keys it.
	Set   map[string]any
	names []string
}

// NewFields is an empty style object, setting nothing.
func NewFields() *Fields { return &Fields{Set: map[string]any{}} }

// Put states one field, and names it in the mask.
func (f *Fields) Put(name string, v any) {
	f.Set[name] = v
	f.names = append(f.names, name)
}

// Mask is the field mask, in the order the fields were stated.
func (f *Fields) Mask() string { return strings.Join(f.names, ",") }

// Empty says whether nothing was stated. A request setting nothing is a
// request that resets nothing, and it is not worth sending.
func (f *Fields) Empty() bool { return len(f.names) == 0 }
