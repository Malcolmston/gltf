// Package gltf is a dependency-free toolkit for reading, writing, and inspecting
// glTF 2.0 assets in both the JSON (.gltf) and binary (.glb) container formats.
//
// # Document model
//
// [Document] is the root object and mirrors the glTF 2.0 JSON schema. Every
// top-level collection (scenes, nodes, meshes, accessors, buffer views,
// buffers, materials, textures, images, samplers, animations, skins, and
// cameras) is a slice addressed by integer index from elsewhere in the
// document, exactly as in the specification. Unknown or vendor-specific data is
// preserved through the Extensions and Extras fields, which are typed as
// [encoding/json.RawMessage].
//
// Typed enumerations ([ComponentType], [AccessorType], [PrimitiveMode],
// [Filter], [WrapMode], [Interpolation], [CameraType], [AlphaMode], and others)
// carry the spec-defined constant values and helper methods.
//
// # Reading and writing
//
// For JSON documents use [Decode] and [Encode], or the file helpers [Open] and
// [Save]. For the binary container use [ReadGLB] and [WriteGLB], or the file
// helpers [OpenGLB] and [SaveGLB]. [Open] and [OpenGLB] additionally resolve
// buffer data.
//
// # Buffers
//
// Buffer payloads come from three sources: an embedded GLB BIN chunk, a base64
// "data:" URI, or an external file. [Document.ResolveBuffers] loads all three
// into each [Buffer]'s Data field; [EncodeDataURI] performs the reverse for
// embedding.
//
// # Accessors
//
// Once buffers are resolved, the Decode* methods on [Document] read typed
// vertex data out of accessors, honoring byteOffset, byteStride, the normalized
// flag, and sparse substitutions:
//
//   - [Document.DecodeAccessorFloat32] returns flattened float32 components;
//   - [Document.DecodeAccessorVec2], [Document.DecodeAccessorVec3], and
//     [Document.DecodeAccessorVec4] return typed vectors;
//   - [Document.DecodeAccessorUint32] and [Document.DecodeIndices] return
//     unsigned integer index data.
//
// The Add* methods provide the write path: [Document.AddAccessorVec3],
// [Document.AddAccessorVec2], [Document.AddAccessorVec4],
// [Document.AddAccessorFloat32], and [Document.AddIndicesUint32] append typed
// data to buffer 0 as a new bufferView and accessor, computing min/max.
//
// # Scene evaluation
//
// The transform helpers work with a column-major [Mat4], a [Quat] rotation, and
// [Vec3] vectors. [TRS] composes a transform; [Mat4.Decompose] recovers it;
// [Node.LocalMatrix] and [Document.GlobalMatrix] give local and world
// transforms. [Document.EvaluateSampler] and [Document.SampleChannel] evaluate
// animation channels (STEP, LINEAR, and CUBICSPLINE, with quaternion [Slerp]
// for rotations), and [Document.ApplyAnimation] poses nodes in place.
// [Document.JointMatrices] computes skinning matrices, and
// [Document.MorphedPositions] blends morph targets.
//
// # Extensions
//
// Named Khronos extensions are modeled by typed structs (for example
// [MaterialsUnlit], [MaterialsTransmission], [TextureTransform], and
// [LightsPunctual]) with parse/encode helpers [GetExtension] and [SetExtension];
// unknown extensions round-trip verbatim through [ExtensionMap] and
// [MarshalExtensions].
//
// # Images
//
// [Document.DecodeImage] decodes embedded, data-URI, and external PNG/JPEG
// images to an [image.Image].
//
// # Validation
//
// [Document.Validate] performs required-field, index-range, and structural
// consistency checks (accessor and bufferView bounds, component/type rules,
// animation, camera, material, image, and skin constraints), returning a
// [ValidationErrors] value that lists every problem with a descriptive path.
//
// # Building
//
// [Triangle] constructs a minimal, valid single-triangle document plus its
// binary buffer; [WriteTriangleGLB] and [WriteTriangleGLTF] emit it directly.
package gltf
