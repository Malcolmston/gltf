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
		if a.Sparse != nil {
			s := a.Sparse
			if s.Count <= 0 {
				add(ap+".sparse.count", "sparse count must be positive, got %d", s.Count)
			}
			if s.Indices.BufferView < 0 || s.Indices.BufferView >= len(d.BufferViews) {
				add(ap+".sparse.indices.bufferView", "bufferView index %d out of range [0,%d)", s.Indices.BufferView, len(d.BufferViews))
			}
			if s.Values.BufferView < 0 || s.Values.BufferView >= len(d.BufferViews) {
				add(ap+".sparse.values.bufferView", "bufferView index %d out of range [0,%d)", s.Values.BufferView, len(d.BufferViews))
			}
		}
	}

	for i := range d.BufferViews {
		bv := &d.BufferViews[i]
		bp := fmt.Sprintf("bufferViews[%d]", i)
		if bv.Buffer < 0 || bv.Buffer >= len(d.Buffers) {
			add(bp+".buffer", "buffer index %d out of range [0,%d)", bv.Buffer, len(d.Buffers))
		}
		if bv.ByteLength <= 0 {
			add(bp+".byteLength", "byteLength must be positive, got %d", bv.ByteLength)
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
		checkIndex(fmt.Sprintf("images[%d].bufferView", i), img.BufferView, len(d.BufferViews), "bufferView")
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
			if s.Input < 0 || s.Input >= len(d.Accessors) {
				add(sp+".input", "accessor index %d out of range [0,%d)", s.Input, len(d.Accessors))
			}
			if s.Output < 0 || s.Output >= len(d.Accessors) {
				add(sp+".output", "accessor index %d out of range [0,%d)", s.Output, len(d.Accessors))
			}
		}
	}

	for i := range d.Materials {
		m := &d.Materials[i]
		mp := fmt.Sprintf("materials[%d]", i)
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
			}
		case CameraTypeOrthographic:
			if c.Orthographic == nil {
				add(cp, "orthographic camera missing orthographic properties")
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
