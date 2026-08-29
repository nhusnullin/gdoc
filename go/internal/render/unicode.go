package render

import "golang.org/x/text/unicode/rangetable"

// The combining marks NFKD leaves behind once an accented letter is decomposed.
// Dropping them is what turns "Résumé" into "resume" rather than into "rsum".
var unicodeMarks = rangetable.Merge(
	unicodeCategory("Mn"), unicodeCategory("Me"), unicodeCategory("Mc"))
