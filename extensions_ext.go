package gltf

import "encoding/json"

// Extension name constants for the additional Khronos PBR material extensions
// modeled below.
const (
	// ExtMaterialsClearcoat is the name of the KHR_materials_clearcoat extension.
	ExtMaterialsClearcoat = "KHR_materials_clearcoat"
	// ExtMaterialsSheen is the name of the KHR_materials_sheen extension.
	ExtMaterialsSheen = "KHR_materials_sheen"
	// ExtMaterialsSpecular is the name of the KHR_materials_specular extension.
	ExtMaterialsSpecular = "KHR_materials_specular"
	// ExtMaterialsVolume is the name of the KHR_materials_volume extension.
	ExtMaterialsVolume = "KHR_materials_volume"
)

// MaterialsClearcoat is the KHR_materials_clearcoat extension, adding a clear
// protective coating layer (as on car paint or lacquered wood) over the base
// material.
type MaterialsClearcoat struct {
	// ClearcoatFactor is the clearcoat layer intensity. The glTF default is 0.
	ClearcoatFactor float64 `json:"clearcoatFactor,omitempty"`
	// ClearcoatTexture modulates the clearcoat factor (red channel).
	ClearcoatTexture *TextureInfo `json:"clearcoatTexture,omitempty"`
	// ClearcoatRoughnessFactor is the clearcoat layer roughness. Default 0.
	ClearcoatRoughnessFactor float64 `json:"clearcoatRoughnessFactor,omitempty"`
	// ClearcoatRoughnessTexture modulates the clearcoat roughness (green channel).
	ClearcoatRoughnessTexture *TextureInfo `json:"clearcoatRoughnessTexture,omitempty"`
	// ClearcoatNormalTexture is a normal map applied to the clearcoat layer.
	ClearcoatNormalTexture *NormalTexture  `json:"clearcoatNormalTexture,omitempty"`
	Extras                 json.RawMessage `json:"extras,omitempty"`
}

// MaterialsSheen is the KHR_materials_sheen extension, adding a retro-reflective
// sheen used to model cloth and fabric.
type MaterialsSheen struct {
	// SheenColorFactor is the sheen RGB color. The glTF default is [0,0,0].
	SheenColorFactor *[3]float64 `json:"sheenColorFactor,omitempty"`
	// SheenColorTexture modulates the sheen color (RGB channels).
	SheenColorTexture *TextureInfo `json:"sheenColorTexture,omitempty"`
	// SheenRoughnessFactor is the sheen roughness. The glTF default is 0.
	SheenRoughnessFactor float64 `json:"sheenRoughnessFactor,omitempty"`
	// SheenRoughnessTexture modulates the sheen roughness (alpha channel).
	SheenRoughnessTexture *TextureInfo    `json:"sheenRoughnessTexture,omitempty"`
	Extras                json.RawMessage `json:"extras,omitempty"`
}

// MaterialsSpecular is the KHR_materials_specular extension, allowing the
// specular reflection strength and color of a metallic-roughness material to be
// tuned independently.
type MaterialsSpecular struct {
	// SpecularFactor scales the specular reflection strength. Default 1.
	SpecularFactor *float64 `json:"specularFactor,omitempty"`
	// SpecularTexture modulates the specular factor (alpha channel).
	SpecularTexture *TextureInfo `json:"specularTexture,omitempty"`
	// SpecularColorFactor is the specular RGB tint. The glTF default is [1,1,1].
	SpecularColorFactor *[3]float64 `json:"specularColorFactor,omitempty"`
	// SpecularColorTexture modulates the specular color (RGB channels).
	SpecularColorTexture *TextureInfo    `json:"specularColorTexture,omitempty"`
	Extras               json.RawMessage `json:"extras,omitempty"`
}

// MaterialsVolume is the KHR_materials_volume extension, giving a transmissive
// surface a physical thickness so light is attenuated as it passes through the
// enclosed volume.
type MaterialsVolume struct {
	// ThicknessFactor is the thickness of the volume in local space. Default 0
	// (a thin-walled surface with no volume).
	ThicknessFactor float64 `json:"thicknessFactor,omitempty"`
	// ThicknessTexture modulates the thickness (green channel).
	ThicknessTexture *TextureInfo `json:"thicknessTexture,omitempty"`
	// AttenuationDistance is the average distance light travels before being
	// attenuated. Nil means infinity (no attenuation).
	AttenuationDistance *float64 `json:"attenuationDistance,omitempty"`
	// AttenuationColor is the color that white light turns into by the
	// attenuation distance. The glTF default is [1,1,1].
	AttenuationColor *[3]float64     `json:"attenuationColor,omitempty"`
	Extras           json.RawMessage `json:"extras,omitempty"`
}

// Clearcoat decodes the KHR_materials_clearcoat extension from the material,
// reporting whether it is present.
func (m *Material) Clearcoat() (MaterialsClearcoat, bool) {
	var c MaterialsClearcoat
	found, err := GetExtension(m.Extensions, ExtMaterialsClearcoat, &c)
	if err != nil || !found {
		return MaterialsClearcoat{}, false
	}
	return c, true
}

// Sheen decodes the KHR_materials_sheen extension from the material, reporting
// whether it is present.
func (m *Material) Sheen() (MaterialsSheen, bool) {
	var s MaterialsSheen
	found, err := GetExtension(m.Extensions, ExtMaterialsSheen, &s)
	if err != nil || !found {
		return MaterialsSheen{}, false
	}
	return s, true
}

// Specular decodes the KHR_materials_specular extension from the material,
// reporting whether it is present.
func (m *Material) Specular() (MaterialsSpecular, bool) {
	var s MaterialsSpecular
	found, err := GetExtension(m.Extensions, ExtMaterialsSpecular, &s)
	if err != nil || !found {
		return MaterialsSpecular{}, false
	}
	return s, true
}

// Volume decodes the KHR_materials_volume extension from the material, reporting
// whether it is present.
func (m *Material) Volume() (MaterialsVolume, bool) {
	var v MaterialsVolume
	found, err := GetExtension(m.Extensions, ExtMaterialsVolume, &v)
	if err != nil || !found {
		return MaterialsVolume{}, false
	}
	return v, true
}
