package gltf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Version is the glTF specification version this package targets.
const Version = "2.0"

// Decode reads a .gltf JSON document from r and returns the parsed Document.
// It does not resolve buffer data; call ResolveBuffers for that.
func Decode(r io.Reader) (*Document, error) {
	var doc Document
	dec := json.NewDecoder(r)
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("gltf: decoding JSON: %w", err)
	}
	return &doc, nil
}

// Encode writes doc as indented .gltf JSON to w.
func Encode(w io.Writer, doc *Document) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("gltf: encoding JSON: %w", err)
	}
	return nil
}

// MarshalJSON is a convenience wrapper returning the indented JSON encoding of
// doc as a byte slice.
func MarshalJSON(doc *Document) ([]byte, error) {
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Open reads and decodes a .gltf file at path and resolves its buffers relative
// to the file's directory. External file and data-URI buffers are loaded.
func Open(path string) (*Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("gltf: opening %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	doc, err := Decode(f)
	if err != nil {
		return nil, err
	}
	if err := doc.ResolveBuffers(dirOf(path), nil); err != nil {
		return nil, err
	}
	return doc, nil
}

// Save encodes doc and writes it to a .gltf file at path. Buffer data is not
// written to separate files; callers embedding buffers should use data URIs or
// the GLB container.
func Save(path string, doc *Document) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("gltf: creating %q: %w", path, err)
	}
	if err := Encode(f, doc); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("gltf: closing %q: %w", path, err)
	}
	return nil
}
