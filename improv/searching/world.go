package searching

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// set is one retained scene: geometry is made once, actor poses and light
// are changed for an exposure. It owns no encoder or global render state.
type set struct {
	World           *thing.World
	Actors          []actor
	Mist            *medium.Layer
	Lasers          []*emit.Beam
	Sun, Moon       film.Sun
	Space           bool
	Site            string
	Disc, Jet, Lobe *thing.Material
	JetField        *medium.Jet
	prototypes      map[string][]thing.Thing
}

// actor is a manufactured assembly and its fixed position on the terrain.
// This proof slews in azimuth; elevation articulation remains future work.
type actor struct {
	things []*thing.Thing
	at     maths.Vec
	scale  float64
}

// newSet builds one named site without taking any GPU resources.
func newSet(plan Plan, site string) (*set, error) {
	set := &set{World: thing.NewWorld(), Site: site, prototypes: map[string][]thing.Thing{}}
	if site == "blazar" {
		return set, set.blazar()
	}
	if err := set.land(site); err != nil {
		return nil, err
	}
	if stringsRadio(site) {
		if err := set.array(plan); err != nil {
			return nil, err
		}
	} else {
		if err := set.observatory(plan); err != nil {
			return nil, err
		}
	}
	set.World.Lamp(set.Sun)
	set.World.Lamp(set.Moon)
	weather := medium.NewWeather(0.006, 55, 0.72, 421,
		maths.V(2.5, 0, -0.7))
	weather.PatchScale = 95
	if site == "atacama" || site == "radio-atacama" {
		weather.Extinction, weather.Coverage = 0.002, 0.42
	}
	var err error
	set.Mist, err = medium.NewLayer(weather,
		maths.Box{Min: maths.V(-420, -90, -420), Max: maths.V(420, 70, 420)},
		groundHeight)
	if err != nil {
		return nil, err
	}
	set.World.AddVolume(thing.Volume{Medium: set.Mist,
		Material: &thing.Material{Name: "authored flowing mist", Colour: maths.Grey(0.78)}})
	set.World.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 900, Thick: 0.00012},
		Material: &thing.Material{Name: "thin authored air", Colour: maths.Grey(0.6)},
		Motion:   thing.Still(maths.NewSubspace(maths.V(0, 150, 0), maths.Identity, 1))})
	return set, nil
}

// stringsRadio keeps the finite site inventory explicit.
func stringsRadio(site string) bool {
	return site == "radio-hawaii" || site == "radio-atacama"
}

// groundHeight is a fixed authored summit, not a measured reconstruction.
func groundHeight(x, z float64) float64 {
	radius := math.Hypot(x, z)
	plateau := math.Max(0, radius-45)
	return -0.11*plateau + 3*math.Sin(x/70)*math.Sin(z/90)
}

// land supplies a continuous far floor below the bounded summit mesh.
func (set *set) land(site string) error {
	const side, extent = 49, 800.0
	heights := make([]float64, side*side)
	for row := range side {
		for col := range side {
			x, z := -extent/2+extent*float64(col)/(side-1), -extent/2+extent*float64(row)/(side-1)
			heights[row*side+col] = groundHeight(x, z)
		}
	}
	ground, err := shape.NewHeightField(heights, side, extent)
	if err != nil {
		return err
	}
	paint := maths.RGB(0.075, 0.1, 0.085)
	if site == "atacama" || site == "radio-atacama" {
		paint = maths.RGB(0.16, 0.105, 0.075)
	}
	material := &thing.Material{Name: "authored summit ground", Colour: paint}
	set.World.Add(thing.Thing{Shape: ground, Material: material})
	far, err := shape.NewMesh([]maths.Vec{maths.V(-10000, -80, -10000),
		maths.V(10000, -80, -10000), maths.V(10000, -80, 10000), maths.V(-10000, -80, 10000)},
		[][3]int{{0, 2, 1}, {0, 3, 2}}, nil, false)
	if err != nil {
		return err
	}
	set.World.Add(thing.Thing{Shape: far, Material: material})
	for index := range 12 {
		x := -100 + 23*float64(index)
		z := 35 + 22*math.Sin(float64(index)*2.1)
		if err := set.scenery("boulder", uint64(index), maths.V(x, groundHeight(x, z), z), 1); err != nil {
			return err
		}
	}
	if site == "hawaii" {
		for index := range 9 {
			x, z := -70+float64(index)*19, 40+25*math.Sin(float64(index)*1.7)
			if err := set.scenery("coast-redwood", 77, maths.V(x, groundHeight(x, z), z), 1); err != nil {
				return err
			}
		}
	}
	return nil
}

