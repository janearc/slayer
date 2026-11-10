package oceantex

import (
	"errors"
	"math"
	"runtime"
	"sync"
)

var (
	cascadeExtents = nums("view.cascade.extents")
	cascadeGrid    = int(num("view.cascade.grid"))
	filterKeep     = num("view.filter.keep")
	filterDrop     = num("view.filter.drop")
	slopeBound     = num("view.slope.bound")
	foldFloor      = num("view.fold.floor")
	chop           = num("choppy.scale")
	marchSteps     = int(num("view.march.steps"))
	marchNear      = num("view.march.near")
	invertRounds   = int(num("view.invert.rounds"))
)

// layer is one patch of waves of a view, covering one band of wavelengths.
type layer struct {
	field               *Field
	texel               float64 // metres between its samples
	gotAlong, gotAcross float64 // the slope variance its band carries
	reach               float64 // its highest crest or deepest trough
}

// surface is the sea of a view: patches of different sizes, each carrying
// the waves too short for the next larger one, tiled without end.
//
// GLUE: the tiling, and the sizes. The largest patch repeats every four
// kilometres, which shows in a view that takes in more than that.
type surface struct {
	layers []layer
	reach  float64
	// shore is the ground the sea meets, already told what it does to
	// this sea's waves, or nil over open water.
	shore *Shore
}

// newSurface builds the sea's patches at a moment, each band carrying the
// waves too short for the next larger one.
func newSurface(sea Sea, seconds float64, seed uint64) (*surface, error) {
	s := &surface{}
	shortest := 0.0
	for i, extent := range cascadeExtents {
		texel := extent / float64(cascadeGrid)
		longest := math.Pi / texel // the shortest wave the grid can hold
		patch, err := newBand(sea, cascadeGrid, extent, seed+uint64(i),
			shortest, longest)
		if err != nil {
			return nil, err
		}
		l := layer{field: patch.At(seconds), texel: texel}
		l.gotAlong, l.gotAcross = patch.Resolved()
		for _, h := range l.field.Height {
			l.reach = math.Max(l.reach, math.Abs(h))
		}
		s.layers = append(s.layers, l)
		s.reach += l.reach
		shortest = longest
	}
	return s, nil
}

// share is how much of a layer a pixel shows, given how much sea the
// pixel covers: all of it while the pixel is no bigger than a texel, none
// once it covers two, and a straight line between.
func (l *layer) share(footprint float64) float64 {
	texels := footprint / l.texel
	switch {
	case texels <= filterKeep:
		return 1
	case texels >= filterDrop:
		return 0
	}
	return (filterDrop - texels) / (filterDrop - filterKeep)
}

// flat is whether a pixel covering this much sea shows no waves at all.
func (s *surface) flat(footprint float64) bool {
	return s.layers[0].share(footprint) == 0
}

// source is the place on the undisturbed sea whose water the choppy
// waves have carried to (x, y): the fixed point of p = (x, y) - push(p).
func (s *surface) source(x, y, footprint float64, rounds int) (px, py float64) {
	px, py = x, y
	for range rounds {
		var pushX, pushY float64
		for i := range s.layers {
			l := &s.layers[i]
			if w := l.share(footprint); w > 0 {
				pushX += w * l.field.sample(l.field.PushX, px, py)
				pushY += w * l.field.sample(l.field.PushY, px, py)
			}
		}
		px, py = x-chop*pushX, y-chop*pushY
	}
	return px, py
}

// height is the sea's height at a place, as a pixel covering this much
// sea shows it: the open sea's, and over a shore, that with the waves a
// wall sends back added, all made as tall as the depth there makes them.
func (s *surface) height(x, y, footprint float64, rounds int) float64 {
	if s.shore == nil || !s.shore.Holds(x, y) {
		return s.open(x, y, footprint, rounds)
	}
	return s.arriving(x, y, footprint, rounds) *
		s.shore.read(s.shore.gain, x, y)
}

// arriving is the sea's height at a place over a shore before the depth
// has done anything to it: the open sea's, and the waves a wall sends
// back.
func (s *surface) arriving(x, y, footprint float64, rounds int) float64 {
	h := s.open(x, y, footprint, rounds)
	if mx, my, returns, faces := s.shore.mirror(x, y); faces {
		h += returns * s.open(mx, my, footprint, rounds)
	}
	return h
}

