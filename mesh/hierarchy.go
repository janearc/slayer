package mesh

import "github.com/janearc/slayer/maths"

// MostTreeDepth is the most levels the mesh's tree has below its root: no
// cut goes past it. A walk that keeps the boxes waiting on a stack needs
// room for one more than this.
const MostTreeDepth = deepest

// Growers grows the trees of many meshes at once, on no more goroutines
// than it was made with: each mesh being grown holds one, and a mesh
// large enough to be cut by several takes any not held. The trees are
// the ones TreeSize would have grown, to the box and the place.
type Growers struct{ places chan struct{} }

// NewGrowers is growers of at most workers goroutines at once, and of one
// at least.
func NewGrowers(workers int) *Growers {
	return &Growers{places: make(chan struct{}, max(unity, workers))}
}

// Grow grows the mesh's tree if nothing has yet, waiting for a goroutine
// of the growers' to do it on. A tree already grown, or being grown by
// another call, is waited for and not grown twice.
func (grown *Growers) Grow(made *Mesh) {
	grown.places <- struct{}{}
	defer func() { <-grown.places }()
	made.treeOnce.Do(func() {
		made.treeMu.Lock()
		defer made.treeMu.Unlock()
		made.tree = grownBy(made.points, made.faces, grown.places)
	})
}

// TreeSize is how many boxes the mesh's tree holds, the root first. It
// grows the tree if nothing has yet, which is the whole cost of a first
// call; every call after is a read. The tree is the one rays are met by.
func (made *Mesh) TreeSize() int {
	made.ensureTree()
	return len(made.tree.nodes)
}

// TreeBox is box i of the tree, the root being box 0: its corners, the
// smaller and the larger, and a place and a count. A leaf has a count
// above zero and holds the faces TreeFace gives for the places from the
// place on, count of them; a branch has a count of zero and its two
// children are the boxes at the place and the one after. i must be less
// than TreeSize.
func (made *Mesh) TreeBox(i int) (smaller, larger maths.Vec, place, count int) {
	made.ensureTree()
	at := &made.tree.nodes[i]
	return vectorOf(at.bounds[low]), vectorOf(at.bounds[high]),
		at.first, at.count
}

// TreeFace is the face held at place i of the order the tree's leaves
// hold the faces in: a face number, as Corners takes. Every face is held
// at one place. i must be less than FaceCount.
func (made *Mesh) TreeFace(i int) int {
	made.ensureTree()
	return made.tree.order[i]
}
