package named

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
)

// surfaceDistrict adds varied inhabited structures around the selected city.
// Every specimen is generated with its own seed, bent onto the native floor,
// and remains in the same rotating frame as the camera and surrounding land.
func (one hoop) surfaceDistrict(world *thing.World, geography *biome.World, centre maths.Vec, seed uint64, day time.Duration, count int, city bool, existing ...orbitalFootprint) ([]orbitalFootprint, error) {
	return one.surfaceDistrictDirected(world, geography, centre, seed, day, count, city, false, existing...)
}

// surfaceDistrictDirected places district footprints on the surface. Each
// district must fit clear of the footprints placed before it.
func (one hoop) surfaceDistrictDirected(world *thing.World, geography *biome.World, centre maths.Vec, seed uint64, day time.Duration, count int, city, flyby bool, existing ...orbitalFootprint) ([]orbitalFootprint, error) {
	styles := []manufacture.MacrostructureStyle{manufacture.TerracedArcology, manufacture.TwistedSpires, manufacture.InhabitedBridge}
	placed := []orbitalFootprint{{radius: 1050000}}
	if city {
		placed = append(placed, orbitalFootprint{offset: maths.V(0, 0, -1400000), radius: 905100})
	}
	occupied := append([]orbitalFootprint(nil), placed...)
	occupied = append(occupied, existing...)
	next := 0
	previous := -1
	for i := 0; i < count; i++ {
		own := seed + uint64(i+1)*100003
		choice := i % len(styles)
		if flyby {
			choice = orbitalDistrictFamily(own, i, previous, city)
		}
		previous = choice
		spec := manufacture.DefaultMacrostructure(styles[choice])
		size := .45 + .8*noise.Hash(own, 2)
		wide, deep, tall := 1.0, 1.0, 1.0
		if flyby {
			wide = .75 + .5*noise.Hash(own, 10)
			deep = .75 + .5*noise.Hash(own, 11)
			tall = .75 + .5*noise.Hash(own, 12)
		}
		if spec.Style == manufacture.InhabitedBridge {
			spec.Width, spec.Depth, spec.Height = measure.Metres(800000*size*wide), measure.Metres(140000*size*deep), measure.Metres(160000*size*tall)
		} else {
			spec.Width, spec.Depth, spec.Height = measure.Metres(160000*size*wide), measure.Metres(135000*size*deep), measure.Metres(240000*size*tall)
		}
		spec.Wildness, spec.LivingCover = .85, true
		spec.Twist = measure.Degrees(50 + 120*noise.Hash(own, 3))
		kind, err := manufacture.Macrostructure(spec)
		if err != nil {
			return nil, err
		}
		// The circumscribed circle also covers the independently turned footprint.
		radius := .52 * math.Max(spec.Width.InMetres(), spec.Depth.InMetres())
		if flyby {
			radius = .5 * math.Hypot(spec.Width.InMetres(), spec.Depth.InMetres())
		}
		var offset maths.Vec
		if i == 0 && city && flyby {
			var ok bool
			offset, ok = one.districtFlybyLand(geography, centre, radius, occupied)
			if !ok {
				// Keep the same corridor constraints for the fallback search.
				// An ordinary district location may lie across the camera path.
				for candidate := 0; candidate < 10000; candidate++ {
					where := orbitalDistrictOffset(seed, candidate)
					if one.districtFits(geography, centre, where, radius, occupied, true) && orbitalFlybyClear(where, radius, occupied) {
						offset, next, ok = where, candidate+1, true
						break
					}
				}
			}
			if !ok {
				return nil, fmt.Errorf("orbital district could not find a dry landmark with a clear flyby")
			}
		}
		if offset == (maths.Vec{}) {
			var candidate int
			offset, candidate, err = one.districtLand(geography, centre, seed, next, radius, occupied, city && flyby)
			if err != nil {
				return nil, err
			}
			next = candidate + 1
		}
		site := one.surfacePoint(centre, offset)
		turn := maths.Orientation{}
		if flyby {
			turn = maths.About(maths.YAxis, 2*math.Pi*noise.Hash(own, 13))
		}
		if err = one.surfaceSpecimenTurned(world, kind, own, site, day, turn); err != nil {
			return nil, err
		}
		placed = append(placed, orbitalFootprint{offset: offset, radius: radius + 50000})
		occupied = append(occupied, placed[len(placed)-1])
	}
	// A local volcanic massif and mountain accompany the inhabited district.
	// These are explicit surface landmarks rather than distant colour patches.
	for i, style := range []manufacture.LandformStyle{manufacture.CalderaVolcano, manufacture.MountainMassif} {
		spec := manufacture.DefaultLandform(style)
		spec.Radius, spec.Height, spec.Ruggedness = measure.Metres(300000), measure.Metres(24000), 1
		if style == manufacture.CalderaVolcano {
			spec.CraterRadius, spec.CraterDepth = measure.Metres(45000), measure.Metres(9000)
		}
		kind, err := manufacture.Landform(spec)
		if err != nil {
			return nil, err
		}
		offset, candidate, err := one.districtLand(geography, centre, seed, next, spec.Radius.InMetres(), occupied, city && flyby)
		if err != nil {
			return nil, err
		}
		next = candidate + 1
		site := one.surfacePoint(centre, offset)
		if err = one.surfaceSpecimen(world, kind, seed+uint64(i+1)*700001, site, day); err != nil {
			return nil, err
		}
		placed = append(placed, orbitalFootprint{offset: offset, radius: spec.Radius.InMetres() + 50000})
		occupied = append(occupied, placed[len(placed)-1])
	}
	return placed, nil
}

