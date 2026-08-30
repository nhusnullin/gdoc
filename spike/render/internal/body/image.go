package body

// Embedding a picture: read the bytes, work out how big it wants to be, add the
// media part and its relationship, then write the DrawingML that points at it.
//
// python-docx owns all of this behind run.add_picture. Doing it by hand is the
// one place where the Go port carries real weight the Python one does not, and
// it is also the one place where a wrong answer is visible on the page: the size
// comes from the image's own DPI, so a picture whose pHYs chunk is misread comes
// out at the wrong scale.

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/beevik/etree"

	"spike/gdocgo/internal/docx"
	"spike/gdocgo/internal/ooxml"
)

// 1 inch is 1440 twips and 914400 EMU, so one twip is exactly 635 EMU. An image
// is never widened past the text column, and never upscaled past its own size.
const (
	twipEMU     = 635
	emuPerInch  = 914400
	imageMaxEMU = ooxml.UsableTwips * twipEMU
	// python-docx falls back to 72 dpi whenever the file declares none, and a
	// picture pasted into a Google Doc usually declares none. Matching that
	// fallback is what keeps the two renderers the same size on the page.
	defaultDPI = 72
)

type imageInfo struct {
	data      []byte
	extension string // "png", "jpeg", ...
	contentTy string
	pxWidth   int
	pxHeight  int
	horzDPI   int
	vertDPI   int
}

// widthEMU and heightEMU are the picture's native size, which is what
// python-docx uses when add_picture is given no explicit dimensions.
func (i imageInfo) widthEMU() int64 {
	return int64(math.Round(float64(i.pxWidth) / float64(i.horzDPI) * emuPerInch))
}

func (i imageInfo) heightEMU() int64 {
	return int64(math.Round(float64(i.pxHeight) / float64(i.vertDPI) * emuPerInch))
}

func identify(data []byte) (imageInfo, error) {
	switch {
	case len(data) > 8 && string(data[1:4]) == "PNG":
		return identifyPNG(data)
	case len(data) > 3 && data[0] == 0xFF && data[1] == 0xD8:
		return identifyJPEG(data)
	case len(data) > 6 && string(data[:3]) == "GIF":
		return identifyGIF(data)
	}
	return imageInfo{}, fmt.Errorf("the picture is in a format this build does not read " +
		"(PNG, JPEG and GIF are handled)")
}

func identifyPNG(data []byte) (imageInfo, error) {
	info := imageInfo{data: data, extension: "png", contentTy: "image/png",
		horzDPI: defaultDPI, vertDPI: defaultDPI}
	offset := 8
	for offset+8 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[offset:]))
		kind := string(data[offset+4 : offset+8])
		payload := offset + 8
		if payload+length > len(data) {
			break
		}
		switch kind {
		case "IHDR":
			if length < 8 {
				return info, fmt.Errorf("the PNG header is truncated")
			}
			info.pxWidth = int(binary.BigEndian.Uint32(data[payload:]))
			info.pxHeight = int(binary.BigEndian.Uint32(data[payload+4:]))
		case "pHYs":
			// Units 1 means pixels per metre. Anything else declares no
			// physical size at all, so 72 dpi stands.
			if length >= 9 && data[payload+8] == 1 {
				horizontal := binary.BigEndian.Uint32(data[payload:])
				vertical := binary.BigEndian.Uint32(data[payload+4:])
				if horizontal > 0 {
					info.horzDPI = int(math.Round(float64(horizontal) * 0.0254))
				}
				if vertical > 0 {
					info.vertDPI = int(math.Round(float64(vertical) * 0.0254))
				}
			}
		case "IEND":
			offset = len(data)
			continue
		}
		offset = payload + length + 4
	}
	if info.pxWidth == 0 || info.pxHeight == 0 {
		return info, fmt.Errorf("the PNG declares no size")
	}
	return info, nil
}

