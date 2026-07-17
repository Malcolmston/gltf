package gltf

import (
	"bytes"
	"encoding/base64"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// buildFloat32LE encodes float32 values as little-endian bytes.
func buildFloat32LE(vals ...float32) []byte {
	b := make([]byte, 0, len(vals)*4)
	for _, v := range vals {
		b = le.AppendUint32(b, math.Float32bits(v))
	}
	return b
}

// buildUint16LE encodes uint16 values as little-endian bytes.
func buildUint16LE(vals ...uint16) []byte {
	b := make([]byte, 0, len(vals)*2)
	for _, v := range vals {
		b = le.AppendUint16(b, v)
	}
	return b
}

func TestGLTFRoundTrip(t *testing.T) {
	doc, _ := Triangle()

	data, err := MarshalJSON(doc)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	got, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	// Data is not serialized, so clear it before comparing structure.
	doc.Buffers[0].Data = nil
	if !reflect.DeepEqual(doc, got) {
		t.Errorf("round trip mismatch:\noriginal: %+v\ndecoded:  %+v", doc, got)
	}
	if got.Asset.Version != "2.0" {
		t.Errorf("asset version = %q, want 2.0", got.Asset.Version)
	}
}

func TestGLBRoundTrip(t *testing.T) {
	doc, bin := Triangle()

	var buf bytes.Buffer
	if err := WriteGLB(&buf, doc, bin); err != nil {
		t.Fatalf("WriteGLB: %v", err)
	}

	// Verify the header magic and version explicitly.
	raw := buf.Bytes()
	if magic := le.Uint32(raw[0:4]); magic != GLBMagic {
		t.Fatalf("magic = 0x%08X, want 0x%08X", magic, GLBMagic)
	}
	if v := le.Uint32(raw[4:8]); v != GLBVersion {
		t.Fatalf("version = %d, want %d", v, GLBVersion)
	}
	if l := le.Uint32(raw[8:12]); int(l) != len(raw) {
		t.Fatalf("declared length %d != actual %d", l, len(raw))
	}

	got, gotBin, err := ReadGLB(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("ReadGLB: %v", err)
	}
	if !bytes.Equal(gotBin, bin) {
		t.Errorf("BIN chunk mismatch: got %v, want %v", gotBin, bin)
	}
	if err := got.ResolveBuffers("", gotBin); err != nil {
		t.Fatalf("ResolveBuffers: %v", err)
	}
	positions, err := got.DecodeAccessorVec3(0)
	if err != nil {
		t.Fatalf("DecodeAccessorVec3: %v", err)
	}
	if len(positions) != 3 {
		t.Fatalf("got %d positions, want 3", len(positions))
	}
	if positions[1] != [3]float32{1, 0, 0} {
		t.Errorf("positions[1] = %v, want [1 0 0]", positions[1])
	}
}

func TestGLBChunkPadding(t *testing.T) {
	// A three-byte buffer forces the BIN chunk to be padded to four bytes.
	bin := []byte{1, 2, 3}
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: len(bin)}},
	}
	var buf bytes.Buffer
	if err := WriteGLB(&buf, doc, bin); err != nil {
		t.Fatalf("WriteGLB: %v", err)
	}
	if len(buf.Bytes())%4 != 0 {
		t.Errorf("GLB total length %d not 4-byte aligned", len(buf.Bytes()))
	}
	got, gotBin, err := ReadGLB(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("ReadGLB: %v", err)
	}
	if !bytes.Equal(gotBin, bin) {
		t.Errorf("BIN after padding trim = %v, want %v", gotBin, bin)
	}
	if got.Buffers[0].ByteLength != 3 {
		t.Errorf("buffer byteLength = %d, want 3", got.Buffers[0].ByteLength)
	}
}

