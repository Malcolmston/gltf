package gltf

import (
	"encoding/json"
	"math"
)

// Extension name constants for the Khronos (KHR_) extensions supported by this
// package.
const (
	// ExtMaterialsUnlit is the name of the KHR_materials_unlit extension.
	ExtMaterialsUnlit = "KHR_materials_unlit"
	// ExtMaterialsEmissiveStrength is the name of the
	// KHR_materials_emissive_strength extension.
	ExtMaterialsEmissiveStrength = "KHR_materials_emissive_strength"
	// ExtMaterialsTransmission is the name of the KHR_materials_transmission
	// extension.
	ExtMaterialsTransmission = "KHR_materials_transmission"
	// ExtMaterialsIOR is the name of the KHR_materials_ior extension.
	ExtMaterialsIOR = "KHR_materials_ior"
	// ExtTextureTransform is the name of the KHR_texture_transform extension.
	ExtTextureTransform = "KHR_texture_transform"
	// ExtLightsPunctual is the name of the KHR_lights_punctual extension.
	ExtLightsPunctual = "KHR_lights_punctual"
	// ExtMaterialsPBRSpecularGlossiness is the name of the
	// KHR_materials_pbrSpecularGlossiness extension.
	ExtMaterialsPBRSpecularGlossiness = "KHR_materials_pbrSpecularGlossiness"
)