func identifyJPEG(data []byte) (imageInfo, error) {
	info := imageInfo{data: data, extension: "jpeg", contentTy: "image/jpeg",
		horzDPI: defaultDPI, vertDPI: defaultDPI}
	offset := 2
	for offset+4 <= len(data) {
		if data[offset] != 0xFF {
			offset++
			continue
		}
		marker := data[offset+1]
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			offset += 2
			continue
		}
		length := int(binary.BigEndian.Uint16(data[offset+2:]))
		payload := offset + 4
		switch {
		case marker == 0xE0 && payload+12 <= len(data) && string(data[payload:payload+4]) == "JFIF":
			// density units: 1 dots per inch, 2 dots per cm, 0 none.
			units := data[payload+7]
			horizontal := int(binary.BigEndian.Uint16(data[payload+8:]))
			vertical := int(binary.BigEndian.Uint16(data[payload+10:]))
			if units == 1 && horizontal > 0 && vertical > 0 {
				info.horzDPI, info.vertDPI = horizontal, vertical
			} else if units == 2 && horizontal > 0 && vertical > 0 {
				info.horzDPI = int(math.Round(float64(horizontal) * 2.54))
				info.vertDPI = int(math.Round(float64(vertical) * 2.54))
			}
		case marker >= 0xC0 && marker <= 0xCF && marker != 0xC4 && marker != 0xC8 && marker != 0xCC:
			if payload+5 > len(data) {
				return info, fmt.Errorf("the JPEG frame header is truncated")
			}
			info.pxHeight = int(binary.BigEndian.Uint16(data[payload+1:]))
			info.pxWidth = int(binary.BigEndian.Uint16(data[payload+3:]))
		}
		offset = offset + 2 + length
	}
	if info.pxWidth == 0 || info.pxHeight == 0 {
		return info, fmt.Errorf("the JPEG declares no size")
	}
	return info, nil
}

func identifyGIF(data []byte) (imageInfo, error) {
	if len(data) < 10 {
		return imageInfo{}, fmt.Errorf("the GIF is truncated")
	}
	return imageInfo{
		data: data, extension: "gif", contentTy: "image/gif",
		pxWidth:  int(binary.LittleEndian.Uint16(data[6:])),
		pxHeight: int(binary.LittleEndian.Uint16(data[8:])),
		horzDPI:  defaultDPI, vertDPI: defaultDPI,
	}, nil
}

// --- the media part, its relationship and its content type -------------------

var (
	relationshipIDRE = regexp.MustCompile(`Id="rId(\d+)"`)
	defaultExtRE     = regexp.MustCompile(`<Default Extension="([^"]+)"`)
)

// Media adds pictures to a package: one media part, one relationship and one
// content-type default per distinct format.
type Media struct {
	pkg   *docx.Package
	next  int
	relID int
	docPr int
}

func NewMedia(pkg *docx.Package) *Media {
	m := &Media{pkg: pkg, next: 1, docPr: 1}
	rels, _ := pkg.Get("word/_rels/document.xml.rels")
	highest := 0
	for _, match := range relationshipIDRE.FindAllStringSubmatch(string(rels), -1) {
		if id, err := strconv.Atoi(match[1]); err == nil && id > highest {
			highest = id
		}
	}
	m.relID = highest + 1
	// Start after the media the template already carries, so nothing is
	// overwritten: the master's logo lives in word/media too.
	m.next = len(pkg.HasPrefix("word/media/")) + 1
	return m
}

// add writes the bytes into the package and returns the relationship id that
// points at them.
func (m *Media) add(info imageInfo) (string, string, error) {
	name := fmt.Sprintf("image%d.%s", m.next, info.extension)
	m.next++
	m.pkg.Set("word/media/"+name, info.data)

	if err := m.ensureContentType(info); err != nil {
		return "", "", err
	}

	relationshipID := fmt.Sprintf("rId%d", m.relID)
	m.relID++
	rels, ok := m.pkg.Get("word/_rels/document.xml.rels")
	if !ok {
		return "", "", fmt.Errorf("the package holds no word/_rels/document.xml.rels")
	}
	entry := fmt.Sprintf(
		`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/%s"/>`,
		relationshipID, name)
	patched := strings.Replace(string(rels), "</Relationships>", entry+"</Relationships>", 1)
	m.pkg.Set("word/_rels/document.xml.rels", []byte(patched))
	return relationshipID, name, nil
}

