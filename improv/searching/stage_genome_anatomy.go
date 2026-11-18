package searching

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// AnatomyPlan composes independently placed bodies, closed bands, and open
// sails for an authored individual stage. An omitted plan keeps the seeded
// historical individual constructor.
type AnatomyPlan struct {
	Cores []CorePlan `yaml:"cores,omitempty" json:"cores,omitempty"`
	Bands []BandPlan `yaml:"bands,omitempty" json:"bands,omitempty"`
	Sails []SailPlan `yaml:"sails,omitempty" json:"sails,omitempty"`
}

// AuthoredBeingPlan places one independently constructed member in an
// authored-multitude stage. Identity supplies its repeatable motion only.
type AuthoredBeingPlan struct {
	Identity int          `yaml:"identity" json:"identity"`
	At       [3]float64   `yaml:"at" json:"at"`
	Scale    float64      `yaml:"scale" json:"scale"`
	Anatomy  *AnatomyPlan `yaml:"anatomy" json:"anatomy"`
}

// CorePlan describes one solid body. Rotation uses x, y, z Euler degrees.
type CorePlan struct {
	Form     string     `yaml:"form" json:"form"`
	At       [3]float64 `yaml:"at,omitempty" json:"at,omitempty"`
	Rotation [3]float64 `yaml:"rotation-degrees,omitempty" json:"rotation-degrees,omitempty"`
	SpinRate float64    `yaml:"spin-radians-per-second,omitempty" json:"spin-radians-per-second,omitempty"`
	Radius   float64    `yaml:"radius" json:"radius"`
	Colour   [3]float64 `yaml:"colour,omitempty" json:"colour,omitempty"`
	Glow     [3]float64 `yaml:"glow,omitempty" json:"glow,omitempty"`
}

// BandPlan describes concentric, closed geometry. Radius is the first band's
// centreline; spacing locates later bands. Undulating bands wave along turns.
type BandPlan struct {
	Form         string     `yaml:"form" json:"form"`
	At           [3]float64 `yaml:"at,omitempty" json:"at,omitempty"`
	Rotation     [3]float64 `yaml:"rotation-degrees,omitempty" json:"rotation-degrees,omitempty"`
	SpinRate     float64    `yaml:"spin-radians-per-second,omitempty" json:"spin-radians-per-second,omitempty"`
	Count        int        `yaml:"count" json:"count"`
	Segments     int        `yaml:"segments" json:"segments"`
	Radius       float64    `yaml:"radius" json:"radius"`
	Width        float64    `yaml:"width" json:"width"`
	Thickness    float64    `yaml:"thickness" json:"thickness"`
	Spacing      float64    `yaml:"spacing" json:"spacing"`
	Eccentricity float64    `yaml:"eccentricity,omitempty" json:"eccentricity,omitempty"`
	WarpShare    float64    `yaml:"warp-share,omitempty" json:"warp-share,omitempty"`
	Waves        int        `yaml:"waves,omitempty" json:"waves,omitempty"`
	PhaseDegrees float64    `yaml:"phase-degrees,omitempty" json:"phase-degrees,omitempty"`
	Colour       [3]float64 `yaml:"colour,omitempty" json:"colour,omitempty"`
	Glow         [3]float64 `yaml:"glow,omitempty" json:"glow,omitempty"`
}

// SailPlan describes repeated open sheets. Sweep and camber are shares of
// length and width; twist rotates the sheet through its length.
type SailPlan struct {
	At           [3]float64 `yaml:"at,omitempty" json:"at,omitempty"`
	Rotation     [3]float64 `yaml:"rotation-degrees,omitempty" json:"rotation-degrees,omitempty"`
	SpinRate     float64    `yaml:"spin-radians-per-second,omitempty" json:"spin-radians-per-second,omitempty"`
	Count        int        `yaml:"count" json:"count"`
	Segments     int        `yaml:"segments" json:"segments"`
	SpanSegments int        `yaml:"span-segments" json:"span-segments"`
	Length       float64    `yaml:"length" json:"length"`
	Width        float64    `yaml:"width" json:"width"`
	Sweep        float64    `yaml:"sweep,omitempty" json:"sweep,omitempty"`
	Camber       float64    `yaml:"camber,omitempty" json:"camber,omitempty"`
	TwistDegrees float64    `yaml:"twist-degrees,omitempty" json:"twist-degrees,omitempty"`
	Colour       [3]float64 `yaml:"colour,omitempty" json:"colour,omitempty"`
	Glow         [3]float64 `yaml:"glow,omitempty" json:"glow,omitempty"`
}