// scenery grows a factory object; recipes and their refusal paths stay shared.
func (set *set) scenery(name string, seed uint64, at maths.Vec, scale float64) error {
	key := fmt.Sprintf("%s/%d", name, seed)
	if prototype, ok := set.prototypes[key]; ok {
		for _, item := range prototype {
			item.Motion = thing.Still(maths.NewSubspace(at, maths.Identity, scale))
			set.World.Add(item)
		}
		return nil
	}
	kind, err := manufacture.KindNamed(name)
	if err != nil {
		return err
	}
	detail := "quick"
	if name == "coast-redwood" {
		detail = "clumps"
	}
	object, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(seed), detail)
	if err != nil {
		return err
	}
	things, err := object.Things(maths.Local())
	if err != nil {
		return err
	}
	set.prototypes[key] = things
	for _, item := range things {
		item.Motion = thing.Still(maths.NewSubspace(at, maths.Identity, scale))
		set.World.Add(item)
	}
	return nil
}

// observatory retains the factory's hollow dome and optical structure.
func (set *set) observatory(plan Plan) error {
	kind, err := manufacture.Observatory(manufacture.Optical{
		Diameter: measure.Metres(plan.Diameter), Aperture: measure.Metres(plan.Aperture),
		Opening: measure.Degrees(42), Altitude: measure.Degrees(plan.Altitude),
		Azimuth: measure.Degrees(0), Segmented: true})
	if err != nil {
		return err
	}
	if err := set.instrument(kind, maths.Vec{}, 1); err != nil {
		return err
	}
	for index := range 3 {
		if err := set.scenery([]string{"service-truck", "mobile-crane", "scaffold-tower"}[index],
			uint64(index+1), maths.V(-35+float64(index)*35, 0, -27), 1); err != nil {
			return err
		}
	}
	for range 4 {
		laser, err := emit.NewBeam(emit.Lit{At: maths.V(0, 30, 0),
			Toward:   maths.YAxis,
			Strength: maths.RGB(180, 100, 12).Scale(guideBeamPeakGain)},
			measure.Metres(guideBeamDiameterMetres/2),
			measure.Metres(650), measure.Degrees(0))
		if err != nil {
			return err
		}
		set.Lasers = append(set.Lasers, laser)
		set.World.Lamp(laser)
	}
	return nil
}

// instrument shares manufactured meshes while retaining one actor pose.
func (set *set) instrument(kind manufacture.Kind, at maths.Vec, scale float64) error {
	object, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(9), "quick")
	if err != nil {
		return err
	}
	things, err := object.Things(maths.Local())
	if err != nil {
		return err
	}
	actor := actor{at: at, scale: scale}
	for _, item := range things {
		actor.things = append(actor.things, set.World.Add(item))
	}
	set.Actors = append(set.Actors, actor)
	return nil
}

// array places distinct antennas across either a compact field or wide arms.
func (set *set) array(plan Plan) error {
	kind, err := manufacture.KindNamed("sma-antenna")
	if err != nil {
		return err
	}
	object, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(9), "quick")
	if err != nil {
		return err
	}
	prototype, err := object.Things(maths.Local())
	if err != nil {
		return err
	}
	for index := range plan.ArrayCount {
		radius := 17 * math.Sqrt(float64(index))
		angle := float64(index) * 2.399963229728653 // Authored golden-angle layout.
		scale := 1.0
		if set.Site == "radio-atacama" {
			radius *= 1.55
			scale = 1.4
		}
		x, z := radius*math.Cos(angle), radius*math.Sin(angle)
		actor := actor{at: maths.V(x, groundHeight(x, z), z), scale: scale}
		for _, item := range prototype {
			actor.things = append(actor.things, set.World.Add(item))
		}
		set.Actors = append(set.Actors, actor)
	}
	return nil
}

