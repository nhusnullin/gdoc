// Package generate turns approved markdown into a new Google Doc.
//
// Drive converts the .docx into a native Doc on create. The template path
// uploads twice, because page numbers do not exist until something lays the
// document out and Google is that something. The first upload carries blank page
// numbers and exists only to be measured: it is exported as PDF, read for the
// page each heading landed on, and then trashed. The second upload is the
// document that gets published, and it is measured too, so a contents list that
// disagrees with its own document is reported rather than hidden.
package generate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"google.golang.org/api/drive/v3"

	"spike/gdocgo/internal/pagination"
	"spike/gdocgo/internal/render"
)

const (
	googleDocMIME = "application/vnd.google-apps.document"
	docxMIME      = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	pdfMIME       = "application/pdf"
	// The throwaway says what it is in its own title, so a run that dies before
	// the cleanup leaves something a human can recognise and delete.
	measuringSuffix = " (pagination pass, delete me)"
)

// Result is what one publish produced.
type Result struct {
	DocxPath string
	DocID    string
	Link     string
	Reason   string
	// Drift lists headings whose published page differs from the number written
	// next to them. Empty means the contents list describes its own document.
	Drift map[string][2]int
	Pages map[string]int
}

func Upload(service *drive.Service, docxPath, name, folderID string) (*drive.File, error) {
	data, err := os.ReadFile(docxPath)
	if err != nil {
		return nil, err
	}
	return service.Files.
		Create(&drive.File{Name: name, MimeType: googleDocMIME, Parents: []string{folderID}}).
		Media(bytes.NewReader(data), googleContentType(docxMIME)).
		Fields("id,webViewLink").
		SupportsAllDrives(true).
		Do()
}

func ExportPDF(service *drive.Service, docID, pdfPath string) error {
	response, err := service.Files.Export(docID, pdfMIME).Download()
	if err != nil {
		return err
	}
	defer response.Body.Close()
	file, err := os.Create(pdfPath)
	if err != nil {
		return err
	}
	defer file.Close()
	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(response.Body); err != nil {
		return err
	}
	_, err = file.Write(buffer.Bytes())
	return err
}

// Generate renders through the house template, with the page numbers Google
// measured, and publishes the result.
func Generate(ctx context.Context, service *drive.Service, mdPath, name, outPath,
	folderID string, opts render.Options) (*Result, error) {

	workspace, err := os.MkdirTemp("", "gdocgo-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workspace)

	blankPath := filepath.Join(workspace, "blank.docx")
	blank, err := render.Build(mdPath, blankPath, opts)
	if err != nil {
		return nil, err
	}

	var measured *drive.File
	trash := func() string {
		if measured == nil {
			return ""
		}
		_, err := service.Files.Update(measured.Id, &drive.File{Trashed: true}).
			SupportsAllDrives(true).Do()
		if err != nil {
			return fmt.Sprintf("the measuring copy %s could not be trashed (%v), "+
				"so delete it by hand", measured.Id, err)
		}
		return ""
	}

	// The upload is its own statement so the id is held before anything else can
	// fail. Bound inside the call, a failure would lose it and leave the
	// measuring copy in the folder for good.
	measured, err = Upload(service, blank.DocxPath, name+measuringSuffix, folderID)
	if err != nil {
		return nil, err
	}

	blankPDF := filepath.Join(workspace, "blank.pdf")
	if err := ExportPDF(service, measured.Id, blankPDF); err != nil {
		trash()
		return nil, err
	}
	pages, err := pagination.FromPDF(blankPDF, blank.Headings)
	if err != nil {
		trash()
		return nil, err
	}

	withPages := opts
	withPages.Pages = pages
	if _, err := render.Build(mdPath, outPath, withPages); err != nil {
		trash()
		return nil, err
	}

	created, err := Upload(service, outPath, name, folderID)
	if err != nil {
		trash()
		return nil, err
	}

	// The document exists from here on, so nothing below may withhold its id.
	// The measuring copy goes first for the same reason: a drift check that
	// blows up must not leave a second stray document behind.
	notes := []string{}
	if note := trash(); note != "" {
		notes = append(notes, note)
	}

	moved := map[string][2]int{}
	publishedPDF := filepath.Join(workspace, "published.pdf")
	if err := ExportPDF(service, created.Id, publishedPDF); err != nil {
		notes = append(notes, "the published document could not be read back, so its "+
			"page numbers were never checked against its contents list: "+err.Error())
	} else if published, err := pagination.FromPDF(publishedPDF, blank.Headings); err != nil {
		notes = append(notes, "the published document could not be read back: "+err.Error())
	} else {
		moved = pagination.Drift(pages, published)
		if len(moved) > 0 {
			notes = append(notes, driftNote(moved))
		}
	}

	return &Result{
		DocxPath: outPath,
		DocID:    created.Id,
		Link:     created.WebViewLink,
		Reason:   strings.Join(notes, ". "),
		Drift:    moved,
		Pages:    pages,
	}, nil
}

func driftNote(moved map[string][2]int) string {
	keys := make([]string, 0, len(moved))
	for key := range moved {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	listed := make([]string, 0, len(keys))
	for _, key := range keys {
		listed = append(listed, fmt.Sprintf("%q says page %d but sits on page %d",
			key, moved[key][0], moved[key][1]))
	}
	return "the contents list disagrees with the document it sits in: " +
		strings.Join(listed, ", ") +
		". The document was created, so check its contents page before sharing it"
}
