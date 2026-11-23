package searching

import (
	"context"
	"fmt"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/footprint"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/scene/island"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/terrain"
	"math"
)

const keckLatitude = 19.82611111111111
const keckLongitude = -155.47444444444446
const keckEarthRadius = 6371000.0

// summitHeights retains NOAA MSL as the existing study's explicitly authored
// radial offsets, without claiming a geoid conversion. Only the observatory
// platform is graded; the surrounding summit and distant flanks remain DEM.
type summitHeights struct {
	source   island.Heights
	platform float64
	pads     []summitPad
}

type summitPad struct{ x, z, radius, height float64 }

// Height returns the height at a longitude and latitude, with the summit pads
// applied.
func (h summitHeights) Height(longitude, latitude float64) (float64, bool, error) {
	height, has, err := h.source.Height(longitude, latitude)
	if err != nil || !has {
		return height, has, err
	}
	x := (longitude - keckLongitude) * maths.Degree * keckEarthRadius * math.Cos(keckLatitude*maths.Degree)
	z := (keckLatitude - latitude) * maths.Degree * keckEarthRadius
	a, b := maths.V(-28.669, 0, 18.299), maths.V(22.005, 0, -49.972)
	point := maths.V(x, 0, z)
	edge := b.Sub(a)
	u := math.Max(0, math.Min(1, point.Sub(a).Dot(edge)/edge.Dot(edge)))
	distance := point.Sub(a.Add(edge.Scale(u))).Len()
	grade := 1 - maths.Smooth(30, 48, distance)
	height = height*(1-grade) + h.platform*grade
	for _, pad := range h.pads {
		blend := 1 - maths.Smooth(pad.radius, pad.radius+12, math.Hypot(x-pad.x, z-pad.z))
		height = height*(1-blend) + pad.height*blend
	}
	return height, true, nil
}

// HeightRange preserves the indexed DEM bounds for view selection. Losing
// them forces every patch to use the whole ocean-to-summit height envelope.
func (h summitHeights) HeightRange(west, south, east, north float64) (float64, float64, bool, error) {
	ranged, ok := h.source.(interface {
		HeightRange(float64, float64, float64, float64) (float64, float64, bool, error)
	})
	if !ok {
		return 0, 0, false, nil
	}
	low, high, has, err := ranged.HeightRange(west, south, east, north)
	if err != nil || !has {
		return low, high, has, err
	}
	if east >= keckLongitude-.001 && west <= keckLongitude+.001 && north >= keckLatitude-.001 && south <= keckLatitude+.001 {
		low = math.Min(low, h.platform)
		high = math.Max(high, h.platform)
	}
	for _, pad := range h.pads {
		lon := keckLongitude + pad.x/(maths.Degree*keckEarthRadius*math.Cos(keckLatitude*maths.Degree))
		lat := keckLatitude - pad.z/(maths.Degree*keckEarthRadius)
		extent := (pad.radius + 15) / (maths.Degree * keckEarthRadius)
		if east >= lon-extent*1.1 && west <= lon+extent*1.1 && north >= lat-extent && south <= lat+extent {
			low = math.Min(low, pad.height)
			high = math.Max(high, pad.height)
		}
	}
	return low, high, true, nil
}

