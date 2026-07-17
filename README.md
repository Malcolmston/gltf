# gltf

A dependency-free [glTF 2.0](https://registry.khronos.org/glTF/specs/2.0/glTF-2.0.html)
and GLB toolkit for Go. Read, write, validate, and decode glTF assets in both
the JSON (`.gltf`) and binary (`.glb`) container formats using only the standard
library.

```
go get github.com/malcolmston/gltf
```

- Stdlib only — no third-party dependencies, no cgo.
- Complete glTF 2.0 document model with typed enums.
- `.gltf` JSON and `.glb` binary read/write.
- Accessor decoding (SCALAR/VEC/MAT over all component types) honoring
  `byteOffset`, `byteStride`, `normalized`, and sparse substitutions.
- Accessor **encoding** (write path): build bufferViews and accessors from typed
  `[][3]float32`, `[]uint32`, etc. with automatic min/max.
- Buffer resolution from GLB BIN chunks, base64 `data:` URIs, and external files.
- Structural validation with descriptive, path-qualified errors, covering
  accessor/bufferView bounds, component/type consistency, index ranges, and
  animation/camera/material/image rules.
- **Transform math**: TRS → matrix, matrix decompose, quaternion helpers, and
  world/global matrices up the node hierarchy.
- **Animation sampling**: evaluate a sampler at time `t` with STEP, LINEAR, and
  CUBICSPLINE interpolation (quaternion slerp for rotations).
- **Skinning**: joint matrices from a skin's inverse bind matrices and global
  joint transforms.
- **Morph targets**: decode target deltas and weighted-blend helpers.
- **Image decoding**: decode embedded or data-URI PNG/JPEG images to `image.Image`.
- **Khronos extensions** (typed structs, parse/encode, unknown-extension
  round-trip): `KHR_materials_unlit`, `KHR_materials_emissive_strength`,
  `KHR_materials_transmission`, `KHR_materials_ior`, `KHR_texture_transform`,
  `KHR_lights_punctual`, and `KHR_materials_pbrSpecularGlossiness`.

## Reading

```go
// From a .glb file (BIN chunk and external buffers are resolved automatically).
doc, err := gltf.OpenGLB("model.glb")
if err != nil {
    log.Fatal(err)
}

// Or from a .gltf file.
doc, err = gltf.Open("model.gltf")

// Decode the first mesh primitive's positions.
prim := doc.Meshes[0].Primitives[0]
positions, err := doc.DecodeAccessorVec3(prim.Attributes["POSITION"])
indices, err := doc.DecodeIndices(&prim)
```

`Open` and `OpenGLB` resolve buffer data for you. When you decode from an
in-memory document, call `doc.ResolveBuffers(baseDir, binChunk)` first.

## Writing

```go
// Build a minimal triangle and its binary buffer.
doc, bin := gltf.Triangle()

// Write a binary GLB.
gltf.SaveGLB("triangle.glb", doc, bin)

// Or a self-contained .gltf with the buffer embedded as a base64 data URI.
f, _ := os.Create("triangle.gltf")
defer f.Close()
gltf.WriteTriangleGLTF(f)
```

Lower-level streaming entry points are also available: `Decode`/`Encode` for
JSON, and `ReadGLB`/`WriteGLB` for the binary container.

## Accessors

Once buffers are resolved, typed decoders read vertex data out of accessors:

| Method | Returns |
| --- | --- |
| `DecodeAccessorFloat32(i)` | `[]float32` (flattened components) |
| `DecodeAccessorVec2/Vec3/Vec4(i)` | `[][2]float32` / `[][3]float32` / `[][4]float32` |
| `DecodeAccessorUint32(i)` | `[]uint32` (unsigned integer data) |
| `DecodeIndices(&prim)` | `[]uint32` index list, or `nil` if non-indexed |

All decoders honor `byteOffset`, `byteStride`, the `normalized` flag, and sparse
accessors.

## Validation

```go
if err := doc.Validate(); err != nil {
    if verrs, ok := gltf.AsValidationErrors(err); ok {
        for _, v := range verrs {
            fmt.Printf("%s: %s\n", v.Path, v.Message)
        }
    }
}
```

`Validate` checks required fields and index ranges across the whole document and
reports every problem with a path like `meshes[0].primitives[0].attributes.POSITION`.

## Scene evaluation

```go
// World transform of a node (walks the hierarchy to the root).
world, _ := doc.GlobalMatrix(nodeIndex)
p := world.TransformPoint(gltf.Vec3{0, 0, 0})

// Compose / decompose a local transform.
m := gltf.TRS(translation, rotation, scale)
t, r, s := m.Decompose()

// Sample an animation channel at time t (LINEAR/STEP/CUBICSPLINE, slerp for rotation).
path, values, _ := doc.SampleChannel(&doc.Animations[0], 0, 1.5)
_ = doc.ApplyAnimation(&doc.Animations[0], 1.5) // pose nodes in place

// Skinning: joint matrices = globalJointTransform * inverseBindMatrix.
joints, _ := doc.JointMatrices(skinIndex)

// Morph targets.
positions, _ := doc.MorphedPositions(&prim, []float64{0.5, 0.25})
```

## Building geometry (write path)

```go
doc := &gltf.Document{Asset: gltf.Asset{Version: "2.0"}}
pos := doc.AddAccessorVec3([][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}})
idx := doc.AddIndicesUint32([]uint32{0, 1, 2})
// pos and idx are accessor indices with bufferViews and min/max filled in.
```

## Extensions

```go
// Read a known extension.
if unlit := doc.Materials[0].Unlit(); unlit { /* ... */ }
ior, _ := doc.Materials[0].IOR()

// Write one (unknown extensions are preserved on round-trip).
doc.Materials[0].SetExtension(gltf.ExtMaterialsEmissiveStrength,
    gltf.MaterialsEmissiveStrength{EmissiveStrength: 3})

// Punctual lights (document-level array + node reference).
lights, _ := doc.Lights()
```

Supported: `KHR_materials_unlit`, `KHR_materials_emissive_strength`,
`KHR_materials_transmission`, `KHR_materials_ior`, `KHR_texture_transform`,
`KHR_lights_punctual`, `KHR_materials_pbrSpecularGlossiness`. Any other extension
is preserved verbatim through the raw `Extensions` field and the
`ExtensionMap`/`SetExtension` helpers.

## Documentation

Full API documentation is available via `go doc`:

```
go doc github.com/malcolmston/gltf
```

## License

See repository.
