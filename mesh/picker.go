package mesh

import "github.com/janearc/slayer/maths"

// Picker intersects a mesh's original faces without growing its render tree.
// It bounds runs of adjacent faces in one pass over the immutable mesh.
type Picker struct {
	mesh   *Mesh
	blocks []node
}

// NewPicker builds an index for interactive picking without sorting faces.
func (made *Mesh) NewPicker() *Picker {
	const blockFaces = 128
	picker := &Picker{mesh: made, blocks: make([]node, 0, (len(made.faces)+blockFaces-1)/blockFaces)}
	for first := 0; first < len(made.faces); first += blockFaces {
		last := min(first+blockFaces, len(made.faces))
		box := spanOf(made.points, made.faces[first])
		for _, face := range made.faces[first+1 : last] {
			box = box.joined(spanOf(made.points, face))
		}
		picker.blocks = append(picker.blocks, node{bounds: [boundEnds][axes]float64{box.low, box.high}, first: first, count: last - first})
	}
	return picker
}

// Hit returns the nearest producer triangle hit strictly between near and far.
// The interval describes a forward ray: near is nonnegative, and far may be
// positive infinity.
func (picker *Picker) Hit(ray maths.Ray, near, far float64) (float64, bool) {
	look := aimed(ray)
	best := far
	for i := range picker.blocks {
		block := &picker.blocks[i]
		if _, enters := look.enters(block, near, best); !enters {
			continue
		}
		for face := block.first; face < block.first+block.count; face++ {
			if distance, _, hit := look.meets(picker.mesh, face, near, best); hit {
				best = distance
			}
		}
	}
	return best, best < far
}
