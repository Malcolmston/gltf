# Changelog

All notable changes to this project are documented here. The format is loosely
based on [Keep a Changelog](https://keepachangelog.com/).

## [0.3.0]

Further expansion toward glTF 2.0 rendering parity. All additions are pure
standard library, deterministic, and covered by known-answer tests.

### Added

- **Vector and quaternion math** (`vecmath.go`): `Vec3` arithmetic
  (`Add`, `Sub`, `Scale`, `Dot`, `Cross`, `Length`, `Normalize`, `Lerp`);
  quaternion operations (`Quat.Mul`, `Quat.Conjugate`, `Quat.Dot`,
  `Quat.Rotate`, `QuatFromAxisAngle`, `QuatFromEuler`); matrix helpers
  (`Mat4.Transpose`, `Mat4.TransformDir`); and view/projection builders
  `LookAt`, `Perspective`, and `Orthographic`.
- **Camera projection matrices** (`camera.go`): `Camera.ProjectionMatrix`,
  `CameraPerspective.ProjectionMatrix`, and
  `CameraOrthographic.ProjectionMatrix`, following the glTF 2.0 camera
  conventions (finite and infinite far planes).
- **Bounding boxes** (`bounds.go`): a `Box` axis-aligned bounding-box type with
  `EmptyBox`, `Add`, `Union`, `Center`, `Size`, `Contains`, `Empty`, and
  `Transform`; plus `Document.AccessorBounds`, `Document.PrimitiveBounds`, and
  `Document.SceneBounds` (world-space scene extent).
- **Scene traversal** (`scene.go`): `Document.RootNodes`,
  `Document.GlobalMatrices` (world transforms for every node, cycle-detecting),
  and `Document.NodesInScene`.
- **Additional PBR material extensions** (`extensions_ext.go`): typed
  `MaterialsClearcoat`, `MaterialsSheen`, `MaterialsSpecular`, and
  `MaterialsVolume` structs with matching `Material.Clearcoat`,
  `Material.Sheen`, `Material.Specular`, and `Material.Volume` accessors
  (`KHR_materials_clearcoat`, `_sheen`, `_specular`, `_volume`).
- **Enum names** (`enums_ext.go`): `String` methods for `PrimitiveMode`,
  `Filter`, `WrapMode`, and `TargetType`.

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
