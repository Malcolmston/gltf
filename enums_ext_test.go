package gltf

import "testing"

func TestPrimitiveModeString(t *testing.T) {
	cases := map[PrimitiveMode]string{
		PrimitivePoints:        "POINTS",
		PrimitiveLines:         "LINES",
		PrimitiveLineLoop:      "LINE_LOOP",
		PrimitiveLineStrip:     "LINE_STRIP",
		PrimitiveTriangles:     "TRIANGLES",
		PrimitiveTriangleStrip: "TRIANGLE_STRIP",
		PrimitiveTriangleFan:   "TRIANGLE_FAN",
		PrimitiveMode(99):      "PrimitiveMode(99)",
	}
	for m, want := range cases {
		if got := m.String(); got != want {
			t.Errorf("PrimitiveMode(%d).String() = %q, want %q", int(m), got, want)
		}
	}
}

func TestFilterString(t *testing.T) {
	cases := map[Filter]string{
		FilterNearest:              "NEAREST",
		FilterLinear:               "LINEAR",
		FilterNearestMipmapNearest: "NEAREST_MIPMAP_NEAREST",
		FilterLinearMipmapNearest:  "LINEAR_MIPMAP_NEAREST",
		FilterNearestMipmapLinear:  "NEAREST_MIPMAP_LINEAR",
		FilterLinearMipmapLinear:   "LINEAR_MIPMAP_LINEAR",
		Filter(1):                  "Filter(1)",
	}
	for f, want := range cases {
		if got := f.String(); got != want {
			t.Errorf("Filter(%d).String() = %q, want %q", int(f), got, want)
		}
	}
}

func TestWrapModeString(t *testing.T) {
	cases := map[WrapMode]string{
		WrapClampToEdge:    "CLAMP_TO_EDGE",
		WrapMirroredRepeat: "MIRRORED_REPEAT",
		WrapRepeat:         "REPEAT",
		WrapMode(2):        "WrapMode(2)",
	}
	for w, want := range cases {
		if got := w.String(); got != want {
			t.Errorf("WrapMode(%d).String() = %q, want %q", int(w), got, want)
		}
	}
}

func TestTargetTypeString(t *testing.T) {
	cases := map[TargetType]string{
		TargetArrayBuffer:        "ARRAY_BUFFER",
		TargetElementArrayBuffer: "ELEMENT_ARRAY_BUFFER",
		TargetType(7):            "TargetType(7)",
	}
	for tt, want := range cases {
		if got := tt.String(); got != want {
			t.Errorf("TargetType(%d).String() = %q, want %q", int(tt), got, want)
		}
	}
}
