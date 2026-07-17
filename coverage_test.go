package gltf

import (
	"bytes"
	"strings"
	"testing"
)

func TestDecodeVec2AndVec4(t *testing.T) {
	bin := buildFloat32LE(1, 2, 3, 4, 5, 6, 7, 8)
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: len(bin)}},
		Accessors: []Accessor{
			{BufferView: intPtr(0), ComponentType: ComponentFloat, Count: 4, Type: AccessorVec2},
			{BufferView: intPtr(0), ComponentType: ComponentFloat, Count: 2, Type: AccessorVec4},
		},
	}
	v2, err := doc.DecodeAccessorVec2(0)
	if err != nil {
		t.Fatalf("Vec2: %v", err)
	}
	if v2[0] != [2]float32{1, 2} || v2[3] != [2]float32{7, 8} {
		t.Errorf("vec2 = %v", v2)
	}
	v4, err := doc.DecodeAccessorVec4(1)
	if err != nil {
		t.Fatalf("Vec4: %v", err)
	}
	if v4[0] != [4]float32{1, 2, 3, 4} || v4[1] != [4]float32{5, 6, 7, 8} {
		t.Errorf("vec4 = %v", v4)
	}
}

func TestDecodeSignedShortAndByte(t *testing.T) {
	// SHORT normalized: -32767 -> -1, 32767 -> 1.
	bin := []byte{0x01, 0x80, 0xFF, 0x7F} // -32767, 32767
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: len(bin)}},
		Accessors: []Accessor{{
			BufferView: intPtr(0), ComponentType: ComponentShort, Normalized: true,
			Count: 2, Type: AccessorScalar,
		}},
	}
	got, err := doc.DecodeAccessorFloat32(0)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got[0] != -1 || got[1] != 1 {
		t.Errorf("normalized short = %v, want [-1 1]", got)
	}

	// BYTE non-normalized.
	bin2 := []byte{0xFF, 0x02} // -1, 2
	doc2 := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: len(bin2), Data: bin2}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: len(bin2)}},
		Accessors: []Accessor{{
			BufferView: intPtr(0), ComponentType: ComponentByte,
			Count: 2, Type: AccessorScalar,
		}},
	}
	g2, err := doc2.DecodeAccessorFloat32(0)
	if err != nil {
		t.Fatalf("decode byte: %v", err)
	}
	if g2[0] != -1 || g2[1] != 2 {
		t.Errorf("byte = %v, want [-1 2]", g2)
	}
}

func TestDecodeUnsignedIntAndByteIndices(t *testing.T) {
	bin := le.AppendUint32(nil, 4000000000) // > 2^31
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: len(bin)}},
		Accessors: []Accessor{{
			BufferView: intPtr(0), ComponentType: ComponentUnsignedInt,
			Count: 1, Type: AccessorScalar,
		}},
	}
	got, err := doc.DecodeAccessorUint32(0)
	if err != nil {
		t.Fatalf("decode uint: %v", err)
	}
	if got[0] != 4000000000 {
		t.Errorf("uint = %d, want 4000000000", got[0])
	}

	// UNSIGNED_BYTE indices.
	binB := []byte{0, 1, 2}
	docB := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: len(binB), Data: binB}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: len(binB)}},
		Accessors: []Accessor{{
			BufferView: intPtr(0), ComponentType: ComponentUnsignedByte,
			Count: 3, Type: AccessorScalar,
		}},
	}
	gb, err := docB.DecodeAccessorUint32(0)
	if err != nil || gb[2] != 2 {
		t.Errorf("ubyte indices = %v, %v", gb, err)
	}
}

func TestDecodeErrors(t *testing.T) {
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: 12, Data: make([]byte, 12)}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: 12}},
		Accessors: []Accessor{
			{BufferView: intPtr(0), ComponentType: ComponentFloat, Count: 1, Type: AccessorVec3},
		},
	}

	// Accessor index out of range.
	if _, err := doc.DecodeAccessorVec3(5); err == nil {
		t.Error("expected out-of-range accessor error")
	}
	// Wrong accessor type for Vec2.
	if _, err := doc.DecodeAccessorVec2(0); err == nil {
		t.Error("expected type mismatch error")
	}
	// Uint decode of a float accessor.
	if _, err := doc.DecodeAccessorUint32(0); err == nil {
		t.Error("expected non-integer component error")
	}
	// Unknown accessor type.
	doc.Accessors = append(doc.Accessors, Accessor{
		BufferView: intPtr(0), ComponentType: ComponentFloat, Count: 1, Type: "BOGUS",
	})
	if _, err := doc.DecodeAccessorFloat32(1); err == nil {
		t.Error("expected unknown type error")
	}
	// Unknown component type.
	doc.Accessors = append(doc.Accessors, Accessor{
		BufferView: intPtr(0), ComponentType: 9999, Count: 1, Type: AccessorScalar,
	})
	if _, err := doc.DecodeAccessorFloat32(2); err == nil {
		t.Error("expected unknown component error")
	}
}

