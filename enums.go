package gltf

// ComponentType identifies the datatype of the components that make up an
// accessor element. The integer values match the glTF 2.0 specification (they
// are OpenGL constants).
type ComponentType int

// Component types defined by the glTF 2.0 specification.
const (
	ComponentByte          ComponentType = 5120 // signed 8-bit integer
	ComponentUnsignedByte  ComponentType = 5121 // unsigned 8-bit integer
	ComponentShort         ComponentType = 5122 // signed 16-bit integer
	ComponentUnsignedShort ComponentType = 5123 // unsigned 16-bit integer
	ComponentUnsignedInt   ComponentType = 5125 // unsigned 32-bit integer
	ComponentFloat         ComponentType = 5126 // 32-bit IEEE-754 float
)

// SizeInBytes returns the size, in bytes, of a single component of this type.
// It returns 0 for an unknown component type.
func (c ComponentType) SizeInBytes() int {
	switch c {
	case ComponentByte, ComponentUnsignedByte:
		return 1
	case ComponentShort, ComponentUnsignedShort:
		return 2
	case ComponentUnsignedInt, ComponentFloat:
		return 4
	default:
		return 0
	}
}

// String returns a human-readable name for the component type.
func (c ComponentType) String() string {
	switch c {
	case ComponentByte:
		return "BYTE"
	case ComponentUnsignedByte:
		return "UNSIGNED_BYTE"
	case ComponentShort:
		return "SHORT"
	case ComponentUnsignedShort:
		return "UNSIGNED_SHORT"
	case ComponentUnsignedInt:
		return "UNSIGNED_INT"
	case ComponentFloat:
		return "FLOAT"
	default:
		return "UNKNOWN"
	}
}

// AccessorType specifies whether an accessor element is a scalar, vector, or
// matrix. It is encoded as a JSON string in glTF.
type AccessorType string

// Accessor element types defined by the glTF 2.0 specification.
const (
	AccessorScalar AccessorType = "SCALAR"
	AccessorVec2   AccessorType = "VEC2"
	AccessorVec3   AccessorType = "VEC3"
	AccessorVec4   AccessorType = "VEC4"
	AccessorMat2   AccessorType = "MAT2"
	AccessorMat3   AccessorType = "MAT3"
	AccessorMat4   AccessorType = "MAT4"
)

// ComponentCount returns the number of components in a single element of this
// accessor type. It returns 0 for an unknown type.
func (t AccessorType) ComponentCount() int {
	switch t {
	case AccessorScalar:
		return 1
	case AccessorVec2:
		return 2
	case AccessorVec3:
		return 3
	case AccessorVec4, AccessorMat2:
		return 4
	case AccessorMat3:
		return 9
	case AccessorMat4:
		return 16
	default:
		return 0
	}
}

// PrimitiveMode specifies the type of primitives to render. The integer values
// match the glTF 2.0 specification (OpenGL primitive modes).
type PrimitiveMode int

// Primitive rendering modes defined by the glTF 2.0 specification.
const (
	PrimitivePoints        PrimitiveMode = 0
	PrimitiveLines         PrimitiveMode = 1
	PrimitiveLineLoop      PrimitiveMode = 2
	PrimitiveLineStrip     PrimitiveMode = 3
	PrimitiveTriangles     PrimitiveMode = 4
	PrimitiveTriangleStrip PrimitiveMode = 5
	PrimitiveTriangleFan   PrimitiveMode = 6
)

// TargetType is the OpenGL buffer binding target hint carried by a bufferView.
type TargetType int

// BufferView target hints defined by the glTF 2.0 specification.
const (
	TargetArrayBuffer        TargetType = 34962 // vertex attributes
	TargetElementArrayBuffer TargetType = 34963 // vertex indices
)

// Filter identifies a texture magnification or minification filter.
type Filter int

// Texture filters defined by the glTF 2.0 specification.
const (
	FilterNearest              Filter = 9728
	FilterLinear               Filter = 9729
	FilterNearestMipmapNearest Filter = 9984
	FilterLinearMipmapNearest  Filter = 9985
	FilterNearestMipmapLinear  Filter = 9986
	FilterLinearMipmapLinear   Filter = 9987
)

// WrapMode identifies a texture wrapping mode for the S or T axis.
type WrapMode int

// Texture wrapping modes defined by the glTF 2.0 specification.
const (
	WrapClampToEdge    WrapMode = 33071
	WrapMirroredRepeat WrapMode = 33648
	WrapRepeat         WrapMode = 10497
)

// Interpolation is the interpolation algorithm used by an animation sampler.
type Interpolation string

// Animation interpolation algorithms defined by the glTF 2.0 specification.
const (
	InterpolationLinear      Interpolation = "LINEAR"
	InterpolationStep        Interpolation = "STEP"
	InterpolationCubicSpline Interpolation = "CUBICSPLINE"
)

// AnimationPath is the node property targeted by an animation channel.
type AnimationPath string

// Animation target paths defined by the glTF 2.0 specification.
const (
	PathTranslation AnimationPath = "translation"
	PathRotation    AnimationPath = "rotation"
	PathScale       AnimationPath = "scale"
	PathWeights     AnimationPath = "weights"
)

// CameraType distinguishes perspective and orthographic cameras.
type CameraType string

// Camera types defined by the glTF 2.0 specification.
const (
	CameraTypePerspective  CameraType = "perspective"
	CameraTypeOrthographic CameraType = "orthographic"
)

// AlphaMode controls how a material's alpha value is interpreted.
type AlphaMode string

// Material alpha modes defined by the glTF 2.0 specification.
const (
	AlphaOpaque AlphaMode = "OPAQUE"
	AlphaMask   AlphaMode = "MASK"
	AlphaBlend  AlphaMode = "BLEND"
)
