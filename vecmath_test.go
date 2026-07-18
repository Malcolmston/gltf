package gltf

import (
	"math"
	"testing"
)

func vec3Close(a, b Vec3, eps float64) bool {
	for i := 0; i < 3; i++ {
		if math.Abs(a[i]-b[i]) > eps {
			return false
		}
	}
	return true
}

func quatClose(a, b Quat, eps float64) bool {
	for i := 0; i < 4; i++ {
		if math.Abs(a[i]-b[i]) > eps {
			return false
		}
	}
	return true
}

func mat4Close(a, b Mat4, eps float64) bool {
	for i := range a {
		if math.Abs(a[i]-b[i]) > eps {
			return false
		}
	}
	return true
}

func TestVec3Ops(t *testing.T) {
	a := Vec3{1, 2, 3}
	b := Vec3{4, 5, 6}
	if got := a.Add(b); got != (Vec3{5, 7, 9}) {
		t.Errorf("Add = %v", got)
	}
	if got := b.Sub(a); got != (Vec3{3, 3, 3}) {
		t.Errorf("Sub = %v", got)
	}
	if got := a.Scale(2); got != (Vec3{2, 4, 6}) {
		t.Errorf("Scale = %v", got)
	}
	if got := a.Dot(b); got != 32 {
		t.Errorf("Dot = %v, want 32", got)
	}
	if got := (Vec3{1, 0, 0}).Cross(Vec3{0, 1, 0}); got != (Vec3{0, 0, 1}) {
		t.Errorf("Cross = %v, want {0,0,1}", got)
	}
	if got := (Vec3{3, 4, 0}).Length(); got != 5 {
		t.Errorf("Length = %v, want 5", got)
	}
	if got := (Vec3{0, 5, 0}).Normalize(); !vec3Close(got, Vec3{0, 1, 0}, 1e-12) {
		t.Errorf("Normalize = %v", got)
	}
	if got := (Vec3{0, 0, 0}).Normalize(); got != (Vec3{0, 0, 0}) {
		t.Errorf("Normalize zero = %v", got)
	}
	if got := a.Lerp(b, 0.5); got != (Vec3{2.5, 3.5, 4.5}) {
		t.Errorf("Lerp = %v", got)
	}
}

func TestQuatMulIdentity(t *testing.T) {
	q := QuatFromAxisAngle(Vec3{0, 1, 0}, 0.7)
	if got := IdentityQuat.Mul(q); !quatClose(got, q, 1e-12) {
		t.Errorf("identity*q = %v, want %v", got, q)
	}
	if got := q.Mul(IdentityQuat); !quatClose(got, q, 1e-12) {
		t.Errorf("q*identity = %v, want %v", got, q)
	}
}

func TestQuatMulCompose(t *testing.T) {
	// Two 90-degree rotations about Z compose to a 180-degree rotation.
	z90 := QuatFromAxisAngle(Vec3{0, 0, 1}, math.Pi/2)
	z180 := QuatFromAxisAngle(Vec3{0, 0, 1}, math.Pi)
	if got := z90.Mul(z90); !quatClose(got, z180, 1e-12) {
		t.Errorf("z90*z90 = %v, want %v", got, z180)
	}
}

func TestQuatConjugateDot(t *testing.T) {
	q := QuatFromAxisAngle(Vec3{1, 0, 0}, 0.9)
	// q * conj(q) == identity for a unit quaternion.
	if got := q.Mul(q.Conjugate()); !quatClose(got, IdentityQuat, 1e-12) {
		t.Errorf("q*conj = %v, want identity", got)
	}
	if got := q.Dot(q); math.Abs(got-1) > 1e-12 {
		t.Errorf("unit quat dot self = %v, want 1", got)
	}
}

func TestQuatRotate(t *testing.T) {
	z90 := QuatFromAxisAngle(Vec3{0, 0, 1}, math.Pi/2)
	if got := z90.Rotate(Vec3{1, 0, 0}); !vec3Close(got, Vec3{0, 1, 0}, 1e-12) {
		t.Errorf("rotate X by Z90 = %v, want {0,1,0}", got)
	}
	x90 := QuatFromAxisAngle(Vec3{1, 0, 0}, math.Pi/2)
	if got := x90.Rotate(Vec3{0, 1, 0}); !vec3Close(got, Vec3{0, 0, 1}, 1e-12) {
		t.Errorf("rotate Y by X90 = %v, want {0,0,1}", got)
	}
}