func TestUnresolvedBufferError(t *testing.T) {
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: 12}}, // Data nil
		BufferViews: []BufferView{{Buffer: 0, ByteLength: 12}},
		Accessors: []Accessor{{
			BufferView: intPtr(0), ComponentType: ComponentFloat, Count: 1, Type: AccessorVec3,
		}},
	}
	if _, err := doc.DecodeAccessorVec3(0); err == nil {
		t.Error("expected unresolved-buffer error")
	}
}

func TestReadBeyondBuffer(t *testing.T) {
	bin := buildFloat32LE(1, 2, 3) // only one vec3
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{{Buffer: 0, ByteLength: len(bin)}},
		Accessors: []Accessor{{
			BufferView: intPtr(0), ComponentType: ComponentFloat, Count: 2, Type: AccessorVec3,
		}},
	}
	if _, err := doc.DecodeAccessorVec3(0); err == nil {
		t.Error("expected out-of-bounds read error")
	}
}

func TestResolveBuffersErrors(t *testing.T) {
	// URI-less buffer with no BIN provided.
	doc := &Document{Buffers: []Buffer{{ByteLength: 4}}}
	if err := doc.ResolveBuffers("", nil); err == nil {
		t.Error("expected error for missing BIN chunk")
	}

	// Unsupported (non-base64) data URI.
	doc2 := &Document{Buffers: []Buffer{{URI: "data:text/plain,hello", ByteLength: 5}}}
	if err := doc2.ResolveBuffers("", nil); err == nil {
		t.Error("expected error for non-base64 data URI")
	}

	// Malformed data URI (no comma).
	doc3 := &Document{Buffers: []Buffer{{URI: "data:base64stuff", ByteLength: 5}}}
	if err := doc3.ResolveBuffers("", nil); err == nil {
		t.Error("expected error for malformed data URI")
	}

	// Missing external file.
	doc4 := &Document{Buffers: []Buffer{{URI: "does-not-exist.bin", ByteLength: 5}}}
	if err := doc4.ResolveBuffers("/nonexistent", nil); err == nil {
		t.Error("expected error for missing external file")
	}

	// BIN chunk too small for declared length.
	doc5 := &Document{Buffers: []Buffer{{ByteLength: 100}}}
	if err := doc5.ResolveBuffers("", []byte{1, 2, 3}); err == nil {
		t.Error("expected error for short BIN chunk")
	}
}

func TestGLBMissingJSONChunk(t *testing.T) {
	// Header only, no chunks.
	buf := le.AppendUint32(nil, GLBMagic)
	buf = le.AppendUint32(buf, GLBVersion)
	buf = le.AppendUint32(buf, 12)
	if _, _, err := ReadGLB(bytes.NewReader(buf)); err == nil {
		t.Error("expected missing-JSON-chunk error")
	}
}

func TestSaveGLTFFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/out.gltf"
	doc, _ := Triangle()
	doc.Buffers[0].URI = EncodeDataURI(doc.Buffers[0].Data)
	doc.Buffers[0].Data = nil
	if err := Save(path, doc); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if loaded.Asset.Version != "2.0" {
		t.Errorf("version = %q", loaded.Asset.Version)
	}
}

func TestWriteTriangleGLBHelper(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTriangleGLB(&buf); err != nil {
		t.Fatalf("WriteTriangleGLB: %v", err)
	}
	doc, bin, err := ReadGLB(&buf)
	if err != nil {
		t.Fatalf("ReadGLB: %v", err)
	}
	if len(bin) != 36 {
		t.Errorf("bin length = %d, want 36", len(bin))
	}
	if len(doc.Meshes) != 1 {
		t.Errorf("meshes = %d, want 1", len(doc.Meshes))
	}
}

func TestValidationErrorStrings(t *testing.T) {
	e := ValidationError{Path: "a.b", Message: "bad"}
	if e.Error() != "a.b: bad" {
		t.Errorf("Error() = %q", e.Error())
	}
	e2 := ValidationError{Message: "bad"}
	if e2.Error() != "bad" {
		t.Errorf("Error() = %q", e2.Error())
	}
	var empty ValidationErrors
	if !strings.Contains(empty.Error(), "no validation errors") {
		t.Errorf("empty errors string = %q", empty.Error())
	}
	errs := ValidationErrors{e, e2}
	if !strings.Contains(errs.Error(), "2 validation error") {
		t.Errorf("errors string = %q", errs.Error())
	}
}

