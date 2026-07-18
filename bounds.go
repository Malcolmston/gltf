package gltf

import (
	"fmt"
	"math"
)

// Box is an axis-aligned bounding box in world or local space, defined by its
// minimum and maximum corner. An empty box (one that contains no points) has a
// Min component greater than the corresponding Max component; see [EmptyBox].
type Box struct {
	Min Vec3
	Max Vec3
}

// EmptyBox returns an empty bounding box: its Min is +Inf and its Max is -Inf on
// every axis, so the first point added by [Box.Add] establishes both corners.
func EmptyBox() Box {
	inf := math.Inf(1)
	return Box{
		Min: Vec3{inf, inf, inf},
		Max: Vec3{-inf, -inf, -inf},
	}
}

// Empty reports whether the box contains no points, which is true when any axis
// has Min greater than Max.
func (b Box) Empty() bool {
	return b.Min[0] > b.Max[0] || b.Min[1] > b.Max[1] || b.Min[2] > b.Max[2]
}

// Add returns the smallest box that contains both b and the point p.
func (b Box) Add(p Vec3) Box {
	for i := 0; i < 3; i++ {
		if p[i] < b.Min[i] {
			b.Min[i] = p[i]
		}
		if p[i] > b.Max[i] {
			b.Max[i] = p[i]
		}
	}
	return b
}

// Union returns the smallest box that contains both b and o. Empty operands are
// ignored, so unioning with an [EmptyBox] returns the other box.
func (b Box) Union(o Box) Box {
	if o.Empty() {
		return b
	}
	if b.Empty() {
		return o
	}
	return b.Add(o.Min).Add(o.Max)
}

// Center returns the midpoint of the box. It is only meaningful for a non-empty
// box.
func (b Box) Center() Vec3 {
	return Vec3{
		(b.Min[0] + b.Max[0]) / 2,
		(b.Min[1] + b.Max[1]) / 2,
		(b.Min[2] + b.Max[2]) / 2,
	}
}

// Size returns the extent of the box along each axis (Max minus Min). It is only
// meaningful for a non-empty box.
func (b Box) Size() Vec3 {
	return Vec3{
		b.Max[0] - b.Min[0],
		b.Max[1] - b.Min[1],
		b.Max[2] - b.Min[2],
	}
}

// Contains reports whether the point p lies within the box, inclusive of its
// boundary.
func (b Box) Contains(p Vec3) bool {
	return p[0] >= b.Min[0] && p[0] <= b.Max[0] &&
		p[1] >= b.Min[1] && p[1] <= b.Max[1] &&
		p[2] >= b.Min[2] && p[2] <= b.Max[2]
}

// Transform returns the axis-aligned bounding box that encloses b after each of
// its eight corners is transformed by the column-major matrix m. An empty box is
// returned unchanged.
func (b Box) Transform(m Mat4) Box {
	if b.Empty() {
		return b
	}
	out := EmptyBox()
	for i := 0; i < 8; i++ {
		corner := Vec3{b.Min[0], b.Min[1], b.Min[2]}
		if i&1 != 0 {
			corner[0] = b.Max[0]
		}
		if i&2 != 0 {
			corner[1] = b.Max[1]
		}
		if i&4 != 0 {
			corner[2] = b.Max[2]
		}
		out = out.Add(m.TransformPoint(corner))
	}
	return out
}

// AccessorBounds returns the axis-aligned bounding box of a VEC3 accessor (for
// example a POSITION attribute). When the accessor carries a three-component Min
// and Max they are used directly; otherwise the accessor is decoded and scanned.
// It returns an error if the accessor is not VEC3 or cannot be decoded.
func (d *Document) AccessorBounds(index int) (Box, error) {
	a, err := d.accessorAt(index)
	if err != nil {
		return Box{}, err
	}
	if a.Type != AccessorVec3 {
		return Box{}, fmt.Errorf("gltf: accessor %d is %s, want %s", index, a.Type, AccessorVec3)
	}
	if len(a.Min) >= 3 && len(a.Max) >= 3 {
		return Box{
			Min: Vec3{a.Min[0], a.Min[1], a.Min[2]},
			Max: Vec3{a.Max[0], a.Max[1], a.Max[2]},
		}, nil
	}
	verts, err := d.DecodeAccessorVec3(index)
	if err != nil {
		return Box{}, err
	}
	box := EmptyBox()
	for _, v := range verts {
		box = box.Add(Vec3{float64(v[0]), float64(v[1]), float64(v[2])})
	}
	return box, nil
}

// PrimitiveBounds returns the local-space bounding box of a primitive, computed
// from its POSITION attribute. It returns an error when the primitive has no
// POSITION attribute or its accessor cannot be read.
func (d *Document) PrimitiveBounds(p *Primitive) (Box, error) {
	idx, ok := p.Attributes["POSITION"]
	if !ok {
		return Box{}, fmt.Errorf("gltf: primitive has no POSITION attribute")
	}
	return d.AccessorBounds(idx)
}

// SceneBounds returns the world-space bounding box of every mesh reachable from
// the scene at sceneIndex, transforming each primitive's local bounds by the
// global matrix of the node that carries it. An empty scene (no meshes) yields
// an empty box. It returns an error for an out-of-range scene index or a node
// hierarchy cycle.
func (d *Document) SceneBounds(sceneIndex int) (Box, error) {
	nodes, err := d.NodesInScene(sceneIndex)
	if err != nil {
		return Box{}, err
	}
	globals, err := d.GlobalMatrices()
	if err != nil {
		return Box{}, err
	}
	box := EmptyBox()
	for _, ni := range nodes {
		n := &d.Nodes[ni]
		if n.Mesh == nil || *n.Mesh < 0 || *n.Mesh >= len(d.Meshes) {
			continue
		}
		mesh := &d.Meshes[*n.Mesh]
		for pi := range mesh.Primitives {
			local, err := d.PrimitiveBounds(&mesh.Primitives[pi])
			if err != nil {
				continue
			}
			box = box.Union(local.Transform(globals[ni]))
		}
	}
	return box, nil
}