// open is the open sea's height at a place, with no shore.
func (s *surface) open(x, y, footprint float64, rounds int) float64 {
	px, py := s.source(x, y, footprint, rounds)
	var h float64
	for i := range s.layers {
		l := &s.layers[i]
		if w := l.share(footprint); w > 0 {
			h += w * l.field.sample(l.field.Height, px, py)
		}
	}
	return h
}

// slopes is the sea's slope at a place, and the slope variance the
// layers a pixel shows account for: the open sea's, and over a shore,
// that with a wall's mirrored waves added and the shore's gain applied.
//
// GLUE: the gain is taken as level across a pixel, so the slope it adds
// where the depth changes quickly is left out.
func (s *surface) slopes(
	x, y, footprint float64,
) (slopeX, slopeY, gotAlong, gotAcross float64) {
	slopeX, slopeY, gotAlong, gotAcross = s.openSlopes(x, y, footprint)
	if s.shore == nil || !s.shore.Holds(x, y) {
		return slopeX, slopeY, gotAlong, gotAcross
	}
	both := 1.0
	if mx, my, returns, faces := s.shore.mirror(x, y); faces {
		backX, backY, _, _ := s.openSlopes(mx, my, footprint)
		// a mirrored wave's slope is mirrored too
		n := s.shore.wall.seaward
		across := 2 * (backX*n.X + backY*n.Y)
		slopeX += returns * (backX - across*n.X)
		slopeY += returns * (backY - across*n.Y)
		both += returns * returns
	}
	gain := s.shore.read(s.shore.gain, x, y)
	return slopeX * gain, slopeY * gain, gotAlong * both * gain * gain,
		gotAcross * both * gain * gain
}

// openSlopes is the open sea's slope at a place, with no shore.
func (s *surface) openSlopes(
	x, y, footprint float64,
) (slopeX, slopeY, gotAlong, gotAcross float64) {
	px, py := s.source(x, y, footprint, invertRounds)
	var gx, gy, xx, xy, yy float64
	for i := range s.layers {
		l := &s.layers[i]
		w := l.share(footprint)
		if w == 0 {
			continue
		}
		f := l.field
		gx += w * f.sample(f.SlopeX, px, py)
		gy += w * f.sample(f.SlopeY, px, py)
		xx += w * f.sample(f.StretchXX, px, py)
		xy += w * f.sample(f.StretchXY, px, py)
		yy += w * f.sample(f.StretchYY, px, py)
		gotAlong += w * w * l.gotAlong
		gotAcross += w * w * l.gotAcross
	}
	// The slopes above are per metre of undisturbed sea. The choppy push
	// stretches the sea by J = 1 + its gradient, so per metre of the sea
	// as it lies they are J's inverse times those. GLUE: where the push
	// would fold the surface through itself, J is held just short of it.
	a, b, d := 1+chop*xx, chop*xy, 1+chop*yy
	det := math.Max(a*d-b*b, foldFloor)
	return (d*gx - b*gy) / det, (a*gy - b*gx) / det, gotAlong, gotAcross
}

// View is a picture of a scene from a camera: any height, any direction,
// any lens, the horizon included. Pixels that miss the sea show the scene's Sky, or
// black without one.
//
// Each pixel's ray is marched to the first wave it meets, so near waves
// hide far ones. A pixel shows the waves at least as long as the sea it
// covers; shorter ones become roughness, widening the sun's glint, so the
// sea toward the horizon and from high up smooths into the Cox-Munk
// glitter pattern with no seam.
//
// The sea is flat in the large: there is no curve of the Earth, so the
// horizon is at eye level and a view from orbit has no limb.
func View(scene Scene, camera Camera) (*Texture, error) {
	switch {
	case camera.Width < 1 || camera.Height < 1:
		return nil, errors.New("oceantex: a picture needs a size")
	case !(camera.Eye.Origin.Z > 0):
		return nil, errors.New("oceantex: the camera must be above the sea")
	}
	limits, err := readLimits()
	if err != nil {
		return nil, err
	}
	if camera.Width > limits.ImagePixels/camera.Height || !finite(camera.Seconds) {
		return nil, errors.New("oceantex: view exceeds pixel budget or has invalid time")
	}
	l, err := scene.lighting()

	if err != nil {
		return nil, err
	}
	sea, err := newSurface(scene.Sea, camera.Seconds, camera.Seed)
	if err != nil {
		return nil, err
	}
	if scene.Shore != nil {
		// the shore is told about this sea on a copy, so one shore can
		// stand in many scenes
		shore := *scene.Shore
		shore.meet(scene.Sea)
		sea.shore = &shore
		tallest := 0.0
		for _, gain := range shore.gain {
			tallest = math.Max(tallest, gain)
		}
		if shore.wall != nil {
			tallest *= 1 + shore.wall.returns
		}
		sea.reach = math.Max(sea.reach*math.Max(tallest, 1), shore.top)
	}

	out := &Texture{Width: camera.Width, Height: camera.Height,
		Light: make([][3]float64, camera.Width*camera.Height), Exposure: 1}
	rows := make(chan int)
	var workers sync.WaitGroup
	for range min(runtime.GOMAXPROCS(0), camera.Height) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for row := range rows {
				for col := 0; col < camera.Width; col++ {
					ray := Look(camera.Eye, camera.Lens, col, row,
						camera.Width, camera.Height)
					// a pixel is as wide as the angle to the one beside
					// it, which is how slayer's tracer measures one too
					beside := col + 1
					if beside == camera.Width && camera.Width > 1 {
						beside = col - 1
					}
					next := Look(camera.Eye, camera.Lens, beside, row,
						camera.Width, camera.Height)
					pixel := math.Acos(math.Min(1, ray.Dir.dot(next.Dir)))
					if camera.Width == 1 {
						pixel = math.Pi / 3
					}
					// row zero is the top of the picture; a Texture
					// keeps its rows from the bottom up
					at := (camera.Height-1-row)*camera.Width + col
					out.Light[at] = l.along1(sea, ray.At, ray.Dir, pixel)
				}
			}
		}()
	}
	for row := 0; row < camera.Height; row++ {
		rows <- row
	}
	close(rows)
	workers.Wait()
	return out, nil
}

