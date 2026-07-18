package gltf

import (
	"math"
	"testing"
)

func TestCameraPerspectiveProjection(t *testing.T) {
	far := 3.0
	aspect := 1.0
	c := &CameraPerspective{YFOV: math.Pi / 2, ZNear: 1, ZFar: &far, AspectRatio: &aspect}
	got := c.ProjectionMatrix()
	want := Perspective(math.Pi/2, 1, 1, 3)
	if !mat4Close(got, want, 1e-12) {
		t.Errorf("perspective projection = %v, want %v", got, want)
	}
}

func TestCameraPerspectiveInfinite(t *testing.T) {
	c := &CameraPerspective{YFOV: math.Pi / 2, ZNear: 2} // ZFar nil, AspectRatio nil
	got := c.ProjectionMatrix()
	if math.Abs(got[10]-(-1)) > 1e-12 || math.Abs(got[14]-(-4)) > 1e-12 {
		t.Errorf("infinite projection m[10]=%v m[14]=%v, want -1 and -4", got[10], got[14])
	}
	// Aspect defaults to 1, so m[0]==m[5].
	if math.Abs(got[0]-got[5]) > 1e-12 {
		t.Errorf("default aspect not 1: m[0]=%v m[5]=%v", got[0], got[5])
	}
}

func TestCameraOrthographicProjection(t *testing.T) {
	c := &CameraOrthographic{XMag: 2, YMag: 4, ZNear: 1, ZFar: 5}
	got := c.ProjectionMatrix()
	want := Orthographic(2, 4, 1, 5)
	if !mat4Close(got, want, 1e-12) {
		t.Errorf("orthographic projection = %v, want %v", got, want)
	}
}

func TestCameraProjectionDispatch(t *testing.T) {
	far := 100.0
	pc := &Camera{Type: CameraTypePerspective, Perspective: &CameraPerspective{YFOV: 1, ZNear: 0.1, ZFar: &far}}
	if _, err := pc.ProjectionMatrix(); err != nil {
		t.Errorf("perspective dispatch: %v", err)
	}
	oc := &Camera{Type: CameraTypeOrthographic, Orthographic: &CameraOrthographic{XMag: 1, YMag: 1, ZNear: 0, ZFar: 1}}
	if _, err := oc.ProjectionMatrix(); err != nil {
		t.Errorf("orthographic dispatch: %v", err)
	}
	// Missing block yields an error.
	bad := &Camera{Type: CameraTypePerspective}
	if _, err := bad.ProjectionMatrix(); err == nil {
		t.Errorf("expected error for perspective camera with no block")
	}
	unknown := &Camera{Type: "weird"}
	if _, err := unknown.ProjectionMatrix(); err == nil {
		t.Errorf("expected error for unknown camera type")
	}
}
