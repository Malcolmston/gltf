package gltf

import "strconv"

// String returns the glTF specification name of the primitive mode (for example
// "TRIANGLES"), or a numeric fallback for an unknown value.
func (m PrimitiveMode) String() string {
	switch m {
	case PrimitivePoints:
		return "POINTS"
	case PrimitiveLines:
		return "LINES"
	case PrimitiveLineLoop:
		return "LINE_LOOP"
	case PrimitiveLineStrip:
		return "LINE_STRIP"
	case PrimitiveTriangles:
		return "TRIANGLES"
	case PrimitiveTriangleStrip:
		return "TRIANGLE_STRIP"
	case PrimitiveTriangleFan:
		return "TRIANGLE_FAN"
	default:
		return "PrimitiveMode(" + strconv.Itoa(int(m)) + ")"
	}
}

// String returns the glTF specification name of the texture filter (for example
// "LINEAR_MIPMAP_LINEAR"), or a numeric fallback for an unknown value.
func (f Filter) String() string {
	switch f {
	case FilterNearest:
		return "NEAREST"
	case FilterLinear:
		return "LINEAR"
	case FilterNearestMipmapNearest:
		return "NEAREST_MIPMAP_NEAREST"
	case FilterLinearMipmapNearest:
		return "LINEAR_MIPMAP_NEAREST"
	case FilterNearestMipmapLinear:
		return "NEAREST_MIPMAP_LINEAR"
	case FilterLinearMipmapLinear:
		return "LINEAR_MIPMAP_LINEAR"
	default:
		return "Filter(" + strconv.Itoa(int(f)) + ")"
	}
}

// String returns the glTF specification name of the wrap mode (for example
// "CLAMP_TO_EDGE"), or a numeric fallback for an unknown value.
func (w WrapMode) String() string {
	switch w {
	case WrapClampToEdge:
		return "CLAMP_TO_EDGE"
	case WrapMirroredRepeat:
		return "MIRRORED_REPEAT"
	case WrapRepeat:
		return "REPEAT"
	default:
		return "WrapMode(" + strconv.Itoa(int(w)) + ")"
	}
}

// String returns the glTF specification name of the bufferView target (for
// example "ARRAY_BUFFER"), or a numeric fallback for an unknown value.
func (t TargetType) String() string {
	switch t {
	case TargetArrayBuffer:
		return "ARRAY_BUFFER"
	case TargetElementArrayBuffer:
		return "ELEMENT_ARRAY_BUFFER"
	default:
		return "TargetType(" + strconv.Itoa(int(t)) + ")"
	}
}
