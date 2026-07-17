package gltf

import (
	"encoding/binary"
	"fmt"
	"math"
)

// le is the little-endian byte order used throughout glTF binary data.
var le = binary.LittleEndian

// resolveBufferView returns the bufferView at index and the resolved bytes of
// its backing buffer. ResolveBuffers (or a GLB/gltf Open) must have run first.
func (d *Document) resolveBufferView(index int) (*BufferView, []byte, error) {
	if index < 0 || index >= len(d.BufferViews) {
		return nil, nil, fmt.Errorf("gltf: bufferView index %d out of range [0,%d)", index, len(d.BufferViews))
	}
	bv := &d.BufferViews[index]
	if bv.Buffer < 0 || bv.Buffer >= len(d.Buffers) {
		return nil, nil, fmt.Errorf("gltf: bufferView %d references buffer %d out of range [0,%d)", index, bv.Buffer, len(d.Buffers))
	}
	buf := &d.Buffers[bv.Buffer]
	if buf.Data == nil {
		return nil, nil, fmt.Errorf("gltf: buffer %d not resolved; call ResolveBuffers first", bv.Buffer)
	}
	end := bv.ByteOffset + bv.ByteLength
	if bv.ByteOffset < 0 || end > len(buf.Data) {
		return nil, nil, fmt.Errorf("gltf: bufferView %d [%d,%d) exceeds buffer %d length %d", index, bv.ByteOffset, end, bv.Buffer, len(buf.Data))
	}
	return bv, buf.Data, nil
}

// accessorBytes returns the accessor's elements as a densely packed byte slice
// (count * componentCount * componentSize bytes), with byteStride removed and
// any sparse substitutions applied. Components remain in their stored
// componentType; use the typed Decode* helpers to convert them.
func (d *Document) accessorBytes(a *Accessor) ([]byte, error) {
	compCount := a.Type.ComponentCount()
	if compCount == 0 {
		return nil, fmt.Errorf("gltf: unknown accessor type %q", a.Type)
	}
	compSize := a.ComponentType.SizeInBytes()
	if compSize == 0 {
		return nil, fmt.Errorf("gltf: unknown component type %d", a.ComponentType)
	}
	if a.Count < 0 {
		return nil, fmt.Errorf("gltf: negative accessor count %d", a.Count)
	}
	elemSize := compCount * compSize
	packed := make([]byte, a.Count*elemSize)

	if a.BufferView != nil {
		bv, buf, err := d.resolveBufferView(*a.BufferView)
		if err != nil {
			return nil, err
		}
		stride := bv.ByteStride
		if stride == 0 {
			stride = elemSize
		}
		base := bv.ByteOffset + a.ByteOffset
		for i := 0; i < a.Count; i++ {
			src := base + i*stride
			if src < 0 || src+elemSize > len(buf) {
				return nil, fmt.Errorf("gltf: accessor element %d reads [%d,%d) beyond buffer length %d", i, src, src+elemSize, len(buf))
			}
			copy(packed[i*elemSize:(i+1)*elemSize], buf[src:src+elemSize])
		}
	}

	if a.Sparse != nil {
		if err := d.applySparse(a, packed, elemSize); err != nil {
			return nil, err
		}
	}
	return packed, nil
}

// applySparse overwrites the packed elements named by the sparse index buffer
// with the values from the sparse value buffer.
func (d *Document) applySparse(a *Accessor, packed []byte, elemSize int) error {
	s := a.Sparse
	if s.Count < 0 {
		return fmt.Errorf("gltf: negative sparse count %d", s.Count)
	}
	idxSize := s.Indices.ComponentType.SizeInBytes()
	if idxSize == 0 {
		return fmt.Errorf("gltf: unknown sparse index component type %d", s.Indices.ComponentType)
	}

	_, idxBuf, err := d.resolveBufferView(s.Indices.BufferView)
	if err != nil {
		return fmt.Errorf("gltf: sparse indices: %w", err)
	}
	idxBV := &d.BufferViews[s.Indices.BufferView]
	idxBase := idxBV.ByteOffset + s.Indices.ByteOffset

	_, valBuf, err := d.resolveBufferView(s.Values.BufferView)
	if err != nil {
		return fmt.Errorf("gltf: sparse values: %w", err)
	}
	valBV := &d.BufferViews[s.Values.BufferView]
	valBase := valBV.ByteOffset + s.Values.ByteOffset

	for j := 0; j < s.Count; j++ {
		idxOff := idxBase + j*idxSize
		if idxOff < 0 || idxOff+idxSize > len(idxBuf) {
			return fmt.Errorf("gltf: sparse index %d out of buffer range", j)
		}
		target, err := componentToUint64(idxBuf[idxOff:], s.Indices.ComponentType)
		if err != nil {
			return err
		}
		if int(target) >= a.Count {
			return fmt.Errorf("gltf: sparse index %d targets accessor element %d beyond count %d", j, target, a.Count)
		}
		valOff := valBase + j*elemSize
		if valOff < 0 || valOff+elemSize > len(valBuf) {
			return fmt.Errorf("gltf: sparse value %d out of buffer range", j)
		}
		dst := int(target) * elemSize
		copy(packed[dst:dst+elemSize], valBuf[valOff:valOff+elemSize])
	}
	return nil
}