// update changes the light and poses for one shot, keeping terrain fixed.
func (set *set) update(plan Plan, state State) (maths.Subspace, float64, error) {
	if set.Space {
		eye, wide := set.spaceView(state)
		return eye, wide, nil
	}
	yaw := maths.About(maths.YAxis, -state.Aim*maths.Degree)
	for _, actor := range set.Actors {
		for _, item := range actor.things {
			item.Motion = thing.Still(maths.NewSubspace(actor.at, yaw, actor.scale))
		}
	}
	phase := state.SceneSeconds/86400*2*math.Pi - math.Pi/2
	altitude := math.Sin(phase)
	sun := maths.V(math.Cos(phase), altitude, 0.35).Unit()
	moon := maths.V(-sun.X, -sun.Y, -sun.Z)
	day := math.Max(0, altitude)
	set.Sun.Toward, set.Sun.Strength = sun, maths.RGB(0.10, 0.065, 0.035).Scale(day)
	// Authored moon exposure keeps the moving dishes readable in this music
	// preview; these strengths do not represent measured lunar irradiance.
	set.Moon.Toward, set.Moon.Strength = moon, maths.RGB(0.24, 0.36, 0.64).Scale(math.Max(0, moon.Y))
	if plan.FullCut {
		// Authored warm, restrained moon exposure for the revised rough cut.
		set.Moon.Strength = maths.RGB(0.16, 0.13, 0.09).Scale(math.Max(0, moon.Y))
	}
	set.World.Lamps[len(set.Lasers)] = set.Sun
	set.World.Lamps[len(set.Lasers)+1] = set.Moon
	set.World.Sky = siteSky{sun: sun, moon: moon, day: day}
	if set.Mist != nil {
		set.World.Volumes[0].Medium = set.Mist.Tick(time.Duration(state.At * 4 * float64(time.Second)))
	}
	az, alt := state.Aim*maths.Degree, plan.Altitude*maths.Degree
	axis := maths.V(math.Sin(az)*math.Cos(alt), math.Sin(alt), -math.Cos(az)*math.Cos(alt))
	right := maths.V(math.Cos(az), 0, math.Sin(az))
	up := right.Cross(axis).Unit()
	centre := maths.V(0, plan.Diameter*0.19, 0).Add(axis.Scale(plan.Diameter * 0.52))
	if plan.FullCut {
		// Place the visible beam's start inside this generic instrument.
		// This is an authored offset, not a telescope-specific launcher model.
		centre = centre.Sub(axis.Scale(plan.Diameter * 0.35))
	}
	strength := maths.RGB(180, 100, 12).Scale(guideBeamPeakGain)
	for index := range set.Lasers {
		x, y := float64(index%2)*1.2-0.6, float64(index/2)*1.2-0.6
		laser, err := emit.NewBeam(emit.Lit{At: centre.Add(right.Scale(x)).Add(up.Scale(y)),
			Toward: axis, Strength: strength.Scale(1 + state.Flash*0.7)},
			measure.Metres(guideBeamDiameterMetres/2),
			measure.Metres(650), measure.Degrees(0))
		if err != nil {
			return maths.Subspace{}, 0, err
		}
		set.Lasers[index] = laser
		set.World.Lamps[index] = laser
	}
	angle := plan.Azimuth*maths.Degree + 0.36*math.Sin(state.Camera*0.055)
	distance, height := 100.0, 27.0
	if stringsRadio(set.Site) {
		distance, height = 190, 75
	}
	if set.Site == "atacama" {
		angle += 0.6
	}
	from := maths.V(distance*math.Sin(angle), height, -distance*math.Cos(angle))
	look := maths.V(0, 14, 0)
	return maths.NewSubspace(from, maths.LookAt(from, look), 1), 55 - 3*state.Flash, nil
}

// siteSky is an authored time-lapse sky; its discs and illumination share
// the same directions, avoiding the earlier backwards-lighting mistake.
type siteSky struct {
	sun, moon maths.Vec
	day       float64
}

