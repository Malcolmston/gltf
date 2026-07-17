package gltf

import "fmt"

// InverseBindMatrices returns the skin's inverse bind matrices. When the skin
// declares an inverseBindMatrices accessor it is decoded; otherwise the glTF
// default of one identity matrix per joint is returned. Buffers must be
// resolved first when an accessor is present.
func (d *Document) InverseBindMatrices(skinIndex int) ([]Mat4, error) {
	if skinIndex < 0 || skinIndex >= len(d.Skins) {
		return nil, errIndexRange("skin", skinIndex, len(d.Skins))
	}
	sk := &d.Skins[skinIndex]
	if sk.InverseBindMatrices == nil {
		out := make([]Mat4, len(sk.Joints))
		for i := range out {
			out[i] = IdentityMatrix()
		}
		return out, nil
	}
	ibm, err := d.DecodeAccessorMat4(*sk.InverseBindMatrices)
	if err != nil {
		return nil, err
	}
	if len(ibm) < len(sk.Joints) {
		return nil, fmt.Errorf("gltf: skin %d has %d inverse bind matrices for %d joints", skinIndex, len(ibm), len(sk.Joints))
	}
	return ibm, nil
}

// JointMatrices computes the skinning matrix for each joint of the skin as
// globalJointTransform * inverseBindMatrix. The result has one matrix per
// joint, in joint order. This is the joint matrix used by vertex skinning when
// the skinned mesh node's own transform is identity; use [Document.JointMatricesForNode]
// to account for a non-identity mesh node. Buffers must be resolved first.
func (d *Document) JointMatrices(skinIndex int) ([]Mat4, error) {
	return d.jointMatrices(skinIndex, nil)
}

// JointMatricesForNode computes joint matrices for the skin as it is used by
// the mesh node meshNodeIndex, following the glTF formula
//
//	jointMatrix(j) = inverse(globalTransform(meshNode)) *
//	                 globalTransform(joints[j]) * inverseBindMatrices[j]
//
// The mesh node's inverse global transform maps joints into the mesh's local
// space. Buffers must be resolved first.
func (d *Document) JointMatricesForNode(skinIndex, meshNodeIndex int) ([]Mat4, error) {
	if meshNodeIndex < 0 || meshNodeIndex >= len(d.Nodes) {
		return nil, errIndexRange("node", meshNodeIndex, len(d.Nodes))
	}
	global, err := d.GlobalMatrix(meshNodeIndex)
	if err != nil {
		return nil, err
	}
	inv, ok := global.Inverse()
	if !ok {
		return nil, fmt.Errorf("gltf: mesh node %d global transform is not invertible", meshNodeIndex)
	}
	return d.jointMatrices(skinIndex, &inv)
}

// jointMatrices is the shared implementation. meshInverse, when non-nil, is
// premultiplied into every joint matrix.
func (d *Document) jointMatrices(skinIndex int, meshInverse *Mat4) ([]Mat4, error) {
	if skinIndex < 0 || skinIndex >= len(d.Skins) {
		return nil, errIndexRange("skin", skinIndex, len(d.Skins))
	}
	sk := &d.Skins[skinIndex]
	ibm, err := d.InverseBindMatrices(skinIndex)
	if err != nil {
		return nil, err
	}
	out := make([]Mat4, len(sk.Joints))
	for j, joint := range sk.Joints {
		global, err := d.GlobalMatrix(joint)
		if err != nil {
			return nil, err
		}
		m := global.Mul(ibm[j])
		if meshInverse != nil {
			m = meshInverse.Mul(m)
		}
		out[j] = m
	}
	return out, nil
}
