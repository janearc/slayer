package named

import (
	"fmt"
	"io"
	"math"
	"sort"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// orbitalSurfaceSite carries a surface camera with the same ring as its scenery.
type orbitalSurfaceSite struct {
	frame          maths.Subspace
	motion         thing.Motion
	tick           func(time.Duration)
	ring           hoop
	centre         maths.Vec
	cityTour       bool
	landmark       maths.Vec
	landmarkRadius float64
}

type orbitalFocusCandidate struct {
	site         orbitalSurfaceSite
	at           maths.Vec
	seed         uint64
	illumination float64
}

// camera returns the surface camera's place at time at. The camera stands at
// a fixed offset from the site's frame and looks at the frame's origin. It
// moves with the site's motion.
func (site orbitalSurfaceSite) camera(at time.Duration) (maths.Subspace, error) {
	from, to := site.frame.Out(maths.V(-1.4, .6, -2)), site.frame.Origin
	up := site.frame.Orientation.Out(maths.YAxis)
	local := maths.NewSubspace(from, maths.LookAtUp(from, to, up), 1)
	return local.Within(site.motion.At(at)), nil
}

// tour approaches from above and flies past the city, keeping the site's local
// up and the ring's actual motion. Offset allows a later cut to resume it.
func (site orbitalSurfaceSite) tour(duration, offset time.Duration) func(time.Duration) (maths.Subspace, error) {
	return func(at time.Duration) (maths.Subspace, error) {
		t := math.Max(0, math.Min(1, float64(at+offset)/float64(duration)))
		if site.cityTour {
			// Follow the central park path at human scale. Positions are
			// expressed in the original factory city before its 250x orbital
			// transformation; the route stays within the open path corridor.
			park := manufacture.DefaultClusteredCity().Parks[0]
			walk := math.Min(1, 2*t)
			walk = 1 - (1-walk)*(1-walk)
			city := manufacture.CityParkLongPathAt(park, .5+.0003*walk)
			ground := city.Scale(orbitalCityScale).Add(orbitalCityOffset)
			ground.Y += 25
			target := site.landmark
			if target == (maths.Vec{}) {
				target = maths.V(0, 100000, 0)
			}
			if t > .5 && t <= .65 {
				// Rise vertically in the cleared courtyard before crossing
				// the buildings; ease both ends of the ascent.
				u := (t - .5) / .15
				ground.Y += 80000 * u * u * (3 - 2*u)
			} else if t > .65 {
				// Approach above the city, ending outside the landmark's
				// conservative footprint.
				ground.Y += 80000
				radius := math.Max(site.landmarkRadius, 100000)
				end := target.Add(maths.V(radius+50000, 60000, -100000))
				ground = orbitalBezier(ground,
					ground,
					target.Add(maths.V(radius+250000, 200000, -120000)), end,
					(t-.65)/.35)
			}
			from := site.ring.surfacePoint(site.centre, ground)
			to := site.ring.surfacePoint(site.centre, target)
			up := site.ring.surfaceFrame(from).Orientation.Out(maths.YAxis)
			camera := maths.NewSubspace(from, maths.LookAtUp(from, to, up), 1)
			return camera.Within(site.motion.At(at)), nil
		}
		// Fly along the side of the landmark rather than looking down on it.
		// Coordinates are in the selected structure's HEIGHT, including for a
		// bridge whose much wider span must not push the camera into space.
		scale := site.frame.Scale
		// Positions are in millions of metres and keep the actual city footprint
		// in view regardless of the chosen tower or bridge's dimensions.
		local := func(x, altitude, z float64) maths.Vec { return maths.V(x/scale, altitude/scale-.45, z/scale) }
		p := orbitalBezier(local(-.45, .075, -1.9), local(-.32, .065, -1.8), local(-.12, .05, -1.55), local(.20, .045, -1.2), t)
		look := local(.7*(1-t), .04, 0)
		from, to := site.frame.Out(p), site.frame.Out(look)
		up := site.frame.Orientation.Out(maths.YAxis)
		camera := maths.NewSubspace(from, maths.LookAtUp(from, to, up), 1)
		return camera.Within(site.motion.At(at)), nil
	}
}

// orbitalBezier is a continuous cubic path through the city approach.
func orbitalBezier(a, b, c, d maths.Vec, t float64) maths.Vec {
	u := 1 - t
	return a.Scale(u * u * u).Add(b.Scale(3 * u * u * t)).Add(c.Scale(3 * u * t * t)).Add(d.Scale(t * t * t))
}

// surfaceFrame uses east around the band, up inward and south across its width.
func (one hoop) surfaceFrame(at maths.Vec) maths.Subspace {
	angle := math.Atan2(at.X, at.Z)
	sine, cosine := math.Sin(angle), math.Cos(angle)
	at = maths.V(one.surfaceRadius(angle)*sine, at.Y, one.surfaceRadius(angle)*cosine)
	return maths.NewSubspace(at, maths.Orientation{X: maths.V(cosine, 0, -sine), Y: maths.V(-sine, 0, -cosine), Z: maths.YAxis}, 1)
}

// surfaceRadius is the actual polygonal floor, rather than its circumscribed circle.
func (one hoop) surfaceRadius(angle float64) float64 {
	step := 2 * math.Pi / bandAround
	middle := (math.Floor(angle/step) + .5) * step
	return one.radius * math.Cos(step/2) / math.Cos(angle-middle)
}

// surfacePoint wraps physical metres onto the retained floor's local geometry.
func (one hoop) surfacePoint(site, point maths.Vec) maths.Vec {
	angle := math.Atan2(site.X, site.Z) + point.X*1e-6/one.radius
	radius := one.surfaceRadius(angle) - point.Y*1e-6
	return maths.V(radius*math.Sin(angle), site.Y+point.Z*1e-6, radius*math.Cos(angle))
}

// surfaceFeatures consumes the existing geography; cities stay in its cities and
// relief in its highlands. Dimensions are authored Culture-scale scenery, not
// a physical engineering claim. The legacy zero-count floor is unchanged.
func (one hoop) surfaceFeatures(world *thing.World, geography *biome.World, floor *thing.Material,
	number int, day time.Duration, buildings, landforms, district int, relief, city, park, dense bool, style string, said io.Writer) (*orbitalSurfaceSite, error) {
	if buildings == 0 && landforms == 0 && district == 0 && !relief && !city {
		return nil, nil
	}
	features := geography.Features()
	seed := uint64(orbitalSeed + number)
	sort.Slice(features, func(a, b int) bool {
		return noise.Hash2(seed, int64(features[a].Tile.Column), int64(features[a].Tile.Row)) < noise.Hash2(seed, int64(features[b].Tile.Column), int64(features[b].Tile.Row))
	})
	built, raised := 0, 0
	var focus *orbitalSurfaceSite
	var builtSites []orbitalFootprint
	var candidates []orbitalFocusCandidate
	var focusAt maths.Vec
	var focusSeed uint64
	focusLight := -2.0
	for _, feature := range features {
		if math.Abs(feature.At.Y) > bandWidth/2-1 {
			continue
		}
		own := seed + uint64(feature.Tile.Column)*65537 + uint64(feature.Tile.Row)*257
		if feature.City && built < buildings {
			styles := []manufacture.MacrostructureStyle{manufacture.TerracedArcology, manufacture.TwistedSpires, manufacture.InhabitedBridge}
			family := styles[built%len(styles)]
			if style != "mixed" {
				family = manufacture.MacrostructureStyle(style)
			}
			spec := manufacture.DefaultMacrostructure(family)
			size := .65 + 1.1*noise.Hash(own, 1)
			// Regenerate dimensional architecture: 1,000 km bridge spans, 100-400 km towers.
			if spec.Style == manufacture.InhabitedBridge {
				spec.Width = measure.Metres(1000000 * size)
				spec.Depth = measure.Metres(180000 * size)
				spec.Height = measure.Metres(220000 * size)
			} else {
				spec.Width = measure.Metres(180000 * size)
				spec.Depth = measure.Metres(150000 * size)
				spec.Height = measure.Metres(300000 * size)
			}
			if (district > 0 || relief || city) && !one.surfaceFootprintLand(geography, feature.At, maths.Vec{}, .52*math.Max(spec.Width.InMetres(), spec.Depth.InMetres())) {
				continue
			}
			// When a local flythrough is requested, unsuitable candidates must
			// not consume the limited global structure count before a viable
			// camera focus and the future city have been found.
			if (district > 0 || relief || city) && math.Abs(feature.At.Y) >= bandWidth/2-6 {
				continue
			}
			if city && !one.surfaceFootprintLand(geography, feature.At, maths.V(0, 0, -1400000), 905100) {
				continue
			}
			spec.Wildness = .85
			spec.LivingCover = true
			spec.Twist = measure.Degrees(50 + 120*noise.Hash(own, 2))
			kind, err := manufacture.Macrostructure(spec)
			if err != nil {
				return nil, err
			}
			if err = one.surfaceSpecimen(world, kind, own, feature.At, day); err != nil {
				return nil, err
			}
			builtSites = append(builtSites, orbitalFootprint{offset: feature.At, radius: .52*math.Max(spec.Width.InMetres(), spec.Depth.InMetres()) + 50000})
			illumination := one.space().Orientation.Out(maths.V(-feature.At.X, 0, -feature.At.Z).Unit()).Dot(maths.V(nearSunX, nearSunY, nearSunZ).Unit())
			cityLand := !city || one.surfaceFootprintLand(geography, feature.At, maths.V(0, 0, -1400000), 905100)
			if illumination > focusLight && cityLand && (district == 0 && !relief && !city || math.Abs(feature.At.Y) < bandWidth/2-6) {
				focusLight = illumination
				frame := one.surfaceFrame(feature.At)
				frame.Origin = one.surfacePoint(feature.At, maths.V(0, spec.Height.InMetres()*.45, 0))
				frame.Scale = spec.Height.InMetres() * 1e-6
				focus = &orbitalSurfaceSite{frame: frame, motion: one.turning(day), ring: one, centre: feature.At}
				focusAt, focusSeed = feature.At, own
			}
			if city && cityLand {
				frame := one.surfaceFrame(feature.At)
				frame.Origin = one.surfacePoint(feature.At, maths.V(0, spec.Height.InMetres()*.45, 0))
				frame.Scale = spec.Height.InMetres() * 1e-6
				candidates = append(candidates, orbitalFocusCandidate{site: orbitalSurfaceSite{frame: frame, motion: one.turning(day), ring: one, centre: feature.At}, at: feature.At, seed: own, illumination: illumination})
			}
			built++
		} else if feature.Highland && !feature.City && feature.HeightAboveSea > 0 && raised < landforms {
			style := manufacture.MountainMassif
			if feature.Volcano {
				style = manufacture.CalderaVolcano
			}
			spec := manufacture.DefaultLandform(style)
			spec.Ruggedness = 1
			size := .6 + noise.Hash(own, 3)
			spec.Radius = measure.Metres(300000 * size)
			spec.Height = measure.Metres(24000 * size)
			if style == manufacture.CalderaVolcano {
				spec.CraterRadius = measure.Metres(45000 * size)
				spec.CraterDepth = measure.Metres(9000 * size)
			}
			kind, err := manufacture.Landform(spec)
			if err != nil {
				return nil, err
			}
			if (district > 0 || relief || city) && !one.surfaceFootprintLand(geography, feature.At, maths.Vec{}, spec.Radius.InMetres()) {
				continue
			}
			if err = one.surfaceSpecimen(world, kind, own, feature.At, day); err != nil {
				return nil, err
			}
			builtSites = append(builtSites, orbitalFootprint{offset: feature.At, radius: spec.Radius.InMetres() + 50000})
			if focus == nil && buildings == 0 {
				focus = &orbitalSurfaceSite{frame: one.surfaceFrame(feature.At), motion: one.turning(day)}
			}
			raised++
		}
		if built == buildings && raised == landforms {
			break
		}
	}
	if city {
		focus = nil
		focusLight = -2
		for i := range candidates {
			candidate := &candidates[i]
			if candidate.illumination <= focusLight || one.cityOverlapsSites(candidate.at, builtSites) {
				continue
			}
			focus, focusAt, focusSeed = &candidate.site, candidate.at, candidate.seed
			focusLight = candidate.illumination
		}
	}
	var districtSites []orbitalFootprint
	if district > 0 {
		if focus == nil || buildings == 0 {
			return nil, fmt.Errorf("orbital surface-district needs an inhabited surface site")
		}
		var existing []orbitalFootprint
		for _, other := range builtSites {
			a := math.Remainder(math.Atan2(other.offset.X, other.offset.Z)-math.Atan2(focusAt.X, focusAt.Z), 2*math.Pi)
			other.offset = maths.V(a*one.radius*1e6, 0, (other.offset.Y-focusAt.Y)*1e6)
			existing = append(existing, other)
		}
		var err error
		if park {
			districtSites, err = one.surfaceDistrictDirected(world, geography, focusAt, focusSeed, day, district, city, true, existing...)
		} else {
			districtSites, err = one.surfaceDistrict(world, geography, focusAt, focusSeed, day, district, city)
		}
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(said, "ring %d: %d additional distinct structures at the inhabited site\n", number, district)
		if park && city && len(districtSites) > 2 {
			focus.landmark = districtSites[2].offset.Add(maths.V(0, 100000, 0))
			focus.landmarkRadius = districtSites[2].radius
		}
	}
	if city {
		if focus == nil || buildings == 0 {
			return nil, fmt.Errorf("orbital surface-city needs a dry inhabited surface site")
		}
		advance, err := one.surfaceCity(world, focusAt, focusSeed, day, dense)
		if err != nil {
			return nil, err
		}
		focus.tick = advance
		focus.cityTour = park
		fmt.Fprintf(said, "ring %d: 10,000-building city with foliage and six moving lights\n", number)
	}
	if relief {
		if focus == nil || buildings == 0 {
			return nil, fmt.Errorf("orbital surface-relief needs an inhabited surface site")
		}
		floor.Texture = orbitalGroundPaint{base: floor.Texture, ring: one, site: focusAt, seed: focusSeed, geography: geography}
		if city {
			districtSites = append(districtSites, orbitalFootprint{offset: maths.V(0, 0, -1400000), radius: 905100})
		}
		for _, site := range builtSites {
			a := math.Remainder(math.Atan2(site.offset.X, site.offset.Z)-math.Atan2(focusAt.X, focusAt.Z), 2*math.Pi)
			site.offset = maths.V(a*one.radius*1e6, 0, (site.offset.Y-focusAt.Y)*1e6)
			districtSites = append(districtSites, site)
		}
		if err := one.surfaceRelief(world, geography, floor, focusAt, focusSeed, day, districtSites); err != nil {
			return nil, err
		}
		fmt.Fprintf(said, "ring %d: procedural coastal terrain around inhabited site\n", number)
	}
	fmt.Fprintf(said, "ring %d: %d inhabited macrostructures, %d landforms from existing geography\n", number, built, raised)
	return focus, nil
}

// cityOverlapsSites conservatively checks the proposed city's circumscribed
// circle against every other globally placed structure and landform.
func (one hoop) cityOverlapsSites(centre maths.Vec, sites []orbitalFootprint) bool {
	for _, site := range sites {
		if site.offset == centre {
			continue
		}
		a := math.Remainder(math.Atan2(site.offset.X, site.offset.Z)-math.Atan2(centre.X, centre.Z), 2*math.Pi)
		x := a * one.radius * 1e6
		z := (site.offset.Y-centre.Y)*1e6 + 1400000
		if math.Hypot(x, z) < site.radius+905100 {
			return true
		}
	}
	return false
}

// surfaceSpecimen keeps native material meshes and deforms their vertices onto
// the band's own floor. Every part rotates with that floor, including shadows.
func (one hoop) surfaceSpecimen(world *thing.World, kind manufacture.Kind,
	seed uint64, site maths.Vec, day time.Duration) error {
	return one.surfaceSpecimenTurned(world, kind, seed, site, day, maths.Orientation{})
}

// surfaceSpecimenTurned rotates a specimen and its local paint together before
// curving it onto the floor; independent bearings break repeated silhouettes.
func (one hoop) surfaceSpecimenTurned(world *thing.World, kind manufacture.Kind,
	seed uint64, site maths.Vec, day time.Duration, turn maths.Orientation) error {
	made, err := manufacture.Make(kind, manufacture.Sow(seed))
	if err != nil {
		return err
	}
	parts, err := made.Things(maths.Local())
	if err != nil {
		return err
	}
	for _, part := range parts {
		original, ok := part.Shape.(*mesh.Mesh)
		if !ok {
			return fmt.Errorf("orbital surface kind needs native mesh, got %T", part.Shape)
		}
		points := original.Points()
		for i, p := range points {
			points[i] = one.surfacePoint(site, turn.Out(p))
		}
		bent, err := mesh.New(points, original.Faces(), nil, false)
		if err != nil {
			return fmt.Errorf("orbital surface: %w", err)
		}
		part.Shape = bent
		part.Motion = one.turning(day)
		look := *part.Material
		if look.Texture != nil {
			look.Texture = orbitalSurfacePaint{look.Texture, one, site, turn}
		}
		if look.Shine != nil {
			look.Shine = orbitalSurfacePaint{look.Shine, one, site, turn}
		}
		if look.Name == "macro blue glazing" || look.Name == "ancient living macro blue glazing" {
			look.Shine = orbitalSurfacePaint{orbitalWindows{seed}, one, site, turn}
		}
		part.Material = &look
		world.Add(part)
	}
	return nil
}

// orbitalSurfacePaint retains physical metre coordinates after floor wrapping.
type orbitalSurfacePaint struct {
	paint texture.Texture
	ring  hoop
	site  maths.Vec
	turn  maths.Orientation
}

// At returns the paint at at. The paint's coordinates are in metres from the
// site: round the ring, out from the ring's surface, and up from the site's
// height.
func (p orbitalSurfacePaint) At(at maths.Vec) maths.Light {
	angle := math.Atan2(at.X, at.Z)
	delta := math.Remainder(angle-math.Atan2(p.site.X, p.site.Z), 2*math.Pi)
	local := maths.V(delta*p.ring.radius*1e6, (p.ring.surfaceRadius(angle)-math.Hypot(at.X, at.Z))*1e6, (at.Y-p.site.Y)*1e6)
	return p.paint.At(p.turn.In(local))
}
