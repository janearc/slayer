package city

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// Builder adds the deterministic Griffith ridge and basin meshes to a world.
// Ground may supply shot-specific elevations; nil uses the Griffith formula.
type Builder struct {
	world          *thing.World
	ground         func(float64, float64) float64
	cityGround     func(float64, float64) float64
	haze           bool
	cityForward    float64
	cityDetail     float64
	terrainSamples int
	lightScale     float64
	lightFootprint float64
}

// New creates a builder in metres. haze opts into the legacy Griffith basin
// scattering volume; clear-boundary underwater sets should pass false.
func New(world *thing.World, ground func(float64, float64) float64, haze bool) *Builder {
	if ground == nil {
		ground = GriffithGround
	}
	return &Builder{world: world, ground: ground, haze: haze, cityDetail: 1, terrainSamples: 513, lightScale: 1, lightFootprint: 1}
}

// valid reports whether the builder has a world to build into.
func (s *Builder) valid() error {
	if s == nil || s.world == nil {
		return fmt.Errorf("the Griffith builder needs a world")
	}
	return nil
}

// urbanGround returns the city ground height, or the site ground when unset.
func (s *Builder) urbanGround(x, z float64) float64 {
	if s.cityGround != nil {
		return s.cityGround(x, z)
	}
	return s.ground(x, z)
}

// ShotOptions controls the offshore view without changing the historic
// Griffith stage defaults. CityForward moves the local city toward the sea;
// CityDetail deterministically thins repeat lots; TerrainSamples lowers the
// distant terrain mesh cost; LightScale adjusts emissive window/lamp cues.
// Its small positive range supports night scenes with high global exposure.
type ShotOptions struct {
	CityForward    float64
	RidgeForward   float64
	CityDetail     float64
	TerrainSamples int
	LightScale     float64
	LightFootprint float64
}

// AddGriffithBasinWithOptions builds the shared landscape with independent
// far-view controls. A zero-valued option uses the previous full-detail
// defaults except CityForward, which remains zero by design. LightScale may
// be very small for night scenes where global print exposure is high.
func AddGriffithBasinWithOptions(world *thing.World, at maths.Vec, heading, scale, reliefScale float64, options ShotOptions) error {
	if world == nil || !finite(at.X) || !finite(at.Y) || !finite(at.Z) ||
		!finite(heading) || heading < 0 || heading > 360 || !finite(scale) || scale < .01 || scale > 10 ||
		!finite(reliefScale) || reliefScale < .01 || reliefScale > 2 ||
		!finite(options.CityForward) || math.Abs(options.CityForward) > 2000 ||
		!finite(options.RidgeForward) || options.RidgeForward < 0 || options.RidgeForward > 10000 ||
		(options.CityDetail != 0 && (options.CityDetail < .1 || options.CityDetail > 1)) ||
		(options.TerrainSamples != 0 && (options.TerrainSamples < 129 || options.TerrainSamples > 513 || (options.TerrainSamples-1)&(options.TerrainSamples-2) != 0)) ||
		(options.LightScale != 0 && (options.LightScale < 1e-9 || options.LightScale > 20)) ||
		(options.LightFootprint != 0 && (options.LightFootprint < .5 || options.LightFootprint > 6)) {
		return fmt.Errorf("the Griffith basin needs finite metre coordinates, heading 0–360 degrees, scale 0.01–10, relief-scale 0.01–2, city-forward ±2000m, ridge-forward 0–10000m, city-detail 0.1–1, terrain-samples 129/257/513, light-scale 1e-9–20, and light-footprint 0.5–6")
	}
	if options.CityDetail == 0 {
		options.CityDetail = 1
	}
	if options.TerrainSamples == 0 {
		options.TerrainSamples = 513
	}
	if options.LightScale == 0 {
		options.LightScale = 1
	}
	if options.LightFootprint == 0 {
		options.LightFootprint = 1
	}
	local := thing.NewWorld()
	cityGround := func(_, _ float64) float64 { return -285 * reliefScale }
	ground := func(x, z float64) float64 {
		y := GriffithGround(x, z-options.RidgeForward) * reliefScale
		// The shot-only shelf descends below the mean sea surface across the
		// seaward overlap. The ocean remains continuous over it; its edge is
		// not a vertical floating terrain boundary.
		if z < -600 {
			y -= 120 * maths.Smooth(0, 600, -z-600)
		}
		return y
	}
	stage := New(local, ground, false)
	stage.cityGround = cityGround
	stage.cityForward, stage.cityDetail = options.CityForward, options.CityDetail
	stage.terrainSamples, stage.lightScale, stage.lightFootprint = options.TerrainSamples, options.LightScale, options.LightFootprint
	if err := stage.offshoreLand(); err != nil {
		return err
	}
	if err := stage.LosAngelesBasin(); err != nil {
		return err
	}
	r := heading * math.Pi / 180
	turn := maths.Orientation{
		X: maths.V(-math.Cos(r), 0, -math.Sin(r)),
		Y: maths.V(0, 1, 0),
		Z: maths.V(math.Sin(r), 0, -math.Cos(r)),
	}
	placement := maths.NewSubspace(at, turn, scale)
	for _, each := range local.Things {
		each.Motion = thing.Still(each.At(0).Within(placement))
		world.Add(*each)
	}
	for _, each := range local.Volumes {
		each.Motion = thing.Still(each.At(0).Within(placement))
		world.AddVolume(*each)
	}
	return nil
}