// Pick from the three existing construction families without cycling them or
// putting identical families next to one another. Geometry, dimensions and
// orientation are separately seeded for every specimen.
func orbitalDistrictFamily(seed uint64, index, previous int, city bool) int {
	choice := min(2, int(noise.Hash(seed, 8)*3))
	if index == 0 && city && choice == 2 {
		choice = 1
	}
	if choice == previous {
		choice = (choice + 1 + int(noise.Hash(seed, 9)*2)) % 3
	}
	return choice
}

// districtLand chooses the next non-overlapping footprint and samples its
// interior across the retained biome map, rather than only its center point.
func (one hoop) districtLand(geography *biome.World, centre maths.Vec, seed uint64, start int, radius float64, placed []orbitalFootprint, city ...bool) (maths.Vec, int, error) {
	for candidate := start; candidate < start+10000; candidate++ {
		offset := orbitalDistrictOffset(seed, candidate)
		if one.districtFits(geography, centre, offset, radius, placed, len(city) > 0 && city[0]) {
			return offset, candidate, nil
		}
	}
	return maths.Vec{}, 0, fmt.Errorf("orbital district could not find dry, disjoint footprints")
}

// A landmark near the park's west side becomes a readable destination from
// the camera corridor. The generated candidate is still dry and disjoint.
func (one hoop) districtFlybyLand(geography *biome.World, centre maths.Vec, radius float64, placed []orbitalFootprint) (maths.Vec, bool) {
	for _, x := range []float64{-900000, -1050000, -1200000, -1350000, -1500000} {
		for _, z := range []float64{-1400000, -1150000, -1650000, -900000, -1900000} {
			offset := maths.V(x, 0, z)
			if one.districtFits(geography, centre, offset, radius, placed, true) && orbitalFlybyClear(offset, radius, placed) {
				return offset, true
			}
		}
	}
	return maths.Vec{}, false
}

