package gltf

import "math"

// Add returns the component-wise sum v+o.
func (v Vec3) Add(o Vec3) Vec3 {
	return Vec3{v[0] + o[0], v[1] + o[1], v[2] + o[2]}
}

// Sub returns the component-wise difference v-o.
func (v Vec3) Sub(o Vec3) Vec3 {
	return Vec3{v[0] - o[0], v[1] - o[1], v[2] - o[2]}
}

// Scale returns v with every component multiplied by s.
func (v Vec3) Scale(s float64) Vec3 {
	return Vec3{v[0] * s, v[1] * s, v[2] * s}
}

// Dot returns the dot product of v and o.
func (v Vec3) Dot(o Vec3) float64 {
	return v[0]*o[0] + v[1]*o[1] + v[2]*o[2]
}

// Cross returns the cross product v×o, a vector perpendicular to both operands.
func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{
		v[1]*o[2] - v[2]*o[1],
		v[2]*o[0] - v[0]*o[2],
		v[0]*o[1] - v[1]*o[0],
	}
}

// Length returns the Euclidean length (magnitude) of v.
func (v Vec3) Length() float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

// Normalize returns v scaled to unit length. A zero-length vector is returned
// unchanged (as the zero vector).
func (v Vec3) Normalize() Vec3 {
	n := v.Length()
	if n == 0 {
		return Vec3{0, 0, 0}
	}
	return Vec3{v[0] / n, v[1] / n, v[2] / n}
}

// Lerp returns the linear interpolation between v and o at parameter t. t=0
// yields v and t=1 yields o; values outside [0,1] extrapolate.
func (v Vec3) Lerp(o Vec3, t float64) Vec3 {
	return Vec3{
		v[0] + t*(o[0]-v[0]),
		v[1] + t*(o[1]-v[1]),
		v[2] + t*(o[2]-v[2]),
	}
}

// Mul returns the Hamilton product q*o. When both are unit quaternions the
// result represents applying rotation o first and then q, and is itself a unit
// quaternion. Components are in glTF order (x, y, z, w).
func (q Quat) Mul(o Quat) Quat {
	ax, ay, az, aw := q[0], q[1], q[2], q[3]
	bx, by, bz, bw := o[0], o[1], o[2], o[3]
	return Quat{
		aw*bx + ax*bw + ay*bz - az*by,
		aw*by - ax*bz + ay*bw + az*bx,
		aw*bz + ax*by - ay*bx + az*bw,
		aw*bw - ax*bx - ay*by - az*bz,
	}
}

// Conjugate returns the quaternion conjugate (-x, -y, -z, w). For a unit
// quaternion the conjugate is also its inverse.
func (q Quat) Conjugate() Quat {
	return Quat{-q[0], -q[1], -q[2], q[3]}
}

// Dot returns the four-component dot product of q and o.
func (q Quat) Dot(o Quat) float64 {
	return q[0]*o[0] + q[1]*o[1] + q[2]*o[2] + q[3]*o[3]
}

// Rotate applies the rotation represented by q to the vector v and returns the
// rotated vector. The quaternion is normalized first.
func (q Quat) Rotate(v Vec3) Vec3 {
	q = q.Normalize()
	u := Vec3{q[0], q[1], q[2]}
	w := q[3]
	// v' = v + 2*w*(u×v) + 2*(u×(u×v))
	t := u.Cross(v).Scale(2)
	return v.Add(t.Scale(w)).Add(u.Cross(t))
}

// QuatFromAxisAngle returns the unit quaternion representing a rotation of angle
// radians about axis. A zero-length axis yields the identity rotation.
func QuatFromAxisAngle(axis Vec3, angle float64) Quat {
	a := axis.Normalize()
	if a == (Vec3{0, 0, 0}) {
		return IdentityQuat
	}
	half := angle / 2
	s := math.Sin(half)
	return Quat{a[0] * s, a[1] * s, a[2] * s, math.Cos(half)}
}

