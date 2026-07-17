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
- Buffer resolution from GLB BIN chunks, base64 `data:` URIs, and external files.
- Structural validation with descriptive, path-qualified errors.

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

## Documentation

Full API documentation is available via `go doc`:

```
go doc github.com/malcolmston/gltf
```

## License

See repository.
