package gltf

import "math"

// Vec3 is a three-component vector of float64 values. It is used for node
// translation and scale, and as a general-purpose 3D vector by the transform
// helpers.
type Vec3 [3]float64

// Vec4 is a four-component vector of float64 values.
type Vec4 [4]float64

// Quat is a rotation quaternion stored in glTF component order (x, y, z, w).
// The identity rotation is [0, 0, 0, 1].
type Quat [4]float64

// Mat4 is a 4x4 matrix stored in column-major order, matching the layout of a
// glTF [Node] matrix: element (row r, column c) is at index c*4+r.
type Mat4 [16]float64

// IdentityQuat is the rotation quaternion representing no rotation.
var IdentityQuat = Quat{0, 0, 0, 1}

// IdentityMatrix returns the 4x4 identity matrix.
func IdentityMatrix() Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// At returns the element at the given row and column (both 0-based).
func (m Mat4) At(row, col int) float64 {
	return m[col*4+row]
}

// Mul returns the matrix product m*n, using column-major conventions. When m
// and n are transforms, the result applies n first and then m.
func (m Mat4) Mul(n Mat4) Mat4 {
	var out Mat4
	for c := 0; c < 4; c++ {
		for r := 0; r < 4; r++ {
			var sum float64
			for k := 0; k < 4; k++ {
				sum += m[k*4+r] * n[c*4+k]
			}
			out[c*4+r] = sum
		}
	}
	return out
}

// MulVec4 returns the product of the matrix and the column vector v.
func (m Mat4) MulVec4(v Vec4) Vec4 {
	var out Vec4
	for r := 0; r < 4; r++ {
		out[r] = m[0*4+r]*v[0] + m[1*4+r]*v[1] + m[2*4+r]*v[2] + m[3*4+r]*v[3]
	}
	return out
}

// TransformPoint applies the matrix to a 3D point (implicit w=1) and performs
// the perspective divide when the resulting w is not 1.
func (m Mat4) TransformPoint(p Vec3) Vec3 {
	v := m.MulVec4(Vec4{p[0], p[1], p[2], 1})
	if v[3] != 0 && v[3] != 1 {
		return Vec3{v[0] / v[3], v[1] / v[3], v[2] / v[3]}
	}
	return Vec3{v[0], v[1], v[2]}
}

// Normalize returns the quaternion scaled to unit length. A zero quaternion is
// returned as the identity.
func (q Quat) Normalize() Quat {
	n := math.Sqrt(q[0]*q[0] + q[1]*q[1] + q[2]*q[2] + q[3]*q[3])
	if n == 0 {
		return IdentityQuat
	}
	return Quat{q[0] / n, q[1] / n, q[2] / n, q[3] / n}
}

// Matrix returns the 4x4 rotation matrix equivalent to the quaternion. The
// quaternion is normalized first.
func (q Quat) Matrix() Mat4 {
	q = q.Normalize()
	x, y, z, w := q[0], q[1], q[2], q[3]
	xx, yy, zz := x*x, y*y, z*z
	xy, xz, yz := x*y, x*z, y*z
	wx, wy, wz := w*x, w*y, w*z

	var m Mat4
	m[0] = 1 - 2*(yy+zz)
	m[1] = 2 * (xy + wz)
	m[2] = 2 * (xz - wy)
	m[4] = 2 * (xy - wz)
	m[5] = 1 - 2*(xx+zz)
	m[6] = 2 * (yz + wx)
	m[8] = 2 * (xz + wy)
	m[9] = 2 * (yz - wx)
	m[10] = 1 - 2*(xx+yy)
	m[15] = 1
	return m
}