func TestDecodeVec3Float(t *testing.T) {
	bin := buildFloat32LE(
		1, 2, 3,
		4, 5, 6,
	)
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: len(bin)}},
		Accessors: []Accessor{{
			BufferView:    intPtr(0),
			ComponentType: ComponentFloat,
			Count:         2,
			Type:          AccessorVec3,
		}},
	}
	got, err := doc.DecodeAccessorVec3(0)
	if err != nil {
		t.Fatalf("DecodeAccessorVec3: %v", err)
	}
	want := [][3]float32{{1, 2, 3}, {4, 5, 6}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDecodeScalarUShortStrided(t *testing.T) {
	// Two SCALAR USHORT elements stored with a byteStride of 4: each two-byte
	// value is followed by two bytes of padding.
	bin := []byte{
		0x0A, 0x00, 0xFF, 0xFF, // value 10, then padding
		0x14, 0x00, 0xFF, 0xFF, // value 20, then padding
	}
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: len(bin), ByteStride: 4}},
		Accessors: []Accessor{{
			BufferView:    intPtr(0),
			ComponentType: ComponentUnsignedShort,
			Count:         2,
			Type:          AccessorScalar,
		}},
	}
	got, err := doc.DecodeAccessorUint32(0)
	if err != nil {
		t.Fatalf("DecodeAccessorUint32: %v", err)
	}
	want := []uint32{10, 20}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDecodeByteOffset(t *testing.T) {
	// Accessor starts 8 bytes into the bufferView.
	bin := append([]byte{0, 0, 0, 0, 0, 0, 0, 0}, buildFloat32LE(7, 8, 9)...)
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: len(bin)}},
		Accessors: []Accessor{{
			BufferView:    intPtr(0),
			ByteOffset:    8,
			ComponentType: ComponentFloat,
			Count:         1,
			Type:          AccessorVec3,
		}},
	}
	got, err := doc.DecodeAccessorVec3(0)
	if err != nil {
		t.Fatalf("DecodeAccessorVec3: %v", err)
	}
	if got[0] != [3]float32{7, 8, 9} {
		t.Errorf("got %v, want [7 8 9]", got[0])
	}
}

func TestDecodeNormalizedUByte(t *testing.T) {
	bin := []byte{0, 255, 128}
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: len(bin)}},
		Accessors: []Accessor{{
			BufferView:    intPtr(0),
			ComponentType: ComponentUnsignedByte,
			Normalized:    true,
			Count:         3,
			Type:          AccessorScalar,
		}},
	}
	got, err := doc.DecodeAccessorFloat32(0)
	if err != nil {
		t.Fatalf("DecodeAccessorFloat32: %v", err)
	}
	if got[0] != 0 || got[1] != 1 || math.Abs(float64(got[2])-128.0/255.0) > 1e-6 {
		t.Errorf("normalized decode = %v", got)
	}
}

func TestSparseAccessor(t *testing.T) {
	// Base data: four VEC3 float elements, all zero.
	baseBytes := make([]byte, 4*3*4)
	// Sparse indices: replace elements 1 and 3 (USHORT).
	idxBytes := buildUint16LE(1, 3)
	// Sparse values: two VEC3 float elements.
	valBytes := buildFloat32LE(
		10, 11, 12,
		20, 21, 22,
	)

	bin := append(append(append([]byte{}, baseBytes...), idxBytes...), valBytes...)
	baseLen := len(baseBytes)
	idxLen := len(idxBytes)

	doc := &Document{
		Asset:   Asset{Version: Version},
		Buffers: []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{
			{Buffer: 0, ByteOffset: 0, ByteLength: baseLen},
			{Buffer: 0, ByteOffset: baseLen, ByteLength: idxLen},
			{Buffer: 0, ByteOffset: baseLen + idxLen, ByteLength: len(valBytes)},
		},
		Accessors: []Accessor{{
			BufferView:    intPtr(0),
			ComponentType: ComponentFloat,
			Count:         4,
			Type:          AccessorVec3,
			Sparse: &Sparse{
				Count:   2,
				Indices: SparseIndices{BufferView: 1, ComponentType: ComponentUnsignedShort},
				Values:  SparseValues{BufferView: 2},
			},
		}},
	}

	got, err := doc.DecodeAccessorVec3(0)
	if err != nil {
		t.Fatalf("DecodeAccessorVec3: %v", err)
	}
	want := [][3]float32{
		{0, 0, 0},
		{10, 11, 12},
		{0, 0, 0},
		{20, 21, 22},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sparse decode = %v, want %v", got, want)
	}
}

