package skylight

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"testing"
)

// The slabs the two walks are held to each other on.
const (
	slabBand    = math.Pi / 18 // defined: the subsolar band, ten degrees
	slabZeniths = 18           // defined: bands of ten degrees
	slabViews   = 40           // picked: view cosines a twentieth wide
	slabPhotons = 1000000      // picked: thirty thousand on the band
	slabWalks   = 1000000      // picked: the plane walk's photons
	slabSeed    = 3            // picked: any seed but the sphere's
	thickSlab   = 16           // brief: the thickest slab
)

// Where a photon leaves the plane slab.
const (
	floorDiffuse = iota // defined: out of the floor, scattered
	floorDirect         // defined: out of the floor, never scattered
	slabTop             // defined: back out of the top
)

// slab follows photons through a plane slab of optical depth depth that
// scatters alike every way and absorbs nothing. It shares nothing with
// the sphere's walk but the phase draw.
//
// Each photon enters the top going down at a zenith drawn as photons
// fall on the sphere's subsolar band: its sine squared even from nought
// to the band's. The shares are of photons leaving the floor scattered,
// leaving it unscattered, and leaving the top.
func slab(depth float64) (diffuse, direct, reflected float64) {
	random := rand.New(rand.NewPCG(slabSeed, slabSeed))
	spread := math.Pow(math.Sin(slabBand), two)
	for range slabWalks {
		way := math.Sqrt(one - spread*random.Float64())
		switch slabPhoton(random, depth, way) {
		case floorDiffuse:
			diffuse++
		case floorDirect:
			direct++
		default:
			reflected++
		}
	}
	return diffuse / slabWalks, direct / slabWalks, reflected / slabWalks
}

// slabPhoton walks one photon from the top of the slab, going down at
// a cosine way from the downward vertical, and says where it left.
func slabPhoton(random *rand.Rand, depth, way float64) int {
	into, scattered := 0.0, false
	for {
		into += way * -math.Log1p(-random.Float64())
		if into >= depth && scattered {
			return floorDiffuse
		}
		if into >= depth {
			return floorDirect
		}
		if into <= none {
			return slabTop
		}
		// alike every way, the new way is uniform on the sphere, so its
		// cosine to the vertical is distributed as the turn's
		way = henyeyCosine(none, random.Float64())
		scattered = true
	}
}

// spheres are the slabs on the large planet, each walked once.
var spheres = map[float64]func() (*Field, error){}

// sphereOf is the field of a slab of depth on the large planet: an
// isotropic, conservative cloud a kilometre thick over black ground,
// kept at the ground and the top, walked the first time it is asked.
func sphereOf(depth float64) (*Field, error) {
	if _, walked := spheres[depth]; !walked {
		spheres[depth] = sync.OnceValues(func() (*Field, error) {
			cloud := Shell{ParticleExtinction: depth / thinTop,
				ParticleAlbedo: one}
			grid := Grid{Heights: []float64{none, thinTop},
				Zeniths: slabZeniths, Views: slabViews, Azimuths: one}
			field, _, err := Solve(uniformColumn(farRadius, thinTop, cloud,
				none), grid, slabPhotons, seedOf, workersOf)
			return field, err
		})
	}
	return spheres[depth]()
}

// fluxes is the light through the subsolar band at the kept height,
// going down or up, from the radiance by view: each bin's radiance
// times its middle cosine times its solid angle. Its variance is each
// bin's part squared over the crossings behind it, summed.
func fluxes(field *Field, keep int, down bool) (flux, variance float64) {
	solid := two / float64(field.views) * fullTurn /
		float64(field.azimuths)
	for v := range field.views {
		cosine := -one + (float64(v)+half)*two/float64(field.views)
		for a := range field.azimuths {
			index := field.index(keep, none, v, a)
			counts := float64(field.counts[index])
			part := field.radiance[index] * math.Abs(cosine) * solid
			if (cosine > none) == down && counts > none {
				flux += part
				variance += part * part / counts
			}
		}
	}
	return flux, variance
}

// a share is a fraction of the light and its standard error.
type share struct{ value, error float64 }

// sphereShares is what the sphere's subsolar band sends down through
// the ground and up through the top, each as a share of what falls on
// the band. What falls, per unit of the band's area, is the zenith's
// mean cosine over it: one and the cosine of its edge, over two.
func sphereShares(field *Field) (down, up share) {
	falling := (one + math.Cos(slabBand)) / two
	downward, downVariance := fluxes(field, none, true)
	upward, upVariance := fluxes(field, one, false)
	return share{downward / falling, math.Sqrt(downVariance) / falling},
		share{upward / falling, math.Sqrt(upVariance) / falling}
}

// agree is nil when the sphere's share and the plane walk's are within
// sigmas of their combined standard error. Each plane photon leaves by
// one way or another, so its share's variance is p(1-p)/n.
func agree(sphere share, plane float64) error {
	planeError := math.Sqrt(plane * (one - plane) / slabWalks)
	allowed := sigmas * math.Hypot(sphere.error, planeError)
	if !(math.Abs(sphere.value-plane) <= allowed) {
		return fmt.Errorf("sphere %.5g, plane %.5g, allowed %.3g",
			sphere.value, plane, allowed)
	}
	return nil
}

// the spherical walk on a planet so large it is flat sends through and
// back what an independent plane walk does, for conservative isotropic
// slabs of optical depth one and four, and sixteen when not short, lit
// near the vertical.
func TestSlabsAgree(t *testing.T) {
	depths := []float64{1, 4} // brief: two of the three depths
	if !testing.Short() {
		depths = append(depths, thickSlab)
	}
	for _, depth := range depths {
		field, err := sphereOf(depth)
		if err != nil {
			t.Fatal(err)
		}
		down, up := sphereShares(field)
		diffuse, direct, reflected := slab(depth)
		t.Logf("depth %g: through %+v plane %.5g, back %+v plane %.5g, "+
			"direct %.5g", depth, down, diffuse, up, reflected, direct)
		if err := agree(down, diffuse); err != nil {
			t.Errorf("depth %g, through: %v", depth, err)
		}
		if err := agree(up, reflected); err != nil {
			t.Errorf("depth %g, back: %v", depth, err)
		}
	}
}

// the anti-test: the plane walk's light through the floor counted with
// the sun's direct beam is not the sphere's diffuse light, so the check
// would catch the direct beam tallied as sky.
func TestDirectLightCountedAsSkyIsCaught(t *testing.T) {
	depth := 1.0 // brief: the thinnest slab
	field, err := sphereOf(depth)
	if err != nil {
		t.Fatal(err)
	}
	down, _ := sphereShares(field)
	diffuse, direct, _ := slab(depth)
	if agree(down, diffuse+direct) == nil {
		t.Error("the direct beam passes as sky")
	}
}
