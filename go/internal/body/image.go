package body

// Embedding a picture: read the bytes, work out how big it wants to be, hand
// the part and its relationship back to the caller, then write the DrawingML
// that points at it.
//
// The size comes from the image's own DPI, so a picture whose pHYs chunk is
// misread comes out at the wrong scale. That is the one place here where a
// wrong answer is visible on the page.
//
// Nothing here downloads. A relative path is read from the note's own
// directory and a data: URI is decoded, because those bytes arrived with the
// markdown. An http address is refused naming the line: a publish that reached
// the network would depend on a link that expires.

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

const (
	// 1 inch is 914400 EMU and 72 points, so one point is exactly 12700 EMU.
	emuPerPoint = 12700
	emuPerInch  = 914400
	// python-docx falls back to 72 dpi whenever the file declares none, and a
	// picture pasted into a Google Doc usually declares none. Matching that
	// fallback is what keeps v1 and v2 the same size on the page.
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

// widthEMU and heightEMU are the picture's native size, which is the size v1
// gives a picture that names no dimensions.
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
	}
	return imageInfo{}, fmt.Errorf("the picture is in a format the house style " +
		"does not carry, and only PNG and JPEG are read")
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
				// The test is on the converted value, not on the declared one.
				// A pHYs under about 20 pixels per metre is non-zero and still
				// rounds to nought dpi, and widthEMU divides by it: the extent
				// goes to infinity and lands in the XML as a negative integer
				// Word offers to repair. Below one dpi the default stands.
				if dpi := int(math.Round(float64(horizontal) * 0.0254)); dpi > 0 {
					info.horzDPI = dpi
				}
				if dpi := int(math.Round(float64(vertical) * 0.0254)); dpi > 0 {
					info.vertDPI = dpi
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

// --- reading one picture -----------------------------------------------------

func isDataURI(target string) bool {
	return strings.HasPrefix(strings.ToLower(target), "data:")
}

func isRemote(target string) bool {
	lower := strings.ToLower(target)
	for _, prefix := range []string{"http://", "https://", "//"} {
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
// sent people looking for a file the export had already handed over.
//
// No network and no temporary file: this is decoding what is already in the
// note.
func decodeDataURI(target string) ([]byte, error) {
	header, payload, found := strings.Cut(target, ",")
	if !found || payload == "" || !strings.Contains(header, "base64") {
		return nil, fmt.Errorf("an inline picture could not be read: only base64 "+
			"data: URIs are understood, and this one says %q", shorten(header))
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimPrefix(header, "data:")), "image/") {
		return nil, fmt.Errorf("the inline data is not a picture, it says %q", shorten(header))
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("an inline picture could not be decoded: %w", err)
	}
	return decoded, nil
}

// shorten keeps a refusal readable when the offending value is a whole data
// URI.
func shorten(value string) string {
	if len(value) > 40 {
		return value[:40]
	}
	return value
}

// imagePath is the file a picture names, resolved against the note's own
// directory. It says nothing about whether that file is there, and it is empty
// for a picture that is not a file at all: a data URI came with the markdown,
// a link is refused rather than downloaded, and a relative target with no
// directory to resolve it against cannot be placed.
//
// It is what readImage reads from and what warnImages records, so a picture the
// walk left out is still a file the run must not write over.
func imagePath(target, base string) string {
	if isDataURI(target) || isRemote(target) {
		return ""
	}
	if filepath.IsAbs(target) {
		return target
	}
	if base == "" {
		return ""
	}
	return filepath.Join(base, target)
}

// readImage is the bytes of one picture, however the note names it, and the
// file they came from. A picture written inline as a data URI came with the
// markdown and has no file, so the path is empty for one.
func readImage(target, base string, line int) ([]byte, string, error) {
	switch {
	case isDataURI(target):
		data, err := decodeDataURI(target)
		return data, "", err
	case isRemote(target):
		return nil, "", fmt.Errorf("line %d: the picture at %s is a link, not a file, "+
			"and nothing here downloads. Pull the document again with its pictures "+
			"beside the markdown and publish that", line, target)
	}
	path := imagePath(target, base)
	if path == "" {
		return nil, "", fmt.Errorf("line %d: the note has a picture (%s) and no "+
			"directory to resolve it against", line, target)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("line %d: the picture %s could not be read, looked in %s",
			line, target, base)
	}
	return data, path, nil
}

// place reads one picture and returns the centred paragraph that holds it,
// with the media part it needs. A picture wider than the text column is scaled
// down to it, and never scaled up.
func (r *renderer) place(target string, line int) (*etree.Element, error) {
	data, path, err := readImage(target, r.base, line)
	if err != nil {
		return nil, err
	}
	if path != "" {
		r.sources = append(r.sources, path)
	}
	info, err := identify(data)
	if err != nil {
		shown := target
		if isDataURI(target) {
			shown = "an inline picture"
		}
		return nil, fmt.Errorf("line %d: %s could not be embedded: %w", line, shown, err)
	}

	r.imageCount++
	name := fmt.Sprintf("image%d.%s", r.imageCount, info.extension)
	relID := r.nextRelID()
	r.media = append(r.media, Media{RelID: relID, Name: name, Data: info.data})

	width, height := info.widthEMU(), info.heightEMU()
	maxWidth := int64(math.Round(r.cfg.Page.UsableWidthPt() * emuPerPoint))
	if width > maxWidth {
		height = int64(math.Round(float64(height) * float64(maxWidth) / float64(width)))
		width = maxWidth
	}

	r.docPr++
	return r.pictureParagraph(relID, name, r.docPr, width, height), nil
}

// pictureParagraph writes the DrawingML an inline picture is made of, with the
// three namespaces declared on the elements that need them. The part's root
// declares them too; declaring them here as well is what a .docx does, and it
// costs nothing.
func (r *renderer) pictureParagraph(relID, name string, docPrID int, width, height int64) *etree.Element {
	const (
		nsWP  = "http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
		nsA   = "http://schemas.openxmlformats.org/drawingml/2006/main"
		nsPic = "http://schemas.openxmlformats.org/drawingml/2006/picture"
		nsR   = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	)
	before, after := 0.0, r.cfg.Body.SpaceAfterPt
	p := r.para(paraOpts{Align: "center", BeforePt: &before, AfterPt: &after})

	drawing := sub(sub(p, "w:r"), "w:drawing")
	inline := sub(drawing, "wp:inline",
		"xmlns:wp", nsWP, "distT", "0", "distB", "0", "distL", "0", "distR", "0")
	sub(inline, "wp:extent",
		"cx", strconv.FormatInt(width, 10), "cy", strconv.FormatInt(height, 10))
	sub(inline, "wp:docPr",
		"id", strconv.Itoa(docPrID), "name", fmt.Sprintf("Picture %d", docPrID), "descr", "")
	locks := sub(sub(inline, "wp:cNvGraphicFramePr"), "a:graphicFrameLocks")
	locks.CreateAttr("xmlns:a", nsA)
	locks.CreateAttr("noChangeAspect", "1")

	graphic := sub(inline, "a:graphic", "xmlns:a", nsA)
	graphicData := sub(graphic, "a:graphicData", "uri", nsPic)
	pic := sub(graphicData, "pic:pic", "xmlns:pic", nsPic)

	nvPicPr := sub(pic, "pic:nvPicPr")
	sub(nvPicPr, "pic:cNvPr", "id", strconv.Itoa(docPrID), "name", name, "descr", "")
	sub(nvPicPr, "pic:cNvPicPr")

	blipFill := sub(pic, "pic:blipFill")
	sub(blipFill, "a:blip", "xmlns:r", nsR, "r:embed", relID)
	sub(sub(blipFill, "a:stretch"), "a:fillRect")

	spPr := sub(pic, "pic:spPr")
	xfrm := sub(spPr, "a:xfrm")
	sub(xfrm, "a:off", "x", "0", "y", "0")
	sub(xfrm, "a:ext",
		"cx", strconv.FormatInt(width, 10), "cy", strconv.FormatInt(height, 10))
	sub(sub(spPr, "a:prstGeom", "prst", "rect"), "a:avLst")
	return p
}