func TestQuatFromAxisAngleZeroAxis(t *testing.T) {
	if got := QuatFromAxisAngle(Vec3{0, 0, 0}, 1.5); got != IdentityQuat {
		t.Errorf("zero axis = %v, want identity", got)
	}
}

func TestQuatFromEuler(t *testing.T) {
	// A single-axis Euler rotation matches the axis-angle quaternion.
	got := QuatFromEuler(0, 0, math.Pi/3)
	want := QuatFromAxisAngle(Vec3{0, 0, 1}, math.Pi/3)
	if !quatClose(got, want, 1e-12) {
		t.Errorf("euler Z = %v, want %v", got, want)
	}
	// Consistency with quaternion Matrix: rotating a vector two ways agrees.
	q := QuatFromEuler(0.3, -0.5, 1.1)
	p := Vec3{1, 2, 3}
	viaRotate := q.Rotate(p)
	viaMatrix := q.Matrix().TransformDir(p)
	if !vec3Close(viaRotate, viaMatrix, 1e-9) {
		t.Errorf("Rotate=%v Matrix=%v disagree", viaRotate, viaMatrix)
	}
}

func TestMat4Transpose(t *testing.T) {
	m := Mat4{
		0, 1, 2, 3,
		4, 5, 6, 7,
		8, 9, 10, 11,
		12, 13, 14, 15,
	}
	tr := m.Transpose()
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			if m.At(r, c) != tr.At(c, r) {
				t.Fatalf("transpose mismatch at %d,%d", r, c)
			}
		}
	}
	if got := m.Transpose().Transpose(); got != m {
		t.Errorf("double transpose != original")
	}
}

func TestMat4TransformDir(t *testing.T) {
	// A pure translation must not affect a direction vector.
	m := TRS(Vec3{10, 20, 30}, IdentityQuat, Vec3{1, 1, 1})
	if got := m.TransformDir(Vec3{1, 0, 0}); !vec3Close(got, Vec3{1, 0, 0}, 1e-12) {
		t.Errorf("TransformDir under translation = %v, want {1,0,0}", got)
	}
}

func TestLookAt(t *testing.T) {
	m := LookAt(Vec3{0, 0, 5}, Vec3{0, 0, 0}, Vec3{0, 1, 0})
	// The look-at target maps to a point along -Z at distance 5.
	if got := m.TransformPoint(Vec3{0, 0, 0}); !vec3Close(got, Vec3{0, 0, -5}, 1e-12) {
		t.Errorf("origin in view space = %v, want {0,0,-5}", got)
	}
	// The eye maps to the view-space origin.
	if got := m.TransformPoint(Vec3{0, 0, 5}); !vec3Close(got, Vec3{0, 0, 0}, 1e-12) {
		t.Errorf("eye in view space = %v, want {0,0,0}", got)
	}
}

func TestPerspective(t *testing.T) {
	m := Perspective(math.Pi/2, 1, 1, 3)
	want := Mat4{}
	want[0] = 1
	want[5] = 1
	want[10] = -2
	want[11] = -1
	want[14] = -3
	if !mat4Close(m, want, 1e-12) {
		t.Errorf("Perspective = %v, want %v", m, want)
	}
	// Infinite far.
	mi := Perspective(math.Pi/2, 1, 1, 0)
	if math.Abs(mi[10]-(-1)) > 1e-12 || math.Abs(mi[14]-(-2)) > 1e-12 {
		t.Errorf("infinite Perspective m[10]=%v m[14]=%v", mi[10], mi[14])
	}
}

func TestOrthographic(t *testing.T) {
	m := Orthographic(2, 4, 1, 5)
	want := Mat4{}
	want[0] = 0.5
	want[5] = 0.25
	want[10] = -0.5
	want[14] = -1.5
	want[15] = 1
	if !mat4Close(m, want, 1e-12) {
		t.Errorf("Orthographic = %v, want %v", m, want)
	}
}

func BenchmarkQuatRotate(b *testing.B) {
	q := QuatFromAxisAngle(Vec3{0.3, 0.5, 0.8}, 1.1)
	v := Vec3{1, 2, 3}
	var sink Vec3
	for i := 0; i < b.N; i++ {
		sink = q.Rotate(v)
	}
	_ = sink
}

func BenchmarkMat4Transpose(b *testing.B) {
	m := TRS(Vec3{1, 2, 3}, QuatFromAxisAngle(Vec3{0, 1, 0}, 0.5), Vec3{2, 2, 2})
	var sink Mat4
	for i := 0; i < b.N; i++ {
		sink = m.Transpose()
	}
	_ = sink
}
