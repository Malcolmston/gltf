package gltf

import "math"

// ensureBuffer returns the index of the buffer used for generated accessor
// data, creating buffer 0 with an empty, ready-to-append Data slice if the
// document has no buffers yet.
func (d *Document) ensureBuffer() int {
	if len(d.Buffers) == 0 {
		d.Buffers = append(d.Buffers, Buffer{Data: []byte{}})
	}
	if d.Buffers[0].Data == nil {
		d.Buffers[0].Data = []byte{}
	}
	return 0
}

// addBufferView appends data to buffer 0 (four-byte aligned) and adds a
// bufferView covering it, returning the new bufferView index. The buffer's
// ByteLength is kept in sync with its Data.
func (d *Document) addBufferView(data []byte, stride int, target *TargetType) int {
	bi := d.ensureBuffer()
	buf := &d.Buffers[bi]
	// Align the start of the new view to four bytes.
	for len(buf.Data)%4 != 0 {
		buf.Data = append(buf.Data, 0)
	}
	offset := len(buf.Data)
	buf.Data = append(buf.Data, data...)
	buf.ByteLength = len(buf.Data)

	bv := BufferView{
		Buffer:     bi,
		ByteOffset: offset,
		ByteLength: len(data),
		Target:     target,
	}
	if stride > 0 {
		bv.ByteStride = stride
	}
	d.BufferViews = append(d.BufferViews, bv)
	return len(d.BufferViews) - 1
}

// AddBinData appends raw bytes (for example an encoded PNG or JPEG image) to
// buffer 0 and adds a bufferView covering them, returning the new bufferView
// index. It is the write-path counterpart to a bufferView-backed [Image].
func (d *Document) AddBinData(data []byte) int {
	return d.addBufferView(data, 0, nil)
}

// AddAccessorFloat32 appends flattened float32 data as a new accessor of the
// given element type to buffer 0, computing per-component min/max, and returns
// the new accessor index. len(data) must be a multiple of the type's component
// count. The bytes are stored little-endian and, when the document has no
// resolved buffer 0, one is created so the data can be decoded immediately.
func (d *Document) AddAccessorFloat32(data []float32, typ AccessorType) int {
	cc := typ.ComponentCount()
	if cc == 0 || len(data)%cc != 0 {
		cc = 1
	}
	count := len(data) / cc

	b := make([]byte, 0, len(data)*4)
	for _, f := range data {
		b = le.AppendUint32(b, math.Float32bits(f))
	}
	bv := d.addBufferView(b, 0, targetPtr(TargetArrayBuffer))

	minv := make([]float64, cc)
	maxv := make([]float64, cc)
	for c := 0; c < cc; c++ {
		minv[c] = math.Inf(1)
		maxv[c] = math.Inf(-1)
	}
	for i := 0; i < count; i++ {
		for c := 0; c < cc; c++ {
			v := float64(data[i*cc+c])
			minv[c] = math.Min(minv[c], v)
			maxv[c] = math.Max(maxv[c], v)
		}
	}
	if count == 0 {
		minv, maxv = nil, nil
	}

	d.Accessors = append(d.Accessors, Accessor{
		BufferView:    intPtr(bv),
		ComponentType: ComponentFloat,
		Count:         count,
		Type:          typ,
		Min:           minv,
		Max:           maxv,
	})
	return len(d.Accessors) - 1
}

// AddAccessorVec2 appends VEC2 float data as a new accessor and returns its
// index. See [Document.AddAccessorFloat32].
func (d *Document) AddAccessorVec2(data [][2]float32) int {
	flat := make([]float32, 0, len(data)*2)
	for _, v := range data {
		flat = append(flat, v[0], v[1])
	}
	return d.AddAccessorFloat32(flat, AccessorVec2)
}

// AddAccessorVec3 appends VEC3 float data (for example positions or normals) as
// a new accessor and returns its index. See [Document.AddAccessorFloat32].
func (d *Document) AddAccessorVec3(data [][3]float32) int {
	flat := make([]float32, 0, len(data)*3)
	for _, v := range data {
		flat = append(flat, v[0], v[1], v[2])
	}
	return d.AddAccessorFloat32(flat, AccessorVec3)
}

// AddAccessorVec4 appends VEC4 float data as a new accessor and returns its
// index. See [Document.AddAccessorFloat32].
func (d *Document) AddAccessorVec4(data [][4]float32) int {
	flat := make([]float32, 0, len(data)*4)
	for _, v := range data {
		flat = append(flat, v[0], v[1], v[2], v[3])
	}
	return d.AddAccessorFloat32(flat, AccessorVec4)
}

// AddIndicesUint32 appends unsigned 32-bit index data as a new SCALAR accessor
// bound to an element-array bufferView, and returns the new accessor index. It
// is the write-path inverse of [Document.DecodeIndices].
func (d *Document) AddIndicesUint32(data []uint32) int {
	b := make([]byte, 0, len(data)*4)
	minv, maxv := math.Inf(1), math.Inf(-1)
	for _, v := range data {
		b = le.AppendUint32(b, v)
		minv = math.Min(minv, float64(v))
		maxv = math.Max(maxv, float64(v))
	}
	bv := d.addBufferView(b, 0, targetPtr(TargetElementArrayBuffer))
	acc := Accessor{
		BufferView:    intPtr(bv),
		ComponentType: ComponentUnsignedInt,
		Count:         len(data),
		Type:          AccessorScalar,
	}
	if len(data) > 0 {
		acc.Min = []float64{minv}
		acc.Max = []float64{maxv}
	}
	d.Accessors = append(d.Accessors, acc)
	return len(d.Accessors) - 1
}

// targetPtr returns a pointer to the given bufferView target.
func targetPtr(t TargetType) *TargetType {
	return &t
}
