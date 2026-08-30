package render

import "unicode"

func unicodeCategory(name string) *unicode.RangeTable {
	return unicode.Categories[name]
}
