package gltf

import (
	"errors"
	"fmt"
	"strings"
)

// ValidationError describes a single problem found by Validate. Path locates the
// offending element within the document (for example "meshes[0].primitives[1]").
type ValidationError struct {
	Path    string
	Message string
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	if e.Path == "" {
		return e.Message
	}
	return e.Path + ": " + e.Message
}

// ValidationErrors is a collection of validation problems. It is returned as a
// single error by Validate when the document is invalid.
type ValidationErrors []ValidationError

// Error implements the error interface, joining each problem on its own line.
func (errs ValidationErrors) Error() string {
	if len(errs) == 0 {
		return "gltf: no validation errors"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "gltf: %d validation error(s):", len(errs))
	for _, e := range errs {
		b.WriteString("\n  - ")
		b.WriteString(e.Error())
	}
	return b.String()
}

// Validate checks the document for required fields and index ranges, returning
// a ValidationErrors describing every problem found, or nil if the document is
// structurally valid. It does not require buffer data to be resolved.
func (d *Document) Validate() error {
	var errs ValidationErrors
	add := func(path, format string, args ...any) {
		errs = append(errs, ValidationError{Path: path, Message: fmt.Sprintf(format, args...)})
	}

	if strings.TrimSpace(d.Asset.Version) == "" {
		add("asset.version", "required field is empty")
	}

	// Every required extension must also be listed as used.
	used := make(map[string]bool, len(d.ExtensionsUsed))
	for _, e := range d.ExtensionsUsed {
		used[e] = true
	}
	for i, e := range d.ExtensionsRequired {
		if !used[e] {
			add(fmt.Sprintf("extensionsRequired[%d]", i), "required extension %q is not listed in extensionsUsed", e)
		}
	}

	checkIndex := func(path string, idx *Index, n int, what string) {
		if idx == nil {
			return
		}
		if *idx < 0 || *idx >= n {
			add(path, "%s index %d out of range [0,%d)", what, *idx, n)
		}
	}

	if d.Scene != nil && (*d.Scene < 0 || *d.Scene >= len(d.Scenes)) {
		add("scene", "scene index %d out of range [0,%d)", *d.Scene, len(d.Scenes))
	}

	for i := range d.Scenes {
		for j, n := range d.Scenes[i].Nodes {
			if n < 0 || n >= len(d.Nodes) {
				add(fmt.Sprintf("scenes[%d].nodes[%d]", i, j), "node index %d out of range [0,%d)", n, len(d.Nodes))
			}
		}
	}

	for i := range d.Nodes {
		nd := &d.Nodes[i]
		p := fmt.Sprintf("nodes[%d]", i)
		checkIndex(p+".mesh", nd.Mesh, len(d.Meshes), "mesh")
		checkIndex(p+".camera", nd.Camera, len(d.Cameras), "camera")
		checkIndex(p+".skin", nd.Skin, len(d.Skins), "skin")
		if nd.Matrix != nil && (nd.Translation != nil || nd.Rotation != nil || nd.Scale != nil) {
			add(p, "node must not specify matrix together with translation/rotation/scale")
		}
		for j, c := range nd.Children {
			if c < 0 || c >= len(d.Nodes) {
				add(fmt.Sprintf("%s.children[%d]", p, j), "node index %d out of range [0,%d)", c, len(d.Nodes))
			}
		}
	}

	for i := range d.Meshes {
		m := &d.Meshes[i]
		mp := fmt.Sprintf("meshes[%d]", i)
		if len(m.Primitives) == 0 {
			add(mp+".primitives", "mesh must have at least one primitive")
		}
		for j := range m.Primitives {
			prim := &m.Primitives[j]
			pp := fmt.Sprintf("%s.primitives[%d]", mp, j)
			if len(prim.Attributes) == 0 {
				add(pp+".attributes", "primitive must have at least one attribute")
			}
			for name, ai := range prim.Attributes {
				if ai < 0 || ai >= len(d.Accessors) {
					add(fmt.Sprintf("%s.attributes.%s", pp, name), "accessor index %d out of range [0,%d)", ai, len(d.Accessors))
				}
			}
			checkIndex(pp+".indices", prim.Indices, len(d.Accessors), "accessor")
			checkIndex(pp+".material", prim.Material, len(d.Materials), "material")
			if prim.Mode != nil && (*prim.Mode < PrimitivePoints || *prim.Mode > PrimitiveTriangleFan) {
				add(pp+".mode", "unknown primitive mode %d", *prim.Mode)
			}
			// An index accessor must be a scalar of an unsigned integer type.
			if prim.Indices != nil && *prim.Indices >= 0 && *prim.Indices < len(d.Accessors) {
				ia := &d.Accessors[*prim.Indices]
				if ia.Type != AccessorScalar {
					add(pp+".indices", "index accessor must be SCALAR, got %s", ia.Type)
				}
				switch ia.ComponentType {
				case ComponentUnsignedByte, ComponentUnsignedShort, ComponentUnsignedInt:
				default:
					add(pp+".indices", "index accessor must have an unsigned integer component type, got %s", ia.ComponentType)
				}
			}
			for ti, target := range prim.Targets {
				for name, ai := range target {
					if ai < 0 || ai >= len(d.Accessors) {
						add(fmt.Sprintf("%s.targets[%d].%s", pp, ti, name), "accessor index %d out of range [0,%d)", ai, len(d.Accessors))
					}
				}
			}
		}
	}

	for i := range d.Accessors {
		a := &d.Accessors[i]
		ap := fmt.Sprintf("accessors[%d]", i)
		if a.Type.ComponentCount() == 0 {
			add(ap+".type", "unknown accessor type %q", a.Type)
		}
		if a.ComponentType.SizeInBytes() == 0 {
			add(ap+".componentType", "unknown component type %d", a.ComponentType)
		}
		if a.Count <= 0 {
			add(ap+".count", "count must be positive, got %d", a.Count)
		}
		checkIndex(ap+".bufferView", a.BufferView, len(d.BufferViews), "bufferView")
		// normalized is not allowed for FLOAT or UNSIGNED_INT component types.
		if a.Normalized && (a.ComponentType == ComponentFloat || a.ComponentType == ComponentUnsignedInt) {
			add(ap+".normalized", "normalized must not be set for component type %s", a.ComponentType)
		}
		// min/max, when present, must match the component count.
		if cc := a.Type.ComponentCount(); cc > 0 {
			if len(a.Min) != 0 && len(a.Min) != cc {
				add(ap+".min", "min has %d components, want %d", len(a.Min), cc)
			}
			if len(a.Max) != 0 && len(a.Max) != cc {
				add(ap+".max", "max has %d components, want %d", len(a.Max), cc)
			}
		}
		// The accessor's elements must fit inside the referenced bufferView.
		if a.BufferView != nil && *a.BufferView >= 0 && *a.BufferView < len(d.BufferViews) {
			bv := &d.BufferViews[*a.BufferView]
			cc := a.Type.ComponentCount()
			cs := a.ComponentType.SizeInBytes()
			if cc > 0 && cs > 0 && a.Count > 0 {
				elem := cc * cs
				stride := bv.ByteStride
				if stride == 0 {
					stride = elem
				}
				need := a.ByteOffset + (a.Count-1)*stride + elem
				if need > bv.ByteLength {
					add(ap, "accessor requires %d bytes but bufferView %d is only %d bytes", need, *a.BufferView, bv.ByteLength)
				}
			}
		}
		if a.Sparse != nil {
			s := a.Sparse
			if s.Count <= 0 {
				add(ap+".sparse.count", "sparse count must be positive, got %d", s.Count)
			}
			if s.Count > a.Count {
				add(ap+".sparse.count", "sparse count %d exceeds accessor count %d", s.Count, a.Count)
			}
			switch s.Indices.ComponentType {
			case ComponentUnsignedByte, ComponentUnsignedShort, ComponentUnsignedInt:
			default:
				add(ap+".sparse.indices.componentType", "must be an unsigned integer type, got %s", s.Indices.ComponentType)
			}
			if s.Indices.BufferView < 0 || s.Indices.BufferView >= len(d.BufferViews) {
				add(ap+".sparse.indices.bufferView", "bufferView index %d out of range [0,%d)", s.Indices.BufferView, len(d.BufferViews))
			}
			if s.Values.BufferView < 0 || s.Values.BufferView >= len(d.BufferViews) {
				add(ap+".sparse.values.bufferView", "bufferView index %d out of range [0,%d)", s.Values.BufferView, len(d.BufferViews))
			}
			d.validateSparseIndices(a, ap, add)
		}
	}

	for i := range d.BufferViews {
		bv := &d.BufferViews[i]
		bp := fmt.Sprintf("bufferViews[%d]", i)
		if bv.Buffer < 0 || bv.Buffer >= len(d.Buffers) {
			add(bp+".buffer", "buffer index %d out of range [0,%d)", bv.Buffer, len(d.Buffers))
		} else if bv.ByteLength > 0 {
			// The view must lie within its buffer's declared byte length.
			end := bv.ByteOffset + bv.ByteLength
			if bv.ByteOffset < 0 || end > d.Buffers[bv.Buffer].ByteLength {
				add(bp, "view [%d,%d) exceeds buffer %d length %d", bv.ByteOffset, end, bv.Buffer, d.Buffers[bv.Buffer].ByteLength)
			}
		}
		if bv.ByteLength <= 0 {
			add(bp+".byteLength", "byteLength must be positive, got %d", bv.ByteLength)
		}
		// byteStride, when present, must be a multiple of 4 in [4,252].
		if bv.ByteStride != 0 && (bv.ByteStride < 4 || bv.ByteStride > 252 || bv.ByteStride%4 != 0) {
			add(bp+".byteStride", "byteStride %d must be a multiple of 4 in [4,252]", bv.ByteStride)
		}
	}

	for i := range d.Buffers {
		if d.Buffers[i].ByteLength <= 0 {
			add(fmt.Sprintf("buffers[%d].byteLength", i), "byteLength must be positive, got %d", d.Buffers[i].ByteLength)
		}
	}

	for i := range d.Textures {
		t := &d.Textures[i]
		tp := fmt.Sprintf("textures[%d]", i)
		checkIndex(tp+".sampler", t.Sampler, len(d.Samplers), "sampler")
		checkIndex(tp+".source", t.Source, len(d.Images), "image")
	}

	for i := range d.Images {
		img := &d.Images[i]
		ip := fmt.Sprintf("images[%d]", i)
		checkIndex(ip+".bufferView", img.BufferView, len(d.BufferViews), "bufferView")
		switch {
		case img.URI == "" && img.BufferView == nil:
			add(ip, "image must specify either uri or bufferView")
		case img.URI != "" && img.BufferView != nil:
			add(ip, "image must not specify both uri and bufferView")
		}
		if img.BufferView != nil && img.MimeType == "" {
			add(ip+".mimeType", "mimeType is required for a bufferView-backed image")
		}
	}

	for i := range d.Skins {
		sk := &d.Skins[i]
		sp := fmt.Sprintf("skins[%d]", i)
		if len(sk.Joints) == 0 {
			add(sp+".joints", "skin must have at least one joint")
		}
		checkIndex(sp+".inverseBindMatrices", sk.InverseBindMatrices, len(d.Accessors), "accessor")
		checkIndex(sp+".skeleton", sk.Skeleton, len(d.Nodes), "node")
		for j, joint := range sk.Joints {
			if joint < 0 || joint >= len(d.Nodes) {
				add(fmt.Sprintf("%s.joints[%d]", sp, j), "node index %d out of range [0,%d)", joint, len(d.Nodes))
			}
		}
		// The inverse bind matrices accessor must be MAT4 with one entry per
		// joint.
		if sk.InverseBindMatrices != nil && *sk.InverseBindMatrices >= 0 && *sk.InverseBindMatrices < len(d.Accessors) {
			ibm := &d.Accessors[*sk.InverseBindMatrices]
			if ibm.Type != AccessorMat4 {
				add(sp+".inverseBindMatrices", "accessor must be MAT4, got %s", ibm.Type)
			}
			if ibm.Count < len(sk.Joints) {
				add(sp+".inverseBindMatrices", "accessor has %d matrices for %d joints", ibm.Count, len(sk.Joints))
			}
		}
	}

	for i := range d.Animations {
		an := &d.Animations[i]
		anp := fmt.Sprintf("animations[%d]", i)
		for j := range an.Channels {
			ch := &an.Channels[j]
			cp := fmt.Sprintf("%s.channels[%d]", anp, j)
			if ch.Sampler < 0 || ch.Sampler >= len(an.Samplers) {
				add(cp+".sampler", "sampler index %d out of range [0,%d)", ch.Sampler, len(an.Samplers))
			}
			checkIndex(cp+".target.node", ch.Target.Node, len(d.Nodes), "node")
		}
		for j := range an.Samplers {
			s := &an.Samplers[j]
			sp := fmt.Sprintf("%s.samplers[%d]", anp, j)
			inOK := s.Input >= 0 && s.Input < len(d.Accessors)
			outOK := s.Output >= 0 && s.Output < len(d.Accessors)
			if !inOK {
				add(sp+".input", "accessor index %d out of range [0,%d)", s.Input, len(d.Accessors))
			}
			if !outOK {
				add(sp+".output", "accessor index %d out of range [0,%d)", s.Output, len(d.Accessors))
			}
			switch s.Interpolation {
			case "", InterpolationLinear, InterpolationStep, InterpolationCubicSpline:
			default:
				add(sp+".interpolation", "unknown interpolation %q", s.Interpolation)
			}
			// The input accessor must be a scalar float keyframe-time buffer.
			if inOK {
				in := &d.Accessors[s.Input]
				if in.Type != AccessorScalar || in.ComponentType != ComponentFloat {
					add(sp+".input", "input accessor must be SCALAR FLOAT, got %s %s", in.Type, in.ComponentType)
				}
				// The output count must be a multiple of the keyframe count
				// (three times for CUBICSPLINE, which stores tangents).
				if outOK && in.Count > 0 {
					factor := in.Count
					if s.GetInterpolation() == InterpolationCubicSpline {
						factor *= 3
					}
					if factor > 0 && d.Accessors[s.Output].Count%factor != 0 {
						add(sp+".output", "output count %d is not a multiple of keyframe count %d", d.Accessors[s.Output].Count, factor)
					}
				}
			}
		}
	}

	for i := range d.Materials {
		m := &d.Materials[i]
		mp := fmt.Sprintf("materials[%d]", i)
		switch m.AlphaMode {
		case "", AlphaOpaque, AlphaMask, AlphaBlend:
		default:
			add(mp+".alphaMode", "unknown alpha mode %q", m.AlphaMode)
		}
		if m.PBRMetallicRoughness != nil {
			pbr := m.PBRMetallicRoughness
			if pbr.BaseColorTexture != nil {
				checkTextureRef(add, mp+".pbrMetallicRoughness.baseColorTexture", pbr.BaseColorTexture.Index, len(d.Textures))
			}
			if pbr.MetallicRoughnessTexture != nil {
				checkTextureRef(add, mp+".pbrMetallicRoughness.metallicRoughnessTexture", pbr.MetallicRoughnessTexture.Index, len(d.Textures))
			}
		}
		if m.NormalTexture != nil {
			checkTextureRef(add, mp+".normalTexture", m.NormalTexture.Index, len(d.Textures))
		}
		if m.OcclusionTexture != nil {
			checkTextureRef(add, mp+".occlusionTexture", m.OcclusionTexture.Index, len(d.Textures))
		}
		if m.EmissiveTexture != nil {
			checkTextureRef(add, mp+".emissiveTexture", m.EmissiveTexture.Index, len(d.Textures))
		}
	}

	for i := range d.Cameras {
		c := &d.Cameras[i]
		cp := fmt.Sprintf("cameras[%d]", i)
		switch c.Type {
		case CameraTypePerspective:
			if c.Perspective == nil {
				add(cp, "perspective camera missing perspective properties")
			} else {
				pc := c.Perspective
				if pc.YFOV <= 0 {
					add(cp+".perspective.yfov", "yfov must be positive, got %g", pc.YFOV)
				}
				if pc.ZNear <= 0 {
					add(cp+".perspective.znear", "znear must be positive, got %g", pc.ZNear)
				}
				if pc.ZFar != nil && *pc.ZFar <= pc.ZNear {
					add(cp+".perspective.zfar", "zfar %g must be greater than znear %g", *pc.ZFar, pc.ZNear)
				}
				if pc.AspectRatio != nil && *pc.AspectRatio <= 0 {
					add(cp+".perspective.aspectRatio", "aspectRatio must be positive, got %g", *pc.AspectRatio)
				}
			}
		case CameraTypeOrthographic:
			if c.Orthographic == nil {
				add(cp, "orthographic camera missing orthographic properties")
			} else {
				oc := c.Orthographic
				if oc.XMag == 0 || oc.YMag == 0 {
					add(cp+".orthographic", "xmag and ymag must be non-zero")
				}
				if oc.ZFar <= oc.ZNear {
					add(cp+".orthographic.zfar", "zfar %g must be greater than znear %g", oc.ZFar, oc.ZNear)
				}
			}
		default:
			add(cp+".type", "unknown camera type %q", c.Type)
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

// validateSparseIndices enforces the runtime rules the glTF 2.0 specification
// places on a sparse accessor's index array: the decoded indices MUST form a
// strictly increasing sequence and none of them MUST be greater than or equal
// to the base accessor's element count. Both rules require the index data
// itself, so the check runs only when the referenced bufferView is in range and
// its backing buffer has been resolved (via ResolveBuffers, Open, or OpenGLB);
// otherwise it is skipped so that structural validation still works on
// documents whose buffers have not been loaded.
func (d *Document) validateSparseIndices(a *Accessor, ap string, add func(string, string, ...any)) {
	s := a.Sparse
	if s.Count <= 0 {
		return
	}
	if s.Indices.BufferView < 0 || s.Indices.BufferView >= len(d.BufferViews) {
		return
	}
	idxSize := s.Indices.ComponentType.SizeInBytes()
	if idxSize == 0 {
		return
	}
	_, idxBuf, err := d.resolveBufferView(s.Indices.BufferView)
	if err != nil {
		// Buffers not resolved (or out of range already reported); the
		// strictly-increasing/in-range rules cannot be checked without data.
		return
	}
	idxBV := &d.BufferViews[s.Indices.BufferView]
	idxBase := idxBV.ByteOffset + s.Indices.ByteOffset

	prev := int64(-1)
	for j := 0; j < s.Count; j++ {
		off := idxBase + j*idxSize
		if off < 0 || off+idxSize > len(idxBuf) {
			// Out-of-bounds reads are reported elsewhere; stop here.
			return
		}
		v, err := componentToUint64(idxBuf[off:], s.Indices.ComponentType)
		if err != nil {
			return
		}
		cur := int64(v)
		if cur <= prev {
			add(fmt.Sprintf("%s.sparse.indices[%d]", ap, j),
				"sparse indices must be strictly increasing, but %d follows %d", cur, prev)
		}
		if a.Count >= 0 && cur >= int64(a.Count) {
			add(fmt.Sprintf("%s.sparse.indices[%d]", ap, j),
				"sparse index %d is not less than accessor count %d", cur, a.Count)
		}
		prev = cur
	}
}

// checkTextureRef validates a texture index used by a material.
func checkTextureRef(add func(string, string, ...any), path string, idx, n int) {
	if idx < 0 || idx >= n {
		add(path, "texture index %d out of range [0,%d)", idx, n)
	}
}

// AsValidationErrors returns the underlying ValidationErrors when err was
// produced by Validate, and reports whether the conversion succeeded.
func AsValidationErrors(err error) (ValidationErrors, bool) {
	var v ValidationErrors
	if errors.As(err, &v) {
		return v, true
	}
	return nil, false
}
