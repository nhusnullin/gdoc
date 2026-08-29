package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/beevik/etree"
)

func main() {
	in, _ := os.ReadFile("document.xml")
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(in); err != nil {
		fmt.Println("parse error:", err)
		return
	}
	got, err := doc.WriteToBytes()
	if err != nil {
		fmt.Println("write error:", err)
		return
	}
	fmt.Printf("in  bytes: %d\nout bytes: %d\nbyte-identical: %v\n", len(in), len(got), bytes.Equal(in, got))
	os.WriteFile("etree-out.xml", got, 0o644)

	// namespace-aware selection, the way OOXML surgery needs it
	root := doc.Root()
	fmt.Printf("root Space=%q Tag=%q\n", root.Space, root.Tag)
	body := root.SelectElement("w:body")
	fmt.Printf("body found: %v\n", body != nil)
	n := 0
	for _, p := range body.SelectElements("w:p") {
		if pPr := p.SelectElement("w:pPr"); pPr != nil {
			if st := pPr.SelectElement("w:pStyle"); st != nil {
				if v := st.SelectAttrValue("w:val", ""); len(v) > 7 && v[:7] == "Heading" {
					n++
				}
			}
		}
	}
	fmt.Printf("top-level Heading* paragraphs found: %d\n", n)
	fmt.Printf("sdt elements preserved: %d\n", bytes.Count(got, []byte("<w:sdt>")))
	fmt.Printf("w14: attrs preserved:   %d\n", bytes.Count(got, []byte("w14:")))
}
