package render

import (
	"gdoc/internal/house"
	"github.com/beevik/etree"
)

// The relationship the first page's header names for the logo. It lives in
// word/_rels/header2.xml.rels, which holds that one relationship and nothing
// else.
const logoRelID = "rId1"

// headerPart writes one header. logo says whether this region may carry the
// mark, which only the first page's does.
func (b *builder) headerPart(region house.Region, logo bool, name string) []byte {
	doc, root := newPart("w:hdr")
	for _, spec := range region.Paragraphs {
		root.AddChild(b.regionParagraph(spec, region, logo))
	}
	return b.serialise(doc, name)
}

// footerPart writes one footer. A footer carries no logo, and it takes its
// indent from the region rather than the paragraph, which is what the house
// file states.
func (b *builder) footerPart(region house.Region, name string) []byte {
	doc, root := newPart("w:ftr")
	for _, spec := range region.Paragraphs {
		p := b.para(paraOpts{
			Before:      spec.SpaceBeforePt,
			After:       spec.SpaceAfterPt,
			Line:        lineOr(spec.LineSpacing, region.LineSpacing),
			IndentStart: region.IndentStartPt,
			Mark:        regionMark(spec),
		})
		b.regionRuns(p, spec)
		root.AddChild(p)
	}
	return b.serialise(doc, name)
}

// regionParagraph is one line of a header: a rule, a row of runs, or a row of
// runs with the logo anchored beside them.
func (b *builder) regionParagraph(spec house.Paragraph, region house.Region, logo bool) *etree.Element {
	align := spec.Align
	if align == "" {
		align = region.Align
	}
	p := b.para(paraOpts{
		Align:  align,
		Before: spec.SpaceBeforePt,
		After:  spec.SpaceAfterPt,
		Line:   lineOr(spec.LineSpacing, region.LineSpacing),
		Mark:   regionMark(spec),
	})
	if spec.Rule != nil {
		b.ruleRun(p, *spec.Rule)
		return p
	}
	b.regionRuns(p, spec)
	if spec.Logo {
		if !logo {
			b.fail("the logo is anchored in a header that is not the first page's")
			return p
		}
		b.logoDrawing(p)
	}
	return p
}

// regionMark is the paragraph mark's own run properties, which is where a
// header or footer paragraph's own size and colour belong.
//
// It is not decoration. An empty line's height is its paragraph mark's size,
// so the two 9pt lines under the running head and the 12pt line above the
// footer text fell back to the document's 11pt default and the block came out
// taller than the master's. The house file states those sizes on the
// paragraph, and the master carries them on the paragraph mark.
func regionMark(spec house.Paragraph) *runOpts {
	o := runOpts{SizePt: spec.SizePt, Color: spec.Color}
	if o.empty() {
		return nil
	}
	return &o
}

// regionRuns writes a header or footer paragraph's runs: words, tabs, or the
// page number as a field. A run naming a placeholder the note filled carries
// the note's own words instead of the template's.
func (b *builder) regionRuns(p *etree.Element, spec house.Paragraph) {
	for _, r := range spec.Runs {
		o := runOpts{SizePt: r.SizePt, Color: r.Color, Bold: r.Bold}
		switch {
		case r.PageNumber:
			pageFieldRun(p, o)
		case r.Tabs > 0:
			tabsRun(p, r.Tabs, o)
		case r.Tab:
			tabsRun(p, 1, runOpts{})
		default:
			textRun(p, b.text(r.Text, r.Placeholder), o)
		}
	}
}

// ruleRun is the horizontal line under the running head. Word draws it as a
// VML rectangle, which is the one place this generator writes VML at all.
func (b *builder) ruleRun(p *etree.Element, rule house.Rule) {
	r := sub(p, "w:r")
	pict := sub(r, "w:pict")
	rect := sub(pict, "v:rect")
	rect.CreateAttr("style", "width:0.0pt;height:"+trimFloat(rule.HeightPt)+"pt")
	rect.CreateAttr("o:hr", "t")
	rect.CreateAttr("o:hrstd", "t")
	rect.CreateAttr("o:hralign", "center")
	rect.CreateAttr("fillcolor", rule.Color)
	rect.CreateAttr("stroked", "f")
}

// logoDrawing anchors the mark where the house file puts it. The offsets are
// the config's points in EMU, and the wrap is the config's too: a wrap this
// generator does not write is refused by name rather than guessed at.
func (b *builder) logoDrawing(p *etree.Element) {
	lg := b.cfg.Logo
	a := lg.Anchor
	if a.Wrap != "square" {
		b.fail("logo: wrap %q is not one this generator writes", a.Wrap)
		return
	}
	dist := emu(a.WrapDistPt)

	r := sub(p, "w:r")
	drawing := sub(r, "w:drawing")
	anchor := sub(drawing, "wp:anchor",
		"distT", dist, "distB", dist, "distL", dist, "distR", dist,
		"simplePos", "0", "relativeHeight", "0", "behindDoc", "0",
		"locked", "0", "layoutInCell", "1", "allowOverlap", "1")
	sub(anchor, "wp:simplePos", "x", "0", "y", "0")

	h := sub(anchor, "wp:positionH", "relativeFrom", a.RelativeH)
	sub(h, "wp:posOffset").SetText(emu(a.OffsetXPt))
	v := sub(anchor, "wp:positionV", "relativeFrom", a.RelativeV)
	sub(v, "wp:posOffset").SetText(emu(a.OffsetYPt))

	sub(anchor, "wp:extent", "cx", emu(lg.WidthPt), "cy", emu(lg.HeightPt))
	sub(anchor, "wp:effectExtent", "l", "0", "t", "0", "r", "0", "b", "0")
	sub(anchor, "wp:wrapSquare", "wrapText", "bothSides",
		"distT", dist, "distB", dist, "distL", dist, "distR", dist)
	sub(anchor, "wp:docPr", "id", "18", "name", "logo.png")

	graphic := sub(anchor, "a:graphic")
	data := sub(graphic, "a:graphicData",
		"uri", "http://schemas.openxmlformats.org/drawingml/2006/picture")
	pic := sub(data, "pic:pic")
	nv := sub(pic, "pic:nvPicPr")
	sub(nv, "pic:cNvPr", "id", "0", "name", "logo.png")
	sub(nv, "pic:cNvPicPr", "preferRelativeResize", "0")
	fill := sub(pic, "pic:blipFill")
	sub(fill, "a:blip", "r:embed", logoRelID)
	sub(fill, "a:srcRect", "b", "0", "l", "0", "r", "0", "t", "0")
	sub(sub(fill, "a:stretch"), "a:fillRect")
	spPr := sub(pic, "pic:spPr")
	xfrm := sub(spPr, "a:xfrm")
	sub(xfrm, "a:off", "x", "0", "y", "0")
	sub(xfrm, "a:ext", "cx", emu(lg.WidthPt), "cy", emu(lg.HeightPt))
	sub(spPr, "a:prstGeom", "prst", "rect")
	sub(spPr, "a:ln")
}

// lineOr takes the paragraph's own line spacing, else the region's.
func lineOr(own, region *float64) *float64 {
	if own != nil {
		return own
	}
	return region
}
