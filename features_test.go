package gltf_test

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"

	"github.com/malcolmston/gltf"
)

const eps = 1e-5

func approx(a, b float64) bool { return math.Abs(a-b) <= eps }

func vecApprox(t *testing.T, got gltf.Vec3, wx, wy, wz float64) {
	t.Helper()
	if !approx(got[0], wx) || !approx(got[1], wy) || !approx(got[2], wz) {
		t.Fatalf("got %v, want [%g %g %g]", got, wx, wy, wz)
	}
}

func newDoc() *gltf.Document {
	return &gltf.Document{Asset: gltf.Asset{Version: "2.0"}}
}

// --- Transform math ---------------------------------------------------------

func TestTRSMatrixAndTransform(t *testing.T) {
	m := gltf.TRS(gltf.Vec3{1, 2, 3}, gltf.IdentityQuat, gltf.Vec3{2, 2, 2})
	// Point (1,0,0) scaled by 2 -> (2,0,0), then translated -> (3,2,3).
	vecApprox(t, m.TransformPoint(gltf.Vec3{1, 0, 0}), 3, 2, 3)
}

func TestQuatRotation(t *testing.T) {
	// 90 degrees about +Z should map +X to +Y.
	half := math.Pi / 4
	q := gltf.Quat{0, 0, math.Sin(half), math.Cos(half)}
	m := q.Matrix()
	vecApprox(t, m.TransformPoint(gltf.Vec3{1, 0, 0}), 0, 1, 0)
}

func TestDecomposeRoundTrip(t *testing.T) {
	tr := gltf.Vec3{5, -3, 2}
	half := math.Pi / 6
	rot := gltf.Quat{0, math.Sin(half), 0, math.Cos(half)}
	sc := gltf.Vec3{2, 3, 4}
	m := gltf.TRS(tr, rot, sc)

	gt, gr, gs := m.Decompose()
	vecApprox(t, gt, tr[0], tr[1], tr[2])
	vecApprox(t, gs, sc[0], sc[1], sc[2])
	// Recompose and compare matrices element-wise.
	m2 := gltf.TRS(gt, gr, gs)
	for i := range m {
		if !approx(m[i], m2[i]) {
			t.Fatalf("recomposed matrix element %d: got %g want %g", i, m2[i], m[i])
		}
	}
}

func TestInverse(t *testing.T) {
	m := gltf.TRS(gltf.Vec3{1, 2, 3}, gltf.Quat{0, 0, math.Sin(0.3), math.Cos(0.3)}, gltf.Vec3{2, 2, 2})
	inv, ok := m.Inverse()
	if !ok {
		t.Fatal("expected invertible")
	}
	id := m.Mul(inv)
	want := gltf.IdentityMatrix()
	for i := range id {
		if !approx(id[i], want[i]) {
			t.Fatalf("m*inv element %d = %g, want %g", i, id[i], want[i])
		}
	}
}

func TestGlobalMatrixHierarchy(t *testing.T) {
	d := newDoc()
	d.Nodes = []gltf.Node{
		{Translation: &[3]float64{10, 0, 0}, Children: []int{1}},
		{Translation: &[3]float64{0, 5, 0}},
	}
	g, err := d.GlobalMatrix(1)
	if err != nil {
		t.Fatal(err)
	}
	// Child origin in world space = parent(10,0,0) + child(0,5,0).
	vecApprox(t, g.TransformPoint(gltf.Vec3{0, 0, 0}), 10, 5, 0)
}

func TestGlobalMatrixCycle(t *testing.T) {
	d := newDoc()
	d.Nodes = []gltf.Node{{Children: []int{1}}, {Children: []int{0}}}
	if _, err := d.GlobalMatrix(0); err == nil {
		t.Fatal("expected cycle error")
	}
}

// --- Animation --------------------------------------------------------------

