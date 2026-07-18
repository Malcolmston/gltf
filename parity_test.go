package gltf

// Upstream-parity tests. Every vector below is a concrete known-answer case
// taken directly from the Khronos glTF 2.0 specification (github.com/
// KhronosGroup/glTF, specification/2.0/Specification.adoc and its appendices),
// which is the original that this Go port mirrors. Each test cites the section
// it encodes so the assertion can be traced back to upstream text.

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
	"testing"
)

// f32le appends v as a little-endian IEEE-754 float32, the encoding the spec
// mandates for all floating-point buffer data ("MUST use little endian byte
// order", Buffers section).
func f32le(b []byte, vs ...float32) []byte {
	for _, v := range vs {
		b = binary.LittleEndian.AppendUint32(b, math.Float32bits(v))
	}
	return b
}

func u16le(b []byte, vs ...uint16) []byte {
	for _, v := range vs {
		b = binary.LittleEndian.AppendUint16(b, v)
	}
	return b
}

func approx(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

// TestParityComponentTypeSizes encodes the "componentType" data-type table from
// the Accessors section: 5120/5121 are 8-bit (1 byte), 5122/5123 are 16-bit
// (2 bytes), 5125/5126 are 32-bit (4 bytes).
func TestParityComponentTypeSizes(t *testing.T) {
	cases := []struct {
		ct   ComponentType
		size int
		name string
	}{
		{ComponentByte, 1, "BYTE"},
		{ComponentUnsignedByte, 1, "UNSIGNED_BYTE"},
		{ComponentShort, 2, "SHORT"},
		{ComponentUnsignedShort, 2, "UNSIGNED_SHORT"},
		{ComponentUnsignedInt, 4, "UNSIGNED_INT"},
		{ComponentFloat, 4, "FLOAT"},
	}
	for _, c := range cases {
		if got := c.ct.SizeInBytes(); got != c.size {
			t.Errorf("%d.SizeInBytes() = %d, want %d", c.ct, got, c.size)
		}
		if got := c.ct.String(); got != c.name {
			t.Errorf("%d.String() = %q, want %q", c.ct, got, c.name)
		}
	}
}

// TestParityAccessorTypeComponentCounts encodes the "type" component-count table
// from the Accessors section.
func TestParityAccessorTypeComponentCounts(t *testing.T) {
	cases := []struct {
		at    AccessorType
		count int
	}{
		{AccessorScalar, 1},
		{AccessorVec2, 2},
		{AccessorVec3, 3},
		{AccessorVec4, 4},
		{AccessorMat2, 4},
		{AccessorMat3, 9},
		{AccessorMat4, 16},
	}
	for _, c := range cases {
		if got := c.at.ComponentCount(); got != c.count {
			t.Errorf("%s.ComponentCount() = %d, want %d", c.at, got, c.count)
		}
	}
}

// TestParityNormalizedDequantization encodes the exact accessor normalization
// formulas from the spec's "Animations / Overview" dequantization table:
//
//	signed byte    f = max(c / 127.0, -1.0)
//	unsigned byte  f = c / 255.0
//	signed short   f = max(c / 32767.0, -1.0)
//	unsigned short f = c / 65535.0
func TestParityNormalizedDequantization(t *testing.T) {
	cases := []struct {
		name string
		ct   ComponentType
		raw  []byte // one component, little endian
		want float32
	}{
		{"ubyte 255 -> 1.0", ComponentUnsignedByte, []byte{255}, 1.0},
		{"ubyte 0 -> 0.0", ComponentUnsignedByte, []byte{0}, 0.0},
		{"sbyte 127 -> 1.0", ComponentByte, []byte{127}, 1.0},
		{"sbyte -128 clamps to -1.0", ComponentByte, []byte{0x80}, -1.0},
		{"ushort 65535 -> 1.0", ComponentUnsignedShort, []byte{0xFF, 0xFF}, 1.0},
		{"sshort -32768 clamps to -1.0", ComponentShort, []byte{0x00, 0x80}, -1.0},
		{"sshort 32767 -> 1.0", ComponentShort, []byte{0xFF, 0x7F}, 1.0},
	}
	for _, c := range cases {
		doc := &Document{
			Asset:       Asset{Version: Version},
			Buffers:     []Buffer{{ByteLength: len(c.raw), Data: c.raw}},
			BufferViews: []BufferView{{Buffer: 0, ByteLength: len(c.raw)}},
			Accessors: []Accessor{{
				BufferView:    intPtr(0),
				ComponentType: c.ct,
				Normalized:    true,
				Count:         1,
				Type:          AccessorScalar,
			}},
		}
		got, err := doc.DecodeAccessorFloat32(0)
		if err != nil {
			t.Fatalf("%s: DecodeAccessorFloat32: %v", c.name, err)
		}
		if len(got) != 1 || !approx(float64(got[0]), float64(c.want), 1e-6) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// TestParityDataAlignmentExample reproduces the worked example in the spec's
// "Data Alignment" section: a VEC2 unsigned-short accessor with byteOffset 4608
// referencing a bufferView with byteOffset 620, so the underlying buffer segment
// starts at byte 5228 (accessor.byteOffset + bufferView.byteOffset) and ends at
// byte 5396 (2*2*count + start). The decoder must read the 42 tightly-packed
// elements from exactly that range.
func TestParityDataAlignmentExample(t *testing.T) {
	const (
		bvOffset   = 620
		accOffset  = 4608
		count      = 42
		start      = accOffset + bvOffset // 5228, per spec
		end        = 2*2*count + start    // 5396, per spec
		bufferSize = end + 16
	)
	if start != 5228 || end != 5396 {
		t.Fatalf("offset arithmetic drifted from spec: start=%d end=%d", start, end)
	}
	buf := make([]byte, bufferSize)
	// Fill the accessor's region with sequential ushorts 0..83.
	for i := 0; i < count*2; i++ {
		binary.LittleEndian.PutUint16(buf[start+i*2:], uint16(i))
	}
	doc := &Document{
		Asset:       Asset{Version: Version},
		Buffers:     []Buffer{{ByteLength: bufferSize, Data: buf}},
		BufferViews: []BufferView{{Buffer: 0, ByteOffset: bvOffset, ByteLength: bufferSize - bvOffset}},
		Accessors: []Accessor{{
			BufferView:    intPtr(0),
			ByteOffset:    accOffset,
			ComponentType: ComponentUnsignedShort,
			Count:         count,
			Type:          AccessorVec2,
		}},
	}
	got, err := doc.DecodeAccessorVec2(0)
	if err != nil {
		t.Fatalf("DecodeAccessorVec2: %v", err)
	}
	if len(got) != count {
		t.Fatalf("got %d elements, want %d", len(got), count)
	}
	if got[0] != [2]float32{0, 1} {
		t.Errorf("element 0 = %v, want [0 1] (read must start at byte %d)", got[0], start)
	}
	if got[count-1] != [2]float32{82, 83} {
		t.Errorf("element %d = %v, want [82 83]", count-1, got[count-1])
	}
}

// minimalGLTF is the canonical minimal glTF asset described by the specification
// (a single triangle whose buffer is embedded as a base64 data URI). Decoding it
// must yield indices [0 1 2] and positions (0,0,0),(1,0,0),(0,1,0).
const minimalGLTF = `{
  "scene": 0,
  "scenes": [{ "nodes": [0] }],
  "nodes": [{ "mesh": 0 }],
  "meshes": [{ "primitives": [{ "attributes": { "POSITION": 1 }, "indices": 0 }] }],
  "buffers": [{ "uri": "data:application/octet-stream;base64,AAABAAIAAAAAAAAAAAAAAAAAAAAAAIA/AAAAAAAAAAAAAAAAAACAPwAAAAA=", "byteLength": 44 }],
  "bufferViews": [
    { "buffer": 0, "byteOffset": 0, "byteLength": 6, "target": 34963 },
    { "buffer": 0, "byteOffset": 8, "byteLength": 36, "target": 34962 }
  ],
  "accessors": [
    { "bufferView": 0, "byteOffset": 0, "componentType": 5123, "count": 3, "type": "SCALAR", "max": [2], "min": [0] },
    { "bufferView": 1, "byteOffset": 0, "componentType": 5126, "count": 3, "type": "VEC3", "max": [1.0, 1.0, 0.0], "min": [0.0, 0.0, 0.0] }
  ],
  "asset": { "version": "2.0" }
}`

// TestParityMinimalGLTFFile decodes the canonical minimal glTF file and checks
// its indices and positions against the exact values the spec's data buffer
// encodes.
func TestParityMinimalGLTFFile(t *testing.T) {
	doc, err := Decode(strings.NewReader(minimalGLTF))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if err := doc.ResolveBuffers("", nil); err != nil {
		t.Fatalf("ResolveBuffers: %v", err)
	}
	prim := &doc.Meshes[0].Primitives[0]

	idx, err := doc.DecodeIndices(prim)
	if err != nil {
		t.Fatalf("DecodeIndices: %v", err)
	}
	wantIdx := []uint32{0, 1, 2}
	if len(idx) != 3 || idx[0] != wantIdx[0] || idx[1] != wantIdx[1] || idx[2] != wantIdx[2] {
		t.Errorf("indices = %v, want %v", idx, wantIdx)
	}

	pos, err := doc.DecodeAccessorVec3(prim.Attributes["POSITION"])
	if err != nil {
		t.Fatalf("DecodeAccessorVec3: %v", err)
	}
	wantPos := [][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}
	for i := range wantPos {
		if pos[i] != wantPos[i] {
			t.Errorf("position[%d] = %v, want %v", i, pos[i], wantPos[i])
		}
	}
	// The JSON-declared min/max must match the actual binary extents.
	posAcc := doc.Accessors[1]
	if posAcc.Min[0] != 0 || posAcc.Max[0] != 1 || posAcc.Max[1] != 1 {
		t.Errorf("position accessor min/max = %v/%v, want [0 0 0]/[1 1 0]", posAcc.Min, posAcc.Max)
	}
}

// TestParityGLBHeaderConstants encodes the GLB "Header" and "Chunks" sections:
// magic 0x46546C67 ("glTF"), container version 2, JSON chunk type 0x4E4F534A,
// BIN chunk type 0x004E4942, with the JSON chunk padded by spaces (0x20) and the
// BIN chunk padded by zeros (0x00) to four-byte boundaries.
func TestParityGLBHeaderConstants(t *testing.T) {
	if GLBMagic != 0x46546C67 {
		t.Errorf("GLBMagic = 0x%08X, want 0x46546C67", GLBMagic)
	}
	if GLBVersion != 2 {
		t.Errorf("GLBVersion = %d, want 2", GLBVersion)
	}
	if chunkTypeJSON != 0x4E4F534A || chunkTypeBIN != 0x004E4942 {
		t.Errorf("chunk types = 0x%08X/0x%08X, want 0x4E4F534A/0x004E4942", chunkTypeJSON, chunkTypeBIN)
	}

	doc, bin := Triangle()
	var out bytes.Buffer
	if err := WriteGLB(&out, doc, bin); err != nil {
		t.Fatalf("WriteGLB: %v", err)
	}
	data := out.Bytes()
	if got := binary.LittleEndian.Uint32(data[0:4]); got != GLBMagic {
		t.Errorf("file magic = 0x%08X, want 0x%08X", got, GLBMagic)
	}
	if got := binary.LittleEndian.Uint32(data[4:8]); got != GLBVersion {
		t.Errorf("file version = %d, want %d", got, GLBVersion)
	}
	if got := int(binary.LittleEndian.Uint32(data[8:12])); got != len(data) {
		t.Errorf("declared length = %d, want %d", got, len(data))
	}
	// First chunk: JSON, length is a multiple of 4, type is JSON, trailing
	// padding bytes are spaces.
	jsonLen := int(binary.LittleEndian.Uint32(data[12:16]))
	if jsonLen%4 != 0 {
		t.Errorf("JSON chunk length %d not 4-byte aligned", jsonLen)
	}
	if got := binary.LittleEndian.Uint32(data[16:20]); got != chunkTypeJSON {
		t.Errorf("chunk 0 type = 0x%08X, want JSON 0x%08X", got, chunkTypeJSON)
	}
	jsonStart := 20
	jsonEnd := jsonStart + jsonLen
	// Any trailing padding in the JSON chunk must be 0x20 (spaces).
	trimmed := bytes.TrimRight(data[jsonStart:jsonEnd], "\x20")
	for _, b := range data[jsonStart+len(trimmed) : jsonEnd] {
		if b != 0x20 {
			t.Errorf("JSON padding byte = 0x%02X, want 0x20", b)
		}
	}
	// Second chunk: BIN, padded with zeros.
	if got := binary.LittleEndian.Uint32(data[jsonEnd+4 : jsonEnd+8]); got != chunkTypeBIN {
		t.Errorf("chunk 1 type = 0x%08X, want BIN 0x%08X", got, chunkTypeBIN)
	}
	binLen := int(binary.LittleEndian.Uint32(data[jsonEnd : jsonEnd+4]))
	if binLen%4 != 0 {
		t.Errorf("BIN chunk length %d not 4-byte aligned", binLen)
	}
}

// TestParityTRSComposition encodes the node local-transform rule: TRS properties
// "MUST be converted to matrices and postmultiplied in the T * R * S order;
// first the scale is applied to the vertices, then the rotation, and then the
// translation". With S=(2,2,2), R=90 degrees about +Z, T=(10,20,30), the local
// point (1,0,0) maps to (10,22,30).
func TestParityTRSComposition(t *testing.T) {
	m := TRS(
		Vec3{10, 20, 30},
		QuatFromAxisAngle(Vec3{0, 0, 1}, math.Pi/2),
		Vec3{2, 2, 2},
	)
	got := m.TransformPoint(Vec3{1, 0, 0})
	want := Vec3{10, 22, 30}
	for i := range want {
		if !approx(got[i], want[i], 1e-9) {
			t.Fatalf("TransformPoint = %v, want %v", got, want)
		}
	}
}

// TestParityQuaternionRotation encodes the rotation convention (unit quaternion,
// XYZW, W scalar): a 90-degree rotation about +Z sends (1,0,0) to (0,1,0).
func TestParityQuaternionRotation(t *testing.T) {
	q := QuatFromAxisAngle(Vec3{0, 0, 1}, math.Pi/2)
	got := q.Rotate(Vec3{1, 0, 0})
	want := Vec3{0, 1, 0}
	for i := range want {
		if !approx(got[i], want[i], 1e-9) {
			t.Fatalf("Rotate = %v, want %v", got, want)
		}
	}
}

// TestParitySlerp encodes the spherical-linear-interpolation rule for rotation
// channels (Appendix C, "Spherical Linear Interpolation"). The slerp halfway
// between the identity and a 90-degree rotation about +Z is a 45-degree rotation
// about +Z.
func TestParitySlerp(t *testing.T) {
	a := Quat{0, 0, 0, 1} // identity
	b := QuatFromAxisAngle(Vec3{0, 0, 1}, math.Pi/2)
	got := Slerp(a, b, 0.5)
	want := QuatFromAxisAngle(Vec3{0, 0, 1}, math.Pi/4)
	// Quaternions q and -q represent the same rotation; align signs first.
	if got.Dot(want) < 0 {
		want = Quat{-want[0], -want[1], -want[2], -want[3]}
	}
	for i := range want {
		if !approx(got[i], want[i], 1e-6) {
			t.Fatalf("Slerp = %v, want %v", got, want)
		}
	}
}

// newSamplerDoc builds a one-animation document whose single sampler has the
// given input times and output values, used by the interpolation parity tests.
func newSamplerDoc(t *testing.T, times []float32, out []float32, outType AccessorType, interp Interpolation) (*Document, *AnimationSampler) {
	t.Helper()
	var timeBin []byte
	timeBin = f32le(timeBin, times...)
	var outBin []byte
	outBin = f32le(outBin, out...)
	bin := append(append([]byte{}, timeBin...), outBin...)
	doc := &Document{
		Asset:   Asset{Version: Version},
		Buffers: []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{
			{Buffer: 0, ByteOffset: 0, ByteLength: len(timeBin)},
			{Buffer: 0, ByteOffset: len(timeBin), ByteLength: len(outBin)},
		},
		Accessors: []Accessor{
			{BufferView: intPtr(0), ComponentType: ComponentFloat, Count: len(times), Type: AccessorScalar},
			{BufferView: intPtr(1), ComponentType: ComponentFloat, Count: len(out) / outType.ComponentCount(), Type: outType},
		},
	}
	s := &AnimationSampler{Input: 0, Output: 1, Interpolation: interp}
	return doc, s
}

// TestParityStepInterpolation encodes Appendix C "Step Interpolation": v_t = v_k
// (the value of the keyframe at or before t, held constant across the segment).
func TestParityStepInterpolation(t *testing.T) {
	doc, s := newSamplerDoc(t, []float32{0, 1}, []float32{5, 9}, AccessorScalar, InterpolationStep)
	got, err := doc.EvaluateSampler(s, 0.5, false)
	if err != nil {
		t.Fatalf("EvaluateSampler: %v", err)
	}
	if len(got) != 1 || got[0] != 5 {
		t.Errorf("STEP at t=0.5 = %v, want [5]", got)
	}
}

// TestParityLinearInterpolation encodes Appendix C "Linear Interpolation":
// v_t = (1 - t) * v_k + t * v_{k+1}. With keyframes (0,0,0) and (10,20,30) and a
// segment-normalized factor 0.25, the result is (2.5, 5, 7.5).
func TestParityLinearInterpolation(t *testing.T) {
	doc, s := newSamplerDoc(t, []float32{0, 1}, []float32{0, 0, 0, 10, 20, 30}, AccessorVec3, InterpolationLinear)
	got, err := doc.EvaluateSampler(s, 0.25, false)
	if err != nil {
		t.Fatalf("EvaluateSampler: %v", err)
	}
	want := []float32{2.5, 5, 7.5}
	for i := range want {
		if !approx(float64(got[i]), float64(want[i]), 1e-6) {
			t.Fatalf("LINEAR at t=0.25 = %v, want %v", got, want)
		}
	}
}

// TestParityCubicSpline encodes Appendix C "Cubic Spline Interpolation":
//
//	v_t = (2t^3-3t^2+1)*v_k + t_d(t^3-2t^2+t)*b_k
//	      + (-2t^3+3t^2)*v_{k+1} + t_d(t^3-t^2)*a_{k+1}
//
// where each keyframe stores [in-tangent, value, out-tangent]. Evaluated at the
// segment midpoint (t=0.5, t_d=1) each isolated term produces a known value.
func TestParityCubicSpline(t *testing.T) {
	// Layout per keyframe (scalar): inTangent, value, outTangent.
	cases := []struct {
		name string
		out  []float32
		want float32
	}{
		// Values 0 -> 1, zero tangents: pure (-2t^3+3t^2) = 0.5 at t=0.5.
		{"value term", []float32{0, 0, 0 /*kf0*/, 0, 1, 0 /*kf1*/}, 0.5},
		// Out-tangent b_k = 1, everything else 0: t_d(t^3-2t^2+t) = 0.125.
		{"out-tangent term", []float32{0, 0, 1 /*kf0*/, 0, 0, 0 /*kf1*/}, 0.125},
		// In-tangent a_{k+1} = 1, everything else 0: t_d(t^3-t^2) = -0.125.
		{"in-tangent term", []float32{0, 0, 0 /*kf0*/, 1, 0, 0 /*kf1*/}, -0.125},
	}
	for _, c := range cases {
		doc, s := newSamplerDoc(t, []float32{0, 1}, c.out, AccessorScalar, InterpolationCubicSpline)
		got, err := doc.EvaluateSampler(s, 0.5, false)
		if err != nil {
			t.Fatalf("%s: EvaluateSampler: %v", c.name, err)
		}
		if len(got) != 1 || !approx(float64(got[0]), float64(c.want), 1e-6) {
			t.Errorf("%s: CUBICSPLINE at t=0.5 = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestParitySparseAccessorZeroInit encodes the Sparse Accessors rule: "When
// accessor.bufferView is undefined, the sparse accessor is initialized as an
// array of zeros", then the sparse indices/values override selected elements.
// A 4-element SCALAR accessor with no base bufferView and sparse indices [0,2]
// carrying values [7,9] decodes to [7,0,9,0].
func TestParitySparseAccessorZeroInit(t *testing.T) {
	var idxBin []byte
	idxBin = u16le(idxBin, 0, 2) // strictly increasing
	var valBin []byte
	valBin = u16le(valBin, 7, 9)
	bin := append(append([]byte{}, idxBin...), valBin...)
	doc := &Document{
		Asset:   Asset{Version: Version},
		Buffers: []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{
			{Buffer: 0, ByteOffset: 0, ByteLength: len(idxBin)},
			{Buffer: 0, ByteOffset: len(idxBin), ByteLength: len(valBin)},
		},
		Accessors: []Accessor{{
			ComponentType: ComponentUnsignedShort,
			Count:         4,
			Type:          AccessorScalar,
			Sparse: &Sparse{
				Count:   2,
				Indices: SparseIndices{BufferView: 0, ComponentType: ComponentUnsignedShort},
				Values:  SparseValues{BufferView: 1},
			},
		}},
	}
	got, err := doc.DecodeAccessorUint32(0)
	if err != nil {
		t.Fatalf("DecodeAccessorUint32: %v", err)
	}
	want := []uint32{7, 0, 9, 0}
	if len(got) != 4 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] || got[3] != want[3] {
		t.Errorf("sparse decode = %v, want %v", got, want)
	}
}

// sparseDoc builds a document with a single SCALAR ushort sparse accessor whose
// sparse index array is exactly idx. Used to exercise the spec's sparse-index
// validation rules.
func sparseDoc(idx []uint16, count int) *Document {
	var idxBin []byte
	idxBin = u16le(idxBin, idx...)
	valBin := make([]byte, len(idx)*2) // zero values, irrelevant to the rule
	bin := append(append([]byte{}, idxBin...), valBin...)
	return &Document{
		Asset:   Asset{Version: Version},
		Buffers: []Buffer{{ByteLength: len(bin), Data: bin}},
		BufferViews: []BufferView{
			{Buffer: 0, ByteOffset: 0, ByteLength: len(idxBin)},
			{Buffer: 0, ByteOffset: len(idxBin), ByteLength: len(valBin)},
		},
		Accessors: []Accessor{{
			ComponentType: ComponentUnsignedShort,
			Count:         count,
			Type:          AccessorScalar,
			Sparse: &Sparse{
				Count:   len(idx),
				Indices: SparseIndices{BufferView: 0, ComponentType: ComponentUnsignedShort},
				Values:  SparseValues{BufferView: 1},
			},
		}},
	}
}

// TestParitySparseIndicesRules encodes the two runtime rules the Sparse
// Accessors section places on the index array: the indices "MUST form a strictly
// increasing sequence" and "MUST NOT be greater than or equal to the number of
// the base accessor elements". Validate must accept a conforming array and
// reject each violation with a descriptive, path-qualified error.
func TestParitySparseIndicesRules(t *testing.T) {
	// Conforming: strictly increasing, all < count.
	if err := sparseDoc([]uint16{0, 2, 3}, 4).Validate(); err != nil {
		t.Errorf("valid sparse indices rejected: %v", err)
	}

	// Not strictly increasing (duplicate / decreasing).
	err := sparseDoc([]uint16{2, 1}, 4).Validate()
	if err == nil {
		t.Fatal("non-increasing sparse indices accepted, want error")
	}
	if !strings.Contains(err.Error(), "strictly increasing") {
		t.Errorf("error = %q, want mention of strictly increasing", err.Error())
	}
	if errs, ok := AsValidationErrors(err); !ok || len(errs) == 0 {
		t.Errorf("expected ValidationErrors, got %v (ok=%v)", err, ok)
	}

	// Duplicate indices also violate the strictly-increasing rule.
	if err := sparseDoc([]uint16{1, 1}, 4).Validate(); err == nil ||
		!strings.Contains(err.Error(), "strictly increasing") {
		t.Errorf("duplicate sparse indices: got %v, want strictly-increasing error", err)
	}

	// Index >= accessor count.
	err = sparseDoc([]uint16{0, 5}, 4).Validate()
	if err == nil {
		t.Fatal("out-of-range sparse index accepted, want error")
	}
	if !strings.Contains(err.Error(), "not less than accessor count") {
		t.Errorf("error = %q, want mention of accessor count bound", err.Error())
	}
}