// along1 is the light arriving along one ray.
func (l *lighting) along1(sea *surface, from, ray Vec, pixel float64) light {
	sky := func() light {
		if l.sky == nil {
			return light{}
		}
		return lightOf(l.sky(ray))
	}
	// GLUE: a pixel covers a patch of sea as wide as its angle times the
	// distance, and longer than that along the line of sight by one over
	// the sine of the grazing angle. The size used is between the two.
	graze := math.Sqrt(math.Max(math.Abs(ray.Z), viewFloor))
	level := math.Hypot(ray.X, ray.Y)

	t := 0.0
	if from.Z > sea.reach {
		if ray.Z >= 0 {
			return sky()
		}
		t = (from.Z - sea.reach) / -ray.Z
	}
	eye := ray.times(-1)
	shore := sea.shore
	steps := marchSteps
	if shore != nil {
		steps = shoreSteps
	}
	for range steps {
		footprint := t * pixel / graze
		at := from.plus(ray.times(t))
		over := shore != nil && shore.Holds(at.X, at.Y)
		if sea.flat(footprint) && !over {
			// too far for any wave to show: the mean sea, all roughness
			if ray.Z >= 0 {
				return sky()
			}
			return l.seen(eye, 0, 0, 0, 0)
		}
		top := sea.height(at.X, at.Y, footprint, 1)
		if over {
			top = math.Max(top, shore.Height(at.X, at.Y))
		}
		gap := at.Z - top
		if gap <= marchNear*math.Max(t, 1) {
			break
		}
		if ray.Z > 0 && at.Z > sea.reach {
			return sky()
		}
		// no wave is steeper than the bound, so the ray cannot meet one
		// sooner than this
		closing := slopeBound*level - ray.Z
		if closing <= 0 {
			return sky()
		}
		step := gap / closing
		if over {
			// a cliff is steeper than any wave. over a shore a ray goes
			// no further at a time than the ground's samples are apart,
			// or a couple of its own pixels where those are bigger
			step = math.Min(step,
				math.Max(shore.step, shoreWiden*t*pixel))
		}
		t += step
	}
	at := from.plus(ray.times(t))
	footprint := t * pixel / graze
	if shore != nil && shore.Holds(at.X, at.Y) {
		water := sea.height(at.X, at.Y, footprint, invertRounds)
		if shore.Height(at.X, at.Y) >= water {
			return l.rock(shore.slope(at.X, at.Y))
		}
	}
	slopeX, slopeY, gotAlong, gotAcross := sea.slopes(at.X, at.Y, footprint)
	seen := l.seen(eye, slopeX, slopeY, gotAlong, gotAcross)
	if shore != nil && shore.Holds(at.X, at.Y) {
		// where a wave is breaking its crest is foam
		surf := shore.foam(at.X, at.Y,
			sea.arriving(at.X, at.Y, footprint, invertRounds))
		seen = seen.times(1 - surf).plus(l.whitewater().times(surf))
	}
	return seen
}
