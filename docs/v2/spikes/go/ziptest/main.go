package main

// Pure Go stdlib: extract every embedded image from a .docx, resolved through
// the relationship parts so header/footer images are found too. No third-party
// code at all.

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path"
	"strings"
)

type Relationships struct {
	Rel []struct {
		ID         string `xml:"Id,attr"`
		Type       string `xml:"Type,attr"`
		Target     string `xml:"Target,attr"`
		TargetMode string `xml:"TargetMode,attr"`
	} `xml:"Relationship"`
}

const imageRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"

func main() {
	r, err := zip.OpenReader("template.docx")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	files := map[string]*zip.File{}
	for _, f := range r.File {
		files[f.Name] = f
	}

	// every rels part, so header/footer images are found as well as body ones
	found := map[string][]string{}
	for name, f := range files {
		if !strings.HasSuffix(name, ".rels") {
			continue
		}
		rc, _ := f.Open()
		data, _ := io.ReadAll(rc)
		rc.Close()
		var rels Relationships
		if err := xml.Unmarshal(data, &rels); err != nil {
			fmt.Println("rels parse error", name, err)
			continue
		}
		owner := strings.TrimSuffix(path.Base(name), ".rels")
		base := path.Dir(path.Dir(name)) // strip _rels/
		for _, rel := range rels.Rel {
			if rel.Type != imageRelType {
				continue
			}
			if rel.TargetMode == "External" {
				found["EXTERNAL(no bytes)"] = append(found["EXTERNAL(no bytes)"], owner+":"+rel.ID+" -> "+rel.Target)
				continue
			}
			target := path.Clean(path.Join(base, rel.Target))
			found[target] = append(found[target], owner+":"+rel.ID)
		}
	}

	fmt.Println("=== images resolved through relationships ===")
	for target, refs := range found {
		f := files[target]
		if f == nil {
			fmt.Printf("%-24s MISSING PART  refs=%v\n", target, refs)
			continue
		}
		rc, _ := f.Open()
		cfg, format, err := image.DecodeConfig(rc)
		rc.Close()
		dims := "undecodable"
		if err == nil {
			dims = fmt.Sprintf("%s %dx%d", format, cfg.Width, cfg.Height)
		}
		fmt.Printf("%-24s %8d bytes  %-14s refs=%v\n", target, f.UncompressedSize64, dims, refs)
	}

	fmt.Println("\n=== raw word/media listing (the naive approach) ===")
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "word/media/") {
			fmt.Printf("%-24s %8d bytes\n", f.Name, f.UncompressedSize64)
		}
	}
}
