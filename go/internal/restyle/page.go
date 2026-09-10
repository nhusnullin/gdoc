package restyle

import (
	"gdoc/internal/docsreq"
	"gdoc/internal/house"
)

// PageRequest is the page: one updateDocumentStyle carrying the house style's
// A4 geometry and its four margins. It is the only styling request a restyle
// sends that names no range, because a document has one page geometry.
//
// It is a pure function of the house style. Nothing here reads the document,
// because nothing needs to: the mask below names five fields and the request
// sets all five, so what is there before the write does not change what is
// written.
//
// The mask names exactly what the request sets, and that rule is this
// milestone's, not this function's. The Docs reference: "To reset a property
// to its default value, include its field name in the field mask but leave the
// field itself unset." So a path in the mask the request leaves unset is a
// property destroyed, and a field set outside the mask is a value the server
// ignores. The guard refuses a star and refuses an empty mask, and the builder
// never relies on being refused: a request that reached the guard's refusal
// would be a run that failed rather than a run that wrote the right thing.
//
// The header and footer margins are deliberately not set. gdoc creates no
// header and no footer here, and the first-page header carrying the logo is one
// of the three things this milestone reports as unreachable, so moving the
// margins the document reserves for its own header and footer would be styling
// something gdoc is not writing. The two header toggles are refused by the
// guard for the same reason: switching useFirstPageHeaderFooter off hides
// exactly that header.
//
// A document carrying section breaks has its margins governed by its
// sectionStyle, and updateSectionStyle is not on the in-place allowlist. So
// this request can be accepted and change nothing a reader sees. That is a fact
// the read-back reports rather than something this builder can fix: a restyle
// must not claim the page was restyled when the document's own sections
// overrode it.
func PageRequest(cfg *house.Config) map[string]any {
	p := cfg.Page
	return map[string]any{
		"updateDocumentStyle": map[string]any{
			"documentStyle": map[string]any{
				"pageSize": map[string]any{
					"width":  points(p.WidthPt),
					"height": points(p.HeightPt),
				},
				"marginTop":    points(p.MarginTopPt),
				"marginBottom": points(p.MarginBottomPt),
				"marginLeft":   points(p.MarginLeftPt),
				"marginRight":  points(p.MarginRightPt),
			},
			"fields": pageMask,
		},
	}
}

// pageMask is the five fields PageRequest sets, written out in the order the
// request builds them. It is a constant rather than something built from the
// map above, because a mask assembled from a map has no order and a request
// that reads differently on each run is one nobody can compare against a log.
const pageMask = "pageSize,marginTop,marginBottom,marginLeft,marginRight"

// points is one Docs Dimension. The house style stores every measurement in
// points and the Docs API's only unit is PT, so this milestone converts
// nothing: the twips, half-points and EMU the docx writer computes live in
// internal/render and stay there. The shape is docsreq's, because
// internal/prelude states the same measurements on what it proposes.
func points(v float64) map[string]any { return docsreq.Points(v) }
