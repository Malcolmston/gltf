package gltf

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// GLB container constants defined by the glTF 2.0 specification.
const (
	// GLBMagic is the little-endian "glTF" magic that begins every GLB file.
	GLBMagic uint32 = 0x46546C67
	// GLBVersion is the GLB container version supported by this package.
	GLBVersion uint32 = 2

	// chunkTypeJSON identifies the structured JSON chunk ("JSON").
	chunkTypeJSON uint32 = 0x4E4F534A
	// chunkTypeBIN identifies the binary buffer chunk ("BIN\0").
	chunkTypeBIN uint32 = 0x004E4942

	glbHeaderSize = 12
	glbChunkHead  = 8
)

// ReadGLB reads a binary GLB stream from r, returning the parsed Document and
// the raw bytes of its embedded BIN chunk (nil if absent). The BIN chunk is
// also attached to the first buffer without a URI via ResolveBuffers-style
// wiring: callers that need buffer data resolved should pass the returned bin
// to Document.ResolveBuffers.
func ReadGLB(r io.Reader) (*Document, []byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("gltf: reading GLB: %w", err)
	}
	if len(data) < glbHeaderSize {
		return nil, nil, fmt.Errorf("gltf: GLB too short: %d bytes", len(data))
	}

	magic := binary.LittleEndian.Uint32(data[0:4])
	if magic != GLBMagic {
		return nil, nil, fmt.Errorf("gltf: bad GLB magic 0x%08X", magic)
	}
	version := binary.LittleEndian.Uint32(data[4:8])
	if version != GLBVersion {
		return nil, nil, fmt.Errorf("gltf: unsupported GLB version %d", version)
	}
	length := binary.LittleEndian.Uint32(data[8:12])
	if int(length) != len(data) {
		return nil, nil, fmt.Errorf("gltf: GLB length mismatch: header %d, actual %d", length, len(data))
	}

	var jsonChunk, binChunk []byte
	haveJSON := false
	off := glbHeaderSize
	for off+glbChunkHead <= len(data) {
		chunkLen := binary.LittleEndian.Uint32(data[off : off+4])
		chunkType := binary.LittleEndian.Uint32(data[off+4 : off+8])
		start := off + glbChunkHead
		end := start + int(chunkLen)
		if end > len(data) {
			return nil, nil, fmt.Errorf("gltf: GLB chunk overruns file: need %d, have %d", end, len(data))
		}
		payload := data[start:end]
		switch chunkType {
		case chunkTypeJSON:
			jsonChunk = payload
			haveJSON = true
		case chunkTypeBIN:
			binChunk = payload
		default:
			// Unknown chunk types are ignored per the specification.
		}
		off = end
	}

	if !haveJSON {
		return nil, nil, fmt.Errorf("gltf: GLB missing JSON chunk")
	}

	doc, err := Decode(bytes.NewReader(jsonChunk))
	if err != nil {
		return nil, nil, err
	}
	// Trim BIN chunk padding to the declared buffer length when possible.
	if binChunk != nil && len(doc.Buffers) > 0 && doc.Buffers[0].URI == "" {
		if n := doc.Buffers[0].ByteLength; n >= 0 && n <= len(binChunk) {
			binChunk = binChunk[:n]
		}
	}
	return doc, binChunk, nil
}

// WriteGLB writes doc and an optional BIN chunk to w as a binary GLB stream. If
// bin is nil, the BIN chunk is omitted. The JSON chunk is padded with spaces
// and the BIN chunk with zeros to a four-byte boundary, as required.
func WriteGLB(w io.Writer, doc *Document, bin []byte) error {
	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("gltf: encoding GLB JSON: %w", err)
	}

	jsonPadded := padChunk(jsonBytes, 0x20) // pad JSON with spaces
	total := glbHeaderSize + glbChunkHead + len(jsonPadded)

	var binPadded []byte
	if bin != nil {
		binPadded = padChunk(bin, 0x00) // pad BIN with zeros
		total += glbChunkHead + len(binPadded)
	}

	buf := make([]byte, 0, total)
	buf = binary.LittleEndian.AppendUint32(buf, GLBMagic)
	buf = binary.LittleEndian.AppendUint32(buf, GLBVersion)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(total))

	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(jsonPadded)))
	buf = binary.LittleEndian.AppendUint32(buf, chunkTypeJSON)
	buf = append(buf, jsonPadded...)

	if bin != nil {
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(binPadded)))
		buf = binary.LittleEndian.AppendUint32(buf, chunkTypeBIN)
		buf = append(buf, binPadded...)
	}

	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("gltf: writing GLB: %w", err)
	}
	return nil
}

// padChunk returns b padded with pad to a multiple of four bytes.
func padChunk(b []byte, pad byte) []byte {
	rem := len(b) % 4
	if rem == 0 {
		return b
	}
	out := make([]byte, len(b)+(4-rem))
	copy(out, b)
	for i := len(b); i < len(out); i++ {
		out[i] = pad
	}
	return out
}

// OpenGLB reads and decodes a .glb file at path, attaching its BIN chunk to the
// document's buffers and resolving external and data-URI buffers relative to
// the file's directory.
func OpenGLB(path string) (*Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("gltf: opening %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	doc, bin, err := ReadGLB(f)
	if err != nil {
		return nil, err
	}
	if err := doc.ResolveBuffers(dirOf(path), bin); err != nil {
		return nil, err
	}
	return doc, nil
}

// SaveGLB writes doc and an optional BIN chunk to a .glb file at path.
func SaveGLB(path string, doc *Document, bin []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("gltf: creating %q: %w", path, err)
	}
	if err := WriteGLB(f, doc, bin); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("gltf: closing %q: %w", path, err)
	}
	return nil
}
