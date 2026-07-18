package gltf

import "testing"

func TestClearcoatRoundTrip(t *testing.T) {
	m := &Material{}
	set := MaterialsClearcoat{ClearcoatFactor: 0.8, ClearcoatRoughnessFactor: 0.2}
	if err := m.SetExtension(ExtMaterialsClearcoat, set); err != nil {
		t.Fatal(err)
	}
	got, ok := m.Clearcoat()
	if !ok {
		t.Fatal("clearcoat not found")
	}
	if got.ClearcoatFactor != 0.8 || got.ClearcoatRoughnessFactor != 0.2 {
		t.Errorf("clearcoat = %+v", got)
	}
	// Absent extension.
	if _, ok := (&Material{}).Clearcoat(); ok {
		t.Error("expected clearcoat absent")
	}
}

func TestSheenRoundTrip(t *testing.T) {
	m := &Material{}
	set := MaterialsSheen{SheenColorFactor: &[3]float64{0.5, 0.4, 0.3}, SheenRoughnessFactor: 0.6}
	if err := m.SetExtension(ExtMaterialsSheen, set); err != nil {
		t.Fatal(err)
	}
	got, ok := m.Sheen()
	if !ok {
		t.Fatal("sheen not found")
	}
	if got.SheenColorFactor == nil || *got.SheenColorFactor != [3]float64{0.5, 0.4, 0.3} {
		t.Errorf("sheen color = %+v", got.SheenColorFactor)
	}
	if got.SheenRoughnessFactor != 0.6 {
		t.Errorf("sheen roughness = %v", got.SheenRoughnessFactor)
	}
}

func TestSpecularRoundTrip(t *testing.T) {
	m := &Material{}
	sf := 0.75
	set := MaterialsSpecular{SpecularFactor: &sf, SpecularColorFactor: &[3]float64{1, 0.5, 0}}
	if err := m.SetExtension(ExtMaterialsSpecular, set); err != nil {
		t.Fatal(err)
	}
	got, ok := m.Specular()
	if !ok {
		t.Fatal("specular not found")
	}
	if got.SpecularFactor == nil || *got.SpecularFactor != 0.75 {
		t.Errorf("specular factor = %v", got.SpecularFactor)
	}
	if got.SpecularColorFactor == nil || *got.SpecularColorFactor != [3]float64{1, 0.5, 0} {
		t.Errorf("specular color = %v", got.SpecularColorFactor)
	}
}

func TestVolumeRoundTrip(t *testing.T) {
	m := &Material{}
	ad := 2.5
	set := MaterialsVolume{ThicknessFactor: 1.5, AttenuationDistance: &ad, AttenuationColor: &[3]float64{0.9, 0.9, 1}}
	if err := m.SetExtension(ExtMaterialsVolume, set); err != nil {
		t.Fatal(err)
	}
	got, ok := m.Volume()
	if !ok {
		t.Fatal("volume not found")
	}
	if got.ThicknessFactor != 1.5 {
		t.Errorf("thickness = %v", got.ThicknessFactor)
	}
	if got.AttenuationDistance == nil || *got.AttenuationDistance != 2.5 {
		t.Errorf("attenuation distance = %v", got.AttenuationDistance)
	}
	if got.AttenuationColor == nil || *got.AttenuationColor != [3]float64{0.9, 0.9, 1} {
		t.Errorf("attenuation color = %v", got.AttenuationColor)
	}
}

func TestNewExtensionsCoexist(t *testing.T) {
	// Multiple extensions on one material must not clobber each other.
	m := &Material{}
	if err := m.SetExtension(ExtMaterialsClearcoat, MaterialsClearcoat{ClearcoatFactor: 1}); err != nil {
		t.Fatal(err)
	}
	if err := m.SetExtension(ExtMaterialsVolume, MaterialsVolume{ThicknessFactor: 2}); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Clearcoat(); !ok {
		t.Error("clearcoat lost after adding volume")
	}
	if _, ok := m.Volume(); !ok {
		t.Error("volume missing")
	}
}