// Resource limits keep authored surfaces bounded before allocating meshes.
const (
	maxAnatomyParts        = 64
	maxAnatomyFaces        = 65536
	maxAnatomySegments     = 128
	maxAnatomyRadius       = 1000.0
	maxAnatomyBandCount    = 32
	maxAnatomySailCount    = 32
	maxAnatomyBandWaves    = 16
	maxAnatomySpanSegments = 16
	maxAnatomyColour       = 1.0
	maxAnatomyGlow         = 10.0
	maxAnatomySpinRate     = 20.0
	maxAuthoredPopulation  = 32
	maxAuthoredBeingScale  = 20.0
)

// check rejects empty, unbounded, or contradictory geometry before creation.
func (plan *AnatomyPlan) check() error {
	if plan == nil {
		return fmt.Errorf("anatomy: plan is nil")
	}
	parts := len(plan.Cores)
	for index, band := range plan.Bands {
		if err := band.check(index); err != nil {
			return err
		}
		parts += band.Count
	}
	for index, sail := range plan.Sails {
		if err := sail.check(index); err != nil {
			return err
		}
		parts += sail.Count
	}
	if parts == 0 || parts > maxAnatomyParts {
		return fmt.Errorf("anatomy: component count %d outside 1..%d", parts, maxAnatomyParts)
	}
	for index, core := range plan.Cores {
		if err := core.check(index); err != nil {
			return err
		}
	}
	if faces := plan.faceCountEstimate(); faces > maxAnatomyFaces {
		return fmt.Errorf("anatomy: face count %d exceeds %d", faces, maxAnatomyFaces)
	}
	return nil
}

// componentCount returns the number of solid parts the plan creates.
func (plan *AnatomyPlan) componentCount() int {
	if plan == nil {
		return 0
	}
	count := len(plan.Cores)
	for _, band := range plan.Bands {
		count += band.Count
	}
	for _, sail := range plan.Sails {
		count += sail.Count
	}
	return count
}

// faceCountEstimate returns an upper bound on the faces the plan creates.
func (plan *AnatomyPlan) faceCountEstimate() int {
	if plan == nil {
		return 0
	}
	faces := 0
	for _, core := range plan.Cores {
		if core.Form == "octahedron" {
			faces += 8
		}
	}
	for _, band := range plan.Bands {
		faces += band.Count * band.Segments * 8
	}
	for _, sail := range plan.Sails {
		faces += sail.Count * sail.Segments * sail.SpanSegments * 2
	}
	return faces
}

// check rejects a population member with bad identity, placement, scale or anatomy.
func (being AuthoredBeingPlan) check(index int) error {
	if being.Identity < 0 || being.Identity > 1000000 || !pointOK(being.At) ||
		!bounded(being.Scale, .01, maxAuthoredBeingScale) {
		return fmt.Errorf("anatomy: population member %d has invalid identity, position, or scale", index)
	}
	if err := being.Anatomy.check(); err != nil {
		return fmt.Errorf("anatomy: population member %d: %w", index, err)
	}
	return nil
}

// checkAuthoredPopulation bounds the population size, parts and faces.
func checkAuthoredPopulation(population []AuthoredBeingPlan) error {
	if len(population) < 2 || len(population) > maxAuthoredPopulation {
		return fmt.Errorf("anatomy: population size %d outside 2..%d", len(population), maxAuthoredPopulation)
	}
	parts := 0
	faces := 0
	for index, being := range population {
		if err := being.check(index); err != nil {
			return err
		}
		parts += being.Anatomy.componentCount()
		faces += being.Anatomy.faceCountEstimate()
	}
	if parts > maxAnatomyParts {
		return fmt.Errorf("anatomy: population component count %d exceeds %d", parts, maxAnatomyParts)
	}
	if faces > maxAnatomyFaces {
		return fmt.Errorf("anatomy: population face count %d exceeds %d", faces, maxAnatomyFaces)
	}
	return nil
}

// check rejects a core with an unknown form or out-of-range values.
func (core CorePlan) check(index int) error {
	if core.Form != "sphere" && core.Form != "octahedron" {
		return fmt.Errorf("anatomy: core %d has unknown form %q", index, core.Form)
	}
	if !bounded(core.Radius, .01, maxAnatomyRadius) || !pointOK(core.At) || !rotationOK(core.Rotation) ||
		!bounded(core.SpinRate, -maxAnatomySpinRate, maxAnatomySpinRate) ||
		!lightOK(core.Colour, maxAnatomyColour) || !lightOK(core.Glow, maxAnatomyGlow) {
		return fmt.Errorf("anatomy: core %d has invalid dimensions, placement, or material", index)
	}
	return nil
}

