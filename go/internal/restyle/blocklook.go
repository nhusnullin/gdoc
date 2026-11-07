// This file is the house look restyle hands to propose, so a paragraph a block
// proposal adds looks like one a restyle styled.

package restyle

import (
	"gdoc/internal/docsreq"
	"gdoc/internal/house"
	"gdoc/internal/propose"
)

// BlockLook is the look a block's new paragraphs take: for each named style,
// the paragraph properties and the run properties a restyle writes over a
// paragraph of that style.
//
// It is restyle's own paragraphLook and textLook, not a second copy of them,
// because a proposed paragraph that matched a restyled one by coincidence would
// stop matching the first time somebody changed one of the two. Body text comes
// out at the body size, the body alignment and the body spacing rather than the
// Normal style's 11pt, which is Nail's rule of 2026-11-07: body text is 12pt
// whatever gdoc is doing. A style outside houseStyleKey gets no look, as a
// restyle leaves it alone. TestBlockLookIsTheHouseBodyLook is the pin.
func BlockLook(cfg *house.Config) propose.Look {
	return func(style string, bulleted, lastInList bool) (*docsreq.Fields, *docsreq.Fields) {
		key, ok := houseStyleKey[style]
		if !ok {
			return nil, nil
		}
		s := cfg.Styles[key]
		return paragraphLook(s, key, cfg, bulleted, lastInList), textLook(s, key, cfg)
	}
}
