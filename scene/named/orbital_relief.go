package named

import (
	"math"
	"time"

	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// orbitalRelief spans 8,000 km in metre coordinates. Its finer facets and
// multiscale rock paint make land visibly dimensional at flythrough scale.
const orbitalReliefSteps = 320
const orbitalReliefHalf = 4000000.0

// surfaceRelief keeps existing geographic colour and lights while providing
// actual land geometry. Sea, rivers and lakes remain the existing floor.
type orbitalFootprint struct {
	offset maths.Vec
	radius float64
}

// surfaceRelief adds sampled relief terrain under the orbital surface, leaving
// out the footprints already placed.
func (one hoop) surfaceRelief(world *thing.World, geography *biome.World, floor *thing.Material, centre maths.Vec, seed uint64, day time.Duration, footprints []orbitalFootprint) error {
	points := make([]maths.Vec, 0, (orbitalReliefSteps+1)*(orbitalReliefSteps+1))
	usable := make([]bool, 0, cap(points))
	for row := 0; row <= orbitalReliefSteps; row++ {
		z := -orbitalReliefHalf + 2*orbitalReliefHalf*float64(row)/orbitalReliefSteps
		for col := 0; col <= orbitalReliefSteps; col++ {
			x := -orbitalReliefHalf + 2*orbitalReliefHalf*float64(col)/orbitalReliefSteps
			base := one.surfacePoint(centre, maths.V(x, 0, z))
			clearance := orbitalReliefClearance(x, z, footprints)
			land := !geography.WaterAt(base)
			above := geography.Above(base)
			usable = append(usable, clearance > 0 && land && above > 0)
			altitude := orbitalReliefHeight(x, z, seed, clearance, above)
			points = append(points, one.surfacePoint(centre, maths.V(x, altitude, z)))
		}
	}
	faces := make([][3]int, 0, orbitalReliefSteps*orbitalReliefSteps*2)
	for row := 0; row < orbitalReliefSteps; row++ {
		for col := 0; col < orbitalReliefSteps; col++ {
			a := row*(orbitalReliefSteps+1) + col
			b, c, d := a+1, a+orbitalReliefSteps+1, a+orbitalReliefSteps+2
			if usable[a] && usable[b] && usable[c] {
				faces = append(faces, [3]int{a, c, b})
			}
			if usable[b] && usable[c] && usable[d] {
				faces = append(faces, [3]int{b, c, d})
			}
		}
	}
	if len(faces) == 0 {
		return nil
	}
	land, err := mesh.New(points, faces, nil, false)
	if err != nil {
		return err
	}
	look := *floor
	look.Name = "orbital dimensional terrain"
	look.Texture = orbitalReliefPaint{base: floor.Texture, ring: one, seed: seed}
	world.Add(thing.Thing{Shape: land, Material: &look, Motion: one.turning(day)})
	return nil
}

// orbitalReliefClearance tapers the terrain to zero around each footprint and
// at the patch boundary. Faces entering a footprint are omitted entirely;
// the original floor remains beneath the building instead of burying it.
func orbitalReliefClearance(x, z float64, footprints []orbitalFootprint) float64 {
	edge := orbitalReliefHalf - math.Max(math.Abs(x), math.Abs(z))
	clearance := edge
	for _, area := range footprints {
		clearance = math.Min(clearance, math.Hypot(x-area.offset.X, z-area.offset.Z)-area.radius)
	}
	return clearance
}

// orbitalReliefHeight fades smoothly to the floor at coastal and footprint
// boundaries. No arbitrary low-detail substitution is made near the camera.
func orbitalReliefHeight(x, z float64, seed uint64, clearance, aboveSea float64) float64 {
	if clearance <= 0 || aboveSea <= 0 {
		return 0
	}
	coast := maths.Smooth(0, .08, aboveSea)
	edge := maths.Smooth(0, 550000, clearance)
	continental := .5 + .5*noise.Fractal(seed+4001, x/850000, 0, z/850000, 4)
	ridged := 1 - math.Abs(noise.Fractal(seed+4002, x/310000, 1, z/310000, 4))
	fine := .5 + .5*noise.Fractal(seed+4003, x/75000, 2, z/75000, 3)
	height := 5000 + 52000*continental*ridged*ridged + 5500*fine
	return math.Max(0, height) * coast * edge
}

// orbitalReliefPaint retains the original land/water palette and places
// mottled rock and high snow over the raised ground at multiple scales.
type orbitalReliefPaint struct {
	base texture.Texture
	ring hoop
	seed uint64
}

// orbitalGroundPaint adds soil, vegetation and mineral variation inside the
// flythrough district, including flat clearances beneath structures. The
// existing biome still decides land and water and the existing city lights.
type orbitalGroundPaint struct {
	base      texture.Texture
	ring      hoop
	site      maths.Vec
	seed      uint64
	geography *biome.World
}

// At returns the ground light at a point on the relief.
func (paint orbitalGroundPaint) At(at maths.Vec) maths.Light {
	return paint.colour(at, paint.base.At(at))
}

// Within returns the ground light at a point, filtered over a footprint.
func (paint orbitalGroundPaint) Within(at maths.Vec, footprint float64) maths.Light {
	return paint.colour(at, texture.Sample(paint.base, at, footprint))
}

// colour varies base by a coarse and a fine noise field near the site, and
// leaves water unchanged.
func (paint orbitalGroundPaint) colour(at maths.Vec, base maths.Light) maths.Light {
	if paint.geography.WaterAt(at) {
		return base
	}
	a := math.Remainder(math.Atan2(at.X, at.Z)-math.Atan2(paint.site.X, paint.site.Z), 2*math.Pi)
	x, z := a*paint.ring.radius*1e6, (at.Y-paint.site.Y)*1e6
	blend := 1 - maths.Smooth(3500000, orbitalReliefHalf, math.Max(math.Abs(x), math.Abs(z)))
	if blend <= 0 {
		return base
	}
	coarse := noise.Fractal(paint.seed+5021, x/180000, 0, z/180000, 4)
	fine := noise.Fractal(paint.seed+5022, x/35000, 1, z/35000, 3)
	variation := math.Max(.55, math.Min(1.3, 1+.37*coarse+.14*fine))
	return base.Scale(1 - blend + blend*variation)
}

// At returns the relief light at a point.
func (paint orbitalReliefPaint) At(at maths.Vec) maths.Light {
	return paint.colour(at, paint.base.At(at))
}

// Within returns the relief light at a point, filtered over a footprint.
func (paint orbitalReliefPaint) Within(at maths.Vec, footprint float64) maths.Light {
	return paint.colour(at, texture.Sample(paint.base, at, footprint))
}

// colour shades base by height above the ring, with seeded grain and rock.
func (paint orbitalReliefPaint) colour(at maths.Vec, base maths.Light) maths.Light {
	angle := math.Atan2(at.X, at.Z)
	height := (paint.ring.surfaceRadius(angle) - math.Hypot(at.X, at.Z)) * 1e6
	grain := noise.Fractal(paint.seed+4010, at.X*20, at.Y*20, at.Z*20, 4)
	rock := maths.Smooth(10000, 32000, height) * (0.55 + 0.25*grain)
	rock = math.Max(0, math.Min(.8, rock))
	col := base.Scale(1 - rock).Add(maths.RGB(.22, .18, .14).Scale(rock))
	snow := maths.Smooth(36000, 56000, height) * (.65 + .2*grain)
	snow = math.Max(0, math.Min(.85, snow))
	return col.Scale(1 - snow).Add(maths.RGB(.72, .76, .77).Scale(snow))
}