// check rejects a band with an unknown form or out-of-range values.
func (band BandPlan) check(index int) error {
	if band.Form != "annular" && band.Form != "undulating" {
		return fmt.Errorf("anatomy: band %d has unknown form %q", index, band.Form)
	}
	if band.Count < 1 || band.Count > maxAnatomyBandCount || band.Segments < 8 || band.Segments > maxAnatomySegments ||
		!bounded(band.Radius, .01, maxAnatomyRadius) || !bounded(band.Width, .01, maxAnatomyRadius) ||
		!bounded(band.Thickness, .001, maxAnatomyRadius) || !bounded(band.Spacing, .001, maxAnatomyRadius) ||
		!bounded(band.Eccentricity, 0, .8) || !bounded(band.WarpShare, -1, 1) ||
		!bounded(band.PhaseDegrees, -360, 360) || !pointOK(band.At) || !rotationOK(band.Rotation) ||
		!bounded(band.SpinRate, -maxAnatomySpinRate, maxAnatomySpinRate) ||
		!lightOK(band.Colour, maxAnatomyColour) || !lightOK(band.Glow, maxAnatomyGlow) {
		return fmt.Errorf("anatomy: band %d has invalid geometry, placement, or material", index)
	}
	if band.Form == "annular" && (band.Waves != 0 || band.WarpShare != 0) ||
		band.Form == "undulating" && (band.Waves < 1 || band.Waves > maxAnatomyBandWaves) {
		return fmt.Errorf("anatomy: band %d has wave settings incompatible with %q form", index, band.Form)
	}
	outer := band.Radius + float64(band.Count-1)*band.Spacing + band.Thickness/2
	if outer > maxAnatomyRadius {
		return fmt.Errorf("anatomy: band %d exceeds maximum radial extent", index)
	}
	return nil
}

// check rejects a sail with out-of-range geometry or material values.
func (sail SailPlan) check(index int) error {
	if sail.Count < 1 || sail.Count > maxAnatomySailCount || sail.Segments < 8 || sail.Segments > maxAnatomySegments ||
		sail.SpanSegments < 1 || sail.SpanSegments > maxAnatomySpanSegments ||
		!bounded(sail.Length, .01, maxAnatomyRadius) || !bounded(sail.Width, .01, maxAnatomyRadius) ||
		!bounded(sail.Sweep, -2, 2) || !bounded(sail.Camber, -1, 1) ||
		!bounded(sail.TwistDegrees, -180, 180) || !pointOK(sail.At) || !rotationOK(sail.Rotation) ||
		!bounded(sail.SpinRate, -maxAnatomySpinRate, maxAnatomySpinRate) ||
		!lightOK(sail.Colour, maxAnatomyColour) || !lightOK(sail.Glow, maxAnatomyGlow) {
		return fmt.Errorf("anatomy: sail %d has invalid geometry, placement, or material", index)
	}
	if sail.Length*(1+math.Abs(sail.Sweep)/2) > maxAnatomyRadius {
		return fmt.Errorf("anatomy: sail %d exceeds maximum swept extent", index)
	}
	return nil
}

// rotationOK reports whether every Euler angle lies within a full turn.
func rotationOK(rotation [3]float64) bool {
	for _, angle := range rotation {
		if !bounded(angle, -360, 360) {
			return false
		}
	}
	return true
}

// lightOK reports whether every channel lies between zero and maximum.
func lightOK(light [3]float64, maximum float64) bool {
	for _, channel := range light {
		if !bounded(channel, 0, maximum) {
			return false
		}
	}
	return true
}

// anatomyPose returns the pose for a position and Euler degrees.
func anatomyPose(at, rotation [3]float64) maths.Subspace {
	orientation := maths.About(maths.XAxis, rotation[0]*maths.Degree).
		Then(maths.About(maths.YAxis, rotation[1]*maths.Degree)).
		Then(maths.About(maths.ZAxis, rotation[2]*maths.Degree))
	return maths.NewSubspace(maths.V(at[0], at[1], at[2]), orientation, 1)
}