func TestSparseWithoutBufferView(t *testing.T) {
	// An accessor with no bufferView is initialized to zeros, then sparse
	// substitutions fill in some elements.
	idxBytes := buildUint16LE(0, 2)
	valBytes := buildFloat32LE(1, 2, 3)
	bin := append(append([]byte{}, idxBytes...), valBytes...)

	doc := &Document{
		Asset:   Asset{Version: Version},
		Buffers: []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{
			{Buffer: 0, ByteOffset: 0, ByteLength: len(idxBytes)},
			{Buffer: 0, ByteOffset: len(idxBytes), ByteLength: len(valBytes)},
		},
		Accessors: []Accessor{{
			ComponentType: ComponentFloat,
			Count:         3,
			Type:          AccessorScalar,
			Sparse: &Sparse{
				Count:   2,
				Indices: SparseIndices{BufferView: 0, ComponentType: ComponentUnsignedShort},
				Values:  SparseValues{BufferView: 1},
			},
		}},
	}
	got, err := doc.DecodeAccessorFloat32(0)
	if err != nil {
		t.Fatalf("DecodeAccessorFloat32: %v", err)
	}
	want := []float32{1, 0, 2}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBase64DataURIBuffer(t *testing.T) {
	bin := buildFloat32LE(1, 2, 3)
	uri := "data:application/octet-stream;base64," + base64.StdEncoding.EncodeToString(bin)
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{URI: uri, ByteLength: len(bin)}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: len(bin)}},
		Accessors: []Accessor{{
			BufferView:    intPtr(0),
			ComponentType: ComponentFloat,
			Count:         1,
			Type:          AccessorVec3,
		}},
	}
	if err := doc.ResolveBuffers("", nil); err != nil {
		t.Fatalf("ResolveBuffers: %v", err)
	}
	if !bytes.Equal(doc.Buffers[0].Data, bin) {
		t.Fatalf("resolved data = %v, want %v", doc.Buffers[0].Data, bin)
	}
	got, err := doc.DecodeAccessorVec3(0)
	if err != nil {
		t.Fatalf("DecodeAccessorVec3: %v", err)
	}
	if got[0] != [3]float32{1, 2, 3} {
		t.Errorf("got %v, want [1 2 3]", got[0])
	}
}

func TestEncodeDataURIRoundTrip(t *testing.T) {
	bin := []byte{9, 8, 7, 6}
	uri := EncodeDataURI(bin)
	doc := &Document{Buffers: []Buffer{{URI: uri, ByteLength: len(bin)}}}
	if err := doc.ResolveBuffers("", nil); err != nil {
		t.Fatalf("ResolveBuffers: %v", err)
	}
	if !bytes.Equal(doc.Buffers[0].Data, bin) {
		t.Errorf("got %v, want %v", doc.Buffers[0].Data, bin)
	}
}

func TestExternalFileBuffer(t *testing.T) {
	dir := t.TempDir()
	bin := buildFloat32LE(3, 2, 1)
	if err := os.WriteFile(filepath.Join(dir, "data.bin"), bin, 0o600); err != nil {
		t.Fatal(err)
	}
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{URI: "data.bin", ByteLength: len(bin)}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: len(bin)}},
		Accessors: []Accessor{{
			BufferView:    intPtr(0),
			ComponentType: ComponentFloat,
			Count:         1,
			Type:          AccessorVec3,
		}},
	}
	if err := doc.ResolveBuffers(dir, nil); err != nil {
		t.Fatalf("ResolveBuffers: %v", err)
	}
	got, err := doc.DecodeAccessorVec3(0)
	if err != nil {
		t.Fatalf("DecodeAccessorVec3: %v", err)
	}
	if got[0] != [3]float32{3, 2, 1} {
		t.Errorf("got %v, want [3 2 1]", got[0])
	}
}

func TestValidateOutOfRangeIndex(t *testing.T) {
	doc, _ := Triangle()
	// Point the primitive's POSITION attribute at a non-existent accessor.
	doc.Meshes[0].Primitives[0].Attributes["POSITION"] = 99

	err := doc.Validate()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	verrs, ok := AsValidationErrors(err)
	if !ok {
		t.Fatalf("error is not ValidationErrors: %T", err)
	}
	found := false
	for _, v := range verrs {
		if v.Path == "meshes[0].primitives[0].attributes.POSITION" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected out-of-range POSITION error, got: %v", verrs)
	}
}

