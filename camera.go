package gltf

import "fmt"

// ProjectionMatrix returns the column-major projection matrix for the
// perspective camera, following the glTF 2.0 specification. When ZFar is nil the
// projection uses an infinite far plane. When AspectRatio is nil an aspect ratio
// of 1 is assumed (the specification defers to the viewport aspect ratio).
func (c *CameraPerspective) ProjectionMatrix() Mat4 {
	aspect := 1.0
	if c.AspectRatio != nil {
		aspect = *c.AspectRatio
	}
	if c.ZFar != nil {
		return perspectiveMatrix(c.YFOV, aspect, c.ZNear, *c.ZFar, true)
	}
	return perspectiveMatrix(c.YFOV, aspect, c.ZNear, 0, false)
}

// ProjectionMatrix returns the column-major projection matrix for the
// orthographic camera, following the glTF 2.0 specification.
func (c *CameraOrthographic) ProjectionMatrix() Mat4 {
	return orthographicMatrix(c.XMag, c.YMag, c.ZNear, c.ZFar)
}

// ProjectionMatrix returns the column-major projection matrix for the camera,
// dispatching on its Type to the perspective or orthographic projection. It
// returns an error when the camera is missing the projection block that matches
// its Type.
func (c *Camera) ProjectionMatrix() (Mat4, error) {
	switch c.Type {
	case CameraTypePerspective:
		if c.Perspective == nil {
			return IdentityMatrix(), fmt.Errorf("gltf: perspective camera has no perspective block")
		}
		return c.Perspective.ProjectionMatrix(), nil
	case CameraTypeOrthographic:
		if c.Orthographic == nil {
			return IdentityMatrix(), fmt.Errorf("gltf: orthographic camera has no orthographic block")
		}
		return c.Orthographic.ProjectionMatrix(), nil
	default:
		return IdentityMatrix(), fmt.Errorf("gltf: unknown camera type %q", c.Type)
	}
}
