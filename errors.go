package gltf

import "fmt"

// errIndexRange returns a standard out-of-range error for a named index.
func errIndexRange(what string, idx, n int) error {
	return fmt.Errorf("gltf: %s index %d out of range [0,%d)", what, idx, n)
}

// errCycle returns an error describing a cycle discovered in the node
// hierarchy at the given node index.
func errCycle(node int) error {
	return fmt.Errorf("gltf: node hierarchy contains a cycle at node %d", node)
}