func TestValidateMissingVersion(t *testing.T) {
	doc := &Document{}
	err := doc.Validate()
	if err == nil {
		t.Fatal("expected error for missing asset.version")
	}
	verrs, _ := AsValidationErrors(err)
	found := false
	for _, v := range verrs {
		if v.Path == "asset.version" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected asset.version error, got: %v", err)
	}
}

func TestValidateTriangleValid(t *testing.T) {
	doc, _ := Triangle()
	if err := doc.Validate(); err != nil {
		t.Errorf("Triangle should validate, got: %v", err)
	}
}

func TestFileRoundTrip(t *testing.T) {
	dir := t.TempDir()

	// GLB file round trip.
	glbPath := filepath.Join(dir, "tri.glb")
	doc, bin := Triangle()
	if err := SaveGLB(glbPath, doc, bin); err != nil {
		t.Fatalf("SaveGLB: %v", err)
	}
	loaded, err := OpenGLB(glbPath)
	if err != nil {
		t.Fatalf("OpenGLB: %v", err)
	}
	pos, err := loaded.DecodeAccessorVec3(0)
	if err != nil || len(pos) != 3 {
		t.Fatalf("decode after OpenGLB: %v (len %d)", err, len(pos))
	}

	// Embedded .gltf file round trip.
	gltfPath := filepath.Join(dir, "tri.gltf")
	f, err := os.Create(gltfPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteTriangleGLTF(f); err != nil {
		t.Fatalf("WriteTriangleGLTF: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	loaded2, err := Open(gltfPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	pos2, err := loaded2.DecodeAccessorVec3(0)
	if err != nil || len(pos2) != 3 {
		t.Fatalf("decode after Open: %v (len %d)", err, len(pos2))
	}
}

func TestReadGLBErrors(t *testing.T) {
	cases := map[string][]byte{
		"too short":  {1, 2, 3},
		"bad magic":  append(le.AppendUint32(nil, 0xDEADBEEF), make([]byte, 8)...),
		"bad length": append(le.AppendUint32(le.AppendUint32(le.AppendUint32(nil, GLBMagic), GLBVersion), 999), make([]byte, 0)...),
	}
	for name, data := range cases {
		if _, _, err := ReadGLB(bytes.NewReader(data)); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

func TestDecodeIndices(t *testing.T) {
	bin := buildUint16LE(0, 1, 2)
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: len(bin)}},
		Accessors: []Accessor{{
			BufferView:    intPtr(0),
			ComponentType: ComponentUnsignedShort,
			Count:         3,
			Type:          AccessorScalar,
		}},
	}
	prim := &Primitive{Attributes: map[string]Index{"POSITION": 0}, Indices: intPtr(0)}
	got, err := doc.DecodeIndices(prim)
	if err != nil {
		t.Fatalf("DecodeIndices: %v", err)
	}
	want := []uint32{0, 1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// A primitive without indices returns nil.
	nonIndexed := &Primitive{Attributes: map[string]Index{"POSITION": 0}}
	if idx, err := doc.DecodeIndices(nonIndexed); err != nil || idx != nil {
		t.Errorf("non-indexed primitive: got %v, %v; want nil, nil", idx, err)
	}
}

func TestEnumHelpers(t *testing.T) {
	if ComponentFloat.SizeInBytes() != 4 {
		t.Errorf("float size = %d, want 4", ComponentFloat.SizeInBytes())
	}
	if ComponentUnsignedShort.SizeInBytes() != 2 {
		t.Errorf("ushort size = %d, want 2", ComponentUnsignedShort.SizeInBytes())
	}
	if AccessorMat4.ComponentCount() != 16 {
		t.Errorf("mat4 count = %d, want 16", AccessorMat4.ComponentCount())
	}
	if ComponentFloat.String() != "FLOAT" {
		t.Errorf("float string = %q", ComponentFloat.String())
	}
	p := &Primitive{}
	if p.GetMode() != PrimitiveTriangles {
		t.Errorf("default mode = %v, want triangles", p.GetMode())
	}
}