// Slerp returns the spherical linear interpolation between quaternions a and b
// at parameter t in [0,1]. Both are treated as unit quaternions; the result is
// normalized. It is the interpolation glTF uses for rotation channels.
func Slerp(a, b Quat, t float64) Quat {
	a = a.Normalize()
	b = b.Normalize()
	dot := a[0]*b[0] + a[1]*b[1] + a[2]*b[2] + a[3]*b[3]
	// Take the shorter arc.
	if dot < 0 {
		b = Quat{-b[0], -b[1], -b[2], -b[3]}
		dot = -dot
	}
	const threshold = 0.9995
	if dot > threshold {
		// Very close: fall back to normalized linear interpolation.
		r := Quat{
			a[0] + t*(b[0]-a[0]),
			a[1] + t*(b[1]-a[1]),
			a[2] + t*(b[2]-a[2]),
			a[3] + t*(b[3]-a[3]),
		}
		return r.Normalize()
	}
	theta0 := math.Acos(dot)
	theta := theta0 * t
	sinTheta0 := math.Sin(theta0)
	sinTheta := math.Sin(theta)
	s0 := math.Cos(theta) - dot*sinTheta/sinTheta0
	s1 := sinTheta / sinTheta0
	return Quat{
		s0*a[0] + s1*b[0],
		s0*a[1] + s1*b[1],
		s0*a[2] + s1*b[2],
		s0*a[3] + s1*b[3],
	}
}

// TRS composes a translation, rotation, and scale into a single column-major
// transform matrix, equivalent to T * R * S: scale is applied first, then
// rotation, then translation.
func TRS(t Vec3, r Quat, s Vec3) Mat4 {
	m := r.Matrix()
	// Scale each rotation column.
	m[0] *= s[0]
	m[1] *= s[0]
	m[2] *= s[0]
	m[4] *= s[1]
	m[5] *= s[1]
	m[6] *= s[1]
	m[8] *= s[2]
	m[9] *= s[2]
	m[10] *= s[2]
	// Set translation.
	m[12] = t[0]
	m[13] = t[1]
	m[14] = t[2]
	return m
}

// Decompose extracts the translation, rotation, and scale that produce the
// matrix when combined by [TRS]. It assumes m is a valid affine transform
// (no shear); a negative determinant is folded into the X scale.
func (m Mat4) Decompose() (translation Vec3, rotation Quat, scale Vec3) {
	translation = Vec3{m[12], m[13], m[14]}

	col0 := Vec3{m[0], m[1], m[2]}
	col1 := Vec3{m[4], m[5], m[6]}
	col2 := Vec3{m[8], m[9], m[10]}

	sx := vecLen(col0)
	sy := vecLen(col1)
	sz := vecLen(col2)

	// Fold a mirror (negative determinant) into the X axis.
	if mat3Det(m) < 0 {
		sx = -sx
	}
	scale = Vec3{sx, sy, sz}

	// Build the pure rotation matrix by removing scale.
	rm := IdentityMatrix()
	if sx != 0 {
		rm[0], rm[1], rm[2] = col0[0]/sx, col0[1]/sx, col0[2]/sx
	}
	if sy != 0 {
		rm[4], rm[5], rm[6] = col1[0]/sy, col1[1]/sy, col1[2]/sy
	}
	if sz != 0 {
		rm[8], rm[9], rm[10] = col2[0]/sz, col2[1]/sz, col2[2]/sz
	}
	rotation = quatFromMatrix(rm)
	return translation, rotation, scale
}

// vecLen returns the Euclidean length of a Vec3.
func vecLen(v Vec3) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

// mat3Det returns the determinant of the upper-left 3x3 block of m.
func mat3Det(m Mat4) float64 {
	return m[0]*(m[5]*m[10]-m[6]*m[9]) -
		m[4]*(m[1]*m[10]-m[2]*m[9]) +
		m[8]*(m[1]*m[6]-m[2]*m[5])
}

