package gltf

import (
	"fmt"
	"sort"
)

// GetInterpolation returns the sampler's interpolation algorithm, defaulting to
// [InterpolationLinear] when unset, as required by the specification.
func (s *AnimationSampler) GetInterpolation() Interpolation {
	if s.Interpolation == "" {
		return InterpolationLinear
	}
	return s.Interpolation
}

// SamplerKeyframes decodes an animation sampler's input (times) and output
// (values) accessors. times has one entry per keyframe. values is flattened
// with stride perKeyframe components; for CUBICSPLINE each keyframe occupies
// three consecutive strides (in-tangent, value, out-tangent). Buffers must be
// resolved first.
func (d *Document) SamplerKeyframes(s *AnimationSampler) (times []float32, values []float32, stride int, err error) {
	times, err = d.DecodeAccessorFloat32(s.Input)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("gltf: animation sampler input: %w", err)
	}
	values, err = d.DecodeAccessorFloat32(s.Output)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("gltf: animation sampler output: %w", err)
	}
	if len(times) == 0 {
		return nil, nil, 0, fmt.Errorf("gltf: animation sampler has no keyframes")
	}
	div := len(times)
	if s.GetInterpolation() == InterpolationCubicSpline {
		div *= 3
	}
	if div == 0 || len(values)%div != 0 {
		return nil, nil, 0, fmt.Errorf("gltf: animation sampler output length %d not divisible by keyframe count", len(values))
	}
	stride = len(values) / div
	return times, values, stride, nil
}

// EvaluateSampler evaluates the animation sampler at time t, returning the
// interpolated value as a slice of stride float32 components. Times outside the
// keyframe range are clamped to the first or last keyframe. When rotation is
// true (a rotation channel), LINEAR interpolation uses quaternion slerp and the
// CUBICSPLINE result is normalized, matching the glTF specification. Buffers
// must be resolved first.
func (d *Document) EvaluateSampler(s *AnimationSampler, t float64, rotation bool) ([]float32, error) {
	times, values, stride, err := d.SamplerKeyframes(s)
	if err != nil {
		return nil, err
	}
	n := len(times)
	interp := s.GetInterpolation()

	// Clamp before the first / after the last keyframe.
	if t <= float64(times[0]) {
		return keyframeValue(values, stride, interp, 0), nil
	}
	if t >= float64(times[n-1]) {
		return keyframeValue(values, stride, interp, n-1), nil
	}

	// Find the segment [i, i+1] containing t.
	i := sort.Search(n, func(k int) bool { return float64(times[k]) > t }) - 1
	if i < 0 {
		i = 0
	}
	if i >= n-1 {
		i = n - 2
	}
	t0 := float64(times[i])
	t1 := float64(times[i+1])
	dt := t1 - t0
	u := 0.0
	if dt > 0 {
		u = (t - t0) / dt
	}

	switch interp {
	case InterpolationStep:
		return keyframeValue(values, stride, interp, i), nil
	case InterpolationCubicSpline:
		return cubicSpline(values, stride, i, u, dt, rotation), nil
	default: // LINEAR
		v0 := keyframeValue(values, stride, interp, i)
		v1 := keyframeValue(values, stride, interp, i+1)
		if rotation && stride == 4 {
			q := Slerp(Quat{float64(v0[0]), float64(v0[1]), float64(v0[2]), float64(v0[3])},
				Quat{float64(v1[0]), float64(v1[1]), float64(v1[2]), float64(v1[3])}, u)
			return []float32{float32(q[0]), float32(q[1]), float32(q[2]), float32(q[3])}, nil
		}
		out := make([]float32, stride)
		for c := 0; c < stride; c++ {
			out[c] = v0[c] + float32(u)*(v1[c]-v0[c])
		}
		return out, nil
	}
}

// keyframeValue returns the value components of keyframe index i. For
// CUBICSPLINE each keyframe spans three strides and the middle (value) stride
// is returned.
func keyframeValue(values []float32, stride int, interp Interpolation, i int) []float32 {
	out := make([]float32, stride)
	if interp == InterpolationCubicSpline {
		base := i*3*stride + stride // skip in-tangent
		copy(out, values[base:base+stride])
		return out
	}
	copy(out, values[i*stride:i*stride+stride])
	return out
}

