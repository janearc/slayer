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
	gravity                           bool
	volumeCells                       int
	header, spheres, triangles, nodes packer
	materials, lamps, volumes, trees  packer
	instances, tops, sky, bends       packer
	// fields is every height field's placement and sizes, and heights
	// their samples, flat, each field's run beginning at its first.
	fields, heights packer
	// Clear-medium data is an opt-in ABI extension. With no named optical
	// medium and no explicit initial stack, these buffers remain absent.
	clearMedia, initialState packer
	opticalState             bool
	oceanSpotScattering      bool
	oceanMoon                *trace.OceanMoonPlan
	mediumIDs                map[string]int
	// boxes is every boxed medium's cells, prepared on CPU or GPU: density,
	// then its shine and paint three numbers a cell
	boxes packer
	// boxedCPU measures only generic grid sampling, before float packing.
	boxedCPU BoxedPreparation
	// rays is every sample's ray, its start and its way, six floats
	// each, made on the cpu through a lens that is neither a pinhole nor
	// a thin lens; none through those, whose rays the kernel makes
	// (exposure.go). through is how
	// they are made, and with it they are made a stripe at a time when
	// the picture's would not fit in one input (see rays.go)
	rays    packer
	through *through
	counts  struct {
		spheres, triangles, nodes, lamps, volumes, trees int
		instances, tops, beams, shells, fields           int
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
	// order is the standings' tree and each group's, worked once for
	// the three that read it; see ordered
	order *standingOrder
	// keeper is a retained drawer's kept face trees, or none for the
	// per-call path, which grows every tree
	keeper *keptTrees
	// pending is the meshes' trees in line for the pool, in the order
	// they were first seen, and pendingNodes how many boxes they hold
	pending      []pendingMesh
	pendingNodes int
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
	at                       time.Duration
	paints                   []painting
	painted, glossy, cleared bool
	// spread is how big the world is where floats are concerned, as
	// slayer's tracer measures it, which a ray leaving a surface is
	// lifted by a share of
	spread float64
	// looked is whether the rays were made on the cpu
	looked bool
	// volumeSky is zero for legacy media, otherwise an even full-sphere count.
	volumeSky int
	incident  *incidentGrid
	// reached is every boxed medium fast light keeps lamps' grids over,
	// noted as it is encoded; reach is the store made of them when the
	// mode asks for it, and nil otherwise (see reach.go)
	reached []reachGrid
	reach   *reachStore
}

