package gltf

import "fmt"

// EffectiveWeights returns the morph-target weights in effect for the given
// mesh when instantiated by node. Per the specification, a node's weights
// override the mesh's default weights; when neither is present a nil slice is
// returned. node may be nil to use only the mesh weights.
func EffectiveWeights(node *Node, mesh *Mesh) []float64 {
	if node != nil && len(node.Weights) > 0 {
		return node.Weights
	}
	return mesh.Weights
}

// DecodeMorphTargetVec3 decodes the VEC3 delta values of a single morph target
// attribute (for example "POSITION" or "NORMAL") on a primitive. targetIndex
// selects the entry in the primitive's Targets slice. Buffers must be resolved
// first.
func (d *Document) DecodeMorphTargetVec3(prim *Primitive, targetIndex int, attribute string) ([][3]float32, error) {
	if targetIndex < 0 || targetIndex >= len(prim.Targets) {
		return nil, errIndexRange("morph target", targetIndex, len(prim.Targets))
	}
	acc, ok := prim.Targets[targetIndex][attribute]
	if !ok {
		return nil, fmt.Errorf("gltf: morph target %d has no %q attribute", targetIndex, attribute)
	}
	return d.DecodeAccessorVec3(acc)
}

// DecodeMorphTargetsVec3 decodes the VEC3 deltas of the given attribute for
// every morph target of the primitive, returning one delta slice per target.
// Buffers must be resolved first.
func (d *Document) DecodeMorphTargetsVec3(prim *Primitive, attribute string) ([][][3]float32, error) {
	out := make([][][3]float32, len(prim.Targets))
	for i := range prim.Targets {
		deltas, err := d.DecodeMorphTargetVec3(prim, i, attribute)
		if err != nil {
			return nil, err
		}
		out[i] = deltas
	}
	return out, nil
}

// BlendMorphTargetsVec3 applies weighted morph-target deltas to base vertex
// data, returning base + sum(weights[i] * targets[i]). Each target slice and
// base must have the same length; weights shorter than targets treat missing
// entries as zero. The base slice is not modified.
func BlendMorphTargetsVec3(base [][3]float32, targets [][][3]float32, weights []float64) [][3]float32 {
	out := make([][3]float32, len(base))
	copy(out, base)
	for ti, target := range targets {
		if ti >= len(weights) {
			break
		}
		w := float32(weights[ti])
		if w == 0 {
			continue
		}
		n := len(out)
		if len(target) < n {
			n = len(target)
		}
		for v := 0; v < n; v++ {
			out[v][0] += w * target[v][0]
			out[v][1] += w * target[v][1]
			out[v][2] += w * target[v][2]
		}
	}
	return out
}

// MorphedPositions returns a primitive's POSITION attribute with the primitive's
// morph targets applied at the given weights. It is a convenience combining
// [Document.DecodeAccessorVec3], [Document.DecodeMorphTargetsVec3], and
// [BlendMorphTargetsVec3]. Buffers must be resolved first.
func (d *Document) MorphedPositions(prim *Primitive, weights []float64) ([][3]float32, error) {
	posIdx, ok := prim.Attributes["POSITION"]
	if !ok {
		return nil, fmt.Errorf("gltf: primitive has no POSITION attribute")
	}
	base, err := d.DecodeAccessorVec3(posIdx)
	if err != nil {
		return nil, err
	}
	if len(prim.Targets) == 0 || len(weights) == 0 {
		return base, nil
	}
	targets, err := d.DecodeMorphTargetsVec3(prim, "POSITION")
	if err != nil {
		return nil, err
	}
	return BlendMorphTargetsVec3(base, targets, weights), nil
}