// quatFromMatrix converts a pure-rotation (orthonormal) matrix into a
// quaternion using Shepperd's method.
func quatFromMatrix(m Mat4) Quat {
	// Access as row-major rotation elements.
	m00, m11, m22 := m[0], m[5], m[10]
	trace := m00 + m11 + m22
	var q Quat
	switch {
	case trace > 0:
		s := math.Sqrt(trace+1.0) * 2 // s = 4*w
		q[3] = 0.25 * s
		q[0] = (m[6] - m[9]) / s
		q[1] = (m[8] - m[2]) / s
		q[2] = (m[1] - m[4]) / s
	case m00 > m11 && m00 > m22:
		s := math.Sqrt(1.0+m00-m11-m22) * 2 // s = 4*x
		q[3] = (m[6] - m[9]) / s
		q[0] = 0.25 * s
		q[1] = (m[4] + m[1]) / s
		q[2] = (m[8] + m[2]) / s
	case m11 > m22:
		s := math.Sqrt(1.0+m11-m00-m22) * 2 // s = 4*y
		q[3] = (m[8] - m[2]) / s
		q[0] = (m[4] + m[1]) / s
		q[1] = 0.25 * s
		q[2] = (m[9] + m[6]) / s
	default:
		s := math.Sqrt(1.0+m22-m00-m11) * 2 // s = 4*z
		q[3] = (m[1] - m[4]) / s
		q[0] = (m[8] + m[2]) / s
		q[1] = (m[9] + m[6]) / s
		q[2] = 0.25 * s
	}
	return q.Normalize()
}

// Inverse returns the inverse of the matrix and reports whether it is
// invertible. When the matrix is singular the identity matrix is returned with
// ok=false.
func (m Mat4) Inverse() (Mat4, bool) {
	var inv Mat4
	inv[0] = m[5]*m[10]*m[15] - m[5]*m[11]*m[14] - m[9]*m[6]*m[15] + m[9]*m[7]*m[14] + m[13]*m[6]*m[11] - m[13]*m[7]*m[10]
	inv[4] = -m[4]*m[10]*m[15] + m[4]*m[11]*m[14] + m[8]*m[6]*m[15] - m[8]*m[7]*m[14] - m[12]*m[6]*m[11] + m[12]*m[7]*m[10]
	inv[8] = m[4]*m[9]*m[15] - m[4]*m[11]*m[13] - m[8]*m[5]*m[15] + m[8]*m[7]*m[13] + m[12]*m[5]*m[11] - m[12]*m[7]*m[9]
	inv[12] = -m[4]*m[9]*m[14] + m[4]*m[10]*m[13] + m[8]*m[5]*m[14] - m[8]*m[6]*m[13] - m[12]*m[5]*m[10] + m[12]*m[6]*m[9]
	inv[1] = -m[1]*m[10]*m[15] + m[1]*m[11]*m[14] + m[9]*m[2]*m[15] - m[9]*m[3]*m[14] - m[13]*m[2]*m[11] + m[13]*m[3]*m[10]
	inv[5] = m[0]*m[10]*m[15] - m[0]*m[11]*m[14] - m[8]*m[2]*m[15] + m[8]*m[3]*m[14] + m[12]*m[2]*m[11] - m[12]*m[3]*m[10]
	inv[9] = -m[0]*m[9]*m[15] + m[0]*m[11]*m[13] + m[8]*m[1]*m[15] - m[8]*m[3]*m[13] - m[12]*m[1]*m[11] + m[12]*m[3]*m[9]
	inv[13] = m[0]*m[9]*m[14] - m[0]*m[10]*m[13] - m[8]*m[1]*m[14] + m[8]*m[2]*m[13] + m[12]*m[1]*m[10] - m[12]*m[2]*m[9]
	inv[2] = m[1]*m[6]*m[15] - m[1]*m[7]*m[14] - m[5]*m[2]*m[15] + m[5]*m[3]*m[14] + m[13]*m[2]*m[7] - m[13]*m[3]*m[6]
	inv[6] = -m[0]*m[6]*m[15] + m[0]*m[7]*m[14] + m[4]*m[2]*m[15] - m[4]*m[3]*m[14] - m[12]*m[2]*m[7] + m[12]*m[3]*m[6]
	inv[10] = m[0]*m[5]*m[15] - m[0]*m[7]*m[13] - m[4]*m[1]*m[15] + m[4]*m[3]*m[13] + m[12]*m[1]*m[7] - m[12]*m[3]*m[5]
	inv[14] = -m[0]*m[5]*m[14] + m[0]*m[6]*m[13] + m[4]*m[1]*m[14] - m[4]*m[2]*m[13] - m[12]*m[1]*m[6] + m[12]*m[2]*m[5]
	inv[3] = -m[1]*m[6]*m[11] + m[1]*m[7]*m[10] + m[5]*m[2]*m[11] - m[5]*m[3]*m[10] - m[9]*m[2]*m[7] + m[9]*m[3]*m[6]
	inv[7] = m[0]*m[6]*m[11] - m[0]*m[7]*m[10] - m[4]*m[2]*m[11] + m[4]*m[3]*m[10] + m[8]*m[2]*m[7] - m[8]*m[3]*m[6]
	inv[11] = -m[0]*m[5]*m[11] + m[0]*m[7]*m[9] + m[4]*m[1]*m[11] - m[4]*m[3]*m[9] - m[8]*m[1]*m[7] + m[8]*m[3]*m[5]
	inv[15] = m[0]*m[5]*m[10] - m[0]*m[6]*m[9] - m[4]*m[1]*m[10] + m[4]*m[2]*m[9] + m[8]*m[1]*m[6] - m[8]*m[2]*m[5]

	det := m[0]*inv[0] + m[1]*inv[4] + m[2]*inv[8] + m[3]*inv[12]
	if det == 0 {
		return IdentityMatrix(), false
	}
	invDet := 1.0 / det
	for i := range inv {
		inv[i] *= invDet
	}
	return inv, true
}