// buildAuthoredIndividual adds the plan's cores, bands and sails to a being.
func (s *heardStage) buildAuthoredIndividual(b *crowdBeing, plan *AnatomyPlan,
	cache *anatomyMeshCache,
) error {
	if err := plan.check(); err != nil {
		return err
	}
	add := func(solid shape.Shape, material *thing.Material, pose maths.Subspace, rate float64) {
		part := s.world.Add(thing.Thing{Shape: solid, Material: material})
		b.parts = append(b.parts, crowdPart{part: part, local: pose,
			glow: material.Glow, axis: maths.YAxis, rate: rate})
	}
	for _, core := range plan.Cores {
		material := &thing.Material{Name: "authored individual core",
			Colour: maths.RGB(core.Colour[0], core.Colour[1], core.Colour[2]),
			Glow:   maths.RGB(core.Glow[0], core.Glow[1], core.Glow[2])}
		pose := anatomyPose(core.At, core.Rotation)
		pose.Scale = core.Radius
		if core.Form == "sphere" {
			add(shape.Sphere{Radius: 1}, material, pose, core.SpinRate)
			continue
		}
		if core.Form == "octahedron" {
			octahedron, err := cache.octahedron()
			if err != nil {
				return err
			}
			add(octahedron, material, pose, core.SpinRate)
		}
	}
	for _, band := range plan.Bands {
		for index := range band.Count {
			solid, err := anatomyBand(band, band.Radius+float64(index)*band.Spacing)
			if err != nil {
				return err
			}
			material := &thing.Material{Name: "authored individual band",
				Colour: maths.RGB(band.Colour[0], band.Colour[1], band.Colour[2]),
				Glow:   maths.RGB(band.Glow[0], band.Glow[1], band.Glow[2])}
			add(solid, material, anatomyPose(band.At, band.Rotation), band.SpinRate)
		}
	}
	for _, sail := range plan.Sails {
		for index := range sail.Count {
			key := sailGeometryKeyOf(sail)
			solid, err := cache.sail(key, sail)
			if err != nil {
				return err
			}
			material := &thing.Material{Name: "authored individual sail",
				Colour: maths.RGB(sail.Colour[0], sail.Colour[1], sail.Colour[2]),
				Glow:   maths.RGB(sail.Glow[0], sail.Glow[1], sail.Glow[2])}
			pose := anatomyPose(sail.At, sail.Rotation)
			angle := 2 * math.Pi * float64(index) / float64(sail.Count)
			pose.Orientation = maths.About(maths.YAxis, angle).Then(pose.Orientation)
			add(solid, material, pose, sail.SpinRate)
		}
	}
	return nil
}

type sailGeometryKey struct {
	segments, spanSegments int
	length, width          float64
	sweep, camber, twist   float64
}

// anatomyMeshCache is scoped to one authored stage, so identical immutable
// components share storage while each thing keeps its own material and pose.
type anatomyMeshCache struct {
	octa  *mesh.Mesh
	sails map[sailGeometryKey]*mesh.Mesh
}

// newAnatomyMeshCache returns an empty cache for one authored stage.
func newAnatomyMeshCache() *anatomyMeshCache {
	return &anatomyMeshCache{sails: make(map[sailGeometryKey]*mesh.Mesh)}
}

// anatomyCacheFor preserves prototype sharing unless an authored stage
// explicitly requests separate geometry objects.
func anatomyCacheFor(plan *StagePlan) *anatomyMeshCache {
	if plan != nil && plan.ShareAnatomyPrototypes != nil && !*plan.ShareAnatomyPrototypes {
		return nil
	}
	return newAnatomyMeshCache()
}

// octahedron returns the shared octahedron mesh, making it on first use.
func (cache *anatomyMeshCache) octahedron() (*mesh.Mesh, error) {
	if cache == nil {
		return anatomyOctahedron()
	}
	if cache.octa == nil {
		made, err := anatomyOctahedron()
		if err != nil {
			return nil, err
		}
		cache.octa = made
	}
	return cache.octa, nil
}

// sail returns the cached sail mesh for key, making it on a miss.
func (cache *anatomyMeshCache) sail(key sailGeometryKey,
	plan SailPlan,
) (*mesh.Mesh, error) {
	if cache == nil {
		return anatomySail(plan)
	}
	if made := cache.sails[key]; made != nil {
		return made, nil
	}
	made, err := anatomySail(plan)
	if err != nil {
		return nil, err
	}
	cache.sails[key] = made
	return made, nil
}