// Check the actual curved route's ground projection against existing landmark
// footprints. The city is traversed intentionally after the vertical climb.
func orbitalFlybyClear(to maths.Vec, radius float64, placed []orbitalFootprint) bool {
	park := manufacture.DefaultClusteredCity().Parks[0]
	from := manufacture.CityParkLongPathAt(park, .5003).Scale(orbitalCityScale).Add(orbitalCityOffset)
	end := to.Add(maths.V(radius+50000, 60000, -100000))
	control := to.Add(maths.V(radius+250000, 200000, -120000))
	for index, other := range placed {
		if index == 1 {
			continue
		}
		last := from
		for step := 1; step <= 128; step++ {
			at := orbitalBezier(from, from, control, end, float64(step)/128)
			delta, part := at.Sub(last), other.offset.Sub(last)
			length := delta.X*delta.X + delta.Z*delta.Z
			u := 0.0
			if length > 0 {
				u = math.Max(0, math.Min(1, (part.X*delta.X+part.Z*delta.Z)/length))
			}
			x, z := last.X+u*delta.X, last.Z+u*delta.Z
			if math.Hypot(other.offset.X-x, other.offset.Z-z) < other.radius+50010 {
				return false
			}
			last = at
		}
	}
	return true
}

// districtFits reports whether a district at offset is on land and clear of
// the footprints already placed.
func (one hoop) districtFits(geography *biome.World, centre, offset maths.Vec, radius float64, placed []orbitalFootprint, city bool) bool {
	if !one.surfaceFootprintLand(geography, centre, offset, radius) {
		return false
	}
	for index, other := range placed {
		if city && index == 1 {
			// The city occupies a square; its conservative relief circle
			// need not push the landmark hundreds of kilometres away.
			dx := math.Max(0, math.Abs(offset.X)-640000)
			dz := math.Max(0, math.Abs(offset.Z+1400000)-640000)
			if math.Hypot(dx, dz) < radius+50000 {
				return false
			}
		} else if math.Hypot(offset.X-other.offset.X, offset.Z-other.offset.Z) < radius+other.radius+50000 {
			return false
		}
	}
	return true
}

// surfaceFootprintLand reports whether a square footprint is dry land, sampling
// a 9 by 9 grid across it.
func (one hoop) surfaceFootprintLand(geography *biome.World, centre, offset maths.Vec, radius float64) bool {
	if geography == nil {
		return true
	}
	for ix := -4; ix <= 4; ix++ {
		for iz := -4; iz <= 4; iz++ {
			x, z := radius*float64(ix)/4, radius*float64(iz)/4
			at := one.surfacePoint(centre, offset.Add(maths.V(x, 0, z)))
			if math.Abs(at.Y) > bandWidth/2 || geography.WaterAt(at) {
				return false
			}
		}
	}
	return true
}

// orbitalDistrictOffset spreads district centres around the orbit on a
// golden-angle spiral, seeded by the district index.
func orbitalDistrictOffset(seed uint64, i int) maths.Vec {
	own := seed + uint64(i+1)*100003
	a := float64(i)*2.399963229728653 + noise.Hash(own, 4)*.25
	r := 1.1 + .25*math.Sqrt(float64(i))
	return maths.V(r*math.Cos(a)*1e6, 0, r*math.Sin(a)*1e6)
}

// reverseOrbitalSun places the same ground site into the star's night side;
// emissive cities are unchanged, and the dim space fill still lights outlines.
func reverseOrbitalSun(world *thing.World) {
	for i, lamp := range world.Lamps {
		sun, ok := lamp.(film.Sun)
		if ok && sun.Toward == maths.V(nearSunX, nearSunY, nearSunZ) {
			sun.Toward = sun.Toward.Neg()
			world.Lamps[i] = sun
		}
	}
}

// orbitalWindows gives the glazed surfaces irregular, seeded inhabited lights
// in their own metre coordinates. They stay visible when the sun sets.
type orbitalWindows struct{ seed uint64 }

// At returns the window light at a point, lit or dark by a seeded hash of its
// cell.
func (windows orbitalWindows) At(p maths.Vec) maths.Light {
	column, row, depth := int64(math.Floor(p.X/1800)), int64(math.Floor(p.Y/1200)), int64(math.Floor(p.Z/1800))
	if noise.Hash2(windows.seed+uint64(depth)*7919, column, row) < .47 {
		return maths.Light{}
	}
	return maths.RGB(.18, .075, .022)
}