// ensureContentType declares the extension once. A picture whose type the
// package never declares is dropped silently by Word, which is exactly the kind
// of failure that only shows up after somebody reads the document.
func (m *Media) ensureContentType(info imageInfo) error {
	data, ok := m.pkg.Get("[Content_Types].xml")
	if !ok {
		return fmt.Errorf("the package holds no [Content_Types].xml")
	}
	for _, match := range defaultExtRE.FindAllStringSubmatch(string(data), -1) {
		if strings.EqualFold(match[1], info.extension) {
			return nil
		}
	}
	entry := fmt.Sprintf(`<Default Extension="%s" ContentType="%s"/>`,
		info.extension, info.contentTy)
	// A Default belongs immediately after the opening <Types ...> tag.
	patched := string(data)
	open := strings.Index(patched, "<Types")
	if open < 0 {
		return fmt.Errorf("[Content_Types].xml has no <Types> element")
	}
	end := strings.Index(patched[open:], ">")
	if end < 0 {
		return fmt.Errorf("[Content_Types].xml is malformed")
	}
	cut := open + end + 1
	m.pkg.Set("[Content_Types].xml", []byte(patched[:cut]+entry+patched[cut:]))
	return nil
}

func isDataURI(target string) bool {
	return strings.HasPrefix(strings.ToLower(target), "data:")
}

func isRemote(target string) bool {
	lower := strings.ToLower(target)
	for _, prefix := range []string{"http://", "https://", "//", "data:"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// decodeDataURI reads the bytes of an inline picture.
//
// Drive's markdown export writes an embedded picture as a reference-style link
// to a base64 data: URI, so the bytes arrive with the markdown. Refusing them
// sent people to --media-dir for a picture the export had already handed over.
//
// No network, no temporary file: this is decoding what is already in the file.
func decodeDataURI(target string) ([]byte, error) {
	header, payload, found := strings.Cut(target, ",")
	if !found || payload == "" || !strings.Contains(header, "base64") {
		shown := header
		if len(shown) > 40 {
			shown = shown[:40]
		}
		return nil, fmt.Errorf("an inline picture could not be read: only base64 "+
			"data: URIs are understood, and this one says %q", shown)
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimPrefix(header, "data:")), "image/") {
		shown := header
		if len(shown) > 40 {
			shown = shown[:40]
		}
		return nil, fmt.Errorf("the inline data is not a picture, it says %q", shown)
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("an inline picture could not be decoded: %w", err)
	}
	return decoded, nil
}

// Place builds the centred paragraph holding one picture.
func (m *Media) Place(target, baseDir string) (*etree.Element, error) {
	var data []byte
	var err error
	switch {
	case isDataURI(target):
		data, err = decodeDataURI(target)
		if err != nil {
			return nil, err
		}
	case isRemote(target):
		// A URL is not a picture: nothing here downloads, and a publish that
		// reached the network would depend on a link that expires.
		return nil, fmt.Errorf("the picture at %s is a link, not a file. Nothing here "+
			"downloads pictures. Pull the document again with `gdoc export --out "+
			"<note>.md --media-dir <note>-media`, which writes the pictures beside "+
			"the markdown, and publish that", target)
	default:
		path := target
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, target)
		}
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("image not found: %s, looked in %s", target, baseDir)
		}
	}

	info, err := identify(data)
	if err != nil {
		shown := target
		if isDataURI(target) {
			shown = "an inline picture"
		}
		return nil, fmt.Errorf("the image could not be embedded, %s: %w", shown, err)
	}

	relationshipID, name, err := m.add(info)
	if err != nil {
		return nil, err
	}

	width, height := info.widthEMU(), info.heightEMU()
	if width > imageMaxEMU {
		height = int64(math.Round(float64(height) * imageMaxEMU / float64(width)))
		width = imageMaxEMU
	}

	docPrID := m.docPr
	m.docPr++
	return pictureParagraph(relationshipID, name, docPrID, width, height), nil
}

