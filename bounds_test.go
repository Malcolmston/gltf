package gltf

import "testing"

func TestBoxAddUnion(t *testing.T) {
	b := EmptyBox()
	if !b.Empty() {
		t.Fatal("EmptyBox should be empty")
	}
	b = b.Add(Vec3{1, 2, 3})
	b = b.Add(Vec3{-1, 5, 0})
	if b.Min != (Vec3{-1, 2, 0}) || b.Max != (Vec3{1, 5, 3}) {
		t.Errorf("box after adds = %+v", b)
	}
	if b.Empty() {
		t.Error("box with points should not be empty")
	}
	if got := b.Center(); got != (Vec3{0, 3.5, 1.5}) {
		t.Errorf("Center = %v", got)
	}
	if got := b.Size(); got != (Vec3{2, 3, 3}) {
		t.Errorf("Size = %v", got)
	}
	if !b.Contains(Vec3{0, 3, 1}) {
		t.Error("should contain interior point")
	}
	if b.Contains(Vec3{2, 3, 1}) {
		t.Error("should not contain exterior point")
	}

	other := EmptyBox().Add(Vec3{10, 10, 10})
	u := b.Union(other)
	if u.Min != (Vec3{-1, 2, 0}) || u.Max != (Vec3{10, 10, 10}) {
		t.Errorf("Union = %+v", u)
	}
	// Union with empty is identity.
	if got := b.Union(EmptyBox()); got != b {
		t.Errorf("union with empty = %+v, want %+v", got, b)
	}
	if got := EmptyBox().Union(b); got != b {
		t.Errorf("empty union box = %+v, want %+v", got, b)
	}
}

func TestBoxTransform(t *testing.T) {
	b := Box{Min: Vec3{0, 0, 0}, Max: Vec3{1, 1, 1}}
	m := TRS(Vec3{10, 0, 0}, IdentityQuat, Vec3{2, 2, 2})
	tb := b.Transform(m)
	if !vec3Close(tb.Min, Vec3{10, 0, 0}, 1e-12) || !vec3Close(tb.Max, Vec3{12, 2, 2}, 1e-12) {
		t.Errorf("Transform = %+v", tb)
	}
	// Empty box stays empty.
	if got := EmptyBox().Transform(m); !got.Empty() {
		t.Error("transformed empty box should be empty")
	}
}

func TestAccessorBounds(t *testing.T) {
	d := &Document{Asset: Asset{Version: Version}}
	idx := d.AddAccessorVec3([][3]float32{{-1, -2, -3}, {4, 5, 6}, {0, 0, 0}})
	box, err := d.AccessorBounds(idx)
	if err != nil {
		t.Fatal(err)
	}
	if box.Min != (Vec3{-1, -2, -3}) || box.Max != (Vec3{4, 5, 6}) {
		t.Errorf("AccessorBounds = %+v", box)
	}
	// Wrong type errors.
	sc := d.AddIndicesUint32([]uint32{0, 1, 2})
	if _, err := d.AccessorBounds(sc); err == nil {
		t.Error("expected error for non-VEC3 accessor")
	}
}

func TestAccessorBoundsDecodePath(t *testing.T) {
	// Build a VEC3 accessor without Min/Max so the decode path runs.
	d := &Document{Asset: Asset{Version: Version}}
	idx := d.AddAccessorVec3([][3]float32{{2, 2, 2}, {-3, 7, 1}})
	d.Accessors[idx].Min = nil
	d.Accessors[idx].Max = nil
	box, err := d.AccessorBounds(idx)
	if err != nil {
		t.Fatal(err)
	}
	if box.Min != (Vec3{-3, 2, 1}) || box.Max != (Vec3{2, 7, 2}) {
		t.Errorf("decoded bounds = %+v", box)
	}
}

func TestSceneBounds(t *testing.T) {
	doc, _ := Triangle()
	// Triangle positions span (0,0,0)..(1,1,0). Shift the node by +10 in X.
	doc.Nodes[0].Translation = &[3]float64{10, 0, 0}
	box, err := doc.SceneBounds(0)
	if err != nil {
		t.Fatal(err)
	}
	if !vec3Close(box.Min, Vec3{10, 0, 0}, 1e-6) || !vec3Close(box.Max, Vec3{11, 1, 0}, 1e-6) {
		t.Errorf("SceneBounds = %+v", box)
	}
	if _, err := doc.SceneBounds(9); err == nil {
		t.Error("expected error for out-of-range scene")
	}
}

func BenchmarkAccessorBoundsDecode(b *testing.B) {
	d := &Document{Asset: Asset{Version: Version}}
	verts := make([][3]float32, 1000)
	for i := range verts {
		verts[i] = [3]float32{float32(i), float32(-i), 0}
	}
	idx := d.AddAccessorVec3(verts)
	d.Accessors[idx].Min = nil
	d.Accessors[idx].Max = nil
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := d.AccessorBounds(idx); err != nil {
			b.Fatal(err)
		}
	}
}