func TestEvaluateSamplerLinear(t *testing.T) {
	d := newDoc()
	in := d.AddAccessorFloat32([]float32{0, 1, 2}, gltf.AccessorScalar)
	out := d.AddAccessorVec3([][3]float32{{0, 0, 0}, {10, 0, 0}, {20, 0, 0}})
	s := &gltf.AnimationSampler{Input: in, Output: out, Interpolation: gltf.InterpolationLinear}

	v, err := d.EvaluateSampler(s, 0.5, false)
	if err != nil {
		t.Fatal(err)
	}
	if !approx(float64(v[0]), 5) {
		t.Fatalf("linear at t=0.5: got %v want x=5", v)
	}
	// Clamp beyond end.
	v, _ = d.EvaluateSampler(s, 99, false)
	if !approx(float64(v[0]), 20) {
		t.Fatalf("clamp: got %v want x=20", v)
	}
}

func TestEvaluateSamplerStep(t *testing.T) {
	d := newDoc()
	in := d.AddAccessorFloat32([]float32{0, 1}, gltf.AccessorScalar)
	out := d.AddAccessorVec3([][3]float32{{0, 0, 0}, {10, 0, 0}})
	s := &gltf.AnimationSampler{Input: in, Output: out, Interpolation: gltf.InterpolationStep}
	v, _ := d.EvaluateSampler(s, 0.9, false)
	if !approx(float64(v[0]), 0) {
		t.Fatalf("step at t=0.9: got %v want x=0", v)
	}
}

func TestEvaluateSamplerCubicSpline(t *testing.T) {
	d := newDoc()
	in := d.AddAccessorFloat32([]float32{0, 1}, gltf.AccessorScalar)
	// Per keyframe: inTangent, value, outTangent. Tangents zero.
	out := d.AddAccessorVec3([][3]float32{
		{0, 0, 0}, {0, 0, 0}, {0, 0, 0}, // keyframe 0
		{0, 0, 0}, {1, 0, 0}, {0, 0, 0}, // keyframe 1
	})
	s := &gltf.AnimationSampler{Input: in, Output: out, Interpolation: gltf.InterpolationCubicSpline}
	v, err := d.EvaluateSampler(s, 0.5, false)
	if err != nil {
		t.Fatal(err)
	}
	// With zero tangents the Hermite reduces to h00*p0 + h01*p1 = 0.5.
	if !approx(float64(v[0]), 0.5) {
		t.Fatalf("cubicspline at t=0.5: got %v want x=0.5", v)
	}
}

func TestEvaluateSamplerRotationSlerp(t *testing.T) {
	d := newDoc()
	in := d.AddAccessorFloat32([]float32{0, 1}, gltf.AccessorScalar)
	s45 := float32(math.Sin(math.Pi / 4))
	c45 := float32(math.Cos(math.Pi / 4))
	out := d.AddAccessorVec4([][4]float32{{0, 0, 0, 1}, {0, 0, s45, c45}})
	s := &gltf.AnimationSampler{Input: in, Output: out, Interpolation: gltf.InterpolationLinear}
	v, _ := d.EvaluateSampler(s, 0.5, true)
	// Halfway is a 22.5-degree rotation about Z.
	wantZ := math.Sin(math.Pi / 8)
	wantW := math.Cos(math.Pi / 8)
	if !approx(float64(v[2]), wantZ) || !approx(float64(v[3]), wantW) {
		t.Fatalf("slerp: got %v want z=%g w=%g", v, wantZ, wantW)
	}
}

func TestApplyAnimation(t *testing.T) {
	d := newDoc()
	d.Nodes = []gltf.Node{{}}
	in := d.AddAccessorFloat32([]float32{0, 1}, gltf.AccessorScalar)
	out := d.AddAccessorVec3([][3]float32{{0, 0, 0}, {4, 0, 0}})
	anim := &gltf.Animation{
		Samplers: []gltf.AnimationSampler{{Input: in, Output: out, Interpolation: gltf.InterpolationLinear}},
		Channels: []gltf.AnimationChannel{{Sampler: 0, Target: gltf.AnimationChannelTarget{Node: intp(0), Path: gltf.PathTranslation}}},
	}
	if err := d.ApplyAnimation(anim, 0.5); err != nil {
		t.Fatal(err)
	}
	if d.Nodes[0].Translation == nil || !approx(d.Nodes[0].Translation[0], 2) {
		t.Fatalf("apply animation: got %v want x=2", d.Nodes[0].Translation)
	}
}

