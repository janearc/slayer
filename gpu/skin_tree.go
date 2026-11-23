package gpu

import "github.com/janearc/slayer/mesh"

// faceRun is a mesh's faces as they lie in the pool: the place of its
// first face, and how many follow. The pool keeps one for each mesh, by
// the mesh's root, where it used to keep every box of every tree.
type faceRun struct{ first, count int }

// skinTree is the tree of one of slayer's meshes: the mesh itself, with
// the tree it has grown, which is grown now if it has not been. It copies
// no face and no box; packTree reads them from the mesh.
func skinTree(skin *mesh.Mesh) tree {
	skin.TreeSize()
	return tree{skin: skin}
}

// meshTree is the tree of one of slayer's meshes with its faces and boxes
// copied out as float64s: the mesh's own tree, its faces in the order its
// leaves hold them. Only the low-retention trial, which pools faces as
// values, and the tests, which weigh packTree against it, ask for it.
func meshTree(skin *mesh.Mesh) tree {
	made := tree{triangles: make([]triangle, skin.FaceCount()),
		nodes: make([]node, skin.TreeSize())}
	for place := range made.triangles {
		face := skin.TreeFace(place)
		one, two, three := skin.Corners(face)
		normals, smooth := skin.CornerNormals(face)
		made.triangles[place] = triangle{corners: threeOf(one, two, three),
			normals: normals, smooth: smooth}
	}
	for at := range made.nodes {
		smaller, larger, first, count := skin.TreeBox(at)
		made.nodes[at] = node{bounds: box{low: smaller, high: larger},
			first: first, count: count}
	}
	return made
}

// faced is the tree with its faces and boxes held as values: itself, or
// a skin's copied out by meshTree.
func (grown tree) faced() tree {
	if grown.skin == nil {
		return grown
	}
	return meshTree(grown.skin)
}

// faceCount is how many faces the tree holds.
func (grown tree) faceCount() int {
	if grown.skin != nil {
		return grown.skin.FaceCount()
	}
	return len(grown.triangles)
}

// nodeCount is how many boxes the tree holds.
func (grown tree) nodeCount() int {
	if grown.skin != nil {
		return grown.skin.TreeSize()
	}
	return len(grown.nodes)
}

// bendCount is how many of the tree's faces are shaded smooth. a skin
// is shaded smooth all over or flat all over.
func (grown tree) bendCount() int {
	if grown.skin != nil {
		if grown.skin.Smooth() {
			return grown.skin.FaceCount()
		}
		return 0
	}
	bends := 0
	for index := range grown.triangles {
		if grown.triangles[index].smooth {
			bends++
		}
	}
	return bends
}

// rootBounds is the box round the whole tree, in its mesh's own space.
func (grown tree) rootBounds() box {
	if grown.skin != nil {
		smaller, larger, _, _ := grown.skin.TreeBox(0)
		return box{low: smaller, high: larger}
	}
	return grown.nodes[0].bounds
}

// finite is whether every face of the tree fits float32, as finiteFace
// asks of each.
func (grown tree) finite() bool {
	if grown.skin == nil {
		for index := range grown.triangles {
			if !finiteFace(grown.triangles[index]) {
				return false
			}
		}
		return true
	}
	for face := range grown.skin.FaceCount() {
		one, two, three := grown.skin.Corners(face)
		normals, smooth := grown.skin.CornerNormals(face)
		if !finiteFace(triangle{corners: threeOf(one, two, three),
			normals: normals, smooth: smooth}) {
			return false
		}
	}
	return true
}

// packTree packs a tree's faces, the bends of those shaded smooth, and
// its boxes, after those already packed: place says how many faces,
// bends and boxes are before it, so a branch's children and a leaf's
// faces are moved past them, and a smooth face's bend is counted on.
func (encoded *scene) packTree(grown tree, place pageSize) {
	if grown.skin != nil {
		encoded.packSkin(grown.skin, place)
		return
	}
	bend := place.bends
	for _, face := range grown.triangles {
		number := -1
		if face.smooth {
			number = bend
			bend++
		}
		encoded.packedFace(face, number)
	}
	for _, branch := range grown.nodes {
		branch.first += place.firstBefore(branch.count)
		encoded.packedNode(branch)
	}
}

// packSkin packs one of slayer's meshes where it lies: each face read
// from the mesh in the order its leaves hold them, and each box from its
// tree, with no face or box copied between. it packs what packTree packs
// for meshTree's copy, byte for byte.
func (encoded *scene) packSkin(skin *mesh.Mesh, place pageSize) {
	smooth := skin.Smooth()
	for at := range skin.FaceCount() {
		face := skin.TreeFace(at)
		one, two, three := skin.Corners(face)
		bend := -1
		if smooth {
			normals, _ := skin.CornerNormals(face)
			encoded.packedBend(normals, false)
			bend = place.bends + at
		}
		encoded.packedCorners(threeOf(one, two, three), bend)
	}
	for at := range skin.TreeSize() {
		smaller, larger, first, count := skin.TreeBox(at)
		encoded.packedBox(smaller, larger, first+place.firstBefore(count),
			count)
	}
}

// firstBefore is what a box's first is moved by: the boxes before it for
// a branch, whose count is nought, and the faces before it for a leaf.
func (place pageSize) firstBefore(count int) int {
	if count == 0 {
		return place.nodes
	}
	return place.triangles
}