func TestValidateComprehensive(t *testing.T) {
	// Build a document that trips many validation branches at once.
	doc := &Document{
		Asset:  Asset{Version: "2.0"},
		Scene:  intPtr(9), // out of range
		Scenes: []Scene{{Nodes: []Index{5}}},
		Nodes: []Node{{
			Mesh:        intPtr(9),
			Camera:      intPtr(9),
			Skin:        intPtr(9),
			Children:    []Index{9},
			Matrix:      &[16]float64{},
			Translation: &[3]float64{}, // matrix + TRS conflict
		}},
		Meshes: []Mesh{
			{Primitives: nil}, // empty primitives
			{Primitives: []Primitive{{Attributes: nil, Indices: intPtr(9), Material: intPtr(9)}}},
		},
		Accessors: []Accessor{
			{Type: "BAD", ComponentType: 0, Count: 0},
			{Type: AccessorScalar, ComponentType: ComponentFloat, Count: 1, BufferView: intPtr(9),
				Sparse: &Sparse{Count: 0, Indices: SparseIndices{BufferView: 9}, Values: SparseValues{BufferView: 9}}},
		},
		BufferViews: []BufferView{{Buffer: 9, ByteLength: 0}},
		Buffers:     []Buffer{{ByteLength: 0}},
		Textures:    []Texture{{Sampler: intPtr(9), Source: intPtr(9)}},
		Images:      []Image{{BufferView: intPtr(9)}},
		Samplers:    []Sampler{{}},
		Skins:       []Skin{{Joints: nil, InverseBindMatrices: intPtr(9), Skeleton: intPtr(9)}},
		Animations: []Animation{{
			Channels: []AnimationChannel{{Sampler: 9, Target: AnimationChannelTarget{Node: intPtr(9), Path: PathTranslation}}},
			Samplers: []AnimationSampler{{Input: 9, Output: 9}},
		}},
		Materials: []Material{{
			PBRMetallicRoughness: &PBRMetallicRoughness{
				BaseColorTexture:         &TextureInfo{Index: 9},
				MetallicRoughnessTexture: &TextureInfo{Index: 9},
			},
			NormalTexture:    &NormalTexture{Index: 9},
			OcclusionTexture: &OcclusionTexture{Index: 9},
			EmissiveTexture:  &TextureInfo{Index: 9},
		}},
		Cameras: []Camera{
			{Type: CameraTypePerspective},  // missing perspective
			{Type: CameraTypeOrthographic}, // missing orthographic
			{Type: "weird"},
		},
	}
	err := doc.Validate()
	if err == nil {
		t.Fatal("expected validation errors")
	}
	verrs, ok := AsValidationErrors(err)
	if !ok {
		t.Fatalf("not ValidationErrors: %T", err)
	}
	if len(verrs) < 20 {
		t.Errorf("expected many validation errors, got %d:\n%v", len(verrs), err)
	}

	// A valid skin/camera should not error.
	good := &Document{
		Asset:   Asset{Version: "2.0"},
		Nodes:   []Node{{}},
		Cameras: []Camera{{Type: CameraTypePerspective, Perspective: &CameraPerspective{YFOV: 1, ZNear: 0.1}}},
	}
	if err := good.Validate(); err != nil {
		t.Errorf("valid doc failed: %v", err)
	}
}

func TestEnumStringAndSizeDefaults(t *testing.T) {
	if ComponentType(0).SizeInBytes() != 0 {
		t.Error("unknown component size should be 0")
	}
	if ComponentType(0).String() != "UNKNOWN" {
		t.Error("unknown component string")
	}
	if AccessorType("X").ComponentCount() != 0 {
		t.Error("unknown accessor count should be 0")
	}
	for _, c := range []ComponentType{ComponentByte, ComponentUnsignedByte, ComponentShort, ComponentUnsignedShort, ComponentUnsignedInt} {
		if c.String() == "UNKNOWN" || c.SizeInBytes() == 0 {
			t.Errorf("component %d not handled", c)
		}
	}
	for _, a := range []AccessorType{AccessorVec2, AccessorVec3, AccessorVec4, AccessorMat2, AccessorMat3} {
		if a.ComponentCount() == 0 {
			t.Errorf("accessor %s count 0", a)
		}
	}
	// Primitive mode round trip via pointer.
	m := PrimitiveLines
	p := &Primitive{Mode: &m}
	if p.GetMode() != PrimitiveLines {
		t.Error("GetMode with set mode")
	}
}