// --- Skinning ---------------------------------------------------------------

func TestJointMatrices(t *testing.T) {
	d := newDoc()
	d.Nodes = []gltf.Node{{Translation: &[3]float64{5, 0, 0}}}
	d.Skins = []gltf.Skin{{Joints: []int{0}}} // no IBM -> identity
	mats, err := d.JointMatrices(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(mats) != 1 {
		t.Fatalf("want 1 joint matrix, got %d", len(mats))
	}
	vecApprox(t, mats[0].TransformPoint(gltf.Vec3{0, 0, 0}), 5, 0, 0)
}

func TestJointMatricesWithIBM(t *testing.T) {
	d := newDoc()
	d.Nodes = []gltf.Node{{Translation: &[3]float64{5, 0, 0}}}
	// IBM = inverse translation (-5,0,0) so joint*IBM = identity.
	ibm := gltf.TRS(gltf.Vec3{-5, 0, 0}, gltf.IdentityQuat, gltf.Vec3{1, 1, 1})
	flat := make([]float32, 16)
	for i := range flat {
		flat[i] = float32(ibm[i])
	}
	acc := d.AddAccessorFloat32(flat, gltf.AccessorMat4)
	d.Skins = []gltf.Skin{{Joints: []int{0}, InverseBindMatrices: intp(acc)}}
	mats, err := d.JointMatrices(0)
	if err != nil {
		t.Fatal(err)
	}
	vecApprox(t, mats[0].TransformPoint(gltf.Vec3{1, 1, 1}), 1, 1, 1)
}

// --- Morph targets ----------------------------------------------------------

func TestMorphBlend(t *testing.T) {
	d := newDoc()
	pos := d.AddAccessorVec3([][3]float32{{0, 0, 0}, {1, 1, 1}})
	tgt := d.AddAccessorVec3([][3]float32{{1, 0, 0}, {0, 1, 0}})
	prim := &gltf.Primitive{
		Attributes: map[string]int{"POSITION": pos},
		Targets:    []map[string]int{{"POSITION": tgt}},
	}
	got, err := d.MorphedPositions(prim, []float64{0.5})
	if err != nil {
		t.Fatal(err)
	}
	if !approx(float64(got[0][0]), 0.5) || !approx(float64(got[1][1]), 1.5) {
		t.Fatalf("morph blend: got %v", got)
	}
}

func TestEffectiveWeights(t *testing.T) {
	mesh := &gltf.Mesh{Weights: []float64{1, 2}}
	node := &gltf.Node{Weights: []float64{9}}
	if w := gltf.EffectiveWeights(node, mesh); len(w) != 1 || w[0] != 9 {
		t.Fatalf("node weights should override: got %v", w)
	}
	if w := gltf.EffectiveWeights(nil, mesh); len(w) != 2 {
		t.Fatalf("mesh weights fallback: got %v", w)
	}
}

// --- Accessor encode -> decode round trip -----------------------------------

func TestAccessorEncodeDecodeRoundTrip(t *testing.T) {
	d := newDoc()
	verts := [][3]float32{{-1, 2, 3}, {4, -5, 6}, {7, 8, -9}}
	idx := d.AddAccessorVec3(verts)
	back, err := d.DecodeAccessorVec3(idx)
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != len(verts) {
		t.Fatalf("len mismatch: %d vs %d", len(back), len(verts))
	}
	for i := range verts {
		for c := 0; c < 3; c++ {
			if back[i][c] != verts[i][c] {
				t.Fatalf("vert %d comp %d: got %v want %v", i, c, back[i][c], verts[i][c])
			}
		}
	}
	// min/max should be computed.
	a := d.Accessors[idx]
	if a.Min[0] != -1 || a.Max[0] != 7 {
		t.Fatalf("min/max x: got min=%v max=%v", a.Min, a.Max)
	}

	ii := d.AddIndicesUint32([]uint32{0, 1, 2, 2, 1, 0})
	indices, err := d.DecodeAccessorUint32(ii)
	if err != nil {
		t.Fatal(err)
	}
	if len(indices) != 6 || indices[3] != 2 {
		t.Fatalf("indices round trip: got %v", indices)
	}
	// A document assembled purely from Add* helpers must validate.
	d.Meshes = []gltf.Mesh{{Primitives: []gltf.Primitive{{Attributes: map[string]int{"POSITION": idx}, Indices: intp(ii)}}}}
	if err := d.Validate(); err != nil {
		t.Fatalf("assembled document should validate: %v", err)
	}
}

// --- Extensions round trip --------------------------------------------------

func TestExtensionRoundTrip(t *testing.T) {
	d := newDoc()
	m := gltf.Material{Name: "glass"}
	if err := m.SetExtension(gltf.ExtMaterialsUnlit, gltf.MaterialsUnlit{}); err != nil {
		t.Fatal(err)
	}
	if err := m.SetExtension(gltf.ExtMaterialsIOR, gltf.MaterialsIOR{IOR: 1.33}); err != nil {
		t.Fatal(err)
	}
	if err := m.SetExtension(gltf.ExtMaterialsEmissiveStrength, gltf.MaterialsEmissiveStrength{EmissiveStrength: 5}); err != nil {
		t.Fatal(err)
	}
	// An extension this package does not model must survive round-tripping.
	if err := m.SetExtension("VENDOR_custom", map[string]any{"foo": 42}); err != nil {
		t.Fatal(err)
	}
	d.Materials = []gltf.Material{m}

	var buf bytes.Buffer
	if err := gltf.Encode(&buf, d); err != nil {
		t.Fatal(err)
	}
	loaded, err := gltf.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	lm := &loaded.Materials[0]
	if !lm.Unlit() {
		t.Error("unlit not preserved")
	}
	if ior, ok := lm.IOR(); !ok || !approx(ior, 1.33) {
		t.Errorf("ior: got %v ok=%v", ior, ok)
	}
	if es, ok := lm.EmissiveStrength(); !ok || !approx(es, 5) {
		t.Errorf("emissive strength: got %v ok=%v", es, ok)
	}
	extMap, err := gltf.ExtensionMap(lm.Extensions)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := extMap["VENDOR_custom"]; !ok {
		t.Errorf("unknown extension not preserved: %v", extMap)
	}
}

func TestLightsPunctualRoundTrip(t *testing.T) {
	d := newDoc()
	raw, err := gltf.SetExtension(nil, gltf.ExtLightsPunctual, gltf.LightsPunctual{
		Lights: []gltf.Light{{Name: "sun", Type: gltf.LightTypeDirectional}},
	})
	if err != nil {
		t.Fatal(err)
	}
	d.Extensions = raw
	lights, ok := d.Lights()
	if !ok || len(lights) != 1 || lights[0].Type != gltf.LightTypeDirectional {
		t.Fatalf("lights: got %v ok=%v", lights, ok)
	}
}

func TestTextureTransform(t *testing.T) {
	ti := &gltf.TextureInfo{Index: 0}
	raw, err := gltf.SetExtension(nil, gltf.ExtTextureTransform, gltf.TextureTransform{
		Offset: &[2]float64{0.5, 0.25}, Scale: &[2]float64{2, 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	ti.Extensions = raw
	tt, ok := ti.TextureTransform()
	if !ok || tt.Offset[0] != 0.5 || tt.Scale[1] != 2 {
		t.Fatalf("texture transform: got %+v ok=%v", tt, ok)
	}
	_ = tt.UVMatrix()
}

// --- Image decoding ---------------------------------------------------------

func TestDecodeImageDataURI(t *testing.T) {
	// Build a 2x2 PNG and embed it as a data URI.
	im := image.NewRGBA(image.Rect(0, 0, 2, 2))
	im.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, im); err != nil {
		t.Fatal(err)
	}
	uri := "data:image/png;base64," + b64(pngBuf.Bytes())

	d := newDoc()
	d.Images = []gltf.Image{{URI: uri}}
	decoded, format, err := d.DecodeImage(0, "")
	if err != nil {
		t.Fatal(err)
	}
	if format != "png" || decoded.Bounds().Dx() != 2 {
		t.Fatalf("image decode: format=%q bounds=%v", format, decoded.Bounds())
	}
}

func TestDecodeImageBufferView(t *testing.T) {
	im := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, im); err != nil {
		t.Fatal(err)
	}
	d := newDoc()
	bv := d.AddBinData(pngBuf.Bytes())
	d.Images = []gltf.Image{{BufferView: intp(bv), MimeType: "image/png"}}
	_, format, err := d.DecodeImage(0, "")
	if err != nil {
		t.Fatal(err)
	}
	if format != "png" {
		t.Fatalf("format=%q", format)
	}
}

// --- Expanded validation ----------------------------------------------------

func TestValidateExpanded(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(d *gltf.Document)
		wantSub string
	}{
		{"normalized float", func(d *gltf.Document) {
			d.Accessors = []gltf.Accessor{{ComponentType: gltf.ComponentFloat, Type: gltf.AccessorVec3, Count: 1, Normalized: true}}
		}, "normalized"},
		{"min length", func(d *gltf.Document) {
			d.Accessors = []gltf.Accessor{{ComponentType: gltf.ComponentFloat, Type: gltf.AccessorVec3, Count: 1, Min: []float64{0}}}
		}, "min has"},
		{"required ext", func(d *gltf.Document) {
			d.ExtensionsRequired = []string{"KHR_missing"}
		}, "not listed"},
		{"camera znear", func(d *gltf.Document) {
			d.Cameras = []gltf.Camera{{Type: gltf.CameraTypePerspective, Perspective: &gltf.CameraPerspective{YFOV: 1, ZNear: -1}}}
		}, "znear"},
		{"image both", func(d *gltf.Document) {
			d.Images = []gltf.Image{{URI: "x.png", BufferView: intp(0)}}
			d.BufferViews = []gltf.BufferView{{Buffer: 0, ByteLength: 1}}
			d.Buffers = []gltf.Buffer{{ByteLength: 1}}
		}, "both uri and bufferView"},
		{"bad interp", func(d *gltf.Document) {
			d.Accessors = []gltf.Accessor{{ComponentType: gltf.ComponentFloat, Type: gltf.AccessorScalar, Count: 1}}
			d.Animations = []gltf.Animation{{
				Samplers: []gltf.AnimationSampler{{Input: 0, Output: 0, Interpolation: "BOGUS"}},
				Channels: []gltf.AnimationChannel{{Sampler: 0, Target: gltf.AnimationChannelTarget{Path: gltf.PathTranslation}}},
			}}
		}, "interpolation"},
		{"bufferView overrun", func(d *gltf.Document) {
			d.Buffers = []gltf.Buffer{{ByteLength: 4}}
			d.BufferViews = []gltf.BufferView{{Buffer: 0, ByteOffset: 2, ByteLength: 8}}
		}, "exceeds buffer"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newDoc()
			tc.mutate(d)
			err := d.Validate()
			if err == nil {
				t.Fatalf("expected validation error containing %q", tc.wantSub)
			}
			if !bytes.Contains([]byte(err.Error()), []byte(tc.wantSub)) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestValidateValidDocument(t *testing.T) {
	doc, _ := gltf.Triangle()
	if err := doc.Validate(); err != nil {
		t.Fatalf("triangle should be valid: %v", err)
	}
}

// helpers

func intp(v int) *int { return &v }

func b64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