// maunaKeaGround uses Slayer's spherical, view-selected terrain builder.
// It replaces the coarse local grid and infinite flat horizon. Site anchor,
// CRS, MSL approximation and DEM identity are recorded in the source study.
func (s *heardStage) maunaKeaGround(p *stagePlan, raster *terrain.Raster) error {
	s.summit = true
	zero, has, err := raster.GeographicElevation(keckLongitude, keckLatitude)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("the Keck anchor is outside the DEM")
	}
	sphere, err := frame.NewSphere(measure.Metres(keckEarthRadius), frame.Location{Latitude: measure.Degrees(keckLatitude), Longitude: measure.Degrees(keckLongitude), Height: measure.Metres(zero)})
	if err != nil {
		return err
	}
	source, err := island.NewRasterHeights(raster, 4326, "NOAA CRM2023 MSL retained as authored radial offsets", measure.Metres(0))
	if err != nil {
		return err
	}
	source, err = source.Indexed(context.Background(), 4, 67108864)
	if err != nil {
		return err
	}
	heights := summitHeights{source: source, platform: zero + 2.5}
	for _, site := range maunaKeaNeighbours {
		lon := keckLongitude + site.x/(maths.Degree*keckEarthRadius*math.Cos(keckLatitude*maths.Degree))
		lat := keckLatitude - site.z/(maths.Degree*keckEarthRadius)
		height, has, err := source.Height(lon, lat)
		if err != nil {
			return err
		}
		if !has {
			return fmt.Errorf("summit neighbour %s lacks terrain", site.kind)
		}
		heights.pads = append(heights.pads, summitPad{site.x, site.z, site.radius, height + .5})
	}
	// Cache local floor queries before the caller closes the source raster.
	const side = 257
	const spacing = 5.0
	floor := make([]float64, side*side)
	for row := range side {
		for col := range side {
			x, z := float64(col-128)*spacing, float64(row-128)*spacing
			location, err := sphere.Location(maths.V(x, 0, z))
			if err != nil {
				return err
			}
			height, has, err := heights.Height(location.Longitude.InRadians()/maths.Degree, location.Latitude.InRadians()/maths.Degree)
			if err != nil {
				return err
			}
			if !has {
				return fmt.Errorf("the Keck platform lacks source coverage")
			}
			location.Height = measure.Metres(height)
			point, err := sphere.Position(location)
			if err != nil {
				return err
			}
			floor[row*side+col] = point.Y
		}
	}
	s.ground = func(x, z float64) float64 {
		a, b := math.Max(0, math.Min(255.999999, x/spacing+128)), math.Max(0, math.Min(255.999999, z/spacing+128))
		col, row := int(a), int(b)
		a -= float64(col)
		b -= float64(row)
		near := floor[row*side+col]*(1-a) + floor[row*side+col+1]*a
		far := floor[(row+1)*side+col]*(1-a) + floor[(row+1)*side+col+1]*a
		return near*(1-b) + far*b
	}
	coverage, ok := raster.Coverage()
	if !ok {
		return fmt.Errorf("the Keck DEM has no declared coverage")
	}
	region := island.Region{West: math.Max(coverage.West+.001, keckLongitude-.5), East: math.Min(coverage.East-.001, keckLongitude+.5), South: math.Max(coverage.South+.001, keckLatitude-.45), North: math.Min(coverage.North-.001, keckLatitude+.45)}
	ground, err := island.New(sphere, heights, region, island.Detail{Patches: 4096, Evaluations: 65536, Depth: 14, Side: 17, Points: 2097152, CellPixels: .75, MinimumHeight: -6000, MaximumHeight: 4500, NormalLongitude: 1.0 / 3600, NormalLatitude: 1.0 / 3600})
	if err != nil {
		return err
	}
	eye, err := maths.View(vector(p.Eye), vector(p.Look), maths.YAxis)
	if err != nil {
		return err
	}
	view, err := footprint.New(*eye, measure.Degrees(p.Wide), 2048, 1152, 1, measure.Metres(.1))
	if err != nil {
		return err
	}
	origin, err := frame.NewRelative(maths.Vec{})
	if err != nil {
		return err
	}
	prepared, err := ground.Prepare(context.Background(), view, origin)
	if err != nil {
		return err
	}
	s.summitGround = newSummitSurface(prepared.Surfaces)
	sourceFloor := s.ground
	s.ground = func(x, z float64) float64 {
		if y, has := s.summitGround.height(x, z); has {
			return y
		}
		return sourceFloor(x, z)
	}
	for _, surface := range prepared.Surfaces {
		s.world.Add(thing.Thing{Shape: surface, Material: &thing.Material{Name: "NOAA Mauna Kea spherical terrain with authored summit grading", Texture: maunaKeaAlbedo{anchor: zero}}})
	}
	ocean, err := globe.New(keckEarthRadius)
	if err != nil {
		return err
	}
	s.world.Add(thing.Thing{Shape: ocean, Material: &thing.Material{Name: "distant spherical Pacific reference surface", Colour: maths.RGB(.015, .035, .065)}, Motion: thing.Still(maths.NewSubspace(sphere.Centre(), maths.Identity, 1))})
	reference, err := thing.NewRayOriginReference(measure.Metres(1))
	if err != nil {
		return err
	}
	s.world.RayOriginReference = reference
	return s.fitSummitRoads()
}

// maunaKeaAlbedo is authored volcanic cover over the sourced relief. Its
// altitude zoning keeps vegetation on lower flanks; weathered pale patches
// and patchy summit snow are a reference-led weather choice, not a survey.
type maunaKeaAlbedo struct{ anchor float64 }

// At returns the surface colour at p, graded by altitude.
func (a maunaKeaAlbedo) At(p maths.Vec) maths.Light {
	height := a.anchor + p.Y + (p.X*p.X+p.Z*p.Z)/(2*keckEarthRadius)
	strata := .5 + .5*noise.Fractal(1181, p.X/90, p.Y/35, p.Z/90, 4)
	grain := .78 + .22*noise.Fractal(1182, p.X*1.8, p.Y*1.8, p.Z*1.8, 3)
	basalt := maths.RGB(.075, .078, .09)
	rust := maths.RGB(.32, .17, .12)
	rock := basalt.Scale(1 - strata).Add(rust.Scale(strata))
	ash := maths.Smooth(.52, .7, noise.Fractal(1183, p.X/32, p.Y/23, p.Z/32, 3)) * .5
	rock = rock.Scale(1 - ash).Add(maths.RGB(.49, .43, .39).Scale(ash))
	snow := maths.Smooth(4100, 4170, height) * maths.Smooth(.18, .5, noise.Fractal(1184, p.X/65, p.Y/25, p.Z/65, 4))
	rock = rock.Scale(1 - snow).Add(maths.RGB(.8, .86, .92).Scale(snow))
	vegetation := 1 - maths.Smooth(2650, 3150, height)
	return rock.Scale(1 - vegetation).Add(maths.RGB(.09, .13, .065).Scale(vegetation)).Scale(grain)
}