// finite reports whether value is a real number, not NaN or infinite.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// GriffithGround is the authored non-survey relief function used by both
// the observatory scene and the offshore basin composition.
func GriffithGround(x, z float64) float64 {
	knoll := 285 * math.Exp(-math.Pow(x/730, 2)-math.Pow(z/520, 2))
	ridge := 90 * math.Exp(-math.Pow(x/1200, 2)) * maths.Smooth(0, 600, -z)
	folds := 25 * noise.Fractal(531, x/380, 0, z/380, 3) * (1 - math.Exp(-(x*x+z*z)/40000))
	raw := -285 + knoll + ridge + folds*math.Exp(-math.Pow(z/1700, 2))
	grade := 1 - maths.Smooth(1, 1.4, math.Max(math.Abs(x)/65, math.Abs(z+30)/80))
	return raw*(1-grade) + .2*grade
}

// griffithHill is an authored ridge descending into a broad urban basin.
// Metres are local to the observatory terrace; it does not claim DEM data.
func (s *Builder) GriffithHill() error {
	if err := s.valid(); err != nil {
		return err
	}
	const side = 513
	const reach = 9600.0
	coordinate := func(i int) float64 { return reach * math.Sinh(float64(i-256)/50) / math.Sinh(256.0/50) }
	points := make([]maths.Vec, side*side)
	faces := make([][3]int, 0, 2*(side-1)*(side-1))
	for z := range side {
		for x := range side {
			a, b := coordinate(x), coordinate(z)
			points[z*side+x] = maths.V(a, s.ground(a, b), b)
			if x < side-1 && z < side-1 {
				i := z*side + x
				j := i + side
				faces = append(faces, [3]int{i, j + 1, i + 1}, [3]int{i, j, j + 1})
			}
		}
	}
	mesh, err := shape.NewMesh(points, faces, nil, false)
	if err != nil {
		return err
	}
	s.world.Add(thing.Thing{Shape: mesh, Material: &thing.Material{Name: "authored Griffith park hillside and LA basin", Colour: maths.RGB(.07, .085, .055)}})
	s.world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Name: "distant LA basin floor", Colour: maths.RGB(.025, .032, .04)}, Motion: thing.Still(maths.NewSubspace(maths.V(0, -500, 0), maths.Identity, 1))})
	return nil
}

// offshoreLand builds a finite coastal patch for a set over the ocean. The
// x/z bounds intentionally leave open water on the seaward side; there is no
// infinite basin floor and no mesh under the submarine.
func (s *Builder) offshoreLand() error {
	if err := s.valid(); err != nil {
		return err
	}
	side := s.terrainSamples
	if side == 0 {
		side = 513
	}
	const west, east = -4500.0, 4500.0
	const seaward, inland = -1200.0, 7200.0
	points := make([]maths.Vec, side*side)
	faces := make([][3]int, 0, 2*(side-1)*(side-1))
	for z := range side {
		zAt := seaward + (inland-seaward)*float64(z)/float64(side-1)
		for x := range side {
			xAt := west + (east-west)*float64(x)/float64(side-1)
			points[z*side+x] = maths.V(xAt, s.ground(xAt, zAt), zAt)
			if x < side-1 && z < side-1 {
				i, j := z*side+x, (z+1)*side+x
				faces = append(faces, [3]int{i, j + 1, i + 1}, [3]int{i, j, j + 1})
			}
		}
	}
	mesh, err := shape.NewMesh(points, faces, nil, false)
	if err != nil {
		return err
	}
	s.world.Add(thing.Thing{Shape: mesh, Material: &thing.Material{Name: "authored finite Griffith coastal ridge", Colour: maths.RGB(.07, .085, .055)}})
	return nil
}

