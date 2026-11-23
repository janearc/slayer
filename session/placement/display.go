package placement

import (
	"sync"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// DisplayVertex is a native object-local corner for opaque working display.
type DisplayVertex struct {
	Position maths.Vec
	Normal   maths.Vec
	Colour   maths.Light
}

// DisplayMesh is immutable cached native geometry; indices are triangle/line lists.
type DisplayMesh struct {
	Vertices           []DisplayVertex
	Indices            []uint32
	Edges              []uint32
	Notice, Diagnostic string
	hitter             shape.Hitter
}

// DisplayObject places one shared mesh; only At changes during a gesture.
type DisplayObject struct {
	ID, Name string
	Mesh     *DisplayMesh
	At       maths.Subspace
}

// DisplayScene is a mutable view over immutable shared object-local meshes.
type DisplayScene struct {
	Objects       []DisplayObject
	Eye           maths.Subspace
	Focus         maths.Vec
	Lens          film.Pinhole
	Width, Height int
}

// DisplayCache retains manufactured geometry independently of object pose/camera.
type DisplayCache struct {
	mu     sync.Mutex
	meshes map[displayKey]*DisplayMesh
	worlds map[string]*DisplayMesh
}

// displayKey names everything that decides a cached mesh: the kind, the
// detail, the digest of the recipe the kind was built from (empty for a
// plain kind) and both seeds.
type displayKey struct {
	kind, detail, recipe  string
	sceneSeed, objectSeed uint64
}

// NewDisplayCache creates a cache for one open editor/document lifetime.
func NewDisplayCache() *DisplayCache {
	return &DisplayCache{meshes: map[displayKey]*DisplayMesh{}, worlds: map[string]*DisplayMesh{}}
}

// SetViewport aligns native film rays with the actual display aspect ratio.
func (scene *DisplayScene) SetViewport(width, height int) {
	if scene == nil || width <= 0 || height <= 0 || scene.Width <= 0 || scene.Height <= 0 {
		return
	}
	scene.Lens.Pixel = float64(width) * float64(scene.Height) / (float64(height) * float64(scene.Width))
}