// sailGeometryKeyOf returns the fields that decide a sail's geometry.
func sailGeometryKeyOf(sail SailPlan) sailGeometryKey {
	return sailGeometryKey{segments: sail.Segments,
		spanSegments: sail.SpanSegments, length: sail.Length, width: sail.Width,
		sweep: sail.Sweep, camber: sail.Camber, twist: sail.TwistDegrees}
}

// anatomyOctahedron builds a unit octahedron with outward faces.
func anatomyOctahedron() (*mesh.Mesh, error) {
	vertices := []maths.Vec{
		maths.V(1, 0, 0), maths.V(-1, 0, 0), maths.V(0, 1, 0),
		maths.V(0, -1, 0), maths.V(0, 0, 1), maths.V(0, 0, -1),
	}
	var faces [][3]int
	for _, vertical := range []int{2, 3} {
		for _, horizontal := range []int{0, 1} {
			for _, depth := range []int{4, 5} {
				face := [3]int{vertical, horizontal, depth}
				a, b, c := vertices[face[0]], vertices[face[1]], vertices[face[2]]
				if b.Sub(a).Cross(c.Sub(a)).Dot(a.Add(b).Add(c)) < 0 {
					face[1], face[2] = face[2], face[1]
				}
				faces = append(faces, face)
			}
		}
	}
	return mesh.New(vertices, faces, nil, false)
}

// anatomyBand builds one closed band ring around the vertical axis.
func anatomyBand(plan BandPlan, radius float64) (*mesh.Mesh, error) {
	vertices := make([]maths.Vec, 0, plan.Segments*4)
	faces := make([][3]int, 0, plan.Segments*8)
	phase := plan.PhaseDegrees * maths.Degree
	for segment := 0; segment < plan.Segments; segment++ {
		angle := 2 * math.Pi * float64(segment) / float64(plan.Segments)
		radial := radius * (1 + plan.Eccentricity*math.Cos(angle))
		warp := 0.0
		if plan.Form == "undulating" {
			warp = plan.Width * plan.WarpShare * math.Sin(float64(plan.Waves)*angle+phase)
		}
		for _, across := range [][2]float64{
			{-plan.Thickness / 2, -plan.Width / 2},
			{plan.Thickness / 2, -plan.Width / 2},
			{plan.Thickness / 2, plan.Width / 2},
			{-plan.Thickness / 2, plan.Width / 2},
		} {
			r := radial + across[0]
			vertices = append(vertices, maths.V(r*math.Cos(angle), across[1]+warp, r*math.Sin(angle)))
		}
	}
	for segment := 0; segment < plan.Segments; segment++ {
		for corner := 0; corner < 4; corner++ {
			a := segment*4 + corner
			b := segment*4 + (corner+1)%4
			c := ((segment+1)%plan.Segments)*4 + corner
			d := ((segment+1)%plan.Segments)*4 + (corner+1)%4
			faces = append(faces, [3]int{a, b, c}, [3]int{b, d, c})
		}
	}
	return mesh.New(vertices, faces, nil, false)
}

// anatomySail builds one open sheet from its segment and span grid.
func anatomySail(plan SailPlan) (*mesh.Mesh, error) {
	columns := plan.Segments + 1
	rows := plan.SpanSegments + 1
	vertices := make([]maths.Vec, 0, columns*rows)
	faces := make([][3]int, 0, plan.Segments*plan.SpanSegments*2)
	for segment := 0; segment <= plan.Segments; segment++ {
		u := float64(segment) / float64(plan.Segments)
		for span := 0; span <= plan.SpanSegments; span++ {
			v := float64(span)/float64(plan.SpanSegments) - .5
			x := plan.Length*u + plan.Sweep*plan.Length*math.Abs(v)*u
			y := plan.Width * v
			z := plan.Camber * plan.Width * math.Sin(math.Pi*u) * (1 - 4*v*v)
			twist := plan.TwistDegrees * maths.Degree * u
			y, z = y*math.Cos(twist)-z*math.Sin(twist),
				y*math.Sin(twist)+z*math.Cos(twist)
			vertices = append(vertices, maths.V(x, y, z))
		}
	}
	for segment := 0; segment < plan.Segments; segment++ {
		for span := 0; span < plan.SpanSegments; span++ {
			a := segment*rows + span
			b := a + 1
			c := (segment+1)*rows + span
			d := c + 1
			faces = append(faces, [3]int{a, b, c}, [3]int{b, d, c})
		}
	}
	return mesh.New(vertices, faces, nil, false)
}
