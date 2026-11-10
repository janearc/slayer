package gpu

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// packer is one buffer as the kernel reads it, every number a float32
// or an int32, laid out in fours. a row of a structure is written to it
// by field name; see structure.go.
type packer struct{ bytes []byte }

// scene is a world at one time as the kernel reads it, buffer by
// buffer, and how many of each thing there are.
type scene struct {
	header, spheres, triangles, nodes packer
	materials, lamps, volumes, trees  packer
	instances, tops, sky, bends       packer
	// boxes is every boxed medium's cells, prepared on CPU or GPU: density,
	// then its shine and paint three numbers a cell
	boxes packer
	// rays is every sample's ray, its start and its way, six floats
	// each, made on the cpu through a lens that is not a pinhole; none
	// through a pinhole, whose rays the kernel makes
	rays   packer
	counts struct {
		spheres, triangles, nodes, lamps, volumes, trees int
		instances, tops, beams                           int
	}
	materialOf map[*thing.Material]int
	// faces is every mesh's faces in its own space and their trees,
	// each mesh once; meshes is where each went, by the mesh; and
	// standings is every place a mesh stands
	faces     tree
	meshes    map[any]pooled
	standings []standing
	groups    []meshGroup
	groupOf   map[*shape.MeshGroup]int
	// bakes are the volumes to bake, and cells how many cells all
	// their grids hold together
	smokes []smokeGrid
	flames []flameBake
	bakes  []bake
	cells  int
	// baked is every baked cell, filled in after encoding by the bake
	// kernel, before the trace
	baked packer
	// at is the picture's time, which the textures are ticked to;
	// paints is every material's textures, by its index; painted is
	// whether any material has one, and glossy whether any mirrors
	at              time.Duration
	paints          []painting
	painted, glossy bool
	// spread is how big the world is where floats are concerned, as
	// slayer's tracer measures it, which a ray leaving a surface is
	// lifted by a share of
	spread float64
	// looked is whether the rays were made on the cpu
	looked bool
}

// records is how many places a sample hands back for the cpu to paint:
// none when nothing is painted, the place the ray met, and the place
// its mirrored ray met when anything mirrors.
func (encoded *scene) records() int {
	switch {
	case !encoded.painted:
		return 0
	case encoded.glossy:
		return 2
	}
	return 1
}

// perSample is how many numbers the kernel gives back for a sample:
// the twelve every sample has, and a record's for each place it hands
// back.
func (encoded *scene) perSample() int {
	return per + perRecord*encoded.records()
}

// newScene is a scene with nothing in it yet.
func newScene() *scene {
	return &scene{materialOf: map[*thing.Material]int{},
		meshes: map[any]pooled{}, groupOf: map[*shape.MeshGroup]int{}}
}

// discardFaceValues releases float64 construction faces after serialization.
// Native planning uses node ranges and packed triangles, not these values.
func (encoded *scene) discardFaceValues() { encoded.faces.triangles = nil }

// discardBuildHierarchy releases descriptors after native IDs/plans are made.
// Rendering and lighting use the packed buffers retained by this scene.
func (encoded *scene) discardBuildHierarchy() {
	encoded.faces.nodes = nil
	encoded.standings = nil
	encoded.groups = nil
	encoded.meshes = nil
	encoded.groupOf = nil
	encoded.materialOf = nil
}

// unsupported is a world the kernel cannot draw, and what in it.
type unsupported struct{ what string }

// Error says what the kernel cannot draw.
func (refused unsupported) Error() string {
	return "gpu: cannot draw " + refused.what + " yet"
}

// inputs is the scene's buffers in the order the kernel binds them.
func (encoded *scene) inputs() [][]byte {
	return [][]byte{encoded.header.bytes, encoded.spheres.bytes,
		encoded.triangles.bytes, encoded.nodes.bytes,
		encoded.materials.bytes, encoded.lamps.bytes,
		encoded.volumes.bytes, encoded.baked.bytes, encoded.trees.bytes,
		encoded.instances.bytes, encoded.tops.bytes, encoded.sky.bytes,
		encoded.bends.bytes, encoded.rays.bytes, encoded.boxes.bytes}
}

// refuse says why a world cannot be drawn here, or nothing when it can:
// the world as a whole. the rest is refused where it is encoded, the
// shapes in parts.go and the volumes in volumes.go.
func refuse(world *thing.World) error {
	switch {
	case len(world.Fields) > 0:
		return unsupported{"a field"}
	case world.Haze.Density > 0:
		return unsupported{"the world's haze"}
	}
	for _, lamp := range world.Lamps {
		switch source := lamp.(type) {
		case emit.Lit, emit.Distant, emit.Volume:
		case *emit.Beam:
			if source == nil {
				return unsupported{"a nil beam"}
			}
		default:
			kind := fmt.Sprintf("a lamp of kind %T", lamp)
			return unsupported{kind}
		}
	}
	return nil
}

// encode is the world at time t, seen through a lens from eye, at a
// width and height that are already the samples', not the pixels'.
func encode(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height int) (*scene, error) {
	return encodeGrown(world, nil, eye, lens, at, width, height)
}

// encodeGrown is encode with conifers the kernel grows.
func encodeGrown(world *thing.World, trees []grown, eye maths.Subspace,
	lens film.Sensor, at time.Duration, width,
	height int) (*scene, error) {
	if err := refuse(world); err != nil {
		return nil, err
	}
	encoded := newScene()
	encoded.at = at
	encoded.spread = trace.Spread(world, at)
	if err := encoded.things(world, at); err != nil {
		return nil, err
	}
	if err := encoded.volumesOf(world, at); err != nil {
		return nil, err
	}
	encoded.conifers(trees)
	encoded.lampsOf(world)
	encoded.skyOf(world)
	pinhole, isPinhole := pinholeOf(lens)
	if !isPinhole {
		encoded.raysOf(eye, lens, width, height)
	}
	encoded.camera(eye, pinhole, width, height)
	return encoded, nil
}

// pinholeOf is the lens as a pinhole, a nil lens being the zero film.Pinhole,
// and whether it is one.
func pinholeOf(lens film.Sensor) (film.Pinhole, bool) {
	if lens == nil {
		return film.Pinhole{}, true
	}
	// PinholeOf returns a checked pointer. It has the same ray geometry as
	// its value; retaining the generic-lens path here uploads a full CPU ray
	// array even though the kernel can generate those pinhole rays itself.
	if pointer, ok := lens.(*film.Pinhole); ok && pointer != nil {
		return *pointer, true
	}
	pinhole, ok := lens.(film.Pinhole)
	return pinhole, ok
}

// raysOf is every sample's ray through a lens, in the world, row by row,
// each made by film.Look as slayer's tracer makes it: so whatever a lens
// does, its aperture, its focus, its distortion, the kernel's rays are
// the tracer's own.
func (encoded *scene) raysOf(eye maths.Subspace, lens film.Sensor, width,
	height int) {
	encoded.rays.bytes = make([]byte, 0, width*height*rayWords*4)
	for y := range height {
		for x := range width {
			encoded.ray(film.Look(eye, lens, x, y, width, height,
				maths.Band{}))
		}
	}
	encoded.looked = true
}

// ray appends one ray: its start and its way, as floats
func (encoded *scene) ray(made maths.Ray) {
	for _, part := range append(point(made.At), point(made.Dir)...) {
		encoded.rays.word(math.Float32bits(float32(part)))
	}
}

// rayWords is how many numbers a ray takes: its start and its way
const rayWords = 6