// componentToFloat reads one component from the start of b, converting it to a
// float64. When normalized is true, integer components are mapped to the [0,1]
// or [-1,1] range as defined by the glTF specification.
func componentToFloat(b []byte, ct ComponentType, normalized bool) float64 {
	switch ct {
	case ComponentFloat:
		return float64(math.Float32frombits(le.Uint32(b)))
	case ComponentByte:
		v := int8(b[0])
		if normalized {
			return math.Max(float64(v)/127.0, -1.0)
		}
		return float64(v)
	case ComponentUnsignedByte:
		v := b[0]
		if normalized {
			return float64(v) / 255.0
		}
		return float64(v)
	case ComponentShort:
		v := int16(le.Uint16(b))
		if normalized {
			return math.Max(float64(v)/32767.0, -1.0)
		}
		return float64(v)
	case ComponentUnsignedShort:
		v := le.Uint16(b)
		if normalized {
			return float64(v) / 65535.0
		}
		return float64(v)
	case ComponentUnsignedInt:
		return float64(le.Uint32(b))
	default:
		return 0
	}
}

// componentToUint64 reads one unsigned-integer component from the start of b.
// It returns an error for non-integer component types.
func componentToUint64(b []byte, ct ComponentType) (uint64, error) {
	switch ct {
	case ComponentUnsignedByte:
		return uint64(b[0]), nil
	case ComponentUnsignedShort:
		return uint64(le.Uint16(b)), nil
	case ComponentUnsignedInt:
		return uint64(le.Uint32(b)), nil
	default:
		return 0, fmt.Errorf("gltf: component type %s is not an unsigned integer", ct)
	}
}

// DecodeAccessorFloat32 decodes accessor at index into a flat slice of
// float32 values with length Count*ComponentCount, applying normalization when
// the accessor's Normalized flag is set. It works for any component type.
func (d *Document) DecodeAccessorFloat32(index int) ([]float32, error) {
	a, err := d.accessorAt(index)
	if err != nil {
		return nil, err
	}
	packed, err := d.accessorBytes(a)
	if err != nil {
		return nil, err
	}
	compSize := a.ComponentType.SizeInBytes()
	n := a.Count * a.Type.ComponentCount()
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		out[i] = float32(componentToFloat(packed[i*compSize:], a.ComponentType, a.Normalized))
	}
	return out, nil
}

// DecodeAccessorVec2 decodes a VEC2 accessor into a slice of 2-component
// float32 vectors.
func (d *Document) DecodeAccessorVec2(index int) ([][2]float32, error) {
	flat, err := d.decodeVecN(index, AccessorVec2, 2)
	if err != nil {
		return nil, err
	}
	out := make([][2]float32, len(flat)/2)
	for i := range out {
		copy(out[i][:], flat[i*2:])
	}
	return out, nil
}

// DecodeAccessorVec3 decodes a VEC3 accessor into a slice of 3-component
// float32 vectors (for example vertex positions or normals).
func (d *Document) DecodeAccessorVec3(index int) ([][3]float32, error) {
	flat, err := d.decodeVecN(index, AccessorVec3, 3)
	if err != nil {
		return nil, err
	}
	out := make([][3]float32, len(flat)/3)
	for i := range out {
		copy(out[i][:], flat[i*3:])
	}
	return out, nil
}

// DecodeAccessorVec4 decodes a VEC4 accessor into a slice of 4-component
// float32 vectors.
func (d *Document) DecodeAccessorVec4(index int) ([][4]float32, error) {
	flat, err := d.decodeVecN(index, AccessorVec4, 4)
	if err != nil {
		return nil, err
	}
	out := make([][4]float32, len(flat)/4)
	for i := range out {
		copy(out[i][:], flat[i*4:])
	}
	return out, nil
}

// decodeVecN validates the accessor type and returns its flattened float32
// components.
func (d *Document) decodeVecN(index int, want AccessorType, n int) ([]float32, error) {
	a, err := d.accessorAt(index)
	if err != nil {
		return nil, err
	}
	if a.Type != want {
		return nil, fmt.Errorf("gltf: accessor %d is %s, want %s", index, a.Type, want)
	}
	return d.DecodeAccessorFloat32(index)
}

// DecodeAccessorUint32 decodes an accessor with an unsigned-integer component
// type into a flat slice of uint32 values (length Count*ComponentCount). It is
// the correct decoder for index accessors (SCALAR UNSIGNED_BYTE/SHORT/INT).
func (d *Document) DecodeAccessorUint32(index int) ([]uint32, error) {
	a, err := d.accessorAt(index)
	if err != nil {
		return nil, err
	}
	packed, err := d.accessorBytes(a)
	if err != nil {
		return nil, err
	}
	compSize := a.ComponentType.SizeInBytes()
	n := a.Count * a.Type.ComponentCount()
	out := make([]uint32, n)
	for i := 0; i < n; i++ {
		v, err := componentToUint64(packed[i*compSize:], a.ComponentType)
		if err != nil {
			return nil, err
		}
		out[i] = uint32(v)
	}
	return out, nil
}

// DecodeIndices decodes the primitive's index accessor into a flat slice of
// uint32 vertex indices. It returns nil, nil for a non-indexed primitive.
func (d *Document) DecodeIndices(p *Primitive) ([]uint32, error) {
	if p.Indices == nil {
		return nil, nil
	}
	return d.DecodeAccessorUint32(*p.Indices)
}

// accessorAt returns the accessor at index, bounds-checked.
func (d *Document) accessorAt(index int) (*Accessor, error) {
	if index < 0 || index >= len(d.Accessors) {
		return nil, fmt.Errorf("gltf: accessor index %d out of range [0,%d)", index, len(d.Accessors))
	}
	return &d.Accessors[index], nil
}