// ExtensionMap decodes a raw extensions object into a map from extension name
// to its raw JSON value. It returns an empty (non-nil) map when raw is empty.
// The map preserves every extension verbatim, including ones this package does
// not model, so it can be round-tripped with [MarshalExtensions].
func ExtensionMap(raw json.RawMessage) (map[string]json.RawMessage, error) {
	m := map[string]json.RawMessage{}
	if len(raw) == 0 {
		return m, nil
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// MarshalExtensions encodes an extension map back into a raw extensions object.
// It returns nil (which serializes as an omitted field) when the map is empty.
func MarshalExtensions(m map[string]json.RawMessage) (json.RawMessage, error) {
	if len(m) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// GetExtension decodes the named extension from raw into v, reporting whether
// the extension was present. v must be a non-nil pointer.
func GetExtension(raw json.RawMessage, name string, v any) (bool, error) {
	m, err := ExtensionMap(raw)
	if err != nil {
		return false, err
	}
	ext, ok := m[name]
	if !ok {
		return false, nil
	}
	if v == nil {
		return true, nil
	}
	if err := json.Unmarshal(ext, v); err != nil {
		return true, err
	}
	return true, nil
}

// SetExtension returns a new raw extensions object with the named extension set
// to the JSON encoding of v, preserving every other extension already present
// in raw. Passing a nil v removes the named extension.
func SetExtension(raw json.RawMessage, name string, v any) (json.RawMessage, error) {
	m, err := ExtensionMap(raw)
	if err != nil {
		return nil, err
	}
	if v == nil {
		delete(m, name)
		return MarshalExtensions(m)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	m[name] = json.RawMessage(b)
	return MarshalExtensions(m)
}

// MaterialsUnlit is the KHR_materials_unlit extension. It carries no data; its
// presence marks a material as unlit (rendered with its base color only).
type MaterialsUnlit struct {
	Extras json.RawMessage `json:"extras,omitempty"`
}

// MaterialsEmissiveStrength is the KHR_materials_emissive_strength extension,
// scaling a material's emissive factor beyond the usual [0,1] range.
type MaterialsEmissiveStrength struct {
	// EmissiveStrength multiplies the material's emissive factor. The glTF
	// default is 1.0.
	EmissiveStrength float64         `json:"emissiveStrength"`
	Extras           json.RawMessage `json:"extras,omitempty"`
}

// MaterialsTransmission is the KHR_materials_transmission extension, describing
// optically transparent surfaces such as glass.
type MaterialsTransmission struct {
	// TransmissionFactor is the base percentage of light transmitted through
	// the surface. The glTF default is 0.0.
	TransmissionFactor float64 `json:"transmissionFactor,omitempty"`
	// TransmissionTexture modulates the transmission factor (red channel).
	TransmissionTexture *TextureInfo    `json:"transmissionTexture,omitempty"`
	Extras              json.RawMessage `json:"extras,omitempty"`
}

// MaterialsIOR is the KHR_materials_ior extension, overriding a material's
// index of refraction.
type MaterialsIOR struct {
	// IOR is the index of refraction. The glTF default is 1.5.
	IOR    float64         `json:"ior"`
	Extras json.RawMessage `json:"extras,omitempty"`
}

// TextureTransform is the KHR_texture_transform extension, applying an offset,
// rotation, and scale to a texture's UV coordinates. It appears inside a
// [TextureInfo]'s extensions.
type TextureTransform struct {
	// Offset is the UV translation applied after scaling. Default [0,0].
	Offset *[2]float64 `json:"offset,omitempty"`
	// Rotation is the counter-clockwise rotation in radians. Default 0.
	Rotation float64 `json:"rotation,omitempty"`
	// Scale is the UV scale. Default [1,1].
	Scale *[2]float64 `json:"scale,omitempty"`
	// TexCoord overrides the texture's UV set index when non-nil.
	TexCoord *int            `json:"texCoord,omitempty"`
	Extras   json.RawMessage `json:"extras,omitempty"`
}

// UVMatrix returns the 3x3 UV transform (column-major, with the last column
// carrying the translation) implied by the texture transform, applying scale,
// then rotation, then offset. Absent fields use their glTF defaults.
func (t *TextureTransform) UVMatrix() [9]float64 {
	sx, sy := 1.0, 1.0
	if t.Scale != nil {
		sx, sy = t.Scale[0], t.Scale[1]
	}
	ox, oy := 0.0, 0.0
	if t.Offset != nil {
		ox, oy = t.Offset[0], t.Offset[1]
	}
	c, s := math.Cos(t.Rotation), math.Sin(t.Rotation)
	// Column-major 3x3: T * R * S applied to (u, v, 1).
	return [9]float64{
		c * sx, -s * sx, 0,
		s * sy, c * sy, 0,
		ox, oy, 1,
	}
}

// LightsPunctual is the KHR_lights_punctual document-level extension, holding
// the array of lights that nodes reference by index.
type LightsPunctual struct {
	Lights []Light         `json:"lights"`
	Extras json.RawMessage `json:"extras,omitempty"`
}

// NodeLight is the KHR_lights_punctual node-level extension, referencing a
// light in the document-level [LightsPunctual] array.
type NodeLight struct {
	Light  Index           `json:"light"`
	Extras json.RawMessage `json:"extras,omitempty"`
}

// LightType is the kind of a punctual light.
type LightType string

// Punctual light types defined by KHR_lights_punctual.
const (
	LightTypeDirectional LightType = "directional"
	LightTypePoint       LightType = "point"
	LightTypeSpot        LightType = "spot"
)

// Light is a punctual light source defined by KHR_lights_punctual.
type Light struct {
	Name string `json:"name,omitempty"`
	// Color is the linear RGB light color. Default [1,1,1].
	Color *[3]float64 `json:"color,omitempty"`
	// Intensity is in candela (point/spot) or lux (directional). Default 1.
	Intensity *float64  `json:"intensity,omitempty"`
	Type      LightType `json:"type"`
	// Range is the maximum distance the light affects (point/spot). Nil means
	// infinite.
	Range *float64 `json:"range,omitempty"`
	// Spot holds cone parameters when Type is "spot".
	Spot       *LightSpot      `json:"spot,omitempty"`
	Extensions json.RawMessage `json:"extensions,omitempty"`
	Extras     json.RawMessage `json:"extras,omitempty"`
}

// LightSpot holds the cone angles of a spot light.
type LightSpot struct {
	// InnerConeAngle is the angle (radians) at which falloff begins. Default 0.
	InnerConeAngle float64 `json:"innerConeAngle,omitempty"`
	// OuterConeAngle is the angle (radians) at which falloff ends. Default π/4.
	OuterConeAngle *float64        `json:"outerConeAngle,omitempty"`
	Extras         json.RawMessage `json:"extras,omitempty"`
}

// MaterialsPBRSpecularGlossiness is the (archived) KHR_materials_pbrSpecularGlossiness
// extension, providing the specular-glossiness material model as an alternative
// to metallic-roughness.
type MaterialsPBRSpecularGlossiness struct {
	// DiffuseFactor is the reflected diffuse RGBA factor. Default [1,1,1,1].
	DiffuseFactor *[4]float64 `json:"diffuseFactor,omitempty"`
	// DiffuseTexture is the diffuse texture.
	DiffuseTexture *TextureInfo `json:"diffuseTexture,omitempty"`
	// SpecularFactor is the specular RGB factor. Default [1,1,1].
	SpecularFactor *[3]float64 `json:"specularFactor,omitempty"`
	// GlossinessFactor is the glossiness (smoothness). Default 1.
	GlossinessFactor *float64 `json:"glossinessFactor,omitempty"`
	// SpecularGlossinessTexture holds specular (RGB) and glossiness (A).
	SpecularGlossinessTexture *TextureInfo    `json:"specularGlossinessTexture,omitempty"`
	Extras                    json.RawMessage `json:"extras,omitempty"`
}

// Unlit reports whether the material carries the KHR_materials_unlit extension.
func (m *Material) Unlit() bool {
	found, _ := GetExtension(m.Extensions, ExtMaterialsUnlit, nil)
	return found
}

// EmissiveStrength returns the KHR_materials_emissive_strength value and whether
// the extension is present. When absent it returns the default 1.0.
func (m *Material) EmissiveStrength() (float64, bool) {
	var e MaterialsEmissiveStrength
	e.EmissiveStrength = 1.0
	found, err := GetExtension(m.Extensions, ExtMaterialsEmissiveStrength, &e)
	if err != nil || !found {
		return 1.0, false
	}
	return e.EmissiveStrength, true
}

// IOR returns the KHR_materials_ior index of refraction and whether the
// extension is present. When absent it returns the default 1.5.
func (m *Material) IOR() (float64, bool) {
	e := MaterialsIOR{IOR: 1.5}
	found, err := GetExtension(m.Extensions, ExtMaterialsIOR, &e)
	if err != nil || !found {
		return 1.5, false
	}
	return e.IOR, true
}

// Transmission decodes the KHR_materials_transmission extension, reporting
// whether it is present.
func (m *Material) Transmission() (MaterialsTransmission, bool) {
	var t MaterialsTransmission
	found, err := GetExtension(m.Extensions, ExtMaterialsTransmission, &t)
	if err != nil || !found {
		return MaterialsTransmission{}, false
	}
	return t, true
}

// SpecularGlossiness decodes the KHR_materials_pbrSpecularGlossiness extension,
// reporting whether it is present.
func (m *Material) SpecularGlossiness() (MaterialsPBRSpecularGlossiness, bool) {
	var sg MaterialsPBRSpecularGlossiness
	found, err := GetExtension(m.Extensions, ExtMaterialsPBRSpecularGlossiness, &sg)
	if err != nil || !found {
		return MaterialsPBRSpecularGlossiness{}, false
	}
	return sg, true
}

// SetExtension sets or replaces a named extension on the material, preserving
// any other extensions already present.
func (m *Material) SetExtension(name string, v any) error {
	raw, err := SetExtension(m.Extensions, name, v)
	if err != nil {
		return err
	}
	m.Extensions = raw
	return nil
}

// TextureTransform decodes the KHR_texture_transform extension from a texture
// reference, reporting whether it is present.
func (ti *TextureInfo) TextureTransform() (TextureTransform, bool) {
	var t TextureTransform
	found, err := GetExtension(ti.Extensions, ExtTextureTransform, &t)
	if err != nil || !found {
		return TextureTransform{}, false
	}
	return t, true
}

// Lights decodes the document-level KHR_lights_punctual extension, returning
// its light array and whether the extension is present.
func (d *Document) Lights() ([]Light, bool) {
	var lp LightsPunctual
	found, err := GetExtension(d.Extensions, ExtLightsPunctual, &lp)
	if err != nil || !found {
		return nil, false
	}
	return lp.Lights, true
}

// NodeLight decodes the node-level KHR_lights_punctual reference, returning the
// referenced light index and whether the extension is present.
func (n *Node) NodeLight() (int, bool) {
	var nl NodeLight
	found, err := GetExtension(n.Extensions, ExtLightsPunctual, &nl)
	if err != nil || !found {
		return 0, false
	}
	return nl.Light, true
}
