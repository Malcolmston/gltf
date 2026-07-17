package gltf

import (
	"io"
	"math"
)

// TrianglePositions are the three VEC3 vertex positions used by Triangle.
var TrianglePositions = [3][3]float32{
	{0, 0, 0},
	{1, 0, 0},
	{0, 1, 0},
}

// Triangle builds a minimal, valid glTF document describing a single triangle:
// one buffer, one bufferView, one POSITION accessor, one primitive, one mesh,
// one node, and one scene. It returns the document together with the binary
// buffer data (the GLB BIN chunk). Buffer 0 is URI-less and references that
// data, which is also attached to the buffer's Data field so the document can
// be decoded immediately without calling ResolveBuffers.
func Triangle() (*Document, []byte) {
	bin := make([]byte, 0, len(TrianglePositions)*3*4)
	var minv = [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	var maxv = [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, p := range TrianglePositions {
		for c := 0; c < 3; c++ {
			bin = le.AppendUint32(bin, math.Float32bits(p[c]))
			minv[c] = math.Min(minv[c], float64(p[c]))
			maxv[c] = math.Max(maxv[c], float64(p[c]))
		}
	}

	posMode := PrimitiveTriangles
	target := TargetArrayBuffer
	doc := &Document{
		Asset: Asset{Version: Version, Generator: "github.com/malcolmston/gltf"},
		Scene: intPtr(0),
		Scenes: []Scene{
			{Name: "triangle", Nodes: []Index{0}},
		},
		Nodes: []Node{
			{Name: "triangle", Mesh: intPtr(0)},
		},
		Meshes: []Mesh{
			{
				Name: "triangle",
				Primitives: []Primitive{
					{
						Attributes: map[string]Index{"POSITION": 0},
						Mode:       &posMode,
					},
				},
			},
		},
		Accessors: []Accessor{
			{
				BufferView:    intPtr(0),
				ComponentType: ComponentFloat,
				Count:         len(TrianglePositions),
				Type:          AccessorVec3,
				Min:           minv[:],
				Max:           maxv[:],
			},
		},
		BufferViews: []BufferView{
			{
				Buffer:     0,
				ByteLength: len(bin),
				Target:     &target,
			},
		},
		Buffers: []Buffer{
			{ByteLength: len(bin), Data: bin},
		},
	}
	return doc, bin
}

// WriteTriangleGLB writes the single-triangle asset to w as a binary .glb.
func WriteTriangleGLB(w io.Writer) error {
	doc, bin := Triangle()
	return WriteGLB(w, doc, bin)
}

// WriteTriangleGLTF writes the single-triangle asset to w as an embedded .gltf,
// with the buffer stored as a base64 data URI so the file is self-contained.
func WriteTriangleGLTF(w io.Writer) error {
	doc, bin := Triangle()
	doc.Buffers[0].URI = EncodeDataURI(bin)
	return Encode(w, doc)
}

// intPtr returns a pointer to v, for building optional index fields.
func intPtr(v int) *Index {
	return &v
}
