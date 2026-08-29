package body

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func pngBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.Black)
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestAPNGWithNoDensityIsReadAt72DPI(t *testing.T) {
	// python-docx falls back to 72 for a file that declares none, and a picture
	// pasted into a Google Doc usually declares none. Matching that fallback is
	// what keeps the two renderers the same size on the page.
	info, err := identify(pngBytes(t, 640, 360))
	if err != nil {
		t.Fatal(err)
	}

	if info.pxWidth != 640 || info.pxHeight != 360 {
		t.Errorf("size = %dx%d, want 640x360", info.pxWidth, info.pxHeight)
	}
	if info.horzDPI != 72 || info.vertDPI != 72 {
		t.Errorf("dpi = %dx%d, want 72x72", info.horzDPI, info.vertDPI)
	}
	// 640px at 72dpi is 8.888in, and one inch is 914400 EMU.
	if got := info.widthEMU(); got != 8128000 {
		t.Errorf("width = %d EMU, want 8128000", got)
	}
}

func TestAPNGDeclaringItsDensityIsReadAtThatDensity(t *testing.T) {
	// pHYs units 1 is pixels per metre. 300 dpi is 11811 per metre.
	base := pngBytes(t, 300, 300)
	withPhys := insertPHYs(t, base, 11811, 11811)

	info, err := identify(withPhys)
	if err != nil {
		t.Fatal(err)
	}

	if info.horzDPI != 300 {
		t.Errorf("dpi = %d, want 300 read out of the pHYs chunk", info.horzDPI)
	}
	if got := info.widthEMU(); got != 914400 {
		t.Errorf("width = %d EMU, want one inch", got)
	}
}

func TestAJPEGSizeComesFromItsFrameHeader(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 900, 300))
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, img, nil); err != nil {
		t.Fatal(err)
	}

	info, err := identify(buffer.Bytes())
	if err != nil {
		t.Fatalf("a JPEG with no JFIF segment must still be read: %v", err)
	}

	if info.pxWidth != 900 || info.pxHeight != 300 {
		t.Errorf("size = %dx%d, want 900x300", info.pxWidth, info.pxHeight)
	}
	if info.extension != "jpeg" {
		t.Errorf("extension = %q", info.extension)
	}
}

func TestAnUnknownFormatIsRefusedByName(t *testing.T) {
	_, err := identify([]byte("not a picture at all"))

	if err == nil {
		t.Fatal("an unreadable picture must be refused, not embedded blind")
	}
}

func TestARemoteImageIsRefusedRatherThanDownloaded(t *testing.T) {
	// A publish that reached the network would depend on a link that expires.
	if !isRemote("https://example.com/a.png") {
		t.Error("an https target must be treated as remote")
	}
	if isRemote("diagram.png") {
		t.Error("a relative path is not remote")
	}
}

func TestADataURIMustDeclareItselfAsABase64Image(t *testing.T) {
	if _, err := decodeDataURI("data:text/plain;base64,aGk="); err == nil {
		t.Error("a data URI that is not an image must be refused")
	}
	if _, err := decodeDataURI("data:image/png,notbase64"); err == nil {
		t.Error("a data URI that is not base64 must be refused")
	}
	decoded, err := decodeDataURI("data:image/png;base64,aGk=")
	if err != nil || string(decoded) != "hi" {
		t.Errorf("decoded = %q, err = %v", decoded, err)
	}
}

// insertPHYs splices a pHYs chunk in after the IHDR, so a test can make a PNG
// that declares its own density without carrying a binary fixture.
func insertPHYs(t *testing.T, data []byte, x, y uint32) []byte {
	t.Helper()
	const headerEnd = 8 + 4 + 4 + 13 + 4 // signature, then the whole IHDR chunk
	payload := []byte{
		byte(x >> 24), byte(x >> 16), byte(x >> 8), byte(x),
		byte(y >> 24), byte(y >> 16), byte(y >> 8), byte(y),
		1,
	}
	chunk := []byte{0, 0, 0, 9, 'p', 'H', 'Y', 's'}
	chunk = append(chunk, payload...)
	chunk = append(chunk, 0, 0, 0, 0) // the CRC is never checked here
	out := make([]byte, 0, len(data)+len(chunk))
	out = append(out, data[:headerEnd]...)
	out = append(out, chunk...)
	return append(out, data[headerEnd:]...)
}
