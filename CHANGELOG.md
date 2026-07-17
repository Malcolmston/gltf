# Changelog

All notable changes to this project are documented here. The format is loosely
based on [Keep a Changelog](https://keepachangelog.com/).

## [0.2.0]

Major expansion toward full glTF 2.0 parity. All additions are pure standard
library.

### Added

- **Transform math** (`math.go`): `Vec3`, `Vec4`, `Quat`, and column-major
  `Mat4` types; `TRS` composition; `Mat4.Decompose`, `Mat4.Mul`, `Mat4.Inverse`,
  `Mat4.TransformPoint`; quaternion `Matrix`/`Normalize`/`Slerp`;
  `Node.LocalMatrix` and `Document.GlobalMatrix` (world transforms up the node
  hierarchy, with cycle detection).
- **Animation sampling** (`animation.go`): `Document.EvaluateSampler` with STEP,
  LINEAR, and CUBICSPLINE interpolation (quaternion slerp for rotations),
  `Document.SampleChannel`, `Document.SamplerKeyframes`, and
  `Document.ApplyAnimation` to pose a document at a time `t`.
- **Skinning** (`skin.go`): `Document.InverseBindMatrices`,
  `Document.JointMatrices`, and `Document.JointMatricesForNode`.
- **Morph targets** (`morph.go`): `Document.DecodeMorphTargetVec3`,
  `DecodeMorphTargetsVec3`, `BlendMorphTargetsVec3`, `MorphedPositions`, and
  `EffectiveWeights`.
- **Accessor encoding** (`encode.go`): `Document.AddAccessorFloat32`,
  `AddAccessorVec2/Vec3/Vec4`, `AddIndicesUint32`, and `AddBinData`, building
  bufferViews and accessors (with automatic min/max) from typed slices — the
  inverse of the decoders.
- **Image decoding** (`image.go`): `Document.ImageBytes` and
  `Document.DecodeImage` for embedded, data-URI, and external PNG/JPEG images.
- **Khronos extensions** (`extensions.go`): typed structs and parse/encode
  helpers for `KHR_materials_unlit`, `KHR_materials_emissive_strength`,
  `KHR_materials_transmission`, `KHR_materials_ior`, `KHR_texture_transform`,
  `KHR_lights_punctual`, and `KHR_materials_pbrSpecularGlossiness`, plus generic
  `ExtensionMap`, `MarshalExtensions`, `GetExtension`, and `SetExtension`
  helpers that preserve unknown extensions on round-trip.
- `Document.DecodeAccessorMat4` for MAT4 accessors (e.g. inverse bind matrices).

### Changed

- **Validation** (`validate.go`) greatly expanded: accessor bounds within their
  bufferView, bufferView bounds within their buffer, `normalized`/component-type
  rules, min/max length consistency, sparse index component types, index
  accessor type checks, primitive mode and morph-target references, animation
  input/output/interpolation consistency, camera numeric constraints, material
  alpha modes, image uri/bufferView exclusivity, skin inverse-bind-matrix
  counts, and `extensionsRequired` ⊆ `extensionsUsed`.

## [0.1.0]

- Initial release: glTF 2.0 document model, `.gltf`/`.glb` read/write, accessor
  decoding, buffer resolution, and structural validation.
