package gltf_test

import (
	"bytes"
	"fmt"

	"github.com/malcolmston/gltf"
)

// Example builds a single-triangle mesh, writes it to an in-memory GLB
// container, reads it back, resolves the embedded buffer, and prints the number
// of decoded vertices.
func Example() {
	// Build a minimal triangle document and its binary buffer.
	doc, bin := gltf.Triangle()

	// Write it to an in-memory GLB stream.
	var buf bytes.Buffer
	if err := gltf.WriteGLB(&buf, doc, bin); err != nil {
		panic(err)
	}

	// Read the GLB back and resolve its embedded binary chunk.
	loaded, chunk, err := gltf.ReadGLB(&buf)
	if err != nil {
		panic(err)
	}
	if err := loaded.ResolveBuffers("", chunk); err != nil {
		panic(err)
	}

	// Decode the POSITION accessor into typed vertices.
	positions, err := loaded.DecodeAccessorVec3(0)
	if err != nil {
		panic(err)
	}

	fmt.Printf("vertices: %d\n", len(positions))
	fmt.Printf("first: %v\n", positions[0])
	// Output:
	// vertices: 3
	// first: [0 0 0]
}