// QuatFromEuler returns the unit quaternion for the given intrinsic Euler
// angles in radians, applied in X, then Y, then Z order (roll, pitch, yaw).
// The result equals QuatFromAxisAngle(Z,z) * QuatFromAxisAngle(Y,y) *
// QuatFromAxisAngle(X,x).
func QuatFromEuler(x, y, z float64) Quat {
	qx := QuatFromAxisAngle(Vec3{1, 0, 0}, x)
	qy := QuatFromAxisAngle(Vec3{0, 1, 0}, y)
	qz := QuatFromAxisAngle(Vec3{0, 0, 1}, z)
	return qz.Mul(qy).Mul(qx)
}

// Transpose returns the matrix transpose of m.
func (m Mat4) Transpose() Mat4 {
	var out Mat4
	for c := 0; c < 4; c++ {
		for r := 0; r < 4; r++ {
			out[c*4+r] = m[r*4+c]
		}
	}
	return out
}

// TransformDir applies the matrix to a direction vector (implicit w=0), so the
// translation component is ignored. It is the correct transform for normals'
// tangent directions but not for normals themselves under non-uniform scale.
func (m Mat4) TransformDir(v Vec3) Vec3 {
	return Vec3{
		m[0]*v[0] + m[4]*v[1] + m[8]*v[2],
		m[1]*v[0] + m[5]*v[1] + m[9]*v[2],
		m[2]*v[0] + m[6]*v[1] + m[10]*v[2],
	}
}

// LookAt returns a right-handed view matrix that positions the camera at eye
// looking toward center, with up giving the approximate world up direction. It
// matches the convention of gluLookAt: the resulting matrix maps world space
// into the camera's view space (camera looking down its local -Z axis).
func LookAt(eye, center, up Vec3) Mat4 {
	f := center.Sub(eye).Normalize()
	s := f.Cross(up).Normalize()
	u := s.Cross(f)
	var m Mat4
	m[0], m[4], m[8] = s[0], s[1], s[2]
	m[1], m[5], m[9] = u[0], u[1], u[2]
	m[2], m[6], m[10] = -f[0], -f[1], -f[2]
	m[12] = -s.Dot(eye)
	m[13] = -u.Dot(eye)
	m[14] = f.Dot(eye)
	m[15] = 1
	return m
}

// Perspective returns a right-handed perspective projection matrix using the
// glTF camera convention. yfov is the vertical field of view in radians, aspect
// is width/height, and near/far are the clip-plane distances. A non-positive
// far value produces an infinite-far projection.
func Perspective(yfov, aspect, near, far float64) Mat4 {
	return perspectiveMatrix(yfov, aspect, near, far, far > 0)
}

// Orthographic returns a right-handed orthographic projection matrix using the
// glTF camera convention. xmag and ymag are the horizontal and vertical
// magnifications (half-extents), and near/far are the clip-plane distances.
func Orthographic(xmag, ymag, near, far float64) Mat4 {
	return orthographicMatrix(xmag, ymag, near, far)
}

// perspectiveMatrix builds a glTF-convention perspective projection. When
// finite is false the far plane is treated as infinite.
func perspectiveMatrix(yfov, aspect, near, far float64, finite bool) Mat4 {
	var m Mat4
	t := math.Tan(yfov / 2)
	m[0] = 1 / (aspect * t)
	m[5] = 1 / t
	m[11] = -1
	if finite {
		m[10] = (far + near) / (near - far)
		m[14] = (2 * far * near) / (near - far)
	} else {
		m[10] = -1
		m[14] = -2 * near
	}
	return m
}

// orthographicMatrix builds a glTF-convention orthographic projection.
func orthographicMatrix(xmag, ymag, near, far float64) Mat4 {
	var m Mat4
	m[0] = 1 / xmag
	m[5] = 1 / ymag
	m[10] = 2 / (near - far)
	m[14] = (far + near) / (near - far)
	m[15] = 1
	return m
}