// Light keeps the night dark, with a brief subdued warm daylight horizon.
func (skybox siteSky) Light(toward maths.Vec) maths.Light {
	direction := toward.Unit()
	horizon := maths.RGB(0.008+0.06*skybox.day, 0.011+0.022*skybox.day, 0.022+0.012*skybox.day)
	base := sky.Gradient{Horizon: horizon, Zenith: maths.RGB(0.001, 0.002, 0.008).Scale(1 + skybox.day*3), Ground: maths.RGB(0.003, 0.004, 0.005)}.Light(direction)
	if direction.Dot(skybox.sun) > math.Cos(0.012) && skybox.sun.Y > 0 {
		return base.Add(maths.RGB(0.8, 0.35, 0.08))
	}
	if direction.Dot(skybox.moon) > math.Cos(0.014) && skybox.moon.Y > 0 {
		return base.Add(maths.RGB(0.20, 0.27, 0.4))
	}
	stars := noise.Fractal(41, direction.X*450, direction.Y*450, direction.Z*450, 1)
	if skybox.day < 0.1 && stars > 0.91 {
		base = base.Add(maths.Grey(0.1))
	}
	return base
}

// blazar uses actual emissive volumes, with authored component weights per
// view. Those views are illustrations, not measured spectral intensities.
func (set *set) blazar() error {
	set.Space = true
	set.World.Sky = sky.Gradient{Horizon: maths.RGB(0.001, 0.0005, 0.004), Zenith: maths.RGB(0.001, 0.0005, 0.004)}
	set.World.Add(thing.Thing{Shape: shape.Sphere{Radius: 2.1}, Material: &thing.Material{Name: "stylized dark core", Colour: maths.Grey(0)}})
	set.Disc = &thing.Material{Name: "authored hot disc", Colour: maths.Grey(0.4), Glow: maths.RGB(9, 2.6, 0.5)}
	set.Jet = &thing.Material{Name: "authored jet", Colour: maths.RGB(0.05, 0.1, 0.2), Glow: maths.RGB(2, 4, 9)}
	set.Lobe = &thing.Material{Name: "authored extended radio glow", Glow: maths.RGB(0.5, 0.1, 1.2)}
	pose := maths.NewSubspace(maths.Vec{}, maths.About(maths.XAxis, 62*maths.Degree), 1)
	set.World.AddVolume(thing.Volume{Medium: medium.Disc{Inner: 2.2, Outer: 8, Thick: 0.45, Dense: 0.7, Grain: 0.65, Seed: 13}, Material: set.Disc, Motion: thing.Still(pose)})
	set.JetField = &medium.Jet{Length: 32, Base: 1.4, Spread: 0.075, Thick: 0.11, From: 2.7, Knots: 0.8, Seed: 77, Flow: 0.4}
	set.World.AddVolume(thing.Volume{Medium: set.JetField, Material: set.Jet, Motion: thing.Still(pose)})
	for _, sign := range []float64{-1, 1} {
		at := pose.Out(maths.V(0, sign*27, 0))
		set.World.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 8, Thick: 0.08}, Material: set.Lobe, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1))})
	}
	return nil
}

// spaceView changes which physical components of the illustration dominate.
func (set *set) spaceView(state State) (maths.Subspace, float64) {
	weights := map[string][3]float64{"radio": {0.07, 0.5, 4}, "infrared": {1.5, 0.3, 0.3}, "visible": {1, 1, 0.15}, "x-ray": {0.3, 2.5, 0.03}, "gamma": {0.04, 5, 0.01}}
	w, ok := weights[state.Spectrum]
	if !ok {
		w = weights["visible"]
	}
	set.Disc.Glow = maths.RGB(9, 2.6, 0.5).Scale(w[0])
	set.Jet.Glow = maths.RGB(2, 4, 9).Scale(w[1] * (1 + state.Flash*0.5))
	set.Lobe.Glow = maths.RGB(0.5, 0.1, 1.2).Scale(w[2])
	set.JetField.Time = state.At
	angle := 0.08 * math.Sin(state.Camera*0.02)
	from := maths.V(8+60*math.Sin(angle), 15, 68)
	return maths.NewSubspace(from, maths.LookAt(from, maths.Vec{}), 1), 52 - state.Flash*3
}

// String names a retained set in progress reports.
func (set *set) String() string { return fmt.Sprintf("searching/%s", set.Site) }
