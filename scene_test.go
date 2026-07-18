package gltf

import (
	"reflect"
	"testing"
)

func hierarchyDoc() *Document {
	// 0 -> 1 -> 2, and 3 as a separate root. Node 4 is unreferenced (also a root).
	return &Document{
		Asset:  Asset{Version: Version},
		Scenes: []Scene{{Nodes: []Index{0, 3}}},
		Nodes: []Node{
			{Translation: &[3]float64{1, 0, 0}, Children: []Index{1}},
			{Translation: &[3]float64{0, 1, 0}, Children: []Index{2}},
			{Translation: &[3]float64{0, 0, 1}},
			{Translation: &[3]float64{5, 5, 5}},
			{Translation: &[3]float64{9, 9, 9}},
		},
	}
}

func TestRootNodes(t *testing.T) {
	d := hierarchyDoc()
	if got := d.RootNodes(); !reflect.DeepEqual(got, []int{0, 3, 4}) {
		t.Errorf("RootNodes = %v, want [0 3 4]", got)
	}
}

func TestGlobalMatrices(t *testing.T) {
	d := hierarchyDoc()
	gm, err := d.GlobalMatrices()
	if err != nil {
		t.Fatal(err)
	}
	// Node 2 global translation is the sum of the chain 0,1,2.
	if got := gm[2].TransformPoint(Vec3{0, 0, 0}); !vec3Close(got, Vec3{1, 1, 1}, 1e-12) {
		t.Errorf("node 2 global origin = %v, want {1,1,1}", got)
	}
	if got := gm[1].TransformPoint(Vec3{0, 0, 0}); !vec3Close(got, Vec3{1, 1, 0}, 1e-12) {
		t.Errorf("node 1 global origin = %v, want {1,1,0}", got)
	}
	// Compare against GlobalMatrix for consistency.
	for i := range d.Nodes {
		single, err := d.GlobalMatrix(i)
		if err != nil {
			t.Fatal(err)
		}
		if !mat4Close(gm[i], single, 1e-12) {
			t.Errorf("node %d: GlobalMatrices != GlobalMatrix", i)
		}
	}
}

func TestGlobalMatricesCycle(t *testing.T) {
	d := &Document{
		Asset: Asset{Version: Version},
		Nodes: []Node{
			{Children: []Index{1}},
			{Children: []Index{0}},
		},
	}
	if _, err := d.GlobalMatrices(); err == nil {
		t.Error("expected cycle error")
	}
}

func TestNodesInScene(t *testing.T) {
	d := hierarchyDoc()
	nodes, err := d.NodesInScene(0)
	if err != nil {
		t.Fatal(err)
	}
	// Pre-order DFS from roots 0 and 3: 0,1,2 then 3. Node 4 is not in the scene.
	if !reflect.DeepEqual(nodes, []int{0, 1, 2, 3}) {
		t.Errorf("NodesInScene = %v, want [0 1 2 3]", nodes)
	}
	if _, err := d.NodesInScene(5); err == nil {
		t.Error("expected out-of-range scene error")
	}
}

func BenchmarkGlobalMatrices(b *testing.B) {
	d := hierarchyDoc()
	for i := 0; i < b.N; i++ {
		if _, err := d.GlobalMatrices(); err != nil {
			b.Fatal(err)
		}
	}
}