// records is how many places a sample hands back for the cpu to paint:
// none when nothing is painted; the place the ray met; the place its
// mirrored ray met when anything mirrors; and the place its bent chain
// ended when anything is clear. The slots are fixed: met, mirrored,
// passed, so a clear scene carries the mirrored slot whether or not
// anything mirrors.
func (encoded *scene) records() int {
	switch {
	case !encoded.painted:
		return 0
	case encoded.cleared:
		return 3
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
		meshes: map[any]pooled{}, groupOf: map[*shape.MeshGroup]int{},
		mediumIDs: map[string]int{}}
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

// MaxGravityFields is the number of independently placed fields the Metal
// transport currently supports. Larger scenes need the CPU reference path.
const MaxGravityFields = 1

// fieldSeedWords is one camera ray, its observer state, entry and energy.
const fieldSeedWords = 16

// Error says what the kernel cannot draw.
func (refused unsupported) Error() string {
	return "gpu: cannot draw " + refused.what + " yet"
}

// inputs is the scene's buffers in the order the kernel binds them.
func (encoded *scene) inputs() [][]byte {
	inputs := [][]byte{encoded.header.bytes, encoded.spheres.bytes,
		encoded.triangles.bytes, encoded.nodes.bytes,
		encoded.materials.bytes, encoded.lamps.bytes,
		encoded.volumes.bytes, encoded.baked.bytes, encoded.trees.bytes,
		encoded.instances.bytes, encoded.tops.bytes, encoded.sky.bytes,
		encoded.bends.bytes, encoded.rays.bytes, encoded.boxes.bytes,
		encoded.fields.bytes, encoded.heights.bytes}
	if encoded.opticalState {
		inputs = append(inputs, encoded.clearMedia.bytes,
			encoded.initialState.bytes)
	}
	return inputs
}

// prepareOpticalState builds a deterministic per-world table and the small
// per-draw camera stack. The IDs are local to this packed scene; names remain
// the source identity validated by trace.ValidateClearState.
func (encoded *scene) prepareOpticalState(world *thing.World,
	media []trace.InitialMedium) error {
	for index := range world.Things {
		material := world.Things[index].Material
		if material == nil {
			continue
		}
		if material.Through > 0 && !fitsInt32(material.Rank) {
			return unsupported{"a clear-boundary rank outside the Metal integer range"}
		}
		if err := encoded.registerClearMedium(material.ClearMedium); err != nil {
			return err
		}
	}
	for _, initial := range media {
		if !fitsInt32(initial.Rank) {
			return unsupported{"an initial-medium rank outside the Metal integer range"}
		}
		if err := encoded.registerClearMedium(initial.ClearMedium); err != nil {
			return err
		}
	}
	encoded.initialState.word(uint32(int32(len(media))))
	encoded.initialState.word(uint32(int32(len(encoded.mediumIDs))))
	encoded.initialState.word(0)
	encoded.initialState.word(0)
	for index := 0; index < trace.MaxInitialMedia; index++ {
		if index >= len(media) {
			encoded.initialState.word(0)
			encoded.initialState.word(0)
			encoded.initialState.word(0)
			encoded.initialState.word(0)
			continue
		}
		initial := media[index]
		packed := float32(initial.Index)
		if !(packed > 0) || math.IsInf(float64(packed), 0) {
			return unsupported{"an initial refractive index unresolved in float32"}
		}
		id := encoded.mediumID(initial.ClearMedium)
		encoded.initialState.word(math.Float32bits(packed))
		encoded.initialState.word(uint32(int32(initial.Rank)))
		encoded.initialState.word(uint32(int32(id)))
		encoded.initialState.word(0)
	}
	return nil
}

// registerClearMedium gives a clear medium its Metal ID. It refuses a
// medium with in-scattering outside the opted-in routes.
func (encoded *scene) registerClearMedium(medium *thing.ClearMedium) error {
	if !medium.Defined() {
		return nil
	}
	moon := encoded.oceanMoon
	moonMedium := moon != nil && moon.Medium.Name() == medium.Name()
	if medium.HasScattering() && !encoded.oceanSpotScattering && !moonMedium {
		return unsupported{"clear-medium in-scattering outside the opted-in ocean spot route"}
	}
	if _, exists := encoded.mediumIDs[medium.Name()]; exists {
		return nil
	}
	if int64(len(encoded.mediumIDs)) > int64(1<<31-1) {
		return unsupported{"too many named clear media for Metal IDs"}
	}
	extinction := medium.Extinction()
	scattering := medium.Scattering()
	coefficients := []float64{extinction.R, extinction.G, extinction.B,
		medium.MetresPerUnit(), scattering.R, scattering.G, scattering.B,
		medium.Phase(), medium.MaxInscatterDistanceMetres()}
	packed := make([]float64, len(coefficients))
	for index, value := range coefficients {
		minimum := 0.0
		if value > 0 {
			minimum = float64(math.Float32frombits(0x00800000))
		}
		number := float32(value)
		if math.IsInf(float64(number), 0) || math.IsNaN(float64(number)) ||
			(value > 0 && float64(number) < minimum) {
			return unsupported{fmt.Sprintf("clear medium %q has a coefficient outside Metal float32 range", medium.Name())}
		}
		packed[index] = float64(number)
	}
	id := len(encoded.mediumIDs)
	encoded.mediumIDs[medium.Name()] = id
	row := values{
		"extinction_mpu": packed[:4],
		"scatter_phase":  packed[4:8],
		"profile":        {float64(medium.InscatterSteps()), packed[8]},
		"moon_air":       {0, 0, 0, 0}, "moon_light": {0, 0, 0, 0}, "moon_plane": {0, 0, 0, 0},
	}
	if moonMedium {
		row["profile"] = []float64{float64(medium.InscatterSteps()), packed[8], float64(flag(moon.MixedCloud)), float64(flag(moon.RaySpacing))}
		row["moon_air"] = []float64{moon.AirToward.X, moon.AirToward.Y, moon.AirToward.Z, moon.Index}
		row["moon_light"] = []float64{moon.Strength.R, moon.Strength.G, moon.Strength.B, float64(moon.LampIndex + 1)}
		row["moon_plane"] = []float64{moon.Centre.X, moon.Centre.Y, moon.Centre.Z, moon.HalfExtent}
		for _, name := range []string{"moon_air", "moon_light", "moon_plane"} {
			for _, value := range row[name] {
				number := float32(value)
				if math.IsInf(float64(number), 0) || math.IsNaN(float64(number)) ||
					(value != 0 && math.Abs(float64(number)) < float64(math.Float32frombits(0x00800000))) {
					return unsupported{"lunar source plan outside normal float32 range"}
				}
			}
		}
	}
	encoded.clearMedia.row(clearMediumStruct, row)
	return nil
}

// mediumID returns the Metal ID that a clear medium was registered under.
func (encoded *scene) mediumID(medium *thing.ClearMedium) int {
	if !medium.Defined() {
		return -1
	}
	id, ok := encoded.mediumIDs[medium.Name()]
	if !ok {
		panic("gpu: clear medium was not registered before material packing")
	}
	return id
}

// fitsInt32 reports whether value fits in a signed 32-bit integer.
func fitsInt32(value int) bool {
	return int64(value) >= -1<<31 && int64(value) <= 1<<31-1
}

// refuse says why a world cannot be drawn here, or nothing when it can:
// the world as a whole. the rest is refused where it is encoded, the
// shapes in parts.go and the volumes in volumes.go.
func refuse(world *thing.World) error {
	if world.OceanSurface != nil && world.OceanSurface.InterfaceEvents > 0 {
		return unsupported{"CPU-only ocean interface-event transport"}
	}
	if world.OceanSurface != nil && world.OceanSurface.InfiniteMeanPlane {
		if !world.OceanSurface.DirectMoonMeanPlane {
			return unsupported{"CPU-only infinite mean-plane ocean continuation"}
		}
	}
	if world.OceanSurface != nil && world.OceanSurface.DirectSunMeanPlane {
		return unsupported{"CPU-only ocean mean-plane direct-sun transport"}
	}
	for index := range world.Things {
		made := world.Things[index]
		if made.Material == nil || made.Material.Through <= 0 {
			continue
		}
	}
	switch {
	case len(world.Fields) > MaxGravityFields:
		return unsupported{"multiple fields"}
	case world.Haze.Density > 0:
		return unsupported{"the world's haze"}
	}
	for _, lamp := range world.Lamps {
		if err := refuseLamp(lamp, len(world.Lamps)); err != nil {
			return err
		}
	}
	return nil
}

// refuseLamp says why a lamp, one of lamps in its world, cannot be drawn
// here, or nothing when it can. A flood is refused among more than
// lampsEvery lamps: there the tracer keeps every flood and picks among
// the rest, and the kernel's picks do not leave room for the floods.
func refuseLamp(lamp thing.Lamp, lamps int) error {
	switch source := lamp.(type) {
	case emit.Lit, emit.Distant, emit.Volume, film.Sun:
	case *emit.Beam:
		if source == nil {
			return unsupported{"a nil beam"}
		}
	case emit.Beams:
		if source.Clock == nil {
			return unsupported{"beams with no clock"}
		}
	case film.Flood:
		if lamps > lampsEvery {
			return unsupported{"a flood among more lamps than are each lit"}
		}
	case film.Spot:
		// a spot is drawn as a point emitter kept to a cone, whose
		// width is the spot's cone less its soft edge. an emitter of no
		// width lights every way, so a spot soft all the way to its
		// middle cannot be said as one; nor can one aimed no way
		if !(source.Cone-source.Soft > 0) {
			return unsupported{"a spot soft to its middle"}
		}
		if !(source.Aim.Len() > 0) {
			return unsupported{"a spot aimed no way"}
		}
	default:
		kind := fmt.Sprintf("a lamp of kind %T", lamp)
		return unsupported{kind}
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
	height int, volumeCells ...int) (*scene, error) {
	return encodeKept(world, trees, eye, lens, at, width, height, nil, volumeCells...)
}

// encodeKept is encodeGrown taking slayer's meshes' face trees from a
// retained drawer's keeper, where it is given one.
func encodeKept(world *thing.World, trees []grown, eye maths.Subspace,
	lens film.Sensor, at time.Duration, width, height int,
	keeper *keptTrees, volumeCells ...int) (*scene, error) {
	return encodeKeptWithInitialMedia(world, trees, eye, lens, at, width,
		height, keeper, nil, volumeCells...)
}

// encodeKeptWithInitialMedia is encodeKept with the explicit camera stack
// used by the CPU tracer. Named optical data is packed only when either the
// world or the supplied stack needs it.
func encodeKeptWithInitialMedia(world *thing.World, trees []grown,
	eye maths.Subspace, lens film.Sensor, at time.Duration, width, height int,
	keeper *keptTrees, media []trace.InitialMedium,
	volumeCells ...int) (*scene, error) {
	if trace.HasClearCloudOverlay(world) &&
		(world.OceanSurface == nil || !world.OceanSurface.DirectMoonMeanPlane) {
		return nil, unsupported{"named clear-water transport with local cloud volumes outside finite lunar route"}
	}
	hasNamedOptics, err := trace.PrepareClearState(world, media)
	if err != nil {
		return nil, err
	}
	moon, err := trace.PlanOceanMoon(world, at)
	if err != nil {
		return nil, unsupported{err.Error()}
	}
	if err := refuse(world); err != nil {
		return nil, err
	}
	encoded := newScene()
	encoded.oceanMoon = moon
	encoded.opticalState = len(media) > 0 || hasNamedOptics
	encoded.oceanSpotScattering = world.OceanSurface != nil &&
		world.OceanSurface.SpotInscatter
	if encoded.opticalState {
		if err := encoded.prepareOpticalState(world, media); err != nil {
			return nil, err
		}
	}
	encoded.keeper = keeper
	encoded.gravity = len(world.Fields) == MaxGravityFields
	if len(volumeCells) > 0 {
		encoded.volumeCells = volumeCells[0]
	}
	encoded.at = at
	encoded.volumeSky = world.VolumeIllumination.Samples()
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
	if err := encoded.reserveIncidentGrid(world.VolumeIllumination); err != nil {
		return nil, err
	}
	if !kernelMade(lens) {
		encoded.lensRays(eye, lens, width, height)
	}
	encoded.camera(eye, lens, width, height)
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