// siteBlocks batches urban surfaces by material without thousands of
// acceleration structures. Each block has its own dimensions and placement.
type siteBlocks struct {
	vertices []maths.Vec
	faces    [][3]int
}

// box adds the twelve triangles of an axis-aligned box at a placement.
func (b *siteBlocks) box(at, size maths.Vec) {
	first := len(b.vertices)
	for _, v := range []maths.Vec{
		maths.V(0, 0, 0), maths.V(size.X, 0, 0),
		maths.V(size.X, size.Y, 0), maths.V(0, size.Y, 0),
		maths.V(0, 0, size.Z), maths.V(size.X, 0, size.Z),
		maths.V(size.X, size.Y, size.Z), maths.V(0, size.Y, size.Z),
	} {
		b.vertices = append(b.vertices, v.Add(at))
	}
	for _, f := range [][3]int{{0, 2, 1}, {0, 3, 2}, {4, 5, 6}, {4, 6, 7}, {0, 1, 5}, {0, 5, 4}, {3, 7, 6}, {3, 6, 2}, {0, 4, 7}, {0, 7, 3}, {1, 2, 6}, {1, 6, 5}} {
		b.faces = append(b.faces, [3]int{first + f[0], first + f[1], first + f[2]})
	}
}

// place adds the collected blocks to the world as one mesh with a material.
func (b *siteBlocks) place(s *Builder, material *thing.Material) error {
	if len(b.faces) == 0 {
		return nil
	}
	mesh, err := shape.NewMesh(b.vertices, b.faces, nil, false)
	if err != nil {
		return err
	}
	s.world.Add(thing.Thing{Shape: mesh, Material: material})
	return nil
}

