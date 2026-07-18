package gltf

// RootNodes returns the indices of every node that is not referenced as a child
// of another node, in ascending order. These are the roots of the document's
// node forest, regardless of which scene they belong to.
func (d *Document) RootNodes() []int {
	parents := d.parentIndex()
	roots := make([]int, 0, len(d.Nodes))
	for i := range d.Nodes {
		if parents[i] == -1 {
			roots = append(roots, i)
		}
	}
	return roots
}

// GlobalMatrices returns the world (global) transform of every node in the
// document, indexed by node index. Each matrix is the product of the local
// matrices from the node's root ancestor down to the node. It returns an error
// if the node hierarchy contains a cycle.
func (d *Document) GlobalMatrices() ([]Mat4, error) {
	parents := d.parentIndex()
	out := make([]Mat4, len(d.Nodes))
	done := make([]bool, len(d.Nodes))
	var resolve func(i int, stack []bool) (Mat4, error)
	resolve = func(i int, stack []bool) (Mat4, error) {
		if done[i] {
			return out[i], nil
		}
		if stack[i] {
			return IdentityMatrix(), errCycle(i)
		}
		stack[i] = true
		local := d.Nodes[i].LocalMatrix()
		var global Mat4
		if p := parents[i]; p == -1 {
			global = local
		} else {
			pm, err := resolve(p, stack)
			if err != nil {
				return IdentityMatrix(), err
			}
			global = pm.Mul(local)
		}
		stack[i] = false
		out[i] = global
		done[i] = true
		return global, nil
	}
	for i := range d.Nodes {
		if _, err := resolve(i, make([]bool, len(d.Nodes))); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// NodesInScene returns the indices of every node reachable from the scene at
// sceneIndex, obtained by a depth-first walk of the scene's root nodes and their
// descendants. Each node appears once, in depth-first pre-order. It returns an
// error for an out-of-range scene index or a cycle in the node hierarchy.
func (d *Document) NodesInScene(sceneIndex int) ([]int, error) {
	if sceneIndex < 0 || sceneIndex >= len(d.Scenes) {
		return nil, errIndexRange("scene", sceneIndex, len(d.Scenes))
	}
	visited := make([]bool, len(d.Nodes))
	out := make([]int, 0, len(d.Nodes))
	var walk func(i int) error
	walk = func(i int) error {
		if i < 0 || i >= len(d.Nodes) {
			return errIndexRange("node", i, len(d.Nodes))
		}
		if visited[i] {
			return errCycle(i)
		}
		visited[i] = true
		out = append(out, i)
		for _, c := range d.Nodes[i].Children {
			if err := walk(c); err != nil {
				return err
			}
		}
		return nil
	}
	for _, root := range d.Scenes[sceneIndex].Nodes {
		if err := walk(root); err != nil {
			return nil, err
		}
	}
	return out, nil
}