// pictureParagraph writes the DrawingML python-docx generates, with the three
// namespaces declared inline. Declaring them here rather than trusting the
// root's nsmap is deliberate: the bundled master is a Google Docs export, and
// which namespaces it happens to declare is not something to depend on.
func pictureParagraph(relationshipID, name string, docPrID int, width, height int64) *etree.Element {
	const (
		nsWP  = "http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
		nsA   = "http://schemas.openxmlformats.org/drawingml/2006/main"
		nsPic = "http://schemas.openxmlformats.org/drawingml/2006/picture"
		nsR   = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	)
	paragraph := etree.NewElement("w:p")
	pPr := paragraph.CreateElement("w:pPr")
	ooxml.Sub(pPr, "w:jc", "val", "center")

	run := paragraph.CreateElement("w:r")
	drawing := run.CreateElement("w:drawing")

	inline := drawing.CreateElement("wp:inline")
	inline.CreateAttr("xmlns:wp", nsWP)
	inline.CreateAttr("distT", "0")
	inline.CreateAttr("distB", "0")
	inline.CreateAttr("distL", "0")
	inline.CreateAttr("distR", "0")

	extent := inline.CreateElement("wp:extent")
	extent.CreateAttr("cx", strconv.FormatInt(width, 10))
	extent.CreateAttr("cy", strconv.FormatInt(height, 10))

	docPr := inline.CreateElement("wp:docPr")
	docPr.CreateAttr("id", strconv.Itoa(docPrID))
	docPr.CreateAttr("name", fmt.Sprintf("Picture %d", docPrID))
	docPr.CreateAttr("descr", "")

	framePr := inline.CreateElement("wp:cNvGraphicFramePr")
	locks := framePr.CreateElement("a:graphicFrameLocks")
	locks.CreateAttr("xmlns:a", nsA)
	locks.CreateAttr("noChangeAspect", "1")

	graphic := inline.CreateElement("a:graphic")
	graphic.CreateAttr("xmlns:a", nsA)
	graphicData := graphic.CreateElement("a:graphicData")
	graphicData.CreateAttr("uri", nsPic)

	pic := graphicData.CreateElement("pic:pic")
	pic.CreateAttr("xmlns:pic", nsPic)

	nvPicPr := pic.CreateElement("pic:nvPicPr")
	cNvPr := nvPicPr.CreateElement("pic:cNvPr")
	cNvPr.CreateAttr("id", strconv.Itoa(docPrID))
	cNvPr.CreateAttr("name", name)
	cNvPr.CreateAttr("descr", "")
	nvPicPr.CreateElement("pic:cNvPicPr")

	blipFill := pic.CreateElement("pic:blipFill")
	blip := blipFill.CreateElement("a:blip")
	blip.CreateAttr("xmlns:r", nsR)
	blip.CreateAttr("r:embed", relationshipID)
	blipFill.CreateElement("a:stretch").CreateElement("a:fillRect")

	spPr := pic.CreateElement("pic:spPr")
	xfrm := spPr.CreateElement("a:xfrm")
	off := xfrm.CreateElement("a:off")
	off.CreateAttr("x", "0")
	off.CreateAttr("y", "0")
	ext := xfrm.CreateElement("a:ext")
	ext.CreateAttr("cx", strconv.FormatInt(width, 10))
	ext.CreateAttr("cy", strconv.FormatInt(height, 10))
	prstGeom := spPr.CreateElement("a:prstGeom")
	prstGeom.CreateAttr("prst", "rect")
	prstGeom.CreateElement("a:avLst")

	return paragraph
}
