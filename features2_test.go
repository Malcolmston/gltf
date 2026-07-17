package gltf_test

import (
	"math"
	"testing"

	"github.com/malcolmston/gltf"
)

func TestMat4At(t *testing.T) {
	m := gltf.IdentityMatrix()
	if m.At(0, 0) != 1 || m.At(1, 0) != 0 {
		t.Fatalf("At: got %v", m)
	}
}

func TestDecomposeAllAxes(t *testing.T) {
	// Exercise each quatFromMatrix branch with rotations dominant on x, y, z.
	angs := []gltf.Quat{
		{math.Sin(0.6), 0, 0, math.Cos(0.6)}, // x
		{0, math.Sin(0.6), 0, math.Cos(0.6)}, // y
		{0, 0, math.Sin(0.6), math.Cos(0.6)}, // z
		{0.5, 0.5, 0.5, 0.5},                 // trace path
	}
	for i, q := range angs {
		m := gltf.TRS(gltf.Vec3{}, q, gltf.Vec3{1, 1, 1})
		_, gr, _ := m.Decompose()
		m2 := gr.Matrix()
		for k := range m {
			if math.Abs(m[k]-m2[k]) > 1e-5 {
				t.Fatalf("case %d element %d: got %g want %g", i, k, m2[k], m[k])
			}
		}
	}
}

func TestJointMatricesForNode(t *testing.T) {
	d := newDoc()
	// Mesh node at (2,0,0) with a child joint at (3,0,0).
	d.Nodes = []gltf.Node{
		{Translation: &[3]float64{2, 0, 0}, Children: []int{1}},
		{Translation: &[3]float64{3, 0, 0}},
	}
	d.Skins = []gltf.Skin{{Joints: []int{1}}}
	mats, err := d.JointMatricesForNode(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Joint global is (5,0,0); mesh-node inverse maps it back to (3,0,0).
	vecApprox(t, mats[0].TransformPoint(gltf.Vec3{0, 0, 0}), 3, 0, 0)
}

func TestAddAccessorVec2Vec4(t *testing.T) {
	d := newDoc()
	uv := d.AddAccessorVec2([][2]float32{{0, 0}, {1, 1}})
	col := d.AddAccessorVec4([][4]float32{{1, 0, 0, 1}})
	if back, err := d.DecodeAccessorVec2(uv); err != nil || back[1][0] != 1 {
		t.Fatalf("vec2 round trip: %v err=%v", back, err)
	}
	if back, err := d.DecodeAccessorVec4(col); err != nil || back[0][3] != 1 {
		t.Fatalf("vec4 round trip: %v err=%v", back, err)
	}
}

func TestDecodeOutOfRange(t *testing.T) {
	d := newDoc()
	if _, err := d.DecodeAccessorVec3(5); err == nil {
		t.Fatal("expected out-of-range error")
	}
	if _, err := d.GlobalMatrix(9); err == nil {
		t.Fatal("expected node range error")
	}
}

func TestMaterialTransmissionAndSpecGloss(t *testing.T) {
	m := gltf.Material{}
	if err := m.SetExtension(gltf.ExtMaterialsTransmission, gltf.MaterialsTransmission{TransmissionFactor: 0.9}); err != nil {
		t.Fatal(err)
	}
	if err := m.SetExtension(gltf.ExtMaterialsPBRSpecularGlossiness, gltf.MaterialsPBRSpecularGlossiness{GlossinessFactor: fp(0.5)}); err != nil {
		t.Fatal(err)
	}
	if tr, ok := m.Transmission(); !ok || !approx(tr.TransmissionFactor, 0.9) {
		t.Fatalf("transmission: %+v ok=%v", tr, ok)
	}
	if sg, ok := m.SpecularGlossiness(); !ok || sg.GlossinessFactor == nil || !approx(*sg.GlossinessFactor, 0.5) {
		t.Fatalf("spec gloss: %+v ok=%v", sg, ok)
	}
	// Defaults when absent.
	empty := gltf.Material{}
	if v, ok := empty.IOR(); ok || v != 1.5 {
		t.Fatalf("default ior: %v ok=%v", v, ok)
	}
}

func TestNodeLight(t *testing.T) {
	n := gltf.Node{}
	raw, err := gltf.SetExtension(nil, gltf.ExtLightsPunctual, gltf.NodeLight{Light: 3})
	if err != nil {
		t.Fatal(err)
	}
	n.Extensions = raw
	if idx, ok := n.NodeLight(); !ok || idx != 3 {
		t.Fatalf("node light: %d ok=%v", idx, ok)
	}
}

func TestApplyAnimationRotationScaleWeights(t *testing.T) {
	d := newDoc()
	d.Nodes = []gltf.Node{{}, {}, {}}
	in := d.AddAccessorFloat32([]float32{0, 1}, gltf.AccessorScalar)
	rot := d.AddAccessorVec4([][4]float32{{0, 0, 0, 1}, {0, 0, 0, 1}})
	scl := d.AddAccessorVec3([][3]float32{{1, 1, 1}, {3, 3, 3}})
	wts := d.AddAccessorFloat32([]float32{0, 0, 1, 1}, gltf.AccessorScalar) // 2 targets x 2 keyframes
	anim := &gltf.Animation{
		Samplers: []gltf.AnimationSampler{
			{Input: in, Output: rot},
			{Input: in, Output: scl},
			{Input: in, Output: wts},
		},
		Channels: []gltf.AnimationChannel{
			{Sampler: 0, Target: gltf.AnimationChannelTarget{Node: intp(0), Path: gltf.PathRotation}},
			{Sampler: 1, Target: gltf.AnimationChannelTarget{Node: intp(1), Path: gltf.PathScale}},
			{Sampler: 2, Target: gltf.AnimationChannelTarget{Node: intp(2), Path: gltf.PathWeights}},
		},
	}
	if err := d.ApplyAnimation(anim, 0.5); err != nil {
		t.Fatal(err)
	}
	if d.Nodes[0].Rotation == nil || !approx(d.Nodes[0].Rotation[3], 1) {
		t.Fatalf("rotation not applied: %v", d.Nodes[0].Rotation)
	}
	if d.Nodes[1].Scale == nil || !approx(d.Nodes[1].Scale[0], 2) {
		t.Fatalf("scale not applied: %v", d.Nodes[1].Scale)
	}
	if len(d.Nodes[2].Weights) != 2 || !approx(d.Nodes[2].Weights[0], 0.5) {
		t.Fatalf("weights not applied: %v", d.Nodes[2].Weights)
	}
}

func fp(v float64) *float64 { return &v }
