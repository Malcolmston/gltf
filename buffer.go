package gltf

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// dataURIPrefix marks the start of an RFC 2397 data URI.
const dataURIPrefix = "data:"

// ResolveBuffers loads the raw bytes for every buffer in the document, storing
// them in each Buffer's Data field. It handles three sources:
//
//   - an embedded GLB BIN chunk (a buffer with an empty URI), supplied via bin;
//   - base64 "data:" URIs;
//   - external files, resolved relative to baseDir.
//
// baseDir may be empty when no external file references are expected. bin may
// be nil when there is no GLB binary chunk.
func (d *Document) ResolveBuffers(baseDir string, bin []byte) error {
	for i := range d.Buffers {
		b := &d.Buffers[i]
		switch {
		case b.URI == "":
			// A URI-less buffer refers to the GLB BIN chunk.
			if bin == nil {
				return fmt.Errorf("gltf: buffer %d has no URI and no GLB binary chunk was provided", i)
			}
			if len(bin) < b.ByteLength {
				return fmt.Errorf("gltf: buffer %d declares %d bytes but GLB binary chunk has %d", i, b.ByteLength, len(bin))
			}
			b.Data = bin[:b.ByteLength]
		case strings.HasPrefix(b.URI, dataURIPrefix):
			data, err := decodeDataURI(b.URI)
			if err != nil {
				return fmt.Errorf("gltf: buffer %d: %w", i, err)
			}
			b.Data = data
		default:
			data, err := readExternal(baseDir, b.URI)
			if err != nil {
				return fmt.Errorf("gltf: buffer %d: %w", i, err)
			}
			b.Data = data
		}
	}
	return nil
}

// decodeDataURI decodes a base64-encoded RFC 2397 data URI into raw bytes. Only
// base64 payloads are supported, which is what glTF uses for embedded buffers.
func decodeDataURI(uri string) ([]byte, error) {
	comma := strings.IndexByte(uri, ',')
	if comma < 0 {
		return nil, fmt.Errorf("malformed data URI: missing comma")
	}
	meta := uri[len(dataURIPrefix):comma]
	payload := uri[comma+1:]
	if !strings.Contains(meta, "base64") {
		return nil, fmt.Errorf("unsupported data URI encoding (only base64 is supported)")
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("decoding base64 data URI: %w", err)
	}
	return data, nil
}

// readExternal reads an external buffer file. The URI is treated as a (possibly
// percent-free) relative path joined to baseDir.
func readExternal(baseDir, uri string) ([]byte, error) {
	p := uri
	if !filepath.IsAbs(p) {
		p = filepath.Join(baseDir, filepath.FromSlash(uri))
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("reading external buffer %q: %w", p, err)
	}
	return data, nil
}

// EncodeDataURI returns a base64 application/octet-stream data URI for data,
// suitable for use as a Buffer.URI in an embedded .gltf file.
func EncodeDataURI(data []byte) string {
	return "data:application/octet-stream;base64," + base64.StdEncoding.EncodeToString(data)
}

// dirOf returns the directory containing path, for use as a buffer base dir.
func dirOf(path string) string {
	return filepath.Dir(path)
}
