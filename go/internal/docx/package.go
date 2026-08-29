// Package docx opens a .docx as what it is: an ordered list of zip parts.
//
// Nothing here models a document. python-docx does, and that modelling is the
// reason its output has to be re-serialised wholesale. Keeping every part as
// bytes and touching only the ones we edit is what lets the cover, the logo,
// the running head and the coloured tables survive unchanged.
package docx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Part is one entry in the package, held as raw bytes.
type Part struct {
	Name   string
	Data   []byte
	Method uint16
}

// Package is every part, in the order the archive listed them.
type Package struct {
	Parts []Part
}

func Open(path string) (*Package, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer reader.Close()

	pkg := &Package{}
	for _, entry := range reader.File {
		handle, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("read %s from %s: %w", entry.Name, path, err)
		}
		data, err := io.ReadAll(handle)
		handle.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s from %s: %w", entry.Name, path, err)
		}
		pkg.Parts = append(pkg.Parts, Part{Name: entry.Name, Data: data, Method: entry.Method})
	}
	return pkg, nil
}

func (p *Package) Get(name string) ([]byte, bool) {
	for _, part := range p.Parts {
		if part.Name == name {
			return part.Data, true
		}
	}
	return nil, false
}

// MustGet returns a part, naming it when it is absent. Every caller here is
// asking for a part the OOXML format guarantees, so absence is a broken file.
func (p *Package) MustGet(name string) ([]byte, error) {
	data, ok := p.Get(name)
	if !ok {
		return nil, fmt.Errorf("the package holds no %s, so it is not a Word document", name)
	}
	return data, nil
}

// Set replaces a part, or appends it if it is new.
func (p *Package) Set(name string, data []byte) {
	for i := range p.Parts {
		if p.Parts[i].Name == name {
			p.Parts[i].Data = data
			return
		}
	}
	p.Parts = append(p.Parts, Part{Name: name, Data: data, Method: zip.Deflate})
}

// Remove drops a part. Silent when it was not there: the callers removing
// comment parts should not have to know whether the template carried them.
func (p *Package) Remove(name string) {
	kept := p.Parts[:0]
	for _, part := range p.Parts {
		if part.Name != name {
			kept = append(kept, part)
		}
	}
	p.Parts = kept
}

func (p *Package) Names() []string {
	out := make([]string, 0, len(p.Parts))
	for _, part := range p.Parts {
		out = append(out, part.Name)
	}
	return out
}

// HasPrefix lists every part under a directory, such as "word/media/".
func (p *Package) HasPrefix(prefix string) []string {
	var out []string
	for _, part := range p.Parts {
		if strings.HasPrefix(part.Name, prefix) {
			out = append(out, part.Name)
		}
	}
	return out
}

func (p *Package) Bytes() ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, part := range p.Parts {
		method := part.Method
		if method != zip.Store {
			method = zip.Deflate
		}
		entry, err := writer.CreateHeader(&zip.FileHeader{Name: part.Name, Method: method})
		if err != nil {
			return nil, fmt.Errorf("write %s: %w", part.Name, err)
		}
		if _, err := entry.Write(part.Data); err != nil {
			return nil, fmt.Errorf("write %s: %w", part.Name, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (p *Package) Save(path string) error {
	data, err := p.Bytes()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