// losAngelesBasin makes the view down from the hill intelligible: a dense
// street fabric, varied blocks and a compact downtown skyline to the south
// east. Street/building positions and night lighting are authored scenery.
// LosAngelesBasin adds grouped urban blocks, window lights and stepped
// downtown towers. Include haze only in routes that support its volume medium.
func (s *Builder) LosAngelesBasin() error {
	if err := s.valid(); err != nil {
		return err
	}
	var buildings [9]siteBlocks
	var warm, cool, lamps, traffic siteBlocks
	// Disjoint city lots and window faces share one mesh per material. They
	// need no per-piece acceleration structure; touching architectural parts
	// still use placeSeparate in the foreground building.
	// A city has a common street fabric but unequal lots, building massing,
	// occupancy and neighbourhood density. Lamps remain individual points.
	for row := range 50 {
		for col := range 72 {
			block := int64(row*72 + col)
			bx := float64(col-36) * 118
			bz := 600 + float64(row)*122 + s.cityForward
			if bx*bx+bz*bz < 700*700 {
				continue
			}
			for lot := range 7 {
				id := block*11 + int64(lot)
				if noise.Hash(2740, id) > s.cityDetail {
					continue
				}
				x := bx + 8 + float64(lot%3)*31 + 15*noise.Hash(2711, id)
				z := bz + 7 + float64(lot/3)*34 + 13*noise.Hash(2712, id)
				v := noise.Hash(2713, id)
				h := 3 + 11*v*v
				if col%11 == 0 || row%13 == 0 {
					h += 8 + 28*noise.Hash(2714, id)
				}
				w, d := 12+14*noise.Hash(2715, id), 13+15*noise.Hash(2716, id)
				y := s.urbanGround(x, z)
				buildings[int(noise.Hash(2717, id)*9)%9].box(maths.V(x, y, z), maths.V(w, h, d))
				// Rooftop plant, terraces and sheds give every lot a separate profile.
				if v > .42 {
					buildings[(lot+col)%9].box(maths.V(x+2, y+h, z+3), maths.V(w*.5, .7+2*v, d*.45))
				}
				for floor := 0; floor < int(h/3.3); floor++ {
					pitch := 3 * s.lightFootprint
					for bay := 0; bay < int(w/pitch); bay++ {
						if noise.Hash3(2718, id, int64(floor), int64(bay)) > .52 {
							continue
						}
						target := &warm
						if (lot+bay+floor)%5 == 0 {
							target = &cool
						}
						target.box(maths.V(x+1+float64(bay)*pitch, y+1.1+float64(floor)*3.3, z-.08), maths.V(1.1*s.lightFootprint, .85*math.Sqrt(s.lightFootprint), .1))
					}
				}
			}
			for _, offset := range []float64{0, 58} {
				x, z := bx+offset, bz-5
				y := s.urbanGround(x, z)
				if noise.Hash(2719, block) > .18 {
					lamps.box(maths.V(x, y+.8, z), maths.V(.6, .25, .6))
				}
			}
			if noise.Hash(2720, block) > .6 {
				x, z := bx+103, bz+45*noise.Hash(2721, block)
				y := s.urbanGround(x, z)
				traffic.box(maths.V(x, y+.35, z), maths.V(.7, .25, 1.4))
			}
		}
	}
	// A concentrated, irregular skyline south of the park: towers have
	// different setbacks and tops instead of seven copies of one block.
	for i := range 67 {
		id := int64(i)
		x := -850 + 1450*noise.Hash(2730, id)
		z := 3000 + 1100*noise.Hash(2731, id) + s.cityForward
		y := s.urbanGround(x, z)
		w := 30 + 38*noise.Hash(2732, id)
		d := 30 + 40*noise.Hash(2733, id)
		h := 45 + 210*math.Pow(noise.Hash(2734, id), 2)
		buildings[i%9].box(maths.V(x, y, z), maths.V(w, h, d))
		for step := range 3 {
			inset := float64(step+1) * w * .09
			buildings[i%9].box(maths.V(x+inset, y+h+float64(step)*4, z+inset), maths.V(w-2*inset, 4, d*(1-float64(step+1)*.13)))
		}
		for floor := 0; floor < int(h/3.4); floor++ {
			pitch := 4 * s.lightFootprint
			for bay := 0; bay < int(w/pitch); bay++ {
				if noise.Hash3(2735, id, int64(floor), int64(bay)) < .42 {
					cool.box(maths.V(x+2+float64(bay)*pitch, y+2+float64(floor)*3.4, z-.1), maths.V(1.8*s.lightFootprint, 1.1*math.Sqrt(s.lightFootprint), .12))
				}
			}
		}
	}
	for i := range buildings {
		shade := .035 + .012*float64(i)
		if err := buildings[i].place(s, &thing.Material{Name: "authored varied Los Angeles lots and skyline", Colour: maths.RGB(shade, shade*1.04, shade*1.12)}); err != nil {
			return err
		}
	}
	for _, row := range []struct {
		mesh *siteBlocks
		mat  *thing.Material
	}{
		{&warm, &thing.Material{Name: "LA warm individual occupied windows", Glow: maths.RGB(.35*s.lightScale, .24*s.lightScale, .12*s.lightScale)}},
		{&cool, &thing.Material{Name: "LA cool individual occupied windows", Glow: maths.RGB(.15*s.lightScale, .24*s.lightScale, .32*s.lightScale)}},
		{&lamps, &thing.Material{Name: "LA discontinuous street lamps", Glow: maths.RGB(1.2*s.lightScale, .63*s.lightScale, .22*s.lightScale)}},
		{&traffic, &thing.Material{Name: "LA scattered traffic lights", Glow: maths.RGB(.6*s.lightScale, .19*s.lightScale, .07*s.lightScale)}},
	} {
		if err := row.mesh.place(s, row.mat); err != nil {
			return err
		}
	}
	if s.haze {
		s.world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 6500, Thick: .000006}, Material: &thing.Material{Name: "thin cool Los Angeles basin air", Colour: maths.RGB(.24, .32, .45)}, Motion: thing.Still(maths.NewSubspace(maths.V(0, -300, 3000), maths.Identity, 1))})
	}
	return nil
}