// LocalMatrix returns the node's local transform as a column-major matrix. If
// the node has an explicit matrix it is returned directly; otherwise the
// translation, rotation, and scale (each defaulting to identity) are composed
// with [TRS].
func (n *Node) LocalMatrix() Mat4 {
	if n.Matrix != nil {
		return Mat4(*n.Matrix)
	}
	t := Vec3{0, 0, 0}
	if n.Translation != nil {
		t = Vec3{n.Translation[0], n.Translation[1], n.Translation[2]}
	}
	r := IdentityQuat
	if n.Rotation != nil {
		r = Quat{n.Rotation[0], n.Rotation[1], n.Rotation[2], n.Rotation[3]}
	}
	s := Vec3{1, 1, 1}
	if n.Scale != nil {
		s = Vec3{n.Scale[0], n.Scale[1], n.Scale[2]}
	}
	return TRS(t, r, s)
}

// parentIndex builds a child->parent lookup for the node hierarchy. A node that
// is not referenced as any node's child maps to -1 (a root).
func (d *Document) parentIndex() []int {
	parents := make([]int, len(d.Nodes))
	for i := range parents {
		parents[i] = -1
	}
	for i := range d.Nodes {
		for _, c := range d.Nodes[i].Children {
			if c >= 0 && c < len(parents) {
				parents[c] = i
			}
		}
	}
	return parents
}

// GlobalMatrix returns the world (global) transform of the node at nodeIndex,
// obtained by multiplying the local matrices of every ancestor from the root
// down to the node. It returns an error if nodeIndex is out of range or the
// hierarchy contains a cycle.
func (d *Document) GlobalMatrix(nodeIndex int) (Mat4, error) {
	if nodeIndex < 0 || nodeIndex >= len(d.Nodes) {
		return IdentityMatrix(), errIndexRange("node", nodeIndex, len(d.Nodes))
	}
	parents := d.parentIndex()
	// Walk to the root, collecting the chain.
	chain := make([]int, 0, len(d.Nodes))
	seen := make([]bool, len(d.Nodes))
	for i := nodeIndex; i != -1; i = parents[i] {
		if seen[i] {
			return IdentityMatrix(), errCycle(i)
		}
		seen[i] = true
		chain = append(chain, i)
	}
	// Multiply from root (last in chain) down to the node.
	m := IdentityMatrix()
	for i := len(chain) - 1; i >= 0; i-- {
		m = m.Mul(d.Nodes[chain[i]].LocalMatrix())
	}
	return m, nil
}
