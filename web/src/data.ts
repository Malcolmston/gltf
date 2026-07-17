// Library content for the gltf documentation site. Mirrors the shape used by
// the malcolmston/go landing site's data.ts so the sibling sites stay in sync.
export interface Lib {
  id: string; name: string; icon: string; accent: string; pkg: string; node: string;
  repo: string; docs: string; tagline: string; blurb: string; tags: string[];
  features: string[]; node_code: string; go_code: string; integrate: string;
}

export const NODE_ACCENT = '#8cc84b';

export const GLTF: Lib = {
  id:"gltf", name:"glTF", icon:'<i class="fa-solid fa-cube"></i>', accent:"#e8613c",
  pkg:"github.com/malcolmston/gltf", node:"KhronosGroup/glTF",
  repo:"https://github.com/malcolmston/gltf", docs:"https://malcolmston.github.io/gltf/",
  tagline:"Read, write, validate, and decode glTF 2.0 / GLB 3D assets in Go.",
  blurb:"A dependency-free, standard-library-only Go toolkit for reading, writing, and inspecting glTF 2.0 "+
    "assets in both the JSON (.gltf) and binary (.glb) container formats. The Document type mirrors the full "+
    "glTF 2.0 schema — scenes, nodes, meshes, accessors, buffer views, buffers, materials, textures, "+
    "animations, skins and cameras — as spec-indexed slices, with typed enumerations and vendor Extensions/"+
    "Extras preserved as raw JSON. Buffers resolve from GLB BIN chunks, base64 data URIs or external files; "+
    "typed accessor decoders read vertex data honoring byteOffset, byteStride, normalization and sparse "+
    "substitutions; Validate reports every structural problem with a descriptive path; and a triangle builder "+
    "emits a minimal valid asset. No cgo, no third-party dependencies.",
  tags:["Document model","glTF JSON I/O","GLB binary I/O","accessor decoding","sparse accessors","buffer resolution","validation","stdlib only"],
  features:[
    "<code>Document</code> model — the glTF 2.0 root, with every collection (scenes, nodes, meshes, accessors, buffer views, buffers, materials, textures, animations, skins, cameras) a spec-indexed slice; <code>Extensions</code> and <code>Extras</code> preserved as <code>json.RawMessage</code>",
    "JSON I/O — <code>Decode</code> / <code>Encode</code> plus the <code>Open</code> / <code>Save</code> file helpers for <code>.gltf</code> documents",
    "Binary GLB container — <code>ReadGLB</code> / <code>WriteGLB</code> and <code>OpenGLB</code> / <code>SaveGLB</code>, handling the JSON + BIN chunks, magic/version checks and 4-byte padding",
    "Buffer resolution from GLB BIN chunks, base64 <code>data:</code> URIs and external files via <code>Document.ResolveBuffers</code>, with <code>EncodeDataURI</code> for embedding",
    "Typed accessor decoders — <code>DecodeAccessorVec2</code>/<code>Vec3</code>/<code>Vec4</code>, <code>DecodeAccessorFloat32</code>, <code>DecodeAccessorUint32</code> and <code>DecodeIndices</code>, honoring <code>byteOffset</code>, <code>byteStride</code>, <code>normalized</code> and sparse substitutions",
    "Typed enumerations — <code>ComponentType</code>, <code>AccessorType</code>, <code>PrimitiveMode</code>, <code>Filter</code>, <code>WrapMode</code>, <code>Interpolation</code>, <code>CameraType</code> and <code>AlphaMode</code> carry the spec-defined constant values",
    "Structural validation — <code>Document.Validate</code> checks required fields and index ranges, returning <code>ValidationErrors</code> with paths like <code>meshes[0].primitives[0].attributes.POSITION</code> (<code>AsValidationErrors</code> unwraps them)",
    "Triangle builder — <code>Triangle</code> constructs a minimal valid document plus its BIN buffer, and <code>WriteTriangleGLB</code> / <code>WriteTriangleGLTF</code> emit it directly"
  ],
  node_code:
`// Load a .glb and read the first mesh's vertex positions (three.js GLTFLoader).
import { GLTFLoader } from "three/examples/jsm/loaders/GLTFLoader.js";

new GLTFLoader().load("model.glb", (gltf) => {
  const mesh = gltf.scene.getObjectByProperty("type", "Mesh");
  const positions = mesh.geometry.getAttribute("position");
  console.log(positions.count, "vertices");
});`,
  go_code:
`import "github.com/malcolmston/gltf"

// OpenGLB resolves the BIN chunk and external buffers automatically.
doc, _ := gltf.OpenGLB("model.glb")

prim := doc.Meshes[0].Primitives[0]
positions, _ := doc.DecodeAccessorVec3(prim.Attributes["POSITION"])
indices, _ := doc.DecodeIndices(&prim)
fmt.Println(len(positions), "vertices,", len(indices), "indices")`,
  integrate:
`<span class="tok-c">// Build a minimal triangle document plus its BIN buffer and write a binary GLB.</span>
doc, bin := gltf.Triangle()
if err := gltf.SaveGLB("triangle.glb", doc, bin); err != nil {
    log.Fatal(err)
}

<span class="tok-c">// Read it back; OpenGLB resolves the BIN chunk and any external buffers.</span>
doc, _ = gltf.OpenGLB("triangle.glb")

<span class="tok-c">// Validate structure (required fields + index ranges) before decoding.</span>
if err := doc.Validate(); err != nil {
    if verrs, ok := gltf.AsValidationErrors(err); ok {
        for _, v := range verrs {
            fmt.Printf("%s: %s\\n", v.Path, v.Message)
        }
    }
}

<span class="tok-c">// Decode typed vertex data — byteOffset, byteStride, normalization and sparse are all honored.</span>
prim := doc.Meshes[0].Primitives[0]
positions, _ := doc.DecodeAccessorVec3(prim.Attributes["POSITION"])
indices, _ := doc.DecodeIndices(&prim)

<span class="tok-c">// Or emit a self-contained .gltf with the buffer embedded as a base64 data URI.</span>
f, _ := os.Create("triangle.gltf")
defer f.Close()
gltf.WriteTriangleGLTF(f)`
};