// cubicSpline evaluates the Hermite spline segment between keyframes i and i+1.
func cubicSpline(values []float32, stride, i int, u, dt float64, rotation bool) []float32 {
	// Layout per keyframe: [inTangent, value, outTangent], each of `stride`.
	p0 := values[i*3*stride+stride : i*3*stride+2*stride]
	m0 := values[i*3*stride+2*stride : i*3*stride+3*stride] // out-tangent of i
	p1 := values[(i+1)*3*stride+stride : (i+1)*3*stride+2*stride]
	m1 := values[(i+1)*3*stride : (i+1)*3*stride+stride] // in-tangent of i+1

	u2 := u * u
	u3 := u2 * u
	h00 := 2*u3 - 3*u2 + 1
	h10 := u3 - 2*u2 + u
	h01 := -2*u3 + 3*u2
	h11 := u3 - u2

	out := make([]float32, stride)
	for c := 0; c < stride; c++ {
		v := h00*float64(p0[c]) + h10*dt*float64(m0[c]) + h01*float64(p1[c]) + h11*dt*float64(m1[c])
		out[c] = float32(v)
	}
	if rotation && stride == 4 {
		q := Quat{float64(out[0]), float64(out[1]), float64(out[2]), float64(out[3])}.Normalize()
		out[0], out[1], out[2], out[3] = float32(q[0]), float32(q[1]), float32(q[2]), float32(q[3])
	}
	return out
}

// SampleChannel evaluates animation channel channelIndex of anim at time t. It
// returns the target property path and the interpolated value: three components
// for translation/scale, four (a quaternion) for rotation, and one per morph
// target for weights. Buffers must be resolved first.
func (d *Document) SampleChannel(anim *Animation, channelIndex int, t float64) (AnimationPath, []float32, error) {
	if channelIndex < 0 || channelIndex >= len(anim.Channels) {
		return "", nil, errIndexRange("animation channel", channelIndex, len(anim.Channels))
	}
	ch := &anim.Channels[channelIndex]
	if ch.Sampler < 0 || ch.Sampler >= len(anim.Samplers) {
		return "", nil, errIndexRange("animation sampler", ch.Sampler, len(anim.Samplers))
	}
	s := &anim.Samplers[ch.Sampler]
	rotation := ch.Target.Path == PathRotation
	vals, err := d.EvaluateSampler(s, t, rotation)
	if err != nil {
		return "", nil, err
	}
	return ch.Target.Path, vals, nil
}

// ApplyAnimation samples every channel of anim at time t and writes the results
// into the targeted nodes' Translation, Rotation, Scale, and Weights fields.
// Channels whose target node is nil are skipped. Buffers must be resolved
// first. It is a convenience for posing a document at a point in time.
func (d *Document) ApplyAnimation(anim *Animation, t float64) error {
	for i := range anim.Channels {
		ch := &anim.Channels[i]
		if ch.Target.Node == nil {
			continue
		}
		ni := *ch.Target.Node
		if ni < 0 || ni >= len(d.Nodes) {
			return errIndexRange("node", ni, len(d.Nodes))
		}
		path, vals, err := d.SampleChannel(anim, i, t)
		if err != nil {
			return err
		}
		nd := &d.Nodes[ni]
		switch path {
		case PathTranslation:
			if len(vals) >= 3 {
				nd.Translation = &[3]float64{float64(vals[0]), float64(vals[1]), float64(vals[2])}
			}
		case PathRotation:
			if len(vals) >= 4 {
				nd.Rotation = &[4]float64{float64(vals[0]), float64(vals[1]), float64(vals[2]), float64(vals[3])}
			}
		case PathScale:
			if len(vals) >= 3 {
				nd.Scale = &[3]float64{float64(vals[0]), float64(vals[1]), float64(vals[2])}
			}
		case PathWeights:
			w := make([]float64, len(vals))
			for k, v := range vals {
				w[k] = float64(v)
			}
			nd.Weights = w
		}
	}
	return nil
}
